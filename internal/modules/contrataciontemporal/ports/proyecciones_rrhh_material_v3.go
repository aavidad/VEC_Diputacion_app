package ports

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"time"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

// MaterialAutorizacionConsultaRRHH conserva el conjunto probatorio VEC-AD-3
// completo que PostgreSQL deberá verificar y consumir. El resumen tipado solo
// permite cotejos defensivos en Go; nunca sustituye las diez piezas originales.
type MaterialAutorizacionConsultaRRHH struct {
	bloqueoSerializacionConsultaRRHH
	solicitud       dominiovec.SolicitudAutorizacionLigadaV3
	decision        dominiovec.DecisionAutorizacionLigadaV3
	confirmacion    puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3
	resultado       dominiovec.ResultadoContextoActorRegistradoV2
	exportacion     puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3
	materialHuella  string
	capacidadHuella string
	decisionHuella  string
	motivoHuella    string
	operacion       string
	recursoRef      string
	recursoHuella   string
	emitidaEn       time.Time
	expiraEn        time.Time
}

func nuevoMaterialAutorizacionConsultaRRHH(
	contexto ContextoConsultaRRHH,
	solicitud dominiovec.SolicitudAutorizacionLigadaV3,
	decision dominiovec.DecisionAutorizacionLigadaV3,
	confirmacion puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3,
	resultado dominiovec.ResultadoContextoActorRegistradoV2,
	exportador puertosvec.ExportadorMaterialConsumoAutorizacionAtestadaV3,
	instante time.Time,
) (MaterialAutorizacionConsultaRRHH, error) {
	if exportadorMaterialConsultaRRHHNulo(exportador) ||
		contexto.validarEn(instante) != nil {
		return MaterialAutorizacionConsultaRRHH{}, ErrCapacidadConsultaRRHHInvalida
	}
	exportacion, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || exportacion.ValidarEstructura() != nil {
		return MaterialAutorizacionConsultaRRHH{}, ErrCapacidadConsultaRRHHInvalida
	}
	clonResultado, err := resultado.Clonar()
	if err != nil {
		return MaterialAutorizacionConsultaRRHH{}, ErrCapacidadConsultaRRHHInvalida
	}
	material := MaterialAutorizacionConsultaRRHH{
		solicitud: solicitud, decision: decision, confirmacion: confirmacion,
		resultado: clonResultado, exportacion: exportacion,
	}
	material.materialHuella, err = exportacion.HuellaConjuntoSHA256()
	capacidad := exportacion.CapacidadCanonica()
	suma := sha256.Sum256(capacidad)
	material.capacidadHuella = hex.EncodeToString(suma[:])
	if err != nil ||
		material.cotejar(contexto, exportacion.ResumenCapacidad(), instante) != nil {
		return MaterialAutorizacionConsultaRRHH{}, ErrCapacidadConsultaRRHHInvalida
	}
	return material, nil
}

