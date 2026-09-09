package httpinterno

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"time"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const RutaResolucionFormalizacion = "/api/vec/contratacion-temporal/resoluciones-formalizacion"
const EsquemaResolucionFormalizacion = "vec.contratacion-temporal.resolucion-formalizacion.v1"

type EntradaResolucionFormalizacion struct {
	ExpedienteRef             string `json:"expediente_ref"`
	VersionEsperada           uint64 `json:"version_esperada"`
	PropuestaRef              string `json:"propuesta_ref"`
	ClaveIdempotencia         string `json:"clave_idempotencia"`
	NumeroResolucion          string `json:"numero_resolucion"`
	FechaResolucion           string `json:"fecha_resolucion"`
	Motivo                    string `json:"motivo"`
	ConfirmaRevisionPropuesta bool   `json:"confirma_revision_propuesta"`
	ConfirmaEjercicioManual   bool   `json:"confirma_ejercicio_manual"`
}

type ResultadoResolucionFormalizacion struct {
	Esquema                    string    `json:"esquema"`
	Estado                     string    `json:"estado"`
	ExpedienteRef              string    `json:"expediente_ref"`
	VersionResultante          uint64    `json:"version_resultante"`
	PropuestaRef               string    `json:"propuesta_ref"`
	ResolucionFormalizacionRef string    `json:"resolucion_formalizacion_ref"`
	DocumentoResolucionRef     string    `json:"documento_resolucion_ref"`
	DocumentoResolucionVersion uint64    `json:"documento_resolucion_version"`
	DocumentoResolucionSHA256  string    `json:"documento_resolucion_sha256"`
	ActuacionRef               string    `json:"actuacion_ref"`
	AuditoriaRef               string    `json:"auditoria_ref"`
	OutboxRef                  string    `json:"outbox_ref"`
	ReciboRef                  string    `json:"recibo_ref"`
	RegistradaEn               time.Time `json:"registrada_en"`
	TipoValidacion             string    `json:"tipo_validacion"`
	FirmaOficial               bool      `json:"firma_oficial"`
	EficaciaAdministrativa     bool      `json:"eficacia_administrativa"`
}

type EjecutorResolucionFormalizacion = ports.TransaccionResolucionFormalizacion
type AutoridadServidorResolucionFormalizacion interface{ ResolverContextoResolucionFormalizacion(context.Context) error }

