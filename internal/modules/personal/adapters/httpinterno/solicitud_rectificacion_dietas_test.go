package httpinterno

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
)

type casoRectificacionPrueba struct{ llamadas int }

func (c *casoRectificacionPrueba) Ejecutar(context.Context, personaldomain.SolicitudRectificacionDietas) (personalports.ResultadoRectificacionDietas, error) {
	c.llamadas++
	return personalports.ResultadoRectificacionDietas{}, errors.New("caso no esperado")
}

type listaRectificacionPrueba struct{ llamadas int }

func (c *listaRectificacionPrueba) Consultar(context.Context, personaldomain.SolicitudRectificacionesCompetentesDietas) (personalports.ResultadoConsultaRectificacionesCompetentesDietas, error) {
	c.llamadas++
	return personalports.ResultadoConsultaRectificacionesCompetentesDietas{}, errors.New("lista no esperada")
}

type auditoriaRectificacionPrueba struct {
	orden    personalports.OrdenAuditoriaFronteraRectificacionDietas
	fallar   bool
	llamadas int
}

func (a *auditoriaRectificacionPrueba) RegistrarAuditoriaFronteraRectificacionDietas(_ context.Context, o personalports.OrdenAuditoriaFronteraRectificacionDietas) error {
	a.llamadas++
	a.orden = o
	if a.fallar {
		return errors.New("auditoria no disponible")
	}
	return nil
}

func TestRectificacionDietasExigeIdentidadYAuditaRechazo(t *testing.T) {
	identidad := &identidadAsignacionPrueba{}
	caso := &casoRectificacionPrueba{}
	auditoria := &auditoriaRectificacionPrueba{}
	m, err := NuevoManejadorRectificacionDietas(identidad, caso, &listaRectificacionPrueba{}, auditoria)
	if err != nil {
		t.Fatal(err)
	}
	cuerpo := `{"relacion_ref":"rel_aaaaaaaaaaaaaaaaaaaaaaaa","unidad_ref":"unidad_sintetica","asignacion_ref":"ads_aaaaaaaaaaaaaaaaaaaaaaaa","version_esperada":1,"fecha_referencia":"2026-09-20","clave_idempotencia":"12345678-1234-4123-8123-123456789abc","campos_a_revisar":["centro_ref"],"motivo_revision":"centro incorrecto"}`
	r := httptest.NewRequest(http.MethodPost, RutaSolicitudesRectificacionDietas, strings.NewReader(cuerpo))
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Content-Type", "application/json; charset=utf-8")
	w := httptest.NewRecorder()
	m.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized || identidad.llamadas != 1 || caso.llamadas != 0 || auditoria.llamadas != 1 || auditoria.orden.Accion != "solicitar" {
		t.Fatalf("frontera: estado=%d identidad=%d caso=%d auditoria=%+v", w.Code, identidad.llamadas, caso.llamadas, auditoria.orden)
	}
	auditoria.fallar = true
	w = httptest.NewRecorder()
	m.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaSolicitudesRectificacionDietas, nil))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("auditoria caída debe cerrar respuesta: %d", w.Code)
	}
}

func TestListaCompetenteNoAceptaUnidadEnumeradoraDelCliente(t *testing.T) {
	identidad := &identidadAsignacionPrueba{}
	lista := &listaRectificacionPrueba{}
	auditoria := &auditoriaRectificacionPrueba{}
	m, err := NuevoManejadorRectificacionDietas(identidad, &casoRectificacionPrueba{}, lista, auditoria)
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodGet, RutaSolicitudesRectificacionDietas+"/competentes?unidad_ref=ajena", nil)
	r.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	m.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest || identidad.llamadas != 0 || lista.llamadas != 0 || auditoria.orden.Accion != "consultar_competentes" {
		t.Fatalf("enumeración libre: estado=%d identidad=%d lista=%d auditoria=%+v", w.Code, identidad.llamadas, lista.llamadas, auditoria.orden)
	}
	r = httptest.NewRequest(http.MethodGet, RutaSolicitudesRectificacionDietas+"/competentes", nil)
	r.Header.Set("Accept", "application/json")
	w = httptest.NewRecorder()
	m.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized || identidad.llamadas != 1 || lista.llamadas != 0 {
		t.Fatalf("sin identidad: estado=%d identidad=%d lista=%d", w.Code, identidad.llamadas, lista.llamadas)
	}
}
