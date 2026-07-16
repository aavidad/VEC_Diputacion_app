package ports

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"sort"
	"strings"
	"time"
	"unicode"

	"vec-diputacion-granada/internal/vec/domain"
)

var (
	ErrPasarelaCobroNoDisponible          = errors.New("vec: pasarela de cobro no disponible")
	ErrCapacidadPasarelaCobroNoDisponible = errors.New("vec: capacidad de pasarela de cobro no disponible")
	ErrSolicitudOperacionCobroInvalida    = errors.New("vec: solicitud de operacion de cobro invalida")
	ErrInicioOperacionCobroInvalido       = errors.New("vec: inicio de operacion de cobro invalido")
	ErrReferenciaOperacionCobroInvalida   = errors.New("vec: referencia de operacion de cobro invalida")
	ErrNotificacionCobroInvalida          = errors.New("vec: notificacion de cobro invalida")
	ErrSolicitudDevolucionCobroInvalida   = errors.New("vec: solicitud de devolucion de cobro invalida")
	ErrSolicitudConciliacionCobroInvalida = errors.New("vec: solicitud de conciliacion de cobro invalida")
	ErrResultadoPasarelaCobroInvalido     = errors.New("vec: resultado de pasarela de cobro invalido")
	ErrIdempotenciaCobroReutilizada       = errors.New("vec: idempotencia de cobro reutilizada con otros datos")
	ErrResultadoPasarelaCobroConflictivo  = errors.New("vec: resultados de pasarela de cobro incompatibles")
	ErrOrdenCobroNoEncontrada             = errors.New("vec: orden de cobro no encontrada")
	ErrOrdenCobroYaExiste                 = errors.New("vec: orden de cobro ya existente")
	ErrVersionOrdenCobroConflicto         = errors.New("vec: version de orden de cobro en conflicto")
	ErrHuellaOrdenCobroConflicto          = errors.New("vec: huella de orden de cobro en conflicto")
	ErrReservaOrdenCobroInvalida          = errors.New("vec: reserva de orden de cobro invalida")
	ErrReservaOrdenCobroCaducada          = errors.New("vec: reserva de orden de cobro caducada")
	ErrControlAutorizacionCobroConflicto  = errors.New("vec: control autoritativo de autorizacion de cobro en conflicto")
	ErrControlLiquidacionCobroConflicto   = errors.New("vec: control autoritativo de liquidacion en conflicto")
	ErrMutacionOrdenCobroInvalida         = errors.New("vec: mutacion de orden de cobro invalida")
	ErrNotificacionCobroYaConsumida       = errors.New("vec: notificacion de cobro ya consumida")
	ErrNotificacionCobroCaducada          = errors.New("vec: notificacion de cobro caducada")
)

const (
	maximoCaracteresPuertoCobro = 512
	audienciaPuertoCobro        = "vec.cobros"
	dominioHMACAltaPuertoCobro  = "pagos-v1"
	dominioHMACPeticionCobro    = "peticion-v1"
	dominioHMACDevolucionCobro  = "devoluciones-v1"
)

type SolicitudReservaOrdenCobro struct {
	OrdenRef               string
	IndiceIdempotenciaHMAC string
	HuellaSolicitudHMAC    string
	PrincipalRef           string
	SolicitadaEn           time.Time
	ExpiraEn               time.Time
}

func (s SolicitudReservaOrdenCobro) Validar() error {
	if !idOrdenPuertoCobroValido(s.OrdenRef) ||
		!huellaHMACPuertoCobroDeDominioValida(s.IndiceIdempotenciaHMAC, dominioHMACAltaPuertoCobro) ||
		!huellaHMACPuertoCobroDeDominioValida(s.HuellaSolicitudHMAC, dominioHMACPeticionCobro) ||
		!idPersonaPuertoCobroValido(s.PrincipalRef) ||
		s.SolicitadaEn.IsZero() || !s.ExpiraEn.After(s.SolicitadaEn) || s.ExpiraEn.Sub(s.SolicitadaEn) > 15*time.Minute {
		return ErrReservaOrdenCobroInvalida
	}
	return nil
}

type ReservaOrdenCobro struct {
	Token    TokenReservaOrdenCobro
	Repetida bool
	Orden    *domain.OrdenCobro
}

func (r ReservaOrdenCobro) Validar() error {
	if r.Repetida {
		if r.Token.Valido() || r.Orden == nil || r.Orden.Validar() != nil {
			return ErrReservaOrdenCobroInvalida
		}
		return nil
	}
	if !r.Token.Valido() || r.Orden != nil {
		return ErrReservaOrdenCobroInvalida
	}
	return nil
}

func (ReservaOrdenCobro) MarshalJSON() ([]byte, error) { return nil, ErrReservaOrdenCobroInvalida }
func (ReservaOrdenCobro) String() string               { return "[RESERVA-COBRO-INTERNA]" }
func (r ReservaOrdenCobro) GoString() string           { return r.String() }
func (r ReservaOrdenCobro) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, r.String())
}

type RegistroAuditoriaCobro struct {
	ID                          string
	ActorRef                    string
	PerfilActivoRef             string
	DecisionAutorizacionRef     string
	HuellaDecisionSHA256        string
	AutorizacionEmitidaEn       time.Time
	AutorizacionValidaHasta     time.Time
	AutorizacionEvaluadaEn      time.Time
	AtestacionAutenticacionRef  string
	AtestacionEmitidaEn         time.Time
	AtestacionValidaHasta       time.Time
	AutenticacionVerificadaEn   time.Time
	SesionRef                   string
	HuellaSesionHMAC            string
	MetodoAutenticacion         domain.AuthMethod
	GarantiaAutenticacion       domain.AuthAssurance
	Accion                      domain.AccionCobro
	Hecho                       domain.TipoHechoCobro
	OrdenRef                    string
	ExpedienteRef               string
	VersionAnterior             int
	VersionPosterior            int
	HuellaAnteriorSHA256        string
	HuellaPosteriorSHA256       string
	EvidenciaRef                string
	HuellaEvidenciaSHA256       string
	VerificacionEvidenciaRef    string
	HuellaVerificacionSHA256    string
	MetodoVerificacionEvidencia domain.MetodoAutenticacionEvidenciaCobro
	AudienciaEvidencia          string
	EvidenciaEmitidaEn          time.Time
	EvidenciaRecibidaEn         time.Time
	EvidenciaVerificadaEn       time.Time
	Resultado                   string
	Motivo                      string
	CorrelacionRef              string
	OcurridoEn                  time.Time
	Metadatos                   MetadatosAuditoriaCobro
}

// CanalAuditoriaCobro es informativo y nunca concede acceso ni cambia una
// decision. Se mantiene cerrado para evitar convertir metadatos libres en una
// segunda politica de autorizacion accidental.
type CanalAuditoriaCobro string

const (
	CanalAuditoriaCobroInterno           CanalAuditoriaCobro = "interno"
	CanalAuditoriaCobroPasarela          CanalAuditoriaCobro = "pasarela"
	CanalAuditoriaCobroProcesoAutomatico CanalAuditoriaCobro = "proceso_automatico"
)

type MetadatosAuditoriaCobro struct {
	Canal CanalAuditoriaCobro
}

func (m MetadatosAuditoriaCobro) validar() error {
	switch m.Canal {
	case CanalAuditoriaCobroInterno, CanalAuditoriaCobroPasarela, CanalAuditoriaCobroProcesoAutomatico:
		return nil
	default:
		return ErrMutacionOrdenCobroInvalida
	}
}

func (r RegistroAuditoriaCobro) Validar() error {
	canalEsperado, existeCanal := canalAuditoriaCobroParaHecho(r.Hecho, r.Accion)
	if !idPersonaPuertoCobroValido(r.ActorRef) || !idPerfilPuertoCobroValido(r.PerfilActivoRef) ||
		!referenciaPuertoCobroValida(r.DecisionAutorizacionRef) || !huellaSHA256PuertoCobroValida(r.HuellaDecisionSHA256) ||
		r.AutorizacionEmitidaEn.IsZero() || !r.AutorizacionValidaHasta.After(r.AutorizacionEmitidaEn) ||
		r.AutorizacionEvaluadaEn.Before(r.AutorizacionEmitidaEn) ||
		!r.AutorizacionEvaluadaEn.Before(r.AutorizacionValidaHasta) ||
		!referenciaOpacaPuertoCobroValida(r.AtestacionAutenticacionRef, "aut_") ||
		r.AtestacionEmitidaEn.IsZero() || !r.AtestacionValidaHasta.After(r.AtestacionEmitidaEn) ||
		r.AutenticacionVerificadaEn.Before(r.AtestacionEmitidaEn) ||
		!r.AutenticacionVerificadaEn.Before(r.AtestacionValidaHasta) ||
		!referenciaOpacaPuertoCobroValida(r.SesionRef, "ses_") || !huellaSesionPuertoCobroValida(r.HuellaSesionHMAC) ||
		!metodoAutenticacionPuertoCobroPermitido(r.MetodoAutenticacion) ||
		!garantiaAutenticacionPuertoCobroPermitida(r.GarantiaAutenticacion) ||
		!accionAuditoriaPuertoCobroPermitida(r.Accion) || !r.Hecho.Valido() ||
		!domain.TuplaHechoCobroValida(r.Hecho, domain.EstadoCobro(r.Resultado), r.Accion) ||
		!idOrdenPuertoCobroValido(r.OrdenRef) || !referenciaPuertoCobroValida(r.ExpedienteRef) ||
		r.VersionAnterior < 0 || r.VersionPosterior != r.VersionAnterior+1 ||
		!huellaSHA256PuertoCobroValida(r.HuellaAnteriorSHA256) || !huellaSHA256PuertoCobroValida(r.HuellaPosteriorSHA256) ||
		!referenciaPuertoCobroValida(r.EvidenciaRef) || !huellaSHA256PuertoCobroValida(r.HuellaEvidenciaSHA256) ||
		!textoPuertoCobroValido(r.Resultado, 128) || !textoPuertoCobroValido(r.Motivo, maximoCaracteresPuertoCobro) ||
		!referenciaPuertoCobroValida(r.CorrelacionRef) || r.OcurridoEn.IsZero() ||
		r.Metadatos.validar() != nil || !existeCanal || r.Metadatos.Canal != canalEsperado ||
		!matrizEvidenciaAuditoriaCobroValida(r) ||
		r.ID != idDeterministaAuditoriaCobro(r.OrdenRef, r.VersionPosterior, r.HuellaPosteriorSHA256, r.Hecho, r.Accion) {
		return ErrMutacionOrdenCobroInvalida
	}
	return nil
}

