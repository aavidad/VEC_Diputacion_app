package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
)

type consultorListaB2Prueba struct {
	resultado personalports.ResultadoEmpleadosB2
	solicitud personaldomain.SolicitudEmpleadosB2
	err       error
	llamadas  int
}

func (c *consultorListaB2Prueba) ConsultarEmpleados(_ context.Context, s personaldomain.SolicitudEmpleadosB2) (personalports.ResultadoEmpleadosB2, error) {
	c.llamadas++
	c.solicitud = s
	return c.resultado, c.err
}

func TestListaEmpleadosB2HTTPPublicaPaginaDelOrganismoServidor(t *testing.T) {
	actor := actorOrganizacionHistoricaPrueba(t)
	a := &autoridadRegistroEmpleadoB2Prueba{actor: actor, organismo: "org_prueba"}
	instante := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	c := &consultorListaB2Prueba{resultado: personalports.ResultadoEmpleadosB2{
		Pagina: personaldomain.PaginaEmpleadosB2{OrganismoRef: "org_prueba", Corte: personaldomain.CorteEmpleadoB2{VigenteEn: "2026-09-25", ConocidoEn: instante}, Limite: 25,
			Empleados: []personaldomain.EmpleadoOrganismoB2{{EmpleadoRef: empRefRegistroEmpleadoB2Prueba, Relaciones: []personaldomain.RelacionVigenteEmpleadoB2{}}}},
		Evidencia: personalports.EvidenciaRegistroEmpleadoB2{ReciboRef: "recibo_1", DecisionRef: "decision_1", EfectoRef: "org_prueba", ConsumoHuellaSHA256: strings.Repeat("a", 64), AuditoriaRef: "audit_1", ConsultadaEn: instante},
	}}
	audit := &auditorRegistroEmpleadoB2Prueba{}
	h, err := NewHandlerEmpleadosOrganismoB2(a, c, audit)
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodGet, RutaEmpleadosOrganismoB2+"?"+queryRegistroEmpleadoB2Prueba+"&limite=25", nil)
	r.Header.Set("X-VEC-Organismo", "org_ajeno")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 || c.solicitud.OrganismoRef != "org_prueba" || c.solicitud.Limite != 25 || c.solicitud.Actor.Principal.ID != actor.Principal.ID ||
		!strings.Contains(w.Body.String(), `"empleados"`) || strings.Contains(w.Body.String(), "per_") || w.Header().Get("Set-Cookie") != "" {
		t.Fatalf("estado=%d solicitud=%+v cuerpo=%s", w.Code, c.solicitud, w.Body.String())
	}
	c.resultado.Pagina.OrganismoRef = "org_ajeno"
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaEmpleadosOrganismoB2+"?"+queryRegistroEmpleadoB2Prueba+"&limite=25", nil))
	if w.Code != 503 || strings.Contains(w.Body.String(), "org_ajeno") {
		t.Fatalf("página de otro organismo publicada: %d %s", w.Code, w.Body.String())
	}
}

func TestListaEmpleadosB2HTTPRechazaYAuditaConRutaFija(t *testing.T) {
	a := &autoridadRegistroEmpleadoB2Prueba{actor: actorOrganizacionHistoricaPrueba(t), organismo: "org_prueba"}
	c, audit := &consultorListaB2Prueba{}, &auditorRegistroEmpleadoB2Prueba{}
	h, _ := NewHandlerEmpleadosOrganismoB2(a, c, audit)
	for _, caso := range []struct {
		metodo, ruta string
		estado       int
	}{
		{http.MethodPost, RutaEmpleadosOrganismoB2 + "?" + queryRegistroEmpleadoB2Prueba, 405},
		{http.MethodGet, RutaEmpleadosOrganismoB2 + "?" + queryRegistroEmpleadoB2Prueba + "&organismo_ref=otro", 400},
		{http.MethodGet, RutaEmpleadosOrganismoB2 + "?" + queryRegistroEmpleadoB2Prueba + "&limite=101", 400},
		{http.MethodGet, RutaEmpleadosOrganismoB2 + "/otra?" + queryRegistroEmpleadoB2Prueba, 404},
	} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(caso.metodo, caso.ruta, nil))
		if w.Code != caso.estado || c.llamadas != 0 {
			t.Fatalf("%s %s: %d llamadas=%d", caso.metodo, caso.ruta, w.Code, c.llamadas)
		}
	}
	c.err = personaldomain.ErrRegistroEmpleadoB2Denegado
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaEmpleadosOrganismoB2+"?"+queryRegistroEmpleadoB2Prueba, nil))
	if w.Code != 403 || len(audit.ordenes) != 1 || audit.ordenes[0].Ruta != RutaEmpleadosOrganismoB2 || c.solicitud.Limite != 50 {
		t.Fatalf("denegación: %d auditoría=%+v solicitud=%+v", w.Code, audit.ordenes, c.solicitud)
	}
	if RutaAuditoriaRegistroEmpleadoB2(PrefijoFichaEmpleadoB2+empRefRegistroEmpleadoB2Prueba) != PrefijoFichaEmpleadoB2+"{emp_ref}" {
		t.Fatal("la auditoría guardaría la referencia del empleado")
	}
	if h, err := NewHandlerEmpleadosOrganismoB2(a, nil, audit); h != nil || err == nil {
		t.Fatal("lista sin consultor montada")
	}
}
