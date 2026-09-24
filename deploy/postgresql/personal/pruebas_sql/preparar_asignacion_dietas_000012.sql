\set ON_ERROR_STOP on
-- Sólo para PostgreSQL 18 efímero. Las fachadas AD3-59 no se instalan aquí:
-- esta preimagen mínima no sustituye
-- sus contratos. Las fachadas AD3 nominales TEST-ONLY permiten aislar las
-- guardas, ACL, recibos e historia de 000012; sin simular Personal 000010a.
CREATE SCHEMA vec_autorizacion_atestada_v3;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_personal_propietario;
CREATE TABLE vec_autorizacion_atestada_v3.d7_consumo_prueba (
  nonce text PRIMARY KEY,
  operacion text NOT NULL,
  actor_ref text NOT NULL
);
GRANT INSERT, SELECT ON vec_autorizacion_atestada_v3.d7_consumo_prueba TO vec_personal_propietario;

CREATE FUNCTION vec_autorizacion_atestada_v3.d7_consumir_prueba(p_operacion text, p_capacidad bytea, p_decision bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $$
DECLARE c jsonb; d jsonb; nuevo boolean;
BEGIN
  c:=convert_from(p_capacidad,'UTF8')::jsonb;
  d:=convert_from(p_decision,'UTF8')::jsonb;
  INSERT INTO vec_autorizacion_atestada_v3.d7_consumo_prueba(nonce,operacion,actor_ref)
  VALUES(c->>'nonce',p_operacion,d->>'principal_id') ON CONFLICT DO NOTHING RETURNING true INTO nuevo;
  RETURN QUERY SELECT d->>'decision_ref',d->>'recurso_ref',repeat('a',64),
    encode(sha256(convert_to(c->>'nonce','UTF8')),'hex'),'aud_'||substr(c->>'nonce',1,20),clock_timestamp(),coalesce(nuevo,false);
END $$;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_asignacion_dietas_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql SECURITY DEFINER SET search_path=pg_catalog AS $$ SELECT * FROM vec_autorizacion_atestada_v3.d7_consumir_prueba('consulta',$1,$2) $$;
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_alta_inicial_asignacion_dietas_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql SECURITY DEFINER SET search_path=pg_catalog AS $$ SELECT * FROM vec_autorizacion_atestada_v3.d7_consumir_prueba('alta_inicial',$1,$2) $$;
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_correccion_asignacion_dietas_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql SECURITY DEFINER SET search_path=pg_catalog AS $$ SELECT * FROM vec_autorizacion_atestada_v3.d7_consumir_prueba('correccion',$1,$2) $$;
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_correccion_grupo_dieta_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql SECURITY DEFINER SET search_path=pg_catalog AS $$ SELECT * FROM vec_autorizacion_atestada_v3.d7_consumir_prueba('grupo',$1,$2) $$;
ALTER FUNCTION vec_autorizacion_atestada_v3.d7_consumir_prueba(text,bytea,bytea) OWNER TO vec_personal_propietario;
ALTER FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_alta_inicial_asignacion_dietas_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) OWNER TO vec_personal_propietario;
ALTER FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_asignacion_dietas_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) OWNER TO vec_personal_propietario;
ALTER FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_correccion_asignacion_dietas_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) OWNER TO vec_personal_propietario;
ALTER FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_correccion_grupo_dieta_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) OWNER TO vec_personal_propietario;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA vec_autorizacion_atestada_v3 TO vec_personal_propietario;

-- 000009 aporta únicamente la relación. La alta inicial la ejerce 000012.
BEGIN;
SET LOCAL ROLE vec_personal_propietario;
SELECT set_config('vec.dietas.persona_ref','per_abcdefghijklmnopqrstuv',true);
INSERT INTO vec_personal.relacion_empleado_dietas(relacion_ref,persona_ref,empleado_ref,unidad_ref,estado,desde,hasta,version,procedencia_acto_ref,fuente_ref,fuente_version)
VALUES('rel_abcdefghijklmnopqrstuv','per_abcdefghijklmnopqrstuv','emp_abcdefghijklmnopqrstuv','unidad-sintetica','activa',DATE '2026-01-01',NULL,1,'acto-sintetico','fuente-sintetica',1);
COMMIT;

CREATE SCHEMA vec_prueba_d7;
CREATE ROLE vec_prueba_d7_dietas NOLOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_prueba_d7_personal NOLOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
GRANT vec_dietas_ejecutor TO vec_prueba_d7_dietas WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
GRANT USAGE ON SCHEMA vec_prueba_d7 TO vec_dietas_ejecutor,vec_dietas_propietario,vec_personal_ejecutor;

