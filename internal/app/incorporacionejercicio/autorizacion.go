package incorporacionejercicio

import (
	"bytes"
	"context"
	"time"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	pa "vec-diputacion-granada/internal/modules/personal/adapters/contrataciontemporal"
	pl "vec-diputacion-granada/internal/modules/personal/adapters/lecturaincorporacion"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	app "vec-diputacion-granada/internal/vec/application"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

// EmisionAutoridad usa exclusivamente emisores ya gobernados por composición.
// No acepta secretos, firmantes HTTP, claims ni una bandera de autorización.
type EmisionAutoridad struct {
	Emisor *confianza.EmisorCapacidadesAtestacionAutorizacionV3
	Raiz   confianza.RaizPublicaAtestacionAutorizacionV3
}
type EmisionesAutoridad struct{ Alta, Lectura, CT EmisionAutoridad }
type CadenaAutorizacionAplicacion struct {
	servicio  *app.ServicioAutorizacionSolicitudLigadaV3
	atestador *app.ServicioAtestacionesAutorizacionV3
	confianza *confianza.ServicioConfianzaAtestacionAutorizacionV3
	emisiones EmisionesAutoridad
}

func NuevaCadenaAutorizacionAplicacion(servicio *app.ServicioAutorizacionSolicitudLigadaV3, atestador *app.ServicioAtestacionesAutorizacionV3, verificador *confianza.ServicioConfianzaAtestacionAutorizacionV3, emisiones EmisionesAutoridad) (*CadenaAutorizacionAplicacion, error) {
	c := &CadenaAutorizacionAplicacion{servicio, atestador, verificador, emisiones}
	if !c.valida() {
		return nil, ErrAutoridadAplicacion
	}
	return c, nil
}
func (c *CadenaAutorizacionAplicacion) valida() bool {
	return c != nil && c.servicio != nil && c.atestador != nil && c.confianza != nil && c.emisiones.Alta.Emisor != nil && c.emisiones.Lectura.Emisor != nil && c.emisiones.CT.Emisor != nil
}

type autoridadOperacion uint8

const (
	autoridadAlta autoridadOperacion = iota + 1
	autoridadLectura
	autoridadCT
)

type autoridadContrato struct{ accion, finalidad, modulo, tipo, audiencia string }

func autoridadContratoPara(o autoridadOperacion) (autoridadContrato, error) {
	switch o {
	case autoridadAlta:
		return autoridadContrato{pa.AccionAltaEjercicio, pa.FinalidadAltaEjercicio, "personal", pa.TipoRecursoAltaEjercicio, pa.AudienciaAltaEjercicio}, nil
	case autoridadLectura:
		return autoridadContrato{pl.Accion, pl.Finalidad, "personal", pl.TipoRecursoV2, pl.AudienciaV2}, nil
	case autoridadCT:
		return autoridadContrato{ct.AccionConfirmarIncorporacion, ct.FinalidadConfirmarIncorporacion, "contratacion_temporal", ct.TipoRecursoConfirmacionIncorporacionV2, ct.AudienciaConfirmacionIncorporacionV2}, nil
	}
	return autoridadContrato{}, ErrAutoridadAplicacion
}
func (a *AutoridadAplicacion) conceder(ctx context.Context, inicio time.Time, o autoridadOperacion, r core.RecursoAutorizable, m core.ReferenciaEntradaCatalogo, cor core.ReferenciaCorrelacionAutorizacionV2) (ct.AutorizacionConfirmacionIncorporacionV2, time.Time, error) {
	var cero ct.AutorizacionConfirmacionIncorporacionV2
	t, e := a.revalidar(ctx, inicio)
	if e != nil {
		return cero, time.Time{}, e
	}
	contrato, e := autoridadContratoPara(o)
	if e != nil || !a.cadena.valida() || r.ModuloID != contrato.modulo || r.Tipo != contrato.tipo {
		return cero, time.Time{}, ErrAutoridadAplicacion
	}
	contexto, e := a.ContextoAutoridad()
	if e != nil {
		return cero, time.Time{}, e
	}
	s, e := core.NuevaSolicitudAutorizacionLigadaV3(core.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: contexto.Vinculo, Accion: contrato.accion, Finalidad: contrato.finalidad, Recurso: r, ReferenciaMotivo: m, Correlacion: cor})
	if e != nil {
		return cero, time.Time{}, autoridadFallo(ctx, e)
	}
	t, e = a.comprobar(ctx, t)
	if e != nil {
		return cero, time.Time{}, e
	}
	d, confirmacion, e := a.cadena.servicio.ExigirSolicitudLigadaV3(ctx, s, contexto.Resultado)
	if e != nil {
		return cero, time.Time{}, autoridadFallo(ctx, e)
	}
	t, e = a.comprobar(ctx, t)
	if e != nil {
		return cero, time.Time{}, e
	}
	orden, e := vp.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(s, d, m, contexto.Resultado)
	if e != nil || confirmacion.ValidarPara(orden) != nil || !confirmacion.DentroDeVentanaEn(t) {
		return cero, time.Time{}, ErrAutoridadAplicacion
	}
	at, e := a.cadena.atestador.Atestar(ctx, d, m, contexto.Resultado)
	if e != nil {
		return cero, time.Time{}, autoridadFallo(ctx, e)
	}
	t, e = a.comprobar(ctx, t)
	if e != nil {
		return cero, time.Time{}, e
	}
	prueba, e := a.cadena.confianza.Verificar(ctx, s, d, m, contexto.Resultado, at)
	if e != nil {
		return cero, time.Time{}, autoridadFallo(ctx, e)
	}
	t, e = a.comprobar(ctx, t)
	if e != nil {
		return cero, time.Time{}, e
	}
	emision := a.cadena.emisiones.Alta
	if o == autoridadLectura {
		emision = a.cadena.emisiones.Lectura
	}
	if o == autoridadCT {
		emision = a.cadena.emisiones.CT
	}
	cap, e := emision.Emisor.Emitir(ctx, s, d, m, contexto.Resultado, at, prueba)
	if e != nil {
		return cero, time.Time{}, autoridadFallo(ctx, e)
	}
	t, e = a.comprobar(ctx, t)
	if e != nil {
		return cero, time.Time{}, e
	}
	material, e := confianza.NuevoMaterialConsumoAutorizacionAtestadaV3(s, d, m, contexto.Resultado, at, prueba, cap, emision.Raiz)
	if e != nil {
		return cero, time.Time{}, autoridadFallo(ctx, e)
	}
	x, e := material.ExportarMaterialParaConsumidor()
	if e != nil {
		return cero, time.Time{}, autoridadFallo(ctx, e)
	}
	t, e = a.revalidar(ctx, t)
	if e != nil {
		return cero, time.Time{}, e
	}
	c := ct.AutorizacionConfirmacionIncorporacionV2{Solicitud: s, Decision: d, Confirmacion: confirmacion, Exportacion: x}
	if e = autoridadCotejarExportacion(c, contexto, r, contrato, t); e != nil {
		return cero, time.Time{}, e
	}
	if e = autoridadContextoError(ctx); e != nil {
		return cero, time.Time{}, e
	}
	return c, t, nil
}

