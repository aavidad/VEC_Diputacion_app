package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

var (
	ErrDependenciaAltaCobroRequerida   = errors.New("vec: dependencia de alta de cobro requerida")
	ErrSolicitudAltaCobroInvalida      = errors.New("vec: solicitud de alta de cobro invalida")
	ErrLiquidacionCobroNoConfiable     = errors.New("vec: liquidacion de cobro no confiable")
	ErrLiquidacionCobroNoExigible      = errors.New("vec: liquidacion de cobro no exigible")
	ErrResultadoAltaCobroNoConfiable   = errors.New("vec: resultado de alta de cobro no confiable")
	ErrPersistenciaAltaCobroIncompleta = errors.New("vec: persistencia atomica de alta de cobro incompleta")
	ErrSerializacionAltaCobroProhibida = errors.New("vec: serializacion directa de alta de cobro prohibida")
)

const (
	versionInstantaneaLiquidacionCobro = 1
	duracionReservaAltaCobro           = 30 * time.Second
	duracionLimpiezaReservaAltaCobro   = 2 * time.Second
	vigenciaMaximaAltaCobro            = 366 * 24 * time.Hour
	motivoAltaOrdenCobro               = "Alta desde liquidacion autoritativa exigible"
)

// EstadoLiquidacionCobro es una lista cerrada. Solo Exigible permite crear
// una orden; los demas estados existen para que una fuente pueda expresar una
// negativa explicita sin convertir una omision o un valor nuevo en permiso.
type EstadoLiquidacionCobro string

const (
	EstadoLiquidacionCobroEmitida    EstadoLiquidacionCobro = "emitida"
	EstadoLiquidacionCobroExigible   EstadoLiquidacionCobro = "exigible"
	EstadoLiquidacionCobroSuspendida EstadoLiquidacionCobro = "suspendida"
	EstadoLiquidacionCobroAnulada    EstadoLiquidacionCobro = "anulada"
	EstadoLiquidacionCobroPagada     EstadoLiquidacionCobro = "pagada"
	EstadoLiquidacionCobroCaducada   EstadoLiquidacionCobro = "caducada"
)

func (e EstadoLiquidacionCobro) valida() bool {
	switch e {
	case EstadoLiquidacionCobroEmitida, EstadoLiquidacionCobroExigible,
		EstadoLiquidacionCobroSuspendida, EstadoLiquidacionCobroAnulada,
		EstadoLiquidacionCobroPagada, EstadoLiquidacionCobroCaducada:
		return true
	default:
		return false
	}
}

// DatosLiquidacionCobroAutoritativa es la instantanea completa que debe
// obtener un adaptador de liquidaciones desde su registro oficial. No es un
// DTO de entrada: importe, tarifa, sujeto, estado y vigencia nunca proceden de
// la peticion que inicia el caso de uso.
type DatosLiquidacionCobroAutoritativa struct {
	LiquidacionRef    string
	Revision          uint64
	HuellaSHA256      string
	ExpedienteRef     string
	SolicitudRef      string
	Tarifa            domain.ReferenciaTarifaCobro
	SujetoRef         string
	RepresentacionRef string
	Importe           domain.DineroCobro
	Concepto          string
	Finalidad         string
	Estado            EstadoLiquidacionCobro
	ExigibleDesde     time.Time
	ExigibleHasta     time.Time
}

type datosLiquidacionCobroAutoritativa struct {
	DatosLiquidacionCobroAutoritativa
}

// LiquidacionCobroAutoritativa es opaca y no serializable para impedir que un
// adaptador HTTP la reconstruya accidentalmente. Su constructor comprueba que
// la huella publicada liga exactamente todos los datos funcionales.
type LiquidacionCobroAutoritativa struct {
	datos *datosLiquidacionCobroAutoritativa
}

func NuevaLiquidacionCobroAutoritativa(
	datos DatosLiquidacionCobroAutoritativa,
) (LiquidacionCobroAutoritativa, error) {
	huella, err := CalcularHuellaLiquidacionCobroAutoritativa(datos)
	if err != nil || datos.HuellaSHA256 != huella {
		return LiquidacionCobroAutoritativa{}, ErrLiquidacionCobroNoConfiable
	}
	resultado := LiquidacionCobroAutoritativa{datos: &datosLiquidacionCobroAutoritativa{
		DatosLiquidacionCobroAutoritativa: datos,
	}}
	if resultado.validar() != nil {
		return LiquidacionCobroAutoritativa{}, ErrLiquidacionCobroNoConfiable
	}
	return resultado, nil
}

func (l LiquidacionCobroAutoritativa) Datos() (DatosLiquidacionCobroAutoritativa, error) {
	if err := l.validar(); err != nil {
		return DatosLiquidacionCobroAutoritativa{}, err
	}
	return l.datos.DatosLiquidacionCobroAutoritativa, nil
}

func (l LiquidacionCobroAutoritativa) validar() error {
	if l.datos == nil {
		return ErrLiquidacionCobroNoConfiable
	}
	datos := l.datos.DatosLiquidacionCobroAutoritativa
	huella, err := CalcularHuellaLiquidacionCobroAutoritativa(datos)
	if err != nil || huella != datos.HuellaSHA256 {
		return ErrLiquidacionCobroNoConfiable
	}
	return nil
}

