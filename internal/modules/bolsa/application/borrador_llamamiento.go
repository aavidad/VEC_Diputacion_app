package application

import (
	"bytes"
	"context"
	"errors"
	"reflect"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

var ErrServicioBorradorLlamamientoInvalido = errors.New("bolsa: servicio de borrador de llamamiento invalido")

type ServicioBorradorLlamamiento struct {
	contexto    puertosbolsa.ResolutorContextoBorradorLlamamiento
	autorizador puertosbolsa.AutorizadorBorradorLlamamientoV3
	transaccion puertosbolsa.TransaccionBorradorLlamamiento
	lector      puertosbolsa.LectorBorradorLlamamiento
}

func NuevoServicioBorradorLlamamiento(c puertosbolsa.ResolutorContextoBorradorLlamamiento, a puertosbolsa.AutorizadorBorradorLlamamientoV3, t puertosbolsa.TransaccionBorradorLlamamiento, l puertosbolsa.LectorBorradorLlamamiento) (*ServicioBorradorLlamamiento, error) {
	if nulo(c) || nulo(a) || nulo(t) || nulo(l) {
		return nil, ErrServicioBorradorLlamamientoInvalido
	}
	return &ServicioBorradorLlamamiento{contexto: c, autorizador: a, transaccion: t, lector: l}, nil
}

func (s *ServicioBorradorLlamamiento) Crear(ctx context.Context, solicitud puertosbolsa.SolicitudCrearBorradorLlamamiento) (puertosbolsa.ReciboBorradorLlamamiento, error) {
	if ctx == nil || s == nil || nulo(s.contexto) || nulo(s.autorizador) || nulo(s.transaccion) || solicitud.Validar() != nil {
		return puertosbolsa.ReciboBorradorLlamamiento{}, errors.Join(dominiovec.ErrAutorizacionDenegada, ErrServicioBorradorLlamamientoInvalido)
	}
	if err := ctx.Err(); err != nil {
		return puertosbolsa.ReciboBorradorLlamamiento{}, err
	}
	actor := solicitud.ResultadoContexto.Contexto
	resuelto, err := s.contexto.ResolverContextoBorradorLlamamiento(ctx, actor)
	if err != nil || resuelto.Validar() != nil {
		return puertosbolsa.ReciboBorradorLlamamiento{}, errorDependenciaBorradorLlamamiento(err)
	}
	huella, err := dominiobolsa.HuellaComandoCrearBorradorLlamamiento(actor.PersonaRef, resuelto.UnidadRef, resuelto.AmbitoRef, solicitud.ClaveIdempotencia, solicitud.Contenido)
	if err != nil {
		return puertosbolsa.ReciboBorradorLlamamiento{}, err
	}
	recurso := dominiovec.RecursoAutorizable{Referencia: "borrador-llamamiento:alta:" + huella, ModuloID: puertosbolsa.ModuloBorradorLlamamiento, Tipo: puertosbolsa.TipoRecursoBorradorLlamamiento, Ambitos: map[string]string{"unidad_ref": resuelto.UnidadRef, "ambito_ref": resuelto.AmbitoRef}}
	auth, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(dominiovec.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: solicitud.Vinculo, ReferenciaMotivo: solicitud.Motivo, Accion: puertosbolsa.AccionCrearBorradorLlamamientoInterno, Recurso: recurso, Finalidad: puertosbolsa.FinalidadCrearBorradorLlamamientoInterno, Correlacion: solicitud.Correlacion})
	if err != nil {
		return puertosbolsa.ReciboBorradorLlamamiento{}, errors.Join(dominiovec.ErrAutorizacionDenegada, err)
	}
	decision, confirmacion, exportador, err := s.autorizador.EmitirMaterialAutorizacionAtestadaV3(ctx, auth, solicitud.ResultadoContexto)
	if err != nil || nulo(exportador) {
		return puertosbolsa.ReciboBorradorLlamamiento{}, errorDependenciaBorradorLlamamiento(err)
	}
	if decision.ValidarPara(auth) != nil {
		return puertosbolsa.ReciboBorradorLlamamiento{}, errorDependenciaBorradorLlamamiento(ErrServicioBorradorLlamamientoInvalido)
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !materialAutorizacionBorradorLlamamientoExacto(auth, decision, confirmacion, solicitud.ResultadoContexto, solicitud.Motivo, material, puertosbolsa.AudienciaCrearBorradorLlamamientoInterno) {
		return puertosbolsa.ReciboBorradorLlamamiento{}, errorDependenciaBorradorLlamamiento(err)
	}
	borrador, err := dominiobolsa.NuevoBorradorLlamamiento("borrador-llamamiento:alta:"+huella, actor.PersonaRef, resuelto.UnidadRef, resuelto.AmbitoRef, solicitud.Contenido)
	if err != nil {
		return puertosbolsa.ReciboBorradorLlamamiento{}, err
	}
	recibo, err := s.transaccion.CrearBorradorLlamamiento(ctx, puertosbolsa.ComandoCrearBorradorLlamamiento{Borrador: borrador, ClaveIdempotencia: solicitud.ClaveIdempotencia, HuellaComandoSHA256: huella, SolicitudAutorizacion: auth, Decision: decision, Confirmacion: confirmacion, Material: material})
	if err != nil {
		return puertosbolsa.ReciboBorradorLlamamiento{}, err
	}
	if recibo.Validar() != nil || recibo.HuellaComandoSHA256 != huella || recibo.Borrador.PropietarioRef() != actor.PersonaRef || recibo.Borrador.UnidadRef() != resuelto.UnidadRef || recibo.Borrador.AmbitoRef() != resuelto.AmbitoRef || recibo.Borrador.Contenido() != solicitud.Contenido {
		return puertosbolsa.ReciboBorradorLlamamiento{}, ErrServicioBorradorLlamamientoInvalido
	}
	return recibo, nil
}

func (s *ServicioBorradorLlamamiento) Consultar(ctx context.Context, solicitud puertosbolsa.SolicitudConsultarBorradorLlamamiento) (puertosbolsa.ReciboBorradorLlamamiento, error) {
	if ctx == nil || s == nil || nulo(s.contexto) || nulo(s.autorizador) || nulo(s.lector) || solicitud.Validar() != nil {
		return puertosbolsa.ReciboBorradorLlamamiento{}, errors.Join(dominiovec.ErrAutorizacionDenegada, ErrServicioBorradorLlamamientoInvalido)
	}
	actor := solicitud.ResultadoContexto.Contexto
	resuelto, err := s.contexto.ResolverContextoBorradorLlamamiento(ctx, actor)
	if err != nil || resuelto.Validar() != nil {
		return puertosbolsa.ReciboBorradorLlamamiento{}, errorDependenciaBorradorLlamamiento(err)
	}
	recurso := dominiovec.RecursoAutorizable{Referencia: solicitud.BorradorRef, ModuloID: puertosbolsa.ModuloBorradorLlamamiento, Tipo: puertosbolsa.TipoRecursoBorradorLlamamiento, Ambitos: map[string]string{"unidad_ref": resuelto.UnidadRef, "ambito_ref": resuelto.AmbitoRef}}
	auth, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(dominiovec.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: solicitud.Vinculo, ReferenciaMotivo: solicitud.Motivo, Accion: puertosbolsa.AccionConsultarBorradorLlamamientoInterno, Recurso: recurso, Finalidad: puertosbolsa.FinalidadConsultarBorradorLlamamientoInterno, Correlacion: solicitud.Correlacion})
	if err != nil {
		return puertosbolsa.ReciboBorradorLlamamiento{}, errors.Join(dominiovec.ErrAutorizacionDenegada, err)
	}
	decision, confirmacion, exportador, err := s.autorizador.EmitirMaterialAutorizacionAtestadaV3(ctx, auth, solicitud.ResultadoContexto)
	if err != nil || nulo(exportador) || decision.ValidarPara(auth) != nil {
		return puertosbolsa.ReciboBorradorLlamamiento{}, errorDependenciaBorradorLlamamiento(err)
	}
	material, materialErr := exportador.ExportarMaterialParaConsumidor()
	if materialErr != nil || !materialAutorizacionBorradorLlamamientoExacto(auth, decision, confirmacion, solicitud.ResultadoContexto, solicitud.Motivo, material, puertosbolsa.AudienciaConsultarBorradorLlamamientoInterno) {
		return puertosbolsa.ReciboBorradorLlamamiento{}, errorDependenciaBorradorLlamamiento(materialErr)
	}
	recibo, err := s.lector.ObtenerBorradorLlamamiento(ctx, solicitud.BorradorRef, actor.PersonaRef, resuelto.UnidadRef, resuelto.AmbitoRef, auth, decision, confirmacion, material)
	if err != nil {
		return puertosbolsa.ReciboBorradorLlamamiento{}, err
	}
	if recibo.Validar() != nil || recibo.Borrador.Referencia() != solicitud.BorradorRef || recibo.Borrador.PropietarioRef() != actor.PersonaRef || recibo.Borrador.UnidadRef() != resuelto.UnidadRef || recibo.Borrador.AmbitoRef() != resuelto.AmbitoRef {
		return puertosbolsa.ReciboBorradorLlamamiento{}, ErrServicioBorradorLlamamientoInvalido
	}
	return recibo, nil
}

