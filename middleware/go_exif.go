package middleware

import (
	"fmt"
	goexif "github.com/dsoprea/go-exif/v3"
	exifcommon "github.com/dsoprea/go-exif/v3/common"
	"img_process/tools"
	"time"
)

// DateTime    DateTimeOriginal    DateTimeDigitized
func GetExifValue(updatedExifIfd *goexif.Ifd, key string) (string, error) {

	results, err := updatedExifIfd.FindTagWithName(key)
	if err != nil {
		//tools.FancyHandleError(err)
		return "", err
	}

	ite := results[0]

	phrase, err := ite.FormatFirst()
	if err != nil {
		//tools.FancyHandleError(err)
		return "", err
	}

	return phrase, nil
}

func GetExifInfoGo(path string) (string, string, error) {

	rawExif, err := goexif.SearchFileAndExtractExif(path)
	if err != nil {
		//tools.FancyHandleError(err)
		return "", "", err
	}

	im, err := exifcommon.NewIfdMappingWithStandard()
	if err != nil {
		//tools.FancyHandleError(err)
		return "", "", err
	}

	ti := goexif.NewTagIndex()

	_, index, err := goexif.Collect(im, ti, rawExif)
	if err != nil {
		//tools.FancyHandleError(err)
		return "", "", err
	}

	var shootTime string
	updatedRootIfd := index.RootIfd
	updatedExifIfd, err := updatedRootIfd.ChildWithIfdPath(exifcommon.IfdExifStandardIfdIdentity)
	if err == nil {
		value, err := GetExifValue(updatedExifIfd, "DateTimeOriginal")
		if err != nil {
			value, err = GetExifValue(updatedExifIfd, "DateTime")
			if err != nil {
				value, err = GetExifValue(updatedExifIfd, "DateTimeDigitized")
			}
		}

		if value != "" {
			exifTimeLayout := "2006:01:02 15:04:05"
			t, err := time.Parse(exifTimeLayout, value)
			if err == nil {
				shootTime = t.Format("2006-01-02")
			}
		}
	}

	var locNum string
	updatedRootIfd2 := index.RootIfd
	updatedRootIfd2, err = updatedRootIfd2.ChildWithIfdPath(exifcommon.IfdGpsInfoStandardIfdIdentity)
	if err == nil {
		gi, err := updatedRootIfd2.GpsInfo()
		if err == nil {
			locNum = fmt.Sprintf("%.6f", gi.Longitude.Decimal()) + "," + fmt.Sprintf("%.6f", gi.Latitude.Decimal())
		}
	}

	return shootTime, locNum, nil
}

func GetGpsData(path string) {

	rawExif, err := goexif.SearchFileAndExtractExif(path)
	if err != nil {
		tools.FancyHandleError(err)
		return
	}

	im, err := exifcommon.NewIfdMappingWithStandard()
	if err != nil {
		tools.FancyHandleError(err)
		return
	}

	ti := goexif.NewTagIndex()

	_, index, err := goexif.Collect(im, ti, rawExif)
	if err != nil {
		tools.FancyHandleError(err)
		return
	}

	ifd, err := index.RootIfd.ChildWithIfdPath(exifcommon.IfdGpsInfoStandardIfdIdentity)
	if err != nil {
		tools.FancyHandleError(err)
		return
	}

	gi, err := ifd.GpsInfo()
	if err != nil {
		tools.FancyHandleError(err)
		return
	}

	fmt.Printf("%s\n", gi)
}

func PrintExifData(path string) {

	opt := goexif.ScanOptions{}
	dt, err := goexif.SearchFileAndExtractExif(path)
	if err != nil {
		tools.FancyHandleError(err)
		return
	}
	ets, _, err := goexif.GetFlatExifData(dt, &opt)
	if err != nil {
		tools.FancyHandleError(err)
		return
	}
	for _, et := range ets {
		fmt.Println(et.TagId, et.TagName, et.TagTypeName, et.Value)
	}
}
