#!/usr/bin/env bash
set -euo pipefail

raiz=$(cd "$(dirname "$0")/../../../.." && pwd)
datos=$(mktemp -d /dev/shm/vec-correos-preflight-XXXXXX)
contenedor="vec-correos-preflight-${RANDOM}${RANDOM}"
limpiar() {
  docker stop "$contenedor" >/dev/null 2>&1 || true
  docker run --rm -v /dev/shm:/s alpine rm -rf -- "/s/$(basename "$datos")" >/dev/null
}
trap limpiar EXIT

docker run --rm -v "$datos:/d" alpine chown -R 999:999 /d >/dev/null
docker run -d --rm --name "$contenedor" --network none \
  -e POSTGRES_HOST_AUTH_METHOD=trust -v "$datos:/var/lib/postgresql" postgres:18.4 >/dev/null
for _ in $(seq 1 60); do
  if docker exec "$contenedor" pg_isready -U postgres >/dev/null 2>&1; then break; fi
  sleep 1
done
docker exec "$contenedor" pg_isready -U postgres >/dev/null

# Preimagen mínima sintética de #145 y #151: propietarios, ocho tablas con
# FORCE RLS y login preflight con una sola membresía. Este guion no sustituye
# el ensayo sobre el clon de la principal con la cadena SQL completa.
docker exec -i "$contenedor" psql -U postgres -v ON_ERROR_STOP=1 >/dev/null <<'SQL'
CREATE ROLE vec_usuarios_correos_externo_propietario NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_autorizacion_atestada_v3_preflight_externo NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_externo_preflight_v3_desarrollo LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
GRANT vec_autorizacion_atestada_v3_preflight_externo TO vec_externo_preflight_v3_desarrollo WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
CREATE SCHEMA vec_usuarios_correos_externo AUTHORIZATION vec_usuarios_correos_externo_propietario;
CREATE SCHEMA vec_usuarios_correos_interno;
DO $f$ DECLARE t text; BEGIN
 FOREACH t IN ARRAY ARRAY['correos_conjunto','correos_direccion','correos_desafio','correos_intento_fallido',
  'correos_historia','correos_recibo','correos_envio','correos_contexto'] LOOP
  EXECUTE format('CREATE TABLE vec_usuarios_correos_externo.%I (id integer PRIMARY KEY, clave_igualdad_ref text, clave_sobre_ref text, clave_ref text, huella_clave_ref text)',t);
  EXECUTE format('ALTER TABLE vec_usuarios_correos_externo.%I OWNER TO vec_usuarios_correos_externo_propietario',t);
  EXECUTE format('ALTER TABLE vec_usuarios_correos_externo.%I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE format('ALTER TABLE vec_usuarios_correos_externo.%I FORCE ROW LEVEL SECURITY',t);
 END LOOP;
END $f$;
INSERT INTO vec_usuarios_correos_externo.correos_direccion(id,clave_sobre_ref,clave_igualdad_ref)
VALUES (1,'clave:kms:desarrollo:usuarios-correos-cifrado:v1','clave:kms:desarrollo:usuarios-correos-igualdad:v1');
SQL

docker exec -i "$contenedor" psql -U postgres -v ON_ERROR_STOP=1 >/dev/null < \
  "$raiz/deploy/postgresql/usuarios_vec/migraciones/000012_preflight_claves_correos_externos.up.sql"

previa=$(docker exec -i "$contenedor" psql -U postgres -At -v ON_ERROR_STOP=1 <<'SQL'
SET SESSION AUTHORIZATION vec_externo_preflight_v3_desarrollo;
SELECT vec_usuarios_correos_externo.poblacion_sin_claves_ajenas_v1();
SELECT has_table_privilege(session_user,'vec_usuarios_correos_externo.correos_direccion','SELECT');
SQL
)
[[ "$previa" == *$'\nf\nf' ]] || { echo 'FALLO: clave previa o ACL aceptada'; exit 1; }

docker exec "$contenedor" psql -U postgres -v ON_ERROR_STOP=1 -c \
  'DELETE FROM vec_usuarios_correos_externo.correos_direccion WHERE id=1' >/dev/null
vacio=$(docker exec -i "$contenedor" psql -U postgres -At -v ON_ERROR_STOP=1 <<'SQL'
SET SESSION AUTHORIZATION vec_externo_preflight_v3_desarrollo;
SELECT vec_usuarios_correos_externo.poblacion_sin_claves_ajenas_v1();
SQL
)
[[ "$vacio" == *$'\nt' ]] || { echo 'FALLO: población vacía no acreditada'; exit 1; }

docker exec "$contenedor" psql -U postgres -v ON_ERROR_STOP=1 -c \
  "INSERT INTO vec_usuarios_correos_externo.correos_direccion(id,clave_sobre_ref,clave_igualdad_ref) VALUES (2,'clave:kms:desarrollo:usuarios-correos-externo-cifrado:v1','clave:kms:desarrollo:usuarios-correos-externo-igualdad:v1')" >/dev/null
propia=$(docker exec -i "$contenedor" psql -U postgres -At -v ON_ERROR_STOP=1 <<'SQL'
SET SESSION AUTHORIZATION vec_externo_preflight_v3_desarrollo;
SELECT vec_usuarios_correos_externo.poblacion_sin_claves_ajenas_v1();
SQL
)
[[ "$propia" == *$'\nt' ]] || { echo 'FALLO: un reinicio con clave propia se denegaría'; exit 1; }

if docker exec "$contenedor" psql -U postgres -v ON_ERROR_STOP=1 -c \
  "INSERT INTO vec_usuarios_correos_externo.correos_direccion(id,clave_sobre_ref,clave_igualdad_ref) VALUES (3,'clave:kms:desarrollo:usuarios-correos-cifrado:v1','clave:kms:desarrollo:usuarios-correos-igualdad:v1')" >/dev/null 2>&1; then
  echo 'FALLO: escritura nueva con clave antigua aceptada'; exit 1
fi
echo 'ENSAYO-OK sintético: clave previa denegada, vacío y clave propia aptos, sin SELECT directo'
