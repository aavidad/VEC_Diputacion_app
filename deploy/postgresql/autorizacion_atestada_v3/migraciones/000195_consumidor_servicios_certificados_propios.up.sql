\set ON_ERROR_STOP on
-- AD195: consumidor nominal propio de personal.servicios_certificados.consultar
-- (audiencia vec_personal.servicios_certificados.v1) para la fachada Personal36.
-- Solo autoservicio: la persona consulta los servicios de su propio empleado
-- canónico; Personal36 revalida esa pertenencia antes de consumir. No presta
-- ficha propia, historia, RRHH ni exportación, y no concede permisos.
-- Preimágenes medidas en la copia fría H10-30 tras AD194/IS16/CA36/AUT47/AD193
-- (núcleo POST193 y CHECK de audiencias POST185). Las huellas son literales;
-- un núcleo distinto detiene la migración antes de modificar nada.
-- Una sola vez; sin DOWN. Orden: AD193 -> AD195 -> Personal36.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000195',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
DO $pre$
DECLARE rol text;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR current_user<>'vec_autorizacion_atestada_v3_propietario'
 OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_servicios_certificados_propios_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.resolver_origen_consumo_v1(text,text,text)') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.capacidad_cruda_prevalida(bytea)') IS NULL
 OR NOT EXISTS(SELECT 1 FROM pg_attribute WHERE attrelid=to_regclass('vec_autorizacion_atestada_v3.consumo_decision_v3')
   AND attname='transaccion_origen' AND NOT attisdropped)
 THEN RAISE EXCEPTION 'AD195: PARO clave=preimagen esperado=PG18_AD193_sin_AD195 actual=incompatible' USING ERRCODE='55000'; END IF;
 FOREACH rol IN ARRAY ARRAY['vec_personal_propietario','vec_personal_ejecutor'] LOOP
  IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=rol AND NOT rolcanlogin AND NOT rolsuper
   AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls) THEN
   RAISE EXCEPTION 'AD195: PARO clave=rol_% esperado=NOLOGIN_sin_atributos actual=incompatible',rol USING ERRCODE='55000'; END IF;
 END LOOP;
 IF EXISTS(SELECT 1 FROM pg_auth_members WHERE member='vec_personal_ejecutor'::regrole) THEN
  RAISE EXCEPTION 'AD195: PARO clave=herencia_ejecutor esperado=sin_grupos actual=con_grupos' USING ERRCODE='55000'; END IF;
END $pre$;

DO $nucleo$
DECLARE f oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 original text;fuente text;nueva text;actual text;meta jsonb;deps jsonb;compartidas jsonb;h text;
 marca text:=E'       )\n       OR c ->> ''suite'' <> ''VEC-AD-3-COSE-EDDSA-1''';
 runtime_marca text:=E'       OR NOT (\n           (p_perfil_mutacion IN (''usuarios_admin_listar'',''usuarios_admin_consultar'')';
 runtime_nueva text:=E'       OR NOT (\n           (p_perfil_mutacion IS NOT DISTINCT FROM ''servicios_certificados_propios''\n'
  ||E'            AND EXISTS(SELECT 1 FROM pg_catalog.pg_roles r WHERE r.rolname=session_user AND r.rolcanlogin AND r.rolinherit\n'
  ||E'             AND NOT(r.rolsuper OR r.rolcreatedb OR r.rolcreaterole OR r.rolreplication OR r.rolbypassrls))\n'
  ||E'            AND EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.member=session_user::regrole\n'
  ||E'             AND m.roleid=''vec_personal_ejecutor''::regrole\n'
  ||E'             AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)\n'
  ||E'            AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.member=''vec_personal_ejecutor''::regrole)\n'
  ||E'            AND (SELECT count(*) FROM pg_catalog.pg_auth_members m WHERE m.member=session_user::regrole)=1)\n'
  ||E'           OR (p_perfil_mutacion IN (''usuarios_admin_listar'',''usuarios_admin_consultar'')';
 excl_marca text:=$x$p_perfil_mutacion IS DISTINCT FROM 'bolsa_llamamiento' AND p_perfil_mutacion IS DISTINCT FROM 'publicacion_certificado_nominal' AND p_perfil_mutacion IS DISTINCT FROM 'publicacion_cargo_competencial' AND p_perfil_mutacion IS DISTINCT FROM 'capacidades_admin' AND p_perfil_mutacion IS DISTINCT FROM 'persona_denominacion_publicar' AND p_perfil_mutacion IS DISTINCT FROM 'persona_denominacion_leer' AND p_perfil_mutacion IS DISTINCT FROM 'usuarios_admin_listar' AND p_perfil_mutacion IS DISTINCT FROM 'usuarios_admin_consultar'$x$;
 excl_nueva text:=excl_marca||' AND p_perfil_mutacion IS DISTINCT FROM ''servicios_certificados_propios''';
 extension text:=$x$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'servicios_certificados_propios'
 AND c->>'operacion' IS NOT DISTINCT FROM 'personal.servicios_certificados.consultar'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.servicios_certificados.v1'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'personal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'servicios_certificados'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_servicios_para_certificados'
 AND d->>'recurso_ref' IS NOT NULL AND d->>'recurso_ref' ~ '^emp_[A-Za-z0-9_-]{22,128}$'
 AND c->>'efecto_ref' IS NOT DISTINCT FROM d->>'recurso_ref'
 AND c->>'huella_efecto_sha256' IS NOT DISTINCT FROM d->>'contexto_recurso_huella_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["cobertura","corte","evidencia","servicios"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa')