func (l LiquidacionCobroAutoritativa) exigibleEn(instante time.Time) bool {
	datos, err := l.Datos()
	return err == nil && instanteAplicacionCobroCanonico(instante) &&
		datos.Estado == EstadoLiquidacionCobroExigible &&
		!instante.Before(datos.ExigibleDesde) && instante.Before(datos.ExigibleHasta) &&
		datos.ExigibleHasta.Sub(instante) <= vigenciaMaximaAltaCobro
}

func (LiquidacionCobroAutoritativa) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionAltaCobroProhibida
}
func (*LiquidacionCobroAutoritativa) UnmarshalJSON([]byte) error {
	return ErrSerializacionAltaCobroProhibida
}
func (LiquidacionCobroAutoritativa) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionAltaCobroProhibida
}
func (LiquidacionCobroAutoritativa) String() string     { return "[LIQUIDACION-COBRO-AUTORITATIVA]" }
func (l LiquidacionCobroAutoritativa) GoString() string { return l.String() }
func (l LiquidacionCobroAutoritativa) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, l.String())
}

type instantaneaLiquidacionCobroCanonica struct {
	Version           int
	LiquidacionRef    string
	Revision          uint64
	ExpedienteRef     string
	SolicitudRef      string
	Tarifa            domain.ReferenciaTarifaCobro
	SujetoRef         string
	RepresentacionRef string
	Importe           domain.DineroCobro
	Concepto          string
	Finalidad         string
	Estado            EstadoLiquidacionCobro
	ExigibleDesde     time.Time
	ExigibleHasta     time.Time
}

// CalcularHuellaLiquidacionCobroAutoritativa ofrece al adaptador oficial el
// mismo esquema canonico que valida el caso de uso. La huella acredita
// integridad de la instantanea, no autoridad por si sola.
func CalcularHuellaLiquidacionCobroAutoritativa(
	datos DatosLiquidacionCobroAutoritativa,
) (string, error) {
	if datos.Revision == 0 || !datos.Estado.valida() ||
		!instanteAplicacionCobroCanonico(datos.ExigibleDesde) ||
		!instanteAplicacionCobroCanonico(datos.ExigibleHasta) ||
		!datos.ExigibleHasta.After(datos.ExigibleDesde) {
		return "", ErrLiquidacionCobroNoConfiable
	}
	// Reutiliza la lista positiva del dominio para referencias, tarifa,
	// sujeto, representacion, dinero y textos. No se normaliza ningun valor.
	_, err := domain.BytesCanonicosIdempotenciaAltaCobro(domain.AltaOrdenCobro{
		ExpedienteRef: datos.ExpedienteRef, SolicitudRef: datos.SolicitudRef,
		LiquidacionRef: datos.LiquidacionRef, Tarifa: datos.Tarifa,
		SujetoRef: datos.SujetoRef, RepresentacionRef: datos.RepresentacionRef,
		Importe: datos.Importe, Concepto: datos.Concepto, Finalidad: datos.Finalidad,
	})
	if err != nil {
		return "", ErrLiquidacionCobroNoConfiable
	}
	canonica := instantaneaLiquidacionCobroCanonica{
		Version: versionInstantaneaLiquidacionCobro, LiquidacionRef: datos.LiquidacionRef,
		Revision: datos.Revision, ExpedienteRef: datos.ExpedienteRef, SolicitudRef: datos.SolicitudRef,
		Tarifa: datos.Tarifa, SujetoRef: datos.SujetoRef, RepresentacionRef: datos.RepresentacionRef,
		Importe: datos.Importe, Concepto: datos.Concepto, Finalidad: datos.Finalidad,
		Estado: datos.Estado, ExigibleDesde: datos.ExigibleDesde, ExigibleHasta: datos.ExigibleHasta,
	}
	contenido, err := json.Marshal(canonica)
	if err != nil {
		return "", ErrLiquidacionCobroNoConfiable
	}
	suma := sha256.Sum256(contenido)
	return hex.EncodeToString(suma[:]), nil
}

// ConsultaLiquidacionCobro contiene exclusivamente la referencia opaca
// declarada por el iniciador. La fuente debe buscar todas las coincidencias;
// el servicio deniega cero o mas de una y nunca elige la primera.
type ConsultaLiquidacionCobro struct {
	LiquidacionRef string
}

// FuenteLiquidacionesCobro es una frontera de confianza pendiente de adaptador
// productivo. Debe leer liquidacion y version exacta de tarifa en una unica
// instantanea coherente del registro oficial. Una cache no revalidada, un DTO
// o una tabla editable por el usuario no satisfacen este contrato. Esta lectura
// permite decidir y construir la propuesta, pero nunca se considera atomica
// con su persistencia: ConfirmarCreacion debe volver a comparar el control
// oficial dentro de su propia transaccion.
type FuenteLiquidacionesCobro interface {
	BuscarLiquidacionesCobro(context.Context, ConsultaLiquidacionCobro) ([]LiquidacionCobroAutoritativa, error)
}

// SolicitudAltaOrdenCobro no contiene importe, moneda, tarifa, concepto,
// sujeto, estado ni caducidad. ContextoActor y Vinculo son capacidades opacas
// resueltas por la frontera de identidad. SesionRef y su HMAC deben obtenerse
// del contexto confiable del adaptador, nunca del cuerpo de la peticion.
type SolicitudAltaOrdenCobro struct {
	ContextoActor             domain.ContextoActor
	VinculoAutenticacionActor domain.VinculoAutenticacionActorV1
	SesionRef                 string
	HuellaSesionHMAC          string
	LiquidacionRef            string
	CorrelacionRef            string
}

