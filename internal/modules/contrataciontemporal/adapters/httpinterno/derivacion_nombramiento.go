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

type EjecutorDerivacionNombramiento interface {
	Derivar(context.Context, ports.EntradaDerivacionNombramiento) (ports.ResultadoDerivacionNombramiento, error)
}

type manejadorDerivacionNombramiento struct {
	ejecutor EjecutorDerivacionNombramiento
}

func NuevoManejadorDerivacionNombramiento(e EjecutorDerivacionNombramiento) (http.Handler, error) {
	if dependenciaNula(e) {
		return nil, ports.ErrDerivacionNombramientoNoDisponible
	}
	return &manejadorDerivacionNombramiento{ejecutor: e}, nil
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
	if problema := validarMetadatosPropuestaFormalizacion(r); problema != nil {
		responderErrorDerivacionNombramiento(w, r, *problema)
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
	var entrada ports.EntradaDerivacionNombramiento
	dec := json.NewDecoder(bytes.NewReader(contenido))
	dec.DisallowUnknownFields()
	if dec.Decode(&entrada) != nil || dec.Decode(&struct{}{}) != io.EOF || entrada.Validar() != nil {
		responderErrorDerivacionNombramiento(w, r, errorContenidoCoberturaInvalido)
		return
	}
	resultado, err := h.ejecutor.Derivar(r.Context(), entrada)
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
		Data map[string]any `json:"data"`
	}{Data: map[string]any{
		"esquema": resultado.Esquema, "estado": resultado.Estado,
		"padre_ref": resultado.PadreRef, "hijo_ref": resultado.HijoRef,
		"numero_visible": resultado.NumeroVisible, "numero_plaza": resultado.NumeroPlaza,
		"total_personas": resultado.TotalPersonas, "version": resultado.Version,
		"aceptacion_ref": resultado.AceptacionRef, "recibo_ref": resultado.ReciboRef,
		"confirmada_en": resultado.ConfirmadaEn.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
	}})
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
	e.claveI18n = "api.contratacion_temporal.cobertura.error." + e.codigo
	responderErrorCobertura(w, r, e, causas...)
}

var _ EjecutorDerivacionNombramiento = (*application.ServicioDerivacionNombramiento)(nil)
