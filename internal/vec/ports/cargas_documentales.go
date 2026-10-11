package ports

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	canonicodoc "vec-diputacion-granada/internal/vec/canonico/documental"
	"vec-diputacion-granada/internal/vec/domain"
)

var (
	ErrRepositorioCargasNoDisponible        = errors.New("vec: repositorio de cargas documentales no disponible")
	ErrReservaCargaDocumentalInvalida       = errors.New("vec: reserva de carga documental invalida")
	ErrReservaCargaDocumentalOcupada        = errors.New("vec: reserva de carga documental ocupada")
	ErrCargaDocumentalNoEncontrada          = errors.New("vec: carga documental no encontrada")
	ErrConflictoVersionCargaDocumental      = errors.New("vec: conflicto de version de carga documental")
	ErrConfirmacionCargaDocumentalInvalida  = errors.New("vec: confirmacion de carga documental invalida")
	ErrManifiestoPreparacionNoEncontrado    = errors.New("vec: manifiesto de preparacion de carga directa no encontrado")
	ErrConfirmacionCargaDocumentalPendiente = errors.New("vec: confirmacion de carga documental pendiente de reconciliacion")
	ErrDecisionPreparacionCargaNoDisponible = errors.New("vec: decision de preparacion de carga no disponible")
	ErrDecisionPreparacionCargaYaConsumida  = errors.New("vec: decision de preparacion de carga ya consumida")
	ErrRecursoBaseCargaDocumentalInvalido   = errors.New("vec: recurso base de carga documental invalido")
	ErrSerializacionTokenReservaProhibida   = errors.New("vec: serializacion de token de reserva prohibida")
	ErrPreparacionCargaDocumentalPendiente  = errors.New("vec: preparacion documental pendiente de conciliacion")
)

const duracionMaximaReservaRepositorioCarga = 10 * time.Minute

const EsquemaHuellaRecursoBaseCargaDocumentalV1 = "vec.carga-documental.recurso-base.v1"

const dominioHuellaTokenReservaCargaDocumental = "vec:token-reserva-carga-documental:v1"

const (
	accionCargaDocumentalPreparar  = "vec.documentos.carga.preparar"
	accionCargaDocumentalConfirmar = "vec.documentos.carga.confirmar"
	accionCargaDocumentalAnalizar  = "vec.documentos.carga.analizar"
	accionCargaDocumentalPromover  = "vec.documentos.carga.promover"

	eventoCargaDocumentalPreparada = "vec.documentos.carga.preparada"
	eventoCargaDocumentalRecibida  = "vec.documentos.carga.recibida"
	eventoCargaDocumentalAnalizada = "vec.documentos.carga.analizada"
	eventoCargaDocumentalPromovida = "vec.documentos.carga.promovida"
)

// TokenReservaCargaDocumental es una capacidad efimera y nominal entre el caso
// de uso y el repositorio. Su material CSPRNG vive exclusivamente en un cierre
// privado e inmutable ligado al dominio de carga documental. Nunca forma parte
// del agregado, la auditoria, el outbox, una respuesta HTTP o un mensaje de
// error. Los repositorios persisten solo HuellaSHA256 y verifican mediante
// CoincideConHuellaSHA256.
type TokenReservaCargaDocumental struct {
	operar operacionCapacidadReserva
}

func NuevoTokenReservaCargaDocumental() (TokenReservaCargaDocumental, error) {
	operar, err := nuevaOperacionCapacidadReserva(dominioHuellaTokenReservaCargaDocumental)
	if err != nil {
		return TokenReservaCargaDocumental{}, ErrReservaCargaDocumentalInvalida
	}
	return TokenReservaCargaDocumental{operar: operar}, nil
}

func (t TokenReservaCargaDocumental) Valido() bool {
	return operacionCapacidadReservaValida(t.operar)
}

func (t TokenReservaCargaDocumental) HuellaSHA256() (string, error) {
	huella, valida := huellaCapacidadReserva(t.operar)
	if !valida {
		return "", ErrReservaCargaDocumentalInvalida
	}
	return huella, nil
}

func (t TokenReservaCargaDocumental) CoincideConHuellaSHA256(huella string) bool {
	return coincideHuellaCapacidadReserva(t.operar, huella)
}

func (TokenReservaCargaDocumental) String() string     { return "[TOKEN-RESERVA-CARGA-CONFIDENCIAL]" }
func (t TokenReservaCargaDocumental) GoString() string { return t.String() }

func (t TokenReservaCargaDocumental) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, t.String())
}

func (t TokenReservaCargaDocumental) LogValue() slog.Value {
	return slog.StringValue(t.String())
}

func (TokenReservaCargaDocumental) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionTokenReservaProhibida
}

func (*TokenReservaCargaDocumental) UnmarshalJSON([]byte) error {
	return ErrSerializacionTokenReservaProhibida
}

func (TokenReservaCargaDocumental) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionTokenReservaProhibida
}

func (*TokenReservaCargaDocumental) UnmarshalText([]byte) error {
	return ErrSerializacionTokenReservaProhibida
}

func (TokenReservaCargaDocumental) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionTokenReservaProhibida
}

func (*TokenReservaCargaDocumental) UnmarshalBinary([]byte) error {
	return ErrSerializacionTokenReservaProhibida
}

func (TokenReservaCargaDocumental) MarshalXML(*xml.Encoder, xml.StartElement) error {
	return ErrSerializacionTokenReservaProhibida
}

func (*TokenReservaCargaDocumental) UnmarshalXML(*xml.Decoder, xml.StartElement) error {
	return ErrSerializacionTokenReservaProhibida
}

type SolicitudReservarCargaDocumental struct {
	Token                  TokenReservaCargaDocumental
	IndiceIdempotenciaHMAC string
	HuellaSolicitudHMAC    string
	Carga                  domain.CargaDocumental
	DecisionPreparacion    ConsumoDecisionPreparacionCargaDocumentalV1
	SolicitadaEn           time.Time
	ReservaExpiraEn        time.Time
	ReservaRecursoV3       ReservaRecursoCargaDocumentalV3
	ActorRegistradoV2      domain.ResultadoContextoActorRegistradoV2
	AutorizacionV3         AutorizacionCargaDocumentalV3
}

func (s SolicitudReservarCargaDocumental) Validar() error {
	if !hmacSHA256PuertoValido(s.IndiceIdempotenciaHMAC) || !hmacSHA256PuertoValido(s.HuellaSolicitudHMAC) ||
		s.Carga.Validar() != nil || s.Carga.Estado != domain.EstadoCargaDocumentalReservada ||
		s.DecisionPreparacion.Validar() != nil ||
		s.Carga.IndiceIdempotenciaHMAC != s.IndiceIdempotenciaHMAC ||
		s.Carga.HuellaSolicitudHMAC != s.HuellaSolicitudHMAC || s.SolicitadaEn.IsZero() ||
		s.ReservaExpiraEn.IsZero() || !s.ReservaExpiraEn.After(s.SolicitadaEn) ||
		s.ReservaExpiraEn.Sub(s.SolicitadaEn) > duracionMaximaReservaRepositorioCarga ||
		s.ReservaExpiraEn.After(s.Carga.ExpiraEn) {
		return ErrReservaCargaDocumentalInvalida
	}
	return nil
}

func (s SolicitudReservarCargaDocumental) ValidarParaPersistenciaV3(ahora time.Time) error {
	if !s.Token.Valido() {
		return ErrReservaCargaDocumentalInvalida
	}
	preimagen, err := PreimagenReservaCargaDocumentalV3(s, s.ReservaRecursoV3)
	if err != nil || s.ActorRegistradoV2.Validar() != nil ||
		s.AutorizacionV3.ValidarPara(AccionCargaDocumentalPrepararReservaV3, preimagen,
			s.ActorRegistradoV2, s.Carga, ahora) != nil {
		return ErrReservaCargaDocumentalInvalida
	}
	return nil
}

// ReservaCargaDocumental devuelve exactamente uno de dos resultados: token
// nuevo, o agregado previamente confirmado. Una reserva en curso ajena se
// comunica como error y nunca se roba silenciosamente.
type ReservaCargaDocumental struct {
	Token           TokenReservaCargaDocumental
	Repetida        bool
	Pendiente       bool
	EstadoOperativo EstadoOperativoCargaDocumental
	Carga           domain.CargaDocumental
}

