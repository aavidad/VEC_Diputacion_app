package httpinterno

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"golang.org/x/text/unicode/norm"
)

func mapaSolicitudValidaPrueba(t *testing.T) map[string]any {
	t.Helper()
	var sobre map[string]any
	if err := json.Unmarshal(cuerpoValidoPrueba(), &sobre); err != nil {
		t.Fatal(err)
	}
	solicitud, correcta := sobre["solicitud"].(map[string]any)
	if !correcta {
		t.Fatalf("solicitud de prueba no es objeto: %T", sobre["solicitud"])
	}
	return solicitud
}

func codificarMapaPrueba(t *testing.T, valor any) []byte {
	t.Helper()
	contenido, err := json.Marshal(valor)
	if err != nil {
		t.Fatal(err)
	}
	return contenido
}

func codificarSolicitudPrueba(t *testing.T, solicitud map[string]any) []byte {
	t.Helper()
	return codificarMapaPrueba(t, map[string]any{
		"clave_idempotencia": claveIdempotenciaPrueba,
		"solicitud":          solicitud,
	})
}

func exigirRechazoLocalPrueba(t *testing.T, cuerpo []byte, estados ...int) {
	t.Helper()
	manejador, autoridad, ejecutor := nuevoEscenarioPrueba(t)
	respuesta := ejecutarPeticionPrueba(t, manejador, nuevaPeticionPrueba(t, cuerpo))
	coincide := false
	for _, estado := range estados {
		coincide = coincide || respuesta.Code == estado
	}
	if !coincide {
		t.Fatalf("entrada aceptada o estado inesperado: %d %s", respuesta.Code, respuesta.Body.String())
	}
	if autoridad.numeroLlamadas() != 0 {
		t.Fatal("entrada inválida alcanzó la autoridad")
	}
	if llamadas, _ := ejecutor.instantanea(); llamadas != 0 {
		t.Fatal("entrada inválida alcanzó el ejecutor")
	}
}

func exigirExitoLocalPrueba(t *testing.T, cuerpo []byte) {
	t.Helper()
	manejador, _, _ := nuevoEscenarioPrueba(t)
	respuesta := ejecutarPeticionPrueba(t, manejador, nuevaPeticionPrueba(t, cuerpo))
	if respuesta.Code != http.StatusCreated {
		t.Fatalf("límite válido rechazado: %d %s", respuesta.Code, respuesta.Body.String())
	}
}

func TestManejadorAltaRechazaJSONAmbiguoOMalformado(t *testing.T) {
	casos := []struct {
		nombre string
		cuerpo []byte
		estado int
	}{
		{"vacío", nil, http.StatusBadRequest},
		{"espacios", []byte(" \n\t "), http.StatusBadRequest},
		{"raíz nula", []byte("null"), http.StatusBadRequest},
		{"segundo documento", append(append([]byte(nil), cuerpoValidoPrueba()...), []byte(` {}`)...), http.StatusBadRequest},
		{"campo extra", []byte(`{"campo":"extra"}`), http.StatusBadRequest},
		{"caja alternativa", bytes.Replace(cuerpoValidoPrueba(), []byte(`"centro_ref"`), []byte(`"Centro_Ref"`), 1), http.StatusBadRequest},
		{"clave duplicada en raíz", bytes.Replace(cuerpoValidoPrueba(), []byte(`"clave_idempotencia":"`+claveIdempotenciaPrueba+`"`), []byte(`"clave_idempotencia":"`+claveIdempotenciaPrueba+`","clave_idempotencia":"`+claveIdempotenciaPrueba+`"`), 1), http.StatusBadRequest},
		{"duplicado en solicitud", bytes.Replace(cuerpoValidoPrueba(), []byte(`"centro_ref":"centro:solicitante:001"`), []byte(`"centro_ref":"centro:uno","centro_ref":"centro:dos"`), 1), http.StatusBadRequest},
		{"duplicado anidado", bytes.Replace(cuerpoValidoPrueba(), []byte(`"inicio":"2026-08-01T00:00:00Z"`), []byte(`"inicio":"2026-08-01T00:00:00Z","inicio":"2026-08-02T00:00:00Z"`), 1), http.StatusBadRequest},
		{"observaciones nulas", bytes.Replace(cuerpoValidoPrueba(), []byte(`"observaciones":"Tramitación ordinaria."`), []byte(`"observaciones":null`), 1), http.StatusBadRequest},
		{"adjuntos nulos", bytes.Replace(cuerpoValidoPrueba(), []byte(`["documento:opaco:001"]`), []byte(`null`), 1), http.StatusBadRequest},
		{"periodo nulo", bytes.Replace(cuerpoValidoPrueba(), []byte(`{"inicio":"2026-08-01T00:00:00Z","fin":"2026-12-31T00:00:00Z"}`), []byte(`null`), 1), http.StatusBadRequest},
		{"UTF-8 inválido", append([]byte(`{"detalle":"`), 0xff, '"', '}'), http.StatusBadRequest},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			exigirRechazoLocalPrueba(t, caso.cuerpo, caso.estado)
		})
	}
}

