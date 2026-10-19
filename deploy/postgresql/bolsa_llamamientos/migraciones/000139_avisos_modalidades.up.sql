\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000139',0));

DO $pre$
BEGIN
 IF current_setting('server_version_num')::integer<180000
 OR to_regclass('vec_bolsa_llamamientos.preferencia_modalidades_version') IS NULL
 OR to_regclass('vec_bolsa_llamamientos.vinculo_candidato') IS NULL
 OR to_regclass('vec_bolsa_llamamientos.oferta_publicada') IS NULL
 OR to_regclass('vec_bolsa_llamamientos.aviso_oferta_destinatario') IS NULL
 OR NOT EXISTS (SELECT 1 FROM pg_class c
   WHERE c.oid='vec_bolsa_llamamientos.preferencia_modalidades_version'::regclass
   AND c.relowner='vec_bolsa_llamamientos_propietario'::regrole
   AND c.relrowsecurity AND c.relforcerowsecurity)
 OR NOT EXISTS (SELECT 1 FROM pg_trigger t WHERE t.tgrelid='vec_bolsa_llamamientos.oferta_publicada'::regclass
   AND t.tgname='crear_aviso_oferta_b129' AND t.tgfoid='vec_bolsa_llamamientos.crear_aviso_oferta_b129()'::regprocedure
   AND t.tgenabled='O' AND t.tgtype=5 AND t.tgqual IS NULL
   AND t.tgnargs=0 AND NOT t.tgisinternal)
 OR (SELECT encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex')
      FROM pg_proc p WHERE p.oid='vec_bolsa_llamamientos.crear_aviso_oferta_b129()'::regprocedure
      AND p.proowner='vec_bolsa_llamamientos_propietario'::regrole AND p.prosecdef
      AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on']::text[]
      AND p.proacl::text='{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario}')
    IS DISTINCT FROM '464f8f0ce0c2b194c631ea0eea0d483737d60effd17db36a00bbc9a7091c7a8d'
 THEN RAISE EXCEPTION 'B139: PARO preimagen o dependencias incompatibles' USING ERRCODE='55000'; END IF;
END $pre$;

LOCK TABLE vec_bolsa_llamamientos.oferta_publicada,
 vec_bolsa_llamamientos.preferencia_modalidades_version IN SHARE ROW EXCLUSIVE MODE;

CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.crear_aviso_oferta_b129()
 RETURNS trigger
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
AS $function$
BEGIN
 INSERT INTO vec_bolsa_llamamientos.aviso_oferta_cabecera(oferta_ref,publicada_en)
 VALUES(NEW.oferta_ref,NEW.publicada_en);
 INSERT INTO vec_bolsa_llamamientos.aviso_oferta_destinatario
   (oferta_ref,participacion_ref,message_id,actualizado_en)
 SELECT NEW.oferta_ref,s.participacion_ref,
   '<'||encode(sha256(convert_to(NEW.oferta_ref||chr(31)||s.participacion_ref,'UTF8')),'hex')||'@vec.dipgra.local>',
   NEW.publicada_en
 FROM vec_bolsa_llamamientos.situaciones_en_v1(NEW.publicada_en) s
 LEFT JOIN vec_bolsa_llamamientos.vinculo_candidato vc
   ON vc.participacion_ref=s.participacion_ref
 LEFT JOIN LATERAL (
   SELECT p.modalidades FROM vec_bolsa_llamamientos.preferencia_modalidades_version p
   WHERE p.candidato_ref=vc.candidato_ref ORDER BY p.version DESC LIMIT 1
 ) pm ON true
 WHERE s.bolsa_ref=NEW.bolsa_ref AND s.situacion='disponible'
   AND (NOT (NEW.datos ? 'modalidad_clave') OR pm.modalidades IS NULL
        OR (NEW.datos->>'modalidad_clave')=ANY(pm.modalidades))
   AND EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.constitucion c
     JOIN vec_bolsa_llamamientos.bolsa_constituida b
       ON b.bolsa_ref=c.bolsa_ref AND b.version=c.version_bolsa
     JOIN vec_bolsa_llamamientos.constitucion_entrada e
       ON e.instantanea_ref=c.instantanea_ref
      AND e.version_instantanea=c.version_instantanea
      AND e.participacion_ref=s.participacion_ref
     WHERE c.bolsa_ref=NEW.bolsa_ref AND b.estado='vigente'
       AND b.vigente_desde<=NEW.publicada_en
       AND (b.vigente_hasta IS NULL OR NEW.publicada_en<b.vigente_hasta));
 RETURN NEW;
END $function$;

DO $post$
BEGIN
 IF (SELECT encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex')
      FROM pg_proc p WHERE p.oid='vec_bolsa_llamamientos.crear_aviso_oferta_b129()'::regprocedure
      AND p.proowner='vec_bolsa_llamamientos_propietario'::regrole AND p.prosecdef
      AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on']::text[]
      AND p.proacl::text='{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario}')
    IS DISTINCT FROM '56206ed1c71c99becfe1bd122ff44bd0b4b7e50bae5e4d23fad0a7d4a7aca8d5'
 THEN RAISE EXCEPTION 'B139: postimagen de aviso incompatible' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
