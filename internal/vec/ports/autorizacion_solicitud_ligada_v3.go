package ports

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

var (
	ErrOrdenRegistroAutorizacionLigadaV3Invalida = errors.New(
		"vec: orden de registro de autorizacion ligada V3 invalida",
	)
	ErrConfirmacionRegistroConcesionAutorizacionLigadaV3Invalida = errors.New(
		"vec: confirmacion de registro de concesion de autorizacion ligada V3 invalida",
	)
	ErrRegistroConcesionAutorizacionLigadaV3NoDisponible = errors.New(
		"vec: registro de concesiones de autorizacion ligada V3 no disponible",
	)
	ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible = errors.New(
		"vec: registro de denegaciones de autorizacion ligada V3 no disponible",
	)
	ErrSerializacionRegistroAutorizacionLigadaV3Prohibida = errors.New(
		"vec: serializacion de registro de autorizacion ligada V3 prohibida",
	)
)

// DatosOrdenRegistroAutorizacionLigadaV3 es la entrega defensiva y deliberada
// al adaptador durable. Conserva el resultado registrado completo porque la
// proyeccion minimizada incluida en la decision no permite revalidar por si
// sola la procedencia, las versiones ni la vigencia del contexto de actor.
type DatosOrdenRegistroAutorizacionLigadaV3 struct {
	bloqueoSerializacionRegistroAutorizacionLigadaV3
	Solicitud         domain.SolicitudAutorizacionLigadaV3
	Decision          domain.DecisionAutorizacionLigadaV3
	ReferenciaMotivo  domain.ReferenciaEntradaCatalogo
	ResultadoContexto domain.ResultadoContextoActorRegistradoV2
}

type datosOrdenRegistroAutorizacionLigadaV3 struct {
	solicitud         domain.SolicitudAutorizacionLigadaV3
	decision          domain.DecisionAutorizacionLigadaV3
	referenciaMotivo  domain.ReferenciaEntradaCatalogo
	resultadoContexto domain.ResultadoContextoActorRegistradoV2
}

// OrdenRegistroConcesionCandidataAutorizacionLigadaV3 nunca es una capacidad
// ejecutable. Solo solicita al adaptador el CAS y registro durable de una
// concesion evaluada en memoria.
type OrdenRegistroConcesionCandidataAutorizacionLigadaV3 struct {
	bloqueoSerializacionRegistroAutorizacionLigadaV3
	datos *datosOrdenRegistroAutorizacionLigadaV3
}

// OrdenRegistroDenegacionAutorizacionLigadaV3 tiene identidad nominal propia:
// un adaptador de denegaciones no puede recibir una concesion por accidente.
type OrdenRegistroDenegacionAutorizacionLigadaV3 struct {
	bloqueoSerializacionRegistroAutorizacionLigadaV3
	datos *datosOrdenRegistroAutorizacionLigadaV3
}

func NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(
	solicitud domain.SolicitudAutorizacionLigadaV3,
	decision domain.DecisionAutorizacionLigadaV3,
	referenciaMotivo domain.ReferenciaEntradaCatalogo,
	resultadoContexto domain.ResultadoContextoActorRegistradoV2,
) (OrdenRegistroConcesionCandidataAutorizacionLigadaV3, error) {
	datos, err := nuevosDatosOrdenRegistroAutorizacionLigadaV3(
		solicitud, decision, referenciaMotivo, resultadoContexto, true,
	)
	if err != nil {
		return OrdenRegistroConcesionCandidataAutorizacionLigadaV3{}, err
	}
	return OrdenRegistroConcesionCandidataAutorizacionLigadaV3{datos: datos}, nil
}

func NuevaOrdenRegistroDenegacionAutorizacionLigadaV3(
	solicitud domain.SolicitudAutorizacionLigadaV3,
	decision domain.DecisionAutorizacionLigadaV3,
	referenciaMotivo domain.ReferenciaEntradaCatalogo,
	resultadoContexto domain.ResultadoContextoActorRegistradoV2,
) (OrdenRegistroDenegacionAutorizacionLigadaV3, error) {
	datos, err := nuevosDatosOrdenRegistroAutorizacionLigadaV3(
		solicitud, decision, referenciaMotivo, resultadoContexto, false,
	)
	if err != nil {
		return OrdenRegistroDenegacionAutorizacionLigadaV3{}, err
	}
	return OrdenRegistroDenegacionAutorizacionLigadaV3{datos: datos}, nil
}

