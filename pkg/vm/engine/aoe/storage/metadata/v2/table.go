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

package metadata

import (
	"encoding/json"
	"fmt"
	"matrixone/pkg/logutil"
	"matrixone/pkg/vm/engine/aoe/storage/common"
	"matrixone/pkg/vm/engine/aoe/storage/logstore"
	"sync"
)

type tableLogEntry struct {
	BaseEntry
	Prev    *Table
	Catalog *Catalog `json:"-"`
}

func (e *tableLogEntry) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

func (e *tableLogEntry) Unmarshal(buf []byte) error {
	return json.Unmarshal(buf, e)
}

func (e *tableLogEntry) ToEntry() *Table {
	e.BaseEntry.CommitInfo.SetNext(e.Prev.CommitInfo)
	e.Prev.BaseEntry = e.BaseEntry
	return e.Prev
}

// func createTableHandle(r io.Reader, meta *LogEntryMeta) (LogEntry, int64, error) {
// 	entry := Table{}
// 	logEntry
// 	// entry.Unmarshal()

// }

type Table struct {
	BaseEntry
	Schema     *Schema
	SegmentSet []*Segment
	IdIndex    map[uint64]int `json:"-"`
	Catalog    *Catalog       `json:"-"`
}

func NewTableEntry(catalog *Catalog, schema *Schema, tranId uint64, exIndex *ExternalIndex) *Table {
	schema.BlockMaxRows = catalog.Cfg.BlockMaxRows
	schema.SegmentMaxBlocks = catalog.Cfg.SegmentMaxBlocks
	e := &Table{
		BaseEntry: BaseEntry{
			Id: catalog.NextTableId(),
			CommitInfo: &CommitInfo{
				TranId:        tranId,
				CommitId:      tranId,
				SSLLNode:      *common.NewSSLLNode(),
				Op:            OpCreate,
				ExternalIndex: exIndex,
			},
		},
		Schema:     schema,
		Catalog:    catalog,
		SegmentSet: make([]*Segment, 0),
		IdIndex:    make(map[uint64]int),
	}
	return e
}

func NewEmptyTableEntry(catalog *Catalog) *Table {
	e := &Table{
		BaseEntry: BaseEntry{
			CommitInfo: &CommitInfo{
				SSLLNode: *common.NewSSLLNode(),
			},
		},
		SegmentSet: make([]*Segment, 0),
		IdIndex:    make(map[uint64]int),
		Catalog:    catalog,
	}
	return e
}

// Threadsafe
// It is used to take a snapshot of table base on a commit id. It goes through
// the version chain to find a "safe" commit version and create a view base on
// that version.
// v2(commitId=7) -> v1(commitId=4) -> v0(commitId=2)
//      |                 |                  |
//      |                 |                   -------- CommittedView [0,2]
//      |                  --------------------------- CommittedView [4,6]
//       --------------------------------------------- CommittedView [7,+oo)
func (e *Table) CommittedView(id uint64) *Table {
	// TODO: if baseEntry op is drop, should introduce an index to
	// indicate weather to return nil
	baseEntry := e.UseCommitted(id)
	if baseEntry == nil {
		return nil
	}
	view := &Table{
		Schema:     e.Schema,
		BaseEntry:  *baseEntry,
		SegmentSet: make([]*Segment, 0),
	}
	e.RLock()
	segs := make([]*Segment, 0, len(e.SegmentSet))
	for _, seg := range e.SegmentSet {
		segs = append(segs, seg)
	}
	e.RUnlock()
	for _, seg := range segs {
		segView := seg.CommittedView(id)
		if segView == nil {
			continue
		}
		view.SegmentSet = append(view.SegmentSet, segView)
	}
	return view
}

// Not threadsafe, and not needed
// Only used during data replay by the catalog replayer
func (e *Table) rebuild(catalog *Catalog) {
	e.Catalog = catalog
	e.IdIndex = make(map[uint64]int)
	for i, seg := range e.SegmentSet {
		catalog.Sequence.TryUpdateSegmentId(seg.Id)
		seg.rebuild(e)
		e.IdIndex[seg.Id] = i
	}
}

// Threadsafe
// It should be applied on a table that was previously soft-deleted
// It is always driven by engine internal scheduler. It means all the
// table related data resources were deleted. A hard-deleted table will
// be deleted from catalog later
func (e *Table) HardDelete() {
	cInfo := &CommitInfo{
		CommitId: e.Catalog.NextUncommitId(),
		Op:       OpHardDelete,
		SSLLNode: *common.NewSSLLNode(),
	}
	e.Lock()
	defer e.Unlock()
	if e.IsHardDeletedLocked() {
		logutil.Warnf("HardDelete %d but already hard deleted", e.Id)
		return
	}
	if !e.IsSoftDeletedLocked() {
		panic("logic error: Cannot hard delete entry that not soft deleted")
	}
	e.onNewCommit(cInfo)
	e.Catalog.Commit(e, ETHardDeleteTable, &e.RWMutex)
}

