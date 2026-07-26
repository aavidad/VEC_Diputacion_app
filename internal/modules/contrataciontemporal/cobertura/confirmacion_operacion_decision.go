package cobertura

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	puertosct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

var (
	ErrContratoConfirmacionOperacionDecisionCoberturaInvalido = errors.New(
		"contratacion temporal: contrato de confirmacion de decision de cobertura invalido",
	)
	ErrResultadoConfirmacionOperacionDecisionCoberturaAmbiguo = errors.New(
		"contratacion temporal: resultado de confirmacion de decision de cobertura ambiguo; requiere reconciliacion primaria",
	)
	ErrResultadoConfirmacionOperacionDecisionCoberturaNoDisponible = errors.New(
		"contratacion temporal: resultado de confirmacion de decision de cobertura no disponible",
	)
	errFalloAntesCommitOperacionDecisionCobertura = errors.New(
		"fallo interno anterior al commit de decision de cobertura",
	)
)

const esquemaHuellaConfirmacionOperacionDecisionCobertura = "" +
	"VEC-CT-CONFIRMACION-OPERACION-DECISION-COBERTURA-C3-V1"

// TransaccionOperacionDecisionCobertura es la frontera sellada que el servicio
// recibe de la raíz de composición. Solo cobertura puede implementarla; el
// adaptador productivo aporta exclusivamente EjecutorSesionTCB y nunca recibe
// la orden opaca.
type TransaccionOperacionDecisionCobertura interface {
	confirmarOperacionDecisionCobertura(
		context.Context,
		OrdenOperacionDecisionCobertura,
	) (ResultadoConfirmacionOperacionDecisionCobertura, error)
}

// ConfirmarOperacionDecisionCobertura invoca la frontera sellada sin exponer
// su operación privada.
func ConfirmarOperacionDecisionCobertura(
	ctx context.Context,
	transaccion TransaccionOperacionDecisionCobertura,
	orden OrdenOperacionDecisionCobertura,
) (ResultadoConfirmacionOperacionDecisionCobertura, error) {
	if dependenciaGobiernoOperacionCoberturaNula(ctx) ||
		dependenciaGobiernoOperacionCoberturaNula(transaccion) ||
		orden.validar() != nil {
		return ResultadoConfirmacionOperacionDecisionCobertura{},
			ErrContratoConfirmacionOperacionDecisionCoberturaInvalido
	}
	resultado, err :=
		transaccion.confirmarOperacionDecisionCobertura(ctx, orden)
	if err == errFalloAntesCommitOperacionDecisionCobertura {
		return ResultadoConfirmacionOperacionDecisionCobertura{},
			ErrResultadoConfirmacionOperacionDecisionCoberturaNoDisponible
	}
	return resultado, err
}

// ReconciliadorResultadoAmbiguoOperacionDecisionCobertura consulta el primario
// tras una respuesta perdida o indeterminada. Es un contrato distinto de la
// confirmación: nunca repite la orden ni concede permiso para reintentarla.
type ReconciliadorResultadoAmbiguoOperacionDecisionCobertura interface {
	ReconciliarResultadoAmbiguoOperacionDecisionCobertura(
		context.Context,
		SolicitudReconciliacionOperacionDecisionCobertura,
	) (ResultadoReconciliacionOperacionDecisionCobertura, error)
}

// ResultadoConfirmacionOperacionDecisionCobertura liga un recibo terminal a
// la orden exacta. Es prueba nominal del valor devuelto, no del COMMIT: solo
// el adaptador O4-04 puede afirmar durabilidad desde la transacción primaria.
type ResultadoConfirmacionOperacionDecisionCobertura struct {
	bloqueoSerializacionOperacionDecisionCobertura
	datos *datosResultadoConfirmacionOperacionDecisionCobertura
}

type datosResultadoConfirmacionOperacionDecisionCobertura struct {
	huellaOrdenSHA256 string
	recibo            ReciboOperacionDecisionCobertura
}

