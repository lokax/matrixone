package meta

import (
	"matrixone/pkg/vm/engine/aoe/storage/events/memdata"
	"matrixone/pkg/vm/engine/aoe/storage/layout/table/v2/iface"
	md "matrixone/pkg/vm/engine/aoe/storage/metadata/v1"
	"matrixone/pkg/vm/engine/aoe/storage/sched"
	// log "github.com/sirupsen/logrus"
)

type createBlkEvent struct {
	baseEvent
	NewSegment bool
	TableID    uint64
	TableData  iface.ITableData
	Block      iface.IBlock
}

func NewCreateBlkEvent(ctx *Context, tid uint64, tableData iface.ITableData) *createBlkEvent {
	e := &createBlkEvent{
		TableData: tableData,
		TableID:   tid,
	}
	e.baseEvent = baseEvent{
		Ctx:       ctx,
		BaseEvent: *sched.NewBaseEvent(e, sched.MetaCreateBlkTask, ctx.DoneCB, ctx.Waitable),
	}
	return e
}
func (e *createBlkEvent) HasNewSegment() bool {
	return e.NewSegment
}

func (e *createBlkEvent) GetBlock() *md.Block {
	if e.Err != nil {
		return nil
	}
	return e.Result.(*md.Block)
}

func (e *createBlkEvent) Execute() error {
	table, err := e.Ctx.Opts.Meta.Info.ReferenceTable(e.TableID)
	if err != nil {
		return err
	}

	seg := table.GetActiveSegment()
	if seg == nil {
		seg = table.NextActiveSegment()
	}
	if seg == nil {
		seg, err = table.CreateSegment()
		if err != nil {
			return err
		}
		err = table.RegisterSegment(seg)
		if err != nil {
			return err
		}
		e.NewSegment = true
	}

	var cloned *md.Block
	blk := seg.GetActiveBlk()
	if blk == nil {
		blk = seg.NextActiveBlk()
	} else {
		seg.NextActiveBlk()
	}
	if blk == nil {
		blk, err = seg.CreateBlock()
		if err != nil {
			return err
		}
		err = seg.RegisterBlock(blk)
		if err != nil {
			return err
		}
		ctx := md.CopyCtx{}
		cloned, err = seg.CloneBlock(blk.ID, ctx)
		if err != nil {
			return err
		}
	} else {
		cloned = blk.Copy()
		cloned.Detach()
	}
	e.Result = cloned
	if e.TableData != nil {
		ctx := &memdata.Context{Opts: e.Ctx.Opts, Waitable: true}
		event := memdata.NewCreateSegBlkEvent(ctx, e.NewSegment, cloned, e.TableData)
		if err = e.Ctx.Opts.Scheduler.Schedule(event); err != nil {
			return err
		}
		if err = event.WaitDone(); err != nil {
			return err
		}
		e.Block = event.Block
	}
	return err
}