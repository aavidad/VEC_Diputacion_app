package httpinterno

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"unicode/utf8"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const RutaDerivacionNombramiento = "/api/vec/contratacion-temporal/nombramientos/derivaciones"
const maximoCuerpoDerivacionNombramiento = 4096

type ejecutorDerivacionNombramiento interface {
	Derivar(context.Context, ports.EntradaDerivacionNombramiento) (ports.ResultadoDerivacionNombramiento, error)
}

type manejadorDerivacionNombramiento struct {
	ejecutor ejecutorDerivacionNombramiento
}

func NuevoManejadorDerivacionNombramiento(e ejecutorDerivacionNombramiento) (http.Handler, error) {
	if dependenciaNula(e) {
		return nil, ports.ErrDerivacionNombramientoNoDisponible
	}
	return &manejadorDerivacionNombramiento{ejecutor: e}, nil
}

type entradaDerivacionNombramientoJSON struct {
	PadreRef              string `json:"padre_ref"`
	VersionPadre          uint64 `json:"version_padre"`
	OfertaRef             string `json:"oferta_ref"`
	NumeroPlaza           uint32 `json:"numero_plaza"`
	SecuenciaPlaza        uint32 `json:"secuencia_plaza"`
	ParticipacionRef      string `json:"participacion_ref"`
	ReciboAdjudicacionRef string `json:"recibo_adjudicacion_ref"`
	ClaveIdempotencia     string `json:"clave_idempotencia"`
}

func (e entradaDerivacionNombramientoJSON) puerto() ports.EntradaDerivacionNombramiento {
	return ports.EntradaDerivacionNombramiento{
		PadreRef: e.PadreRef, VersionPadre: e.VersionPadre, OfertaRef: e.OfertaRef,
		NumeroPlaza: e.NumeroPlaza, SecuenciaPlaza: e.SecuenciaPlaza,
		ParticipacionRef: e.ParticipacionRef, ReciboAdjudicacionRef: e.ReciboAdjudicacionRef,
		ClaveIdempotencia: e.ClaveIdempotencia,
	}
}

type reciboDerivacionNombramientoJSON struct {
	Esquema       string `json:"esquema"`
	Estado        string `json:"estado"`
	PadreRef      string `json:"padre_ref"`
	HijoRef       string `json:"hijo_ref"`
	NumeroVisible string `json:"numero_visible"`
	NumeroPlaza   uint32 `json:"numero_plaza"`
	TotalPersonas uint32 `json:"total_personas"`
	Version       uint64 `json:"version"`
	AceptacionRef string `json:"aceptacion_ref"`
	ReciboRef     string `json:"recibo_ref"`
	ConfirmadaEn  string `json:"confirmada_en"`
}

