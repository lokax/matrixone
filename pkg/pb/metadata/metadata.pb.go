// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: metadata.proto

package metadata

import (
	fmt "fmt"
	io "io"
	math "math"
	math_bits "math/bits"

	_ "github.com/gogo/protobuf/gogoproto"
	proto "github.com/gogo/protobuf/proto"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

// DNShardRecord is DN shard metadata describing what is a DN shard. It
// is internally used by HAKeeper to maintain how many DNs available in
// the system.
type DNShardRecord struct {
	// ShardID the id of the DN shard.
	ShardID uint64 `protobuf:"varint,1,opt,name=ShardID,proto3" json:"ShardID,omitempty"`
	// LogShardID a DN corresponds to a unique Shard of LogService.
	LogShardID           uint64   `protobuf:"varint,2,opt,name=LogShardID,proto3" json:"LogShardID,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *DNShardRecord) Reset()         { *m = DNShardRecord{} }
func (m *DNShardRecord) String() string { return proto.CompactTextString(m) }
func (*DNShardRecord) ProtoMessage()    {}
func (*DNShardRecord) Descriptor() ([]byte, []int) {
	return fileDescriptor_56d9f74966f40d04, []int{0}
}
func (m *DNShardRecord) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *DNShardRecord) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_DNShardRecord.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *DNShardRecord) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DNShardRecord.Merge(m, src)
}
func (m *DNShardRecord) XXX_Size() int {
	return m.Size()
}
func (m *DNShardRecord) XXX_DiscardUnknown() {
	xxx_messageInfo_DNShardRecord.DiscardUnknown(m)
}

var xxx_messageInfo_DNShardRecord proto.InternalMessageInfo

func (m *DNShardRecord) GetShardID() uint64 {
	if m != nil {
		return m.ShardID
	}
	return 0
}

func (m *DNShardRecord) GetLogShardID() uint64 {
	if m != nil {
		return m.LogShardID
	}
	return 0
}

// DNShard
type DNShard struct {
	// DNShard extends DNShardRecord
	DNShardRecord `protobuf:"bytes,1,opt,name=DNShardRecord,proto3,embedded=DNShardRecord" json:"DNShardRecord"`
	// ReplicaID only one replica for one DN. The ReplicaID is specified at
	// the time the DN is created. DN restart ReplicaID will not change, when
	// a DN is migrated to another node, the ReplicaID will be reset.
	ReplicaID uint64 `protobuf:"varint,2,opt,name=ReplicaID,proto3" json:"ReplicaID,omitempty"`
	// Address is DN's external service address.
	Address              string   `protobuf:"bytes,3,opt,name=Address,proto3" json:"Address,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *DNShard) Reset()         { *m = DNShard{} }
func (m *DNShard) String() string { return proto.CompactTextString(m) }
func (*DNShard) ProtoMessage()    {}
func (*DNShard) Descriptor() ([]byte, []int) {
	return fileDescriptor_56d9f74966f40d04, []int{1}
}
func (m *DNShard) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *DNShard) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_DNShard.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *DNShard) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DNShard.Merge(m, src)
}
func (m *DNShard) XXX_Size() int {
	return m.Size()
}
func (m *DNShard) XXX_DiscardUnknown() {
	xxx_messageInfo_DNShard.DiscardUnknown(m)
}

var xxx_messageInfo_DNShard proto.InternalMessageInfo

func (m *DNShard) GetReplicaID() uint64 {
	if m != nil {
		return m.ReplicaID
	}
	return 0
}

func (m *DNShard) GetAddress() string {
	if m != nil {
		return m.Address
	}
	return ""
}

