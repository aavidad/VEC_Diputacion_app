package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const variableRaizGobiernoCoberturaO404B = "VEC_O404B_RAIZ"
const claveBarreraGobiernoCoberturaO404B = "vec_contratacion_temporal:o4_04:migraciones"

func verificarBarreraGobiernoCoberturaO404B(
	t *testing.T,
	ctx context.Context,
	administrador *pgxpool.Pool,
) {
	t.Helper()
	// La fila común control_migracion_cobertura_o4 nace en C/000020. Antes,
	// 000017-000019 comparten la advisory y serializan el checkpoint propio de B.
	var funcionesValidas bool
	if err := administrador.QueryRow(ctx, `
		WITH definiciones AS (
		  SELECT
		    pg_catalog.pg_get_functiondef(
		      'vec_contratacion_temporal.gobi_o404b_publicar(jsonb)'::regprocedure
		    ) publicar,
		    pg_catalog.pg_get_functiondef(
		      'vec_contratacion_temporal.gobi_o404b_retirar(jsonb)'::regprocedure
		    ) retirar
		)
		SELECT publicar LIKE '%' || $1 || '%'
		  AND retirar LIKE '%' || $1 || '%'
		  AND publicar LIKE '%gobi_o404b_checkpoint%WHERE control FOR UPDATE%'
		  AND retirar LIKE '%gobi_o404b_checkpoint%WHERE control FOR UPDATE%'
		FROM definiciones`,
		claveBarreraGobiernoCoberturaO404B,
	).Scan(&funcionesValidas); err != nil || !funcionesValidas {
		t.Fatalf("barrera/checkpoint O4-04B divergente: %t / %v", funcionesValidas, err)
	}
	raiz := os.Getenv(variableRaizGobiernoCoberturaO404B)
	if raiz == "" {
		t.Fatal("raíz del repositorio O4-04B no configurada")
	}
	for _, archivo := range []string{
		"000017_gobierno_cobertura_o4_04b.up.sql",
		"000017_gobierno_cobertura_o4_04b.down.sql",
		"000018_politicas_gobierno_cobertura_o4_04b.up.sql",
		"000018_politicas_gobierno_cobertura_o4_04b.down.sql",
		"000019_resolucion_gobierno_cobertura_o4_04b.up.sql",
		"000019_resolucion_gobierno_cobertura_o4_04b.down.sql",
	} {
		contenido, err := os.ReadFile(filepath.Join(
			raiz,
			"deploy/postgresql/contratacion_temporal/migraciones",
			archivo,
		))
		if err != nil ||
			!strings.Contains(string(contenido), claveBarreraGobiernoCoberturaO404B) ||
			strings.Contains(string(contenido), "vec_contratacion_temporal:migraciones_o4_04") {
			t.Fatalf("barrera advisory divergente en %s: %v", archivo, err)
		}
	}
}

