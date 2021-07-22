package protocol

import (
	"bytes"
	"fmt"
	"matrixone/pkg/container/batch"
	"matrixone/pkg/container/types"
	"matrixone/pkg/container/vector"
	"matrixone/pkg/encoding"
	"matrixone/pkg/sql/colexec/extend"
	"matrixone/pkg/vm/engine"
)

func EncodePartition(def *engine.PartitionBy, buf *bytes.Buffer) error {
	data, err := encoding.Encode(def.Fields)
	if err != nil {
		return err
	}
	buf.Write(encoding.EncodeUint32(uint32(len(data))))
	buf.Write(data)
	if n := len(def.List); n > 0 {
		buf.Write(encoding.EncodeUint32(uint32(n)))
		for i := 0; i < n; i++ {
			if err := EncodeListPartition(def.List[i], buf); err != nil {
				return err
			}
		}
	} else {
		buf.Write(encoding.EncodeUint32(0))
	}
	if n := len(def.Range); n > 0 {
		buf.Write(encoding.EncodeUint32(uint32(n)))
		for i := 0; i < n; i++ {
			if err := EncodeRangePartition(def.Range[i], buf); err != nil {
				return err
			}
		}
	} else {
		buf.Write(encoding.EncodeUint32(0))
	}
	return nil
}

func DecodePartition(data []byte) (*engine.PartitionBy, []byte, error) {
	def := new(engine.PartitionBy)
	if n := encoding.DecodeUint32(data[:4]); n > 0 {
		data = data[4:]
		if err := encoding.Decode(data[:n], &def.Fields); err != nil {
			return nil, nil, err
		}
		data = data[n:]
	} else {
		data = data[4:]
	}
	if n := encoding.DecodeUint32(data[:4]); n > 0 {
		data = data[4:]
		for i := uint32(0); i < n; i++ {
			ldef, remaing, err := DecodeListPartition(data)
			if err != nil {
				return nil, nil, err
			}
			def.List = append(def.List, ldef)
			data = remaing
		}
	} else {
		data = data[4:]
	}
	if n := encoding.DecodeUint32(data[:4]); n > 0 {
		data = data[4:]
		for i := uint32(0); i < n; i++ {
			rdef, remaing, err := DecodeRangePartition(data)
			if err != nil {
				return nil, nil, err
			}
			def.Range = append(def.Range, rdef)
			data = remaing
		}
	} else {
		data = data[4:]
	}
	return def, data, nil
}

func EncodeListPartition(def engine.ListPartition, buf *bytes.Buffer) error {
	if def.Subpartition != nil {
		buf.WriteByte(ListWithSub)
	} else {
		buf.WriteByte(List)
	}
	buf.Write(encoding.EncodeUint32(uint32(len(def.Name))))
	buf.WriteString(def.Name)
	if n := len(def.Extends); n > 0 {
		buf.Write(encoding.EncodeUint32(uint32(n)))
		for _, e := range def.Extends {
			if err := EncodeExtend(e, buf); err != nil {
				return err
			}
		}
	}
	if def.Subpartition != nil {
		return EncodePartition(def.Subpartition, buf)
	}
	return nil
}

func DecodeListPartition(data []byte) (engine.ListPartition, []byte, error) {
	var def engine.ListPartition

	typ := data[0]
	data = data[1:]
	if n := encoding.DecodeUint32(data[:4]); n > 0 {
		data = data[4:]
		def.Name = string(data[:n])
		data = data[n:]
	} else {
		data = data[4:]
	}
	if n := encoding.DecodeUint32(data[:4]); n > 0 {
		data = data[4:]
		for i := uint32(0); i < n; i++ {
			e, remaing, err := DecodeExtend(data)
			if err != nil {
				return def, nil, err
			}
			def.Extends = append(def.Extends, e)
			data = remaing
		}
	} else {
		data = data[4:]
	}
	if typ == ListWithSub {
		sub, remaing, err := DecodePartition(data)
		if err != nil {
			return def, nil, err
		}
		data = remaing
		def.Subpartition = sub
	}
	return def, data, nil
}

