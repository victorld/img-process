package dao

import (
	"img_process/model"
	"img_process/plugin/orm"
)

type GisDatabaseService struct {
}

func (gisDatabaseService *GisDatabaseService) RegisterGisDatabase(gisDatabase *model.GisDatabaseDB) (err error) {
	err = orm.ImgMysqlDB.AutoMigrate(&gisDatabase)
	return err
}

// CreateGisDatabase 创建gisDatabase表记录
// Author [piexlmax](https://github.com/piexlmax)
func (gisDatabaseService *GisDatabaseService) CreateGisDatabase(gisDatabase *model.GisDatabaseDB) (err error) {
	//
	err = orm.ImgMysqlDB.Create(gisDatabase).Error
	return err
}

/*func (gisDatabaseService *GisDatabaseService) TruncateGisDatabase() (err error) {
	err = orm.ImgMysqlDB.Exec("truncate table gis_database").Error
	return err
}*/

// DeleteGisDatabase 删除gisDatabase表记录
// Author [piexlmax](https://github.com/piexlmax)
func (gisDatabaseService *GisDatabaseService) DeleteGisDatabase(gisDatabase model.GisDatabaseDB) (err error) {
	err = orm.ImgMysqlDB.Delete(&gisDatabase).Error
	return err
}

// DeleteGisDatabaseByIds 批量删除gisDatabase表记录
// Author [piexlmax](https://github.com/piexlmax)
func (gisDatabaseService *GisDatabaseService) DeleteGisDatabaseByIds(ids model.IdsReq) (err error) {
	err = orm.ImgMysqlDB.Delete(&[]model.GisDatabaseDB{}, "id in ?", ids.Ids).Error
	return err
}

// UpdateGisDatabase 更新gisDatabase表记录
// Author [piexlmax](https://github.com/piexlmax)
func (gisDatabaseService *GisDatabaseService) UpdateGisDatabase(gisDatabase model.GisDatabaseDB) (err error) {
	err = orm.ImgMysqlDB.Save(&gisDatabase).Error
	return err
}
func (gisDatabaseService *GisDatabaseService) UpdateGisDatabaseBatch(gisDatabaseList []model.GisDatabaseDB, batchSize int) (err error) {
	for i := 0; i < len(gisDatabaseList); i += batchSize {
		end := i + batchSize
		if end > len(gisDatabaseList) {
			end = len(gisDatabaseList)
		}
		dbs := gisDatabaseList[i:end]
		//fmt.Println(dbs)
		updateGisDatabaseBatchCommit(dbs)
	}

	return nil
}

func updateGisDatabaseBatchCommit(gisDatabaseList []model.GisDatabaseDB) (err error) {

	tx := orm.ImgMysqlDB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	for _, gisDatabase := range gisDatabaseList {
		if err := tx.Save(&gisDatabase).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error

}

// GetGisDatabase 根据id获取gisDatabase表记录
// Author [piexlmax](https://github.com/piexlmax)
func (gisDatabaseService *GisDatabaseService) GetGisDatabase(id uint) (gisDatabase model.GisDatabaseDB, err error) {
	err = orm.ImgMysqlDB.Where("id = ?", id).First(&gisDatabase).Error
	return
}

// GetGisDatabase 根据id获取gisDatabase表记录
// Author [piexlmax](https://github.com/piexlmax)
func (gisDatabaseService *GisDatabaseService) GetGisDatabaseByLocNum(locNum string) (gisDatabase model.GisDatabaseDB, err error) {
	err = orm.ImgMysqlDB.Where("loc_num = ?", locNum).First(&gisDatabase).Error
	return
}

// GetGisDatabaseInfoList 分页获取gisDatabase表记录
// Author [piexlmax](https://github.com/piexlmax)
func (gisDatabaseService *GisDatabaseService) GetGisDatabaseInfoList(info model.GisDatabaseSearch) (list []model.GisDatabaseDB, total int64, err error) {
	limit := info.PageSize
	offset := info.PageSize * (info.Page - 1)
	// 创建db
	db := orm.ImgMysqlDB.Model(&model.GisDatabaseDB{})
	var gisDatabases []model.GisDatabaseDB
	// 如果有条件搜索 下方会自动创建搜索语句
	if info.StartCreatedAt != nil && info.EndCreatedAt != nil {
		db = db.Where("created_at BETWEEN ? AND ?", info.StartCreatedAt, info.EndCreatedAt)
	}
	err = db.Count(&total).Error
	if err != nil {
		return
	}

	if limit != 0 {
		db = db.Limit(limit).Offset(offset)
	}

	err = db.Find(&gisDatabases).Error
	return gisDatabases, total, err
}
