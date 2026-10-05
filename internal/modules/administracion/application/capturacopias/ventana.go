package capturacopias

import (
	"context"
	"sync"
	"time"

	"vec-diputacion-granada/internal/modules/administracion/domain/copias"
	puertos "vec-diputacion-granada/internal/modules/administracion/ports/capturacopias"
)

// Ventana solo se obtiene mediante AbrirVentana. Sus campos privados vinculan
// operación, origen, inventario y exclusión, y excluyen uso paralelo/cerrado.
type Ventana struct {
	mu          sync.Mutex
	servicio    Servicio
	peticion    puertos.Peticion
	observado   copias.Inventario
	huella      string
	inicio      time.Time
	ctx         context.Context
	cancel      context.CancelFunc
	liberar     func() error
	activa      bool
	usada       bool
	errorCierre error
}

func (w *Ventana) vigente(ctx context.Context) bool {
	return w.activa && w.ctx != nil && ctx.Err() == nil && w.ctx.Err() == nil && enVentana(w.servicio.Ahora(), w.peticion)
}

func (w *Ventana) cerrar() error {
	if !w.activa {
		return w.errorCierre
	}
	w.activa = false
	// Reabrir también después de cierre parcial o cancelación. El propietario
	// conserva la ventana hasta llamar explícitamente aquí: no hay reapertura
	// automática entre copia previa y sustitución.
	limpieza, cancel := context.WithTimeout(context.WithoutCancel(w.ctx), w.servicio.TiempoLiberacion)
	defer cancel()
	defer w.cancel()
	if w.servicio.Escritores.Reabrir(limpieza) != nil {
		w.errorCierre = ErrLiberacion
	}
	if w.liberar() != nil {
		w.errorCierre = ErrLiberacion
	}
	return w.errorCierre
}

func (w *Ventana) contexto(ctx context.Context) (context.Context, func()) {
	actual, cancel := context.WithCancel(ctx)
	stop := context.AfterFunc(w.ctx, cancel)
	return actual, func() { stop(); cancel() }
}

// CapturarEnVentana no libera escritores. Solo admite una captura y usa siempre
// los puertos de la capability original, aunque se llame desde otro receptor.
func (s Servicio) CapturarEnVentana(ctx context.Context, w *Ventana) (r puertos.Parcial, err error) {
	r = puertos.Parcial{FormatoVersion: 1, Estado: "captura_fallida"}
	if w == nil || !w.mu.TryLock() {
		return r, ErrPrecondicion
	}
	defer w.mu.Unlock()
	if !w.vigente(ctx) || w.usada {
		return r, ErrPrecondicion
	}
	w.usada = true
	r.OperacionRef = w.peticion.OperacionRef
	r.Inicio = w.inicio
	r.InventarioSHA256 = w.huella
	defer func() { r.Fin = w.servicio.Ahora().UTC() }()
	ctx, cancel := w.contexto(ctx)
	defer cancel()
	if w.servicio.Escritores.ComprobarExclusion(ctx) != nil {
		return r, ErrControl
	}
	// A held window can outlive the original capture call. Reobserve before its
	// first effect as well as after capture; retain the original sealed digest.
	observado, e := w.servicio.Inventario.Observar(ctx)
	if e != nil || copias.HuellaInventario(observado) != w.huella {
		return r, ErrInventario
	}
	r.Componentes, e = w.servicio.Logico.Capturar(ctx, clonar(w.observado))
	if e != nil {
		return r, ErrCaptura
	}
	if w.servicio.Componentes != nil {
		if ctx.Err() != nil || w.servicio.Escritores.ComprobarExclusion(ctx) != nil {
			return r, ErrControl
		}
		componentes, e := w.servicio.Componentes.Capturar(ctx, clonar(w.observado))
		if e != nil {
			return r, ErrCaptura
		}
		r.Componentes = append(r.Componentes, componentes...)
	}
	final, e := w.servicio.Inventario.Observar(ctx)
	if e != nil || copias.HuellaInventario(final) != w.huella {
		return r, ErrInventario
	}
	if !w.vigente(ctx) || w.servicio.Escritores.ComprobarExclusion(ctx) != nil {
		return r, ErrControl
	}
	r.Estado = "captura_parcial_pendiente"
	return r, nil
}

// ComprobarVentana revalida la misma operación/origen/preimagen antes de un
// siguiente efecto de plataforma. No concede autorización ni sustituye CS10/11.
func (s Servicio) ComprobarVentana(ctx context.Context, w *Ventana, p puertos.Peticion) error {
	if w == nil || !w.mu.TryLock() {
		return ErrPrecondicion
	}
	defer w.mu.Unlock()
	if !w.vigente(ctx) || p.OrigenRef != w.peticion.OrigenRef || p.OperacionRef != w.peticion.OperacionRef || copias.HuellaInventario(p.Esperado) != copias.HuellaInventario(w.peticion.Esperado) {
		return ErrPrecondicion
	}
	ctx, cancel := w.contexto(ctx)
	defer cancel()
	if w.servicio.Escritores.ComprobarExclusion(ctx) != nil {
		return ErrControl
	}
	actual, err := w.servicio.Inventario.Observar(ctx)
	if err != nil || copias.HuellaInventario(actual) != w.huella {
		return ErrInventario
	}
	if !w.vigente(ctx) {
		return ErrPrecondicion
	}
	return nil
}
