-- F1-10: datos ficticios para las pruebas de B140. Lo incluye otra prueba con
-- \ir dentro de una transacción SERIALIZABLE de superusuario local; no abre
-- ni cierra transacción. Deja el contexto en pg_temp.b140_caso.
CREATE TEMP TABLE b140_caso(nombre text PRIMARY KEY,valor jsonb NOT NULL) ON COMMIT DROP;
GRANT SELECT,INSERT,UPDATE ON pg_temp.b140_caso TO vec_bolsa_llamamientos_propietario;

-- 1. Un expediente fiscalizado favorable pasa, solo aquí, a la categoría de la bolsa.
SET LOCAL session_replication_role=replica;
DO $ct$ DECLARE r record; necesidad text; ambito text; BEGIN
 SELECT e.expediente_ref,e.organizacion_ref,i.version,v.agregado_json AS a INTO r
 FROM vec_contratacion_temporal.expediente_alta e
 JOIN vec_contratacion_temporal.expediente_integral_actual i USING(expediente_ref)
 JOIN vec_contratacion_temporal.expediente_version_integral v
   ON v.expediente_ref=i.expediente_ref AND v.version=i.version
 WHERE v.agregado_json->>'fase_actual'='fiscalizacion' AND v.agregado_json->>'estado_actual'='en_curso'
   AND v.agregado_json#>>'{fiscalizacion,resultado}' IN ('favorable','favorable_con_observaciones')
   AND v.agregado_json#>>'{asignacion,unidad_ref}' IS NOT NULL
 ORDER BY e.expediente_ref LIMIT 1;
 IF NOT FOUND THEN RAISE EXCEPTION 'B140: la copia no tiene expediente fiscalizado favorable'; END IF;
 r.a:=jsonb_set(jsonb_set(jsonb_set(r.a,
   '{analisis,categoria_ref}','"categoria:rpt:administrativo"'),
   '{analisis,modalidad_clave}','"interinidad"'),'{analisis,causa_clave}','"vacante"');
 UPDATE vec_contratacion_temporal.expediente_version_integral v
 SET agregado_json=r.a,agregado_json_huella_sha256=encode(sha256(convert_to(r.a::text,'UTF8')),'hex')
 WHERE v.expediente_ref=r.expediente_ref AND v.version=r.version;
 necesidad:='necesidad:'||translate(encode(sha256(convert_to(
  replace(jsonb_build_array('vec.ct.bolsa.desarrollo.v1','necesidad',r.organizacion_ref,
   r.expediente_ref,'6')::text,', ',','),'UTF8')),'hex'),'0123456789','ghijklmnop');
 ambito:=(vec_catalogos_configurables.cotejar_conjunto_gestion_rrhh_inscripcion_actual_v1())->>'ambito_ref';
 INSERT INTO pg_temp.b140_caso VALUES('ctx',jsonb_build_object(
  'ambito_ref',ambito,'categoria_ref','categoria:rpt:administrativo','causa_ofrecida','vacante',
  'expediente_ref',r.expediente_ref,'modalidad_ofrecida','interinidad','necesidad_ref',necesidad,
  'organizacion_ref',r.organizacion_ref,'unidad_ref',r.a#>>'{asignacion,unidad_ref}',
  'version_expediente',r.version));
END $ct$;
SET LOCAL session_replication_role=origin;

-- 2. Bolsa: criterios publicados y una relación ficticia que deja pendiente a
-- la segunda persona elegible (receta de b118_contrato_contextual_fixture).
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
DO $bolsa$ DECLARE
 c record;pol record;fila record;bolsa text;criterios jsonb;po jsonb;v int;
 ahora timestamptz:=clock_timestamp()-interval '2 seconds';
 llam text;rec text;exp text;origen jsonb;evento jsonb;res record;
