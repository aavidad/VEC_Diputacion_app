\set ON_ERROR_STOP on
-- B86: proyecta una sola vez la completitud que B17 calcula sobre cada llamamiento.
-- La historia B13/B17 permanece inmutable; la proyeccion y su guardia son privadas.
BEGIN ISOLATION LEVEL READ COMMITTED;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='5min';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000086',0));

DO $pre$
DECLARE f oid:=pg_catalog.to_regprocedure('vec_bolsa_llamamientos.leer_llamamientos_en_curso_bolsas_v1()');
        v_hash text; v_metadata text; v_acl text;
BEGIN
 IF pg_catalog.current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999 THEN
   RAISE EXCEPTION 'B86: clave=server_version_num esperado=18xxxx obtenido=%',pg_catalog.current_setting('server_version_num') USING ERRCODE='55000';
 END IF;
 IF current_user<>'vec_bolsa_llamamientos_propietario' THEN
   RAISE EXCEPTION 'B86: clave=current_user esperado=vec_bolsa_llamamientos_propietario obtenido=%',current_user USING ERRCODE='42501';
 END IF;
 IF pg_catalog.to_regclass('vec_bolsa_llamamientos.llamamiento_emitido') IS NULL
    OR pg_catalog.to_regclass('vec_bolsa_llamamientos.contacto_participacion') IS NULL
    OR pg_catalog.to_regprocedure('vec_bolsa_llamamientos.contar_llamamientos_en_curso_v1(text)') IS NULL
    OR pg_catalog.to_regprocedure('vec_bolsa_llamamientos.listar_constituciones_v1()') IS NULL
    OR f IS NULL THEN
   RAISE EXCEPTION 'B86: clave=dependencias_B17_B85 esperado=presentes obtenido=%',
     pg_catalog.concat_ws(',',pg_catalog.to_regclass('vec_bolsa_llamamientos.llamamiento_emitido'),
       pg_catalog.to_regclass('vec_bolsa_llamamientos.contacto_participacion'),
       pg_catalog.to_regprocedure('vec_bolsa_llamamientos.contar_llamamientos_en_curso_v1(text)'),
       pg_catalog.to_regprocedure('vec_bolsa_llamamientos.listar_constituciones_v1()'),f) USING ERRCODE='55000';
 END IF;
 IF pg_catalog.to_regclass('vec_bolsa_llamamientos.completitud_correo_llamamiento') IS NOT NULL
    OR pg_catalog.to_regclass('vec_bolsa_llamamientos.coordinacion_completitud_correo') IS NOT NULL THEN
   RAISE EXCEPTION 'B86: clave=tablas_B86 esperado=ausentes obtenido=%',
     pg_catalog.concat_ws(',',pg_catalog.to_regclass('vec_bolsa_llamamientos.completitud_correo_llamamiento'),
       pg_catalog.to_regclass('vec_bolsa_llamamientos.coordinacion_completitud_correo')) USING ERRCODE='55000';
 END IF;
 SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p.prosrc,'UTF8')),'hex'),
   pg_catalog.concat_ws(',',p.proowner::pg_catalog.regrole::text,p.prosecdef::text,p.provolatile,
     pg_catalog.array_to_string(p.proconfig,';'))
 INTO v_hash,v_metadata FROM pg_catalog.pg_proc p WHERE p.oid=f;
 IF v_hash IS DISTINCT FROM '732e9998e9de10b5ccfc1bbf5b3b79432f80699e6ee3bfadb497d53d2429baf0' THEN
   RAISE EXCEPTION 'B86: clave=B85.prosrc_sha256 esperado=732e9998e9de10b5ccfc1bbf5b3b79432f80699e6ee3bfadb497d53d2429baf0 obtenido=%',v_hash USING ERRCODE='55000';
 END IF;
 IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_proc p WHERE p.oid=f
    AND p.proowner='vec_bolsa_llamamientos_propietario'::pg_catalog.regrole
    AND p.prosecdef AND p.provolatile='s'
    AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']) THEN
   RAISE EXCEPTION 'B86: clave=B85.metadata esperado=owner,SECDEF,STABLE,path+timeout obtenido=%',v_metadata USING ERRCODE='55000';
 END IF;
 SELECT pg_catalog.string_agg(
   (CASE WHEN a.grantee=0 THEN 'PUBLIC' ELSE a.grantee::pg_catalog.regrole::text END)
     ||'/'||a.grantor::pg_catalog.regrole::text||':'||a.privilege_type||':'||a.is_grantable::text,
   ',' ORDER BY a.grantee)
 INTO v_acl FROM pg_catalog.pg_proc p
 CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
 WHERE p.oid=f;
 IF (SELECT pg_catalog.count(*) FROM pg_catalog.pg_proc p
       CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,
         pg_catalog.acldefault('f',p.proowner))) a WHERE p.oid=f)<>2
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc p
       CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,
         pg_catalog.acldefault('f',p.proowner))) a WHERE p.oid=f
       AND (a.grantee NOT IN (p.proowner,'vec_bolsa_llamamientos_ejecutor'::pg_catalog.regrole)
         OR a.grantor<>p.proowner OR a.privilege_type<>'EXECUTE' OR a.is_grantable)) THEN
   RAISE EXCEPTION 'B86: clave=B85.ACL esperado=owner+ejecutor_EXECUTE_sin_grant_option obtenido=%',v_acl USING ERRCODE='55000';
 END IF;
 IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_trigger t WHERE t.tgrelid='vec_bolsa_llamamientos.llamamiento_emitido'::pg_catalog.regclass
      AND t.tgname='llamamiento_emitido_inmutable' AND t.tgenabled='O' AND NOT t.tgisinternal) THEN
   RAISE EXCEPTION 'B86: clave=B17.inmutabilidad esperado=O obtenido=%',
     coalesce((SELECT t.tgenabled::text FROM pg_catalog.pg_trigger t WHERE t.tgrelid='vec_bolsa_llamamientos.llamamiento_emitido'::pg_catalog.regclass
       AND t.tgname='llamamiento_emitido_inmutable'),'ausente') USING ERRCODE='55000';
 END IF;
 IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_trigger t WHERE t.tgrelid='vec_bolsa_llamamientos.contacto_participacion'::pg_catalog.regclass
      AND t.tgname='contacto_participacion_inmutable' AND t.tgenabled='O' AND NOT t.tgisinternal) THEN
   RAISE EXCEPTION 'B86: clave=B13.inmutabilidad esperado=O obtenido=%',
     coalesce((SELECT t.tgenabled::text FROM pg_catalog.pg_trigger t WHERE t.tgrelid='vec_bolsa_llamamientos.contacto_participacion'::pg_catalog.regclass
       AND t.tgname='contacto_participacion_inmutable'),'ausente') USING ERRCODE='55000';
 END IF;