func (o OrdenRegistroConcesionCandidataAutorizacionLigadaV3) Datos() (
	DatosOrdenRegistroAutorizacionLigadaV3,
	error,
) {
	return copiarDatosOrdenRegistroAutorizacionLigadaV3(o.datos, true)
}

func (o OrdenRegistroDenegacionAutorizacionLigadaV3) Datos() (
	DatosOrdenRegistroAutorizacionLigadaV3,
	error,
) {
	return copiarDatosOrdenRegistroAutorizacionLigadaV3(o.datos, false)
}

func nuevosDatosOrdenRegistroAutorizacionLigadaV3(
	solicitud domain.SolicitudAutorizacionLigadaV3,
	decision domain.DecisionAutorizacionLigadaV3,
	referenciaMotivo domain.ReferenciaEntradaCatalogo,
	resultadoContexto domain.ResultadoContextoActorRegistradoV2,
	concedidaEsperada bool,
) (*datosOrdenRegistroAutorizacionLigadaV3, error) {
	datos := &datosOrdenRegistroAutorizacionLigadaV3{
		solicitud: solicitud, decision: decision, referenciaMotivo: referenciaMotivo,
		resultadoContexto: resultadoContexto,
	}
	copia, err := copiarDatosOrdenRegistroAutorizacionLigadaV3(datos, concedidaEsperada)
	if err != nil {
		return nil, err
	}
	return &datosOrdenRegistroAutorizacionLigadaV3{
		solicitud: copia.Solicitud, decision: copia.Decision,
		referenciaMotivo: copia.ReferenciaMotivo, resultadoContexto: copia.ResultadoContexto,
	}, nil
}

func copiarDatosOrdenRegistroAutorizacionLigadaV3(
	datos *datosOrdenRegistroAutorizacionLigadaV3,
	concedidaEsperada bool,
) (DatosOrdenRegistroAutorizacionLigadaV3, error) {
	if datos == nil || validarDatosOrdenRegistroAutorizacionLigadaV3(datos, concedidaEsperada) != nil {
		return DatosOrdenRegistroAutorizacionLigadaV3{},
			ErrOrdenRegistroAutorizacionLigadaV3Invalida
	}
	resultado, err := datos.resultadoContexto.Clonar()
	if err != nil {
		return DatosOrdenRegistroAutorizacionLigadaV3{},
			ErrOrdenRegistroAutorizacionLigadaV3Invalida
	}
	return DatosOrdenRegistroAutorizacionLigadaV3{
		Solicitud: datos.solicitud, Decision: datos.decision,
		ReferenciaMotivo: datos.referenciaMotivo, ResultadoContexto: resultado,
	}, nil
}

func validarDatosOrdenRegistroAutorizacionLigadaV3(
	datos *datosOrdenRegistroAutorizacionLigadaV3,
	concedidaEsperada bool,
) error {
	if datos == nil || datos.decision.ValidarPara(datos.solicitud) != nil ||
		datos.resultadoContexto.Validar() != nil ||
		!domain.ReferenciaMotivoAutorizacionV2Valida(datos.referenciaMotivo) {
		return ErrOrdenRegistroAutorizacionLigadaV3Invalida
	}
	solicitud, err := datos.solicitud.Datos()
	if err != nil || solicitud.ReferenciaMotivo != datos.referenciaMotivo ||
		solicitud.VinculoAutenticacionActor.ValidarPara(datos.resultadoContexto) != nil {
		return ErrOrdenRegistroAutorizacionLigadaV3Invalida
	}
	concedida, _, err := datos.decision.Resultado()
	emitidaEn, _, errVentana := datos.decision.VentanaValidez()
	if err != nil || errVentana != nil || concedida != concedidaEsperada ||
		!solicitud.VinculoAutenticacionActor.VigenteEn(emitidaEn, datos.resultadoContexto) {
		return ErrOrdenRegistroAutorizacionLigadaV3Invalida
	}
	return nil
}

// DatosConfirmacionRegistroConcesionAutorizacionLigadaV3 omite identidad,
// recurso, motivo y contexto. Solo expone el minimo necesario para consumir
// la concesion confirmada durante su ventana half-open.
type DatosConfirmacionRegistroConcesionAutorizacionLigadaV3 struct {
	bloqueoSerializacionRegistroAutorizacionLigadaV3
	DecisionRef          string
	DecisionHuellaSHA256 string
	EmitidaEn            time.Time
	ValidaHasta          time.Time
	RegistradaEn         time.Time
}