// Simple* wrappes simple usage of wrapped operation
func (e *Table) SimpleSoftDelete(exIndex *ExternalIndex) {
	e.SoftDelete(e.Catalog.NextUncommitId(), exIndex, true)
}

// Threadsafe
// It is driven by external command. The engine then schedules a GC task to hard delete
// related resources.
func (e *Table) SoftDelete(tranId uint64, exIndex *ExternalIndex, autoCommit bool) {
	cInfo := &CommitInfo{
		TranId:        tranId,
		CommitId:      tranId,
		ExternalIndex: exIndex,
		Op:            OpSoftDelete,
		SSLLNode:      *common.NewSSLLNode(),
	}
	e.Lock()
	defer e.Unlock()
	if e.IsSoftDeletedLocked() {
		return
	}
	e.onNewCommit(cInfo)

	if !autoCommit {
		return
	}
	e.Catalog.Commit(e, ETSoftDeleteTable, &e.RWMutex)
}

// Not safe
func (e *Table) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

// Not safe
func (e *Table) Unmarshal(buf []byte) error {
	return json.Unmarshal(buf, e)
}

// Not safe
func (e *Table) String() string {
	buf, _ := e.Marshal()
	return string(buf)
}

// Not safe
// Usually it is used during creating a table. We need to commit the new table entry
// to the store.
func (e *Table) ToLogEntry(eType LogEntryType) LogEntry {
	var buf []byte
	switch eType {
	case ETCreateTable:
		buf, _ = e.Marshal()
	case ETSoftDeleteTable:
		if !e.IsSoftDeletedLocked() {
			panic("logic error")
		}
		entry := tableLogEntry{
			BaseEntry: e.BaseEntry,
		}
		buf, _ = entry.Marshal()
	case ETHardDeleteTable:
		if !e.IsHardDeletedLocked() {
			panic("logic error")
		}
		entry := tableLogEntry{
			BaseEntry: e.BaseEntry,
		}
		buf, _ = entry.Marshal()
	default:
		panic("not supported")
	}
	logEntry := logstore.GetEmptyEntry()
	logEntry.Meta.SetType(eType)
	logEntry.Unmarshal(buf)
	return logEntry
}

// Safe
func (e *Table) SimpleGetCurrSegment() *Segment {
	e.RLock()
	if len(e.SegmentSet) == 0 {
		e.RUnlock()
		return nil
	}
	seg := e.SegmentSet[len(e.SegmentSet)-1]
	e.RUnlock()
	return seg
}

// Not safe and no need
// Only used during data replay
// TODO: Only compatible with v1. Remove later
func (e *Table) GetReplayIndex() *LogIndex {
	for i := len(e.SegmentSet) - 1; i >= 0; i-- {
		seg := e.SegmentSet[i]
		idx := seg.GetReplayIndex()
		if idx != nil {
			return idx
		}
	}
	return nil
}

// Safe
// TODO: Only compatible with v1. Remove later
func (e *Table) GetAppliedIndex(rwmtx *sync.RWMutex) (uint64, bool) {
	if rwmtx == nil {
		e.RLock()
		defer e.RUnlock()
	}
	if e.IsDeletedLocked() {
		return e.BaseEntry.GetAppliedIndex()
	}
	var (
		id uint64
		ok bool
	)
	for i := len(e.SegmentSet) - 1; i >= 0; i-- {
		seg := e.SegmentSet[i]
		id, ok = seg.GetAppliedIndex(nil)
		if ok {
			break
		}
	}
	if !ok {
		return e.BaseEntry.GetAppliedIndex()
	}
	return id, ok
}

// Not safe. One writer, multi-readers
func (e *Table) SimpleCreateBlock(exIndex *ExternalIndex) (*Block, *Segment) {
	var prevSeg *Segment
	currSeg := e.SimpleGetCurrSegment()
	if currSeg == nil || currSeg.HasMaxBlocks() {
		prevSeg = currSeg
		currSeg = e.SimpleCreateSegment(exIndex)
	}
	blk := currSeg.SimpleCreateBlock(exIndex)
	return blk, prevSeg
}

