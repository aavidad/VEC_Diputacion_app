package bootstrap

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	postgresqlcompartido "vec-diputacion-granada/internal/shared/postgresql"
	"vec-diputacion-granada/internal/vec/auditoria"
)

// Consultas que la CLI lanzaba antes de AD235 contra las tablas y preimágenes
// privadas. Solo se conservan aquí, como referencia del ensayo: las fachadas
// tienen que devolver exactamente el mismo jsonb.
const consultaInstantaneaGobiernoAnteriorAD235 = `SELECT jsonb_build_object('revision',c.revision,'secuencia',c.secuencia,'spki',encode(r.clave_publica_spki,'base64'),'clave_id',r.clave_id,'version',r.version,'audiencia',r.audiencia_despliegue,'desde',r.valida_desde,'hasta',r.valida_hasta,'orden',(SELECT max(orden) FROM vec_autorizacion_atestada_v3.puntero_clave_emision),'max_version',(SELECT max(version) FROM vec_autorizacion_atestada_v3.clave_capacidad_version),'max_revision',(SELECT max(revision_gobierno) FROM vec_autorizacion_atestada_v3.clave_capacidad_version),'pre_sha',encode(sha256(convert_to(vec_autorizacion_atestada_v3.preimagen_gobierno_usuarios_admin_v1()::text,'UTF8')),'hex')) FROM vec_autorizacion_atestada_v3.puntero_configuracion_actual p JOIN vec_autorizacion_atestada_v3.configuracion_confianza_version c ON c.revision=p.configuracion_revision JOIN vec_autorizacion_atestada_v3.configuracion_raiz cr ON cr.configuracion_revision=c.revision JOIN vec_autorizacion_atestada_v3.raiz_confianza_version r ON r.clave_id=cr.raiz_clave_id AND r.version=cr.raiz_version ORDER BY p.orden DESC LIMIT 1`

var consultaInstantaneaCapacidadesAnteriorAD235 = strings.Replace(consultaInstantaneaGobiernoAnteriorAD235,
	"vec_autorizacion_atestada_v3.preimagen_gobierno_usuarios_admin_v1()",
	"vec_autorizacion_atestada_v3.preimagen_gobierno_capacidades_admin_v1($1::integer)", 1)

const consultaCadenaGobiernoAnteriorAD235 = `WITH corte AS (SELECT secuencia n FROM vec_autorizacion_atestada_v3.control_cadena_auditoria),
cadena AS (
 SELECT a.*,a.secuencia posicion,NULL::jsonb eslabon,a.anterior_sha256 enlace_previo,a.huella_sha256 enlace
 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a,corte WHERE a.secuencia<=corte.n
 UNION ALL
 SELECT a.*,e.posicion,jsonb_build_object('posicion',e.posicion,'secuencia',e.secuencia,'anterior_sha256',e.anterior_sha256,'eslabon_sha256',e.eslabon_sha256,'registrada_en',to_char(a.registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'sellado_en',to_char(e.sellado_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')),e.anterior_sha256,e.eslabon_sha256
 FROM vec_autorizacion_atestada_v3.eslabon_auditoria_v5 e JOIN vec_autorizacion_atestada_v3.auditoria_consumo_v3 a USING (secuencia)
), rows AS (
 SELECT * FROM cadena
 WHERE tipo_registro IN ('gobierno_usuarios_admin','intento_gobierno_usuarios_admin')
 AND posicion > COALESCE((SELECT max(posicion) FROM cadena
 WHERE tipo_registro NOT IN ('gobierno_usuarios_admin','intento_gobierno_usuarios_admin')),0)
), rango AS(SELECT min(posicion) primero,max(posicion) ultimo,count(*) cuenta FROM rows)
 SELECT jsonb_build_object('esquema',$1::text,'manifiesto',jsonb_build_object('cadena_id','cadena:comun:interna','primera_secuencia',ra.primero,'ultima_secuencia',ra.ultimo,'registros',ra.cuenta,'anterior_sha256',(SELECT enlace_previo FROM rows ORDER BY posicion LIMIT 1),'cabeza_sha256',(SELECT enlace FROM rows ORDER BY posicion DESC LIMIT 1)),
 'registros',(SELECT jsonb_agg(jsonb_build_object('tipo_registro',a.tipo_registro) || CASE WHEN a.eslabon IS NULL THEN '{}'::jsonb ELSE jsonb_build_object('eslabon',a.eslabon) END ||
 CASE WHEN a.tipo_registro='gobierno_usuarios_admin' THEN jsonb_build_object('gobierno_usuarios',jsonb_build_object('auditoria_ref',a.auditoria_ref,'secuencia',a.secuencia,'anterior_sha256',a.anterior_sha256,'huella_sha256',a.huella_sha256,'registrada_en',to_char(a.registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'evento_ref',a.evento_ref,'evento_material_sha256',a.evento_material_sha256,'operador_login',a.operador_login,'accion',a.accion,'modulo_id',a.modulo_id,'recurso_ref',a.recurso_ref,'resultado',a.resultado,'motivo_ref',a.motivo_ref,'proceso',a.proceso,'canal',a.canal,'finalidad_ref',a.finalidad_ref,'correlacion_ref',a.correlacion_ref) || jsonb_build_object('plan_sha256',a.plan_sha256,'preimagen_sha256',a.gobierno_usuarios_detalle->>'preimagen_sha256','configuracion_origen_ref',a.gobierno_usuarios_detalle->>'configuracion_origen_ref','configuracion_destino_ref',a.gobierno_usuarios_detalle->>'configuracion_destino_ref','claves_sha256',a.gobierno_usuarios_detalle->>'claves_sha256'))
 ELSE jsonb_build_object('intento_gobierno_usuarios',jsonb_build_object('auditoria_ref',a.auditoria_ref,'secuencia',a.secuencia,'anterior_sha256',a.anterior_sha256,'huella_sha256',a.huella_sha256,'registrada_en',to_char(a.registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'evento_ref',a.evento_ref,'evento_material_sha256',a.evento_material_sha256,'operador_login',a.operador_login,'accion',a.accion,'modulo_id',a.modulo_id,'recurso_ref',a.recurso_ref,'resultado',a.resultado,'motivo_ref',a.motivo_ref,'proceso',a.proceso,'canal',a.canal,'finalidad_ref',a.finalidad_ref,'correlacion_ref',a.correlacion_ref) || jsonb_build_object('solicitud_sha256',a.gobierno_usuarios_solicitud_sha256)) END
 ORDER BY a.posicion) FROM rows a)) FROM rango ra`