END $pre$;

-- El bloqueo precede tanto al barrido historico como a los nuevos disparadores.
-- Ningun INSERT puede atravesar la ventana de instalacion sin proyectarse.
LOCK TABLE vec_bolsa_llamamientos.llamamiento_emitido IN SHARE ROW EXCLUSIVE MODE;
LOCK TABLE vec_bolsa_llamamientos.contacto_participacion IN SHARE ROW EXCLUSIVE MODE;

CREATE TABLE vec_bolsa_llamamientos.coordinacion_completitud_correo (
 llamamiento_ref text PRIMARY KEY CHECK(llamamiento_ref ~ '^llamamiento:[0-9a-f]{64}$'),
 revision bigint NOT NULL CHECK(revision>0)
);
ALTER TABLE vec_bolsa_llamamientos.coordinacion_completitud_correo ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.coordinacion_completitud_correo FORCE ROW LEVEL SECURITY;
CREATE POLICY coordinacion_completitud_correo_solo_propietario
 ON vec_bolsa_llamamientos.coordinacion_completitud_correo
 TO vec_bolsa_llamamientos_propietario
 USING(current_user='vec_bolsa_llamamientos_propietario')
 WITH CHECK(current_user='vec_bolsa_llamamientos_propietario');
REVOKE ALL ON vec_bolsa_llamamientos.coordinacion_completitud_correo FROM PUBLIC;
REVOKE USAGE ON TYPE vec_bolsa_llamamientos.coordinacion_completitud_correo FROM PUBLIC;

CREATE TABLE vec_bolsa_llamamientos.completitud_correo_llamamiento (
 llamamiento_ref text PRIMARY KEY REFERENCES vec_bolsa_llamamientos.llamamiento_emitido(llamamiento_ref),
 bolsa_ref text NOT NULL
);
CREATE INDEX completitud_correo_llamamiento_bolsa
 ON vec_bolsa_llamamientos.completitud_correo_llamamiento(bolsa_ref,llamamiento_ref);
