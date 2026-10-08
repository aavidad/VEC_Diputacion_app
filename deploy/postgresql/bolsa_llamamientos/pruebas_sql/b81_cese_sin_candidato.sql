\set ON_ERROR_STOP on
-- Ejecutar en un clon PostgreSQL 18 de la principal con CT197, B81 y B90.
-- La fixture usa la incorporación sintética conservada, CT129 real y B13 real.
-- Solo la preparación de historia de prueba omite triggers de transición;
-- la operación B81 y el verificador CT se ejecutan con todos activos.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL statement_timeout = '30s';
SET LOCAL session_replication_role = replica;

CREATE ROLE vec_b81_774_relevo LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_b81_774_intruso LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_b81_774_ejecutor LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
GRANT vec_bolsa_llamamientos_relevo_cese TO vec_b81_774_relevo;
GRANT vec_bolsa_llamamientos_ejecutor TO vec_b81_774_ejecutor;

DO $fixture$
DECLARE
 i vec_contratacion_temporal.incorporacion_registro_v2;
 v vec_contratacion_temporal.expediente_version_integral;
 c vec_contratacion_temporal.cese_nombramiento_v1;
 j jsonb;
 publicacion jsonb;
 outbox jsonb;
 prueba bytea;
 payload bytea;
 anterior text := repeat('0',64);
 inicio timestamptz;
 instante timestamptz := date_trunc('microseconds',clock_timestamp());
 origen text := 'evento:ct:b81:774:cese';
 evento text := 'evento:ct:contrato-bolsa:' || encode(sha256(convert_to('cese'||chr(31)||'evento:ct:b81:774:cese','UTF8')),'hex');
 bolsa text := 'bolsa-sintetica:abcdefghijklmnop';
 participacion text := 'participacion-sintetica-1:abcdefghijklmnop';
 propuesta bytea;
