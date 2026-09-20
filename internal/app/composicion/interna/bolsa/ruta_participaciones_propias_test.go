package bolsa

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	httpinternobolsa "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
	aplicacionbolsa "vec-diputacion-granada/internal/modules/bolsa/application"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

type preparadorB11Prueba struct {
	llamadas int
	err      error
}

func (p *preparadorB11Prueba) PrepararOrdenConsultaParticipacionesPropias(context.Context) (aplicacionbolsa.OrdenConsultaParticipacionesPropias, error) {
	p.llamadas++
	return aplicacionbolsa.OrdenConsultaParticipacionesPropias{}, p.err
}

func TestNuevaRutaParticipacionesPropiasB11ClasificaRevocacionComoAutenticacion(t *testing.T) {
	preparador := &preparadorB11Prueba{err: errors.Join(
		ErrPreparadorParticipacionesPropiasB11Invalido,
		httpinternobolsa.ErrAutenticacionInternaAusente,
	)}
	consultor := &consultorB11Prueba{}
	handler, err := NuevaRutaParticipacionesPropiasB11(DependenciasRutaParticipacionesPropiasB11{
		Autenticador:  &autenticadorB11Prueba{},
		Sesiones:      &resolvedorB11Prueba{},
		Extractor:     &extractorB11Prueba{sobre: []byte("sobre")},
		Preparador:    preparador,
		Consultor:     consultor,
		Denegaciones:  &registradorDenegacionB11Prueba{},
		Correlaciones: &correladorDenegacionB11Prueba{},
	})
	if err != nil {
		t.Fatal(err)
	}
	peticion := httptest.NewRequest(http.MethodGet, rutaParticipacionesPropiasB11, nil)
	peticion.TLS = &tls.ConnectionState{}
	respuesta := httptest.NewRecorder()
	handler.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusUnauthorized || preparador.llamadas != 1 || consultor.llamadas != 0 {
		t.Fatalf("revocacion mal clasificada: codigo=%d preparador=%d consultor=%d", respuesta.Code, preparador.llamadas, consultor.llamadas)
	}
}

type consultorB11Prueba struct{ llamadas int }

func (c *consultorB11Prueba) Consultar(context.Context, aplicacionbolsa.OrdenConsultaParticipacionesPropias) (puertosbolsa.ResultadoParticipacionesPropias, error) {
	c.llamadas++
	return puertosbolsa.ResultadoParticipacionesPropias{Esquema: puertosbolsa.EsquemaParticipacionesPropiasV1, ConsultadaEn: time.Now().UTC()}, nil
}

func TestNuevaRutaParticipacionesPropiasB11ComponeDependenciasExplicitas(t *testing.T) {
	preparador := &preparadorB11Prueba{}
	consultor := &consultorB11Prueba{}
	handler, err := NuevaRutaParticipacionesPropiasB11(DependenciasRutaParticipacionesPropiasB11{
		Autenticador:  &autenticadorB11Prueba{},
		Sesiones:      &resolvedorB11Prueba{},
		Extractor:     &extractorB11Prueba{sobre: []byte("sobre")},
		Preparador:    preparador,
		Consultor:     consultor,
		Denegaciones:  &registradorDenegacionB11Prueba{},
		Correlaciones: &correladorDenegacionB11Prueba{},
	})
	if err != nil || handler == nil {
		t.Fatalf("componer B11: handler_nulo=%t err=%v", handler == nil, err)
	}
	peticion := httptest.NewRequest(http.MethodGet, rutaParticipacionesPropiasB11, nil)
	peticion.TLS = &tls.ConnectionState{}
	respuesta := httptest.NewRecorder()
	handler.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusOK || preparador.llamadas != 1 || consultor.llamadas != 1 {
		t.Fatalf("handler B11 no alcanzo el contrato existente: codigo=%d preparador=%d consultor=%d", respuesta.Code, preparador.llamadas, consultor.llamadas)
	}
}

func TestNuevaRutaParticipacionesPropiasB11RechazaDependenciasAusentes(t *testing.T) {
	_, err := NuevaRutaParticipacionesPropiasB11(DependenciasRutaParticipacionesPropiasB11{})
	if err == nil {
		t.Fatal("la ruta B11 acepto dependencias vacias")
	}
}