func EncodeRangePartition(def engine.RangePartition, buf *bytes.Buffer) error {
	if def.Subpartition != nil {
		buf.WriteByte(RangeWithSub)
	} else {
		buf.WriteByte(Range)
	}
	buf.Write(encoding.EncodeUint32(uint32(len(def.Name))))
	buf.WriteString(def.Name)
	if n := len(def.From); n > 0 {
		buf.Write(encoding.EncodeUint32(uint32(n)))
		for _, e := range def.From {
			if err := EncodeExtend(e, buf); err != nil {
				return err
			}
		}
	}
	if n := len(def.To); n > 0 {
		buf.Write(encoding.EncodeUint32(uint32(n)))
		for _, e := range def.To {
			if err := EncodeExtend(e, buf); err != nil {
				return err
			}
		}
	}
	if def.Subpartition != nil {
		return EncodePartition(def.Subpartition, buf)
	}
	return nil
}

func DecodeRangePartition(data []byte) (engine.RangePartition, []byte, error) {
	var def engine.RangePartition

	typ := data[0]
	data = data[1:]
	if n := encoding.DecodeUint32(data[:4]); n > 0 {
		data = data[4:]
		def.Name = string(data[:n])
		data = data[n:]
	} else {
		data = data[4:]
	}
	if n := encoding.DecodeUint32(data[:4]); n > 0 {
		data = data[4:]
		for i := uint32(0); i < n; i++ {
			e, remaing, err := DecodeExtend(data)
			if err != nil {
				return def, nil, err
			}
			def.From = append(def.From, e)
			data = remaing
		}
	} else {
		data = data[4:]
	}
	if n := encoding.DecodeUint32(data[:4]); n > 0 {
		data = data[4:]
		for i := uint32(0); i < n; i++ {
			e, remaing, err := DecodeExtend(data)
			if err != nil {
				return def, nil, err
			}
			def.To = append(def.To, e)
			data = remaing
		}
	} else {
		data = data[4:]
	}
	if typ == ListWithSub {
		sub, remaing, err := DecodePartition(data)
		if err != nil {
			return def, nil, err
		}
		data = remaing
		def.Subpartition = sub
	}
	return def, data, nil
}

func EncodeExtend(e extend.Extend, buf *bytes.Buffer) error {
	switch v := e.(type) {
	case *extend.Attribute:
		buf.WriteByte(Attr)
		buf.Write(encoding.EncodeUint16(uint16(v.Type)))
		buf.Write(encoding.EncodeUint32(uint32(len(v.Name))))
		buf.WriteString(v.Name)
		return nil
	case *extend.UnaryExtend:
		buf.WriteByte(Unary)
		buf.Write(encoding.EncodeUint32(uint32(v.Op)))
		return nil
	case *extend.ParenExtend:
		buf.WriteByte(Paren)
		return EncodeExtend(v.E, buf)
	case *extend.ValueExtend:
		buf.WriteByte(Value)
		return EncodeVector(v.V, buf)
		return nil
	case *extend.BinaryExtend:
		buf.WriteByte(Binary)
		buf.Write(encoding.EncodeUint32(uint32(v.Op)))
		if err := EncodeExtend(v.Left, buf); err != nil {
			return err
		}
		return EncodeExtend(v.Right, buf)
	}
	return fmt.Errorf("'%s' not yet support", e)
}

