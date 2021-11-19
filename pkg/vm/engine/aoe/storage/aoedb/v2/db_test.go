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

package aoedb

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"path"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/matrixorigin/matrixone/pkg/vm/engine/aoe/storage"
	"github.com/matrixorigin/matrixone/pkg/vm/engine/aoe/storage/common"
	"github.com/matrixorigin/matrixone/pkg/vm/engine/aoe/storage/db"
	"github.com/matrixorigin/matrixone/pkg/vm/engine/aoe/storage/dbi"
	"github.com/matrixorigin/matrixone/pkg/vm/engine/aoe/storage/internal/invariants"
	"github.com/matrixorigin/matrixone/pkg/vm/engine/aoe/storage/layout/dataio"
	"github.com/matrixorigin/matrixone/pkg/vm/engine/aoe/storage/metadata/v1"
	"github.com/matrixorigin/matrixone/pkg/vm/engine/aoe/storage/mock"
	"github.com/matrixorigin/matrixone/pkg/vm/engine/aoe/storage/testutils"
	"github.com/matrixorigin/matrixone/pkg/vm/engine/aoe/storage/wal/shard"
	"github.com/matrixorigin/matrixone/pkg/vm/mmu/guest"
	"github.com/matrixorigin/matrixone/pkg/vm/mmu/host"
	"github.com/matrixorigin/matrixone/pkg/vm/process"
	"github.com/panjf2000/ants/v2"
	"github.com/stretchr/testify/assert"
)

func TestCreateTable(t *testing.T) {
	initTestEnv(t)
	inst, gen, _ := initTestDB2(t)
	assert.NotNil(t, inst)
	defer inst.Close()
	tblCnt := rand.Intn(5) + 3
	var wg sync.WaitGroup
	names := make([]string, 0)

	for i := 0; i < tblCnt; i++ {
		schema := metadata.MockSchema(2)
		names = append(names, schema.Name)
		wg.Add(1)
		go func(w *sync.WaitGroup, id uint64) {
			ctx := &CreateDBCtx{
				DB: strconv.FormatUint(id, 10),
			}
			database, err := inst.CreateDatabase(ctx)
			assert.Nil(t, err)

			createCtx := &CreateTableCtx{
				DBMutationCtx: *CreateDBMutationCtx(database, gen),
				Schema:        schema,
			}
			_, err = inst.CreateTable(createCtx)
			assert.Nil(t, err)
			w.Done()
		}(&wg, uint64(i)+1)
	}
	wg.Wait()
	dbNames := inst.Store.Catalog.SimpleGetDatabaseNames()
	assert.Equal(t, tblCnt, len(dbNames))
	t.Log(inst.Store.Catalog.PString(metadata.PPL0, 0))
}

func TestCreateDuplicateTable(t *testing.T) {
	initTestEnv(t)
	inst, gen, database := initTestDB1(t)
	defer inst.Close()

	schema := metadata.MockSchema(2)
	createCtx := &CreateTableCtx{
		DBMutationCtx: *CreateDBMutationCtx(database, gen),
		Schema:        schema,
	}
	_, err := inst.CreateTable(createCtx)
	assert.Nil(t, err)
	createCtx.Id = gen.Alloc(database.GetShardId())
	_, err = inst.CreateTable(createCtx)
	assert.Equal(t, metadata.DuplicateErr, err)
}

func TestDropEmptyTable(t *testing.T) {
	initTestEnv(t)
	inst, gen, database := initTestDB1(t)
	defer inst.Close()

	schema := metadata.MockSchema(2)
	createCtx := &CreateTableCtx{
		DBMutationCtx: *CreateDBMutationCtx(database, gen),
		Schema:        schema,
	}
	_, err := inst.CreateTable(createCtx)
	assert.Nil(t, err)

	dropCtx := CreateTableMutationCtx(database, gen, schema.Name)
	_, err = inst.DropTable(dropCtx)
	assert.Nil(t, err)
}

func TestDropTable(t *testing.T) {
	initTestEnv(t)
	inst, gen, database := initTestDB1(t)
	defer inst.Close()

	name := "t1"
	schema := metadata.MockSchema(2)
	schema.Name = name

	createCtx := &CreateTableCtx{
		DBMutationCtx: *CreateDBMutationCtx(database, gen),
		Schema:        schema,
	}
	createMeta, err := inst.CreateTable(createCtx)
	assert.Nil(t, err)

	ssCtx := &dbi.GetSnapshotCtx{
		DBName:    database.Name,
		TableName: name,
		Cols:      []int{0},
		ScanAll:   true,
	}

	ss, err := inst.GetSnapshot(ssCtx)
	assert.Nil(t, err)
	ss.Close()

	dropCtx := CreateTableMutationCtx(database, gen, schema.Name)
	dropMeta, err := inst.DropTable(dropCtx)
	assert.Nil(t, err)
	assert.Equal(t, createMeta.Id, dropMeta.Id)

	ss, err = inst.GetSnapshot(ssCtx)
	assert.NotNil(t, err)
	assert.Nil(t, ss)

	createCtx.Id = gen.Alloc(database.GetShardId())
	createMeta2, err := inst.CreateTable(createCtx)
	assert.Nil(t, err)
	assert.NotEqual(t, createMeta.Id, createMeta2.Id)

	ss, err = inst.GetSnapshot(ssCtx)
	assert.Nil(t, err)
	ss.Close()
	t.Log(inst.Wal.String())
}

