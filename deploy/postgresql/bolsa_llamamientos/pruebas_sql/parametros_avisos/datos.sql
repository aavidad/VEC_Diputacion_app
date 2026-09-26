-- Bolsa 000041: dos bolsas vigentes. Personas (vínculo de candidato):
--  A: part:pa:1 (bolsa 1) y part:pa:5 (bolsa 2, trabajando hace 37 meses).
--  B: part:pa:2 (bolsa 1, renuncia desde el portal sin reflejar) y part:pa:6
--     (bolsa 2, trabajando hace 35 meses y 10 días; dos contratos solapados
--     que sumados superarían 18 meses pero unidos son 11).
--  C: part:pa:3 (bolsa 1): dos contratos, 10 meses cerrados y 9 abiertos.
--  part:pa:4 (bolsa 1, sin vínculo): un contrato de 4 meses y una solicitud
--     de pausa del portal pendiente de validar.
-- Siembra como superusuario con session_replication_role=replica.
\set ON_ERROR_STOP on
BEGIN;
SET LOCAL timezone = 'UTC';
SET LOCAL session_replication_role = replica;
INSERT INTO vec_bolsa_llamamientos.bolsa_constituida VALUES
 ('bolsa:pa:1',1,encode(sha256('{}'::bytea),'hex'),'{}'::bytea,'categoria:rpt:aux',now()-interval '5 years',NULL,'vigente',now()-interval '5 years'),
 ('bolsa:pa:2',1,encode(sha256('{}'::bytea),'hex'),'{}'::bytea,'categoria:rpt:adm',now()-interval '5 years',NULL,'vigente',now()-interval '5 years');
INSERT INTO vec_bolsa_llamamientos.instantanea_orden_bolsa(instantanea_ref,version,huella_instantanea_sha256,instantanea_canonica,bolsa_ref,version_bolsa,huella_bolsa_sha256,total_participaciones,referida_en,generada_en,registrada_en)
VALUES ('inst:pa:1',1,encode(sha256('{}'::bytea),'hex'),'{}'::bytea,'bolsa:pa:1',1,encode(sha256('{}'::bytea),'hex'),4,now()-interval '5 years',now()-interval '5 years',now()-interval '5 years'),
       ('inst:pa:2',1,encode(sha256('{}'::bytea),'hex'),'{}'::bytea,'bolsa:pa:2',1,encode(sha256('{}'::bytea),'hex'),2,now()-interval '5 years',now()-interval '5 years',now()-interval '5 years');
INSERT INTO vec_bolsa_llamamientos.constitucion VALUES
 ('acta:pa:1','bolsa:pa:1',1,encode(sha256('{}'::bytea),'hex'),'inst:pa:1',1,encode(sha256('{}'::bytea),'hex'),'categoria:rpt:aux','per_actoractoractoractoractor',now()-interval '5 years',now()-interval '5 years'),
 ('acta:pa:2','bolsa:pa:2',1,encode(sha256('{}'::bytea),'hex'),'inst:pa:2',1,encode(sha256('{}'::bytea),'hex'),'categoria:rpt:adm','per_actoractoractoractoractor',now()-interval '5 years',now()-interval '5 years');
INSERT INTO vec_bolsa_llamamientos.constitucion_entrada VALUES
 ('inst:pa:1',1,1,'part:pa:1',1),('inst:pa:1',1,2,'part:pa:2',2),('inst:pa:1',1,3,'part:pa:3',3),('inst:pa:1',1,4,'part:pa:4',4),
 ('inst:pa:2',1,1,'part:pa:5',1),('inst:pa:2',1,2,'part:pa:6',2);
