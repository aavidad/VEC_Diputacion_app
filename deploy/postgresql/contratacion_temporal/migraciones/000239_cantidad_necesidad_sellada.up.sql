\set ON_ERROR_STOP on
-- F1-73: cantidad de la necesidad sellada en la solicitud de alta.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000239',0));
DO $pre$ DECLARE r record; v_oid text; v_oids text := ''; BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario'
    OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999 THEN
  RAISE EXCEPTION 'CT239: version PostgreSQL incompatible' USING ERRCODE='55000'; END IF;
 FOR r IN SELECT * FROM (VALUES
  ('vec_contratacion_temporal.datos_puesto_peticion_validos_ct200(jsonb)','c44d2fde96c5519556da232e4b9f1a9f11463d9953628c36ba2ff3ece2860cc3','{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario}',ARRAY['search_path=pg_catalog, pg_temp']::text[],false,'i',true,'u',false),
  ('vec_contratacion_temporal.reconstruir_solicitud_efecto_v2(jsonb)','27be1ce9ddcfc29ecea84873cdd86051994c1bc334b3270d8e8af1731dbfb08f','{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario}',ARRAY['search_path=pg_catalog']::text[],false,'i',true,'u',false),
  ('vec_contratacion_temporal.confirmar_alta_atestada_v1(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea)','7ac9119b0957ce300ee66a904015893bdeb186a575e27e54fa786689576f51cf','{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario}',ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']::text[],true,'v',false,'u',true),
  ('vec_contratacion_temporal.expediente_analisis_valido_v2(jsonb,boolean)','9375556f82f1ed8dc61249ada052e2f7ea77744bd62b994684743b0eae3a7fc1','{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario}',ARRAY['search_path=pg_catalog']::text[],false,'i',false,'u',false)
 ) AS x(firma,sha,acl,config,secdef,vol,strict,parallel,retset) LOOP
  IF pg_catalog.to_regprocedure(r.firma) IS NULL THEN
   RAISE EXCEPTION 'CT239: funcion ausente %',r.firma USING ERRCODE='55000'; END IF;
  SELECT p.oid::text INTO v_oid FROM pg_catalog.pg_proc p
   WHERE p.oid=pg_catalog.to_regprocedure(r.firma)
    AND p.proowner='vec_contratacion_temporal_propietario'::regrole
    AND p.prosecdef=r.secdef AND p.provolatile=r.vol
    AND p.proisstrict=r.strict AND p.proparallel=r.parallel
    AND p.proretset=r.retset
    AND p.proacl::text=r.acl AND p.proconfig=r.config
    AND pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
      pg_catalog.pg_get_functiondef(p.oid),'UTF8')),'hex')=r.sha;
  IF v_oid IS NULL THEN RAISE EXCEPTION 'CT239: preimagen incompatible %',r.firma
   USING ERRCODE='55000'; END IF;
  v_oids:=v_oids||'/'||v_oid; v_oid:=NULL;
 END LOOP;
 PERFORM pg_catalog.set_config('vec.ct239.oids',v_oids,true);
END $pre$;

CREATE OR REPLACE FUNCTION vec_contratacion_temporal.datos_puesto_peticion_validos_ct200(s jsonb)
 RETURNS boolean
 LANGUAGE sql
 IMMUTABLE STRICT
 SET search_path TO 'pg_catalog', 'pg_temp'
AS $function$
 SELECT CASE
 WHEN s ? 'numero_personas'
      AND NOT (s ?| ARRAY['jornada_minutos','puesto_solicitado']) THEN
   (CASE WHEN pg_catalog.jsonb_typeof(s->'numero_personas')='number'
               AND s->>'numero_personas' ~ '^[1-9][0-9]{0,9}$'
      THEN (s->>'numero_personas')::numeric BETWEEN 1 AND 4294967295
      ELSE false END)
   AND s ? 'necesidad'
   AND vec_contratacion_temporal.necesidad_alta_valida_v3(s) IS TRUE
   AND s->>'numero_personas' = s#>>'{necesidad,campos,numero_personas}'
 WHEN s ?| ARRAY['jornada_minutos','numero_personas','puesto_solicitado'] THEN
   s ?& ARRAY['jornada_minutos','numero_personas','puesto_solicitado']
   AND (CASE WHEN pg_catalog.jsonb_typeof(s->'jornada_minutos')='number'
                   AND s->>'jornada_minutos' ~ '^[0-9]+$'
             THEN (s->>'jornada_minutos')::numeric BETWEEN 1 AND 10080 ELSE false END)
   AND (CASE WHEN pg_catalog.jsonb_typeof(s->'numero_personas')='number'
                   AND s->>'numero_personas' ~ '^[0-9]+$'
             THEN (s->>'numero_personas')::numeric BETWEEN 1 AND 4294967295 ELSE false END)
   AND pg_catalog.jsonb_typeof(s->'puesto_solicitado')='string'
   AND pg_catalog.length(s->>'puesto_solicitado') BETWEEN 1 AND 160
   AND pg_catalog.btrim(s->>'puesto_solicitado')=s->>'puesto_solicitado'
   AND (s->>'puesto_solicitado') !~ '[[:cntrl:]]'
 ELSE true END
$function$
;

CREATE OR REPLACE FUNCTION vec_contratacion_temporal.reconstruir_solicitud_efecto_v2(s jsonb)
 RETURNS text
 LANGUAGE sql
 IMMUTABLE STRICT
 SET search_path TO 'pg_catalog'
