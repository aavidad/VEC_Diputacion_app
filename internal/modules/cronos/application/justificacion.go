package application

import (
	"context"
	"errors"
	"time"
	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type ServicioJustificacion struct {
	fuente     ports.FuenteJustificacion
	documentos ports.DocumentosJustificacion
	repo       ports.RepositorioJustificacion
	reloj      ports.Reloj
}

func NuevoServicioJustificacion(f ports.FuenteJustificacion, d ports.DocumentosJustificacion, r ports.RepositorioJustificacion, reloj ports.Reloj) (*ServicioJustificacion, error) {
	for _, v := range []any{f, d, r, reloj} {
		if dependenciaRemotaNula(v) {
			return nil, ports.ErrJustificacionNoDisponible
		}
	}
	return &ServicioJustificacion{f, d, r, reloj}, nil
}
func (s *ServicioJustificacion) preparar(ctx context.Context, o ports.OrdenJustificacion, ref string) (vecdomain.ContextoActor, ports.PreparacionJustificacion, error) {
	if s == nil || ctx == nil || ctx.Err() != nil {
		return vecdomain.ContextoActor{}, ports.PreparacionJustificacion{}, ports.ErrJustificacionNoDisponible
	}
	for _, v := range []any{s.fuente, s.documentos, s.repo, s.reloj, o.ProveedorMaterial()} {
		if dependenciaRemotaNula(v) {
			return vecdomain.ContextoActor{}, ports.PreparacionJustificacion{}, ports.ErrJustificacionNoDisponible
		}
	}
	a, e := o.ContextoActor()
	if e != nil {
		return a, ports.PreparacionJustificacion{}, ports.ErrJustificacionNoDisponible
	}
	if _, _, e = empleadoVigente(a, s.reloj); e != nil {
		return a, ports.PreparacionJustificacion{}, ports.ErrJustificacionNoDisponible
	}
	if !domain.SolicitudPermisoRefValida(ref) {
		return a, ports.PreparacionJustificacion{}, domain.ErrJustificacionInvalida
	}
	p, e := s.fuente.PrepararJustificacion(ctx, o, ref)
	if e != nil {
		return a, ports.PreparacionJustificacion{}, e
	}
	if p.Solicitud.SolicitudRef != ref || p.Solicitud.Validar(p.Politica) != nil || (p.Actual != nil && p.Actual.Validar(p.Solicitud, p.Politica) != nil) {
		return a, ports.PreparacionJustificacion{}, ports.ErrJustificacionNoDisponible
	}
	return a, p, nil
}
func materialJustificacion(a vecdomain.ContextoActor, p ports.PreparacionJustificacion, v domain.VinculoJustificacion, key, accion string, version int64) domain.MaterialJustificacion {
	return domain.MaterialJustificacion{ActorRef: a.PersonaRef, PerfilRef: a.PerfilActivoRef, ClaveOperacion: key, Accion: accion, SolicitudVersion: p.Solicitud.Version, VersionEsperada: version, PoliticaRef: p.Politica.Referencia, PoliticaVersion: p.Politica.Version, PoliticaSHA256: p.Politica.SHA256, Vinculo: v}
}
func reciboJustificacionCoherente(r ports.ReciboJustificacion, m domain.MaterialJustificacion, p ports.PreparacionJustificacion) bool {
	h, e := m.Huella()
	if e != nil || r.HuellaMaterial != h || r.ReciboRef == "" || r.FechaUTC.IsZero() || r.FechaUTC.Location() != time.UTC || r.FechaUTC.Nanosecond()%1000 != 0 || r.Justificacion.Validar(p.Solicitud, p.Politica) != nil || r.Justificacion.Vinculo != m.Vinculo || r.Justificacion.Version != m.VersionEsperada+1 {
		return false
	}
	if m.Accion == domain.AccionAnexarJustificacion {
		return r.Justificacion.Estado == domain.JustificacionPendiente && r.Justificacion.MotivoRef == ""
	}
	return r.Justificacion.Estado == m.Decision && r.Justificacion.MotivoRef == m.MotivoRef
}
func (s *ServicioJustificacion) Anexar(ctx context.Context, o ports.OrdenJustificacion, in ports.PeticionAnexoJustificacion) (ports.ResultadoAnexoJustificacion, error) {
	a, p, e := s.preparar(ctx, o, in.SolicitudRef)
	if e != nil {
		return ports.ResultadoAnexoJustificacion{}, e
	}
	v := domain.VinculoJustificacion{SolicitudRef: p.Solicitud.SolicitudRef, EmpleadoRef: p.Solicitud.EmpleadoRef, CatalogoVersionRef: p.Solicitud.CatalogoVersionRef, PermisoRef: p.Solicitud.PermisoRef, ExpedienteDocumentalRef: p.Solicitud.ExpedienteDocumentalRef, Documento: in.Documento}
	m := materialJustificacion(a, p, v, in.ClaveOperacion, domain.AccionAnexarJustificacion, in.VersionEsperada)
	if v.Validar(p.Solicitud, p.Politica) != nil {
		return ports.ResultadoAnexoJustificacion{}, domain.ErrJustificacionInvalida
	}
	if _, e = m.Canonico(); e != nil {
		return ports.ResultadoAnexoJustificacion{}, e
	}
	// Preflight antes incluso del replay: no emplear una autoridad histórica.
	if e = s.documentos.PrepararRegistro(ctx, o, p.Solicitud, p.Politica); e != nil {
		return ports.ResultadoAnexoJustificacion{}, e
	}
	r, ok, e := s.repo.RecuperarJustificacion(ctx, o, m)
	if e != nil {
		return ports.ResultadoAnexoJustificacion{}, e
	}
	if ok {
		if !reciboJustificacionCoherente(r, m, p) {
			return ports.ResultadoAnexoJustificacion{}, ports.ErrJustificacionNoDisponible
		}
		r.Replay = true
		d := r.Justificacion.Vinculo.Documento
		return ports.ResultadoAnexoJustificacion{Documento: &d, ReciboCronos: &r}, nil
	}
	siguiente, e := domain.PrepararAnexoJustificacion(p.Solicitud, p.Politica, p.Actual, v, in.VersionEsperada)
	if e != nil {
		return ports.ResultadoAnexoJustificacion{}, e
	}
	// Solicitar autoridad Cronos antes del alta; el repositorio la revalida al
	// confirmar. No se considera permiso por el actor ni por la preimagen sola.
	autorizacion, e := o.ProveedorMaterial().ProveerMaterialJustificacion(ctx, m)
	if e != nil {
		return ports.ResultadoAnexoJustificacion{}, e
	}
	if !s.materialLigado(autorizacion, m) {
		return ports.ResultadoAnexoJustificacion{}, ports.ErrJustificacionNoDisponible
	}
	d, e := s.documentos.RegistrarJustificante(ctx, o, p.Solicitud, p.Politica, in.Documento, in.ClaveOperacion)
	if e != nil {
		return ports.ResultadoAnexoJustificacion{}, e
	}
	if d != in.Documento {
		return ports.ResultadoAnexoJustificacion{EnlacePendiente: true}, ports.ErrEnlaceJustificacionPendiente
	}
	resultado := ports.ResultadoAnexoJustificacion{Documento: &d, EnlacePendiente: true}
	r, e = s.repo.ConfirmarJustificacion(ctx, m, siguiente, autorizacion)
	if e != nil {
		return resultado, errors.Join(ports.ErrEnlaceJustificacionPendiente, e)
	}
	if !reciboJustificacionCoherente(r, m, p) {
		return resultado, ports.ErrEnlaceJustificacionPendiente
	}
	resultado.EnlacePendiente = false
	resultado.ReciboCronos = &r
	return resultado, nil
}
func (s *ServicioJustificacion) Revisar(ctx context.Context, o ports.OrdenJustificacion, in ports.PeticionRevisionJustificacion) (ports.ReciboJustificacion, error) {
	a, p, e := s.preparar(ctx, o, in.SolicitudRef)
	if e != nil {
		return ports.ReciboJustificacion{}, e
	}
	if in.Vinculo.Validar(p.Solicitud, p.Politica) != nil {
		return ports.ReciboJustificacion{}, domain.ErrJustificacionInvalida
	}
	m := materialJustificacion(a, p, in.Vinculo, in.ClaveOperacion, domain.AccionRevisarJustificacion, in.VersionEsperada)
	m.Decision = in.Decision
	m.MotivoRef = in.MotivoRef
	if _, e = m.Canonico(); e != nil {
		return ports.ReciboJustificacion{}, e
	}
	r, ok, e := s.repo.RecuperarJustificacion(ctx, o, m)
	if e != nil {
		return ports.ReciboJustificacion{}, e
	}
	if ok {
		if !reciboJustificacionCoherente(r, m, p) {
			return ports.ReciboJustificacion{}, ports.ErrJustificacionNoDisponible
		}
		r.Replay = true
		return r, nil
	}
	if p.Actual == nil {
		return ports.ReciboJustificacion{}, domain.ErrJustificacionConflicto
	}
	siguiente, e := domain.PrepararRevisionJustificacion(p.Solicitud, p.Politica, *p.Actual, in.Vinculo, in.VersionEsperada, in.Decision, in.MotivoRef)
	if e != nil {
		return ports.ReciboJustificacion{}, e
	}
	autorizacion, e := o.ProveedorMaterial().ProveerMaterialJustificacion(ctx, m)
	if e != nil {
		return ports.ReciboJustificacion{}, e
	}
	if !s.materialLigado(autorizacion, m) {
		return ports.ReciboJustificacion{}, ports.ErrJustificacionNoDisponible
	}
	r, e = s.repo.ConfirmarJustificacion(ctx, m, siguiente, autorizacion)
	if e != nil {
		return ports.ReciboJustificacion{}, e
	}
	if !reciboJustificacionCoherente(r, m, p) {
		return ports.ReciboJustificacion{}, ports.ErrJustificacionNoDisponible
	}
	return r, nil
}

// Validación estructural y ligadura previa; la comprobación criptográfica y
// el consumo siguen perteneciendo a la transacción del repositorio.
func (s *ServicioJustificacion) materialLigado(a vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, m domain.MaterialJustificacion) bool {
	h, e := m.Huella()
	r := a.ResumenCapacidad()
	ahora := s.reloj.AhoraUTC()
	return e == nil && a.ValidarEstructura() == nil && r.Operacion() == m.Accion && r.EfectoRef() == m.Vinculo.SolicitudRef && r.EfectoHuellaSHA256() == h && r.AudienciaConsumo() == AudienciaJustificacion && !ahora.Before(r.EmitidaEn()) && ahora.Before(r.ExpiraEn())
}

const AudienciaJustificacion = "vec_cronos_v1.justificacion.v1"