func (SolicitudAltaOrdenCobro) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionAltaCobroProhibida
}
func (*SolicitudAltaOrdenCobro) UnmarshalJSON([]byte) error {
	return ErrSerializacionAltaCobroProhibida
}
func (SolicitudAltaOrdenCobro) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionAltaCobroProhibida
}
func (SolicitudAltaOrdenCobro) String() string { return "[SOLICITUD-ALTA-COBRO-INTERNA]" }

type DatosAltaOrdenCobroCompletada struct {
	Vista    domain.VistaTitularOrdenCobro
	Repetida bool
}

// AltaOrdenCobroCompletada solo expone la proyeccion minima del titular, nunca
// el agregado, su historial, la decision ni las huellas HMAC internas.
type AltaOrdenCobroCompletada struct {
	datos *DatosAltaOrdenCobroCompletada
}

func (a AltaOrdenCobroCompletada) Datos() (DatosAltaOrdenCobroCompletada, error) {
	if a.datos == nil || a.datos.Vista.OrdenRef == "" || !a.datos.Vista.Estado.Valido() ||
		a.datos.Vista.Importe.Validar() != nil || a.datos.Vista.CreadaEn.IsZero() ||
		a.datos.Vista.CaducaEn.IsZero() || a.datos.Vista.UltimoCambioEn.IsZero() {
		return DatosAltaOrdenCobroCompletada{}, ErrResultadoAltaCobroNoConfiable
	}
	return *a.datos, nil
}

func (AltaOrdenCobroCompletada) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionAltaCobroProhibida
}
func (*AltaOrdenCobroCompletada) UnmarshalJSON([]byte) error {
	return ErrSerializacionAltaCobroProhibida
}
func (AltaOrdenCobroCompletada) String() string { return "[ALTA-ORDEN-COBRO-COMPLETADA]" }

// ServicioAltaOrdenCobro es deliberadamente no exponible: no existe adaptador
// HTTP/CLI/MCP ni composicion de produccion. Solo sera habilitable cuando la
// fuente autoritativa y RepositorioOrdenesCobro duradero satisfagan sus
// contratos. El repositorio confirma agregado, auditoria y outbox en una unica
// transaccion; no se ofrece una ruta degradada de tres escrituras.
//
// La confirmacion exige CAS autoritativo de liquidacion y reloj transaccional.
// Sigue siendo puerta de despliegue disponer de adaptadores duraderos que
// cumplan realmente ambos contratos; mientras no existan, este servicio no
// debe cablearse a entradas.
type ServicioAltaOrdenCobro struct {
	repositorio   ports.RepositorioOrdenesCobro
	liquidaciones FuenteLiquidacionesCobro
	autorizador   ports.Autorizador
	verificador   domain.VerificadorAutenticacionCobro
	sellador      ports.SelladorSolicitudCobro
	generador     ports.GeneradorIDOrdenCobro
	reloj         ports.Reloj
}

func NuevoServicioAltaOrdenCobro(
	repositorio ports.RepositorioOrdenesCobro,
	liquidaciones FuenteLiquidacionesCobro,
	autorizador ports.Autorizador,
	verificador domain.VerificadorAutenticacionCobro,
	sellador ports.SelladorSolicitudCobro,
	generador ports.GeneradorIDOrdenCobro,
	reloj ports.Reloj,
) (*ServicioAltaOrdenCobro, error) {
	if dependenciaAltaCobroNula(repositorio) || dependenciaAltaCobroNula(liquidaciones) ||
		dependenciaAltaCobroNula(autorizador) || dependenciaAltaCobroNula(verificador) ||
		dependenciaAltaCobroNula(sellador) || dependenciaAltaCobroNula(generador) ||
		dependenciaAltaCobroNula(reloj) {
		return nil, ErrDependenciaAltaCobroRequerida
	}
	return &ServicioAltaOrdenCobro{
		repositorio: repositorio, liquidaciones: liquidaciones, autorizador: autorizador,
		verificador: verificador, sellador: sellador, generador: generador, reloj: reloj,
	}, nil
}

