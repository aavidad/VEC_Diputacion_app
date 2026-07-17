package confianzaatestacionv2

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const variableDSNIntegracionConfianzaAtestacionV2 = "VEC_CONFIANZA_ATESTACION_V2_POSTGRES_DSN"

// TestIntegracionPostgreSQLCargaConfianzaV2Real consume exclusivamente el
// catalogo que el runner SQL haya sembrado. No instala datos, no usa memoria y
// no dispone de una segunda fuente a la que recurrir si PostgreSQL falla.
func TestIntegracionPostgreSQLCargaConfianzaV2Real(t *testing.T) {
	dsn := os.Getenv(variableDSNIntegracionConfianzaAtestacionV2)
	if strings.TrimSpace(dsn) == "" {
		t.Skip("prueba PostgreSQL de confianza V2 omitida: falta " +
			variableDSNIntegracionConfianzaAtestacionV2)
	}

	ctx, cancelar := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelar()
	configuracionPool, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal("configurar pool PostgreSQL aislado para confianza V2")
	}
	configuracionPool.MaxConns = 1
	configuracionPool.MinConns = 0
	pool, err := pgxpool.NewWithConfig(ctx, configuracionPool)
	if err != nil {
		t.Fatal("crear pool PostgreSQL aislado para confianza V2")
	}
	defer pool.Close()
	if err = pool.Ping(ctx); err != nil {
		t.Fatal("conectar con PostgreSQL para confianza V2")
	}

	configuracion, err := CargarConfiguracionActual(ctx, pool)
	if err != nil {
		t.Fatal("el cargador productivo rechazo el catalogo PostgreSQL V2")
	}
	huellaEsperada, err := leerHuellaActualComoOraculoIntegracion(ctx, pool)
	if err != nil {
		t.Fatal("no se pudo contrastar la huella PostgreSQL V2")
	}
	if err = configuracion.ValidarHuellaSHA256Esperada(huellaEsperada); err != nil {
		t.Fatal("la configuracion cargada no conserva la huella autoritativa")
	}

	servicio, err := NuevoServicioActual(ctx, pool)
	if err != nil || servicio == nil {
		t.Fatal("el constructor productivo no creo el servicio de confianza V2")
	}
}

// leerHuellaActualComoOraculoIntegracion no alimenta al cargador: solo
// contrasta despues la configuracion ya construida por la ruta productiva.
func leerHuellaActualComoOraculoIntegracion(
	ctx context.Context,
	pool *pgxpool.Pool,
) (string, error) {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel:   pgx.Serializable,
		AccessMode: pgx.ReadWrite,
	})
	if err != nil {
		return "", err
	}
	defer func() {
		ctxRollback, cancelar := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancelar()
		_ = tx.Rollback(ctxRollback)
	}()
	if _, err = tx.Exec(ctx, `SET LOCAL ROLE `+RolLectorAutoridadPostgreSQL); err != nil {
		return "", err
	}
	var huella string
	if err = tx.QueryRow(ctx, `
		SELECT huella_configuracion_sha256
		  FROM vec_confianza_atestacion_v2.obtener_confianza_actual()
		 LIMIT 1`).Scan(&huella); err != nil {
		return "", err
	}
	if err = tx.Commit(ctx); err != nil {
		return "", err
	}
	return huella, nil
}
