package httpinterno

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	vecports "vec-diputacion-granada/internal/vec/ports"
)

// La correlación la coloca una sola vez la frontera común en el contexto.
// Cabeceras, cuerpo y referencias del documento no son fuentes de correlación.
func correlacionPeticionFirma(r *http.Request) string {
	if r != nil {
		if ref, ok := vecports.CorrelacionIncidenciasPeticion(r.Context()); ok {
			return "correlacion_" + ref
		}
	}
	return "corr_no_disponible"
}

func responderJSONFirmaNominal(w http.ResponseWriter, r *http.Request, estado int, valor any, limite int, causas ...error) {
	contenido, err := json.Marshal(valor)
	if err != nil || len(contenido) > limite {
		estado = http.StatusInternalServerError
		valor = map[string]any{"error": map[string]string{"codigo": "resultado_no_confiable", "clave_i18n": "api.contratacion_temporal.registro_firma_vec.error.resultado_no_confiable", "correlacion_ref": correlacionPeticionFirma(r)}}
		contenido, _ = json.Marshal(valor)
	}
	if estado >= http.StatusInternalServerError && r != nil {
		codigo, correlacion := metadatosErrorContratacion(valor)
		operacion := "firma"
		ruta := "ruta_no_reconocida"
		if r.URL != nil {
			switch r.URL.Path {
			case RutaRegistroFirmaVec:
				ruta, operacion = RutaRegistroFirmaVec, "registro_firma_vec"
			case RutaRegistroFirmaExterna:
				ruta, operacion = RutaRegistroFirmaExterna, "registro_firma_externa"
			case RutaPreflightFirmaR5:
				ruta, operacion = RutaPreflightFirmaR5, "preflight_firma"
			case RutaOriginalFirmableCT:
				ruta, operacion = RutaOriginalFirmableCT, "original_firmable"
			}
		}
		var causa error
		if len(causas) > 0 {
			causa = causas[0]
		}
		// Solo metadatos cerrados: nunca se escribe el texto de un error ni PDF.
		slog.ErrorContext(r.Context(), "contratacion_firma_fallida", "operacion", operacion, "ruta", ruta,
			"estado_http", estado, "codigo", codigo, "correlacion_ref", correlacion,
			"centinela", centinelaSeguroContratacion(causa))
	}
	aplicarCabecerasCobertura(w)
	w.Header().Set("Content-Length", strconv.Itoa(len(contenido)))
	w.WriteHeader(estado)
	_, _ = w.Write(contenido)
}

func responderErrorPreflightFirma(w http.ResponseWriter, r *http.Request, causa error, problema errorPublicoConsultaRRHH) {
	responderJSONFirmaNominal(w, r, problema.estado, map[string]any{"error": map[string]string{
		"codigo": problema.codigo, "clave_i18n": problema.claveI18n, "correlacion_ref": correlacionPeticionFirma(r),
	}}, MaximoRespuestaConsultaRRHHBytes, causa)
}

func camposFirmaNominalExactos(contenido []byte, esperados ...string) bool {
	var campos map[string]json.RawMessage
	if json.Unmarshal(contenido, &campos) != nil || len(campos) != len(esperados) {
		return false
	}
	for _, nombre := range esperados {
		if _, ok := campos[nombre]; !ok || string(campos[nombre]) == "null" {
			return false
		}
	}
	return true
}
