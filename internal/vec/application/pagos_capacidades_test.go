package application

import (
	"context"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/vec/ports"
)

func TestRepositorioAltaCobroConsumeTokenYNoConfirmaDosVeces(t *testing.T) {
	escenario := nuevoEscenarioAltaCobroPrueba(t)
	if _, err := escenario.servicio.Crear(context.Background(), escenario.solicitud); err != nil {
		t.Fatalf("Crear() error = %v", err)
	}
	escenario.repositorio.mu.Lock()
	confirmacion := escenario.repositorio.ultimaConfirmacion
	auditoriaID := escenario.repositorio.auditoria.ID
	eventoID := escenario.repositorio.evento.ID
	if confirmacion.Token.Valido() || escenario.repositorio.huellaTokenReserva != "" {
		escenario.repositorio.mu.Unlock()
		t.Fatal("el repositorio retuvo la capacidad consumida o su huella")
	}
	escenario.repositorio.mu.Unlock()

	tokenReplay, err := ports.NuevoTokenReservaOrdenCobro()
	if err != nil {
		t.Fatalf("generar capacidad para probar replay: %v", err)
	}
	confirmacion.Token = tokenReplay
	err = escenario.repositorio.ConfirmarCreacion(context.Background(), confirmacion)
	if !errors.Is(err, ports.ErrMutacionOrdenCobroInvalida) {
		t.Fatalf("segunda ConfirmarCreacion() error = %v", err)
	}
	escenario.repositorio.mu.Lock()
	defer escenario.repositorio.mu.Unlock()
	version, _, errControl := escenario.repositorio.orden.ControlConcurrencia()
	if errControl != nil || version != 1 || escenario.repositorio.confirmaciones != 2 ||
		escenario.repositorio.auditoria.ID != auditoriaID || escenario.repositorio.evento.ID != eventoID {
		t.Fatalf("el token consumido duplico o altero efectos: %+v", escenario.repositorio)
	}
}
