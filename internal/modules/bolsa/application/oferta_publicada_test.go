package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/pruebas"
)

type plazoOfertaPrueba struct {
	err     error
	llamado int
}

func (p *plazoOfertaPrueba) PlazoDisposicion(_ context.Context, desde time.Time) (puertosbolsa.PlazoOferta, time.Time, error) {
	p.llamado++
	if p.err != nil {
		return puertosbolsa.PlazoOferta{}, time.Time{}, p.err
	}
	return puertosbolsa.PlazoOferta{ReglaRef: "vec.bolsa.reglas:1:b10.plazo_publicacion", Unidad: "dias_habiles", Cantidad: 2, UltimoDia: "2026-09-29"}, desde.Add(72 * time.Hour), nil
}

type repositorioOfertasPrueba struct {
	publicado *puertosbolsa.ComandoPublicarOferta
	resuelto  *puertosbolsa.ComandoResolverOferta
	devolver  *puertosbolsa.OfertaPublicada
	err       error
	listadas  int
}

func (r *repositorioOfertasPrueba) Publicar(_ context.Context, c puertosbolsa.ComandoPublicarOferta) (puertosbolsa.OfertaPublicada, error) {
	r.publicado = &c
	if r.err != nil {
		return puertosbolsa.OfertaPublicada{}, r.err
	}
	if r.devolver != nil {
		return *r.devolver, nil
	}
	return puertosbolsa.OfertaPublicada{OfertaRef: c.OfertaRef, BolsaRef: c.BolsaRef, Datos: c.Datos, Plazo: c.Plazo, PublicadaEn: c.PublicadaEn, VenceAntesDe: c.VenceAntesDe, Estado: dominiobolsa.EstadoOfertaAbierta}, nil
}

func (r *repositorioOfertasPrueba) Resolver(_ context.Context, c puertosbolsa.ComandoResolverOferta) (puertosbolsa.OfertaPublicada, error) {
	r.resuelto = &c
	if r.err != nil {
		return puertosbolsa.OfertaPublicada{}, r.err
	}
	return puertosbolsa.OfertaPublicada{OfertaRef: c.OfertaRef, BolsaRef: c.BolsaRef, Estado: dominiobolsa.EstadoOfertaAdjudicada}, nil
}

func (r *repositorioOfertasPrueba) Listar(context.Context, string, time.Time, int) ([]puertosbolsa.OfertaPublicada, error) {
	r.listadas++
	return []puertosbolsa.OfertaPublicada{}, r.err
}

func datosOfertaPrueba() dominiobolsa.DatosOferta {
	return dominiobolsa.DatosOferta{Categoria: "Auxiliar administrativo", Centro: "Residencia Sierra", FechaInicio: "2026-10-01", FechaFin: "2026-12-31", Descripcion: "Sustitución por baja"}
}

