package incorporacionejercicio

import (
	"bytes"
	"context"
	"time"
	dom "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	personal "vec-diputacion-granada/internal/modules/personal/adapters/contrataciontemporal"
	"vec-diputacion-granada/internal/modules/personal/adapters/fuenteejercicio"
	lector "vec-diputacion-granada/internal/modules/personal/adapters/lecturaincorporacion"
)

type Servicio struct{ c Configuracion }

func Nuevo(c Configuracion) (*Servicio, error) {
	if nulo(c.Preparador) || nulo(c.ProveedorAlta) || nulo(c.TransaccionAlta) || c.LectorPersonal == nil || c.Confirmador == nil || nulo(c.Reloj) {
		return nil, ct.ErrComposicionIncorporacionAplicacion
	}
	if _, e := fuenteejercicio.NuevaFuenteEjercicio(c.FuentePersonal, c.TernaPersonal); e != nil {
		return nil, ct.ErrComposicionIncorporacionAplicacion
	}
	c.FuentePersonal = bytes.Clone(c.FuentePersonal)
	return &Servicio{c}, nil
}
func (s *Servicio) Consultar(ctx context.Context, exp string) (ct.ProyeccionIncorporacionAplicacionV2, error) {
	var cero ct.ProyeccionIncorporacionAplicacionV2
	if s == nil || ctx == nil || !dom.ReferenciaOpacaValida(exp) {
		return cero, ct.ErrIntencionIncorporacionAplicacion
	}
	if e := ctx.Err(); e != nil {
		return cero, e
	}
	reloj := &relojOperacion{base: s.c.Reloj}
	t := reloj.Ahora()
	if e := ctx.Err(); e != nil {
		return cero, e
	}
	if t.IsZero() {
		return cero, ct.ErrComposicionIncorporacionAplicacion
	}
	p, e := s.c.Preparador.Consultar(ctx, exp)
	if e != nil {
		return cero, fallo(ctx, e)
	}
	if e = ctx.Err(); e != nil {
		return cero, e
	}
	p = p.Copia()
	t = reloj.Ahora()
	if e = ctx.Err(); e != nil {
		return cero, e
	}
	if t.IsZero() || !proyeccionValida(p, exp, t) {
		return cero, ct.ErrComposicionIncorporacionAplicacion
	}
	return p, nil
}
func (s *Servicio) Confirmar(ctx context.Context, i ct.IntencionIncorporacionAplicacionV2) (ct.ReciboIncorporacionAplicacionV2, error) {
	var cero ct.ReciboIncorporacionAplicacionV2
	if s == nil || ctx == nil {
		return cero, ct.ErrComposicionIncorporacionAplicacion
	}
	if e := ctx.Err(); e != nil {
		return cero, e
	}
	if i.Validar() != nil {
		return cero, ct.ErrIntencionIncorporacionAplicacion
	}
	i = i.Copia()
	reloj := &relojOperacion{base: s.c.Reloj}
	t := reloj.Ahora()
	if e := ctx.Err(); e != nil {
		return cero, e
	}
	if t.IsZero() {
		return cero, ct.ErrComposicionIncorporacionAplicacion
	}
	p, e := s.c.Preparador.Preparar(ctx, i.Copia())
	if e != nil {
		return cero, fallo(ctx, e)
	}
	if e = ctx.Err(); e != nil {
		return cero, e
	}
	p, e = clonarPreparacion(p)
	if e != nil {
		return cero, e
	}
	t = reloj.Ahora()
	if e = ctx.Err(); e != nil {
		return cero, e
	}
	if e = validarPreparacion(p, i, t); e != nil {
		return cero, e
	}
	captura := &capturaAlta{delegado: s.c.TransaccionAlta, reloj: reloj}
	consumidor, e := personal.NuevoConsumidor(s.c.FuentePersonal, s.c.TernaPersonal, proveedorAltaLigado{s.c.ProveedorAlta, p}, captura, reloj)
	if e != nil {
		return cero, fallo(ctx, e)
	}
	reducido, e := consumidor.SolicitarAlta(ctx, p.SolicitudPersonal)
	if e != nil {
		return cero, fallo(ctx, e)
	}
	o, alta, e := captura.tomar(ctx)
	if e != nil {
		return cero, e
	}
	if reducido != alta.Recibo.Resultado || o.Material().Preparacion.Solicitud != p.SolicitudPersonal {
		return cero, ct.ErrComposicionIncorporacionAplicacion
	}
	h, e := alta.Recibo.Material.HuellaSHA256()
	if e != nil {
		return cero, ct.ErrComposicionIncorporacionAplicacion
	}
	sel := lector.Selector{OrganizacionRef: p.Preparacion.OrganizacionRef, SolicitudRef: p.SolicitudPersonal.SolicitudRef, ExpedienteRef: p.SolicitudPersonal.ExpedienteRef,
		VersionExpediente: p.SolicitudPersonal.VersionExpediente, ResultadoRef: reducido.ResultadoRef, ReciboRef: reducido.ReciboRef, RelacionRef: reducido.RelacionRef, OcupacionRef: reducido.OcupacionRef, MaterialSHA256: h}
	t = reloj.Ahora()
	if e = ctx.Err(); e != nil {
		return cero, e
	}
	if validarPreparacion(p, i, t) != nil {
		return cero, ct.ErrComposicionIncorporacionAplicacion
	}
	leido, e := s.c.LectorPersonal.Leer(ctx, sel, p.Preparacion.UnidadRef, p.Contexto)
	if e != nil {
		return cero, fallo(ctx, e)
	}
	if e = ctx.Err(); e != nil {
		return cero, e
	}
	t = reloj.Ahora()
	if e = ctx.Err(); e != nil {
		return cero, e
	}
	if validarPreparacion(p, i, t) != nil || !mismoOriginal(leido.Registro, alta.Recibo, t) {
		return cero, ct.ErrComposicionIncorporacionAplicacion
	}
	r := cloneRegistro(leido.Registro)
	m, e := ct.NuevoMaterialConfirmacionIncorporacionV2(ct.DatosMaterialConfirmacionIncorporacionV2{
		Confirmacion: ct.DatosConfirmacionIncorporacion{SolicitudPersonal: p.SolicitudPersonal, ResultadoPersonal: r.Resultado, VersionSeguimientoEsperada: p.VersionSeguimientoEsperada,
			PeriodoIncorporacion: p.Periodo, MotivoClave: p.MotivoClave, Documentos: p.Documentos},
		VersionActualExpediente: p.VersionActualExpediente, Preparacion: p.Preparacion, SolicitudContexto: p.SolicitudContexto, Contexto: p.Contexto,
		MotivoV3: p.MotivoV3, CorrelacionV3: p.CorrelacionV3, Personal: r, EjercicioSintetico: true}, t)
	if e != nil {
		return cero, ct.ErrComposicionIncorporacionAplicacion
	}
	if e = ctx.Err(); e != nil {
		return cero, e
	}
	recibo, e := s.c.Confirmador.Confirmar(ctx, m)
	if e != nil {
		return cero, fallo(ctx, e)
	}
	if e = ctx.Err(); e != nil {
		return cero, e
	}
	t = reloj.Ahora()
	if e = ctx.Err(); e != nil {
		return cero, e
	}
	if t.IsZero() || ct.ValidarMaterialRegistroIncorporacionV2(m, t) != nil || recibo.Transicion.Periodo == nil {
		return cero, ct.ErrComposicionIncorporacionAplicacion
	}
	// Servicio CT validó historia/orden original, incluso replay. No reetiquetar
	// fechas/contexto desde m actual ni fabricar un indicador Recuperado.
	out := reciboAplicacionDesdeRegistro(r, recibo)
	if !reciboVisibleValido(out, i.ExpedienteRef, t) {
		return cero, ct.ErrComposicionIncorporacionAplicacion
	}
	return out, nil
}
func reciboVisibleValido(r ct.ReciboIncorporacionAplicacionV2, exp string, t time.Time) bool {
	if r.Esquema != "vec.contratacion-temporal.incorporacion-ejercicio.recibo.v2" || r.ExpedienteRef != exp || r.Periodo.Validar() != nil ||
		!dom.InstanteUTCCanonico(r.RegistradaEn) || r.RegistradaEn.After(t) || !r.EjercicioSintetico || r.FirmaOficial || r.EficaciaAdministrativa ||
		r.VersionSolicitudPersonal == 0 || r.VersionSolicitudPersonal > ct.MaximoEnteroSeguroOperacionAnalisis || r.VersionActualExpediente == 0 || r.VersionActualExpediente > ct.MaximoEnteroSeguroOperacionAnalisis ||
		r.VersionSeguimientoAnterior >= ct.MaximoEnteroSeguroOperacionAnalisis || r.VersionSeguimientoResultante != r.VersionSeguimientoAnterior+1 {
		return false
	}
	for _, ref := range []string{r.ExpedienteRef, r.SolicitudPersonalRef, r.RelacionRef, r.ReciboRef, r.ActuacionRef, r.SeguimientoRef, r.AuditoriaRef, r.OutboxRef} {
		if !dom.ReferenciaOpacaValida(ref) {
			return false
		}
	}
	return true
}
func proyeccionValida(p ct.ProyeccionIncorporacionAplicacionV2, exp string, t time.Time) bool {
	if p.Esquema != "vec.contratacion-temporal.incorporacion-ejercicio.preparacion.v2" || p.ExpedienteRef != exp ||
		p.VersionActualExpediente == 0 || p.VersionActualExpediente > ct.MaximoEnteroSeguroOperacionAnalisis || (p.Preparacion == nil) == (p.Recibo == nil) {
		return false
	}
	if p.Recibo != nil {
		return reciboVisibleValido(*p.Recibo, exp, t)
	}
	x := p.Preparacion
	if !dom.ReferenciaOpacaValida(x.SolicitudPersonalRef) || x.VersionSolicitudPersonal == 0 || x.VersionSolicitudPersonal > ct.MaximoEnteroSeguroOperacionAnalisis ||
		x.VersionSeguimientoEsperada > ct.MaximoEnteroSeguroOperacionAnalisis || x.Periodo.Validar() != nil || len(x.Motivos) > 32 || len(x.DocumentosRefs) > 32 {
		return false
	}
	seen := map[string]bool{}
	for _, v := range x.DocumentosRefs {
		if !dom.ReferenciaOpacaValida(v) || seen[v] {
			return false
		}
		seen[v] = true
	}
	for _, v := range x.Motivos {
		if !v.Valida() {
			return false
		}
	}
	return true
}

var _ ct.ServicioIncorporacionAplicacionV2 = (*Servicio)(nil)
