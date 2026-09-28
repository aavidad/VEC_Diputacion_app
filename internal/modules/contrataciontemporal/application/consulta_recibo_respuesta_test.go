package application

import (
	"context"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type lectorReciboRespuestaPrueba struct {
	resultado ports.ReciboRespuestaConsultado
	llamadas  int
}

func (l *lectorReciboRespuestaPrueba) ConsultarReciboRespuesta(_ context.Context, _ ports.SolicitudConsultaReciboRespuesta) (ports.ReciboRespuestaConsultado, error) {
	l.llamadas++
	return l.resultado, nil
}

func TestConsultaReciboRespuestaVerificaResultadoAntesDeEntregarlo(t *testing.T) {
	l := &lectorReciboRespuestaPrueba{resultado: ports.ReciboRespuestaConsultado{
		OrganizacionRef: "organizacion:otra", ComunicacionRef: "comunicacion:prueba",
		Respuesta: ports.RespuestaLlamamientoRenunciada, JustificanteRef: "justificante:prueba",
		ReciboRef: "recibo:prueba", AuditoriaRef: "auditoria:prueba",
		RegistradaEn: time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC), Estado: ports.EstadoRespuestaRecibidaRegistrada,
	}}
	s, err := NuevoServicioConsultaReciboRespuesta(l)
	if err != nil {
		t.Fatal(err)
	}
	r, err := s.Consultar(context.Background(), ports.SolicitudConsultaReciboRespuesta{OrganizacionRef: "organizacion:prueba", ComunicacionRef: "comunicacion:prueba"})
	if err != ports.ErrReciboRespuestaNoConfiable || r != (ports.ReciboRespuestaConsultado{}) || l.llamadas != 1 {
		t.Fatalf("resultado=%+v error=%v llamadas=%d", r, err, l.llamadas)
	}
}
