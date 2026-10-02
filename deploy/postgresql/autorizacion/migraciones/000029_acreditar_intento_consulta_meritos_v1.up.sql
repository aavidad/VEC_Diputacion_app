\set ON_ERROR_STOP on
-- AUT29, borrador: identidad histórica para auditar un intento de consumo.
-- No concede lectura de méritos, consume V3 ni altera el resultado del PDP.
BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path=pg_catalog; SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s'; SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock_shared(hashtextextended('vec_autorizacion:migracion:registro_contexto_actor_v3:000005',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion:migracion:000029',0));
DO $pre$ BEGIN
 IF current_user<>'vec_autorizacion_propietario'
 OR to_regprocedure('vec_autorizacion.acreditar_intento_consulta_meritos_v1(text,text)') IS NOT NULL
 OR NOT EXISTS(SELECT 1 FROM pg_class WHERE oid=to_regclass('vec_autorizacion.decision_concedida_contexto_actor_v3')
  AND relowner=current_user::regrole AND relrowsecurity AND relforcerowsecurity
  AND obj_description(oid,'pg_class')='vec_autorizacion:registro-contexto-actor-v3:000005')
 OR to_regprocedure('vec_autorizacion.registrar_decision_contexto_actor_v3(bytea,bytea,numeric,numeric)') IS NULL
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_meritos_registrador_intento_consulta'
  AND NOT rolcanlogin AND NOT rolinherit AND NOT rolsuper AND NOT rolcreatedb
  AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls)
 THEN RAISE EXCEPTION 'AUT29: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
