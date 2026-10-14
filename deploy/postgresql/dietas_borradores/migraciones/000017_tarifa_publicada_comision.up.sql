\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_dietas_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_dietas:migracion:000017',0));

-- El cuerpo instalado se toma de la copia de la principal, no de migraciones
-- antiguas. D13 crea el registro relacional y las publicaciones inmutables.
DO $pre$ BEGIN
 IF current_setting('server_version_num')::int<180000
    OR to_regclass('vec_dietas.catalogo_liquidacion_publicado') IS NULL
    OR to_regclass('vec_dietas.version_tarifa_registrada') IS NULL
    OR to_regclass('vec_dietas.regla_devengo_registrada') IS NULL
    OR to_regprocedure('vec_dietas.calcular_tramos_publicados_v1(date,text,date,text,bigint,text,text,text,jsonb)') IS NULL
    OR to_regprocedure('vec_dietas.consultar_tarifa_publicada_v1(text,date,smallint,text)') IS NULL
    OR encode(sha256(convert_to(pg_get_functiondef('vec_dietas.validar_documento_v2(jsonb)'::regprocedure),'UTF8')),'hex')
       <> '5b2ff05d41e7d6c3faf06d079b34ecf870cd6ce05cafd5da8441181d1cea1f1e'
    OR encode(sha256(convert_to(pg_get_functiondef('vec_dietas.validar_calculo_catalogado_v2(jsonb)'::regprocedure),'UTF8')),'hex')
       <> '98d07dc11aca629c2337157b22941e6ddbf6409ea59c3c3b320fefae95353c85'
    OR encode(sha256(convert_to(pg_get_functiondef('vec_dietas.consultar_regla_devengo_dietas_v1(text,date,text,text)'::regprocedure),'UTF8')),'hex')
       <> 'cd07f30fb57df0abf1616e570f3a90884efb183011ea4fd7663925275ba123f2'
    OR encode(sha256(convert_to(pg_get_functiondef('vec_dietas.crear_o_recuperar_comision_catalogada_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure),'UTF8')),'hex')
       <> '1aaa25a5d3b3e9880f43c4bd2c41b986e85e0894d0e9ad7bc3ed1ce8bc843b7d'
    OR encode(sha256(convert_to(pg_get_functiondef('vec_dietas.mutar_comision_propia_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure),'UTF8')),'hex')
       <> '8b5ce8d2b560b8cbf3d0c585954e2c152336e97a86dbe5ed6d332ea49f2b2915'
    OR EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid IN (
       'vec_dietas.consultar_regla_devengo_dietas_v1(text,date,text,text)'::regprocedure,
       'vec_dietas.crear_o_recuperar_comision_catalogada_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
       'vec_dietas.mutar_comision_propia_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure)
       AND (p.proowner<>'vec_dietas_propietario'::regrole OR p.prosecdef IS NOT TRUE
        OR p.proacl::text IS DISTINCT FROM
           '{vec_dietas_propietario=X/vec_dietas_propietario,vec_dietas_ejecutor=X/vec_dietas_propietario}'
        OR p.proconfig IS DISTINCT FROM
           ARRAY['search_path=pg_catalog','row_security=on','TimeZone=UTC','lock_timeout=2s']::text[]))
    OR EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid IN (
       'vec_dietas.validar_calculo_catalogado_v2(jsonb)'::regprocedure,
       'vec_dietas.validar_documento_v2(jsonb)'::regprocedure)
       AND (p.proowner<>'vec_dietas_propietario'::regrole OR p.prosecdef IS NOT TRUE
        OR p.proacl::text IS DISTINCT FROM
           '{vec_dietas_propietario=X/vec_dietas_propietario}'
        OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog']::text[]))
    OR NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='vec_dietas.calculo_comision'::regclass
         AND conname='calculo_comision_version_tarifa_fkey' AND pg_get_constraintdef(oid)=
          'FOREIGN KEY (version_tarifa) REFERENCES vec_dietas.version_tarifa_provisional(version_ref)')
    OR NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='vec_dietas.comision_revision'::regclass
         AND conname='comision_revision_regla_ref_fkey' AND pg_get_constraintdef(oid)=
          'FOREIGN KEY (regla_ref) REFERENCES vec_dietas.regla_devengo_provisional(regla_ref)')
    OR NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='vec_dietas.recibo_operacion_comision'::regclass
         AND conname='recibo_operacion_comision_regla_ref_fkey' AND pg_get_constraintdef(oid)=
          'FOREIGN KEY (regla_ref) REFERENCES vec_dietas.regla_devengo_provisional(regla_ref)')
    OR NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='vec_dietas.recibo_regla_creacion_comision'::regclass
         AND conname='recibo_regla_creacion_comision_regla_ref_fkey' AND pg_get_constraintdef(oid)=
          'FOREIGN KEY (regla_ref) REFERENCES vec_dietas.regla_devengo_provisional(regla_ref)')
 THEN RAISE EXCEPTION 'Dietas 000017: PARO preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

-- El cambio de FK no sustituye la comprobación del origen. Cada fila del
-- registro común conserva FK a la tabla provisional o al catálogo publicado.
ALTER TABLE vec_dietas.calculo_comision DROP CONSTRAINT calculo_comision_version_tarifa_fkey;
ALTER TABLE vec_dietas.calculo_comision ADD CONSTRAINT calculo_comision_version_tarifa_fkey
 FOREIGN KEY(version_tarifa) REFERENCES vec_dietas.version_tarifa_registrada(version_ref);
ALTER TABLE vec_dietas.comision_revision DROP CONSTRAINT comision_revision_regla_ref_fkey;
ALTER TABLE vec_dietas.comision_revision ADD CONSTRAINT comision_revision_regla_ref_fkey
 FOREIGN KEY(regla_ref) REFERENCES vec_dietas.regla_devengo_registrada(regla_ref);
ALTER TABLE vec_dietas.recibo_operacion_comision DROP CONSTRAINT recibo_operacion_comision_regla_ref_fkey;
ALTER TABLE vec_dietas.recibo_operacion_comision ADD CONSTRAINT recibo_operacion_comision_regla_ref_fkey
 FOREIGN KEY(regla_ref) REFERENCES vec_dietas.regla_devengo_registrada(regla_ref);
ALTER TABLE vec_dietas.recibo_regla_creacion_comision DROP CONSTRAINT recibo_regla_creacion_comision_regla_ref_fkey;
ALTER TABLE vec_dietas.recibo_regla_creacion_comision ADD CONSTRAINT recibo_regla_creacion_comision_regla_ref_fkey
 FOREIGN KEY(regla_ref) REFERENCES vec_dietas.regla_devengo_registrada(regla_ref);



CREATE FUNCTION vec_dietas.validar_calculo_publicado_v1(p_comando jsonb)
RETURNS boolean LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
AS $calculo$
DECLARE calc jsonb; catalogo vec_dietas.catalogo_liquidacion_publicado%ROWTYPE;
 regla jsonb; item jsonb; tr jsonb; opcion jsonb; esperado jsonb;
 fecha_inicio date; fecha_fin date; v_version_tarifa text; v_vehiculo text;
 km numeric:=0; tarifa numeric; importe_km bigint; base bigint;
 v_grupo integer; n integer; cantidad integer; referencia text; huella text;
 publicacion text; normas jsonb;
BEGIN
 IF jsonb_typeof(p_comando) IS DISTINCT FROM 'object'
    OR jsonb_typeof(p_comando->'calculo') IS DISTINCT FROM 'object'
    OR jsonb_typeof(p_comando->'codigos_ruta') IS DISTINCT FROM 'array'
    OR jsonb_array_length(p_comando->'codigos_ruta') NOT BETWEEN 2 AND 12
    OR p_comando->>'fecha_inicio' !~ '^20[0-9]{2}-[0-9]{2}-[0-9]{2}$'
    OR p_comando->>'fecha_fin' !~ '^20[0-9]{2}-[0-9]{2}-[0-9]{2}$'
 THEN RETURN false; END IF;
 calc:=p_comando->'calculo';
 fecha_inicio:=(p_comando->>'fecha_inicio')::date;
 fecha_fin:=(p_comando->>'fecha_fin')::date;
 v_version_tarifa:=calc->>'version_tarifa';
 v_vehiculo:=CASE WHEN p_comando->>'vehiculo_propio'='false' THEN 'sin_vehiculo_propio' ELSE 'propio' END;
 IF fecha_fin<fecha_inicio OR fecha_fin-fecha_inicio>30
    OR v_version_tarifa !~ '^[A-Za-z][A-Za-z0-9:_-]{2,127}$'
    OR v_version_tarifa LIKE 'provisional:%'
    OR calc->>'rotulo' IS DISTINCT FROM 'dietas_tarifa_publicada'
    OR calc->>'hora_inicio' IS DISTINCT FROM p_comando->>'hora_inicio'
    OR calc->>'hora_fin' IS DISTINCT FROM p_comando->>'hora_fin'
    OR calc->>'kilometros' !~ '^(0|[1-9][0-9]{0,4})\.[0-9]{4}$'
    OR calc->>'eur_por_km' !~ '^0\.[0-9]{4}$'
    OR jsonb_typeof(calc->'opciones_dieta') IS DISTINCT FROM 'array'
    OR jsonb_array_length(calc->'opciones_dieta')<>3
    OR jsonb_typeof(calc->'tramos_ruta') IS DISTINCT FROM 'array'
    OR NOT EXISTS (SELECT 1 FROM vec_dietas.version_tarifa_registrada v
      WHERE v.version_ref=v_version_tarifa AND v.origen='publicada')
    OR NOT EXISTS (SELECT 1 FROM vec_dietas.regla_devengo_registrada r
      WHERE r.regla_ref=calc->>'regla_ref' AND r.version_ref=v_version_tarifa
        AND r.huella_sha256=calc->>'regla_huella_sha256' AND r.origen='publicada')
 THEN RETURN false; END IF;
 IF v_vehiculo='propio' THEN
  IF calc->>'procedencia' IS DISTINCT FROM 'osrm_interno'
     OR calc->>'motor' IS DISTINCT FROM 'OSRM'
     OR length(coalesce(calc->>'version_grafo','')) NOT BETWEEN 1 AND 160
  THEN RETURN false; END IF;
 ELSE
  IF calc->>'procedencia' IS DISTINCT FROM 'sin_vehiculo_propio'
     OR calc->>'motor' IS DISTINCT FROM 'no_aplica'
     OR calc->>'version_grafo' IS DISTINCT FROM 'no_aplica'
     OR calc->>'kilometros' IS DISTINCT FROM '0.0000'
     OR calc->>'eur_por_km' IS DISTINCT FROM '0.0000'
     OR calc->>'importe_kilometraje_centimos' IS DISTINCT FROM '0'
     OR jsonb_array_length(calc->'tramos_ruta')<>0
  THEN RETURN false; END IF;
 END IF;
 referencia:=calc->>'regla_ref'; huella:=calc->>'regla_huella_sha256';
 FOR v_grupo IN 1..3 LOOP
  SELECT * INTO STRICT catalogo FROM vec_dietas.catalogo_liquidacion_publicado x
   WHERE x.version_tarifa_ref=v_version_tarifa AND x.pais_iso2='ES'
     AND x.grupo=v_grupo AND x.vehiculo=v_vehiculo
     AND x.fecha_publicacion<=current_date
     AND fecha_inicio>=x.vigente_desde AND fecha_fin<=x.vigente_hasta;
  IF catalogo.regla_ref IS DISTINCT FROM referencia
     OR catalogo.regla_huella_sha256 IS DISTINCT FROM huella
     OR catalogo.catalogo->>'rotulo_clave' IS DISTINCT FROM 'dietas_tarifa_publicada'
     OR catalogo.catalogo->'regla_devengo'->>'referencia' IS DISTINCT FROM referencia
     OR catalogo.catalogo->'regla_devengo'->>'huella_sha256' IS DISTINCT FROM huella
  THEN RETURN false; END IF;
  IF v_grupo=1 THEN
   publicacion:=catalogo.publicacion_ref;
   normas:=catalogo.normas;
  ELSIF catalogo.publicacion_ref IS DISTINCT FROM publicacion
     OR catalogo.normas IS DISTINCT FROM normas THEN RETURN false;
  END IF;
  SELECT count(*),max((r->>'tope_centimos')::bigint)
    INTO cantidad,base FROM jsonb_array_elements(catalogo.catalogo->'reglas') r
    WHERE r->>'tipo'='dieta' AND r->>'concepto'='manutencion'
      AND (r->>'grupo')::integer=v_grupo;
  IF cantidad<>1 OR base NOT BETWEEN 1 AND 1000000 THEN RETURN false; END IF;
  esperado:=vec_dietas.calcular_tramos_publicados_v1(fecha_inicio,p_comando->>'hora_inicio',
   fecha_fin,p_comando->>'hora_fin',base,v_version_tarifa,referencia,huella,
   catalogo.catalogo->'regla_devengo'->'configuracion');
  opcion:=calc->'opciones_dieta'->(v_grupo-1);
  IF esperado IS NULL OR (opcion->>'grupo')::integer IS DISTINCT FROM v_grupo
     OR opcion->'calculo' IS DISTINCT FROM esperado THEN RETURN false; END IF;
  IF v_grupo=1 THEN
   IF v_vehiculo='propio' THEN
    SELECT count(*),max((r->>'centimos_por_km')::integer)
      INTO cantidad,n FROM jsonb_array_elements(catalogo.catalogo->'reglas') r
      WHERE r->>'tipo'='kilometraje' AND r->>'concepto'='vehiculo_propio';
    IF cantidad<>1 OR n NOT BETWEEN 1 AND 99 THEN RETURN false; END IF;
    tarifa:=n::numeric/100;
   ELSE tarifa:=0; END IF;
  ELSIF v_vehiculo='propio' THEN
   IF NOT EXISTS (SELECT 1 FROM jsonb_array_elements(catalogo.catalogo->'reglas') r
      WHERE r->>'tipo'='kilometraje' AND r->>'concepto'='vehiculo_propio'
        AND (r->>'centimos_por_km')::integer=n)
   THEN RETURN false; END IF;
  END IF;
 END LOOP;
 IF calc->>'eur_por_km' IS DISTINCT FROM tarifa::numeric(5,4)::text
 THEN RETURN false; END IF;
 IF v_vehiculo='propio' AND jsonb_array_length(calc->'tramos_ruta')>0 THEN
  IF jsonb_array_length(calc->'tramos_ruta')<>jsonb_array_length(p_comando->'codigos_ruta')-1
  THEN RETURN false; END IF;
  FOR n IN 0..jsonb_array_length(calc->'tramos_ruta')-1 LOOP
   tr:=calc->'tramos_ruta'->n;
   IF tr->>'origen_codigo' IS DISTINCT FROM p_comando->'codigos_ruta'->>n
      OR tr->>'destino_codigo' IS DISTINCT FROM p_comando->'codigos_ruta'->>(n+1)
      OR tr->>'kilometros' !~ '^(0|[1-9][0-9]{0,4})\.[0-9]{4}$'
      OR (tr->>'kilometros')::numeric<=0 THEN RETURN false; END IF;
   km:=km+(tr->>'kilometros')::numeric;
  END LOOP;
  IF km>10000 OR km IS DISTINCT FROM (calc->>'kilometros')::numeric
     OR (calc->>'importe_kilometraje_centimos')::bigint
        IS DISTINCT FROM round(km*tarifa*100)::bigint
  THEN RETURN false; END IF;
 ELSIF v_vehiculo='sin_vehiculo_propio' AND
    (calc->>'importe_kilometraje_centimos')::bigint<>0 THEN RETURN false;
 END IF;
 RETURN true;
EXCEPTION WHEN others THEN RETURN false;
END $calculo$;
REVOKE ALL ON FUNCTION vec_dietas.validar_calculo_publicado_v1(jsonb) FROM PUBLIC;

CREATE FUNCTION vec_dietas.validar_documento_publicado_v1(p_comando jsonb)
RETURNS boolean LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
AS $documento$
DECLARE calc jsonb; doc jsonb; opcion jsonb; tramo jsonb; seleccion jsonb;
 linea jsonb; esperado jsonb:='[]'::jsonb; rutas jsonb;
 catalogo vec_dietas.catalogo_liquidacion_publicado%ROWTYPE;
 v_grupo integer; n_tramos integer; indice integer; anterior_indice integer:=-1;
 otros integer:=0; base_lineas integer; man bigint:=0; km bigint:=0; suma_otros bigint:=0;
 fecha_inicio date; fecha_fin date; v_version_tarifa text; tarifa numeric;
BEGIN
 IF vec_dietas.validar_calculo_publicado_v1(p_comando) IS NOT TRUE
    OR jsonb_typeof(p_comando->'documento') IS DISTINCT FROM 'object'
    OR jsonb_typeof(p_comando->'asignacion') IS DISTINCT FROM 'object'
    OR jsonb_typeof(p_comando->'tramos_aceptados') IS DISTINCT FROM 'array'
    OR jsonb_typeof(p_comando->'vehiculo_propio') IS DISTINCT FROM 'boolean'
    OR jsonb_typeof(p_comando->'rutas') IS DISTINCT FROM 'array'
 THEN RETURN false; END IF;
 calc:=p_comando->'calculo'; doc:=p_comando->'documento';
 v_version_tarifa:=calc->>'version_tarifa';
 fecha_inicio:=(p_comando->>'fecha_inicio')::date;
 fecha_fin:=(p_comando->>'fecha_fin')::date;
 v_grupo:=(p_comando->'asignacion'->>'grupo_dieta')::integer;
 seleccion:=p_comando->'tramos_aceptados';
 IF v_grupo NOT BETWEEN 1 AND 3
    OR jsonb_typeof(doc->'lineas') IS DISTINCT FROM 'array'
    OR jsonb_array_length(doc->'lineas')>256
    OR doc->'vehiculo_propio' IS DISTINCT FROM p_comando->'vehiculo_propio'
    OR doc->>'grupo_dieta' IS DISTINCT FROM v_grupo::text
    OR doc->>'version_tarifa_aceptada' IS DISTINCT FROM v_version_tarifa
    OR p_comando->>'version_tarifa_aceptada' IS DISTINCT FROM v_version_tarifa
    OR doc->'tramos_aceptados' IS DISTINCT FROM seleccion
 THEN RETURN false; END IF;
 SELECT * INTO STRICT catalogo FROM vec_dietas.catalogo_liquidacion_publicado x
 WHERE x.version_tarifa_ref=v_version_tarifa AND x.pais_iso2='ES' AND x.grupo=v_grupo
   AND x.vehiculo=CASE WHEN p_comando->>'vehiculo_propio'='true'
      THEN 'propio' ELSE 'sin_vehiculo_propio' END
   AND fecha_inicio>=x.vigente_desde AND fecha_fin<=x.vigente_hasta;
 opcion:=calc->'opciones_dieta'->(v_grupo-1);
 n_tramos:=jsonb_array_length(opcion->'calculo'->'tramos');
 IF (n_tramos=0 AND jsonb_array_length(seleccion)<>0)
    OR (n_tramos>0 AND jsonb_array_length(seleccion) NOT IN (1,n_tramos))
 THEN RETURN false; END IF;
 FOR linea IN SELECT value FROM jsonb_array_elements(seleccion) LOOP
  IF jsonb_typeof(linea) IS DISTINCT FROM 'number'
     OR linea::text !~ '^(0|[1-9][0-9]*)$' THEN RETURN false; END IF;
  indice:=(linea::text)::integer;
  IF indice<=anterior_indice OR indice>=n_tramos THEN RETURN false; END IF;
  anterior_indice:=indice;
  tramo:=opcion->'calculo'->'tramos'->indice;
  man:=man+(tramo->>'importe_centimos')::bigint;
  esperado:=esperado||jsonb_build_array(jsonb_build_object(
   'tipo','dieta','grupo',v_grupo,'indice_tramo',indice,
   'fecha',tramo->>'fecha','concepto','manutencion',
   'importe_centimos',(tramo->>'importe_centimos')::bigint,
   'version_tarifa_ref',v_version_tarifa,'rotulo','dietas_tarifa_publicada'));
 END LOOP;
 IF n_tramos>0 AND jsonb_array_length(seleccion)=n_tramos
    AND (SELECT jsonb_agg(to_jsonb(s.n) ORDER BY s.n)
         FROM generate_series(0,n_tramos-1) s(n)) IS DISTINCT FROM seleccion
 THEN RETURN false; END IF;
 tarifa:=(calc->>'eur_por_km')::numeric;
 rutas:=vec_dietas.validar_rutas_d4_v2(p_comando,tarifa,v_version_tarifa,'dietas_tarifa_publicada');
 IF rutas IS NULL THEN RETURN false; END IF;
 esperado:=esperado||(rutas->'lineas');
 km:=(rutas->>'importe_centimos')::bigint;
 base_lineas:=jsonb_array_length(esperado);
 IF coalesce(jsonb_typeof(p_comando->'otros'),'array')<>'array'
    OR (p_comando ? 'otros' AND jsonb_array_length(p_comando->'otros')>32)
 THEN RETURN false; END IF;
 FOR linea IN SELECT x.value FROM jsonb_array_elements(doc->'lineas') WITH ORDINALITY x(value,ord)
   WHERE x.ord>base_lineas LOOP
  otros:=otros+1;
  IF otros>32 OR linea->>'tipo' NOT IN ('otro_medio','otro_gasto')
     OR jsonb_typeof(linea->'importe_centimos') IS DISTINCT FROM 'number'
     OR (linea->>'importe_centimos')::bigint NOT BETWEEN 1 AND 100000000
     OR length(coalesce(linea->>'concepto','')) NOT BETWEEN 3 AND 500
     OR linea->>'concepto' IS DISTINCT FROM btrim(linea->>'concepto')
     OR linea->>'concepto'~'[[:cntrl:]]'
     OR coalesce(linea->>'justificante_ref','') !~ '^[A-Za-z][A-Za-z0-9:_-]{2,127}$'
     OR coalesce(linea->>'justificante_sha256','') !~ '^[0-9a-f]{64}$'
     OR coalesce(linea->>'fecha','') !~ '^20[0-9]{2}-[0-9]{2}-[0-9]{2}$'
     OR (linea->>'fecha')::date NOT BETWEEN fecha_inicio AND fecha_fin
     OR linea->>'catalogo_version' IS DISTINCT FROM catalogo.catalogo->>'catalogo_otros_gastos_version_ref'
     OR NOT EXISTS (SELECT 1 FROM jsonb_array_elements(catalogo.catalogo->'reglas') r
       WHERE r->>'tipo'=linea->>'tipo' AND r->>'concepto'=linea->>'tipo_gasto')
     OR linea IS DISTINCT FROM jsonb_build_object('tipo',linea->>'tipo',
       'tipo_gasto',linea->>'tipo_gasto','catalogo_version',linea->>'catalogo_version',
       'fecha',linea->>'fecha','concepto',linea->>'concepto',
       'importe_centimos',(linea->>'importe_centimos')::bigint,
       'justificante_ref',linea->>'justificante_ref',
       'justificante_sha256',linea->>'justificante_sha256')
     OR linea IS DISTINCT FROM p_comando->'otros'->(otros-1)
  THEN RETURN false; END IF;
  suma_otros:=suma_otros+(linea->>'importe_centimos')::bigint;
  esperado:=esperado||jsonb_build_array(linea);
 END LOOP;
 IF otros IS DISTINCT FROM coalesce(jsonb_array_length(p_comando->'otros'),0)
 THEN RETURN false; END IF;
 RETURN doc->'lineas' IS NOT DISTINCT FROM esperado
   AND (doc->>'manutencion_centimos')::bigint IS NOT DISTINCT FROM man
   AND (doc->>'alojamiento_tope_centimos')::bigint IS NOT DISTINCT FROM 0
   AND (doc->>'kilometraje_centimos')::bigint IS NOT DISTINCT FROM km
   AND (doc->>'otros_centimos')::bigint IS NOT DISTINCT FROM suma_otros
   AND (doc->>'total_orientativo_centimos')::bigint IS NOT DISTINCT FROM
      man+km+suma_otros;
EXCEPTION WHEN others THEN RETURN false;
END $documento$;
REVOKE ALL ON FUNCTION vec_dietas.validar_documento_publicado_v1(jsonb) FROM PUBLIC;




-- Lectura histórica provisional y selección prospectiva publicada.
CREATE OR REPLACE FUNCTION vec_dietas.consultar_regla_devengo_dietas_v1(p_version_tarifa_ref text, p_fecha date, p_pais_iso2 text, p_variante text)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
AS $function$
DECLARE r vec_dietas.regla_devengo_provisional%ROWTYPE; n int; fecha_elegida date;
 publicado vec_dietas.catalogo_liquidacion_publicado%ROWTYPE;
BEGIN
 IF current_user<>'vec_dietas_propietario' OR session_user=current_user
    OR NOT ((pg_has_role(session_user,'vec_dietas_ejecutor','MEMBER')
        AND NOT pg_has_role(session_user,'vec_dietas_propietario','MEMBER')
        AND NOT pg_has_role(session_user,'vec_dietas_migrador','MEMBER'))
      OR EXISTS(SELECT 1 FROM pg_roles WHERE rolname=session_user AND rolsuper)) THEN
  RAISE EXCEPTION 'ejecutor catálogo Dietas inválido' USING ERRCODE='42501'; END IF;
 IF p_version_tarifa_ref IS NOT NULL AND p_version_tarifa_ref<>''
    AND p_version_tarifa_ref NOT LIKE 'provisional:%' THEN
  SELECT * INTO publicado FROM vec_dietas.catalogo_liquidacion_publicado x
   WHERE x.version_tarifa_ref=p_version_tarifa_ref AND x.grupo=1
     AND x.pais_iso2=p_pais_iso2 AND p_variante='nacional_ordinaria'
     AND p_fecha BETWEEN x.vigente_desde AND x.vigente_hasta
     AND x.fecha_publicacion<=current_date
   ORDER BY x.vehiculo LIMIT 1;
  IF NOT FOUND OR NOT EXISTS (SELECT 1 FROM vec_dietas.regla_devengo_registrada rr
       WHERE rr.regla_ref=publicado.regla_ref AND rr.version_ref=p_version_tarifa_ref
         AND rr.huella_sha256=publicado.regla_huella_sha256 AND rr.origen='publicada')
  THEN RAISE EXCEPTION 'regla publicada Dietas no disponible' USING ERRCODE='PD010'; END IF;
  RETURN jsonb_build_object('regla_ref',publicado.regla_ref,
   'version_tarifa_ref',publicado.version_tarifa_ref,'pais_iso2','ES',
   'variante','nacional_ordinaria',
   'configuracion',publicado.catalogo->'regla_devengo'->'configuracion',
   'huella_sha256',publicado.regla_huella_sha256);
 END IF;
 IF p_fecha IS NULL OR p_pais_iso2<>'ES' OR p_variante<>'nacional_ordinaria'
    OR p_version_tarifa_ref IS NULL
    OR (p_version_tarifa_ref<>'' AND p_version_tarifa_ref !~ '^provisional:[a-z0-9:-]{8,120}$') THEN
  RAISE EXCEPTION 'consulta regla Dietas inválida' USING ERRCODE='22023'; END IF;
 IF p_version_tarifa_ref='' THEN
  SELECT max(v.vigente_desde) INTO fecha_elegida
  FROM vec_dietas.regla_devengo_provisional x
  JOIN vec_dietas.version_tarifa_provisional v ON v.version_ref=x.version_tarifa_ref
  WHERE x.pais_iso2=p_pais_iso2 AND x.variante=p_variante
    AND v.vigente_desde<=p_fecha AND (v.vigente_hasta IS NULL OR p_fecha<v.vigente_hasta);
 END IF;
 SELECT count(*) INTO n FROM vec_dietas.regla_devengo_provisional x
 JOIN vec_dietas.version_tarifa_provisional v ON v.version_ref=x.version_tarifa_ref
 WHERE ((p_version_tarifa_ref='' AND v.vigente_desde=fecha_elegida)
    OR (p_version_tarifa_ref<>'' AND x.version_tarifa_ref=p_version_tarifa_ref))
   AND x.pais_iso2=p_pais_iso2 AND x.variante=p_variante
   AND v.vigente_desde<=p_fecha AND (v.vigente_hasta IS NULL OR p_fecha<v.vigente_hasta);
 IF n<>1 THEN RAISE EXCEPTION 'regla Dietas no disponible' USING ERRCODE='PD010'; END IF;
 SELECT x.* INTO STRICT r FROM vec_dietas.regla_devengo_provisional x
 JOIN vec_dietas.version_tarifa_provisional v ON v.version_ref=x.version_tarifa_ref
 WHERE ((p_version_tarifa_ref='' AND v.vigente_desde=fecha_elegida)
    OR (p_version_tarifa_ref<>'' AND x.version_tarifa_ref=p_version_tarifa_ref))
   AND x.pais_iso2=p_pais_iso2 AND x.variante=p_variante
   AND v.vigente_desde<=p_fecha AND (v.vigente_hasta IS NULL OR p_fecha<v.vigente_hasta);
 IF r.huella_sha256 IS DISTINCT FROM encode(sha256(convert_to(r.configuracion::text,'UTF8')),'hex')
    OR ARRAY(SELECT jsonb_object_keys(r.configuracion) ORDER BY 1) IS DISTINCT FROM
       ARRAY['alojamiento','dias_maximos','duracion_minima_mismo_dia_horas',
        'hora_regreso_50_despues_de','hora_regreso_mismo_dia_despues_de',
        'hora_salida_100_antes_de','hora_salida_50_antes_de','liquidable',
        'porcentaje_alojamiento_tope','porcentaje_intermedio','porcentaje_mismo_dia',
        'porcentaje_regreso','porcentaje_salida_media','porcentaje_salida_temprana',
        'regla','zona']
    OR r.configuracion->>'regla'<>'nacional_ordinaria_provisional_v1'
    OR r.configuracion->>'zona'<>'Europe/Madrid'
    OR r.configuracion->>'alojamiento'<>'tope_pendiente_justificante'
    OR r.configuracion->>'liquidable'<>'false'
    OR (r.configuracion->>'dias_maximos')::int NOT BETWEEN 1 AND 31
    OR (r.configuracion->>'duracion_minima_mismo_dia_horas')::int NOT BETWEEN 1 AND 24
    OR EXISTS (SELECT 1 FROM jsonb_each_text(r.configuracion) h(clave,valor)
       WHERE h.clave LIKE 'hora_%' AND
         (h.valor !~ '^(0|[1-9][0-9]?)$' OR h.valor::int NOT BETWEEN 0 AND 23))
    OR EXISTS (SELECT 1 FROM jsonb_each_text(r.configuracion) p(clave,valor)
       WHERE p.clave LIKE 'porcentaje_%' AND
         (p.valor !~ '^(0|[1-9][0-9]{0,2})$' OR p.valor::int NOT BETWEEN 0 AND 100)) THEN
  RAISE EXCEPTION 'regla Dietas incoherente' USING ERRCODE='PD010'; END IF;
 RETURN jsonb_build_object('regla_ref',r.regla_ref,
  'version_tarifa_ref',r.version_tarifa_ref,'pais_iso2',r.pais_iso2,
  'variante',r.variante,'configuracion',r.configuracion,'huella_sha256',r.huella_sha256);
END $function$;



-- La rama instalada permanece literal para expedientes anteriores.
CREATE OR REPLACE FUNCTION vec_dietas.validar_calculo_catalogado_v2(p_comando jsonb)
 RETURNS boolean
 LANGUAGE plpgsql
 STABLE SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
AS $function$
DECLARE calc jsonb; op jsonb; tr jsonb; regla jsonb; cfg jsonb;
        tarifa numeric; man numeric; alo numeric; km numeric:=0; importe bigint;
        suma_man bigint; suma_alo bigint; porcentajes int[]; pct int;
        v_tarifa text; inicio date; fin date; g int; n int;
BEGIN
 IF p_comando->'calculo'->>'rotulo'='dietas_tarifa_publicada' THEN
  RETURN vec_dietas.validar_calculo_publicado_v1(p_comando);
 END IF;
 IF jsonb_typeof(p_comando) IS DISTINCT FROM 'object'
    OR jsonb_typeof(p_comando->'calculo') IS DISTINCT FROM 'object'
    OR jsonb_typeof(p_comando->'codigos_ruta') IS DISTINCT FROM 'array'
    OR jsonb_array_length(p_comando->'codigos_ruta') NOT BETWEEN 2 AND 12 THEN RETURN false; END IF;
 calc:=p_comando->'calculo';
 IF calc->>'procedencia' IS DISTINCT FROM 'osrm_interno'
    OR calc->>'motor' IS DISTINCT FROM 'OSRM'
    OR length(coalesce(calc->>'version_grafo','')) NOT BETWEEN 1 AND 160
    OR calc->>'rotulo' IS DISTINCT FROM 'PROVISIONAL · pendiente de confirmación por RRHH'
    OR calc->>'version_tarifa' !~ '^provisional:[a-z0-9:-]{8,120}$'
    OR calc->>'hora_inicio' IS DISTINCT FROM p_comando->>'hora_inicio'
    OR calc->>'hora_fin' IS DISTINCT FROM p_comando->>'hora_fin'
    OR calc->>'kilometros' !~ '^(0|[1-9][0-9]{0,4})\.[0-9]{4}$'
    OR calc->>'eur_por_km' !~ '^0\.[0-9]{4}$'
    OR jsonb_typeof(calc->'tramos_ruta') IS DISTINCT FROM 'array'
    OR jsonb_array_length(calc->'tramos_ruta')<>jsonb_array_length(p_comando->'codigos_ruta')-1
    OR jsonb_typeof(calc->'opciones_dieta') IS DISTINCT FROM 'array'
    OR jsonb_array_length(calc->'opciones_dieta')<>3 THEN RETURN false; END IF;
 inicio:=(p_comando->>'fecha_inicio')::date; fin:=(p_comando->>'fecha_fin')::date;
 IF fin<inicio OR fin-inicio>30 THEN RETURN false; END IF;
 v_tarifa:=calc->>'version_tarifa';
 regla:=vec_dietas.consultar_regla_devengo_dietas_v1(v_tarifa,inicio,'ES','nacional_ordinaria');
 cfg:=regla->'configuracion';
 IF calc->>'regla_ref' IS DISTINCT FROM regla->>'regla_ref'
    OR calc->>'regla_huella_sha256' IS DISTINCT FROM regla->>'huella_sha256' THEN RETURN false; END IF;
 porcentajes:=ARRAY[(cfg->>'porcentaje_mismo_dia')::int,
  (cfg->>'porcentaje_salida_temprana')::int,(cfg->>'porcentaje_salida_media')::int,
  (cfg->>'porcentaje_regreso')::int,(cfg->>'porcentaje_intermedio')::int];
 SELECT k.eur_por_km INTO STRICT tarifa FROM vec_dietas.importe_km_provisional k
 JOIN vec_dietas.version_tarifa_provisional v ON v.version_ref=k.version_ref
 WHERE k.version_ref=v_tarifa AND k.vehiculo='automovil'
   AND v.vigente_desde<=inicio AND (v.vigente_hasta IS NULL OR fin<v.vigente_hasta);
 IF tarifa IS DISTINCT FROM (calc->>'eur_por_km')::numeric THEN RETURN false; END IF;
 FOR n IN 0..jsonb_array_length(calc->'tramos_ruta')-1 LOOP
  tr:=calc->'tramos_ruta'->n;
  IF tr->>'origen_codigo' IS DISTINCT FROM p_comando->'codigos_ruta'->>n
     OR tr->>'destino_codigo' IS DISTINCT FROM p_comando->'codigos_ruta'->>(n+1)
     OR tr->>'kilometros' !~ '^(0|[1-9][0-9]{0,4})\.[0-9]{4}$'
     OR (tr->>'kilometros')::numeric<=0 THEN RETURN false; END IF;
  km:=km+(tr->>'kilometros')::numeric;
 END LOOP;
 IF km>10000 OR km IS DISTINCT FROM (calc->>'kilometros')::numeric
    OR (calc->>'importe_kilometraje_centimos')::bigint
       IS DISTINCT FROM round(km*tarifa*100)::bigint THEN RETURN false; END IF;
 FOR g IN 1..3 LOOP
  op:=calc->'opciones_dieta'->(g-1);
  SELECT d.manutencion_eur,d.alojamiento_eur INTO STRICT man,alo
  FROM vec_dietas.importe_dieta_provisional d
  WHERE d.version_ref=v_tarifa AND d.pais_iso2='ES' AND d.grupo=g;
  IF (op->>'grupo')::int IS DISTINCT FROM g
     OR op->'calculo'->>'version_tarifa_ref' IS DISTINCT FROM v_tarifa
     OR op->'calculo'->>'rotulo' IS DISTINCT FROM calc->>'rotulo'
     OR jsonb_typeof(op->'calculo'->'tramos') IS DISTINCT FROM 'array'
     OR jsonb_array_length(op->'calculo'->'tramos')>62 THEN RETURN false; END IF;
  suma_man:=0; suma_alo:=0;
  FOR tr IN SELECT value FROM jsonb_array_elements(op->'calculo'->'tramos') LOOP
   IF tr->>'version_tarifa_ref' IS DISTINCT FROM v_tarifa
      OR tr->>'rotulo' IS DISTINCT FROM calc->>'rotulo'
      OR (tr->>'fecha')::date NOT BETWEEN inicio AND fin THEN RETURN false; END IF;
   pct:=(tr->>'porcentaje')::int; importe:=(tr->>'importe_centimos')::bigint;
   IF tr->>'tipo'='manutencion' AND pct=ANY(porcentajes)
      AND importe=ceil(man*100*pct/100)::bigint THEN suma_man:=suma_man+importe;
   ELSIF tr->>'tipo'='alojamiento_tope_pendiente_justificante'
      AND pct=(cfg->>'porcentaje_alojamiento_tope')::int
      AND importe=ceil(alo*100*pct/100)::bigint THEN suma_alo:=suma_alo+importe;
   ELSE RETURN false; END IF;
  END LOOP;
  IF suma_man IS DISTINCT FROM (op->'calculo'->>'manutencion_centimos')::bigint
     OR suma_alo IS DISTINCT FROM (op->'calculo'->>'alojamiento_tope_centimos')::bigint
     OR suma_man+suma_alo IS DISTINCT FROM (op->'calculo'->>'total_maximo_orientativo_centimos')::bigint
  THEN RETURN false; END IF;
 END LOOP;
 RETURN true;
EXCEPTION WHEN others THEN RETURN false;
END $function$;



-- La rama instalada permanece literal para expedientes anteriores.
CREATE OR REPLACE FUNCTION vec_dietas.validar_documento_v2(p_comando jsonb)
 RETURNS boolean
 LANGUAGE plpgsql
 STABLE SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
AS $function$
DECLARE calc jsonb; doc jsonb; linea jsonb; tramo jsonb; opcion jsonb; esperado jsonb:='[]'::jsonb;
        rutas_d4 jsonb; seleccion jsonb; regla_catalogo jsonb; regla_config jsonb;
        v_version_tarifa text; rotulo text; tarifa numeric; pct int; porcentajes int[];
        man numeric; alo numeric;
        importe bigint; suma_man bigint; suma_alo bigint; seleccionado_man bigint:=0;
        seleccionado_alo bigint:=0; km_centimos bigint:=0; otros_centimos bigint:=0;
        n int; grupo int; grupo_actual int; indice int; anterior_indice int:=-1; n_tramos int;
        otros int:=0; base_lineas int; fecha_inicio date; fecha_fin date;
BEGIN
 IF p_comando->'calculo'->>'rotulo'='dietas_tarifa_publicada' THEN
  RETURN vec_dietas.validar_documento_publicado_v1(p_comando);
 END IF;
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
END $function$;



-- La repetición anterior conserva su recibo; el alta nueva exige publicación.
CREATE OR REPLACE FUNCTION vec_dietas.crear_o_recuperar_comision_catalogada_v2(p_material text, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
AS $function$
DECLARE m jsonb; c jsonb; calc jsonb; salida jsonb; anterior jsonb;
        ref text; recibo_ref text; repeticion boolean;
        regla vec_dietas.regla_devengo_registrada%ROWTYPE;
BEGIN
 IF current_user<>'vec_dietas_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_dietas_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_dietas_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_dietas_migrador','MEMBER') THEN
  RAISE EXCEPTION 'ejecutor Dietas inválido' USING ERRCODE='42501'; END IF;
 IF current_setting('transaction_isolation')<>'serializable'
    OR current_setting('TimeZone')<>'UTC' THEN
  RAISE EXCEPTION 'transacción Dietas incompatible' USING ERRCODE='25000'; END IF;
 BEGIN m:=p_material::jsonb; c:=m->'comando'; calc:=c->'calculo';
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'alta Dietas inválida' USING ERRCODE='22023'; END;
 IF m->>'esquema'<>'vec.dietas.borrador-operacion.v1'
    OR m->>'operacion'<>'crear' OR m->>'recurso_ref'<>'dietas:borradores:propios'
    OR jsonb_typeof(calc) IS DISTINCT FROM 'object' THEN
  RAISE EXCEPTION 'alta Dietas inválida' USING ERRCODE='22023'; END IF;
 -- El núcleo v1 consume AD3-49, liga material completo, revalida Personal y
 -- crea el recibo base. Cualquier rechazo posterior revierte TODA la tx.
 salida:=vec_dietas.crear_o_recuperar_borrador_propio_v1(
  p_material,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,
  p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 ref:=salida->'comision'->>'referencia';
 recibo_ref:=salida->'recibo'->>'referencia';
 repeticion:=(salida->'recibo'->>'repeticion')::boolean;
 IF ref !~ '^dco_[A-Za-z0-9_-]{22,128}$'
    OR recibo_ref !~ '^rcd_[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
    OR repeticion IS NULL THEN RAISE EXCEPTION 'alta Dietas no reconciliable' USING ERRCODE='PD003'; END IF;
 IF NOT repeticion THEN
  IF calc->>'rotulo' IS DISTINCT FROM 'dietas_tarifa_publicada'
     OR vec_dietas.validar_calculo_catalogado_v2(c) IS NOT TRUE THEN
   RAISE EXCEPTION 'cálculo catalogado Dietas inválido' USING ERRCODE='PD001'; END IF;
  SELECT * INTO STRICT regla FROM vec_dietas.regla_devengo_registrada
   WHERE regla_ref=calc->>'regla_ref' AND version_ref=calc->>'version_tarifa'
     AND huella_sha256=calc->>'regla_huella_sha256' AND origen='publicada';
  INSERT INTO vec_dietas.calculo_comision(comision_ref,calculo,version_tarifa,registrado_en)
   VALUES(ref,calc,calc->>'version_tarifa',clock_timestamp());
  INSERT INTO vec_dietas.recibo_regla_creacion_comision
   (comision_ref,recibo_ref,regla_ref,regla_huella_sha256,registrada_en)
   VALUES(ref,recibo_ref,regla.regla_ref,regla.huella_sha256,clock_timestamp());
 END IF;
 SELECT x.calculo INTO STRICT anterior FROM vec_dietas.calculo_comision x WHERE x.comision_ref=ref;
 IF anterior ? 'regla_ref' AND NOT EXISTS (
   SELECT 1 FROM vec_dietas.recibo_regla_creacion_comision x
   WHERE x.comision_ref=ref AND x.recibo_ref=recibo_ref
     AND x.regla_ref=anterior->>'regla_ref'
     AND x.regla_huella_sha256=anterior->>'regla_huella_sha256') THEN
  RAISE EXCEPTION 'recibo de regla Dietas incoherente' USING ERRCODE='PD003'; END IF;
 RETURN jsonb_set(salida,'{comision,calculo}',anterior,true);
END $function$;



-- Edición nueva publicada; envío y borrado conservan historia y recibos.
CREATE OR REPLACE FUNCTION vec_dietas.mutar_comision_propia_v2(p_material text, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
AS $function$
DECLARE m jsonb; i jsonb; c jsonb; asignacion jsonb; a jsonb;
        b vec_dietas.borrador_comision%ROWTYPE; anterior vec_dietas.comision_revision%ROWTYPE;
        r vec_dietas.recibo_operacion_comision%ROWTYPE;
        estado_anterior text; estado_nuevo text; operacion text; version_actual bigint;
        nuevo bigint; ref_recibo text; fecha_inicio date; fecha_fin date;
        hora_inicio text; hora_fin text; motivo text; codigos jsonb; calculo jsonb; documento jsonb;
        vehiculo boolean; rutas jsonb;
        regla text; regla_huella text;
        ahora timestamptz(6):=date_trunc('microseconds',clock_timestamp());
        huella text; salida jsonb;
BEGIN
 BEGIN m:=p_material::jsonb; i:=m->'identidad'; c:=m->'comando';
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'material Dietas inválido' USING ERRCODE='22023'; END;
 operacion:=m->>'operacion';
 IF operacion NOT IN ('editar','borrar','enviar')
    OR jsonb_typeof(c)<>'object'
    OR m->>'recurso_ref' !~ '^dco_[A-Za-z0-9_-]{22,128}$'
    OR m->>'referencia' IS DISTINCT FROM m->>'recurso_ref'
    OR c->>'clave_idempotencia' !~ '^[A-Za-z0-9_-]{16,128}$'
    OR (c->>'version_esperada')::bigint<1
    OR c->>'relacion_ref' IS DISTINCT FROM i->>'relacion_ref'
    OR m->>'huella_semantica' !~ '^[0-9a-f]{64}$'
 THEN RAISE EXCEPTION 'material Dietas inválido' USING ERRCODE='22023'; END IF;
 IF operacion IN ('borrar','enviar') AND
    ((operacion='borrar' AND c-(ARRAY['clave_idempotencia','version_esperada','relacion_ref'])<>'{}'::jsonb)
      OR (operacion='enviar' AND c-(ARRAY['clave_idempotencia','version_esperada','relacion_ref','asignacion'])<>'{}'::jsonb))
 THEN RAISE EXCEPTION 'comando Dietas inválido' USING ERRCODE='22023'; END IF;
 a:=vec_dietas.autorizar_documento_v2(p_material,p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 SELECT * INTO b FROM vec_dietas.borrador_comision
 WHERE referencia=m->>'referencia' AND persona_ref=i->>'persona_ref'
   AND empleado_ref=i->>'empleado_ref' AND relacion_ref=i->>'relacion_ref'
   AND unidad_ref=i->>'unidad_ref' FOR UPDATE;
 IF NOT FOUND THEN RAISE EXCEPTION 'comisión Dietas ausente' USING ERRCODE='PD004'; END IF;
 SELECT * INTO r FROM vec_dietas.recibo_operacion_comision rco
 WHERE rco.persona_ref=i->>'persona_ref' AND rco.operacion=m->>'operacion'
   AND rco.clave_idempotencia=c->>'clave_idempotencia';
 IF FOUND THEN
  IF r.comision_ref IS DISTINCT FROM b.referencia
     OR vec_dietas.intencion_documento_v2(r.comando)
        IS DISTINCT FROM vec_dietas.intencion_documento_v2(c) THEN
   RAISE EXCEPTION 'conflicto de idempotencia Dietas' USING ERRCODE='PD002'; END IF;
  IF operacion IN ('editar','enviar') AND vec_personal.revalidar_asignacion_dietas_v1(
      b.relacion_ref,b.persona_ref,b.unidad_ref,
      r.comando->'asignacion'->>'asignacion_ref',
      (r.comando->'asignacion'->>'version')::bigint,
      (r.comando->'asignacion'->>'grupo_dieta')::smallint,
      r.comando->'asignacion'->>'centro_ref',
      r.comando->'asignacion'->>'administrativo_persona_ref',
      r.comando->'asignacion'->>'responsable_persona_ref',
      (i->>'fecha_referencia')::date) IS NOT TRUE THEN
   RAISE EXCEPTION 'asignación Dietas no vigente' USING ERRCODE='PD003'; END IF;
  RETURN vec_dietas.proyectar_revision_exacta_v2(r.comision_ref,r.version);
 END IF;
 SELECT * INTO anterior FROM vec_dietas.comision_revision
 WHERE comision_ref=b.referencia ORDER BY version DESC LIMIT 1;
 IF FOUND THEN
  version_actual:=anterior.version; estado_anterior:=anterior.estado;
  fecha_inicio:=anterior.fecha_inicio; fecha_fin:=anterior.fecha_fin;
  hora_inicio:=anterior.hora_inicio; hora_fin:=anterior.hora_fin;
  motivo:=anterior.motivo; codigos:=anterior.codigos_ruta;
  calculo:=anterior.calculo; documento:=anterior.documento;
  vehiculo:=anterior.vehiculo_propio; rutas:=anterior.rutas;
  regla:=anterior.regla_ref;
 ELSE
  version_actual:=1; estado_anterior:='borrador';
  fecha_inicio:=b.fecha_inicio; fecha_fin:=b.fecha_fin;
  SELECT x.calculo INTO STRICT calculo FROM vec_dietas.calculo_comision x
   WHERE x.comision_ref=b.referencia;
  hora_inicio:=calculo->>'hora_inicio'; hora_fin:=calculo->>'hora_fin';
  motivo:=b.motivo; codigos:=b.codigos_ruta; documento:=NULL;
  vehiculo:=NULL; rutas:=NULL;
  -- Solo el cálculo v1 instalado podía omitir la referencia de regla.
  -- La publicación nueva jamás recibe una regla por sustitución.
  regla:=CASE WHEN calculo->>'version_tarifa'='provisional:rd462:20260923'
                   AND calculo->>'regla_ref' IS NULL
    THEN 'provisional:regla:nacional-ordinaria:20260923'
    ELSE calculo->>'regla_ref' END;
 END IF;
 IF (c->>'version_esperada')::bigint IS DISTINCT FROM version_actual
    OR (operacion='borrar' AND estado_anterior<>'borrador')
    OR (operacion IN ('editar','enviar') AND estado_anterior NOT IN ('borrador','devuelta')) THEN
  RAISE EXCEPTION 'versión o estado Dietas incompatible' USING ERRCODE='PD005'; END IF;
 IF operacion IN ('editar','enviar') THEN
  asignacion:=c->'asignacion';
  IF jsonb_typeof(asignacion) IS DISTINCT FROM 'object'
     OR asignacion->>'relacion_ref' IS DISTINCT FROM b.relacion_ref
     OR asignacion->>'persona_ref' IS DISTINCT FROM b.persona_ref
     OR asignacion->>'unidad_ref' IS DISTINCT FROM b.unidad_ref
     OR asignacion->>'asignacion_ref' !~ '^ads_[A-Za-z0-9_-]{22,128}$'
     OR (asignacion->>'version')::bigint<1
     OR (asignacion->>'grupo_dieta')::smallint NOT BETWEEN 1 AND 3
     OR asignacion->>'administrativo_persona_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
     OR asignacion->>'responsable_persona_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
     OR asignacion->>'centro_ref' IS NULL
     OR vec_personal.revalidar_asignacion_dietas_v1(
       b.relacion_ref,b.persona_ref,b.unidad_ref,asignacion->>'asignacion_ref',
       (asignacion->>'version')::bigint,(asignacion->>'grupo_dieta')::smallint,
       asignacion->>'centro_ref',asignacion->>'administrativo_persona_ref',
       asignacion->>'responsable_persona_ref',(i->>'fecha_referencia')::date) IS NOT TRUE THEN
   RAISE EXCEPTION 'asignación Dietas no vigente' USING ERRCODE='PD003'; END IF;
 END IF;
 IF operacion='editar' THEN
  IF c-(ARRAY['clave_idempotencia','version_esperada','relacion_ref','fecha_inicio','fecha_fin','hora_inicio','hora_fin','motivo','codigos_ruta','vehiculo_propio','rutas','tramos_aceptados','version_tarifa_aceptada','asignacion','calculo','documento','otros'])<>'{}'::jsonb
     OR c->>'version_tarifa_aceptada' IS DISTINCT FROM calculo->>'version_tarifa'
     OR c->'calculo'->>'version_tarifa' IS DISTINCT FROM calculo->>'version_tarifa'
     OR c->>'fecha_inicio' !~ '^20[0-9]{2}-[0-9]{2}-[0-9]{2}$'
     OR c->>'fecha_fin' !~ '^20[0-9]{2}-[0-9]{2}-[0-9]{2}$'
     OR c->>'hora_inicio' !~ '^([01][0-9]|2[0-3]):[0-5][0-9]$'
     OR c->>'hora_fin' !~ '^([01][0-9]|2[0-3]):[0-5][0-9]$'
     OR length(coalesce(c->>'motivo','')) NOT BETWEEN 3 AND 600
     OR jsonb_typeof(c->'codigos_ruta')<>'array'
     OR jsonb_typeof(c->'vehiculo_propio') IS DISTINCT FROM 'boolean'
     OR jsonb_typeof(c->'rutas') IS DISTINCT FROM 'array'
     OR vec_dietas.validar_documento_v2(c) IS NOT TRUE THEN
   RAISE EXCEPTION 'edición Dietas inválida' USING ERRCODE='PD001'; END IF;
  fecha_inicio:=(c->>'fecha_inicio')::date; fecha_fin:=(c->>'fecha_fin')::date;
  hora_inicio:=c->>'hora_inicio'; hora_fin:=c->>'hora_fin'; motivo:=c->>'motivo';
  codigos:=c->'codigos_ruta'; calculo:=c->'calculo'; documento:=c->'documento';
  vehiculo:=(c->>'vehiculo_propio')::boolean; rutas:=c->'rutas';
  regla:=calculo->>'regla_ref';
  IF fecha_fin<fecha_inicio OR fecha_fin-fecha_inicio>30
     OR calculo->>'rotulo' IS DISTINCT FROM 'dietas_tarifa_publicada'
     OR NOT EXISTS (SELECT 1 FROM vec_dietas.regla_devengo_registrada p
         WHERE p.regla_ref=regla AND p.version_ref=calculo->>'version_tarifa'
           AND p.huella_sha256=calculo->>'regla_huella_sha256' AND p.origen='publicada') THEN
   RAISE EXCEPTION 'regla Dietas incompatible' USING ERRCODE='PD001'; END IF;
  estado_nuevo:='borrador';
 ELSIF operacion='borrar' THEN
  estado_nuevo:='eliminado';
 ELSE
  IF documento IS NULL OR vehiculo IS NULL OR rutas IS NULL THEN
   RAISE EXCEPTION 'documento Dietas incompleto' USING ERRCODE='PD005'; END IF;
  IF anterior.asignacion_ref IS DISTINCT FROM asignacion->>'asignacion_ref'
     OR anterior.asignacion_version IS DISTINCT FROM (asignacion->>'version')::bigint
     OR anterior.grupo_dieta IS DISTINCT FROM (asignacion->>'grupo_dieta')::smallint THEN
   RAISE EXCEPTION 'documento Dietas requiere actualización de asignación' USING ERRCODE='PD005'; END IF;
  estado_nuevo:='enviado_pendiente_revision';
 END IF;
 IF operacion IN ('editar','enviar') THEN
  -- El sello de asignación usa fecha de canal actual; la relación Personal
  -- también debe cubrir los dos extremos civiles de la comisión declarada.
  PERFORM vec_personal.revalidar_relacion_dietas_v1(
   i->>'relacion_ref',i->>'persona_ref',i->>'empleado_ref',i->>'unidad_ref',
   i->>'vigente_desde',coalesce(i->>'vigente_hasta',''),
   (i->>'relacion_version')::bigint,i->>'procedencia_acto_ref',i->>'fuente_ref',
   (i->>'fuente_version')::bigint,fecha_inicio);
  PERFORM vec_personal.revalidar_relacion_dietas_v1(
   i->>'relacion_ref',i->>'persona_ref',i->>'empleado_ref',i->>'unidad_ref',
   i->>'vigente_desde',coalesce(i->>'vigente_hasta',''),
   (i->>'relacion_version')::bigint,i->>'procedencia_acto_ref',i->>'fuente_ref',
   (i->>'fuente_version')::bigint,fecha_fin);
 END IF;
 SELECT p.huella_sha256 INTO STRICT regla_huella
 FROM vec_dietas.regla_devengo_registrada p WHERE p.regla_ref=regla;
 IF calculo ? 'regla_huella_sha256'
    AND calculo->>'regla_huella_sha256' IS DISTINCT FROM regla_huella THEN
  RAISE EXCEPTION 'regla Dietas incompatible' USING ERRCODE='PD001'; END IF;
 nuevo:=version_actual+1;
 INSERT INTO vec_dietas.comision_revision(comision_ref,version,estado,fecha_inicio,fecha_fin,
   hora_inicio,hora_fin,motivo,codigos_ruta,vehiculo_propio,rutas,calculo,documento,regla_ref,
   asignacion_ref,asignacion_version,grupo_dieta,centro_ref,
   administrativo_persona_ref,responsable_persona_ref,registrada_en)
 VALUES(b.referencia,nuevo,estado_nuevo,fecha_inicio,fecha_fin,hora_inicio,hora_fin,motivo,
   codigos,vehiculo,rutas,calculo,documento,regla,
   CASE WHEN operacion IN ('editar','enviar') THEN asignacion->>'asignacion_ref' ELSE anterior.asignacion_ref END,
   CASE WHEN operacion IN ('editar','enviar') THEN (asignacion->>'version')::bigint ELSE anterior.asignacion_version END,
   CASE WHEN operacion IN ('editar','enviar') THEN (asignacion->>'grupo_dieta')::smallint ELSE anterior.grupo_dieta END,
   CASE WHEN operacion IN ('editar','enviar') THEN asignacion->>'centro_ref' ELSE anterior.centro_ref END,
   CASE WHEN operacion IN ('editar','enviar') THEN asignacion->>'administrativo_persona_ref' ELSE anterior.administrativo_persona_ref END,
   CASE WHEN operacion IN ('editar','enviar') THEN asignacion->>'responsable_persona_ref' ELSE anterior.responsable_persona_ref END,
   ahora);
 huella:=encode(sha256(convert_to(jsonb_build_object('persona_ref',b.persona_ref,
   'empleado_ref',b.empleado_ref,'relacion_ref',b.relacion_ref,
   'unidad_ref',b.unidad_ref,'relacion_version',b.relacion_version,
   'referencia',b.referencia,'operacion',operacion,
   'comando',vec_dietas.intencion_documento_v2(c))::text,'UTF8')),'hex');
 ref_recibo:='rcd_'||substr(md5(b.referencia||nuevo::text||ahora::text||random()::text),1,8)||'-'||
   substr(md5(b.referencia||nuevo::text||ahora::text||random()::text),1,4)||'-'||
   substr(md5(b.referencia||nuevo::text||ahora::text||random()::text),1,4)||'-'||
   substr(md5(b.referencia||nuevo::text||ahora::text||random()::text),1,4)||'-'||
   substr(md5(b.referencia||nuevo::text||ahora::text||random()::text),1,12);
 INSERT INTO vec_dietas.recibo_operacion_comision VALUES(ref_recibo,b.referencia,nuevo,
  operacion,c->>'clave_idempotencia',c,huella,a->>'decision_ref',
  a->>'consumo_huella_sha256',a->>'auditoria_ad3_ref',a->>'actor_ref',b.persona_ref,
  regla,regla_huella,ahora);
 INSERT INTO vec_dietas.historia_operacion_comision VALUES(
  'hdi_'||md5(ref_recibo),b.referencia,nuevo,ref_recibo,estado_anterior,estado_nuevo,
  operacion,NULL,a->>'actor_ref',ahora);
 IF operacion='enviar' THEN
  INSERT INTO vec_dietas.outbox_comision(evento_ref,comision_ref,version,tipo,persona_ref,
   destinatario_persona_ref,registrada_en)
  VALUES('odi_'||md5(ref_recibo),b.referencia,nuevo,'pendiente_revision',b.persona_ref,
   asignacion->>'administrativo_persona_ref',ahora);
 END IF;
 salida:=vec_dietas.proyectar_revision_exacta_v2(b.referencia,nuevo);
 RETURN jsonb_set(salida,'{recibo,repeticion}','false'::jsonb);
END $function$;
DO $post$
DECLARE firma text;
BEGIN
 FOREACH firma IN ARRAY ARRAY[
  'vec_dietas.consultar_regla_devengo_dietas_v1(text,date,text,text)',
  'vec_dietas.validar_calculo_catalogado_v2(jsonb)',
  'vec_dietas.validar_documento_v2(jsonb)',
  'vec_dietas.crear_o_recuperar_comision_catalogada_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_dietas.mutar_comision_propia_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_dietas.validar_calculo_publicado_v1(jsonb)',
  'vec_dietas.validar_documento_publicado_v1(jsonb)'
 ] LOOP
  IF NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=to_regprocedure(firma)
      AND p.proowner='vec_dietas_propietario'::regrole AND p.prosecdef
      AND p.proconfig @> ARRAY['search_path=pg_catalog, pg_temp']::text[]
      AND p.proacl::text=CASE WHEN firma IN (
        'vec_dietas.validar_calculo_catalogado_v2(jsonb)',
        'vec_dietas.validar_documento_v2(jsonb)',
        'vec_dietas.validar_calculo_publicado_v1(jsonb)',
        'vec_dietas.validar_documento_publicado_v1(jsonb)')
       THEN '{vec_dietas_propietario=X/vec_dietas_propietario}'
       ELSE '{vec_dietas_propietario=X/vec_dietas_propietario,vec_dietas_ejecutor=X/vec_dietas_propietario}' END)
  THEN RAISE EXCEPTION 'Dietas 000017: PARO función incompatible %',firma USING ERRCODE='55000'; END IF;
 END LOOP;
 IF (SELECT count(*) FROM pg_constraint c WHERE
     (c.conrelid='vec_dietas.calculo_comision'::regclass AND c.conname='calculo_comision_version_tarifa_fkey'
      AND pg_get_constraintdef(c.oid)='FOREIGN KEY (version_tarifa) REFERENCES vec_dietas.version_tarifa_registrada(version_ref)')
     OR (c.conrelid='vec_dietas.comision_revision'::regclass AND c.conname='comision_revision_regla_ref_fkey'
      AND pg_get_constraintdef(c.oid)='FOREIGN KEY (regla_ref) REFERENCES vec_dietas.regla_devengo_registrada(regla_ref)')
     OR (c.conrelid='vec_dietas.recibo_operacion_comision'::regclass AND c.conname='recibo_operacion_comision_regla_ref_fkey'
      AND pg_get_constraintdef(c.oid)='FOREIGN KEY (regla_ref) REFERENCES vec_dietas.regla_devengo_registrada(regla_ref)')
     OR (c.conrelid='vec_dietas.recibo_regla_creacion_comision'::regclass AND c.conname='recibo_regla_creacion_comision_regla_ref_fkey'
      AND pg_get_constraintdef(c.oid)='FOREIGN KEY (regla_ref) REFERENCES vec_dietas.regla_devengo_registrada(regla_ref)'))<>4
 THEN RAISE EXCEPTION 'Dietas 000017: PARO FK incompatible' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
