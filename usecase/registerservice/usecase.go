package registerservice

import (
	"context"
	"fmt"
	"log"
	"the_governor/models"
	"the_governor/repository/servicerepository"
	"the_governor/usecase"

	"github.com/go-sql-driver/mysql"
)

type RegisterServiceUsecase struct {
	registerServiceRepo servicerepository.ServiceRepository
}

func (u *RegisterServiceUsecase) RegisterService(model *models.RegisteredService) error {
	log.Printf("Registering service: %+v", model)
	if u.registerServiceRepo.Register(context.Background(), model) != nil {
		log.Printf("Error registering service: %+v", model)
		return fmt.Errorf("failed to register service for %s/%s", model.Owner, model.Repository)
	}
	return nil
}
func (u *RegisterServiceUsecase) RegisterServiceV2(model *models.RegisterServiceV2) error {
	log.Printf("Registering service: %+v", model)
	if err := u.registerServiceRepo.RegisterV2(context.Background(), model); err != nil {
		log.Printf("Error registering service: %+v", model)
		if mysqlErr, ok := err.(*mysql.MySQLError); ok {
			log.Printf("MySQL error code: %d, message: %s", mysqlErr.Number, mysqlErr.Message)
			if mysqlErr.Number == 1062 {
				return fmt.Errorf("service with name %s already exists, Please update", model.ServiceName)
			}
			return fmt.Errorf("failed to register service for %s/%s", model.ServiceName, model.TeamName)
		}
	}
	return nil
}

func NewRegisterServiceUsecaseHandler(repo servicerepository.ServiceRepository) usecase.RegistrationServiceUsecaseHandler {
	return &RegisterServiceUsecase{
		registerServiceRepo: repo,
	}
}