// NuevaResultadoConfirmacionOperacionDecisionCobertura valida la ligadura
// Recibo↔Orden y conserva una copia defensiva. No ejecuta ningún efecto.
func NuevaResultadoConfirmacionOperacionDecisionCobertura(
	orden OrdenOperacionDecisionCobertura,
	recibo ReciboOperacionDecisionCobertura,
) (ResultadoConfirmacionOperacionDecisionCobertura, error) {
	huella, errHuella := huellaConfirmacionOperacionDecisionCobertura(orden)
	if errHuella != nil ||
		validarReciboParaOrdenOperacionDecisionCobertura(orden, recibo) != nil {
		return ResultadoConfirmacionOperacionDecisionCobertura{},
			ErrContratoConfirmacionOperacionDecisionCoberturaInvalido
	}
	resultado := ResultadoConfirmacionOperacionDecisionCobertura{
		datos: &datosResultadoConfirmacionOperacionDecisionCobertura{
			huellaOrdenSHA256: huella,
			recibo:            clonarReciboOperacionDecisionCobertura(recibo),
		},
	}
	if resultado.validarPara(orden) != nil {
		return ResultadoConfirmacionOperacionDecisionCobertura{},
			ErrContratoConfirmacionOperacionDecisionCoberturaInvalido
	}
	return resultado, nil
}

// ReciboPara entrega una copia solo al poseedor de la misma orden opaca.
func (r ResultadoConfirmacionOperacionDecisionCobertura) ReciboPara(
	orden OrdenOperacionDecisionCobertura,
) (ReciboOperacionDecisionCobertura, error) {
	if r.validarPara(orden) != nil {
		return ReciboOperacionDecisionCobertura{},
			ErrContratoConfirmacionOperacionDecisionCoberturaInvalido
	}
	return clonarReciboOperacionDecisionCobertura(r.datos.recibo), nil
}

func (r ResultadoConfirmacionOperacionDecisionCobertura) validarPara(
	orden OrdenOperacionDecisionCobertura,
) error {
	huella, err := huellaConfirmacionOperacionDecisionCobertura(orden)
	if r.datos == nil || err != nil ||
		!huellaSHA256OperacionDecisionCoberturaValida(
			r.datos.huellaOrdenSHA256,
		) ||
		!referenciasOperacionDecisionCoberturaIguales(
			huella,
			r.datos.huellaOrdenSHA256,
		) ||
		validarReciboParaOrdenOperacionDecisionCobertura(
			orden,
			r.datos.recibo,
		) != nil {
		return ErrContratoConfirmacionOperacionDecisionCoberturaInvalido
	}
	return nil
}

// DatosConsultaPrimariaOperacionDecisionCobertura contiene exclusivamente las
// coordenadas minimizadas que permiten buscar el recibo en el primario.
type DatosConsultaPrimariaOperacionDecisionCobertura struct {
	bloqueoSerializacionOperacionDecisionCobertura
	OrganizacionRef   string
	ExpedienteRef     string
	VersionExpediente uint64
	ReservaRef        string
	ReciboRef         string
	CorrelacionVECRef string
	DecisionVECRef    string
	RevisionCercado   uint64
}

// SolicitudReconciliacionOperacionDecisionCobertura no contiene token,
// agregado, C1, C2, identidad personal, motivo, recurso ni órdenes de consumo.
type SolicitudReconciliacionOperacionDecisionCobertura struct {
	bloqueoSerializacionOperacionDecisionCobertura
	datos *datosSolicitudReconciliacionOperacionDecisionCobertura
}

type datosSolicitudReconciliacionOperacionDecisionCobertura struct {
	coordenadas       DatosConsultaPrimariaOperacionDecisionCobertura
	huellaOrdenSHA256 string
}

// NuevaSolicitudReconciliacionOperacionDecisionCobertura deriva la consulta
// primaria de la orden; no admite coordenadas suministradas por un canal.
func NuevaSolicitudReconciliacionOperacionDecisionCobertura(
	orden OrdenOperacionDecisionCobertura,
) (SolicitudReconciliacionOperacionDecisionCobertura, error) {
	datos, huellaOrden, err :=
		datosConsultaPrimariaOperacionDecisionCobertura(orden)
	if err != nil {
		return SolicitudReconciliacionOperacionDecisionCobertura{},
			ErrContratoConfirmacionOperacionDecisionCoberturaInvalido
	}
	solicitud := SolicitudReconciliacionOperacionDecisionCobertura{
		datos: &datosSolicitudReconciliacionOperacionDecisionCobertura{
			coordenadas:       datos,
			huellaOrdenSHA256: huellaOrden,
		},
	}
	if solicitud.validarPara(orden) != nil {
		return SolicitudReconciliacionOperacionDecisionCobertura{},
			ErrContratoConfirmacionOperacionDecisionCoberturaInvalido
	}
	return solicitud, nil
}

