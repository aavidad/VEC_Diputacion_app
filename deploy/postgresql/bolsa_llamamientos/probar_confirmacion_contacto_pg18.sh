#!/usr/bin/env bash
# Ensayo de AD3-86 y Bolsa 000040 (confirmación del contacto propio desde
# «Mi bolsa», duda 45) en PostgreSQL 18 efímero. Sobre una preimagen mínima
# del núcleo AD3 (tabla de claves con la lista única de audiencias y un doble
# del núcleo que conserva sus marcas) instala AD3-84 y AD3-86 reales, y la
# cadena real de Bolsa hasta 000035 con dobles de los demás consumidores.
# Comprueba AD3-86 antes de AD3-84 rechazada, ROLLBACK, UP/DOWN/UP y dobles
# aplicaciones de ambas, ACL con roles reales, la fachada, la confirmación
# con replay, versión vista, cotejo y negativos, historia inmutable,
# concurrencia, reinicio y DOWN protegidos. Datos en /dev/shm montados con
# -v (sin volúmenes anónimos); se borran al terminar.
set -Eeuo pipefail
repo=$(git -C "$(dirname -- "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)
pruebas=$repo/deploy/postgresql/bolsa_llamamientos/pruebas_sql/contacto_propio
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm}
contenedor=vec-pg-contacto-propio-$$
datos=/dev/shm/vec-pg-contacto-propio-$$
trabajo=$(mktemp -d /dev/shm/vec-contacto-propio-XXXXXX)
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
psql_pg < "$repo/deploy/postgresql/bolsa_llamamientos/pruebas_sql/contacto_origen/dobles_ad3.sql" >/dev/null 2>&1
psql_pg < "$pruebas/preimagen_ad3.sql" >/dev/null
ad386=$repo/deploy/postgresql/autorizacion_atestada_v3/migraciones/000086_consumidor_contacto_propio_bolsa
m=$repo/deploy/postgresql/bolsa_llamamientos/migraciones/000040_confirmacion_contacto_propio
fachada='vec_autorizacion_atestada_v3.registrar_y_consumir_contacto_propio_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
nucleo='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
if psql_pg < "$ad386.up.sql" >/dev/null 2>&1; then echo 'AD3-86 aceptada sin AD3-84'; exit 1; fi
psql_pg < "$repo/deploy/postgresql/autorizacion_atestada_v3/migraciones/000084_consumidor_portal_candidato_bolsa.up.sql" >/dev/null
# 1. AD3-86: ROLLBACK sin rastro, UP, doble UP, DOWN que deja el núcleo y las
#    audiencias como estaban, doble DOWN y UP otra vez.
antes=$(psql_pg -tAc "SELECT md5(pg_get_functiondef('$nucleo'::regprocedure))||md5(pg_get_constraintdef((SELECT oid FROM pg_constraint WHERE conname='clave_capacidad_version_audiencia_consumo_check')))")
sed 's/^COMMIT;$/ROLLBACK;/' "$ad386.up.sql" | psql_pg >/dev/null
psql_pg -tAc "SELECT to_regprocedure('$fachada') IS NULL" | grep -qx t || { echo 'ROLLBACK de AD3-86 dejó la fachada'; exit 1; }
psql_pg < "$ad386.up.sql" >/dev/null
if psql_pg < "$ad386.up.sql" >/dev/null 2>&1; then echo 'doble UP de AD3-86 aceptado'; exit 1; fi
psql_pg < "$ad386.down.sql" >/dev/null
despues=$(psql_pg -tAc "SELECT md5(pg_get_functiondef('$nucleo'::regprocedure))||md5(pg_get_constraintdef((SELECT oid FROM pg_constraint WHERE conname='clave_capacidad_version_audiencia_consumo_check')))")
[[ $antes == "$despues" ]] || { echo 'DOWN de AD3-86 no restaura el núcleo o las audiencias'; exit 1; }
if psql_pg < "$ad386.down.sql" >/dev/null 2>&1; then echo 'doble DOWN de AD3-86 aceptado'; exit 1; fi
psql_pg < "$ad386.up.sql" >/dev/null
echo 'AD3-86 UP/DOWN/UP y dobles aplicaciones: OK'
for f in "$repo"/deploy/postgresql/bolsa_llamamientos/migraciones/0000{01,02,03,04,05,06,07,08,10,11,12,13,14,16,17,18,19,20,21,22,23,24,25,26,28,29,35}_*.up.sql; do
  psql_pg < "$f" >/dev/null 2>"$trabajo/migracion.err" || { echo "falló $(basename "$f")" >&2; cat "$trabajo/migracion.err" >&2; exit 1; }
  if [[ $(basename "$f") == 000010_* ]]; then psql_pg < "$repo/deploy/postgresql/bolsa_llamamientos/roles_registrador_frontera_up.sql" >/dev/null; fi
