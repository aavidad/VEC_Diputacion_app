package bootstrap

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/portafirmasapagado"
)

const rutaCircuitoFirmaCTEjemploPrueba = "../../../data/demo/reglas/ct_circuito_firma.ejemplo.demo.json"
const rutaCircuitoFirmaCTAlternativaPrueba = "../../../data/demo/reglas/ct_circuito_firma.rrhh.v2.json"

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
	ruta := nuevaRutaCircuitoFirmaContratacionTemporalDesarrollo(compuestas.circuitoFirmaCT, portafirmasapagado.Conector{})
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
		len(datos.Documentos) != 2 {
		t.Fatalf("circuito inesperado: %+v", datos)
	}
	if strings.Contains(respuesta.Body.String(), "perfiles_ref_alternativos") {
		t.Fatal("un catálogo antiguo no debe declarar alternativas ni cambiar el contrato v1")
	}
	if strings.Contains(respuesta.Body.String(), "misma_persona_en_dos_pasos") {
		t.Fatal("la política de firma no forma parte de la consulta v1")
	}
	// Firmadoc apagado: no conectado, con motivo, y nada que parezca envío.
	if datos.Portafirmas.Conectado || datos.Portafirmas.Motivo != "conexion_pendiente" {
		t.Fatalf("portafirmas inesperado: %+v", datos.Portafirmas)
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

func TestCircuitoFirmaRRHHExponeAlternativaVersionada(t *testing.T) {
	relojRRHH := relojReglasEjemploPrueba{ahora: time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)}
	compuestas, err := nuevasReglasEjemploDesarrollo(configuracionCircuitoFirmaPrueba(rutaCircuitoFirmaCTAlternativaPrueba), nil, relojRRHH)
	if err != nil {
		t.Fatal(err)
	}
	respuesta := httptest.NewRecorder()
	nuevaRutaCircuitoFirmaContratacionTemporalDesarrollo(compuestas.circuitoFirmaCT, portafirmasapagado.Conector{}).Manejador.ServeHTTP(
		respuesta, httptest.NewRequest(http.MethodGet, rutaCircuitoFirmaContratacionTemporalDesarrollo, nil))
	if respuesta.Code != http.StatusOK {
		t.Fatalf("consulta: %d %s", respuesta.Code, respuesta.Body.String())
	}
	var cuerpo struct {
		Data circuitoFirmaDesarrollo `json:"data"`
	}
	if err := json.Unmarshal(respuesta.Body.Bytes(), &cuerpo); err != nil {
		t.Fatal(err)
	}
	datos := cuerpo.Data
	if datos.Esquema != esquemaCircuitoFirmaAlternativasDesarrollo || datos.CatalogoRef != "vec.contratacion_temporal.circuito_firma:2" ||
		!datos.Ejemplo || datos.FirmaEficaz || len(datos.Documentos) != 2 {
		t.Fatalf("circuito RRHH inesperado: %+v", datos)
	}
	if strings.Contains(respuesta.Body.String(), "misma_persona_en_dos_pasos") {
		t.Fatal("la política de firma no debe salir en la consulta v2")
	}
	configurado, err := compuestas.circuitoFirmaCT.CircuitoFirma(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if configurado.PermiteMismaPersonaEnPasos ||
		configurado.HuellaCatalogo != datos.HuellaSHA256 ||
		configurado.Version != 2 {
		t.Fatalf("la política interna debe denegar por defecto y conservar la procedencia v2: %+v", configurado)
	}
	configurado.PermiteMismaPersonaEnPasos = true
	conPolitica, err := json.Marshal(vistaCircuitoFirmaDesarrollo(configurado))
	if err != nil || strings.Contains(string(conPolitica), "misma_persona_en_dos_pasos") {
		t.Fatalf("la política interna no debe aparecer aun si el catálogo la admite: %s, %v", conPolitica, err)
	}
	informe, resolucion := datos.Documentos[0], datos.Documentos[1]
	if informe.Documento != "informe_definitivo" || len(informe.Pasos) != 1 ||
		informe.Pasos[0].PerfilRef != "perfil:ct:jefatura_servicio_rrhh" || len(informe.Pasos[0].PerfilesAlternativos) != 0 ||
		resolucion.Documento != "resolucion" || len(resolucion.Pasos) != 2 ||
		resolucion.Pasos[0].PerfilRef != "perfil:ct:jefatura_servicio_rrhh" ||
		len(resolucion.Pasos[0].PerfilesAlternativos) != 1 || resolucion.Pasos[0].PerfilesAlternativos[0] != "perfil:ct:direccion_rrhh" ||
		resolucion.Pasos[0].Accion != "visto_bueno" || resolucion.Pasos[1].PerfilRef != "perfil:ct:diputacion_delegada_rrhh" ||
		len(resolucion.Pasos[1].PerfilesAlternativos) != 0 || resolucion.Pasos[1].Accion != "firma" {
		t.Fatalf("visto bueno alternativo y firma de Diputación: %+v %+v", informe, resolucion)
	}
	var bruto struct {
		Data struct {
			Documentos []struct {
				Pasos []map[string]json.RawMessage `json:"pasos"`
			} `json:"documentos"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respuesta.Body.Bytes(), &bruto); err != nil {
		t.Fatal(err)
	}
	for _, paso := range []map[string]json.RawMessage{bruto.Data.Documentos[0].Pasos[0], bruto.Data.Documentos[1].Pasos[1]} {
		if _, presente := paso["perfiles_ref_alternativos"]; presente {
			t.Fatalf("paso sin alternativa expuesto con campo nuevo: %v", paso)
		}
	}
}

func TestCircuitoFirmaRechazosYFaltaDeCatalogo(t *testing.T) {
	sinCatalogo := nuevaRutaCircuitoFirmaContratacionTemporalDesarrollo(nil, nil)
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

// Sin conector compuesto, Firmadoc también figura como no conectado.
func TestCircuitoFirmaSinConectorDeclaraFirmadocNoConectado(t *testing.T) {
	compuestas, err := nuevasReglasEjemploDesarrollo(configuracionCircuitoFirmaPrueba(rutaCircuitoFirmaCTEjemploPrueba), nil, relojPresentacionReglasEjemplo)
	if err != nil {
		t.Fatal(err)
	}
	respuesta := httptest.NewRecorder()
	nuevaRutaCircuitoFirmaContratacionTemporalDesarrollo(compuestas.circuitoFirmaCT, nil).Manejador.ServeHTTP(respuesta,
		httptest.NewRequest(http.MethodGet, rutaCircuitoFirmaContratacionTemporalDesarrollo, nil))
	var cuerpo struct {
		Data circuitoFirmaDesarrollo `json:"data"`
	}
	if err := json.Unmarshal(respuesta.Body.Bytes(), &cuerpo); err != nil || respuesta.Code != http.StatusOK ||
		cuerpo.Data.Portafirmas.Conectado || cuerpo.Data.Portafirmas.Motivo != "conexion_pendiente" {
		t.Fatalf("%d %+v %v", respuesta.Code, cuerpo.Data.Portafirmas, err)
	}
}