func (r ReservaCargaDocumental) Validar() error {
	if r.Carga.Validar() != nil {
		return ErrReservaCargaDocumentalInvalida
	}
	if r.Repetida {
		if r.Token.Valido() || (r.Carga.Estado == domain.EstadoCargaDocumentalReservada && !r.Pendiente) ||
			r.Pendiente && !r.EstadoOperativo.Valido() {
			return ErrReservaCargaDocumentalInvalida
		}
		return nil
	}
	if !r.Token.Valido() || r.Carga.Estado != domain.EstadoCargaDocumentalReservada {
		return ErrReservaCargaDocumentalInvalida
	}
	return nil
}

// EstadoOperativoCargaDocumental registra el avance del protocolo y no cambia
// el estado del contenido. Los estados inciertos se cierran por conciliacion
// nominal; nunca autorizan otro intento automatico sobre S3 o recibos.
type EstadoOperativoCargaDocumental string

const (
	EstadoCargaOperacionReservada                EstadoOperativoCargaDocumental = "reservada"
	EstadoCargaOperacionPreparacionIniciada      EstadoOperativoCargaDocumental = "preparacion_iniciada"
	EstadoCargaOperacionPreparada                EstadoOperativoCargaDocumental = "preparada"
	EstadoCargaOperacionEmisionReciboIniciada    EstadoOperativoCargaDocumental = "emision_recibo_iniciada"
	EstadoCargaOperacionReciboRegistrado         EstadoOperativoCargaDocumental = "recibo_registrado"
	EstadoCargaOperacionPreparacionIndeterminada EstadoOperativoCargaDocumental = "preparacion_indeterminada"
	EstadoCargaOperacionReciboIndeterminado      EstadoOperativoCargaDocumental = "recibo_indeterminado"
	EstadoCargaOperacionNoRecuperable            EstadoOperativoCargaDocumental = "no_recuperable"
)

func (e EstadoOperativoCargaDocumental) Valido() bool {
	switch e {
	case EstadoCargaOperacionReservada, EstadoCargaOperacionPreparacionIniciada,
		EstadoCargaOperacionPreparada, EstadoCargaOperacionEmisionReciboIniciada,
		EstadoCargaOperacionReciboRegistrado, EstadoCargaOperacionPreparacionIndeterminada,
		EstadoCargaOperacionReciboIndeterminado, EstadoCargaOperacionNoRecuperable:
		return true
	default:
		return false
	}
}

type ConfirmacionTransicionCargaDocumental struct {
	VersionEsperada      int
	HuellaAnteriorSHA256 string
	Carga                domain.CargaDocumental
	Auditoria            domain.AuditEntry
	Evento               domain.Event
	ActorRegistradoV2    domain.ResultadoContextoActorRegistradoV2
	AutorizacionV3       AutorizacionCargaDocumentalV3
}

func (c ConfirmacionTransicionCargaDocumental) ValidarParaPersistenciaV3(ahora time.Time) error {
	preimagen, err := PreimagenTransicionCargaDocumentalV3(c)
	if err != nil || c.ActorRegistradoV2.Validar() != nil ||
		c.AutorizacionV3.ValidarPara(c.Auditoria.Action, preimagen,
			c.ActorRegistradoV2, c.Carga, ahora) != nil {
		return ErrConfirmacionCargaDocumentalInvalida
	}
	return nil
}

// InstantaneaConfirmacionTransicionCargaDocumental corta todos los alias
// mutables antes de consultar el estado autoritativo. El repositorio debe
// obtener esta copia antes del bloqueo/transaccion, validarla contra la version
// leida dentro de ese bloqueo y persistir exactamente la misma copia.
func InstantaneaConfirmacionTransicionCargaDocumental(
	confirmacion ConfirmacionTransicionCargaDocumental,
) ConfirmacionTransicionCargaDocumental {
	return clonarConfirmacionCargaDocumental(confirmacion)
}

// SolicitudConfirmarPreparacionCargaDocumental obliga al repositorio a
// consumir la reserva y conservar el agregado, la auditoria, el outbox y el
// manifiesto historico en una unica transaccion. Persistir cualquiera de esas
// piezas por separado no satisface el contrato.
type SolicitudConfirmarPreparacionCargaDocumental struct {
	Token             TokenReservaCargaDocumental
	Confirmacion      ConfirmacionTransicionCargaDocumental
	Manifiesto        domain.ManifiestoPreparacionCargaDirectaV1
	ActorRegistradoV2 domain.ResultadoContextoActorRegistradoV2
	AutorizacionV3    AutorizacionCargaDocumentalV3
}

func (s SolicitudConfirmarPreparacionCargaDocumental) ValidarParaPersistenciaV3(ahora time.Time) error {
	preimagen, err := PreimagenConfirmacionPreparacionCargaDocumentalV3(s)
	if err != nil || s.ActorRegistradoV2.Validar() != nil ||
		s.AutorizacionV3.ValidarPara(AccionCargaDocumentalPrepararConfirmacionV3,
			preimagen, s.ActorRegistradoV2, s.Confirmacion.Carga, ahora) != nil {
		return ErrConfirmacionCargaDocumentalInvalida
	}
	return nil
}

func (s SolicitudConfirmarPreparacionCargaDocumental) Validar() error {
	_, err := InstantaneaSolicitudConfirmarPreparacionCargaDocumental(s)
	return err
}

// InstantaneaSolicitudConfirmarPreparacionCargaDocumental copia una sola vez
// toda la entrada mutable antes de validarla. Los adaptadores deben invocarla
// antes de adquirir su bloqueo o transaccion y persistir exclusivamente la
// copia devuelta; volver a leer la solicitud original reabriria un TOCTOU.
func InstantaneaSolicitudConfirmarPreparacionCargaDocumental(
	solicitud SolicitudConfirmarPreparacionCargaDocumental,
) (SolicitudConfirmarPreparacionCargaDocumental, error) {
	datosManifiesto, err := solicitud.Manifiesto.Datos()
	if err != nil {
		return SolicitudConfirmarPreparacionCargaDocumental{}, ErrConfirmacionCargaDocumentalInvalida
	}
	manifiesto, err := domain.RestaurarManifiestoPreparacionCargaDirectaV1(datosManifiesto)
	if err != nil {
		return SolicitudConfirmarPreparacionCargaDocumental{}, ErrConfirmacionCargaDocumentalInvalida
	}
	copia := SolicitudConfirmarPreparacionCargaDocumental{
		Token:        solicitud.Token,
		Confirmacion: clonarConfirmacionCargaDocumental(solicitud.Confirmacion),
		Manifiesto:   manifiesto,
	}
	material := clonarConfirmacionCargaDocumental(ConfirmacionTransicionCargaDocumental{
		ActorRegistradoV2: solicitud.ActorRegistradoV2,
		AutorizacionV3:    solicitud.AutorizacionV3,
	})
	copia.ActorRegistradoV2 = material.ActorRegistradoV2
	copia.AutorizacionV3 = material.AutorizacionV3
	if validarSolicitudConfirmarPreparacionCargaDocumental(copia) != nil {
		return SolicitudConfirmarPreparacionCargaDocumental{}, ErrConfirmacionCargaDocumentalInvalida
	}
	return copia, nil
}

func validarSolicitudConfirmarPreparacionCargaDocumental(
	solicitud SolicitudConfirmarPreparacionCargaDocumental,
) error {
	if !solicitud.Token.Valido() || solicitud.Confirmacion.Carga.Validar() != nil ||
		solicitud.Confirmacion.Carga.Estado != domain.EstadoCargaDocumentalPreparada ||
		solicitud.Manifiesto.ValidarContraCarga(solicitud.Confirmacion.Carga) != nil {
		return ErrConfirmacionCargaDocumentalInvalida
	}
	return nil
}

// ConsumoDecisionPreparacionCargaDocumentalV1 es la tupla durable que impide
// reutilizar una misma DecisionRef para otro efecto o repetir el mismo. El
// repositorio la reclama con restriccion UNIQUE dentro de Reservar, antes de
// crear una sesion remota, y consume esa misma reclamacion en el commit
// atomico de agregado, manifiesto, auditoria y outbox. Abandono o caducidad no
// liberan DecisionRef para otra reclamacion.
type ConsumoDecisionPreparacionCargaDocumentalV1 struct {
	DecisionRef            string
	EfectoRef              string
	HuellaPlanEfectoSHA256 string
	EsquemaHuellaDecision  string
	HuellaDecisionSHA256   string
}

func (c ConsumoDecisionPreparacionCargaDocumentalV1) Validar() error {
	if !referenciaOpacaAlmacenValida(c.DecisionRef, 512) || strings.ContainsRune(c.DecisionRef, '*') ||
		!referenciaOpacaAlmacenValida(c.EfectoRef, 512) || strings.ContainsRune(c.EfectoRef, '*') ||
		!esSHA256Hexadecimal(c.HuellaPlanEfectoSHA256) ||
		c.EsquemaHuellaDecision != EsquemaHuellaDecisionAutorizacionReforzadaV1 ||
		!esSHA256Hexadecimal(c.HuellaDecisionSHA256) {
		return ErrConfirmacionCargaDocumentalInvalida
	}
	return nil
}

