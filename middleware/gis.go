package middleware

import (
	"encoding/json"
	"errors"
	"fmt"
	"img_process/model"
	"io"
	"net/http"
	"net/url"
)

var gisDatabaseCacheMap = map[string]string{}

func CreateGisDatabaseCache() {

	var gisDatabaseSearch model.GisDatabaseSearch
	list, _, err := gisDatabaseService.GetGisDatabaseInfoList(gisDatabaseSearch)
	if err != nil {

	}
	for _, isd := range list {
		gisDatabaseCacheMap[isd.LocNum] = isd.LocAddr
	}

}

func GetLocationAddressByCache(locNum string) (gitAddress string, err error) {

	if value, ok := gisDatabaseCacheMap[locNum]; ok {
		return value, nil
	} else {
		locJson, err := GetLocationAddress(locNum)
		if err == nil {
			var ret map[string]any
			json.Unmarshal([]byte(locJson), &ret)
			temp := ret["regeocode"].(map[string]any)
			if _, ok := temp["formatted_address"].(string); ok {

			} else {
				return "", errors.New("formatted_address 不是string")
			}

			locAddr := temp["formatted_address"].(string)

			var gisDatabaseDB model.GisDatabaseDB
			gisDatabaseDB.LocNum = locNum
			gisDatabaseDB.LocAddr = locAddr
			gisDatabaseDB.LocJson = locJson
			gisDatabaseService.CreateGisDatabase(&gisDatabaseDB)
			return locAddr, nil
		} else {
			return "", err
		}
	}
}

func GetLocationAddress(locNum string) (locJson string, err error) {
	// 此处填写您在控制台-应用管理-创建应用后获取的AK
	key := "96fd6b0ee12dfee568cc3489f7d7bb28"

	// 服务地址
	host := "https://restapi.amap.com"

	// 接口地址
	uri := "/v3/geocode/regeo"

	// 设置请求参数
	params := url.Values{
		"location": []string{locNum},
		"output":   []string{"json"},
		"radius":   []string{"0"},
		"key":      []string{key},
	}

	// 发起请求
	request, err := url.Parse(host + uri + "?" + params.Encode())
	if nil != err {
		fmt.Println("host error: ", err)
		return "", err
	}

	resp, err1 := http.Get(request.String())
	fmt.Println("url: ", request.String())

	if err1 != nil {
		fmt.Println("request error: ", err1)
		return "", err
	}
	body, err2 := io.ReadAll(resp.Body)
	if err2 != nil {
		fmt.Println("response error: ", err2)
	}
	resp.Body.Close()

	locJson = string(body)
	fmt.Println(locJson)

	return locJson, nil

}
