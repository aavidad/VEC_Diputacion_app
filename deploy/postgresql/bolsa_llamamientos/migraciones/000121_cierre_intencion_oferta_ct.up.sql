\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000121',0));
DO $pre$
DECLARE h text;
BEGIN
 SELECT encode(sha256(convert_to(pg_get_functiondef(to_regprocedure(
  'vec_bolsa_llamamientos.publicar_oferta_v5(text,text,text,text,text,jsonb,jsonb,timestamptz,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text,integer,jsonb)')),'UTF8')),'hex') INTO h;
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR h IS DISTINCT FROM '3811f4fbb392f0b8c37f2583ae163e0aefafdf71fa34e2f7fc62454ef236d995'
    OR to_regclass('vec_bolsa_llamamientos.origen_oferta_ct_v1') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.oferta_publicada') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.verificar_oferta_ct_v1(text,text,bigint,text,text,text,text,text,text,integer,jsonb,text)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_cierre_intencion_oferta_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.operacion_oferta_clave_ct_v1') IS NOT NULL
    OR to_regclass('vec_bolsa_llamamientos.intencion_oferta_ct_terminal_v1') IS NOT NULL
 THEN RAISE EXCEPTION 'B121: preimagen B117/B120/AD278 incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
LOCK TABLE vec_bolsa_llamamientos.oferta_publicada,
 vec_bolsa_llamamientos.origen_oferta_ct_v1 IN SHARE ROW EXCLUSIVE MODE;

-- Primera exclusión física: cualquier publicación posterior a B121 ocupa
-- (bolsa,clave). El cierre ocupa exactamente la misma PK. UNIQUE ve la fila
-- concurrente aun con una instantánea SERIALIZABLE anterior.
CREATE TABLE vec_bolsa_llamamientos.operacion_oferta_clave_ct_v1(
 bolsa_ref text NOT NULL,clave_bolsa text NOT NULL,
 resultado text NOT NULL CHECK(resultado IN ('publicada','cerrada_sin_publicar')),
 actor_ref text NOT NULL,
 oferta_ref text UNIQUE REFERENCES vec_bolsa_llamamientos.oferta_publicada(oferta_ref),
 recibo_oferta_ref text UNIQUE,
 recibo_cierre_ref text UNIQUE,
 registrada_en timestamptz(6) NOT NULL,
 PRIMARY KEY(bolsa_ref,clave_bolsa),
 CHECK((resultado='publicada' AND oferta_ref IS NOT NULL AND recibo_oferta_ref IS NOT NULL AND recibo_cierre_ref IS NULL)
    OR (resultado='cerrada_sin_publicar' AND oferta_ref IS NULL AND recibo_oferta_ref IS NULL AND recibo_cierre_ref IS NOT NULL))
);
ALTER TABLE vec_bolsa_llamamientos.operacion_oferta_clave_ct_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.operacion_oferta_clave_ct_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY solo_propietario ON vec_bolsa_llamamientos.operacion_oferta_clave_ct_v1
 TO vec_bolsa_llamamientos_propietario
 USING(current_user='vec_bolsa_llamamientos_propietario')
 WITH CHECK(current_user='vec_bolsa_llamamientos_propietario');
REVOKE ALL ON TABLE vec_bolsa_llamamientos.operacion_oferta_clave_ct_v1 FROM PUBLIC;
REVOKE ALL ON TYPE vec_bolsa_llamamientos.operacion_oferta_clave_ct_v1 FROM PUBLIC;
CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.operacion_oferta_clave_ct_v1
 FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_bolsa_llamamientos.operacion_oferta_clave_ct_v1
 FOR EACH STATEMENT EXECUTE FUNCTION vec_bolsa_llamamientos.rechazar_truncado_origen_oferta_ct_v1();

