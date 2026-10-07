-- B70: resultado de adjudicación telemática sobre datos sintéticos.
-- Se comprueban proyección, orden/disponibilidad e historia sin reemplazar AD3.
-- La preparación se revierte completa; no acredita consumo V3 de una escritura.
\set ON_ERROR_STOP on
BEGIN;
SET LOCAL timezone = 'UTC';
SET LOCAL session_replication_role = replica;
INSERT INTO vec_bolsa_llamamientos.bolsa_constituida VALUES
 ('bolsa:r2tele:1',1,encode(sha256('{}'::bytea),'hex'),'{}'::bytea,'categoria:rpt:aux',now()-interval '10 days',NULL,'vigente',now()-interval '10 days'),
 ('bolsa:r2tele:2',1,encode(sha256('{}'::bytea),'hex'),'{}'::bytea,'categoria:rpt:aux',now()-interval '10 days',NULL,'vigente',now()-interval '10 days');
INSERT INTO vec_bolsa_llamamientos.instantanea_orden_bolsa(instantanea_ref,version,huella_instantanea_sha256,instantanea_canonica,bolsa_ref,version_bolsa,huella_bolsa_sha256,total_participaciones,referida_en,generada_en,registrada_en)
VALUES ('inst:r2tele:1',1,encode(sha256('{}'::bytea),'hex'),'{}'::bytea,'bolsa:r2tele:1',1,encode(sha256('{}'::bytea),'hex'),4,now()-interval '10 days',now()-interval '10 days',now()-interval '10 days'),
       ('inst:r2tele:2',1,encode(sha256('{}'::bytea),'hex'),'{}'::bytea,'bolsa:r2tele:2',1,encode(sha256('{}'::bytea),'hex'),1,now()-interval '10 days',now()-interval '10 days',now()-interval '10 days');
INSERT INTO vec_bolsa_llamamientos.constitucion VALUES
 ('acta:r2tele:1','bolsa:r2tele:1',1,encode(sha256('{}'::bytea),'hex'),'inst:r2tele:1',1,encode(sha256('{}'::bytea),'hex'),'categoria:rpt:aux','per_actoractoractoractoractor',now()-interval '10 days',now()-interval '10 days'),
 ('acta:r2tele:2','bolsa:r2tele:2',1,encode(sha256('{}'::bytea),'hex'),'inst:r2tele:2',1,encode(sha256('{}'::bytea),'hex'),'categoria:rpt:aux','per_actoractoractoractoractor',now()-interval '10 days',now()-interval '10 days');
INSERT INTO vec_bolsa_llamamientos.constitucion_entrada VALUES
 ('inst:r2tele:1',1,1,'part:r2tele:1',1),('inst:r2tele:1',1,2,'part:r2tele:2',2),('inst:r2tele:1',1,3,'part:r2tele:3',3),('inst:r2tele:1',1,4,'part:r2tele:4',4),
 ('inst:r2tele:2',1,1,'part:r2tele:9',1);
-- Candidato A en la posición 3, B en la 2 y D en la 4; el candidato C solo
-- está en la otra bolsa.
INSERT INTO vec_bolsa_llamamientos.vinculo_candidato VALUES
 ('part:r2tele:3','can_r2teleosicion_candidato_A01','acta:r2tele:1','inst:r2tele:1',1,now()-interval '10 days'),
 ('part:r2tele:2','can_r2teleosicion_candidato_B01','acta:r2tele:1','inst:r2tele:1',1,now()-interval '10 days'),
 ('part:r2tele:4','can_r2teleosicion_candidato_D01','acta:r2tele:1','inst:r2tele:1',1,now()-interval '10 days'),
 ('part:r2tele:9','can_r2teleosicion_candidato_C01','acta:r2tele:2','inst:r2tele:2',1,now()-interval '10 days');
INSERT INTO vec_bolsa_llamamientos.situacion_participacion(participacion_ref,situacion,desde,hasta,fecha_disponible,motivo,actor,registrada_en,clave_idempotencia,recibo_ref)
SELECT p, 'disponible', now()-interval '9 days', NULL, NULL, 'alta', 'per_actoractoractoractoractor', now()-interval '9 days', 'clave:'||p, 'recibo:'||p
  FROM unnest(ARRAY['part:r2tele:1','part:r2tele:2','part:r2tele:3','part:r2tele:4']) p;