type datosConfirmacionRegistroConcesionAutorizacionLigadaV3 struct {
	decisionRef          string
	decisionHuellaSHA256 string
	emitidaEn            time.Time
	validaHasta          time.Time
	registradaEn         time.Time
}

// ConfirmacionRegistroConcesionAutorizacionLigadaV3 es la respuesta nominal que
// este paquete fabrica cuando el adaptador retorna despues del COMMIT/CAS. El
// tipo prueba ligadura e integridad estructural, no constituye por si solo una
// prueba criptografica de I/O durable. El adaptador de registro forma parte de
// la TCB y cualquier consumidor con efecto debe cotejar/consumir DecisionRef y
// huella en su propia transaccion autoritativa.
type ConfirmacionRegistroConcesionAutorizacionLigadaV3 struct {
	bloqueoSerializacionRegistroAutorizacionLigadaV3
	datos *datosConfirmacionRegistroConcesionAutorizacionLigadaV3
}

func nuevaConfirmacionRegistroConcesionAutorizacionLigadaV3(
	orden OrdenRegistroConcesionCandidataAutorizacionLigadaV3,
	registradaEn time.Time,
) (ConfirmacionRegistroConcesionAutorizacionLigadaV3, error) {
	datosOrden, err := orden.Datos()
	if err != nil {
		return ConfirmacionRegistroConcesionAutorizacionLigadaV3{},
			ErrConfirmacionRegistroConcesionAutorizacionLigadaV3Invalida
	}
	resumen, err := resumenDecisionAutorizacionLigadaV3(datosOrden.Decision)
	if err != nil || !instanteRegistroAutorizacionLigadaV3Canonico(registradaEn) ||
		registradaEn.Before(resumen.EmitidaEn) || !registradaEn.Before(resumen.ValidaHasta) {
		return ConfirmacionRegistroConcesionAutorizacionLigadaV3{},
			ErrConfirmacionRegistroConcesionAutorizacionLigadaV3Invalida
	}
	huella, err := domain.HuellaSHA256DecisionAutorizacionV3(datosOrden.Decision)
	if err != nil {
		return ConfirmacionRegistroConcesionAutorizacionLigadaV3{},
			ErrConfirmacionRegistroConcesionAutorizacionLigadaV3Invalida
	}
	confirmacion := ConfirmacionRegistroConcesionAutorizacionLigadaV3{
		datos: &datosConfirmacionRegistroConcesionAutorizacionLigadaV3{
			decisionRef: resumen.DecisionRef, decisionHuellaSHA256: huella,
			emitidaEn: resumen.EmitidaEn, validaHasta: resumen.ValidaHasta, registradaEn: registradaEn,
		},
	}
	if confirmacion.ValidarPara(orden) != nil {
		return ConfirmacionRegistroConcesionAutorizacionLigadaV3{},
			ErrConfirmacionRegistroConcesionAutorizacionLigadaV3Invalida
	}
	return confirmacion, nil
}

func (c ConfirmacionRegistroConcesionAutorizacionLigadaV3) Datos() (
	DatosConfirmacionRegistroConcesionAutorizacionLigadaV3,
	error,
) {
	if c.Validar() != nil {
		return DatosConfirmacionRegistroConcesionAutorizacionLigadaV3{},
			ErrConfirmacionRegistroConcesionAutorizacionLigadaV3Invalida
	}
	return DatosConfirmacionRegistroConcesionAutorizacionLigadaV3{
		DecisionRef: c.datos.decisionRef, DecisionHuellaSHA256: c.datos.decisionHuellaSHA256,
		EmitidaEn: c.datos.emitidaEn, ValidaHasta: c.datos.validaHasta, RegistradaEn: c.datos.registradaEn,
	}, nil
}

