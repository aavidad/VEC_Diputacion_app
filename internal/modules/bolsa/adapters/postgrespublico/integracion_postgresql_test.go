package postgrespublico

import (
	"context"
	"errors"
	"os"
	"sort"
	"strings"
	"sync"
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
	dsnPublicador := os.Getenv("VEC_PRUEBA_BOLSA_PUBLICA_PUBLICADOR_DSN")
	ancla := os.Getenv("VEC_PRUEBA_BOLSA_PUBLICA_MANIFIESTO_SHA256")
	if dsn == "" || dsnAdmin == "" || dsnPublicador == "" || ancla == "" {
		t.Skip("integracion PostgreSQL no solicitada")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancelar()
	fuente, err := Abrir(
		ctx, dsn, "categorias-profesionales", 2, strings.Repeat("b", 64),
		"b661b37ca7323fa168734899038f8fa99cb77ff07d114e4a7d787d62b5d36593",
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
	archivo, err := fuente.BuscarPublicadas(ctx, puertosbolsa.FiltroConvocatoriasPublicas{
		Instante: instante, Limite: 24,
	})
	if err != nil || archivo.Total != 2 || len(archivo.Convocatorias) != 2 {
		t.Fatalf("archivo multiversión inesperado: total=%d filas=%d error=%v",
			archivo.Total, len(archivo.Convocatorias), err)
	}
	historica, err := fuente.ObtenerPublicada(ctx, "auxiliares-2024")
	if err != nil || historica.Convocatoria.DatosPublicos == nil ||
		historica.Convocatoria.DatosPublicos.CatalogoCategorias.CatalogoVersion != 1 {
		t.Fatalf("convocatoria histórica no resoluble: %+v, %v", historica, err)
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
	assertErroresPublicacionRedactados(t, ctx, admin, dsnPublicador)
	fuenteB := assertPublicacionMVCCAtomica(
		t, ctx, fuente, admin, dsnPublicador, dsn, instante,
	)
	defer fuenteB.Cerrar()

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
		"configuracion": func() error { return fuenteB.ValidarConfiguracionPublica(ctx, instante) },
		"categorias":    func() error { _, err := fuenteB.ObtenerPublicadas(ctx, instante); return err },
		"listado": func() error {
			_, err := fuenteB.BuscarPublicadas(ctx, puertosbolsa.FiltroConvocatoriasPublicas{Instante: instante, Limite: 24})
			return err
		},
		"detalle": func() error { _, err := fuenteB.ObtenerPublicada(ctx, "auxiliares-2026"); return err },
	}
	for nombre, operar := range operaciones {
		if err := operar(); !errors.Is(err, ErrDatosPostgreSQLPublicosNoConfiables) {
			t.Fatalf("%s no fallo cerrada tras DML ajeno: %v", nombre, err)
		}
	}
}

const marcadorPrivacidadPostgreSQL = "DNI-PRIVADO-C3-00000000T"

func assertPublicacionMVCCAtomica(
	t *testing.T,
	ctx context.Context,
	fuente *Fuente,
	admin *pgxpool.Pool,
	dsnPublicador string,
	dsnLector string,
	instante time.Time,
) *Fuente {
	t.Helper()
	txLectura, err := fuente.iniciarLectura(ctx, true)
	if err != nil {
		t.Fatal(err)
	}
	manifiestoB, _, err := fuente.leerManifiestoPublico(ctx, txLectura)
	if err != nil {
		_ = txLectura.Rollback(context.Background())
		t.Fatalf("leer manifiesto A para calcular B: %v", err)
	}
	if err := txLectura.Commit(ctx); err != nil {
		t.Fatalf("cerrar cálculo de manifiesto B: %v", err)
	}
	manifiestoB.Fuente.Revision = "revision-atomica-b"
	anclaB, err := manifiestoB.HuellaSHA256()
	if err != nil {
		t.Fatalf("calcular ancla B: %v", err)
	}
	var payload string
	if err := admin.QueryRow(ctx, `SELECT payload::text FROM public.proyeccion_publica_prueba`).Scan(&payload); err != nil {
		t.Fatalf("leer fixture de publicación: %v", err)
	}
	publicador, err := pgx.Connect(ctx, dsnPublicador)
	if err != nil {
		t.Fatalf("abrir LOGIN publicador: %v", err)
	}
	defer publicador.Close(context.Background())
	transaccion, err := publicador.Begin(ctx)
	if err != nil {
		t.Fatalf("iniciar publicación B: %v", err)
	}
	defer func() { _ = transaccion.Rollback(context.Background()) }()
	if _, err := transaccion.Exec(ctx, `
		SELECT vec_bolsa_publica_publicacion.publicar_proyeccion_v2(
			jsonb_set($1::jsonb, '{fuente,revision}', to_jsonb($2::text)), $3
		)`, payload, manifiestoB.Fuente.Revision, anclaB); err != nil {
		t.Fatalf("ejecutar función publicadora B: %v", err)
	}

	const lectores = 8
	errores := make(chan error, lectores)
	var grupo sync.WaitGroup
	inicio := time.Now()
	for range lectores {
		grupo.Add(1)
		go func() {
			defer grupo.Done()
			ctxLectura, cancelar := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancelar()
			pagina, err := fuente.BuscarPublicadas(ctxLectura, puertosbolsa.FiltroConvocatoriasPublicas{
				Instante: instante, Limite: 24,
			})
			if err == nil && (pagina.Total != 2 || pagina.Fuente.Revision != "revision-001") {
				err = errors.New("lector no observó A coherente durante publicación B")
			}
			errores <- err
		}()
	}
	grupo.Wait()
	close(errores)
	for err := range errores {
		if err != nil {
			t.Fatalf("lector no sirvió A mientras B estaba sin confirmar: %v", err)
		}
	}
	if transcurrido := time.Since(inicio); transcurrido > 2*time.Second {
		t.Fatalf("los lectores esperaron al publicador: %s", transcurrido)
	}
	estadisticas := fuente.pool.Stat()
	if estadisticas.AcquiredConns() != 0 || estadisticas.TotalConns() > 6 {
		t.Fatalf("pool acumulado durante publicación: adquiridas=%d totales=%d",
			estadisticas.AcquiredConns(), estadisticas.TotalConns())
	}
	if err := transaccion.Commit(ctx); err != nil {
		t.Fatalf("confirmar publicación B: %v", err)
	}
	operacionesA := []func() error{
		func() error { return fuente.ValidarConfiguracionPublica(ctx, instante) },
		func() error { _, err := fuente.ObtenerPublicadas(ctx, instante); return err },
		func() error {
			_, err := fuente.BuscarPublicadas(ctx, puertosbolsa.FiltroConvocatoriasPublicas{
				Instante: instante, Limite: 24,
			})
			return err
		},
		func() error { _, err := fuente.ObtenerPublicada(ctx, "auxiliares-2024"); return err },
	}
	for _, operar := range operacionesA {
		if err := operar(); !errors.Is(err, ErrDatosPostgreSQLPublicosNoConfiables) {
			t.Fatalf("fuente A no falló cerrada tras COMMIT B: %v", err)
		}
	}
	fuenteB, err := Abrir(
		ctx, dsnLector, "categorias-profesionales", 2, strings.Repeat("b", 64),
		"b661b37ca7323fa168734899038f8fa99cb77ff07d114e4a7d787d62b5d36593",
		anclaB,
	)
	if err != nil {
		t.Fatalf("abrir fuente B: %v", err)
	}
	if err := fuenteB.ValidarConfiguracionPublica(ctx, instante); err != nil {
		fuenteB.Cerrar()
		t.Fatalf("validar fuente B: %v", err)
	}
	return fuenteB
}

func assertErroresPublicacionRedactados(
	t *testing.T,
	ctx context.Context,
	admin *pgxpool.Pool,
	dsnPublicador string,
) {
	t.Helper()
	var payload string
	if err := admin.QueryRow(ctx, `SELECT payload::text FROM public.proyeccion_publica_prueba`).Scan(&payload); err != nil {
		t.Fatal(err)
	}
	publicador, err := pgx.Connect(ctx, dsnPublicador)
	if err != nil {
		t.Fatalf("abrir publicador para probar redacción: %v", err)
	}
	defer publicador.Close(context.Background())
	var limiteParametros string
	if err := publicador.QueryRow(ctx, `SHOW log_parameter_max_length_on_error`).Scan(&limiteParametros); err != nil || limiteParametros != "0" {
		t.Fatalf("LOGIN publicador sin redacción de parámetros: %q, %v", limiteParametros, err)
	}
	casos := []string{
		`SELECT vec_bolsa_publica_publicacion.publicar_proyeccion_v2(
			jsonb_set($1::jsonb, '{categorias,snapshots,0,categorias,0,vigente_desde}', to_jsonb($2::text)),
			repeat('e', 64))`,
		`SELECT vec_bolsa_publica_publicacion.publicar_proyeccion_v2(
			jsonb_set($1::jsonb, '{catalogos,0,referencia}', to_jsonb(($2::text || '@'))),
			repeat('f', 64))`,
	}
	for _, consulta := range casos {
		_, err := publicador.Exec(ctx, consulta, payload, marcadorPrivacidadPostgreSQL)
		var postgres *pgconn.PgError
		if !errors.As(err, &postgres) || postgres.Code != "22023" ||
			postgres.Message != "publicacion rechazada: contenido invalido" ||
			postgres.Detail != "" || postgres.Hint != "" ||
			strings.Contains(err.Error(), marcadorPrivacidadPostgreSQL) ||
			strings.Contains(postgres.Where, marcadorPrivacidadPostgreSQL) {
			t.Fatalf("error de publicación no redactado: %#v, %v", postgres, err)
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
		"GRANT SELECT ON vec_bolsa_publica_lectura.fuente_publica_v2 TO "+pgx.Identifier{login}.Sanitize(),
		"REVOKE SELECT ON vec_bolsa_publica_lectura.fuente_publica_v2 FROM "+pgx.Identifier{login}.Sanitize(),
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
		"escritura vista": "DELETE FROM vec_bolsa_publica_lectura.convocatorias_publicadas_v2",
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
		"fuente_publica_v2": {"actualizada_en", "control_id", "manifiesto_sha256", "revision"},
		"entradas_catalogos_publicos_v2": {
			"clave", "descripcion", "etiqueta", "orden", "publicable", "referencia", "semantica", "version",
		},
		"catalogos_categorias_publicos_v2": {
			"actual", "actualizada_en", "catalogo_id", "huella_gobernada_sha256",
			"huella_proyeccion_publica_sha256", "revision", "version",
		},
		"categorias_publicas_v2": {
			"area", "area_etiqueta", "catalogo_id", "clave", "descripcion", "etiqueta", "orden",
			"semantica", "suscribible", "version", "vigente_desde", "vigente_hasta",
		},
		"convocatorias_publicadas_v2": {
			"actualizada_en", "catalogo_categorias_huella_proyeccion_sha256",
			"catalogo_categorias_huella_sha256", "catalogo_categorias_id",
			"catalogo_categorias_version", "descripcion", "estado", "huella_publica_sha256",
			"huella_resumen_publico_sha256",
			"identificador_publico", "publicada_en", "resumen", "tipo", "titulo", "version_publica",
			"busqueda",
		},
		"categorias_convocatorias_publicas_v2": {
			"catalogo_id", "catalogo_version", "categoria_clave", "identificador_publico",
		},
		"plazos_convocatorias_publicas_v2": {
			"abre_en", "cierra_en", "descripcion", "identificador_publico", "referencia", "tipo", "titulo",
		},
		"requisitos_convocatorias_publicas_v2": {
			"descripcion", "identificador_publico", "obligatorio", "orden", "referencia", "titulo",
		},
		"documentos_convocatorias_publicas_v2": {
			"descripcion", "formato", "identificador_publico", "orden", "publicado_en", "referencia",
			"tipo", "titulo", "url",
		},
		"ayuda_convocatorias_publicas_v2": {
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