// CoordenadasPrimarias entrega una copia minimizada. No es permiso de retry.
func (s SolicitudReconciliacionOperacionDecisionCobertura) CoordenadasPrimarias() (
	DatosConsultaPrimariaOperacionDecisionCobertura,
	error,
) {
	if s.datos == nil ||
		validarDatosConsultaPrimariaOperacionDecisionCobertura(
			s.datos.coordenadas,
		) != nil ||
		!huellaSHA256OperacionDecisionCoberturaValida(
			s.datos.huellaOrdenSHA256,
		) {
		return DatosConsultaPrimariaOperacionDecisionCobertura{},
			ErrContratoConfirmacionOperacionDecisionCoberturaInvalido
	}
	return s.datos.coordenadas, nil
}

func (s SolicitudReconciliacionOperacionDecisionCobertura) validarPara(
	orden OrdenOperacionDecisionCobertura,
) error {
	esperados, huellaOrden, err :=
		datosConsultaPrimariaOperacionDecisionCobertura(orden)
	if s.datos == nil || err != nil ||
		validarDatosConsultaPrimariaOperacionDecisionCobertura(
			s.datos.coordenadas,
		) != nil ||
		!datosConsultaPrimariaOperacionDecisionCoberturaIguales(
			s.datos.coordenadas,
			esperados,
		) ||
		!referenciasOperacionDecisionCoberturaIguales(
			s.datos.huellaOrdenSHA256,
			huellaOrden,
		) {
		return ErrContratoConfirmacionOperacionDecisionCoberturaInvalido
	}
	return nil
}

// ResultadoIntentoConfirmacionOperacionDecisionCobertura es una unión exacta:
// contiene una confirmación válida o una solicitud de reconciliación. La rama
// ambigua carece deliberadamente de una señal «reintentar».
type ResultadoIntentoConfirmacionOperacionDecisionCobertura struct {
	bloqueoSerializacionOperacionDecisionCobertura
	confirmacion     *ResultadoConfirmacionOperacionDecisionCobertura
	reconciliacion   *SolicitudReconciliacionOperacionDecisionCobertura
	falloAntesCommit *pruebaFalloAntesCommitOperacionDecisionCobertura
}

type pruebaFalloAntesCommitOperacionDecisionCobertura struct {
	huellaOrdenSHA256 string
}

// ConfirmacionPara devuelve el resultado únicamente en la rama confirmada.
func (r ResultadoIntentoConfirmacionOperacionDecisionCobertura) ConfirmacionPara(
	orden OrdenOperacionDecisionCobertura,
) (ResultadoConfirmacionOperacionDecisionCobertura, bool) {
	if r.confirmacion == nil || r.reconciliacion != nil ||
		r.confirmacion.validarPara(orden) != nil {
		return ResultadoConfirmacionOperacionDecisionCobertura{}, false
	}
	return *r.confirmacion, true
}

// ReconciliacionPara devuelve la consulta primaria únicamente en la rama
// ambigua. Su presencia nunca autoriza a repetir la confirmación.
func (r ResultadoIntentoConfirmacionOperacionDecisionCobertura) ReconciliacionPara(
	orden OrdenOperacionDecisionCobertura,
) (SolicitudReconciliacionOperacionDecisionCobertura, bool) {
	if r.reconciliacion == nil || r.confirmacion != nil ||
		r.reconciliacion.validarPara(orden) != nil {
		return SolicitudReconciliacionOperacionDecisionCobertura{}, false
	}
	return *r.reconciliacion, true
}

// FalloAntesCommitPara acredita la rama no ambigua únicamente para la orden
// que la originó. No existe constructor público ni error sentinel que un
// adaptador o canal pueda forjar para omitir la reconciliación.
func (r ResultadoIntentoConfirmacionOperacionDecisionCobertura) FalloAntesCommitPara(
	orden OrdenOperacionDecisionCobertura,
) bool {
	huella, err := huellaConfirmacionOperacionDecisionCobertura(orden)
	return err == nil &&
		r.confirmacion == nil &&
		r.reconciliacion == nil &&
		r.falloAntesCommit != nil &&
		huellaSHA256OperacionDecisionCoberturaValida(
			r.falloAntesCommit.huellaOrdenSHA256,
		) &&
		referenciasOperacionDecisionCoberturaIguales(
			huella,
			r.falloAntesCommit.huellaOrdenSHA256,
		)
}

