package ports

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"regexp"
	"strconv"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	vd "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

const (
	EsquemaSolicitudDerivacionNombramiento = "vec.ct.nombramiento.derivar.v1"
	EsquemaResultadoDerivacionNombramiento = "vec.ct.nombramiento.derivado.v1"
	AudienciaDerivacionNombramiento        = "vec.ct.nombramiento-derivado.v1"
	TipoRecursoDerivacionNombramiento      = "nombramiento_derivado_ct"
	MaximoMaterialDerivacionNombramiento   = 256 * 1024
)

var (
	patronOfertaDerivacionNombramiento     = regexp.MustCompile(`^oferta:[0-9a-f]{64}$`)
	ErrDerivacionNombramientoInvalida      = errors.New("contratacion temporal: derivacion de nombramiento invalida")
	ErrDerivacionNombramientoNoConfiable   = errors.New("contratacion temporal: derivacion de nombramiento no confiable")
	ErrDerivacionNombramientoDenegada      = errors.New("contratacion temporal: derivacion de nombramiento denegada")
	ErrDerivacionNombramientoConflicto     = errors.New("contratacion temporal: plaza o clave en conflicto")
	ErrDerivacionNombramientoConcurrente   = errors.New("contratacion temporal: oferta ocupada")
	ErrDerivacionNombramientoIndeterminada = errors.New("contratacion temporal: derivacion indeterminada; recuperar con la misma clave")
	ErrDerivacionNombramientoNoDisponible  = errors.New("contratacion temporal: derivacion no disponible")
)

// Entrada: solo las referencias elegidas por RRHH; el preparador confiable
// resuelve organización, unidad e identidad opaca del hijo.
type EntradaDerivacionNombramiento struct {
	PadreRef              string `json:"padre_ref"`
	VersionPadre          uint64 `json:"version_padre"`
	OfertaRef             string `json:"oferta_ref"`
	NumeroPlaza           uint32 `json:"numero_plaza"`
	SecuenciaPlaza        uint32 `json:"secuencia_plaza"`
	ParticipacionRef      string `json:"participacion_ref"`
	ReciboAdjudicacionRef string `json:"recibo_adjudicacion_ref"`
	ClaveIdempotencia     string `json:"clave_idempotencia"`
}

func (e EntradaDerivacionNombramiento) Validar() error {
	if !domain.ReferenciaOpacaValida(e.PadreRef) || !patronOfertaDerivacionNombramiento.MatchString(e.OfertaRef) ||
		!domain.ReferenciaOpacaValida(e.ParticipacionRef) || !domain.ReferenciaOpacaValida(e.ReciboAdjudicacionRef) ||
		!ClaveIdempotenciaValida(e.ClaveIdempotencia) || e.VersionPadre < 6 || e.VersionPadre > 1<<53-1 ||
		e.NumeroPlaza < 1 || e.NumeroPlaza > 100 || e.SecuenciaPlaza < 1 || e.SecuenciaPlaza > 10000 {
		return ErrDerivacionNombramientoInvalida
	}
	return nil
}

func HijoRefDerivacionNombramiento(organizacionRef, padreRef string, plaza uint32) string {
	h := sha256.Sum256([]byte(organizacionRef + "\n" + padreRef + "\n" + strconv.FormatUint(uint64(plaza), 10)))
	return "expediente:nombramiento:" + hex.EncodeToString(h[:])
}

type SolicitudDerivacionNombramiento struct {
	Esquema               string `json:"esquema"`
	OrganizacionRef       string `json:"organizacion_ref"`
	PadreRef              string `json:"padre_ref"`
	VersionPadre          uint64 `json:"version_padre"`
	HijoRef               string `json:"hijo_ref"`
	OfertaRef             string `json:"oferta_ref"`
	NumeroPlaza           uint32 `json:"numero_plaza"`
	SecuenciaPlaza        uint32 `json:"secuencia_plaza"`
	ParticipacionRef      string `json:"participacion_ref"`
	ReciboAdjudicacionRef string `json:"recibo_adjudicacion_ref"`
	ClaveIdempotencia     string `json:"clave_idempotencia"`
	UnidadRef             string `json:"unidad_ref"`
}

func (s SolicitudDerivacionNombramiento) Entrada() EntradaDerivacionNombramiento {
	return EntradaDerivacionNombramiento{s.PadreRef, s.VersionPadre, s.OfertaRef, s.NumeroPlaza,
		s.SecuenciaPlaza, s.ParticipacionRef, s.ReciboAdjudicacionRef, s.ClaveIdempotencia}
}

func (s SolicitudDerivacionNombramiento) Canon() ([]byte, error) {
	if s.Esquema != EsquemaSolicitudDerivacionNombramiento || s.Entrada().Validar() != nil ||
		!domain.ReferenciaOpacaValida(s.OrganizacionRef) || !domain.ReferenciaOpacaValida(s.UnidadRef) ||
		s.HijoRef != HijoRefDerivacionNombramiento(s.OrganizacionRef, s.PadreRef, s.NumeroPlaza) {
		return nil, ErrDerivacionNombramientoInvalida
	}
	b, err := json.Marshal(s)
	if err != nil || len(b) > MaximoMaterialDerivacionNombramiento {
		return nil, ErrDerivacionNombramientoInvalida
	}
	return b, nil
}

