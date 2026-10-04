package internagobierno

import (
	"context"
	"reflect"
	"testing"
	"time"

	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/httpapi"
	core "vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/pruebas"
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

func TestContextoOriginalOHCaducadoSoloSirveParaAuditoria(t *testing.T) {
	ahora := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	r, v, err := pruebas.NuevoContextoRegistradoYVinculoV2(ahora, "per_0123456789abcdefghijkl", "prf_0123456789abcdefghijkl", core.AuthMethodCertificate, core.AuthAssuranceSubstantial)
	if err != nil {
		t.Fatal(err)
	}
	f, err := NuevaFuenteF1(configuracionFuentePrueba())
	if err != nil {
		t.Fatal(err)
	}
	scope := AmbitoOrganizacionHistorica{PerfilActivoRef: r.Contexto.PerfilActivoRef, PerfilVersion: r.Contexto.Instantanea.PerfilVersion, OrganismoRef: "dipgra"}
	ctx := context.WithValue(context.Background(), claveContextoOrganizacionHistorica{}, contextoOrganizacionHistoricaSellado{fuente: f, valor: ct.ContextoAutorizacionAltaV3{Resultado: r, Vinculo: v}, ambito: scope})
	if _, _, _, err := f.ContextoVinculadoOrganizacionHistorica(ctx); err != nil {
		t.Fatal(err)
	}
	datos, _ := v.Datos()
	f.reloj = relojPrueba{datos.SesionValidaHasta}
	if _, _, _, err := f.ContextoVinculadoOrganizacionHistorica(ctx); err == nil {
		t.Fatal("captura histórica concede lectura caducada")
	}
	got, org, _, err := f.ContextoOriginalOrganizacionHistoricaParaAuditoria(ctx)
	if err != nil || org != "dipgra" || !reflect.DeepEqual(got.Resultado, r) {
		t.Fatal("identidad original perdida al vencer")
	}
	got.Resultado.RepresentacionCanonica[0] = '!'
	again, _, _, err := f.ContextoOriginalOrganizacionHistoricaParaAuditoria(ctx)
	if err != nil || again.Resultado.Validar() != nil {
		t.Fatal("captura no defensiva")
	}
	otra, err := NuevaFuenteF1(configuracionFuentePrueba())
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := otra.ContextoOriginalOrganizacionHistoricaParaAuditoria(ctx); err == nil {
		t.Fatal("sello de otra fuente aceptado")
	}
	if _, _, _, err := f.ContextoOriginalOrganizacionHistoricaParaAuditoria(context.Background()); err == nil {
		t.Fatal("captura sin sello aceptada")
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