BEGIN
 SELECT * INTO STRICT i FROM vec_contratacion_temporal.incorporacion_registro_v2
  WHERE organizacion_ref ~ '^organizacion:desarrollo:' ORDER BY recibo_ref LIMIT 1;
 SELECT * INTO STRICT v FROM vec_contratacion_temporal.expediente_version_integral
  WHERE expediente_ref=i.expediente_ref ORDER BY version DESC LIMIT 1;
 IF v.agregado_json #>> '{analisis,modalidad_clave}' IS NULL
    OR v.agregado_json #>> '{analisis,categoria_ref}' IS NULL
    OR i.material_json #>> '{Confirmacion,ResultadoPersonal,relacion_ref}' !~ '^ref:[0-9a-f]{64}$' THEN
  RAISE EXCEPTION 'B81 fixture: incorporación postHX no satisface CT129';
 END IF;
 inicio := (i.material_json #>> '{Confirmacion,PeriodoIncorporacion,desde}')::timestamptz;
 j := v.agregado_json || jsonb_build_object('version',v.version+1);
 prueba := convert_to(rpad('b81-774-prueba-cese-'||v.version::text,160,'x'),'UTF8');
 INSERT INTO vec_contratacion_temporal.expediente_version_integral(
  expediente_ref,version,agregado_json,agregado_json_huella_sha256,prueba_canonica,prueba_huella_sha256,
  flujo_ref,flujo_version,flujo_huella_sha256,fase_clave,estado,origen_version,operacion_ref,registrada_en)
 VALUES(i.expediente_ref,v.version+1,j,encode(sha256(convert_to(j::text,'UTF8')),'hex'),
  prueba,encode(sha256(prueba),'hex'),v.flujo_ref,v.flujo_version,v.flujo_huella_sha256,
  v.fase_clave,'en_curso','cese_nombramiento_ct115','operacion:b81:774:cese',instante);

 INSERT INTO vec_contratacion_temporal.cese_nombramiento_v1(
  ambito_hmac,huella_peticion_hmac,organizacion_ref,expediente_ref,version_esperada,actor_ref,perfil_ref,
  causa_clave,fecha_efecto,justificante_tipo,justificante_ref,justificante_sha256,observaciones,
  incorporacion_ref,inicio_incorporacion,llamamiento_ref,estado,reserva_ref,recibo_ref,evento_ref,
  expediente_anterior_json,expediente_siguiente_json,recibo_json,decision_ref,decision_huella_sha256,
  consumo_huella_sha256,auditoria_ref,politica_ref,politica_version,politica_huella_sha256,
  registrada_en,confirmada_en)
 VALUES('hmac-sha256:vec.contratacion-temporal.cese.ambito/v1:'||repeat('a',64),
  'hmac-sha256:vec.contratacion-temporal.cese.peticion/v1:'||repeat('b',64),
  i.organizacion_ref,i.expediente_ref,v.version,'per_b81_774','prf_b81_774','fin_sustitucion',
  '2030-01-01','comunicacion_reincorporacion','documento:b81:774',repeat('c',64),'',
  i.recibo_ref,(inicio AT TIME ZONE 'UTC')::date,'llamamiento:b81:774','confirmada',
  'reserva:b81:774','recibo:ct:b81:774',origen,v.agregado_json,j,
  jsonb_build_object('registrada_en',to_char(instante AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
                     'auditoria_ref','aud_v3_'||repeat('f',32)),
  'decision:b81:774',repeat('d',64),repeat('e',64),'aud_v3_'||repeat('f',32),
  'politica:b81:774',1,repeat('a',64),instante,instante);
 SELECT * INTO STRICT c FROM vec_contratacion_temporal.cese_nombramiento_v1 WHERE evento_ref=origen;

 outbox := jsonb_build_object('esquema','vec.contratacion-temporal.cese.v1',
   'organizacion_ref',c.organizacion_ref,'expediente_ref',c.expediente_ref,
   'version_resultante',c.version_esperada+1,'incorporacion_ref',c.incorporacion_ref,
   'llamamiento_ref',c.llamamiento_ref,'causa_clave',c.causa_clave,
   'fecha_efecto',to_char(c.fecha_efecto,'YYYY-MM-DD'),'recibo_ref',c.recibo_ref,
   'registrada_en',c.recibo_json->'registrada_en');
 payload := convert_to(outbox::text,'UTF8');
 INSERT INTO vec_contratacion_temporal.outbox_expediente_integral(
  evento_ref,secuencia,operacion_ref,expediente_ref,version_expediente,tipo_evento,
  payload_canonico,payload_huella_sha256,anterior_sha256,huella_sha256,registrada_en)
 VALUES(origen,(SELECT max(secuencia)+1 FROM vec_contratacion_temporal.outbox_expediente_integral),
  'operacion:b81:774:cese',c.expediente_ref,c.version_esperada+1,'ct.cese.v1',
  payload,encode(sha256(payload),'hex'),anterior,
  encode(sha256(anterior::bytea||payload),'hex'),instante);

 publicacion := jsonb_build_object(
  'esquema','vec.contratacion-temporal.contrato-bolsa.v1','tipo','cese',
  'origen_ref',c.evento_ref,'organizacion_ref',c.organizacion_ref,
  'expediente_ref',c.expediente_ref,'llamamiento_ref',c.llamamiento_ref,
  'inicio',vec_contratacion_temporal.instante_contrato_bolsa_v1(inicio),
  'fin_previsto',vec_contratacion_temporal.instante_contrato_bolsa_v1(c.fecha_efecto::timestamp AT TIME ZONE 'UTC'),
  'modalidad_clave',c.expediente_siguiente_json #>> '{analisis,modalidad_clave}',
  'categoria_ref',c.expediente_siguiente_json #>> '{analisis,categoria_ref}',
  'causa_clave',c.causa_clave,
  'ocurrido_en',vec_contratacion_temporal.instante_contrato_bolsa_v1(c.registrada_en),
  'evento_ref',evento);
 INSERT INTO vec_bolsa_llamamientos.contrato_participacion(
  evento_ref,huella_sha256,evento,origen_ref,origen_creada_en,origen_posicion,tipo,
  organizacion_ref,expediente_ref,llamamiento_ref,participacion_ref,bolsa_ref,ocurrido_en,recibido_en)
 VALUES(evento,encode(sha256(convert_to(publicacion::text,'UTF8')),'hex'),publicacion,
  origen,instante,vec_contratacion_temporal.posicion_contrato_bolsa_v1(c.transaccion_publicacion),
  'cese',c.organizacion_ref,c.expediente_ref,c.llamamiento_ref,participacion,bolsa,instante,instante);

 INSERT INTO vec_bolsa_llamamientos.integracion_desarrollo(
  operacion_ref,tipo,necesidad_ref,version_necesidad,orden_operacion_ref,registro_canonico,
  registro_huella_sha256,contexto_huella_sha256,decision_ref,recibo_ref,confirmada_en)
 VALUES('operacion:b81:774:orden','orden','necesidad:b81:774',1,NULL,'{}'::bytea,
  encode(sha256('{}'::bytea),'hex'),repeat('1',64),'decision:b81:774:orden','recibo:b81:774:orden',instante);
 propuesta := convert_to(jsonb_build_object('propuesta',jsonb_build_object('participacion_seleccionada_ref',participacion))::text,'UTF8');
 INSERT INTO vec_bolsa_llamamientos.integracion_desarrollo(
  operacion_ref,tipo,necesidad_ref,version_necesidad,orden_operacion_ref,registro_canonico,
  registro_huella_sha256,contexto_huella_sha256,decision_ref,recibo_ref,confirmada_en)
 VALUES('operacion:b81:774:propuesta','propuesta','necesidad:b81:774',1,'operacion:b81:774:orden',
  propuesta,encode(sha256(propuesta),'hex'),repeat('2',64),'decision:b81:774:propuesta','recibo:b81:774:propuesta',instante);
 INSERT INTO vec_bolsa_llamamientos.llamamiento_integracion_desarrollo(
  llamamiento_ref,operacion_ref,propuesta_ref,bolsa_ref,necesidad_ref,version,estado,abierto_en,datos_canonicos)
 VALUES(c.llamamiento_ref,'operacion:b81:774:propuesta','propuesta:b81:774',bolsa,
  'necesidad:b81:774',1,'abierto',instante,'{}'::jsonb);

 PERFORM set_config('vec.b81.origen',origen,true);
 PERFORM set_config('vec.b81.evento',evento,true);
 PERFORM set_config('vec.b81.huella',encode(sha256(convert_to(publicacion::text,'UTF8')),'hex'),true);
 PERFORM set_config('vec.b81.posicion',vec_contratacion_temporal.posicion_contrato_bolsa_v1(c.transaccion_publicacion)::text,true);
 PERFORM set_config('vec.b81.bolsa',bolsa,true);
 PERFORM set_config('vec.b81.participacion',participacion,true);
END $fixture$;
SET LOCAL session_replication_role = origin;

-- Una bolsa constituida nunca entra en la rama sin candidato.
SAVEPOINT bolsa_constituida;
INSERT INTO vec_bolsa_llamamientos.bolsa_constituida(
 bolsa_ref,version,huella_bolsa_sha256,bolsa_canonica,categoria_ref,
 vigente_desde,vigente_hasta,estado,registrada_en)
VALUES(current_setting('vec.b81.bolsa'),1,encode(sha256('{}'::bytea),'hex'),'{}'::bytea,
 'categoria:b81:774',clock_timestamp(),NULL,'vigente',clock_timestamp());
SET SESSION AUTHORIZATION vec_b81_774_relevo;
DO $denegada$
DECLARE o text:=current_setting('vec.b81.origen'); h text:=current_setting('vec.b81.huella');
 p bigint:=current_setting('vec.b81.posicion')::bigint;
BEGIN
 BEGIN
  PERFORM vec_bolsa_llamamientos.confirmar_cese_sin_candidato_bolsa_v1(o,h,p);
  RAISE EXCEPTION 'B81: bolsa constituida aceptada';
 EXCEPTION WHEN foreign_key_violation THEN NULL; END;
END $denegada$;
RESET SESSION AUTHORIZATION;
ROLLBACK TO SAVEPOINT bolsa_constituida;

-- CT197 debe devolver el vínculo sin dejar la marca RLS abierta.
SELECT set_config('vec.ct129.origen_ref','marca-previa-b81',true);
SET SESSION AUTHORIZATION vec_b81_774_relevo;
DO $ejecucion$
DECLARE o text:=current_setting('vec.b81.origen'); h text:=current_setting('vec.b81.huella');
 p bigint:=current_setting('vec.b81.posicion')::bigint; v record;
BEGIN
 IF vec_bolsa_llamamientos.confirmar_cese_sin_candidato_bolsa_v1(o,repeat('0',64),p) IS NOT NULL THEN
  RAISE EXCEPTION 'B81: huella falsa aceptada';
 END IF;
EXCEPTION WHEN insufficient_privilege THEN
 IF vec_bolsa_llamamientos.confirmar_cese_sin_candidato_bolsa_v1(o,h,p) IS DISTINCT FROM false THEN
  RAISE EXCEPTION 'B81: alta no fue nueva';
 END IF;
 IF vec_bolsa_llamamientos.confirmar_cese_sin_candidato_bolsa_v1(o,h,p) IS DISTINCT FROM true THEN
  RAISE EXCEPTION 'B81: replay no reutilizó alta';
 END IF;
 SELECT * INTO STRICT v FROM vec_bolsa_llamamientos.cursor_restriccion_cese_bolsa_v1();
 IF v.origen_posicion<>p OR v.origen_ref<>o THEN
  RAISE EXCEPTION 'B81: cursor esperado %/% actual %/%',p,o,v.origen_posicion,v.origen_ref;
 END IF;
 IF current_setting('vec.ct129.origen_ref',true)<>'marca-previa-b81' THEN
  RAISE EXCEPTION 'CT197: marca RLS no restaurada tras éxito';
 END IF;
END $ejecucion$;
RESET SESSION AUTHORIZATION;

-- La fuente CT115 mantiene formato y obligatoriedad incluso en esta fixture.
SET LOCAL session_replication_role = replica;
DO $fuente$
DECLARE o text:=current_setting('vec.b81.origen');
BEGIN
 BEGIN
  UPDATE vec_contratacion_temporal.cese_nombramiento_v1 SET auditoria_ref='auditoria_invalida'
   WHERE evento_ref=o;
  RAISE EXCEPTION 'CT197: formato de auditoría aceptado';
 EXCEPTION WHEN check_violation THEN NULL; END;
 BEGIN
  UPDATE vec_contratacion_temporal.cese_nombramiento_v1 SET auditoria_ref=NULL WHERE evento_ref=o;
  RAISE EXCEPTION 'CT197: auditoría nula aceptada';
 EXCEPTION WHEN not_null_violation THEN NULL; END;
END $fuente$;
SET LOCAL session_replication_role = origin;

-- Una referencia local ya confirmada no se reutiliza si el recibo CT diverge.
SAVEPOINT auditoria_divergente;
SET LOCAL session_replication_role = replica;
UPDATE vec_contratacion_temporal.cese_nombramiento_v1
 SET recibo_json=jsonb_set(recibo_json,'{auditoria_ref}',to_jsonb('aud_v3_'||repeat('e',32)))
 WHERE evento_ref=current_setting('vec.b81.origen');
SET LOCAL session_replication_role = origin;
SET SESSION AUTHORIZATION vec_b81_774_relevo;
DO $divergente$
BEGIN
 BEGIN
  PERFORM vec_bolsa_llamamientos.confirmar_cese_sin_candidato_bolsa_v1(
   current_setting('vec.b81.origen'),current_setting('vec.b81.huella'),current_setting('vec.b81.posicion')::bigint);
  RAISE EXCEPTION 'CT197: recibo con auditoría divergente aceptado';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 IF current_setting('vec.ct129.origen_ref',true)<>'marca-previa-b81' THEN
  RAISE EXCEPTION 'CT197: marca RLS no restaurada tras error';
 END IF;
END $divergente$;
RESET SESSION AUTHORIZATION;
ROLLBACK TO SAVEPOINT auditoria_divergente;

-- Aunque la fuente presentase otro testigo coherente, el replay no reescribe
-- la referencia de auditoría que quedó en la proyección original.
SAVEPOINT auditoria_cambiada;
SET LOCAL session_replication_role = replica;
UPDATE vec_contratacion_temporal.cese_nombramiento_v1
 SET auditoria_ref='aud_v3_'||repeat('e',32),
     recibo_json=jsonb_set(recibo_json,'{auditoria_ref}',to_jsonb('aud_v3_'||repeat('e',32)))
 WHERE evento_ref=current_setting('vec.b81.origen');
SET LOCAL session_replication_role = origin;
SET SESSION AUTHORIZATION vec_b81_774_relevo;
DO $replay_divergente$
BEGIN
 BEGIN
  PERFORM vec_bolsa_llamamientos.confirmar_cese_sin_candidato_bolsa_v1(
   current_setting('vec.b81.origen'),current_setting('vec.b81.huella'),current_setting('vec.b81.posicion')::bigint);
  RAISE EXCEPTION 'B81: replay sustituyó la auditoría de origen';
 EXCEPTION WHEN SQLSTATE 'VBC01' THEN NULL; END;
END $replay_divergente$;
RESET SESSION AUTHORIZATION;
ROLLBACK TO SAVEPOINT auditoria_cambiada;

DO $auditoria$
DECLARE s vec_bolsa_llamamientos.cese_sin_candidato_bolsa;
BEGIN
 SELECT * INTO STRICT s FROM vec_bolsa_llamamientos.cese_sin_candidato_bolsa
  WHERE origen_ref=current_setting('vec.b81.origen');
 IF s.evento_ref<>current_setting('vec.b81.evento')
    OR s.origen_huella_sha256<>current_setting('vec.b81.huella')
    OR s.auditoria_ct_ref<>'aud_v3_'||repeat('f',32)
    OR s.registro_sha256<>encode(sha256(convert_to(s.registro::text,'UTF8')),'hex')
    OR s.registro->>'proceso'<>'vec_b81_774_relevo'
    OR s.registro->>'esquema'<>'vec.bolsa.cese.sin-candidato.proyeccion.v1'
    OR s.registro->>'origen_evento_ref'<>s.origen_ref
    OR s.registro->>'auditoria_ct_ref'<>s.auditoria_ct_ref
    OR s.registro->>'motivo'<>'participacion_no_constituida'
    OR s.registro->>'recibo_ct_ref'<>'recibo:ct:b81:774'
    OR (SELECT count(*) FROM vec_bolsa_llamamientos.cese_sin_candidato_bolsa
        WHERE origen_ref=s.origen_ref)<>1
    OR EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.restriccion_cese_bolsa WHERE origen_ref=s.origen_ref)
    OR EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.cese_ajeno_bolsa WHERE origen_ref=s.origen_ref)
    OR EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.situacion_participacion
              WHERE participacion_ref=current_setting('vec.b81.participacion')) THEN
  RAISE EXCEPTION 'B81: auditoría, unicidad o ausencia de efectos divergente';
 END IF;
 BEGIN
  UPDATE vec_bolsa_llamamientos.cese_sin_candidato_bolsa SET motivo='participacion_no_constituida'
   WHERE origen_ref=s.origen_ref;
  RAISE EXCEPTION 'B81: UPDATE aceptado en historia';
 EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL; END;
 BEGIN
  DELETE FROM vec_bolsa_llamamientos.cese_sin_candidato_bolsa WHERE origen_ref=s.origen_ref;
  RAISE EXCEPTION 'B81: DELETE aceptado en historia';
 EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL; END;
 IF has_table_privilege('vec_bolsa_llamamientos_relevo_cese',
    'vec_bolsa_llamamientos.cese_sin_candidato_bolsa','SELECT,INSERT,UPDATE,DELETE')
    OR has_function_privilege('vec_bolsa_llamamientos_ejecutor',
    'vec_bolsa_llamamientos.confirmar_cese_sin_candidato_bolsa_v1(text,text,bigint)','EXECUTE') THEN
  RAISE EXCEPTION 'B81: ACL de tabla o función demasiado amplia';
 END IF;
