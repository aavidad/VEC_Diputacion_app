\set ON_ERROR_STOP on
-- Ejecutar tras Bolsa 000045 y CT129 en una base sintética desechable con un
-- cese CT115 de llamamiento. Crea y retira roles de prueba; no toca negocio.
CREATE FUNCTION pg_temp.exigir_ct129(p_ok boolean,p_caso text)
RETURNS void LANGUAGE plpgsql AS $$
BEGIN
    IF p_ok IS DISTINCT FROM true THEN
        RAISE EXCEPTION 'CT129: %',p_caso;
    END IF;
END
$$;
CREATE FUNCTION pg_temp.exigir_feed_denegado_ct129()
RETURNS void LANGUAGE plpgsql AS $$
BEGIN
    PERFORM 1 FROM vec_contratacion_temporal.leer_ceses_bolsa_v1(NULL,NULL,10);
    RAISE EXCEPTION 'CT129: feed accesible con roles CT/Bolsa combinados';
EXCEPTION WHEN SQLSTATE '42501' THEN
    RETURN;
END
$$;

SELECT c.evento_ref AS origen, c.justificante_tipo AS tipo, c.justificante_ref AS fuente,
       c.justificante_sha256 AS fuente_sha, c.fecha_efecto::text AS fecha,
       c.recibo_ref AS recibo, c.incorporacion_ref AS incorporacion,
       c.llamamiento_ref AS llamamiento, c.organizacion_ref AS organizacion,
       c.expediente_ref AS expediente, (c.version_esperada+1)::text AS version,
       c.causa_clave AS causa_cese,
       c.expediente_siguiente_json #>> '{analisis,modalidad_clave}' AS modalidad,
       c.expediente_siguiente_json #>> '{analisis,causa_clave}' AS causa_contrato,
       i.material_json #>> '{Confirmacion,ResultadoPersonal,relacion_ref}' AS relacion
  FROM vec_contratacion_temporal.cese_nombramiento_v1 c
  JOIN vec_contratacion_temporal.incorporacion_registro_v2 i ON i.recibo_ref=c.incorporacion_ref
 WHERE c.llamamiento_ref IS NOT NULL
 ORDER BY c.transaccion_publicacion,c.evento_ref LIMIT 1 \gset

SET SESSION AUTHORIZATION vec_ct115_runtime;
BEGIN ISOLATION LEVEL REPEATABLE READ READ ONLY;
SELECT huella_sha256 AS huella, origen_posicion AS posicion
  FROM vec_contratacion_temporal.leer_ceses_bolsa_v1(NULL,NULL,100)
 WHERE origen_ref=:'origen' AND evento->>'tipo'='cese' \gset
SELECT pg_temp.exigir_ct129(
    (SELECT bool_and(evento->>'tipo'='cese') FROM vec_contratacion_temporal.leer_ceses_bolsa_v1(NULL,NULL,100))
    AND NOT EXISTS (SELECT 1 FROM vec_contratacion_temporal.leer_ceses_bolsa_v1(:'posicion'::bigint,:'origen',100)
                    WHERE origen_ref=:'origen')
    AND coalesce(current_setting('vec.ct115.publicacion_bolsa',true),'')='',
    'feed solo de ceses, cursor exclusivo y marca RLS restaurada');
COMMIT;
RESET SESSION AUTHORIZATION;

SELECT pg_temp.exigir_ct129(
    NOT has_table_privilege('vec_bolsa_llamamientos_propietario','vec_contratacion_temporal.cese_nombramiento_v1','SELECT')
    AND NOT has_table_privilege('vec_bolsa_llamamientos_propietario','vec_contratacion_temporal.incorporacion_registro_v2','SELECT')
    AND NOT has_table_privilege('vec_contratacion_temporal_ejecutor','vec_contratacion_temporal.cese_nombramiento_v1','SELECT')
    AND has_function_privilege('vec_contratacion_temporal_ejecutor','vec_contratacion_temporal.leer_ceses_bolsa_v1(bigint,text,integer)','EXECUTE')
    AND NOT has_function_privilege('vec_bolsa_llamamientos_propietario','vec_contratacion_temporal.leer_ceses_bolsa_v1(bigint,text,integer)','EXECUTE')
    AND NOT has_function_privilege('public','vec_contratacion_temporal.leer_ceses_bolsa_v1(bigint,text,integer)','EXECUTE'),
    'Bolsa y ejecutor CT sin tablas; feed solo para ejecutor CT');
