#!/usr/bin/env bash
set -euo pipefail
raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm}
contenedor="vec-ct140-ad3105-$$"
datos="/tmp/${contenedor}"
limpiar() {
 docker rm -f "$contenedor" >/dev/null 2>&1 || true
 docker run --rm --network none -v /tmp:/limpiar --entrypoint rm "$imagen" -rf "/limpiar/$contenedor" >/dev/null 2>&1 || true
}
trap limpiar EXIT
mkdir -p "$datos"
docker run --detach --rm --network none --name "$contenedor" \
 -e POSTGRES_HOST_AUTH_METHOD=trust -v "$datos:/var/lib/postgresql" \
 -v "$raiz:/repo:ro" "$imagen" >/dev/null
for _ in $(seq 1 60); do
 if docker exec "$contenedor" pg_isready -q -U postgres -d postgres; then break; fi
 sleep 1
done
docker exec "$contenedor" pg_isready -q -U postgres -d postgres
psql_super() { docker exec -i "$contenedor" psql -X -v ON_ERROR_STOP=1 -U postgres -d postgres "$@"; }
psql_login() { docker exec -i "$contenedor" psql -X -v ON_ERROR_STOP=1 -U vec_ct140_login -d postgres "$@"; }
ct=/repo/deploy/postgresql/contratacion_temporal
ad3=/repo/deploy/postgresql/autorizacion_atestada_v3
psql_super -f "$ct/pruebas_sql/ct140_fixture_pg18.sql" >/dev/null
psql_super -f "$ad3/migraciones/000105_consumidor_consulta_comunicaciones_expediente_ct.up.sql" >/dev/null
psql_super -f "$ct/migraciones/000140_consulta_comunicaciones_expediente.up.sql" >/dev/null
psql_super -f "$ct/pruebas_sql/ct140_contrato_pg18.sql" >/dev/null
psql_super -c "DO \$\$ BEGIN IF (SELECT count(*) FROM vec_autorizacion_atestada_v3.prueba_consumos WHERE efecto_ref='expediente:ct140-vacio')<>1 THEN RAISE EXCEPTION 'lectura vacía sin consumo'; END IF; END \$\$" >/dev/null
psql_super -c "DO \$\$ BEGIN IF (SELECT count(*) FROM vec_autorizacion_atestada_v3.prueba_consumos WHERE efecto_ref='expediente:ct140-a')<>4 THEN RAISE EXCEPTION '404 sin consumo o 403 consumió'; END IF; END \$\$" >/dev/null
if psql_super -f "$ad3/migraciones/000105_consumidor_consulta_comunicaciones_expediente_ct.up.sql" >/dev/null 2>&1; then
 echo 'AD3-105 admitió doble UP' >&2; exit 1
fi
if psql_super -f "$ct/migraciones/000140_consulta_comunicaciones_expediente.up.sql" >/dev/null 2>&1; then
 echo 'CT140 admitió doble UP' >&2; exit 1
fi
psql_super -c "DO \$\$ BEGIN IF has_function_privilege('public','vec_contratacion_temporal.consultar_comunicaciones_expediente_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE') THEN RAISE EXCEPTION 'CT140: PUBLIC puede ejecutar'; END IF; END \$\$" >/dev/null
docker restart "$contenedor" >/dev/null
for _ in $(seq 1 60); do
 if docker exec "$contenedor" pg_isready -q -U postgres -d postgres; then break; fi
 sleep 1
