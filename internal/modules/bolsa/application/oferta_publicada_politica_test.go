package application

import (
	"context"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

type plazoOfertaVersionadaPrueba struct {
	bolsa  string
	legacy int
}

func (p *plazoOfertaVersionadaPrueba) PlazoDisposicion(context.Context, time.Time) (ports.PlazoOferta, time.Time, error) {
	p.legacy++
	return ports.PlazoOferta{}, time.Time{}, ports.ErrPlazoOfertaNoConfigurado
}
func (p *plazoOfertaVersionadaPrueba) PlazoDisposicionBolsa(_ context.Context, bolsa string, desde time.Time) (ports.PlazoOferta, time.Time, error) {
	p.bolsa = bolsa
	return ports.PlazoOferta{ReglaRef: "politica-ofertas:" + bolsa + ":3", HuellaCatalogo: strings.Repeat("a", 64), PoliticaVersion: 3, MunicipioSede: "18087", Unidad: "dias_habiles", Cantidad: 2, Computo: "administrativo", UltimoDia: "2026-09-29", Ejemplo: true, Calendarios: []string{"calendario:2026:1"}}, desde.Add(72 * time.Hour), nil
}

func TestPublicarOfertaUsaPoliticaVersionadaDeBolsa(t *testing.T) {
	ahora := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	repo := &repositorioOfertasPrueba{}
	plazo := &plazoOfertaVersionadaPrueba{}
	s, err := NuevoServicioOfertasPublicadas(contextoContactoPrueba{}, &autorizadorBorradorPrueba{t: t, instante: ahora}, repo, plazo, func() time.Time { return ahora })
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.PublicarOferta(t.Context(), solicitudPublicarOfertaPrueba(t, ahora))
	if err != nil || repo.publicado == nil || plazo.bolsa != "bolsa:of" || plazo.legacy != 0 ||
		repo.publicado.Plazo.PoliticaVersion != 3 || repo.publicado.Plazo.MunicipioSede != "18087" ||
		len(repo.publicado.Plazo.Calendarios) != 1 {
		t.Fatalf("err=%v bolsa=%q legacy=%d comando=%+v", err, plazo.bolsa, plazo.legacy, repo.publicado)
	}
}
