#!/usr/bin/env bash
# Ensayo manual PostgreSQL 18.4 desechable de CRN14/C8 contra un borrador no
# instalable. Instala 000001–000010 y carga el borrador con fachadas AD146 DE
# PRUEBA, aisladas con --network none. El stub
# sólo comprueba sesión, audiencia y nonce; este arnés no acredita MAC, COSE,
# revocación, gobierno AD146, Documentos ni custodia externa.
#
# La parte funcional se completa contra la candidata final: anexo/revisión,
# CAS, replay, recuperación, ACL/RLS y saldo/permisos inmutables. No ejecutar
# sobre una instancia compartida o con historia.
set -euo pipefail

base_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
mig=$(CDPATH= cd -- "$base_dir/../migraciones" && pwd)
crn14_borrador=$(CDPATH= cd -- "$base_dir/../borradores/crn14_justificacion" && pwd)
container="vec-cronos-000014-${RANDOM}${RANDOM}"

cleanup() { docker rm -f "$container" >/dev/null 2>&1 || true; }
trap cleanup EXIT

docker run -d --network none --name "$container" \
  -e POSTGRES_HOST_AUTH_METHOD=trust postgres:18.4-alpine >/dev/null

esperar() {
  for _ in $(seq 1 80); do
    if docker exec "$container" psql -X -qAt -U postgres -d postgres -c 'SELECT 1' >/dev/null 2>&1; then
      sleep 0.3
      if docker exec "$container" psql -X -qAt -U postgres -d postgres -c 'SELECT 1' >/dev/null 2>&1; then
        return 0
      fi
    fi
    sleep 0.3
  done
  echo 'PostgreSQL no disponible' >&2
  exit 2
}

run() { docker exec -i "$container" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d postgres "$@"; }
scalar() { docker exec "$container" psql -X -qAt -v ON_ERROR_STOP=1 -U "${2:-postgres}" -d postgres -c "$1"; }
comprobar() {
  local obtenido
  obtenido=$(scalar "$1" "${4:-postgres}")
  if [ "$obtenido" != "$2" ]; then
    printf 'FALLO %s: %s (esperado %s)\n' "$3" "$obtenido" "$2" >&2
    exit 1
  fi
  printf 'OK %s\n' "$3"
}

esperar
[ "$(scalar 'SHOW server_version')" = 18.4 ] || { echo 'Se requiere PostgreSQL 18.4' >&2; exit 2; }

run < "$mig/000001_esquema_marcajes.up.sql" >/dev/null
run < "$base_dir/cronos_empleado_000007_stub_ad3.sql" >/dev/null
run < "$base_dir/cronos_empleado_000008_stub_ad3.sql" >/dev/null
run < "$base_dir/cronos_resolucion_000009_stub_ad3.sql" >/dev/null
run < "$base_dir/cronos_notificaciones_000010_stub_ad3.sql" >/dev/null
for n in \
  000002_registrar_marcaje_propio \
  000003_resultado_ejecucion_marcaje \
  000004_programacion_y_libro_saldo \
  000005_periodos_teletrabajo_y_cierre_remoto \
  000006_rls_resultado_ejecucion \
  000007_saldo_y_fichaje_remoto_empleado \
  000008_movimientos_y_permisos_empleado \
  000009_resolucion_permisos_y_avisos \
  000010_circuito_y_notificaciones_rrhh
do
  run < "$mig/$n.up.sql" >/dev/null
done

m14="$crn14_borrador/000014_justificacion.sql.borrador"
if run < "$m14" >/dev/null 2>&1; then
  echo 'FALLO 000014 aceptada sin las fachadas AD146' >&2
  exit 1
fi
printf 'OK 000014 rechazada sin consumidores nominales AD146\n'
run < "$base_dir/cronos_justificacion_000014_stub_ad146.sql" >/dev/null

# La migración debe conservar la posibilidad de rollback transaccional en el
# arnés vacío. No se ejecuta DOWN: no existe una reversión de historia.
ruta_previa=$(scalar "SELECT pg_get_constraintdef(oid) FROM pg_constraint WHERE conname='solicitud_outbox_tipo_check'")
sed '$s/^COMMIT;$/ROLLBACK;/' "$m14" | run >/dev/null
comprobar "SELECT to_regclass('vec_cronos_v1.justificacion_operacion') IS NULL" t 'ROLLBACK de 000014 sin historia C8'
comprobar "SELECT pg_get_constraintdef(oid) FROM pg_constraint WHERE conname='solicitud_outbox_tipo_check'" "$ruta_previa" 'ROLLBACK conserva tipos previos de outbox'

