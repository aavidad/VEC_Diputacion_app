\set ON_ERROR_STOP on
-- D5: otros medios de transporte y otros gastos con justificante.
-- 1) Catálogo versionado de tipos, rotulado provisional hasta que RRHH
--    confirme tipos admitidos y límites (dudas.md). Solo adición.
-- 2) validar_documento_v2 (misma firma, dueño y ACL) exige en cada línea de
--    otros medios o gastos que se guarde: tipo del catálogo coherente con su
--    apartado, fecha del gasto dentro de la comisión y justificante por
--    referencia y huella SHA-256. VEC no recibe el documento: queda en
--    custodia de la persona. Las líneas guardadas antes siguen intactas; la
--    validación solo se aplica al editar.
BEGIN;
SET LOCAL ROLE vec_dietas_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_dietas:migracion:000009:otros-gastos:v1',0));
DO $pre$
BEGIN
 IF current_user<>'vec_dietas_propietario'
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_dietas_ejecutor')
    OR to_regclass('vec_dietas.comision_revision') IS NULL
    OR to_regprocedure('vec_dietas.rechazar_mutacion_borrador_v1()') IS NULL
    OR to_regprocedure('vec_dietas.consultar_documento_circuito_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regclass('vec_dietas.catalogo_otros_gastos_provisional') IS NOT NULL
    OR to_regclass('vec_dietas.tipo_otro_gasto_provisional') IS NOT NULL
    OR NOT EXISTS (SELECT 1 FROM pg_proc p JOIN pg_roles r ON r.oid=p.proowner
        WHERE p.oid=to_regprocedure('vec_dietas.validar_documento_v2(jsonb)')
          AND r.rolname='vec_dietas_propietario' AND p.prosecdef
          AND md5(p.prosrc)='bda3d5b21490b858869b1ab99043a9eb')
    OR has_function_privilege('vec_dietas_ejecutor','vec_dietas.validar_documento_v2(jsonb)','EXECUTE')
 THEN RAISE EXCEPTION 'Dietas 000009: falta 000008, preimagen alterada o ya instalada' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE TABLE vec_dietas.catalogo_otros_gastos_provisional (
 version_ref text PRIMARY KEY CHECK(version_ref~'^provisional:otros-gastos:[0-9]{8}$'),
 rotulo text NOT NULL CHECK(rotulo='PROVISIONAL · pendiente de confirmación por RRHH'),
 publicada_en timestamptz(6) NOT NULL
);
CREATE TABLE vec_dietas.tipo_otro_gasto_provisional (
 version_ref text NOT NULL REFERENCES vec_dietas.catalogo_otros_gastos_provisional(version_ref),
 codigo text NOT NULL CHECK(codigo~'^[a-z][a-z_]{1,40}$'),
 clase text NOT NULL CHECK(clase IN ('otro_medio','otro_gasto')),
 PRIMARY KEY(version_ref,codigo)
);
INSERT INTO vec_dietas.catalogo_otros_gastos_provisional VALUES
 ('provisional:otros-gastos:20260925','PROVISIONAL · pendiente de confirmación por RRHH',clock_timestamp());
-- El orden y los códigos coinciden con domain.CatalogoOtrosGastosVigente.
INSERT INTO vec_dietas.tipo_otro_gasto_provisional(version_ref,codigo,clase) VALUES
 ('provisional:otros-gastos:20260925','tren','otro_medio'),
 ('provisional:otros-gastos:20260925','autobus','otro_medio'),
 ('provisional:otros-gastos:20260925','metro_tranvia','otro_medio'),
 ('provisional:otros-gastos:20260925','taxi','otro_medio'),
 ('provisional:otros-gastos:20260925','avion','otro_medio'),
 ('provisional:otros-gastos:20260925','barco','otro_medio'),
 ('provisional:otros-gastos:20260925','vehiculo_alquiler','otro_medio'),
 ('provisional:otros-gastos:20260925','aparcamiento','otro_gasto'),
 ('provisional:otros-gastos:20260925','peaje','otro_gasto'),
 ('provisional:otros-gastos:20260925','consigna','otro_gasto');

DO $politicas$ DECLARE tabla text;
BEGIN
 FOREACH tabla IN ARRAY ARRAY['catalogo_otros_gastos_provisional','tipo_otro_gasto_provisional'] LOOP
  EXECUTE format('ALTER TABLE vec_dietas.%I ENABLE ROW LEVEL SECURITY',tabla);
  EXECUTE format('ALTER TABLE vec_dietas.%I FORCE ROW LEVEL SECURITY',tabla);
  EXECUTE format('CREATE POLICY propietario_catalogo ON vec_dietas.%I FOR ALL TO vec_dietas_propietario USING (true) WITH CHECK (true)',tabla);
  EXECUTE format('CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_dietas.%I FOR EACH ROW EXECUTE FUNCTION vec_dietas.rechazar_mutacion_borrador_v1()',tabla);
  EXECUTE format('CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_dietas.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_dietas.rechazar_mutacion_borrador_v1()',tabla);
  EXECUTE format('REVOKE ALL ON vec_dietas.%I FROM PUBLIC,vec_dietas_ejecutor',tabla);
 END LOOP;
END $politicas$;
COMMENT ON TABLE vec_dietas.catalogo_otros_gastos_provisional IS 'D5: versiones provisionales del catálogo de otros medios y gastos; pendientes de RRHH.';
COMMENT ON TABLE vec_dietas.tipo_otro_gasto_provisional IS 'D5: tipos admitidos por versión y apartado del documento; sin límites aprobados.';

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
  -- D5: tipo del catálogo versionado coherente con el apartado, fecha del
  -- gasto dentro de la comisión y justificante obligatorio (referencia y
  -- huella SHA-256; el documento queda en custodia de la persona).
  IF otros>32 OR linea->>'tipo' NOT IN ('otro_medio','otro_gasto')
     OR jsonb_typeof(linea->'importe_centimos') IS DISTINCT FROM 'number'
     OR length(coalesce(linea->>'concepto','')) NOT BETWEEN 3 AND 500
     OR linea->>'concepto'<>btrim(linea->>'concepto')
     OR linea->>'concepto'~'[[:cntrl:]]'
     OR coalesce(linea->>'justificante_ref','') !~ '^[A-Za-z][A-Za-z0-9:_-]{2,127}$'
     OR coalesce(linea->>'justificante_sha256','') !~ '^[0-9a-f]{64}$'
     OR coalesce(linea->>'fecha','') !~ '^20[0-9]{2}-[0-9]{2}-[0-9]{2}$'
     OR (linea->>'fecha')::date NOT BETWEEN fecha_inicio AND fecha_fin
     OR NOT EXISTS (SELECT 1 FROM vec_dietas.tipo_otro_gasto_provisional t
         WHERE t.version_ref=linea->>'catalogo_version'
           AND t.codigo=linea->>'tipo_gasto' AND t.clase=linea->>'tipo')
     OR (linea->>'importe_centimos')::bigint NOT BETWEEN 1 AND 100000000
     OR linea IS DISTINCT FROM jsonb_build_object('tipo',linea->>'tipo',
       'tipo_gasto',linea->>'tipo_gasto','catalogo_version',linea->>'catalogo_version',
       'fecha',linea->>'fecha','concepto',linea->>'concepto',
       'importe_centimos',(linea->>'importe_centimos')::bigint,
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

DO $post$
BEGIN
 IF (SELECT r.rolname FROM pg_proc p JOIN pg_roles r ON r.oid=p.proowner
     WHERE p.oid='vec_dietas.validar_documento_v2(jsonb)'::regprocedure)<>'vec_dietas_propietario'
    OR has_function_privilege('vec_dietas_ejecutor','vec_dietas.validar_documento_v2(jsonb)','EXECUTE')
    OR (SELECT count(*) FROM vec_dietas.tipo_otro_gasto_provisional)<>10
 THEN RAISE EXCEPTION 'Dietas 000009: postcondición incumplida' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
