package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// funcionPostimagenPersonalDietas fija una función de Personal que la
// composición de Dietas consume o que protege su frontera. La huella es el
// SHA-256 hexadecimal de pg_proc.prosrc tal como lo deja la migración canónica;
// la prueba focal la recalcula desde el fichero de migración para impedir que
// constante y fuente diverjan en silencio.
type funcionPostimagenPersonalDietas struct {
	migracion   string
	firma       string
	secdef      bool
	lenguaje    string
	config      string
	huella      string
	concedidas  string // grupos con EXECUTE además del propietario, en orden C
	soloFirma   bool   // prerrequisito de orden: existe, propietario y ACL; sin huella
	opcional    bool   // ampliación no consumida aún: puede faltar; si existe, exacta
	fichero     string // fichero canónico de la migración (para la prueba focal)
	nombreCorto string
}

const propietarioPostimagenPersonalDietas = "vec_personal_propietario"

// postimagenPersonalDietas es la lista positiva exacta. Personal 000010/000011
// sólo se exigen como prerrequisito de orden de 000012 (Base de organización):
// Dietas no las consume y su texto evolucionó antes de instalarse, por lo que
// se acredita firma, propietario y ACL, no su huella.
var postimagenPersonalDietas = []funcionPostimagenPersonalDietas{
	{migracion: "000007", nombreCorto: "resolver_relacion_dietas_v1", firma: "vec_personal.resolver_relacion_dietas_v1(text,text,text,date)", secdef: true, lenguaje: "plpgsql", config: "lock_timeout=2s,row_security=on,search_path=pg_catalog", huella: "7add7a9f10ea6d04d35376f7cc78f95b60e2908d26875ea2944c635cb1290805", fichero: "000007_relacion_empleado_dietas.up.sql"},
	{migracion: "000007", nombreCorto: "revalidar_relacion_dietas_v1", firma: "vec_personal.revalidar_relacion_dietas_v1(text,text,text,text,text,text,bigint,text,text,bigint,date)", secdef: true, lenguaje: "plpgsql", config: "lock_timeout=2s,row_security=on,search_path=pg_catalog", huella: "f306ea38197f4e45a0f4617c39cd38cc2bbc81490da57813fa1603a130cdea2d", concedidas: "vec_dietas_propietario", fichero: "000007_relacion_empleado_dietas.up.sql"},
	{migracion: "000008", nombreCorto: "consultar_relaciones_propias_dietas_v1", firma: "vec_personal.consultar_relaciones_propias_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)", secdef: true, lenguaje: "plpgsql", config: "lock_timeout=2s,row_security=on,search_path=pg_catalog", huella: "969c5b265046d8b306a95c2e16efdebcc9595ab0eb7bbf57c142e47f64ba7606", concedidas: "vec_dietas_ejecutor", fichero: "000008_consulta_relaciones_propias_dietas.up.sql"},
	{migracion: "000010", nombreCorto: "consultar_organizacion_historica_v1", firma: "vec_personal.consultar_organizacion_historica_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)", soloFirma: true, concedidas: "vec_personal_ejecutor"},
	{migracion: "000011", nombreCorto: "ejecutar_importacion_organizacion_v1", firma: "vec_personal.ejecutar_importacion_organizacion_v1(text,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)", soloFirma: true, concedidas: "vec_personal_ejecutor"},
	{migracion: "000012", nombreCorto: "ejecutar_asignacion_dietas_interna_v1", firma: "vec_personal.ejecutar_asignacion_dietas_interna_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)", secdef: true, lenguaje: "plpgsql", config: "lock_timeout=2s,row_security=on,search_path=pg_catalog", huella: "6284b96d87e9da622071bd8f12233713f382c3a7b9807b928fd1b82e9be36849", fichero: "000012_asignacion_dietas.up.sql"},
	{migracion: "000012", nombreCorto: "revalidar_asignacion_dietas_v1", firma: "vec_personal.revalidar_asignacion_dietas_v1(text,text,text,text,bigint,smallint,text,text,text,date)", secdef: true, lenguaje: "plpgsql", config: "lock_timeout=2s,row_security=on,search_path=pg_catalog", huella: "5df47bd11e07c3d990c7280a74e4cc088ff0f3f94a19643794c3140cc9576104", concedidas: "vec_dietas_propietario", fichero: "000012_asignacion_dietas.up.sql"},
	{migracion: "000012", nombreCorto: "registrar_asignacion_dietas_inicial_v1", firma: "vec_personal.registrar_asignacion_dietas_inicial_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)", secdef: true, lenguaje: "sql", config: "lock_timeout=2s,row_security=on,search_path=pg_catalog", huella: "3fabebe589c5d83f585f0d7682861af3ba9cb8e0baebc45104627e285f4bb267", concedidas: "vec_personal_d7_ejecutor", fichero: "000012_asignacion_dietas.up.sql"},
	{migracion: "000012", nombreCorto: "consultar_asignacion_dietas_v1", firma: "vec_personal.consultar_asignacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)", secdef: true, lenguaje: "sql", config: "lock_timeout=2s,row_security=on,search_path=pg_catalog", huella: "4ec7cbfa29f7259231e8e083bb7f74b043c8005bf5e5755a1e98942f5f8fea6d", concedidas: "vec_personal_d7_ejecutor", fichero: "000012_asignacion_dietas.up.sql"},
	{migracion: "000012", nombreCorto: "corregir_asignacion_dietas_v1", firma: "vec_personal.corregir_asignacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)", secdef: true, lenguaje: "sql", config: "lock_timeout=2s,row_security=on,search_path=pg_catalog", huella: "c3f08bb26b6a88878513f0aeea7dc662c9011575a5bfd6df25bb16a612270a30", concedidas: "vec_personal_d7_ejecutor", fichero: "000012_asignacion_dietas.up.sql"},
	{migracion: "000012", nombreCorto: "corregir_grupo_dieta_v1", firma: "vec_personal.corregir_grupo_dieta_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)", secdef: true, lenguaje: "sql", config: "lock_timeout=2s,row_security=on,search_path=pg_catalog", huella: "ed9e453cf2dac52d244f558e2856b6fa7a0d064c0d21c1bccf2187e3be2904b0", concedidas: "vec_personal_d7_ejecutor", fichero: "000012_asignacion_dietas.up.sql"},
	{migracion: "000013", nombreCorto: "registrar_auditoria_frontera_asignacion_dietas_v1", firma: "vec_personal.registrar_auditoria_frontera_asignacion_dietas_v1(text,text,text,text,text,text,text,smallint)", secdef: true, lenguaje: "plpgsql", config: "TimeZone=UTC,lock_timeout=1s,row_security=on,search_path=pg_catalog,statement_timeout=2s", huella: "b127dce774c3a3051d7ce46d30aea5477084391d0db06fa43154ceb830ecdf04", concedidas: "vec_personal_registrador_frontera", fichero: "000013_auditoria_frontera_asignacion_dietas.up.sql"},
	// Ampliaciones D7b/D7c: la composición actual no las consume, pero si están
	// instaladas sus concesiones a D7 y auditoría deben ser exactamente éstas.
	{migracion: "000014", nombreCorto: "consultar_competencias_asignacion_dietas_v1", firma: "vec_personal.consultar_competencias_asignacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)", secdef: true, lenguaje: "plpgsql", config: "lock_timeout=2s,row_security=on,search_path=pg_catalog,statement_timeout=5s", huella: "2eebb77b967f6ceaad471b78e8c9be69725312a02f655ac913f0f3357fc872b4", concedidas: "vec_personal_d7_ejecutor", fichero: "000014_competencias_asignacion_dietas.up.sql", opcional: true},
	{migracion: "000015", nombreCorto: "registrar_auditoria_frontera_rectificacion_dietas_v1", firma: "vec_personal.registrar_auditoria_frontera_rectificacion_dietas_v1(text,text,text,text,text,text,text,integer)", secdef: true, lenguaje: "plpgsql", config: "TimeZone=UTC,lock_timeout=1s,row_security=on,search_path=pg_catalog,statement_timeout=2s", huella: "3ee1a4ad23aed75348aa464dc519d3fb0b18db125cd5acc7b9330798fa238980", concedidas: "vec_personal_registrador_frontera", fichero: "000015_solicitud_rectificacion_dietas.up.sql", opcional: true},
	{migracion: "000015", nombreCorto: "ejecutar_rectificacion_dietas_interna_v1", firma: "vec_personal.ejecutar_rectificacion_dietas_interna_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)", secdef: true, lenguaje: "plpgsql", config: "lock_timeout=2s,row_security=on,search_path=pg_catalog", huella: "a9beba7f2340b443388968dfa6a9d63a28b8f17a78c7ed5246a9faa43a3db736", concedidas: "", fichero: "000015_solicitud_rectificacion_dietas.up.sql", opcional: true},
	{migracion: "000015", nombreCorto: "solicitar_rectificacion_dietas_v1", firma: "vec_personal.solicitar_rectificacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)", secdef: true, lenguaje: "sql", config: "lock_timeout=2s,row_security=on,search_path=pg_catalog", huella: "68b4b3e7d188da3f150941a7a1293c45bc3685dfc1d39b8e0ccdf8ad4e3ba1f7", concedidas: "vec_personal_d7_ejecutor", fichero: "000015_solicitud_rectificacion_dietas.up.sql", opcional: true},
	{migracion: "000015", nombreCorto: "consultar_rectificacion_dietas_v1", firma: "vec_personal.consultar_rectificacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)", secdef: true, lenguaje: "sql", config: "lock_timeout=2s,row_security=on,search_path=pg_catalog", huella: "f5c47926d905973cc1ceec59060f1362592ba8dd5f705094eca3329b2fe881b2", concedidas: "vec_personal_d7_ejecutor", fichero: "000015_solicitud_rectificacion_dietas.up.sql", opcional: true},
	{migracion: "000015", nombreCorto: "resolver_rectificacion_dietas_v1", firma: "vec_personal.resolver_rectificacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)", secdef: true, lenguaje: "sql", config: "lock_timeout=2s,row_security=on,search_path=pg_catalog", huella: "d69e006c47abecbe505529bd9358075bb2ea1c8c6e9d80e818e0bc3cb5bf4e29", concedidas: "vec_personal_d7_ejecutor", fichero: "000015_solicitud_rectificacion_dietas.up.sql", opcional: true},
	{migracion: "000015", nombreCorto: "consultar_rectificaciones_competentes_dietas_v1", firma: "vec_personal.consultar_rectificaciones_competentes_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)", secdef: true, lenguaje: "plpgsql", config: "lock_timeout=2s,row_security=on,search_path=pg_catalog,statement_timeout=5s", huella: "6fc2385256f55ee898c71f3d61f11e2c02efa8f86453430c840cf8ca79f5dfe6", concedidas: "vec_personal_d7_ejecutor", fichero: "000015_solicitud_rectificacion_dietas.up.sql", opcional: true},
	{migracion: "000015", nombreCorto: "confirmar_rectificacion_dietas_v1", firma: "vec_personal.confirmar_rectificacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)", secdef: true, lenguaje: "sql", config: "lock_timeout=2s,row_security=on,search_path=pg_catalog", huella: "2add0a4708272a0b1b8786a098e234ea3639d5dbb373cb5ceec5a87151a0899d", concedidas: "vec_personal_d7_ejecutor", fichero: "000015_solicitud_rectificacion_dietas.up.sql", opcional: true},
}

