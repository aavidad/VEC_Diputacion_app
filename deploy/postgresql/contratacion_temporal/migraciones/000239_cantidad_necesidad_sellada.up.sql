\set ON_ERROR_STOP on
-- F1-73: cantidad de la necesidad sellada en la solicitud de alta.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000239',0));
DO $pre$ DECLARE r record; v_oid text; v_oids text := ''; BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario'
    OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999 THEN
  RAISE EXCEPTION 'CT239: version PostgreSQL incompatible' USING ERRCODE='55000'; END IF;
 FOR r IN SELECT * FROM (VALUES
  ('vec_contratacion_temporal.datos_puesto_peticion_validos_ct200(jsonb)','c44d2fde96c5519556da232e4b9f1a9f11463d9953628c36ba2ff3ece2860cc3','{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario}',ARRAY['search_path=pg_catalog, pg_temp']::text[],false,'i',true,'u',false),
  ('vec_contratacion_temporal.reconstruir_solicitud_efecto_v2(jsonb)','27be1ce9ddcfc29ecea84873cdd86051994c1bc334b3270d8e8af1731dbfb08f','{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario}',ARRAY['search_path=pg_catalog']::text[],false,'i',true,'u',false),
  ('vec_contratacion_temporal.confirmar_alta_atestada_v1(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea)','7ac9119b0957ce300ee66a904015893bdeb186a575e27e54fa786689576f51cf','{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario}',ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']::text[],true,'v',false,'u',true),
  ('vec_contratacion_temporal.expediente_analisis_valido_v2(jsonb,boolean)','9375556f82f1ed8dc61249ada052e2f7ea77744bd62b994684743b0eae3a7fc1','{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario}',ARRAY['search_path=pg_catalog']::text[],false,'i',false,'u',false)
 ) AS x(firma,sha,acl,config,secdef,vol,strict_flag,parallel,retset) LOOP
  IF pg_catalog.to_regprocedure(r.firma) IS NULL THEN
   RAISE EXCEPTION 'CT239: funcion ausente %',r.firma USING ERRCODE='55000'; END IF;
  SELECT p.oid::text INTO v_oid FROM pg_catalog.pg_proc p
   WHERE p.oid=pg_catalog.to_regprocedure(r.firma)
    AND p.proowner='vec_contratacion_temporal_propietario'::regrole
    AND p.prosecdef=r.secdef AND p.provolatile=r.vol
    AND p.proisstrict=r.strict_flag AND p.proparallel=r.parallel
    AND p.proretset=r.retset
    AND p.proacl::text=r.acl AND p.proconfig=r.config
    AND pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
      pg_catalog.pg_get_functiondef(p.oid),'UTF8')),'hex')=r.sha;
  IF v_oid IS NULL THEN RAISE EXCEPTION 'CT239: preimagen incompatible %',r.firma
   USING ERRCODE='55000'; END IF;
  v_oids:=v_oids||'/'||v_oid; v_oid:=NULL;
 END LOOP;
 PERFORM pg_catalog.set_config('vec.ct239.oids',v_oids,true);
END $pre$;

-- Reconstituir desde cada cuerpo instalado, cuya huella se ha cotejado arriba.
-- Cada marca debe aparecer una sola vez; CREATE OR REPLACE conserva OID y ACL.
DO $ct239_rewrite$
DECLARE
  v_body text;
  v_old text;
  v_new text;
BEGIN
  v_body := pg_catalog.pg_get_functiondef(pg_catalog.to_regprocedure('vec_contratacion_temporal.datos_puesto_peticion_validos_ct200(jsonb)'));
  v_old := $ct239_ct200_datos_puesto_1_old$AS $function$
 SELECT CASE WHEN s ?| ARRAY['jornada_minutos','numero_personas','puesto_solicitado'] THEN
   s ?& ARRAY['jornada_minutos','numero_personas','puesto_solicitado']