func ConsumoDecisionDesdeManifiestoPreparacionCargaDocumental(
	manifiesto domain.ManifiestoPreparacionCargaDirectaV1,
) (ConsumoDecisionPreparacionCargaDocumentalV1, error) {
	datos, err := manifiesto.Datos()
	if err != nil {
		return ConsumoDecisionPreparacionCargaDocumentalV1{}, ErrConfirmacionCargaDocumentalInvalida
	}
	consumo := ConsumoDecisionPreparacionCargaDocumentalV1{
		DecisionRef: datos.DecisionRef, EfectoRef: datos.EfectoRef,
		HuellaPlanEfectoSHA256: datos.HuellaPlanEfectoSHA256,
		EsquemaHuellaDecision:  datos.EsquemaHuellaDecision,
		HuellaDecisionSHA256:   datos.HuellaDecisionSHA256,
	}
	if consumo.Validar() != nil {
		return ConsumoDecisionPreparacionCargaDocumentalV1{}, ErrConfirmacionCargaDocumentalInvalida
	}
	return consumo, nil
}

// ConsumoDecisionDesdeContextoPreparacionCargaDocumental permite reclamar la
// decision antes de crear una sesion remota. La reclamacion no concede ninguna
// capacidad: solo reserva de forma fail-closed la misma tupla que mas tarde
// debera aparecer en el manifiesto y consumirse en el commit final.
func ConsumoDecisionDesdeContextoPreparacionCargaDocumental(
	contexto ContextoOperacionAlmacen,
) (ConsumoDecisionPreparacionCargaDocumentalV1, error) {
	if contexto.validarParaPaso(AccionAlmacenPrepararCargaDirecta) != nil {
		return ConsumoDecisionPreparacionCargaDocumentalV1{}, ErrConfirmacionCargaDocumentalInvalida
	}
	proyeccion, err := contexto.Proyeccion()
	if err != nil || proyeccion.AccionNegocio != AccionNegocioPrepararCargaDocumental ||
		proyeccion.AccionTecnica != AccionAlmacenPrepararCargaDirecta ||
		proyeccion.PasoRef != PasoAlmacenPrepararCargaDirecta {
		return ConsumoDecisionPreparacionCargaDocumentalV1{}, ErrConfirmacionCargaDocumentalInvalida
	}
	evidencia, err := contexto.EvidenciaAutorizacion()
	if err != nil {
		return ConsumoDecisionPreparacionCargaDocumentalV1{}, ErrConfirmacionCargaDocumentalInvalida
	}
	datosEvidencia, err := evidencia.Datos()
	if err != nil || datosEvidencia.Decision.DecisionRef != proyeccion.AutorizacionRef ||
		datosEvidencia.HuellaDecisionSHA256 != proyeccion.HuellaDecisionSHA256 {
		return ConsumoDecisionPreparacionCargaDocumentalV1{}, ErrConfirmacionCargaDocumentalInvalida
	}
	consumo := ConsumoDecisionPreparacionCargaDocumentalV1{
		DecisionRef: proyeccion.AutorizacionRef, EfectoRef: proyeccion.EfectoRef,
		HuellaPlanEfectoSHA256: proyeccion.HuellaPlanEfectoSHA256,
		EsquemaHuellaDecision:  datosEvidencia.EsquemaHuella,
		HuellaDecisionSHA256:   proyeccion.HuellaDecisionSHA256,
	}
	if consumo.Validar() != nil {
		return ConsumoDecisionPreparacionCargaDocumentalV1{}, ErrConfirmacionCargaDocumentalInvalida
	}
	return consumo, nil
}

// PreparacionCargaDocumentalPersistida es una instantanea coherente. El
// repositorio nunca devuelve el agregado y el manifiesto mediante dos lecturas
// independientes que puedan observar versiones distintas.
type PreparacionCargaDocumentalPersistida struct {
	Carga      domain.CargaDocumental
	Manifiesto domain.ManifiestoPreparacionCargaDirectaV1
}

func (p PreparacionCargaDocumentalPersistida) Validar() error {
	if p.Carga.Validar() != nil || p.Carga.Estado != domain.EstadoCargaDocumentalPreparada ||
		p.Manifiesto.ValidarContraCarga(p.Carga) != nil {
		return ErrConfirmacionCargaDocumentalInvalida
	}
	return nil
}

func NuevaConfirmacionTransicionCargaDocumental(
	anterior, siguiente domain.CargaDocumental,
	auditoria domain.AuditEntry,
	evento domain.Event,
) (ConfirmacionTransicionCargaDocumental, error) {
	anterior = clonarCargaDocumentalPuerto(anterior)
	siguiente = clonarCargaDocumentalPuerto(siguiente)
	auditoria = clonarAuditoriaCargaDocumental(auditoria)
	evento = clonarEventoCargaDocumental(evento)
	huellaAnterior, err := anterior.HuellaSHA256()
	if err != nil {
		return ConfirmacionTransicionCargaDocumental{}, ErrConfirmacionCargaDocumentalInvalida
	}
	confirmacion := ConfirmacionTransicionCargaDocumental{
		VersionEsperada: anterior.Version, HuellaAnteriorSHA256: huellaAnterior,
		Carga: siguiente, Auditoria: auditoria, Evento: evento,
	}
	if err := confirmacion.ValidarContra(anterior); err != nil {
		return ConfirmacionTransicionCargaDocumental{}, err
	}
	return confirmacion, nil
}

func (c ConfirmacionTransicionCargaDocumental) ValidarContra(anterior domain.CargaDocumental) error {
	anterior = clonarCargaDocumentalPuerto(anterior)
	c.Carga = clonarCargaDocumentalPuerto(c.Carga)
	c.Auditoria = clonarAuditoriaCargaDocumental(c.Auditoria)
	c.Evento = clonarEventoCargaDocumental(c.Evento)
	accionEsperada, eventoEsperado, correspondenciaValida := correspondenciaTransicionCargaDocumental(
		anterior.Estado,
		c.Carga.Estado,
	)
	if anterior.Validar() != nil || c.Carga.Validar() != nil || c.VersionEsperada != anterior.Version ||
		!transicionCargaDocumentalExacta(anterior, c.Carga) || !correspondenciaValida ||
		c.HuellaAnteriorSHA256 == "" || c.Auditoria.ActorID == "" || c.Auditoria.Action != accionEsperada ||
		c.Auditoria.ModuleID != c.Carga.ModuloID || c.Auditoria.SubjectRef != c.Carga.ID ||
		c.Auditoria.ObjectVersion != c.Carga.Version || c.Auditoria.BeforeHash != c.HuellaAnteriorSHA256 ||
		c.Auditoria.AfterHash == "" || c.Auditoria.AuthorizationRef != autorizacionTransicionCarga(c.Carga) ||
		c.Auditoria.Purpose != c.Carga.Finalidad ||
		c.Auditoria.CorrelationRef != c.Carga.CorrelacionRef || c.Auditoria.Result != "correcto" ||
		c.Auditoria.OccurredAt.IsZero() || !c.Auditoria.OccurredAt.Equal(c.Carga.ActualizadaEn) ||
		c.Evento.Type != eventoEsperado || c.Evento.ModuleID != c.Carga.ModuloID || c.Evento.SubjectRef != c.Carga.ID ||
		c.Evento.ActorID != c.Auditoria.ActorID || c.Evento.OccurredAt.IsZero() ||
		!c.Evento.OccurredAt.Equal(c.Auditoria.OccurredAt) || !payloadEventoCargaDocumentalExacto(c.Evento, c.Carga) {
		return ErrConfirmacionCargaDocumentalInvalida
	}
	huellaAnteriorReal, err := anterior.HuellaSHA256()
	if err != nil || c.HuellaAnteriorSHA256 != huellaAnteriorReal {
		return ErrConfirmacionCargaDocumentalInvalida
	}
	huellaSiguiente, err := c.Carga.HuellaSHA256()
	if err != nil || c.Auditoria.AfterHash != huellaSiguiente {
		return ErrConfirmacionCargaDocumentalInvalida
	}
	return nil
}

