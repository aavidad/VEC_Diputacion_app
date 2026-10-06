package telemetria

import (
	"context"
	"io"
	"net/http"
	"sync"
	"time"
)

const plazoCierreRegistro = 2 * time.Second

// MontarEnServidor crea el registro de acceso y envuelve srv.Handler. Debe
// llamarse antes de componer la supervisión de respuestas, para que esta
// quede por fuera y aporte la correlación. Si el registro no puede crearse,
// deja una línea fija en avisos y el servidor sigue igual: la telemetría
// nunca impide arrancar. Devuelve el registro (o nil) y su cierre
// idempotente, que también se registra en el apagado ordenado.
func MontarEnServidor(srv *http.Server, o Opciones, avisos io.Writer) (*Registro, func()) {
	if srv == nil || srv.Handler == nil {
		return nil, func() {}
	}
	reg, err := NuevoRegistro(o)
	if err != nil {
		if avisos != nil {
			_, _ = io.WriteString(avisos, "telemetria: registro de acceso no disponible\n")
		}
		return nil, func() {}
	}
	srv.Handler = reg.Envolver(srv.Handler)
	var una sync.Once
	cerrar := func() {
		una.Do(func() {
			ctx, cancelar := context.WithTimeout(context.Background(), plazoCierreRegistro)
			defer cancelar()
			if reg.Cerrar(ctx) != nil && avisos != nil {
				_, _ = io.WriteString(avisos, "telemetria: lineas de acceso pendientes sin vaciar al cerrar\n")
			}
		})
	}
	srv.RegisterOnShutdown(cerrar)
	return reg, cerrar
}