// Crear crea como maximo una orden para la instantanea autoritativa exacta.
// Una repeticion semanticamente identica devuelve la orden ya existente. Toda
// ausencia, ambiguedad, estado desconocido, representacion no acreditada o
// resultado inconsistente falla cerrado antes de persistir.
func (s *ServicioAltaOrdenCobro) Crear(
	ctx context.Context,
	solicitud SolicitudAltaOrdenCobro,
) (AltaOrdenCobroCompletada, error) {
	if ctx == nil || s == nil || dependenciaAltaCobroNula(s.repositorio) ||
		dependenciaAltaCobroNula(s.liquidaciones) || dependenciaAltaCobroNula(s.autorizador) ||
		dependenciaAltaCobroNula(s.verificador) || dependenciaAltaCobroNula(s.sellador) ||
		dependenciaAltaCobroNula(s.generador) || dependenciaAltaCobroNula(s.reloj) {
		return AltaOrdenCobroCompletada{}, denegacionAltaCobro(ErrDependenciaAltaCobroRequerida)
	}
	if err := ctx.Err(); err != nil {
		return AltaOrdenCobroCompletada{}, denegacionAltaCobro(err)
	}
	if err := validarSolicitudAltaCobro(solicitud); err != nil {
		return AltaOrdenCobroCompletada{}, denegacionAltaCobro(err)
	}
	actor, err := solicitud.ContextoActor.Clonar()
	if err != nil {
		return AltaOrdenCobroCompletada{}, denegacionAltaCobro(err)
	}
	// Principal no necesita contenedores vacios. Se eliminan para que ningun
	// conector inyectado pueda conservar y mutar memoria compartida del llamador.
	actor.Principal = domain.Principal{
		ID: actor.Principal.ID, AuthMethod: actor.Principal.AuthMethod,
		AuthAssurance: actor.Principal.AuthAssurance,
	}
	ahora := s.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if !instanteAplicacionCobroCanonico(ahora) ||
		!solicitud.VinculoAutenticacionActor.VigenteEn(ahora, actor) {
		return AltaOrdenCobroCompletada{}, denegacionAltaCobro(ErrSolicitudAltaCobroInvalida)
	}
	datosVinculo, err := solicitud.VinculoAutenticacionActor.Datos()
	if err != nil || datosVinculo.SesionRef != solicitud.SesionRef {
		return AltaOrdenCobroCompletada{}, denegacionAltaCobro(ErrSolicitudAltaCobroInvalida)
	}

	atestacion, err := domain.NuevaAtestacionAutenticacionCobro(
		ctx, s.verificador, solicitud.SesionRef, solicitud.HuellaSesionHMAC, ahora,
	)
	if err != nil {
		return AltaOrdenCobroCompletada{}, denegacionAltaCobro(err)
	}
	if err := ctx.Err(); err != nil {
		return AltaOrdenCobroCompletada{}, denegacionAltaCobro(err)
	}

	coincidencias, err := s.liquidaciones.BuscarLiquidacionesCobro(ctx, ConsultaLiquidacionCobro{
		LiquidacionRef: solicitud.LiquidacionRef,
	})
	if err != nil {
		if contextoErr := ctx.Err(); contextoErr != nil {
			return AltaOrdenCobroCompletada{}, denegacionAltaCobro(contextoErr)
		}
		return AltaOrdenCobroCompletada{}, denegacionAltaCobro(ErrLiquidacionCobroNoConfiable)
	}
	if err := ctx.Err(); err != nil {
		return AltaOrdenCobroCompletada{}, denegacionAltaCobro(err)
	}
	if len(coincidencias) != 1 {
		return AltaOrdenCobroCompletada{}, denegacionAltaCobro(ErrLiquidacionCobroNoConfiable)
	}
	liquidacion := coincidencias[0]
	datosLiquidacion, err := liquidacion.Datos()
	if err != nil || datosLiquidacion.LiquidacionRef != solicitud.LiquidacionRef ||
		!liquidacion.exigibleEn(ahora) {
		return AltaOrdenCobroCompletada{}, denegacionAltaCobro(ErrLiquidacionCobroNoExigible)
	}
	// Este primer corte solo concede pago propio. Representacion y alta por
	// personal tramitador requieren un puerto autoritativo adicional que pruebe
	// el mandato vigente; hasta entonces no se infieren de un identificador.
	if datosLiquidacion.SujetoRef != actor.PersonaRef ||
		datosLiquidacion.RepresentacionRef != "" {
		return AltaOrdenCobroCompletada{}, denegacionAltaCobro(ErrLiquidacionCobroNoExigible)
	}

	recurso, err := recursoAutorizableLiquidacionCobro(datosLiquidacion)
	if err != nil {
		return AltaOrdenCobroCompletada{}, denegacionAltaCobro(err)
	}
	actorAutorizacion, err := actor.Clonar()
	if err != nil {
		return AltaOrdenCobroCompletada{}, denegacionAltaCobro(err)
	}
	recursoAutorizacion := clonarRecursoAutorizableAltaCobro(recurso)
	decision, err := s.autorizador.Exigir(ctx, domain.SolicitudAutorizacion{
		Principal: actorAutorizacion.Principal, PerfilActivoRef: actorAutorizacion.PerfilActivoRef,
		ContextoActor: actorAutorizacion, VinculoAutenticacionActor: solicitud.VinculoAutenticacionActor,
		Accion: string(domain.AccionCobroCrearOrden), Recurso: recursoAutorizacion,
		Finalidad: datosLiquidacion.Finalidad, CorrelacionRef: solicitud.CorrelacionRef,
		Motivo: motivoAltaOrdenCobro,
	})
	if err != nil {
		if contextoErr := ctx.Err(); contextoErr != nil {
			return AltaOrdenCobroCompletada{}, denegacionAltaCobro(contextoErr)
		}
		return AltaOrdenCobroCompletada{}, denegacionAltaCobro(err)
	}
	if err := ctx.Err(); err != nil {
		return AltaOrdenCobroCompletada{}, denegacionAltaCobro(err)
	}
	if !solicitud.VinculoAutenticacionActor.CoincideExactamenteCon(
		decision.VinculoAutenticacionActor,
	) {
		return AltaOrdenCobroCompletada{}, denegacionAltaCobro(domain.ErrContextoAutorizacionCobroInvalido)
	}
	decision = clonarDecisionAutorizacionAltaCobro(decision)
	autorizacion, err := domain.NuevoContextoAutorizacionCobro(decision, atestacion, recurso, ahora)
	if err != nil {
		return AltaOrdenCobroCompletada{}, denegacionAltaCobro(err)
	}
	datosAutorizacion, err := autorizacion.Datos()
	if err != nil || datosAutorizacion.ActorRef != actor.PersonaRef ||
		datosAutorizacion.PerfilActivoRef != actor.PerfilActivoRef ||
		datosAutorizacion.AutenticacionRef != datosVinculo.AutenticacionRef ||
		datosAutorizacion.SesionRef != datosVinculo.SesionRef ||
		datosAutorizacion.Metodo != datosVinculo.MetodoObservado ||
		datosAutorizacion.Garantia != datosVinculo.GarantiaObservada {
		return AltaOrdenCobroCompletada{}, denegacionAltaCobro(domain.ErrContextoAutorizacionCobroInvalido)
	}

	alta := domain.AltaOrdenCobro{
		ExpedienteRef: datosLiquidacion.ExpedienteRef, SolicitudRef: datosLiquidacion.SolicitudRef,
		LiquidacionRef: datosLiquidacion.LiquidacionRef, Tarifa: datosLiquidacion.Tarifa,
		SujetoRef: datosLiquidacion.SujetoRef, RepresentacionRef: datosLiquidacion.RepresentacionRef,
		Importe: datosLiquidacion.Importe, Concepto: datosLiquidacion.Concepto,
		Finalidad: datosLiquidacion.Finalidad, CorrelacionRef: solicitud.CorrelacionRef,
		CreadaEn: ahora, CaducaEn: datosLiquidacion.ExigibleHasta,
		EvidenciaCreacionRef:  datosLiquidacion.LiquidacionRef,
		HuellaEvidenciaSHA256: datosLiquidacion.HuellaSHA256, Motivo: motivoAltaOrdenCobro,
	}
	bytesIdempotencia, err := domain.BytesCanonicosIdempotenciaAltaCobro(alta)
	if err != nil {
		return AltaOrdenCobroCompletada{}, denegacionAltaCobro(err)
	}
	indiceIdempotencia, err := s.sellador.SellarIndiceAltaCobro(ctx, bytesIdempotencia)
	if err != nil {
		return AltaOrdenCobroCompletada{}, errorPersistenciaAltaCobro(ctx, err)
	}
	if err := ctx.Err(); err != nil {
		return AltaOrdenCobroCompletada{}, errorPersistenciaAltaCobro(ctx, err)
	}
	alta.IndiceIdempotenciaHMAC = indiceIdempotencia
	bytesPeticion, err := bytesCanonicosPeticionAltaCobro(alta, actor, datosLiquidacion.HuellaSHA256)
	if err != nil {
		return AltaOrdenCobroCompletada{}, denegacionAltaCobro(err)
	}
	huellaPeticion, err := s.sellador.SellarHuellaPeticionCobro(ctx, bytesPeticion)
	if err != nil {
		return AltaOrdenCobroCompletada{}, errorPersistenciaAltaCobro(ctx, err)
	}
	if err := ctx.Err(); err != nil {
		return AltaOrdenCobroCompletada{}, errorPersistenciaAltaCobro(ctx, err)
	}
	ordenRef, err := s.generador.NuevoIDOrdenCobro()
	if err != nil || ctx.Err() != nil {
		return AltaOrdenCobroCompletada{}, errorPersistenciaAltaCobro(ctx, errors.Join(err, ctx.Err()))
	}
	alta.ID = ordenRef
	instanteReserva, err := s.instanteAltaCobroVigente(
		ahora, solicitud.VinculoAutenticacionActor, actor, liquidacion, autorizacion,
		datosLiquidacion.Finalidad, solicitud.CorrelacionRef,
	)
	if err != nil {
		return AltaOrdenCobroCompletada{}, denegacionAltaCobro(err)
	}
	alta.CreadaEn = instanteReserva

	reservaSolicitada := ports.SolicitudReservaOrdenCobro{
		OrdenRef: ordenRef, IndiceIdempotenciaHMAC: indiceIdempotencia,
		HuellaSolicitudHMAC: huellaPeticion, PrincipalRef: actor.PersonaRef,
		SolicitadaEn: instanteReserva, ExpiraEn: instanteReserva.Add(duracionReservaAltaCobro),
	}
	if reservaSolicitada.Validar() != nil {
		return AltaOrdenCobroCompletada{}, errorPersistenciaAltaCobro(ctx, ports.ErrReservaOrdenCobroInvalida)
	}
	reserva, err := s.repositorio.ReservarCreacion(ctx, reservaSolicitada)
	if err != nil {
		return AltaOrdenCobroCompletada{}, errorPersistenciaAltaCobro(ctx, err)
	}
	if reserva.Validar() != nil {
		return AltaOrdenCobroCompletada{}, errorPersistenciaAltaCobro(ctx, ports.ErrReservaOrdenCobroInvalida)
	}
	if err := ctx.Err(); err != nil {
		if reserva.Repetida {
			return AltaOrdenCobroCompletada{}, errorPersistenciaAltaCobro(ctx, err)
		}
		return AltaOrdenCobroCompletada{}, s.abandonarReserva(ctx, reserva.Token, reservaSolicitada, err)
	}
	instanteCreacion, err := s.instanteAltaCobroVigente(
		instanteReserva, solicitud.VinculoAutenticacionActor, actor, liquidacion, autorizacion,
		datosLiquidacion.Finalidad, solicitud.CorrelacionRef,
	)
	if err != nil || !instanteCreacion.Before(reservaSolicitada.ExpiraEn) {
		causa := err
		if causa == nil {
			causa = ports.ErrReservaOrdenCobroInvalida
		}
		if reserva.Repetida {
			return AltaOrdenCobroCompletada{}, denegacionAltaCobro(causa)
		}
		return AltaOrdenCobroCompletada{}, s.abandonarReserva(ctx, reserva.Token, reservaSolicitada, causa)
	}
	alta.CreadaEn = instanteCreacion
	if reserva.Repetida {
		return resultadoAltaCobroRepetida(reserva.Orden, alta, actor)
	}
	evidenciaAutorizacion, err := ports.NuevaEvidenciaUsoDecisionAutorizacion(
		decision,
		instanteCreacion,
	)
	if err != nil {
		return AltaOrdenCobroCompletada{}, s.abandonarReserva(
			ctx, reserva.Token, reservaSolicitada, err,
		)
	}

	orden, err := domain.NuevaOrdenCobro(alta, autorizacion)
	if err != nil {
		return AltaOrdenCobroCompletada{}, s.abandonarReserva(ctx, reserva.Token, reservaSolicitada, err)
	}
	mutacion, err := ports.NuevaMutacionOrdenCobro(orden)
	if err != nil {
		return AltaOrdenCobroCompletada{}, s.abandonarReserva(ctx, reserva.Token, reservaSolicitada, err)
	}
	_, huellaEfecto, err := orden.ControlConcurrencia()
	if err != nil {
		return AltaOrdenCobroCompletada{}, s.abandonarReserva(ctx, reserva.Token, reservaSolicitada, err)
	}
	confirmacion := ports.SolicitudConfirmarCreacionOrdenCobro{
		Token: reserva.Token, OrdenRef: ordenRef, PrincipalRef: actor.PersonaRef,
		IndiceIdempotenciaHMAC: indiceIdempotencia, HuellaSolicitudHMAC: huellaPeticion,
		ReservaSolicitadaEn: reservaSolicitada.SolicitadaEn, ReservaExpiraEn: reservaSolicitada.ExpiraEn,
		DecisionAutorizacionRef:  datosAutorizacion.DecisionRef,
		HuellaDecisionSHA256:     datosAutorizacion.HuellaDecisionSHA256,
		DecisionValidaHasta:      datosAutorizacion.VigenteHasta,
		HuellaEfectoSHA256:       huellaEfecto,
		EvidenciaAutorizacion:    evidenciaAutorizacion,
		ContextoAutorizacion:     autorizacion,
		SesionRef:                datosAutorizacion.SesionRef,
		HuellaSesionHMAC:         datosAutorizacion.HuellaSesionHMAC,
		SesionValidaHasta:        datosVinculo.SesionValidaHasta,
		LiquidacionRef:           datosLiquidacion.LiquidacionRef,
		LiquidacionRevision:      datosLiquidacion.Revision,
		LiquidacionHuellaSHA256:  datosLiquidacion.HuellaSHA256,
		LiquidacionEstado:        ports.EstadoControlLiquidacionCobro(datosLiquidacion.Estado),
		LiquidacionExigibleDesde: datosLiquidacion.ExigibleDesde,
		LiquidacionExigibleHasta: datosLiquidacion.ExigibleHasta,
		Mutacion:                 mutacion,
	}
	if confirmacion.Validar() != nil {
		return AltaOrdenCobroCompletada{}, s.abandonarReserva(
			ctx, reserva.Token, reservaSolicitada, ports.ErrMutacionOrdenCobroInvalida,
		)
	}
	if err := s.repositorio.ConfirmarCreacion(ctx, confirmacion); err != nil {
		return AltaOrdenCobroCompletada{}, s.abandonarReserva(ctx, reserva.Token, reservaSolicitada, err)
	}
	// ConfirmarCreacion es el punto atomico. Si devuelve nil, no se transforma
	// una cancelacion concurrente en falso fallo: el reintento encontraria la
	// misma orden por idempotencia.
	vista, err := orden.VistaTitular()
	if err != nil {
		return AltaOrdenCobroCompletada{}, errors.Join(ErrResultadoAltaCobroNoConfiable, err)
	}
	return nuevaAltaCobroCompletada(vista, false)
}

