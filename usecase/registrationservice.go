package usecase

import "the_governor/models"

type RegistrationServiceUsecaseHandler interface {
	RegisterService(model *models.RegisteredService) error
	RegisterServiceV2(model *models.RegisterServiceV2) error
}
