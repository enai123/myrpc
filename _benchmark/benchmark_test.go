package myrpc

import (
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/net-rpc-msgpackrpc"
	cli "github.com/henrylee2cn/myrpc/client"
	"github.com/henrylee2cn/myrpc/client/selector"
	"github.com/henrylee2cn/myrpc/codec"
	"github.com/henrylee2cn/myrpc/codec/gencode"
	"github.com/henrylee2cn/myrpc/codec/gob"
	"github.com/henrylee2cn/myrpc/codec/protobuf"
	"github.com/henrylee2cn/myrpc/log"
	srv "github.com/henrylee2cn/myrpc/server"
)

// don't use it to test benchmark. It is only used to evaluate libs internally.

func listenTCP() (net.Listener, string) {
	l, e := net.Listen("tcp", "127.0.0.1:0") // any available address
	if e != nil {
		log.Fatalf("net.Listen tcp :0: %v", e)
	}
	return l, l.Addr().String()
}

func benchmarkClient(client *rpc.Client, b *testing.B) {
	// Synchronous calls
	args := &codec.Args{7, 8}
	procs := runtime.GOMAXPROCS(-1)
	N := int32(b.N)
	var wg sync.WaitGroup
	wg.Add(procs)
	b.StartTimer()

	for p := 0; p < procs; p++ {
		go func() {
			reply := new(codec.Reply)
			for atomic.AddInt32(&N, -1) >= 0 {
				err := client.Call("Arith.Mul", args, reply)
				if err != nil {
					b.Errorf("rpc error: Mul: expected no error but got string %q", err.Error())
				}
				if reply.C != args.A*args.B {
					b.Errorf("rpc error: Mul: expected %d got %d", reply.C, args.A*args.B)
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()
	b.StopTimer()
}

func benchmarkMyrpcClient(client *cli.Client, b *testing.B) {
	// Synchronous calls
	args := &codec.Args{7, 8}
	procs := runtime.GOMAXPROCS(-1)
	N := int32(b.N)
	var wg sync.WaitGroup
	wg.Add(procs)
	b.StartTimer()

	for p := 0; p < procs; p++ {
		go func() {
			reply := new(codec.Reply)
			for atomic.AddInt32(&N, -1) >= 0 {
				err := client.Call("/arith/mul", args, reply)
				if err != nil {
					b.Errorf("rpc error: Mul: expected no error but got string %q", err.Error())
				}
				if reply.C != args.A*args.B {
					b.Errorf("rpc error: Mul: expected %d got %d", reply.C, args.A*args.B)
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()
	b.StopTimer()
}

func benchmarkMyrpcGencodeClient(client *cli.Client, b *testing.B) {
	// Synchronous calls
	args := &GencodeArgs{7, 8}
	procs := runtime.GOMAXPROCS(-1)
	N := int32(b.N)
	var wg sync.WaitGroup
	wg.Add(procs)
	b.StartTimer()

	for p := 0; p < procs; p++ {
		go func() {
			reply := new(GencodeReply)
			for atomic.AddInt32(&N, -1) >= 0 {
				err := client.Call("/arith/mul", args, reply)
				if err != nil {
					b.Errorf("rpc error: Mul: expected no error but got string %q", err.Error())
				}
				if reply.C != args.A*args.B {
					b.Errorf("rpc error: Mul: expected %d got %d", reply.C, args.A*args.B)
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()
	b.StopTimer()
}

func benchmarkMyrpcProtobufClient(client *cli.Client, b *testing.B) {
	// Synchronous calls
	args := &ProtoArgs{7, 8}
	procs := runtime.GOMAXPROCS(-1)
	N := int32(b.N)
	var wg sync.WaitGroup
	wg.Add(procs)
	b.StartTimer()

	for p := 0; p < procs; p++ {
		go func() {
			reply := new(ProtoReply)
			for atomic.AddInt32(&N, -1) >= 0 {
				err := client.Call("/arith/mul", args, reply)
				if err != nil {
					b.Errorf("rpc error: Mul: expected no error but got string %q", err.Error())
				}
				if reply.C != args.A*args.B {
					b.Errorf("rpc error: Mul: expected %d got %d", reply.C, args.A*args.B)
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()
	b.StopTimer()
}
func startNetRPCWithGob() (ln net.Listener, address string) {
	rpc.Register(new(codec.Arith))
	ln, address = listenTCP()

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				log.Fatal("accept error:", err)
			}

			go rpc.ServeConn(conn)
		}
	}()

	return
}

func BenchmarkNetRPC_gob(b *testing.B) {
	b.StopTimer()
	_, address := startNetRPCWithGob()

	conn, err := net.Dial("tcp", address)
	if err != nil {
		log.Fatal("error dialing:", err)
	}
	client := rpc.NewClient(conn)
	defer client.Close()

	benchmarkClient(client, b)
}

func startNetRPCWithJson() (ln net.Listener, address string) {
	rpc.Register(new(codec.Arith))
	ln, address = listenTCP()

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				log.Fatal("accept error:", err)
			}

			go jsonrpc.ServeConn(conn)
		}
	}()

	return
}

func BenchmarkNetRPC_jsonrpc(b *testing.B) {
	b.StopTimer()
	_, address := startNetRPCWithJson()

	conn, err := net.Dial("tcp", address)
	if err != nil {
		b.Fatal("error dialing:", err)
	}
	client := jsonrpc.NewClient(conn)
	defer client.Close()

	benchmarkClient(client, b)
}

func startNetRPCWithMsgp() (ln net.Listener, address string) {
	rpc.Register(new(codec.Arith))
	ln, address = listenTCP()

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				log.Fatal("accept error:", err)
			}

			go msgpackrpc.ServeConn(conn)
		}
	}()

	return
}

func BenchmarkNetRPC_msgp(b *testing.B) {
	b.StopTimer()
	_, address := startNetRPCWithMsgp()

	conn, err := net.Dial("tcp", address)
	if err != nil {
		b.Fatal("error dialing:", err)
	}
	client := msgpackrpc.NewClient(conn)
	defer client.Close()

	benchmarkClient(client, b)
}

func startMyrpcWithGob() *srv.Server {
	server := srv.NewServer(srv.Server{
		ServerCodecFunc: gob.NewGobServerCodec,
	})
	server.NamedRegister("Arith", new(codec.Arith))
	ln, _ := listenTCP()
	go server.ServeListener(ln)

	return server
}

func BenchmarkMyrpc_gob(b *testing.B) {
	b.StopTimer()
	server := startMyrpcWithGob()
	time.Sleep(5 * time.Second) //waiting for starting server

	client := cli.NewClient(
		cli.Client{
			ClientCodecFunc: gob.NewGobClientCodec,
			FailMode:        cli.Failtry,
		},
		&selector.DirectSelector{
			Network:     "tcp",
			Address:     server.Address(),
			DialTimeout: 10 * time.Second,
		},
	)
	defer client.Close()

	benchmarkMyrpcClient(client, b)
}

func startMyrpcWithJson() *srv.Server {
	server := srv.NewServer(srv.Server{
		ServerCodecFunc: jsonrpc.NewServerCodec,
	})
	server.NamedRegister("Arith", new(codec.Arith))
	ln, _ := listenTCP()
	go server.ServeListener(ln)

	return server
}

func BenchmarkMyrpc_jsonrpc(b *testing.B) {
	b.StopTimer()
	server := startMyrpcWithJson()
	time.Sleep(5 * time.Second) //waiting for starting server
	client := cli.NewClient(
		cli.Client{
			ClientCodecFunc: jsonrpc.NewClientCodec,
			FailMode:        cli.Failtry,
		},
		&selector.DirectSelector{
			Network:     "tcp",
			Address:     server.Address(),
			DialTimeout: 10 * time.Second,
		},
	)
	defer client.Close()

	benchmarkMyrpcClient(client, b)
}

func startMyrpcWithMsgP() *srv.Server {
	server := srv.NewServer(srv.Server{
		ServerCodecFunc: msgpackrpc.NewServerCodec,
	})
	server.NamedRegister("Arith", new(codec.Arith))
	ln, _ := listenTCP()
	go server.ServeListener(ln)

	return server
}

func BenchmarkMyrpc_msgp(b *testing.B) {
	b.StopTimer()
	server := startMyrpcWithMsgP()
	time.Sleep(5 * time.Second) //waiting for starting server
	client := cli.NewClient(
		cli.Client{
			ClientCodecFunc: msgpackrpc.NewClientCodec,
			FailMode:        cli.Failtry,
		},
		&selector.DirectSelector{
			Network:     "tcp",
			Address:     server.Address(),
			DialTimeout: 10 * time.Second,
		},
	)
	defer client.Close()

	benchmarkMyrpcClient(client, b)
}

type GencodeArith int

func (t *GencodeArith) Mul(args *GencodeArgs, reply *GencodeReply) error {
	reply.C = args.A * args.B
	return nil
}

func (t *GencodeArith) Error(args *GencodeArgs, reply *GencodeReply) error {
	panic("ERROR")
}

func startMyrpcWithGencodec() *srv.Server {
	server := srv.NewServer(srv.Server{
		ServerCodecFunc: gencode.NewGencodeServerCodec,
	})
	server.NamedRegister("Arith", new(GencodeArith))
	ln, _ := listenTCP()
	go server.ServeListener(ln)

	return server
}

func BenchmarkMyrpc_gencodec(b *testing.B) {
	b.StopTimer()
	server := startMyrpcWithGencodec()
	time.Sleep(5 * time.Second) //waiting for starting server
	client := cli.NewClient(
		cli.Client{
			ClientCodecFunc: gencode.NewGencodeClientCodec,
			FailMode:        cli.Failtry,
		},
		&selector.DirectSelector{
			Network:     "tcp",
			Address:     server.Address(),
			DialTimeout: 10 * time.Second,
		},
	)
	defer client.Close()

	benchmarkMyrpcGencodeClient(client, b)
}

type ProtoArith int

func (t *ProtoArith) Mul(args *ProtoArgs, reply *ProtoReply) error {
	reply.C = args.A * args.B
	return nil
}

func (t *ProtoArith) Error(args *ProtoArgs, reply *ProtoReply) error {
	panic("ERROR")
}

func startMyrpcWithProtobuf() *srv.Server {
	server := srv.NewServer(srv.Server{
		ServerCodecFunc: protobuf.NewProtobufServerCodec,
	})
	server.NamedRegister("Arith", new(ProtoArith))
	ln, _ := listenTCP()
	go server.ServeListener(ln)

	return server
}

func BenchmarkMyrpc_protobuf(b *testing.B) {
	b.StopTimer()
	server := startMyrpcWithProtobuf()
	time.Sleep(5 * time.Second) //waiting for starting server
	client := cli.NewClient(
		cli.Client{
			ClientCodecFunc: protobuf.NewProtobufClientCodec,
			FailMode:        cli.Failtry,
		},
		&selector.DirectSelector{
			Network:     "tcp",
			Address:     server.Address(),
			DialTimeout: 10 * time.Second,
		},
	)
	defer client.Close()

	benchmarkMyrpcProtobufClient(client, b)
}