func canalAuditoriaCobroParaHecho(hecho domain.TipoHechoCobro, accion domain.AccionCobro) (CanalAuditoriaCobro, bool) {
	switch hecho {
	case domain.HechoCobroOrdenCreada, domain.HechoCobroDevolucionSolicitada, domain.HechoCobroCancelado:
		return CanalAuditoriaCobroInterno, true
	case domain.HechoCobroCaducado:
		return CanalAuditoriaCobroProcesoAutomatico, true
	case domain.HechoCobroOperacionEnviada, domain.HechoCobroResultadoPendiente,
		domain.HechoCobroResultadoDesconocido, domain.HechoCobroConfirmado, domain.HechoCobroRechazado,
		domain.HechoCobroConciliado, domain.HechoCobroDevolucionResultadoPendiente,
		domain.HechoCobroDevolucionResultadoDesconocido, domain.HechoCobroDevolucionRechazada,
		domain.HechoCobroDevuelto, domain.HechoCobroDevolucionConciliada:
		return CanalAuditoriaCobroPasarela, true
	case domain.HechoCobroIncidenciaDetectada, domain.HechoCobroEvidenciaAdicional:
		switch accion {
		case domain.AccionCobroIniciarOperacion, domain.AccionCobroProcesarResultado,
			domain.AccionCobroProcesarDevolucion, domain.AccionCobroConciliar:
			return CanalAuditoriaCobroPasarela, true
		case domain.AccionCobroSolicitarDevolucion, domain.AccionCobroCancelar:
			return CanalAuditoriaCobroInterno, true
		case domain.AccionCobroCaducar:
			return CanalAuditoriaCobroProcesoAutomatico, true
		}
	}
	return "", false
}

func matrizEvidenciaAuditoriaCobroValida(r RegistroAuditoriaCobro) bool {
	remota := r.VerificacionEvidenciaRef != "" || r.HuellaVerificacionSHA256 != "" ||
		r.MetodoVerificacionEvidencia != "" || r.AudienciaEvidencia != "" ||
		!r.EvidenciaEmitidaEn.IsZero() || !r.EvidenciaRecibidaEn.IsZero() || !r.EvidenciaVerificadaEn.IsZero()
	esperadaRemota := false
	switch r.Hecho {
	case domain.HechoCobroOperacionEnviada, domain.HechoCobroResultadoPendiente,
		domain.HechoCobroResultadoDesconocido, domain.HechoCobroConfirmado, domain.HechoCobroRechazado,
		domain.HechoCobroConciliado, domain.HechoCobroDevolucionResultadoPendiente,
		domain.HechoCobroDevolucionResultadoDesconocido, domain.HechoCobroDevolucionRechazada,
		domain.HechoCobroDevuelto, domain.HechoCobroDevolucionConciliada,
		domain.HechoCobroEvidenciaAdicional:
		esperadaRemota = true
	case domain.HechoCobroIncidenciaDetectada:
		switch r.Accion {
		case domain.AccionCobroIniciarOperacion, domain.AccionCobroProcesarResultado,
			domain.AccionCobroProcesarDevolucion, domain.AccionCobroConciliar:
			esperadaRemota = true
		case domain.AccionCobroSolicitarDevolucion, domain.AccionCobroCancelar, domain.AccionCobroCaducar:
			esperadaRemota = false
		default:
			return false
		}
	}
	if remota != esperadaRemota {
		return false
	}
	if !remota {
		return true
	}
	return referenciaPuertoCobroValida(r.VerificacionEvidenciaRef) &&
		huellaSHA256PuertoCobroValida(r.HuellaVerificacionSHA256) &&
		r.MetodoVerificacionEvidencia.Valido() && r.AudienciaEvidencia == audienciaPuertoCobro &&
		!r.EvidenciaEmitidaEn.IsZero() && !r.EvidenciaRecibidaEn.Before(r.EvidenciaEmitidaEn) &&
		!r.EvidenciaVerificadaEn.Before(r.EvidenciaRecibidaEn) &&
		r.EvidenciaVerificadaEn.Sub(r.EvidenciaRecibidaEn) <= 2*time.Minute
}

func idDeterministaAuditoriaCobro(
	ordenRef string,
	version int,
	huellaPosterior string,
	hecho domain.TipoHechoCobro,
	accion domain.AccionCobro,
) string {
	contenido := fmt.Sprintf("vec.cobros.auditoria.v1\x00%s\x00%d\x00%s\x00%s\x00%s",
		ordenRef, version, huellaPosterior, hecho, accion)
	huella := sha256.Sum256([]byte(contenido))
	return fmt.Sprintf("aud_cob_%x", huella)
}

type TipoEventoSalidaCobro string

const (
	EventoCobroOrdenCreada                    TipoEventoSalidaCobro = "cobro.orden.creada"
	EventoCobroOperacionEnviada               TipoEventoSalidaCobro = "cobro.operacion.enviada"
	EventoCobroResultadoPendiente             TipoEventoSalidaCobro = "cobro.resultado.pendiente"
	EventoCobroResultadoDesconocido           TipoEventoSalidaCobro = "cobro.resultado.desconocido"
	EventoCobroConfirmado                     TipoEventoSalidaCobro = "cobro.confirmado"
	EventoCobroRechazado                      TipoEventoSalidaCobro = "cobro.rechazado"
	EventoCobroCancelado                      TipoEventoSalidaCobro = "cobro.cancelado"
	EventoCobroCaducado                       TipoEventoSalidaCobro = "cobro.caducado"
	EventoCobroConciliado                     TipoEventoSalidaCobro = "cobro.conciliado"
	EventoCobroDevolucionSolicitada           TipoEventoSalidaCobro = "cobro.devolucion.solicitada"
	EventoCobroDevolucionResultadoPendiente   TipoEventoSalidaCobro = "cobro.devolucion.resultado_pendiente"
	EventoCobroDevolucionResultadoDesconocido TipoEventoSalidaCobro = "cobro.devolucion.resultado_desconocido"
	EventoCobroDevolucionRechazada            TipoEventoSalidaCobro = "cobro.devolucion.rechazada"
	EventoCobroDevuelto                       TipoEventoSalidaCobro = "cobro.devuelto"
	EventoCobroDevolucionConciliada           TipoEventoSalidaCobro = "cobro.devolucion.conciliada"
	EventoCobroIncidenciaDetectada            TipoEventoSalidaCobro = "cobro.incidencia.detectada"
	EventoCobroEvidenciaAdicional             TipoEventoSalidaCobro = "cobro.evidencia.adicional"
)

func (t TipoEventoSalidaCobro) Valido() bool {
	switch t {
	case EventoCobroOrdenCreada, EventoCobroOperacionEnviada, EventoCobroResultadoPendiente,
		EventoCobroResultadoDesconocido, EventoCobroConfirmado, EventoCobroRechazado,
		EventoCobroCancelado, EventoCobroCaducado, EventoCobroConciliado,
		EventoCobroDevolucionSolicitada, EventoCobroDevolucionResultadoPendiente,
		EventoCobroDevolucionResultadoDesconocido, EventoCobroDevolucionRechazada,
		EventoCobroDevuelto, EventoCobroDevolucionConciliada, EventoCobroIncidenciaDetectada,
		EventoCobroEvidenciaAdicional:
		return true
	default:
		return false
	}
}

// AtributosEventoSalidaCobro sustituye el mapa abierto del outbox. Todos sus
// valores se derivan del ultimo hecho confirmado y no pueden describir un
// resultado distinto del realmente persistido.
type AtributosEventoSalidaCobro struct {
	Hecho  domain.TipoHechoCobro
	Estado domain.EstadoCobro
	Accion domain.AccionCobro
}

func (a AtributosEventoSalidaCobro) validar() error {
	if !a.Hecho.Valido() || !a.Estado.Valido() || !a.Accion.Valida() {
		return ErrMutacionOrdenCobroInvalida
	}
	return nil
}

type EventoSalidaCobro struct {
	ID                string
	Tipo              TipoEventoSalidaCobro
	OrdenRef          string
	VersionOrden      int
	SecuenciaHecho    int64
	HuellaHechoSHA256 string
	HuellaOrdenSHA256 string
	CorrelacionRef    string
	OcurridoEn        time.Time
	Atributos         AtributosEventoSalidaCobro
}

