#!/usr/bin/env bash
set -euo pipefail

# Restaura estructura main hasta AD3-50 y recompone T13/1-5 en PG18.4 sin red.
# No acepta la base histórica ni instala nada en cidonia.
# El corte Contacto3 revoca el POST directo. Su ensayo final debe acreditar
# juntos: denegación ACL del legado, cinco rutas con ServicioOperaciones/V3,
# recibo original, replay, concurrencia y recuperación tras reiniciar PG/app.
# No instalar Contacto3 en un runtime que aún publique sólo el POST directo.


: "${VEC_F2_SCHEMA_SQL:?falta copia local schema-only F2}"
: "${VEC_F2_ROLES_TXT:?falta lista local de roles F2}"
: "${VEC_F2_SCHEMA_SHA256:?falta huella fijada de schema F2}"
: "${VEC_F2_ROLES_SHA256:?falta huella fijada de roles F2}"
: "${VEC_F2_ROLES_ACL_TSV:?falta inventario saneado roles/ACL}"
: "${VEC_F2_ROLES_ACL_SHA256:?falta huella inventario roles/ACL}"
[[ -s "$VEC_F2_SCHEMA_SQL" && -s "$VEC_F2_ROLES_TXT" && -s "$VEC_F2_ROLES_ACL_TSV" ]] || { echo 'F2: preimagen vacía' >&2; exit 1; }
[[ "$(sha256sum "$VEC_F2_SCHEMA_SQL" | cut -d' ' -f1)" == "$VEC_F2_SCHEMA_SHA256" && "$(sha256sum "$VEC_F2_ROLES_TXT" | cut -d' ' -f1)" == "$VEC_F2_ROLES_SHA256" && "$(sha256sum "$VEC_F2_ROLES_ACL_TSV" | cut -d' ' -f1)" == "$VEC_F2_ROLES_ACL_SHA256" ]] || { echo 'F2: huellas de preimagen divergentes' >&2; exit 1; }
[[ "$(stat -c %a "$VEC_F2_SCHEMA_SQL")" == 600 && "$(stat -c %a "$VEC_F2_ROLES_TXT")" == 600 && "$(stat -c %a "$VEC_F2_ROLES_ACL_TSV")" == 600 ]] || { echo 'F2: preimagen debe tener modo 0600' >&2; exit 1; }
head -n 9 "$VEC_F2_SCHEMA_SQL" | grep -q 'Dumped from database version 18.4' || { echo 'F2: dump no procede de PG18.4' >&2; exit 1; }
if rg -q '^(COPY |INSERT INTO |\\copy )' "$VEC_F2_SCHEMA_SQL" || rg -qv '^vec_[a-z0-9_]+$' "$VEC_F2_ROLES_TXT"; then
  echo 'F2: entradas contienen datos o roles inválidos' >&2; exit 1
fi
raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)
contenedor="vec-f2-pg18-$$"
base="vec_f2_$$"
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}
limpiar(){ docker rm -f "$contenedor" >/dev/null 2>&1 || true; for tmp in "${transaccion:-}" "${roles_sql:-}" "${prueba_json:-}" "${rol_ct_copia:-}" "${neg_rol_log:-}"; do if [[ -n "$tmp" && -f "$tmp" ]]; then unlink "$tmp"; fi; done; if [[ -n "${socketdir:-}" ]]; then rm -f "$socketdir/.s.PGSQL.5432" "$socketdir/.s.PGSQL.5432.lock"; rmdir "$socketdir" 2>/dev/null || true; fi; if [[ -n "${materialdir:-}" ]]; then rm -rf -- "$materialdir"; fi; }
trap limpiar EXIT INT TERM
volumen=()
if [[ "${VEC_F2_POSITIVO:-}" == 1 ]]; then
  socketdir=$(mktemp -d)
  chmod 0777 "$socketdir"
  volumen=(--volume "$socketdir:/var/run/postgresql")
  materialdir=$(mktemp -d)
  rmdir "$materialdir"
  "$raiz/scripts/generar_credenciales_desarrollo.sh" "$materialdir" >/dev/null 2>&1 || { echo 'F2: material efímero sintético no disponible' >&2; exit 1; }
