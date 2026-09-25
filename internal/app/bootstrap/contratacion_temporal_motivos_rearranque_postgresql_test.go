package bootstrap

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// El contenido publicado depende solo de las entradas y del instante: la
// misma entrada produce los mismos bytes, que es lo que exige el replay SQL.
func TestContenidoCatalogoMotivosDeterminista(t *testing.T) {
	motivos := []dominiovec.ReferenciaEntradaCatalogo{motivoPortalMiBolsaDesarrollo()}
	instante := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	primero, err := contenidoCatalogoMotivosPostgreSQLContratacionTemporalDesarrollo(motivos, instante)
	if err != nil {
		t.Fatal(err)
	}
	// El mismo instante en otra zona horaria (como lo devuelve pgx) no cambia
	// los bytes: la vigencia se expresa siempre en UTC.
	segundo, err := contenidoCatalogoMotivosPostgreSQLContratacionTemporalDesarrollo(motivos, instante.In(time.FixedZone("CEST", 2*3600)))
	if err != nil || !bytes.Equal(primero, segundo) {
		t.Fatalf("contenido no determinista: %s / %s", primero, segundo)
	}
}

// Dos arranques seguidos con el portal del candidato publican el mismo
// catálogo de motivos. Reproduce el fallo del rearranque: el primero lo
// publicó con el reloj y el segundo, con otro instante, chocaba con el
// replay (etapa=publicacion_ejecucion). Ahora la misma huella se reutiliza.
// Solo contra una base desechable con AD3-3 real (volcado sintético).
func TestPublicacionCatalogoMotivosRearranquePostgreSQL(t *testing.T) {
	if os.Getenv("VEC_MOTIVOS_REARRANQUE_PG_DESECHABLE") != "1" {
		t.Skip("requiere PostgreSQL 18 desechable con el volcado sintético")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancelar()
	pool, err := pgxpool.New(ctx, os.Getenv("VEC_MOTIVOS_REARRANQUE_PG_DSN"))
	if err != nil {
		t.Fatal("pool de prueba no disponible")
	}
	defer pool.Close()

	var aleatorio [8]byte
	if _, err := rand.Read(aleatorio[:]); err != nil {
		t.Fatal(err)
	}
	sufijo := hex.EncodeToString(aleatorio[:])
	motivo := motivoPortalMiBolsaDesarrollo()
	motivo.CatalogoID = "vec.prueba.rearranque." + sufijo
	motivos := []dominiovec.ReferenciaEntradaCatalogo{motivo}

	contar := func() (int64, int64, time.Time) {
		t.Helper()
		var cabeceras, eventos int64
		var publicado time.Time
		if err := pool.QueryRow(ctx, `SELECT
 (SELECT count(*) FROM vec_autorizacion.motivo_v2_catalogo_publicado WHERE catalogo_id=$1),
 (SELECT count(*) FROM vec_autorizacion.motivo_v2_evento_origen WHERE catalogo_id=$1),
 (SELECT publicado_en FROM vec_autorizacion.motivo_v2_catalogo_publicado WHERE catalogo_id=$1)`,
			motivo.CatalogoID).Scan(&cabeceras, &eventos, &publicado); err != nil {
			t.Fatal(err)
		}
		return cabeceras, eventos, publicado
	}

	// Primer arranque antiguo: instante de reloj, variable.
	primerArranque := time.Now().UTC().Add(-time.Hour).Truncate(time.Microsecond)
	if err := publicarCatalogoMotivosPostgreSQLContratacionTemporalDesarrollo(ctx, pool, motivos, primerArranque); err != nil {
		t.Fatalf("primer arranque: %v", err)
	}
	// Arranques siguientes: el mismo instante y otro distinto (el reloj de
	// cada arranque, o el instante fijo tras la corrección).
	for i, instante := range []time.Time{primerArranque, primerArranque.Add(37 * time.Minute), primerArranque.Add(-24 * time.Hour)} {
		if err := publicarCatalogoMotivosPostgreSQLContratacionTemporalDesarrollo(ctx, pool, motivos, instante); err != nil {
			t.Fatalf("rearranque %d: %v", i+1, err)
		}
	}
	cabeceras, eventos, publicado := contar()
	if cabeceras != 1 || eventos != 1 || !publicado.Equal(primerArranque) {
		t.Fatalf("publicación duplicada o alterada: cabeceras=%d eventos=%d publicado=%s", cabeceras, eventos, publicado)
	}

	// Otra huella para la misma versión sigue rechazándose sin efectos.
	distinto := motivo
	distinto.CatalogoHuellaSHA256 = huellaAltaContratacionTemporalDesarrollo("otra-huella-" + sufijo)
	if err := publicarCatalogoMotivosPostgreSQLContratacionTemporalDesarrollo(ctx, pool, []dominiovec.ReferenciaEntradaCatalogo{distinto}, primerArranque); err == nil {
		t.Fatal("otra huella para la misma versión debía rechazarse")
	}
	// Otras entradas con la misma huella también: el replay compara contenido.
	otraEntrada := motivo
	otraEntrada.EntradaClave = referenciaAltaContratacionTemporalDesarrollo("motivo_", "otra-entrada-"+sufijo)
	if err := publicarCatalogoMotivosPostgreSQLContratacionTemporalDesarrollo(ctx, pool, []dominiovec.ReferenciaEntradaCatalogo{otraEntrada}, primerArranque); err == nil {
		t.Fatal("otras entradas con la misma versión debían rechazarse")
	}
	if c, e, p := contar(); c != 1 || e != 1 || !p.Equal(primerArranque) {
		t.Fatalf("un rechazo dejó efectos: cabeceras=%d eventos=%d publicado=%s", c, e, p)
	}
}
