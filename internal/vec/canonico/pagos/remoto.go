package pagos

import (
	"errors"
	"io"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

var (
	ErrCapacidadPasarelaCobroNoDisponible = errors.New("vec: capacidad de pasarela de cobro no disponible")
	ErrSolicitudOperacionCobroInvalida    = errors.New("vec: solicitud de operacion de cobro invalida")
	ErrReferenciaOperacionCobroInvalida   = errors.New("vec: referencia de operacion de cobro invalida")
	ErrNotificacionCobroInvalida          = errors.New("vec: notificacion de cobro invalida")
	ErrSolicitudDevolucionCobroInvalida   = errors.New("vec: solicitud de devolucion de cobro invalida")
	ErrSolicitudConciliacionCobroInvalida = errors.New("vec: solicitud de conciliacion de cobro invalida")
	ErrResultadoPasarelaCobroInvalido     = errors.New("vec: resultado de pasarela de cobro invalido")
)

// CapacidadesPasarelaCobro evita suponer funciones de un proveedor concreto.
type CapacidadesPasarelaCobro struct {
	ConectorID              string `json:"conector_id"`
	VersionConector         int    `json:"version_conector"`
	RedireccionAlojada      bool   `json:"redireccion_alojada"`
	NotificacionAutenticada bool   `json:"notificacion_autenticada"`
	ConsultaOperacion       bool   `json:"consulta_operacion"`
	Devolucion              bool   `json:"devolucion"`
	Conciliacion            bool   `json:"conciliacion"`
	Justificante            bool   `json:"justificante"`
	TLSMutuo                bool   `json:"tls_mutuo"`
	IdempotenciaProveedor   bool   `json:"idempotencia_proveedor"`
}

func (c CapacidadesPasarelaCobro) Validar() error {
	if !ClaveValida(c.ConectorID) || c.VersionConector < 1 || !c.RedireccionAlojada ||
		(!c.NotificacionAutenticada && !c.ConsultaOperacion) || !c.IdempotenciaProveedor {
		return ErrCapacidadPasarelaCobroNoDisponible
	}
	return nil
}

type SolicitudOperacionCobro struct {
	Comando domain.ComandoInicioOperacionCobro
}

func (s SolicitudOperacionCobro) Validar() error {
	if s.Comando.Validar() != nil {
		return ErrSolicitudOperacionCobroInvalida
	}
	return nil
}

type ReferenciaOperacionCobro struct {
	ConectorID            string `json:"conector_id"`
	VersionConector       int    `json:"version_conector"`
	OrdenRef              string `json:"orden_ref"`
	OperacionProveedorRef string `json:"operacion_proveedor_ref"`
	CorrelacionRef        string `json:"correlacion_ref"`
}

func (r ReferenciaOperacionCobro) Validar() error {
	if !ClaveValida(r.ConectorID) || r.VersionConector < 1 || !ReferenciaOpacaValida(r.OrdenRef, "cob_") ||
		!TextoValido(r.OperacionProveedorRef, 512) || !TextoValido(r.CorrelacionRef, 512) {
		return ErrReferenciaOperacionCobroInvalida
	}
	return nil
}

// NotificacionCobro solo transporta una referencia opaca a una recepcion
// temporal controlada por el adaptador.
type NotificacionCobro struct {
	ConectorID      string    `json:"conector_id"`
	VersionConector int       `json:"version_conector"`
	RecepcionRef    string    `json:"recepcion_ref"`
	Audiencia       string    `json:"audiencia"`
	RecibidaEn      time.Time `json:"recibida_en"`
}

func (n NotificacionCobro) Validar() error {
	if !ClaveValida(n.ConectorID) || n.VersionConector < 1 || !ReferenciaOpacaValida(n.RecepcionRef, "rec_") ||
		n.Audiencia != audienciaPuertoCobro || n.RecibidaEn.IsZero() {
		return ErrNotificacionCobroInvalida
	}
	return nil
}

type SolicitudCustodiarNotificacionCobro struct {
	ConectorID      string
	VersionConector int
	RecepcionRef    string
	Audiencia       string
	TipoContenido   string
	Tamano          int64
	HuellaSHA256    string
	RecibidaEn      time.Time
	ExpiraEn        time.Time
}

func (s SolicitudCustodiarNotificacionCobro) Validar() error {
	if !ClaveValida(s.ConectorID) || s.VersionConector < 1 || !ReferenciaOpacaValida(s.RecepcionRef, "rec_") ||
		s.Audiencia != audienciaPuertoCobro || !TipoContenidoNotificacionPermitido(s.TipoContenido) ||
		s.Tamano < 1 || s.Tamano > 1024*1024 || !HuellaSHA256Valida(s.HuellaSHA256) || s.RecibidaEn.IsZero() ||
		!s.ExpiraEn.After(s.RecibidaEn) || s.ExpiraEn.Sub(s.RecibidaEn) > 15*time.Minute {
		return ErrNotificacionCobroInvalida
	}
	return nil
}

type ContenidoNotificacionCobroUnico struct {
	Metadatos SolicitudCustodiarNotificacionCobro
	Contenido io.ReadCloser
}

func (c ContenidoNotificacionCobroUnico) Validar() error {
	if c.Metadatos.Validar() != nil || c.Contenido == nil {
		return ErrNotificacionCobroInvalida
	}
	return nil
}

type ResultadoOperacionCobro struct {
	Evidencia domain.EvidenciaResultadoCobro `json:"-"`
}

func (r ResultadoOperacionCobro) Validar() error {
	if r.Evidencia.Validar() != nil {
		return ErrResultadoPasarelaCobroInvalido
	}
	return nil
}

func (ResultadoOperacionCobro) MarshalJSON() ([]byte, error) {
	return nil, ErrResultadoPasarelaCobroInvalido
}

type SolicitudDevolucionCobro struct {
	Comando domain.ComandoDevolucionCobro
}

func (s SolicitudDevolucionCobro) Validar() error {
	if s.Comando.Validar() != nil {
		return ErrSolicitudDevolucionCobroInvalida
	}
	return nil
}

type ResultadoDevolucionCobro struct {
	Evidencia domain.EvidenciaResultadoDevolucionCobro `json:"-"`
}

type ReferenciaDevolucionCobro struct {
	ConectorID            string
	VersionConector       int
	OrdenRef              string
	DevolucionRef         string
	OperacionProveedorRef string
	CorrelacionRef        string
}

func (r ReferenciaDevolucionCobro) Validar() error {
	if !ClaveValida(r.ConectorID) || r.VersionConector < 1 || !ReferenciaOpacaValida(r.OrdenRef, "cob_") ||
		!ReferenciaOpacaValida(r.DevolucionRef, "dev_") || !TextoValido(r.OperacionProveedorRef, 512) ||
		!TextoValido(r.CorrelacionRef, 512) {
		return ErrReferenciaOperacionCobroInvalida
	}
	return nil
}

func (r ResultadoDevolucionCobro) Validar() error {
	if r.Evidencia.Validar() != nil {
		return ErrResultadoPasarelaCobroInvalido
	}
	return nil
}

func (ResultadoDevolucionCobro) MarshalJSON() ([]byte, error) {
	return nil, ErrResultadoPasarelaCobroInvalido
}

type SolicitudConciliacionCobro struct {
	Comando domain.ComandoConciliacionCobro
}

func (s SolicitudConciliacionCobro) Validar() error {
	if s.Comando.Validar() != nil {
		return ErrSolicitudConciliacionCobroInvalida
	}
	return nil
}

type ResultadoConciliacionCobro struct {
	Evidencia domain.EvidenciaConciliacionCobro `json:"-"`
}

func (r ResultadoConciliacionCobro) Validar() error {
	if r.Evidencia.Validar() != nil {
		return ErrResultadoPasarelaCobroInvalido
	}
	return nil
}

func (ResultadoConciliacionCobro) MarshalJSON() ([]byte, error) {
	return nil, ErrResultadoPasarelaCobroInvalido
}