func TestSSOnMutation(t *testing.T) {
	initTestEnv(t)
	inst, gen, database := initTestDB1(t)
	defer inst.Close()
	schema := metadata.MockSchema(2)
	createCtx := &CreateTableCtx{
		DBMutationCtx: *CreateDBMutationCtx(database, gen),
		Schema:        schema,
	}
	_, err := inst.CreateTable(createCtx)
	assert.Nil(t, err)
	rows := inst.Store.Catalog.Cfg.BlockMaxRows / 10
	ck := mock.MockBatch(schema.Types(), rows)
	appendCtx := CreateAppendCtx(database, gen, schema.Name, ck)
	err = inst.Append(appendCtx)
	assert.Nil(t, err)
	err = inst.FlushTable(database.Name, schema.Name)
	assert.Nil(t, err)
	testutils.WaitExpect(200, func() bool {
		return database.GetCheckpointId() == gen.Get(database.GetShardId())
	})
	assert.Equal(t, database.GetCheckpointId(), gen.Get(database.GetShardId()))

	idx := database.GetCheckpointId()
	appendCtx.Id = gen.Alloc(database.GetShardId())
	err = inst.Append(appendCtx)

	view := database.View(idx)
	t.Log(view.Database.PString(metadata.PPL1, 0))
}

func TestAppend(t *testing.T) {
	initTestEnv(t)
	inst, gen, database := initTestDB1(t)

	schema := metadata.MockSchema(2)
	createCtx := &CreateTableCtx{
		DBMutationCtx: *CreateDBMutationCtx(database, gen),
		Schema:        schema,
	}

	tblMeta, err := inst.CreateTable(createCtx)
	assert.Nil(t, err)
	assert.NotNil(t, tblMeta)
	blkCnt := 2
	rows := inst.Store.Catalog.Cfg.BlockMaxRows * uint64(blkCnt)
	ck := mock.MockBatch(tblMeta.Schema.Types(), rows)
	assert.Equal(t, int(rows), ck.Vecs[0].Length())
	invalidName := "xxx"
	appendCtx := CreateAppendCtx(database, gen, invalidName, ck)
	err = inst.Append(appendCtx)
	assert.NotNil(t, err)
	insertCnt := 8
	appendCtx.Table = tblMeta.Schema.Name
	for i := 0; i < insertCnt; i++ {
		appendCtx.Id = gen.Alloc(database.GetShardId())
		err = inst.Append(appendCtx)
		assert.Nil(t, err)
	}

	cols := []int{0, 1}
	tbl, _ := inst.Store.DataTables.WeakRefTable(tblMeta.Id)
	segIds := tbl.SegmentIds()
	ssCtx := &dbi.GetSnapshotCtx{
		DBName:     database.Name,
		TableName:  schema.Name,
		SegmentIds: segIds,
		Cols:       cols,
	}

	blkCount := 0
	segCount := 0
	ss, err := inst.GetSnapshot(ssCtx)
	assert.Nil(t, err)
	segIt := ss.NewIt()
	assert.NotNil(t, segIt)

	for segIt.Valid() {
		segCount++
		segH := segIt.GetHandle()
		assert.NotNil(t, segH)
		blkIt := segH.NewIt()
		// segH.Close()
		assert.NotNil(t, blkIt)
		for blkIt.Valid() {
			blkCount++
			blkIt.Next()
		}
		blkIt.Close()
		segIt.Next()
	}
	segIt.Close()
	assert.Equal(t, insertCnt, segCount)
	assert.Equal(t, blkCnt*insertCnt, blkCount)
	ss.Close()

	ssPath := prepareSnapshotPath(defaultSnapshotPath, t)

	createSSCtx := &CreateSnapshotCtx{
		DB:   database.Name,
		Path: ssPath,
		Sync: true,
	}
	idx, err := inst.CreateSnapshot(createSSCtx)
	assert.Nil(t, err)
	assert.Equal(t, database.GetCheckpointId(), idx)

	applySSCtx := &ApplySnapshotCtx{
		DB:   database.Name,
		Path: ssPath,
	}
	err = inst.ApplySnapshot(applySSCtx)
	assert.Nil(t, err)
	assert.True(t, database.IsDeleted())

	database2, err := database.Catalog.SimpleGetDatabaseByName(database.Name)
	assert.Nil(t, err)

	tMeta, err := inst.Store.Catalog.SimpleGetTableByName(database.Name, schema.Name)
	assert.Nil(t, err)
	t.Log(tMeta.PString(metadata.PPL0, 0))

	err = inst.Append(appendCtx)
	assert.Nil(t, err)
	err = inst.FlushTable(database2.Name, schema.Name)
	t.Log(inst.Wal.String())
	gen.Reset(database2.GetShardId(), appendCtx.Id)

	schema2 := metadata.MockSchema(3)
	createCtx.Id = gen.Alloc(database2.GetShardId())
	createCtx.Schema = schema2
	_, err = inst.CreateTable(createCtx)
	assert.Nil(t, err)
	testutils.WaitExpect(200, func() bool {
		return database2.GetCheckpointId() == gen.Get(database2.GetShardId())
	})
	assert.Equal(t, database2.GetCheckpointId(), gen.Get(database2.GetShardId()))
	inst.Store.Catalog.Compact(nil, nil)

	// t.Log(inst.MTBufMgr.String())
	// t.Log(inst.SSTBufMgr.String())
	// t.Log(inst.IndexBufMgr.String())
	// t.Log(inst.FsMgr.String())
	// t.Log(tbl.GetIndexHolder().String())
	t.Log(inst.Wal.String())
	inst.Close()
}

