package bootstrap

import (
	"context"
	"errors"
	"testing"

	"vec-diputacion-granada/config"
)

func TestBolsasPublicasPortalExternoFallaCerradoConConfiguracionParcial(t *testing.T) {
	casos := []struct {
		nombre string
		cfg    config.Config
		dsn    string
	}{
		{"falta DSN", config.Config{BolsaPublicaManifiestoSHA256: huellaBolsasPublicasPrueba()}, ""},
		{"falta manifiesto", config.Config{}, "postgres://vec_externo_bolsa_publica@localhost/publica"},
		{"credencial publica antigua", config.Config{BolsaPublicaPostgreSQL: configuracionPublicaPrueba(t), BolsaPublicaManifiestoSHA256: huellaBolsasPublicasPrueba()}, "postgres://vec_externo_bolsa_publica@localhost/publica"},
		{"credencial publica antigua sin manifiesto", config.Config{BolsaPublicaPostgreSQL: configuracionPublicaPrueba(t)}, ""},
		{"falta huella de categorias", config.Config{BolsaPublicaManifiestoSHA256: huellaBolsasPublicasPrueba()}, "postgres://vec_externo_bolsa_publica@localhost/publica"},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			manejador, cerrar, err := nuevasBolsasPublicasPortalExterno(context.Background(), caso.cfg, caso.dsn)
			if !errors.Is(err, ErrBolsasPublicasPortalExternoNoDisponibles) || manejador != nil || cerrar == nil {
				t.Fatalf("configuracion parcial aceptada: manejador=%v cerrar=%v error=%v", manejador, cerrar != nil, err)
			}
		})
	}
}

func TestBolsasPublicasPortalExternoExigeLoginPropio(t *testing.T) {
	huella := huellaBolsasPublicasPrueba()
	cfg := config.Config{
		BolsaCategoriesCatalogID: "rpt", BolsaCategoriesVersion: 1,
		BolsaCategoriesSHA256: huella, BolsaCategoriesPublicProjectionSHA256: huella,
		BolsaPublicaManifiestoSHA256: huella,
	}
	h, cerrar, err := nuevasBolsasPublicasPortalExterno(context.Background(), cfg,
		"postgres://vec_publico_login@localhost/publica?sslmode=verify-full")
	if !errors.Is(err, ErrBolsasPublicasPortalExternoNoDisponibles) || h != nil || cerrar == nil {
		t.Fatalf("login publico interno aceptado: manejador=%v cerrar=%v error=%v", h, cerrar != nil, err)
	}
}

func huellaBolsasPublicasPrueba() string {
	return "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
}

func configuracionPublicaPrueba(t *testing.T) config.ConfiguracionPostgreSQLPublica {
	t.Helper()
	cfg, err := config.NuevaConfiguracionPostgreSQLPublica("postgres://vec_publico@localhost/publica")
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}