run < "$m14" >/dev/null
if run < "$m14" >/dev/null 2>&1; then
  echo 'FALLO segunda aplicación de 000014 aceptada' >&2
  exit 1
fi
printf 'OK segunda aplicación de 000014 rechazada\n'

# Datos y consumidores de PRUEBA exclusivamente para este contenedor. La
# capacidad fabricada conserva acción, tipo, recurso y huella coherentes con
# C8, pero no prueba criptografía, revocación ni gobierno AD146.
run < "$base_dir/cronos_empleado_000007_preparar.sql" >/dev/null
run < "$base_dir/cronos_empleado_000008_preparar.sql" >/dev/null
run < "$base_dir/cronos_resolucion_000009_preparar.sql" >/dev/null
run <<'SQL' >/dev/null
BEGIN;
SET LOCAL ROLE vec_cronos_v1_propietario;
INSERT INTO vec_cronos_v1.justificacion_politica(
  politica_ref,version,sha256,catalogo_version_ref,permiso_ref,
  tipo_documental_ref,custodio_id,motivos_ref,vigente_desde,vigente_hasta,
  sintetico,fuente_ref,publicada_en
) VALUES (
  'politica:cronos:justificacion:v1',1,repeat('c',64),
  'catalogo:cronos:traslado:v1','permiso:cronos:traslado',
  'ref:1111111111111111111111111111111111111111111111111111111111111111',
  'custodio_sintetico',ARRAY['motivo:cronos:documento-cotejado'],
  clock_timestamp()-interval '1 day',NULL,true,'fuente:sintetica:crn14',clock_timestamp()
);
INSERT INTO vec_cronos_v1.justificacion_expediente(
  solicitud_ref,empleado_ref,expediente_documental_ref,politica_ref,
  politica_version,sintetico,fuente_ref,publicada_en
) VALUES (
  'permiso:cronos:solicitud:concedido-0001','emp_AAAAAAAAAAAAAAAAAAAAAA',
  'ref:2222222222222222222222222222222222222222222222222222222222222222',
  'politica:cronos:justificacion:v1',1,true,'fuente:sintetica:crn14',clock_timestamp()
);
COMMIT;

CREATE FUNCTION prueba.v3j(
  p_audiencia text,p_accion text,p_tipo text,p_finalidad text,p_recurso text,
  p_material text,p_empleado text,p_actor text,p_perfil text,p_actor_empleado text,p_nonce text
) RETURNS TABLE(cap bytea,dcs bytea,ctx bytea)
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE huella text; ahora timestamptz:=clock_timestamp();
BEGIN
  huella:=encode(sha256(convert_to(
    '{"ambitos":{"empleado_ref":'||to_jsonb(p_empleado)::text||
    ',"solicitud_ref":'||to_jsonb(p_recurso)::text||
    '},"atributos":{"material_sha256":"'||
    encode(sha256(convert_to(p_material,'UTF8')),'hex')||'"}}','UTF8')),'hex');
  RETURN QUERY SELECT
    convert_to(jsonb_build_object('nonce',p_nonce,'decision_ref','decision:prueba:'||p_nonce,
      'efecto_ref',p_recurso,'huella_efecto_sha256',huella,'audiencia_consumo',p_audiencia,
      'operacion',p_accion,'expira_en',ahora+interval '30 seconds',
      'decision_valida_hasta',ahora+interval '30 seconds',
      'configuracion_expira_en',ahora+interval '1 day','raiz_valida_hasta',ahora+interval '1 day')::text,'UTF8'),
    convert_to(jsonb_build_object('accion',p_accion,'modulo_id','cronos','tipo_recurso',p_tipo,
      'finalidad',p_finalidad,'recurso_ref',p_recurso,'principal_id',p_actor,
      'perfil_activo_ref',p_perfil,'contexto_recurso_huella_sha256',huella,
      'valida_hasta',ahora+interval '30 seconds')::text,'UTF8'),
    convert_to(jsonb_build_object('principal_ref',p_actor,'perfil_activo_ref',p_perfil,
      'vigente_hasta',ahora+interval '1 hour','vinculos',jsonb_build_array(
        jsonb_build_object('tipo','empleado','estado','activo','referencia',p_actor_empleado,
          'vigente_desde',ahora-interval '1 day','vigente_hasta',ahora+interval '1 day')))::text,'UTF8');
END $f$;

