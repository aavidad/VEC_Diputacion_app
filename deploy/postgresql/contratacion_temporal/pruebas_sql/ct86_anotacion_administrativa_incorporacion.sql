\set ON_ERROR_STOP on
\set VERBOSITY terse
-- HARNESS PREPARADO, NO EJECUTADO: sólo clon desechable con datos sintéticos,
-- nombre vec_ct86_prueba_<12 hex>, CT75/CT86/AD3-30 previamente instaladas.
-- No reaplica migraciones ni modifica ACL producto. Dos CONEXIONES REALES:
-- 1. Administrador del clon: ct86_preparar=true y ct86_login_ejecutor=<rol de
--    prueba ya provisionado>. Crea observador sólo lectura de SQL FIJO y trigger
--    de fallo, ambos exclusivos de este clon. COMMIT sólo de auxiliares test.
-- 2. Login ejecutor real: ct86_preparar=false (predeterminado). No SET ROLE ni
--    SET SESSION AUTHORIZATION. Sus consumos y efectos viven en una sola
--    transacción SERIALIZABLE que siempre termina con ROLLBACK.
-- El runner abre ambas conexiones por su canal privado, sin credenciales en
-- este archivo. No basta conectar como admin y después hacer SET ROLE.
-- Antes de cada inclusión, la sesión correspondiente aporta:
--   CREATE TEMP TABLE ct86_entorno(base name PRIMARY KEY, solo_sinteticos boolean);
--   INSERT ... VALUES (current_database(), true);
-- Además, la sesión EJECUTORA aporta:
--   CREATE TEMP TABLE ct86_vectores(
--     caso text PRIMARY KEY, peticion jsonb NOT NULL,
--     capacidad bytea NOT NULL, decision bytea NOT NULL, motivo bytea NOT NULL,
--     contexto bytea NOT NULL, persona_version numeric NOT NULL,
--     perfil_version numeric NOT NULL, payload bytea NOT NULL, sobre bytea NOT NULL,
--     evidencia bytea NOT NULL, raiz bytea NOT NULL);
-- Dos filas EXACTAS inicial/replay, diez piezas V3 reales por fila, decisiones
-- y nonce distintos, aún sin consumir, fuentes y claves sintéticas vigentes.
-- No insertar consumos/atestaciones a mano ni sustituir consumidores o cripto.
-- peticion es JSON exacto producido por dominio/adaptador Go: material,
-- expediente_anterior, expediente_siguiente, seguimiento_original,
-- estado_seguimiento_sha256, recibo_incorporacion_ref, ambito_idempotencia_hmac,
-- huella_peticion_hmac, referencias, politica, instante_efecto. Antecedente CT75
-- real 0->1, versión actual de expediente, todavía sin anotación.
-- INICIAL: observaciones de 2000 caracteres NFC y más de 4096 bytes UTF8
-- (ejemplo de contenido sintético: repetir un carácter de tres bytes). Firmar
-- ese material de verdad en el runner; no parchear el vector después de firmar.
-- Replay mantiene material/sellos/origen/preimagen; nueva autorización temporal.
-- Puede tener expediente_siguiente null. El test exige recibo idéntico.
-- Observador sin parámetros, sin SQL dinámico, sólo SELECT de 22 tablas fijas
-- listadas abajo; no es un consumidor ni modifica producto. Permite observar
-- desde la transacción ejecutora sus efectos sincommit bajo RLS. No acredita
-- preservación de tablas fuera de esta lista: revisar inventario aparte como
-- administrador. Sólo la preparación administrativa instala el trigger de test.
-- No echo ni impresión de vectores, certificados, recibos o snapshots.
-- En error: ON_ERROR_STOP aborta; cerrar sesión o ROLLBACK. Desechar el clon
-- tras la prueba, incluidos los auxiliares test; nunca instalarlos en otra base.
\if :{?ct86_preparar}
\else
\set ct86_preparar false
\endif
\if :ct86_preparar
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='30s';
-- La variable es un nombre de rol, no una conexión ni una credencial.
CREATE TEMP TABLE ct86_login_preparacion AS SELECT :'ct86_login_ejecutor'::name AS login;
DO $preparar_guard$
DECLARE login name;
BEGIN
 SELECT x.login INTO STRICT login FROM pg_temp.ct86_login_preparacion x;
 IF current_database() !~ '^vec_ct86_prueba_[a-f0-9]{12}$'
  OR current_user<>session_user
  OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=session_user AND rolsuper)
  OR to_regnamespace('ct86_prueba') IS NOT NULL
  OR to_regclass('pg_temp.ct86_entorno') IS NULL THEN
  RAISE EXCEPTION 'CT86: preparación sólo en clon nuevo por administrador separado'; END IF;
 IF (SELECT count(*) FROM pg_temp.ct86_entorno)<>1 OR NOT EXISTS(
  SELECT 1 FROM pg_temp.ct86_entorno WHERE base=current_database() AND solo_sinteticos IS TRUE) THEN
  RAISE EXCEPTION 'CT86: clon no identificado'; END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=login AND rolcanlogin
  AND NOT rolsuper AND NOT rolbypassrls AND NOT rolcreaterole AND NOT rolcreatedb AND NOT rolreplication)
  OR NOT pg_has_role(login,'vec_contratacion_temporal_ejecutor','MEMBER')
  OR pg_has_role(login,'vec_contratacion_temporal_propietario','MEMBER')
  OR pg_has_role(login,'vec_contratacion_temporal_migrador','MEMBER')
  OR pg_has_role(login,'vec_autorizacion_atestada_v3_propietario','MEMBER') THEN
  RAISE EXCEPTION 'CT86: login de prueba incompatible con frontera V3'; END IF;