-- Segunda exclusión física: el mismo expediente/operación CT tampoco puede
-- cerrarse con la clave Bolsa A y publicarse más tarde con la B.
CREATE TABLE vec_bolsa_llamamientos.intencion_oferta_ct_terminal_v1(
 organizacion_ref text NOT NULL,expediente_ref text NOT NULL,clave_ct text NOT NULL,
 version_expediente bigint NOT NULL CHECK(version_expediente BETWEEN 1 AND 9007199254740991),
 bolsa_ref text NOT NULL,clave_bolsa text NOT NULL,
 actor_ref text NOT NULL,numero_plazas integer NOT NULL CHECK(numero_plazas BETWEEN 1 AND 100),
 datos_sha256 text NOT NULL CHECK(datos_sha256 ~ '^[0-9a-f]{64}$'),
 origen_sha256 text NOT NULL CHECK(origen_sha256 ~ '^[0-9a-f]{64}$'),
 comando_sha256 text CHECK(comando_sha256 IS NULL OR comando_sha256 ~ '^[0-9a-f]{64}$'),
 resultado text NOT NULL CHECK(resultado IN ('publicada','cerrada_sin_publicar')),
 oferta_ref text UNIQUE REFERENCES vec_bolsa_llamamientos.origen_oferta_ct_v1(oferta_ref)
   DEFERRABLE INITIALLY DEFERRED,
 recibo_ref text NOT NULL UNIQUE,
 decision_ref text NOT NULL,auditoria_ref text NOT NULL,
 evento_ref text UNIQUE,recibo_json jsonb,
 registrada_en timestamptz(6) NOT NULL,
 PRIMARY KEY(organizacion_ref,expediente_ref,clave_ct),
 FOREIGN KEY(bolsa_ref,clave_bolsa)
   REFERENCES vec_bolsa_llamamientos.operacion_oferta_clave_ct_v1(bolsa_ref,clave_bolsa),
 CHECK((resultado='publicada' AND oferta_ref IS NOT NULL AND comando_sha256 IS NULL
        AND evento_ref IS NULL AND recibo_json IS NULL)
    OR (resultado='cerrada_sin_publicar' AND oferta_ref IS NULL AND comando_sha256 IS NOT NULL
        AND evento_ref IS NOT NULL AND jsonb_typeof(recibo_json)='object'))
);
ALTER TABLE vec_bolsa_llamamientos.intencion_oferta_ct_terminal_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.intencion_oferta_ct_terminal_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY solo_propietario ON vec_bolsa_llamamientos.intencion_oferta_ct_terminal_v1
 TO vec_bolsa_llamamientos_propietario
 USING(current_user='vec_bolsa_llamamientos_propietario')
 WITH CHECK(current_user='vec_bolsa_llamamientos_propietario');
REVOKE ALL ON TABLE vec_bolsa_llamamientos.intencion_oferta_ct_terminal_v1 FROM PUBLIC;
REVOKE ALL ON TYPE vec_bolsa_llamamientos.intencion_oferta_ct_terminal_v1 FROM PUBLIC;
CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.intencion_oferta_ct_terminal_v1
 FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_bolsa_llamamientos.intencion_oferta_ct_terminal_v1
 FOR EACH STATEMENT EXECUTE FUNCTION vec_bolsa_llamamientos.rechazar_truncado_origen_oferta_ct_v1();

CREATE TABLE vec_bolsa_llamamientos.cierre_oferta_ct_outbox_v1(
 evento_ref text PRIMARY KEY,
 organizacion_ref text NOT NULL,expediente_ref text NOT NULL,clave_ct text NOT NULL,
 recibo_cierre_ref text NOT NULL UNIQUE,
 tipo text NOT NULL CHECK(tipo='bolsa.oferta_ct.intencion_cerrada.v1'),
 payload jsonb NOT NULL CHECK(jsonb_typeof(payload)='object'),
 payload_sha256 text NOT NULL CHECK(payload_sha256 ~ '^[0-9a-f]{64}$'
  AND payload_sha256=encode(sha256(convert_to(payload::text,'UTF8')),'hex')),
 registrada_en timestamptz(6) NOT NULL,
 FOREIGN KEY(organizacion_ref,expediente_ref,clave_ct)
  REFERENCES vec_bolsa_llamamientos.intencion_oferta_ct_terminal_v1(organizacion_ref,expediente_ref,clave_ct)
);
ALTER TABLE vec_bolsa_llamamientos.cierre_oferta_ct_outbox_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.cierre_oferta_ct_outbox_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY solo_propietario ON vec_bolsa_llamamientos.cierre_oferta_ct_outbox_v1
 TO vec_bolsa_llamamientos_propietario
 USING(current_user='vec_bolsa_llamamientos_propietario')
 WITH CHECK(current_user='vec_bolsa_llamamientos_propietario');
