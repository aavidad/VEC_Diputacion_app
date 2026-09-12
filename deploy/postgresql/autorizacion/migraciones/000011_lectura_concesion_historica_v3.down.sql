\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock_shared(hashtextextended('vec_autorizacion:migracion:registro_contexto_actor_v3:000005',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion:migracion:lectura_concesion_historica_v3:000011',0));
SET LOCAL ROLE vec_autorizacion_propietario;
-- Quita únicamente el lector; la historia, los codecs y sus ACL no se alteran.
DO $retirada$
DECLARE f oid; propietario oid:='vec_autorizacion_propietario'::regrole;
 registro oid:='vec_autorizacion_registro'::regrole;
BEGIN
 IF current_user<>'vec_autorizacion_propietario' OR NOT EXISTS (SELECT 1 FROM pg_namespace
  WHERE nspname='vec_autorizacion' AND nspowner=propietario) THEN
  RAISE EXCEPTION 'auth11: retirada incompatible' USING ERRCODE='55000'; END IF;
 f:=to_regprocedure('vec_autorizacion.leer_concesion_historica_contexto_actor_v3(bytea,bytea,numeric,numeric)');
 IF f IS NULL OR (SELECT count(*) FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
  WHERE n.nspname='vec_autorizacion' AND p.proname='leer_concesion_historica_contexto_actor_v3')<>1 THEN
  RAISE EXCEPTION 'auth11: función ausente o sobrecarga conservada' USING ERRCODE='55000'; END IF;
 IF NOT EXISTS (SELECT 1 FROM pg_proc p JOIN pg_language l ON l.oid=p.prolang
  WHERE p.oid=f AND p.proowner=propietario AND l.lanname='plpgsql' AND p.prokind='f'
   AND p.provolatile='v' AND p.proparallel='u' AND p.prosecdef AND NOT p.proleakproof AND NOT p.proisstrict
   AND p.proretset AND p.pronargs=4 AND p.pronargdefaults=0 AND p.provariadic=0 AND p.prorettype='record'::regtype
   AND p.proargmodes::text[]=ARRAY['i','i','i','i','t','t','t','t']
   AND p.proargnames=ARRAY['p_decision_canonica','p_motivo_canonico','p_persona_version','p_perfil_version','concedida','codigo','decision_huella_sha256','registrada_en']
   AND p.proallargtypes=ARRAY['bytea'::regtype::oid,'bytea'::regtype::oid,'numeric'::regtype::oid,'numeric'::regtype::oid,'boolean'::regtype::oid,'text'::regtype::oid,'text'::regtype::oid,'timestamptz'::regtype::oid]
   AND p.proconfig=ARRAY['search_path=pg_catalog','row_security=on','lock_timeout=2s','statement_timeout=5s']
   AND p.procost=100 AND p.prorows=1 AND p.prosupport=0 AND p.probin IS NULL AND p.prosqlbody IS NULL
   AND p.protrftypes IS NULL
   AND obj_description(p.oid,'pg_proc')='vec_autorizacion:lectura-concesion-historica-v3:000011'
   AND octet_length(p.prosrc)=6021
   AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='67caa4373dc990ddcb50e01c606ea61307958a6676ce43063cf7792ba0d36d26') THEN
  RAISE EXCEPTION 'auth11: definición alterada conservada' USING ERRCODE='55000'; END IF;
 IF EXISTS (
  WITH esperado(grantor,grantee,privilegio,delegable) AS (
   VALUES (propietario,propietario,'EXECUTE'::text,false),(propietario,registro,'EXECUTE'::text,false)
  ), actual AS (
   SELECT a.grantor,a.grantee,a.privilege_type,a.is_grantable
    FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f
  ), diferencia AS (
   (SELECT * FROM esperado EXCEPT ALL SELECT * FROM actual)
   UNION ALL (SELECT * FROM actual EXCEPT ALL SELECT * FROM esperado)
  ) SELECT 1 FROM diferencia
 ) THEN RAISE EXCEPTION 'auth11: ACL alterada conservada' USING ERRCODE='55000'; END IF;
 IF EXISTS (SELECT 1 FROM pg_depend WHERE refclassid='pg_proc'::regclass AND refobjid=f)
 OR EXISTS (SELECT 1 FROM pg_depend WHERE classid='pg_proc'::regclass AND objid=f AND deptype='e')
 OR EXISTS (SELECT 1 FROM pg_proc WHERE oid<>f AND strpos(prosrc,'leer_concesion_historica_contexto_actor_v3')>0) THEN
  RAISE EXCEPTION 'auth11: dependencia conservada' USING ERRCODE='55000'; END IF;
END $retirada$;
DROP FUNCTION vec_autorizacion.leer_concesion_historica_contexto_actor_v3(bytea,bytea,numeric,numeric) RESTRICT;
-- Ningún DROP/DELETE/TRUNCATE de tablas: concesiones originales siguen legibles
-- para el propietario; una instalación posterior puede reponer esta capacidad.
COMMIT;