// LogShardRecord is Log Shard Metadata describing what is a Log shard. It is
// internally used by the HAKeeper to maintain how many Log shards are available
// in the system.
type LogShardRecord struct {
	// ShardID is the id of the Log Shard.
	ShardID uint64 `protobuf:"varint,1,opt,name=ShardID,proto3" json:"ShardID,omitempty"`
	// NumberOfReplicas is the number of replicas in the shard.
	NumberOfReplicas uint64 `protobuf:"varint,2,opt,name=NumberOfReplicas,proto3" json:"NumberOfReplicas,omitempty"`
	// Name is the human readable name of the shard given by the DN.
	Name                 string   `protobuf:"bytes,3,opt,name=Name,proto3" json:"Name,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *LogShardRecord) Reset()         { *m = LogShardRecord{} }
func (m *LogShardRecord) String() string { return proto.CompactTextString(m) }
func (*LogShardRecord) ProtoMessage()    {}
func (*LogShardRecord) Descriptor() ([]byte, []int) {
	return fileDescriptor_56d9f74966f40d04, []int{2}
}
func (m *LogShardRecord) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *LogShardRecord) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_LogShardRecord.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *LogShardRecord) XXX_Merge(src proto.Message) {
	xxx_messageInfo_LogShardRecord.Merge(m, src)
}
func (m *LogShardRecord) XXX_Size() int {
	return m.Size()
}
func (m *LogShardRecord) XXX_DiscardUnknown() {
	xxx_messageInfo_LogShardRecord.DiscardUnknown(m)
}

var xxx_messageInfo_LogShardRecord proto.InternalMessageInfo

func (m *LogShardRecord) GetShardID() uint64 {
	if m != nil {
		return m.ShardID
	}
	return 0
}

func (m *LogShardRecord) GetNumberOfReplicas() uint64 {
	if m != nil {
		return m.NumberOfReplicas
	}
	return 0
}

func (m *LogShardRecord) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

// DNStore DN store metadata
type DNStore struct {
	// UUID DNStore uuid id
	UUID string `protobuf:"bytes,1,opt,name=UUID,proto3" json:"UUID,omitempty"`
	// Shards DNShards
	Shards               []DNShard `protobuf:"bytes,2,rep,name=Shards,proto3" json:"Shards"`
	XXX_NoUnkeyedLiteral struct{}  `json:"-"`
	XXX_unrecognized     []byte    `json:"-"`
	XXX_sizecache        int32     `json:"-"`
}

func (m *DNStore) Reset()         { *m = DNStore{} }
func (m *DNStore) String() string { return proto.CompactTextString(m) }
func (*DNStore) ProtoMessage()    {}
func (*DNStore) Descriptor() ([]byte, []int) {
	return fileDescriptor_56d9f74966f40d04, []int{3}
}
func (m *DNStore) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *DNStore) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_DNStore.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *DNStore) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DNStore.Merge(m, src)
}
func (m *DNStore) XXX_Size() int {
	return m.Size()
}
func (m *DNStore) XXX_DiscardUnknown() {
	xxx_messageInfo_DNStore.DiscardUnknown(m)
}

var xxx_messageInfo_DNStore proto.InternalMessageInfo

func (m *DNStore) GetUUID() string {
	if m != nil {
		return m.UUID
	}
	return ""
}

func (m *DNStore) GetShards() []DNShard {
	if m != nil {
		return m.Shards
	}
	return nil
}

func init() {
	proto.RegisterType((*DNShardRecord)(nil), "metadata.DNShardRecord")
	proto.RegisterType((*DNShard)(nil), "metadata.DNShard")
	proto.RegisterType((*LogShardRecord)(nil), "metadata.LogShardRecord")
	proto.RegisterType((*DNStore)(nil), "metadata.DNStore")
}

func init() { proto.RegisterFile("metadata.proto", fileDescriptor_56d9f74966f40d04) }

var fileDescriptor_56d9f74966f40d04 = []byte{
	// 317 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x84, 0x51, 0xcd, 0x4a, 0x03, 0x31,
	0x10, 0x36, 0xb6, 0xf4, 0x27, 0xc5, 0xa2, 0xb9, 0x58, 0x44, 0xb6, 0x65, 0x4f, 0x45, 0xb0, 0xc1,
	0xfa, 0x00, 0x62, 0x29, 0x48, 0x41, 0x56, 0x88, 0xf4, 0xe2, 0x2d, 0xdb, 0x4d, 0xd3, 0x55, 0xb7,
	0x59, 0xb2, 0x59, 0xf0, 0x19, 0x7c, 0xb2, 0x1e, 0xfb, 0x04, 0x45, 0xf6, 0x49, 0x64, 0xc7, 0xa4,
	0xd6, 0xf6, 0xe0, 0x6d, 0xbe, 0xf9, 0x26, 0xdf, 0xf7, 0x4d, 0x06, 0xb7, 0x13, 0x61, 0x78, 0xc4,
	0x0d, 0x1f, 0xa4, 0x5a, 0x19, 0x45, 0x1a, 0x0e, 0x5f, 0x5c, 0xcb, 0xd8, 0x2c, 0xf2, 0x70, 0x30,
	0x53, 0x09, 0x95, 0x4a, 0x2a, 0x0a, 0x03, 0x61, 0x3e, 0x07, 0x04, 0x00, 0xaa, 0x9f, 0x87, 0xfe,
	0x04, 0x9f, 0x8c, 0x83, 0xe7, 0x05, 0xd7, 0x11, 0x13, 0x33, 0xa5, 0x23, 0xd2, 0xc1, 0x75, 0x80,
	0x93, 0x71, 0x07, 0xf5, 0x50, 0xbf, 0xca, 0x1c, 0x24, 0x1e, 0xc6, 0x8f, 0x4a, 0x3a, 0xf2, 0x18,
	0xc8, 0x9d, 0x8e, 0xff, 0x89, 0x70, 0xdd, 0x6a, 0x91, 0x87, 0x3d, 0x59, 0xd0, 0x6a, 0x0d, 0xcf,
	0x07, 0xdb, 0xdc, 0x7f, 0xe8, 0x51, 0x63, 0xb5, 0xe9, 0x1e, 0xad, 0x37, 0x5d, 0xc4, 0xf6, 0xe2,
	0x5c, 0xe2, 0x26, 0x13, 0xe9, 0x7b, 0x3c, 0xe3, 0x5b, 0xcf, 0xdf, 0x46, 0x19, 0xf6, 0x3e, 0x8a,
	0xb4, 0xc8, 0xb2, 0x4e, 0xa5, 0x87, 0xfa, 0x4d, 0xe6, 0xa0, 0xff, 0x8a, 0xdb, 0x2e, 0xda, 0xbf,
	0x8b, 0x5d, 0xe1, 0xd3, 0x20, 0x4f, 0x42, 0xa1, 0x9f, 0xe6, 0x56, 0x3a, 0xb3, 0x56, 0x07, 0x7d,
	0x42, 0x70, 0x35, 0xe0, 0x89, 0xb0, 0x76, 0x50, 0xfb, 0x01, 0xec, 0x6d, 0x94, 0x16, 0x25, 0x3d,
	0x9d, 0x5a, 0x87, 0x26, 0x83, 0x9a, 0x50, 0x5c, 0x03, 0xa7, 0x52, 0xb4, 0xd2, 0x6f, 0x0d, 0xcf,
	0x0e, 0x3e, 0x61, 0x54, 0x2d, 0xd7, 0x67, 0x76, 0x6c, 0x74, 0xb7, 0x2a, 0x3c, 0xb4, 0x2e, 0x3c,
	0xf4, 0x55, 0x78, 0xe8, 0xe5, 0x66, 0xe7, 0xa0, 0x09, 0x37, 0x3a, 0xfe, 0x50, 0x3a, 0x96, 0xf1,
	0xd2, 0x81, 0xa5, 0xa0, 0xe9, 0x9b, 0xa4, 0x69, 0x48, 0x9d, 0x6c, 0x58, 0x83, 0xdb, 0xde, 0x7e,
	0x07, 0x00, 0x00, 0xff, 0xff, 0x5b, 0xd0, 0xb2, 0x72, 0x26, 0x02, 0x00, 0x00,
}

func (m *DNShardRecord) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *DNShardRecord) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *DNShardRecord) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		i -= len(m.XXX_unrecognized)
		copy(dAtA[i:], m.XXX_unrecognized)
	}
	if m.LogShardID != 0 {
		i = encodeVarintMetadata(dAtA, i, uint64(m.LogShardID))
		i--
		dAtA[i] = 0x10
	}
	if m.ShardID != 0 {
		i = encodeVarintMetadata(dAtA, i, uint64(m.ShardID))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func (m *DNShard) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *DNShard) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *DNShard) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		i -= len(m.XXX_unrecognized)
		copy(dAtA[i:], m.XXX_unrecognized)
	}
	if len(m.Address) > 0 {
		i -= len(m.Address)
		copy(dAtA[i:], m.Address)
		i = encodeVarintMetadata(dAtA, i, uint64(len(m.Address)))
		i--
		dAtA[i] = 0x1a
	}
	if m.ReplicaID != 0 {
		i = encodeVarintMetadata(dAtA, i, uint64(m.ReplicaID))
		i--
		dAtA[i] = 0x10
	}
	{
		size, err := m.DNShardRecord.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
		i = encodeVarintMetadata(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0xa
	return len(dAtA) - i, nil
}

func (m *LogShardRecord) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *LogShardRecord) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *LogShardRecord) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		i -= len(m.XXX_unrecognized)
		copy(dAtA[i:], m.XXX_unrecognized)
	}
	if len(m.Name) > 0 {
		i -= len(m.Name)
		copy(dAtA[i:], m.Name)
		i = encodeVarintMetadata(dAtA, i, uint64(len(m.Name)))
		i--
		dAtA[i] = 0x1a
	}
	if m.NumberOfReplicas != 0 {
		i = encodeVarintMetadata(dAtA, i, uint64(m.NumberOfReplicas))
		i--
		dAtA[i] = 0x10
	}
	if m.ShardID != 0 {
		i = encodeVarintMetadata(dAtA, i, uint64(m.ShardID))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func (m *DNStore) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *DNStore) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *DNStore) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		i -= len(m.XXX_unrecognized)
		copy(dAtA[i:], m.XXX_unrecognized)
	}
	if len(m.Shards) > 0 {
		for iNdEx := len(m.Shards) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.Shards[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintMetadata(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x12
		}
	}
	if len(m.UUID) > 0 {
		i -= len(m.UUID)
		copy(dAtA[i:], m.UUID)
		i = encodeVarintMetadata(dAtA, i, uint64(len(m.UUID)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintMetadata(dAtA []byte, offset int, v uint64) int {
	offset -= sovMetadata(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *DNShardRecord) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.ShardID != 0 {
		n += 1 + sovMetadata(uint64(m.ShardID))
	}
	if m.LogShardID != 0 {
		n += 1 + sovMetadata(uint64(m.LogShardID))
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func (m *DNShard) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = m.DNShardRecord.Size()
	n += 1 + l + sovMetadata(uint64(l))
	if m.ReplicaID != 0 {
		n += 1 + sovMetadata(uint64(m.ReplicaID))
	}
	l = len(m.Address)
	if l > 0 {
		n += 1 + l + sovMetadata(uint64(l))
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func (m *LogShardRecord) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.ShardID != 0 {
		n += 1 + sovMetadata(uint64(m.ShardID))
	}
	if m.NumberOfReplicas != 0 {
		n += 1 + sovMetadata(uint64(m.NumberOfReplicas))
	}
	l = len(m.Name)
	if l > 0 {
		n += 1 + l + sovMetadata(uint64(l))
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func (m *DNStore) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.UUID)
	if l > 0 {
		n += 1 + l + sovMetadata(uint64(l))
	}
	if len(m.Shards) > 0 {
		for _, e := range m.Shards {
			l = e.Size()
			n += 1 + l + sovMetadata(uint64(l))
		}
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func sovMetadata(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozMetadata(x uint64) (n int) {
	return sovMetadata(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *DNShardRecord) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowMetadata
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: DNShardRecord: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: DNShardRecord: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field ShardID", wireType)
			}
			m.ShardID = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMetadata
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.ShardID |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field LogShardID", wireType)
			}
			m.LogShardID = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMetadata
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.LogShardID |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipMetadata(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthMetadata
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, dAtA[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *DNShard) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowMetadata
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: DNShard: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: DNShard: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field DNShardRecord", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMetadata
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthMetadata
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthMetadata
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.DNShardRecord.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field ReplicaID", wireType)
			}
			m.ReplicaID = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMetadata
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.ReplicaID |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Address", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMetadata
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthMetadata
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthMetadata
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Address = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipMetadata(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthMetadata
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, dAtA[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *LogShardRecord) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowMetadata
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: LogShardRecord: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: LogShardRecord: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field ShardID", wireType)
			}
			m.ShardID = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMetadata
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.ShardID |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field NumberOfReplicas", wireType)
			}
			m.NumberOfReplicas = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMetadata
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.NumberOfReplicas |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Name", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMetadata
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthMetadata
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthMetadata
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Name = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipMetadata(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthMetadata
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, dAtA[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *DNStore) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowMetadata
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: DNStore: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: DNStore: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field UUID", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMetadata
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthMetadata
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthMetadata
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.UUID = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Shards", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMetadata
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthMetadata
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthMetadata
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Shards = append(m.Shards, DNShard{})
			if err := m.Shards[len(m.Shards)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipMetadata(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthMetadata
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, dAtA[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipMetadata(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowMetadata
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowMetadata
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
		case 1:
			iNdEx += 8
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowMetadata
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, ErrInvalidLengthMetadata
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupMetadata
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthMetadata
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthMetadata        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowMetadata          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupMetadata = fmt.Errorf("proto: unexpected end of group")
)
