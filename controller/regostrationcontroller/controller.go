package registrationcontroller

import (
	"log"
	"net/http"
	"the_governor/controller"
	"the_governor/models"
	"the_governor/request"
	"the_governor/usecase"

	"github.com/labstack/echo/v5"
)

type Controller struct {
	req request.RegistrationRequestHandler
	uc  usecase.RegistrationServiceUsecaseHandler
}

func (c *Controller) Register(ctx *echo.Context) error {
	var err error
	req := new(request.RegisterServiceRequest)
	model := new(models.RegisteredService)
	if err = c.req.Bind(ctx, req, model); err != nil {
		log.Println("Error binding request:", err.Error())
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	log.Printf("Model is %+v", model)
	if err = c.uc.RegisterService(model); err != nil {
		log.Println("Error registering service:", err.Error())
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to register service"})
	}
	return ctx.JSON(http.StatusCreated, map[string]string{"message": "Service registered successfully", "owner": model.Owner, "repository": model.Repository})
}

func (c *Controller) RegisterV2(ctx *echo.Context) error {
	var err error
	req := new(request.RegisterServiceRequestV2)
	model := new(models.RegisterServiceV2)
	if err = c.req.Bind(ctx, req, model); err != nil {
		log.Println("Error binding request:", err.Error())
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	log.Printf("Model is %+v", model)
	if err = c.uc.RegisterServiceV2(model); err != nil {
		log.Println("Error registering service:", err.Error())
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return ctx.JSON(http.StatusCreated, map[string]string{"message": "Service registered successfully", "service_id": model.ServiceID, "service_name": model.ServiceName, "team_name": model.TeamName})
}

func NewRegistrationController(e *echo.Echo, req request.RegistrationRequestHandler, uc usecase.RegistrationServiceUsecaseHandler) controller.RegistrationController {
	registrationController := &Controller{
		req: req,
		uc:  uc,
	}

	e.POST("/v1/register", registrationController.Register)
	e.POST("/v2/register", registrationController.RegisterV2)

	return registrationController
}
