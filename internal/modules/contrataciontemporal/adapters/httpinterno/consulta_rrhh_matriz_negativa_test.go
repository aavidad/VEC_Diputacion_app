package httpinterno

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestManejadoresConsultaRRHHRechazanEnterosNoCanonicos(t *testing.T) {
	casos := []struct {
		nombre    string
		ruta      string
		cuerpo    string
		manejador func() (http.Handler, *int)
	}{
		{
			"cuadro negativo",
			RutaConsultaCuadroRRHH,
			`{"filtros":{},"paginacion":{"limite":-1}}`,
			func() (http.Handler, *int) {
				consultor := &consultorCuadroRRHHPrueba{}
				manejador, _ := NuevoManejadorConsultaCuadroRRHH(consultor)
				return manejador, &consultor.llamadas
			},
		},
		{
			"cuadro exponente",
			RutaConsultaCuadroRRHH,
			`{"filtros":{},"paginacion":{"limite":1e1}}`,
			func() (http.Handler, *int) {
				consultor := &consultorCuadroRRHHPrueba{}
				manejador, _ := NuevoManejadorConsultaCuadroRRHH(consultor)
				return manejador, &consultor.llamadas
			},
		},
		{
			"cuadro desborde",
			RutaConsultaCuadroRRHH,
			`{"filtros":{},"paginacion":{"limite":65536}}`,
			func() (http.Handler, *int) {
				consultor := &consultorCuadroRRHHPrueba{}
				manejador, _ := NuevoManejadorConsultaCuadroRRHH(consultor)
				return manejador, &consultor.llamadas
			},
		},
		{
			"detalle negativo",
			RutaConsultaDetalleRRHH,
			`{"expediente_ref":"expediente:ct:0001","version_observada":-1}`,
			func() (http.Handler, *int) {
				consultor := &consultorDetalleRRHHPrueba{}
				manejador, _ := NuevoManejadorConsultaDetalleRRHH(consultor)
				return manejador, &consultor.llamadas
			},
		},
		{
			"detalle exponente",
			RutaConsultaDetalleRRHH,
			`{"expediente_ref":"expediente:ct:0001","version_observada":1e1}`,
			func() (http.Handler, *int) {
				consultor := &consultorDetalleRRHHPrueba{}
				manejador, _ := NuevoManejadorConsultaDetalleRRHH(consultor)
				return manejador, &consultor.llamadas
			},
		},
		{
			"detalle desborde",
			RutaConsultaDetalleRRHH,
			`{"expediente_ref":"expediente:ct:0001",` +
				`"version_observada":18446744073709551616}`,
			func() (http.Handler, *int) {
				consultor := &consultorDetalleRRHHPrueba{}
				manejador, _ := NuevoManejadorConsultaDetalleRRHH(consultor)
				return manejador, &consultor.llamadas
			},
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			manejador, llamadas := caso.manejador()
			respuesta := httptest.NewRecorder()
			manejador.ServeHTTP(
				respuesta,
				nuevaPeticionConsultaRRHHPrueba(caso.ruta, caso.cuerpo),
			)
			if respuesta.Code != http.StatusBadRequest || *llamadas != 0 {
				t.Fatalf(
					"estado=%d llamadas=%d cuerpo=%s",
					respuesta.Code, *llamadas, respuesta.Body,
				)
			}
		})
	}
}

func TestManejadorConsultaRRHHCierraCuerposYMetadatosAnomalos(t *testing.T) {
	const cuerpo = `{"filtros":{},"paginacion":{"limite":1}}`
	casos := []struct {
		nombre   string
		preparar func(*http.Request)
	}{
		{
			"cuerpo nil",
			func(r *http.Request) {
				r.Body = nil
				r.ContentLength = 1
			},
		},
		{
			"NoBody",
			func(r *http.Request) {
				r.Body = http.NoBody
				r.ContentLength = 1
			},
		},
		{
			"Trailer",
			func(r *http.Request) {
				r.Trailer = http.Header{"X-Integridad": []string{"forjada"}}
			},
		},
		{
			"Transfer-Encoding",
			func(r *http.Request) {
				r.TransferEncoding = []string{"gzip"}
			},
		},
		{
			"Content-Encoding",
			func(r *http.Request) {
				r.Header.Set("Content-Encoding", "gzip")
			},
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			consultor := &consultorCuadroRRHHPrueba{}
			manejador, err := NuevoManejadorConsultaCuadroRRHH(consultor)
			if err != nil {
				t.Fatal(err)
			}
			peticion := nuevaPeticionConsultaRRHHPrueba(
				RutaConsultaCuadroRRHH,
				cuerpo,
			)
			caso.preparar(peticion)
			respuesta := httptest.NewRecorder()
			manejador.ServeHTTP(respuesta, peticion)
			if respuesta.Code != http.StatusBadRequest ||
				consultor.llamadas != 0 {
				t.Fatalf(
					"estado=%d llamadas=%d cuerpo=%s",
					respuesta.Code, consultor.llamadas, respuesta.Body,
				)
			}
		})
	}
}

func TestManejadorConsultaRRHHRechazaUTF8Invalido(t *testing.T) {
	consultor := &consultorCuadroRRHHPrueba{}
	manejador, err := NuevoManejadorConsultaCuadroRRHH(consultor)
	if err != nil {
		t.Fatal(err)
	}
	contenido := append(
		[]byte(`{"filtros":{"texto":"`),
		0xff,
	)
	contenido = append(
		contenido,
		[]byte(`"},"paginacion":{"limite":1}}`)...,
	)
	peticion := httptest.NewRequest(
		http.MethodPost,
		RutaConsultaCuadroRRHH,
		bytes.NewReader(contenido),
	)
	peticion.Header.Set("Content-Type", "application/json")
	peticion.Header.Set("Accept", "application/json")
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusBadRequest || consultor.llamadas != 0 {
		t.Fatalf(
			"estado=%d llamadas=%d cuerpo=%s",
			respuesta.Code, consultor.llamadas, respuesta.Body,
		)
	}
}

func TestManejadorConsultaRRHHRechazaPeticionNulaSinInvocarNegocio(
	t *testing.T,
) {
	consultor := &consultorCuadroRRHHPrueba{}
	manejador, err := NuevoManejadorConsultaCuadroRRHH(consultor)
	if err != nil {
		t.Fatal(err)
	}
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, nil)
	if respuesta.Code != http.StatusNotFound || consultor.llamadas != 0 {
		t.Fatalf(
			"estado=%d llamadas=%d cuerpo=%s",
			respuesta.Code, consultor.llamadas, respuesta.Body,
		)
	}
}
