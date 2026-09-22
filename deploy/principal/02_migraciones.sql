\set ON_ERROR_STOP on
BEGIN;
-- Paquete incremental B4 parte 2 + D3-B7. Preimagen exigida: AD3-46 y Bolsa-15.
-- INICIO deploy/postgresql/autorizacion_atestada_v3/migraciones/000047_consumidor_datos_contacto_participacion.up.sql
-- AD3-000047. Extensión nominal B4 (datos de contacto del candidato: correo y
-- dos teléfonos) sobre la estructura de AD3-000046; conserva las guardas
-- anteriores sin exigir una autohuella del núcleo. Solo la escritura consume
-- V3: la lectura de B4 se resuelve con contexto y pertenencia, como B5.
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000047',0));

DO $precondicion$
DECLARE f oid := 'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_creacion_borrador_llamamiento_interno_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_borrador_llamamiento_interno_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_despacho_correo_llamamiento_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_resultado_correo_llamamiento_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_mi_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_situacion_participacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_contacto_participacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_contacto_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_datos_contacto_participacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR NOT EXISTS(SELECT 1 FROM pg_proc WHERE oid=f AND proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND prosecdef AND prokind='f' AND provolatile='v' AND pg_get_function_identity_arguments(oid)='p_perfil_mutacion text, p_capacidad_canonica bytea, p_decision_canonica bytea, p_motivo_canonico bytea, p_contexto_actor_canonico bytea, p_persona_version numeric, p_perfil_version numeric, p_payload_vec_ad_3 bytea, p_sobre_cose_sign1 bytea, p_evidencia_verificacion bytea, p_raiz_publica_spki bytea' AND proconfig=ARRAY['search_path=pg_catalog','lock_timeout=2s']) THEN
  RAISE EXCEPTION 'estructura AD3-000046 incompatible para B4' USING ERRCODE='55000';
 END IF;
END $precondicion$;

DO $nucleo$
DECLARE f oid := 'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 original text; esperada text; actual text; reconstruida text; metadata jsonb; deps jsonb; acl aclitem[]; propietario oid; configuracion text[]; es_definidora boolean;
 marca text := E'       )\n       OR c ->> ''suite'' <> ''VEC-AD-3-COSE-EDDSA-1''';
 exclusion_pre text := $x$               AND p_perfil_mutacion IS DISTINCT FROM 'consulta_borrador_llamamiento_interno_bolsa'
               AND p_perfil_mutacion IS DISTINCT FROM 'situacion_participacion_bolsa'
               AND p_perfil_mutacion IS DISTINCT FROM 'contacto_participacion_bolsa'$x$;
 exclusion_post text := exclusion_pre||E'\n               AND p_perfil_mutacion IS DISTINCT FROM ''datos_contacto_participacion_bolsa''';
 runtime_pre text := $x$               OR p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_borrador_llamamiento_interno_bolsa'
               OR p_perfil_mutacion IS NOT DISTINCT FROM 'situacion_participacion_bolsa'
               OR p_perfil_mutacion IS NOT DISTINCT FROM 'contacto_participacion_bolsa')$x$;
 runtime_post text := $x$               OR p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_borrador_llamamiento_interno_bolsa'
               OR p_perfil_mutacion IS NOT DISTINCT FROM 'situacion_participacion_bolsa'
               OR p_perfil_mutacion IS NOT DISTINCT FROM 'contacto_participacion_bolsa'
               OR p_perfil_mutacion IS NOT DISTINCT FROM 'datos_contacto_participacion_bolsa')$x$;
 extension text := $p$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'datos_contacto_participacion_bolsa'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.datos_contacto_participacion.registrar.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'bolsa.datos_contacto_participacion.registrar'
 AND d->>'accion' IS NOT DISTINCT FROM 'bolsa.datos_contacto_participacion.registrar'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'participacion_bolsa'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestion_datos_contacto_participacion')
$p$;
BEGIN
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc',p.proacl,p.proowner,p.proconfig,p.prosecdef INTO STRICT original,metadata,acl,propietario,configuracion,es_definidora FROM pg_proc p WHERE p.oid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb) INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 IF length(original)-length(replace(original,marca,''))<>length(marca) OR length(original)-length(replace(original,exclusion_pre,''))<>length(exclusion_pre) OR length(original)-length(replace(original,runtime_pre,''))<>length(runtime_pre) OR strpos(original,'creacion_borrador_llamamiento_interno_bolsa')=0 OR strpos(original,'consulta_participaciones_propias_bolsa')=0 OR strpos(original,'despacho_correo_llamamiento_ct')=0 OR strpos(original,'situacion_participacion_bolsa')=0 OR strpos(original,'contacto_participacion_bolsa')=0 OR strpos(original,'datos_contacto_participacion_bolsa')<>0 THEN RAISE EXCEPTION 'núcleo AD3-000046 no admite extensión B4' USING ERRCODE='55000'; END IF;
 esperada:=replace(original,exclusion_pre,exclusion_post); esperada:=replace(esperada,runtime_pre,runtime_post); esperada:=replace(esperada,marca,extension||marca); EXECUTE esperada;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 reconstruida:=replace(actual,extension||marca,marca); reconstruida:=replace(reconstruida,runtime_post,runtime_pre); reconstruida:=replace(reconstruida,exclusion_post,exclusion_pre);
 IF actual IS DISTINCT FROM esperada OR reconstruida IS DISTINCT FROM original OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM metadata OR (SELECT proacl FROM pg_proc WHERE oid=f) IS DISTINCT FROM acl OR (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM propietario OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM configuracion OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS DISTINCT FROM es_definidora OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb) FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps THEN RAISE EXCEPTION 'B4 alteró el núcleo AD3 fuera de su extensión nominal' USING ERRCODE='55000'; END IF;
