package application

import (
	"context"
	"testing"
	"time"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

type repoC23 struct{ abiertas, resultados int }

func (r *repoC23) ContactosParticipacion(context.Context, string) ([]ports.ContactoLlamamientoOperativo, error) {
	return []ports.ContactoLlamamientoOperativo{{CandidatoRef: "can_abcdefghijklmnopqrstuv", Canal: "correo"}}, nil
}
func (r *repoC23) AbrirLlamamiento(context.Context, ports.AperturaLlamamientoOperativo) (ports.ReciboLlamamientoOperativo, error) {
	r.abiertas++
	return ports.ReciboLlamamientoOperativo{}, nil
}
func (r *repoC23) RegistrarResultadoLlamamiento(context.Context, ports.ResultadoLlamamientoOperativo) (ports.ReciboLlamamientoOperativo, error) {
	r.resultados++
	return ports.ReciboLlamamientoOperativo{}, nil
}
func instanteC23() time.Time { return time.Date(2026, 9, 18, 10, 0, 0, 0, time.UTC) }
func aperturaC23() ports.AperturaLlamamientoOperativo {
	a := instanteC23()
	return ports.AperturaLlamamientoOperativo{OperacionRef: "operacion:c23:abrir", LlamamientoRef: "llamamiento:c23:uno", ParticipacionRef: "participacion:c23:uno", ActorRef: "actor:rrhh:c23", Canal: ports.CanalCorreoLlamamiento, ComunicadoEn: a, PlazoRespuestaHasta: a.Add(time.Hour)}
}
func TestServicioLlamamientosOperativosValidaAntesDePersistir(t *testing.T) {
	r := &repoC23{}
	s, err := NuevoServicioLlamamientosOperativos(r)
	if err != nil {
		t.Fatal(err)
	}
	a := aperturaC23()
	a.PlazoRespuestaHasta = a.ComunicadoEn
	if _, err = s.Abrir(context.Background(), a); err == nil || r.abiertas != 0 {
		t.Fatalf("persistió apertura inválida: %v", err)
	}
	a = aperturaC23()
	if _, err = s.Abrir(context.Background(), a); err != nil || r.abiertas != 1 {
		t.Fatalf("no delegó apertura válida: %v", err)
	}
}
func TestServicioLlamamientosOperativosRechazaResultadoNoGobernado(t *testing.T) {
	r := &repoC23{}
	s, _ := NuevoServicioLlamamientosOperativos(r)
	_, err := s.RegistrarResultado(context.Background(), ports.ResultadoLlamamientoOperativo{OperacionRef: "operacion:c23:resultado", LlamamientoRef: "llamamiento:c23:uno", ActorRef: "actor:rrhh:c23", Resultado: "inventado", RegistradoEn: instanteC23()})
	if err == nil || r.resultados != 0 {
		t.Fatalf("aceptó resultado no gobernado: %v", err)
	}
}
