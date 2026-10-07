\set ON_ERROR_STOP on
-- DOWN Personal26 preparado. Nunca se ejecuta sobre recibos conservados.
-- No modifica Personal16, CA7 ni su historia; AD149 se retiraría después.
BEGIN;
SET LOCAL ROLE vec_personal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC'; SET LOCAL lock_timeout='5s'; SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal:migracion:000026:crn11',0));
LOCK TABLE vec_personal.recibo_vinculo_propio_crn11 IN ACCESS EXCLUSIVE MODE;
DO $historia$
DECLARE f oid:=to_regprocedure('vec_personal.consultar_vinculo_propio_historico_crn11_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
BEGIN
 IF current_user<>'vec_personal_propietario' OR f IS NULL
    OR EXISTS(SELECT 1 FROM vec_personal.recibo_vinculo_propio_crn11) THEN
  RAISE EXCEPTION 'Personal26: historia impide DOWN' USING ERRCODE='55000';
 END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f
      AND p.proowner='vec_personal_propietario'::regrole AND p.prosecdef
      AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='ab9dc0b7838bfd0fa580239cee7f6ac1e8506ccfcb98d4af5201cf0720dab8d7'
      AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s'])
    OR EXISTS(SELECT 1 FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
       WHERE p.oid=f AND (a.grantee NOT IN(p.proowner,'vec_personal_ejecutor'::regrole)
          OR a.grantor<>p.proowner OR a.privilege_type<>'EXECUTE' OR a.is_grantable)) THEN
  RAISE EXCEPTION 'Personal26: postimagen propia incompatible' USING ERRCODE='55000';
 END IF;
END $historia$;
REVOKE ALL ON FUNCTION vec_personal.consultar_vinculo_propio_historico_crn11_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC,vec_personal_ejecutor;
DROP FUNCTION vec_personal.consultar_vinculo_propio_historico_crn11_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP TABLE vec_personal.recibo_vinculo_propio_crn11;
COMMIT;
