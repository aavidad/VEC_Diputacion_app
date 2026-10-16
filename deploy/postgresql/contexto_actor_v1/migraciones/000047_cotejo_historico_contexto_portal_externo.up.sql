\set ON_ERROR_STOP on
-- CA47: cotejo histórico del contexto de una lectura del portal externo.
-- Desde CA15 el portal externo registra su contexto en registros_contexto_externo_v2;
-- CA26/CA27 solo cotejan registros_contexto y fallaban con P0002 (lectura de
-- inscripción del aspirante, AD228). Misma regla de despacho que
-- acreditar_uso_registro_contexto_actor_v2 (CA15): un recibo vive en un solo
-- almacén; el externo se reconstruye desde su provisión versionada y el interno
-- sigue en CA26/CA27 sin cambios. Solo acredita el hecho histórico; no renueva
-- concesiones ni toca CA26/CA27, tablas, datos o permisos existentes.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contexto_actor_v1:migracion:000047:cotejo-portal-externo',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contexto_actor_v1:migracion:cotejo_auditoria_intentos:v1',0));
SET LOCAL ROLE vec_contexto_actor_v1_propietario;

DO $pre$
DECLARE
 interno regprocedure := to_regprocedure('vec_contexto_actor_v1.cotejar_contexto_historico_auditoria_v1(text,text,text,bytea)');
 p pg_proc%ROWTYPE;
BEGIN
 SELECT * INTO p FROM pg_proc WHERE oid=interno;
 IF current_user <> 'vec_contexto_actor_v1_propietario'
    OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR interno IS NULL
    OR to_regprocedure('vec_contexto_actor_v1.cotejar_contexto_historico_portal_externo_v1(text,text,text,bytea)') IS NOT NULL
    OR to_regprocedure('vec_contexto_actor_v1.canon_snapshot_contexto_externo_v2(jsonb,timestamptz)') IS NULL
    OR to_regprocedure('vec_contexto_actor_v1.huella_snapshot_contexto_externo_v1(jsonb,numeric)') IS NULL
    OR to_regprocedure('vec_contexto_actor_v1.snapshot_contexto_externo_valido_v1(jsonb)') IS NULL
    OR to_regprocedure('vec_contexto_actor_v1.referencia_operacion_valida(text,text)') IS NULL
    OR to_regprocedure('vec_contexto_actor_v1.instante_valido(timestamptz)') IS NULL
    OR to_regclass('vec_contexto_actor_v1.registros_contexto_externo_v2') IS NULL
    OR to_regclass('vec_contexto_actor_v1.contexto_externo_versiones') IS NULL
    OR to_regclass('vec_contexto_actor_v1.contexto_externo_identidad') IS NULL
    OR to_regclass('vec_contexto_actor_v1.registro_candidato_externo_v1') IS NULL
    OR to_regclass('vec_contexto_actor_v1.perfil_usuarios_externo_identidad') IS NULL
    OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_autorizacion_atestada_v3_propietario'
       AND NOT rolcanlogin AND NOT rolsuper AND NOT rolbypassrls)
    -- Preimagen CA27 instalada: el despacho delega en ella sin modificarla.
    OR encode(sha256(convert_to(pg_get_functiondef(interno),'UTF8')),'hex')
       IS DISTINCT FROM 'e3b0b4ce0a6bed2b0ff6718e42c0112f24122551115cd6a460e095a538672ca7'
    OR p.proowner IS DISTINCT FROM 'vec_contexto_actor_v1_propietario'::regrole
    OR p.proacl IS DISTINCT FROM ARRAY['vec_contexto_actor_v1_propietario=X/vec_contexto_actor_v1_propietario',
       'vec_autorizacion_atestada_v3_propietario=X/vec_contexto_actor_v1_propietario']::aclitem[]
 THEN RAISE EXCEPTION 'CA47: preimagen o dependencias incompatibles; no reaplicar' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE FUNCTION vec_contexto_actor_v1.cotejar_contexto_historico_portal_externo_v1(
 p_registro_ref text, p_contexto_sha256 text, p_procedencia_sha256 text,
 p_contexto_canonico bytea
) RETURNS boolean
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog, pg_temp
SET statement_timeout='5s'
SET lock_timeout='2s'
AS $historia$
DECLARE
 externo boolean; interno boolean;
 registro record; provision record; identidad record; canon record;
 componente jsonb; clave text;
 inicio timestamptz := pg_catalog.clock_timestamp();