func (s *ServicioAltaOrdenCobro) instanteAltaCobroVigente(
	noAntes time.Time,
	vinculo domain.VinculoAutenticacionActorV1,
	actor domain.ContextoActor,
	liquidacion LiquidacionCobroAutoritativa,
	autorizacion domain.ContextoAutorizacionCobro,
	finalidad, correlacion string,
) (time.Time, error) {
	instante := s.reloj.Ahora().UTC().Truncate(time.Microsecond)
	datosLiquidacion, err := liquidacion.Datos()
	if err != nil || !instanteAplicacionCobroCanonico(instante) || instante.Before(noAntes) ||
		!vinculo.VigenteEn(instante, actor) || !liquidacion.exigibleEn(instante) ||
		autorizacion.ValidarEn(
			domain.AccionCobroCrearOrden, datosLiquidacion.LiquidacionRef,
			finalidad, correlacion, instante,
		) != nil {
		return time.Time{}, domain.ErrContextoAutorizacionCobroInvalido
	}
	return instante, nil
}

func validarSolicitudAltaCobro(solicitud SolicitudAltaOrdenCobro) error {
	if solicitud.ContextoActor.Validar() != nil ||
		solicitud.VinculoAutenticacionActor.ValidarPara(solicitud.ContextoActor) != nil ||
		!referenciaAplicacionCobroValida(solicitud.SesionRef) ||
		!huellaSesionAplicacionCobroValida(solicitud.HuellaSesionHMAC) ||
		!referenciaAplicacionCobroValida(solicitud.LiquidacionRef) ||
		!referenciaAplicacionCobroValida(solicitud.CorrelacionRef) {
		return ErrSolicitudAltaCobroInvalida
	}
	return nil
}

