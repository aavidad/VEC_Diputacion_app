// Package application coordina los casos de uso de Selección: publicación
// de convocatorias, solicitudes propias de la persona y consulta de RRHH.
// Solo conoce el dominio y los puertos; es neutral al cliente (web,
// escritorio, CLI o MCP construyen la misma Orden desde su frontera).
package application

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"regexp"
	"time"

	"vec-diputacion-granada/internal/modules/seleccion/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

// Orden procede solo de la frontera autenticada (certificado hoy; Cl@ve o
// DNIe mañana por otro adaptador): contexto atestado, vínculo de la
// autenticación, motivo y correlación. Ambitos son los del perfil activo que
// acotan el recurso; el cliente no aporta ninguno de estos campos.
type Orden struct {
	ResultadoContexto dominiovec.ResultadoContextoActorRegistradoV2
	Vinculo           dominiovec.VinculoAutenticacionActorV2
	Motivo            dominiovec.ReferenciaEntradaCatalogo
	Correlacion       dominiovec.ReferenciaCorrelacionAutorizacionV2
	Ambitos           map[string]string
}

var (
	personaRefValida   = regexp.MustCompile(`^per_[A-Za-z0-9_-]{22,128}$`)
	claveValida        = regexp.MustCompile(`^[A-Za-z0-9._:-]{8,128}$`)
	solicitudRefValida = regexp.MustCompile(`^sol_[A-Za-z0-9_-]{22,64}$`)
)

func nula(v any) bool {
	if v == nil {
		return true
	}
	x := reflect.ValueOf(v)
	switch x.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return x.IsNil()
	}
	return false
}

func denegar(err error) error {
	return errors.Join(dominiovec.ErrAutorizacionDenegada, err)
}

// validarOrden exige una orden vigente de la superficie esperada, sin método
// de demostración, y devuelve la persona del contexto atestado.
func validarOrden(o Orden, superficie dominiovec.SuperficieAutenticacionActorV1, ahora time.Time) (dominiovec.ResultadoContextoActorRegistradoV2, string, error) {
	r, err := o.ResultadoContexto.Clonar()
	if err != nil {
		return dominiovec.ResultadoContextoActorRegistradoV2{}, "", denegar(err)
	}
	d, err := o.Vinculo.Datos()
	if err != nil || ahora.IsZero() || r.Contexto.Principal.AuthMethod == dominiovec.AuthMethodDemo || d.MetodoObservado == dominiovec.AuthMethodDemo ||
		d.Superficie != superficie || o.Vinculo.ValidarPara(r) != nil || !o.Vinculo.VigenteEn(ahora, r) ||
		!dominiovec.ReferenciaMotivoAutorizacionV2Valida(o.Motivo) || o.Correlacion.Validar() != nil ||
		!personaRefValida.MatchString(r.Contexto.PersonaRef) || len(o.Ambitos) == 0 {
		return dominiovec.ResultadoContextoActorRegistradoV2{}, "", denegar(err)
	}
	return r, r.Contexto.PersonaRef, nil
}

// autorizador reúne la emisión de material V3 común a los dos servicios.
type autorizador struct {
	emisor ports.EmisorMaterialV3
	reloj  ports.Reloj
}

// autorizar pide la decisión para la acción sobre el recurso, emite el
// material y comprueba que corresponde exactamente a la solicitud y a la
// audiencia de la acción que lo consumirá.
func (a autorizador) autorizar(ctx context.Context, o Orden, r dominiovec.ResultadoContextoActorRegistradoV2, accion, audiencia, finalidad string, recurso dominiovec.RecursoAutorizable) (ports.MaterialConsumoV3, error) {
	if recurso.Validar() != nil {
		return nil, denegar(ports.ErrDatosNoValidos)
	}
	solicitud, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(dominiovec.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: o.Vinculo, ReferenciaMotivo: o.Motivo, Accion: accion, Recurso: recurso,
		Finalidad: finalidad, Correlacion: o.Correlacion,
	})
	if err != nil {
		return nil, denegar(err)
	}
	decision, confirmacion, exportador, err := a.emisor.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, r)
	if err != nil {
		if errors.Is(err, dominiovec.ErrAutorizacionDenegada) || errors.Is(err, dominiovec.ErrPermissionDenied) ||
			errors.Is(err, puertosvec.ErrDenegacionExplicitaAutorizacionLigadaV3) {
			return nil, denegar(err)
		}
		return nil, errors.Join(ports.ErrNoDisponible, err)
	}
	if nula(exportador) || decision.ValidarPara(solicitud) != nil {
		return nil, denegar(nil)
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil {
		return nil, errors.Join(ports.ErrNoDisponible, err)
	}
	if err := verificarMaterial(solicitud, decision, confirmacion, r, o.Motivo, material, audiencia, a.reloj.Ahora().UTC()); err != nil {
		return nil, denegar(err)
	}
	return material, nil
}

