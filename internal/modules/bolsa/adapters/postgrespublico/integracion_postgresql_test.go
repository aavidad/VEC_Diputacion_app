package postgrespublico

import (
	"context"
	"errors"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/publico/puertos"
)

func TestIntegracionPostgreSQLPublicoTLSACLConsultasYRevocacion(t *testing.T) {
	dsn := os.Getenv("VEC_PRUEBA_BOLSA_PUBLICA_DSN")
	dsnAdmin := os.Getenv("VEC_PRUEBA_BOLSA_PUBLICA_ADMIN_DSN")
	ancla := os.Getenv("VEC_PRUEBA_BOLSA_PUBLICA_MANIFIESTO_SHA256")
	if dsn == "" || dsnAdmin == "" || ancla == "" {
		t.Skip("integracion PostgreSQL no solicitada")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelar()
	fuente, err := Abrir(
		ctx, dsn, "categorias-profesionales", 1, strings.Repeat("a", 64),
		"4125f5b5f12f3da31fff30aa699239592d02b01b1676e98d8fa1ab7beb30ad7d",
		ancla,
	)
	if err != nil {
		t.Fatalf("abrir proyeccion con TLS verificado: %v", err)
	}
	defer fuente.Cerrar()

	var soloLectura string
	if err := fuente.pool.QueryRow(ctx, "SHOW transaction_read_only").Scan(&soloLectura); err != nil || soloLectura != "on" {
		t.Fatalf("sesion no quedo en solo lectura: %q, %v", soloLectura, err)
	}
	assertColumnasVistas(t, ctx, fuente.pool)
	assertACLNegativas(t, ctx, fuente.pool)

	instante := time.Date(2026, 7, 22, 12, 0, 0, 0, time.UTC)
	if err := fuente.ValidarConfiguracionPublica(ctx, instante); err != nil {
		t.Fatalf("validar manifiesto de arranque: %v", err)
	}
	categorias, err := fuente.ObtenerPublicadas(ctx, instante)
	if err != nil || categorias.Fuente.Demostracion || len(categorias.Categorias) != 1 ||
		categorias.Categorias[0].Clave != "auxiliar-administrativo" {
		t.Fatalf("catalogo publico inesperado: %+v, %v", categorias, err)
	}
	pagina, err := fuente.BuscarPublicadas(ctx, puertosbolsa.FiltroConvocatoriasPublicas{
		Texto: "auxiliares", Categoria: "auxiliar-administrativo", SoloPlazoAbierto: true,
		Instante: instante, Limite: 24,
	})
	if err != nil || pagina.Total != 1 || len(pagina.Convocatorias) != 1 ||
		pagina.Fuente.Demostracion || pagina.Convocatorias[0].DatosPublicos == nil {
		t.Fatalf("listado publico inesperado: total=%d filas=%d fuente=%+v error=%v",
			pagina.Total, len(pagina.Convocatorias), pagina.Fuente, err)
	}
	if pagina.ConteosCategorias["auxiliar-administrativo"].NumeroPlazosAbiertos != 1 {
		t.Fatalf("faceta de plazos inesperada: %+v", pagina.ConteosCategorias)
	}
	detalle, err := fuente.ObtenerPublicada(ctx, "auxiliares-2026")
	if err != nil || detalle.Convocatoria.ValidarPublicacion() != nil ||
		len(detalle.Convocatoria.DatosPublicos.Documentos) != 1 {
		t.Fatalf("detalle publico inesperado: %+v, %v", detalle, err)
	}
	cierre := time.Date(2026, 7, 31, 23, 59, 59, 0, time.UTC)
	for _, frontera := range []struct {
		instante time.Time
		total    int
	}{
		{instante: cierre, total: 1},
		{instante: cierre.Add(time.Microsecond), total: 0},
	} {
		paginaFrontera, err := fuente.BuscarPublicadas(ctx, puertosbolsa.FiltroConvocatoriasPublicas{
			SoloPlazoAbierto: true, Instante: frontera.instante, Limite: 24,
		})
		if err != nil || paginaFrontera.Total != frontera.total {
			t.Fatalf("frontera inclusiva de cierre en %s: total=%d error=%v",
				frontera.instante, paginaFrontera.Total, err)
		}
	}

	admin, err := pgxpool.New(ctx, dsnAdmin)
	if err != nil {
		t.Fatalf("abrir conexion administradora de prueba: %v", err)
	}
	defer admin.Close()

	assertDerivaACLRechazada(t, ctx, fuente, admin, instante)

	// Cualquier DML ajeno a la publicación atómica invalida el testigo. Todas
	// las rutas fallan cerradas antes de devolver siquiera una faceta parcial.
	if _, err := admin.Exec(ctx, `
			UPDATE vec_bolsa_publica_datos.categoria_publica
		   SET etiqueta = 'Auxiliar administrativo alterado'
		 WHERE catalogo_id = 'categorias-profesionales' AND version = 1
			   AND clave = 'auxiliar-administrativo'`); err != nil {
		t.Fatal(err)
	}
	operaciones := map[string]func() error{
		"configuracion": func() error { return fuente.ValidarConfiguracionPublica(ctx, instante) },
		"categorias":    func() error { _, err := fuente.ObtenerPublicadas(ctx, instante); return err },
		"listado": func() error {
			_, err := fuente.BuscarPublicadas(ctx, puertosbolsa.FiltroConvocatoriasPublicas{Instante: instante, Limite: 24})
			return err
		},
		"detalle": func() error { _, err := fuente.ObtenerPublicada(ctx, "auxiliares-2026"); return err },
	}
	for nombre, operar := range operaciones {
		if err := operar(); !errors.Is(err, ErrDatosPostgreSQLPublicosNoConfiables) {
			t.Fatalf("%s no fallo cerrada tras DML ajeno: %v", nombre, err)
		}
	}
}

func assertDerivaACLRechazada(
	t *testing.T,
	ctx context.Context,
	fuente *Fuente,
	admin *pgxpool.Pool,
	instante time.Time,
) {
	t.Helper()
	const login = "vec_bolsa_publica_integracion_login"
	probar := func(nombre, conceder, revocar string) {
		t.Helper()
		if _, err := admin.Exec(ctx, conceder); err != nil {
			t.Fatalf("preparar deriva %s: %v", nombre, err)
		}
		_, err := fuente.ObtenerPublicadas(ctx, instante)
		if !errors.Is(err, ErrIdentidadPostgreSQLPublicaInvalida) {
			t.Fatalf("deriva %s no fue rechazada: %v", nombre, err)
		}
		if _, err := admin.Exec(ctx, revocar); err != nil {
			t.Fatalf("revertir deriva %s: %v", nombre, err)
		}
		if _, err := fuente.ObtenerPublicadas(ctx, instante); err != nil {
			t.Fatalf("fuente no se recuperó tras %s: %v", nombre, err)
		}
	}
	probar(
		"grant directo en vista esperada",
		"GRANT SELECT ON vec_bolsa_publica_lectura.fuente_publica_v1 TO "+pgx.Identifier{login}.Sanitize(),
		"REVOKE SELECT ON vec_bolsa_publica_lectura.fuente_publica_v1 FROM "+pgx.Identifier{login}.Sanitize(),
	)
	probar(
		"lectura directa de tabla base",
		"GRANT SELECT ON vec_bolsa_publica_datos.convocatoria_publica TO "+pgx.Identifier{login}.Sanitize(),
		"REVOKE SELECT ON vec_bolsa_publica_datos.convocatoria_publica FROM "+pgx.Identifier{login}.Sanitize(),
	)
	probar(
		"schema adicional",
		"CREATE SCHEMA vec_intrusion; GRANT USAGE, CREATE ON SCHEMA vec_intrusion TO "+pgx.Identifier{login}.Sanitize(),
		"DROP SCHEMA vec_intrusion",
	)
	if _, err := admin.Exec(ctx, "CREATE DATABASE vec_intrusion"); err != nil {
		t.Fatalf("crear base adicional: %v", err)
	}
	if _, err := admin.Exec(ctx, "GRANT CONNECT ON DATABASE vec_intrusion TO "+pgx.Identifier{login}.Sanitize()); err != nil {
		t.Fatalf("conceder base adicional: %v", err)
	}
	if _, err := fuente.ObtenerPublicadas(ctx, instante); !errors.Is(err, ErrIdentidadPostgreSQLPublicaInvalida) {
		t.Fatalf("otra base accesible no fue rechazada: %v", err)
	}
	if _, err := admin.Exec(ctx, "REVOKE CONNECT ON DATABASE vec_intrusion FROM "+pgx.Identifier{login}.Sanitize()); err != nil {
		t.Fatalf("revocar base adicional: %v", err)
	}
	if _, err := admin.Exec(ctx, "DROP DATABASE vec_intrusion"); err != nil {
		t.Fatalf("eliminar base adicional: %v", err)
	}
	if _, err := fuente.ObtenerPublicadas(ctx, instante); err != nil {
		t.Fatalf("fuente no se recuperó tras base adicional: %v", err)
	}

	conexion, err := fuente.pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conexion.Exec(ctx, "SET statement_timeout = 0; SET default_transaction_read_only = off"); err != nil {
		conexion.Release()
		t.Fatalf("preparar deriva de GUC: %v", err)
	}
	conexion.Release()
	if _, err := fuente.ObtenerPublicadas(ctx, instante); !errors.Is(err, ErrIdentidadPostgreSQLPublicaInvalida) {
		t.Fatalf("deriva de GUC no fue rechazada: %v", err)
	}
	if _, err := fuente.ObtenerPublicadas(ctx, instante); err != nil {
		t.Fatalf("fuente no se recuperó tras deriva de GUC: %v", err)
	}
}

func assertACLNegativas(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	for nombre, consulta := range map[string]string{
		"tabla base":      "SELECT count(*) FROM vec_bolsa_publica_datos.convocatoria_publica",
		"escritura vista": "DELETE FROM vec_bolsa_publica_lectura.convocatorias_publicadas_v1",
		"creacion":        "CREATE TABLE public.intrusion_vec_bolsa_publica(id integer)",
		"tabla temporal":  "CREATE TEMP TABLE intrusion_vec_bolsa_publica(id integer)",
	} {
		if _, err := pool.Exec(ctx, consulta); err == nil {
			t.Fatalf("la cuenta publica obtuvo privilegio de %s", nombre)
		} else if codigoSQL(err) != "42501" && codigoSQL(err) != "25006" && codigoSQL(err) != "55000" {
			t.Fatalf("%s fallo por causa inesperada: %v", nombre, err)
		}
	}
}

func assertColumnasVistas(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	esperadas := map[string][]string{
		"fuente_publica_v1": {"actualizada_en", "control_id", "manifiesto_sha256", "revision"},
		"entradas_catalogos_publicos_v1": {
			"clave", "descripcion", "etiqueta", "orden", "publicable", "referencia", "semantica", "version",
		},
		"catalogos_categorias_publicos_v1": {
			"actualizada_en", "catalogo_id", "huella_gobernada_sha256",
			"huella_proyeccion_publica_sha256", "revision", "version",
		},
		"categorias_publicas_v1": {
			"area", "area_etiqueta", "catalogo_id", "clave", "descripcion", "etiqueta", "orden",
			"semantica", "suscribible", "version", "vigente_desde", "vigente_hasta",
		},
		"convocatorias_publicadas_v1": {
			"actualizada_en", "catalogo_categorias_huella_sha256", "catalogo_categorias_id",
			"catalogo_categorias_version", "descripcion", "estado", "huella_publica_sha256",
			"huella_resumen_publico_sha256",
			"identificador_publico", "publicada_en", "resumen", "tipo", "titulo", "version_publica",
			"busqueda",
		},
		"categorias_convocatorias_publicas_v1": {"categoria_clave", "identificador_publico"},
		"plazos_convocatorias_publicas_v1": {
			"abre_en", "cierra_en", "descripcion", "identificador_publico", "referencia", "tipo", "titulo",
		},
		"requisitos_convocatorias_publicas_v1": {
			"descripcion", "identificador_publico", "obligatorio", "orden", "referencia", "titulo",
		},
		"documentos_convocatorias_publicas_v1": {
			"descripcion", "formato", "identificador_publico", "orden", "publicado_en", "referencia",
			"tipo", "titulo", "url",
		},
		"ayuda_convocatorias_publicas_v1": {
			"categoria", "identificador_publico", "orden", "pregunta", "referencia", "respuesta",
		},
	}
	for vista, columnasEsperadas := range esperadas {
		var puedeLeer, puedeMutar bool
		if err := pool.QueryRow(ctx, `
			SELECT has_table_privilege(current_user, 'vec_bolsa_publica_lectura.' || $1, 'SELECT'),
			       has_table_privilege(current_user, 'vec_bolsa_publica_lectura.' || $1,
			           'INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER')`, vista,
		).Scan(&puedeLeer, &puedeMutar); err != nil || !puedeLeer || puedeMutar {
			t.Fatalf("ACL de %s inesperada: leer=%t mutar=%t error=%v", vista, puedeLeer, puedeMutar, err)
		}
		filas, err := pool.Query(ctx, `
			SELECT column_name
			  FROM information_schema.columns
			 WHERE table_schema = 'vec_bolsa_publica_lectura' AND table_name = $1
		 ORDER BY column_name`, vista)
		if err != nil {
			t.Fatalf("inventariar %s: %v", vista, err)
		}
		var columnas []string
		for filas.Next() {
			var columna string
			if err := filas.Scan(&columna); err != nil {
				filas.Close()
				t.Fatal(err)
			}
			columnas = append(columnas, columna)
		}
		filas.Close()
		sort.Strings(columnasEsperadas)
		if strings.Join(columnas, ",") != strings.Join(columnasEsperadas, ",") {
			t.Fatalf("allowlist de %s distinta: %v", vista, columnas)
		}
		for _, columna := range columnas {
			minuscula := strings.ToLower(columna)
			for _, pii := range []string{"dni", "nif", "correo", "telefono", "persona", "principal", "actor", "candidato"} {
				segmentos := strings.Split(minuscula, "_")
				if contieneCadena(segmentos, pii) {
					t.Fatalf("columna potencialmente personal expuesta: %s.%s", vista, columna)
				}
			}
		}
	}
	filasPrivilegios, err := pool.Query(ctx, `
		SELECT table_name, privilege_type
		  FROM information_schema.role_table_grants
		 WHERE table_schema = 'vec_bolsa_publica_lectura'
		   AND grantee = 'vec_bolsa_publica_consulta'
		 ORDER BY table_name, privilege_type`)
	if err != nil {
		t.Fatalf("inventariar privilegios de consulta: %v", err)
	}
	defer filasPrivilegios.Close()
	privilegios := make(map[string][]string)
	for filasPrivilegios.Next() {
		var tabla, privilegio string
		if err := filasPrivilegios.Scan(&tabla, &privilegio); err != nil {
			t.Fatal(err)
		}
		privilegios[tabla] = append(privilegios[tabla], privilegio)
	}
	if err := filasPrivilegios.Err(); err != nil {
		t.Fatal(err)
	}
	if len(privilegios) != len(esperadas) {
		t.Fatalf("el rol de consulta no tiene la lista exacta de vistas: %+v", privilegios)
	}
	for vista := range esperadas {
		if strings.Join(privilegios[vista], ",") != "SELECT" {
			t.Fatalf("privilegios directos inesperados en %s: %v", vista, privilegios[vista])
		}
	}
	var usoLectura, usoDatos, temporal bool
	if err := pool.QueryRow(ctx, `
		SELECT has_schema_privilege(current_user, 'vec_bolsa_publica_lectura', 'USAGE'),
		       has_schema_privilege(current_user, 'vec_bolsa_publica_datos', 'USAGE'),
		       has_database_privilege(current_user, current_database(), 'TEMPORARY')`,
	).Scan(&usoLectura, &usoDatos, &temporal); err != nil || !usoLectura || usoDatos || temporal {
		t.Fatalf("frontera de esquemas/base inesperada: lectura=%t datos=%t temporal=%t error=%v",
			usoLectura, usoDatos, temporal, err)
	}
}

func contieneCadena(valores []string, buscado string) bool {
	for _, valor := range valores {
		if valor == buscado {
			return true
		}
	}
	return false
}

func codigoSQL(err error) string {
	var postgres *pgconn.PgError
	if errors.As(err, &postgres) {
		return postgres.Code
	}
	return ""
}
