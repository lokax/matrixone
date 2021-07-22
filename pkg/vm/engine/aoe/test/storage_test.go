package test

import (
	"bytes"
	"fmt"
	"github.com/fagongzi/util/format"
	"github.com/matrixorigin/matrixcube/pb/bhmetapb"
	"github.com/matrixorigin/matrixcube/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	stdLog "log"
	"matrixone/pkg/container/types"
	"matrixone/pkg/sql/protocol"
	"matrixone/pkg/vm/engine/aoe/common/helper"
	daoe "matrixone/pkg/vm/engine/aoe/dist/aoe"
	"matrixone/pkg/vm/engine/aoe/dist/pb"
	"matrixone/pkg/vm/engine/aoe/dist/testutil"
	e "matrixone/pkg/vm/engine/aoe/storage"
	md "matrixone/pkg/vm/engine/aoe/storage/metadata/v1"
	"matrixone/pkg/vm/engine/aoe/storage/mock/type/chunk"
	"testing"
	"time"
)

const (
	blockRows          uint64 = 2
	blockCntPerSegment uint64 = 4
	insertRows                = blockRows * blockCntPerSegment * 10
	insertCnt          uint64 = 20
	batchInsertRows           = insertRows / insertCnt
)


func TestAOEStorage(t *testing.T) {
	c, err := testutil.NewTestClusterStore(t, func(path string) (storage.DataStorage, error) {
		opts     := &e.Options{}
		mdCfg := &md.Configuration{
			Dir:              path,
			SegmentMaxBlocks: blockCntPerSegment,
			BlockMaxRows:     blockRows,
		}
		opts.CacheCfg = &e.CacheCfg{
			IndexCapacity:  blockRows * blockCntPerSegment * 80,
			InsertCapacity: blockRows * uint64(colCnt) * 2000,
			DataCapacity:   blockRows * uint64(colCnt) * 2000,
		}
		opts.MetaCleanerCfg = &e.MetaCleanerCfg{
			Interval: time.Duration(1) * time.Second,
		}
		opts.Meta.Conf = mdCfg
		return daoe.NewStorageWithOptions(path, opts)
	})
	defer c.Stop()
	time.Sleep(2 * time.Second)

	assert.NoError(t, err)
	stdLog.Printf("app all started.")

	//testKVStorage(t, c)
	testAOEStorage(t, c)
}

func testKVStorage(t *testing.T, c *testutil.TestCluster) {
	//Set Test
	resp, err := c.Applications[0].Exec(pb.Request{
		Type: pb.Set,
		Set: pb.SetRequest{
			Key:   []byte("Hello"),
			Value: []byte("World"),
		},
	})
	require.NoError(t, err)
	require.Equal(t, "OK", string(resp))

	//Get Test
	value, err := c.Applications[0].Exec(pb.Request{
		Type: pb.Get,
		Get: pb.GetRequest{
			Key : []byte("Hello"),
		},
	})
	require.NoError(t, err)
	require.Equal(t, value, []byte("World"))

	//Prefix Test
	for i:=uint64(0); i< 20; i++ {
		key := fmt.Sprintf("prefix-%d", i)
		_, err = c.Applications[0].Exec(pb.Request{
			Type: pb.Set,
			Set: pb.SetRequest{
				Key: []byte(key),
				Value: format.Uint64ToBytes(i),
			},
		})
		require.NoError(t, err)
	}

	keys, err := c.Applications[0].PrefixKeys([]byte("prefix-"), 0)
	require.NoError(t, err)
	require.Equal(t, 20, len(keys))


	kvs, err := c.Applications[0].PrefixScan([]byte("prefix-"), 0)
	require.NoError(t, err)
	require.Equal(t, 40, len(kvs))

	err = c.Applications[0].Delete([]byte("prefix-0"))
	require.NoError(t, err)
	keys, err = c.Applications[0].PrefixKeys([]byte("prefix-"), 0)
	require.NoError(t, err)
	require.Equal(t, 19, len(keys))
}

func testAOEStorage(t *testing.T, c *testutil.TestCluster)  {
	//CreateTableTest
	colCnt := 4
	tableInfo := md.MockTableInfo(colCnt)
	toShard := uint64(0)
	c.Applications[0].RaftStore().GetRouter().Every(uint64(pb.AOEGroup), false, func(shard *bhmetapb.Shard, address string){
		toShard = shard.ID
	})
	require.Less(t, uint64(0), toShard)
	err := c.Applications[0].CreateTablet(fmt.Sprintf("%d#%d", tableInfo.Id, toShard),toShard, tableInfo)
	require.NoError(t, err)

	names, err := c.Applications[0].TabletNames(toShard)

	require.NoError(t, err)
	require.Equal(t, 1, len(names))

	//AppendTest
	attrs := helper.Attribute(*tableInfo)
	var typs []types.Type
	for _, attr := range attrs {
		typs = append(typs, attr.Type)
	}
	ibat := chunk.MockBatch(typs, batchInsertRows)
	var buf bytes.Buffer
	err = protocol.EncodeBatch(ibat, &buf)
	require.NoError(t, err)
	ids, err := c.Applications[0].GetSegmentIds(fmt.Sprintf("%d#%d", tableInfo.Id, toShard), toShard)
	require.NoError(t, err)
	fmt.Printf("SegmentIds is %v", ids)
	err = c.Applications[0].Append(fmt.Sprintf("%d#%d", tableInfo.Id, toShard), toShard, buf.Bytes())
	require.NoError(t, err)
}