package application

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

// repositorioPoliticaTransicionesPrueba imita una base con la migración
// 000032: publica y devuelve la política vigente.
type repositorioPoliticaTransicionesPrueba struct {
	*repositorioOperacionPrueba
	vigente    ports.PoliticaTransicionesVigente
	err        error
	publicadas int
}

func (r *repositorioPoliticaTransicionesPrueba) PoliticaTransicionesSituacion(context.Context) (ports.PoliticaTransicionesVigente, error) {
	return r.vigente, r.err
}

func (r *repositorioPoliticaTransicionesPrueba) PublicarPoliticaTransicionesSituacion(_ context.Context, p ports.PublicacionPoliticaTransicionesSituacion) (ports.PoliticaTransicionesVigente, error) {
	if r.err != nil {
		return ports.PoliticaTransicionesVigente{}, r.err
	}
	r.publicadas++
	r.vigente = ports.PoliticaTransicionesVigente{Version: r.vigente.Version + 1, CatalogoRef: p.CatalogoRef, Politica: p.Politica}
	return r.vigente, nil
}

func politicaReglamentoPrueba(t *testing.T) domain.PoliticaTransicionesSituacion {
	t.Helper()
	tabla := map[string][]string{}
	for _, origen := range domain.SituacionesParticipacion() {
		tabla[origen] = domain.DestinosSituacionParticipacion(origen)
	}
	tabla[domain.SituacionRenuncia] = []string{domain.SituacionNoDisponible, domain.SituacionExcluido}
	politica, err := domain.NuevaPoliticaTransicionesSituacion(tabla)
	if err != nil {
		t.Fatal(err)
	}
	return politica
}

func servicioConPoliticaPrueba(t *testing.T) (*ServicioSituacionParticipacion, *repositorioPoliticaTransicionesPrueba, time.Time) {
	t.Helper()
	ahora := time.Date(2026, 9, 25, 9, 0, 0, 0, time.UTC)
	repo := &repositorioPoliticaTransicionesPrueba{repositorioOperacionPrueba: &repositorioOperacionPrueba{repositorioSituacionPrueba: &repositorioSituacionPrueba{pertenece: true, vigente: ports.SituacionParticipacion{ParticipacionRef: "participacion:b2", Situacion: domain.SituacionRenuncia, Desde: ahora}}}}
	servicio, err := NuevoServicioSituacionParticipacion(contextoSituacionPrueba{}, &autorizadorBorradorPrueba{t: t, instante: ahora}, repo, func() time.Time { return ahora })
	if err != nil {
		t.Fatal(err)
	}
	return servicio, repo, ahora
}

func TestPoliticaPublicadaSustituyeALaTablaCompilada(t *testing.T) {
	servicio, repo, ahora := servicioConPoliticaPrueba(t)
	vigente, err := servicio.PublicarPoliticaTransiciones(context.Background(), ports.PublicacionPoliticaTransicionesSituacion{CatalogoRef: "catalogo:1:b28.transiciones", CatalogoSHA256: "ab", Politica: politicaReglamentoPrueba(t)})
	if err != nil || vigente.Version != 1 || repo.publicadas != 1 {
		t.Fatalf("publicación: %+v %v", vigente, err)
	}
	s := solicitudSituacionPrueba(t, ahora)
	s.Destino = domain.SituacionDisponible
	if _, err := servicio.Cambiar(context.Background(), s); !errors.Is(err, domain.ErrCambioSituacionParticipacionInvalido) || repo.llamadas != 0 {
		t.Fatalf("la política publicada cierra renuncia→disponible: err=%v", err)
	}
	s.Destino = domain.SituacionNoDisponible
	s.ClaveIdempotencia = "b2-renuncia-justificada"
	if _, err := servicio.Cambiar(context.Background(), s); err != nil || repo.llamadas != 1 {
		t.Fatalf("la política publicada abre renuncia→no_disponible: err=%v", err)
	}
	transiciones, err := servicio.TransicionesAdmitidas(context.Background())
	if err != nil || !slices.Equal(transiciones[domain.SituacionRenuncia], []string{domain.SituacionNoDisponible, domain.SituacionExcluido}) {
		t.Fatalf("lectura: %v %v", transiciones, err)
	}
}

func TestOperacionB8PausaUnaRenunciaJustificadaConLaPoliticaPublicada(t *testing.T) {
	servicio, repo, ahora := servicioConPoliticaPrueba(t)
	repo.vigente = ports.PoliticaTransicionesVigente{Version: 2, Politica: politicaReglamentoPrueba(t)}
	q := ports.SolicitudOperacionSituacion{SolicitudCambiarSituacionParticipacion: solicitudSituacionPrueba(t, ahora), Operacion: domain.OperacionPausar, Justificante: domain.JustificanteOperacionSituacion{Tipo: domain.JustificanteInformeMedico, Referencia: "justificante:01", SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}, Validador: "persona:validadora"}
	q.Destino = domain.SituacionNoDisponible
	if _, err := servicio.Operar(context.Background(), q); err != nil || repo.escrituras != 1 {
		t.Fatalf("pausar una renuncia justificada: err=%v escrituras=%d", err, repo.escrituras)
	}
}

func TestPoliticaIlegibleNoEsPermiso(t *testing.T) {
	servicio, repo, ahora := servicioConPoliticaPrueba(t)
	repo.err = errors.New("caida")
	s := solicitudSituacionPrueba(t, ahora)
	s.Destino = domain.SituacionExcluido
	if _, err := servicio.Cambiar(context.Background(), s); !errors.Is(err, ErrCambioSituacionParticipacionNoDisponible) || repo.llamadas != 0 {
		t.Fatalf("política ilegible: err=%v llamadas=%d", err, repo.llamadas)
	}
	if _, err := servicio.TransicionesAdmitidas(context.Background()); !errors.Is(err, ErrCambioSituacionParticipacionNoDisponible) {
		t.Fatalf("lectura con política ilegible: %v", err)
	}
}

func TestPublicarSinRepositorioQueLaAdmitaEsNoInstalada(t *testing.T) {
	servicio, _, _ := servicioDesdeRenunciaPrueba(t, nil)
	if _, err := servicio.PublicarPoliticaTransiciones(context.Background(), ports.PublicacionPoliticaTransicionesSituacion{}); !errors.Is(err, ports.ErrPoliticaTransicionesNoInstalada) {
		t.Fatalf("sin 000032: %v", err)
	}
}
