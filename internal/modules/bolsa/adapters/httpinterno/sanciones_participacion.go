package httpinterno

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

// GestorSancionesParticipacion es la capacidad de aplicación que consume
// este adaptador; web y otros clientes usan la misma.
type GestorSancionesParticipacion interface {
	Registrar(context.Context, ports.SolicitudRegistrarSancion) (ports.RegistroSancion, error)
	RegistrarRecurso(context.Context, ports.SolicitudRegistrarRecursoSancion) (ports.RegistroRecursoSancion, error)
	Consultar(context.Context, ports.SolicitudCambiarSituacionParticipacion) (ports.VistaSancionesParticipacion, error)
}

type HandlerSancionesParticipacion struct {
	preparador PreparadorSituacionParticipacion
	gestor     GestorSancionesParticipacion
}

func NuevoHandlerSancionesParticipacion(p PreparadorSituacionParticipacion, g GestorSancionesParticipacion) (http.Handler, error) {
	if p == nil || g == nil {
		return nil, ports.ErrSituacionParticipacionNoDisponible
	}
	return &HandlerSancionesParticipacion{p, g}, nil
}

// ReferenciasRutaSancionesParticipacion reconoce
// /{bolsa}/candidatos/{participacion}/sanciones y
// /{bolsa}/candidatos/{participacion}/sanciones/{sancion}/recursos. La
// sanción es vacía en la primera.
func ReferenciasRutaSancionesParticipacion(r *http.Request) (string, string, string, bool) {
	if r == nil || r.URL == nil || r.URL.RawPath != "" || r.URL.RawQuery != "" || r.RequestURI != r.URL.Path || strings.Contains(r.URL.EscapedPath(), "%") ||
		!strings.HasPrefix(r.URL.Path, RutaBolsasGestion+"/") {
		return "", "", "", false
	}
	partes := strings.Split(strings.TrimPrefix(r.URL.Path, RutaBolsasGestion+"/"), "/")
	if len(partes) < 4 || partes[0] == "" || partes[1] != "candidatos" || partes[2] == "" || partes[3] != "sanciones" {
		return "", "", "", false
	}
	sancion := ""
	switch len(partes) {
	case 4:
	case 6:
		if partes[4] == "" || partes[5] != "recursos" {
			return "", "", "", false
		}
		sancion = partes[4]
	default:
		return "", "", "", false
	}
	// La ruta escapada ya no contiene «%»: no hay nada que decodificar.
	for _, v := range []string{partes[0], partes[2], sancion} {
		if strings.ContainsAny(v, "?# \\%") || len(v) > 512 {
			return "", "", "", false
		}
	}
	return partes[0], partes[2], sancion, true
}

func (h *HandlerSancionesParticipacion) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	bolsa, participacion, sancion, ok := ReferenciasRutaSancionesParticipacion(r)
	if !ok {
		responderOperacion(w, 404, "recurso_no_encontrado")
		return
	}
	permitidos := "GET, POST"
	if sancion != "" {
		permitidos = "POST"
	}
	if r.Method != http.MethodPost && (r.Method != http.MethodGet || sancion != "") {
		w.Header().Set("Allow", permitidos)
		responderOperacion(w, 405, "metodo_no_permitido")
		return
	}
	if len(r.Header.Values("Accept")) != 1 || r.Header.Get("Accept") != "application/json" {
		responderOperacion(w, 400, "solicitud_invalida")
		return
	}
	if r.Method == http.MethodGet {
		h.consultar(w, r, bolsa, participacion)
		return
	}
	if r.Body == nil || r.Body == http.NoBody || r.ContentLength <= 0 || r.ContentLength > 4096 || len(r.TransferEncoding) != 0 || len(r.Header.Values("Content-Type")) != 1 || r.Header.Get("Content-Type") != "application/json" {
		responderOperacion(w, 400, "solicitud_invalida")
		return
	}
	clave := r.Header.Get("Idempotency-Key")
	if len(r.Header.Values("Idempotency-Key")) != 1 || clave == "" || strings.TrimSpace(clave) != clave || len(clave) > 256 {
		responderOperacion(w, 400, "solicitud_invalida")
		return
	}
	if sancion == "" {
		h.registrar(w, r, bolsa, participacion, clave)
		return
	}
	h.registrarRecurso(w, r, bolsa, participacion, sancion, clave)
}

