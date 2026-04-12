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
	"time"
)

type GisData struct {
	LocStreet string
	LocAddr   string
}

var (
	amapBaseURL = "https://restapi.amap.com"
	amapClient  = &http.Client{Timeout: 5 * time.Second}
)

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
func GetLocationAddressOnline(locNum string) (string, error) {
	requestURL, err := buildAmapRequestURL(locNum)
	if err != nil {
		tools.Logger.Error("build gis request error : ", err)
		return "", err
	}

	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := amapClient.Do(req)
	if err != nil {
		tools.Logger.Warn("gis request failed for locNum ", locNum, " : ", err)
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("gis request status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	locJSON := string(body)
	if _, err := GetGisDataFromJson(locJSON); err != nil {
		return "", err
	}

	return locJSON, nil
}

func buildAmapRequestURL(locNum string) (string, error) {
	params := url.Values{
		"location": []string{locNum},
		"output":   []string{"json"},
		"radius":   []string{"0"},
		"key":      []string{cons.GisKey},
	}

	request, err := url.Parse(amapBaseURL + "/v3/geocode/regeo?" + params.Encode())
	if err != nil {
		return "", err
	}

	return request.String(), nil
}

// GetGisDataFromJson 从线上返回的json数据，组合GisData结构体
func GetGisDataFromJson(locJson string) (GisData, error) {
	var ret map[string]any
	if err := json.Unmarshal([]byte(locJson), &ret); err != nil {
		return GisData{}, err
	}

	regeocode, ok := ret["regeocode"].(map[string]any)
	if !ok {
		return GisData{}, fmt.Errorf("invalid regeocode payload")
	}
	addressComponent, ok := regeocode["addressComponent"].(map[string]any)
	if !ok {
		return GisData{}, fmt.Errorf("invalid addressComponent payload")
	}

	province := getStringValue(addressComponent["province"])
	if strings.Contains(province, "中华人民共和国") {
		province = ""
	}
	district := getStringValue(addressComponent["district"])
	township := getStringValue(addressComponent["township"])

	street := ""
	if streetNumber, ok := addressComponent["streetNumber"].(map[string]any); ok {
		street = getStringValue(streetNumber["street"])
	}

	locStreet := province + district + township + street
	locAddr := getStringValue(regeocode["formatted_address"])

	return GisData{LocStreet: locStreet, LocAddr: locAddr}, nil
}

func getStringValue(v any) string {
	s, _ := v.(string)
	return s
}