$x$;
BEGIN
 IF f IS NULL THEN RAISE EXCEPTION 'AD195: PARO clave=nucleo esperado=presente actual=ausente' USING ERRCODE='55000'; END IF;
 SELECT pg_get_functiondef(f),p.prosrc,to_jsonb(p)-'prosrc' INTO STRICT original,fuente,meta FROM pg_proc p WHERE p.oid=f;
 h:=encode(sha256(convert_to(original,'UTF8')),'hex');
 IF h IS DISTINCT FROM 'f581dbf9aa01d454caa6906ece774f97cf16cca9e9b8910d0eb348e23ef8c34b'
 THEN RAISE EXCEPTION 'AD195: PARO clave=nucleo_postAD193_def_SHA actual=% esperado=f581dbf9aa01d454caa6906ece774f97cf16cca9e9b8910d0eb348e23ef8c34b',h USING ERRCODE='55000'; END IF;
 h:=encode(sha256(convert_to(fuente,'UTF8')),'hex');
 IF h IS DISTINCT FROM 'bb21afce73af87d532574c99da4f3ea8cd0534edc4910f14018d9ee55eb2a79b'
 THEN RAISE EXCEPTION 'AD195: PARO clave=nucleo_postAD193_src_SHA actual=% esperado=bb21afce73af87d532574c99da4f3ea8cd0534edc4910f14018d9ee55eb2a79b',h USING ERRCODE='55000'; END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef
   AND p.provolatile='v' AND p.proparallel='u' AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s'])
 OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f)<>1
 OR NOT EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
   WHERE p.oid=f AND a.grantee=p.proowner AND a.grantor=p.proowner AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
 THEN RAISE EXCEPTION 'AD195: PARO clave=nucleo_metadatos esperado=propietario_privado actual=incompatible' USING ERRCODE='55000'; END IF;
 -- Cada marca aparece una sola vez en el cuerpo real; el perfil es nuevo.
 IF length(original)-length(replace(original,marca,''))<>length(marca)
 OR length(original)-length(replace(original,runtime_marca,''))<>length(runtime_marca)
 OR length(original)-length(replace(original,excl_marca,''))<>length(excl_marca)
 OR strpos(original,'servicios_certificados')<>0
 OR strpos(original,'resolver_origen_consumo_v1')=0
 OR strpos(original,'aud_v3_')=0
 THEN RAISE EXCEPTION 'AD195: PARO clave=marcas esperado=tres_marcas_unicas_sin_perfil actual=incompatible' USING ERRCODE='55000'; END IF;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
  INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
  INTO compartidas FROM pg_shdepend d WHERE d.classid='pg_proc'::regclass AND d.objid=f
   AND d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database());
 nueva:=replace(replace(replace(original,runtime_marca,runtime_nueva),excl_marca,excl_nueva),marca,extension||marca);
 EXECUTE nueva;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nueva
 OR replace(replace(replace(actual,extension||marca,marca),excl_nueva,excl_marca),runtime_nueva,runtime_marca) IS DISTINCT FROM original
 OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
 OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
     FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
 OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
     FROM pg_shdepend d WHERE d.classid='pg_proc'::regclass AND d.objid=f
      AND d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database())) IS DISTINCT FROM compartidas
 THEN RAISE EXCEPTION 'AD195: PARO clave=postimagen_nucleo esperado=extension_minima_metadatos_intactos actual=divergente' USING ERRCODE='55000'; END IF;
END $nucleo$;

LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencias$
DECLARE anterior text;nueva text;h text;
BEGIN
 SELECT pg_get_constraintdef(c.oid,false) INTO STRICT anterior FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
   AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 h:=encode(sha256(convert_to(anterior,'UTF8')),'hex');
 -- Preimagen medida tras AD185 (AD193 no toca este CHECK).
 IF h IS DISTINCT FROM '62ad0be0944790785a298a8387dca798b03e392d3208b5d59adf06793a6f4236'
 OR left(anterior,7)<>'CHECK (' OR right(anterior,1)<>')'
 OR strpos(anterior,'vec_personal.servicios_certificados.v1')<>0
 THEN RAISE EXCEPTION 'AD195: PARO clave=CHECK_audiencias_SHA actual=% esperado=62ad0be0944790785a298a8387dca798b03e392d3208b5d59adf06793a6f4236',h USING ERRCODE='55000'; END IF;
 nueva:='CHECK (('||substr(anterior,8,length(anterior)-8)||') OR audiencia_consumo = ''vec_personal.servicios_certificados.v1'')';
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||nueva;
 IF NOT EXISTS(SELECT 1 FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
   AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated
   AND strpos(pg_get_constraintdef(c.oid,false),'vec_personal.servicios_certificados.v1')>0)
 THEN RAISE EXCEPTION 'AD195: PARO clave=CHECK_postimagen esperado=audiencia_anadida actual=ausente' USING ERRCODE='55000'; END IF;
