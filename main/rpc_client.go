package main

import (
	"img_process/model"
	"img_process/tools"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
)

// rpc客户端
func RpcClient() {

	tools.InitLogger()

	var startPath = ""
	var startPathBak = ""

	var deleteShow = true      //是否统计并显示非法文件和空目录
	var moveFileShow = true    //是否统计并显示需要移动目录的文件
	var modifyDateShow = false //是否统计并显示需要修改日期的文件
	var renameFileShow = false //是否统计并显示需要修改名称的文件
	var md5Show = true         //是否统计并显示重复文件

	var deleteAction = false     //是否操作删除非法文件和空目录
	var moveFileAction = false   //是否操作需要移动目录的文件
	var modifyDateAction = false //是否操作修改日期的文件
	var renameFileAction = false //是否操作修改文件名称

	scanArgs := model.DoScanImgArg{DeleteShow: &deleteShow, MoveFileShow: &moveFileShow, ModifyDateShow: &modifyDateShow, RenameFileShow: &renameFileShow, Md5Show: &md5Show, DeleteAction: &deleteAction, MoveFileAction: &moveFileAction, ModifyDateAction: &modifyDateAction, RenameFileAction: &renameFileAction, StartPath: &startPath, StartPathBak: &startPathBak}
	tools.Logger.Info("DoScanImg rpc args : " + tools.MarshalJsonToString(scanArgs))

	// 建立TCP连接
	conn, err := net.Dial("tcp", "127.0.0.1:9091")
	if err != nil {
		tools.Logger.Info("dialing:", err)
	}
	// 使用JSON协议
	client := rpc.NewClientWithCodec(jsonrpc.NewClientCodec(conn))
	// 同步调用

	var reply string
	err = client.Call("Img.DoScan", scanArgs, &reply)
	if err != nil {
		tools.Logger.Info("img_rpc call error:", err)
	}
	tools.Logger.Info("img_rpc call ret : ", reply)

	// 异步调用
	//var reply2 int
	//divCall := client.Go("img_rpc.DoScan", args, &reply2, nil)
	//replyCall := <-divCall.Done // 接收调用结果
	//fmt.Println(replyCall.Error)
	//fmt.Println(reply2)
}