BEGIN
 IF current_setting('transaction_isolation') <> 'serializable'
    OR current_setting('transaction_read_only') <> 'off'
    OR current_setting('TimeZone') <> 'UTC' THEN
   RAISE EXCEPTION 'cotejo historico requiere SERIALIZABLE READ WRITE UTC' USING ERRCODE='25000';
 END IF;
 IF p_contexto_canonico IS NULL OR octet_length(p_contexto_canonico) NOT BETWEEN 1 AND 65536
    OR encode(sha256(p_contexto_canonico),'hex') IS DISTINCT FROM p_contexto_sha256 THEN
   RAISE EXCEPTION 'contexto historico invalido' USING ERRCODE='22023';
 END IF;
 IF vec_contexto_actor_v1.referencia_operacion_valida(p_registro_ref,'rca_') IS NOT TRUE
    OR p_contexto_sha256 IS NULL OR p_contexto_sha256 !~ '^[0-9a-f]{64}$'
    OR p_procedencia_sha256 IS NULL OR p_procedencia_sha256 !~ '^[0-9a-f]{64}$' THEN
   RAISE EXCEPTION 'selector historico invalido' USING ERRCODE='22023';
 END IF;
 PERFORM pg_advisory_xact_lock_shared(hashtextextended(
   'vec_contexto_actor_v1:migracion:cotejo_auditoria_intentos:v1',0));
 SELECT EXISTS(SELECT 1 FROM vec_contexto_actor_v1.registros_contexto_externo_v2 r
               WHERE r.registro_contexto_ref=p_registro_ref),
        EXISTS(SELECT 1 FROM vec_contexto_actor_v1.registros_contexto r
               WHERE r.registro_contexto_ref=p_registro_ref)
   INTO externo, interno;
 -- Un recibo en los dos almacenes nunca es válido (misma regla que CA15).
 IF externo AND interno THEN
   RAISE EXCEPTION 'contexto historico en ambos almacenes' USING ERRCODE='42501';
 END IF;
 IF NOT externo THEN
   -- Recibo anterior a CA15: solo si la cuenta no tiene población externa
   -- segregada; entonces coteja CA26/CA27 exactamente como hasta ahora.
   IF EXISTS(SELECT 1 FROM vec_contexto_actor_v1.registro_candidato_externo_v1 r
             WHERE r.registro_contexto_ref=p_registro_ref)
      OR EXISTS(SELECT 1 FROM vec_contexto_actor_v1.registros_contexto r
                JOIN vec_contexto_actor_v1.perfil_usuarios_externo_identidad i
                  ON i.cuenta_ref=r.cuenta_ref OR i.perfil_ref=r.perfil_ref
                WHERE r.registro_contexto_ref=p_registro_ref)
      OR EXISTS(SELECT 1 FROM vec_contexto_actor_v1.registros_contexto r
                JOIN vec_contexto_actor_v1.contexto_externo_identidad i
                  ON i.cuenta_ref=r.cuenta_ref OR i.perfil_ref=r.perfil_ref
                WHERE r.registro_contexto_ref=p_registro_ref) THEN
     RAISE EXCEPTION 'contexto historico de poblacion externa en almacen interno' USING ERRCODE='42501';
   END IF;
   RETURN vec_contexto_actor_v1.cotejar_contexto_historico_auditoria_v1(
     p_registro_ref,p_contexto_sha256,p_procedencia_sha256,p_contexto_canonico);
 END IF;
 -- Sin row locks ni punteros actuales: recibo, provisión e identidad son
 -- historia de solo adición.
 SELECT r.* INTO STRICT registro FROM vec_contexto_actor_v1.registros_contexto_externo_v2 r
  WHERE r.registro_contexto_ref=p_registro_ref
    AND r.huella_sha256=p_contexto_sha256
    AND r.manifiesto_procedencia_huella_sha256=p_procedencia_sha256;
 IF octet_length(registro.representacion_canonica) NOT BETWEEN 1 AND 65536
    OR octet_length(registro.manifiesto_procedencia_canonico) NOT BETWEEN 1 AND 65536
    OR encode(sha256(registro.representacion_canonica),'hex') IS DISTINCT FROM p_contexto_sha256
    OR encode(sha256(registro.manifiesto_procedencia_canonico),'hex') IS DISTINCT FROM p_procedencia_sha256
    OR registro.autoridad_efectiva IS DISTINCT FROM 'autoridad_maestra_acreditada'
    OR registro.metodo IS DISTINCT FROM 'certificado'
    OR registro.garantia IS DISTINCT FROM 'alto'
    OR vec_contexto_actor_v1.instante_valido(registro.resuelto_en) IS NOT TRUE
    OR vec_contexto_actor_v1.instante_valido(registro.solicitado_en) IS NOT TRUE
    OR registro.resuelto_en < registro.solicitado_en
    OR registro.resuelto_en > registro.solicitado_en + interval '5 seconds' THEN
   RAISE EXCEPTION 'original historico incoherente' USING ERRCODE='22023';
 END IF;
 SELECT v.* INTO STRICT provision FROM vec_contexto_actor_v1.contexto_externo_versiones v
  WHERE v.provision_ref=registro.provision_ref AND v.version=registro.provision_version;
 SELECT i.* INTO STRICT identidad FROM vec_contexto_actor_v1.contexto_externo_identidad i
  WHERE i.provision_ref=registro.provision_ref;
 IF provision.huella_sha256 IS DISTINCT FROM registro.provision_huella_sha256
    OR vec_contexto_actor_v1.huella_snapshot_contexto_externo_v1(provision.snapshot,provision.version)
       IS DISTINCT FROM registro.provision_huella_sha256
    OR vec_contexto_actor_v1.snapshot_contexto_externo_valido_v1(provision.snapshot) IS NOT TRUE
    OR provision.cuenta_ref IS DISTINCT FROM registro.cuenta_ref
    OR provision.perfil_ref IS DISTINCT FROM registro.perfil_ref
    OR identidad.cuenta_ref IS DISTINCT FROM registro.cuenta_ref
    OR identidad.perfil_ref IS DISTINCT FROM registro.perfil_ref
    OR identidad.persona_ref IS DISTINCT FROM provision.persona_ref
    OR identidad.familia IS DISTINCT FROM provision.familia
    OR identidad.contexto_ref IS DISTINCT FROM provision.contexto_ref
    OR provision.snapshot->>'estado' IS DISTINCT FROM 'activo' THEN
   RAISE EXCEPTION 'provision historica incoherente' USING ERRCODE='22023';
 END IF;
 -- Vigencia en resuelto_en ORIGINAL, nunca reloj actual: no renueva permiso.
 FOREACH clave IN ARRAY ARRAY['cuenta','persona','perfil','contexto','vinculo_candidato'] LOOP
   componente := provision.snapshot->clave;
   IF componente = 'null'::jsonb THEN CONTINUE; END IF;
   IF registro.resuelto_en < (componente->>'vigente_desde')::timestamptz
      OR registro.resuelto_en >= (componente->>'vigente_hasta')::timestamptz THEN
     RAISE EXCEPTION 'fuente historica incoherente' USING ERRCODE='22023';
   END IF;
 END LOOP;
 -- Mismo serializador que el resolutor CA15: comparar bytes cierra claves,
 -- tipos, orden y escapes.
 SELECT * INTO STRICT canon
   FROM vec_contexto_actor_v1.canon_snapshot_contexto_externo_v2(provision.snapshot,registro.resuelto_en);
 IF p_contexto_canonico IS DISTINCT FROM registro.representacion_canonica
    OR canon.representacion IS DISTINCT FROM registro.representacion_canonica
    OR canon.manifiesto IS DISTINCT FROM registro.manifiesto_procedencia_canonico
    OR encode(sha256(canon.representacion),'hex') IS DISTINCT FROM p_contexto_sha256
    OR encode(sha256(canon.manifiesto),'hex') IS DISTINCT FROM p_procedencia_sha256 THEN
   RAISE EXCEPTION 'canon historico divergente' USING ERRCODE='22023';
 END IF;
 IF clock_timestamp() > inicio + interval '5 seconds' THEN
   RAISE EXCEPTION 'lectura historica fuera de ventana' USING ERRCODE='57014';
 END IF;
 RETURN true;