END $preparar_guard$;
CREATE SCHEMA ct86_prueba;
REVOKE ALL ON SCHEMA ct86_prueba FROM PUBLIC;
CREATE TABLE ct86_prueba.entorno(base name PRIMARY KEY, login name NOT NULL);
INSERT INTO ct86_prueba.entorno SELECT current_database(),login FROM pg_temp.ct86_login_preparacion;
REVOKE ALL ON TABLE ct86_prueba.entorno FROM PUBLIC;

CREATE FUNCTION ct86_prueba.snapshot() RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=off AS $f$
DECLARE resultado jsonb;
BEGIN
 IF current_database() !~ '^vec_ct86_prueba_[a-f0-9]{12}$'
  OR NOT EXISTS(SELECT 1 FROM ct86_prueba.entorno WHERE base=current_database() AND login=session_user)
  OR EXISTS(SELECT 1 FROM pg_roles WHERE rolname=session_user AND (rolsuper OR rolbypassrls))
  OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
  OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
  OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER') THEN
  RAISE EXCEPTION 'CT86: observador fuera de su sesión ejecutora sintética'; END IF;
 SELECT jsonb_build_object(
  'vec_contratacion_temporal.anotacion_administrativa_incorporacion_v1',(SELECT coalesce(jsonb_agg(to_jsonb(t) ORDER BY to_jsonb(t)::text COLLATE "C"),'[]'::jsonb) FROM vec_contratacion_temporal.anotacion_administrativa_incorporacion_v1 t),
  'vec_contratacion_temporal.expediente_version_integral',(SELECT coalesce(jsonb_agg(to_jsonb(t) ORDER BY to_jsonb(t)::text COLLATE "C"),'[]'::jsonb) FROM vec_contratacion_temporal.expediente_version_integral t),
  'vec_contratacion_temporal.expediente_integral_actual',(SELECT coalesce(jsonb_agg(to_jsonb(t) ORDER BY to_jsonb(t)::text COLLATE "C"),'[]'::jsonb) FROM vec_contratacion_temporal.expediente_integral_actual t),
  'vec_contratacion_temporal.actuacion_expediente_integral',(SELECT coalesce(jsonb_agg(to_jsonb(t) ORDER BY to_jsonb(t)::text COLLATE "C"),'[]'::jsonb) FROM vec_contratacion_temporal.actuacion_expediente_integral t),
  'vec_contratacion_temporal.outbox_expediente_integral',(SELECT coalesce(jsonb_agg(to_jsonb(t) ORDER BY to_jsonb(t)::text COLLATE "C"),'[]'::jsonb) FROM vec_contratacion_temporal.outbox_expediente_integral t),
  'vec_contratacion_temporal.control_cadenas_expediente_integral',(SELECT coalesce(jsonb_agg(to_jsonb(t) ORDER BY to_jsonb(t)::text COLLATE "C"),'[]'::jsonb) FROM vec_contratacion_temporal.control_cadenas_expediente_integral t),
  'vec_contratacion_temporal.incorporacion_registro_v2',(SELECT coalesce(jsonb_agg(to_jsonb(t) ORDER BY to_jsonb(t)::text COLLATE "C"),'[]'::jsonb) FROM vec_contratacion_temporal.incorporacion_registro_v2 t),
  'vec_contratacion_temporal.incorporacion_auditoria_v2',(SELECT coalesce(jsonb_agg(to_jsonb(t) ORDER BY to_jsonb(t)::text COLLATE "C"),'[]'::jsonb) FROM vec_contratacion_temporal.incorporacion_auditoria_v2 t),
  'vec_contratacion_temporal.incorporacion_outbox_v2',(SELECT coalesce(jsonb_agg(to_jsonb(t) ORDER BY to_jsonb(t)::text COLLATE "C"),'[]'::jsonb) FROM vec_contratacion_temporal.incorporacion_outbox_v2 t),
  'vec_contratacion_temporal.seguimiento_definicion_v2',(SELECT coalesce(jsonb_agg(to_jsonb(t) ORDER BY to_jsonb(t)::text COLLATE "C"),'[]'::jsonb) FROM vec_contratacion_temporal.seguimiento_definicion_v2 t),
  'vec_contratacion_temporal.seguimiento_raiz_v2',(SELECT coalesce(jsonb_agg(to_jsonb(t) ORDER BY to_jsonb(t)::text COLLATE "C"),'[]'::jsonb) FROM vec_contratacion_temporal.seguimiento_raiz_v2 t),
  'vec_contratacion_temporal.seguimiento_estado_v2',(SELECT coalesce(jsonb_agg(to_jsonb(t) ORDER BY to_jsonb(t)::text COLLATE "C"),'[]'::jsonb) FROM vec_contratacion_temporal.seguimiento_estado_v2 t),
  'vec_personal.relacion_alta_ejercicio',(SELECT coalesce(jsonb_agg(to_jsonb(t) ORDER BY to_jsonb(t)::text COLLATE "C"),'[]'::jsonb) FROM vec_personal.relacion_alta_ejercicio t),
  'vec_personal.ocupacion_alta_ejercicio',(SELECT coalesce(jsonb_agg(to_jsonb(t) ORDER BY to_jsonb(t)::text COLLATE "C"),'[]'::jsonb) FROM vec_personal.ocupacion_alta_ejercicio t),
  'vec_personal.registro_alta_ejercicio',(SELECT coalesce(jsonb_agg(to_jsonb(t) ORDER BY to_jsonb(t)::text COLLATE "C"),'[]'::jsonb) FROM vec_personal.registro_alta_ejercicio t),
  'vec_personal.auditoria_alta_ejercicio',(SELECT coalesce(jsonb_agg(to_jsonb(t) ORDER BY to_jsonb(t)::text COLLATE "C"),'[]'::jsonb) FROM vec_personal.auditoria_alta_ejercicio t),
  'vec_personal.outbox_alta_ejercicio',(SELECT coalesce(jsonb_agg(to_jsonb(t) ORDER BY to_jsonb(t)::text COLLATE "C"),'[]'::jsonb) FROM vec_personal.outbox_alta_ejercicio t),
  'vec_personal.auditoria_lectura_incorporacion',(SELECT coalesce(jsonb_agg(to_jsonb(t) ORDER BY to_jsonb(t)::text COLLATE "C"),'[]'::jsonb) FROM vec_personal.auditoria_lectura_incorporacion t),
  'vec_autorizacion_atestada_v3.atestacion_decision_v3',(SELECT coalesce(jsonb_agg(to_jsonb(t) ORDER BY to_jsonb(t)::text COLLATE "C"),'[]'::jsonb) FROM vec_autorizacion_atestada_v3.atestacion_decision_v3 t),
  'vec_autorizacion_atestada_v3.consumo_decision_v3',(SELECT coalesce(jsonb_agg(to_jsonb(t) ORDER BY to_jsonb(t)::text COLLATE "C"),'[]'::jsonb) FROM vec_autorizacion_atestada_v3.consumo_decision_v3 t),
  'vec_autorizacion_atestada_v3.auditoria_consumo_v3',(SELECT coalesce(jsonb_agg(to_jsonb(t) ORDER BY to_jsonb(t)::text COLLATE "C"),'[]'::jsonb) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 t),
  'vec_autorizacion_atestada_v3.control_cadena_auditoria',(SELECT coalesce(jsonb_agg(to_jsonb(t) ORDER BY to_jsonb(t)::text COLLATE "C"),'[]'::jsonb) FROM vec_autorizacion_atestada_v3.control_cadena_auditoria t)
 ) INTO resultado;
 RETURN resultado;
