package middleware

import (
	"img_process/model"
	"img_process/tools"
)

var ImgCacheMap map[string]ImgCacheData
var ImgCacheMapBak map[string]ImgCacheData //复制一份ImgCacheMap，一次遍历匹配上的删掉ImgCacheMapBak里的key，剩下没匹配上的做清理操作（单独构建是因为直接操作ImgCacheMap删除有多线程问题）

//var GpsCacheMap map[string]string

type ImgCacheData struct {
	LocStreet string
	ShootDate string
}

// CreateImgCache 构建ImgCache
func CreateImgCache() {

	ImgCacheMap = map[string]ImgCacheData{}
	ImgCacheMapBak = map[string]ImgCacheData{}

	var imgDatabaseSearch model.ImgDatabaseSearch
	list, _, err := imgDatabaseService.GetImgDatabaseInfoList(imgDatabaseSearch)
	if err != nil {

	}
	for _, isd := range list {
		ImgCacheMap[isd.ImgKey] = ImgCacheData{LocStreet: isd.LocStreet, ShootDate: isd.ShootDate}
		ImgCacheMapBak[isd.ImgKey] = ImgCacheData{LocStreet: isd.LocStreet, ShootDate: isd.ShootDate}
		/*if isd.LocAddr != "" {
			GpsCacheMap[isd.ImgKey] = isd.LocAddr
		}*/
	}

	tools.Logger.Info("use imageCache , cache size : ", len(ImgCacheMap))

}
