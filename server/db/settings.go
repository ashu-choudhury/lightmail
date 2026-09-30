package db

import (
	"time"

	"github.com/Jinnrry/pmail/models"
)

// GetSetting retrieves a persisted value from SQLite by key.
func GetSetting(key string) string {
	if Instance == nil {
		return ""
	}
	var s models.Setting
	has, err := Instance.Where("key = ?", key).Get(&s)
	if err == nil && has {
		return s.Value
	}
	return ""
}

// SetSetting writes or updates a persisted key-value setting in SQLite.
func SetSetting(key, value string) error {
	if Instance == nil {
		return nil
	}
	var s models.Setting
	has, err := Instance.Where("key = ?", key).Get(&s)
	if err != nil {
		return err
	}
	if has {
		s.Value = value
		s.UpdatedAt = time.Now()
		_, err = Instance.Where("key = ?", key).Cols("value", "updated_at").Update(&s)
		return err
	}
	s.Key = key
	s.Value = value
	s.UpdatedAt = time.Now()
	_, err = Instance.Insert(&s)
	return err
}