func DecodeExtend(data []byte) (extend.Extend, []byte, error) {
	switch data[0] {
	case Attr:
		e := new(extend.Attribute)
		data = data[1:]
		e.Type = types.T(encoding.DecodeUint16(data[:2]))
		data = data[2:]
		if n := encoding.DecodeUint32(data[:4]); n > 0 {
			data = data[4:]
			e.Name = string(data[:n])
			data = data[n:]
		} else {
			data = data[4:]
		}
		return e, data, nil
	case Unary:
		e := new(extend.UnaryExtend)
		data = data[1:]
		e.Op = int(encoding.DecodeUint32(data[:4]))
		data = data[4:]
		ext, data, err := DecodeExtend(data)
		if err != nil {
			return nil, nil, err
		}
		e.E = ext
		return e, data, nil
	case Paren:
		e := new(extend.ParenExtend)
		data = data[1:]
		ext, data, err := DecodeExtend(data)
		if err != nil {
			return nil, nil, err
		}
		e.E = ext
		return e, data, nil
	case Value:
		e := new(extend.ValueExtend)
		data = data[1:]
		v, data, err := DecodeVector(data)
		if err != nil {
			return nil, nil, err
		}
		e.V = v
		return e, data, nil
	case Binary:
		e := new(extend.BinaryExtend)
		data = data[1:]
		e.Op = int(encoding.DecodeUint32(data[:4]))
		data = data[4:]
		le, data, err := DecodeExtend(data)
		if err != nil {
			return nil, nil, err
		}
		re, data, err := DecodeExtend(data)
		if err != nil {
			return nil, nil, err
		}
		e.Left, e.Right = le, re
		return e, data, nil
	}
	return nil, nil, fmt.Errorf("'%v' extend not yet support", data[0])
}

func EncodeBatch(bat *batch.Batch, buf *bytes.Buffer) error {
	sn := len(bat.Sels)
	buf.Write(encoding.EncodeUint32(uint32(sn)))
	if sn > 0 {
		buf.Write(encoding.EncodeInt64Slice(bat.Sels))
	}
	data, err := encoding.Encode(bat.Attrs)
	if err != nil {
		return err
	}
	buf.Write(encoding.EncodeUint32(uint32(len(data))))
	buf.Write(data)
	buf.Write(encoding.EncodeUint32(uint32(len(bat.Vecs))))
	for _, vec := range bat.Vecs {
		if err := EncodeVector(vec, buf); err != nil {
			return err
		}
	}
	return nil
}

func DecodeBatch(data []byte) (*batch.Batch, []byte, error) {
	bat := batch.New(true, []string{})
	sn := encoding.DecodeUint32(data[:4])
	data = data[4:]
	if sn > 0 {
		bat.Sels = encoding.DecodeInt64Slice(data[:sn*8])
		data = data[sn*8:]
	}
	if n := encoding.DecodeUint32(data); n > 0 {
		data = data[4:]
		if err := encoding.Decode(data[:n], &bat.Attrs); err != nil {
			return nil, nil, err
		}
		data = data[n:]
	} else {
		data = data[4:]
	}
	if n := encoding.DecodeUint32(data); n > 0 {
		data = data[4:]
		for i := uint32(0); i < n; i++ {
			vec, remaing, err := DecodeVector(data)
			if err != nil {
				return nil, nil, err
			}
			bat.Vecs = append(bat.Vecs, vec)
			data = remaing
		}
	} else {
		data = data[4:]
	}
	return bat, data, nil
}