END $f$;
REVOKE ALL ON FUNCTION ct86_prueba.snapshot() FROM PUBLIC;
GRANT USAGE ON SCHEMA ct86_prueba TO :"ct86_login_ejecutor";
GRANT EXECUTE ON FUNCTION ct86_prueba.snapshot() TO :"ct86_login_ejecutor";

CREATE FUNCTION ct86_prueba.fallo_final() RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER
SET search_path=pg_catalog AS $f$
BEGIN
 IF current_database() !~ '^vec_ct86_prueba_[a-f0-9]{12}$' THEN
  RAISE EXCEPTION 'CT86: auxiliar fuera del clon'; END IF;
 IF current_setting('ct86_prueba.fallo_final',true)='on'
  AND EXISTS(SELECT 1 FROM ct86_prueba.entorno WHERE base=current_database() AND login=session_user) THEN
  RAISE EXCEPTION 'CT86: fallo sintético de última escritura' USING ERRCODE='P8686'; END IF;
 RETURN NEW;
END $f$;
REVOKE ALL ON FUNCTION ct86_prueba.fallo_final() FROM PUBLIC;
CREATE TRIGGER ct86_prueba_fallo_final BEFORE INSERT ON vec_contratacion_temporal.anotacion_administrativa_incorporacion_v1
 FOR EACH ROW EXECUTE FUNCTION ct86_prueba.fallo_final();