fi
docker run --detach --rm --network none --name "$contenedor" --env POSTGRES_HOST_AUTH_METHOD=trust "${volumen[@]}" "$imagen" >/dev/null
for _ in $(seq 1 60); do docker exec "$contenedor" pg_isready -q -U postgres -d postgres && break; sleep 0.5; done
docker exec "$contenedor" pg_isready -q -U postgres -d postgres
docker exec "$contenedor" createdb -U postgres "$base"
consulta(){ docker exec "$contenedor" psql -X -A -t -q --set ON_ERROR_STOP=1 --username postgres --dbname "$base" --command "$1"; }
aplicar(){ printf "F2 ensaya %s\n" "$(basename "$1")" >&2; docker exec -i "$contenedor" psql -X -q --set ON_ERROR_STOP=1 --username postgres --dbname "$base" < "$1"; }
roles_sql=$(mktemp)
python3 - "$VEC_F2_ROLES_ACL_TSV" "$base" > "$roles_sql" <<'PYROLES'
import re,sys
filas=open(sys.argv[1]).read().splitlines();base=sys.argv[2]
if not re.fullmatch(r'vec_f2_[0-9]+',base): raise SystemExit('base F2 inválida')
roles=set();membresias=[];acl=[]
for fila in filas:
 p=fila.split('|')
 if p[0]=='ROLE' and len(p)==5:
  _,nombre,login,herencia,limite=p
  if not re.fullmatch(r'vec_[a-z0-9_]+',nombre) or login not in ('t','f') or herencia not in ('t','f') or not re.fullmatch(r'-?[0-9]+',limite):raise SystemExit('atributos rol inválidos')
  roles.add(nombre);print(f'CREATE ROLE {nombre} NOLOGIN {"INHERIT" if herencia=="t" else "NOINHERIT"};')
 elif p[0]=='MEMBER' and len(p)==6:membresias.append(p)
 elif p[0]=='DBACL' and len(p)==5:acl.append(p)
 else:raise SystemExit('inventario saneado inválido')
for _,miembro,rol,admin,herencia,establecer in membresias:
 if miembro not in roles or rol not in roles or any(v not in ('t','f') for v in (admin,herencia,establecer)):raise SystemExit('membresía inválida')
 print(f'GRANT {rol} TO {miembro} WITH ADMIN {"TRUE" if admin=="t" else "FALSE"}, INHERIT {"TRUE" if herencia=="t" else "FALSE"}, SET {"TRUE" if establecer=="t" else "FALSE"};')
for _,bd,rol,privilegio,otorgable in acl:
 if bd!='postgres' or rol not in roles or privilegio not in ('CONNECT','CREATE','TEMPORARY') or otorgable!='f':raise SystemExit('ACL de base inválida')
 print(f'GRANT {privilegio} ON DATABASE {base} TO {rol};')
PYROLES
aplicar "$roles_sql"
aplicar "$VEC_F2_SCHEMA_SQL"
# Cidonia no tiene T13; se recompone desde fuentes Git antes del delta F2.
aplicar "$raiz/deploy/postgresql/bolsa_registro_accesos/roles_up.sql"
if [[ "$(consulta "SELECT to_regprocedure('vec_autorizacion.revalidar_decision_registro_accesos_bolsa_v2(jsonb,bytea,bytea,text,text,text,jsonb,text,text,text)') IS NULL")" == t ]]; then
  aplicar "$raiz/deploy/postgresql/bolsa_registro_accesos/migraciones_autorizacion/000001_revalidacion_registro_accesos_v2.up.sql"