$ct239_ct200_datos_puesto_1_old$;
  v_new := $ct239_ct200_datos_puesto_1_new$AS $function$
 SELECT CASE
 WHEN s ? 'numero_personas'
      AND NOT (s ?| ARRAY['jornada_minutos','puesto_solicitado']) THEN
   (CASE WHEN pg_catalog.jsonb_typeof(s->'numero_personas')='number'
               AND s->>'numero_personas' ~ '^[1-9][0-9]{0,9}$'
      THEN (s->>'numero_personas')::numeric BETWEEN 1 AND 4294967295
      ELSE false END)
   AND s ? 'necesidad'
   AND vec_contratacion_temporal.necesidad_alta_valida_v3(s) IS TRUE
   AND s->>'numero_personas' = s#>>'{necesidad,campos,numero_personas}'
 WHEN s ?| ARRAY['jornada_minutos','numero_personas','puesto_solicitado'] THEN
   s ?& ARRAY['jornada_minutos','numero_personas','puesto_solicitado']
$ct239_ct200_datos_puesto_1_new$;
  IF pg_catalog.length(v_body) - pg_catalog.length(pg_catalog.replace(v_body,v_old,''))
     <> pg_catalog.length(v_old) THEN
    RAISE EXCEPTION 'CT239: marca no unica ct200_datos_puesto:1' USING ERRCODE='55000';
  END IF;
  v_body := pg_catalog.replace(v_body,v_old,v_new);
  EXECUTE v_body;
  v_body := pg_catalog.pg_get_functiondef(pg_catalog.to_regprocedure('vec_contratacion_temporal.reconstruir_solicitud_efecto_v2(jsonb)'));
  v_old := $ct239_ct200_reconstruir_1_old$        ',"puesto_solicitado":' || vec_contratacion_temporal.texto_json_go_v1(s ->> 'puesto_solicitado')
       ELSE '' END) || '}'
$function$
$ct239_ct200_reconstruir_1_old$;
  v_new := $ct239_ct200_reconstruir_1_new$        ',"puesto_solicitado":' || vec_contratacion_temporal.texto_json_go_v1(s ->> 'puesto_solicitado')
       ELSE CASE WHEN s ? 'numero_personas' THEN
         ',"numero_personas":' || (s -> 'numero_personas')::text
       ELSE '' END END) || '}'
