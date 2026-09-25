-- Bolsa 000040: una bolsa vigente con tres participaciones; A y B vinculadas
-- a candidatos. A tiene un contacto de origen CONVOCA (versión 1); B no tiene
-- contacto. Siembra como superusuario con session_replication_role=replica.
\set ON_ERROR_STOP on
BEGIN;
SET LOCAL timezone = 'UTC';
SET LOCAL session_replication_role = replica;
INSERT INTO vec_bolsa_llamamientos.bolsa_constituida VALUES
 ('bolsa:cp:1',1,encode(sha256('{}'::bytea),'hex'),'{}'::bytea,'categoria:rpt:aux',now()-interval '10 days',NULL,'vigente',now()-interval '10 days');
INSERT INTO vec_bolsa_llamamientos.instantanea_orden_bolsa(instantanea_ref,version,huella_instantanea_sha256,instantanea_canonica,bolsa_ref,version_bolsa,huella_bolsa_sha256,total_participaciones,referida_en,generada_en,registrada_en)
VALUES ('inst:cp:1',1,encode(sha256('{}'::bytea),'hex'),'{}'::bytea,'bolsa:cp:1',1,encode(sha256('{}'::bytea),'hex'),3,now()-interval '10 days',now()-interval '10 days',now()-interval '10 days');
INSERT INTO vec_bolsa_llamamientos.constitucion VALUES
 ('acta:cp:1','bolsa:cp:1',1,encode(sha256('{}'::bytea),'hex'),'inst:cp:1',1,encode(sha256('{}'::bytea),'hex'),'categoria:rpt:aux','per_actoractoractoractoractor',now()-interval '10 days',now()-interval '10 days');
INSERT INTO vec_bolsa_llamamientos.constitucion_entrada VALUES ('inst:cp:1',1,1,'part:cp:1',1),('inst:cp:1',1,2,'part:cp:2',2),('inst:cp:1',1,3,'part:cp:3',3);
INSERT INTO vec_bolsa_llamamientos.vinculo_candidato VALUES
 ('part:cp:1','can_contacto_propio_candidato_A1','acta:cp:1','inst:cp:1',1,now()-interval '10 days'),
 ('part:cp:2','can_contacto_propio_candidato_B1','acta:cp:1','inst:cp:1',1,now()-interval '10 days');
INSERT INTO vec_bolsa_llamamientos.datos_contacto_participacion VALUES
 ('part:cp:1',1,'kms:prueba',decode(repeat('01',12),'hex'),decode(repeat('02',32),'hex'),'Importado de CONVOCA','per_actoractoractoractoractor',now()-interval '2 days','convoca-1','recibo:contacto:cp1');
INSERT INTO vec_bolsa_llamamientos.origen_datos_contacto_participacion VALUES
 ('part:cp:1',1,'convoca',now()+interval '1 year',(now()+interval '1 year')::date,'vec.bolsa.reglas:1:b29.contacto_origen_convoca',repeat('d',64),now()-interval '2 days');
SET LOCAL session_replication_role = origin;

CREATE SCHEMA prueba_cp;
-- Material sintético de la acción propia (el núcleo doble no verifica
-- firmas); los parámetros permiten alterar cada pieza para los negativos.
CREATE FUNCTION prueba_cp.confirmar(p_candidato text, p_version bigint, p_clave text,
  p_efecto text DEFAULT NULL, p_accion text DEFAULT 'bolsa.participaciones_propias.confirmar_contacto',
  p_audiencia text DEFAULT 'vec_bolsa_llamamientos.participaciones_propias.confirmar_contacto.v1',
  p_tipo text DEFAULT 'participaciones_candidato', p_vinculos jsonb DEFAULT NULL, p_repetida boolean DEFAULT false,
  p_bolsa text DEFAULT 'bolsa:cp:1')