INSERT INTO vec_bolsa_llamamientos.politica_orden_bolsa(politica_ref,bolsa_ref,version,criterio,tipo_lista,reposicion,provisional,rotulo,actor,vigente_desde,vigente_hasta,registrada_en)
VALUES ('politica:r2tele:1','bolsa:r2tele:1',1,'puntuacion_desc_acta','rotatoria','misma_posicion',false,'Ejemplo','per_actoractoractoractoractor',now()-interval '10 days',NULL,now()-interval '10 days');
SET LOCAL session_replication_role = origin;


INSERT INTO vec_bolsa_llamamientos.politica_ofertas_version(
 bolsa_ref,version,politica,huella_sha256,ejemplo,actor_ref,clave_idempotencia,version_esperada,recibo_ref,publicada_en,decision_ref,auditoria_ref)
SELECT 'bolsa:r2tele:1',v,
 jsonb_build_object('plazo',jsonb_build_object('unidad','dias_naturales','cantidad',2,'computo','administrativo','municipio_sede','18087'),
 'adjudicacion',jsonb_build_object('criterio','orden_vigente','elegibilidad','disposicion_en_plazo')
   || CASE WHEN v=2 THEN '{"confirmacion":"aceptacion_previa"}'::jsonb ELSE '{}'::jsonb END,
 'no_cubierta',jsonb_build_object('accion','llamamiento_directo','condicion','sin_disposiciones_elegibles'),
 'plazas',jsonb_build_object('llamada','simultanea','respuesta_horas',24,'tras_renuncia','siguiente_en_orden')),
 repeat('c',64),true,'per_actoractoractoractoractor','r2tele-politica-'||v,v-1,
 'recibo:politica-ofertas:'||encode(sha256(convert_to('r2tele-politica-'||v,'UTF8')),'hex'),
 now()-interval '3 days','decision:r2tele-politica-'||v,'auditoria:r2tele-politica-'||v
FROM generate_series(1,2) v;

-- Siembra de historia previa, no una publicación: se pausa el trigger B71
-- sólo mientras se prepara el fixture y se vuelve a origin antes de leer.
SET LOCAL session_replication_role=replica;
INSERT INTO vec_bolsa_llamamientos.oferta_publicada(
 oferta_ref,recibo_ref,bolsa_ref,actor_ref,clave_idempotencia,datos,plazo,publicada_en,vence_antes_de,huella_comando_sha256,decision_ref)
SELECT 'oferta:'||encode(sha256(convert_to(nombre,'UTF8')),'hex'),
 'recibo:oferta:'||encode(sha256(convert_to(nombre,'UTF8')),'hex'),
 'bolsa:r2tele:1','per_actoractoractoractoractor',nombre,
 '{"categoria":"Auxiliar administrativo","centro":"Residencia Sierra","fecha_inicio":"2026-10-01","descripcion":"Sustitución por baja"}'::jsonb,
 jsonb_build_object('politica_version',v),now()-interval '3 days',now()-interval '1 day',repeat('d',64),'decision:'||nombre
FROM (VALUES ('r2tele-historica',1),('r2tele-nueva',2),('r2tele-sin-aceptaciones',2),('r2tele-adjudicada',2)) f(nombre,v);
INSERT INTO vec_bolsa_llamamientos.plazas_oferta
SELECT oferta_ref,bolsa_ref,1,(plazo->>'politica_version')::bigint,publicada_en FROM vec_bolsa_llamamientos.oferta_publicada
WHERE bolsa_ref='bolsa:r2tele:1';
INSERT INTO vec_bolsa_llamamientos.disposicion_oferta
SELECT o.oferta_ref,p,now()-interval '2 days','r2tele-acepta-'||p,
 'recibo:disposicion:'||encode(sha256(convert_to(o.oferta_ref||p,'UTF8')),'hex')
FROM vec_bolsa_llamamientos.oferta_publicada o
CROSS JOIN unnest(ARRAY['part:r2tele:1','part:r2tele:2','part:r2tele:3']) p
WHERE o.clave_idempotencia IN ('r2tele-historica','r2tele-nueva','r2tele-adjudicada');
-- Mejor posición del acta no disponible: nunca puede adjudicarse.
INSERT INTO vec_bolsa_llamamientos.situacion_participacion
 (participacion_ref,situacion,desde,hasta,fecha_disponible,motivo,actor,registrada_en,clave_idempotencia,recibo_ref)
VALUES ('part:r2tele:1','no_disponible',now()-interval '1 hour',NULL,NULL,'pausa',
 'per_actoractoractoractoractor',now()-interval '1 hour','r2tele-pausa-1','recibo:r2tele-pausa-1');