func EncodeVector(v *vector.Vector, buf *bytes.Buffer) error {
	switch v.Typ.Oid {
	case types.T_int8:
		buf.Write(encoding.EncodeType(v.Typ))
		nb, err := v.Nsp.Show()
		if err != nil {
			return err
		}
		buf.Write(encoding.EncodeUint32(uint32(len(nb))))
		if len(nb) > 0 {
			buf.Write(nb)
		}
		vs := v.Col.([]int8)
		buf.Write(encoding.EncodeUint32(uint32(len(vs))))
		buf.Write(encoding.EncodeInt8Slice(vs))
	case types.T_int16:
		buf.Write(encoding.EncodeType(v.Typ))
		nb, err := v.Nsp.Show()
		if err != nil {
			return err
		}
		buf.Write(encoding.EncodeUint32(uint32(len(nb))))
		if len(nb) > 0 {
			buf.Write(nb)
		}
		vs := v.Col.([]int16)
		buf.Write(encoding.EncodeUint32(uint32(len(vs))))
		buf.Write(encoding.EncodeInt16Slice(vs))
	case types.T_int32:
		buf.Write(encoding.EncodeType(v.Typ))
		nb, err := v.Nsp.Show()
		if err != nil {
			return err
		}
		buf.Write(encoding.EncodeUint32(uint32(len(nb))))
		if len(nb) > 0 {
			buf.Write(nb)
		}
		vs := v.Col.([]int32)
		buf.Write(encoding.EncodeUint32(uint32(len(vs))))
		buf.Write(encoding.EncodeInt32Slice(vs))
	case types.T_int64, types.T_sel:
		buf.Write(encoding.EncodeType(v.Typ))
		nb, err := v.Nsp.Show()
		if err != nil {
			return err
		}
		buf.Write(encoding.EncodeUint32(uint32(len(nb))))
		if len(nb) > 0 {
			buf.Write(nb)
		}
		vs := v.Col.([]int64)
		buf.Write(encoding.EncodeUint32(uint32(len(vs))))
		buf.Write(encoding.EncodeInt64Slice(vs))
	case types.T_uint8:
		buf.Write(encoding.EncodeType(v.Typ))
		nb, err := v.Nsp.Show()
		if err != nil {
			return err
		}
		buf.Write(encoding.EncodeUint32(uint32(len(nb))))
		if len(nb) > 0 {
			buf.Write(nb)
		}
		vs := v.Col.([]uint8)
		buf.Write(encoding.EncodeUint32(uint32(len(vs))))
		buf.Write(encoding.EncodeUint8Slice(vs))
	case types.T_uint16:
		buf.Write(encoding.EncodeType(v.Typ))
		nb, err := v.Nsp.Show()
		if err != nil {
			return err
		}
		buf.Write(encoding.EncodeUint32(uint32(len(nb))))
		if len(nb) > 0 {
			buf.Write(nb)
		}
		vs := v.Col.([]uint16)
		buf.Write(encoding.EncodeUint32(uint32(len(vs))))
		buf.Write(encoding.EncodeUint16Slice(vs))
	case types.T_uint32:
		buf.Write(encoding.EncodeType(v.Typ))
		nb, err := v.Nsp.Show()
		if err != nil {
			return err
		}
		buf.Write(encoding.EncodeUint32(uint32(len(nb))))
		if len(nb) > 0 {
			buf.Write(nb)
		}
		vs := v.Col.([]uint32)
		buf.Write(encoding.EncodeUint32(uint32(len(vs))))
		buf.Write(encoding.EncodeUint32Slice(vs))
	case types.T_uint64:
		buf.Write(encoding.EncodeType(v.Typ))
		nb, err := v.Nsp.Show()
		if err != nil {
			return err
		}
		buf.Write(encoding.EncodeUint32(uint32(len(nb))))
		if len(nb) > 0 {
			buf.Write(nb)
		}
		vs := v.Col.([]uint64)
		buf.Write(encoding.EncodeUint32(uint32(len(vs))))
		buf.Write(encoding.EncodeUint64Slice(vs))
	case types.T_float32:
		buf.Write(encoding.EncodeType(v.Typ))
		nb, err := v.Nsp.Show()
		if err != nil {
			return err
		}
		buf.Write(encoding.EncodeUint32(uint32(len(nb))))
		if len(nb) > 0 {
			buf.Write(nb)
		}
		vs := v.Col.([]float32)
		buf.Write(encoding.EncodeUint32(uint32(len(vs))))
		buf.Write(encoding.EncodeFloat32Slice(vs))
	case types.T_float64:
		buf.Write(encoding.EncodeType(v.Typ))
		nb, err := v.Nsp.Show()
		if err != nil {
			return err
		}
		buf.Write(encoding.EncodeUint32(uint32(len(nb))))
		if len(nb) > 0 {
			buf.Write(nb)
		}
		vs := v.Col.([]float64)
		buf.Write(encoding.EncodeUint32(uint32(len(vs))))
		buf.Write(encoding.EncodeFloat64Slice(vs))
	case types.T_char, types.T_varchar:
		buf.Write(encoding.EncodeType(v.Typ))
		nb, err := v.Nsp.Show()
		if err != nil {
			return err
		}
		buf.Write(encoding.EncodeUint32(uint32(len(nb))))
		if len(nb) > 0 {
			buf.Write(nb)
		}
		Col := v.Col.(*types.Bytes)
		cnt := int32(len(Col.Offsets))
		buf.Write(encoding.EncodeInt32(cnt))
		if cnt != 0 {
			buf.Write(encoding.EncodeUint32Slice(Col.Lengths))
			buf.Write(encoding.EncodeUint64(uint64(len(Col.Data))))
			buf.Write(Col.Data)
		}
	case types.T_tuple:
		buf.Write(encoding.EncodeType(v.Typ))
		nb, err := v.Nsp.Show()
		if err != nil {
			return err
		}
		buf.Write(encoding.EncodeUint32(uint32(len(nb))))
		if len(nb) > 0 {
			buf.Write(nb)
		}
		data, err := encoding.Encode(v.Col.([][]interface{}))
		if err != nil {
			return err
		}
		buf.Write(encoding.EncodeUint32(uint32(len(data))))
		if len(data) > 0 {
			buf.Write(data)
		}
	default:
		return fmt.Errorf("unsupport vector type '%s'", v.Typ)
	}
	return nil
}