EXCEPTION
 WHEN no_data_found OR too_many_rows THEN
   RAISE EXCEPTION 'contexto historico no disponible' USING ERRCODE='P0002';
 WHEN data_exception THEN
   RAISE EXCEPTION 'contexto historico invalido' USING ERRCODE='22023';
END
$historia$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.cotejar_contexto_historico_portal_externo_v1(text,text,text,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.cotejar_contexto_historico_portal_externo_v1(text,text,text,bytea)
 TO vec_autorizacion_atestada_v3_propietario;

DO $post$
DECLARE
 f oid := 'vec_contexto_actor_v1.cotejar_contexto_historico_portal_externo_v1(text,text,text,bytea)'::regprocedure;
 i oid := 'vec_contexto_actor_v1.cotejar_contexto_historico_auditoria_v1(text,text,text,bytea)'::regprocedure;
 o oid := 'vec_contexto_actor_v1_propietario'::regrole;
 r oid := 'vec_autorizacion_atestada_v3_propietario'::regrole;
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.proowner=o AND p.prosecdef
       AND p.provolatile='v' AND p.proparallel='u' AND p.prorettype='boolean'::regtype
       AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','statement_timeout=5s','lock_timeout=2s']::text[])
    OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) a WHERE p.oid=f)<>2
    OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) a WHERE p.oid=f
       AND (a.grantee NOT IN(o,r) OR a.grantor<>o OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
    OR encode(sha256(convert_to(pg_get_functiondef(i),'UTF8')),'hex')
       IS DISTINCT FROM 'e3b0b4ce0a6bed2b0ff6718e42c0112f24122551115cd6a460e095a538672ca7'
 THEN RAISE EXCEPTION 'CA47: posimagen o ACL distinta' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
