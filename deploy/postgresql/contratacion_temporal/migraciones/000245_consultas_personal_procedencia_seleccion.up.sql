\set ON_ERROR_STOP on
-- CT245: referencias de lectura Personal por página dentro de la procedencia
-- firmada del recibo O6. Conserva exactamente el canon anterior si faltan.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000245',0));

DO $pre$
DECLARE r record; p pg_proc%ROWTYPE; actual text;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
  OR pg_is_in_recovery()
  OR to_regprocedure('vec_contratacion_temporal.consultas_personal_json_seleccion_o6_v1(jsonb)') IS NOT NULL
 THEN RAISE EXCEPTION 'CT245: entorno o función auxiliar divergente' USING ERRCODE='55000'; END IF;
 FOR r IN SELECT * FROM (VALUES
  ('vec_contratacion_temporal.recibo_json_seleccion_llamamiento_o6_v1(jsonb)', 'b44fac672d69921fe48b57142895f7a4603c6f138808b41aae716412bea26a84', false,
   ARRAY['search_path=pg_catalog']::text[], ARRAY['vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario']::aclitem[]),
  ('vec_contratacion_temporal.artefacto_json_seleccion_llamamiento_o6_v1(jsonb,boolean)', 'f5c2c1a88e9700c6473ecf2f707eaa779774af0af6e952f614a0f7f92fdc4ee9', false,
   ARRAY['search_path=pg_catalog']::text[], ARRAY['vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario']::aclitem[]),
  ('vec_contratacion_temporal.huellas_materiales_seleccion_llamamiento_o6_v1(jsonb)', '144b8375e27e01a811dd3b3b34ff1d26ad172c2fbb9fd8a0fd0257a1e9662c79', false,
   ARRAY['search_path=pg_catalog']::text[], ARRAY['vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario']::aclitem[]),
  ('vec_contratacion_temporal.confirmar_seleccion_llamamiento_o6_v1(uuid,text,text,text,text,text)', '4aeb436aa99b051ea819cbf54f1e73b4b8a21eda54b0f68b7454b7e1b43450b3', true,
   ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s','statement_timeout=15s','idle_in_transaction_session_timeout=20s']::text[],
   ARRAY['vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario','vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario']::aclitem[])
 ) AS esperado(firma,huella,definer,config,acl) LOOP
  SELECT * INTO p FROM pg_proc WHERE oid=to_regprocedure(r.firma);
  IF NOT FOUND OR p.proowner IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
   OR p.prosecdef IS DISTINCT FROM r.definer OR p.proconfig IS DISTINCT FROM r.config
   OR p.proacl IS DISTINCT FROM r.acl THEN
   RAISE EXCEPTION 'CT245: propiedad, ACL o configuración divergente: %',r.firma USING ERRCODE='55000';
  END IF;
  actual:=encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex');
  IF actual IS DISTINCT FROM r.huella THEN
   RAISE EXCEPTION 'CT245: preimagen divergente: % actual=% esperado=%',r.firma,actual,r.huella USING ERRCODE='55000';
  END IF;
 END LOOP;
END $pre$;

