package smtp

import (
	"context"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type enviadorSMTPPrueba struct {
	resultado Resultado
	recibido  Mensaje
}

func (e *enviadorSMTPPrueba) Enviar(_ context.Context, m Mensaje) Resultado {
	e.recibido = m
	return e.resultado
}

func TestTransportadorLlamamientoMapeaEstadosYContenidoEfimero(t *testing.T) {
	casos := []struct {
		estado   Estado
		esperado ports.EstadoCorreoLlamamiento
	}{
		{NoAceptadoTransitorio, ports.CorreoLlamamientoNoAceptadoTransitorio}, {NoAceptadoPermanente, ports.CorreoLlamamientoNoAceptadoPermanente}, {Indeterminado, ports.CorreoLlamamientoIndeterminado}, {AceptadoPorRelay, ports.CorreoLlamamientoAceptadoPorRelay}, {Estado(99), ports.CorreoLlamamientoIndeterminado},
	}
	for _, caso := range casos {
		t.Run("estado", func(t *testing.T) {
			doble := &enviadorSMTPPrueba{resultado: Resultado{Estado: caso.estado}}
			transportador, err := nuevoTransportadorLlamamiento(doble)
			if err != nil {
				t.Fatal(err)
			}
			m, err := ports.NuevoMensajeCorreoLlamamiento("persona@prueba.local", "asunto", "cuerpo")
			if err != nil {
				t.Fatal(err)
			}
			m.MessageID, m.FechaOrigen = "<intento@prueba.local>", time.Date(2026, 9, 12, 10, 0, 0, 0, time.UTC)
			if got := transportador.Enviar(context.Background(), m); got != caso.esperado {
				t.Fatalf("estado=%v", got)
			}
			if doble.recibido.Destino != "persona@prueba.local" || doble.recibido.Asunto != "asunto" || doble.recibido.Cuerpo != "cuerpo" || doble.recibido.MessageID != m.MessageID || !doble.recibido.FechaOrigen.Equal(m.FechaOrigen) {
				t.Fatal("mensaje no conservado")
			}
		})
	}
}

func TestTransportadorLlamamientoRechazaDependenciasNulas(t *testing.T) {
	var nulo *enviadorSMTPPrueba
	if _, err := nuevoTransportadorLlamamiento(nulo); err != ErrTransportadorCorreoLlamamientoInvalido {
		t.Fatalf("err=%v", err)
	}
	if got := (&TransportadorLlamamiento{}).Enviar(context.Background(), ports.MensajeCorreoLlamamiento{}); got != ports.CorreoLlamamientoIndeterminado {
		t.Fatalf("estado=%v", got)
	}
}
