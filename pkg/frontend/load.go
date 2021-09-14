// Copyright 2021 Matrix Origin
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package frontend

import (
	"fmt"
	"github.com/matrixorigin/simdcsv"
	"matrixone/pkg/container/batch"
	"matrixone/pkg/container/nulls"
	"matrixone/pkg/container/types"
	"matrixone/pkg/container/vector"
	"matrixone/pkg/logutil"
	"matrixone/pkg/sql/tree"
	"matrixone/pkg/vm/engine"
	"matrixone/pkg/vm/metadata"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"
)

type LoadResult struct {
	Records, Deleted, Skipped, Warnings uint64
}

type DebugTime struct {
	row2col time.Duration
	fillBlank  time.Duration
	toStorage time.Duration

	writeBatch time.Duration
	resetBatch time.Duration

	prefix time.Duration
	skip_bytes time.Duration


	process_field time.Duration
	split_field time.Duration
	split_before_loop time.Duration
	wait_loop time.Duration
	handler_get time.Duration
	wait_switch time.Duration
	field_first_byte time.Duration
	field_enclosed time.Duration
	field_without time.Duration
	field_skip_bytes time.Duration

	callback  time.Duration
	asyncChan time.Duration
	csvLineArray1 time.Duration
	csvLineArray2 time.Duration
	asyncChanLoop time.Duration
	saveParsedLine time.Duration
	choose_true time.Duration
	choose_false time.Duration
}

type SharePart struct {
	//load reference
	load *tree.Load

	//index of line in line array
	lineIdx int
	maxFieldCnt int

	lineCount uint64

	//batch
	batchSize int

	//map column id in from data to column id in table
	dataColumnId2TableColumnId []int

	cols []metadata.Attribute
	attrName []string
	timestamp uint64

	//simd csv
	simdCsvLineArray [][]string

	//storage
	dbHandler engine.Database
	tableHandler engine.Relation

	//result of load
	result *LoadResult
}

type ParseLineHandler struct {
	SharePart
	DebugTime

	simdCsvReader                    *simdcsv.Reader
	//csv read put lines into the channel
	simdCsvGetParsedLinesChan                    chan simdcsv.LineOut
	//the count of writing routine
	simdCsvConcurrencyCountOfWriteBatch          int
	simdCsvConcurrencyCountSemaphoreOfWriteBatch chan int
	//write routines put the writing results to the channel
	simdCsvResultsOfWriteBatchChan chan *WriteBatchHandler
	//wait write routines to quit
	simdCsvWaitWriteRoutineToQuit *sync.WaitGroup
	closeOnce sync.Once

	closeRef *CloseLoadData
}

type WriteBatchHandler struct {
	SharePart
	DebugTime

	batchData *batch.Batch
	batchFilled int
	simdCsvErr error

	closeRef *CloseLoadData
}

type CloseLoadData struct {
	closeReadParsedLines *CloseFlag
	closeStatistics      *CloseFlag
	stopLoadData chan struct{}
}

func NewCloseLoadData() *CloseLoadData{
	return &CloseLoadData{
		closeReadParsedLines: &CloseFlag{},
		closeStatistics:      &CloseFlag{},
		stopLoadData: make(chan struct{}),
	}
}

func (cld *CloseLoadData) Open() {
	cld.closeReadParsedLines.Open()
	cld.closeStatistics.Open()
}

func (cld *CloseLoadData) Close() {
	cld.closeReadParsedLines.Close()
	cld.closeStatistics.Close()
	close(cld.stopLoadData)
}

func (plh *ParseLineHandler) getLineOutFromSimdCsvRoutine() error {
	wait_a := time.Now()
	defer func() {
		plh.asyncChan += time.Since(wait_a)
	}()

	var lineOut simdcsv.LineOut
	plh.closeRef.closeReadParsedLines.Open()
	for plh.closeRef.closeReadParsedLines.IsOpened() {
		select {
		case <- plh.closeRef.stopLoadData:
			logutil.Infof("----- read parsed lines close")
			return nil
			case lineOut = <- plh.simdCsvGetParsedLinesChan:
		}

		wait_d := time.Now()
		if lineOut.Line == nil  && lineOut.Lines == nil {
			break
		}
		if lineOut.Line != nil {
			//step 1 : skip dropped lines
			if plh.lineCount < plh.load.IgnoredLines {
				plh.lineCount++
				return nil
			}

			wait_b := time.Now()
			//step 2 : append line into line array
			plh.simdCsvLineArray[plh.lineIdx] = lineOut.Line
			plh.lineIdx++
			plh.maxFieldCnt = Max(plh.maxFieldCnt,len(lineOut.Line))

			plh.csvLineArray1 += time.Since(wait_b)

			if plh.lineIdx == plh.batchSize {
				err := doWriteBatch(plh, false)
				if err != nil {
					return err
				}

				plh.lineIdx = 0
				plh.maxFieldCnt = 0
			}

		}else if lineOut.Lines != nil {
			from := 0
			countOfLines := len(lineOut.Lines)
			//step 1 : skip dropped lines
			if plh.lineCount < plh.load.IgnoredLines {
				skipped := MinUint64(uint64(countOfLines), plh.load.IgnoredLines - plh.lineCount)
				plh.lineCount += skipped
				from += int(skipped)
			}

			fill := 0
			//step 2 : append lines into line array
			for i := from; i < countOfLines;  i += fill {
				fill = Min(countOfLines - i, plh.batchSize - plh.lineIdx)
				wait_c := time.Now()
				for j := 0; j < fill; j++ {
					plh.simdCsvLineArray[plh.lineIdx] = lineOut.Lines[i + j]
					plh.lineIdx++
					plh.maxFieldCnt = Max(plh.maxFieldCnt,len(lineOut.Lines[i + j]))
				}
				plh.csvLineArray2 += time.Since(wait_c)

				if plh.lineIdx == plh.batchSize {
					err := doWriteBatch(plh, false)
					if err != nil {
						return err
					}

					plh.lineIdx = 0
					plh.maxFieldCnt = 0
				}
			}
		}
		plh.asyncChanLoop += time.Since(wait_d)
	}

	//last batch
	err := doWriteBatch(plh, true)
	if err != nil {
		return err
	}
	return nil
}

