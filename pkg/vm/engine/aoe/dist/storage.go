package dist

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/fagongzi/util/format"
	"github.com/matrixorigin/matrixcube/aware"
	"github.com/matrixorigin/matrixcube/components/prophet/metadata"
	"github.com/matrixorigin/matrixcube/components/prophet/pb/metapb"
	"github.com/matrixorigin/matrixcube/config"
	"github.com/matrixorigin/matrixcube/pb/bhmetapb"
	"github.com/matrixorigin/matrixcube/pb/raftcmdpb"
	"github.com/matrixorigin/matrixcube/proxy"
	"github.com/matrixorigin/matrixcube/raftstore"
	"github.com/matrixorigin/matrixcube/server"
	cstorage "github.com/matrixorigin/matrixcube/storage"
	"matrixone/pkg/vm/engine/aoe"
	"matrixone/pkg/vm/engine/aoe/common/helper"
	"matrixone/pkg/vm/engine/aoe/dist/pb"
	adb "matrixone/pkg/vm/engine/aoe/storage/db"
	"matrixone/pkg/vm/engine/aoe/storage/dbi"
	"matrixone/pkg/vm/engine/aoe/storage/layout/table/v2/handle"
	"sync"
	"time"
)

const (
	defaultRPCTimeout = time.Second * 3
)

// Storage storage
type Storage interface {
	// Start the storage
	Start() error
	// Close close the storage
	Close()

	// Set set key value
	Set([]byte, []byte) error
	// SetWithGroup set key value
	SetWithGroup([]byte, []byte, pb.Group) error

	// Get returns the value of key
	Get([]byte) ([]byte, error)
	// GetWithGroup returns the value of key
	GetWithGroup([]byte, pb.Group) ([]byte, error)
	// Delete remove the key from the store
	Delete([]byte) error
	// Scan scan [start,end) data
	Scan([]byte, []byte, uint64) ([][]byte, error)
	// ScanWithGroup Scan scan [start,end) data
	ScanWithGroup([]byte, []byte, uint64, pb.Group) ([][]byte, error)
	// PrefixScan scan k-vs which k starts with prefix
	PrefixScan([]byte, uint64) ([][]byte, error)
	// PrefixScanWithGroup scan k-vs which k starts with prefix
	PrefixScanWithGroup([]byte, uint64, pb.Group) ([][]byte, error)
	PrefixKeys([]byte, uint64) ([][]byte, error)
	PrefixKeysWithGroup([]byte, uint64, pb.Group) ([][]byte, error)
	AllocID([]byte) (uint64, error)

	Append(string, uint64, []byte) error
	GetSnapshot(dbi.GetSnapshotCtx) (*handle.Snapshot, error)
	GetSegmentIds(string, uint64) (adb.IDS, error)
	CreateTablet(name string, shardId uint64, tbl *aoe.TableInfo) error
	DropTablet(string) (uint64, error)
	TabletIDs()([]uint64, error)
	TabletNames(uint64) ([]string, error)
	// Exec exec command
	Exec(cmd interface{}) ([]byte, error)
	// AsyncExec async exec command
	AsyncExec(interface{}, func(interface{}, []byte, error), interface{})
	// ExecWithGroup exec command with group
	ExecWithGroup(interface{}, pb.Group) ([]byte, error)
	// AsyncExecWithGroup async exec command with group
	AsyncExecWithGroup(interface{}, pb.Group, func(interface{}, []byte, error), interface{})
	// RaftStore returns the raft store
	RaftStore() raftstore.Store
}

type aoeStorage struct {
	app   *server.Application
	store raftstore.Store
	locks sync.Map // key -> lock
	cmds  map[uint64]raftcmdpb.CMDType
}

func (h *aoeStorage) Start() error {
	return h.app.Start()
}

func (h *aoeStorage) Close() {
	h.app.Stop()
}

