package application

import (
	"context"
	"errors"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

var ErrResolucionFormalizacionNoDisponible = errors.New("contratacion temporal: resolucion de formalizacion no disponible")

type ServicioResolucionFormalizacion struct {
	tx ports.TransaccionResolucionFormalizacion
}

type ServicioPreparacionResolucionFormalizacion struct {
	autoridad ports.AutoridadContextoConsultaRRHH
	emisor    *ports.EmisorMaterialConsultaRRHH
	sesion    ports.SesionPreparacionResolucionFormalizacion
	reloj     ports.Reloj
}

func NuevoServicioPreparacionResolucionFormalizacion(a ports.AutoridadContextoConsultaRRHH, e *ports.EmisorMaterialConsultaRRHH,
	s ports.SesionPreparacionResolucionFormalizacion, r ports.Reloj) (*ServicioPreparacionResolucionFormalizacion, error) {
	if dependenciaNula(a) || e == nil || dependenciaNula(s) || dependenciaNula(r) {
		return nil, ports.ErrResolucionFormalizacionNoDisponible
	}
	return &ServicioPreparacionResolucionFormalizacion{a, e, s, r}, nil
}

// Cada lectura reutiliza íntegro el caso de uso nominal de detalle. El captor
// pertenece solo a esta llamada; no conserva estado entre peticiones.
func (s *ServicioPreparacionResolucionFormalizacion) ConsultarPreparacionResolucionFormalizacion(ctx context.Context, expediente string) (ports.PreparacionResolucionFormalizacion, error) {
	z := ports.PreparacionResolucionFormalizacion{}
	if s == nil || ctx == nil || dependenciaNula(s.sesion) {
		return z, ports.ErrResolucionFormalizacionNoDisponible
	}
	q, err := ports.NuevaSolicitudDetalleRRHH(expediente, 0)
	if err != nil {
		return z, ports.ErrSolicitudResolucionFormalizacionInvalida
	}
	captor := &sesionPreparacionResolucion{sesion: s.sesion}
	detalle, err := NuevoServicioConsultaDetalleRRHH(s.autoridad, s.emisor, captor, s.reloj)
	if err != nil {
		return z, ports.ErrResolucionFormalizacionNoDisponible
	}
	d, err := detalle.Consultar(ctx, q)
	if ctx.Err() != nil {
		return z, ctx.Err()
	}
	if err != nil {
		return z, err
	}
	p := captor.preparacion
	if p.ValidarPara(expediente) != nil || d.Resumen.Version != p.VersionActual ||
		len(d.Hitos) != int(p.VersionActual) || d.Hitos[6].AccionClave != "registrar_propuesta_formalizacion" ||
		(p.VersionActual == 8 && (d.Hitos[7].AccionClave != "registrar_resolucion_formalizacion" ||
			!d.Hitos[7].RealizadaEn.Equal(p.Recibo.RegistradaEn))) {
		return z, ports.ErrResultadoResolucionFormalizacionNoConfiable
	}
	return p.Clonar(), nil
}

type sesionPreparacionResolucion struct {
	sesion      ports.SesionPreparacionResolucionFormalizacion
	preparacion ports.PreparacionResolucionFormalizacion
}

func (s *sesionPreparacionResolucion) ConsultarCuadroYRegistrar(context.Context, ports.OrdenConsultaCuadroRRHH) (ports.PaginaCuadroRRHH, error) {
	return ports.PaginaCuadroRRHH{}, ports.ErrConsultaRRHHNoDisponible
}
func (s *sesionPreparacionResolucion) ConsultarDetalleYRegistrar(ctx context.Context, o ports.OrdenConsultaDetalleRRHH) (ports.DetalleExpedienteRRHH, error) {
	d, p, err := s.sesion.ConsultarPreparacionYRegistrarAcceso(ctx, o)
	if err != nil {
		return ports.DetalleExpedienteRRHH{}, err
	}
	if p.ValidarPara(o.Solicitud().ExpedienteRef()) != nil {
		return ports.DetalleExpedienteRRHH{}, ports.ErrResultadoConsultaRRHHNoConfiable
	}
	s.preparacion = p.Clonar()
	return d, nil
}

func NuevoServicioResolucionFormalizacion(tx ports.TransaccionResolucionFormalizacion) (*ServicioResolucionFormalizacion, error) {
	if dependenciaNula(tx) {
		return nil, ErrResolucionFormalizacionNoDisponible
	}
	return &ServicioResolucionFormalizacion{tx}, nil
}
func (s *ServicioResolucionFormalizacion) RegistrarResolucionFormalizacion(ctx context.Context, q ports.SolicitudResolucionFormalizacion) (ports.ResultadoResolucionFormalizacion, error) {
	var z ports.ResultadoResolucionFormalizacion
	if s == nil || ctx == nil || dependenciaNula(s.tx) {
		return z, ErrResolucionFormalizacionNoDisponible
	}
	if ctx.Err() != nil {
		return z, ctx.Err()
	}
	if q.Validar() != nil {
		return z, ports.ErrSolicitudResolucionFormalizacionInvalida
	}
	r, e := s.tx.RegistrarResolucionFormalizacion(ctx, q)
	if ctx.Err() != nil {
		return z, ctx.Err()
	}
	if e != nil {
		if r != z {
			return z, ports.ErrResultadoResolucionFormalizacionNoConfiable
		}
		return z, e
	}
	if r.ValidarPara(q) != nil {
		return z, ports.ErrResultadoResolucionFormalizacionNoConfiable
	}
	return r, nil
}
