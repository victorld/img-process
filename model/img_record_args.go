package model

type DoScanImgArg struct {
	DeleteShow       *bool   `json:"deleteShow" form:"deleteShow"`
	MoveFileShow     *bool   `json:"moveFileShow" form:"moveFileShow"`
	ModifyDateShow   *bool   `json:"modifyDateShow" form:"modifyDateShow"`
	RenameFileShow   *bool   `json:"renameFileShow" form:"renameFileShow"`
	Md5Show          *bool   `json:"md5Show" form:"md5Show"`
	DeleteAction     *bool   `json:"deleteAction" form:"deleteAction"`
	MoveFileAction   *bool   `json:"moveFileAction" form:"moveFileAction"`
	ModifyDateAction *bool   `json:"modifyDateAction" form:"modifyDateAction"`
	RenameFileAction *bool   `json:"renameFileAction" form:"renameFileAction"`
	StartPath        *string `json:"startPath" form:"startPath"`
	StartPathBak     *string `json:"StartPathBak" form:"StartPathBak"`
}
