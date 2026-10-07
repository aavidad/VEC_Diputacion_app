\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL TIME ZONE 'UTC';
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_autorizacion_atestada_v3:migracion:000171',0));

-- AD171 depende de AD169. Añade familias sin inventar decisión, perfil ni sesión.
-- Los owners de IS/AUT construyen el evento desde su fuente privada acreditada.
DO $pre$
BEGIN
 IF pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_intento_nominal_v1(bytea,bytea,jsonb)') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.encuadrar_mac(text)') IS NULL
 OR pg_catalog.to_regrole('vec_identidad_sesiones_v1_propietario') IS NULL
 OR pg_catalog.to_regrole('vec_autorizacion_propietario') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_evento_admin_preperfil_v1(jsonb)') IS NOT NULL
 OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_constraint c
   WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
     AND c.conname='auditoria_tipo_disjunto_v1' AND c.contype='c' AND c.convalidated)
 THEN RAISE EXCEPTION 'AD171: PARO clave=preimagen actual=incompatible esperado=AD169_sin_AD171' USING ERRCODE='55000'; END IF;
END $pre$;

LOCK TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 IN ACCESS EXCLUSIVE MODE;
ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3
 ADD COLUMN evento_ref text UNIQUE,
 ADD COLUMN evento_material_sha256 text,
 ADD COLUMN fuente_ref text,
 ADD COLUMN fuente_sha256 text,
 ADD COLUMN operador_login name,
 ADD COLUMN plan_sha256 text,
 ADD COLUMN aprobacion_ref text;
-- Conserva literalmente la condición AD169 presente; no reconstruye ni debilita
-- la condición de las dos familias anteriores y no toca ninguna fila histórica.
DO $familias$
DECLARE v_preimagen text;v_nueva text;
BEGIN
 SELECT pg_catalog.pg_get_constraintdef(c.oid,false) INTO STRICT v_preimagen
 FROM pg_catalog.pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
   AND c.conname='auditoria_tipo_disjunto_v1' AND c.contype='c' AND c.convalidated;
 IF pg_catalog.left(v_preimagen,7)<>'CHECK (' OR pg_catalog.right(v_preimagen,1)<>')'
 THEN RAISE EXCEPTION 'AD171: PARO clave=CHECK actual=incompatible esperado=CHECK_validado' USING ERRCODE='55000'; END IF;
 v_nueva:='CHECK (('||$nulos$evento_ref IS NULL AND evento_material_sha256 IS NULL AND fuente_ref IS NULL AND fuente_sha256 IS NULL AND operador_login IS NULL AND plan_sha256 IS NULL AND aprobacion_ref IS NULL$nulos$||' AND ('||
   pg_catalog.substr(v_preimagen,8,pg_catalog.length(v_preimagen)-8)||')) OR '||
   $extension$   (tipo_registro IN ('preperfil_autenticado','bootstrap_operador')
    AND decision_ref IS NULL AND efecto_ref IS NULL AND huella_efecto_sha256 IS NULL
    AND intento_ref IS NULL AND intento_material_sha256 IS NULL
    AND perfil_activo_ref IS NULL AND registro_contexto_ref IS NULL
    AND contexto_sha256 IS NULL AND procedencia_sha256 IS NULL
    AND autenticacion_ref IS NULL AND sesion_ref IS NULL
    AND autenticacion_sha256 IS NULL AND vinculo_sha256 IS NULL
    AND evento_ref IS NOT NULL AND evento_material_sha256 IS NOT NULL
    AND fuente_ref IS NOT NULL AND fuente_sha256 IS NOT NULL
    AND accion IS NOT NULL AND modulo_id IS NOT DISTINCT FROM 'administracion'
    AND recurso_ref IS NOT NULL AND finalidad_ref IS NOT NULL
    AND resultado IS NOT NULL AND motivo_ref IS NOT NULL
    AND proceso IS NOT NULL AND canal IS NOT NULL AND correlacion_ref IS NOT NULL
    AND (
      (tipo_registro='preperfil_autenticado' AND actor_ref IS NOT NULL
       AND operador_login IS NULL AND plan_sha256 IS NULL AND aprobacion_ref IS NULL
       AND accion IN ('listar_perfiles_propios_admin','seleccionar_perfil_admin')
       AND canal='administracion_privilegiada' AND finalidad_ref='seleccion_perfil')
      OR
      (tipo_registro='bootstrap_operador' AND actor_ref IS NULL
       AND operador_login IS NOT NULL AND plan_sha256 IS NOT NULL AND aprobacion_ref IS NOT NULL
       AND accion='ejecutar_plan_bootstrap_admin'
       AND canal='operacion_tecnica_privada' AND finalidad_ref='bootstrap_admin')
    ))$extension$||')';
 ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 DROP CONSTRAINT auditoria_tipo_disjunto_v1;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 ADD CONSTRAINT auditoria_tipo_disjunto_v2 '||v_nueva;