func solicitudPublicarOfertaPrueba(t *testing.T, ahora time.Time) puertosbolsa.SolicitudPublicarOferta {
	t.Helper()
	resultado, vinculo, err := pruebas.NuevoContextoRegistradoYVinculoV2(ahora, "per_0123456789abcdefghijkl", "prf_0123456789abcdefghijkl", dominiovec.AuthMethodCertificate, dominiovec.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	return puertosbolsa.SolicitudPublicarOferta{Vinculo: vinculo, ResultadoContexto: resultado, BolsaRef: "bolsa:of", Datos: datosOfertaPrueba(), ClaveIdempotencia: "oferta-clave-0001", Correlacion: correlacionBorradorPrueba(t), MotivoAutorizacion: motivoBorradorPrueba()}
}

func TestPublicarOfertaFijaPlazoDeLaReglaYConsumeLaAutorizacionDeEmision(t *testing.T) {
	ahora := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	repo, plazos := &repositorioOfertasPrueba{}, &plazoOfertaPrueba{}
	autorizador := &autorizadorBorradorPrueba{t: t, instante: ahora}
	servicio, err := NuevoServicioOfertasPublicadas(contextoContactoPrueba{}, autorizador, repo, plazos, func() time.Time { return ahora })
	if err != nil {
		t.Fatal(err)
	}
	oferta, err := servicio.PublicarOferta(context.Background(), solicitudPublicarOfertaPrueba(t, ahora))
	if err != nil || oferta.Estado != dominiobolsa.EstadoOfertaAbierta || repo.publicado == nil || autorizador.llamadas != 1 {
		t.Fatalf("oferta=%+v err=%v", oferta, err)
	}
	c := repo.publicado
	if !strings.HasPrefix(c.OfertaRef, "oferta:") || strings.TrimPrefix(c.OfertaRef, "oferta:") != strings.TrimPrefix(c.ReciboRef, "recibo:oferta:") ||
		!c.VenceAntesDe.Equal(ahora.Add(72*time.Hour)) || c.Plazo.ReglaRef == "" || c.Material.ValidarEstructura() != nil || c.ActorRef != "per_0123456789abcdefghijkl" {
		t.Fatalf("comando incorrecto: %+v", c)
	}
}

func TestPublicarOfertaSinReglaNoInventaPlazo(t *testing.T) {
	ahora := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	repo := &repositorioOfertasPrueba{}
	servicio, _ := NuevoServicioOfertasPublicadas(contextoContactoPrueba{}, &autorizadorBorradorPrueba{t: t, instante: ahora}, repo, &plazoOfertaPrueba{err: puertosbolsa.ErrPlazoOfertaNoConfigurado}, func() time.Time { return ahora })
	if _, err := servicio.PublicarOferta(context.Background(), solicitudPublicarOfertaPrueba(t, ahora)); !errors.Is(err, puertosbolsa.ErrPlazoOfertaNoConfigurado) || repo.publicado != nil {
		t.Fatalf("err=%v publicado=%v", err, repo.publicado)
	}
}

func TestPublicarOfertaRechazaDatosInvalidosAntesDeAutorizar(t *testing.T) {
	ahora := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	autorizador := &autorizadorBorradorPrueba{t: t, instante: ahora}
	plazos := &plazoOfertaPrueba{}
	servicio, _ := NuevoServicioOfertasPublicadas(contextoContactoPrueba{}, autorizador, &repositorioOfertasPrueba{}, plazos, func() time.Time { return ahora })
	for nombre, mutar := range map[string]func(*puertosbolsa.SolicitudPublicarOferta){
		"fin anterior":  func(s *puertosbolsa.SolicitudPublicarOferta) { s.Datos.FechaFin = "2026-09-01" },
		"centro vacío":  func(s *puertosbolsa.SolicitudPublicarOferta) { s.Datos.Centro = "" },
		"clave corta":   func(s *puertosbolsa.SolicitudPublicarOferta) { s.ClaveIdempotencia = "corta" },
		"bolsa espacio": func(s *puertosbolsa.SolicitudPublicarOferta) { s.BolsaRef = " bolsa" },
	} {
		s := solicitudPublicarOfertaPrueba(t, ahora)
		mutar(&s)
		if _, err := servicio.PublicarOferta(context.Background(), s); !errors.Is(err, puertosbolsa.ErrOfertaInvalida) {
			t.Fatalf("%s: err=%v", nombre, err)
		}
	}
	if autorizador.llamadas != 0 || plazos.llamado != 0 {
		t.Fatalf("autorizó o calculó plazo con entrada inválida: %d %d", autorizador.llamadas, plazos.llamado)
	}
}

func TestPublicarOfertaDetectaReplayConOtroContenido(t *testing.T) {
	ahora := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	otra := datosOfertaPrueba()
	otra.Centro = "Otro centro"
	repo := &repositorioOfertasPrueba{devolver: &puertosbolsa.OfertaPublicada{OfertaRef: "oferta:x", BolsaRef: "bolsa:of", Datos: otra, Estado: "abierta", Reutilizada: true}}
	servicio, _ := NuevoServicioOfertasPublicadas(contextoContactoPrueba{}, &autorizadorBorradorPrueba{t: t, instante: ahora}, repo, &plazoOfertaPrueba{}, func() time.Time { return ahora })
	if _, err := servicio.PublicarOferta(context.Background(), solicitudPublicarOfertaPrueba(t, ahora)); !errors.Is(err, puertosbolsa.ErrOfertaConflicto) {
		t.Fatalf("err=%v", err)
	}
}

func TestPublicarOfertaConservaDenegacionDeAmbito(t *testing.T) {
	ahora := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	repo := &repositorioOfertasPrueba{}
	autorizador := &autorizadorBorradorPrueba{t: t, instante: ahora}
	servicio, _ := NuevoServicioOfertasPublicadas(contextoContactoPrueba{err: dominiovec.ErrAutorizacionDenegada}, autorizador, repo, &plazoOfertaPrueba{}, func() time.Time { return ahora })
	if _, err := servicio.PublicarOferta(context.Background(), solicitudPublicarOfertaPrueba(t, ahora)); !errors.Is(err, dominiovec.ErrAutorizacionDenegada) || repo.publicado != nil || autorizador.llamadas != 0 {
		t.Fatalf("err=%v", err)
	}
}

func TestResolverOfertaPasaLaPropuestaYElReciboDeterminista(t *testing.T) {
	ahora := time.Date(2026, 9, 29, 10, 0, 0, 0, time.UTC)
	repo := &repositorioOfertasPrueba{}
	servicio, _ := NuevoServicioOfertasPublicadas(contextoContactoPrueba{}, &autorizadorBorradorPrueba{t: t, instante: ahora}, repo, &plazoOfertaPrueba{}, func() time.Time { return ahora })
	base := solicitudPublicarOfertaPrueba(t, ahora)
	q := puertosbolsa.SolicitudResolverOferta{Vinculo: base.Vinculo, ResultadoContexto: base.ResultadoContexto, BolsaRef: "bolsa:of", OfertaRef: "oferta:" + strings.Repeat("a", 64), ParticipacionRef: "participacion:3", ClaveIdempotencia: "resolucion-0001", Correlacion: base.Correlacion, MotivoAutorizacion: base.MotivoAutorizacion}
	if _, err := servicio.ResolverOferta(context.Background(), q); err != nil || repo.resuelto == nil {
		t.Fatalf("err=%v", err)
	}
	if repo.resuelto.ParticipacionRef != "participacion:3" || !strings.HasPrefix(repo.resuelto.ReciboRef, "recibo:resolucion-oferta:") || repo.resuelto.Material.ValidarEstructura() != nil {
		t.Fatalf("comando=%+v", repo.resuelto)
	}
	q.OfertaRef = "llamamiento:x"
	if _, err := servicio.ResolverOferta(context.Background(), q); !errors.Is(err, puertosbolsa.ErrOfertaInvalida) {
		t.Fatalf("referencia ajena aceptada: %v", err)
	}
}

func TestConsultarOfertasExigeBolsaAdmitidaYLimite(t *testing.T) {
	ahora := time.Date(2026, 9, 29, 10, 0, 0, 0, time.UTC)
	repo := &repositorioOfertasPrueba{}
	servicio, _ := NuevoServicioOfertasPublicadas(contextoContactoPrueba{err: dominiovec.ErrAutorizacionDenegada}, &autorizadorBorradorPrueba{t: t, instante: ahora}, repo, &plazoOfertaPrueba{}, func() time.Time { return ahora })
	actor := dominiovec.ContextoActor{PersonaRef: "per_0123456789abcdefghijkl"}
	if _, err := servicio.ConsultarOfertas(context.Background(), puertosbolsa.SolicitudConsultarOfertas{ContextoActor: actor, BolsaRef: "bolsa:of", Limite: 10}); !errors.Is(err, dominiovec.ErrAutorizacionDenegada) || repo.listadas != 0 {
		t.Fatalf("err=%v listadas=%d", err, repo.listadas)
	}
	if _, err := servicio.ConsultarOfertas(context.Background(), puertosbolsa.SolicitudConsultarOfertas{ContextoActor: actor, BolsaRef: "bolsa:of", Limite: 101}); !errors.Is(err, puertosbolsa.ErrOfertaInvalida) {
		t.Fatalf("límite excesivo aceptado: %v", err)
	}
}