// IntentarConfirmacionOperacionDecisionCobertura invoca exactamente una vez
// la transacción. Un recibo válido prevalece sobre un error o una cancelación
// observada después del COMMIT. Un fallo acreditado antes de COMMIT termina
// sin reconciliar; cualquier otro intento no confirmado es ambiguo y obliga a
// consultar el primario. Esta función nunca hace retry.
func IntentarConfirmacionOperacionDecisionCobertura(
	ctx context.Context,
	transaccion TransaccionOperacionDecisionCobertura,
	orden OrdenOperacionDecisionCobertura,
) (
	ResultadoIntentoConfirmacionOperacionDecisionCobertura,
	error,
) {
	if dependenciaGobiernoOperacionCoberturaNula(ctx) ||
		dependenciaGobiernoOperacionCoberturaNula(transaccion) ||
		orden.validar() != nil {
		return ResultadoIntentoConfirmacionOperacionDecisionCobertura{},
			ErrContratoConfirmacionOperacionDecisionCoberturaInvalido
	}
	if err := ctx.Err(); err != nil {
		return ResultadoIntentoConfirmacionOperacionDecisionCobertura{}, err
	}
	confirmacion, errConfirmacion := transaccion.confirmarOperacionDecisionCobertura(
		ctx,
		orden,
	)
	if confirmacion.validarPara(orden) == nil {
		copia := confirmacion
		return ResultadoIntentoConfirmacionOperacionDecisionCobertura{
			confirmacion: &copia,
		}, nil
	}
	if errConfirmacion == errFalloAntesCommitOperacionDecisionCobertura {
		huellaOrden, errHuella :=
			huellaConfirmacionOperacionDecisionCobertura(orden)
		if errHuella != nil {
			return ResultadoIntentoConfirmacionOperacionDecisionCobertura{},
				ErrContratoConfirmacionOperacionDecisionCoberturaInvalido
		}
		return ResultadoIntentoConfirmacionOperacionDecisionCobertura{
				falloAntesCommit: &pruebaFalloAntesCommitOperacionDecisionCobertura{
					huellaOrdenSHA256: huellaOrden,
				},
			},
			ErrResultadoConfirmacionOperacionDecisionCoberturaNoDisponible
	}
	solicitud, err := NuevaSolicitudReconciliacionOperacionDecisionCobertura(
		orden,
	)
	if err != nil {
		return ResultadoIntentoConfirmacionOperacionDecisionCobertura{},
			ErrContratoConfirmacionOperacionDecisionCoberturaInvalido
	}
	return ResultadoIntentoConfirmacionOperacionDecisionCobertura{
			reconciliacion: &solicitud,
		},
		ErrResultadoConfirmacionOperacionDecisionCoberturaAmbiguo
}

// ResultadoReconciliacionOperacionDecisionCobertura contiene una confirmación
// exacta o una observación no concluyente del primario. La segunda rama no
// equivale a rollback probado y nunca concede permiso para reintentar.
type ResultadoReconciliacionOperacionDecisionCobertura struct {
	bloqueoSerializacionOperacionDecisionCobertura
	huellaSolicitudSHA256 string
	observadaEnPrimario   time.Time
	reciboCandidato       *ReciboOperacionDecisionCobertura
	noConcluyente         bool
}

func NuevaResultadoReconciliacionConfirmadaOperacionDecisionCobertura(
	solicitud SolicitudReconciliacionOperacionDecisionCobertura,
	recibo ReciboOperacionDecisionCobertura,
	observadaEnPrimario time.Time,
) (ResultadoReconciliacionOperacionDecisionCobertura, error) {
	huellaSolicitud, errHuella :=
		huellaSolicitudReconciliacionOperacionDecisionCobertura(solicitud)
	if errHuella != nil ||
		validarReciboCandidatoReconciliacionOperacionDecisionCobertura(
			solicitud,
			recibo,
		) != nil ||
		!instanteOperacionDecisionCoberturaValido(observadaEnPrimario) ||
		observadaEnPrimario.Before(recibo.ConfirmadaEn) {
		return ResultadoReconciliacionOperacionDecisionCobertura{},
			ErrContratoConfirmacionOperacionDecisionCoberturaInvalido
	}
	copiaRecibo := clonarReciboOperacionDecisionCobertura(recibo)
	resultado := ResultadoReconciliacionOperacionDecisionCobertura{
		huellaSolicitudSHA256: huellaSolicitud,
		observadaEnPrimario:   observadaEnPrimario,
		reciboCandidato:       &copiaRecibo,
	}
	if resultado.validarPara(solicitud) != nil {
		return ResultadoReconciliacionOperacionDecisionCobertura{},
			ErrContratoConfirmacionOperacionDecisionCoberturaInvalido
	}
	return resultado, nil
}

