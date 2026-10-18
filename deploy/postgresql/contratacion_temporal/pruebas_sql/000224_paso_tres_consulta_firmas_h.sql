\set ON_ERROR_STOP on
-- Puerta focal de instalación para la copia H, tras CT224.
-- El recorrido real de Secretaría conserva su PDF y su clave originales.
BEGIN READ ONLY;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
DO $prueba$
DECLARE
 f record;
 esperado text;
 cuerpo text;
 total integer:=0;
BEGIN
 FOR f IN SELECT n.nspname,p.proname,p.prosrc,p.proowner,p.proacl,p.proconfig
  FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
  WHERE n.nspname='vec_contratacion_temporal'
    AND p.proname='consultar_firmas_r5_atestadas_v3' LOOP
  total:=total+1;
  esperado:='d71de0dbe2a8677c4d24f7058886a3309ab1262bd0af6b9e6f170933a1210fd4';
  cuerpo:=f.prosrc;
  IF encode(sha256(convert_to(cuerpo,'UTF8')),'hex')<>esperado
   OR (length(cuerpo)-length(replace(cuerpo,'^[1-3]$','')))<>length('^[1-3]$')
   OR (length(cuerpo)-length(replace(cuerpo,$marker$s->>'PasoOrden'='3' AND s->'UnidadRef' IS NOT DISTINCT FROM 'null'::jsonb$marker$,'')))
      <>length($marker$s->>'PasoOrden'='3' AND s->'UnidadRef' IS NOT DISTINCT FROM 'null'::jsonb$marker$)
   OR f.proconfig[1] IS DISTINCT FROM 'search_path=pg_catalog, pg_temp'
   OR f.proacl IS NULL OR EXISTS (SELECT 1 FROM aclexplode(f.proacl) WHERE grantee=0)
   OR f.proowner IS DISTINCT FROM to_regrole(
     CASE WHEN f.nspname='vec_autorizacion_atestada_v3' THEN 'vec_autorizacion_atestada_v3_propietario'
          ELSE 'vec_contratacion_temporal_propietario' END) THEN
   RAISE EXCEPTION 'Paso 3: función % . % distinta',f.nspname,f.proname USING ERRCODE='55000';
  END IF;
 END LOOP;
 IF total<>1 THEN RAISE EXCEPTION 'Paso 3: falta consulta nominal' USING ERRCODE='55000'; END IF;
 IF NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=to_regprocedure(
   'vec_contratacion_temporal.recuperar_firmas_r5_atestadas_v3(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')
   AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='b202a04472fd5819c72694cb176ba9d7047e29bec4d1362ea89904cdb97fdd7b')
  OR NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=to_regprocedure(
   'vec_autorizacion_atestada_v3.comprobar_consumo_recuperacion_firmas_ct_v2(bytea,jsonb)')
   AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='897eac68c1f79337250e825c41e500a79e45c3237908a336576f82d469e35fcb') THEN
  RAISE EXCEPTION 'Paso 3: recuperación histórica modificada' USING ERRCODE='55000';
 END IF;
 IF vec_catalogos_configurables.paso_plan_firma_con_unidad_v1(
   '6701a125002172bf3688507da68d5cded7015230fa0f94214702e28b2656d512',
   'resolucion',3,'organizacion:desarrollo:dipgra','unidad:desarrollo:rrhh') IS NOT TRUE
  OR vec_catalogos_configurables.paso_plan_firma_con_unidad_v1(
   '6701a125002172bf3688507da68d5cded7015230fa0f94214702e28b2656d512',
   'informe_definitivo',3,'organizacion:desarrollo:dipgra','unidad:desarrollo:rrhh') IS NOT FALSE
  OR vec_catalogos_configurables.paso_plan_firma_con_unidad_v1(
   '6701a125002172bf3688507da68d5cded7015230fa0f94214702e28b2656d512',
   'resolucion',4,'organizacion:desarrollo:dipgra','unidad:desarrollo:rrhh') IS NOT FALSE THEN
  RAISE EXCEPTION 'Paso 3: publicación nominal distinta' USING ERRCODE='55000';
 END IF;
 RAISE NOTICE 'Paso 3: consulta acotada, plan y recuperación histórica verificados; falta recorrido real con Secretaría';
END $prueba$;
ROLLBACK;