func TestManejadorAltaImponeTamanoProfundidadYComplejidad(t *testing.T) {
	t.Run("cuerpo mayor", func(t *testing.T) {
		exigirRechazoLocalPrueba(
			t,
			bytes.Repeat([]byte("x"), MaximoCuerpoAltaBytes+1),
			http.StatusRequestEntityTooLarge,
		)
	})
	t.Run("profundidad", func(t *testing.T) {
		cuerpo := []byte(`{"a":[[[[[[[[[0]]]]]]]]]}`)
		exigirRechazoLocalPrueba(t, cuerpo, http.StatusRequestEntityTooLarge)
	})
	t.Run("tokens", func(t *testing.T) {
		valores := make([]string, tokensMaximosJSONAlta+1)
		for indice := range valores {
			valores[indice] = fmt.Sprintf("documento:%04d", indice)
		}
		cuerpo := codificarMapaPrueba(t, map[string]any{"documentos_adjuntos": valores})
		exigirRechazoLocalPrueba(t, cuerpo, http.StatusRequestEntityTooLarge)
	})
}

func TestManejadorAltaFechasYPeriodoEnLimites(t *testing.T) {
	t.Run("cien años exactos", func(t *testing.T) {
		solicitud := mapaSolicitudValidaPrueba(t)
		solicitud["periodo"] = map[string]any{
			"inicio": "2026-08-01T00:00:00Z",
			"fin":    "2126-08-01T00:00:00Z",
		}
		exigirExitoLocalPrueba(t, codificarSolicitudPrueba(t, solicitud))
	})
	casos := []struct {
		nombre string
		inicio string
		fin    string
	}{
		{"offset UTC textual", "2026-08-01T00:00:00+00:00", "2026-08-02T00:00:00Z"},
		{"hora no civil", "2026-08-01T01:00:00Z", "2026-08-02T00:00:00Z"},
		{"fracción", "2026-08-01T00:00:00.000000Z", "2026-08-02T00:00:00Z"},
		{"día inexistente", "2026-02-30T00:00:00Z", "2026-08-02T00:00:00Z"},
		{"fin anterior", "2026-08-02T00:00:00Z", "2026-08-01T00:00:00Z"},
		{"más de cien años", "2026-08-01T00:00:00Z", "2126-08-02T00:00:00Z"},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			solicitud := mapaSolicitudValidaPrueba(t)
			solicitud["periodo"] = map[string]any{"inicio": caso.inicio, "fin": caso.fin}
			exigirRechazoLocalPrueba(t, codificarSolicitudPrueba(t, solicitud), http.StatusUnprocessableEntity)
		})
	}
}

func TestManejadorAltaRCImporteYMonedaEnLimites(t *testing.T) {
	rcValida := func(centimos any) map[string]any {
		return map[string]any{
			"existe":        true,
			"numero":        "rc:numero:001",
			"fecha":         "2026-07-01T00:00:00Z",
			"importe":       map[string]any{"centimos": centimos, "moneda": "EUR"},
			"documento_ref": "documento:rc:001",
		}
	}
	t.Run("máximo", func(t *testing.T) {
		solicitud := mapaSolicitudValidaPrueba(t)
		solicitud["rc"] = rcValida(MaximoCentimosJSON)
		exigirExitoLocalPrueba(t, codificarSolicitudPrueba(t, solicitud))
	})
	for _, caso := range []struct {
		nombre   string
		centimos any
		moneda   string
	}{
		{"cero", int64(0), "EUR"},
		{"negativo", int64(-1), "EUR"},
		{"excesivo", MaximoCentimosJSON + 1, "EUR"},
		{"fraccionario", 1.5, "EUR"},
		{"cadena", "100", "EUR"},
		{"moneda", int64(100), "USD"},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			solicitud := mapaSolicitudValidaPrueba(t)
			rc := rcValida(caso.centimos)
			rc["importe"].(map[string]any)["moneda"] = caso.moneda
			solicitud["rc"] = rc
			exigirRechazoLocalPrueba(
				t,
				codificarSolicitudPrueba(t, solicitud),
				http.StatusBadRequest,
				http.StatusUnprocessableEntity,
			)
		})
	}
	t.Run("exponente", func(t *testing.T) {
		cuerpo := bytes.Replace(
			codificarMapaPrueba(t, map[string]any{
				"centro_ref":          "centro:solicitante:001",
				"contacto_ref":        "contacto:opaco:001",
				"categoria_ref":       "categoria:tecnica:001",
				"grupo_subgrupo":      "A1",
				"motivo_clave":        "necesidad_temporal",
				"detalle":             "Detalle.",
				"periodo":             map[string]any{"inicio": "2026-08-01T00:00:00Z", "fin": "2026-08-02T00:00:00Z"},
				"rc":                  rcValida(int64(100)),
				"documentos_adjuntos": []string{},
			}),
			[]byte(`"centimos":100`),
			[]byte(`"centimos":1e2`),
			1,
		)
		exigirRechazoLocalPrueba(t, cuerpo, http.StatusBadRequest)
	})
	t.Run("rc falsa con evidencia", func(t *testing.T) {
		solicitud := mapaSolicitudValidaPrueba(t)
		solicitud["rc"] = map[string]any{"existe": false, "numero": "rc:numero:001"}
		exigirRechazoLocalPrueba(t, codificarSolicitudPrueba(t, solicitud), http.StatusUnprocessableEntity)
	})
	t.Run("rc verdadera incompleta", func(t *testing.T) {
		solicitud := mapaSolicitudValidaPrueba(t)
		solicitud["rc"] = map[string]any{"existe": true}
		exigirRechazoLocalPrueba(t, codificarSolicitudPrueba(t, solicitud), http.StatusUnprocessableEntity)
	})
}

