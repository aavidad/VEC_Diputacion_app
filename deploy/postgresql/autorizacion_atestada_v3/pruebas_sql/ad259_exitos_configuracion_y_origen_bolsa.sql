-- Solo copia PG18 sintética HZ21 con AD259 instalada. No ejecutar en principal.
\set ON_ERROR_STOP on
\timing on

-- La copia HZ21 arranca con latido AD207 caducado; se renueva sólo aquí.
UPDATE vec_autorizacion_atestada_v3.sellado_auditoria_v5
SET latido=pg_catalog.clock_timestamp();

CREATE TEMP TABLE ad259_muestra ON COMMIT PRESERVE ROWS AS
SELECT r.representacion_canonica AS contexto,
       pg_catalog.convert_to((d.vinculo)::text,'UTF8') AS vinculo_bytes,
       r.registro_contexto_ref, r.huella_sha256 AS contexto_sha256,
       r.manifiesto_procedencia_huella_sha256 AS procedencia_sha256,
       d.vinculo AS vinculo_json
FROM (
  SELECT pg_catalog.convert_from(decision_canonica,'UTF8')::jsonb
         -> 'vinculo_autenticacion_actor' AS vinculo
  FROM vec_autorizacion.decision_concedida_contexto_actor_v3
) d
JOIN vec_contexto_actor_v1.registros_contexto r
  ON r.registro_contexto_ref=d.vinculo ->> 'registro_contexto_ref'
JOIN vec_autorizacion.sesion_autenticacion_v1 b
  ON b.autenticacion_ref=d.vinculo ->> 'autenticacion_ref'
 AND b.sesion_ref=d.vinculo ->> 'sesion_ref'
JOIN vec_identidad_sesiones_v1.consumo_asercion c
  ON c.sesion_ref=b.sesion_ref AND c.autenticacion_ref=b.autenticacion_ref
JOIN vec_autorizacion.control_sesion_v1 s
  ON s.sesion_ref=c.sesion_ref
 AND s.control_sesion_ref=c.control_sesion_ref
 AND s.revision=c.control_sesion_revision
WHERE d.vinculo ->> 'superficie'='interna_corporativa'
  AND d.vinculo ->> 'autoridad_efectiva'='autoridad_maestra_acreditada'
  AND (d.vinculo ->> 'control_sesion_revision')::numeric=c.control_sesion_revision
  AND d.vinculo ->> 'control_sesion_ref'=c.control_sesion_ref
LIMIT 1;

DO $pre$
BEGIN
  IF (SELECT count(*) FROM pg_temp.ad259_muestra) <> 1 THEN
    RAISE EXCEPTION 'AD259 sin evidencia sintética nominal';
  END IF;
END $pre$;
SELECT login_nombre::text AS registrador_login
FROM vec_autorizacion_atestada_v3.configuracion_runtime_intentos
WHERE proceso='vec-rrhh' AND canal='interna_corporativa'
\gset
GRANT SELECT ON pg_temp.ad259_muestra TO :"registrador_login";

CREATE TEMP TABLE ad259_antes ON COMMIT PRESERVE ROWS AS
SELECT count(*) AS filas, max(secuencia) AS secuencia
FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3;

-- El registrador ve una sesión de LOGIN real y rol GUC none, como en la app.
SET SESSION AUTHORIZATION :"registrador_login";
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TIME ZONE 'UTC';
DO $ensayo$
DECLARE
  m record;
  o jsonb;
  clave text := 'intento_' || pg_catalog.md5('ad259-' ||
    pg_catalog.clock_timestamp()::text || pg_catalog.random()::text);
  primera record;
  replay record;
  accion text;
  finalidad text;
  recurso text;
  resultado text;
  errores integer := 0;
