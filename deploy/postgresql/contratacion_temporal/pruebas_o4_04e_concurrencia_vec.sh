#!/usr/bin/env bash
# Carrera aislada CT contra escritor VEC externo; se carga con source.
# shellcheck disable=SC2154
paso 'concurrencia adversaria: reserva CT contra escritor VEC externo'
preparar_normal_o404e adv_vec
snapshot_vec_producto_antes=$(snapshot_identidad_o404e adv_vec producto)
snapshot_vec_cadenas_antes=$(snapshot_cadenas_o404e)
IFS='|' read -r sec_auditoria_vec_antes sec_outbox_vec_antes < <(
  psql_admin --tuples-only --no-align --field-separator='|' --command \
    'SELECT secuencia_auditoria,secuencia_outbox FROM
       vec_contratacion_temporal.control_cadenas_expediente_integral' \
    </dev/null)
psql_admin <<'SQL' >/dev/null
CREATE FUNCTION vec_o404e_fallos.escribir_vec_externo_adversario(p_caso text)
RETURNS void LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security='on' SET timezone='UTC'
AS $$
DECLARE
  q jsonb:=vec_o404e_concedida.carga(p_caso);
  c jsonb:=q->'cabecera'; x jsonb:=q->'decision_vec'; l jsonb;
  v record;
BEGIN
  IF session_user IS DISTINCT FROM 'vec_o404e_tcb' THEN
    RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='actor VEC externo inválido';
  END IF;
  l:=vec_contratacion_temporal.o404e_construir_lote_c1_v1(
    q,c->>'ambito_idempotencia_hmac');
  SELECT * INTO v FROM
    vec_autorizacion.registrar_decision_cobertura_contexto_exacto_o404e_v1(
      decode(x->>'decision_canonica_hex','hex'),
      decode(x->>'motivo_canonico_hex','hex'),
      (x->>'persona_version')::numeric,(x->>'perfil_version')::numeric,
      jsonb_build_object(
        'rama','concedida','accion',x->>'accion',
        'organizacion_ref',c->>'organizacion_ref',
        'expediente_ref',c->>'expediente_ref',
        'version_expediente',c->'version_expediente',
        'reserva_ref',c->>'reserva_ref','decision_ref',c->>'decision_vec_ref',
        'correlacion_ref',c->>'correlacion_vec_ref',
        'finalidad',x->>'finalidad',
        'contexto_recurso_huella_sha256',
          x->>'contexto_recurso_huella_sha256',
        'recurso_modulo',x->>'recurso_modulo',
        'recurso_ref',x->>'recurso_ref','recurso_tipo',x->>'recurso_tipo',
        'ambitos',x->'ambitos','atributos',x->'atributos',
        'huella_orden_sha256',CASE
          WHEN c->>'huella_orden_sha256' IS NOT DISTINCT FROM
            repeat('a',64) THEN repeat('b',64)
          ELSE repeat('a',64) END,
        'lote_huella_sha256',l->>'lote_huella_sha256'));
  IF NOT FOUND THEN
    RAISE EXCEPTION USING ERRCODE='Z04E3',MESSAGE='escritor VEC no enlazó';
  END IF;
END
$$;
REVOKE ALL ON FUNCTION
  vec_o404e_fallos.escribir_vec_externo_adversario(text) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_o404e_fallos TO vec_o404e_tcb;
GRANT EXECUTE ON FUNCTION
  vec_o404e_fallos.escribir_vec_externo_adversario(text) TO vec_o404e_tcb;
SQL
lanzar_vec_externo_o404e() {
  local salida=$1 estado=$2
  (
    set +e
    trap - ERR
    ejecutar_acotado_o404e 17s 1s docker exec \
      --env PGAPPNAME=o404e_r4_b --interactive "$contenedor" \
      psql -X --set ON_ERROR_STOP=1 --set VERBOSITY=sqlstate --quiet \
      --username vec_o404e_tcb --dbname postgres >"$salida" 2>&1 <<'SQL'
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
SELECT vec_o404e_fallos.escribir_vec_externo_adversario('adv_vec');
COMMIT;
\echo O404E_COMMIT
SQL
    printf '%s\n' "$?" >"$estado"
  ) &
  o404e_pid_ultimo=$!
}
iniciar_bloqueador_o404e o404e_bloqueo_vec \
  'LOCK TABLE vec_autorizacion.enlace_decision_cobertura_ct_o404e IN SHARE MODE'
lanzar_confirmacion_o404e adv_vec o404e_r4_a \
  "$o404e_tmp/r4-a.out" "$o404e_tmp/r4-a.rc"
pid_a=$o404e_pid_ultimo
lanzar_vec_externo_o404e "$o404e_tmp/r4-b.out" "$o404e_tmp/r4-b.rc"
pid_b=$o404e_pid_ultimo
esperar_actividad_o404e o404e_r4_ 2
liberar_bloqueador_o404e
esperar_carrera_o404e "$pid_a" "$pid_b"
validar_un_ganador_o404e \
  "$o404e_tmp/r4-a.out" "$o404e_tmp/r4-a.rc" \
  "$o404e_tmp/r4-b.out" "$o404e_tmp/r4-b.rc" \
  '23505|40001' 'identidad/enlace VEC'
ganador_ct=false
[[ $o404e_ganador == a ]] && ganador_ct=true
snapshot_vec_producto_despues=$(snapshot_identidad_o404e adv_vec producto)
snapshot_vec_cadenas_despues=$(snapshot_cadenas_o404e)
if [[ $ganador_ct == true ]]; then
  reconciliar_exacto_o404e adv_vec
else
  reconciliar_exacto_o404e adv_vec ausente
  exigir_snapshot_igual_o404e "$snapshot_vec_producto_antes" \
    "$snapshot_vec_producto_despues" 'rollback CT ante escritor VEC'
  exigir_snapshot_igual_o404e "$snapshot_vec_cadenas_antes" \
    "$snapshot_vec_cadenas_despues" 'cadenas con ganador VEC externo'
fi
psql_admin --set "o404e_ganador_ct=$ganador_ct" \
  --set "o404e_sec_a=$sec_auditoria_vec_antes" \
  --set "o404e_sec_o=$sec_outbox_vec_antes" <<'SQL' >/dev/null
SELECT set_config('vec.prueba_o404e_ganador_ct',:'o404e_ganador_ct',false);
SELECT set_config('vec.prueba_o404e_sec_a',:'o404e_sec_a',false);
SELECT set_config('vec.prueba_o404e_sec_o',:'o404e_sec_o',false);
DO $vec$
DECLARE
  ct boolean:=current_setting('vec.prueba_o404e_ganador_ct')::boolean;
  n integer:=ct::integer;
  q jsonb:=vec_o404e_concedida.carga('adv_vec');
  c jsonb:=q->'cabecera'; x jsonb:=q->'decision_vec';
  l jsonb:=vec_contratacion_temporal.o404e_construir_lote_c1_v1(
    q,c->>'ambito_idempotencia_hmac');
  orden_esperado text:=CASE WHEN ct THEN c->>'huella_orden_sha256'
    WHEN c->>'huella_orden_sha256' IS NOT DISTINCT FROM
      repeat('a',64) THEN repeat('b',64)
    ELSE repeat('a',64) END;
  sec_a numeric:=current_setting('vec.prueba_o404e_sec_a')::numeric;
  sec_o numeric:=current_setting('vec.prueba_o404e_sec_o')::numeric;
BEGIN
  IF (SELECT count(*) FROM
        vec_autorizacion.enlace_decision_cobertura_ct_o404e
       WHERE decision_ref=c->>'decision_vec_ref') IS DISTINCT FROM 1
     OR (SELECT count(*) FILTER (WHERE
          e.rama IS NOT DISTINCT FROM 'concedida'
          AND e.concedida IS NOT DISTINCT FROM true
          AND e.codigo IS NOT DISTINCT FROM 'concedida'
          AND e.accion IS NOT DISTINCT FROM x->>'accion'
          AND e.decision_huella_sha256 IS NOT DISTINCT FROM
            x->>'decision_huella_sha256'
          AND e.correlacion_ref IS NOT DISTINCT FROM c->>'correlacion_vec_ref'
          AND e.organizacion_ref IS NOT DISTINCT FROM c->>'organizacion_ref'
          AND e.expediente_ref IS NOT DISTINCT FROM c->>'expediente_ref'
          AND e.version_expediente IS NOT DISTINCT FROM
            (c->>'version_expediente')::numeric
          AND e.reserva_ref IS NOT DISTINCT FROM c->>'reserva_ref'
          AND e.contexto_recurso_huella_sha256 IS NOT DISTINCT FROM
            x->>'contexto_recurso_huella_sha256'
          AND e.huella_orden_sha256 IS NOT DISTINCT FROM orden_esperado
          AND e.lote_huella_sha256 IS NOT DISTINCT FROM
            l->>'lote_huella_sha256'
          AND e.prueba_vinculo_sha256 IS NOT DISTINCT FROM
            pg_catalog.encode(pg_catalog.sha256(
              pg_catalog.decode(e.decision_huella_sha256,'hex')||
              pg_catalog.decode(e.contexto_recurso_huella_sha256,'hex')||
              pg_catalog.decode(e.huella_orden_sha256,'hex')||
              pg_catalog.decode(e.lote_huella_sha256,'hex')||
              pg_catalog.int4send(pg_catalog.octet_length(e.decision_ref))||
              pg_catalog.convert_to(e.decision_ref,'UTF8')||
              pg_catalog.int4send(
                pg_catalog.octet_length(e.correlacion_ref))||
              pg_catalog.convert_to(e.correlacion_ref,'UTF8')||
              pg_catalog.int8send((extract(epoch FROM e.registrada_en)::
                numeric*1000000)::bigint)||
              pg_catalog.int8send((extract(epoch FROM e.revalidada_en)::
                numeric*1000000)::bigint)
            ),'hex')
          AND e.registrada_en IS NOT DISTINCT FROM
            (SELECT d.registrada_en FROM
              vec_autorizacion.decision_concedida_contexto_actor_v3 d
             WHERE d.decision_ref=c->>'decision_vec_ref')
          AND e.revalidada_en IS NOT NULL
          AND e.revalidada_en>=e.registrada_en)
       FROM vec_autorizacion.enlace_decision_cobertura_ct_o404e e
       WHERE e.decision_ref=c->>'decision_vec_ref') IS DISTINCT FROM 1
     OR (SELECT count(*) FROM
        vec_autorizacion.decision_concedida_contexto_actor_v3 d
       WHERE d.decision_ref=c->>'decision_vec_ref') IS DISTINCT FROM 1
     OR (SELECT count(*) FILTER (WHERE
          d.huella_decision_sha256 IS NOT DISTINCT FROM
            x->>'decision_huella_sha256'
          AND d.decision_canonica IS NOT DISTINCT FROM
            decode(x->>'decision_canonica_hex','hex')
          AND d.documento->>'decision_ref' IS NOT DISTINCT FROM
            c->>'decision_vec_ref'
          AND d.documento->>'correlacion_ref' IS NOT DISTINCT FROM
            c->>'correlacion_vec_ref'
          AND d.documento->>'recurso_ref' IS NOT DISTINCT FROM
            c->>'reserva_ref'
          AND d.documento->>'contexto_recurso_huella_sha256'
            IS NOT DISTINCT FROM
            x->>'contexto_recurso_huella_sha256')
       FROM vec_autorizacion.decision_concedida_contexto_actor_v3 d
       WHERE d.decision_ref=c->>'decision_vec_ref') IS DISTINCT FROM 1
     OR (SELECT count(*) FROM
        vec_contratacion_temporal.confirmacion_operacion_decision_cobertura
       WHERE reserva_ref=c->>'reserva_ref') IS DISTINCT FROM n
     OR (SELECT count(*) FROM vec_contratacion_temporal.consumo_cobertura_lote
       WHERE reserva_ref=c->>'reserva_ref') IS DISTINCT FROM n
     OR (SELECT count(*) FROM
        vec_contratacion_temporal.consumo_cobertura_evidencia e
        JOIN vec_contratacion_temporal.consumo_cobertura_lote l USING(lote_ref)
       WHERE l.reserva_ref=c->>'reserva_ref') IS DISTINCT FROM n
     OR (SELECT count(*) FROM vec_contratacion_temporal
          .acreditacion_gobierno_decision_cobertura
       WHERE reserva_ref=c->>'reserva_ref') IS DISTINCT FROM n
     OR (SELECT count(*) FROM
        vec_contratacion_temporal.expediente_version_integral
       WHERE operacion_ref=c->>'actuacion_ref') IS DISTINCT FROM n
     OR (SELECT count(*) FROM
        vec_contratacion_temporal.actuacion_expediente_integral
       WHERE operacion_ref=c->>'actuacion_ref') IS DISTINCT FROM n
     OR (SELECT count(*) FROM vec_contratacion_temporal
          .decision_cobertura_gobernada_durable
       WHERE reserva_ref=c->>'reserva_ref') IS DISTINCT FROM n
     OR (SELECT count(*) FROM
        vec_contratacion_temporal.auditoria_decision_cobertura
       WHERE reserva_ref=c->>'reserva_ref') IS DISTINCT FROM n
     OR (SELECT count(*) FROM
        vec_contratacion_temporal.outbox_expediente_integral
       WHERE evento_ref=c->>'evento_ref') IS DISTINCT FROM n
     OR (SELECT count(*) FROM
        vec_contratacion_temporal.terminal_operacion_decision_cobertura t
        JOIN vec_contratacion_temporal.reserva_operacion_decision_cobertura b
          USING(ambito_raiz_hmac)
       WHERE b.reserva_ref=c->>'reserva_ref') IS DISTINCT FROM n
     OR (SELECT version FROM
        vec_contratacion_temporal.expediente_integral_actual
       WHERE expediente_ref=c->>'expediente_ref') IS DISTINCT FROM
          (CASE WHEN ct THEN 3 ELSE 2 END) THEN
    RAISE EXCEPTION 'carrera VEC dejó efectos parciales';
  END IF;
  IF ct AND (
       (SELECT secuencia FROM
          vec_contratacion_temporal.reserva_operacion_decision_cobertura_actual a
          JOIN vec_contratacion_temporal.reserva_operacion_decision_cobertura b
            USING(ambito_raiz_hmac) WHERE b.reserva_ref=c->>'reserva_ref')
         IS DISTINCT FROM 2
       OR (SELECT estado FROM
          vec_contratacion_temporal.reserva_operacion_decision_cobertura_version v
          JOIN vec_contratacion_temporal.reserva_operacion_decision_cobertura b
            USING(ambito_raiz_hmac) WHERE b.reserva_ref=c->>'reserva_ref'
            ORDER BY v.secuencia DESC LIMIT 1) IS DISTINCT FROM 'aplicada'
       OR (SELECT secuencia_auditoria FROM
          vec_contratacion_temporal.control_cadenas_expediente_integral)
         IS DISTINCT FROM sec_a+1
       OR (SELECT secuencia_outbox FROM
          vec_contratacion_temporal.control_cadenas_expediente_integral)
         IS DISTINCT FROM sec_o+1
       OR (SELECT cabeza_auditoria_sha256 FROM
          vec_contratacion_temporal.control_cadenas_expediente_integral)
         IS DISTINCT FROM (SELECT huella_sha256 FROM
          vec_contratacion_temporal.auditoria_decision_cobertura
          WHERE reserva_ref=c->>'reserva_ref')
       OR (SELECT cabeza_outbox_sha256 FROM
          vec_contratacion_temporal.control_cadenas_expediente_integral)
         IS DISTINCT FROM (SELECT huella_sha256 FROM
          vec_contratacion_temporal.outbox_expediente_integral
          WHERE evento_ref=c->>'evento_ref')) THEN
    RAISE EXCEPTION 'ganador CT no cerró estado/cadenas exactos';
  ELSIF NOT ct AND (
       (SELECT secuencia FROM
          vec_contratacion_temporal.reserva_operacion_decision_cobertura_actual a
          JOIN vec_contratacion_temporal.reserva_operacion_decision_cobertura b
            USING(ambito_raiz_hmac) WHERE b.reserva_ref=c->>'reserva_ref')
         IS DISTINCT FROM 1
       OR (SELECT estado FROM
          vec_contratacion_temporal.reserva_operacion_decision_cobertura_version v
          JOIN vec_contratacion_temporal.reserva_operacion_decision_cobertura b
            USING(ambito_raiz_hmac) WHERE b.reserva_ref=c->>'reserva_ref'
            ORDER BY v.secuencia DESC LIMIT 1) IS DISTINCT FROM 'reservada') THEN
    RAISE EXCEPTION 'ganador VEC externo dejó reserva CT parcial';
  END IF;
END
$vec$;
DROP FUNCTION vec_o404e_fallos.escribir_vec_externo_adversario(text);
SQL
