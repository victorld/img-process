package model

type DoScanImgArg struct {
	DeleteShow       bool   `json:"deleteShow" form:"deleteShow,default=true"`
	MoveFileShow     bool   `json:"moveFileShow" form:"moveFileShow,default=true"`
	ModifyDateShow   bool   `json:"modifyDateShow" form:"modifyDateShow,default=true"`
	Md5Show          bool   `json:"md5Show" form:"md5Show,default=true"`
	DeleteAction     bool   `json:"deleteAction" form:"deleteAction,default=false"`
	MoveFileAction   bool   `json:"moveFileAction" form:"moveFileAction,default=false"`
	ModifyDateAction bool   `json:"modifyDateAction" form:"modifyDateAction,default=false"`
	StartPath        string `json:"startPath" form:"startPath"`
}
