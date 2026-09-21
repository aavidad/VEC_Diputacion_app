package bootstrap

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
)

type filaAuditoriaFronteraBolsaPrueba struct {
	usuario string
	valida  bool
	err     error
}

func (f filaAuditoriaFronteraBolsaPrueba) Scan(destinos ...any) error {
	if f.err != nil {
		return f.err
	}
	*(destinos[0].(*string)) = f.usuario
	*(destinos[1].(*bool)) = f.valida
	return nil
}

type consultadorAuditoriaFronteraBolsaPrueba struct {
	fila     filaAuditoriaFronteraBolsaPrueba
	rol      string
	consulta string
}

func (c *consultadorAuditoriaFronteraBolsaPrueba) QueryRow(_ context.Context, consulta string, argumentos ...any) pgx.Row {
	c.consulta = consulta
	if len(argumentos) == 1 {
		c.rol, _ = argumentos[0].(string)
	}
	return c.fila
}

type filaPreflightAuditoriaFronteraBolsaPrueba struct{ valida bool }

func (f filaPreflightAuditoriaFronteraBolsaPrueba) Scan(destinos ...any) error {
	*(destinos[0].(*bool)) = f.valida
	return nil
}

type consultadorPreflightAuditoriaFronteraBolsaPrueba struct {
	fila     filaPreflightAuditoriaFronteraBolsaPrueba
	consulta string
}

func (c *consultadorPreflightAuditoriaFronteraBolsaPrueba) QueryRow(_ context.Context, consulta string, _ ...any) pgx.Row {
	c.consulta = consulta
	return c.fila
}

func TestComprobarIdentidadAuditoriaFronteraBolsaExigeSoloElRegistrador(t *testing.T) {
	consultador := &consultadorAuditoriaFronteraBolsaPrueba{fila: filaAuditoriaFronteraBolsaPrueba{usuario: "login_registrador", valida: true}}
	usuario, err := comprobarIdentidadPostgreSQLContratacionTemporalDesarrollo(context.Background(), consultador, rolAuditoriaFronteraPostgreSQLBolsaDesarrollo)
	if err != nil || usuario != "login_registrador" || consultador.rol != rolAuditoriaFronteraPostgreSQLBolsaDesarrollo {
		t.Fatalf("identidad Bolsa = (%q, %q, %v)", usuario, consultador.rol, err)
	}
	for _, fragmento := range []string{"NOT grupo.rolcanlogin", "grupo.rolinherit", "grupo.rolbypassrls", "membresias_efectivas", "admin_option", "rol_id <> $1::regrole OR admin_option", "grupo.oid = $1::regrole", "pg_has_role(session_user, $1::regrole, 'MEMBER')", "pg_has_role(session_user, $1::regrole, 'USAGE')"} {
		if !strings.Contains(consultador.consulta, fragmento) {
			t.Fatalf("checker no verificó grupo mínimo: %q", fragmento)
		}
	}
	if strings.Contains(consultador.consulta, "grupo.rolname = $1") {
		t.Fatal("checker compara name con regrole y PostgreSQL rechaza el preflight")
	}
	_, err = comprobarIdentidadPostgreSQLContratacionTemporalDesarrollo(context.Background(), &consultadorAuditoriaFronteraBolsaPrueba{fila: filaAuditoriaFronteraBolsaPrueba{usuario: "login_compartido"}}, rolAuditoriaFronteraPostgreSQLBolsaDesarrollo)
	if !errors.Is(err, errPostgreSQLContratacionTemporalDesarrolloNoDisponible) {
		t.Fatalf("membresía compartida aceptada: %v", err)
	}
}

