\set ON_ERROR_STOP on
-- AD229: decisión nominal RRHH para incorporar inscripción y asociar acta.
-- Causal justo tras AD228. Preimagen medida sobre HZ12 + AD233 (#962) +
-- estructura B1 + AD234 + AD228: núcleo interno def ff64dcb9…, prosrc 81e298dd…;
-- CHECK de audiencias 40c514b3…. No cambia la familia ni los emisores de auditoría.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000229',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));

DO $pre$
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
 OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_revision_inscripcion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_incorporacion_inscripcion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 THEN RAISE EXCEPTION 'AD229: preimagen causal incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

-- Extiende el núcleo instalado exclusivamente en sus puntos de despacho,
-- runtime y contrato nominal. La reconstrucción inversa y metadatos deben
-- permanecer idénticos; otra postimagen concurrente obliga a migración nueva.
DO $nucleo$
DECLARE f oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 original text; fuente text; nuevo text; actual text; meta jsonb; deps jsonb; compartidas jsonb;
 exclusion text:=$m$AND p_perfil_mutacion IS DISTINCT FROM 'gobierno_rol_nuevo' AND p_perfil_mutacion IS DISTINCT FROM 'revision_inscripcion'$m$;
 nueva_exclusion text:=$m$AND p_perfil_mutacion IS DISTINCT FROM 'gobierno_rol_nuevo' AND p_perfil_mutacion IS DISTINCT FROM 'revision_inscripcion' AND p_perfil_mutacion IS DISTINCT FROM 'incorporacion_inscripcion'$m$;
 runtime text:=$m$           OR (p_perfil_mutacion IS NOT DISTINCT FROM 'revision_inscripcion'
            AND session_user='vec_bolsa_llamamientos_desarrollo'$m$;
 rama_runtime text:=$r$           OR (p_perfil_mutacion IS NOT DISTINCT FROM 'incorporacion_inscripcion'
            AND session_user='vec_bolsa_llamamientos_desarrollo'
            AND current_setting('role')='none'
            AND EXISTS(SELECT 1 FROM pg_roles l JOIN pg_auth_members m ON m.member=l.oid
              WHERE l.rolname=session_user AND l.rolcanlogin AND l.rolinherit
              AND NOT(l.rolsuper OR l.rolcreatedb OR l.rolcreaterole OR l.rolreplication OR l.rolbypassrls)
              AND l.rolconfig IS NULL AND m.roleid='vec_bolsa_llamamientos_ejecutor'::regrole
              AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option
              AND (SELECT count(*) FROM pg_auth_members x WHERE x.member=l.oid)=1
              AND NOT EXISTS(SELECT 1 FROM pg_auth_members x WHERE x.member='vec_bolsa_llamamientos_ejecutor'::regrole)
              AND NOT EXISTS(SELECT 1 FROM pg_db_role_setting x WHERE x.setrole IN(l.oid,m.roleid))))
$r$;
 marca text:=$m$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'revision_inscripcion'$m$;
 rama text:=$r$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'incorporacion_inscripcion'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.inscripcion.incorporar.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'bolsa.inscripcion.rrhh.incorporar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'solicitud_inscripcion'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'incorporar_inscripcion'
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'recurso_ref' ~ '^solicitud_inscripcion_[0-9a-f]{64}$'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '[]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
$r$;
BEGIN
 IF f IS NULL THEN RAISE EXCEPTION 'AD229: núcleo interno ausente' USING ERRCODE='55000'; END IF;
 SELECT pg_get_functiondef(f),p.prosrc,to_jsonb(p)-'prosrc' INTO STRICT original,fuente,meta FROM pg_proc p WHERE p.oid=f;
 IF encode(sha256(convert_to(original,'UTF8')),'hex') IS DISTINCT FROM 'ff64dcb99df54e8985dca5425dd5df05930c8e8da00a8e92e145b0dc31ad3691'
 OR encode(sha256(convert_to(fuente,'UTF8')),'hex') IS DISTINCT FROM '81e298dd7ddf8ad8b379c8fae68eddd8579ff88073006bea977a726a4f305944'
 OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
   AND p.prosecdef AND p.provolatile='v' AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s'])
 OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f)<>1
 OR length(original)-length(replace(original,exclusion,''))<>length(exclusion)
 OR length(original)-length(replace(original,runtime,''))<>length(runtime)
 OR length(original)-length(replace(original,marca,''))<>length(marca)
 OR strpos(original,'incorporacion_inscripcion')<>0
 THEN RAISE EXCEPTION 'AD229: preimagen núcleo divergente' USING ERRCODE='55000'; END IF;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
 INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
 INTO compartidas FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database()) AND d.classid='pg_proc'::regclass AND d.objid=f;
 nuevo:=replace(original,exclusion,nueva_exclusion);
 nuevo:=replace(nuevo,runtime,rama_runtime||runtime);
 nuevo:=replace(nuevo,marca,rama||marca);
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo
 OR replace(replace(replace(actual,rama||marca,marca),rama_runtime||runtime,runtime),nueva_exclusion,exclusion) IS DISTINCT FROM original
 OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
 OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb) FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
 OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb) FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database()) AND d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM compartidas
 THEN RAISE EXCEPTION 'AD229: postimagen núcleo divergente' USING ERRCODE='55000'; END IF;