ALTER TABLE vec_bolsa_llamamientos.completitud_correo_llamamiento ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.completitud_correo_llamamiento FORCE ROW LEVEL SECURITY;
CREATE POLICY completitud_correo_llamamiento_solo_propietario
 ON vec_bolsa_llamamientos.completitud_correo_llamamiento
 TO vec_bolsa_llamamientos_propietario
 USING(current_user='vec_bolsa_llamamientos_propietario')
 WITH CHECK(current_user='vec_bolsa_llamamientos_propietario');
REVOKE ALL ON vec_bolsa_llamamientos.completitud_correo_llamamiento FROM PUBLIC;
REVOKE USAGE ON TYPE vec_bolsa_llamamientos.completitud_correo_llamamiento FROM PUBLIC;

CREATE FUNCTION vec_bolsa_llamamientos.proyectar_completitud_correo_insert_v1()
RETURNS trigger LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET lock_timeout='2s' AS $f$
DECLARE v_ref text;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR TG_OP<>'INSERT' OR TG_LEVEL<>'STATEMENT'
    OR TG_RELID NOT IN ('vec_bolsa_llamamientos.llamamiento_emitido'::pg_catalog.regclass,
      'vec_bolsa_llamamientos.contacto_participacion'::pg_catalog.regclass)
 THEN RAISE EXCEPTION 'B86: origen de proyeccion incompatible' USING ERRCODE='42501'; END IF;

 FOR v_ref IN
   SELECT DISTINCT n.llamamiento_ref FROM nuevas_completitud n
   WHERE n.llamamiento_ref ~ '^llamamiento:[0-9a-f]{64}$'
   ORDER BY n.llamamiento_ref
 LOOP
   -- Una escritura real serializa las carreras: RC relee tras esperar;
   -- RR/SER aborta la transaccion fuente si su instantanea es obsoleta.
   INSERT INTO vec_bolsa_llamamientos.coordinacion_completitud_correo(llamamiento_ref,revision)
     VALUES(v_ref,1)
   ON CONFLICT (llamamiento_ref) DO UPDATE
     SET revision=vec_bolsa_llamamientos.coordinacion_completitud_correo.revision+1;

   -- Sentencia VOLATILE separada del UPSERT: conserva exactamente B17.
   INSERT INTO vec_bolsa_llamamientos.completitud_correo_llamamiento(llamamiento_ref,bolsa_ref)
   SELECT l.llamamiento_ref,l.bolsa_ref
   FROM vec_bolsa_llamamientos.llamamiento_emitido l
   WHERE l.llamamiento_ref=v_ref
     AND (SELECT pg_catalog.count(*)
       FROM pg_catalog.jsonb_array_elements_text(l.participaciones) WITH ORDINALITY x(ref,ordinality)
       JOIN vec_bolsa_llamamientos.contacto_participacion c
         ON c.participacion_ref=x.ref AND c.llamamiento_ref=l.llamamiento_ref
        AND c.canal='correo'
        AND c.clave_idempotencia=l.clave_idempotencia||':correo:'||x.ordinality
        AND c.recibo_ref='recibo:contacto:'||pg_catalog.encode(pg_catalog.sha256(
          pg_catalog.convert_to(l.bolsa_ref||pg_catalog.chr(31)||l.clave_idempotencia||
            pg_catalog.chr(31)||x.ref,'UTF8')),'hex'))
       =pg_catalog.jsonb_array_length(l.participaciones)
   ON CONFLICT (llamamiento_ref) DO NOTHING;
 END LOOP;
 RETURN NULL;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.proyectar_completitud_correo_insert_v1() FROM PUBLIC;

CREATE TRIGGER proyectar_completitud_correo_emision
 AFTER INSERT ON vec_bolsa_llamamientos.llamamiento_emitido
 REFERENCING NEW TABLE AS nuevas_completitud
 FOR EACH STATEMENT EXECUTE FUNCTION vec_bolsa_llamamientos.proyectar_completitud_correo_insert_v1();
CREATE TRIGGER proyectar_completitud_correo_contacto
 AFTER INSERT ON vec_bolsa_llamamientos.contacto_participacion
 REFERENCING NEW TABLE AS nuevas_completitud
 FOR EACH STATEMENT EXECUTE FUNCTION vec_bolsa_llamamientos.proyectar_completitud_correo_insert_v1();