func (e EventoSalidaCobro) Validar() error {
	tipoEsperado, existe := tipoEventoSalidaCobroParaHecho(e.Atributos.Hecho)
	if !e.Tipo.Valido() || !existe || e.Tipo != tipoEsperado || !idOrdenPuertoCobroValido(e.OrdenRef) ||
		e.VersionOrden < 1 || e.SecuenciaHecho < 1 || int64(e.VersionOrden) != e.SecuenciaHecho ||
		!huellaSHA256PuertoCobroValida(e.HuellaHechoSHA256) || e.HuellaHechoSHA256 != e.HuellaOrdenSHA256 ||
		!referenciaPuertoCobroValida(e.CorrelacionRef) || e.OcurridoEn.IsZero() ||
		e.Atributos.validar() != nil ||
		!domain.TuplaHechoCobroValida(e.Atributos.Hecho, e.Atributos.Estado, e.Atributos.Accion) ||
		e.ID != idDeterministaEventoSalidaCobro(e.OrdenRef, e.VersionOrden, e.SecuenciaHecho,
			e.HuellaHechoSHA256, e.Atributos.Hecho, e.Atributos.Estado, e.Atributos.Accion) {
		return ErrMutacionOrdenCobroInvalida
	}
	return nil
}

// NuevoEventoSalidaCobro deriva el mensaje completo, incluido un identificador
// determinista ligado a orden, version, secuencia y huella del ultimo hecho.
// No recibe ningun campo semantico del llamador.
func NuevoEventoSalidaCobro(orden domain.OrdenCobro) (EventoSalidaCobro, error) {
	if orden.Validar() != nil || len(orden.Historial) == 0 {
		return EventoSalidaCobro{}, ErrMutacionOrdenCobroInvalida
	}
	version, huella, err := orden.ControlConcurrencia()
	if err != nil {
		return EventoSalidaCobro{}, ErrMutacionOrdenCobroInvalida
	}
	hecho := orden.Historial[len(orden.Historial)-1]
	tipo, existe := tipoEventoSalidaCobroParaHecho(hecho.Tipo)
	if !existe {
		return EventoSalidaCobro{}, ErrMutacionOrdenCobroInvalida
	}
	evento := EventoSalidaCobro{
		Tipo: tipo, OrdenRef: orden.ID, VersionOrden: version, SecuenciaHecho: hecho.Secuencia,
		HuellaHechoSHA256: hecho.HuellaEstadoPosteriorSHA256,
		HuellaOrdenSHA256: huella, CorrelacionRef: orden.CorrelacionRef,
		OcurridoEn: hecho.OcurridoEn,
		Atributos: AtributosEventoSalidaCobro{
			Hecho: hecho.Tipo, Estado: hecho.EstadoPosterior, Accion: hecho.AccionAutorizada,
		},
	}
	evento.ID = idDeterministaEventoSalidaCobro(evento.OrdenRef, evento.VersionOrden, evento.SecuenciaHecho,
		evento.HuellaHechoSHA256, evento.Atributos.Hecho, evento.Atributos.Estado, evento.Atributos.Accion)
	if evento.Validar() != nil {
		return EventoSalidaCobro{}, ErrMutacionOrdenCobroInvalida
	}
	return evento, nil
}

func idDeterministaEventoSalidaCobro(
	ordenRef string,
	version int,
	secuencia int64,
	huellaHecho string,
	hecho domain.TipoHechoCobro,
	estado domain.EstadoCobro,
	accion domain.AccionCobro,
) string {
	contenido := fmt.Sprintf("vec.cobros.evento.v1\x00%s\x00%d\x00%d\x00%s\x00%s\x00%s\x00%s",
		ordenRef, version, secuencia, huellaHecho, hecho, estado, accion)
	huella := sha256.Sum256([]byte(contenido))
	return fmt.Sprintf("evt_cob_%x", huella)
}

type datosMutacionOrdenCobro struct {
	orden     domain.OrdenCobro
	auditoria RegistroAuditoriaCobro
	evento    EventoSalidaCobro
}

// MutacionOrdenCobro es una unidad opaca para persistir agregado, auditoria y
// outbox en una sola transaccion. El constructor copia el agregado y deriva el
// evento; el llamador no puede compartir ni sustituir memoria interna.
type MutacionOrdenCobro struct{ datos *datosMutacionOrdenCobro }

func (MutacionOrdenCobro) MarshalJSON() ([]byte, error) { return nil, ErrMutacionOrdenCobroInvalida }
func (*MutacionOrdenCobro) UnmarshalJSON([]byte) error  { return ErrMutacionOrdenCobroInvalida }
func (MutacionOrdenCobro) MarshalText() ([]byte, error) { return nil, ErrMutacionOrdenCobroInvalida }
func (MutacionOrdenCobro) String() string               { return "[MUTACION-COBRO-INTERNA]" }
func (m MutacionOrdenCobro) GoString() string           { return m.String() }
func (m MutacionOrdenCobro) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, m.String())
}

type DatosMutacionOrdenCobro struct {
	Orden     domain.OrdenCobro
	Auditoria RegistroAuditoriaCobro
	Evento    EventoSalidaCobro
}

func NuevaMutacionOrdenCobro(orden domain.OrdenCobro) (MutacionOrdenCobro, error) {
	evento, err := NuevoEventoSalidaCobro(orden)
	if err != nil {
		return MutacionOrdenCobro{}, ErrMutacionOrdenCobroInvalida
	}
	auditoria, err := nuevoRegistroAuditoriaCobro(orden)
	if err != nil {
		return MutacionOrdenCobro{}, ErrMutacionOrdenCobroInvalida
	}
	mutacion := MutacionOrdenCobro{datos: &datosMutacionOrdenCobro{
		orden: orden.Clonar(), auditoria: auditoria, evento: evento,
	}}
	if mutacion.Validar() != nil {
		return MutacionOrdenCobro{}, ErrMutacionOrdenCobroInvalida
	}
	return mutacion, nil
}

func nuevoRegistroAuditoriaCobro(orden domain.OrdenCobro) (RegistroAuditoriaCobro, error) {
	if orden.Validar() != nil || len(orden.Historial) == 0 {
		return RegistroAuditoriaCobro{}, ErrMutacionOrdenCobroInvalida
	}
	hecho := orden.Historial[len(orden.Historial)-1]
	canal, existe := canalAuditoriaCobroParaHecho(hecho.Tipo, hecho.AccionAutorizada)
	if !existe {
		return RegistroAuditoriaCobro{}, ErrMutacionOrdenCobroInvalida
	}
	registro := RegistroAuditoriaCobro{
		ActorRef: hecho.ActorRef, PerfilActivoRef: hecho.PerfilActivoRef,
		DecisionAutorizacionRef: hecho.AutorizacionRef, HuellaDecisionSHA256: hecho.HuellaDecisionSHA256,
		AutorizacionEmitidaEn: hecho.AutorizacionEmitidaEn, AutorizacionValidaHasta: hecho.AutorizacionValidaHasta,
		AutorizacionEvaluadaEn:     hecho.AutorizacionEvaluadaEn,
		AtestacionAutenticacionRef: hecho.AtestacionAutenticacionRef,
		AtestacionEmitidaEn:        hecho.AtestacionEmitidaEn, AtestacionValidaHasta: hecho.AtestacionValidaHasta,
		AutenticacionVerificadaEn: hecho.AutenticacionVerificadaEn,
		SesionRef:                 hecho.SesionRef, HuellaSesionHMAC: hecho.HuellaSesionHMAC,
		MetodoAutenticacion: hecho.MetodoAutenticacion, GarantiaAutenticacion: hecho.GarantiaAutenticacion,
		Accion: hecho.AccionAutorizada, Hecho: hecho.Tipo, OrdenRef: orden.ID, ExpedienteRef: orden.ExpedienteRef,
		VersionAnterior: orden.Version - 1, VersionPosterior: orden.Version,
		HuellaAnteriorSHA256:  hecho.HuellaEstadoAnteriorSHA256,
		HuellaPosteriorSHA256: hecho.HuellaEstadoPosteriorSHA256,
		EvidenciaRef:          hecho.EvidenciaRef, HuellaEvidenciaSHA256: hecho.HuellaEvidenciaSHA256,
		VerificacionEvidenciaRef:    hecho.VerificacionEvidenciaRef,
		HuellaVerificacionSHA256:    hecho.HuellaVerificacionSHA256,
		MetodoVerificacionEvidencia: hecho.MetodoVerificacionEvidencia,
		AudienciaEvidencia:          hecho.AudienciaEvidencia,
		EvidenciaEmitidaEn:          hecho.EvidenciaEmitidaEn, EvidenciaRecibidaEn: hecho.EvidenciaRecibidaEn,
		EvidenciaVerificadaEn: hecho.EvidenciaVerificadaEn,
		Resultado:             string(hecho.EstadoPosterior), Motivo: hecho.Motivo,
		CorrelacionRef: orden.CorrelacionRef, OcurridoEn: hecho.OcurridoEn,
		Metadatos: MetadatosAuditoriaCobro{Canal: canal},
	}
	registro.ID = idDeterministaAuditoriaCobro(registro.OrdenRef, registro.VersionPosterior,
		registro.HuellaPosteriorSHA256, registro.Hecho, registro.Accion)
	if registro.Validar() != nil {
		return RegistroAuditoriaCobro{}, ErrMutacionOrdenCobroInvalida
	}
	return registro, nil
}

