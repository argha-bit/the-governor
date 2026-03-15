package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// RegisteredService represents a service registered with The Governor
type RegisteredService struct {
	ID             uint64          `gorm:"primary_key;auto_increment" json:"id"`
	InstallationID int64           `gorm:"type:bigint;not null" json:"installation_id"`
	Owner          string          `gorm:"type:varchar(255);not null" json:"owner"`
	Repository     string          `gorm:"type:varchar(255);not null" json:"repository"`
	ConfigPaths    JSONStringArray `gorm:"type:json;not null" json:"config_paths"`
	Branch         string          `gorm:"type:varchar(255);default:'main'" json:"branch"`
	Metadata       JSONMap         `gorm:"type:json" json:"metadata"`
	RegisteredAt   *time.Time      `json:"registered_at"`
	UpdatedAt      *time.Time      `json:"updated_at"`
}

type RegisterServiceV2 struct {
	ServiceID      string     `json:"service_id"`
	ServiceName    string     `json:"service_name"`
	TeamName       string     `json:"team_name"`
	Namespace      string     `json:"namespace"`
	ContactEmail   string     `json:"contact_email"`
	ConfigEndpoint string     `json:"config_endpoint"`
	WebhookURL     string     `json:"webhook_url"`
	Metadata       JSONMap    `gorm:"type:json" json:"metadata"`
	CreatedAt      *time.Time `json:"created_at"`
	UpdatedAt      *time.Time `json:"updated_at"`
}

// TableName specifies the table name
func (RegisteredService) TableName() string {
	return "registered_services"
}

// ConfigFetchHistory represents a config fetch record
type ConfigFetchHistory struct {
	ID           uint64           `gorm:"primary_key;auto_increment" json:"id"`
	ServiceID    uint64           `gorm:"not null;index" json:"service_id"`
	Owner        string           `gorm:"type:varchar(255);not null" json:"owner"`
	Repository   string           `gorm:"type:varchar(255);not null" json:"repository"`
	CommitSHA    string           `gorm:"type:varchar(40);not null;index" json:"commit_sha"`
	Branch       string           `gorm:"type:varchar(255);not null" json:"branch"`
	FilesFetched JSONFetchedFiles `gorm:"type:json;not null" json:"files_fetched"`
	FetchedAt    *time.Time       `gorm:"index" json:"fetched_at"`
	Status       string           `gorm:"type:enum('pending','success','failed');not null;default:'pending';index" json:"status"`
	ErrorMessage string           `gorm:"type:text" json:"error_message,omitempty"`
}

func (RegisterServiceV2) TableName() string {
	return "services"
}

// TableName specifies the table name
func (ConfigFetchHistory) TableName() string {
	return "config_fetch_history"
}

// FetchedFile represents a single fetched file
type FetchedFile struct {
	Path       string `json:"path"`
	Content    string `json:"content"`
	ConfigType string `json:"config_type"`
}

// JSONStringArray is a custom type for string arrays stored as JSON
type JSONStringArray []string

// Scan implements sql.Scanner
func (j *JSONStringArray) Scan(value interface{}) error {
	if value == nil {
		*j = []string{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, j)
}

// Value implements driver.Valuer
func (j JSONStringArray) Value() (driver.Value, error) {
	if len(j) == 0 {
		return "[]", nil
	}
	return json.Marshal(j)
}

// JSONMap is a custom type for map stored as JSON
type JSONMap map[string]interface{}

// Scan implements sql.Scanner
func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = make(map[string]interface{})
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, j)
}

// Value implements driver.Valuer
func (j JSONMap) Value() (driver.Value, error) {
	if len(j) == 0 {
		return "{}", nil
	}
	return json.Marshal(j)
}

// JSONFetchedFiles is a custom type for fetched files stored as JSON
type JSONFetchedFiles []FetchedFile

// Scan implements sql.Scanner
func (j *JSONFetchedFiles) Scan(value interface{}) error {
	if value == nil {
		*j = []FetchedFile{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, j)
}

// Value implements driver.Valuer
func (j JSONFetchedFiles) Value() (driver.Value, error) {
	if len(j) == 0 {
		return "[]", nil
	}
	return json.Marshal(j)
}
