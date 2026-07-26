-- O4-04E/9: ligaduras causales de la concesión antes de VEC y C1.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_catalog.pg_advisory_xact_lock(
  pg_catalog.hashtextextended(
    'vec_contratacion_temporal:o4_04:migraciones',0
  )
);
SELECT control
  FROM vec_contratacion_temporal.control_migracion_cobertura_o4
 WHERE control AND version_esquema=11
 FOR UPDATE;
DO $prevalidacion$
BEGIN
  IF NOT EXISTS(
    SELECT 1
      FROM vec_contratacion_temporal.control_migracion_cobertura_o4
     WHERE control AND version_esquema=11
  ) OR pg_catalog.to_regprocedure(
    'vec_contratacion_temporal.o404e_concesion_ligada_v1(jsonb,jsonb)'
  ) IS NOT NULL THEN
    RAISE EXCEPTION USING ERRCODE='55000',
      MESSAGE='estado incompatible para ligaduras concedidas O4-04E';
  END IF;
END
$prevalidacion$;

-- Barrera O4-04E nominativa: el TCB confirmador no hereda el ejecutor
-- genérico O4B. Revalida la misma historia actual desde el propietario.
CREATE FUNCTION
vec_contratacion_temporal.o404e_revalidar_gobierno_actual_v1(
  p_organizacion_ref text,p_accion text,
  p_catalogo_ref text,p_catalogo_version numeric,p_catalogo_huella text,
  p_politica_ref text,p_politica_version numeric,p_politica_huella text,
  p_actuacion_ref text,p_actuacion_version numeric,p_actuacion_huella text
)
RETURNS boolean
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path=pg_catalog
SET row_security='on'
SET timezone='UTC'
SET lock_timeout='2s'
AS $funcion$
DECLARE
  v_ahora timestamptz(6):=pg_catalog.date_trunc(
    'microseconds',pg_catalog.clock_timestamp());
  v_secuencia bigint;
