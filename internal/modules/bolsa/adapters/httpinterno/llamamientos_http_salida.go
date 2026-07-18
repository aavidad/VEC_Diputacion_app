package httpinterno

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
)

const (
	esquemaConfirmacionPropuestaLlamamiento = "vec.bolsa.propuesta-llamamiento.confirmacion.v1"
	maximoCuerpoRespuestaLlamamiento        = 8 * 1024
)

var errSalidaLlamamientoInsegura = errors.New(
	"bolsa http interno: salida de propuesta de llamamiento no confiable",
)

type envelopePropuestaLlamamientoJSON struct {
	Data propuestaLlamamientoJSON `json:"data"`
}

type propuestaLlamamientoJSON struct {
	Esquema                         string                       `json:"esquema"`
	PropuestaRef                    string                       `json:"propuesta_ref"`
	HuellaPropuestaSHA256           string                       `json:"huella_propuesta_sha256"`
	Bolsa                           versionHuellaLlamamientoJSON `json:"bolsa"`
	Necesidad                       versionHuellaLlamamientoJSON `json:"necesidad"`
	Instantanea                     versionHuellaLlamamientoJSON `json:"instantanea"`
	Politica                        versionHuellaLlamamientoJSON `json:"politica"`
	InstanteReferencia              string                       `json:"instante_referencia"`
	InstantaneaGeneradaEn           string                       `json:"instantanea_generada_en"`
	TotalParticipacionesInstantanea string                       `json:"total_participaciones_instantanea"`
	TotalEvaluaciones               string                       `json:"total_evaluaciones"`
	OrdenSeleccionado               string                       `json:"orden_seleccionado"`
	GeneradaEn                      string                       `json:"generada_en"`
}

type versionHuellaLlamamientoJSON struct {
	Referencia   string `json:"referencia"`
	Version      string `json:"version"`
	HuellaSHA256 string `json:"huella_sha256"`
}

type envelopeErrorLlamamiento struct {
	Error detalleErrorLlamamiento `json:"error"`
}

type detalleErrorLlamamiento struct {
	Codigo         string `json:"codigo"`
	CorrelacionRef string `json:"correlacion_ref"`
}

func nuevaRespuestaPropuestaLlamamiento(
	origen dominiobolsa.PropuestaLlamamiento,
	necesidadEsperada string,
) (envelopePropuestaLlamamientoJSON, string, error) {
	propuesta, err := origen.ClonarCanonica()
	if err != nil || propuesta.NecesidadRef != necesidadEsperada {
		return envelopePropuestaLlamamientoJSON{}, "", errSalidaLlamamientoInsegura
	}
	respuesta := envelopePropuestaLlamamientoJSON{Data: propuestaLlamamientoJSON{
		Esquema:               esquemaConfirmacionPropuestaLlamamiento,
		PropuestaRef:          propuesta.PropuestaRef,
		HuellaPropuestaSHA256: propuesta.HuellaContenidoSHA256,
		Bolsa: versionHuellaLlamamientoJSON{
			Referencia: propuesta.BolsaRef, Version: decimalLlamamiento(propuesta.VersionBolsa),
			HuellaSHA256: propuesta.HuellaBolsaSHA256,
		},
		Necesidad: versionHuellaLlamamientoJSON{
			Referencia: propuesta.NecesidadRef, Version: decimalLlamamiento(propuesta.VersionNecesidad),
			HuellaSHA256: propuesta.HuellaNecesidadSHA256,
		},
		Instantanea: versionHuellaLlamamientoJSON{
			Referencia: propuesta.InstantaneaRef, Version: decimalLlamamiento(propuesta.VersionInstantanea),
			HuellaSHA256: propuesta.HuellaInstantaneaSHA256,
		},
		Politica: versionHuellaLlamamientoJSON{
			Referencia: propuesta.PoliticaRef, Version: decimalLlamamiento(propuesta.VersionPolitica),
			HuellaSHA256: propuesta.HuellaPoliticaSHA256,
		},
		InstanteReferencia:              instanteLlamamiento(propuesta.InstanteReferencia),
		InstantaneaGeneradaEn:           instanteLlamamiento(propuesta.InstantaneaGeneradaEn),
		TotalParticipacionesInstantanea: decimalLlamamiento(propuesta.TotalParticipacionesInstantanea),
		TotalEvaluaciones:               decimalLlamamiento(uint64(len(propuesta.Evaluaciones))),
		OrdenSeleccionado:               decimalLlamamiento(propuesta.OrdenSeleccionado),
		GeneradaEn:                      instanteLlamamiento(propuesta.GeneradaEn),
	}}
	etag := fmt.Sprintf(`"vec-propuesta-llamamiento-v1.sha256-%s"`, propuesta.HuellaContenidoSHA256)
	return respuesta, etag, nil
}

func decimalLlamamiento(valor uint64) string {
	return strconv.FormatUint(valor, 10)
}

func instanteLlamamiento(valor time.Time) string {
	return valor.UTC().Truncate(time.Microsecond).Format(time.RFC3339Nano)
}

func responderErrorLlamamiento(w http.ResponseWriter, estado int, codigo, correlacion string) {
	if !correlacionBorradorValida(correlacion) {
		correlacion = nuevaCorrelacionErrorLlamamiento()
	}
	responderJSONLlamamiento(w, estado, envelopeErrorLlamamiento{Error: detalleErrorLlamamiento{
		Codigo: codigo, CorrelacionRef: correlacion,
	}}, correlacion)
}

func responderJSONLlamamiento(w http.ResponseWriter, estado int, valor any, correlacion string) {
	contenido, err := json.Marshal(valor)
	if err != nil || len(contenido) > maximoCuerpoRespuestaLlamamiento {
		estado = http.StatusInternalServerError
		w.Header().Del("ETag")
		if !correlacionBorradorValida(correlacion) {
			correlacion = nuevaCorrelacionErrorLlamamiento()
		}
		contenido, _ = json.Marshal(envelopeErrorLlamamiento{Error: detalleErrorLlamamiento{
			Codigo: "error_interno", CorrelacionRef: correlacion,
		}})
	}
	aplicarCabeceras(w)
	w.Header().Set("Content-Length", strconv.Itoa(len(contenido)))
	w.WriteHeader(estado)
	_, _ = w.Write(contenido)
}

func nuevaCorrelacionErrorLlamamiento() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "corr_error_generacion_no_disponible"
	}
	return "corr_" + hex.EncodeToString(bytes)
}
