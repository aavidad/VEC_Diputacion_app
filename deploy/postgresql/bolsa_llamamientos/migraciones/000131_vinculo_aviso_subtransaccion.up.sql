\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000131',0));

DO $pre$
DECLARE p record;c record;
BEGIN
 SELECT pg_get_userbyid(x.proowner) propietario,x.proacl,x.proconfig,
   encode(sha256(convert_to(pg_get_functiondef(x.oid),'UTF8')),'hex') huella
 INTO p FROM pg_proc x WHERE x.oid=to_regprocedure(
  'vec_bolsa_llamamientos.vincular_despacho_aviso_v1(text,text,text,bytea,text)');
 SELECT relowner,relrowsecurity,relforcerowsecurity,relacl INTO c
 FROM pg_class WHERE oid=to_regclass('vec_bolsa_llamamientos.aviso_oferta_cabecera');
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR current_setting('server_version_num')::integer<180000
    OR p.huella IS DISTINCT FROM 'cbca3ac03558e0b9e7deb5258af41bdc156f05f3b43bc8369fb6bcf4eadc426c'
    OR p.propietario IS DISTINCT FROM 'vec_bolsa_llamamientos_propietario'
    OR p.proacl::text IS DISTINCT FROM '{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario,vec_bolsa_llamamientos_ejecutor=X/vec_bolsa_llamamientos_propietario}'
    OR p.proconfig::text IS DISTINCT FROM '{"search_path=pg_catalog, pg_temp",row_security=on,lock_timeout=2s}'
    OR c.relowner IS DISTINCT FROM 'vec_bolsa_llamamientos_propietario'::regrole
    OR c.relrowsecurity IS DISTINCT FROM true OR c.relforcerowsecurity IS DISTINCT FROM true
    OR c.relacl::text IS DISTINCT FROM '{vec_bolsa_llamamientos_propietario=arwdDxtm/vec_bolsa_llamamientos_propietario}'
    OR EXISTS(SELECT 1 FROM pg_attribute WHERE attrelid='vec_bolsa_llamamientos.aviso_oferta_cabecera'::regclass
      AND attname='publicacion_xid' AND NOT attisdropped)
 THEN RAISE EXCEPTION 'B131: PARO preimagen de vínculo incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

-- DEFAULT usa el identificador principal también dentro de un bloque EXCEPTION.
-- Las cabeceras anteriores conservan NULL; no se reescribe ninguna publicación.
ALTER TABLE vec_bolsa_llamamientos.aviso_oferta_cabecera ADD COLUMN publicacion_xid xid8;
ALTER TABLE vec_bolsa_llamamientos.aviso_oferta_cabecera
 ALTER COLUMN publicacion_xid SET DEFAULT pg_current_xact_id();

-- Definición literal de pg_get_functiondef instalada con preimagen B129 arriba.
-- Solo cambian los dos cotejos xmin: la oferta conserva actor y clave exactos;
-- la cabecera, de acceso exclusivo al propietario, acredita la transacción.
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.vincular_despacho_aviso_v1(p_oferta text, p_actor text, p_clave text, p_token_sha bytea, p_version text)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET lock_timeout TO '2s'
AS $function$
DECLARE o record;h record;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR session_user=current_user
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER') IS NOT TRUE
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER') IS TRUE
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR current_setting('statement_timeout')::interval NOT BETWEEN interval '1 millisecond' AND interval '15 seconds'
    OR current_setting('idle_in_transaction_session_timeout')::interval NOT BETWEEN interval '1 millisecond' AND interval '20 seconds'
    OR p_oferta IS NULL OR p_oferta !~ '^oferta:[0-9a-f]{64}$'
    OR p_actor !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR p_clave IS NULL OR octet_length(p_clave) NOT BETWEEN 8 AND 256
    OR octet_length(p_token_sha) IS DISTINCT FROM 32
    OR p_version IS NULL OR p_version !~ '^[A-Za-z0-9._:-]{1,128}$'
 THEN RAISE EXCEPTION 'B129: reserva de aviso inválida' USING ERRCODE='42501'; END IF;
 SELECT * INTO o FROM vec_bolsa_llamamientos.oferta_publicada
  WHERE oferta_ref=p_oferta AND actor_ref=p_actor AND clave_idempotencia=p_clave;
 IF NOT FOUND THEN RAISE EXCEPTION 'B129: reserva fuera de publicación' USING ERRCODE='42501'; END IF;
 SELECT * INTO h FROM vec_bolsa_llamamientos.aviso_oferta_cabecera
  WHERE oferta_ref=p_oferta AND publicacion_xid=pg_current_xact_id_if_assigned() FOR UPDATE;
 IF NOT FOUND OR h.huella_token IS NOT NULL OR h.version_token IS NOT NULL THEN
  RAISE EXCEPTION 'B129: reserva repetida o ajena' USING ERRCODE='42501';
 END IF;
 UPDATE vec_bolsa_llamamientos.aviso_oferta_cabecera
   SET huella_token=p_token_sha,version_token=p_version WHERE oferta_ref=p_oferta;
 RETURN vec_bolsa_llamamientos.proyectar_oferta_v2(p_oferta,clock_timestamp());
END $function$;

DO $post$
DECLARE p record;a record;
BEGIN
 SELECT pg_get_userbyid(x.proowner) propietario,x.proacl,x.proconfig,
   pg_get_functiondef(x.oid) definicion INTO p
 FROM pg_proc x WHERE x.oid='vec_bolsa_llamamientos.vincular_despacho_aviso_v1(text,text,text,bytea,text)'::regprocedure;
 SELECT t.atttypid,t.attnotnull,pg_get_expr(d.adbin,d.adrelid) defecto INTO a
 FROM pg_attribute t LEFT JOIN pg_attrdef d ON d.adrelid=t.attrelid AND d.adnum=t.attnum
 WHERE t.attrelid='vec_bolsa_llamamientos.aviso_oferta_cabecera'::regclass
   AND t.attname='publicacion_xid' AND NOT t.attisdropped;
 IF NOT FOUND OR a.atttypid IS DISTINCT FROM 'pg_catalog.xid8'::regtype
    OR a.attnotnull IS DISTINCT FROM false
    OR a.defecto IS DISTINCT FROM 'pg_current_xact_id()'
    OR p.propietario IS DISTINCT FROM 'vec_bolsa_llamamientos_propietario'
    OR p.proacl::text IS DISTINCT FROM '{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario,vec_bolsa_llamamientos_ejecutor=X/vec_bolsa_llamamientos_propietario}'
    OR p.proconfig::text IS DISTINCT FROM '{"search_path=pg_catalog, pg_temp",row_security=on,lock_timeout=2s}'
    OR position('publicacion_xid=pg_current_xact_id_if_assigned()' IN p.definicion)=0
    OR position('xmin::text' IN p.definicion)>0
 THEN RAISE EXCEPTION 'B131: PARO postimagen de vínculo incompatible' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