func (v EntradaResolucionFormalizacion) solicitud() ports.SolicitudResolucionFormalizacion {
	return ports.SolicitudResolucionFormalizacion{ExpedienteRef: v.ExpedienteRef, VersionEsperada: v.VersionEsperada,
		PropuestaRef: v.PropuestaRef, ClaveIdempotencia: v.ClaveIdempotencia, NumeroResolucion: v.NumeroResolucion,
		FechaResolucion: v.FechaResolucion, Motivo: v.Motivo, ConfirmaRevisionPropuesta: v.ConfirmaRevisionPropuesta,
		ConfirmaEjercicioManual: v.ConfirmaEjercicioManual}
}
func errorHTTPResolucion(w http.ResponseWriter, status int, codigo string) {
	responderJSONCobertura(w, status, map[string]any{"error": map[string]string{"codigo": codigo,
		"clave_i18n": "api.contratacion_temporal.resolucion_formalizacion.error." + codigo, "correlacion_ref": "corr_no_disponible"}})
}
func errorOperacionResolucion(w http.ResponseWriter, e error) {
	switch {
	case errors.Is(e, ports.ErrSolicitudResolucionFormalizacionInvalida):
		errorHTTPResolucion(w, 422, "contenido_no_valido")
	case errors.Is(e, ports.ErrResolucionFormalizacionDenegada), errors.Is(e, ports.ErrAutorizacionDenegada):
		errorHTTPResolucion(w, 403, "acceso_denegado")
	case errors.Is(e, ports.ErrClaveResolucionFormalizacionUsada):
		errorHTTPResolucion(w, 409, "conflicto")
	case errors.Is(e, ports.ErrResolucionFormalizacionEnConflicto):
		errorHTTPResolucion(w, 409, "conflicto")
	default:
		errorHTTPResolucion(w, 503, "servicio_no_disponible")
	}
}
func NuevoManejadorResolucionFormalizacion(a AutoridadServidorResolucionFormalizacion, e EjecutorResolucionFormalizacion) (http.Handler, error) {
	if dependenciaNula(a) || dependenciaNula(e) {
		return nil, ports.ErrResolucionFormalizacionNoDisponible
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r == nil || r.URL == nil || r.URL.Path != RutaResolucionFormalizacion || r.URL.RawPath != "" || (r.Method != http.MethodGet && r.URL.RawQuery != "") ||
			r.URL.ForceQuery || r.URL.Scheme != "" || r.URL.Host != "" || r.URL.User != nil || r.URL.Opaque != "" || r.URL.Fragment != "" || r.URL.RawFragment != "" {
			errorHTTPResolucion(w, 400, "peticion_no_valida")
			return
		}
		if r.Method == http.MethodGet {
			responderPreparacionResolucion(w, r, a, e)
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", "GET, POST")
			errorHTTPResolucion(w, 405, "metodo_no_permitido")
			return
		}
		if r.Context().Err() != nil {
			errorOperacionResolucion(w, r.Context().Err())
			return
		}
		if r.Body == nil || r.Body == http.NoBody || r.ContentLength > 4096 || len(r.Trailer) != 0 ||
			!transferenciaAltaPermitida(r.TransferEncoding) || !cabecerasPropuestaFormalizacionPermitidas(r) ||
			!tipoContenidoJSON(r.Header) || !acceptCompatibleJSON(r.Header) {
			errorHTTPResolucion(w, 400, "peticion_no_valida")
			return
		}
		raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 4096))
		if err != nil || validarJSONPropuestaFormalizacionSinDuplicados(raw) != nil {
			errorHTTPResolucion(w, 400, "peticion_no_valida")
			return
		}
		var in EntradaResolucionFormalizacion
		d := json.NewDecoder(bytes.NewReader(raw))
		d.DisallowUnknownFields()
		if d.Decode(&in) != nil || d.Decode(&struct{}{}) != io.EOF {
			errorHTTPResolucion(w, 400, "peticion_no_valida")
			return
		}
		var campos map[string]json.RawMessage
		if json.Unmarshal(raw, &campos) != nil || len(campos) != 9 {
			errorHTTPResolucion(w, 400, "peticion_no_valida")
			return
		}
		s := in.solicitud()
		if s.Validar() != nil {
			errorHTTPResolucion(w, 422, "contenido_no_valido")
			return
		}
		if err = a.ResolverContextoResolucionFormalizacion(r.Context()); err != nil {
			errorOperacionResolucion(w, err)
			return
		}
		if r.Context().Err() != nil {
			errorOperacionResolucion(w, r.Context().Err())
			return
		}
		out, err := e.RegistrarResolucionFormalizacion(r.Context(), s)
		if r.Context().Err() != nil {
			errorOperacionResolucion(w, r.Context().Err())
			return
		}
		if err != nil {
			if out != (ports.ResultadoResolucionFormalizacion{}) {
				err = ports.ErrResultadoResolucionFormalizacionNoConfiable
			}
			errorOperacionResolucion(w, err)
			return
		}
		if out.ValidarPara(s) != nil {
			errorOperacionResolucion(w, ports.ErrResultadoResolucionFormalizacionNoConfiable)
			return
		}
		status := 201
		if out.Estado == "replay_registrada" {
			status = 200
		}
		responderJSONCobertura(w, status, struct {
			Data ResultadoResolucionFormalizacion `json:"data"`
		}{proyectarResolucionFormalizacion(out)})
	}), nil
}