// NewStorage returns a beehive request handler
func NewStorage(
	metadataStorage cstorage.MetadataStorage,
	kvDataStorage cstorage.DataStorage,
	aoeDataStorage cstorage.DataStorage) (Storage, error) {
	return NewStorageWithOptions(metadataStorage, kvDataStorage, aoeDataStorage, nil, server.Cfg{})
}

// NewStorageWithOptions returns a beehive request handler
func NewStorageWithOptions(
	metaStorage cstorage.MetadataStorage,
	kvDataStorage cstorage.DataStorage,
	aoeDataStorage cstorage.DataStorage,
	adjustFunc func(cfg *config.Config),
	scfg server.Cfg) (Storage, error) {

	h := &aoeStorage{
		cmds: make(map[uint64]raftcmdpb.CMDType),
	}

	cfg := &config.Config{}
	cfg.Customize.CustomSplitCompletedFuncFactory = func(group uint64) func(old *bhmetapb.Shard, news []bhmetapb.Shard) {
		switch group {
		case uint64(pb.AOEGroup):
			return func(old *bhmetapb.Shard, news []bhmetapb.Shard) {
				//panic("not impl")
			}
		default:
			return func(old *bhmetapb.Shard, news []bhmetapb.Shard) {

			}
		}
	}
	cfg.Storage.MetaStorage = metaStorage
	cfg.Storage.DataStorageFactory = func(group, shardID uint64) cstorage.DataStorage {
		switch group {
		case uint64(pb.KVGroup):
			return kvDataStorage
		case uint64(pb.AOEGroup):
			return aoeDataStorage
		}
		return nil
	}
	cfg.Storage.ForeachDataStorageFunc = func(cb func(cstorage.DataStorage)) {
		cb(kvDataStorage)
		cb(aoeDataStorage)
	}
	cfg.Prophet.Replication.Groups = []uint64{uint64(pb.KVGroup), uint64(pb.AOEGroup)}
	cfg.ShardGroups = 2
	cfg.Customize.CustomInitShardsFactory = func() []bhmetapb.Shard {
		return []bhmetapb.Shard{
			{
				Group: uint64(pb.KVGroup),
			},
			{
				Group: uint64(pb.AOEGroup),
				Start: []byte("0"),
				End:   []byte("1"),
			},
		}
	}

	cfg.Prophet.ResourceStateChangedHandler = func(res metadata.Resource, from metapb.ResourceState, to metapb.ResourceState) {
		if from == metapb.ResourceState_WaittingCreate && to == metapb.ResourceState_Running {
			if res.Data() == nil {
				return
			}
			//Key Job名称，唯一
			//Data Job内容
			//JobHandler处理逻辑
			//Remove Job
			//逻辑需要保证幂等
			//Job只会在etcd的leader执行
			data := format.Uint64ToBytes(res.ID())
			data = append(data, res.Data()...)
			if err := h.RaftStore().Prophet().GetStorage().PutJob([]byte("test"), data); err != nil {
				//TODO: how to handle local create table failing
			}else {
				//h.RaftStore().Prophet().GetStorage().RemoveJob([]byte("test"))
			}

		}
	}

	cfg.Prophet.JobCheckerDuration = 1000 * time.Millisecond
	cfg.Prophet.JobHandler = func(key, data []byte) {
		if bytes.Equal(key, []byte("test")) {
			toShard := format.MustBytesToUint64(data[0:8])
			header := format.MustBytesToUint64(data[8:16])
			keys := bytes.Split(data[16:16+header], []byte("#"))
			tKey := keys[0]
			rKey := []byte(fmt.Sprintf("%s%d", string(keys[1]), toShard))
			t, _ := helper.DecodeTable(data[16+header:])
			// TODO: Re-design group store and set value to <partition, segment_ids>
			if err := h.CreateTablet(fmt.Sprintf("%d#%d", t.Id, toShard), toShard, &t); err != nil {
				//TODO: how to handle local create table failing
			}else {
				t.State = aoe.StatePublic
				meta, _ := helper.EncodeTable(t)
				_ = h.Set(tKey, meta)
				_ = h.Set(rKey, []byte(t.Name))
			}
		}
	}

	cfg.Customize.CustomShardStateAwareFactory = func() aware.ShardStateAware {
		return h
	}

	cfg.Customize.CustomAdjustCompactFuncFactory = func(group uint64) func(shard bhmetapb.Shard, compactIndex uint64) (newCompactIdx uint64, err error) {
		//TODO: 询问所有tablet
		return func(shard bhmetapb.Shard, compactIndex uint64) (newCompactIdx uint64, err error) {
			return newCompactIdx, err
		}
	}

	cfg.Customize.CustomAdjustInitAppliedIndexFactory = func(group uint64) func(shard bhmetapb.Shard, initAppliedIndex uint64) (adjustAppliedIndex uint64) {
		//TODO:aoe group only
		return func(shard bhmetapb.Shard, initAppliedIndex uint64) (adjustAppliedIndex uint64) {
			return adjustAppliedIndex
		}
	}

	if adjustFunc != nil {
		adjustFunc(cfg)
	}

	h.store = raftstore.NewStore(cfg)
	scfg.Store = h.store
	scfg.Handler = h
	h.app = server.NewApplicationWithDispatcher(scfg, func(req *raftcmdpb.Request, cmd interface{}, proxy proxy.ShardsProxy) error {
		if req.Group == uint64(pb.KVGroup) {
			return proxy.Dispatch(req)
		}
		args := cmd.(pb.Request)
		if args.Shard == 0 {
			return proxy.Dispatch(req)
		}
		req.ToShard = args.Shard
		return proxy.DispatchTo(req, args.Shard, h.store.GetRouter().LeaderAddress(req.ToShard))
	})
	h.init()
	if err := h.app.Start(); err != nil {
		return nil, err
	}
	return h, nil
}

