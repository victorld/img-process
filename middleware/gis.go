package middleware

import (
	"encoding/json"
	"fmt"
	"img_process/cons"
	"img_process/model"
	"img_process/tools"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type GisData struct {
	LocStreet string
	LocAddr   string
}

// LoadGisCache 创建gis database的cache快照
func LoadGisCache() (map[string]GisData, error) {
	gisDatabaseCacheMap := map[string]GisData{}
	var gisDatabaseSearch model.GisDatabaseSearch
	list, _, err := gisDatabaseService.GetGisDatabaseInfoList(gisDatabaseSearch)
	if err != nil {
		return nil, err
	}
	for _, isd := range list {
		t := GisData{LocStreet: isd.LocStreet, LocAddr: isd.LocAddr}
		gisDatabaseCacheMap[isd.LocNum] = t
	}
	tools.Logger.Info("use gisCache , cache size : ", len(gisDatabaseCacheMap))
	return gisDatabaseCacheMap, nil
}

// 线上根据经纬度查询地址json
func GetLocationAddressOnline(locNum string) (locJson string, err error) {
	// 此处填写您在控制台-应用管理-创建应用后获取的AK
	key := cons.GisKey

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

// GetGisDataFromJson 从线上返回的json数据，组合GisData结构体
func GetGisDataFromJson(locJson string) GisData {
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

	var locAddr string
	if _, ok := regeocode["formatted_address"].(string); ok {
		locAddr = regeocode["formatted_address"].(string)
	} else {
		//fmt.Println("locAddr not string : ", locJson)
	}

	return GisData{LocStreet: locStreet, LocAddr: locAddr}
}
