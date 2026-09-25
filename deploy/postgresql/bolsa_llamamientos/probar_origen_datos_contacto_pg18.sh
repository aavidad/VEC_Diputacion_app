#!/usr/bin/env bash
# Ensayo de Bolsa 000035 (contacto de origen CONVOCA, duda 45) en PostgreSQL 18
# efímero. Instala la cadena real de Bolsa hasta 000023 con dobles de los
# consumidores AD3 (solo firmas y retorno: la autorización real se ensaya en su
# propio esquema) y comprueba ROLLBACK, UP/DOWN/UP, dobles aplicaciones, ACL con
# roles reales, alta y replay con origen, negativos, sustitución por una
# versión propia, concurrencia y DOWN protegido. Los datos viven en /dev/shm y
# se borran al terminar.
set -Eeuo pipefail
repo=$(git -C "$(dirname -- "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)
pruebas=$repo/deploy/postgresql/bolsa_llamamientos/pruebas_sql/contacto_origen
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm}
contenedor=vec-pg-origen-contacto-$$
datos=/dev/shm/vec-pg-origen-contacto-$$
trabajo=$(mktemp -d /dev/shm/vec-origen-contacto-XXXXXX)
limpiar() {
  docker rm -f "$contenedor" >/dev/null 2>&1 || true
  if [[ -d "$datos" ]]; then docker run --rm -v "$datos":/d --entrypoint rm "$imagen" -rf /d/18 >/dev/null 2>&1 || true; fi
  rm -rf "$datos" "$trabajo"
}
trap limpiar EXIT
mkdir -p "$datos"
clave=$(od -An -N32 -tx1 /dev/urandom | tr -d '[:space:]')
docker run --detach --rm --name "$contenedor" -v "$datos":/var/lib/postgresql -e POSTGRES_PASSWORD="$clave" "$imagen" >/dev/null
for _ in $(seq 1 60); do docker exec "$contenedor" pg_isready -U postgres >/dev/null 2>&1 && break; sleep 1; done
sleep 2
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
for f in "$repo"/deploy/postgresql/bolsa_llamamientos/migraciones/0000{01,02,03,04,05,06,07,08,10,11,12,13,14,16,17,18,19,20,21,22,23}_*.up.sql; do
  psql_pg < "$f" >/dev/null 2>"$trabajo/migracion.err" || { echo "falló $(basename "$f")" >&2; cat "$trabajo/migracion.err" >&2; exit 1; }
  if [[ $(basename "$f") == 000010_* ]]; then psql_pg < "$repo/deploy/postgresql/bolsa_llamamientos/roles_registrador_frontera_up.sql" >/dev/null; fi
done
m=$repo/deploy/postgresql/bolsa_llamamientos/migraciones/000035_origen_datos_contacto
alta='vec_bolsa_llamamientos.registrar_datos_contacto_origen_participacion_v1(text,text,bigint,text,bytea,bytea,text,text,timestamptz,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,timestamptz,date,text,text)'
lectura='vec_bolsa_llamamientos.leer_origen_datos_contacto_participacion_v1(text,bigint)'
# 1. Ensayo en ROLLBACK: no deja nada.
sed 's/^COMMIT;$/ROLLBACK;/' "$m.up.sql" | psql_pg >/dev/null
psql_pg -tAc "SELECT to_regprocedure('$alta') IS NULL AND to_regclass('vec_bolsa_llamamientos.origen_datos_contacto_participacion') IS NULL" | grep -qx t || { echo 'ROLLBACK dejó objetos'; exit 1; }
# 2. UP, doble UP rechazado, DOWN sin historia, doble DOWN rechazado, UP otra vez.
psql_pg < "$m.up.sql" >/dev/null
if psql_pg < "$m.up.sql" >/dev/null 2>&1; then echo 'doble UP aceptado'; exit 1; fi
psql_pg < "$m.down.sql" >/dev/null
psql_pg -tAc "SELECT to_regprocedure('$alta') IS NULL AND to_regprocedure('$lectura') IS NULL AND to_regclass('vec_bolsa_llamamientos.origen_datos_contacto_participacion') IS NULL" | grep -qx t || { echo 'DOWN incompleto'; exit 1; }
if psql_pg < "$m.down.sql" >/dev/null 2>&1; then echo 'doble DOWN aceptado'; exit 1; fi
psql_pg < "$m.up.sql" >/dev/null
echo 'UP/DOWN/UP y dobles aplicaciones: OK'
# 3. ACL con roles reales.
psql_pg <<SQL
DO \$acl\$ BEGIN
 IF NOT has_function_privilege('vec_bolsa_llamamientos_ejecutor','$alta','EXECUTE') OR NOT has_function_privilege('vec_bolsa_llamamientos_ejecutor','$lectura','EXECUTE') THEN RAISE EXCEPTION 'el ejecutor no puede usar las funciones'; END IF;
 IF has_function_privilege('vec_bolsa_llamamientos_registrador_frontera','$alta','EXECUTE') OR has_function_privilege('vec_bolsa_llamamientos_registrador_frontera','$lectura','EXECUTE') THEN RAISE EXCEPTION 'el registrador de frontera puede usar las funciones'; END IF;
 IF EXISTS(SELECT 1 FROM pg_proc p, aclexplode(p.proacl) a WHERE p.oid IN ('$alta'::regprocedure,'$lectura'::regprocedure) AND a.grantee=0) THEN RAISE EXCEPTION 'PUBLIC puede ejecutar'; END IF;
 IF has_table_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.origen_datos_contacto_participacion','SELECT,INSERT,UPDATE,DELETE') THEN RAISE EXCEPTION 'el ejecutor usa la tabla'; END IF;
 IF (SELECT NOT prosecdef OR proowner<>'vec_bolsa_llamamientos_propietario'::regrole OR proconfig<>ARRAY['search_path=pg_catalog','lock_timeout=2s'] FROM pg_proc WHERE oid='$alta'::regprocedure) THEN RAISE EXCEPTION 'alta sin definidor o configuración'; END IF;
 IF (SELECT NOT prosecdef OR proowner<>'vec_bolsa_llamamientos_propietario'::regrole FROM pg_proc WHERE oid='$lectura'::regprocedure) THEN RAISE EXCEPTION 'lectura sin definidor'; END IF;