func (s SolicitudDerivacionNombramiento) Recurso() (vd.RecursoAutorizable, error) {
	b, err := s.Canon()
	if err != nil {
		return vd.RecursoAutorizable{}, err
	}
	h := sha256.Sum256(b)
	return vd.RecursoAutorizable{Referencia: s.HijoRef, ModuloID: "contratacion_temporal",
		Tipo:      TipoRecursoDerivacionNombramiento,
		Ambitos:   map[string]string{"organizacion_ref": s.OrganizacionRef, "unidad_ref": s.UnidadRef},
		Atributos: map[string]string{"material_sha256": hex.EncodeToString(h[:])}}, nil
}

type ResultadoDerivacionNombramiento struct {
	Esquema       string            `json:"esquema"`
	Estado        string            `json:"estado"`
	PadreRef      string            `json:"padre_ref"`
	HijoRef       string            `json:"hijo_ref"`
	NumeroVisible string            `json:"numero_visible"`
	NumeroPlaza   uint32            `json:"numero_plaza"`
	TotalPersonas uint32            `json:"total_personas"`
	Version       uint64            `json:"version"`
	AceptacionRef string            `json:"aceptacion_ref"`
	ReciboRef     string            `json:"recibo_ref"`
	AuditoriaRef  string            `json:"auditoria_ref"`
	ConfirmadaEn  time.Time         `json:"confirmada_en"`
	Expediente    domain.Expediente `json:"expediente"`
}

func (r ResultadoDerivacionNombramiento) ValidarPara(s SolicitudDerivacionNombramiento) error {
	if _, err := s.Canon(); err != nil || r.Expediente.InstruccionCompartida == nil {
		return ErrDerivacionNombramientoNoConfiable
	}
	i := r.Expediente.InstruccionCompartida
	padre := i.ExpedientePadre
	if padre.InstruccionCompartida != nil || padre.Validar() != nil ||
		padre.OrganizacionRef != s.OrganizacionRef || padre.Referencia != s.PadreRef ||
		padre.Version != s.VersionPadre || padre.Asignacion == nil || padre.Asignacion.UnidadRef != s.UnidadRef ||
		padre.Fiscalizacion == nil || (padre.Fiscalizacion.Resultado != domain.FiscalizacionFavorable &&
		padre.Fiscalizacion.Resultado != domain.FiscalizacionFavorableConObservaciones) ||
		padre.FaseActual != domain.FaseFiscalizacion || padre.EstadoActual != domain.EstadoEnCurso ||
		padre.Solicitud.NumeroPersonas < 2 || s.NumeroPlaza > padre.Solicitud.NumeroPersonas ||
		r.Esquema != EsquemaResultadoDerivacionNombramiento ||
		(r.Estado != "confirmado" && r.Estado != "replay_confirmado") ||
		r.PadreRef != s.PadreRef || r.HijoRef != s.HijoRef || r.NumeroPlaza != s.NumeroPlaza ||
		r.TotalPersonas != padre.Solicitud.NumeroPersonas || r.Version != 6 ||
		r.NumeroVisible != padre.NumeroVisible+"-P"+strconv.FormatUint(uint64(s.NumeroPlaza), 10) ||
		i.OfertaRef != s.OfertaRef || i.ReciboDerivacionRef != r.ReciboRef ||
		!domain.ReferenciaOpacaValida(r.AceptacionRef) || !domain.ReferenciaOpacaValida(r.ReciboRef) ||
		!domain.ReferenciaOpacaValida(r.AuditoriaRef) || !domain.InstanteUTCCanonico(r.ConfirmadaEn) ||
		r.Expediente.Validar() != nil || len(r.Expediente.Actuaciones) != 1 {
		return ErrDerivacionNombramientoNoConfiable
	}
	a := r.Expediente.Actuaciones[0]
	construido, err := domain.NuevoExpedienteDerivado(domain.AltaExpedienteDerivado{
		Padre: padre, PreimagenPadreJSON: i.PreimagenPadreJSON,
		Referencia: r.HijoRef, OfertaRef: s.OfertaRef, NumeroPlaza: s.NumeroPlaza,
		ActorRef: a.ActorRef, ReciboRef: r.ReciboRef, CreadaEn: r.ConfirmadaEn,
	})
	// Clonar normaliza las listas vacías del JSONB sin cambiar hechos ni
	// recibos. La comparación sigue abarcando todo el agregado hijo.
	if err != nil || !reflect.DeepEqual(construido.Clonar(), r.Expediente.Clonar()) {
		return ErrDerivacionNombramientoNoConfiable
	}
	return nil
}

type PreparadorDerivacionNombramiento interface {
	PrepararDerivacionNombramiento(context.Context, EntradaDerivacionNombramiento) (SolicitudDerivacionNombramiento, error)
}
type AutorizadorDerivacionNombramiento interface {
	AutorizarDerivacionNombramiento(context.Context, SolicitudDerivacionNombramiento) (vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}
type TransaccionDerivacionNombramiento interface {
	ConfirmarDerivacionNombramiento(context.Context, SolicitudDerivacionNombramiento, vp.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ResultadoDerivacionNombramiento, error)
}
