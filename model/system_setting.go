package model

type SystemSettingDB struct {
	CommonModel
	Section   string `json:"section" gorm:"column:section;size:64;not null;uniqueIndex:idx_system_setting_section_key;comment:配置分组"`
	ItemKey   string `json:"itemKey" gorm:"column:item_key;size:128;not null;uniqueIndex:idx_system_setting_section_key;comment:配置键"`
	ValueJSON string `json:"valueJson" gorm:"column:value_json;type:longtext;comment:配置值JSON"`
	ValueType string `json:"valueType" gorm:"column:value_type;size:32;not null;comment:配置值类型"`
}

func (SystemSettingDB) TableName() string {
	return "system_setting"
}