// RepositorioCargasDocumentales aplica idempotencia, control optimista de
// concurrencia y la escritura atomica del agregado, auditoria y outbox.
// Reservar reclama DecisionRef+efecto+plan+huella antes de cualquier sesion
// remota. La reclamacion tiene estados explicitos y nunca vuelve a quedar
// disponible tras abandono o expiracion: un reintento necesita otra decision.
// ConfirmarPreparacion consume el token, consume DecisionRef una sola vez con
// su efecto/plan/huella y fija un unico manifiesto inmutable, todo en el mismo
// commit. DecisionRef debe tener una restriccion UNIQUE: tanto el replay exacto
// como su cruce con otro efecto se deniegan. ConfirmarTransicion nunca acepta
// el token ni sustituye ese manifiesto. ErrDecisionPreparacionCargaNoDisponible
// y ErrDecisionPreparacionCargaYaConsumida son conflictos inequivocos sin
// commit parcial; cualquier otro error de ConfirmarPreparacion puede ser una
// respuesta ambigua y exige reconciliacion.
type RepositorioCargasDocumentales interface {
	Reservar(context.Context, SolicitudReservarCargaDocumental) (ReservaCargaDocumental, error)
	ConfirmarPreparacion(context.Context, SolicitudConfirmarPreparacionCargaDocumental) error
	ConfirmarTransicion(context.Context, ConfirmacionTransicionCargaDocumental) error
	AbandonarReserva(context.Context, TokenReservaCargaDocumental) error
	Obtener(context.Context, string) (domain.CargaDocumental, error)
	ObtenerPreparacion(context.Context, string) (PreparacionCargaDocumentalPersistida, error)
}

// ReservaRecursoCargaDocumentalV3 procede del recibo durable de Documentos D16.
// El adaptador PostgreSQL coteja estos datos mediante la fachada propietaria
// en la misma transaccion que la reserva de carga; ninguna referencia concede
// titularidad por si sola.
type ReservaRecursoCargaDocumentalV3 struct {
	RecursoRef          string
	PersonaRef          string
	ClaveReserva        string
	HuellaReservaSHA256 string
	VersionReserva      uint64
}

// AutoridadCargaDocumentalV3 obtiene cinco decisiones independientes sobre
// preimagenes cerradas. La composicion productiva la alimenta desde el actor
// registrado y revalidado; ningun campo de la orden HTTP es autoridad.
type AutoridadCargaDocumentalV3 interface {
	AutorizarConsultaCarga(context.Context, []byte, domain.ResultadoContextoActorRegistradoV2, string) (AutorizacionCargaDocumentalV3, error)
	AutorizarReservaCarga(context.Context, []byte, domain.ResultadoContextoActorRegistradoV2, domain.CargaDocumental) (AutorizacionCargaDocumentalV3, error)
	AutorizarConfirmacionPreparacionCarga(context.Context, []byte, domain.ResultadoContextoActorRegistradoV2, domain.CargaDocumental) (AutorizacionCargaDocumentalV3, error)
	AutorizarRespuestaCarga(context.Context, []byte, domain.ResultadoContextoActorRegistradoV2, domain.CargaDocumental) (AutorizacionCargaDocumentalV3, error)
	AutorizarConfirmacionCarga(context.Context, []byte, domain.ResultadoContextoActorRegistradoV2, domain.CargaDocumental) (AutorizacionCargaDocumentalV3, error)
	AutorizarAnalisisCarga(context.Context, []byte, domain.ResultadoContextoActorRegistradoV2, domain.CargaDocumental) (AutorizacionCargaDocumentalV3, error)
	AutorizarPromocionCarga(context.Context, []byte, domain.ResultadoContextoActorRegistradoV2, domain.CargaDocumental) (AutorizacionCargaDocumentalV3, error)
}

func (r ReservaRecursoCargaDocumentalV3) ValidarPara(carga domain.CargaDocumental) error {
	if r.RecursoRef == "" || r.PersonaRef == "" || r.ClaveReserva == "" ||
		!esSHA256Hexadecimal(r.HuellaReservaSHA256) || r.VersionReserva == 0 ||
		r.RecursoRef != carga.RecursoRef || r.PersonaRef != carga.PrincipalID {
		return ErrReservaCargaDocumentalInvalida
	}
	return nil
}

const (
	AccionCargaDocumentalPrepararReservaV3         = "documentos.carga.preparar_reserva"
	AccionCargaDocumentalPrepararConfirmacionV3    = "documentos.carga.preparar_confirmacion"
	AccionCargaDocumentalPrepararRespuestaV3       = "documentos.carga.preparar_respuesta"
	AccionCargaDocumentalConsultarPropiaV3         = "documentos.carga.consultar_propia"
	AudienciaCargaDocumentalPrepararReservaV3      = "vec_documentos.carga.preparar_reserva.v1"
	AudienciaCargaDocumentalPrepararConfirmacionV3 = "vec_documentos.carga.preparar_confirmacion.v1"
	AudienciaCargaDocumentalPrepararRespuestaV3    = "vec_documentos.carga.preparar_respuesta.v1"
	AudienciaCargaDocumentalConsultarPropiaV3      = "vec_documentos.carga.consultar_propia.v1"
	AudienciaCargaDocumentalConfirmarV3            = "vec_documentos.carga.confirmar.v1"
	AudienciaCargaDocumentalAnalizarV3             = "vec_documentos.carga.analizar.v1"
	AudienciaCargaDocumentalPromoverV3             = "vec_documentos.carga.promover.v1"
)

func audienciaCargaDocumentalV3(accion string) string {
	switch accion {
	case AccionCargaDocumentalPrepararReservaV3:
		return AudienciaCargaDocumentalPrepararReservaV3
	case AccionCargaDocumentalPrepararConfirmacionV3:
		return AudienciaCargaDocumentalPrepararConfirmacionV3
	case AccionCargaDocumentalPrepararRespuestaV3:
		return AudienciaCargaDocumentalPrepararRespuestaV3
	case AccionCargaDocumentalConsultarPropiaV3:
		return AudienciaCargaDocumentalConsultarPropiaV3
	case accionCargaDocumentalConfirmar:
		return AudienciaCargaDocumentalConfirmarV3
	case accionCargaDocumentalAnalizar:
		return AudienciaCargaDocumentalAnalizarV3
	case accionCargaDocumentalPromover:
		return AudienciaCargaDocumentalPromoverV3
	default:
		return ""
	}
}

// AutorizacionCargaDocumentalV3 transporta material integro emitido por la
// autoridad nominal. Su validacion local detecta cruces y caducidad; solo el
// consumidor AD240 puede verificarlo y consumirlo dentro del efecto SQL.
type AutorizacionCargaDocumentalV3 struct {
	Material        ExportacionMaterialConsumoAutorizacionAtestadaV3
	Accion          string
	Finalidad       string
	RecursoRef      string
	AmbitoRef       string
	PersonaRef      string
	PerfilActivoRef string
	CorrelacionRef  string
}

func (a AutorizacionCargaDocumentalV3) ValidarPara(
	accion string, preimagen []byte, actor domain.ResultadoContextoActorRegistradoV2,
	carga domain.CargaDocumental, ahora time.Time,
) error {
	if actor.Validar() != nil || a.Material.ValidarEstructura() != nil ||
		a.Accion != accion || a.Finalidad != carga.Finalidad ||
		a.RecursoRef != carga.ID || a.AmbitoRef != carga.RecursoRef || a.PersonaRef != carga.PrincipalID ||
		a.PersonaRef != actor.Contexto.PersonaRef ||
		a.PerfilActivoRef != actor.Contexto.PerfilActivoRef ||
		a.CorrelacionRef != carga.CorrelacionRef || len(preimagen) == 0 ||
		ahora.IsZero() {
		return ErrDecisionPreparacionCargaNoDisponible
	}
	r := a.Material.ResumenCapacidad()
	if r.ValidarEstructura() != nil || r.AudienciaConsumo() != audienciaCargaDocumentalV3(accion) ||
		r.Operacion() != accion || r.EfectoRef() != carga.ID ||
		r.EfectoHuellaSHA256() != HuellaPreimagenCargaDocumentalV3(preimagen) ||
		ahora.Before(r.EmitidaEn()) || !ahora.Before(r.ExpiraEn()) {
		return ErrDecisionPreparacionCargaNoDisponible
	}
	return nil
}

type SolicitudConsultarCargaPropiaV3 struct {
	CargaID           string
	RecursoRef        string
	ActorRegistradoV2 domain.ResultadoContextoActorRegistradoV2
	Finalidad         string
	CorrelacionRef    string
	IncluyeManifiesto bool
	AutorizacionV3    AutorizacionCargaDocumentalV3
}

