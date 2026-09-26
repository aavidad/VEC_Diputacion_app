\set ON_ERROR_STOP on
-- Documentos-5: admite como principal_ref el identificador que da el vínculo
-- de autenticación V2 real («per_» y un token de 22 a 128 caracteres
-- [A-Za-z0-9_-], como referenciaOpacaContextoActorValida en Go), además de
-- las referencias opacas que ya admitía.
--
-- 000001 y 000003 exigían referencia_opaca_v1(principal_ref) en documento y
-- referencia_externa. El PDP V3 liga la decisión al principal del vínculo y
-- consumir_v3_v2 exige que p_auth->>'principal_id' sea ese mismo valor, así
-- que con la composición real el INSERT violaba el CHECK (23514) y la API
-- respondía 422 contenido_no_valido aunque la consulta (sin ese CHECK en
-- auditoria_operacion) funcionara. No cambia ninguna función de efecto ni
-- sus huellas (000004), ni reescribe filas: solo sustituye los dos CHECK.
-- Conserva 000001–000004 e historia.
BEGIN;
SET LOCAL ROLE vec_documentos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_documentos:migracion:000005',0));
DO $pre$
BEGIN
 IF current_user<>'vec_documentos_propietario'
    OR to_regprocedure('vec_documentos.huella_efecto_v1(bytea)') IS NULL
    OR to_regclass('vec_documentos.denegacion_frontera') IS NULL
    OR to_regprocedure('vec_documentos.principal_ref_v1(text)') IS NOT NULL
    OR (SELECT pg_get_constraintdef(oid) FROM pg_constraint WHERE conname='documento_principal_ref_check' AND conrelid='vec_documentos.documento'::regclass)
       IS DISTINCT FROM 'CHECK (vec_documentos.referencia_opaca_v1(principal_ref))'
    OR (SELECT pg_get_constraintdef(oid) FROM pg_constraint WHERE conname='referencia_externa_principal_ref_check' AND conrelid='vec_documentos.referencia_externa'::regclass)
       IS DISTINCT FROM 'CHECK (vec_documentos.referencia_opaca_v1(principal_ref))'
 THEN RAISE EXCEPTION 'Documentos-5: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

-- Principal de un efecto documental: referencia opaca o principal del vínculo V2.
CREATE FUNCTION vec_documentos.principal_ref_v1(p text) RETURNS boolean
LANGUAGE sql IMMUTABLE SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT vec_documentos.referencia_opaca_v1(p)
  OR (p IS NOT NULL AND p ~ '^per_[A-Za-z0-9_-]{22,128}$')
 $f$;
REVOKE ALL ON FUNCTION vec_documentos.principal_ref_v1(text) FROM PUBLIC;

ALTER TABLE vec_documentos.documento
 DROP CONSTRAINT documento_principal_ref_check,
 ADD CONSTRAINT documento_principal_ref_check CHECK(vec_documentos.principal_ref_v1(principal_ref));
ALTER TABLE vec_documentos.referencia_externa
 DROP CONSTRAINT referencia_externa_principal_ref_check,
 ADD CONSTRAINT referencia_externa_principal_ref_check CHECK(vec_documentos.principal_ref_v1(principal_ref));
COMMIT;