func recursoAutorizableLiquidacionCobro(
	datos DatosLiquidacionCobroAutoritativa,
) (domain.RecursoAutorizable, error) {
	recurso := domain.RecursoAutorizable{
		Referencia: datos.LiquidacionRef, ModuloID: "pagos", Tipo: "orden_cobro",
		Ambitos: map[string]string{"sujeto_ref": datos.SujetoRef},
		Atributos: map[string]string{
			"estado_liquidacion":        string(datos.Estado),
			"liquidacion_revision":      strconv.FormatUint(datos.Revision, 10),
			"liquidacion_huella_sha256": datos.HuellaSHA256,
			"tarifa_ref":                datos.Tarifa.Referencia(),
			"tarifa_huella_sha256":      datos.Tarifa.HuellaSHA256,
		},
	}
	if err := recurso.Validar(); err != nil {
		return domain.RecursoAutorizable{}, ErrLiquidacionCobroNoConfiable
	}
	return recurso, nil
}

func clonarRecursoAutorizableAltaCobro(recurso domain.RecursoAutorizable) domain.RecursoAutorizable {
	copia := recurso
	copia.Ambitos = make(map[string]string, len(recurso.Ambitos))
	for clave, valor := range recurso.Ambitos {
		copia.Ambitos[clave] = valor
	}
	copia.Atributos = make(map[string]string, len(recurso.Atributos))
	for clave, valor := range recurso.Atributos {
		copia.Atributos[clave] = valor
	}
	return copia
}