END $auditoria$;

-- Política exclusivamente de prueba, dentro de ROLLBACK. El postHX tiene
-- modalidad «sustitucion» y el catálogo de ejemplo B45 no la mapea. La
-- fixture reutiliza la clase existente de interinidad sin fijar una regla
-- funcional ni publicar ese mapeo fuera de esta transacción.
DO $politica_prueba$
DECLARE p vec_bolsa_llamamientos.politica_cese_bolsa;
 modalidad text; clase text; m jsonb;
BEGIN
 SELECT * INTO STRICT p FROM vec_bolsa_llamamientos.politica_cese_bolsa ORDER BY version DESC LIMIT 1;
 SELECT c.expediente_siguiente_json #>> '{analisis,modalidad_clave}' INTO STRICT modalidad
  FROM vec_contratacion_temporal.cese_nombramiento_v1 c
  WHERE c.evento_ref=current_setting('vec.b81.origen');
 IF p.mapeo ? modalidad THEN RETURN; END IF;
 clase:=p.mapeo->>'interinidad';
 IF clase NOT IN ('general','acumulacion_tareas') THEN
  RAISE EXCEPTION 'B81 fixture: falta clase de ejemplo para modalidad %',modalidad;
 END IF;
 m:=p.mapeo||jsonb_build_object(modalidad,clase);
 INSERT INTO vec_bolsa_llamamientos.politica_cese_bolsa(version,catalogo_ref,catalogo_sha256,mapeo,
  meses_general,meses_acumulacion,computo,estado,publicada_en)
 VALUES(p.version+1,'catalogo:bolsa:cese:ensayo-b81-774:v2',
  encode(sha256(convert_to(jsonb_build_object('mapeo',m,'meses_general',p.meses_general,
   'meses_acumulacion',p.meses_acumulacion,'computo',p.computo,'estado',p.estado)::text,'UTF8')),'hex'),
  m,p.meses_general,p.meses_acumulacion,p.computo,p.estado,clock_timestamp());
