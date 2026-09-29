\set ON_ERROR_STOP on
-- AUT-15: tres cuentas nominales del portal candidato. La fuente y los motivos
-- son de lectura; el registro V3 conserva decisiones externas en historia propia.
-- Requiere ContextoActor 000012 y su comprobación positiva de provisión.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion:migracion:000015',0));
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion:migracion:registro_contexto_actor_v3:000005',0));

DO $preimagen$
DECLARE f regprocedure := pg_catalog.to_regprocedure('vec_autorizacion.registrar_decision_contexto_actor_v3(bytea,bytea,numeric,numeric)');
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
    OR pg_catalog.to_regrole('vec_autorizacion_propietario') IS NULL
    OR pg_catalog.to_regprocedure('vec_contexto_actor_v1.perfil_candidato_externo_provisionado_v1(text,text)') IS NULL
    OR pg_catalog.to_regprocedure('vec_identidad_externa_v1.acreditar_sesion_externa_v1(text,text,text,text,text,text,boolean,text,text,text,text,text,timestamptz,timestamptz,text,text,text,text,timestamptz,timestamptz)') IS NULL
    OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_proc p
      WHERE p.oid='vec_identidad_externa_v1.acreditar_sesion_externa_v1(text,text,text,text,text,text,boolean,text,text,text,text,text,timestamptz,timestamptz,text,text,text,text,timestamptz,timestamptz)'::regprocedure
        AND p.proowner='vec_identidad_sesiones_v1_propietario'::regrole AND p.prosecdef)
    OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_proc p
      WHERE p.oid='vec_contexto_actor_v1.perfil_candidato_externo_provisionado_v1(text,text)'::regprocedure
        AND p.proowner='vec_contexto_actor_v1_propietario'::regrole AND p.prosecdef)
    OR NOT pg_catalog.has_function_privilege('vec_autorizacion_propietario',
      'vec_contexto_actor_v1.perfil_candidato_externo_provisionado_v1(text,text)','EXECUTE')
    OR NOT pg_catalog.has_function_privilege('vec_autorizacion_propietario',
      'vec_identidad_externa_v1.acreditar_sesion_externa_v1(text,text,text,text,text,text,boolean,text,text,text,text,text,timestamptz,timestamptz,text,text,text,text,timestamptz,timestamptz)','EXECUTE')
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc p,
       LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
       WHERE p.oid='vec_identidad_externa_v1.acreditar_sesion_externa_v1(text,text,text,text,text,text,boolean,text,text,text,text,text,timestamptz,timestamptz,text,text,text,text,timestamptz,timestamptz)'::regprocedure
         AND a.grantee=0 AND a.privilege_type='EXECUTE')
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc p,
       LATERAL pg_catalog.aclexplode(coalesce(p.proacl,
         pg_catalog.acldefault('f',p.proowner))) a
       WHERE p.oid='vec_contexto_actor_v1.perfil_candidato_externo_provisionado_v1(text,text)'::regprocedure
         AND a.grantee=0 AND a.privilege_type='EXECUTE')
    OR f IS NULL
    OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_proc WHERE oid=f AND proowner='vec_autorizacion_propietario'::regrole AND prosecdef)
    OR pg_catalog.to_regclass('vec_autorizacion.decision_concedida_contexto_actor_v3') IS NULL
    OR pg_catalog.to_regclass('vec_autorizacion.decision_denegada_contexto_actor_v3') IS NULL
    OR pg_catalog.to_regrole('vec_autorizacion_fuente_externa') IS NOT NULL
    OR pg_catalog.to_regrole('vec_autorizacion_motivos_externos') IS NOT NULL
    OR pg_catalog.to_regrole('vec_autorizacion_registro_externo') IS NOT NULL
 THEN RAISE EXCEPTION 'AUT-15: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $preimagen$;

