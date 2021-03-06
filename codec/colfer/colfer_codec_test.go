package colfer

import (
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/henrylee2cn/myrpc"
	codecpkg "github.com/henrylee2cn/myrpc/codec"
)

//go:generate colf go colfer_codec_test.colf

func TestColferCodec(t *testing.T) {
	// server
	server := myrpc.NewServer(60e9, 0, 0, NewColferServerCodec)
	group, _ := server.Group(codecpkg.ServiceGroup)
	err := group.NamedRegister(codecpkg.ServiceName, new(ColfArith))
	if err != nil {
		t.Fatal(err)
	}
	go server.ListenAndServe(codecpkg.Network, codecpkg.ServerAddr)
	time.Sleep(2e9)

	// client
	client, err := myrpc.NewDialer(codecpkg.Network, codecpkg.ServerAddr, NewColferClientCodec).Dial()
	if err != nil {
		t.Fatal(err)
	}

	args := &ColfArgs{7, 8}
	var reply ColfReply
	err = client.Call(codecpkg.ServiceMethodName, args, &reply)
	if err != nil {
		t.Errorf("error for Arith: %d*%d, %v \n", args.A, args.B, err)
	} else {
		t.Logf("Arith: %d*%d=%d \n", args.A, args.B, reply.C)
	}

	client.Close()
}

func TestColferCodec2(t *testing.T) {
	// server
	server := myrpc.NewServer(60e9, 0, 0, NewColferServerCodec)
	group, _ := server.Group(codecpkg.ServiceGroup)
	err := group.NamedRegister(codecpkg.ServiceName, new(ColfArith))
	if err != nil {
		t.Fatal(err)
	}
	serverAddr := codecpkg.ServerAddr[:len(codecpkg.ServerAddr)-1] + "1"
	go server.ListenAndServe(codecpkg.Network, serverAddr)
	time.Sleep(2e9)

	// client
	var args = &ColfArgs{7, 8}
	var reply ColfReply

	err = myrpc.
		NewDialer(codecpkg.Network, serverAddr, NewColferClientCodec).
		Remote(func(client myrpc.IClient) error {
			return client.Call(codecpkg.ServiceMethodName, args, &reply)
		})

	if err != nil {
		t.Errorf("error for Arith: %d*%d, %v \n", args.A, args.B, err)
	} else {
		t.Logf("Arith: %d*%d=%d \n", args.A, args.B, reply.C)
	}
}

type ColfArith int

func (t *ColfArith) Mul(args *ColfArgs, reply *ColfReply) error {
	reply.C = args.A * args.B
	return nil
}

type ColfArgs struct {
	A int32
	B int32
}

// MarshalTo encodes o as Colfer into buf and returns the number of bytes written.
// If the buffer is too small, MarshalTo will panic.
func (o *ColfArgs) MarshalTo(buf []byte) int {
	var i int

	if v := o.A; v != 0 {
		x := uint32(v)
		if v >= 0 {
			buf[i] = 0
		} else {
			x = ^x + 1
			buf[i] = 0 | 0x80
		}
		i++
		for x >= 0x80 {
			buf[i] = byte(x | 0x80)
			x >>= 7
			i++
		}
		buf[i] = byte(x)
		i++
	}

	if v := o.B; v != 0 {
		x := uint32(v)
		if v >= 0 {
			buf[i] = 1
		} else {
			x = ^x + 1
			buf[i] = 1 | 0x80
		}
		i++
		for x >= 0x80 {
			buf[i] = byte(x | 0x80)
			x >>= 7
			i++
		}
		buf[i] = byte(x)
		i++
	}

	buf[i] = 0x7f
	i++
	return i
}

// MarshalLen returns the Colfer serial byte size.
// The error return option is testdata.ColferMax.
func (o *ColfArgs) MarshalLen() (int, error) {
	l := 1

	if v := o.A; v != 0 {
		l += 2
		x := uint32(v)
		if v < 0 {
			x = ^x + 1
		}
		for x >= 0x80 {
			x >>= 7
			l++
		}
	}

	if v := o.B; v != 0 {
		l += 2
		x := uint32(v)
		if v < 0 {
			x = ^x + 1
		}
		for x >= 0x80 {
			x >>= 7
			l++
		}
	}

	if l > ColferSizeMax {
		return l, ColferMax(fmt.Sprintf("colfer: struct testdata.ColfArgs exceeds %d bytes", ColferSizeMax))
	}
	return l, nil
}

// MarshalBinary encodes o as Colfer conform encoding.BinaryMarshaler.
// The error return option is testdata.ColferMax.
func (o *ColfArgs) MarshalBinary() (data []byte, err error) {
	l, err := o.MarshalLen()
	if err != nil {
		return nil, err
	}
	data = make([]byte, l)
	o.MarshalTo(data)
	return data, nil
}