END $politica_prueba$;

-- B8 puede vincular después de B81. Antes de B45, B90 lo deja pendiente sin
-- inventar fecha; el relevo aplica después la restricción real de B45.
DO $vinculo_posterior$
DECLARE b text:=current_setting('vec.b81.bolsa'); p text:=current_setting('vec.b81.participacion');
 ahora timestamptz:=date_trunc('microseconds',clock_timestamp());
 bolsa_material bytea:=convert_to('{"bolsa":"b81-774"}','UTF8');
 instante_material bytea:=convert_to('{"participaciones":1}','UTF8');
 bolsa_sha text; instante_sha text;
BEGIN
 bolsa_sha:=encode(sha256(bolsa_material),'hex');
 instante_sha:=encode(sha256(instante_material),'hex');
 INSERT INTO vec_bolsa_llamamientos.bolsa_constituida(
  bolsa_ref,version,huella_bolsa_sha256,bolsa_canonica,categoria_ref,
  vigente_desde,vigente_hasta,estado,registrada_en)
 VALUES(b,1,bolsa_sha,bolsa_material,'categoria:b81:774',ahora,NULL,'vigente',ahora);
 INSERT INTO vec_bolsa_llamamientos.instantanea_orden_bolsa(
  instantanea_ref,version,huella_instantanea_sha256,instantanea_canonica,
  bolsa_ref,version_bolsa,huella_bolsa_sha256,total_participaciones,
  referida_en,generada_en,registrada_en)
 VALUES('instantanea:b81:774',1,instante_sha,instante_material,
  b,1,bolsa_sha,1,ahora,ahora,ahora);
 INSERT INTO vec_bolsa_llamamientos.constitucion(
  acta_ref,bolsa_ref,version_bolsa,huella_bolsa_sha256,instantanea_ref,
  version_instantanea,huella_instantanea_sha256,categoria_ref,actor_ref,
  confirmada_en,registrada_en)
 VALUES('acta:b81:774',b,1,bolsa_sha,'instantanea:b81:774',1,instante_sha,
  'categoria:b81:774','actor:b81:774',ahora,ahora);
 INSERT INTO vec_bolsa_llamamientos.constitucion_entrada(
  instantanea_ref,version_instantanea,orden,participacion_ref,fila_numero)
 VALUES('instantanea:b81:774',1,1,p,1);
 INSERT INTO vec_bolsa_llamamientos.vinculo_candidato(
  participacion_ref,candidato_ref,acta_ref,instantanea_ref,version_instantanea,registrada_en)
 VALUES(p,'can_'||repeat('a',22),'acta:b81:774','instantanea:b81:774',1,ahora);
