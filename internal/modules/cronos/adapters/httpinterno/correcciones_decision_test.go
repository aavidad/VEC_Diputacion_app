package httpinterno

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

type resolverDecisionCorreccionPrueba struct{ llamadas int }

func (r *resolverDecisionCorreccionPrueba) ResolverDecisionCorrecciones(*http.Request) (ports.OrdenConsumoCorreccion, error) {
	r.llamadas++
	return ports.OrdenConsumoCorreccion{}, nil
}

type casoDecisionCorreccionPrueba struct {
	peticion ports.PeticionActuacionCorreccion
	llamadas int
	err      error
}

func (c *casoDecisionCorreccionPrueba) ConsultarBandejaCorrecciones(_ context.Context, _ ports.OrdenConsumoCorreccion, paso string) (ports.BandejaCorrecciones, error) {
	c.llamadas++
	return ports.BandejaCorrecciones{Paso: paso, Pendientes: []ports.CorreccionPendiente{}}, c.err
}
func (c *casoDecisionCorreccionPrueba) RegistrarDecisionCorreccion(_ context.Context, _ ports.OrdenConsumoCorreccion, p ports.PeticionActuacionCorreccion) (ports.ReciboCorreccion, error) {
	c.llamadas++
	c.peticion = p
	return ports.ReciboCorreccion{SolicitudRef: p.SolicitudRef, ActuacionRef: "correccion:actuacion:00000000-0000-4000-8000-000000000000",
		ReciboRef: "recibo:cronos:00000000-0000-4000-8000-000000000000", Estado: domain.CorreccionPendienteRRHH, Version: 2, InstanteUTC: time.Now().UTC()}, c.err
}
func (c *casoDecisionCorreccionPrueba) ConsultarHistoriaCorreccion(_ context.Context, _ ports.OrdenConsumoCorreccion, p ports.PeticionHistoriaCorreccion) (ports.HistoriaCorreccion, error) {
	c.llamadas++
	return ports.HistoriaCorreccion{SolicitudRef: p.SolicitudRef, Estado: domain.CorreccionPendienteResponsable, Version: 1,
		Actuaciones: []ports.HistoriaActuacionCorreccion{}, Asientos: []ports.HistoriaAsientoCorreccion{}}, c.err
}

func TestCorreccionDenegadaAuditadaYFalloAuditorNoEs403(t *testing.T) {
	c, r := &casoDecisionCorreccionPrueba{err: ports.ErrResolucionNoCompetente}, &resolverDecisionCorreccionPrueba{}
	h, _ := NuevoManejadorDecisionCorrecciones(c, r)
	const cuerpo = `{"solicitud_ref":"correccion:cronos:solicitud-0001","clave_operacion":"actuacion-0001","paso":"responsable","resultado":"favorable","motivo":"Olvido acreditado","version_esperada":"1"}`
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionJSON(http.MethodPost, RutaActuacionCorrecciones, cuerpo))
	if w.Code != http.StatusForbidden {
		t.Fatalf("rechazo auditado: %d", w.Code)
	}
	c.err = ports.ErrDependenciaNoDisponible
	w = httptest.NewRecorder()
	h.ServeHTTP(w, peticionJSON(http.MethodPost, RutaActuacionCorrecciones, cuerpo))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("auditoría fallida atribuida a 403: %d", w.Code)
	}
}

func TestCorreccionesDecisionRechazaSuplantacionYEntregaPaso(t *testing.T) {
	c, r := &casoDecisionCorreccionPrueba{}, &resolverDecisionCorreccionPrueba{}
	h, err := NuevoManejadorDecisionCorrecciones(c, r)
	if err != nil {
		t.Fatal(err)
	}
	const cuerpo = `{"solicitud_ref":"correccion:cronos:solicitud-0001","clave_operacion":"actuacion-0001","paso":"responsable","resultado":"favorable","motivo":"Olvido acreditado","version_esperada":"1"}`
	for _, x := range []struct {
		ruta, metodo, cuerpo string
		codigo               int
	}{
		{RutaBandejaCorrecciones + "?paso=responsable", http.MethodGet, "", http.StatusOK},
		{RutaBandejaCorrecciones + "?paso=administracion", http.MethodGet, "", http.StatusOK},
		{RutaBandejaCorrecciones + "?paso=responsable&empleado_ref=emp_ajeno", http.MethodGet, "", http.StatusBadRequest},
		{RutaHistoriaCorrecciones + "?solicitud_ref=correccion:cronos:solicitud-0001&vista=propia", http.MethodGet, "", http.StatusOK},
		{RutaHistoriaCorrecciones + "?solicitud_ref=correccion:cronos:solicitud-0001&vista=equipo&paso=responsable", http.MethodGet, "", http.StatusOK},
		{RutaHistoriaCorrecciones + "?solicitud_ref=correccion:cronos:solicitud-0001&vista=propia&paso=responsable", http.MethodGet, "", http.StatusBadRequest},
		{RutaActuacionCorrecciones, http.MethodPost, cuerpo, http.StatusCreated},
		{RutaActuacionCorrecciones, http.MethodPost, strings.Replace(cuerpo, `"1"`, `"01"`, 1), http.StatusBadRequest},
		{RutaActuacionCorrecciones, http.MethodPost, strings.Replace(cuerpo, `}`, `,"actor_ref":"per_ajeno"}`, 1), http.StatusBadRequest},
	} {
		var req *http.Request
		if x.metodo == http.MethodPost {
			req = peticionJSON(x.metodo, x.ruta, x.cuerpo)
		} else {
			req = httptest.NewRequest(x.metodo, x.ruta, nil)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != x.codigo || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("%s: %d %s", x.ruta, w.Code, w.Body.String())
		}
	}
	if c.peticion.Paso != domain.PasoDecisionResponsable || c.peticion.Motivo != "Olvido acreditado" || c.peticion.VersionEsperada != 1 || r.llamadas != 5 {
		t.Fatalf("contrato HTTP divergente: %+v, resolver %d", c.peticion, r.llamadas)
	}
}