RETURNS TABLE(reutilizada boolean, recibo_ref text, version bigint, confirmada_en timestamptz)
LANGUAGE sql AS $f$
 SELECT * FROM vec_bolsa_llamamientos.confirmar_contacto_propio_v1(
  p_candidato, p_bolsa, p_version, p_clave,
  'recibo:confirmacion-contacto:'||encode(sha256(convert_to(p_candidato||p_bolsa||p_clave,'UTF8')),'hex'), clock_timestamp(),
  convert_to((jsonb_build_object('efecto_ref', coalesce(p_efecto, 'mi-bolsa:'||p_candidato), 'operacion', p_accion,
    'audiencia_consumo', p_audiencia, 'huella_efecto_sha256', repeat('e',64), 'suite', 'VEC-AD-3-COSE-EDDSA-1', 'nonce', gen_random_uuid())
    || CASE WHEN p_repetida THEN '{"repetida":true}'::jsonb ELSE '{}'::jsonb END)::text, 'UTF8'),
  convert_to(jsonb_build_object('recurso_ref', coalesce(p_efecto, 'mi-bolsa:'||p_candidato), 'accion', p_accion, 'tipo_recurso', p_tipo,
    'modulo_id', 'bolsa', 'finalidad', 'gestion_participaciones_propias', 'contexto_recurso_huella_sha256', repeat('e',64),
    'campos_permitidos', '[]'::jsonb, 'obligaciones', '[]'::jsonb)::text, 'UTF8'),
  '\x00'::bytea,
  convert_to(jsonb_build_object('vinculos', coalesce(p_vinculos, jsonb_build_array(
    jsonb_build_object('tipo','candidato','estado','activo','referencia',p_candidato))))::text, 'UTF8'),
  1, 1, '\x00'::bytea, '\x00'::bytea, '\x00'::bytea, '\x00'::bytea)
$f$;
-- Consulta Mi bolsa con material sintético (doble de AD3): consume la
-- decisión propia y deja la marca que exigen las lecturas del candidato.
CREATE FUNCTION prueba_cp.consultar(p_candidato text) RETURNS jsonb LANGUAGE sql AS $f$
 SELECT vec_bolsa_llamamientos.consultar_mi_bolsa_portal_v1(p_candidato, clock_timestamp(),
  convert_to(jsonb_build_object('efecto_ref','mi-bolsa:'||p_candidato,'huella_efecto_sha256',repeat('e',64),'nonce',gen_random_uuid())::text,'UTF8'),
  convert_to(jsonb_build_object('recurso_ref','mi-bolsa:'||p_candidato,'contexto_recurso_huella_sha256',repeat('e',64))::text,'UTF8'),
  '\x00'::bytea,
  convert_to(jsonb_build_object('vinculos', jsonb_build_array(jsonb_build_object('tipo','candidato','estado','activo','referencia',p_candidato)))::text,'UTF8'),
  1, 1, '\x00'::bytea, '\x00'::bytea, '\x00'::bytea, '\x00'::bytea)
$f$;
CREATE FUNCTION prueba_cp.contacto(p_candidato text) RETURNS jsonb LANGUAGE sql AS $f$
 SELECT prueba_cp.consultar(p_candidato);
 SELECT vec_bolsa_llamamientos.leer_contacto_candidato_v1(p_candidato, clock_timestamp());
$f$;
CREATE FUNCTION prueba_cp.espera(p_sql text, p_codigo text) RETURNS void LANGUAGE plpgsql AS $f$
BEGIN
 EXECUTE p_sql;
 RAISE EXCEPTION 'se esperaba % en: %', p_codigo, p_sql;
EXCEPTION WHEN OTHERS THEN
 IF SQLSTATE <> p_codigo THEN RAISE EXCEPTION 'código % (%) en lugar de % en: %', SQLSTATE, SQLERRM, p_codigo, p_sql; END IF;
END $f$;
GRANT USAGE ON SCHEMA prueba_cp TO vec_bolsa_llamamientos_ejecutor;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA prueba_cp TO vec_bolsa_llamamientos_ejecutor;
COMMIT;
