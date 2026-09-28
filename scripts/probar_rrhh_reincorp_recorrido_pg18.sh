#!/usr/bin/env bash
# Cadena SQL CT115 -> CT130/AD3-92 -> Bolsa46 en PG18 efimero.
# Uso: este_script GLOBALS_SQL VOLCADO_PG_DUMP
# El volcado debe contener exclusivamente datos sinteticos y la preimagen
# instalada hasta CT129, AD3-91 y Bolsa45, sin CT130/AD3-92/Bolsa46.
# Exige VEC_REINCORP_DATOS_SINTETICOS=1 del custodio del volcado.
# La fixture sustituye las fachadas criptograficas V3 por dobles explicitos:
# acredita la transaccion y el relevo SQL, no la firma, HTTP ni un E2E.
set -Eeuo pipefail

repo=$(git -C "$(dirname -- "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)
globales=${1:?falta pg_dumpall --globals-only sintetico}
volcado=${2:?falta pg_dump -Fc sintetico}
[[ ${VEC_REINCORP_DATOS_SINTETICOS:-} == 1 ]] || {
  echo 'se exige VEC_REINCORP_DATOS_SINTETICOS=1 para un volcado verificado como sintético' >&2; exit 2;
}
[[ -s $globales && -s $volcado ]] || { echo 'faltan volcados sinteticos' >&2; exit 2; }
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4}
nombre="vec-pg-rrhh-reincorp-$$"
datos="/dev/shm/$nombre"
limpiar() {
  docker rm -f "$nombre" >/dev/null 2>&1 || true
  if [[ -d $datos ]]; then
    docker run --rm -v "$datos:/d" --entrypoint rm "$imagen" -rf /d/18 >/dev/null 2>&1 || true
    rm -rf -- "$datos"
  fi
}
trap limpiar EXIT
mkdir -p "$datos"
docker run -d --rm --network none --name "$nombre" \
  -e POSTGRES_HOST_AUTH_METHOD=trust -v "$datos:/var/lib/postgresql" "$imagen" >/dev/null
for _ in $(seq 1 240); do
  if docker logs "$nombre" 2>&1 | grep -q 'PostgreSQL init process complete' &&
    docker exec "$nombre" pg_isready -q -U postgres -d postgres; then break; fi
  sleep 0.5
done
run() { docker exec -i "$nombre" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d postgres "$@"; }
scalar() { run -At -c "$1"; }
[[ $(scalar 'SHOW server_version') == 18.4* ]] || { echo 'PG18.4 no disponible' >&2; exit 2; }
[[ -z $(docker inspect --format '{{range .Mounts}}{{if eq .Type "volume"}}{{.Name}}{{end}}{{end}}' "$nombre") ]] || {
  echo 'volumen anonimo inesperado' >&2; exit 65;
}

# El volcado es de una base sintetica preexistente; nunca se instala sobre ella.
docker exec -i "$nombre" psql -X -q -U postgres -d postgres <"$globales" >/dev/null 2>&1 || true
docker exec -i "$nombre" pg_restore -U postgres -d postgres <"$volcado" >/dev/null 2>&1 || true
preimagen="SELECT to_regclass('vec_contratacion_temporal.cese_nombramiento_v1') IS NOT NULL
 AND to_regclass('vec_bolsa_llamamientos.restriccion_cese_bolsa') IS NOT NULL
 AND to_regprocedure('vec_contratacion_temporal.posicion_contrato_bolsa_v1(xid8)') IS NOT NULL
 AND to_regprocedure('vec_contratacion_temporal.verificar_cese_publicado_bolsa_v1(text,text,bigint)') IS NOT NULL
 AND to_regclass('vec_contratacion_temporal.reincorporacion_titular_v1') IS NULL
 AND to_regclass('vec_bolsa_llamamientos.reincorporacion_titular_ct') IS NULL
 AND to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_reincorporacion_titular_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL"
[[ $(scalar "$preimagen") == t ]] || { echo 'preimagen CT129/AD3-91/Bolsa45 incompatible o ya migrada' >&2; exit 2; }
ct="$repo/deploy/postgresql/contratacion_temporal/migraciones/000130_reincorporacion_titular"
ad="$repo/deploy/postgresql/autorizacion_atestada_v3/migraciones/000092_consumidor_reincorporacion_titular_ct"
bolsa="$repo/deploy/postgresql/bolsa_llamamientos/migraciones/000046_reincorporacion_titular_ct"
for m in "$ad" "$ct" "$bolsa"; do
  sed 's/^COMMIT;$/ROLLBACK;/' "$m.up.sql" | run >/dev/null
  run <"$m.up.sql" >/dev/null
  if run <"$m.up.sql" >/dev/null 2>&1; then echo "doble UP aceptado: $(basename "$m")" >&2; exit 1; fi
