#!/usr/bin/env bash
set -euo pipefail

# Restaura estructura main hasta AD3-50 y recompone T13/1-5 en PG18.4 sin red.
# No acepta la base histórica ni instala nada en cidonia.


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
limpiar(){ docker rm -f "$contenedor" >/dev/null 2>&1 || true; for tmp in "${transaccion:-}" "${roles_sql:-}"; do if [[ -n "$tmp" && -f "$tmp" ]]; then unlink "$tmp"; fi; done; }
trap limpiar EXIT INT TERM
docker run --detach --rm --network none --name "$contenedor" --env POSTGRES_HOST_AUTH_METHOD=trust "$imagen" >/dev/null
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
core_original=$(consulta "SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') FROM pg_proc WHERE oid='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure")
preimagen=$(consulta "SELECT current_setting('server_version_num')::int BETWEEN 180000 AND 189999 AND current_database() LIKE 'vec_f2_%' AND to_regnamespace('vec_bolsa_registro_accesos') IS NOT NULL AND to_regnamespace('vec_contacto_usuario_v1') IS NULL AND to_regrole('vec_contacto_usuario_owner') IS NULL AND to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL AND to_regprocedure('vec_bolsa_registro_accesos.registrar_resultado_correo_llamamiento_ct_v1(bytea,text,bytea,bytea,text,text,text,boolean,text)') IS NOT NULL")
[[ "$preimagen" == t ]] || { echo 'F2: preimagen PG18/main50 no acreditada' >&2; exit 1; }
archivos=(
  "$raiz/deploy/postgresql/contacto_usuario_vec/roles_up.sql"
  "$raiz/deploy/postgresql/autorizacion_atestada_v3/migraciones/000051_consumidor_contacto_usuario.up.sql"
  "$raiz/deploy/postgresql/bolsa_registro_accesos/migraciones/000006_registrar_contacto_usuario.up.sql"
  "$raiz/deploy/postgresql/contacto_usuario_vec/migraciones/000001_almacen_contacto_usuario.up.sql"
  "$raiz/deploy/postgresql/autorizacion_atestada_v3/migraciones/000052_consulta_recibo_contacto_propio.up.sql"
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
    if [[ "${VEC_F2_DIAGNOSTICO:-}" == 1 && "$archivo" == *000052_consulta_recibo_contacto_propio.up.sql ]]; then
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
aplicar "$raiz/deploy/postgresql/contacto_usuario_vec/pruebas_sql/ensayar_contacto_f2.sql"
aplicar "$raiz/deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/contacto_usuario_codec.sql"
aplicar "$raiz/deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/contacto_recibo_codec.sql"
[[ "$(consulta "SELECT to_regprocedure('vec_contacto_usuario_v1.consultar_version_contacto_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)') IS NOT NULL AND to_regprocedure('vec_contacto_usuario_v1.consultar_recibo_contacto_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)') IS NOT NULL AND NOT has_table_privilege('vec_contacto_usuario_writer','vec_contacto_usuario_v1.versiones','SELECT') AND NOT has_table_privilege('vec_contacto_usuario_reader','vec_contacto_usuario_v1.actual','SELECT')")" == t ]] || { echo 'F2: postimagen/ACL no acreditadas' >&2; exit 1; }
# DOWN solo con tablas F2 vacías en este contenedor desechable.
[[ "$(consulta "SELECT NOT EXISTS (SELECT 1 FROM vec_contacto_usuario_v1.versiones) AND NOT EXISTS (SELECT 1 FROM vec_bolsa_registro_accesos.registro_acceso)")" == t ]] || { echo 'F2: DOWN rechazado por historia' >&2; exit 1; }
for archivo in \
  "$raiz/deploy/postgresql/contacto_usuario_vec/migraciones/000002_consulta_recibo_contacto_propio.down.sql" \
  "$raiz/deploy/postgresql/bolsa_registro_accesos/migraciones/000007_consulta_recibo_contacto_propio.down.sql" \
  "$raiz/deploy/postgresql/autorizacion_atestada_v3/migraciones/000052_consulta_recibo_contacto_propio.down.sql" \
  "$raiz/deploy/postgresql/contacto_usuario_vec/migraciones/000001_almacen_contacto_usuario.down.sql" \
  "$raiz/deploy/postgresql/bolsa_registro_accesos/migraciones/000006_registrar_contacto_usuario.down.sql" \
  "$raiz/deploy/postgresql/autorizacion_atestada_v3/migraciones/000051_consumidor_contacto_usuario.down.sql"; do aplicar "$archivo"; done
[[ "$(consulta "SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') FROM pg_proc WHERE oid='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure")" == "$core_original" ]] || { echo 'F2: DOWN alteró núcleo previo' >&2; exit 1; }
for archivo in "${archivos[@]:1}"; do aplicar "$archivo"; done
aplicar "$raiz/deploy/postgresql/contacto_usuario_vec/pruebas_sql/ensayar_contacto_f2.sql"
echo 'F2: ROLLBACK, COMMIT, DOWN vacío y reinstalación AD3 50→51/52 acreditados en PG18 desechable; Cronos 55 pertenece a F3'
