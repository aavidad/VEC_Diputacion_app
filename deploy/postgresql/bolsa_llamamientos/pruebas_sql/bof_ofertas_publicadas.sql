-- Focal B-OF (migración 000028) para PostgreSQL 18 efímero con la estructura
-- real de Bolsa. Se ejecuta como superusuario y termina en ROLLBACK: siembra
-- una bolsa con session_replication_role=replica (sin su cadena de acta) y
-- sustituye, solo dentro de esta transacción, el consumidor de autorización
-- de la emisión por un doble que devuelve un consumo nuevo. La criptografía
-- atestada real no se prueba aquí; sí el contrato, la idempotencia, la
-- propuesta por orden vigente, las guardas y la ACL.
\set ON_ERROR_STOP on
BEGIN;
SET LOCAL timezone = 'UTC';

CREATE ROLE vec_bof_runtime_prueba NOLOGIN;
GRANT vec_bolsa_llamamientos_ejecutor TO vec_bof_runtime_prueba;

CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_emision_llamamiento_v3_atestada(p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql VOLATILE AS $d$
 SELECT 'decision:'||gen_random_uuid(), convert_from(p_capacidad,'UTF8')::jsonb->>'efecto_ref', repeat('a',64), repeat('b',64), 'auditoria:x', clock_timestamp(), true
$d$;

SET LOCAL session_replication_role = replica;
INSERT INTO vec_bolsa_llamamientos.bolsa_constituida VALUES
 ('bolsa:of:1',1,encode(sha256('{}'::bytea),'hex'),'{}'::bytea,'categoria:rpt:aux',now()-interval '10 days',NULL,'vigente',now()-interval '10 days'),
 ('bolsa:of:x',1,encode(sha256('{}'::bytea),'hex'),'{}'::bytea,'categoria:rpt:aux',now()-interval '10 days',NULL,'extinguida',now()-interval '10 days');
INSERT INTO vec_bolsa_llamamientos.constitucion VALUES
 ('acta:of:1','bolsa:of:1',1,encode(sha256('{}'::bytea),'hex'),'inst:of:1',1,repeat('c',64),'categoria:rpt:aux','per_actoractoractoractoractor',now()-interval '10 days',now()-interval '10 days'),
 ('acta:of:x','bolsa:of:x',1,encode(sha256('{}'::bytea),'hex'),'inst:of:x',1,repeat('c',64),'categoria:rpt:aux','per_actoractoractoractoractor',now()-interval '10 days',now()-interval '10 days');
INSERT INTO vec_bolsa_llamamientos.constitucion_entrada VALUES
 ('inst:of:1',1,1,'part:of:1',1),('inst:of:1',1,2,'part:of:2',2),('inst:of:1',1,3,'part:of:3',3),('inst:of:1',1,4,'part:of:4',4);
INSERT INTO vec_bolsa_llamamientos.situacion_participacion(participacion_ref,situacion,desde,hasta,fecha_disponible,motivo,actor,registrada_en,clave_idempotencia,recibo_ref)
SELECT p, CASE WHEN p='part:of:1' THEN 'no_disponible' ELSE 'disponible' END, now()-interval '9 days', NULL, NULL, 'alta', 'per_actoractoractoractoractor', now()-interval '9 days', 'clave:'||p, 'recibo:'||p
  FROM unnest(ARRAY['part:of:1','part:of:2','part:of:3','part:of:4']) p;
INSERT INTO vec_bolsa_llamamientos.politica_orden_bolsa(politica_ref,bolsa_ref,version,criterio,tipo_lista,reposicion,provisional,rotulo,actor,vigente_desde,vigente_hasta,registrada_en)
VALUES ('politica:of:1','bolsa:of:1',1,'puntuacion_desc_acta','rotatoria','misma_posicion',false,'Ejemplo','per_actoractoractoractoractor',now()-interval '10 days',NULL,now()-interval '10 days');
SET LOCAL session_replication_role = origin;

DO $prueba$
DECLARE
 cap bytea := convert_to('{"efecto_ref":"bolsa:of:1"}','UTF8');
 dec bytea := convert_to('{"principal_id":"per_actoractoractoractoractor","accion":"llamamiento.emitir.v1","tipo_recurso":"bolsa_constituida"}','UTF8');
 datos jsonb := '{"categoria":"Auxiliar administrativo","centro":"Residencia Sierra","fecha_inicio":"2026-10-01","fecha_fin":"2026-12-31","descripcion":"Sustitución por baja"}';
 plazo jsonb := '{"regla_ref":"vec.bolsa.reglas:1:b10.plazo_publicacion","huella_catalogo":"'||repeat('d',64)||'","unidad":"dias_habiles","cantidad":2,"computo":"administrativo","ultimo_dia":"2026-10-02","ejemplo":false}';
 r record; r2 record; estado text; codigo text;
 o1 text := 'oferta:'||repeat('1',64); o2 text := 'oferta:'||repeat('2',64); o3 text := 'oferta:'||repeat('3',64);
BEGIN
 SET LOCAL ROLE vec_bof_runtime_prueba;

 -- La cuenta de aplicación no lee tablas ni la proyección interna.
 BEGIN PERFORM 1 FROM vec_bolsa_llamamientos.oferta_publicada; RAISE EXCEPTION 'lectura directa permitida';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 BEGIN PERFORM vec_bolsa_llamamientos.proyectar_oferta_v1(o1, now()); RAISE EXCEPTION 'proyección interna ejecutable';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;

 -- Publicación válida con plazo de un segundo para probar el vencimiento.
 SELECT * INTO r FROM vec_bolsa_llamamientos.publicar_oferta_v1(o1,'recibo:oferta:'||repeat('1',64),'bolsa:of:1','per_actoractoractoractoractor','clave-oferta-1',datos,plazo,clock_timestamp(),clock_timestamp()+interval '1 second',cap,dec,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
 IF r.reutilizada OR r.oferta->>'estado'<>'abierta' OR r.oferta->>'oferta_ref'<>o1 OR r.oferta->'propuesta'<>'null'::jsonb THEN RAISE EXCEPTION 'publicación inicial incorrecta: %', r.oferta; END IF;

 -- Reintento exacto: misma oferta; otro contenido con la misma clave: conflicto.
 SELECT * INTO r2 FROM vec_bolsa_llamamientos.publicar_oferta_v1(o1,'recibo:oferta:'||repeat('1',64),'bolsa:of:1','per_actoractoractoractoractor','clave-oferta-1',datos,plazo,clock_timestamp(),clock_timestamp()+interval '1 day',cap,dec,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
 IF NOT r2.reutilizada OR r2.oferta->>'oferta_ref'<>o1 THEN RAISE EXCEPTION 'reintento no idempotente'; END IF;
 BEGIN
  PERFORM vec_bolsa_llamamientos.publicar_oferta_v1(o2,'recibo:oferta:'||repeat('2',64),'bolsa:of:1','per_actoractoractoractoractor','clave-oferta-1',datos||'{"centro":"Otro centro"}',plazo,clock_timestamp(),clock_timestamp()+interval '1 day',cap,dec,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
  RAISE EXCEPTION 'clave divergente aceptada';
 EXCEPTION WHEN SQLSTATE 'VBO01' THEN NULL; END;

 -- Entradas inválidas, autorización ajena y bolsa no vigente.
 FOREACH codigo IN ARRAY ARRAY['clave','fin','plazo'] LOOP
  BEGIN
   PERFORM vec_bolsa_llamamientos.publicar_oferta_v1(o2,'recibo:oferta:'||repeat('2',64),'bolsa:of:1','per_actoractoractoractoractor','clave-oferta-2',
     CASE codigo WHEN 'clave' THEN datos||'{"dni":"12345678Z"}' WHEN 'fin' THEN datos||'{"fecha_fin":"2026-09-01"}' ELSE datos END,
     plazo,clock_timestamp(),clock_timestamp()+CASE codigo WHEN 'plazo' THEN interval '200 days' ELSE interval '1 day' END,cap,dec,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
   RAISE EXCEPTION 'entrada inválida aceptada: %', codigo;
  EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 END LOOP;
 BEGIN
  PERFORM vec_bolsa_llamamientos.publicar_oferta_v1(o2,'recibo:oferta:'||repeat('2',64),'bolsa:of:1','per_actoractoractoractoractor','clave-oferta-2',datos,plazo,clock_timestamp(),clock_timestamp()+interval '1 day',cap,convert_to('{"principal_id":"per_actoractoractoractoractor","accion":"situacion.cambiar","tipo_recurso":"bolsa_constituida"}','UTF8'),'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
  RAISE EXCEPTION 'acción ajena aceptada';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 BEGIN
  PERFORM vec_bolsa_llamamientos.publicar_oferta_v1(o2,'recibo:oferta:'||repeat('2',64),'bolsa:of:x','per_actoractoractoractoractor','clave-oferta-2',datos,plazo,clock_timestamp(),clock_timestamp()+interval '1 day',convert_to('{"efecto_ref":"bolsa:of:x"}','UTF8'),dec,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
  RAISE EXCEPTION 'bolsa extinguida aceptada';
 EXCEPTION WHEN foreign_key_violation THEN NULL; END;

 -- Segunda oferta, sin disposiciones.
 PERFORM vec_bolsa_llamamientos.publicar_oferta_v1(o3,'recibo:oferta:'||repeat('3',64),'bolsa:of:1','per_actoractoractoractoractor','clave-oferta-3',datos,plazo,clock_timestamp(),clock_timestamp()+interval '1 second',cap,dec,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');

 -- Antes de vencer no se resuelve.
 BEGIN
  PERFORM vec_bolsa_llamamientos.resolver_oferta_v1(o1,'recibo:resolucion-oferta:'||repeat('1',64),'bolsa:of:1',NULL,'per_actoractoractoractoractor','clave-resolucion-1',cap,dec,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
  RAISE EXCEPTION 'resolución con plazo abierto';
 EXCEPTION WHEN SQLSTATE 'VBO03' THEN NULL; END;
 RESET ROLE;

 -- Disposiciones sembradas por el propietario (su escritura por el candidato
 -- llegará con su consumidor): la 1 está en pausa, la 3 y la 4 disponibles.
 INSERT INTO vec_bolsa_llamamientos.disposicion_oferta VALUES
  (o1,'part:of:1',clock_timestamp(),'clave-disp-1','recibo:disposicion:'||repeat('a',64)),
  (o1,'part:of:4',clock_timestamp(),'clave-disp-4','recibo:disposicion:'||repeat('b',64)),
  (o1,'part:of:3',clock_timestamp(),'clave-disp-3','recibo:disposicion:'||repeat('c',64));
 PERFORM pg_sleep(1.2);

 SET LOCAL ROLE vec_bof_runtime_prueba;
 SELECT x INTO r FROM jsonb_array_elements(vec_bolsa_llamamientos.listar_ofertas_bolsa_v1('bolsa:of:1',clock_timestamp(),10)) x WHERE x->>'oferta_ref'=o1;
 IF r.x->>'estado'<>'pendiente_resolucion' OR r.x->'propuesta'->>'tipo'<>'adjudicar' OR r.x->'propuesta'->>'participacion_ref'<>'part:of:3'
    OR (r.x->>'disposiciones_total')::int<>3 OR r.x->'disposiciones'->0->>'participacion_ref'<>'part:of:3' THEN
  RAISE EXCEPTION 'propuesta por orden vigente incorrecta: %', r.x;
 END IF;
 BEGIN
  PERFORM vec_bolsa_llamamientos.resolver_oferta_v1(o1,'recibo:resolucion-oferta:'||repeat('1',64),'bolsa:of:1','part:of:4','per_actoractoractoractoractor','clave-resolucion-1',cap,dec,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
  RAISE EXCEPTION 'resolución distinta de la propuesta aceptada';
 EXCEPTION WHEN SQLSTATE 'VBO04' THEN NULL; END;
 SELECT * INTO r FROM vec_bolsa_llamamientos.resolver_oferta_v1(o1,'recibo:resolucion-oferta:'||repeat('1',64),'bolsa:of:1','part:of:3','per_actoractoractoractoractor','clave-resolucion-1',cap,dec,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
 IF r.reutilizada OR r.oferta->>'estado'<>'adjudicada' OR r.oferta->'resolucion'->>'participacion_ref'<>'part:of:3' OR (r.oferta->'resolucion'->>'orden_vigente')::int<>2 THEN
  RAISE EXCEPTION 'adjudicación incorrecta: %', r.oferta;
 END IF;
 SELECT * INTO r FROM vec_bolsa_llamamientos.resolver_oferta_v1(o1,'recibo:resolucion-oferta:'||repeat('1',64),'bolsa:of:1','part:of:3','per_actoractoractoractoractor','clave-resolucion-1',cap,dec,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
 IF NOT r.reutilizada THEN RAISE EXCEPTION 'reintento de resolución no idempotente'; END IF;
 BEGIN
  PERFORM vec_bolsa_llamamientos.resolver_oferta_v1(o1,'recibo:resolucion-oferta:'||repeat('9',64),'bolsa:of:1','part:of:3','per_actoractoractoractoractor','clave-resolucion-otra',cap,dec,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
  RAISE EXCEPTION 'segunda resolución aceptada';
 EXCEPTION WHEN SQLSTATE 'VBO02' THEN NULL; END;

 -- Sin disposiciones: VEC propone llamamiento directo.
 SELECT * INTO r FROM vec_bolsa_llamamientos.resolver_oferta_v1(o3,'recibo:resolucion-oferta:'||repeat('3',64),'bolsa:of:1',NULL,'per_actoractoractoractoractor','clave-resolucion-3',cap,dec,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
 IF r.oferta->>'estado'<>'llamamiento_directo' OR r.oferta->'resolucion'->>'participacion_ref' IS NOT NULL THEN
  RAISE EXCEPTION 'llamamiento directo incorrecto: %', r.oferta;
 END IF;
 RESET ROLE;

 -- Historia de solo adición, también para el propietario.
 SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
 BEGIN UPDATE vec_bolsa_llamamientos.oferta_publicada SET datos='{}' WHERE oferta_ref=o1; RAISE EXCEPTION 'oferta mutable';
 EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL; END;
 BEGIN DELETE FROM vec_bolsa_llamamientos.resolucion_oferta WHERE oferta_ref=o1; RAISE EXCEPTION 'resolución borrable';
 EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL; END;
 RESET ROLE;
END $prueba$;

SELECT 'OK B-OF focal' AS resultado;
ROLLBACK;
