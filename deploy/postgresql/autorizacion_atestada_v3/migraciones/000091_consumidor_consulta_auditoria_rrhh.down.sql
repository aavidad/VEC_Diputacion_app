\set ON_ERROR_STOP on
-- AD3-91 DOWN: solo para un entorno sin historia de esta audiencia y sin
-- consumidores CT132/Bolsa48. No ejecutar sobre las bases conservadas.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000091',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;

DO $pre$
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_auditoria_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_auditoria_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.clave_capacidad_version WHERE audiencia_consumo='vec_auditoria.consulta_rrhh.v1')
    OR EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
      WHERE (n.nspname='vec_contratacion_temporal' AND p.proname='consultar_auditoria_ct_atestada_v1')
         OR (n.nspname='vec_bolsa_llamamientos' AND p.proname='consultar_auditoria_participacion_v1'))
 THEN RAISE EXCEPTION 'AD3-91 DOWN: historia o consumidores presentes' USING ERRCODE='55000'; END IF;
END $pre$;

DROP FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_auditoria_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_auditoria_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);

DO $audiencia$
DECLARE d text; resto text; a text:='vec_auditoria.consulta_rrhh.v1';
BEGIN
 SELECT regexp_replace(pg_get_constraintdef(c.oid,true),'\s+',' ','g') INTO STRICT d
 FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
 AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 IF strpos(d,'CHECK (audiencia_consumo = ANY (ARRAY[')<>1 OR right(d,3)<>']))'
    OR length(d)-length(replace(d,quote_literal(a),''))<>length(quote_literal(a))
 THEN RAISE EXCEPTION 'AD3-91 DOWN: audiencia incompatible' USING ERRCODE='55000'; END IF;
 resto:=replace(d,', '||quote_literal(a)||'::text','');
 IF resto=d OR strpos(resto,quote_literal(a))<>0 THEN RAISE EXCEPTION 'AD3-91 DOWN: audiencia no localizada' USING ERRCODE='55000'; END IF;
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||resto;
END $audiencia$;

DO $nucleo_bolsa$
DECLARE
 f oid:='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 original text; nuevo text; actual text; meta jsonb; deps jsonb; acl aclitem[]; propietario oid; config text[]; definidora boolean;
 marca text:=E'       )\n       OR c ->> ''suite'' <> ''VEC-AD-3-COSE-EDDSA-1''';
 excl text:=E'               p_perfil_mutacion IS DISTINCT FROM ''bolsa_llamamiento''\n';
 excl_nuevo text:=excl||E'               AND p_perfil_mutacion IS DISTINCT FROM ''consulta_auditoria_bolsa''\n';
 runtime text:=E'(p_perfil_mutacion IS NOT DISTINCT FROM ''bolsa_llamamiento''\n';
 runtime_nuevo text:=runtime||E'               OR p_perfil_mutacion IS NOT DISTINCT FROM ''consulta_auditoria_bolsa''\n';
 extension text:=$x$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_auditoria_bolsa'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_auditoria.consulta_rrhh.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'vec.auditoria.consultar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'auditoria'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'historial_auditoria'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->>'finalidad' ~ '^[a-z][a-z0-9_]{0,127}$'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["accion","actor_ref","antes","antes_sha256","datos_disponibles","despues","despues_sha256","expediente_ref","fuente","id","modulo_id","motivo","ocurrido_en","recibo_ref","resultado"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
$x$;
BEGIN
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc',p.proacl,p.proowner,p.proconfig,p.prosecdef
 INTO STRICT original,meta,acl,propietario,config,definidora FROM pg_proc p WHERE p.oid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
 INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 IF propietario<>'vec_autorizacion_atestada_v3_propietario'::regrole OR NOT definidora
    OR config IS DISTINCT FROM ARRAY['search_path=pg_catalog','lock_timeout=2s']
    OR length(original)-length(replace(original,extension,''))<>length(extension)
    OR length(original)-length(replace(original,excl_nuevo,''))<>length(excl_nuevo)
    OR length(original)-length(replace(original,runtime_nuevo,''))<>length(runtime_nuevo)
    OR length(original)-length(replace(original,marca,''))<>length(marca)
 THEN RAISE EXCEPTION 'AD3-91 DOWN: núcleo Bolsa incompatible' USING ERRCODE='55000'; END IF;
 nuevo:=replace(original,extension,'');
 nuevo:=replace(nuevo,runtime_nuevo,runtime);
 nuevo:=replace(nuevo,excl_nuevo,excl);
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo OR strpos(actual,'consulta_auditoria_bolsa')<>0
    OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
    OR (SELECT proacl FROM pg_proc WHERE oid=f) IS DISTINCT FROM acl
    OR (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM propietario
    OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM config
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS DISTINCT FROM definidora
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
        FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
 THEN RAISE EXCEPTION 'AD3-91 DOWN: núcleo Bolsa alterado fuera de contrato' USING ERRCODE='55000'; END IF;
END $nucleo_bolsa$;

DO $lector_ct$
DECLARE
 f oid:='vec_autorizacion_atestada_v3.consumir_consulta_rrhh_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 original text; nuevo text; actual text; meta jsonb; deps jsonb; acl aclitem[]; propietario oid; config text[]; definidora boolean;
 marca text:=E'       )\n       OR c ->> ''suite'' <> ''VEC-AD-3-COSE-EDDSA-1''';
 extension text:=$x$           OR (
               p_perfil_consulta = 'auditoria_ct'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM 'vec_auditoria.consulta_rrhh.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM 'vec.auditoria.consultar'
               AND d ->> 'accion' IS NOT DISTINCT FROM c ->> 'operacion'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'auditoria'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'historial_auditoria'
               AND d ->> 'recurso_ref' IS NOT DISTINCT FROM c ->> 'efecto_ref'
               AND d ->> 'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c ->> 'huella_efecto_sha256'
               AND d ->> 'finalidad' ~ '^[a-z][a-z0-9_]{0,127}$'
               AND d -> 'campos_permitidos' IS NOT DISTINCT FROM '["accion","actor_ref","antes","antes_sha256","datos_disponibles","despues","despues_sha256","expediente_ref","fuente","id","modulo_id","motivo","ocurrido_en","recibo_ref","resultado"]'::jsonb
               AND d -> 'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb
           )
$x$;
BEGIN
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc',p.proacl,p.proowner,p.proconfig,p.prosecdef
 INTO STRICT original,meta,acl,propietario,config,definidora FROM pg_proc p WHERE p.oid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
 INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 IF propietario<>'vec_autorizacion_atestada_v3_propietario'::regrole OR NOT definidora
    OR config IS DISTINCT FROM ARRAY['search_path=pg_catalog','lock_timeout=2s']
    OR length(original)-length(replace(original,extension,''))<>length(extension)
    OR length(original)-length(replace(original,marca,''))<>length(marca)
 THEN RAISE EXCEPTION 'AD3-91 DOWN: lector CT incompatible' USING ERRCODE='55000'; END IF;
 nuevo:=replace(original,extension,'');
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo OR strpos(actual,'auditoria_ct')<>0
    OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
    OR (SELECT proacl FROM pg_proc WHERE oid=f) IS DISTINCT FROM acl
    OR (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM propietario
    OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM config
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS DISTINCT FROM definidora
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
        FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
 THEN RAISE EXCEPTION 'AD3-91 DOWN: lector CT alterado fuera de contrato' USING ERRCODE='55000'; END IF;
END $lector_ct$;
COMMIT;