func (h *HandlerSancionesParticipacion) consultar(w http.ResponseWriter, r *http.Request, bolsa, participacion string) {
	if r.ContentLength != 0 || len(r.TransferEncoding) != 0 {
		responderOperacion(w, 400, "solicitud_invalida")
		return
	}
	q, err := h.preparador.PrepararSolicitudCambiarSituacion(r.Context(), EntradaCambiarSituacionParticipacion{BolsaRef: bolsa, ParticipacionRef: participacion, Destino: domain.SituacionDisponible, Motivo: "consulta", ClaveIdempotencia: "consulta"})
	if err != nil {
		responderErrorSancion(w, err)
		return
	}
	vista, err := h.gestor.Consultar(r.Context(), q)
	if err != nil {
		responderErrorSancion(w, err)
		return
	}
	consecuencias := make([]map[string]any, 0, len(vista.Consecuencias))
	for _, c := range vista.Consecuencias {
		consecuencias = append(consecuencias, map[string]any{"clave": c.Clave, "etiqueta": c.Etiqueta, "descripcion": c.Descripcion, "efecto": c.Efecto, "articulo": c.Articulo, "ejemplo": c.Ejemplo, "regla_ref": c.ReglaRef,
			"orden_final": c.OrdenFinal, "fin_automatico": c.FinAutomatico})
	}
	items := make([]map[string]any, 0, len(vista.Sanciones))
	for _, s := range vista.Sanciones {
		items = append(items, salidaSancion(s))
	}
	estados, revocatorios := vista.EstadosRecurso, vista.EstadosRevocatorios
	if estados == nil {
		estados = []string{}
	}
	if revocatorios == nil {
		revocatorios = []string{}
	}
	responderSituacion(w, 200, map[string]any{"data": map[string]any{
		"esquema": "vec.bolsa.rrhh.sanciones.v1", "catalogo_disponible": vista.CatalogoDisponible,
		"consecuencias": consecuencias, "estados_recurso": estados, "estados_revocatorios": revocatorios, "items": items,
	}})
}

func salidaSancion(s domain.SancionParticipacion) map[string]any {
	eventos := make([]map[string]any, 0, len(s.Recursos))
	for _, e := range s.Recursos {
		var documento any
		if e.Documento != nil {
			documento = map[string]string{"referencia": e.Documento.Referencia, "sha256": e.Documento.SHA256}
		}
		eventos = append(eventos, map[string]any{"estado": e.Estado, "fecha": e.Fecha, "documento": documento, "actor": e.Actor, "registrada_en": e.RegistradaEn.UTC().Format(time.RFC3339Nano)})
	}
	var suspension, situacionDesde, estadoRecurso any
	if s.SuspensionHasta != "" {
		suspension = s.SuspensionHasta
	}
	if s.SituacionDesde != nil {
		situacionDesde = s.SituacionDesde.UTC().Format(time.RFC3339Nano)
	}
	if estado := s.EstadoRecurso(); estado != "" {
		estadoRecurso = estado
	}
	return map[string]any{
		"efecto_aplicado": salidaEfectoAplicado(s), "reversion": salidaReversion(s.Reversion),
		"sancion_ref": s.SancionRef, "consecuencia": s.Consecuencia, "consecuencia_etiqueta": s.ConsecuenciaEtiqueta,
		"efecto": s.Efecto, "causa": s.Datos.Causa, "fecha_notificacion": s.Datos.FechaNotificacion,
		"resolucion":   map[string]string{"referencia": s.Datos.Resolucion.Referencia, "sha256": s.Datos.Resolucion.SHA256},
		"resuelta_por": s.Datos.ResueltaPor, "regla_ref": s.ReglaRef, "suspension_hasta": suspension,
		"recurso":         map[string]any{"vence": s.RecursoVence, "regla_ref": s.RecursoReglaRef, "estado": estadoRecurso, "eventos": eventos},
		"situacion_desde": situacionDesde, "actor": s.Actor, "registrada_en": s.RegistradaEn.UTC().Format(time.RFC3339Nano),
	}
}