func TestConcurrency(t *testing.T) {
	initTestEnv(t)
	inst, gen, database := initTestDB3(t)
	schema := metadata.MockSchema(2)

	shardId := database.GetShardId()
	createCtx := &CreateTableCtx{
		DBMutationCtx: *CreateDBMutationCtx(database, gen),
		Schema:        schema,
	}
	tblMeta, err := inst.CreateTable(createCtx)
	assert.Nil(t, err)
	assert.NotNil(t, tblMeta)
	blkCnt := inst.Store.Catalog.Cfg.SegmentMaxBlocks
	rows := inst.Store.Catalog.Cfg.BlockMaxRows * blkCnt
	baseCk := mock.MockBatch(tblMeta.Schema.Types(), rows)
	insertCh := make(chan *AppendCtx)
	searchCh := make(chan *dbi.GetSnapshotCtx)

	p, _ := ants.NewPool(40)

	reqCtx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	var searchWg sync.WaitGroup
	wg.Add(1)
	go func(ctx context.Context) {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case req := <-searchCh:
				f := func() {
					defer searchWg.Done()
					{
						ss, err := inst.GetSnapshot(req)
						assert.Nil(t, err)
						segIt := ss.NewIt()
						assert.Nil(t, err)
						if segIt == nil {
							return
						}
						segCnt := 0
						blkCnt := 0
						for segIt.Valid() {
							segCnt++
							sh := segIt.GetHandle()
							blkIt := sh.NewIt()
							for blkIt.Valid() {
								blkCnt++
								blkHandle := blkIt.GetHandle()
								hh := blkHandle.Prefetch()
								hh.Close()
								// blkHandle.Close()
								blkIt.Next()
							}
							blkIt.Close()
							segIt.Next()
						}
						segIt.Close()
						ss.Close()
					}
				}
				p.Submit(f)
			case req := <-insertCh:
				wg.Add(1)
				go func() {
					err := inst.Append(req)
					assert.Nil(t, err)
					wg.Done()
				}()
			}
		}
	}(reqCtx)

	insertCnt := 8
	var wg2 sync.WaitGroup

	wg2.Add(1)
	go func() {
		defer wg2.Done()
		for i := 0; i < insertCnt; i++ {
			insertReq := CreateAppendCtx(database, gen, schema.Name, baseCk)
			insertCh <- insertReq
		}
	}()

	cols := make([]int, 0)
	for i := 0; i < len(tblMeta.Schema.ColDefs); i++ {
		cols = append(cols, i)
	}
	wg2.Add(1)
	go func() {
		defer wg2.Done()
		reqCnt := 10000
		for i := 0; i < reqCnt; i++ {
			tbl, _ := inst.Store.DataTables.WeakRefTable(tblMeta.Id)
			for tbl == nil {
				time.Sleep(time.Duration(100) * time.Microsecond)
				tbl, _ = inst.Store.DataTables.WeakRefTable(tblMeta.Id)
			}
			segIds := tbl.SegmentIds()
			searchReq := &dbi.GetSnapshotCtx{
				ShardId:    shardId,
				DBName:     database.Name,
				TableName:  schema.Name,
				SegmentIds: segIds,
				Cols:       cols,
			}
			searchWg.Add(1)
			searchCh <- searchReq
		}
	}()

	wg2.Wait()
	searchWg.Wait()
	cancel()
	wg.Wait()
	tbl, _ := inst.Store.DataTables.WeakRefTable(tblMeta.Id)
	root := tbl.WeakRefRoot()
	testutils.WaitExpect(300, func() bool {
		return int64(1) == root.RefCount()
	})

	assert.Equal(t, int64(1), root.RefCount())
	opts := &dbi.GetSnapshotCtx{
		ShardId:   shardId,
		DBName:    database.Name,
		TableName: schema.Name,
		Cols:      cols,
		ScanAll:   true,
	}
	now := time.Now()
	ss, err := inst.GetSnapshot(opts)
	assert.Nil(t, err)
	segIt := ss.NewIt()
	segCnt := 0
	tblkCnt := 0
	for segIt.Valid() {
		segCnt++
		h := segIt.GetHandle()
		blkIt := h.NewIt()
		for blkIt.Valid() {
			tblkCnt++
			blkHandle := blkIt.GetHandle()
			// col0 := blkHandle.GetColumn(0)
			// ctx := index.NewFilterCtx(index.OpEq)
			// ctx.Val = int32(0 + col0.GetColIdx()*100)
			// err = col0.EvalFilter(ctx)
			// assert.Nil(t, err)
			// if col0.GetBlockType() > base.PERSISTENT_BLK {
			// 	assert.False(t, ctx.BoolRes)
			// }
			// ctx.Reset()
			// ctx.Code = index.OpEq
			// ctx.Val = int32(1 + col0.GetColIdx()*100)
			// err = col0.EvalFilter(ctx)
			// assert.Nil(t, err)
			// if col0.GetBlockType() > base.PERSISTENT_BLK {
			// 	assert.True(t, ctx.BoolRes)
			// }
			hh := blkHandle.Prefetch()
			vec0, err := hh.GetReaderByAttr(1)
			assert.Nil(t, err)
			val, err := vec0.GetValue(22)
			assert.Nil(t, err)
			t.Logf("vec0[22]=%s, type=%d", val, vec0.GetType())
			hh.Close()
			// blkHandle.Close()
			blkIt.Next()
		}
		blkIt.Close()
		// h.Close()
		segIt.Next()
	}
	segIt.Close()
	ss.Close()
	assert.Equal(t, insertCnt*int(blkCnt), tblkCnt)
	assert.Equal(t, insertCnt, segCnt)
	assert.Equal(t, int64(1), root.RefCount())

	t.Logf("Takes %v", time.Since(now))
	t.Log(tbl.String())
	time.Sleep(time.Duration(100) * time.Millisecond)
	if invariants.RaceEnabled {
		time.Sleep(time.Duration(400) * time.Millisecond)
	}

	t.Log(inst.MTBufMgr.String())
	t.Log(inst.SSTBufMgr.String())
	t.Log(inst.MemTableMgr.String())
	// t.Log(inst.IndexBufMgr.String())
	// t.Log(tbl.GetIndexHolder().String())
	// t.Log(common.GPool.String())
	inst.Close()
}