func NuevaResultadoReconciliacionNoConcluyenteOperacionDecisionCobertura(
	solicitud SolicitudReconciliacionOperacionDecisionCobertura,
	observadaEnPrimario time.Time,
) (ResultadoReconciliacionOperacionDecisionCobertura, error) {
	huella, err :=
		huellaSolicitudReconciliacionOperacionDecisionCobertura(solicitud)
	if err != nil ||
		!instanteOperacionDecisionCoberturaValido(observadaEnPrimario) {
		return ResultadoReconciliacionOperacionDecisionCobertura{},
			ErrContratoConfirmacionOperacionDecisionCoberturaInvalido
	}
	return ResultadoReconciliacionOperacionDecisionCobertura{
		huellaSolicitudSHA256: huella,
		observadaEnPrimario:   observadaEnPrimario,
		noConcluyente:         true,
	}, nil
}

func (r ResultadoReconciliacionOperacionDecisionCobertura) validarPara(
	solicitud SolicitudReconciliacionOperacionDecisionCobertura,
) error {
	huella, err :=
		huellaSolicitudReconciliacionOperacionDecisionCobertura(solicitud)
	if err != nil ||
		!huellaSHA256OperacionDecisionCoberturaValida(
			r.huellaSolicitudSHA256,
		) ||
		!referenciasOperacionDecisionCoberturaIguales(
			huella,
			r.huellaSolicitudSHA256,
		) ||
		!instanteOperacionDecisionCoberturaValido(r.observadaEnPrimario) ||
		(r.reciboCandidato != nil) == r.noConcluyente {
		return ErrContratoConfirmacionOperacionDecisionCoberturaInvalido
	}
	if r.reciboCandidato != nil &&
		validarReciboCandidatoReconciliacionOperacionDecisionCobertura(
			solicitud,
			*r.reciboCandidato,
		) != nil {
		return ErrContratoConfirmacionOperacionDecisionCoberturaInvalido
	}
	if r.reciboCandidato != nil &&
		r.observadaEnPrimario.Before(
			r.reciboCandidato.ConfirmadaEn,
		) {
		return ErrContratoConfirmacionOperacionDecisionCoberturaInvalido
	}
	return nil
}

// ConfirmacionPara eleva el recibo no confiable leído del primario únicamente
// si coincide con la orden opaca exacta conservada por la aplicación.
func (r ResultadoReconciliacionOperacionDecisionCobertura) ConfirmacionPara(
	orden OrdenOperacionDecisionCobertura,
) (ResultadoConfirmacionOperacionDecisionCobertura, bool) {
	solicitud, err :=
		NuevaSolicitudReconciliacionOperacionDecisionCobertura(orden)
	if err != nil || r.reciboCandidato == nil ||
		r.validarPara(solicitud) != nil {
		return ResultadoConfirmacionOperacionDecisionCobertura{}, false
	}
	confirmacion, err :=
		NuevaResultadoConfirmacionOperacionDecisionCobertura(
			orden,
			*r.reciboCandidato,
		)
	if err != nil {
		return ResultadoConfirmacionOperacionDecisionCobertura{}, false
	}
	return confirmacion, true
}

// ReconciliarConfirmacionOperacionDecisionCobertura consulta una sola vez el
// primario. Un recibo exacto prevalece sobre cancelación tardía o error de
// transporte; cualquier otra respuesta conserva la ambigüedad.
func ReconciliarConfirmacionOperacionDecisionCobertura(
	ctx context.Context,
	reconciliador ReconciliadorResultadoAmbiguoOperacionDecisionCobertura,
	solicitud SolicitudReconciliacionOperacionDecisionCobertura,
	orden OrdenOperacionDecisionCobertura,
) (ResultadoConfirmacionOperacionDecisionCobertura, error) {
	if dependenciaGobiernoOperacionCoberturaNula(ctx) ||
		dependenciaGobiernoOperacionCoberturaNula(reconciliador) ||
		solicitud.validarPara(orden) != nil {
		return ResultadoConfirmacionOperacionDecisionCobertura{},
			ErrContratoConfirmacionOperacionDecisionCoberturaInvalido
	}
	if err := ctx.Err(); err != nil {
		return ResultadoConfirmacionOperacionDecisionCobertura{}, err
	}
	resultado, _ :=
		reconciliador.ReconciliarResultadoAmbiguoOperacionDecisionCobertura(
			ctx,
			solicitud,
		)
	if resultado.validarPara(solicitud) == nil {
		if confirmacion, valida := resultado.ConfirmacionPara(orden); valida {
			return confirmacion, nil
		}
	}
	return ResultadoConfirmacionOperacionDecisionCobertura{},
		ErrResultadoConfirmacionOperacionDecisionCoberturaAmbiguo
}