AS $function$
    SELECT
      '{"centro_ref":' || vec_contratacion_temporal.texto_json_go_v1(s ->> 'centro_ref') ||
      ',"contacto_ref":' || vec_contratacion_temporal.texto_json_go_v1(s ->> 'contacto_ref') ||
      ',"categoria_ref":' || vec_contratacion_temporal.texto_json_go_v1(s ->> 'categoria_ref') ||
      ',"grupo_subgrupo":' || vec_contratacion_temporal.texto_json_go_v1(s ->> 'grupo_subgrupo') ||
      ',"motivo_clave":' || vec_contratacion_temporal.texto_json_go_v1(s ->> 'motivo_clave') ||
      ',"detalle":' || vec_contratacion_temporal.texto_json_go_v1(s ->> 'detalle') ||
      ',"periodo":{"inicio":' || vec_contratacion_temporal.texto_json_go_v1(s #>> '{periodo,inicio}') ||
      (CASE WHEN s #> '{periodo,fin}' IS NOT NULL THEN
          ',"fin":' || vec_contratacion_temporal.texto_json_go_v1(s #>> '{periodo,fin}')
        ELSE
          ',"causa_fin":' || vec_contratacion_temporal.texto_json_go_v1(s #>> '{periodo,causa_fin}')
        END) ||
        (CASE WHEN s #> '{periodo,politica_fin}' IS NOT NULL THEN
          ',"politica_fin":{"regla_ref":' ||
          vec_contratacion_temporal.texto_json_go_v1(s #>> '{periodo,politica_fin,regla_ref}') ||
          ',"catalogo_version":' || (s #> '{periodo,politica_fin,catalogo_version}')::text ||
          ',"catalogo_huella_sha256":' ||
          vec_contratacion_temporal.texto_json_go_v1(s #>> '{periodo,politica_fin,catalogo_huella_sha256}') ||
          ',"fecha_fin":' ||
          vec_contratacion_temporal.texto_json_go_v1(s #>> '{periodo,politica_fin,fecha_fin}') ||
          (CASE WHEN s #> '{periodo,politica_fin,causa_fin}' IS NOT NULL THEN
            ',"causa_fin":' ||
            vec_contratacion_temporal.texto_json_go_v1(s #>> '{periodo,politica_fin,causa_fin}')
           ELSE '' END) || '}'
         ELSE '' END) || '}' ||
      ',"rc":{"existe":' || (s #> '{rc,existe}')::text ||
      ',"numero":' || vec_contratacion_temporal.texto_json_go_v1(s #>> '{rc,numero}') ||
      ',"fecha":' || vec_contratacion_temporal.texto_json_go_v1(s #>> '{rc,fecha}') ||
      ',"importe":{"centimos":' || (s #> '{rc,importe,centimos}')::text ||
      ',"moneda":' || vec_contratacion_temporal.texto_json_go_v1(s #>> '{rc,importe,moneda}') || '}' ||
      ',"documento_ref":' || vec_contratacion_temporal.texto_json_go_v1(s #>> '{rc,documento_ref}') || '}' ||
      ',"documentos_adjuntos":' ||
        vec_contratacion_temporal.lista_textos_json_v1(
            s -> 'documentos_adjuntos'
        ) ||
      ',"observaciones":' || vec_contratacion_temporal.texto_json_go_v1(s ->> 'observaciones') ||
      (CASE WHEN s ? 'puesto_solicitado' THEN
        ',"jornada_minutos":' || (s -> 'jornada_minutos')::text ||
        ',"numero_personas":' || (s -> 'numero_personas')::text ||
        ',"puesto_solicitado":' || vec_contratacion_temporal.texto_json_go_v1(s ->> 'puesto_solicitado')
       ELSE CASE WHEN s ? 'numero_personas' THEN
         ',"numero_personas":' || (s -> 'numero_personas')::text
       ELSE '' END END) || '}'
$function$
;

CREATE OR REPLACE FUNCTION vec_contratacion_temporal.confirmar_alta_atestada_v1(p_capacidad_canonica bytea, p_decision_canonica bytea, p_motivo_canonico bytea, p_contexto_actor_canonico bytea, p_persona_version numeric, p_perfil_version numeric, p_payload_vec_ad_3 bytea, p_sobre_cose_sign1 bytea, p_evidencia_verificacion bytea, p_raiz_publica_spki bytea, p_alta_canonica bytea, p_sellos_hmac_canonicos bytea)
 RETURNS TABLE(expediente_ref text, numero_visible text, version numeric, recibo_ref text, auditoria_ref text, evento_ref text, confirmada_en timestamp with time zone, recibo_huella_sha256 text)
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET lock_timeout TO '2s'
AS $function$
DECLARE
    a jsonb;
    s jsonb;
    d jsonb;
    v_consumo record;
    v_identidad record;
    v_par jsonb;
    v_pares jsonb;
    v_generaciones integer[];
    v_generaciones_politica integer[];
    v_aliases text[];
    v_raices text[];
    v_raiz text;
    v_activo_ambito text;
    v_activo_huella text;
    v_huella_alta text;
    v_huella_contexto_recurso text;
    v_contexto_recurso bytea;
    v_ahora timestamptz(6);
    v_revision bigint;
    v_auditoria_ref text;
    v_evento_ref text;
    v_anterior_auditoria text;
    v_anterior_outbox text;
    v_secuencia_auditoria numeric(20, 0);
    v_secuencia_outbox numeric(20, 0);
    v_huella_auditoria text;
    v_huella_outbox text;
    v_payload_outbox bytea;
    v_huella_payload text;
    v_confirmacion_ref text;
    v_huella_actuacion text;
    v_huella_agregado text;
    v_recibo_huella text;
    v_statement numeric;
    v_idle numeric;
BEGIN
    IF pg_catalog.current_setting('transaction_isolation') <> 'serializable'
       OR pg_catalog.current_setting('transaction_read_only') <> 'off'
       OR pg_catalog.current_setting('TimeZone') <> 'UTC'
       OR session_user = current_user
       OR NOT pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER')
       OR pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_migrador', 'MEMBER')
       OR pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_propietario', 'MEMBER') THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'confirmación de alta rechazada';
    END IF;
    SELECT setting::numeric INTO v_statement
      FROM pg_catalog.pg_settings
     WHERE name = 'statement_timeout' AND unit = 'ms';
    SELECT setting::numeric INTO v_idle
      FROM pg_catalog.pg_settings
     WHERE name = 'idle_in_transaction_session_timeout' AND unit = 'ms';
    IF v_statement IS NULL OR v_statement NOT BETWEEN 1 AND 15000
       OR v_idle IS NULL OR v_idle NOT BETWEEN 1 AND 20000
       OR pg_catalog.octet_length(p_alta_canonica) NOT BETWEEN 256 AND 32768
       OR pg_catalog.octet_length(p_sellos_hmac_canonicos)
          NOT BETWEEN 256 AND 8192 THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'entrada de alta inválida';
    END IF;
    BEGIN
        a := pg_catalog.convert_from(p_alta_canonica, 'UTF8')::jsonb;
        s := pg_catalog.convert_from(p_sellos_hmac_canonicos, 'UTF8')::jsonb;
        d := pg_catalog.convert_from(p_decision_canonica, 'UTF8')::jsonb;
    EXCEPTION
        WHEN data_exception OR invalid_text_representation
          OR character_not_in_repertoire OR untranslatable_character THEN
            RAISE EXCEPTION USING
                ERRCODE = '22023',
                MESSAGE = 'entrada de alta inválida';
    END;
    IF pg_catalog.jsonb_typeof(a) <> 'object'
       OR (SELECT pg_catalog.count(*)
             FROM pg_catalog.jsonb_object_keys(a)) <> 16
       OR NOT (a ?& ARRAY[
           'esquema', 'reserva_ref', 'expediente_ref', 'numero_visible',
           'recibo_ref', 'organizacion_ref', 'actor_ref', 'perfil_ref',
           'version', 'flujo', 'fase_actual', 'estado_actual',
           'solicitud', 'creado_en', 'actualizado_en', 'actuacion'
       ])
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.jsonb_object_keys(a) AS k(clave)
            WHERE CASE WHEN k.clave IN (
                'version', 'flujo', 'solicitud', 'actuacion') THEN false
            ELSE pg_catalog.jsonb_typeof(a -> k.clave) <> 'string'
            END)
       OR pg_catalog.jsonb_typeof(a -> 'version') <> 'number'
       OR pg_catalog.jsonb_typeof(a -> 'flujo') <> 'object'
       OR (SELECT pg_catalog.count(*)
             FROM pg_catalog.jsonb_object_keys(a -> 'flujo')) <> 3
       OR NOT ((a -> 'flujo') ?& ARRAY[
           'definicion_ref', 'version', 'huella_sha256'
       ])
       OR pg_catalog.jsonb_typeof(a #> '{flujo,definicion_ref}')
          <> 'string'
       OR pg_catalog.jsonb_typeof(a #> '{flujo,version}') <> 'number'
       OR pg_catalog.jsonb_typeof(a #> '{flujo,huella_sha256}')
          <> 'string'
       OR pg_catalog.jsonb_typeof(a -> 'solicitud') <> 'object'
       OR (SELECT pg_catalog.count(*)
             FROM pg_catalog.jsonb_object_keys(a -> 'solicitud')) <>
       (CASE WHEN a ->> 'esquema' = 'vec.contratacion-temporal.efecto-alta.v3'
             THEN 11 ELSE 10 END) +
       (CASE WHEN (a -> 'solicitud') ? 'puesto_solicitado' THEN 3 ELSE 0 END) +
       (CASE WHEN a ->> 'esquema' = 'vec.contratacion-temporal.efecto-alta.v3'
              AND (a -> 'solicitud') ? 'numero_personas'
              AND NOT ((a -> 'solicitud') ?| ARRAY['jornada_minutos','puesto_solicitado'])
              AND vec_contratacion_temporal.datos_puesto_peticion_validos_ct200(
                    a -> 'solicitud') IS TRUE
             THEN 1 ELSE 0 END)
       OR NOT ((a -> 'solicitud') ?& ARRAY[
           'centro_ref', 'contacto_ref', 'categoria_ref',
           'grupo_subgrupo', 'motivo_clave', 'detalle', 'periodo', 'rc',
           'documentos_adjuntos', 'observaciones'
       ])
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.jsonb_object_keys(a -> 'solicitud') AS k(clave)
            WHERE CASE WHEN k.clave IN (
                'periodo', 'rc', 'documentos_adjuntos', 'necesidad',
                'jornada_minutos', 'numero_personas') THEN false
            ELSE pg_catalog.jsonb_typeof(
                (a -> 'solicitud') -> k.clave) <> 'string'
            END)
       OR vec_contratacion_temporal.datos_puesto_peticion_validos_ct200(a -> 'solicitud') IS DISTINCT FROM true
       OR pg_catalog.jsonb_typeof(a #> '{solicitud,periodo}') <> 'object'
       OR NOT vec_contratacion_temporal.periodo_previsto_estructural_v1(a #> '{solicitud,periodo}')
       OR pg_catalog.jsonb_typeof(a #> '{solicitud,rc}') <> 'object'
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(
               a #> '{solicitud,rc}')) <> 5
       OR NOT ((a #> '{solicitud,rc}') ?& ARRAY[
           'existe', 'numero', 'fecha', 'importe', 'documento_ref'
       ])
       OR pg_catalog.jsonb_typeof(a #> '{solicitud,rc,existe}')
          <> 'boolean'
       OR pg_catalog.jsonb_typeof(a #> '{solicitud,rc,numero}')
          <> 'string'
       OR pg_catalog.jsonb_typeof(a #> '{solicitud,rc,fecha}')
          <> 'string'
       OR pg_catalog.jsonb_typeof(a #> '{solicitud,rc,documento_ref}')
          <> 'string'
       OR pg_catalog.jsonb_typeof(a #> '{solicitud,rc,importe}')
          <> 'object'
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(
               a #> '{solicitud,rc,importe}')) <> 2
       OR NOT ((a #> '{solicitud,rc,importe}') ?&
               ARRAY['centimos', 'moneda'])
       OR pg_catalog.jsonb_typeof(
           a #> '{solicitud,rc,importe,centimos}') <> 'number'
       OR pg_catalog.jsonb_typeof(a #> '{solicitud,rc,importe,moneda}')
          <> 'string'
       OR pg_catalog.jsonb_typeof(
           a #> '{solicitud,documentos_adjuntos}') <> 'array'
       OR pg_catalog.jsonb_array_length(
           a #> '{solicitud,documentos_adjuntos}') > 100
       OR EXISTS (
           SELECT 1 FROM pg_catalog.jsonb_array_elements(
               a #> '{solicitud,documentos_adjuntos}') AS e(valor)
           WHERE pg_catalog.jsonb_typeof(e.valor) <> 'string'
              OR e.valor #>> '{}' !~
                 '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$')
       OR pg_catalog.jsonb_typeof(a -> 'actuacion') <> 'object'
       OR (SELECT pg_catalog.count(*)
             FROM pg_catalog.jsonb_object_keys(a -> 'actuacion')) <> 13
       OR NOT ((a -> 'actuacion') ?& ARRAY[
           'secuencia', 'version_expediente', 'accion_clave', 'actor_ref',
           'unidad_ref', 'recibo_ref', 'realizada_en', 'fase_origen',
           'fase_destino', 'estado_origen', 'estado_destino',
           'observaciones', 'documentos_ref'
       ])
       OR pg_catalog.jsonb_typeof(a #> '{actuacion,secuencia}')
          <> 'number'
       OR pg_catalog.jsonb_typeof(a #> '{actuacion,version_expediente}')
          <> 'number'
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.jsonb_object_keys(a -> 'actuacion') AS k(clave)
            WHERE CASE WHEN k.clave IN (
                'secuencia', 'version_expediente', 'documentos_ref') THEN false
            ELSE pg_catalog.jsonb_typeof(
                (a -> 'actuacion') -> k.clave) <> 'string'
            END)
       OR pg_catalog.jsonb_typeof(a #> '{actuacion,documentos_ref}')
          <> 'array'
       OR pg_catalog.jsonb_array_length(
           a #> '{actuacion,documentos_ref}') > 100
       OR EXISTS (
           SELECT 1 FROM pg_catalog.jsonb_array_elements(
               a #> '{actuacion,documentos_ref}') AS e(valor)
           WHERE pg_catalog.jsonb_typeof(e.valor) <> 'string'
              OR e.valor #>> '{}' !~
                 '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$')
       OR vec_contratacion_temporal.reconstruir_efecto_segun_esquema_v3(a)
          IS DISTINCT FROM p_alta_canonica
       OR a ->> 'esquema' NOT IN (
          'vec.contratacion-temporal.efecto-alta.v2',
          'vec.contratacion-temporal.efecto-alta.v3')
       OR (a ->> 'esquema' = 'vec.contratacion-temporal.efecto-alta.v3'
           AND vec_contratacion_temporal.necesidad_alta_valida_v3(a -> 'solicitud') IS NOT TRUE)
       OR (a ->> 'version') !~ '^[1-9][0-9]{0,15}$'
       OR (a ->> 'version')::numeric <> 1
       OR (a #>> '{flujo,version}') !~ '^[1-9][0-9]{0,15}$'
       OR (a #>> '{flujo,version}')::numeric >
          9007199254740991::numeric
       OR (a #>> '{actuacion,secuencia}') !~ '^[1-9][0-9]{0,15}$'
       OR (a #>> '{actuacion,secuencia}')::numeric <> 1
       OR (a #>> '{actuacion,version_expediente}')
          !~ '^[1-9][0-9]{0,15}$'
       OR (a #>> '{actuacion,version_expediente}')::numeric <> 1
       OR (a #>> '{solicitud,rc,importe,centimos}')
          !~ '^(0|[1-9][0-9]{0,15})$'
       OR (a #>> '{solicitud,rc,importe,centimos}')::numeric >
          9007199254740991::numeric
       OR a ->> 'estado_actual' <> 'en_curso'
       OR a ->> 'numero_visible' !~
          '^[0-9]{4}/[A-Za-z0-9._-]{1,40}$'
       OR EXISTS (
           SELECT 1 FROM pg_catalog.unnest(ARRAY[
               a ->> 'reserva_ref', a ->> 'expediente_ref',
               a ->> 'recibo_ref', a ->> 'organizacion_ref',
               a #>> '{solicitud,centro_ref}',
               a #>> '{solicitud,categoria_ref}',
               a ->> 'actor_ref', a ->> 'perfil_ref',
               a #>> '{flujo,definicion_ref}', a ->> 'fase_actual',
               a #>> '{actuacion,accion_clave}',
               a #>> '{actuacion,unidad_ref}'
           ]) AS r(valor)
           WHERE r.valor !~
             '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$')
       OR a #>> '{flujo,huella_sha256}' !~ '^[0-9a-f]{64}$'
       OR a ->> 'creado_en' !~
          '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}[.][0-9]{6}Z$'
       OR a ->> 'actualizado_en' <> a ->> 'creado_en'
       OR a #>> '{actuacion,realizada_en}' <> a ->> 'creado_en'
       OR a #>> '{actuacion,actor_ref}' <> a ->> 'actor_ref'
       OR a #>> '{actuacion,recibo_ref}' <> a ->> 'recibo_ref'
       OR a #>> '{actuacion,fase_origen}' <> ''
       OR a #>> '{actuacion,fase_destino}' <> a ->> 'fase_actual'
       OR a #>> '{actuacion,estado_origen}' <> 'pendiente'
       OR a #>> '{actuacion,estado_destino}' <> a ->> 'estado_actual'
       OR a ->> 'actor_ref' <> d ->> 'principal_id'
       OR a ->> 'perfil_ref' <> d ->> 'perfil_activo_ref' THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'proyección de alta inválida';
    END IF;
    IF pg_catalog.jsonb_typeof(s) <> 'object'
       OR (SELECT pg_catalog.count(*)
             FROM pg_catalog.jsonb_object_keys(s)) <> 3
       OR NOT (s ?& ARRAY['esquema', 'activo', 'retenidos'])
       OR s ->> 'esquema' <>
          'vec.contratacion-temporal.sellos-hmac.v1'
       OR pg_catalog.jsonb_typeof(s -> 'activo') <> 'object'
       OR pg_catalog.jsonb_typeof(s -> 'retenidos') <> 'array'
       OR pg_catalog.jsonb_array_length(s -> 'retenidos') > 3
       OR vec_contratacion_temporal.reconstruir_sellos_hmac_v1(s)
          IS DISTINCT FROM p_sellos_hmac_canonicos THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'sellos HMAC inválidos';
    END IF;
    v_pares := pg_catalog.jsonb_build_array(s -> 'activo') ||
               (s -> 'retenidos');
    IF EXISTS (
        SELECT 1
          FROM pg_catalog.jsonb_array_elements(v_pares) AS e(valor)
         WHERE pg_catalog.jsonb_typeof(e.valor) <> 'object'
            OR (SELECT pg_catalog.count(*)
                  FROM pg_catalog.jsonb_object_keys(e.valor)) <> 3
            OR NOT (e.valor ?& ARRAY[
                'generacion', 'ambito_hmac', 'huella_hmac'
            ])
            OR pg_catalog.jsonb_typeof(e.valor -> 'generacion')
               <> 'number'
            OR pg_catalog.jsonb_typeof(e.valor -> 'ambito_hmac')
               <> 'string'
            OR pg_catalog.jsonb_typeof(e.valor -> 'huella_hmac')
               <> 'string'
            OR e.valor ->> 'generacion' !~ '^[1-9][0-9]{0,8}$'
            OR (e.valor ->> 'generacion')::numeric >
               9007199254740991::numeric
            OR e.valor ->> 'ambito_hmac' !~
               (
                 '^hmac-sha256:vec[.]contratacion-temporal[.]'
                 || 'ambito-idempotencia/v[1-9][0-9]{0,8}:[a-f0-9]{64}$')
            OR e.valor ->> 'huella_hmac' !~
               (
                 '^hmac-sha256:vec[.]contratacion-temporal[.]'
                 || 'huella-peticion/v[1-9][0-9]{0,8}:[a-f0-9]{64}$')
            OR pg_catalog.right(e.valor ->> 'ambito_hmac', 64) =
               pg_catalog.repeat('0', 64)
            OR pg_catalog.right(e.valor ->> 'huella_hmac', 64) =
               pg_catalog.repeat('0', 64)
            OR substring(
                 e.valor ->> 'ambito_hmac'
                 FROM '/v([1-9][0-9]{0,8}):')::integer <> (e.valor ->> 'generacion')::integer
            OR substring(
                 e.valor ->> 'huella_hmac'
                 FROM '/v([1-9][0-9]{0,8}):')::integer <> (e.valor ->> 'generacion')::integer) THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'sellos HMAC inválidos';
    END IF;
    SELECT pg_catalog.array_agg(
               (e.valor ->> 'generacion')::integer ORDER BY e.orden),
           pg_catalog.array_agg(
               e.valor ->> 'ambito_hmac' ORDER BY e.orden)
      INTO v_generaciones, v_aliases
      FROM pg_catalog.jsonb_array_elements(v_pares)
           WITH ORDINALITY AS e(valor, orden);
    SELECT pg_catalog.array_agg(
               generacion ORDER BY posicion)
      INTO v_generaciones_politica
      FROM vec_contratacion_temporal.politica_generaciones_hmac_alta;
    IF v_generaciones IS DISTINCT FROM v_generaciones_politica
       OR pg_catalog.cardinality(v_generaciones) <>
          pg_catalog.cardinality(
              ARRAY(SELECT DISTINCT x FROM pg_catalog.unnest(
                  v_generaciones) AS u(x))) THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'política HMAC no satisfecha';
    END IF;
    v_activo_ambito := s #>> '{activo,ambito_hmac}';
    v_activo_huella := s #>> '{activo,huella_hmac}';
    v_huella_alta := pg_catalog.encode(
        pg_catalog.sha256(p_alta_canonica), 'hex');
    -- Cierra la ligadura completa del recurso que originó la decisión V3.
    v_contexto_recurso := pg_catalog.convert_to(
        '{"ambitos":{"categoria_ref":' ||
          vec_contratacion_temporal.texto_json_go_v1(
              a #>> '{solicitud,categoria_ref}') ||
        ',"centro_ref":' ||
          vec_contratacion_temporal.texto_json_go_v1(
              a #>> '{solicitud,centro_ref}') ||
        ',"organizacion_ref":' ||
          vec_contratacion_temporal.texto_json_go_v1(
              a ->> 'organizacion_ref') ||
        '},"atributos":{"efecto_huella_sha256":' ||
          vec_contratacion_temporal.texto_json_go_v1(v_huella_alta) ||
        ',"flujo_huella_sha256":' ||
          vec_contratacion_temporal.texto_json_go_v1(
              a #>> '{flujo,huella_sha256}') ||
        ',"flujo_ref":' ||
          vec_contratacion_temporal.texto_json_go_v1(
              a #>> '{flujo,definicion_ref}') ||
        ',"flujo_version":' ||
          vec_contratacion_temporal.texto_json_go_v1(
              a #>> '{flujo,version}') ||
        ',"huella_peticion_hmac_activa":' ||
          vec_contratacion_temporal.texto_json_go_v1(v_activo_huella) ||
        '}}',
        'UTF8');
    v_huella_contexto_recurso := pg_catalog.encode(
        pg_catalog.sha256(v_contexto_recurso), 'hex');
    IF d ->> 'recurso_ref' <> v_activo_ambito
       OR d ->> 'modulo_id' <> 'contratacion_temporal'
       OR d ->> 'tipo_recurso' <> 'expediente_contratacion_temporal'
       OR d ->> 'accion' <> 'contratacion_temporal.solicitud.crear'
       OR d ->> 'finalidad' <> 'tramitar_necesidad_personal_temporal'
       OR d ->> 'contexto_recurso_huella_sha256' <>
          v_huella_contexto_recurso THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'efecto de alta no autorizado';
    END IF;
    SELECT * INTO STRICT v_consumo
      FROM vec_autorizacion_atestada_v3.
           registrar_y_consumir_decision_v3_atestada(
               p_capacidad_canonica, p_decision_canonica,
               p_motivo_canonico, p_contexto_actor_canonico,
               p_persona_version, p_perfil_version,
               p_payload_vec_ad_3, p_sobre_cose_sign1,
               p_evidencia_verificacion, p_raiz_publica_spki);
    IF v_consumo.efecto_ref <> v_activo_ambito
       OR v_consumo.huella_efecto_sha256 <>
          v_huella_contexto_recurso
       OR v_consumo.consumo_nuevo IS NULL THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'consumo de alta incoherente';
    END IF;
    v_confirmacion_ref := 'cnf_ct_' || pg_catalog.substr(
        pg_catalog.encode(pg_catalog.sha256(
            vec_contratacion_temporal.encuadrar_texto_v1(
                v_consumo.decision_ref
            ) ||
            vec_contratacion_temporal.encuadrar_texto_v1(
                a ->> 'expediente_ref'
            ) ||
            vec_contratacion_temporal.encuadrar_texto_v1(
                v_consumo.consumo_huella_sha256
            )
        ), 'hex'), 1, 32
    );
    IF v_consumo.consumo_nuevo IS FALSE THEN
        RETURN QUERY
        SELECT *
          FROM vec_contratacion_temporal.reconciliar_agregado_alta_v1(
              p_alta_canonica, v_activo_ambito, v_activo_huella,
              v_consumo.decision_ref, v_consumo.efecto_ref,
              v_consumo.huella_efecto_sha256,
              v_consumo.consumo_huella_sha256
          );
        RETURN;
    END IF;
    -- Orden total de locks de alias para que todas las sesiones converjan.
    PERFORM pg_catalog.pg_advisory_xact_lock(
        pg_catalog.hashtextextended('vec_ct:alias:' || alias, 0))
      FROM pg_catalog.unnest(v_aliases) AS u(alias)
     ORDER BY alias COLLATE "C";
    SELECT pg_catalog.array_agg(
               DISTINCT ambito_raiz_hmac ORDER BY ambito_raiz_hmac)
      INTO v_raices
      FROM vec_contratacion_temporal.alias_ambito_alta
     WHERE alias_hmac = ANY (v_aliases);
    IF pg_catalog.cardinality(v_raices) > 1 THEN
        RAISE EXCEPTION USING
            ERRCODE = '23505',
            MESSAGE = 'alias HMAC divergentes';
    END IF;
    IF pg_catalog.cardinality(v_raices) = 1 THEN
        v_raiz := v_raices[1];
    ELSE
        v_raiz := v_activo_ambito;
        INSERT INTO vec_contratacion_temporal.identidad_reserva_alta (
            ambito_hmac, reserva_ref, expediente_ref, numero_visible,
            recibo_ref, huella_peticion_hmac, organizacion_ref,
            actor_ref, perfil_ref, creada_en) VALUES (
            v_raiz, a ->> 'reserva_ref', a ->> 'expediente_ref',
            a ->> 'numero_visible', a ->> 'recibo_ref',
            v_activo_huella, a ->> 'organizacion_ref',
            a ->> 'actor_ref', a ->> 'perfil_ref',
            (a #>> '{actuacion,realizada_en}')::timestamptz);
        INSERT INTO vec_contratacion_temporal.reserva_alta_version (
            ambito_hmac, revision, estado, registrada_en) VALUES (v_raiz, 1, 'reservada', clock_timestamp());
        INSERT INTO vec_contratacion_temporal.reserva_alta_actual
            VALUES (v_raiz, 1);
    END IF;
    SELECT * INTO STRICT v_identidad
      FROM vec_contratacion_temporal.identidad_reserva_alta
     WHERE ambito_hmac = v_raiz;
    IF v_identidad.reserva_ref <> a ->> 'reserva_ref'
       OR v_identidad.expediente_ref <> a ->> 'expediente_ref'
       OR v_identidad.numero_visible <> a ->> 'numero_visible'
       OR v_identidad.recibo_ref <> a ->> 'recibo_ref'
       OR v_identidad.organizacion_ref <> a ->> 'organizacion_ref'
       OR v_identidad.actor_ref <> a ->> 'actor_ref'
       OR v_identidad.perfil_ref <> a ->> 'perfil_ref' THEN
        RAISE EXCEPTION USING
            ERRCODE = '23505',
            MESSAGE = 'reserva de alta en conflicto';
    END IF;
    FOR v_par IN
        SELECT e.valor
          FROM pg_catalog.jsonb_array_elements(v_pares)
               WITH ORDINALITY AS e(valor, orden)
         ORDER BY e.orden
    LOOP
        IF EXISTS (
            SELECT 1
              FROM vec_contratacion_temporal.alias_ambito_alta x
             WHERE x.alias_hmac = v_par ->> 'ambito_hmac'
               AND x.ambito_raiz_hmac <> v_raiz) OR EXISTS (
            SELECT 1
              FROM vec_contratacion_temporal.alias_huella_alta x
             WHERE x.ambito_raiz_hmac = v_raiz
               AND x.generacion =
                   (v_par ->> 'generacion')::integer
               AND x.alias_hmac <> v_par ->> 'huella_hmac') THEN
            RAISE EXCEPTION USING
                ERRCODE = '23505',
                MESSAGE = 'par HMAC en conflicto';
        END IF;
        INSERT INTO vec_contratacion_temporal.alias_ambito_alta (
            alias_hmac, ambito_raiz_hmac, generacion, registrada_en) VALUES (
            v_par ->> 'ambito_hmac', v_raiz,
            (v_par ->> 'generacion')::integer, clock_timestamp()) ON CONFLICT DO NOTHING;
        INSERT INTO vec_contratacion_temporal.alias_huella_alta (
            ambito_raiz_hmac, generacion, alias_hmac, registrada_en) VALUES (
            v_raiz, (v_par ->> 'generacion')::integer,
            v_par ->> 'huella_hmac', clock_timestamp()) ON CONFLICT DO NOTHING;
    END LOOP;
    IF EXISTS (
        SELECT 1 FROM vec_contratacion_temporal.expediente_alta e
         WHERE e.expediente_ref = a ->> 'expediente_ref'
    ) OR EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.confirmacion_agregado_alta m
         WHERE m.confirmacion_ref = v_confirmacion_ref
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = 'V2070',
            MESSAGE = 'estado previo de alta incoherente';
    END IF;
    v_ahora := pg_catalog.date_trunc(
        'microseconds', clock_timestamp());
    IF v_ahora < (a #>> '{actuacion,realizada_en}')::timestamptz THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'instante de alta inválido';
    END IF;
    v_auditoria_ref := 'aud_ct_' || pg_catalog.substr(
        pg_catalog.encode(pg_catalog.sha256(
            vec_contratacion_temporal.encuadrar_texto_v1(
                v_consumo.decision_ref) ||
            vec_contratacion_temporal.encuadrar_texto_v1(
                a ->> 'expediente_ref')), 'hex'), 1, 32);
    v_evento_ref := 'evt_ct_' || pg_catalog.substr(
        pg_catalog.encode(pg_catalog.sha256(
            vec_contratacion_temporal.encuadrar_texto_v1(
                a ->> 'expediente_ref') ||
            vec_contratacion_temporal.encuadrar_texto_v1(
                v_consumo.consumo_huella_sha256)), 'hex'), 1, 32);
    INSERT INTO vec_contratacion_temporal.expediente_alta (
        expediente_ref, reserva_ref, numero_visible, organizacion_ref,
        actor_ref, perfil_ref, decision_ref, efecto_ref,
        huella_efecto_sha256, creada_en, confirmacion_ref) VALUES (
        a ->> 'expediente_ref', a ->> 'reserva_ref',
        a ->> 'numero_visible', a ->> 'organizacion_ref',
        a ->> 'actor_ref', a ->> 'perfil_ref',
        v_consumo.decision_ref, v_consumo.efecto_ref,
        v_consumo.huella_efecto_sha256,
        (a ->> 'creado_en')::timestamptz, v_confirmacion_ref);
    INSERT INTO vec_contratacion_temporal.expediente_alta_version (
        expediente_ref, version, alta_canonica, huella_alta_sha256,
        flujo_ref, flujo_version, flujo_huella_sha256, fase_clave,
        estado, solicitud_huella_sha256, registrada_en,
        confirmacion_ref) VALUES (
        a ->> 'expediente_ref', 1, p_alta_canonica, v_huella_alta,
        a #>> '{flujo,definicion_ref}',
        (a #>> '{flujo,version}')::numeric,
        a #>> '{flujo,huella_sha256}', a ->> 'fase_actual',
        a ->> 'estado_actual',
        pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
            vec_contratacion_temporal.reconstruir_solicitud_segun_esquema_v3(
                a -> 'solicitud'), 'UTF8')), 'hex'),
        v_ahora, v_confirmacion_ref);
    v_huella_actuacion := pg_catalog.encode(pg_catalog.sha256(
        vec_contratacion_temporal.encuadrar_texto_v1(
            a ->> 'expediente_ref'
        ) ||
        vec_contratacion_temporal.encuadrar_texto_v1(
            (a #> '{actuacion}')::text
        )
    ), 'hex');
    INSERT INTO vec_contratacion_temporal.actuacion_alta (
        expediente_ref, secuencia, version_expediente, accion_clave,
        actor_ref, unidad_ref, recibo_ref, fase_destino,
        estado_destino, realizada_en, huella_sha256,
        confirmacion_ref) VALUES (
        a ->> 'expediente_ref', 1, 1,
        a #>> '{actuacion,accion_clave}',
        a #>> '{actuacion,actor_ref}',
        a #>> '{actuacion,unidad_ref}',
        a #>> '{actuacion,recibo_ref}',
        a #>> '{actuacion,fase_destino}',
        a #>> '{actuacion,estado_destino}',
        (a #>> '{actuacion,realizada_en}')::timestamptz,
        v_huella_actuacion, v_confirmacion_ref);
    SELECT secuencia_auditoria, cabeza_auditoria_sha256,
           secuencia_outbox, cabeza_outbox_sha256
      INTO STRICT v_secuencia_auditoria, v_anterior_auditoria,
                  v_secuencia_outbox, v_anterior_outbox
     FROM vec_contratacion_temporal.control_cadenas_alta
     WHERE control_id
     FOR UPDATE;
    IF v_secuencia_auditoria >= 9007199254740991::numeric
       OR v_secuencia_outbox >= 9007199254740991::numeric THEN
        RAISE EXCEPTION USING
            ERRCODE = '22003',
            MESSAGE = 'límite de secuencia alcanzado';
    END IF;
    v_secuencia_auditoria := v_secuencia_auditoria + 1;
    v_secuencia_outbox := v_secuencia_outbox + 1;
    v_huella_auditoria := pg_catalog.encode(pg_catalog.sha256(
        vec_contratacion_temporal.encuadrar_texto_v1(
            v_secuencia_auditoria::text) ||
        vec_contratacion_temporal.encuadrar_texto_v1(
            v_anterior_auditoria) ||
        vec_contratacion_temporal.encuadrar_texto_v1(v_auditoria_ref) ||
        vec_contratacion_temporal.encuadrar_texto_v1(
            a ->> 'expediente_ref') ||
        vec_contratacion_temporal.encuadrar_texto_v1(
            v_consumo.decision_ref) ||
        vec_contratacion_temporal.encuadrar_texto_v1(
            v_consumo.consumo_huella_sha256) ||
        vec_contratacion_temporal.encuadrar_texto_v1(v_huella_alta)), 'hex');
    INSERT INTO vec_contratacion_temporal.auditoria_alta (
        auditoria_ref, secuencia, expediente_ref, decision_ref,
        consumo_huella_sha256, anterior_sha256, huella_sha256,
        registrada_en, confirmacion_ref) VALUES (
        v_auditoria_ref, v_secuencia_auditoria,
        a ->> 'expediente_ref', v_consumo.decision_ref,
        v_consumo.consumo_huella_sha256, v_anterior_auditoria,
        v_huella_auditoria, v_ahora, v_confirmacion_ref);
    v_payload_outbox := pg_catalog.convert_to(
        '{"esquema":"vec.contratacion-temporal.evento-expediente-registrado.v1"' ||
        ',"evento_ref":' ||
          vec_contratacion_temporal.texto_json_go_v1(v_evento_ref) ||
        ',"expediente_ref":' ||
          vec_contratacion_temporal.texto_json_go_v1(
              a ->> 'expediente_ref') ||
        ',"version":1,"ocurrido_en":' ||
          vec_contratacion_temporal.texto_json_go_v1(
              vec_contratacion_temporal.instante_utc_v1(v_ahora)) || '}',
        'UTF8');
    v_huella_payload := pg_catalog.encode(
        pg_catalog.sha256(v_payload_outbox), 'hex');
    v_huella_outbox := pg_catalog.encode(pg_catalog.sha256(
        vec_contratacion_temporal.encuadrar_texto_v1(
            v_secuencia_outbox::text) ||
        vec_contratacion_temporal.encuadrar_texto_v1(
            v_anterior_outbox) ||
        vec_contratacion_temporal.encuadrar_texto_v1(v_evento_ref) ||
        vec_contratacion_temporal.encuadrar_texto_v1(v_huella_payload)), 'hex');
    INSERT INTO vec_contratacion_temporal.outbox_alta (
        evento_ref, secuencia, expediente_ref, tipo_evento,
        payload_canonico, payload_huella_sha256, anterior_sha256,
        huella_sha256, registrada_en, confirmacion_ref) VALUES (
        v_evento_ref, v_secuencia_outbox, a ->> 'expediente_ref',
        'contratacion_temporal.expediente.registrado.v1',
        v_payload_outbox, v_huella_payload, v_anterior_outbox,
        v_huella_outbox, v_ahora, v_confirmacion_ref);
    UPDATE vec_contratacion_temporal.control_cadenas_alta
       SET secuencia_auditoria = v_secuencia_auditoria,
           cabeza_auditoria_sha256 = v_huella_auditoria,
           secuencia_outbox = v_secuencia_outbox,
           cabeza_outbox_sha256 = v_huella_outbox,
           actualizada_en = v_ahora
     WHERE control_id;
    SELECT revision INTO STRICT v_revision
      FROM vec_contratacion_temporal.reserva_alta_actual
     WHERE ambito_hmac = v_raiz
     FOR UPDATE;
    v_revision := v_revision + 1;
    INSERT INTO vec_contratacion_temporal.reserva_alta_version (
        ambito_hmac, revision, estado, version_expediente,
        auditoria_ref, evento_ref, confirmada_en, registrada_en,
        confirmacion_ref) VALUES (
        v_raiz, v_revision, 'confirmada', 1,
        v_auditoria_ref, v_evento_ref, v_ahora, v_ahora,
        v_confirmacion_ref);
    UPDATE vec_contratacion_temporal.reserva_alta_actual
       SET revision = v_revision
     WHERE ambito_hmac = v_raiz;
    v_recibo_huella := pg_catalog.encode(pg_catalog.sha256(
        vec_contratacion_temporal.encuadrar_texto_v1(
            a ->> 'expediente_ref') ||
        vec_contratacion_temporal.encuadrar_texto_v1(
            a ->> 'numero_visible') ||
        vec_contratacion_temporal.encuadrar_texto_v1('1') ||
        vec_contratacion_temporal.encuadrar_texto_v1(a ->> 'recibo_ref') ||
        vec_contratacion_temporal.encuadrar_texto_v1(v_auditoria_ref) ||
        vec_contratacion_temporal.encuadrar_texto_v1(v_evento_ref) ||
        vec_contratacion_temporal.encuadrar_texto_v1(
            vec_contratacion_temporal.instante_utc_v1(v_ahora))), 'hex');
    v_huella_agregado :=
      vec_contratacion_temporal.huella_prueba_agregado_alta_v1(
        VARIADIC ARRAY[
          'vec.contratacion-temporal.confirmacion-agregado-alta.v1',
          v_confirmacion_ref, v_raiz, v_revision::text,
          a ->> 'reserva_ref', a ->> 'expediente_ref',
          a ->> 'numero_visible', a ->> 'recibo_ref',
          v_consumo.decision_ref, v_consumo.efecto_ref,
          v_consumo.huella_efecto_sha256,
          v_consumo.consumo_huella_sha256, '1', v_huella_alta, '1',
          v_huella_actuacion, v_auditoria_ref,
          v_secuencia_auditoria::text, v_anterior_auditoria,
          v_huella_auditoria, v_evento_ref, v_secuencia_outbox::text,
          v_huella_payload, v_anterior_outbox, v_huella_outbox,
          vec_contratacion_temporal.instante_utc_v1(v_ahora),
          v_recibo_huella
        ]
      );
    INSERT INTO vec_contratacion_temporal.confirmacion_agregado_alta (
        confirmacion_ref, agregado_huella_sha256, ambito_hmac,
        reserva_revision, reserva_ref, expediente_ref, numero_visible,
        recibo_ref, decision_ref, efecto_ref, huella_efecto_sha256,
        consumo_huella_sha256, version_expediente, huella_alta_sha256,
        actuacion_secuencia, actuacion_huella_sha256, auditoria_ref,
        auditoria_secuencia, auditoria_anterior_sha256,
        auditoria_huella_sha256, evento_ref, outbox_secuencia,
        payload_huella_sha256, outbox_anterior_sha256,
        outbox_huella_sha256, confirmada_en, recibo_huella_sha256,
        creada_en
    ) VALUES (
        v_confirmacion_ref, v_huella_agregado, v_raiz, v_revision,
        a ->> 'reserva_ref', a ->> 'expediente_ref',
        a ->> 'numero_visible', a ->> 'recibo_ref',
        v_consumo.decision_ref, v_consumo.efecto_ref,
        v_consumo.huella_efecto_sha256,
        v_consumo.consumo_huella_sha256, 1, v_huella_alta, 1,
        v_huella_actuacion, v_auditoria_ref, v_secuencia_auditoria,
        v_anterior_auditoria, v_huella_auditoria, v_evento_ref,
        v_secuencia_outbox, v_huella_payload, v_anterior_outbox,
        v_huella_outbox, v_ahora, v_recibo_huella, v_ahora
    );
    RETURN QUERY
    SELECT *
      FROM vec_contratacion_temporal.reconciliar_agregado_alta_v1(
          p_alta_canonica, v_activo_ambito, v_activo_huella,
          v_consumo.decision_ref, v_consumo.efecto_ref,
          v_consumo.huella_efecto_sha256,
          v_consumo.consumo_huella_sha256
      );
EXCEPTION
    WHEN invalid_text_representation OR datetime_field_overflow
      OR numeric_value_out_of_range THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'entrada de alta inválida';
END
$function$
;

CREATE OR REPLACE FUNCTION vec_contratacion_temporal.expediente_analisis_valido_v2(e jsonb, p_exige_analisis boolean)
 RETURNS boolean
 LANGUAGE plpgsql
 IMMUTABLE
 SET search_path TO 'pg_catalog'
AS $function$
DECLARE
    s jsonb := e -> 'solicitud';
    rc jsonb := e #> '{solicitud,rc}';
    v_claves text[] := ARRAY[
      'actuaciones', 'actualizado_en', 'creado_en', 'estado_actual',
      'fase_actual', 'flujo', 'numero_visible', 'organizacion_ref',
      'referencia', 'solicitud', 'version'
    ]::text[];
    v_claves_solicitud text[] := ARRAY[
      'categoria_ref', 'centro_ref', 'contacto_ref', 'detalle',
      'documentos_adjuntos', 'grupo_subgrupo', 'motivo_clave',
      'periodo', 'rc'
    ]::text[];
    v_claves_rc text[];
    v_actuacion jsonb;
    v_documento jsonb;
    v_necesidad boolean;
    v_solicitud_civil jsonb;
    v_inicio_text text;
    v_fin_text text;
BEGIN
    IF vec_contratacion_temporal.circuito_agregado_valido_ct164(e) IS NOT TRUE THEN RETURN false; END IF;
    IF e ? 'circuito' THEN
        SELECT array_agg(k ORDER BY k) INTO v_claves FROM unnest(array_append(v_claves,'circuito')) k;
    END IF;
    IF p_exige_analisis IS NULL
       OR pg_catalog.jsonb_typeof(e) <> 'object'
       OR pg_catalog.jsonb_exists(e, 'via_cobertura')
       OR pg_catalog.jsonb_exists(e, 'asignacion') THEN
        RETURN false;
    END IF;
    IF p_exige_analisis THEN
        v_claves := pg_catalog.array_append(v_claves, 'analisis');
        SELECT pg_catalog.array_agg(x ORDER BY x)
          INTO v_claves
          FROM pg_catalog.unnest(v_claves) AS c(x);
    ELSIF pg_catalog.jsonb_exists(e, 'analisis') THEN
        RETURN false;
    END IF;
    v_necesidad := s ? 'necesidad';
    IF v_necesidad THEN
        v_claves_solicitud := pg_catalog.array_append(v_claves_solicitud,'necesidad');
        -- O3 conserva la proyección UTC del dominio. CT193 gobierna la fecha
        -- civil: se retroconvierte una copia estricta, nunca el agregado/HMAC.
        v_inicio_text := s #>> '{periodo,inicio}';
        IF v_inicio_text !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T00:00:00Z$'
           OR v_inicio_text IS DISTINCT FROM s #>> '{necesidad,periodo,inicio}' THEN
            RETURN false;
        END IF;
        v_solicitud_civil := pg_catalog.jsonb_set(s,'{periodo,inicio}',
            pg_catalog.to_jsonb(pg_catalog.left(v_inicio_text,10)),false);
        v_solicitud_civil := pg_catalog.jsonb_set(v_solicitud_civil,
            '{necesidad,periodo,inicio}',
            pg_catalog.to_jsonb(pg_catalog.left(v_inicio_text,10)),false);
        v_fin_text := s #>> '{periodo,fin}';
        IF v_fin_text IS NOT NULL THEN
            IF v_fin_text !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T00:00:00Z$'
               OR v_fin_text IS DISTINCT FROM s #>> '{necesidad,periodo,fin}' THEN
                RETURN false;
            END IF;
            v_solicitud_civil := pg_catalog.jsonb_set(v_solicitud_civil,
                '{periodo,fin}',pg_catalog.to_jsonb(pg_catalog.left(v_fin_text,10)),false);
            v_solicitud_civil := pg_catalog.jsonb_set(v_solicitud_civil,
                '{necesidad,periodo,fin}',
                pg_catalog.to_jsonb(pg_catalog.left(v_fin_text,10)),false);
        ELSIF s #> '{periodo,fin}' IS DISTINCT FROM
              s #> '{necesidad,periodo,fin}' THEN
            RETURN false;
        END IF;
        IF vec_contratacion_temporal.necesidad_alta_valida_v3(v_solicitud_civil) IS NOT TRUE THEN
            RETURN false;
        END IF;
    END IF;
    -- La variante sellada admite solo el numero superior. La ausencia de
    -- ese numero conserva literalmente la solicitud historica.
    IF s ? 'numero_personas'
       AND NOT (s ?| ARRAY['jornada_minutos','puesto_solicitado']) THEN
        IF NOT v_necesidad
           OR vec_contratacion_temporal.datos_puesto_peticion_validos_ct200(
                v_solicitud_civil) IS NOT TRUE THEN
            RETURN false;
        END IF;
        v_claves_solicitud := pg_catalog.array_append(
            v_claves_solicitud,'numero_personas');
    -- El trio anterior sigue completo y con la misma validacion.
    ELSIF s ?| ARRAY['jornada_minutos','numero_personas','puesto_solicitado'] THEN
        IF NOT (s ?& ARRAY['jornada_minutos','numero_personas','puesto_solicitado'])
           OR NOT (CASE WHEN pg_catalog.jsonb_typeof(s->'jornada_minutos')='number'
                         AND s->>'jornada_minutos' ~ '^[0-9]+$'
                    THEN (s->>'jornada_minutos')::numeric BETWEEN 1 AND 10080
                    ELSE false END)
           OR NOT (CASE WHEN pg_catalog.jsonb_typeof(s->'numero_personas')='number'
                         AND s->>'numero_personas' ~ '^[0-9]+$'
                    THEN (s->>'numero_personas')::numeric BETWEEN 1 AND 4294967295
                    ELSE false END)
           OR pg_catalog.jsonb_typeof(s->'puesto_solicitado')<>'string'
           OR pg_catalog.length(s->>'puesto_solicitado') NOT BETWEEN 1 AND 160
           OR pg_catalog.btrim(s->>'puesto_solicitado') IS DISTINCT FROM s->>'puesto_solicitado'
           OR (s->>'puesto_solicitado') ~ '[[:cntrl:]]' THEN
            RETURN false;
        END IF;
        v_claves_solicitud := v_claves_solicitud ||
            ARRAY['jornada_minutos','numero_personas','puesto_solicitado'];
    END IF;
    IF pg_catalog.jsonb_exists(s, 'observaciones') THEN
        v_claves_solicitud :=
            pg_catalog.array_append(v_claves_solicitud, 'observaciones');
        SELECT pg_catalog.array_agg(x ORDER BY x)
          INTO v_claves_solicitud
          FROM pg_catalog.unnest(v_claves_solicitud) AS c(x);
    END IF;
    SELECT pg_catalog.array_agg(x ORDER BY x) INTO v_claves_solicitud
    FROM pg_catalog.unnest(v_claves_solicitud) AS c(x);
    v_claves_rc := CASE WHEN rc ->> 'existe' = 'true' THEN ARRAY[
      'documento_ref', 'existe', 'fecha', 'importe', 'numero'
    ]::text[] ELSE ARRAY['existe', 'fecha', 'importe']::text[] END;
    IF NOT vec_contratacion_temporal.claves_json_exactas_v1(e, v_claves)
       OR NOT vec_contratacion_temporal.claves_json_exactas_v1(
           e -> 'flujo',
           ARRAY['definicion_ref', 'huella_sha256', 'version']::text[]
       )
       OR NOT vec_contratacion_temporal.claves_json_exactas_v1(
           s, v_claves_solicitud
       )
       OR (NOT v_necesidad AND NOT vec_contratacion_temporal.periodo_previsto_analisis_valido_v1(s -> 'periodo'))
       OR NOT vec_contratacion_temporal.claves_json_exactas_v1(
           rc, v_claves_rc
       )
       OR NOT vec_contratacion_temporal.numero_entero_json_canonico_v2(
           e -> 'version', 1, 9007199254740991::numeric
       )
       OR NOT vec_contratacion_temporal.numero_entero_json_canonico_v2(
           e #> '{flujo,version}', 1, 9007199254740991::numeric
       )
       OR NOT vec_contratacion_temporal.instante_utc_json_canonico_v2(
           e -> 'creado_en', false
       )
       OR NOT vec_contratacion_temporal.instante_utc_json_canonico_v2(
           e -> 'actualizado_en', false
       )
       OR NOT vec_contratacion_temporal.instante_utc_json_canonico_v2(
           s #> '{periodo,inicio}', true
       )
       OR (s #> '{periodo,fin}' IS NOT NULL
           AND NOT vec_contratacion_temporal.instante_utc_json_canonico_v2(
             s #> '{periodo,fin}', true
           ))
       OR pg_catalog.jsonb_typeof(e -> 'referencia') <> 'string'
       OR pg_catalog.jsonb_typeof(e -> 'organizacion_ref') <> 'string'
       OR pg_catalog.jsonb_typeof(e -> 'numero_visible') <> 'string'
       OR pg_catalog.jsonb_typeof(e -> 'fase_actual') <> 'string'
       OR pg_catalog.jsonb_typeof(e -> 'estado_actual') <> 'string'
       OR pg_catalog.jsonb_typeof(e #> '{flujo,definicion_ref}') <> 'string'
       OR pg_catalog.jsonb_typeof(e #> '{flujo,huella_sha256}') <> 'string'
       OR pg_catalog.jsonb_typeof(s -> 'centro_ref') <> 'string'
       OR pg_catalog.jsonb_typeof(s -> 'contacto_ref') <> 'string'
       OR pg_catalog.jsonb_typeof(s -> 'categoria_ref') <> 'string'
       OR pg_catalog.jsonb_typeof(s -> 'grupo_subgrupo') <> 'string'
       OR pg_catalog.jsonb_typeof(s -> 'motivo_clave') <> 'string'
       OR pg_catalog.jsonb_typeof(s -> 'detalle') <> 'string'
       OR pg_catalog.jsonb_typeof(rc -> 'existe') <> 'boolean'
       OR pg_catalog.jsonb_typeof(e -> 'actuaciones') <> 'array'
       OR pg_catalog.jsonb_typeof(s -> 'documentos_adjuntos')
            NOT IN ('array', 'null')
       OR (e ->> 'actualizado_en')::timestamptz <
          (e ->> 'creado_en')::timestamptz
       OR (s #>> '{periodo,fin}')::timestamptz <
          (s #>> '{periodo,inicio}')::timestamptz THEN
        RETURN false;
    END IF;
    IF p_exige_analisis
       AND vec_contratacion_temporal.huella_analisis_derivado_v2(
               e -> 'analisis'
           ) IS NULL THEN
        RETURN false;
    END IF;
    IF rc ->> 'existe' = 'true' THEN
        IF NOT vec_contratacion_temporal.instante_utc_json_canonico_v2(
               rc -> 'fecha', true
           )
           OR NOT vec_contratacion_temporal.claves_json_exactas_v1(
               rc -> 'importe', ARRAY['centimos', 'moneda']::text[]
           )
           OR NOT vec_contratacion_temporal.numero_entero_json_canonico_v2(
               rc #> '{importe,centimos}', 1,
               9223372036854775807::numeric
           )
           OR pg_catalog.jsonb_typeof(rc -> 'numero') <> 'string'
           OR pg_catalog.jsonb_typeof(rc -> 'documento_ref') <> 'string'
           OR pg_catalog.jsonb_typeof(rc #> '{importe,moneda}') <> 'string' THEN
            RETURN false;
        END IF;
    ELSE
        IF rc ->> 'fecha' <> '0001-01-01T00:00:00Z'
           OR NOT vec_contratacion_temporal.claves_json_exactas_v1(
               rc -> 'importe', ARRAY['centimos', 'moneda']::text[]
           )
           OR NOT vec_contratacion_temporal.numero_entero_json_canonico_v2(
               rc #> '{importe,centimos}', 0, 0
           )
           OR rc #>> '{importe,moneda}' <> '' THEN
            RETURN false;
        END IF;
    END IF;
    FOR v_documento IN
        SELECT d.v
          FROM pg_catalog.jsonb_array_elements(
              CASE
                WHEN s -> 'documentos_adjuntos' = 'null'::jsonb
                THEN '[]'::jsonb
                ELSE s -> 'documentos_adjuntos'
              END
          ) AS d(v)
    LOOP
        IF pg_catalog.jsonb_typeof(v_documento) <> 'string' THEN
            RETURN false;
        END IF;
    END LOOP;
    FOR v_actuacion IN
        SELECT a.v
          FROM pg_catalog.jsonb_array_elements(e -> 'actuaciones') AS a(v)
    LOOP
        IF vec_contratacion_temporal.actuacion_analisis_valida_v2(
               v_actuacion
           ) IS NOT TRUE THEN
            RETURN false;
        END IF;
    END LOOP;
    RETURN true;
EXCEPTION
    WHEN data_exception OR datetime_field_overflow
      OR invalid_text_representation OR numeric_value_out_of_range THEN
        RETURN false;
END
$function$
;

DO $post$ DECLARE r record; v_oids text:=''; v_oid text; BEGIN
 FOR r IN SELECT * FROM (VALUES
  ('vec_contratacion_temporal.datos_puesto_peticion_validos_ct200(jsonb)','2a76bbd78cf1c907d84d5def7c5a5cc3d2113b1985467ef1c7c9721f77e214a4','{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario}',ARRAY['search_path=pg_catalog, pg_temp']::text[],false,'i',true,'u',false),
  ('vec_contratacion_temporal.reconstruir_solicitud_efecto_v2(jsonb)','771ee8bd20d0a1d26567a2bb6dd169d597a1f1ce23ba3bc6fa67f4c7d876de5e','{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario}',ARRAY['search_path=pg_catalog']::text[],false,'i',true,'u',false),
  ('vec_contratacion_temporal.confirmar_alta_atestada_v1(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea)','e831914180bca9cd63c8178aaa03ab1c86b7935bd05838368764c045b7a48ef4','{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario}',ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']::text[],true,'v',false,'u',true),
  ('vec_contratacion_temporal.expediente_analisis_valido_v2(jsonb,boolean)','e8080b22a5201ede7f08ca1076e09870d00e69fe1ebaa01bce8bb14f03a17f95','{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario}',ARRAY['search_path=pg_catalog']::text[],false,'i',false,'u',false)
 ) AS x(firma,sha,acl,config,secdef,vol,strict,parallel,retset) LOOP
  SELECT p.oid::text INTO v_oid FROM pg_catalog.pg_proc p
   WHERE p.oid=pg_catalog.to_regprocedure(r.firma)
    AND p.proowner='vec_contratacion_temporal_propietario'::regrole
    AND p.prosecdef=r.secdef AND p.provolatile=r.vol
    AND p.proisstrict=r.strict AND p.proparallel=r.parallel
    AND p.proretset=r.retset
    AND p.proacl::text=r.acl AND p.proconfig=r.config
    AND pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
      pg_catalog.pg_get_functiondef(p.oid),'UTF8')),'hex')=r.sha;
  IF v_oid IS NULL THEN RAISE EXCEPTION 'CT239: postimagen incompatible %',r.firma
   USING ERRCODE='55000'; END IF;
  v_oids:=v_oids||'/'||v_oid; v_oid:=NULL;
 END LOOP;
 IF v_oids IS DISTINCT FROM pg_catalog.current_setting('vec.ct239.oids') THEN
  RAISE EXCEPTION 'CT239: OID alterado' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
