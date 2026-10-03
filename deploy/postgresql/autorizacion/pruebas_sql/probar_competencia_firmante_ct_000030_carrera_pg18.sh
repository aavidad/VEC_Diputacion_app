#!/usr/bin/env bash
# AUT30: carrera real contra ambos punteros, solo en PostgreSQL 18 desechable.
set -Eeuo pipefail

raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}
[[ $imagen =~ ^[^[:space:]@]+@sha256:[0-9a-f]{64}$ ]] || {
  echo 'AUT30: la imagen PG18 debe estar fijada por digest' >&2
  exit 2
}
contenedor="vec-aut30-carrera-${USER:-usuario}-$$"
base=vec_aut30_carrera
temporal=$(mktemp -d /dev/shm/vec-aut30-carrera.XXXXXX)
umask 077

limpiar() {
  local estado=$?
  if [[ $estado -ne 0 ]]; then
    docker logs --tail 80 "$contenedor" >&2 || true
  fi
  docker rm -f "$contenedor" >/dev/null 2>&1 || true
  rm -rf -- "$temporal"
}
trap limpiar EXIT INT TERM

psql_admin() {
  docker exec -i "$contenedor" psql -Xq -v ON_ERROR_STOP=1 -U postgres -d "$base" "$@"
}
psql_login() {
  docker exec -i "$contenedor" psql -XAtq -v ON_ERROR_STOP=1 -U aut30_carrera_login -d "$base" "$@"
}
valor() {
  docker exec "$contenedor" psql -XAtq -v ON_ERROR_STOP=1 -U postgres -d "$base" -c "$1"
}
esperar_texto() {
  local archivo=$1 texto=$2
  for _ in $(seq 1 300); do
    grep -Fqx "$texto" "$archivo" && return 0
    sleep 0.01
  done
  echo "AUT30: no se alcanzó la barrera $texto" >&2
  return 1
}

docker run -d --rm --pull never --network none --name "$contenedor" \
  -e POSTGRES_DB="$base" -e POSTGRES_HOST_AUTH_METHOD=trust "$imagen" >/dev/null
for _ in $(seq 1 120); do
  docker exec "$contenedor" psql -XAtq -U postgres -d "$base" -c 'SELECT 1' \
    >/dev/null 2>&1 && break
  sleep 0.25
done
docker exec "$contenedor" psql -XAtq -U postgres -d "$base" -c 'SELECT 1' >/dev/null
[[ $(valor 'SHOW server_version_num') == 180004 ]] || {
  echo 'AUT30: se requiere PostgreSQL 18.4' >&2
  exit 2
}

psql_admin < "$raiz/deploy/postgresql/autorizacion/roles_up.sql" >/dev/null
psql_admin <<'SQL'
CREATE ROLE vec_contratacion_temporal_propietario NOLOGIN NOSUPERUSER NOCREATEDB
  NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_contratacion_temporal_ejecutor NOLOGIN NOSUPERUSER NOCREATEDB
  NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
SQL
psql_admin < "$raiz/deploy/postgresql/autorizacion/migraciones/000001_autorizacion.up.sql" >/dev/null
psql_admin < "$raiz/deploy/postgresql/autorizacion/migraciones/000030_competencia_firmante_ct.up.sql" >/dev/null

# Datos estrictamente sintéticos: hacen que la fachada tome los dos FOR UPDATE
# antes del cierre provisional por ausencia de fuente nominal.
psql_admin <<'SQL'
SET ROLE vec_autorizacion_propietario;
INSERT INTO vec_autorizacion.version_rol
  (version_rol_ref,rol_id,version,huella_sha256,publicada_en,documento)
VALUES ('rol:aut30_firmante:v1','aut30_firmante',1,repeat('a',64),
  '2026-10-02T00:00:00Z',
  '{"rol_id":"aut30_firmante","version":1,"publicada_en":"2026-10-02T00:00:00Z","estado":"publicada","concesiones":[{"accion":"firmar","modulo_id":"contratacion_temporal","tipo_recurso":"documento","finalidades":["firma"],"campos_permitidos":[],"obligaciones":[],"garantia_minima":"alto"}]}'::jsonb);
INSERT INTO vec_autorizacion.control_vigencia_version_rol
  (version_rol_ref,revision,estado,huella_sha256,actualizado_en,documento)