CREATE FUNCTION vec_prueba_d7.material(p_operacion text,p_clave text DEFAULT '',p_centro text DEFAULT 'centro-corregido',p_grupo smallint DEFAULT 1,p_version bigint DEFAULT NULL)
RETURNS text LANGUAGE plpgsql STABLE AS $$
DECLARE ar text:=CASE WHEN p_operacion='consultar' THEN 'actor-sujeto' ELSE 'actor-administrativo' END; ap text:=CASE WHEN p_operacion='consultar' THEN 'per_abcdefghijklmnopqrstuv' ELSE 'per_bbbbbbbbbbbbbbbbbbbbbb' END; ae text:=CASE WHEN p_operacion='consultar' THEN 'emp_abcdefghijklmnopqrstuv' ELSE 'emp_bbbbbbbbbbbbbbbbbbbbbb' END;
BEGIN
 RETURN jsonb_build_object('administrativo_persona_ref',CASE WHEN p_operacion='consultar' THEN '' ELSE 'per_bbbbbbbbbbbbbbbbbbbbbb' END,'centro_ref',CASE WHEN p_operacion='consultar' THEN '' ELSE p_centro END,'clave_idempotencia',coalesce(p_clave,''),'empleado_ref','emp_abcdefghijklmnopqrstuv','esquema','vec.personal.asignacion-dietas.v1','fecha_referencia',current_date::text,'grupo_dieta',CASE WHEN p_operacion='consultar' THEN 0 ELSE p_grupo END,'identidad',jsonb_build_object('actor_ref',ar,'contexto_actor_ref','ctx-a','contexto_version','1','cuenta_ref','cuenta-a','cuenta_version','1','empleado_ref',ae,'perfil_ref','perfil-a','perfil_version','1','persona_ref',ap,'persona_version','1'),'motivo_revision',CASE WHEN p_operacion='consultar' THEN '' ELSE 'corrección sintética' END,'operacion',p_operacion,'persona_ref','per_abcdefghijklmnopqrstuv','procedencia_acto_ref',CASE WHEN p_operacion='consultar' THEN '' ELSE 'acto-sintetico-corregido' END,'relacion_ref','rel_abcdefghijklmnopqrstuv','responsable_persona_ref',CASE WHEN p_operacion='consultar' THEN '' ELSE 'per_cccccccccccccccccccccc' END,'unidad_ref','unidad-sintetica','version_esperada',CASE WHEN p_operacion IN ('consultar','registrar_inicial') THEN 0 ELSE coalesce(p_version,1) END,'vigente_desde',CASE WHEN p_operacion='consultar' THEN '' ELSE current_date::text END)::text;
END $$;
CREATE FUNCTION vec_prueba_d7.contexto(p_operacion text) RETURNS bytea LANGUAGE plpgsql IMMUTABLE AS $$
DECLARE ar text:=CASE WHEN p_operacion='consultar' THEN 'actor-sujeto' ELSE 'actor-administrativo' END; ap text:=CASE WHEN p_operacion='consultar' THEN 'per_abcdefghijklmnopqrstuv' ELSE 'per_bbbbbbbbbbbbbbbbbbbbbb' END; ae text:=CASE WHEN p_operacion='consultar' THEN 'emp_abcdefghijklmnopqrstuv' ELSE 'emp_bbbbbbbbbbbbbbbbbbbbbb' END;
BEGIN RETURN convert_to(jsonb_build_object('esquema','vec.contexto-actor.vinculado.v2','principal_ref',ar,'contexto_actor_ref','ctx-a','contexto_version','1','cuenta_ref','cuenta-a','cuenta_version','1','perfil_activo_ref','perfil-a','persona_ref',ap,'persona_version','1','perfil_version','1','vinculos',jsonb_build_array(jsonb_build_object('tipo','empleado','estado','activo','referencia',ae)))::text,'UTF8'); END $$;
CREATE FUNCTION vec_prueba_d7.capacidad(p_operacion text,p_nonce text) RETURNS bytea LANGUAGE sql IMMUTABLE AS $$
 SELECT convert_to(jsonb_build_object('nonce',p_nonce,'operacion',CASE p_operacion WHEN 'registrar_inicial' THEN 'personal.asignacion_dietas.registrar_inicial' WHEN 'consultar' THEN 'personal.asignacion_dietas.consultar' WHEN 'corregir' THEN 'personal.asignacion_dietas.corregir' ELSE 'personal.asignacion_dietas.grupo_corregir' END,'audiencia_consumo',CASE p_operacion WHEN 'registrar_inicial' THEN 'vec_personal.asignacion_dietas.registrar_inicial.v1' WHEN 'consultar' THEN 'vec_personal.asignacion_dietas.consultar.v1' WHEN 'corregir' THEN 'vec_personal.asignacion_dietas.corregir.v1' ELSE 'vec_personal.asignacion_dietas.grupo_corregir.v1' END)::text,'UTF8') $$;
