\set ON_ERROR_STOP on
-- CT84: referencias externas originales; compatibilidad sintáctica, no autoridad.
-- Sólo tres cuerpos; CREATE OR REPLACE desde catálogo conserva OID/firmas/ACL.
-- Postimagen CT82/CT83 obligatoria. Avance: no reaplicar ni DOWN sobre historia.
-- Unidad = legacy ref64 no cero, o namespace unidad: no vacío + dominio opaco
-- CT tipos.go/ReferenciaOpacaValida (ASCII, máximo 160 bytes). Personal reutiliza
-- su referencia_alta_ejercicio_valida_v1; su legacy SQL también excluye cero.
-- Actor = legacy o per_32lowerhex; documento = legacy o documento-resolucion:
-- seguido de 64lowerhex no cero. Ningún identificador se transforma.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal.dependencias.alta_ejercicio.v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal.dependencias.incorporacion_ejercicio.v2',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:000073:seguimiento:v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:000084:referencias-externas',0));
SET LOCAL ROLE vec_contratacion_temporal_propietario;

DO $ct84$
DECLARE
 propietario oid:='vec_contratacion_temporal_propietario'::regrole;
 r record; cambio record; p pg_proc%ROWTYPE; f oid; objetivos oid[]; ddl text; nuevo text;
 antes jsonb; otras jsonb; deps jsonb; comentario text;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario' OR getdatabaseencoding()<>'UTF8'
  OR NOT EXISTS (SELECT 1 FROM pg_namespace WHERE nspname='vec_contratacion_temporal' AND nspowner=propietario)
  OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE oid=propietario AND NOT rolcanlogin AND NOT rolinherit
   AND NOT rolsuper AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls)
 THEN RAISE EXCEPTION 'CT84: propietario CT incompatible' USING ERRCODE='55000'; END IF;
 -- Dependencias ya instaladas: se cotejan, nunca se reaplican ni se reescriben.
 IF (SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') FROM pg_proc
      WHERE oid=to_regprocedure('vec_contratacion_temporal.leer_raices_incorporacion_ejercicio_v2(text,text,text)'))
       IS DISTINCT FROM 'bd1a710efadaf37702cffa672e64ddd87f94eb5e7a90bffa02e0f4cfafbb55d8'
  OR (SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') FROM pg_proc
      WHERE oid=to_regprocedure('vec_contratacion_temporal.registrar_incorporacion_ejercicio_v2(jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bigint,bigint,bytea,bytea,bytea,bytea,jsonb)'))
       IS DISTINCT FROM '44dab34a4359acc0982096b3d26e9583570c69b60fc17c68030afe53ea0686f7'
 THEN RAISE EXCEPTION 'CT84: postimagen CT82/CT83 requerida' USING ERRCODE='55000'; END IF;
 objetivos:=ARRAY[to_regprocedure('vec_contratacion_temporal.seguimiento73_escalar(jsonb,text)')::oid,
                  to_regprocedure('vec_contratacion_temporal.seguimiento73_nodo(jsonb,text)')::oid];
 SELECT jsonb_agg(jsonb_build_object('funcion',to_jsonb(z),'comentario',obj_description(z.oid,'pg_proc')) ORDER BY z.oid)
  INTO otras FROM pg_proc z WHERE z.pronamespace='vec_contratacion_temporal'::regnamespace AND NOT z.oid=ANY(objetivos);
 FOR r IN SELECT * FROM (VALUES
  ('vec_contratacion_temporal.seguimiento73_escalar(jsonb,text)','ce177f68dc65a4f73ac044880c6eed21b75885f951e3ee463e36cec921033a25','eefa5c10216a846e4721b40fe596e3416f8243d7b2c99d5cebc8c99625a5d9c3','e062319e3792a9ff04b5c6f5d0e387c6748ebe144574463acd1c116e8813a486'),
  ('vec_contratacion_temporal.seguimiento73_nodo(jsonb,text)','f3ec7c10365b28c589d26b5d431bb893ffe1e47eabfccdb89c99002d0922c813','b20927a4ec4dcc599c6506d6e9a6f50834ebdf407a1492b1d353645c156a9788','94aa984b54212b1d0ff456359a322e861b11b9f4bd28e9472f9543af059b1d38')
 ) AS cambios(firma,pre_sha,ddl_sha,post_sha) LOOP
  f:=to_regprocedure(r.firma);
  SELECT * INTO STRICT p FROM pg_proc WHERE oid=f;
  ddl:=pg_get_functiondef(f);
  IF p.proowner<>propietario OR p.prosecdef
   OR encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM r.pre_sha
   OR encode(sha256(convert_to(ddl,'UTF8')),'hex') IS DISTINCT FROM r.ddl_sha
   OR p.proacl IS DISTINCT FROM ARRAY[makeaclitem(propietario,propietario,'EXECUTE',false)]
   OR (SELECT count(*) FROM pg_proc WHERE pronamespace=p.pronamespace AND proname=p.proname)<>1
  THEN RAISE EXCEPTION 'CT84: preimagen/contrato/ACL CT no exactos' USING ERRCODE='55000'; END IF;
  antes:=to_jsonb(p)-'prosrc'; comentario:=obj_description(f,'pg_proc');
  SELECT jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype)
   INTO deps FROM pg_depend d WHERE (d.classid='pg_proc'::regclass AND d.objid=f)
    OR (d.refclassid='pg_proc'::regclass AND d.refobjid=f);
  nuevo:=p.prosrc;
  FOR cambio IN SELECT * FROM (VALUES
  ('seguimiento73_escalar',$o0$  WHEN 'hash' THEN s ~ '^[0-9a-f]{64}$' AND s<>repeat('0',64)$o0$,$d0$  WHEN 'actor' THEN
   (s ~ '^ref:[0-9a-f]{64}$' AND s<>'ref:'||repeat('0',64))
   OR s COLLATE "C" ~ '^per_[0-9a-f]{32}$' AND s<>'per_'||repeat('0',32)
  WHEN 'unidad' THEN
   (s ~ '^ref:[0-9a-f]{64}$' AND s<>'ref:'||repeat('0',64))
   OR (octet_length(s) BETWEEN 8 AND 160 AND s COLLATE "C" ~ '^unidad:[A-Za-z0-9._:/#-]+$')
  WHEN 'documento_ref' THEN
   (s ~ '^ref:[0-9a-f]{64}$' AND s<>'ref:'||repeat('0',64))
   OR (s COLLATE "C" ~ '^documento-resolucion:[0-9a-f]{64}$'
    AND s<>'documento-resolucion:'||repeat('0',64))
  WHEN 'hash' THEN s ~ '^[0-9a-f]{64}$' AND s<>repeat('0',64)$d0$),
  ('seguimiento73_nodo',$o1$'u1','ref','organizacion','expediente','hash'$o1$,$d1$'u1','ref','organizacion','expediente','actor','unidad','documento_ref','hash'$d1$),
  ('seguimiento73_nodo',$o2$actor_ref:ref|unidad_ref:ref$o2$,$d2$actor_ref:actor|unidad_ref:unidad$d2$),
  ('seguimiento73_nodo',$o3$WHEN 'documento' THEN spec:='tipo_clave:clave|referencia:ref';$o3$,$d3$WHEN 'documento' THEN spec:='tipo_clave:clave|referencia:documento_ref';$d3$)
  ) AS sustituciones(nombre,origen,destino) WHERE nombre=p.proname LOOP
   IF (length(nuevo)-length(replace(nuevo,cambio.origen,'')))/length(cambio.origen)<>1 THEN
    RAISE EXCEPTION 'CT84: sustitución CT ambigua' USING ERRCODE='55000'; END IF;
   nuevo:=replace(nuevo,cambio.origen,cambio.destino);
  END LOOP;
  IF (length(ddl)-length(replace(ddl,p.prosrc,'')))/length(p.prosrc)<>1
   OR encode(sha256(convert_to(nuevo,'UTF8')),'hex') IS DISTINCT FROM r.post_sha THEN
   RAISE EXCEPTION 'CT84: postimagen CT no exacta' USING ERRCODE='55000'; END IF;
  ddl:=replace(ddl,p.prosrc,nuevo);
  EXECUTE ddl;
  IF (SELECT to_jsonb(z)-'prosrc' FROM pg_proc z WHERE oid=f) IS DISTINCT FROM antes
   OR pg_get_functiondef(f) IS DISTINCT FROM ddl OR obj_description(f,'pg_proc') IS DISTINCT FROM comentario
   OR (SELECT prosrc FROM pg_proc WHERE oid=f) IS DISTINCT FROM nuevo
   OR (SELECT jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype)
       FROM pg_depend d WHERE (d.classid='pg_proc'::regclass AND d.objid=f)
        OR (d.refclassid='pg_proc'::regclass AND d.refobjid=f)) IS DISTINCT FROM deps
  THEN RAISE EXCEPTION 'CT84: metadata/dependencias CT modificadas' USING ERRCODE='55000'; END IF;
 END LOOP;
 IF (SELECT jsonb_agg(jsonb_build_object('funcion',to_jsonb(z),'comentario',obj_description(z.oid,'pg_proc')) ORDER BY z.oid)
      FROM pg_proc z WHERE z.pronamespace='vec_contratacion_temporal'::regnamespace AND NOT z.oid=ANY(objetivos))
      IS DISTINCT FROM otras THEN
  RAISE EXCEPTION 'CT84: otra función CT modificada' USING ERRCODE='55000'; END IF;