func validarReciboParaOrdenOperacionDecisionCobertura(
	orden OrdenOperacionDecisionCobertura,
	recibo ReciboOperacionDecisionCobertura,
) error {
	if orden.validar() != nil {
		return ErrContratoConfirmacionOperacionDecisionCoberturaInvalido
	}
	if orden.datos.concesion != nil {
		return validarReciboConcedidoParaOrdenOperacionDecisionCobertura(
			orden.datos.concesion,
			recibo,
		)
	}
	return validarReciboDenegadoParaOrdenOperacionDecisionCobertura(
		orden.datos.denegacion,
		recibo,
	)
}

func validarReciboConcedidoParaOrdenOperacionDecisionCobertura(
	datos *datosOrdenConcedidaOperacionDecisionCobertura,
	recibo ReciboOperacionDecisionCobertura,
) error {
	resumen, err := datos.resumen.Datos()
	if err != nil || datos.preparacion == nil ||
		recibo.ValidarParaReservaCongelada(
			datos.preparacion.solicitudReserva,
			datos.preparacion.reserva,
		) != nil ||
		!reciboOperacionDecisionCoberturaCoincideConResumen(
			recibo,
			resumen,
		) ||
		!recibo.ConcedidaVEC || recibo.Aplicada == nil ||
		recibo.DenegadaVEC != nil ||
		recibo.ConfirmadaEn.Before(datos.efectoEn) ||
		!recibo.ConfirmadaEn.Before(datos.validaHasta) ||
		datos.agregadoSiguiente.Version >
			MaximoEnteroSeguroOperacionDecisionCobertura ||
		len(datos.agregadoSiguiente.DecisionesCobertura) == 0 {
		return ErrContratoConfirmacionOperacionDecisionCoberturaInvalido
	}
	decision := datos.agregadoSiguiente.DecisionesCobertura[len(datos.agregadoSiguiente.DecisionesCobertura)-1]
	aplicada := recibo.Aplicada
	if !referenciasOperacionDecisionCoberturaIguales(
		aplicada.DecisionCoberturaRef,
		decision.Referencia,
	) ||
		!referenciasOperacionDecisionCoberturaIguales(
			aplicada.DecisionCoberturaHuella,
			decision.HuellaSHA256,
		) ||
		aplicada.VersionResultante != datos.agregadoSiguiente.Version ||
		!referenciasOperacionDecisionCoberturaIguales(
			aplicada.EventoRef,
			datos.preparacion.reserva.EventoRef,
		) ||
		!referenciasOperacionDecisionCoberturaIguales(
			aplicada.ActuacionRef,
			datos.preparacion.reserva.ActuacionRef,
		) {
		return ErrContratoConfirmacionOperacionDecisionCoberturaInvalido
	}
	return nil
}

