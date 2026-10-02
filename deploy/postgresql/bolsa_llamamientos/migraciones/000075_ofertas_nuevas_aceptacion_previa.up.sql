\set ON_ERROR_STOP on
-- RRHH 02/10/2026: una oferta telemática nueva sólo usa aceptación previa.
-- B71 y B70 preceden a B75. La guarda actúa sólo al insertar: una operación
-- histórica recuperada por su clave conserva su política, recibo e historia.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000075',0));

DO $pre$
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR to_regclass('vec_bolsa_llamamientos.oferta_publicada') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.politica_ofertas_version') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.publicar_oferta_v4(text,text,text,text,text,jsonb,jsonb,timestamptz,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text,integer)') IS NULL
    OR position('confirmacion_adjudicacion' IN pg_get_functiondef(to_regprocedure('vec_bolsa_llamamientos.proyectar_oferta_v2(text,timestamp with time zone)'))) = 0
 THEN
  RAISE EXCEPTION 'B75: clave=dependencias_B71_B70 esperado=presentes actual=ausentes' USING ERRCODE='55000';
 END IF;
 IF to_regprocedure('vec_bolsa_llamamientos.verificar_aceptacion_previa_oferta_b75()') IS NOT NULL
    OR EXISTS (SELECT 1 FROM pg_trigger
                WHERE tgrelid='vec_bolsa_llamamientos.oferta_publicada'::regclass
                  AND tgname='verificar_aceptacion_previa_oferta_b75')
 THEN
  RAISE EXCEPTION 'B75: clave=instalacion esperado=ausente actual=presente' USING ERRCODE='55000';
 END IF;
END $pre$;

CREATE FUNCTION vec_bolsa_llamamientos.verificar_aceptacion_previa_oferta_b75()
RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE v_confirmacion text; v_tiene_plazas boolean;
BEGIN
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:politica-ofertas:'||NEW.bolsa_ref,0));
 SELECT jsonb_typeof(pv.politica->'plazas')='object', pv.politica#>>'{adjudicacion,confirmacion}'
 INTO v_tiene_plazas,v_confirmacion
 FROM vec_bolsa_llamamientos.politica_ofertas_version pv
 WHERE pv.bolsa_ref=NEW.bolsa_ref
   AND pv.version::text=NEW.plazo->>'politica_version';
 -- Sin apartado de plazas la adjudicación de la única plaza ya era definitiva.
 IF v_tiene_plazas IS TRUE AND v_confirmacion IS DISTINCT FROM 'aceptacion_previa' THEN
  RAISE EXCEPTION 'B75: clave=confirmacion_oferta_nueva esperado=aceptacion_previa actual=ausente_o_distinta' USING ERRCODE='22023';
 END IF;
 RETURN NEW;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.verificar_aceptacion_previa_oferta_b75() FROM PUBLIC;

CREATE TRIGGER verificar_aceptacion_previa_oferta_b75
BEFORE INSERT ON vec_bolsa_llamamientos.oferta_publicada
FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.verificar_aceptacion_previa_oferta_b75();

DO $post$
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_trigger
                 WHERE tgrelid='vec_bolsa_llamamientos.oferta_publicada'::regclass
                   AND tgname='verificar_aceptacion_previa_oferta_b75'
                   AND NOT tgisinternal AND tgenabled='O')
    OR EXISTS (SELECT 1 FROM pg_proc p
               CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
               WHERE p.oid=to_regprocedure('vec_bolsa_llamamientos.verificar_aceptacion_previa_oferta_b75()')
                 AND a.grantee<>p.proowner)
 THEN
  RAISE EXCEPTION 'B75: clave=guarda_y_acl esperado=activa_privada actual=ausente_o_publica' USING ERRCODE='55000';
 END IF;
END $post$;
COMMIT;
