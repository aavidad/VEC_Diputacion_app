#!/usr/bin/env bash
# Ensayo F4b en PostgreSQL 18.4 desechable sin red, con TLS de CA sintética.
# Uso: VEC_F4B_TEST_BASE=<directorio_privado_0700> bash probar_pg18.sh
set -euo pipefail
umask 077
: "${VEC_F4B_TEST_BASE:?directorio privado 0700 fuera de Git requerido}"
base_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
repo_dir="$(cd "$base_dir/../../.." && pwd -P)"
test_base="$(realpath -e "$VEC_F4B_TEST_BASE")"
case "$test_base/" in "$repo_dir/"*) echo 'base de prueba dentro de Git' >&2; exit 2;; esac
[[ "$(stat -c '%a' "$test_base")" == 700 && "$(stat -c '%u' "$test_base")" == "$(id -u)" ]] || exit 2
test_dir="$(mktemp -d "$test_base/f4b-pg18.XXXXXXXX")"
container="f4b-dietas-pg18-$$"
trap 'docker rm -f "$container" >/dev/null 2>&1 || true' EXIT

# CA y certificado de servidor sintéticos (SAN localhost) para verify-full.
openssl req -x509 -newkey rsa:2048 -nodes -days 2 -subj '/CN=vec-f4b-ca-prueba' \
  -keyout "$test_dir/ca.key" -out "$test_dir/ca.crt" >/dev/null 2>&1
openssl req -newkey rsa:2048 -nodes -subj '/CN=localhost' \
  -keyout "$test_dir/servidor.key" -out "$test_dir/servidor.csr" >/dev/null 2>&1
printf 'subjectAltName=DNS:localhost\n' > "$test_dir/san.ext"
openssl x509 -req -in "$test_dir/servidor.csr" -CA "$test_dir/ca.crt" -CAkey "$test_dir/ca.key" \
  -CAcreateserial -days 2 -extfile "$test_dir/san.ext" -out "$test_dir/servidor.crt" >/dev/null 2>&1
openssl req -x509 -newkey rsa:2048 -nodes -days 2 -subj '/CN=vec-f4b-ca-ajena' \
  -keyout "$test_dir/ajena.key" -out "$test_dir/ajena.crt" >/dev/null 2>&1

docker run --rm --network none -d --name "$container" \
  -e POSTGRES_HOST_AUTH_METHOD=trust \
  -v "$repo_dir:$repo_dir:ro" -v "$test_dir:$test_dir:rw" \
  postgres:18.4 >/dev/null
for _ in $(seq 1 60); do
  if docker exec "$container" pg_isready -q -h /var/run/postgresql -U postgres; then break; fi
  sleep 0.5
done
sleep 1
docker exec "$container" pg_isready -q -h /var/run/postgresql -U postgres

