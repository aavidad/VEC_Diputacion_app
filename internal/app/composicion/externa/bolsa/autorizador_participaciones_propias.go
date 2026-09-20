// Package bolsa compone capacidades externas nominales del portal de Bolsa.
package bolsa

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"reflect"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

var ErrAutorizadorParticipacionesPropiasInvalido = errors.New("bolsa externa: autorizador de participaciones propias invalido")

// EmisorMaterialParticipacionesPropias es la autoridad V3 ya compuesta. Este
// adaptador no concede permisos ni conserva decisiones entre peticiones.
type EmisorMaterialParticipacionesPropias interface {
	EmitirMaterialAutorizacionAtestadaV3(context.Context, dominiovec.SolicitudAutorizacionLigadaV3, dominiovec.ResultadoContextoActorRegistradoV2) (dominiovec.DecisionAutorizacionLigadaV3, puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3, puertosvec.ExportadorMaterialConsumoAutorizacionAtestadaV3, error)
}

// GeneradorCorrelacionParticipacionesPropias debe usar el generador
// criptografico gobernado de la composicion.
type GeneradorCorrelacionParticipacionesPropias interface {
	NuevaReferenciaCorrelacionAutorizacionV2(context.Context) (string, error)
}

// AutorizadorParticipacionesPropiasV3 transforma exclusivamente el par
// vínculo/resultado ya resuelto por la frontera de identidad en material AD3.
type AutorizadorParticipacionesPropiasV3 struct {
	emisor     EmisorMaterialParticipacionesPropias
	correlador GeneradorCorrelacionParticipacionesPropias
	motivo     dominiovec.ReferenciaEntradaCatalogo
}

func NuevoAutorizadorParticipacionesPropiasV3(emisor EmisorMaterialParticipacionesPropias, correlador GeneradorCorrelacionParticipacionesPropias, motivo dominiovec.ReferenciaEntradaCatalogo) (*AutorizadorParticipacionesPropiasV3, error) {
	if dependenciaNula(emisor) || dependenciaNula(correlador) || !dominiovec.ReferenciaMotivoAutorizacionV2Valida(motivo) {
		return nil, ErrAutorizadorParticipacionesPropiasInvalido
	}
	return &AutorizadorParticipacionesPropiasV3{emisor: emisor, correlador: correlador, motivo: motivo}, nil
}

// AutorizarOperacion conserva exactamente el vínculo y resultado de una sola
// resolución de identidad. No vuelve a resolver el actor ni acepta candidato,
// finalidad o motivo de la petición web.
func (a *AutorizadorParticipacionesPropiasV3) AutorizarOperacion(ctx context.Context, vinculo dominiovec.VinculoAutenticacionActorV2, resultado dominiovec.ResultadoContextoActorRegistradoV2, accion string, recurso dominiovec.RecursoAutorizable) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	vacio := puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	if a == nil || ctx == nil || ctx.Err() != nil || dependenciaNula(a.emisor) || dependenciaNula(a.correlador) || !dominiovec.ReferenciaMotivoAutorizacionV2Valida(a.motivo) ||
		vinculo.ValidarPara(resultado) != nil || !contratoB11Valido(accion, recurso) {
		return vacio, ErrAutorizadorParticipacionesPropiasInvalido
	}
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(ctx, a.correlador)
	if err != nil {
		return vacio, errors.Join(ErrAutorizadorParticipacionesPropiasInvalido, err)
	}
	solicitud, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(dominiovec.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: vinculo, ReferenciaMotivo: a.motivo, Accion: accion,
		Recurso: recurso, Finalidad: puertosbolsa.FinalidadConsultarParticipacionesPropias, Correlacion: correlacion,
	})
	if err != nil {
		return vacio, errors.Join(ErrAutorizadorParticipacionesPropiasInvalido, err)
	}
	decision, confirmacion, exportador, err := a.emisor.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, resultado)
	if err != nil || dependenciaNula(exportador) || ctx.Err() != nil {
		return vacio, errors.Join(ErrAutorizadorParticipacionesPropiasInvalido, err, ctx.Err())
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || ctx.Err() != nil || !materialB11Valido(material, solicitud, decision, confirmacion, resultado, accion, recurso) {
		return vacio, errors.Join(ErrAutorizadorParticipacionesPropiasInvalido, err, ctx.Err())
	}
	return material, nil
}

