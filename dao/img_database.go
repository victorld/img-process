package dao

import (
	"img_process/model"
	"img_process/plugin/orm"
)

type ImgDatabaseService struct {
}

func (imgDatabaseService *ImgDatabaseService) RegisterImgDatabase(imgDatabase *model.ImgDatabaseDB) (err error) {
	err = orm.ImgMysqlDB.AutoMigrate(&imgDatabase)
	return err
}

// CreateImgDatabase 创建imgDatabase表记录
// Author [piexlmax](https://github.com/piexlmax)
func (imgDatabaseService *ImgDatabaseService) CreateImgDatabase(imgDatabase *model.ImgDatabaseDB) (err error) {
	//
	err = orm.ImgMysqlDB.Create(imgDatabase).Error
	return err
}

// CreateImgDatabaseBatch 批量创建imgDatabase表记录
func (imgDatabaseService *ImgDatabaseService) CreateImgDatabaseBatch(imgDatabaseList []*model.ImgDatabaseDB) (err error) {
	//
	err = orm.ImgMysqlDB.CreateInBatches(imgDatabaseList, 5000).Error
	return err
}

func (imgDatabaseService *ImgDatabaseService) TruncateImgDatabase() (err error) {
	err = orm.ImgMysqlDB.Exec("truncate table img_database").Error
	return err
}

// DeleteImgDatabase 删除imgDatabase表记录
// Author [piexlmax](https://github.com/piexlmax)
func (imgDatabaseService *ImgDatabaseService) DeleteImgDatabase(imgDatabase model.ImgDatabaseDB) (err error) {
	err = orm.ImgMysqlDB.Delete(&imgDatabase).Error
	return err
}

// DeleteImgDatabaseByIds 批量删除imgDatabase表记录
// Author [piexlmax](https://github.com/piexlmax)
func (imgDatabaseService *ImgDatabaseService) DeleteImgDatabaseByIds(ids model.IdsReq) (err error) {
	err = orm.ImgMysqlDB.Delete(&[]model.ImgDatabaseDB{}, "id in ?", ids.Ids).Error
	return err
}

// UpdateImgDatabase 更新imgDatabase表记录
// Author [piexlmax](https://github.com/piexlmax)
func (imgDatabaseService *ImgDatabaseService) UpdateImgDatabase(imgDatabase model.ImgDatabaseDB) (err error) {
	err = orm.ImgMysqlDB.Save(&imgDatabase).Error
	return err
}

// GetImgDatabase 根据id获取imgDatabase表记录
// Author [piexlmax](https://github.com/piexlmax)
func (imgDatabaseService *ImgDatabaseService) GetImgDatabase(id uint) (imgDatabase model.ImgDatabaseDB, err error) {
	err = orm.ImgMysqlDB.Where("id = ?", id).First(&imgDatabase).Error
	return
}

// GetImgDatabaseInfoList 分页获取imgDatabase表记录
// Author [piexlmax](https://github.com/piexlmax)
func (imgDatabaseService *ImgDatabaseService) GetImgDatabaseInfoList(info model.ImgDatabaseSearch) (list []model.ImgDatabaseDB, total int64, err error) {
	limit := info.PageSize
	offset := info.PageSize * (info.Page - 1)
	// 创建db
	db := orm.ImgMysqlDB.Model(&model.ImgDatabaseDB{})
	var imgDatabases []model.ImgDatabaseDB
	// 如果有条件搜索 下方会自动创建搜索语句
	db = db.Select("img_key", "shoot_date", "loc_num", "loc_addr", "state")
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

	err = db.Find(&imgDatabases).Error
	return imgDatabases, total, err
}