// salidaEfectoAplicado describe lo que la sanción dejó hecho: la situación,
// cuándo vuelve sola al turno y si la colocó al final del orden vigente.
func salidaEfectoAplicado(s domain.SancionParticipacion) map[string]any {
	var situacion, vuelve any
	if s.SituacionAplicada != "" {
		situacion = s.SituacionAplicada
	}
	if s.FechaDisponible != nil {
		vuelve = s.FechaDisponible.UTC().Format(time.RFC3339Nano)
	}
	return map[string]any{"situacion": situacion, "vuelve_al_turno": vuelve, "orden_final": s.OrdenFinal}
}

func salidaReversion(r *domain.ReversionSancion) any {
	if r == nil {
		return nil
	}
	var situacion, desde any
	if r.SituacionRestaurada != "" {
		situacion = r.SituacionRestaurada
	}
	if r.SituacionDesde != nil {
		desde = r.SituacionDesde.UTC().Format(time.RFC3339Nano)
	}
	return map[string]any{
		"estado_recurso": r.EstadoRecurso, "regla_ref": r.ReglaRef, "efecto_revertido": r.EfectoRevertido,
		"situacion_restaurada": situacion, "situacion_desde": desde, "resuelta_por": r.ResueltaPor,
		"actor": r.Actor, "recibo_ref": r.ReciboRef, "registrada_en": r.RegistradaEn.UTC().Format(time.RFC3339Nano),
	}
}

type documentoSancionEntrada struct {
	Referencia string `json:"referencia"`
	SHA256     string `json:"sha256"`
}

func decodificarSancion(r *http.Request, destino any) bool {
	dec := json.NewDecoder(io.LimitReader(r.Body, 4097))
	dec.DisallowUnknownFields()
	return dec.Decode(destino) == nil && dec.Decode(&struct{}{}) == io.EOF
}

func (h *HandlerSancionesParticipacion) registrar(w http.ResponseWriter, r *http.Request, bolsa, participacion, clave string) {
	var cuerpo struct {
		Consecuencia      string                  `json:"consecuencia"`
		Causa             string                  `json:"causa"`
		FechaNotificacion string                  `json:"fecha_notificacion"`
		ResueltaPor       string                  `json:"resuelta_por"`
		Resolucion        documentoSancionEntrada `json:"resolucion"`
	}
	if !decodificarSancion(r, &cuerpo) {
		responderOperacion(w, 400, "solicitud_invalida")
		return
	}
	datos := domain.DatosSancion{Consecuencia: cuerpo.Consecuencia, Causa: cuerpo.Causa, FechaNotificacion: cuerpo.FechaNotificacion,
		ResueltaPor: cuerpo.ResueltaPor, Resolucion: domain.DocumentoSancion{Referencia: cuerpo.Resolucion.Referencia, SHA256: cuerpo.Resolucion.SHA256}}
	if datos.Validar(time.Now()) != nil {
		responderOperacion(w, 400, "solicitud_invalida")
		return
	}
	q, err := h.preparador.PrepararSolicitudCambiarSituacion(r.Context(), EntradaCambiarSituacionParticipacion{BolsaRef: bolsa, ParticipacionRef: participacion, Destino: domain.SituacionDisponible, Motivo: datos.Causa, ClaveIdempotencia: clave})
	if err != nil {
		responderErrorSancion(w, err)
		return
	}
	res, err := h.gestor.Registrar(r.Context(), ports.SolicitudRegistrarSancion{SolicitudCambiarSituacionParticipacion: q, Datos: datos})
	if err != nil {
		responderErrorSancion(w, err)
		return
	}
	var situacion, desde any
	if res.Situacion != "" {
		situacion = res.Situacion
	}
	if res.Desde != nil {
		desde = res.Desde.UTC().Format(time.RFC3339Nano)
	}
	responderSituacion(w, estadoAlta(res.Reutilizada), map[string]any{"data": map[string]any{"sancion_ref": res.SancionRef, "recibo_ref": res.ReciboRef, "situacion": situacion, "desde": desde, "reutilizada": res.Reutilizada}})
}

