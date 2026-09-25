\set ON_ERROR_STOP on
-- Sólo para PostgreSQL desechable sin líneas D5 guardadas. Nunca en una base
-- con historia: una revisión que use el catálogo impide el DOWN.
BEGIN;
RESET ROLE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_dietas:migracion:000009:otros-gastos:v1',0));
DO $preimagen_dba$
BEGIN
 IF current_user IS DISTINCT FROM session_user
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
    OR to_regclass('vec_dietas.catalogo_otros_gastos_provisional') IS NULL
    OR to_regclass('vec_dietas.tipo_otro_gasto_provisional') IS NULL
    OR NOT EXISTS (SELECT 1 FROM pg_proc p
        WHERE p.oid=to_regprocedure('vec_dietas.validar_documento_v2(jsonb)')
          AND md5(p.prosrc)='d5615a8e6035946f8bf72cfeec3c2f17')
 THEN RAISE EXCEPTION 'Dietas 000009 DOWN requiere DBA y preimagen completa' USING ERRCODE='55000'; END IF;
END $preimagen_dba$;
-- El DBA ve todas las filas pese a FORCE RLS contextual.
LOCK TABLE vec_dietas.comision_revision IN ACCESS EXCLUSIVE MODE;
DO $historia_dba$
BEGIN
 IF EXISTS (SELECT 1 FROM vec_dietas.comision_revision r,
            jsonb_array_elements(coalesce(r.documento->'lineas','[]'::jsonb)) l(v)
            WHERE l.v ? 'catalogo_version')
 THEN RAISE EXCEPTION 'Dietas 000009 DOWN protege líneas D5 guardadas' USING ERRCODE='55000'; END IF;
END $historia_dba$;
SET LOCAL ROLE vec_dietas_propietario;
CREATE OR REPLACE FUNCTION vec_dietas.validar_documento_v2(p_comando jsonb) RETURNS boolean
LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path=pg_catalog AS $$
DECLARE calc jsonb; doc jsonb; linea jsonb; tramo jsonb; opcion jsonb; esperado jsonb:='[]'::jsonb;
        rutas_d4 jsonb; seleccion jsonb; regla_catalogo jsonb; regla_config jsonb;
        v_version_tarifa text; rotulo text; tarifa numeric; pct int; porcentajes int[];
        man numeric; alo numeric;
        importe bigint; suma_man bigint; suma_alo bigint; seleccionado_man bigint:=0;
        seleccionado_alo bigint:=0; km_centimos bigint:=0; otros_centimos bigint:=0;
        n int; grupo int; grupo_actual int; indice int; anterior_indice int:=-1; n_tramos int;
        otros int:=0; base_lineas int; fecha_inicio date; fecha_fin date;