func TestMultiTables(t *testing.T) {
	initTestEnv(t)
	inst, gen, database := initTestDB3(t)
	prefix := "mtable"
	tblCnt := 8
	var names []string
	for i := 0; i < tblCnt; i++ {
		name := fmt.Sprintf("%s_%d", prefix, i)
		schema := metadata.MockSchema(2)
		schema.Name = name
		createCtx := &CreateTableCtx{
			DBMutationCtx: *CreateDBMutationCtx(database, gen),
			Schema:        schema,
		}
		_, err := inst.CreateTable(createCtx)
		assert.Nil(t, err)
		names = append(names, name)
	}
	tblMeta := database.SimpleGetTableByName(names[0])
	assert.NotNil(t, tblMeta)
	rows := uint64(tblMeta.Database.Catalog.Cfg.BlockMaxRows / 2)
	baseCk := mock.MockBatch(tblMeta.Schema.Types(), rows)
	p1, _ := ants.NewPool(4)
	p2, _ := ants.NewPool(4)
	attrs := []string{}
	for _, colDef := range tblMeta.Schema.ColDefs {
		attrs = append(attrs, colDef.Name)
	}
	refs := make([]uint64, len(attrs))

	tnames := database.SimpleGetTableNames()
	assert.Equal(t, len(names), len(tnames))
	insertCnt := 7
	searchCnt := 100
	var wg sync.WaitGroup
	for i := 0; i < insertCnt; i++ {
		for _, name := range tnames {
			task := func(tname string) func() {
				return func() {
					defer wg.Done()
					appendCtx := CreateAppendCtx(database, gen, tname, baseCk)
					err := inst.Append(appendCtx)
					assert.Nil(t, err)
				}
			}
			wg.Add(1)
			p1.Submit(task(name))
		}
	}

	doneCB := func() {
		wg.Done()
	}
	for i := 0; i < searchCnt; i++ {
		for _, name := range tnames {
			task := func(opIdx uint64, tname string, donecb func()) func() {
				return func() {
					if donecb != nil {
						defer donecb()
					}
					rel, err := inst.Relation(database.Name, tname)
					assert.Nil(t, err)
					for _, segId := range rel.SegmentIds().Ids {
						seg := rel.Segment(segId, nil)
						for _, id := range seg.Blocks() {
							blk := seg.Block(id, nil)
							cds := make([]*bytes.Buffer, len(attrs))
							dds := make([]*bytes.Buffer, len(attrs))
							for i := range cds {
								cds[i] = bytes.NewBuffer(make([]byte, 0))
								dds[i] = bytes.NewBuffer(make([]byte, 0))
							}
							bat, err := blk.Read(refs, attrs, cds, dds)
							//{
							//	for i, attr := range bat.Attrs {
							//		if bat.Vecs[i], err = bat.Is[i].R.Read(bat.Is[i].Len, bat.Is[i].Ref, attr, proc); err != nil {
							//			log.Fatal(err)
							//		}
							//	}
							//}
							assert.Nil(t, err)
							for attri, attr := range attrs {
								v := bat.GetVector(attr)
								if attri == 0 && v.Length() > 5000 {
									// edata := baseCk.Vecs[attri].Col.([]int32)

									// odata := v.Col.([]int32)

									// assert.Equal(t, edata[4999], data[4999])
									// assert.Equal(t, edata[5000], data[5000])

									// t.Logf("data[4998]=%d", data[4998])
									// t.Logf("data[4999]=%d", data[4999])

									// t.Logf("data[5000]=%d", data[5000])
									// t.Logf("data[5001]=%d", data[5001])
								}
								assert.True(t, v.Length() <= int(tblMeta.Database.Catalog.Cfg.BlockMaxRows))
								// t.Logf("%s, seg=%v, blk=%v, attr=%s, len=%d", tname, segId, id, attr, v.Length())
							}
						}
					}
					rel.Close()
				}
			}
			wg.Add(1)
			p2.Submit(task(uint64(i), name, doneCB))
		}
	}
	wg.Wait()
	time.Sleep(time.Duration(100) * time.Millisecond)
	{
		for _, name := range names {
			rel, err := inst.Relation(database.Name, name)
			assert.Nil(t, err)
			sids := rel.SegmentIds().Ids
			segId := sids[len(sids)-1]
			seg := rel.Segment(segId, nil)
			blks := seg.Blocks()
			blk := seg.Block(blks[len(blks)-1], nil)
			cds := make([]*bytes.Buffer, len(attrs))
			dds := make([]*bytes.Buffer, len(attrs))
			for i := range cds {
				cds[i] = bytes.NewBuffer(make([]byte, 0))
				dds[i] = bytes.NewBuffer(make([]byte, 0))
			}
			bat, err := blk.Read(refs, attrs, cds, dds)
			//{
			//	for i, attr := range bat.Attrs {
			//		if bat.Vecs[i], err = bat.Is[i].R.Read(bat.Is[i].Len, bat.Is[i].Ref, attr, proc); err != nil {
			//			log.Fatal(err)
			//		}
			//	}
			//}
			assert.Nil(t, err)
			for _, attr := range attrs {
				v := bat.GetVector(attr)
				assert.Equal(t, int(rows), v.Length())
				// t.Log(v.Length())
				// t.Logf("%s, seg=%v, attr=%s, len=%d", name, segId, attr, v.Length())
			}
		}
	}
	t.Log(inst.MTBufMgr.String())
	t.Log(inst.SSTBufMgr.String())
	inst.Close()
}

