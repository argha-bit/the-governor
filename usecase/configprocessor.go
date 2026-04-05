package usecase

import "the_governor/models"

type ConfigProcessorWebhookUsecase interface {
	ReadConfig(serviceDetails *models.RegisterServiceV2) error
}

type ConfigProcessorPluginUsecaseHandler interface {
	ReadConfig(fileName string) error
}
