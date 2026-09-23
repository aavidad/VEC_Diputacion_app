package composicion

import (
	"bytes"
	"context"
	"errors"

	dietasapp "vec-diputacion-granada/internal/modules/dietas/application"
	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var ErrEmisorBorradorNoDisponible = errors.New("dietas: emisor de autorización no disponible")

// EmisorMaterialBorradorV3 es la autoridad común V3, ya compuesta con PDP,
// firmante y verificador. Dietas solo recibe el material exportable nominal.
type EmisorMaterialBorradorV3 interface {
	EmitirMaterialAutorizacionAtestadaV3(context.Context, vecdomain.SolicitudAutorizacionLigadaV3, vecdomain.ResultadoContextoActorRegistradoV2) (vecdomain.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vecports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error)
}

type EmisorAutorizacionBorrador struct {
	emisor EmisorMaterialBorradorV3
	motivo vecdomain.ReferenciaEntradaCatalogo
}

func NuevoEmisorAutorizacionBorrador(emisor EmisorMaterialBorradorV3, motivo vecdomain.ReferenciaEntradaCatalogo) (*EmisorAutorizacionBorrador, error) {
	if nulo(emisor) || motivo.Validar() != nil {
		return nil, ErrEmisorBorradorNoDisponible
	}
	return &EmisorAutorizacionBorrador{emisor: emisor, motivo: motivo}, nil
}

func (e *EmisorAutorizacionBorrador) AutorizarBorradorPropio(ctx context.Context, base IdentidadRegistradaBorrador, operacion dietasports.SolicitudOperacionBorrador, sello dietasports.RevalidacionRelacionPersonal, efecto dietasports.EfectoAutorizacionBorrador) (dietasports.AutorizacionBorradorDurable, error) {
	vacio := dietasports.AutorizacionBorradorDurable{}
	if e == nil || nulo(e.emisor) || ctx == nil || ctx.Err() != nil || base.Contexto.Validar() != nil || base.Vinculo.ValidarPara(base.Contexto) != nil || len(efecto.Material) == 0 {
		return vacio, dietasports.ErrAccesoBorradorDenegado
	}
	if operacion.Operacion != dietasports.OperacionCrearBorrador && operacion.Operacion != dietasports.OperacionConsultarBorrador {
		return vacio, dietasports.ErrAccesoBorradorDenegado
	}
	accion, recurso, finalidad := contratoSolicitud(operacion)
	if accion == "" || recurso != efecto.Recurso.Referencia || efecto.Recurso.ModuloID != dietasports.ModuloDietas || efecto.Recurso.Tipo != dietasports.TipoRecursoComisionBorrador || sello.RelacionRef == "" || sello.PersonaRef != base.Contexto.Contexto.PersonaRef {
		return vacio, dietasports.ErrAccesoBorradorDenegado
	}
	correlacion, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridadvec.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return vacio, dietasports.ErrAccesoBorradorDenegado
	}
	solicitud, err := vecdomain.NuevaSolicitudAutorizacionLigadaV3(vecdomain.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: base.Vinculo,
		ReferenciaMotivo:          e.motivo,
		Accion:                    accion,
		Recurso:                   efecto.Recurso,
		Finalidad:                 finalidad,
		Correlacion:               correlacion,
	})
	if err != nil {
		return vacio, dietasports.ErrAccesoBorradorDenegado
	}
	decision, confirmacion, exportador, err := e.emisor.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, base.Contexto)
	if err != nil || decision.ValidarPara(solicitud) != nil || nulo(exportador) {
		return vacio, dietasports.ErrAccesoBorradorDenegado
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil {
		return vacio, dietasports.ErrAccesoBorradorDenegado
	}
	audiencia := "vec_dietas.borrador_propio.consultar.v1"
	if accion == dietasapp.AccionCrearBorradorPropio {
		audiencia = "vec_dietas.borrador_propio.crear.v1"
	}
	if !materialBorradorExacto(solicitud, decision, confirmacion, base.Contexto, e.motivo, material, audiencia) {
		return vacio, dietasports.ErrAccesoBorradorDenegado
	}
	return dietasports.AutorizacionBorradorDurable{Material: material, Accion: accion, RecursoRef: recurso, Finalidad: finalidad, Revalidacion: sello}, nil
}

func materialBorradorExacto(solicitud vecdomain.SolicitudAutorizacionLigadaV3, decision vecdomain.DecisionAutorizacionLigadaV3, confirmacion vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, resultado vecdomain.ResultadoContextoActorRegistradoV2, motivo vecdomain.ReferenciaEntradaCatalogo, material vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, audiencia string) bool {
	orden, err := vecports.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(solicitud, decision, motivo, resultado)
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

var _ ProveedorAutorizacionBorrador = (*EmisorAutorizacionBorrador)(nil)
