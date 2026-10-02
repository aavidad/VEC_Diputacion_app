\set ON_ERROR_STOP on
-- Consulta del hecho propio actual. Ninguna escritura de negocio RUM03.
BEGIN;
SET LOCAL ROLE vec_meritos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp; SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s'; SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_meritos:migracion:000002',0));
DO $pre$ BEGIN
 IF current_user<>'vec_meritos_propietario' OR to_regclass('vec_meritos.hecho_version') IS NULL
 OR to_regclass('vec_meritos.consulta_propia_v1') IS NOT NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_consulta_hecho_propio_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 OR to_regprocedure('vec_autorizacion.acreditar_intento_consulta_meritos_v1(text,text)') IS NULL
 THEN RAISE EXCEPTION 'meritos.error.consulta_preimagen_incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
CREATE TABLE vec_meritos.consulta_propia_v1 (
 recibo_ref text PRIMARY KEY CHECK(vec_meritos.referencia_valida_v1(recibo_ref)),
 persona_ref text NOT NULL CHECK(vec_meritos.referencia_valida_v1(persona_ref)),
 hecho_ref text NOT NULL CHECK(vec_meritos.referencia_valida_v1(hecho_ref)),
 version_consultada integer NOT NULL CHECK(version_consultada>=0 AND version_consultada<=1073741824),
 resultado text NOT NULL CHECK(resultado IN ('obtenida','no_encontrada')),
 decision_ref text NOT NULL UNIQUE,
 consumo_huella_sha256 text NOT NULL UNIQUE CHECK(consumo_huella_sha256 ~ '^[0-9a-f]{64}$'),
 auditoria_ref text NOT NULL UNIQUE CHECK(vec_meritos.referencia_valida_v1(auditoria_ref)),
 auditoria_consumo_ref text NOT NULL,
 correlacion_ref text NOT NULL,
 consultada_en timestamptz(6) NOT NULL,
 CHECK((resultado='obtenida' AND version_consultada>0) OR (resultado='no_encontrada' AND version_consultada=0))
);
ALTER TABLE vec_meritos.consulta_propia_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_meritos.consulta_propia_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY consulta_propia_lectura ON vec_meritos.consulta_propia_v1 FOR SELECT TO vec_meritos_propietario USING(persona_ref=current_setting('vec_meritos.persona_ref',true));
CREATE POLICY consulta_propia_alta ON vec_meritos.consulta_propia_v1 FOR INSERT TO vec_meritos_propietario WITH CHECK(persona_ref=current_setting('vec_meritos.persona_ref',true));
CREATE TRIGGER consulta_propia_historia BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_meritos.consulta_propia_v1 FOR EACH STATEMENT EXECUTE FUNCTION vec_meritos.historia_inmutable_v1();
REVOKE ALL ON TABLE vec_meritos.consulta_propia_v1 FROM PUBLIC,vec_meritos_consulta_propia_interno,vec_meritos_ejecutor,vec_meritos_interno,vec_meritos_externo,vec_meritos_migrador;
REVOKE ALL ON TYPE vec_meritos.consulta_propia_v1 FROM PUBLIC,vec_meritos_consulta_propia_interno,vec_meritos_registrador_intento_consulta,vec_meritos_ejecutor,vec_meritos_interno,vec_meritos_externo,vec_meritos_migrador;
CREATE FUNCTION vec_meritos.consultar_hecho_propio_v1(p_selector bytea,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET lock_timeout='2s' AS $f$
DECLARE s jsonb; d jsonb; cx jsonb; material record; actor text; contexto text; v integer:=0; h jsonb; ficha jsonb:=NULL; rev jsonb; codigo text:='no_encontrada'; recibo text; audit text; ahora timestamptz(6); r jsonb;
BEGIN
 IF current_user<>'vec_meritos_propietario' OR session_user=current_user
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=session_user AND rolcanlogin AND NOT rolsuper AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls)
 OR NOT EXISTS(SELECT 1 FROM pg_auth_members WHERE member=session_user::regrole AND roleid='vec_meritos_consulta_propia_interno'::regrole AND inherit_option AND NOT set_option AND NOT admin_option)
 OR (SELECT count(*) FROM pg_auth_members WHERE member=session_user::regrole)<>1
 OR EXISTS(SELECT 1 FROM pg_auth_members WHERE member='vec_meritos_consulta_propia_interno'::regrole)
 OR current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off' OR current_setting('TimeZone')<>'UTC'
 THEN RAISE EXCEPTION 'meritos.error.consulta_identidad_transaccion_denegada' USING ERRCODE='42501'; END IF;
 IF p_selector IS NULL OR octet_length(p_selector) NOT BETWEEN 1 AND 4096 THEN RAISE EXCEPTION 'meritos.error.consulta_selector_invalido' USING ERRCODE='22023'; END IF;
 BEGIN s:=convert_from(p_selector,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb; cx:=convert_from(p_contexto,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'meritos.error.consulta_material_invalido' USING ERRCODE='22023'; END;
 IF vec_meritos.claves_exactas_v1(s,ARRAY['esquema','hecho_ref','persona_ref']) IS NOT TRUE
 OR s->>'esquema' IS DISTINCT FROM 'vec.meritos.hecho.consulta_propia.v1'
 OR jsonb_typeof(s->'hecho_ref') IS DISTINCT FROM 'string' OR jsonb_typeof(s->'persona_ref') IS DISTINCT FROM 'string'
 OR vec_meritos.referencia_valida_v1(s->>'hecho_ref') IS NOT TRUE OR vec_meritos.referencia_valida_v1(s->>'persona_ref') IS NOT TRUE
 THEN RAISE EXCEPTION 'meritos.error.consulta_selector_invalido' USING ERRCODE='22023'; END IF;
 contexto:='{"ambitos":{"huella_consulta_sha256":"'||encode(sha256(p_selector),'hex')||'","persona_ref":"'||(s->>'persona_ref')||'"},"atributos":{}}';
 actor:=d->>'principal_id';
 IF actor IS DISTINCT FROM s->>'persona_ref' OR cx->>'persona_ref' IS DISTINCT FROM actor
 OR d->>'accion' IS DISTINCT FROM 'meritos.hecho.consultar_propio' OR d->>'finalidad' IS DISTINCT FROM 'consulta_hecho_propio'
 OR d->>'modulo_id' IS DISTINCT FROM 'meritos' OR d->>'tipo_recurso' IS DISTINCT FROM 'hecho' OR d->>'recurso_ref' IS DISTINCT FROM s->>'hecho_ref'
 OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM encode(sha256(convert_to(contexto,'UTF8')),'hex')
 OR d->'campos_permitidos' IS DISTINCT FROM '["hecho_actual","recibo_consulta"]'::jsonb OR d->'obligaciones' IS DISTINCT FROM '["auditar"]'::jsonb
 OR d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'interna_corporativa'
 THEN RAISE EXCEPTION 'meritos.error.consulta_autorizacion_denegada' USING ERRCODE='42501'; END IF;
 -- No hay lectura de negocio antes de consumir la autoridad nominal real.
 SELECT * INTO STRICT material FROM vec_autorizacion_atestada_v3.consumir_consulta_hecho_propio_v3_atestada(p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF material.decision_ref IS DISTINCT FROM d->>'decision_ref' OR material.efecto_ref IS DISTINCT FROM s->>'hecho_ref'
 OR material.huella_efecto_sha256 IS DISTINCT FROM d->>'contexto_recurso_huella_sha256' OR material.consumo_nuevo IS NOT TRUE
 THEN RAISE EXCEPTION 'meritos.error.consulta_consumo_incompatible' USING ERRCODE='42501'; END IF;
 PERFORM set_config('vec_meritos.persona_ref',actor,true);
 SELECT hv.version,hv.registro->'hecho' INTO v,h FROM vec_meritos.hecho_version hv
 WHERE hv.hecho_ref=s->>'hecho_ref' AND hv.persona_ref=actor ORDER BY hv.version DESC LIMIT 1;
 IF FOUND THEN
  codigo:='obtenida';
  ficha:=jsonb_build_object('referencia',h->'referencia','version',v,'tipo',h->'tipo','concepto_ref',h->'concepto_ref','denominacion',h->'denominacion','procedencia',h->'procedencia','vigencia',h->'vigencia','estado',h->'estado','evidencias',coalesce(h->'evidencias','[]'::jsonb));
  IF h ? 'horas' THEN ficha:=ficha||jsonb_build_object('horas',h->'horas'); END IF;
  rev:=h->'revision';
  IF rev IS NOT NULL AND rev<>'null'::jsonb THEN ficha:=ficha||jsonb_build_object('revision',jsonb_build_object('referencia',rev->'referencia','motivo_ref',rev->'motivo_ref','fecha',rev->'fecha')); END IF;
 ELSE v:=0; END IF;
 ahora:=clock_timestamp(); recibo:='consulta_merito:'||gen_random_uuid()::text; audit:='auditoria_consulta:'||gen_random_uuid()::text;
 INSERT INTO vec_meritos.consulta_propia_v1 VALUES(recibo,actor,s->>'hecho_ref',v,codigo,material.decision_ref,material.consumo_huella_sha256,audit,material.auditoria_ref,d->>'correlacion_ref',ahora);
 r:=jsonb_build_object('referencia',recibo,'hecho_ref',s->>'hecho_ref','version_consultada',v,'decision_ref',material.decision_ref,'consumo_huella_sha256',material.consumo_huella_sha256,'auditoria_ref',audit,'correlacion_ref',d->>'correlacion_ref','consultada_en',to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'));
 RETURN jsonb_build_object('codigo',codigo,'hecho_actual',ficha,'recibo_consulta',r);
END $f$;
REVOKE ALL ON FUNCTION vec_meritos.consultar_hecho_propio_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_meritos.consultar_hecho_propio_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_meritos_consulta_propia_interno;

-- Otra transacción, después del rollback de la lectura. La identidad viene
-- de Autorización; el resultado es la observación del registrador confiable.
CREATE FUNCTION vec_meritos.registrar_intento_consulta_propia_v1(p_decision_ref text,p_correlacion_ref text,p_resultado text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='1s' SET statement_timeout='2s' AS $f$
DECLARE identidad jsonb; entrada jsonb; anterior jsonb; referencia text; clave text; ahora timestamptz(6);
BEGIN
 IF current_user<>'vec_meritos_propietario' OR session_user=current_user
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
 THEN RAISE EXCEPTION 'meritos.error.registrador_intento_denegado' USING ERRCODE='42501'; END IF;
 IF p_resultado IS NULL OR p_resultado NOT IN ('denegada','no_confirmado')
 THEN RAISE EXCEPTION 'meritos.error.resultado_intento_invalido' USING ERRCODE='22023'; END IF;
 identidad:=vec_autorizacion.acreditar_intento_consulta_meritos_v1(p_decision_ref,p_correlacion_ref);
 IF identidad IS NULL OR identidad->>'pdp_resultado' IS DISTINCT FROM 'concedida'
 OR vec_meritos.referencia_valida_v1(identidad->>'actor_id') IS NOT TRUE
 THEN RAISE EXCEPTION 'meritos.error.identidad_intento_no_acreditada' USING ERRCODE='42501'; END IF;
 clave:='intento_consulta:'||encode(sha256(convert_to(p_decision_ref||'|'||p_correlacion_ref,'UTF8')),'hex');
 referencia:='auditoria_'||clave;
 PERFORM set_config('vec_meritos.persona_ref',identidad->>'actor_id',true);
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_meritos:'||clave,0));
 entrada:=jsonb_build_object('id',referencia,'actor_id',identidad->>'actor_id','actor_profile',identidad->>'actor_profile',
  'action','meritos.hecho.consultar_propio','module_id','meritos','purpose','consulta_hecho_propio',
  'subject_ref',identidad->>'subject_ref','result',p_resultado,'correlation_ref',p_correlacion_ref,
  'authorization_ref',p_decision_ref,'rule_ref',identidad->>'rule_ref',
  'metadata',jsonb_build_object('tipo_evento','intento_consumo_consulta_propia',
    'pdp_resultado','concedida','origen_resultado','observacion_registrador',
    'registro_contexto_ref',identidad->>'registro_contexto_ref',
    'contexto_actor_huella_sha256',identidad->>'contexto_actor_huella_sha256',
    'concesion_huella_sha256',identidad->>'concesion_huella_sha256'));
 SELECT a.entrada INTO anterior FROM vec_meritos.auditoria_operacion a WHERE a.auditoria_ref=referencia;
 IF FOUND THEN
  IF anterior-'occurred_at' IS DISTINCT FROM entrada THEN
   RAISE EXCEPTION 'meritos.error.replay_intento_divergente' USING ERRCODE='23505'; END IF;
  RETURN anterior;
 END IF;
 ahora:=date_trunc('microseconds',clock_timestamp());
 entrada:=entrada||jsonb_build_object('occurred_at',to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'));
 INSERT INTO vec_meritos.auditoria_operacion(auditoria_ref,clave,persona_ref,entrada,registrada_en)
 VALUES(referencia,clave,identidad->>'actor_id',entrada,ahora);
 RETURN entrada;
END $f$;
REVOKE ALL ON FUNCTION vec_meritos.registrar_intento_consulta_propia_v1(text,text,text) FROM PUBLIC,vec_meritos_ejecutor,vec_meritos_interno,vec_meritos_externo,vec_meritos_migrador,vec_meritos_consulta_propia_interno;
DO $acl_intento$ DECLARE f regprocedure:='vec_meritos.registrar_intento_consulta_propia_v1(text,text,text)'::regprocedure; x record; BEGIN
 FOR x IN SELECT DISTINCT a.grantee FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
 WHERE p.oid=f AND a.grantee<>p.proowner LOOP
  EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %s',f::text,CASE WHEN x.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(x.grantee)) END);
 END LOOP;
END $acl_intento$;
REVOKE ALL ON TABLE vec_meritos.auditoria_operacion FROM vec_meritos_registrador_intento_consulta;
REVOKE ALL ON TYPE vec_meritos.auditoria_operacion FROM vec_meritos_registrador_intento_consulta;
GRANT EXECUTE ON FUNCTION vec_meritos.registrar_intento_consulta_propia_v1(text,text,text) TO vec_meritos_registrador_intento_consulta;
COMMIT;