func (c ConfirmacionRegistroConcesionAutorizacionLigadaV3) Validar() error {
	if c.datos == nil || !referenciaDecisionAutorizacionLigadaV3Valida(c.datos.decisionRef) ||
		!huellaSHA256RegistroAutorizacionLigadaV3Valida(c.datos.decisionHuellaSHA256) ||
		!instanteRegistroAutorizacionLigadaV3Canonico(c.datos.emitidaEn) ||
		!instanteRegistroAutorizacionLigadaV3Canonico(c.datos.validaHasta) ||
		!instanteRegistroAutorizacionLigadaV3Canonico(c.datos.registradaEn) ||
		!c.datos.validaHasta.After(c.datos.emitidaEn) ||
		c.datos.registradaEn.Before(c.datos.emitidaEn) ||
		!c.datos.registradaEn.Before(c.datos.validaHasta) {
		return ErrConfirmacionRegistroConcesionAutorizacionLigadaV3Invalida
	}
	return nil
}

func (c ConfirmacionRegistroConcesionAutorizacionLigadaV3) ValidarPara(
	orden OrdenRegistroConcesionCandidataAutorizacionLigadaV3,
) error {
	if c.Validar() != nil {
		return ErrConfirmacionRegistroConcesionAutorizacionLigadaV3Invalida
	}
	datosOrden, err := orden.Datos()
	if err != nil {
		return ErrConfirmacionRegistroConcesionAutorizacionLigadaV3Invalida
	}
	resumen, err := resumenDecisionAutorizacionLigadaV3(datosOrden.Decision)
	huella, errHuella := domain.HuellaSHA256DecisionAutorizacionV3(datosOrden.Decision)
	if err != nil || errHuella != nil || c.datos.decisionRef != resumen.DecisionRef ||
		c.datos.decisionHuellaSHA256 != huella || !c.datos.emitidaEn.Equal(resumen.EmitidaEn) ||
		!c.datos.validaHasta.Equal(resumen.ValidaHasta) {
		return ErrConfirmacionRegistroConcesionAutorizacionLigadaV3Invalida
	}
	return nil
}

// DentroDeVentanaEn solo comprueba la ventana y la integridad local. No hace
// ejecutable la concesion, no evita replay y no sustituye el consumo/cotejo
// autoritativo de DecisionRef y huella en la transaccion con efecto.
func (c ConfirmacionRegistroConcesionAutorizacionLigadaV3) DentroDeVentanaEn(
	instante time.Time,
) bool {
	return c.Validar() == nil && instanteRegistroAutorizacionLigadaV3Canonico(instante) &&
		!instante.Before(c.datos.registradaEn) && !instante.Before(c.datos.emitidaEn) &&
		instante.Before(c.datos.validaHasta)
}

type resumenDecisionAutorizacionLigadaV3Canonico struct {
	DecisionRef string `json:"decision_ref"`
	Concedida   bool   `json:"concedida"`
	EmitidaEn   string `json:"emitida_en"`
	ValidaHasta string `json:"valida_hasta"`
}

type resumenDecisionAutorizacionLigadaV3Datos struct {
	DecisionRef string
	EmitidaEn   time.Time
	ValidaHasta time.Time
}

func resumenDecisionAutorizacionLigadaV3(
	decision domain.DecisionAutorizacionLigadaV3,
) (resumenDecisionAutorizacionLigadaV3Datos, error) {
	canon, err := domain.RepresentacionCanonicaDecisionAutorizacionV3(decision)
	if err != nil {
		return resumenDecisionAutorizacionLigadaV3Datos{}, err
	}
	var dto resumenDecisionAutorizacionLigadaV3Canonico
	if err := json.Unmarshal(canon, &dto); err != nil || !dto.Concedida {
		return resumenDecisionAutorizacionLigadaV3Datos{},
			ErrConfirmacionRegistroConcesionAutorizacionLigadaV3Invalida
	}
	emitidaEn, errEmitida := time.Parse("2006-01-02T15:04:05.000000Z", dto.EmitidaEn)
	validaHasta, errValida := time.Parse("2006-01-02T15:04:05.000000Z", dto.ValidaHasta)
	if errEmitida != nil || errValida != nil ||
		!referenciaDecisionAutorizacionLigadaV3Valida(dto.DecisionRef) {
		return resumenDecisionAutorizacionLigadaV3Datos{},
			ErrConfirmacionRegistroConcesionAutorizacionLigadaV3Invalida
	}
	return resumenDecisionAutorizacionLigadaV3Datos{
		DecisionRef: dto.DecisionRef, EmitidaEn: emitidaEn, ValidaHasta: validaHasta,
	}, nil
}

