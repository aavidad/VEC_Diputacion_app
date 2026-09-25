\set ON_ERROR_STOP on
-- Superficie EXCLUSIVA del contenedor desechable del ensayo de Dietas 000008.
-- Nunca se instala en una base conservada. Tras comprobar AD3-80 real, sustituye
-- en ESTE contenedor las dos fachadas AD3 que consume el circuito por dobles de
-- un solo uso (nonce), para recorrer la lógica de Dietas sin claves V3. Siembra
-- personas y una comisión enviada sintéticas.
BEGIN;
SET LOCAL search_path=pg_catalog;
CREATE SCHEMA prueba8 AUTHORIZATION postgres;
REVOKE ALL ON SCHEMA prueba8 FROM PUBLIC;
CREATE TABLE prueba8.consumo(nonce text PRIMARY KEY, audiencia text NOT NULL);
REVOKE ALL ON prueba8.consumo FROM PUBLIC;
GRANT USAGE ON SCHEMA prueba8 TO vec_autorizacion_atestada_v3_propietario;
GRANT SELECT,INSERT ON prueba8.consumo TO vec_autorizacion_atestada_v3_propietario;

-- Doble de consumo: exige audiencia, operación y referencias coherentes y
-- consume cada nonce una sola vez, como el núcleo real.
CREATE FUNCTION prueba8.consumir(p_audiencia text,p_capacidad bytea,p_decision bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
DECLARE c jsonb; d jsonb; nuevo boolean;
BEGIN
 c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 IF c->>'audiencia_consumo' IS DISTINCT FROM p_audiencia OR c->>'decision_ref' IS DISTINCT FROM d->>'decision_ref'
    OR d->>'accion' IS DISTINCT FROM c->>'operacion' OR c->>'nonce' IS NULL THEN
  RAISE EXCEPTION 'doble AD3: material incoherente' USING ERRCODE='42501';
 END IF;
 INSERT INTO prueba8.consumo VALUES(c->>'nonce',p_audiencia) ON CONFLICT DO NOTHING RETURNING true INTO nuevo;
 RETURN QUERY SELECT c->>'decision_ref',c->>'efecto_ref',c->>'huella_efecto_sha256',
  encode(sha256(convert_to(c->>'nonce','UTF8')),'hex'),'aud_doble_'||left(c->>'nonce',16),clock_timestamp(),coalesce(nuevo,false);
END $f$;
ALTER FUNCTION prueba8.consumir(text,bytea,bytea) OWNER TO vec_autorizacion_atestada_v3_propietario;

CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_dietas_circuito_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
BEGIN
 RETURN QUERY SELECT * FROM prueba8.consumir(convert_from(p_capacidad,'UTF8')::jsonb->>'audiencia_consumo',p_capacidad,p_decision);
END $f$;
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_dietas_revisor_documento_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
BEGIN
 RETURN QUERY SELECT * FROM prueba8.consumir('vec_dietas.circuito.documento.consultar.v1',p_capacidad,p_decision);
END $f$;

-- Personas sintéticas: T titular, A administrativo, R responsable, G gestión.
CREATE TABLE prueba8.persona(clave text PRIMARY KEY, persona_ref text NOT NULL);
INSERT INTO prueba8.persona VALUES
 ('T','per_titularSintetico000000001'),('A','per_administrativoSintetico01'),
 ('R','per_responsableSintetico00001'),('G','per_gestionSintetica000000001');
GRANT SELECT ON prueba8.persona TO PUBLIC;

INSERT INTO vec_personal.relacion_empleado_dietas
 (relacion_ref,persona_ref,empleado_ref,unidad_ref,estado,desde,hasta,version,procedencia_acto_ref,fuente_ref,fuente_version)
VALUES ('rel_relacionSintetica000000001','per_titularSintetico000000001','emp_empleadoSintetico00000001',
 'unidad:sintetica:01','activa',current_date-30,NULL,1,'acto:ensayo:000008','fuente:ensayo:000008',1);
INSERT INTO vec_personal.asignacion_dietas
 (asignacion_ref,relacion_ref,persona_ref,unidad_ref,version,centro_ref,administrativo_persona_ref,
  responsable_persona_ref,grupo_dieta,vigente_desde,motivo_revision,procedencia_acto_ref,registrada_por_ref,registrada_en)
VALUES ('ads_asignacionSintetica0000001','rel_relacionSintetica000000001','per_titularSintetico000000001',
 'unidad:sintetica:01',1,'centro:sintetico:01','per_administrativoSintetico01','per_responsableSintetico00001',
 2,current_date-30,'Asignación sintética del ensayo','acto:ensayo:000008','ensayo:000008',clock_timestamp());

-- Comisión enviada (v2) con su número y su cola de revisión. Los disparadores
-- exigen el ejecutor de la persona titular; en el ensayo se siembran a mano.
ALTER TABLE vec_dietas.borrador_comision DISABLE TRIGGER numerar_comision;
ALTER TABLE vec_dietas.comision_revision DISABLE TRIGGER abrir_cola_revision;
INSERT INTO vec_dietas.borrador_comision
 (referencia,persona_ref,empleado_ref,relacion_ref,unidad_ref,relacion_version,procedencia_acto_ref,
  fuente_ref,fuente_version,fecha_inicio,fecha_fin,motivo,codigos_ruta,clave_idempotencia,huella_semantica_sha256,creada_en)
VALUES ('dco_comisionSintetica0000000001','per_titularSintetico000000001','emp_empleadoSintetico00000001',
 'rel_relacionSintetica000000001','unidad:sintetica:01',1,'acto:ensayo:000008','fuente:ensayo:000008',1,
 current_date,current_date,'Reunión técnica sintética','["18087","18140"]','claveEnsayoAlta000001',
 repeat('a',64),clock_timestamp());
INSERT INTO vec_dietas.numero_documento_comision(comision_ref,secuencia,numero_documento,fecha_apertura)
VALUES ('dco_comisionSintetica0000000001',nextval('vec_dietas.numero_documento_comision_seq'),
 'VEC-D-'||to_char(current_date,'YYYY')||'-990001',clock_timestamp());
INSERT INTO vec_dietas.comision_revision
 (comision_ref,version,estado,fecha_inicio,fecha_fin,hora_inicio,hora_fin,motivo,codigos_ruta,
  vehiculo_propio,rutas,calculo,documento,regla_ref,asignacion_ref,asignacion_version,grupo_dieta,
  centro_ref,administrativo_persona_ref,responsable_persona_ref,registrada_en)
SELECT 'dco_comisionSintetica0000000001',2,'enviado_pendiente_revision',current_date,current_date,'08:00','16:30',
 'Reunión técnica sintética','["18087","18140"]',NULL,NULL,
 jsonb_build_object('version_tarifa',r.version_tarifa_ref,'regla_ref',r.regla_ref,'regla_huella_sha256',r.huella_sha256),
 jsonb_build_object('lineas',jsonb_build_array(jsonb_build_object('tipo','otro_gasto','concepto','Aparcamiento',
  'importe_centimos',650,'justificante',jsonb_build_object('referencia','just:ensayo:01','sha256',repeat('b',64))))),
 r.regla_ref,'ads_asignacionSintetica0000001',1,2,'centro:sintetico:01',
 'per_administrativoSintetico01','per_responsableSintetico00001',clock_timestamp()
FROM vec_dietas.regla_devengo_provisional r;
INSERT INTO vec_dietas.recibo_operacion_comision
 (referencia,comision_ref,version,operacion,clave_idempotencia,comando,huella_semantica_sha256,decision_ref,
  consumo_huella_sha256,auditoria_ad3_ref,actor_ref,persona_ref,regla_ref,regla_huella_sha256,registrada_en)
SELECT 'rcd_'||gen_random_uuid()::text,'dco_comisionSintetica0000000001',2,'enviar','claveEnsayoEnvio00001','{}',
 repeat('c',64),'dec_envio_ensayo',repeat('d',64),'aud_envio_ensayo','act_per_titularSintetico000000001',
 'per_titularSintetico000000001',r.regla_ref,r.huella_sha256,clock_timestamp()
FROM vec_dietas.regla_devengo_provisional r;
INSERT INTO vec_dietas.cola_circuito_comision
 (comision_ref,version,etapa,persona_ref,unidad_ref,asignacion_ref,asignacion_version,destinatario_persona_ref,
  fecha_inicio,fecha_fin,registrada_en)
VALUES ('dco_comisionSintetica0000000001',2,'revision','per_titularSintetico000000001','unidad:sintetica:01',
 'ads_asignacionSintetica0000001',1,'per_administrativoSintetico01',current_date,current_date,clock_timestamp());
ALTER TABLE vec_dietas.borrador_comision ENABLE TRIGGER numerar_comision;
ALTER TABLE vec_dietas.comision_revision ENABLE TRIGGER abrir_cola_revision;

-- Construye material, decisión, capacidad y contexto como los emite Go y los
-- pasa a la función nominal indicada. Se ejecuta con el LOGIN del ejecutor.
CREATE FUNCTION prueba8.invocar(p_funcion text,p_actor text,p_operacion text,p_etapa text,
 p_decision text,p_motivo text,p_clave text,p_version bigint,p_nonce text,p_unidad text DEFAULT 'unidad:sintetica:01')
RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
DECLARE persona text; ref text:='dco_comisionSintetica0000000001'; i jsonb; x jsonb; m jsonb; mt text;
 comando jsonb; huella_sem text; huella text; accion text; finalidad text; audiencia text; campos jsonb;
 d jsonb; c jsonb; bd bytea; bx bytea; verbo text; salida jsonb;
BEGIN
 SELECT persona_ref INTO STRICT persona FROM prueba8.persona WHERE clave=p_actor;
 i:=jsonb_build_object('actor_ref','act_'||persona,'perfil_ref','perfil:ensayo:'||p_actor,'persona_ref',persona,
  'contexto_actor_ref','ctx_'||p_actor,'contexto_version',1,'cuenta_ref','cue_'||p_actor,'cuenta_version',1,
  'persona_version',1,'perfil_version',1);
 x:=jsonb_build_object('esquema','vec.contexto-actor.vinculado.v2','contexto_actor_ref','ctx_'||p_actor,
  'contexto_version',1,'cuenta_ref','cue_'||p_actor,'cuenta_version',1,'persona_ref',persona,
  'principal_ref',persona,'perfil_activo_ref','perfil:ensayo:'||p_actor,'persona_version',1,'perfil_version',1,'estado','activo');
 IF p_operacion='decidir' THEN
  verbo:=CASE p_etapa WHEN 'revision' THEN 'revisar' WHEN 'autorizacion' THEN 'autorizar'
   WHEN 'liquidacion' THEN 'liquidar' ELSE 'fiscalizar' END;
  accion:='dietas.documento.'||verbo; finalidad:=verbo||'_documento_dietas';
  audiencia:='vec_dietas.documento.'||verbo||'.v1';
  campos:='["comision.estado","comision.referencia","comision.version","recibo.referencia","recibo.registrado_en","recibo.repeticion","recibo.version"]';
  comando:=jsonb_build_object('referencia',ref,'unidad_ref',p_unidad,'etapa',p_etapa,'decision',p_decision,
   'motivo',p_motivo,'clave_idempotencia',p_clave,'version_esperada',p_version);
  huella_sem:=encode(sha256(convert_to(concat_ws(chr(31),persona,ref,p_unidad,p_etapa,p_decision,p_motivo,p_clave,p_version::text),'UTF8')),'hex');
  m:=jsonb_build_object('esquema','vec.dietas.circuito-operacion.v2','operacion','decidir','recurso_ref',ref,
   'unidad_ref',p_unidad,'identidad',i,'huella_semantica',huella_sem,'comando',comando);
 ELSE
  accion:='dietas.circuito.documento.consultar'; finalidad:='revisar_documento_circuito_dietas';
  audiencia:='vec_dietas.circuito.documento.consultar.v1';
  campos:='["comision.calculo","comision.codigos_ruta","comision.documento","comision.estado","comision.fecha_apertura","comision.fecha_fin","comision.fecha_inicio","comision.hora_fin","comision.hora_inicio","comision.motivo","comision.numero_documento","comision.referencia","comision.rutas","comision.vehiculo_propio","comision.version","resultado"]';
  m:=jsonb_build_object('esquema','vec.dietas.circuito-operacion.v2','operacion','consultar_documento',
   'recurso_ref',ref,'unidad_ref',p_unidad,'etapa',p_etapa,'identidad',i);
 END IF;
 mt:=m::text;
 huella:=encode(sha256(convert_to('{"ambitos":{"persona_ref":'||to_json(persona)::text||',"unidad_ref":'||to_json(p_unidad)::text
  ||'},"atributos":{"contexto_actor_ref":'||to_json('ctx_'||p_actor)::text||',"contexto_version":"1","cuenta_ref":'
  ||to_json('cue_'||p_actor)::text||',"cuenta_version":"1"'
  ||CASE WHEN p_operacion='decidir' THEN '' ELSE ',"etapa":'||to_json(p_etapa)::text END
  ||',"material_sha256":"'||encode(sha256(convert_to(mt,'UTF8')),'hex')||'","operacion":'||to_json(p_operacion)::text
  ||',"perfil_version":"1","persona_version":"1","recurso_ref":'||to_json(ref)::text||'}}','UTF8')),'hex');
 d:=jsonb_build_object('decision_ref','dec_'||p_nonce,'principal_id','act_'||persona,'perfil_activo_ref','perfil:ensayo:'||p_actor,
  'concedida',true,'modulo_id','dietas','tipo_recurso','documento_dietas','obligaciones','[]'::jsonb,'recurso_ref',ref,
  'accion',accion,'finalidad',finalidad,'campos_permitidos',campos,'contexto_recurso_huella_sha256',huella);
 bd:=convert_to(d::text,'UTF8'); bx:=convert_to(x::text,'UTF8');
 c:=jsonb_build_object('decision_ref','dec_'||p_nonce,'efecto_ref',ref,'huella_efecto_sha256',huella,
  'huella_contexto_sha256',encode(sha256(bx),'hex'),'huella_decision_sha256',encode(sha256(bd),'hex'),
  'operacion',accion,'audiencia_consumo',audiencia,'nonce',encode(sha256(convert_to(p_nonce,'UTF8')),'hex'));
 EXECUTE format('SELECT vec_dietas.%I($1,$2,$3,$4,$5,1,1,$6,$6,$6,$6)',p_funcion)
  INTO salida USING mt,convert_to(c::text,'UTF8'),bd,'\x6d'::bytea,bx,'\x70'::bytea;
 RETURN salida;
END $f$;
REVOKE ALL ON FUNCTION prueba8.invocar(text,text,text,text,text,text,text,bigint,text,text) FROM PUBLIC;

-- Un único LOGIN, miembro solo del ejecutor Dietas, como en producción.
CREATE ROLE vec_prueba8_dietas LOGIN INHERIT NOBYPASSRLS;
GRANT vec_dietas_ejecutor TO vec_prueba8_dietas WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
GRANT USAGE ON SCHEMA prueba8 TO vec_prueba8_dietas;
GRANT EXECUTE ON FUNCTION prueba8.invocar(text,text,text,text,text,text,text,bigint,text,text) TO vec_prueba8_dietas;
COMMIT;