-- Defaults del administrador no deben abrir estos auxiliares a terceros.
-- Se rechaza la preparación completa si hay grants adicionales; no se cambian
-- ACL de producto ni defaults para corregirlos dentro de esta prueba.
DO $acl_auxiliares$
DECLARE login oid;
BEGIN
 SELECT e.login::regrole::oid INTO STRICT login FROM ct86_prueba.entorno e;
 IF EXISTS(SELECT 1 FROM pg_namespace n,
   LATERAL aclexplode(coalesce(n.nspacl,acldefault('n',n.nspowner))) a
   WHERE n.nspname='ct86_prueba' AND a.grantee<>n.nspowner
    AND (a.grantee<>login OR a.privilege_type<>'USAGE' OR a.is_grantable))
  OR EXISTS(SELECT 1 FROM pg_class c,
   LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) a
   WHERE c.oid='ct86_prueba.entorno'::regclass AND a.grantee<>c.relowner)
  OR EXISTS(SELECT 1 FROM pg_proc f,
   LATERAL aclexplode(coalesce(f.proacl,acldefault('f',f.proowner))) a
   WHERE f.pronamespace='ct86_prueba'::regnamespace AND a.grantee<>f.proowner
    AND (f.proname<>'snapshot' OR a.grantee<>login OR a.privilege_type<>'EXECUTE' OR a.is_grantable)) THEN
  RAISE EXCEPTION 'CT86: ACL auxiliar ampliada por defaults'; END IF;
END $acl_auxiliares$;
COMMIT;
\else
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
SET LOCAL ct86_prueba.fallo_final='off';
DO $guard$
BEGIN
 IF current_database() !~ '^vec_ct86_prueba_[a-f0-9]{12}$'
  OR current_user<>session_user
  OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=session_user AND rolcanlogin
    AND NOT rolsuper AND NOT rolbypassrls AND NOT rolcreaterole AND NOT rolcreatedb AND NOT rolreplication)
  OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
  OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
  OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
  OR to_regclass('pg_temp.ct86_entorno') IS NULL
  OR to_regclass('pg_temp.ct86_vectores') IS NULL THEN
  RAISE EXCEPTION 'CT86: falta sesión ejecutora real y fixture privada'; END IF;
 IF (SELECT count(*) FROM pg_temp.ct86_entorno)<>1
  OR NOT EXISTS(SELECT 1 FROM pg_temp.ct86_entorno WHERE base=current_database() AND solo_sinteticos IS TRUE)
  OR (SELECT count(*) FROM pg_temp.ct86_vectores)<>2
  OR NOT EXISTS(SELECT 1 FROM pg_temp.ct86_vectores WHERE caso='inicial')
  OR NOT EXISTS(SELECT 1 FROM pg_temp.ct86_vectores WHERE caso='replay') THEN
  RAISE EXCEPTION 'CT86: contrato de fixture incompleto'; END IF;
 PERFORM ct86_prueba.snapshot();
END $guard$;

