\set ON_ERROR_STOP on
-- IS22: cotejo de la autenticación de una lectura del portal externo (F1-84).
-- El portal externo separado registra sus sesiones en vec_identidad_externa_v1
-- (IS07); IS13/IS20 solo cotejan el almacén interno y la lectura de inscripción
-- del aspirante fallaba con P0002. Función nueva con la misma regla de despacho
-- que CA47: una sesión vive en un solo almacén. Histórico puro, como IS20 y
-- CA47: coteja la sesión registrada en su revisión original, sin punteros
-- actuales ni reloj (la vigencia en vivo ya la exige la decisión). La rama
-- interna (sesiones externa_personal del proceso integrado) delega en IS20 sin
-- cambios. No toca IS20, tablas, datos ni el ACL de los esquemas.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_identidad_sesiones_v1:migracion:000022',0));
-- Mismo cerrojo que los lectores IS13/IS20.
SELECT pg_advisory_xact_lock(hashtextextended('vec_identidad_sesiones_v1:migracion:cotejo_auditoria_intentos:v1',0));
SET LOCAL ROLE vec_identidad_sesiones_v1_propietario;

DO $pre$
DECLARE
 interno oid:=to_regprocedure('vec_identidad_sesiones_v1.cotejar_autenticacion_historica_auditoria_v1(bytea)');
 huella oid:=to_regprocedure('vec_identidad_sesiones_v1.huella_control_sesion_v1(text,numeric,text,text,timestamptz,timestamptz,text)');
 o oid:=to_regrole('vec_identidad_sesiones_v1_propietario');
 a oid:=to_regrole('vec_autorizacion_atestada_v3_propietario');
 t text;
BEGIN
 IF current_user<>'vec_identidad_sesiones_v1_propietario'
    OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR interno IS NULL OR huella IS NULL OR o IS NULL OR a IS NULL
    OR to_regprocedure('vec_identidad_sesiones_v1.cotejar_autenticacion_historica_portal_externo_v1(bytea)') IS NOT NULL
    OR to_regprocedure('vec_identidad_sesiones_v1.referencia_valida(text,text)') IS NULL
    OR to_regclass('vec_autorizacion.sesion_autenticacion_v1') IS NULL
    OR to_regclass('vec_identidad_sesiones_v1.consumo_asercion') IS NULL
    -- Preimagen IS20 instalada: la rama interna delega en ella sin modificarla.
    OR encode(sha256(convert_to(pg_get_functiondef(interno),'UTF8')),'hex')
       IS DISTINCT FROM 'bff185afc93d70e7f5d1d4a8604f8d5d434aa017b624cbf5d122ce069bea6b24'
    OR encode(sha256(convert_to(pg_get_functiondef(huella),'UTF8')),'hex')
       IS DISTINCT FROM '918695217398029a8b1e008376db501e56b643ce8ad0cf99988e4928159166ca'
    OR (SELECT p.proacl FROM pg_proc p WHERE p.oid=interno) IS DISTINCT FROM
       ARRAY['vec_identidad_sesiones_v1_propietario=X/vec_identidad_sesiones_v1_propietario',
        'vec_autorizacion_atestada_v3_propietario=X/vec_identidad_sesiones_v1_propietario']::aclitem[]
    OR EXISTS(SELECT 1 FROM pg_roles WHERE oid=a AND (rolcanlogin OR rolsuper OR rolbypassrls))
 THEN RAISE EXCEPTION 'IS22: preimagen o dependencias incompatibles; no reaplicar' USING ERRCODE='55000'; END IF;
 -- Almacén externo IS07: tablas del propietario con RLS forzada.
 FOREACH t IN ARRAY ARRAY['sesion','control_sesion','cuenta','estado_cuenta'] LOOP
  IF NOT EXISTS(SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
      WHERE n.nspname='vec_identidad_externa_v1' AND c.relname=t AND c.relkind='r'
        AND c.relowner=o AND c.relrowsecurity AND c.relforcerowsecurity) THEN
   RAISE EXCEPTION 'IS22: almacén externo incompatible' USING ERRCODE='55000';
  END IF;
 END LOOP;