INSERT INTO vec_bolsa_llamamientos.acto_plaza_oferta
 (oferta_ref,numero_de_plaza,secuencia,tipo,participacion_ref,orden_vigente,responder_antes_de,
 recibo_ref,actor_ref,clave_idempotencia,huella_comando_sha256,decision_ref,auditoria_ref,registrado_en)
SELECT oferta_ref,1,1,'adjudicada','part:r2tele:2',1,
 CASE WHEN clave_idempotencia='r2tele-historica' THEN now()+interval '1 day' END,
 'recibo:plaza-oferta:'||encode(sha256(convert_to(oferta_ref,'UTF8')),'hex'),
 'per_actoractoractoractoractor','r2tele-adjudica-'||clave_idempotencia,repeat('a',64),
 'decision:acto:'||clave_idempotencia,'auditoria:acto:'||clave_idempotencia,now()-interval '1 hour'
FROM vec_bolsa_llamamientos.oferta_publicada WHERE clave_idempotencia IN ('r2tele-historica','r2tele-adjudicada');

SET LOCAL session_replication_role=origin;

DO $prueba$
DECLARE j jsonb; r record; situaciones bigint; actos bigint;
BEGIN
 SELECT count(*) INTO situaciones FROM vec_bolsa_llamamientos.situacion_participacion;
 SELECT count(*) INTO actos FROM vec_bolsa_llamamientos.acto_plaza_oferta;
 FOR r IN SELECT oferta_ref,clave_idempotencia AS nombre FROM vec_bolsa_llamamientos.oferta_publicada
           WHERE bolsa_ref='bolsa:r2tele:1' LOOP
  j:=vec_bolsa_llamamientos.proyectar_oferta_v2(r.oferta_ref,now());
  CASE r.nombre
   WHEN 'r2tele-historica' THEN
    IF j#>>'{plazas,0,estado}' IS DISTINCT FROM 'pendiente_respuesta' OR coalesce(j#>'{plazas,0,responder_antes_de}','null'::jsonb)='null'::jsonb
    THEN RAISE EXCEPTION 'B70: historia anterior alterada'; END IF;
   WHEN 'r2tele-nueva' THEN
    IF j#>>'{plazas,0,propuesta,participacion_ref}' IS DISTINCT FROM 'part:r2tele:2'
       OR j#>>'{plazas,0,propuesta,orden_vigente}' IS DISTINCT FROM '1'
       OR j->>'confirmacion_adjudicacion' IS DISTINCT FROM 'aceptacion_previa'
    THEN RAISE EXCEPTION 'B70: no propone a la mejor aceptación disponible'; END IF;
   WHEN 'r2tele-sin-aceptaciones' THEN
    IF j#>>'{plazas,0,propuesta,tipo}' IS DISTINCT FROM 'llamamiento_directo'
    THEN RAISE EXCEPTION 'B70: sin aceptaciones no propone llamamiento directo'; END IF;
   WHEN 'r2tele-adjudicada' THEN
    IF j->>'estado' IS DISTINCT FROM 'adjudicada' OR j#>>'{plazas,0,estado}' IS DISTINCT FROM 'cubierta'
       OR j#>'{plazas,0,responder_antes_de}' IS DISTINCT FROM 'null'::jsonb
       OR j#>'{plazas,0,puede_sin_respuesta}' IS DISTINCT FROM 'false'::jsonb
       OR jsonb_array_length(j#>'{plazas,0,historial}') IS DISTINCT FROM 1
    THEN RAISE EXCEPTION 'B70: adjudicación exige una segunda respuesta o pierde historia'; END IF;
  END CASE;
 END LOOP;
 IF situaciones<>(SELECT count(*) FROM vec_bolsa_llamamientos.situacion_participacion)
    OR actos<>(SELECT count(*) FROM vec_bolsa_llamamientos.acto_plaza_oferta)
 THEN RAISE EXCEPTION 'B70: silencio o consulta modificó historia'; END IF;
 IF (SELECT situacion FROM vec_bolsa_llamamientos.situacion_participacion WHERE participacion_ref='part:r2tele:3'
      ORDER BY desde DESC LIMIT 1) IS DISTINCT FROM 'disponible'
 THEN RAISE EXCEPTION 'B70: aceptación no elegida pierde disponibilidad'; END IF;
 RAISE NOTICE 'B70 RESULTADO-OK: mejor aceptación disponible, sin respuesta adicional, silencio inocuo, historia intacta y directo sin aceptaciones';
END $prueba$;
ROLLBACK;
