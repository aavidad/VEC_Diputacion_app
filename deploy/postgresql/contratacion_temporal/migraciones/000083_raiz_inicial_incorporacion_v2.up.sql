\set ON_ERROR_STOP on
-- CT83: inicialización técnica tras el alta Personal REAL y ambos consumos.
-- Misma función CT75, 23 argumentos, OID/ACL; no tabla, permiso ni publicación.
-- Avance con historia: no reaplicar ni revertir sobre incorporaciones guardadas.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal.dependencias.alta_ejercicio.v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal.dependencias.incorporacion_ejercicio.v2',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:000083:raiz-inicial:v2',0));
SET LOCAL ROLE vec_contratacion_temporal_propietario;
DO $ct83$
DECLARE
 f oid:=to_regprocedure('vec_contratacion_temporal.registrar_incorporacion_ejercicio_v2(jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bigint,bigint,bytea,bytea,bytea,bytea,jsonb)');
 propietario oid:='vec_contratacion_temporal_propietario'::regrole;
 ejecutor oid:='vec_contratacion_temporal_ejecutor'::regrole;
 p pg_proc%ROWTYPE; antes jsonb; otras jsonb; deps jsonb; ddl text;
 origen text; destino text; i int;
 origenes text[]:=ARRAY[
 $o1$DECLARE
 pc bytea[]$o1$,
 $o2$ mc:=vec_contratacion_temporal.material_incorporacion_ejercicio_canonico_v2(material);$o2$,
 $o3$ SELECT z.* INTO raiz FROM vec_contratacion_temporal.seguimiento_raiz_v2 z WHERE z.seguimiento_ref=registrar_incorporacion_ejercicio_v2.seguimiento_ref FOR UPDATE;
 IF NOT FOUND THEN RAISE EXCEPTION 'CT75: raíz gobernada no disponible' USING ERRCODE='55000'; END IF;$o3$];
 destinos text[]:=ARRAY[
 $d1$DECLARE
 inicial jsonb; estado_cero jsonb; canon_cero bytea; canon_raiz bytea;
 raiz_encontrada boolean; version_raiz numeric;
 pc bytea[]$d1$,
 $d2$ -- Envoltorio interno transitorio: se conserva SOLAMENTE la evidencia original.
 -- No llega desde HTTP; tampoco otorga autoridad al llamador SQL.
 IF evidencia_orden ? 'esquema' THEN
  PERFORM vec_contratacion_temporal.seguimiento73_forma(evidencia_orden,ARRAY['esquema','orden','raiz_inicial']);
  IF evidencia_orden->>'esquema' IS DISTINCT FROM 'vec.ct.incorporacion.raiz-inicial.v1'
   OR pg_column_size(evidencia_orden)>16777216 THEN
   RAISE EXCEPTION 'CT83: envoltorio inválido' USING ERRCODE='22023'; END IF;
  inicial:=evidencia_orden->'raiz_inicial';
  PERFORM vec_contratacion_temporal.seguimiento73_forma(inicial,ARRAY['estado','version_expediente']);
  estado_cero:=inicial->'estado';
  PERFORM vec_contratacion_temporal.seguimiento73_escalar(inicial->'version_expediente','u64');
  version_raiz:=(inicial->>'version_expediente')::numeric;
  IF version_raiz NOT BETWEEN 1 AND 9007199254740991 OR pg_column_size(estado_cero)>131072 THEN
   RAISE EXCEPTION 'CT83: raíz inicial inválida' USING ERRCODE='22023'; END IF;
  evidencia_orden:=evidencia_orden->'orden';
 END IF;
 mc:=vec_contratacion_temporal.material_incorporacion_ejercicio_canonico_v2(material);$d2$,
 $d3$ SELECT z.* INTO raiz FROM vec_contratacion_temporal.seguimiento_raiz_v2 z WHERE z.seguimiento_ref=registrar_incorporacion_ejercicio_v2.seguimiento_ref FOR UPDATE;
 raiz_encontrada:=FOUND;
 IF inicial IS NOT NULL THEN
  -- Se llega aquí sólo tras consumo CT, lectura Personal nominal actual y lock
  -- de idempotencia/expediente. Cualquier fallo posterior revierte raíz y estado.
  IF esperada<>0 OR version_raiz IS DISTINCT FROM (material->>'VersionActualExpediente')::numeric
   OR estado_cero->>'referencia' IS DISTINCT FROM seguimiento_ref
   OR estado_cero->>'organizacion_ref' IS DISTINCT FROM preparacion->>'OrganizacionRef'
   OR estado_cero->>'expediente_ref' IS DISTINCT FROM s->>'expediente_ref'
   OR estado_cero->>'relacion_ref' IS DISTINCT FROM resultado_personal->>'relacion_ref'
   OR estado_cero->'version' IS DISTINCT FROM '0'::jsonb
   OR estado_cero->'periodo_previsto' IS DISTINCT FROM material#>'{Confirmacion,PeriodoIncorporacion}'
   OR (estado_cero->>'creado_en')::timestamptz IS DISTINCT FROM (registro_personal->>'registrado_en')::timestamptz
   OR (estado_cero->>'actualizado_en')::timestamptz IS DISTINCT FROM (registro_personal->>'registrado_en')::timestamptz
   OR estado_cero->'actuaciones' IS DISTINCT FROM '[]'::jsonb
   OR coalesce(estado_cero->'periodos_resultantes','{}'::jsonb) NOT IN ('[]'::jsonb,'null'::jsonb)
   OR estado_cero ? 'cese_efectivo'
   OR NOT EXISTS (SELECT 1 FROM vec_contratacion_temporal.expediente_alta a
     WHERE a.expediente_ref=s->>'expediente_ref' AND a.organizacion_ref=preparacion->>'OrganizacionRef') THEN
   conflicto:=true; RAISE EXCEPTION 'CT83: raíz ajena al original' USING ERRCODE='P1102'; END IF;
  SELECT f.* INTO STRICT definicion FROM vec_contratacion_temporal.seguimiento_definicion_v2 f
   WHERE f.definicion_ref=estado_cero#>>'{definicion,referencia}'
    AND f.definicion_version=(estado_cero#>>'{definicion,version}')::numeric
    AND f.definicion_sha256=estado_cero#>>'{definicion,huella_sha256}' FOR SHARE;
  -- La publicación ya existe; no se recibe ni inserta desde la orden.
  pub:=definicion.publicacion_json; def:=vec_contratacion_temporal.seguimiento73_definicion(pub);
  IF definicion.definicion_canonica IS DISTINCT FROM vec_contratacion_temporal.seguimiento73_nodo(def,'publicacion') THEN
   RAISE EXCEPTION 'CT83: publicación no reproducible' USING ERRCODE='55000'; END IF;
  canon_cero:=vec_contratacion_temporal.estado_seguimiento_canonico_v1(pub,estado_cero);
  raiz_json:=jsonb_build_object('referencia',estado_cero->'referencia','organizacion_ref',estado_cero->'organizacion_ref',
   'expediente_ref',estado_cero->'expediente_ref','relacion_ref',estado_cero->'relacion_ref',
   'definicion',estado_cero->'definicion','estado_actual',def->'estado_inicial',
   'periodo_previsto',estado_cero->'periodo_previsto','creado_en',estado_cero->'creado_en');
  canon_raiz:=vec_contratacion_temporal.seguimiento73_nodo(raiz_json,'raiz');
  IF raiz_encontrada THEN
   -- Un reintento coteja estado0; jamás lo sustituye ni crea otra raíz.
   SELECT e.* INTO STRICT estado_a FROM vec_contratacion_temporal.seguimiento_estado_v2 e
    WHERE e.seguimiento_ref=raiz.seguimiento_ref AND e.version_seguimiento=0 FOR SHARE;
   IF raiz.raiz_canonica IS DISTINCT FROM canon_raiz OR raiz.version_expediente_observada IS DISTINCT FROM version_raiz
    OR estado_a.estado_canonico IS DISTINCT FROM canon_cero
    OR vec_contratacion_temporal.estado_seguimiento_canonico_v1(pub,estado_a.estado_json) IS DISTINCT FROM canon_cero THEN
    conflicto:=true; RAISE EXCEPTION 'CT83: raíz inicial en conflicto' USING ERRCODE='P1102'; END IF;
  ELSE
   IF version_actual IS DISTINCT FROM version_raiz THEN
    conflicto:=true; RAISE EXCEPTION 'CT83: expediente avanzado' USING ERRCODE='P1102'; END IF;
   INSERT INTO vec_contratacion_temporal.seguimiento_raiz_v2
    (seguimiento_ref,organizacion_ref,expediente_ref,relacion_ref,version_expediente_observada,
     definicion_ref,definicion_version,definicion_sha256,raiz_canonica,raiz_sha256,version_inicial)
    VALUES (registrar_incorporacion_ejercicio_v2.seguimiento_ref,preparacion->>'OrganizacionRef',s->>'expediente_ref',resultado_personal->>'relacion_ref',version_raiz,
     definicion.definicion_ref,definicion.definicion_version,definicion.definicion_sha256,canon_raiz,encode(sha256(canon_raiz),'hex'),0)
    RETURNING * INTO raiz;
   INSERT INTO vec_contratacion_temporal.seguimiento_estado_v2
    (seguimiento_ref,organizacion_ref,expediente_ref,relacion_ref,definicion_ref,definicion_version,definicion_sha256,
     raiz_sha256,version_seguimiento,version_anterior,estado_anterior_sha256,estado_json,estado_canonico,estado_sha256)
    VALUES (raiz.seguimiento_ref,raiz.organizacion_ref,raiz.expediente_ref,raiz.relacion_ref,
     raiz.definicion_ref,raiz.definicion_version,raiz.definicion_sha256,raiz.raiz_sha256,
     0,NULL,NULL,estado_cero,canon_cero,encode(sha256(canon_cero),'hex'));
  END IF;
 ELSIF NOT raiz_encontrada THEN
  RAISE EXCEPTION 'CT75: raíz gobernada no disponible' USING ERRCODE='55000';
 END IF;$d3$];
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario' OR getdatabaseencoding()<>'UTF8' OR f IS NULL THEN
  RAISE EXCEPTION 'CT83: propietario o función ausentes' USING ERRCODE='55000'; END IF;
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=f;
 IF octet_length(convert_to(p.prosrc,'UTF8'))<>23398
  OR encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM 'b4c9c6a4eb59a42974fc0cf085b4a77ca7e1427880a257177761ed9d13072a39'
  OR p.proowner<>propietario OR NOT p.prosecdef OR p.pronargs<>23 OR p.pronargdefaults<>0
  OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog','row_security=on','lock_timeout=2s','statement_timeout=5s']::text[]
  OR obj_description(f,'pg_proc') IS DISTINCT FROM 'CT79:cotejo-personal11:sobre-CT76'
  OR (SELECT count(*) FROM pg_proc z WHERE z.pronamespace=p.pronamespace AND z.proname=p.proname)<>1
  OR (SELECT array_agg(a::text ORDER BY a::text) FROM unnest(coalesce(p.proacl,acldefault('f',propietario))) a)
    IS DISTINCT FROM (SELECT array_agg(a::text ORDER BY a::text) FROM unnest(ARRAY[
     makeaclitem(propietario,propietario,'EXECUTE',false),makeaclitem(ejecutor,propietario,'EXECUTE',false)]) a) THEN
  RAISE EXCEPTION 'CT83: preimagen/contrato/ACL no exactos' USING ERRCODE='55000'; END IF;
 antes:=to_jsonb(p)-'prosrc';
 SELECT jsonb_agg(to_jsonb(z) ORDER BY z.oid) INTO otras FROM pg_proc z WHERE z.pronamespace=p.pronamespace AND z.oid<>f;
 SELECT jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype)
  INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 ddl:=pg_get_functiondef(f);
 FOR i IN 1..3 LOOP
  origen:=origenes[i]; destino:=destinos[i];
  IF (length(p.prosrc)-length(replace(p.prosrc,origen,'')))/length(origen)<>1
   OR (length(ddl)-length(replace(ddl,origen,'')))/length(origen)<>1 THEN
   RAISE EXCEPTION 'CT83: sustitución ambigua' USING ERRCODE='55000'; END IF;
  ddl:=replace(ddl,origen,destino);
 END LOOP;
 EXECUTE ddl;
 IF (SELECT to_jsonb(z)-'prosrc' FROM pg_proc z WHERE oid=f) IS DISTINCT FROM antes
  OR pg_get_functiondef(f) IS DISTINCT FROM ddl
  OR (SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') FROM pg_proc WHERE oid=f) IS DISTINCT FROM '44dab34a4359acc0982096b3d26e9583570c69b60fc17c68030afe53ea0686f7'
  OR (SELECT jsonb_agg(to_jsonb(z) ORDER BY z.oid) FROM pg_proc z WHERE z.pronamespace=p.pronamespace AND z.oid<>f) IS DISTINCT FROM otras
  OR (SELECT jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype)
   FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps THEN
  RAISE EXCEPTION 'CT83: postimagen/metadata/dependencias no exactas' USING ERRCODE='55000'; END IF;
 EXECUTE format('COMMENT ON FUNCTION %s IS %L',f::regprocedure,'CT83:raiz-inicial-atomica:sobre-CT79');
END $ct83$;
COMMIT;
