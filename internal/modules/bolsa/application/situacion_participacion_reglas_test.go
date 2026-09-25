package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

// reglasTransicionesPrueba imita el catálogo: solo restringe los orígenes
// que declara.
type reglasTransicionesPrueba struct {
	destinos map[string][]string
	err      error
}

func (r reglasTransicionesPrueba) DestinosSituacion(_ context.Context, origen string) ([]string, bool, error) {
	if r.err != nil {
		return nil, false, r.err
	}
	destinos, ok := r.destinos[origen]
	return destinos, ok, nil
}

func servicioDesdeRenunciaPrueba(t *testing.T, reglas ports.ReglasTransicionesSituacion) (*ServicioSituacionParticipacion, *repositorioOperacionPrueba, time.Time) {
	t.Helper()
	ahora := time.Date(2026, 9, 25, 9, 0, 0, 0, time.UTC)
	repo := &repositorioOperacionPrueba{repositorioSituacionPrueba: &repositorioSituacionPrueba{pertenece: true, vigente: ports.SituacionParticipacion{ParticipacionRef: "participacion:b2", Situacion: domain.SituacionRenuncia, Desde: ahora}}}
	servicio, err := NuevoServicioSituacionParticipacion(contextoSituacionPrueba{}, &autorizadorBorradorPrueba{t: t, instante: ahora}, repo, func() time.Time { return ahora })
	if err != nil {
		t.Fatal(err)
	}
	if reglas != nil {
		servicio.EstablecerReglasTransiciones(reglas)
	}
	return servicio, repo, ahora
}

func TestServicioSituacionSinCatalogoConservaLaTablaCompilada(t *testing.T) {
	servicio, repo, ahora := servicioDesdeRenunciaPrueba(t, nil)
	s := solicitudSituacionPrueba(t, ahora)
	s.Destino = domain.SituacionDisponible
	if _, err := servicio.Cambiar(context.Background(), s); err != nil || repo.llamadas != 1 {
		t.Fatalf("sin catálogo la renuncia vuelve a disponible como hasta ahora: err=%v llamadas=%d", err, repo.llamadas)
	}
}

func TestServicioSituacionAplicaLasTransicionesDelCatalogo(t *testing.T) {
	reglas := reglasTransicionesPrueba{destinos: map[string][]string{domain.SituacionRenuncia: {domain.SituacionExcluido}}}
	servicio, repo, ahora := servicioDesdeRenunciaPrueba(t, reglas)
	s := solicitudSituacionPrueba(t, ahora)
	s.Destino = domain.SituacionDisponible
	if _, err := servicio.Cambiar(context.Background(), s); !errors.Is(err, domain.ErrCambioSituacionParticipacionInvalido) || repo.llamadas != 0 {
		t.Fatalf("el catálogo cierra renuncia→disponible: err=%v llamadas=%d", err, repo.llamadas)
	}
	s.Destino = domain.SituacionExcluido
	s.ClaveIdempotencia = "b2-cambio-excluir"
	if _, err := servicio.Cambiar(context.Background(), s); err != nil || repo.llamadas != 1 {
		t.Fatalf("renuncia→excluido sigue admitida: err=%v llamadas=%d", err, repo.llamadas)
	}
}

func TestServicioSituacionElCatalogoNoAbreTransicionesNuevas(t *testing.T) {
	reglas := reglasTransicionesPrueba{destinos: map[string][]string{domain.SituacionRenuncia: {domain.SituacionTrabajando, domain.SituacionExcluido}}}
	servicio, repo, ahora := servicioDesdeRenunciaPrueba(t, reglas)
	s := solicitudSituacionPrueba(t, ahora)
	s.Destino = domain.SituacionTrabajando
	if _, err := servicio.Cambiar(context.Background(), s); !errors.Is(err, domain.ErrCambioSituacionParticipacionInvalido) || repo.llamadas != 0 {
		t.Fatalf("una transición fuera de la tabla compilada no se abre: err=%v", err)
	}
}

func TestServicioSituacionCatalogoIlegibleNoEsPermiso(t *testing.T) {
	servicio, repo, ahora := servicioDesdeRenunciaPrueba(t, reglasTransicionesPrueba{err: ports.ErrReglasSituacionNoDisponibles})
	s := solicitudSituacionPrueba(t, ahora)
	s.Destino = domain.SituacionExcluido
	if _, err := servicio.Cambiar(context.Background(), s); !errors.Is(err, ErrCambioSituacionParticipacionNoDisponible) || repo.llamadas != 0 {
		t.Fatalf("catálogo ilegible: err=%v llamadas=%d", err, repo.llamadas)
	}
}

func TestOperacionB8ReactivarDesdeRenunciaSigueElCatalogo(t *testing.T) {
	operacion := func(t *testing.T, s *ServicioSituacionParticipacion, ahora time.Time) error {
		q := ports.SolicitudOperacionSituacion{SolicitudCambiarSituacionParticipacion: solicitudSituacionPrueba(t, ahora), Operacion: domain.OperacionReactivar, Justificante: domain.JustificanteOperacionSituacion{Tipo: domain.JustificanteSolicitudCandidato, Referencia: "justificante:01", SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}, Validador: "persona:validadora"}
		q.Destino = domain.SituacionDisponible
		_, err := s.Operar(context.Background(), q)
		return err
	}
	sinCatalogo, repo, ahora := servicioDesdeRenunciaPrueba(t, nil)
	if err := operacion(t, sinCatalogo, ahora); err != nil || repo.escrituras != 1 {
		t.Fatalf("sin catálogo decide la base de datos, como hasta ahora: err=%v", err)
	}
	conCatalogo, repo, ahora := servicioDesdeRenunciaPrueba(t, reglasTransicionesPrueba{destinos: map[string][]string{domain.SituacionRenuncia: {domain.SituacionExcluido}}})
	if err := operacion(t, conCatalogo, ahora); !errors.Is(err, domain.ErrCambioSituacionParticipacionInvalido) || repo.escrituras != 0 {
		t.Fatalf("con catálogo reactivar una renuncia se rechaza sin escribir: err=%v", err)
	}
}