END $pre$;

CREATE FUNCTION vec_identidad_sesiones_v1.cotejar_autenticacion_historica_portal_externo_v1(p_vinculo_canonico bytea)
 RETURNS boolean
 LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
 SET search_path=pg_catalog, pg_temp
 SET row_security=on
 SET statement_timeout='5s'
 SET lock_timeout='2s'
AS $historia$
DECLARE
 doc jsonb; v_aut text; v_ses text; v_ase text; v_sha text; v_cta text;
 externo boolean; interno boolean; s record; instante timestamptz;
 inicio timestamptz:=clock_timestamp();
BEGIN
 IF current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR current_setting('TimeZone')<>'UTC' THEN
   RAISE EXCEPTION 'cotejo historico requiere SERIALIZABLE READ WRITE UTC' USING ERRCODE='25000';
 END IF;
 IF p_vinculo_canonico IS NULL OR octet_length(p_vinculo_canonico) NOT BETWEEN 1 AND 16384 THEN
   RAISE EXCEPTION 'vinculo historico invalido' USING ERRCODE='22023';
 END IF;
 doc:=convert_from(p_vinculo_canonico,'UTF8')::jsonb;
 -- Mismo esquema de 31 claves que IS20, restringido a la superficie externa.
 IF jsonb_typeof(doc) IS DISTINCT FROM 'object'
    OR (SELECT count(*) FROM jsonb_object_keys(doc))<>31
    OR NOT (doc ?& ARRAY[
      'esquema','bloque_version','autenticacion_ref','autenticacion_huella_sha256',
      'asercion_ref','sesion_ref','control_sesion_ref','control_sesion_revision',
      'control_sesion_huella_sha256','cuenta_ref','cuenta_ordinaria_ref',
      'principal_id','perfil_activo_ref','cuenta_privilegiada','superficie',
      'metodo_observado','garantia_observada','politica_garantia_ref',
      'politica_garantia_huella_sha256','autenticacion_verificada_en',
      'sesion_emitida_en','sesion_valida_hasta','sesion_revalidada_en',
      'registro_contexto_ref','contexto_actor_esquema','contexto_actor_ref',
      'contexto_actor_version','contexto_actor_cuenta_version',
      'contexto_actor_huella_sha256','manifiesto_procedencia_huella_sha256',
      'autoridad_efectiva'])
    OR doc->>'esquema' IS DISTINCT FROM 'vec.autenticacion-actor.vinculo.v2.contexto-registrado'
    OR doc->'bloque_version' IS DISTINCT FROM '2'::jsonb
    OR doc->>'superficie' IS DISTINCT FROM 'externa_personal'
    OR doc->'cuenta_privilegiada' IS DISTINCT FROM 'false'::jsonb
    OR doc->>'cuenta_ref' IS DISTINCT FROM doc->>'cuenta_ordinaria_ref' THEN
   RAISE EXCEPTION 'esquema de vinculo historico invalido' USING ERRCODE='22023';
 END IF;
 v_aut:=doc->>'autenticacion_ref'; v_ses:=doc->>'sesion_ref';
 v_sha:=doc->>'autenticacion_huella_sha256'; v_cta:=doc->>'cuenta_ref'; v_ase:=doc->>'asercion_ref';
 IF vec_identidad_sesiones_v1.referencia_valida(v_aut,'aut_') IS NOT TRUE
    OR vec_identidad_sesiones_v1.referencia_valida(v_ses,'ses_') IS NOT TRUE
    OR vec_identidad_sesiones_v1.referencia_valida(v_cta,'cta_') IS NOT TRUE
    OR vec_identidad_sesiones_v1.referencia_valida(v_ase,'ase_') IS NOT TRUE
    OR v_sha IS NULL OR v_sha !~ '^[0-9a-f]{64}$' OR v_sha=repeat('0',64) THEN
   RAISE EXCEPTION 'selector historico invalido' USING ERRCODE='22023';
 END IF;
 PERFORM pg_advisory_xact_lock_shared(hashtextextended(
   'vec_identidad_sesiones_v1:migracion:cotejo_auditoria_intentos:v1',0));
 -- Despacho: cualquiera de las tres referencias en cualquiera de los almacenes.
 -- En sesion_autenticacion_v1 solo sesión y autenticación, que tienen índice:
 -- la aserción interna se busca en consumo_asercion (índice único). Así no se
 -- recorre ni se bloquea en SERIALIZABLE la tabla entera de sesiones internas.
 SELECT EXISTS(SELECT 1 FROM vec_identidad_externa_v1.sesion x
               WHERE x.sesion_ref=v_ses OR x.autenticacion_ref=v_aut OR x.asercion_ref=v_ase),
        EXISTS(SELECT 1 FROM vec_autorizacion.sesion_autenticacion_v1 x
               WHERE x.sesion_ref=v_ses OR x.autenticacion_ref=v_aut)
        OR EXISTS(SELECT 1 FROM vec_identidad_sesiones_v1.consumo_asercion x
               WHERE x.sesion_ref=v_ses OR x.autenticacion_ref=v_aut OR x.asercion_ref=v_ase)
   INTO externo, interno;
 IF externo AND interno THEN
   RAISE EXCEPTION 'autenticacion historica en ambos almacenes' USING ERRCODE='42501';
 END IF;
 IF NOT externo AND NOT interno THEN
   RAISE EXCEPTION 'autenticacion historica ausente' USING ERRCODE='42501';
 END IF;
 IF NOT externo THEN
   -- Sesión externa_personal del proceso integrado (almacén interno): solo si
   -- la fila interna es externa, no privilegiada y su cuenta no pertenece al
   -- almacén externo segregado; entonces IS20 sin cambios.
   IF EXISTS(SELECT 1 FROM vec_identidad_externa_v1.cuenta k WHERE k.cuenta_ref=v_cta)
      OR EXISTS(SELECT 1 FROM vec_autorizacion.sesion_autenticacion_v1 x
                WHERE (x.sesion_ref=v_ses OR x.autenticacion_ref=v_aut)
                  AND (x.superficie IS DISTINCT FROM 'externa_personal'
                    OR x.cuenta_privilegiada IS DISTINCT FROM false
                    OR EXISTS(SELECT 1 FROM vec_identidad_externa_v1.cuenta k
                              WHERE k.cuenta_ref IN (x.cuenta_ref,x.cuenta_ordinaria_ref)))) THEN
     RAISE EXCEPTION 'autenticacion historica de cuenta externa en almacen interno' USING ERRCODE='42501';
   END IF;
   RETURN vec_identidad_sesiones_v1.cotejar_autenticacion_historica_auditoria_v1(p_vinculo_canonico);
 END IF;
 -- Rama externa: historia de solo adición de IS07, sin cerrojos de fila ni
 -- punteros actuales; la cuenta se mira en la revisión fijada por la sesión.
 SELECT s0.sesion_ref, s0.autenticacion_ref, s0.asercion_ref, s0.operacion_ref,
        s0.cuenta_ref, s0.cuenta_revision, s0.dominio_hmac_ref,
        s0.metodo_observado, s0.garantia_observada, s0.autenticacion_huella_sha256,
        s0.autenticacion_verificada_en, s0.sesion_emitida_en,
        s0.politica_garantia_ref, s0.politica_garantia_huella_sha256, s0.registrada_en,
        c.control_sesion_ref, c.revision AS control_revision, c.estado AS control_estado,
        c.huella_sha256 AS control_huella, c.sesion_revalidada_en, c.sesion_valida_hasta,
        e.estado AS estado_cuenta, e.registrada_en AS estado_registrado_en
   INTO s
   FROM vec_identidad_externa_v1.sesion s0
   JOIN vec_identidad_externa_v1.control_sesion c ON c.sesion_ref=s0.sesion_ref
   JOIN vec_identidad_externa_v1.cuenta k
     ON k.cuenta_ref=s0.cuenta_ref AND k.dominio_hmac_ref=s0.dominio_hmac_ref
   JOIN vec_identidad_externa_v1.estado_cuenta e
     ON e.cuenta_ref=s0.cuenta_ref AND e.revision=s0.cuenta_revision
  WHERE s0.autenticacion_ref=v_aut AND s0.sesion_ref=v_ses
    AND s0.asercion_ref=v_ase AND s0.autenticacion_huella_sha256=v_sha;
 -- El despacho ya la encontró por alguna referencia: lo que falta es un
 -- vínculo que no corresponde a la sesión registrada, no indisponibilidad.
 IF NOT FOUND THEN
   RAISE EXCEPTION 'autenticacion historica divergente' USING ERRCODE='42501';
 END IF;
 IF s.control_revision<>1 OR s.control_estado<>'activa'
    OR s.estado_cuenta<>'activa' OR s.estado_registrado_en>s.sesion_revalidada_en
    OR vec_identidad_sesiones_v1.referencia_valida(s.asercion_ref,'ase_') IS NOT TRUE
    OR vec_identidad_sesiones_v1.referencia_valida(s.control_sesion_ref,'cse_') IS NOT TRUE
    OR vec_identidad_sesiones_v1.referencia_valida(s.operacion_ref,'opr_') IS NOT TRUE
    OR s.metodo_observado NOT IN ('certificado','dnie','sso','clave','kerberos_ad')
    OR s.garantia_observada NOT IN ('sustancial','alto')
    OR s.control_huella IS DISTINCT FROM vec_identidad_sesiones_v1.huella_control_sesion_v1(
         s.control_sesion_ref,1,s.sesion_ref,'activa',s.sesion_revalidada_en,
         s.sesion_valida_hasta,s.operacion_ref)
    OR s.sesion_revalidada_en IS DISTINCT FROM s.registrada_en
    OR s.autenticacion_verificada_en>s.sesion_emitida_en
    OR s.sesion_revalidada_en<s.sesion_emitida_en
    OR s.sesion_valida_hasta<=s.sesion_revalidada_en
    OR s.sesion_valida_hasta>s.sesion_emitida_en+interval '5 minutes'
    OR s.sesion_revalidada_en>=s.autenticacion_verificada_en+interval '12 hours' THEN
   RAISE EXCEPTION 'autenticacion historica incoherente' USING ERRCODE='22023';
 END IF;
 FOREACH instante IN ARRAY ARRAY[s.autenticacion_verificada_en,s.sesion_emitida_en,
    s.sesion_valida_hasta,s.sesion_revalidada_en] LOOP
   IF instante IS NULL OR NOT isfinite(instante)
      OR instante<timestamptz '0001-01-01 00:00:00+00'
      OR instante>=timestamptz '10000-01-01 00:00:00+00' THEN
     RAISE EXCEPTION 'fecha historica invalida' USING ERRCODE='22023';
   END IF;
 END LOOP;
 IF jsonb_typeof(doc->'control_sesion_revision') IS DISTINCT FROM 'number'
    OR doc->>'asercion_ref' IS DISTINCT FROM s.asercion_ref
    OR doc->>'control_sesion_ref' IS DISTINCT FROM s.control_sesion_ref
    OR (doc->>'control_sesion_revision')::numeric IS DISTINCT FROM 1
    OR doc->>'control_sesion_huella_sha256' IS DISTINCT FROM s.control_huella
    OR v_cta IS DISTINCT FROM s.cuenta_ref
    OR doc->>'metodo_observado' IS DISTINCT FROM s.metodo_observado
    OR doc->>'garantia_observada' IS DISTINCT FROM s.garantia_observada
    OR doc->>'politica_garantia_ref' IS DISTINCT FROM s.politica_garantia_ref
    OR doc->>'politica_garantia_huella_sha256' IS DISTINCT FROM s.politica_garantia_huella_sha256
    OR (doc->>'autenticacion_verificada_en')::timestamptz IS DISTINCT FROM s.autenticacion_verificada_en
    OR (doc->>'sesion_emitida_en')::timestamptz IS DISTINCT FROM s.sesion_emitida_en
    OR (doc->>'sesion_valida_hasta')::timestamptz IS DISTINCT FROM s.sesion_valida_hasta
    OR (doc->>'sesion_revalidada_en')::timestamptz IS DISTINCT FROM s.sesion_revalidada_en THEN
   RAISE EXCEPTION 'vinculo historico divergente' USING ERRCODE='22023';
 END IF;
 -- El reloj actual solo limita la ejecución. No exige permiso actual.
 IF clock_timestamp()>inicio+interval '5 seconds' THEN
   RAISE EXCEPTION 'lectura historica fuera de ventana' USING ERRCODE='57014';
 END IF;
 RETURN true;
