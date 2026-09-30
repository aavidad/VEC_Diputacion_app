\set ON_ERROR_STOP on
-- AD3-120: auditoría técnica interna del outbox Bolsa, sin identidad humana V3.
-- Requiere AD3 base y esquema Bolsa. Instalar antes de Bolsa 000062.
-- No modifica la cadena externa ni crea LOGIN o roles de ejecución.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000120',0));
DO $pre$
DECLARE p oid:=to_regrole('vec_autorizacion_atestada_v3_propietario'); u oid:=to_regrole('vec_bolsa_llamamientos_propietario');
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR p IS NULL OR u IS NULL OR p=u
 OR NOT EXISTS(SELECT 1 FROM pg_namespace WHERE oid=to_regnamespace('vec_autorizacion_atestada_v3') AND nspowner=p)
 OR NOT EXISTS(SELECT 1 FROM pg_namespace WHERE oid=to_regnamespace('vec_bolsa_llamamientos') AND nspowner=u)
 OR EXISTS(SELECT 1 FROM pg_roles WHERE oid IN(p,u) AND (rolcanlogin OR rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls))
 OR to_regprocedure('vec_autorizacion_atestada_v3.rechazar_mutacion()') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.rechazar_truncado()') IS NULL
 OR to_regclass('vec_autorizacion_atestada_v3.auditoria_tecnica_outbox_interna') IS NOT NULL
 OR to_regclass('vec_autorizacion_atestada_v3.control_cadena_tecnica_outbox_interna') IS NOT NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_auditoria_outbox_interno_v1(text,text,text,text,text,text,text,bigint,text)') IS NOT NULL
 THEN RAISE EXCEPTION 'AD3-120 preimagen incompatible o ya instalada' USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