VALUES
 ('rol:aut30_firmante:v1',1,'habilitada',repeat('b',64),'2026-10-02T00:00:00Z',
  '{"version_rol_ref":"rol:aut30_firmante:v1","revision":1,"estado":"habilitada","actualizado_en":"2026-10-02T00:00:00Z"}'::jsonb),
 ('rol:aut30_firmante:v1',2,'retirada',repeat('c',64),'2026-10-02T00:00:01Z',
  '{"version_rol_ref":"rol:aut30_firmante:v1","revision":2,"estado":"retirada","actualizado_en":"2026-10-02T00:00:01Z"}'::jsonb);
INSERT INTO vec_autorizacion.control_vigencia_version_rol_actual
  (version_rol_ref,revision,actualizada_en,actualizada_por,acto_ref)
VALUES ('rol:aut30_firmante:v1',1,'2026-10-02T00:00:00Z','aut30_prueba','acto:aut30:1');
INSERT INTO vec_autorizacion.asignacion_perfil
  (asignacion_ref,asignacion_id,version,perfil_activo_ref,principal_id,version_rol_ref,huella_sha256,emitida_en,documento)
VALUES
 ('asignacion:aut30_firmante:v1','aut30_firmante',1,'perfil:aut30_firmante','principal:aut30_firmante','rol:aut30_firmante:v1',repeat('d',64),'2026-10-02T00:00:00Z',
  '{"asignacion_id":"aut30_firmante","version":1,"perfil_activo_ref":"perfil:aut30_firmante","principal_id":"principal:aut30_firmante","version_rol_ref":"rol:aut30_firmante:v1","estado":"activa","emitida_en":"2026-10-02T00:00:00Z","vigente_desde":"2026-10-02T00:00:00Z","vigente_hasta":"2030-01-01T00:00:00Z","ambitos":[{"clave":"organizacion_ref","valores":["organizacion:aut30"]},{"clave":"unidad_ref","valores":["unidad:aut30"]}]}'::jsonb),
 ('asignacion:aut30_firmante:v2','aut30_firmante',2,'perfil:aut30_firmante','principal:aut30_firmante','rol:aut30_firmante:v1',repeat('e',64),'2026-10-02T00:00:01Z',
  '{"asignacion_id":"aut30_firmante","version":2,"perfil_activo_ref":"perfil:aut30_firmante","principal_id":"principal:aut30_firmante","version_rol_ref":"rol:aut30_firmante:v1","estado":"revocada","emitida_en":"2026-10-02T00:00:01Z","vigente_desde":"2026-10-02T00:00:01Z","vigente_hasta":"2030-01-01T00:00:00Z","ambitos":[{"clave":"organizacion_ref","valores":["organizacion:aut30"]},{"clave":"unidad_ref","valores":["unidad:aut30"]}]}'::jsonb);
INSERT INTO vec_autorizacion.asignacion_perfil_actual
  (perfil_activo_ref,asignacion_ref,actualizada_en,actualizada_por,acto_ref)
VALUES ('perfil:aut30_firmante','asignacion:aut30_firmante:v1','2026-10-02T00:00:00Z','aut30_prueba','acto:aut30:1');
RESET ROLE;

CREATE ROLE aut30_carrera_login LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
  INHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_contratacion_temporal_ejecutor TO aut30_carrera_login;
