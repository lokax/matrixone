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

package tuplecodec

import (
	"errors"
	"fmt"
	"github.com/matrixorigin/matrixone/pkg/container/batch"
	"github.com/matrixorigin/matrixone/pkg/container/nulls"
	"github.com/matrixorigin/matrixone/pkg/container/types"
	"github.com/matrixorigin/matrixone/pkg/container/vector"
	"github.com/matrixorigin/matrixone/pkg/logutil"
	"github.com/matrixorigin/matrixone/pkg/vm/engine/tpe/descriptor"
	"github.com/matrixorigin/matrixone/pkg/vm/engine/tpe/orderedcodec"
)

var (
	errorColumnIndexIsInvalid = errors.New("column index is invalid")
	errorAttributeCountNotEqual = errors.New("attribute count is not equal to the definition")
	errorInvalidAttributeId = errors.New("attributeId is invalid")
)

var _ RowColumnConverter = &RowColumnConverterImpl{}
var _ Tuple = &TupleBatchImpl{}

type RowColumnConverter interface {
	GetTupleFromBatch(bat *batch.Batch,rowID int)(Tuple,error)

	GetTuplesFromBatch(bat *batch.Batch)(Tuples,error)

	//FillBatchFromDecodedIndexKey fills the batch at row i with the data from the decoded key
	//The attributes and the attributeID are the
	FillBatchFromDecodedIndexKey(index *descriptor.IndexDesc,
		columnGroupID uint64,attributes [] *orderedcodec.DecodedItem,
		attributeID []int,bat *batch.Batch,rowID int)(error)
}

type TupleBatchImpl struct {
	//hold the data
	bat *batch.Batch

	//row
	row []interface{}
}

func NewTupleBatchImpl(bat *batch.Batch,row []interface{}) *TupleBatchImpl{
	return &TupleBatchImpl{
		bat: bat,
		row: row,
	}
}

func (tbi *TupleBatchImpl) GetAttributeCount() (uint32, error) {
	return uint32(len(tbi.bat.Vecs)), nil
}

func (tbi *TupleBatchImpl) GetAttribute(colIdx uint32) (types.Type, string, error) {
	attrCnt,_ := tbi.GetAttributeCount()
	if colIdx >= attrCnt {
		return types.Type{},"",errorColumnIndexIsInvalid
	}

	vec := tbi.bat.Vecs[colIdx]
	return vec.Typ,tbi.bat.Attrs[colIdx],nil
}

func (tbi *TupleBatchImpl) IsNull(colIdx uint32) (bool, error) {
	attrCnt,_ := tbi.GetAttributeCount()
	if colIdx >= attrCnt {
		return false,errorColumnIndexIsInvalid
	}
	return tbi.row[colIdx] == nil,nil
}

func (tbi *TupleBatchImpl) GetValue(colIdx uint32) (interface{}, error) {
	attrCnt,_ := tbi.GetAttributeCount()
	if colIdx >= attrCnt {
		return nil,errorColumnIndexIsInvalid
	}
	return tbi.row[colIdx], nil
}

func (tbi *TupleBatchImpl) GetInt(colIdx uint32) (int, error) {
	panic("implement me")
}

type RowColumnConverterImpl struct {}