CREATE ROLE vec_autorizacion_fuente_externa NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_autorizacion_motivos_externos NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_autorizacion_registro_externo NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
DO $base$
DECLARE r text;
BEGIN
 FOREACH r IN ARRAY ARRAY['vec_autorizacion_fuente_externa','vec_autorizacion_motivos_externos','vec_autorizacion_registro_externo'] LOOP
  EXECUTE pg_catalog.format('GRANT CONNECT ON DATABASE %I TO %I',pg_catalog.current_database(),r);
 END LOOP;
END $base$;

SET LOCAL ROLE vec_autorizacion_propietario;
CREATE TABLE vec_autorizacion.decision_concedida_contexto_actor_v3_externa
 (LIKE vec_autorizacion.decision_concedida_contexto_actor_v3 INCLUDING ALL);
CREATE TABLE vec_autorizacion.decision_denegada_contexto_actor_v3_externa
 (LIKE vec_autorizacion.decision_denegada_contexto_actor_v3 INCLUDING ALL);
ALTER TABLE vec_autorizacion.decision_concedida_contexto_actor_v3_externa
 ADD FOREIGN KEY (asignacion_ref) REFERENCES vec_autorizacion.asignacion_perfil(asignacion_ref),
 ADD FOREIGN KEY (version_rol_ref) REFERENCES vec_autorizacion.version_rol(version_rol_ref),
 ADD FOREIGN KEY (motivo_catalogo_id,motivo_catalogo_version) REFERENCES vec_autorizacion.motivo_v2_catalogo_publicado(catalogo_id,catalogo_version),
 ADD FOREIGN KEY (motivo_catalogo_id,motivo_catalogo_version,motivo_entrada_clave) REFERENCES vec_autorizacion.motivo_v2_entrada(catalogo_id,catalogo_version,entrada_clave);
ALTER TABLE vec_autorizacion.decision_denegada_contexto_actor_v3_externa
 ADD FOREIGN KEY (asignacion_ref) REFERENCES vec_autorizacion.asignacion_perfil(asignacion_ref),
 ADD FOREIGN KEY (version_rol_ref) REFERENCES vec_autorizacion.version_rol(version_rol_ref),
 ADD FOREIGN KEY (motivo_catalogo_id,motivo_catalogo_version) REFERENCES vec_autorizacion.motivo_v2_catalogo_publicado(catalogo_id,catalogo_version),
 ADD FOREIGN KEY (motivo_catalogo_id,motivo_catalogo_version,motivo_entrada_clave) REFERENCES vec_autorizacion.motivo_v2_entrada(catalogo_id,catalogo_version,entrada_clave);
DO $protecciones$
DECLARE n text;
BEGIN
 FOREACH n IN ARRAY ARRAY['decision_concedida_contexto_actor_v3_externa','decision_denegada_contexto_actor_v3_externa'] LOOP
  EXECUTE pg_catalog.format('ALTER TABLE vec_autorizacion.%I ENABLE ROW LEVEL SECURITY',n);
  EXECUTE pg_catalog.format('ALTER TABLE vec_autorizacion.%I FORCE ROW LEVEL SECURITY',n);
  EXECUTE pg_catalog.format('CREATE POLICY acceso_propietario_exacto ON vec_autorizacion.%I FOR ALL TO vec_autorizacion_propietario USING (current_user = %L) WITH CHECK (current_user = %L)',n,'vec_autorizacion_propietario','vec_autorizacion_propietario');
  EXECUTE pg_catalog.format('CREATE TRIGGER decision_v3_inmutable BEFORE UPDATE OR DELETE ON vec_autorizacion.%I FOR EACH ROW EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable()',n);
  EXECUTE pg_catalog.format('CREATE TRIGGER decision_v3_no_truncar BEFORE TRUNCATE ON vec_autorizacion.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable()',n);
 END LOOP;
