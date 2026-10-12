\set ON_ERROR_STOP on
-- PG18 desechable: psql -v propuesta_ref='propuesta_admin:...' -f este_archivo.
-- Requiere una propuesta AUT75 válida aún abierta; simula retirada antes del
-- cierre en una TX reversible y prueba el cotejo que el cierre nuevo invoca.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog,pg_temp;
SELECT set_config('vec.aut75.propuesta_ref',:'propuesta_ref',true);
SET LOCAL ROLE vec_autorizacion_propietario;
DO $prueba$
DECLARE plan jsonb;ref text;revision_base numeric;revision_nueva numeric;
 instante timestamptz(6);fecha text;doc jsonb;huella text;
BEGIN
 SELECT convert_from(p.material,'UTF8')::jsonb->'Plan' INTO STRICT plan
 FROM vec_autorizacion.propuesta_version_rol_bolsa_v2 p
 WHERE p.propuesta_ref=current_setting('vec.aut75.propuesta_ref');
 IF EXISTS(SELECT 1 FROM vec_autorizacion.cierre_version_rol_bolsa_v2 c
  WHERE c.propuesta_ref=current_setting('vec.aut75.propuesta_ref'))
 THEN RAISE EXCEPTION 'AUT75 prueba: propuesta ya cerrada' USING ERRCODE='55000';END IF;
 ref:=plan#>>'{base,control_vigencia,version_rol_ref}';
 revision_base:=(plan#>>'{base,control_vigencia,revision}')::numeric;
 IF NOT EXISTS(SELECT 1 FROM vec_autorizacion.control_vigencia_version_rol_actual q
  WHERE q.version_rol_ref=ref AND q.revision=revision_base)
 THEN RAISE EXCEPTION 'AUT75 prueba: preimagen ya divergente' USING ERRCODE='55000';END IF;
 -- Acredita que el plan todavía es válido antes de la retirada sintética.
 PERFORM vec_autorizacion.validar_plan_version_rol_bolsa_v2(plan,true);
 revision_nueva:=revision_base+1;instante:=clock_timestamp();
 fecha:=vec_autorizacion.fecha_canonica_go_admin_v1(
  to_char(instante AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'));
 doc:=jsonb_build_object('version_rol_ref',ref,'revision',revision_nueva,
  'estado','retirada','actualizado_por','prueba:aut75','actualizado_en',fecha);
 huella:=encode(sha256(convert_to(vec_autorizacion.canon_control_rol_admin_v1(doc),'UTF8')),'hex');
 INSERT INTO vec_autorizacion.control_vigencia_version_rol(
  version_rol_ref,revision,estado,huella_sha256,actualizado_en,documento)
 VALUES(ref,revision_nueva,'retirada',huella,instante,doc);
 UPDATE vec_autorizacion.control_vigencia_version_rol_actual
 SET revision=revision_nueva,actualizada_en=instante,
  actualizada_por='prueba:aut75',acto_ref='prueba:aut75'
 WHERE version_rol_ref=ref AND revision=revision_base;
 IF NOT FOUND THEN RAISE EXCEPTION 'AUT75 prueba: CAS de retirada perdido' USING ERRCODE='40001';END IF;
 BEGIN
  PERFORM vec_autorizacion.validar_plan_version_rol_bolsa_v2(plan,true);
  RAISE EXCEPTION 'AUT75: cierre aceptó control base retirado' USING ERRCODE='55000';
 EXCEPTION WHEN SQLSTATE '40001' THEN NULL;
 END;
END $prueba$;
ROLLBACK;
