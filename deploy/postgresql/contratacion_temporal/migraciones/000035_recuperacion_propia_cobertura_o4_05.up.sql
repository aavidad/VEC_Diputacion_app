-- O4-05: recuperación propia terminal, sin reservar ni repetir el efecto.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_catalog.pg_advisory_xact_lock(
  pg_catalog.hashtextextended(
    'vec_contratacion_temporal:o4_04:migraciones',0));
SELECT control FROM vec_contratacion_temporal.control_migracion_cobertura_o4
 WHERE control AND version_esquema=14 FOR UPDATE;
DO $prevalidacion$
BEGIN
  IF NOT EXISTS(
    SELECT 1 FROM vec_contratacion_temporal
      .control_migracion_cobertura_o4
     WHERE control AND version_esquema=14
  ) OR pg_catalog.to_regprocedure(
    'vec_contratacion_temporal.recuperar_resultado_propio_decision_cobertura_o405_v1(jsonb)'
  ) IS NOT NULL OR pg_catalog.to_regprocedure(
    'vec_contratacion_temporal.o404e_leer_terminal_interno_v1(text,text)'
  ) IS NULL OR NOT EXISTS(
    SELECT 1 FROM pg_catalog.pg_roles
     WHERE rolname=
       'vec_contratacion_temporal_lector_resultado_cobertura'
       AND NOT rolcanlogin AND NOT rolsuper AND NOT rolbypassrls
  ) THEN
    RAISE EXCEPTION USING ERRCODE='55000',
      MESSAGE='estado incompatible para recuperación propia O4-05';
  END IF;
END
$prevalidacion$;

CREATE FUNCTION
vec_contratacion_temporal
  .recuperar_resultado_propio_decision_cobertura_o405_v1(
    p_consulta jsonb
)
RETURNS TABLE(resultado_json jsonb)
LANGUAGE plpgsql
STABLE
SECURITY DEFINER
SET search_path=pg_catalog
SET row_security='on'
SET timezone='UTC'
SET lock_timeout='2s'
AS $funcion$
DECLARE
  v_alias text;
  v_ambitos text[]:=ARRAY[]::text[];
  v_generacion integer;
  v_anterior integer:=1000000000;
  v_raices text[];
  v_base record;
  v_recibo jsonb;
  v_observada timestamptz(6):=pg_catalog.date_trunc(
    'microseconds',pg_catalog.clock_timestamp());
