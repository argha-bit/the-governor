package historyrepository

import (
	"context"
	"the_governor/adapters/db"
	"the_governor/models"
)

type HistoryRepository struct {
	db *db.DBConfig
}

func NewHistoryRepository(db *db.DBConfig) *HistoryRepository {
	return &HistoryRepository{db: db}
}

// Create creates a new history record
func (r *HistoryRepository) Create(ctx context.Context, history *models.ConfigFetchHistory) error {
	return r.db.Create(history).Error
}

// UpdateStatus updates the status of a history record
func (r *HistoryRepository) UpdateStatus(ctx context.Context, id uint64, status string, errorMsg string) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if errorMsg != "" {
		updates["error_message"] = errorMsg
	}
	return r.db.Model(&models.ConfigFetchHistory{}).Where("id = ?", id).Updates(updates).Error
}

// UpdateFilesFetched updates the files_fetched field
func (r *HistoryRepository) UpdateFilesFetched(ctx context.Context, id uint64, files models.JSONFetchedFiles) error {
	return r.db.Model(&models.ConfigFetchHistory{}).Where("id = ?", id).
		Update("files_fetched", files).Error
}

// GetByCommitSHA gets all history records for a commit SHA
func (r *HistoryRepository) GetByCommitSHA(ctx context.Context, commitSHA string) ([]*models.ConfigFetchHistory, error) {
	var history []*models.ConfigFetchHistory
	err := r.db.Where("commit_sha = ?", commitSHA).Find(&history).Error
	return history, err
}

// GetByServiceID gets history for a service
func (r *HistoryRepository) GetByServiceID(ctx context.Context, serviceID uint64, limit int) ([]*models.ConfigFetchHistory, error) {
	var history []*models.ConfigFetchHistory
	query := r.db.Where("service_id = ?", serviceID).Order("fetched_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&history).Error
	return history, err
}
