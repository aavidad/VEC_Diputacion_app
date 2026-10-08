package httpinterno

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type OperadorOperacionesSituacion interface {
	Operar(context.Context, ports.SolicitudOperacionSituacion) (ports.RegistroSituacionParticipacion, error)
	ListarOperaciones(context.Context, ports.SolicitudCambiarSituacionParticipacion) ([]ports.RegistroOperacionSituacion, error)
}

type HandlerOperacionesSituacion struct {
	preparador PreparadorSituacionParticipacion
	operador   OperadorOperacionesSituacion
	proyector  ProyectorDisponibilidadFichaOperaciones
}

// ProyectorDisponibilidadFichaOperaciones sólo recibe la solicitud del GET
// cuya lectura autorizada y auditada ya concluyó. Su salida no autoriza rutas.
type ProyectorDisponibilidadFichaOperaciones interface {
	ProyectarDisponibilidadFichaOperaciones(context.Context, ports.SolicitudCambiarSituacionParticipacion) (DisponibilidadFichaOperaciones, error)
}

type EstadoDisponibilidadFichaOperaciones struct {
	Estado           string `json:"estado"`
	BolsaRef         string `json:"bolsa_ref"`
	ParticipacionRef string `json:"participacion_ref"`
}

type DisponibilidadFichaOperaciones struct {
	SolicitudesDocumentales  EstadoDisponibilidadFichaOperaciones `json:"solicitudes_documentales"`
	ReincorporacionesTitular EstadoDisponibilidadFichaOperaciones `json:"reincorporaciones_titular"`
}

func NuevoHandlerOperacionesSituacion(p PreparadorSituacionParticipacion, o OperadorOperacionesSituacion, proyectores ...ProyectorDisponibilidadFichaOperaciones) (http.Handler, error) {
	if p == nil || o == nil || len(proyectores) > 1 || len(proyectores) == 1 && proyectores[0] == nil {
		return nil, ports.ErrSituacionParticipacionNoDisponible
	}
	h := &HandlerOperacionesSituacion{preparador: p, operador: o}
	if len(proyectores) == 1 {
		h.proyector = proyectores[0]
	}
	return h, nil
}

func ReferenciasRutaOperacionesSituacion(r *http.Request) (string, string, bool) {
	if r == nil || r.URL == nil || r.URL.RawPath != "" || r.URL.RawQuery != "" || r.RequestURI != r.URL.Path || strings.Contains(r.URL.EscapedPath(), "%") {
		return "", "", false
	}
	partes := strings.Split(strings.TrimPrefix(r.URL.Path, RutaBolsasGestion+"/"), "/")
	if len(partes) != 4 || partes[0] == "" || partes[1] != "candidatos" || partes[2] == "" || partes[3] != "operaciones" {
		return "", "", false
	}
	for _, v := range []string{partes[0], partes[2]} {
		if _, err := url.PathUnescape(v); err != nil || strings.ContainsAny(v, "?# \\%") || len(v) > 512 {
			return "", "", false
		}
	}
	return partes[0], partes[2], true
}