func (plh *ParseLineHandler) close() {
	plh.closeOnce.Do(func() {
		close(plh.simdCsvGetParsedLinesChan)
		close(plh.simdCsvResultsOfWriteBatchChan)
		close(plh.simdCsvConcurrencyCountSemaphoreOfWriteBatch)
		plh.closeRef.Close()
	})
}

/*
Init ParseLineHandler
 */
func initParseLineHandler(handler *ParseLineHandler) error {
	relation := handler.tableHandler
	load := handler.load

	cols := relation.Attribute()
	attrName := make([]string,len(cols))
	tableName2ColumnId := make(map[string]int)
	for i, col := range cols {
		attrName[i] = col.Name
		tableName2ColumnId[col.Name] = i
	}

	handler.cols = cols
	handler.attrName = attrName

	//define the peer column for LOAD DATA's column list.
	var dataColumnId2TableColumnId []int = nil
	if len(load.ColumnList) == 0{
		dataColumnId2TableColumnId = make([]int,len(cols))
		for i := 0; i < len(cols); i++ {
			dataColumnId2TableColumnId[i] = i
		}
	}else{
		dataColumnId2TableColumnId = make([]int,len(load.ColumnList))
		for i, col := range load.ColumnList {
			switch realCol := col.(type) {
			case *tree.UnresolvedName:
				tid,ok := tableName2ColumnId[realCol.Parts[0]]
				if !ok {
					return fmt.Errorf("no such column %s", realCol.Parts[0])
				}
				dataColumnId2TableColumnId[i] = tid
			case *tree.VarExpr:
				//NOTE:variable like '@abc' will be passed by.
				dataColumnId2TableColumnId[i] = -1
			default:
				return fmt.Errorf("unsupported column type %v",realCol)
			}
		}
	}
	handler.dataColumnId2TableColumnId = dataColumnId2TableColumnId
	return nil
}

func initWriteBatchHandler(handler *ParseLineHandler,wHandler *WriteBatchHandler,) error {
	batchSize := handler.batchSize
	wHandler.cols = handler.cols
	wHandler.dataColumnId2TableColumnId = handler.dataColumnId2TableColumnId
	wHandler.batchSize = handler.batchSize
	wHandler.attrName = handler.attrName
	wHandler.dbHandler = handler.dbHandler
	wHandler.tableHandler = handler.tableHandler
	wHandler.timestamp = handler.timestamp
	wHandler.result = &LoadResult{}
	wHandler.closeRef = handler.closeRef

	batchData := batch.New(true,handler.attrName)

	//fmt.Printf("----- batchSize %d attrName %v \n",batchSize,handler.attrName)

	//alloc space for vector
	for i := 0; i < len(handler.attrName); i++ {
		vec := vector.New(wHandler.cols[i].Type)
		switch vec.Typ.Oid {
		case types.T_int8:
			vec.Col = make([]int8, batchSize)
		case types.T_int16:
			vec.Col = make([]int16, batchSize)
		case types.T_int32:
			vec.Col = make([]int32, batchSize)
		case types.T_int64:
			vec.Col = make([]int64, batchSize)
		case types.T_uint8:
			vec.Col = make([]uint8, batchSize)
		case types.T_uint16:
			vec.Col = make([]uint16, batchSize)
		case types.T_uint32:
			vec.Col = make([]uint32, batchSize)
		case types.T_uint64:
			vec.Col = make([]uint64, batchSize)
		case types.T_float32:
			vec.Col = make([]float32, batchSize)
		case types.T_float64:
			vec.Col = make([]float64, batchSize)
		case types.T_char, types.T_varchar:
			vBytes := &types.Bytes{
				Offsets: make([]uint32,batchSize),
				Lengths: make([]uint32,batchSize),
				Data: nil,
			}
			vec.Col = vBytes
		default:
			panic("unsupported vector type")
		}
		batchData.Vecs[i] = vec
	}
	wHandler.batchData = batchData
	return nil
}