done
[[ $(scalar "SELECT to_regclass('vec_contratacion_temporal.reincorporacion_titular_v1') IS NOT NULL
 AND to_regclass('vec_bolsa_llamamientos.reincorporacion_titular_ct') IS NOT NULL") == t ]] || {
  echo 'migraciones no instaladas' >&2; exit 1;
}
echo 'OK AD3-92, CT130, Bolsa46: ROLLBACK, UP y doble UP rechazado'

aux="$repo/scripts/rrhh_reincorp"
bolsa_antes=$(scalar "SELECT (SELECT count(*) FROM vec_bolsa_llamamientos.restriccion_cese_bolsa)::text||'/'||
 (SELECT count(*) FROM vec_bolsa_llamamientos.situacion_participacion)::text||'/'||
 (SELECT count(*) FROM vec_bolsa_llamamientos.llamamiento_integracion_desarrollo)::text||'/'||
 (SELECT count(*) FROM vec_bolsa_llamamientos.integracion_desarrollo)::text")
run -At <"$aux/recorrido.sql" >"$datos/resultado" || { tail -30 "$datos/resultado" >&2; exit 1; }
grep -q '^RRHH_REINCORP_CADENA_OK$' "$datos/resultado" || {
  tail -30 "$datos/resultado" >&2; echo 'falta marca de cadena' >&2; exit 1;
}
echo 'OK antecedente CT115 fin_sustitucion, CT130 recibo e idempotencia, feed y B46 sin duplicado'
recibo_antes=$(scalar "SELECT md5(recibo_json::text)||'/'||confirmada_en::text FROM vec_contratacion_temporal.reincorporacion_titular_v1 WHERE evento_ref='evento:ct130:ensayo'")
[[ -n $recibo_antes ]] || { echo 'falta recibo CT130' >&2; exit 1; }

# Puerta opcional CT134/AD3-98: sólo un árbol cuyo HEAD sea el hash exacto
# revisado. Instala en esta misma base efímera y comprueba firmas; la lectura
# V3 real y HTTP necesitan el siguiente recorrido de aplicación.
if [[ -n ${VEC_REINCORP_CT134_HASH:-} ]]; then
  arbol=${VEC_REINCORP_CT134_ROOT:-$repo}
  [[ $(git -C "$arbol" rev-parse HEAD) == "$VEC_REINCORP_CT134_HASH" ]] || {
    echo 'CT134: HEAD no coincide con el hash revisado' >&2; exit 2;
  }
  ad98="$arbol/deploy/postgresql/autorizacion_atestada_v3/migraciones/000098_consumidor_lectura_reincorporacion_ct.up.sql"
  ct134="$arbol/deploy/postgresql/contratacion_temporal/migraciones/000134_lectura_reincorporacion_titular.up.sql"
  [[ -s $ad98 && -s $ct134 ]] || { echo 'CT134/AD3-98: falta una migración' >&2; exit 2; }
  for m in "$ad98" "$ct134"; do
    sed 's/^COMMIT;$/ROLLBACK;/' "$m" | run >/dev/null
    run <"$m" >/dev/null
    if run <"$m" >/dev/null 2>&1; then echo "doble UP aceptado: $(basename "$m")" >&2; exit 1; fi
  done
  [[ $(scalar "SELECT to_regprocedure('vec_contratacion_temporal.leer_antecedente_reincorporacion_titular_atestada_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL") == t ]] || {
    echo 'CT134: falta fachada de lectura' >&2; exit 1;
  }
  echo "OK AD3-98/CT134 instaladas sobre la cadena en $VEC_REINCORP_CT134_HASH (puerta estructural)"
fi

docker restart "$nombre" >/dev/null
for _ in $(seq 1 120); do docker exec "$nombre" pg_isready -q -U postgres -d postgres && break; sleep 0.5; done
run -At <"$aux/reinicio.sql" >"$datos/reinicio" || { tail -20 "$datos/reinicio" >&2; exit 1; }
grep -q '^RRHH_REINCORP_REINICIO_OK$' "$datos/reinicio" || { echo 'fallo al recuperar tras reinicio' >&2; exit 1; }
if [[ -n ${VEC_REINCORP_CT134_HASH:-} ]]; then
  [[ $(scalar "SELECT to_regprocedure('vec_contratacion_temporal.leer_antecedente_reincorporacion_titular_atestada_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL") == t ]] || {
    echo 'CT134 ausente tras reinicio' >&2; exit 1;
  }
fi
[[ $(scalar "SELECT md5(recibo_json::text)||'/'||confirmada_en::text FROM vec_contratacion_temporal.reincorporacion_titular_v1 WHERE evento_ref='evento:ct130:ensayo'") == "$recibo_antes" ]] || {
  echo 'recibo o fecha cambiaron tras reinicio' >&2; exit 1;
}
[[ $(scalar "SELECT (SELECT count(*) FROM vec_bolsa_llamamientos.restriccion_cese_bolsa)::text||'/'||
 (SELECT count(*) FROM vec_bolsa_llamamientos.situacion_participacion)::text||'/'||
 (SELECT count(*) FROM vec_bolsa_llamamientos.llamamiento_integracion_desarrollo)::text||'/'||
 (SELECT count(*) FROM vec_bolsa_llamamientos.integracion_desarrollo)::text") == "$bolsa_antes" ]] || {
  echo 'B46 alteró restricciones, situaciones o llamamientos (+5/+9)' >&2; exit 1;
}
echo 'OK reinicio: mismo recibo y una sola fila CT/Bolsa'
echo 'LIMITE: V3 de CT130/B46 usa doble sintético; se acredita 42501 SQL ajeno, no HTTP 403 ni navegador.'
echo 'CT134/AD3-98: la puerta opcional acredita migraciones y persistencia de firmas, no lectura con V3 real.'