INSERT INTO vec_bolsa_llamamientos.vinculo_candidato VALUES
 ('part:pa:1','can_parametros_avisos_persona_A1','acta:pa:1','inst:pa:1',1,now()-interval '5 years'),
 ('part:pa:2','can_parametros_avisos_persona_B1','acta:pa:1','inst:pa:1',1,now()-interval '5 years'),
 ('part:pa:3','can_parametros_avisos_persona_C1','acta:pa:1','inst:pa:1',1,now()-interval '5 years'),
 ('part:pa:5','can_parametros_avisos_persona_A1','acta:pa:2','inst:pa:2',1,now()-interval '5 years'),
 ('part:pa:6','can_parametros_avisos_persona_B1','acta:pa:2','inst:pa:2',1,now()-interval '5 years');
INSERT INTO vec_bolsa_llamamientos.situacion_participacion(participacion_ref,situacion,desde,motivo,actor,registrada_en,clave_idempotencia,recibo_ref) VALUES
 ('part:pa:5','trabajando',now()-interval '37 months','Nombramiento sintético','per_actoractoractoractoractor',now()-interval '37 months','pa:5:t','recibo:pa:5:t'),
 ('part:pa:6','trabajando',now()-interval '35 months'-interval '10 days','Nombramiento sintético','per_actoractoractoractoractor',now()-interval '35 months','pa:6:t','recibo:pa:6:t');
-- Llamamiento de la renuncia comunicada desde el portal.
INSERT INTO vec_bolsa_llamamientos.llamamiento_emitido(llamamiento_ref,recibo_ref,bolsa_ref,actor_ref,clave_idempotencia,participaciones,configuracion,huella_comando_sha256,huella_finalizacion,estado,emitido_en,decision_ref)
VALUES ('llamamiento:'||repeat('a',64),'recibo:llamamiento:'||repeat('a',64),'bolsa:pa:1','per_actoractoractoractoractor','pa:emision:1','["part:pa:2"]','{}',repeat('b',64),decode(repeat('0c',32),'hex'),'emision_reservada',now()-interval '3 days','decision:pa:1');
INSERT INTO vec_bolsa_llamamientos.respuesta_portal_llamamiento(respuesta_ref,recibo_ref,bolsa_ref,participacion_ref,candidato_ref,llamamiento_ref,respuesta,modo,contacto_en,vence_antes_de,regla_ref,clave_idempotencia,decision_ref,respondida_en)
VALUES ('respuesta-portal:'||repeat('d',64),'recibo:respuesta-portal:'||repeat('d',64),'bolsa:pa:1','part:pa:2','can_parametros_avisos_persona_B1','llamamiento:'||repeat('a',64),'renuncia','propuesta_rrhh',now()-interval '2 days',now()+interval '1 day','vec.bolsa.reglas:1:b29.portal_candidato','clave-respuesta-pa','decision:pa:respuesta',now()-interval '1 day');
INSERT INTO vec_bolsa_llamamientos.solicitud_portal_candidato(solicitud_ref,recibo_ref,bolsa_ref,participacion_ref,candidato_ref,tipo,pausa_hasta,situacion_previa,regla_ref,clave_idempotencia,decision_ref,registrada_en)
VALUES ('solicitud-portal:'||repeat('e',64),'recibo:solicitud-portal:'||repeat('e',64),'bolsa:pa:1','part:pa:4','can_parametros_avisos_persona_D1','pausa',now()+interval '6 months','disponible','vec.bolsa.reglas:1:b18.pausa_voluntaria','clave-solicitud-pa','decision:pa:solicitud',now()-interval '2 days');
SET LOCAL session_replication_role = origin;
COMMIT;

-- Histórico de contratos (000024): evento completo con su huella exacta.
CREATE SCHEMA prueba_pa;
CREATE FUNCTION prueba_pa.contrato(p_tipo text, p_llamamiento text, p_participacion text, p_bolsa text,
  p_inicio timestamptz, p_fin timestamptz, p_posicion bigint) RETURNS void LANGUAGE plpgsql AS $f$