func (h *HandlerSancionesParticipacion) registrarRecurso(w http.ResponseWriter, r *http.Request, bolsa, participacion, sancion, clave string) {
	var cuerpo struct {
		Estado      string                   `json:"estado"`
		Fecha       string                   `json:"fecha"`
		Documento   *documentoSancionEntrada `json:"documento"`
		ResueltaPor string                   `json:"resuelta_por"`
	}
	if !decodificarSancion(r, &cuerpo) {
		responderOperacion(w, 400, "solicitud_invalida")
		return
	}
	evento := domain.EventoRecursoSancion{Estado: cuerpo.Estado, Fecha: cuerpo.Fecha}
	if cuerpo.Documento != nil {
		evento.Documento = &domain.DocumentoSancion{Referencia: cuerpo.Documento.Referencia, SHA256: cuerpo.Documento.SHA256}
	}
	if evento.ValidarDatos(time.Now()) != nil || (cuerpo.ResueltaPor != "" && !domain.IdentidadResolucionValida(cuerpo.ResueltaPor)) {
		responderOperacion(w, 400, "solicitud_invalida")
		return
	}
	q, err := h.preparador.PrepararSolicitudCambiarSituacion(r.Context(), EntradaCambiarSituacionParticipacion{BolsaRef: bolsa, ParticipacionRef: participacion, Destino: domain.SituacionDisponible, Motivo: "recurso de reposicion", ClaveIdempotencia: clave})
	if err != nil {
		responderErrorSancion(w, err)
		return
	}
	res, err := h.gestor.RegistrarRecurso(r.Context(), ports.SolicitudRegistrarRecursoSancion{SolicitudCambiarSituacionParticipacion: q, SancionRef: sancion, Evento: evento, ResueltaPor: cuerpo.ResueltaPor})
	if err != nil {
		responderErrorSancion(w, err)
		return
	}
	var recibo, situacion, desde any
	if res.Revertida {
		recibo = res.ReciboRef
	}
	if res.Situacion != "" {
		situacion = res.Situacion
	}
	if res.Desde != nil {
		desde = res.Desde.UTC().Format(time.RFC3339Nano)
	}
	responderSituacion(w, estadoAlta(res.Reutilizada), map[string]any{"data": map[string]any{"sancion_ref": res.SancionRef, "estado": res.Estado,
		"registrada_en": res.RegistradaEn.UTC().Format(time.RFC3339Nano), "reutilizada": res.Reutilizada,
		"revertida": res.Revertida, "recibo_ref": recibo, "situacion": situacion, "desde": desde}})
}

func estadoAlta(reutilizada bool) int {
	if reutilizada {
		return 200
	}
	return 201
}

func responderErrorSancion(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ports.ErrSancionesNoConfiguradas):
		responderOperacion(w, 503, "sanciones_no_configuradas")
	case errors.Is(err, domain.ErrSancionParticipacionInvalida):
		responderOperacion(w, 400, "solicitud_invalida")
	default:
		responderErrorOperacion(w, err)
	}
}