// Tablas de Personal que respaldan las funciones anteriores: propiedad del
// propietario de Personal, RLS forzada y ninguna concesión directa.
var tablasPostimagenPersonalDietas = []string{
	"vec_personal.relacion_empleado_dietas",
	"vec_personal.recibo_consulta_relacion_propia_dietas",
	"vec_personal.evidencia_consulta_relacion_propia_dietas",
	"vec_personal.asignacion_dietas",
	"vec_personal.recibo_asignacion_dietas",
	"vec_personal.evidencia_asignacion_dietas",
	"vec_personal.auditoria_frontera_asignacion_dietas",
}

// Grupos técnicos con LOGIN nominal en la composición de Dietas. Fuera de la
// lista positiva no pueden tener EXECUTE en Personal ni privilegio alguno
// sobre sus tablas o secuencias.
var gruposFronteraPostimagenPersonalDietas = []string{
	"vec_dietas_ejecutor",
	"vec_dietas_registrador_frontera",
	"vec_personal_d7_ejecutor",
	"vec_personal_registrador_frontera",
}

// Grupos que Personal 000012/000013 crean o que Dietas hereda de sus roles
// base: deben existir, sin LOGIN y sin pertenecer a otro rol. El registrador
// de frontera de Dietas lo acredita su propio adaptador.
var gruposExigidosPostimagenPersonalDietas = []string{
	"vec_dietas_ejecutor",
	"vec_personal_d7_ejecutor",
	"vec_personal_registrador_frontera",
}