SELECT pg_temp.exigir_ct129(to_regrole('vec_bolsa_llamamientos_relevo_cese') IS NOT NULL,
    'Bolsa 000045 requerida para el relevo nominal');
-- Cada LOGIN conserva el EXECUTE de CT, pero combina un rol Bolsa incompatible.
-- La denegación debe proceder de la guarda de la función, no de su ACL.
CREATE ROLE vec_ct129_feed_prop_prueba LOGIN INHERIT;
GRANT vec_contratacion_temporal_ejecutor,vec_bolsa_llamamientos_propietario TO vec_ct129_feed_prop_prueba;
CREATE ROLE vec_ct129_feed_mig_prueba LOGIN INHERIT;
GRANT vec_contratacion_temporal_ejecutor,vec_bolsa_llamamientos_migrador TO vec_ct129_feed_mig_prueba;
CREATE ROLE vec_ct129_feed_eje_prueba LOGIN INHERIT;
GRANT vec_contratacion_temporal_ejecutor,vec_bolsa_llamamientos_ejecutor TO vec_ct129_feed_eje_prueba;
SET SESSION AUTHORIZATION vec_ct129_feed_prop_prueba;
SELECT pg_temp.exigir_ct129(has_function_privilege(current_user,'vec_contratacion_temporal.leer_ceses_bolsa_v1(bigint,text,integer)','EXECUTE'),
    'LOGIN combinado propietario conserva EXECUTE CT');
SELECT pg_temp.exigir_feed_denegado_ct129();
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION vec_ct129_feed_mig_prueba;
SELECT pg_temp.exigir_ct129(has_function_privilege(current_user,'vec_contratacion_temporal.leer_ceses_bolsa_v1(bigint,text,integer)','EXECUTE'),
    'LOGIN combinado migrador conserva EXECUTE CT');
SELECT pg_temp.exigir_feed_denegado_ct129();
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION vec_ct129_feed_eje_prueba;
SELECT pg_temp.exigir_ct129(has_function_privilege(current_user,'vec_contratacion_temporal.leer_ceses_bolsa_v1(bigint,text,integer)','EXECUTE'),
    'LOGIN combinado ejecutor conserva EXECUTE CT');
SELECT pg_temp.exigir_feed_denegado_ct129();
RESET SESSION AUTHORIZATION;
CREATE ROLE vec_ct129_relevo_prueba LOGIN INHERIT;
GRANT vec_bolsa_llamamientos_relevo_cese TO vec_ct129_relevo_prueba;
CREATE ROLE vec_ct129_bolsa_ajena_prueba LOGIN INHERIT;
GRANT vec_bolsa_llamamientos_ejecutor TO vec_ct129_bolsa_ajena_prueba;
CREATE ROLE vec_ct129_migrador_prueba LOGIN NOINHERIT;
GRANT vec_contratacion_temporal_migrador TO vec_ct129_migrador_prueba WITH INHERIT FALSE, SET TRUE;
GRANT vec_bolsa_llamamientos_relevo_cese TO vec_ct129_migrador_prueba;

-- La envoltura de prueba reproduce solo la cadena de roles del consumidor:
-- LOGIN nominal -> función definidora Bolsa -> función definidora CT.
CREATE SCHEMA prueba_ct129 AUTHORIZATION vec_bolsa_llamamientos_propietario;
SET ROLE vec_bolsa_llamamientos_propietario;
CREATE FUNCTION prueba_ct129.verificar(p_origen text,p_huella text,p_posicion bigint)
RETURNS jsonb LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $$
    SELECT to_jsonb(v) FROM vec_contratacion_temporal.verificar_cese_publicado_bolsa_v1(p_origen,p_huella,p_posicion) v
$$;
REVOKE ALL ON FUNCTION prueba_ct129.verificar(text,text,bigint) FROM PUBLIC;
GRANT USAGE ON SCHEMA prueba_ct129 TO vec_bolsa_llamamientos_relevo_cese,vec_bolsa_llamamientos_ejecutor;
GRANT EXECUTE ON FUNCTION prueba_ct129.verificar(text,text,bigint)
    TO vec_bolsa_llamamientos_relevo_cese,vec_bolsa_llamamientos_ejecutor;
RESET ROLE;