func PreimagenConsultaCargaPropiaV3(s SolicitudConsultarCargaPropiaV3) ([]byte, error) {
	if !referenciaOpacaAlmacenValida(s.CargaID, 512) ||
		!referenciaOpacaAlmacenValida(s.RecursoRef, 512) || s.ActorRegistradoV2.Validar() != nil ||
		s.Finalidad == "" || s.CorrelacionRef == "" {
		return nil, ErrCargaDocumentalNoEncontrada
	}
	return (canonicodoc.ConsultaCargaV3{
		Accion: AccionCargaDocumentalConsultarPropiaV3, CargaID: s.CargaID,
		RecursoRef:           s.RecursoRef,
		PersonaRef:           s.ActorRegistradoV2.Contexto.PersonaRef,
		PerfilActivoRef:      s.ActorRegistradoV2.Contexto.PerfilActivoRef,
		RegistroContextoRef:  s.ActorRegistradoV2.RegistroContextoRef,
		HuellaContextoSHA256: s.ActorRegistradoV2.HuellaSHA256,
		Finalidad:            s.Finalidad, CorrelacionRef: s.CorrelacionRef,
		IncluyeManifiesto: s.IncluyeManifiesto,
	}).Preimagen()
}

func (s SolicitudConsultarCargaPropiaV3) Validar(ahora time.Time) error {
	preimagen, err := PreimagenConsultaCargaPropiaV3(s)
	if err != nil || s.AutorizacionV3.Material.ValidarEstructura() != nil ||
		s.AutorizacionV3.Accion != AccionCargaDocumentalConsultarPropiaV3 ||
		s.AutorizacionV3.RecursoRef != s.CargaID ||
		s.AutorizacionV3.AmbitoRef != s.RecursoRef ||
		s.AutorizacionV3.PersonaRef != s.ActorRegistradoV2.Contexto.PersonaRef ||
		s.AutorizacionV3.PerfilActivoRef != s.ActorRegistradoV2.Contexto.PerfilActivoRef ||
		s.AutorizacionV3.Finalidad != s.Finalidad ||
		s.AutorizacionV3.CorrelacionRef != s.CorrelacionRef || ahora.IsZero() {
		return ErrCargaDocumentalNoEncontrada
	}
	r := s.AutorizacionV3.Material.ResumenCapacidad()
	if r.ValidarEstructura() != nil || r.AudienciaConsumo() != AudienciaCargaDocumentalConsultarPropiaV3 ||
		r.Operacion() != AccionCargaDocumentalConsultarPropiaV3 || r.EfectoRef() != s.CargaID ||
		r.EfectoHuellaSHA256() != HuellaPreimagenCargaDocumentalV3(preimagen) ||
		ahora.Before(r.EmitidaEn()) || !ahora.Before(r.ExpiraEn()) {
		return ErrCargaDocumentalNoEncontrada
	}
	return nil
}

type ConsultaCargaPropiaV3 struct {
	Carga      domain.CargaDocumental
	Manifiesto *domain.ManifiestoPreparacionCargaDirectaV1
}

type RepositorioConsultaCargasDocumentalesV3 interface {
	ConsultarPropia(context.Context, SolicitudConsultarCargaPropiaV3) (ConsultaCargaPropiaV3, error)
}

// HuellaPreimagenCargaDocumentalV3 fija la identidad exacta del efecto que
// AD240 vuelve a calcular desde los bytes de la preimagen.
func HuellaPreimagenCargaDocumentalV3(preimagen []byte) string {
	return canonicodoc.HuellaEfectoCargaV3(preimagen)
}

type estadoEmisionReciboCargaV3 struct {
	mu               sync.Mutex
	material         []byte
	registro         RegistroReciboCargaDirecta
	carga            domain.CargaDocumental
	actor            domain.ResultadoContextoActorRegistradoV2
	huellaManifiesto string
	preimagen        []byte
	usada            bool
}

// PreparacionEmisionReciboCargaDirectaV3 retiene el secreto solo en memoria,
// sin serializacion ni logs. ConsumirParaRegistro lo usa una vez y lo borra
// aun si la respuesta SQL es incierta: el replay no genera otro recibo.
type PreparacionEmisionReciboCargaDirectaV3 struct{ estado *estadoEmisionReciboCargaV3 }

func NuevaPreparacionEmisionReciboCargaDirectaV3(
	material []byte, registro RegistroReciboCargaDirecta,
	carga domain.CargaDocumental, actor domain.ResultadoContextoActorRegistradoV2,
	manifiesto domain.ManifiestoPreparacionCargaDirectaV1,
) (PreparacionEmisionReciboCargaDirectaV3, error) {
	if len(material) == 0 || len(material) > 1024 || registro.Validar() != nil ||
		carga.Validar() != nil || carga.Estado != domain.EstadoCargaDocumentalPreparada ||
		actor.Validar() != nil || actor.Contexto.PersonaRef != carga.PrincipalID ||
		manifiesto.ValidarContraCarga(carga) != nil {
		return PreparacionEmisionReciboCargaDirectaV3{}, ErrReciboCargaDirectaNoValido
	}
	huellaManifiesto, err := manifiesto.HuellaSHA256()
	if err != nil {
		return PreparacionEmisionReciboCargaDirectaV3{}, ErrReciboCargaDirectaNoValido
	}
	actor, err = actor.Clonar()
	if err != nil {
		return PreparacionEmisionReciboCargaDirectaV3{}, ErrReciboCargaDirectaNoValido
	}
	preimagen, err := (canonicodoc.EmisionReciboCargaV3{
		Accion:  AccionCargaDocumentalPrepararRespuestaV3,
		CargaID: carga.ID, RecursoRef: carga.RecursoRef,
		VersionCarga: carga.Version, PersonaRef: carga.PrincipalID,
		RegistroContextoRef: actor.RegistroContextoRef, HuellaContextoSHA256: actor.HuellaSHA256,
		HuellaManifiestoSHA256: huellaManifiesto, IndiceHMAC: registro.IndiceHMAC,
		GrupoHMAC: registro.GrupoHMAC, VinculoHMAC: registro.VinculoHMAC,
		EvidenciaAltaRef: registro.EvidenciaAltaRef,
		ExpiraEn:         registro.ExpiraEn.Format(time.RFC3339Nano),
	}).Preimagen()
	if err != nil {
		return PreparacionEmisionReciboCargaDirectaV3{}, ErrReciboCargaDirectaNoValido
	}
	return PreparacionEmisionReciboCargaDirectaV3{estado: &estadoEmisionReciboCargaV3{
		material: append([]byte(nil), material...), registro: registro, carga: clonarCargaDocumentalPuerto(carga),
		actor: actor, huellaManifiesto: huellaManifiesto, preimagen: preimagen,
	}}, nil
}

func (p PreparacionEmisionReciboCargaDirectaV3) Preimagen() ([]byte, error) {
	if p.estado == nil {
		return nil, ErrReciboCargaDirectaNoValido
	}
	p.estado.mu.Lock()
	defer p.estado.mu.Unlock()
	if p.estado.usada {
		return nil, ErrReciboCargaDirectaNoValido
	}
	return append([]byte(nil), p.estado.preimagen...), nil
}

func (p PreparacionEmisionReciboCargaDirectaV3) ConsumirParaRegistro(
	fn func([]byte, RegistroReciboCargaDirectaV3) (ReciboCargaDirecta, error),
	autorizacion AutorizacionCargaDocumentalV3, ahora time.Time,
) (ReciboCargaDirecta, error) {
	if p.estado == nil || fn == nil {
		return ReciboCargaDirecta{}, ErrReciboCargaDirectaNoValido
	}
	p.estado.mu.Lock()
	if p.estado.usada {
		p.estado.mu.Unlock()
		return ReciboCargaDirecta{}, ErrReciboCargaDirectaNoValido
	}
	p.estado.usada = true
	material := p.estado.material
	p.estado.material = nil
	registro := RegistroReciboCargaDirectaV3{Registro: p.estado.registro, Carga: clonarCargaDocumentalPuerto(p.estado.carga),
		ActorRegistradoV2: p.estado.actor, HuellaManifiestoSHA256: p.estado.huellaManifiesto,
		Preimagen: append([]byte(nil), p.estado.preimagen...), AutorizacionV3: autorizacion}
	p.estado.mu.Unlock()
	defer func() {
		for i := range material {
			material[i] = 0
		}
	}()
	if registro.Validar(ahora) != nil {
		return ReciboCargaDirecta{}, ErrReciboCargaDirectaNoValido
	}
	return fn(material, registro)
}

func (p PreparacionEmisionReciboCargaDirectaV3) Descartar() {
	if p.estado == nil {
		return
	}
	p.estado.mu.Lock()
	defer p.estado.mu.Unlock()
	for i := range p.estado.material {
		p.estado.material[i] = 0
	}
	p.estado.material = nil
	p.estado.usada = true
}