// La consulta devuelve la lista de discrepancias; vacía significa acreditada.
// Cada entrada identifica la migración y el objeto, nunca datos ni DSN.
const consultaPostimagenPersonalDietas = `
WITH esperadas AS (
 SELECT e->>'migracion' AS migracion, e->>'firma' AS firma, (e->>'secdef')::boolean AS secdef,
        e->>'lenguaje' AS lenguaje, e->>'config' AS config, e->>'huella' AS huella,
        e->>'concedidas' AS concedidas, (e->>'solo_firma')::boolean AS solo_firma,
        (e->>'opcional')::boolean AS opcional
   FROM pg_catalog.jsonb_array_elements($1::jsonb) AS e
), observadas AS (
 SELECT x.*, p.oid AS proc_oid, p.proowner, p.prosecdef, l.lanname,
        (SELECT pg_catalog.string_agg(c, ',' ORDER BY c COLLATE "C") FROM pg_catalog.unnest(p.proconfig) AS c) AS config_real,
        pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p.prosrc,'UTF8')),'hex') AS huella_real,
        (SELECT pg_catalog.string_agg(g.nombre, ',' ORDER BY g.nombre COLLATE "C")
           FROM (SELECT CASE WHEN a.grantee=0 THEN 'PUBLIC' ELSE a.grantee::regrole::text END AS nombre
                   FROM pg_catalog.aclexplode(COALESCE(p.proacl, pg_catalog.acldefault('f', p.proowner))) AS a
                  WHERE a.grantee<>p.proowner) AS g) AS concedidas_real,
        EXISTS (SELECT 1 FROM pg_catalog.aclexplode(COALESCE(p.proacl, pg_catalog.acldefault('f', p.proowner))) AS a
                 WHERE a.is_grantable) AS con_opcion
   FROM esperadas x
   LEFT JOIN pg_catalog.pg_proc p ON p.oid=pg_catalog.to_regprocedure(x.firma)
   LEFT JOIN pg_catalog.pg_language l ON l.oid=p.prolang
), fallos AS (
 SELECT migracion||' '||firma||': ausente' AS f FROM observadas WHERE proc_oid IS NULL AND NOT opcional
 UNION ALL
 SELECT migracion||' '||firma||': propietario distinto' FROM observadas
  WHERE proc_oid IS NOT NULL AND proowner IS DISTINCT FROM pg_catalog.to_regrole($2)
 UNION ALL
 SELECT migracion||' '||firma||': ACL distinta' FROM observadas
  WHERE proc_oid IS NOT NULL AND (COALESCE(concedidas_real,'') IS DISTINCT FROM COALESCE(concedidas,'') OR con_opcion)
 UNION ALL
 SELECT migracion||' '||firma||': SECURITY DEFINER, lenguaje o configuracion distintos' FROM observadas
  WHERE proc_oid IS NOT NULL AND NOT solo_firma
    AND (prosecdef IS DISTINCT FROM secdef OR lanname IS DISTINCT FROM lenguaje OR COALESCE(config_real,'') IS DISTINCT FROM config)
 UNION ALL
 SELECT migracion||' '||firma||': huella distinta' FROM observadas
  WHERE proc_oid IS NOT NULL AND NOT solo_firma AND huella_real IS DISTINCT FROM huella
 UNION ALL
 SELECT 'tabla '||t||': ausente, propietario, RLS o ACL distintos'
   FROM pg_catalog.unnest($3::text[]) AS t
   LEFT JOIN pg_catalog.pg_class c ON c.oid=pg_catalog.to_regclass(t)
  WHERE c.oid IS NULL OR c.relowner IS DISTINCT FROM pg_catalog.to_regrole($2)
     OR NOT c.relrowsecurity OR NOT c.relforcerowsecurity
     OR EXISTS (SELECT 1 FROM pg_catalog.aclexplode(c.relacl) AS a WHERE a.grantee<>c.relowner)
 UNION ALL
 SELECT 'grupo '||g||': ausente, LOGIN o miembro de otro rol'
   FROM pg_catalog.unnest($5::text[]) AS g
   LEFT JOIN pg_catalog.pg_roles r ON r.rolname=g
  WHERE r.oid IS NULL OR r.rolcanlogin OR r.rolsuper OR r.rolbypassrls OR r.rolcreaterole OR r.rolcreatedb
     OR EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.member=r.oid)
 UNION ALL
 SELECT 'grupo '||r.rolname||': EXECUTE ajeno en '||p.oid::regprocedure::text
   FROM pg_catalog.pg_proc p
   CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(p.proacl, pg_catalog.acldefault('f', p.proowner))) AS a
   JOIN pg_catalog.pg_roles r ON r.oid=a.grantee
  WHERE p.pronamespace=pg_catalog.to_regnamespace('vec_personal')
    AND r.rolname=ANY($4::text[])
    AND NOT EXISTS (SELECT 1 FROM esperadas x
                     WHERE pg_catalog.to_regprocedure(x.firma)=p.oid
                       AND r.rolname=ANY(pg_catalog.string_to_array(COALESCE(x.concedidas,''),',')))
 UNION ALL
 SELECT 'EXECUTE a PUBLIC en '||p.oid::regprocedure::text
   FROM pg_catalog.pg_proc p
   CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(p.proacl, pg_catalog.acldefault('f', p.proowner))) AS a
  WHERE p.pronamespace=pg_catalog.to_regnamespace('vec_personal') AND a.grantee=0
 UNION ALL
 SELECT 'privilegio directo de '||CASE WHEN a.grantee=0 THEN 'PUBLIC' ELSE a.grantee::regrole::text END||' en '||c.oid::regclass::text
   FROM pg_catalog.pg_class c
   CROSS JOIN LATERAL pg_catalog.aclexplode(c.relacl) AS a
  WHERE c.relnamespace=pg_catalog.to_regnamespace('vec_personal')
    AND (a.grantee=0 OR a.grantee IN (SELECT oid FROM pg_catalog.pg_roles WHERE rolname=ANY($4::text[])))
)
SELECT COALESCE((SELECT pg_catalog.array_agg(f ORDER BY f COLLATE "C") FROM fallos), ARRAY[]::text[])`

