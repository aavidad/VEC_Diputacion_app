#!/usr/bin/env bash
# Ensayo de Bolsa 000031 (intentos telefónicos y rebote de correo) en
# PostgreSQL 18 efímero. Instala la cadena real de Bolsa hasta 000023 con
# dobles de los consumidores AD3 (solo firmas y retorno: la autorización real
# se ensaya en su propio esquema) y comprueba ROLLBACK, UP/DOWN/UP, dobles
# aplicaciones, ACL con roles reales, reglas, negativos, concurrencia y DOWN
# protegido. Los datos viven en /dev/shm y se borran al terminar.
set -Eeuo pipefail
repo=$(git -C "$(dirname -- "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)
pruebas=$repo/deploy/postgresql/bolsa_llamamientos/pruebas_sql/intentos_contacto
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm}
contenedor=vec-pg-intentos-$$
datos=/dev/shm/vec-pg-intentos-$$
trabajo=$(mktemp -d /dev/shm/vec-intentos-XXXXXX)
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
  psql_pg < "$f" >/dev/null 2>&1 || { echo "falló $(basename "$f")" >&2; exit 1; }
  if [[ $(basename "$f") == 000010_* ]]; then psql_pg < "$repo/deploy/postgresql/bolsa_llamamientos/roles_registrador_frontera_up.sql" >/dev/null; fi
done
m=$repo/deploy/postgresql/bolsa_llamamientos/migraciones/000031_intentos_contacto_llamamiento
v2='vec_bolsa_llamamientos.registrar_contacto_participacion_v2(text,text,text,text,text,timestamptz,text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,integer,integer,boolean,text[])'
# 1. Ensayo en ROLLBACK: no deja nada.
sed 's/^COMMIT;$/ROLLBACK;/' "$m.up.sql" | psql_pg >/dev/null
psql_pg -tAc "SELECT to_regprocedure('$v2') IS NULL" | grep -qx t || { echo 'ROLLBACK dejó la v2'; exit 1; }
# 2. UP, doble UP rechazado, DOWN sin historia, UP otra vez.
psql_pg < "$m.up.sql" >/dev/null
if psql_pg < "$m.up.sql" >/dev/null 2>&1; then echo 'doble UP aceptado'; exit 1; fi
psql_pg < "$m.down.sql" >/dev/null
psql_pg -tAc "SELECT to_regprocedure('$v2') IS NULL AND strpos(pg_get_constraintdef(oid),'numero_erroneo')=0 FROM pg_constraint WHERE conname='contacto_participacion_resultado_check'" | grep -qx t || { echo 'DOWN incompleto'; exit 1; }
if psql_pg < "$m.down.sql" >/dev/null 2>&1; then echo 'doble DOWN aceptado'; exit 1; fi
psql_pg < "$m.up.sql" >/dev/null
echo 'UP/DOWN/UP y dobles aplicaciones: OK'
# 3. ACL con roles reales.
psql_pg <<SQL
DO \$acl\$ BEGIN
 IF NOT has_function_privilege('vec_bolsa_llamamientos_ejecutor','$v2','EXECUTE') THEN RAISE EXCEPTION 'el ejecutor no puede registrar'; END IF;
 IF has_function_privilege('vec_bolsa_llamamientos_registrador_frontera','$v2','EXECUTE') THEN RAISE EXCEPTION 'el registrador de frontera puede registrar'; END IF;
 IF EXISTS(SELECT 1 FROM pg_proc p, aclexplode(p.proacl) a WHERE p.oid='$v2'::regprocedure AND a.grantee=0) THEN RAISE EXCEPTION 'PUBLIC puede ejecutar la v2'; END IF;
 IF has_table_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.contacto_participacion','SELECT,INSERT,UPDATE,DELETE') THEN RAISE EXCEPTION 'el ejecutor lee la tabla'; END IF;
 IF (SELECT NOT prosecdef OR proowner<>'vec_bolsa_llamamientos_propietario'::regrole OR proconfig<>ARRAY['search_path=pg_catalog','lock_timeout=2s'] FROM pg_proc WHERE oid='$v2'::regprocedure) THEN RAISE EXCEPTION 'v2 sin definidor o configuración'; END IF;