END $familias$;
ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3
 ADD CONSTRAINT auditoria_evento_admin_formato_v1 CHECK (
   tipo_registro NOT IN ('preperfil_autenticado','bootstrap_operador') OR (
    evento_ref ~ '^evento_[0-9a-f]{32}$'
    AND evento_material_sha256 ~ '^[0-9a-f]{64}$'
    AND fuente_sha256 ~ '^[0-9a-f]{64}$'
    AND fuente_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
    AND recurso_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,199}$'
    AND proceso ~ '^[a-z][a-z0-9._-]{1,79}$'
    AND correlacion_ref ~ '^correlacion_[0-9a-f]{32}$'
    AND resultado IN ('permitido','denegado','error')
    AND motivo_ref ~ '^[a-z][a-z0-9._:-]{0,159}$'
    AND ((tipo_registro='preperfil_autenticado'
          AND actor_ref ~ '^per_[A-Za-z0-9_-]{22,128}$')
      OR (tipo_registro='bootstrap_operador'
          AND plan_sha256 ~ '^[0-9a-f]{64}$'
          AND aprobacion_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'))
   )
 );

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_evento_admin_preperfil_v1(p_evento jsonb)
RETURNS TABLE(auditoria_ref text,secuencia numeric,huella_sha256 text,
              correlacion_ref text,registrada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog SET row_security=on
SET lock_timeout='2s' SET statement_timeout='10s'
AS $funcion$
DECLARE
 v_orden text[];
 v_claves text[];
 v_clave text;
 v_tipo text;
 v_material bytea;
 v_material_sha text;
 v_anterior text;
 v_secuencia numeric;
 v_instante timestamptz(6);
 v_ref text;
 v_huella text;
 v_existente record;
BEGIN
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable'
 OR pg_catalog.current_setting('transaction_read_only')<>'off'
 OR pg_catalog.current_setting('TimeZone')<>'UTC'
 THEN RAISE EXCEPTION 'AD171: PARO clave=transaccion actual=incompatible esperado=serializable_rw_UTC' USING ERRCODE='25000'; END IF;
 IF pg_catalog.jsonb_typeof(p_evento) IS DISTINCT FROM 'object'
 OR pg_catalog.octet_length(p_evento::text)>8192
 THEN RAISE EXCEPTION 'AD171: PARO clave=evento actual=incompatible esperado=objeto_acotado' USING ERRCODE='22023'; END IF;
 v_tipo:=p_evento->>'tipo_registro';
 IF v_tipo='preperfil_autenticado' THEN
  v_orden:=ARRAY['tipo_registro','evento_ref','actor_ref','accion','recurso_ref','resultado',
    'motivo_ref','proceso','canal','finalidad_ref','correlacion_ref','fuente_ref','fuente_sha256'];
 ELSIF v_tipo='bootstrap_operador' THEN
  v_orden:=ARRAY['tipo_registro','evento_ref','operador_login','plan_sha256','aprobacion_ref',
    'accion','recurso_ref','resultado','motivo_ref','proceso','canal','finalidad_ref',
    'correlacion_ref','fuente_ref','fuente_sha256'];
 ELSE RAISE EXCEPTION 'AD171: PARO clave=tipo actual=incompatible esperado=familia_cerrada' USING ERRCODE='22023'; END IF;
 SELECT pg_catalog.array_agg(k ORDER BY k COLLATE "C") INTO v_claves
   FROM pg_catalog.jsonb_object_keys(p_evento) k;
 IF v_claves IS DISTINCT FROM (SELECT pg_catalog.array_agg(k ORDER BY k COLLATE "C") FROM pg_catalog.unnest(v_orden) k)
 THEN RAISE EXCEPTION 'AD171: PARO clave=campos actual=incompatible esperado=ABI_exacto' USING ERRCODE='22023'; END IF;
 FOREACH v_clave IN ARRAY v_orden LOOP
  IF pg_catalog.jsonb_typeof(p_evento->v_clave) IS DISTINCT FROM 'string'
  OR pg_catalog.octet_length(p_evento->>v_clave) NOT BETWEEN 1 AND 200
  THEN RAISE EXCEPTION 'AD171: PARO clave=valor actual=incompatible esperado=string_acotado' USING ERRCODE='22023'; END IF;
 END LOOP;
 IF p_evento->>'evento_ref' !~ '^evento_[0-9a-f]{32}$'
 OR p_evento->>'fuente_sha256' !~ '^[0-9a-f]{64}$'
 OR p_evento->>'fuente_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
 OR p_evento->>'recurso_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,199}$'
 OR p_evento->>'proceso' !~ '^[a-z][a-z0-9._-]{1,79}$'
 OR p_evento->>'correlacion_ref' !~ '^correlacion_[0-9a-f]{32}$'
 OR p_evento->>'resultado' NOT IN ('permitido','denegado','error')
 OR p_evento->>'motivo_ref' !~ '^[a-z][a-z0-9._:-]{0,159}$'
 OR (v_tipo='preperfil_autenticado' AND (
       p_evento->>'actor_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
       OR p_evento->>'accion' NOT IN ('listar_perfiles_propios_admin','seleccionar_perfil_admin')
       OR p_evento->>'canal' IS DISTINCT FROM 'administracion_privilegiada'
       OR p_evento->>'finalidad_ref' IS DISTINCT FROM 'seleccion_perfil'))
 OR (v_tipo='bootstrap_operador' AND (
       p_evento->>'operador_login' IS DISTINCT FROM session_user::text
       OR pg_catalog.octet_length(p_evento->>'operador_login')>63
       OR p_evento->>'plan_sha256' !~ '^[0-9a-f]{64}$'
       OR p_evento->>'aprobacion_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
       OR p_evento->>'accion' IS DISTINCT FROM 'ejecutar_plan_bootstrap_admin'
       OR p_evento->>'canal' IS DISTINCT FROM 'operacion_tecnica_privada'
       OR p_evento->>'finalidad_ref' IS DISTINCT FROM 'bootstrap_admin'))
 THEN RAISE EXCEPTION 'AD171: PARO clave=semantica actual=incompatible esperado=familia_acreditada' USING ERRCODE='22023'; END IF;
 -- Preimagen cerrada por familia; JSON no participa como serialización canónica.
 v_material:=vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.admin-preperfil.v1');
 FOREACH v_clave IN ARRAY v_orden LOOP
  v_material:=v_material||vec_autorizacion_atestada_v3.encuadrar_mac(p_evento->>v_clave);
 END LOOP;
 v_material_sha:=pg_catalog.encode(pg_catalog.sha256(v_material),'hex');
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
   'vec_autorizacion_atestada_v3:evento-admin:'||(p_evento->>'evento_ref'),0));
 SELECT a.auditoria_ref,a.secuencia,a.huella_sha256,a.correlacion_ref,a.registrada_en,
        a.evento_material_sha256 INTO v_existente
 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a WHERE a.evento_ref=p_evento->>'evento_ref';
 IF FOUND THEN
  IF v_existente.evento_material_sha256 IS DISTINCT FROM v_material_sha
  THEN RAISE EXCEPTION 'AD171: PARO clave=replay actual=material_distinto esperado=material_original' USING ERRCODE='23505'; END IF;
  RETURN QUERY SELECT v_existente.auditoria_ref,v_existente.secuencia,v_existente.huella_sha256,
    v_existente.correlacion_ref,v_existente.registrada_en;
  RETURN;
 END IF;
 SELECT c.secuencia,c.cabeza_sha256 INTO STRICT v_secuencia,v_anterior
 FROM vec_autorizacion_atestada_v3.control_cadena_auditoria c WHERE c.control_id FOR UPDATE;
 IF v_secuencia>=9007199254740991::numeric
 THEN RAISE EXCEPTION 'AD171: PARO clave=secuencia actual=limite esperado=entero_seguro' USING ERRCODE='22003'; END IF;
 v_secuencia:=v_secuencia+1;
 v_instante:=pg_catalog.clock_timestamp();
 v_ref:='aud_v3_p_'||pg_catalog.substr(p_evento->>'evento_ref',8,32);
 v_huella:=pg_catalog.encode(pg_catalog.sha256(
   vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.eslabon.admin-preperfil.v1')||
   vec_autorizacion_atestada_v3.encuadrar_mac(v_secuencia::text)||
   vec_autorizacion_atestada_v3.encuadrar_mac(v_anterior)||
   vec_autorizacion_atestada_v3.encuadrar_mac(v_ref)||
   vec_autorizacion_atestada_v3.encuadrar_mac(v_material_sha)||
   vec_autorizacion_atestada_v3.encuadrar_mac(pg_catalog.to_char(v_instante AT TIME ZONE 'UTC',
      'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))),'hex');
 INSERT INTO vec_autorizacion_atestada_v3.auditoria_consumo_v3(
   auditoria_ref,secuencia,anterior_sha256,huella_sha256,registrada_en,tipo_registro,
   evento_ref,evento_material_sha256,fuente_ref,fuente_sha256,actor_ref,
   operador_login,plan_sha256,aprobacion_ref,accion,modulo_id,recurso_ref,
   finalidad_ref,resultado,motivo_ref,proceso,canal,correlacion_ref)
 VALUES(v_ref,v_secuencia,v_anterior,v_huella,v_instante,v_tipo,
   p_evento->>'evento_ref',v_material_sha,p_evento->>'fuente_ref',p_evento->>'fuente_sha256',
   p_evento->>'actor_ref',(p_evento->>'operador_login')::name,
   p_evento->>'plan_sha256',p_evento->>'aprobacion_ref',p_evento->>'accion','administracion',
   p_evento->>'recurso_ref',p_evento->>'finalidad_ref',p_evento->>'resultado',
   p_evento->>'motivo_ref',p_evento->>'proceso',p_evento->>'canal',p_evento->>'correlacion_ref');
 UPDATE vec_autorizacion_atestada_v3.control_cadena_auditoria
 SET secuencia=v_secuencia,cabeza_sha256=v_huella,actualizada_en=v_instante WHERE control_id;
 RETURN QUERY SELECT v_ref,v_secuencia,v_huella,p_evento->>'correlacion_ref',v_instante;
END $funcion$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_evento_admin_preperfil_v1(jsonb)
 FROM PUBLIC,vec_autorizacion_atestada_v3_consumidor,vec_autorizacion_atestada_v3_emisor,
      vec_autorizacion_atestada_v3_registrador_intentos;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3
 TO vec_identidad_sesiones_v1_propietario,vec_autorizacion_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_evento_admin_preperfil_v1(jsonb)
 TO vec_identidad_sesiones_v1_propietario,vec_autorizacion_propietario;
DO $acl$
DECLARE f oid:=pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_evento_admin_preperfil_v1(jsonb)');
BEGIN
 IF EXISTS(SELECT 1 FROM pg_catalog.pg_proc p,
   LATERAL pg_catalog.aclexplode(COALESCE(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
   WHERE p.oid=f AND (a.grantee NOT IN (
    'vec_autorizacion_atestada_v3_propietario'::regrole,
    'vec_identidad_sesiones_v1_propietario'::regrole,'vec_autorizacion_propietario'::regrole)
    OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
 THEN RAISE EXCEPTION 'AD171: PARO clave=ACL actual=ampliada esperado=owners_privados' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