func TestPreflightAuditoriaFronteraBolsaExigeSoloLaCapacidadDeBitacora(t *testing.T) {
	consultador := &consultadorPreflightAuditoriaFronteraBolsaPrueba{fila: filaPreflightAuditoriaFronteraBolsaPrueba{valida: true}}
	if err := comprobarPreflightAuditoriaFronteraBorradorLlamamientoPostgreSQLDesarrollo(context.Background(), consultador); err != nil {
		t.Fatal(err)
	}
	for _, fragmento := range []string{"WITH funcion_nominal", "esquemas_usuario", "NOT IN ('pg_catalog', 'information_schema')", "NOT LIKE 'pg_toast%'", "NOT LIKE 'pg_temp_%'", "pg_is_other_temp_schema", "has_database_privilege(session_user, pg_catalog.current_database(), 'CREATE')", "has_database_privilege(session_user, pg_catalog.current_database(), 'TEMPORARY')", "pg_catalog.pg_database AS base", "base.datdba", "aclexplode(base.datacl)", "aclexplode(espacio.nspacl)", "acl.is_grantable", "has_any_column_privilege", "has_sequence_privilege", "has_schema_privilege", "pg_has_role(session_user, espacio.nspowner, 'MEMBER')", "pg_has_role(session_user, funcion.proowner, 'MEMBER')", "pg_has_role(session_user, objeto.relowner, 'MEMBER')", "pg_has_role(session_user, tipo.typowner, 'MEMBER')", "aclexplode(funcion.proacl)", "acl.grantee = 0", "acl.privilege_type = 'EXECUTE'", "'CREATE'", "TRIGGER,MAINTAIN", "objeto.relkind IN ('r', 'p', 'v', 'm', 'f')"} {
		if !strings.Contains(consultador.consulta, fragmento) {
			t.Fatalf("preflight no cierra capacidad efectiva: %q", fragmento)
		}
	}
	for _, fragmento := range []string{"nspowner = session_user", "proowner = session_user", "relowner = session_user", "typowner = session_user"} {
		if strings.Contains(consultador.consulta, fragmento) {
			t.Fatalf("preflight solo comprobó propiedad directa: %q", fragmento)
		}
	}
	for _, fragmento := range []string{"nspowner, 'USAGE'", "proowner, 'USAGE'", "relowner, 'USAGE'", "typowner, 'USAGE'"} {
		if strings.Contains(consultador.consulta, fragmento) {
			t.Fatalf("ownership no cubre pertenencia SET ROLE: %q", fragmento)
		}
	}
	if strings.Contains(consultador.consulta, "has_type_privilege") {
		t.Fatal("USAGE de tipo heredado de PUBLIC no concede negocio ni debe bloquear el registrador")
	}
	for _, fragmento := range []string{"LIKE 'vec\\_%' ESCAPE '\\'", "coalesce(funcion.proacl", "acldefault('f'"} {
		if strings.Contains(consultador.consulta, fragmento) {
			t.Fatalf("preflight conserva un filtro Bolsa o convierte el EXECUTE público intrínseco en concesión: %q", fragmento)
		}
	}
	if err := comprobarPreflightAuditoriaFronteraBorradorLlamamientoPostgreSQLDesarrollo(context.Background(), &consultadorPreflightAuditoriaFronteraBolsaPrueba{}); !errors.Is(err, errBorradorLlamamientoDesarrolloNoDisponible) {
		t.Fatalf("preflight inválido = %v", err)
	}
}

func TestAuditoriaFronteraBolsaVerificaCadaConexionSinCambiarPoolsCT(t *testing.T) {
	configuracion, err := pgxpool.ParseConfig("postgres://registrador@localhost/vec?sslmode=require")
	if err != nil {
		t.Fatal(err)
	}
	configuracion.MaxConns = 4
	configurarVerificacionPorConexionAuditoriaFronteraBolsaDesarrollo(configuracion, rolAuditoriaFronteraPostgreSQLBolsaDesarrollo)
	if configuracion.AfterConnect == nil || configuracion.BeforeAcquire == nil || configuracion.MaxConns != 4 {
		t.Fatal("el registrador no conserva callbacks por conexión y límites del pool")
	}
	poolCT, err := pgxpool.ParseConfig("postgres://ejecutor@localhost/vec?sslmode=require")
	if err != nil {
		t.Fatal(err)
	}
	configurarVerificacionPorConexionAuditoriaFronteraBolsaDesarrollo(poolCT, rolEjecucionPostgreSQLContratacionTemporalDesarrollo)
	if poolCT.AfterConnect != nil || poolCT.BeforeAcquire != nil {
		t.Fatal("la verificación Bolsa alteró un pool CT")
	}
}

func TestNuevaAuditoriaFronteraBolsaNoAbreSinConfiguracionCompleta(t *testing.T) {
	base, err := config.NuevaConfiguracionPostgreSQLContratacionTemporal("ejecucion", "gobierno", "registro", "confirmador", "lector")
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = nuevaAuditoriaFronteraBorradorLlamamientoPostgreSQLDesarrollo(context.Background(), config.Config{ContratacionTemporalPostgreSQL: base})
	if !errors.Is(err, errBorradorLlamamientoDesarrolloNoDisponible) {
		t.Fatalf("auditoría ausente = %v", err)
	}
	t.Setenv(config.EnvBolsaLlamamientosDatabaseURL, "postgres://bolsa@localhost/vec")
	configurada := config.Load()
	_, _, err = nuevaAuditoriaFronteraBorradorLlamamientoPostgreSQLDesarrollo(context.Background(), configurada)
	if !errors.Is(err, errBorradorLlamamientoDesarrolloNoDisponible) {
		t.Fatalf("auditoría parcial = %v", err)
	}
	if strings.Contains(err.Error(), "bolsa") && strings.Contains(err.Error(), "dsn") {
		t.Fatal("error expuso configuración")
	}
}
