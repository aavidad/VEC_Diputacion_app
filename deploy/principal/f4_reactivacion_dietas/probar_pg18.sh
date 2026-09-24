#!/usr/bin/env bash
set -euo pipefail
umask 077
: "${VEC_F4_TEST_BASE:?directorio privado 0700 fuera de Git requerido}"
base_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
repo_dir="$(cd "$base_dir/../../.." && pwd -P)"
test_base="$(realpath -e "$VEC_F4_TEST_BASE")"
case "$test_base/" in "$repo_dir/"*) echo 'base de prueba dentro de Git' >&2; exit 2;; esac
[[ "$(stat -c '%a' "$test_base")" == 700 && "$(stat -c '%u' "$test_base")" == "$(id -u)" ]] || exit 2
test_dir="$(mktemp -d "$test_base/f4-pg18.XXXXXXXX")"
container="f4-dietas-pg18-$$"
trap 'docker rm -f "$container" >/dev/null 2>&1 || true' EXIT
docker run --rm --network none -d --name "$container" \
  -e POSTGRES_HOST_AUTH_METHOD=trust \
  -v "$repo_dir:$repo_dir:ro" -v "$test_dir:$test_dir:rw" \
  postgres:18.4 >/dev/null
for _ in $(seq 1 60); do
  if docker exec "$container" pg_isready -h 127.0.0.1 -U postgres >/dev/null 2>&1; then break; fi
  sleep 1
done
docker exec "$container" pg_isready -h 127.0.0.1 -U postgres >/dev/null

cat > "$test_dir/podman" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
[[ "${1:-}" == exec && "${2:-}" == -i && "${3:-}" == "$VEC_F4_TEST_CONTAINER" ]] || exit 2
exec docker exec -i -u root "$VEC_F4_TEST_CONTAINER" "${@:4}"
SH
chmod 700 "$test_dir/podman"
export VEC_F4_TEST_CONTAINER="$container" PATH="$test_dir:$PATH"
psql_admin() { docker exec -i -u root "$container" psql -XAtq -v ON_ERROR_STOP=1 -U postgres -d postgres "$@"; }
run_p6() {
  env -u PGSERVICE -u PGSERVICEFILE -u PGPASSFILE -u PGHOST -u PGHOSTADDR \
    VEC_P6_POSTGRES_CONTAINER="$container" VEC_P6_PG_SOCKET_DIR=/var/run/postgresql \
    VEC_P6_PG_PORT=5432 VEC_P6_EVIDENCIA_DIR="$test_dir" VEC_P6_APLICAR=SI-P6-REVISADO \
    bash "$repo_dir/deploy/principal/p6_retirada_dietas/ejecutar.sh" "$@"
}
run_f4() {
  env -u PGSERVICE -u PGSERVICEFILE -u PGPASSFILE -u PGHOST -u PGHOSTADDR \
    VEC_F4_POSTGRES_CONTAINER="$container" VEC_F4_PG_SOCKET_DIR=/var/run/postgresql \
    VEC_F4_PG_PORT=5432 VEC_F4_EVIDENCIA_DIR="$test_dir" VEC_F4_APLICAR=SI-F4-REVISADO \
    bash "$base_dir/ejecutar.sh" "$@"
}

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

# El fixture P6 concede CONNECT directamente para su propia prueba. La
# preparación R1D real lo concede por grupos; restituimos esa preimagen.
psql_admin <<'SQL' >/dev/null
DO $x$ DECLARE r record; BEGIN
 FOR r IN SELECT rolname FROM pg_roles WHERE rolname ~ '^vec_dietas_r1d_.*_desarrollo$' LOOP
  EXECUTE format('REVOKE CONNECT ON DATABASE postgres FROM %I',r.rolname);
 END LOOP;
END $x$;
GRANT CONNECT ON DATABASE postgres TO vec_identidad_sesiones_v1_registrador,
 vec_identidad_sesiones_v1_revalidador,vec_contexto_actor_v1_runtime,
 vec_autorizacion_fuente,vec_autorizacion_registro,
 vec_autorizacion_motivos_evaluador,vec_dietas_ejecutor;
