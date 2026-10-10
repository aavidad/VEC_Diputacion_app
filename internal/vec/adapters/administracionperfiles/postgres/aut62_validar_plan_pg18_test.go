package postgres

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/vec/domain"
)

// TestValidarPlanVersionRolBolsaRealConAsignacionesPostgreSQL18 valida con la
// función SQL real de AUT62 un plan B1 construido por el dominio Go con las
// asignaciones vigentes del rol base, en sus dos modos (primera aplicación y
// replay). Antes de corregir el alias de la comprobación de revocación,
// PL/pgSQL tomaba «h.asignacion_id» como campo de la variable «h» y cualquier
// plan con asignaciones fallaba con 42703 (503 sql_intento_error en vec-admin).
//
// Sólo corre contra una base desechable con AUT62 y el catálogo B1 cargado
// (por ejemplo, una copia tras los actos ADMIN). El DSN debe ser de un
// superusuario de esa copia: la función sólo tiene EXECUTE para su
// propietario. Todo va en una transacción que se revierte.
func TestValidarPlanVersionRolBolsaRealConAsignacionesPostgreSQL18(t *testing.T) {
	if os.Getenv("VEC_AUT62_PG_DESECHABLE") != "si" {
		t.Skip("requiere PostgreSQL 18 desechable con AUT62 y el catálogo B1")
	}
	dsn := os.Getenv("VEC_AUT62_PG_DSN")
	if dsn == "" {
		t.Fatal("falta VEC_AUT62_PG_DSN")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("conexión: %v", err)
	}
	defer pool.Close()
	plan := planB1RealConAsignaciones(ctx, t, pool)
	planJSON, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	for _, primera := range []bool{true, false} {
		tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
		if err != nil {
			t.Fatal(err)
		}
		var r []byte
		err = tx.QueryRow(ctx, `SELECT vec_autorizacion.validar_plan_version_rol_bolsa_v1($1::jsonb,$2)`,
			string(planJSON), primera).Scan(&r)
		_ = tx.Rollback(context.Background())
		var pg *pgconn.PgError
		if errors.As(err, &pg) {
			t.Fatalf("primera=%v: SQLSTATE %s %q", primera, pg.Code, pg.Message)
		}
		if err != nil || len(r) == 0 {
			t.Fatalf("primera=%v: %v (respuesta %d bytes)", primera, err, len(r))
		}
	}
}

// planB1RealConAsignaciones prepara con el dominio Go un plan sobre la cabeza
// del catálogo y las asignaciones actuales del rol base, como lo haría el DTO.
func planB1RealConAsignaciones(ctx context.Context, t *testing.T, pool *pgxpool.Pool) domain.PlanVersionarRolBolsa {
	t.Helper()
	fuente, err := nuevaFuenteCatalogoAcciones(ctx, pool)
	if err != nil {
		t.Fatal("fuente:", err)
	}
	var ref, huella string
	var version int
	if err := pool.QueryRow(ctx, `SELECT catalogo_ref,version,huella_sha256
	 FROM vec_autorizacion.cabeza_catalogo_acciones_admin_v1`).Scan(&ref, &version, &huella); err != nil {
		t.Fatal("cabeza del catálogo:", err)
	}
	catalogo, err := fuente.ObtenerCatalogoAccionesAdministracionV1(ctx, ref, version, huella)
	if err != nil {
		t.Fatal("catálogo:", err)
	}
	filas, err := pool.Query(ctx, `SELECT a.documento::text FROM vec_autorizacion.asignacion_perfil_actual q
	 JOIN vec_autorizacion.asignacion_perfil a USING(asignacion_ref)
	 WHERE a.version_rol_ref LIKE 'rol:'||$1||':v%' ORDER BY 1`, domain.RolIDVersionarBolsaB1)
	if err != nil {
		t.Fatal(err)
	}
	var asignaciones []domain.SeleccionAsignacionVersionarRolBolsa
	for filas.Next() {
		var doc string
		if err := filas.Scan(&doc); err != nil {
			t.Fatal(err)
		}
		var a domain.AsignacionPerfil
		if err := json.Unmarshal([]byte(doc), &a); err != nil {
			t.Fatal(err)
		}
		h, err := a.HuellaSHA256()
		if err != nil {
			t.Fatal(err)
		}
		asignaciones = append(asignaciones, domain.SeleccionAsignacionVersionarRolBolsa{
			AsignacionRef: a.Referencia(), HuellaSHA256: h, Documento: a})
	}
	filas.Close()
	if err := filas.Err(); err != nil {
		t.Fatal(err)
	}
	if len(asignaciones) == 0 {
		t.Fatal("la base no tiene asignaciones actuales del rol base: el caso que se prueba exige al menos una")
	}
	var base domain.PerfilPublicadoAdministracionV1
	for _, p := range catalogo.Perfiles {
		if p.Rol.RolID == domain.RolIDVersionarBolsaB1 {
			base = p
		}
	}
	hb, err := base.Rol.HuellaSHA256()
	if err != nil {
		t.Fatal("rol base:", err)
	}
	hcv, err := base.ControlVigencia.HuellaSHA256()
	if err != nil {
		t.Fatal("control de vigencia:", err)
	}
	var e domain.EntradaAccionAdministracionV1
	for _, x := range catalogo.Entradas {
		if x.Concesion.ModuloID == "bolsa" && x.Concesion.Accion == "bolsa.carga_convoca.confirmar" {
			e = x
		}
	}
	if e.Referencia == "" {
		t.Fatal("el catálogo no tiene la entrada B1 de carga CONVOCA")
	}
	he, err := e.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	motivo := domain.ReferenciaEntradaCatalogo{CatalogoID: "motivos.admin.prueba", CatalogoVersion: 1,
		CatalogoHuellaSHA256: hex.EncodeToString(make([]byte, 31)) + "01", EntradaClave: "motivo_" + hex.EncodeToString([]byte("prueba-aut62-pg1"))}
	intencion := domain.IntencionVersionarRolBolsa{CatalogoRef: catalogo.Referencia, CatalogoVersion: catalogo.Version,
		CatalogoHuellaSHA256: huella, BaseRef: base.Rol.Referencia(), BaseHuellaSHA256: hb,
		ControlRevision: base.ControlVigencia.Revision, ControlHuellaSHA256: hcv,
		Seleccion: domain.SeleccionAccionAdministracionV1{EntradaRef: e.Referencia, EntradaVersion: e.Version,
			EntradaHuellaSHA256: he},
		Asignaciones: asignaciones, Motivo: motivo}
	plan, err := domain.PrepararPlanVersionarRolBolsa(catalogo, intencion, time.Now().UTC().Truncate(time.Microsecond))
	if err != nil {
		t.Fatal("plan:", err)
	}
	return plan
}
