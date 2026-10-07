\set ON_ERROR_STOP on
-- Mismo fixture privado/GUC que pruebas IS15; sólo clon, rollback final.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
CREATE FUNCTION pg_temp.fallo_segunda_persona_fuentes() RETURNS trigger
LANGUAGE plpgsql AS $f$
BEGIN
 IF NEW.persona_ref=(current_setting('vec.ensayo.plan_fuentes')::jsonb#>>'{personas,1,persona_ref}')
 THEN RAISE EXCEPTION 'ensayo fallo segunda persona' USING ERRCODE='23514'; END IF;
 RETURN NEW;
END $f$;
DO $vectores$
DECLARE p jsonb:=current_setting('vec.ensayo.plan_fuentes')::jsonb;
 m text:=current_setting('vec.ensayo.material_fuentes');
 pre_ca jsonb; pre_is jsonb; sha_ca text; sha_is text; plan_sha text;
 is_r jsonb; ca_r jsonb; replay jsonb; antes_cuentas bigint; antes_perfiles bigint; antes_personas bigint; rechazo boolean;
BEGIN
 plan_sha:=encode(pg_catalog.sha256(convert_to(p::text,'UTF8')),'hex');
 SELECT count(*) INTO antes_cuentas FROM vec_identidad_sesiones_v1.cuenta;
 SELECT count(*) INTO antes_perfiles FROM vec_contexto_actor_v1.perfil_versiones;
 SELECT count(*) INTO antes_personas FROM vec_contexto_actor_v1.persona_versiones;
 pre_ca:=vec_contexto_actor_v1.preimagen_fuentes_iniciales_admin_v1(p);
 pre_is:=vec_identidad_sesiones_v1.preimagen_fuentes_iniciales_admin_v1(p,m);
 sha_ca:=encode(pg_catalog.sha256(convert_to(pre_ca::text,'UTF8')),'hex');
 sha_is:=encode(pg_catalog.sha256(convert_to(pre_is::text,'UTF8')),'hex');
 -- Error real del productor CA en segunda Persona: ambos módulos revierten.
 rechazo:=false;
 BEGIN
  CREATE TRIGGER ensayo_fallo_segunda_persona BEFORE INSERT ON vec_contexto_actor_v1.persona_versiones FOR EACH ROW EXECUTE FUNCTION pg_temp.fallo_segunda_persona_fuentes();
  is_r:=vec_identidad_sesiones_v1.aplicar_fuentes_iniciales_admin_v1(p,m,sha_is,p->>'operacion_ref',plan_sha,'aprobacion_ensayo');
  PERFORM vec_contexto_actor_v1.confirmar_fuentes_iniciales_admin_v1(p,is_r,sha_ca,p->>'operacion_ref',plan_sha,'aprobacion_ensayo');
 EXCEPTION WHEN check_violation THEN rechazo:=true; END;
 IF NOT rechazo OR (SELECT count(*) FROM vec_identidad_sesiones_v1.cuenta)<>antes_cuentas
 OR (SELECT count(*) FROM vec_contexto_actor_v1.persona_versiones)<>antes_personas
 OR EXISTS(SELECT 1 FROM vec_identidad_sesiones_v1.fuentes_iniciales_admin_v1 WHERE operacion_ref=p->>'operacion_ref')
 THEN RAISE EXCEPTION 'CA33: rollback integral no conservado'; END IF;
 is_r:=vec_identidad_sesiones_v1.aplicar_fuentes_iniciales_admin_v1(p,m,sha_is,p->>'operacion_ref',plan_sha,'aprobacion_ensayo');
 rechazo:=false;
 BEGIN
  PERFORM vec_contexto_actor_v1.confirmar_fuentes_iniciales_admin_v1(p,is_r||'{"huella_sha256":"falsa"}',sha_ca,p->>'operacion_ref',plan_sha,'aprobacion_ensayo');
 EXCEPTION WHEN object_not_in_prerequisite_state THEN rechazo:=true; END;
 IF NOT rechazo THEN RAISE EXCEPTION 'CA33: recibo inventado aceptado'; END IF;
 ca_r:=vec_contexto_actor_v1.confirmar_fuentes_iniciales_admin_v1(p,is_r,sha_ca,p->>'operacion_ref',plan_sha,'aprobacion_ensayo');
 replay:=vec_contexto_actor_v1.confirmar_fuentes_iniciales_admin_v1(p,is_r,sha_ca,p->>'operacion_ref',plan_sha,'aprobacion_ensayo');
 IF ca_r IS DISTINCT FROM replay OR (SELECT count(*) FROM vec_contexto_actor_v1.perfil_versiones)<>antes_perfiles
 OR (SELECT count(*) FROM vec_contexto_actor_v1.persona_versiones)<>antes_personas+2
 OR (SELECT count(*) FROM vec_contexto_actor_v1.titularidad_cuenta_persona_v1 WHERE operacion_ref=p->>'operacion_ref')<>4
 THEN RAISE EXCEPTION 'CA33: replay, perfiles o titularidad divergentes'; END IF;
 IF ca_r->>'huella_sha256' IS DISTINCT FROM encode(pg_catalog.sha256(convert_to((ca_r-'huella_sha256')::text,'UTF8')),'hex')
 THEN RAISE EXCEPTION 'CA33: huella recibo divergente'; END IF;
 SET CONSTRAINTS ALL IMMEDIATE;
END $vectores$;
ROLLBACK;
