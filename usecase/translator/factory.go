package translator

import (
	"log"
	"os"
	"the_governor/usecase"
)

// NewTranslatorFromEnv reads GATEWAY_PROVIDER and returns the matching translator.
// Callers (main.go, webhook, operator) should use this instead of hardcoding the impl.
func NewTranslatorFromEnv(namespace string) usecase.GatewayTranslator {
	provider := os.Getenv("GATEWAY_PROVIDER")
	switch provider {
	case "GLOO_EDGE":
		log.Println("selecting Gloo Edge translator")
		return NewGlooEdgeTranslator(namespace)
	default:
		log.Println("selecting Base (Gateway API) translator")
		return NewBaseGatewayTranslator(namespace)
	}
}