func (tbi *RowColumnConverterImpl) FillBatchFromDecodedIndexKey(
		index *descriptor.IndexDesc,
		columnGroupID uint64,
		attributes []*orderedcodec.DecodedItem,
		attributeID []int,
		bat *batch.Batch,
		rowIdx int) error {
	if len(index.Attributes) != len(attributes) {
		return errorAttributeCountNotEqual
	}

	for _, attrID := range attributeID {
		if attrID < 0 || attrID >= len(attributes) ||
				attrID >= len(index.Attributes){
			return errorInvalidAttributeId
		}

		//attribute data
		attr := attributes[attrID]
		colIdx := attrID

		isNullOrEmpty := attr.ValueType == orderedcodec.VALUE_TYPE_NULL

		//put it into batch
		vec := bat.Vecs[colIdx]
		//vecAttr := batchData.Attrs[colIdx]

		switch vec.Typ.Oid {
		case types.T_int8:
			cols := vec.Col.([]int8)
			if isNullOrEmpty {
				nulls.Add(vec.Nsp, uint64(rowIdx))
			} else {
				d,err := attr.GetInt8()
				if err != nil {
					return err
				}
				cols[rowIdx] = d
			}
		case types.T_int16:
			cols := vec.Col.([]int16)
			if isNullOrEmpty {
				nulls.Add(vec.Nsp, uint64(rowIdx))
			} else {
				d, err := attr.GetInt16()
				if err != nil {
					return err
				}
				cols[rowIdx] = d
			}
		case types.T_int32:
			cols := vec.Col.([]int32)
			if isNullOrEmpty {
				nulls.Add(vec.Nsp, uint64(rowIdx))
			} else {
				d, err := attr.GetInt32()
				if err != nil {
					return err
				}
				cols[rowIdx] = d
			}
		case types.T_int64:
			cols := vec.Col.([]int64)
			if isNullOrEmpty {
				nulls.Add(vec.Nsp, uint64(rowIdx))
			} else {
				d, err := attr.GetInt64()
				if err != nil {
					return err
				}
				cols[rowIdx] = d
			}
		case types.T_uint8:
			cols := vec.Col.([]uint8)
			if isNullOrEmpty {
				nulls.Add(vec.Nsp, uint64(rowIdx))
			} else {
				d, err := attr.GetUint8()
				if err != nil {
					return err
				}
				cols[rowIdx] = d
			}
		case types.T_uint16:
			cols := vec.Col.([]uint16)
			if isNullOrEmpty {
				nulls.Add(vec.Nsp, uint64(rowIdx))
			} else {
				d, err := attr.GetUint16()
				if err != nil {
					return err
				}
				cols[rowIdx] = d
			}
		case types.T_uint32:
			cols := vec.Col.([]uint32)
			if isNullOrEmpty {
				nulls.Add(vec.Nsp, uint64(rowIdx))
			} else {
				d, err := attr.GetUint32()
				if err != nil {
					return err
				}
				cols[rowIdx] = d
			}
		case types.T_uint64:
			cols := vec.Col.([]uint64)
			if isNullOrEmpty {
				nulls.Add(vec.Nsp, uint64(rowIdx))
			} else {
				d, err := attr.GetUint64()
				if err != nil {
					return err
				}
				cols[rowIdx] = d
			}
		case types.T_float32:
			cols := vec.Col.([]float32)
			if isNullOrEmpty {
				nulls.Add(vec.Nsp, uint64(rowIdx))
			} else {
				d, err := attr.GetFloat32()
				if err != nil {
					return err
				}
				cols[rowIdx] = d
			}
		case types.T_float64:
			cols := vec.Col.([]float64)
			if isNullOrEmpty {
				nulls.Add(vec.Nsp, uint64(rowIdx))
			} else {
				d, err := attr.GetFloat64()
				if err != nil {
					return err
				}
				cols[rowIdx] = d
			}
		case types.T_char, types.T_varchar:
			vBytes := vec.Col.(*types.Bytes)
			if isNullOrEmpty {
				nulls.Add(vec.Nsp, uint64(rowIdx))
				vBytes.Offsets[rowIdx] = uint32(len(vBytes.Data))
				vBytes.Lengths[rowIdx] = uint32(0)
			} else {
				d, err := attr.GetBytes()
				if err != nil {
					return err
				}
				vBytes.Offsets[rowIdx] = uint32(len(vBytes.Data))
				vBytes.Data = append(vBytes.Data, d...)
				vBytes.Lengths[rowIdx] = uint32(len(d))
			}
		case types.T_date:
			cols := vec.Col.([]types.Date)
			if isNullOrEmpty {
				nulls.Add(vec.Nsp, uint64(rowIdx))
			} else {
				d, err := attr.GetDate()
				if err != nil {
					return err
				}
				cols[rowIdx] = d
			}
		case types.T_datetime:
			cols := vec.Col.([]types.Datetime)
			if isNullOrEmpty {
				nulls.Add(vec.Nsp, uint64(rowIdx))
			} else {
				d, err := attr.GetDatetime()
				if err != nil {
					return err
				}
				cols[rowIdx] = d
			}
		default:
			panic("unsupported oid")
		}
	}
	return nil
}

