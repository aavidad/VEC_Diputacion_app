\set ON_ERROR_STOP on
-- Documentos-14. Sólo los campos expediente/ámbito admiten una referencia
-- tipada. Los demás identificadores conservan referencia_opaca_v1 sin cambios.
BEGIN;
SET LOCAL ROLE vec_documentos_propietario;
SET LOCAL search_path=pg_catalog, pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_documentos:migracion:000014',0));

DO $pre$
DECLARE t text;
BEGIN
 IF current_user<>'vec_documentos_propietario'
    OR to_regclass('vec_documentos.modulo_referencia_expediente_v1') IS NOT NULL
    OR to_regprocedure('vec_documentos.referencia_expediente_v1(text)') IS NOT NULL
    OR to_regprocedure('vec_documentos.referencia_expediente_modulo_v1(text,text)') IS NOT NULL
    OR to_regclass('vec_documentos.documento') IS NULL
    OR to_regclass('vec_documentos.referencia_externa') IS NULL
    OR to_regclass('vec_documentos.reserva_original_firmable') IS NULL
    OR to_regprocedure('vec_documentos.rechazar_mutacion_v1()') IS NULL
 THEN RAISE EXCEPTION 'Documentos-14: preimagen ausente o ya instalada' USING ERRCODE='55000'; END IF;
 FOREACH t IN ARRAY ARRAY['documento','referencia_externa','reserva_original_firmable'] LOOP
  IF (SELECT pg_get_constraintdef(c.oid) FROM pg_constraint c
      WHERE c.conrelid=('vec_documentos.'||t)::regclass
        AND c.conname=t||'_expediente_ref_check')
     IS DISTINCT FROM 'CHECK (vec_documentos.referencia_opaca_v1(expediente_ref))'
  THEN RAISE EXCEPTION 'Documentos-14: CHECK previo incompatible en %', t USING ERRCODE='55000'; END IF;
 END LOOP;
END $pre$;

-- Catálogo cerrado y versionado. Una ampliación futura requiere otra migración
-- y el validador Go correspondiente; el ejecutor nunca puede publicar módulos.
CREATE TABLE vec_documentos.modulo_referencia_expediente_v1 (
 codigo text PRIMARY KEY CHECK(codigo='ct'),
 modulo_id text NOT NULL UNIQUE CHECK(modulo_id='contratacion_temporal'),
 version integer NOT NULL CHECK(version=1),
 patron_id text NOT NULL CHECK(patron_id='^[0-9a-f]{64}$')
);
REVOKE ALL ON TABLE vec_documentos.modulo_referencia_expediente_v1 FROM PUBLIC,vec_documentos_ejecutor;
INSERT INTO vec_documentos.modulo_referencia_expediente_v1(codigo,modulo_id,version,patron_id)
 VALUES('ct','contratacion_temporal',1,'^[0-9a-f]{64}$');
ALTER TABLE vec_documentos.modulo_referencia_expediente_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_documentos.modulo_referencia_expediente_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY lectura_catalogo ON vec_documentos.modulo_referencia_expediente_v1
 FOR SELECT TO vec_documentos_propietario USING (true);
CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_documentos.modulo_referencia_expediente_v1
 FOR EACH ROW EXECUTE FUNCTION vec_documentos.rechazar_mutacion_v1();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_documentos.modulo_referencia_expediente_v1
 FOR EACH STATEMENT EXECUTE FUNCTION vec_documentos.rechazar_mutacion_v1();

CREATE FUNCTION vec_documentos.referencia_expediente_v1(p text) RETURNS boolean
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog, pg_temp SET row_security=on AS $f$
 SELECT coalesce(vec_documentos.referencia_opaca_v1(p) OR (
   p ~ '^expediente:[a-z][a-z0-9_]{1,31}:[0-9a-f]{64}$'
   AND EXISTS (SELECT 1 FROM vec_documentos.modulo_referencia_expediente_v1 c
     WHERE c.codigo=split_part(p,':',2) AND c.version=1
       AND split_part(p,':',3) ~ c.patron_id)),false)
$f$;
REVOKE ALL ON FUNCTION vec_documentos.referencia_expediente_v1(text) FROM PUBLIC;