func clonarDecisionAutorizacionAltaCobro(
	decision domain.DecisionAutorizacion,
) domain.DecisionAutorizacion {
	copia := decision
	copia.PoliticasEvaluadasRefs = append([]string(nil), decision.PoliticasEvaluadasRefs...)
	copia.PoliticasRefs = append([]string(nil), decision.PoliticasRefs...)
	copia.CamposPermitidos = append([]string(nil), decision.CamposPermitidos...)
	copia.Obligaciones = append([]string(nil), decision.Obligaciones...)
	copia.PoliticasEvaluadasHuellasSHA256 = clonarMapaAltaCobro(decision.PoliticasEvaluadasHuellasSHA256)
	copia.PoliticasHuellasSHA256 = clonarMapaAltaCobro(decision.PoliticasHuellasSHA256)
	// La evidencia de uso canoniza estos conjuntos. Se aplica la misma
	// representacion antes de crear el contexto opaco para que ambos puedan
	// ligarse mediante el predicado del dominio, nunca por campos parciales.
	sort.Strings(copia.PoliticasEvaluadasRefs)
	sort.Strings(copia.PoliticasRefs)
	sort.Strings(copia.CamposPermitidos)
	sort.Strings(copia.Obligaciones)
	return copia
}

func clonarMapaAltaCobro(origen map[string]string) map[string]string {
	if origen == nil {
		return nil
	}
	copia := make(map[string]string, len(origen))
	for clave, valor := range origen {
		copia[clave] = valor
	}
	return copia
}

type peticionAltaCobroCanonica struct {
	Version                 int
	IndiceIdempotenciaHMAC  string
	LiquidacionHuellaSHA256 string
	PrincipalRef            string
	PerfilActivoRef         string
	CorrelacionRef          string
}

func bytesCanonicosPeticionAltaCobro(
	alta domain.AltaOrdenCobro,
	actor domain.ContextoActor,
	huellaLiquidacion string,
) ([]byte, error) {
	if actor.Validar() != nil || !referenciaAplicacionCobroValida(alta.CorrelacionRef) ||
		!huellaSHA256AplicacionCobroValida(huellaLiquidacion) ||
		!huellaHMACAplicacionCobroValida(alta.IndiceIdempotenciaHMAC, "pagos-v1") {
		return nil, ErrSolicitudAltaCobroInvalida
	}
	contenido, err := json.Marshal(peticionAltaCobroCanonica{
		Version:                 versionInstantaneaLiquidacionCobro,
		IndiceIdempotenciaHMAC:  alta.IndiceIdempotenciaHMAC,
		LiquidacionHuellaSHA256: huellaLiquidacion,
		PrincipalRef:            actor.PersonaRef, PerfilActivoRef: actor.PerfilActivoRef,
		CorrelacionRef: alta.CorrelacionRef,
	})
	if err != nil {
		return nil, ErrSolicitudAltaCobroInvalida
	}
	return append([]byte(nil), contenido...), nil
}

