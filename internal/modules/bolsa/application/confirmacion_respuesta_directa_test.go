package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type repoConfirmacionObsoletaPrueba struct {
	*repositorioSituacionPrueba
	escrituras int
}

func (r *repoConfirmacionObsoletaPrueba) ConfirmarAceptacionDirecta(_ context.Context,
	_ ports.ComandoConfirmarAceptacionDirecta,
) (ports.RegistroConfirmacionAceptacionDirecta, error) {
	r.escrituras++
	return ports.RegistroConfirmacionAceptacionDirecta{}, nil
}

func TestConfirmacionDirectaPropagaInstantaneaObsoletaSinLeerNiEscribirBolsa(t *testing.T) {
	ahora := time.Date(2026, 10, 16, 9, 0, 0, 0, time.UTC)
	repo := &repoConfirmacionObsoletaPrueba{repositorioSituacionPrueba: &repositorioSituacionPrueba{}}
	emisor := &autorizadorBorradorPrueba{t: t, instante: ahora,
		err: vecports.ErrInstantaneaAutorizacionObsoleta}
	s, err := NuevoServicioSituacionParticipacion(contextoSituacionPrueba{}, emisor, repo,
		func() time.Time { return ahora })
	if err != nil {
		t.Fatal(err)
	}
	base := solicitudSituacionPrueba(t, ahora)
	base.Destino = domain.SituacionPendienteIncorporacion
	q := ports.SolicitudConfirmarAceptacionDirecta{SolicitudCambiarSituacionParticipacion: base,
		LlamamientoRef: "llamamiento:01", SituacionEsperadaDesde: ahora}
	_, err = s.ConfirmarAceptacionDirecta(context.Background(), q)
	if !errors.Is(err, vecports.ErrInstantaneaAutorizacionObsoleta) ||
		repo.consultas != 0 || repo.escrituras != 0 {
		t.Fatalf("rol obsoleto alcanzó datos: err=%v consultas=%d escrituras=%d", err, repo.consultas, repo.escrituras)
	}
}

func TestCanonConfirmacionDirectaLigaFichaMotivoCASYActor(t *testing.T) {
	ahora := time.Date(2026, 10, 16, 9, 0, 0, 0, time.UTC)
	base := solicitudSituacionPrueba(t, ahora)
	base.Destino = domain.SituacionPendienteIncorporacion
	q := ports.SolicitudConfirmarAceptacionDirecta{SolicitudCambiarSituacionParticipacion: base,
		LlamamientoRef: "llamamiento:01", SituacionEsperadaDesde: ahora}
	canon, huella, err := ports.CanonConfirmacionAceptacionDirecta(q)
	if err != nil || !strings.Contains(canon, `"llamamiento_ref":"llamamiento:01"`) ||
		!strings.Contains(canon, `"destino":"pendiente_incorporacion"`) ||
		!strings.Contains(canon, `"actor_ref":"`+base.ResultadoContexto.Contexto.PersonaRef+`"`) {
		t.Fatalf("canon no liga el acto: %v", err)
	}
	suma := sha256.Sum256([]byte(canon))
	if huella != hex.EncodeToString(suma[:]) {
		t.Fatal("huella distinta de los bytes canónicos")
	}
	q.Motivo = "Aceptación comprobada"
	_, distinta, err := ports.CanonConfirmacionAceptacionDirecta(q)
	if err != nil || distinta == huella {
		t.Fatal("alterar el motivo conservó la huella")
	}
}
