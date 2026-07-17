package postgres

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const (
	variableDSNPostgreSQLPruebasMotivosEvaluador = "VEC_POSTGRES_TEST_MOTIVOS_EVALUADOR_DSN"
	variableDSNPostgreSQLPruebasMotivosProyector = "VEC_POSTGRES_TEST_MOTIVOS_PROYECTOR_DSN"
	variableDSNPostgreSQLPruebasMotivosFuenteV1  = "VEC_POSTGRES_TEST_MOTIVOS_FUENTE_V1_DSN"
)

func TestIntegracionMotivosAutorizacionV2PostgreSQL(t *testing.T) {
	dsnEvaluador := os.Getenv(variableDSNPostgreSQLPruebasMotivosEvaluador)
	dsnProyector := os.Getenv(variableDSNPostgreSQLPruebasMotivosProyector)
	dsnFuenteV1 := os.Getenv(variableDSNPostgreSQLPruebasMotivosFuenteV1)
	if dsnEvaluador == "" || dsnProyector == "" || dsnFuenteV1 == "" {
		t.Skipf(
			"prueba PostgreSQL de motivos V2 omitida: defina %s, %s y %s o ejecute deploy/postgresql/autorizacion/probar_integracion.sh",
			variableDSNPostgreSQLPruebasMotivosEvaluador,
			variableDSNPostgreSQLPruebasMotivosProyector,
			variableDSNPostgreSQLPruebasMotivosFuenteV1,
		)
	}

	ctx, cancelar := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelar()
	poolEvaluador := abrirPoolMotivoAutorizacionV2PostgreSQLIntegracion(t, ctx, dsnEvaluador)
	defer poolEvaluador.Close()
	poolProyector := abrirPoolMotivoAutorizacionV2PostgreSQLIntegracion(t, ctx, dsnProyector)
	defer poolProyector.Close()
	poolFuenteV1 := abrirPoolMotivoAutorizacionV2PostgreSQLIntegracion(t, ctx, dsnFuenteV1)
	defer poolFuenteV1.Close()
	verificarIdentidadesMotivoAutorizacionV2PostgreSQLSeparadas(
		t, ctx, poolEvaluador, poolProyector, poolFuenteV1,
	)

	const catalogoID = "motivos_autorizacion"
	referencia := domain.ReferenciaEntradaCatalogo{
		CatalogoID:           catalogoID,
		CatalogoVersion:      1,
		CatalogoHuellaSHA256: strings.Repeat("b", 64),
		EntradaClave:         "motivo_cccccccccccccccccccccccccccccccc",
	}
	instante := time.Date(2026, time.January, 15, 0, 0, 0, 0, time.UTC)

	validador, err := NuevoValidadorReferenciaMotivoPostgreSQLV2(poolEvaluador, catalogoID)
	if err != nil {
		t.Fatal("crear adaptador historico con identidad evaluadora")
	}
	if err = validador.ValidarReferenciaMotivoAutorizacionV2(
		ctx, referencia, instante,
	); err != nil {
		t.Fatal("la identidad evaluadora no resolvio la referencia historica")
	}

	t.Run("coordenadas historicas negativas no son averias", func(t *testing.T) {
		casos := []struct {
			nombre     string
			referencia domain.ReferenciaEntradaCatalogo
			instante   time.Time
		}{
			{"huella_distinta", func() domain.ReferenciaEntradaCatalogo {
				r := referencia
				r.CatalogoHuellaSHA256 = strings.Repeat("d", 64)
				return r
			}(), instante},
			{"clave_inexistente", func() domain.ReferenciaEntradaCatalogo {
				r := referencia
				r.EntradaClave = "motivo_eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
				return r
			}(), instante},
			{"antes_de_publicacion", referencia, time.Date(2025, time.December, 31, 23, 59, 59, 999_999_000, time.UTC)},
		}
		for _, caso := range casos {
			caso := caso
			t.Run(caso.nombre, func(t *testing.T) {
				err := validador.ValidarReferenciaMotivoAutorizacionV2(
					ctx, caso.referencia, caso.instante,
				)
				if !errors.Is(err, domain.ErrSolicitudAutorizacionInvalida) {
					t.Fatal("una coordenada historica negativa no fallo cerrada")
				}
				if errors.Is(err, ports.ErrFuenteAutorizacionNoDisponible) {
					t.Fatal("una respuesta historica negativa se clasifico como averia")
				}
			})
		}
	})

	t.Run("identidades proyectora y fuente V1 no pueden evaluar", func(t *testing.T) {
		for _, poolAjeno := range []*pgxpool.Pool{poolProyector, poolFuenteV1} {
			validadorAjeno, err := NuevoValidadorReferenciaMotivoPostgreSQLV2(
				poolAjeno, catalogoID,
			)
			if err != nil {
				t.Fatal("el constructor local rechazo un pool no nulo")
			}
			err = validadorAjeno.ValidarReferenciaMotivoAutorizacionV2(
				ctx, referencia, instante,
			)
			if !errors.Is(err, domain.ErrSolicitudAutorizacionInvalida) ||
				!errors.Is(err, ports.ErrFuenteAutorizacionNoDisponible) {
				t.Fatal("una identidad ajena no fallo como fuente sin capacidad")
			}
		}
	})

	t.Run("identidad evaluadora no proyecta ni usa la barrera actual", func(t *testing.T) {
		var resultado bool
		err := poolEvaluador.QueryRow(ctx, `
			SELECT vec_autorizacion.publicar_motivos_autorizacion_v2(
				$1::text, $2::bigint, $3::text, $4::text, $5::integer,
				$6::text, $7::timestamptz, $8::jsonb
			)`,
			"evento_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			int64(1), strings.Repeat("a", 64), catalogoID, 1,
			strings.Repeat("b", 64),
			time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
			`[{"clave":"motivo_cccccccccccccccccccccccccccccccc","vigente_desde":"2026-01-01T00:00:00.000000Z","vigente_hasta":null}]`,
		).Scan(&resultado)
		if err == nil {
			t.Fatal("la identidad evaluadora pudo proyectar")
		}

		err = poolEvaluador.QueryRow(ctx, `
			SELECT vec_autorizacion.resolver_motivo_autorizacion_v2_actual(
				$1::text, $2::integer, $3::text, $4::text
			)`,
			referencia.CatalogoID,
			referencia.CatalogoVersion,
			referencia.CatalogoHuellaSHA256,
			referencia.EntradaClave,
		).Scan(&resultado)
		if err == nil {
			t.Fatal("la identidad evaluadora pudo ejecutar la barrera actual")
		}
	})
}

