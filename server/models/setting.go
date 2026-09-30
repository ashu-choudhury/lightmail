package models

import "time"

type Setting struct {
	Key       string    `xorm:"pk varchar(64) 'key'"`
	Value     string    `xorm:"text 'value'"`
	UpdatedAt time.Time `xorm:"updated 'updated_at'"`
}

func (Setting) TableName() string {
	return "setting"
}
