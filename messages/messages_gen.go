package messages

// Code generated by github.com/tinylib/msgp DO NOT EDIT.

import (
	"github.com/tinylib/msgp/msgp"
)

// DecodeMsg implements msgp.Decodable
func (z *ChatMessage) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, err = dc.ReadMapHeader()
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		switch msgp.UnsafeString(field) {
		case "From":
			z.From, err = dc.ReadString()
			if err != nil {
				err = msgp.WrapError(err, "From")
				return
			}
		case "Message":
			z.Message, err = dc.ReadString()
			if err != nil {
				err = msgp.WrapError(err, "Message")
				return
			}
		default:
			err = dc.Skip()
			if err != nil {
				err = msgp.WrapError(err)
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z ChatMessage) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 2
	// write "From"
	err = en.Append(0x82, 0xa4, 0x46, 0x72, 0x6f, 0x6d)
	if err != nil {
		return
	}
	err = en.WriteString(z.From)
	if err != nil {
		err = msgp.WrapError(err, "From")
		return
	}
	// write "Message"
	err = en.Append(0xa7, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65)
	if err != nil {
		return
	}
	err = en.WriteString(z.Message)
	if err != nil {
		err = msgp.WrapError(err, "Message")
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z ChatMessage) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 2
	// string "From"
	o = append(o, 0x82, 0xa4, 0x46, 0x72, 0x6f, 0x6d)
	o = msgp.AppendString(o, z.From)
	// string "Message"
	o = append(o, 0xa7, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65)
	o = msgp.AppendString(o, z.Message)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *ChatMessage) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		switch msgp.UnsafeString(field) {
		case "From":
			z.From, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "From")
				return
			}
		case "Message":
			z.Message, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Message")
				return
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				err = msgp.WrapError(err)
				return
			}
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z ChatMessage) Msgsize() (s int) {
	s = 1 + 5 + msgp.StringPrefixSize + len(z.From) + 8 + msgp.StringPrefixSize + len(z.Message)
	return
}

// DecodeMsg implements msgp.Decodable
func (z *JoinMessage) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, err = dc.ReadMapHeader()
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		switch msgp.UnsafeString(field) {
		case "From":
			z.From, err = dc.ReadString()
			if err != nil {
				err = msgp.WrapError(err, "From")
				return
			}
		default:
			err = dc.Skip()
			if err != nil {
				err = msgp.WrapError(err)
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z JoinMessage) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 1
	// write "From"
	err = en.Append(0x81, 0xa4, 0x46, 0x72, 0x6f, 0x6d)
	if err != nil {
		return
	}
	err = en.WriteString(z.From)
	if err != nil {
		err = msgp.WrapError(err, "From")
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z JoinMessage) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 1
	// string "From"
	o = append(o, 0x81, 0xa4, 0x46, 0x72, 0x6f, 0x6d)
	o = msgp.AppendString(o, z.From)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *JoinMessage) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		switch msgp.UnsafeString(field) {
		case "From":
			z.From, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "From")
				return
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				err = msgp.WrapError(err)
				return
			}
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z JoinMessage) Msgsize() (s int) {
	s = 1 + 5 + msgp.StringPrefixSize + len(z.From)
	return
}