BEGIN
  IF current_user<>'vec_contratacion_temporal_propietario'
     OR session_user=current_user
     OR NOT pg_catalog.pg_has_role(
       session_user,'vec_contratacion_temporal_confirmador_cobertura','MEMBER')
     OR pg_catalog.pg_has_role(
       session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
     OR pg_catalog.pg_has_role(
       session_user,'vec_contratacion_temporal_migrador','MEMBER')
     OR vec_contratacion_temporal.gobi_o404b_entorno_valido(false)
        IS NOT TRUE
     OR p_organizacion_ref!~'^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
     OR p_accion NOT IN(
       'contratacion_temporal.cobertura.decidir',
       'contratacion_temporal.cobertura.rectificar')
     OR p_catalogo_huella!~'^[a-f0-9]{64}$'
     OR p_politica_huella!~'^[a-f0-9]{64}$'
     OR p_actuacion_huella!~'^[a-f0-9]{64}$'
  THEN RETURN false; END IF;
  PERFORM pg_catalog.pg_advisory_xact_lock_shared(
    pg_catalog.hashtextextended(
      'vec_contratacion_temporal:o4_04:migraciones',0));
  SELECT ultima_secuencia INTO STRICT v_secuencia
    FROM vec_contratacion_temporal.gobi_o404b_checkpoint
   WHERE control FOR SHARE;
  PERFORM 1
    FROM vec_contratacion_temporal.gobi_o404b_actual x
    JOIN vec_contratacion_temporal.gobi_o404b_actuacion a
      ON a.referencia=x.actuacion_ref
     AND a.version=x.actuacion_version
     AND a.huella_sha256=x.actuacion_huella_sha256
    JOIN vec_contratacion_temporal.gobi_o404b_politica d
      ON d.referencia=a.politica_ref AND d.version=a.politica_version
     AND d.huella_sha256=a.politica_huella_sha256
    JOIN vec_contratacion_temporal.gobi_o404b_catalogo c
      ON c.referencia=a.catalogo_ref AND c.version=a.catalogo_version
     AND c.huella_sha256=a.catalogo_huella_sha256
   WHERE x.organizacion_ref=p_organizacion_ref AND x.accion=p_accion
     AND x.secuencia<=v_secuencia
     AND c.referencia=p_catalogo_ref AND c.version=p_catalogo_version
     AND c.huella_sha256=p_catalogo_huella
     AND d.referencia=p_politica_ref AND d.version=p_politica_version
     AND d.huella_sha256=p_politica_huella
     AND a.referencia=p_actuacion_ref AND a.version=p_actuacion_version
     AND a.huella_sha256=p_actuacion_huella
     AND a.publicada_en<=v_ahora AND a.vigente_desde<=v_ahora
     AND v_ahora<a.vigente_hasta
     AND d.publicada_en<=v_ahora AND d.vigente_desde<=v_ahora
     AND v_ahora<d.vigente_hasta
     AND c.publicado_en<=v_ahora AND c.vigente_desde<=v_ahora
     AND (c.vigente_hasta IS NULL OR v_ahora<c.vigente_hasta)
     AND NOT EXISTS(
       SELECT 1 FROM vec_contratacion_temporal.gobi_o404b_retirada r
        WHERE r.organizacion_ref=x.organizacion_ref AND r.accion=x.accion
          AND r.actuacion_ref=x.actuacion_ref
          AND r.actuacion_version=x.actuacion_version
          AND r.actuacion_huella_sha256=x.actuacion_huella_sha256
          AND r.retirada_en<=v_ahora)
   FOR SHARE OF x,a,d,c;
  RETURN FOUND;
EXCEPTION WHEN data_exception OR invalid_text_representation
  OR datetime_field_overflow OR no_data_found OR too_many_rows THEN
  RETURN false;
END
$funcion$;

-- Repite las precondiciones del agregado Go que no forman parte del canon
-- binario de propuesta/decisión. La función exterior solo usa esta barrera.
CREATE FUNCTION
vec_contratacion_temporal.o404e_transicion_dominio_ligada_v1(
  p_anterior jsonb,p_siguiente jsonb,p_propuesta jsonb,
  p_motivo jsonb,p_efecto timestamptz
)
RETURNS boolean
LANGUAGE plpgsql
IMMUTABLE
STRICT
SET search_path=pg_catalog
AS $funcion$
DECLARE
  d jsonb;
  a jsonb;
  u jsonb;
  n integer;
  cero jsonb:=pg_catalog.jsonb_build_object(
    'referencia_catalogo',pg_catalog.jsonb_build_object(
      'catalogo_id','','catalogo_version',0,
      'catalogo_huella_sha256','','entrada_clave',''),
    'clave_i18n','');
BEGIN
  IF vec_contratacion_temporal.o404e_transicion_exacta_v1(
       p_anterior,p_siguiente,p_propuesta,p_motivo,p_efecto) IS NOT TRUE
     OR vec_contratacion_temporal.analisis_rrhh_valido_v3(
          p_anterior->'analisis') IS NOT TRUE
     OR p_propuesta->>'estado'<>'viable'
     OR p_efecto<(p_propuesta->>'generada_en')::timestamptz
     OR p_efecto>=(p_propuesta->>'valida_hasta')::timestamptz
     OR NOT EXISTS(
       SELECT 1 FROM pg_catalog.jsonb_array_elements(
         p_propuesta->'evaluaciones') e
        WHERE e->>'via_clave'=
          p_siguiente#>>'{via_cobertura,via_clave}'
          AND e->>'estado'='viable')
  THEN RETURN false; END IF;
  n:=pg_catalog.jsonb_array_length(
    p_siguiente->'decisiones_cobertura');
  d:=p_siguiente#>ARRAY[
    'decisiones_cobertura',(n-1)::text];
  IF d->>'organizacion_ref'<>p_propuesta->>'organizacion_ref'
     OR d->>'expediente_ref'<>p_propuesta->>'expediente_ref'
     OR d->>'version_expediente_origen'<>
        p_propuesta->>'version_expediente'
     OR (d->>'version_expediente')::numeric<>
        (p_propuesta->>'version_expediente')::numeric+1
     OR d->>'propuesta_ref'<>p_propuesta->>'referencia'
     OR d->>'propuesta_huella_sha256'<>p_propuesta->>'huella_sha256'
     OR d->>'preparacion_evidencias_ref'<>
        p_propuesta->>'preparacion_evidencias_ref'
     OR d->>'preparacion_evidencias_huella_sha256'<>
        p_propuesta->>'preparacion_evidencias_huella_sha256'
     OR d->>'analisis_ref'<>p_propuesta->>'analisis_ref'
     OR d->>'analisis_huella_sha256'<>
        p_propuesta->>'analisis_huella_sha256'
     OR d->'catalogo'<>p_propuesta->'catalogo'
     OR d->'politica'<>p_propuesta->'politica'
     OR d->>'via_recomendada'<>p_propuesta->>'via_propuesta'
     OR p_anterior#>>'{analisis,categoria_ref}'<>
        p_propuesta->>'categoria_ref'
     OR p_anterior#>'{analisis,periodo}'<>p_propuesta->'periodo'
  THEN RETURN false; END IF;
  IF d->>'tipo'='inicial' THEN
    RETURN vec_contratacion_temporal.expediente_analisis_valido_v2(
             p_anterior,true) IS TRUE
      AND NOT (p_anterior?'via_cobertura')
      AND NOT (p_anterior?'decisiones_cobertura')
      AND NOT (p_anterior?'asignacion')
      AND pg_catalog.jsonb_array_length(p_anterior->'actuaciones')=
          (p_anterior->>'version')::integer
      AND NOT (d?'predecesora_ref')
      AND NOT (d?'predecesora_huella_sha256')
      AND ((d->>'via_elegida'=p_propuesta->>'via_propuesta'
            AND p_motivo=cero)
        OR (d->>'via_elegida'<>p_propuesta->>'via_propuesta'
            AND p_motivo<>cero));
  END IF;
  IF d->>'tipo'<>'rectificacion'
     OR NOT (p_anterior?'via_cobertura')
     OR NOT (p_anterior?'decisiones_cobertura')
     OR p_anterior?'asignacion'
     OR pg_catalog.jsonb_array_length(p_anterior->'actuaciones')<>
        (p_anterior->>'version')::integer
  THEN RETURN false; END IF;
  n:=pg_catalog.jsonb_array_length(
    p_anterior->'decisiones_cobertura');
  IF n<1 THEN RETURN false; END IF;
  u:=p_anterior#>ARRAY[
    'decisiones_cobertura',(n-1)::text];
  a:=p_anterior#>ARRAY[
    'actuaciones',(pg_catalog.jsonb_array_length(
      p_anterior->'actuaciones')-1)::text];
  RETURN p_anterior#>'{via_cobertura,decision_gobernada}'=u
    AND p_anterior#>>'{via_cobertura,via_clave}'=u->>'via_elegida'
    AND d->>'predecesora_ref'=u->>'referencia'
    AND d->>'predecesora_huella_sha256'=u->>'huella_sha256'
    AND d->>'via_elegida'<>u->>'via_elegida'
    AND d->>'actor_ref'<>u->>'actor_ref'
    AND (d->>'decidida_en')::timestamptz>
        (u->>'decidida_en')::timestamptz
    AND (u#>>'{actuacion,secuencia}')::integer=
        pg_catalog.jsonb_array_length(p_anterior->'actuaciones')
    AND u#>>'{actuacion,version_expediente}'=p_anterior->>'version'
    AND u->'actuacion'=a
    AND p_siguiente->>'fase_actual'=p_anterior->>'fase_actual'
    AND p_siguiente->>'estado_actual'=p_anterior->>'estado_actual'
    AND p_motivo<>cero;
EXCEPTION WHEN data_exception OR invalid_text_representation
  OR datetime_field_overflow OR numeric_value_out_of_range THEN
  RETURN false;
END
$funcion$;

CREATE FUNCTION
vec_contratacion_temporal.o404e_concesion_ligada_v1(
  p_carga jsonb,p_lote jsonb
)
RETURNS boolean
LANGUAGE plpgsql
IMMUTABLE
STRICT
SET search_path=pg_catalog
AS $funcion$
DECLARE
  c jsonb:=p_carga->'cabecera';
  z jsonb:=p_carga->'concesion';
  p jsonb:=z->'propuesta';
BEGIN
  RETURN p_carga->>'rama'='concedida'
    AND pg_catalog.jsonb_typeof(p)='object'
    AND pg_catalog.jsonb_typeof(p_lote)='object'
    AND p->>'organizacion_ref'=c->>'organizacion_ref'
    AND p->>'expediente_ref'=c->>'expediente_ref'
    AND p->>'version_expediente'=c->>'version_expediente'
    AND p->>'analisis_ref'=c->>'analisis_ref'
    AND p->>'analisis_huella_sha256'=c->>'analisis_huella_sha256'
    AND p->>'preparacion_evidencias_ref'=c->>'preparacion_c1_ref'
    AND p->>'preparacion_evidencias_huella_sha256'=
        c->>'preparacion_c1_huella_sha256'
    AND p_lote->>'preparacion_c1_ref'=c->>'preparacion_c1_ref'
    AND p_lote->>'preparacion_c1_huella_sha256'=
        c->>'preparacion_c1_huella_sha256'
    AND (z->>'efecto_en')::timestamptz >=
        (c->>'observada_en_db')::timestamptz
    AND (z->>'efecto_en')::timestamptz >=
        (p->>'generada_en')::timestamptz
    AND (z->>'efecto_en')::timestamptz <
        (c->>'valida_hasta_orden')::timestamptz
    AND (z->>'valida_hasta')::timestamptz =
        (c->>'valida_hasta_orden')::timestamptz;
EXCEPTION WHEN data_exception OR invalid_text_representation
  OR datetime_field_overflow THEN
  RETURN false;
END
$funcion$;

-- Canon binario de IdentidadSemanticaPropuestaDecisionCobertura. Normaliza
-- resultados por clave/resultado y evaluaciones por prioridad/vía.
CREATE FUNCTION
vec_contratacion_temporal.o404e_semantica_propuesta_v1(p jsonb)
RETURNS text
LANGUAGE plpgsql
IMMUTABLE
STRICT
SET search_path=pg_catalog
AS $funcion$
DECLARE
  v bytea:=''::bytea;
  r record;
  e record;
  k record;
  v_lista jsonb;
  v_vistas text[];
  v_clave text;
  v_total integer;
BEGIN
  IF vec_contratacion_temporal.o404e_propuesta_cobertura_exacta_v1(p)
       IS NOT TRUE
     OR pg_catalog.jsonb_array_length(p->'resultados')>512
     OR pg_catalog.jsonb_array_length(p->'evaluaciones') NOT BETWEEN 1 AND 64
     OR EXISTS(
       SELECT 1 FROM pg_catalog.jsonb_array_elements(p->'resultados') x
        GROUP BY x->>'clave' HAVING pg_catalog.count(*)>1)
     OR EXISTS(
       SELECT 1 FROM pg_catalog.jsonb_array_elements(p->'resultados') x
        WHERE pg_catalog.jsonb_typeof(x->'evidencias')<>'array'
           OR pg_catalog.jsonb_array_length(x->'evidencias') NOT BETWEEN 1 AND 4)
     OR EXISTS(
       SELECT 1 FROM pg_catalog.jsonb_array_elements(p->'evaluaciones') x
        GROUP BY x->>'via_clave' HAVING pg_catalog.count(*)>1)
     OR EXISTS(
       SELECT 1 FROM pg_catalog.jsonb_array_elements(p->'evaluaciones') x
        GROUP BY x->>'prioridad' HAVING pg_catalog.count(*)>1)
     OR EXISTS(
       SELECT 1 FROM pg_catalog.jsonb_array_elements(p->'evaluaciones') x
        WHERE (x->>'prioridad')::integer NOT BETWEEN 1 AND 65535)
  THEN RETURN NULL; END IF;
  FOR e IN
    SELECT x FROM pg_catalog.jsonb_array_elements(p->'evaluaciones') x
  LOOP
    v_vistas:=ARRAY[]::text[];
    v_total:=0;
    FOREACH v_lista IN ARRAY ARRAY[
      coalesce(e.x->'resultados_omitidos','[]'::jsonb),
      coalesce(e.x->'ausencias_bloqueantes','[]'::jsonb),
      coalesce(e.x->'ausencias_admitidas','[]'::jsonb),
      coalesce(e.x->'no_habilitantes','[]'::jsonb),
      coalesce(e.x->'conflictos','[]'::jsonb)
    ] LOOP
      IF pg_catalog.jsonb_typeof(v_lista)<>'array'
         OR pg_catalog.jsonb_array_length(v_lista)>32 THEN RETURN NULL; END IF;
      FOR k IN SELECT value#>>'{}' AS clave
        FROM pg_catalog.jsonb_array_elements(v_lista)
      LOOP
        v_clave:=k.clave;
        IF v_clave IS NULL OR v_clave!~'^[a-z][a-z0-9._-]{1,79}$'
           OR v_clave=ANY(v_vistas) THEN RETURN NULL; END IF;
        v_vistas:=pg_catalog.array_append(v_vistas,v_clave);
        v_total:=v_total+1;
      END LOOP;
    END LOOP;
    IF v_total>32 THEN RETURN NULL; END IF;
  END LOOP;
  v:=vec_contratacion_temporal.o404e_texto_v1(
    v,'vec.dipgra.contratacion-temporal.'||
      'propuesta-decision-cobertura-semantica')||
    pg_catalog.int2send(1::smallint);
  FOREACH v_lista IN ARRAY ARRAY[
    pg_catalog.to_jsonb('sha-256'::text),
    p->'organizacion_ref',p->'expediente_ref'
  ] LOOP
    v:=vec_contratacion_temporal.o404e_texto_v1(v,v_lista#>>'{}');
  END LOOP;
  v:=v||pg_catalog.int8send((p->>'version_expediente')::bigint);
  FOREACH v_lista IN ARRAY ARRAY[
    p->'analisis_ref',p->'analisis_huella_sha256',
    p#>'{catalogo,referencia}'
  ] LOOP
    v:=vec_contratacion_temporal.o404e_texto_v1(v,v_lista#>>'{}');
  END LOOP;
  v:=v||pg_catalog.int8send((p#>>'{catalogo,version}')::bigint);
  FOREACH v_lista IN ARRAY ARRAY[
    p#>'{catalogo,huella_sha256}',p#>'{politica,referencia}'
  ] LOOP
    v:=vec_contratacion_temporal.o404e_texto_v1(v,v_lista#>>'{}');
  END LOOP;
  v:=v||pg_catalog.int8send((p#>>'{politica,version}')::bigint);
  FOREACH v_lista IN ARRAY ARRAY[
    p#>'{politica,huella_sha256}',p->'finalidad_clave',
    p->'finalidad_ref',p->'categoria_ref'
  ] LOOP
    v:=vec_contratacion_temporal.o404e_texto_v1(v,v_lista#>>'{}');
  END LOOP;
  v:=v||pg_catalog.int8send(
    vec_contratacion_temporal.gobi_o404b_microsegundos(
      (p#>>'{periodo,inicio}')::timestamptz))||
    pg_catalog.int8send(
    vec_contratacion_temporal.gobi_o404b_microsegundos(
      (p#>>'{periodo,fin}')::timestamptz));
  v:=vec_contratacion_temporal.o404e_texto_v1(v,p->>'estado');
  v:=vec_contratacion_temporal.o404e_texto_v1(v,p->>'via_propuesta');
  v:=v||pg_catalog.int4send(
    pg_catalog.jsonb_array_length(p->'resultados'));
  FOR r IN
    SELECT x
      FROM pg_catalog.jsonb_array_elements(p->'resultados') x
     ORDER BY x->>'clave' COLLATE "C"
  LOOP
    v:=vec_contratacion_temporal.o404e_texto_v1(v,r.x->>'clave');
    SELECT pg_catalog.jsonb_agg(resultado ORDER BY resultado COLLATE "C")
      INTO v_lista
      FROM (
        SELECT DISTINCT y->>'resultado' AS resultado
          FROM pg_catalog.jsonb_array_elements(r.x->'evidencias') y
      ) q;
    v:=v||pg_catalog.int4send(
      pg_catalog.jsonb_array_length(v_lista));
    FOR k IN SELECT value#>>'{}' AS clave
      FROM pg_catalog.jsonb_array_elements(v_lista)
    LOOP
      v:=vec_contratacion_temporal.o404e_texto_v1(v,k.clave);
    END LOOP;
  END LOOP;
  v:=v||pg_catalog.int4send(
    pg_catalog.jsonb_array_length(p->'evaluaciones'));
  FOR e IN
    SELECT x FROM pg_catalog.jsonb_array_elements(p->'evaluaciones') x
     ORDER BY (x->>'prioridad')::integer,x->>'via_clave' COLLATE "C"
  LOOP
    v:=vec_contratacion_temporal.o404e_texto_v1(v,e.x->>'via_clave')||
      pg_catalog.decode(pg_catalog.lpad(pg_catalog.to_hex(
        (e.x->>'prioridad')::integer),4,'0'),'hex');
    v:=vec_contratacion_temporal.o404e_texto_v1(v,e.x->>'estado');
    FOREACH v_lista IN ARRAY ARRAY[
      coalesce(e.x->'resultados_omitidos','[]'::jsonb),
      coalesce(e.x->'ausencias_bloqueantes','[]'::jsonb),
      coalesce(e.x->'ausencias_admitidas','[]'::jsonb),
      coalesce(e.x->'no_habilitantes','[]'::jsonb),
      coalesce(e.x->'conflictos','[]'::jsonb)
    ] LOOP
      v:=v||pg_catalog.int4send(pg_catalog.jsonb_array_length(v_lista));
      FOR k IN SELECT value#>>'{}' AS clave
        FROM pg_catalog.jsonb_array_elements(v_lista)
        ORDER BY value#>>'{}' COLLATE "C"
      LOOP
        v:=vec_contratacion_temporal.o404e_texto_v1(v,k.clave);
      END LOOP;
    END LOOP;
  END LOOP;
  RETURN pg_catalog.encode(pg_catalog.sha256(v),'hex');
EXCEPTION WHEN data_exception OR invalid_text_representation
  OR datetime_field_overflow OR numeric_value_out_of_range THEN
  RETURN NULL;
END
$funcion$;

CREATE FUNCTION
vec_contratacion_temporal.o404e_contexto_recurso_concesion_v1(
  p_carga jsonb
)
RETURNS bytea
LANGUAGE plpgsql
IMMUTABLE
STRICT
SET search_path=pg_catalog
AS $funcion$
DECLARE
  c jsonb:=p_carga->'cabecera';
  g jsonb:=p_carga->'gobierno';
  x jsonb:=p_carga->'decision_vec';
  z jsonb:=p_carga->'concesion';
  p jsonb:=z->'propuesta';
  a jsonb:=x->'atributos';
  m jsonb:=x->'ambitos';
  d jsonb;
  v_decision jsonb;
  v_semantica text;
  v_contexto bytea;
BEGIN
  d:=pg_catalog.jsonb_build_object(
    'accion_vec',x->'accion','ambitos',m,'atributos',a);
  v_contexto:=
    vec_contratacion_temporal.o404e_contexto_recurso_denegacion_v1(
      pg_catalog.jsonb_build_object('cabecera',c,'denegacion',d));
  v_semantica:=
    vec_contratacion_temporal.o404e_semantica_propuesta_v1(p);
  v_decision:=z#>ARRAY[
    'agregado_siguiente','decisiones_cobertura',
    (pg_catalog.jsonb_array_length(
      z#>'{agregado_siguiente,decisiones_cobertura}')-1)::text];
  IF v_contexto IS NULL OR v_semantica IS NULL
     OR m->>'unidad_ejecutora_ref' IS DISTINCT FROM
        g->>'unidad_ejecutora_ref'
     OR a->>'propuesta_ref' IS DISTINCT FROM p->>'referencia'
     OR a->>'propuesta_huella_sha256' IS DISTINCT FROM p->>'huella_sha256'
     OR a->>'propuesta_semantica_ref' IS DISTINCT FROM
        'propuesta-cobertura-semantica:sha256:'||v_semantica
     OR a->>'propuesta_semantica_huella_sha256' IS DISTINCT FROM
        v_semantica
     OR a->>'preparacion_evidencias_ref' IS DISTINCT FROM
        p->>'preparacion_evidencias_ref'
     OR a->>'preparacion_evidencias_huella_sha256' IS DISTINCT FROM
        p->>'preparacion_evidencias_huella_sha256'
     OR a->>'analisis_ref' IS DISTINCT FROM p->>'analisis_ref'
     OR a->>'analisis_huella_sha256' IS DISTINCT FROM
        p->>'analisis_huella_sha256'
     OR a->>'catalogo_ref' IS DISTINCT FROM p#>>'{catalogo,referencia}'
     OR a->>'catalogo_version' IS DISTINCT FROM p#>>'{catalogo,version}'
     OR a->>'catalogo_huella_sha256' IS DISTINCT FROM
        p#>>'{catalogo,huella_sha256}'
     OR a->>'politica_ref' IS DISTINCT FROM p#>>'{politica,referencia}'
     OR a->>'politica_version' IS DISTINCT FROM p#>>'{politica,version}'
     OR a->>'politica_huella_sha256' IS DISTINCT FROM
        p#>>'{politica,huella_sha256}'
     OR a->>'politica_actuacion_ref' IS DISTINCT FROM
        g#>>'{politica_actuacion,referencia}'
     OR a->>'politica_actuacion_version' IS DISTINCT FROM
        g#>>'{politica_actuacion,version}'
     OR a->>'politica_actuacion_huella_sha256' IS DISTINCT FROM
        g#>>'{politica_actuacion,huella_sha256}'
     OR a->>'tipo_operacion' IS DISTINCT FROM v_decision->>'tipo'
     OR a->>'accion' IS DISTINCT FROM
        v_decision#>>'{actuacion,accion_clave}'
     OR a->>'via_elegida' IS DISTINCT FROM v_decision->>'via_elegida'
     OR (a->>'predecesora_ref') IS DISTINCT FROM
        nullif(v_decision->>'predecesora_ref','')
     OR (a->>'predecesora_huella_sha256') IS DISTINCT FROM
        nullif(v_decision->>'predecesora_huella_sha256','') THEN
    RETURN NULL;
  END IF;
  RETURN v_contexto;
END
$funcion$;

UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
   SET version_esquema=12,
       actualizada_en=
         pg_catalog.date_trunc('microseconds',pg_catalog.clock_timestamp())
 WHERE control AND version_esquema=11;
REVOKE ALL ON FUNCTION
  vec_contratacion_temporal.o404e_revalidar_gobierno_actual_v1(
    text,text,text,numeric,text,text,numeric,text,text,numeric,text),
  vec_contratacion_temporal.o404e_transicion_dominio_ligada_v1(
    jsonb,jsonb,jsonb,jsonb,timestamptz),
  vec_contratacion_temporal.o404e_concesion_ligada_v1(jsonb,jsonb),
  vec_contratacion_temporal.o404e_semantica_propuesta_v1(jsonb),
  vec_contratacion_temporal.o404e_contexto_recurso_concesion_v1(jsonb)
FROM PUBLIC,vec_contratacion_temporal_ejecutor,
  vec_contratacion_temporal_confirmador_cobertura,
  vec_contratacion_temporal_migrador,
  vec_contratacion_temporal_gobernador;
COMMIT;