// maxFallosPostimagenPersonalDietas acota el diagnóstico en el registro.
const maxFallosPostimagenPersonalDietas = 12

type consultorPostimagenPersonalDietas interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

// acreditarPostimagenPersonalDietas comprueba, con el pool de lectura de
// relaciones (LOGIN de vec_dietas_ejecutor), que la base destino contiene la
// postimagen exacta de Personal 000007–000013 que Dietas consume: firmas,
// propietario, SECURITY DEFINER, configuración, huella del cuerpo y ACL
// positiva; y que ningún grupo técnico de la frontera Dietas ni PUBLIC tiene
// privilegios fuera de esa lista. Falla cerrado y enumera lo que falta.
func acreditarPostimagenPersonalDietas(ctx context.Context, consultor consultorPostimagenPersonalDietas) error {
	if ctx == nil || consultor == nil || poolPostgreSQLDietasDesarrolloNulo(consultor) || ctx.Err() != nil {
		return errPostimagenPersonalDietasNoAcreditada
	}
	especificacion, err := especificacionPostimagenPersonalDietas()
	if err != nil {
		return errPostimagenPersonalDietasNoAcreditada
	}
	sonda, cancelar := context.WithTimeout(ctx, 10*time.Second)
	defer cancelar()
	var fallos []string
	if err := consultor.QueryRow(sonda, consultaPostimagenPersonalDietas, especificacion,
		propietarioPostimagenPersonalDietas, tablasPostimagenPersonalDietas,
		gruposFronteraPostimagenPersonalDietas, gruposExigidosPostimagenPersonalDietas).Scan(&fallos); err != nil {
		return fmt.Errorf("%w: catalogo PostgreSQL no consultable", errPostimagenPersonalDietasNoAcreditada)
	}
	if len(fallos) == 0 {
		return nil
	}
	resto := ""
	if len(fallos) > maxFallosPostimagenPersonalDietas {
		resto = fmt.Sprintf("; y %d mas", len(fallos)-maxFallosPostimagenPersonalDietas)
		fallos = fallos[:maxFallosPostimagenPersonalDietas]
	}
	return fmt.Errorf("%w: %s%s", errPostimagenPersonalDietasNoAcreditada, strings.Join(fallos, "; "), resto)
}

func especificacionPostimagenPersonalDietas() (string, error) {
	type entrada struct {
		Migracion  string `json:"migracion"`
		Firma      string `json:"firma"`
		Secdef     bool   `json:"secdef"`
		Lenguaje   string `json:"lenguaje"`
		Config     string `json:"config"`
		Huella     string `json:"huella"`
		Concedidas string `json:"concedidas"`
		SoloFirma  bool   `json:"solo_firma"`
		Opcional   bool   `json:"opcional"`
	}
	entradas := make([]entrada, 0, len(postimagenPersonalDietas))
	for _, f := range postimagenPersonalDietas {
		entradas = append(entradas, entrada{f.migracion, f.firma, f.secdef, f.lenguaje, f.config, f.huella, f.concedidas, f.soloFirma, f.opcional})
	}
	contenido, err := json.Marshal(entradas)
	return string(contenido), err
}
