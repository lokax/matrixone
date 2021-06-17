package dataio

import (
	"io"
	"matrixone/pkg/vm/engine/aoe/storage/common"
	"matrixone/pkg/vm/engine/aoe/storage/layout/base"
)

type Key struct {
	Col uint64
	ID  common.ID
	// BlockID uint64
	// PartID  uint32
}

type IManager interface {
	RegisterSortedFiles(common.ID) (ISegmentFile, error)
	RegisterUnsortedFiles(common.ID) (ISegmentFile, error)
	UpgradeFile(common.ID) ISegmentFile
	GetSortedFile(common.ID) ISegmentFile
	GetUnsortedFile(common.ID) ISegmentFile
	String() string
}

type ISegmentFile interface {
	io.Closer
	Destory()
	RefBlock(blkId common.ID)
	UnrefBlock(blkId common.ID)
	RefIndex()
	UnrefIndex()
	// RefBlockIndex(common.ID)
	// UnrefBlockIndex(common.ID)
	MakeColPartFile(id *common.ID) IColPartFile
	ReadPart(colIdx uint64, id common.ID, buf []byte)
	ReadPoint(ptr *base.Pointer, buf []byte)
	ReadBlockPoint(id common.ID, ptr *base.Pointer, buf []byte)
}

type IColSegmentFile interface {
	ReadPart(id common.ID, buf []byte)
}

type IColPartFile interface {
	io.Reader
}

type IIndexFile interface {
	io.Reader
}