CREATE FUNCTION prueba.material_j(p_clave text,p_accion text,p_version bigint,p_decision text DEFAULT NULL)
RETURNS text LANGUAGE plpgsql STABLE SET search_path=pg_catalog AS $f$
DECLARE documento jsonb:=jsonb_build_object('id','ref:3333333333333333333333333333333333333333333333333333333333333333',
  'version',1,'sha256',repeat('d',64),'custodio_id','custodio_sintetico','custodia_ref','custodia:sintetica:crn14');
  vinculo jsonb:=jsonb_build_object('solicitud_ref','permiso:cronos:solicitud:concedido-0001',
  'empleado_ref','emp_AAAAAAAAAAAAAAAAAAAAAA','catalogo_version_ref','catalogo:cronos:traslado:v1',
  'permiso_ref','permiso:cronos:traslado','expediente_documental_ref',
  'ref:2222222222222222222222222222222222222222222222222222222222222222','documento',documento);
BEGIN
  RETURN (jsonb_build_object('actor_ref',CASE WHEN p_accion='cronos.justificacion.revisar' THEN 'per_RRRRRRRRRRRRRRRRRRRRRR' ELSE 'per_AAAAAAAAAAAAAAAAAAAAAA' END,
    'perfil_ref',CASE WHEN p_accion='cronos.justificacion.revisar' THEN 'prf_QQQQQQQQQQQQQQQQQQQQQQ' ELSE 'prf_PPPPPPPPPPPPPPPPPPPPPP' END,
    'clave_operacion',p_clave,'accion',p_accion,'solicitud_version',2,'version_esperada',p_version,
    'politica_ref','politica:cronos:justificacion:v1','politica_version',1,'politica_sha256',repeat('c',64),
    'vinculo',vinculo) || CASE WHEN p_decision IS NULL THEN '{}'::jsonb ELSE
    jsonb_build_object('decision',p_decision,'motivo_ref','motivo:cronos:documento-cotejado') END)::text;
END $f$;

CREATE FUNCTION prueba.registro_j() RETURNS text LANGUAGE sql STABLE AS $f$
 SELECT jsonb_build_object('documento',jsonb_build_object('id','ref:3333333333333333333333333333333333333333333333333333333333333333',
   'version',1,'sha256',repeat('d',64),'custodio_id','custodio_sintetico','custodia_ref','custodia:sintetica:crn14'),
   'modulo_id','cronos','expediente_ref','ref:2222222222222222222222222222222222222222222222222222222222222222',
   'tipo_ref','ref:1111111111111111111111111111111111111111111111111111111111111111',
   'numero_vec','VEC-2026-14','creado_en_utc','2026-01-01T00:00:00.000000Z',
   'politica_ref','ref:4444444444444444444444444444444444444444444444444444444444444444','politica_version',1,
   'politica_sha256',repeat('e',64),'conservacion_hasta_utc','2030-01-01T00:00:00.000000Z',
   'proteccion','conservacion','estado_politica','provisional')::text
$f$;

CREATE FUNCTION prueba.anexar_j(p_clave text,p_nonce text) RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE m text:=prueba.material_j(p_clave,'cronos.justificacion.anexar',0); v record;
BEGIN
 SELECT * INTO v FROM prueba.v3j('vec_cronos_v1.justificacion.v1','cronos.justificacion.anexar','justificacion',
  'anexar_justificacion','permiso:cronos:solicitud:concedido-0001',m,'emp_AAAAAAAAAAAAAAAAAAAAAA',
  'per_AAAAAAAAAAAAAAAAAAAAAA','prf_PPPPPPPPPPPPPPPPPPPPPP','emp_AAAAAAAAAAAAAAAAAAAAAA',p_nonce);
 RETURN vec_cronos_v1.anexar_justificacion_v1(prueba.registro_j(),m,v.cap,v.dcs,'\\x00'::bytea,v.ctx,1,1,'\\x00'::bytea,'\\x00'::bytea,'\\x00'::bytea,'\\x00'::bytea);
END $f$;

CREATE FUNCTION prueba.revisar_j(p_clave text,p_version bigint,p_nonce text) RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE m text:=prueba.material_j(p_clave,'cronos.justificacion.revisar',p_version,'aceptada'); v record;
BEGIN
 SELECT * INTO v FROM prueba.v3j('vec_cronos_v1.justificacion.v1','cronos.justificacion.revisar','justificacion',
  'revisar_justificacion','permiso:cronos:solicitud:concedido-0001',m,'emp_AAAAAAAAAAAAAAAAAAAAAA',
  'per_RRRRRRRRRRRRRRRRRRRRRR','prf_QQQQQQQQQQQQQQQQQQQQQQ','emp_RRRRRRRRRRRRRRRRRRRRRR',p_nonce);
 RETURN vec_cronos_v1.revisar_justificacion_v1(m,v.cap,v.dcs,'\\x00'::bytea,v.ctx,1,1,'\\x00'::bytea,'\\x00'::bytea,'\\x00'::bytea,'\\x00'::bytea);
