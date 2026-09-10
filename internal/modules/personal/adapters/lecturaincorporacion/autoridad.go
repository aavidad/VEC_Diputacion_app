package lecturaincorporacion

import (
	"bytes"
	"maps"
	"time"

	ctdomain "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
)

// validar coteja la cadena nominal completa, no implementa criptografía ni
// sustituye el consumo V3 transaccional y la revalidación viva del propietario.
func (a Autorizacion) validar(m Material, ahora time.Time) error {
	r, err := m.Recurso()
	if err != nil {
		return ErrDenegada
	}
	return validarAutorizacionLectura(a, m.contexto, m.preparadoEn, r, Audiencia, ahora)
}

// Compartido por contratos versionados: cada llamador fija recurso y audiencia,
// nunca se deducen de la autorización recibida.
func validarAutorizacionLectura(a Autorizacion, contexto ct.ContextoAutorizacionAltaV3, preparadoEn time.Time, r core.RecursoAutorizable, audiencia string, ahora time.Time) error {
	v, ev := contexto.Vinculo.Datos()
	if ev != nil || v.GarantiaObservada != core.AuthAssuranceHigh ||
		(v.Superficie != core.SuperficieAutenticacionInternaCorporativaV1 && v.Superficie != core.SuperficieAutenticacionAdministracionPrivilegiadaV1) ||
		!ctdomain.InstanteUTCCanonico(ahora) || !ctdomain.InstanteUTCCanonico(preparadoEn) || ahora.Before(preparadoEn) ||
		contexto.ValidarPara(ct.SolicitudResolverContextoAutorizacionAltaV3{AutenticacionRef: v.AutenticacionRef, SesionRef: v.SesionRef, PerfilRef: v.PerfilActivoRef}, ahora) != nil ||
		a.Exportacion.ValidarEstructura() != nil || a.Decision.ValidarPara(a.Solicitud) != nil {
		return ErrDenegada
	}
	s, es := a.Solicitud.Datos()
	c, ec := a.Confirmacion.Datos()
	concedida, _, ed := a.Decision.Resultado()
	if es != nil || ec != nil || ed != nil || !concedida || !s.VinculoAutenticacionActor.CoincideExactamenteCon(contexto.Vinculo) ||
		s.Accion != Accion || s.Finalidad != Finalidad || !a.Confirmacion.DentroDeVentanaEn(ahora) {
		return ErrDenegada
	}
	h, eh := r.HuellaContextoAutorizacionSHA256()
	hs, ehs := s.Recurso.HuellaContextoAutorizacionSHA256()
	dh, edh := core.HuellaSHA256DecisionAutorizacionV3(a.Decision)
	mh, emh := core.HuellaSHA256MotivoAutorizacionV2(s.ReferenciaMotivo)
	desde, hasta, et := a.Decision.VentanaValidez()
	x := a.Exportacion.ResumenCapacidad()
	if eh != nil || ehs != nil || edh != nil || emh != nil || et != nil || h != hs ||
		s.Recurso.Referencia != r.Referencia || s.Recurso.ModuloID != r.ModuloID || s.Recurso.Tipo != r.Tipo ||
		!maps.Equal(s.Recurso.Ambitos, r.Ambitos) || !maps.Equal(s.Recurso.Atributos, r.Atributos) ||
		c.DecisionHuellaSHA256 != dh || !c.EmitidaEn.Equal(desde) || !c.ValidaHasta.Equal(hasta) ||
		x.DecisionRef() != c.DecisionRef || x.DecisionHuellaSHA256() != dh || x.MotivoHuellaSHA256() != mh ||
		x.ContextoRef() != contexto.Resultado.RegistroContextoRef || x.ContextoHuellaSHA256() != contexto.Resultado.HuellaSHA256 ||
		x.Operacion() != Accion || x.EfectoRef() != r.Referencia || x.EfectoHuellaSHA256() != h || x.AudienciaConsumo() != audiencia ||
		ahora.Before(x.EmitidaEn()) || !ahora.Before(x.ExpiraEn()) || x.EmitidaEn().Before(desde) || x.ExpiraEn().After(hasta) {
		return ErrDenegada
	}
	dc, edc := core.RepresentacionCanonicaDecisionAutorizacionV3(a.Decision)
	mc, emc := core.RepresentacionCanonicaMotivoAutorizacionV2(s.ReferenciaMotivo)
	if edc != nil || emc != nil || !bytes.Equal(dc, a.Exportacion.DecisionCanonica()) || !bytes.Equal(mc, a.Exportacion.MotivoCanonico()) ||
		!bytes.Equal(contexto.Resultado.RepresentacionCanonica, a.Exportacion.ContextoActorCanonico()) ||
		a.Exportacion.PersonaVersion() != contexto.Resultado.Contexto.Instantanea.PersonaVersion ||
		a.Exportacion.PerfilVersion() != contexto.Resultado.Contexto.Instantanea.PerfilVersion {
		return ErrDenegada
	}
	return nil
}