END $vinculo_posterior$;

SET SESSION AUTHORIZATION vec_b81_774_ejecutor;
DO $estado_pendiente$
DECLARE v record;
BEGIN
 SELECT * INTO STRICT v FROM vec_bolsa_llamamientos.consultar_estado_cese_bolsa_v2(
  current_setting('vec.b81.participacion'),'2030-01-02T00:00:00Z'::timestamptz);
 IF v.cese_pendiente IS NOT TRUE OR v.fecha_efecto IS NOT NULL
    OR v.disponible_desde IS NOT NULL OR v.en_restriccion IS NOT FALSE
    OR v.trabajo_cesado IS NOT FALSE THEN
  RAISE EXCEPTION 'B90: candidato con cese pendiente aparece elegible o con fecha inventada';
 END IF;
END $estado_pendiente$;
RESET SESSION AUTHORIZATION;

SET SESSION AUTHORIZATION vec_b81_774_relevo;
DO $resolver_pendiente$
DECLARE o text:=current_setting('vec.b81.origen'); h text:=current_setting('vec.b81.huella');
 p bigint:=current_setting('vec.b81.posicion')::bigint; pagina jsonb; r record;
BEGIN
 IF vec_bolsa_llamamientos.confirmar_cese_sin_candidato_bolsa_v1(o,h,p) IS DISTINCT FROM true THEN
  RAISE EXCEPTION 'B81: replay tras constituir no reutilizó proyección';
 END IF;
 pagina:=vec_bolsa_llamamientos.listar_ceses_sin_candidato_pendientes_v1(100);
 IF jsonb_array_length(pagina)<>1 OR pagina->0->>'origen_ref'<>o
    OR pagina->0->>'huella_sha256'<>h
    OR (pagina->0->>'origen_posicion')::bigint<>p THEN
  RAISE EXCEPTION 'B81: cese vinculado ausente de pendientes';
 END IF;
 SELECT * INTO STRICT r FROM vec_bolsa_llamamientos.registrar_restriccion_cese_bolsa_v1(o,h,p);
 IF r.reutilizada IS NOT FALSE OR r.candidato_ref<>'can_'||repeat('a',22)
    OR r.recibo_ref !~ '^recibo:bolsa:cese:' THEN
  RAISE EXCEPTION 'B45: reconciliación no aplicó restricción';
 END IF;
 IF vec_bolsa_llamamientos.confirmar_cese_sin_candidato_bolsa_v1(o,h,p) IS DISTINCT FROM true
    OR vec_bolsa_llamamientos.listar_ceses_sin_candidato_pendientes_v1(100)<>'[]'::jsonb THEN
  RAISE EXCEPTION 'B81: replay o salida de pendientes divergente';
 END IF;
