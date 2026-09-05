-- Integración sintética mínima: una línea canónica de orden/propuesta/apertura.
-- No habilita guardar_propuesta_v1 ni afirma cerrar el proveedor Bolsa V2.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

CREATE SCHEMA IF NOT EXISTS vec_bolsa_llamamientos AUTHORIZATION vec_bolsa_llamamientos_propietario;
REVOKE ALL ON SCHEMA vec_bolsa_llamamientos FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_bolsa_llamamientos_propietario
 IN SCHEMA vec_bolsa_llamamientos REVOKE ALL ON TABLES FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_bolsa_llamamientos_propietario REVOKE ALL ON FUNCTIONS FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_bolsa_llamamientos TO vec_bolsa_llamamientos_ejecutor;

DO $dependencia$
BEGIN
 IF to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_bolsa_llamamiento_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL THEN
  RAISE EXCEPTION USING ERRCODE='55000',MESSAGE='falta consumidor atestado nominal de Bolsa';
 END IF;
END
$dependencia$;

-- Esta tabla es también el inbox idempotente: operación y resultado completos
-- son inmutables. No hay una segunda versión en memoria ni una copia en CT.
CREATE TABLE vec_bolsa_llamamientos.integracion_desarrollo (
 operacion_ref text PRIMARY KEY,
 tipo text NOT NULL CHECK (tipo IN ('orden','propuesta')),
 necesidad_ref text NOT NULL,
 version_necesidad bigint NOT NULL CHECK (version_necesidad BETWEEN 1 AND 9007199254740991),
 orden_operacion_ref text REFERENCES vec_bolsa_llamamientos.integracion_desarrollo(operacion_ref),
 registro_canonico bytea NOT NULL CHECK (octet_length(registro_canonico) BETWEEN 1 AND 4194304),
 registro_huella_sha256 text NOT NULL CHECK (registro_huella_sha256=encode(sha256(registro_canonico),'hex')),
 contexto_huella_sha256 text NOT NULL CHECK (contexto_huella_sha256 ~ '^[0-9a-f]{64}$'),
 decision_ref text NOT NULL UNIQUE,
 recibo_ref text NOT NULL UNIQUE,
 confirmada_en timestamptz NOT NULL,
 UNIQUE(tipo,necesidad_ref,version_necesidad),
 CHECK (operacion_ref ~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{0,191}$'),
 CHECK ((tipo='orden' AND orden_operacion_ref IS NULL) OR (tipo='propuesta' AND orden_operacion_ref IS NOT NULL))
);

CREATE TABLE vec_bolsa_llamamientos.llamamiento_integracion_desarrollo (
 llamamiento_ref text PRIMARY KEY,
 operacion_ref text NOT NULL UNIQUE REFERENCES vec_bolsa_llamamientos.integracion_desarrollo(operacion_ref),
 propuesta_ref text NOT NULL UNIQUE,
 bolsa_ref text NOT NULL,
 necesidad_ref text NOT NULL,
 version bigint NOT NULL CHECK (version=1),
 estado text NOT NULL CHECK (estado='abierto'),
 abierto_en timestamptz NOT NULL,
 datos_canonicos jsonb NOT NULL
);

CREATE TABLE vec_bolsa_llamamientos.auditoria_integracion_desarrollo (
 auditoria_ref text PRIMARY KEY,
 operacion_ref text NOT NULL UNIQUE REFERENCES vec_bolsa_llamamientos.integracion_desarrollo(operacion_ref),
 secuencia bigint NOT NULL UNIQUE CHECK (secuencia>0),
 anterior_sha256 text NOT NULL CHECK (anterior_sha256 ~ '^[0-9a-f]{64}$'),
 registro_canonico bytea NOT NULL,
 huella_sha256 text NOT NULL CHECK (huella_sha256=encode(sha256(registro_canonico),'hex'))
);

CREATE TABLE vec_bolsa_llamamientos.outbox_integracion_desarrollo (
 evento_ref text PRIMARY KEY,
 operacion_ref text NOT NULL UNIQUE REFERENCES vec_bolsa_llamamientos.integracion_desarrollo(operacion_ref),
 evento_canonico bytea NOT NULL,
 huella_sha256 text NOT NULL CHECK (huella_sha256=encode(sha256(evento_canonico),'hex')),
 emitido_en timestamptz NOT NULL
);

CREATE FUNCTION vec_bolsa_llamamientos.rechazar_mutacion_integracion_desarrollo()
RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
BEGIN
 RAISE EXCEPTION USING ERRCODE='55000',MESSAGE='historia de integración Bolsa inmutable';
END
$f$;
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.integracion_desarrollo
 FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.rechazar_mutacion_integracion_desarrollo();
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.llamamiento_integracion_desarrollo
 FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.rechazar_mutacion_integracion_desarrollo();
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.auditoria_integracion_desarrollo
 FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.rechazar_mutacion_integracion_desarrollo();
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.outbox_integracion_desarrollo
 FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.rechazar_mutacion_integracion_desarrollo();

CREATE FUNCTION vec_bolsa_llamamientos.exigir_runtime_integracion_desarrollo()
RETURNS void LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR session_user=current_user OR
  NOT pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER') OR
  pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER') OR
  pg_has_role(session_user,'vec_bolsa_llamamientos_migrador','MEMBER') OR
  current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off' OR
  current_setting('TimeZone')<>'UTC' OR
  NOT EXISTS(SELECT 1 FROM pg_settings WHERE name='statement_timeout' AND setting::numeric BETWEEN 1 AND 15000) OR
  NOT EXISTS(SELECT 1 FROM pg_settings WHERE name='idle_in_transaction_session_timeout' AND setting::numeric BETWEEN 1 AND 20000)
 THEN
  RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='runtime de integración Bolsa rechazado';
 END IF;
END
$f$;

-- Puerto privado del proveedor para preparar la autorización ligada al replay.
-- Nunca se compone directamente como una consulta HTTP ni como recibo para CT.
CREATE FUNCTION vec_bolsa_llamamientos.buscar_integracion_desarrollo_v1(p_operacion_ref text)
RETURNS TABLE(registro_canonico bytea)
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
BEGIN
 PERFORM vec_bolsa_llamamientos.exigir_runtime_integracion_desarrollo();
 RETURN QUERY SELECT r.registro_canonico FROM vec_bolsa_llamamientos.integracion_desarrollo r WHERE r.operacion_ref=p_operacion_ref;
END
$f$;

CREATE FUNCTION vec_bolsa_llamamientos.guardar_integracion_desarrollo_v1(
 p_registro bytea,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_cose bytea,p_evidencia bytea,p_raiz bytea
)
RETURNS TABLE(registro_canonico bytea,recibo_ref text,auditoria_ref text,evento_ref text,confirmada_en timestamptz)
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE
 r jsonb; i jsonb; f jsonb; p jsonb; l jsonb; c jsonb; e jsonb; entrada jsonb; situacion jsonb; regla jsonb;
 v_hash text; v_contexto text; v_contexto_hash text; v_accion text; v_indice integer;
 v_orden record; v_anterior record; v_consumo record; v_existente record;
 v_ahora timestamptz; v_recibo text; v_auditoria text; v_evento text;
 v_auditoria_bytes bytea; v_evento_bytes bytea; v_secuencia bigint; v_anterior_hash text;
BEGIN
 PERFORM vec_bolsa_llamamientos.exigir_runtime_integracion_desarrollo();
 IF p_registro IS NULL OR octet_length(p_registro) NOT BETWEEN 1 AND 4194304 OR
  p_capacidad IS NULL OR octet_length(p_capacidad) NOT BETWEEN 512 AND 32768 THEN
  RAISE EXCEPTION USING ERRCODE='22023',MESSAGE='registro Bolsa inválido';
 END IF;
 r:=convert_from(p_registro,'UTF8')::jsonb;
 c:=convert_from(p_capacidad,'UTF8')::jsonb;
 i:=r->'instantanea'; f:=r->'fuente'; p:=r->'propuesta'; l:=r->'llamamiento';
 IF (r->>'esquema'='vec.bolsa.integracion-llamamientos-desarrollo.v1' AND
  r->>'tipo' IN ('orden','propuesta') AND
  r->>'operacion_ref' ~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{0,191}$' AND
  r->>'necesidad_ref'=f#>>'{datos,Necesidad,necesidad_ref}' AND
  r->>'version_necesidad'=f#>>'{datos,Necesidad,version}' AND
  r->>'categoria_ref'=f#>>'{datos,Necesidad,categoria_ref}' AND
  r->>'unidad_ref'=f#>>'{datos,Necesidad,unidad_ref}' AND
  f->>'esquema'='vec.bolsa.fuente-llamamientos-sintetica.v1' AND
  octet_length(decode(r->>'firma_fuente','base64'))=64 AND
  i->'entradas'=f#>'{datos,Entradas}' AND
  i->>'bolsa_ref'=f#>>'{datos,Bolsa,bolsa_ref}' AND
  i->>'version_bolsa'=f#>>'{datos,Bolsa,version}' AND
  jsonb_array_length(i->'entradas') BETWEEN 1 AND 128 AND
  jsonb_array_length(f->'reglas')=5) IS NOT TRUE THEN
  RAISE EXCEPTION USING ERRCODE='22023',MESSAGE='fuente u orden Bolsa incoherente';
 END IF;
 v_hash:=encode(sha256(p_registro),'hex');
 v_contexto:='{"ambitos":{"categoria_ref":'||to_json(r->>'categoria_ref')::text||
  ',"unidad_ref":'||to_json(r->>'unidad_ref')::text||'},"atributos":{"contenido_sha256":'||
  to_json(v_hash)::text||',"necesidad_ref":'||to_json(r->>'necesidad_ref')::text||'}}';
 v_contexto_hash:=encode(sha256(convert_to(v_contexto,'UTF8')),'hex');
 v_accion:=CASE r->>'tipo' WHEN 'orden' THEN 'bolsa.orden.preparar' ELSE 'bolsa.llamamiento.abrir' END;
 IF c->>'efecto_ref' IS DISTINCT FROM r->>'operacion_ref' OR
  c->>'huella_efecto_sha256' IS DISTINCT FROM v_contexto_hash OR
  c->>'operacion' IS DISTINCT FROM v_accion OR
  c->>'audiencia_consumo' IS DISTINCT FROM 'vec_bolsa_llamamientos.confirmar_integracion_desarrollo.v1' THEN
  RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='autorización no ligada a la integración Bolsa';
 END IF;
 -- Todos los bloqueos preceden a la revalidación temporal del consumidor VEC.
 PERFORM pg_advisory_xact_lock(hashtextextended('bolsa:integracion:necesidad:'||(r->>'necesidad_ref'),0));
 PERFORM pg_advisory_xact_lock(hashtextextended('bolsa:integracion:auditoria',0));
 IF r->>'tipo'='orden' THEN
  IF r->>'orden_operacion_ref' IS DISTINCT FROM '' OR p IS NOT NULL OR l IS NOT NULL OR r?'estado_llamamiento' THEN
   RAISE EXCEPTION USING ERRCODE='22023',MESSAGE='orden con efectos no solicitados';
  END IF;
 ELSE
  SELECT * INTO STRICT v_orden FROM vec_bolsa_llamamientos.integracion_desarrollo o
   WHERE o.operacion_ref=r->>'orden_operacion_ref' AND o.tipo='orden'
   AND o.necesidad_ref=r->>'necesidad_ref' AND o.version_necesidad=(r->>'version_necesidad')::bigint FOR SHARE;
  IF (convert_from(v_orden.registro_canonico,'UTF8')::jsonb->'instantanea'=i AND
   convert_from(v_orden.registro_canonico,'UTF8')::jsonb->'fuente'=f AND
   p->>'instantanea_ref'=i->>'instantanea_ref' AND p->>'huella_instantanea_sha256'=i->>'huella_contenido_sha256' AND
   p->>'necesidad_ref'=r->>'necesidad_ref' AND p->>'version_necesidad'=r->>'version_necesidad' AND
   p->>'bolsa_ref'=i->>'bolsa_ref' AND
   l->>'LlamamientoRef' ~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{0,191}$' AND
   l->>'BolsaRef'=p->>'bolsa_ref' AND l->>'NecesidadRef'=p->>'necesidad_ref' AND
   l->>'PropuestaRef'=p->>'propuesta_ref' AND l->>'Version'='1' AND r->>'estado_llamamiento'='abierto' AND
   jsonb_array_length(p->'evaluaciones') BETWEEN 1 AND jsonb_array_length(i->'entradas') AND
   (p->>'orden_seleccionado')::integer=jsonb_array_length(p->'evaluaciones')) IS NOT TRUE THEN
   RAISE EXCEPTION USING ERRCODE='22023',MESSAGE='propuesta o agregado abierto incoherente';
  END IF;
  FOR v_indice IN 0..jsonb_array_length(p->'evaluaciones')-1 LOOP
   e:=p->'evaluaciones'->v_indice; entrada:=i->'entradas'->v_indice;
   SELECT s INTO STRICT situacion FROM jsonb_array_elements(entrada#>'{participacion,situaciones}') s
    WHERE s->>'secuencia'=e->>'situacion_secuencia';
   SELECT g INTO STRICT regla FROM jsonb_array_elements(f->'reglas') g
    WHERE g->>'estado_clave'=situacion->>'estado_clave' AND g->>'estado_version'=situacion->>'estado_version'
     AND g->>'huella_estado_sha256'=situacion->>'huella_estado_sha256';
   IF ((e->>'orden')::integer=v_indice+1 AND (entrada->>'orden')::integer=v_indice+1 AND
    e->>'participacion_ref'=entrada#>>'{participacion,participacion_ref}' AND
    e->>'sujeto_ref'=entrada#>>'{participacion,sujeto_ref}' AND
    e->>'estado_clave'=situacion->>'estado_clave' AND e->>'estado_version'=situacion->>'estado_version' AND
    e->>'huella_estado_sha256'=situacion->>'huella_estado_sha256' AND
    e->>'resultado'=regla->>'resultado' AND
    e->>'resultado'=CASE WHEN v_indice=jsonb_array_length(p->'evaluaciones')-1 THEN 'elegible' ELSE 'no_elegible' END AND
    (situacion->>'desde')::timestamptz<=(i->>'referida_en')::timestamptz AND
    (situacion->>'hasta' IS NULL OR (i->>'referida_en')::timestamptz<(situacion->>'hasta')::timestamptz)) IS NOT TRUE THEN
    RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='orden de elegibilidad no acreditado';
   END IF;
  END LOOP;
  IF p->>'participacion_seleccionada_ref' IS DISTINCT FROM e->>'participacion_ref' OR
   p->>'sujeto_seleccionado_ref' IS DISTINCT FROM e->>'sujeto_ref' THEN
   RAISE EXCEPTION USING ERRCODE='22023',MESSAGE='selección distinta del primer elegible';
  END IF;
 END IF;
 SELECT * INTO v_existente FROM vec_bolsa_llamamientos.integracion_desarrollo o
  WHERE o.operacion_ref=r->>'operacion_ref' FOR SHARE;
 IF FOUND AND v_existente.registro_canonico IS DISTINCT FROM p_registro THEN
  RAISE EXCEPTION USING ERRCODE='23505',MESSAGE='operación Bolsa divergente';
 END IF;
 IF v_existente.operacion_ref IS NULL AND EXISTS(
  SELECT 1 FROM vec_bolsa_llamamientos.integracion_desarrollo o
  WHERE o.tipo=r->>'tipo' AND o.necesidad_ref=r->>'necesidad_ref' AND o.version_necesidad=(r->>'version_necesidad')::bigint
 ) THEN
  RAISE EXCEPTION USING ERRCODE='23505',MESSAGE='necesidad Bolsa ya confirmada';
 END IF;
 v_ahora:=clock_timestamp();
 IF ((f->>'vigente_desde')::timestamptz<=v_ahora AND v_ahora<(f->>'vigente_hasta')::timestamptz AND
  (i->>'generada_en')::timestamptz<=v_ahora AND v_ahora<(f#>>'{datos,Necesidad,fin_previsto}')::timestamptz) IS NOT TRUE THEN
  RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='fuente Bolsa no vigente';
 END IF;
 SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_bolsa_llamamiento_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_cose,p_evidencia,p_raiz);
 -- Incluso al recuperar un efecto, exigir una decisión nueva. El consumo
 -- histórico del núcleo no acredita vigencia ni ausencia de revocación hoy.
 IF v_consumo.efecto_ref IS DISTINCT FROM r->>'operacion_ref' OR
  v_consumo.huella_efecto_sha256 IS DISTINCT FROM v_contexto_hash OR
  v_consumo.consumo_nuevo IS NOT TRUE THEN
  RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='consumo Bolsa no ligado al efecto';
 END IF;
 IF v_existente.operacion_ref IS NOT NULL THEN
  RETURN QUERY SELECT o.registro_canonico,o.recibo_ref,a.auditoria_ref,x.evento_ref,o.confirmada_en
   FROM vec_bolsa_llamamientos.integracion_desarrollo o JOIN vec_bolsa_llamamientos.auditoria_integracion_desarrollo a USING(operacion_ref)
   JOIN vec_bolsa_llamamientos.outbox_integracion_desarrollo x USING(operacion_ref) WHERE o.operacion_ref=v_existente.operacion_ref;
  RETURN;
 END IF;
 v_ahora:=clock_timestamp();
 -- El consumidor puede haber esperado cerrojos VEC. No usar la ventana de
 -- fuente comprobada antes de esa espera para confirmar un efecto nuevo.
 IF ((f->>'vigente_desde')::timestamptz<=v_ahora AND v_ahora<(f->>'vigente_hasta')::timestamptz AND
  v_ahora<(f#>>'{datos,Necesidad,fin_previsto}')::timestamptz AND
  (r->>'tipo'='orden' OR ((p->>'generada_en')::timestamptz<=v_ahora AND
   (p->>'generada_en')::timestamptz>=(i->>'generada_en')::timestamptz))) IS NOT TRUE THEN
  RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='fuente Bolsa expirada durante autorización';
 END IF;
 v_recibo:='recibo:'||translate(v_hash,'0123456789','ghijklmnop');
 INSERT INTO vec_bolsa_llamamientos.integracion_desarrollo VALUES(
  r->>'operacion_ref',r->>'tipo',r->>'necesidad_ref',(r->>'version_necesidad')::bigint,
  nullif(r->>'orden_operacion_ref',''),p_registro,v_hash,v_contexto_hash,v_consumo.decision_ref,v_recibo,v_ahora);
 IF r->>'tipo'='propuesta' THEN
  INSERT INTO vec_bolsa_llamamientos.llamamiento_integracion_desarrollo VALUES(
   l->>'LlamamientoRef',r->>'operacion_ref',l->>'PropuestaRef',l->>'BolsaRef',l->>'NecesidadRef',1,'abierto',v_ahora,l);
 END IF;
 SELECT a.secuencia,a.huella_sha256 INTO v_anterior FROM vec_bolsa_llamamientos.auditoria_integracion_desarrollo a ORDER BY a.secuencia DESC LIMIT 1;
 v_secuencia:=coalesce(v_anterior.secuencia,0)+1; v_anterior_hash:=coalesce(v_anterior.huella_sha256,repeat('0',64));
 v_auditoria:='auditoria:'||translate(encode(sha256(convert_to(v_hash||':'||v_anterior_hash,'UTF8')),'hex'),'0123456789','ghijklmnop');
 v_auditoria_bytes:=convert_to(jsonb_build_object('esquema','vec.bolsa.integracion.auditoria.v1',
  'operacion_ref',r->>'operacion_ref','secuencia',v_secuencia,'anterior_sha256',v_anterior_hash,
  'contenido_sha256',v_hash,'decision_ref',v_consumo.decision_ref,'consumo_huella_sha256',v_consumo.consumo_huella_sha256,
  'registrada_en',v_ahora)::text,'UTF8');
 INSERT INTO vec_bolsa_llamamientos.auditoria_integracion_desarrollo VALUES(v_auditoria,r->>'operacion_ref',v_secuencia,v_anterior_hash,v_auditoria_bytes,encode(sha256(v_auditoria_bytes),'hex'));
 v_evento:='evento:'||translate(v_hash,'0123456789','ghijklmnop');
 v_evento_bytes:=convert_to(jsonb_build_object('esquema','vec.bolsa.integracion.outbox.v1',
  'tipo',CASE r->>'tipo' WHEN 'orden' THEN 'bolsa.orden.confirmada' ELSE 'bolsa.llamamiento.abierto' END,
  'operacion_ref',r->>'operacion_ref','recibo_ref',v_recibo,'contenido_sha256',v_hash,'emitido_en',v_ahora)::text,'UTF8');
 INSERT INTO vec_bolsa_llamamientos.outbox_integracion_desarrollo VALUES(v_evento,r->>'operacion_ref',v_evento_bytes,encode(sha256(v_evento_bytes),'hex'),v_ahora);
 RETURN QUERY SELECT p_registro,v_recibo,v_auditoria,v_evento,v_ahora;
EXCEPTION WHEN no_data_found OR too_many_rows THEN
 RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='fuente Bolsa ausente o ambigua';
END
$f$;
REVOKE ALL ON vec_bolsa_llamamientos.integracion_desarrollo,
 vec_bolsa_llamamientos.llamamiento_integracion_desarrollo,
 vec_bolsa_llamamientos.auditoria_integracion_desarrollo,
 vec_bolsa_llamamientos.outbox_integracion_desarrollo FROM PUBLIC,vec_bolsa_llamamientos_ejecutor;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.buscar_integracion_desarrollo_v1(text) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.guardar_integracion_desarrollo_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.buscar_integracion_desarrollo_v1(text),
 vec_bolsa_llamamientos.guardar_integracion_desarrollo_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_bolsa_llamamientos_ejecutor;
COMMIT;