// configuracionEnsayoLectorGobierno apunta a una copia DESECHABLE con AD235:
// un DSN superusuario (solo para la consulta anterior) y el del LOGIN lector,
// miembro únicamente de vec_autorizacion_atestada_v3_lector_gobierno.
type configuracionEnsayoLectorGobierno struct {
	DSNSuperusuario string `json:"dsn_superusuario"`
	DSNLector       string `json:"dsn_lector"`
}

// Opt-in: VEC_GOBIERNO_LECTOR_ENSAYO_CONFIG (fichero 0600). Sin él no abre PG.
func TestGobiernoLectorAD235PostgreSQLPrivado(t *testing.T) {
	ruta := os.Getenv("VEC_GOBIERNO_LECTOR_ENSAYO_CONFIG")
	if ruta == "" {
		t.Skip("ensayo_lector_no_configurado")
	}
	b, err := leerFicheroMaterialSeguro(ruta, 4096)
	if err != nil {
		t.Fatal("ensayo_config_invalida")
	}
	var cfg configuracionEnsayoLectorGobierno
	if decodificarGobiernoUsuarios(b, &cfg) != nil || cfg.DSNSuperusuario == "" || cfg.DSNLector == "" {
		t.Fatal("ensayo_config_invalida")
	}
	ctx := t.Context()
	su, err := pgxpool.New(ctx, cfg.DSNSuperusuario)
	if err != nil {
		t.Fatal("ensayo_pg_no_disponible")
	}
	defer su.Close()
	lector, err := pgxpool.New(ctx, cfg.DSNLector)
	if err != nil {
		t.Fatal("ensayo_pg_no_disponible")
	}
	defer lector.Close()

	// El LOGIN lector no es superusuario y vec-server lo admite (sin TEMP).
	var super bool
	if lector.QueryRow(ctx, `SELECT rolsuper FROM pg_catalog.pg_roles WHERE rolname=session_user`).Scan(&super) != nil || super {
		t.Fatal("lector_superusuario")
	}
	if err := postgresqlcompartido.ComprobarTEMPArranque(ctx, lector); err != nil {
		t.Fatal("preflight_temp_rechaza_lector")
	}

	// Mismo jsonb, byte a byte, para todos los conjuntos y uno inexistente.
	for conjunto := int64(0); conjunto <= 6; conjunto++ {
		consulta, args := consultaInstantaneaGobiernoAnteriorAD235, []any{}
		if conjunto != 0 {
			consulta, args = consultaInstantaneaCapacidadesAnteriorAD235, []any{conjunto}
		}
		var antes []byte
		if su.QueryRow(ctx, consulta, args...).Scan(&antes) != nil {
			t.Fatal("consulta_anterior", conjunto)
		}
		despues, err := leerDocumentoGobiernoLectura(ctx, lector, consultaInstantaneaGobiernoLectura, conjunto)
		if err != nil || !bytes.Equal(antes, despues) {
			t.Fatalf("instantanea_distinta conjunto=%d", conjunto)
		}
	}
	var antes []byte
	if su.QueryRow(ctx, consultaCadenaGobiernoAnteriorAD235, auditoria.EsquemaVerificacionGobiernoUsuarios).Scan(&antes) != nil {
		t.Fatal("cadena_anterior")
	}
	despues, err := leerDocumentoGobiernoLectura(ctx, lector, consultaCadenaGobiernoLectura, auditoria.EsquemaVerificacionGobiernoUsuarios)
	if err != nil || !bytes.Equal(antes, despues) {
		t.Fatal("cadena_distinta")
	}

	// verificar con el lector da el mismo informe que la consulta anterior.
	var docAntes auditoria.DocumentoVerificacionMixta
	if decodificarGobiernoUsuarios(antes, &docAntes) != nil {
		t.Fatal("documento_anterior")
	}
	informeAntes, _ := json.Marshal(auditoria.VerificarCadenaGobiernoUsuariosV1(docAntes, docAntes.Manifiesto, 100))
	r, _ := raizOperacionGobiernoUsuarios(t)
	informe, err := VerificarCadenaGobiernoUsuariosAdmin(ctx, lector, r, "verificacion-lector.json")
	if err != nil {
		t.Fatal("verificar_lector")
	}
	informeDespues, _ := json.Marshal(informe)
	if !bytes.Equal(informeAntes, informeDespues) {
		t.Fatal("informe_distinto")
	}
	t.Logf("cadena=%s registros=%d", informe.Estado, informe.Cobertura.Registros)

	// El lector no alcanza nada más: ni preimágenes, ni tablas, ni efectos,
	// ni tablas temporales; y las fachadas no admiten argumentos fuera de rango.
	for _, q := range []string{
		`SELECT vec_autorizacion_atestada_v3.preimagen_gobierno_capacidades_admin_v1(5)`,
		`SELECT vec_autorizacion_atestada_v3.preimagen_gobierno_usuarios_admin_v1()`,
		`SELECT count(*) FROM vec_autorizacion_atestada_v3.clave_capacidad_version`,
		`SELECT count(*) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3`,
		`SELECT vec_autorizacion_atestada_v3.aprovisionar_gobierno_capacidades_admin_v1('{}','x','{}')`,
		`INSERT INTO vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1(version,audiencias,segmentos) VALUES (999,'{a}','{b}')`,
		`CREATE TEMP TABLE lector_temp(x integer)`,
		`CREATE TABLE vec_autorizacion_atestada_v3.lector_tabla(x integer)`,
	} {
		_, err := lector.Exec(ctx, q)
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "42501" {
			t.Fatalf("lector_alcanza %q", q)
		}
	}
	for _, q := range []struct {
		sql string
		arg any
	}{{`SELECT vec_autorizacion_atestada_v3.instantanea_gobierno_admin_lectura_v1($1)`, int64(-1)}, {`SELECT vec_autorizacion_atestada_v3.cadena_gobierno_admin_lectura_v1($1)`, "Esquema Libre"}} {
		_, err := lector.Exec(ctx, q.sql, q.arg)
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "22023" {
			t.Fatalf("argumento_admitido %q", q.sql)
		}
	}
}