func (h *aoeStorage) Set(key, value []byte) error {
	return h.SetWithGroup(key, value, pb.KVGroup)
}

func (h *aoeStorage) SetWithGroup(key, value []byte, group pb.Group) error {
	req := pb.Request{
		Type: pb.Set,
		Group: group,
		Set: pb.SetRequest{
			Key: key,
			Value: value,
		},
	}
	_, err := h.ExecWithGroup(req, group)
	return err
}

func (h *aoeStorage) Get(key []byte) ([]byte, error) {
	return h.GetWithGroup(key, pb.KVGroup)
}

// GetWithGroup returns the value of key
func (h *aoeStorage) GetWithGroup(key []byte, group pb.Group) ([]byte, error) {
	req := pb.Request{
		Type: pb.Get,
		Group: group,
		Get: pb.GetRequest{
			Key: key,
		},
	}
	value, err := h.ExecWithGroup(req, group)
	return value, err
}

func (h *aoeStorage) Delete(key []byte) error {
	req := pb.Request{
		Type: pb.Del,
		Group: pb.KVGroup,
		Delete: pb.DeleteRequest{
			Key: key,
		},
	}
	_, err := h.ExecWithGroup(req, pb.KVGroup)
	return err
}

func (h *aoeStorage) Scan(start []byte, end []byte, limit uint64) ([][]byte, error) {
	return h.ScanWithGroup(start, end, limit, pb.KVGroup)
}

func (h *aoeStorage) ScanWithGroup(start []byte, end []byte, limit uint64, group pb.Group) ([][]byte, error) {
	req := pb.Request{
		Type: pb.Scan,
		Group: group,
		Scan: pb.ScanRequest{
			Start: start,
			End: end,
			Limit: limit,
		},
	}
	data, err := h.ExecWithGroup(req, group)
	if err != nil {
		return nil, err
	}
	var pairs [][]byte
	err = json.Unmarshal(data, &pairs)
	if err != nil {
		return nil, err
	}
	return pairs, nil
}