done
confirmar='vec_bolsa_llamamientos.confirmar_contacto_propio_v1(text,text,bigint,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
leer='vec_bolsa_llamamientos.leer_contacto_candidato_v1(text,timestamptz)'
leer_confirmacion='vec_bolsa_llamamientos.leer_confirmacion_contacto_participacion_v1(text,bigint)'
interna='vec_bolsa_llamamientos.registrar_confirmacion_contacto_interna_v1(text,text,text,bigint,text,text,timestamptz,text)'
participacion='vec_bolsa_llamamientos.participacion_contacto_candidato_v1(text,text)'
# 2. Bolsa 000040: ROLLBACK, UP, doble UP, DOWN, doble DOWN y UP.
sed 's/^COMMIT;$/ROLLBACK;/' "$m.up.sql" | psql_pg >/dev/null
psql_pg -tAc "SELECT to_regprocedure('$confirmar') IS NULL AND to_regclass('vec_bolsa_llamamientos.confirmacion_contacto_participacion') IS NULL" | grep -qx t || { echo 'ROLLBACK dejó objetos'; exit 1; }
psql_pg < "$m.up.sql" >/dev/null
if psql_pg < "$m.up.sql" >/dev/null 2>&1; then echo 'doble UP aceptado'; exit 1; fi
psql_pg < "$m.down.sql" >/dev/null
psql_pg -tAc "SELECT to_regprocedure('$confirmar') IS NULL AND to_regprocedure('$leer') IS NULL AND to_regprocedure('$leer_confirmacion') IS NULL AND to_regprocedure('$interna') IS NULL AND to_regprocedure('$participacion') IS NULL AND to_regclass('vec_bolsa_llamamientos.confirmacion_contacto_participacion') IS NULL" | grep -qx t || { echo 'DOWN incompleto'; exit 1; }
if psql_pg < "$m.down.sql" >/dev/null 2>&1; then echo 'doble DOWN aceptado'; exit 1; fi
psql_pg < "$m.up.sql" >/dev/null
echo 'Bolsa 000040 UP/DOWN/UP y dobles aplicaciones: OK'
if psql_pg < "$ad386.down.sql" >/dev/null 2>&1; then echo 'DOWN de AD3-86 aceptado con Bolsa 000040 instalada'; exit 1; fi
# 3. ACL con roles reales.
psql_pg <<SQL
DO \$acl\$ BEGIN
 IF NOT has_function_privilege('vec_bolsa_llamamientos_ejecutor','$confirmar','EXECUTE') OR NOT has_function_privilege('vec_bolsa_llamamientos_ejecutor','$leer','EXECUTE') OR NOT has_function_privilege('vec_bolsa_llamamientos_ejecutor','$leer_confirmacion','EXECUTE') THEN RAISE EXCEPTION 'el ejecutor no puede usar las funciones públicas'; END IF;
 IF has_function_privilege('vec_bolsa_llamamientos_ejecutor','$interna','EXECUTE') OR has_function_privilege('vec_bolsa_llamamientos_ejecutor','$participacion','EXECUTE') THEN RAISE EXCEPTION 'el ejecutor alcanza el núcleo interno'; END IF;
 IF has_function_privilege('vec_bolsa_llamamientos_ejecutor','$fachada','EXECUTE') OR has_function_privilege('vec_bolsa_llamamientos_registrador_frontera','$fachada','EXECUTE') OR NOT has_function_privilege('vec_bolsa_llamamientos_propietario','$fachada','EXECUTE') THEN RAISE EXCEPTION 'ACL de la fachada AD3-86'; END IF;
 IF has_function_privilege('vec_bolsa_llamamientos_registrador_frontera','$confirmar','EXECUTE') THEN RAISE EXCEPTION 'el registrador de frontera puede confirmar'; END IF;
 IF EXISTS(SELECT 1 FROM pg_proc p, aclexplode(p.proacl) a WHERE p.oid IN ('$confirmar'::regprocedure,'$leer'::regprocedure,'$leer_confirmacion'::regprocedure,'$interna'::regprocedure,'$participacion'::regprocedure,'$fachada'::regprocedure) AND a.grantee=0) THEN RAISE EXCEPTION 'PUBLIC puede ejecutar'; END IF;
 IF has_table_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.confirmacion_contacto_participacion','SELECT,INSERT,UPDATE,DELETE') THEN RAISE EXCEPTION 'el ejecutor usa la tabla'; END IF;