BEGIN
  IF current_user<>'vec_contratacion_temporal_propietario'
     OR session_user=current_user
     OR NOT EXISTS(
       SELECT 1 FROM pg_catalog.pg_roles
        WHERE rolname=session_user AND rolcanlogin AND NOT rolsuper
          AND NOT rolcreatedb AND NOT rolcreaterole
          AND NOT rolreplication AND NOT rolbypassrls
     )
     OR (SELECT pg_catalog.count(*)
           FROM pg_catalog.pg_auth_members m
           JOIN pg_catalog.pg_roles u ON u.oid=m.member
          WHERE u.rolname=session_user)<>1
     OR NOT EXISTS(
       SELECT 1 FROM pg_catalog.pg_auth_members m
       JOIN pg_catalog.pg_roles u ON u.oid=m.member
       JOIN pg_catalog.pg_roles r ON r.oid=m.roleid
        WHERE u.rolname=session_user
          AND r.rolname=
            'vec_contratacion_temporal_lector_resultado_cobertura'
          AND NOT m.admin_option AND m.inherit_option
          AND NOT m.set_option
     )
     OR pg_catalog.pg_has_role(
       session_user,'vec_contratacion_temporal_propietario','MEMBER')
     OR pg_catalog.pg_has_role(
       session_user,'vec_contratacion_temporal_migrador','MEMBER')
     OR pg_catalog.pg_has_role(
       session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
     OR pg_catalog.pg_has_role(
       session_user,'vec_contratacion_temporal_gobernador','MEMBER')
     OR pg_catalog.pg_has_role(
       session_user,
       'vec_contratacion_temporal_confirmador_cobertura','MEMBER')
     OR vec_contratacion_temporal.gobi_o404b_entorno_valido(true)
        IS NOT TRUE OR pg_catalog.pg_is_in_recovery()
     OR p_consulta IS NULL
     OR pg_catalog.jsonb_typeof(p_consulta)<>'object'
     OR pg_catalog.pg_column_size(p_consulta)>8192
     OR NOT vec_contratacion_temporal.o404e_claves_exactas_v1(
       p_consulta,ARRAY[
         'ambitos_idempotencia_hmac','esquema','expediente_ref',
         'organizacion_ref']::text[])
     OR p_consulta->>'esquema'<>
       'vec.contratacion-temporal.consulta-recuperacion-propia-decision-cobertura.o4-05.v1'
     OR p_consulta->>'organizacion_ref'!~
       '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
     OR p_consulta->>'expediente_ref'!~
       '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
     OR pg_catalog.jsonb_typeof(
       p_consulta->'ambitos_idempotencia_hmac')<>'array' THEN
    RAISE EXCEPTION USING ERRCODE='42501',
      MESSAGE='recuperación propia O4-05 no autorizada';
  END IF;
  IF pg_catalog.jsonb_array_length(
       p_consulta->'ambitos_idempotencia_hmac') NOT BETWEEN 1 AND 4 THEN
    RAISE EXCEPTION USING ERRCODE='42501',
      MESSAGE='recuperación propia O4-05 no autorizada';
  END IF;
  FOR v_alias IN
    SELECT valor#>>'{}'
      FROM pg_catalog.jsonb_array_elements(
        p_consulta->'ambitos_idempotencia_hmac'
      ) WITH ORDINALITY AS elemento(valor,orden)
     ORDER BY orden
  LOOP
    IF v_alias IS NULL OR v_alias!~(
         '^hmac-sha256:vec[.]contratacion-temporal[.]'
         ||'cobertura-decision[.]ambito/v[1-9][0-9]{0,8}:'
         ||'[a-f0-9]{64}$')
       OR pg_catalog.right(v_alias,64)=pg_catalog.repeat('0',64) THEN
      RAISE EXCEPTION USING ERRCODE='42501',
        MESSAGE='recuperación propia O4-05 no autorizada';
    END IF;
    v_generacion:=substring(
      v_alias FROM '/v([1-9][0-9]{0,8}):')::integer;
    IF v_generacion>=v_anterior OR v_alias=ANY(v_ambitos) THEN
      RAISE EXCEPTION USING ERRCODE='42501',
        MESSAGE='recuperación propia O4-05 no autorizada';
    END IF;
    v_ambitos:=pg_catalog.array_append(v_ambitos,v_alias);
    v_anterior:=v_generacion;
  END LOOP;

  PERFORM pg_catalog.pg_advisory_xact_lock_shared(
    pg_catalog.hashtextextended(
      'vec_contratacion_temporal:o4_04:migraciones',0));
  PERFORM 1 FROM vec_contratacion_temporal.control_migracion_cobertura_o4
   WHERE control AND version_esquema=15;
  IF NOT FOUND THEN
    RAISE EXCEPTION USING ERRCODE='55000',
      MESSAGE='barrera de recuperación O4-05 no disponible';
  END IF;
  SELECT pg_catalog.array_agg(DISTINCT a.ambito_raiz_hmac)
    INTO v_raices
    FROM vec_contratacion_temporal
      .alias_operacion_decision_cobertura a
   WHERE a.alias_ambito_hmac=ANY(v_ambitos);
  IF coalesce(pg_catalog.cardinality(v_raices),0)=0 THEN
    RETURN QUERY SELECT pg_catalog.jsonb_build_object(
      'esquema',
       'vec.contratacion-temporal.resultado-recuperacion-propia-decision-cobertura.o4-05.v1',
      'estado','no_observable',
      'observada_en',
        vec_contratacion_temporal.texto_instante_utc_go_v2(
          v_observada::text));
    RETURN;
  ELSIF pg_catalog.cardinality(v_raices)<>1 THEN
    RAISE EXCEPTION USING ERRCODE='23505',
      MESSAGE='alias de recuperación propia O4-05 divergentes';
  END IF;

  SELECT b.organizacion_ref,b.expediente_ref,b.version_expediente,
         b.reserva_ref,b.recibo_ref,b.actuacion_ref,b.auditoria_ref,
         b.evento_ref,b.correlacion_vec_ref,b.decision_vec_ref,
         rv.estado,rv.huella_orden_sha256 AS huella_version,
         rv.observada_en,rv.ambito_raiz_hmac AS version_raiz,
         c.huella_orden_sha256,c.ambito_idempotencia_hmac,
         c.huella_semantica_hmac,c.revision_cercado,c.rama,
         t.ambito_raiz_hmac AS terminal_raiz
    INTO v_base
    FROM vec_contratacion_temporal
      .reserva_operacion_decision_cobertura b
    LEFT JOIN vec_contratacion_temporal
      .reserva_operacion_decision_cobertura_actual a
      USING(ambito_raiz_hmac)
    LEFT JOIN vec_contratacion_temporal
      .reserva_operacion_decision_cobertura_version rv
      ON rv.ambito_raiz_hmac=a.ambito_raiz_hmac
     AND rv.secuencia=a.secuencia
    LEFT JOIN vec_contratacion_temporal
      .confirmacion_operacion_decision_cobertura c
      ON c.ambito_raiz_hmac=b.ambito_raiz_hmac
    LEFT JOIN vec_contratacion_temporal
      .terminal_operacion_decision_cobertura t
      ON t.ambito_raiz_hmac=b.ambito_raiz_hmac
   WHERE b.ambito_raiz_hmac=v_raices[1];
  IF NOT FOUND OR v_base.version_raiz IS NULL THEN
    RAISE EXCEPTION USING ERRCODE='55000',
      MESSAGE='resultado terminal O4-05 no confiable';
  END IF;
  IF v_base.organizacion_ref IS DISTINCT FROM
        p_consulta->>'organizacion_ref'
     OR v_base.expediente_ref IS DISTINCT FROM
        p_consulta->>'expediente_ref' THEN
    RETURN QUERY SELECT pg_catalog.jsonb_build_object(
      'esquema',
       'vec.contratacion-temporal.resultado-recuperacion-propia-decision-cobertura.o4-05.v1',
      'estado','no_observable',
      'observada_en',
        vec_contratacion_temporal.texto_instante_utc_go_v2(
          v_observada::text));
    RETURN;
  END IF;
  IF v_base.estado='reservada'
     AND v_base.huella_version IS NULL
     AND v_base.huella_orden_sha256 IS NULL
     AND v_base.terminal_raiz IS NULL THEN
    RETURN QUERY SELECT pg_catalog.jsonb_build_object(
      'esquema',
       'vec.contratacion-temporal.resultado-recuperacion-propia-decision-cobertura.o4-05.v1',
      'estado','no_observable',
      'observada_en',
        vec_contratacion_temporal.texto_instante_utc_go_v2(
          v_observada::text));
    RETURN;
  END IF;
  IF v_base.estado NOT IN ('aplicada','denegada_vec')
     OR v_base.huella_orden_sha256 IS NULL
     OR v_base.huella_version IS DISTINCT FROM
        v_base.huella_orden_sha256
     OR v_base.terminal_raiz IS NULL
     OR NOT (v_base.ambito_idempotencia_hmac=ANY(v_ambitos))
     OR NOT EXISTS(
       SELECT 1 FROM vec_contratacion_temporal
         .alias_operacion_decision_cobertura a
        WHERE a.ambito_raiz_hmac=v_raices[1]
          AND a.alias_ambito_hmac=
            v_base.ambito_idempotencia_hmac
          AND a.alias_huella_semantica_hmac=
            v_base.huella_semantica_hmac
     ) THEN
    RAISE EXCEPTION USING ERRCODE='55000',
      MESSAGE='resultado terminal O4-05 no confiable';
  END IF;
  v_recibo:=
    vec_contratacion_temporal.o404e_leer_terminal_interno_v1(
      v_raices[1],v_base.huella_orden_sha256);
  IF v_recibo IS NULL
     OR v_recibo->>'ambito_idempotencia_hmac' IS DISTINCT FROM
        v_base.ambito_idempotencia_hmac
     OR v_recibo->>'huella_semantica_hmac' IS DISTINCT FROM
        v_base.huella_semantica_hmac THEN
    RAISE EXCEPTION USING ERRCODE='55000',
      MESSAGE='resultado terminal O4-05 no confiable';
  END IF;
  RETURN QUERY SELECT pg_catalog.jsonb_build_object(
    'esquema',
     'vec.contratacion-temporal.resultado-recuperacion-propia-decision-cobertura.o4-05.v1',
    'estado','confirmado',
    'organizacion_ref',v_base.organizacion_ref,
    'expediente_ref',v_base.expediente_ref,
    'version_expediente',v_base.version_expediente,
    'reserva_ref',v_base.reserva_ref,
    'recibo_ref',v_base.recibo_ref,
    'actuacion_ref',v_base.actuacion_ref,
    'auditoria_ref',v_base.auditoria_ref,
    'evento_ref',v_base.evento_ref,
    'correlacion_vec_ref',v_base.correlacion_vec_ref,
    'decision_vec_ref',v_base.decision_vec_ref,
    'ambito_idempotencia_hmac',v_base.ambito_idempotencia_hmac,
    'huella_semantica_hmac',v_base.huella_semantica_hmac,
    'revision_cercado',v_base.revision_cercado,
    'observada_en_db',
      vec_contratacion_temporal.texto_instante_utc_go_v2(
        v_base.observada_en::text),
    'recibo',v_recibo,
    'observada_en',
      vec_contratacion_temporal.texto_instante_utc_go_v2(
        v_observada::text));
END
$funcion$;

UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
 SET version_esquema=15,
 actualizada_en=pg_catalog.date_trunc(
   'microseconds',pg_catalog.clock_timestamp())
 WHERE control AND version_esquema=14;
REVOKE ALL ON FUNCTION
 vec_contratacion_temporal
  .recuperar_resultado_propio_decision_cobertura_o405_v1(jsonb)
FROM PUBLIC,vec_contratacion_temporal_migrador,
 vec_contratacion_temporal_gobernador,
 vec_contratacion_temporal_ejecutor,
 vec_contratacion_temporal_confirmador_cobertura;
GRANT USAGE ON SCHEMA vec_contratacion_temporal
 TO vec_contratacion_temporal_lector_resultado_cobertura;
GRANT EXECUTE ON FUNCTION
 vec_contratacion_temporal
  .recuperar_resultado_propio_decision_cobertura_o405_v1(jsonb)
 TO vec_contratacion_temporal_lector_resultado_cobertura;
COMMIT;
