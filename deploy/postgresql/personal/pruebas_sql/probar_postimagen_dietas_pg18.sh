#!/usr/bin/env bash
# Ensayo PostgreSQL 18.4 desechable, sin red, del preflight de postimagen
# Personal que exige la composición de Dietas (bootstrap). Instala Personal
# 000007–000013 con fachadas AD3 TEST-ONLY, crea un LOGIN nominal de
# vec_dietas_ejecutor y ejecuta dentro del contenedor la prueba Go
# TestAcreditarPostimagenPersonalDietasPGReal (positivo y nueve mutaciones)
# antes y después de instalar las ampliaciones opcionales 000014/000015.
set -euo pipefail
base_dir=$(cd -- "$(dirname -- "$0")" && pwd)
repo_dir=$(cd -- "$base_dir/../../../.." && pwd)
migraciones="$repo_dir/deploy/postgresql/personal/migraciones"
trabajo=$(mktemp -d "${TMPDIR:-/tmp}/vec-postimagen-dietas.XXXXXXXX")
container="vec-postimagen-dietas-$$-$RANDOM"
cleanup() { docker rm -f "$container" >/dev/null 2>&1 || true; rm -rf -- "$trabajo"; }
trap cleanup EXIT

(cd "$repo_dir" && CGO_ENABLED=0 go test -c -o "$trabajo/bootstrap.test" ./internal/app/bootstrap)

docker run -d --rm --network none --name "$container" \
  -e POSTGRES_HOST_AUTH_METHOD=trust postgres:18.4 >/dev/null
for _ in $(seq 1 60); do
  if docker exec "$container" pg_isready -q -h /var/run/postgresql -U postgres; then break; fi
  sleep 0.5
done
sleep 1
psql_dba() { docker exec -i "$container" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d postgres; }

psql_dba < "$repo_dir/deploy/postgresql/personal/roles_up.sql"
psql_dba < "$repo_dir/deploy/postgresql/dietas_borradores/roles_up.sql"
psql_dba < "$migraciones/000007_relacion_empleado_dietas.up.sql"
{
  echo 'CREATE SCHEMA vec_autorizacion_atestada_v3;'
  echo 'GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_personal_propietario;'
  cat "$base_dir/postimagen_dietas_stub_000008.sql"
  # Fachadas AD3 de organización (sin su tabla ficticia ni su LOGIN de prueba).
  sed -n '3,12p;18,$p' "$base_dir/organizacion_historica_000010_stub.sql"
  # Fachadas AD3-59 TEST-ONLY de D7 (sin los datos del ensayo focal).
  sed -n '1,46p' "$base_dir/preparar_asignacion_dietas_000012.sql" \
    | grep -v '^CREATE SCHEMA vec_autorizacion_atestada_v3;' \
    | grep -v '^GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3'
} | psql_dba
for m in 000008_consulta_relaciones_propias_dietas 000009_asignacion_dietas \
         000010_organizacion_historica 000011_importacion_organizacion_historica \
         000012_asignacion_dietas 000013_auditoria_frontera_asignacion_dietas; do
  psql_dba < "$migraciones/$m.up.sql" >/dev/null
done
psql_dba <<'SQL'
CREATE ROLE vec_postimagen_dietas_prueba LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_dietas_ejecutor TO vec_postimagen_dietas_prueba WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
SQL

docker cp "$trabajo/bootstrap.test" "$container:/tmp/bootstrap.test"
ejecutar_prueba() {
  docker exec -u postgres \
    -e VEC_DIETAS_POSTIMAGEN_PG_URL='host=/var/run/postgresql user=vec_postimagen_dietas_prueba dbname=postgres' \
    -e VEC_DIETAS_POSTIMAGEN_ADMIN_URL='host=/var/run/postgresql user=postgres dbname=postgres' \
    "$container" /tmp/bootstrap.test -test.run '^TestAcreditarPostimagenPersonalDietasPGReal$' -test.count=1 -test.v \
    | tee "$trabajo/salida-$1.txt"
  grep -q '^--- PASS: TestAcreditarPostimagenPersonalDietasPGReal' "$trabajo/salida-$1.txt"
}
# Etapa 1: 000007–000013, sin las ampliaciones opcionales 000014/000015.
ejecutar_prueba base
# Etapa 2: con 000014/000015 instaladas, sus concesiones a D7 y auditoría
# forman parte de la lista positiva exacta y no deben romper el arranque.
{
  cat "$base_dir/preparar_competencias_asignacion_dietas_000014.sql"
  cat "$migraciones/000014_competencias_asignacion_dietas.up.sql"
  sed -n '1,20p' "$base_dir/rectificacion_dietas_000015_preparar.sql"
  cat "$migraciones/000015_solicitud_rectificacion_dietas.up.sql"
} | psql_dba >/dev/null
ejecutar_prueba ampliada
echo 'OK: PG18.4 postimagen Personal 000007–000013 (y con 000014/000015) acreditada por el preflight de Dietas; nueve mutaciones rechazadas y restauradas en cada etapa (fachadas AD3 TEST-ONLY).'
