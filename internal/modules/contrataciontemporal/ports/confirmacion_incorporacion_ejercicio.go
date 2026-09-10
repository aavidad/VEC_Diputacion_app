package ports

import (
	"bytes"
	"context"
	"maps"
	"time"

	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// AutorizacionConfirmacionIncorporacionV2 exige exportación AD3 completa.
// Reutiliza la acción CT, NUNCA el permiso de alta Personal. La nueva audiencia
// y tipo de recurso necesitan gobierno/consumidor SQL futuros; no concede roles.
type AutorizacionConfirmacionIncorporacionV2 struct {
	Solicitud    core.SolicitudAutorizacionLigadaV3
	Decision     core.DecisionAutorizacionLigadaV3
	Confirmacion vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3
	Exportacion  vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

// ProveedorAutorizacionConfirmacionIncorporacionV2 es una dependencia confiable
// de composición, no una estructura del canal. Emite por la cadena nominal de
// decisión, registro, firma/verificación y exportación comunes de VEC.
type ProveedorAutorizacionConfirmacionIncorporacionV2 interface {
	AutorizarConfirmacionIncorporacion(context.Context, MaterialConfirmacionIncorporacionV2) (AutorizacionConfirmacionIncorporacionV2, error)
}

func (a AutorizacionConfirmacionIncorporacionV2) ValidarPara(m MaterialConfirmacionIncorporacionV2, ahora time.Time) error {
	if m.validarEn(ahora) != nil || a.Exportacion.ValidarEstructura() != nil || a.Decision.ValidarPara(a.Solicitud) != nil {
		return ErrContextoConfirmacionIncorporacionInvalido
	}
	d, _ := m.Datos()
	s, es := a.Solicitud.Datos()
	c, ec := a.Confirmacion.Datos()
	concedida, _, ed := a.Decision.Resultado()
	cor, ecor := s.Correlacion.ValorCanonico()
	corEsperada, _ := d.CorrelacionV3.ValorCanonico()
	if es != nil || ec != nil || ed != nil || ecor != nil || !concedida ||
		!s.VinculoAutenticacionActor.CoincideExactamenteCon(d.Contexto.Vinculo) ||
		s.Accion != AccionConfirmarIncorporacion || s.Finalidad != FinalidadConfirmarIncorporacion ||
		s.ReferenciaMotivo != d.MotivoV3 || cor != corEsperada || !a.Confirmacion.DentroDeVentanaEn(ahora) {
		return ErrContextoConfirmacionIncorporacionInvalido
	}
	r, er := RecursoConfirmacionIncorporacionV2(m)
	h, eh := r.HuellaContextoAutorizacionSHA256()
	hs, ehs := s.Recurso.HuellaContextoAutorizacionSHA256()
	dh, edh := core.HuellaSHA256DecisionAutorizacionV3(a.Decision)
	mh, emh := core.HuellaSHA256MotivoAutorizacionV2(s.ReferenciaMotivo)
	desde, hasta, ev := a.Decision.VentanaValidez()
	x := a.Exportacion.ResumenCapacidad()
	if er != nil || eh != nil || ehs != nil || h != hs || edh != nil || emh != nil || ev != nil ||
		s.Recurso.Referencia != r.Referencia || s.Recurso.ModuloID != r.ModuloID || s.Recurso.Tipo != r.Tipo ||
		!maps.Equal(s.Recurso.Ambitos, r.Ambitos) || !maps.Equal(s.Recurso.Atributos, r.Atributos) ||
		c.DecisionHuellaSHA256 != dh || !c.EmitidaEn.Equal(desde) || !c.ValidaHasta.Equal(hasta) ||
		x.DecisionRef() != c.DecisionRef || x.DecisionHuellaSHA256() != dh || x.MotivoHuellaSHA256() != mh ||
		x.ContextoRef() != d.Contexto.Resultado.RegistroContextoRef || x.ContextoHuellaSHA256() != d.Contexto.Resultado.HuellaSHA256 ||
		x.Operacion() != AccionConfirmarIncorporacion || x.EfectoRef() != r.Referencia || x.EfectoHuellaSHA256() != h ||
		x.AudienciaConsumo() != AudienciaConfirmacionIncorporacionV2 || ahora.Before(x.EmitidaEn()) || !ahora.Before(x.ExpiraEn()) ||
		x.EmitidaEn().Before(desde) || x.ExpiraEn().After(hasta) {
		return ErrContextoConfirmacionIncorporacionInvalido
	}
	dc, edc := core.RepresentacionCanonicaDecisionAutorizacionV3(a.Decision)
	mc, emc := core.RepresentacionCanonicaMotivoAutorizacionV2(s.ReferenciaMotivo)
	if edc != nil || emc != nil || !bytes.Equal(dc, a.Exportacion.DecisionCanonica()) ||
		!bytes.Equal(mc, a.Exportacion.MotivoCanonico()) || !bytes.Equal(d.Contexto.Resultado.RepresentacionCanonica, a.Exportacion.ContextoActorCanonico()) ||
		a.Exportacion.PersonaVersion() != d.Contexto.Resultado.Contexto.Instantanea.PersonaVersion ||
		a.Exportacion.PerfilVersion() != d.Contexto.Resultado.Contexto.Instantanea.PerfilVersion {
		return ErrContextoConfirmacionIncorporacionInvalido
	}
	return nil
}

// OrdenConfirmacionIncorporacionV2 es una intención nominal, NO prueba de commit
// Personal ni recibo CT. No se convierte a OrdenConfirmarIncorporacion v1:
// hacerlo perdería el compromiso del material completo. La API histórica sigue
// intacta para su historia y recuperación, no es un fallback de esta versión.
type OrdenConfirmacionIncorporacionV2 struct {
	material     MaterialConfirmacionIncorporacionV2
	autorizacion AutorizacionConfirmacionIncorporacionV2
	evaluadaEn   time.Time
}

func NuevaOrdenConfirmacionIncorporacionV2(m MaterialConfirmacionIncorporacionV2, a AutorizacionConfirmacionIncorporacionV2, ahora time.Time) (OrdenConfirmacionIncorporacionV2, error) {
	if a.ValidarPara(m, ahora) != nil {
		return OrdenConfirmacionIncorporacionV2{}, ErrOrdenConfirmacionIncorporacionInvalida
	}
	return OrdenConfirmacionIncorporacionV2{material: m, autorizacion: a, evaluadaEn: ahora}, nil
}

func (o OrdenConfirmacionIncorporacionV2) ValidarEn(ahora time.Time) error {
	if ahora.Before(o.evaluadaEn) || o.autorizacion.ValidarPara(o.material, ahora) != nil {
		return ErrOrdenConfirmacionIncorporacionInvalida
	}
	return nil
}

func (o OrdenConfirmacionIncorporacionV2) Material() MaterialConfirmacionIncorporacionV2 {
	return o.material
}
func (o OrdenConfirmacionIncorporacionV2) Exportacion() vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	return o.autorizacion.Exportacion
}

// La futura transacción única debe revalidar esta orden con reloj propio,
// verificar/consumir AD3 fresco, cotejar origen mediante API propietaria Personal
// (no SELECT cruzado), CAS de expediente/seguimiento, historial, recibo, audit y
// outbox atómicos. Una AcreditacionPersonalIncorporacion en memoria NO reemplaza
// esa comprobación transaccional. Replay conserva la orden/recibo originales y
// valida permiso fresco por separado; no refecha ni reemite Personal. Este
// contrato no implementa transacción, definición temporal ni avance legal.
