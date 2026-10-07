package bootstrap

import (
	"errors"
	"net/http"
	"testing"

	"vec-diputacion-granada/config"
)

func TestComposicionesNormalesRechazanCualquierSelectorPresentacion(t *testing.T) {
	selectores := []config.Config{
		{ExecutionProfile: config.ExecutionProfileRRHHPresentation},
		{RRHHPresentationEnabled: true},
		{RRHHPresentationGuardOne: config.RRHHPresentationGuardOneAcknowledgement},
		{RRHHPresentationGuardTwo: config.RRHHPresentationGuardTwoAcknowledgement},
	}
	for _, cfg := range selectores {
		for _, constructor := range []func(config.Config) (*http.Server, error){NewHTTPServerWithConfig, NewHTTPServerPublicoWithConfig} {
			if _, err := constructor(cfg); !errors.Is(err, ErrPresentacionRRHHEnComposicionNormal) {
				t.Fatalf("selector no rechazado: %v", err)
			}
		}
	}
}