CREATE TABLE vec_autorizacion_atestada_v3.control_cadena_tecnica_outbox_interna (
 control_id boolean PRIMARY KEY CHECK(control_id),
 secuencia bigint NOT NULL CHECK(secuencia BETWEEN 0 AND 9007199254740991),
 cabeza_sha256 text NOT NULL CHECK(cabeza_sha256 ~ '^[0-9a-f]{64}$')
);
INSERT INTO vec_autorizacion_atestada_v3.control_cadena_tecnica_outbox_interna VALUES(true,0,repeat('0',64));
CREATE TABLE vec_autorizacion_atestada_v3.auditoria_tecnica_outbox_interna (
 auditoria_ref text PRIMARY KEY CHECK(auditoria_ref ~ '^auditoria_tecnica_interna:[0-9a-f]{32}$'),
 secuencia bigint NOT NULL UNIQUE CHECK(secuencia BETWEEN 1 AND 9007199254740991),
 actor_tecnico text NOT NULL CHECK(actor_tecnico='vec_externo_avisos_bolsa'),
 perfil_tecnico text NOT NULL CHECK(perfil_tecnico='bolsa.outbox_avisos_externos.v1'),
 accion text NOT NULL CHECK(accion IN('extraer','aceptar','resultado')),
 productor_ref text CHECK(productor_ref ~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{0,191}$'),
 evento_ref text CHECK(evento_ref ~ '^evento_aviso:[0-9a-f]{64}$'),
 recibo_ref text CHECK(recibo_ref ~ '^aviso_recibo:[0-9a-f]{32}$'),
 recurso_ref text NOT NULL CHECK(recurso_ref='bolsa.outbox' OR recurso_ref ~ '^evento_aviso:[0-9a-f]{64}$' OR recurso_ref ~ '^aviso_recibo:[0-9a-f]{32}$'),
 correlacion_ref text NOT NULL CHECK(correlacion_ref ~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{0,191}$'),
 antes_sha256 text NOT NULL CHECK(antes_sha256 ~ '^[0-9a-f]{64}$'),
 despues_sha256 text NOT NULL CHECK(despues_sha256 ~ '^[0-9a-f]{64}$'),
 version bigint NOT NULL CHECK(version BETWEEN 0 AND 9007199254740991),
 resultado text NOT NULL CHECK(resultado IN('extraido','sin_registro','aceptado','no_aceptado','sin_destino','reservado_incierto','replay','denegado')),
 registrada_en timestamptz(6) NOT NULL,
 anterior_sha256 text NOT NULL CHECK(anterior_sha256 ~ '^[0-9a-f]{64}$'),
 huella_sha256 text NOT NULL UNIQUE CHECK(huella_sha256 ~ '^[0-9a-f]{64}$'),
 CHECK((productor_ref IS NULL AND evento_ref IS NULL AND recibo_ref IS NULL AND version=0 AND resultado IN('sin_registro','denegado'))
    OR (productor_ref IS NOT NULL AND evento_ref IS NOT NULL AND version>0 AND resultado<>'sin_registro' AND (recibo_ref IS NOT NULL OR accion='extraer' OR resultado='denegado'))),
 CHECK((accion='extraer' AND resultado IN('extraido','sin_registro','replay','denegado'))
    OR (accion='aceptar' AND resultado IN('aceptado','replay','denegado'))
    OR (accion='resultado' AND resultado IN('aceptado','no_aceptado','sin_destino','reservado_incierto','replay','denegado')))
);
CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_autorizacion_atestada_v3.auditoria_tecnica_outbox_interna
 FOR EACH ROW EXECUTE FUNCTION vec_autorizacion_atestada_v3.rechazar_mutacion();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_autorizacion_atestada_v3.auditoria_tecnica_outbox_interna
 FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion_atestada_v3.rechazar_truncado();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_autorizacion_atestada_v3.control_cadena_tecnica_outbox_interna
 FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion_atestada_v3.rechazar_truncado();
DO $tablas$ DECLARE t text; BEGIN
 FOREACH t IN ARRAY ARRAY['control_cadena_tecnica_outbox_interna','auditoria_tecnica_outbox_interna'] LOOP
  EXECUTE format('ALTER TABLE vec_autorizacion_atestada_v3.%I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE format('ALTER TABLE vec_autorizacion_atestada_v3.%I FORCE ROW LEVEL SECURITY',t);
  EXECUTE format('CREATE POLICY propietario_exacto ON vec_autorizacion_atestada_v3.%I FOR ALL TO vec_autorizacion_atestada_v3_propietario USING (current_user=''vec_autorizacion_atestada_v3_propietario'') WITH CHECK (current_user=''vec_autorizacion_atestada_v3_propietario'')',t);
  EXECUTE format('REVOKE ALL ON TABLE vec_autorizacion_atestada_v3.%I FROM PUBLIC,vec_bolsa_llamamientos_propietario,vec_bolsa_llamamientos_ejecutor',t);
  EXECUTE format('REVOKE ALL ON TYPE vec_autorizacion_atestada_v3.%I FROM PUBLIC,vec_bolsa_llamamientos_propietario,vec_bolsa_llamamientos_ejecutor',t);
 END LOOP;
END $tablas$;
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_auditoria_outbox_interno_v1(
 p_accion text,p_productor text,p_evento text,p_recibo text,p_correlacion text,
 p_antes_sha256 text,p_despues_sha256 text,p_version bigint,p_resultado text
) RETURNS text LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' SET statement_timeout='30s' AS $f$
DECLARE login_oid oid; grupo_oid oid; s bigint; cabeza text; ref text; ahora timestamptz(6); recurso text; preimagen jsonb; huella text;
BEGIN
 SELECT oid INTO login_oid FROM pg_roles WHERE rolname=session_user;
 SELECT oid INTO grupo_oid FROM pg_roles WHERE rolname='vec_bolsa_avisos_externos_consumidor';
 -- No hay actor, perfil ni instante aportados por el cliente. Sólo entra el
 -- propietario funcional por EXECUTE; session_user identifica al trabajador.
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
 OR session_user<>'vec_externo_avisos_bolsa'
 OR login_oid IS NULL OR grupo_oid IS NULL
 OR current_setting('transaction_isolation')<>'serializable'
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=session_user AND rolcanlogin AND rolinherit
    AND NOT rolsuper AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls
    AND (rolvaliduntil IS NULL OR rolvaliduntil>clock_timestamp()))
 OR (SELECT count(*) FROM pg_auth_members WHERE member=login_oid)<>1
 OR NOT EXISTS(SELECT 1 FROM pg_auth_members WHERE member=login_oid
    AND roleid=grupo_oid AND inherit_option AND NOT set_option AND NOT admin_option)
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_bolsa_avisos_externos_consumidor' AND rolinherit
    AND NOT rolcanlogin AND NOT rolsuper AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls)
 OR EXISTS(SELECT 1 FROM pg_auth_members WHERE member=grupo_oid OR roleid=login_oid)
 THEN RAISE EXCEPTION 'AD3-120 ejecutor técnico no admitido' USING ERRCODE='42501'; END IF;
 IF p_accion IS NULL OR p_resultado IS NULL OR p_correlacion IS NULL
 OR p_antes_sha256 IS NULL OR p_despues_sha256 IS NULL OR p_version IS NULL
 OR (p_productor IS NOT NULL AND p_productor !~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{0,191}$')
 OR (p_evento IS NOT NULL AND p_evento !~ '^evento_aviso:[0-9a-f]{64}$')
 OR p_correlacion !~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{0,191}$'
 OR p_antes_sha256 !~ '^[0-9a-f]{64}$' OR p_despues_sha256 !~ '^[0-9a-f]{64}$'
 OR p_version NOT BETWEEN 0 AND 9007199254740991
 OR (p_recibo IS NOT NULL AND p_recibo !~ '^aviso_recibo:[0-9a-f]{32}$')
 OR NOT ((p_productor IS NULL AND p_evento IS NULL AND p_recibo IS NULL AND p_version=0 AND p_resultado IN('sin_registro','denegado'))
    OR (p_productor IS NOT NULL AND p_evento IS NOT NULL AND p_version>0 AND p_resultado<>'sin_registro' AND (p_recibo IS NOT NULL OR p_accion='extraer' OR p_resultado='denegado')))
 OR NOT ((p_accion='extraer' AND p_resultado IN('extraido','sin_registro','replay','denegado'))
    OR (p_accion='aceptar' AND p_resultado IN('aceptado','replay','denegado'))
    OR (p_accion='resultado' AND p_resultado IN('aceptado','no_aceptado','sin_destino','reservado_incierto','replay','denegado')))
 THEN RAISE EXCEPTION 'AD3-120 material nominal inválido' USING ERRCODE='22023'; END IF;
 -- Un bloqueo único serializa sólo esta cadena; lock_timeout limita la espera.
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:cadena:outbox-interno.v1',0));
 SELECT secuencia,cabeza_sha256 INTO STRICT s,cabeza
 FROM vec_autorizacion_atestada_v3.control_cadena_tecnica_outbox_interna WHERE control_id FOR UPDATE;
 IF s>=9007199254740991 THEN RAISE EXCEPTION 'AD3-120 secuencia agotada' USING ERRCODE='22003'; END IF;
 s:=s+1; ahora:=clock_timestamp();
 ref:='auditoria_tecnica_interna:'||replace(gen_random_uuid()::text,'-','');
 recurso:=coalesce(p_recibo,p_evento,'bolsa.outbox');
 preimagen:=jsonb_build_object('tipo','vec.auditoria.outbox-interno.v1','auditoria_ref',ref,'secuencia',s,
   'actor_tecnico',session_user,'perfil_tecnico','bolsa.outbox_avisos_externos.v1','accion',p_accion,
   'productor_ref',p_productor,'evento_ref',p_evento,'recibo_ref',p_recibo,'recurso_ref',recurso,
   'correlacion_ref',p_correlacion,'antes_sha256',p_antes_sha256,'despues_sha256',p_despues_sha256,
   'version',p_version,'resultado',p_resultado,'registrada_en',to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
   'anterior_sha256',cabeza);
 huella:=encode(sha256(convert_to(preimagen::text,'UTF8')),'hex');
 INSERT INTO vec_autorizacion_atestada_v3.auditoria_tecnica_outbox_interna VALUES(
   ref,s,session_user,'bolsa.outbox_avisos_externos.v1',p_accion,p_productor,p_evento,p_recibo,recurso,
   p_correlacion,p_antes_sha256,p_despues_sha256,p_version,p_resultado,ahora,cabeza,huella);
 UPDATE vec_autorizacion_atestada_v3.control_cadena_tecnica_outbox_interna SET secuencia=s,cabeza_sha256=huella WHERE control_id;
 RETURN ref;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_auditoria_outbox_interno_v1(text,text,text,text,text,text,text,bigint,text)
 FROM PUBLIC,vec_bolsa_llamamientos_ejecutor;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_bolsa_llamamientos_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_auditoria_outbox_interno_v1(text,text,text,text,text,text,text,bigint,text)
 TO vec_bolsa_llamamientos_propietario;
RESET ROLE;
DO $post$
DECLARE t text; p oid:='vec_autorizacion_atestada_v3_propietario'::regrole; u oid:='vec_bolsa_llamamientos_propietario'::regrole;
 f regprocedure:='vec_autorizacion_atestada_v3.registrar_auditoria_outbox_interno_v1(text,text,text,text,text,text,text,bigint,text)'::regprocedure;
BEGIN
 FOREACH t IN ARRAY ARRAY['control_cadena_tecnica_outbox_interna','auditoria_tecnica_outbox_interna'] LOOP
  IF NOT EXISTS(SELECT 1 FROM pg_class WHERE oid=to_regclass('vec_autorizacion_atestada_v3.'||t) AND relowner=p AND relrowsecurity AND relforcerowsecurity)
  OR EXISTS(SELECT 1 FROM pg_class c CROSS JOIN LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) a
     WHERE c.oid=to_regclass('vec_autorizacion_atestada_v3.'||t) AND a.grantee<>p)
  OR EXISTS(SELECT 1 FROM pg_type y CROSS JOIN LATERAL aclexplode(coalesce(y.typacl,acldefault('T',y.typowner))) a
     WHERE y.oid=to_regtype('vec_autorizacion_atestada_v3.'||t) AND a.grantee<>p)
  THEN RAISE EXCEPTION 'AD3-120 tabla incompatible' USING ERRCODE='55000'; END IF;
 END LOOP;
 IF NOT EXISTS(SELECT 1 FROM pg_proc WHERE oid=f AND proowner=p AND prosecdef AND proconfig @> ARRAY['search_path=pg_catalog, pg_temp','row_security=on','lock_timeout=2s','statement_timeout=30s'])
 OR NOT has_function_privilege(u,f,'EXECUTE')
 OR EXISTS(SELECT 1 FROM pg_proc x CROSS JOIN LATERAL aclexplode(coalesce(x.proacl,acldefault('f',x.proowner))) a WHERE x.oid=f AND a.grantee NOT IN(p,u))
 THEN RAISE EXCEPTION 'AD3-120 escritor incompatible' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
