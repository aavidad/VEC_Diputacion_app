// Package capturacopias coordina capturas. No concede permisos administrativos.
package capturacopias

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/administracion/domain/copias"
	puertos "vec-diputacion-granada/internal/modules/administracion/ports/capturacopias"
)

var (
	ErrPrecondicion = errors.New("copias_captura_precondicion")
	ErrInventario   = errors.New("copias_captura_inventario")
	ErrControl      = errors.New("copias_captura_control")
	ErrCaptura      = errors.New("copias_captura_fallo")
	ErrLiberacion   = errors.New("copias_captura_liberacion")
)

type Servicio struct {
	Exclusor         puertos.Exclusor
	Escritores       puertos.ControlEscritores
	Inventario       puertos.Inventariador
	Logico           puertos.CapturadorLogico
	Componentes      puertos.CapturadorComponentes
	Ahora            func() time.Time
	TiempoLiberacion time.Duration
}

// Capturar compone una ventana ordinaria y la libera incluso ante cancelación.
// Exige una petición ya autorizada por su composición, no sustituye CS07/08.
func (s Servicio) Capturar(ctx context.Context, p puertos.Peticion) (r puertos.Parcial, err error) {
	r = puertos.Parcial{FormatoVersion: 1, OperacionRef: p.OperacionRef, Estado: "captura_fallida"}
	w, e := s.AbrirVentana(ctx, p)
	if e != nil {
		return r, e
	}
	defer func() {
		if s.CerrarVentana(w) != nil {
			err = ErrLiberacion
			r.Estado = "captura_fallida"
		}
	}()
	return s.CapturarEnVentana(ctx, w)
}

// AbrirVentana transfiere al propietario servidor una capability opaca. Debe
// cerrar la ventana después de capturar/sustituir, también si el contexto vence.
// Ningún booleano, JSON o cliente puede construir una ventana activa.
func (s Servicio) AbrirVentana(ctx context.Context, p puertos.Peticion) (w *Ventana, err error) {
	if s.Exclusor == nil || s.Escritores == nil || s.Inventario == nil || s.Logico == nil || s.Ahora == nil || s.TiempoLiberacion <= 0 || p.OrigenRef == "" || p.OperacionRef == "" || !p.FinVentana.After(p.InicioVentana) {
		return nil, ErrPrecondicion
	}
	if ctx.Err() != nil || p.OrigenRef != p.Esperado.PostgreSQL.ClusterRef || len(copias.ValidarInventario(p.Esperado)) != 0 || !enVentana(s.Ahora(), p) {
		return nil, ErrPrecondicion
	}
	p.Esperado = clonar(p.Esperado)
	ventanaCtx, cancel := context.WithDeadline(ctx, p.FinVentana)
	liberar, e := s.Exclusor.Adquirir(ventanaCtx, p.OrigenRef)
	if e != nil || liberar == nil {
		cancel()
		return nil, ErrControl
	}
	w = &Ventana{servicio: s, peticion: p, ctx: ventanaCtx, cancel: cancel, liberar: liberar, activa: true, inicio: s.Ahora().UTC()}
	defer func() {
		if err != nil {
			if w.cerrar() != nil {
				err = ErrLiberacion
			}
			w = nil
		}
	}()
	if s.Escritores.CerrarAdmision(ventanaCtx) != nil || s.Escritores.Drenar(ventanaCtx) != nil || s.Escritores.ComprobarExclusion(ventanaCtx) != nil {
		return w, ErrControl
	}
	observado, e := s.Inventario.Observar(ventanaCtx)
	if e != nil || copias.CompararInventarios(p.Esperado, observado).Estado != copias.Compatible {
		return w, ErrInventario
	}
	w.observado = clonar(observado)
	w.huella = copias.HuellaInventario(observado)
	return w, nil
}

// CerrarVentana es idempotente y opera siempre con los puertos originales de
// la capability. Una ventana cerrada nunca vuelve a ser utilizable.
func (s Servicio) CerrarVentana(w *Ventana) error {
	if w == nil {
		return ErrPrecondicion
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.cerrar()
}

func enVentana(t time.Time, p puertos.Peticion) bool {
	return !t.Before(p.InicioVentana) && t.Before(p.FinVentana)
}
