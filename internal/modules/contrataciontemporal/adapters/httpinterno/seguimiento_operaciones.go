package httpinterno

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// Rutas de la fase de seguimiento: cese, cierre del expediente y
// modificación tras el nombramiento, y la consulta de su estado y opciones.
const (
	RutaCesesNombramiento          = "/api/vec/contratacion-temporal/ceses"
	RutaCierresExpediente          = "/api/vec/contratacion-temporal/cierres-expediente"
	RutaModificacionesNombramiento = "/api/vec/contratacion-temporal/modificaciones-nombramiento"
	RutaSeguimientoCese            = "/api/vec/contratacion-temporal/seguimiento-cese"
	// RutaConfirmacionesGINPIX registra el número de alta de GINPIX (CT124).
	RutaConfirmacionesGINPIX = "/api/vec/contratacion-temporal/confirmaciones-ginpix"
	maximoCuerpoSeguimiento  = 16 * 1024
)

// AutoridadCanalSeguimiento resuelve la identidad autenticada del canal; el
// cuerpo nunca aporta actor, perfil ni organización.
type AutoridadCanalSeguimiento interface {
	ResolverContextoCanalSeguimiento(context.Context) (application.ContextoCanalSeguimiento, error)
}

// AutorizadorLecturaSeguimiento acredita, con la consulta de detalle V3, el
// acceso al expediente exacto antes de leer su cese y su cierre.
type AutorizadorLecturaSeguimiento interface {
	AutorizarLecturaSeguimiento(ctx context.Context, organizacionRef, expedienteRef string) error
}

type EjecutorOperacionesSeguimiento interface {
	RegistrarCese(context.Context, application.SolicitudRegistrarCese) (ports.ReciboOperacionSeguimiento, error)
	CerrarExpediente(context.Context, application.SolicitudCerrarExpediente) (ports.ReciboOperacionSeguimiento, error)
	ModificarTrasNombramiento(context.Context, application.SolicitudModificarTrasNombramiento) (ports.ReciboOperacionSeguimiento, error)
	Opciones(context.Context) (ports.OpcionesSeguimiento, error)
	Estado(context.Context, string, string) (ports.EstadoSeguimientoExpediente, error)
}

// EjecutorConfirmacionGINPIX es opcional: solo con la incorporación
// acreditada compuesta existe la ruta de la confirmación de GINPIX.
type EjecutorConfirmacionGINPIX interface {
	ConfirmarGINPIX(context.Context, application.SolicitudConfirmarGINPIX) (ports.ReciboOperacionSeguimiento, error)
}

type manejadorSeguimiento struct {
	ruta      string
	autoridad AutoridadCanalSeguimiento
	lectura   AutorizadorLecturaSeguimiento
	ejecutor  EjecutorOperacionesSeguimiento
}

// NuevosManejadoresSeguimiento devuelve los cuatro manejadores por ruta.
func NuevosManejadoresSeguimiento(a AutoridadCanalSeguimiento, l AutorizadorLecturaSeguimiento, e EjecutorOperacionesSeguimiento) (map[string]http.Handler, error) {
	if dependenciaNula(a) || dependenciaNula(l) || dependenciaNula(e) {
		return nil, errors.New("contratacion temporal http: cese, cierre y modificacion no disponibles")
	}
	m := map[string]http.Handler{}
	for _, ruta := range []string{RutaCesesNombramiento, RutaCierresExpediente, RutaModificacionesNombramiento, RutaSeguimientoCese} {
		m[ruta] = &manejadorSeguimiento{ruta: ruta, autoridad: a, lectura: l, ejecutor: e}
	}
	if _, ok := e.(EjecutorConfirmacionGINPIX); ok {
		m[RutaConfirmacionesGINPIX] = &manejadorSeguimiento{ruta: RutaConfirmacionesGINPIX, autoridad: a, lectura: l, ejecutor: e}
	}
	if _, ok := e.(EjecutorNoIncorporacion); ok {
		m[RutaNoIncorporaciones] = &manejadorSeguimiento{ruta: RutaNoIncorporaciones, autoridad: a, lectura: l, ejecutor: e}
	}
	return m, nil
}

