package ports

import (
	"bytes"

	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

// MaterialAtestadoLigadoV3 comprueba la preimagen exportada antes de entregarla
// a un consumidor nominal. La concesión y el consumo siguen siendo autoridad
// del núcleo AD3; este cotejo evita que el adaptador mezcle dos operaciones.
func MaterialAtestadoLigadoV3(solicitud vecdomain.SolicitudAutorizacionLigadaV3, decision vecdomain.DecisionAutorizacionLigadaV3, confirmacion ConfirmacionRegistroConcesionAutorizacionLigadaV3, resultado vecdomain.ResultadoContextoActorRegistradoV2, motivo vecdomain.ReferenciaEntradaCatalogo, material ExportacionMaterialConsumoAutorizacionAtestadaV3, audiencia string) bool {
	orden, err := NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(solicitud, decision, motivo, resultado)
	if err != nil || confirmacion.ValidarPara(orden) != nil || material.ValidarEstructura() != nil {
		return false
	}
	datos, err := solicitud.Datos()
	decisionCanonica, errDecision := vecdomain.RepresentacionCanonicaDecisionAutorizacionV3(decision)
	motivoCanonico, errMotivo := vecdomain.RepresentacionCanonicaMotivoAutorizacionV2(motivo)
	huellaDecision, errHuellaDecision := vecdomain.HuellaSHA256DecisionAutorizacionV3(decision)
	huellaMotivo, errHuellaMotivo := vecdomain.HuellaSHA256MotivoAutorizacionV2(motivo)
	huellaRecurso, errHuellaRecurso := datos.Recurso.HuellaContextoAutorizacionSHA256()
	datosConfirmacion, errConfirmacion := confirmacion.Datos()
	resumen := material.ResumenCapacidad()
	return err == nil && errDecision == nil && errMotivo == nil && errHuellaDecision == nil && errHuellaMotivo == nil && errHuellaRecurso == nil && errConfirmacion == nil &&
		bytes.Equal(material.DecisionCanonica(), decisionCanonica) &&
		bytes.Equal(material.MotivoCanonico(), motivoCanonico) &&
		bytes.Equal(material.ContextoActorCanonico(), resultado.RepresentacionCanonica) &&
		material.PersonaVersion() == resultado.Contexto.Instantanea.PersonaVersion &&
		material.PerfilVersion() == resultado.Contexto.Instantanea.PerfilVersion &&
		resumen.DecisionRef() == datosConfirmacion.DecisionRef &&
		resumen.DecisionHuellaSHA256() == huellaDecision &&
		resumen.MotivoHuellaSHA256() == huellaMotivo &&
		resumen.ContextoRef() == resultado.RegistroContextoRef &&
		resumen.ContextoHuellaSHA256() == resultado.HuellaSHA256 &&
		resumen.Operacion() == datos.Accion &&
		resumen.EfectoRef() == datos.Recurso.Referencia &&
		resumen.EfectoHuellaSHA256() == huellaRecurso &&
		resumen.AudienciaConsumo() == audiencia &&
		!resumen.EmitidaEn().Before(datosConfirmacion.EmitidaEn) &&
		!resumen.EmitidaEn().Before(datosConfirmacion.RegistradaEn) &&
		resumen.EmitidaEn().Before(datosConfirmacion.ValidaHasta) &&
		!resumen.ExpiraEn().After(datosConfirmacion.ValidaHasta)
}
