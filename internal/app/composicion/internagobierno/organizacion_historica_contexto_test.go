package internagobierno

import (
	"context"
	"testing"

	"vec-diputacion-granada/internal/vec/adapters/httpapi"
)

func TestContextoOrganizacionHistoricaNoHeredaSelloB2(t *testing.T) {
	f, err := NuevaFuenteF1(configuracionFuentePrueba())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.WithValue(context.Background(), claveContextoPersonalB2{}, contextoPersonalB2Sellado{fuente: f})
	if _, _, _, err := f.ContextoVinculadoOrganizacionHistorica(ctx); err == nil {
		t.Fatal("sello B2 admitido como histórico")
	}
	a, _ := NuevaAutoridadRutaSeguimiento(f)
	if err := a.AutorizarRutaExacta(ctx, httpapi.RutaOrganizacionHistoricaPersonal); err == nil {
		t.Fatal("autoridad aceptó sello de otra capacidad")
	}
}

func TestAmbitoOrganizacionHistoricaVersionYDominioCerrados(t *testing.T) {
	a := AmbitoOrganizacionHistorica{PerfilActivoRef: "prf_0123456789abcdef0123456789abcdef", PerfilVersion: 1, OrganismoRef: "org_prueba", UnidadClave: "centro_prueba"}
	if err := a.Validar(); err != nil {
		t.Fatal(err)
	}
	a.PerfilVersion = 0
	if a.Validar() == nil {
		t.Fatal("versión ausente aceptada")
	}
	a.PerfilVersion = 1
	a.OrganismoRef = "org*"
	if a.Validar() == nil {
		t.Fatal("comodín positivo aceptado")
	}
}
