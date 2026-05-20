package bootstrap

import (
	"fmt"
	"img_process/cons"
	"img_process/plugin/orm"
	"img_process/tools"
	"time"
)

const (
	dbInitRetryInterval = 5 * time.Second
	dbInitRetryTimeout  = 2 * time.Minute
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

	if err := initMysqlWithRetry(); err != nil {
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

func initMysqlWithRetry() error {
	deadline := time.Now().Add(dbInitRetryTimeout)
	var lastErr error
	for attempt := 1; ; attempt++ {
		if err := orm.InitMysql(); err != nil {
			lastErr = err
			if time.Now().Add(dbInitRetryInterval).After(deadline) {
				return fmt.Errorf("init mysql failed after %s: %w", dbInitRetryTimeout, lastErr)
			}
			tools.Logger.Warn("init mysql failed, retrying : ", err)
			time.Sleep(dbInitRetryInterval)
			continue
		}
		if attempt > 1 {
			tools.Logger.Info("init mysql retry success")
		}
		return nil
	}
}