END $protecciones$;
REVOKE ALL ON TABLE vec_autorizacion.decision_concedida_contexto_actor_v3_externa,vec_autorizacion.decision_denegada_contexto_actor_v3_externa FROM PUBLIC;
REVOKE ALL ON TYPE vec_autorizacion.decision_concedida_contexto_actor_v3_externa,vec_autorizacion.decision_denegada_contexto_actor_v3_externa FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.login_candidato_externo_v1(p_login text,p_grupo text)
RETURNS boolean LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog AS $funcion$
 SELECT session_user = p_login
    AND EXISTS (SELECT 1 FROM pg_catalog.pg_roles r WHERE r.rolname=session_user
      AND r.rolcanlogin AND r.rolinherit AND NOT r.rolsuper AND NOT r.rolcreatedb
      AND NOT r.rolcreaterole AND NOT r.rolreplication AND NOT r.rolbypassrls)
    AND (SELECT pg_catalog.count(*) FROM pg_catalog.pg_auth_members m
      WHERE m.member=session_user::regrole)=1
    AND EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m
      JOIN pg_catalog.pg_roles g ON g.oid=m.roleid
      WHERE m.member=session_user::regrole AND g.rolname=p_grupo
        AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)
$funcion$;


CREATE FUNCTION vec_autorizacion.revalidar_sesion_vinculo_externo_v2(
 p_vinculo jsonb,p_emitida_en timestamptz,p_valida_hasta timestamptz,p_instante timestamptz)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $funcion$
BEGIN
 IF vec_autorizacion.vinculo_contexto_actor_v2_valido(p_vinculo) IS NOT TRUE
    OR NOT (vec_autorizacion.login_candidato_externo_v1('vec_externo_v3_registro_autorizacion_desarrollo','vec_autorizacion_registro_externo')
       OR vec_autorizacion.login_candidato_externo_v1('vec_externo_bolsa_desarrollo','vec_bolsa_llamamientos_portal_externo'))
    OR p_vinculo->>'superficie' IS DISTINCT FROM 'externa_personal'
    OR p_vinculo->>'cuenta_privilegiada' IS DISTINCT FROM 'false'
    OR p_emitida_en IS NULL OR p_valida_hasta IS NULL OR p_instante IS NULL
    OR p_valida_hasta<=p_emitida_en
 THEN RETURN false; END IF;
 IF vec_identidad_externa_v1.acreditar_sesion_externa_v1(
   p_vinculo->>'autenticacion_ref',p_vinculo->>'autenticacion_huella_sha256',
   p_vinculo->>'asercion_ref',p_vinculo->>'sesion_ref',p_vinculo->>'cuenta_ref',
   p_vinculo->>'cuenta_ordinaria_ref',(p_vinculo->>'cuenta_privilegiada')::boolean,
   p_vinculo->>'superficie',p_vinculo->>'metodo_observado',p_vinculo->>'garantia_observada',
   p_vinculo->>'politica_garantia_ref',p_vinculo->>'politica_garantia_huella_sha256',
   (p_vinculo->>'autenticacion_verificada_en')::timestamptz,(p_vinculo->>'sesion_emitida_en')::timestamptz,
   p_vinculo->>'control_sesion_ref',p_vinculo->>'control_sesion_revision','activa',
   p_vinculo->>'control_sesion_huella_sha256',
   (p_vinculo->>'sesion_revalidada_en')::timestamptz,(p_vinculo->>'sesion_valida_hasta')::timestamptz
 ) IS NOT TRUE THEN RETURN false; END IF;
 RETURN p_emitida_en >= (p_vinculo->>'sesion_revalidada_en')::timestamptz
    AND p_valida_hasta <= (p_vinculo->>'sesion_valida_hasta')::timestamptz
    AND p_instante >= (p_vinculo->>'sesion_revalidada_en')::timestamptz
    AND p_instante < (p_vinculo->>'sesion_valida_hasta')::timestamptz;
EXCEPTION WHEN data_exception OR invalid_text_representation OR datetime_field_overflow THEN RETURN false;
END $funcion$;