func TestDropTable2(t *testing.T) {
	initTestEnv(t)
	inst, gen, database := initTestDB3(t)
	schema := metadata.MockSchema(2)
	createCtx := &CreateTableCtx{
		DBMutationCtx: *CreateDBMutationCtx(database, gen),
		Schema:        schema,
	}
	tblMeta, err := inst.CreateTable(createCtx)
	assert.Nil(t, err)
	assert.NotNil(t, tblMeta)
	blkCnt := inst.Store.Catalog.Cfg.SegmentMaxBlocks
	rows := inst.Store.Catalog.Cfg.BlockMaxRows * blkCnt
	baseCk := mock.MockBatch(tblMeta.Schema.Types(), rows)

	insertCnt := uint64(1)

	var wg sync.WaitGroup
	{
		for i := uint64(0); i < insertCnt; i++ {
			wg.Add(1)
			go func() {
				appendCtx := CreateAppendCtx(database, gen, schema.Name, baseCk)
				inst.Append(appendCtx)
				wg.Done()
			}()
		}
	}
	wg.Wait()
	testutils.WaitExpect(100, func() bool {
		return int(blkCnt*insertCnt) == inst.SSTBufMgr.NodeCount()+inst.MTBufMgr.NodeCount()
	})

	t.Log(inst.MTBufMgr.String())
	t.Log(inst.SSTBufMgr.String())
	assert.Equal(t, int(blkCnt*insertCnt), inst.SSTBufMgr.NodeCount()+inst.MTBufMgr.NodeCount())
	cols := make([]int, 0)
	for i := 0; i < len(tblMeta.Schema.ColDefs); i++ {
		cols = append(cols, i)
	}
	opts := &dbi.GetSnapshotCtx{
		DBName:    database.Name,
		TableName: schema.Name,
		Cols:      cols,
		ScanAll:   true,
	}
	ss, err := inst.GetSnapshot(opts)
	assert.Nil(t, err)

	assert.True(t, database.GetSize() > 0)

	dropCtx := CreateTableMutationCtx(database, gen, schema.Name)
	_, err = inst.DropTable(dropCtx)
	assert.Nil(t, err)
	dropDBCtx := &DropDBCtx{
		DB: database.Name,
		Id: gen.Alloc(database.GetShardId()),
	}
	_, err = inst.DropDatabase(dropDBCtx)
	assert.Nil(t, err)

	testutils.WaitExpect(50, func() bool {
		return int(blkCnt*insertCnt) == inst.SSTBufMgr.NodeCount()+inst.MTBufMgr.NodeCount()
	})
	assert.Equal(t, int(blkCnt*insertCnt), inst.SSTBufMgr.NodeCount()+inst.MTBufMgr.NodeCount())
	ss.Close()

	testutils.WaitExpect(50, func() bool {
		return inst.SSTBufMgr.NodeCount()+inst.MTBufMgr.NodeCount() == 0
	})
	t.Log(inst.MTBufMgr.String())
	t.Log(inst.SSTBufMgr.String())
	t.Log(inst.MutationBufMgr.String())
	// t.Log(inst.IndexBufMgr.String())
	// t.Log(inst.MemTableMgr.String())
	assert.Equal(t, 0, inst.SSTBufMgr.NodeCount()+inst.MTBufMgr.NodeCount())
	testutils.WaitExpect(100, func() bool {
		return database.GetSize() == int64(0)
	})
	assert.Equal(t, int64(0), database.GetSize())
	inst.Close()
}