// errMaterialInexacto: el material no corresponde a la decisión, al motivo,
// al contexto o a la audiencia de la acción.
var errMaterialInexacto = errors.New("seleccion: material de autorizacion inexacto")

// verificarMaterial coteja el material exportado con la decisión, el motivo,
// el contexto y la audiencia; su vigencia cubre el instante actual.
func verificarMaterial(s dominiovec.SolicitudAutorizacionLigadaV3, d dominiovec.DecisionAutorizacionLigadaV3, c puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3,
	r dominiovec.ResultadoContextoActorRegistradoV2, motivo dominiovec.ReferenciaEntradaCatalogo, m puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, audiencia string, ahora time.Time) error {
	orden, err := puertosvec.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(s, d, motivo, r)
	if err != nil {
		return err
	}
	if err := c.ValidarPara(orden); err != nil {
		return err
	}
	if err := m.ValidarEstructura(); err != nil {
		return err
	}
	datos, err := s.Datos()
	if err != nil {
		return err
	}
	decisionCanonica, err := dominiovec.RepresentacionCanonicaDecisionAutorizacionV3(d)
	if err != nil {
		return err
	}
	motivoCanonico, err := dominiovec.RepresentacionCanonicaMotivoAutorizacionV2(motivo)
	if err != nil {
		return err
	}
	huellaDecision, err := dominiovec.HuellaSHA256DecisionAutorizacionV3(d)
	if err != nil {
		return err
	}
	huellaMotivo, err := dominiovec.HuellaSHA256MotivoAutorizacionV2(motivo)
	if err != nil {
		return err
	}
	huellaRecurso, err := datos.Recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		return err
	}
	confirmada, err := c.Datos()
	if err != nil {
		return err
	}
	resumen := m.ResumenCapacidad()
	if !bytes.Equal(m.DecisionCanonica(), decisionCanonica) || !bytes.Equal(m.MotivoCanonico(), motivoCanonico) ||
		!bytes.Equal(m.ContextoActorCanonico(), r.RepresentacionCanonica) ||
		m.PersonaVersion() != r.Contexto.Instantanea.PersonaVersion || m.PerfilVersion() != r.Contexto.Instantanea.PerfilVersion ||
		resumen.DecisionRef() != confirmada.DecisionRef || resumen.DecisionHuellaSHA256() != huellaDecision ||
		resumen.MotivoHuellaSHA256() != huellaMotivo || resumen.ContextoRef() != r.RegistroContextoRef ||
		resumen.ContextoHuellaSHA256() != r.HuellaSHA256 || resumen.Operacion() != datos.Accion ||
		resumen.EfectoRef() != datos.Recurso.Referencia || resumen.EfectoHuellaSHA256() != huellaRecurso ||
		resumen.AudienciaConsumo() != audiencia || ahora.Before(resumen.EmitidaEn()) || !ahora.Before(resumen.ExpiraEn()) {
		return errMaterialInexacto
	}
	return nil
}

// clonarAmbitos copia los ámbitos del perfil para el recurso.
func clonarAmbitos(a map[string]string) map[string]string {
	copia := make(map[string]string, len(a))
	for k, v := range a {
		copia[k] = v
	}
	return copia
}
