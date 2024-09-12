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
		regeocode := ret["regeocode"].(map[string]any)
		addressComponent := regeocode["addressComponent"].(map[string]any)
		var province string
		var district string
		var township string
		var street string
		if _, ok := addressComponent["province"].(string); ok {
			province = addressComponent["province"].(string)
			if strings.Contains(province, "中华人民共和国") {
				province = ""
			}
		} else {
			//fmt.Println("province not string : ", locJson)
		}
		if _, ok := addressComponent["district"].(string); ok {
			district = addressComponent["district"].(string)
		} else {
			//fmt.Println("district not string : ", locJson)
		}
		if _, ok := addressComponent["township"].(string); ok {
			township = addressComponent["township"].(string)
		} else {
			//fmt.Println("township not string : ", locJson)
		}
		if _, ok := addressComponent["streetNumber"].(map[string]any)["street"].(string); ok {
			street = addressComponent["streetNumber"].(map[string]any)["street"].(string)
		} else {
			//fmt.Println("street not string : ", locJson)
		}

		var locStreet string
		locStreet = province + "" + district + "" + township + "" + street
		//fmt.Println("locStreet : ", locStreet)

		list[i].LocStreet = locStreet

		var locAddr string
		if _, ok := regeocode["formatted_address"].(string); ok {
			locAddr = regeocode["formatted_address"].(string)
		} else {
			//fmt.Println("locAddr not string : ", locJson)
		}

		list[i].LocAddr = locAddr

	}

	fmt.Println()

	gisDatabaseService.UpdateGisDatabaseBatch(list, 1000)

}
