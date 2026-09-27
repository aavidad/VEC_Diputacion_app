\set ON_ERROR_STOP on
-- Ejecutar tras CT129 sobre la base sintética de pruebas con un cese CT115
-- de llamamiento. Solo lectura: ninguna fila de negocio se altera.
CREATE FUNCTION pg_temp.exigir_ct129(p_ok boolean,p_caso text)
RETURNS void LANGUAGE plpgsql AS $$
BEGIN
    IF p_ok IS DISTINCT FROM true THEN
        RAISE EXCEPTION 'CT129: %',p_caso;
    END IF;
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
 ORDER BY c.confirmada_en DESC LIMIT 1 \gset

SET SESSION AUTHORIZATION vec_ct115_runtime;
BEGIN ISOLATION LEVEL REPEATABLE READ READ ONLY;
SELECT huella_sha256 AS huella, origen_posicion AS posicion
  FROM vec_contratacion_temporal.leer_contratos_bolsa_v1(NULL,NULL,100)
 WHERE origen_ref=:'origen' AND evento->>'tipo'='cese' \gset
COMMIT;
RESET SESSION AUTHORIZATION;

SELECT pg_temp.exigir_ct129(
    NOT has_table_privilege('vec_bolsa_llamamientos_propietario','vec_contratacion_temporal.cese_nombramiento_v1','SELECT')
    AND NOT has_table_privilege('vec_bolsa_llamamientos_propietario','vec_contratacion_temporal.incorporacion_registro_v2','SELECT'),
    'Bolsa no puede leer tablas CT');
SET ROLE vec_bolsa_llamamientos_propietario;
SELECT * FROM vec_contratacion_temporal.verificar_cese_publicado_bolsa_v1(:'origen',:'huella',:'posicion'::bigint) \gset verificado_
SELECT pg_temp.exigir_ct129(
    :'verificado_modalidad_clave'=:'modalidad'
    AND :'verificado_causa_contrato_clave'=:'causa_contrato'
    AND :'verificado_causa_cese_clave'=:'causa_cese'
    AND :'verificado_fecha_efecto'=:'fecha'
    AND :'verificado_fuente_tipo'=:'tipo'
    AND :'verificado_fuente_ref'=:'fuente'
    AND :'verificado_fuente_sha256'=:'fuente_sha'
    AND :'verificado_relacion_ref'=:'relacion'
    AND :'verificado_version_resultante'=:'version'
    AND :'verificado_recibo_ref'=:'recibo'
    AND :'verificado_incorporacion_ref'=:'incorporacion'
    AND :'verificado_llamamiento_ref'=:'llamamiento'
    AND :'verificado_organizacion_ref'=:'organizacion'
    AND :'verificado_expediente_ref'=:'expediente',
    'hecho acreditado y fuente exactos');
SELECT pg_temp.exigir_ct129(
    (SELECT count(*) FROM vec_contratacion_temporal.verificar_cese_publicado_bolsa_v1(
        :'origen',repeat('0',64),:'posicion'::bigint))=0
    AND (SELECT count(*) FROM vec_contratacion_temporal.verificar_cese_publicado_bolsa_v1(
        :'origen',:'huella',:'posicion'::bigint+1))=0
    AND (SELECT count(*) FROM vec_contratacion_temporal.verificar_cese_publicado_bolsa_v1(
        'evento:ct:inventado',:'huella',:'posicion'::bigint))=0,
    'origen, huella y posición forjados denegados');
SELECT pg_temp.exigir_ct129(
    coalesce(current_setting('vec.ct129.origen_ref',true),'')='',
    'marca RLS restaurada');
RESET ROLE;
SELECT 'CT129 OK';