func (e *Table) getFirstInfullSegment(from *Segment) (*Segment, *Segment) {
	if len(e.SegmentSet) == 0 {
		return nil, nil
	}
	var curr, next *Segment
	for i := len(e.SegmentSet) - 1; i >= 0; i-- {
		seg := e.SegmentSet[i]
		if seg.Appendable() && from.LE(seg) {
			curr, next = seg, curr
		} else {
			break
		}
	}
	return curr, next
}

// Not safe. One writer, multi-readers
func (e *Table) SimpleGetOrCreateNextBlock(from *Block) *Block {
	var fromSeg *Segment
	if from != nil {
		fromSeg = from.Segment
	}
	curr, next := e.getFirstInfullSegment(fromSeg)
	// logutil.Infof("%s, %s", seg.PString(PPL0), fromSeg.PString(PPL1))
	if curr == nil {
		curr = e.SimpleCreateSegment(nil)
	}
	blk := curr.SimpleGetOrCreateNextBlock(from)
	if blk != nil {
		return blk
	}
	if next == nil {
		next = e.SimpleCreateSegment(nil)
	}
	return next.SimpleGetOrCreateNextBlock(nil)
}

func (e *Table) SimpleCreateSegment(exIndex *ExternalIndex) *Segment {
	return e.CreateSegment(e.Catalog.NextUncommitId(), exIndex, true)
}

// Safe
func (e *Table) SimpleGetSegmentIds() []uint64 {
	e.RLock()
	defer e.RUnlock()
	arrLen := len(e.SegmentSet)
	ret := make([]uint64, arrLen)
	for i, seg := range e.SegmentSet {
		ret[i] = seg.Id
	}
	return ret
}

// Safe
func (e *Table) SimpleGetSegmentCount() int {
	e.RLock()
	defer e.RUnlock()
	return len(e.SegmentSet)
}

// Safe
func (e *Table) CreateSegment(tranId uint64, exIndex *ExternalIndex, autoCommit bool) *Segment {
	se := newSegmentEntry(e.Catalog, e, tranId, exIndex)
	e.Lock()
	e.onNewSegment(se)
	e.Unlock()
	if !autoCommit {
		return se
	}
	e.Catalog.Commit(se, ETCreateSegment, nil)
	return se
}

func (e *Table) onNewSegment(entry *Segment) {
	e.IdIndex[entry.Id] = len(e.SegmentSet)
	e.SegmentSet = append(e.SegmentSet, entry)
}

// Safe
func (e *Table) SimpleGetBlock(segId, blkId uint64) (*Block, error) {
	seg := e.SimpleGetSegment(segId)
	if seg == nil {
		return nil, SegmentNotFoundErr
	}
	blk := seg.SimpleGetBlock(blkId)
	if blk == nil {
		return nil, BlockNotFoundErr
	}
	return blk, nil
}

// Safe
func (e *Table) SimpleGetSegment(id uint64) *Segment {
	e.RLock()
	defer e.RUnlock()
	return e.GetSegment(id, MinUncommitId)
}

func (e *Table) GetSegment(id, tranId uint64) *Segment {
	pos, ok := e.IdIndex[id]
	if !ok {
		return nil
	}
	entry := e.SegmentSet[pos]
	return entry
}

// Not safe
func (e *Table) PString(level PPLevel) string {
	s := fmt.Sprintf("<Table[%s]>(%s)(Cnt=%d)", e.Schema.Name, e.BaseEntry.PString(level), len(e.SegmentSet))
	if level > PPL0 && len(e.SegmentSet) > 0 {
		s = fmt.Sprintf("%s{", s)
		for _, seg := range e.SegmentSet {
			s = fmt.Sprintf("%s\n%s", s, seg.PString(level))
		}
		s = fmt.Sprintf("%s\n}", s)
	}
	return s
}

func MockTable(catalog *Catalog, schema *Schema, blkCnt uint64, idx *LogIndex) *Table {
	if schema == nil {
		schema = MockSchema(2)
	}
	if idx == nil {
		idx = &LogIndex{
			Id: SimpleBatchId(common.NextGlobalSeqNum()),
		}
	}
	tbl, err := catalog.SimpleCreateTable(schema, idx)
	if err != nil {
		panic(err)
	}

	var activeSeg *Segment
	for i := uint64(0); i < blkCnt; i++ {
		if activeSeg == nil {
			activeSeg = tbl.SimpleCreateSegment(nil)
		}
		activeSeg.SimpleCreateBlock(nil)
		if len(activeSeg.BlockSet) == int(tbl.Schema.SegmentMaxBlocks) {
			activeSeg = nil
		}
	}
	return tbl
}