func (h *manejadorDerivacionNombramiento) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || dependenciaNula(h.ejecutor) {
		responderErrorDerivacionNombramiento(w, r, errorServicioCoberturaNoDisponible)
		return
	}
	if r == nil || r.URL == nil || r.URL.Path != RutaDerivacionNombramiento ||
		r.URL.EscapedPath() != RutaDerivacionNombramiento || r.URL.RawPath != "" ||
		r.URL.RawQuery != "" || r.URL.ForceQuery || r.URL.Scheme != "" || r.URL.Host != "" ||
		r.URL.User != nil || r.URL.Opaque != "" || r.URL.Fragment != "" || r.URL.RawFragment != "" {
		responderErrorDerivacionNombramiento(w, r, errorRecursoCoberturaNoEncontrado)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		responderErrorDerivacionNombramiento(w, r, errorMetodoCoberturaNoPermitido)
		return
	}
	if r.ContentLength > maximoCuerpoDerivacionNombramiento {
		responderErrorDerivacionNombramiento(w, r, errorCuerpoCoberturaDemasiadoGrande)
		return
	}
	if r.Body == nil || r.Body == http.NoBody || r.ContentLength == 0 ||
		len(r.Trailer) != 0 || !transferenciaAltaPermitida(r.TransferEncoding) ||
		!cabecerasPropuestaFormalizacionPermitidas(r) {
		responderErrorDerivacionNombramiento(w, r, errorPeticionCoberturaNoPermitida)
		return
	}
	if !tipoContenidoJSON(r.Header) || !acceptCompatibleJSON(r.Header) {
		responderErrorDerivacionNombramiento(w, r, errorTipoCoberturaNoAdmitido)
		return
	}
	contenido, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maximoCuerpoDerivacionNombramiento+1))
	if err != nil || len(contenido) > maximoCuerpoDerivacionNombramiento {
		responderErrorDerivacionNombramiento(w, r, errorCuerpoCoberturaDemasiadoGrande, err)
		return
	}
	if !utf8.Valid(contenido) || validarJSONPropuestaFormalizacionSinDuplicados(contenido) != nil {
		responderErrorDerivacionNombramiento(w, r, errorPeticionCoberturaNoValida)
		return
	}
	var entrada entradaDerivacionNombramientoJSON
	dec := json.NewDecoder(bytes.NewReader(contenido))
	dec.DisallowUnknownFields()
	if dec.Decode(&entrada) != nil || dec.Decode(&struct{}{}) != io.EOF || entrada.puerto().Validar() != nil {
		responderErrorDerivacionNombramiento(w, r, errorContenidoCoberturaInvalido)
		return
	}
	resultado, err := h.ejecutor.Derivar(r.Context(), entrada.puerto())
	if err != nil {
		responderErrorDerivacionNombramiento(w, r, clasificarErrorDerivacionNombramiento(err), err)
		return
	}
	if resultado.Esquema != ports.EsquemaResultadoDerivacionNombramiento ||
		(resultado.Estado != "confirmado" && resultado.Estado != "replay_confirmado") ||
		resultado.PadreRef != entrada.PadreRef || resultado.NumeroPlaza != entrada.NumeroPlaza ||
		resultado.ConfirmadaEn.IsZero() {
		responderErrorDerivacionNombramiento(w, r, errorResultadoCoberturaNoConfiable)
		return
	}
	estado := http.StatusCreated
	if resultado.Estado == "replay_confirmado" {
		estado = http.StatusOK
	}
	responderJSONCobertura(w, r, estado, struct {
		Data reciboDerivacionNombramientoJSON `json:"data"`
	}{
		Data: reciboDerivacionNombramientoJSON{resultado.Esquema, resultado.Estado, resultado.PadreRef,
			resultado.HijoRef, resultado.NumeroVisible, resultado.NumeroPlaza, resultado.TotalPersonas,
			resultado.Version, resultado.AceptacionRef, resultado.ReciboRef,
			resultado.ConfirmadaEn.UTC().Format("2006-01-02T15:04:05.999999999Z07:00")},
	})
}

func clasificarErrorDerivacionNombramiento(err error) errorPublicoCobertura {
	switch {
	case errors.Is(err, ports.ErrDerivacionNombramientoDenegada):
		return errorAccesoCoberturaDenegado
	case errors.Is(err, ports.ErrDerivacionNombramientoConflicto), errors.Is(err, ports.ErrDerivacionNombramientoConcurrente):
		return errorConflictoCobertura
	case errors.Is(err, ports.ErrDerivacionNombramientoIndeterminada):
		return errorOperacionCoberturaPendiente
	case errors.Is(err, ports.ErrDerivacionNombramientoInvalida):
		return errorContenidoCoberturaInvalido
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return clasificarErrorCobertura(err)
	default:
		return errorServicioCoberturaNoDisponible
	}
}

func responderErrorDerivacionNombramiento(w http.ResponseWriter, r *http.Request, e errorPublicoCobertura, causas ...error) {
	// Se reutiliza el contrato de errores de CT; ninguna referencia de persona,
	// snapshot o material firmado aparece en la respuesta.
	responderErrorCobertura(w, r, e, causas...)
}

var _ ejecutorDerivacionNombramiento = (*application.ServicioDerivacionNombramiento)(nil)