func DecodeVector(data []byte) (*vector.Vector, []byte, error) {
	typ := encoding.DecodeType(data[:encoding.TypeSize])
	data = data[encoding.TypeSize:]
	switch typ.Oid {
	case types.T_int8:
		v := vector.New(typ)
		if n := encoding.DecodeUint32(data[:4]); n > 0 {
			data = data[4:]
			if err := v.Nsp.Read(data[:n]); err != nil {
				return nil, nil, err
			}
			data = data[n:]
		} else {
			data = data[4:]
		}
		if n := encoding.DecodeUint32(data[:4]); n > 0 {
			data = data[4:]
			v.Col = encoding.DecodeInt8Slice(data[:n])
			data = data[n:]
		}
		return v, data, nil
	case types.T_int16:
		v := vector.New(typ)
		if n := encoding.DecodeUint32(data[:4]); n > 0 {
			data = data[4:]
			if err := v.Nsp.Read(data[:n]); err != nil {
				return nil, nil, err
			}
			data = data[n:]
		} else {
			data = data[4:]
		}
		if n := encoding.DecodeUint32(data[:4]); n > 0 {
			data = data[4:]
			v.Col = encoding.DecodeInt16Slice(data[:n*2])
			data = data[n*2:]
		}
		return v, data, nil
	case types.T_int32:
		v := vector.New(typ)
		if n := encoding.DecodeUint32(data[:4]); n > 0 {
			data = data[4:]
			if err := v.Nsp.Read(data[:n]); err != nil {
				return nil, nil, err
			}
			data = data[n:]
		} else {
			data = data[4:]
		}
		if n := encoding.DecodeUint32(data[:4]); n > 0 {
			data = data[4:]
			v.Col = encoding.DecodeInt32Slice(data[:n*4])
			data = data[n*4:]
		}
		return v, data, nil
	case types.T_int64, types.T_sel:
		v := vector.New(typ)
		if n := encoding.DecodeUint32(data[:4]); n > 0 {
			data = data[4:]
			if err := v.Nsp.Read(data[:n]); err != nil {
				return nil, nil, err
			}
			data = data[n:]
		} else {
			data = data[4:]
		}
		if n := encoding.DecodeUint32(data[:4]); n > 0 {
			data = data[4:]
			v.Col = encoding.DecodeInt64Slice(data[:n*8])
			data = data[n*8:]
		}
		return v, data, nil
	case types.T_uint8:
		v := vector.New(typ)
		if n := encoding.DecodeUint32(data[:4]); n > 0 {
			data = data[4:]
			if err := v.Nsp.Read(data[:n]); err != nil {
				return nil, nil, err
			}
			data = data[n:]
		} else {
			data = data[4:]
		}
		if n := encoding.DecodeUint32(data[:4]); n > 0 {
			data = data[4:]
			v.Col = encoding.DecodeUint8Slice(data[:n])
			data = data[n:]
		}
		return v, data, nil
	case types.T_uint16:
		v := vector.New(typ)
		if n := encoding.DecodeUint32(data[:4]); n > 0 {
			data = data[4:]
			if err := v.Nsp.Read(data[:n]); err != nil {
				return nil, nil, err
			}
			data = data[n:]
		} else {
			data = data[4:]
		}
		if n := encoding.DecodeUint32(data[:4]); n > 0 {
			data = data[4:]
			v.Col = encoding.DecodeUint16Slice(data[:n*2])
			data = data[n*2:]
		}
		return v, data, nil
	case types.T_uint32:
		v := vector.New(typ)
		if n := encoding.DecodeUint32(data[:4]); n > 0 {
			data = data[4:]
			if err := v.Nsp.Read(data[:n]); err != nil {
				return nil, nil, err
			}
			data = data[n:]
		} else {
			data = data[4:]
		}
		if n := encoding.DecodeUint32(data[:4]); n > 0 {
			data = data[4:]
			v.Col = encoding.DecodeUint32Slice(data[:n*4])
			data = data[n*4:]
		}
		return v, data, nil
	case types.T_uint64:
		v := vector.New(typ)
		if n := encoding.DecodeUint32(data[:4]); n > 0 {
			data = data[4:]
			if err := v.Nsp.Read(data[:n]); err != nil {
				return nil, nil, err
			}
			data = data[n:]
		} else {
			data = data[4:]
		}
		if n := encoding.DecodeUint32(data[:4]); n > 0 {
			data = data[4:]
			v.Col = encoding.DecodeUint64Slice(data[:n*8])
			data = data[n*8:]
		}
		return v, data, nil
	case types.T_float32:
		v := vector.New(typ)
		if n := encoding.DecodeUint32(data[:4]); n > 0 {
			data = data[4:]
			if err := v.Nsp.Read(data[:n]); err != nil {
				return nil, nil, err
			}
			data = data[n:]
		} else {
			data = data[4:]
		}
		if n := encoding.DecodeUint32(data[:4]); n > 0 {
			data = data[4:]
			v.Col = encoding.DecodeFloat32Slice(data[:n*4])
			data = data[n*4:]
		}
		return v, data, nil
	case types.T_float64:
		v := vector.New(typ)
		if n := encoding.DecodeUint32(data[:4]); n > 0 {
			data = data[4:]
			if err := v.Nsp.Read(data[:n]); err != nil {
				return nil, nil, err
			}
			data = data[n:]
		} else {
			data = data[4:]
		}
		if n := encoding.DecodeUint32(data[:4]); n > 0 {
			data = data[4:]
			v.Col = encoding.DecodeFloat64Slice(data[:n*8])
			data = data[n*8:]
		}
		return v, data, nil
	case types.T_char, types.T_varchar:
		v := vector.New(typ)
		if n := encoding.DecodeUint32(data[:4]); n > 0 {
			data = data[4:]
			if err := v.Nsp.Read(data[:n]); err != nil {
				return nil, nil, err
			}
			data = data[n:]
		} else {
			data = data[4:]
		}
		col := new(types.Bytes)
		if n := encoding.DecodeUint32(data[:4]); n > 0 {
			data = data[4:]
			col.Lengths = encoding.DecodeUint32Slice(data[:4*n])
			data = data[4*n:]
			m := encoding.DecodeUint64(data[:8])
			data = data[8:]
			col.Data = data[:m]
			data = data[m:]
			{
				o := uint32(0)
				col.Offsets = make([]uint32, n)
				for i, n := range col.Lengths {
					col.Offsets[i] = 0
					o += n
				}
			}
			v.Col = col
		}
		return v, data, nil
	case types.T_tuple:
		v := vector.New(typ)
		if n := encoding.DecodeUint32(data[:4]); n > 0 {
			data = data[4:]
			if err := v.Nsp.Read(data[:n]); err != nil {
				return nil, nil, err
			}
			data = data[n:]
		} else {
			data = data[4:]
		}
		if n := encoding.DecodeUint32(data[:4]); n > 0 {
			data = data[4:]
			col := v.Col.([][]interface{})
			if err := encoding.Decode(data[:n], &col); err != nil {
				return nil, nil, err
			}
			data = data[n:]
			v.Col = col
		}
		return v, data, nil
	}
	return nil, nil, fmt.Errorf("unsupport vector type '%s'", typ)
}