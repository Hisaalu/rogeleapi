package domain

import "time"

type Setting struct {
	ID           int64      `json:"id"`
	SettingKey   string     `json:"setting_key"`
	SettingValue *string    `json:"setting_value,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    *time.Time `json:"updated_at,omitempty"`
}
