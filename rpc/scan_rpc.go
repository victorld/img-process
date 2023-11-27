package rpc

import (
	"img_process/service"
	"img_process/tools"
)

type Img struct{}

var sl = tools.InitLogger()

func (img *Img) DoScan(scanArgs *service.ScanArgs, reply *string) error {
	sl.Info("received call")

	ret, err := service.DoScan(*scanArgs)
	if err != nil {
		*reply = ""
		return err
	} else {
		*reply = ret
	}
	return nil
}