GRANT CONNECT ON DATABASE vec_aut30_carrera TO aut30_carrera_login;
GRANT USAGE ON SCHEMA public TO aut30_carrera_login;
CREATE FUNCTION public.aut30_carrera_ct(p_material text) RETURNS boolean
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp
AS $$ SELECT vec_autorizacion.revalidar_competencia_firmante_ct_v1(p_material) $$;
ALTER FUNCTION public.aut30_carrera_ct(text) OWNER TO vec_contratacion_temporal_propietario;
REVOKE ALL ON FUNCTION public.aut30_carrera_ct(text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION public.aut30_carrera_ct(text) TO aut30_carrera_login;
SQL

material=$(cat <<'JSON'
{"FirmantePrincipalRef":"principal:aut30_firmante","PerfilFirmanteRef":"perfil-firmante-aut30","CargoFirmante":"cargo-aut30","OrganizacionRef":"organizacion:aut30","UnidadFirmanteRef":"unidad:aut30","PerfilActivoFirmanteRef":"perfil:aut30_firmante","AsignacionFirmanteRef":"asignacion:aut30_firmante:v1","AsignacionFirmanteVersion":1,"AsignacionFirmanteHuella":"dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd","VersionRolFirmanteRef":"rol:aut30_firmante:v1","VersionRolFirmanteHuella":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","ControlVigenciaFirmanteRef":"rol:aut30_firmante:v1","ControlVigenciaFirmanteRevision":1,"ControlVigenciaFirmanteHuella":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","AsignacionVigenteDesde":"2026-10-02T00:00:00Z","AsignacionVigenteHasta":"2030-01-01T00:00:00Z","CatalogoRef":"catalogo:aut30","CatalogoHuella":"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff","PasoRef":"paso:aut30","PasoOrden":1}
JSON
)

cat > "$temporal/sesion_a.sql" <<SQL
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SELECT public.aut30_carrera_ct('$material')::text;
SELECT 'AUT30-A-DOS-PUNTEROS-BLOQUEADOS';
SELECT pg_sleep(2);
COMMIT;
SQL
psql_login < "$temporal/sesion_a.sql" > "$temporal/sesion_a.log" 2>&1 &
pid_a=$!
esperar_texto "$temporal/sesion_a.log" 'AUT30-A-DOS-PUNTEROS-BLOQUEADOS'
grep -Fqx 'false' "$temporal/sesion_a.log" || {
  echo 'AUT30: la llamada inicial no cerró con false' >&2
  cat "$temporal/sesion_a.log" >&2
  exit 1
}

# Cada actualización toca un puntero distinto. Ambas deben esperar a la misma
# transacción que ya revalidó y retiene los bloqueos hasta su cierre.
for objetivo in asignacion control; do
  if [[ $objetivo == asignacion ]]; then
    sql="BEGIN ISOLATION LEVEL SERIALIZABLE; SET LOCAL lock_timeout='250ms'; UPDATE vec_autorizacion.asignacion_perfil_actual SET asignacion_ref='asignacion:aut30_firmante:v2',actualizada_en='2026-10-02T00:00:01Z',actualizada_por='aut30_revocacion',acto_ref='acto:aut30:2' WHERE perfil_activo_ref='perfil:aut30_firmante'; COMMIT;"
  else
    sql="BEGIN ISOLATION LEVEL SERIALIZABLE; SET LOCAL lock_timeout='250ms'; UPDATE vec_autorizacion.control_vigencia_version_rol_actual SET revision=2,actualizada_en='2026-10-02T00:00:01Z',actualizada_por='aut30_revocacion',acto_ref='acto:aut30:2' WHERE version_rol_ref='rol:aut30_firmante:v1'; COMMIT;"
  fi
  if psql_admin -c "$sql" > "$temporal/revocacion_$objetivo.log" 2>&1; then
    echo "AUT30: la revocación de $objetivo no esperó el bloqueo" >&2
    exit 1
  fi
  grep -Eq '55P03|lock timeout' "$temporal/revocacion_$objetivo.log" || {
    cat "$temporal/revocacion_$objetivo.log" >&2
    exit 1
  }
done
wait "$pid_a"

# La revocación gana después del cierre de la revalidación: ambos punteros
# avanzan en una única transacción. La referencia anterior nunca escribe efecto.
psql_admin <<'SQL'
BEGIN ISOLATION LEVEL SERIALIZABLE;
UPDATE vec_autorizacion.asignacion_perfil_actual
   SET asignacion_ref='asignacion:aut30_firmante:v2',actualizada_en='2026-10-02T00:00:01Z',
       actualizada_por='aut30_revocacion',acto_ref='acto:aut30:2'
 WHERE perfil_activo_ref='perfil:aut30_firmante';
UPDATE vec_autorizacion.control_vigencia_version_rol_actual
   SET revision=2,actualizada_en='2026-10-02T00:00:01Z',actualizada_por='aut30_revocacion',acto_ref='acto:aut30:2'
 WHERE version_rol_ref='rol:aut30_firmante:v1';
COMMIT;
SQL
antes=$(valor 'SELECT count(*) FROM vec_autorizacion.decision_autorizacion')
resultado=$(psql_login -At -c "BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE; SELECT public.aut30_carrera_ct('$material')::text; COMMIT" | grep -E '^(true|false)$' | tail -1)
despues=$(valor 'SELECT count(*) FROM vec_autorizacion.decision_autorizacion')
[[ $resultado == false && $antes == "$despues" ]] || {
  echo "AUT30: la referencia revocada produjo resultado=$resultado, efectos $antes->$despues" >&2
  exit 1
}
[[ $(valor "SELECT asignacion_ref='asignacion:aut30_firmante:v2' FROM vec_autorizacion.asignacion_perfil_actual WHERE perfil_activo_ref='perfil:aut30_firmante'") == t ]]
[[ $(valor "SELECT revision=2 FROM vec_autorizacion.control_vigencia_version_rol_actual WHERE version_rol_ref='rol:aut30_firmante:v1'") == t ]]
echo 'AUT30-CARRERA-PG18-OK: wrapper CT SERIALIZABLE, dos punteros bloqueados, revocación vence y referencia antigua=false sin efecto'