DECLARE v_origen text := 'origen:pa:' || p_tipo || ':' || p_llamamiento;
 v_ref text := 'evento:ct:contrato-bolsa:' || encode(sha256(convert_to(p_tipo || chr(31) || v_origen, 'UTF8')), 'hex');
 v_evento jsonb;
 fmt text := 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"';
BEGIN
 SET LOCAL timezone = 'UTC';
 v_evento := jsonb_build_object('esquema','vec.contratacion-temporal.contrato-bolsa.v1','evento_ref',v_ref,'tipo',p_tipo,
   'origen_ref',v_origen,'organizacion_ref','org:pa','expediente_ref','exp:pa:'||p_llamamiento,'llamamiento_ref',p_llamamiento,
   'inicio',to_char(p_inicio,fmt),'fin_previsto',CASE WHEN p_fin IS NULL THEN NULL ELSE to_char(p_fin,fmt) END,
   'modalidad_clave','vacante','categoria_ref','categoria:rpt:aux','causa_clave',NULL,'ocurrido_en',to_char(coalesce(p_fin,p_inicio),fmt));
 INSERT INTO vec_bolsa_llamamientos.contrato_participacion(evento_ref,huella_sha256,evento,origen_ref,origen_creada_en,origen_posicion,tipo,
   organizacion_ref,expediente_ref,llamamiento_ref,participacion_ref,bolsa_ref,inicio,fin_previsto,modalidad_clave,categoria_ref,causa_clave,ocurrido_en,recibido_en)
 VALUES (v_ref,encode(sha256(convert_to(v_evento::text,'UTF8')),'hex'),v_evento,v_origen,now(),p_posicion,p_tipo,'org:pa','exp:pa:'||p_llamamiento,
   p_llamamiento,p_participacion,p_bolsa,date_trunc('microseconds',p_inicio),date_trunc('microseconds',p_fin),'vacante','categoria:rpt:aux',NULL,
   date_trunc('microseconds',coalesce(p_fin,p_inicio)),now()-interval '1 minute');
END $f$;
SELECT prueba_pa.contrato('incorporacion','llam:pa:c1','part:pa:3','bolsa:pa:1',now()-interval '20 months',now()-interval '10 months',1);
SELECT prueba_pa.contrato('cese','llam:pa:c1','part:pa:3','bolsa:pa:1',now()-interval '20 months',now()-interval '10 months',2);
SELECT prueba_pa.contrato('incorporacion','llam:pa:c2','part:pa:3','bolsa:pa:1',now()-interval '9 months',NULL,3);
SELECT prueba_pa.contrato('incorporacion','llam:pa:d1','part:pa:4','bolsa:pa:1',now()-interval '5 months',now()-interval '1 month',4);
SELECT prueba_pa.contrato('incorporacion','llam:pa:b1','part:pa:6','bolsa:pa:2',now()-interval '12 months',now()-interval '2 months',5);
SELECT prueba_pa.contrato('incorporacion','llam:pa:b2','part:pa:6','bolsa:pa:2',now()-interval '11 months',now()-interval '1 month',6);

-- Publicación y lectura como la aplicación (rol ejecutor).
CREATE FUNCTION prueba_pa.publicar(p_ref text, p_meses integer, p_antelacion integer, p_umbral integer, p_ventana integer,
  p_modo text, p_situaciones text[]) RETURNS TABLE(version bigint, reutilizada boolean) LANGUAGE sql AS $f$
 SELECT * FROM vec_bolsa_llamamientos.publicar_politica_avisos_bolsa_v1(p_ref, repeat('a',64), p_meses, p_antelacion, p_umbral, p_ventana, p_modo, p_situaciones)
$f$;
GRANT USAGE ON SCHEMA prueba_pa TO vec_bolsa_llamamientos_ejecutor;
GRANT EXECUTE ON FUNCTION prueba_pa.publicar(text,integer,integer,integer,integer,text,text[]) TO vec_bolsa_llamamientos_ejecutor;