END $ct84$;

-- Se cambia bajo el propietario Personal: no se otorga autoridad cruzada.
SET LOCAL ROLE vec_personal_propietario;
DO $personal84$
DECLARE
 propietario oid:='vec_personal_propietario'::regrole;
 f oid:=to_regprocedure('vec_personal.acreditar_alta_ejercicio_v2(text,text,text,bigint,text,text,text,text,text,text,bytea,bytea,bytea,bytea,bigint,bigint,bytea,bytea,bytea,bytea)');
 p pg_proc%ROWTYPE; antes jsonb; otras jsonb; deps jsonb; comentario text; ddl text; nuevo text;
 origen text:=$origen$    IF p_unidad_ref IS NULL OR p_unidad_ref !~ '^ref:[0-9a-f]{64}$'
       OR p_unidad_ref='ref:'||repeat('0',64) THEN$origen$;
 destino text:=$destino$    IF (
       (p_unidad_ref ~ '^ref:[0-9a-f]{64}$' AND p_unidad_ref<>'ref:'||repeat('0',64))
       OR (left(p_unidad_ref,7)='unidad:' AND octet_length(p_unidad_ref)>7
           AND vec_personal.referencia_alta_ejercicio_valida_v1(p_unidad_ref COLLATE "C"))
    ) IS NOT TRUE THEN$destino$;
