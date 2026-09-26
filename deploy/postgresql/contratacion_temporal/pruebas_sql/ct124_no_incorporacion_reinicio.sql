\set ON_ERROR_STOP on
-- Tras reiniciar PostgreSQL: la no incorporación, la baja en Bolsa, el
-- siguiente llamamiento y la continuación se recuperan idénticos con los
-- mismos materiales, sin filas nuevas. Requiere ct124_utilidades.sql y las
-- funciones de ct124_no_incorporacion_cadena.sql (se vuelven a crear aquí).
CREATE FUNCTION pg_temp.continuar(p_material text) RETURNS jsonb LANGUAGE plpgsql AS $$
DECLARE r jsonb; d bytea:=convert_to(jsonb_build_object('accion','contratacion_temporal.llamamiento.siguiente.continuar',
   'modulo_id','contratacion_temporal','tipo_recurso','continuacion_llamamiento_ct','finalidad','gestionar_contratacion_temporal',
   'recurso_ref',(p_material::jsonb)->'Solicitud'->>'ExpedienteRef',
   'contexto_recurso_huella_sha256',encode(sha256(convert_to('{"ambitos":{"organizacion_ref":"'||((p_material::jsonb)->'Solicitud'->>'OrganizacionRef')||
     '"},"atributos":{"material_sha256":"'||encode(sha256(convert_to(p_material,'UTF8')),'hex')||'"}}','UTF8')),'hex'),
   'principal_id','per_ct124_actor','perfil_activo_ref','prf_ct124')::text,'UTF8');
