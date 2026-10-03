package httpinterno

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type autoridadCircuitoRRHHPrueba struct {
	llamadas int
}

func (a *autoridadCircuitoRRHHPrueba) ResolverContextoCanalCircuitoRRHH(context.Context) (ContextoCanalCircuitoRRHH, error) {
	a.llamadas++
	return ContextoCanalCircuitoRRHH{
		AutenticacionRef: "aut_aaaaaaaaaaaaaaaaaaaaaaaa",
		SesionRef:        "ses_bbbbbbbbbbbbbbbbbbbbbbbb",
		PerfilRef:        "prf_cccccccccccccccccccccccc",
		OrganizacionRef:  "organizacion:dipgra:circuito:001",
	}, nil
}

type consultorCircuitoRRHHPrueba struct {
	llamadas  int
	solicitud ports.SolicitudConsultaCircuitoRRHH
}

func (c *consultorCircuitoRRHHPrueba) Consultar(_ context.Context, s ports.SolicitudConsultaCircuitoRRHH) (ports.ResultadoConsultaCircuitoRRHH, error) {
	c.llamadas++
	c.solicitud = s
	return ports.ResultadoConsultaCircuitoRRHH{}, ports.ErrConsultaRRHHNoDisponible
}

func TestConsultaCircuitoRRHHTomaIdentidadSoloDelCanal(t *testing.T) {
	autoridad := &autoridadCircuitoRRHHPrueba{}
	consultor := &consultorCircuitoRRHHPrueba{}
	h, err := NuevoManejadorConsultaCircuitoRRHH(autoridad, consultor)
	if err != nil {
		t.Fatal(err)
	}
	nuevaPeticion := func(cuerpo string) *http.Request {
		t.Helper()
		r := httptest.NewRequest(http.MethodPost, RutaConsultaCircuitoRRHH, strings.NewReader(cuerpo))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Accept", "application/json")
		return r
	}
	respuesta := httptest.NewRecorder()
	h.ServeHTTP(respuesta, nuevaPeticion(`{"expediente_ref":"expediente:ct:circuito:001","version_observada":3}`))
	if respuesta.Code != http.StatusServiceUnavailable || autoridad.llamadas != 1 || consultor.llamadas != 1 {
		t.Fatalf("consulta: estado=%d autoridad=%d consultor=%d", respuesta.Code, autoridad.llamadas, consultor.llamadas)
	}
	s := consultor.solicitud
	if s.ExpedienteRef != "expediente:ct:circuito:001" || s.VersionObservada != 3 ||
		s.AutenticacionRef != "aut_aaaaaaaaaaaaaaaaaaaaaaaa" ||
		s.SesionRef != "ses_bbbbbbbbbbbbbbbbbbbbbbbb" ||
		s.PerfilRef != "prf_cccccccccccccccccccccccc" ||
		s.OrganizacionRef != "organizacion:dipgra:circuito:001" {
		t.Fatalf("contexto no ligado al canal: %+v", s)
	}
	respuesta = httptest.NewRecorder()
	h.ServeHTTP(respuesta, nuevaPeticion(`{"expediente_ref":"expediente:ct:circuito:001","version_observada":3,"perfil_ref":"prf_ajeno"}`))
	if respuesta.Code != http.StatusBadRequest || autoridad.llamadas != 1 || consultor.llamadas != 1 {
		t.Fatalf("identidad en JSON aceptada: estado=%d autoridad=%d consultor=%d", respuesta.Code, autoridad.llamadas, consultor.llamadas)
	}
}