func (PreparacionEmisionReciboCargaDirectaV3) String() string {
	return "[PREPARACION-RECIBO-CONFIDENCIAL]"
}
func (p PreparacionEmisionReciboCargaDirectaV3) GoString() string { return p.String() }
func (p PreparacionEmisionReciboCargaDirectaV3) LogValue() slog.Value {
	return slog.StringValue(p.String())
}
func (PreparacionEmisionReciboCargaDirectaV3) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionReciboCargaProhibida
}
func (PreparacionEmisionReciboCargaDirectaV3) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionReciboCargaProhibida
}

type RegistroReciboCargaDirectaV3 struct {
	Registro               RegistroReciboCargaDirecta
	Carga                  domain.CargaDocumental
	ActorRegistradoV2      domain.ResultadoContextoActorRegistradoV2
	HuellaManifiestoSHA256 string
	Preimagen              []byte
	AutorizacionV3         AutorizacionCargaDocumentalV3
}

func (r RegistroReciboCargaDirectaV3) Validar(ahora time.Time) error {
	if r.Registro.Validar() != nil || r.Carga.Validar() != nil ||
		r.Carga.Estado != domain.EstadoCargaDocumentalPreparada ||
		r.ActorRegistradoV2.Validar() != nil || r.ActorRegistradoV2.Contexto.PersonaRef != r.Carga.PrincipalID ||
		!esSHA256Hexadecimal(r.HuellaManifiestoSHA256) ||
		r.AutorizacionV3.ValidarPara(AccionCargaDocumentalPrepararRespuestaV3, r.Preimagen,
			r.ActorRegistradoV2, r.Carga, ahora) != nil {
		return ErrReciboCargaDirectaNoValido
	}
	return nil
}

// PreimagenReservaCargaDocumentalV3 se construye solo con la carga sellada y
// el recibo D16 del servidor. La clave del navegador nunca pasa directamente.
func PreimagenReservaCargaDocumentalV3(s SolicitudReservarCargaDocumental,
	r ReservaRecursoCargaDocumentalV3) ([]byte, error) {
	if s.Validar() != nil || r.ValidarPara(s.Carga) != nil || !s.Token.Valido() {
		return nil, ErrReservaCargaDocumentalInvalida
	}
	huellaToken, err := s.Token.HuellaSHA256()
	if err != nil {
		return nil, ErrReservaCargaDocumentalInvalida
	}
	huellaCargaJSON, err := s.Carga.HuellaSHA256()
	if err != nil {
		return nil, ErrReservaCargaDocumentalInvalida
	}
	return (canonicodoc.ReservaCargaV3{
		Accion: AccionCargaDocumentalPrepararReservaV3, CargaID: s.Carga.ID,
		RecursoRef: r.RecursoRef, PersonaRef: r.PersonaRef, ClaveReserva: r.ClaveReserva,
		HuellaReservaSHA256: r.HuellaReservaSHA256, VersionReserva: r.VersionReserva,
		IndiceCargaHMAC: s.IndiceIdempotenciaHMAC, HuellaCargaHMAC: s.HuellaSolicitudHMAC,
		MIME: s.Carga.MIMEDeclarado, Tamano: s.Carga.TamanoDeclarado,
		HuellaContenido: s.Carga.HuellaDeclaradaSHA256, HuellaToken: huellaToken,
		HuellaCargaJSONSHA256: huellaCargaJSON,
	}).Preimagen()
}

// PreimagenConfirmacionPreparacionCargaDocumentalV3 ata el segundo COMMIT al
// token reservado, la postimagen y el manifiesto exacto ya obtenido de S3.
func PreimagenConfirmacionPreparacionCargaDocumentalV3(
	s SolicitudConfirmarPreparacionCargaDocumental,
) ([]byte, error) {
	if validarSolicitudConfirmarPreparacionCargaDocumental(s) != nil {
		return nil, ErrConfirmacionCargaDocumentalInvalida
	}
	huellaToken, errToken := s.Token.HuellaSHA256()
	huellaManifiesto, errManifiesto := s.Manifiesto.HuellaSHA256()
	datosManifiesto, errDatos := s.Manifiesto.Datos()
	huellaManifiestoJSON, errJSON := canonicodoc.HuellaJSONCargaV3(datosManifiesto)
	huellaCarga, errCarga := s.Confirmacion.Carga.HuellaSHA256()
	if errToken != nil || errManifiesto != nil || errDatos != nil || errJSON != nil || errCarga != nil ||
		s.Confirmacion.Auditoria.AfterHash != huellaCarga {
		return nil, ErrConfirmacionCargaDocumentalInvalida
	}
	return (canonicodoc.ConfirmacionPreparacionCargaV3{
		Accion:  AccionCargaDocumentalPrepararConfirmacionV3,
		CargaID: s.Confirmacion.Carga.ID, RecursoRef: s.Confirmacion.Carga.RecursoRef,
		PersonaRef: s.Confirmacion.Carga.PrincipalID, HuellaToken: huellaToken,
		HuellaManifiesto: huellaManifiesto, HuellaManifiestoJSON: huellaManifiestoJSON,
		VersionEsperada:  s.Confirmacion.VersionEsperada,
		HuellaAnterior:   s.Confirmacion.HuellaAnteriorSHA256,
		VersionSiguiente: s.Confirmacion.Carga.Version, HuellaSiguiente: huellaCarga,
	}).Preimagen()
}

// PreimagenTransicionCargaDocumentalV3 liga el CAS, la postimagen y la
// evidencia de auditoria del paso. Cada accion usa audiencia AD240 separada.
func PreimagenTransicionCargaDocumentalV3(c ConfirmacionTransicionCargaDocumental) ([]byte, error) {
	if c.Carga.Validar() != nil || c.VersionEsperada < 1 ||
		!esSHA256Hexadecimal(c.HuellaAnteriorSHA256) ||
		c.Carga.Version != c.VersionEsperada+1 ||
		c.Auditoria.Action != accionCargaDocumentalConfirmar &&
			c.Auditoria.Action != accionCargaDocumentalAnalizar &&
			c.Auditoria.Action != accionCargaDocumentalPromover ||
		c.Auditoria.ActorID == "" || c.Auditoria.AuthorizationRef == "" {
		return nil, ErrConfirmacionCargaDocumentalInvalida
	}
	huellaSiguiente, err := c.Carga.HuellaSHA256()
	if err != nil || c.Auditoria.AfterHash != huellaSiguiente ||
		c.Auditoria.BeforeHash != c.HuellaAnteriorSHA256 {
		return nil, ErrConfirmacionCargaDocumentalInvalida
	}
	return (canonicodoc.TransicionCargaV3{
		Accion: c.Auditoria.Action, CargaID: c.Carga.ID, RecursoRef: c.Carga.RecursoRef,
		PersonaRef: c.Carga.PrincipalID, VersionEsperada: c.VersionEsperada,
		HuellaAnterior: c.HuellaAnteriorSHA256, VersionSiguiente: c.Carga.Version,
		HuellaSiguiente: huellaSiguiente, AutorizacionV1Ref: c.Auditoria.AuthorizationRef,
		Evento: c.Evento.Type,
	}).Preimagen()
}

// ValidarRecursoBaseManifiestoPreparacionCargaDocumental comprueba antes del
// PDP que el contexto ABAC completo sigue siendo exactamente el preparado.
// Referencia, modulo y tipo se contrastan ademas de la huella para que esta no
// pueda emplearse como sustituto ambiguo de la identidad del recurso.
func ValidarRecursoBaseManifiestoPreparacionCargaDocumental(
	manifiesto domain.ManifiestoPreparacionCargaDirectaV1,
	carga domain.CargaDocumental,
	recursoBase domain.RecursoAutorizable,
) error {
	if manifiesto.ValidarContraCarga(carga) != nil {
		return ErrConfirmacionCargaDocumentalInvalida
	}
	datos, err := manifiesto.Datos()
	if err != nil || recursoBase.Referencia != datos.RecursoRef ||
		recursoBase.ModuloID != datos.ModuloID || recursoBase.Tipo != datos.TipoRecurso {
		return ErrConfirmacionCargaDocumentalInvalida
	}
	huellaBase, err := HuellaRecursoBaseCargaDocumental(recursoBase)
	if err != nil || huellaBase != datos.HuellaRecursoBaseSHA256 {
		return ErrConfirmacionCargaDocumentalInvalida
	}
	return nil
}

