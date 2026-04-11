package middleware

import (
	"reflect"
	"testing"
)

func TestParseExiftoolOutput(t *testing.T) {
	output := `[EXIF]          Date/Time Original              : 2024:01:02 03:04:05
[XMP]           Metadata Date                   : 2024:01:02 03:04:05
[Composite]     GPS Position                    : 30 deg 33' 33.635" N, 114 deg 16' 46.761" E
`

	shootTime, locNum, dateTagNames := parseExiftoolOutput(output)
	if shootTime != "2024:01:02 03:04:05" {
		t.Fatalf("shootTime = %q, want %q", shootTime, "2024:01:02 03:04:05")
	}
	if locNum != "114.279656,30.559343" {
		t.Fatalf("locNum = %q, want %q", locNum, "114.279656,30.559343")
	}
	wantTags := []string{"[EXIF]Date/Time Original"}
	if !reflect.DeepEqual(dateTagNames, wantTags) {
		t.Fatalf("dateTagNames = %#v, want %#v", dateTagNames, wantTags)
	}
}

func TestBuildExiftoolCommandPreservesPath(t *testing.T) {
	path := `/tmp/含 空格 'quote'/IMG_01.JPG`
	cmd := buildExiftoolCommand("-G", path)
	want := []string{"exiftool", "-G", path}
	if !reflect.DeepEqual(cmd.Args, want) {
		t.Fatalf("cmd.Args = %#v, want %#v", cmd.Args, want)
	}
}