END $resolver_pendiente$;
RESET SESSION AUTHORIZATION;

SET SESSION AUTHORIZATION vec_b81_774_ejecutor;
DO $estado_restringido$
DECLARE v record;
BEGIN
 SELECT * INTO STRICT v FROM vec_bolsa_llamamientos.consultar_estado_cese_bolsa_v2(
  current_setting('vec.b81.participacion'),'2030-01-02T00:00:00Z'::timestamptz);
 IF v.cese_pendiente IS NOT FALSE OR v.fecha_efecto IS NULL
    OR v.disponible_desde IS NULL OR v.en_restriccion IS NOT TRUE THEN
  RAISE EXCEPTION 'B90: restricción confirmada no sustituye estado pendiente';
 END IF;
END $estado_restringido$;
RESET SESSION AUTHORIZATION;

-- Otro cese B13 todavía pendiente puede coexistir con una restricción B45
-- anterior del mismo candidato: las fechas acreditadas siguen presentes y
-- el pendiente conserva prioridad hasta su propia confirmación.
SAVEPOINT segundo_cese;
DO $segundo_cese$
DECLARE c vec_bolsa_llamamientos.contrato_participacion;
 origen text:='evento:ct:b81:774:cese:segundo'; evento_ref text; material jsonb;
 ahora timestamptz:=date_trunc('microseconds',clock_timestamp());