func verificarCanonesGobiernoCoberturaO404B(
	t *testing.T,
	ctx context.Context,
	administrador *pgxpool.Pool,
	catalogoJSON string,
	politicaJSON string,
	actuacionJSON string,
) {
	t.Helper()
	var catalogo, politica, actuacion map[string]any
	decodificarJSONGobiernoCoberturaO404BPrueba(
		t,
		catalogoJSON,
		&catalogo,
	)
	decodificarJSONGobiernoCoberturaO404BPrueba(
		t,
		politicaJSON,
		&politica,
	)
	decodificarJSONGobiernoCoberturaO404BPrueba(
		t,
		actuacionJSON,
		&actuacion,
	)
	var huellaCatalogo, huellaPolitica, huellaActuacion string
	if err := administrador.QueryRow(ctx, `
		SELECT
		  pg_catalog.encode(pg_catalog.sha256(
		    vec_contratacion_temporal
		      .gobi_o404b_material_catalogo($1::jsonb)
		  ), 'hex'),
		  pg_catalog.encode(pg_catalog.sha256(
		    vec_contratacion_temporal
		      .gobi_o404b_material_politica($2::jsonb)
		  ), 'hex'),
		  pg_catalog.encode(pg_catalog.sha256(
		    vec_contratacion_temporal
		      .gobi_o404b_material_actuacion($3::jsonb)
		  ), 'hex')`,
		catalogoJSON,
		politicaJSON,
		actuacionJSON,
	).Scan(
		&huellaCatalogo,
		&huellaPolitica,
		&huellaActuacion,
	); err != nil ||
		huellaCatalogo != catalogo["huella_sha256"] ||
		huellaPolitica != politica["huella_sha256"] ||
		huellaActuacion != actuacion["huella_sha256"] {
		t.Fatalf(
			"huellas SQL/Go divergentes: %s/%s/%s / %v",
			huellaCatalogo,
			huellaPolitica,
			huellaActuacion,
			err,
		)
	}
	for _, caso := range []struct {
		funcion string
		valor   map[string]any
	}{
		{"gobi_o404b_material_catalogo", catalogo},
		{"gobi_o404b_material_politica", politica},
		{"gobi_o404b_material_actuacion", actuacion},
	} {
		conCampoAjeno := clonarJSONGobiernoCoberturaO404BPrueba(
			t,
			caso.valor,
		)
		conCampoAjeno["campo_fuera_del_canon"] = true
		exigirMaterialNuloGobiernoCoberturaO404B(
			t,
			ctx,
			administrador,
			caso.funcion,
			conCampoAjeno,
		)
	}
	catalogoDecimal := clonarJSONGobiernoCoberturaO404BPrueba(t, catalogo)
	catalogoDecimal["version"] = 1.5
	exigirMaterialNuloGobiernoCoberturaO404B(
		t,
		ctx,
		administrador,
		"gobi_o404b_material_catalogo",
		catalogoDecimal,
	)
	catalogoCanonTexto := clonarJSONGobiernoCoberturaO404BPrueba(t, catalogo)
	catalogoCanonTexto["canon"].(map[string]any)["version_esquema"] = "1"
	exigirMaterialNuloGobiernoCoberturaO404B(
		t,
		ctx,
		administrador,
		"gobi_o404b_material_catalogo",
		catalogoCanonTexto,
	)
	catalogoInstante := clonarJSONGobiernoCoberturaO404BPrueba(t, catalogo)
	catalogoInstante["publicado_en"] = "2026-07-25T24:00:00Z"
	exigirMaterialNuloGobiernoCoberturaO404B(
		t,
		ctx,
		administrador,
		"gobi_o404b_material_catalogo",
		catalogoInstante,
	)
	catalogoProcedencia := clonarJSONGobiernoCoberturaO404BPrueba(t, catalogo)
	vias := catalogoProcedencia["vias"].([]any)
	viaNueva := clonarJSONGobiernoCoberturaO404BPrueba(
		t,
		vias[0].(map[string]any),
	)
	viaNueva["clave"] = "via_configurable_o404b_otra"
	viaNueva["orden"] = float64(2)
	comprobacion := viaNueva["comprobaciones"].([]any)[0].(map[string]any)
	comprobacion["procedencia"].(map[string]any)["definicion_fuente_ref"] =
		"fuente_cobertura_o404b_otra"
	catalogoProcedencia["vias"] = append(vias, viaNueva)
	exigirMaterialNuloGobiernoCoberturaO404B(
		t,
		ctx,
		administrador,
		"gobi_o404b_material_catalogo",
		catalogoProcedencia,
	)
	politicaFueraVigencia := clonarJSONGobiernoCoberturaO404BPrueba(t, politica)
	desde, err := time.Parse(
		time.RFC3339Nano,
		politicaFueraVigencia["vigencia"].(map[string]any)["desde"].(string),
	)
	if err != nil {
		t.Fatal(err)
	}
	politicaFueraVigencia["publicada_en"] =
		desde.Add(time.Microsecond).Format(time.RFC3339Nano)
	exigirMaterialNuloGobiernoCoberturaO404B(
		t,
		ctx,
		administrador,
		"gobi_o404b_material_politica",
		politicaFueraVigencia,
	)
	actuacionMotivo := clonarJSONGobiernoCoberturaO404BPrueba(t, actuacion)
	motivoDecidir :=
		actuacionMotivo["motivo_autorizacion_decidir"].(map[string]any)
	motivoDecidir["entrada_clave"] = "motivo_libre"
	exigirMaterialNuloGobiernoCoberturaO404B(
		t,
		ctx,
		administrador,
		"gobi_o404b_material_actuacion",
		actuacionMotivo,
	)
	actuacionVersion := clonarJSONGobiernoCoberturaO404BPrueba(t, actuacion)
	actuacionVersion["catalogo"].(map[string]any)["version"] = 1.5
	exigirMaterialNuloGobiernoCoberturaO404B(
		t,
		ctx,
		administrador,
		"gobi_o404b_material_actuacion",
		actuacionVersion,
	)
}

