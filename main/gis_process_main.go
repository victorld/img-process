package main

import (
	"encoding/json"
	"fmt"
	"img_process/cons"
	"img_process/dao"
	"img_process/model"
	"img_process/plugin/orm"
	"img_process/tools"
	"strings"
)

var gisDatabaseService = dao.GisDatabaseService{}

func main() {
	tools.InitLogger()
	tools.InitViper()
	cons.InitConst()
	orm.InitMysql()

	var gisDatabaseSearch model.GisDatabaseSearch
	list, _, err := gisDatabaseService.GetGisDatabaseInfoList(gisDatabaseSearch)
	if err != nil {

	}
	for i := range list {

		locJson := list[i].LocJson

		var ret map[string]any
		json.Unmarshal([]byte(locJson), &ret)
		temp := (ret["regeocode"].(map[string]any))["addressComponent"].(map[string]any)
		var province string
		var district string
		var township string
		var street string
		if _, ok := temp["province"].(string); ok {
			province = temp["province"].(string)
			if strings.Contains(province, "中华人民共和国") {
				province = ""
			}
		} else {
			//fmt.Println("province not string : ", locJson)
		}
		if _, ok := temp["district"].(string); ok {
			district = temp["district"].(string)
		} else {
			//fmt.Println("district not string : ", locJson)
		}
		if _, ok := temp["township"].(string); ok {
			township = temp["township"].(string)
		} else {
			//fmt.Println("township not string : ", locJson)
		}
		if _, ok := temp["streetNumber"].(map[string]any)["street"].(string); ok {
			street = temp["streetNumber"].(map[string]any)["street"].(string)
		} else {
			//fmt.Println("street not string : ", locJson)
		}

		var locStreet string
		locStreet = province + "" + district + "" + township + "" + street
		//fmt.Println("locStreet : ", locStreet)

		list[i].LocStreet = locStreet

	}

	fmt.Println()

	gisDatabaseService.UpdateGisDatabaseBatch(list, 1000)

}
