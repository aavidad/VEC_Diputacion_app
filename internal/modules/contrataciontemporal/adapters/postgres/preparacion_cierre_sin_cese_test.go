package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"reflect"
	"testing"
	"time"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type poolConsultaCierre87 struct {
	tx       *txConsultaCierre87
	opciones pgx.TxOptions
	inicios  int
}

func (p *poolConsultaCierre87) BeginTx(_ context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	p.inicios++
	p.opciones = o
	return p.tx, nil
}

type txConsultaCierre87 struct {
	pgx.Tx
	t                           *testing.T
	s                           ports.SolicitudPreparacionCierreAdministrativo
	contenido                   []byte
	consultas, rollback, commit int
	cancelar                    func()
}

func (t *txConsultaCierre87) Exec(_ context.Context, q string, args ...any) (pgconn.CommandTag, error) {
	if q != ajustesRegistroIncorporacionTXV2 || len(args) != 0 {
		t.t.Fatal("configuración alterada")
	}
	return pgconn.CommandTag{}, nil
}
func (t *txConsultaCierre87) QueryRow(_ context.Context, q string, args ...any) pgx.Row {
	t.consultas++
	if q != consultarPreparacionCierreSQL87 || !reflect.DeepEqual(args, []any{t.s.OrganizacionRef, t.s.ExpedienteRef, t.s.SeguimientoRef}) {
		t.t.Fatal("lectura fuera de ámbito o función mutadora")
	}
	return filaConsultaCierre87{t}
}
func (t *txConsultaCierre87) Rollback(context.Context) error { t.rollback++; return nil }
func (t *txConsultaCierre87) Commit(context.Context) error {
	t.commit++
	return errors.New("consulta no confirma escritura")
}

type filaConsultaCierre87 struct{ tx *txConsultaCierre87 }

func (f filaConsultaCierre87) Scan(dest ...any) error {
	*dest[0].(*[]byte) = append([]byte(nil), f.tx.contenido...)
	if f.tx.cancelar != nil {
		f.tx.cancelar()
	}
	return nil
}
func fixtureConsultaCierre87(t *testing.T) (ports.SolicitudPreparacionCierreAdministrativo, ports.PreparacionCierreAdministrativo) {
	s := ports.SolicitudPreparacionCierreAdministrativo{OrganizacionRef: "organizacion:dipgra:prueba", ExpedienteRef: "expediente:ct:prueba", SeguimientoRef: refCierre87("seguimiento")}
	p := ports.PreparacionCierreAdministrativo{ExpedienteRef: s.ExpedienteRef, SeguimientoRef: s.SeguimientoRef, VersionActual: 1, EstadoActual: "vigente", PreparadaEn: time.Date(2026, 9, 10, 16, 0, 0, 0, time.UTC), Acciones: []ports.AccionPreparacionCierreAdministrativo{{TransicionClave: domain.TransicionCerrarAdministrativamenteSinCese, Motivos: []ports.MotivoPreparacionCierreAdministrativo{{MotivoClave: "cierre_administrativo_ejercicio"}}}}}
	return s, p
}
func TestPreparacionCierreSinCeseLeeInstantaneaSinEscrituras(t *testing.T) {
	s, p := fixtureConsultaCierre87(t)
	b, _ := json.Marshal(p)
	tx := &txConsultaCierre87{t: t, s: s, contenido: b}
	pool := &poolConsultaCierre87{tx: tx}
	l := &LectorPreparacionCierreAdministrativoPostgreSQL{pool: pool}
	got, err := l.ConsultarPreparacionCierreAdministrativo(context.Background(), s)
	if err != nil || !reflect.DeepEqual(got, p) || pool.opciones.IsoLevel != pgx.RepeatableRead || pool.opciones.AccessMode != pgx.ReadOnly || tx.consultas != 1 || tx.rollback != 1 || tx.commit != 0 {
		t.Fatalf("consulta no limitada: %v", err)
	}
}
func TestPreparacionCierreSinCeseDeniegaAmbitoYAccionesNoLigadas(t *testing.T) {
	s, p := fixtureConsultaCierre87(t)
	for _, c := range []struct {
		nombre  string
		cambiar func(*ports.PreparacionCierreAdministrativo)
	}{
		{"expediente", func(p *ports.PreparacionCierreAdministrativo) { p.ExpedienteRef = "expediente:ct:ajeno" }},
		{"seguimiento", func(p *ports.PreparacionCierreAdministrativo) { p.SeguimientoRef = refCierre87("ajeno") }},
		{"version", func(p *ports.PreparacionCierreAdministrativo) { p.VersionActual = 2 }},
		{"estado", func(p *ports.PreparacionCierreAdministrativo) { p.EstadoActual = "cerrado_administrativamente" }},
		{"sin_acciones", func(p *ports.PreparacionCierreAdministrativo) { p.Acciones = nil }},
		{"otra_transicion", func(p *ports.PreparacionCierreAdministrativo) { p.Acciones[0].TransicionClave = "cerrar_con_cese" }},
		{"motivo_repetido", func(p *ports.PreparacionCierreAdministrativo) {
			p.Acciones[0].Motivos = append(p.Acciones[0].Motivos, p.Acciones[0].Motivos[0])
		}},
	} {
		t.Run(c.nombre, func(t *testing.T) {
			_, x := fixtureConsultaCierre87(t)
			c.cambiar(&x)
			b, _ := json.Marshal(x)
			tx := &txConsultaCierre87{t: t, s: s, contenido: b}
			l := &LectorPreparacionCierreAdministrativoPostgreSQL{pool: &poolConsultaCierre87{tx: tx}}
			if _, err := l.ConsultarPreparacionCierreAdministrativo(context.Background(), s); !errors.Is(err, ports.ErrConsultaPreparacionCierreAdministrativoNoDisponible) || tx.rollback != 1 || tx.commit != 0 {
				t.Fatalf("dato ajeno: %v", err)
			}
		})
	}
	p.VersionActual = 2
	p.EstadoActual = "cerrado_administrativamente"
	p.Acciones = []ports.AccionPreparacionCierreAdministrativo{}
	if p.ValidarPara(s) != nil {
		t.Fatal("estado cerrado sin acción rechazado")
	}
}
func TestPreparacionCierreSinCesePropagaCancelacionYTieneEntradaNominal(t *testing.T) {
	s, p := fixtureConsultaCierre87(t)
	b, _ := json.Marshal(p)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tx := &txConsultaCierre87{t: t, s: s, contenido: b, cancelar: cancel}
	pool := &poolConsultaCierre87{tx: tx}
	l := &LectorPreparacionCierreAdministrativoPostgreSQL{pool: pool}
	if _, err := l.ConsultarPreparacionCierreAdministrativo(ctx, s); !errors.Is(err, context.Canceled) || tx.rollback != 1 || tx.commit != 0 {
		t.Fatalf("cancelación no preservada: %v", err)
	}
	s.OrganizacionRef = ""
	if _, err := l.ConsultarPreparacionCierreAdministrativo(context.Background(), s); !errors.Is(err, ports.ErrConsultaPreparacionCierreAdministrativoInvalida) || pool.inicios != 1 {
		t.Fatal("consulta abrió conexión sin organización confiable")
	}
}
