-- Bolsa 000029: una bolsa vigente con cuatro participaciones, tres de ellas
-- vinculadas a candidatos, orden vigente y funciones auxiliares de prueba.
-- Se siembra como superusuario con session_replication_role=replica (sin la
-- cadena de acta), igual que el focal de 000028.
\set ON_ERROR_STOP on
BEGIN;
SET LOCAL timezone = 'UTC';
SET LOCAL session_replication_role = replica;
INSERT INTO vec_bolsa_llamamientos.bolsa_constituida VALUES
 ('bolsa:disp:1',1,encode(sha256('{}'::bytea),'hex'),'{}'::bytea,'categoria:rpt:aux',now()-interval '10 days',NULL,'vigente',now()-interval '10 days'),
 ('bolsa:disp:2',1,encode(sha256('{}'::bytea),'hex'),'{}'::bytea,'categoria:rpt:aux',now()-interval '10 days',NULL,'vigente',now()-interval '10 days');
INSERT INTO vec_bolsa_llamamientos.instantanea_orden_bolsa(instantanea_ref,version,huella_instantanea_sha256,instantanea_canonica,bolsa_ref,version_bolsa,huella_bolsa_sha256,total_participaciones,referida_en,generada_en,registrada_en)
VALUES ('inst:disp:1',1,encode(sha256('{}'::bytea),'hex'),'{}'::bytea,'bolsa:disp:1',1,encode(sha256('{}'::bytea),'hex'),4,now()-interval '10 days',now()-interval '10 days',now()-interval '10 days'),
       ('inst:disp:2',1,encode(sha256('{}'::bytea),'hex'),'{}'::bytea,'bolsa:disp:2',1,encode(sha256('{}'::bytea),'hex'),1,now()-interval '10 days',now()-interval '10 days',now()-interval '10 days');
INSERT INTO vec_bolsa_llamamientos.constitucion VALUES
 ('acta:disp:1','bolsa:disp:1',1,encode(sha256('{}'::bytea),'hex'),'inst:disp:1',1,encode(sha256('{}'::bytea),'hex'),'categoria:rpt:aux','per_actoractoractoractoractor',now()-interval '10 days',now()-interval '10 days'),
 ('acta:disp:2','bolsa:disp:2',1,encode(sha256('{}'::bytea),'hex'),'inst:disp:2',1,encode(sha256('{}'::bytea),'hex'),'categoria:rpt:aux','per_actoractoractoractoractor',now()-interval '10 days',now()-interval '10 days');
INSERT INTO vec_bolsa_llamamientos.constitucion_entrada VALUES
 ('inst:disp:1',1,1,'part:disp:1',1),('inst:disp:1',1,2,'part:disp:2',2),('inst:disp:1',1,3,'part:disp:3',3),('inst:disp:1',1,4,'part:disp:4',4),
 ('inst:disp:2',1,1,'part:disp:9',1);
-- Candidato A en la posición 3, B en la 2 y D en la 4; el candidato C solo
-- está en la otra bolsa.
INSERT INTO vec_bolsa_llamamientos.vinculo_candidato VALUES
 ('part:disp:3','can_disposicion_candidato_A01','acta:disp:1','inst:disp:1',1,now()-interval '10 days'),
 ('part:disp:2','can_disposicion_candidato_B01','acta:disp:1','inst:disp:1',1,now()-interval '10 days'),
 ('part:disp:4','can_disposicion_candidato_D01','acta:disp:1','inst:disp:1',1,now()-interval '10 days'),
 ('part:disp:9','can_disposicion_candidato_C01','acta:disp:2','inst:disp:2',1,now()-interval '10 days');
INSERT INTO vec_bolsa_llamamientos.situacion_participacion(participacion_ref,situacion,desde,hasta,fecha_disponible,motivo,actor,registrada_en,clave_idempotencia,recibo_ref)
SELECT p, 'disponible', now()-interval '9 days', NULL, NULL, 'alta', 'per_actoractoractoractoractor', now()-interval '9 days', 'clave:'||p, 'recibo:'||p
  FROM unnest(ARRAY['part:disp:1','part:disp:2','part:disp:3','part:disp:4']) p;
INSERT INTO vec_bolsa_llamamientos.politica_orden_bolsa(politica_ref,bolsa_ref,version,criterio,tipo_lista,reposicion,provisional,rotulo,actor,vigente_desde,vigente_hasta,registrada_en)
VALUES ('politica:disp:1','bolsa:disp:1',1,'puntuacion_desc_acta','rotatoria','misma_posicion',false,'Ejemplo','per_actoractoractoractoractor',now()-interval '10 days',NULL,now()-interval '10 days');
SET LOCAL session_replication_role = origin;

