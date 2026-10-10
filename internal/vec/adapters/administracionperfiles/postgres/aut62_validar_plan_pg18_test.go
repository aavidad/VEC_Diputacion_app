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
// replay), y comprueba que una asignación revocada se rechaza con 40001.
// Antes de corregir el alias de la comprobación de revocación, PL/pgSQL tomaba
// «h.asignacion_id» como campo de la variable «h» y cualquier plan con
// asignaciones fallaba con 42703 (503 sql_intento_error en vec-admin).
//
// Sólo corre contra una base desechable con AUT62 y el catálogo B1 cargado
// (por ejemplo, una copia tras los actos ADMIN). El DSN debe ser de un
// superusuario de esa copia: la función sólo tiene EXECUTE para su
// propietario. Cada caso va en una transacción que se revierte.
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
	plan, _ := planB1RealConAsignaciones(ctx, t, pool)
	planJSON, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	for _, primera := range []bool{true, false} {
		r, err := validarPlanB1EnTx(ctx, pool, planJSON, primera, "")
		if err != nil {
			t.Errorf("primera=%v: %v", primera, err)
			continue
		}
		var entrada struct {
			Referencia string `json:"referencia"`
		}
		if json.Unmarshal(r, &entrada) != nil || entrada.Referencia != plan.Seleccion.EntradaRef {
			t.Errorf("primera=%v: entrada devuelta %q, esperada %q", primera, entrada.Referencia, plan.Seleccion.EntradaRef)
		}
	}
	// Una versión posterior «revocada» de la primera asignación del plan.
	revocar := `INSERT INTO vec_autorizacion.asignacion_perfil(asignacion_ref,asignacion_id,version,perfil_activo_ref,
	 principal_id,version_rol_ref,huella_sha256,emitida_en,documento)
	 SELECT 'asignacion:'||asignacion_id||':v'||(version+1),asignacion_id,version+1,perfil_activo_ref,principal_id,
	  version_rol_ref,repeat('0',64),emitida_en,
	  jsonb_set(jsonb_set(documento,'{version}',to_jsonb(version+1)),'{estado}','"revocada"')
	 FROM vec_autorizacion.asignacion_perfil WHERE asignacion_ref=$1`
	_, err = validarPlanB1EnTx(ctx, pool, planJSON, true, revocar, plan.Asignaciones[0].AsignacionRef)
	var pg *pgconn.PgError
	if !errors.As(err, &pg) || pg.Code != "40001" {
		t.Errorf("asignación revocada: se esperaba 40001 y llegó %v", err)
	}
}

// validarPlanB1EnTx llama a la función real dentro de una transacción que se
// revierte; si previo no está vacío, lo ejecuta antes en la misma transacción.
func validarPlanB1EnTx(ctx context.Context, pool *pgxpool.Pool, plan []byte, primera bool,
	previo string, args ...any) ([]byte, error) {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if previo != "" {
		if tag, err := tx.Exec(ctx, previo, args...); err != nil || tag.RowsAffected() != 1 {
			return nil, errors.Join(errors.New("preparación del caso"), err)
		}
	}
	var r []byte
	err = tx.QueryRow(ctx, `SELECT vec_autorizacion.validar_plan_version_rol_bolsa_v1($1::jsonb,$2)`,
		string(plan), primera).Scan(&r)
	if err == nil && len(r) == 0 {
		err = errors.New("respuesta vacía")
	}
	return r, err
}

// planB1RealConAsignaciones prepara con el dominio Go un plan sobre la cabeza
// del catálogo y las asignaciones actuales del rol base, como lo haría el DTO,
// y devuelve también la intención de la que sale.
func planB1RealConAsignaciones(ctx context.Context, t *testing.T,
	pool *pgxpool.Pool) (domain.PlanVersionarRolBolsa, domain.IntencionVersionarRolBolsa) {
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
	return plan, intencion
}
