package middleware

import (
	"img_process/model"
	"img_process/tools"
)

type ImgCacheData struct {
	LocStreet string
	ShootDate string
}

// LoadImgCache 构建ImgCache快照
func LoadImgCache() (map[string]ImgCacheData, error) {
	imgCacheMap := map[string]ImgCacheData{}
	var imgDatabaseSearch model.ImgDatabaseSearch
	list, _, err := imgDatabaseService.GetImgDatabaseInfoList(imgDatabaseSearch)
	if err != nil {
		return nil, err
	}
	for _, isd := range list {
		imgCacheMap[isd.ImgKey] = ImgCacheData{LocStreet: isd.LocStreet, ShootDate: isd.ShootDate}
	}

	tools.Logger.Info("use imageCache , cache size : ", len(imgCacheMap))
	return imgCacheMap, nil
}
