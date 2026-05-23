package dao

import (
	"strings"
	"testing"
	"time"

	"img_process/model"
)

func TestFileAnalysisStats(t *testing.T) {
	rows := []model.ImgDatabaseDB{
		{ImgKey: "2025-01-02|IMG_0003.JPG"},
		{ImgKey: "2023-08-19|VID_0001.MP4"},
		{ImgKey: "2023-08-20|VID_0002.MP4"},
		{ImgKey: "2024-01-02|IMG_0001.JPG"},
		{ImgKey: "2024-01-03|IMG_0002.HEIC"},
		{ImgKey: "2023-08-21|VID_0003.MP4"},
		{ImgKey: "unknown|RAW_FILE"},
	}

	yearStats := fileAnalysisYearStats(rows)
	if len(yearStats) != 3 {
		t.Fatalf("year stats len = %d, want 3", len(yearStats))
	}
	if yearStats[0].Key != "2025" || yearStats[0].Count != 1 {
		t.Fatalf("first year stat = %+v, want 2025/1", yearStats[0])
	}
	if yearStats[1].Key != "2024" || yearStats[2].Key != "2023" {
		t.Fatalf("year stats order = %+v, want year desc", yearStats)
	}
	if yearStats[0].Percent < 14 || yearStats[0].Percent > 15 {
		t.Fatalf("first year percent = %f, want about 14.3", yearStats[0].Percent)
	}

	suffixStats := fileAnalysisSuffixStats(rows)
	if len(suffixStats) != 4 {
		t.Fatalf("suffix stats len = %d, want 4", len(suffixStats))
	}
	got := map[string]int64{}
	for _, item := range suffixStats {
		got[item.Key] = item.Count
	}
	want := map[string]int64{".jpg": 2, ".heic": 1, ".mp4": 3, "无后缀": 1}
	for suffix, count := range want {
		if got[suffix] != count {
			t.Fatalf("suffix %s count = %d, want %d", suffix, got[suffix], count)
		}
	}
}

func TestFileAnalysisListOrder(t *testing.T) {
	if !strings.Contains(fileAnalysisListOrder, "SUBSTRING_INDEX(img_key, '|', 1) asc") {
		t.Fatalf("list order should sort by dir date asc: %s", fileAnalysisListOrder)
	}
	if !strings.Contains(fileAnalysisListOrder, "REPLACE(LEFT(shoot_date, 19), ':', '-') asc") {
		t.Fatalf("list order should sort by shoot date asc: %s", fileAnalysisListOrder)
	}
	if !strings.Contains(fileAnalysisListOrder, "THEN 1 ELSE 0 END asc") {
		t.Fatalf("list order should put missing shoot date after present values: %s", fileAnalysisListOrder)
	}
}

func TestToFileAnalysisItemSplitsImgKey(t *testing.T) {
	updatedAt := time.Date(2026, 5, 23, 8, 42, 0, 0, time.Local)
	item := toFileAnalysisItem(model.ImgDatabaseDB{
		CommonModel: model.CommonModel{ID: 7, UpdatedAt: updatedAt},
		ImgKey:      "2024-01-02|IMG_0001.JPG",
		ShootDate:   "2024:01:02 01:02:03",
		LocNum:      "31.230416,121.473701",
		LocStreet:   "黄浦区中山东一路",
		LocAddr:     "上海市黄浦区",
		Remark:      "ok",
	})

	if item.ID != 7 || item.DirDate != "2024-01-02" || item.FileName != "IMG_0001.JPG" || item.Suffix != ".jpg" {
		t.Fatalf("item split = %+v", item)
	}
	if item.UpdatedAt != updatedAt {
		t.Fatalf("updatedAt = %v, want %v", item.UpdatedAt, updatedAt)
	}
}

func TestNormalizeShootDateFilter(t *testing.T) {
	if got := normalizeShootDateFilter("2024:01:02", false); got != "2024-01-02" {
		t.Fatalf("full date = %q", got)
	}
	if got := normalizeShootDateFilter("2024-01", false); got != "2024-01-01" {
		t.Fatalf("month start = %q", got)
	}
	if got := normalizeShootDateFilter("2024-01", true); got != "2024-01-31" {
		t.Fatalf("month end = %q", got)
	}
	if got := normalizeShootDateFilter("2024", true); got != "2024-12-31" {
		t.Fatalf("year end = %q", got)
	}
}