// RegistroConcesionesCandidatasAutorizacionLigadaV3 debe ejecutar CAS e
// insercion en una unica transaccion durable. Su instante de registro solo es
// un dato del resultado: no es una capacidad y no permite construir una
// confirmacion fuera de este paquete. El metodo solo puede retornar nil tras
// COMMIT; el adaptador forma parte de la TCB de persistencia.
type RegistroConcesionesCandidatasAutorizacionLigadaV3 interface {
	RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(
		context.Context,
		OrdenRegistroConcesionCandidataAutorizacionLigadaV3,
	) (time.Time, error)
}

// RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente es la
// unica fabrica de confirmaciones. Primero entrega la orden al adaptador y solo
// construye el valor nominal cuando este ha retornado exito, que por contrato
// sucede despues del COMMIT/CAS durable. Una implementacion deshonesta del
// puerto sigue pudiendo mentir sobre I/O; ningun tipo local prueba persistencia.
func RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(
	ctx context.Context,
	registro RegistroConcesionesCandidatasAutorizacionLigadaV3,
	orden OrdenRegistroConcesionCandidataAutorizacionLigadaV3,
) (ConfirmacionRegistroConcesionAutorizacionLigadaV3, error) {
	if ctx == nil || dependenciaRegistroAutorizacionLigadaV3Nula(registro) {
		return ConfirmacionRegistroConcesionAutorizacionLigadaV3{},
			ErrRegistroConcesionAutorizacionLigadaV3NoDisponible
	}
	if err := ctx.Err(); err != nil {
		return ConfirmacionRegistroConcesionAutorizacionLigadaV3{},
			nuevoErrorRegistroConcesionAutorizacionLigadaV3(err, nil)
	}
	registradaEn, err := registro.
		RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(ctx, orden)
	if err != nil {
		return ConfirmacionRegistroConcesionAutorizacionLigadaV3{},
			nuevoErrorRegistroConcesionAutorizacionLigadaV3(err, ctx.Err())
	}
	// No se consulta ctx.Err tras el retorno: un COMMIT ya confirmado no se
	// convierte retroactivamente en fallo ambiguo por una cancelacion tardia.
	return nuevaConfirmacionRegistroConcesionAutorizacionLigadaV3(orden, registradaEn)
}

type errorRegistroConcesionAutorizacionLigadaV3 struct{ causas []error }

func (e errorRegistroConcesionAutorizacionLigadaV3) Error() string {
	return ErrRegistroConcesionAutorizacionLigadaV3NoDisponible.Error()
}

func (e errorRegistroConcesionAutorizacionLigadaV3) Unwrap() []error {
	return append([]error(nil), e.causas...)
}

func (e errorRegistroConcesionAutorizacionLigadaV3) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, e.Error())
}

func (e errorRegistroConcesionAutorizacionLigadaV3) LogValue() slog.Value {
	return slog.StringValue(e.Error())
}

func nuevoErrorRegistroConcesionAutorizacionLigadaV3(causas ...error) error {
	filtradas := []error{ErrRegistroConcesionAutorizacionLigadaV3NoDisponible}
	for _, causa := range causas {
		if causa != nil {
			filtradas = append(filtradas, causa)
		}
	}
	return errorRegistroConcesionAutorizacionLigadaV3{causas: filtradas}
}

func dependenciaRegistroAutorizacionLigadaV3Nula(valor any) bool {
	if valor == nil {
		return true
	}
	reflejo := reflect.ValueOf(valor)
	switch reflejo.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflejo.IsNil()
	default:
		return false
	}
}

// RegistroDenegacionesAutorizacionLigadaV3 es append-only y no concede ni
// confirma capacidades. Su fallo no cambia el resultado negativo.
type RegistroDenegacionesAutorizacionLigadaV3 interface {
	RegistrarDenegacionAutorizacionLigadaV3(
		context.Context,
		OrdenRegistroDenegacionAutorizacionLigadaV3,
	) error
}

// AutorizadorSolicitudLigadaV3 exige conjuntamente solicitud y recibo durable
// completo. La decision devuelta documenta el resultado y la confirmacion es
// solo un handle opaco para cotejo/consumo autoritativo; ninguno de los dos
// valores constituye aisladamente una capacidad ejecutable.
type AutorizadorSolicitudLigadaV3 interface {
	ExigirSolicitudLigadaV3(
		context.Context,
		domain.SolicitudAutorizacionLigadaV3,
		domain.ResultadoContextoActorRegistradoV2,
	) (
		domain.DecisionAutorizacionLigadaV3,
		ConfirmacionRegistroConcesionAutorizacionLigadaV3,
		error,
	)
}