func (h *aoeStorage) PrefixScan(prefix []byte, limit uint64) ([][]byte, error) {
	return h.PrefixScanWithGroup(prefix, limit, pb.KVGroup)
}

func (h *aoeStorage) PrefixScanWithGroup(prefix []byte, limit uint64, group pb.Group) ([][]byte, error) {
	req := pb.Request{
		Type: pb.PrefixScan,
		Group: group,
		PrefixScan: pb.PrefixScanRequest{
			Prefix: prefix,
			StartKey: prefix,
			Limit: limit,
		},
	}
	var pairs [][]byte
	var err error
	var data []byte
	i := 0
	for {
		i = i + 1
		data, err = h.ExecWithGroup(req, group)
		if data == nil || err != nil {
			break
		}
		var kvs [][]byte
		err = json.Unmarshal(data, &kvs)
		if err != nil || kvs == nil || len(kvs) == 0 {
			break
		}
		if len(kvs)%2 == 0 {
			pairs = append(pairs, kvs...)
			break
		}

		pairs = append(pairs, kvs[0:len(kvs)-1]...)
		req.PrefixScan.StartKey = raftstore.EncodeDataKey(uint64(group), kvs[len(kvs)-1])
	}
	return pairs, err
}

func (h *aoeStorage) PrefixKeys(prefix []byte, limit uint64) ([][]byte, error) {
	return h.PrefixKeysWithGroup(prefix, limit, pb.KVGroup)
}

func (h *aoeStorage) PrefixKeysWithGroup(prefix []byte, limit uint64, group pb.Group) ([][]byte, error) {
	req := pb.Request{
		Type: pb.PrefixScan,
		Group: group,
		PrefixScan: pb.PrefixScanRequest{
			Prefix: prefix,
			StartKey: prefix,
			Limit: limit,
		},
	}
	var values [][]byte
	var err error
	var data []byte
	i := 0
	for {
		i = i + 1
		data, err = h.ExecWithGroup(req, group)
		if data == nil || err != nil {
			break
		}
		var kvs [][]byte
		err = json.Unmarshal(data, &kvs)
		if err != nil || kvs == nil || len(kvs) == 0 {
			break
		}

		for i:=0; i<len(kvs)-1; i+=2 {
		 values = append(values, kvs[i])
		}

		if len(kvs)%2 == 0 {
			break
		}

		req.PrefixScan.StartKey = raftstore.EncodeDataKey(uint64(group), kvs[len(kvs)-1])
	}
	return values, err
}

func (h *aoeStorage) AllocID(idkey []byte) (uint64, error) {
	req := pb.Request{
		Type: pb.Incr,
		Group: pb.KVGroup,
		AllocID: pb.AllocIDRequest{
			Key: idkey,
		},
	}
	data, err := h.ExecWithGroup(req, pb.KVGroup)
	if err != nil {
		return 0, err
	}
	resp := format.MustBytesToUint64(data)
	return resp, nil
}


func (h *aoeStorage) Append(name string, shardId uint64, data []byte) error {
	req := pb.Request{
		Type: pb.Append,
		Group: pb.AOEGroup,
		Shard: shardId,
		Append: pb.AppendRequest{
			Data: data,
			TabletName: name,
		},
	}
	data, err := h.ExecWithGroup(req, pb.AOEGroup)
	return err
}