CREATE FUNCTION vec_contratacion_temporal.consultas_personal_json_seleccion_o6_v1(p_procedencia jsonb)
RETURNS text LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE STRICT
SET search_path TO 'pg_catalog'
AS $function$
DECLARE v_consultas jsonb; v_pagina jsonb; v_texto text:='[';
BEGIN
 IF jsonb_typeof(p_procedencia) IS DISTINCT FROM 'object' THEN RETURN NULL; END IF;
 IF NOT p_procedencia ? 'consultas_personal' THEN RETURN ''; END IF;
 IF p_procedencia->'sin_retencion_declarada' IS DISTINCT FROM 'true'::jsonb THEN RETURN NULL; END IF;
 v_consultas:=p_procedencia->'consultas_personal';
 IF jsonb_typeof(v_consultas) IS DISTINCT FROM 'array' THEN RETURN NULL; END IF;
 IF jsonb_array_length(v_consultas) NOT BETWEEN 1 AND 64 THEN RETURN NULL; END IF;
 FOR v_pagina IN SELECT value FROM jsonb_array_elements(v_consultas) WITH ORDINALITY AS paginas(value, orden) ORDER BY orden LOOP
  IF jsonb_typeof(v_pagina) IS DISTINCT FROM 'object'
   OR (v_pagina ?& ARRAY['recibo_ref','auditoria_ref']) IS DISTINCT FROM true
   OR v_pagina - ARRAY['recibo_ref','auditoria_ref'] IS DISTINCT FROM '{}'::jsonb
   OR jsonb_typeof(v_pagina->'recibo_ref') IS DISTINCT FROM 'string'
   OR jsonb_typeof(v_pagina->'auditoria_ref') IS DISTINCT FROM 'string'
   OR v_pagina->>'recibo_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
   OR v_pagina->>'auditoria_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
  THEN RETURN NULL; END IF;
  IF v_texto <> '[' THEN v_texto:=v_texto||','; END IF;
  v_texto:=v_texto||'{"recibo_ref":'||(v_pagina->'recibo_ref')::text||
   ',"auditoria_ref":'||(v_pagina->'auditoria_ref')::text||'}';
 END LOOP;
 RETURN v_texto||']';
END
$function$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.consultas_personal_json_seleccion_o6_v1(jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.consultas_personal_json_seleccion_o6_v1(jsonb) TO vec_contratacion_temporal_propietario;

-- Los reemplazos se realizan sobre pg_get_functiondef de la copia instalada.
-- Cada marca debe aparecer una sola vez dentro de su función preatestada.
DO $patch$
DECLARE r record; v_oid oid; v_def text; v_nuevo text; v_num integer;
BEGIN
 FOR r IN SELECT * FROM (VALUES
  ('vec_contratacion_temporal.recibo_json_seleccion_llamamiento_o6_v1(jsonb)',
   $old$        CASE WHEN v_procedencia->'politica_provisional' = 'true'::jsonb
             THEN ',"politica_provisional":true' ELSE '' END || '}';$old$,
   $new$        CASE WHEN v_procedencia->'politica_provisional' = 'true'::jsonb
             THEN ',"politica_provisional":true' ELSE '' END ||
        CASE WHEN v_procedencia ? 'consultas_personal'
             THEN ',"consultas_personal":' || vec_contratacion_temporal.consultas_personal_json_seleccion_o6_v1(v_procedencia)
             ELSE '' END || '}';$new$),
  ('vec_contratacion_temporal.artefacto_json_seleccion_llamamiento_o6_v1(jsonb,boolean)',
   $old$        CASE WHEN v_procedencia->'politica_provisional' = 'true'::jsonb
             THEN ',"politica_provisional":true' ELSE '' END || '}';$old$,
   $new$        CASE WHEN v_procedencia->'politica_provisional' = 'true'::jsonb
             THEN ',"politica_provisional":true' ELSE '' END ||
        CASE WHEN v_procedencia ? 'consultas_personal'
             THEN ',"consultas_personal":' || vec_contratacion_temporal.consultas_personal_json_seleccion_o6_v1(v_procedencia)
             ELSE '' END || '}';$new$),
  ('vec_contratacion_temporal.huellas_materiales_seleccion_llamamiento_o6_v1(jsonb)',
   $old$        ELSE '' END;
    v_respuesta := v_respuesta ||
        vec_contratacion_temporal.referencia_material_seleccion_llamamiento_o6_v1(
            'comando_necesidad'$old$,
   $new$        ELSE '' END;
    IF v_procedencia ? 'consultas_personal' THEN
        v_respuesta := v_respuesta ||
            vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
                'consultas_personal',
                vec_contratacion_temporal.consultas_personal_json_seleccion_o6_v1(v_procedencia));
    END IF;
    v_respuesta := v_respuesta ||
        vec_contratacion_temporal.referencia_material_seleccion_llamamiento_o6_v1(
            'comando_necesidad'$new$),
  ('vec_contratacion_temporal.confirmar_seleccion_llamamiento_o6_v1(uuid,text,text,text,text,text)',
   $old$           'fuente','evidencia','sin_retencion_declarada','politica_provisional'] IS DISTINCT FROM '{}'::jsonb$old$,
   $new$           'fuente','evidencia','sin_retencion_declarada','politica_provisional','consultas_personal'] IS DISTINCT FROM '{}'::jsonb
       OR vec_contratacion_temporal.consultas_personal_json_seleccion_o6_v1(v_procedencia) IS NULL$new$)
 ) AS cambios(firma,anterior,nuevo) LOOP
  v_oid:=to_regprocedure(r.firma);
  v_def:=pg_get_functiondef(v_oid);
  v_num:=(length(v_def)-length(replace(v_def,r.anterior,'')))/length(r.anterior);
  IF v_num IS DISTINCT FROM 1 THEN
   RAISE EXCEPTION 'CT245: marca no única en %: %',r.firma,v_num USING ERRCODE='55000';
  END IF;
  v_nuevo:=replace(v_def,r.anterior,r.nuevo);
  EXECUTE v_nuevo;
 END LOOP;