cat > "$test_dir/podman" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
[[ "${1:-}" == exec && "${2:-}" == -i && "${3:-}" == "$VEC_F4B_TEST_CONTAINER" ]] || exit 2
exec docker exec -i -u root "$VEC_F4B_TEST_CONTAINER" "${@:4}"
SH
# La sonda TLS usa psql del contenedor: misma red interna, TCP a localhost.
cat > "$test_dir/psql" <<'SH'
#!/usr/bin/env bash
exec docker exec -i -u root -e PGPASSFILE="${PGPASSFILE:-}" "$VEC_F4B_TEST_CONTAINER" psql "$@"
SH
chmod 700 "$test_dir/podman" "$test_dir/psql"
export VEC_F4B_TEST_CONTAINER="$container" PATH="$test_dir:$PATH"
psql_admin() { docker exec -i -u root "$container" psql -h /var/run/postgresql -XAtq -v ON_ERROR_STOP=1 -U postgres -d postgres "$@"; }
run_p6() {
  env -u PGSERVICE -u PGSERVICEFILE -u PGPASSFILE -u PGHOST -u PGHOSTADDR \
    VEC_P6_POSTGRES_CONTAINER="$container" VEC_P6_PG_SOCKET_DIR=/var/run/postgresql \
    VEC_P6_PG_PORT=5432 VEC_P6_EVIDENCIA_DIR="$test_dir" VEC_P6_APLICAR=SI-P6-REVISADO \
    bash "$repo_dir/deploy/principal/p6_retirada_dietas/ejecutar.sh" "$@"
}
estado="$test_dir/estado/f4b-estado.json"
mkdir -m 700 "$test_dir/estado"
run_f4b() {
  env -u PGSERVICE -u PGSERVICEFILE -u PGPASSFILE -u PGHOST -u PGHOSTADDR \
    VEC_F4B_POSTGRES_CONTAINER="$container" VEC_F4B_PG_SOCKET_DIR=/var/run/postgresql \
    VEC_F4B_PG_PORT=5432 VEC_F4B_EVIDENCIA_DIR="$test_dir" VEC_F4B_APLICAR=SI-F4B-REVISADO \
    VEC_F4B_ESTADO="$estado" VEC_F4B_TLS_HOST=localhost VEC_F4B_TLS_PORT=5432 \
    VEC_F4B_TLS_CA="${F4B_CA:-$test_dir/ca.crt}" \
    bash "$base_dir/ejecutar.sh" "$@"
}
debe_fallar() {
  local etiqueta=$1 mensaje=$2; shift 2
  if run_f4b "$@" > "$test_dir/$etiqueta.out" 2> "$test_dir/$etiqueta.err"; then
    echo "F4b aceptó caso negativo: $etiqueta" >&2; exit 1
  fi
  rg -q -- "$mensaje" "$test_dir/$etiqueta.err" || { echo "mensaje inesperado en $etiqueta" >&2; cat "$test_dir/$etiqueta.err" >&2; exit 1; }
}
sin_efecto() {
  [[ "$(psql_admin -c "SELECT (SELECT count(*) FROM pg_roles WHERE rolname IN ('vec_dietas_f4b_auditoria_frontera_desarrollo','vec_personal_d7_asignacion','vec_personal_d7_auditoria_frontera'))||'|'||(SELECT count(*) FROM pg_roles WHERE rolcanlogin AND rolname ~ '^vec_(dietas_r1d|dietas_f4b|personal_d7)_')||'|'||(SELECT count(*) FROM vec_autorizacion.asignacion_perfil WHERE version=3)")" == '0|0|0' ]] || {
    echo "caso negativo dejó efectos: $1" >&2; exit 1;
  }
}

# TLS del servidor y pg_hba: las once cuentas solo por hostssl con SCRAM.
docker exec -u root "$container" sh -c "mkdir -p /var/lib/postgresql/tls && cp '$test_dir/servidor.crt' '$test_dir/servidor.key' /var/lib/postgresql/tls/ && chown -R postgres:postgres /var/lib/postgresql/tls && chmod 600 /var/lib/postgresql/tls/servidor.key"
hba="$(psql_admin -c 'SHOW hba_file')"
cuentas=(vec_dietas_r1d_registro_identidad_desarrollo vec_dietas_r1d_revalidacion_identidad_desarrollo
  vec_dietas_r1d_contexto_desarrollo vec_dietas_r1d_fuente_autorizacion_desarrollo
  vec_dietas_r1d_registro_autorizacion_desarrollo vec_dietas_r1d_motivos_desarrollo
  vec_dietas_r1d_dietas_desarrollo vec_dietas_r1d_personal_desarrollo
  vec_dietas_f4b_auditoria_frontera_desarrollo vec_personal_d7_asignacion vec_personal_d7_auditoria_frontera)
lista="$(IFS=,; echo "${cuentas[*]}")"
docker exec -u root "$container" sh -c "{ printf 'hostssl all %s all scram-sha-256\nhost all %s all reject\n' '$lista' '$lista'; cat '$hba'; } > /tmp/hba && cat /tmp/hba > '$hba'"
psql_admin <<'SQL' >/dev/null
ALTER SYSTEM SET ssl = on;
ALTER SYSTEM SET ssl_cert_file = '/var/lib/postgresql/tls/servidor.crt';
ALTER SYSTEM SET ssl_key_file = '/var/lib/postgresql/tls/servidor.key';
SELECT pg_reload_conf();
SQL
sleep 1
[[ "$(psql_admin -c 'SHOW ssl')" == on ]] || { echo 'TLS del servidor de prueba no activo' >&2; exit 1; }