func validarReciboDenegadoParaOrdenOperacionDecisionCobertura(
	datos *datosOrdenDenegadaOperacionDecisionCobertura,
	recibo ReciboOperacionDecisionCobertura,
) error {
	if datos == nil {
		return ErrContratoConfirmacionOperacionDecisionCoberturaInvalido
	}
	resumen, err := datos.resumen.Datos()
	consulta, errConsulta := datos.prueba.reserva.solicitud.consultaConfirmada()
	reserva := datos.prueba.reserva
	if err != nil || errConsulta != nil ||
		recibo.ValidarPara(consulta) != nil ||
		!reciboOperacionDecisionCoberturaCoincideConResumen(
			recibo,
			resumen,
		) ||
		recibo.ConcedidaVEC || recibo.Aplicada != nil ||
		recibo.DenegadaVEC == nil ||
		!referenciasOperacionDecisionCoberturaIguales(
			recibo.ReciboRef,
			reserva.reciboRef,
		) ||
		!referenciasOperacionDecisionCoberturaIguales(
			recibo.ReservaRef,
			reserva.reservaRef,
		) ||
		!referenciasOperacionDecisionCoberturaIguales(
			recibo.AuditoriaRef,
			reserva.auditoriaRef,
		) ||
		!referenciasOperacionDecisionCoberturaIguales(
			recibo.CorrelacionVECRef,
			reserva.correlacionVECRef,
		) ||
		!referenciasOperacionDecisionCoberturaIguales(
			recibo.DecisionVECRef,
			reserva.decisionVECRef,
		) ||
		recibo.RevisionCercado != reserva.revisionCercado ||
		!referenciasOperacionDecisionCoberturaIguales(
			recibo.AmbitoIdempotenciaHMAC,
			reserva.ambitoHMAC,
		) ||
		!referenciasOperacionDecisionCoberturaIguales(
			recibo.HuellaSemanticaHMAC,
			reserva.semanticaHMAC,
		) ||
		recibo.ConfirmadaEn.Before(resumen.EmitidaEn) ||
		recibo.ConfirmadaEn.Before(reserva.observadaEnDB) ||
		!recibo.ConfirmadaEn.Before(datos.validaHasta) {
		return ErrContratoConfirmacionOperacionDecisionCoberturaInvalido
	}
	return nil
}

func reciboOperacionDecisionCoberturaCoincideConResumen(
	recibo ReciboOperacionDecisionCobertura,
	resumen puertosvec.DatosResumenCandidataRegistroDecisionAutorizacionLigadaV3,
) bool {
	return referenciasOperacionDecisionCoberturaIguales(
		recibo.DecisionVECRef,
		resumen.DecisionRef,
	) &&
		referenciasOperacionDecisionCoberturaIguales(
			recibo.DecisionVECHuellaSHA256,
			resumen.DecisionHuellaSHA256,
		) &&
		recibo.CodigoProbatorioVEC == resumen.CodigoProbatorio &&
		recibo.ConcedidaVEC == resumen.Concedida
}

func validarReciboCandidatoReconciliacionOperacionDecisionCobertura(
	solicitud SolicitudReconciliacionOperacionDecisionCobertura,
	recibo ReciboOperacionDecisionCobertura,
) error {
	datos, err := solicitud.CoordenadasPrimarias()
	if err != nil ||
		!domain.ReferenciaOpacaValida(recibo.AuditoriaRef) ||
		!huellaSHA256OperacionDecisionCoberturaValida(
			recibo.DecisionVECHuellaSHA256,
		) ||
		!dominiovec.CodigoResultadoEvaluacionAutorizacionV3Valido(
			recibo.CodigoProbatorioVEC,
			recibo.ConcedidaVEC,
		) ||
		!puertosct.SelloHMACSHA256Valido(
			recibo.AmbitoIdempotenciaHMAC,
		) ||
		!puertosct.SelloHMACSHA256Valido(
			recibo.HuellaSemanticaHMAC,
		) ||
		!instanteOperacionDecisionCoberturaValido(recibo.ConfirmadaEn) ||
		(recibo.Aplicada == nil) == (recibo.DenegadaVEC == nil) ||
		!referenciasOperacionDecisionCoberturaIguales(
			recibo.ReciboRef,
			datos.ReciboRef,
		) ||
		!referenciasOperacionDecisionCoberturaIguales(
			recibo.ReservaRef,
			datos.ReservaRef,
		) ||
		!referenciasOperacionDecisionCoberturaIguales(
			recibo.CorrelacionVECRef,
			datos.CorrelacionVECRef,
		) ||
		!referenciasOperacionDecisionCoberturaIguales(
			recibo.DecisionVECRef,
			datos.DecisionVECRef,
		) ||
		recibo.RevisionCercado != datos.RevisionCercado {
		return ErrContratoConfirmacionOperacionDecisionCoberturaInvalido
	}
	if recibo.Aplicada == nil {
		if recibo.ConcedidaVEC {
			return ErrContratoConfirmacionOperacionDecisionCoberturaInvalido
		}
		return nil
	}
	aplicada := recibo.Aplicada
	if !recibo.ConcedidaVEC ||
		!referenciaDecisionCoberturaLigadaAHuella(
			aplicada.DecisionCoberturaRef,
			aplicada.DecisionCoberturaHuella,
		) ||
		aplicada.VersionResultante != datos.VersionExpediente+1 ||
		aplicada.VersionResultante >
			MaximoEnteroSeguroOperacionDecisionCobertura ||
		!domain.ReferenciaOpacaValida(aplicada.EventoRef) ||
		!domain.ReferenciaOpacaValida(aplicada.ActuacionRef) {
		return ErrContratoConfirmacionOperacionDecisionCoberturaInvalido
	}
	return nil
}

