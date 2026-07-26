package postgresimportacionconvoca

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	dominio "vec-diputacion-granada/internal/modules/bolsa/domain/importacionconvoca"
)

type entornoPostgreSQLIntegracion struct {
	ejecutor    *pgxpool.Pool
	recuperador *pgxpool.Pool
	conciliador *pgxpool.Pool
	retencion   *pgxpool.Pool
	gobernanza  *pgxpool.Pool
	admin       *pgxpool.Pool
	protector   *protectorAEADIntegracion
}

func abrirEntornoPostgreSQLIntegracion(t *testing.T) *entornoPostgreSQLIntegracion {
	t.Helper()
	variables := []string{
		"VEC_PRUEBA_BOLSA_CONVOCA_EJECUTOR_DSN",
		"VEC_PRUEBA_BOLSA_CONVOCA_RECUPERADOR_DSN",
		"VEC_PRUEBA_BOLSA_CONVOCA_CONCILIADOR_DSN",
		"VEC_PRUEBA_BOLSA_CONVOCA_RETENCION_DSN",
		"VEC_PRUEBA_BOLSA_CONVOCA_GOBERNANZA_DSN",
		"VEC_PRUEBA_BOLSA_CONVOCA_ADMIN_DSN",
	}
	for _, variable := range variables {
		if os.Getenv(variable) == "" {
			t.Skip("integracion PostgreSQL Convoca no solicitada")
		}
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelar()
	entorno := &entornoPostgreSQLIntegracion{
		ejecutor:    abrirPoolIntegracion(t, ctx, os.Getenv(variables[0])),
		recuperador: abrirPoolIntegracion(t, ctx, os.Getenv(variables[1])),
		conciliador: abrirPoolIntegracion(t, ctx, os.Getenv(variables[2])),
		retencion:   abrirPoolIntegracion(t, ctx, os.Getenv(variables[3])),
		gobernanza:  abrirPoolIntegracion(t, ctx, os.Getenv(variables[4])),
		admin:       abrirPoolIntegracion(t, ctx, os.Getenv(variables[5])),
		protector:   nuevoProtectorAEADIntegracion(),
	}
	t.Cleanup(func() {
		entorno.ejecutor.Close()
		entorno.recuperador.Close()
		entorno.conciliador.Close()
		entorno.retencion.Close()
		entorno.gobernanza.Close()
		entorno.admin.Close()
	})
	return entorno
}

func abrirPoolIntegracion(t *testing.T, ctx context.Context, dsn string) *pgxpool.Pool {
	t.Helper()
	configuracion, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("interpretar DSN de integracion: %v", err)
	}
	configuracion.MaxConns = 32
	pool, err := pgxpool.NewWithConfig(ctx, configuracion)
	if err != nil {
		t.Fatalf("abrir PostgreSQL de integracion: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatalf("comprobar PostgreSQL de integracion: %v", err)
	}
	return pool
}

func repositorioIntegracion(
	t *testing.T,
	entorno *entornoPostgreSQLIntegracion,
	politica string,
	duracion time.Duration,
) *RepositorioPostgreSQL {
	t.Helper()
	publicarPoliticaIntegracion(t, entorno, politica, 1, duracion)
	repositorio, err := NuevoRepositorioPostgreSQL(
		entorno.ejecutor, entorno.protector,
	)
	if err != nil {
		t.Fatalf("construir repositorio PostgreSQL: %v", err)
	}
	return repositorio
}

func recuperadorIntegracion(
	t *testing.T,
	entorno *entornoPostgreSQLIntegracion,
) *RepositorioRecuperacionPostgreSQL {
	t.Helper()
	repositorio, err := NuevoRepositorioRecuperacionPostgreSQL(
		entorno.recuperador, entorno.protector,
	)
	if err != nil {
		t.Fatalf("construir recuperador PostgreSQL: %v", err)
	}
	return repositorio
}

func publicarPoliticaIntegracion(
	t *testing.T,
	entorno *entornoPostgreSQLIntegracion,
	referencia string,
	version uint64,
	duracion time.Duration,
) {
	t.Helper()
	ctx, cancelar := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelar()
	_, err := entorno.gobernanza.Exec(ctx, `
		SELECT vec_bolsa_importacion_convoca.publicar_politica_retencion_v1(
		    $1::text, $2::bigint, $3::bigint, $4::text
		)`, referencia, version, int64(duracion/time.Second),
		"actor:gobernanza:integracion",
	)
	if err != nil {
		t.Fatalf("publicar politica de integracion: %v", err)
	}
}

func loteIntegracion(huella string, registrada time.Time, filas int) dominio.LoteValidado {
	aceptadas := make([]dominio.FilaAceptada, filas)
	for i := range aceptadas {
		aceptadas[i] = dominio.FilaAceptada{
			Numero: i + 2, Esquema: dominio.EsquemaResumenPersona,
			Identidad: dominio.IdentidadEnmascarada{
				Documento: "***0001**", PrimerApellido: "Sintetica",
				SegundoApellido: "Prueba", Nombre: "Persona",
			},
			Turno: "Libre",
			Resumen: &dominio.ResumenPersona{
				Experiencia: "1", Formacion: "1", Total: "2",
			},
		}
	}
	return dominio.LoteValidado{
		Acta: dominio.ActaImportacion{
			ActaRef:              "acta:importacion-convoca:" + huella,
			ImportacionRef:       "importacion:convoca:" + huella,
			HuellaFicheroSHA256:  huella,
			FicheroCustodiadoRef: "almacen:objeto:convoca:" + huella,
			NombreFichero:        "exportacion-sintetica-" + huella[:8] + ".xls",
			ActorRef:             "actor:rrhh:integracion", RegistradaEn: registrada.UTC(),
			Esquema:     dominio.EsquemaResumenPersona,
			FilasLeidas: filas, FilasAceptadas: filas,
			Procedencia: dominio.NuevaProcedenciaNoAutoritativa(),
		},
		Aceptadas: aceptadas,
	}
}

func huellaIntegracion(caracter string) string {
	return strings.Repeat(caracter, 64)
}

func exigirSQLState(t *testing.T, err error, codigo string) {
	t.Helper()
	var errorPG *pgconn.PgError
	if !errors.As(err, &errorPG) || errorPG.Code != codigo {
		t.Fatalf("SQLSTATE inesperado: esperado=%s error=%v", codigo, err)
	}
}
