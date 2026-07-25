package ports

import (
	"context"
	"crypto/subtle"
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
	ErrCandidataRegistroDecisionAutorizacionLigadaV3Invalida = errors.New(
		"vec: candidata de registro de decision de autorizacion ligada V3 invalida",
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

// CandidataRegistroDecisionAutorizacionLigadaV3 es una union nominal opaca.
// Contiene exactamente una orden de concesion o de denegacion y permite que un
// adaptador compuesto registre cualquiera de los dos resultados dentro de su
// propia transaccion autoritativa. No registra, confirma ni concede por si sola.
type CandidataRegistroDecisionAutorizacionLigadaV3 struct {
	bloqueoSerializacionRegistroAutorizacionLigadaV3
	concedida  bool
	concesion  OrdenRegistroConcesionCandidataAutorizacionLigadaV3
	denegacion OrdenRegistroDenegacionAutorizacionLigadaV3
}

// DatosResumenCandidataRegistroDecisionAutorizacionLigadaV3 es la vista
// minimizada de una candidata ya verificada. No contiene identidad, recurso,
// motivo, políticas ni contexto de actor. Sigue siendo opaca a codecs y logs.
type DatosResumenCandidataRegistroDecisionAutorizacionLigadaV3 struct {
	bloqueoSerializacionRegistroAutorizacionLigadaV3
	DecisionRef          string
	DecisionHuellaSHA256 string
	CodigoProbatorio     string
	Concedida            bool
	EmitidaEn            time.Time
	ValidaHasta          time.Time
}

// ResumenCandidataRegistroDecisionAutorizacionLigadaV3 es un valor nominal
// que solo puede nacer de la unión candidata exacta. No registra ni confirma
// la decisión y no es una capacidad ejecutable.
type ResumenCandidataRegistroDecisionAutorizacionLigadaV3 struct {
	bloqueoSerializacionRegistroAutorizacionLigadaV3
	datos *DatosResumenCandidataRegistroDecisionAutorizacionLigadaV3
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

// NuevaCandidataRegistroDecisionAutorizacionLigadaV3 fabrica la unica variante
// compatible con el resultado sellado de la decision.
func NuevaCandidataRegistroDecisionAutorizacionLigadaV3(
	solicitud domain.SolicitudAutorizacionLigadaV3,
	decision domain.DecisionAutorizacionLigadaV3,
	referenciaMotivo domain.ReferenciaEntradaCatalogo,
	resultadoContexto domain.ResultadoContextoActorRegistradoV2,
) (CandidataRegistroDecisionAutorizacionLigadaV3, error) {
	concedida, _, err := decision.Resultado()
	if err != nil {
		return CandidataRegistroDecisionAutorizacionLigadaV3{},
			ErrCandidataRegistroDecisionAutorizacionLigadaV3Invalida
	}
	if concedida {
		orden, errOrden := NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(
			solicitud, decision, referenciaMotivo, resultadoContexto,
		)
		if errOrden != nil {
			return CandidataRegistroDecisionAutorizacionLigadaV3{},
				errors.Join(
					ErrCandidataRegistroDecisionAutorizacionLigadaV3Invalida,
					errOrden,
				)
		}
		return CandidataRegistroDecisionAutorizacionLigadaV3{
			concedida: true,
			concesion: orden,
		}, nil
	}
	orden, err := NuevaOrdenRegistroDenegacionAutorizacionLigadaV3(
		solicitud, decision, referenciaMotivo, resultadoContexto,
	)
	if err != nil {
		return CandidataRegistroDecisionAutorizacionLigadaV3{},
			errors.Join(
				ErrCandidataRegistroDecisionAutorizacionLigadaV3Invalida,
				err,
			)
	}
	return CandidataRegistroDecisionAutorizacionLigadaV3{
		denegacion: orden,
	}, nil
}

// Resultado entrega copias nominales de las ordenes. Exactamente una de ellas
// es valida, según concedida; la variante opuesta permanece en valor cero.
func (c CandidataRegistroDecisionAutorizacionLigadaV3) Resultado() (
	concedida bool,
	concesion OrdenRegistroConcesionCandidataAutorizacionLigadaV3,
	denegacion OrdenRegistroDenegacionAutorizacionLigadaV3,
	err error,
) {
	if c.concedida {
		if _, errConcesion := c.concesion.Datos(); errConcesion != nil ||
			ordenDenegacionAutorizacionLigadaV3Valida(c.denegacion) {
			return false, OrdenRegistroConcesionCandidataAutorizacionLigadaV3{},
				OrdenRegistroDenegacionAutorizacionLigadaV3{},
				ErrCandidataRegistroDecisionAutorizacionLigadaV3Invalida
		}
		return true, c.concesion, OrdenRegistroDenegacionAutorizacionLigadaV3{}, nil
	}
	if _, errDenegacion := c.denegacion.Datos(); errDenegacion != nil ||
		ordenConcesionAutorizacionLigadaV3Valida(c.concesion) {
		return false, OrdenRegistroConcesionCandidataAutorizacionLigadaV3{},
			OrdenRegistroDenegacionAutorizacionLigadaV3{},
			ErrCandidataRegistroDecisionAutorizacionLigadaV3Invalida
	}
	return false, OrdenRegistroConcesionCandidataAutorizacionLigadaV3{},
		c.denegacion, nil
}

// Resumen verifica de nuevo la rama y deriva referencia, huella, código y
// ventana desde la decisión sellada. Nunca acepta esos valores por separado.
func (c CandidataRegistroDecisionAutorizacionLigadaV3) Resumen() (
	ResumenCandidataRegistroDecisionAutorizacionLigadaV3,
	error,
) {
	concedida, concesion, denegacion, err := c.Resultado()
	if err != nil {
		return ResumenCandidataRegistroDecisionAutorizacionLigadaV3{},
			ErrCandidataRegistroDecisionAutorizacionLigadaV3Invalida
	}
	var datosOrden DatosOrdenRegistroAutorizacionLigadaV3
	if concedida {
		datosOrden, err = concesion.Datos()
	} else {
		datosOrden, err = denegacion.Datos()
	}
	if err != nil {
		return ResumenCandidataRegistroDecisionAutorizacionLigadaV3{},
			ErrCandidataRegistroDecisionAutorizacionLigadaV3Invalida
	}
	resumen, err := resumenDecisionAutorizacionLigadaV3(datosOrden.Decision)
	huella, errHuella := domain.HuellaSHA256DecisionAutorizacionV3(
		datosOrden.Decision,
	)
	if err != nil || errHuella != nil || resumen.Concedida != concedida {
		return ResumenCandidataRegistroDecisionAutorizacionLigadaV3{},
			ErrCandidataRegistroDecisionAutorizacionLigadaV3Invalida
	}
	datos := &DatosResumenCandidataRegistroDecisionAutorizacionLigadaV3{
		DecisionRef: resumen.DecisionRef, DecisionHuellaSHA256: huella,
		CodigoProbatorio: resumen.Codigo, Concedida: resumen.Concedida,
		EmitidaEn: resumen.EmitidaEn, ValidaHasta: resumen.ValidaHasta,
	}
	resultado := ResumenCandidataRegistroDecisionAutorizacionLigadaV3{
		datos: datos,
	}
	if resultado.validar() != nil {
		return ResumenCandidataRegistroDecisionAutorizacionLigadaV3{},
			ErrCandidataRegistroDecisionAutorizacionLigadaV3Invalida
	}
	return resultado, nil
}

// Datos entrega una copia del resumen, que conserva el bloqueo de codecs y
// formateo. El resumen no transporta el documento VEC ni datos personales.
func (r ResumenCandidataRegistroDecisionAutorizacionLigadaV3) Datos() (
	DatosResumenCandidataRegistroDecisionAutorizacionLigadaV3,
	error,
) {
	if r.validar() != nil {
		return DatosResumenCandidataRegistroDecisionAutorizacionLigadaV3{},
			ErrCandidataRegistroDecisionAutorizacionLigadaV3Invalida
	}
	return *r.datos, nil
}

// ValidarPara vuelve a derivar el resumen desde la candidata. Las referencias
// y huellas se cotejan en tiempo constante para no crear una segunda fuente
// de autoridad a partir de campos minimizados.
func (r ResumenCandidataRegistroDecisionAutorizacionLigadaV3) ValidarPara(
	candidata CandidataRegistroDecisionAutorizacionLigadaV3,
) error {
	if r.validar() != nil {
		return ErrCandidataRegistroDecisionAutorizacionLigadaV3Invalida
	}
	esperado, err := candidata.Resumen()
	if err != nil || esperado.validar() != nil ||
		!textoRegistroAutorizacionLigadaV3Igual(
			r.datos.DecisionRef,
			esperado.datos.DecisionRef,
		) ||
		!textoRegistroAutorizacionLigadaV3Igual(
			r.datos.DecisionHuellaSHA256,
			esperado.datos.DecisionHuellaSHA256,
		) ||
		!textoRegistroAutorizacionLigadaV3Igual(
			r.datos.CodigoProbatorio,
			esperado.datos.CodigoProbatorio,
		) ||
		r.datos.Concedida != esperado.datos.Concedida ||
		!r.datos.EmitidaEn.Equal(esperado.datos.EmitidaEn) ||
		!r.datos.ValidaHasta.Equal(esperado.datos.ValidaHasta) {
		return ErrCandidataRegistroDecisionAutorizacionLigadaV3Invalida
	}
	return nil
}

func (r ResumenCandidataRegistroDecisionAutorizacionLigadaV3) validar() error {
	if r.datos == nil ||
		!referenciaDecisionAutorizacionLigadaV3Valida(r.datos.DecisionRef) ||
		!huellaSHA256RegistroAutorizacionLigadaV3Valida(
			r.datos.DecisionHuellaSHA256,
		) ||
		!domain.CodigoResultadoEvaluacionAutorizacionV3Valido(
			r.datos.CodigoProbatorio,
			r.datos.Concedida,
		) ||
		!instanteRegistroAutorizacionLigadaV3Canonico(r.datos.EmitidaEn) ||
		!instanteRegistroAutorizacionLigadaV3Canonico(r.datos.ValidaHasta) ||
		!r.datos.ValidaHasta.After(r.datos.EmitidaEn) {
		return ErrCandidataRegistroDecisionAutorizacionLigadaV3Invalida
	}
	return nil
}

func ordenConcesionAutorizacionLigadaV3Valida(
	orden OrdenRegistroConcesionCandidataAutorizacionLigadaV3,
) bool {
	_, err := orden.Datos()
	return err == nil
}

func ordenDenegacionAutorizacionLigadaV3Valida(
	orden OrdenRegistroDenegacionAutorizacionLigadaV3,
) bool {
	_, err := orden.Datos()
	return err == nil
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
	Codigo      string `json:"codigo"`
	EmitidaEn   string `json:"emitida_en"`
	ValidaHasta string `json:"valida_hasta"`
}

type resumenDecisionAutorizacionLigadaV3Datos struct {
	DecisionRef string
	Concedida   bool
	Codigo      string
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
	if err := json.Unmarshal(canon, &dto); err != nil ||
		!domain.CodigoResultadoEvaluacionAutorizacionV3Valido(
			dto.Codigo,
			dto.Concedida,
		) {
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
		DecisionRef: dto.DecisionRef, Concedida: dto.Concedida,
		Codigo: dto.Codigo, EmitidaEn: emitidaEn, ValidaHasta: validaHasta,
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

// PreparadorSolicitudLigadaV3 evalua con el contexto V3 completo. Una
// evaluacion positiva solo devuelve una orden candidata opaca: no prueba
// persistencia, no confirma una concesion y no es una capacidad ejecutable.
type PreparadorSolicitudLigadaV3 interface {
	PrepararSolicitudLigadaV3(
		context.Context,
		domain.SolicitudAutorizacionLigadaV3,
		domain.ResultadoContextoActorRegistradoV2,
	) (
		domain.DecisionAutorizacionLigadaV3,
		OrdenRegistroConcesionCandidataAutorizacionLigadaV3,
		error,
	)
}

// PreparadorRegistroCompuestoSolicitudLigadaV3 evalua sin ejecutar ninguna
// escritura. El generador pertenece exclusivamente a la operacion y debe
// devolver la DecisionRef previamente fijada por la reserva autoritativa.
type PreparadorRegistroCompuestoSolicitudLigadaV3 interface {
	PrepararRegistroCompuestoSolicitudLigadaV3(
		context.Context,
		domain.SolicitudAutorizacionLigadaV3,
		domain.ResultadoContextoActorRegistradoV2,
		GeneradorReferenciaDecisionAutorizacion,
	) (
		domain.DecisionAutorizacionLigadaV3,
		CandidataRegistroDecisionAutorizacionLigadaV3,
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

func textoRegistroAutorizacionLigadaV3Igual(primero string, segundo string) bool {
	return len(primero) == len(segundo) &&
		subtle.ConstantTimeCompare([]byte(primero), []byte(segundo)) == 1
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