# Preimagen: V3 con R1D y P6 aplicada, como en F4.
psql_admin -f "$repo_dir/deploy/postgresql/autorizacion/roles_up.sql" >/dev/null
psql_admin -f "$repo_dir/deploy/postgresql/autorizacion/migraciones/000001_autorizacion.up.sql" >/dev/null
python3 "$repo_dir/deploy/principal/p6_retirada_dietas/fixture_pg18.py" > "$test_dir/fixture.sql"
psql_admin -f "$test_dir/fixture.sql" >/dev/null
psql_admin <<'SQL' >/dev/null
BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
INSERT INTO vec_autorizacion.control_vigencia_version_rol
(version_rol_ref,revision,estado,huella_sha256,actualizado_en,documento) VALUES
('rol:dietas_r1d_provisional:v1',1,'habilitada',repeat('0',64),'2026-09-23T20:00:00Z',
 '{"version_rol_ref":"rol:dietas_r1d_provisional:v1","revision":1,"estado":"habilitada","actualizado_por":"desarrollo:dietas-r1d:provisional","actualizado_en":"2026-09-23T20:00:00Z"}'::jsonb);
INSERT INTO vec_autorizacion.control_vigencia_version_rol_actual
(version_rol_ref,revision,actualizada_en,actualizada_por,acto_ref) VALUES
('rol:dietas_r1d_provisional:v1',1,'2026-09-23T20:00:00Z','desarrollo:dietas-r1d:provisional','acto:fixture:control');
COMMIT;
SQL
run_p6 --commit > "$test_dir/p6.out"
# R1D concedía CONNECT por grupo y dejaba contraseña en cada cuenta: se
# reproduce esa preimagen y se guardan las contraseñas en el estado de prueba.
python3 - "$test_dir/r1d-contrasenas.json" > "$test_dir/r1d-contrasenas.sql" <<'PY'
import json, secrets, sys
nombres = ["registro_identidad", "revalidacion_identidad", "contexto", "fuente_autorizacion",
           "registro_autorizacion", "motivos", "dietas", "personal"]
c = {f"vec_dietas_r1d_{n}_desarrollo": secrets.token_urlsafe(36) for n in nombres}
json.dump(c, open(sys.argv[1], "w"))
for k, v in c.items():
    print(f"REVOKE CONNECT ON DATABASE postgres FROM {k}; ALTER ROLE {k} PASSWORD '{v}';")
PY
psql_admin -f "$test_dir/r1d-contrasenas.sql" >/dev/null
psql_admin <<'SQL' >/dev/null
GRANT CONNECT ON DATABASE postgres TO vec_identidad_sesiones_v1_registrador,
 vec_identidad_sesiones_v1_revalidador,vec_contexto_actor_v1_runtime,
 vec_autorizacion_fuente,vec_autorizacion_registro,
 vec_autorizacion_motivos_evaluador,vec_dietas_ejecutor;
-- Grupos que crean Dietas 000005 y Personal 000012/000013 (el de D7 sin CONNECT).
CREATE ROLE vec_dietas_registrador_frontera NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_personal_d7_ejecutor NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_personal_registrador_frontera NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
GRANT CONNECT ON DATABASE postgres TO vec_dietas_registrador_frontera, vec_personal_registrador_frontera;
SQL

# Firmas y ACL requeridas: funciones SQL sintéticas solo en esta base, como F4.
python3 - "$base_dir/transaccion.sql" > "$test_dir/stubs.sql" <<'PY'
import re, sys
sql = open(sys.argv[1], encoding="utf-8").read()
filas = re.findall(r"\('([a-z_0-9]+)','(esquema|relacion|funcion)','([^']+)','([A-Z]+)'\)", sql)
assert len(filas) == 39, len(filas)
for grupo, clase, objeto, _ in filas:
    if clase == "esquema":
        print(f"CREATE SCHEMA IF NOT EXISTS {objeto};")
for grupo, clase, objeto, priv in filas:
    if clase == "esquema":
        print(f"GRANT USAGE ON SCHEMA {objeto} TO {grupo};")
    elif clase == "relacion":
        print(f"CREATE TABLE IF NOT EXISTS {objeto} (id integer); GRANT SELECT ON {objeto} TO {grupo};")
    else:
        print(f"DO $s$ BEGIN IF to_regprocedure('{objeto}') IS NULL THEN "
              f"EXECUTE $c$CREATE FUNCTION {objeto} RETURNS jsonb LANGUAGE sql AS $$SELECT '{{}}'::jsonb$$$c$; END IF; END $s$;")
        print(f"REVOKE ALL ON FUNCTION {objeto} FROM PUBLIC; GRANT EXECUTE ON FUNCTION {objeto} TO {grupo};")
