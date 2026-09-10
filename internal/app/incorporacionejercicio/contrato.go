// Package incorporacionejercicio compone fronteras propietarias, nunca HTTP ni SQL.
// Sus dependencias nominales deben configurarse por petición/autenticación. No
// conserva órdenes entre operaciones ni sustituye restauradores históricos.
package incorporacionejercicio

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"time"
	appct "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	dom "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	personal "vec-diputacion-granada/internal/modules/personal/adapters/contrataciontemporal"
	"vec-diputacion-granada/internal/modules/personal/adapters/fuenteejercicio"
	lector "vec-diputacion-granada/internal/modules/personal/adapters/lecturaincorporacion"
	core "vec-diputacion-granada/internal/vec/domain"
)

type Configuracion struct {
	Preparador      ct.ProveedorPreparacionIncorporacionAplicacionV2
	FuentePersonal  []byte
	TernaPersonal   fuenteejercicio.TernaEsperada
	ProveedorAlta   personal.ProveedorNominalAlta
	TransaccionAlta personal.TransaccionAltaPersonal
	LectorPersonal  *lector.ConsumidorV2
	Confirmador     *appct.ServicioConfirmacionIncorporacionV2
	Reloj           ct.Reloj
}

func nulo(v any) bool {
	if v == nil {
		return true
	}
	x := reflect.ValueOf(v)
	switch x.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Func, reflect.Map, reflect.Slice, reflect.Chan:
		return x.IsNil()
	}
	return false
}
func fallo(ctx context.Context, e error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(e, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(e, context.DeadlineExceeded) {
		return context.DeadlineExceeded
	}
	if errors.Is(e, ct.ErrIntencionIncorporacionAplicacion) {
		return ct.ErrIntencionIncorporacionAplicacion
	}
	if errors.Is(e, ct.ErrDenegadaIncorporacionAplicacion) || errors.Is(e, personal.ErrDenegado) || errors.Is(e, lector.ErrDenegada) || errors.Is(e, appct.ErrConfirmacionIncorporacionDenegada) {
		return ct.ErrDenegadaIncorporacionAplicacion
	}
	if errors.Is(e, personal.ErrConflicto) || errors.Is(e, ct.ErrConflictoIncorporacionAplicacion) {
		return ct.ErrConflictoIncorporacionAplicacion
	}
	return ct.ErrComposicionIncorporacionAplicacion
}

type relojOperacion struct {
	base   ct.Reloj
	ultimo time.Time
	roto   bool
}

func (r *relojOperacion) Ahora() time.Time {
	t := r.base.Ahora()
	if r.roto || !dom.InstanteUTCCanonico(t) || (!r.ultimo.IsZero() && t.Before(r.ultimo)) {
		r.roto = true
		return time.Time{}
	}
	r.ultimo = t
	return t
}
func mismoContexto(a, b ct.ContextoAutorizacionAltaV3) bool {
	return a.Vinculo.CoincideExactamenteCon(b.Vinculo) && reflect.DeepEqual(a.Resultado, b.Resultado)
}

type proveedorAltaLigado struct {
	base personal.ProveedorNominalAlta
	p    ct.PreparacionIncorporacionAplicacionV2
}

func (p proveedorAltaLigado) ResolverAutoridad(ctx context.Context, a personal.PreparacionAlta) (personal.AutoridadAlta, error) {
	if a.Solicitud != p.p.SolicitudPersonal {
		return personal.AutoridadAlta{}, ct.ErrComposicionIncorporacionAplicacion
	}
	r, e := p.base.ResolverAutoridad(ctx, a)
	if ctx.Err() != nil {
		return personal.AutoridadAlta{}, ctx.Err()
	}
	if e != nil || r.OrganizacionRef != p.p.Preparacion.OrganizacionRef || !mismoContexto(r.Contexto, p.p.Contexto) {
		return personal.AutoridadAlta{}, ct.ErrComposicionIncorporacionAplicacion
	}
	return r, nil
}
func (p proveedorAltaLigado) AutorizarAlta(ctx context.Context, m personal.MaterialAlta) (personal.AutorizacionAlta, error) {
	return p.base.AutorizarAlta(ctx, m)
}
func clonarPreparacion(p ct.PreparacionIncorporacionAplicacionV2) (ct.PreparacionIncorporacionAplicacionV2, error) {
	p.Documentos = append([]dom.DocumentoSeguimiento(nil), p.Documentos...)
	r, e := p.Contexto.Resultado.Clonar()
	if e != nil {
		return ct.PreparacionIncorporacionAplicacionV2{}, ct.ErrComposicionIncorporacionAplicacion
	}
	p.Contexto.Resultado = r
	return p, nil
}
func validarPreparacion(p ct.PreparacionIncorporacionAplicacionV2, i ct.IntencionIncorporacionAplicacionV2, t time.Time) error {
	if _, err := core.HuellaSHA256MotivoAutorizacionV2(p.MotivoV3); err != nil {
		return ct.ErrIntencionIncorporacionAplicacion
	}
	if _, err := p.CorrelacionV3.ValorCanonico(); err != nil {
		return ct.ErrIntencionIncorporacionAplicacion
	}
	if !dom.InstanteUTCCanonico(t) {
		return ct.ErrComposicionIncorporacionAplicacion
	}
	if i.Validar() != nil || p.SolicitudPersonal.Validar() != nil || p.SolicitudPersonal.SolicitudRef != i.SolicitudPersonalRef ||
		p.SolicitudPersonal.ExpedienteRef != i.ExpedienteRef || p.VersionActualExpediente != i.VersionActualExpedienteObservada ||
		p.VersionSeguimientoEsperada > ct.MaximoEnteroSeguroOperacionAnalisis || p.MotivoClave != i.MotivoClave || !p.MotivoClave.Valida() ||
		p.Periodo.Validar() != nil || p.Contexto.ValidarPara(p.SolicitudContexto, t) != nil || len(p.Documentos) != len(i.DocumentosRefs) {
		return ct.ErrIntencionIncorporacionAplicacion
	}
	for _, r := range []string{p.Preparacion.OrganizacionRef, p.Preparacion.UnidadRef, p.Preparacion.ActorRef, p.Preparacion.CorrelacionRef} {
		if !dom.ReferenciaOpacaValida(r) {
			return ct.ErrIntencionIncorporacionAplicacion
		}
	}
	refs := map[string]bool{}
	for _, r := range i.DocumentosRefs {
		refs[r] = true
	}
	for _, d := range p.Documentos {
		if !d.TipoClave.Valida() || !refs[d.Referencia] {
			return ct.ErrIntencionIncorporacionAplicacion
		}
		delete(refs, d.Referencia)
	}
	return nil
}
func mismoOriginal(r ct.RegistroPersonalEjercicio, a personal.ReciboAlta, t time.Time) bool {
	h, e := a.Material.HuellaSHA256()
	return e == nil && r.ValidarEstructuraPara(a.Material.Preparacion.Solicitud, t) == nil && r.Resultado == a.Resultado && r.MaterialSHA256 == h &&
		r.RegistradoEn.Equal(a.RegistradoEn) && r.DecisionOriginalRef == a.DecisionOriginalRef && r.AuditoriaRef == a.AuditoriaRef && r.OutboxRef == a.OutboxRef &&
		r.EjercicioSintetico == a.EjercicioSintetico && r.FirmaOficial == a.FirmaOficial && r.EficaciaAdministrativa == a.EficaciaAdministrativa
}

// cloneRegistro no transforma el DTO en acreditación.
func cloneRegistro(r ct.RegistroPersonalEjercicio) ct.RegistroPersonalEjercicio {
	r.MaterialCanonico = bytes.Clone(r.MaterialCanonico)
	return r
}
