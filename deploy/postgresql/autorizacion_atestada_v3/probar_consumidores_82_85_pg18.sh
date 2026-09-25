#!/usr/bin/env bash
# Ensayo de AD3-82, AD3-83, AD3-84 y AD3-85 en PostgreSQL 18.4 desechable
# sobre la estructura real restaurada (volcado con datos sintéticos).
# Uso: probar_consumidores_82_85_pg18.sh GLOBALES_SQL VOLCADO_PG_DUMP
# Comprueba: ROLLBACK sin rastro; UP y doble UP rechazado de cada una; ida y
# vuelta con huella del núcleo y de las audiencias (todas las DOWN, también
# fuera de orden, devuelven exactamente la preimagen y un UP posterior
# reproduce el mismo núcleo); las condiciones de cada extensión presentes en
# el núcleo instalado según una lista independiente de esta prueba (un UP y
# un DOWN que las debiliten a la vez no pasan); ACL cerrada aunque haya
# privilegios por defecto para otro rol, y las comprobaciones previas de la
# fachada de firma con 42501. El contenedor usa --rm, sin red ni volúmenes
# anónimos; sus datos viven en /dev/shm/vec-pg-ad3-82-85-<pid> y se borran.
set -Eeuo pipefail

repo=$(git -C "$(dirname -- "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)
globales=${1:?falta el volcado de roles (pg_dumpall --globals-only)}
volcado=${2:?falta el volcado de la base (pg_dump -Fc)}
[[ -s $globales && -s $volcado ]] || { echo 'Faltan los volcados' >&2; exit 2; }
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm}
nombre="vec-pg-ad3-82-85-$$"
datos="/dev/shm/$nombre"
limpiar() {
  docker rm -f "$nombre" >/dev/null 2>&1 || true
  if [[ -d $datos ]]; then docker run --rm -v "$datos:/d" --entrypoint /bin/sh "$imagen" -c 'rm -rf /d/* /d/.[!.]*' >/dev/null 2>&1 || true; fi
  rmdir "$datos" 2>/dev/null || true
}
trap limpiar EXIT
mkdir -p "$datos"
docker run -d --rm --network none --name "$nombre" -e POSTGRES_HOST_AUTH_METHOD=trust \
  -v "$datos:/var/lib/postgresql" -v "$repo:/repo:ro" "$imagen" >/dev/null
for _ in $(seq 1 240); do
  if docker logs "$nombre" 2>&1 | grep -q 'PostgreSQL init process complete' && docker exec "$nombre" pg_isready -q -U postgres; then break; fi
  sleep 0.5
done
run() { docker exec -i "$nombre" psql -X -q -o /dev/null -v ON_ERROR_STOP=1 -U postgres -d postgres "$@"; }
escalar() { docker exec "$nombre" psql -X -At -v ON_ERROR_STOP=1 -U postgres -d postgres -c "$1"; }
ok() { printf 'OK %s\n' "$1"; }
igual() { [[ $1 == "$2" ]] || { echo "FALLO $3: obtenido «$1», esperado «$2»" >&2; exit 1; }; ok "$3"; }
falla() { if run -f "/repo/$1" >/dev/null 2>&1; then echo "FALLO: se aceptó $1" >&2; exit 1; fi; ok "rechazo de $(basename "$1")"; }
echo '== Restaurando roles y estructura real'
docker exec -i "$nombre" psql -X -q -U postgres -d postgres <"$globales" >/dev/null 2>&1 || true
docker exec -i "$nombre" pg_restore -U postgres -d postgres <"$volcado" >/dev/null 2>&1 || true
[[ $(escalar "SELECT to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL") == t ]] || { echo 'Restauración sin núcleo AD3' >&2; exit 2; }

m=deploy/postgresql/autorizacion_atestada_v3/migraciones
todas=(000082_consumidor_cese_cierre_contratacion_temporal 000083_consumidor_modificacion_tras_nombramiento_ct
       000084_consumidor_portal_candidato_bolsa 000085_consumidor_firma_documento_ct)
huella="SELECT md5(pg_get_functiondef('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure))
  ||md5(pg_get_constraintdef(c.oid)) FROM pg_constraint c WHERE c.conname='clave_capacidad_version_audiencia_consumo_check'"
# Un rol ajeno con EXECUTE por defecto sobre las funciones nuevas del
# propietario de AD3: las fachadas deben quedar cerradas igualmente.
run <<'SQL'
CREATE ROLE vec_ad3_prueba_ajeno NOLOGIN;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_autorizacion_atestada_v3_propietario IN SCHEMA vec_autorizacion_atestada_v3
  GRANT EXECUTE ON FUNCTIONS TO vec_ad3_prueba_ajeno;
SQL
inicial=$(escalar "$huella")
echo '== ROLLBACK, UP y doble UP'
for x in "${todas[@]}"; do
  sed 's/^COMMIT;$/ROLLBACK;/' "$repo/$m/$x.up.sql" | run
done
igual "$(escalar "$huella")" "$inicial" 'ROLLBACK de las cuatro no deja rastro'
for x in "${todas[@]}"; do run -f "/repo/$m/$x.up.sql"; ok "UP $x"; falla "$m/$x.up.sql"; done
completo=$(escalar "$huella")

