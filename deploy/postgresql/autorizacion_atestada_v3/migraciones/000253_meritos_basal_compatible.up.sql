\set ON_ERROR_STOP on
-- AD253: instala los consumidores basales de Méritos ausentes en HZ24.
-- Conserva literalmente la función V3 instalada, sus consumidores CT/Documentos/I/L
-- y las migraciones históricas AD142/AD145. Sin cuentas ni concesiones personales.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000253',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_meritos:bootstrap:v1',0));
DO $pre$
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR to_regnamespace('vec_meritos') IS NOT NULL
 OR EXISTS (SELECT 1 FROM pg_roles WHERE rolname IN
   ('vec_meritos_propietario','vec_meritos_migrador','vec_meritos_ejecutor',
    'vec_meritos_externo','vec_meritos_interno','vec_meritos_consulta_propia_interno'))
 OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_operacion_meritos_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_consulta_hecho_propio_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 THEN RAISE EXCEPTION 'AD253: basal existente o privilegio insuficiente' USING ERRCODE='55000'; END IF;
END $pre$;

-- Mismo bootstrap nominal que RUM1/RUM2, instalado aquí porque AD253 precede RUM5.
CREATE ROLE vec_meritos_propietario NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_meritos_migrador NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_meritos_ejecutor NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_meritos_externo NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_meritos_interno NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_meritos_consulta_propia_interno NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_meritos_propietario TO vec_meritos_migrador WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;
CREATE SCHEMA vec_meritos AUTHORIZATION vec_meritos_propietario;
REVOKE ALL ON SCHEMA vec_meritos FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_meritos TO vec_meritos_ejecutor,vec_meritos_migrador,vec_meritos_consulta_propia_interno;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_meritos_propietario REVOKE ALL ON TABLES FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_meritos_propietario REVOKE EXECUTE ON FUNCTIONS FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_meritos_propietario REVOKE USAGE ON TYPES FROM PUBLIC;
DO $db$ BEGIN EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_meritos_ejecutor,vec_meritos_migrador,vec_meritos_consulta_propia_interno',current_database()); END $db$;

SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
DO $nucleo$
DECLARE
 f oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 original text; nuevo text; actual text; fuente text; meta jsonb; acl aclitem[];
 propietario oid; config text[]; definidora boolean;
 -- POST-AD256 sobre HZ24 restaurado el 13/10/2026; cotejo literal obligatorio.
 esperada_def constant text:='4cb910a878f7617233cc8718be1b443b93228861c4c6d1bcf9b2148cc7e08751';
 esperada_fuente constant text:='f81171daa94432a551daa37fb97d9457bfa7a3fadbd2efdaab0158dbe4277799';
 runtime_marca text:=$runtime_marca$           OR (
               p_perfil_mutacion IS DISTINCT FROM 'bolsa_llamamiento'$runtime_marca$;
 runtime_extension text:=$runtime_extension$           OR (
               p_perfil_mutacion IN ('meritos_hecho_propio_interno','meritos_hecho_rechazar')
               AND EXISTS (SELECT 1 FROM pg_roles r WHERE r.rolname=session_user AND r.rolcanlogin
                 AND NOT (r.rolsuper OR r.rolcreaterole OR r.rolcreatedb OR r.rolreplication OR r.rolbypassrls))
               AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole
                 AND m.roleid='vec_meritos_ejecutor'::regrole AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
               AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole
                 AND m.roleid='vec_meritos_interno'::regrole AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
               AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=2
               AND NOT EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member IN
                 ('vec_meritos_ejecutor'::regrole,'vec_meritos_interno'::regrole,'vec_meritos_externo'::regrole))
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'meritos_hecho_propio_externo'
               AND EXISTS (SELECT 1 FROM pg_roles r WHERE r.rolname=session_user AND r.rolcanlogin
                 AND NOT (r.rolsuper OR r.rolcreaterole OR r.rolcreatedb OR r.rolreplication OR r.rolbypassrls))
               AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole
                 AND m.roleid='vec_meritos_ejecutor'::regrole AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
               AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole
                 AND m.roleid='vec_meritos_externo'::regrole AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
               AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=2
               AND NOT EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member IN
                 ('vec_meritos_ejecutor'::regrole,'vec_meritos_interno'::regrole,'vec_meritos_externo'::regrole))
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'meritos_consulta_propia_interna'
               AND EXISTS (SELECT 1 FROM pg_roles r WHERE r.rolname=session_user AND r.rolcanlogin
                 AND NOT (r.rolsuper OR r.rolcreaterole OR r.rolcreatedb OR r.rolreplication OR r.rolbypassrls))
               AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole
                 AND m.roleid='vec_meritos_consulta_propia_interno'::regrole AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
               AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1
               AND NOT EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member='vec_meritos_consulta_propia_interno'::regrole)
           )