func (m *MaterialAutorizacionConsultaRRHH) cotejar(
	contexto ContextoConsultaRRHH,
	resumen puertosvec.ResumenCapacidadAtestacionAutorizacionV3,
	instante time.Time,
) error {
	datosSolicitud, errSolicitud := m.solicitud.Datos()
	datosVinculo, errVinculo := datosSolicitud.VinculoAutenticacionActor.Datos()
	decisionHuella, errDecision := dominiovec.HuellaSHA256DecisionAutorizacionV3(m.decision)
	motivoHuella, errMotivo := dominiovec.HuellaSHA256MotivoAutorizacionV2(
		datosSolicitud.ReferenciaMotivo,
	)
	recursoHuella, errRecurso := datosSolicitud.Recurso.HuellaContextoAutorizacionSHA256()
	decisionCanonica, errDecisionCanonica :=
		dominiovec.RepresentacionCanonicaDecisionAutorizacionV3(m.decision)
	motivoCanonico, errMotivoCanonico :=
		dominiovec.RepresentacionCanonicaMotivoAutorizacionV2(
			datosSolicitud.ReferenciaMotivo,
		)
	datosConfirmacion, errConfirmacion := m.confirmacion.Datos()
	concedida, _, errResultado := m.decision.Resultado()
	ordenRegistro, errOrden := puertosvec.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(
		m.solicitud, m.decision, datosSolicitud.ReferenciaMotivo, m.resultado,
	)
	if errSolicitud != nil || errVinculo != nil || errDecision != nil ||
		errMotivo != nil || errRecurso != nil || errDecisionCanonica != nil ||
		errMotivoCanonico != nil || errConfirmacion != nil ||
		errResultado != nil || errOrden != nil || !concedida ||
		m.exportacion.ValidarEstructura() != nil ||
		m.decision.ValidarPara(m.solicitud) != nil ||
		m.confirmacion.ValidarPara(ordenRegistro) != nil ||
		!m.confirmacion.DentroDeVentanaEn(instante) ||
		m.resultado.Validar() != nil ||
		datosSolicitud.VinculoAutenticacionActor.ValidarPara(m.resultado) != nil ||
		resumen.ValidarEstructura() != nil {
		return ErrCapacidadConsultaRRHHInvalida
	}
	if !bytes.Equal(m.exportacion.DecisionCanonica(), decisionCanonica) ||
		!bytes.Equal(m.exportacion.MotivoCanonico(), motivoCanonico) ||
		!bytes.Equal(
			m.exportacion.ContextoActorCanonico(),
			m.resultado.RepresentacionCanonica,
		) ||
		m.exportacion.PersonaVersion() !=
			m.resultado.Contexto.Instantanea.PersonaVersion ||
		m.exportacion.PerfilVersion() !=
			m.resultado.Contexto.Instantanea.PerfilVersion {
		return ErrCapacidadConsultaRRHHInvalida
	}
	if contexto.autenticacionRef != datosVinculo.AutenticacionRef ||
		contexto.autenticacionHuella != datosVinculo.AutenticacionHuellaSHA256 ||
		contexto.sesionRef != datosVinculo.SesionRef ||
		contexto.controlSesionRef != datosVinculo.ControlSesionRef ||
		contexto.controlSesionRevision != datosVinculo.ControlSesionRevision ||
		contexto.controlSesionHuellaSHA256 !=
			datosVinculo.ControlSesionHuellaSHA256 ||
		contexto.actorRef != datosVinculo.PrincipalID ||
		contexto.perfilRef != datosVinculo.PerfilActivoRef ||
		contexto.perfilVersion !=
			m.resultado.Contexto.Instantanea.PerfilVersion ||
		contexto.registroContextoRef != m.resultado.RegistroContextoRef ||
		contexto.contextoActorHuella != m.resultado.HuellaSHA256 ||
		contexto.validarEn(resumen.EmitidaEn()) != nil ||
		contexto.validarEn(datosConfirmacion.RegistradaEn) != nil {
		return ErrCapacidadConsultaRRHHInvalida
	}
	audienciaEsperada := audienciaConsumoConsultaRRHH(datosSolicitud.Accion)
	if validarParejaNominalMaterialConsultaRRHH(
		datosSolicitud, contexto,
	) != nil {
		return ErrCapacidadConsultaRRHHInvalida
	}
	if resumen.DecisionRef() != datosConfirmacion.DecisionRef ||
		resumen.DecisionHuellaSHA256() != decisionHuella ||
		resumen.DecisionHuellaSHA256() != datosConfirmacion.DecisionHuellaSHA256 ||
		resumen.MotivoHuellaSHA256() != motivoHuella ||
		resumen.ContextoRef() != m.resultado.RegistroContextoRef ||
		resumen.ContextoHuellaSHA256() != m.resultado.HuellaSHA256 ||
		resumen.Operacion() != datosSolicitud.Accion ||
		resumen.EfectoRef() != datosSolicitud.Recurso.Referencia ||
		resumen.EfectoHuellaSHA256() != recursoHuella ||
		audienciaEsperada == "" ||
		resumen.AudienciaConsumo() != audienciaEsperada ||
		resumen.EmitidaEn().Before(datosConfirmacion.EmitidaEn) ||
		resumen.EmitidaEn().Before(datosConfirmacion.RegistradaEn) ||
		!resumen.EmitidaEn().Before(datosConfirmacion.ValidaHasta) ||
		resumen.ExpiraEn().After(datosConfirmacion.ValidaHasta) ||
		instante.Before(resumen.EmitidaEn()) ||
		!instante.Before(resumen.ExpiraEn()) ||
		resumen.ExpiraEn().Sub(resumen.EmitidaEn()) >
			DuracionMaximaCapacidadConsultaRRHH {
		return ErrCapacidadConsultaRRHHInvalida
	}
	m.decisionHuella = decisionHuella
	m.motivoHuella = motivoHuella
	m.operacion = datosSolicitud.Accion
	m.recursoRef = datosSolicitud.Recurso.Referencia
	m.recursoHuella = recursoHuella
	m.emitidaEn = resumen.EmitidaEn()
	m.expiraEn = resumen.ExpiraEn()
	return nil
}

func audienciaConsumoConsultaRRHH(accion string) string {
	switch accion {
	case AccionConsultarCuadroRRHH:
		return AudienciaConsumoConsultaCuadroRRHHV3
	case AccionConsultarDetalleRRHH:
		return AudienciaConsumoConsultaDetalleRRHHV3
	default:
		return ""
	}
}