type preparacionResolucionJSON struct {
	Esquema         string                            `json:"esquema"`
	ExpedienteRef   string                            `json:"expediente_ref"`
	PropuestaRef    string                            `json:"propuesta_ref"`
	VersionEsperada uint64                            `json:"version_esperada"`
	VersionActual   uint64                            `json:"version_actual"`
	Recibo          *ResultadoResolucionFormalizacion `json:"recibo"`
}

func responderPreparacionResolucion(w http.ResponseWriter, r *http.Request, a AutoridadServidorResolucionFormalizacion, e EjecutorResolucionFormalizacion) {
	if len(r.URL.RawQuery) > 600 || r.ContentLength != 0 || len(r.TransferEncoding) != 0 || len(r.Trailer) != 0 ||
		!cabecerasPropuestaFormalizacionPermitidas(r) || !acceptCompatibleJSON(r.Header) {
		errorHTTPResolucion(w, 400, "peticion_no_valida")
		return
	}
	if r.Body != nil && r.Body != http.NoBody {
		b, err := io.ReadAll(io.LimitReader(r.Body, 1))
		if err != nil || len(b) != 0 {
			errorHTTPResolucion(w, 400, "peticion_no_valida")
			return
		}
	}
	q, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil || len(q) != 1 || len(q["expediente_ref"]) != 1 {
		errorHTTPResolucion(w, 400, "peticion_no_valida")
		return
	}
	ref := q.Get("expediente_ref")
	if _, err = ports.NuevaSolicitudDetalleRRHH(ref, 0); err != nil {
		errorHTTPResolucion(w, 400, "peticion_no_valida")
		return
	}
	if r.Context().Err() != nil {
		errorOperacionResolucion(w, r.Context().Err())
		return
	}
	lector, ok := e.(ports.ConsultorPreparacionResolucionFormalizacion)
	if !ok || dependenciaNula(lector) {
		errorOperacionResolucion(w, ports.ErrResolucionFormalizacionNoDisponible)
		return
	}
	if err = a.ResolverContextoResolucionFormalizacion(r.Context()); err != nil {
		errorOperacionResolucion(w, err)
		return
	}
	p, err := lector.ConsultarPreparacionResolucionFormalizacion(r.Context(), ref)
	if r.Context().Err() != nil {
		errorOperacionResolucion(w, r.Context().Err())
		return
	}
	if err != nil {
		errorOperacionResolucion(w, err)
		return
	}
	if p.ValidarPara(ref) != nil {
		errorOperacionResolucion(w, ports.ErrResultadoResolucionFormalizacionNoConfiable)
		return
	}
	out := preparacionResolucionJSON{Esquema: "vec.contratacion-temporal.resolucion-formalizacion.preparacion.v1",
		ExpedienteRef: p.ExpedienteRef, PropuestaRef: p.PropuestaRef, VersionEsperada: 7, VersionActual: p.VersionActual}
	if p.Recibo != nil {
		recibo := proyectarResolucionFormalizacion(*p.Recibo)
		out.Recibo = &recibo
	}
	responderJSONCobertura(w, 200, struct {
		Data preparacionResolucionJSON `json:"data"`
	}{out})
}

func proyectarResolucionFormalizacion(v ports.ResultadoResolucionFormalizacion) ResultadoResolucionFormalizacion {
	return ResultadoResolucionFormalizacion{Esquema: EsquemaResolucionFormalizacion, Estado: v.Estado, ExpedienteRef: v.Solicitud.ExpedienteRef,
		VersionResultante: v.VersionResultante, PropuestaRef: v.Solicitud.PropuestaRef, ResolucionFormalizacionRef: v.ResolucionRef,
		DocumentoResolucionRef: v.DocumentoRef, DocumentoResolucionVersion: v.DocumentoVersion, DocumentoResolucionSHA256: v.DocumentoSHA256,
		ActuacionRef: v.ActuacionRef, AuditoriaRef: v.AuditoriaRef, OutboxRef: v.OutboxRef, ReciboRef: v.ReciboRef,
		RegistradaEn: v.RegistradaEn, TipoValidacion: "manual_de_ejercicio", FirmaOficial: false, EficaciaAdministrativa: false}
}