func saveParsedLinesToBatchSimdCsvConcurrentWrite(handler *WriteBatchHandler, forceConvert bool) error {
	begin := time.Now()
	defer func() {
		handler.saveParsedLine += time.Since(begin)
		//fmt.Printf("-----saveParsedLinesToBatchSimdCsv %s\n",time.Since(begin))
	}()

	countOfLineArray := handler.lineIdx
	if !forceConvert {
		if countOfLineArray != handler.batchSize {
			fmt.Printf("---->countOfLineArray %d batchSize %d \n",countOfLineArray,handler.batchSize)
			panic("-----write a batch")
		}
	}

	batchData := handler.batchData
	columnFLags := make([]byte,len(batchData.Vecs))
	fetchCnt := 0
	var err error
	allFetchCnt := 0

	row2col := time.Duration(0)
	fillBlank := time.Duration(0)
	toStorage := time.Duration(0)
	//write batch of  lines
	//for lineIdx := 0; lineIdx < countOfLineArray; lineIdx += fetchCnt {
	//fill batch
	fetchCnt = countOfLineArray
	//fmt.Printf("-----fetchCnt %d len(lineArray) %d\n",fetchCnt,len(handler.lineArray))
	fetchLines := handler.simdCsvLineArray[:fetchCnt]

	/*
		row to column
	*/

	batchBegin := handler.batchFilled

	chose := true

	if chose {
		wait_d := time.Now()
		for i, line := range fetchLines {
			//fmt.Printf("line %d %v \n",i,line)
			//wait_a := time.Now()
			rowIdx := batchBegin + i
			//record missing column
			for k := 0; k < len(columnFLags); k++ {
				columnFLags[k] = 0
			}

			for j, field := range line {
				//fmt.Printf("data col %d : %v \n",j,field)
				//where will column j go ?
				colIdx := -1
				if j < len(handler.dataColumnId2TableColumnId) {
					colIdx = handler.dataColumnId2TableColumnId[j]
				}
				//drop this field
				if colIdx == -1 {
					continue
				}

				isNullOrEmpty := len(field) == 0

				//put it into batch
				vec := batchData.Vecs[colIdx]

				//record colIdx
				columnFLags[colIdx] = 1

				//fmt.Printf("data set col %d : %v \n",j,field)

				switch vec.Typ.Oid {
				case types.T_int8:
					cols := vec.Col.([]int8)
					if isNullOrEmpty {
						vec.Nsp.Add(uint64(rowIdx))
					} else {
						d, err := strconv.ParseInt(field, 10, 8)
						if err != nil {
							logutil.Errorf("parse field[%v] err:%v", field, err)
							d = 0
							//break
						}
						cols[rowIdx] = int8(d)
					}
				case types.T_int16:
					cols := vec.Col.([]int16)
					if isNullOrEmpty {
						vec.Nsp.Add(uint64(rowIdx))
					} else {
						d, err := strconv.ParseInt(field, 10, 16)
						if err != nil {
							logutil.Errorf("parse field[%v] err:%v", field, err)
							d = 0
							//break
						}
						cols[rowIdx] = int16(d)
					}
				case types.T_int32:
					cols := vec.Col.([]int32)
					if isNullOrEmpty {
						vec.Nsp.Add(uint64(rowIdx))
					} else {
						d, err := strconv.ParseInt(field, 10, 32)
						if err != nil {
							logutil.Errorf("parse field[%v] err:%v", field, err)
							d = 0
							//break
						}
						cols[rowIdx] = int32(d)
					}
				case types.T_int64:
					cols := vec.Col.([]int64)
					if isNullOrEmpty {
						vec.Nsp.Add(uint64(rowIdx))
					} else {
						d, err := strconv.ParseInt(field, 10, 64)
						if err != nil {
							logutil.Errorf("parse field[%v] err:%v", field, err)
							d = 0
							//break
						}
						cols[rowIdx] = d
					}
				case types.T_uint8:
					cols := vec.Col.([]uint8)
					if isNullOrEmpty {
						vec.Nsp.Add(uint64(rowIdx))
					} else {
						d, err := strconv.ParseInt(field, 10, 8)
						if err != nil {
							logutil.Errorf("parse field[%v] err:%v", field, err)
							d = 0
							//break
						}
						cols[rowIdx] = uint8(d)
					}
				case types.T_uint16:
					cols := vec.Col.([]uint16)
					if isNullOrEmpty {
						vec.Nsp.Add(uint64(rowIdx))
					} else {
						d, err := strconv.ParseInt(field, 10, 16)
						if err != nil {
							logutil.Errorf("parse field[%v] err:%v", field, err)
							d = 0
							//break
						}
						cols[rowIdx] = uint16(d)
					}
				case types.T_uint32:
					cols := vec.Col.([]uint32)
					if isNullOrEmpty {
						vec.Nsp.Add(uint64(rowIdx))
					} else {
						d, err := strconv.ParseInt(field, 10, 32)
						if err != nil {
							logutil.Errorf("parse field[%v] err:%v", field, err)
							d = 0
							//break
						}
						cols[rowIdx] = uint32(d)
					}
				case types.T_uint64:
					cols := vec.Col.([]uint64)
					if isNullOrEmpty {
						vec.Nsp.Add(uint64(rowIdx))
					} else {
						d, err := strconv.ParseInt(field, 10, 64)
						if err != nil {
							logutil.Errorf("parse field[%v] err:%v", field, err)
							d = 0
							//break
						}
						cols[rowIdx] = uint64(d)
					}
				case types.T_float32:
					cols := vec.Col.([]float32)
					if isNullOrEmpty {
						vec.Nsp.Add(uint64(rowIdx))
					} else {
						d, err := strconv.ParseFloat(field, 32)
						if err != nil {
							logutil.Errorf("parse field[%v] err:%v", field, err)
							d = 0
							//break
						}
						cols[rowIdx] = float32(d)
					}
				case types.T_float64:
					cols := vec.Col.([]float64)
					if isNullOrEmpty {
						vec.Nsp.Add(uint64(rowIdx))
					} else {
						fs := field
						//fmt.Printf("==== > field string [%s] \n",fs)
						d, err := strconv.ParseFloat(fs, 64)
						if err != nil {
							logutil.Errorf("parse field[%v] err:%v", field, err)
							d = 0
							//break
						}
						cols[rowIdx] = d
					}
				case types.T_char, types.T_varchar:
					vBytes := vec.Col.(*types.Bytes)
					if isNullOrEmpty {
						vec.Nsp.Add(uint64(rowIdx))
						vBytes.Offsets[rowIdx] = uint32(len(vBytes.Data))
						vBytes.Lengths[rowIdx] = uint32(len(field))
					} else {
						vBytes.Offsets[rowIdx] = uint32(len(vBytes.Data))
						vBytes.Data = append(vBytes.Data, field...)
						vBytes.Lengths[rowIdx] = uint32(len(field))
					}
				default:
					panic("unsupported oid")
				}
			}
			//row2col += time.Since(wait_a)

			//wait_b := time.Now()
			//the row does not have field
			for k := 0; k < len(columnFLags); k++ {
				if 0 == columnFLags[k] {
					vec := batchData.Vecs[k]
					switch vec.Typ.Oid {
					case types.T_char, types.T_varchar:
						vBytes := vec.Col.(*types.Bytes)
						vBytes.Offsets[rowIdx] = uint32(len(vBytes.Data))
						vBytes.Lengths[rowIdx] = uint32(0)
					}
					vec.Nsp.Add(uint64(rowIdx))
				}
			}
			//fillBlank += time.Since(wait_b)
		}
		handler.choose_true += time.Since(wait_d)
	} else{
		wait_d := time.Now()
		//record missing column
		for k := 0; k < len(columnFLags); k++ {
			columnFLags[k] = 0
		}

		wait_a := time.Now()
		//column
		for j := 0; j < handler.maxFieldCnt; j++ {
			//where will column j go ?
			colIdx := -1
			if j < len(handler.dataColumnId2TableColumnId) {
				colIdx = handler.dataColumnId2TableColumnId[j]
			}
			//drop this field
			if colIdx == -1 {
				continue
			}

			//put it into batch
			vec := batchData.Vecs[colIdx]

			columnFLags[j] = 1

			switch vec.Typ.Oid {
			case types.T_int8:
				cols := vec.Col.([]int8)
				//row
				for i := 0; i < countOfLineArray; i++ {
					line := fetchLines[i]
					if j >= len(line) || len(line[j]) == 0 {
						vec.Nsp.Add(uint64(i))
					} else {
						field := line[j]
						d, err := strconv.ParseInt(field, 10, 8)
						if err != nil {
							logutil.Errorf("parse field[%v] err:%v", field, err)
							d = 0
							//break
						}
						cols[i] = int8(d)
					}
				}
			case types.T_int16:
				cols := vec.Col.([]int16)
				//row
				for i := 0; i < countOfLineArray; i++ {
					line := fetchLines[i]
					if j >= len(line) || len(line[j]) == 0 {
						vec.Nsp.Add(uint64(i))
					} else {
						field := line[j]
						d, err := strconv.ParseInt(field, 10, 16)
						if err != nil {
							logutil.Errorf("parse field[%v] err:%v", field, err)
							d = 0
							//break
						}
						cols[i] = int16(d)
					}
				}
			case types.T_int32:
				cols := vec.Col.([]int32)
				//row
				for i := 0; i < countOfLineArray; i++ {
					line := fetchLines[i]
					if j >= len(line) || len(line[j]) == 0 {
						vec.Nsp.Add(uint64(i))
					} else {
						field := line[j]
						d, err := strconv.ParseInt(field, 10, 32)
						if err != nil {
							logutil.Errorf("parse field[%v] err:%v", field, err)
							d = 0
							//break
						}
						cols[i] = int32(d)
					}
				}
			case types.T_int64:
				cols := vec.Col.([]int64)
				//row
				for i := 0; i < countOfLineArray; i++ {
					line := fetchLines[i]
					if j >= len(line) || len(line[j]) == 0 {
						vec.Nsp.Add(uint64(i))
					} else {
						field := line[j]
						d, err := strconv.ParseInt(field, 10, 64)
						if err != nil {
							logutil.Errorf("parse field[%v] err:%v", field, err)
							d = 0
							//break
						}
						cols[i] = d
					}
				}
			case types.T_uint8:
				cols := vec.Col.([]uint8)
				//row
				for i := 0; i < countOfLineArray; i++ {
					line := fetchLines[i]
					if j >= len(line) || len(line[j]) == 0 {
						vec.Nsp.Add(uint64(i))
					} else {
						field := line[j]
						d, err := strconv.ParseInt(field, 10, 8)
						if err != nil {
							logutil.Errorf("parse field[%v] err:%v", field, err)
							d = 0
							//break
						}
						cols[i] = uint8(d)
					}
				}
			case types.T_uint16:
				cols := vec.Col.([]uint16)
				//row
				for i := 0; i < countOfLineArray; i++ {
					line := fetchLines[i]
					if j >= len(line) || len(line[j]) == 0 {
						vec.Nsp.Add(uint64(i))
					} else {
						field := line[j]
						d, err := strconv.ParseInt(field, 10, 16)
						if err != nil {
							logutil.Errorf("parse field[%v] err:%v", field, err)
							d = 0
							//break
						}
						cols[i] = uint16(d)
					}
				}
			case types.T_uint32:
				cols := vec.Col.([]uint32)
				//row
				for i := 0; i < countOfLineArray; i++ {
					line := fetchLines[i]
					if j >= len(line) || len(line[j]) == 0 {
						vec.Nsp.Add(uint64(i))
					} else {
						field := line[j]
						d, err := strconv.ParseInt(field, 10, 32)
						if err != nil {
							logutil.Errorf("parse field[%v] err:%v", field, err)
							d = 0
							//break
						}
						cols[i] = uint32(d)
					}
				}
			case types.T_uint64:
				cols := vec.Col.([]uint64)
				//row
				for i := 0; i < countOfLineArray; i++ {
					line := fetchLines[i]
					if j >= len(line) || len(line[j]) == 0 {
						vec.Nsp.Add(uint64(i))
					} else {
						field := line[j]
						d, err := strconv.ParseInt(field, 10, 64)
						if err != nil {
							logutil.Errorf("parse field[%v] err:%v", field, err)
							d = 0
							//break
						}
						cols[i] = uint64(d)
					}
				}
			case types.T_float32:
				cols := vec.Col.([]float32)
				//row
				for i := 0; i < countOfLineArray; i++ {
					line := fetchLines[i]
					if j >= len(line) || len(line[j]) == 0 {
						vec.Nsp.Add(uint64(i))
					} else {
						field := line[j]
						d, err := strconv.ParseFloat(field, 32)
						if err != nil {
							logutil.Errorf("parse field[%v] err:%v", field, err)
							d = 0
							//break
						}
						cols[i] = float32(d)
					}
				}
			case types.T_float64:
				cols := vec.Col.([]float64)
				//row
				for i := 0; i < countOfLineArray; i++ {
					line := fetchLines[i]
					if j >= len(line) || len(line[j]) == 0 {
						vec.Nsp.Add(uint64(i))
					} else {
						field := line[j]
						//fmt.Printf("==== > field string [%s] \n",fs)
						d, err := strconv.ParseFloat(field, 64)
						if err != nil {
							logutil.Errorf("parse field[%v] err:%v", field, err)
							d = 0
							//break
						}
						cols[i] = d
					}
				}
			case types.T_char, types.T_varchar:
				vBytes := vec.Col.(*types.Bytes)
				//row
				for i := 0; i < countOfLineArray; i++ {
					line := fetchLines[i]
					if j >= len(line) || len(line[j]) == 0 {
						vec.Nsp.Add(uint64(i))
						vBytes.Offsets[i] = uint32(len(vBytes.Data))
						vBytes.Lengths[i] = uint32(len(line[j]))
					} else {
						field := line[j]
						vBytes.Offsets[i] = uint32(len(vBytes.Data))
						vBytes.Data = append(vBytes.Data, field...)
						vBytes.Lengths[i] = uint32(len(field))
					}
				}
			default:
				panic("unsupported oid")
			}
		}
		row2col += time.Since(wait_a)

		wait_b := time.Now()
		//the row does not have field
		for k := 0; k < len(columnFLags	); k++ {
			if 0 == columnFLags[k] {
				vec := batchData.Vecs[k]
				//row
				for i := 0; i < countOfLineArray; i++ {
					switch vec.Typ.Oid {
					case types.T_char, types.T_varchar:
						vBytes := vec.Col.(*types.Bytes)
						vBytes.Offsets[i] = uint32(len(vBytes.Data))
						vBytes.Lengths[i] = uint32(0)
					}
					vec.Nsp.Add(uint64(i))
				}
			}
		}
		fillBlank += time.Since(wait_b)
		handler.choose_false += time.Since(wait_d)
	}
	handler.lineCount += uint64(fetchCnt)
	handler.batchFilled = batchBegin + fetchCnt

	//if handler.batchFilled == handler.batchSize {
	//	minLen := math.MaxInt64
	//	maxLen := 0
	//	for _, vec := range batchData.Vecs {
	//		fmt.Printf("len %d type %d %s \n",vec.Length(),vec.Typ.Oid,vec.Typ.String())
	//		minLen = Min(vec.Length(),int(minLen))
	//		maxLen = Max(vec.Length(),int(maxLen))
	//	}
	//
	//	if minLen != maxLen{
	//		logutil.Errorf("vector length mis equal %d %d",minLen,maxLen)
	//		return fmt.Errorf("vector length mis equal %d %d",minLen,maxLen)
	//	}
	//}

	wait_c := time.Now()
	/*
		write batch into the engine
	*/
	//the second parameter must be FALSE here
	err = saveBatchToStorageConcurrentWrite(handler,forceConvert)
	if err != nil {
		logutil.Errorf("saveBatchToStorage failed. err:%v",err)
		return err
	}
	toStorage += time.Since(wait_c)

	allFetchCnt += fetchCnt
	//}

	handler.row2col += row2col
	handler.fillBlank += fillBlank
	handler.toStorage += toStorage

	//fmt.Printf("----- row2col %s fillBlank %s toStorage %s\n",
	//	row2col,fillBlank,toStorage)

	if allFetchCnt != countOfLineArray {
		return fmt.Errorf("allFetchCnt %d != countOfLineArray %d ",allFetchCnt,countOfLineArray)
	}

	return nil
}

