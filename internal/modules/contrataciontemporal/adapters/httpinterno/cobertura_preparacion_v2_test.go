package httpinterno

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

func TestPropuestaCoberturaSubconjuntoInsuficienteRespondeClaveNeutraSinCatalogo(t *testing.T) {
	servicio := &servicioCoberturaPrueba{err: errors.Join(
		application.ErrPreparacionCatalogoCoberturaNoDisponiblePerfil,
		errors.New("catalogo:privado:documento_sae"),
	)}
	manejador, err := NuevoManejadorCobertura(
		autoridadCoberturaPrueba{contexto: contextoCoberturaValidoPrueba()},
		servicio, servicio,
	)
	if err != nil {
		t.Fatal(err)
	}
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, nuevaPeticionCoberturaPrueba(RutaPropuestaCobertura,
		`{"expediente_ref":"expediente:ct:0001","version_esperada":1}`))
	if respuesta.Code != http.StatusForbidden || servicio.proponerLlamadas != 1 {
		t.Fatalf("estado o caso de uso: %d/%d", respuesta.Code, servicio.proponerLlamadas)
	}
	var raiz map[string]any
	if err := json.Unmarshal(respuesta.Body.Bytes(), &raiz); err != nil {
		t.Fatal(err)
	}
	exigirClavesCoberturaV2(t, raiz, "error")
	problema := raiz["error"].(map[string]any)
	exigirClavesCoberturaV2(t, problema, "codigo", "clave_i18n", "correlacion_ref")
	if problema["codigo"] != "datos_no_disponibles_perfil" ||
		problema["clave_i18n"] != "api.contratacion_temporal.cobertura.error.datos_no_disponibles_perfil" {
		t.Fatalf("código contextual no traducible: %v", problema)
	}
	for _, prohibido := range []string{"catalogo:privado", "documento_sae", "evaluaciones", "es_ejemplo", "vias"} {
		if strings.Contains(respuesta.Body.String(), prohibido) {
			t.Fatalf("error filtró %q", prohibido)
		}
	}
	if clasificarErrorCobertura(application.ErrPresentacionPropuestaCoberturaDenegada).codigo != "acceso_denegado" {
		t.Fatal("la denegación general perdió su código")
	}
}

func TestProyeccionPreparacionCoberturaV2SoloCamposConcedidos(t *testing.T) {
	catalogo := proyectarPreparacionCatalogoCobertura(&application.PreparacionCatalogoPropuestaCobertura{
		Identidad: domain.IdentidadCatalogoViasCobertura{
			Referencia: "catalogo:ct:preparacion:v2", Version: 2,
			HuellaSHA256: strings.Repeat("a", 64),
		},
		Canon: domain.CanonHuellaCatalogoCoberturaV2(), EsEjemplo: true,
		Vias: []application.PreparacionViaPropuestaCobertura{{
			Clave: "oferta_sae", Orden: 1,
			Documentos: []domain.ElementoPreparacionViaCobertura{{
				Clave: "documento_sae", Orden: 1, ClaveI18n: "ct.cobertura.documento_sae",
			}},
			Datos: []domain.ElementoPreparacionViaCobertura{{
				Clave: "dato_sae", Orden: 1, ClaveI18n: "ct.cobertura.dato_sae",
			}},
		}},
	})
	contenido, err := json.Marshal(catalogo)
	if err != nil {
		t.Fatal(err)
	}
	var raiz map[string]any
	if err := json.Unmarshal(contenido, &raiz); err != nil {
		t.Fatal(err)
	}
	exigirClavesCoberturaV2(t, raiz, "referencia", "version", "huella_sha256", "es_ejemplo", "vias")
	vias := raiz["vias"].([]any)
	if len(vias) != 1 {
		t.Fatal("SAE no apareció como lista publicada")
	}
	via := vias[0].(map[string]any)
	exigirClavesCoberturaV2(t, via, "clave", "orden", "documentos", "datos")
	if via["clave"] != "oferta_sae" || raiz["es_ejemplo"] != true {
		t.Fatalf("identidad o ejemplo perdido: %s", contenido)
	}
	for _, nombre := range []string{"documentos", "datos"} {
		elementos := via[nombre].([]any)
		if len(elementos) != 1 {
			t.Fatalf("%s incompleto: %s", nombre, contenido)
		}
		exigirClavesCoberturaV2(t, elementos[0].(map[string]any), "clave", "orden", "clave_i18n")
	}
	for _, prohibido := range []string{"contacto", "envio", "seleccion", "comprobaciones", "procedencia"} {
		if strings.Contains(string(contenido), prohibido) {
			t.Fatalf("campo operativo %q filtrado en SAE", prohibido)
		}
	}
}

func exigirClavesCoberturaV2(t *testing.T, objeto map[string]any, claves ...string) {
	t.Helper()
	if len(objeto) != len(claves) {
		t.Fatalf("campos inesperados: %v", objeto)
	}
	for _, clave := range claves {
		if _, existe := objeto[clave]; !existe {
			t.Fatalf("falta %s: %v", clave, objeto)
		}
	}
}

