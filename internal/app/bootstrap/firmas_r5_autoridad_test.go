package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestAutoridadFirmaR5ExternaConservaPerfilEntreTresRutas(t *testing.T) {
	e := nuevoEscenarioFirmaExternaV2Prueba(t)
	a := &autoridadFirmaR5Externa{soporte: e.soporte, perfil: e.perfil}
	if org, err := a.ResolverOrganizacionFirmaVec(e.ctx(httpinterno.RutaOriginalFirmableCT, http.MethodPost)); err != nil || org != organizacionAltaContratacionTemporalDesarrollo {
		t.Fatalf("original: %s, %v", org, err)
	}
	if org, err := a.ResolverOrganizacionFirmaExterna(e.ctx(httpinterno.RutaRegistroFirmaExterna, http.MethodPost)); err != nil || org != organizacionAltaContratacionTemporalDesarrollo {
		t.Fatalf("registro: %s, %v", org, err)
	}
	ctx := e.ctx(httpinterno.RutaPreflightFirmaR5, http.MethodPost)
	canal, err := a.ResolverContextoCanalCircuitoRRHH(ctx)
	if err != nil || canal.PerfilRef != e.perfil.perfilRef() {
		t.Fatalf("preparación: %+v, %v", canal, err)
	}
	s := ports.SolicitudResolverContextoAutorizacionAltaV3{AutenticacionRef: canal.AutenticacionRef,
		SesionRef: canal.SesionRef, PerfilRef: canal.PerfilRef}
	if c, err := a.ResolverContextoAutorizacionAltaV3(ctx, s); err != nil ||
		c.Resultado.Contexto.PerfilActivoRef != canal.PerfilRef {
		t.Fatalf("contexto distinto entre canal y preparación: %v", err)
	}
	s.PerfilRef = "prf_otro_perfil_000000000"
	if _, err := a.ResolverContextoAutorizacionAltaV3(ctx, s); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) {
		t.Fatalf("cambio de perfil aceptado: %v", err)
	}
}

func TestAutoridadFirmaR5ExternaDeniegaCrucesMetodosYRevocacion(t *testing.T) {
	e := nuevoEscenarioFirmaExternaV2Prueba(t)
	a := &autoridadFirmaR5Externa{soporte: e.soporte, perfil: e.perfil}
	for _, ruta := range []string{httpinterno.RutaRegistroFirmaVec, httpinterno.RutaRegistroFirmaExterna, httpinterno.RutaPreflightFirmaR5} {
		if _, err := a.ResolverOrganizacionFirmaVec(e.ctx(ruta, http.MethodPost)); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) {
			t.Fatalf("original aceptó ruta ajena %s: %v", ruta, err)
		}
	}
	if _, err := a.ResolverOrganizacionFirmaExterna(e.ctx(httpinterno.RutaRegistroFirmaExterna, http.MethodGet)); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) {
		t.Fatalf("registro por GET: %v", err)
	}
	e.soporte.autoridadAsignaciones.(*autoridadAsignacionesContratacionTemporalDesarrolloPrueba).asignaciones = nil
	if _, err := a.ResolverOrganizacionFirmaExterna(e.ctx(httpinterno.RutaRegistroFirmaExterna, http.MethodPost)); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) {
		t.Fatalf("asignación retirada: %v", err)
	}
	if _, err := (*autoridadFirmaR5Externa)(nil).ResolverOrganizacionFirmaExterna(context.Background()); !errors.Is(err, ports.ErrRegistroFirmaDocumentoNoDisponible) {
		t.Fatalf("autoridad ausente: %v", err)
	}
}