// ValidarManifiestoPreparacionParaConfirmacion cruza los hechos persistidos de
// la preparacion con la capacidad nueva de confirmar. No reconstruye el
// contexto anterior ni interpreta DecisionRef como autoridad.
func ValidarManifiestoPreparacionParaConfirmacion(
	manifiesto domain.ManifiestoPreparacionCargaDirectaV1,
	carga domain.CargaDocumental,
	contexto ContextoOperacionAlmacen,
	recursoBase domain.RecursoAutorizable,
) error {
	if ValidarRecursoBaseManifiestoPreparacionCargaDocumental(manifiesto, carga, recursoBase) != nil ||
		contexto.validarParaPaso(AccionAlmacenConfirmarCargaDirecta) != nil {
		return ErrConfirmacionCargaDocumentalInvalida
	}
	datos, errDatos := manifiesto.Datos()
	proyeccion, errContexto := contexto.Proyeccion()
	if errDatos != nil || errContexto != nil ||
		proyeccion.Esquema != datos.EsquemaContexto ||
		proyeccion.OperacionRef != datos.OperacionRef ||
		proyeccion.CorrelacionRef != datos.CorrelacionRef ||
		proyeccion.Finalidad != datos.Finalidad || proyeccion.Clasificacion != datos.Clasificacion ||
		proyeccion.AccionNegocio != AccionNegocioConfirmarCargaDocumental ||
		proyeccion.AccionTecnica != AccionAlmacenConfirmarCargaDirecta ||
		proyeccion.PasoRef != PasoAlmacenConfirmarCargaDirecta ||
		proyeccion.CargaRef != datos.CargaRef ||
		proyeccion.SujetoSeudonimoHMAC != datos.SujetoSeudonimoHMAC ||
		proyeccion.RecursoRef != datos.RecursoRef || proyeccion.ModuloID != datos.ModuloID ||
		proyeccion.TipoRecurso != datos.TipoRecurso ||
		proyeccion.HuellaSolicitudHMAC != datos.HuellaSolicitudHMAC ||
		proyeccion.ObjetoVinculado != (ReferenciaObjetoAlmacen{}) {
		return ErrConfirmacionCargaDocumentalInvalida
	}
	return nil
}

// ValidarResultadoCargaDirectaConManifiesto valida la respuesta contra hechos
// historicos durables y la capacidad de confirmacion actual. Deliberadamente
// no recibe ni fabrica una SolicitudPrepararCargaDirecta.
func ValidarResultadoCargaDirectaConManifiesto(
	resultado ResultadoOperacionObjeto,
	manifiesto domain.ManifiestoPreparacionCargaDirectaV1,
	carga domain.CargaDocumental,
	confirmacion SolicitudConfirmarCargaDirecta,
	capacidades CapacidadesAlmacenObjetos,
	recursoBase domain.RecursoAutorizable,
) error {
	comprobante, errComprobante := proyectarComprobanteConfirmacion(confirmacion)
	if ValidarManifiestoPreparacionParaConfirmacion(
		manifiesto, carga, confirmacion.contexto, recursoBase,
	) != nil ||
		errComprobante != nil || resultado.Validar() != nil ||
		!capacidades.CargaDirectaTemporal || !capacidades.ReferenciasOpacas ||
		!capacidades.IntegridadSHA256 {
		return ErrConfirmacionCargaDocumentalInvalida
	}
	datos, err := manifiesto.Datos()
	if err != nil || capacidades.TamanoMaximoObjeto < datos.Tamano ||
		resultado.Objeto.ConectorID != capacidades.ConectorID ||
		resultado.Objeto.ConectorID != datos.ConectorAlmacenID ||
		resultado.Evidencia.Accion != AccionAlmacenConfirmarCargaDirecta ||
		resultado.Evidencia.FundamentoRef != comprobante.intencionRef ||
		resultado.Evidencia.RealizadaEn.Before(comprobante.consumidoEn) ||
		!resultado.Evidencia.RealizadaEn.Before(comprobante.expiraEn) ||
		!resultado.Evidencia.RealizadaEn.Before(comprobante.validaHasta) ||
		comprobante.registradoEn.Before(datos.PreparadaEn) ||
		!comprobante.expiraEn.Equal(datos.ExpiraEn) ||
		resultado.Objeto.Zona != ZonaAlmacenCuarentena || resultado.Objeto.MIME != datos.MIME ||
		resultado.Objeto.Tamano != datos.Tamano ||
		resultado.Objeto.HuellaSHA256 != datos.HuellaContenidoSHA256 ||
		!evidenciaAlmacenLigada(resultado.Evidencia, confirmacion.contexto) {
		return ErrConfirmacionCargaDocumentalInvalida
	}
	return nil
}

// datosPrivadosConfirmacionCargaDirecta limita la proyeccion a los vinculos
// temporales que necesitan los validadores del puerto; no incluye HMAC ni
// atestacion y nunca forma parte de su API publica.
type datosPrivadosConfirmacionCargaDirecta struct {
	intencionRef                                     string
	registradoEn, consumidoEn, expiraEn, validaHasta time.Time
}

func proyectarComprobanteConfirmacion(
	confirmacion SolicitudConfirmarCargaDirecta,
) (datosPrivadosConfirmacionCargaDirecta, error) {
	if err := confirmacion.Validar(); err != nil {
		return datosPrivadosConfirmacionCargaDirecta{}, err
	}
	c := confirmacion.comprobante
	return datosPrivadosConfirmacionCargaDirecta{
		intencionRef: c.intencionRef, registradoEn: c.registradoEn, consumidoEn: c.consumidoEn,
		expiraEn: c.expiraEn, validaHasta: c.validaHasta,
	}, nil
}

type GeneradorIDCargaDocumental interface {
	NuevoIDCargaDocumental() (string, error)
}

var atributosReservadosRecursoBaseCargaDocumental = [...]string{
	AtributoAlmacenOperacionRef,
	AtributoAlmacenCargaRef,
	AtributoAlmacenClasificacion,
	AtributoAlmacenSujetoSeudonimoHMAC,
	AtributoAlmacenHuellaSolicitudHMAC,
	AtributoAlmacenEfectoRef,
	AtributoAlmacenObjetoRef,
	AtributoAlmacenObjetoVersion,
	AtributoAlmacenHuellaManifiestoSHA256,
}

// HuellaRecursoBaseCargaDocumental fija el recurso ABAC anterior a cualquier
// enriquecimiento tecnico. Incluye referencia, modulo, tipo y el contexto
// canonico completo de ambitos y atributos. Una clave reservada no se elimina
// silenciosamente: se rechaza para impedir que una entrada inyectada quede
// blanqueada al calcular la huella.
func HuellaRecursoBaseCargaDocumental(recurso domain.RecursoAutorizable) (string, error) {
	copia := domain.RecursoAutorizable{
		Referencia: recurso.Referencia, ModuloID: recurso.ModuloID, Tipo: recurso.Tipo,
		Ambitos:   clonarMapaCargaDocumental(recurso.Ambitos),
		Atributos: clonarMapaCargaDocumental(recurso.Atributos),
	}
	if copia.Validar() != nil {
		return "", ErrRecursoBaseCargaDocumentalInvalido
	}
	for clave, valor := range copia.Ambitos {
		if strings.ContainsRune(clave, '*') || strings.ContainsRune(valor, '*') ||
			strings.HasPrefix(clave, "almacen_") {
			return "", ErrRecursoBaseCargaDocumentalInvalido
		}
	}
	for clave, valor := range copia.Atributos {
		if strings.ContainsRune(clave, '*') || strings.ContainsRune(valor, '*') ||
			strings.HasPrefix(clave, "almacen_") {
			return "", ErrRecursoBaseCargaDocumentalInvalido
		}
	}
	for _, clave := range atributosReservadosRecursoBaseCargaDocumental {
		if _, existe := copia.Ambitos[clave]; existe {
			return "", ErrRecursoBaseCargaDocumentalInvalido
		}
		if _, existe := copia.Atributos[clave]; existe {
			return "", ErrRecursoBaseCargaDocumentalInvalido
		}
	}
	huellaContexto, err := copia.HuellaContextoAutorizacionSHA256()
	if err != nil {
		return "", ErrRecursoBaseCargaDocumentalInvalido
	}
	valores := []string{
		EsquemaHuellaRecursoBaseCargaDocumentalV1,
		copia.Referencia,
		copia.ModuloID,
		copia.Tipo,
		huellaContexto,
	}
	var canonico strings.Builder
	for _, valor := range valores {
		canonico.WriteString(strconv.Itoa(len(valor)))
		canonico.WriteByte(':')
		canonico.WriteString(valor)
		canonico.WriteByte('\n')
	}
	suma := sha256.Sum256([]byte(canonico.String()))
	return hex.EncodeToString(suma[:]), nil
}