/*
save batch to storage.
when force is true, batchsize will be changed.
*/
func saveBatchToStorageConcurrentWrite(handler *WriteBatchHandler,force bool) error {
	if handler.batchFilled == handler.batchSize{
		//for _, vec := range handler.batchData.Vecs {
		//	fmt.Printf("len %d type %d %s \n",vec.Length(),vec.Typ.Oid,vec.Typ.String())
		//}
		wait_a := time.Now()
		err := handler.tableHandler.Write(handler.timestamp,handler.batchData)
		if err != nil {
			logutil.Errorf("write failed. err: %v",err)
			return err
		}

		handler.writeBatch += time.Since(wait_a)

		handler.result.Records += uint64(handler.batchSize)

		wait_b := time.Now()
		//clear batch
		//clear vector.nulls.Nulls
		for _, vec := range handler.batchData.Vecs {
			vec.Nsp = &nulls.Nulls{}
			switch vec.Typ.Oid {
			case types.T_char, types.T_varchar:
				vBytes := vec.Col.(*types.Bytes)
				vBytes.Data = vBytes.Data[:0]
			}
		}
		handler.batchFilled = 0

		handler.resetBatch += time.Since(wait_b)
	}else{
		if force {
			//first, remove redundant rows at last
			needLen := handler.batchFilled
			if needLen > 0{
				//logutil.Infof("needLen: %d batchSize %d", needLen, handler.batchSize)
				for _, vec := range handler.batchData.Vecs {
					//fmt.Printf("needLen %d %d type %d %s \n",needLen,i,vec.Typ.Oid,vec.Typ.String())
					//remove nulls.NUlls
					for j := uint64(handler.batchFilled); j < uint64(handler.batchSize); j++ {
						vec.Nsp.Del(j)
					}
					//remove row
					switch vec.Typ.Oid {
					case types.T_int8:
						cols := vec.Col.([]int8)
						vec.Col = cols[:needLen]
					case types.T_int16:
						cols := vec.Col.([]int16)
						vec.Col = cols[:needLen]
					case types.T_int32:
						cols := vec.Col.([]int32)
						vec.Col = cols[:needLen]
					case types.T_int64:
						cols := vec.Col.([]int64)
						vec.Col = cols[:needLen]
					case types.T_uint8:
						cols := vec.Col.([]uint8)
						vec.Col = cols[:needLen]
					case types.T_uint16:
						cols := vec.Col.([]uint16)
						vec.Col = cols[:needLen]
					case types.T_uint32:
						cols := vec.Col.([]uint32)
						vec.Col = cols[:needLen]
					case types.T_uint64:
						cols := vec.Col.([]uint64)
						vec.Col = cols[:needLen]
					case types.T_float32:
						cols := vec.Col.([]float32)
						vec.Col = cols[:needLen]
					case types.T_float64:
						cols := vec.Col.([]float64)
						vec.Col = cols[:needLen]
					case types.T_char, types.T_varchar://bytes is different
						vBytes := vec.Col.(*types.Bytes)
						//fmt.Printf("saveBatchToStorage before data %s \n",vBytes.String())
						if len(vBytes.Offsets) > needLen{
							vec.Col = vBytes.Window(0, needLen)
						}

						//fmt.Printf("saveBatchToStorage after data %s \n",vBytes.String())
					}
				}

				//for _, vec := range handler.batchData.Vecs {
				//	fmt.Printf("len %d type %d %s \n",vec.Length(),vec.Typ.Oid,vec.Typ.String())
				//}

				err := handler.tableHandler.Write(handler.timestamp, handler.batchData)
				if err != nil {
					logutil.Errorf("write failed. err:%v \n", err)
					return err
				}
			}

			handler.result.Records += uint64(needLen)
		}
	}
	return nil
}

