package bootstrap

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"sync"
	"testing"
	"time"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
)

// Sólo el contenedor desechable preparado con roles y migraciones AD3 reales.
// No crea bases, esquemas, roles, funciones ni sustitutos de las migraciones.
func TestConfianzaRenovableCTPostgreSQL(t *testing.T) {
	if os.Getenv("VEC_R37_PG_DESECHABLE") != "1" {
		t.Skip("requiere PostgreSQL desechable AD3-1/2 real")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancelar()
	pool, err := pgxpool.New(ctx, os.Getenv("VEC_R37_PG_DSN"))
	if err != nil {
		t.Fatal("pool de prueba no disponible")
	}
	defer pool.Close()
	var vacia bool
	if err := pool.QueryRow(ctx, `SELECT current_setting('server_version_num')::int / 10000=18 AND
 (SELECT count(*) FROM vec_autorizacion_atestada_v3.configuracion_confianza_version)=0 AND
 (SELECT count(*) FROM vec_autorizacion_atestada_v3.raiz_confianza_version)=0 AND
 (SELECT count(*) FROM vec_autorizacion_atestada_v3.clave_capacidad_version)=0`).Scan(&vacia); err != nil || !vacia {
		t.Fatal("requiere gobierno vacío en contenedor desechable PostgreSQL 18")
	}
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	anterior := materialRenovableCTPrueba(t, ahora.Add(-24*time.Hour))
	if err := publicarGobiernoAtestacionContratacionTemporalDesarrollo(ctx, pool, &anterior); err != nil {
		t.Fatalf("publicación inicial: %v", err)
	}
	contar := func() [3]int64 {
		t.Helper()
		var n [3]int64
		if err := pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM vec_autorizacion_atestada_v3.configuracion_confianza_version),
 (SELECT count(*) FROM vec_autorizacion_atestada_v3.raiz_confianza_version),
 (SELECT count(*) FROM vec_autorizacion_atestada_v3.clave_capacidad_version)`).Scan(&n[0], &n[1], &n[2]); err != nil {
			t.Fatal(err)
		}
		return n
	}
	if n := contar(); n != [3]int64{1, 1, 1} {
		t.Fatalf("preimagen: %v", n)
	}
	otra := materialRenovableCTPrueba(t, ahora)
	casos := []struct {
		nombre, sql string
		args        []any
	}{
		{"revision_revocada", `INSERT INTO vec_autorizacion_atestada_v3.revocacion_configuracion VALUES ($1,$2,'motivo:prueba','acto:prueba:revocacion',clock_timestamp())`, []any{anterior.configuracionRef, ahora}},
		{"revocacion_programada", `INSERT INTO vec_autorizacion_atestada_v3.revocacion_configuracion VALUES ($1,$2,'motivo:prueba','acto:prueba:revocacion',clock_timestamp())`, []any{anterior.configuracionRef, ahora.Add(time.Hour)}},
		{"raiz_revocada", `INSERT INTO vec_autorizacion_atestada_v3.revocacion_raiz VALUES ($1,$2,$3,'motivo:prueba','acto:prueba:revocacion',clock_timestamp())`, []any{anterior.claveID, anterior.claveVersion, ahora}},
		{"checkpoint_incompatible", `UPDATE vec_autorizacion_atestada_v3.checkpoint_gobierno SET configuracion_secuencia_minima=$1 WHERE control_id`, []any{anterior.configuracionOrden + 1}},
		{"conjunto_de_raices_ampliado", `INSERT INTO vec_autorizacion_atestada_v3.raiz_confianza_version
 (clave_id,version,clave_publica_spki,huella_spki_sha256,valida_desde,valida_hasta,suite,audiencia_despliegue,acto_ref)
 VALUES ($1,1,$2,$3,$4,$5,$6,$7,'acto:ct:desarrollo:raiz-atestacion:otra');`, []any{otra.claveID + ":otra", otra.spki, otra.spkiHuella, otra.validaDesde, otra.validaHasta, confianza.SuiteAtestacionAutorizacionV3COSEEdDSA, audienciaAtestacionContratacionTemporalDesarrollo}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback(context.Background())
			if _, err := tx.Exec(ctx, `SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario`); err != nil {
				t.Fatal(err)
			}
			if _, err := tx.Exec(ctx, caso.sql, caso.args...); err != nil {
				t.Fatal(err)
			}
			if caso.nombre == "conjunto_de_raices_ampliado" {
				if _, err := tx.Exec(ctx, `INSERT INTO vec_autorizacion_atestada_v3.configuracion_raiz VALUES($1,$2,1)`, anterior.configuracionRef, otra.claveID+":otra"); err != nil {
					t.Fatal(err)
				}
			}
			siguiente := anterior
			if err := renovarConfiguracionConfianzaCTEnTxDesarrollo(ctx, tx, anterior, ahora, &siguiente); err == nil {
				t.Fatal("renovación aceptó gobierno retirado/incompatible")
			}
			var n int
			if err := tx.QueryRow(ctx, `SELECT count(*) FROM vec_autorizacion_atestada_v3.configuracion_confianza_version`).Scan(&n); err != nil || n != 1 {
				t.Fatal("renovación rechazada dejó publicación")
			}
		})
	}
	rollback := errors.New("rollback controlado")
	if err := ejecutarTransaccionGobiernoCTDesarrollo(ctx, pool, func(tx pgx.Tx) error {
		siguiente := anterior
		if err := renovarConfiguracionConfianzaCTEnTxDesarrollo(ctx, tx, anterior, ahora, &siguiente); err != nil {
			return err
		}
		return rollback
	}); !errors.Is(err, rollback) {
		t.Fatalf("falló ensayo rollback: %v", err)
	}
	if n := contar(); n != [3]int64{1, 1, 1} {
		t.Fatalf("rollback alteró historia: %v", n)
	}
	var wg sync.WaitGroup
	resultados := make([]materialAtestacionContratacionTemporalDesarrollo, 2)
	errores := make([]error, 2)
	inicio := make(chan struct{})
	for i := range 2 {
		wg.Go(func() {
			<-inicio
			resultados[i], errores[i] = renovarConfiguracionConfianzaCTDesarrollo(ctx, pool, anterior, ahora)
		})
	}
	close(inicio)
	wg.Wait()
	for _, err := range errores {
		if err != nil {
			t.Fatalf("renovación/adopción concurrente: %v", err)
		}
	}
	if resultados[0].configuracionRef != resultados[1].configuracionRef || resultados[0].configuracionHuella != resultados[1].configuracionHuella || resultados[0].configuracionOrden <= anterior.configuracionOrden {
		t.Fatal("instancias no adoptaron revisión idéntica")
	}
	if n := contar(); n != [3]int64{2, 1, 1} {
		t.Fatalf("renovación duplicó configuración o cambió claves: %v", n)
	}
	// Una instancia cuyo COMMIT no confirmó recibe la revisión exacta al releer.
	repetida, err := renovarConfiguracionConfianzaCTDesarrollo(ctx, pool, anterior, ahora)
	if err != nil || repetida.configuracionHuella != resultados[0].configuracionHuella || contar() != [3]int64{2, 1, 1} {
		t.Fatal("reintento no adoptó publicación durable")
	}
	t.Run("no_revive_anterior_revocada_tras_adopcion", func(t *testing.T) {
		tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(context.Background())
		if _, err := tx.Exec(ctx, `SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario`); err != nil {
			t.Fatal(err)
		}
		if _, err := tx.Exec(ctx, casos[0].sql, casos[0].args...); err != nil {
			t.Fatal(err)
		}
		siguiente := anterior
		if err := renovarConfiguracionConfianzaCTEnTxDesarrollo(ctx, tx, anterior, ahora, &siguiente); err == nil {
			t.Fatal("adoptó pese a revocación explícita anterior")
		}
	})
}
