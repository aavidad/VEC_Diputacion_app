\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000057',0));

-- B57: causas administrativas versionadas de Bolsa. La etiqueta publicable
-- es una clasificación aprobada, nunca el motivo libre de B12/B16.
DO $pre$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR to_regclass('vec_bolsa_llamamientos.situacion_participacion') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.datos_contacto_participacion') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.traza_valor_participacion') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.registrar_intento_borrador_llamamiento_v1(text,text,text,text,text)') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.consultar_auditoria_participacion_v1(text,text,timestamptz,timestamptz,timestamptz,text,text,integer,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.registrar_operacion_situacion_participacion_v1(text,text,text,timestamptz,timestamptz,text,text,text,text,timestamptz,text,text,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR strpos((SELECT prosrc FROM pg_proc WHERE oid='vec_bolsa_llamamientos.consultar_auditoria_participacion_v1(text,text,timestamptz,timestamptz,timestamptz,text,text,integer,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure),
              $$s.recibo_ref='recibo:situacion:constitucion:' || s.participacion_ref$$)=0
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_catalogo_causas_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_catalogo_causas_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.existe_historia_catalogo_causas_bolsa_v1()') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_propuesta_causas_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_consulta_propuesta_causas_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.causa_participacion_catalogo') IS NOT NULL
 THEN RAISE EXCEPTION 'B57: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

DO $acl_b56$
DECLARE f regprocedure:='vec_bolsa_llamamientos.consultar_auditoria_participacion_v1(text,text,timestamptz,timestamptz,timestamptz,text,text,integer,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f
    AND p.proowner='vec_bolsa_llamamientos_propietario'::regrole
    AND p.prosecdef AND p.proconfig=ARRAY['search_path=pg_catalog','lock_timeout=2s']
    AND EXISTS(SELECT 1 FROM aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
               WHERE a.grantee='vec_bolsa_llamamientos_ejecutor'::regrole AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
    AND NOT EXISTS(SELECT 1 FROM aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
               WHERE a.grantee NOT IN(p.proowner,'vec_bolsa_llamamientos_ejecutor'::regrole)
                  OR a.privilege_type<>'EXECUTE'))
 THEN RAISE EXCEPTION 'B57: función/ACL B56 incompatibles' USING ERRCODE='55000'; END IF;
END $acl_b56$;

-- Extensión B57 de la bitácora B17 instalada, sin reescribir sus filas.
DO $bitacora_b57$
DECLARE v_accion text; v_ruta text;
BEGIN
 SELECT pg_get_constraintdef(oid,true) INTO STRICT v_accion FROM pg_constraint
 WHERE conrelid='vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento'::regclass
   AND conname='bitacora_intento_borrador_llamamiento_accion_check' AND contype='c';
 SELECT pg_get_constraintdef(oid,true) INTO STRICT v_ruta FROM pg_constraint
 WHERE conrelid='vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento'::regclass
   AND conname='bitacora_intento_borrador_llamamiento_ruta_clase_check' AND contype='c';
 IF strpos(v_accion,'''recuperar_llamamiento''')=0 OR strpos(v_ruta,'''emisiones''')=0
    OR strpos(v_accion,'''proponer_causa_participacion''')<>0
    OR strpos(v_ruta,'''propuestas_causas''')<>0
 THEN RAISE EXCEPTION 'B57: bitácora anterior incompatible' USING ERRCODE='55000'; END IF;
 ALTER TABLE vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento
  DROP CONSTRAINT bitacora_intento_borrador_llamamiento_accion_check;
 ALTER TABLE vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento
  ADD CONSTRAINT bitacora_intento_borrador_llamamiento_accion_check CHECK (accion IN ('crear','consultar','cambiar_situacion','registrar_contacto','consultar_datos_contacto','registrar_datos_contacto','emitir_llamamiento','recuperar_llamamiento','consultar_causas_participacion','proponer_causa_participacion','consultar_propuesta_causa_participacion','publicar_causa_participacion'));
 ALTER TABLE vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento
  DROP CONSTRAINT bitacora_intento_borrador_llamamiento_ruta_clase_check;
 ALTER TABLE vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento
  ADD CONSTRAINT bitacora_intento_borrador_llamamiento_ruta_clase_check CHECK (ruta_clase IN ('coleccion','detalle','situacion','contactos','datos_contacto','emisiones','catalogo_causas','propuestas_causas','propuesta_causas','publicacion_causas'));
END $bitacora_b57$;
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.registrar_intento_borrador_llamamiento_v1(p_correlacion_ref text,p_accion text,p_ruta_clase text,p_actor_ref text,p_resultado text) RETURNS void LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE v_ref text; BEGIN
 PERFORM vec_bolsa_llamamientos.exigir_runtime_registrador_frontera_borrador_llamamiento();
	 IF p_correlacion_ref IS NULL OR p_correlacion_ref !~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{7,191}$' OR p_accion NOT IN('crear','consultar','cambiar_situacion','registrar_contacto','consultar_datos_contacto','registrar_datos_contacto','emitir_llamamiento','recuperar_llamamiento','consultar_causas_participacion','proponer_causa_participacion','consultar_propuesta_causa_participacion','publicar_causa_participacion') OR p_ruta_clase NOT IN('coleccion','detalle','situacion','contactos','datos_contacto','emisiones','catalogo_causas','propuestas_causas','propuesta_causas','publicacion_causas') OR (p_actor_ref IS NOT NULL AND p_actor_ref !~ '^per_[A-Za-z0-9_-]{22,128}$') OR p_resultado NOT IN('autenticacion_requerida','acceso_denegado','recurso_no_disponible','infraestructura_no_disponible','resultado_indeterminado','correcto') THEN RAISE EXCEPTION 'intento de frontera Bolsa inválido' USING ERRCODE='22023'; END IF;
 v_ref:='intento:'||translate(encode(sha256(convert_to(p_correlacion_ref||':'||p_accion||':'||p_ruta_clase||':'||coalesce(p_actor_ref,'')||':'||p_resultado||':'||clock_timestamp()::text,'UTF8')),'hex'),'0123456789','ghijklmnop');
 INSERT INTO vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento(intento_ref,correlacion_ref,accion,ruta_clase,actor_ref,resultado,registrada_en) VALUES(v_ref,p_correlacion_ref,p_accion,p_ruta_clase,p_actor_ref,p_resultado,clock_timestamp());
END $f$;

CREATE FUNCTION vec_bolsa_llamamientos.huella_causa_participacion_v1(
 p_codigo text,p_version bigint,p_etiqueta text,p_situacion boolean,p_contacto boolean,p_publicable boolean,p_activa boolean)
RETURNS text LANGUAGE sql IMMUTABLE STRICT PARALLEL SAFE SET search_path=pg_catalog AS $f$
 SELECT encode(sha256(convert_to(array_to_string(ARRAY[
 'bolsa.causa_participacion.v1',p_codigo,p_version::text,p_etiqueta,
 p_situacion::text,p_contacto::text,p_publicable::text,p_activa::text],E'\n'),'UTF8')),'hex')
$f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.huella_causa_participacion_v1(text,bigint,text,boolean,boolean,boolean,boolean) FROM PUBLIC;