func doWriteBatch(handler *ParseLineHandler, force bool) error {
	writeHandler := &WriteBatchHandler{
		SharePart:SharePart{
			lineIdx: handler.lineIdx,
			simdCsvLineArray: handler.simdCsvLineArray[:handler.lineIdx],
			maxFieldCnt: handler.maxFieldCnt,
		},
	}
	err := initWriteBatchHandler(handler,writeHandler)
	if err != nil {
		writeHandler.simdCsvErr =err
		return err
	}

	//acquire semaphore
	handler.simdCsvConcurrencyCountSemaphoreOfWriteBatch <- 1

	go func() {
		handler.simdCsvWaitWriteRoutineToQuit.Add(1)
		defer handler.simdCsvWaitWriteRoutineToQuit.Done()

		//step 3 : save into storage
		err = saveParsedLinesToBatchSimdCsvConcurrentWrite(writeHandler,force)
		writeHandler.simdCsvErr = err

		//release semaphore
		<- handler.simdCsvConcurrencyCountSemaphoreOfWriteBatch

		//output handler
		handler.simdCsvResultsOfWriteBatchChan <- writeHandler
	}()
	return nil
}

/*
LoadLoop reads data from stream, extracts the fields, and saves into the table
 */
func (mce *MysqlCmdExecutor) LoadLoop(load *tree.Load, dbHandler engine.Database, tableHandler engine.Relation, closeRef *CloseLoadData) (*LoadResult, error) {
	var err error
	ses := mce.routine.GetSession()

	//begin:=  time.Now()
	//defer func() {
	//	fmt.Printf("-----load loop exit %s\n",time.Since(begin))
	//}()

	result := &LoadResult{}

	/*
	step1 : read block from file
	 */
	dataFile,err := os.Open(load.File)
	if err != nil {
		logutil.Errorf("open file failed. err:%v",err)
		return nil, err
	}
	defer func() {
		err := dataFile.Close()
		if err != nil{
			logutil.Errorf("close file failed. err:%v",err)
		}
	}()

	//processTime := time.Now()
	process_block := time.Duration(0)

	//simdcsv
	handler := &ParseLineHandler{
		SharePart:SharePart{
			load: load,
			lineIdx: 0,
			simdCsvLineArray: make([][]string, int(ses.Pu.SV.GetBatchSizeInLoadData())),
			dbHandler: dbHandler,
			tableHandler: tableHandler,
			lineCount: 0,
			batchSize: int(ses.Pu.SV.GetBatchSizeInLoadData()),
			result: result,
		},
		simdCsvReader: simdcsv.NewReaderWithOptions(dataFile,
			rune(load.Fields.Terminated[0]),
			'#',
			false,
			false),
			simdCsvGetParsedLinesChan:           make(chan simdcsv.LineOut,100 * int(ses.Pu.SV.GetBatchSizeInLoadData())),
			simdCsvConcurrencyCountOfWriteBatch: int(ses.Pu.SV.GetLoadDataConcurrencyCount()),
			simdCsvResultsOfWriteBatchChan:      make(chan *WriteBatchHandler,100 * int(ses.Pu.SV.GetBatchSizeInLoadData())),
			simdCsvWaitWriteRoutineToQuit:       &sync.WaitGroup{},
			closeRef: closeRef,
	}

	//enable close flag
	//handler.closeRef.Open()

	//defer handler.close()

	handler.simdCsvConcurrencyCountOfWriteBatch = Min(handler.simdCsvConcurrencyCountOfWriteBatch,runtime.NumCPU())
	handler.simdCsvConcurrencyCountOfWriteBatch = Max(1,handler.simdCsvConcurrencyCountOfWriteBatch)
	handler.simdCsvConcurrencyCountSemaphoreOfWriteBatch = make(chan int,handler.simdCsvConcurrencyCountOfWriteBatch)

	//fmt.Printf("-----write concurrent count %d \n",handler.simdCsvConcurrencyCountOfWriteBatch)

	err = initParseLineHandler(handler)
	if err != nil {
		return nil, err
	}

	wg := sync.WaitGroup{}

	/*
	read from the output channel of the simdcsv parser, make a batch,
	deliver it to async routine writing batch
	 */
	go func() {
		wg.Add(1)
		defer wg.Done()
		err = handler.getLineOutFromSimdCsvRoutine()
		if err != nil {
			logutil.Errorf("get line from simdcsv failed. err:%v",err)
		}
	}()

	/*
	statistic routine
	collect statistics from every batch
	 */
	go func() {
		wg.Add(1)
		defer wg.Done()

		var wh *WriteBatchHandler = nil
		handler.closeRef.closeStatistics.Open()
		//collect statistics
		for handler.closeRef.closeStatistics.IsOpened() {
			select {
			case <- handler.closeRef.stopLoadData:
				logutil.Infof("----- statistics close")
				return
				case wh = <- handler.simdCsvResultsOfWriteBatchChan:
			}

			if wh == nil {
				break
			}

			//
			//fmt.Printf("++++> %d %d %d %d \n",
			//	wh.result.Skipped,
			//	wh.result.Deleted,
			//	wh.result.Warnings,
			//	wh.result.Records,
			//	)
			handler.result.Skipped += wh.result.Skipped
			handler.result.Deleted += wh.result.Deleted
			handler.result.Warnings += wh.result.Warnings
			handler.result.Records += wh.result.Records

			//
			handler.row2col += wh.row2col
			handler.fillBlank += wh.fillBlank
			handler.toStorage += wh.toStorage

			handler.writeBatch += wh.writeBatch
			handler.resetBatch += wh.resetBatch

			//
			handler.callback += wh.callback
			handler.asyncChan += wh.asyncChan
			handler.asyncChanLoop += wh.asyncChanLoop
			handler.csvLineArray1 += wh.csvLineArray1
			handler.csvLineArray2 += wh.csvLineArray2
			handler.saveParsedLine += wh.saveParsedLine
			handler.choose_true += wh.choose_true
			handler.choose_false += wh.choose_false
		}
	}()

	wait_b := time.Now()
	/*
	get lines from simdcsv, deliver them to the output channel.
	 */
	err = handler.simdCsvReader.ReadLoop(handler.simdCsvGetParsedLinesChan)
	if err != nil {
		return result, err
	}
	process_block += time.Since(wait_b)

	//wait write to quit
	handler.simdCsvWaitWriteRoutineToQuit.Wait()
	//notify statistics to quit
	handler.simdCsvResultsOfWriteBatchChan <- nil

	//wait read and statistics to quit
	wg.Wait()

	//fmt.Printf("-----total row2col %s fillBlank %s toStorage %s\n",
	//	handler.row2col,handler.fillBlank,handler.toStorage)
	//fmt.Printf("-----write batch %s reset batch %s\n",
	//	handler.writeBatch,handler.resetBatch)
	//fmt.Printf("----- simdcsv end %s " +
	//	"stage1_first_chunk %s stage1_end %s " +
	//	"stage2_first_chunkinfo - [begin end] [%s %s ] [%s %s ] [%s %s ] " +
	//	"readLoop_first_records %s \n",
	//	handler.simdCsvReader.End,
	//	handler.simdCsvReader.Stage1_first_chunk,
	//	handler.simdCsvReader.Stage1_end,
	//	handler.simdCsvReader.Stage2_first_chunkinfo[0],
	//	handler.simdCsvReader.Stage2_end[0],
	//	handler.simdCsvReader.Stage2_first_chunkinfo[1],
	//	handler.simdCsvReader.Stage2_end[1],
	//	handler.simdCsvReader.Stage2_first_chunkinfo[2],
	//	handler.simdCsvReader.Stage2_end[2],
	//	handler.simdCsvReader.ReadLoop_first_records,
	//	)
	//
	//fmt.Printf("-----call_back %s " +
	//	"process_block - callback %s " +
	//	"asyncChan %s asyncChanLoop %s asyncChan - asyncChanLoop %s " +
	//	"csvLineArray1 %s csvLineArray2 %s saveParsedLineToBatch %s " +
	//	"choose_true %s choose_false %s \n",
	//	handler.callback,
	//	process_block - handler.callback,
	//	handler.asyncChan,
	//	handler.asyncChanLoop,
	//	handler.asyncChan -	handler.asyncChanLoop,
	//	handler.csvLineArray1,
	//	handler.csvLineArray2,
	//	handler.saveParsedLine,
	//	handler.choose_true,
	//	handler.choose_false,
	//	)

//		fmt.Printf("-----process time %s \n",time.Since(processTime))

	return result, nil
}