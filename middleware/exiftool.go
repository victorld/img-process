package middleware

import (
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"img_process/tools"
)

var runExiftoolFunc = runExiftool

func IsExiftoolAvailable() bool {
	_, err := exec.LookPath("exiftool")
	return err == nil
}

// GetExifInfoCommand 用命令行找到照片的拍摄时间和地理位置
func GetExifInfoCommand(path string) (string, string, string, []string, error) {
	output, err := runExiftoolFunc("-G", path)
	if err != nil {
		return "", "", "", nil, err
	}

	shootTime, locNum, dateTagNames := parseExiftoolOutput(output)
	return shootTime, locNum, output, dateTagNames, nil
}

func parseExiftoolOutput(output string) (string, string, []string) {
	var (
		dateList     []string
		dateTagNames []string
		gpsLine      string
		locNum       string
		shootTime    string
	)

	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "[File]") ||
			strings.Contains(line, "0000") ||
			strings.Contains(line, "Stamp") ||
			strings.Contains(line, "Profile") ||
			strings.Contains(line, "Create Date") ||
			strings.Contains(line, "Metadata") ||
			strings.Contains(line, "Media") ||
			strings.Contains(line, "Track") ||
			strings.Contains(line, "GPS Date") ||
			strings.Contains(line, "Sony") {
			continue
		}
		if strings.Contains(line, "GPS Position") {
			gpsLine = line
		}
		if strings.Contains(line, "Date") {
			dateList = append(dateList, line)
		}
	}

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
	}

	dateRegexp := regexp.MustCompile(`^.*(\d{4}:\d{2}:\d{2} \d{2}:\d{2}:\d{2}).*$`)
	for _, line := range dateList {
		dateValList := dateRegexp.FindStringSubmatch(line)
		if len(dateValList) != 2 {
			continue
		}

		dateVal := dateValList[1]
		if strings.Contains(line, "QuickTime") && strings.Contains(line, "Modify Date") {
			loc, _ := time.LoadLocation("UTC")
			t, _ := time.ParseInLocation("2006:01:02 15:04:05", dateVal, loc)
			dateVal = t.Local().Format("2006:01:02 15:04:05")
		}
		if shootTime == "" || dateVal < shootTime {
			shootTime = dateVal
		}

		parts := strings.Split(strings.Split(line, ":")[0], "]")
		if len(parts) >= 2 {
			dateTagNames = append(dateTagNames, strings.TrimSpace(parts[0])+"]"+strings.TrimSpace(parts[1]))
		}
	}

	return shootTime, locNum, dateTagNames
}

func ModifyShootDate(path string, shootDate string) error {
	output, err := runExiftoolFunc("-DateTimeOriginal="+shootDate, path)
	if err != nil {
		if !IsExiftoolAvailable() {
			output, err = runExiftoolWriteViaPerl(shootDate, path)
		}
	}
	if err != nil {
		tools.FancyHandleError(err)
		return err
	}
	if !strings.Contains(output, "updated") {
		tools.Logger.Error("ModifyShootDate failed,file : ", path)
		return errors.New("ModifyShootDate failed ")
	}

	return nil
}

func buildExiftoolCommand(args ...string) *exec.Cmd {
	return exec.Command("exiftool", args...)
}

func runExiftool(args ...string) (string, error) {
	cmd := buildExiftoolCommand(args...)
	bytes, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(string(bytes)))
	}
	return string(bytes), nil
}

func runExiftoolWriteViaPerl(shootDate string, path string) (string, error) {
	cmd := exec.Command(
		"perl",
		"-MImage::ExifTool",
		"-e",
		`my $et = Image::ExifTool->new; $et->SetNewValue("DateTimeOriginal" => $ARGV[0]); my $ok = $et->WriteInfo($ARGV[1]); if ($ok) { print "1 image files updated\n"; exit 0; } print "failed to update DateTimeOriginal\n"; exit 1;`,
		shootDate,
		path,
	)
	bytes, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(string(bytes)))
	}
	return string(bytes), nil
}