CREATE FUNCTION pg_temp.ct86_invocar(caso_elegido text,p jsonb) RETURNS jsonb LANGUAGE plpgsql AS $f$
DECLARE v record; resultado jsonb;
BEGIN
 SELECT * INTO STRICT v FROM pg_temp.ct86_vectores WHERE caso=caso_elegido;
 resultado:=vec_contratacion_temporal.registrar_anotacion_administrativa_incorporacion_v1(
  p,v.capacidad,v.decision,v.motivo,v.contexto,v.persona_version,v.perfil_version,v.payload,v.sobre,v.evidencia,v.raiz);
 RETURN resultado;
END $f$;

CREATE FUNCTION pg_temp.ct86_rechazar(etiqueta text,caso_elegido text,p jsonb,estados text[]) RETURNS void LANGUAGE plpgsql AS $f$
DECLARE antes jsonb; rechazado boolean:=false; estado text;
BEGIN
 antes:=ct86_prueba.snapshot();
 BEGIN
  PERFORM pg_temp.ct86_invocar(caso_elegido,p);
 EXCEPTION WHEN OTHERS THEN
  GET STACKED DIAGNOSTICS estado=RETURNED_SQLSTATE;
  IF NOT (estado=ANY(estados)) THEN
   RAISE EXCEPTION 'CT86 caso %: SQLSTATE inesperado %',etiqueta,estado;
  END IF;
  rechazado:=true;
 END;
 IF NOT rechazado THEN RAISE EXCEPTION 'CT86 caso %: aceptó entrada adversa',etiqueta; END IF;
 IF ct86_prueba.snapshot() IS DISTINCT FROM antes THEN
  RAISE EXCEPTION 'CT86 caso %: rechazo con efectos persistentes',etiqueta;
 END IF;
END $f$;

-- Comprueba delta exacto: historia append-only +1 y punteros actualizados;
-- las demás tablas observadas (Personal, Seguimiento y CT75) idénticas.
CREATE FUNCTION pg_temp.ct86_delta(antes jsonb,despues jsonb,replay boolean,expediente text) RETURNS void LANGUAGE plpgsql AS $f$
DECLARE k text; delta integer; anexo boolean;
 ct text[]:=ARRAY['vec_contratacion_temporal.anotacion_administrativa_incorporacion_v1',
  'vec_contratacion_temporal.expediente_version_integral','vec_contratacion_temporal.actuacion_expediente_integral',
  'vec_contratacion_temporal.outbox_expediente_integral'];
 ad text[]:=ARRAY['vec_autorizacion_atestada_v3.atestacion_decision_v3',
  'vec_autorizacion_atestada_v3.consumo_decision_v3','vec_autorizacion_atestada_v3.auditoria_consumo_v3'];
BEGIN
 IF (SELECT array_agg(key ORDER BY key) FROM jsonb_each(antes)) IS DISTINCT FROM
    (SELECT array_agg(key ORDER BY key) FROM jsonb_each(despues)) THEN
  RAISE EXCEPTION 'CT86: inventario de tablas alterado'; END IF;
 FOREACH k IN ARRAY ct||ad||ARRAY['vec_contratacion_temporal.expediente_integral_actual',
  'vec_contratacion_temporal.control_cadenas_expediente_integral','vec_autorizacion_atestada_v3.control_cadena_auditoria'] LOOP
  IF NOT (antes ? k) THEN RAISE EXCEPTION 'CT86: falta tabla obligatoria %',k; END IF;
 END LOOP;
 FOR k IN SELECT key FROM jsonb_each(antes) LOOP
  anexo:=(k=ANY(ad) OR (NOT replay AND k=ANY(ct)));
  delta:=jsonb_array_length(despues->k)-jsonb_array_length(antes->k);
  IF anexo THEN
   IF delta<>1 OR NOT ((despues->k) @> (antes->k)) THEN
    RAISE EXCEPTION 'CT86: delta/historia inesperados en %',k; END IF;
  ELSIF k='vec_autorizacion_atestada_v3.control_cadena_auditoria'
     OR (NOT replay AND k IN ('vec_contratacion_temporal.expediente_integral_actual','vec_contratacion_temporal.control_cadenas_expediente_integral')) THEN
   IF delta<>0 OR antes->k IS NOT DISTINCT FROM despues->k THEN
    RAISE EXCEPTION 'CT86: puntero no actualizado en %',k; END IF;
   IF k='vec_contratacion_temporal.expediente_integral_actual' THEN
    IF (SELECT coalesce(jsonb_agg(x ORDER BY x::text COLLATE "C"),'[]') FROM jsonb_array_elements(antes->k) x WHERE x->>'expediente_ref' IS DISTINCT FROM expediente)
     IS DISTINCT FROM
       (SELECT coalesce(jsonb_agg(x ORDER BY x::text COLLATE "C"),'[]') FROM jsonb_array_elements(despues->k) x WHERE x->>'expediente_ref' IS DISTINCT FROM expediente) THEN
     RAISE EXCEPTION 'CT86: puntero de otro expediente alterado'; END IF;
   ELSIF k='vec_contratacion_temporal.control_cadenas_expediente_integral' THEN
    IF jsonb_array_length(antes->k)<>1 OR (despues->k->0->>'secuencia_outbox')::numeric
     IS DISTINCT FROM (antes->k->0->>'secuencia_outbox')::numeric+1 THEN
     RAISE EXCEPTION 'CT86: secuencia outbox divergente'; END IF;
   ELSE
    IF jsonb_array_length(antes->k)<>1 OR (despues->k->0->>'secuencia')::numeric
     IS DISTINCT FROM (antes->k->0->>'secuencia')::numeric+1 THEN
     RAISE EXCEPTION 'CT86: secuencia auditoría V3 divergente'; END IF;
   END IF;
  ELSIF antes->k IS DISTINCT FROM despues->k THEN
   RAISE EXCEPTION 'CT86: tabla ajena modificada %',k;
  END IF;
 END LOOP;
