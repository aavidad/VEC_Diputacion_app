package postgres

import (
	"bytes"
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"reflect"
	"testing"
	"time"
	h "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/historiaincorporacion"
)

// Dobles de transporte explícitos; el Documento procede de APIs nominales reales.
type dobleLecturaHistoriaV2 struct {
	pgx.Tx
	t          *testing.T
	raw        []byte
	s          h.Selector
	fallo      string
	pasos      []string
	confirmado bool
	cancel     context.CancelFunc
}

func (m *dobleLecturaHistoriaV2) BeginTx(_ context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	m.pasos = append(m.pasos, "begin")
	if o != (pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadOnly}) {
		m.t.Fatal("opciones")
	}
	if m.fallo == "begin" {
		return m, errors.New("opaco")
	}
	return m, nil
}
func (m *dobleLecturaHistoriaV2) Exec(_ context.Context, q string, a ...any) (pgconn.CommandTag, error) {
	m.pasos = append(m.pasos, "settings")
	if q != ajustesHistoriaIncorporacionV2 || len(a) != 0 {
		m.t.Fatal("settings")
	}
	if m.fallo == "settings" {
		return pgconn.CommandTag{}, errors.New("opaco")
	}
	return pgconn.CommandTag{}, nil
}
func (m *dobleLecturaHistoriaV2) Query(_ context.Context, q string, a ...any) (pgx.Rows, error) {
	m.pasos = append(m.pasos, "query")
	if q != consultaHistoriaIncorporacionV2 || !reflect.DeepEqual(a, []any{m.s.ReciboRef, m.s.MaterialSHA256, m.s.IntencionSHA256}) {
		m.t.Fatal("selector3")
	}
	if m.fallo == "query" {
		return nil, errors.New("opaco")
	}
	return &filasLecturaHistoriaV2{m: m}, nil
}
func (m *dobleLecturaHistoriaV2) Commit(context.Context) error {
	m.pasos = append(m.pasos, "commit")
	if m.fallo == "commit" {
		return errors.New("opaco")
	}
	m.confirmado = true
	return nil
}
func (m *dobleLecturaHistoriaV2) Rollback(c context.Context) error {
	m.pasos = append(m.pasos, "rollback")
	d, ok := c.Deadline()
	if c.Err() != nil || !ok || time.Until(d) > 2*time.Second {
		m.t.Fatal("rollback no acotado")
	}
	return nil
}

type filasLecturaHistoriaV2 struct {
	pgx.Rows
	m *dobleLecturaHistoriaV2
	n int
}

func (f *filasLecturaHistoriaV2) Next() bool {
	f.n++
	return f.m.fallo != "sin_filas" && (f.n == 1 || (f.n == 2 && f.m.fallo == "dos_filas"))
}
func (f *filasLecturaHistoriaV2) Scan(d ...any) error {
	if f.m.fallo == "scan" {
		return errors.New("opaco")
	}
	*d[0].(*[]byte) = f.m.raw
	return nil
}
func (f *filasLecturaHistoriaV2) Err() error {
	if f.m.fallo == "rows" {
		return errors.New("opaco")
	}
	return nil
}
func (f *filasLecturaHistoriaV2) Close() {}

type relojLecturaHistoriaV2 func() time.Time

func (r relojLecturaHistoriaV2) Ahora() time.Time { return r() }
func TestLectorHistoriaIncorporacionV2NominalYFronteras(t *testing.T) {
	o, _, now := fixtureTransporte(t)
	w, _ := wireTransporte(t, o, now)
	b := serialTransporte(t, w.Historia)
	s := h.Selector{ReciboRef: w.Recibo.Transicion.ReciboRef, MaterialSHA256: w.Recibo.MaterialOriginalSHA256, IntencionSHA256: w.Recibo.IntencionSHA256}
	for _, caso := range []string{"nominal", "begin", "settings", "query", "sin_filas", "dos_filas", "scan", "rows", "json", "cota", "commit", "cancel_ultimo_reloj", "reloj_atras"} {
		t.Run(caso, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			m := &dobleLecturaHistoriaV2{t: t, raw: bytes.Clone(b), s: s, fallo: caso, cancel: cancel}
			if caso == "json" {
				m.raw = []byte("null")
			}
			if caso == "cota" {
				m.raw = make([]byte, h.MaximoBytesDocumento+1)
			}
			llamadas := 0
			r := relojLecturaHistoriaV2(func() time.Time {
				llamadas++
				if caso == "cancel_ultimo_reloj" && m.confirmado {
					cancel()
				}
				if caso == "reloj_atras" && llamadas > 1 {
					return now.Add(-time.Hour)
				}
				return now.Add(48 * time.Hour)
			})
			l, e := nuevoLectorHistoriaIncorporacionV2(m, r)
			if e != nil {
				t.Fatal("constructor")
			}
			got, e := l.LeerRegistroOriginal(ctx, s)
			if caso == "nominal" {
				if e != nil || !bytes.Equal(got, b) || !reflect.DeepEqual(m.pasos, []string{"begin", "settings", "query", "commit"}) {
					t.Fatal("nominal")
				}
				got[0] ^= 1
				if !bytes.Equal(m.raw, b) {
					t.Fatal("alias")
				}
				return
			}
			if e == nil || got != nil {
				t.Fatal("resultado no cero")
			}
			if caso == "cancel_ultimo_reloj" {
				if !errors.Is(e, context.Canceled) || !reflect.DeepEqual(m.pasos, []string{"begin", "settings", "query", "commit"}) {
					t.Fatal("cancelación postcommit")
				}
			} else if len(m.pasos) == 0 || m.pasos[len(m.pasos)-1] != "rollback" {
				t.Fatal("rollback ausente")
			}
		})
	}
}