func (m MutacionOrdenCobro) Datos() (DatosMutacionOrdenCobro, error) {
	if m.Validar() != nil {
		return DatosMutacionOrdenCobro{}, ErrMutacionOrdenCobroInvalida
	}
	return DatosMutacionOrdenCobro{
		Orden: m.datos.orden.Clonar(), Auditoria: m.datos.auditoria, Evento: m.datos.evento,
	}, nil
}

type SolicitudConfirmarCreacionOrdenCobro struct {
	Token                    TokenReservaOrdenCobro
	OrdenRef                 string
	PrincipalRef             string
	IndiceIdempotenciaHMAC   string
	HuellaSolicitudHMAC      string
	ReservaSolicitadaEn      time.Time
	ReservaExpiraEn          time.Time
	DecisionAutorizacionRef  string
	HuellaDecisionSHA256     string
	DecisionValidaHasta      time.Time
	HuellaEfectoSHA256       string
	EvidenciaAutorizacion    EvidenciaUsoDecisionAutorizacion
	ContextoAutorizacion     domain.ContextoAutorizacionCobro
	SesionRef                string
	HuellaSesionHMAC         string
	SesionValidaHasta        time.Time
	LiquidacionRef           string
	LiquidacionRevision      uint64
	LiquidacionHuellaSHA256  string
	LiquidacionEstado        EstadoControlLiquidacionCobro
	LiquidacionExigibleDesde time.Time
	LiquidacionExigibleHasta time.Time
	Mutacion                 MutacionOrdenCobro
}

// EstadoControlLiquidacionCobro es deliberadamente mas estrecho que el
// catalogo funcional de liquidaciones. En el commit de un alta solo existe un
// estado positivo: cualquier otro valor, incluido uno futuro, deniega.
type EstadoControlLiquidacionCobro string

const EstadoControlLiquidacionCobroExigible EstadoControlLiquidacionCobro = "exigible"

func (s SolicitudConfirmarCreacionOrdenCobro) Validar() error {
	datos, errorDatos := s.Mutacion.Datos()
	datosEvidencia, errorEvidencia := s.EvidenciaAutorizacion.Datos()
	decision := datosEvidencia.Decision
	datosVinculo, errorVinculo := decision.VinculoAutenticacionActor.Datos()
	datosContexto, errorContexto := s.ContextoAutorizacion.Datos()
	version, huellaEfecto, err := datos.Orden.ControlConcurrencia()
	if !s.Token.Valido() ||
		!idOrdenPuertoCobroValido(s.OrdenRef) ||
		!idPersonaPuertoCobroValido(s.PrincipalRef) ||
		!huellaHMACPuertoCobroDeDominioValida(s.IndiceIdempotenciaHMAC, dominioHMACAltaPuertoCobro) ||
		!huellaHMACPuertoCobroDeDominioValida(s.HuellaSolicitudHMAC, dominioHMACPeticionCobro) ||
		!instanteConfirmacionCobroCanonico(s.ReservaSolicitadaEn) ||
		!instanteConfirmacionCobroCanonico(s.ReservaExpiraEn) ||
		!s.ReservaExpiraEn.After(s.ReservaSolicitadaEn) ||
		s.ReservaExpiraEn.Sub(s.ReservaSolicitadaEn) > 15*time.Minute ||
		!referenciaPuertoCobroValida(s.DecisionAutorizacionRef) ||
		!huellaSHA256PuertoCobroValida(s.HuellaDecisionSHA256) ||
		!instanteConfirmacionCobroCanonico(s.DecisionValidaHasta) ||
		!huellaSHA256PuertoCobroValida(s.HuellaEfectoSHA256) ||
		errorEvidencia != nil || errorVinculo != nil || errorContexto != nil ||
		s.EvidenciaAutorizacion.ValidarEn(datos.Orden.CreadaEn) != nil ||
		!s.ContextoAutorizacion.CoincideExactamenteConDecision(decision) ||
		!referenciaOpacaPuertoCobroValida(s.SesionRef, "ses_") ||
		!huellaSesionPuertoCobroValida(s.HuellaSesionHMAC) ||
		!instanteConfirmacionCobroCanonico(s.SesionValidaHasta) ||
		!referenciaPuertoCobroValida(s.LiquidacionRef) || s.LiquidacionRevision == 0 ||
		!huellaSHA256PuertoCobroValida(s.LiquidacionHuellaSHA256) ||
		s.LiquidacionEstado != EstadoControlLiquidacionCobroExigible ||
		!instanteConfirmacionCobroCanonico(s.LiquidacionExigibleDesde) ||
		!instanteConfirmacionCobroCanonico(s.LiquidacionExigibleHasta) ||
		!s.LiquidacionExigibleHasta.After(s.LiquidacionExigibleDesde) ||
		errorDatos != nil || err != nil || version != 1 ||
		huellaEfecto != s.HuellaEfectoSHA256 ||
		datos.Auditoria.Accion != domain.AccionCobroCrearOrden ||
		datos.Auditoria.VersionAnterior != 0 ||
		datos.Orden.ID != s.OrdenRef || datos.Orden.IndiceIdempotenciaHMAC != s.IndiceIdempotenciaHMAC ||
		datos.Auditoria.ActorRef != s.PrincipalRef ||
		datos.Auditoria.DecisionAutorizacionRef != s.DecisionAutorizacionRef ||
		datos.Auditoria.HuellaDecisionSHA256 != s.HuellaDecisionSHA256 ||
		!datos.Auditoria.AutorizacionValidaHasta.Equal(s.DecisionValidaHasta) ||
		datosContexto.DecisionRef != s.DecisionAutorizacionRef ||
		datosContexto.ActorRef != s.PrincipalRef ||
		datosContexto.PerfilActivoRef != datos.Auditoria.PerfilActivoRef ||
		datosContexto.Accion != domain.AccionCobroCrearOrden ||
		datosContexto.RecursoRef != s.LiquidacionRef ||
		datosContexto.Finalidad != datos.Orden.Finalidad ||
		datosContexto.CorrelacionRef != datos.Orden.CorrelacionRef ||
		datosContexto.HuellaDecisionSHA256 != s.HuellaDecisionSHA256 ||
		!datosContexto.VigenteHasta.Equal(s.DecisionValidaHasta) ||
		decision.DecisionRef != s.DecisionAutorizacionRef ||
		decision.PrincipalID != s.PrincipalRef ||
		decision.PerfilActivoRef != datos.Auditoria.PerfilActivoRef ||
		decision.Accion != string(domain.AccionCobroCrearOrden) ||
		decision.RecursoRef != s.LiquidacionRef || decision.ModuloID != "pagos" ||
		decision.TipoRecurso != "orden_cobro" ||
		decision.Finalidad != datos.Orden.Finalidad ||
		decision.CorrelacionRef != datos.Orden.CorrelacionRef ||
		decision.ValidaHasta.Before(s.DecisionValidaHasta) ||
		datosEvidencia.VerificadaEn.After(datos.Orden.CreadaEn) ||
		datos.Auditoria.SesionRef != s.SesionRef ||
		datos.Auditoria.HuellaSesionHMAC != s.HuellaSesionHMAC ||
		datosVinculo.SesionRef != s.SesionRef ||
		!datosVinculo.SesionValidaHasta.Equal(s.SesionValidaHasta) ||
		datosVinculo.PrincipalID != s.PrincipalRef ||
		datosVinculo.PerfilActivoRef != datos.Auditoria.PerfilActivoRef ||
		datos.Orden.LiquidacionRef != s.LiquidacionRef ||
		datos.Auditoria.EvidenciaRef != s.LiquidacionRef ||
		datos.Auditoria.HuellaEvidenciaSHA256 != s.LiquidacionHuellaSHA256 ||
		!datos.Orden.CaducaEn.Equal(s.LiquidacionExigibleHasta) ||
		datos.Orden.CreadaEn.Before(s.ReservaSolicitadaEn) ||
		!datos.Orden.CreadaEn.Before(s.ReservaExpiraEn) ||
		datos.Orden.CreadaEn.Before(s.LiquidacionExigibleDesde) ||
		!datos.Orden.CreadaEn.Before(s.LiquidacionExigibleHasta) ||
		!datos.Orden.CreadaEn.Before(s.DecisionValidaHasta) ||
		!datos.Orden.CreadaEn.Before(s.SesionValidaHasta) {
		return ErrMutacionOrdenCobroInvalida
	}
	return nil
}

func instanteConfirmacionCobroCanonico(instante time.Time) bool {
	return !instante.IsZero() && instante.Location() == time.UTC &&
		instante.Equal(instante.Truncate(time.Microsecond))
}

type SolicitudConfirmarTransicionOrdenCobro struct {
	VersionEsperada      int
	HuellaEsperadaSHA256 string
	Mutacion             MutacionOrdenCobro
}

func (s SolicitudConfirmarTransicionOrdenCobro) Validar() error {
	datos, errorDatos := s.Mutacion.Datos()
	version, _, err := datos.Orden.ControlConcurrencia()
	if s.VersionEsperada < 1 || !huellaSHA256PuertoCobroValida(s.HuellaEsperadaSHA256) ||
		errorDatos != nil || err != nil || version != s.VersionEsperada+1 ||
		datos.Auditoria.VersionAnterior != s.VersionEsperada ||
		datos.Auditoria.HuellaAnteriorSHA256 != s.HuellaEsperadaSHA256 ||
		(datos.Auditoria.Accion == domain.AccionCobroCrearOrden ||
			datos.Auditoria.Accion == domain.AccionCobroSolicitarDevolucion) {
		return ErrMutacionOrdenCobroInvalida
	}
	return nil
}

