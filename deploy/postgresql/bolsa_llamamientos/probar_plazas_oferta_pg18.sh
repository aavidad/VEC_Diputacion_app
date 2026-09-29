#!/usr/bin/env bash
# Bolsa 000058 (ofertas con varias plazas) en PostgreSQL 18 efímero. Instala
# la cadena real de Bolsa hasta B54 con dobles de los consumidores AD3 (solo
# firmas y retorno: la criptografía real se ensaya en su propio esquema) y
# comprueba ROLLBACK, UP/DOWN/UP, doble UP, ACL, política con plazas,
# publicación ligada al número de plazas, adjudicación por orden, respuesta,
# renuncia y siguiente, llamamiento directo, modo sucesivo, ofertas
# anteriores, «Mi bolsa», historia inmutable, concurrencia, reinicio y DOWN
# protegido. Los datos viven en /dev/shm (montados con -v, sin volúmenes
# anónimos) y se borran al terminar.
set -Eeuo pipefail
repo=$(git -C "$(dirname -- "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)
pruebas=$repo/deploy/postgresql/bolsa_llamamientos/pruebas_sql
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm}
contenedor=vec-pg-plazas-oferta-$$
datos=/dev/shm/vec-pg-plazas-oferta-$$
trabajo=$(mktemp -d /dev/shm/vec-plazas-oferta-XXXXXX)
limpiar() {
  docker rm -f "$contenedor" >/dev/null 2>&1 || true
  if [[ -d "$datos" ]]; then docker run --rm --pull never -v "$datos":/d --entrypoint rm "$imagen" -rf /d/18 >/dev/null 2>&1 || true; fi
  rm -rf "$datos" "$trabajo"
}
trap limpiar EXIT
mkdir -p "$datos"
clave=$(od -An -N32 -tx1 /dev/urandom | tr -d '[:space:]')
arrancar() {
  docker run --detach --rm --pull never --network none --name "$contenedor" -v "$datos":/var/lib/postgresql -e POSTGRES_PASSWORD="$clave" "$imagen" >/dev/null
  for _ in $(seq 1 60); do docker exec "$contenedor" pg_isready -U postgres >/dev/null 2>&1 && break; sleep 1; done
  sleep 2
}
arrancar
if [[ -n $(docker inspect --format '{{range .Mounts}}{{if eq .Type "volume"}}{{.Name}}{{end}}{{end}}' "$contenedor") ]]; then
  echo 'volumen anónimo inesperado' >&2; exit 65
fi
psql_pg() { docker exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U postgres "$@"; }
for ruta in \
  deploy/postgresql/autorizacion/roles_up.sql \
  deploy/postgresql/autorizacion/migraciones/000001_autorizacion.up.sql \
  deploy/postgresql/ejecucion_documental_v4/migraciones_autorizacion/000002_vinculo_autenticacion_actor_actual.up.sql \
  deploy/postgresql/bolsa_llamamientos/roles_up.sql \
  deploy/postgresql/bolsa_llamamientos/migraciones_autorizacion/000001_revalidacion_llamamientos.up.sql; do
  psql_pg < "$repo/$ruta" >/dev/null
done
psql_pg < "$pruebas/disposicion_oferta/dobles_ad3.sql" >/dev/null 2>&1
psql_pg < "$pruebas/plazas_oferta/dobles_ad3.sql" >/dev/null
m=$repo/deploy/postgresql/bolsa_llamamientos/migraciones
for f in "$m"/0000{01,02,03,04,05,06,07,08,10,11,12,13,14,16,17,18,19,20,21,22,23,24,25,26,28,29}_*.up.sql; do
  psql_pg < "$f" >/dev/null 2>"$trabajo/migracion.err" || { echo "falló $(basename "$f")" >&2; cat "$trabajo/migracion.err" >&2; exit 1; }
  if [[ $(basename "$f") == 000010_* ]]; then psql_pg < "$repo/deploy/postgresql/bolsa_llamamientos/roles_registrador_frontera_up.sql" >/dev/null; fi
done
psql_pg < "$pruebas/disposicion_oferta/datos.sql" >/dev/null
psql_pg < "$m/000047_politica_ofertas_ejemplo.up.sql" >/dev/null
psql_pg < "$repo/deploy/postgresql/bolsa_llamamientos/roles_calculador_politica_up.sql" >/dev/null
psql_pg < "$m/000051_consulta_politica_ofertas_v3.up.sql" >/dev/null
psql_pg < "$m/000054_plazo_ofertas_48_horas.up.sql" >/dev/null

