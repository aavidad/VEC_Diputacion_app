package postgres

import (
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vp "vec-diputacion-granada/internal/vec/ports"
)

type transaccionPlanB2SinFila struct {
	pgx.Tx
	t              *testing.T
	funcion        string
	consultas      int
	confirmaciones int
}

func (tx *transaccionPlanB2SinFila) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (tx *transaccionPlanB2SinFila) QueryRow(_ context.Context, consulta string, args ...any) pgx.Row {
	tx.consultas++
	if !strings.Contains(consulta, tx.funcion+"(") || len(args) != 11 {
		tx.t.Fatal("función SQL o parámetros inesperados")
	}
	return filaPlanB2SQLNulo{}
}
func (tx *transaccionPlanB2SinFila) Commit(context.Context) error {
	tx.confirmaciones++
	return nil
}
func (tx *transaccionPlanB2SinFila) Rollback(context.Context) error { return pgx.ErrTxClosed }

type filaPlanB2SQLNulo struct{}

func (filaPlanB2SQLNulo) Scan(dest ...any) error {
	*dest[0].(*[]byte) = nil // SQL NULL tras Scan de pgx.
	return nil
}

type iniciadorPlanB2SinFila struct{ tx *transaccionPlanB2SinFila }

func (p iniciadorPlanB2SinFila) BeginTx(_ context.Context, opciones pgx.TxOptions) (pgx.Tx, error) {
	if opciones.IsoLevel != pgx.Serializable {
		p.tx.t.Fatal("la lectura perdió aislamiento serializable")
	}
	return p.tx, nil
}

func capacidadPlanB2SinFila(t *testing.T, accion, audiencia string, material []byte) vp.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	t.Helper()
	huella, e := huellaContextoVinculoRPT("expediente:uno", "contratacion_temporal", ports.TipoRecursoPlanNominalB2,
		map[string]string{"organizacion_ref": "org:uno", "unidad_ref": "unidad:rrhh"}, material)
	if e != nil {
		t.Fatal(e)
	}
	ahora := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	resumen, e := vp.NuevoResumenCapacidadAtestacionAutorizacionV3("decision:uno", strings.Repeat("a", 64), strings.Repeat("b", 64),
		"contexto:uno", strings.Repeat("c", 64), accion, "expediente:uno", huella, audiencia, ahora, ahora.Add(time.Second))
	if e != nil {
		t.Fatal(e)
	}
	publica := ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize)).Public()
	spki, e := x509.MarshalPKIXPublicKey(publica)
	if e != nil {
		t.Fatal(e)
	}
	a, e := vp.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3([]byte(strings.Repeat("d", 600)), resumen,
		[]byte("decision"), []byte("motivo"), []byte("contexto"), 1, 1, []byte("payload"), []byte("sobre"), []byte("evidencia"), spki)
	if e != nil {
		t.Fatal(e)
	}
	return a
}

func TestPlanPersonalB2PrimeraLecturaSQLNula(t *testing.T) {
	selector := map[string]string{"organizacion_ref": "org:uno", "expediente_ref": "expediente:uno", "unidad_ref": "unidad:rrhh"}
	material, e := domain.CanonicoPlanPersonalB2(selector)
	if e != nil {
		t.Fatal(e)
	}
	a := capacidadPlanB2SinFila(t, ports.AccionLeerPlanNominalB2, ports.AudienciaLeerPlanNominalB2, material)
	for _, tc := range []struct {
		nombre  string
		funcion string
		leer    func(*FuentePlanNominalB2PostgreSQL) error
	}{
		{"GET plan", "leer_plan_nominal_b2_v1", func(f *FuentePlanNominalB2PostgreSQL) error {
			_, err := f.LeerContratoPlanNominal(context.Background(), "org:uno", "expediente:uno", a, "unidad:rrhh")
			return err
		}},
		{"GET origen antes del POST", "leer_origen_incorporacion_b2_v1", func(f *FuentePlanNominalB2PostgreSQL) error {
			_, encontrado, err := f.LeerOrigenIncorporacionB2(context.Background(), "org:uno", "expediente:uno", a, "unidad:rrhh")
			if encontrado {
				t.Fatal("origen inexistente comunicado como encontrado")
			}
			return err
		}},
	} {
		t.Run(tc.nombre, func(t *testing.T) {
			tx := &transaccionPlanB2SinFila{t: t, funcion: tc.funcion}
			f := &FuentePlanNominalB2PostgreSQL{pool: iniciadorPlanB2SinFila{tx}}
			err := tc.leer(f)
			if tc.funcion == "leer_plan_nominal_b2_v1" && !errors.Is(err, ports.ErrPlanNominalB2NoEncontrado) {
				t.Fatalf("primera lectura sin plan: %v", err)
			}
			if tc.funcion == "leer_origen_incorporacion_b2_v1" && err != nil {
				t.Fatalf("origen ausente: %v", err)
			}
			if tx.consultas != 1 || tx.confirmaciones != 1 {
				t.Fatalf("lectura ausente sin cierre limpio: consultas=%d commits=%d", tx.consultas, tx.confirmaciones)
			}
		})
	}
}
