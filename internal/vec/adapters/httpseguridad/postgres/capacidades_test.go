package postgres

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
)

type consultorCapacidadPrueba struct {
	fila       pgx.Row
	consulta   string
	argumentos []any
}

func (c *consultorCapacidadPrueba) QueryRow(
	_ context.Context,
	consulta string,
	argumentos ...any,
) pgx.Row {
	c.consulta = consulta
	c.argumentos = append([]any(nil), argumentos...)
	return c.fila
}

func TestManifiestosCapacidadIdentidadSonCerradosYExactos(t *testing.T) {
	casos := []struct {
		capacidad string
		funciones [2]string
	}{
		{
			capacidad: capacidadProvisionar,
			funciones: [2]string{
				"vec_identidad_sesiones_v1.provisionar_cuenta_v1(text,text,text,text,bigint,bytea,bytea,boolean,bytea)",
				"vec_identidad_sesiones_v1.registrar_alias_hmac_cuenta_v1(text,text,text,text,text,bigint,bytea,bytea)",
			},
		},
		{
			capacidad: capacidadRegistrar,
			funciones: [2]string{
				"vec_identidad_sesiones_v1.registrar_sesion_v1(text,text,text,text,bigint,bytea,bytea,bytea,bytea,bytea,boolean,text,text,text,text,timestamptz,timestamptz,timestamptz,text,text)",
				"vec_identidad_sesiones_v1.reconciliar_registro_sesion_v1(text,text,text,text,bigint,bytea,bytea,bytea,bytea,bytea,boolean,text,text,text,text,timestamptz,timestamptz,timestamptz,text,text)",
			},
		},
		{
			capacidad: capacidadRevalidar,
			funciones: [2]string{
				"vec_identidad_sesiones_v1.revalidar_sesion_y_cuentas_v1(text,text,text,text,text,text,boolean,text,text,text,text,text,timestamptz,timestamptz,text,text,text,text,timestamptz,timestamptz)",
				"vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(text,text)",
			},
		},
		{
			capacidad: capacidadRevocar,
			funciones: [2]string{
				"vec_identidad_sesiones_v1.cambiar_estado_cuenta_v1(text,text,text,text)",
				"vec_identidad_sesiones_v1.revocar_sesion_v1(text,text,text,text)",
			},
		},
	}
	for _, caso := range casos {
		t.Run(caso.capacidad, func(t *testing.T) {
			manifiesto, encontrado := manifiestoParaCapacidad(caso.capacidad)
			if !encontrado || !manifiesto.valido() ||
				manifiesto.grupo != caso.capacidad ||
				manifiesto.funciones != caso.funciones {
				t.Fatalf("manifiesto inesperado: %#v, encontrado=%t", manifiesto, encontrado)
			}
			firmas := manifiesto.firmasFunciones()
			firmas[0] = "mutada"
			if manifiesto.funciones != caso.funciones {
				t.Fatal("el manifiesto expuso almacenamiento mutable")
			}
		})
	}

	for _, capacidad := range []string{"", "vec_identidad_sesiones_v1_otra"} {
		if manifiesto, encontrado := manifiestoParaCapacidad(capacidad); encontrado ||
			manifiesto.valido() {
			t.Fatalf("capacidad fuera del cierre aceptada: %q", capacidad)
		}
	}
}

