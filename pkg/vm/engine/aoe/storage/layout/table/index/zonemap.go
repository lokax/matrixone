package index

import (
	"bytes"
	"matrixone/pkg/container/types"
	"matrixone/pkg/encoding"
	"matrixone/pkg/vm/engine/aoe/storage/layout/base"
)

// TODO: Just for temp test
type ZoneMapIndex struct {
	T    types.Type
	MinV interface{}
	MaxV interface{}
	Col  int16
}

func NewZoneMap(t types.Type, minv, maxv interface{}, colIdx int16) Index {
	return &ZoneMapIndex{
		T:    t,
		MinV: minv,
		MaxV: maxv,
		Col:  colIdx,
	}
}

func (i *ZoneMapIndex) Unmarshall(data []byte) error {
	buf := data
	i.Col = encoding.DecodeInt16(buf[:2])
	buf = buf[2:]
	i.T = encoding.DecodeType(buf[:encoding.TypeSize])
	buf = buf[encoding.TypeSize:]
	switch i.T.Oid {
	case types.T_int8:
		i.MinV = encoding.DecodeInt8(buf[:1])
		buf = buf[1:]
		i.MaxV = encoding.DecodeInt8(buf[:1])
		return nil
	case types.T_int16:
		i.MinV = encoding.DecodeInt16(buf[:2])
		buf = buf[2:]
		i.MaxV = encoding.DecodeInt16(buf[:2])
		return nil
	case types.T_int32:
		i.MinV = encoding.DecodeInt32(buf[:4])
		buf = buf[4:]
		i.MaxV = encoding.DecodeInt32(buf[:4])
		return nil
	case types.T_int64:
		i.MinV = encoding.DecodeInt64(buf[:8])
		buf = buf[8:]
		i.MaxV = encoding.DecodeInt64(buf[:8])
		return nil
	case types.T_uint8:
		i.MinV = encoding.DecodeUint8(buf[:1])
		buf = buf[1:]
		i.MaxV = encoding.DecodeUint8(buf[:1])
		return nil
	case types.T_uint16:
		i.MinV = encoding.DecodeUint16(buf[:2])
		buf = buf[2:]
		i.MaxV = encoding.DecodeUint16(buf[:2])
		return nil
	case types.T_uint32:
		i.MinV = encoding.DecodeUint32(buf[:4])
		buf = buf[4:]
		i.MaxV = encoding.DecodeUint32(buf[:4])
		return nil
	case types.T_uint64:
		i.MinV = encoding.DecodeUint32(buf[:8])
		buf = buf[8:]
		i.MaxV = encoding.DecodeUint32(buf[:8])
		return nil
	case types.T_float32:
		i.MinV = encoding.DecodeFloat32(buf[:4])
		buf = buf[4:]
		i.MaxV = encoding.DecodeFloat32(buf[:4])
		return nil
	case types.T_float64:
		i.MinV = encoding.DecodeFloat64(buf[:8])
		buf = buf[4:]
		i.MaxV = encoding.DecodeFloat64(buf[:8])
		return nil
	case types.T_char, types.T_varchar, types.T_json:
		lenminv := encoding.DecodeInt16(buf[:2])
		buf = buf[2:]
		minBuf := make([]byte, int(lenminv))
		copy(minBuf, buf[:int(lenminv)])
		buf = buf[int(lenminv):]

		lenmaxv := encoding.DecodeInt16(buf[:2])
		buf = buf[2:]
		maxBuf := make([]byte, int(lenmaxv))
		copy(maxBuf, buf[:int(lenmaxv)])
		i.MinV = minBuf
		i.MaxV = maxBuf
		return nil
	}
	panic("unsupported")
}

