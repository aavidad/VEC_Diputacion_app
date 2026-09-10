// Package incorporacionejercicio compone autoridades; no publica gobierno ni consume efectos.
package incorporacionejercicio

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"time"
	cd "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	pa "vec-diputacion-granada/internal/modules/personal/adapters/contrataciontemporal"
	pl "vec-diputacion-granada/internal/modules/personal/adapters/lecturaincorporacion"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

var ErrAutoridadAplicacion = errors.New("incorporacion: autoridad nominal no disponible")

// PeticionAutoridad NO es DTO de canal. Solo una captura verificada propietaria
// puede entregar estos datos, incluido el mapeo explícito de referencias CT.
type PeticionAutoridad struct {
	Autenticacion             core.SolicitudRevalidacionAutenticacionActorV1
	Contexto                  core.SolicitudContextoActor
	PreparacionCT             ct.PreparacionSeguimientoConfirmacionIncorporacion
	MotivoAlta, MotivoLectura core.ReferenciaEntradaCatalogo
}
type FuentePeticionAutoridad interface {
	PeticionVerificada(context.Context) (PeticionAutoridad, error)
}
type GeneradorCorrelacionAutoridad interface {
	NuevaReferenciaCorrelacionAutorizacionV2(context.Context) (string, error)
}

// AutoridadAplicacion se construye por petición. No almacena decisiones ni
// capacidades. Sus dependencias deben tolerar concurrencia; sus datos son privados.
type AutoridadAplicacion struct {
	peticion    PeticionAutoridad
	contexto    ct.ContextoAutorizacionAltaV3
	revalidador vp.RevalidadorAutenticacionActorV1
	cadena      *CadenaAutorizacionAplicacion
	correlador  GeneradorCorrelacionAutoridad
	reloj       ct.Reloj
	creadaEn    time.Time
}