func TestE2E(t *testing.T) {
	if !dataio.FlushIndex {
		dataio.FlushIndex = true
		defer func() {
			dataio.FlushIndex = false
		}()
	}
	waitTime := time.Duration(100) * time.Millisecond
	if invariants.RaceEnabled {
		waitTime *= 2
	}
	initTestEnv(t)
	inst, gen, database := initTestDB3(t)
	schema := metadata.MockSchema(2)
	createCtx := &CreateTableCtx{
		DBMutationCtx: *CreateDBMutationCtx(database, gen),
		Schema:        schema,
	}
	tblMeta, err := inst.CreateTable(createCtx)
	assert.Nil(t, err)
	assert.NotNil(t, tblMeta)
	blkCnt := inst.Store.Catalog.Cfg.SegmentMaxBlocks
	rows := inst.Store.Catalog.Cfg.BlockMaxRows * blkCnt
	baseCk := mock.MockBatch(tblMeta.Schema.Types(), rows)

	insertCnt := uint64(10)

	var wg sync.WaitGroup
	{
		for i := uint64(0); i < insertCnt; i++ {
			wg.Add(1)
			go func() {
				appendCtx := CreateAppendCtx(database, gen, schema.Name, baseCk)
				inst.Append(appendCtx)
				wg.Done()
			}()
		}
	}
	wg.Wait()
	time.Sleep(waitTime)
	tblData, err := inst.Store.DataTables.WeakRefTable(tblMeta.Id)
	assert.Nil(t, err)
	t.Log(tblData.String())
	// t.Log(tblData.GetIndexHolder().String())

	segs := tblData.SegmentIds()
	for _, segId := range segs {
		seg := tblData.WeakRefSegment(segId)
		seg.GetIndexHolder().Init(seg.GetSegmentFile())
		//t.Log(seg.GetIndexHolder().Inited)
		segment := &db.Segment{
			Data: seg,
			Ids:  new(atomic.Value),
		}
		spf := segment.NewSparseFilter()
		f := segment.NewFilter()
		sumr := segment.NewSummarizer()
		t.Log(spf.Eq("mock_0", int32(1)))
		t.Log(f == nil)
		t.Log(sumr.Count("mock_0", nil))
		//t.Log(spf.Eq("mock_0", int32(1)))
		//t.Log(f.Eq("mock_0", int32(-1)))
		//t.Log(sumr.Count("mock_0", nil))
		t.Log(inst.IndexBufMgr.String())
	}

	time.Sleep(waitTime)
	t.Log(inst.IndexBufMgr.String())

	dropCtx := CreateTableMutationCtx(database, gen, schema.Name)
	_, err = inst.DropTable(dropCtx)
	assert.Nil(t, err)
	time.Sleep(waitTime / 2)

	t.Log(inst.FsMgr.String())
	t.Log(inst.MTBufMgr.String())
	t.Log(inst.SSTBufMgr.String())
	t.Log(inst.IndexBufMgr.String())
	inst.Close()
}
func TestEngine(t *testing.T) {
	initTestEnv(t)
	inst, gen, database := initTestDB3(t)
	schema := metadata.MockSchema(2)
	shardId := database.GetShardId()
	createCtx := &CreateTableCtx{
		DBMutationCtx: *CreateDBMutationCtx(database, gen),
		Schema:        schema,
	}
	tblMeta, err := inst.CreateTable(createCtx)
	assert.Nil(t, err)
	assert.NotNil(t, tblMeta)
	blkCnt := inst.Store.Catalog.Cfg.SegmentMaxBlocks
	rows := inst.Store.Catalog.Cfg.BlockMaxRows * blkCnt
	baseCk := mock.MockBatch(tblMeta.Schema.Types(), rows)
	insertCh := make(chan *AppendCtx)
	searchCh := make(chan *dbi.GetSnapshotCtx)

	p, _ := ants.NewPool(40)
	attrs := []string{}
	cols := make([]int, 0)
	for i, colDef := range tblMeta.Schema.ColDefs {
		attrs = append(attrs, colDef.Name)
		cols = append(cols, i)
	}
	hm := host.New(1 << 40)
	gm := guest.New(1<<40, hm)
	proc := process.New(gm)

	tableCnt := 20
	var twg sync.WaitGroup
	for i := 0; i < tableCnt; i++ {
		twg.Add(1)
		f := func(idx int) func() {
			return func() {
				schema := metadata.MockSchema(2)
				schema.Name = fmt.Sprintf("%s-%d", schema.Name, idx)
				createCtx := &CreateTableCtx{
					DBMutationCtx: *CreateDBMutationCtx(database, gen),
					Schema:        schema,
				}
				_, err := inst.CreateTable(createCtx)
				assert.Nil(t, err)
				twg.Done()
			}
		}
		p.Submit(f(i))
	}

	reqCtx, cancel := context.WithCancel(context.Background())
	var (
		loopWg   sync.WaitGroup
		searchWg sync.WaitGroup
		loadCnt  uint32
	)
	assert.Nil(t, err)
	task := func(ctx *dbi.GetSnapshotCtx) func() {
		return func() {
			defer searchWg.Done()
			rel, err := inst.Relation(database.Name, tblMeta.Schema.Name)
			assert.Nil(t, err)
			for _, segId := range rel.SegmentIds().Ids {
				seg := rel.Segment(segId, proc)
				for _, id := range seg.Blocks() {
					blk := seg.Block(id, proc)
					blk.Prefetch(attrs)
					//assert.Nil(t, err)
					//for _, attr := range attrs {
					//	bat.GetVector(attr, proc)
					//	atomic.AddUint32(&loadCnt, uint32(1))
					//}
				}
			}
			rel.Close()
		}
	}
	assert.NotNil(t, task)
	task2 := func(ctx *dbi.GetSnapshotCtx) func() {
		return func() {
			defer searchWg.Done()
			ss, err := inst.GetSnapshot(ctx)
			assert.Nil(t, err)
			segIt := ss.NewIt()
			assert.Nil(t, err)
			if segIt == nil {
				return
			}
			for segIt.Valid() {
				sh := segIt.GetHandle()
				blkIt := sh.NewIt()
				for blkIt.Valid() {
					blkHandle := blkIt.GetHandle()
					hh := blkHandle.Prefetch()
					for idx, _ := range attrs {
						hh.GetReaderByAttr(idx)
						atomic.AddUint32(&loadCnt, uint32(1))
					}
					hh.Close()
					// blkHandle.Close()
					blkIt.Next()
				}
				blkIt.Close()
				segIt.Next()
			}
			segIt.Close()
		}
	}
	assert.NotNil(t, task2)
	loopWg.Add(1)
	loop := func(ctx context.Context) {
		defer loopWg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case req := <-searchCh:
				p.Submit(task2(req))
			case req := <-insertCh:
				loopWg.Add(1)
				t := func() {
					err := inst.Append(req)
					assert.Nil(t, err)
					loopWg.Done()
				}
				go t()
			}
		}
	}
	go loop(reqCtx)

	insertCnt := 8
	var driverWg sync.WaitGroup
	driverWg.Add(1)
	go func() {
		defer driverWg.Done()
		for i := 0; i < insertCnt; i++ {
			req := CreateAppendCtx(database, gen, schema.Name, baseCk)
			insertCh <- req
		}
	}()

	time.Sleep(time.Duration(500) * time.Millisecond)
	searchCnt := 10000
	driverWg.Add(1)
	go func() {
		defer driverWg.Done()
		for i := 0; i < searchCnt; i++ {
			req := &dbi.GetSnapshotCtx{
				ShardId:   shardId,
				DBName:    database.Name,
				TableName: schema.Name,
				ScanAll:   true,
				Cols:      cols,
			}
			searchWg.Add(1)
			searchCh <- req
		}
	}()
	driverWg.Wait()
	searchWg.Wait()
	cancel()
	loopWg.Wait()
	twg.Wait()
	t.Log(inst.MTBufMgr.String())
	t.Log(inst.SSTBufMgr.String())
	t.Log(inst.MemTableMgr.String())
	t.Logf("Load: %d", loadCnt)
	tbl, err := inst.Store.DataTables.WeakRefTable(tblMeta.Id)
	assert.Equal(t, tbl.GetRowCount(), rows*uint64(insertCnt))
	t.Log(tbl.GetRowCount())
	attr := tblMeta.Schema.ColDefs[0].Name
	t.Log(tbl.Size(attr))
	attr = tblMeta.Schema.ColDefs[1].Name
	t.Log(tbl.Size(attr))
	t.Log(tbl.String())
	rel, err := inst.Relation(database.Name, tblMeta.Schema.Name)
	assert.Nil(t, err)
	t.Logf("Rows: %d, Size: %d", rel.Rows(), rel.Size(tblMeta.Schema.ColDefs[0].Name))
	t.Log(inst.GetSegmentIds(database.Name, tblMeta.Schema.Name))
	t.Log(common.GPool.String())
	inst.Close()
}