type SolicitudReservaDevolucionCobro struct {
	OrdenRef               string
	DevolucionRef          string
	IndiceIdempotenciaHMAC string
	HuellaSolicitudHMAC    string
	PrincipalRef           string
	SolicitadaEn           time.Time
	ExpiraEn               time.Time
}

func (s SolicitudReservaDevolucionCobro) Validar() error {
	if !idOrdenPuertoCobroValido(s.OrdenRef) || !idDevolucionPuertoCobroValido(s.DevolucionRef) ||
		!huellaHMACPuertoCobroDeDominioValida(s.IndiceIdempotenciaHMAC, dominioHMACDevolucionCobro) ||
		!huellaHMACPuertoCobroDeDominioValida(s.HuellaSolicitudHMAC, dominioHMACPeticionCobro) ||
		!idPersonaPuertoCobroValido(s.PrincipalRef) || s.SolicitadaEn.IsZero() ||
		!s.ExpiraEn.After(s.SolicitadaEn) || s.ExpiraEn.Sub(s.SolicitadaEn) > 15*time.Minute {
		return ErrReservaOrdenCobroInvalida
	}
	return nil
}

type ReservaDevolucionCobro struct {
	Token    TokenReservaDevolucionCobro
	Repetida bool
	Orden    *domain.OrdenCobro
}

func (r ReservaDevolucionCobro) Validar() error {
	if r.Repetida {
		if r.Token.Valido() || r.Orden == nil || r.Orden.Validar() != nil {
			return ErrReservaOrdenCobroInvalida
		}
		return nil
	}
	if !r.Token.Valido() || r.Orden != nil {
		return ErrReservaOrdenCobroInvalida
	}
	return nil
}

func (ReservaDevolucionCobro) MarshalJSON() ([]byte, error) { return nil, ErrReservaOrdenCobroInvalida }
func (ReservaDevolucionCobro) String() string               { return "[RESERVA-DEVOLUCION-INTERNA]" }
func (r ReservaDevolucionCobro) GoString() string           { return r.String() }
func (r ReservaDevolucionCobro) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, r.String())
}

type SolicitudConfirmarReservaDevolucionCobro struct {
	Token                TokenReservaDevolucionCobro
	HuellaSolicitudHMAC  string
	VersionEsperada      int
	HuellaEsperadaSHA256 string
	Mutacion             MutacionOrdenCobro
}

func (s SolicitudConfirmarReservaDevolucionCobro) Validar() error {
	datos, errorDatos := s.Mutacion.Datos()
	version, _, err := datos.Orden.ControlConcurrencia()
	if !s.Token.Valido() ||
		!huellaHMACPuertoCobroDeDominioValida(s.HuellaSolicitudHMAC, dominioHMACPeticionCobro) ||
		s.VersionEsperada < 1 || !huellaSHA256PuertoCobroValida(s.HuellaEsperadaSHA256) ||
		errorDatos != nil || err != nil || version != s.VersionEsperada+1 ||
		datos.Auditoria.VersionAnterior != s.VersionEsperada ||
		datos.Auditoria.HuellaAnteriorSHA256 != s.HuellaEsperadaSHA256 ||
		datos.Auditoria.Accion != domain.AccionCobroSolicitarDevolucion {
		return ErrMutacionOrdenCobroInvalida
	}
	return nil
}

type SolicitudAbandonarReservaOrdenCobro struct {
	Token               TokenReservaOrdenCobro
	OrdenRef            string
	PrincipalRef        string
	HuellaSolicitudHMAC string
}

func (s SolicitudAbandonarReservaOrdenCobro) Validar() error {
	if !s.Token.Valido() || !idOrdenPuertoCobroValido(s.OrdenRef) ||
		!idPersonaPuertoCobroValido(s.PrincipalRef) ||
		!huellaHMACPuertoCobroDeDominioValida(s.HuellaSolicitudHMAC, dominioHMACPeticionCobro) {
		return ErrReservaOrdenCobroInvalida
	}
	return nil
}

type SolicitudAbandonarReservaDevolucionCobro struct {
	Token               TokenReservaDevolucionCobro
	OrdenRef            string
	DevolucionRef       string
	PrincipalRef        string
	HuellaSolicitudHMAC string
}

func (s SolicitudAbandonarReservaDevolucionCobro) Validar() error {
	if !s.Token.Valido() || !idOrdenPuertoCobroValido(s.OrdenRef) ||
		!idDevolucionPuertoCobroValido(s.DevolucionRef) || !idPersonaPuertoCobroValido(s.PrincipalRef) ||
		!huellaHMACPuertoCobroDeDominioValida(s.HuellaSolicitudHMAC, dominioHMACPeticionCobro) {
		return ErrReservaOrdenCobroInvalida
	}
	return nil
}

func (m MutacionOrdenCobro) Validar() error {
	if m.datos == nil {
		return ErrMutacionOrdenCobroInvalida
	}
	orden, auditoria, evento := m.datos.orden, m.datos.auditoria, m.datos.evento
	if orden.Validar() != nil || auditoria.Validar() != nil || evento.Validar() != nil {
		return ErrMutacionOrdenCobroInvalida
	}
	vista, err := orden.VistaTitular()
	version, huella, errControl := orden.ControlConcurrencia()
	ultimoHecho := orden.Historial[len(orden.Historial)-1]
	eventoDerivado, errEvento := NuevoEventoSalidaCobro(orden)
	auditoriaDerivada, errAuditoria := nuevoRegistroAuditoriaCobro(orden)
	if err != nil || vista.OrdenRef != auditoria.OrdenRef || vista.OrdenRef != evento.OrdenRef ||
		errControl != nil || errEvento != nil || errAuditoria != nil || evento != eventoDerivado || auditoria != auditoriaDerivada ||
		evento.VersionOrden != auditoria.VersionPosterior ||
		version != evento.VersionOrden || auditoria.VersionAnterior != version-1 ||
		huella != evento.HuellaOrdenSHA256 || huella != auditoria.HuellaPosteriorSHA256 ||
		orden.ExpedienteRef != auditoria.ExpedienteRef || orden.CorrelacionRef != auditoria.CorrelacionRef ||
		orden.CorrelacionRef != evento.CorrelacionRef || auditoria.Accion != ultimoHecho.AccionAutorizada ||
		auditoria.ActorRef != ultimoHecho.ActorRef || auditoria.PerfilActivoRef != ultimoHecho.PerfilActivoRef ||
		auditoria.DecisionAutorizacionRef != ultimoHecho.AutorizacionRef ||
		auditoria.AtestacionAutenticacionRef != ultimoHecho.AtestacionAutenticacionRef ||
		auditoria.SesionRef != ultimoHecho.SesionRef || auditoria.HuellaSesionHMAC != ultimoHecho.HuellaSesionHMAC ||
		auditoria.MetodoAutenticacion != ultimoHecho.MetodoAutenticacion ||
		auditoria.GarantiaAutenticacion != ultimoHecho.GarantiaAutenticacion ||
		auditoria.Motivo != ultimoHecho.Motivo || auditoria.Resultado != string(ultimoHecho.EstadoPosterior) ||
		!auditoria.OcurridoEn.Equal(ultimoHecho.OcurridoEn) || !evento.OcurridoEn.Equal(ultimoHecho.OcurridoEn) ||
		(auditoria.VersionAnterior == 0 && auditoria.HuellaAnteriorSHA256 != strings.Repeat("0", 64)) ||
		(auditoria.VersionAnterior > 0 && auditoria.HuellaAnteriorSHA256 == strings.Repeat("0", 64)) {
		return ErrMutacionOrdenCobroInvalida
	}
	return nil
}

func tipoEventoSalidaCobroParaHecho(hecho domain.TipoHechoCobro) (TipoEventoSalidaCobro, bool) {
	switch hecho {
	case domain.HechoCobroOrdenCreada:
		return EventoCobroOrdenCreada, true
	case domain.HechoCobroOperacionEnviada:
		return EventoCobroOperacionEnviada, true
	case domain.HechoCobroResultadoPendiente:
		return EventoCobroResultadoPendiente, true
	case domain.HechoCobroResultadoDesconocido:
		return EventoCobroResultadoDesconocido, true
	case domain.HechoCobroConfirmado:
		return EventoCobroConfirmado, true
	case domain.HechoCobroRechazado:
		return EventoCobroRechazado, true
	case domain.HechoCobroCancelado:
		return EventoCobroCancelado, true
	case domain.HechoCobroCaducado:
		return EventoCobroCaducado, true
	case domain.HechoCobroConciliado:
		return EventoCobroConciliado, true
	case domain.HechoCobroDevolucionSolicitada:
		return EventoCobroDevolucionSolicitada, true
	case domain.HechoCobroDevolucionResultadoPendiente:
		return EventoCobroDevolucionResultadoPendiente, true
	case domain.HechoCobroDevolucionResultadoDesconocido:
		return EventoCobroDevolucionResultadoDesconocido, true
	case domain.HechoCobroDevolucionRechazada:
		return EventoCobroDevolucionRechazada, true
	case domain.HechoCobroDevuelto:
		return EventoCobroDevuelto, true
	case domain.HechoCobroDevolucionConciliada:
		return EventoCobroDevolucionConciliada, true
	case domain.HechoCobroIncidenciaDetectada:
		return EventoCobroIncidenciaDetectada, true
	case domain.HechoCobroEvidenciaAdicional:
		return EventoCobroEvidenciaAdicional, true
	default:
		return "", false
	}
}