-- Se congela la semántica de validación/CAS/replay de AUT-6 y se cambia solo
-- el almacén de decisiones. La preimagen comprueba cada sustitución exacta.
DO $copiar_registro$
DECLARE original text; nuevo text; f regprocedure;
        vieja_c text := 'vec_autorizacion.decision_concedida_contexto_actor_v3';
        vieja_d text := 'vec_autorizacion.decision_denegada_contexto_actor_v3';
BEGIN
 f := 'vec_autorizacion.registrar_decision_contexto_actor_v3(bytea,bytea,numeric,numeric)'::regprocedure;
 SELECT pg_catalog.pg_get_functiondef(f) INTO STRICT original;
 IF pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(original,'UTF8')),'hex')
      <> '45c7708252254d24df761419fe80fbfd00749cb0f5b47e5221631fd70b5d3e3f'
    OR (pg_catalog.length(original)-pg_catalog.length(pg_catalog.replace(original,vieja_c,'')))/pg_catalog.length(vieja_c) <> 6
    OR (pg_catalog.length(original)-pg_catalog.length(pg_catalog.replace(original,vieja_d,'')))/pg_catalog.length(vieja_d) <> 6
    OR (pg_catalog.length(original)-pg_catalog.length(pg_catalog.replace(original,'vec_autorizacion.revalidar_sesion_vinculo_v2','')))/pg_catalog.length('vec_autorizacion.revalidar_sesion_vinculo_v2') <> 2
    OR pg_catalog.strpos(original,'CREATE OR REPLACE FUNCTION vec_autorizacion.registrar_decision_contexto_actor_v3(') <> 1
 THEN RAISE EXCEPTION 'AUT-15: registrador V3 incompatible' USING ERRCODE='55000'; END IF;
 nuevo := pg_catalog.replace(original,'CREATE OR REPLACE FUNCTION vec_autorizacion.registrar_decision_contexto_actor_v3(',
     'CREATE FUNCTION vec_autorizacion.registrar_decision_contexto_actor_v3_externa_interna(');
 nuevo := pg_catalog.replace(nuevo,vieja_c,vieja_c||'_externa');
 nuevo := pg_catalog.replace(nuevo,vieja_d,vieja_d||'_externa');
 nuevo := pg_catalog.replace(nuevo,'vec_autorizacion.revalidar_sesion_vinculo_v2','vec_autorizacion.revalidar_sesion_vinculo_externo_v2');
 EXECUTE nuevo;
END $copiar_registro$;

-- AUT-7 revalida una fila concreta y todos los controles vivos. Se conserva
-- su cuerpo exacto; la única sustitución es la tabla de concesiones externas.
DO $copiar_revalidacion$
DECLARE original text; nuevo text; f regprocedure;
        vieja text := 'vec_autorizacion.decision_concedida_contexto_actor_v3';
BEGIN
 f := 'vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(bytea,bytea,numeric,numeric)'::regprocedure;
 SELECT pg_catalog.pg_get_functiondef(f) INTO STRICT original;
 IF pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(original,'UTF8')),'hex')
      <> '0a0cd2e853081f47be05e1bb183c5dc09f7480433df25aa79536cc66e3a606a7'
    OR (pg_catalog.length(original)-pg_catalog.length(pg_catalog.replace(original,vieja,'')))/pg_catalog.length(vieja) <> 1
    OR (pg_catalog.length(original)-pg_catalog.length(pg_catalog.replace(original,'vec_autorizacion.revalidar_sesion_vinculo_v2','')))/pg_catalog.length('vec_autorizacion.revalidar_sesion_vinculo_v2') <> 1
    OR pg_catalog.strpos(original,'CREATE OR REPLACE FUNCTION vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(') <> 1
 THEN RAISE EXCEPTION 'AUT-15: revalidador V3 incompatible' USING ERRCODE='55000'; END IF;
 nuevo := pg_catalog.replace(original,'CREATE OR REPLACE FUNCTION vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(',
     'CREATE FUNCTION vec_autorizacion.revalidar_decision_contexto_actor_v3_externa_interna(');
 nuevo := pg_catalog.replace(nuevo,vieja,vieja||'_externa');
 nuevo := pg_catalog.replace(nuevo,'vec_autorizacion.revalidar_sesion_vinculo_v2','vec_autorizacion.revalidar_sesion_vinculo_externo_v2');
 EXECUTE nuevo;
