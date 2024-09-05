package main

import (
	"img_process/cons"
	"img_process/plugin/orm"
	img_rpc "img_process/rpc"
	"img_process/tools"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
)

func main() {

	tools.InitLogger()
	tools.InitViper()
	cons.InitConst()
	orm.InitMysql()

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