BEGIN
 SELECT * INTO STRICT c FROM vec_bolsa_llamamientos.contrato_participacion
  WHERE origen_ref=current_setting('vec.b81.origen');
 evento_ref:='evento:ct:contrato-bolsa:'||encode(sha256(convert_to('cese'||chr(31)||origen,'UTF8')),'hex');
 material:=c.evento||jsonb_build_object('origen_ref',origen,'evento_ref',evento_ref);
 INSERT INTO vec_bolsa_llamamientos.contrato_participacion(
  evento_ref,huella_sha256,evento,origen_ref,origen_creada_en,origen_posicion,tipo,
  organizacion_ref,expediente_ref,llamamiento_ref,participacion_ref,bolsa_ref,
  inicio,fin_previsto,modalidad_clave,categoria_ref,causa_clave,ocurrido_en,recibido_en)
 VALUES(evento_ref,encode(sha256(convert_to(material::text,'UTF8')),'hex'),material,
  origen,ahora,c.origen_posicion+1,'cese',c.organizacion_ref,c.expediente_ref,c.llamamiento_ref,
  c.participacion_ref,c.bolsa_ref,c.inicio,c.fin_previsto,c.modalidad_clave,c.categoria_ref,
  c.causa_clave,c.ocurrido_en,ahora);