// errorDependenciaBorradorLlamamiento conserva exclusivamente una denegación
// positiva de la autoridad. Cualquier otro resultado de un resolutor, emisor
// o exportador se reduce al centinela de indisponibilidad: no puede filtrarse
// un centinela técnico de validación, recurso o conflicto hacia HTTP.
func errorDependenciaBorradorLlamamiento(err error) error {
	if errors.Is(err, dominiovec.ErrAutorizacionDenegada) || errors.Is(err, dominiovec.ErrPermissionDenied) {
		return err
	}
	return puertosbolsa.ErrFuenteBorradorLlamamientoNoDisponible
}

func materialAutorizacionBorradorLlamamientoExacto(
	solicitud dominiovec.SolicitudAutorizacionLigadaV3,
	decision dominiovec.DecisionAutorizacionLigadaV3,
	confirmacion puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3,
	resultado dominiovec.ResultadoContextoActorRegistradoV2,
	motivo dominiovec.ReferenciaEntradaCatalogo,
	material puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3,
	audiencia string,
) bool {
	orden, err := puertosvec.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(solicitud, decision, motivo, resultado)
	if err != nil || confirmacion.ValidarPara(orden) != nil || material.ValidarEstructura() != nil {
		return false
	}
	datosSolicitud, err := solicitud.Datos()
	decisionCanonica, errDecision := dominiovec.RepresentacionCanonicaDecisionAutorizacionV3(decision)
	motivoCanonico, errMotivo := dominiovec.RepresentacionCanonicaMotivoAutorizacionV2(motivo)
	huellaDecision, errHuellaDecision := dominiovec.HuellaSHA256DecisionAutorizacionV3(decision)
	huellaMotivo, errHuellaMotivo := dominiovec.HuellaSHA256MotivoAutorizacionV2(motivo)
	huellaRecurso, errHuellaRecurso := datosSolicitud.Recurso.HuellaContextoAutorizacionSHA256()
	datosConfirmacion, errConfirmacion := confirmacion.Datos()
	resumen := material.ResumenCapacidad()
	return err == nil && errDecision == nil && errMotivo == nil && errHuellaDecision == nil &&
		errHuellaMotivo == nil && errHuellaRecurso == nil && errConfirmacion == nil &&
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
		resumen.Operacion() == datosSolicitud.Accion &&
		resumen.EfectoRef() == datosSolicitud.Recurso.Referencia &&
		resumen.EfectoHuellaSHA256() == huellaRecurso &&
		resumen.AudienciaConsumo() == audiencia &&
		!resumen.EmitidaEn().Before(datosConfirmacion.EmitidaEn) &&
		!resumen.EmitidaEn().Before(datosConfirmacion.RegistradaEn) &&
		resumen.EmitidaEn().Before(datosConfirmacion.ValidaHasta) &&
		!resumen.ExpiraEn().After(datosConfirmacion.ValidaHasta)
}

func nulo(x any) bool {
	if x == nil {
		return true
	}
	v := reflect.ValueOf(x)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	}
	return false
}
