package comm

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes"
)

func HashPassword(pwd string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	return string(b), err
}

func CheckPassword(hash string, pwd string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pwd)) == nil
}

// ParseOptionalTime 解析可选时间字符串
func ParseOptionalTime(input string, layout string) (*time.Time, error) {
	value := strings.TrimSpace(input)
	if value == "" {
		return nil, nil
	}

	var t time.Time
	var err error

	if t, err = time.Parse(time.RFC3339, value); err == nil {
		return &t, nil
	}

	if layout != "" {
		t, err = time.Parse(layout, value)
	} else {
		t, err = time.Parse("2006-01-02 15:04:05", value)
	}

	if err != nil {
		return nil, err
	}
	return &t, nil
}

// UnmarshalImages 解析图片JSON字符串
func UnmarshalImages(raw string) []string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil
	}
	var images []string
	if err := json.Unmarshal([]byte(value), &images); err != nil {
		return nil
	}
	return images
}

// ParseEventTime 解析事件时间字符串
func ParseEventTime(input string) (time.Time, error) {
	value := strings.TrimSpace(input)
	if value == "" {
		return time.Time{}, errors.New("empty time")
	}
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02 15:04:05", value)
}

// MarshalImages 序列化图片列表为JSON
func MarshalImages(images []string) (datatypes.JSON, error) {
	if len(images) == 0 {
		return nil, nil
	}
	b, err := json.Marshal(images)
	if err != nil {
		return nil, err
	}
	return b, nil
}
