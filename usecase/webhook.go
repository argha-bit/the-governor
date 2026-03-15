package usecase

import "the_governor/request"

type WebhookUsecaseHandler interface {
	HandleWebhook(request *request.WebhookRequest) error
	HandleWebhookV2(request *request.WebhookRequestV2) error
}
