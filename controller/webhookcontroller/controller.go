package webhookcontroller

import (
	"log"
	"net/http"
	"the_governor/controller"
	"the_governor/request"
	"the_governor/usecase"

	"github.com/labstack/echo/v5"
)

type Controller struct {
	req request.WebhookRequestHandler
	uc  usecase.WebhookUsecaseHandler
}

func (c *Controller) HandleWebhook(ctx *echo.Context) error {
	var err error
	req := new(request.WebhookRequest)
	if err = c.req.Bind(ctx, req); err != nil {
		return ctx.JSON(400, map[string]string{"error": err.Error()})
	}
	go func() {
		if err = c.uc.HandleWebhook(req); err != nil {
			// Log the error but do not return it in the response since the webhook has already been acknowledged
			log.Println("Error handling webhook:", err.Error())
		}
	}()
	return ctx.JSON(202, map[string]string{"message": "Webhook received"})
}
func (c *Controller) HandleWebhookV2(ctx *echo.Context) error {
	var err error
	req := new(request.WebhookRequestV2)
	if err = c.req.Bind(ctx, req); err != nil {
		return ctx.JSON(400, map[string]string{"error": err.Error()})
	}
	//TODO: Verify if the service ID is valid and then process the webhook accordingly in a goroutine
	if err = c.uc.HandleWebhookV2(req); err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return ctx.JSON(202, map[string]string{"message": "Webhook V2 received"})
}

func NewWebhookController(e *echo.Echo, req request.WebhookRequestHandler, uc usecase.WebhookUsecaseHandler) controller.WebhookController {
	webhookController := &Controller{
		req: req,
		uc:  uc,
	}

	e.POST("/webhook", webhookController.HandleWebhook)
	e.POST("/v2/webhook", webhookController.HandleWebhookV2) // New endpoint for WebhookRequestV2

	return webhookController
}
