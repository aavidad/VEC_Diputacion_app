package bootstrap

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Lectura por operación del gobierno de atestación CT en PostgreSQL real
// (mismo contenedor desechable que TestConfianzaRenovableCTPostgreSQL, con el
// gobierno vacío). Comprueba que la lectura de solo lectura:
//   - devuelve la publicación vigente y respeta una revocación inmediata;
//   - no espera a una consulta que tiene bloqueado el checkpoint (como hace
//     cada consumo de autorización V3 hasta su COMMIT), a diferencia de la
//     transacción de publicación, que sí espera.
func TestLecturaGobiernoConfianzaCTPostgreSQL(t *testing.T) {
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
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	m := materialRenovableCTPrueba(t, ahora)
	if err := publicarGobiernoAtestacionContratacionTemporalDesarrollo(ctx, pool, &m); err != nil {
		t.Fatalf("publicación inicial: %v", err)
	}
	leerNueva := func(c context.Context) (materialAtestacionContratacionTemporalDesarrollo, error) {
		var leido materialAtestacionContratacionTemporalDesarrollo
		err := ejecutarLecturaGobiernoCTDesarrollo(c, pool, func(tx pgx.Tx) error {
			var e error
			leido, e = leerConfiguracionVigenteCTDesarrollo(c, tx, m, time.Now().UTC())
			return e
		})
		return leido, err
	}
	leerAnterior := func(c context.Context) error {
		return ejecutarTransaccionGobiernoCTDesarrollo(c, pool, func(tx pgx.Tx) error {
			_, e := leerConfiguracionRenovableCTDesarrollo(c, tx, m, time.Now().UTC())
			return e
		})
	}
	leido, err := leerNueva(ctx)
	if err != nil || leido.configuracionRef != m.configuracionRef || leido.configuracionHuella != m.configuracionHuella {
		t.Fatalf("lectura vigente: %v", err)
	}
	const vueltas = 200
	medir := func(leer func(context.Context) error) time.Duration {
		inicio := time.Now()
		for range vueltas {
			if err := leer(ctx); err != nil {
				t.Fatal(err)
			}
		}
		return time.Since(inicio) / vueltas
	}
	porLecturaAnterior := medir(leerAnterior)
	porLecturaNueva := medir(func(c context.Context) error { _, e := leerNueva(c); return e })
	t.Logf("por lectura: publicación %v, solo lectura %v", porLecturaAnterior, porLecturaNueva)

	// Una consulta CT en curso retiene el checkpoint hasta su COMMIT.
	bloqueo, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bloqueo.Exec(ctx, `SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario`); err != nil {
		t.Fatal(err)
	}
	if _, err := bloqueo.Exec(ctx, `SELECT 1 FROM vec_autorizacion_atestada_v3.checkpoint_gobierno WHERE control_id FOR UPDATE`); err != nil {
		t.Fatal(err)
	}
	corto, cancelarCorto := context.WithTimeout(ctx, 500*time.Millisecond)
	if _, err := leerNueva(corto); err != nil {
		t.Fatalf("la lectura esperó al checkpoint bloqueado: %v", err)
	}
	if err := leerAnterior(corto); err == nil {
		t.Fatal("la transacción de publicación no esperó al checkpoint bloqueado")
	}
	cancelarCorto()
	_ = bloqueo.Rollback(context.Background())

	// Revocación inmediata: la siguiente lectura ya no confirma la publicación.
	if _, err := pool.Exec(ctx, `BEGIN; SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
 INSERT INTO vec_autorizacion_atestada_v3.revocacion_configuracion VALUES ('`+m.configuracionRef+`', clock_timestamp(), 'motivo:prueba', 'acto:prueba:revocacion', clock_timestamp()); COMMIT`); err != nil {
		t.Fatal(err)
	}
	if _, err := leerNueva(ctx); err == nil {
		t.Fatal("la lectura confirmó una configuración revocada")
	}
}
