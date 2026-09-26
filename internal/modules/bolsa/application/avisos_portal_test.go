package application

import (
	"context"
	"testing"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
)

type consultaAvisosPortalPrueba struct{ conteos map[string]int }

func (c consultaAvisosPortalPrueba) ListarAvisosRRHH(_ context.Context, corte time.Time, _, _ int) ([]dominiobolsa.AvisoRRHH, error) {
	return []dominiobolsa.AvisoRRHH{{Tipo: dominiobolsa.AvisoSolicitudPortal, BolsaRef: "bolsa:1", Referencia: "solicitud-portal:1", Detalle: map[string]any{"solicitud": "pausa"}, Fecha: corte.Add(-time.Minute)}}, nil
}

func (c consultaAvisosPortalPrueba) ContarAvisosRRHH(context.Context, time.Time) (map[string]int, error) {
	return c.conteos, nil
}

func TestAvisosIncluyenElPortalDelCandidato(t *testing.T) {
	ahora := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	s, err := NuevoServicioAvisosRRHH(consultaAvisosPortalPrueba{conteos: map[string]int{"salto_orden": 1, "tres_anos": 0, "solicitud_portal": 1, "respuesta_portal": 2}}, func() time.Time { return ahora })
	if err != nil {
		t.Fatal(err)
	}
	pagina, err := s.Consultar(context.Background(), ConsultaAvisos{Limite: 5})
	if err != nil || pagina.Total != 4 || pagina.Conteos["respuesta_portal"] != 2 || len(pagina.Avisos) != 1 {
		t.Fatalf("avisos del portal: %+v %v", pagina, err)
	}
	s, _ = NuevoServicioAvisosRRHH(consultaAvisosPortalPrueba{conteos: map[string]int{"salto_orden": 0, "tres_anos": 0, "inventado": 1}}, func() time.Time { return ahora })
	if _, err := s.Consultar(context.Background(), ConsultaAvisos{Limite: 5}); err == nil {
		t.Fatal("tipo de aviso desconocido admitido")
	}
}

func TestAvisosIncluyenElEncadenamientoSoloSiLaConsultaLoTrae(t *testing.T) {
	ahora := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	s, _ := NuevoServicioAvisosRRHH(consultaAvisosPortalPrueba{conteos: map[string]int{"salto_orden": 0, "tres_anos": 1, "encadenamiento": 3}}, func() time.Time { return ahora })
	pagina, err := s.Consultar(context.Background(), ConsultaAvisos{Limite: 5})
	if err != nil || pagina.Total != 4 || pagina.Conteos[dominiobolsa.AvisoEncadenamiento] != 3 {
		t.Fatalf("encadenamiento: %+v %v", pagina, err)
	}
	s, _ = NuevoServicioAvisosRRHH(consultaAvisosPortalPrueba{conteos: map[string]int{"salto_orden": 0, "tres_anos": 1}}, func() time.Time { return ahora })
	if pagina, err = s.Consultar(context.Background(), ConsultaAvisos{Limite: 5}); err != nil {
		t.Fatal(err)
	}
	if _, incluido := pagina.Conteos[dominiobolsa.AvisoEncadenamiento]; incluido {
		t.Fatal("sin 000041 no se cuenta el encadenamiento")
	}
}