func (h *HandlerOperacionesSituacion) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	bolsa, participacion, ok := ReferenciasRutaOperacionesSituacion(r)
	if !ok {
		responderOperacion(w, 404, "recurso_no_encontrado")
		return
	}
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET, POST")
		responderOperacion(w, 405, "metodo_no_permitido")
		return
	}
	if len(r.Header.Values("Accept")) != 1 || r.Header.Get("Accept") != "application/json" {
		responderOperacion(w, 400, "solicitud_invalida")
		return
	}
	if r.Method == http.MethodGet {
		if r.ContentLength != 0 || len(r.TransferEncoding) != 0 {
			responderOperacion(w, 400, "solicitud_invalida")
			return
		}
		q, err := h.preparador.PrepararSolicitudCambiarSituacion(r.Context(), EntradaCambiarSituacionParticipacion{BolsaRef: bolsa, ParticipacionRef: participacion, Destino: domain.SituacionDisponible, Motivo: "consulta", ClaveIdempotencia: "consulta"})
		if err != nil {
			responderErrorOperacion(w, err)
			return
		}
		items, cambios, vigente, err := listarHistorialOperaciones(r.Context(), h.operador, q)
		if err != nil {
			responderErrorOperacion(w, err)
			return
		}
		salida := make([]map[string]any, 0, len(items))
		for _, o := range items {
			salida = append(salida, map[string]any{"desde": o.Desde.UTC().Format(time.RFC3339Nano), "recibo_ref": o.ReciboRef, "operacion": o.Operacion, "situacion": o.Situacion, "motivo": o.Motivo, "justificante": map[string]string{"tipo": o.Justificante.Tipo, "referencia": o.Justificante.Referencia, "sha256": o.Justificante.SHA256}, "actor": o.Actor, "validador": o.Validador, "validada_en": o.ValidadaEn.UTC().Format(time.RFC3339Nano)})
		}
		datos := map[string]any{"esquema": "vec.bolsa.rrhh.operaciones_situacion.v1", "items": salida}
		if cambios != nil {
			datos["cambios"] = cambios
		}
		if vigente != nil {
			datos["situacion_vigente"] = map[string]any{"situacion": vigente.Situacion, "desde": vigente.Desde.UTC().Format(time.RFC3339Nano), "fecha_disponible": vigente.FechaDisponible}
		}
		if h.proyector != nil {
			disponibilidad, err := h.proyector.ProyectarDisponibilidadFichaOperaciones(r.Context(), q)
			if err != nil || !disponibilidadFichaOperacionesValida(disponibilidad, q) {
				disponibilidad = disponibilidadFichaOperacionesIndisponible(q)
			}
			datos["capacidades_ficha"] = disponibilidad
		}
		responderSituacion(w, 200, map[string]any{"data": datos})
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
	var cuerpo struct {
		Operacion                string `json:"operacion"`
		Motivo                   string `json:"motivo"`
		Validador                string `json:"validador"`
		SituacionEsperadaDesde   string `json:"situacion_esperada_desde"`
		CausaFinalizadaEn        string `json:"causa_finalizada_en"`
		SolicitudRef             string `json:"solicitud_ref"`
		SolicitudVersionEsperada int64  `json:"solicitud_version_esperada"`
		SolicitudContenidoSHA256 string `json:"solicitud_contenido_sha256"`
		Justificante             struct {
			Tipo       string `json:"tipo"`
			Referencia string `json:"referencia"`
			SHA256     string `json:"sha256"`
		} `json:"justificante"`
	}
	dec := json.NewDecoder(io.LimitReader(r.Body, 4097))
	dec.DisallowUnknownFields()
	if dec.Decode(&cuerpo) != nil || dec.Decode(&struct{}{}) != io.EOF || cuerpo.Motivo == "" || strings.TrimSpace(cuerpo.Motivo) != cuerpo.Motivo || len(cuerpo.Motivo) > 1000 || cuerpo.Validador == "" || strings.TrimSpace(cuerpo.Validador) != cuerpo.Validador || len(cuerpo.Validador) > 256 {
		responderOperacion(w, 400, "solicitud_invalida")
		return
	}
	destino, ok := domain.DestinoOperacionSituacion(cuerpo.Operacion)
	j := domain.JustificanteOperacionSituacion{Tipo: cuerpo.Justificante.Tipo, Referencia: cuerpo.Justificante.Referencia, SHA256: cuerpo.Justificante.SHA256}
	if !ok || j.Validar() != nil {
		responderOperacion(w, 400, "solicitud_invalida")
		return
	}
	var esperada time.Time
	var finCausa *time.Time
	if cuerpo.SituacionEsperadaDesde != "" {
		valor, parseErr := time.Parse(time.RFC3339Nano, cuerpo.SituacionEsperadaDesde)
		if parseErr != nil || valor.IsZero() {
			responderOperacion(w, 400, "solicitud_invalida")
			return
		}
		esperada = valor.UTC()
	}
	if cuerpo.CausaFinalizadaEn != "" {
		madrid, locErr := time.LoadLocation("Europe/Madrid")
		if locErr != nil {
			responderOperacion(w, 503, "servicio_no_disponible")
			return
		}
		valor, parseErr := time.ParseInLocation("2006-01-02", cuerpo.CausaFinalizadaEn, madrid)
		if parseErr != nil || valor.IsZero() || valor.Format("2006-01-02") != cuerpo.CausaFinalizadaEn {
			responderOperacion(w, 400, "solicitud_invalida")
			return
		}
		utc := valor.UTC()
		finCausa = &utc
	}
	q, err := h.preparador.PrepararSolicitudCambiarSituacion(r.Context(), EntradaCambiarSituacionParticipacion{BolsaRef: bolsa, ParticipacionRef: participacion, Destino: destino, Motivo: cuerpo.Motivo, ClaveIdempotencia: clave})
	if err != nil {
		responderErrorOperacion(w, err)
		return
	}
	res, err := h.operador.Operar(r.Context(), ports.SolicitudOperacionSituacion{SolicitudCambiarSituacionParticipacion: q, Operacion: cuerpo.Operacion, Justificante: j, Validador: cuerpo.Validador, SituacionEsperadaDesde: esperada, CausaFinalizadaEn: finCausa, SolicitudRef: cuerpo.SolicitudRef, SolicitudVersionEsperada: cuerpo.SolicitudVersionEsperada, SolicitudContenidoSHA256: cuerpo.SolicitudContenidoSHA256})
	if err != nil {
		responderErrorOperacion(w, err)
		return
	}
	status := 201
	if res.Reutilizada {
		status = 200
	}
	datos := map[string]any{"recibo_ref": res.ReciboRef, "situacion": res.Situacion, "desde": res.Desde.UTC().Format(time.RFC3339Nano), "reutilizada": res.Reutilizada}
	if cuerpo.SolicitudRef != "" || res.ReciboResolucionRef != "" {
		if res.ReciboResolucionRef == "" || res.ResueltaEn == nil {
			responderOperacion(w, 503, "servicio_no_disponible")
			return
		}
		datos["recibo_resolucion_ref"] = res.ReciboResolucionRef
		datos["resuelta_en"] = res.ResueltaEn.UTC().Format(time.RFC3339Nano)
	}
	responderSituacion(w, status, map[string]any{"data": datos})
}