// RepositorioOrdenesCobro debe aplicar cada confirmacion en una unica
// transaccion: agregado, auditoria inmutable y outbox, o ninguno. Al confirmar
// una creacion obtiene el instante de un reloj transaccional (nunca de la
// solicitud), valida EvidenciaAutorizacion en ese instante y consume su
// DecisionRef de forma atomica con el efecto. El consumo es una relacion unica
// DecisionRef -> (OrdenRef, HuellaEfectoSHA256). Una decision ya consumida solo
// puede resolver idempotentemente ese mismo efecto existente; nunca vuelve a
// escribirlo. HuellaEfectoSHA256 es el control de concurrencia de la orden y
// Mutacion.Validar liga a esa orden la auditoria y el outbox derivados.
// Reutilizar la decision para otra orden o huella deniega. Datos() fija la lista
// positiva exacta que el adaptador debe comparar contra sus registros
// oficiales: decision activa y su huella canonica; asignacion activa, vigente
// y su huella; version activa y control de vigencia del rol (referencia,
// revision, estado y huellas); revision y
// huella del catalogo completo; control de sesion (referencia, revision,
// huella y vigencia); y contexto de actor (referencia, version y huella). Una
// retirada, revocacion, ausencia, ambiguedad, CAS perdido o control desconocido
// deniega sin escrituras. La evidencia no reemplaza estas lecturas ni permite
// confiar en una copia anterior a la transaccion.
//
// En la misma transaccion hace ademas CAS de todos los datos de la reserva y
// comprueba en el registro oficial el control exacto de liquidacion:
// referencia, revision, huella, estado exigible y vigencia. El instante debe
// estar dentro de las vigencias de reserva, decision, sesion y liquidacion. La
// transicion ordinaria usa comparacion simultanea de version y huella.
//
// Un adaptador cuya liquidacion autoritativa vive en una fuente externa y no
// puede ofrecer un bloqueo, fence o CAS que permanezca valido hasta el commit
// NO satisface este puerto. Consultarla antes y confiar despues en la copia no
// constituye atomicidad y el modulo debe permanecer sin cablear.
type RepositorioOrdenesCobro interface {
	ReservarCreacion(context.Context, SolicitudReservaOrdenCobro) (ReservaOrdenCobro, error)
	ConfirmarCreacion(context.Context, SolicitudConfirmarCreacionOrdenCobro) error
	AbandonarReservaCreacion(context.Context, SolicitudAbandonarReservaOrdenCobro) error
	ReservarDevolucion(context.Context, SolicitudReservaDevolucionCobro) (ReservaDevolucionCobro, error)
	ConfirmarDevolucion(context.Context, SolicitudConfirmarReservaDevolucionCobro) error
	AbandonarReservaDevolucion(context.Context, SolicitudAbandonarReservaDevolucionCobro) error
	ObtenerOrden(context.Context, string) (domain.OrdenCobro, error)
	ObtenerOrdenPorOperacion(context.Context, ReferenciaOperacionCobro) (domain.OrdenCobro, error)
	ConfirmarTransicion(context.Context, SolicitudConfirmarTransicionOrdenCobro) error
}

type SelladorSolicitudCobro interface {
	// Cada operacion tiene un metodo distinto: el adaptador no puede elegir un
	// dominio libre ni reutilizar accidentalmente el de otra finalidad.
	SellarIndiceAltaCobro(context.Context, []byte) (string, error)
	SellarHuellaPeticionCobro(context.Context, []byte) (string, error)
	SellarIndiceDevolucionCobro(context.Context, []byte) (string, error)
}

type GeneradorIDOrdenCobro interface {
	NuevoIDOrdenCobro() (string, error)
	NuevoIDDevolucionCobro() (string, error)
}

// CapacidadesPasarelaCobro evita suponer funciones de un proveedor concreto.
// El perfil se versiona y se valida antes de habilitar un flujo de cobro.
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
	if !clavePuertoCobroValida(c.ConectorID) || c.VersionConector < 1 || !c.RedireccionAlojada ||
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

type MetodoHandoffCobro string

const MetodoHandoffCobroPOSTFormulario MetodoHandoffCobro = "post_formulario"

type OrigenPasarelaCobroPublicado struct {
	ID                        string
	Version                   int
	BaseHTTPS                 string
	RutasPermitidas           []string
	CamposHandoffPermitidos   []string
	HuellaConfiguracionSHA256 string
	PublicadaEn               time.Time
}

func (o OrigenPasarelaCobroPublicado) validarSinHuella() error {
	base, err := url.Parse(o.BaseHTTPS)
	if err != nil || !clavePuertoCobroValida(o.ID) || o.Version < 1 || base.Scheme != "https" ||
		base.Host == "" || base.Hostname() == "" || base.User != nil || base.Opaque != "" ||
		base.RawQuery != "" || base.ForceQuery || base.Fragment != "" || base.RawPath != "" ||
		(base.Path != "" && base.Path != "/") || len(o.BaseHTTPS) > 2048 || o.PublicadaEn.IsZero() ||
		!listaCerradaPuertoCobroValida(o.RutasPermitidas, true) ||
		!listaCerradaPuertoCobroValida(o.CamposHandoffPermitidos, false) {
		return ErrInicioOperacionCobroInvalido
	}
	return nil
}

// BytesCanonicosConfiguracionOrigenPasarelaCobro fija el origen, las rutas y
// los campos publicados. Las listas son conjuntos y se ordenan en copias para
// que su orden accidental no cambie la huella.
func BytesCanonicosConfiguracionOrigenPasarelaCobro(o OrigenPasarelaCobroPublicado) ([]byte, error) {
	if o.validarSinHuella() != nil {
		return nil, ErrInicioOperacionCobroInvalido
	}
	rutas := append([]string(nil), o.RutasPermitidas...)
	campos := append([]string(nil), o.CamposHandoffPermitidos...)
	sort.Strings(rutas)
	sort.Strings(campos)
	valor := struct {
		VersionEsquema int
		ID             string
		Version        int
		BaseHTTPS      string
		Rutas          []string
		Campos         []string
		PublicadaEn    string
	}{
		VersionEsquema: 1, ID: o.ID, Version: o.Version, BaseHTTPS: o.BaseHTTPS,
		Rutas: rutas, Campos: campos, PublicadaEn: o.PublicadaEn.UTC().Format(time.RFC3339Nano),
	}
	bytes, err := json.Marshal(valor)
	if err != nil {
		return nil, ErrInicioOperacionCobroInvalido
	}
	return append([]byte(nil), bytes...), nil
}

func CalcularHuellaConfiguracionOrigenPasarelaCobro(o OrigenPasarelaCobroPublicado) (string, error) {
	bytes, err := BytesCanonicosConfiguracionOrigenPasarelaCobro(o)
	if err != nil {
		return "", err
	}
	huella := sha256.Sum256(bytes)
	return fmt.Sprintf("%x", huella), nil
}

func (o OrigenPasarelaCobroPublicado) Validar() error {
	huella, err := CalcularHuellaConfiguracionOrigenPasarelaCobro(o)
	if err != nil || !huellaSHA256PuertoCobroValida(o.HuellaConfiguracionSHA256) ||
		o.HuellaConfiguracionSHA256 != huella {
		return ErrInicioOperacionCobroInvalido
	}
	return nil
}

type CampoHandoffCobro struct {
	Nombre string
	Valor  string
}

type CargaHandoffCobro struct{ campos []CampoHandoffCobro }

func NuevaCargaHandoffCobro(campos []CampoHandoffCobro, permitidos []string) (CargaHandoffCobro, error) {
	if len(campos) == 0 || len(campos) > 32 || !listaCerradaPuertoCobroValida(permitidos, false) {
		return CargaHandoffCobro{}, ErrInicioOperacionCobroInvalido
	}
	permitido := make(map[string]struct{}, len(permitidos))
	for _, campo := range permitidos {
		permitido[campo] = struct{}{}
	}
	vistos := make(map[string]struct{}, len(campos))
	copia := make([]CampoHandoffCobro, len(campos))
	for indice, campo := range campos {
		if !clavePuertoCobroValida(campo.Nombre) || !textoPuertoCobroValido(campo.Valor, 4096) ||
			contieneDatoTarjetaPuertoCobro(campo.Valor) {
			return CargaHandoffCobro{}, ErrInicioOperacionCobroInvalido
		}
		if _, existe := permitido[campo.Nombre]; !existe {
			return CargaHandoffCobro{}, ErrInicioOperacionCobroInvalido
		}
		if _, repetido := vistos[campo.Nombre]; repetido {
			return CargaHandoffCobro{}, ErrInicioOperacionCobroInvalido
		}
		vistos[campo.Nombre] = struct{}{}
		copia[indice] = campo
	}
	return CargaHandoffCobro{campos: copia}, nil
}

func (c CargaHandoffCobro) copiarCampos() ([]CampoHandoffCobro, error) {
	if len(c.campos) == 0 {
		return nil, ErrInicioOperacionCobroInvalido
	}
	return append([]CampoHandoffCobro(nil), c.campos...), nil
}

func (CargaHandoffCobro) MarshalJSON() ([]byte, error) { return nil, ErrInicioOperacionCobroInvalido }
func (CargaHandoffCobro) String() string               { return "[CARGA-HANDOFF-OCULTA]" }
func (c CargaHandoffCobro) GoString() string           { return c.String() }

