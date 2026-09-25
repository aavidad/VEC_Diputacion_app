package ports

import (
	"context"
	"testing"

	"vec-diputacion-granada/internal/vec/domain"
)

type emisorContextoPuerto struct{ recibidas int }

func (e *emisorContextoPuerto) Emitir(domain.SolicitudIncidenciaTecnica) { e.recibidas++ }

func TestEmisorIncidenciasEnContexto(t *testing.T) {
	solicitud := domain.SolicitudIncidenciaTecnica{Codigo: domain.IncidenciaHTTPInternoFallido, Componente: domain.ComponenteIncidenciaHTTP, Etapa: domain.EtapaIncidenciaPeticion}
	// Sin emisor, con contexto nil o con emisor nil: no hace nada ni falla.
	EmitirIncidenciaTecnicaDesdeContexto(context.Background(), solicitud)
	EmitirIncidenciaTecnicaDesdeContexto(nil, solicitud)
	if ctx := ConEmisorIncidenciasTecnicas(context.Background(), nil); ctx != context.Background() {
		t.Fatal("un emisor nil no debe derivar el contexto")
	}
	emisor := &emisorContextoPuerto{}
	ctx := ConEmisorIncidenciasTecnicas(context.Background(), emisor)
	EmitirIncidenciaTecnicaDesdeContexto(ctx, solicitud)
	if emisor.recibidas != 1 {
		t.Fatalf("recibidas = %d", emisor.recibidas)
	}
}
