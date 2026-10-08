package bootstrap

import (
	"errors"
	"testing"

	"vec-diputacion-granada/config"
)

func configFirmaR5ExternaPrueba() config.Config {
	return config.Config{ExecutionProfile: config.ExecutionProfileDevelopment,
		AuthMode: config.AuthModeDevelopment, DevelopmentGuard: config.DevelopmentGuardAcknowledgement}
}

func TestFirmaR5ExternaMontajeApagadoNoPublicaRutas(t *testing.T) {
	t.Setenv(envCTFirmaR5ExternaEnabled, "false")
	r, err := nuevasRutasFirmaR5ExternaCT(config.Config{}, insumosFirmaR5ExternaCT{}, nil, nil, nil)
	if err != nil || r != nil {
		t.Fatalf("selector apagado publicó rutas R5 externas: %v, %#v", err, r)
	}
}

func TestFirmaR5ExternaMontajeExigeConfiguracionYFuentes(t *testing.T) {
	t.Setenv(envCTFirmaR5ExternaEnabled, "true")
	if r, err := nuevasRutasFirmaR5ExternaCT(config.Config{}, insumosFirmaR5ExternaCT{}, nil, nil, nil); r != nil ||
		!errors.Is(err, ErrActivacionDesarrolloInvalida) {
		t.Fatalf("selector público admitido: %v, %#v", err, r)
	}
	base := baseR5MontajePrueba(t)
	firma := &firmaDocumentoCTDesarrollo{servicio: base, custodiaR5Compuesta: true}
	r, err := nuevasRutasFirmaR5ExternaCT(configFirmaR5ExternaPrueba(),
		insumosFirmaR5ExternaCT{firma: firma}, nil, nil, nil)
	if r != nil || !errors.Is(err, errFirmaR5ExternaMontajeNoDisponible) {
		t.Fatalf("montaje sin fuentes admitido: %v, %#v", err, r)
	}
	if firma.servicio != base || firma.firmaExterna != nil || firma.firmaVec != nil || firma.registroR5 != nil {
		t.Fatal("montaje fallido cambió el servicio anterior")
	}
}

func TestFirmaR5ExternaMontajeSelectorInvalidoDeniega(t *testing.T) {
	t.Setenv(envCTFirmaR5ExternaEnabled, "1")
	if r, err := nuevasRutasFirmaR5ExternaCT(configFirmaR5ExternaPrueba(), insumosFirmaR5ExternaCT{}, nil, nil, nil); r != nil ||
		!errors.Is(err, ErrActivacionDesarrolloInvalida) {
		t.Fatalf("selector inválido admitido: %v, %#v", err, r)
	}
}