// InicioOperacionCobro separa el origen publicado de la carga de handoff. No
// acepta una URL devuelta libremente por el proveedor ni secretos en query.
type InicioOperacionCobro struct {
	Evidencia                 domain.EvidenciaInicioOperacionCobro
	Origen                    OrigenPasarelaCobroPublicado
	VersionOrden              int
	HuellaOrdenSHA256         string
	HuellaConfiguracionSHA256 string
	Ruta                      string
	Metodo                    MetodoHandoffCobro
	Carga                     CargaHandoffCobro
	GeneradaEn                time.Time
	ExpiraEn                  time.Time
}

func (i InicioOperacionCobro) Validar() error {
	control, errControl := i.Evidencia.Control()
	if i.Origen.Validar() != nil || i.VersionOrden < 1 ||
		!huellaSHA256PuertoCobroValida(i.HuellaOrdenSHA256) ||
		!huellaSHA256PuertoCobroValida(i.HuellaConfiguracionSHA256) ||
		i.HuellaConfiguracionSHA256 != i.Origen.HuellaConfiguracionSHA256 ||
		i.Metodo != MetodoHandoffCobroPOSTFormulario ||
		i.GeneradaEn.IsZero() || !i.ExpiraEn.After(i.GeneradaEn) ||
		i.ExpiraEn.Sub(i.GeneradaEn) > 15*time.Minute || i.GeneradaEn.Before(i.Origen.PublicadaEn) ||
		errControl != nil || control.ConectorID != i.Origen.ID || control.VersionConector != i.Origen.Version ||
		i.GeneradaEn.Before(control.RecibidaEn) || i.GeneradaEn.Sub(control.RecibidaEn) > 2*time.Minute ||
		len(i.Ruta) > 1024 ||
		!contieneCadenaExacta(i.Origen.RutasPermitidas, i.Ruta) {
		return ErrInicioOperacionCobroInvalido
	}
	if !rutaHandoffPuertoCobroValida(i.Ruta) {
		return ErrInicioOperacionCobroInvalido
	}
	campos, err := i.Carga.copiarCampos()
	if err != nil || !camposHandoffCoinciden(campos, i.Origen.CamposHandoffPermitidos) {
		return ErrInicioOperacionCobroInvalido
	}
	return nil
}

// ValidarContra es la unica validacion suficiente para entregar un handoff.
// Liga la respuesta al comando sellado, al origen exacto publicado y al reloj
// confiable. Validar solo comprueba estructura y no autoriza su entrega.
func (i InicioOperacionCobro) ValidarContra(
	comando domain.ComandoInicioOperacionCobro,
	origen OrigenPasarelaCobroPublicado,
	ahora time.Time,
) error {
	datos, errComando := comando.Datos()
	control, errControl := i.Evidencia.Control()
	if i.Validar() != nil || errComando != nil || errControl != nil || origen.Validar() != nil || ahora.IsZero() ||
		!origenesPasarelaCobroIguales(i.Origen, origen) ||
		i.VersionOrden != datos.VersionOrden || i.HuellaOrdenSHA256 != datos.HuellaOrdenSHA256 ||
		i.HuellaConfiguracionSHA256 != origen.HuellaConfiguracionSHA256 ||
		control.OrdenRef != datos.OrdenRef || control.LiquidacionRef != datos.LiquidacionRef ||
		!control.Importe.Igual(datos.Importe) || control.Concepto != datos.Concepto ||
		ahora.UTC().Before(i.GeneradaEn.UTC()) || !ahora.UTC().Before(i.ExpiraEn.UTC()) ||
		i.ExpiraEn.After(datos.CaducaEn) {
		return ErrInicioOperacionCobroInvalido
	}
	return nil
}

// CamposRespuestaPOSTContra devuelve una copia solo tras la validacion
// completa. No promete consumo unico: esa propiedad requiere una custodia
// durable aun no implementada y el modulo permanece NO EXPUESTO hasta tenerla.
func (i InicioOperacionCobro) CamposRespuestaPOSTContra(
	comando domain.ComandoInicioOperacionCobro,
	origen OrigenPasarelaCobroPublicado,
	ahora time.Time,
) ([]CampoHandoffCobro, error) {
	if i.ValidarContra(comando, origen, ahora) != nil {
		return nil, ErrInicioOperacionCobroInvalido
	}
	return i.Carga.copiarCampos()
}

func origenesPasarelaCobroIguales(a, b OrigenPasarelaCobroPublicado) bool {
	if a.ID != b.ID || a.Version != b.Version || a.BaseHTTPS != b.BaseHTTPS ||
		a.HuellaConfiguracionSHA256 != b.HuellaConfiguracionSHA256 || !a.PublicadaEn.Equal(b.PublicadaEn) ||
		len(a.RutasPermitidas) != len(b.RutasPermitidas) || len(a.CamposHandoffPermitidos) != len(b.CamposHandoffPermitidos) {
		return false
	}
	bytesA, errA := BytesCanonicosConfiguracionOrigenPasarelaCobro(a)
	bytesB, errB := BytesCanonicosConfiguracionOrigenPasarelaCobro(b)
	return errA == nil && errB == nil && string(bytesA) == string(bytesB)
}

type CatalogoOrigenesPasarelaCobro interface {
	ObtenerOrigenPublicado(context.Context, string, int) (OrigenPasarelaCobroPublicado, error)
}

type ReferenciaOperacionCobro struct {
	ConectorID            string `json:"conector_id"`
	VersionConector       int    `json:"version_conector"`
	OrdenRef              string `json:"orden_ref"`
	OperacionProveedorRef string `json:"operacion_proveedor_ref"`
	CorrelacionRef        string `json:"correlacion_ref"`
}

func (r ReferenciaOperacionCobro) Validar() error {
	if !clavePuertoCobroValida(r.ConectorID) || r.VersionConector < 1 ||
		!idOrdenPuertoCobroValido(r.OrdenRef) || !referenciaPuertoCobroValida(r.OperacionProveedorRef) ||
		!referenciaPuertoCobroValida(r.CorrelacionRef) {
		return ErrReferenciaOperacionCobroInvalida
	}
	return nil
}

// NotificacionCobro solo transporta una referencia opaca a una recepcion
// temporal controlada por el adaptador. El nucleo no recibe cuerpos, cookies,
// cabeceras ni datos de tarjeta del proveedor.
type NotificacionCobro struct {
	ConectorID      string    `json:"conector_id"`
	VersionConector int       `json:"version_conector"`
	RecepcionRef    string    `json:"recepcion_ref"`
	Audiencia       string    `json:"audiencia"`
	RecibidaEn      time.Time `json:"recibida_en"`
}