// Unmarshal decodes data as Colfer and returns the number of bytes read.
// The error return options are io.EOF, testdata.ColferError and testdata.ColferMax.
func (o *ColfArgs) Unmarshal(data []byte) (int, error) {
	if len(data) > ColferSizeMax {
		n, err := o.Unmarshal(data[:ColferSizeMax])
		if err == io.EOF {
			return 0, ColferMax(fmt.Sprintf("colfer: struct testdata.ColfArgs exceeds %d bytes", ColferSizeMax))
		}
		return n, err
	}

	if len(data) == 0 {
		return 0, io.EOF
	}
	header := data[0]
	i := 1

	if header == 0 {
		var x uint32
		for shift := uint(0); ; shift += 7 {
			if i >= len(data) {
				return 0, io.EOF
			}
			b := data[i]
			i++
			if b < 0x80 {
				x |= uint32(b) << shift
				break
			}
			x |= (uint32(b) & 0x7f) << shift
		}
		o.A = int32(x)

		if i >= len(data) {
			return 0, io.EOF
		}
		header = data[i]
		i++
	} else if header == 0|0x80 {
		var x uint32
		for shift := uint(0); ; shift += 7 {
			if i >= len(data) {
				return 0, io.EOF
			}
			b := data[i]
			i++
			if b < 0x80 {
				x |= uint32(b) << shift
				break
			}
			x |= (uint32(b) & 0x7f) << shift
		}
		o.A = int32(^x + 1)

		if i >= len(data) {
			return 0, io.EOF
		}
		header = data[i]
		i++
	}

	if header == 1 {
		var x uint32
		for shift := uint(0); ; shift += 7 {
			if i >= len(data) {
				return 0, io.EOF
			}
			b := data[i]
			i++
			if b < 0x80 {
				x |= uint32(b) << shift
				break
			}
			x |= (uint32(b) & 0x7f) << shift
		}
		o.B = int32(x)

		if i >= len(data) {
			return 0, io.EOF
		}
		header = data[i]
		i++
	} else if header == 1|0x80 {
		var x uint32
		for shift := uint(0); ; shift += 7 {
			if i >= len(data) {
				return 0, io.EOF
			}
			b := data[i]
			i++
			if b < 0x80 {
				x |= uint32(b) << shift
				break
			}
			x |= (uint32(b) & 0x7f) << shift
		}
		o.B = int32(^x + 1)

		if i >= len(data) {
			return 0, io.EOF
		}
		header = data[i]
		i++
	}

	if header != 0x7f {
		return 0, ColferError(i - 1)
	}
	return i, nil
}

// UnmarshalBinary decodes data as Colfer conform encoding.BinaryUnmarshaler.
// The error return options are io.EOF, testdata.ColferError, testdata.ColferTail and testdata.ColferMax.
func (o *ColfArgs) UnmarshalBinary(data []byte) error {
	i, err := o.Unmarshal(data)
	if err != nil {
		return err
	}
	if i != len(data) {
		return ColferTail(i)
	}
	return nil
}

type ColfReply struct {
	C int32
}

// MarshalTo encodes o as Colfer into buf and returns the number of bytes written.
// If the buffer is too small, MarshalTo will panic.
func (o *ColfReply) MarshalTo(buf []byte) int {
	var i int

	if v := o.C; v != 0 {
		x := uint32(v)
		if v >= 0 {
			buf[i] = 0
		} else {
			x = ^x + 1
			buf[i] = 0 | 0x80
		}
		i++
		for x >= 0x80 {
			buf[i] = byte(x | 0x80)
			x >>= 7
			i++
		}
		buf[i] = byte(x)
		i++
	}

	buf[i] = 0x7f
	i++
	return i
}

// MarshalLen returns the Colfer serial byte size.
// The error return option is testdata.ColferMax.
func (o *ColfReply) MarshalLen() (int, error) {
	l := 1

	if v := o.C; v != 0 {
		l += 2
		x := uint32(v)
		if v < 0 {
			x = ^x + 1
		}
		for x >= 0x80 {
			x >>= 7
			l++
		}
	}

	if l > ColferSizeMax {
		return l, ColferMax(fmt.Sprintf("colfer: struct testdata.ColfReply exceeds %d bytes", ColferSizeMax))
	}
	return l, nil
}

// MarshalBinary encodes o as Colfer conform encoding.BinaryMarshaler.
// The error return option is testdata.ColferMax.
func (o *ColfReply) MarshalBinary() (data []byte, err error) {
	l, err := o.MarshalLen()
	if err != nil {
		return nil, err
	}
	data = make([]byte, l)
	o.MarshalTo(data)
	return data, nil
}

// Unmarshal decodes data as Colfer and returns the number of bytes read.
// The error return options are io.EOF, testdata.ColferError and testdata.ColferMax.
func (o *ColfReply) Unmarshal(data []byte) (int, error) {
	if len(data) > ColferSizeMax {
		n, err := o.Unmarshal(data[:ColferSizeMax])
		if err == io.EOF {
			return 0, ColferMax(fmt.Sprintf("colfer: struct testdata.ColfReply exceeds %d bytes", ColferSizeMax))
		}
		return n, err
	}

	if len(data) == 0 {
		return 0, io.EOF
	}
	header := data[0]
	i := 1

	if header == 0 {
		var x uint32
		for shift := uint(0); ; shift += 7 {
			if i >= len(data) {
				return 0, io.EOF
			}
			b := data[i]
			i++
			if b < 0x80 {
				x |= uint32(b) << shift
				break
			}
			x |= (uint32(b) & 0x7f) << shift
		}
		o.C = int32(x)

		if i >= len(data) {
			return 0, io.EOF
		}
		header = data[i]
		i++
	} else if header == 0|0x80 {
		var x uint32
		for shift := uint(0); ; shift += 7 {
			if i >= len(data) {
				return 0, io.EOF
			}
			b := data[i]
			i++
			if b < 0x80 {
				x |= uint32(b) << shift
				break
			}
			x |= (uint32(b) & 0x7f) << shift
		}
		o.C = int32(^x + 1)

		if i >= len(data) {
			return 0, io.EOF
		}
		header = data[i]
		i++
	}

	if header != 0x7f {
		return 0, ColferError(i - 1)
	}
	return i, nil
}

// UnmarshalBinary decodes data as Colfer conform encoding.BinaryUnmarshaler.
// The error return options are io.EOF, testdata.ColferError, testdata.ColferTail and testdata.ColferMax.
func (o *ColfReply) UnmarshalBinary(data []byte) error {
	i, err := o.Unmarshal(data)
	if err != nil {
		return err
	}
	if i != len(data) {
		return ColferTail(i)
	}
	return nil
}
