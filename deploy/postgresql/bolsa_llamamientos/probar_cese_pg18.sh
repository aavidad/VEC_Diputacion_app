#!/usr/bin/env bash
# Ensayo aislado de Bolsa 000045. El verificador CT es un doble cerrado por
# origen/huella/posición; la instalación conjunta con CT129 se verifica aparte.
set -Eeuo pipefail
repo=$(git -C "$(dirname -- "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm}
contenedor=vec-pg-cese-bolsa-$$
datos=/dev/shm/vec-pg-cese-bolsa-$$
limpiar() {
 docker rm -f "$contenedor" >/dev/null 2>&1 || true
 if [[ -d $datos ]]; then docker run --rm -v "$datos":/d --entrypoint /bin/sh "$imagen" -c 'rm -rf /d/* /d/.[!.]*' >/dev/null 2>&1 || true; fi
 rmdir "$datos" 2>/dev/null || true
}
trap limpiar EXIT
mkdir -p "$datos"
docker run --detach --rm --network none --name "$contenedor" -v "$datos":/var/lib/postgresql -v "$repo":/repo:ro \
 -e POSTGRES_HOST_AUTH_METHOD=trust "$imagen" >/dev/null
for _ in $(seq 1 60); do docker exec "$contenedor" pg_isready -q -U postgres 2>/dev/null && break; sleep 0.5; done
sleep 2
psql_pg() { docker exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U postgres "$@"; }
fichero() { psql_pg -f "/repo/$1"; }
for ruta in deploy/postgresql/autorizacion/roles_up.sql deploy/postgresql/autorizacion/migraciones/000001_autorizacion.up.sql \
 deploy/postgresql/ejecucion_documental_v4/migraciones_autorizacion/000002_vinculo_autenticacion_actor_actual.up.sql \
 deploy/postgresql/bolsa_llamamientos/roles_up.sql deploy/postgresql/bolsa_llamamientos/migraciones_autorizacion/000001_revalidacion_llamamientos.up.sql; do
 fichero "$ruta" >/dev/null
