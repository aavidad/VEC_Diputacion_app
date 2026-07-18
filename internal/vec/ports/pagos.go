package ports

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	pagoscanonicos "vec-diputacion-granada/internal/vec/canonico/pagos"
	"vec-diputacion-granada/internal/vec/domain"
)

var (
	ErrPasarelaCobroNoDisponible          = errors.New("vec: pasarela de cobro no disponible")
	ErrCapacidadPasarelaCobroNoDisponible = pagoscanonicos.ErrCapacidadPasarelaCobroNoDisponible
	ErrSolicitudOperacionCobroInvalida    = pagoscanonicos.ErrSolicitudOperacionCobroInvalida
	ErrInicioOperacionCobroInvalido       = pagoscanonicos.ErrInicioOperacionCobroInvalido
	ErrReferenciaOperacionCobroInvalida   = pagoscanonicos.ErrReferenciaOperacionCobroInvalida
	ErrNotificacionCobroInvalida          = pagoscanonicos.ErrNotificacionCobroInvalida
	ErrSolicitudDevolucionCobroInvalida   = pagoscanonicos.ErrSolicitudDevolucionCobroInvalida
	ErrSolicitudConciliacionCobroInvalida = pagoscanonicos.ErrSolicitudConciliacionCobroInvalida
	ErrResultadoPasarelaCobroInvalido     = pagoscanonicos.ErrResultadoPasarelaCobroInvalido
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
	ErrMutacionOrdenCobroInvalida         = pagoscanonicos.ErrMutacionOrdenCobroInvalida
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

type RegistroAuditoriaCobro = pagoscanonicos.RegistroAuditoriaCobro

type CanalAuditoriaCobro = pagoscanonicos.CanalAuditoriaCobro

const (
	CanalAuditoriaCobroInterno           = pagoscanonicos.CanalAuditoriaCobroInterno
	CanalAuditoriaCobroPasarela          = pagoscanonicos.CanalAuditoriaCobroPasarela
	CanalAuditoriaCobroProcesoAutomatico = pagoscanonicos.CanalAuditoriaCobroProcesoAutomatico
)

type MetadatosAuditoriaCobro = pagoscanonicos.MetadatosAuditoriaCobro

func matrizEvidenciaAuditoriaCobroValida(r RegistroAuditoriaCobro) bool {
	return pagoscanonicos.MatrizEvidenciaAuditoriaValida(r)
}

type TipoEventoSalidaCobro = pagoscanonicos.TipoEventoSalidaCobro

const (
	EventoCobroOrdenCreada                    = pagoscanonicos.EventoCobroOrdenCreada
	EventoCobroOperacionEnviada               = pagoscanonicos.EventoCobroOperacionEnviada
	EventoCobroResultadoPendiente             = pagoscanonicos.EventoCobroResultadoPendiente
	EventoCobroResultadoDesconocido           = pagoscanonicos.EventoCobroResultadoDesconocido
	EventoCobroConfirmado                     = pagoscanonicos.EventoCobroConfirmado
	EventoCobroRechazado                      = pagoscanonicos.EventoCobroRechazado
	EventoCobroCancelado                      = pagoscanonicos.EventoCobroCancelado
	EventoCobroCaducado                       = pagoscanonicos.EventoCobroCaducado
	EventoCobroConciliado                     = pagoscanonicos.EventoCobroConciliado
	EventoCobroDevolucionSolicitada           = pagoscanonicos.EventoCobroDevolucionSolicitada
	EventoCobroDevolucionResultadoPendiente   = pagoscanonicos.EventoCobroDevolucionResultadoPendiente
	EventoCobroDevolucionResultadoDesconocido = pagoscanonicos.EventoCobroDevolucionResultadoDesconocido
	EventoCobroDevolucionRechazada            = pagoscanonicos.EventoCobroDevolucionRechazada
	EventoCobroDevuelto                       = pagoscanonicos.EventoCobroDevuelto
	EventoCobroDevolucionConciliada           = pagoscanonicos.EventoCobroDevolucionConciliada
	EventoCobroIncidenciaDetectada            = pagoscanonicos.EventoCobroIncidenciaDetectada
	EventoCobroEvidenciaAdicional             = pagoscanonicos.EventoCobroEvidenciaAdicional
)

type AtributosEventoSalidaCobro = pagoscanonicos.AtributosEventoSalidaCobro

type EventoSalidaCobro = pagoscanonicos.EventoSalidaCobro

// NuevoEventoSalidaCobro deriva el mensaje completo, incluido un identificador
// determinista ligado a orden, version, secuencia y huella del ultimo hecho.
// No recibe ningun campo semantico del llamador.
func NuevoEventoSalidaCobro(orden domain.OrdenCobro) (EventoSalidaCobro, error) {
	return pagoscanonicos.NuevoEventoSalidaCobro(orden)
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
	return pagoscanonicos.IDEvento(ordenRef, version, secuencia, huellaHecho, hecho, estado, accion)
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
	canal, existe := pagoscanonicos.CanalAuditoriaParaHecho(hecho.Tipo, hecho.AccionAutorizada)
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
	registro.ID = pagoscanonicos.IDAuditoria(registro.OrdenRef, registro.VersionPosterior,
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
	return pagoscanonicos.TipoEventoParaHecho(hecho)
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

type CapacidadesPasarelaCobro = pagoscanonicos.CapacidadesPasarelaCobro

type SolicitudOperacionCobro = pagoscanonicos.SolicitudOperacionCobro

type MetodoHandoffCobro = pagoscanonicos.MetodoHandoffCobro

const MetodoHandoffCobroPOSTFormulario = pagoscanonicos.MetodoHandoffCobroPOSTFormulario

type OrigenPasarelaCobroPublicado = pagoscanonicos.OrigenPasarelaCobroPublicado

// BytesCanonicosConfiguracionOrigenPasarelaCobro fija el origen, las rutas y
// los campos publicados. Las listas son conjuntos y se ordenan en copias para
// que su orden accidental no cambie la huella.
func BytesCanonicosConfiguracionOrigenPasarelaCobro(o OrigenPasarelaCobroPublicado) ([]byte, error) {
	bytes, err := pagoscanonicos.BytesConfiguracionOrigen(o)
	if err != nil {
		return nil, ErrInicioOperacionCobroInvalido
	}
	return append([]byte(nil), bytes...), nil
}

func CalcularHuellaConfiguracionOrigenPasarelaCobro(o OrigenPasarelaCobroPublicado) (string, error) {
	huella, err := pagoscanonicos.HuellaConfiguracionOrigen(o)
	if err != nil {
		return "", ErrInicioOperacionCobroInvalido
	}
	return huella, nil
}

type CampoHandoffCobro = pagoscanonicos.CampoHandoffCobro

type CargaHandoffCobro = pagoscanonicos.CargaHandoffCobro

func NuevaCargaHandoffCobro(campos []CampoHandoffCobro, permitidos []string) (CargaHandoffCobro, error) {
	return pagoscanonicos.NuevaCargaHandoffCobro(campos, permitidos)
}

type InicioOperacionCobro = pagoscanonicos.InicioOperacionCobro

type CatalogoOrigenesPasarelaCobro interface {
	ObtenerOrigenPublicado(context.Context, string, int) (OrigenPasarelaCobroPublicado, error)
}

type ReferenciaOperacionCobro = pagoscanonicos.ReferenciaOperacionCobro

type NotificacionCobro = pagoscanonicos.NotificacionCobro

type SolicitudCustodiarNotificacionCobro = pagoscanonicos.SolicitudCustodiarNotificacionCobro

type ContenidoNotificacionCobroUnico = pagoscanonicos.ContenidoNotificacionCobroUnico

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

type ResultadoOperacionCobro = pagoscanonicos.ResultadoOperacionCobro

type SolicitudDevolucionCobro = pagoscanonicos.SolicitudDevolucionCobro

type ResultadoDevolucionCobro = pagoscanonicos.ResultadoDevolucionCobro

type ReferenciaDevolucionCobro = pagoscanonicos.ReferenciaDevolucionCobro

type SolicitudConciliacionCobro = pagoscanonicos.SolicitudConciliacionCobro

type ResultadoConciliacionCobro = pagoscanonicos.ResultadoConciliacionCobro

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

func idOrdenPuertoCobroValido(valor string) bool {
	return referenciaOpacaPuertoCobroValida(valor, "cob_")
}

func idDevolucionPuertoCobroValido(valor string) bool {
	return referenciaOpacaPuertoCobroValida(valor, "dev_")
}

func referenciaOpacaPuertoCobroValida(valor, prefijo string) bool {
	return pagoscanonicos.ReferenciaOpacaValida(valor, prefijo)
}

func huellaHMACPuertoCobroDeDominioValida(valor, dominio string) bool {
	return pagoscanonicos.HuellaHMACDeDominioValida(valor, dominio)
}

func idPersonaPuertoCobroValido(valor string) bool {
	return referenciaOpacaPuertoCobroValida(valor, "per_")
}

func huellaSesionPuertoCobroValida(valor string) bool {
	return pagoscanonicos.HuellaHMACDeDominioValida(valor, "sesion-v1")
}

func huellaSHA256PuertoCobroValida(valor string) bool {
	return pagoscanonicos.HuellaSHA256Valida(valor)
}

func referenciaPuertoCobroValida(valor string) bool {
	return textoPuertoCobroValido(valor, 512)
}

func textoPuertoCobroValido(valor string, maximo int) bool {
	return pagoscanonicos.TextoValido(valor, maximo)
}

var (
	_ json.Marshaler = ReservaOrdenCobro{}
	_ json.Marshaler = ReservaDevolucionCobro{}
	_ json.Marshaler = ResultadoOperacionCobro{}
	_ json.Marshaler = ResultadoDevolucionCobro{}
	_ json.Marshaler = ResultadoConciliacionCobro{}
)
