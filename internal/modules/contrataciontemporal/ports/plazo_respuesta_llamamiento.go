package ports

import (
	"context"
	"errors"
	"regexp"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

var (
	ErrSolicitudEventoPlazoInvalida    = errors.New("ct_evento_plazo_solicitud_invalida")
	ErrClaveEventoPlazoUsada           = errors.New("ct_evento_plazo_clave_usada")
	ErrEventoPlazoEnConflicto          = errors.New("ct_evento_plazo_en_conflicto")
	ErrOperacionEventoPlazoDenegada    = errors.New("ct_evento_plazo_denegado")
	ErrEventoPlazoNoDisponible         = errors.New("ct_evento_plazo_no_disponible")
	ErrResultadoEventoPlazoNoConfiable = errors.New("ct_evento_plazo_resultado_no_confiable")
	// ErrReglasPlazoNoDisponibles: sin catálogo de reglas vigente o sin
	// cálculo de plazos. Nunca se sustituye por un plazo supuesto.
	ErrReglasPlazoNoDisponibles = errors.New("ct_reglas_plazo_no_disponibles")
	// ErrRespuestaFueraDePlazoLlamamiento: la regla capturada al abrir el
	// plazo no admite la respuesta tardía (o falta la causa acreditada).
	ErrRespuestaFueraDePlazoLlamamiento = errors.New("ct_respuesta_fuera_de_plazo")
	// ErrPlazoRespuestaNoVencido: la expiración solo se confirma tras vencer.
	ErrPlazoRespuestaNoVencido = errors.New("ct_plazo_respuesta_no_vencido")
)

type TipoEventoPlazoLlamamiento string

const (
	// EventoPlazoContactoEfectivo abre el plazo. El aviso por correo no lo es.
	EventoPlazoContactoEfectivo TipoEventoPlazoLlamamiento = "contacto_efectivo"
	// EventoPlazoCausaJustificada acredita la causa que admite una respuesta
	// tardía cuando la regla lo exige. RRHH la declara; VEC no la deduce.
	EventoPlazoCausaJustificada TipoEventoPlazoLlamamiento = "causa_justificada"
)

const (
	EstadoEventoPlazoRegistrado = "registrado"
	EstadoEventoPlazoReplay     = "replay_registrado"
)

var diaCivilPlazo = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}$`)

// SolicitudRegistrarEventoPlazoLlamamiento es lo que declara RRHH. El material
// autorizado es json.Marshal directo: conservar estos nueve campos y nombres.
// El actor procede del contexto confiable, nunca de la solicitud.
type SolicitudRegistrarEventoPlazoLlamamiento struct {
	ClaveIdempotencia           string
	OrganizacionRef             string
	ExpedienteRef               string
	LlamamientoRef              string
	ComunicacionRef             string
	VersionComunicacionEsperada uint64
	Tipo                        TipoEventoPlazoLlamamiento
	InstanteEn                  time.Time
	PruebaRef                   string
}

// Validar comprueba representación. La persistencia rechaza instantes futuros
// o anteriores al aviso con su propio reloj transaccional.
func (s SolicitudRegistrarEventoPlazoLlamamiento) Validar() error {
	if !ClaveIdempotenciaValida(s.ClaveIdempotencia) ||
		!domain.ReferenciaOpacaValida(s.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(s.ExpedienteRef) ||
		!domain.ReferenciaOpacaValida(s.LlamamientoRef) ||
		!domain.ReferenciaOpacaValida(s.ComunicacionRef) ||
		s.VersionComunicacionEsperada != 2 ||
		(s.Tipo != EventoPlazoContactoEfectivo && s.Tipo != EventoPlazoCausaJustificada) ||
		!domain.InstanteUTCCanonico(s.InstanteEn) ||
		!domain.ReferenciaOpacaValida(s.PruebaRef) {
		return ErrSolicitudEventoPlazoInvalida
	}
	return nil
}

// PlazoRespuestaGobernado es el vencimiento calculado en el servidor con las
// reglas del catálogo vigente al abrir el plazo. Politica identifica la regla
// del plazo; CriterioRespuesta y CriterioExpiracion, las que RRHH aplicará al
// resolver una respuesta o al confirmar la expiración. ReglaEjemplo marca que
// alguna de ellas no procede del Reglamento y debe rotularse como ejemplo.
type PlazoRespuestaGobernado struct {
	RespuestaHasta          time.Time
	UltimoDia               string
	Politica                ReferenciaGobernadaComunicacionLlamamiento
	TratamientoFueraDePlazo domain.TratamientoRespuestaFueraDePlazo
	ConfirmacionExpiracion  domain.ConfirmacionExpiracionLlamamiento
	CriterioRespuesta       ReferenciaGobernadaComunicacionLlamamiento
	CriterioExpiracion      ReferenciaGobernadaComunicacionLlamamiento
	ReglaEjemplo            bool
}

func (p PlazoRespuestaGobernado) ValidarDesde(inicio time.Time) error {
	if !domain.InstanteUTCCanonico(p.RespuestaHasta) || !p.RespuestaHasta.After(inicio) ||
		!diaCivilPlazo.MatchString(p.UltimoDia) ||
		p.Politica.Validar() != nil || p.CriterioRespuesta.Validar() != nil ||
		p.CriterioExpiracion.Validar() != nil ||
		!p.TratamientoFueraDePlazo.Valido() || !p.ConfirmacionExpiracion.Valida() {
		return ErrReglasPlazoNoDisponibles
	}
	return nil
}

// EventoPlazoLlamamientoRegistrado acredita solo el registro del hecho. Plazo
// existe únicamente para el contacto efectivo y es el original persistido,
// también en un replay posterior a un cambio de catálogo.
type EventoPlazoLlamamientoRegistrado struct {
	Solicitud    SolicitudRegistrarEventoPlazoLlamamiento
	Plazo        *PlazoRespuestaGobernado `json:",omitempty"`
	EventoRef    string
	ReciboRef    string
	AuditoriaRef string
	RegistradoEn time.Time
	Estado       string
}

func (r EventoPlazoLlamamientoRegistrado) ValidarPara(s SolicitudRegistrarEventoPlazoLlamamiento) error {
	if s.Validar() != nil || r.Solicitud != s ||
		!domain.ReferenciaOpacaValida(r.EventoRef) ||
		!domain.ReferenciaOpacaValida(r.ReciboRef) ||
		!domain.ReferenciaOpacaValida(r.AuditoriaRef) ||
		!domain.InstanteUTCCanonico(r.RegistradoEn) || s.InstanteEn.After(r.RegistradoEn) ||
		(r.Estado != EstadoEventoPlazoRegistrado && r.Estado != EstadoEventoPlazoReplay) ||
		(s.Tipo == EventoPlazoContactoEfectivo) != (r.Plazo != nil) ||
		(r.Plazo != nil && r.Plazo.ValidarDesde(s.InstanteEn) != nil) {
		return ErrResultadoEventoPlazoNoConfiable
	}
	return nil
}

func (r EventoPlazoLlamamientoRegistrado) EsReplay() bool { return r.Estado == EstadoEventoPlazoReplay }

// ReglasPlazoRespuestaLlamamiento calcula el vencimiento desde el contacto
// efectivo con las reglas publicadas. Sin catálogo o sin calendario responde
// ErrReglasPlazoNoDisponibles; nunca inventa un plazo.
type ReglasPlazoRespuestaLlamamiento interface {
	PlazoRespuesta(ctx context.Context, contactoEfectivoEn time.Time) (PlazoRespuestaGobernado, error)
}

// RegistroEventosPlazoLlamamiento une autorización fresca, hecho, recibo y
// auditoría en una transacción. Un evento por tipo y comunicación.
type RegistroEventosPlazoLlamamiento interface {
	RegistrarEventoPlazo(context.Context, SolicitudRegistrarEventoPlazoLlamamiento, *PlazoRespuestaGobernado) (EventoPlazoLlamamientoRegistrado, error)
}