END $nucleo$;

LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencias$
DECLARE anterior text;nueva text;
BEGIN
 SELECT pg_get_constraintdef(c.oid,false) INTO STRICT anterior FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
 AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 IF encode(sha256(convert_to(anterior,'UTF8')),'hex') IS DISTINCT FROM '40c514b3833ba78cd0a0feb8b588a231b84c4dc3e790c38eddf7881b771060a0'
 OR left(anterior,7)<>'CHECK (' OR right(anterior,1)<>')'
 OR strpos(anterior,'vec_bolsa_llamamientos.inscripcion.presentar.v1')=0
 OR strpos(anterior,'vec_bolsa_llamamientos.inscripcion.revisar.v1')=0
 OR strpos(anterior,'vec_bolsa_llamamientos.inscripcion.incorporar.v1')<>0
 THEN RAISE EXCEPTION 'AD229: CHECK audiencias incompatible' USING ERRCODE='55000'; END IF;
 nueva:='CHECK (('||substr(anterior,8,length(anterior)-8)||') OR audiencia_consumo = ''vec_bolsa_llamamientos.inscripcion.incorporar.v1'')';
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||nueva;
END $audiencias$;

CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_incorporacion_inscripcion_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,
 auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET lock_timeout='2s' AS $f$
DECLARE c jsonb;d jsonb;x record;
BEGIN
 IF session_user<>'vec_bolsa_llamamientos_desarrollo' OR current_setting('role')<>'none'
 OR current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
 OR current_setting('TimeZone')<>'UTC' OR p_capacidad IS NULL OR p_decision IS NULL
 THEN RAISE EXCEPTION 'AD229: incorporación denegada' USING ERRCODE='42501'; END IF;
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb;d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'AD229: material inválido' USING ERRCODE='22023'; END;
 IF jsonb_typeof(c) IS DISTINCT FROM 'object' OR jsonb_typeof(d) IS DISTINCT FROM 'object'
 OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_bolsa_llamamientos.inscripcion.incorporar.v1'
 OR c->>'operacion' IS DISTINCT FROM 'bolsa.inscripcion.rrhh.incorporar'
 OR d->>'accion' IS DISTINCT FROM c->>'operacion' OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
 OR d->>'tipo_recurso' IS DISTINCT FROM 'solicitud_inscripcion'
 OR d->>'finalidad' IS DISTINCT FROM 'incorporar_inscripcion'
 OR d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'interna_corporativa'
 OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
 OR d->>'recurso_ref' !~ '^solicitud_inscripcion_[0-9a-f]{64}$'
 OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
 OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'AD229: incorporación sin ligadura' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  'incorporacion_inscripcion',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE OR x.decision_ref IS DISTINCT FROM d->>'decision_ref'
 OR x.efecto_ref IS DISTINCT FROM c->>'efecto_ref' OR x.huella_efecto_sha256 IS DISTINCT FROM c->>'huella_efecto_sha256'
 OR x.consumo_huella_sha256 !~ '^[0-9a-f]{64}$' OR x.auditoria_ref IS DISTINCT FROM 'aud_v3_'||substr(x.consumo_huella_sha256,1,32)
 OR x.consumida_en IS NULL OR NOT isfinite(x.consumida_en)
 THEN RAISE EXCEPTION 'AD229: incorporación divergente' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_incorporacion_inscripcion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_incorporacion_inscripcion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_propietario;
REVOKE GRANT OPTION FOR EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_incorporacion_inscripcion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM vec_bolsa_llamamientos_propietario;
COMMIT;