SQL
[[ "$(psql_admin -c "SELECT count(*) FROM pg_shdepend d JOIN pg_roles r ON r.oid=d.refobjid WHERE d.refclassid='pg_authid'::regclass AND d.deptype='a' AND r.rolname ~ '^vec_dietas_r1d_.*_desarrollo$'")" == 0 ]] || exit 1
for nombre in registro_identidad revalidacion_identidad contexto fuente_autorizacion registro_autorizacion motivos dietas personal; do
  if docker exec -u root "$container" psql -Xq -h 127.0.0.1 -U "vec_dietas_r1d_${nombre}_desarrollo" -d postgres -c 'SELECT 1' \
    > "$test_dir/login-previo-${nombre}.out" 2> "$test_dir/login-previo-${nombre}.err"; then
    echo 'P6 permitio un LOGIN anterior' >&2; exit 1
  fi
done
run_f4 --rollback > "$test_dir/rollback.out"
[[ "$(psql_admin -c "SELECT a.version||'|'||(a.documento->>'estado') FROM vec_autorizacion.asignacion_perfil_actual p JOIN vec_autorizacion.asignacion_perfil a USING(asignacion_ref)")" == '2|revocada' ]] || exit 1

# Una ACL directa inesperada debe detener F4 sin escribir v3.
psql_admin -c 'GRANT CONNECT ON DATABASE postgres TO vec_dietas_r1d_dietas_desarrollo' >/dev/null
if run_f4 --rollback > "$test_dir/acl.out" 2> "$test_dir/acl.err"; then
  echo 'ACL directa inesperada aceptada' >&2; exit 1
fi
psql_admin -c 'REVOKE CONNECT ON DATABASE postgres FROM vec_dietas_r1d_dietas_desarrollo' >/dev/null

# El noveno LOGIN que el preparador histórico podía crear no se absorbe.
psql_admin -c 'CREATE ROLE vec_dietas_r1d_auditoria_frontera_desarrollo LOGIN' >/dev/null
if run_f4 --rollback > "$test_dir/noveno.out" 2> "$test_dir/noveno.err"; then
  echo 'noveno LOGIN inesperado aceptado' >&2; exit 1
fi
psql_admin -c 'DROP ROLE vec_dietas_r1d_auditoria_frontera_desarrollo' >/dev/null

# NOLOGIN no mata una sesión ya abierta: F4 debe detectar la sesión residual.
psql_admin -c 'ALTER ROLE vec_dietas_r1d_dietas_desarrollo LOGIN' >/dev/null
docker exec -u root "$container" psql -XAtq -h 127.0.0.1 -U vec_dietas_r1d_dietas_desarrollo -d postgres \
  -c 'SELECT pg_sleep(8)' > "$test_dir/sesion-residual.out" &
sesion_pid=$!
visto=f
for _ in $(seq 1 40); do
  visto="$(psql_admin -c "SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE usename='vec_dietas_r1d_dietas_desarrollo')")"
  [[ "$visto" == t ]] && break
  sleep 0.1
done
[[ "$visto" == t ]] || { echo 'sesion de prueba ausente' >&2; exit 1; }
psql_admin -c 'ALTER ROLE vec_dietas_r1d_dietas_desarrollo NOLOGIN' >/dev/null
if run_f4 --rollback > "$test_dir/sesion-f4.out" 2> "$test_dir/sesion-f4.err"; then
  echo 'sesion residual aceptada' >&2; exit 1
fi
wait "$sesion_pid"

# Una preimagen P6 con huella distinta se rechaza antes de tocar LOGIN.
pre_sha="$(psql_admin -c "SELECT huella_sha256 FROM vec_autorizacion.asignacion_perfil WHERE asignacion_ref='asignacion:dietas_r1d_0123456789abcdef:v2'")"
[[ "$pre_sha" =~ ^[0-9a-f]{64}$ ]] || exit 1
psql_admin <<'SQL' >/dev/null
ALTER TABLE vec_autorizacion.asignacion_perfil DISABLE TRIGGER USER;
UPDATE vec_autorizacion.asignacion_perfil SET huella_sha256=repeat('f',64)
 WHERE asignacion_ref='asignacion:dietas_r1d_0123456789abcdef:v2';
ALTER TABLE vec_autorizacion.asignacion_perfil ENABLE TRIGGER USER;
SQL
if run_f4 --rollback > "$test_dir/huella.out" 2> "$test_dir/huella.err"; then
  echo 'preimagen distinta aceptada' >&2; exit 1
