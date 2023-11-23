package rpc

import (
	"img_process/service"
	"img_process/tools"
)

type ScanArgs struct {
	DeleteShow       bool
	MoveFileShow     bool
	ModifyDateShow   bool
	Md5Show          bool
	DeleteAction     bool
	MoveFileAction   bool
	ModifyDateAction bool
}

type Img struct{}

var sl = tools.InitLogger()

func (img *Img) DoScan(args *ScanArgs, reply *string) error {
	sl.Info("received call")
	ret, err := service.DoScan(args.DeleteShow, args.MoveFileShow, args.ModifyDateShow, args.Md5Show, args.DeleteAction, args.MoveFileAction, args.ModifyDateAction)
	if err != nil {
		*reply = ""
		return err
	} else {
		*reply = ret
	}
	return nil
}
