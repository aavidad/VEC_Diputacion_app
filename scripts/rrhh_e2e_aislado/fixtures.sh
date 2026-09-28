#!/usr/bin/env bash
# Sourceable. Solo material de desarrollo y lecturas de una base PG18 desechable.
# No fabrica actuaciones, concesiones ni recibos mediante INSERT administrativo.

rrhh_e2e_error_fixture() {
  printf 'RRHH E2E fixture: %s\n' "$1" >&2
  return 1
}

rrhh_e2e_preparar_material() {
  local raiz material
  raiz=${VEC_E2E_ROOT:-}
  material=${VEC_E2E_MATERIAL:-}
  [[ -n $raiz && -d $raiz && -f $raiz/ESPECIFICACIONES_AGENTES.md ]] ||
    { rrhh_e2e_error_fixture 'VEC_E2E_ROOT no apunta al repositorio'; return 1; }
  [[ $material == /* && $material != "$raiz" && $material != "$raiz/"* ]] ||
    { rrhh_e2e_error_fixture 'VEC_E2E_MATERIAL debe ser absoluto y externo al repositorio'; return 1; }
  [[ -x $raiz/scripts/generar_credenciales_desarrollo.sh ]] ||
    { rrhh_e2e_error_fixture 'falta el generador de credenciales de desarrollo'; return 1; }
  # El generador comprueba la integridad al repetir la llamada. Ocultar rutas
  # privadas y material criptográfico en los registros del arnés.
  if ! "$raiz/scripts/generar_credenciales_desarrollo.sh" "$material" >/dev/null 2>&1; then
    rrhh_e2e_error_fixture 'material mTLS sintético no generado o no verificable'
    return 1
  fi
  [[ -s $material/mtls/cliente.crt && -s $material/mtls/cliente.key &&
     -s $material/mtls/intervencion.crt && -s $material/mtls/intervencion.key &&
     -s $material/ca/ca.crt ]] ||
    { rrhh_e2e_error_fixture 'faltan certificados mTLS sintéticos'; return 1; }
  export VEC_AUDITORIA_E2E_CA="$material/ca/ca.crt"
  export VEC_AUDITORIA_E2E_CERT="$material/mtls/cliente.crt"
  export VEC_AUDITORIA_E2E_KEY="$material/mtls/cliente.key"
}

rrhh_e2e_psql_fixture() {
  local consulta=$1
  docker exec -i -e PGOPTIONS='-c statement_timeout=4000 -c default_transaction_read_only=on' \
    "$VEC_E2E_PG_CONTAINER" psql -X -A -t -q -v ON_ERROR_STOP=1 \
    -U postgres -d postgres -c "$consulta" 2>/dev/null
}

rrhh_e2e_verificar_sql_fixture() {
  local estado
  [[ -n ${VEC_E2E_PG_CONTAINER:-} &&
     ${VEC_E2E_PG_PORT:-} =~ ^[0-9]{1,5}$ ]] ||
    { rrhh_e2e_error_fixture 'faltan contenedor o puerto PostgreSQL'; return 1; }
  command -v docker >/dev/null 2>&1 ||
    { rrhh_e2e_error_fixture 'docker no disponible para verificar el PG18 aislado'; return 1; }
  [[ $(docker inspect -f '{{.State.Running}}' "$VEC_E2E_PG_CONTAINER" 2>/dev/null) == true ]] ||
    { rrhh_e2e_error_fixture 'contenedor PostgreSQL aislado no está en ejecución'; return 1; }
  docker exec "$VEC_E2E_PG_CONTAINER" sh -c 'command -v psql >/dev/null' 2>/dev/null ||
    { rrhh_e2e_error_fixture 'psql no disponible dentro del contenedor PG18'; return 1; }
  estado=$(rrhh_e2e_psql_fixture "SELECT CASE
    WHEN current_setting('server_version_num')::integer/10000<>18 THEN 'requiere PostgreSQL 18'
    WHEN to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_auditoria_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL THEN 'falta AD3-91, consumo Audit CT'
    WHEN to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_auditoria_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL THEN 'falta AD3-91, consumo Audit Bolsa'
    WHEN to_regprocedure('vec_contratacion_temporal.consultar_auditoria_ct_atestada_v1(text,text,text,timestamptz,timestamptz,integer,timestamptz,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL THEN 'falta CT132, lectura Audit CT'
    WHEN to_regprocedure('vec_bolsa_llamamientos.consultar_auditoria_participacion_v1(text,text,timestamptz,timestamptz,timestamptz,text,text,integer,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL THEN 'falta B48, lectura Audit Bolsa'
    WHEN to_regclass('vec_bolsa_llamamientos.politica_ofertas_version') IS NULL THEN 'falta B47, política de ofertas'
    WHEN to_regprocedure('vec_bolsa_llamamientos.consultar_politica_ofertas_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL THEN 'falta B51, lectura autorizada de política'
    WHEN to_regprocedure('vec_bolsa_llamamientos.verificar_politica_oferta_b47()') IS NULL OR
         position('horas_naturales' in pg_get_functiondef(to_regprocedure('vec_bolsa_llamamientos.verificar_politica_oferta_b47()')))=0 THEN 'falta B54, cómputo 48 horas naturales'
    WHEN to_regclass('vec_contratacion_temporal.cese_nombramiento_v1') IS NULL THEN 'falta CT115, antecedente de cese'
    WHEN to_regclass('vec_contratacion_temporal.reincorporacion_titular_v1') IS NULL THEN 'falta CT130, retorno de titular'
    WHEN to_regprocedure('vec_contratacion_temporal.preparar_reincorporacion_titular_acreditada_v1(jsonb,text,text)') IS NULL THEN 'falta CT134, lectura acreditada del antecedente'
    ELSE 'ok' END" ) ||
    { rrhh_e2e_error_fixture 'no se pudo leer PostgreSQL aislado como postgres'; return 1; }
  [[ $estado == ok ]] || { rrhh_e2e_error_fixture "$estado"; return 1; }
}

# Después de que los casos de uso creen historia, seleccionar un par real de
# actuaciones por actor y fuente. Se exportan solo referencias sintéticas.
rrhh_e2e_seleccionar_auditoria_fixture() {
  local ct bolsa
  ct=$(rrhh_e2e_psql_fixture "SELECT v.expediente_ref || E'\\t' || actor
    FROM (SELECT v.expediente_ref,v.registrada_en,
                 CASE WHEN v.version=1 THEN al.actor_ref ELSE a.actuacion_json->>'actor_ref' END actor,
                 v.version
            FROM vec_contratacion_temporal.expediente_version_integral v
            LEFT JOIN vec_contratacion_temporal.actuacion_alta al
              ON al.expediente_ref=v.expediente_ref AND v.version=1
            LEFT JOIN vec_contratacion_temporal.actuacion_expediente_integral a
              ON a.expediente_ref=v.expediente_ref AND a.version_expediente=v.version) v
    WHERE actor<>'' AND registrada_en>=clock_timestamp()-interval '28 days'
    GROUP BY expediente_ref,actor HAVING count(*)>=2 AND max(version)>=2
    ORDER BY expediente_ref,actor LIMIT 1") ||
    { rrhh_e2e_error_fixture 'no se pudo consultar la historia CT'; return 1; }
  [[ $ct == *$'\t'* ]] ||
    { rrhh_e2e_error_fixture 'falta expediente CT con dos versiones del mismo actor creadas por casos de uso'; return 1; }
  bolsa=$(rrhh_e2e_psql_fixture "WITH hechos AS (
      SELECT participacion_ref,actor,registrada_en,false AS con_cambio
        FROM vec_bolsa_llamamientos.situacion_participacion
      UNION ALL
      SELECT participacion_ref,actor,registrada_en,
             valor_anterior IS NOT NULL AND valor_nuevo IS NOT NULL AS con_cambio
        FROM vec_bolsa_llamamientos.traza_valor_participacion)
    SELECT participacion_ref || E'\\t' || actor FROM hechos
    WHERE actor<>'' AND registrada_en>=clock_timestamp()-interval '28 days'
    GROUP BY participacion_ref,actor
    HAVING count(*)>=2 AND bool_or(con_cambio)
    ORDER BY participacion_ref,actor LIMIT 1") ||
    { rrhh_e2e_error_fixture 'no se pudo consultar la historia Bolsa'; return 1; }
  [[ $bolsa == *$'\t'* ]] ||
    { rrhh_e2e_error_fixture 'falta participación Bolsa con dos hechos y una traza antes/después del mismo actor'; return 1; }
  if [[ -n ${VEC_AUDITORIA_CONSULTA_EXPEDIENTE_CT:-} &&
        ${VEC_AUDITORIA_CONSULTA_EXPEDIENTE_CT} != "${ct%%$'\t'*}" ]]; then
    rrhh_e2e_error_fixture 'la concesión Audit CT apunta a otro expediente que la historia seleccionada'
    return 1
  fi
  if [[ -n ${VEC_AUDITORIA_CONSULTA_EXPEDIENTE_BOLSA:-} &&
        ${VEC_AUDITORIA_CONSULTA_EXPEDIENTE_BOLSA} != "${bolsa%%$'\t'*}" ]]; then
    rrhh_e2e_error_fixture 'la concesión Audit Bolsa apunta a otra participación que la historia seleccionada'
    return 1
  fi
  export VEC_AUDITORIA_E2E_CT_EXPEDIENTE=${ct%%$'\t'*}
  export VEC_AUDITORIA_E2E_CT_ACTOR=${ct#*$'\t'}
  export VEC_AUDITORIA_E2E_BOLSA_EXPEDIENTE=${bolsa%%$'\t'*}
  export VEC_AUDITORIA_E2E_BOLSA_ACTOR=${bolsa#*$'\t'}
  export VEC_AUDITORIA_E2E_CT_AJENO='expediente:ct:e2e:ajeno'
  export VEC_AUDITORIA_E2E_BOLSA_AJENO='participacion:bolsa:e2e:ajena'
  export VEC_AUDITORIA_E2E_DESDE VEC_AUDITORIA_E2E_HASTA
  VEC_AUDITORIA_E2E_HASTA=$(date -u -d '+1 minute' +'%Y-%m-%dT%H:%M:%SZ')
  VEC_AUDITORIA_E2E_DESDE=$(date -u -d '29 days ago' +'%Y-%m-%dT%H:%M:%SZ')
}

rrhh_e2e_verificar_politica_fixture() {
  local bolsa_ref
  bolsa_ref=$(rrhh_e2e_psql_fixture "SELECT p.bolsa_ref
      FROM vec_bolsa_llamamientos.politica_ofertas_version p
      WHERE p.politica#>>'{plazo,unidad}'='horas_naturales'
        AND p.politica#>>'{plazo,cantidad}'='48'
        AND p.politica#>>'{plazo,computo}'='continuo_utc'
        AND p.politica#>>'{no_cubierta,accion}'='llamamiento_directo'
        AND p.politica#>>'{no_cubierta,condicion}'='sin_disposiciones_elegibles'
        AND NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.politica_ofertas_version newer
                        WHERE newer.bolsa_ref=p.bolsa_ref AND newer.version>p.version)
      ORDER BY p.bolsa_ref LIMIT 1") ||
    { rrhh_e2e_error_fixture 'no se pudo consultar la política B47'; return 1; }
  [[ $bolsa_ref == bolsa:* ]] ||
    { rrhh_e2e_error_fixture 'falta publicación nominal B47 de 48 horas y no cubierta'; return 1; }
  export VEC_E2E_BOLSA_REF="$bolsa_ref"
}

# Puerta opcional para el tramo posterior; no se declara completado por tener
# funciones instaladas. Exige un cese y un retorno enlazados por recibo real.
rrhh_e2e_verificar_cese_reincorporacion_fixture() {
  local estado
  estado=$(rrhh_e2e_psql_fixture "SELECT CASE WHEN EXISTS (
    SELECT 1 FROM vec_contratacion_temporal.reincorporacion_titular_v1 r
    JOIN vec_contratacion_temporal.cese_nombramiento_v1 c
      ON c.evento_ref=r.cese_evento_ref AND c.recibo_ref=r.cese_recibo_ref
    WHERE r.confirmada_en>=clock_timestamp()-interval '28 days'
  ) THEN 'ok' ELSE 'falta CT115/CT130 enlazados por caso de uso y recibo conservado' END") ||
    { rrhh_e2e_error_fixture 'no se pudo consultar cese y reincorporación CT'; return 1; }
  [[ $estado == ok ]] || { rrhh_e2e_error_fixture "$estado"; return 1; }
}

rrhh_e2e_preparar_fixtures() {
  rrhh_e2e_preparar_material || return 1
  rrhh_e2e_verificar_sql_fixture || return 1
  rrhh_e2e_seleccionar_auditoria_fixture || return 1
  rrhh_e2e_verificar_politica_fixture || return 1
}