CREATE FUNCTION vec_prueba_d7.decision(p_operacion text,p_nonce text,p_clave text DEFAULT '',p_centro text DEFAULT 'centro-corregido',p_grupo smallint DEFAULT 1,p_version bigint DEFAULT NULL)
RETURNS bytea LANGUAGE plpgsql STABLE AS $$
DECLARE m text:=vec_prueba_d7.material(p_operacion,p_clave,p_centro,p_grupo,p_version); h text; recurso text; accion text; finalidad text;
BEGIN
 h:=encode(sha256(convert_to(m,'UTF8')),'hex');
 recurso:='{"ambitos":{"empleado_ref":"emp_abcdefghijklmnopqrstuv","persona_ref":"per_abcdefghijklmnopqrstuv","relacion_ref":"rel_abcdefghijklmnopqrstuv","unidad_ref":"unidad-sintetica"},"atributos":{"fecha_referencia":"'||current_date::text||'","material_sha256":"'||h||'","operacion":"'||p_operacion||'"}}';
 accion:=CASE p_operacion WHEN 'registrar_inicial' THEN 'personal.asignacion_dietas.registrar_inicial' WHEN 'consultar' THEN 'personal.asignacion_dietas.consultar' WHEN 'corregir' THEN 'personal.asignacion_dietas.corregir' ELSE 'personal.asignacion_dietas.grupo_corregir' END;
 finalidad:=CASE p_operacion WHEN 'registrar_inicial' THEN 'registrar_asignacion_dietas_inicial' WHEN 'consultar' THEN 'preparar_borrador_dietas' WHEN 'corregir' THEN 'corregir_asignacion_dietas' ELSE 'corregir_grupo_dieta' END;
 RETURN convert_to(jsonb_build_object('concedida','true','accion',accion,'modulo_id','personal','tipo_recurso','asignacion_dietas','recurso_ref','rel_abcdefghijklmnopqrstuv','finalidad',finalidad,'obligaciones','[]'::jsonb,'principal_id',CASE WHEN p_operacion='consultar' THEN 'actor-sujeto' ELSE 'actor-administrativo' END,'perfil_activo_ref','perfil-a','decision_ref','dec_'||p_nonce,'contexto_recurso_huella_sha256',encode(sha256(convert_to(recurso,'UTF8')),'hex'),'campos_permitidos','["administrativo_persona_ref","asignacion_ref","auditoria_ad3_ref","centro_ref","consumo_huella_sha256","decision_ref","efecto_ref","estado_local","grupo_dieta","persona_ref","recibo_ref","registrada_en","relacion_ref","responsable_persona_ref","unidad_ref","version","vigente_desde"]'::jsonb)::text,'UTF8');
END $$;
CREATE FUNCTION vec_prueba_d7.material_grupo_propio(p_clave text)
RETURNS text LANGUAGE sql STABLE AS $$
 SELECT jsonb_set(
   jsonb_set(
    jsonb_set(vec_prueba_d7.material('grupo_corregir',p_clave,'centro-corregido',2::smallint,2)::jsonb,'{identidad,actor_ref}','"actor-sujeto"'),
    '{identidad,persona_ref}','"per_abcdefghijklmnopqrstuv"'),
   '{identidad,empleado_ref}','"emp_abcdefghijklmnopqrstuv"')::text
$$;
CREATE FUNCTION vec_prueba_d7.decision_grupo_propio(p_nonce text,p_clave text)
RETURNS bytea LANGUAGE plpgsql STABLE AS $$
DECLARE m text:=vec_prueba_d7.material_grupo_propio(p_clave); h text; recurso text;
BEGIN
 h:=encode(sha256(convert_to(m,'UTF8')),'hex');
 recurso:='{"ambitos":{"empleado_ref":"emp_abcdefghijklmnopqrstuv","persona_ref":"per_abcdefghijklmnopqrstuv","relacion_ref":"rel_abcdefghijklmnopqrstuv","unidad_ref":"unidad-sintetica"},"atributos":{"fecha_referencia":"'||current_date::text||'","material_sha256":"'||h||'","operacion":"grupo_corregir"}}';
 RETURN convert_to(jsonb_build_object('concedida','true','accion','personal.asignacion_dietas.grupo_corregir','modulo_id','personal','tipo_recurso','asignacion_dietas','recurso_ref','rel_abcdefghijklmnopqrstuv','finalidad','corregir_grupo_dieta','obligaciones','[]'::jsonb,'principal_id','actor-sujeto','perfil_activo_ref','perfil-a','decision_ref','dec_'||p_nonce,'contexto_recurso_huella_sha256',encode(sha256(convert_to(recurso,'UTF8')),'hex'),'campos_permitidos','["administrativo_persona_ref","asignacion_ref","auditoria_ad3_ref","centro_ref","consumo_huella_sha256","decision_ref","efecto_ref","estado_local","grupo_dieta","persona_ref","recibo_ref","registrada_en","relacion_ref","responsable_persona_ref","unidad_ref","version","vigente_desde"]'::jsonb)::text,'UTF8');
END $$;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA vec_prueba_d7 TO vec_dietas_ejecutor,vec_dietas_propietario,vec_personal_ejecutor;
CREATE FUNCTION vec_prueba_d7.asignacion_v3_prueba() RETURNS text
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $$
BEGIN
 PERFORM set_config('vec.dietas.persona_ref','per_abcdefghijklmnopqrstuv',true);
 RETURN (SELECT asignacion_ref FROM vec_personal.asignacion_dietas WHERE version=3);
END $$;
ALTER FUNCTION vec_prueba_d7.asignacion_v3_prueba() OWNER TO vec_personal_propietario;
REVOKE ALL ON FUNCTION vec_prueba_d7.asignacion_v3_prueba() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_prueba_d7.asignacion_v3_prueba() TO vec_dietas_propietario;