END $segundo_cese$;
SET SESSION AUTHORIZATION vec_b81_774_ejecutor;
DO $estado_mixto$
DECLARE v record;
BEGIN
 SELECT * INTO STRICT v FROM vec_bolsa_llamamientos.consultar_estado_cese_bolsa_v2(
  current_setting('vec.b81.participacion'),'2030-01-02T00:00:00Z'::timestamptz);
 IF v.cese_pendiente IS NOT TRUE OR v.fecha_efecto IS NULL
    OR v.disponible_desde IS NULL OR v.en_restriccion IS NOT TRUE THEN
  RAISE EXCEPTION 'B90: cese pendiente adicional ocultó fechas acreditadas o quedó elegible';
 END IF;
END $estado_mixto$;
RESET SESSION AUTHORIZATION;
ROLLBACK TO SAVEPOINT segundo_cese;

SET SESSION AUTHORIZATION vec_b81_774_intruso;
DO $intruso$
BEGIN
 BEGIN
  PERFORM vec_contratacion_temporal.verificar_auditoria_cese_publicado_bolsa_v1(
   current_setting('vec.b81.origen'),current_setting('vec.b81.huella'),current_setting('vec.b81.posicion')::bigint);
  RAISE EXCEPTION 'CT197: intruso consultó auditoría de origen';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 BEGIN
  PERFORM vec_bolsa_llamamientos.confirmar_cese_sin_candidato_bolsa_v1(
   current_setting('vec.b81.origen'),current_setting('vec.b81.huella'),current_setting('vec.b81.posicion')::bigint);
  RAISE EXCEPTION 'B81: intruso ejecutó función';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 BEGIN
  PERFORM 1 FROM vec_bolsa_llamamientos.cese_sin_candidato_bolsa;
  RAISE EXCEPTION 'B81: intruso leyó tabla';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $intruso$;
RESET SESSION AUTHORIZATION;

-- El plan debe tener un camino por el índice del cursor en la tabla nueva.
SET LOCAL enable_seqscan = off;
EXPLAIN (COSTS OFF) SELECT origen_posicion,origen_ref
 FROM vec_bolsa_llamamientos.cese_sin_candidato_bolsa
 ORDER BY origen_posicion DESC,origen_ref DESC LIMIT 1;
ROLLBACK;