CREATE FUNCTION vec_autorizacion.acreditar_intento_consulta_meritos_v1(p_decision_ref text,p_correlacion_ref text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET timezone='UTC'
SET lock_timeout='1s' SET statement_timeout='2s' AS $f$
DECLARE original vec_autorizacion.decision_concedida_contexto_actor_v3%ROWTYPE; d jsonb; v jsonb;
BEGIN
 IF current_user<>'vec_autorizacion_propietario' OR session_user=current_user
 OR current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=session_user AND rolcanlogin
  AND NOT rolsuper AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls)
 OR NOT EXISTS(SELECT 1 FROM pg_auth_members WHERE member=session_user::regrole
  AND roleid='vec_meritos_registrador_intento_consulta'::regrole AND inherit_option AND NOT set_option AND NOT admin_option)
 OR (SELECT count(*) FROM pg_auth_members WHERE member=session_user::regrole)<>1
 OR EXISTS(SELECT 1 FROM pg_auth_members WHERE member='vec_meritos_registrador_intento_consulta'::regrole)
 OR EXISTS(SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
  WHERE n.nspname IN ('vec_meritos','vec_autorizacion') AND c.relkind IN ('r','p','v','m','f')
   AND (has_table_privilege(session_user::regrole::oid,c.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER')
    OR has_any_column_privilege(session_user::regrole::oid,c.oid,'SELECT,INSERT,UPDATE,REFERENCES')))
 THEN RAISE EXCEPTION 'AUT29: registrador incompatible' USING ERRCODE='42501'; END IF;
 IF p_decision_ref IS NULL OR octet_length(p_decision_ref) NOT BETWEEN 1 AND 256
 OR p_decision_ref !~ '^[A-Za-z0-9:._-]+$'
 OR p_correlacion_ref IS NULL OR p_correlacion_ref !~ '^corr_[0-9a-f]{32}$'
 THEN RAISE EXCEPTION 'AUT29: selector inválido' USING ERRCODE='22023'; END IF;
 PERFORM pg_advisory_xact_lock_shared(hashtextextended('vec_autorizacion:migracion:registro_contexto_actor_v3:000005',0));
 SELECT * INTO original FROM vec_autorizacion.decision_concedida_contexto_actor_v3 WHERE decision_ref=p_decision_ref;
 IF NOT FOUND THEN RAISE EXCEPTION 'AUT29: concesión original no acreditada' USING ERRCODE='42501'; END IF;
 d:=original.documento; v:=d->'vinculo_autenticacion_actor';
 -- El registrador V3 acreditó ContextoActor y sesión antes y después de su
 -- CAS. Esta fila histórica es la fuente; no revalidamos vigencia actual,
 -- porque una revocación posterior puede ser la causa del intento rechazado.
 IF vec_autorizacion.decision_contexto_actor_v3_valida(d) IS NOT TRUE
 OR vec_autorizacion.decision_contexto_actor_v3_canonica(d) IS DISTINCT FROM original.decision_canonica
 OR vec_autorizacion.motivo_contexto_actor_v3_canonico(convert_from(original.motivo_canonico,'UTF8')::jsonb) IS DISTINCT FROM original.motivo_canonico
 OR encode(sha256(original.motivo_canonico),'hex') IS DISTINCT FROM d->>'motivo_huella_sha256'
 OR encode(sha256(original.decision_canonica),'hex') IS DISTINCT FROM original.huella_decision_sha256
 OR original.huella_decision_sha256 IS DISTINCT FROM encode(sha256(vec_autorizacion.decision_contexto_actor_v3_canonica(d)),'hex')
 OR d->'concedida' IS DISTINCT FROM 'true'::jsonb OR d->>'codigo' IS DISTINCT FROM 'concedida'
 OR d->>'decision_ref' IS DISTINCT FROM p_decision_ref OR d->>'correlacion_ref' IS DISTINCT FROM p_correlacion_ref
 OR d->>'accion' IS DISTINCT FROM 'meritos.hecho.consultar_propio'
 OR d->>'modulo_id' IS DISTINCT FROM 'meritos' OR d->>'tipo_recurso' IS DISTINCT FROM 'hecho'
 OR d->>'finalidad' IS DISTINCT FROM 'consulta_hecho_propio'
 OR d->'campos_permitidos' IS DISTINCT FROM '["hecho_actual","recibo_consulta"]'::jsonb
 OR d->'obligaciones' IS DISTINCT FROM '["auditar"]'::jsonb
 OR v->>'superficie' IS DISTINCT FROM 'interna_corporativa'
 OR v->>'principal_id' IS DISTINCT FROM d->>'principal_id'
 OR v->>'perfil_activo_ref' IS DISTINCT FROM d->>'perfil_activo_ref'
 OR v->>'registro_contexto_ref' IS DISTINCT FROM original.registro_contexto_ref
 OR v->>'contexto_actor_huella_sha256' IS DISTINCT FROM original.contexto_actor_huella_sha256
 OR v->>'manifiesto_procedencia_huella_sha256' IS DISTINCT FROM original.manifiesto_procedencia_huella_sha256
 OR v->>'autoridad_efectiva' IS DISTINCT FROM 'autoridad_maestra_acreditada'
 OR original.registrada_en<original.emitida_en OR original.registrada_en>=original.valida_hasta
 THEN RAISE EXCEPTION 'AUT29: concesión original no acreditada' USING ERRCODE='42501'; END IF;
 RETURN jsonb_build_object('actor_id',d->>'principal_id','actor_profile',d->>'perfil_activo_ref',
  'subject_ref',d->>'recurso_ref','authorization_ref',p_decision_ref,'correlation_ref',p_correlacion_ref,
  'rule_ref',original.motivo_entrada_clave,'registro_contexto_ref',original.registro_contexto_ref,
  'contexto_actor_huella_sha256',original.contexto_actor_huella_sha256,
  'concesion_huella_sha256',original.huella_decision_sha256,'pdp_resultado','concedida');
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.acreditar_intento_consulta_meritos_v1(text,text) FROM PUBLIC;
-- Ninguna ACL por defecto puede abrir esta proyección histórica al runtime.
DO $acl$ DECLARE f regprocedure:='vec_autorizacion.acreditar_intento_consulta_meritos_v1(text,text)'::regprocedure; x record; BEGIN
 FOR x IN SELECT DISTINCT a.grantee FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
 WHERE p.oid=f AND a.grantee<>p.proowner LOOP
  EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %s',f::text,CASE WHEN x.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(x.grantee)) END);
 END LOOP;
END $acl$;
DO $comentario$ DECLARE previo boolean; antes text; despues text; funciones text; BEGIN
 previo:=has_schema_privilege('vec_meritos_propietario','vec_autorizacion','USAGE');
 SELECT nspacl::text INTO antes FROM pg_namespace WHERE nspname='vec_autorizacion';
 IF NOT previo AND antes IS NULL THEN
  RAISE EXCEPTION 'AUT29: ACL de esquema sin preimagen explícita' USING ERRCODE='55000'; END IF;
 -- Si ya accedía por otra concesión, no añadimos otra ACL directa.
 IF NOT previo THEN GRANT USAGE ON SCHEMA vec_autorizacion TO vec_meritos_propietario; END IF;
 SELECT nspacl::text INTO despues FROM pg_namespace WHERE nspname='vec_autorizacion';
 SELECT encode(sha256(convert_to(coalesce(jsonb_agg(jsonb_build_object('oid',p.oid,'def',pg_get_functiondef(p.oid),'acl',p.proacl) ORDER BY p.oid),'[]'::jsonb)::text,'UTF8')),'hex')
 INTO funciones FROM pg_proc p WHERE p.proowner='vec_meritos_propietario'::regrole AND p.prokind IN ('f','p');
 EXECUTE format('COMMENT ON FUNCTION vec_autorizacion.acreditar_intento_consulta_meritos_v1(text,text) IS %L',
  jsonb_build_object('migracion','AUT29','meritos_usage_preexisting',previo,
   'schema_acl_before',antes,'schema_acl_after',despues,'meritos_funciones_sha256',funciones)::text);
END $comentario$;
GRANT EXECUTE ON FUNCTION vec_autorizacion.acreditar_intento_consulta_meritos_v1(text,text) TO vec_meritos_propietario;
COMMIT;