func abrirPoolMotivoAutorizacionV2PostgreSQLIntegracion(
	t *testing.T,
	ctx context.Context,
	dsn string,
) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal("crear pool PostgreSQL para motivos V2")
	}
	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatal("conectar pool PostgreSQL para motivos V2")
	}
	return pool
}

func verificarIdentidadesMotivoAutorizacionV2PostgreSQLSeparadas(
	t *testing.T,
	ctx context.Context,
	pools ...*pgxpool.Pool,
) {
	t.Helper()
	identidadesSesion := make(map[string]struct{}, len(pools))
	for _, pool := range pools {
		var identidadSesion, identidadEfectiva string
		if err := pool.QueryRow(ctx, `SELECT session_user::text, current_user::text`).Scan(
			&identidadSesion,
			&identidadEfectiva,
		); err != nil || identidadSesion == "" || identidadEfectiva == "" {
			t.Fatal("no se pudo verificar una identidad PostgreSQL")
		}
		if identidadSesion != identidadEfectiva {
			t.Fatal("una capacidad PostgreSQL suplanta su identidad mediante SET ROLE")
		}
		if _, repetida := identidadesSesion[identidadSesion]; repetida {
			t.Fatal("dos capacidades PostgreSQL comparten LOGIN")
		}
		identidadesSesion[identidadSesion] = struct{}{}
	}
}