func (n NotificacionCobro) Validar() error {
	if !clavePuertoCobroValida(n.ConectorID) || n.VersionConector < 1 ||
		!referenciaOpacaPuertoCobroValida(n.RecepcionRef, "rec_") || n.Audiencia != audienciaPuertoCobro || n.RecibidaEn.IsZero() {
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
	if !clavePuertoCobroValida(s.ConectorID) || s.VersionConector < 1 ||
		!referenciaOpacaPuertoCobroValida(s.RecepcionRef, "rec_") || s.Audiencia != audienciaPuertoCobro ||
		!tipoContenidoNotificacionCobroPermitido(s.TipoContenido) || s.Tamano < 1 || s.Tamano > 1024*1024 ||
		!huellaSHA256PuertoCobroValida(s.HuellaSHA256) || s.RecibidaEn.IsZero() ||
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

type CustodiaNotificacionesCobro interface {
	Custodiar(context.Context, SolicitudCustodiarNotificacionCobro, io.Reader) (NotificacionCobro, error)
	ConsumirUnaVez(context.Context, NotificacionCobro) (ContenidoNotificacionCobroUnico, error)
	Descartar(context.Context, NotificacionCobro, string) error
}

// VerificadorPasarelaCobro es la unica frontera autorizada para convertir una
// recepcion custodiada o una respuesta remota en evidencia verificada. Sus
// implementaciones verifican criptografia, audiencia, vigencia y replay antes
// de usar las fabricas de dominio *Verificada.
type VerificadorPasarelaCobro interface {
	VerificarNotificacionCobro(context.Context, NotificacionCobro) (ResultadoOperacionCobro, error)
	VerificarNotificacionDevolucion(context.Context, NotificacionCobro, ReferenciaDevolucionCobro) (ResultadoDevolucionCobro, error)
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
	if !clavePuertoCobroValida(r.ConectorID) || r.VersionConector < 1 ||
		!idOrdenPuertoCobroValido(r.OrdenRef) || !idDevolucionPuertoCobroValido(r.DevolucionRef) ||
		!referenciaPuertoCobroValida(r.OperacionProveedorRef) || !referenciaPuertoCobroValida(r.CorrelacionRef) {
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

// PasarelaCobro es el unico contrato remoto del nucleo de cobros. No conoce
// proveedores, protocolos ni redes concretas y no sirve para pagos salientes.
type PasarelaCobro interface {
	VerificadorPasarelaCobro
	Capacidades(context.Context) (CapacidadesPasarelaCobro, error)
	CrearOperacion(context.Context, SolicitudOperacionCobro) (InicioOperacionCobro, error)
	ConsultarOperacion(context.Context, ReferenciaOperacionCobro) (ResultadoOperacionCobro, error)
	SolicitarDevolucion(context.Context, SolicitudDevolucionCobro) (ResultadoDevolucionCobro, error)
	ConsultarDevolucion(context.Context, ReferenciaDevolucionCobro) (ResultadoDevolucionCobro, error)
	Conciliar(context.Context, SolicitudConciliacionCobro) (ResultadoConciliacionCobro, error)
}

func clavePuertoCobroValida(valor string) bool {
	if valor == "" || valor != strings.TrimSpace(valor) || len(valor) > 128 {
		return false
	}
	for indice, caracter := range valor {
		if (caracter >= 'a' && caracter <= 'z') || (indice > 0 && caracter >= '0' && caracter <= '9') ||
			(indice > 0 && (caracter == '.' || caracter == '_' || caracter == '-')) {
			continue
		}
		return false
	}
	return true
}

func idOrdenPuertoCobroValido(valor string) bool {
	return referenciaOpacaPuertoCobroValida(valor, "cob_")
}

func idDevolucionPuertoCobroValido(valor string) bool {
	return referenciaOpacaPuertoCobroValida(valor, "dev_")
}

func referenciaOpacaPuertoCobroValida(valor, prefijo string) bool {
	if !strings.HasPrefix(valor, prefijo) {
		return false
	}
	parte := strings.TrimPrefix(valor, prefijo)
	if len(parte) < 22 || len(parte) > 128 {
		return false
	}
	for _, caracter := range parte {
		if (caracter >= 'a' && caracter <= 'z') || (caracter >= 'A' && caracter <= 'Z') ||
			(caracter >= '0' && caracter <= '9') || caracter == '_' || caracter == '-' {
			continue
		}
		return false
	}
	return true
}

func huellaHMACPuertoCobroDeDominioValida(valor, dominio string) bool {
	partes := strings.Split(valor, ":")
	return len(partes) == 3 && partes[0] == "hmac-sha256" && partes[1] == dominio &&
		huellaSHA256PuertoCobroValida(partes[2])
}

func idPersonaPuertoCobroValido(valor string) bool {
	return referenciaOpacaPuertoCobroValida(valor, "per_")
}

func idPerfilPuertoCobroValido(valor string) bool {
	return referenciaOpacaPuertoCobroValida(valor, "prf_")
}

func huellaSesionPuertoCobroValida(valor string) bool {
	partes := strings.Split(valor, ":")
	return len(partes) == 3 && partes[0] == "hmac-sha256" && partes[1] == "sesion-v1" &&
		huellaSHA256PuertoCobroValida(partes[2])
}

func metodoAutenticacionPuertoCobroPermitido(metodo domain.AuthMethod) bool {
	switch metodo {
	case domain.AuthMethodCertificate, domain.AuthMethodDNIe, domain.AuthMethodSSO,
		domain.AuthMethodClave, domain.AuthMethodKerberos:
		return true
	default:
		return false
	}
}

func garantiaAutenticacionPuertoCobroPermitida(garantia domain.AuthAssurance) bool {
	switch garantia {
	case domain.AuthAssuranceLow, domain.AuthAssuranceSubstantial, domain.AuthAssuranceHigh:
		return true
	default:
		return false
	}
}

func accionAuditoriaPuertoCobroPermitida(accion domain.AccionCobro) bool {
	switch accion {
	case domain.AccionCobroCrearOrden, domain.AccionCobroIniciarOperacion, domain.AccionCobroProcesarResultado,
		domain.AccionCobroSolicitarDevolucion, domain.AccionCobroProcesarDevolucion, domain.AccionCobroConciliar,
		domain.AccionCobroCancelar, domain.AccionCobroCaducar:
		return true
	default:
		return false
	}
}

func huellaSHA256PuertoCobroValida(valor string) bool {
	if len(valor) != 64 {
		return false
	}
	for _, caracter := range valor {
		if (caracter < '0' || caracter > '9') && (caracter < 'a' || caracter > 'f') {
			return false
		}
	}
	return true
}

func referenciaPuertoCobroValida(valor string) bool {
	return textoPuertoCobroValido(valor, 512)
}

func textoPuertoCobroValido(valor string, maximo int) bool {
	if valor == "" || valor != strings.TrimSpace(valor) || len(valor) > maximo {
		return false
	}
	for _, caracter := range valor {
		if caracter < 0x20 || caracter == 0x7f {
			return false
		}
	}
	return !contieneDatoTarjetaPuertoCobro(valor)
}

func contieneDatoTarjetaPuertoCobro(valor string) bool {
	minusculas := strings.Map(func(caracter rune) rune {
		if unicode.Is(unicode.Cf, caracter) {
			return -1
		}
		return unicode.ToLower(caracter)
	}, valor)
	reemplazador := strings.NewReplacer("_", " ", "-", " ", ":", " ", "=", " ", ".", " ", "/", " ")
	for _, palabra := range strings.Fields(reemplazador.Replace(minusculas)) {
		switch palabra {
		case "pan", "cvv", "cvc", "cvn", "pin", "criptograma", "cryptogram", "tarjeta", "card", "cardnumber":
			return true
		}
	}
	digitos := make([]byte, 0, 32)
	comprobar := func() bool {
		for longitud := 13; longitud <= 19 && longitud <= len(digitos); longitud++ {
			for inicio := 0; inicio+longitud <= len(digitos); inicio++ {
				if numeroTarjetaPuertoCobroValido(digitos[inicio : inicio+longitud]) {
					return true
				}
			}
		}
		return false
	}
	for _, caracter := range valor {
		if numero, esDigito := valorDigitoDecimalPuertoCobro(caracter); esDigito {
			digitos = append(digitos, byte('0'+numero))
			continue
		}
		if (unicode.IsSpace(caracter) || unicode.Is(unicode.Dash, caracter) ||
			unicode.Is(unicode.Cf, caracter) || caracter == '.') && len(digitos) > 0 {
			continue
		}
		if comprobar() {
			return true
		}
		digitos = digitos[:0]
	}
	if comprobar() {
		return true
	}
	return false
}

func valorDigitoDecimalPuertoCobro(caracter rune) (byte, bool) {
	switch {
	case caracter >= '0' && caracter <= '9':
		return byte(caracter - '0'), true
	case caracter >= '\u0660' && caracter <= '\u0669':
		return byte(caracter - '\u0660'), true
	case caracter >= '\u06f0' && caracter <= '\u06f9':
		return byte(caracter - '\u06f0'), true
	case caracter >= '\uff10' && caracter <= '\uff19':
		return byte(caracter - '\uff10'), true
	default:
		return 0, false
	}
}

func numeroTarjetaPuertoCobroValido(digitos []byte) bool {
	suma := 0
	par := len(digitos)%2 == 0
	for indice, caracter := range digitos {
		numero := int(caracter - '0')
		if (indice%2 == 0) == par {
			numero *= 2
			if numero > 9 {
				numero -= 9
			}
		}
		suma += numero
	}
	return suma > 0 && suma%10 == 0
}

func listaCerradaPuertoCobroValida(valores []string, rutas bool) bool {
	if len(valores) == 0 || len(valores) > 64 {
		return false
	}
	vistos := make(map[string]struct{}, len(valores))
	for _, valor := range valores {
		valido := clavePuertoCobroValida(valor)
		if rutas {
			valido = rutaHandoffPuertoCobroValida(valor)
		}
		if !valido {
			return false
		}
		if _, repetido := vistos[valor]; repetido {
			return false
		}
		vistos[valor] = struct{}{}
	}
	return true
}

func rutaHandoffPuertoCobroValida(valor string) bool {
	if valor == "" || valor != strings.TrimSpace(valor) || len(valor) > 1024 ||
		!strings.HasPrefix(valor, "/") || strings.HasPrefix(valor, "//") || strings.Contains(valor, "\\") ||
		contieneDatoTarjetaPuertoCobro(valor) {
		return false
	}
	ruta, err := url.Parse(valor)
	if err != nil || ruta.IsAbs() || ruta.Opaque != "" || ruta.Host != "" || ruta.User != nil ||
		ruta.RawQuery != "" || ruta.ForceQuery || ruta.Fragment != "" || ruta.RawPath != "" || ruta.Path != valor {
		return false
	}
	segmentos := strings.Split(strings.TrimPrefix(valor, "/"), "/")
	for _, segmento := range segmentos {
		if segmento == "" || segmento == "." || segmento == ".." {
			return false
		}
		for _, caracter := range segmento {
			if (caracter >= 'a' && caracter <= 'z') || (caracter >= 'A' && caracter <= 'Z') ||
				(caracter >= '0' && caracter <= '9') || caracter == '-' || caracter == '_' || caracter == '.' || caracter == '~' {
				continue
			}
			return false
		}
	}
	return true
}

func contieneCadenaExacta(valores []string, buscado string) bool {
	for _, valor := range valores {
		if valor == buscado {
			return true
		}
	}
	return false
}

func camposHandoffCoinciden(campos []CampoHandoffCobro, permitidos []string) bool {
	for _, campo := range campos {
		if !contieneCadenaExacta(permitidos, campo.Nombre) {
			return false
		}
	}
	return len(campos) > 0
}

func tipoContenidoNotificacionCobroPermitido(valor string) bool {
	switch valor {
	case "application/json", "application/jose", "application/jose+json":
		return true
	default:
		return false
	}
}

var (
	_ json.Marshaler = ReservaOrdenCobro{}
	_ json.Marshaler = ReservaDevolucionCobro{}
	_ json.Marshaler = ResultadoOperacionCobro{}
	_ json.Marshaler = ResultadoDevolucionCobro{}
	_ json.Marshaler = ResultadoConciliacionCobro{}
)