END $nucleo$;

LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencia$ DECLARE definicion text; nueva text; BEGIN
 SELECT pg_get_constraintdef(c.oid,true) INTO STRICT definicion FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c';
 IF strpos(definicion,'vec_bolsa_llamamientos.datos_contacto_participacion.registrar.v1')<>0 OR strpos(definicion,'CHECK (audiencia_consumo = ANY (ARRAY[')<>1 OR right(definicion,3)<>']))' THEN RAISE EXCEPTION 'audiencias incompatibles para B4' USING ERRCODE='55000'; END IF;
 nueva:=left(definicion,length(definicion)-3)||', ''vec_bolsa_llamamientos.datos_contacto_participacion.registrar.v1''::text]))';
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||nueva;
END $audiencia$;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_datos_contacto_participacion_v3_atestada(p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; x record; BEGIN
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb; EXCEPTION WHEN others THEN RAISE EXCEPTION 'material B4 inválido' USING ERRCODE='22023'; END;
 IF c->>'audiencia_consumo' IS DISTINCT FROM 'vec_bolsa_llamamientos.datos_contacto_participacion.registrar.v1' OR c->>'operacion' IS DISTINCT FROM 'bolsa.datos_contacto_participacion.registrar' OR d->>'accion' IS DISTINCT FROM c->>'operacion' OR d->>'modulo_id' IS DISTINCT FROM 'bolsa' OR d->>'tipo_recurso' IS DISTINCT FROM 'participacion_bolsa' OR d->>'finalidad' IS DISTINCT FROM 'gestion_datos_contacto_participacion' OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref' OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256' THEN RAISE EXCEPTION 'material B4 rechazado' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna('datos_contacto_participacion_bolsa',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'B4 requiere consumo nuevo' USING ERRCODE='42501'; END IF; RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true; END $f$;
ALTER FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_datos_contacto_participacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) OWNER TO vec_autorizacion_atestada_v3_propietario;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_datos_contacto_participacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_bolsa_llamamientos_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_datos_contacto_participacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_propietario;
REVOKE GRANT OPTION FOR EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_datos_contacto_participacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM vec_bolsa_llamamientos_propietario;
DO $acl_cerrada$
DECLARE funcion regprocedure := 'vec_autorizacion_atestada_v3.registrar_y_consumir_datos_contacto_participacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure; a record;
BEGIN
 FOR a IN SELECT DISTINCT x.grantee FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x WHERE p.oid=funcion AND x.grantee<>p.proowner AND x.grantee<>'vec_bolsa_llamamientos_propietario'::regrole LOOP
  IF a.grantee=0 THEN EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC',funcion::text);
  ELSE EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %I',funcion::text,pg_get_userbyid(a.grantee)); END IF;
 END LOOP;
 IF NOT has_function_privilege('vec_bolsa_llamamientos_propietario',funcion,'EXECUTE') OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x WHERE p.oid=funcion AND x.privilege_type='EXECUTE' AND (x.grantee NOT IN (p.proowner,'vec_bolsa_llamamientos_propietario'::regrole) OR x.is_grantable AND x.grantee='vec_bolsa_llamamientos_propietario'::regrole)) THEN RAISE EXCEPTION 'ACL B4 incompatible' USING ERRCODE='55000'; END IF;
END $acl_cerrada$;
-- FIN deploy/postgresql/autorizacion_atestada_v3/migraciones/000047_consumidor_datos_contacto_participacion.up.sql
-- INICIO deploy/postgresql/bolsa_llamamientos/migraciones/000016_datos_contacto_participacion.up.sql
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SELECT pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000016', 0));

-- B4 (Peticion.pdf p.1: «correo electrónico y dos teléfonos» del candidato;
-- pliego SE15/2020 §1 c.2): datos de contacto de una participación, versionados
-- append-only. El agregado (correo, teléfono 1, teléfono 2) se guarda como un
-- único sobre AEAD cifrado por la aplicación con clave del KMS y ligado a la
-- participación y a la versión; la base nunca ve el claro ni ninguna huella de
-- él. Mismo circuito que B2: concesión V3 consumida, auditoría y alta en una
-- transacción; idempotencia por clave y recibo determinista.
CREATE TABLE vec_bolsa_llamamientos.datos_contacto_participacion (
    participacion_ref text NOT NULL,
    version bigint NOT NULL CHECK (version > 0),
    clave_ref text NOT NULL CHECK (octet_length(clave_ref) BETWEEN 1 AND 256),
    nonce bytea NOT NULL CHECK (octet_length(nonce) BETWEEN 12 AND 32),
    cifrado bytea NOT NULL CHECK (octet_length(cifrado) BETWEEN 16 AND 4096),
    motivo text NOT NULL CHECK (octet_length(motivo) BETWEEN 1 AND 1000 AND motivo = btrim(motivo)),
    actor text NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    clave_idempotencia text NOT NULL CHECK (octet_length(clave_idempotencia) BETWEEN 1 AND 256 AND clave_idempotencia = btrim(clave_idempotencia)),
    recibo_ref text NOT NULL CHECK (octet_length(recibo_ref) BETWEEN 1 AND 256 AND recibo_ref = btrim(recibo_ref)),
    PRIMARY KEY (participacion_ref, version),
    UNIQUE (participacion_ref, clave_idempotencia),
    UNIQUE (recibo_ref)
);
ALTER TABLE vec_bolsa_llamamientos.datos_contacto_participacion ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.datos_contacto_participacion FORCE ROW LEVEL SECURITY;
CREATE POLICY datos_contacto_participacion_solo_propietario ON vec_bolsa_llamamientos.datos_contacto_participacion TO vec_bolsa_llamamientos_propietario USING (current_user = 'vec_bolsa_llamamientos_propietario') WITH CHECK (current_user = 'vec_bolsa_llamamientos_propietario');
REVOKE ALL ON vec_bolsa_llamamientos.datos_contacto_participacion FROM PUBLIC;
CREATE TRIGGER datos_contacto_participacion_inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.datos_contacto_participacion FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();