BEGIN
 IF jsonb_typeof(p_comando)<>'object' OR jsonb_typeof(p_comando->'calculo')<>'object'
    OR jsonb_typeof(p_comando->'documento')<>'object' THEN RETURN false; END IF;
 calc:=p_comando->'calculo'; doc:=p_comando->'documento';
 IF jsonb_typeof(doc->'lineas') IS DISTINCT FROM 'array'
    OR doc->'vehiculo_propio' IS DISTINCT FROM p_comando->'vehiculo_propio'
    OR jsonb_array_length(doc->'lineas')>256
    OR jsonb_typeof(p_comando->'asignacion') IS DISTINCT FROM 'object'
    OR jsonb_typeof(p_comando->'tramos_aceptados') IS DISTINCT FROM 'array'
    OR p_comando->>'version_tarifa_aceptada' IS DISTINCT FROM calc->>'version_tarifa'
    OR doc->>'version_tarifa_aceptada' IS DISTINCT FROM calc->>'version_tarifa'
    OR doc->'tramos_aceptados' IS DISTINCT FROM p_comando->'tramos_aceptados'
    OR doc->>'grupo_dieta' IS DISTINCT FROM p_comando->'asignacion'->>'grupo_dieta'
    OR jsonb_typeof(calc->'opciones_dieta') IS DISTINCT FROM 'array'
    OR jsonb_array_length(calc->'opciones_dieta')<>3
    OR jsonb_typeof(p_comando->'codigos_ruta') IS DISTINCT FROM 'array'
    OR jsonb_array_length(p_comando->'codigos_ruta') NOT BETWEEN 2 AND 12
    OR EXISTS (SELECT 1 FROM jsonb_array_elements(p_comando->'codigos_ruta') x(valor)
        WHERE jsonb_typeof(x.valor) IS DISTINCT FROM 'string'
           OR x.valor #>> '{}' !~ '^[A-Za-z0-9:_-]{1,64}$')
    OR EXISTS (SELECT 1 FROM jsonb_array_elements_text(p_comando->'codigos_ruta') x(valor)
        GROUP BY x.valor HAVING count(*)>1)
    OR (p_comando->>'vehiculo_propio'='true' AND
        (calc->>'procedencia' IS DISTINCT FROM 'osrm_interno'
         OR calc->>'motor' IS DISTINCT FROM 'OSRM'
         OR length(coalesce(calc->>'version_grafo','')) NOT BETWEEN 1 AND 160))
    OR (p_comando->>'vehiculo_propio'='false' AND
        (calc->>'procedencia' IS DISTINCT FROM 'sin_vehiculo_propio'
         OR calc->>'motor' IS DISTINCT FROM 'no_aplica'
         OR calc->>'version_grafo' IS DISTINCT FROM 'no_aplica'))
    OR calc->>'rotulo'<>'PROVISIONAL · pendiente de confirmación por RRHH'
    OR calc->>'version_tarifa' !~ '^provisional:[a-z0-9:-]{8,120}$'
    OR calc->>'hora_inicio' IS DISTINCT FROM p_comando->>'hora_inicio'
    OR calc->>'hora_fin' IS DISTINCT FROM p_comando->>'hora_fin'
    OR calc->>'kilometros' !~ '^(0|[1-9][0-9]{0,4})\.[0-9]{4}$'
    OR calc->>'eur_por_km' !~ '^0\.[0-9]{4}$' THEN RETURN false; END IF;
 fecha_inicio:=(p_comando->>'fecha_inicio')::date;
 fecha_fin:=(p_comando->>'fecha_fin')::date;
 v_version_tarifa:=calc->>'version_tarifa'; rotulo:=calc->>'rotulo';
 regla_catalogo:=vec_dietas.consultar_regla_devengo_dietas_v1(
  v_version_tarifa,fecha_inicio,'ES','nacional_ordinaria');
 regla_config:=regla_catalogo->'configuracion';
 IF calc->>'regla_ref' IS DISTINCT FROM regla_catalogo->>'regla_ref'
    OR calc->>'regla_huella_sha256' IS DISTINCT FROM regla_catalogo->>'huella_sha256' THEN RETURN false; END IF;
 porcentajes:=ARRAY[(regla_config->>'porcentaje_mismo_dia')::int,
  (regla_config->>'porcentaje_salida_temprana')::int,
  (regla_config->>'porcentaje_salida_media')::int,
  (regla_config->>'porcentaje_regreso')::int,
  (regla_config->>'porcentaje_intermedio')::int];
 SELECT k.eur_por_km INTO STRICT tarifa FROM vec_dietas.importe_km_provisional k
 JOIN vec_dietas.version_tarifa_provisional v ON v.version_ref=k.version_ref
 WHERE k.version_ref=v_version_tarifa AND k.vehiculo='automovil'
   AND v.vigente_desde<=fecha_inicio AND (v.vigente_hasta IS NULL OR fecha_fin<v.vigente_hasta);
 IF tarifa IS DISTINCT FROM (calc->>'eur_por_km')::numeric THEN RETURN false; END IF;
 FOR grupo_actual IN 1..3 LOOP
  opcion:=calc->'opciones_dieta'->(grupo_actual-1);
  SELECT d.manutencion_eur,d.alojamiento_eur INTO STRICT man,alo
  FROM vec_dietas.importe_dieta_provisional d
  WHERE d.version_ref=v_version_tarifa AND d.pais_iso2='ES' AND d.grupo=grupo_actual;
  IF (opcion->>'grupo')::int IS DISTINCT FROM grupo_actual
     OR opcion->'calculo'->>'version_tarifa_ref' IS DISTINCT FROM v_version_tarifa
     OR opcion->'calculo'->>'rotulo' IS DISTINCT FROM rotulo
     OR jsonb_typeof(opcion->'calculo'->'tramos')<>'array'
     OR jsonb_array_length(opcion->'calculo'->'tramos')>62 THEN RETURN false; END IF;
  suma_man:=0; suma_alo:=0;
  FOR tramo IN SELECT value FROM jsonb_array_elements(opcion->'calculo'->'tramos') LOOP
   IF tramo->>'version_tarifa_ref' IS DISTINCT FROM v_version_tarifa
      OR tramo->>'rotulo' IS DISTINCT FROM rotulo
      OR tramo->>'fecha' !~ '^20[0-9]{2}-[0-9]{2}-[0-9]{2}$'
      OR (tramo->>'fecha')::date NOT BETWEEN fecha_inicio AND fecha_fin THEN RETURN false; END IF;
   importe:=(tramo->>'importe_centimos')::bigint;
   pct:=(tramo->>'porcentaje')::int;
   IF tramo->>'tipo'='manutencion' AND pct=ANY(porcentajes)
      AND importe=ceil(man*100*pct/100)::bigint
      THEN suma_man:=suma_man+importe;
   ELSIF tramo->>'tipo'='alojamiento_tope_pendiente_justificante'
      AND pct=(regla_config->>'porcentaje_alojamiento_tope')::int
      AND importe=ceil(alo*100*pct/100)::bigint
      THEN suma_alo:=suma_alo+importe;
   ELSE RETURN false; END IF;
  END LOOP;
  IF suma_man IS DISTINCT FROM (opcion->'calculo'->>'manutencion_centimos')::bigint
     OR suma_alo IS DISTINCT FROM (opcion->'calculo'->>'alojamiento_tope_centimos')::bigint
     OR suma_man+suma_alo IS DISTINCT FROM (opcion->'calculo'->>'total_maximo_orientativo_centimos')::bigint
  THEN RETURN false; END IF;
 END LOOP;
 grupo:=(p_comando->'asignacion'->>'grupo_dieta')::int;
 IF grupo NOT BETWEEN 1 AND 3 THEN RETURN false; END IF;
 seleccion:=p_comando->'tramos_aceptados';
 opcion:=calc->'opciones_dieta'->(grupo-1); n_tramos:=jsonb_array_length(opcion->'calculo'->'tramos');
 IF (n_tramos=0 AND jsonb_array_length(seleccion)<>0)
    OR (n_tramos>0 AND jsonb_array_length(seleccion) NOT IN (1,n_tramos)) THEN RETURN false; END IF;
 FOR linea IN SELECT value FROM jsonb_array_elements(seleccion) LOOP
  IF jsonb_typeof(linea) IS DISTINCT FROM 'number'
     OR linea::text !~ '^(0|[1-9][0-9]*)$' THEN RETURN false; END IF;
  indice:=(linea::text)::int;
  IF indice<=anterior_indice OR indice>=n_tramos THEN RETURN false; END IF;
  anterior_indice:=indice;
  tramo:=opcion->'calculo'->'tramos'->indice;
  IF tramo->>'tipo'='manutencion' THEN seleccionado_man:=seleccionado_man+(tramo->>'importe_centimos')::bigint;
  ELSIF tramo->>'tipo'='alojamiento_tope_pendiente_justificante' THEN seleccionado_alo:=seleccionado_alo+(tramo->>'importe_centimos')::bigint;
  ELSE RETURN false; END IF;
  esperado:=esperado||jsonb_build_array(jsonb_build_object(
   'tipo','dieta','grupo',grupo,'indice_tramo',indice,
   'fecha',tramo->>'fecha','concepto',tramo->>'tipo',
   'importe_centimos',(tramo->>'importe_centimos')::bigint,
   'version_tarifa_ref',v_version_tarifa,'rotulo',rotulo));
 END LOOP;
 IF n_tramos>0 AND jsonb_array_length(seleccion)=n_tramos
    AND (SELECT jsonb_agg(to_jsonb(s.n) ORDER BY s.n) FROM generate_series(0,n_tramos-1) s(n))
        IS DISTINCT FROM seleccion THEN RETURN false; END IF;
 rutas_d4:=vec_dietas.validar_rutas_d4_v2(p_comando,tarifa,v_version_tarifa,rotulo);
 IF rutas_d4 IS NULL THEN RETURN false; END IF;
 esperado:=esperado||(rutas_d4->'lineas');
 km_centimos:=(rutas_d4->>'importe_centimos')::bigint;
 base_lineas:=jsonb_array_length(esperado);
 IF coalesce(jsonb_typeof(p_comando->'otros'),'array')<>'array'
    OR (p_comando ? 'otros' AND jsonb_array_length(p_comando->'otros')>32) THEN RETURN false; END IF;
 FOR linea IN SELECT x.value FROM jsonb_array_elements(doc->'lineas') WITH ORDINALITY x(value,ord)
   WHERE x.ord>base_lineas LOOP
  otros:=otros+1;
  IF otros>32 OR linea->>'tipo' NOT IN ('otro_medio','otro_gasto')
     OR linea->>'justificante_ref' IS NULL
     OR linea->>'justificante_sha256' IS NULL
     OR jsonb_typeof(linea->'importe_centimos') IS DISTINCT FROM 'number'
     OR length(coalesce(linea->>'concepto','')) NOT BETWEEN 3 AND 500
     OR linea->>'concepto'<>btrim(linea->>'concepto')
     OR linea->>'concepto'~'[[:cntrl:]]'
     OR NOT (
       (linea->>'justificante_ref'='' AND linea->>'justificante_sha256'='')
       OR (linea->>'justificante_ref' ~ '^[A-Za-z][A-Za-z0-9:_-]{2,127}$'
           AND linea->>'justificante_sha256' ~ '^[0-9a-f]{64}$'))
     OR (linea->>'importe_centimos')::bigint NOT BETWEEN 1 AND 100000000
     OR linea IS DISTINCT FROM jsonb_build_object('tipo',linea->>'tipo',
       'concepto',linea->>'concepto','importe_centimos',(linea->>'importe_centimos')::bigint,
       'justificante_ref',linea->>'justificante_ref',
       'justificante_sha256',linea->>'justificante_sha256') THEN RETURN false; END IF;
  IF linea IS DISTINCT FROM p_comando->'otros'->(otros-1) THEN RETURN false; END IF;
  otros_centimos:=otros_centimos+(linea->>'importe_centimos')::bigint;
  esperado:=esperado||jsonb_build_array(linea);
 END LOOP;
 IF otros IS DISTINCT FROM coalesce(jsonb_array_length(p_comando->'otros'),0) THEN RETURN false; END IF;
 RETURN doc->'lineas' IS NOT DISTINCT FROM esperado
   AND (doc->>'manutencion_centimos')::bigint IS NOT DISTINCT FROM seleccionado_man
   AND (doc->>'alojamiento_tope_centimos')::bigint IS NOT DISTINCT FROM seleccionado_alo
   AND (doc->>'kilometraje_centimos')::bigint IS NOT DISTINCT FROM km_centimos
   AND (doc->>'otros_centimos')::bigint IS NOT DISTINCT FROM otros_centimos
   AND (doc->>'total_orientativo_centimos')::bigint IS NOT DISTINCT FROM
     seleccionado_man+seleccionado_alo+km_centimos+otros_centimos;
EXCEPTION WHEN others THEN RETURN false;
END $$;

DROP TABLE vec_dietas.tipo_otro_gasto_provisional;
DROP TABLE vec_dietas.catalogo_otros_gastos_provisional;
COMMIT;
