package composicion

import (
	"context"
	"errors"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
)

type destinoIntentosRPTPrueba struct {
	evento         ports.EventoIntentoLectorRelacionRPT
	err            error
	checks, writes int
}

func (d *destinoIntentosRPTPrueba) VerificarDestinoRelacionRPT(context.Context) error {
	d.checks++
	return d.err
}
func (d *destinoIntentosRPTPrueba) RegistrarEventoRelacionRPT(_ context.Context, e ports.EventoIntentoLectorRelacionRPT) error {
	d.writes++
	d.evento = e
	return d.err
}
func TestLectorRPTIntentoNoInventaActorSinIdentidadActual(t *testing.T) {
	d := &destinoIntentosRPTPrueba{}
	r, err := NuevoRegistroIntentosLectorRelacionRPT(identidadLectorRelacionRPTPrueba{err: errors.New("sin identidad")}, d)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.RegistrarIntentoRelacionRPT(context.Background(), ports.IntentoLectorRelacionRPT{Motivo: "denegado", RelacionRef: "rel_" + strings.Repeat("a", 24)}); err != nil {
		t.Fatal(err)
	}
	if d.writes != 1 || d.evento.ActorRef != "" || d.evento.RelacionRef != "rel_"+strings.Repeat("a", 24) || !strings.HasPrefix(d.evento.CorrelacionRef, "correlacion_") {
		t.Fatal("registró actor no acreditado", d.evento)
	}
}
func TestLectorRPTIntentosCierranDestinoAusenteOFallido(t *testing.T) {
	if _, err := NuevoRegistroIntentosLectorRelacionRPT(identidadLectorRelacionRPTPrueba{}, nil); !errors.Is(err, domain.ErrLectorRelacionRPTNoDisponible) {
		t.Fatal("registrador nil admitido", err)
	}
	d := &destinoIntentosRPTPrueba{err: errors.New("detalle privado")}
	r, _ := NuevoRegistroIntentosLectorRelacionRPT(identidadLectorRelacionRPTPrueba{}, d)
	if err := r.VerificarRegistroRelacionRPT(context.Background()); !errors.Is(err, domain.ErrLectorRelacionRPTNoDisponible) || strings.Contains(err.Error(), "privado") {
		t.Fatal("preflight no cerrado", err)
	}
	if err := r.RegistrarIntentoRelacionRPT(context.Background(), ports.IntentoLectorRelacionRPT{Motivo: "no_disponible"}); !errors.Is(err, domain.ErrLectorRelacionRPTNoDisponible) || strings.Contains(err.Error(), "privado") {
		t.Fatal("destino no cerrado", err)
	}
}