func TestLogIndex(t *testing.T) {
	initTestEnv(t)
	inst, gen, database := initTestDB3(t)
	schema := metadata.MockSchema(2)
	shardId := database.GetShardId()
	createCtx := &CreateTableCtx{
		DBMutationCtx: *CreateDBMutationCtx(database, gen),
		Schema:        schema,
	}
	tblMeta, err := inst.CreateTable(createCtx)
	assert.Nil(t, err)
	assert.NotNil(t, tblMeta)
	rows := inst.Store.Catalog.Cfg.BlockMaxRows * 2 / 5
	baseCk := mock.MockBatch(tblMeta.Schema.Types(), rows)

	appendCtx := new(AppendCtx)
	for i := 0; i < 50; i++ {
		appendCtx = CreateAppendCtx(database, gen, schema.Name, baseCk)
		err = inst.Append(appendCtx)
		assert.Nil(t, err)
	}

	dropCtx := CreateTableMutationCtx(database, gen, schema.Name)
	_, err = inst.DropTable(dropCtx)
	assert.Nil(t, err)
	testutils.WaitExpect(500, func() bool {
		return inst.GetShardCheckpointId(shardId) == inst.Wal.GetShardCurrSeqNum(shardId)
	})
	// assert.Equal(t, gen.Get(shardId), inst.Wal.GetShardCurrSeqNum(shardId))
	assert.Equal(t, inst.Wal.GetShardCurrSeqNum(shardId), inst.GetShardCheckpointId(shardId))

	inst.Close()
}