echo '== Condiciones de las extensiones en el núcleo instalado'
condiciones=(
  "p_perfil_mutacion IS NOT DISTINCT FROM 'cese_ct'"
  "d->>'finalidad' IS NOT DISTINCT FROM 'registrar_cese_contratacion_temporal'"
  "d->>'tipo_recurso' IS NOT DISTINCT FROM 'cese_contratacion_temporal'"
  "p_perfil_mutacion IS NOT DISTINCT FROM 'cierre_expediente_ct'"
  "d->>'finalidad' IS NOT DISTINCT FROM 'cerrar_expediente_tras_cese'"
  "d->>'tipo_recurso' IS NOT DISTINCT FROM 'cierre_expediente_contratacion_temporal'"
  "p_perfil_mutacion IS NOT DISTINCT FROM 'modificacion_tras_nombramiento_ct'"
  "d->>'finalidad' IS NOT DISTINCT FROM 'modificar_expediente_tras_nombramiento'"
  "d->>'tipo_recurso' IS NOT DISTINCT FROM 'modificacion_contratacion_temporal'"
  "d->>'finalidad' IS NOT DISTINCT FROM 'gestion_participaciones_propias'"
  "d ->> 'finalidad' IS NOT DISTINCT FROM 'gestionar_contratacion_temporal'"
)
for condicion in "${condiciones[@]}"; do
  igual "$(escalar "SELECT strpos(pg_get_functiondef('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure), \$q\$$condicion\$q\$)>0")" t "núcleo con: $condicion"
done
# Cada perfil de CT (82, 83, 85) y el portal (84) exigen obligaciones vacías.
igual "$(escalar "SELECT (length(d)-length(replace(d, \$q\$AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)\$q\$, '')))/length(\$q\$AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)\$q\$)
  FROM (SELECT pg_get_functiondef('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure) d) x")" 4 'cese, cierre, modificación y portal exigen obligaciones vacías'

echo '== ACL de las fachadas'
fachadas="ARRAY['vec_autorizacion_atestada_v3.registrar_y_consumir_cese_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
 'vec_autorizacion_atestada_v3.registrar_y_consumir_cierre_expediente_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
 'vec_autorizacion_atestada_v3.registrar_y_consumir_modificacion_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
 'vec_autorizacion_atestada_v3.registrar_y_consumir_portal_candidato_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
 'vec_autorizacion_atestada_v3.registrar_y_consumir_firma_documento_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)']::regprocedure[]"
igual "$(escalar "SELECT bool_or(has_function_privilege('vec_ad3_prueba_ajeno', f, 'EXECUTE') OR has_function_privilege('public', f, 'EXECUTE')) FROM unnest($fachadas) f")" f 'ni PUBLIC ni un rol con privilegios por defecto ejecutan las fachadas'

echo '== Comprobaciones previas de la fachada de firma (42501)'
codigo=$(docker exec -i "$nombre" psql -X -At -U postgres -d postgres <<'SQL' 2>&1 | grep -o 'codigo:[0-9A-Z]*\|aceptada' | tail -1
SET ROLE vec_contratacion_temporal_propietario;
DO $p$ BEGIN
 PERFORM vec_autorizacion_atestada_v3.registrar_y_consumir_firma_documento_ct_v3_atestada(
  convert_to('{"operacion":"contratacion_temporal.documento.firmar","audiencia_consumo":"vec_contratacion_temporal.firma_documento.v1","efecto_ref":"x","huella_efecto_sha256":"h"}','UTF8'),
  convert_to('{"accion":"contratacion_temporal.documento.firmar","modulo_id":"otro_modulo","tipo_recurso":"firma_documento_contratacion_temporal","finalidad":"gestionar_contratacion_temporal","recurso_ref":"x","contexto_recurso_huella_sha256":"h","obligaciones":[]}','UTF8'),
  '\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
 RAISE NOTICE 'aceptada';
EXCEPTION WHEN OTHERS THEN RAISE NOTICE 'codigo:%', SQLSTATE; END $p$;
SQL
)
[[ $codigo == *'codigo:42501'* ]] || { echo "FALLO fachada de firma: $codigo" >&2; exit 1; }
ok 'material ajeno a la firma denegado con 42501 antes del núcleo'

echo '== DOWN fuera de orden y en orden: vuelta exacta a la preimagen'
run -f "/repo/$m/000084_consumidor_portal_candidato_bolsa.down.sql"; ok 'DOWN AD3-84 con AD3-85 instalada después'
run -f "/repo/$m/000082_consumidor_cese_cierre_contratacion_temporal.down.sql"; ok 'DOWN AD3-82 con AD3-83 y AD3-85 instaladas después'
run -f "/repo/$m/000085_consumidor_firma_documento_ct.down.sql"; ok 'DOWN AD3-85'
falla "$m/000085_consumidor_firma_documento_ct.down.sql"
run -f "/repo/$m/000083_consumidor_modificacion_tras_nombramiento_ct.down.sql"; ok 'DOWN AD3-83'
igual "$(escalar "$huella")" "$inicial" 'las DOWN devuelven exactamente núcleo y audiencias'
for x in "${todas[@]}"; do run -f "/repo/$m/$x.up.sql"; done
igual "$(escalar "$huella")" "$completo" 'UP tras DOWN reproduce el mismo núcleo y audiencias'
echo 'OK AD3-82..85: ida y vuelta, condiciones, ACL y fachada de firma'
