package adminperfiles

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type relojSeleccion struct{ ahora time.Time }

func (r relojSeleccion) Ahora() time.Time { return r.ahora }

type filaSeleccion struct {
	perfil string
	fecha  time.Time
}

func (f filaSeleccion) Scan(destinos ...any) error {
	if len(destinos) != 4 {
		return errors.New("firma de selector inesperada")
	}
	*destinos[0].(*string) = f.perfil
	*destinos[1].(*string) = "1"
	*destinos[2].(*time.Time) = f.fecha
	*destinos[3].(*string) = "auditoria_seleccion_admin:" + strings.Repeat("a", 64)
	return nil
}

type txSeleccion struct {
	pgx.Tx
	fila    pgx.Row
	commits int
	query   string
	args    []any
}

type filasSinPerfiles struct{ pgx.Rows }

func (filasSinPerfiles) Next() bool { return false }
func (filasSinPerfiles) Err() error { return nil }
func (filasSinPerfiles) Close()     {}
func (tx *txSeleccion) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return filasSinPerfiles{}, nil
}

func (tx *txSeleccion) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (tx *txSeleccion) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	tx.query, tx.args = query, append([]any(nil), args...)
	return tx.fila
}
func (tx *txSeleccion) Commit(context.Context) error   { tx.commits++; return nil }
func (tx *txSeleccion) Rollback(context.Context) error { return nil }

type poolSeleccion struct {
	tx       *txSeleccion
	opciones pgx.TxOptions
}

func (p *poolSeleccion) BeginTx(_ context.Context, opciones pgx.TxOptions) (pgx.Tx, error) {
	p.opciones = opciones
	return p.tx, nil
}
func (*poolSeleccion) QueryRow(context.Context, string, ...any) pgx.Row { return filaContextoFalsa{} }

func TestSelectorPOSTSoloConfirmaPerfilElegidoTrasCAS(t *testing.T) {
	ahora := time.Date(2026, 10, 2, 14, 0, 0, 0, time.UTC)
	perfil := "prf_" + strings.Repeat("a", 22)
	o := ObservacionADMIN{Entorno: "desarrollo", Host: "admin.example.invalid", Audiencia: "vec.admin.selector.v1",
		CertificadoSHA256: strings.Repeat("a", 64), CASHA256: strings.Repeat("b", 64),
		AutenticacionVerificadaEn: ahora.Add(-time.Minute), RevocacionVerificadaEn: ahora,
		CRLVigenteHasta: ahora.Add(time.Minute), CertificadoVigenteHasta: ahora.Add(time.Hour)}
	for _, caso := range []struct {
		nombre, devuelto string
		commits          int
	}{
		{"perfil elegido", perfil, 1},
		{"respuesta cruzada", "prf_" + strings.Repeat("c", 22), 0},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			tx := &txSeleccion{fila: filaSeleccion{perfil: caso.devuelto, fecha: ahora}}
			pool := &poolSeleccion{tx: tx}
			p := &PostgreSQL{pool: pool, reloj: relojSeleccion{ahora}}
			seleccion, err := p.SeleccionarPerfilADMIN(context.Background(), o, perfil, 0)
			if (err == nil) != (caso.commits == 1) || tx.commits != caso.commits ||
				pool.opciones.IsoLevel != pgx.Serializable || pool.opciones.AccessMode != pgx.ReadWrite ||
				!strings.Contains(tx.query, "seleccionar_perfil_admin_v1") || len(tx.args) != 11 ||
				tx.args[9] != perfil || tx.args[10] != "0" {
				t.Fatalf("resultado=%+v error=%v commits=%d consulta=%q args=%d", seleccion, err, tx.commits, tx.query, len(tx.args))
			}
		})
	}
}

func TestListadoPropioRecuperaConReadCommitted(t *testing.T) {
	ahora := time.Date(2026, 10, 2, 14, 0, 0, 0, time.UTC)
	o := ObservacionADMIN{Entorno: "desarrollo", Host: "admin.example.invalid", Audiencia: "vec.admin.selector.v1",
		CertificadoSHA256: strings.Repeat("a", 64), CASHA256: strings.Repeat("b", 64),
		AutenticacionVerificadaEn: ahora.Add(-time.Minute), RevocacionVerificadaEn: ahora,
		CRLVigenteHasta: ahora.Add(time.Minute), CertificadoVigenteHasta: ahora.Add(time.Hour)}
	pool := &poolSeleccion{tx: &txSeleccion{}}
	p := &PostgreSQL{pool: pool, reloj: relojSeleccion{ahora}}
	_, err := p.ListarPropiosADMIN(context.Background(), o)
	if err == nil || pool.opciones.IsoLevel != pgx.ReadCommitted || pool.opciones.AccessMode != pgx.ReadWrite {
		t.Fatalf("lectura sin datos/aislamiento: error=%v opciones=%+v", err, pool.opciones)
	}
}
