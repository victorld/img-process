package middleware

import (
	"img_process/model"
	"img_process/tools"
)

var ShootDateCacheMap map[string]string

//var GpsCacheMap map[string]string

func CreateImgCache() {

	ShootDateCacheMap = map[string]string{}
	var imgDatabaseSearch model.ImgDatabaseSearch
	list, _, err := imgDatabaseService.GetImgDatabaseInfoList(imgDatabaseSearch)
	if err != nil {

	}
	for _, isd := range list {
		ShootDateCacheMap[isd.ImgKey] = isd.ShootDate
		/*if isd.LocAddr != "" {
			GpsCacheMap[isd.ImgKey] = isd.LocAddr
		}*/
	}

	tools.Logger.Info("use imageCache , cache size : ", len(ShootDateCacheMap))

}