func NuevaAutoridadAplicacion(ctx context.Context, fuente FuentePeticionAutoridad,
	revalidador vp.RevalidadorAutenticacionActorV1, resolutor core.ResolutorContextoActorRegistradoV2,
	cadena *CadenaAutorizacionAplicacion, correlador GeneradorCorrelacionAutoridad, reloj ct.Reloj) (*AutoridadAplicacion, error) {
	if err := autoridadContextoError(ctx); err != nil {
		return nil, err
	}
	for _, d := range []any{fuente, revalidador, resolutor, cadena, correlador, reloj} {
		if autoridadNula(d) {
			return nil, ErrAutoridadAplicacion
		}
	}
	if !cadena.valida() {
		return nil, ErrAutoridadAplicacion
	}
	inicio := reloj.Ahora()
	if err := autoridadInstante(ctx, inicio, time.Time{}); err != nil {
		return nil, err
	}
	p, e := fuente.PeticionVerificada(ctx)
	if e != nil {
		return nil, autoridadFallo(ctx, e)
	}
	if err := autoridadContextoError(ctx); err != nil {
		return nil, err
	}
	if p.Autenticacion.Validar() != nil || p.Contexto.Validar() != nil {
		return nil, ErrAutoridadAplicacion
	}
	for _, r := range []string{p.PreparacionCT.OrganizacionRef, p.PreparacionCT.UnidadRef, p.PreparacionCT.ActorRef, p.PreparacionCT.CorrelacionRef} {
		if !cd.ReferenciaOpacaValida(r) {
			return nil, ErrAutoridadAplicacion
		}
	}
	for _, m := range []core.ReferenciaEntradaCatalogo{p.MotivoAlta, p.MotivoLectura} {
		if _, e := core.HuellaSHA256MotivoAutorizacionV2(m); e != nil {
			return nil, ErrAutoridadAplicacion
		}
	}
	v, r, e := core.CrearVinculoAutenticacionActorV2ConResultado(ctx, revalidador, p.Autenticacion, resolutor, p.Contexto, reloj)
	if e != nil {
		return nil, autoridadFallo(ctx, e)
	}
	a := &AutoridadAplicacion{peticion: p, contexto: ct.ContextoAutorizacionAltaV3{Vinculo: v, Resultado: r}, revalidador: revalidador, cadena: cadena, correlador: correlador, reloj: reloj, creadaEn: inicio}
	if _, e = a.comprobar(ctx, inicio); e != nil {
		return nil, e
	}
	a.contexto.Resultado, e = r.Clonar()
	if e != nil {
		return nil, ErrAutoridadAplicacion
	}
	if e = autoridadContextoError(ctx); e != nil {
		return nil, e
	}
	return a, nil
}
func (a *AutoridadAplicacion) ContextoAutoridad() (ct.ContextoAutorizacionAltaV3, error) {
	if a == nil {
		return ct.ContextoAutorizacionAltaV3{}, ErrAutoridadAplicacion
	}
	r, e := a.contexto.Resultado.Clonar()
	if e != nil {
		return ct.ContextoAutorizacionAltaV3{}, ErrAutoridadAplicacion
	}
	return ct.ContextoAutorizacionAltaV3{Vinculo: a.contexto.Vinculo, Resultado: r}, nil
}
func (a *AutoridadAplicacion) PreparacionAutoridadCT() ct.PreparacionSeguimientoConfirmacionIncorporacion {
	if a == nil {
		return ct.PreparacionSeguimientoConfirmacionIncorporacion{}
	}
	return a.peticion.PreparacionCT
}
func (a *AutoridadAplicacion) solicitudContexto() ct.SolicitudResolverContextoAutorizacionAltaV3 {
	return ct.SolicitudResolverContextoAutorizacionAltaV3{AutenticacionRef: a.peticion.Autenticacion.AutenticacionRef, SesionRef: a.peticion.Autenticacion.SesionRef, PerfilRef: a.peticion.Contexto.PerfilActivoRef}
}
func (a *AutoridadAplicacion) comprobar(ctx context.Context, previo time.Time) (time.Time, error) {
	if e := autoridadContextoError(ctx); e != nil {
		return time.Time{}, e
	}
	if a == nil || autoridadNula(a.reloj) {
		return time.Time{}, ErrAutoridadAplicacion
	}
	ahora := a.reloj.Ahora()
	if e := autoridadInstante(ctx, ahora, previo); e != nil {
		return time.Time{}, e
	}
	if ahora.Before(a.creadaEn) || a.contexto.ValidarPara(a.solicitudContexto(), ahora) != nil {
		return time.Time{}, ErrAutoridadAplicacion
	}
	v, e := a.contexto.Vinculo.Datos()
	if e != nil || v.GarantiaObservada != core.AuthAssuranceHigh || (v.Superficie != core.SuperficieAutenticacionInternaCorporativaV1 && v.Superficie != core.SuperficieAutenticacionAdministracionPrivilegiadaV1) {
		return time.Time{}, ErrAutoridadAplicacion
	}
	return ahora, autoridadContextoError(ctx)
}