END $audiencias$;

-- Fachada de consumo: sólo la ejecuta el propietario de Personal desde su
-- lectura nominal. Comprueba el contrato cerrado antes de entrar en el núcleo,
-- que verifica firma, gobierno, revocación, origen técnico y vigencia.
CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_servicios_certificados_propios_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,
 consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; x record;
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR current_setting('TimeZone')<>'UTC'
    OR p_capacidad IS NULL OR vec_autorizacion_atestada_v3.capacidad_cruda_prevalida(p_capacidad) IS NOT TRUE
    OR p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 1 AND 524288
    OR p_contexto IS NULL OR octet_length(p_contexto) NOT BETWEEN 1 AND 262144
    OR p_motivo IS NULL OR octet_length(p_motivo) NOT BETWEEN 1 AND 65536
    OR p_persona_version IS NULL OR p_perfil_version IS NULL
    OR p_payload IS NULL OR octet_length(p_payload) NOT BETWEEN 1 AND 1048576
    OR p_sobre IS NULL OR octet_length(p_sobre) NOT BETWEEN 1 AND 1048576
    OR p_evidencia IS NULL OR octet_length(p_evidencia) NOT BETWEEN 1 AND 262144
    OR p_raiz IS NULL OR octet_length(p_raiz)<>44 THEN
  RAISE EXCEPTION 'AD195: consumo de servicios para certificados denegado' USING ERRCODE='42501';
 END IF;
 BEGIN
  c:=convert_from(p_capacidad,'UTF8')::jsonb;
  d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN
  RAISE EXCEPTION 'AD195: material de servicios para certificados inválido' USING ERRCODE='22023';
 END;
 IF jsonb_typeof(c) IS DISTINCT FROM 'object' OR jsonb_typeof(d) IS DISTINCT FROM 'object'
    OR d->'concedida' IS DISTINCT FROM 'true'::jsonb
    OR d->>'modulo_id' IS DISTINCT FROM 'personal'
    OR d->>'accion' IS DISTINCT FROM 'personal.servicios_certificados.consultar'
    OR c->>'operacion' IS DISTINCT FROM d->>'accion'
    OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_personal.servicios_certificados.v1'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'servicios_certificados'
    OR d->>'finalidad' IS DISTINCT FROM 'consultar_servicios_para_certificados'
    OR d->'campos_permitidos' IS DISTINCT FROM '["cobertura","corte","evidencia","servicios"]'::jsonb
    OR d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'interna_corporativa'
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
    OR d->>'recurso_ref' IS NULL OR d->>'recurso_ref' !~ '^emp_[A-Za-z0-9_-]{22,128}$'
    OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
    OR d->>'contexto_recurso_huella_sha256' IS NULL
    OR d->>'contexto_recurso_huella_sha256' !~ '^[0-9a-f]{64}$'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256' THEN
  RAISE EXCEPTION 'AD195: consumo de servicios para certificados denegado' USING ERRCODE='42501';
 END IF;
 SELECT * INTO STRICT x
 FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  'servicios_certificados_propios',p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE OR x.decision_ref IS DISTINCT FROM d->>'decision_ref'
    OR x.efecto_ref IS DISTINCT FROM c->>'efecto_ref'
    OR x.huella_efecto_sha256 IS DISTINCT FROM c->>'huella_efecto_sha256'
    OR x.consumo_huella_sha256 IS NULL OR x.consumo_huella_sha256 !~ '^[0-9a-f]{64}$'
    OR x.auditoria_ref IS DISTINCT FROM 'aud_v3_'||substr(x.consumo_huella_sha256,1,32)
    OR x.consumida_en IS NULL OR NOT isfinite(x.consumida_en) THEN
  RAISE EXCEPTION 'AD195: consumo de servicios para certificados divergente' USING ERRCODE='42501';
 END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,
  x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_servicios_certificados_propios_v3_atestada(
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_servicios_certificados_propios_v3_atestada(
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_personal_propietario;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_personal_propietario;
DO $acl$
DECLARE f oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_servicios_certificados_propios_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef
   AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s'])
 OR NOT has_function_privilege('vec_personal_propietario',f,'EXECUTE')
 OR has_function_privilege('vec_personal_ejecutor',f,'EXECUTE')
 OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f)<>2
 OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
   WHERE p.oid=f AND (a.grantee NOT IN(p.proowner,'vec_personal_propietario'::regrole) OR a.privilege_type<>'EXECUTE'
     OR a.is_grantable OR a.grantor<>p.proowner))
 THEN RAISE EXCEPTION 'AD195: PARO clave=ACL esperado=propietario_AD_y_propietario_Personal actual=ampliada' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