REVOKE ALL ON TABLE vec_bolsa_llamamientos.cierre_oferta_ct_outbox_v1 FROM PUBLIC;
REVOKE ALL ON TYPE vec_bolsa_llamamientos.cierre_oferta_ct_outbox_v1 FROM PUBLIC;
CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.cierre_oferta_ct_outbox_v1
 FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_bolsa_llamamientos.cierre_oferta_ct_outbox_v1
 FOR EACH STATEMENT EXECUTE FUNCTION vec_bolsa_llamamientos.rechazar_truncado_origen_oferta_ct_v1();

-- Proyección exacta de las ofertas ya publicadas antes de instalar el
-- arbitraje. No altera ninguna fila ni recibo histórico. El bloqueo anterior
-- impide una inserción entre esta copia y la creación de los triggers.
INSERT INTO vec_bolsa_llamamientos.operacion_oferta_clave_ct_v1(
 bolsa_ref,clave_bolsa,resultado,actor_ref,oferta_ref,recibo_oferta_ref,registrada_en)
SELECT o.bolsa_ref,o.clave_idempotencia,'publicada',o.actor_ref,o.oferta_ref,o.recibo_ref,o.publicada_en
 FROM vec_bolsa_llamamientos.oferta_publicada o;
INSERT INTO vec_bolsa_llamamientos.intencion_oferta_ct_terminal_v1(
 organizacion_ref,expediente_ref,clave_ct,version_expediente,
 bolsa_ref,clave_bolsa,actor_ref,numero_plazas,datos_sha256,origen_sha256,
 resultado,oferta_ref,recibo_ref,decision_ref,auditoria_ref,registrada_en)
SELECT origen.organizacion_ref,origen.expediente_ref,origen.clave_ct,origen.version_expediente,
 origen.bolsa_ref,o.clave_idempotencia,o.actor_ref,origen.numero_plazas,
 encode(sha256(convert_to(o.datos::text,'UTF8')),'hex'),origen.origen_huella_sha256,
 'publicada',origen.oferta_ref,origen.recibo_ref,origen.decision_ref,origen.auditoria_ref,
 origen.publicada_en
 FROM vec_bolsa_llamamientos.origen_oferta_ct_v1 origen
 JOIN vec_bolsa_llamamientos.oferta_publicada o ON o.oferta_ref=origen.oferta_ref;

-- B117 inserta primero la oferta y después su origen. Ambos triggers se
-- ejecutan en esa misma transacción; cualquier conflicto revierte todo.
CREATE FUNCTION vec_bolsa_llamamientos.arbitrar_clave_oferta_ct_v1()
RETURNS trigger LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on AS $f$
BEGIN
 INSERT INTO vec_bolsa_llamamientos.operacion_oferta_clave_ct_v1(
  bolsa_ref,clave_bolsa,resultado,actor_ref,oferta_ref,recibo_oferta_ref,registrada_en)
 VALUES(NEW.bolsa_ref,NEW.clave_idempotencia,'publicada',NEW.actor_ref,
  NEW.oferta_ref,NEW.recibo_ref,date_trunc('microseconds',clock_timestamp()));
 RETURN NEW;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.arbitrar_clave_oferta_ct_v1() FROM PUBLIC;
CREATE TRIGGER arbitrar_clave_oferta_ct_v1 AFTER INSERT ON vec_bolsa_llamamientos.oferta_publicada
 FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.arbitrar_clave_oferta_ct_v1();

