\set ON_ERROR_STOP on
-- Solo con ct140_fixture_pg18.sql: autorización V3 sustituida por doble.
CREATE FUNCTION public.probar_ct140(p_expediente text,p_organizacion text,p_limite integer,
 p_cursor text DEFAULT '',p_campos_correctos boolean DEFAULT true)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER SET search_path=pg_catalog AS $f$
DECLARE m text; d bytea; c bytea; h text; campos jsonb; decision jsonb; capacidad jsonb;
BEGIN
 m:=jsonb_build_object('organizacion_ref',p_organizacion,'expediente_ref',p_expediente,
  'limite',p_limite,'cursor',p_cursor)::text;
 h:=encode(sha256(convert_to(
  '{"ambitos":{"organizacion_ref":"'||p_organizacion||
  '"},"atributos":{"material_sha256":"'||encode(sha256(convert_to(m,'UTF8')),'hex')||'"}}','UTF8')),'hex');
 campos:=CASE WHEN p_campos_correctos THEN
  '["antecedente_tipo","comunicacion_ref","estado","estado_respuesta","expediente_ref","llamamiento_ref","organizacion_ref","recibo_antecedente_ref","recibo_comunicacion_ref","registrada_en","version"]'::jsonb
  ELSE '["antecedente_tipo","comunicacion_ref","estado","expediente_ref","llamamiento_ref","organizacion_ref","recibo_antecedente_ref","recibo_comunicacion_ref","registrada_en","version"]'::jsonb END;
 decision:=jsonb_build_object('accion','contratacion_temporal.llamamiento.comunicaciones.consultar',
  'modulo_id','contratacion_temporal','tipo_recurso','expediente_contratacion_temporal',
  'finalidad','gestionar_contratacion_temporal','recurso_ref',p_expediente,
  'contexto_recurso_huella_sha256',h,'campos_permitidos',campos,'obligaciones','[]'::jsonb);
 d:=convert_to(decision::text,'UTF8');
 capacidad:=jsonb_build_object('suite','VEC-AD-3-COSE-EDDSA-1',
  'audiencia_consumo','vec_contratacion_temporal.comunicaciones_expediente.consultar.v1',
  'operacion','contratacion_temporal.llamamiento.comunicaciones.consultar',
  'efecto_ref',p_expediente,'huella_efecto_sha256',h,
  'huella_decision_sha256',encode(sha256(d),'hex'),'relleno',repeat('x',550));
 c:=convert_to(capacidad::text,'UTF8');
 RETURN vec_contratacion_temporal.consultar_comunicaciones_expediente_rrhh_v1(
  m,c,d,'motivo'::bytea,'contexto'::bytea,1,1,'payload'::bytea,
  'sobre'::bytea,'evidencia'::bytea,repeat('r',44)::bytea);
END $f$;
REVOKE ALL ON FUNCTION public.probar_ct140(text,text,integer,text,boolean) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION public.probar_ct140(text,text,integer,text,boolean) TO vec_ct140_login;

\connect postgres vec_ct140_login
BEGIN ISOLATION LEVEL SERIALIZABLE;
DO $prueba$
DECLARE p jsonb; q jsonb; v jsonb; ausente jsonb;
BEGIN
 p:=public.probar_ct140('expediente:ct140-a','organizacion:ct140-a',1);
 IF jsonb_array_length(p->'comunicaciones')<>1
    OR p->'comunicaciones'->0->>'comunicacion_ref'<>'comunicacion:ct140-a'
    OR p->'comunicaciones'->0->>'estado_respuesta'<>'registrada'
    OR p->'comunicaciones'->0->>'recibo_antecedente_ref'<>'recibo:seleccion-a'
    OR p->>'siguiente_cursor' IS DISTINCT FROM
      'comunicacion:ct140-a#'||encode(sha256(convert_to(
       'organizacion:ct140-a'||chr(10)||'expediente:ct140-a'||chr(10)||'2'||chr(10)||
       'comunicacion:ct140-a=registrada'||chr(10)||
       'comunicacion:ct140-a2=sin_respuesta'||chr(10),'UTF8')),'hex') THEN
  RAISE EXCEPTION 'CT140: primera página incorrecta: %',p; END IF;
 q:=public.probar_ct140('expediente:ct140-a','organizacion:ct140-a',1,p->>'siguiente_cursor');
 IF jsonb_array_length(q->'comunicaciones')<>1
    OR q->'comunicaciones'->0->>'comunicacion_ref'<>'comunicacion:ct140-a2'
    OR q->'comunicaciones'->0->>'estado_respuesta'<>'sin_respuesta'
    OR q->'comunicaciones'->0->>'antecedente_tipo'<>'continuacion_confirmada'
    OR q->'comunicaciones'->0->>'recibo_antecedente_ref'<>'recibo:continuacion-a'
    OR q->>'siguiente_cursor'<>''
 THEN RAISE EXCEPTION 'CT140: continuación incorrecta: %',q; END IF;
 v:=public.probar_ct140('expediente:ct140-vacio','organizacion:ct140-a',10);
 IF jsonb_array_length(v->'comunicaciones')<>0 THEN
  RAISE EXCEPTION 'CT140: vacío incorrecto: %',v; END IF;
 BEGIN
  PERFORM public.probar_ct140('expediente:ct140-a','organizacion:ct140-a',1,'',false);
  RAISE EXCEPTION 'CT140: aceptó concesión V3 antigua sin estado_respuesta';
 EXCEPTION WHEN SQLSTATE 'P1403' THEN NULL; END;
 ausente:=public.probar_ct140('expediente:ct140-a','organizacion:ct140-b',1);
 IF ausente->'encontrado' IS DISTINCT FROM 'false'::jsonb
    OR jsonb_array_length(ausente->'comunicaciones')<>0
    OR ausente->>'siguiente_cursor'<>''
 THEN RAISE EXCEPTION 'CT140: filtró expediente ajeno: %',ausente; END IF;
 ausente:=public.probar_ct140('expediente:ct140-a','organizacion:ct140-a',1,
  'comunicacion:ct140-c#'||right(p->>'siguiente_cursor',64));
 IF ausente->'encontrado' IS DISTINCT FROM 'false'::jsonb
    OR jsonb_array_length(ausente->'comunicaciones')<>0
    OR ausente->>'siguiente_cursor'<>''
 THEN RAISE EXCEPTION 'CT140: aceptó cursor ajeno: %',ausente; END IF;
END $prueba$;
COMMIT;