func TestRespuestaCoberturaV2RespetaLimiteHTTP(t *testing.T) {
	// Combina los 512 elementos V2 con 64 vías, 32 claves de evaluación por
	// vía y 64 motivos, que son los máximos simultáneos del contrato.
	salida := propuestaCoberturaSalidaJSON{
		Esquema: "vec.contratacion-temporal.propuesta-cobertura.v2",
		Estado:  "viable", ViaRecomendada: "via_00",
		Catalogo: &catalogoPreparacionCoberturaJSON{
			Referencia: "catalogo:ct:preparacion:v2", Version: 2,
			HuellaSHA256: strings.Repeat("a", 64), EsEjemplo: true,
		},
	}
	for indice := 0; indice < maximasViasCoberturaHTTP; indice++ {
		via := viaPreparacionCoberturaJSON{
			Clave: fmt.Sprintf("via_%02d", indice), Orden: uint16(indice + 1),
		}
		for elemento := 0; elemento < 4; elemento++ {
			dato := elementoPreparacionCoberturaJSON{
				Clave: fmt.Sprintf("%s%02d", strings.Repeat("c", 78), elemento), Orden: uint16(elemento + 1),
				ClaveI18n: "ct." + strings.Repeat("a", 77),
			}
			via.Documentos = append(via.Documentos, dato)
			via.Datos = append(via.Datos, dato)
		}
		salida.Catalogo.Vias = append(salida.Catalogo.Vias, via)
		evaluacion := evaluacionCoberturaJSON{
			ViaClave: via.Clave, Prioridad: via.Orden, Estado: "viable",
		}
		for clave := 0; clave < maximasClavesPorViaHTTP; clave++ {
			evaluacion.AusenciasAdmitidas = append(evaluacion.AusenciasAdmitidas,
				fmt.Sprintf("%s%02d", strings.Repeat("a", 78), clave))
		}
		salida.Evaluaciones = append(salida.Evaluaciones, evaluacion)
		salida.MotivosAlternativa = append(salida.MotivosAlternativa,
			motivoAlternativaCoberturaJSON{
				Clave: fmt.Sprintf("motivo_%02d", indice), ViaClave: via.Clave,
				EtiquetaI18n: "ct." + strings.Repeat("m", 77),
			})
	}
	peticion := httptest.NewRequest(http.MethodPost, RutaPropuestaCobertura, nil)
	respuesta := httptest.NewRecorder()
	responderJSONCobertura(respuesta, peticion, http.StatusOK, envoltorioPropuestaCobertura{Data: salida})
	if respuesta.Code != http.StatusOK || respuesta.Body.Len() <= MaximoRespuestaCoberturaBytes ||
		respuesta.Body.Len() > MaximoRespuestaPropuestaCoberturaV2Bytes {
		t.Fatalf("límite HTTP V2: estado=%d tamaño=%d", respuesta.Code, respuesta.Body.Len())
	}
	var recibido envoltorioPropuestaCobertura
	if err := json.Unmarshal(respuesta.Body.Bytes(), &recibido); err != nil || !reflect.DeepEqual(recibido.Data.Catalogo, salida.Catalogo) {
		t.Fatal("la respuesta truncó o alteró el catálogo V2")
	}
	// Un contrato abusivo superior a 384 KiB falla cerrado sin filtrar datos.
	for len(salida.Catalogo.Vias) < 180 {
		salida.Catalogo.Vias = append(salida.Catalogo.Vias, salida.Catalogo.Vias[0])
	}
	respuesta = httptest.NewRecorder()
	responderJSONCobertura(respuesta, peticion, http.StatusOK, envoltorioPropuestaCobertura{Data: salida})
	if respuesta.Code != http.StatusInternalServerError ||
		strings.Contains(respuesta.Body.String(), "catalogo:ct:preparacion:v2") {
		t.Fatalf("salida abusiva no cerrada: estado=%d tamaño=%d", respuesta.Code, respuesta.Body.Len())
	}
}

func TestRespuestaCoberturaV2NoAmpliaLimiteDeOtrasRutas(t *testing.T) {
	grande := envoltorioReciboCobertura{Data: reciboCoberturaJSON{
		ReciboRef: strings.Repeat("r", MaximoRespuestaCoberturaBytes),
	}}
	for _, ruta := range []string{RutaDecisionCobertura, RutaRectificacionCobertura} {
		respuesta := httptest.NewRecorder()
		peticion := httptest.NewRequest(http.MethodPost, ruta, nil)
		responderJSONCobertura(respuesta, peticion, http.StatusCreated, grande)
		if respuesta.Code != http.StatusInternalServerError ||
			strings.Contains(respuesta.Body.String(), strings.Repeat("r", 100)) {
			t.Fatalf("%s amplió el límite de 256 KiB", ruta)
		}
	}
	respuesta := httptest.NewRecorder()
	peticion := httptest.NewRequest(http.MethodPost, RutaPropuestaCobertura, nil)
	responderJSONCobertura(respuesta, peticion, http.StatusOK,
		envoltorioPropuestaCobertura{Data: propuestaCoberturaSalidaJSON{
			Esquema:        "vec.contratacion-temporal.propuesta-cobertura.v1",
			ViaRecomendada: strings.Repeat("r", MaximoRespuestaCoberturaBytes),
		}})
	if respuesta.Code != http.StatusInternalServerError {
		t.Fatal("la propuesta V1 amplió el límite histórico")
	}
}
