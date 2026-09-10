package contrataciontemporal

import (
	"bytes"
	"context"
	"time"

	ctdomain "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type AutoridadAlta struct {
	OrganizacionRef string
	Contexto        ctports.ContextoAutorizacionAltaV3
}

func (a AutoridadAlta) validar(ahora time.Time) error {
	v, err := a.Contexto.Vinculo.Datos()
	if err != nil || !ctdomain.ReferenciaOpacaValida(a.OrganizacionRef) || v.GarantiaObservada != core.AuthAssuranceHigh ||
		(v.Superficie != core.SuperficieAutenticacionInternaCorporativaV1 && v.Superficie != core.SuperficieAutenticacionAdministracionPrivilegiadaV1) ||
		a.Contexto.ValidarPara(ctports.SolicitudResolverContextoAutorizacionAltaV3{AutenticacionRef: v.AutenticacionRef, SesionRef: v.SesionRef, PerfilRef: v.PerfilActivoRef}, ahora) != nil {
		return ErrDenegado
	}
	return nil
}

// El proveedor es una dependencia de composición confiable: ni CapacidadRef
// ni un resumen estructural sustituyen la decisión concedida y registrada V3.
type ProveedorNominalAlta interface {
	ResolverAutoridad(context.Context, PreparacionAlta) (AutoridadAlta, error)
	AutorizarAlta(context.Context, MaterialAlta) (AutorizacionAlta, error)
}

type AutorizacionAlta struct {
	Solicitud    core.SolicitudAutorizacionLigadaV3
	Decision     core.DecisionAutorizacionLigadaV3
	Confirmacion vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3
	Exportacion  vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

func (a AutorizacionAlta) validar(m MaterialAlta, actor AutoridadAlta, ahora time.Time) error {
	if actor.validar(ahora) != nil || a.Exportacion.ValidarEstructura() != nil || a.Decision.ValidarPara(a.Solicitud) != nil {
		return ErrDenegado
	}
	s, e1 := a.Solicitud.Datos()
	c, e2 := a.Confirmacion.Datos()
	concedida, _, e3 := a.Decision.Resultado()
	if e1 != nil || e2 != nil || e3 != nil || !concedida || !a.Confirmacion.DentroDeVentanaEn(ahora) ||
		!s.VinculoAutenticacionActor.CoincideExactamenteCon(actor.Contexto.Vinculo) || s.Accion != AccionAltaEjercicio || s.Finalidad != FinalidadAltaEjercicio {
		return ErrDenegado
	}
	recurso, err := RecursoAltaEjercicio(m)
	if err != nil {
		return ErrDenegado
	}
	h, err := recurso.HuellaContextoAutorizacionSHA256()
	hs, errS := s.Recurso.HuellaContextoAutorizacionSHA256()
	if err != nil || errS != nil || h != hs {
		return ErrDenegado
	}
	dh, errD := core.HuellaSHA256DecisionAutorizacionV3(a.Decision)
	mh, errM := core.HuellaSHA256MotivoAutorizacionV2(s.ReferenciaMotivo)
	de, dhasta, errV := a.Decision.VentanaValidez()
	r := a.Exportacion.ResumenCapacidad()
	if errD != nil || errM != nil || errV != nil || c.DecisionHuellaSHA256 != dh || !c.EmitidaEn.Equal(de) || !c.ValidaHasta.Equal(dhasta) ||
		r.DecisionRef() != c.DecisionRef || r.DecisionHuellaSHA256() != dh || r.MotivoHuellaSHA256() != mh ||
		r.ContextoRef() != actor.Contexto.Resultado.RegistroContextoRef || r.ContextoHuellaSHA256() != actor.Contexto.Resultado.HuellaSHA256 ||
		r.Operacion() != AccionAltaEjercicio || r.EfectoRef() != recurso.Referencia || r.EfectoHuellaSHA256() != h || r.AudienciaConsumo() != AudienciaAltaEjercicio ||
		ahora.Before(r.EmitidaEn()) || !ahora.Before(r.ExpiraEn()) || r.EmitidaEn().Before(de) || r.ExpiraEn().After(dhasta) {
		return ErrDenegado
	}
	dc, errD := core.RepresentacionCanonicaDecisionAutorizacionV3(a.Decision)
	mc, errM := core.RepresentacionCanonicaMotivoAutorizacionV2(s.ReferenciaMotivo)
	if errD != nil || errM != nil || !bytes.Equal(dc, a.Exportacion.DecisionCanonica()) || !bytes.Equal(mc, a.Exportacion.MotivoCanonico()) ||
		!bytes.Equal(actor.Contexto.Resultado.RepresentacionCanonica, a.Exportacion.ContextoActorCanonico()) ||
		a.Exportacion.PersonaVersion() != actor.Contexto.Resultado.Contexto.Instantanea.PersonaVersion || a.Exportacion.PerfilVersion() != actor.Contexto.Resultado.Contexto.Instantanea.PerfilVersion {
		return ErrDenegado
	}
	return nil
}

// OrdenAlta solo nace del consumidor tras preparar fuente y autoridad exactas.
// Transporta el material nominal al commit; sus accesores no exponen mutabilidad.
type OrdenAlta struct {
	material     MaterialAlta
	actor        AutoridadAlta
	autorizacion AutorizacionAlta
	preparadaEn  time.Time
}

func (o OrdenAlta) Material() MaterialAlta { return o.material }
func (o OrdenAlta) Exportacion() vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	return o.autorizacion.Exportacion
}
func (o OrdenAlta) ValidarEn(ahora time.Time) error {
	v, err := o.actor.Contexto.Vinculo.Datos()
	if err != nil || !ctdomain.InstanteUTCCanonico(o.preparadaEn) || !ctdomain.InstanteUTCCanonico(ahora) || ahora.Before(o.preparadaEn) ||
		o.material.Validar() != nil || o.material.ActorRef != v.PrincipalID || o.material.PerfilRef != v.PerfilActivoRef || o.material.OrganizacionRef != o.actor.OrganizacionRef {
		return ErrDenegado
	}
	return o.autorizacion.validar(o.material, o.actor, ahora)
}

// TransaccionAltaPersonal debe verificar canon/COSE/raíz gobernada/revocación,
// revalidar OrdenAlta con reloj propio, consumir la concesión y escribir relación,
// ocupación, recibo, auditoría y outbox en UN commit de Personal. No escribe CT.
// Bloquea idempotencia y compara TODO el material/actor/perfil: divergencia es
// ErrConflicto. Replay consume permiso fresco y devuelve el recibo original
// íntegro del mismo historial verificado, sin nuevas relación/ocupación.
// Error/rollback/commit incierto: error y resultado cero; no reintenta por dentro.
// Ningún resumen Go ni doble de prueba acredita esa verificación durable.
type TransaccionAltaPersonal interface {
	RegistrarORecuperarAlta(context.Context, OrdenAlta) (ResultadoTransaccionAlta, error)
}