-- La bitácora de frontera compartida admite también los fallos de B4.
DO $bitacora$
DECLARE accion text; ruta text;
BEGIN
 SELECT pg_get_constraintdef(oid,true) INTO STRICT accion FROM pg_constraint WHERE conrelid='vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento'::regclass AND conname='bitacora_intento_borrador_llamamiento_accion_check' AND contype='c';
 SELECT pg_get_constraintdef(oid,true) INTO STRICT ruta FROM pg_constraint WHERE conrelid='vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento'::regclass AND conname='bitacora_intento_borrador_llamamiento_ruta_clase_check' AND contype='c';
 IF strpos(accion,'''registrar_datos_contacto''')<>0 OR strpos(ruta,'''datos_contacto''')<>0 OR strpos(accion,'CHECK (accion = ANY (ARRAY[')<>1 OR right(accion,3)<>']))' OR strpos(ruta,'CHECK (ruta_clase = ANY (ARRAY[')<>1 OR right(ruta,3)<>']))' THEN
  RAISE EXCEPTION 'bitácora de frontera incompatible con B4' USING ERRCODE='55000';
 END IF;
 ALTER TABLE vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento DROP CONSTRAINT bitacora_intento_borrador_llamamiento_accion_check;
 EXECUTE 'ALTER TABLE vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento ADD CONSTRAINT bitacora_intento_borrador_llamamiento_accion_check '||left(accion,length(accion)-3)||', ''registrar_datos_contacto''::text]))';
 ALTER TABLE vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento DROP CONSTRAINT bitacora_intento_borrador_llamamiento_ruta_clase_check;
 EXECUTE 'ALTER TABLE vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento ADD CONSTRAINT bitacora_intento_borrador_llamamiento_ruta_clase_check '||left(ruta,length(ruta)-3)||', ''datos_contacto''::text]))';
END $bitacora$;

CREATE FUNCTION vec_bolsa_llamamientos.registrar_datos_contacto_participacion_v1(
    p_bolsa_ref text, p_participacion_ref text, p_version bigint, p_clave_ref text, p_nonce bytea, p_cifrado bytea,
    p_motivo text, p_actor text, p_registrada_en timestamptz, p_clave_idempotencia text, p_recibo_ref text,
    p_capacidad bytea, p_decision bytea, p_motivo_autorizacion bytea, p_contexto bytea, p_persona_version numeric,
    p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
RETURNS TABLE(reutilizada boolean, recibo_ref text, version bigint, registrada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = pg_catalog AS $f$
DECLARE consumo record; decision jsonb; v_vigente bigint;
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario' OR p_bolsa_ref IS NULL OR p_participacion_ref IS NULL OR p_version IS NULL OR p_version < 1
    OR p_clave_ref IS NULL OR octet_length(p_clave_ref) NOT BETWEEN 1 AND 256 OR p_nonce IS NULL OR octet_length(p_nonce) NOT BETWEEN 12 AND 32
    OR p_cifrado IS NULL OR octet_length(p_cifrado) NOT BETWEEN 16 AND 4096
    OR p_motivo IS NULL OR p_motivo <> btrim(p_motivo) OR octet_length(p_motivo) NOT BETWEEN 1 AND 1000 OR p_actor IS NULL OR p_registrada_en IS NULL
    OR p_clave_idempotencia IS NULL OR p_clave_idempotencia <> btrim(p_clave_idempotencia) OR octet_length(p_clave_idempotencia) NOT BETWEEN 1 AND 256
    OR p_recibo_ref IS NULL OR p_recibo_ref <> btrim(p_recibo_ref) OR octet_length(p_recibo_ref) NOT BETWEEN 1 AND 256 THEN
  RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='datos de contacto invalidos';
 END IF;
 IF NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.constitucion_entrada e JOIN vec_bolsa_llamamientos.constitucion c USING (instantanea_ref, version_instantanea) WHERE e.participacion_ref = p_participacion_ref AND c.bolsa_ref = p_bolsa_ref) THEN
  RAISE EXCEPTION USING ERRCODE='23503', MESSAGE='participacion ajena a la bolsa';
 END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:datos_contacto:' || p_participacion_ref, 0));
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_datos_contacto_participacion_v3_atestada(p_capacidad, p_decision, p_motivo_autorizacion, p_contexto, p_persona_version, p_perfil_version, p_payload, p_sobre, p_evidencia, p_raiz);
 BEGIN decision := convert_from(p_decision, 'UTF8')::jsonb; EXCEPTION WHEN others THEN RAISE EXCEPTION USING ERRCODE='42501', MESSAGE='datos de contacto no autorizados'; END;
 IF consumo.efecto_ref IS DISTINCT FROM p_participacion_ref OR consumo.consumo_nuevo IS NOT TRUE
    OR decision->>'principal_id' IS DISTINCT FROM p_actor OR decision->>'accion' IS DISTINCT FROM 'bolsa.datos_contacto_participacion.registrar'
    OR decision->>'modulo_id' IS DISTINCT FROM 'bolsa' OR decision->>'tipo_recurso' IS DISTINCT FROM 'participacion_bolsa'
    OR decision->>'finalidad' IS DISTINCT FROM 'gestion_datos_contacto_participacion' OR decision->>'recurso_ref' IS DISTINCT FROM p_participacion_ref
    OR decision->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb OR decision->'obligaciones' IS DISTINCT FROM '[]'::jsonb
    OR consumo.huella_efecto_sha256 IS DISTINCT FROM decision->>'contexto_recurso_huella_sha256' THEN
  RAISE EXCEPTION USING ERRCODE='42501', MESSAGE='datos de contacto no autorizados';
 END IF;
 -- Replay: la aplicación ya comparó el claro; aquí basta devolver el recibo original.
 IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.datos_contacto_participacion d WHERE d.participacion_ref = p_participacion_ref AND d.clave_idempotencia = p_clave_idempotencia) THEN
  RETURN QUERY SELECT true, d.recibo_ref, d.version, d.registrada_en FROM vec_bolsa_llamamientos.datos_contacto_participacion d WHERE d.participacion_ref = p_participacion_ref AND d.clave_idempotencia = p_clave_idempotencia;
  RETURN;
 END IF;
 SELECT coalesce(max(d.version), 0) INTO v_vigente FROM vec_bolsa_llamamientos.datos_contacto_participacion d WHERE d.participacion_ref = p_participacion_ref;
 IF p_version <> v_vigente + 1 THEN
  RAISE EXCEPTION USING ERRCODE='VBS02', MESSAGE='version de datos de contacto no consecutiva';
 END IF;
 INSERT INTO vec_bolsa_llamamientos.datos_contacto_participacion (participacion_ref, version, clave_ref, nonce, cifrado, motivo, actor, registrada_en, clave_idempotencia, recibo_ref)
 VALUES (p_participacion_ref, p_version, p_clave_ref, p_nonce, p_cifrado, p_motivo, p_actor, p_registrada_en, p_clave_idempotencia, p_recibo_ref);
 RETURN QUERY SELECT false, p_recibo_ref, p_version, p_registrada_en;