// DecodeMsg implements msgp.Decodable
func (z *OpenPortalMessage) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, err = dc.ReadMapHeader()
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		switch msgp.UnsafeString(field) {
		case "From":
			z.From, err = dc.ReadString()
			if err != nil {
				err = msgp.WrapError(err, "From")
				return
			}
		case "To":
			z.To, err = dc.ReadString()
			if err != nil {
				err = msgp.WrapError(err, "To")
				return
			}
		case "ToLandId":
			z.ToLandId, err = dc.ReadString()
			if err != nil {
				err = msgp.WrapError(err, "ToLandId")
				return
			}
		case "LocationX":
			z.LocationX, err = dc.ReadInt64()
			if err != nil {
				err = msgp.WrapError(err, "LocationX")
				return
			}
		case "LocationY":
			z.LocationY, err = dc.ReadInt64()
			if err != nil {
				err = msgp.WrapError(err, "LocationY")
				return
			}
		default:
			err = dc.Skip()
			if err != nil {
				err = msgp.WrapError(err)
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *OpenPortalMessage) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 5
	// write "From"
	err = en.Append(0x85, 0xa4, 0x46, 0x72, 0x6f, 0x6d)
	if err != nil {
		return
	}
	err = en.WriteString(z.From)
	if err != nil {
		err = msgp.WrapError(err, "From")
		return
	}
	// write "To"
	err = en.Append(0xa2, 0x54, 0x6f)
	if err != nil {
		return
	}
	err = en.WriteString(z.To)
	if err != nil {
		err = msgp.WrapError(err, "To")
		return
	}
	// write "ToLandId"
	err = en.Append(0xa8, 0x54, 0x6f, 0x4c, 0x61, 0x6e, 0x64, 0x49, 0x64)
	if err != nil {
		return
	}
	err = en.WriteString(z.ToLandId)
	if err != nil {
		err = msgp.WrapError(err, "ToLandId")
		return
	}
	// write "LocationX"
	err = en.Append(0xa9, 0x4c, 0x6f, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x58)
	if err != nil {
		return
	}
	err = en.WriteInt64(z.LocationX)
	if err != nil {
		err = msgp.WrapError(err, "LocationX")
		return
	}
	// write "LocationY"
	err = en.Append(0xa9, 0x4c, 0x6f, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x59)
	if err != nil {
		return
	}
	err = en.WriteInt64(z.LocationY)
	if err != nil {
		err = msgp.WrapError(err, "LocationY")
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *OpenPortalMessage) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 5
	// string "From"
	o = append(o, 0x85, 0xa4, 0x46, 0x72, 0x6f, 0x6d)
	o = msgp.AppendString(o, z.From)
	// string "To"
	o = append(o, 0xa2, 0x54, 0x6f)
	o = msgp.AppendString(o, z.To)
	// string "ToLandId"
	o = append(o, 0xa8, 0x54, 0x6f, 0x4c, 0x61, 0x6e, 0x64, 0x49, 0x64)
	o = msgp.AppendString(o, z.ToLandId)
	// string "LocationX"
	o = append(o, 0xa9, 0x4c, 0x6f, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x58)
	o = msgp.AppendInt64(o, z.LocationX)
	// string "LocationY"
	o = append(o, 0xa9, 0x4c, 0x6f, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x59)
	o = msgp.AppendInt64(o, z.LocationY)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *OpenPortalMessage) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		switch msgp.UnsafeString(field) {
		case "From":
			z.From, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "From")
				return
			}
		case "To":
			z.To, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "To")
				return
			}
		case "ToLandId":
			z.ToLandId, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "ToLandId")
				return
			}
		case "LocationX":
			z.LocationX, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "LocationX")
				return
			}
		case "LocationY":
			z.LocationY, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "LocationY")
				return
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				err = msgp.WrapError(err)
				return
			}
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *OpenPortalMessage) Msgsize() (s int) {
	s = 1 + 5 + msgp.StringPrefixSize + len(z.From) + 3 + msgp.StringPrefixSize + len(z.To) + 9 + msgp.StringPrefixSize + len(z.ToLandId) + 10 + msgp.Int64Size + 10 + msgp.Int64Size
	return
}