END \$acl\$;
SQL
echo 'ACL: OK'
psql_pg < "$pruebas/datos.sql" >/dev/null
psql_pg < "$pruebas/funciones_prueba.sql" >/dev/null
# 4. Comportamiento como ejecutor.
{ echo 'SET ROLE vec_bolsa_llamamientos_ejecutor;'; cat <<'SQL'
DO $p$ DECLARE r record; o record; BEGIN
 -- Alta con origen CONVOCA: versión 1 y marca con la vigencia de la regla.
 SELECT * INTO STRICT r FROM prueba.origen('participacion:1',1,'origen-1','2026-09-28T08:00:00Z');
 IF r.reutilizada OR r.version<>1 OR r.recibo_ref<>'recibo:origen-1' THEN RAISE EXCEPTION 'alta: %',r; END IF;
 SELECT * INTO STRICT o FROM vec_bolsa_llamamientos.leer_origen_datos_contacto_participacion_v1('participacion:1',1);
 IF o.origen<>'convoca' OR o.vigente_hasta<>'2027-09-28T08:00:00Z' OR o.regla_ref<>'vec.bolsa.reglas:1:b29.contacto_origen_convoca' OR o.regla_huella_sha256<>repeat('d',64) THEN RAISE EXCEPTION 'marca: %',o; END IF;
 -- Replay exacto: mismo recibo, sin versión ni marca nuevas.
 SELECT * INTO STRICT r FROM prueba.origen('participacion:1',1,'origen-1','2026-09-28T08:00:00Z');
 IF NOT r.reutilizada OR r.version<>1 THEN RAISE EXCEPTION 'replay: %',r; END IF;
 -- La misma clave usada antes sin origen no se convierte en contacto CONVOCA.
 PERFORM prueba.propio('participacion:2',1,'propio-2','2026-09-28T08:00:00Z');
 PERFORM prueba.espera($$SELECT * FROM prueba.origen('participacion:2',1,'propio-2','2026-09-28T08:00:00Z')$$,'22023');
 -- Parámetros de origen no válidos: nada se registra.
 PERFORM prueba.espera($$SELECT * FROM prueba.origen('participacion:2',2,'x1','2026-09-28T09:00:00Z','persona')$$,'22023');
 PERFORM prueba.espera($$SELECT * FROM prueba.origen('participacion:2',2,'x2','2026-09-28T09:00:00Z',p_hasta=>'2026-09-28T09:00:00Z')$$,'22023');
 PERFORM prueba.espera($$SELECT * FROM prueba.origen('participacion:2',2,'x3','2026-09-28T09:00:00Z',p_dia=>'2026-09-27')$$,'22023');
 PERFORM prueba.espera($$SELECT * FROM prueba.origen('participacion:2',2,'x4','2026-09-28T09:00:00Z',p_regla=>' ')$$,'22023');
 PERFORM prueba.espera($$SELECT * FROM prueba.origen('participacion:2',2,'x5','2026-09-28T09:00:00Z',p_huella=>'XYZ')$$,'22023');
 -- La participación ajena y la versión no consecutiva siguen rechazándose en B4.
 PERFORM prueba.espera($$SELECT * FROM prueba.origen('participacion:ajena',1,'x6','2026-09-28T09:00:00Z')$$,'23503');
 PERFORM prueba.espera($$SELECT * FROM prueba.origen('participacion:2',5,'x7','2026-09-28T09:00:00Z')$$,'VBS02');
 -- Una concesión ya consumida no registra nada.
 PERFORM prueba.espera($$SELECT * FROM prueba.origen('participacion:2',2,'x8','2026-09-28T09:00:00Z',p_capacidad=>'repetida')$$,'42501');
 -- La persona confirma (versión propia): la versión 2 no lleva marca.
 SELECT * INTO STRICT r FROM prueba.propio('participacion:1',2,'confirmado-1','2026-10-01T08:00:00Z');
 IF r.version<>2 OR EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.leer_origen_datos_contacto_participacion_v1('participacion:1',2)) THEN RAISE EXCEPTION 'confirmación: %',r; END IF;
 -- El ejecutor no lee ni escribe la tabla directamente.
 PERFORM prueba.espera($$SELECT count(*) FROM vec_bolsa_llamamientos.origen_datos_contacto_participacion$$,'42501');
 PERFORM prueba.espera($$INSERT INTO vec_bolsa_llamamientos.origen_datos_contacto_participacion VALUES('participacion:2',1,'convoca','2027-01-01','2026-12-31','r',repeat('d',64),'2026-01-01')$$,'42501');