CREATE FUNCTION vec_bolsa_llamamientos.arbitrar_intencion_oferta_ct_v1()
RETURNS trigger LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on AS $f$
DECLARE o vec_bolsa_llamamientos.oferta_publicada%ROWTYPE;
BEGIN
 SELECT * INTO STRICT o FROM vec_bolsa_llamamientos.oferta_publicada
 WHERE oferta_ref=NEW.oferta_ref;
 IF o.bolsa_ref IS DISTINCT FROM NEW.bolsa_ref
    OR o.recibo_ref IS DISTINCT FROM NEW.recibo_ref
    OR NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.operacion_oferta_clave_ct_v1 c
      WHERE c.bolsa_ref=o.bolsa_ref AND c.clave_bolsa=o.clave_idempotencia
        AND c.resultado='publicada' AND c.oferta_ref=o.oferta_ref)
 THEN RAISE EXCEPTION 'B121: origen sin publicación íntegra' USING ERRCODE='42501'; END IF;
 INSERT INTO vec_bolsa_llamamientos.intencion_oferta_ct_terminal_v1(
  organizacion_ref,expediente_ref,clave_ct,version_expediente,
  bolsa_ref,clave_bolsa,actor_ref,numero_plazas,datos_sha256,origen_sha256,
  resultado,oferta_ref,recibo_ref,decision_ref,auditoria_ref,registrada_en)
 VALUES(NEW.organizacion_ref,NEW.expediente_ref,NEW.clave_ct,NEW.version_expediente,
  NEW.bolsa_ref,o.clave_idempotencia,o.actor_ref,NEW.numero_plazas,
  encode(sha256(convert_to(o.datos::text,'UTF8')),'hex'),NEW.origen_huella_sha256,
  'publicada',NEW.oferta_ref,NEW.recibo_ref,NEW.decision_ref,NEW.auditoria_ref,
  date_trunc('microseconds',clock_timestamp()));
 RETURN NEW;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.arbitrar_intencion_oferta_ct_v1() FROM PUBLIC;
CREATE TRIGGER arbitrar_intencion_oferta_ct_v1 BEFORE INSERT ON vec_bolsa_llamamientos.origen_oferta_ct_v1
 FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.arbitrar_intencion_oferta_ct_v1();

