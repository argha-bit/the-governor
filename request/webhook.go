package request

import (
	"fmt"
	"log"

	"github.com/labstack/echo/v5"
)

type WebhookRequest struct {
	Owner      string `json:"owner"`
	Repository string `json:"repository"`
	Branch     string `json:"branch"`
	CommitSHA  string `json:"commit_sha"`
}
type WebhookRequestV2 struct {
	ServiceID string `json:"service_id"`
}
type WebhookRequestHandler interface {
	Bind(c *echo.Context, request interface{}) error
}

func NewWebhookRequestHandler() WebhookRequestHandler {
	return &WebhookRequest{}
}

func (w *WebhookRequest) Bind(c *echo.Context, request interface{}) error {
	var err error
	if err = c.Bind(request); err != nil {
		log.Println("Error in reading request", err.Error())
		return err
	}
	switch request.(type) {
	case *WebhookRequest:
		req := request.(*WebhookRequest)
		if req.Owner == "" {
			return fmt.Errorf("owner is required")
		}
		if req.Repository == "" {
			return fmt.Errorf("repository is required")
		}
		if req.CommitSHA == "" {
			return fmt.Errorf("commit_sha is required")
		}
	case *WebhookRequestV2:
		req := request.(*WebhookRequestV2)
		log.Printf("Received WebhookRequestV2: ServiceID=%s", req.ServiceID)
	default:
		return fmt.Errorf("invalid request type")
	}
	return nil
}
