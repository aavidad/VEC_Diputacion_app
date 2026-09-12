package bootstrap

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	httpct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
)

func TestNuevoServicioCierreAdministrativoSinCeseDesarrolloFallaCerradoSinProveedor(t *testing.T) {
	if servicio, err := NuevoServicioCierreAdministrativoSinCeseDesarrollo(ConfiguracionCierreAdministrativoSinCeseDesarrollo{}); servicio != nil || err == nil {
		t.Fatalf("configuracion incompleta aceptada: servicio=%#v err=%v", servicio, err)
	}
}

type detallePreparacionCierrePrueba struct {
	consultar func(context.Context, ct.SolicitudDetalleRRHH) (ct.DetalleExpedienteRRHH, error)
}

func (d detallePreparacionCierrePrueba) Consultar(ctx context.Context, s ct.SolicitudDetalleRRHH) (ct.DetalleExpedienteRRHH, error) {
	return d.consultar(ctx, s)
}

type lectorPreparacionCierrePrueba struct {
	consultar func(context.Context, ct.SolicitudPreparacionCierreAdministrativo) (ct.PreparacionCierreAdministrativo, error)
}

func (l lectorPreparacionCierrePrueba) ConsultarPreparacionCierreAdministrativo(ctx context.Context, s ct.SolicitudPreparacionCierreAdministrativo) (ct.PreparacionCierreAdministrativo, error) {
	return l.consultar(ctx, s)
}

func TestContinuidadNominalPreparacionCierreExigeDetalleAntesDelLector(t *testing.T) {
	alta, consultas, principal := escenarioConsultasRRHHDesarrolloPrueba(t)
	v, _ := alta.soporte.contexto.Vinculo.Datos()
	a := &autoridadContinuidadNominal{soporte: alta.soporte, consultas: consultas, referencias: ReferenciasCTIncorporacionDesarrollo{PrincipalV3Ref: v.PrincipalID, PerfilV3Ref: v.PerfilActivoRef, OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo}, accion: ct.AccionAutorizacionCerrarAdministrativamente, reloj: alta.soporte.reloj}
	ctx := contextoRutaConsultasRRHHDesarrolloPrueba(alta.soporte, principal, httpct.RutaPreparacionCierreSinCese)
	ctx, e := contextoDetalleIncorporacionV2Desarrollo(ctx, alta.soporte)
	if e != nil {
		t.Fatal(e)
	}
	ctx = context.WithValue(ctx, claveRutaContinuidadNominal{}, httpct.RutaPreparacionCierreSinCese)
	solicitud := ct.SolicitudPreparacionCierreAdministrativo{OrganizacionRef: a.referencias.OrganizacionRef, ExpedienteRef: "expediente:prueba", SeguimientoRef: "seguimiento:prueba"}
	orden := []string{}
	denegar := false
	otroExpediente := false
	l := &lectorPreparacionCierreNominal{autoridad: a, detalle: detallePreparacionCierrePrueba{func(_ context.Context, s ct.SolicitudDetalleRRHH) (ct.DetalleExpedienteRRHH, error) {
		orden = append(orden, "detalleV3")
		if s.ExpedienteRef() != solicitud.ExpedienteRef || s.VersionObservada() != 0 {
			t.Fatal("consulta de expediente o versión incorrecta")
		}
		if denegar {
			return ct.DetalleExpedienteRRHH{}, ct.ErrAutorizacionDenegada
		}
		exp := s.ExpedienteRef()
		if otroExpediente {
			exp = "expediente:ajeno"
		}
		return ct.DetalleExpedienteRRHH{Resumen: ct.ResumenExpedienteRRHH{ExpedienteRef: exp, OrganizacionRef: solicitud.OrganizacionRef}}, nil
	}}, lector: lectorPreparacionCierrePrueba{func(_ context.Context, s ct.SolicitudPreparacionCierreAdministrativo) (ct.PreparacionCierreAdministrativo, error) {
		orden = append(orden, "lectorCT87")
		if s != solicitud {
			t.Fatal("selector alterado")
		}
		return ct.PreparacionCierreAdministrativo{ExpedienteRef: s.ExpedienteRef, SeguimientoRef: s.SeguimientoRef, VersionActual: 2, EstadoActual: "cerrado_administrativamente", Acciones: []ct.AccionPreparacionCierreAdministrativo{}, PreparadaEn: alta.soporte.reloj.Ahora()}, nil
	}}}
	p, e := l.ConsultarPreparacionCierreAdministrativo(ctx, solicitud)
	if e != nil {
		t.Fatal(e)
	}
	if p.VersionActual != 2 || len(p.Acciones) != 0 || strings.Join(orden, ",") != "detalleV3,lectorCT87" {
		t.Fatal("no conserva estado actual o secuencia de autorización")
	}
	// También atravesamos el HTTP real con la cápsula nominal de prueba.
	h, err := httpct.NuevoManejadorPreparacionCierreSinCese(a, l)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, httpct.RutaPreparacionCierreSinCese+"?expediente_ref="+solicitud.ExpedienteRef+"&seguimiento_ref="+solicitud.SeguimientoRef, nil).WithContext(ctx)
	respuesta := httptest.NewRecorder()
	h.ServeHTTP(respuesta, req)
	if respuesta.Code != http.StatusOK || !strings.Contains(respuesta.Body.String(), `"version_actual":2`) {
		t.Fatalf("HTTP preparación no conserva estado actual: %d %s", respuesta.Code, respuesta.Body.String())
	}
	for _, modo := range []string{"denegacion", "expediente_ajeno"} {
		orden = nil
		denegar = modo == "denegacion"
		otroExpediente = modo == "expediente_ajeno"
		if _, e = l.ConsultarPreparacionCierreAdministrativo(ctx, solicitud); e == nil {
			t.Fatal("detalle inválido admitido")
		}
		if strings.Join(orden, ",") != "detalleV3" {
			t.Fatal("lector privado invocado sin lectura autorizada")
		}
	}
	orden = nil
	ajena := solicitud
	ajena.OrganizacionRef = "organizacion:ajena"
	if _, e = l.ConsultarPreparacionCierreAdministrativo(ctx, ajena); e == nil || len(orden) != 0 {
		t.Fatal("selector ajeno alcanzó consultas")
	}
	// El GET no puede conceder escritura ni aun con un objeto solicitud V3.
	if _, _, e := a.ExigirSolicitudLigadaV3(ctx, core.SolicitudAutorizacionLigadaV3{}, core.ResultadoContextoActorRegistradoV2{}); e == nil {
		t.Fatal("preparación habilitó escritura")
	}
}
