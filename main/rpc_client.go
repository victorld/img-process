package main

import (
	"fmt"
	"img_process/model"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
)

func main() {

	var deleteShow = true     //是否统计并显示非法文件和空目录
	var moveFileShow = true   //是否统计并显示需要移动目录的文件
	var modifyDateShow = true //是否统计并显示需要修改日期的文件
	var md5Show = true        //是否统计并显示重复文件

	var deleteAction = false     //是否操作删除非法文件和空目录
	var moveFileAction = false   //是否操作需要移动目录的文件
	var modifyDateAction = false //是否操作修改日期的文件
	var startpath = ""
	var startpathBak = ""

	// 建立TCP连接
	conn, err := net.Dial("tcp", "127.0.0.1:9091")
	if err != nil {
		fmt.Println("dialing:", err)
	}
	// 使用JSON协议
	client := rpc.NewClientWithCodec(jsonrpc.NewClientCodec(conn))
	// 同步调用
	args := &model.DoScanImgArg{&deleteShow, &moveFileShow, &modifyDateShow, &md5Show, &deleteAction, &moveFileAction, &modifyDateAction, &startpath, &startpathBak}
	fmt.Println("img_rpc call args :", *args)
	var reply string
	err = client.Call("Img.DoScan", args, &reply)
	if err != nil {
		fmt.Println("img_rpc call error:", err)
	}
	fmt.Println("img_rpc call ret : ", reply)

	// 异步调用
	//var reply2 int
	//divCall := client.Go("img_rpc.DoScan", args, &reply2, nil)
	//replyCall := <-divCall.Done // 接收调用结果
	//fmt.Println(replyCall.Error)
	//fmt.Println(reply2)
}
