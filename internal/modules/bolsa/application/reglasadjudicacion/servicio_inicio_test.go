package reglasadjudicacion

import (
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

func TestPoliticaPublicacionGobiernaInicioYPlazoSinCorreo(t *testing.T) {
	politica := politicaPrueba()
	politica.Plazo.Inicio = domain.InicioPlazoOfertaPublicacion
	politica.Plazo.Unidad, politica.Plazo.Cantidad, politica.Plazo.Computo = "horas_naturales", 48, "continuo_utc"
	servicio, err := NuevoServicio(repoPrueba{version: ports.VersionPoliticaOfertas{
		BolsaRef: "bolsa:prueba", Version: 3, HuellaSHA256: strings.Repeat("a", 64),
		Ejemplo: true, Configurada: true, Politica: &politica,
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	inicio, err := servicio.InicioPlazoDisposicion(t.Context(), "bolsa:prueba")
	if err != nil || inicio != domain.InicioPlazoOfertaPublicacion {
		t.Fatalf("inicio=%q err=%v", inicio, err)
	}
	publicada := time.Date(2026, 10, 15, 10, 0, 0, 123456000, time.UTC)
	plazo, vence, err := servicio.PlazoDisposicionBolsa(t.Context(), "bolsa:prueba", publicada)
	if err != nil || plazo.Inicio != inicio || plazo.Notificacion != nil ||
		!vence.Equal(publicada.Add(48*time.Hour)) || plazo.AperturaEn != "2026-10-15T10:00:00.123456Z" {
		t.Fatalf("plazo=%+v vence=%v err=%v", plazo, vence, err)
	}
}

func TestPoliticaHistoricaSinInicioNoSeReinterpreta(t *testing.T) {
	politica := politicaPrueba()
	politica.Plazo.Inicio = ""
	servicio, _ := NuevoServicio(repoPrueba{version: ports.VersionPoliticaOfertas{
		BolsaRef: "bolsa:prueba", Version: 1, HuellaSHA256: strings.Repeat("a", 64),
		Ejemplo: true, Configurada: true, Politica: &politica,
	}}, nil)
	if _, err := servicio.InicioPlazoDisposicion(t.Context(), "bolsa:prueba"); !errors.Is(err, ports.ErrPlazoOfertaNoConfigurado) {
		t.Fatalf("inicio histórico: %v", err)
	}
}