END $p$;
SQL
} | psql_pg >/dev/null
# Historia de solo adición: la marca no se modifica ni se borra.
psql_pg <<'SQL' >/dev/null
SET ROLE vec_bolsa_llamamientos_propietario;
DO $i$ BEGIN
 BEGIN UPDATE vec_bolsa_llamamientos.origen_datos_contacto_participacion SET regla_ref='x'; RAISE EXCEPTION 'la marca debe ser inmutable'; EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL; END;
 BEGIN DELETE FROM vec_bolsa_llamamientos.origen_datos_contacto_participacion; RAISE EXCEPTION 'la marca debe ser inmutable'; EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL; END;
END $i$;
SQL
psql_pg -tAc "SELECT (SELECT count(*) FROM vec_bolsa_llamamientos.origen_datos_contacto_participacion)||'/'||(SELECT count(*) FROM vec_bolsa_llamamientos.datos_contacto_participacion)" | grep -qx '1/3' || { echo 'recuento inesperado'; exit 1; }
echo 'Comportamiento: OK'
# 5. Concurrencia: dos altas con origen a la vez para la misma versión.
lanzar() {
  { echo 'SET ROLE vec_bolsa_llamamientos_ejecutor;'; echo "BEGIN; SELECT * FROM prueba.origen('participacion:2',2,'concurrente-$1','2026-10-02T08:00:00Z'); SELECT pg_sleep(2); COMMIT;"; } | docker exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U postgres >/dev/null 2>"$trabajo/concurrencia_$1.err"
}
lanzar 1 & p1=$!; sleep 0.5; lanzar 2 & p2=$!
r1=0; r2=0; wait $p1 || r1=$?; wait $p2 || r2=$?
[[ $r1 -eq 0 && $r2 -ne 0 ]] && grep -q 'no consecutiva' "$trabajo/concurrencia_2.err" || { echo "concurrencia: $r1 $r2"; cat "$trabajo"/concurrencia_*.err; exit 1; }
psql_pg -tAc "SELECT count(*) FROM vec_bolsa_llamamientos.origen_datos_contacto_participacion WHERE participacion_ref='participacion:2'" | grep -qx 1 || { echo 'concurrencia duplicó la marca'; exit 1; }
echo 'Concurrencia: OK'
# 6. DOWN con historia rechazado.
if psql_pg < "$m.down.sql" >/dev/null 2>&1; then echo 'DOWN con historia aceptado'; exit 1; fi
echo 'DOWN con historia rechazado: OK'
printf 'PG18 Bolsa 000035: ensayo, UP/DOWN/UP, ACL, origen, replay, negativos, concurrencia y DOWN protegido OK\n'