func (h *manejadorSeguimiento) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r == nil || r.URL == nil || r.URL.Path != h.ruta {
		responderErrorSeguimiento(w, r, http.StatusNotFound, "recurso_no_encontrado")
		return
	}
	if r.URL.RawQuery != "" || r.URL.ForceQuery {
		responderErrorSeguimiento(w, r, http.StatusNotFound, "recurso_no_encontrado")
		return
	}
	if r.Method != http.MethodPost {
		responderErrorSeguimiento(w, r, http.StatusMethodNotAllowed, "metodo_no_permitido")
		return
	}
	if !tipoContenidoJSON(r.Header) || cabeceraCoberturaProhibida(r.Header) {
		responderErrorSeguimiento(w, r, http.StatusBadRequest, "peticion_no_permitida")
		return
	}
	contenido, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maximoCuerpoSeguimiento+1))
	if err != nil || len(contenido) == 0 || len(contenido) > maximoCuerpoSeguimiento {
		responderErrorSeguimiento(w, r, http.StatusUnprocessableEntity, "contenido_no_valido")
		return
	}
	canal, err := h.autoridad.ResolverContextoCanalSeguimiento(r.Context())
	if err != nil || !canal.Valido() {
		responderErrorSeguimiento(w, r, http.StatusForbidden, "acceso_denegado")
		return
	}
	if h.ruta == RutaSeguimientoCese {
		h.consultar(w, r, canal, contenido)
		return
	}
	var recibo ports.ReciboOperacionSeguimiento
	switch h.ruta {
	case RutaCesesNombramiento:
		recibo, err = h.cese(r.Context(), canal, contenido)
	case RutaCierresExpediente:
		recibo, err = h.cierre(r.Context(), canal, contenido)
	case RutaConfirmacionesGINPIX:
		recibo, err = h.ginpix(r.Context(), canal, contenido)
	case RutaNoIncorporaciones:
		recibo, err = h.noIncorporacion(r.Context(), canal, contenido)
	default:
		recibo, err = h.modificacion(r.Context(), canal, contenido)
	}
	if err != nil {
		estado, codigo := estadoErrorSeguimiento(err)
		responderErrorSeguimiento(w, r, estado, codigo, err)
		return
	}
	if recibo.OrganizacionRef != canal.OrganizacionRef || !domain.InstanteUTCCanonico(recibo.RegistradaEn) {
		responderErrorSeguimiento(w, r, http.StatusBadGateway, "resultado_no_confiable")
		return
	}
	responderJSONCobertura(w, r, http.StatusCreated, map[string]any{"data": reciboSeguimientoJSON(recibo)})
}

var errContenidoSeguimiento = errors.New("contratacion temporal http: contenido de seguimiento no valido")

func decodificarCuerpoSeguimiento(contenido []byte, destino any) error {
	dec := json.NewDecoder(bytes.NewReader(contenido))
	dec.DisallowUnknownFields()
	if dec.Decode(destino) != nil || dec.Decode(&struct{}{}) != io.EOF {
		return errContenidoSeguimiento
	}
	return nil
}

func fechaCivilSeguimiento(valor string) (time.Time, error) {
	f, err := time.Parse(time.DateOnly, valor)
	if err != nil || f.Format(time.DateOnly) != valor {
		return time.Time{}, errContenidoSeguimiento
	}
	return f.UTC(), nil
}

