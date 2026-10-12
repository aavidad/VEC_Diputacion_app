package application

import (
	"context"
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

type lectorRespuestasPortalRRHHPrueba struct {
	resultado ports.RespuestasPortalLlamamientoRRHH
	lecturas  int
}

func (l *lectorRespuestasPortalRRHHPrueba) ConsultarRespuestasPortalLlamamiento(
	_ context.Context, _ ports.ConsultaRespuestasPortalRRHH,
) (ports.RespuestasPortalLlamamientoRRHH, error) {
	l.lecturas++
	return l.resultado, nil
}

func TestRespuestasPortalRRHHRechazaDatosAjenosODuplicados(t *testing.T) {
	resultado, vinculo := resultadoContextoContactoPrueba(t)
	q := ports.ConsultaRespuestasPortalRRHH{BolsaRef: "bolsa:1", LlamamientoRef: "llamamiento:1",
		Vinculo: vinculo, ResultadoContexto: resultado}
	q.Captura.BolsaRef, q.Captura.LlamamientoRef = q.BolsaRef, q.LlamamientoRef
	lector := &lectorRespuestasPortalRRHHPrueba{resultado: ports.RespuestasPortalLlamamientoRRHH{
		LlamamientoRef: q.LlamamientoRef, Respuestas: []ports.RespuestaPortalRRHH{{ParticipacionRef: "participacion:1", Respuesta: "acepta"}},
	}}
	servicio, err := NuevoServicioRespuestasPortalRRHH(lector)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := servicio.ConsultarRespuestasPortalLlamamiento(context.Background(), q); err != nil || lector.lecturas != 1 {
		t.Fatalf("lectura autorizada: err=%v lecturas=%d", err, lector.lecturas)
	}
	lector.resultado.LlamamientoRef = "llamamiento:ajeno"
	if _, err := servicio.ConsultarRespuestasPortalLlamamiento(context.Background(), q); err == nil {
		t.Fatal("aceptó respuesta de otro llamamiento")
	}
	lector.resultado.LlamamientoRef = q.LlamamientoRef
	lector.resultado.Respuestas = []ports.RespuestaPortalRRHH{
		{ParticipacionRef: "participacion:1", Respuesta: "acepta"},
		{ParticipacionRef: "participacion:1", Respuesta: "renuncia"},
	}
	if _, err := servicio.ConsultarRespuestasPortalLlamamiento(context.Background(), q); err == nil {
		t.Fatal("aceptó respuestas contradictorias para la misma participación")
	}
}