END $f$;

CREATE FUNCTION vec_bolsa_llamamientos.leer_datos_contacto_participacion_v1(p_participacion_ref text)
RETURNS TABLE(version bigint, clave_ref text, nonce bytea, cifrado bytea, motivo text, registrada_en timestamptz, recibo_ref text)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = pg_catalog AS $f$
 SELECT d.version, d.clave_ref, d.nonce, d.cifrado, d.motivo, d.registrada_en, d.recibo_ref
   FROM vec_bolsa_llamamientos.datos_contacto_participacion d
  WHERE d.participacion_ref = p_participacion_ref
  ORDER BY d.version DESC LIMIT 1
$f$;

CREATE FUNCTION vec_bolsa_llamamientos.recuperar_datos_contacto_participacion_v1(p_participacion_ref text, p_clave_idempotencia text)
RETURNS TABLE(version bigint, clave_ref text, nonce bytea, cifrado bytea, motivo text, registrada_en timestamptz, recibo_ref text)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = pg_catalog AS $f$
 SELECT d.version, d.clave_ref, d.nonce, d.cifrado, d.motivo, d.registrada_en, d.recibo_ref
   FROM vec_bolsa_llamamientos.datos_contacto_participacion d
  WHERE d.participacion_ref = p_participacion_ref AND d.clave_idempotencia = p_clave_idempotencia
$f$;

REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.registrar_datos_contacto_participacion_v1(text,text,bigint,text,bytea,bytea,text,text,timestamptz,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.leer_datos_contacto_participacion_v1(text) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.recuperar_datos_contacto_participacion_v1(text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.registrar_datos_contacto_participacion_v1(text,text,bigint,text,bytea,bytea,text,text,timestamptz,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_ejecutor;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.leer_datos_contacto_participacion_v1(text) TO vec_bolsa_llamamientos_ejecutor;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.recuperar_datos_contacto_participacion_v1(text,text) TO vec_bolsa_llamamientos_ejecutor;
-- FIN deploy/postgresql/bolsa_llamamientos/migraciones/000016_datos_contacto_participacion.up.sql
-- INICIO deploy/postgresql/autorizacion_atestada_v3/migraciones/000048_consumidor_emision_llamamiento.up.sql
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000048',0));
DO $audiencias$ DECLARE d text; n text; BEGIN
 SELECT pg_get_constraintdef(oid,true) INTO STRICT d FROM pg_constraint WHERE conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass AND conname='clave_capacidad_version_audiencia_consumo_check';
 IF strpos(d,'''vec_bolsa_llamamientos.llamamiento.emitir.v1''::text')<>0 THEN RAISE EXCEPTION 'audiencia B7 ya presente' USING ERRCODE='55000'; END IF;
 IF strpos(d,'CHECK (audiencia_consumo = ANY (ARRAY[')<>1 OR right(d,3)<>']))' THEN RAISE EXCEPTION 'catálogo de audiencias incompatible' USING ERRCODE='55000'; END IF;
 n:=left(d,length(d)-3)||', ''vec_bolsa_llamamientos.llamamiento.emitir.v1''::text]))';
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||n;
END $audiencias$;
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_emision_llamamiento_v3_atestada(p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; x record; BEGIN
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb; EXCEPTION WHEN others THEN RAISE EXCEPTION 'material B7 inválido' USING ERRCODE='22023'; END;
 IF c->>'audiencia_consumo' IS DISTINCT FROM 'vec_bolsa_llamamientos.llamamiento.emitir.v1' OR c->>'operacion' IS DISTINCT FROM 'llamamiento.emitir.v1' OR d->>'accion' IS DISTINCT FROM c->>'operacion' OR d->>'modulo_id' IS DISTINCT FROM 'bolsa' OR d->>'tipo_recurso' IS DISTINCT FROM 'bolsa_constituida' OR d->>'finalidad' IS DISTINCT FROM 'gestion_llamamientos_bolsa' OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref' OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256' THEN RAISE EXCEPTION 'material B7 rechazado' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna('emision_llamamiento_bolsa',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'B7 requiere consumo nuevo' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;
ALTER FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_emision_llamamiento_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) OWNER TO vec_autorizacion_atestada_v3_propietario;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_emision_llamamiento_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_emision_llamamiento_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_propietario;
REVOKE GRANT OPTION FOR EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_emision_llamamiento_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM vec_bolsa_llamamientos_propietario;
-- FIN deploy/postgresql/autorizacion_atestada_v3/migraciones/000048_consumidor_emision_llamamiento.up.sql
-- INICIO deploy/postgresql/bolsa_llamamientos/migraciones/000017_emision_llamamiento.up.sql
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000017',0));
DO $pre$ BEGIN
 IF to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_emision_llamamiento_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL OR to_regclass('vec_bolsa_llamamientos.contacto_participacion') IS NULL OR to_regclass('vec_bolsa_llamamientos.datos_contacto_participacion') IS NULL THEN RAISE EXCEPTION 'dependencias B7 ausentes' USING ERRCODE='55000'; END IF;
END $pre$;
DO $bitacora$
DECLARE accion text; ruta text;
BEGIN
 SELECT pg_get_constraintdef(oid,true) INTO STRICT accion FROM pg_constraint WHERE conrelid='vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento'::regclass AND conname='bitacora_intento_borrador_llamamiento_accion_check' AND contype='c';
 SELECT pg_get_constraintdef(oid,true) INTO STRICT ruta FROM pg_constraint WHERE conrelid='vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento'::regclass AND conname='bitacora_intento_borrador_llamamiento_ruta_clase_check' AND contype='c';
 IF strpos(accion,'''consultar_datos_contacto''')<>0 OR strpos(accion,'''emitir_llamamiento''')<>0 OR strpos(accion,'''recuperar_llamamiento''')<>0 OR strpos(ruta,'''emisiones''')<>0 OR strpos(accion,'CHECK (accion = ANY (ARRAY[')<>1 OR right(accion,3)<>']))' OR strpos(ruta,'CHECK (ruta_clase = ANY (ARRAY[')<>1 OR right(ruta,3)<>']))' THEN RAISE EXCEPTION 'bitácora de frontera incompatible con B7' USING ERRCODE='55000'; END IF;
 ALTER TABLE vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento DROP CONSTRAINT bitacora_intento_borrador_llamamiento_accion_check;
 EXECUTE 'ALTER TABLE vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento ADD CONSTRAINT bitacora_intento_borrador_llamamiento_accion_check '||left(accion,length(accion)-3)||', ''consultar_datos_contacto''::text, ''emitir_llamamiento''::text, ''recuperar_llamamiento''::text]))';
 ALTER TABLE vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento DROP CONSTRAINT bitacora_intento_borrador_llamamiento_ruta_clase_check;
 EXECUTE 'ALTER TABLE vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento ADD CONSTRAINT bitacora_intento_borrador_llamamiento_ruta_clase_check '||left(ruta,length(ruta)-3)||', ''emisiones''::text]))';
END $bitacora$;
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.registrar_intento_borrador_llamamiento_v1(p_correlacion_ref text,p_accion text,p_ruta_clase text,p_actor_ref text,p_resultado text) RETURNS void LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE v_ref text; BEGIN
 PERFORM vec_bolsa_llamamientos.exigir_runtime_registrador_frontera_borrador_llamamiento();
 IF p_correlacion_ref IS NULL OR p_correlacion_ref !~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{7,191}$' OR p_accion NOT IN('crear','consultar','cambiar_situacion','registrar_contacto','consultar_datos_contacto','registrar_datos_contacto','emitir_llamamiento','recuperar_llamamiento') OR p_ruta_clase NOT IN('coleccion','detalle','situacion','contactos','datos_contacto','emisiones') OR (p_actor_ref IS NOT NULL AND p_actor_ref !~ '^per_[A-Za-z0-9_-]{22,128}$') OR p_resultado NOT IN('autenticacion_requerida','acceso_denegado','recurso_no_disponible','infraestructura_no_disponible','resultado_indeterminado') THEN RAISE EXCEPTION 'intento de frontera Bolsa inválido' USING ERRCODE='22023'; END IF;
 v_ref:='intento:'||translate(encode(sha256(convert_to(p_correlacion_ref||':'||p_accion||':'||p_ruta_clase||':'||coalesce(p_actor_ref,'')||':'||p_resultado||':'||clock_timestamp()::text,'UTF8')),'hex'),'0123456789','ghijklmnop');
 INSERT INTO vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento(intento_ref,correlacion_ref,accion,ruta_clase,actor_ref,resultado,registrada_en) VALUES(v_ref,p_correlacion_ref,p_accion,p_ruta_clase,p_actor_ref,p_resultado,clock_timestamp());
END $f$;
ALTER TABLE vec_bolsa_llamamientos.contacto_participacion DROP CONSTRAINT contacto_participacion_resultado_check;
ALTER TABLE vec_bolsa_llamamientos.contacto_participacion ADD CONSTRAINT contacto_participacion_resultado_check CHECK(resultado IN('contactado','no_contesta','buzon','acepta','rechaza','aplazado','otro','enviado','no_enviado'));
CREATE TABLE vec_bolsa_llamamientos.llamamiento_emitido(
 llamamiento_ref text PRIMARY KEY,
 recibo_ref text NOT NULL UNIQUE,
 bolsa_ref text NOT NULL,
 actor_ref text NOT NULL,
 clave_idempotencia text NOT NULL,
 participaciones jsonb NOT NULL,
 configuracion jsonb NOT NULL,
 huella_comando_sha256 text NOT NULL,
 estado text NOT NULL CHECK(estado='emision_reservada'),
 emitido_en timestamptz(6) NOT NULL,
 decision_ref text NOT NULL UNIQUE,
 UNIQUE(bolsa_ref,clave_idempotencia),
 CHECK(llamamiento_ref ~ '^llamamiento:[0-9a-f]{64}$' AND recibo_ref ~ '^recibo:llamamiento:[0-9a-f]{64}$'),
 CHECK(actor_ref ~ '^per_[A-Za-z0-9_-]{22,128}$'),
 CHECK(jsonb_typeof(participaciones)='array' AND jsonb_array_length(participaciones) BETWEEN 1 AND 100),
 CHECK(jsonb_typeof(configuracion)='object'),
 CHECK(huella_comando_sha256 ~ '^[0-9a-f]{64}$')
);
ALTER TABLE vec_bolsa_llamamientos.llamamiento_emitido ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.llamamiento_emitido FORCE ROW LEVEL SECURITY;
CREATE POLICY llamamiento_emitido_solo_propietario ON vec_bolsa_llamamientos.llamamiento_emitido TO vec_bolsa_llamamientos_propietario USING(current_user='vec_bolsa_llamamientos_propietario') WITH CHECK(current_user='vec_bolsa_llamamientos_propietario');
REVOKE ALL ON vec_bolsa_llamamientos.llamamiento_emitido FROM PUBLIC;
CREATE TRIGGER llamamiento_emitido_inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.llamamiento_emitido FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();
CREATE FUNCTION vec_bolsa_llamamientos.reservar_llamamiento_v1(p_llamamiento text,p_recibo text,p_bolsa text,p_actor text,p_clave text,p_participaciones jsonb,p_configuracion jsonb,p_emitido timestamptz,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(emision jsonb,reutilizada boolean) LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE previo record; consumo record; d jsonb; v_huella text; v_total int; v_validas int; v_ordenadas int; v_contactos jsonb; BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR p_llamamiento !~ '^llamamiento:[0-9a-f]{64}$' OR p_recibo !~ '^recibo:llamamiento:[0-9a-f]{64}$' OR p_actor !~ '^per_[A-Za-z0-9_-]{22,128}$' OR p_clave IS NULL OR p_clave<>btrim(p_clave) OR octet_length(p_clave) NOT BETWEEN 8 AND 256 OR jsonb_typeof(p_participaciones)<>'array' OR jsonb_array_length(p_participaciones) NOT BETWEEN 1 AND 100 OR NOT(SELECT bool_and(jsonb_typeof(value)='string') FROM jsonb_array_elements(p_participaciones)) OR jsonb_typeof(p_configuracion)<>'object' OR p_configuracion ?& ARRAY['referencia','descripcion','categoria','centro','modalidad','fecha_inicio','plazo','plantilla_version','asunto','cuerpo'] IS NOT TRUE OR NOT(SELECT count(*)=10 AND bool_and(jsonb_typeof(value)='string' AND octet_length(value#>>'{}') BETWEEN 2 AND 4000 AND value#>>'{}'=btrim(value#>>'{}')) FROM jsonb_each(p_configuracion)) OR octet_length(p_configuracion->>'plantilla_version')>900 OR p_emitido IS NULL THEN RAISE EXCEPTION 'emisión B7 inválida' USING ERRCODE='22023'; END IF;
 v_huella:=encode(sha256(convert_to(p_bolsa||chr(31)||p_participaciones::text||chr(31)||p_configuracion::text,'UTF8')),'hex');
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:emision:'||p_bolsa||':'||p_clave,0));
 SELECT * INTO previo FROM vec_bolsa_llamamientos.llamamiento_emitido WHERE bolsa_ref=p_bolsa AND clave_idempotencia=p_clave;
 IF FOUND THEN
  IF previo.actor_ref<>p_actor OR previo.huella_comando_sha256<>v_huella THEN RAISE EXCEPTION 'clave B7 divergente' USING ERRCODE='VBE01'; END IF;
  SELECT coalesce(jsonb_agg(jsonb_build_object('participacion_ref',c.participacion_ref,'resultado',c.resultado,'recibo_ref',c.recibo_ref) ORDER BY x.ordinality),'[]'::jsonb) INTO v_contactos FROM jsonb_array_elements_text(previo.participaciones) WITH ORDINALITY x(ref,ordinality) JOIN vec_bolsa_llamamientos.contacto_participacion c ON c.participacion_ref=x.ref AND c.llamamiento_ref=previo.llamamiento_ref;
  RETURN QUERY SELECT jsonb_build_object('llamamiento_ref',previo.llamamiento_ref,'recibo_ref',previo.recibo_ref,'bolsa_ref',previo.bolsa_ref,'estado',CASE WHEN jsonb_array_length(v_contactos)=jsonb_array_length(previo.participaciones) THEN 'emitido_pendiente_respuesta' ELSE 'emision_reservada_resultado_pendiente' END,'participaciones',previo.participaciones,'configuracion',previo.configuracion,'emitido_en',previo.emitido_en,'contactos',v_contactos),true; RETURN;
 END IF;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_emision_llamamiento_v3_atestada(p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 BEGIN d:=convert_from(p_decision,'UTF8')::jsonb; EXCEPTION WHEN others THEN RAISE EXCEPTION 'decisión B7 inválida' USING ERRCODE='42501'; END;
 IF consumo.efecto_ref IS DISTINCT FROM p_bolsa OR consumo.consumo_nuevo IS NOT TRUE OR d->>'principal_id' IS DISTINCT FROM p_actor OR d->>'accion' IS DISTINCT FROM 'llamamiento.emitir.v1' OR d->>'tipo_recurso' IS DISTINCT FROM 'bolsa_constituida' THEN RAISE EXCEPTION 'emisión B7 no autorizada' USING ERRCODE='42501'; END IF;
 IF NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.constitucion c JOIN vec_bolsa_llamamientos.bolsa_constituida b ON b.bolsa_ref=c.bolsa_ref AND b.version=c.version_bolsa WHERE c.bolsa_ref=p_bolsa AND b.estado='vigente' AND b.vigente_desde<=p_emitido AND (b.vigente_hasta IS NULL OR p_emitido<b.vigente_hasta)) THEN RAISE EXCEPTION 'bolsa no vigente' USING ERRCODE='23503'; END IF;
 v_total:=jsonb_array_length(p_participaciones);
 SELECT count(*) INTO v_validas FROM jsonb_array_elements_text(p_participaciones) x(ref) JOIN vec_bolsa_llamamientos.constitucion c ON c.bolsa_ref=p_bolsa JOIN vec_bolsa_llamamientos.constitucion_entrada e ON e.instantanea_ref=c.instantanea_ref AND e.version_instantanea=c.version_instantanea AND e.participacion_ref=x.ref;
 SELECT count(*) INTO v_ordenadas FROM (SELECT e.orden,lag(e.orden) OVER(ORDER BY x.n) anterior FROM jsonb_array_elements_text(p_participaciones) WITH ORDINALITY x(ref,n) JOIN vec_bolsa_llamamientos.constitucion c ON c.bolsa_ref=p_bolsa JOIN vec_bolsa_llamamientos.constitucion_entrada e ON e.instantanea_ref=c.instantanea_ref AND e.version_instantanea=c.version_instantanea AND e.participacion_ref=x.ref) q WHERE anterior IS NULL OR orden>anterior;
 IF v_validas<>v_total OR v_ordenadas<>v_total OR (SELECT count(DISTINCT value) FROM jsonb_array_elements_text(p_participaciones))<>v_total THEN RAISE EXCEPTION 'participaciones ajenas, repetidas o desordenadas' USING ERRCODE='22023'; END IF;
 INSERT INTO vec_bolsa_llamamientos.llamamiento_emitido VALUES(p_llamamiento,p_recibo,p_bolsa,p_actor,p_clave,p_participaciones,p_configuracion,v_huella,'emision_reservada',p_emitido,consumo.decision_ref);
 RETURN QUERY SELECT jsonb_build_object('llamamiento_ref',p_llamamiento,'recibo_ref',p_recibo,'bolsa_ref',p_bolsa,'estado','emision_reservada_resultado_pendiente','participaciones',p_participaciones,'configuracion',p_configuracion,'emitido_en',p_emitido,'contactos','[]'::jsonb),false;
END $f$;
CREATE FUNCTION vec_bolsa_llamamientos.registrar_contactos_llamamiento_v1(p_bolsa text,p_clave text,p_actor text,p_contactos jsonb) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE previo record; v_contactos jsonb; BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR p_actor !~ '^per_[A-Za-z0-9_-]{22,128}$' OR jsonb_typeof(p_contactos)<>'array' THEN RAISE EXCEPTION 'resultado B7 inválido' USING ERRCODE='22023'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:emision:'||p_bolsa||':'||p_clave,0));
 SELECT * INTO previo FROM vec_bolsa_llamamientos.llamamiento_emitido WHERE bolsa_ref=p_bolsa AND clave_idempotencia=p_clave;
 IF NOT FOUND OR previo.actor_ref<>p_actor OR jsonb_array_length(p_contactos)<>jsonb_array_length(previo.participaciones) OR NOT(SELECT bool_and(jsonb_typeof(c.value)='object' AND (SELECT count(*)=3 FROM jsonb_object_keys(c.value)) AND c.value->>'participacion_ref'=previo.participaciones->>(c.ordinality-1)::int AND c.value->>'resultado' IN('enviado','no_enviado') AND c.value->>'recibo_ref' ~ '^recibo:contacto:[0-9a-f]{64}$') FROM jsonb_array_elements(p_contactos) WITH ORDINALITY c(value,ordinality)) THEN RAISE EXCEPTION 'resultado B7 divergente' USING ERRCODE='VBE01'; END IF;
 SELECT coalesce(jsonb_agg(jsonb_build_object('participacion_ref',c.participacion_ref,'resultado',c.resultado,'recibo_ref',c.recibo_ref) ORDER BY x.ordinality),'[]'::jsonb) INTO v_contactos FROM jsonb_array_elements_text(previo.participaciones) WITH ORDINALITY x(ref,ordinality) JOIN vec_bolsa_llamamientos.contacto_participacion c ON c.participacion_ref=x.ref AND c.llamamiento_ref=previo.llamamiento_ref;
 IF jsonb_array_length(v_contactos)=jsonb_array_length(previo.participaciones) THEN RETURN jsonb_build_object('llamamiento_ref',previo.llamamiento_ref,'recibo_ref',previo.recibo_ref,'bolsa_ref',previo.bolsa_ref,'estado','emitido_pendiente_respuesta','participaciones',previo.participaciones,'configuracion',previo.configuracion,'emitido_en',previo.emitido_en,'contactos',v_contactos); END IF;
 IF jsonb_array_length(v_contactos)<>0 THEN RAISE EXCEPTION 'resultado B7 parcial' USING ERRCODE='55000'; END IF;
 INSERT INTO vec_bolsa_llamamientos.contacto_participacion(contacto_ref,bolsa_ref,participacion_ref,llamamiento_ref,canal,instante,actor,resultado,anotacion,clave_idempotencia,recibo_ref)
 SELECT 'contacto:'||encode(sha256(convert_to(p_bolsa||chr(31)||p_clave||chr(31)||(c.value->>'participacion_ref'),'UTF8')),'hex'),p_bolsa,c.value->>'participacion_ref',previo.llamamiento_ref,'correo',previo.emitido_en,p_actor,c.value->>'resultado','Emisión de llamamiento por plantilla '||(previo.configuracion->>'plantilla_version'),p_clave||':correo:'||c.ordinality,c.value->>'recibo_ref' FROM jsonb_array_elements(p_contactos) WITH ORDINALITY c(value,ordinality);
 RETURN jsonb_build_object('llamamiento_ref',previo.llamamiento_ref,'recibo_ref',previo.recibo_ref,'bolsa_ref',previo.bolsa_ref,'estado','emitido_pendiente_respuesta','participaciones',previo.participaciones,'configuracion',previo.configuracion,'emitido_en',previo.emitido_en,'contactos',p_contactos);
END $f$;
CREATE FUNCTION vec_bolsa_llamamientos.recuperar_llamamiento_emitido_v1(p_bolsa text,p_clave text) RETURNS jsonb LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT jsonb_build_object('llamamiento_ref',l.llamamiento_ref,'recibo_ref',l.recibo_ref,'bolsa_ref',l.bolsa_ref,'estado',CASE WHEN (SELECT count(*) FROM vec_bolsa_llamamientos.contacto_participacion c WHERE c.llamamiento_ref=l.llamamiento_ref)=jsonb_array_length(l.participaciones) THEN 'emitido_pendiente_respuesta' ELSE 'emision_reservada_resultado_pendiente' END,'participaciones',l.participaciones,'configuracion',l.configuracion,'emitido_en',l.emitido_en,'contactos',coalesce((SELECT jsonb_agg(jsonb_build_object('participacion_ref',c.participacion_ref,'resultado',c.resultado,'recibo_ref',c.recibo_ref) ORDER BY x.ordinality) FROM jsonb_array_elements_text(l.participaciones) WITH ORDINALITY x(ref,ordinality) JOIN vec_bolsa_llamamientos.contacto_participacion c ON c.participacion_ref=x.ref AND c.llamamiento_ref=l.llamamiento_ref),'[]'::jsonb)) FROM vec_bolsa_llamamientos.llamamiento_emitido l WHERE l.bolsa_ref=p_bolsa AND l.clave_idempotencia=p_clave
$f$;
CREATE FUNCTION vec_bolsa_llamamientos.contar_llamamientos_en_curso_v1(p_bolsa text) RETURNS bigint LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog AS $f$ SELECT count(*) FROM vec_bolsa_llamamientos.llamamiento_emitido l WHERE l.bolsa_ref=p_bolsa AND (SELECT count(*) FROM vec_bolsa_llamamientos.contacto_participacion c WHERE c.llamamiento_ref=l.llamamiento_ref)=jsonb_array_length(l.participaciones) $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.reservar_llamamiento_v1(text,text,text,text,text,jsonb,jsonb,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),vec_bolsa_llamamientos.registrar_contactos_llamamiento_v1(text,text,text,jsonb),vec_bolsa_llamamientos.recuperar_llamamiento_emitido_v1(text,text),vec_bolsa_llamamientos.contar_llamamientos_en_curso_v1(text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.reservar_llamamiento_v1(text,text,text,text,text,jsonb,jsonb,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),vec_bolsa_llamamientos.registrar_contactos_llamamiento_v1(text,text,text,jsonb),vec_bolsa_llamamientos.recuperar_llamamiento_emitido_v1(text,text),vec_bolsa_llamamientos.contar_llamamientos_en_curso_v1(text) TO vec_bolsa_llamamientos_ejecutor;
-- FIN deploy/postgresql/bolsa_llamamientos/migraciones/000017_emision_llamamiento.up.sql
:finalizar;
