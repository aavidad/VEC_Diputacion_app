package httpinterno

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
	"vec-diputacion-granada/internal/shared/plazoarranque"
)

const RutaHistoriaServiciosPropia = "/api/interna/personal/mi-ficha/servicios/historia"
const limitePeticionHistoriaServiciosPropia = 4096

type ConsultorHistoriaServiciosPropia interface {
	Consultar(context.Context, domain.SolicitudHistoriaServiciosPropia) (ports.ResultadoHistoriaServiciosPropia, error)
}
type ManejadorHistoriaServiciosPropia struct {
	actor    ResolutorActorFichaPropia
	consulta ConsultorHistoriaServiciosPropia
	registro ports.RegistroIntentosHistoriaServiciosPropia
	ahora    func() time.Time
}

// La composición debe capturar identidad y correlación originales antes de
// ServeHTTP. El resolutor no acepta datos HTTP ni vuelve a resolver el perfil.
func NuevoManejadorHistoriaServiciosPropia(a ResolutorActorFichaPropia, c ConsultorHistoriaServiciosPropia, r ports.RegistroIntentosHistoriaServiciosPropia, ahora func() time.Time) (*ManejadorHistoriaServiciosPropia, error) {
	if nuloRelacionesDietas(a) || nuloRelacionesDietas(c) || nuloRelacionesDietas(r) || ahora == nil {
		return nil, domain.ErrHistoriaServiciosPropiaNoDisponible
	}
	return &ManejadorHistoriaServiciosPropia{a, c, r, ahora}, nil
}
func (m *ManejadorHistoriaServiciosPropia) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if w == nil {
		return
	}
	if m == nil || r == nil || r.URL == nil || nuloRelacionesDietas(m.actor) || nuloRelacionesDietas(m.consulta) || nuloRelacionesDietas(m.registro) || m.ahora == nil {
		responderFichaPropia(w, 503, "no_disponible", nil)
		return
	}
	actor, err := m.actor.ResolverActorFichaPropia(r.Context())
	if err != nil || actor.Validar() != nil {
		responderFichaPropia(w, 503, "no_disponible", nil)
		return
	}
	if m.registro.VerificarRegistroHistoriaServiciosPropia(r.Context()) != nil {
		m.rechazar(w, r, 503, "no_disponible", "no_disponible")
		return
	}
	if r.URL.Path != RutaHistoriaServiciosPropia || r.URL.RawPath != "" || r.URL.RawQuery != "" || r.URL.ForceQuery {
		m.rechazar(w, r, 404, "no_encontrada", "entrada_invalida")
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		m.rechazar(w, r, 400, "peticion_invalida", "entrada_invalida")
		return
	}
	if cabeceraLibreRelacionesDietas(r.Header) || r.Header.Get("Content-Encoding") != "" || len(r.TransferEncoding) != 0 || r.ContentLength > limitePeticionHistoriaServiciosPropia || len(r.Header.Values("Content-Type")) != 1 {
		m.rechazar(w, r, 400, "peticion_invalida", "entrada_invalida")
		return
	}
	tipo, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || tipo != "application/json" || len(params) > 1 || (params["charset"] != "" && !strings.EqualFold(params["charset"], "utf-8")) || (len(params) == 1 && params["charset"] == "") {
		m.rechazar(w, r, 400, "peticion_invalida", "entrada_invalida")
		return
	}
	corte, err := decodificarPeticionHistoriaServiciosPropia(r.Body)
	if err != nil {
		m.rechazar(w, r, 400, "peticion_invalida", "entrada_invalida")
		return
	}
	corte.ConocidoEn = m.ahora().UTC().Truncate(time.Microsecond)
	solicitud := domain.SolicitudHistoriaServiciosPropia{Actor: actor, Corte: corte}
	material, err := domain.NuevoMaterialHistoriaServiciosPropia(solicitud)
	if err != nil {
		estado, codigo, motivo := 503, "no_disponible", "no_disponible"
		if errors.Is(err, domain.ErrHistoriaServiciosPropiaDenegada) {
			estado, codigo, motivo = 403, "acceso_denegado", "denegado"
		}
		m.rechazar(w, r, estado, codigo, motivo)
		return
	}
	resultado, err := m.consulta.Consultar(r.Context(), solicitud)
	if err != nil {
		if r.Context().Err() != nil {
			return
		}
		estado, codigo := 503, "no_disponible"
		if errors.Is(err, domain.ErrHistoriaServiciosPropiaInvalida) {
			estado, codigo = 400, "peticion_invalida"
		}
		if errors.Is(err, domain.ErrHistoriaServiciosPropiaDenegada) {
			estado, codigo = 403, "acceso_denegado"
		}
		if errors.Is(err, domain.ErrHistoriaServiciosPropiaExcedeLimite) {
			estado, codigo = 422, "excede_limite"
		}
		// El servicio ya registró el fallo después de cerrar la lectura.
		responderFichaPropia(w, estado, codigo, nil)
		return
	}
	if resultado.Historia.ValidarPara(material) != nil || !domain.ReciboHistoriaServiciosPropiaLigado(resultado.Evidencia.ReciboRef, resultado.Evidencia.AuditoriaRef, resultado.Evidencia.ConsumoHuellaSHA256) || resultado.Evidencia.ConsultadaEn.IsZero() || resultado.Evidencia.ConsultadaEn.Nanosecond()%1000 != 0 {
		m.rechazar(w, r, 503, "no_disponible", "no_disponible")
		return
	}
	_, offset := resultado.Evidencia.ConsultadaEn.Zone()
	if offset != 0 || r.Context().Err() != nil {
		m.rechazar(w, r, 503, "no_disponible", "no_disponible")
		return
	}
	// Proyección cerrada: las referencias de empleado y evidencia V3 interna
	// no se serializan. Acto/fuente siguen siendo referencias, no documentos.
	revisiones := make([]map[string]any, 0, len(resultado.Historia.Revisiones))
	for _, revision := range resultado.Historia.Revisiones {
		traza := map[string]any{"desde": revision.Traza.Desde, "registrada_en": revision.Traza.RegistradaEn.UTC().Format("2006-01-02T15:04:05.000000Z"), "version": revision.Traza.Version, "acto_ref": revision.Traza.ActoRef, "fuente_ref": revision.Traza.FuenteRef, "fuente_version": revision.Traza.FuenteVersion}
		if revision.Traza.Hasta != "" {
			traza["hasta"] = revision.Traza.Hasta
		}
		revisiones = append(revisiones, map[string]any{"servicio_ref": revision.ServicioRef, "relacion_ref": revision.RelacionRef, "periodo_desde": revision.PeriodoDesde, "periodo_hasta": revision.PeriodoHasta, "dias_reconocidos": revision.DiasReconocidos, "estado": revision.Estado, "clase": revision.Clase, "traza": traza})
	}
	corteRespuesta := resultado.Historia.Corte
	historia := map[string]any{"corte": map[string]any{"efectos_desde": corteRespuesta.Desde, "efectos_hasta": corteRespuesta.Hasta, "conocido_en": corteRespuesta.ConocidoEn.UTC().Format("2006-01-02T15:04:05.000000Z")}, "cobertura": resultado.Historia.Cobertura, "revisiones": revisiones}

	responderFichaPropia(w, 200, "", map[string]any{"data": map[string]any{"historia": historia, "consultada_en": resultado.Evidencia.ConsultadaEn.UTC().Format("2006-01-02T15:04:05.000000Z"), "recibo_ref": resultado.Evidencia.ReciboRef}})
}
func (m *ManejadorHistoriaServiciosPropia) rechazar(w http.ResponseWriter, r *http.Request, estado int, codigo, motivo string) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), plazoarranque.Ampliar(2*time.Second))
	defer cancel()
	if m.registro.RegistrarIntentoHistoriaServiciosPropia(ctx, ports.IntentoHistoriaServiciosPropia{Motivo: motivo}) != nil {
		estado, codigo = 503, "no_disponible"
	}
	responderFichaPropia(w, estado, codigo, nil)
}
func decodificarPeticionHistoriaServiciosPropia(body io.Reader) (domain.CorteHistoriaServiciosPropia, error) {
	var cero domain.CorteHistoriaServiciosPropia
	if body == nil {
		return cero, domain.ErrHistoriaServiciosPropiaInvalida
	}
	b, err := io.ReadAll(io.LimitReader(body, limitePeticionHistoriaServiciosPropia+1))
	if err != nil || len(b) == 0 || len(b) > limitePeticionHistoriaServiciosPropia || !utf8.Valid(b) || jsonUnicoOrganizacionHistorica(b) != nil {
		return cero, domain.ErrHistoriaServiciosPropiaInvalida
	}
	var obj map[string]json.RawMessage
	if json.Unmarshal(b, &obj) != nil || len(obj) != 2 || obj["efectos_desde"] == nil || obj["efectos_hasta"] == nil {
		return cero, domain.ErrHistoriaServiciosPropiaInvalida
	}
	var desde, hasta string
	if json.Unmarshal(obj["efectos_desde"], &desde) != nil || json.Unmarshal(obj["efectos_hasta"], &hasta) != nil || len(desde) != 10 || len(hasta) != 10 || strings.HasPrefix(desde, "0000") || strings.HasPrefix(hasta, "0000") {
		return cero, domain.ErrHistoriaServiciosPropiaInvalida
	}
	d, err := domain.NuevaFechaCivil(desde)
	if err != nil {
		return cero, domain.ErrHistoriaServiciosPropiaInvalida
	}
	h, err := domain.NuevaFechaCivil(hasta)
	if err != nil || !d.AntesDe(h) {
		return cero, domain.ErrHistoriaServiciosPropiaInvalida
	}
	return domain.CorteHistoriaServiciosPropia{Desde: d, Hasta: h}, nil
}