fi
psql_admin -v "pre_sha=$pre_sha" <<'SQL' >/dev/null
ALTER TABLE vec_autorizacion.asignacion_perfil DISABLE TRIGGER USER;
UPDATE vec_autorizacion.asignacion_perfil SET huella_sha256=:'pre_sha'
 WHERE asignacion_ref='asignacion:dietas_r1d_0123456789abcdef:v2';
ALTER TABLE vec_autorizacion.asignacion_perfil ENABLE TRIGGER USER;
SQL

run_f4 --commit > "$test_dir/commit.out"
if run_f4 --commit > "$test_dir/reentrada.out" 2> "$test_dir/reentrada.err"; then
  echo 'reentrada F4 aceptada' >&2; exit 1
fi
for nombre in registro_identidad revalidacion_identidad contexto fuente_autorizacion registro_autorizacion motivos dietas personal; do
  [[ "$(docker exec -u root "$container" psql -XAtq -h 127.0.0.1 -U "vec_dietas_r1d_${nombre}_desarrollo" -d postgres -c 'SELECT 1')" == 1 ]] || exit 1
done
instantanea="$(docker exec -u root "$container" psql -XAtq -h 127.0.0.1 -U vec_dietas_r1d_fuente_autorizacion_desarrollo -d postgres \
  -c "SELECT (documento_asignacion->>'version')||'|'||(documento_asignacion->>'estado') FROM vec_autorizacion.obtener_instantanea('per_0123456789abcdef0123456789abcdef','prf_0123456789abcdef0123456789abcdef')")"
[[ "$instantanea" == '3|activa' ]] || { echo 'fuente V3 no devolvio nueva asignacion' >&2; exit 1; }
resultado="$(psql_admin -c "SELECT count(*)||'|'||(SELECT a.version||':'||(a.documento->>'estado') FROM vec_autorizacion.asignacion_perfil_actual p JOIN vec_autorizacion.asignacion_perfil a USING(asignacion_ref))||'|'||(SELECT count(*) FROM pg_roles WHERE rolname ~ '^vec_dietas_r1d_.*_desarrollo$' AND rolcanlogin)||'|'||(SELECT n FROM vec_ct_sentinel.control)||'|'||(SELECT n FROM vec_bolsa_sentinel.control) FROM vec_autorizacion.asignacion_perfil WHERE asignacion_id='dietas_r1d_0123456789abcdef'")"
[[ "$resultado" == '3|3:activa|8|7|11' ]] || { echo 'postcondicion F4 PG18 fallida' >&2; exit 1; }
run_f4 --retirar-rollback > "$test_dir/retirada-rollback.out"
[[ "$(psql_admin -c "SELECT a.version||'|'||(a.documento->>'estado') FROM vec_autorizacion.asignacion_perfil_actual p JOIN vec_autorizacion.asignacion_perfil a USING(asignacion_ref)")" == '3|activa' ]] || exit 1
run_f4 --retirar-commit > "$test_dir/retirada-commit.out"
if run_f4 --retirar-commit > "$test_dir/retirada-reentrada.out" 2> "$test_dir/retirada-reentrada.err"; then
  echo 'reentrada retirada F4 aceptada' >&2; exit 1
fi
if docker exec -u root "$container" psql -Xq -h 127.0.0.1 -U vec_dietas_r1d_dietas_desarrollo -d postgres -c 'SELECT 1' \
  > "$test_dir/login-retirado.out" 2> "$test_dir/login-retirado.err"; then
  echo 'retirada F4 permitio LOGIN' >&2; exit 1
fi
resultado="$(psql_admin -c "SELECT count(*)||'|'||(SELECT a.version||':'||(a.documento->>'estado') FROM vec_autorizacion.asignacion_perfil_actual p JOIN vec_autorizacion.asignacion_perfil a USING(asignacion_ref))||'|'||(SELECT count(*) FROM pg_roles WHERE rolname ~ '^vec_dietas_r1d_.*_desarrollo$' AND rolcanlogin)||'|'||(SELECT n FROM vec_ct_sentinel.control)||'|'||(SELECT n FROM vec_bolsa_sentinel.control) FROM vec_autorizacion.asignacion_perfil WHERE asignacion_id='dietas_r1d_0123456789abcdef'")"
[[ "$resultado" == '4|4:revocada|0|7|11' ]] || { echo 'postcondicion retirada F4 PG18 fallida' >&2; exit 1; }
echo "PG18 F4 ROLLBACK/COMMIT/reentrada, ACL y preimagen negativa, ocho LOGIN, retirada v4 y testigos CT/Bolsa OK"