func disponibilidadFichaOperacionesIndisponible(q ports.SolicitudCambiarSituacionParticipacion) DisponibilidadFichaOperaciones {
	base := EstadoDisponibilidadFichaOperaciones{Estado: "indisponible", BolsaRef: q.BolsaRef, ParticipacionRef: q.ParticipacionRef}
	return DisponibilidadFichaOperaciones{SolicitudesDocumentales: base, ReincorporacionesTitular: base}
}

func disponibilidadFichaOperacionesValida(d DisponibilidadFichaOperaciones, q ports.SolicitudCambiarSituacionParticipacion) bool {
	for _, estado := range []EstadoDisponibilidadFichaOperaciones{d.SolicitudesDocumentales, d.ReincorporacionesTitular} {
		if estado.BolsaRef != q.BolsaRef || estado.ParticipacionRef != q.ParticipacionRef {
			return false
		}
		switch estado.Estado {
		case "disponible", "no_autorizado", "sin_montaje", "indisponible":
		default:
			return false
		}
	}
	return true
}

func responderOperacion(w http.ResponseWriter, status int, codigo string) {
	responderSituacion(w, status, map[string]any{"error": map[string]string{"codigo": codigo}})
}

func responderErrorOperacion(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, dominiovec.ErrAutorizacionDenegada), errors.Is(err, dominiovec.ErrPermissionDenied):
		responderOperacion(w, 403, "acceso_denegado")
	case errors.Is(err, ports.ErrSituacionParticipacionNoEncontrada):
		responderOperacion(w, 404, "recurso_no_encontrado")
	case errors.Is(err, ports.ErrClaveOperacionReutilizada):
		responderOperacion(w, 409, "clave_reutilizada")
	case errors.Is(err, domain.ErrCambioSituacionParticipacionInvalido):
		responderOperacion(w, 409, "transicion_no_valida")
	case errors.Is(err, domain.ErrOperacionSituacionParticipacionInvalida):
		responderOperacion(w, 400, "solicitud_invalida")
	default:
		responderOperacion(w, 503, "servicio_no_disponible")
	}
}