func (h *aoeStorage) GetSnapshot(ctx dbi.GetSnapshotCtx) (*handle.Snapshot, error) {
	ctxStr, err := json.Marshal(ctx)
	req := pb.Request{
		Type: pb.GetSnapshot,
		Group: pb.AOEGroup,
		GetSnapshot: pb.GetSnapshotRequest{
			Ctx: ctxStr,
		},
	}
	value, err := h.ExecWithGroup(req, pb.AOEGroup)
	if err != nil {
		return nil, err
	}
	var s handle.Snapshot
	err = json.Unmarshal(value, &s)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (h *aoeStorage) GetSegmentIds(tabletName string, toShard uint64) (ids adb.IDS, err error) {
	req := pb.Request{
		Type: pb.GetSegmentIds,
		Group: pb.AOEGroup,
		Shard: toShard,
		GetSegmentIds: pb.GetSegmentIdsRequest{
			Name: tabletName,
		},
	}
	value, err := h.ExecWithGroup(req, pb.AOEGroup)
	if err != nil {
		return ids, err
	}
	err = json.Unmarshal(value, &ids)
	if err != nil {
		return ids, err
	}
	return ids, nil
}

func (h *aoeStorage) CreateTablet(name string, shardId uint64, tbl *aoe.TableInfo) (err error) {
	info, _ := helper.EncodeTable(*tbl)
	req := pb.Request{
		Shard: shardId,
		Group: pb.AOEGroup,
		Type: pb.CreateTablet,
		CreateTablet: pb.CreateTabletRequest{
			Name: name,
			TableInfo: info,
		},
	}
	_, err = h.ExecWithGroup(req, pb.AOEGroup)
	if err != nil {
		return err
	}
	return nil
}

func (h *aoeStorage) DropTablet(name string) (id uint64, err error){
	req := pb.Request{
		Type: pb.DropTablet,
		Group: pb.AOEGroup,
		DropTablet: pb.DropTabletRequest{
			Name: name,
		},
	}
	value, err := h.ExecWithGroup(req, pb.AOEGroup)
	if err != nil {
		return id, err
	}
	return format.MustBytesToUint64(value), nil
}

func (h *aoeStorage) TabletIDs() ([]uint64, error) {
	req := pb.Request{
		Type: pb.TabletIds,
		Group: pb.AOEGroup,
		TabletIds: pb.TabletIDsRequest{
		},
	}
	value, err := h.ExecWithGroup(req, pb.AOEGroup)
	if err != nil {
		return nil, err
	}
	var rsp []uint64
	err = json.Unmarshal(value, &rsp)
	if err != nil {
		return nil, err
	}
	return rsp, nil
}

func (h *aoeStorage) TabletNames(toShard uint64) ([]string, error) {
	req := pb.Request{
		Shard: toShard,
		Group: pb.AOEGroup,
		Type: pb.TabletNames,
		TabletIds: pb.TabletIDsRequest{
		},
	}
	value, err := h.ExecWithGroup(req, pb.AOEGroup)
	if err != nil {
		return nil, err
	}
	var rsp []string
	err = json.Unmarshal(value, &rsp)
	if err != nil {
		return nil, err
	}
	return rsp, nil
}

func (h *aoeStorage) Exec(cmd interface{}) ([]byte, error) {
	return h.app.Exec(cmd, defaultRPCTimeout)
}

func (h *aoeStorage) AsyncExec(cmd interface{}, cb func(interface{}, []byte, error), arg interface{}) {
	h.app.AsyncExecWithTimeout(cmd, cb, defaultRPCTimeout, arg)
}

func (h *aoeStorage) AsyncExecWithGroup(cmd interface{}, group pb.Group, cb func(interface{}, []byte, error), arg interface{}) {
	h.app.AsyncExecWithGroupAndTimeout(cmd, uint64(group), cb, defaultRPCTimeout, arg)
}

func (h *aoeStorage) ExecWithGroup(cmd interface{}, group pb.Group) ([]byte, error) {
	return h.app.ExecWithGroup(cmd, uint64(group), defaultRPCTimeout)
}

func (h *aoeStorage) RaftStore() raftstore.Store {
	return h.store
}

func (h *aoeStorage) getStoreByGroup(group uint64, shard uint64) cstorage.DataStorage {
	return h.store.DataStorageByGroup(group, shard)
}