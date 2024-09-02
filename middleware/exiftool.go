package middleware

import (
	"fmt"
	"img_process/tools"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func GetExifInfoCommand(path string) (string, string, error) {

	var shootTime string
	var locNum string

	output, err := tools.GetOutputCommand("exiftool -G '" + path + "' | grep -v 'File'| grep -v '0000' | grep -E 'GPS Position|Date'")
	var dateList []string
	var gpsLine string

	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "GPS Position") {
			gpsLine = line
		}
		if strings.Contains(line, "Date") {
			dateList = append(dateList, line)
		}
	}
	if err != nil {
		//tools.FancyHandleError(err)
		return "", "", err
	}

	//tools.Logger.Info("cmd output : ", gpsLine)

	gpsRegexp := regexp.MustCompile(`^.*: (\d*) deg (\d*)' (\d*\.?\d*)" N, (\d*) deg (\d*)' (\d*\.?\d*)" E.*$`)
	gpsVal := gpsRegexp.FindStringSubmatch(gpsLine)

	if len(gpsVal) == 7 {
		p1, _ := strconv.ParseFloat(gpsVal[1], 64)
		p2, _ := strconv.ParseFloat(gpsVal[2], 64)
		p3, _ := strconv.ParseFloat(gpsVal[3], 64)
		lat := p1 + p2/60 + p3/3600
		p4, _ := strconv.ParseFloat(gpsVal[4], 64)
		p5, _ := strconv.ParseFloat(gpsVal[5], 64)
		p6, _ := strconv.ParseFloat(gpsVal[6], 64)
		lon := p4 + p5/60 + p6/3600
		locNum = fmt.Sprintf("%.6f", lon) + "," + fmt.Sprintf("%.6f", lat)
		//tools.Logger.Info("locNum : ", locNum)
	} else {
		//tools.Logger.Error("gps解析失败 ", gpsLine)
	}

	dateRegexp := regexp.MustCompile(`^.*(\d{4}:\d{2}:\d{2}).*$`)
	var minDate string
	for _, line := range dateList {
		dateVal := dateRegexp.FindStringSubmatch(line)
		if len(dateVal) == 2 {
			if minDate == "" {
				minDate = dateVal[1]
			} else {
				if dateVal[1] < minDate {
					minDate = dateVal[1]
				}
			}
		} else {
			tools.Logger.Error("date解析失败 ", dateList)
		}
	}
	if minDate != "" {
		t, err := time.Parse("2006:01:02", minDate)
		if err == nil {
			shootTime = t.Format("2006-01-02")
		}
	}

	return shootTime, locNum, nil

}
