package server

import (
	"log"
	"os"
	"strconv"
	"the_governor/adapters/db"
	registrationcontroller "the_governor/controller/regostrationcontroller"
	"the_governor/controller/webhookcontroller"
	"the_governor/repository/historyrepository"
	"the_governor/repository/servicerepository"
	"the_governor/request"
	"the_governor/usecase/githubutility"
	"the_governor/usecase/registerservice"
	"the_governor/usecase/webhook"
	validator "the_governor/utils/validator"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

func newRouter() *echo.Echo {
	// Initialize Echo router
	e := echo.New()
	e.Use(middleware.RequestIDWithConfig(middleware.RequestIDConfig{
		Generator: func() string {
			return uuid.New().String()
		},
	}))
	dbConfig, err := db.GetMySQL()
	if err != nil {
		log.Println("Error loading database configuration", err.Error())
		return nil
	}
	if dbConfig == nil {
		log.Println("Database configuration is nil")
		return nil
	}
	e.Validator = validator.NewValidator()
	log.Println("Database configuration loaded successfully")
	appID, _ := strconv.Atoi(os.Getenv("APP_ID"))
	privateKeyPath := os.Getenv("PRIVATE_KEY_PATH")
	historyRepositoryHandler := historyrepository.NewHistoryRepository(dbConfig)
	registrationRequestHandler := request.NewRegistrationRequestHandler()
	webhookRequestHandler := request.NewWebhookRequestHandler()
	registrationRepositoryHandler := servicerepository.NewServiceRepository(dbConfig)
	registerUsecase := registerservice.NewRegisterServiceUsecaseHandler(*registrationRepositoryHandler)
	ghAppClient := githubutility.NewGitHubAppClient(int64(appID), privateKeyPath)
	webhookUsecase := webhook.NewWebhookUsecaseHandler(*registrationRepositoryHandler, *historyRepositoryHandler, ghAppClient)

	registrationcontroller.NewRegistrationController(e, registrationRequestHandler, registerUsecase)
	webhookcontroller.NewWebhookController(e, webhookRequestHandler, webhookUsecase)
	return e
}
