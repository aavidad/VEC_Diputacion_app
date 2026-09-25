package main

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"vec-diputacion-granada/internal/app/composicion/internactproveedores"
)

// Segundo paso del ensayo que prepara probar_postgresql_pg18.sh (cadena AD3
// 1/2 real). El primer paso ya publicó el gobierno con el publicador real de
// vec-server (TestPublicaGobiernoB2ComoVecServerParaEnsayoPostgreSQL18) desde
// el material de idempotencia VEC_T4_MATERIAL_IDEMPOTENCIA; aquí la
// herramienta lee ese mismo material y ese gobierno con el LOGIN de gobierno
// de vec-server, dentro de READ ONLY, y debe aceptarlo.
func TestCotejoContraGobiernoPostgreSQL18(t *testing.T) {
	if os.Getenv("VEC_T4_PG_DESECHABLE") != "si" {
		t.Skip("requiere PostgreSQL 18.4 desechable (probar_postgresql_pg18.sh)")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancelar()
	admin, err := pgx.Connect(ctx, os.Getenv("VEC_T4_PG_ADMIN_DSN"))
	if err != nil {
		t.Fatal("DBA de prueba no disponible")
	}
	defer admin.Close(context.Background())
	var publicado bool
	if err := admin.QueryRow(ctx, `SELECT current_setting('server_version_num') = '180004'
	   AND (SELECT count(*) FROM vec_autorizacion_atestada_v3.clave_capacidad_version
	         WHERE audiencia_consumo LIKE 'vec_personal.registro_empleado.%') = 8
	   AND (SELECT count(*) FROM vec_autorizacion_atestada_v3.raiz_confianza_version) = 1`).Scan(&publicado); err != nil || !publicado {
		t.Fatal("se exige PostgreSQL 18.4 desechable con el gobierno B2 publicado por vec-server")
	}
	contar := func() string {
		var s string
		if err := admin.QueryRow(ctx, `SELECT concat_ws(',',
		  (SELECT count(*) FROM vec_autorizacion_atestada_v3.clave_capacidad_version),
		  (SELECT count(*) FROM vec_autorizacion_atestada_v3.puntero_clave_emision),
		  (SELECT count(*) FROM vec_autorizacion_atestada_v3.revocacion_clave_capacidad),
		  (SELECT count(*) FROM vec_autorizacion_atestada_v3.configuracion_confianza_version),
		  (SELECT revision FROM vec_autorizacion_atestada_v3.checkpoint_gobierno))`).Scan(&s); err != nil {
			t.Fatal(err)
		}
		return s
	}
	antes := contar()
	gobiernoDSN := os.Getenv("VEC_T4_PG_GOBIERNO_DSN")
	d := dependencias{abrirGobierno: abrirGobiernoPostgreSQL, reloj: time.Now}
	// escenarioPublicado usa el material que publicó vec-server en lugar del
	// sintético propio del escenario.
	escenarioPublicado := func(t *testing.T) *escenario {
		e := nuevoEscenario(t)
		e.idempotencia = os.Getenv("VEC_T4_MATERIAL_IDEMPOTENCIA")
		return e
	}
	ejecutarCon := func(e *escenario, dsn string) (int, string) {
		var out, errOut bytes.Buffer
		sinArchivo := e.args()[:len(e.args())-2]
		codigo := ejecutar(ctx, sinArchivo, dsn, true, &out, &errOut, d)
		texto := out.String() + errOut.String()
		if strings.Contains(texto, "postgres") || strings.Contains(texto, "host=") {
			t.Fatal("la salida menciona el DSN")
		}
		return codigo, texto
	}
	e := escenarioPublicado(t)

	// El publicador asigna a cada clave B2 una revisión posterior a la del
	// checkpoint, que sólo avanza al publicarse una configuración o raíz
	// nueva (avanzar_checkpoint de AD3-2). Recién publicadas, la sonda de
	// vec-interno (AD3-69: revision_gobierno <= checkpoint) las rechazaría, y
	// la herramienta también.
	t.Run("publicacion_aun_fuera_del_checkpoint", func(t *testing.T) {
		e2 := escenarioPublicado(t)
		if codigo, texto := ejecutarCon(e2, gobiernoDSN); codigo != 1 || !strings.Contains(texto, string(errClaveB2)) {
			t.Fatalf("aceptó claves fuera del checkpoint: %s", texto)
		}
		e2.sinResiduos(t)
	})
	// El DBA sólo adelanta el checkpoint, como harían las renovaciones
	// posteriores; claves, punteros, raíz y configuración son los que dejó
	// el publicador real.
	if _, err := admin.Exec(ctx, `UPDATE vec_autorizacion_atestada_v3.checkpoint_gobierno
	   SET revision = (SELECT max(revision_gobierno) FROM vec_autorizacion_atestada_v3.clave_capacidad_version)
	 WHERE control_id`); err != nil {
		t.Fatal(err)
	}
	antes = contar()

	t.Run("positivo_publicado_por_vec_server", func(t *testing.T) {
		if codigo, texto := ejecutarCon(e, gobiernoDSN); codigo != 0 {
			t.Fatalf("código %d: %s", codigo, texto)
		}
		m, err := internactproveedores.CargarMaterialPersonalB2(e.salida)
		if err != nil {
			t.Fatal("cargador real rechazó el material")
		}
		defer m.Cerrar()
		var version, revision int64
		if err := admin.QueryRow(ctx, `SELECT version::bigint, revision_gobierno::bigint FROM vec_autorizacion_atestada_v3.clave_capacidad_version
		  WHERE clave_id = $1`, m.V3.Capacidades.Empleados.ClaveID).Scan(&version, &revision); err != nil {
			t.Fatal(err)
		}
		if m.V3.Capacidades.Empleados.Version != uint64(version) || m.V3.Capacidades.Empleados.RevisionGobierno != uint64(revision) {
			t.Fatal("versión o revisión no tomadas del gobierno publicado")
		}
		if contar() != antes {
			t.Fatal("la herramienta modificó el gobierno")
		}
	})
	t.Run("transaccion_solo_lectura", func(t *testing.T) {
		g, err := abrirGobiernoPostgreSQL(ctx, gobiernoDSN)
		if err != nil {
			t.Fatal("no abrió la instantánea")
		}
		defer g.cerrar()
		_, err = g.(*gobiernoPostgreSQL).tx.Exec(ctx, `INSERT INTO vec_autorizacion_atestada_v3.revocacion_clave_capacidad
		  SELECT clave_id, version, clock_timestamp(), 'motivo:prueba', 'acto:prueba:t4', clock_timestamp()
		    FROM vec_autorizacion_atestada_v3.clave_capacidad_version LIMIT 1`)
		if err == nil || !strings.Contains(err.Error(), "25006") {
			t.Fatal("la transacción admitió una escritura")
		}
	})
	// Identidad como vec-server: LOGIN sin el grupo de gobierno, superusuario,
	// con BYPASSRLS, con CREATEROLE, con una membresía de más o NOINHERIT.
	for _, caso := range []string{"SIN_ROL", "SUPERUSUARIO", "BYPASSRLS", "CREATEROLE", "MEMBRESIA_EXTRA", "NOINHERIT"} {
		t.Run("identidad_"+strings.ToLower(caso), func(t *testing.T) {
			e2 := escenarioPublicado(t)
			if codigo, texto := ejecutarCon(e2, os.Getenv("VEC_T4_PG_"+caso+"_DSN")); codigo != 1 || !strings.Contains(texto, string(errIdentidad)) {
				t.Fatalf("aceptó un LOGIN no admitido: %s", texto)
			}
			e2.sinResiduos(t)
		})
	}
	t.Run("material_divergente", func(t *testing.T) {
		e2 := nuevoEscenario(t) // material sintético propio, nunca publicado
		if codigo, texto := ejecutarCon(e2, gobiernoDSN); codigo != 1 || !strings.Contains(texto, string(errClaveB2)) {
			t.Fatalf("material no publicado aceptado: %s", texto)
		}
		e2.sinResiduos(t)
	})
	t.Run("revocacion_programada_de_una_clave_b2", func(t *testing.T) {
		if _, err := admin.Exec(ctx, `INSERT INTO vec_autorizacion_atestada_v3.revocacion_clave_capacidad
		  (clave_id, version, revocada_en, motivo_catalogado_ref, acto_ref)
		  SELECT clave_id, version, clock_timestamp() + interval '1 day', 'motivo:prueba:t4', 'acto:prueba:t4:revocacion'
		    FROM vec_autorizacion_atestada_v3.clave_capacidad_version
		   WHERE audiencia_consumo = 'vec_personal.registro_empleado.hecho.v1'`); err != nil {
			t.Fatal(err)
		}
		e2 := escenarioPublicado(t)
		if codigo, texto := ejecutarCon(e2, gobiernoDSN); codigo != 1 || !strings.Contains(texto, string(errClaveB2)) {
			t.Fatalf("clave revocada aceptada: %s", texto)
		}
		e2.sinResiduos(t)
	})
	t.Run("puntero_rotado_a_otra_clave", func(t *testing.T) {
		// La historia es de solo adición: se comprueba sobre la audiencia ficha,
		// que el caso anterior no revocó.
		const idRotada = "clave:capacidad:personal-b2-ficha:rotada"
		if _, err := admin.Exec(ctx, `WITH s AS (SELECT gen_random_bytes(32) AS b),
		       o AS (SELECT max(orden) + 1 AS n FROM vec_autorizacion_atestada_v3.puntero_clave_emision)
		  INSERT INTO vec_autorizacion_atestada_v3.clave_capacidad_version
		   (clave_id, version, revision_gobierno, huella_gobierno_sha256, secreto_hmac, huella_secreto_sha256,
		    emisor_id, audiencia_consumo, valida_desde, valida_hasta, acto_ref)
		  SELECT $4, o.n, o.n, repeat('4', 64), s.b, encode(sha256(s.b), 'hex'),
		         $1, 'vec_personal.registro_empleado.ficha.v1', $2, $3, 'acto:ct:desarrollo:clave-capacidad:t4:rotada' FROM s, o`,
			emisorPrueba, desdePrueba, hastaPrueba, idRotada); err != nil {
			t.Fatal(err)
		}
		if _, err := admin.Exec(ctx, `INSERT INTO vec_autorizacion_atestada_v3.puntero_clave_emision (orden, clave_id, version, establecida_en, acto_ref)
		  SELECT version, clave_id, version, clock_timestamp() - interval '1 second', 'acto:ct:desarrollo:puntero-clave:t4:rotada'
		    FROM vec_autorizacion_atestada_v3.clave_capacidad_version WHERE clave_id = $1`, idRotada); err != nil {
			t.Fatal(err)
		}
		g, err := abrirGobiernoPostgreSQL(ctx, gobiernoDSN)
		if err != nil {
			t.Fatal("no abrió la instantánea")
		}
		defer g.cerrar()
		var huella string
		if err := admin.QueryRow(ctx, `SELECT huella_secreto_sha256 FROM vec_autorizacion_atestada_v3.clave_capacidad_version
		  WHERE audiencia_consumo = 'vec_personal.registro_empleado.ficha.v1' AND clave_id <> $1`, idRotada).Scan(&huella); err != nil {
			t.Fatal(err)
		}
		f, err := g.clavePorHuellaSecreto(ctx, huella)
		if err != nil || f.PunteroVigente || !f.Vigente || f.Revocada {
			t.Fatal("la consulta no detecta el puntero rotado")
		}
	})
}
