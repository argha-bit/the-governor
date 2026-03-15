package servicerepository

import (
	"context"
	"log"
	"the_governor/adapters/db"
	"the_governor/models"

	"github.com/jinzhu/gorm"
)

type ServiceRepository struct {
	db *db.DBConfig
}

// NewServiceRepository creates a new service repository
func NewServiceRepository(db *db.DBConfig) *ServiceRepository {
	return &ServiceRepository{db: db}
}

func (r *ServiceRepository) Register(ctx context.Context, service *models.RegisteredService) error {
	// Upsert: update if exists, insert if not
	log.Printf("Registering service: %+v", service)
	if r.db == nil {
		log.Println("Database connection is nil")
		return gorm.ErrInvalidSQL
	}
	return r.db.Where("owner = ? AND repository = ?", service.Owner, service.Repository).
		Assign(service).
		FirstOrCreate(service).Error
}

func (r *ServiceRepository) RegisterV2(ctx context.Context, service *models.RegisterServiceV2) error {
	// Upsert: update if exists, insert if not
	log.Printf("Registering service V2: %+v", service)
	if r.db == nil {
		log.Println("Database connection is nil")
		return gorm.ErrInvalidSQL
	}
	return r.db.Create(service).Error
}

func (r *ServiceRepository) GetByID(ctx context.Context, id string) (*models.RegisterServiceV2, error) {
	var service models.RegisterServiceV2
	err := r.db.Where("service_id = ?", id).First(&service).Error
	if err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, err
		}
		return nil, err
	}
	return &service, nil
}

func (r *ServiceRepository) GetByOwnerRepo(ctx context.Context, owner, repo string) (*models.RegisteredService, error) {
	var service models.RegisteredService
	err := r.db.Where("owner = ? AND repository = ?", owner, repo).First(&service).Error
	if err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, nil
		}
		return nil, err
	}
	return &service, nil
}

// Unregister removes a service
func (r *ServiceRepository) Unregister(ctx context.Context, owner, repo string) error {
	return r.db.Where("owner = ? AND repository = ?", owner, repo).
		Delete(&models.RegisteredService{}).Error
}

// ListAll lists all registered services
func (r *ServiceRepository) ListAll(ctx context.Context) ([]*models.RegisteredService, error) {
	var services []*models.RegisteredService
	err := r.db.Find(&services).Error
	return services, err
}
