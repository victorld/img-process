package dao

import (
	"img_process/model"
	"img_process/plugin/orm"
)

type ImgShootDateService struct {
}

func (imgShootDateService *ImgShootDateService) RegisterImgShootDate(imgShootDate *model.ImgShootDateDB) (err error) {
	err = orm.ImgMysqlDB.AutoMigrate(&imgShootDate)
	return err
}

// CreateImgShootDate 创建imgShootDate表记录
// Author [piexlmax](https://github.com/piexlmax)
func (imgShootDateService *ImgShootDateService) CreateImgShootDate(imgShootDate *model.ImgShootDateDB) (err error) {
	//
	err = orm.ImgMysqlDB.Create(imgShootDate).Error
	return err
}

// DeleteImgShootDate 删除imgShootDate表记录
// Author [piexlmax](https://github.com/piexlmax)
func (imgShootDateService *ImgShootDateService) DeleteImgShootDate(imgShootDate model.ImgShootDateDB) (err error) {
	err = orm.ImgMysqlDB.Delete(&imgShootDate).Error
	return err
}

// DeleteImgShootDateByIds 批量删除imgShootDate表记录
// Author [piexlmax](https://github.com/piexlmax)
func (imgShootDateService *ImgShootDateService) DeleteImgShootDateByIds(ids model.IdsReq) (err error) {
	err = orm.ImgMysqlDB.Delete(&[]model.ImgShootDateDB{}, "id in ?", ids.Ids).Error
	return err
}

// UpdateImgShootDate 更新imgShootDate表记录
// Author [piexlmax](https://github.com/piexlmax)
func (imgShootDateService *ImgShootDateService) UpdateImgShootDate(imgShootDate model.ImgShootDateDB) (err error) {
	err = orm.ImgMysqlDB.Save(&imgShootDate).Error
	return err
}

// GetImgShootDate 根据id获取imgShootDate表记录
// Author [piexlmax](https://github.com/piexlmax)
func (imgShootDateService *ImgShootDateService) GetImgShootDate(id uint) (imgShootDate model.ImgShootDateDB, err error) {
	err = orm.ImgMysqlDB.Where("id = ?", id).First(&imgShootDate).Error
	return
}

// GetImgShootDateInfoList 分页获取imgShootDate表记录
// Author [piexlmax](https://github.com/piexlmax)
func (imgShootDateService *ImgShootDateService) GetImgShootDateInfoList(info model.ImgShootDateSearch) (list []model.ImgShootDateDB, total int64, err error) {
	limit := info.PageSize
	offset := info.PageSize * (info.Page - 1)
	// 创建db
	db := orm.ImgMysqlDB.Model(&model.ImgShootDateDB{})
	var imgShootDates []model.ImgShootDateDB
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

	err = db.Find(&imgShootDates).Error
	return imgShootDates, total, err
}
