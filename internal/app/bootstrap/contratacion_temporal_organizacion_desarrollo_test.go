package bootstrap

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"vec-diputacion-granada/config"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
)

func configuracionOrganizacionPrueba() config.Config {
	return config.Config{
		PersonalOrganizacionSourcePath: "../../../data/catalogos/estructura-organizativa/v1.rpt-publica.json",
		PersonalOrganizacionVersion:    1,
	}
}

func TestOrganizacionSirveCatalogoExistenteSinCambiarAlta(t *testing.T) {
	ruta, err := nuevaRutaOrganizacionContratacionTemporalDesarrollo(configuracionOrganizacionPrueba())
	if err != nil {
		t.Fatal(err)
	}
	respuesta := httptest.NewRecorder()
	ruta.Manejador.ServeHTTP(respuesta, httptest.NewRequest(http.MethodGet, ruta.Ruta, nil))
	if respuesta.Code != http.StatusOK {
		t.Fatalf("estado = %d", respuesta.Code)
	}
	var resultado struct {
		Data personalports.EstructuraOrganizativaConsultable `json:"data"`
	}
	if err := json.Unmarshal(respuesta.Body.Bytes(), &resultado); err != nil {
		t.Fatal(err)
	}
	if resultado.Data.Estado != "borrador" || resultado.Data.CatalogoVersion != 1 || len(resultado.Data.Unidades) != 66 {
		t.Fatalf("catálogo inesperado: estado=%s versión=%d unidades=%d", resultado.Data.Estado, resultado.Data.CatalogoVersion, len(resultado.Data.Unidades))
	}
	conteos := map[string]int{}
	alfaNumerico := false
	for _, unidad := range resultado.Data.Unidades {
		conteos[unidad.Tipo]++
		if unidad.CodigoFuente == "810A" && unidad.Tipo == "centro" {
			alfaNumerico = true
		}
	}
	if conteos["centro"] != 41 || conteos["delegacion"] != 14 || conteos["puesto_responsabilidad"] != 11 || !alfaNumerico {
		t.Fatalf("conteos/código alfanumérico incorrectos: %v, %v", conteos, alfaNumerico)
	}
	alta, err := nuevoOrigenConsultasContratacionTemporalDesarrollo().catalogosAlta()
	if err != nil || len(alta.Centros) != 1 || alta.Centros[0].Referencia != centroAltaContratacionTemporalDesarrollo {
		t.Fatal("la consulta organizativa no debe ampliar centros autorizados")
	}
}

func TestOrganizacionTransporteSoloConsulta(t *testing.T) {
	ruta, err := nuevaRutaOrganizacionContratacionTemporalDesarrollo(configuracionOrganizacionPrueba())
	if err != nil {
		t.Fatal(err)
	}
	for _, caso := range []struct {
		nombre, metodo, sufijo, cabecera string
		estado                           int
	}{
		{"consulta", "GET", "", "", 200},
		{"cabeceras", "HEAD", "", "", 200},
		{"sin escritura", "POST", "", "", 405},
		{"sin parámetros", "GET", "?version=2", "", 400},
		{"sin cookies", "GET", "", "Cookie", 400},
		{"sin autoridad libre", "GET", "", "X-User", 400},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			peticion := httptest.NewRequest(caso.metodo, ruta.Ruta+caso.sufijo, nil)
			if caso.cabecera != "" {
				peticion.Header.Set(caso.cabecera, "sintetico")
			}
			respuesta := httptest.NewRecorder()
			ruta.Manejador.ServeHTTP(respuesta, peticion)
			if respuesta.Code != caso.estado {
				t.Fatalf("estado=%d, esperado=%d", respuesta.Code, caso.estado)
			}
			if respuesta.Header().Get("Set-Cookie") != "" || respuesta.Header().Get("Cache-Control") != "no-store, no-transform" {
				t.Fatal("cabeceras de conservación inválidas")
			}
			if caso.metodo == "HEAD" && respuesta.Body.Len() != 0 {
				t.Fatal("HEAD incluye cuerpo")
			}
		})
	}
}

func TestOrganizacionConfiguracionAusenteOIncompatible(t *testing.T) {
	ruta, err := nuevaRutaOrganizacionContratacionTemporalDesarrollo(config.Config{})
	if err != nil {
		t.Fatal(err)
	}
	respuesta := httptest.NewRecorder()
	ruta.Manejador.ServeHTTP(respuesta, httptest.NewRequest(http.MethodGet, ruta.Ruta, nil))
	if respuesta.Code != http.StatusServiceUnavailable {
		t.Fatalf("estado=%d", respuesta.Code)
	}
	cfg := configuracionOrganizacionPrueba()
	cfg.PersonalOrganizacionVersion = 2
	if _, err := nuevaRutaOrganizacionContratacionTemporalDesarrollo(cfg); err == nil {
		t.Fatal("no debe sustituir una versión ausente por la última disponible")
	}
}
