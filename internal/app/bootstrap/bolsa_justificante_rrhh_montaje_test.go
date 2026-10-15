package bootstrap

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/config"
	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
)

func TestJustificanteRenunciaRRHHSinConfiguracionDejaRutaCerrada(t *testing.T) {
	ruta, cerrar := nuevaRutaJustificanteRenunciaRRHH(config.Config{}, nil, nil, nil)
	if ruta == nil || ruta.Ruta != bolsahttp.RutaJustificanteRenunciaRRHH || ruta.Manejador == nil || cerrar == nil {
		t.Fatal("la falta de configuración impidió montar la ruta cerrada")
	}
	w := httptest.NewRecorder()
	ruta.Manejador.ServeHTTP(w, httptest.NewRequest(http.MethodGet,
		ruta.Ruta+"?bolsa=bolsa_1&participacion=part_1", nil))
	if w.Code != http.StatusServiceUnavailable || w.Header().Get("Cache-Control") == "" {
		t.Fatalf("sin configuración: estado=%d cabeceras=%v", w.Code, w.Header())
	}
	cerrar()
}

func TestConfiguracionJustificanteRenunciaRRHHNoExponeDSN(t *testing.T) {
	c := configuracionJustificanteRenunciaRRHH{DSNDocumentos: "postgres://usuario:clave-ficticia@127.0.0.1/vec"}
	b, err := json.Marshal(c)
	if err != nil || strings.Contains(string(b), "clave-ficticia") ||
		strings.Contains(fmt.Sprintf("%+v %#v", c, c), "clave-ficticia") {
		t.Fatal("la configuración expuso el DSN")
	}
}
