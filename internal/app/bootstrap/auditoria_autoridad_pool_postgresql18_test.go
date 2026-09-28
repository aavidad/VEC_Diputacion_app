package bootstrap

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"vec-diputacion-granada/internal/vec/auditoria"
)

// Requiere una instancia PostgreSQL 18 desechable y un DSN de superusuario.
// Solo crea tres grupos y tres LOGIN sintéticos; no instala migraciones.
func TestSondaAutoridadAuditoriaRolesPostgreSQL18(t *testing.T) {
	dsn := os.Getenv("VEC_AUDITORIA_SONDA_PG18_DSN")
	if dsn == "" {
		t.Skip("sin PostgreSQL 18 desechable para la sonda de roles Audit")
	}
	ctx := t.Context()
	admin, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { admin.Close(context.Background()) })
	var versionTexto string
	if err := admin.QueryRow(ctx, "SHOW server_version_num").Scan(&versionTexto); err != nil {
		t.Fatal(err)
	}
	version, err := strconv.Atoi(versionTexto)
	if err != nil || version/10000 != 18 {
		t.Fatalf("la sonda exige PostgreSQL 18: version=%s error=%v", versionTexto, err)
	}
	secret := make([]byte, 24)
	if _, err := rand.Read(secret); err != nil {
		t.Fatal(err)
	}
	password := hex.EncodeToString(secret)
	grupos := []struct {
		rol, login, herencia string
	}{
		{"vec_autorizacion_fuente", "vec_audit_fuente_sonda", "INHERIT"},
		{"vec_autorizacion_motivos_evaluador", "vec_audit_motivos_sonda", "NOINHERIT"},
		{"vec_contratacion_temporal_registrador_auditoria", "vec_audit_frontera_sonda", "INHERIT"},
	}
	for _, g := range grupos {
		if _, err := admin.Exec(ctx, fmt.Sprintf("CREATE ROLE %s NOLOGIN %s", g.rol, g.herencia)); err != nil {
			t.Fatalf("base de prueba no aislada o grupo no creable %s: %v", g.rol, err)
		}
		t.Cleanup(func() {
			_, _ = admin.Exec(context.Background(), "DROP ROLE "+g.rol)
		})
		if _, err := admin.Exec(ctx, fmt.Sprintf("CREATE ROLE %s LOGIN INHERIT PASSWORD '%s'", g.login, password)); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			_, _ = admin.Exec(context.Background(), "DROP ROLE "+g.login)
		})
		if _, err := admin.Exec(ctx, fmt.Sprintf(
			"GRANT %s TO %s WITH ADMIN FALSE, INHERIT TRUE, SET FALSE", g.rol, g.login)); err != nil {
			t.Fatal(err)
		}
	}
	comprobar := func(login, rol, esperado string, permite bool) {
		t.Helper()
		cfg, err := pgx.ParseConfig(dsn)
		if err != nil {
			t.Fatal(err)
		}
		cfg.User, cfg.Password = login, password
		conn, err := pgx.ConnectConfig(ctx, cfg)
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close(context.Background())
		err = comprobarPoolAutoridadAuditoriaDesarrollo(ctx, conn, esperado, rol)
		if permite && err != nil {
			t.Fatalf("%s/%s rechazado: %v", login, rol, err)
		}
		if !permite && !errors.Is(err, auditoria.ErrNoDisponible) {
			t.Fatalf("%s/%s admitido con identidad o permisos inválidos: %v", login, rol, err)
		}
	}
	for _, g := range grupos {
		comprobar(g.login, g.rol, g.login, true)
	}
	cfg, _ := pgx.ParseConfig(dsn)
	cfg.User, cfg.Password = grupos[1].login, password
	motivos, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	var uso bool
	if err := motivos.QueryRow(ctx, "SELECT pg_has_role(session_user, 'vec_autorizacion_motivos_evaluador', 'USAGE')").Scan(&uso); err != nil || !uso {
		t.Fatalf("PG18 no concede USAGE por membresía INHERIT TRUE de grupo NOINHERIT: uso=%v error=%v", uso, err)
	}
	motivos.Close(context.Background())

	cambiar := func(sql string) {
		t.Helper()
		if _, err := admin.Exec(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}
	cambiar("ALTER ROLE vec_autorizacion_motivos_evaluador INHERIT")
	comprobar(grupos[1].login, grupos[1].rol, grupos[1].login, false)
	cambiar("ALTER ROLE vec_autorizacion_motivos_evaluador NOINHERIT")

	cambiar("REVOKE vec_autorizacion_motivos_evaluador FROM vec_audit_motivos_sonda")
	cambiar("GRANT vec_autorizacion_motivos_evaluador TO vec_audit_motivos_sonda WITH ADMIN FALSE, INHERIT FALSE, SET FALSE")
	comprobar(grupos[1].login, grupos[1].rol, grupos[1].login, false)
	cambiar("REVOKE vec_autorizacion_motivos_evaluador FROM vec_audit_motivos_sonda")
	cambiar("GRANT vec_autorizacion_motivos_evaluador TO vec_audit_motivos_sonda WITH ADMIN FALSE, INHERIT TRUE, SET FALSE")

	cambiar("REVOKE vec_autorizacion_motivos_evaluador FROM vec_audit_motivos_sonda")
	cambiar("GRANT vec_autorizacion_motivos_evaluador TO vec_audit_motivos_sonda WITH ADMIN TRUE, INHERIT TRUE, SET FALSE")
	comprobar(grupos[1].login, grupos[1].rol, grupos[1].login, false)
	cambiar("REVOKE vec_autorizacion_motivos_evaluador FROM vec_audit_motivos_sonda")
	cambiar("GRANT vec_autorizacion_motivos_evaluador TO vec_audit_motivos_sonda WITH ADMIN FALSE, INHERIT TRUE, SET TRUE")
	comprobar(grupos[1].login, grupos[1].rol, grupos[1].login, false)
	cambiar("REVOKE vec_autorizacion_motivos_evaluador FROM vec_audit_motivos_sonda")
	cambiar("GRANT vec_autorizacion_motivos_evaluador TO vec_audit_motivos_sonda WITH ADMIN FALSE, INHERIT TRUE, SET FALSE")

	cambiar("GRANT vec_autorizacion_fuente TO vec_audit_motivos_sonda WITH ADMIN FALSE, INHERIT TRUE, SET FALSE")
	comprobar(grupos[1].login, grupos[1].rol, grupos[1].login, false)
	cambiar("REVOKE vec_autorizacion_fuente FROM vec_audit_motivos_sonda")

	comprobar(grupos[1].login, grupos[1].rol, grupos[0].login, false)
	cambiar("REVOKE vec_autorizacion_motivos_evaluador FROM vec_audit_motivos_sonda")
	comprobar(grupos[1].login, grupos[1].rol, grupos[1].login, false)
}

func TestSondaAutoridadAuditoriaSoloAceptaTresRoles(t *testing.T) {
	q := &consultadorPoolAuditoriaCTPrueba{fila: filaPoolAuditoriaCTPrueba{login: "login", valido: true}}
	for _, rol := range []string{"vec_autorizacion_fuente", "vec_autorizacion_motivos_evaluador", "vec_contratacion_temporal_registrador_auditoria"} {
		if err := comprobarPoolAutoridadAuditoriaDesarrollo(t.Context(), q, "login", rol); err != nil ||
			len(q.args) != 1 || q.args[0] != rol {
			t.Fatalf("rol nominal %q rechazado: error=%v args=%v", rol, err, q.args)
		}
	}
	for _, rol := range []string{"", "vec_autorizacion_motivos_proyector", "vec_autorizacion_registro", strings.Repeat("x", 64)} {
		if err := comprobarPoolAutoridadAuditoriaDesarrollo(t.Context(), q, "login", rol); !errors.Is(err, auditoria.ErrNoDisponible) {
			t.Fatalf("rol no autorizado %q: %v", rol, err)
		}
	}
}
