package application

import (
	"errors"
	"strings"
	"testing"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

func TestPublicacionCalculaDesdeNotificacionYConservaFechaPublicacion(t *testing.T) {
	ahora := time.Date(2026, 10, 2, 11, 0, 0, 0, time.UTC)
	q := solicitudPublicarOfertaPrueba(t, ahora)
	q.Notificacion.NotificadaEn = ahora.Add(-time.Hour)
	repo := &repositorioOfertasPrueba{}
	servicio, _ := NuevoServicioOfertasPublicadas(contextoContactoPrueba{}, &autorizadorBorradorPrueba{t: t, instante: ahora}, repo, &plazoOfertaPrueba{}, func() time.Time { return ahora })
	o, err := servicio.PublicarOferta(t.Context(), q)
	if err != nil || !o.PublicadaEn.Equal(ahora) || !o.VenceAntesDe.Equal(q.Notificacion.NotificadaEn.Add(72*time.Hour)) || o.Plazo.Notificacion == nil || *o.Plazo.Notificacion != q.Notificacion {
		t.Fatalf("oferta=%+v err=%v", o, err)
	}
}

func TestPublicacionNoSustituyeNotificacionAusenteOFuturaPorElReloj(t *testing.T) {
	ahora := time.Date(2026, 10, 2, 11, 0, 0, 0, time.UTC)
	for _, notificacion := range []dominiobolsa.NotificacionOferta{{}, {NotificadaEn: ahora.Add(time.Hour), ReferenciaCorreo: "correo:extracto:1", HuellaCorreoSHA256: strings.Repeat("a", 64), Fuente: "correo_externo_declarado_rrhh"}} {
		q := solicitudPublicarOfertaPrueba(t, ahora)
		q.Notificacion = notificacion
		repo, plazo := &repositorioOfertasPrueba{}, &plazoOfertaPrueba{}
		auth := &autorizadorBorradorPrueba{t: t, instante: ahora}
		servicio, _ := NuevoServicioOfertasPublicadas(contextoContactoPrueba{}, auth, repo, plazo, func() time.Time { return ahora })
		if _, err := servicio.PublicarOferta(t.Context(), q); !errors.Is(err, puertosbolsa.ErrOfertaInvalida) || repo.publicado != nil || plazo.llamado != 0 || auth.llamadas != 0 {
			t.Fatalf("err=%v publicación=%+v", err, repo.publicado)
		}
	}
}

func TestMaterialAutorizadoLigaLaEvidenciaDeNotificacion(t *testing.T) {
	ahora := time.Date(2026, 10, 2, 11, 0, 0, 0, time.UTC)
	q := solicitudPublicarOfertaPrueba(t, ahora)
	plazo := puertosbolsa.PlazoOferta{Notificacion: &q.Notificacion}
	original := huellaMaterialPlazoOfertaConPlazas(q.BolsaRef, ahora, ahora.Add(48*time.Hour), plazo, 1)
	q.Notificacion.HuellaCorreoSHA256 = strings.Repeat("b", 64)
	if original == huellaMaterialPlazoOfertaConPlazas(q.BolsaRef, ahora, ahora.Add(48*time.Hour), plazo, 1) {
		t.Fatal("el material no liga el correo declarado")
	}
}