PY
psql_admin -f "$test_dir/stubs.sql" >/dev/null

# Estado privado: se crea una vez y no se sobrescribe.
run_f4b --preparar-estado > "$test_dir/estado.out"
[[ "$(stat -c '%a' "$estado")" == 600 ]] || exit 1
if run_f4b --preparar-estado > "$test_dir/estado2.out" 2> "$test_dir/estado2.err"; then
  echo 'F4b sobrescribió el estado privado' >&2; exit 1
fi
if rg -q -f <(python3 -c 'import json,sys; [print(v) for v in json.load(open(sys.argv[1]))["contrasenas"].values()]' "$estado") "$test_dir"/*.out 2>/dev/null; then
  echo 'una contraseña apareció en la salida' >&2; exit 1
fi
chmod 644 "$estado"
debe_fallar estado-permisos 'estado F4b ausente, enlace, ajeno o sin modo 0600' --rollback
chmod 600 "$estado"
sin_efecto estado-permisos

# Negativos: nada se escribe y ninguna cuenta obtiene LOGIN.
psql_admin -c 'CREATE ROLE f4b_login_ajeno LOGIN; GRANT vec_personal_d7_ejecutor TO f4b_login_ajeno' >/dev/null
debe_fallar login-ajeno 'LOGIN ajeno conserva ruta a grupo Dietas o Personal D7' --rollback
psql_admin -c 'DROP ROLE f4b_login_ajeno' >/dev/null
sin_efecto login-ajeno

psql_admin -c 'CREATE ROLE vec_personal_d7_asignacion NOLOGIN; GRANT vec_personal_d7_ejecutor TO vec_personal_d7_asignacion WITH ADMIN FALSE, INHERIT TRUE, SET TRUE' >/dev/null
debe_fallar set-true 'cuentas o grupos F4b con atributos' --commit
psql_admin -c 'DROP ROLE vec_personal_d7_asignacion' >/dev/null
sin_efecto set-true

psql_admin -c 'CREATE ROLE vec_dietas_r1d_auditoria_frontera_desarrollo NOLOGIN' >/dev/null
debe_fallar noveno-r1d 'exactamente las ocho cuentas R1D' --rollback
psql_admin -c 'DROP ROLE vec_dietas_r1d_auditoria_frontera_desarrollo' >/dev/null
sin_efecto noveno-r1d

psql_admin -c 'GRANT USAGE ON SCHEMA vec_ct_sentinel TO vec_personal_registrador_frontera; GRANT SELECT ON vec_ct_sentinel.control TO vec_personal_registrador_frontera' >/dev/null
debe_fallar grupo-ct 'ACL de grupo fuera de los esquemas permitidos F4b' --commit
psql_admin -c 'REVOKE SELECT ON vec_ct_sentinel.control FROM vec_personal_registrador_frontera; REVOKE USAGE ON SCHEMA vec_ct_sentinel FROM vec_personal_registrador_frontera' >/dev/null
sin_efecto grupo-ct

psql_admin -c 'GRANT INSERT ON vec_dietas.version_tarifa_provisional TO vec_dietas_ejecutor' >/dev/null
debe_fallar escritura-directa 'ACL de grupo fuera de los esquemas permitidos F4b' --rollback
psql_admin -c 'REVOKE INSERT ON vec_dietas.version_tarifa_provisional FROM vec_dietas_ejecutor' >/dev/null
sin_efecto escritura-directa

psql_admin -c 'GRANT pg_read_all_data TO vec_dietas_registrador_frontera' >/dev/null
debe_fallar ascenso 'cuentas o grupos F4b con atributos' --rollback
psql_admin -c 'REVOKE pg_read_all_data FROM vec_dietas_registrador_frontera' >/dev/null
sin_efecto ascenso

psql_admin -c 'REVOKE EXECUTE ON FUNCTION vec_personal.corregir_grupo_dieta_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM vec_personal_d7_ejecutor' >/dev/null
debe_fallar acl-requerida 'objeto o ACL requerida para Dietas F4b ausente' --commit
psql_admin -c 'GRANT EXECUTE ON FUNCTION vec_personal.corregir_grupo_dieta_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_personal_d7_ejecutor' >/dev/null
sin_efecto acl-requerida

psql_admin -c 'GRANT EXECUTE ON FUNCTION vec_personal.consultar_asignacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO PUBLIC' >/dev/null
debe_fallar public 'ACL PUBLIC fuera de preimagen F4b permitida' --rollback
psql_admin -c 'REVOKE EXECUTE ON FUNCTION vec_personal.consultar_asignacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC' >/dev/null
sin_efecto public

psql_admin -c "ALTER ROLE vec_dietas_r1d_motivos_desarrollo PASSWORD NULL" >/dev/null
debe_fallar sin-contrasena 'cuentas o grupos F4b con atributos' --rollback
psql_admin -c "ALTER ROLE vec_dietas_r1d_motivos_desarrollo PASSWORD '$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["vec_dietas_r1d_motivos_desarrollo"])' "$test_dir/r1d-contrasenas.json")'" >/dev/null
sin_efecto sin-contrasena

# Desde aquí el servidor registra todas las sentencias, fallidas y por
# duración: F4b debe silenciarlas en su transacción (verificadores SCRAM).
inicio_log="$(docker logs "$container" 2>&1 | wc -l)"
psql_admin <<'SQL' >/dev/null
ALTER SYSTEM SET log_statement = 'all';
ALTER SYSTEM SET log_min_error_statement = 'error';
ALTER SYSTEM SET log_min_duration_statement = 0;
SELECT pg_reload_conf();
SQL
sleep 1

# Ensayo ROLLBACK: nada cambia, ni siquiera el CONNECT del grupo D7.
run_f4b --rollback > "$test_dir/rollback.out"
sin_efecto rollback
[[ "$(psql_admin -c "SELECT has_database_privilege('vec_personal_d7_ejecutor','postgres','CONNECT')")" == f ]] || exit 1

# D7 pudo preparar ya una cuenta NOLOGIN correcta: F4b la adopta.
psql_admin -c 'CREATE ROLE vec_personal_d7_auditoria_frontera NOLOGIN; GRANT vec_personal_registrador_frontera TO vec_personal_d7_auditoria_frontera WITH ADMIN FALSE, INHERIT TRUE, SET FALSE' >/dev/null

run_f4b --commit > "$test_dir/commit.out"
debe_fallar reentrada 'F4b requiere exactamente|preimagen F4b unica ausente|cuentas o grupos F4b' --commit
resultado="$(psql_admin -c "SELECT (SELECT count(*) FROM pg_authid a WHERE a.rolcanlogin AND a.rolpassword LIKE 'SCRAM-SHA-256\$%' AND a.rolname ~ '^vec_(dietas_r1d|dietas_f4b|personal_d7)_' AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=a.oid AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)=1 AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=a.oid)=1)||'|'||(SELECT a.version||':'||(a.documento->>'estado') FROM vec_autorizacion.asignacion_perfil_actual p JOIN vec_autorizacion.asignacion_perfil a USING(asignacion_ref))||'|'||(SELECT n FROM vec_ct_sentinel.control)||'|'||(SELECT n FROM vec_bolsa_sentinel.control)")"
[[ "$resultado" == '11|3:activa|7|11' ]] || { echo "postcondicion F4b PG18 fallida: $resultado" >&2; exit 1; }

# Sonda TLS verify-full con las once cuentas; sin TLS y CA ajena, rechazo.
python3 - "$estado" "$test_dir/r1d-contrasenas.json" <<'PY'
import json, sys
e = json.load(open(sys.argv[1])); e["contrasenas"].update(json.load(open(sys.argv[2])))
json.dump(e, open(sys.argv[1], "w"))
PY
run_f4b --sonda-tls > "$test_dir/sonda.out"
[[ "$(rg -c 'verify-full OK y sin TLS rechazado' "$test_dir/sonda.out")" == 11 ]] || { echo 'sonda TLS incompleta' >&2; exit 1; }
# require_auth: una línea hostssl trust previa para una cuenta hace fallar la
# sonda aunque haya TLS verify-full.
docker exec -u root "$container" sh -c "cp '$hba' /tmp/hba.scram && { printf 'hostssl all vec_personal_d7_asignacion all trust\n'; cat /tmp/hba.scram; } > '$hba'"
psql_admin -c 'SELECT pg_reload_conf()' >/dev/null
sleep 1
if run_f4b --sonda-tls > "$test_dir/sonda-trust.out" 2> "$test_dir/sonda-trust.err"; then
  echo 'sonda TLS aceptó autenticación trust' >&2; exit 1
fi
rg -q 'vec_personal_d7_asignacion no conecta con verify-full' "$test_dir/sonda-trust.err" || { echo 'sonda trust: cuenta no señalada' >&2; exit 1; }
docker exec -u root "$container" sh -c "cat /tmp/hba.scram > '$hba'"
psql_admin -c 'SELECT pg_reload_conf()' >/dev/null
sleep 1
if F4B_CA="$test_dir/ajena.crt" run_f4b --sonda-tls > "$test_dir/sonda-ajena.out" 2> "$test_dir/sonda-ajena.err"; then
  echo 'sonda TLS aceptó una CA ajena' >&2; exit 1
fi

# Retirada: v4 revocada y once NOLOGIN, con testigos CT/Bolsa intactos.
run_f4b --retirar-rollback > "$test_dir/retirada-rollback.out"
[[ "$(psql_admin -c "SELECT a.version||'|'||(a.documento->>'estado') FROM vec_autorizacion.asignacion_perfil_actual p JOIN vec_autorizacion.asignacion_perfil a USING(asignacion_ref)")" == '3|activa' ]] || exit 1
run_f4b --retirar-commit > "$test_dir/retirada-commit.out"
if run_f4b --retirar-commit > "$test_dir/retirada-reentrada.out" 2> "$test_dir/retirada-reentrada.err"; then
  echo 'reentrada retirada F4b aceptada' >&2; exit 1
fi
if run_f4b --sonda-tls > "$test_dir/sonda-retirada.out" 2> "$test_dir/sonda-retirada.err"; then
  echo 'retirada F4b permitió LOGIN' >&2; exit 1
fi
resultado="$(psql_admin -c "SELECT (SELECT count(*) FROM pg_roles WHERE rolcanlogin AND rolname ~ '^vec_(dietas_r1d|dietas_f4b|personal_d7)_')||'|'||(SELECT a.version||':'||(a.documento->>'estado') FROM vec_autorizacion.asignacion_perfil_actual p JOIN vec_autorizacion.asignacion_perfil a USING(asignacion_ref))||'|'||(SELECT n FROM vec_ct_sentinel.control)||'|'||(SELECT n FROM vec_bolsa_sentinel.control)")"
[[ "$resultado" == '0|4:revocada|7|11' ]] || { echo "postcondicion retirada F4b PG18 fallida: $resultado" >&2; exit 1; }
# Registro del servidor: el registro de sentencias estuvo activo (aparece el
# inventario) y ni verificadores SCRAM ni su JSON en base64 llegaron a él.
docker logs "$container" 2>&1 | tail -n "+$((inicio_log + 1))" > "$test_dir/servidor.log"
rg -q 'statement: SELECT pg_reload_conf' "$test_dir/servidor.log" || { echo 'registro de sentencias no activo en el ensayo' >&2; exit 1; }
if rg -q -e 'SCRAM-SHA-256\$4096:' -e 'eyJ2ZWNfZGlldGFzX2Y0' -e 'f4b_verificador' -e 'PASSWORD' "$test_dir/servidor.log"; then
  echo 'verificador SCRAM o su transporte en el registro del servidor' >&2; exit 1
fi
if rg -q -e 'SCRAM-SHA-256\$4096:' -e 'eyJ2ZWNfZGlldGFzX2Y0' "$test_dir"/*.err "$test_dir"/*.out; then
  echo 'verificador SCRAM en la salida del ejecutor' >&2; exit 1
fi
echo "PG18 F4b: estado 0600 sin sobrescritura, nueve negativos sin efecto, ROLLBACK limpio, COMMIT con once LOGIN (una membresía INHERIT/SET FALSE/ADMIN FALSE), sonda TLS verify-full 11/11 y sin TLS/CA ajena rechazados, reentrada denegada, retirada v4, testigos CT/Bolsa, require_auth frente a trust y registro del servidor sin verificadores OK"
