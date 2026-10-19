\set ON_ERROR_STOP on
-- F1-61, semilla de la copia local: declara una publicación oficial ficticia
-- para cada convocatoria cuya versión vigente tenga hoy un único plazo de
-- inscripción abierto y todavía no tenga declaración. La fecha declarada es el
-- día anterior al primer día de ese plazo (hora de Madrid), de modo que la
-- copia conserva las convocatorias que tenía abiertas antes de BC14.
-- Medio: tablón de edictos electrónico, sin boletín; anuncio y clave derivados
-- de la versión, con prefijo COPIA. Es un dato ficticio, no una publicación
-- real ni un acto de RRHH.
-- Solo la aplica `copia_local.py` en cada arranque de la copia, ya con la
-- lista SQL y el kit de datos aplicados. No aplicarla en cidonia, en el
-- servidor nuevo ni en producción. Repetirla no escribe nada: las versiones ya
-- declaradas se saltan y la clave de idempotencia devuelve el mismo recibo.
-- Sin BC14 no hace nada.
SELECT pg_catalog.to_regprocedure(
 'vec_bolsa_convocatorias.declarar_publicacion_oficial_bases_v1(jsonb)') IS NOT NULL AS bc14 \gset
\if :bc14
BEGIN;
SET LOCAL ROLE vec_bolsa_convocatorias_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
DO $semilla$
DECLARE r record; v_huella text; v_recibo jsonb; v_declaradas integer:=0;
BEGIN
 FOR r IN
  SELECT v.convocatoria_id,v.secuencia,v.huella_version_sha256,min(p.abre) AS abre
  FROM vec_bolsa_convocatorias.version_convocatoria v
  CROSS JOIN LATERAL (
   SELECT (x.valor->>'abre_en')::timestamptz AS abre,(x.valor->>'cierra_en')::timestamptz AS cierra
   FROM jsonb_array_elements(
    CASE WHEN jsonb_typeof(convert_from(v.version_canonica,'UTF8')::jsonb#>'{contenido,plazos}')='array'
    THEN convert_from(v.version_canonica,'UTF8')::jsonb#>'{contenido,plazos}' ELSE '[]'::jsonb END) AS x(valor)
   WHERE x.valor->>'tipo'='inscripcion') AS p
  WHERE v.estado='publicada'
   AND p.abre<=clock_timestamp() AND clock_timestamp()<p.cierra
   AND NOT EXISTS(SELECT 1 FROM vec_bolsa_convocatorias.version_convocatoria posterior
    WHERE posterior.convocatoria_id=v.convocatoria_id AND posterior.secuencia>v.secuencia
     AND posterior.estado IN('publicada','sustituida','retirada'))
   AND NOT EXISTS(SELECT 1 FROM vec_bolsa_convocatorias.publicacion_oficial_bases d
    WHERE d.convocatoria_id=v.convocatoria_id AND d.secuencia=v.secuencia)
  GROUP BY v.convocatoria_id,v.secuencia,v.huella_version_sha256
  HAVING count(*)=1
  ORDER BY v.convocatoria_id,v.secuencia
 LOOP
  v_huella:=encode(sha256(convert_to(r.convocatoria_id||'#'||r.secuencia::text,'UTF8')),'hex');
  v_recibo:=vec_bolsa_convocatorias.declarar_publicacion_oficial_bases_v1(jsonb_build_object(
   'convocatoria_id',r.convocatoria_id,'secuencia',r.secuencia,
   'huella_version_sha256',r.huella_version_sha256,
   'medio','tablon_edictos_electronico','boletin_numero',NULL,
   'anuncio_ref','COPIA-'||upper(left(v_huella,16)),
   'fecha_publicacion',to_char((r.abre AT TIME ZONE 'Europe/Madrid')::date-1,'YYYY-MM-DD'),
   'direccion',NULL,'sustituye_ref',NULL,
   'declarada_por','actor:copia-local:datos-ficticios',
   'motivo_ref','motivos_bolsa:1:publicacion_bases_ensayo',
   'clave_idempotencia','copia-publicacion-'||left(v_huella,40)));
  RAISE NOTICE 'F1-61 copia: % v% declarada (%)',r.convocatoria_id,r.secuencia,v_recibo->>'recibo_ref';
  v_declaradas:=v_declaradas+1;
 END LOOP;
 RAISE NOTICE 'F1-61 copia: % declaraciones nuevas',v_declaradas;
END $semilla$;
COMMIT;
\endif
