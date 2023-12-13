package rpc

import (
	"fmt"
	"img_process/model"
	"img_process/service"
)

type Img struct{}

func (img *Img) DoScan(scanArgs *model.DoScanImgArg, reply *string) error {
	fmt.Println("received call")

	ret, err := service.DoScan(*scanArgs)
	if err != nil {
		*reply = ""
		return err
	} else {
		*reply = ret
	}
	return nil
}
