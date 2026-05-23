package dao

import (
	"img_process/cons"
	"img_process/model"
	"img_process/plugin/orm"
	"path/filepath"
	"sort"
	"strings"

	"gorm.io/gorm"
)

type ImgDatabaseService struct {
}

const fileAnalysisListOrder = "SUBSTRING_INDEX(img_key, '|', 1) asc, CASE WHEN shoot_date IS NULL OR shoot_date = '' THEN 1 ELSE 0 END asc, REPLACE(LEFT(shoot_date, 19), ':', '-') asc, id asc"

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
	err = orm.ImgMysqlDB.CreateInBatches(imgDatabaseList, cons.IDInsertBatchSize).Error
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

// DeleteImgDatabaseByImgKey 批量删除imgDatabase表记录
func (imgDatabaseService *ImgDatabaseService) DeleteImgDatabaseByImgKey(imgKeys []string) (err error) {
	err = orm.ImgMysqlDB.Delete(&[]model.ImgDatabaseDB{}, "img_key in ?", imgKeys).Error
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
	db = db.Select("img_key", "shoot_date", "loc_num", "loc_addr", "loc_street", "state")
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

// GetImgDatabaseInfoCount 分页获取imgDatabase表记录数
func (imgDatabaseService *ImgDatabaseService) GetImgDatabaseInfoCount(info model.ImgDatabaseSearch) (total int64, err error) {

	// 创建db
	db := orm.ImgMysqlDB.Model(&model.ImgDatabaseDB{})
	// 如果有条件搜索 下方会自动创建搜索语句
	if info.StartCreatedAt != nil && info.EndCreatedAt != nil {
		db = db.Where("created_at BETWEEN ? AND ?", info.StartCreatedAt, info.EndCreatedAt)
	}
	err = db.Count(&total).Error
	if err != nil {
		return
	}

	return total, err
}

func (imgDatabaseService *ImgDatabaseService) GetFileAnalysis(info model.FileAnalysisSearch) (model.FileAnalysisResult, error) {
	return imgDatabaseService.getFileAnalysisWithDB(info)
}

func (imgDatabaseService *ImgDatabaseService) getFileAnalysisWithDB(info model.FileAnalysisSearch) (model.FileAnalysisResult, error) {
	result := model.FileAnalysisResult{}
	summary, err := fileAnalysisSummary(info)
	if err != nil {
		return result, err
	}
	result.Summary = summary

	var rows []model.ImgDatabaseDB
	if err := buildFileAnalysisDB(orm.ImgMysqlDB.Model(&model.ImgDatabaseDB{}), info).
		Select("id", "img_key", "shoot_date", "loc_num", "loc_addr", "loc_street", "remark", "updated_at").
		Find(&rows).Error; err != nil {
		return result, err
	}
	result.Total = int64(len(rows))
	result.YearStats = fileAnalysisYearStats(rows)
	result.SuffixStats = fileAnalysisSuffixStats(rows)

	page := info.Page
	if page <= 0 {
		page = 1
	}
	pageSize := info.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := pageSize * (page - 1)
	pagedDB := buildFileAnalysisDB(orm.ImgMysqlDB.Model(&model.ImgDatabaseDB{}), info).
		Select("id", "img_key", "shoot_date", "loc_num", "loc_addr", "loc_street", "remark", "updated_at").
		Order(fileAnalysisListOrder).
		Limit(pageSize).
		Offset(offset)
	var list []model.ImgDatabaseDB
	if err := pagedDB.Find(&list).Error; err != nil {
		return result, err
	}
	result.List = make([]model.FileAnalysisItem, 0, len(list))
	for _, item := range list {
		result.List = append(result.List, toFileAnalysisItem(item))
	}
	return result, nil
}

func buildFileAnalysisDB(db *gorm.DB, info model.FileAnalysisSearch) *gorm.DB {
	if keyword := strings.TrimSpace(info.FileKey); keyword != "" {
		db = db.Where("img_key LIKE ?", "%"+keyword+"%")
	}
	if keyword := strings.TrimSpace(info.LocAddrKeyword); keyword != "" {
		db = db.Where("loc_addr LIKE ?", "%"+keyword+"%")
	}
	switch strings.TrimSpace(info.ShootDateStatus) {
	case "present":
		db = db.Where("shoot_date <> ''")
	case "missing":
		db = db.Where("(shoot_date = '' OR shoot_date IS NULL)")
	}
	switch strings.TrimSpace(info.GeoStatus) {
	case "full":
		db = db.Where("loc_num <> '' AND (loc_addr <> '' OR loc_street <> '')")
	case "loc_num":
		db = db.Where("loc_num <> ''")
	case "missing":
		db = db.Where("(loc_num = '' OR loc_num IS NULL) AND (loc_addr = '' OR loc_addr IS NULL) AND (loc_street = '' OR loc_street IS NULL)")
	}
	if start := normalizeShootDateFilter(info.ShootDateStart, false); start != "" {
		db = db.Where("REPLACE(LEFT(shoot_date, 10), ':', '-') >= ?", start)
	}
	if end := normalizeShootDateFilter(info.ShootDateEnd, true); end != "" {
		db = db.Where("REPLACE(LEFT(shoot_date, 10), ':', '-') <= ?", end)
	}
	return db
}

func fileAnalysisSummary(info model.FileAnalysisSearch) (model.FileAnalysisSummary, error) {
	var summary model.FileAnalysisSummary
	if err := buildFileAnalysisDB(orm.ImgMysqlDB.Model(&model.ImgDatabaseDB{}), info).Count(&summary.TotalCount).Error; err != nil {
		return summary, err
	}
	if err := buildFileAnalysisDB(orm.ImgMysqlDB.Model(&model.ImgDatabaseDB{}), info).Where("shoot_date <> ''").Count(&summary.WithShootDateCount).Error; err != nil {
		return summary, err
	}
	if err := buildFileAnalysisDB(orm.ImgMysqlDB.Model(&model.ImgDatabaseDB{}), info).Where("loc_num <> ''").Count(&summary.WithLocNumCount).Error; err != nil {
		return summary, err
	}
	if err := buildFileAnalysisDB(orm.ImgMysqlDB.Model(&model.ImgDatabaseDB{}), info).Where("(loc_addr <> '' OR loc_street <> '')").Count(&summary.WithLocAddrCount).Error; err != nil {
		return summary, err
	}
	summary.ShootDateCoverage = percent(summary.WithShootDateCount, summary.TotalCount)
	summary.LocNumCoverage = percent(summary.WithLocNumCount, summary.TotalCount)
	summary.LocAddrCoverage = percent(summary.WithLocAddrCount, summary.TotalCount)
	return summary, nil
}

func fileAnalysisYearStats(rows []model.ImgDatabaseDB) []model.FileAnalysisStatItem {
	counts := map[string]int64{}
	for _, row := range rows {
		dirDate, _ := splitFileAnalysisImgKey(row.ImgKey)
		if hasYearPrefix(dirDate) {
			counts[dirDate[:4]]++
		}
	}
	items := statItemsFromCounts(counts, int64(len(rows)))
	sort.Slice(items, func(i, j int) bool {
		return items[i].Key > items[j].Key
	})
	return items
}

func hasYearPrefix(value string) bool {
	if len(value) < 4 {
		return false
	}
	for _, char := range value[:4] {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

func fileAnalysisSuffixStats(rows []model.ImgDatabaseDB) []model.FileAnalysisStatItem {
	counts := map[string]int64{}
	for _, row := range rows {
		_, fileName := splitFileAnalysisImgKey(row.ImgKey)
		suffix := strings.ToLower(filepath.Ext(fileName))
		if suffix == "" {
			suffix = "无后缀"
		}
		counts[suffix]++
	}
	return statItemsFromCounts(counts, int64(len(rows)))
}

func statItemsFromCounts(counts map[string]int64, total int64) []model.FileAnalysisStatItem {
	items := make([]model.FileAnalysisStatItem, 0, len(counts))
	for key, count := range counts {
		items = append(items, model.FileAnalysisStatItem{Key: key, Count: count, Percent: percent(count, total)})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Count == items[j].Count {
			return items[i].Key > items[j].Key
		}
		return items[i].Count > items[j].Count
	})
	return items
}

func toFileAnalysisItem(item model.ImgDatabaseDB) model.FileAnalysisItem {
	dirDate, fileName := splitFileAnalysisImgKey(item.ImgKey)
	return model.FileAnalysisItem{
		ID:        item.ID,
		ImgKey:    item.ImgKey,
		DirDate:   dirDate,
		FileName:  fileName,
		Suffix:    strings.ToLower(filepath.Ext(fileName)),
		ShootDate: item.ShootDate,
		LocNum:    item.LocNum,
		LocStreet: item.LocStreet,
		LocAddr:   item.LocAddr,
		Remark:    item.Remark,
		UpdatedAt: item.UpdatedAt,
	}
}

func splitFileAnalysisImgKey(imgKey string) (string, string) {
	parts := strings.SplitN(imgKey, "|", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", imgKey
}

func normalizeShootDateFilter(value string, end bool) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, ":", "-"))
	if len(value) >= 10 {
		return value[:10]
	}
	if len(value) == 7 {
		if end {
			return value + "-31"
		}
		return value + "-01"
	}
	if len(value) == 4 {
		if end {
			return value + "-12-31"
		}
		return value + "-01-01"
	}
	return ""
}

func percent(count int64, total int64) float64 {
	if total == 0 {
		return 0
	}
	return float64(count) * 100 / float64(total)
}
