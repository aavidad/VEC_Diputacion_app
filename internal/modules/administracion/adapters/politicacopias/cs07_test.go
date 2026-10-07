package politicacopias

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	cs07 "vec-diputacion-granada/internal/modules/administracion/adapters/registrocopias"
	"vec-diputacion-granada/internal/modules/administracion/domain/operacionescopias"
	d "vec-diputacion-granada/internal/modules/administracion/domain/politicacopias"
	p "vec-diputacion-granada/internal/modules/administracion/ports/politicacopias"
	reg "vec-diputacion-granada/internal/modules/administracion/ports/registrocopias"
)

func TestCS07RealRegisterReopenReplayAndDestinationExclusion(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "control")
	if e := os.Mkdir(dir, 0700); e != nil {
		t.Fatal(e)
	}
	config := cs07.Config{Directorio: dir, RaicesRestauradas: []string{t.TempDir()}}
	backend, e := cs07.Abrir(config)
	if e != nil {
		t.Fatal(e)
	}
	bridge := ReservadorCS07{backend}
	ctx := context.Background()
	at := time.Date(2026, 10, 1, 0, 30, 0, 0, time.UTC)
	reservation := p.Reserva{Clave: "agenda:test", Politica: "politica:test", Version: 1, PoliticaSHA256: strings.Repeat("a", 64), Destino: "destino:test", Evento: d.EventoAgenda{Fecha: at, FinVentana: at.Add(time.Hour), FechaCivil: "2026-10-01"}}
	actor := attribution()
	first, e := bridge.Reservar(ctx, reservation, actor)
	if e != nil || first.Replay {
		t.Fatal(first, e)
	}
	backend, e = cs07.Abrir(config)
	if e != nil {
		t.Fatal(e)
	}
	bridge.Registro = backend
	second, e := bridge.Reservar(ctx, reservation, actor)
	if e != nil || !second.Replay || second.Operacion != first.Operacion {
		t.Fatal(second, e)
	}
	next := reservation
	next.Clave = "agenda:next"
	next.Evento.Fecha = next.Evento.Fecha.AddDate(0, 0, 1)
	next.Evento.FinVentana = next.Evento.FinVentana.AddDate(0, 0, 1)
	next.Evento.FechaCivil = "2026-10-02"
	if _, e = bridge.Reservar(ctx, next, actor); !errors.Is(e, reg.ErrDestinoOcupado) {
		t.Fatal("overlap permitted", e)
	}
	altered := reservation
	altered.Version = 2
	if _, e = bridge.Reservar(ctx, altered, actor); !errors.Is(e, operacionescopias.ErrConflicto) {
		t.Fatal("changed policy duplicated", e)
	}
	listed, e := backend.Listar(ctx, reg.Declaracion{Actor: actor.Actor, Correlacion: actor.Correlacion}, reg.Consulta{Limite: 25})
	if e != nil || len(listed.Operaciones) != 1 {
		t.Fatal("duplicate operation", listed, e)
	}
	// This proves durable reservation metadata only, not backup capture or validity.
}