done
psql_super -c "DO \$\$ BEGIN IF (SELECT count(*) FROM vec_autorizacion_atestada_v3.prueba_consumos WHERE efecto_ref='expediente:ct140-vacio')<>1 THEN RAISE EXCEPTION 'reinicio perdió consumo vacío'; END IF; END \$\$" >/dev/null
psql_login -c "BEGIN ISOLATION LEVEL SERIALIZABLE; DO \$\$ DECLARE p jsonb; BEGIN p:=public.probar_ct140('expediente:ct140-a','organizacion:ct140-a',10); IF jsonb_array_length(p->'comunicaciones')<>2 OR p->'comunicaciones'->0->>'estado_respuesta'<>'registrada' OR p->'comunicaciones'->1->>'estado_respuesta'<>'sin_respuesta' THEN RAISE EXCEPTION 'reinicio perdió estados'; END IF; END \$\$; COMMIT" >/dev/null
psql_super -c "SET ROLE vec_contratacion_temporal_propietario; INSERT INTO vec_contratacion_temporal.respuesta_recibida_rrhh SELECT 'justificante:ct140-a2','organizacion:ct140-a','expediente:ct140-a','llamamiento:ct140-a2','comunicacion:ct140-a2','10000000-0000-4000-8000-000000000001','actor:otro','perfil:rrhh-otro',2,'renuncia','recibo:respuesta-a2',r,m,'registrada_por_rrhh','2026-09-28 12:11:00+00'::timestamptz FROM (SELECT '{\"Respuesta\":\"renuncia\",\"OrganizacionRef\":\"organizacion:ct140-a\",\"ExpedienteRef\":\"expediente:ct140-a\",\"LlamamientoRef\":\"llamamiento:ct140-a2\",\"ComunicacionRef\":\"comunicacion:ct140-a2\"}'::jsonb m) s CROSS JOIN LATERAL (SELECT jsonb_build_object('Solicitud',m,'JustificanteRef','justificante:ct140-a2','ReciboRef','recibo:respuesta-a2','AuditoriaRef','auditoria:respuesta-a2','Estado','registrada_por_rrhh','RegistradaEn','2026-09-28T12:11:00.000000Z') r) z; INSERT INTO vec_contratacion_temporal.historia_respuesta_recibida_rrhh VALUES('justificante:ct140-a2','auditoria:respuesta-a2','actor:otro','perfil:rrhh-otro')" >/dev/null
psql_login -c "BEGIN ISOLATION LEVEL SERIALIZABLE; DO \$\$ DECLARE cursor_v text; BEGIN cursor_v:='comunicacion:ct140-a#'||encode(sha256(convert_to('organizacion:ct140-a'||chr(10)||'expediente:ct140-a'||chr(10)||'2'||chr(10)||'comunicacion:ct140-a=registrada'||chr(10)||'comunicacion:ct140-a2=sin_respuesta'||chr(10),'UTF8')),'hex'); BEGIN PERFORM public.probar_ct140('expediente:ct140-a','organizacion:ct140-a',1,cursor_v); RAISE EXCEPTION 'deriva de respuesta no detectada'; EXCEPTION WHEN SQLSTATE 'P1405' THEN NULL; END; END \$\$; COMMIT" >/dev/null
psql_login -c "BEGIN ISOLATION LEVEL SERIALIZABLE; DO \$\$ DECLARE p jsonb; BEGIN p:=public.probar_ct140('expediente:ct140-a','organizacion:ct140-a',10); IF p->'comunicaciones'->0->>'estado_respuesta'<>'registrada' OR p->'comunicaciones'->1->>'estado_respuesta'<>'registrada' THEN RAISE EXCEPTION 'respuesta nueva omitida'; END IF; END \$\$; COMMIT" >/dev/null
psql_super -c "SET ROLE vec_contratacion_temporal_propietario; UPDATE vec_contratacion_temporal.historia_respuesta_recibida_rrhh SET actor_ref='actor:incompatible' WHERE justificante_ref='justificante:ct140-a2'" >/dev/null
psql_login -c "BEGIN ISOLATION LEVEL SERIALIZABLE; DO \$\$ BEGIN BEGIN PERFORM public.probar_ct140('expediente:ct140-a','organizacion:ct140-a',10); RAISE EXCEPTION 'historia inconsistente produjo estado'; EXCEPTION WHEN SQLSTATE 'P1405' THEN NULL; END; END \$\$; COMMIT" >/dev/null
psql_super -c "SET ROLE vec_contratacion_temporal_propietario; UPDATE vec_contratacion_temporal.historia_respuesta_recibida_rrhh SET actor_ref='actor:otro' WHERE justificante_ref='justificante:ct140-a2'" >/dev/null
psql_super -c "SET ROLE vec_contratacion_temporal_propietario; INSERT INTO vec_contratacion_temporal.comunicacion_llamamiento_local SELECT 'comunicacion:ct140-before','organizacion:ct140-a','expediente:ct140-a','llamamiento:ct140-a','10000000-0000-4000-8000-000000000001',2,m,r,'registrada_localmente','2026-09-28 11:59:00+00'::timestamptz FROM (SELECT jsonb_build_object('solicitud',jsonb_build_object('OrganizacionRef','organizacion:ct140-a','ExpedienteRef','expediente:ct140-a','LlamamientoRef','llamamiento:ct140-a','PruebaEntregaRef','recibo:seleccion-a')) m) s CROSS JOIN LATERAL (SELECT jsonb_build_object('ComunicacionRef','comunicacion:ct140-before','ReciboRef','recibo:ct140-before','Solicitud',m->'solicitud','Estado','registrada_localmente','RegistradaEn','2026-09-28 11:59:00+00'::timestamptz) r) z" >/dev/null
psql_login -c "BEGIN ISOLATION LEVEL SERIALIZABLE; DO \$\$ DECLARE cursor_v text; BEGIN cursor_v:='comunicacion:ct140-a#'||encode(sha256(convert_to('organizacion:ct140-a'||chr(10)||'expediente:ct140-a'||chr(10)||'2'||chr(10)||'comunicacion:ct140-a=registrada'||chr(10)||'comunicacion:ct140-a2=registrada'||chr(10),'UTF8')),'hex'); BEGIN PERFORM public.probar_ct140('expediente:ct140-a','organizacion:ct140-a',1,cursor_v); RAISE EXCEPTION 'deriva de comunicación no detectada'; EXCEPTION WHEN SQLSTATE 'P1405' THEN NULL; END; END \$\$; COMMIT" >/dev/null
psql_super -c "SET ROLE vec_contratacion_temporal_propietario; INSERT INTO vec_contratacion_temporal.respuesta_recibida_rrhh SELECT 'justificante:ct140-duplicado','organizacion:ct140-a','expediente:ct140-a','llamamiento:ct140-a','comunicacion:ct140-a','10000000-0000-4000-8000-000000000001','actor:otro','perfil:rrhh-otro',2,'aceptacion','recibo:respuesta-duplicada',r,m,'registrada_por_rrhh','2026-09-28 12:12:00+00'::timestamptz FROM (SELECT '{\"Respuesta\":\"aceptacion\",\"OrganizacionRef\":\"organizacion:ct140-a\",\"ExpedienteRef\":\"expediente:ct140-a\",\"LlamamientoRef\":\"llamamiento:ct140-a\",\"ComunicacionRef\":\"comunicacion:ct140-a\"}'::jsonb m) s CROSS JOIN LATERAL (SELECT jsonb_build_object('Solicitud',m,'JustificanteRef','justificante:ct140-duplicado','ReciboRef','recibo:respuesta-duplicada','AuditoriaRef','auditoria:respuesta-duplicada','Estado','registrada_por_rrhh','RegistradaEn','2026-09-28T12:12:00.000000Z') r) z; INSERT INTO vec_contratacion_temporal.historia_respuesta_recibida_rrhh VALUES('justificante:ct140-duplicado','auditoria:respuesta-duplicada','actor:otro','perfil:rrhh-otro')" >/dev/null
psql_login -c "BEGIN ISOLATION LEVEL SERIALIZABLE; DO \$\$ BEGIN BEGIN PERFORM public.probar_ct140('expediente:ct140-a','organizacion:ct140-a',10); RAISE EXCEPTION 'multiplicidad semántica aceptada'; EXCEPTION WHEN SQLSTATE 'P1405' THEN NULL; END; END \$\$; COMMIT" >/dev/null
printf 'CT140/AD3-105 PG18 sintético: estado semántico, páginas, deriva de respuesta/comunicación, ACL, 403/404/503, doble UP y reinicio OK; V3 es doble, no criptografía real\n'
