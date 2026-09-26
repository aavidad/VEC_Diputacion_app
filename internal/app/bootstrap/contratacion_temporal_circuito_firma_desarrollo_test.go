package bootstrap

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"vec-diputacion-granada/config"
)

const rutaCircuitoFirmaCTEjemploPrueba = "../../../data/demo/reglas/ct_circuito_firma.ejemplo.demo.json"

func configuracionCircuitoFirmaPrueba(ruta string) config.Config {
	cfg := configuracionDesarrolloReglasEjemplo("", "")
	cfg.ReglasEjemplo.CTCircuitoFirmaSourcePath = ruta
	return cfg
}

func TestCircuitoFirmaEjemploSeConsultaConEstadoSinFirmas(t *testing.T) {
	compuestas, err := nuevasReglasEjemploDesarrollo(configuracionCircuitoFirmaPrueba(rutaCircuitoFirmaCTEjemploPrueba), nil, relojPresentacionReglasEjemplo)
	if err != nil || !compuestas.circuitoFirmaCT.Disponible() || compuestas.contratacionTemporal.Disponible() {
		t.Fatalf("el circuito se compone por separado: %+v %v", compuestas, err)
	}
	ruta := nuevaRutaCircuitoFirmaContratacionTemporalDesarrollo(compuestas.circuitoFirmaCT)
	respuesta := httptest.NewRecorder()
	ruta.Manejador.ServeHTTP(respuesta, httptest.NewRequest(http.MethodGet, rutaCircuitoFirmaContratacionTemporalDesarrollo, nil))
	if respuesta.Code != http.StatusOK || respuesta.Header().Get("Cache-Control") != "no-store, no-transform" ||
		respuesta.Header().Get("Set-Cookie") != "" {
		t.Fatalf("respuesta inesperada: %d %v", respuesta.Code, respuesta.Header())
	}
	var cuerpo struct {
		Data circuitoFirmaDesarrollo `json:"data"`
	}
	if err := json.Unmarshal(respuesta.Body.Bytes(), &cuerpo); err != nil {
		t.Fatal(err)
	}
	datos := cuerpo.Data
	if datos.Esquema != esquemaCircuitoFirmaContratacionTemporalDesarrollo || !datos.Ejemplo || datos.FirmaEficaz ||
		datos.CatalogoRef != "vec.contratacion_temporal.circuito_firma:1" || len(datos.HuellaSHA256) != 64 ||
		len(datos.Documentos) != 6 {
		t.Fatalf("circuito inesperado: %+v", datos)
	}
	for _, documento := range datos.Documentos {
		for _, paso := range documento.Pasos {
			esperado := "en_espera"
			if paso.Orden == 1 {
				esperado = "pendiente_firma"
			}
			if paso.Estado != esperado {
				t.Fatalf("%s paso %d en %q; sin firmas registradas debe estar %q", documento.Documento, paso.Orden, paso.Estado, esperado)
			}
		}
	}
}

func TestCircuitoFirmaRechazosYFaltaDeCatalogo(t *testing.T) {
	sinCatalogo := nuevaRutaCircuitoFirmaContratacionTemporalDesarrollo(nil)
	casos := []struct {
		metodo, destino string
		estado          int
	}{
		{http.MethodGet, rutaCircuitoFirmaContratacionTemporalDesarrollo, http.StatusServiceUnavailable},
		{http.MethodPost, rutaCircuitoFirmaContratacionTemporalDesarrollo, http.StatusMethodNotAllowed},
		{http.MethodGet, rutaCircuitoFirmaContratacionTemporalDesarrollo + "?documento=x", http.StatusBadRequest},
	}
	for _, caso := range casos {
		respuesta := httptest.NewRecorder()
		sinCatalogo.Manejador.ServeHTTP(respuesta, httptest.NewRequest(caso.metodo, caso.destino, nil))
		if respuesta.Code != caso.estado {
			t.Errorf("%s %s: %d, esperado %d", caso.metodo, caso.destino, respuesta.Code, caso.estado)
		}
	}
	peticion := httptest.NewRequest(http.MethodGet, rutaCircuitoFirmaContratacionTemporalDesarrollo, nil)
	peticion.Header.Set("Cookie", "sesion=1")
	respuesta := httptest.NewRecorder()
	sinCatalogo.Manejador.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusBadRequest {
		t.Fatalf("una cookie no es una credencial admitida: %d", respuesta.Code)
	}
	if !esRutaContratacionTemporalDesarrollo(httptest.NewRequest(http.MethodGet, rutaCircuitoFirmaContratacionTemporalDesarrollo, nil)) {
		t.Fatal("la consulta debe quedar tras la frontera mTLS de Contratación temporal")
	}
}

func TestCircuitoFirmaInvalidoImpideArrancar(t *testing.T) {
	for _, ruta := range []string{rutaReglasCTEjemploPrueba, rutaInexistenteReglasEjemploPr} {
		if _, err := nuevasReglasEjemploDesarrollo(configuracionCircuitoFirmaPrueba(ruta), nil, relojPresentacionReglasEjemplo); !errors.Is(err, errReglasEjemploNoValidas) {
			t.Errorf("%s aceptado como circuito: %v", ruta, err)
		}
	}
}

// El circuito de ejemplo marca el paso 2 del informe definitivo como el que
// habilita la remisión a Intervención; la fuente debe conservarlo para que
// la fiscalización lo exija (duda 4).
func TestCircuitoFirmaEjemploConservaHabilitacionRemision(t *testing.T) {
	compuestas, err := nuevasReglasEjemploDesarrollo(configuracionCircuitoFirmaPrueba(rutaCircuitoFirmaCTEjemploPrueba), nil, relojPresentacionReglasEjemplo)
	if err != nil {
		t.Fatal(err)
	}
	circuito, err := fuenteCircuitoFirmaReglasDesarrollo{resolutor: compuestas.circuitoFirmaCT}.CircuitoFirma(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	var habilitan []string
	for _, d := range circuito.Documentos {
		for _, p := range d.Pasos {
			if p.Habilita == "remision_intervencion" {
				habilitan = append(habilitan, d.Documento+"."+string(rune('0'+p.Orden)))
			}
		}
	}
	if len(habilitan) != 1 || habilitan[0] != "informe_definitivo.2" {
		t.Fatalf("pasos que habilitan la remisión: %v", habilitan)
	}
}
