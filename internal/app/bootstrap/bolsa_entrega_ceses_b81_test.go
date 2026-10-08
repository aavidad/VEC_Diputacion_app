package bootstrap

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	puertosct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type filaCeseB81 struct {
	err     error
	valores []any
}

func (f filaCeseB81) Scan(destinos ...any) error {
	if f.err != nil {
		return f.err
	}
	if len(destinos) != len(f.valores) {
		return errors.New("número de campos inesperado")
	}
	for i, valor := range f.valores {
		reflect.ValueOf(destinos[i]).Elem().Set(reflect.ValueOf(valor))
	}
	return nil
}

type pasoCeseB81 struct {
	funcion string
	fila    filaCeseB81
}

type consultasCeseB81 struct {
	t        *testing.T
	pasos    []pasoCeseB81
	llamadas int
}

func (q *consultasCeseB81) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	q.t.Helper()
	if q.llamadas >= len(q.pasos) {
		q.t.Fatalf("consulta adicional: %s", sql)
	}
	p := q.pasos[q.llamadas]
	q.llamadas++
	if !strings.Contains(sql, p.funcion) {
		q.t.Fatalf("función esperada=%s actual=%s", p.funcion, sql)
	}
	if q.llamadas > 1 && (len(args) != 3 || args[0] == "" || args[1] == "" || args[2].(int64) < 0) {
		q.t.Fatalf("se perdieron los datos del cese: %v", args)
	}
	return p.fila
}

type lectorCesesB81 struct{ *lectorContratosCTPrueba }

func (l lectorCesesB81) LeerCesesBolsa(ctx context.Context, desde puertosct.CursorPublicacionContratosBolsa, limite int) ([]puertosct.EventoContratoBolsaPublicado, error) {
	return l.LeerContratosBolsa(ctx, desde, limite)
}

func eventoCeseB81(i int) puertosct.EventoContratoBolsaPublicado {
	e := eventoPublicadoPrueba(i, time.Now())
	e.Contenido = []byte(strings.ReplaceAll(string(e.Contenido), "incorporacion", "cese"))
	return e
}

func TestEntregaCesesB81AvanzaAlSiguienteSinDuplicarContadores(t *testing.T) {
	for _, reutilizada := range []bool{false, true} {
		t.Run(map[bool]string{false: "nuevo", true: "replay"}[reutilizada], func(t *testing.T) {
			fk := &pgconn.PgError{Code: "23503"}
			q := &consultasCeseB81{t: t, pasos: []pasoCeseB81{
				{"cursor_restriccion", filaCeseB81{err: pgx.ErrNoRows}},
				{"registrar_restriccion", filaCeseB81{err: fk}},
				{"confirmar_cese_ajeno", filaCeseB81{err: fk}},
				{"confirmar_cese_sin_candidato", filaCeseB81{valores: []any{reutilizada}}},
				{"registrar_restriccion", filaCeseB81{valores: []any{false, "recibo:2", "candidato:2", time.Now(), int64(1)}}},
			}}
			lector := lectorCesesB81{&lectorContratosCTPrueba{eventos: []puertosct.EventoContratoBolsaPublicado{eventoCeseB81(1), eventoCeseB81(2)}}}
			r, err := (&entregaCesesCTBolsa{lector: lector, pool: q, lote: 1}).entregar(context.Background())
			if err != nil || r.nuevos != 1 || r.rechazados != 0 || q.llamadas != 5 {
				t.Fatalf("resultado=%+v error=%v consultas=%d", r, err, q.llamadas)
			}
			if reutilizada && (r.reentregas != 1 || r.sinCandidato != 0) || !reutilizada && (r.sinCandidato != 1 || r.reentregas != 0) {
				t.Fatalf("contadores duplicados: %+v", r)
			}
			if len(lector.desdes) != 3 || lector.desdes[1].OrigenRef != "ref:outbox:001" || lector.desdes[2].OrigenRef != "ref:outbox:002" {
				t.Fatalf("cursor no avanzó: %v", lector.desdes)
			}
		})
	}
}

func TestEntregaCesesB81NoSaltaErroresNiBolsaConstituida(t *testing.T) {
	for _, caso := range []struct {
		nombre, codigo string
		paso           int
	}{
		{"restriccion_denegada", "42501", 1},
		{"ajeno_caido", "08006", 2},
		{"constituida_pendiente", "23503", 3},
		{"huella_divergente", "VBC01", 3},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			q := &consultasCeseB81{t: t, pasos: []pasoCeseB81{
				{"cursor_restriccion", filaCeseB81{err: pgx.ErrNoRows}},
				{"registrar_restriccion", filaCeseB81{err: &pgconn.PgError{Code: "23503"}}},
				{"confirmar_cese_ajeno", filaCeseB81{err: &pgconn.PgError{Code: "23503"}}},
				{"confirmar_cese_sin_candidato", filaCeseB81{err: &pgconn.PgError{Code: "23503"}}},
			}}
			q.pasos[caso.paso].fila.err = &pgconn.PgError{Code: caso.codigo}
			lector := lectorCesesB81{&lectorContratosCTPrueba{eventos: []puertosct.EventoContratoBolsaPublicado{eventoCeseB81(1), eventoCeseB81(2)}}}
			r, err := (&entregaCesesCTBolsa{lector: lector, pool: q, lote: 1}).entregar(context.Background())
			if err == nil || r != (resultadoEntregaContratosCT{}) || q.llamadas != caso.paso+1 || len(lector.desdes) != 1 {
				t.Fatalf("cese rechazado saltado: resultado=%+v error=%v consultas=%d cursores=%v", r, err, q.llamadas, lector.desdes)
			}
		})
	}
}