func validarParejaNominalMaterialConsultaRRHH(
	solicitud dominiovec.DatosSolicitudAutorizacionLigadaV3,
	contexto ContextoConsultaRRHH,
) error {
	dominio, finalidad, expedienteRef := "", "", ""
	switch solicitud.Accion {
	case AccionConsultarCuadroRRHH:
		dominio = DominioHuellaConsultaCuadroRRHH
		finalidad = FinalidadConsultarCuadroRRHH
	case AccionConsultarDetalleRRHH:
		dominio = DominioHuellaConsultaDetalleRRHH
		finalidad = FinalidadConsultarDetalleRRHH
		expedienteRef = solicitud.Recurso.Referencia
	default:
		return ErrCapacidadConsultaRRHHInvalida
	}
	if solicitud.Finalidad != finalidad {
		return ErrCapacidadConsultaRRHHInvalida
	}
	huella := solicitud.Recurso.Atributos[atributoHuellaConsultaRRHH]
	_, _, err := validarRecursoCapacidadConsultaRRHH(
		solicitud.Recurso, contexto, dominio, huella,
		solicitud.Accion, expedienteRef,
	)
	return err
}

func (m MaterialAutorizacionConsultaRRHH) validarPara(
	contexto ContextoConsultaRRHH,
	instante time.Time,
) error {
	huellaMaterial, err := m.exportacion.HuellaConjuntoSHA256()
	if err != nil ||
		contexto.validarEn(instante) != nil ||
		!m.confirmacion.DentroDeVentanaEn(instante) ||
		m.materialHuella != huellaMaterial {
		return ErrCapacidadConsultaRRHHInvalida
	}
	capacidad := m.exportacion.CapacidadCanonica()
	suma := sha256.Sum256(capacidad)
	if m.capacidadHuella != hex.EncodeToString(suma[:]) {
		return ErrCapacidadConsultaRRHHInvalida
	}
	copia := m
	return copia.cotejar(contexto, m.exportacion.ResumenCapacidad(), instante)
}

func (m MaterialAutorizacionConsultaRRHH) exportacionParaSQL() (
	puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3,
	error,
) {
	return clonarExportacionMaterialConsultaRRHH(m.exportacion, m.materialHuella)
}

func (m MaterialAutorizacionConsultaRRHH) clonarDefensivo() (
	MaterialAutorizacionConsultaRRHH,
	error,
) {
	if _, err := m.solicitud.Datos(); err != nil ||
		m.decision.ValidarPara(m.solicitud) != nil ||
		m.confirmacion.Validar() != nil {
		return MaterialAutorizacionConsultaRRHH{}, ErrCapacidadConsultaRRHHInvalida
	}
	resultado, err := m.resultado.Clonar()
	if err != nil {
		return MaterialAutorizacionConsultaRRHH{}, ErrCapacidadConsultaRRHHInvalida
	}
	exportacion, err := clonarExportacionMaterialConsultaRRHH(
		m.exportacion,
		m.materialHuella,
	)
	if err != nil {
		return MaterialAutorizacionConsultaRRHH{}, ErrCapacidadConsultaRRHHInvalida
	}
	copia := m
	copia.resultado = resultado
	copia.exportacion = exportacion
	return copia, nil
}

func clonarExportacionMaterialConsultaRRHH(
	exportacion puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3,
	huellaEsperada string,
) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	huella, err := exportacion.HuellaConjuntoSHA256()
	if err != nil || huella != huellaEsperada {
		return puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{},
			ErrCapacidadConsultaRRHHInvalida
	}
	copia, err := puertosvec.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(
		exportacion.CapacidadCanonica(), exportacion.ResumenCapacidad(),
		exportacion.DecisionCanonica(), exportacion.MotivoCanonico(),
		exportacion.ContextoActorCanonico(), exportacion.PersonaVersion(),
		exportacion.PerfilVersion(), exportacion.PayloadVECAD3(),
		exportacion.SobreCOSESign1(), exportacion.EvidenciaVerificacion(),
		exportacion.RaizPublicaSPKI(),
	)
	huellaCopia, errHuella := copia.HuellaConjuntoSHA256()
	if err != nil || errHuella != nil || huellaCopia != huellaEsperada {
		return puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{},
			ErrCapacidadConsultaRRHHInvalida
	}
	return copia, nil
}

func exportadorMaterialConsultaRRHHNulo(
	exportador puertosvec.ExportadorMaterialConsumoAutorizacionAtestadaV3,
) bool {
	if exportador == nil {
		return true
	}
	valor := reflect.ValueOf(exportador)
	switch valor.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map,
		reflect.Pointer, reflect.Slice:
		return valor.IsNil()
	default:
		return false
	}
}
