package portafirmasapagado

import (
	"context"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestConectorApagadoNoConectaNiEnvia(t *testing.T) {
	estado, err := Conector{}.EstadoConexion(context.Background())
	if err != nil || estado.Conectado || estado.Motivo != ports.MotivoPortafirmasConexionPendiente || estado.Validar() != nil {
		t.Fatalf("estado = %+v, %v", estado, err)
	}
	recibo, err := Conector{}.Enviar(context.Background(), ports.SolicitudEnvioPortafirmas{Original: []byte("%PDF-")})
	if !errors.Is(err, ports.ErrPortafirmasNoDisponible) || recibo != (ports.ReciboEnvioPortafirmas{}) {
		t.Fatalf("envío = %+v, %v", recibo, err)
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := (Conector{}).EstadoConexion(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("contexto cancelado: %v", err)
	}
	//nolint:staticcheck // se comprueba el rechazo de un contexto nulo.
	if _, err := (Conector{}).EstadoConexion(nil); !errors.Is(err, ports.ErrPortafirmasNoDisponible) {
		t.Fatalf("contexto nulo: %v", err)
	}
}