BEGIN
 -- La constitución vigente es la que elige la propia lectura de origen.
 SELECT x.* INTO STRICT c FROM vec_bolsa_llamamientos.constitucion x
 WHERE x.acta_ref=(SELECT l.acta_ref FROM vec_bolsa_llamamientos.leer_origen_llamamiento_ct_contextual_v1(
   'categoria:rpt:administrativo','interinidad','vacante') l);
 bolsa:=c.bolsa_ref;
 SELECT * INTO STRICT pol FROM vec_bolsa_llamamientos.politica_orden_bolsa
 WHERE bolsa_ref=bolsa ORDER BY version DESC LIMIT 1;
 SELECT e.participacion_ref,e.orden,vc.candidato_ref INTO STRICT fila
 FROM vec_bolsa_llamamientos.leer_orden_contextual_oferta_v1(bolsa,'interinidad','vacante',ahora) o
 JOIN vec_bolsa_llamamientos.constitucion_entrada e
   ON e.participacion_ref=o.participacion_ref AND e.orden=o.orden_acta
  AND e.instantanea_ref=c.instantanea_ref AND e.version_instantanea=c.version_instantanea
 JOIN vec_bolsa_llamamientos.vinculo_candidato vc ON vc.participacion_ref=e.participacion_ref
 WHERE o.orden_vigente=2;
 criterios:='{"fuente_normativa":"Prueba ficticia F1-10","mejora":{
  "habilitada":true,"modalidades_destino":["interinidad"],"causas_destino":["vacante"],
  "maximo_vacante_meses":36,"reduccion_por_modalidades_previas":["acumulacion_tareas"],
  "condiciones_actuales":[{"modalidad":"acumulacion_tareas"}]}}'::jsonb;
 po:=jsonb_build_object('criterios_ofrecimiento',criterios);
 SELECT coalesce(max(version),0) INTO v FROM vec_bolsa_llamamientos.politica_ofertas_version
 WHERE bolsa_ref=bolsa;
 INSERT INTO vec_bolsa_llamamientos.politica_ofertas_version(
  bolsa_ref,version,politica,huella_sha256,ejemplo,actor_ref,clave_idempotencia,
  version_esperada,recibo_ref,publicada_en,decision_ref,auditoria_ref)
 VALUES(bolsa,v+1,po,encode(sha256(convert_to(po::text,'UTF8')),'hex'),true,
  'per_'||repeat('a',22),'b140fixturepolitica',v,'recibo:politica-ofertas:'||repeat('c',64),
  ahora,'decision:b140:politica','auditoria:b140:politica');
 llam:='llamamiento:'||encode(sha256(convert_to('b140-fixture','UTF8')),'hex');
 rec:='recibo:llamamiento:'||encode(sha256(convert_to('b140-fixture-recibo','UTF8')),'hex');
 exp:='expediente:b140:fixture';
 INSERT INTO vec_bolsa_llamamientos.llamamiento_emitido(
  llamamiento_ref,recibo_ref,bolsa_ref,actor_ref,clave_idempotencia,participaciones,
  configuracion,huella_comando_sha256,huella_finalizacion,estado,emitido_en,decision_ref)
 VALUES(llam,rec,bolsa,'per_'||repeat('a',22),'b140fixturellamamiento',
  jsonb_build_array(fila.participacion_ref),'{}'::jsonb,repeat('a',64),
  decode(repeat('b',64),'hex'),'emision_reservada',ahora,'decision:b140:llamamiento');
 origen:=jsonb_build_object('organizacion_ref','organizacion:desarrollo:dipgra',
  'unidad_ref','unidad:b140:rrhh','expediente_ref',exp,'clave_ct','clave:b140',
  'bolsa_ref',bolsa,'participacion_ref',fila.participacion_ref,
  'candidato_ref',fila.candidato_ref,'version_token','b140');
 INSERT INTO vec_bolsa_llamamientos.origen_llamamiento_ct_v1(
  organizacion_ref,unidad_ref,expediente_ref,version_expediente,clave_ct,necesidad_ref,
  acta_ref,categoria_ref,bolsa_ref,version_bolsa,huella_bolsa_sha256,instantanea_ref,
  version_instantanea,huella_instantanea_sha256,politica_ref,version_politica,
  huella_politica_sha256,participacion_ref,candidato_ref,orden_seleccionado,version_token,
  llamamiento_ref,recibo_ref,decision_ref,auditoria_ref,evento_ref,emitido_en,
  origen_canonico,origen_huella_sha256)
 VALUES('organizacion:desarrollo:dipgra','unidad:b140:rrhh',exp,1,'clave:b140',
  'necesidad:b140',c.acta_ref,c.categoria_ref,bolsa,c.version_bolsa,c.huella_bolsa_sha256,
  c.instantanea_ref,c.version_instantanea,c.huella_instantanea_sha256,pol.politica_ref,
  pol.version,repeat('a',64),fila.participacion_ref,fila.candidato_ref,fila.orden,'b140',
  llam,rec,'decision:b140:origen','auditoria:b140:origen','evento:bolsa:ct:b140',ahora,
  origen,encode(sha256(convert_to(origen::text,'UTF8')),'hex'));
 evento:=jsonb_build_object('categoria_ref',c.categoria_ref,'causa_clave','necesidad_temporal',
  'esquema','vec.contratacion-temporal.contrato-bolsa.v1',
  'evento_ref','evento:ct:contrato-bolsa:'||encode(sha256(convert_to(
    'incorporacion'||chr(31)||'ref:b140:fixture','UTF8')),'hex'),
  'expediente_ref',exp,'fin_previsto',NULL,'inicio',NULL,'llamamiento_ref',llam,
  'modalidad_clave','acumulacion_tareas',
  'ocurrido_en',to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'organizacion_ref','organizacion:desarrollo:dipgra','origen_ref','ref:b140:fixture',
  'tipo','incorporacion');
 SELECT * INTO STRICT res FROM vec_bolsa_llamamientos.registrar_contrato_participacion_v1(
  evento,encode(sha256(convert_to(evento::text,'UTF8')),'hex'),ahora,fila.orden);
 IF res.participacion_ref IS DISTINCT FROM fila.participacion_ref OR res.en_cuarentena
 THEN RAISE EXCEPTION 'B140: la relación ficticia no quedó enlazada'; END IF;
 INSERT INTO vec_bolsa_llamamientos.situacion_participacion(
  participacion_ref,situacion,desde,hasta,fecha_disponible,motivo,actor,registrada_en,
  clave_idempotencia,recibo_ref)
 VALUES(fila.participacion_ref,'trabajando',ahora,NULL,NULL,'Relación ficticia F1-10',
  'sistema:b140',ahora,'b140:trabajando','recibo:b140:trabajando');
 INSERT INTO pg_temp.b140_caso VALUES('pendiente',jsonb_build_object(
  'participacion_ref',fila.participacion_ref,'orden_acta',fila.orden));
END $bolsa$;

RESET ROLE;