END \$acl\$;
SQL
echo 'ACL: OK'
# 4. Comportamiento, con historia confirmada.
psql_pg < "$pruebas/datos.sql" >/dev/null
psql_pg < "$pruebas/pruebas.sql" 2>&1 | grep NOTICE | sed -E 's/^.*NOTICE: +/  /'
# 5. Concurrencia: dos claves para la misma versión vigente a la vez.
psql_pg >/dev/null <<'SQL'
BEGIN; SET LOCAL session_replication_role = replica;
INSERT INTO vec_bolsa_llamamientos.datos_contacto_participacion VALUES
 ('part:cp:2',1,'kms:prueba',decode(repeat('05',12),'hex'),decode(repeat('06',32),'hex'),'Importado de CONVOCA','per_actoractoractoractoractor',now()-interval '1 minute','convoca-b','recibo:contacto:cpb');
COMMIT;
SQL
lanzar() {
  { echo 'SET ROLE vec_bolsa_llamamientos_ejecutor;'; echo "BEGIN; SELECT * FROM prueba_cp.confirmar('can_contacto_propio_candidato_B1',1,'clave-concurrente-$1'); SELECT pg_sleep(2); COMMIT;"; } \
    | docker exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U postgres >/dev/null 2>"$trabajo/concurrencia_$1.err"
}
lanzar 1 & p1=$!; sleep 0.5; lanzar 2 & p2=$!
r1=0; r2=0; wait $p1 || r1=$?; wait $p2 || r2=$?
[[ $r1 -eq 0 && $r2 -ne 0 ]] && grep -q 'ya confirmado' "$trabajo/concurrencia_2.err" || { echo "concurrencia: $r1 $r2"; cat "$trabajo"/concurrencia_*.err; exit 1; }
echo 'Concurrencia: OK'
# 6. Reinicio: la historia persiste y el replay devuelve el mismo recibo.
consulta="SET ROLE vec_bolsa_llamamientos_ejecutor; SELECT recibo_ref||'|'||confirmada_en FROM prueba_cp.confirmar('can_contacto_propio_candidato_B1',1,'clave-concurrente-1');"
antes=$(echo "$consulta" | psql_pg -tA)
docker restart "$contenedor" >/dev/null
for _ in $(seq 1 60); do docker exec "$contenedor" pg_isready -U postgres >/dev/null 2>&1 && break; sleep 1; done
sleep 2
despues=$(echo "$consulta" | psql_pg -tA)
[[ -n $antes && $antes == "$despues" ]] || { echo "replay tras reinicio distinto: $antes / $despues"; exit 1; }
echo 'Reinicio: OK'
# 7. DOWN con historia rechazado en ambas.
if psql_pg < "$m.down.sql" >/dev/null 2>&1; then echo 'DOWN de 000040 con historia aceptado'; exit 1; fi
if psql_pg < "$ad386.down.sql" >/dev/null 2>&1; then echo 'DOWN de AD3-86 con historia aceptado'; exit 1; fi
echo 'DOWN con historia rechazado: OK'
printf 'PG18 AD3-86 y Bolsa 000040: preimagen, UP/DOWN/UP, ACL, fachada, confirmación, replay, negativos, concurrencia, reinicio y DOWN protegido OK\n'
