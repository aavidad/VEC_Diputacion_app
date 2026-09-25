package ports

import (
	"context"

	"vec-diputacion-granada/internal/vec/domain"
)

// claveEmisorIncidenciasTecnicas es privada: solo este paquete puede colocar
// o leer el emisor del contexto.
type claveEmisorIncidenciasTecnicas struct{}

// ConEmisorIncidenciasTecnicas devuelve un contexto que transporta el emisor
// de la composición. Lo coloca la frontera de transporte (middleware del
// servidor) para que un adaptador pueda declarar una incidencia cerrada del
// catálogo sin recibir el emisor por su constructor. Un emisor nil deja el
// contexto intacto.
func ConEmisorIncidenciasTecnicas(ctx context.Context, emisor EmisorIncidenciasTecnicas) context.Context {
	if ctx == nil || emisor == nil {
		return ctx
	}
	return context.WithValue(ctx, claveEmisorIncidenciasTecnicas{}, emisor)
}

// EmitirIncidenciaTecnicaDesdeContexto declara la solicitud con el emisor del
// contexto. Sin emisor no hace nada: la supervisión es una capa y su ausencia
// nunca cambia la respuesta ni bloquea al llamante.
func EmitirIncidenciaTecnicaDesdeContexto(ctx context.Context, solicitud domain.SolicitudIncidenciaTecnica) {
	if ctx == nil {
		return
	}
	if emisor, ok := ctx.Value(claveEmisorIncidenciasTecnicas{}).(EmisorIncidenciasTecnicas); ok && emisor != nil {
		emisor.Emitir(solicitud)
	}
}