func (tbi *RowColumnConverterImpl) GetTupleFromBatch(bat *batch.Batch, rowID int) (Tuple, error) {
	panic("implement me")
}

func (tbi *RowColumnConverterImpl) GetTuplesFromBatch(bat *batch.Batch) (Tuples, error) {
	panic("implement me")
}

// BatchAdapter for convenient access to the batch
type BatchAdapter struct {
	bat *batch.Batch
}

func NewBatchAdapter(bat *batch.Batch) *BatchAdapter {
	return &BatchAdapter{bat: bat}
}

// ForEach process every row with callback function in the batch
func (ba *BatchAdapter) ForEach(callbackCtx interface{},
		callback func(callbackCtx interface{},tuple Tuple) error) error {
	n := vector.Length(ba.bat.Vecs[0])

	row := make([]interface{}, len(ba.bat.Vecs))
	tbi := NewTupleBatchImpl(ba.bat,row)

	for j := 0; j < n; j++ { //row index
		if ba.bat.Zs[j] <= 0 {
			continue
		}

		var rowIndex int64 = int64(j)
		if len(ba.bat.Sels) != 0 {
			rowIndex = ba.bat.Sels[j]
		}

		//get the row
		for i, vec := range ba.bat.Vecs { //col index
			switch vec.Typ.Oid { //get col
			case types.T_int8:
				if !nulls.Any(vec.Nsp) { //all data in this column are not null
					vs := vec.Col.([]int8)
					row[i] = vs[rowIndex]
				} else {
					if nulls.Contains(vec.Nsp, uint64(rowIndex)) { //is null
						row[i] = nil
					} else {
						vs := vec.Col.([]int8)
						row[i] = vs[rowIndex]
					}
				}
			case types.T_uint8:
				if !nulls.Any(vec.Nsp) { //all data in this column are not null
					vs := vec.Col.([]uint8)
					row[i] = vs[rowIndex]
				} else {
					if nulls.Contains(vec.Nsp, uint64(rowIndex)) { //is null
						row[i] = nil
					} else {
						vs := vec.Col.([]uint8)
						row[i] = vs[rowIndex]
					}
				}
			case types.T_int16:
				if !nulls.Any(vec.Nsp) { //all data in this column are not null
					vs := vec.Col.([]int16)
					row[i] = vs[rowIndex]
				} else {
					if nulls.Contains(vec.Nsp, uint64(rowIndex)) { //is null
						row[i] = nil
					} else {
						vs := vec.Col.([]int16)
						row[i] = vs[rowIndex]
					}
				}
			case types.T_uint16:
				if !nulls.Any(vec.Nsp) { //all data in this column are not null
					vs := vec.Col.([]uint16)
					row[i] = vs[rowIndex]
				} else {
					if nulls.Contains(vec.Nsp, uint64(rowIndex)) { //is null
						row[i] = nil
					} else {
						vs := vec.Col.([]uint16)
						row[i] = vs[rowIndex]
					}
				}
			case types.T_int32:
				if !nulls.Any(vec.Nsp) { //all data in this column are not null
					vs := vec.Col.([]int32)
					row[i] = vs[rowIndex]
				} else {
					if nulls.Contains(vec.Nsp, uint64(rowIndex)) { //is null
						row[i] = nil
					} else {
						vs := vec.Col.([]int32)
						row[i] = vs[rowIndex]
					}
				}
			case types.T_uint32:
				if !nulls.Any(vec.Nsp) { //all data in this column are not null
					vs := vec.Col.([]uint32)
					row[i] = vs[rowIndex]
				} else {
					if nulls.Contains(vec.Nsp, uint64(rowIndex)) { //is null
						row[i] = nil
					} else {
						vs := vec.Col.([]uint32)
						row[i] = vs[rowIndex]
					}
				}
			case types.T_int64:
				if !nulls.Any(vec.Nsp) { //all data in this column are not null
					vs := vec.Col.([]int64)
					row[i] = vs[rowIndex]
				} else {
					if nulls.Contains(vec.Nsp, uint64(rowIndex)) { //is null
						row[i] = nil
					} else {
						vs := vec.Col.([]int64)
						row[i] = vs[rowIndex]
					}
				}
			case types.T_uint64:
				if !nulls.Any(vec.Nsp) { //all data in this column are not null
					vs := vec.Col.([]uint64)
					row[i] = vs[rowIndex]
				} else {
					if nulls.Contains(vec.Nsp, uint64(rowIndex)) { //is null
						row[i] = nil
					} else {
						vs := vec.Col.([]uint64)
						row[i] = vs[rowIndex]
					}
				}
			case types.T_float32:
				if !nulls.Any(vec.Nsp) { //all data in this column are not null
					vs := vec.Col.([]float32)
					row[i] = vs[rowIndex]
				} else {
					if nulls.Contains(vec.Nsp, uint64(rowIndex)) { //is null
						row[i] = nil
					} else {
						vs := vec.Col.([]float32)
						row[i] = vs[rowIndex]
					}
				}
			case types.T_float64:
				if !nulls.Any(vec.Nsp) { //all data in this column are not null
					vs := vec.Col.([]float64)
					row[i] = vs[rowIndex]
				} else {
					if nulls.Contains(vec.Nsp, uint64(rowIndex)) { //is null
						row[i] = nil
					} else {
						vs := vec.Col.([]float64)
						row[i] = vs[rowIndex]
					}
				}
			case types.T_char:
				if !nulls.Any(vec.Nsp) { //all data in this column are not null
					vs := vec.Col.(*types.Bytes)
					row[i] = vs.Get(int64(rowIndex))
				} else {
					if nulls.Contains(vec.Nsp, uint64(rowIndex)) { //is null
						row[i] = nil
					} else {
						vs := vec.Col.(*types.Bytes)
						row[i] = vs.Get(int64(rowIndex))
					}
				}
			case types.T_varchar:
				if !nulls.Any(vec.Nsp) { //all data in this column are not null
					vs := vec.Col.(*types.Bytes)
					row[i] = vs.Get(int64(rowIndex))
				} else {
					if nulls.Contains(vec.Nsp, uint64(rowIndex)) { //is null
						row[i] = nil
					} else {
						vs := vec.Col.(*types.Bytes)
						row[i] = vs.Get(int64(rowIndex))
					}
				}
			case types.T_date:
				if !nulls.Any(vec.Nsp) { //all data in this column are not null
					vs := vec.Col.([]types.Date)
					row[i] = vs[rowIndex]
				} else {
					if nulls.Contains(vec.Nsp, uint64(rowIndex)) { //is null
						row[i] = nil
					} else {
						vs := vec.Col.([]types.Date)
						row[i] = vs[rowIndex]
					}
				}
			case types.T_datetime:
				if !nulls.Any(vec.Nsp) { //all data in this column are not null
					vs := vec.Col.([]types.Datetime)
					row[i] = vs[rowIndex]
				} else {
					if nulls.Contains(vec.Nsp, uint64(rowIndex)) { //is null
						row[i] = nil
					} else {
						vs := vec.Col.([]types.Datetime)
						row[i] = vs[rowIndex]
					}
				}
			default:
				logutil.Errorf("getDataFromPipeline : unsupported type %d \n", vec.Typ.Oid)
				return fmt.Errorf("getDataFromPipeline : unsupported type %d \n", vec.Typ.Oid)
			}
		}

		err := callback(callbackCtx,tbi)
		if err != nil {
			return err
		}

		//get duplicate rows
		for i := int64(0); i < ba.bat.Zs[j]-1; i++ {
			err = callback(callbackCtx,tbi)
			if err != nil {
				return err
			}
		}
	}
	return nil
}