\set ON_ERROR_STOP on
-- DOWN Personal27 preparado; no ejecutarlo sobre historia conservada.
-- Requiere ausencia de recibos. Nunca retira auditoría ni roles comunes.
-- Conserva la fuente Personal17, sus versiones y sus triggers originales.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC'; SET LOCAL lock_timeout='5s'; SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal:migracion:000027:relacion-rpt',0));
SET LOCAL ROLE vec_personal_propietario;
LOCK TABLE vec_personal.relacion_servicio_historia IN SHARE ROW EXCLUSIVE MODE;
LOCK TABLE vec_personal.control_generacion_relacion_rpt,vec_personal.recibo_relacion_para_rpt
 IN ACCESS EXCLUSIVE MODE;
DO $historia$
DECLARE x record; f oid;
BEGIN
 IF current_user<>'vec_personal_propietario'
 OR EXISTS(SELECT 1 FROM vec_personal.recibo_relacion_para_rpt)
 THEN
  RAISE EXCEPTION 'Personal27: historia impide DOWN' USING ERRCODE='55000';
 END IF;
 FOR x IN SELECT * FROM (VALUES
  ('vec_personal.consultar_relacion_para_rpt_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)', '9e4733bcb994a3234d8261c107aeb4b49a9d04f9544302ac34f39c77c604a560', 'vec_personal_ejecutor'),
  ('vec_personal.avanzar_generacion_relacion_rpt_v1()', '9846a7e05c26bced4408ab6d9af9b815ff8e5754d833afa670c3fe10fcb68246', 'vec_personal_propietario')
 ) AS contratos(firma,fuente_sha256,permitido) LOOP
  f:=to_regprocedure(x.firma);
  IF f IS NULL OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f
     AND p.proowner='vec_personal_propietario'::regrole
     AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')=x.fuente_sha256
     AND p.provolatile='v' AND p.proparallel='u'
     AND p.prosecdef IS NOT DISTINCT FROM (p.proname<>'avanzar_generacion_relacion_rpt_v1')
     AND p.proconfig @> ARRAY['search_path=pg_catalog, pg_temp','row_security=on'])
   OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
     WHERE p.oid=f AND (a.grantee NOT IN(p.proowner,to_regrole(x.permitido))
       OR a.grantor<>p.proowner OR a.privilege_type<>'EXECUTE' OR a.is_grantable)) THEN
   RAISE EXCEPTION 'Personal27: postimagen propia incompatible' USING ERRCODE='55000';
  END IF;
 END LOOP;
 IF NOT EXISTS(SELECT 1 FROM pg_trigger t WHERE t.tgrelid='vec_personal.relacion_servicio_historia'::regclass
   AND t.tgname='relacion_rpt_generacion_insertada' AND NOT t.tgisinternal AND t.tgenabled='O' AND t.tgtype=5
   AND t.tgfoid='vec_personal.avanzar_generacion_relacion_rpt_v1()'::regprocedure) THEN
  RAISE EXCEPTION 'Personal27: trigger propio incompatible' USING ERRCODE='55000';
 END IF;
END $historia$;
REVOKE ALL ON FUNCTION vec_personal.consultar_relacion_para_rpt_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 FROM PUBLIC,vec_personal_ejecutor;
DROP FUNCTION vec_personal.consultar_relacion_para_rpt_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP TRIGGER relacion_rpt_generacion_insertada ON vec_personal.relacion_servicio_historia;
DROP FUNCTION vec_personal.avanzar_generacion_relacion_rpt_v1();
DROP TABLE vec_personal.recibo_relacion_para_rpt;
DROP TABLE vec_personal.control_generacion_relacion_rpt;
COMMIT;
