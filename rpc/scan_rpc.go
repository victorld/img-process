package rpc

import (
	"img_process/service"
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

func (img *Img) DoScan(args *ScanArgs, reply *string) error {
	*reply = service.DoScan(args.DeleteShow, args.MoveFileShow, args.ModifyDateShow, args.Md5Show, args.DeleteAction, args.MoveFileAction, args.ModifyDateAction)
	return nil
}