func decodificarJSONGobiernoCoberturaO404BPrueba(
	t *testing.T,
	contenido string,
	destino any,
) {
	t.Helper()
	if err := json.Unmarshal([]byte(contenido), destino); err != nil {
		t.Fatal(err)
	}
}

func clonarJSONGobiernoCoberturaO404BPrueba(
	t *testing.T,
	origen map[string]any,
) map[string]any {
	t.Helper()
	contenido, err := json.Marshal(origen)
	if err != nil {
		t.Fatal(err)
	}
	var clon map[string]any
	decodificarJSONGobiernoCoberturaO404BPrueba(
		t,
		string(contenido),
		&clon,
	)
	return clon
}

func exigirMaterialNuloGobiernoCoberturaO404B(
	t *testing.T,
	ctx context.Context,
	administrador *pgxpool.Pool,
	funcion string,
	publicacion map[string]any,
) {
	t.Helper()
	contenido, err := json.Marshal(publicacion)
	if err != nil {
		t.Fatal(err)
	}
	var nulo bool
	if err := administrador.QueryRow(
		ctx,
		"SELECT vec_contratacion_temporal."+
			funcion+"($1::jsonb) IS NULL",
		contenido,
	).Scan(&nulo); err != nil || !nulo {
		t.Fatalf("canon SQL permisivo en %s: nulo=%t err=%v", funcion, nulo, err)
	}
}

func verificarDownFueraDeOrdenGobiernoCoberturaO404B(
	t *testing.T,
	ctx context.Context,
	administrador *pgxpool.Pool,
	archivo string,
) {
	t.Helper()
	ejecutarDownRechazadoGobiernoCoberturaO404B(
		t,
		ctx,
		administrador,
		archivo,
		"fuera de orden",
	)
}

func verificarDownProtegidoGobiernoCoberturaO404B(
	t *testing.T,
	ctx context.Context,
	administrador *pgxpool.Pool,
	archivo string,
) {
	t.Helper()
	ejecutarDownRechazadoGobiernoCoberturaO404B(
		t,
		ctx,
		administrador,
		archivo,
		"existe historia",
	)
}

func instalarBarreraCPosteriorGobiernoCoberturaO404B(
	t *testing.T,
	ctx context.Context,
	administrador *pgxpool.Pool,
) {
	t.Helper()
	raiz := os.Getenv(variableRaizGobiernoCoberturaO404B)
	if raiz == "" {
		t.Fatal("raíz del repositorio O4-04B no configurada")
	}
	contenido, err := os.ReadFile(filepath.Join(
		raiz,
		"deploy/postgresql/contratacion_temporal/migraciones",
		"000020_reserva_terminal_cobertura_o4_04c.up.sql",
	))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := administrador.Exec(ctx, string(contenido)); err != nil {
		t.Fatalf("instalar barrera posterior C/000020: %v", err)
	}
}

func ejecutarDownRechazadoGobiernoCoberturaO404B(
	t *testing.T,
	ctx context.Context,
	administrador *pgxpool.Pool,
	archivo string,
	mensaje string,
) {
	t.Helper()
	raiz := os.Getenv(variableRaizGobiernoCoberturaO404B)
	if raiz == "" {
		t.Fatal("raíz del repositorio O4-04B no configurada")
	}
	contenido, err := os.ReadFile(filepath.Join(
		raiz,
		"deploy/postgresql/contratacion_temporal/migraciones",
		archivo,
	))
	if err != nil {
		t.Fatal(err)
	}
	conexion, err := administrador.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conexion.Release()
	destruccionHabilitada := mensaje == "fuera de orden"
	if destruccionHabilitada {
		if _, err := conexion.Exec(ctx, `
			SELECT pg_catalog.set_config(
			  'vec.confirmar_destruccion_gobierno_cobertura_o4_04b',
			  'DESTRUIR_HISTORIA_GOBIERNO_COBERTURA_O4_04B_IRREVERSIBLE',
			  false
			)`); err != nil {
			t.Fatal(err)
		}
	}
	_, err = conexion.Exec(ctx, string(contenido))
	_, errorRollback := conexion.Exec(ctx, "ROLLBACK")
	var errorReset error
	if destruccionHabilitada {
		_, errorReset = conexion.Exec(
			ctx,
			"RESET vec.confirmar_destruccion_gobierno_cobertura_o4_04b",
		)
	}
	var errorPostgreSQL *pgconn.PgError
	if !errors.As(err, &errorPostgreSQL) ||
		errorPostgreSQL.Code != "55000" ||
		!strings.Contains(errorPostgreSQL.Message, mensaje) ||
		errorRollback != nil ||
		errorReset != nil {
		t.Fatalf(
			"down %s no falló cerrado: err=%v rollback=%v reset=%v",
			archivo,
			err,
			errorRollback,
			errorReset,
		)
	}
	var intacto bool
	if err := administrador.QueryRow(ctx, `
		SELECT
		  pg_catalog.to_regprocedure(
		    'vec_contratacion_temporal.gobi_o404b_publicar(jsonb)'
		  ) IS NOT NULL
		  AND pg_catalog.to_regclass(
		    'vec_contratacion_temporal.gobi_o404b_actual'
		  ) IS NOT NULL`).Scan(&intacto); err != nil || !intacto {
		t.Fatalf("down %s dejó estado parcial: %t / %v", archivo, intacto, err)
	}
}

