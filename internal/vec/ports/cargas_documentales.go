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
	"time"

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
	IndiceIdempotenciaHMAC string
	HuellaSolicitudHMAC    string
	Carga                  domain.CargaDocumental
	DecisionPreparacion    ConsumoDecisionPreparacionCargaDocumentalV1
	SolicitadaEn           time.Time
	ReservaExpiraEn        time.Time
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

// ReservaCargaDocumental devuelve exactamente uno de dos resultados: token
// nuevo, o agregado previamente confirmado. Una reserva en curso ajena se
// comunica como error y nunca se roba silenciosamente.
type ReservaCargaDocumental struct {
	Token    TokenReservaCargaDocumental
	Repetida bool
	Carga    domain.CargaDocumental
}

func (r ReservaCargaDocumental) Validar() error {
	if r.Carga.Validar() != nil {
		return ErrReservaCargaDocumentalInvalida
	}
	if r.Repetida {
		if r.Token.Valido() || r.Carga.Estado == domain.EstadoCargaDocumentalReservada {
			return ErrReservaCargaDocumentalInvalida
		}
		return nil
	}
	if !r.Token.Valido() || r.Carga.Estado != domain.EstadoCargaDocumentalReservada {
		return ErrReservaCargaDocumentalInvalida
	}
	return nil
}

type ConfirmacionTransicionCargaDocumental struct {
	VersionEsperada      int
	HuellaAnteriorSHA256 string
	Carga                domain.CargaDocumental
	Auditoria            domain.AuditEntry
	Evento               domain.Event
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
	Token        TokenReservaCargaDocumental
	Confirmacion ConfirmacionTransicionCargaDocumental
	Manifiesto   domain.ManifiestoPreparacionCargaDirectaV1
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