BEGIN
 r:=vec_contratacion_temporal.continuar_llamamiento_rrhh_v2(p_material,'\x01'::bytea,d,'\x01'::bytea,'\x01'::bytea,1,1,
   '\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea);
 RETURN r;
EXCEPTION WHEN others THEN RETURN jsonb_build_object('error',SQLSTATE,'mensaje',SQLERRM);
END $$;
CREATE FUNCTION pg_temp.guardar_bolsa(p_registro bytea, p_operacion text) RETURNS jsonb LANGUAGE plpgsql AS $$
DECLARE r jsonb:=convert_from(p_registro,'UTF8')::jsonb; c bytea; g record;
BEGIN
 c:=convert_to(jsonb_build_object('efecto_ref',r->>'operacion_ref',
   'huella_efecto_sha256',encode(sha256(convert_to('{"ambitos":{"categoria_ref":'||to_json(r->>'categoria_ref')::text||
     ',"unidad_ref":'||to_json(r->>'unidad_ref')::text||'},"atributos":{"contenido_sha256":'||
     to_json(encode(sha256(p_registro),'hex'))::text||',"necesidad_ref":'||to_json(r->>'necesidad_ref')::text||'}}','UTF8')),'hex'),
   'operacion',p_operacion,'audiencia_consumo','vec_bolsa_llamamientos.confirmar_integracion_desarrollo.v1','relleno',repeat('x',600))::text,'UTF8');
 SELECT * INTO STRICT g FROM vec_bolsa_llamamientos.guardar_integracion_desarrollo_v1(p_registro,c,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,1,1,
   '\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea);
 RETURN jsonb_build_object('recibo_ref',g.recibo_ref,'evento_ref',g.evento_ref,'confirmada_en',g.confirmada_en);
EXCEPTION WHEN others THEN RETURN jsonb_build_object('error',SQLSTATE,'mensaje',SQLERRM);
END $$;
CREATE FUNCTION pg_temp.plazos() RETURNS jsonb LANGUAGE sql AS $$
 SELECT jsonb_build_object('recurso_vence','2026-10-20','recurso_regla_ref','vec.bolsa.reglas:3:b24.consecuencias',
   'recurso_regla_huella_sha256',repeat('d',64),'suspension_hasta',NULL) $$;
DO $$ BEGIN EXECUTE format('GRANT USAGE ON SCHEMA %I TO vec_ct115_runtime, vec_b42_runtime, vec_b42_relevo',pg_my_temp_schema()::regnamespace); END $$;

SELECT ni::text AS ni, r_ni::text AS r_ni, pub::text AS pub, b1::text AS b1, sig_bolsa, g_sig::text AS g_sig, mat_conf, cont::text AS cont
  FROM public.prueba_ct124_ni \gset
SELECT (SELECT count(*) FROM vec_contratacion_temporal.no_incorporacion_v1)||'/'||(SELECT count(*) FROM vec_contratacion_temporal.outbox_expediente_integral)
  ||'/'||(SELECT count(*) FROM vec_bolsa_llamamientos.no_incorporacion_bolsa)||'/'||(SELECT count(*) FROM vec_bolsa_llamamientos.no_incorporacion_bolsa_evaluacion)||'/'||(SELECT count(*) FROM vec_bolsa_llamamientos.sancion_participacion)
  ||'/'||(SELECT count(*) FROM vec_bolsa_llamamientos.situacion_participacion)||'/'||(SELECT count(*) FROM vec_bolsa_llamamientos.integracion_desarrollo)
  AS antes \gset
SET SESSION AUTHORIZATION vec_ct115_runtime;
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT pg_temp.confirmar('confirmar_no_incorporacion_v1',:'ni'::jsonb)::text AS r_ni_b \gset
COMMIT;
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT pg_temp.continuar(:'mat_conf')::text AS cont_b \gset
COMMIT;
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION vec_b42_relevo;
SELECT to_jsonb(x)::text AS b1_b FROM vec_bolsa_llamamientos.registrar_no_incorporacion_bolsa_v1((:'pub'::jsonb)->'evento',(:'pub'::jsonb)->>'huella_sha256',
  ((:'pub'::jsonb)->>'origen_creada_en')::timestamptz,((:'pub'::jsonb)->>'origen_posicion')::bigint,pg_temp.plazos()) x \gset
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION vec_b42_runtime;
SET TimeZone='UTC'; SET statement_timeout='15s'; SET idle_in_transaction_session_timeout='20s';
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT pg_temp.guardar_bolsa(convert_to(:'sig_bolsa','UTF8'),'bolsa.llamamiento.siguiente.abrir')::text AS g_sig_b \gset
COMMIT;
RESET SESSION AUTHORIZATION;
SELECT pg_temp.exigir((:'r_ni_b'::jsonb)->'recibo'=(:'r_ni'::jsonb)->'recibo','no incorporación: mismo recibo tras el reinicio');
SELECT pg_temp.exigir((:'b1_b'::jsonb)->>'reutilizado'='true' AND (:'b1_b'::jsonb)->>'estado'='aplicada','bandeja: reentrega reconocida tras el reinicio');
SELECT pg_temp.exigir((:'g_sig_b'::jsonb)->>'recibo_ref'=(:'g_sig'::jsonb)->>'recibo_ref','siguiente llamamiento: mismo recibo tras el reinicio');
SELECT pg_temp.exigir((:'cont_b'::jsonb)->>'Estado'='replay_confirmado' AND ((:'cont_b'::jsonb)-'Estado')=((:'cont'::jsonb)-'Estado'),'continuación: mismo recibo tras el reinicio');
SELECT pg_temp.exigir((SELECT count(*) FROM vec_contratacion_temporal.no_incorporacion_v1)||'/'||(SELECT count(*) FROM vec_contratacion_temporal.outbox_expediente_integral)
  ||'/'||(SELECT count(*) FROM vec_bolsa_llamamientos.no_incorporacion_bolsa)||'/'||(SELECT count(*) FROM vec_bolsa_llamamientos.no_incorporacion_bolsa_evaluacion)||'/'||(SELECT count(*) FROM vec_bolsa_llamamientos.sancion_participacion)
  ||'/'||(SELECT count(*) FROM vec_bolsa_llamamientos.situacion_participacion)||'/'||(SELECT count(*) FROM vec_bolsa_llamamientos.integracion_desarrollo)=:'antes',
  'ninguna fila nueva tras el reinicio ('||:'antes'||')');
SELECT 'reinicio no incorporación OK';
