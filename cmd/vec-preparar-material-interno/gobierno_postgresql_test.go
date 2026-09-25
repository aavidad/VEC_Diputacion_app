package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"vec-diputacion-granada/internal/app/bootstrap"
	"vec-diputacion-granada/internal/app/composicion/internactproveedores"
)

// Solo contra el PostgreSQL 18.4 desechable que prepara
// probar_postgresql_pg18.sh (cadena AD3 1/2 real, gobierno vacío). El
// fixture de gobierno se inserta como DBA con secretos sintéticos derivados
// por la única derivación de T3; la herramienta lee con un LOGIN que solo
// puede asumir el rol propietario AD3, dentro de READ ONLY.
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
	var vacio bool
	if err := admin.QueryRow(ctx, `SELECT current_setting('server_version_num') = '180004'
	   AND (SELECT count(*) FROM vec_autorizacion_atestada_v3.clave_capacidad_version) = 0
	   AND (SELECT count(*) FROM vec_autorizacion_atestada_v3.raiz_confianza_version) = 0`).Scan(&vacio); err != nil || !vacio {
		t.Fatal("se exige PostgreSQL 18.4 desechable con gobierno vacío")
	}
	e := nuevoEscenario(t)
	publicarFixture(ctx, t, admin, e.base)
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
	lector := os.Getenv("VEC_T4_PG_LECTOR_DSN")
	d := dependencias{abrirGobierno: abrirGobiernoPostgreSQL, reloj: time.Now}
	ejecutarCon := func(e *escenario, dsn string) (int, string) {
		var out, errOut bytes.Buffer
		sinArchivo := e.args()[:len(e.args())-2]
		codigo := ejecutar(ctx, sinArchivo, dsn, true, &out, &errOut, d)
		if strings.Contains(out.String()+errOut.String(), "postgres") {
			t.Fatal("la salida menciona el DSN")
		}
		return codigo, out.String() + errOut.String()
	}

	t.Run("positivo_con_cargadores_reales", func(t *testing.T) {
		if codigo, texto := ejecutarCon(e, lector); codigo != 0 {
			t.Fatalf("código %d: %s", codigo, texto)
		}
		m, err := internactproveedores.CargarMaterialPersonalB2(e.salida)
		if err != nil {
			t.Fatal("cargador real rechazó el material")
		}
		defer m.Cerrar()
		if m.V3.Capacidades.Ficha.Version != 11 || m.V3.Capacidades.Empleados.RevisionGobierno != 18 {
			t.Fatal("versión o revisión no tomadas del gobierno publicado")
		}
		if contar() != antes {
			t.Fatal("la herramienta modificó el gobierno")
		}
	})
	t.Run("transaccion_solo_lectura", func(t *testing.T) {
		g, err := abrirGobiernoPostgreSQL(ctx, lector)
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
	t.Run("login_sin_rol_propietario", func(t *testing.T) {
		e2 := nuevoEscenario(t)
		e2.base, e2.claveBase = e.base, e.claveBase
		if codigo, texto := ejecutarCon(e2, os.Getenv("VEC_T4_PG_SIN_ROL_DSN")); codigo != 1 || !strings.Contains(texto, string(errGobiernoConexion)) {
			t.Fatalf("aceptó un LOGIN sin lectura: %s", texto)
		}
		e2.sinResiduos(t)
	})
	t.Run("clave_base_divergente", func(t *testing.T) {
		e2 := nuevoEscenario(t)
		if codigo, texto := ejecutarCon(e2, lector); codigo != 1 || !strings.Contains(texto, string(errClaveBaseNoPub)) {
			t.Fatalf("clave base no publicada aceptada: %s", texto)
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
		e2 := nuevoEscenario(t)
		e2.base, e2.claveBase = e.base, e.claveBase
		if codigo, texto := ejecutarCon(e2, lector); codigo != 1 || !strings.Contains(texto, string(errClaveB2)) {
			t.Fatalf("clave revocada aceptada: %s", texto)
		}
		e2.sinResiduos(t)
	})
	t.Run("puntero_rotado_a_otra_clave", func(t *testing.T) {
		// La historia es de solo adición: se comprueba sobre la audiencia ficha,
		// que el caso anterior no revocó.
		if _, err := admin.Exec(ctx, `WITH s AS (SELECT gen_random_bytes(32) AS b)
		  INSERT INTO vec_autorizacion_atestada_v3.clave_capacidad_version
		   (clave_id, version, revision_gobierno, huella_gobierno_sha256, secreto_hmac, huella_secreto_sha256,
		    emisor_id, audiencia_consumo, valida_desde, valida_hasta, acto_ref)
		  SELECT 'clave:capacidad:personal-b2-ficha:rotada', 30, 30, repeat('4', 64), s.b, encode(sha256(s.b), 'hex'),
		         $1, 'vec_personal.registro_empleado.ficha.v1', $2, $3, 'acto:ct:desarrollo:clave-capacidad:t4:rotada' FROM s`,
			emisorPrueba, desdePrueba, hastaPrueba); err != nil {
			t.Fatal(err)
		}
		if _, err := admin.Exec(ctx, `INSERT INTO vec_autorizacion_atestada_v3.puntero_clave_emision (orden, clave_id, version, establecida_en, acto_ref)
		  VALUES (30, 'clave:capacidad:personal-b2-ficha:rotada', 30, clock_timestamp() - interval '1 second', 'acto:ct:desarrollo:puntero-clave:t4:rotada')`); err != nil {
			t.Fatal(err)
		}
		g, err := abrirGobiernoPostgreSQL(ctx, lector)
		if err != nil {
			t.Fatal("no abrió la instantánea")
		}
		defer g.cerrar()
		f, err := g.clavePorHuellaSecreto(ctx, e.huellasB2[0])
		if err != nil || f.PunteroVigente || !f.Vigente || f.Revocada {
			t.Fatal("la consulta no detecta el puntero rotado")
		}
	})
}

// publicarFixture inserta como DBA la clave base CT, las ocho claves B2
// derivadas (versión = revisión = orden = 11..18), una raíz Ed25519
// sintética y una configuración vigente que apunta a ella.
func publicarFixture(ctx context.Context, t *testing.T, admin *pgx.Conn, base []byte) {
	t.Helper()
	tx, err := admin.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	insertar := func(id string, n int, huellaGob string, secreto []byte, huellaSec, audiencia string) {
		t.Helper()
		if _, err := tx.Exec(ctx, `INSERT INTO vec_autorizacion_atestada_v3.clave_capacidad_version
		  (clave_id, version, revision_gobierno, huella_gobierno_sha256, secreto_hmac, huella_secreto_sha256,
		   emisor_id, audiencia_consumo, valida_desde, valida_hasta, acto_ref)
		  VALUES ($1,$2::numeric,$2::numeric,$3,$4,$5,$6,$7,$8,$9,'acto:ct:desarrollo:clave-capacidad:t4:'||$2::text)`,
			id, n, huellaGob, secreto, huellaSec, emisorPrueba, audiencia, desdePrueba, hastaPrueba); err != nil {
			t.Fatalf("clave %d: %v", n, err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO vec_autorizacion_atestada_v3.puntero_clave_emision (orden, clave_id, version, establecida_en, acto_ref)
		  VALUES ($1::numeric,$2,$1::numeric,clock_timestamp() - interval '1 minute','acto:ct:desarrollo:puntero-clave:t4:'||$1::text)`, n, id); err != nil {
			t.Fatalf("puntero %d: %v", n, err)
		}
	}
	insertar(baseIDPrueba, 1, strings.Repeat("9", 64), base, huellaHex(base), audienciaClaveBaseCT)
	claves, err := bootstrap.DerivarClavesPersonalB2V3Desarrollo(base, baseIDPrueba, emisorPrueba, desdePrueba, hastaPrueba)
	if err != nil {
		t.Fatal(err)
	}
	for i := range claves {
		s := claves[i].CopiarSecreto()
		insertar(claves[i].ClaveID, 11+i, claves[i].HuellaGobierno, s, claves[i].SHA256, claves[i].Audiencia)
		clear(s)
		claves[i].Borrar()
	}
	publica, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	spki, err := x509.MarshalPKIXPublicKey(publica)
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO vec_autorizacion_atestada_v3.raiz_confianza_version
		   (clave_id, version, clave_publica_spki, huella_spki_sha256, valida_desde, valida_hasta, suite, audiencia_despliegue, acto_ref)
		  VALUES ($1, 1, $2, encode(sha256($2), 'hex'), $3, $4, 'VEC-AD-3-COSE-EDDSA-1', $5, 'acto:ct:desarrollo:raiz-atestacion:t4')`,
			[]any{raizPrueba, spki, desdePrueba, hastaPrueba, audienciaCT}},
		{`INSERT INTO vec_autorizacion_atestada_v3.configuracion_confianza_version
		   (revision, secuencia, huella_configuracion_sha256, publicada_en, expira_en, acto_ref)
		  VALUES ('confianza:t4', 1, repeat('3', 64), clock_timestamp() - interval '1 hour', clock_timestamp() + interval '1 hour', 'acto:ct:desarrollo:configuracion:t4')`, nil},
		{`INSERT INTO vec_autorizacion_atestada_v3.configuracion_raiz VALUES ('confianza:t4', $1, 1)`, []any{raizPrueba}},
		{`INSERT INTO vec_autorizacion_atestada_v3.puntero_configuracion_actual (orden, configuracion_revision, establecida_en, acto_ref)
		  VALUES (1, 'confianza:t4', clock_timestamp() - interval '1 minute', 'acto:ct:desarrollo:puntero-configuracion:t4')`, nil},
		{`UPDATE vec_autorizacion_atestada_v3.checkpoint_gobierno SET revision = 18 WHERE control_id`, nil},
	} {
		if _, err := tx.Exec(ctx, q.sql, q.args...); err != nil {
			t.Fatalf("fixture: %v", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
}