END $copiar_revalidacion$;


CREATE FUNCTION vec_autorizacion.obtener_instantanea_candidato_externo_v1(p_principal_id text,p_perfil_activo_ref text)
RETURNS TABLE(documento_asignacion jsonb,documento_rol jsonb,documento_control_rol jsonb,revision_catalogo text,huella_catalogo text,documentos_politicas jsonb)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $funcion$
BEGIN
 IF vec_autorizacion.login_candidato_externo_v1('vec_externo_v3_fuente_autorizacion_desarrollo','vec_autorizacion_fuente_externa') IS NOT TRUE
    OR vec_contexto_actor_v1.perfil_candidato_externo_provisionado_v1(p_principal_id,p_perfil_activo_ref) IS NOT TRUE
 THEN RAISE EXCEPTION 'fuente externa rechazada' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT i.* FROM vec_autorizacion.obtener_instantanea(p_principal_id,p_perfil_activo_ref) i
  WHERE i.documento_rol->>'rol_id' IN
   ('candidato_bolsa_historial_propio_desarrollo','candidato_bolsa_portal_historial_propio_desarrollo');
END $funcion$;

CREATE FUNCTION vec_autorizacion.resolver_motivo_candidato_externo_v1(p_catalogo_id text,p_catalogo_version integer,p_huella text,p_entrada text,p_instante timestamptz)
RETURNS boolean LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path=pg_catalog AS $funcion$
BEGIN
 IF vec_autorizacion.login_candidato_externo_v1('vec_externo_v3_motivos_autorizacion_desarrollo','vec_autorizacion_motivos_externos') IS NOT TRUE
    OR p_catalogo_id NOT IN ('motivos_mi_bolsa_desarrollo','motivos_historial_mi_bolsa_desarrollo','motivos_portal_mi_bolsa_desarrollo')
 THEN RAISE EXCEPTION 'motivo externo rechazado' USING ERRCODE='42501'; END IF;
 RETURN vec_autorizacion.resolver_motivo_autorizacion_v2_historico(p_catalogo_id,p_catalogo_version,p_huella,p_entrada,p_instante);
END $funcion$;

