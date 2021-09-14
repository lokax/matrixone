// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: meta.proto

package pb

import (
	fmt "fmt"
	_ "github.com/gogo/protobuf/gogoproto"
	proto "github.com/gogo/protobuf/proto"
	math "math"
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

// Group shard group
type Group int32

const (
	KVGroup  Group = 0
	AOEGroup Group = 1
)

var Group_name = map[int32]string{
	0: "KVGroup",
	1: "AOEGroup",
}

var Group_value = map[string]int32{
	"KVGroup":  0,
	"AOEGroup": 1,
}

func (x Group) String() string {
	return proto.EnumName(Group_name, int32(x))
}

func (Group) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_3b5ea8fe65782bcc, []int{0}
}

func init() {
	proto.RegisterEnum("pb.Group", Group_name, Group_value)
}

func init() { proto.RegisterFile("meta.proto", fileDescriptor_3b5ea8fe65782bcc) }

var fileDescriptor_3b5ea8fe65782bcc = []byte{
	// 110 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0xe2, 0xca, 0x4d, 0x2d, 0x49,
	0xd4, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0x62, 0x2a, 0x48, 0x92, 0xd2, 0x4d, 0xcf, 0x2c, 0xc9,
	0x28, 0x4d, 0xd2, 0x4b, 0xce, 0xcf, 0xd5, 0x4f, 0xcf, 0x4f, 0xcf, 0xd7, 0x07, 0x4b, 0x25, 0x95,
	0xa6, 0x81, 0x79, 0x60, 0x0e, 0x98, 0x05, 0xd1, 0xa2, 0xa5, 0xc4, 0xc5, 0xea, 0x5e, 0x94, 0x5f,
	0x5a, 0x20, 0xc4, 0xcd, 0xc5, 0xee, 0x1d, 0x06, 0x66, 0x0a, 0x30, 0x08, 0xf1, 0x70, 0x71, 0x38,
	0xfa, 0xbb, 0x42, 0x78, 0x8c, 0x4e, 0x2c, 0x17, 0x1e, 0xca, 0x31, 0x24, 0xb1, 0x81, 0x35, 0x18,
	0x03, 0x02, 0x00, 0x00, 0xff, 0xff, 0xde, 0x53, 0x0d, 0xca, 0x71, 0x00, 0x00, 0x00,
}