func (i *ZoneMapIndex) Marshall() ([]byte, error) {
	var buf bytes.Buffer
	buf.Write(encoding.EncodeInt16(i.Col))
	switch i.T.Oid {
	case types.T_int8:
		buf.Write(encoding.EncodeType(i.T))
		buf.Write(encoding.EncodeInt8(i.MinV.(int8)))
		buf.Write(encoding.EncodeInt8(i.MaxV.(int8)))
		return buf.Bytes(), nil
	case types.T_int16:
		buf.Write(encoding.EncodeType(i.T))
		buf.Write(encoding.EncodeInt16(i.MinV.(int16)))
		buf.Write(encoding.EncodeInt16(i.MaxV.(int16)))
		return buf.Bytes(), nil
	case types.T_int32:
		buf.Write(encoding.EncodeType(i.T))
		buf.Write(encoding.EncodeInt32(i.MinV.(int32)))
		buf.Write(encoding.EncodeInt32(i.MaxV.(int32)))
		return buf.Bytes(), nil
	case types.T_int64:
		buf.Write(encoding.EncodeType(i.T))
		buf.Write(encoding.EncodeInt64(i.MinV.(int64)))
		buf.Write(encoding.EncodeInt64(i.MaxV.(int64)))
		return buf.Bytes(), nil
	case types.T_uint8:
		buf.Write(encoding.EncodeType(i.T))
		buf.Write(encoding.EncodeUint8(i.MinV.(uint8)))
		buf.Write(encoding.EncodeUint8(i.MaxV.(uint8)))
		return buf.Bytes(), nil
	case types.T_uint16:
		buf.Write(encoding.EncodeType(i.T))
		buf.Write(encoding.EncodeUint16(i.MinV.(uint16)))
		buf.Write(encoding.EncodeUint16(i.MaxV.(uint16)))
		return buf.Bytes(), nil
	case types.T_uint32:
		buf.Write(encoding.EncodeType(i.T))
		buf.Write(encoding.EncodeUint32(i.MinV.(uint32)))
		buf.Write(encoding.EncodeUint32(i.MaxV.(uint32)))
		return buf.Bytes(), nil
	case types.T_uint64:
		buf.Write(encoding.EncodeType(i.T))
		buf.Write(encoding.EncodeUint64(i.MinV.(uint64)))
		buf.Write(encoding.EncodeUint64(i.MaxV.(uint64)))
		return buf.Bytes(), nil
	case types.T_float32:
		buf.Write(encoding.EncodeType(i.T))
		buf.Write(encoding.EncodeFloat32(i.MinV.(float32)))
		buf.Write(encoding.EncodeFloat32(i.MaxV.(float32)))
		return buf.Bytes(), nil
	case types.T_float64:
		buf.Write(encoding.EncodeType(i.T))
		buf.Write(encoding.EncodeFloat64(i.MinV.(float64)))
		buf.Write(encoding.EncodeFloat64(i.MaxV.(float64)))
		return buf.Bytes(), nil
	case types.T_char, types.T_varchar, types.T_json:
		buf.Write(encoding.EncodeType(i.T))
		minv := i.MinV.([]byte)
		maxv := i.MaxV.([]byte)
		buf.Write(encoding.EncodeInt16(int16(len(minv))))
		buf.Write(minv)
		buf.Write(encoding.EncodeInt16(int16(len(maxv))))
		buf.Write(maxv)
		return buf.Bytes(), nil
	}
	panic("unsupported")
}

func (i *ZoneMapIndex) Type() base.IndexType {
	return base.ZoneMap
}

func (i *ZoneMapIndex) Eq(v interface{}) bool {
	switch i.T.Oid {
	case types.T_int8:
		return v.(int8) >= i.MinV.(int8) && v.(int8) <= i.MaxV.(int8)
	case types.T_int16:
		return v.(int16) >= i.MinV.(int16) && v.(int16) <= i.MaxV.(int16)
	case types.T_int32:
		return v.(int32) >= i.MinV.(int32) && v.(int32) <= i.MaxV.(int32)
	case types.T_int64:
		return v.(int64) >= i.MinV.(int64) && v.(int64) <= i.MaxV.(int64)
	case types.T_uint8:
		return v.(uint8) >= i.MinV.(uint8) && v.(uint8) <= i.MaxV.(uint8)
	case types.T_uint16:
		return v.(uint16) >= i.MinV.(uint16) && v.(uint16) <= i.MaxV.(uint16)
	case types.T_uint32:
		return v.(uint32) >= i.MinV.(uint32) && v.(uint32) <= i.MaxV.(uint32)
	case types.T_uint64:
		return v.(uint64) >= i.MinV.(uint64) && v.(uint64) <= i.MaxV.(uint64)
	case types.T_decimal:
		panic("not supported")
	case types.T_float32:
		return v.(float32) >= i.MinV.(float32) && v.(float32) <= i.MaxV.(float32)
	case types.T_float64:
		return v.(float64) >= i.MinV.(float64) && v.(float64) <= i.MaxV.(float64)
	case types.T_date:
		panic("not supported")
	case types.T_datetime:
		panic("not supported")
	case types.T_sel:
		return v.(int64) >= i.MinV.(int64) && v.(int64) <= i.MaxV.(int64)
	case types.T_tuple:
		panic("not supported")
	case types.T_char, types.T_varchar, types.T_json:
		if bytes.Compare(v.([]byte), i.MinV.([]byte)) < 0 {
			return false
		}
		if bytes.Compare(v.([]byte), i.MaxV.([]byte)) > 0 {
			return false
		}
		return true
	}
	panic("not supported")
}

func (i *ZoneMapIndex) Ne(v interface{}) bool {
	return !i.Eq(v)
}

func (i *ZoneMapIndex) Lt(v interface{}) bool {
	panic("TODO")
}

func (i *ZoneMapIndex) Le(v interface{}) bool {
	panic("TODO")
}

func (i *ZoneMapIndex) Gt(v interface{}) bool {
	panic("TODO")
}

func (i *ZoneMapIndex) Ge(v interface{}) bool {
	panic("TODO")
}

func (i *ZoneMapIndex) Btw(v interface{}) bool {
	panic("TODO")
}