done
fichero deploy/postgresql/bolsa_llamamientos/pruebas_sql/contacto_origen/dobles_ad3.sql >/dev/null 2>&1
psql_pg >/dev/null <<'SQL'
DO $f$ DECLARE n text; BEGIN
 FOREACH n IN ARRAY ARRAY['registrar_y_consumir_situacion_participacion_v3_atestada','registrar_y_consumir_portal_candidato_bolsa_v3_atestada','registrar_y_consumir_emision_llamamiento_v3_atestada'] LOOP
  EXECUTE format('DROP FUNCTION IF EXISTS vec_autorizacion_atestada_v3.%I(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)', n);
  EXECUTE format($s$CREATE FUNCTION vec_autorizacion_atestada_v3.%I(p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
   RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
   LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $q$SELECT 'decision:doble', convert_from(p_decision,'UTF8')::jsonb->>'recurso_ref', convert_from(p_decision,'UTF8')::jsonb->>'contexto_recurso_huella_sha256', repeat('c',64), 'auditoria:doble', now(), true$q$$s$, n);
  EXECUTE format('ALTER FUNCTION vec_autorizacion_atestada_v3.%I(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) OWNER TO vec_autorizacion_atestada_v3_propietario', n);
 END LOOP;
END $f$;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_bolsa_llamamientos_propietario;
SQL
m=deploy/postgresql/bolsa_llamamientos/migraciones
for f in "$repo"/$m/*.up.sql; do
 n=$(basename "$f")
 [[ $n > 000031_~ ]] && break
 fichero "$m/$n" >/dev/null 2>&1 || { fichero "$m/$n" >&2; exit 1; }
 if [[ $n == 000010_* ]]; then fichero deploy/postgresql/bolsa_llamamientos/roles_registrador_frontera_up.sql >/dev/null; fi
done
for n in 000032_politica_transiciones_situacion 000033_politica_segregacion 000034_traza_valores_participacion 000035_origen_datos_contacto 000037_efectos_sanciones_participacion; do
 fichero "$m/$n.up.sql" >/dev/null
done
# ROLLBACK, UP/DOWN/UP y doble UP. DOWN con historia se prueba después.
sed 's/^COMMIT;$/ROLLBACK;/' "$repo/$m/000045_restriccion_global_cese.up.sql" | psql_pg >/dev/null
[[ $(psql_pg -tAc "SELECT to_regclass('vec_bolsa_llamamientos.restriccion_cese_bolsa') IS NULL") == t ]]
fichero "$m/000045_restriccion_global_cese.up.sql" >/dev/null
if fichero "$m/000045_restriccion_global_cese.up.sql" >/dev/null 2>&1; then echo 'doble UP aceptado' >&2; exit 1; fi
fichero "$m/000045_restriccion_global_cese.down.sql" >/dev/null
fichero "$m/000045_restriccion_global_cese.up.sql" >/dev/null
# La firma del doble es exactamente la de CT129. Solo responde a una fila
# publicada sintética con triple origen/huella/posición coincidente.
psql_pg >/dev/null <<'SQL'
CREATE SCHEMA vec_contratacion_temporal;
CREATE TABLE vec_contratacion_temporal.cese_prueba(origen_ref text PRIMARY KEY,huella text NOT NULL,posicion bigint NOT NULL,
 modalidad text NOT NULL,causa text,fecha date NOT NULL,llamamiento text NOT NULL,relacion text NOT NULL);
CREATE FUNCTION vec_contratacion_temporal.verificar_cese_publicado_bolsa_v1(p_origen_ref text,p_huella_sha256 text,p_posicion bigint)
RETURNS TABLE(modalidad_clave text,causa_contrato_clave text,causa_cese_clave text,fecha_efecto date,fuente_tipo text,
 fuente_ref text,fuente_sha256 text,relacion_ref text,version_resultante numeric,recibo_ref text,
 incorporacion_ref text,llamamiento_ref text,organizacion_ref text,expediente_ref text)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT c.modalidad,c.causa,'fin_sustitucion',c.fecha,'justificante','fuente:cese',repeat('f',64),c.relacion,9,
  'recibo:ct:cese','incorporacion:ct:1',c.llamamiento,'organizacion:desarrollo:dipgra','expediente:ct:1'
 FROM vec_contratacion_temporal.cese_prueba c
 WHERE c.origen_ref=p_origen_ref AND c.huella=p_huella_sha256 AND c.posicion=p_posicion
$f$;
GRANT USAGE ON SCHEMA vec_contratacion_temporal TO vec_bolsa_llamamientos_propietario;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.verificar_cese_publicado_bolsa_v1(text,text,bigint) TO vec_bolsa_llamamientos_propietario;
CREATE FUNCTION public.huella_cese_b45(p_origen_ref text) RETURNS text LANGUAGE sql STABLE SECURITY DEFINER
 SET search_path=pg_catalog AS $f$ SELECT c.huella FROM vec_contratacion_temporal.cese_prueba c
 WHERE c.origen_ref=p_origen_ref $f$;
REVOKE ALL ON FUNCTION public.huella_cese_b45(text) FROM PUBLIC;
SQL
psql_pg >/dev/null <<'SQL'
CREATE ROLE vec_b45_relevo_test LOGIN INHERIT;
CREATE ROLE vec_b45_ejecutor_test LOGIN INHERIT;
GRANT vec_bolsa_llamamientos_relevo_cese TO vec_b45_relevo_test;
GRANT vec_bolsa_llamamientos_ejecutor TO vec_b45_ejecutor_test;
GRANT USAGE ON SCHEMA public TO vec_b45_relevo_test;
GRANT EXECUTE ON FUNCTION public.huella_cese_b45(text) TO vec_b45_relevo_test;
SQL
fichero deploy/postgresql/bolsa_llamamientos/pruebas_sql/revision/datos.sql >/dev/null
fichero deploy/postgresql/bolsa_llamamientos/pruebas_sql/b45_restriccion_global_cese.sql >/dev/null
if fichero "$m/000045_restriccion_global_cese.down.sql" >/dev/null 2>&1; then echo 'DOWN con historia aceptado' >&2; exit 1; fi
echo 'Bolsa 000045: PostgreSQL 18, origen, +5/+9, orden global, replay, ACL y DOWN protegido OK'