func verificarBloqueoCheckpointGobiernoCoberturaO404B(
	t *testing.T,
	ctx context.Context,
	administrador *pgxpool.Pool,
	publicador *pgxpool.Pool,
	carga []byte,
) {
	t.Helper()
	bloqueador, err := administrador.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer revertirTransaccion(bloqueador)
	if _, err := bloqueador.Exec(ctx, `
		SET LOCAL ROLE vec_contratacion_temporal_propietario;
		SELECT control
		  FROM vec_contratacion_temporal.gobi_o404b_checkpoint
		 WHERE control
		 FOR UPDATE`); err != nil {
		t.Fatal(err)
	}
	_, _, _, err = ejecutarPublicacionGobiernoCoberturaO404BReal(
		ctx,
		publicador,
		carga,
	)
	var errorPostgreSQL *pgconn.PgError
	if !errors.As(err, &errorPostgreSQL) ||
		errorPostgreSQL.Code != "55P03" {
		t.Fatalf(
			"publicación no quedó cercada por FOR UPDATE: %v",
			err,
		)
	}
	if err := bloqueador.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	var secuencia int64
	if err := administrador.QueryRow(ctx, `
		SELECT ultima_secuencia
		  FROM vec_contratacion_temporal.gobi_o404b_checkpoint
		 WHERE control`).Scan(&secuencia); err != nil || secuencia != 0 {
		t.Fatalf(
			"bloqueo de checkpoint dejó efectos: secuencia=%d err=%v",
			secuencia,
			err,
		)
	}
}

func verificarSuperficieFuncionesGobiernoCoberturaO404B(
	t *testing.T,
	ctx context.Context,
	administrador *pgxpool.Pool,
) {
	t.Helper()
	var publicRevocado, searchPathSeguro, rolesMinimos bool
	err := administrador.QueryRow(ctx, `
		SELECT count(*) = 16
		       AND pg_catalog.bool_and(
		         NOT pg_catalog.has_function_privilege(
		           'public', p.oid, 'EXECUTE'
		         )
		       ),
		       pg_catalog.bool_and(
		         COALESCE(p.proconfig, ARRAY[]::text[])
		           @> ARRAY['search_path=pg_catalog']::text[]
		       ),
		       pg_catalog.bool_and(
		         pg_catalog.has_function_privilege(
		           'vec_contratacion_temporal_gobernador',
		           p.oid,
		           'EXECUTE'
		         ) = (
		           p.proname IN (
		             'gobi_o404b_publicar',
		             'gobi_o404b_retirar'
		           )
		         )
		         AND pg_catalog.has_function_privilege(
		           'vec_contratacion_temporal_ejecutor',
		           p.oid,
		           'EXECUTE'
		         ) = (p.proname = 'gobi_o404b_resolver')
		         AND NOT pg_catalog.has_function_privilege(
		           'vec_contratacion_temporal_migrador',
		           p.oid,
		           'EXECUTE'
		         )
		       )
		  FROM pg_catalog.pg_proc p
		  JOIN pg_catalog.pg_namespace n ON n.oid = p.pronamespace
		 WHERE n.nspname = 'vec_contratacion_temporal'
		   AND p.proname LIKE 'gobi_o404b_%'
		   AND p.proname <> 'gobi_o404b_revalidar_prueba_efimera'`).Scan(
		&publicRevocado,
		&searchPathSeguro,
		&rolesMinimos,
	)
	if err != nil || !publicRevocado || !searchPathSeguro || !rolesMinimos {
		t.Fatalf(
			"superficie de funciones O4-04B abierta: "+
				"public=%t search_path=%t roles=%t err=%v",
			publicRevocado,
			searchPathSeguro,
			rolesMinimos,
			err,
		)
	}
}