fi
aplicar "$raiz/deploy/postgresql/bolsa_registro_accesos/migraciones/000001_registro_accesos_t13.up.sql"
# T13 no existe en cidonia: la dependencia AD3 se instala con migración
# productiva T13/4, con guardas y reversión solo en ensayo vacío.
aplicar "$raiz/deploy/postgresql/bolsa_registro_accesos/migraciones/000004_preparar_resultado_correo_ad3.up.sql"
aplicar "$raiz/deploy/postgresql/bolsa_registro_accesos/migraciones/000004_preparar_resultado_correo_ad3.down.sql"
aplicar "$raiz/deploy/postgresql/bolsa_registro_accesos/migraciones/000004_preparar_resultado_correo_ad3.up.sql"
aplicar "$raiz/deploy/postgresql/bolsa_registro_accesos/cerrar_acl_dba.sql"
aplicar "$raiz/deploy/postgresql/bolsa_registro_accesos/migraciones/000005_registrar_resultado_correo.up.sql"
# CT51 exige el rol productivo CT, conservado en el commit revisado. Se ejecuta
# una copia de SU blob exacto; nunca un GRANT diagnóstico ni una ruta mutable.
rol_ct_commit=6058a11dae0579f0a8babcf7c545e965db096624
rol_ct_ruta=deploy/postgresql/contratacion_temporal/roles_consultor_rrhh_ambito_up.sql
rol_ct_blob=130744d492fd005379d299d104a926e154190792
rol_ct_sha=2ca2191eab2c23346efd027f5b7c39d3b6f07e4487953774a39300ba12b8399f
[[ "$(git -C "$raiz" rev-parse "$rol_ct_commit:$rol_ct_ruta" 2>/dev/null)" == "$rol_ct_blob" ]] || { echo 'F2: falta procedencia exacta del rol CT' >&2; exit 1; }
rol_ct_copia=$(mktemp)
git -C "$raiz" show "$rol_ct_commit:$rol_ct_ruta" > "$rol_ct_copia"
[[ "$(git -C "$raiz" hash-object "$rol_ct_copia")" == "$rol_ct_blob" && "$(sha256sum "$rol_ct_copia" | cut -d' ' -f1)" == "$rol_ct_sha" ]] || { echo 'F2: copia del rol CT divergente' >&2; exit 1; }
if [[ -r "$raiz/$rol_ct_ruta" ]]; then
  [[ "$(sha256sum "$raiz/$rol_ct_ruta" | cut -d' ' -f1)" == "$rol_ct_sha" ]] || { echo 'F2: rol CT integrado divergente' >&2; exit 1; }
fi
legacy_pre=$(consulta "SELECT jsonb_build_object('rol',to_jsonb(r),'membresias',coalesce((SELECT jsonb_agg(to_jsonb(m) ORDER BY m.member,m.roleid) FROM pg_auth_members m WHERE m.roleid=r.oid OR m.member=r.oid),'[]'::jsonb),'esquema',to_jsonb(n))::text FROM pg_roles r,pg_namespace n WHERE r.rolname='vec_contratacion_temporal_consultor_rrhh' AND n.nspname='vec_contratacion_temporal'")
[[ -n "$legacy_pre" && "$(consulta "SELECT to_regrole('vec_contratacion_temporal_consultor_rrhh_ambito') IS NULL")" == t ]] || { echo 'F2: preimagen de rol CT incompatible' >&2; exit 1; }
# Negativo real: el UP productivo debe abortar por 55000 si el grupo ya existe.
consulta "CREATE ROLE vec_contratacion_temporal_consultor_rrhh_ambito NOLOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS" >/dev/null
neg_rol_log=$(mktemp)
if docker exec -i "$contenedor" psql -X -q --set ON_ERROR_STOP=1 --set VERBOSITY=verbose --username postgres --dbname "$base" < "$rol_ct_copia" > "$neg_rol_log" 2>&1 || ! rg -q '55000' "$neg_rol_log"; then
  echo 'F2: UP de rol CT aceptó preexistencia o no devolvió 55000' >&2; exit 1
