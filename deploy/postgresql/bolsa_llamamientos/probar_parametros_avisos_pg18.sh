#!/usr/bin/env bash
# Ensayo de Bolsa 000041 (parámetros de avisos y marcas desde el catálogo:
# b16, b17 y b19) en PostgreSQL 18 efímero. Instala la cadena real de Bolsa
# hasta 000030 con dobles de los consumidores AD3 y AD3-84 real; comprueba
# ROLLBACK, UP/DOWN/UP y dobles aplicaciones, ACL con roles reales, la
# conducta por defecto idéntica a 000020, publicación y replay, bandeja v2,
# marcas, exclusión en la base, negativos, historia inmutable, concurrencia,
# reinicio y DOWN protegido. Datos en /dev/shm montados con -v (sin volúmenes
# anónimos); se borran al terminar.
set -Eeuo pipefail
repo=$(git -C "$(dirname -- "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)
pruebas=$repo/deploy/postgresql/bolsa_llamamientos/pruebas_sql/parametros_avisos
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm}
contenedor=vec-pg-parametros-avisos-$$
datos=/dev/shm/vec-pg-parametros-avisos-$$
trabajo=$(mktemp -d /dev/shm/vec-parametros-avisos-XXXXXX)
limpiar() {
  docker rm -f "$contenedor" >/dev/null 2>&1 || true
  if [[ -d "$datos" ]]; then docker run --rm -v "$datos":/d --entrypoint rm "$imagen" -rf /d/18 >/dev/null 2>&1 || true; fi
  rm -rf "$datos" "$trabajo"
}
trap limpiar EXIT
mkdir -p "$datos"
clave=$(od -An -N32 -tx1 /dev/urandom | tr -d '[:space:]')
esperar() {
  for _ in $(seq 1 60); do docker exec "$contenedor" pg_isready -U postgres >/dev/null 2>&1 && break; sleep 1; done
  sleep 2
}
docker run --detach --rm --name "$contenedor" -v "$datos":/var/lib/postgresql -e POSTGRES_PASSWORD="$clave" "$imagen" >/dev/null
esperar
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
psql_pg < "$repo/deploy/postgresql/bolsa_llamamientos/pruebas_sql/contacto_origen/dobles_ad3.sql" >/dev/null 2>&1
psql_pg < "$repo/deploy/postgresql/bolsa_llamamientos/pruebas_sql/contacto_propio/preimagen_ad3.sql" >/dev/null
psql_pg < "$repo/deploy/postgresql/autorizacion_atestada_v3/migraciones/000084_consumidor_portal_candidato_bolsa.up.sql" >/dev/null
for f in "$repo"/deploy/postgresql/bolsa_llamamientos/migraciones/0000{01,02,03,04,05,06,07,08,10,11,12,13,14,16,17,18,19,20,21,22,23,24,25,26,28,29,30}_*.up.sql; do
  psql_pg < "$f" >/dev/null 2>"$trabajo/migracion.err" || { echo "falló $(basename "$f")" >&2; cat "$trabajo/migracion.err" >&2; exit 1; }
  if [[ $(basename "$f") == 000010_* ]]; then psql_pg < "$repo/deploy/postgresql/bolsa_llamamientos/roles_registrador_frontera_up.sql" >/dev/null; fi