// Cotejo común completo también para los consumidores cuyos validadores son
// privados. La exportación conserva bytes propios mediante sus getters opacos.
func autoridadCotejarExportacion(a ct.AutorizacionConfirmacionIncorporacionV2, c ct.ContextoAutorizacionAltaV3, r core.RecursoAutorizable, k autoridadContrato, t time.Time) error {
	if a.Exportacion.ValidarEstructura() != nil || a.Decision.ValidarPara(a.Solicitud) != nil || !a.Confirmacion.DentroDeVentanaEn(t) {
		return ErrAutoridadAplicacion
	}
	s, e := a.Solicitud.Datos()
	if e != nil || !s.VinculoAutenticacionActor.CoincideExactamenteCon(c.Vinculo) || s.Accion != k.accion || s.Finalidad != k.finalidad {
		return ErrAutoridadAplicacion
	}
	rh, e := r.HuellaContextoAutorizacionSHA256()
	if e != nil {
		return ErrAutoridadAplicacion
	}
	sh, e := s.Recurso.HuellaContextoAutorizacionSHA256()
	if e != nil || rh != sh || r.Referencia != s.Recurso.Referencia || r.ModuloID != s.Recurso.ModuloID || r.Tipo != s.Recurso.Tipo {
		return ErrAutoridadAplicacion
	}
	dh, e := core.HuellaSHA256DecisionAutorizacionV3(a.Decision)
	if e != nil {
		return ErrAutoridadAplicacion
	}
	mh, e := core.HuellaSHA256MotivoAutorizacionV2(s.ReferenciaMotivo)
	if e != nil {
		return ErrAutoridadAplicacion
	}
	desde, hasta, e := a.Decision.VentanaValidez()
	if e != nil {
		return ErrAutoridadAplicacion
	}
	ok, _, e := a.Decision.Resultado()
	if e != nil || !ok {
		return ErrAutoridadAplicacion
	}
	d, e := a.Confirmacion.Datos()
	if e != nil {
		return ErrAutoridadAplicacion
	}
	x := a.Exportacion.ResumenCapacidad()
	if d.DecisionHuellaSHA256 != dh || !d.EmitidaEn.Equal(desde) || !d.ValidaHasta.Equal(hasta) || x.DecisionRef() != d.DecisionRef || x.DecisionHuellaSHA256() != dh || x.MotivoHuellaSHA256() != mh || x.ContextoRef() != c.Resultado.RegistroContextoRef || x.ContextoHuellaSHA256() != c.Resultado.HuellaSHA256 || x.Operacion() != k.accion || x.EfectoRef() != r.Referencia || x.EfectoHuellaSHA256() != rh || x.AudienciaConsumo() != k.audiencia || t.Before(x.EmitidaEn()) || !t.Before(x.ExpiraEn()) || x.EmitidaEn().Before(desde) || x.ExpiraEn().After(hasta) {
		return ErrAutoridadAplicacion
	}
	dc, e := core.RepresentacionCanonicaDecisionAutorizacionV3(a.Decision)
	if e != nil {
		return ErrAutoridadAplicacion
	}
	mc, e := core.RepresentacionCanonicaMotivoAutorizacionV2(s.ReferenciaMotivo)
	if e != nil {
		return ErrAutoridadAplicacion
	}
	if !bytes.Equal(dc, a.Exportacion.DecisionCanonica()) || !bytes.Equal(mc, a.Exportacion.MotivoCanonico()) || !bytes.Equal(c.Resultado.RepresentacionCanonica, a.Exportacion.ContextoActorCanonico()) || a.Exportacion.PersonaVersion() != c.Resultado.Contexto.Instantanea.PersonaVersion || a.Exportacion.PerfilVersion() != c.Resultado.Contexto.Instantanea.PerfilVersion {
		return ErrAutoridadAplicacion
	}
	return nil
}
