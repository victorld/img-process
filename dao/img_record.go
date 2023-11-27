package dao

import (
	"img_process/model"
	"img_process/tools"
)

type ImgRecordService struct {
}

func (imgRecordService *ImgRecordService) RegisterImgRecord(imgRecord *model.ImgRecordDB) (err error) {
	err = tools.ImgMysqlDB.AutoMigrate(&imgRecord)
	return err
}

// CreateImgRecord 创建imgRecord表记录
// Author [piexlmax](https://github.com/piexlmax)
func (imgRecordService *ImgRecordService) CreateImgRecord(imgRecord *model.ImgRecordDB) (err error) {
	//
	err = tools.ImgMysqlDB.Create(imgRecord).Error
	return err
}

// DeleteImgRecord 删除imgRecord表记录
// Author [piexlmax](https://github.com/piexlmax)
func (imgRecordService *ImgRecordService) DeleteImgRecord(imgRecord model.ImgRecordDB) (err error) {
	err = tools.ImgMysqlDB.Delete(&imgRecord).Error
	return err
}

// DeleteImgRecordByIds 批量删除imgRecord表记录
// Author [piexlmax](https://github.com/piexlmax)
func (imgRecordService *ImgRecordService) DeleteImgRecordByIds(ids model.IdsReq) (err error) {
	err = tools.ImgMysqlDB.Delete(&[]model.ImgRecordDB{}, "id in ?", ids.Ids).Error
	return err
}

// UpdateImgRecord 更新imgRecord表记录
// Author [piexlmax](https://github.com/piexlmax)
func (imgRecordService *ImgRecordService) UpdateImgRecord(imgRecord model.ImgRecordDB) (err error) {
	err = tools.ImgMysqlDB.Save(&imgRecord).Error
	return err
}

// GetImgRecord 根据id获取imgRecord表记录
// Author [piexlmax](https://github.com/piexlmax)
func (imgRecordService *ImgRecordService) GetImgRecord(id uint) (imgRecord model.ImgRecordDB, err error) {
	err = tools.ImgMysqlDB.Where("id = ?", id).First(&imgRecord).Error
	return
}

// GetImgRecordInfoList 分页获取imgRecord表记录
// Author [piexlmax](https://github.com/piexlmax)
func (imgRecordService *ImgRecordService) GetImgRecordInfoList(info model.ImgRecordSearch) (list []model.ImgRecordDB, total int64, err error) {
	limit := info.PageSize
	offset := info.PageSize * (info.Page - 1)
	// 创建db
	db := tools.ImgMysqlDB.Model(&model.ImgRecordDB{})
	var imgRecords []model.ImgRecordDB
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

	err = db.Find(&imgRecords).Error
	return imgRecords, total, err
}