CREATE SCHEMA prueba_disp;
-- Material sintético de la acción propia; los parámetros permiten alterar
-- cada pieza para los negativos.
CREATE FUNCTION prueba_disp.manifestar(p_oferta text, p_candidato text, p_clave text,
  p_efecto text DEFAULT NULL, p_accion text DEFAULT 'bolsa.participaciones_propias.manifestar_disposicion',
  p_tipo text DEFAULT 'oferta_bolsa', p_vinculos jsonb DEFAULT NULL, p_repetida boolean DEFAULT false,
  p_en timestamptz DEFAULT NULL)
RETURNS TABLE(reutilizada boolean, recibo_ref text, oferta_ref text, manifestada_en timestamptz)
LANGUAGE sql AS $f$
 SELECT * FROM vec_bolsa_llamamientos.manifestar_disposicion_oferta_v1(
  p_oferta, 'recibo:disposicion:'||encode(sha256(convert_to(p_oferta||p_candidato||p_clave,'UTF8')),'hex'), p_candidato, p_clave,
  coalesce(p_en, clock_timestamp()),
  convert_to((jsonb_build_object('efecto_ref', coalesce(p_efecto, p_oferta), 'operacion', p_accion, 'nonce', gen_random_uuid())
    || CASE WHEN p_repetida THEN '{"repetida":true}'::jsonb ELSE '{}'::jsonb END)::text, 'UTF8'),
  convert_to(jsonb_build_object('recurso_ref', coalesce(p_efecto, p_oferta), 'accion', p_accion, 'tipo_recurso', p_tipo)::text, 'UTF8'),
  '\x00'::bytea,
  convert_to(jsonb_build_object('vinculos', coalesce(p_vinculos, jsonb_build_array(
    jsonb_build_object('tipo','candidato','estado','activo','referencia',p_candidato),
    jsonb_build_object('tipo','empleado','estado','activo','referencia','emp_x'))))::text, 'UTF8'),
  1, 1, '\x00'::bytea, '\x00'::bytea, '\x00'::bytea, '\x00'::bytea)
$f$;
CREATE FUNCTION prueba_disp.publicar(p_oferta text, p_vence interval)
RETURNS jsonb LANGUAGE sql AS $f$
 SELECT (vec_bolsa_llamamientos.publicar_oferta_v1(p_oferta, 'recibo:oferta:'||substr(p_oferta,8), 'bolsa:disp:1', 'per_actoractoractoractoractor', 'clave-'||substr(p_oferta,8,12),
   '{"categoria":"Auxiliar administrativo","centro":"Residencia Sierra","fecha_inicio":"2026-10-01","descripcion":"Sustitución por baja"}'::jsonb,
   ('{"regla_ref":"vec.bolsa.reglas:1:b10.plazo_publicacion","huella_catalogo":"'||repeat('d',64)||'","unidad":"dias_habiles","cantidad":2,"computo":"administrativo","ultimo_dia":"2026-10-02","ejemplo":false}')::jsonb,
   clock_timestamp(), clock_timestamp() + p_vence,
   convert_to('{"efecto_ref":"bolsa:disp:1","nonce":"'||gen_random_uuid()||'"}','UTF8'),
   convert_to('{"principal_id":"per_actoractoractoractoractor","accion":"llamamiento.emitir.v1","modulo_id":"bolsa","finalidad":"gestion_llamamientos_bolsa","recurso_ref":"bolsa:disp:1","tipo_recurso":"bolsa_constituida"}','UTF8'),
   '\x00'::bytea,'\x00'::bytea,1,1,'\x00'::bytea,'\x00'::bytea,'\x00'::bytea,'\x00'::bytea)).oferta
$f$;
CREATE FUNCTION prueba_disp.espera(p_sql text, p_codigo text) RETURNS void LANGUAGE plpgsql AS $f$
BEGIN
 EXECUTE p_sql;
 RAISE EXCEPTION 'se esperaba % en: %', p_codigo, p_sql;
EXCEPTION WHEN OTHERS THEN
 IF SQLSTATE <> p_codigo THEN RAISE EXCEPTION 'código % (%) en lugar de % en: %', SQLSTATE, SQLERRM, p_codigo, p_sql; END IF;
END $f$;
GRANT USAGE ON SCHEMA prueba_disp TO vec_bolsa_llamamientos_ejecutor;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA prueba_disp TO vec_bolsa_llamamientos_ejecutor;
COMMIT;