func resultadoAltaCobroRepetida(
	orden *domain.OrdenCobro,
	alta domain.AltaOrdenCobro,
	actor domain.ContextoActor,
) (AltaOrdenCobroCompletada, error) {
	if orden == nil || orden.Validar() != nil || len(orden.Historial) == 0 {
		return AltaOrdenCobroCompletada{}, ErrResultadoAltaCobroNoConfiable
	}
	primero := orden.Historial[0]
	if orden.IndiceIdempotenciaHMAC != alta.IndiceIdempotenciaHMAC ||
		orden.ExpedienteRef != alta.ExpedienteRef || orden.SolicitudRef != alta.SolicitudRef ||
		orden.LiquidacionRef != alta.LiquidacionRef || orden.Tarifa != alta.Tarifa ||
		orden.SujetoRef != alta.SujetoRef || orden.RepresentacionRef != alta.RepresentacionRef ||
		!orden.Importe.Igual(alta.Importe) || orden.Concepto != alta.Concepto ||
		orden.Finalidad != alta.Finalidad || orden.CorrelacionRef != alta.CorrelacionRef ||
		!orden.CaducaEn.Equal(alta.CaducaEn) ||
		primero.EvidenciaRef != alta.EvidenciaCreacionRef ||
		primero.HuellaEvidenciaSHA256 != alta.HuellaEvidenciaSHA256 ||
		primero.ActorRef != actor.PersonaRef || primero.PerfilActivoRef != actor.PerfilActivoRef {
		return AltaOrdenCobroCompletada{}, ErrResultadoAltaCobroNoConfiable
	}
	vista, err := orden.VistaTitular()
	if err != nil {
		return AltaOrdenCobroCompletada{}, ErrResultadoAltaCobroNoConfiable
	}
	return nuevaAltaCobroCompletada(vista, true)
}

func nuevaAltaCobroCompletada(
	vista domain.VistaTitularOrdenCobro,
	repetida bool,
) (AltaOrdenCobroCompletada, error) {
	resultado := AltaOrdenCobroCompletada{datos: &DatosAltaOrdenCobroCompletada{
		Vista: vista, Repetida: repetida,
	}}
	if _, err := resultado.Datos(); err != nil {
		return AltaOrdenCobroCompletada{}, err
	}
	return resultado, nil
}

func (s *ServicioAltaOrdenCobro) abandonarReserva(
	ctx context.Context,
	token ports.TokenReservaOrdenCobro,
	reserva ports.SolicitudReservaOrdenCobro,
	causa error,
) error {
	limpieza, cancelar := context.WithTimeout(context.WithoutCancel(ctx), duracionLimpiezaReservaAltaCobro)
	defer cancelar()
	err := s.repositorio.AbandonarReservaCreacion(limpieza, ports.SolicitudAbandonarReservaOrdenCobro{
		Token: token, OrdenRef: reserva.OrdenRef, PrincipalRef: reserva.PrincipalRef,
		HuellaSolicitudHMAC: reserva.HuellaSolicitudHMAC,
	})
	if err != nil {
		return errors.Join(ErrPersistenciaAltaCobroIncompleta, causa, err)
	}
	return errors.Join(ErrPersistenciaAltaCobroIncompleta, causa)
}

func denegacionAltaCobro(causa error) error {
	return errors.Join(domain.ErrAutorizacionDenegada, ErrSolicitudAltaCobroInvalida, causa)
}

func errorPersistenciaAltaCobro(ctx context.Context, causa error) error {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return errors.Join(ErrPersistenciaAltaCobroIncompleta, causa, err)
		}
	}
	return errors.Join(ErrPersistenciaAltaCobroIncompleta, causa)
}

func dependenciaAltaCobroNula(dependencia any) bool {
	if dependencia == nil {
		return true
	}
	valor := reflect.ValueOf(dependencia)
	switch valor.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return valor.IsNil()
	default:
		return false
	}
}

func instanteAplicacionCobroCanonico(instante time.Time) bool {
	return !instante.IsZero() && instante.Location() == time.UTC && instante.Equal(instante.Truncate(time.Microsecond))
}

func referenciaAplicacionCobroValida(valor string) bool {
	if valor == "" || valor != strings.TrimSpace(valor) {
		return false
	}
	return (domain.RecursoAutorizable{
		Referencia: valor, ModuloID: "pagos", Tipo: "referencia",
	}).Validar() == nil
}

func huellaSHA256AplicacionCobroValida(valor string) bool {
	if len(valor) != 64 {
		return false
	}
	_, err := hex.DecodeString(valor)
	return err == nil && valor == strings.ToLower(valor)
}

func huellaHMACAplicacionCobroValida(valor, dominio string) bool {
	partes := strings.Split(valor, ":")
	return len(partes) == 3 && partes[0] == "hmac-sha256" && partes[1] == dominio &&
		huellaSHA256AplicacionCobroValida(partes[2])
}

func huellaSesionAplicacionCobroValida(valor string) bool {
	return huellaHMACAplicacionCobroValida(valor, "sesion-v1")
}
