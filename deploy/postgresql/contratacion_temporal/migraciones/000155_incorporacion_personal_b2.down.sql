\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='2min';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000155',0));
LOCK TABLE vec_contratacion_temporal.plan_incorporacion_personal_b2_v1,vec_contratacion_temporal.origen_incorporacion_personal_b2_v1 IN ACCESS EXCLUSIVE MODE;
DO $pre$
BEGIN
 IF EXISTS(SELECT 1 FROM vec_contratacion_temporal.plan_incorporacion_personal_b2_v1) OR EXISTS(SELECT 1 FROM vec_contratacion_temporal.origen_incorporacion_personal_b2_v1) THEN RAISE EXCEPTION 'CT155: DOWN prohibido con historia' USING ERRCODE='55000';END IF;
END $pre$;

DO $consumidores$
DECLARE x record;p record;def text;meta jsonb;paso jsonb;
BEGIN
 FOR x IN SELECT firma,jsonb_agg(jsonb_build_object('anterior',anterior,'nuevo',nuevo)) AS pasos FROM (VALUES
('leer_antecedente_reincorporacion_titular_atestada_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',$old$   AND i.protocolo=c.incorporacion_protocolo
   AND i.relacion_ref=p_material->>'relacion_ref'$old$,$new$   AND r.organizacion_ref=c.organizacion_ref AND r.expediente_ref=c.expediente_ref
   AND i.material_json#>>'{Confirmacion,ResultadoPersonal,relacion_ref}'=r.relacion_ref
   AND r.relacion_ref=p_material->>'relacion_ref'$new$),
('leer_antecedente_reincorporacion_titular_atestada_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',$old$ CROSS JOIN LATERAL vec_contratacion_temporal.origen_incorporacion_neutral_ct155(c.organizacion_ref,c.expediente_ref,coalesce(c.incorporacion_ref,c.incorporacion_b2_ref)) i$old$,$new$ JOIN vec_contratacion_temporal.incorporacion_registro_v2 i ON i.recibo_ref=c.incorporacion_ref
 JOIN vec_contratacion_temporal.seguimiento_raiz_v2 r ON r.seguimiento_ref=i.seguimiento_ref$new$),
('confirmar_confirmacion_ginpix_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',$old$WHERE coalesce(incorporacion_ref,incorporacion_b2_ref)=v_inc.recibo_ref$old$,$new$WHERE incorporacion_ref=v_inc.recibo_ref$new$),
('origen_reincorporacion_ct130(jsonb)',$old$ SELECT * INTO i FROM vec_contratacion_temporal.origen_incorporacion_neutral_ct155(c.organizacion_ref,c.expediente_ref,coalesce(c.incorporacion_ref,c.incorporacion_b2_ref));
 IF NOT FOUND OR i.protocolo IS DISTINCT FROM c.incorporacion_protocolo OR i.relacion_ref IS DISTINCT FROM m->>'relacion_ref' OR i.relacion_ref IS NULL THEN
  estado:='cese_no_coincide';RETURN NEXT;RETURN;END IF;
 relacion_ref:=i.relacion_ref;incorporacion_recibo_ref:=i.recibo_ref;$old$,$new$ SELECT * INTO i FROM vec_contratacion_temporal.incorporacion_registro_v2 WHERE recibo_ref=c.incorporacion_ref;
 IF NOT FOUND THEN estado:='cese_no_coincide'; RETURN NEXT; RETURN; END IF;
 SELECT * INTO r FROM vec_contratacion_temporal.seguimiento_raiz_v2 WHERE seguimiento_ref=i.seguimiento_ref;
 IF NOT FOUND OR i.organizacion_ref IS DISTINCT FROM c.organizacion_ref OR i.expediente_ref IS DISTINCT FROM c.expediente_ref
    OR r.organizacion_ref IS DISTINCT FROM c.organizacion_ref OR r.expediente_ref IS DISTINCT FROM c.expediente_ref
    OR i.material_json#>>'{Confirmacion,ResultadoPersonal,relacion_ref}' IS DISTINCT FROM r.relacion_ref
    OR m->>'relacion_ref' IS DISTINCT FROM r.relacion_ref
    OR r.relacion_ref IS NULL THEN
  estado:='cese_no_coincide'; RETURN NEXT; RETURN; END IF;
 relacion_ref:=r.relacion_ref; incorporacion_recibo_ref:=i.recibo_ref;$new$),
('origen_reincorporacion_ct130(jsonb)',$old$DECLARE c vec_contratacion_temporal.cese_nombramiento_v1%ROWTYPE; i record;$old$,$new$DECLARE c vec_contratacion_temporal.cese_nombramiento_v1%ROWTYPE; i vec_contratacion_temporal.incorporacion_registro_v2%ROWTYPE;
 r vec_contratacion_temporal.seguimiento_raiz_v2%ROWTYPE;$new$),
('ginpix_confirmado_ct124(text,text)',$old$coalesce(g.incorporacion_ref,g.incorporacion_b2_ref)=(SELECT$old$,$new$g.incorporacion_ref=(SELECT$new$),
('resultado_ginpix_ct124(vec_contratacion_temporal.confirmacion_ginpix_v1)',$old$coalesce(r.incorporacion_ref,r.incorporacion_b2_ref),'inicio'$old$,$new$r.incorporacion_ref,'inicio'$new$),
('resultado_cese_ct115(vec_contratacion_temporal.cese_nombramiento_v1)',$old$coalesce(r.incorporacion_ref,r.incorporacion_b2_ref),'inicio'$old$,$new$r.incorporacion_ref,'inicio'$new$),
('incorporacion_expediente_ct115(text,text)',$old$    SELECT n.recibo_ref,n.inicio,n.llamamiento_ref FROM vec_contratacion_temporal.origen_incorporacion_neutral_ct155(p_organizacion,p_expediente,NULL) n$old$,$new$    SELECT r.recibo_ref, vec_contratacion_temporal.inicio_incorporacion_ct115(r.material_json),
           (SELECT pf.llamamiento_ref FROM vec_contratacion_temporal.propuesta_formalizacion pf
             WHERE pf.organizacion_ref=p_organizacion AND pf.expediente_ref=p_expediente
             ORDER BY pf.confirmada_en DESC, pf.propuesta_ref COLLATE "C" DESC LIMIT 1)
      FROM vec_contratacion_temporal.incorporacion_registro_v2 r
     WHERE r.organizacion_ref=p_organizacion AND r.expediente_ref=p_expediente
     ORDER BY r.registrada_en DESC, r.recibo_ref COLLATE "C" DESC LIMIT 1$new$)
 ) v(firma,anterior,nuevo) GROUP BY firma LOOP
 SELECT pg_get_functiondef(oid) AS def,to_jsonb(q)-'prosrc' AS meta INTO STRICT p FROM pg_proc q WHERE oid=to_regprocedure('vec_contratacion_temporal.'||x.firma) AND proowner=current_user::regrole;
 def:=p.def;
 FOR paso IN SELECT value FROM jsonb_array_elements(x.pasos) LOOP
 IF length(def)-length(replace(def,paso->>'anterior',''))<>length(paso->>'anterior') THEN RAISE EXCEPTION 'CT155: preimagen incompatible: %',x.firma USING ERRCODE='55000';END IF;
 def:=replace(def,paso->>'anterior',paso->>'nuevo');
 END LOOP;
 EXECUTE def;
 IF (SELECT pg_get_functiondef(oid) FROM pg_proc WHERE oid=to_regprocedure('vec_contratacion_temporal.'||x.firma)) IS DISTINCT FROM def OR (SELECT to_jsonb(q)-'prosrc' FROM pg_proc q WHERE oid=to_regprocedure('vec_contratacion_temporal.'||x.firma)) IS DISTINCT FROM p.meta THEN RAISE EXCEPTION 'CT155: metadatos alterados' USING ERRCODE='55000';END IF;
 END LOOP;
END $consumidores$;

DO $fk$
DECLARE t text;col text;
BEGIN
 FOREACH t IN ARRAY ARRAY['cese_nombramiento_v1','confirmacion_ginpix_v1','reincorporacion_titular_v1'] LOOP
 col:=CASE WHEN t='reincorporacion_titular_v1' THEN 'incorporacion_recibo_ref' ELSE 'incorporacion_ref' END;
 EXECUTE format('DROP TRIGGER enlazar_origen_personal_ct155 ON vec_contratacion_temporal.%I',t);
 EXECUTE format('ALTER TABLE vec_contratacion_temporal.%I DROP CONSTRAINT origen_xor_ct155,DROP CONSTRAINT origen_b2_ct155_fk,DROP COLUMN incorporacion_b2_ref,DROP COLUMN incorporacion_protocolo,ALTER COLUMN %I SET NOT NULL',t,col);
 END LOOP;
END $fk$;
DROP FUNCTION vec_contratacion_temporal.registrar_plan_nominal_b2_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_contratacion_temporal.leer_plan_nominal_b2_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_contratacion_temporal.confirmar_origen_incorporacion_b2_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_contratacion_temporal.leer_origen_incorporacion_b2_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_contratacion_temporal.leer_antecedentes_plan_b2_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_contratacion_temporal.operacion_plan_personal_ct155(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_contratacion_temporal.enlazar_origen_personal_ct155();
DROP FUNCTION vec_contratacion_temporal.origen_incorporacion_neutral_ct155(text,text,text);
DROP FUNCTION vec_contratacion_temporal.validar_plan_personal_ct155(jsonb,jsonb);
DROP FUNCTION vec_contratacion_temporal.antecedentes_plan_personal_ct155(text,text);
DROP FUNCTION vec_contratacion_temporal.evento_plan_personal_ct155(text,text,numeric,text,text,text,text,timestamptz);
DROP FUNCTION vec_contratacion_temporal.sesion_plan_personal_ct155();
DROP TABLE vec_contratacion_temporal.origen_incorporacion_personal_b2_v1;
DROP TABLE vec_contratacion_temporal.plan_incorporacion_personal_b2_v1;
DROP FUNCTION vec_contratacion_temporal.canon_plan_personal_ct155(jsonb);
COMMIT;