func instanteRegistroAutorizacionLigadaV3Canonico(instante time.Time) bool {
	return !instante.IsZero() && instante.Location() == time.UTC &&
		instante.Year() >= 1 && instante.Year() <= 9999 && instante.Nanosecond()%1_000 == 0
}

func referenciaDecisionAutorizacionLigadaV3Valida(valor string) bool {
	if valor == "" || valor != strings.TrimSpace(valor) || len(valor) > 512 {
		return false
	}
	for _, caracter := range valor {
		if caracter < 0x21 || caracter > 0x7e || caracter == '*' {
			return false
		}
	}
	return true
}

func huellaSHA256RegistroAutorizacionLigadaV3Valida(valor string) bool {
	if len(valor) != 64 || valor == strings.Repeat("0", 64) {
		return false
	}
	for _, caracter := range valor {
		if (caracter < '0' || caracter > '9') && (caracter < 'a' || caracter > 'f') {
			return false
		}
	}
	return true
}

type bloqueoSerializacionRegistroAutorizacionLigadaV3 struct{}

func (bloqueoSerializacionRegistroAutorizacionLigadaV3) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionRegistroAutorizacionLigadaV3Prohibida
}
func (*bloqueoSerializacionRegistroAutorizacionLigadaV3) UnmarshalJSON([]byte) error {
	return ErrSerializacionRegistroAutorizacionLigadaV3Prohibida
}
func (bloqueoSerializacionRegistroAutorizacionLigadaV3) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionRegistroAutorizacionLigadaV3Prohibida
}
func (*bloqueoSerializacionRegistroAutorizacionLigadaV3) UnmarshalText([]byte) error {
	return ErrSerializacionRegistroAutorizacionLigadaV3Prohibida
}
func (bloqueoSerializacionRegistroAutorizacionLigadaV3) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionRegistroAutorizacionLigadaV3Prohibida
}
func (*bloqueoSerializacionRegistroAutorizacionLigadaV3) UnmarshalBinary([]byte) error {
	return ErrSerializacionRegistroAutorizacionLigadaV3Prohibida
}
func (bloqueoSerializacionRegistroAutorizacionLigadaV3) GobEncode() ([]byte, error) {
	return nil, ErrSerializacionRegistroAutorizacionLigadaV3Prohibida
}
func (*bloqueoSerializacionRegistroAutorizacionLigadaV3) GobDecode([]byte) error {
	return ErrSerializacionRegistroAutorizacionLigadaV3Prohibida
}
func (bloqueoSerializacionRegistroAutorizacionLigadaV3) MarshalCBOR() ([]byte, error) {
	return nil, ErrSerializacionRegistroAutorizacionLigadaV3Prohibida
}
func (*bloqueoSerializacionRegistroAutorizacionLigadaV3) UnmarshalCBOR([]byte) error {
	return ErrSerializacionRegistroAutorizacionLigadaV3Prohibida
}
func (bloqueoSerializacionRegistroAutorizacionLigadaV3) MarshalYAML() (any, error) {
	return nil, ErrSerializacionRegistroAutorizacionLigadaV3Prohibida
}
func (*bloqueoSerializacionRegistroAutorizacionLigadaV3) UnmarshalYAML(func(any) error) error {
	return ErrSerializacionRegistroAutorizacionLigadaV3Prohibida
}
func (bloqueoSerializacionRegistroAutorizacionLigadaV3) MarshalXML(*xml.Encoder, xml.StartElement) error {
	return ErrSerializacionRegistroAutorizacionLigadaV3Prohibida
}
func (*bloqueoSerializacionRegistroAutorizacionLigadaV3) UnmarshalXML(*xml.Decoder, xml.StartElement) error {
	return ErrSerializacionRegistroAutorizacionLigadaV3Prohibida
}
func (bloqueoSerializacionRegistroAutorizacionLigadaV3) String() string {
	return "[REGISTRO-AUTORIZACION-LIGADA-V3-OPACO]"
}
func (b bloqueoSerializacionRegistroAutorizacionLigadaV3) GoString() string { return b.String() }
func (b bloqueoSerializacionRegistroAutorizacionLigadaV3) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, b.String())
}
func (b bloqueoSerializacionRegistroAutorizacionLigadaV3) LogValue() slog.Value {
	return slog.StringValue(b.String())
}