func (h *manejadorSeguimiento) cese(ctx context.Context, canal application.ContextoCanalSeguimiento, contenido []byte) (ports.ReciboOperacionSeguimiento, error) {
	var in struct {
		ExpedienteRef      string `json:"expediente_ref"`
		VersionEsperada    uint64 `json:"version_esperada"`
		ClaveIdempotencia  string `json:"clave_idempotencia"`
		CausaClave         string `json:"causa_clave"`
		FechaEfecto        string `json:"fecha_efecto"`
		JustificanteRef    string `json:"justificante_ref"`
		JustificanteSHA256 string `json:"justificante_sha256"`
		Observaciones      string `json:"observaciones"`
	}
	if decodificarCuerpoSeguimiento(contenido, &in) != nil {
		return ports.ReciboOperacionSeguimiento{}, errContenidoSeguimiento
	}
	fecha, err := fechaCivilSeguimiento(in.FechaEfecto)
	if err != nil {
		return ports.ReciboOperacionSeguimiento{}, err
	}
	return h.ejecutor.RegistrarCese(ctx, application.SolicitudRegistrarCese{Canal: canal, ExpedienteRef: in.ExpedienteRef,
		VersionEsperada: in.VersionEsperada, ClaveIdempotencia: in.ClaveIdempotencia, CausaClave: domain.ClaveCatalogo(in.CausaClave),
		FechaEfecto: fecha, JustificanteRef: in.JustificanteRef, JustificanteSHA256: in.JustificanteSHA256, Observaciones: in.Observaciones})
}

func (h *manejadorSeguimiento) cierre(ctx context.Context, canal application.ContextoCanalSeguimiento, contenido []byte) (ports.ReciboOperacionSeguimiento, error) {
	var in struct {
		ExpedienteRef      string `json:"expediente_ref"`
		VersionEsperada    uint64 `json:"version_esperada"`
		ClaveIdempotencia  string `json:"clave_idempotencia"`
		GINPIXNumero       string `json:"ginpix_numero"`
		GINPIXConfirmadaEn string `json:"ginpix_confirmada_en"`
		Observaciones      string `json:"observaciones"`
	}
	if decodificarCuerpoSeguimiento(contenido, &in) != nil {
		return ports.ReciboOperacionSeguimiento{}, errContenidoSeguimiento
	}
	var fecha *time.Time
	if in.GINPIXConfirmadaEn != "" {
		f, err := fechaCivilSeguimiento(in.GINPIXConfirmadaEn)
		if err != nil {
			return ports.ReciboOperacionSeguimiento{}, err
		}
		fecha = &f
	}
	return h.ejecutor.CerrarExpediente(ctx, application.SolicitudCerrarExpediente{Canal: canal, ExpedienteRef: in.ExpedienteRef,
		VersionEsperada: in.VersionEsperada, ClaveIdempotencia: in.ClaveIdempotencia, GINPIXNumero: in.GINPIXNumero,
		GINPIXConfirmadaEn: fecha, Observaciones: in.Observaciones})
}

func (h *manejadorSeguimiento) ginpix(ctx context.Context, canal application.ContextoCanalSeguimiento, contenido []byte) (ports.ReciboOperacionSeguimiento, error) {
	ejecutor, ok := h.ejecutor.(EjecutorConfirmacionGINPIX)
	if !ok {
		return ports.ReciboOperacionSeguimiento{}, ports.ErrOperacionSeguimientoNoDisponible
	}
	var in struct {
		ExpedienteRef      string `json:"expediente_ref"`
		VersionEsperada    uint64 `json:"version_esperada"`
		ClaveIdempotencia  string `json:"clave_idempotencia"`
		GINPIXNumero       string `json:"ginpix_numero"`
		GINPIXConfirmadaEn string `json:"ginpix_confirmada_en"`
		Observaciones      string `json:"observaciones"`
	}
	if decodificarCuerpoSeguimiento(contenido, &in) != nil {
		return ports.ReciboOperacionSeguimiento{}, errContenidoSeguimiento
	}
	fecha, err := fechaCivilSeguimiento(in.GINPIXConfirmadaEn)
	if err != nil {
		return ports.ReciboOperacionSeguimiento{}, err
	}
	return ejecutor.ConfirmarGINPIX(ctx, application.SolicitudConfirmarGINPIX{Canal: canal, ExpedienteRef: in.ExpedienteRef,
		VersionEsperada: in.VersionEsperada, ClaveIdempotencia: in.ClaveIdempotencia, GINPIXNumero: in.GINPIXNumero,
		GINPIXConfirmada: fecha, Observaciones: in.Observaciones})
}

