package postgrespreparacionbases

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	bolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	prep "vec-diputacion-granada/internal/modules/bolsa/domain/preparacionbases"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

// Doble del driver exclusivamente de prueba: verifica barrera de COMMIT y
// decodificacion, sin suplantar V3 ni conectarse a una base real.
type txPreparacionPrueba struct {
	pgx.Tx
	fila               filaPreparacion
	commits, rollbacks int
	falloCommit        error
}

func (x *txPreparacionPrueba) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (x *txPreparacionPrueba) QueryRow(context.Context, string, ...any) pgx.Row {
	return filaPreparacionPrueba{x.fila}
}
func (x *txPreparacionPrueba) Commit(context.Context) error   { x.commits++; return x.falloCommit }
func (x *txPreparacionPrueba) Rollback(context.Context) error { x.rollbacks++; return nil }

type poolPreparacionPrueba struct {
	tx       *txPreparacionPrueba
	opciones pgx.TxOptions
}

func (p *poolPreparacionPrueba) BeginTx(_ context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	p.opciones = o
	return p.tx, nil
}

type filaPreparacionPrueba struct{ f filaPreparacion }

func (x filaPreparacionPrueba) Scan(destinos ...any) error {
	f := x.f
	valores := []any{f.Estado, f.Referencia, f.Revision, f.HuellaMaterial, f.Material, f.Recibo, f.Historia, f.AuditoriaEfecto, f.Evento, f.HuellaIntencion, f.ConfirmadaEn, f.Acceso.DecisionRef, f.Acceso.ConsumoHuellaSHA256, f.Acceso.AuditoriaRef, f.Acceso.ReciboRef, f.Acceso.CorrelacionRef, f.Acceso.AccedidaEn}
	if len(destinos) != len(valores) {
		return errors.New("fila_prueba_incompatible")
	}
	for i, v := range valores {
		reflect.ValueOf(destinos[i]).Elem().Set(reflect.ValueOf(v))
	}
	return nil
}

func TestPreparacionPGAusenciaAuditableConfirmaAntesDeDevolver(t *testing.T) {
	a, _ := bolsa.NuevoAmbitoOrganizativoConvocatoria("org_diputaciongranada", "")
	x := &txPreparacionPrueba{fila: filaPreparacion{Estado: "no_encontrada", Acceso: ports.EvidenciaAccesoPreparacionBasesV3{DecisionRef: "decision:prueba", AccedidaEn: time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)}}}
	p := &poolPreparacionPrueba{tx: x}
	r := &Repositorio{}
	resultado, err := r.ejecutar(context.Background(), p, "consulta_sintetica", nil, a, func(v ports.ResultadoPreparacionBasesV3) error {
		if x.commits != 0 || v.Acceso.DecisionRef != "decision:prueba" {
			return ports.ErrResultadoPreparacionBasesInvalido
		}
		return nil
	})
	if err != nil || resultado.Estado != "no_encontrada" || x.commits != 1 || x.rollbacks != 1 || p.opciones.IsoLevel != pgx.Serializable || p.opciones.AccessMode != pgx.ReadWrite {
		t.Fatalf("orden de confirmacion: %+v %v", resultado, err)
	}
}

func TestPreparacionPGResultadoRechazadoNoHaceCommit(t *testing.T) {
	a, _ := bolsa.NuevoAmbitoOrganizativoConvocatoria("org_diputaciongranada", "")
	x := &txPreparacionPrueba{fila: filaPreparacion{Estado: "no_encontrada"}}
	p := &poolPreparacionPrueba{tx: x}
	r := &Repositorio{}
	resultado, err := r.ejecutar(context.Background(), p, "consulta_sintetica", nil, a, func(ports.ResultadoPreparacionBasesV3) error { return ports.ErrResultadoPreparacionBasesInvalido })
	if !errors.Is(err, ports.ErrResultadoPreparacionBasesInvalido) || x.commits != 0 || x.rollbacks != 1 || !reflect.DeepEqual(resultado, ports.ResultadoPreparacionBasesV3{}) {
		t.Fatal("respuesta no confiable confirmada")
	}
}

func TestPreparacionPGCommitFallidoNoDevuelveExitoNiDatos(t *testing.T) {
	a, _ := bolsa.NuevoAmbitoOrganizativoConvocatoria("org_diputaciongranada", "")
	x := &txPreparacionPrueba{fila: filaPreparacion{Estado: "no_encontrada"}, falloCommit: errors.New("commit_prueba_caido")}
	r := &Repositorio{}
	resultado, err := r.ejecutar(context.Background(), &poolPreparacionPrueba{tx: x}, "consulta_sintetica", nil, a, func(ports.ResultadoPreparacionBasesV3) error { return nil })
	if err == nil || !reflect.DeepEqual(resultado, ports.ResultadoPreparacionBasesV3{}) {
		t.Fatal("COMMIT fallido presentado como exito")
	}
}

func TestPreparacionPGNoEntregaMaterialEnConflictoNiBytesAlterados(t *testing.T) {
	a, _ := bolsa.NuevoAmbitoOrganizativoConvocatoria("org_diputaciongranada", "")
	ref := "prep:prueba"
	v := 1
	h := "huella"
	instante := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	for _, f := range []filaPreparacion{
		{Estado: "version_en_conflicto", Referencia: &ref},
		{Estado: "guardada", Referencia: &ref, Revision: &v, HuellaMaterial: &h, Material: []byte(`{}`), Recibo: &ref, Historia: &ref, AuditoriaEfecto: &ref, Evento: &ref, HuellaIntencion: &h, ConfirmadaEn: &instante},
	} {
		if _, err := f.resultado(a); !errors.Is(err, ports.ErrResultadoPreparacionBasesInvalido) {
			t.Fatal("material de fila alterada entregado")
		}
	}
	if _, err := prep.DecodificarMaterialCanonico([]byte(`{}`)); err == nil {
		t.Fatal("fixture no prueba rechazo")
	}
}