EXCEPTION
 WHEN no_data_found OR too_many_rows THEN
   RAISE EXCEPTION 'autenticacion historica no disponible' USING ERRCODE='P0002';
 WHEN data_exception THEN
   RAISE EXCEPTION 'autenticacion historica invalida' USING ERRCODE='22023';
END
$historia$;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.cotejar_autenticacion_historica_portal_externo_v1(bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_identidad_sesiones_v1.cotejar_autenticacion_historica_portal_externo_v1(bytea)
 TO vec_autorizacion_atestada_v3_propietario;
RESET ROLE;

DO $post$
DECLARE
 f oid:='vec_identidad_sesiones_v1.cotejar_autenticacion_historica_portal_externo_v1(bytea)'::regprocedure;
 i oid:='vec_identidad_sesiones_v1.cotejar_autenticacion_historica_auditoria_v1(bytea)'::regprocedure;
 o oid:='vec_identidad_sesiones_v1_propietario'::regrole;
 a oid:='vec_autorizacion_atestada_v3_propietario'::regrole;
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.proowner=o AND p.prosecdef
       AND p.provolatile='v' AND p.proparallel='u' AND p.prorettype='boolean'::regtype
       AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on',
                             'statement_timeout=5s','lock_timeout=2s']::text[])
    OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) x WHERE p.oid=f)<>2
    OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) x WHERE p.oid=f
       AND (x.grantee NOT IN (o,a) OR x.grantor<>o OR x.privilege_type<>'EXECUTE' OR x.is_grantable))
    OR has_function_privilege('vec_identidad_sesiones_v1_revalidador',f,'EXECUTE')
    OR has_function_privilege('vec_autorizacion_propietario',f,'EXECUTE')
    OR (to_regrole('vec_identidad_sesiones_v1_lector_historico') IS NOT NULL
        AND has_function_privilege('vec_identidad_sesiones_v1_lector_historico',f,'EXECUTE'))
    OR (to_regrole('vec_bolsa_llamamientos_propietario') IS NOT NULL
        AND has_function_privilege('vec_bolsa_llamamientos_propietario',f,'EXECUTE'))
    OR encode(sha256(convert_to(pg_get_functiondef(i),'UTF8')),'hex')
       IS DISTINCT FROM 'bff185afc93d70e7f5d1d4a8604f8d5d434aa017b624cbf5d122ce069bea6b24'
    -- Posimagen exacta de la función nueva.
    OR encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex')
       IS DISTINCT FROM 'ca870f4ee3ba01c22a5ab916d94954ebfac19a48f24dc702db5862898508d142'
 THEN RAISE EXCEPTION 'IS22: posimagen o ACL distinta' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