fi
[[ "$(consulta "SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_contratacion_temporal_consultor_rrhh_ambito') AND NOT EXISTS (SELECT 1 FROM pg_auth_members WHERE roleid='vec_contratacion_temporal_consultor_rrhh_ambito'::regrole OR member='vec_contratacion_temporal_consultor_rrhh_ambito'::regrole)")" == t ]] || { echo 'F2: negativo CT cambió membresías' >&2; exit 1; }
consulta "DROP ROLE vec_contratacion_temporal_consultor_rrhh_ambito" >/dev/null
aplicar "$rol_ct_copia"
legacy_post=$(consulta "SELECT jsonb_build_object('rol',to_jsonb(r),'membresias',coalesce((SELECT jsonb_agg(to_jsonb(m) ORDER BY m.member,m.roleid) FROM pg_auth_members m WHERE m.roleid=r.oid OR m.member=r.oid),'[]'::jsonb),'esquema',to_jsonb(n))::text FROM pg_roles r,pg_namespace n WHERE r.rolname='vec_contratacion_temporal_consultor_rrhh' AND n.nspname='vec_contratacion_temporal'")
[[ "$legacy_post" == "$legacy_pre" && "$(consulta "SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_contratacion_temporal_consultor_rrhh_ambito' AND NOT rolcanlogin AND rolinherit AND NOT rolsuper AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls) AND NOT EXISTS (SELECT 1 FROM pg_auth_members WHERE member='vec_contratacion_temporal_consultor_rrhh_ambito'::regrole OR roleid='vec_contratacion_temporal_consultor_rrhh_ambito'::regrole) AND NOT has_schema_privilege('vec_contratacion_temporal_consultor_rrhh_ambito','vec_contratacion_temporal','USAGE') AND NOT has_database_privilege('vec_contratacion_temporal_consultor_rrhh_ambito',current_database(),'CREATE') AND NOT has_database_privilege('vec_contratacion_temporal_consultor_rrhh_ambito',current_database(),'TEMP')")" == t ]] || { echo 'F2: postimagen de rol CT o legado divergente' >&2; exit 1; }
aplicar "$raiz/deploy/postgresql/autorizacion_atestada_v3/migraciones/000051_consulta_rrhh_ambito_productiva.up.sql"
core_original=$(consulta "SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') FROM pg_proc WHERE oid='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure")
preimagen=$(consulta "SELECT current_setting('server_version_num')::int BETWEEN 180000 AND 189999 AND current_database() LIKE 'vec_f2_%' AND to_regnamespace('vec_bolsa_registro_accesos') IS NOT NULL AND to_regnamespace('vec_contacto_usuario_v1') IS NULL AND to_regrole('vec_contacto_usuario_owner') IS NULL AND to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea)') IS NOT NULL AND to_regprocedure('vec_bolsa_registro_accesos.registrar_resultado_correo_llamamiento_ct_v1(bytea,text,bytea,bytea,text,text,text,boolean,text)') IS NOT NULL AND (SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') FROM pg_proc WHERE oid='vec_autorizacion_atestada_v3.consumir_consulta_rrhh_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure)='6baef6127627ce9d6e6146c9d5425d7463f1a89b10411aac70fa70ed1944fd98'")
[[ "$preimagen" == t ]] || { echo 'F2: preimagen PG18/main50 no acreditada' >&2; exit 1; }
archivos=(
  "$raiz/deploy/postgresql/contacto_usuario_vec/roles_up.sql"
  "$raiz/deploy/postgresql/autorizacion_atestada_v3/migraciones/000052_consumidor_contacto_usuario.up.sql"
  "$raiz/deploy/postgresql/bolsa_registro_accesos/migraciones/000006_registrar_contacto_usuario.up.sql"
  "$raiz/deploy/postgresql/contacto_usuario_vec/migraciones/000001_almacen_contacto_usuario.up.sql"
  "$raiz/deploy/postgresql/autorizacion_atestada_v3/migraciones/000053_consulta_recibo_contacto_propio.up.sql"
  "$raiz/deploy/postgresql/bolsa_registro_accesos/migraciones/000007_consulta_recibo_contacto_propio.up.sql"
  "$raiz/deploy/postgresql/contacto_usuario_vec/migraciones/000002_consulta_recibo_contacto_propio.up.sql"
)
if [[ "${VEC_F2_DIAGNOSTICO:-}" == 1 ]]; then
  consulta "SELECT pg_get_constraintdef(oid,true) FROM pg_constraint WHERE conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass AND conname='clave_capacidad_version_audiencia_consumo_check'" >&2
fi
for archivo in "${archivos[@]}"; do [[ -r "$archivo" ]] || { echo 'F2: falta migración' >&2; exit 1; }; done
transaccion=$(mktemp)
{
  echo 'BEGIN;'
  for archivo in "${archivos[@]}"; do
    printf '%s\n' "\\echo F2 ROLLBACK $(basename "$archivo")"
    if [[ "${VEC_F2_DIAGNOSTICO:-}" == 1 && "$archivo" == *000053_consulta_recibo_contacto_propio.up.sql ]]; then
      echo "SELECT 'CORE_PRE='||encode(sha256(convert_to(prosrc,'UTF8')),'hex') FROM pg_proc WHERE oid='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;"
      echo "SELECT 'REVALIDACION_PRE='||encode(sha256(convert_to(prosrc,'UTF8')),'hex') FROM pg_proc WHERE oid='vec_autorizacion_atestada_v3.revalidar_consumo_consulta_rrhh_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;"
    fi
    sed '/^BEGIN;$/d;/^COMMIT;$/d' "$archivo"
    echo 'RESET ROLE;'
  done
  echo 'ROLLBACK;'
} > "$transaccion"
aplicar "$transaccion"
[[ "$(consulta "SELECT to_regnamespace('vec_contacto_usuario_v1') IS NULL AND to_regrole('vec_contacto_usuario_owner') IS NULL")" == t ]] || { echo 'F2: ROLLBACK dejó efectos' >&2; exit 1; }
for archivo in "${archivos[@]}"; do aplicar "$archivo"; done
# El ensayo de ACL usa una identidad LOGIN nominal, no SET ROLE desde DBA.
preparar_login(){ consulta "CREATE ROLE vec_contacto_f2_login LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS; GRANT vec_contacto_usuario_writer TO vec_contacto_f2_login WITH ADMIN FALSE, INHERIT TRUE, SET FALSE" >/dev/null; }
retirar_login(){ consulta "REVOKE vec_contacto_usuario_writer FROM vec_contacto_f2_login; DROP ROLE vec_contacto_f2_login" >/dev/null; }
preparar_login
aplicar "$raiz/deploy/postgresql/contacto_usuario_vec/pruebas_sql/ensayar_contacto_f2.sql"
aplicar "$raiz/deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/contacto_usuario_codec.sql"
aplicar "$raiz/deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/contacto_recibo_codec.sql"
[[ "$(consulta "SELECT to_regprocedure('vec_contacto_usuario_v1.consultar_version_contacto_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)') IS NOT NULL AND to_regprocedure('vec_contacto_usuario_v1.consultar_recibo_contacto_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)') IS NOT NULL AND NOT has_table_privilege('vec_contacto_usuario_writer','vec_contacto_usuario_v1.versiones','SELECT') AND NOT has_table_privilege('vec_contacto_usuario_reader','vec_contacto_usuario_v1.actual','SELECT')")" == t ]] || { echo 'F2: postimagen/ACL no acreditadas' >&2; exit 1; }
if [[ "${VEC_F2_POSITIVO:-}" == 1 ]]; then
  aplicar "$raiz/deploy/postgresql/contexto_actor_v1/pruebas_sql/fixtures_sinteticos.sql"
  aplicar "$raiz/deploy/postgresql/contacto_usuario_vec/pruebas_sql/ensayar_operaciones_f2.sql"
  consulta "CREATE ROLE vec_contacto_f2_contexto_login LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS; GRANT vec_contexto_actor_v1_runtime TO vec_contacto_f2_contexto_login WITH ADMIN FALSE, INHERIT TRUE, SET FALSE; GRANT CONNECT ON DATABASE $base TO vec_contacto_f2_contexto_login" >/dev/null
  consulta "CREATE ROLE vec_contacto_f2_fuente_login LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS; GRANT vec_autorizacion_fuente TO vec_contacto_f2_fuente_login WITH ADMIN FALSE, INHERIT TRUE, SET FALSE; GRANT CONNECT ON DATABASE $base TO vec_contacto_f2_fuente_login" >/dev/null
  consulta "CREATE ROLE vec_contacto_f2_registro_login LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS; GRANT vec_autorizacion_registro TO vec_contacto_f2_registro_login WITH ADMIN FALSE, INHERIT TRUE, SET FALSE; GRANT CONNECT ON DATABASE $base TO vec_contacto_f2_registro_login" >/dev/null
  consulta "CREATE ROLE vec_contacto_f2_motivos_login LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS; GRANT vec_autorizacion_motivos_evaluador TO vec_contacto_f2_motivos_login WITH ADMIN FALSE, INHERIT TRUE, SET FALSE; GRANT CONNECT ON DATABASE $base TO vec_contacto_f2_motivos_login" >/dev/null
  docker exec --user root "$contenedor" chmod 0777 /var/run/postgresql
  dsn(){ printf 'host=%s port=5432 dbname=%s user=%s sslmode=disable' "$socketdir" "$base" "$1"; }
  cd "$raiz"
  prueba_json=$(mktemp)
  if ! VEC_F2_CONTACTO_PG18_DESECHABLE=1 \
  VEC_F2_CONTACTO_MATERIAL_EFIMERO="$materialdir" \
  VEC_F2_CONTACTO_PG_ADMIN_DSN="$(dsn postgres)" \
  VEC_F2_CONTACTO_PG_CONTEXTO_DSN="$(dsn vec_contacto_f2_contexto_login)" \
  VEC_F2_CONTACTO_PG_FUENTE_DSN="$(dsn vec_contacto_f2_fuente_login)" \
  VEC_F2_CONTACTO_PG_REGISTRO_DSN="$(dsn vec_contacto_f2_registro_login)" \
  VEC_F2_CONTACTO_PG_MOTIVOS_DSN="$(dsn vec_contacto_f2_motivos_login)" \
  VEC_F2_CONTACTO_PG_WRITER_DSN="$(dsn vec_contacto_f2_login)" \
  GOMAXPROCS=2 go test ./internal/app/bootstrap -run '^TestContactoPropioPG18MaterialFirmadoYConsumoNominal$' -count=1 -json > "$prueba_json" 2>&1; then
    echo 'F2: prueba positiva PG18 falló; log efímero privado retenido sólo durante este proceso' >&2
    exit 1
  fi
  python3 - "$prueba_json" <<'PYRESULTADO'
import json,sys
objetivo='TestContactoPropioPG18MaterialFirmadoYConsumoNominal'
paso=omito=False
for linea in open(sys.argv[1],encoding='utf-8'):
    try: evento=json.loads(linea)
    except ValueError: continue
    if evento.get('Test')==objetivo:
        paso |= evento.get('Action')=='pass'
        omito |= evento.get('Action')=='skip'
if not paso or omito: raise SystemExit('F2: prueba positiva omitida o sin PASS; no acreditar consumo')
PYRESULTADO
  echo 'F2: fase legado firmada y consumo nominal probados; aún no acredita Contacto3'
  archivos_operaciones=(
    "$raiz/deploy/postgresql/autorizacion_atestada_v3/migraciones/000054_operaciones_contacto_propio.up.sql"
    "$raiz/deploy/postgresql/bolsa_registro_accesos/migraciones/000008_operaciones_contacto_propio.up.sql"
    "$raiz/deploy/postgresql/contacto_usuario_vec/migraciones/000003_operaciones_contacto_propio.up.sql"
  )
  for archivo in "${archivos_operaciones[@]}"; do aplicar "$archivo"; done
  if ! VEC_F2_CONTACTO_OPERACIONES_PG18_DESECHABLE=1 \
  VEC_F2_CONTACTO_MATERIAL_EFIMERO="$materialdir" \
  VEC_F2_CONTACTO_PG_ADMIN_DSN="$(dsn postgres)" \
  VEC_F2_CONTACTO_PG_CONTEXTO_DSN="$(dsn vec_contacto_f2_contexto_login)" \
  VEC_F2_CONTACTO_PG_FUENTE_DSN="$(dsn vec_contacto_f2_fuente_login)" \
  VEC_F2_CONTACTO_PG_REGISTRO_DSN="$(dsn vec_contacto_f2_registro_login)" \
  VEC_F2_CONTACTO_PG_MOTIVOS_DSN="$(dsn vec_contacto_f2_motivos_login)" \
  VEC_F2_CONTACTO_PG_WRITER_DSN="$(dsn vec_contacto_f2_login)" \
  GOMAXPROCS=2 go test ./internal/app/bootstrap -run '^TestContactoOperacionesPG18RecuperacionNominal$' -count=1 -json > "$prueba_json" 2>&1; then
    echo 'F2: prueba Contacto3 PG18 falló; no acreditar operaciones' >&2
    exit 1
  fi
  python3 - "$prueba_json" <<'PYOPERACIONES'
import json,sys
objetivo='TestContactoOperacionesPG18RecuperacionNominal'
paso=omito=False
for linea in open(sys.argv[1],encoding='utf-8'):
    try: evento=json.loads(linea)
    except ValueError: continue
    if evento.get('Test')==objetivo:
        paso |= evento.get('Action')=='pass'
        omito |= evento.get('Action')=='skip'
if not paso or omito: raise SystemExit('F2: operaciones omitidas o sin PASS')
PYOPERACIONES
  referencia_reinicio=$(consulta "SELECT operacion_ref||'|'||recibo_ref FROM vec_contacto_usuario_v1.operaciones WHERE estado='confirmada' AND version_esperada=1")
  [[ "$referencia_reinicio" =~ ^opr_[A-Za-z0-9_-]{22,128}\|acc_[0-9a-f]{40}$ ]] || { echo 'F2: recibo único previo al reinicio no acreditado' >&2; exit 1; }
  IFS='|' read -r operacion_reinicio recibo_reinicio <<< "$referencia_reinicio"
  docker restart --time 5 "$contenedor" >/dev/null
  for _ in $(seq 1 60); do docker exec "$contenedor" pg_isready -q -U postgres -d "$base" && break; sleep 0.5; done
  docker exec "$contenedor" pg_isready -q -U postgres -d "$base"
  if ! VEC_F2_CONTACTO_OPERACIONES_PG18_DESECHABLE=1 \
  VEC_F2_CONTACTO_RECUPERACION_PG18=1 \
  VEC_F2_CONTACTO_RECUPERACION_OP_REF="$operacion_reinicio" \
  VEC_F2_CONTACTO_RECUPERACION_RECIBO_REF="$recibo_reinicio" \
  VEC_F2_CONTACTO_MATERIAL_EFIMERO="$materialdir" \
  VEC_F2_CONTACTO_PG_ADMIN_DSN="$(dsn postgres)" \
  VEC_F2_CONTACTO_PG_CONTEXTO_DSN="$(dsn vec_contacto_f2_contexto_login)" \
  VEC_F2_CONTACTO_PG_FUENTE_DSN="$(dsn vec_contacto_f2_fuente_login)" \
  VEC_F2_CONTACTO_PG_REGISTRO_DSN="$(dsn vec_contacto_f2_registro_login)" \
  VEC_F2_CONTACTO_PG_MOTIVOS_DSN="$(dsn vec_contacto_f2_motivos_login)" \
  VEC_F2_CONTACTO_PG_WRITER_DSN="$(dsn vec_contacto_f2_login)" \
  GOMAXPROCS=2 go test ./internal/app/bootstrap -run '^TestContactoOperacionesPG18RecuperacionNominal$' -count=1 -json > "$prueba_json" 2>&1; then
    echo 'F2: recuperación nominal tras reinicio PG18 falló' >&2
    exit 1
  fi
  python3 - "$prueba_json" <<'PYREINICIO'
import json,sys
objetivo='TestContactoOperacionesPG18RecuperacionNominal'
paso=omito=False
for linea in open(sys.argv[1],encoding='utf-8'):
    try: evento=json.loads(linea)
    except ValueError: continue
    if evento.get('Test')==objetivo:
        paso |= evento.get('Action')=='pass'
        omito |= evento.get('Action')=='skip'
if not paso or omito: raise SystemExit('F2: recuperación tras reinicio omitida o sin PASS')
PYREINICIO
  echo 'F2: Contacto3 firmado/consumido, ACL legado, replay, concurrencia y recuperación nominal tras reinicio PG18 acreditados'
  exit 0
fi
# DOWN solo con tablas F2 vacías en este contenedor desechable.
[[ "$(consulta "SELECT NOT EXISTS (SELECT 1 FROM vec_contacto_usuario_v1.versiones) AND NOT EXISTS (SELECT 1 FROM vec_bolsa_registro_accesos.registro_acceso)")" == t ]] || { echo 'F2: DOWN rechazado por historia' >&2; exit 1; }
retirar_login
for archivo in \
  "$raiz/deploy/postgresql/contacto_usuario_vec/migraciones/000002_consulta_recibo_contacto_propio.down.sql" \
  "$raiz/deploy/postgresql/bolsa_registro_accesos/migraciones/000007_consulta_recibo_contacto_propio.down.sql" \
  "$raiz/deploy/postgresql/autorizacion_atestada_v3/migraciones/000053_consulta_recibo_contacto_propio.down.sql" \
  "$raiz/deploy/postgresql/contacto_usuario_vec/migraciones/000001_almacen_contacto_usuario.down.sql" \
  "$raiz/deploy/postgresql/bolsa_registro_accesos/migraciones/000006_registrar_contacto_usuario.down.sql" \
  "$raiz/deploy/postgresql/autorizacion_atestada_v3/migraciones/000052_consumidor_contacto_usuario.down.sql"; do aplicar "$archivo"; done
[[ "$(consulta "SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') FROM pg_proc WHERE oid='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure")" == "$core_original" ]] || { echo 'F2: DOWN alteró núcleo previo' >&2; exit 1; }
for archivo in "${archivos[@]:1}"; do aplicar "$archivo"; done
preparar_login
aplicar "$raiz/deploy/postgresql/contacto_usuario_vec/pruebas_sql/ensayar_contacto_f2.sql"
echo 'F2: ROLLBACK, COMMIT, DOWN vacío, reinstalación y ACL LOGIN CT51→Contacto52/53 acreditados en PG18 desechable; Cronos pertenece a F3'
