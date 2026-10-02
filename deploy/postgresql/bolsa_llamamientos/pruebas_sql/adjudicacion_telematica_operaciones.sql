-- B70: operaciones reales de Bolsa sobre datos sintéticos en clon desechable.
-- DOBLE CRIPTOGRÁFICO declarado: sólo estos dos consumidores AD3 devuelven
-- decisiones sintéticas. No demuestra firma V3 ni se conecta a una app/PDP.
-- DDL, datos y funciones originales se recuperan juntos mediante ROLLBACK.
\set ON_ERROR_STOP on
BEGIN;
SET LOCAL timezone = 'UTC';
SET LOCAL session_replication_role = replica;
INSERT INTO vec_bolsa_llamamientos.bolsa_constituida VALUES
 ('bolsa:r2op:1',1,encode(sha256('{}'::bytea),'hex'),'{}'::bytea,'categoria:rpt:aux',now()-interval '10 days',NULL,'vigente',now()-interval '10 days'),
 ('bolsa:r2op:2',1,encode(sha256('{}'::bytea),'hex'),'{}'::bytea,'categoria:rpt:aux',now()-interval '10 days',NULL,'vigente',now()-interval '10 days');
INSERT INTO vec_bolsa_llamamientos.instantanea_orden_bolsa(instantanea_ref,version,huella_instantanea_sha256,instantanea_canonica,bolsa_ref,version_bolsa,huella_bolsa_sha256,total_participaciones,referida_en,generada_en,registrada_en)
VALUES ('inst:r2op:1',1,encode(sha256('{}'::bytea),'hex'),'{}'::bytea,'bolsa:r2op:1',1,encode(sha256('{}'::bytea),'hex'),4,now()-interval '10 days',now()-interval '10 days',now()-interval '10 days'),
       ('inst:r2op:2',1,encode(sha256('{}'::bytea),'hex'),'{}'::bytea,'bolsa:r2op:2',1,encode(sha256('{}'::bytea),'hex'),1,now()-interval '10 days',now()-interval '10 days',now()-interval '10 days');
INSERT INTO vec_bolsa_llamamientos.constitucion VALUES
 ('acta:r2op:1','bolsa:r2op:1',1,encode(sha256('{}'::bytea),'hex'),'inst:r2op:1',1,encode(sha256('{}'::bytea),'hex'),'categoria:rpt:aux','per_actoractoractoractoractor',now()-interval '10 days',now()-interval '10 days'),
 ('acta:r2op:2','bolsa:r2op:2',1,encode(sha256('{}'::bytea),'hex'),'inst:r2op:2',1,encode(sha256('{}'::bytea),'hex'),'categoria:rpt:aux','per_actoractoractoractoractor',now()-interval '10 days',now()-interval '10 days');
INSERT INTO vec_bolsa_llamamientos.constitucion_entrada VALUES
 ('inst:r2op:1',1,1,'part:r2op:1',1),('inst:r2op:1',1,2,'part:r2op:2',2),('inst:r2op:1',1,3,'part:r2op:3',3),('inst:r2op:1',1,4,'part:r2op:4',4),
 ('inst:r2op:2',1,1,'part:r2op:9',1);
-- Candidato A en la posición 3, B en la 2 y D en la 4; el candidato C solo
-- está en la otra bolsa.
INSERT INTO vec_bolsa_llamamientos.vinculo_candidato VALUES
 ('part:r2op:3','can_r2oposicion_candidato_A01','acta:r2op:1','inst:r2op:1',1,now()-interval '10 days'),
 ('part:r2op:2','can_r2oposicion_candidato_B01','acta:r2op:1','inst:r2op:1',1,now()-interval '10 days'),
 ('part:r2op:4','can_r2oposicion_candidato_D01','acta:r2op:1','inst:r2op:1',1,now()-interval '10 days'),
 ('part:r2op:9','can_r2oposicion_candidato_C01','acta:r2op:2','inst:r2op:2',1,now()-interval '10 days');
INSERT INTO vec_bolsa_llamamientos.situacion_participacion(participacion_ref,situacion,desde,hasta,fecha_disponible,motivo,actor,registrada_en,clave_idempotencia,recibo_ref)
SELECT p, 'disponible', now()-interval '9 days', NULL, NULL, 'alta', 'per_actoractoractoractoractor', now()-interval '9 days', 'clave:'||p, 'recibo:'||p
  FROM unnest(ARRAY['part:r2op:1','part:r2op:2','part:r2op:3','part:r2op:4']) p;
