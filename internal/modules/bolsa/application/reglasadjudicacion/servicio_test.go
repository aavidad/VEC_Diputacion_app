package reglasadjudicacion

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	calendariosdomain "vec-diputacion-granada/internal/modules/calendarios/domain"
	calendariosports "vec-diputacion-granada/internal/modules/calendarios/ports"
)

type repoPrueba struct{ version ports.VersionPoliticaOfertas }

func (r repoPrueba) Vigente(_ context.Context, _ string) (ports.VersionPoliticaOfertas, error) {
	return r.version, nil
}
func (r repoPrueba) Publicar(_ context.Context, _ ports.ComandoPublicarPoliticaOfertas) (ports.VersionPoliticaOfertas, error) {
	return r.version, nil
}

type calendarioPrueba struct {
	solicitud calendariosports.SolicitudCalculoPlazo
	resultado calendariosports.ResultadoCalculoPlazo
}

func (*calendarioPrueba) Centros(context.Context, int, time.Time) ([]calendariosports.CentroConCalendario, error) {
	return nil, nil
}
func (*calendarioPrueba) CalendarioCentro(context.Context, calendariosports.SolicitudCalendarioCentro) (calendariosports.CalendarioCentro, error) {
	return calendariosports.CalendarioCentro{}, nil
}
func (c *calendarioPrueba) CalcularPlazo(_ context.Context, s calendariosports.SolicitudCalculoPlazo) (calendariosports.ResultadoCalculoPlazo, error) {
	c.solicitud = s
	return c.resultado, nil
}

func politicaPrueba() domain.PoliticaOfertas {
	return domain.PoliticaOfertas{
		Plazo:        domain.PlazoPoliticaOfertas{Unidad: "dias_habiles", Cantidad: 2, Computo: "administrativo", MunicipioSede: "18087"},
		Adjudicacion: domain.AdjudicacionPoliticaOfertas{Criterio: "orden_vigente", Elegibilidad: "disposicion_en_plazo"},
		NoCubierta:   domain.NoCubiertaPoliticaOfertas{Accion: "llamamiento_directo", Condicion: "sin_disposiciones_elegibles"},
	}
}

func TestPlazoOfertaCongelaVersionYCalendario(t *testing.T) {
	p := politicaPrueba()
	dia, err := calendariosdomain.ParsearFechaCivil("2026-09-30")
	if err != nil {
		t.Fatal(err)
	}
	publicada := time.Date(2026, 9, 28, 10, 0, 0, 0, time.UTC)
	vence := time.Date(2026, 9, 30, 22, 0, 0, 0, time.UTC)
	cal := &calendarioPrueba{resultado: calendariosports.ResultadoCalculoPlazo{
		ResultadoPlazo:      calendariosdomain.ResultadoPlazo{Vencimiento: dia, VenceAntesDe: vence},
		VersionesUtilizadas: []calendariosdomain.VersionCalendario{{ID: "calendario:granada:2026:1"}},
	}}
	s, err := NuevoServicio(repoPrueba{version: ports.VersionPoliticaOfertas{
		BolsaRef: "bolsa:prueba", Version: 2, HuellaSHA256: strings.Repeat("a", 64), Ejemplo: true, Configurada: true, Politica: &p,
	}}, cal)
	if err != nil {
		t.Fatal(err)
	}
	plazo, final, err := s.PlazoDisposicionBolsa(t.Context(), "bolsa:prueba", publicada)
	if err != nil || !final.Equal(vence) || plazo.PoliticaVersion != 2 || plazo.MunicipioSede != "18087" ||
		plazo.HuellaCatalogo != strings.Repeat("a", 64) || plazo.UltimoDia != "2026-09-30" ||
		len(plazo.Calendarios) != 1 || plazo.Calendarios[0] != "calendario:granada:2026:1" ||
		cal.solicitud.MunicipioSede != "18087" || cal.solicitud.Cantidad != 2 || !cal.solicitud.NotificadoEn.Equal(publicada) {
		t.Fatalf("plazo=%+v final=%v solicitud=%+v err=%v", plazo, final, cal.solicitud, err)
	}
}

func TestSinPoliticaNoPublicaOferta(t *testing.T) {
	s, err := NuevoServicio(repoPrueba{}, &calendarioPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = s.PlazoDisposicionBolsa(t.Context(), "bolsa:prueba", time.Now())
	if !errors.Is(err, ports.ErrPlazoOfertaNoConfigurado) {
		t.Fatalf("sin política: %v", err)
	}
}