END $f$;
GRANT EXECUTE ON FUNCTION prueba.anexar_j(text,text),prueba.revisar_j(text,bigint,text) TO login_cronos_prueba;
SQL

firma='(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
for f in consultar_justificacion_v1 recuperar_recibo_justificacion_v1 revisar_justificacion_v1; do
  comprobar "SELECT has_function_privilege('vec_cronos_v1_ejecutor','vec_cronos_v1.$f$firma','EXECUTE') AND NOT has_function_privilege('vec_cronos_v1_auditor','vec_cronos_v1.$f$firma','EXECUTE') AND NOT has_function_privilege('public','vec_cronos_v1.$f$firma','EXECUTE') AND prosecdef AND proconfig @> ARRAY['search_path=pg_catalog'] FROM pg_proc WHERE oid='vec_cronos_v1.$f$firma'::regprocedure" t "ACL y search_path de $f"
done
firma_anexo='(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
comprobar "SELECT has_function_privilege('vec_cronos_v1_ejecutor','vec_cronos_v1.anexar_justificacion_v1$firma_anexo','EXECUTE') AND NOT has_function_privilege('vec_cronos_v1_auditor','vec_cronos_v1.anexar_justificacion_v1$firma_anexo','EXECUTE') AND NOT has_function_privilege('public','vec_cronos_v1.anexar_justificacion_v1$firma_anexo','EXECUTE') AND prosecdef AND proconfig @> ARRAY['search_path=pg_catalog'] FROM pg_proc WHERE oid='vec_cronos_v1.anexar_justificacion_v1$firma_anexo'::regprocedure" t 'ACL y search_path de anexar_justificacion_v1'
for t in justificacion_politica justificacion_expediente justificacion_operacion justificacion_acceso justificacion_replay; do
  comprobar "SELECT relrowsecurity AND relforcerowsecurity AND NOT has_table_privilege('vec_cronos_v1_ejecutor','vec_cronos_v1.$t','SELECT,INSERT,UPDATE,DELETE') AND NOT has_table_privilege('public','vec_cronos_v1.$t','SELECT') FROM pg_class WHERE oid='vec_cronos_v1.$t'::regclass" t "RLS FORCE y sin DML directo en $t"
done

estado_previo=$(scalar "SELECT estado||'|'||pendiente_justificar FROM vec_cronos_v1.permiso_estado WHERE solicitud_ref='permiso:cronos:solicitud:concedido-0001' ORDER BY version DESC LIMIT 1")
saldo_previo=$(scalar "SELECT count(*) FROM vec_cronos_v1.marcaje_original")
recibo_anexo=$(scalar "SELECT prueba.anexar_j('ref:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa','n-a-1')->>'recibo_ref'" login_cronos_prueba)
comprobar "SELECT prueba.anexar_j('ref:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa','n-a-2')->>'recibo_ref'" "$recibo_anexo" 'replay del anexo conserva recibo'
comprobar "SELECT prueba.revisar_j('ref:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb',1,'n-r-1')->'justificacion'->>'estado'" aceptada 'revisión acepta el vínculo pendiente' login_cronos_prueba
comprobar "SELECT prueba.espera_error(\$\$SELECT prueba.revisar_j('ref:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc',1,'n-r-2')\$\$,'PC011')" 'OK PC011' 'CAS rechaza revisión obsoleta' login_cronos_prueba
comprobar "SELECT count(*) FROM vec_cronos_v1.justificacion_operacion" 2 'anexo y revisión únicos'
comprobar "SELECT count(*) FROM vec_cronos_v1.justificacion_replay" 1 'replay auditado sin otro efecto'
comprobar "SELECT estado||'|'||pendiente_justificar FROM vec_cronos_v1.permiso_estado WHERE solicitud_ref='permiso:cronos:solicitud:concedido-0001' ORDER BY version DESC LIMIT 1" "$estado_previo" 'justificación no altera permiso'
comprobar "SELECT count(*) FROM vec_cronos_v1.marcaje_original" "$saldo_previo" 'justificación no altera hechos de saldo'

printf 'BLOQUEO DE CONFIANZA: el anexo aceptó un registro Documentos sintácticamente coherente fabricado por este arnés, sin constancia nominal de Documentos.\n' >&2
printf 'CRN14 permanece NO INSTALABLE: el stub no acredita V3 real y la candidata no enlaza el registro a una fuente confiable de Documentos.\n' >&2
exit 1