BEGIN
 IF current_user<>'vec_personal_propietario' OR getdatabaseencoding()<>'UTF8'
  OR NOT EXISTS (SELECT 1 FROM pg_namespace WHERE nspname='vec_personal' AND nspowner=propietario)
  OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE oid=propietario AND NOT rolcanlogin AND NOT rolinherit
   AND NOT rolsuper AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls)
 THEN RAISE EXCEPTION 'CT84: propietario Personal incompatible' USING ERRCODE='55000'; END IF;
 -- Reutilización exacta del validador opaco propietario, sin ampliarlo.
 IF (SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') FROM pg_proc
     WHERE oid=to_regprocedure('vec_personal.referencia_alta_ejercicio_valida_v1(text)'))
     IS DISTINCT FROM '385e11db3631fcb6f1ab0aa7005f31eccc45217d51147f9b0c5813f1eaa06e15' THEN
  RAISE EXCEPTION 'CT84: validador Personal incompatible' USING ERRCODE='55000'; END IF;
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=f;
 ddl:=pg_get_functiondef(f);
 IF p.proowner<>propietario OR NOT p.prosecdef
  OR encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM 'd7973e804e74528b752ad73c2720198e783dddddc0da82615ecaa2a05eca6c27'
  OR encode(sha256(convert_to(ddl,'UTF8')),'hex') IS DISTINCT FROM '06944a69bbf0fa055ff78f164070608d4d70f57266a48c7b5d86b5e70f6d0943'
  OR p.proacl IS DISTINCT FROM ARRAY[makeaclitem(propietario,propietario,'EXECUTE',false),
   makeaclitem('vec_personal_ejecutor'::regrole,propietario,'EXECUTE',false),
   makeaclitem('vec_contratacion_temporal_propietario'::regrole,propietario,'EXECUTE',false)]
  OR obj_description(f,'pg_proc') IS DISTINCT FROM 'Personal000005:lector_incorporacion:v2:ambitos_org_unidad'
  OR (SELECT count(*) FROM pg_proc WHERE pronamespace=p.pronamespace AND proname=p.proname)<>1
 THEN RAISE EXCEPTION 'CT84: preimagen/contrato/ACL Personal no exactos' USING ERRCODE='55000'; END IF;
 antes:=to_jsonb(p)-'prosrc'; comentario:=obj_description(f,'pg_proc');
 SELECT jsonb_agg(jsonb_build_object('funcion',to_jsonb(z),'comentario',obj_description(z.oid,'pg_proc')) ORDER BY z.oid)
  INTO otras FROM pg_proc z WHERE z.pronamespace=p.pronamespace AND z.oid<>f;
 SELECT jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype)
  INTO deps FROM pg_depend d WHERE (d.classid='pg_proc'::regclass AND d.objid=f)
   OR (d.refclassid='pg_proc'::regclass AND d.refobjid=f);
 IF (length(p.prosrc)-length(replace(p.prosrc,origen,'')))/length(origen)<>1
  OR (length(ddl)-length(replace(ddl,p.prosrc,'')))/length(p.prosrc)<>1 THEN
  RAISE EXCEPTION 'CT84: sustitución Personal ambigua' USING ERRCODE='55000'; END IF;
 nuevo:=replace(p.prosrc,origen,destino);
 IF encode(sha256(convert_to(nuevo,'UTF8')),'hex') IS DISTINCT FROM '38e12724f1addf6dbec363394f941641369bfb4dd8906eeb3f469eb243266073' THEN
  RAISE EXCEPTION 'CT84: postimagen Personal no exacta' USING ERRCODE='55000'; END IF;
 ddl:=replace(ddl,p.prosrc,nuevo);
 EXECUTE ddl;
 IF (SELECT to_jsonb(z)-'prosrc' FROM pg_proc z WHERE oid=f) IS DISTINCT FROM antes
  OR pg_get_functiondef(f) IS DISTINCT FROM ddl OR obj_description(f,'pg_proc') IS DISTINCT FROM comentario
  OR (SELECT prosrc FROM pg_proc WHERE oid=f) IS DISTINCT FROM nuevo
  OR (SELECT jsonb_agg(jsonb_build_object('funcion',to_jsonb(z),'comentario',obj_description(z.oid,'pg_proc')) ORDER BY z.oid)
      FROM pg_proc z WHERE z.pronamespace=p.pronamespace AND z.oid<>f) IS DISTINCT FROM otras
  OR (SELECT jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype)
      FROM pg_depend d WHERE (d.classid='pg_proc'::regclass AND d.objid=f)
       OR (d.refclassid='pg_proc'::regclass AND d.refobjid=f)) IS DISTINCT FROM deps THEN
  RAISE EXCEPTION 'CT84: postimagen/metadata/dependencias Personal modificadas' USING ERRCODE='55000'; END IF;
END $personal84$;
COMMIT;
