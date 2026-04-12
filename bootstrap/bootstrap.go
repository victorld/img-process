package bootstrap

import (
	"img_process/cons"
	"img_process/plugin/orm"
	"img_process/tools"
)

func InitApp(requireDB bool) (func(), error) {
	tools.InitLogger()
	if err := tools.InitViper(); err != nil {
		return nil, err
	}
	cons.InitConst()

	closeFn := func() {}
	if !requireDB {
		return closeFn, nil
	}

	if err := orm.InitMysql(); err != nil {
		return nil, err
	}

	if orm.ImgMysqlDB != nil {
		db, err := orm.ImgMysqlDB.DB()
		if err == nil {
			closeFn = func() {
				_ = db.Close()
			}
		}
	}

	return closeFn, nil
}
