package model

type DoScanImgArg struct {
	DeleteShow       *bool   `json:"deleteShow" form:"deleteShow"`
	MoveFileShow     *bool   `json:"moveFileShow" form:"moveFileShow"`
	ModifyDateShow   *bool   `json:"modifyDateShow" form:"modifyDateShow"`
	RenameShow       *bool   `json:"renameShow" form:"renameShow"`
	Md5Show          *bool   `json:"md5Show" form:"md5Show"`
	DeleteAction     *bool   `json:"deleteAction" form:"deleteAction"`
	MoveFileAction   *bool   `json:"moveFileAction" form:"moveFileAction"`
	ModifyDateAction *bool   `json:"modifyDateAction" form:"modifyDateAction"`
	RenameAction     *bool   `json:"renameAction" form:"renameAction"`
	StartPath        *string `json:"startPath" form:"startPath"`
	StartPathBak     *string `json:"StartPathBak" form:"StartPathBak"`
}
