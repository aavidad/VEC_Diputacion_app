package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

type repositorioReintentoAvisosPrueba struct {
	repositorioOfertasPrueba
	comando *ports.ComandoReintentarAvisosOferta
}

func (r *repositorioReintentoAvisosPrueba) ReintentarAvisosSinConfirmacion(
	_ context.Context, c ports.ComandoReintentarAvisosOferta,
) (ports.OfertaPublicada, error) {
	r.comando = &c
	revision := c.RevisionAvisos + 1
	return ports.OfertaPublicada{OfertaRef: c.OfertaRef, BolsaRef: c.BolsaRef,
		Aviso:          &ports.AvisosOferta{Total: 1, Enviados: 1, Revision: &revision},
		ReintentoAviso: &ports.ReintentoAvisoOferta{ReciboRef: "recibo:aviso-oferta:1", Rearmados: 1}}, nil
}

func TestReintentoAvisoExigeConfirmacionYMaterialLigado(t *testing.T) {
	ahora := time.Date(2026, 10, 16, 6, 0, 0, 0, time.UTC)
	base := solicitudPublicarOfertaPrueba(t, ahora)
	q := ports.SolicitudReintentarAvisosOferta{Vinculo: base.Vinculo, ResultadoContexto: base.ResultadoContexto,
		BolsaRef: base.BolsaRef, OfertaRef: "oferta:" + strings.Repeat("a", 64),
		RevisionAvisos: 7, ConfirmarDuplicado: true, ClaveIdempotencia: "reintento-aviso-001",
		Correlacion: base.Correlacion, MotivoAutorizacion: base.MotivoAutorizacion}
	repo := &repositorioReintentoAvisosPrueba{}
	auth := &autorizadorBorradorPrueba{t: t, instante: ahora}
	s, err := NuevoServicioOfertasPublicadas(contextoContactoPrueba{}, auth, repo, &plazoOfertaPrueba{}, func() time.Time { return ahora })
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.ReintentarAvisosSinConfirmacion(t.Context(), q); !errors.Is(err, ports.ErrOfertaNoDisponible) || auth.llamadas != 0 {
		t.Fatalf("sin canal, err=%v llamadas=%d", err, auth.llamadas)
	}
	s.EstablecerAvisosOferta(&DespachadorAvisosOferta{})
	q.ConfirmarDuplicado = false
	if _, err := s.ReintentarAvisosSinConfirmacion(t.Context(), q); !errors.Is(err, ports.ErrOfertaInvalida) || auth.llamadas != 0 {
		t.Fatalf("sin confirmación, err=%v llamadas=%d", err, auth.llamadas)
	}
	q.ConfirmarDuplicado = true
	o, err := s.ReintentarAvisosSinConfirmacion(t.Context(), q)
	if err != nil || o.ReintentoAviso == nil || repo.comando == nil || repo.comando.Material.ValidarEstructura() != nil ||
		repo.comando.ActorRef != base.ResultadoContexto.Contexto.PersonaRef || repo.comando.RevisionAvisos != 7 ||
		!repo.comando.ConfirmarDuplicado || auth.llamadas != 1 {
		t.Fatalf("reintento=%+v comando=%+v err=%v llamadas=%d", o, repo.comando, err, auth.llamadas)
	}
}

func TestHuellaReintentoAvisoCoincideConB132(t *testing.T) {
	campos := []string{"vec.bolsa.oferta.avisos.reintentar.v1", "bolsa:of",
		"oferta:" + strings.Repeat("a", 64), "per_0123456789abcdefghijkl", "7", "reintento-aviso-001", "true"}
	if got := huellaCamposOfertaV5(campos); got != "0bdc1c16510cb5221cc6d742ee52fc7888c39b0884317dc31af529ca1c344d79" {
		t.Fatalf("huella Go/SQL distinta: %s", got)
	}
}