CREATE FUNCTION vec_documentos.referencia_expediente_modulo_v1(modulo text,p text) RETURNS boolean
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog, pg_temp SET row_security=on AS $f$
 SELECT coalesce(vec_documentos.referencia_opaca_v1(p) OR (
   p ~ '^expediente:[a-z][a-z0-9_]{1,31}:[0-9a-f]{64}$'
   AND EXISTS (SELECT 1 FROM vec_documentos.modulo_referencia_expediente_v1 c
     WHERE c.modulo_id=modulo AND c.codigo=split_part(p,':',2) AND c.version=1
       AND split_part(p,':',3) ~ c.patron_id)),false)
$f$;
REVOKE ALL ON FUNCTION vec_documentos.referencia_expediente_modulo_v1(text,text) FROM PUBLIC;

ALTER TABLE vec_documentos.documento DROP CONSTRAINT documento_expediente_ref_check;
ALTER TABLE vec_documentos.documento ADD CONSTRAINT documento_expediente_ref_check
 CHECK(vec_documentos.referencia_expediente_modulo_v1(modulo_id,expediente_ref));
ALTER TABLE vec_documentos.referencia_externa DROP CONSTRAINT referencia_externa_expediente_ref_check;
ALTER TABLE vec_documentos.referencia_externa ADD CONSTRAINT referencia_externa_expediente_ref_check
 CHECK(vec_documentos.referencia_expediente_modulo_v1(modulo_id,expediente_ref));
ALTER TABLE vec_documentos.reserva_original_firmable DROP CONSTRAINT reserva_original_firmable_expediente_ref_check;
ALTER TABLE vec_documentos.reserva_original_firmable ADD CONSTRAINT reserva_original_firmable_expediente_ref_check
 CHECK(vec_documentos.referencia_expediente_modulo_v1(modulo_id,expediente_ref));

-- Reemplazo puntual de las guardas de ámbito instaladas. pg_get_functiondef
-- conserva firma, owner, SECURITY DEFINER, search_path, opciones y ACL; la
-- marca debe aparecer exactamente una vez para impedir una sustitución parcial.
DO $funciones$
DECLARE firma text; marca text; sustituta text; definicion text; n integer;
        huella_esperada text; huella_actual text;
BEGIN
 FOREACH firma IN ARRAY ARRAY[
  'vec_documentos.obtener_original_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_documentos.preparar_notificacion_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_documentos.preparar_notificacion_v2(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_documentos.reservar_original_firmable_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
 ] LOOP
  IF to_regprocedure(firma) IS NULL THEN
   RAISE EXCEPTION 'Documentos-14: función ausente %',firma USING ERRCODE='55000'; END IF;
  huella_esperada:=CASE
   WHEN firma LIKE '%obtener_original_v1%' THEN '0383e9203d7cfc29d72e3b8e498c2914a73109a9c2c1eefe23c9ac6e964096d5'
   WHEN firma LIKE '%preparar_notificacion_v1%' THEN '931ce17a38146ae51ea867de6f64dcc22c0c5eff82ab3308aac4a73d61c3c37d'
   WHEN firma LIKE '%preparar_notificacion_v2%' THEN '5475c032183af3c80deb114be8b607f5981a8f80bc67cacc860f7f8e873dc9d3'
   WHEN firma LIKE '%reservar_original_firmable_v1%' THEN '876ad1d9b6f0e3c4b5ca37b9afe5251a30d98a580f14788d1d940901e6875489'
  END;
  SELECT encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') INTO huella_actual
   FROM pg_proc p WHERE p.oid=to_regprocedure(firma);
  IF huella_actual IS DISTINCT FROM huella_esperada THEN
   RAISE EXCEPTION 'Documentos-14: cuerpo previo incompatible %',firma USING ERRCODE='55000'; END IF;
  definicion:=pg_get_functiondef(to_regprocedure(firma));
  IF firma LIKE '%reservar_original_firmable_v1%' THEN
   marca:='vec_documentos.referencia_opaca_v1(m->>''expediente_ref'')';
   sustituta:='vec_documentos.referencia_expediente_modulo_v1(m->>''modulo_id'',m->>''expediente_ref'')';
  ELSE
   marca:='vec_documentos.referencia_opaca_v1(p_auth->>''ambito_ref'')';
   sustituta:='vec_documentos.referencia_expediente_v1(p_auth->>''ambito_ref'')';
  END IF;
  n:=(length(definicion)-length(replace(definicion,marca,'')))/length(marca);
  IF n<>1 THEN RAISE EXCEPTION 'Documentos-14: marca de ámbito/expediente % veces en %',n,firma USING ERRCODE='55000'; END IF;
  EXECUTE replace(definicion,marca,sustituta);
 END LOOP;
END $funciones$;
COMMIT;
