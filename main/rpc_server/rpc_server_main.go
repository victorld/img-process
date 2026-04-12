package main

import (
	"img_process/bootstrap"
	img_rpc "img_process/rpc"
	"img_process/tools"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
)

// rpc服务端
func main() {
	closeFn, err := bootstrap.InitApp(true)
	if err != nil {
		tools.Logger.Fatal("bootstrap init error : ", err)
	}
	defer closeFn()

	img := new(img_rpc.Img)
	rpc.Register(img) // 注册RPC服务
	l, e := net.Listen("tcp", ":9091")
	if e != nil {
		tools.Logger.Info("net Listen error")
	}
	tools.Logger.Info("started server on 9091")
	for {
		conn, _ := l.Accept()
		// 使用JSON协议
		rpc.ServeCodec(jsonrpc.NewServerCodec(conn))
	}
}