BEGIN
  SELECT * INTO STRICT m FROM pg_temp.ad259_muestra;
  o := pg_catalog.jsonb_build_object(
    'intento_ref',clave,
    'registro_contexto_ref',m.registro_contexto_ref,
    'contexto_sha256',m.contexto_sha256,
    'procedencia_sha256',m.procedencia_sha256,
    'autenticacion_ref',m.vinculo_json ->> 'autenticacion_ref',
    'sesion_ref',m.vinculo_json ->> 'sesion_ref',
    'autenticacion_sha256',m.vinculo_json ->> 'autenticacion_huella_sha256',
    'accion','lectura_bolsas_rrhh_consultar',
    'modulo_id','bolsa','recurso_ref','bolsa:01',
    'finalidad_ref','gestion_bolsa_rrhh',
    'resultado','exito','motivo_ref','motivos_auditoria:1:lectura_acreditada',
    'proceso','vec-rrhh','canal','interna_corporativa',
    'correlacion_ref','correlacion_' || pg_catalog.substr(clave,9));
  SELECT * INTO STRICT primera FROM vec_autorizacion_atestada_v3.registrar_intento_nominal_v1(
    m.contexto,m.vinculo_bytes,o);
  SELECT * INTO STRICT replay FROM vec_autorizacion_atestada_v3.registrar_intento_nominal_v1(
    m.contexto,m.vinculo_bytes,o);
  IF primera IS DISTINCT FROM replay THEN
    RAISE EXCEPTION 'AD259 replay no conserva acuse';
  END IF;
  BEGIN
    PERFORM vec_autorizacion_atestada_v3.registrar_intento_nominal_v1(
      m.contexto,m.vinculo_bytes,o || '{"recurso_ref":"bolsa:02"}'::jsonb);
    RAISE EXCEPTION 'AD259 clave incompatible admitida';
  EXCEPTION WHEN unique_violation THEN errores := errores + 1;
  END;

  -- Cada éxito exacto produce un asiento; I conserva sus dos acciones.
  FOR accion,finalidad,recurso IN SELECT * FROM (VALUES
    ('lectura_bolsas_rrhh_descargar_recibo','gestion_bolsa_rrhh','bolsa:02'),
    ('bolsa.configuracion.consultar','consulta_configuracion_bolsa_rrhh',
       'bolsa:perfil:' || pg_catalog.repeat('a',64)),
    ('bolsa.origen_ct.consultar','gestionar_contratacion_temporal','bolsa:origen:01')
  ) t(accion,finalidad,recurso) LOOP
    PERFORM vec_autorizacion_atestada_v3.registrar_intento_nominal_v1(
      m.contexto,m.vinculo_bytes,o || pg_catalog.jsonb_build_object(
        'intento_ref','intento_' || pg_catalog.md5(clave || accion),
        'accion',accion,'finalidad_ref',finalidad,'recurso_ref',recurso));
  END LOOP;

  -- Cruces de pareja, recurso de perfil incorrecto y extensión ajena.
  FOR accion,finalidad,recurso IN SELECT * FROM (VALUES
    ('lectura_bolsas_rrhh_consultar','consulta_configuracion_bolsa_rrhh','bolsa:01'),
    ('bolsa.configuracion.consultar','gestionar_contratacion_temporal',
       'bolsa:perfil:' || pg_catalog.repeat('a',64)),
    ('bolsa.origen_ct.consultar','gestion_bolsa_rrhh','bolsa:origen:01'),
    ('bolsa.configuracion.consultar','consulta_configuracion_bolsa_rrhh','bolsa:01'),
    ('bolsa.configuracion.exportar','consulta_configuracion_bolsa_rrhh','bolsa:01')
  ) t(accion,finalidad,recurso) LOOP
    BEGIN
      PERFORM vec_autorizacion_atestada_v3.registrar_intento_nominal_v1(
        m.contexto,m.vinculo_bytes,o || pg_catalog.jsonb_build_object(
          'intento_ref','intento_' || pg_catalog.md5(clave || accion || finalidad || recurso),
          'accion',accion,'finalidad_ref',finalidad,'recurso_ref',recurso));
      RAISE EXCEPTION 'AD259 cruce/recurso ajeno admitido';
    EXCEPTION WHEN invalid_parameter_value THEN errores := errores+1;
    END;
  END LOOP;
  FOR accion,finalidad,recurso IN SELECT * FROM (VALUES
    ('bolsa.configuracion.consultar','consulta_configuracion_bolsa_rrhh','persona:01'),
    ('bolsa.origen_ct.consultar','gestionar_contratacion_temporal','persona:01')
  ) t(accion,finalidad,recurso) LOOP
    BEGIN
      PERFORM vec_autorizacion_atestada_v3.registrar_intento_nominal_v1(
        m.contexto,m.vinculo_bytes,o || pg_catalog.jsonb_build_object(
          'intento_ref','intento_' || pg_catalog.md5(clave || accion || recurso),
          'accion',accion,'finalidad_ref',finalidad,'recurso_ref',recurso));
      RAISE EXCEPTION 'AD259 recurso ajeno admitido';
    EXCEPTION WHEN invalid_parameter_value THEN errores := errores+1;
    END;
  END LOOP;
  IF errores <> 8 THEN RAISE EXCEPTION 'AD259 faltan negativos: %',errores; END IF;

  -- Los resultados fallidos conservan el contrato anterior de AD169.
  FOR resultado IN SELECT * FROM (VALUES ('denegado'),('error')) t(resultado) LOOP
    PERFORM vec_autorizacion_atestada_v3.registrar_intento_nominal_v1(
      m.contexto,m.vinculo_bytes,o || pg_catalog.jsonb_build_object(
        'intento_ref','intento_' || pg_catalog.md5(clave || resultado),
        'accion','bolsa.consulta.no_admitida','finalidad_ref','otra_finalidad',
        'resultado',resultado,'motivo_ref','motivos_auditoria:1:fallo_observado'));
  END LOOP;
END $ensayo$;
COMMIT;
RESET SESSION AUTHORIZATION;

DO $post$
DECLARE antes record; despues record;
BEGIN
  SELECT * INTO STRICT antes FROM pg_temp.ad259_antes;
  SELECT count(*) AS filas,max(secuencia) AS secuencia INTO STRICT despues
  FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3;
  -- Las pruebas negativas revierten subtransacciones; la reserva AD207 puede
  -- dejar huecos de secuencia, pero ningún asiento previo cambia ni se duplica.
  IF despues.filas <> antes.filas+6 OR despues.secuencia < antes.secuencia+6 THEN
    RAISE EXCEPTION 'AD259 alteró cardinalidad o progreso de secuencia';
  END IF;
  IF (SELECT count(*) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3
      WHERE secuencia>antes.secuencia AND tipo_registro='intento_nominal'
        AND actor_ref IS NOT NULL AND perfil_activo_ref IS NOT NULL
        AND sesion_ref IS NOT NULL AND contexto_sha256 IS NOT NULL) <> 6 THEN
    RAISE EXCEPTION 'AD259 no conserva identidad nominal';
  END IF;
  RAISE NOTICE 'AD259-ENSAYO-OK: seis asientos, cuatro éxitos exactos, replay y negativos';
END $post$;

EXPLAIN (COSTS OFF)
SELECT auditoria_ref FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3
WHERE intento_ref='intento_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa';