b58=$m/000058_plazas_oferta
registrar='vec_bolsa_llamamientos.registrar_acto_plaza_oferta_v1(text,text,text,integer,text,text,integer,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
# 1. Ensayo en ROLLBACK: no deja nada.
sed 's/^COMMIT;$/ROLLBACK;/' "$b58.up.sql" | psql_pg >/dev/null
psql_pg -tAc "SELECT to_regprocedure('$registrar') IS NULL AND to_regclass('vec_bolsa_llamamientos.plazas_oferta') IS NULL" | grep -qx t || { echo 'B58 ROLLBACK dejó objetos' >&2; exit 1; }
# 2. UP, doble UP rechazado, DOWN sin historia restaura las huellas exactas, UP otra vez.
huellas="SELECT md5(pg_get_functiondef('vec_bolsa_llamamientos.listar_ofertas_candidato_v1(text,timestamptz)'::regprocedure))||md5(pg_get_functiondef('vec_bolsa_llamamientos.listar_ofertas_bolsa_v1(text,timestamptz,integer)'::regprocedure))||md5(pg_get_functiondef('vec_bolsa_llamamientos.publicar_politica_ofertas_v1(text,bigint,jsonb,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure))"
antes=$(psql_pg -tAc "$huellas")
psql_pg < "$b58.up.sql" >/dev/null
if psql_pg < "$b58.up.sql" >/dev/null 2>&1; then echo 'B58 doble UP aceptado' >&2; exit 1; fi
psql_pg < "$b58.down.sql" >/dev/null
[[ $(psql_pg -tAc "$huellas") == "$antes" ]] || { echo 'B58 DOWN no restauró las funciones' >&2; exit 1; }
psql_pg -tAc "SELECT to_regprocedure('$registrar') IS NULL AND to_regclass('vec_bolsa_llamamientos.acto_plaza_oferta') IS NULL AND has_function_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.resolver_oferta_v1(text,text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')" | grep -qx t || { echo 'B58 DOWN incompleto' >&2; exit 1; }
if psql_pg < "$b58.down.sql" >/dev/null 2>&1; then echo 'B58 doble DOWN aceptado' >&2; exit 1; fi
psql_pg < "$b58.up.sql" >/dev/null
echo 'PG18 B58: ROLLBACK, UP, doble UP, DOWN exacto, doble DOWN y UP OK'

# 3. Funcionamiento: política, publicación, plazas, respuestas y «Mi bolsa».
if ! psql_pg < "$pruebas/plazas_oferta/pruebas.sql" >"$trabajo/pruebas.out" 2>&1; then
  echo 'pruebas B58 fallidas' >&2; grep -v '^$' "$trabajo/pruebas.out" | tail -n 15 >&2; exit 1
fi
grep NOTICE "$trabajo/pruebas.out" | sed -E 's/^.*NOTICE: +/  /' || true

# 4. Concurrencia: dos sesiones confirman la misma plaza con claves distintas;
# solo una registra el acto y la otra ve la plaza cambiada.
for n in 1 2; do
  psql_pg -v VERBOSITY=verbose -tAc "SET ROLE vec_bolsa_llamamientos_ejecutor; SELECT prueba_plazas.acto_conc('clave-concurrente-$n')" >"$trabajo/conc-$n.out" 2>&1 &
done
wait || true
[[ $(psql_pg -tAc "SET ROLE vec_bolsa_llamamientos_propietario; SELECT count(*) FROM vec_bolsa_llamamientos.acto_plaza_oferta WHERE clave_idempotencia LIKE 'clave-concurrente-%'" | tail -1) == 1 ]] \
  || { echo 'concurrencia: se registró más de un acto' >&2; cat "$trabajo"/conc-*.out >&2; exit 1; }
grep -q VBO04 "$trabajo"/conc-*.out || { echo 'concurrencia: la sesión perdedora no vio VBO04' >&2; cat "$trabajo"/conc-*.out >&2; exit 1; }
echo 'PG18 B58: concurrencia sobre la misma plaza OK'

# 5. Reinicio: la historia y el replay sobreviven.
antes=$(psql_pg -tAc "SET ROLE vec_bolsa_llamamientos_ejecutor; SELECT prueba_plazas.replay_recibo()" | tail -1)
docker restart "$contenedor" >/dev/null
for _ in $(seq 1 60); do docker exec "$contenedor" pg_isready -U postgres >/dev/null 2>&1 && break; sleep 1; done
sleep 1
despues=$(psql_pg -tAc "SET ROLE vec_bolsa_llamamientos_ejecutor; SELECT prueba_plazas.replay_recibo()" | tail -1)
[[ -n $antes && $antes == "$despues" ]] || { echo "replay tras reinicio distinto: $antes / $despues" >&2; exit 1; }
echo 'PG18 B58: replay tras reinicio OK'

# 6. DOWN con historia rechazado y la historia intacta.
n=$(psql_pg -tAc "SET ROLE vec_bolsa_llamamientos_propietario; SELECT count(*) FROM vec_bolsa_llamamientos.acto_plaza_oferta" | tail -1)
if psql_pg < "$b58.down.sql" >/dev/null 2>&1; then echo 'B58 DOWN con historia aceptado' >&2; exit 1; fi
[[ $(psql_pg -tAc "SET ROLE vec_bolsa_llamamientos_propietario; SELECT count(*) FROM vec_bolsa_llamamientos.acto_plaza_oferta" | tail -1) == "$n" ]] || { echo 'B58 historia perdida' >&2; exit 1; }
printf 'PG18 Bolsa 000058: ensayo, UP/DOWN/UP, ACL, política, publicación, plazas, respuestas, llamamiento directo, Mi bolsa, concurrencia, reinicio y DOWN protegido OK\n'