END \$acl\$;
SQL
echo 'ACL: OK'
psql_pg < "$pruebas/datos.sql" >/dev/null
# 4. Comportamiento como ejecutor.
{ cat "$pruebas/funciones_prueba.sql"; echo 'SET ROLE vec_bolsa_llamamientos_ejecutor;'; cat <<'SQL'
DO $p$ DECLARE r record; sin constant text[]:=ARRAY['no_contesta','buzon','numero_erroneo']; BEGIN
 -- Rebote del correo sin control.
 SELECT * INTO STRICT r FROM prueba.v2('contacto:rebote','correo','2026-09-28T07:00:00Z','no_entregado','rebote',NULL,NULL,NULL,NULL);
 IF r.reutilizado OR r.previos_sin_contacto<>0 THEN RAISE EXCEPTION 'rebote: %',r; END IF;
 -- Primer intento.
 SELECT * INTO STRICT r FROM prueba.v2('contacto:i1','telefono','2026-09-28T08:00:00Z','no_contesta','i1',4,7200,true,sin);
 IF r.reutilizado OR r.previos_sin_contacto<>0 OR r.previo_ultimo IS NOT NULL THEN RAISE EXCEPTION 'i1: %',r; END IF;
 -- Segundo antes de 2 h: impedido; en modo advertir sí se registra.
 PERFORM prueba.espera($$SELECT * FROM prueba.v2('contacto:i2x','telefono','2026-09-28T09:59:59Z','no_contesta','i2x',4,7200,true,ARRAY['no_contesta','buzon','numero_erroneo'])$$,'VBC02');
 PERFORM prueba.espera($$SELECT * FROM prueba.v2('contacto:i2y','telefono','2026-09-28T07:00:01Z','no_contesta','i2y',4,7200,true,ARRAY['no_contesta','buzon','numero_erroneo'])$$,'VBC02');
 SELECT * INTO STRICT r FROM prueba.v2('contacto:i2','telefono','2026-09-28T09:00:00Z','numero_erroneo','i2',4,7200,false,sin);
 IF r.previos_sin_contacto<>1 OR r.previo_ultimo<>'2026-09-28T08:00:00Z' OR r.previo_contactado THEN RAISE EXCEPTION 'i2: %',r; END IF;
 SELECT * INTO STRICT r FROM prueba.v2('contacto:i3','telefono','2026-09-29T08:00:00Z','buzon','i3',4,7200,true,sin);
 SELECT * INTO STRICT r FROM prueba.v2('contacto:i4','telefono','2026-09-29T10:00:00Z','no_contesta','i4',4,7200,true,sin);
 IF r.previos_sin_contacto<>3 THEN RAISE EXCEPTION 'i4: %',r; END IF;
 -- Agotados: ni otro intento ni un contacto tardío por teléfono.
 PERFORM prueba.espera($$SELECT * FROM prueba.v2('contacto:i5','telefono','2026-09-30T08:00:00Z','contactado','i5',4,7200,true,ARRAY['no_contesta','buzon','numero_erroneo'])$$,'VBC03');
 -- La repetición exacta del cuarto intento devuelve su recibo.
 SELECT * INTO STRICT r FROM prueba.v2('contacto:i4','telefono','2026-09-29T10:00:00Z','no_contesta','i4',4,7200,true,sin);
 IF NOT r.reutilizado OR r.recibo_ref<>'recibo:contacto:i4' OR r.previos_sin_contacto<>3 THEN RAISE EXCEPTION 'replay: %',r; END IF;
 PERFORM prueba.espera($$SELECT * FROM prueba.v2('contacto:i4','telefono','2026-09-29T10:00:00Z','buzon','i4',4,7200,true,ARRAY['no_contesta','buzon','numero_erroneo'])$$,'VBC01');
 -- Parámetros de control incoherentes.
 PERFORM prueba.espera($$SELECT * FROM prueba.v2('contacto:x1','correo','2026-09-30T08:00:00Z','no_entregado','x1',4,7200,true,ARRAY['no_contesta'])$$,'22023');
 PERFORM prueba.espera($$SELECT * FROM prueba.v2('contacto:x2','telefono','2026-09-30T08:00:00Z','no_contesta','x2',4,NULL,true,ARRAY['no_contesta'])$$,'22023');
 PERFORM prueba.espera($$SELECT * FROM prueba.v2('contacto:x3','telefono','2026-09-30T08:00:00Z','no_contesta','x3',4,7200,true,ARRAY['colgado'])$$,'22023');
 PERFORM prueba.espera($$SELECT * FROM prueba.v2('contacto:x4','telefono','2026-09-30T08:00:00Z','no_contesta','x4',NULL,7200,NULL,NULL)$$,'22023');
 PERFORM prueba.espera($$SELECT * FROM prueba.v2('contacto:x5','telefono','2026-09-30T08:00:00Z','enviado','x5',NULL,NULL,NULL,NULL)$$,'22023');
 PERFORM prueba.espera($$SELECT * FROM prueba.v2('contacto:x6','telefono','2026-09-30T08:00:00Z','no_contesta','x6',4,7200,true,ARRAY['no_contesta'],'Llamada al dni 1')$$,'23514');
 -- La v1 no cambia: no admite los resultados nuevos.
 PERFORM prueba.espera($$SELECT * FROM vec_bolsa_llamamientos.registrar_contacto_participacion_v1('contacto:v1','bolsa:1','participacion:1',NULL,'telefono','2026-09-30T08:00:00Z','per_aaaaaaaaaaaaaaaaaaaaaa','numero_erroneo','x','v1','recibo:v1','\x00','\x00','\x00','\x00',1,1,'\x00','\x00','\x00','\x00')$$,'22023');
 -- El ejecutor no lee la tabla directamente.
 PERFORM prueba.espera($$SELECT count(*) FROM vec_bolsa_llamamientos.contacto_participacion$$,'42501');
END $p$;
SQL
} | psql_pg >/dev/null
psql_pg -tAc "SELECT count(*) FILTER (WHERE canal='telefono')||'/'||count(*) FROM vec_bolsa_llamamientos.contacto_participacion" | grep -qx '4/5' || { echo 'recuento inesperado'; exit 1; }
echo 'Comportamiento: OK'
# Una decisión ilegible tras un consumo válido es 42501, no 22P02. El doble
# del consumidor se sustituye solo dentro de esta transacción.
psql_pg >/dev/null <<'SQL'
BEGIN;
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_contacto_participacion_v3_atestada(p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
 RETURNS TABLE(efecto_ref text,consumo_nuevo boolean) LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $$ SELECT 'participacion:1', true $$;
SET LOCAL ROLE vec_bolsa_llamamientos_ejecutor;
SELECT prueba.espera($$SELECT * FROM vec_bolsa_llamamientos.registrar_contacto_participacion_v2('contacto:ilegible','bolsa:1','participacion:1',NULL,'telefono','2026-09-30T08:00:00Z','per_aaaaaaaaaaaaaaaaaaaaaa','contactado','Llamada de prueba','ilegible','recibo:ilegible','\x00','\xff','\x00','\x00',1,1,'\x00','\x00','\x00','\x00',NULL,NULL,NULL,NULL)$$,'42501');
ROLLBACK;
SQL
echo 'Decisión ilegible: OK'
# 5. Concurrencia: dos intentos a la vez sobre el mismo llamamiento.
psql_pg < "$pruebas/datos_concurrencia.sql" >/dev/null
lanzar() {
  { echo 'SET ROLE vec_bolsa_llamamientos_ejecutor;'; echo "BEGIN; SELECT * FROM prueba.v2b('contacto:c$1','telefono','2026-10-01T08:0$1:00Z','no_contesta','c$1',4,7200,true,ARRAY['no_contesta']); SELECT pg_sleep(2); COMMIT;"; } | docker exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U postgres >/dev/null 2>"$trabajo/concurrencia_$1.err"
}
lanzar 1 & p1=$!; sleep 0.5; lanzar 2 & p2=$!
r1=0; r2=0; wait $p1 || r1=$?; wait $p2 || r2=$?
[[ $r1 -eq 0 && $r2 -ne 0 ]] && grep -q 'separación' "$trabajo/concurrencia_2.err" || { echo "concurrencia: $r1 $r2"; cat "$trabajo"/concurrencia_*.err; exit 1; }
echo 'Concurrencia: OK'
# 6. DOWN con historia rechazado.
if psql_pg < "$m.down.sql" >/dev/null 2>&1; then echo 'DOWN con historia aceptado'; exit 1; fi
echo 'DOWN con historia rechazado: OK'
printf 'PG18 Bolsa 000031: ensayo, UP/DOWN/UP, ACL, reglas, concurrencia y DOWN protegido OK\n'
