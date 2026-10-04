package interna

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"vec-diputacion-granada/internal/vec/adapters/httpapi"
	ports "vec-diputacion-granada/internal/vec/ports"
)

type auditorFronteraOH struct {
	orden    ports.OrdenAuditoriaFronteraRutaExacta
	llamadas int
	err      error
}

func (a *auditorFronteraOH) RegistrarAuditoriaFronteraRutaExacta(_ context.Context, o ports.OrdenAuditoriaFronteraRutaExacta) error {
	a.orden = o
	a.llamadas++
	return a.err
}

func TestPuenteAuditoriaOHConservaSuperficieYFallaCerrado(t *testing.T) {
	a := &auditorFronteraOH{}
	p := &puenteConsultaSeguimiento{auditoria: a}
	w := httptest.NewRecorder()
	p.responderAutenticacionRequerida(w, context.Background(), httpapi.RutaOrganizacionHistoricaPersonal)
	if w.Code != http.StatusUnauthorized || a.orden.Superficie != ports.SuperficieAuditoriaFronteraRutaExactaOrganizacionHistoricaPersonal || a.orden.Ruta != httpapi.RutaOrganizacionHistoricaPersonal || a.orden.ActorRef != "" {
		t.Fatalf("401 minimizado: estado%d orden%v", w.Code, a.orden)
	}
	if !p.auditarDenegacionPersonalB2(context.Background(), httpapi.RutaOrganizacionHistoricaPersonal) || a.orden.Motivo != ports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado || a.orden.Ruta != httpapi.RutaOrganizacionHistoricaPersonal {
		t.Fatal("403 OH debe usar su ruta nominal")
	}
	a.err = errors.New("auditoria no disponible")
	w = httptest.NewRecorder()
	p.responderAutenticacionRequerida(w, context.Background(), httpapi.RutaOrganizacionHistoricaPersonal)
	if w.Code != http.StatusServiceUnavailable || p.auditarDenegacionPersonalB2(context.Background(), httpapi.RutaOrganizacionHistoricaPersonal) {
		t.Fatal("dependencia caída no confirma una denegación auditada")
	}
}