CREATE FUNCTION vec_bolsa_llamamientos.recuperar_o_cerrar_intencion_oferta_ct_v1(
 p_comando jsonb,p_actor text,p_material text,p_unidad text,p_ambito text,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,
 p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(resultado text,oferta jsonb,cierre jsonb)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='2s' SET statement_timeout='15s' AS $f$
DECLARE m jsonb;d jsonb;o vec_bolsa_llamamientos.oferta_publicada%ROWTYPE;
 origen vec_bolsa_llamamientos.origen_oferta_ct_v1%ROWTYPE;
 k vec_bolsa_llamamientos.operacion_oferta_clave_ct_v1%ROWTYPE;
 t vec_bolsa_llamamientos.intencion_oferta_ct_terminal_v1%ROWTYPE;
 c record;v_origen jsonb;v_bolsa text;v_clave text;v_org text;v_exp text;v_clave_ct text;
 v_version bigint;v_n integer;v_corte timestamptz(6);v_ahora timestamptz(6);
 v_h text;v_ctx text;v_datos_h text;v_origen_h text;v_comando_h text;
 v_recibo text;v_evento text;v_salida jsonb;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR session_user=current_user
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER') IS NOT TRUE
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER') IS TRUE
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR jsonb_typeof(p_comando)<>'object' OR octet_length(p_comando::text)>1048576
    OR p_material IS NULL OR octet_length(p_material) NOT BETWEEN 128 AND 1048576
    OR p_actor !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR p_unidad !~ '^[A-Za-z0-9:_-]+$' OR p_ambito !~ '^[A-Za-z0-9:_-]+$'
 THEN RAISE EXCEPTION 'B121: cierre no autorizable' USING ERRCODE='42501'; END IF;
 BEGIN
  m:=p_material::jsonb;d:=convert_from(p_decision,'UTF8')::jsonb;
  v_origen:=p_comando->'origen_ct';
  v_bolsa:=p_comando->>'bolsa_ref';v_clave:=p_comando->>'clave_bolsa';
  v_org:=p_comando->>'organizacion_ref';v_exp:=p_comando->>'expediente_ref';
  v_clave_ct:=v_origen->>'clave_ct';
  v_version:=(p_comando->>'version_expediente')::bigint;
  v_n:=(p_comando->>'numero_plazas')::integer;
  v_corte:=(v_origen#>>'{lectura,corte}')::timestamptz;
 EXCEPTION WHEN others THEN
  RAISE EXCEPTION 'B121: comando o decisión ilegible' USING ERRCODE='22023';
 END;
 IF m->>'esquema' IS DISTINCT FROM 'vec.bolsa.oferta-ct.cierre-intencion.v1'
    OR m->>'actor_ref' IS DISTINCT FROM p_actor
    OR m->'comando' IS DISTINCT FROM p_comando
    OR m-ARRAY['esquema','actor_ref','comando']<>'{}'::jsonb
    OR p_comando ?& ARRAY['organizacion_ref','expediente_ref','version_expediente',
      'actor_ref','bolsa_ref','necesidad_ref','numero_plazas','numero_elegibles',
      'datos','origen_ct','clave_bolsa','inicio_plazo'] IS NOT TRUE
    OR p_comando-ARRAY['organizacion_ref','expediente_ref','version_expediente',
      'actor_ref','bolsa_ref','necesidad_ref','numero_plazas','numero_elegibles',
      'datos','origen_ct','clave_bolsa','inicio_plazo']<>'{}'::jsonb
    OR p_comando->>'actor_ref' IS DISTINCT FROM p_actor
    OR p_comando->>'inicio_plazo' IS DISTINCT FROM 'publicacion'
    OR p_comando->>'numero_plazas' !~ '^([2-9]|[1-9][0-9]|100)$'
    OR v_bolsa !~ '^bolsa:[A-Za-z0-9:_-]{1,250}$'
    OR v_clave !~ '^ct-oferta:[0-9a-f]{64}$'
    OR v_org !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
    OR v_exp !~ '^expediente:[A-Za-z0-9._:/#-]{2,149}$'
    OR v_clave_ct !~ '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
    OR v_version NOT BETWEEN 1 AND 9007199254740991
    OR v_n NOT BETWEEN 2 AND 100
    OR jsonb_typeof(p_comando->'datos')<>'object'
    OR jsonb_typeof(v_origen)<>'object'
    OR (SELECT count(*) FROM jsonb_object_keys(v_origen))<>18
    OR NOT (v_origen ?& ARRAY['organizacion_ref','unidad_ref','expediente_ref',
      'version_expediente','clave_ct','necesidad_ref','acta_ref','categoria_ref',
      'bolsa_ref','version_bolsa','huella_bolsa_sha256','instantanea_ref',
      'version_instantanea','huella_instantanea_sha256','politica_ref',
      'version_politica','huella_politica_sha256','lectura'])
    OR jsonb_typeof(v_origen->'lectura')<>'object'
    OR jsonb_typeof(v_origen#>'{lectura,entradas}')<>'array'
    OR coalesce(v_origen#>>'{lectura,corte}','') !~
      '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{6}Z$'
    OR v_origen->>'organizacion_ref' IS DISTINCT FROM v_org
    OR v_origen->>'expediente_ref' IS DISTINCT FROM v_exp
    OR v_origen->>'version_expediente' IS DISTINCT FROM v_version::text
    OR v_origen->>'bolsa_ref' IS DISTINCT FROM v_bolsa
    OR v_origen->>'necesidad_ref' IS DISTINCT FROM p_comando->>'necesidad_ref'
    OR v_origen->>'unidad_ref' IS DISTINCT FROM p_unidad
    OR v_corte IS NULL OR v_corte<>date_trunc('microseconds',v_corte)
    OR v_clave IS DISTINCT FROM ('ct-oferta:'||encode(sha256(convert_to(
      v_org||chr(31)||v_exp||chr(31)||v_version::text||chr(31)||v_clave_ct,'UTF8')),'hex'))
 THEN RAISE EXCEPTION 'B121: comando congelado divergente' USING ERRCODE='22023'; END IF;
 v_h:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 v_ctx:=encode(sha256(convert_to('{"ambitos":{"ambito_ref":"'||p_ambito||
  '","unidad_ref":"'||p_unidad||'"},"atributos":{"material_sha256":"'||v_h||'"}}','UTF8')),'hex');
 IF d->>'principal_id' IS DISTINCT FROM p_actor
    OR d->>'accion' IS DISTINCT FROM 'bolsa.oferta.intencion.cerrar'
    OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'bolsa_constituida'
    OR d->>'finalidad' IS DISTINCT FROM 'gestion_llamamientos_bolsa'
    OR d->>'recurso_ref' IS DISTINCT FROM v_bolsa
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_ctx
    OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'B121: decisión de cierre divergente' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT c FROM vec_autorizacion_atestada_v3.registrar_y_consumir_cierre_intencion_oferta_ct_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 IF c.efecto_ref IS DISTINCT FROM v_bolsa OR c.huella_efecto_sha256 IS DISTINCT FROM v_ctx
    OR c.consumo_nuevo IS NOT TRUE THEN
  RAISE EXCEPTION 'B121: consumo V3 divergente' USING ERRCODE='42501'; END IF;
 -- B117 publica advisory→V3 y recupera V3→advisory. Un intento no bloqueante
 -- evita el ciclo de esperas con ambas rutas. La PK física decide la carrera;
 -- quien no obtiene el lock aborta todo, incluido el consumo, y debe iniciar
 -- otra petición autorizada con la misma intención congelada.
 IF pg_try_advisory_xact_lock(hashtextextended(
   'vec_bolsa_llamamientos:oferta:'||v_bolsa||':'||v_clave,0)) IS NOT TRUE THEN
  RAISE EXCEPTION 'B121: operación concurrente pendiente' USING ERRCODE='40001'; END IF;
 SELECT * INTO o FROM vec_bolsa_llamamientos.oferta_publicada x
 WHERE x.bolsa_ref=v_bolsa AND x.clave_idempotencia=v_clave;
 SELECT * INTO origen FROM vec_bolsa_llamamientos.origen_oferta_ct_v1 x
 WHERE x.organizacion_ref=v_org AND x.expediente_ref=v_exp AND x.clave_ct=v_clave_ct;
 SELECT * INTO k FROM vec_bolsa_llamamientos.operacion_oferta_clave_ct_v1 x
 WHERE x.bolsa_ref=v_bolsa AND x.clave_bolsa=v_clave;
 SELECT * INTO t FROM vec_bolsa_llamamientos.intencion_oferta_ct_terminal_v1 x
 WHERE x.organizacion_ref=v_org AND x.expediente_ref=v_exp AND x.clave_ct=v_clave_ct;
 v_datos_h:=encode(sha256(convert_to((p_comando->'datos')::text,'UTF8')),'hex');
 v_origen_h:=encode(sha256(convert_to(v_origen::text,'UTF8')),'hex');
 v_comando_h:=encode(sha256(convert_to(p_comando::text,'UTF8')),'hex');
 IF o.oferta_ref IS NOT NULL THEN
  IF origen.oferta_ref IS DISTINCT FROM o.oferta_ref
     OR origen.origen_canonico IS DISTINCT FROM v_origen
     OR origen.numero_plazas IS DISTINCT FROM v_n
     OR o.actor_ref IS DISTINCT FROM p_actor
     OR o.datos IS DISTINCT FROM p_comando->'datos'
     OR k.resultado IS DISTINCT FROM 'publicada' OR k.oferta_ref IS DISTINCT FROM o.oferta_ref
     OR k.recibo_oferta_ref IS DISTINCT FROM o.recibo_ref
     OR t.resultado IS DISTINCT FROM 'publicada' OR t.oferta_ref IS DISTINCT FROM o.oferta_ref
     OR t.version_expediente IS DISTINCT FROM v_version OR t.recibo_ref IS DISTINCT FROM o.recibo_ref
     OR t.actor_ref IS DISTINCT FROM p_actor OR t.numero_plazas IS DISTINCT FROM v_n
     OR t.datos_sha256 IS DISTINCT FROM v_datos_h OR t.origen_sha256 IS DISTINCT FROM v_origen_h
     OR t.bolsa_ref IS DISTINCT FROM v_bolsa OR t.clave_bolsa IS DISTINCT FROM v_clave
  THEN RAISE EXCEPTION 'B121: publicación ajena o incompleta' USING ERRCODE='VBO01'; END IF;
  RETURN QUERY SELECT 'publicada',vec_bolsa_llamamientos.proyectar_oferta_v2(o.oferta_ref,clock_timestamp()),NULL::jsonb;
  RETURN;
 END IF;
 IF origen.oferta_ref IS NOT NULL THEN
  RAISE EXCEPTION 'B121: origen sin oferta' USING ERRCODE='55000'; END IF;
 IF k.resultado='cerrada_sin_publicar' OR t.resultado='cerrada_sin_publicar' THEN
  IF k.resultado IS DISTINCT FROM 'cerrada_sin_publicar'
     OR t.resultado IS DISTINCT FROM 'cerrada_sin_publicar'
     OR k.actor_ref IS DISTINCT FROM p_actor
     OR k.recibo_cierre_ref IS DISTINCT FROM t.recibo_ref
     OR t.bolsa_ref IS DISTINCT FROM v_bolsa OR t.clave_bolsa IS DISTINCT FROM v_clave
     OR t.actor_ref IS DISTINCT FROM p_actor OR t.version_expediente IS DISTINCT FROM v_version
     OR t.numero_plazas IS DISTINCT FROM v_n OR t.datos_sha256 IS DISTINCT FROM v_datos_h
     OR t.origen_sha256 IS DISTINCT FROM v_origen_h OR t.comando_sha256 IS DISTINCT FROM v_comando_h
     OR NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.cierre_oferta_ct_outbox_v1 ev
       WHERE ev.evento_ref=t.evento_ref AND ev.recibo_cierre_ref=t.recibo_ref
         AND ev.payload=t.recibo_json)
  THEN RAISE EXCEPTION 'B121: cierre previo divergente' USING ERRCODE='VBO01'; END IF;
  RETURN QUERY SELECT 'cerrada',NULL::jsonb,t.recibo_json;
  RETURN;
 END IF;
 IF k.bolsa_ref IS NOT NULL OR t.organizacion_ref IS NOT NULL THEN
  RAISE EXCEPTION 'B121: operación parcial' USING ERRCODE='VBO01'; END IF;
 v_ahora:=date_trunc('microseconds',clock_timestamp());
 IF v_ahora<=v_corte+interval '5 minutes' THEN
  RAISE EXCEPTION 'B121: captura aún vigente' USING ERRCODE='55000'; END IF;
 v_recibo:='recibo:bolsa:oferta-ct-cierre:'||gen_random_uuid()::text;
 v_evento:='evento:bolsa:oferta-ct-cierre:'||gen_random_uuid()::text;
 v_salida:=jsonb_build_object(
  'esquema','vec.bolsa.oferta-ct.cierre.v1','estado','cerrada_sin_publicar',
  'organizacion_ref',v_org,'expediente_ref',v_exp,'version_expediente',v_version,
  'clave_ct',v_clave_ct,'actor_ref',p_actor,'bolsa_ref',v_bolsa,'clave_bolsa',v_clave,
  'numero_plazas',v_n,'datos_sha256',v_datos_h,'origen_sha256',v_origen_h,
  'comando_sha256',v_comando_h,'recibo_cierre_ref',v_recibo,
  'decision_ref',c.decision_ref,'auditoria_ref',c.auditoria_ref,
  'evento_ref',v_evento,'cerrada_en',to_char(v_ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'));
 INSERT INTO vec_bolsa_llamamientos.operacion_oferta_clave_ct_v1(
  bolsa_ref,clave_bolsa,resultado,actor_ref,recibo_cierre_ref,registrada_en)
 VALUES(v_bolsa,v_clave,'cerrada_sin_publicar',p_actor,v_recibo,v_ahora);
 INSERT INTO vec_bolsa_llamamientos.intencion_oferta_ct_terminal_v1(
  organizacion_ref,expediente_ref,clave_ct,version_expediente,
  bolsa_ref,clave_bolsa,actor_ref,numero_plazas,datos_sha256,origen_sha256,
  comando_sha256,resultado,recibo_ref,decision_ref,auditoria_ref,
  evento_ref,recibo_json,registrada_en)
 VALUES(v_org,v_exp,v_clave_ct,v_version,v_bolsa,v_clave,p_actor,v_n,v_datos_h,v_origen_h,
  v_comando_h,'cerrada_sin_publicar',v_recibo,c.decision_ref,c.auditoria_ref,
  v_evento,v_salida,v_ahora);
 INSERT INTO vec_bolsa_llamamientos.cierre_oferta_ct_outbox_v1(
  evento_ref,organizacion_ref,expediente_ref,clave_ct,recibo_cierre_ref,
  tipo,payload,payload_sha256,registrada_en)
 VALUES(v_evento,v_org,v_exp,v_clave_ct,v_recibo,'bolsa.oferta_ct.intencion_cerrada.v1',
  v_salida,encode(sha256(convert_to(v_salida::text,'UTF8')),'hex'),v_ahora);
 RETURN QUERY SELECT 'cerrada',NULL::jsonb,v_salida;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.recuperar_o_cerrar_intencion_oferta_ct_v1(
 jsonb,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.recuperar_o_cerrar_intencion_oferta_ct_v1(
 jsonb,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_ejecutor;

-- CT solo recibe un booleano nominal sobre su cierre, sin SELECT a tablas Bolsa.
CREATE FUNCTION vec_bolsa_llamamientos.verificar_cierre_intencion_oferta_ct_v1(
 p_comando jsonb,p_recibo jsonb)
RETURNS boolean LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC' AS $f$
DECLARE k vec_bolsa_llamamientos.operacion_oferta_clave_ct_v1%ROWTYPE;
 t vec_bolsa_llamamientos.intencion_oferta_ct_terminal_v1%ROWTYPE;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR session_user=current_user
    OR pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER') IS NOT TRUE
    OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER') IS TRUE
    OR jsonb_typeof(p_comando)<>'object' OR jsonb_typeof(p_recibo)<>'object'
    OR octet_length(p_comando::text)>1048576 OR octet_length(p_recibo::text)>65536
 THEN RAISE EXCEPTION 'B121: verificación CT denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO k FROM vec_bolsa_llamamientos.operacion_oferta_clave_ct_v1
 WHERE bolsa_ref=p_comando->>'bolsa_ref' AND clave_bolsa=p_comando->>'clave_bolsa';
 SELECT * INTO t FROM vec_bolsa_llamamientos.intencion_oferta_ct_terminal_v1
 WHERE organizacion_ref=p_comando->>'organizacion_ref'
   AND expediente_ref=p_comando->>'expediente_ref'
   AND clave_ct=p_comando#>>'{origen_ct,clave_ct}';
 RETURN coalesce(k.resultado='cerrada_sin_publicar' AND t.resultado='cerrada_sin_publicar'
  AND k.recibo_cierre_ref=t.recibo_ref AND k.actor_ref=p_comando->>'actor_ref'
  AND t.recibo_json=p_recibo AND t.recibo_ref=p_recibo->>'recibo_cierre_ref'
  AND t.version_expediente=(p_comando->>'version_expediente')::bigint
  AND t.bolsa_ref=p_comando->>'bolsa_ref' AND t.clave_bolsa=p_comando->>'clave_bolsa'
  AND t.actor_ref=p_comando->>'actor_ref'
  AND t.numero_plazas=(p_comando->>'numero_plazas')::integer
  AND t.datos_sha256=encode(sha256(convert_to((p_comando->'datos')::text,'UTF8')),'hex')
  AND t.origen_sha256=encode(sha256(convert_to((p_comando->'origen_ct')::text,'UTF8')),'hex')
  AND t.comando_sha256=encode(sha256(convert_to(p_comando::text,'UTF8')),'hex')
  AND NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.oferta_publicada o
   WHERE o.bolsa_ref=t.bolsa_ref AND o.clave_idempotencia=t.clave_bolsa)
  AND EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.cierre_oferta_ct_outbox_v1 ev
   WHERE ev.evento_ref=t.evento_ref AND ev.recibo_cierre_ref=t.recibo_ref
     AND ev.payload=t.recibo_json),false);
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.verificar_cierre_intencion_oferta_ct_v1(jsonb,jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.verificar_cierre_intencion_oferta_ct_v1(jsonb,jsonb)
 TO vec_contratacion_temporal_propietario;
COMMIT;