done
m=$repo/deploy/postgresql/bolsa_llamamientos/migraciones/000041_parametros_avisos_y_marcas
v1='vec_bolsa_llamamientos.consultar_avisos_rrhh_v1(timestamptz)'
v2='vec_bolsa_llamamientos.consultar_avisos_rrhh_v2(timestamptz)'
publicar='vec_bolsa_llamamientos.publicar_politica_avisos_bolsa_v1(text,text,integer,integer,integer,integer,text,text[])'
consultar='vec_bolsa_llamamientos.consultar_politica_avisos_bolsa_v1()'
marcas='vec_bolsa_llamamientos.consultar_marcas_participaciones_v1(text,timestamptz)'
internas=(
  'vec_bolsa_llamamientos.politica_avisos_bolsa_vigente()'
  'vec_bolsa_llamamientos.situaciones_en_v1(timestamptz)'
  'vec_bolsa_llamamientos.presta_servicios_en_v1(timestamptz)'
  'vec_bolsa_llamamientos.encadenamiento_en_v1(timestamptz)'
  'vec_bolsa_llamamientos.exigir_no_presta_servicios()'
)
v1_previa=$(psql_pg -tAc "SELECT md5(pg_get_functiondef('$v1'::regprocedure))||coalesce((SELECT proacl::text FROM pg_proc WHERE oid='$v1'::regprocedure),'')")
# 1. ROLLBACK, UP, doble UP, DOWN, doble DOWN y UP.
sed 's/^COMMIT;$/ROLLBACK;/' "$m.up.sql" | psql_pg >/dev/null
psql_pg -tAc "SELECT to_regprocedure('$v2') IS NULL AND to_regclass('vec_bolsa_llamamientos.politica_avisos_bolsa') IS NULL" | grep -qx t || { echo 'ROLLBACK dejó objetos'; exit 1; }
psql_pg < "$m.up.sql" >/dev/null
if psql_pg < "$m.up.sql" >/dev/null 2>&1; then echo 'doble UP aceptado'; exit 1; fi
psql_pg < "$m.down.sql" >/dev/null
restos=$(psql_pg -tAc "SELECT count(*) FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname='vec_bolsa_llamamientos' AND p.proname IN ('consultar_avisos_rrhh_v2','publicar_politica_avisos_bolsa_v1','consultar_politica_avisos_bolsa_v1','consultar_marcas_participaciones_v1','politica_avisos_bolsa_vigente','situaciones_en_v1','presta_servicios_en_v1','encadenamiento_en_v1','exigir_no_presta_servicios')")
[[ $restos == 0 ]] && psql_pg -tAc "SELECT to_regclass('vec_bolsa_llamamientos.politica_avisos_bolsa') IS NULL AND NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='llamamiento_emitido_presta_servicios')" | grep -qx t || { echo 'DOWN incompleto'; exit 1; }
if psql_pg < "$m.down.sql" >/dev/null 2>&1; then echo 'doble DOWN aceptado'; exit 1; fi
psql_pg < "$m.up.sql" >/dev/null
[[ $(psql_pg -tAc "SELECT md5(pg_get_functiondef('$v1'::regprocedure))||coalesce((SELECT proacl::text FROM pg_proc WHERE oid='$v1'::regprocedure),'')") == "$v1_previa" ]] || { echo '000041 alteró la bandeja v1'; exit 1; }
echo 'Bolsa 000041 ROLLBACK, UP/DOWN/UP, dobles aplicaciones y v1 intacta: OK'
# 2. ACL con roles reales.
lista_internas=$(printf "'%s'::regprocedure," "${internas[@]}"); lista_internas=${lista_internas%,}
psql_pg <<SQL
DO \$acl\$ BEGIN
 IF NOT has_function_privilege('vec_bolsa_llamamientos_ejecutor','$v2','EXECUTE') OR NOT has_function_privilege('vec_bolsa_llamamientos_ejecutor','$publicar','EXECUTE')
    OR NOT has_function_privilege('vec_bolsa_llamamientos_ejecutor','$consultar','EXECUTE') OR NOT has_function_privilege('vec_bolsa_llamamientos_ejecutor','$marcas','EXECUTE') THEN
  RAISE EXCEPTION 'el ejecutor no puede usar las funciones públicas'; END IF;
 IF EXISTS (SELECT 1 FROM unnest(ARRAY[$lista_internas]) f WHERE has_function_privilege('vec_bolsa_llamamientos_ejecutor', f, 'EXECUTE')) THEN
  RAISE EXCEPTION 'el ejecutor alcanza funciones internas'; END IF;
 IF has_function_privilege('vec_bolsa_llamamientos_registrador_frontera','$publicar','EXECUTE') OR has_function_privilege('vec_bolsa_llamamientos_registrador_frontera','$marcas','EXECUTE') THEN
  RAISE EXCEPTION 'el registrador de frontera alcanza 000041'; END IF;
 IF EXISTS (SELECT 1 FROM pg_proc p, aclexplode(p.proacl) a WHERE p.oid IN ('$v2'::regprocedure,'$publicar'::regprocedure,'$consultar'::regprocedure,'$marcas'::regprocedure,$lista_internas) AND a.grantee=0) THEN
  RAISE EXCEPTION 'PUBLIC puede ejecutar'; END IF;
 IF EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid IN ('$v2'::regprocedure,'$publicar'::regprocedure,'$consultar'::regprocedure,'$marcas'::regprocedure,$lista_internas)
            AND pg_get_userbyid(p.proowner) <> 'vec_bolsa_llamamientos_propietario') THEN RAISE EXCEPTION 'propietario de funciones'; END IF;
 IF has_table_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.politica_avisos_bolsa','SELECT,INSERT,UPDATE,DELETE') THEN RAISE EXCEPTION 'el ejecutor usa la tabla'; END IF;
 IF NOT (SELECT relrowsecurity AND relforcerowsecurity FROM pg_class WHERE oid='vec_bolsa_llamamientos.politica_avisos_bolsa'::regclass) THEN RAISE EXCEPTION 'RLS FORCE'; END IF;