END $patch$;

DO $post$
DECLARE r record; p pg_proc%ROWTYPE; actual text;
BEGIN
 FOR r IN SELECT * FROM (VALUES
  ('vec_contratacion_temporal.consultas_personal_json_seleccion_o6_v1(jsonb)', '38158eb3a52b7f045534fbfbb1c307c21c3811915cc77aa98f27c41743489d39',
   ARRAY['vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario']::aclitem[], ARRAY['search_path=pg_catalog']::text[], false),
  ('vec_contratacion_temporal.recibo_json_seleccion_llamamiento_o6_v1(jsonb)', 'f4d0c82adb74f86ea94ab353cce409b830cb99c055a622bdb484c06794aa061e',
   ARRAY['vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario']::aclitem[], ARRAY['search_path=pg_catalog']::text[], false),
  ('vec_contratacion_temporal.artefacto_json_seleccion_llamamiento_o6_v1(jsonb,boolean)', 'edba5d28a466f2d9599a9d464246eef2374b26df15159bf625f62db9c9d74d19',
   ARRAY['vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario']::aclitem[], ARRAY['search_path=pg_catalog']::text[], false),
  ('vec_contratacion_temporal.huellas_materiales_seleccion_llamamiento_o6_v1(jsonb)', 'af0ee77d62fb27f40f1b82849f051fd428cca5134696317afeb7c71955ce5b9d',
   ARRAY['vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario']::aclitem[], ARRAY['search_path=pg_catalog']::text[], false),
  ('vec_contratacion_temporal.confirmar_seleccion_llamamiento_o6_v1(uuid,text,text,text,text,text)', 'f198a80b22e0abb6b53a3836c37ac129e8fb311f01ba3d3ebdda1f750b8c3577',
   ARRAY['vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario','vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario']::aclitem[],
   ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s','statement_timeout=15s','idle_in_transaction_session_timeout=20s']::text[], true)
 ) AS esperado(firma,huella,acl,config,definer) LOOP
  SELECT * INTO p FROM pg_proc WHERE oid=to_regprocedure(r.firma);
  actual:=encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex');
  IF NOT FOUND OR actual IS DISTINCT FROM r.huella
   OR p.proowner IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
   OR p.proacl IS DISTINCT FROM r.acl OR p.proconfig IS DISTINCT FROM r.config
   OR p.prosecdef IS DISTINCT FROM r.definer THEN
   RAISE EXCEPTION 'CT245: postimagen o ACL divergente: % actual=% esperado=%',r.firma,actual,r.huella USING ERRCODE='55000';
  END IF;
 END LOOP;
END $post$;
COMMIT;