-- El backfill es unico y usa el mismo predicado B17, sin exigir resultado
-- 'enviado' ni bolsa_ref del contacto, que B17 tampoco exige.
INSERT INTO vec_bolsa_llamamientos.completitud_correo_llamamiento(llamamiento_ref,bolsa_ref)
SELECT l.llamamiento_ref,l.bolsa_ref
FROM vec_bolsa_llamamientos.llamamiento_emitido l
WHERE (SELECT pg_catalog.count(*)
  FROM pg_catalog.jsonb_array_elements_text(l.participaciones) WITH ORDINALITY x(ref,ordinality)
  JOIN vec_bolsa_llamamientos.contacto_participacion c
    ON c.participacion_ref=x.ref AND c.llamamiento_ref=l.llamamiento_ref
   AND c.canal='correo'
   AND c.clave_idempotencia=l.clave_idempotencia||':correo:'||x.ordinality
   AND c.recibo_ref='recibo:contacto:'||pg_catalog.encode(pg_catalog.sha256(
     pg_catalog.convert_to(l.bolsa_ref||pg_catalog.chr(31)||l.clave_idempotencia||
       pg_catalog.chr(31)||x.ref,'UTF8')),'hex'))
  =pg_catalog.jsonb_array_length(l.participaciones);

CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.leer_llamamientos_en_curso_bolsas_v1()
RETURNS TABLE(bolsa_ref text,llamamientos_en_curso bigint)
LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET lock_timeout='2s' AS $f$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR session_user=current_user
    OR NOT pg_catalog.pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER')
    OR pg_catalog.pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER')
    OR pg_catalog.pg_has_role(session_user,'vec_bolsa_llamamientos_migrador','MEMBER')
 THEN RAISE EXCEPTION 'B85: lectura de recuentos denegada' USING ERRCODE='42501'; END IF;
 RETURN QUERY
 WITH bolsas AS MATERIALIZED (
   SELECT DISTINCT k.bolsa_ref FROM vec_bolsa_llamamientos.listar_constituciones_v1() k
 )
 SELECT b.bolsa_ref,pg_catalog.count(m.llamamiento_ref)::bigint
 FROM bolsas b
 LEFT JOIN vec_bolsa_llamamientos.completitud_correo_llamamiento m ON m.bolsa_ref=b.bolsa_ref
 GROUP BY b.bolsa_ref ORDER BY b.bolsa_ref;
END $f$;

DO $post$
DECLARE f oid:='vec_bolsa_llamamientos.leer_llamamientos_en_curso_bolsas_v1()'::pg_catalog.regprocedure;
        v_acl text; v_metadata text;
BEGIN
 SELECT pg_catalog.string_agg(
   (CASE WHEN a.grantee=0 THEN 'PUBLIC' ELSE a.grantee::pg_catalog.regrole::text END)
     ||'/'||a.grantor::pg_catalog.regrole::text||':'||a.privilege_type||':'||a.is_grantable::text,
   ',' ORDER BY a.grantee)
 INTO v_acl FROM pg_catalog.pg_proc p
 CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
 WHERE p.oid=f;
 IF NOT pg_catalog.has_function_privilege('vec_bolsa_llamamientos_ejecutor',f,'EXECUTE')
    OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_proc p
       CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,
         pg_catalog.acldefault('f',p.proowner))) a WHERE p.oid=f)<>2
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc p
       CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,
         pg_catalog.acldefault('f',p.proowner))) a WHERE p.oid=f
       AND (a.grantee NOT IN (p.proowner,'vec_bolsa_llamamientos_ejecutor'::pg_catalog.regrole)
         OR a.grantor<>p.proowner OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
 THEN RAISE EXCEPTION 'B86: clave=B86.ACL esperado=owner+ejecutor_EXECUTE_sin_grant_option obtenido=%',v_acl USING ERRCODE='42501'; END IF;
 SELECT pg_catalog.concat_ws(',',p.proowner::pg_catalog.regrole::text,p.prosecdef::text,p.provolatile,
   pg_catalog.array_to_string(p.proconfig,';')) INTO v_metadata FROM pg_catalog.pg_proc p WHERE p.oid=f;
 IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_proc p WHERE p.oid=f
    AND p.proowner='vec_bolsa_llamamientos_propietario'::pg_catalog.regrole
    AND p.prosecdef AND p.provolatile='s'
    AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']) THEN
   RAISE EXCEPTION 'B86: clave=B86.metadata esperado=owner,SECDEF,STABLE,path+timeout obtenido=%',v_metadata USING ERRCODE='42501';
 END IF;
END $post$;
COMMIT;