func (h *manejadorSeguimiento) modificacion(ctx context.Context, canal application.ContextoCanalSeguimiento, contenido []byte) (ports.ReciboOperacionSeguimiento, error) {
	var in struct {
		ExpedienteRef     string `json:"expediente_ref"`
		VersionEsperada   uint64 `json:"version_esperada"`
		ClaveIdempotencia string `json:"clave_idempotencia"`
		MotivoClave       string `json:"motivo_clave"`
		PeriodoInicio     string `json:"periodo_inicio"`
		PeriodoFin        string `json:"periodo_fin"`
		PorcentajeJornada uint16 `json:"porcentaje_jornada"`
		Observaciones     string `json:"observaciones"`
	}
	if decodificarCuerpoSeguimiento(contenido, &in) != nil {
		return ports.ReciboOperacionSeguimiento{}, errContenidoSeguimiento
	}
	inicio, err := fechaCivilSeguimiento(in.PeriodoInicio)
	if err != nil {
		return ports.ReciboOperacionSeguimiento{}, err
	}
	fin, err := fechaCivilSeguimiento(in.PeriodoFin)
	if err != nil {
		return ports.ReciboOperacionSeguimiento{}, err
	}
	return h.ejecutor.ModificarTrasNombramiento(ctx, application.SolicitudModificarTrasNombramiento{Canal: canal, ExpedienteRef: in.ExpedienteRef,
		VersionEsperada: in.VersionEsperada, ClaveIdempotencia: in.ClaveIdempotencia, MotivoClave: domain.ClaveCatalogo(in.MotivoClave),
		Periodo: domain.PeriodoPrevisto{Inicio: inicio, Fin: fin}, Jornada: domain.JornadaDiezmilesimas(in.PorcentajeJornada), Observaciones: in.Observaciones})
}

// consultar devuelve las opciones de los catálogos y, tras acreditar la
// lectura del expediente exacto, su cese y su cierre.
func (h *manejadorSeguimiento) consultar(w http.ResponseWriter, r *http.Request, canal application.ContextoCanalSeguimiento, contenido []byte) {
	var in struct {
		ExpedienteRef string `json:"expediente_ref"`
	}
	if decodificarCuerpoSeguimiento(contenido, &in) != nil || !domain.ReferenciaOpacaValida(in.ExpedienteRef) {
		responderErrorSeguimiento(w, r, http.StatusUnprocessableEntity, "contenido_no_valido")
		return
	}
	if err := h.lectura.AutorizarLecturaSeguimiento(r.Context(), canal.OrganizacionRef, in.ExpedienteRef); err != nil {
		responderErrorSeguimiento(w, r, http.StatusForbidden, "acceso_denegado", err)
		return
	}
	opciones, err := h.ejecutor.Opciones(r.Context())
	if err != nil {
		responderErrorSeguimiento(w, r, http.StatusServiceUnavailable, "servicio_no_disponible", err)
		return
	}
	estado, err := h.ejecutor.Estado(r.Context(), canal.OrganizacionRef, in.ExpedienteRef)
	if err != nil {
		codigo, clave := estadoErrorSeguimiento(err)
		responderErrorSeguimiento(w, r, codigo, clave, err)
		return
	}
	responderJSONCobertura(w, r, http.StatusOK, map[string]any{"data": map[string]any{
		"esquema": "vec.contratacion-temporal.seguimiento-cese.v1", "opciones": opcionesSeguimientoJSON(opciones), "estado": estadoSeguimientoJSON(estado)}})
}