// SelladorSolicitudCargaDocumental usa una clave distinta de la que indexa
// reintentos. La huella identifica todos los datos con efecto de la orden sin
// persistir la clave aportada por el navegador.
type SelladorSolicitudCargaDocumental interface {
	SellarSolicitudCargaDocumental(context.Context, []byte) (string, error)
}

// SelladorVinculoSesionCarga usa una clave distinta de la idempotencia. Su
// salida debe tener el formato hmac-sha256:<version>:<hex>.
type SelladorVinculoSesionCarga interface {
	SellarVinculoSesionCarga(context.Context, string) (string, error)
}

func hmacSHA256PuertoValido(valor string) bool {
	partes := strings.Split(valor, ":")
	return len(partes) == 3 && partes[0] == "hmac-sha256" &&
		referenciaOpacaAlmacenValida(partes[1], 64) && esSHA256Hexadecimal(partes[2])
}

func correspondenciaTransicionCargaDocumental(
	anterior, siguiente domain.EstadoCargaDocumental,
) (string, string, bool) {
	switch anterior {
	case domain.EstadoCargaDocumentalReservada:
		return accionCargaDocumentalPreparar, eventoCargaDocumentalPreparada,
			siguiente == domain.EstadoCargaDocumentalPreparada
	case domain.EstadoCargaDocumentalPreparada:
		return accionCargaDocumentalConfirmar, eventoCargaDocumentalRecibida,
			siguiente == domain.EstadoCargaDocumentalCuarentena
	case domain.EstadoCargaDocumentalCuarentena:
		return accionCargaDocumentalAnalizar, eventoCargaDocumentalAnalizada,
			siguiente == domain.EstadoCargaDocumentalAnalizadaLimpia ||
				siguiente == domain.EstadoCargaDocumentalRetenidaSeguridad
	case domain.EstadoCargaDocumentalAnalizadaLimpia:
		return accionCargaDocumentalPromover, eventoCargaDocumentalPromovida,
			siguiente == domain.EstadoCargaDocumentalAdmitida
	default:
		return "", "", false
	}
}

// transicionCargaDocumentalExacta aplica una lista positiva cerrada. Parte de
// una copia profunda del estado anterior y solo sustituye los campos que la
// transicion concreta puede crear. Cualquier otra diferencia falla cerrada.
func transicionCargaDocumentalExacta(anterior, siguiente domain.CargaDocumental) bool {
	_, _, permitida := correspondenciaTransicionCargaDocumental(anterior.Estado, siguiente.Estado)
	if !permitida || siguiente.Version != anterior.Version+1 ||
		!siguiente.ActualizadaEn.After(anterior.ActualizadaEn) {
		return false
	}

	esperada := clonarCargaDocumentalPuerto(anterior)
	esperada.Version = siguiente.Version
	esperada.Estado = siguiente.Estado
	esperada.ActualizadaEn = siguiente.ActualizadaEn

	switch anterior.Estado {
	case domain.EstadoCargaDocumentalReservada:
		esperada.VinculoSesionHMAC = siguiente.VinculoSesionHMAC
		esperada.AutorizacionPreparacionRef = siguiente.AutorizacionPreparacionRef
		esperada.PreparadaEn = siguiente.PreparadaEn
	case domain.EstadoCargaDocumentalPreparada:
		if !siguiente.ActualizadaEn.Before(anterior.ExpiraEn) {
			return false
		}
		esperada.AutorizacionRecepcionRef = siguiente.AutorizacionRecepcionRef
		esperada.ContenidoCuarentena = clonarContenidoCargaDocumental(siguiente.ContenidoCuarentena)
	case domain.EstadoCargaDocumentalCuarentena:
		esperada.AutorizacionAnalisisRef = siguiente.AutorizacionAnalisisRef
		esperada.Analisis = clonarAnalisisCargaDocumental(siguiente.Analisis)
	case domain.EstadoCargaDocumentalAnalizadaLimpia:
		if anterior.Analisis == nil || !siguiente.ActualizadaEn.After(anterior.Analisis.CompletadoEn) {
			return false
		}
		esperada.AutorizacionPromocionRef = siguiente.AutorizacionPromocionRef
		esperada.ContenidoAdmitido = clonarContenidoCargaDocumental(siguiente.ContenidoAdmitido)
	default:
		return false
	}

	// La comparacion total hace que cualquier campo que se incorpore al agregado
	// en el futuro quede denegado hasta declararlo de forma expresa arriba.
	return reflect.DeepEqual(esperada, siguiente)
}

func payloadEventoCargaDocumentalExacto(evento domain.Event, carga domain.CargaDocumental) bool {
	return len(evento.Payload) == 3 && evento.Payload["carga_ref"] == carga.ID &&
		evento.Payload["estado"] == string(carga.Estado) &&
		evento.Payload["version"] == strconv.Itoa(carga.Version)
}

func clonarCargaDocumentalPuerto(carga domain.CargaDocumental) domain.CargaDocumental {
	clon := carga
	clon.ContenidoCuarentena = clonarContenidoCargaDocumental(carga.ContenidoCuarentena)
	clon.Analisis = clonarAnalisisCargaDocumental(carga.Analisis)
	clon.ContenidoAdmitido = clonarContenidoCargaDocumental(carga.ContenidoAdmitido)
	return clon
}

func clonarConfirmacionCargaDocumental(
	confirmacion ConfirmacionTransicionCargaDocumental,
) ConfirmacionTransicionCargaDocumental {
	clon := confirmacion
	clon.Carga = clonarCargaDocumentalPuerto(confirmacion.Carga)
	clon.Auditoria = clonarAuditoriaCargaDocumental(confirmacion.Auditoria)
	clon.Evento = clonarEventoCargaDocumental(confirmacion.Evento)
	clon.ActorRegistradoV2 = domain.ResultadoContextoActorRegistradoV2{}
	clon.AutorizacionV3.Material = ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	if actor, err := confirmacion.ActorRegistradoV2.Clonar(); err == nil {
		clon.ActorRegistradoV2 = actor
	}
	if material, err := NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(
		confirmacion.AutorizacionV3.Material.CapacidadCanonica(),
		confirmacion.AutorizacionV3.Material.ResumenCapacidad(),
		confirmacion.AutorizacionV3.Material.DecisionCanonica(),
		confirmacion.AutorizacionV3.Material.MotivoCanonico(),
		confirmacion.AutorizacionV3.Material.ContextoActorCanonico(),
		confirmacion.AutorizacionV3.Material.PersonaVersion(),
		confirmacion.AutorizacionV3.Material.PerfilVersion(),
		confirmacion.AutorizacionV3.Material.PayloadVECAD3(),
		confirmacion.AutorizacionV3.Material.SobreCOSESign1(),
		confirmacion.AutorizacionV3.Material.EvidenciaVerificacion(),
		confirmacion.AutorizacionV3.Material.RaizPublicaSPKI(),
	); err == nil {
		clon.AutorizacionV3.Material = material
	}
	return clon
}

func clonarContenidoCargaDocumental(
	contenido *domain.ContenidoCargaDocumental,
) *domain.ContenidoCargaDocumental {
	if contenido == nil {
		return nil
	}
	clon := *contenido
	return &clon
}

func clonarAnalisisCargaDocumental(analisis *domain.AnalisisCargaDocumental) *domain.AnalisisCargaDocumental {
	if analisis == nil {
		return nil
	}
	clon := *analisis
	return &clon
}

func clonarAuditoriaCargaDocumental(auditoria domain.AuditEntry) domain.AuditEntry {
	clon := auditoria
	clon.ActorRoles = append([]string(nil), auditoria.ActorRoles...)
	clon.Metadata = clonarMapaCargaDocumental(auditoria.Metadata)
	return clon
}

func clonarEventoCargaDocumental(evento domain.Event) domain.Event {
	clon := evento
	clon.Payload = clonarMapaCargaDocumental(evento.Payload)
	return clon
}

func clonarMapaCargaDocumental(origen map[string]string) map[string]string {
	if origen == nil {
		return nil
	}
	clon := make(map[string]string, len(origen))
	for clave, valor := range origen {
		clon[clave] = valor
	}
	return clon
}

func autorizacionTransicionCarga(carga domain.CargaDocumental) string {
	switch carga.Estado {
	case domain.EstadoCargaDocumentalPreparada:
		return carga.AutorizacionPreparacionRef
	case domain.EstadoCargaDocumentalCuarentena:
		return carga.AutorizacionRecepcionRef
	case domain.EstadoCargaDocumentalAnalizadaLimpia, domain.EstadoCargaDocumentalRetenidaSeguridad:
		return carga.AutorizacionAnalisisRef
	case domain.EstadoCargaDocumentalAdmitida:
		return carga.AutorizacionPromocionRef
	default:
		return ""
	}
}