END \$acl\$;
SQL
echo 'ACL: OK'
# 3. Comportamiento con datos sintéticos.
psql_pg < "$pruebas/datos.sql" >/dev/null
psql_pg < "$pruebas/pruebas.sql" 2>&1 | grep -E 'NOTICE|ERROR' | sed -E 's/^.*(NOTICE|ERROR): +/  \1 /' | tee "$trabajo/pruebas.log"
if grep -q ERROR "$trabajo/pruebas.log" || [[ $(grep -c 'NOTICE OK' "$trabajo/pruebas.log") -ne 10 ]]; then echo 'pruebas de conducta fallidas' >&2; exit 1; fi
# 4. Concurrencia: dos publicaciones distintas a la vez quedan en versiones
#    consecutivas, sin error ni versión repetida.
antes=$(psql_pg -tAc "SELECT max(version) FROM vec_bolsa_llamamientos.politica_avisos_bolsa")
lanzar() {
  { echo 'SET ROLE vec_bolsa_llamamientos_ejecutor;'; echo "BEGIN; SELECT * FROM prueba_pa.publicar('concurrente:$1', 36, $1, NULL, NULL, NULL, NULL); SELECT pg_sleep(2); COMMIT;"; } \
    | docker exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U postgres >/dev/null 2>"$trabajo/concurrencia_$1.err"
}
lanzar 11 & p1=$!; sleep 0.5; lanzar 12 & p2=$!
r1=0; r2=0; wait $p1 || r1=$?; wait $p2 || r2=$?
[[ $r1 -eq 0 && $r2 -eq 0 ]] || { echo "concurrencia: $r1 $r2"; cat "$trabajo"/concurrencia_*.err; exit 1; }
[[ $(psql_pg -tAc "SELECT string_agg(continuado_antelacion_dias::text, ',' ORDER BY version) FROM vec_bolsa_llamamientos.politica_avisos_bolsa WHERE version > $antes") == '11,12' ]] || { echo 'concurrencia: versiones inesperadas'; exit 1; }
echo 'Concurrencia: OK'
# 5. Reinicio: la política vigente persiste.
vigente=$(psql_pg -tAc "SET ROLE vec_bolsa_llamamientos_ejecutor; SELECT version||'|'||continuado_antelacion_dias FROM $consultar")
docker restart "$contenedor" >/dev/null
esperar
[[ -n $vigente && $vigente == "$(psql_pg -tAc "SET ROLE vec_bolsa_llamamientos_ejecutor; SELECT version||'|'||continuado_antelacion_dias FROM $consultar")" ]] || { echo 'política distinta tras reinicio'; exit 1; }
echo 'Reinicio: OK'
# 6. DOWN con historia rechazado.
if psql_pg < "$m.down.sql" >/dev/null 2>&1; then echo 'DOWN de 000041 con historia aceptado'; exit 1; fi
echo 'DOWN con historia rechazado: OK'
printf 'PG18 Bolsa 000041: ROLLBACK, UP/DOWN/UP, ACL, conducta por defecto, publicación, avisos, marcas, exclusión, negativos, concurrencia, reinicio y DOWN protegido OK\n'