func contratoB11Valido(accion string, recurso dominiovec.RecursoAutorizable) bool {
	if accion != puertosbolsa.AccionConsultarParticipacionesPropias || recurso.Validar() != nil ||
		recurso.ModuloID != puertosbolsa.ModuloParticipacionesPropiasBolsa || recurso.Tipo != puertosbolsa.TipoRecursoParticipacionesPropias ||
		recurso.Atributos["finalidad"] != puertosbolsa.FinalidadConsultarParticipacionesPropias || len(recurso.Atributos) != 1 || len(recurso.Ambitos) != 1 {
		return false
	}
	candidato := recurso.Ambitos["candidato_ref"]
	return candidato != "" && recurso.Referencia == "candidato:"+candidato
}

func materialB11Valido(material puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, solicitud dominiovec.SolicitudAutorizacionLigadaV3, decision dominiovec.DecisionAutorizacionLigadaV3, confirmacion puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3, resultado dominiovec.ResultadoContextoActorRegistradoV2, accion string, recurso dominiovec.RecursoAutorizable) bool {
	if material.ValidarEstructura() != nil || decision.ValidarPara(solicitud) != nil || resultado.Validar() != nil {
		return false
	}
	datos, err := solicitud.Datos()
	if err != nil || datos.VinculoAutenticacionActor.ValidarPara(resultado) != nil || datos.Accion != accion || datos.Finalidad != puertosbolsa.FinalidadConsultarParticipacionesPropias || !reflect.DeepEqual(datos.Recurso, recurso) {
		return false
	}
	concedida, _, err := decision.Resultado()
	orden, errOrden := puertosvec.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(solicitud, decision, datos.ReferenciaMotivo, resultado)
	if err != nil || errOrden != nil || !concedida || confirmacion.ValidarPara(orden) != nil {
		return false
	}
	confirmacionDatos, err := confirmacion.Datos()
	if err != nil || !confirmacion.DentroDeVentanaEn(confirmacionDatos.RegistradaEn) {
		return false
	}
	decisionCanonica, errDecision := dominiovec.RepresentacionCanonicaDecisionAutorizacionV3(decision)
	motivoCanonico, errMotivo := dominiovec.RepresentacionCanonicaMotivoAutorizacionV2(datos.ReferenciaMotivo)
	huellaRecurso, errRecurso := recurso.HuellaContextoAutorizacionSHA256()
	proyeccion, errProyeccion := dominiovec.ParsearMensajeAtestacionAutorizacionV3NoAutoritativo(material.PayloadVECAD3())
	if errDecision != nil || errMotivo != nil || errRecurso != nil || errProyeccion != nil {
		return false
	}
	cabecera, errCabecera := proyeccion.Cabecera()
	mensaje, errMensaje := dominiovec.SerializarMensajeAtestacionAutorizacionV3(cabecera, decision, datos.ReferenciaMotivo, resultado)
	if errCabecera != nil || errMensaje != nil || !bytes.Equal(mensaje, material.PayloadVECAD3()) {
		return false
	}
	huellaDecision := sha256.Sum256(decisionCanonica)
	huellaMotivo := sha256.Sum256(motivoCanonico)
	resumen := material.ResumenCapacidad()
	return resumen.DecisionRef() == confirmacionDatos.DecisionRef && resumen.DecisionHuellaSHA256() == confirmacionDatos.DecisionHuellaSHA256 &&
		bytes.Equal(decisionCanonica, material.DecisionCanonica()) && bytes.Equal(motivoCanonico, material.MotivoCanonico()) && bytes.Equal(resultado.RepresentacionCanonica, material.ContextoActorCanonico()) &&
		resumen.DecisionHuellaSHA256() == hex.EncodeToString(huellaDecision[:]) && resumen.MotivoHuellaSHA256() == hex.EncodeToString(huellaMotivo[:]) &&
		resumen.Operacion() == accion && resumen.EfectoRef() == recurso.Referencia && resumen.EfectoHuellaSHA256() == huellaRecurso && resumen.AudienciaConsumo() == puertosbolsa.AudienciaParticipacionesPropias &&
		resumen.ContextoRef() == resultado.RegistroContextoRef && resumen.ContextoHuellaSHA256() == resultado.HuellaSHA256 &&
		material.PersonaVersion() == resultado.Contexto.Instantanea.PersonaVersion && material.PerfilVersion() == resultado.Contexto.Instantanea.PerfilVersion &&
		!confirmacionDatos.RegistradaEn.Before(resumen.EmitidaEn()) && confirmacionDatos.RegistradaEn.Before(resumen.ExpiraEn())
}

func dependenciaNula(dependencia any) bool {
	if dependencia == nil {
		return true
	}
	v := reflect.ValueOf(dependencia)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}