func estadoErrorSeguimiento(err error) (int, string) {
	if estado, codigo, propio := estadoErrorNoIncorporacion(err); propio {
		return estado, codigo
	}
	switch {
	case errors.Is(err, errContenidoSeguimiento), errors.Is(err, application.ErrSolicitudSeguimientoInvalida), errors.Is(err, ports.ErrOperacionSeguimientoInvalida):
		return http.StatusUnprocessableEntity, "contenido_no_valido"
	case errors.Is(err, ports.ErrAutorizacionDenegada):
		return http.StatusForbidden, "acceso_denegado"
	case errors.Is(err, domain.ErrVersionEnConflicto), errors.Is(err, domain.ErrTransicionInvalida):
		return http.StatusConflict, "version_en_conflicto"
	case errors.Is(err, ports.ErrClaveIdempotenciaUsada):
		return http.StatusConflict, "clave_reutilizada"
	case errors.Is(err, ports.ErrCeseSinIncorporacion):
		return http.StatusConflict, "sin_incorporacion"
	case errors.Is(err, ports.ErrCeseFechaAnteriorIncorporacion):
		return http.StatusConflict, "fecha_anterior_incorporacion"
	case errors.Is(err, ports.ErrCeseYaRegistrado):
		return http.StatusConflict, "cese_existente"
	case errors.Is(err, ports.ErrCierreSinCese):
		return http.StatusConflict, "sin_cese"
	case errors.Is(err, ports.ErrCierreYaRegistrado):
		return http.StatusConflict, "cierre_existente"
	case errors.Is(err, ports.ErrModificacionSinCambios):
		return http.StatusConflict, "sin_cambios"
	case errors.Is(err, ports.ErrModificacionCreditoInsuficiente):
		return http.StatusConflict, "credito_insuficiente"
	case errors.Is(err, ports.ErrGINPIXYaConfirmado):
		return http.StatusConflict, "ginpix_existente"
	case errors.Is(err, ports.ErrGINPIXNoConfirmado):
		return http.StatusConflict, "ginpix_no_confirmado"
	case errors.Is(err, ports.ErrGINPIXDistinto):
		return http.StatusConflict, "ginpix_distinto"
	case errors.Is(err, ports.ErrResultadoSeguimientoNoConfiable):
		return http.StatusBadGateway, "resultado_no_confiable"
	}
	return http.StatusServiceUnavailable, "servicio_no_disponible"
}

func reciboSeguimientoJSON(r ports.ReciboOperacionSeguimiento) map[string]any {
	salida := map[string]any{"esquema": "vec.contratacion-temporal.recibo-seguimiento.v1", "operacion": r.Operacion, "expediente_ref": r.ExpedienteRef,
		"version_anterior": r.VersionAnterior, "version_resultante": r.VersionResultante, "fase_resultante": string(r.FaseResultante),
		"estado_resultante": string(r.EstadoResultante), "recibo_ref": r.ReciboRef, "auditoria_ref": r.AuditoriaRef, "evento_ref": r.EventoRef,
		"registrada_en": r.RegistradaEn.UTC().Format(time.RFC3339Nano)}
	if r.CausaClave != "" {
		salida["causa_clave"], salida["fecha_efecto"] = string(r.CausaClave), r.FechaEfecto
	}
	if r.CeseReciboRef != "" {
		salida["cese_recibo_ref"] = r.CeseReciboRef
	}
	if r.CosteCentimos != 0 {
		salida["coste_centimos"] = r.CosteCentimos
	}
	if r.GINPIXNumero != "" {
		salida["ginpix_numero"] = r.GINPIXNumero
	}
	return salida
}

