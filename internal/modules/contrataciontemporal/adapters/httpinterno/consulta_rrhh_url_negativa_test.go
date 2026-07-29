package httpinterno

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestRutaConsultaRRHHExactaRechazaURLNoCanonica(t *testing.T) {
	t.Parallel()
	rutas := []string{
		RutaConsultaCuadroRRHH,
		RutaConsultaDetalleRRHH,
	}
	anomalias := []struct {
		nombre string
		mutar  func(*http.Request, string)
	}{
		{
			nombre: "URL nula",
			mutar: func(peticion *http.Request, _ string) {
				peticion.URL = nil
			},
		},
		{
			nombre: "consulta vacía forzada",
			mutar: func(peticion *http.Request, _ string) {
				peticion.URL.ForceQuery = true
			},
		},
		{
			nombre: "ruta cruda",
			mutar: func(peticion *http.Request, ruta string) {
				peticion.URL.RawPath = ruta
			},
		},
		{
			nombre: "esquema",
			mutar: func(peticion *http.Request, _ string) {
				peticion.URL.Scheme = "https"
			},
		},
		{
			nombre: "servidor",
			mutar: func(peticion *http.Request, _ string) {
				peticion.URL.Host = "interno.invalid"
			},
		},
		{
			nombre: "usuario",
			mutar: func(peticion *http.Request, _ string) {
				peticion.URL.User = url.User("actor")
			},
		},
		{
			nombre: "fragmento",
			mutar: func(peticion *http.Request, _ string) {
				peticion.URL.Fragment = "fragmento"
			},
		},
		{
			nombre: "fragmento crudo",
			mutar: func(peticion *http.Request, _ string) {
				peticion.URL.RawFragment = "fragmento"
			},
		},
		{
			nombre: "ruta con codificación porcentual",
			mutar: func(peticion *http.Request, ruta string) {
				peticion.URL.RawPath = strings.Replace(
					ruta,
					"contratacion-temporal",
					"contratacion%2Dtemporal",
					1,
				)
			},
		},
	}

	for _, ruta := range rutas {
		ruta := ruta
		t.Run(ruta, func(t *testing.T) {
			t.Parallel()
			base := &http.Request{URL: &url.URL{Path: ruta}}
			if !rutaConsultaRRHHExacta(base, ruta) {
				t.Fatal("la ruta canónica de control fue rechazada")
			}
			for _, anomalia := range anomalias {
				anomalia := anomalia
				t.Run(anomalia.nombre, func(t *testing.T) {
					t.Parallel()
					peticion := &http.Request{
						URL: &url.URL{Path: ruta},
					}
					anomalia.mutar(peticion, ruta)
					if rutaConsultaRRHHExacta(peticion, ruta) {
						t.Fatal("una URL no canónica atravesó la ruta exacta")
					}
				})
			}
		})
	}
}