$runtime_extension$;
 excl_marca text:=$excl_marca$p_perfil_mutacion IS DISTINCT FROM 'bolsa_llamamiento' AND p_perfil_mutacion IS DISTINCT FROM 'consulta_rrhh_bolsa'$excl_marca$;
 excl_nueva text:=$excl_nueva$p_perfil_mutacion IS DISTINCT FROM 'bolsa_llamamiento' AND p_perfil_mutacion IS DISTINCT FROM 'meritos_hecho_propio_interno' AND p_perfil_mutacion IS DISTINCT FROM 'meritos_hecho_propio_externo' AND p_perfil_mutacion IS DISTINCT FROM 'meritos_hecho_rechazar' AND p_perfil_mutacion IS DISTINCT FROM 'meritos_consulta_propia_interna' AND p_perfil_mutacion IS DISTINCT FROM 'consulta_rrhh_bolsa'$excl_nueva$;
 sem_marca text:=$sem_marca$       ) IS NOT TRUE
       OR c ->> 'suite' <> 'VEC-AD-3-COSE-EDDSA-1'$sem_marca$;
 sem_extension text:=$sem_extension$           OR (
 p_perfil_mutacion IN ('meritos_hecho_propio_interno','meritos_hecho_propio_externo','meritos_hecho_rechazar')
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'meritos'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'hecho'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["declarante_ref","hecho","recibo","version"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '["auditar"]'::jsonb
 AND (
   (p_perfil_mutacion='meritos_hecho_propio_interno'
    AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa'
    AND ((c->>'operacion'='meritos.hecho.declarar' AND c->>'audiencia_consumo'='vec_meritos.hecho.declarar.v1' AND d->>'finalidad'='declaracion_hecho_propio')
      OR (c->>'operacion'='meritos.hecho.rectificar' AND c->>'audiencia_consumo'='vec_meritos.hecho.rectificar.v1' AND d->>'finalidad'='rectificacion_hecho_propio')))
   OR (p_perfil_mutacion='meritos_hecho_propio_externo'
    AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'externa_personal'
    AND c->>'operacion' IS NOT DISTINCT FROM 'meritos.hecho.declarar'
    AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_meritos.hecho.declarar.v1'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'declaracion_hecho_propio')
   OR (p_perfil_mutacion='meritos_hecho_rechazar'
    AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa'
    AND c->>'operacion' IS NOT DISTINCT FROM 'meritos.hecho.rechazar'
    AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_meritos.hecho.rechazar.v1'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'revision_hecho_merito')
 ))
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'meritos_consulta_propia_interna'
 AND c->>'operacion' IS NOT DISTINCT FROM 'meritos.hecho.consultar_propio'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_meritos.hecho.consultar_propio.v1'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'meritos'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'hecho'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'consulta_hecho_propio'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["hecho_actual","recibo_consulta"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '["auditar"]'::jsonb)
$sem_extension$;
BEGIN
 IF f IS NULL THEN RAISE EXCEPTION 'AD253: núcleo V3 ausente' USING ERRCODE='55000'; END IF;
 SELECT pg_get_functiondef(f),p.prosrc,to_jsonb(p)-'prosrc',p.proacl,p.proowner,p.proconfig,p.prosecdef
 INTO original,fuente,meta,acl,propietario,config,definidora FROM pg_proc p WHERE p.oid=f;
 IF propietario IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario'::regrole
 OR definidora IS NOT TRUE OR config IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
 OR encode(sha256(convert_to(original,'UTF8')),'hex') IS DISTINCT FROM esperada_def
 OR encode(sha256(convert_to(fuente,'UTF8')),'hex') IS DISTINCT FROM esperada_fuente
 OR length(original)-length(replace(original,runtime_marca,''))<>length(runtime_marca)
 OR length(original)-length(replace(original,excl_marca,''))<>length(excl_marca)
 OR length(original)-length(replace(original,sem_marca,''))<>length(sem_marca)
 THEN RAISE EXCEPTION 'AD253: preimagen literal V3 incompatible' USING ERRCODE='55000'; END IF;
 nuevo:=replace(original,runtime_marca,runtime_extension||runtime_marca);
 nuevo:=replace(nuevo,excl_marca,excl_nueva);
 nuevo:=replace(nuevo,sem_marca,sem_extension||sem_marca);
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo
 OR replace(replace(replace(actual,sem_extension||sem_marca,sem_marca),excl_nueva,excl_marca),runtime_extension||runtime_marca,runtime_marca) IS DISTINCT FROM original
 OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
 OR (SELECT proacl FROM pg_proc WHERE oid=f) IS DISTINCT FROM acl
 OR (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM propietario
 OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM config
 OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS DISTINCT FROM definidora
 THEN RAISE EXCEPTION 'AD253: V3 alterado fuera del contrato' USING ERRCODE='55000'; END IF;
END $nucleo$;

LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $aud$
DECLARE d text;
BEGIN
 SELECT pg_get_constraintdef(c.oid,true) INTO STRICT d FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
 AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 IF encode(sha256(convert_to(d,'UTF8')),'hex') IS DISTINCT FROM '10f045d3b0773cd21758240bb0ae060374295a8410e1865b73a9cadc4d57a509'
 OR right(d,4)<>'])))' OR strpos(d,'vec_meritos.hecho.')<>0
 THEN RAISE EXCEPTION 'AD253: catálogo de audiencias V3 incompatible' USING ERRCODE='55000'; END IF;
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '
 ||left(d,length(d)-4)||', ''vec_meritos.hecho.declarar.v1''::text, ''vec_meritos.hecho.rectificar.v1''::text, ''vec_meritos.hecho.rechazar.v1''::text, ''vec_meritos.hecho.consultar_propio.v1''::text])))';
END $aud$;

-- Los dos consumidores nominales reutilizan exactamente los contratos de
-- AD142 y AD145; sólo la declaración externa se abre con perfil segregado.
CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_operacion_meritos_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog, pg_temp SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; x record; perfil text;
BEGIN
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'AD3-142: material RUM inválido' USING ERRCODE='22023'; END;
 -- La declaración externa exige perfil y rol separados; no se concede a logins.
 -- No se consulta el agregado ni se publica una concesión; solo el núcleo común
 -- revalida y consume. El módulo aplica titularidad/CAS/historia en la misma TX.
 IF (
 ((c->>'operacion' IS NOT DISTINCT FROM 'meritos.hecho.declarar'
       AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_meritos.hecho.declarar.v1'
       AND d->>'finalidad' IS NOT DISTINCT FROM 'declaracion_hecho_propio')
   OR (c->>'operacion' IS NOT DISTINCT FROM 'meritos.hecho.rectificar'
       AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_meritos.hecho.rectificar.v1'
       AND d->>'finalidad' IS NOT DISTINCT FROM 'rectificacion_hecho_propio'))
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa'
 AND pg_has_role(session_user,'vec_meritos_interno','MEMBER')
 AND NOT pg_has_role(session_user,'vec_meritos_externo','MEMBER')
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'meritos'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'hecho'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["declarante_ref","hecho","recibo","version"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '["auditar"]'::jsonb
 ) IS TRUE THEN perfil:='meritos_hecho_propio_interno';
 ELSIF (
 c->>'operacion' IS NOT DISTINCT FROM 'meritos.hecho.rechazar'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_meritos.hecho.rechazar.v1'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'revision_hecho_merito'
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa'
 AND pg_has_role(session_user,'vec_meritos_interno','MEMBER')
 AND NOT pg_has_role(session_user,'vec_meritos_externo','MEMBER')
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'meritos'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'hecho'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["declarante_ref","hecho","recibo","version"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '["auditar"]'::jsonb
 ) IS TRUE THEN perfil:='meritos_hecho_rechazar';
 ELSIF (
 c->>'operacion' IS NOT DISTINCT FROM 'meritos.hecho.declarar'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_meritos.hecho.declarar.v1'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'declaracion_hecho_propio'
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'externa_personal'
 AND pg_has_role(session_user,'vec_meritos_externo','MEMBER')
 AND NOT pg_has_role(session_user,'vec_meritos_interno','MEMBER')
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'meritos'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'hecho'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["declarante_ref","hecho","recibo","version"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '["auditar"]'::jsonb
 ) IS TRUE THEN perfil:='meritos_hecho_propio_externo';
 ELSE RAISE EXCEPTION 'AD3-142: operación RUM denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  perfil,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'AD3-142: requiere consumo nuevo' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_operacion_meritos_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_meritos_propietario;
DO $acl$
DECLARE f regprocedure:='vec_autorizacion_atestada_v3.consumir_operacion_meritos_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 permitido oid:='vec_meritos_propietario'::regrole::oid; x record;
BEGIN
 -- También las ACL por defecto: ningún rol conserva acceso por haber sido
 -- destinatario predeterminado del propietario.
 FOR x IN SELECT DISTINCT a.grantee FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
  WHERE p.oid=f AND a.grantee<>p.proowner LOOP
  EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %s',f::text,CASE WHEN x.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(x.grantee)) END);
 END LOOP;
 EXECUTE format('GRANT EXECUTE ON FUNCTION %s TO vec_meritos_propietario',f::text);
 IF (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario'::regrole
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS NOT TRUE
    OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
    OR NOT has_schema_privilege('vec_meritos_propietario','vec_autorizacion_atestada_v3','USAGE')
 THEN RAISE EXCEPTION 'AD3-142: propietario o entorno de fachada incompatible' USING ERRCODE='55000'; END IF;
 FOR x IN SELECT a.grantee,a.privilege_type,a.is_grantable,p.proowner
  FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f LOOP
  IF (x.grantee<>x.proowner AND x.grantee IS DISTINCT FROM permitido) OR x.privilege_type<>'EXECUTE'
    OR (x.grantee=permitido AND x.is_grantable)
  THEN RAISE EXCEPTION 'AD3-142: ACL de fachada abierta' USING ERRCODE='55000'; END IF;
 END LOOP;
END $acl$;

CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_consulta_hecho_propio_v3_atestada(p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; x record;
BEGIN
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'AD3-145: material inválido' USING ERRCODE='22023'; END;
 IF d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'interna_corporativa'
 OR c->>'operacion' IS DISTINCT FROM 'meritos.hecho.consultar_propio'
 OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_meritos.hecho.consultar_propio.v1'
 OR d->>'accion' IS DISTINCT FROM c->>'operacion'
 OR d->>'modulo_id' IS DISTINCT FROM 'meritos' OR d->>'tipo_recurso' IS DISTINCT FROM 'hecho'
 OR d->>'finalidad' IS DISTINCT FROM 'consulta_hecho_propio'
 OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
 OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
 OR d->'campos_permitidos' IS DISTINCT FROM '["hecho_actual","recibo_consulta"]'::jsonb
 OR d->'obligaciones' IS DISTINCT FROM '["auditar"]'::jsonb
 THEN RAISE EXCEPTION 'AD3-145: consulta propia denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna('meritos_consulta_propia_interna',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'AD3-145: requiere consumo nuevo' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_consulta_hecho_propio_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
DO $acl$
DECLARE f regprocedure:='vec_autorizacion_atestada_v3.consumir_consulta_hecho_propio_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 permitido oid:='vec_meritos_propietario'::regrole::oid; x record;
BEGIN
 -- También las ACL por defecto: ningún rol conserva acceso por haber sido
 -- destinatario predeterminado del propietario.
 FOR x IN SELECT DISTINCT a.grantee FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
  WHERE p.oid=f AND a.grantee<>p.proowner LOOP
  EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %s',f::text,CASE WHEN x.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(x.grantee)) END);
 END LOOP;
 EXECUTE format('GRANT EXECUTE ON FUNCTION %s TO vec_meritos_propietario',f::text);
 IF (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario'::regrole
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS NOT TRUE
    OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
    OR NOT has_schema_privilege('vec_meritos_propietario','vec_autorizacion_atestada_v3','USAGE')
 THEN RAISE EXCEPTION 'AD3-145: propietario o entorno de fachada incompatible' USING ERRCODE='55000'; END IF;
 FOR x IN SELECT a.grantee,a.privilege_type,a.is_grantable,p.proowner
  FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f LOOP
  IF (x.grantee<>x.proowner AND x.grantee IS DISTINCT FROM permitido) OR x.privilege_type<>'EXECUTE'
    OR (x.grantee=permitido AND x.is_grantable)
  THEN RAISE EXCEPTION 'AD3-145: ACL de fachada abierta' USING ERRCODE='55000'; END IF;
 END LOOP;
END $acl$;

COMMIT;