func opcionesSeguimientoJSON(o ports.OpcionesSeguimiento) map[string]any {
	causas := make([]map[string]string, 0, len(o.Causas))
	for _, c := range o.Causas {
		causas = append(causas, map[string]string{"clave": string(c.Clave), "etiqueta": c.Etiqueta, "clave_i18n": c.ClaveI18n, "justificante_tipo": string(c.JustificanteTipo)})
	}
	motivos := make([]map[string]string, 0, len(o.Motivos))
	for _, m := range o.Motivos {
		motivos = append(motivos, map[string]string{"clave": string(m.Clave), "etiqueta": m.Etiqueta, "clave_i18n": m.ClaveI18n})
	}
	salida := map[string]any{"causas_cese": causas, "condiciones_cierre": append([]string{}, o.Condiciones...),
		"fase_retorno_modificacion": string(o.FaseRetorno), "motivos_modificacion": motivos}
	if o.ConfirmacionGINPIX {
		salida["confirmacion_ginpix"] = true
	}
	if o.NoIncorporacion != nil {
		salida["no_incorporacion"] = opcionesNoIncorporacionJSON(o.NoIncorporacion)
	}
	return salida
}

func estadoSeguimientoJSON(e ports.EstadoSeguimientoExpediente) map[string]any {
	salida := map[string]any{"expediente_ref": e.ExpedienteRef, "incorporacion": nil, "cese": nil, "cierre": nil}
	if e.IncorporacionRef != "" {
		salida["incorporacion"] = map[string]string{"inicio": e.InicioIncorporacion}
	}
	if c := e.Cese; c != nil {
		salida["cese"] = map[string]string{"causa_clave": c.CausaClave, "fecha_efecto": c.FechaEfecto, "justificante_tipo": c.JustificanteTipo,
			"justificante_ref": c.JustificanteRef, "justificante_sha256": c.JustificanteSHA256, "observaciones": c.Observaciones,
			"recibo_ref": c.ReciboRef, "registrada_en": c.RegistradaEn.UTC().Format(time.RFC3339Nano)}
	}
	if c := e.Cierre; c != nil {
		salida["cierre"] = map[string]any{"condiciones": append([]string{}, c.Condiciones...), "ginpix_numero": c.GINPIXNumero,
			"ginpix_confirmada_en": c.GINPIXConfirmadaEn, "observaciones": c.Observaciones, "recibo_ref": c.ReciboRef,
			"registrada_en": c.RegistradaEn.UTC().Format(time.RFC3339Nano)}
	}
	if a := e.Acreditada; a != nil {
		salida["ginpix"], salida["confirmacion_centro"] = nil, nil
		if g := a.GINPIX; g != nil {
			salida["ginpix"] = map[string]string{"ginpix_numero": g.Numero, "ginpix_confirmada_en": g.ConfirmadaEn, "recibo_ref": g.ReciboRef,
				"registrada_en": g.RegistradaEn.UTC().Format(time.RFC3339Nano)}
		}
		if c := a.Centro; c != nil {
			salida["confirmacion_centro"] = map[string]string{"fecha_incorporacion": c.FechaIncorporacion, "documento_tipo": c.DocumentoTipo,
				"documento_ref": c.DocumentoRef, "documento_sha256": c.DocumentoSHA256, "recibo_ref": c.ReciboRef,
				"registrada_en": c.RegistradaEn.UTC().Format(time.RFC3339Nano)}
		}
		if n := a.NoIncorporacion; n != nil {
			salida["no_incorporacion"] = estadoNoIncorporacionJSON(n)
		}
		if p := a.PropuestaNoIncorporacion; p != nil {
			salida["no_incorporacion_propuesta"] = estadoPropuestaNoIncorporacionJSON(p)
		}
		salida["propuestas"] = propuestasExpedienteJSON(a.Propuestas)
	}
	return salida
}

func responderErrorSeguimiento(w http.ResponseWriter, peticion *http.Request, estado int, codigo string, causas ...error) {
	responderJSONCobertura(w, peticion, estado, envoltorioErrorCobertura{Error: detalleErrorCobertura{Codigo: codigo,
		ClaveI18n: "api.contratacion_temporal.seguimiento.error." + codigo, CorrelacionRef: nuevaCorrelacionCobertura()}}, causas...)
}
