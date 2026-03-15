package request

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"the_governor/models"

	"github.com/labstack/echo/v5"
)

type RegistrationRequestHandler interface {
	Bind(c *echo.Context, request interface{}, model interface{}) error
}

type RegisterServiceRequest struct {
	Owner          string      `json:"owner"`
	Repository     string      `json:"repository"`
	ConfigPaths    []string    `json:"config_paths"`
	Branch         string      `json:"branch"`
	Metadata       interface{} `json:"metadata"`
	InstallationID int64       `json:"installation_id"`
}

type RegisterServiceRequestV2 struct {
	ServiceName    string      `json:"service_name" validate:"required,min=3,max=100"`
	TeamName       string      `json:"team_name" validate:"required,min=3,max=100"`
	Namespace      string      `json:"namespace" validate:"required,min=3,max=100"`
	ContactEmail   string      `json:"contact_email" validate:"required,email"`
	ConfigEndpoint string      `json:"config_endpoint" validate:"required,url"`
	WebhookURL     string      `json:"webhook_url" validate:"required,url"`
	Metadata       interface{} `json:"metadata"`
}

func NewRegistrationRequestHandler() RegistrationRequestHandler {
	return RegisterServiceRequest{}
}

func (r RegisterServiceRequest) Bind(c *echo.Context, request interface{}, model interface{}) error {
	var err error

	if err = c.Bind(request); err != nil {
		log.Println("Error in reading request", err.Error())
		return err
	}
	if err = c.Validate(request); err != nil {
		log.Println("error in validating request", err.Error())
		return err
	}
	switch request.(type) {
	case *RegisterServiceRequest:
		req := request.(*RegisterServiceRequest)
		model.(*models.RegisteredService).Owner = req.Owner
		model.(*models.RegisteredService).Repository = req.Repository
		model.(*models.RegisteredService).ConfigPaths = req.ConfigPaths
		model.(*models.RegisteredService).Branch = req.Branch
		if meta, ok := req.Metadata.(map[string]interface{}); ok {
			model.(*models.RegisteredService).Metadata = models.JSONMap(meta)
		} else {
			model.(*models.RegisteredService).Metadata = models.JSONMap{}
		}
		model.(*models.RegisteredService).InstallationID = req.InstallationID
	case *RegisterServiceRequestV2:
		req := request.(*RegisterServiceRequestV2)
		model.(*models.RegisterServiceV2).ServiceID = GenerateServiceID()
		model.(*models.RegisterServiceV2).ServiceName = req.ServiceName
		model.(*models.RegisterServiceV2).TeamName = req.TeamName
		model.(*models.RegisterServiceV2).Namespace = req.Namespace
		model.(*models.RegisterServiceV2).ContactEmail = req.ContactEmail
		model.(*models.RegisterServiceV2).ConfigEndpoint = req.ConfigEndpoint
		model.(*models.RegisterServiceV2).WebhookURL = req.WebhookURL
		if meta, ok := req.Metadata.(map[string]interface{}); ok {
			model.(*models.RegisterServiceV2).Metadata = models.JSONMap(meta)
		} else {
			model.(*models.RegisterServiceV2).Metadata = models.JSONMap{}
		}

	default:
		return fmt.Errorf("invalid request type")
	}
	return nil
}

func GenerateServiceID() string {
	bytes := make([]byte, 4)
	rand.Read(bytes)
	return fmt.Sprintf("srv_%s", hex.EncodeToString(bytes))
}
