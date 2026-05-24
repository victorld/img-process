package model

import "time"

const (
	ActionTypeDelete         = "delete"
	ActionTypeMove           = "move"
	ActionTypeRename         = "rename"
	ActionTypeModifyTime     = "modify_time"
	ActionTypeDeleteEmptyDir = "delete_empty_dir"
	ActionTypeDeleteDup      = "delete_duplicate"
	ActionTypeDeletePathDup  = "delete_path_duplicate"

	ActionObjectFile = "file"
	ActionObjectDir  = "dir"

	ActionStageCandidate = "candidate"
	ActionStageExecuted  = "executed"
	ActionStageDiscovery = "discovery"

	ActionStatusPending   = "pending"
	ActionStatusSucceeded = "succeeded"
	ActionStatusFailed    = "failed"
	ActionStatusSkipped   = "skipped"
)

type ScanActionItemDB struct {
	CommonModel
	JobID          uint       `json:"jobId" gorm:"column:job_id;index;comment:任务ID"`
	ActionType     string     `json:"actionType" gorm:"column:action_type;size:64;index;comment:动作类型"`
	ObjectType     string     `json:"objectType" gorm:"column:object_type;size:32;comment:对象类型"`
	SourcePath     string     `json:"sourcePath" gorm:"column:source_path;size:512;index;comment:源路径"`
	TargetPath     string     `json:"targetPath" gorm:"column:target_path;size:1024;comment:目标路径"`
	ReasonCode     string     `json:"reasonCode" gorm:"column:reason_code;size:64;comment:原因码"`
	ReasonText     string     `json:"reasonText" gorm:"column:reason_text;type:text;comment:原因文本"`
	Stage          string     `json:"stage" gorm:"column:stage;size:32;index;comment:阶段"`
	Status         string     `json:"status" gorm:"column:status;size:32;index;comment:状态"`
	DiscoveredAt   *time.Time `json:"discoveredAt" gorm:"column:discovered_at;comment:发现时间"`
	ExecutedAt     *time.Time `json:"executedAt" gorm:"column:executed_at;comment:执行时间"`
	ErrorMessage   string     `json:"errorMessage" gorm:"column:error_message;type:text;comment:错误信息"`
	MetadataJSON   string     `json:"metadataJson" gorm:"column:metadata_json;type:longtext;comment:附加元数据"`
	DuplicateGroup string     `json:"duplicateGroup" gorm:"column:duplicate_group;size:128;index;comment:重复分组"`
}

func (ScanActionItemDB) TableName() string {
	return "scan_action_item"
}

type ScanActionItemSearch struct {
	PageInfo
	JobID      uint   `json:"jobId" form:"jobId"`
	Tab        string `json:"tab" form:"tab"`
	ActionType string `json:"actionType" form:"actionType"`
	Status     string `json:"status" form:"status"`
	Keyword    string `json:"keyword" form:"keyword"`
}

type ScanActionCounts struct {
	Delete          int64 `json:"delete"`
	DeleteEmptyDir  int64 `json:"deleteEmptyDir"`
	Move            int64 `json:"move"`
	ModifyTime      int64 `json:"modifyTime"`
	DeleteDuplicate int64 `json:"deleteDuplicate"`
	DeletePathDup   int64 `json:"deletePathDuplicate"`
	Rename          int64 `json:"rename"`
	Total           int64 `json:"total"`
}

type ScanActionGroupedCounts struct {
	Pending  ScanActionCounts `json:"pending"`
	Executed ScanActionCounts `json:"executed"`
	Error    ScanActionCounts `json:"error"`
}

type ScanActionPreview struct {
	FileName          string `json:"fileName"`
	Path              string `json:"path"`
	PreviewSlot       string `json:"previewSlot,omitempty"`
	SizeBytes         int64  `json:"sizeBytes,omitempty"`
	SizeText          string `json:"sizeText,omitempty"`
	MD5Matched        bool   `json:"md5Matched"`
	MatchKey          string `json:"matchKey,omitempty"`
	MatchType         string `json:"matchType,omitempty"`
	PathSource        string `json:"pathSource,omitempty"`
	RecommendedDelete bool   `json:"recommendedDelete"`
	ExecutedAction    bool   `json:"executedAction"`
	DeleteEligible    bool   `json:"deleteEligible"`
	CandidateIndex    int    `json:"candidateIndex"`
	MatchCount        int    `json:"matchCount,omitempty"`
}

type ScanActionDuplicateMeta struct {
	SizeMatch              bool   `json:"sizeMatch"`
	DeleteEligible         bool   `json:"deleteEligible"`
	DeleteIneligibleReason string `json:"deleteIneligibleReason,omitempty"`
}

type ScanActionDetail struct {
	FileName       string `json:"fileName,omitempty"`
	CurrentPath    string `json:"currentPath,omitempty"`
	TargetPath     string `json:"targetPath,omitempty"`
	TargetFileName string `json:"targetFileName,omitempty"`
	DirDate        string `json:"dirDate,omitempty"`
	ModifyDate     string `json:"modifyDate,omitempty"`
	FileNameDate   string `json:"fileNameDate,omitempty"`
	ShootDate      string `json:"shootDate,omitempty"`
	ShootDateRaw   string `json:"shootDateRaw,omitempty"`
	MinDate        string `json:"minDate,omitempty"`
	PreviewSlot    string `json:"previewSlot,omitempty"`
}

type ScanActionPair struct {
	PhotoA ScanActionPreview `json:"photoA"`
	PhotoB ScanActionPreview `json:"photoB"`
}

type ScanActionItemView struct {
	ID              uint                     `json:"id"`
	ActionType      string                   `json:"actionType"`
	ObjectType      string                   `json:"objectType"`
	SourcePath      string                   `json:"sourcePath"`
	TargetPath      string                   `json:"targetPath,omitempty"`
	ReasonCode      string                   `json:"reasonCode,omitempty"`
	ReasonText      string                   `json:"reasonText,omitempty"`
	Stage           string                   `json:"stage"`
	Status          string                   `json:"status"`
	DiscoveredAt    *time.Time               `json:"discoveredAt,omitempty"`
	ExecutedAt      *time.Time               `json:"executedAt,omitempty"`
	ErrorMessage    string                   `json:"errorMessage,omitempty"`
	MetadataJSON    string                   `json:"metadataJson,omitempty"`
	DuplicateGroup  string                   `json:"duplicateGroup,omitempty"`
	Detail          *ScanActionDetail        `json:"detail,omitempty"`
	Pair            *ScanActionPair          `json:"pair,omitempty"`
	DuplicatePhotos []ScanActionPreview      `json:"duplicatePhotos,omitempty"`
	DuplicateMeta   *ScanActionDuplicateMeta `json:"duplicateMeta,omitempty"`
}