func TestAcreditacionCapacidadExigeIdentidadYPerfilExactos(t *testing.T) {
	filaValida := []any{
		"login-revalidador", "login-revalidador",
		true, true, true, true, true, true, true,
	}
	casos := []struct {
		nombre  string
		cambiar func([]any)
	}{
		{"usuario actual distinto", func(fila []any) { fila[1] = "rol-sustituido" }},
		{"login inseguro", func(fila []any) { fila[2] = false }},
		{"grupo inseguro", func(fila []any) { fila[3] = false }},
		{"membresia directa insegura", func(fila []any) { fila[4] = false }},
		{"membresia total adicional", func(fila []any) { fila[5] = false }},
		{"funciones no resueltas", func(fila []any) { fila[6] = false }},
		{"login con autoridad directa", func(fila []any) { fila[7] = false }},
		{"grupo con perfil adicional", func(fila []any) { fila[8] = false }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			fila := append([]any(nil), filaValida...)
			caso.cambiar(fila)
			consultor := &consultorCapacidadPrueba{fila: filaDoble{valores: fila}}
			usuario, err := acreditarCapacidad(
				context.Background(), consultor, capacidadRevalidar,
			)
			if usuario != "" ||
				!errors.Is(err, httpseguridad.ErrRegistroSesionesAusente) {
				t.Fatalf("acreditacion insegura aceptada: usuario=%q err=%v", usuario, err)
			}
		})
	}

	consultor := &consultorCapacidadPrueba{fila: filaDoble{valores: filaValida}}
	usuario, err := acreditarCapacidad(
		context.Background(), consultor, capacidadRevalidar,
	)
	if err != nil || usuario != "login-revalidador" {
		t.Fatalf("acreditacion exclusiva rechazada: usuario=%q err=%v", usuario, err)
	}
	manifiesto, _ := manifiestoParaCapacidad(capacidadRevalidar)
	if len(consultor.argumentos) != 2 ||
		consultor.argumentos[0] != manifiesto.grupo ||
		!reflect.DeepEqual(consultor.argumentos[1], manifiesto.firmasFunciones()) {
		t.Fatalf("argumentos fuera del manifiesto: %#v", consultor.argumentos)
	}
	fragmentosSQL := []string{
		"pg_catalog.pg_has_role", "'MEMBER'", "pg_catalog.pg_shdepend",
		"pg_catalog.aclexplode", "pg_catalog.pg_default_acl",
		"pg_catalog.pg_policy", "pg_catalog.pg_db_role_setting",
		"login.rolconfig IS NULL", "grupo.rolconfig IS NULL",
		"dependencia.deptype = 'a'", "count(*) = 4",
		"acl.privilege_type = 'CONNECT'", "acl.privilege_type = 'USAGE'",
		"acl.privilege_type = 'EXECUTE'", "NOT acl.is_grantable",
		"pg_catalog.to_regprocedure", "pg_catalog.unnest($2::text[])",
	}
	for _, fragmento := range fragmentosSQL {
		if !strings.Contains(consultor.consulta, fragmento) {
			t.Fatalf("la consulta no fija el perfil exacto: falta %q", fragmento)
		}
	}
}

func TestAcreditacionCapacidadRechazaDependenciasNulasDesconocidasYFilasIncompletas(t *testing.T) {
	var consultorNulo *consultorCapacidadPrueba
	if usuario, err := acreditarCapacidad(
		context.Background(), consultorNulo, capacidadRevalidar,
	); usuario != "" || !errors.Is(err, httpseguridad.ErrRegistroSesionesAusente) {
		t.Fatalf("consultor nulo tipado aceptado: usuario=%q err=%v", usuario, err)
	}

	consultorDesconocido := &consultorCapacidadPrueba{}
	if usuario, err := acreditarCapacidad(
		context.Background(), consultorDesconocido, "capacidad-no-declarada",
	); usuario != "" || !errors.Is(err, httpseguridad.ErrRegistroSesionesAusente) ||
		consultorDesconocido.consulta != "" {
		t.Fatalf("capacidad desconocida consultada: usuario=%q err=%v", usuario, err)
	}

	consultor := &consultorCapacidadPrueba{
		fila: filaDoble{valores: []any{"login", "login", true}},
	}
	if usuario, err := acreditarCapacidad(
		context.Background(), consultor, capacidadRevalidar,
	); usuario != "" || !errors.Is(err, httpseguridad.ErrRegistroSesionesAusente) {
		t.Fatalf("fila parcial aceptada: usuario=%q err=%v", usuario, err)
	}

	consultor.fila = filaDoble{err: errors.New("detalle interno")}
	if usuario, err := acreditarCapacidad(
		context.Background(), consultor, capacidadRevalidar,
	); usuario != "" || !errors.Is(err, httpseguridad.ErrRegistroSesionesAusente) ||
		strings.Contains(err.Error(), "detalle") {
		t.Fatalf("error interno expuesto: usuario=%q err=%v", usuario, err)
	}
}