INSERT INTO vec_bolsa_llamamientos.politica_orden_bolsa(politica_ref,bolsa_ref,version,criterio,tipo_lista,reposicion,provisional,rotulo,actor,vigente_desde,vigente_hasta,registrada_en)
VALUES ('politica:r2op:1','bolsa:r2op:1',1,'puntuacion_desc_acta','rotatoria','misma_posicion',false,'Ejemplo','per_actoractoractoractoractor',now()-interval '10 days',NULL,now()-interval '10 days');
SET LOCAL session_replication_role = origin;


DO $doble$
DECLARE n text;
BEGIN
 FOREACH n IN ARRAY ARRAY['registrar_y_consumir_politica_ofertas_bolsa_v3_atestada','registrar_y_consumir_emision_llamamiento_v3_atestada'] LOOP
  EXECUTE format($s$CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.%I(p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
   RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
   LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $c$
   SELECT 'decision:r2op:'||gen_random_uuid(),convert_from(p_capacidad,'UTF8')::jsonb->>'efecto_ref',
          repeat('a',64),repeat('b',64),'auditoria:r2op:'||gen_random_uuid(),clock_timestamp(),true
   $c$ $s$, n);
 END LOOP;
END $doble$;

CREATE SCHEMA prueba_r2op;
CREATE FUNCTION prueba_r2op.espera(p_sql text,p_estado text) RETURNS void LANGUAGE plpgsql AS $f$
BEGIN
 EXECUTE p_sql;
 RAISE EXCEPTION 'B70: faltó el rechazo %',p_estado;
EXCEPTION WHEN OTHERS THEN
 IF SQLSTATE<>p_estado THEN RAISE EXCEPTION 'B70: esperado %, actual %: %',p_estado,SQLSTATE,SQLERRM; END IF;
END $f$;

CREATE FUNCTION prueba_r2op.politica(p_version bigint,p_politica jsonb,p_clave text) RETURNS jsonb
LANGUAGE sql SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT r.politica||jsonb_build_object('reutilizada',r.reutilizada)
 FROM vec_bolsa_llamamientos.publicar_politica_ofertas_v1('bolsa:r2op:1',p_version,p_politica,'per_actoractoractoractoractor',p_clave,
  'recibo:politica-ofertas:'||encode(sha256(convert_to(p_clave,'UTF8')),'hex'),
  convert_to('{"efecto_ref":"bolsa:r2op:1"}','UTF8'),
  convert_to('{"principal_id":"per_actoractoractoractoractor","accion":"bolsa.politica_ofertas.publicar","modulo_id":"bolsa","tipo_recurso":"bolsa_constituida","finalidad":"gobierno_politica_ofertas_bolsa","recurso_ref":"bolsa:r2op:1"}','UTF8'),
  '\x00','\x00',1,1,'\x00','\x00','\x00','\x00') r
$f$;
ALTER FUNCTION prueba_r2op.politica(bigint,jsonb,text) OWNER TO vec_bolsa_llamamientos_ejecutor;

CREATE FUNCTION prueba_r2op.acto(p_oferta text,p_plaza integer,p_tipo text,p_participacion text,p_secuencia integer,p_clave text) RETURNS jsonb
LANGUAGE sql SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT r.oferta||jsonb_build_object('reutilizada',r.reutilizada)
 FROM vec_bolsa_llamamientos.registrar_acto_plaza_oferta_v1(p_oferta,
  'recibo:plaza-oferta:'||encode(sha256(convert_to(p_oferta||chr(31)||p_clave,'UTF8')),'hex'),'bolsa:r2op:1',
  p_plaza,p_tipo,p_participacion,p_secuencia,'per_actoractoractoractoractor',p_clave,
  convert_to('{"efecto_ref":"bolsa:r2op:1"}','UTF8'),
  convert_to('{"principal_id":"per_actoractoractoractoractor","accion":"llamamiento.emitir.v1","modulo_id":"bolsa","tipo_recurso":"bolsa_constituida","finalidad":"gestion_llamamientos_bolsa","recurso_ref":"bolsa:r2op:1"}','UTF8'),
  '\x00','\x00',1,1,'\x00','\x00','\x00','\x00') r
$f$;
ALTER FUNCTION prueba_r2op.acto(text,integer,text,text,integer,text) OWNER TO vec_bolsa_llamamientos_ejecutor;
GRANT USAGE ON SCHEMA prueba_r2op TO vec_bolsa_llamamientos_ejecutor;

-- Preexistencia histórica, con huella y clave originales: no se publica otra
-- política antigua para preparar el replay que se desea comprobar.
INSERT INTO vec_bolsa_llamamientos.politica_ofertas_version
 (bolsa_ref,version,politica,huella_sha256,ejemplo,actor_ref,clave_idempotencia,version_esperada,recibo_ref,publicada_en,decision_ref,auditoria_ref)
SELECT 'bolsa:r2op:1',1,p,encode(sha256(convert_to(p::text,'UTF8')),'hex'),true,'per_actoractoractoractoractor','r2op-legacy-policy',0,
 'recibo:politica-ofertas:'||encode(sha256('r2op-legacy-policy'::bytea),'hex'),now()-interval '3 days','decision:r2op:historica','auditoria:r2op:historica'
FROM (SELECT '{"plazo":{"unidad":"dias_habiles","cantidad":2,"computo":"administrativo","municipio_sede":"18087"},"adjudicacion":{"criterio":"orden_vigente","elegibilidad":"disposicion_en_plazo"},"no_cubierta":{"accion":"llamamiento_directo","condicion":"sin_disposiciones_elegibles"},"plazas":{"llamada":"simultanea","respuesta_horas":24,"tras_renuncia":"siguiente_en_orden"}}'::jsonb p) s;

DO $politica$
DECLARE antigua jsonb; sin_modo jsonb; nueva jsonb; j jsonb; recibo text; total bigint;
BEGIN
 SELECT politica,recibo_ref INTO antigua,recibo FROM vec_bolsa_llamamientos.politica_ofertas_version WHERE bolsa_ref='bolsa:r2op:1';
 j:=prueba_r2op.politica(0,antigua,'r2op-legacy-policy');
 IF (j->>'reutilizada')::boolean IS NOT TRUE OR j->>'recibo_ref' IS DISTINCT FROM recibo OR j->'politica' IS DISTINCT FROM antigua
 THEN RAISE EXCEPTION 'B70: replay legacy cambia contenido o recibo'; END IF;
 sin_modo:=jsonb_set(antigua,'{plazo,inicio}','"notificacion"');
 PERFORM prueba_r2op.espera(format($$SELECT prueba_r2op.politica(1,%L,'r2op-sin-modo')$$,sin_modo),'22023');
 PERFORM prueba_r2op.espera(format($$SELECT prueba_r2op.politica(1,%L,'r2op-sin-inicio')$$,
  jsonb_set(antigua,'{adjudicacion,confirmacion}','"aceptacion_previa"')),'22023');
 PERFORM prueba_r2op.espera(format($$SELECT prueba_r2op.politica(1,%L,'r2op-modo-vacio')$$,
  jsonb_set(sin_modo,'{adjudicacion,confirmacion}','""')),'22023');
 PERFORM prueba_r2op.espera(format($$SELECT prueba_r2op.politica(1,%L,'r2op-modo-ajeno')$$,
  jsonb_set(sin_modo,'{adjudicacion,confirmacion}','"segunda_respuesta"')),'22023');
 IF (SELECT count(*) FROM vec_bolsa_llamamientos.politica_ofertas_version WHERE bolsa_ref='bolsa:r2op:1')<>1
 THEN RAISE EXCEPTION 'B70: rechazo dejó una versión'; END IF;
 nueva:=jsonb_set(jsonb_set(antigua,'{plazo,inicio}','"notificacion"'),'{adjudicacion,confirmacion}','"aceptacion_previa"');
 j:=prueba_r2op.politica(1,nueva,'r2op-policy-nueva');
 IF j->>'version' IS DISTINCT FROM '2' OR (j->>'reutilizada')::boolean IS NOT FALSE
    OR j#>>'{politica,adjudicacion,confirmacion}' IS DISTINCT FROM 'aceptacion_previa'
 THEN RAISE EXCEPTION 'B70: no publica versión telemática'; END IF;
 recibo:=j->>'recibo_ref';
 j:=prueba_r2op.politica(1,nueva,'r2op-policy-nueva');
 IF (j->>'reutilizada')::boolean IS NOT TRUE OR j->>'recibo_ref' IS DISTINCT FROM recibo
 THEN RAISE EXCEPTION 'B70: replay nuevo cambia recibo'; END IF;
 IF (SELECT count(*) FROM vec_bolsa_llamamientos.politica_ofertas_version WHERE bolsa_ref='bolsa:r2op:1')<>2
    OR (SELECT count(*) FROM vec_bolsa_llamamientos.politica_ofertas_outbox WHERE bolsa_ref='bolsa:r2op:1')<>1
 THEN RAISE EXCEPTION 'B70: política/outbox duplicados'; END IF;
 RAISE NOTICE 'B70 POLITICA-OK: legacy exacto, omisión/modo vacío/ajeno rechazados, versión nueva y replay único';
END $politica$;

-- Siembra administrativa de ofertas previas: no acredita una publicación.
-- Se omite el trigger B71 sólo para preparar el pasado sintético; vuelve a
-- origin antes de llamar a cualquiera de las operaciones que se comprueban.
SET LOCAL session_replication_role=replica;

-- Ofertas vencidas para ejercer adjudicación. No se altera el reloj: el
-- vencimiento histórico y las aceptaciones previas son datos del fixture.
INSERT INTO vec_bolsa_llamamientos.oferta_publicada
 (oferta_ref,recibo_ref,bolsa_ref,actor_ref,clave_idempotencia,datos,plazo,publicada_en,vence_antes_de,huella_comando_sha256,decision_ref)
SELECT 'oferta:'||encode(sha256(convert_to(n,'UTF8')),'hex'),'recibo:oferta:'||encode(sha256(convert_to(n,'UTF8')),'hex'),
 'bolsa:r2op:1','per_actoractoractoractoractor',n,
 '{"categoria":"Auxiliar administrativo","centro":"Residencia Sierra","fecha_inicio":"2026-10-01","descripcion":"Sustitución por baja"}'::jsonb,
 '{"politica_version":2}'::jsonb,now()-interval '3 days',now()-interval '1 day',repeat('d',64),'decision:'||n
FROM unnest(ARRAY['r2op-adjudicar','r2op-multiple','r2op-sin-aceptar']) n;
INSERT INTO vec_bolsa_llamamientos.plazas_oferta
SELECT oferta_ref,bolsa_ref,CASE WHEN clave_idempotencia='r2op-multiple' THEN 2 ELSE 1 END,2,publicada_en
FROM vec_bolsa_llamamientos.oferta_publicada WHERE bolsa_ref='bolsa:r2op:1';
INSERT INTO vec_bolsa_llamamientos.disposicion_oferta
SELECT o.oferta_ref,p,now()-interval '2 days','r2op-acepta-'||p,'recibo:disposicion:'||encode(sha256(convert_to(o.oferta_ref||p,'UTF8')),'hex')
FROM vec_bolsa_llamamientos.oferta_publicada o CROSS JOIN unnest(ARRAY['part:r2op:1','part:r2op:2','part:r2op:3']) p
WHERE o.clave_idempotencia IN ('r2op-adjudicar','r2op-multiple');
INSERT INTO vec_bolsa_llamamientos.situacion_participacion
 (participacion_ref,situacion,desde,hasta,fecha_disponible,motivo,actor,registrada_en,clave_idempotencia,recibo_ref)
VALUES ('part:r2op:1','no_disponible',now()-interval '1 hour',NULL,NULL,'pausa','per_actoractoractoractoractor',now()-interval '1 hour','r2op-pausa','recibo:r2op-pausa');

SET LOCAL session_replication_role=origin;

DO $adjudicacion$
DECLARE ofr text; mul text; nad text; j jsonb; p jsonb; recibo text; situaciones bigint;
BEGIN
 SELECT oferta_ref INTO ofr FROM vec_bolsa_llamamientos.oferta_publicada WHERE clave_idempotencia='r2op-adjudicar';
 SELECT oferta_ref INTO mul FROM vec_bolsa_llamamientos.oferta_publicada WHERE clave_idempotencia='r2op-multiple';
 SELECT oferta_ref INTO nad FROM vec_bolsa_llamamientos.oferta_publicada WHERE clave_idempotencia='r2op-sin-aceptar';
 SELECT count(*) INTO situaciones FROM vec_bolsa_llamamientos.situacion_participacion;
 -- La persona de acta1 está pausada: propone al siguiente disponible que aceptó.
 p:=vec_bolsa_llamamientos.proyectar_oferta_v2(ofr,clock_timestamp());
 IF p#>>'{plazas,0,propuesta,participacion_ref}' IS DISTINCT FROM 'part:r2op:2'
 THEN RAISE EXCEPTION 'B70: propuesta ignora disponibilidad'; END IF;
 PERFORM prueba_r2op.espera(format($$SELECT prueba_r2op.acto(%L,1,'adjudicada','part:r2op:1',0,'r2op-persona-pausada')$$,ofr),'VBO04');
 j:=prueba_r2op.acto(ofr,1,'adjudicada','part:r2op:2',0,'r2op-adjudicar-1');
 IF j->>'estado' IS DISTINCT FROM 'adjudicada' OR j#>>'{plazas,0,estado}' IS DISTINCT FROM 'cubierta'
    OR j#>'{plazas,0,responder_antes_de}' IS DISTINCT FROM 'null'::jsonb
 THEN RAISE EXCEPTION 'B70: adjudicación dejó respuesta adicional'; END IF;
 recibo:=j#>>'{plazas,0,historial,0,recibo_ref}';
 IF recibo IS NULL THEN RAISE EXCEPTION 'B70: adjudicación sin recibo'; END IF;
 j:=prueba_r2op.acto(ofr,1,'adjudicada','part:r2op:2',0,'r2op-adjudicar-1');
 IF (j->>'reutilizada')::boolean IS NOT TRUE OR j#>>'{plazas,0,historial,0,recibo_ref}' IS DISTINCT FROM recibo
 THEN RAISE EXCEPTION 'B70: replay adjudicación divergente'; END IF;
 PERFORM prueba_r2op.espera(format($$SELECT prueba_r2op.acto(%L,1,'aceptada','part:r2op:2',1,'r2op-segunda-aceptacion')$$,ofr),'VBO04');
 PERFORM prueba_r2op.espera(format($$SELECT prueba_r2op.acto(%L,1,'renuncia','part:r2op:2',1,'r2op-renuncia-telematica')$$,ofr),'VBO04');
 PERFORM prueba_r2op.espera(format($$SELECT prueba_r2op.acto(%L,1,'sin_respuesta','part:r2op:2',1,'r2op-silencio-telematica')$$,ofr),'VBO04');
 IF (SELECT count(*) FROM vec_bolsa_llamamientos.acto_plaza_oferta WHERE oferta_ref=ofr)<>1
    OR (SELECT count(*) FROM vec_bolsa_llamamientos.acto_plaza_oferta_outbox WHERE oferta_ref=ofr)<>1
 THEN RAISE EXCEPTION 'B70: replay/negativos dejaron actos u outbox'; END IF;
 IF (SELECT situacion FROM vec_bolsa_llamamientos.situacion_participacion WHERE participacion_ref='part:r2op:3' ORDER BY desde DESC LIMIT 1) IS DISTINCT FROM 'disponible'
    OR (SELECT situacion FROM vec_bolsa_llamamientos.situacion_participacion WHERE participacion_ref='part:r2op:4' ORDER BY desde DESC LIMIT 1) IS DISTINCT FROM 'disponible'
    OR situaciones<>(SELECT count(*) FROM vec_bolsa_llamamientos.situacion_participacion)
 THEN RAISE EXCEPTION 'B70: resto/silencio pierden disponibilidad'; END IF;
 -- Dos plazas: posiciones disponibles distintas, ambas cubiertas directamente.
 j:=prueba_r2op.acto(mul,1,'adjudicada','part:r2op:2',0,'r2op-multiple-1');
 j:=prueba_r2op.acto(mul,2,'adjudicada','part:r2op:3',0,'r2op-multiple-2');
 IF j->>'estado' IS DISTINCT FROM 'adjudicada' OR j#>>'{plazas,0,participacion_ref}'=j#>>'{plazas,1,participacion_ref}'
 THEN RAISE EXCEPTION 'B70: múltiples plazas no quedan cubiertas por personas distintas'; END IF;
 -- Sin aceptaciones, RRHH confirma el paso a directo, con recibo único.
 j:=prueba_r2op.acto(nad,1,'llamamiento_directo',NULL,0,'r2op-directo-1');
 IF j->>'estado' IS DISTINCT FROM 'llamamiento_directo' OR j#>>'{plazas,0,historial,0,recibo_ref}' IS NULL
 THEN RAISE EXCEPTION 'B70: directo no confirmado'; END IF;
 RAISE NOTICE 'B70 OPERACIONES-OK: disponibilidad, adjudicación definitiva, múltiples plazas, replay/recibo/outbox únicos, segunda respuesta rechazada y directo sin aceptaciones';
END $adjudicacion$;
ROLLBACK;