// Revalida contra Identidad; no sustituye el vínculo original ni crea Contexto.
func (a *AutoridadAplicacion) revalidar(ctx context.Context, previo time.Time) (time.Time, error) {
	ahora, e := a.comprobar(ctx, previo)
	if e != nil {
		return time.Time{}, e
	}
	r, e := a.revalidador.RevalidarAutenticacionActorV1(ctx, a.peticion.Autenticacion)
	if e != nil {
		return time.Time{}, autoridadFallo(ctx, e)
	}
	fin, e := a.comprobar(ctx, ahora)
	if e != nil {
		return time.Time{}, e
	}
	v, _ := a.contexto.Vinculo.Datos()
	if !autoridadAutenticacionExacta(r, v, fin) {
		return time.Time{}, ErrAutoridadAplicacion
	}
	return fin, autoridadContextoError(ctx)
}
func autoridadAutenticacionExacta(r core.AutenticacionRevalidadaV1, v core.DatosVinculoAutenticacionActorV2, ahora time.Time) bool {
	if r.Validar() != nil || r.SesionRevalidadaEn.Before(v.SesionRevalidadaEn) || r.SesionRevalidadaEn.After(ahora) || !ahora.Before(r.SesionValidaHasta) {
		return false
	}
	esperado := core.AutenticacionRevalidadaV1{AutenticacionRef: v.AutenticacionRef, AutenticacionHuellaSHA256: v.AutenticacionHuellaSHA256, AsercionRef: v.AsercionRef, SesionRef: v.SesionRef, ControlSesionRef: v.ControlSesionRef, ControlSesionRevision: v.ControlSesionRevision, ControlSesionHuellaSHA256: v.ControlSesionHuellaSHA256, CuentaRef: v.CuentaRef, CuentaOrdinariaRef: v.CuentaOrdinariaRef, CuentaPrivilegiada: v.CuentaPrivilegiada, Superficie: v.Superficie, MetodoObservado: v.MetodoObservado, GarantiaObservada: v.GarantiaObservada, PoliticaGarantiaRef: v.PoliticaGarantiaRef, PoliticaGarantiaHuellaSHA256: v.PoliticaGarantiaHuellaSHA256, AutenticacionVerificadaEn: v.AutenticacionVerificadaEn, SesionEmitidaEn: v.SesionEmitidaEn, SesionValidaHasta: v.SesionValidaHasta, SesionRevalidadaEn: r.SesionRevalidadaEn}
	return r == esperado
}
func (a *AutoridadAplicacion) ResolverAutoridad(ctx context.Context, p pa.PreparacionAlta) (pa.AutoridadAlta, error) {
	if _, e := a.revalidar(ctx, time.Time{}); e != nil {
		return pa.AutoridadAlta{}, e
	}
	v, _ := a.contexto.Vinculo.Datos()
	m := pa.MaterialAlta{Preparacion: p, OrganizacionRef: a.peticion.PreparacionCT.OrganizacionRef, ActorRef: v.PrincipalID, PerfilRef: v.PerfilActivoRef}
	if m.Validar() != nil {
		return pa.AutoridadAlta{}, ErrAutoridadAplicacion
	}
	c, e := a.ContextoAutoridad()
	if e != nil {
		return pa.AutoridadAlta{}, e
	}
	if e = autoridadContextoError(ctx); e != nil {
		return pa.AutoridadAlta{}, e
	}
	return pa.AutoridadAlta{OrganizacionRef: m.OrganizacionRef, Contexto: c}, nil
}
func (a *AutoridadAplicacion) AutorizarAlta(ctx context.Context, m pa.MaterialAlta) (pa.AutorizacionAlta, error) {
	var cero pa.AutorizacionAlta
	inicio, errInicio := a.comprobar(ctx, time.Time{})
	if errInicio != nil {
		return cero, errInicio
	}
	v, _ := a.contexto.Vinculo.Datos()
	if m.Validar() != nil || m.OrganizacionRef != a.peticion.PreparacionCT.OrganizacionRef || m.ActorRef != v.PrincipalID || m.PerfilRef != v.PerfilActivoRef {
		return cero, ErrAutoridadAplicacion
	}
	r, e := pa.RecursoAltaEjercicio(m)
	if e != nil {
		return cero, ErrAutoridadAplicacion
	}
	cor, e := core.GenerarReferenciaCorrelacionAutorizacionV2(ctx, a.correlador)
	if e != nil {
		return cero, autoridadFallo(ctx, e)
	}
	c, _, e := a.conceder(ctx, inicio, autoridadAlta, r, a.peticion.MotivoAlta, cor)
	if e != nil {
		return cero, e
	}
	return pa.AutorizacionAlta{Solicitud: c.Solicitud, Decision: c.Decision, Confirmacion: c.Confirmacion, Exportacion: c.Exportacion}, nil
}
func (a *AutoridadAplicacion) AutorizarLecturaIncorporacionV2(ctx context.Context, m pl.MaterialV2) (pl.AutorizacionV2, error) {
	var cero pl.AutorizacionV2
	inicio, errInicio := a.comprobar(ctx, m.PreparadoEn())
	if errInicio != nil {
		return cero, errInicio
	}
	c, e := m.Contexto()
	if e != nil || !a.contextoExacto(c) || m.Selector().OrganizacionRef != a.peticion.PreparacionCT.OrganizacionRef || m.UnidadRef() != a.peticion.PreparacionCT.UnidadRef {
		return cero, ErrAutoridadAplicacion
	}
	r, e := m.Recurso()
	if e != nil {
		return cero, ErrAutoridadAplicacion
	}
	cor, e := core.GenerarReferenciaCorrelacionAutorizacionV2(ctx, a.correlador)
	if e != nil {
		return cero, autoridadFallo(ctx, e)
	}
	x, _, e := a.conceder(ctx, inicio, autoridadLectura, r, a.peticion.MotivoLectura, cor)
	if e != nil {
		return cero, e
	}
	return pl.AutorizacionV2{Solicitud: x.Solicitud, Decision: x.Decision, Confirmacion: x.Confirmacion, Exportacion: x.Exportacion}, nil
}
func (a *AutoridadAplicacion) AutorizarConfirmacionIncorporacion(ctx context.Context, m ct.MaterialConfirmacionIncorporacionV2) (ct.AutorizacionConfirmacionIncorporacionV2, error) {
	var cero ct.AutorizacionConfirmacionIncorporacionV2
	inicio, errInicio := a.comprobar(ctx, time.Time{})
	if errInicio != nil {
		return cero, errInicio
	}
	d, e := m.Datos()
	if e != nil || d.Preparacion != a.peticion.PreparacionCT || d.SolicitudContexto != a.solicitudContexto() || !a.contextoExacto(d.Contexto) {
		return cero, ErrAutoridadAplicacion
	}
	r, e := ct.RecursoConfirmacionIncorporacionV2(m)
	if e != nil {
		return cero, ErrAutoridadAplicacion
	}
	c, fin, e := a.conceder(ctx, inicio, autoridadCT, r, d.MotivoV3, d.CorrelacionV3)
	if e != nil {
		return cero, e
	}
	if c.ValidarPara(m, fin) != nil {
		return cero, ErrAutoridadAplicacion
	}
	if e = autoridadContextoError(ctx); e != nil {
		return cero, e
	}
	return c, nil
}
func (a *AutoridadAplicacion) contextoExacto(c ct.ContextoAutorizacionAltaV3) bool {
	return c.Resultado.Validar() == nil && c.Vinculo.CoincideExactamenteCon(a.contexto.Vinculo) && reflect.DeepEqual(c.Resultado, a.contexto.Resultado)
}
func autoridadContextoError(ctx context.Context) error {
	if ctx == nil {
		return ErrAutoridadAplicacion
	}
	return ctx.Err()
}
func autoridadInstante(ctx context.Context, t, previo time.Time) error {
	if e := autoridadContextoError(ctx); e != nil {
		return e
	}
	if !cd.InstanteUTCCanonico(t) || t.Before(previo) {
		return ErrAutoridadAplicacion
	}
	return nil
}
func autoridadNula(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Func, reflect.Map, reflect.Slice, reflect.Chan:
		return r.IsNil()
	}
	return false
}

type autoridadError struct{ causa error }

func (autoridadError) Error() string                { return ErrAutoridadAplicacion.Error() }
func (e autoridadError) Unwrap() error              { return e.causa }
func (e autoridadError) Format(s fmt.State, _ rune) { _, _ = io.WriteString(s, e.Error()) }
func (e autoridadError) LogValue() slog.Value       { return slog.StringValue(e.Error()) }
func autoridadFallo(ctx context.Context, e error) error {
	if c := autoridadContextoError(ctx); c != nil {
		return c
	}
	return autoridadError{e}
}

var _ pa.ProveedorNominalAlta = (*AutoridadAplicacion)(nil)
var _ pl.ProveedorV2 = (*AutoridadAplicacion)(nil)
var _ ct.ProveedorAutorizacionConfirmacionIncorporacionV2 = (*AutoridadAplicacion)(nil)