SET SESSION AUTHORIZATION vec_ct129_relevo_prueba;
SELECT prueba_ct129.verificar(:'origen',:'huella',:'posicion'::bigint)::text AS verificado \gset
SELECT pg_temp.exigir_ct129(
    :'verificado'::jsonb->>'modalidad_clave'=:'modalidad'
    AND :'verificado'::jsonb->>'causa_contrato_clave'=:'causa_contrato'
    AND :'verificado'::jsonb->>'causa_cese_clave'=:'causa_cese'
    AND :'verificado'::jsonb->>'fecha_efecto'=:'fecha'
    AND :'verificado'::jsonb->>'fuente_tipo'=:'tipo'
    AND :'verificado'::jsonb->>'fuente_ref'=:'fuente'
    AND :'verificado'::jsonb->>'fuente_sha256'=:'fuente_sha'
    AND :'verificado'::jsonb->>'relacion_ref'=:'relacion'
    AND :'verificado'::jsonb->>'version_resultante'=:'version'
    AND :'verificado'::jsonb->>'recibo_ref'=:'recibo'
    AND :'verificado'::jsonb->>'incorporacion_ref'=:'incorporacion'
    AND :'verificado'::jsonb->>'llamamiento_ref'=:'llamamiento'
    AND :'verificado'::jsonb->>'organizacion_ref'=:'organizacion'
    AND :'verificado'::jsonb->>'expediente_ref'=:'expediente',
    'hecho acreditado y fuente exactos');
SELECT pg_temp.exigir_ct129(
    prueba_ct129.verificar(:'origen',repeat('0',64),:'posicion'::bigint) IS NULL
    AND prueba_ct129.verificar(:'origen',:'huella',:'posicion'::bigint+1) IS NULL
    AND prueba_ct129.verificar('evento:ct:inventado',:'huella',:'posicion'::bigint) IS NULL,
    'origen, huella y posición forjados denegados');
SELECT pg_temp.exigir_ct129(
    coalesce(current_setting('vec.ct129.origen_ref',true),'')='',
    'marca RLS restaurada');
RESET SESSION AUTHORIZATION;

-- Un LOGIN de Bolsa sin el grupo exclusivo recibe cero datos aun si se le
-- concediese por error acceso a la envoltura definidora.
SET SESSION AUTHORIZATION vec_ct129_bolsa_ajena_prueba;
SELECT pg_temp.exigir_ct129(prueba_ct129.verificar(:'origen',:'huella',:'posicion'::bigint) IS NULL,
    'otro runtime Bolsa denegado');
RESET SESSION AUTHORIZATION;

-- El migrador CT puede asumir SET ROLE propietario y fijar la GUC. La
-- política RLS sigue ocultando el cese; la función también devuelve cero.
GRANT USAGE ON SCHEMA prueba_ct129 TO vec_ct129_migrador_prueba;
GRANT EXECUTE ON FUNCTION prueba_ct129.verificar(text,text,bigint) TO vec_ct129_migrador_prueba;
SET SESSION AUTHORIZATION vec_ct129_migrador_prueba;
SELECT pg_temp.exigir_ct129(prueba_ct129.verificar(:'origen',:'huella',:'posicion'::bigint) IS NULL,
    'migrador CT sin consumo de evento');
SET ROLE vec_contratacion_temporal_propietario;
BEGIN;
SELECT set_config('vec.ct129.origen_ref',:'origen',true);
SELECT pg_temp.exigir_ct129((SELECT count(*) FROM vec_contratacion_temporal.cese_nombramiento_v1
    WHERE evento_ref=:'origen')=0,'migrador CT no abre RLS con GUC');
ROLLBACK;
RESET ROLE;
RESET SESSION AUTHORIZATION;
DROP SCHEMA prueba_ct129 CASCADE;
REVOKE vec_bolsa_llamamientos_relevo_cese FROM vec_ct129_relevo_prueba,vec_ct129_migrador_prueba;
REVOKE vec_bolsa_llamamientos_ejecutor FROM vec_ct129_bolsa_ajena_prueba;
REVOKE vec_contratacion_temporal_migrador FROM vec_ct129_migrador_prueba;
DROP ROLE vec_ct129_relevo_prueba,vec_ct129_bolsa_ajena_prueba,vec_ct129_migrador_prueba;
REVOKE vec_contratacion_temporal_ejecutor,vec_bolsa_llamamientos_propietario FROM vec_ct129_feed_prop_prueba;
REVOKE vec_contratacion_temporal_ejecutor,vec_bolsa_llamamientos_migrador FROM vec_ct129_feed_mig_prueba;
REVOKE vec_contratacion_temporal_ejecutor,vec_bolsa_llamamientos_ejecutor FROM vec_ct129_feed_eje_prueba;
DROP ROLE vec_ct129_feed_prop_prueba,vec_ct129_feed_mig_prueba,vec_ct129_feed_eje_prueba;
SELECT 'CT129 OK';