CREATE FUNCTION vec_autorizacion.registrar_decision_candidato_externo_v3(p_decision bytea,p_motivo bytea,p_persona_version numeric,p_perfil_version numeric)
RETURNS TABLE(concedida boolean,codigo text,decision_huella_sha256 text,registrada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $funcion$
DECLARE d jsonb; v jsonb; m jsonb;
BEGIN
 IF vec_autorizacion.login_candidato_externo_v1('vec_externo_v3_registro_autorizacion_desarrollo','vec_autorizacion_registro_externo') IS NOT TRUE
    OR p_decision IS NULL OR p_motivo IS NULL
    OR pg_catalog.octet_length(p_decision) NOT BETWEEN 1 AND 524288
    OR pg_catalog.octet_length(p_motivo) NOT BETWEEN 1 AND 65536
 THEN RAISE EXCEPTION 'registro externo rechazado' USING ERRCODE='42501'; END IF;
 BEGIN
  d := pg_catalog.convert_from(p_decision,'UTF8')::jsonb;
  m := pg_catalog.convert_from(p_motivo,'UTF8')::jsonb;
 EXCEPTION WHEN OTHERS THEN RAISE EXCEPTION 'registro externo rechazado' USING ERRCODE='42501'; END;
 v := d->'vinculo_autenticacion_actor';
 IF vec_autorizacion.decision_contexto_actor_v3_valida(d) IS NOT TRUE
    OR vec_autorizacion.decision_contexto_actor_v3_canonica(d) IS DISTINCT FROM p_decision
    OR vec_autorizacion.motivo_contexto_actor_v3_canonico(m) IS DISTINCT FROM p_motivo
    OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
    OR v->>'superficie' IS DISTINCT FROM 'externa_personal'
    OR v->>'cuenta_privilegiada' IS DISTINCT FROM 'false'
    OR d->>'principal_id' IS DISTINCT FROM v->>'principal_id'
    OR d->>'perfil_activo_ref' IS DISTINCT FROM v->>'perfil_activo_ref'
    OR m->'referencia'->>'catalogo_id' NOT IN ('motivos_mi_bolsa_desarrollo','motivos_historial_mi_bolsa_desarrollo','motivos_portal_mi_bolsa_desarrollo')
    OR NOT EXISTS (SELECT 1 FROM vec_autorizacion.version_rol r
      WHERE r.version_rol_ref=d->>'version_rol_ref'
        AND r.rol_id IN ('candidato_bolsa_historial_propio_desarrollo','candidato_bolsa_portal_historial_propio_desarrollo'))
 THEN RAISE EXCEPTION 'registro externo rechazado' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT * FROM vec_autorizacion.registrar_decision_contexto_actor_v3_externa_interna(p_decision,p_motivo,p_persona_version,p_perfil_version);
END $funcion$;

-- Estas dos fachadas las invoca exclusivamente el núcleo AD3 desde el login
-- propio de Bolsa. No otorgan EXECUTE directo a dicho login sobre AUT.
CREATE FUNCTION vec_autorizacion.revalidar_decision_contexto_actor_externa_v3_viva(
 p_decision bytea,p_motivo bytea,p_persona_version numeric,p_perfil_version numeric)
RETURNS timestamptz LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $funcion$
BEGIN
 IF vec_autorizacion.login_candidato_externo_v1('vec_externo_bolsa_desarrollo','vec_bolsa_llamamientos_portal_externo') IS NOT TRUE
 THEN RAISE EXCEPTION 'revalidación externa rechazada' USING ERRCODE='42501'; END IF;
 RETURN vec_autorizacion.revalidar_decision_contexto_actor_v3_externa_interna(
   p_decision,p_motivo,p_persona_version,p_perfil_version);
END $funcion$;

CREATE FUNCTION vec_autorizacion.registrar_y_revalidar_decision_contexto_actor_externa_v3(
 p_decision bytea,p_motivo bytea,p_persona_version numeric,p_perfil_version numeric)
RETURNS TABLE(concedida boolean,codigo text,decision_huella_sha256 text,registrada_en timestamptz,revalidada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $funcion$
DECLARE d jsonb; m jsonb; v jsonb; registro record; viva timestamptz;
BEGIN
 IF vec_autorizacion.login_candidato_externo_v1('vec_externo_bolsa_desarrollo','vec_bolsa_llamamientos_portal_externo') IS NOT TRUE
    OR p_decision IS NULL OR p_motivo IS NULL
    OR pg_catalog.octet_length(p_decision) NOT BETWEEN 1 AND 524288
    OR pg_catalog.octet_length(p_motivo) NOT BETWEEN 1 AND 65536
 THEN RAISE EXCEPTION 'registro AD3 externo rechazado' USING ERRCODE='42501'; END IF;
 BEGIN
  d:=pg_catalog.convert_from(p_decision,'UTF8')::jsonb;
  m:=pg_catalog.convert_from(p_motivo,'UTF8')::jsonb;
 EXCEPTION WHEN OTHERS THEN RAISE EXCEPTION 'registro AD3 externo rechazado' USING ERRCODE='42501'; END;
 v:=d->'vinculo_autenticacion_actor';
 IF vec_autorizacion.decision_contexto_actor_v3_valida(d) IS NOT TRUE
    OR vec_autorizacion.decision_contexto_actor_v3_canonica(d) IS DISTINCT FROM p_decision
    OR vec_autorizacion.motivo_contexto_actor_v3_canonico(m) IS DISTINCT FROM p_motivo
    OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
    OR v->>'superficie' IS DISTINCT FROM 'externa_personal'
    OR v->>'cuenta_privilegiada' IS DISTINCT FROM 'false'
    OR d->>'principal_id' IS DISTINCT FROM v->>'principal_id'
    OR d->>'perfil_activo_ref' IS DISTINCT FROM v->>'perfil_activo_ref'
    OR m->'referencia'->>'catalogo_id' NOT IN ('motivos_mi_bolsa_desarrollo','motivos_historial_mi_bolsa_desarrollo','motivos_portal_mi_bolsa_desarrollo')
    OR NOT EXISTS (SELECT 1 FROM vec_autorizacion.version_rol r
       WHERE r.version_rol_ref=d->>'version_rol_ref'
         AND r.rol_id IN ('candidato_bolsa_historial_propio_desarrollo','candidato_bolsa_portal_historial_propio_desarrollo'))
 THEN RAISE EXCEPTION 'registro AD3 externo rechazado' USING ERRCODE='42501'; END IF;
 SELECT * INTO registro FROM vec_autorizacion.registrar_decision_contexto_actor_v3_externa_interna(
   p_decision,p_motivo,p_persona_version,p_perfil_version);
 IF NOT FOUND OR registro.concedida IS NOT TRUE THEN RETURN; END IF;
 viva:=vec_autorizacion.revalidar_decision_contexto_actor_v3_externa_interna(
   p_decision,p_motivo,p_persona_version,p_perfil_version);
 IF viva IS NULL THEN RETURN; END IF;
 RETURN QUERY SELECT registro.concedida,registro.codigo,registro.decision_huella_sha256,registro.registrada_en,viva;
END $funcion$;

REVOKE ALL ON FUNCTION vec_autorizacion.registrar_decision_contexto_actor_v3_externa_interna(bytea,bytea,numeric,numeric),
 vec_autorizacion.revalidar_sesion_vinculo_externo_v2(jsonb,timestamptz,timestamptz,timestamptz),
 vec_autorizacion.revalidar_decision_contexto_actor_v3_externa_interna(bytea,bytea,numeric,numeric),
 vec_autorizacion.login_candidato_externo_v1(text,text),
 vec_autorizacion.obtener_instantanea_candidato_externo_v1(text,text),
 vec_autorizacion.resolver_motivo_candidato_externo_v1(text,integer,text,text,timestamptz),
 vec_autorizacion.registrar_decision_candidato_externo_v3(bytea,bytea,numeric,numeric),
 vec_autorizacion.revalidar_decision_contexto_actor_externa_v3_viva(bytea,bytea,numeric,numeric),
 vec_autorizacion.registrar_y_revalidar_decision_contexto_actor_externa_v3(bytea,bytea,numeric,numeric) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion TO vec_autorizacion_fuente_externa,vec_autorizacion_motivos_externos,vec_autorizacion_registro_externo;
GRANT EXECUTE ON FUNCTION vec_autorizacion.obtener_instantanea_candidato_externo_v1(text,text) TO vec_autorizacion_fuente_externa;
GRANT EXECUTE ON FUNCTION vec_autorizacion.resolver_motivo_candidato_externo_v1(text,integer,text,text,timestamptz) TO vec_autorizacion_motivos_externos;
GRANT EXECUTE ON FUNCTION vec_autorizacion.registrar_decision_candidato_externo_v3(bytea,bytea,numeric,numeric) TO vec_autorizacion_registro_externo;
GRANT USAGE ON SCHEMA vec_autorizacion TO vec_autorizacion_atestada_v3_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion.revalidar_decision_contexto_actor_externa_v3_viva(bytea,bytea,numeric,numeric),
 vec_autorizacion.registrar_y_revalidar_decision_contexto_actor_externa_v3(bytea,bytea,numeric,numeric)
 TO vec_autorizacion_atestada_v3_propietario;
COMMIT;
