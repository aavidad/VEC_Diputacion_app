package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vp "vec-diputacion-granada/internal/vec/ports"
)

type filaCerrojoDerivacionPrueba struct {
	valor bool
	err   error
}

func (f filaCerrojoDerivacionPrueba) Scan(dest ...any) error {
	if f.err != nil {
		return f.err
	}
	*(dest[0].(*bool)) = f.valor
	return nil
}

type sesionDerivacionPrueba struct {
	eventos                              []string
	tomado, liberado                     bool
	clave                                string
	falloLock, falloUnlock, falloDiscard error
	tx                                   pgx.Tx
}

func (s *sesionDerivacionPrueba) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	if strings.Contains(sql, "pg_try_advisory_lock") {
		s.eventos = append(s.eventos, "lock")
		s.clave = args[0].(string)
		return filaCerrojoDerivacionPrueba{s.tomado, s.falloLock}
	}
	s.eventos = append(s.eventos, "unlock")
	if args[0] != s.clave {
		return filaCerrojoDerivacionPrueba{false, nil}
	}
	return filaCerrojoDerivacionPrueba{s.liberado, s.falloUnlock}
}
func (s *sesionDerivacionPrueba) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	s.eventos = append(s.eventos, "begin")
	if s.tx != nil {
		return s.tx, nil
	}
	return nil, errors.New("fallo sintético de BEGIN")
}
func (s *sesionDerivacionPrueba) Release() { s.eventos = append(s.eventos, "release") }
func (s *sesionDerivacionPrueba) Discard() error {
	s.eventos = append(s.eventos, "discard")
	return s.falloDiscard
}

type transaccionDerivacionPrueba struct {
	pgx.Tx
	eventos *[]string
}

func (t transaccionDerivacionPrueba) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	*t.eventos = append(*t.eventos, "exec")
	return pgconn.CommandTag{}, errors.New("fallo sintético de SET LOCAL")
}
func (t transaccionDerivacionPrueba) Rollback(context.Context) error {
	*t.eventos = append(*t.eventos, "rollback")
	return errors.New("fallo sintético de rollback")
}

func TestDerivacionCerrojoAntesDeBeginYLiberacionMismaSesion(t *testing.T) {
	s := ports.SolicitudDerivacionNombramiento{OfertaRef: "oferta:sintetica:01"}
	for _, tc := range []struct {
		nombre                               string
		tomado, liberado                     bool
		esperados                            string
		errorEsperado                        error
		falloLock, falloUnlock, falloDiscard error
	}{
		{"ocupado", false, false, "lock,release", ports.ErrDerivacionNombramientoConcurrente, nil, nil, nil},
		{"begin falla", true, true, "lock,begin,unlock,release", ports.ErrDerivacionNombramientoNoDisponible, nil, nil, nil},
		{"unlock falso", true, false, "lock,begin,unlock,discard", ErrCerrojoDerivacionVS001, nil, nil, nil},
		{"lock incierto", false, false, "lock,discard", ErrCerrojoDerivacionVS001, errors.New("respuesta perdida"), nil, nil},
		{"unlock error", true, false, "lock,begin,unlock,discard", ErrCerrojoDerivacionVS001, nil, errors.New("respuesta perdida"), nil},
		{"descarte error", true, false, "lock,begin,unlock,discard", ErrDescarteDerivacionVS001, nil, errors.New("respuesta perdida"), errors.New("close falló")},
	} {
		t.Run(tc.nombre, func(t *testing.T) {
			fake := &sesionDerivacionPrueba{tomado: tc.tomado, liberado: tc.liberado, falloLock: tc.falloLock, falloUnlock: tc.falloUnlock, falloDiscard: tc.falloDiscard}
			_, err := ejecutarDerivacionNombramiento(t.Context(), fake, s, nil, vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, "")
			if !errors.Is(err, tc.errorEsperado) || strings.Join(fake.eventos, ",") != tc.esperados ||
				fake.clave != "vec_bolsa_llamamientos:resolucion-oferta:"+s.OfertaRef {
				t.Fatalf("err=%v, eventos=%v, clave=%q", err, fake.eventos, fake.clave)
			}
		})
	}
}

func TestDerivacionFalloRollbackRetiraSesion(t *testing.T) {
	fake := &sesionDerivacionPrueba{tomado: true, liberado: true}
	fake.tx = transaccionDerivacionPrueba{eventos: &fake.eventos}
	_, err := ejecutarDerivacionNombramiento(t.Context(), fake,
		ports.SolicitudDerivacionNombramiento{OfertaRef: "oferta:sintetica:01"}, nil,
		vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, "")
	if !errors.Is(err, ErrReversionDerivacionVS001) || !errors.Is(err, ErrCerrojoDerivacionVS001) ||
		!errors.Is(err, ports.ErrDerivacionNombramientoNoDisponible) ||
		strings.Join(fake.eventos, ",") != "lock,begin,exec,rollback,discard" {
		t.Fatalf("err=%v eventos=%v", err, fake.eventos)
	}
}
