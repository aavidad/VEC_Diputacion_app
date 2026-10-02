package postgres

import (
	"errors"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestSenalesSQLEntregaDistinguenCreacionYRecuperacion(t *testing.T) {
	verdadero, falso := true, false
	casos := []struct {
		nombre, modo, estado  string
		reserva, confirmacion *bool
		valido                bool
	}{
		{"reserva nueva", "preparar", "preparada", &verdadero, &falso, true},
		{"reserva recuperada", "preparar", "preparada", &falso, &falso, true},
		{"confirmacion nueva", "confirmar", "confirmada", &falso, &verdadero, true},
		{"confirmacion recuperada", "confirmar", "confirmada", &falso, &falso, true},
		{"reconciliacion historica", "preparar", "confirmada", &falso, &verdadero, true},
		{"sin señal de reserva", "preparar", "preparada", nil, &falso, false},
		{"sin señal de confirmacion", "confirmar", "confirmada", &falso, nil, false},
		{"dos inserciones en una llamada SQL", "confirmar", "confirmada", &verdadero, &verdadero, false},
		{"confirmacion con reserva nueva", "confirmar", "confirmada", &verdadero, &falso, false},
		{"confirmacion en estado preparado", "preparar", "preparada", &falso, &verdadero, false},
		{"confirmar no confirmado", "confirmar", "preparada", &falso, &falso, false},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			s := resultadoEntregaPeticionCentroSQL{
				EntregaPeticionCentro: ports.EntregaPeticionCentro{EstadoEntrega: caso.estado},
				ReservaCreadaAhora:    caso.reserva, ConfirmacionCreadaAhora: caso.confirmacion,
			}
			e, err := s.entregaPara(caso.modo, false)
			if caso.valido {
				if err != nil || e.ReservaCreadaAhora != *caso.reserva || e.ConfirmadaAhora != *caso.confirmacion {
					t.Fatalf("señales válidas rechazadas: %+v, %v", e, err)
				}
			} else if !errors.Is(err, ports.ErrReciboPeticionCentroNoConfiable) {
				t.Fatalf("señales inválidas admitidas: %+v, %v", e, err)
			}
		})
	}
}

func TestRespuestaSQLAntiguaSinSenalesFallaCerrada(t *testing.T) {
	var s resultadoEntregaPeticionCentroSQL
	if err := decodificarJSONEstricto([]byte(`{"estado_entrega":"preparada"}`), &s); err != nil {
		t.Fatal(err)
	}
	if _, err := s.entregaPara("preparar", false); !errors.Is(err, ports.ErrReciboPeticionCentroNoConfiable) {
		t.Fatalf("respuesta SQL antigua admitida: %v", err)
	}
}

func TestPrepararEntregaSinMOADRevierteReservaNueva(t *testing.T) {
	verdadero, falso := true, false
	s := resultadoEntregaPeticionCentroSQL{
		EntregaPeticionCentro: ports.EntregaPeticionCentro{EstadoEntrega: "preparada"},
		ReservaCreadaAhora:    &verdadero, ConfirmacionCreadaAhora: &falso,
	}
	if _, err := s.entregaPara("preparar", true); !errors.Is(err, ports.ErrNumeroMOADAusente) {
		t.Fatalf("una reserva nueva sin MOAD alcanzaría COMMIT: %v", err)
	}
	s.ReservaCreadaAhora = &falso
	if _, err := s.entregaPara("preparar", true); err != nil {
		t.Fatalf("la reserva anterior no pudo recuperarse: %v", err)
	}
}
