\set ON_ERROR_STOP on
-- AD3-91: lectura nominal de Auditoría común por propietarios CT y Bolsa.
-- Requiere AD3-90. El perfil CT usa el consumidor lector RRHH (sesión
-- consultor nominal); Bolsa usa su consumidor V3 con sesión Bolsa exclusiva.
-- Ningún perfil concede escritura ni consulta global. Sin DOWN con historia.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000091',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));

DO $pre$
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_historial_propio_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_consulta_rrhh_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_auditoria_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_auditoria_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_contratacion_temporal_propietario' AND NOT rolcanlogin AND NOT rolbypassrls)
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_bolsa_llamamientos_propietario' AND NOT rolcanlogin AND NOT rolbypassrls)
 THEN RAISE EXCEPTION 'AD3-91: preimagen incompatible (AD3-90 requerida)' USING ERRCODE='55000'; END IF;
END $pre$;

-- Consumidor lector CT: conserva la guarda de sesión consultor RRHH de AD3-3.
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
    OR length(original)-length(replace(original,marca,''))<>length(marca)
    OR strpos(original,'vec_contratacion_temporal_consultor_rrhh')=0
    OR strpos(original,'auditoria_ct')<>0
 THEN RAISE EXCEPTION 'AD3-91: lector CT incompatible' USING ERRCODE='55000'; END IF;
 nuevo:=replace(original,marca,extension||marca);
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo OR replace(actual,extension||marca,marca) IS DISTINCT FROM original
    OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
    OR (SELECT proacl FROM pg_proc WHERE oid=f) IS DISTINCT FROM acl
    OR (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM propietario
    OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM config
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS DISTINCT FROM definidora
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
        FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
 THEN RAISE EXCEPTION 'AD3-91: lector CT alterado fuera de contrato' USING ERRCODE='55000'; END IF;
END $lector_ct$;

-- Bolsa: añade un perfil lector en el bloque de sesión Bolsa y lo excluye del
-- bloque CT ejecutor, sin ampliar ninguna capacidad preexistente.
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
    OR length(original)-length(replace(original,marca,''))<>length(marca)
    OR length(original)-length(replace(original,excl,''))<>length(excl)
    OR length(original)-length(replace(original,runtime,''))<>length(runtime)
    OR strpos(original,'bolsa.historial_propio.consultar')=0
    OR strpos(original,'consulta_auditoria_bolsa')<>0
 THEN RAISE EXCEPTION 'AD3-91: núcleo Bolsa incompatible' USING ERRCODE='55000'; END IF;
 nuevo:=replace(original,excl,excl_nuevo);
 nuevo:=replace(nuevo,runtime,runtime_nuevo);
 nuevo:=replace(nuevo,marca,extension||marca);
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo
    OR replace(replace(replace(actual,extension||marca,marca),runtime_nuevo,runtime),excl_nuevo,excl) IS DISTINCT FROM original
    OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
    OR (SELECT proacl FROM pg_proc WHERE oid=f) IS DISTINCT FROM acl
    OR (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM propietario
    OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM config
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS DISTINCT FROM definidora
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
        FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
 THEN RAISE EXCEPTION 'AD3-91: núcleo Bolsa alterado fuera de contrato' USING ERRCODE='55000'; END IF;
END $nucleo_bolsa$;

LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencia$
DECLARE d text; a text:='vec_auditoria.consulta_rrhh.v1';
BEGIN
 SELECT regexp_replace(pg_get_constraintdef(c.oid,true),'\s+',' ','g') INTO STRICT d
 FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
 AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 IF strpos(d,'CHECK (audiencia_consumo = ANY (ARRAY[')<>1 OR right(d,3)<>']))'
    OR strpos(d,'vec.bolsa.mi-bolsa.historial.v1')=0 OR strpos(d,quote_literal(a))<>0
 THEN RAISE EXCEPTION 'AD3-91: audiencias incompatibles' USING ERRCODE='55000'; END IF;
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '
  ||left(d,length(d)-3)||', '||quote_literal(a)||'::text]))';
END $audiencia$;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_auditoria_ct_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; x record;
BEGIN
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb;d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'AD3-91: material CT inválido' USING ERRCODE='22023'; END;
 IF c->>'audiencia_consumo' IS DISTINCT FROM 'vec_auditoria.consulta_rrhh.v1'
    OR c->>'operacion' IS DISTINCT FROM 'vec.auditoria.consultar'
    OR d->>'accion' IS DISTINCT FROM c->>'operacion'
    OR d->>'modulo_id' IS DISTINCT FROM 'auditoria'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'historial_auditoria'
    OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
    OR d->>'finalidad' !~ '^[a-z][a-z0-9_]{0,127}$'
    OR d->'campos_permitidos' IS DISTINCT FROM '["accion","actor_ref","antes","antes_sha256","datos_disponibles","despues","despues_sha256","expediente_ref","fuente","id","modulo_id","motivo","ocurrido_en","recibo_ref","resultado"]'::jsonb
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'AD3-91: material CT rechazado' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_consulta_rrhh_v3_interna(
  'auditoria_ct',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'AD3-91: consulta CT requiere consumo nuevo' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_auditoria_bolsa_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; x record;
BEGIN
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb;d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'AD3-91: material Bolsa inválido' USING ERRCODE='22023'; END;
 IF c->>'audiencia_consumo' IS DISTINCT FROM 'vec_auditoria.consulta_rrhh.v1'
    OR c->>'operacion' IS DISTINCT FROM 'vec.auditoria.consultar'
    OR d->>'accion' IS DISTINCT FROM c->>'operacion'
    OR d->>'modulo_id' IS DISTINCT FROM 'auditoria'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'historial_auditoria'
    OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
    OR d->>'finalidad' !~ '^[a-z][a-z0-9_]{0,127}$'
    OR d->'campos_permitidos' IS DISTINCT FROM '["accion","actor_ref","antes","antes_sha256","datos_disponibles","despues","despues_sha256","expediente_ref","fuente","id","modulo_id","motivo","ocurrido_en","recibo_ref","resultado"]'::jsonb
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'AD3-91: material Bolsa rechazado' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  'consulta_auditoria_bolsa',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'AD3-91: consulta Bolsa requiere consumo nuevo' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;

DO $acl$
DECLARE f regprocedure; autorizado oid; x record;
BEGIN
 FOR f,autorizado IN SELECT * FROM (VALUES
  ('vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_auditoria_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,'vec_contratacion_temporal_propietario'::regrole::oid),
  ('vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_auditoria_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,'vec_bolsa_llamamientos_propietario'::regrole::oid)
 ) AS v(funcion,rol) LOOP
  FOR x IN SELECT DISTINCT a.grantee FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
   WHERE p.oid=f AND a.grantee<>p.proowner LOOP
   EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %s',f::text,CASE WHEN x.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(x.grantee)) END);
  END LOOP;
  EXECUTE format('GRANT EXECUTE ON FUNCTION %s TO %I',f::text,pg_get_userbyid(autorizado));
  IF (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario'::regrole
     OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS NOT TRUE
     OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM ARRAY['search_path=pg_catalog','lock_timeout=2s']
  THEN RAISE EXCEPTION 'AD3-91: fachada incompatible' USING ERRCODE='55000'; END IF;
  FOR x IN SELECT a.grantee,a.privilege_type,a.is_grantable,p.proowner FROM pg_proc p
   CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f LOOP
   IF (x.grantee<>x.proowner AND x.grantee IS DISTINCT FROM autorizado)
      OR x.privilege_type<>'EXECUTE' OR (x.grantee=autorizado AND x.is_grantable)
   THEN RAISE EXCEPTION 'AD3-91: ACL de fachada abierta' USING ERRCODE='55000'; END IF;
  END LOOP;
 END LOOP;
END $acl$;
COMMIT;