func TestManejadorAltaDetalleObservacionesYAdjuntosEnLimites(t *testing.T) {
	t.Run("máximos válidos", func(t *testing.T) {
		solicitud := mapaSolicitudValidaPrueba(t)
		solicitud["detalle"] = strings.Repeat("á", MaximoCaracteresDetalle)
		solicitud["observaciones"] = strings.Repeat("ñ", MaximoCaracteresDetalle)
		adjuntos := make([]string, MaximoDocumentosAdjuntos)
		for indice := range adjuntos {
			adjuntos[indice] = fmt.Sprintf("documento:opaco:%03d", indice)
		}
		solicitud["documentos_adjuntos"] = adjuntos
		exigirExitoLocalPrueba(t, codificarSolicitudPrueba(t, solicitud))
	})
	casos := []struct {
		nombre    string
		modificar func(map[string]any)
	}{
		{"detalle excesivo", func(s map[string]any) { s["detalle"] = strings.Repeat("a", MaximoCaracteresDetalle+1) }},
		{"observaciones excesivas", func(s map[string]any) { s["observaciones"] = strings.Repeat("a", MaximoCaracteresDetalle+1) }},
		{"detalle con espacio exterior", func(s map[string]any) { s["detalle"] = " detalle" }},
		{"detalle no NFC", func(s map[string]any) { s["detalle"] = norm.NFD.String("á") }},
		{"control prohibido", func(s map[string]any) { s["detalle"] = "detalle\u0000" }},
		{"adjunto duplicado", func(s map[string]any) {
			s["documentos_adjuntos"] = []string{"documento:opaco:001", "documento:opaco:001"}
		}},
		{"más de 64 adjuntos", func(s map[string]any) {
			adjuntos := make([]string, MaximoDocumentosAdjuntos+1)
			for indice := range adjuntos {
				adjuntos[indice] = fmt.Sprintf("documento:opaco:%03d", indice)
			}
			s["documentos_adjuntos"] = adjuntos
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			solicitud := mapaSolicitudValidaPrueba(t)
			caso.modificar(solicitud)
			exigirRechazoLocalPrueba(t, codificarSolicitudPrueba(t, solicitud), http.StatusUnprocessableEntity)
		})
	}
}

func TestManejadorAltaReferenciasCatalogoYGrupoEnLimites(t *testing.T) {
	t.Run("límites válidos", func(t *testing.T) {
		solicitud := mapaSolicitudValidaPrueba(t)
		solicitud["centro_ref"] = "a" + strings.Repeat("b", MaximoCaracteresReferencia-1)
		solicitud["motivo_clave"] = "a" + strings.Repeat("b", 79)
		solicitud["grupo_subgrupo"] = "A" + strings.Repeat("1", 19)
		exigirExitoLocalPrueba(t, codificarSolicitudPrueba(t, solicitud))
	})
	for _, caso := range []struct {
		nombre    string
		modificar func(map[string]any)
	}{
		{"referencia corta", func(s map[string]any) { s["centro_ref"] = "ab" }},
		{"referencia larga", func(s map[string]any) { s["centro_ref"] = strings.Repeat("a", 161) }},
		{"clave corta", func(s map[string]any) { s["motivo_clave"] = "a" }},
		{"clave larga", func(s map[string]any) { s["motivo_clave"] = "a" + strings.Repeat("b", 80) }},
		{"grupo vacío", func(s map[string]any) { s["grupo_subgrupo"] = "" }},
		{"grupo largo", func(s map[string]any) { s["grupo_subgrupo"] = "A" + strings.Repeat("1", 20) }},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			solicitud := mapaSolicitudValidaPrueba(t)
			caso.modificar(solicitud)
			exigirRechazoLocalPrueba(t, codificarSolicitudPrueba(t, solicitud), http.StatusUnprocessableEntity)
		})
	}
}
