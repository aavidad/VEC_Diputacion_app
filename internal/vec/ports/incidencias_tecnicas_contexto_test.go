package ports

import (
	"context"
	"testing"

	"vec-diputacion-granada/internal/vec/domain"
)

type emisorContextoPuerto struct{ recibidas int }

func (e *emisorContextoPuerto) Emitir(domain.SolicitudIncidenciaTecnica) { e.recibidas++ }

func TestEmitirIncidenciaTecnicaEnPeticionMarcaLaPeticion(t *testing.T) {
	solicitud := domain.SolicitudIncidenciaTecnica{Codigo: domain.IncidenciaOSRMNoDisponible, Componente: domain.ComponenteIncidenciaOSRM, Etapa: domain.EtapaIncidenciaConsulta}
	ctx, declarada := ConMarcaIncidenciasPeticion(context.Background())
	// Sin emisor: ni emite ni marca.
	EmitirIncidenciaTecnicaEnPeticion(ctx, nil, solicitud)
	if declarada() {
		t.Fatal("sin emisor no hay incidencia declarada")
	}
	emisor := &emisorContextoPuerto{}
	EmitirIncidenciaTecnicaEnPeticion(ctx, emisor, solicitud)
	if emisor.recibidas != 1 || !declarada() {
		t.Fatalf("recibidas = %d, declarada = %v", emisor.recibidas, declarada())
	}
	// Sin marca en el contexto (o contexto nil) emite igualmente.
	EmitirIncidenciaTecnicaEnPeticion(context.Background(), emisor, solicitud)
	EmitirIncidenciaTecnicaEnPeticion(nil, emisor, solicitud)
	if emisor.recibidas != 3 {
		t.Fatalf("recibidas = %d", emisor.recibidas)
	}
	if ctx, declarada := ConMarcaIncidenciasPeticion(nil); ctx == nil || declarada() {
		t.Fatal("contexto nil sin marca utilizable")
	}
	EmisorIncidenciasTecnicasNulo{}.Emitir(solicitud)
}