// DecodeMsg implements msgp.Decodable
func (z *OpenPortalResponseMessage) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, err = dc.ReadMapHeader()
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		switch msgp.UnsafeString(field) {
		case "From":
			z.From, err = dc.ReadString()
			if err != nil {
				err = msgp.WrapError(err, "From")
				return
			}
		case "To":
			z.To, err = dc.ReadString()
			if err != nil {
				err = msgp.WrapError(err, "To")
				return
			}
		case "Accepted":
			z.Accepted, err = dc.ReadBool()
			if err != nil {
				err = msgp.WrapError(err, "Accepted")
				return
			}
		case "Opener":
			z.Opener, err = dc.ReadString()
			if err != nil {
				err = msgp.WrapError(err, "Opener")
				return
			}
		case "LandId":
			z.LandId, err = dc.ReadString()
			if err != nil {
				err = msgp.WrapError(err, "LandId")
				return
			}
		case "LocationX":
			z.LocationX, err = dc.ReadInt64()
			if err != nil {
				err = msgp.WrapError(err, "LocationX")
				return
			}
		case "LocationY":
			z.LocationY, err = dc.ReadInt64()
			if err != nil {
				err = msgp.WrapError(err, "LocationY")
				return
			}
		default:
			err = dc.Skip()
			if err != nil {
				err = msgp.WrapError(err)
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *OpenPortalResponseMessage) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 7
	// write "From"
	err = en.Append(0x87, 0xa4, 0x46, 0x72, 0x6f, 0x6d)
	if err != nil {
		return
	}
	err = en.WriteString(z.From)
	if err != nil {
		err = msgp.WrapError(err, "From")
		return
	}
	// write "To"
	err = en.Append(0xa2, 0x54, 0x6f)
	if err != nil {
		return
	}
	err = en.WriteString(z.To)
	if err != nil {
		err = msgp.WrapError(err, "To")
		return
	}
	// write "Accepted"
	err = en.Append(0xa8, 0x41, 0x63, 0x63, 0x65, 0x70, 0x74, 0x65, 0x64)
	if err != nil {
		return
	}
	err = en.WriteBool(z.Accepted)
	if err != nil {
		err = msgp.WrapError(err, "Accepted")
		return
	}
	// write "Opener"
	err = en.Append(0xa6, 0x4f, 0x70, 0x65, 0x6e, 0x65, 0x72)
	if err != nil {
		return
	}
	err = en.WriteString(z.Opener)
	if err != nil {
		err = msgp.WrapError(err, "Opener")
		return
	}
	// write "LandId"
	err = en.Append(0xa6, 0x4c, 0x61, 0x6e, 0x64, 0x49, 0x64)
	if err != nil {
		return
	}
	err = en.WriteString(z.LandId)
	if err != nil {
		err = msgp.WrapError(err, "LandId")
		return
	}
	// write "LocationX"
	err = en.Append(0xa9, 0x4c, 0x6f, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x58)
	if err != nil {
		return
	}
	err = en.WriteInt64(z.LocationX)
	if err != nil {
		err = msgp.WrapError(err, "LocationX")
		return
	}
	// write "LocationY"
	err = en.Append(0xa9, 0x4c, 0x6f, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x59)
	if err != nil {
		return
	}
	err = en.WriteInt64(z.LocationY)
	if err != nil {
		err = msgp.WrapError(err, "LocationY")
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *OpenPortalResponseMessage) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 7
	// string "From"
	o = append(o, 0x87, 0xa4, 0x46, 0x72, 0x6f, 0x6d)
	o = msgp.AppendString(o, z.From)
	// string "To"
	o = append(o, 0xa2, 0x54, 0x6f)
	o = msgp.AppendString(o, z.To)
	// string "Accepted"
	o = append(o, 0xa8, 0x41, 0x63, 0x63, 0x65, 0x70, 0x74, 0x65, 0x64)
	o = msgp.AppendBool(o, z.Accepted)
	// string "Opener"
	o = append(o, 0xa6, 0x4f, 0x70, 0x65, 0x6e, 0x65, 0x72)
	o = msgp.AppendString(o, z.Opener)
	// string "LandId"
	o = append(o, 0xa6, 0x4c, 0x61, 0x6e, 0x64, 0x49, 0x64)
	o = msgp.AppendString(o, z.LandId)
	// string "LocationX"
	o = append(o, 0xa9, 0x4c, 0x6f, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x58)
	o = msgp.AppendInt64(o, z.LocationX)
	// string "LocationY"
	o = append(o, 0xa9, 0x4c, 0x6f, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x59)
	o = msgp.AppendInt64(o, z.LocationY)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *OpenPortalResponseMessage) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		switch msgp.UnsafeString(field) {
		case "From":
			z.From, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "From")
				return
			}
		case "To":
			z.To, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "To")
				return
			}
		case "Accepted":
			z.Accepted, bts, err = msgp.ReadBoolBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Accepted")
				return
			}
		case "Opener":
			z.Opener, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Opener")
				return
			}
		case "LandId":
			z.LandId, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "LandId")
				return
			}
		case "LocationX":
			z.LocationX, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "LocationX")
				return
			}
		case "LocationY":
			z.LocationY, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "LocationY")
				return
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				err = msgp.WrapError(err)
				return
			}
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *OpenPortalResponseMessage) Msgsize() (s int) {
	s = 1 + 5 + msgp.StringPrefixSize + len(z.From) + 3 + msgp.StringPrefixSize + len(z.To) + 9 + msgp.BoolSize + 7 + msgp.StringPrefixSize + len(z.Opener) + 7 + msgp.StringPrefixSize + len(z.LandId) + 10 + msgp.Int64Size + 10 + msgp.Int64Size
	return
}

// DecodeMsg implements msgp.Decodable
func (z *ShoutMessage) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, err = dc.ReadMapHeader()
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		switch msgp.UnsafeString(field) {
		case "From":
			z.From, err = dc.ReadString()
			if err != nil {
				err = msgp.WrapError(err, "From")
				return
			}
		case "Message":
			z.Message, err = dc.ReadString()
			if err != nil {
				err = msgp.WrapError(err, "Message")
				return
			}
		default:
			err = dc.Skip()
			if err != nil {
				err = msgp.WrapError(err)
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z ShoutMessage) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 2
	// write "From"
	err = en.Append(0x82, 0xa4, 0x46, 0x72, 0x6f, 0x6d)
	if err != nil {
		return
	}
	err = en.WriteString(z.From)
	if err != nil {
		err = msgp.WrapError(err, "From")
		return
	}
	// write "Message"
	err = en.Append(0xa7, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65)
	if err != nil {
		return
	}
	err = en.WriteString(z.Message)
	if err != nil {
		err = msgp.WrapError(err, "Message")
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z ShoutMessage) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 2
	// string "From"
	o = append(o, 0x82, 0xa4, 0x46, 0x72, 0x6f, 0x6d)
	o = msgp.AppendString(o, z.From)
	// string "Message"
	o = append(o, 0xa7, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65)
	o = msgp.AppendString(o, z.Message)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *ShoutMessage) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		switch msgp.UnsafeString(field) {
		case "From":
			z.From, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "From")
				return
			}
		case "Message":
			z.Message, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Message")
				return
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				err = msgp.WrapError(err)
				return
			}
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z ShoutMessage) Msgsize() (s int) {
	s = 1 + 5 + msgp.StringPrefixSize + len(z.From) + 8 + msgp.StringPrefixSize + len(z.Message)
	return
}
