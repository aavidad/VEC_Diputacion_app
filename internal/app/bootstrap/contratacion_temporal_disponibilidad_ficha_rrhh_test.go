package bootstrap

import (
	"context"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type lectorDisponibilidadFichaPrueba struct {
	detalle  ports.DetalleExpedienteRRHH
	err      error
	llamadas int
}

func (l *lectorDisponibilidadFichaPrueba) Consultar(_ context.Context, _ ports.SolicitudDetalleRRHH) (ports.DetalleExpedienteRRHH, error) {
	l.llamadas++
	return l.detalle, l.err
}

func TestDisponibilidadFichaCTSoloTrasLecturaCorrecta(t *testing.T) {
	for _, tc := range []struct {
		montado  bool
		esperado ports.EstadoDisponibilidadBorradoresRRHH
	}{
		{false, ports.BorradoresRRHHSinMontaje},
		{true, ports.BorradoresRRHHMontado},
	} {
		lector := &lectorDisponibilidadFichaPrueba{}
		resultado, err := (consultorDetalleConDisponibilidadRRHH{lector: lector, documentalMontado: tc.montado}).Consultar(context.Background(), ports.SolicitudDetalleRRHH{})
		if err != nil || lector.llamadas != 1 || resultado.EstadoBorradoresPublicados != tc.esperado {
			t.Fatalf("montado=%v, estado=%q, llamadas=%d, err=%v", tc.montado, resultado.EstadoBorradoresPublicados, lector.llamadas, err)
		}
	}
	lector := &lectorDisponibilidadFichaPrueba{err: errors.New("lectura denegada")}
	resultado, err := (consultorDetalleConDisponibilidadRRHH{lector: lector, documentalMontado: true}).Consultar(context.Background(), ports.SolicitudDetalleRRHH{})
	if err == nil || resultado.EstadoBorradoresPublicados != "" {
		t.Fatalf("la lectura fallida obtuvo disponibilidad: %+v, %v", resultado, err)
	}
}
