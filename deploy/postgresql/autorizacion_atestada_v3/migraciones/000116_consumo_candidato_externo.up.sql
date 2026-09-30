\set ON_ERROR_STOP on
-- AD3-116. La decisión externa vive en AUT-15; AD3 conserva su atestación,
-- consumo y cadena de auditoría en tablas propias. El núcleo despacha por
-- LOGIN real y por los dos perfiles de Bolsa del candidato, nunca por datos
-- proporcionados por el cliente. Requiere Contexto-12, AUT-15, B-60 y AD3-115.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:migracion:000116',0));
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));

DO $preimagen$
DECLARE f regprocedure := pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
    OR f IS NULL
    OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_proc WHERE oid=f
       AND proowner='vec_autorizacion_atestada_v3_propietario'::regrole
       AND prosecdef AND proconfig=ARRAY['search_path=pg_catalog','lock_timeout=2s'])
    OR pg_catalog.to_regclass('vec_autorizacion_atestada_v3.atestacion_decision_v3_externa') IS NOT NULL
    OR pg_catalog.to_regclass('vec_autorizacion_atestada_v3.consumo_decision_v3_externa') IS NOT NULL
    OR pg_catalog.to_regclass('vec_autorizacion_atestada_v3.auditoria_consumo_v3_externa') IS NOT NULL
    OR pg_catalog.to_regclass('vec_autorizacion_atestada_v3.control_cadena_auditoria_externa') IS NOT NULL
    OR pg_catalog.to_regclass('vec_autorizacion.decision_concedida_contexto_actor_v3_externa') IS NULL
    OR pg_catalog.to_regprocedure('vec_autorizacion.registrar_y_revalidar_decision_contexto_actor_externa_v3(bytea,bytea,numeric,numeric)') IS NULL
    OR pg_catalog.to_regprocedure('vec_autorizacion.revalidar_decision_contexto_actor_externa_v3_viva(bytea,bytea,numeric,numeric)') IS NULL
    OR NOT pg_catalog.has_function_privilege('vec_autorizacion_atestada_v3_propietario',
       'vec_autorizacion.registrar_y_revalidar_decision_contexto_actor_externa_v3(bytea,bytea,numeric,numeric)','EXECUTE')
    OR NOT pg_catalog.has_function_privilege('vec_autorizacion_atestada_v3_propietario',
       'vec_autorizacion.revalidar_decision_contexto_actor_externa_v3_viva(bytea,bytea,numeric,numeric)','EXECUTE')
    OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname='vec_bolsa_llamamientos_portal_externo'
       AND NOT rolcanlogin AND NOT rolsuper AND NOT rolbypassrls)
 THEN RAISE EXCEPTION 'AD3-116: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $preimagen$;

SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
CREATE TABLE vec_autorizacion_atestada_v3.atestacion_decision_v3_externa
 (LIKE vec_autorizacion_atestada_v3.atestacion_decision_v3 INCLUDING ALL);
CREATE TABLE vec_autorizacion_atestada_v3.consumo_decision_v3_externa
 (LIKE vec_autorizacion_atestada_v3.consumo_decision_v3 INCLUDING ALL);
CREATE TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3_externa
 (LIKE vec_autorizacion_atestada_v3.auditoria_consumo_v3 INCLUDING ALL);
CREATE TABLE vec_autorizacion_atestada_v3.control_cadena_auditoria_externa
 (LIKE vec_autorizacion_atestada_v3.control_cadena_auditoria INCLUDING ALL);
INSERT INTO vec_autorizacion_atestada_v3.control_cadena_auditoria_externa
 (control_id,secuencia,cabeza_sha256,actualizada_en)
 VALUES (true,0,pg_catalog.repeat('0',64),pg_catalog.clock_timestamp());

-- El propietario AD3 carece deliberadamente de REFERENCES sobre AUT. Solo
-- el migrador superusuario crea esta FK; la función de consumo no hereda ese
-- privilegio y nunca puede leer la tabla de decisiones de AUT directamente.
RESET ROLE;
ALTER TABLE vec_autorizacion_atestada_v3.atestacion_decision_v3_externa
 ADD CONSTRAINT atestacion_decision_v3_externa_decision_ref_fkey
 FOREIGN KEY (decision_ref)
 REFERENCES vec_autorizacion.decision_concedida_contexto_actor_v3_externa(decision_ref);
ALTER TABLE vec_autorizacion_atestada_v3.consumo_decision_v3_externa
 ADD CONSTRAINT consumo_decision_v3_externa_atestacion_fkey
 FOREIGN KEY (decision_ref,efecto_ref,huella_efecto_sha256)
 REFERENCES vec_autorizacion_atestada_v3.atestacion_decision_v3_externa
 (decision_ref,efecto_ref,huella_efecto_sha256);
ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3_externa
 ADD CONSTRAINT auditoria_consumo_v3_externa_consumo_fkey
 FOREIGN KEY (decision_ref,efecto_ref,huella_efecto_sha256)
 REFERENCES vec_autorizacion_atestada_v3.consumo_decision_v3_externa
 (decision_ref,efecto_ref,huella_efecto_sha256);

SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
DO $proteccion$
DECLARE n text;
BEGIN
 FOREACH n IN ARRAY ARRAY[
   'atestacion_decision_v3_externa','consumo_decision_v3_externa',
   'auditoria_consumo_v3_externa'] LOOP
  EXECUTE pg_catalog.format('CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_autorizacion_atestada_v3.%I FOR EACH ROW EXECUTE FUNCTION vec_autorizacion_atestada_v3.rechazar_mutacion()',n);
  EXECUTE pg_catalog.format('CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_autorizacion_atestada_v3.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion_atestada_v3.rechazar_truncado()',n);
 END LOOP;
 FOREACH n IN ARRAY ARRAY[
   'atestacion_decision_v3_externa','consumo_decision_v3_externa',
   'auditoria_consumo_v3_externa','control_cadena_auditoria_externa'] LOOP
  EXECUTE pg_catalog.format('ALTER TABLE vec_autorizacion_atestada_v3.%I ENABLE ROW LEVEL SECURITY',n);
  EXECUTE pg_catalog.format('ALTER TABLE vec_autorizacion_atestada_v3.%I FORCE ROW LEVEL SECURITY',n);
  EXECUTE pg_catalog.format('CREATE POLICY propietario_exacto ON vec_autorizacion_atestada_v3.%I FOR ALL TO vec_autorizacion_atestada_v3_propietario USING (current_user=%L) WITH CHECK (current_user=%L)',n,'vec_autorizacion_atestada_v3_propietario','vec_autorizacion_atestada_v3_propietario');
  EXECUTE pg_catalog.format('REVOKE ALL ON TABLE vec_autorizacion_atestada_v3.%I FROM PUBLIC,vec_autorizacion_atestada_v3_emisor,vec_autorizacion_atestada_v3_consumidor',n);
  EXECUTE pg_catalog.format('REVOKE ALL ON TYPE vec_autorizacion_atestada_v3.%I FROM PUBLIC,vec_autorizacion_atestada_v3_emisor,vec_autorizacion_atestada_v3_consumidor',n);
 END LOOP;
END $proteccion$;

-- Clonar la validación del núcleo instalado evita que el carril externo
-- pierda las comprobaciones de ligadura, HMAC, gobierno y vigencia. Solo se
-- sustituyen el guard de LOGIN, dos funciones AUT y las cuatro tablas de
-- historia. Se exige una preimagen inequívoca antes de ejecutar el clon.
DO $carril$
DECLARE f regprocedure := 'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 original text; externo text; nuevo text; inicio integer; fin integer; marca text;
 nombres text[] := ARRAY['atestacion_decision_v3','consumo_decision_v3','auditoria_consumo_v3','control_cadena_auditoria'];
 n text;
 guardia text := $g$BEGIN
    IF pg_catalog.current_setting('transaction_isolation') <> 'serializable'
       OR pg_catalog.current_setting('transaction_read_only') <> 'off'
       OR pg_catalog.current_setting('TimeZone') <> 'UTC'
       OR current_user <> 'vec_autorizacion_atestada_v3_propietario'
       OR session_user <> 'vec_externo_bolsa_desarrollo'
       OR p_perfil_mutacion IS NULL
       OR p_perfil_mutacion NOT IN ('consulta_participaciones_propias_bolsa','portal_candidato_bolsa')
       OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles r WHERE r.rolname=session_user
          AND r.rolcanlogin AND r.rolinherit AND NOT r.rolsuper AND NOT r.rolcreatedb
          AND NOT r.rolcreaterole AND NOT r.rolreplication AND NOT r.rolbypassrls)
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_auth_members m
           WHERE m.member=session_user::regrole) <> 1
       OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m
          WHERE m.member=session_user::regrole
            AND m.roleid='vec_bolsa_llamamientos_portal_externo'::regrole
            AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)
       OR EXISTS (WITH RECURSIVE roles(rol_id) AS (
            SELECT m.roleid FROM pg_catalog.pg_auth_members m
             WHERE m.member=session_user::regrole
            UNION
            SELECT m.roleid FROM pg_catalog.pg_auth_members m
             JOIN roles r ON r.rol_id=m.member)
           SELECT 1 FROM roles
            WHERE rol_id<>'vec_bolsa_llamamientos_portal_externo'::regrole)
    THEN
        RAISE EXCEPTION USING ERRCODE='42501', MESSAGE='consumo externo VEC-AD-3 rechazado';
    END IF;