END $f$;

DO $casos$
DECLARE v record; w record; p jsonb; q jsonb; antes jsonb; despues jsonb; recibo jsonb; recuperado jsonb;
 original jsonb; estado_original jsonb; raiz_original jsonb; n numeric; k text; texto text;
BEGIN
 SELECT * INTO STRICT v FROM pg_temp.ct86_vectores WHERE caso='inicial';
 SELECT * INTO STRICT w FROM pg_temp.ct86_vectores WHERE caso='replay';
 p:=v.peticion; n:=(p#>>'{material,version_esperada}')::numeric;
 IF char_length(p#>>'{material,observaciones}') IS DISTINCT FROM 2000
  OR octet_length(p#>>'{material,observaciones}')<=4096 THEN
  RAISE EXCEPTION 'CT86: inicial requiere vector firmado Unicode de 2000 caracteres y más de 4096 bytes'; END IF;
 antes:=ct86_prueba.snapshot();
 IF NOT EXISTS(SELECT 1 FROM jsonb_array_elements(antes->'vec_contratacion_temporal.expediente_integral_actual') x
   WHERE x->'expediente_ref'=p#>'{material,expediente_ref}' AND x->'version'=to_jsonb(n))
  OR EXISTS(SELECT 1 FROM jsonb_array_elements(antes->'vec_contratacion_temporal.expediente_version_integral') x
   WHERE x->'expediente_ref'=p#>'{material,expediente_ref}' AND (x->>'version')::numeric>n) THEN
  RAISE EXCEPTION 'CT86: fixture requiere la versión actual y sin versiones futuras'; END IF;
 IF v.decision IS NOT DISTINCT FROM w.decision OR v.capacidad IS NOT DISTINCT FROM w.capacidad
  OR (convert_from(v.decision,'UTF8')::jsonb->>'decision_ref') IS NOT DISTINCT FROM
     (convert_from(w.decision,'UTF8')::jsonb->>'decision_ref')
  OR (convert_from(v.capacidad,'UTF8')::jsonb->>'nonce') IS NOT DISTINCT FROM
     (convert_from(w.capacidad,'UTF8')::jsonb->>'nonce') THEN
  RAISE EXCEPTION 'CT86: replay necesita decisión y nonce nuevos'; END IF;
 FOREACH k IN ARRAY ARRAY['material','expediente_anterior','seguimiento_original','estado_seguimiento_sha256',
  'recibo_incorporacion_ref','ambito_idempotencia_hmac','huella_peticion_hmac'] LOOP
  IF p->k IS DISTINCT FROM w.peticion->k THEN RAISE EXCEPTION 'CT86: fixture replay cambia %',k; END IF;
 END LOOP;
 IF EXISTS(SELECT 1 FROM jsonb_array_elements(antes->'vec_contratacion_temporal.anotacion_administrativa_incorporacion_v1') x
  WHERE x->>'expediente_ref'=p#>>'{material,expediente_ref}') THEN
  RAISE EXCEPTION 'CT86: fixture ya anotada'; END IF;
 SELECT x INTO STRICT original FROM jsonb_array_elements(antes->'vec_contratacion_temporal.incorporacion_registro_v2') x
  WHERE x->>'recibo_ref'=p->>'recibo_incorporacion_ref';
 SELECT x INTO STRICT estado_original FROM jsonb_array_elements(antes->'vec_contratacion_temporal.seguimiento_estado_v2') x
  WHERE x->'seguimiento_ref'=original->'seguimiento_ref' AND x->'version_seguimiento'=original->'version_resultante';
 SELECT x INTO STRICT raiz_original FROM jsonb_array_elements(antes->'vec_contratacion_temporal.seguimiento_raiz_v2') x
  WHERE x->'seguimiento_ref'=original->'seguimiento_ref';
 IF original->'version_anterior' IS DISTINCT FROM '0'::jsonb OR original->'version_resultante' IS DISTINCT FROM '1'::jsonb
  OR original->'organizacion_ref' IS DISTINCT FROM p#>'{material,organizacion_ref}'
  OR original->'expediente_ref' IS DISTINCT FROM p#>'{material,expediente_ref}'
  OR original->'solicitud_ref' IS DISTINCT FROM p#>'{material,solicitud_personal_ref}'
  OR original->'estado_resultante_sha256' IS DISTINCT FROM p->'estado_seguimiento_sha256'
  OR original->'estado_resultante_sha256' IS DISTINCT FROM estado_original->'estado_sha256'
  OR p->'seguimiento_original' IS DISTINCT FROM jsonb_build_object('seguimiento_ref',original->'seguimiento_ref',
      'version_seguimiento',original->'version_resultante','huella_raiz_seguimiento_sha256',raiz_original->'raiz_sha256') THEN
  RAISE EXCEPTION 'CT86: fixture no enlaza resultado CT75 exacto'; END IF;
 IF EXISTS(SELECT 1 FROM jsonb_array_elements(antes->'vec_autorizacion_atestada_v3.consumo_decision_v3') x WHERE x->>'decision_ref' IN
  (convert_from(v.decision,'UTF8')::jsonb->>'decision_ref',convert_from(w.decision,'UTF8')::jsonb->>'decision_ref')) THEN
  RAISE EXCEPTION 'CT86: concesión de fixture previamente consumida'; END IF;

 -- n+1 no existe: no_data_found se clasifica P0862; no es conflicto CAS P0865.
 PERFORM pg_temp.ct86_rechazar('version distinta','inicial',jsonb_set(p,'{material,version_esperada}',to_jsonb(n+1)),ARRAY['P0862']);
 PERFORM pg_temp.ct86_rechazar('version null','inicial',jsonb_set(p,'{material,version_esperada}','null'),ARRAY['22023','P0860']);
 PERFORM pg_temp.ct86_rechazar('observaciones null','inicial',jsonb_set(p,'{material,observaciones}','null'),ARRAY['P0860']);
 FOREACH texto IN ARRAY ARRAY['',E' texto con borde ',repeat('x',2001),E'control\001',U&'e\0301'] LOOP
  PERFORM pg_temp.ct86_rechazar('texto inválido','inicial',jsonb_set(p,'{material,observaciones}',to_jsonb(texto)),ARRAY['P0860']);
 END LOOP;
 -- Texto válido distinto mantiene el sobre original: debe fallar el vínculo V3.
 texto:=CASE WHEN p#>>'{material,observaciones}'='Texto alternativo sintético A' THEN 'Texto alternativo sintético B' ELSE 'Texto alternativo sintético A' END;
 PERFORM pg_temp.ct86_rechazar('texto no autorizado','inicial',jsonb_set(p,'{material,observaciones}',to_jsonb(texto)),ARRAY['P0863']);
 PERFORM pg_temp.ct86_rechazar('actor no autorizado','inicial',jsonb_set(p,'{material,actor_ref}','"actor:ct86-ajeno"'),ARRAY['P0863']);
 PERFORM pg_temp.ct86_rechazar('perfil no autorizado','inicial',jsonb_set(p,'{material,perfil_ref}','"perfil:ct86-ajeno"'),ARRAY['P0863']);
 PERFORM pg_temp.ct86_rechazar('política caducada','inicial',jsonb_set(p,'{politica,valida_hasta}',p#>'{politica,evaluada_en}'),ARRAY['P0863']);
 PERFORM pg_temp.ct86_rechazar('recibo CT75 cruzado','inicial',jsonb_set(p,'{recibo_incorporacion_ref}','"recibo:ct86-ajeno"'),ARRAY['P0862']);
 PERFORM pg_temp.ct86_rechazar('versión seguimiento cruzada','inicial',jsonb_set(p,'{seguimiento_original,version_seguimiento}','2'),ARRAY['P0862']);
 PERFORM pg_temp.ct86_rechazar('huella seguimiento cruzada','inicial',jsonb_set(p,'{estado_seguimiento_sha256}',to_jsonb(repeat('f',64))),ARRAY['P0862']);
 PERFORM pg_temp.ct86_rechazar('solicitud Personal cruzada','inicial',jsonb_set(p,'{material,solicitud_personal_ref}','"solicitud:ct86-ajena"'),ARRAY['P0862']);
 -- La postimagen se coteja después de consumir V3: su rechazo debe revertir
 -- también atestación, consumo, auditoría y cabeza V3, no sólo las filas CT.
 PERFORM pg_temp.ct86_rechazar('postimagen null','inicial',jsonb_set(p,'{expediente_siguiente}','null'),ARRAY['P0862']);
 PERFORM pg_temp.ct86_rechazar('postimagen fase alterada','inicial',jsonb_set(p,'{expediente_siguiente,fase_actual}','"ct86_fase_ajena"'),ARRAY['P0862']);

 -- Fallo justo antes del INSERT del recibo, después de versión/actuación/outbox.
 PERFORM set_config('ct86_prueba.fallo_final','on',true);
 PERFORM pg_temp.ct86_rechazar('última escritura','inicial',p,ARRAY['P8686']);
 PERFORM set_config('ct86_prueba.fallo_final','off',true);

 antes:=ct86_prueba.snapshot();
 recibo:=pg_temp.ct86_invocar('inicial',p);
 despues:=ct86_prueba.snapshot();
 PERFORM pg_temp.ct86_delta(antes,despues,false,p#>>'{material,expediente_ref}');
 IF recibo->>'operacion' IS DISTINCT FROM 'registrar_anotacion_administrativa'
  OR recibo->'version_anterior' IS DISTINCT FROM to_jsonb(n)
  OR recibo->'version_resultante' IS DISTINCT FROM to_jsonb(n+1)
  OR recibo->'seguimiento_original' IS DISTINCT FROM p->'seguimiento_original'
  OR recibo->'recibo_ref' IS DISTINCT FROM p#>'{referencias,recibo_ref}'
  OR recibo->'evento_ref' IS DISTINCT FROM p#>'{referencias,evento_ref}'
  OR recibo->'registrada_en' IS DISTINCT FROM p->'instante_efecto'
  OR NOT EXISTS(SELECT 1 FROM jsonb_array_elements(despues->'vec_contratacion_temporal.expediente_integral_actual') a
    JOIN jsonb_array_elements(despues->'vec_contratacion_temporal.expediente_version_integral') z
      ON a->'expediente_ref'=z->'expediente_ref' AND a->'version'=z->'version'
    WHERE a->'expediente_ref'=p#>'{material,expediente_ref}' AND a->'version'=to_jsonb(n+1)
    AND z->'agregado_json'=p->'expediente_siguiente'
    AND z->'fase_clave'=p#>'{expediente_anterior,fase_actual}'
    AND z->'estado'=p#>'{expediente_anterior,estado_actual}') THEN
  RAISE EXCEPTION 'CT86: recibo/postimagen nominal divergente'; END IF;
 PERFORM pg_temp.ct86_rechazar('concesión histórica','inicial',p,ARRAY['P1102']);
 PERFORM pg_temp.ct86_rechazar('clave con otro texto','replay',jsonb_set(w.peticion,'{material,observaciones}','"Otra anotación sintética"'),ARRAY['P0861']);
 antes:=despues;
 recuperado:=pg_temp.ct86_invocar('replay',w.peticion);
 despues:=ct86_prueba.snapshot();
 IF recuperado IS DISTINCT FROM recibo THEN RAISE EXCEPTION 'CT86: replay cambió recibo/fecha/auditoría'; END IF;
 PERFORM pg_temp.ct86_delta(antes,despues,true,p#>>'{material,expediente_ref}');
 RAISE NOTICE 'CT86: escenarios sintéticos completados; se revierte la transacción';
END $casos$;
ROLLBACK;
\endif