func TestMultiInstance(t *testing.T) {
	dir := initTestEnv(t)
	var dirs []string
	for i := 0; i < 10; i++ {
		dirs = append(dirs, path.Join(dir, fmt.Sprintf("wd%d", i)))
	}
	var insts []*DB
	for _, d := range dirs {
		opts := storage.Options{}
		inst, _ := Open(d, &opts)
		insts = append(insts, inst)
		defer inst.Close()
	}

	gen := shard.NewMockIndexAllocator()
	shardId := uint64(100)

	var schema *metadata.Schema
	for _, inst := range insts {
		db, err := inst.Store.Catalog.SimpleCreateDatabase("db1", gen.Next(shardId))
		assert.Nil(t, err)
		schema = metadata.MockSchema(2)
		schema.Name = "xxx"
		createCtx := &CreateTableCtx{
			DBMutationCtx: *CreateDBMutationCtx(db, gen),
			Schema:        schema,
		}
		_, err = inst.CreateTable(createCtx)
		assert.Nil(t, err)
	}
	meta, err := insts[0].Store.Catalog.SimpleGetTableByName("db1", schema.Name)
	assert.Nil(t, err)
	bat := mock.MockBatch(meta.Schema.Types(), 100)
	for _, inst := range insts {
		database, err := inst.Store.Catalog.SimpleGetDatabaseByName("db1")
		assert.Nil(t, err)
		appendCtx := CreateAppendCtx(database, gen, schema.Name, bat)
		err = inst.Append(appendCtx)
		assert.Nil(t, err)
	}

	time.Sleep(time.Duration(50) * time.Millisecond)
}