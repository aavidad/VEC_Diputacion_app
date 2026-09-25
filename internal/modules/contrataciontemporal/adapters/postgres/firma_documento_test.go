package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// Dobles mínimos del pool y la transacción: cada intento consume la
// siguiente respuesta del guion (un error de PostgreSQL o el recibo JSON).
type guionFirma118 struct {
	respuestas         []any
	inicios, confirmas int
	revertidas         int
}

type txFirma118 struct {
	pgx.Tx
	g         *guionFirma118
	respuesta any
}

type filaFirma118 struct{ respuesta any }

func (g *guionFirma118) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	r := g.respuestas[g.inicios]
	g.inicios++
	return &txFirma118{g: g, respuesta: r}, nil
}

func (t *txFirma118) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (t *txFirma118) QueryRow(context.Context, string, ...any) pgx.Row {
	return filaFirma118{respuesta: t.respuesta}
}

func (t *txFirma118) Commit(context.Context) error { t.g.confirmas++; return nil }

func (t *txFirma118) Rollback(context.Context) error { t.g.revertidas++; return nil }

func (f filaFirma118) Scan(destino ...any) error {
	if err, ok := f.respuesta.(error); ok {
		return err
	}
	*(destino[0].(*[]byte)) = []byte(f.respuesta.(string))
	return nil
}

const reciboFirmaPrueba118 = `{"FirmaRef":"firma-ct:1","ReciboRef":"recibo-firma-ct:1","Secuencia":1,"Resultado":"firmado","ExpedienteVersion":7,"ActorRef":"a","PerfilRef":"p","RegistradaEn":"2026-09-25T10:00:00Z","SolicitudHuella":"h","YaRegistrada":true}`

func errorPG118(codigo, restriccion string) error {
	return &pgconn.PgError{Code: codigo, ConstraintName: restriccion}
}

func TestRegistroFirma118CarrerasSimultaneas(t *testing.T) {
	serializacion := errorPG118("40001", "")
	casos := []struct {
		nombre                   string
		respuestas               []any
		validar                  error
		quiere                   error
		inicios, confirmas, revs int
	}{
		{"serializacion y clave ocupada se reintentan hasta recuperar el recibo",
			[]any{serializacion, errorPG118("23505", restriccionClaveFirma118), reciboFirmaPrueba118}, nil, nil, 3, 1, 2},
		{"interbloqueo se reintenta", []any{errorPG118("40P01", ""), reciboFirmaPrueba118}, nil, nil, 2, 1, 1},
		{"serializacion agotada es indisponibilidad, no conflicto",
			[]any{serializacion, serializacion, serializacion}, nil, ports.ErrRegistroFirmaDocumentoNoDisponible, 3, 0, 3},
		{"secuencia ocupada es conflicto sin reintento",
			[]any{errorPG118("23505", restriccionSecuenciaFirma118)}, nil, ports.ErrFirmaDocumentoEnConflicto, 1, 0, 1},
		{"otra restriccion unica no se reintenta ni es conflicto",
			[]any{errorPG118("23505", "firma_documento_v1_recibo_ref_key")}, nil, ports.ErrRegistroFirmaDocumentoNoDisponible, 1, 0, 1},
		{"P1183 es conflicto", []any{errorPG118("P1183", "")}, nil, ports.ErrFirmaDocumentoEnConflicto, 1, 0, 1},
		{"P1181 es clave reutilizada", []any{errorPG118("P1181", "")}, nil, ports.ErrClaveFirmaDocumentoUsada, 1, 0, 1},
		{"recibo incoherente no se confirma", []any{reciboFirmaPrueba118}, ports.ErrResultadoFirmaDocumentoInvalido,
			ports.ErrResultadoFirmaDocumentoInvalido, 1, 0, 1},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			g := &guionFirma118{respuestas: c.respuestas}
			r := &RegistroFirmasDocumentoPostgreSQL{pool: g}
			err := r.registrarConReintentos(context.Background(), nil, func(reciboFirmaSQL118) error { return c.validar })
			if !errors.Is(err, c.quiere) || (c.quiere == nil && err != nil) {
				t.Fatalf("error %v, quiere %v", err, c.quiere)
			}
			if g.inicios != c.inicios || g.confirmas != c.confirmas || g.revertidas != c.revs {
				t.Fatalf("inicios %d confirmas %d revertidas %d", g.inicios, g.confirmas, g.revertidas)
			}
		})
	}
}

func TestErrorFirma118ContextoCancelado(t *testing.T) {
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if err := errorFirma118(ctx, errorPG118("40001", "")); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelación: %v", err)
	}
}
