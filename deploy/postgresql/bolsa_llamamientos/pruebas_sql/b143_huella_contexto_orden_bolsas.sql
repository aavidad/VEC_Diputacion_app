\set ON_ERROR_STOP on
-- B143 en copia desechable, después de B138 y B143. Todo revierte.
-- Una decisión emitida para guardar modalidades no sirve para el orden de
-- bolsas; la del contexto fijo sigue llegando al consumidor V3.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $metadatos$
DECLARE f text;
BEGIN
 FOREACH f IN ARRAY ARRAY[
  'vec_bolsa_llamamientos.ordenar_preferencias_bolsas_v1(text,text[],bigint,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_bolsa_llamamientos.ordenar_preferencias_bolsas_interna_v1(text,text[],bigint,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'] LOOP
  IF strpos((SELECT prosrc FROM pg_proc WHERE oid=f::regprocedure),
   'OR c->>''huella_efecto_sha256'' IS DISTINCT FROM encode(sha256(convert_to(')=0
  OR has_function_privilege('public',f,'EXECUTE')
  THEN RAISE EXCEPTION 'B143: falta el recálculo o hay EXECUTE público en %',f USING ERRCODE='55000'; END IF;
 END LOOP;
 IF encode(sha256(convert_to(pg_get_functiondef(
   'vec_bolsa_llamamientos.ordenar_preferencias_bolsas_v1(text,text[],bigint,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure),'UTF8')),'hex')
   IS DISTINCT FROM '09e84cd5c85318f3f047b73f98dbbfc1ee00d8f695f597de84d31c943b56c11e'
 OR encode(sha256(convert_to(pg_get_functiondef(
   'vec_bolsa_llamamientos.ordenar_preferencias_bolsas_interna_v1(text,text[],bigint,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure),'UTF8')),'hex')
   IS DISTINCT FROM 'ce0e92ef647ea0a4c2e3935e3914c1ffd9da376b58a830520eff32c56ff67a21'
 THEN RAISE EXCEPTION 'B143: postimagen distinta de la migración' USING ERRCODE='55000'; END IF;
END $metadatos$;

SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
DO $casos$
DECLARE cand text:='can_'||repeat('a',43);
 fija text:='b0084709beee455ac6927bc55e23d95e7e1e727a80b2f798db16f0ba8898b19c';
 modal text;caso text;h text;c jsonb;d jsonb;x jsonb;e text;m text;r jsonb:='{}'::jsonb;
BEGIN
 -- Huella fija compartida con Go (HuellaContextoAutorizacionSHA256 del recurso
 -- de Ordenar); la de modalidades sale de la función de B138.
 IF encode(sha256(convert_to('{"ambitos":{"candidato_ref":"'||cand||
   '"},"atributos":{"propiedad":"candidato"}}','UTF8')),'hex') IS DISTINCT FROM fija
 THEN RAISE EXCEPTION 'B143: huella fija Go/SQL divergente' USING ERRCODE='55000'; END IF;
 modal:=vec_bolsa_llamamientos.huella_contexto_modalidades_v1(cand,repeat('c',64));
 x:=jsonb_build_object('vinculos',jsonb_build_array(jsonb_build_object(
  'tipo','candidato','estado','activo','referencia',cand)));
 FOREACH caso IN ARRAY ARRAY['fija','modalidades','ajena'] LOOP
  h:=CASE caso WHEN 'fija' THEN fija WHEN 'modalidades' THEN modal ELSE repeat('d',64) END;
  c:=jsonb_build_object('operacion','bolsa.preferencias_bolsas.ordenar_propias',
   'audiencia_consumo','vec.bolsa.preferencias-bolsas.ordenar.v1',
   'efecto_ref','mi-bolsa:'||cand,'huella_efecto_sha256',h);
  d:=jsonb_build_object('accion','bolsa.preferencias_bolsas.ordenar_propias','modulo_id','bolsa',
   'tipo_recurso','participaciones_candidato','finalidad','gestion_preferencias_bolsas_propias',
   'recurso_ref','mi-bolsa:'||cand,'contexto_recurso_huella_sha256',h,
   'campos_permitidos','[]'::jsonb,'obligaciones','[]'::jsonb);
  BEGIN
   PERFORM * FROM vec_bolsa_llamamientos.ordenar_preferencias_bolsas_v1(
    cand,ARRAY['bolsa:b143'],0,'clave-b143-'||caso,clock_timestamp(),
    convert_to(c::text,'UTF8'),convert_to(d::text,'UTF8'),'\x00'::bytea,convert_to(x::text,'UTF8'),
    1,1,'\x00'::bytea,'\x00'::bytea,'\x00'::bytea,'\x00'::bytea);
   e:='00000';m:='sin error';
  EXCEPTION WHEN OTHERS THEN GET STACKED DIAGNOSTICS e=RETURNED_SQLSTATE,m=MESSAGE_TEXT;
  END;
  r:=r||jsonb_build_object(caso,jsonb_build_object('estado',e,'mensaje',m));
 END LOOP;
 -- La decisión de modalidades y una huella ajena se rechazan antes del consumo.
 IF r#>>'{modalidades,estado}' IS DISTINCT FROM '42501' OR r#>>'{modalidades,mensaje}' IS DISTINCT FROM 'B104: orden denegado'
 OR r#>>'{ajena,estado}' IS DISTINCT FROM '42501' OR r#>>'{ajena,mensaje}' IS DISTINCT FROM 'B104: orden denegado'
 -- La del contexto fijo pasa la guarda nueva y llega al consumidor V3, que
 -- rechaza este material falso con otro error.
 OR r#>>'{fija,estado}'='00000' OR r#>>'{fija,mensaje}' LIKE 'B104:%'
 THEN RAISE EXCEPTION 'B143: casos divergentes %',r USING ERRCODE='55000'; END IF;
 RAISE NOTICE 'B143 casos: %',r;
END $casos$;
SELECT 'B143 OK' AS resultado;
ROLLBACK;
