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

type repoConsultaContada struct {
	repoPrueba
	llamadas int
}

func (r *repoConsultaContada) ConsultarAutorizada(_ context.Context, _ ports.ConsultaPoliticaOfertasAutorizada) (ports.VersionPoliticaOfertas, error) {
	r.llamadas++
	return ports.VersionPoliticaOfertas{}, nil
}

func (r repoPrueba) Vigente(_ context.Context, _ string) (ports.VersionPoliticaOfertas, error) {
	return r.version, nil
}
func (r repoPrueba) ConsultarAutorizada(_ context.Context, _ ports.ConsultaPoliticaOfertasAutorizada) (ports.VersionPoliticaOfertas, error) {
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
		Plazo:        domain.PlazoPoliticaOfertas{Inicio: "notificacion", Unidad: "dias_habiles", Cantidad: 2, Computo: "administrativo", MunicipioSede: "18087"},
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
		cal.solicitud.MunicipioSede != "municipio:ine:18087" || cal.solicitud.Cantidad != 2 || !cal.solicitud.NotificadoEn.Equal(publicada) {
		t.Fatalf("plazo=%+v final=%v solicitud=%+v err=%v", plazo, final, cal.solicitud, err)
	}
}

func TestPlazoHorasNaturalesCruzaCambiosHorarioMadrid(t *testing.T) {
	madrid, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Fatal(err)
	}
	for _, apertura := range []time.Time{
		time.Date(2026, 3, 28, 12, 17, 13, 123456000, time.UTC),
		time.Date(2026, 10, 24, 12, 17, 13, 123456000, time.UTC),
	} {
		p := politicaPrueba()
		p.Plazo = domain.PlazoPoliticaOfertas{Inicio: "notificacion", Unidad: "horas_naturales", Cantidad: 48, Computo: "continuo_utc", MunicipioSede: "18087"}
		s, err := NuevoServicio(repoPrueba{version: ports.VersionPoliticaOfertas{
			BolsaRef: "bolsa:prueba", Version: 3, HuellaSHA256: strings.Repeat("a", 64), Ejemplo: true, Configurada: true, Politica: &p,
		}}, nil)
		if err != nil {
			t.Fatal(err)
		}
		plazo, vence, err := s.PlazoDisposicionBolsa(t.Context(), "bolsa:prueba", apertura)
		if err != nil || vence.Sub(apertura) != 48*time.Hour || plazo.AperturaEn != apertura.Format("2006-01-02T15:04:05.000000Z") ||
			plazo.VenceEn != vence.Format("2006-01-02T15:04:05.000000Z") ||
			plazo.UltimoDia != vence.Add(-time.Nanosecond).In(madrid).Format(time.DateOnly) ||
			plazo.Unidad != "horas_naturales" || len(plazo.Calendarios) != 1 {
			t.Fatalf("apertura=%s plazo=%+v vence=%s err=%v", apertura, plazo, vence, err)
		}
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

func TestCalendariosAusentePermiteConsultarPoliticaPeroNoPublicarOferta(t *testing.T) {
	p := politicaPrueba()
	s, err := NuevoServicio(repoPrueba{version: ports.VersionPoliticaOfertas{
		BolsaRef: "bolsa:prueba", Version: 1, HuellaSHA256: strings.Repeat("a", 64),
		Ejemplo: true, Configurada: true, Politica: &p,
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if v, err := s.Vigente(t.Context(), "bolsa:prueba"); err != nil || v.Version != 1 {
		t.Fatalf("consulta %v %+v", err, v)
	}
	if _, _, err := s.PlazoDisposicionBolsa(t.Context(), "bolsa:prueba", time.Now()); !errors.Is(err, ports.ErrOfertaNoDisponible) {
		t.Fatalf("oferta sin calendario: %v", err)
	}
}

func TestConsultaRRHHExigeMaterialAtestado(t *testing.T) {
	repo := &repoConsultaContada{}
	s, err := NuevoServicio(repo, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.ConsultarAutorizada(t.Context(), ports.ConsultaPoliticaOfertasAutorizada{BolsaRef: "bolsa:prueba"}); !errors.Is(err, ports.ErrPoliticaOfertasNoDisponible) || repo.llamadas != 0 {
		t.Fatalf("GET sin material V3: %v", err)
	}
}

func TestPoliticaHistoricaSeConsultaSinAbrirOtraOfertaDesdeNotificacion(t *testing.T) {
	p := politicaPrueba()
	p.Plazo.Inicio = ""
	s, err := NuevoServicio(repoPrueba{version: ports.VersionPoliticaOfertas{BolsaRef: "bolsa:prueba", Version: 1, HuellaSHA256: strings.Repeat("a", 64), Ejemplo: true, Configurada: true, Politica: &p}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	v, err := s.Vigente(t.Context(), "bolsa:prueba")
	if err != nil || v.Politica.Plazo.Inicio != "" {
		t.Fatalf("política histórica alterada: %+v %v", v, err)
	}
	if _, _, err = s.PlazoDisposicionBolsa(t.Context(), "bolsa:prueba", time.Date(2026, 10, 2, 9, 0, 0, 0, time.UTC)); !errors.Is(err, ports.ErrPlazoOfertaNoConfigurado) {
		t.Fatalf("oferta nueva con política histórica: %v", err)
	}
}