$g$;
 despacho text := $d$BEGIN
    IF session_user = 'vec_externo_bolsa_desarrollo' THEN
        IF p_perfil_mutacion NOT IN ('consulta_participaciones_propias_bolsa','portal_candidato_bolsa')
           OR p_perfil_mutacion IS NULL THEN
            RAISE EXCEPTION USING ERRCODE='42501', MESSAGE='perfil externo VEC-AD-3 rechazado';
        END IF;
        RETURN QUERY SELECT q.* FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_externa_interna(
            p_perfil_mutacion,p_capacidad_canonica,p_decision_canonica,p_motivo_canonico,
            p_contexto_actor_canonico,p_persona_version,p_perfil_version,p_payload_vec_ad_3,
            p_sobre_cose_sign1,p_evidencia_verificacion,p_raiz_publica_spki) q;
        RETURN;
    END IF;
$d$;
BEGIN
 SELECT pg_catalog.pg_get_functiondef(f) INTO STRICT original;
 marca := E'BEGIN\n    IF pg_catalog.current_setting(''transaction_isolation'')';
 inicio := pg_catalog.strpos(original,marca);
 fin := pg_catalog.strpos(original,'    SELECT setting::numeric INTO v_statement');
 IF inicio=0 OR fin<=inicio
    OR (pg_catalog.length(original)-pg_catalog.length(pg_catalog.replace(original,marca,'')))<>pg_catalog.length(marca)
    OR pg_catalog.strpos(original,'CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(')<>1
    OR pg_catalog.strpos(original,'''portal_candidato_bolsa''')=0
    OR pg_catalog.strpos(original,'''consulta_participaciones_propias_bolsa''')=0
    OR pg_catalog.strpos(original,'''vec_bolsa_llamamientos_portal_externo''')=0
    OR pg_catalog.strpos(original,'consumir_decision_mutacion_v3_externa_interna')<>0
    OR pg_catalog.strpos(original,'registrar_y_revalidar_decision_contexto_actor_v3(')=0
    OR pg_catalog.strpos(original,'revalidar_decision_contexto_actor_v3_viva(')=0
 THEN RAISE EXCEPTION 'AD3-116: núcleo incompatible' USING ERRCODE='55000'; END IF;
 FOREACH n IN ARRAY nombres LOOP
  IF pg_catalog.strpos(original,'vec_autorizacion_atestada_v3.'||n)=0 THEN
   RAISE EXCEPTION 'AD3-116: falta tabla %',n USING ERRCODE='55000';
  END IF;
 END LOOP;
 externo := pg_catalog.replace(original,
   'CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(',
   'CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_externa_interna(');
 inicio := pg_catalog.strpos(externo,marca);
 fin := pg_catalog.strpos(externo,'    SELECT setting::numeric INTO v_statement');
 externo := pg_catalog.substr(externo,1,inicio-1)||guardia||pg_catalog.substr(externo,fin);
 externo := pg_catalog.replace(externo,
   'registrar_y_revalidar_decision_contexto_actor_v3(',
   'registrar_y_revalidar_decision_contexto_actor_externa_v3(');
 externo := pg_catalog.replace(externo,
   'revalidar_decision_contexto_actor_v3_viva(',
   'revalidar_decision_contexto_actor_externa_v3_viva(');
 -- El replay externo exige autorización viva antes de revelar el recibo.
 -- El núcleo histórico conserva intacta su propia semántica de recuperación.
 IF pg_catalog.strpos(externo,E'        RETURN QUERY SELECT\n            v_replay.decision_ref')=0 THEN
  RAISE EXCEPTION 'AD3-116: replay incompatible' USING ERRCODE='55000';
 END IF;
 externo := pg_catalog.replace(externo,
   E'        RETURN QUERY SELECT\n            v_replay.decision_ref',
   E'        v_revalidada_en := vec_autorizacion.revalidar_decision_contexto_actor_externa_v3_viva(\n            p_decision_canonica,p_motivo_canonico,p_persona_version,p_perfil_version);\n        IF v_revalidada_en IS NULL THEN\n            RAISE EXCEPTION USING ERRCODE = ''42501'', MESSAGE = ''replay externo VEC-AD-3 rechazado'';\n        END IF;\n        RETURN QUERY SELECT\n            v_replay.decision_ref');
 FOREACH n IN ARRAY nombres LOOP
  externo := pg_catalog.replace(externo,'vec_autorizacion_atestada_v3.'||n,
                                'vec_autorizacion_atestada_v3.'||n||'_externa');
 END LOOP;
 EXECUTE externo;
 nuevo := pg_catalog.replace(original,marca,despacho||'    IF pg_catalog.current_setting(''transaction_isolation'')');
 EXECUTE nuevo;
 IF pg_catalog.pg_get_functiondef(f) IS DISTINCT FROM nuevo
    OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_externa_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 THEN RAISE EXCEPTION 'AD3-116: postimagen incompatible' USING ERRCODE='55000'; END IF;
END $carril$;

REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_externa_interna(
 text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 FROM PUBLIC,vec_autorizacion_atestada_v3_consumidor,vec_autorizacion_atestada_v3_emisor;
COMMIT;
