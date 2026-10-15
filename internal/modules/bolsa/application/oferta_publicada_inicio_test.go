package application

import (
	"context"
	"errors"
	"testing"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

type calculadorInicioOfertaPrueba struct {
	inicio      string
	err         error
	consultas   int
	calculados  int
	comienzo    time.Time
	calendarios []string
}

func (c *calculadorInicioOfertaPrueba) InicioPlazoDisposicion(context.Context, string) (string, error) {
	c.consultas++
	return c.inicio, c.err
}

func (c *calculadorInicioOfertaPrueba) PlazoDisposicion(_ context.Context, comienzo time.Time) (puertosbolsa.PlazoOferta, time.Time, error) {
	c.calculados++
	c.comienzo = comienzo
	calendarios := c.calendarios
	if calendarios == nil {
		calendarios = []string{"calendario:utc-continuo:v1"}
	}
	return puertosbolsa.PlazoOferta{
		Inicio: c.inicio, ReglaRef: "politica-ofertas:prueba:2", HuellaCatalogo: "huella:prueba",
		Unidad: "horas_naturales", Cantidad: 48, Computo: "continuo_utc",
		MunicipioSede: "18087", UltimoDia: "2026-10-17", PoliticaVersion: 2,
		Calendarios: calendarios,
		AperturaEn:  comienzo.Format(formatoInstanteMaterialOferta),
		VenceEn:     comienzo.Add(48 * time.Hour).Format(formatoInstanteMaterialOferta),
	}, comienzo.Add(48 * time.Hour), nil
}

func TestOfertaPublicacionDeniegaCalendarioConSeparadorCanonico(t *testing.T) {
	ahora := time.Date(2026, 10, 15, 10, 0, 0, 0, time.UTC)
	q := solicitudPublicarOfertaPrueba(t, ahora)
	q.Notificacion = dominiobolsa.NotificacionOferta{}
	calculador := &calculadorInicioOfertaPrueba{
		inicio:      dominiobolsa.InicioPlazoOfertaPublicacion,
		calendarios: []string{"calendario:a\x1ecalendario:b"},
	}
	repo := &recuperadorOfertaPrueba{}
	auth := &autorizadorBorradorPrueba{t: t, instante: ahora}
	servicio, _ := NuevoServicioOfertasPublicadas(contextoContactoPrueba{}, auth, repo, calculador, func() time.Time { return ahora })
	if _, err := servicio.PublicarOferta(t.Context(), q); !errors.Is(err, puertosbolsa.ErrOfertaNoDisponible) ||
		repo.publicado != nil || calculador.calculados != 1 || auth.llamadas != 1 {
		t.Fatalf("calendario ambiguo firmado/publicado: %v", err)
	}
}

type recuperadorOfertaPrueba struct {
	repositorioOfertasPrueba
	recuperada puertosbolsa.OfertaPublicada
	encontrada bool
	consultas  int
	comando    puertosbolsa.ComandoRecuperarOferta
}

func (r *recuperadorOfertaPrueba) RecuperarOferta(_ context.Context, c puertosbolsa.ComandoRecuperarOferta) (puertosbolsa.OfertaPublicada, bool, error) {
	r.consultas++
	r.comando = c
	return r.recuperada, r.encontrada, nil
}

func TestOfertaPublicacionEmpiezaEnRelojSinInventarNotificacion(t *testing.T) {
	ahora := time.Date(2026, 10, 15, 10, 0, 0, 123456000, time.UTC)
	q := solicitudPublicarOfertaPrueba(t, ahora)
	q.Notificacion = dominiobolsa.NotificacionOferta{}
	calculador := &calculadorInicioOfertaPrueba{inicio: dominiobolsa.InicioPlazoOfertaPublicacion}
	repo := &recuperadorOfertaPrueba{}
	auth := &autorizadorBorradorPrueba{t: t, instante: ahora}
	servicio, err := NuevoServicioOfertasPublicadas(contextoContactoPrueba{}, auth, repo, calculador, func() time.Time { return ahora })
	if err != nil {
		t.Fatal(err)
	}
	oferta, err := servicio.PublicarOferta(t.Context(), q)
	if err != nil || repo.consultas != 1 || calculador.consultas != 1 || calculador.calculados != 1 ||
		!calculador.comienzo.Equal(ahora) || repo.publicado == nil ||
		repo.publicado.Plazo.Inicio != dominiobolsa.InicioPlazoOfertaPublicacion ||
		repo.publicado.Plazo.Notificacion != nil || !oferta.PublicadaEn.Equal(ahora) ||
		!oferta.VenceAntesDe.Equal(ahora.Add(48*time.Hour)) || auth.llamadas != 2 {
		t.Fatalf("publicación=%+v comando=%+v err=%v", oferta, repo.publicado, err)
	}
	if repo.comando.UnidadRef == "" || repo.comando.AmbitoRef == "" ||
		repo.comando.Material.ValidarEstructura() != nil {
		t.Fatal("recuperación sin ámbitos o material V3")
	}
}

func TestOfertaPublicacionRecuperaReciboAntesDePoliticaNueva(t *testing.T) {
	ahora := time.Date(2026, 10, 15, 10, 0, 0, 0, time.UTC)
	q := solicitudPublicarOfertaPrueba(t, ahora)
	q.Notificacion = dominiobolsa.NotificacionOferta{}
	original := puertosbolsa.OfertaPublicada{
		OfertaRef: "oferta:original", ReciboRef: "recibo:oferta:original", BolsaRef: q.BolsaRef,
		Datos: q.Datos, NumeroPlazas: q.NumeroPlazas,
		Plazo:       puertosbolsa.PlazoOferta{Inicio: dominiobolsa.InicioPlazoOfertaPublicacion},
		PublicadaEn: ahora.Add(-time.Hour), VenceAntesDe: ahora.Add(47 * time.Hour),
	}
	repo := &recuperadorOfertaPrueba{recuperada: original, encontrada: true}
	calculador := &calculadorInicioOfertaPrueba{err: puertosbolsa.ErrPlazoOfertaNoConfigurado}
	auth := &autorizadorBorradorPrueba{t: t, instante: ahora}
	servicio, _ := NuevoServicioOfertasPublicadas(contextoContactoPrueba{}, auth, repo, calculador, func() time.Time { return ahora })
	devuelta, err := servicio.PublicarOferta(t.Context(), q)
	if err != nil || devuelta.ReciboRef != original.ReciboRef || repo.publicado != nil ||
		calculador.consultas != 0 || calculador.calculados != 0 || auth.llamadas != 1 {
		t.Fatalf("recibo=%+v err=%v", devuelta, err)
	}
	// SQL coteja también el origen. El servicio comprueba que la proyección
	// recuperada siga representando la intención estable de RRHH.
	repo.recuperada.Datos.Centro = "Centro distinto"
	if _, err := servicio.PublicarOferta(t.Context(), q); !errors.Is(err, puertosbolsa.ErrOfertaConflicto) || repo.publicado != nil {
		t.Fatalf("recuperación divergente: %v", err)
	}
}

func TestOfertaNotificadaRecuperaOriginalTrasCambioAPublicacion(t *testing.T) {
	ahora := time.Date(2026, 10, 15, 10, 0, 0, 0, time.UTC)
	q := solicitudPublicarOfertaPrueba(t, ahora)
	q.Notificacion.NotificadaEn = ahora.Add(-time.Hour)
	notificacion := q.Notificacion
	original := puertosbolsa.OfertaPublicada{
		OfertaRef: "oferta:notificada", ReciboRef: "recibo:oferta:notificada", BolsaRef: q.BolsaRef,
		Datos: q.Datos, NumeroPlazas: q.NumeroPlazas,
		Plazo:       puertosbolsa.PlazoOferta{Inicio: dominiobolsa.InicioPlazoOfertaNotificacion, Notificacion: &notificacion},
		PublicadaEn: ahora.Add(-30 * time.Minute), VenceAntesDe: ahora.Add(47 * time.Hour),
	}
	repo := &recuperadorOfertaPrueba{recuperada: original, encontrada: true}
	calculador := &calculadorInicioOfertaPrueba{inicio: dominiobolsa.InicioPlazoOfertaPublicacion}
	auth := &autorizadorBorradorPrueba{t: t, instante: ahora}
	servicio, _ := NuevoServicioOfertasPublicadas(contextoContactoPrueba{}, auth, repo, calculador, func() time.Time { return ahora })
	devuelta, err := servicio.PublicarOferta(t.Context(), q)
	if err != nil || devuelta.ReciboRef != original.ReciboRef || repo.publicado != nil ||
		calculador.consultas != 0 || repo.comando.Notificacion != q.Notificacion || auth.llamadas != 1 {
		t.Fatalf("recibo notificado=%+v err=%v", devuelta, err)
	}
	otra := q
	otra.Notificacion.HuellaCorreoSHA256 = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	if _, err := servicio.PublicarOferta(t.Context(), otra); !errors.Is(err, puertosbolsa.ErrOfertaConflicto) ||
		repo.publicado != nil {
		t.Fatalf("correo distinto recuperó recibo ajeno: %v", err)
	}
}

func TestOfertaNotificacionSigueExigiendoDeclaracionCuandoPoliticaLoPide(t *testing.T) {
	ahora := time.Date(2026, 10, 15, 10, 0, 0, 0, time.UTC)
	q := solicitudPublicarOfertaPrueba(t, ahora)
	q.Notificacion = dominiobolsa.NotificacionOferta{}
	calculador := &calculadorInicioOfertaPrueba{inicio: dominiobolsa.InicioPlazoOfertaNotificacion}
	repo := &repositorioOfertasPrueba{}
	servicio, _ := NuevoServicioOfertasPublicadas(contextoContactoPrueba{}, &autorizadorBorradorPrueba{t: t, instante: ahora}, repo, calculador, func() time.Time { return ahora })
	if _, err := servicio.PublicarOferta(t.Context(), q); !errors.Is(err, puertosbolsa.ErrOfertaInvalida) ||
		repo.publicado != nil || calculador.calculados != 0 {
		t.Fatalf("notificación ausente: %v", err)
	}
}