CREATE TABLE vec_bolsa_llamamientos.propuesta_causa_participacion (
 propuesta_ref text PRIMARY KEY CHECK (propuesta_ref ~ '^propuesta:causa:[0-9a-f]{64}$'),
 codigo text NOT NULL CHECK (codigo ~ '^[a-z][a-z0-9_]{2,63}$'),
 version bigint NOT NULL CHECK (version>=1),
 etiqueta text NOT NULL CHECK (octet_length(etiqueta) BETWEEN 1 AND 120 AND etiqueta=btrim(etiqueta) AND etiqueta !~ '[[:cntrl:]]'),
 aplica_situacion boolean NOT NULL,
 aplica_contacto boolean NOT NULL,
 publicable boolean NOT NULL,
 activa boolean NOT NULL,
 huella_sha256 text NOT NULL CHECK (huella_sha256=vec_bolsa_llamamientos.huella_causa_participacion_v1(codigo,version,etiqueta,aplica_situacion,aplica_contacto,publicable,activa)),
 propuesta_por text NOT NULL CHECK (propuesta_por ~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{7,255}$'),
 propuesta_persona_ref text NOT NULL CHECK (propuesta_persona_ref ~ '^per_[A-Za-z0-9_-]{22,128}$'),
 recibo_ref text NOT NULL UNIQUE CHECK (recibo_ref ~ '^recibo:propuesta:causa:[0-9a-f]{64}$'),
 propuesta_en timestamptz(6) NOT NULL,
 CHECK (aplica_situacion OR aplica_contacto)
);
ALTER TABLE vec_bolsa_llamamientos.propuesta_causa_participacion ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.propuesta_causa_participacion FORCE ROW LEVEL SECURITY;
CREATE POLICY propuesta_causa_participacion_propietario ON vec_bolsa_llamamientos.propuesta_causa_participacion
 TO vec_bolsa_llamamientos_propietario USING (current_user='vec_bolsa_llamamientos_propietario') WITH CHECK (current_user='vec_bolsa_llamamientos_propietario');
REVOKE ALL ON vec_bolsa_llamamientos.propuesta_causa_participacion FROM PUBLIC;
CREATE TRIGGER propuesta_causa_participacion_inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.propuesta_causa_participacion
 FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();

CREATE TABLE vec_bolsa_llamamientos.causa_participacion_catalogo (
 codigo text NOT NULL CHECK (codigo ~ '^[a-z][a-z0-9_]{2,63}$'),
 version bigint NOT NULL CHECK (version>=1),
 etiqueta text NOT NULL CHECK (octet_length(etiqueta) BETWEEN 1 AND 120 AND etiqueta=btrim(etiqueta) AND etiqueta !~ '[[:cntrl:]]'),
 aplica_situacion boolean NOT NULL,
 aplica_contacto boolean NOT NULL,
 publicable boolean NOT NULL,
 activa boolean NOT NULL,
 huella_sha256 text NOT NULL CHECK (huella_sha256=vec_bolsa_llamamientos.huella_causa_participacion_v1(codigo,version,etiqueta,aplica_situacion,aplica_contacto,publicable,activa)),
 actor_ref text NOT NULL,
    recibo_ref text NOT NULL UNIQUE CHECK (octet_length(recibo_ref) BETWEEN 1 AND 256),
    propuesta_ref text UNIQUE REFERENCES vec_bolsa_llamamientos.propuesta_causa_participacion(propuesta_ref),
    publicada_persona_ref text CHECK (publicada_persona_ref IS NULL OR publicada_persona_ref ~ '^per_[A-Za-z0-9_-]{22,128}$'),
    publicada_en timestamptz(6) NOT NULL,
    PRIMARY KEY(codigo,version),
    CHECK (aplica_situacion OR aplica_contacto),
    CHECK (actor_ref='sistema:migracion:bolsa57' OR (propuesta_ref IS NOT NULL AND publicada_persona_ref IS NOT NULL))
);
ALTER TABLE vec_bolsa_llamamientos.causa_participacion_catalogo ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.causa_participacion_catalogo FORCE ROW LEVEL SECURITY;
CREATE POLICY causa_participacion_catalogo_propietario ON vec_bolsa_llamamientos.causa_participacion_catalogo
 TO vec_bolsa_llamamientos_propietario USING (current_user='vec_bolsa_llamamientos_propietario') WITH CHECK (current_user='vec_bolsa_llamamientos_propietario');
REVOKE ALL ON vec_bolsa_llamamientos.causa_participacion_catalogo FROM PUBLIC;
CREATE TRIGGER causa_participacion_catalogo_inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.causa_participacion_catalogo
 FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();

-- La versión registrada conserva su etiqueta histórica, pero una versión
-- vigente que retire la publicación oculta también lecturas posteriores de
-- recibos antiguos. La historia y su huella permanecen intactas.
CREATE FUNCTION vec_bolsa_llamamientos.etiqueta_causa_participacion_publicable_v1(
 p_codigo text,p_version bigint,p_huella_sha256 text)
RETURNS text LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT c.etiqueta FROM vec_bolsa_llamamientos.causa_participacion_catalogo c
 JOIN LATERAL (
   SELECT v.publicable,v.activa FROM vec_bolsa_llamamientos.causa_participacion_catalogo v
   WHERE v.codigo=c.codigo ORDER BY v.version DESC LIMIT 1
 ) vigente ON true
 WHERE c.codigo=p_codigo AND c.version=p_version AND c.huella_sha256=p_huella_sha256
   AND c.publicable AND vigente.publicable AND vigente.activa
$f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.etiqueta_causa_participacion_publicable_v1(text,bigint,text) FROM PUBLIC;

-- Defectos sintéticos de clasificación administrativa; no representan una
-- regla jurídica ni atribuyen causa a las filas libres históricas.
INSERT INTO vec_bolsa_llamamientos.causa_participacion_catalogo
 (codigo,version,etiqueta,aplica_situacion,aplica_contacto,publicable,activa,huella_sha256,actor_ref,recibo_ref,publicada_en)
SELECT v.codigo,1,v.etiqueta,v.situacion,v.contacto,true,true,
 vec_bolsa_llamamientos.huella_causa_participacion_v1(v.codigo,1,v.etiqueta,v.situacion,v.contacto,true,true),
 'sistema:migracion:bolsa57','recibo:causa:bolsa57:'||v.codigo,clock_timestamp()
FROM (VALUES ('gestion_situacion','Cambio de situación registrado',true,false),
             ('actualizacion_contacto','Datos de contacto actualizados',false,true)) v(codigo,etiqueta,situacion,contacto);

CREATE FUNCTION vec_bolsa_llamamientos.proponer_causa_participacion_v1(
 p_codigo text,p_version bigint,p_etiqueta text,p_aplica_situacion boolean,p_aplica_contacto boolean,
 p_publicable boolean,p_activa boolean,p_huella_sha256 text,p_actor text,p_propuesta_ref text,p_recibo_ref text,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,
 p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(propuesta_ref text,codigo text,version bigint,huella_sha256 text,recibo_ref text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE v_capacidad jsonb; v_decision jsonb; v_contexto jsonb; v_persona_ref text; v_consumo record; v_previa record; v_ultima bigint;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_migrador','MEMBER')
    OR current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
    OR current_setting('TimeZone')<>'UTC'
    OR p_codigo IS NULL OR p_codigo !~ '^[a-z][a-z0-9_]{2,63}$' OR p_version IS NULL OR p_version<1
    OR p_etiqueta IS NULL OR p_etiqueta<>btrim(p_etiqueta) OR octet_length(p_etiqueta) NOT BETWEEN 1 AND 120 OR p_etiqueta ~ '[[:cntrl:]]'
    OR p_aplica_situacion IS NULL OR p_aplica_contacto IS NULL OR NOT (p_aplica_situacion OR p_aplica_contacto)
    OR p_publicable IS NULL OR p_activa IS NULL
    OR p_huella_sha256 IS DISTINCT FROM vec_bolsa_llamamientos.huella_causa_participacion_v1(p_codigo,p_version,p_etiqueta,p_aplica_situacion,p_aplica_contacto,p_publicable,p_activa)
    OR p_actor IS NULL OR p_actor !~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{7,255}$'
    OR p_propuesta_ref IS DISTINCT FROM 'propuesta:causa:'||encode(sha256(convert_to(
      'bolsa.causa_participacion.propuesta.v1'||E'\n'||p_huella_sha256||E'\n'||p_actor,'UTF8')),'hex')
    OR p_recibo_ref IS DISTINCT FROM 'recibo:propuesta:causa:'||encode(sha256(convert_to(
      'bolsa.causa_participacion.propuesta.recibo.v1'||E'\n'||p_propuesta_ref,'UTF8')),'hex')
 THEN RAISE EXCEPTION 'B57: propuesta denegada' USING ERRCODE='42501'; END IF;
 BEGIN
  v_capacidad:=convert_from(p_capacidad,'UTF8')::jsonb;
  v_decision:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'B57: propuesta denegada' USING ERRCODE='42501'; END;
 IF v_capacidad->>'audiencia_consumo' IS DISTINCT FROM 'vec_bolsa_llamamientos.causas_participacion.proponer.v1'
    OR v_capacidad->>'operacion' IS DISTINCT FROM 'bolsa.causas_participacion.proponer'
    OR v_capacidad->>'efecto_ref' IS DISTINCT FROM 'vec.bolsa.causas_participacion'
    OR v_decision->>'principal_id' IS DISTINCT FROM p_actor
    OR v_decision->>'accion' IS DISTINCT FROM 'bolsa.causas_participacion.proponer'
    OR v_decision->>'modulo_id' IS DISTINCT FROM 'bolsa'
    OR v_decision->>'tipo_recurso' IS DISTINCT FROM 'catalogo_causas_participacion'
    OR v_decision->>'finalidad' IS DISTINCT FROM 'preparar_catalogo_causas_participacion'
    OR v_decision->>'recurso_ref' IS DISTINCT FROM 'vec.bolsa.causas_participacion'
    OR v_decision->>'contexto_recurso_huella_sha256' IS DISTINCT FROM encode(sha256(convert_to(
      '{"ambitos":{},"atributos":{"causa_sha256":"'||p_huella_sha256||'","propuesta_ref":"'||p_propuesta_ref||'","recibo_ref":"'||p_recibo_ref||'"}}','UTF8')),'hex')
    OR v_decision->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_capacidad->>'huella_efecto_sha256'
    OR v_decision->'campos_permitidos' IS DISTINCT FROM '["borrador","recibo"]'::jsonb
    OR v_decision->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'B57: propuesta denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.consumir_propuesta_causas_bolsa_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF v_consumo.consumo_nuevo IS NOT TRUE OR v_consumo.efecto_ref IS DISTINCT FROM 'vec.bolsa.causas_participacion'
    OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM v_decision->>'contexto_recurso_huella_sha256'
 THEN RAISE EXCEPTION 'B57: propuesta denegada' USING ERRCODE='42501'; END IF;
 BEGIN v_contexto:=convert_from(p_contexto,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'B57: contexto de proponente inválido' USING ERRCODE='42501'; END;
 v_persona_ref:=v_contexto->>'persona_ref';
 IF v_contexto->>'esquema' IS DISTINCT FROM 'vec.contexto-actor.vinculado.v2'
    OR v_contexto->>'principal_ref' IS DISTINCT FROM p_actor
    OR v_persona_ref IS NULL OR v_persona_ref !~ '^per_[A-Za-z0-9_-]{22,128}$'
 THEN RAISE EXCEPTION 'B57: contexto de proponente inválido' USING ERRCODE='42501'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:causas_participacion',0));
 SELECT * INTO v_previa FROM vec_bolsa_llamamientos.propuesta_causa_participacion p WHERE p.propuesta_ref=p_propuesta_ref;
 IF FOUND THEN
  IF v_previa.codigo IS DISTINCT FROM p_codigo OR v_previa.version IS DISTINCT FROM p_version
     OR v_previa.etiqueta IS DISTINCT FROM p_etiqueta OR v_previa.aplica_situacion IS DISTINCT FROM p_aplica_situacion
     OR v_previa.aplica_contacto IS DISTINCT FROM p_aplica_contacto OR v_previa.publicable IS DISTINCT FROM p_publicable
     OR v_previa.activa IS DISTINCT FROM p_activa OR v_previa.huella_sha256 IS DISTINCT FROM p_huella_sha256
     OR v_previa.propuesta_por IS DISTINCT FROM p_actor OR v_previa.propuesta_persona_ref IS DISTINCT FROM v_persona_ref
     OR v_previa.recibo_ref IS DISTINCT FROM p_recibo_ref
  THEN RAISE EXCEPTION 'B57: propuesta divergente' USING ERRCODE='VBS01'; END IF;
  RETURN QUERY SELECT v_previa.propuesta_ref,v_previa.codigo,v_previa.version,v_previa.huella_sha256,v_previa.recibo_ref;
  RETURN;
 END IF;
 SELECT coalesce(max(c.version),0) INTO v_ultima FROM vec_bolsa_llamamientos.causa_participacion_catalogo c WHERE c.codigo=p_codigo;
 IF p_version<>v_ultima+1 THEN RAISE EXCEPTION 'B57: versión propuesta en conflicto' USING ERRCODE='VBS57'; END IF;
 INSERT INTO vec_bolsa_llamamientos.propuesta_causa_participacion
  (propuesta_ref,codigo,version,etiqueta,aplica_situacion,aplica_contacto,publicable,activa,huella_sha256,propuesta_por,propuesta_persona_ref,recibo_ref,propuesta_en)
 VALUES(p_propuesta_ref,p_codigo,p_version,p_etiqueta,p_aplica_situacion,p_aplica_contacto,p_publicable,p_activa,p_huella_sha256,p_actor,v_persona_ref,p_recibo_ref,clock_timestamp());
 RETURN QUERY SELECT p_propuesta_ref,p_codigo,p_version,p_huella_sha256,p_recibo_ref;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.proponer_causa_participacion_v1(text,bigint,text,boolean,boolean,boolean,boolean,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.proponer_causa_participacion_v1(text,bigint,text,boolean,boolean,boolean,boolean,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_ejecutor;

CREATE FUNCTION vec_bolsa_llamamientos.consultar_propuesta_causa_participacion_v1(
 p_propuesta_ref text,p_actor text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(propuesta_ref text,codigo text,version bigint,huella_sha256 text,etiqueta text,
 aplica_situacion boolean,aplica_contacto boolean,publicable boolean,activa boolean,recibo_ref text,estado text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE v_capacidad jsonb; v_decision jsonb; v_consumo record;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_migrador','MEMBER')
    OR current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
    OR current_setting('TimeZone')<>'UTC'
    OR p_propuesta_ref IS NULL OR p_propuesta_ref !~ '^propuesta:causa:[0-9a-f]{64}$'
    OR p_actor IS NULL OR p_actor !~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{7,255}$'
 THEN RAISE EXCEPTION 'B57: consulta de propuesta denegada' USING ERRCODE='42501'; END IF;
 BEGIN
  v_capacidad:=convert_from(p_capacidad,'UTF8')::jsonb;
  v_decision:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'B57: consulta de propuesta denegada' USING ERRCODE='42501'; END;
 IF v_capacidad->>'audiencia_consumo' IS DISTINCT FROM 'vec_bolsa_llamamientos.causas_participacion.propuesta.consultar.v1'
    OR v_capacidad->>'operacion' IS DISTINCT FROM 'bolsa.causas_participacion.propuesta.consultar'
    OR v_capacidad->>'efecto_ref' IS DISTINCT FROM p_propuesta_ref
    OR v_decision->>'principal_id' IS DISTINCT FROM p_actor
    OR v_decision->>'accion' IS DISTINCT FROM 'bolsa.causas_participacion.propuesta.consultar'
    OR v_decision->>'modulo_id' IS DISTINCT FROM 'bolsa'
    OR v_decision->>'tipo_recurso' IS DISTINCT FROM 'propuesta_catalogo_causas_participacion'
    OR v_decision->>'finalidad' IS DISTINCT FROM 'revisar_propuesta_catalogo_causas_participacion'
    OR v_decision->>'recurso_ref' IS DISTINCT FROM p_propuesta_ref
    OR v_decision->>'contexto_recurso_huella_sha256' IS DISTINCT FROM encode(sha256(convert_to(
      '{"ambitos":{},"atributos":{"propuesta_ref":"'||p_propuesta_ref||'"}}','UTF8')),'hex')
    OR v_decision->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_capacidad->>'huella_efecto_sha256'
    OR v_decision->'campos_permitidos' IS DISTINCT FROM '["estado","propuesta","recibo"]'::jsonb
    OR v_decision->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'B57: consulta de propuesta denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.consumir_consulta_propuesta_causas_bolsa_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF v_consumo.consumo_nuevo IS NOT TRUE OR v_consumo.efecto_ref IS DISTINCT FROM p_propuesta_ref
    OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM v_decision->>'contexto_recurso_huella_sha256'
 THEN RAISE EXCEPTION 'B57: consulta de propuesta denegada' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT p.propuesta_ref,p.codigo,p.version,p.huella_sha256,p.etiqueta,p.aplica_situacion,
  p.aplica_contacto,p.publicable,p.activa,p.recibo_ref,
  CASE WHEN c.propuesta_ref IS NOT NULL THEN 'publicada'::text
       WHEN EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.causa_participacion_catalogo x
                   WHERE x.codigo=p.codigo AND x.version>=p.version) THEN 'superada'::text
       ELSE 'pendiente'::text END
 FROM vec_bolsa_llamamientos.propuesta_causa_participacion p
 LEFT JOIN vec_bolsa_llamamientos.causa_participacion_catalogo c ON c.propuesta_ref=p.propuesta_ref
 WHERE p.propuesta_ref=p_propuesta_ref;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.consultar_propuesta_causa_participacion_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.consultar_propuesta_causa_participacion_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_ejecutor;

CREATE FUNCTION vec_bolsa_llamamientos.publicar_causa_participacion_v1(
 p_codigo text,p_version bigint,p_etiqueta text,p_aplica_situacion boolean,p_aplica_contacto boolean,
 p_publicable boolean,p_activa boolean,p_huella_sha256 text,p_actor text,p_recibo_ref text,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,
 p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea,p_propuesta_ref text)
RETURNS TABLE(codigo text,version bigint,huella_sha256 text,recibo_ref text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE v_decision jsonb; v_contexto jsonb; v_persona_ref text; v_consumo record; v_ultima bigint; v_previa record; v_propuesta record;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_migrador','MEMBER')
    OR current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
    OR current_setting('TimeZone')<>'UTC'
    OR p_codigo IS NULL OR p_codigo !~ '^[a-z][a-z0-9_]{2,63}$' OR p_version IS NULL OR p_version<1
    OR p_etiqueta IS NULL OR p_etiqueta<>btrim(p_etiqueta) OR octet_length(p_etiqueta) NOT BETWEEN 1 AND 120 OR p_etiqueta ~ '[[:cntrl:]]'
    OR p_aplica_situacion IS NULL OR p_aplica_contacto IS NULL OR NOT (p_aplica_situacion OR p_aplica_contacto)
    OR p_publicable IS NULL OR p_activa IS NULL
    OR p_huella_sha256 IS DISTINCT FROM vec_bolsa_llamamientos.huella_causa_participacion_v1(p_codigo,p_version,p_etiqueta,p_aplica_situacion,p_aplica_contacto,p_publicable,p_activa)
    OR p_actor IS NULL OR p_actor !~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{7,255}$'
    OR p_propuesta_ref IS NULL OR p_propuesta_ref !~ '^propuesta:causa:[0-9a-f]{64}$'
    OR p_recibo_ref IS DISTINCT FROM 'recibo:causa:'||encode(sha256(convert_to(
      'bolsa.causa_participacion.recibo.v1'||E'\n'||p_huella_sha256,'UTF8')),'hex')
 THEN RAISE EXCEPTION 'B57: publicación denegada' USING ERRCODE='42501'; END IF;
 BEGIN v_decision:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'B57: publicación denegada' USING ERRCODE='42501'; END;
 IF v_decision->>'principal_id' IS DISTINCT FROM p_actor
    OR v_decision->>'accion' IS DISTINCT FROM 'bolsa.causas_participacion.publicar'
    OR v_decision->>'modulo_id' IS DISTINCT FROM 'bolsa'
    OR v_decision->>'tipo_recurso' IS DISTINCT FROM 'catalogo_causas_participacion'
    OR v_decision->>'finalidad' IS DISTINCT FROM 'gestionar_catalogo_causas_participacion'
    OR v_decision->>'recurso_ref' IS DISTINCT FROM 'vec.bolsa.causas_participacion'
    -- El contexto V3 compromete la entrada exacta y el recibo. La aplicación
    -- debe resolver estos atributos en servidor, no copiarlos del navegador.
    OR v_decision->>'contexto_recurso_huella_sha256' IS DISTINCT FROM encode(sha256(convert_to(
      '{"ambitos":{},"atributos":{"causa_sha256":"'||p_huella_sha256||'","propuesta_ref":"'||p_propuesta_ref||'","recibo_ref":"'||p_recibo_ref||'"}}','UTF8')),'hex')
    OR v_decision->'campos_permitidos' IS DISTINCT FROM '["catalogo","recibo"]'::jsonb
    OR v_decision->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'B57: publicación denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_catalogo_causas_bolsa_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF v_consumo.consumo_nuevo IS NOT TRUE OR v_consumo.efecto_ref IS DISTINCT FROM 'vec.bolsa.causas_participacion'
    OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM v_decision->>'contexto_recurso_huella_sha256'
 THEN RAISE EXCEPTION 'B57: publicación denegada' USING ERRCODE='42501'; END IF;
 BEGIN v_contexto:=convert_from(p_contexto,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'B57: contexto de publicador inválido' USING ERRCODE='42501'; END;
 v_persona_ref:=v_contexto->>'persona_ref';
 IF v_contexto->>'esquema' IS DISTINCT FROM 'vec.contexto-actor.vinculado.v2'
    OR v_contexto->>'principal_ref' IS DISTINCT FROM p_actor
    OR v_persona_ref IS NULL OR v_persona_ref !~ '^per_[A-Za-z0-9_-]{22,128}$'
 THEN RAISE EXCEPTION 'B57: contexto de publicador inválido' USING ERRCODE='42501'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:causas_participacion',0));
 SELECT * INTO v_propuesta FROM vec_bolsa_llamamientos.propuesta_causa_participacion p
 WHERE p.propuesta_ref=p_propuesta_ref FOR SHARE;
 IF NOT FOUND OR v_propuesta.codigo IS DISTINCT FROM p_codigo OR v_propuesta.version IS DISTINCT FROM p_version
    OR v_propuesta.etiqueta IS DISTINCT FROM p_etiqueta OR v_propuesta.aplica_situacion IS DISTINCT FROM p_aplica_situacion
    OR v_propuesta.aplica_contacto IS DISTINCT FROM p_aplica_contacto OR v_propuesta.publicable IS DISTINCT FROM p_publicable
    OR v_propuesta.activa IS DISTINCT FROM p_activa OR v_propuesta.huella_sha256 IS DISTINCT FROM p_huella_sha256
 THEN RAISE EXCEPTION 'B57: propuesta no coincide' USING ERRCODE='VBS01'; END IF;
 IF v_propuesta.propuesta_por IS NOT DISTINCT FROM p_actor
    OR v_propuesta.propuesta_persona_ref IS NOT DISTINCT FROM v_persona_ref THEN
  RAISE EXCEPTION 'B57: publicador debe ser otra identidad' USING ERRCODE='42501';
 END IF;
 SELECT * INTO v_previa FROM vec_bolsa_llamamientos.causa_participacion_catalogo c WHERE c.recibo_ref=p_recibo_ref;
 IF FOUND THEN
  IF v_previa.codigo IS DISTINCT FROM p_codigo OR v_previa.version IS DISTINCT FROM p_version
     OR v_previa.etiqueta IS DISTINCT FROM p_etiqueta OR v_previa.aplica_situacion IS DISTINCT FROM p_aplica_situacion
     OR v_previa.aplica_contacto IS DISTINCT FROM p_aplica_contacto OR v_previa.publicable IS DISTINCT FROM p_publicable
     OR v_previa.activa IS DISTINCT FROM p_activa OR v_previa.huella_sha256 IS DISTINCT FROM p_huella_sha256
     OR v_previa.actor_ref IS DISTINCT FROM p_actor OR v_previa.publicada_persona_ref IS DISTINCT FROM v_persona_ref
     OR v_previa.propuesta_ref IS DISTINCT FROM p_propuesta_ref
  THEN RAISE EXCEPTION 'B57: recibo divergente' USING ERRCODE='VBS01'; END IF;
  RETURN QUERY SELECT v_previa.codigo,v_previa.version,v_previa.huella_sha256,v_previa.recibo_ref;
  RETURN;
 END IF;
 SELECT coalesce(max(c.version),0) INTO v_ultima FROM vec_bolsa_llamamientos.causa_participacion_catalogo c WHERE c.codigo=p_codigo;
 IF p_version<>v_ultima+1 THEN RAISE EXCEPTION 'B57: versión no consecutiva' USING ERRCODE='VBS57'; END IF;
 INSERT INTO vec_bolsa_llamamientos.causa_participacion_catalogo
  (codigo,version,etiqueta,aplica_situacion,aplica_contacto,publicable,activa,huella_sha256,actor_ref,recibo_ref,propuesta_ref,publicada_persona_ref,publicada_en)
 VALUES(p_codigo,p_version,p_etiqueta,p_aplica_situacion,p_aplica_contacto,p_publicable,p_activa,p_huella_sha256,p_actor,p_recibo_ref,p_propuesta_ref,v_persona_ref,clock_timestamp());
 RETURN QUERY SELECT p_codigo,p_version,p_huella_sha256,p_recibo_ref;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.publicar_causa_participacion_v1(text,bigint,text,boolean,boolean,boolean,boolean,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.publicar_causa_participacion_v1(text,bigint,text,boolean,boolean,boolean,boolean,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text) TO vec_bolsa_llamamientos_ejecutor;

CREATE FUNCTION vec_bolsa_llamamientos.listar_causas_participacion_v1()
RETURNS TABLE(codigo text,version bigint,huella_sha256 text,etiqueta text,aplica_situacion boolean,aplica_contacto boolean)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT c.codigo,c.version,c.huella_sha256,c.etiqueta,c.aplica_situacion,c.aplica_contacto
 FROM vec_bolsa_llamamientos.causa_participacion_catalogo c
 WHERE c.activa AND c.publicable AND c.version=(SELECT max(x.version) FROM vec_bolsa_llamamientos.causa_participacion_catalogo x WHERE x.codigo=c.codigo)
 ORDER BY c.codigo LIMIT 501
$f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.listar_causas_participacion_v1() FROM PUBLIC;

CREATE FUNCTION vec_bolsa_llamamientos.listar_causas_participacion_v2(
 p_actor text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(codigo text,version bigint,huella_sha256 text,etiqueta text,aplica_situacion boolean,aplica_contacto boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE v_capacidad jsonb; v_decision jsonb; v_consumo record;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_migrador','MEMBER')
    OR current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
    OR current_setting('TimeZone')<>'UTC' OR p_actor IS NULL OR p_actor=''
 THEN RAISE EXCEPTION 'B57: consulta de catálogo denegada' USING ERRCODE='42501'; END IF;
 BEGIN
  v_capacidad:=convert_from(p_capacidad,'UTF8')::jsonb;
  v_decision:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'B57: consulta de catálogo denegada' USING ERRCODE='42501'; END;
 IF v_capacidad->>'audiencia_consumo' IS DISTINCT FROM 'vec_bolsa_llamamientos.causas_participacion.consultar.v1'
    OR v_capacidad->>'operacion' IS DISTINCT FROM 'bolsa.causas_participacion.consultar'
    OR v_capacidad->>'efecto_ref' IS DISTINCT FROM 'vec.bolsa.causas_participacion'
    OR v_decision->>'principal_id' IS DISTINCT FROM p_actor
    OR v_decision->>'accion' IS DISTINCT FROM 'bolsa.causas_participacion.consultar'
    OR v_decision->>'modulo_id' IS DISTINCT FROM 'bolsa'
    OR v_decision->>'tipo_recurso' IS DISTINCT FROM 'catalogo_causas_participacion'
    OR v_decision->>'finalidad' IS DISTINCT FROM 'seleccionar_causa_participacion'
    OR v_decision->>'recurso_ref' IS DISTINCT FROM 'vec.bolsa.causas_participacion'
    OR v_decision->>'contexto_recurso_huella_sha256' IS DISTINCT FROM encode(sha256(convert_to('{"ambitos":{},"atributos":{}}','UTF8')),'hex')
    OR v_decision->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_capacidad->>'huella_efecto_sha256'
    OR v_decision->'campos_permitidos' IS DISTINCT FROM '["aplica_contacto","aplica_situacion","codigo","etiqueta","huella_sha256","version"]'::jsonb
    OR v_decision->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'B57: consulta de catálogo denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_catalogo_causas_bolsa_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF v_consumo.consumo_nuevo IS NOT TRUE OR v_consumo.efecto_ref IS DISTINCT FROM 'vec.bolsa.causas_participacion'
    OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM v_decision->>'contexto_recurso_huella_sha256'
 THEN RAISE EXCEPTION 'B57: consulta de catálogo denegada' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT * FROM vec_bolsa_llamamientos.listar_causas_participacion_v1();
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.listar_causas_participacion_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.listar_causas_participacion_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_ejecutor;

CREATE TABLE vec_bolsa_llamamientos.causa_situacion_participacion (
 recibo_ref text PRIMARY KEY REFERENCES vec_bolsa_llamamientos.situacion_participacion(recibo_ref),
 codigo text NOT NULL, version bigint NOT NULL, huella_sha256 text NOT NULL,
 FOREIGN KEY(codigo,version) REFERENCES vec_bolsa_llamamientos.causa_participacion_catalogo(codigo,version)
);
CREATE TABLE vec_bolsa_llamamientos.causa_contacto_participacion (
 recibo_ref text PRIMARY KEY REFERENCES vec_bolsa_llamamientos.datos_contacto_participacion(recibo_ref),
 codigo text NOT NULL, version bigint NOT NULL, huella_sha256 text NOT NULL,
 FOREIGN KEY(codigo,version) REFERENCES vec_bolsa_llamamientos.causa_participacion_catalogo(codigo,version)
);
ALTER TABLE vec_bolsa_llamamientos.causa_situacion_participacion ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.causa_situacion_participacion FORCE ROW LEVEL SECURITY;
CREATE POLICY causa_situacion_propietario ON vec_bolsa_llamamientos.causa_situacion_participacion TO vec_bolsa_llamamientos_propietario USING (current_user='vec_bolsa_llamamientos_propietario') WITH CHECK (current_user='vec_bolsa_llamamientos_propietario');
ALTER TABLE vec_bolsa_llamamientos.causa_contacto_participacion ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.causa_contacto_participacion FORCE ROW LEVEL SECURITY;
CREATE POLICY causa_contacto_propietario ON vec_bolsa_llamamientos.causa_contacto_participacion TO vec_bolsa_llamamientos_propietario USING (current_user='vec_bolsa_llamamientos_propietario') WITH CHECK (current_user='vec_bolsa_llamamientos_propietario');
REVOKE ALL ON vec_bolsa_llamamientos.causa_situacion_participacion,vec_bolsa_llamamientos.causa_contacto_participacion FROM PUBLIC;
CREATE TRIGGER causa_situacion_inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.causa_situacion_participacion FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();
CREATE TRIGGER causa_contacto_inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.causa_contacto_participacion FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();

-- Un selector nuevo debe ser la última versión activa de su código. El
-- cerrojo compartido evita una publicación concurrente hasta confirmar el
-- efecto. Un replay valida el selector unido al recibo, sin reinterpretarlo.
CREATE FUNCTION vec_bolsa_llamamientos.exigir_causa_participacion_v1(
 p_codigo text,p_version bigint,p_huella_sha256 text,p_operacion text)
RETURNS void LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE v_causa record;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR p_operacion NOT IN ('situacion','contacto')
 THEN RAISE EXCEPTION 'B57: causa inválida' USING ERRCODE='42501'; END IF;
 PERFORM pg_advisory_xact_lock_shared(hashtextextended('vec_bolsa_llamamientos:causas_participacion',0));
 SELECT c.* INTO v_causa FROM vec_bolsa_llamamientos.causa_participacion_catalogo c
 WHERE c.codigo=p_codigo ORDER BY c.version DESC LIMIT 1;
 IF NOT FOUND OR v_causa.version IS DISTINCT FROM p_version
    OR v_causa.huella_sha256 IS DISTINCT FROM p_huella_sha256
    OR v_causa.activa IS NOT TRUE
    OR (p_operacion='situacion' AND v_causa.aplica_situacion IS NOT TRUE)
    OR (p_operacion='contacto' AND v_causa.aplica_contacto IS NOT TRUE)
 THEN RAISE EXCEPTION 'B57: causa inválida' USING ERRCODE='22023'; END IF;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.exigir_causa_participacion_v1(text,bigint,text,text) FROM PUBLIC;

CREATE FUNCTION vec_bolsa_llamamientos.registrar_situacion_participacion_v2(
 p_bolsa_ref text,p_participacion_ref text,p_situacion text,p_desde timestamptz,p_fecha_disponible timestamptz,
 p_motivo text,p_actor text,p_clave_idempotencia text,p_recibo_ref text,p_registrada_en timestamptz,
 p_capacidad bytea,p_decision bytea,p_motivo_autorizacion bytea,p_contexto bytea,p_persona_version numeric,
 p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea,
 p_causa_codigo text,p_causa_version bigint,p_causa_sha256 text)
RETURNS TABLE(reutilizada boolean,recibo_ref text,situacion text,desde timestamptz,fecha_disponible timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE v_alta record; v_existente text; v_causa record;
BEGIN
 IF p_motivo IS DISTINCT FROM p_causa_codigo OR p_causa_codigo IS NULL OR p_causa_version IS NULL
    OR p_causa_sha256 IS NULL OR p_causa_sha256 !~ '^[0-9a-f]{64}$'
 THEN RAISE EXCEPTION 'B57: causa inválida' USING ERRCODE='22023'; END IF;
 SELECT s.recibo_ref INTO v_existente FROM vec_bolsa_llamamientos.situacion_participacion s
 WHERE s.participacion_ref=p_participacion_ref AND s.clave_idempotencia=p_clave_idempotencia;
 IF v_existente IS NULL THEN
  PERFORM vec_bolsa_llamamientos.exigir_causa_participacion_v1(p_causa_codigo,p_causa_version,p_causa_sha256,'situacion');
 END IF;
 SELECT * INTO STRICT v_alta FROM vec_bolsa_llamamientos.registrar_situacion_participacion_v1(
  p_bolsa_ref,p_participacion_ref,p_situacion,p_desde,p_fecha_disponible,p_motivo,p_actor,p_clave_idempotencia,p_recibo_ref,p_registrada_en,
  p_capacidad,p_decision,p_motivo_autorizacion,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF v_alta.reutilizada THEN
  SELECT c.* INTO v_causa FROM vec_bolsa_llamamientos.causa_situacion_participacion c WHERE c.recibo_ref=v_alta.recibo_ref;
  IF NOT FOUND OR v_causa.codigo IS DISTINCT FROM p_causa_codigo OR v_causa.version IS DISTINCT FROM p_causa_version
     OR v_causa.huella_sha256 IS DISTINCT FROM p_causa_sha256
  THEN RAISE EXCEPTION 'B57: replay divergente' USING ERRCODE='VBS01'; END IF;
 ELSE
  INSERT INTO vec_bolsa_llamamientos.causa_situacion_participacion(recibo_ref,codigo,version,huella_sha256)
  VALUES(v_alta.recibo_ref,p_causa_codigo,p_causa_version,p_causa_sha256);
 END IF;
 RETURN QUERY SELECT v_alta.reutilizada,v_alta.recibo_ref,v_alta.situacion,v_alta.desde,v_alta.fecha_disponible;
END $f$;

CREATE FUNCTION vec_bolsa_llamamientos.registrar_datos_contacto_participacion_v2(
 p_bolsa_ref text,p_participacion_ref text,p_version bigint,p_clave_ref text,p_nonce bytea,p_cifrado bytea,
 p_motivo text,p_actor text,p_registrada_en timestamptz,p_clave_idempotencia text,p_recibo_ref text,
 p_capacidad bytea,p_decision bytea,p_motivo_autorizacion bytea,p_contexto bytea,p_persona_version numeric,
 p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea,
 p_causa_codigo text,p_causa_version bigint,p_causa_sha256 text)
RETURNS TABLE(reutilizada boolean,recibo_ref text,version bigint,registrada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE v_alta record; v_existente text; v_causa record;
BEGIN
 IF p_motivo IS DISTINCT FROM p_causa_codigo OR p_causa_codigo IS NULL OR p_causa_version IS NULL
    OR p_causa_sha256 IS NULL OR p_causa_sha256 !~ '^[0-9a-f]{64}$'
 THEN RAISE EXCEPTION 'B57: causa inválida' USING ERRCODE='22023'; END IF;
 SELECT d.recibo_ref INTO v_existente FROM vec_bolsa_llamamientos.datos_contacto_participacion d
 WHERE d.participacion_ref=p_participacion_ref AND d.clave_idempotencia=p_clave_idempotencia;
 IF v_existente IS NULL THEN
  PERFORM vec_bolsa_llamamientos.exigir_causa_participacion_v1(p_causa_codigo,p_causa_version,p_causa_sha256,'contacto');
 END IF;
 SELECT * INTO STRICT v_alta FROM vec_bolsa_llamamientos.registrar_datos_contacto_participacion_v1(
  p_bolsa_ref,p_participacion_ref,p_version,p_clave_ref,p_nonce,p_cifrado,p_motivo,p_actor,p_registrada_en,p_clave_idempotencia,p_recibo_ref,
  p_capacidad,p_decision,p_motivo_autorizacion,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF v_alta.reutilizada THEN
  SELECT c.* INTO v_causa FROM vec_bolsa_llamamientos.causa_contacto_participacion c WHERE c.recibo_ref=v_alta.recibo_ref;
  IF NOT FOUND OR v_causa.codigo IS DISTINCT FROM p_causa_codigo OR v_causa.version IS DISTINCT FROM p_causa_version
     OR v_causa.huella_sha256 IS DISTINCT FROM p_causa_sha256
  THEN RAISE EXCEPTION 'B57: replay divergente' USING ERRCODE='VBS01'; END IF;
 ELSE
  INSERT INTO vec_bolsa_llamamientos.causa_contacto_participacion(recibo_ref,codigo,version,huella_sha256)
  VALUES(v_alta.recibo_ref,p_causa_codigo,p_causa_version,p_causa_sha256);
 END IF;
 RETURN QUERY SELECT v_alta.reutilizada,v_alta.recibo_ref,v_alta.version,v_alta.registrada_en;
END $f$;

-- La marca CONVOCA permanece en la misma transacción y conserva su contrato
-- de recuperación; B57 añade la causa al recibo después de B35.
CREATE FUNCTION vec_bolsa_llamamientos.registrar_datos_contacto_origen_participacion_v2(
 p_bolsa_ref text,p_participacion_ref text,p_version bigint,p_clave_ref text,p_nonce bytea,p_cifrado bytea,
 p_motivo text,p_actor text,p_registrada_en timestamptz,p_clave_idempotencia text,p_recibo_ref text,
 p_capacidad bytea,p_decision bytea,p_motivo_autorizacion bytea,p_contexto bytea,p_persona_version numeric,
 p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea,
 p_origen text,p_vigente_hasta timestamptz,p_ultimo_dia date,p_regla_ref text,p_regla_huella_sha256 text,
 p_causa_codigo text,p_causa_version bigint,p_causa_sha256 text)
RETURNS TABLE(o_reutilizada boolean,o_recibo_ref text,o_version bigint,o_registrada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE v_alta record; v_existente text; v_causa record;
BEGIN
 IF p_motivo IS DISTINCT FROM p_causa_codigo OR p_causa_codigo IS NULL OR p_causa_version IS NULL
    OR p_causa_sha256 IS NULL OR p_causa_sha256 !~ '^[0-9a-f]{64}$'
 THEN RAISE EXCEPTION 'B57: causa inválida' USING ERRCODE='22023'; END IF;
 SELECT d.recibo_ref INTO v_existente FROM vec_bolsa_llamamientos.datos_contacto_participacion d
 WHERE d.participacion_ref=p_participacion_ref AND d.clave_idempotencia=p_clave_idempotencia;
 IF v_existente IS NULL THEN
  PERFORM vec_bolsa_llamamientos.exigir_causa_participacion_v1(p_causa_codigo,p_causa_version,p_causa_sha256,'contacto');
 END IF;
 SELECT * INTO STRICT v_alta FROM vec_bolsa_llamamientos.registrar_datos_contacto_origen_participacion_v1(
  p_bolsa_ref,p_participacion_ref,p_version,p_clave_ref,p_nonce,p_cifrado,p_motivo,p_actor,p_registrada_en,p_clave_idempotencia,p_recibo_ref,
  p_capacidad,p_decision,p_motivo_autorizacion,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz,
  p_origen,p_vigente_hasta,p_ultimo_dia,p_regla_ref,p_regla_huella_sha256);
 IF v_alta.o_reutilizada THEN
  SELECT c.* INTO v_causa FROM vec_bolsa_llamamientos.causa_contacto_participacion c WHERE c.recibo_ref=v_alta.o_recibo_ref;
  IF NOT FOUND OR v_causa.codigo IS DISTINCT FROM p_causa_codigo OR v_causa.version IS DISTINCT FROM p_causa_version
     OR v_causa.huella_sha256 IS DISTINCT FROM p_causa_sha256
  THEN RAISE EXCEPTION 'B57: replay divergente' USING ERRCODE='VBS01'; END IF;
 ELSE
  INSERT INTO vec_bolsa_llamamientos.causa_contacto_participacion(recibo_ref,codigo,version,huella_sha256)
  VALUES(v_alta.o_recibo_ref,p_causa_codigo,p_causa_version,p_causa_sha256);
 END IF;
 RETURN QUERY SELECT v_alta.o_reutilizada,v_alta.o_recibo_ref,v_alta.o_version,v_alta.o_registrada_en;
END $f$;

-- B19 también escribe B2; su entrada pública v1 se sustituye por una v2
-- que añade el selector a la misma transacción que operación y situación.
CREATE FUNCTION vec_bolsa_llamamientos.registrar_operacion_situacion_participacion_v2(
 p_bolsa_ref text,p_participacion_ref text,p_operacion text,p_desde timestamptz,p_fecha_disponible timestamptz,
 p_motivo text,p_actor text,p_clave_idempotencia text,p_recibo_ref text,p_registrada_en timestamptz,
 p_justificante_tipo text,p_justificante_ref text,p_justificante_sha256 text,p_validador text,p_validada_en timestamptz,
 p_capacidad bytea,p_decision bytea,p_motivo_autorizacion bytea,p_contexto bytea,p_persona_version numeric,
 p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea,
 p_causa_codigo text,p_causa_version bigint,p_causa_sha256 text)
RETURNS TABLE(reutilizada boolean,recibo_ref text,situacion text,desde timestamptz,fecha_disponible timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE v_alta record; v_existente text; v_causa record;
BEGIN
 IF p_motivo IS DISTINCT FROM p_causa_codigo OR p_causa_codigo IS NULL OR p_causa_version IS NULL
    OR p_causa_sha256 IS NULL OR p_causa_sha256 !~ '^[0-9a-f]{64}$'
 THEN RAISE EXCEPTION 'B57: causa inválida' USING ERRCODE='22023'; END IF;
 SELECT s.recibo_ref INTO v_existente FROM vec_bolsa_llamamientos.situacion_participacion s
 WHERE s.participacion_ref=p_participacion_ref AND s.clave_idempotencia=p_clave_idempotencia;
 IF v_existente IS NULL THEN
  PERFORM vec_bolsa_llamamientos.exigir_causa_participacion_v1(p_causa_codigo,p_causa_version,p_causa_sha256,'situacion');
 END IF;
 SELECT * INTO STRICT v_alta FROM vec_bolsa_llamamientos.registrar_operacion_situacion_participacion_v1(
  p_bolsa_ref,p_participacion_ref,p_operacion,p_desde,p_fecha_disponible,p_motivo,p_actor,p_clave_idempotencia,p_recibo_ref,p_registrada_en,
  p_justificante_tipo,p_justificante_ref,p_justificante_sha256,p_validador,p_validada_en,
  p_capacidad,p_decision,p_motivo_autorizacion,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF v_alta.reutilizada THEN
  SELECT c.* INTO v_causa FROM vec_bolsa_llamamientos.causa_situacion_participacion c WHERE c.recibo_ref=v_alta.recibo_ref;
  IF NOT FOUND OR v_causa.codigo IS DISTINCT FROM p_causa_codigo OR v_causa.version IS DISTINCT FROM p_causa_version
     OR v_causa.huella_sha256 IS DISTINCT FROM p_causa_sha256
  THEN RAISE EXCEPTION 'B57: replay divergente' USING ERRCODE='VBS01'; END IF;
 ELSE
  INSERT INTO vec_bolsa_llamamientos.causa_situacion_participacion(recibo_ref,codigo,version,huella_sha256)
  VALUES(v_alta.recibo_ref,p_causa_codigo,p_causa_version,p_causa_sha256);
 END IF;
 RETURN QUERY SELECT v_alta.reutilizada,v_alta.recibo_ref,v_alta.situacion,v_alta.desde,v_alta.fecha_disponible;
END $f$;

CREATE FUNCTION vec_bolsa_llamamientos.listar_operaciones_situacion_participacion_v2(
 p_participacion_ref text,p_actor text,p_capacidad bytea,p_decision bytea,p_motivo_autorizacion bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(desde timestamptz, operacion text, situacion text, justificante_tipo text, justificante_ref text, justificante_sha256 text, actor text, validador text, validada_en timestamptz, motivo text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = pg_catalog AS $f$
DECLARE v_consumo record; v_decision jsonb;
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario' OR p_participacion_ref IS NULL OR p_actor IS NULL THEN
  RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='consulta de operaciones no autorizada';
 END IF;
 SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_situacion_participacion_v3_atestada(
  p_capacidad,p_decision,p_motivo_autorizacion,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 BEGIN v_decision:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='consulta de operaciones no autorizada'; END;
 IF v_consumo.efecto_ref IS DISTINCT FROM p_participacion_ref OR v_consumo.consumo_nuevo IS NOT TRUE
    OR v_decision->>'principal_id' IS DISTINCT FROM p_actor
    OR v_decision->>'accion' IS DISTINCT FROM 'bolsa.situacion_participacion.cambiar'
    OR v_decision->>'modulo_id' IS DISTINCT FROM 'bolsa'
    OR v_decision->>'tipo_recurso' IS DISTINCT FROM 'participacion_bolsa'
    OR v_decision->>'finalidad' IS DISTINCT FROM 'gestion_situacion_participacion'
    OR v_decision->>'recurso_ref' IS DISTINCT FROM p_participacion_ref
    OR v_decision->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb
    OR v_decision->'obligaciones' IS DISTINCT FROM '[]'::jsonb
    OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM v_decision->>'contexto_recurso_huella_sha256' THEN
  RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='consulta de operaciones no autorizada';
 END IF;
 RETURN QUERY SELECT o.desde, o.operacion, s.situacion, o.justificante_tipo, o.justificante_ref, o.justificante_sha256, o.actor, o.validador, o.validada_en,
   coalesce(vec_bolsa_llamamientos.etiqueta_causa_participacion_publicable_v1(cs.codigo,cs.version,cs.huella_sha256),'Motivo reservado en Bolsa')::text
   FROM vec_bolsa_llamamientos.operacion_situacion_participacion o
   JOIN vec_bolsa_llamamientos.situacion_participacion s USING (participacion_ref, desde)
   LEFT JOIN vec_bolsa_llamamientos.causa_situacion_participacion cs ON cs.recibo_ref=s.recibo_ref
   LEFT JOIN vec_bolsa_llamamientos.causa_participacion_catalogo c ON c.codigo=cs.codigo AND c.version=cs.version
  WHERE o.participacion_ref = p_participacion_ref
  ORDER BY o.desde DESC;
END $f$;

-- B57 conserva exactamente la firma, la autorización V3, el cursor y el ACL de B56.
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.consultar_auditoria_participacion_v1(
 p_participacion_ref text, p_actor_filtro text, p_desde timestamptz, p_hasta timestamptz,
 p_antes_instante timestamptz, p_antes_fuente text, p_antes_id text, p_limite integer,
 p_principal text, p_finalidad text, p_motivo_ref text, p_filtro_sha256 text,
 p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea,
 p_persona_version numeric, p_perfil_version numeric, p_payload bytea,
 p_sobre bytea, p_evidencia bytea, p_raiz bytea)
RETURNS TABLE(id text, ocurrido_en timestamptz, accion text, actor_ref text,
 resultado text, expediente_ref text, recibo_ref text, motivo text,
 campo text, valor_anterior text, valor_nuevo text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE v_consumo record; v_decision jsonb; v_capacidad jsonb; v_huella_recurso text;
BEGIN
 -- B56: motivo unido. Marca de preimagen para denegar doble UP.
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR session_user = current_user
    OR NOT pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_migrador','MEMBER')
    OR current_setting('transaction_isolation') <> 'serializable'
    OR current_setting('transaction_read_only') <> 'off'
    OR current_setting('TimeZone') <> 'UTC'
    OR p_participacion_ref IS NULL OR p_participacion_ref = '' OR octet_length(p_participacion_ref) > 512
    OR p_principal IS NULL OR p_principal = '' OR octet_length(p_principal) > 256
    OR p_finalidad IS NULL OR p_finalidad = '' OR octet_length(p_finalidad) > 256
    OR p_motivo_ref IS NULL OR p_motivo_ref = '' OR octet_length(p_motivo_ref) > 256
    OR p_filtro_sha256 IS NULL OR p_filtro_sha256 !~ '^[0-9a-f]{64}$'
    OR (p_actor_filtro IS NOT NULL AND (p_actor_filtro = '' OR octet_length(p_actor_filtro) > 512))
    OR p_desde IS NULL OR p_hasta IS NULL OR p_desde >= p_hasta
    OR p_hasta - p_desde > interval '31 days'
    OR (p_antes_instante IS NULL) <> (p_antes_id IS NULL)
    OR (p_antes_instante IS NULL) <> (p_antes_fuente IS NULL)
    OR (p_antes_fuente IS NOT NULL AND p_antes_fuente <> 'bolsa')
    OR (p_antes_instante IS NOT NULL AND (p_antes_instante < p_desde OR p_antes_instante >= p_hasta))
    OR (p_antes_id IS NOT NULL AND (p_antes_id = '' OR octet_length(p_antes_id) > 512))
    OR p_limite IS NULL OR p_limite NOT BETWEEN 1 AND 100 THEN
  RAISE EXCEPTION 'consulta de auditoria Bolsa denegada' USING ERRCODE='42501';
 END IF;
 IF p_filtro_sha256 IS DISTINCT FROM encode(sha256(convert_to(array_to_string(ARRAY[
    'vec.auditoria.filtro.v1','bolsa',p_participacion_ref,coalesce(p_actor_filtro,''),
    coalesce(to_char(p_desde AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),''),
    coalesce(to_char(p_hasta AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),''),
    p_limite::text,
    coalesce(to_char(p_antes_instante AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),''),
    coalesce(p_antes_fuente,''),coalesce(p_antes_id,''),p_finalidad,p_motivo_ref
 ],E'\n'),'UTF8')),'hex') THEN
  RAISE EXCEPTION 'consulta de auditoria Bolsa denegada' USING ERRCODE='42501';
 END IF;
 -- El contexto usa la representación compacta de encoding/json de Go.
 -- Se escapan también los caracteres HTML que ese codificador protege.
 v_huella_recurso := encode(sha256(convert_to(
  '{"ambitos":{"expediente_ref":'||
  replace(replace(replace(replace(replace(to_json(p_participacion_ref)::text,
    '&','\u0026'),'<','\u003c'),'>','\u003e'),chr(8232),'\u2028'),chr(8233),'\u2029')||
  ',"fuente":"bolsa"},"atributos":{"filtro_sha256":'||to_json(p_filtro_sha256)::text||'}}','UTF8')),'hex');
 BEGIN
  v_decision := convert_from(p_decision,'UTF8')::jsonb;
  v_capacidad := convert_from(p_capacidad,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN
  RAISE EXCEPTION 'consulta de auditoria Bolsa denegada' USING ERRCODE='42501';
 END;
 IF v_capacidad->>'audiencia_consumo' IS DISTINCT FROM 'vec_auditoria.consulta_rrhh.v1'
    OR v_capacidad->>'operacion' IS DISTINCT FROM 'vec.auditoria.consultar'
    OR v_capacidad->>'efecto_ref' IS DISTINCT FROM p_participacion_ref
    OR v_decision->>'principal_id' IS DISTINCT FROM p_principal
    OR v_decision->>'accion' IS DISTINCT FROM 'vec.auditoria.consultar'
    OR v_decision->>'modulo_id' IS DISTINCT FROM 'auditoria'
    OR v_decision->>'tipo_recurso' IS DISTINCT FROM 'historial_auditoria'
    OR v_decision->>'recurso_ref' IS DISTINCT FROM p_participacion_ref
    OR v_decision->>'finalidad' IS DISTINCT FROM p_finalidad
    OR v_decision->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_huella_recurso
    OR v_decision->'campos_permitidos' IS DISTINCT FROM '["accion","actor_ref","antes","antes_sha256","datos_disponibles","despues","despues_sha256","expediente_ref","fuente","id","modulo_id","motivo","ocurrido_en","recibo_ref","resultado"]'::jsonb
    OR v_decision->'obligaciones' IS DISTINCT FROM '[]'::jsonb
    OR v_decision->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_capacidad->>'huella_efecto_sha256' THEN
  RAISE EXCEPTION 'consulta de auditoria Bolsa denegada' USING ERRCODE='42501';
 END IF;
 SELECT * INTO STRICT v_consumo
   FROM vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_auditoria_bolsa_v3_atestada(
    p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
    p_payload,p_sobre,p_evidencia,p_raiz);
 IF v_consumo.consumo_nuevo IS NOT TRUE
    OR v_consumo.efecto_ref IS DISTINCT FROM p_participacion_ref
    OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM v_decision->>'contexto_recurso_huella_sha256' THEN
  RAISE EXCEPTION 'consulta de auditoria Bolsa denegada' USING ERRCODE='42501';
 END IF;

 RETURN QUERY
 WITH hechos AS (
  SELECT ('situacion:' || s.recibo_ref)::text AS id, s.registrada_en AS ocurrido_en,
         coalesce(o.operacion, 'situacion:' || s.situacion)::text AS accion,
         s.actor AS actor_ref, 'confirmado'::text AS resultado,
         s.participacion_ref AS expediente_ref, s.recibo_ref,
         CASE WHEN vec_bolsa_llamamientos.etiqueta_causa_participacion_publicable_v1(cs.codigo,cs.version,cs.huella_sha256) IS NOT NULL THEN vec_bolsa_llamamientos.etiqueta_causa_participacion_publicable_v1(cs.codigo,cs.version,cs.huella_sha256)
              WHEN s.recibo_ref='recibo:situacion:constitucion:' || s.participacion_ref
                    AND s.actor='sistema:constitucion' AND s.situacion='disponible'
                    AND s.motivo='Constitución de bolsa' THEN 'Constitución de bolsa'
              ELSE 'Motivo reservado en Bolsa' END::text AS motivo,
         NULL::text AS campo, NULL::text AS valor_anterior, NULL::text AS valor_nuevo
    FROM vec_bolsa_llamamientos.situacion_participacion s
    LEFT JOIN vec_bolsa_llamamientos.operacion_situacion_participacion o
      ON o.participacion_ref=s.participacion_ref AND o.desde=s.desde
    LEFT JOIN vec_bolsa_llamamientos.causa_situacion_participacion cs ON cs.recibo_ref=s.recibo_ref
    LEFT JOIN vec_bolsa_llamamientos.causa_participacion_catalogo cc ON cc.codigo=cs.codigo AND cc.version=cs.version
   WHERE s.participacion_ref=p_participacion_ref
     AND NOT EXISTS (
       SELECT 1 FROM vec_bolsa_llamamientos.traza_valor_participacion t
        WHERE t.participacion_ref=s.participacion_ref AND t.recibo_ref=s.recibo_ref)
  UNION ALL
  SELECT ('cambio:' || t.recibo_ref || ':' || t.campo)::text, t.registrada_en,
         CASE WHEN s.recibo_ref IS NOT NULL
                THEN coalesce(o.operacion, 'situacion:' || s.situacion)
              ELSE 'valor:' || t.campo END::text,
         t.actor, 'confirmado'::text,
         t.participacion_ref, t.recibo_ref,
         CASE WHEN s.recibo_ref IS NOT NULL AND vec_bolsa_llamamientos.etiqueta_causa_participacion_publicable_v1(cs.codigo,cs.version,cs.huella_sha256) IS NOT NULL THEN vec_bolsa_llamamientos.etiqueta_causa_participacion_publicable_v1(cs.codigo,cs.version,cs.huella_sha256)
              WHEN d.recibo_ref IS NOT NULL AND vec_bolsa_llamamientos.etiqueta_causa_participacion_publicable_v1(cd.codigo,cd.version,cd.huella_sha256) IS NOT NULL THEN vec_bolsa_llamamientos.etiqueta_causa_participacion_publicable_v1(cd.codigo,cd.version,cd.huella_sha256)
              WHEN t.campo IN ('situacion','fecha_disponible')
                 AND s.recibo_ref='recibo:situacion:constitucion:' || s.participacion_ref
                 AND s.actor='sistema:constitucion' AND s.situacion='disponible'
                 AND s.motivo='Constitución de bolsa'
                THEN 'Constitución de bolsa'
              ELSE 'Motivo reservado en Bolsa' END::text,
         t.campo, t.valor_anterior, t.valor_nuevo
    FROM vec_bolsa_llamamientos.traza_valor_participacion t
    LEFT JOIN vec_bolsa_llamamientos.situacion_participacion s
      ON t.campo IN ('situacion','fecha_disponible')
     AND s.participacion_ref=t.participacion_ref AND s.recibo_ref=t.recibo_ref
    LEFT JOIN vec_bolsa_llamamientos.operacion_situacion_participacion o
      ON s.participacion_ref=o.participacion_ref AND s.desde=o.desde
    LEFT JOIN vec_bolsa_llamamientos.datos_contacto_participacion d
      ON t.campo IN ('datos_contacto','correo','telefono_1','telefono_2')
     AND d.participacion_ref=t.participacion_ref AND d.recibo_ref=t.recibo_ref
    LEFT JOIN vec_bolsa_llamamientos.causa_situacion_participacion cs ON cs.recibo_ref=s.recibo_ref
    LEFT JOIN vec_bolsa_llamamientos.causa_participacion_catalogo ccs ON ccs.codigo=cs.codigo AND ccs.version=cs.version
    LEFT JOIN vec_bolsa_llamamientos.causa_contacto_participacion cd ON cd.recibo_ref=d.recibo_ref
    LEFT JOIN vec_bolsa_llamamientos.causa_participacion_catalogo ccd ON ccd.codigo=cd.codigo AND ccd.version=cd.version
   WHERE t.participacion_ref=p_participacion_ref
     AND (s.recibo_ref IS NOT NULL OR d.recibo_ref IS NOT NULL)
 )
 SELECT h.id,h.ocurrido_en,h.accion,h.actor_ref,h.resultado,h.expediente_ref,
        h.recibo_ref,h.motivo,h.campo,h.valor_anterior,h.valor_nuevo
   FROM hechos h
  WHERE (p_actor_filtro IS NULL OR h.actor_ref=p_actor_filtro)
    AND (p_desde IS NULL OR h.ocurrido_en>=p_desde)
    AND h.ocurrido_en<p_hasta
    AND (p_antes_instante IS NULL OR (h.ocurrido_en,'bolsa',h.id)<(p_antes_instante,p_antes_fuente,p_antes_id))
  ORDER BY h.ocurrido_en DESC,h.id DESC
  LIMIT p_limite+1;
END $f$;


CREATE FUNCTION vec_bolsa_llamamientos.recuperar_situacion_participacion_v2(p_participacion_ref text,p_clave_idempotencia text)
RETURNS TABLE(recibo_ref text,situacion text,desde timestamptz,fecha_disponible timestamptz,motivo text,
 causa_codigo text,causa_version bigint,causa_sha256 text,causa_etiqueta text)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT s.recibo_ref,s.situacion,s.desde,s.fecha_disponible,s.motivo,
        cs.codigo,cs.version,cs.huella_sha256,
        vec_bolsa_llamamientos.etiqueta_causa_participacion_publicable_v1(cs.codigo,cs.version,cs.huella_sha256)
 FROM vec_bolsa_llamamientos.situacion_participacion s
 LEFT JOIN vec_bolsa_llamamientos.causa_situacion_participacion cs ON cs.recibo_ref=s.recibo_ref
 LEFT JOIN vec_bolsa_llamamientos.causa_participacion_catalogo c ON c.codigo=cs.codigo AND c.version=cs.version
 WHERE s.participacion_ref=p_participacion_ref AND s.clave_idempotencia=p_clave_idempotencia
$f$;
CREATE FUNCTION vec_bolsa_llamamientos.recuperar_datos_contacto_participacion_v2(p_participacion_ref text,p_clave_idempotencia text)
RETURNS TABLE(version bigint,clave_ref text,nonce bytea,cifrado bytea,motivo text,registrada_en timestamptz,recibo_ref text,
 causa_codigo text,causa_version bigint,causa_sha256 text,causa_etiqueta text)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT d.version,d.clave_ref,d.nonce,d.cifrado,d.motivo,d.registrada_en,d.recibo_ref,
        cd.codigo,cd.version,cd.huella_sha256,
        vec_bolsa_llamamientos.etiqueta_causa_participacion_publicable_v1(cd.codigo,cd.version,cd.huella_sha256)
 FROM vec_bolsa_llamamientos.datos_contacto_participacion d
 LEFT JOIN vec_bolsa_llamamientos.causa_contacto_participacion cd ON cd.recibo_ref=d.recibo_ref
 LEFT JOIN vec_bolsa_llamamientos.causa_participacion_catalogo c ON c.codigo=cd.codigo AND c.version=cd.version
 WHERE d.participacion_ref=p_participacion_ref AND d.clave_idempotencia=p_clave_idempotencia
$f$;

REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.registrar_situacion_participacion_v1(text,text,text,timestamptz,timestamptz,text,text,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM vec_bolsa_llamamientos_ejecutor;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.registrar_datos_contacto_participacion_v1(text,text,bigint,text,bytea,bytea,text,text,timestamptz,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM vec_bolsa_llamamientos_ejecutor;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.registrar_datos_contacto_origen_participacion_v1(text,text,bigint,text,bytea,bytea,text,text,timestamptz,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,timestamptz,date,text,text) FROM vec_bolsa_llamamientos_ejecutor;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.registrar_operacion_situacion_participacion_v1(text,text,text,timestamptz,timestamptz,text,text,text,text,timestamptz,text,text,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM vec_bolsa_llamamientos_ejecutor;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.listar_operaciones_situacion_participacion_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM vec_bolsa_llamamientos_ejecutor;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.registrar_operacion_situacion_participacion_v2(text,text,text,timestamptz,timestamptz,text,text,text,text,timestamptz,text,text,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,bigint,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.registrar_operacion_situacion_participacion_v2(text,text,text,timestamptz,timestamptz,text,text,text,text,timestamptz,text,text,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,bigint,text) TO vec_bolsa_llamamientos_ejecutor;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.listar_operaciones_situacion_participacion_v2(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.listar_operaciones_situacion_participacion_v2(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_ejecutor;

REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.registrar_situacion_participacion_v2(text,text,text,timestamptz,timestamptz,text,text,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,bigint,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.registrar_situacion_participacion_v2(text,text,text,timestamptz,timestamptz,text,text,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,bigint,text) TO vec_bolsa_llamamientos_ejecutor;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.registrar_datos_contacto_participacion_v2(text,text,bigint,text,bytea,bytea,text,text,timestamptz,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,bigint,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.registrar_datos_contacto_participacion_v2(text,text,bigint,text,bytea,bytea,text,text,timestamptz,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,bigint,text) TO vec_bolsa_llamamientos_ejecutor;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.registrar_datos_contacto_origen_participacion_v2(text,text,bigint,text,bytea,bytea,text,text,timestamptz,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,timestamptz,date,text,text,text,bigint,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.registrar_datos_contacto_origen_participacion_v2(text,text,bigint,text,bytea,bytea,text,text,timestamptz,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,timestamptz,date,text,text,text,bigint,text) TO vec_bolsa_llamamientos_ejecutor;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.recuperar_situacion_participacion_v2(text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.recuperar_situacion_participacion_v2(text,text) TO vec_bolsa_llamamientos_ejecutor;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.recuperar_datos_contacto_participacion_v2(text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.recuperar_datos_contacto_participacion_v2(text,text) TO vec_bolsa_llamamientos_ejecutor;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.consultar_auditoria_participacion_v1(text,text,timestamptz,timestamptz,timestamptz,text,text,integer,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.consultar_auditoria_participacion_v1(text,text,timestamptz,timestamptz,timestamptz,text,text,integer,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_ejecutor;
COMMIT;
