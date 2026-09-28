\set ON_ERROR_STOP on
SET ROLE vec_bolsa_llamamientos_propietario;
INSERT INTO vec_bolsa_llamamientos.situacion_participacion VALUES
 ('participacion:uno','2026-09-28 09:00Z','disponible',NULL,'Constitución de bolsa','sistema:constitucion','recibo:situacion:constitucion:participacion:uno','2026-09-28 09:00Z'),
 ('participacion:uno','2026-09-28 10:00Z','no_disponible',NULL,'Contiene correo privado que no debe salir','actor:rrhh','recibo:s1','2026-09-28 10:00Z'),
 ('participacion:uno','2026-09-28 12:00Z','disponible',NULL,'Constitución de bolsa','actor:rrhh','recibo:s2','2026-09-28 12:00Z'),
 ('participacion:ajena','2026-09-28 13:00Z','disponible',NULL,'Motivo ajeno privado','actor:otro','recibo:ajeno','2026-09-28 13:00Z');
INSERT INTO vec_bolsa_llamamientos.operacion_situacion_participacion VALUES
 ('participacion:uno','2026-09-28 10:00Z','pausar'),
 ('participacion:uno','2026-09-28 12:00Z','reactivar');
INSERT INTO vec_bolsa_llamamientos.datos_contacto_participacion VALUES
 ('participacion:uno',1,'recibo:c1','correo personal privado','actor:rrhh','2026-09-28 11:00Z'),
 ('participacion:uno',2,'recibo:c2','teléfono personal privado','actor:rrhh','2026-09-28 14:00Z');
INSERT INTO vec_bolsa_llamamientos.traza_valor_participacion VALUES
 ('participacion:uno','recibo:s1','situacion','disponible','no_disponible','actor:rrhh','2026-09-28 10:00Z'),
 ('participacion:uno','recibo:s2','situacion','no_disponible','disponible','actor:rrhh','2026-09-28 12:00Z'),
 ('participacion:uno','recibo:c1','datos_contacto',NULL,'version:1','actor:rrhh','2026-09-28 11:00Z'),
 ('participacion:uno','recibo:c1','correo',NULL,'version:1','actor:rrhh','2026-09-28 11:00Z'),
 ('participacion:uno','recibo:c2','datos_contacto','version:1','version:2','actor:rrhh','2026-09-28 14:00Z'),
 ('participacion:uno','recibo:c2','telefono_1','version:1','version:2','actor:rrhh','2026-09-28 14:00Z'),
 ('participacion:ajena','recibo:ajeno','situacion',NULL,'disponible','actor:otro','2026-09-28 13:00Z'),
 ('participacion:uno','recibo:huerfano','correo',NULL,'version:9','actor:otro','2026-09-28 15:00Z');
RESET ROLE;

-- Construye únicamente el material mínimo del doble. La función productiva
-- comprueba íntegramente actor, acción, campos, huella, cursor y sesión.
CREATE FUNCTION public.b56_consultar(
 p_participacion text,p_limite integer,p_antes_instante timestamptz DEFAULT NULL,
 p_antes_id text DEFAULT NULL,p_accion text DEFAULT 'vec.auditoria.consultar')
RETURNS TABLE(id text,ocurrido_en timestamptz,accion text,actor_ref text,
 resultado text,expediente_ref text,recibo_ref text,motivo text,
 campo text,valor_anterior text,valor_nuevo text)
LANGUAGE plpgsql VOLATILE SECURITY INVOKER SET search_path=pg_catalog SET timezone='UTC' AS $f$
DECLARE v_desde timestamptz:='2026-09-28 00:00Z'; v_hasta timestamptz:='2026-09-29 00:00Z';
 v_finalidad text:='revision_rrhh'; v_motivo text:='motivo:ensayo';
 v_filtro text; v_recurso text; v_capacidad bytea; v_decision bytea;
BEGIN
 v_filtro:=encode(sha256(convert_to(array_to_string(ARRAY[
  'vec.auditoria.filtro.v1','bolsa',p_participacion,'',
  to_char(v_desde AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  to_char(v_hasta AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  p_limite::text,
  coalesce(to_char(p_antes_instante AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),''),
  CASE WHEN p_antes_id IS NULL THEN '' ELSE 'bolsa' END,
  coalesce(p_antes_id,''),v_finalidad,v_motivo
 ],E'\n'),'UTF8')),'hex');
 v_recurso:=encode(sha256(convert_to(
  '{"ambitos":{"expediente_ref":'||to_json(p_participacion)::text||
  ',"fuente":"bolsa"},"atributos":{"filtro_sha256":"'||v_filtro||'"}}','UTF8')),'hex');
 v_capacidad:=convert_to(jsonb_build_object(
  'operacion','vec.auditoria.consultar','audiencia_consumo','vec_auditoria.consulta_rrhh.v1',
  'efecto_ref',p_participacion,'huella_efecto_sha256',v_recurso)::text,'UTF8');
 v_decision:=convert_to(jsonb_build_object(
  'principal_id','actor:auditor','accion',p_accion,'modulo_id','auditoria',
  'tipo_recurso','historial_auditoria','recurso_ref',p_participacion,
  'finalidad',v_finalidad,'contexto_recurso_huella_sha256',v_recurso,
  'campos_permitidos',jsonb_build_array('accion','actor_ref','antes','antes_sha256',
   'datos_disponibles','despues','despues_sha256','expediente_ref','fuente','id',
   'modulo_id','motivo','ocurrido_en','recibo_ref','resultado'),
  'obligaciones','[]'::jsonb)::text,'UTF8');
 RETURN QUERY SELECT * FROM vec_bolsa_llamamientos.consultar_auditoria_participacion_v1(
  p_participacion,NULL,v_desde,v_hasta,p_antes_instante,
  CASE WHEN p_antes_id IS NULL THEN NULL ELSE 'bolsa' END,p_antes_id,p_limite,
  'actor:auditor',v_finalidad,v_motivo,v_filtro,v_capacidad,v_decision,
  '\x00'::bytea,'\x00'::bytea,1,1,'\x00'::bytea,'\x00'::bytea,'\x00'::bytea,'\x00'::bytea);
END $f$;
REVOKE ALL ON FUNCTION public.b56_consultar(text,integer,timestamptz,text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION public.b56_consultar(text,integer,timestamptz,text,text) TO vec_b56_rrhh;
