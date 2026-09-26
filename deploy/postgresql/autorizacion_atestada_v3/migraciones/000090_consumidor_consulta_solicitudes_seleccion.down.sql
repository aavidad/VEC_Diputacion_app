\set ON_ERROR_STOP on
-- AD3-90 DOWN: solo sin historia. Se rechaza si Selección 000001 sigue
-- instalada o si ya existe alguna clave de capacidad de sus audiencias; sin
-- historia deja el núcleo, las audiencias y las ACL como antes de AD3-90.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000090',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;

DO $pre$
BEGIN
 IF to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_solicitudes_seleccion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 THEN RAISE EXCEPTION 'AD3-90 DOWN: AD3-90 no instalada' USING ERRCODE='55000'; END IF;
 IF to_regnamespace('vec_seleccion') IS NOT NULL
 THEN RAISE EXCEPTION 'AD3-90 DOWN: Selección 000001 sigue instalada' USING ERRCODE='55000'; END IF;
 IF EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.clave_capacidad_version
            WHERE audiencia_consumo LIKE 'vec\_seleccion.solicitudes.%')
 THEN RAISE EXCEPTION 'AD3-90 DOWN: no admitido con historia de sus audiencias' USING ERRCODE='55000'; END IF;
END $pre$;

DROP FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_solicitudes_seleccion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);

DO $audiencias$
DECLARE d text; resto text; a text;
BEGIN
 SELECT regexp_replace(pg_get_constraintdef(c.oid,true),'\s+',' ','g') INTO STRICT d
 FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
 AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 resto:=d;
 FOREACH a IN ARRAY ARRAY['vec_seleccion.solicitudes.consultar.v1','vec_seleccion.solicitudes.consultar_detalle.v1'] LOOP
  IF length(resto)-length(replace(resto,', '||quote_literal(a)||'::text',''))<>length(', '||quote_literal(a)||'::text')
  THEN RAISE EXCEPTION 'AD3-90 DOWN: audiencia no localizada' USING ERRCODE='55000'; END IF;
  resto:=replace(resto,', '||quote_literal(a)||'::text','');
 END LOOP;
 IF strpos(resto,'vec_seleccion.solicitudes.')<>0 THEN
  RAISE EXCEPTION 'AD3-90 DOWN: audiencias incompatibles' USING ERRCODE='55000';
 END IF;
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version
  DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||resto;
END $audiencias$;

DO $nucleo$
DECLARE
 f oid:='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 original text; nuevo text; actual text; meta jsonb; deps jsonb; acl aclitem[];
 propietario oid; config text[]; definidora boolean;
 excl text:=E'               AND p_perfil_mutacion IS DISTINCT FROM ''consulta_solicitudes_seleccion''\n';
 guarda text:=$g$           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_solicitudes_seleccion'
               AND pg_catalog.pg_has_role(session_user, 'vec_seleccion_ejecutor', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(session_user, 'vec_seleccion_propietario', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(session_user, 'vec_seleccion_migrador', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(session_user, 'vec_contratacion_temporal_propietario', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(session_user, 'vec_contratacion_temporal_migrador', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(session_user, 'vec_autorizacion_atestada_v3_propietario', 'MEMBER')
           )
$g$;
 extension text:=$x$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_solicitudes_seleccion'
 AND ((c->>'operacion' IS NOT DISTINCT FROM 'seleccion.solicitudes.consultar'
       AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_seleccion.solicitudes.consultar.v1')
   OR (c->>'operacion' IS NOT DISTINCT FROM 'seleccion.solicitudes.consultar_detalle'
       AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_seleccion.solicitudes.consultar_detalle.v1'))
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'solicitudes_seleccion'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'seleccion'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'consulta_solicitudes_seleccion'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
$x$;
BEGIN
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc',p.proacl,p.proowner,p.proconfig,p.prosecdef
 INTO STRICT original,meta,acl,propietario,config,definidora FROM pg_proc p WHERE p.oid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
 INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 -- Cada pieza se localiza sola, exactamente una vez: otras extensiones
 -- instaladas después pueden haberse insertado entre ellas.
 IF length(original)-length(replace(original,extension,''))<>length(extension)
    OR length(original)-length(replace(original,guarda,''))<>length(guarda)
    OR length(original)-length(replace(original,excl,''))<>length(excl)
 THEN RAISE EXCEPTION 'AD3-90 DOWN: extensión no localizada en el núcleo' USING ERRCODE='55000'; END IF;
 nuevo:=replace(replace(replace(original,extension,''),guarda,''),excl,'');
 IF strpos(nuevo,'consulta_solicitudes_seleccion')<>0 OR strpos(nuevo,'vec_seleccion.solicitudes.')<>0 THEN
  RAISE EXCEPTION 'AD3-90 DOWN: restos de la extensión en el núcleo' USING ERRCODE='55000';
 END IF;
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo
    OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
    OR (SELECT proacl FROM pg_proc WHERE oid=f) IS DISTINCT FROM acl
    OR (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM propietario
    OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM config
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS DISTINCT FROM definidora
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb) FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
 THEN RAISE EXCEPTION 'AD3-90 DOWN: núcleo alterado fuera del contrato' USING ERRCODE='55000'; END IF;
END $nucleo$;
COMMIT;