func datosConsultaPrimariaOperacionDecisionCobertura(
	orden OrdenOperacionDecisionCobertura,
) (DatosConsultaPrimariaOperacionDecisionCobertura, string, error) {
	huella, err := huellaConfirmacionOperacionDecisionCobertura(orden)
	if err != nil {
		return DatosConsultaPrimariaOperacionDecisionCobertura{}, "",
			ErrContratoConfirmacionOperacionDecisionCoberturaInvalido
	}
	var organizacion, expediente, reserva, recibo, correlacion, decision string
	var version, cercado uint64
	if orden.datos.concesion != nil {
		datos := orden.datos.concesion.preparacion
		organizacion = datos.reserva.AgregadoAnterior.OrganizacionRef
		expediente = datos.reserva.AgregadoAnterior.Referencia
		version = datos.reserva.AgregadoAnterior.Version
		reserva = datos.reserva.ReservaRef
		recibo = datos.reserva.ReciboRef
		correlacion = datos.reserva.CorrelacionVECRef
		decision = datos.reserva.DecisionVECRef
		cercado = datos.reserva.RevisionCercado
	} else {
		datos := orden.datos.denegacion.prueba.reserva
		organizacion = datos.organizacionRef
		expediente = datos.expedienteRef
		version = datos.versionExpediente
		reserva = datos.reservaRef
		recibo = datos.reciboRef
		correlacion = datos.correlacionVECRef
		decision = datos.decisionVECRef
		cercado = datos.revisionCercado
	}
	resultado := DatosConsultaPrimariaOperacionDecisionCobertura{
		OrganizacionRef: organizacion, ExpedienteRef: expediente,
		VersionExpediente: version, ReservaRef: reserva, ReciboRef: recibo,
		CorrelacionVECRef: correlacion, DecisionVECRef: decision,
		RevisionCercado: cercado,
	}
	if validarDatosConsultaPrimariaOperacionDecisionCobertura(resultado) != nil {
		return DatosConsultaPrimariaOperacionDecisionCobertura{}, "",
			ErrContratoConfirmacionOperacionDecisionCoberturaInvalido
	}
	return resultado, huella, nil
}

func validarDatosConsultaPrimariaOperacionDecisionCobertura(
	datos DatosConsultaPrimariaOperacionDecisionCobertura,
) error {
	if !domain.ReferenciaOpacaValida(datos.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(datos.ExpedienteRef) ||
		datos.VersionExpediente < 2 ||
		datos.VersionExpediente >=
			MaximoEnteroSeguroOperacionDecisionCobertura ||
		!domain.ReferenciaOpacaValida(datos.ReservaRef) ||
		!domain.ReferenciaOpacaValida(datos.ReciboRef) ||
		!domain.ReferenciaOpacaValida(datos.CorrelacionVECRef) ||
		!domain.ReferenciaOpacaValida(datos.DecisionVECRef) ||
		datos.RevisionCercado == 0 ||
		datos.RevisionCercado >
			MaximoEnteroSeguroOperacionDecisionCobertura {
		return ErrContratoConfirmacionOperacionDecisionCoberturaInvalido
	}
	return nil
}

func datosConsultaPrimariaOperacionDecisionCoberturaIguales(
	primero DatosConsultaPrimariaOperacionDecisionCobertura,
	segundo DatosConsultaPrimariaOperacionDecisionCobertura,
) bool {
	return primero.OrganizacionRef == segundo.OrganizacionRef &&
		primero.ExpedienteRef == segundo.ExpedienteRef &&
		primero.VersionExpediente == segundo.VersionExpediente &&
		primero.ReservaRef == segundo.ReservaRef &&
		primero.ReciboRef == segundo.ReciboRef &&
		primero.CorrelacionVECRef == segundo.CorrelacionVECRef &&
		primero.DecisionVECRef == segundo.DecisionVECRef &&
		primero.RevisionCercado == segundo.RevisionCercado
}