$function$
$ct239_ct200_reconstruir_1_new$;
  IF pg_catalog.length(v_body) - pg_catalog.length(pg_catalog.replace(v_body,v_old,''))
     <> pg_catalog.length(v_old) THEN
    RAISE EXCEPTION 'CT239: marca no unica ct200_reconstruir:1' USING ERRCODE='55000';
  END IF;
  v_body := pg_catalog.replace(v_body,v_old,v_new);
  EXECUTE v_body;
  v_body := pg_catalog.pg_get_functiondef(pg_catalog.to_regprocedure('vec_contratacion_temporal.confirmar_alta_atestada_v1(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea)'));
  v_old := $ct239_ct200_confirmar_1_old$             THEN 11 ELSE 10 END) +
       (CASE WHEN (a -> 'solicitud') ? 'puesto_solicitado' THEN 3 ELSE 0 END)
       OR NOT ((a -> 'solicitud') ?& ARRAY[
$ct239_ct200_confirmar_1_old$;
  v_new := $ct239_ct200_confirmar_1_new$             THEN 11 ELSE 10 END) +
       (CASE WHEN (a -> 'solicitud') ? 'puesto_solicitado' THEN 3 ELSE 0 END) +
       (CASE WHEN a ->> 'esquema' = 'vec.contratacion-temporal.efecto-alta.v3'
              AND (a -> 'solicitud') ? 'numero_personas'
              AND NOT ((a -> 'solicitud') ?| ARRAY['jornada_minutos','puesto_solicitado'])
              AND vec_contratacion_temporal.datos_puesto_peticion_validos_ct200(
                    a -> 'solicitud') IS TRUE
             THEN 1 ELSE 0 END)
       OR NOT ((a -> 'solicitud') ?& ARRAY[
$ct239_ct200_confirmar_1_new$;
  IF pg_catalog.length(v_body) - pg_catalog.length(pg_catalog.replace(v_body,v_old,''))
     <> pg_catalog.length(v_old) THEN
    RAISE EXCEPTION 'CT239: marca no unica ct200_confirmar:1' USING ERRCODE='55000';
  END IF;
  v_body := pg_catalog.replace(v_body,v_old,v_new);
  EXECUTE v_body;
  v_body := pg_catalog.pg_get_functiondef(pg_catalog.to_regprocedure('vec_contratacion_temporal.expediente_analisis_valido_v2(jsonb,boolean)'));
  v_old := $ct239_ct202_analisis_1_old$    END IF;
    -- CT200 puede aún no estar instalado. Se conserva su contrato de tres
    -- campos como dato opcional sin hacer depender CT202 de esa migración.
    IF s ?| ARRAY['jornada_minutos','numero_personas','puesto_solicitado'] THEN
        IF NOT (s ?& ARRAY['jornada_minutos','numero_personas','puesto_solicitado'])
$ct239_ct202_analisis_1_old$;
  v_new := $ct239_ct202_analisis_1_new$    END IF;
    -- La variante sellada admite solo el numero superior. La ausencia de
    -- ese numero conserva literalmente la solicitud historica.
    IF s ? 'numero_personas'
       AND NOT (s ?| ARRAY['jornada_minutos','puesto_solicitado']) THEN
        IF NOT v_necesidad
           OR vec_contratacion_temporal.datos_puesto_peticion_validos_ct200(
                v_solicitud_civil) IS NOT TRUE THEN
            RETURN false;
        END IF;
        v_claves_solicitud := pg_catalog.array_append(
            v_claves_solicitud,'numero_personas');
    -- El trio anterior sigue completo y con la misma validacion.
    ELSIF s ?| ARRAY['jornada_minutos','numero_personas','puesto_solicitado'] THEN
        IF NOT (s ?& ARRAY['jornada_minutos','numero_personas','puesto_solicitado'])
$ct239_ct202_analisis_1_new$;
  IF pg_catalog.length(v_body) - pg_catalog.length(pg_catalog.replace(v_body,v_old,''))
     <> pg_catalog.length(v_old) THEN
    RAISE EXCEPTION 'CT239: marca no unica ct202_analisis:1' USING ERRCODE='55000';
  END IF;
  v_body := pg_catalog.replace(v_body,v_old,v_new);
  EXECUTE v_body;
END $ct239_rewrite$;

DO $post$ DECLARE r record; v_oids text:=''; v_oid text; BEGIN
 FOR r IN SELECT * FROM (VALUES
  ('vec_contratacion_temporal.datos_puesto_peticion_validos_ct200(jsonb)','2a76bbd78cf1c907d84d5def7c5a5cc3d2113b1985467ef1c7c9721f77e214a4','{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario}',ARRAY['search_path=pg_catalog, pg_temp']::text[],false,'i',true,'u',false),
  ('vec_contratacion_temporal.reconstruir_solicitud_efecto_v2(jsonb)','771ee8bd20d0a1d26567a2bb6dd169d597a1f1ce23ba3bc6fa67f4c7d876de5e','{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario}',ARRAY['search_path=pg_catalog']::text[],false,'i',true,'u',false),
  ('vec_contratacion_temporal.confirmar_alta_atestada_v1(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea)','e831914180bca9cd63c8178aaa03ab1c86b7935bd05838368764c045b7a48ef4','{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario}',ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']::text[],true,'v',false,'u',true),
  ('vec_contratacion_temporal.expediente_analisis_valido_v2(jsonb,boolean)','e8080b22a5201ede7f08ca1076e09870d00e69fe1ebaa01bce8bb14f03a17f95','{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario}',ARRAY['search_path=pg_catalog']::text[],false,'i',false,'u',false)
 ) AS x(firma,sha,acl,config,secdef,vol,strict_flag,parallel,retset) LOOP
  SELECT p.oid::text INTO v_oid FROM pg_catalog.pg_proc p
   WHERE p.oid=pg_catalog.to_regprocedure(r.firma)
    AND p.proowner='vec_contratacion_temporal_propietario'::regrole
    AND p.prosecdef=r.secdef AND p.provolatile=r.vol
    AND p.proisstrict=r.strict_flag AND p.proparallel=r.parallel
    AND p.proretset=r.retset
    AND p.proacl::text=r.acl AND p.proconfig=r.config
    AND pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
      pg_catalog.pg_get_functiondef(p.oid),'UTF8')),'hex')=r.sha;
  IF v_oid IS NULL THEN RAISE EXCEPTION 'CT239: postimagen incompatible %',r.firma
   USING ERRCODE='55000'; END IF;
  v_oids:=v_oids||'/'||v_oid; v_oid:=NULL;
 END LOOP;
 IF v_oids IS DISTINCT FROM pg_catalog.current_setting('vec.ct239.oids') THEN
  RAISE EXCEPTION 'CT239: OID alterado' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