// Sin base: preparar y verificar solo llaman a las dos fachadas de AD235, con
// un único parámetro tipado y sin tocar tablas ni preimágenes privadas.
func TestGobiernoLectorAD235ConsultasSoloFachadas(t *testing.T) {
	esperadas := map[string]string{
		consultaInstantaneaGobiernoLectura: "SELECT vec_autorizacion_atestada_v3.instantanea_gobierno_admin_lectura_v1($1::integer)",
		consultaCadenaGobiernoLectura:      "SELECT vec_autorizacion_atestada_v3.cadena_gobierno_admin_lectura_v1($1::text)",
	}
	for q, e := range esperadas {
		if q != e || strings.Contains(q, "FROM") || strings.Contains(q, "preimagen_") {
			t.Fatal("consulta_no_es_fachada", q)
		}
	}
}

// Sin base: un pool inalcanzable da el error genérico del gobierno, sin
// detalles de la conexión, y no deja informe.
func TestGobiernoLectorAD235FalloLecturaGenerico(t *testing.T) {
	r, d := raizOperacionGobiernoUsuarios(t)
	pool := poolSinConexionGobiernoUsuarios(t)
	if _, err := leerDocumentoGobiernoLectura(t.Context(), pool, consultaCadenaGobiernoLectura, auditoria.EsquemaVerificacionGobiernoUsuarios); err != ErrGobiernoUsuariosAdmin {
		t.Fatal("error_no_generico")
	}
	if _, err := VerificarCadenaGobiernoUsuariosAdmin(t.Context(), pool, r, "verificacion.json"); err != ErrGobiernoUsuariosAdmin {
		t.Fatal("verificar_error_no_generico")
	}
	if _, err := os.Lstat(d + "/verificacion.json"); !os.IsNotExist(err) {
		t.Fatal("informe_sin_lectura")
	}
	origen := MaterialOrigenGobiernoUsuariosAdmin{DirectorioMaterial: "/x", RutaConfiguracionHMAC: "/y", ArchivoSemillaRaiz: "/z", ValidezClaves: time.Hour, ConjuntoVersion: 5}
	if _, err := PrepararGobiernoUsuariosAdmin(t.Context(), pool, origen, r, relojGobiernoUsuariosEnsayo{}); err != ErrGobiernoUsuariosAdmin {
		t.Fatal("preparar_error_no_generico")
	}
}
