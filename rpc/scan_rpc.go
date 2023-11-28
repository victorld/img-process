package rpc

import (
	"fmt"
	"img_process/service"
)

type Img struct{}

func (img *Img) DoScan(scanArgs *service.ScanArgs, reply *string) error {
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
