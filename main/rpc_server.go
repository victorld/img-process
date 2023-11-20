package main

import (
	"fmt"
	img_rpc "img_process/rpc"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
)

func main() {
	img := new(img_rpc.Img)
	rpc.Register(img) // 注册RPC服务
	l, e := net.Listen("tcp", ":9091")
	if e != nil {
		fmt.Errorf("net Listen error")
	}
	fmt.Println("started server on 9091")
	for {
		conn, _ := l.Accept()
		// 使用JSON协议
		rpc.ServeCodec(jsonrpc.NewServerCodec(conn))
	}
}
