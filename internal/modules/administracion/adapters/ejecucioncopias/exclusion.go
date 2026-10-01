package ejecucioncopias

import (
	"context"
	cs04 "vec-diputacion-granada/internal/modules/administracion/application/capturacopias"
	"vec-diputacion-granada/internal/modules/administracion/domain/copias"
	capturap "vec-diputacion-granada/internal/modules/administracion/ports/capturacopias"
	p "vec-diputacion-granada/internal/modules/administracion/ports/ejecucioncopias"
)

// MantenimientoExterno conserva admisión cerrada fuera de los volúmenes que
// se sustituyen. Una marca declarada en JSON no implementa este contrato.
type MantenimientoExterno interface {
	Mantener(context.Context, string, string) error
}
type exclusionCS04 struct {
	owner             *CapturaCS04
	servicio          cs04.Servicio
	ventana           *cs04.Ventana
	solicitud         capturap.Peticion
	lectura           p.Lectura
	logico            *logicoMedido
	retenida, cerrada bool
}

func (v *CapturaCS04) AbrirRestauracion(ctx context.Context, peticion p.Peticion) (p.Exclusion, error) {
	if v == nil || ctx == nil || v.InventarioActual == nil || v.Mantenimiento == nil || v.Medidor == nil || v.Material == nil || v.Servicio.Logico == nil || v.Servicio.Componentes == nil || v.Servicio.Ahora == nil || v.DuracionVentana <= 0 {
		return nil, ErrCapturaEnsemble
	}
	if !v.mu.TryLock() {
		return nil, ErrCapturaEnsemble
	}
	lectura, err := v.InventarioActual.LeerActual(ctx, peticion.DestinoRef)
	if err != nil || copias.CompararInventarios(lectura.Esperado, lectura.Observado).Estado != copias.Compatible {
		v.mu.Unlock()
		return nil, ErrCapturaEnsemble
	}
	servicio := v.Servicio
	logico := &logicoMedido{captura: servicio.Logico, medidor: v.Medidor}
	servicio.Logico = logico
	inicio := servicio.Ahora().UTC()
	solicitud := capturap.Peticion{OrigenRef: lectura.Observado.PostgreSQL.ClusterRef, OperacionRef: peticion.OperacionRef, Esperado: lectura.Esperado, InicioVentana: inicio, FinVentana: inicio.Add(v.DuracionVentana)}
	ventana, err := servicio.AbrirVentana(ctx, solicitud)
	if err != nil {
		v.mu.Unlock()
		return nil, ErrCapturaEnsemble
	}
	return &exclusionCS04{owner: v, servicio: servicio, ventana: ventana, solicitud: solicitud, lectura: lectura, logico: logico}, nil
}
func (x *exclusionCS04) CapturarPrevia(ctx context.Context, peticion p.Peticion, l p.Lectura) (p.Captura, error) {
	if x.cerrada || x.retenida || peticion.OperacionRef != x.solicitud.OperacionRef || copias.HuellaInventario(l.Observado) != copias.HuellaInventario(x.lectura.Observado) || x.servicio.ComprobarVentana(ctx, x.ventana, x.solicitud) != nil {
		return p.Captura{}, ErrCapturaEnsemble
	}
	parcial, err := x.servicio.CapturarEnVentana(ctx, x.ventana)
	if err != nil || parcial.Estado != "captura_parcial_pendiente" {
		return p.Captura{}, ErrCapturaEnsemble
	}
	return x.owner.convertir(ctx, peticion, l, parcial, x.logico.origen)
}
func (x *exclusionCS04) PreimagenActual(ctx context.Context) (string, error) {
	if x.cerrada || x.retenida || x.servicio.ComprobarVentana(ctx, x.ventana, x.solicitud) != nil {
		return "", ErrCapturaEnsemble
	}
	i, err := x.servicio.Inventario.Observar(ctx)
	if err != nil {
		return "", ErrCapturaEnsemble
	}
	e, err := x.owner.Medidor.Medir(ctx, i)
	if err != nil {
		return "", ErrCapturaEnsemble
	}
	if x.servicio.ComprobarVentana(ctx, x.ventana, x.solicitud) != nil {
		return "", ErrCapturaEnsemble
	}
	return PreimagenDeObservacion(i, e)
}
func (x *exclusionCS04) Cerrar(context.Context) error {
	if x.cerrada {
		return nil
	}
	if x.retenida {
		return ErrCapturaEnsemble
	}
	err := x.servicio.CerrarVentana(x.ventana)
	x.cerrada = true
	x.owner.mu.Unlock()
	return err
}
func (x *exclusionCS04) ConservarMantenimiento(ctx context.Context) error {
	if x.cerrada {
		return ErrCapturaEnsemble
	}
	if err := x.owner.Mantenimiento.Mantener(ctx, x.solicitud.OperacionRef, x.solicitud.OrigenRef); err != nil {
		return ErrCapturaEnsemble
	}
	x.retenida = true
	// La capability de escritores se conserva. No CerrarVentana: reabriría
	// admisión. El diario exterior permite la conciliación en otra ejecución.
	return nil
}
