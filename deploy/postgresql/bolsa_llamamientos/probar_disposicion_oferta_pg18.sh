#!/usr/bin/env bash
# Ensayo de Bolsa 000029 (disposición a ofertas desde «Mi bolsa») en
# PostgreSQL 18 efímero. Instala la cadena real de Bolsa hasta 000028 con
# dobles de los consumidores AD3 (solo firmas y retorno: la fachada real
# AD3-84 se ensaya en su propio esquema) y comprueba ROLLBACK, UP/DOWN/UP,
# dobles aplicaciones, ACL con roles reales, alta, replay, cotejo del
# candidato con el contexto, negativos, vencimiento, resolución, historia
# inmutable, concurrencia, reinicio y DOWN protegido. Los datos viven en
# /dev/shm (montados con -v, sin volúmenes anónimos) y se borran al terminar.
set -Eeuo pipefail
repo=$(git -C "$(dirname -- "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)
pruebas=$repo/deploy/postgresql/bolsa_llamamientos/pruebas_sql/disposicion_oferta
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm}
contenedor=vec-pg-disposicion-oferta-$$
datos=/dev/shm/vec-pg-disposicion-oferta-$$
trabajo=$(mktemp -d /dev/shm/vec-disposicion-oferta-XXXXXX)
limpiar() {
  docker rm -f "$contenedor" >/dev/null 2>&1 || true
  if [[ -d "$datos" ]]; then docker run --rm -v "$datos":/d --entrypoint rm "$imagen" -rf /d/18 >/dev/null 2>&1 || true; fi
  rm -rf "$datos" "$trabajo"
}
trap limpiar EXIT
mkdir -p "$datos"
clave=$(od -An -N32 -tx1 /dev/urandom | tr -d '[:space:]')
arrancar() {
  docker run --detach --rm --name "$contenedor" -v "$datos":/var/lib/postgresql -e POSTGRES_PASSWORD="$clave" "$imagen" >/dev/null
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
psql_pg < "$pruebas/dobles_ad3.sql" >/dev/null 2>&1
echo 'GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_bolsa_llamamientos_propietario;' | psql_pg >/dev/null
for f in "$repo"/deploy/postgresql/bolsa_llamamientos/migraciones/0000{01,02,03,04,05,06,07,08,10,11,12,13,14,16,17,18,19,20,21,22,23,24,25,26,28}_*.up.sql; do
  psql_pg < "$f" >/dev/null 2>"$trabajo/migracion.err" || { echo "falló $(basename "$f")" >&2; cat "$trabajo/migracion.err" >&2; exit 1; }
  if [[ $(basename "$f") == 000010_* ]]; then psql_pg < "$repo/deploy/postgresql/bolsa_llamamientos/roles_registrador_frontera_up.sql" >/dev/null; fi
done
m=$repo/deploy/postgresql/bolsa_llamamientos/migraciones/000029_disposicion_oferta_candidato
manifestar='vec_bolsa_llamamientos.manifestar_disposicion_oferta_v1(text,text,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
listar='vec_bolsa_llamamientos.listar_ofertas_candidato_v1(text,timestamptz)'
interna='vec_bolsa_llamamientos.registrar_disposicion_oferta_interna_v1(text,text,text,text,text,timestamptz,text)'
participacion='vec_bolsa_llamamientos.participacion_oferta_candidato_v1(text,text)'
consulta='vec_bolsa_llamamientos.consultar_mi_bolsa_portal_v1(text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
marcas="'vec_bolsa_llamamientos.participaciones_vigentes_candidato_v1(text)','vec_bolsa_llamamientos.firma_marca_consumo_v1(xid8,text,text,text)','vec_bolsa_llamamientos.anotar_consumo_candidato_v1(text,text,text)','vec_bolsa_llamamientos.exigir_consumo_candidato_v1(text[],text)'"
# 1. Ensayo en ROLLBACK: no deja nada.
sed 's/^COMMIT;$/ROLLBACK;/' "$m.up.sql" | psql_pg >/dev/null
psql_pg -tAc "SELECT to_regprocedure('$manifestar') IS NULL AND to_regclass('vec_bolsa_llamamientos.disposicion_oferta_candidato') IS NULL" | grep -qx t || { echo 'ROLLBACK dejó objetos'; exit 1; }
# 2. UP, doble UP rechazado, DOWN sin historia, doble DOWN rechazado, UP otra vez.
psql_pg < "$m.up.sql" >/dev/null
if psql_pg < "$m.up.sql" >/dev/null 2>&1; then echo 'doble UP aceptado'; exit 1; fi
psql_pg < "$m.down.sql" >/dev/null
psql_pg -tAc "SELECT to_regprocedure('$manifestar') IS NULL AND to_regprocedure('$listar') IS NULL AND to_regprocedure('$interna') IS NULL AND to_regprocedure('$participacion') IS NULL AND to_regprocedure('$consulta') IS NULL AND to_regclass('vec_bolsa_llamamientos.disposicion_oferta_candidato') IS NULL AND to_regclass('vec_bolsa_llamamientos.secreto_marca_consumo') IS NULL AND NOT EXISTS (SELECT 1 FROM unnest(ARRAY[$marcas]) f WHERE to_regprocedure(f) IS NOT NULL)" | grep -qx t || { echo 'DOWN incompleto'; exit 1; }
if psql_pg < "$m.down.sql" >/dev/null 2>&1; then echo 'doble DOWN aceptado'; exit 1; fi
psql_pg < "$m.up.sql" >/dev/null
echo 'UP/DOWN/UP y dobles aplicaciones: OK'
# 3. ACL con roles reales.
psql_pg <<SQL
DO \$acl\$ BEGIN
 IF NOT has_function_privilege('vec_bolsa_llamamientos_ejecutor','$manifestar','EXECUTE') OR NOT has_function_privilege('vec_bolsa_llamamientos_ejecutor','$listar','EXECUTE') THEN RAISE EXCEPTION 'el ejecutor no puede usar las funciones públicas'; END IF;
 IF NOT has_function_privilege('vec_bolsa_llamamientos_ejecutor','$consulta','EXECUTE') THEN RAISE EXCEPTION 'el ejecutor no puede consultar Mi bolsa con marca'; END IF;
 IF EXISTS (SELECT 1 FROM unnest(ARRAY[$marcas]) f WHERE has_function_privilege('vec_bolsa_llamamientos_ejecutor', f, 'EXECUTE')) THEN RAISE EXCEPTION 'el ejecutor alcanza la marca de consumo'; END IF;
 IF has_table_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.secreto_marca_consumo','SELECT,INSERT,UPDATE,DELETE') THEN RAISE EXCEPTION 'el ejecutor lee el secreto de la marca'; END IF;
 IF has_function_privilege('vec_bolsa_llamamientos_ejecutor','$interna','EXECUTE') OR has_function_privilege('vec_bolsa_llamamientos_ejecutor','$participacion','EXECUTE') THEN RAISE EXCEPTION 'el ejecutor alcanza el núcleo interno'; END IF;
 IF has_function_privilege('vec_bolsa_llamamientos_registrador_frontera','$manifestar','EXECUTE') OR has_function_privilege('vec_bolsa_llamamientos_registrador_frontera','$listar','EXECUTE') THEN RAISE EXCEPTION 'el registrador de frontera puede usar las funciones'; END IF;
 IF EXISTS(SELECT 1 FROM pg_proc p, aclexplode(p.proacl) a WHERE p.oid IN ('$manifestar'::regprocedure,'$listar'::regprocedure,'$interna'::regprocedure,'$participacion'::regprocedure) AND a.grantee=0) THEN RAISE EXCEPTION 'PUBLIC puede ejecutar'; END IF;
 IF has_table_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.disposicion_oferta_candidato','SELECT,INSERT,UPDATE,DELETE') THEN RAISE EXCEPTION 'el ejecutor usa la tabla'; END IF;
 IF (SELECT NOT prosecdef OR proowner<>'vec_bolsa_llamamientos_propietario'::regrole OR proconfig<>ARRAY['search_path=pg_catalog','TimeZone=UTC','lock_timeout=2s'] FROM pg_proc WHERE oid='$manifestar'::regprocedure) THEN RAISE EXCEPTION 'manifestar sin definidor o configuración'; END IF;
END \$acl\$;
SQL
echo 'ACL: OK'
# 4. Comportamiento como ejecutor, con historia confirmada.
psql_pg < "$pruebas/datos.sql" >/dev/null
psql_pg < "$pruebas/pruebas.sql" 2>&1 | grep NOTICE | sed -E 's/^.*NOTICE: +/  /'
# 5. Concurrencia: dos claves distintas de la misma persona a la vez sobre
#    la oferta abierta: una registra y la otra ve la disposición ya hecha.
lanzar() {
  { echo 'SET ROLE vec_bolsa_llamamientos_ejecutor;'; echo "BEGIN; SELECT * FROM prueba_disp.manifestar('oferta:$(printf '3%.0s' {1..64})','can_disposicion_candidato_D01','clave-concurrente-$1'); SELECT pg_sleep(2); COMMIT;"; } \
    | docker exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U postgres >/dev/null 2>"$trabajo/concurrencia_$1.err"
}
lanzar 1 & p1=$!; sleep 0.5; lanzar 2 & p2=$!
r1=0; r2=0; wait $p1 || r1=$?; wait $p2 || r2=$?
[[ $r1 -eq 0 && $r2 -ne 0 ]] && grep -q 'ya manifestada' "$trabajo/concurrencia_2.err" || { echo "concurrencia: $r1 $r2"; cat "$trabajo"/concurrencia_*.err; exit 1; }
echo 'Concurrencia: OK'
# 6. Reinicio: la historia persiste y el replay devuelve el mismo recibo.
antes=$(psql_pg -tA <<'SQL'
SET ROLE vec_bolsa_llamamientos_ejecutor;
SELECT recibo_ref||'|'||manifestada_en FROM prueba_disp.manifestar('oferta:3333333333333333333333333333333333333333333333333333333333333333','can_disposicion_candidato_D01','clave-concurrente-1');
SQL
)
docker restart "$contenedor" >/dev/null
for _ in $(seq 1 60); do docker exec "$contenedor" pg_isready -U postgres >/dev/null 2>&1 && break; sleep 1; done
sleep 2
despues=$(psql_pg -tA <<'SQL'
SET ROLE vec_bolsa_llamamientos_ejecutor;
SELECT recibo_ref||'|'||manifestada_en FROM prueba_disp.manifestar('oferta:3333333333333333333333333333333333333333333333333333333333333333','can_disposicion_candidato_D01','clave-concurrente-1');
SQL
)
[[ -n $antes && $antes == "$despues" ]] || { echo "replay tras reinicio distinto: $antes / $despues"; exit 1; }
psql_pg -tAc "SET ROLE vec_bolsa_llamamientos_propietario; SELECT count(*) FROM vec_bolsa_llamamientos.disposicion_oferta_candidato" | tail -1 | grep -qx 3 || { echo 'recuento tras reinicio'; exit 1; }
echo 'Reinicio: OK'
# 7. DOWN con historia rechazado.
if psql_pg < "$m.down.sql" >/dev/null 2>&1; then echo 'DOWN con historia aceptado'; exit 1; fi
echo 'DOWN con historia rechazado: OK'
printf 'PG18 Bolsa 000029: ensayo, UP/DOWN/UP, ACL, disposición, replay, cotejo, negativos, concurrencia, reinicio y DOWN protegido OK\n'
