\set ON_ERROR_STOP on
-- F1-19. Ámbito positivo del hijo v6, sin heredar autoridad del padre.
-- Las definiciones completas conservan la fuente instalada y sus ACL.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000247',0));
DO $control$
DECLARE f record; p pg_proc%ROWTYPE;
BEGIN
 FOR f IN SELECT * FROM (VALUES
 ('canon_contenido_detalle_rrhh_v1','dd9db72b5bf262bc14508b5cd768b35de9d01540a9fb039c65e3bfb9f506b6ab',ARRAY['search_path=pg_catalog']::text[],ARRAY['vec_contratacion_temporal_propietario/vec_contratacion_temporal_propietario:EXECUTE:false']::text[],false,'i','s'),
 ('materializar_detalle_rrhh_v1','31b963b0b3b8bdbeb0bcc105e07b8a221e990bc48fa10775dc3a19ad59550725',ARRAY['search_path=pg_catalog','row_security=on','TimeZone=UTC','lock_timeout=1s','statement_timeout=4s','idle_in_transaction_session_timeout=6s']::text[],ARRAY['vec_contratacion_temporal_propietario/vec_contratacion_temporal_propietario:EXECUTE:false']::text[],true,'s','u'),
 ('materializar_cuadro_rrhh_v1','174ee9497abf84536e59d96b8c1331dea84eeb9efde1d3e5777ca314dc975d76',ARRAY['search_path=pg_catalog','row_security=on','TimeZone=UTC','lock_timeout=1s','statement_timeout=4s','idle_in_transaction_session_timeout=6s']::text[],ARRAY['vec_contratacion_temporal_propietario/vec_contratacion_temporal_propietario:EXECUTE:false']::text[],true,'v','u'),
 ('consultar_cambios_expediente_rrhh_atestado_v1','755be29bf05c6be6addcbac1c4dcef30d90366dc2aaa106b1ba18de738ba66fb',ARRAY['search_path=pg_catalog','row_security=on','TimeZone=UTC','lock_timeout=1s','statement_timeout=4s','idle_in_transaction_session_timeout=6s']::text[],ARRAY['vec_contratacion_temporal_consultor_rrhh/vec_contratacion_temporal_propietario:EXECUTE:false','vec_contratacion_temporal_propietario/vec_contratacion_temporal_propietario:EXECUTE:false']::text[],true,'v','u'),
 ('consultar_resumen_seguimiento_rrhh_atestado_v1','73ae49630a485ebfb93677ca0636ff7166e9d69650c13b042bf2de22e889f1fb',ARRAY['search_path=pg_catalog','row_security=on','TimeZone=UTC','lock_timeout=1s','statement_timeout=4s','idle_in_transaction_session_timeout=6s']::text[],ARRAY['vec_contratacion_temporal_consultor_rrhh/vec_contratacion_temporal_propietario:EXECUTE:false','vec_contratacion_temporal_propietario/vec_contratacion_temporal_propietario:EXECUTE:false']::text[],true,'v','u'),
 ('consultar_estadisticas_rrhh_v1','e7904fab7c7145fe0a3e06b8ea24d9b168b3eee0a8a32e323787724535befe2e',ARRAY['search_path=pg_catalog','row_security=on','TimeZone=UTC','lock_timeout=1s','statement_timeout=4s','idle_in_transaction_session_timeout=6s']::text[],ARRAY['vec_contratacion_temporal_consultor_rrhh/vec_contratacion_temporal_propietario:EXECUTE:false','vec_contratacion_temporal_propietario/vec_contratacion_temporal_propietario:EXECUTE:false']::text[],true,'s','u'),
 ('contar_totales_cuadro_rrhh_v1','0b293192c96a7f193fac4160f3755d9809596226d16d4636be64b73c011201e3',ARRAY['search_path=pg_catalog','row_security=on','TimeZone=UTC','lock_timeout=1s','statement_timeout=4s','idle_in_transaction_session_timeout=6s']::text[],ARRAY['vec_contratacion_temporal_propietario/vec_contratacion_temporal_propietario:EXECUTE:false']::text[],true,'s','u'),
 ('contar_resumen_cuadro_rrhh_v1','a31cd5ef76a6d8b6c08abb6a39d46ff88cc3ec58214cd7631a1faa35f2b28981',ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=1s','statement_timeout=4s','idle_in_transaction_session_timeout=6s']::text[],ARRAY['vec_contratacion_temporal_propietario/vec_contratacion_temporal_propietario:EXECUTE:false']::text[],true,'s','u'),
 ('canon_resumen_publicacion_rrhh_v1','464e2c9f355f654adbd0c19d2830309c96a9317604dc947bf76f245be5118289',ARRAY['search_path=pg_catalog']::text[],ARRAY['vec_contratacion_temporal_propietario/vec_contratacion_temporal_propietario:EXECUTE:false']::text[],false,'i','s'),
 ('rechazar_nombramiento_derivado_ct241','211a2f4d7fbcfa09f30d871f4c621846cd52ead3e75e38a43788b2e5d28d9cf2',ARRAY['search_path=pg_catalog, pg_temp','row_security=on']::text[],ARRAY['vec_contratacion_temporal_propietario/vec_contratacion_temporal_propietario:EXECUTE:false']::text[],true,'v','u')
 ) AS esperado(nombre,huella,configuracion,acl,definidor,volatilidad,paralelo) LOOP
  SELECT * INTO STRICT p FROM pg_proc
   WHERE pronamespace='vec_contratacion_temporal'::regnamespace AND proname=f.nombre;
  IF encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM f.huella
     OR p.proowner<>'vec_contratacion_temporal_propietario'::regrole
     OR p.prosecdef IS DISTINCT FROM f.definidor
     OR p.provolatile IS DISTINCT FROM f.volatilidad::"char"
     OR p.proparallel IS DISTINCT FROM f.paralelo::"char"
     OR p.proconfig IS DISTINCT FROM f.configuracion
     OR (SELECT array_agg((a.grantee::regrole::text)||'/'||(a.grantor::regrole::text)||':'||
               a.privilege_type||':'||a.is_grantable::text ORDER BY a.grantee::regrole::text COLLATE "C")
          FROM aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a) IS DISTINCT FROM f.acl
  THEN RAISE EXCEPTION 'CT247: definición o permisos divergentes: %',f.nombre USING ERRCODE='55000'; END IF;
 END LOOP;
END $control$;
DO $tipo$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_type t
     WHERE t.oid='vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1'::regtype
       AND t.typowner='vec_contratacion_temporal_propietario'::regrole
       AND NOT EXISTS(SELECT 1 FROM aclexplode(coalesce(t.typacl,acldefault('T',t.typowner))) a
         WHERE a.grantee<>t.typowner OR a.privilege_type<>'USAGE' OR a.is_grantable))
 THEN RAISE EXCEPTION 'CT247: propietario o permisos del tipo divergentes' USING ERRCODE='55000'; END IF;
 IF (SELECT array_agg(a.attname::text ORDER BY a.attnum) FROM pg_attribute a
      WHERE a.attrelid=(SELECT typrelid FROM pg_type WHERE oid=
        'vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1'::regtype)
        AND a.attnum>0 AND NOT a.attisdropped) IS DISTINCT FROM
    ARRAY['resumen','solicitud','analisis_presente','analisis','referencia_analisis',
      'cobertura_presente','cobertura','referencia_cobertura','asignacion_presente',
      'asignacion','referencia_asignacion','hitos','fiscalizacion_presente',
      'fiscalizacion','referencia_fiscalizacion','referencia_subsanacion']
 THEN RAISE EXCEPTION 'CT247: tipo de detalle divergente' USING ERRCODE='55000'; END IF;
END $tipo$;
ALTER TYPE vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1 ADD ATTRIBUTE instruccion_compartida jsonb;

-- CT247-GENERADO: canon_contenido_detalle_rrhh_v1
CREATE OR REPLACE FUNCTION vec_contratacion_temporal.canon_contenido_detalle_rrhh_v1(p_generada_en timestamp with time zone, p_entrada vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1)
 RETURNS bytea
 LANGUAGE plpgsql
 IMMUTABLE PARALLEL SAFE STRICT
 SET search_path TO 'pg_catalog'
AS $function$
DECLARE
    v_canon bytea :=
        vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
            (CASE WHEN p_entrada.instruccion_compartida IS NOT NULL THEN 'VEC-CT-CONTENIDO-DETALLE-RRHH-V5'
                  WHEN (p_entrada.solicitud).periodo_causa_fin IS NOT NULL
                       OR (p_entrada.analisis).periodo_causa_fin IS NOT NULL
                THEN 'VEC-CT-CONTENIDO-DETALLE-RRHH-V4'
                ELSE 'VEC-CT-CONTENIDO-DETALLE-RRHH-V3' END)
                || pg_catalog.chr(10)
        );
    v_resumen vec_contratacion_temporal.resumen_publicacion_rrhh_v1;
    v_solicitud vec_contratacion_temporal.solicitud_operativa_rrhh_v1;
    v_analisis vec_contratacion_temporal.analisis_operativo_rrhh_v1;
    v_cobertura vec_contratacion_temporal.cobertura_operativa_rrhh_v1;
    v_asignacion vec_contratacion_temporal.asignacion_operativa_rrhh_v1;
    v_fiscalizacion vec_contratacion_temporal.fiscalizacion_operativa_rrhh_v1;
    v_hito vec_contratacion_temporal.hito_expediente_rrhh_v1;
    v_anterior vec_contratacion_temporal.hito_expediente_rrhh_v1;
    v_comprobacion
        vec_contratacion_temporal.comprobacion_operativa_rrhh_v1;
    v_campos text[];
    v_campo text;
    v_claves_vistas text[] := ARRAY[]::text[];
    v_indice integer;
    v_total_hitos integer;
    v_total_instruccion integer;
    v_version_instruccion numeric(20,0);
    v_hitos_instruccion vec_contratacion_temporal.hito_expediente_rrhh_v1[];
    v_instruccion jsonb := p_entrada.instruccion_compartida;
    v_derivado boolean := p_entrada.instruccion_compartida IS NOT NULL;
    v_total_comprobaciones integer;
    v_mascara smallint := 0;
    v_analisis_nulo boolean;
    v_cobertura_nula boolean;
    v_asignacion_nula boolean;
    v_fiscalizacion_nula boolean;
    v_subsanacion_presente boolean;
    v_utc timestamp;
BEGIN
    IF p_entrada.resumen IS NOT DISTINCT FROM NULL
       OR p_entrada.solicitud IS NOT DISTINCT FROM NULL
       OR p_entrada.analisis_presente IS NULL
       OR p_entrada.referencia_analisis IS NULL
       OR p_entrada.cobertura_presente IS NULL
       OR p_entrada.referencia_cobertura IS NULL
       OR p_entrada.asignacion_presente IS NULL
       OR p_entrada.referencia_asignacion IS NULL
       OR p_entrada.hitos IS NULL
       OR p_entrada.fiscalizacion_presente IS NULL
       OR p_entrada.referencia_fiscalizacion IS NULL
       OR p_entrada.referencia_subsanacion IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'contenido de detalle RRHH inválido';
    END IF;
    v_resumen := p_entrada.resumen;
    v_solicitud := p_entrada.solicitud;
    v_analisis_nulo := p_entrada.analisis IS NOT DISTINCT FROM NULL;
    v_cobertura_nula := p_entrada.cobertura IS NOT DISTINCT FROM NULL;
    v_asignacion_nula := p_entrada.asignacion IS NOT DISTINCT FROM NULL;
    v_fiscalizacion_nula := p_entrada.fiscalizacion IS NOT DISTINCT FROM NULL;

    PERFORM vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
        p_generada_en
    );
    v_canon := v_canon
        || vec_contratacion_temporal.canon_resumen_publicacion_rrhh_v1(
            v_resumen
        );
    IF p_generada_en < v_resumen.actualizado_en
       OR v_solicitud.grupo_subgrupo IS NULL
       OR v_solicitud.grupo_subgrupo !~
          '^[A-Z][A-Z0-9/+.-]{0,19}$'
       OR v_solicitud.motivo_clave IS NULL
       OR v_solicitud.motivo_clave !~ '^[a-z][a-z0-9._-]{1,79}$'
       OR v_solicitud.periodo_inicio IS NULL
       OR (v_solicitud.periodo_fin IS NULL
           AND coalesce(v_solicitud.periodo_causa_fin,'') !~
               '^[a-z][a-z0-9._-]{1,79}$')
       OR (v_solicitud.periodo_fin IS NOT NULL
           AND v_solicitud.periodo_causa_fin IS NOT NULL)
       OR v_solicitud.periodo_fin < v_solicitud.periodo_inicio THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'contenido de detalle RRHH inválido';
    END IF;
    PERFORM vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
        v_solicitud.periodo_inicio
    );
    PERFORM vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
        v_solicitud.periodo_fin
    );
    v_utc := v_solicitud.periodo_inicio AT TIME ZONE 'UTC';
    IF extract(hour FROM v_utc) <> 0
       OR extract(minute FROM v_utc) <> 0
       OR extract(second FROM v_utc) <> 0 THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'contenido de detalle RRHH inválido';
    END IF;
    v_utc := v_solicitud.periodo_fin AT TIME ZONE 'UTC';
    IF extract(hour FROM v_utc) <> 0
       OR extract(minute FROM v_utc) <> 0
       OR extract(second FROM v_utc) <> 0 THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'contenido de detalle RRHH inválido';
    END IF;

    v_total_hitos := pg_catalog.cardinality(p_entrada.hitos);
    IF v_total_hitos < 1
       OR v_total_hitos::numeric + (CASE WHEN v_derivado THEN 5 ELSE 0 END) <> v_resumen.version
       OR pg_catalog.array_ndims(p_entrada.hitos) <> 1
       OR pg_catalog.array_lower(p_entrada.hitos, 1) <> 1 THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'contenido de detalle RRHH inválido';
    END IF;
    IF pg_catalog.array_position(p_entrada.hitos, NULL) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'contenido de detalle RRHH inválido';
    END IF;
    v_hitos_instruccion := p_entrada.hitos;
    v_total_instruccion := v_total_hitos;
    v_version_instruccion := v_resumen.version;
    IF v_derivado THEN
        IF v_resumen.version<>6 OR v_total_hitos<>1 THEN
          RAISE EXCEPTION 'detalle derivado fuera de v6' USING ERRCODE='22023'; END IF;
        IF pg_catalog.jsonb_typeof(v_instruccion)<>'object'
           OR NOT (p_entrada.analisis_presente AND p_entrada.cobertura_presente
                   AND p_entrada.asignacion_presente AND p_entrada.fiscalizacion_presente)
           OR (SELECT count(*) FROM pg_catalog.jsonb_object_keys(v_instruccion))<>10
           OR pg_catalog.jsonb_typeof(v_instruccion->'hitos_padre')<>'array'
           OR COALESCE(v_instruccion->>'padre_ref','') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
           OR COALESCE(v_instruccion->>'derivacion_ref','') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
           OR COALESCE(v_instruccion->>'recibo_derivacion_ref','') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
           OR COALESCE(v_instruccion->>'oferta_ref','') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
           OR COALESCE(v_instruccion->>'huella_padre_sha256','') !~ '^[0-9a-f]{64}$'
           OR v_instruccion->>'huella_padre_sha256'=pg_catalog.repeat('0',64)
           OR COALESCE(v_instruccion->>'numero_plaza','') !~ '^(100|[1-9][0-9]?)$'
           OR v_resumen.numero_visible IS DISTINCT FROM
              (v_instruccion->>'numero_visible_padre')||'-P'||(v_instruccion->>'numero_plaza')
           OR v_instruccion->>'padre_ref' = v_resumen.expediente_ref
           OR v_instruccion->>'creado_en_padre' IS NULL
           OR v_instruccion->>'actualizado_en_padre' IS NULL
           OR (v_instruccion->>'creado_en_padre')::timestamptz >
              (v_instruccion->>'actualizado_en_padre')::timestamptz
           OR (v_instruccion->>'actualizado_en_padre')::timestamptz > v_resumen.creado_en
        THEN RAISE EXCEPTION 'contenido de detalle RRHH inválido' USING ERRCODE='22023'; END IF;
        SELECT pg_catalog.array_agg(ROW(
          (nodo->>'secuencia')::numeric(20,0),
          (nodo->>'version_expediente')::numeric(20,0),
          nodo->>'accion_clave',(nodo->>'realizada_en')::timestamptz,
          COALESCE(nodo->>'fase_origen',''),nodo->>'fase_destino',
          nodo->>'estado_origen',nodo->>'estado_destino')::
          vec_contratacion_temporal.hito_expediente_rrhh_v1 ORDER BY ord)
          INTO v_hitos_instruccion
          FROM pg_catalog.jsonb_array_elements(v_instruccion->'hitos_padre')
            WITH ORDINALITY AS original(nodo,ord);
        v_total_instruccion := pg_catalog.cardinality(v_hitos_instruccion);
        v_version_instruccion := v_total_instruccion;
        IF v_total_instruccion<6 OR v_total_instruccion>1500 OR
           pg_catalog.array_position(v_hitos_instruccion,NULL) IS NOT NULL THEN
           RAISE EXCEPTION 'contenido de detalle RRHH inválido' USING ERRCODE='22023'; END IF;
    END IF;

    IF p_entrada.analisis_presente THEN
        IF v_analisis_nulo
           OR p_entrada.referencia_analisis
              NOT BETWEEN 2 AND v_version_instruccion THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de detalle RRHH inválido';
        END IF;
        v_mascara := v_mascara | 1;
        v_analisis := p_entrada.analisis;
    ELSIF NOT v_analisis_nulo OR p_entrada.referencia_analisis <> 0
          OR v_resumen.modalidad_clave <> '' THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'contenido de detalle RRHH inválido';
    END IF;
    IF p_entrada.cobertura_presente THEN
        IF v_cobertura_nula OR NOT p_entrada.analisis_presente
           OR p_entrada.referencia_cobertura
              NOT BETWEEN 2 AND v_version_instruccion
           OR p_entrada.referencia_cobertura
              <= p_entrada.referencia_analisis THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de detalle RRHH inválido';
        END IF;
        v_mascara := v_mascara | 2;
        v_cobertura := p_entrada.cobertura;
    ELSIF NOT v_cobertura_nula OR p_entrada.referencia_cobertura <> 0 THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'contenido de detalle RRHH inválido';
    END IF;
    IF p_entrada.asignacion_presente THEN
        IF v_asignacion_nula OR NOT p_entrada.cobertura_presente
           OR p_entrada.referencia_asignacion
              NOT BETWEEN 2 AND v_version_instruccion
           OR p_entrada.referencia_asignacion
              <= p_entrada.referencia_cobertura THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de detalle RRHH inválido';
        END IF;
        v_mascara := v_mascara | 4;
        v_asignacion := p_entrada.asignacion;
    ELSIF NOT v_asignacion_nula OR p_entrada.referencia_asignacion <> 0
          OR v_resumen.unidad_ref <> '' THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'contenido de detalle RRHH inválido';
    END IF;
    IF p_entrada.fiscalizacion_presente THEN
        IF v_fiscalizacion_nula OR NOT p_entrada.asignacion_presente
           OR p_entrada.referencia_fiscalizacion
              NOT BETWEEN 1 AND v_version_instruccion
           OR p_entrada.referencia_fiscalizacion
              <= p_entrada.referencia_asignacion
           OR p_entrada.referencia_subsanacion < 0
           OR p_entrada.referencia_subsanacion > v_version_instruccion THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de detalle RRHH inválido';
        END IF;
        v_mascara := v_mascara | 8;
        v_fiscalizacion := p_entrada.fiscalizacion;
    ELSIF NOT v_fiscalizacion_nula
          OR p_entrada.referencia_fiscalizacion <> 0
          OR p_entrada.referencia_subsanacion <> 0 THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'contenido de detalle RRHH inválido';
    END IF;
    IF v_mascara NOT IN (0, 1, 3, 7, 15) THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'contenido de detalle RRHH inválido';
    END IF;

    IF p_entrada.analisis_presente THEN
        IF v_analisis.modalidad_clave IS NULL
           OR v_analisis.modalidad_clave !~
              '^[a-z][a-z0-9._-]{1,79}$'
           OR v_analisis.modalidad_clave <> v_resumen.modalidad_clave
           OR v_analisis.categoria_ref IS NULL
           OR v_analisis.categoria_ref !~
              '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
           OR v_analisis.categoria_ref <> v_resumen.categoria_ref
           OR v_analisis.causa_clave IS NULL
           OR v_analisis.causa_clave !~ '^[a-z][a-z0-9._-]{1,79}$'
           OR v_analisis.periodo_inicio IS NULL
           OR (v_analisis.periodo_fin IS NULL
               AND coalesce(v_analisis.periodo_causa_fin,'') !~
                   '^[a-z][a-z0-9._-]{1,79}$')
           OR (v_analisis.periodo_fin IS NOT NULL
               AND v_analisis.periodo_causa_fin IS NOT NULL)
           OR v_analisis.periodo_fin < v_analisis.periodo_inicio
           OR v_analisis.porcentaje_jornada IS NULL
           OR v_analisis.porcentaje_jornada NOT BETWEEN 1 AND 10000
           OR v_analisis.resultado_rc IS NULL
           OR v_analisis.resultado_rc NOT IN (
               'validada', 'no_requerida', 'rechazada'
           )
           OR v_analisis.coste_presente IS NULL
           OR v_analisis.coste_centimos IS NULL
           OR v_analisis.coste_moneda IS NULL
           OR v_analisis.fuente_coste_ref IS NULL
           OR v_analisis.observaciones IS NULL
           OR pg_catalog.length(v_analisis.observaciones) > 4000 THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de detalle RRHH inválido';
        END IF;
        PERFORM vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
            v_analisis.periodo_inicio
        );
        PERFORM vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
            v_analisis.periodo_fin
        );
        v_utc := v_analisis.periodo_inicio AT TIME ZONE 'UTC';
        IF extract(hour FROM v_utc) <> 0
           OR extract(minute FROM v_utc) <> 0
           OR extract(second FROM v_utc) <> 0 THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de detalle RRHH inválido';
        END IF;
        v_utc := v_analisis.periodo_fin AT TIME ZONE 'UTC';
        IF extract(hour FROM v_utc) <> 0
           OR extract(minute FROM v_utc) <> 0
           OR extract(second FROM v_utc) <> 0 THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de detalle RRHH inválido';
        END IF;
        IF v_analisis.coste_presente THEN
            IF v_analisis.coste_centimos <= 0
               OR v_analisis.coste_moneda <> 'EUR'
               OR v_analisis.fuente_coste_ref !~
                  '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN
                RAISE EXCEPTION USING ERRCODE = '22023',
                    MESSAGE = 'contenido de detalle RRHH inválido';
            END IF;
        ELSIF v_analisis.coste_centimos <> 0
              OR v_analisis.coste_moneda <> ''
              OR v_analisis.fuente_coste_ref <> '' THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de detalle RRHH inválido';
        END IF;
    END IF;

    IF p_entrada.cobertura_presente THEN
        IF v_cobertura.via_clave IS NULL
           OR v_cobertura.via_clave !~ '^[a-z][a-z0-9._-]{1,79}$'
           OR v_cobertura.decision_gobernada IS NULL
           OR v_cobertura.procedimiento_ref IS NULL
           OR v_cobertura.bolsa_ref IS NULL
           OR v_cobertura.comprobaciones IS NULL THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de detalle RRHH inválido';
        END IF;
        v_total_comprobaciones :=
            pg_catalog.cardinality(v_cobertura.comprobaciones);
        IF v_cobertura.decision_gobernada THEN
            IF v_cobertura.procedimiento_ref <> ''
               OR v_cobertura.bolsa_ref <> '' THEN
                RAISE EXCEPTION USING ERRCODE = '22023',
                    MESSAGE = 'contenido de detalle RRHH inválido';
            END IF;
            IF v_total_comprobaciones = 0 THEN
                IF pg_catalog.array_ndims(v_cobertura.comprobaciones)
                   IS NOT NULL THEN
                    RAISE EXCEPTION USING ERRCODE = '22023',
                        MESSAGE = 'contenido de detalle RRHH inválido';
                END IF;
            ELSIF v_total_comprobaciones NOT BETWEEN 1 AND 32
               OR pg_catalog.array_ndims(v_cobertura.comprobaciones) <> 1
               OR pg_catalog.array_lower(v_cobertura.comprobaciones, 1) <> 1 THEN
                RAISE EXCEPTION USING ERRCODE = '22023',
                    MESSAGE = 'contenido de detalle RRHH inválido';
            END IF;
        ELSE
            IF v_cobertura.procedimiento_ref !~
               '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
               OR v_cobertura.bolsa_ref !~
                  '^$|^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
               OR v_total_comprobaciones NOT BETWEEN 1 AND 32
               OR pg_catalog.array_ndims(
                   v_cobertura.comprobaciones
               ) <> 1
               OR pg_catalog.array_lower(
                   v_cobertura.comprobaciones, 1
               ) <> 1 THEN
                RAISE EXCEPTION USING ERRCODE = '22023',
                    MESSAGE = 'contenido de detalle RRHH inválido';
            END IF;
        END IF;
        IF v_total_comprobaciones > 0 THEN
            IF pg_catalog.array_position(v_cobertura.comprobaciones, NULL) IS NOT NULL THEN
                RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'contenido de detalle RRHH inválido';
            END IF;
            FOR v_indice IN 1..v_total_comprobaciones LOOP
                v_comprobacion := v_cobertura.comprobaciones[v_indice];
                IF v_comprobacion.clave IS NULL OR v_comprobacion.clave !~ '^[a-z][a-z0-9._-]{1,79}$'
                   OR v_comprobacion.resultado IS NULL OR v_comprobacion.resultado NOT IN ('afirmativa', 'negativa', 'no_aplica', 'no_consta')
                   OR v_comprobacion.clave = ANY(v_claves_vistas) THEN
                    RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'contenido de detalle RRHH inválido';
                END IF;
                v_claves_vistas := pg_catalog.array_append(v_claves_vistas, v_comprobacion.clave);
            END LOOP;
        END IF;
    END IF;

    IF p_entrada.asignacion_presente THEN
        IF v_asignacion.unidad_ref IS NULL
           OR v_asignacion.unidad_ref !~
              '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
           OR v_asignacion.unidad_ref <> v_resumen.unidad_ref
           OR v_asignacion.asignada_en IS NULL
           OR v_asignacion.asignada_en < (CASE WHEN v_derivado THEN (v_instruccion->>'creado_en_padre')::timestamptz ELSE v_resumen.creado_en END)
           OR v_asignacion.asignada_en > v_resumen.actualizado_en
           OR v_asignacion.motivo_clave IS NULL
           OR v_asignacion.motivo_clave !~
              '^$|^[a-z][a-z0-9._-]{1,79}$' THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de detalle RRHH inválido';
        END IF;
        PERFORM vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
            v_asignacion.asignada_en
        );
    END IF;

    FOR v_indice IN 1..v_total_instruccion LOOP
        v_hito := v_hitos_instruccion[v_indice];
        IF v_hito.secuencia IS NULL
           OR v_hito.secuencia <> v_indice
           OR v_hito.version_expediente IS NULL
           OR v_hito.version_expediente <> v_indice
           OR v_hito.accion_clave IS NULL
           OR v_hito.accion_clave !~ '^[a-z][a-z0-9._-]{1,79}$'
           OR v_hito.realizada_en IS NULL
           OR v_hito.realizada_en > v_resumen.actualizado_en
           OR v_hito.fase_origen IS NULL
           OR v_hito.fase_origen !~
              '^$|^[a-z][a-z0-9._-]{1,79}$'
           OR v_hito.fase_destino IS NULL
           OR v_hito.fase_destino !~ '^[a-z][a-z0-9._-]{1,79}$'
           OR v_hito.estado_origen IS NULL
           OR v_hito.estado_origen NOT IN (
               'pendiente', 'en_curso', 'espera_externa',
               'completado', 'incidencia', 'cancelado'
           )
           OR v_hito.estado_destino IS NULL
           OR v_hito.estado_destino NOT IN (
               'pendiente', 'en_curso', 'espera_externa',
               'completado', 'incidencia', 'cancelado'
           ) THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de detalle RRHH inválido';
        END IF;
        PERFORM vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
            v_hito.realizada_en
        );
        IF p_entrada.fiscalizacion_presente
           AND v_indice = p_entrada.referencia_fiscalizacion THEN
            IF v_hito.accion_clave <>
                   'contratacion_temporal.fiscalizacion.registrar'
               OR v_hito.realizada_en <> v_fiscalizacion.registrada_en THEN
                RAISE EXCEPTION USING ERRCODE = '22023',
                    MESSAGE = 'contenido de detalle RRHH inválido';
            END IF;
        END IF;
        IF p_entrada.fiscalizacion_presente
           AND p_entrada.referencia_subsanacion <> 0
           AND v_indice = p_entrada.referencia_subsanacion THEN
            IF v_hito.accion_clave <>
                   'contratacion_temporal.subsanacion_reparos.registrar'
               OR v_hito.realizada_en <>
                  v_fiscalizacion.subsanacion_registrada_en THEN
                RAISE EXCEPTION USING ERRCODE = '22023',
                    MESSAGE = 'contenido de detalle RRHH inválido';
            END IF;
        END IF;
        IF v_indice = 1 THEN
            IF v_hito.fase_origen <> ''
               OR v_hito.estado_origen <> 'pendiente' THEN
                RAISE EXCEPTION USING ERRCODE = '22023',
                    MESSAGE = 'contenido de detalle RRHH inválido';
            END IF;
        ELSIF v_hito.fase_origen <> v_anterior.fase_destino
              OR v_hito.estado_origen <> v_anterior.estado_destino
              OR v_hito.realizada_en < v_anterior.realizada_en THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de detalle RRHH inválido';
        END IF;
        v_anterior := v_hito;
    END LOOP;
    IF (NOT v_derivado AND (v_anterior.fase_destino <> v_resumen.fase_clave
       OR v_anterior.estado_destino <> v_resumen.estado_clave
       OR v_anterior.realizada_en <> v_resumen.actualizado_en))
       OR (v_derivado AND (v_anterior.fase_destino <> 'fiscalizacion'
       OR v_anterior.estado_destino <> 'en_curso'
       OR v_anterior.realizada_en <>
           (v_instruccion->>'actualizado_en_padre')::timestamptz
       OR v_hitos_instruccion[1].realizada_en <>
           (v_instruccion->>'creado_en_padre')::timestamptz))
       OR (p_entrada.asignacion_presente AND
           (v_hitos_instruccion[p_entrada.referencia_asignacion::integer]).realizada_en
           <> v_asignacion.asignada_en) THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'contenido de detalle RRHH inválido';
    END IF;

    IF v_derivado THEN
        v_anterior := NULL;
        FOR v_indice IN 1..v_total_hitos LOOP
            v_hito := p_entrada.hitos[v_indice];
            IF v_hito.secuencia IS DISTINCT FROM v_indice+5
               OR v_hito.version_expediente IS DISTINCT FROM v_indice+5
               OR v_hito.accion_clave !~ '^[a-z][a-z0-9._-]{1,79}$'
               OR v_hito.realizada_en IS NULL
               OR v_hito.realizada_en > v_resumen.actualizado_en
               OR v_hito.fase_destino !~ '^[a-z][a-z0-9._-]{1,79}$'
               OR v_hito.estado_origen NOT IN ('pendiente','en_curso','espera_externa','completado','incidencia','cancelado')
               OR v_hito.estado_destino NOT IN ('pendiente','en_curso','espera_externa','completado','incidencia','cancelado')
            THEN RAISE EXCEPTION 'contenido de detalle RRHH inválido' USING ERRCODE='22023'; END IF;
            PERFORM vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(v_hito.realizada_en);
            IF v_indice=1 THEN
                IF v_hito.accion_clave<>'contratacion_temporal.nombramiento.derivar'
                   OR v_hito.fase_origen<>'fiscalizacion' OR v_hito.estado_origen<>'en_curso'
                   OR v_hito.fase_destino<>'fiscalizacion' OR v_hito.estado_destino<>'en_curso'
                   OR v_hito.realizada_en<>v_resumen.creado_en
                THEN RAISE EXCEPTION 'contenido de detalle RRHH inválido' USING ERRCODE='22023'; END IF;
            ELSIF v_hito.fase_origen<>v_anterior.fase_destino
                OR v_hito.estado_origen<>v_anterior.estado_destino
                OR v_hito.realizada_en<v_anterior.realizada_en THEN
                RAISE EXCEPTION 'contenido de detalle RRHH inválido' USING ERRCODE='22023';
            END IF;
            v_anterior:=v_hito;
        END LOOP;
        IF v_anterior.fase_destino<>v_resumen.fase_clave
           OR v_anterior.estado_destino<>v_resumen.estado_clave
           OR v_anterior.realizada_en<>v_resumen.actualizado_en THEN
           RAISE EXCEPTION 'contenido de detalle RRHH inválido' USING ERRCODE='22023'; END IF;
    END IF;

    v_campos := ARRAY[
        v_solicitud.grupo_subgrupo,
        v_solicitud.motivo_clave,
        vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
            v_solicitud.periodo_inicio
        ),
        CASE WHEN v_solicitud.periodo_fin IS NOT NULL THEN
            (CASE WHEN v_derivado OR (p_entrada.solicitud).periodo_causa_fin IS NOT NULL
                    OR (p_entrada.analisis).periodo_causa_fin IS NOT NULL
                THEN 'fecha:' ELSE '' END) ||
            vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
                v_solicitud.periodo_fin
            )
        ELSE 'causa:' || v_solicitud.periodo_causa_fin END,
        v_mascara::text
    ]::text[];
    FOREACH v_campo IN ARRAY v_campos LOOP
        v_canon := v_canon
            || vec_contratacion_temporal.encuadrar_valor_rrhh_v1(
                vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
                    v_campo
                )
            );
    END LOOP;

    v_campos := ARRAY[
        CASE WHEN p_entrada.analisis_presente THEN '1' ELSE '0' END,
        p_entrada.referencia_analisis::text
    ]::text[];
    IF p_entrada.analisis_presente THEN
        v_campos := v_campos || ARRAY[
            v_analisis.modalidad_clave, v_analisis.categoria_ref,
            v_analisis.causa_clave,
            vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
                v_analisis.periodo_inicio
            ),
            CASE WHEN v_analisis.periodo_fin IS NOT NULL THEN
                (CASE WHEN v_derivado OR (p_entrada.solicitud).periodo_causa_fin IS NOT NULL
                        OR (p_entrada.analisis).periodo_causa_fin IS NOT NULL
                    THEN 'fecha:' ELSE '' END) ||
                vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
                    v_analisis.periodo_fin
                )
            ELSE 'causa:' || v_analisis.periodo_causa_fin END,
            v_analisis.porcentaje_jornada::text,
            v_analisis.resultado_rc,
            CASE WHEN v_analisis.coste_presente THEN '1' ELSE '0' END
        ]::text[];
        IF v_analisis.coste_presente THEN
            v_campos := v_campos || ARRAY[
                v_analisis.coste_centimos::text,
                v_analisis.coste_moneda
            ]::text[];
        END IF;
        v_campos := pg_catalog.array_append(
            v_campos, v_analisis.fuente_coste_ref
        );
        v_campos := pg_catalog.array_append(
            v_campos, v_analisis.observaciones
        );
    END IF;
    FOREACH v_campo IN ARRAY v_campos LOOP
        v_canon := v_canon
            || vec_contratacion_temporal.encuadrar_valor_rrhh_v1(
                vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
                    v_campo
                )
            );
    END LOOP;

    v_campos := ARRAY[
        CASE WHEN p_entrada.cobertura_presente THEN '1' ELSE '0' END,
        p_entrada.referencia_cobertura::text
    ]::text[];
    IF p_entrada.cobertura_presente THEN
        v_campos := v_campos || ARRAY[
            v_cobertura.via_clave,
            CASE WHEN v_cobertura.decision_gobernada THEN '1' ELSE '0' END,
            v_cobertura.procedimiento_ref, v_cobertura.bolsa_ref,
            v_total_comprobaciones::text
        ]::text[];
        FOR v_indice IN 1..v_total_comprobaciones LOOP
            v_comprobacion := v_cobertura.comprobaciones[v_indice];
            v_campos := v_campos || ARRAY[
                v_comprobacion.clave, v_comprobacion.resultado
            ]::text[];
        END LOOP;
    END IF;
    FOREACH v_campo IN ARRAY v_campos LOOP
        v_canon := v_canon
            || vec_contratacion_temporal.encuadrar_valor_rrhh_v1(
                vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
                    v_campo
                )
            );
    END LOOP;

    v_campos := ARRAY[
        CASE WHEN p_entrada.asignacion_presente THEN '1' ELSE '0' END,
        p_entrada.referencia_asignacion::text
    ]::text[];
    IF p_entrada.asignacion_presente THEN
        v_campos := v_campos || ARRAY[
            v_asignacion.unidad_ref,
            vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
                v_asignacion.asignada_en
            ),
            v_asignacion.motivo_clave
        ]::text[];
    END IF;
    FOREACH v_campo IN ARRAY v_campos LOOP
        v_canon := v_canon
            || vec_contratacion_temporal.encuadrar_valor_rrhh_v1(
                vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
                    v_campo
                )
            );
    END LOOP;

    IF p_entrada.fiscalizacion_presente THEN
        v_campos := ARRAY[
            '1', p_entrada.referencia_fiscalizacion::text,
            p_entrada.referencia_subsanacion::text,
            v_fiscalizacion.resultado_clave,
            CASE WHEN v_fiscalizacion.reparo_clave = '' THEN '0' ELSE '1' END
        ]::text[];
        IF v_fiscalizacion.reparo_clave <> '' THEN
            v_campos := v_campos || ARRAY[
                v_fiscalizacion.reparo_clave, v_fiscalizacion.reparo_texto
            ]::text[];
        END IF;
        v_campos := v_campos || ARRAY[
            vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
                v_fiscalizacion.registrada_en
            ),
            CASE WHEN v_fiscalizacion.subsanacion_registrada_en IS NULL
                 THEN '0' ELSE '1' END
        ]::text[];
        IF v_fiscalizacion.subsanacion_registrada_en IS NOT NULL THEN
            v_campos := v_campos || ARRAY[
                vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
                    v_fiscalizacion.subsanacion_registrada_en
                ), v_fiscalizacion.subsanacion_texto
            ]::text[];
        END IF;
    ELSE
        v_campos := ARRAY['0', '0', '0']::text[];
    END IF;
    FOREACH v_campo IN ARRAY v_campos LOOP
        v_canon := v_canon
            || vec_contratacion_temporal.encuadrar_valor_rrhh_v1(
                vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
                    v_campo
                )
            );
    END LOOP;

    v_campos := ARRAY[v_total_hitos::text]::text[];
    FOREACH v_campo IN ARRAY v_campos LOOP
        v_canon := v_canon
            || vec_contratacion_temporal.encuadrar_valor_rrhh_v1(
                vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
                    v_campo
                )
            );
    END LOOP;

    IF p_entrada.fiscalizacion_presente THEN
        v_subsanacion_presente :=
            v_fiscalizacion.subsanacion_registrada_en IS NOT NULL
            OR v_fiscalizacion.subsanacion_texto IS NOT NULL;
        IF v_fiscalizacion.resultado_clave NOT IN (
               'favorable', 'favorable_con_observaciones', 'desfavorable'
           )
           OR v_fiscalizacion.reparo_clave IS NULL
           OR v_fiscalizacion.reparo_texto IS NULL
           OR v_fiscalizacion.registrada_en IS NULL
           OR (v_fiscalizacion.resultado_clave = 'favorable'
               AND (v_fiscalizacion.reparo_clave <> ''
                    OR v_fiscalizacion.reparo_texto <> ''))
           OR (v_fiscalizacion.resultado_clave <> 'favorable'
               AND (v_fiscalizacion.reparo_clave <> 'observaciones_fiscalizacion'
                    OR pg_catalog.char_length(v_fiscalizacion.reparo_texto)
                       NOT BETWEEN 1 AND 2000))
           OR (v_subsanacion_presente
               AND (v_fiscalizacion.resultado_clave <> 'desfavorable'
                    OR v_fiscalizacion.subsanacion_registrada_en IS NULL
                    OR v_fiscalizacion.subsanacion_texto IS NULL
                    OR pg_catalog.char_length(v_fiscalizacion.subsanacion_texto)
                       NOT BETWEEN 1 AND 2000
                    OR p_entrada.referencia_subsanacion
                       <= p_entrada.referencia_fiscalizacion))
           OR (NOT v_subsanacion_presente
               AND p_entrada.referencia_subsanacion <> 0) THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de detalle RRHH inválido';
        END IF;
        PERFORM vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
            v_fiscalizacion.registrada_en
        );
        IF v_subsanacion_presente THEN
            PERFORM vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
                v_fiscalizacion.subsanacion_registrada_en
            );
            IF v_fiscalizacion.subsanacion_registrada_en
               <= v_fiscalizacion.registrada_en THEN
                RAISE EXCEPTION USING ERRCODE = '22023',
                    MESSAGE = 'contenido de detalle RRHH inválido';
            END IF;
        END IF;
    END IF;

    FOR v_indice IN 1..v_total_hitos LOOP
        v_hito := p_entrada.hitos[v_indice];
        v_campos := ARRAY[
            v_hito.secuencia::text,
            v_hito.version_expediente::text,
            v_hito.accion_clave,
            vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
                v_hito.realizada_en
            ),
            v_hito.fase_origen, v_hito.fase_destino,
            v_hito.estado_origen, v_hito.estado_destino
        ]::text[];
        FOREACH v_campo IN ARRAY v_campos LOOP
            v_canon := v_canon
                || vec_contratacion_temporal.encuadrar_valor_rrhh_v1(
                    vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
                        v_campo
                    )
                );
        END LOOP;
        IF pg_catalog.octet_length(v_canon) > 262144 THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de detalle RRHH inválido';
        END IF;
    END LOOP;
    IF v_derivado THEN
        v_campos := ARRAY[
          v_instruccion->>'derivacion_ref',v_instruccion->>'padre_ref',
          v_instruccion->>'numero_visible_padre',v_instruccion->>'oferta_ref',
          v_instruccion->>'numero_plaza',v_instruccion->>'recibo_derivacion_ref',
          v_instruccion->>'huella_padre_sha256',
          vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
            (v_instruccion->>'creado_en_padre')::timestamptz),
          vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
            (v_instruccion->>'actualizado_en_padre')::timestamptz),
          v_total_instruccion::text
        ];
        FOREACH v_campo IN ARRAY v_campos LOOP
          v_canon:=v_canon||vec_contratacion_temporal.encuadrar_valor_rrhh_v1(
            vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(v_campo));
        END LOOP;
        FOR v_indice IN 1..v_total_instruccion LOOP
          v_hito:=v_hitos_instruccion[v_indice];
          v_campos:=ARRAY[v_hito.secuencia::text,v_hito.version_expediente::text,
            v_hito.accion_clave,vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(v_hito.realizada_en),
            v_hito.fase_origen,v_hito.fase_destino,v_hito.estado_origen,v_hito.estado_destino];
          FOREACH v_campo IN ARRAY v_campos LOOP
            v_canon:=v_canon||vec_contratacion_temporal.encuadrar_valor_rrhh_v1(
              vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(v_campo));
          END LOOP;
        END LOOP;
    END IF;
    IF pg_catalog.octet_length(v_canon)>262144 THEN
      RAISE EXCEPTION 'contenido de detalle RRHH inválido' USING ERRCODE='22023'; END IF;
    RETURN v_canon;
EXCEPTION WHEN OTHERS THEN
    RAISE EXCEPTION USING ERRCODE = '22023',
        MESSAGE = 'contenido de detalle RRHH inválido';
END
$function$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.canon_contenido_detalle_rrhh_v1(timestamp with time zone,vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1) FROM PUBLIC;

-- CT247-GENERADO: materializar_detalle_rrhh_v1
CREATE OR REPLACE FUNCTION vec_contratacion_temporal.materializar_detalle_rrhh_v1(p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1, p_consulta vec_contratacion_temporal.consulta_detalle_rrhh_v1, p_corte_global numeric)
 RETURNS vec_contratacion_temporal.materializacion_detalle_rrhh_v1
 LANGUAGE plpgsql
 STABLE SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '1s'
 SET statement_timeout TO '4s'
 SET idle_in_transaction_session_timeout TO '6s'
AS $function$
DECLARE
    v_fila record;
    v_agregado jsonb;
    v_padre jsonb;
    v_instruccion jsonb;
    v_version_instruccion numeric(20,0);
    v_expediente_instruccion_ref text;
    v_json_seguro boolean;
    v_derivacion record;
    v_nodo jsonb;
    v_indice integer;
    v_total integer;
    v_resumen
        vec_contratacion_temporal.resumen_publicacion_rrhh_v1;
    v_solicitud
        vec_contratacion_temporal.solicitud_operativa_rrhh_v1;
    v_analisis
        vec_contratacion_temporal.analisis_operativo_rrhh_v1;
    v_cobertura
        vec_contratacion_temporal.cobertura_operativa_rrhh_v1;
    v_asignacion
        vec_contratacion_temporal.asignacion_operativa_rrhh_v1;
    v_fiscalizacion
        vec_contratacion_temporal.fiscalizacion_operativa_rrhh_v1;
    v_subsanacion_nodo jsonb;
    v_subsanaciones integer;
    v_comprobaciones
        vec_contratacion_temporal.comprobacion_operativa_rrhh_v1[];
    v_hitos
        vec_contratacion_temporal.hito_expediente_rrhh_v1[];
    v_detalle
        vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1;
    v_analisis_presente boolean;
    v_cobertura_presente boolean;
    v_asignacion_presente boolean;
    v_fiscalizacion_presente boolean;
    v_subsanacion_presente boolean;
    v_decision_gobernada boolean;
    v_coste_presente boolean;
    v_referencia_analisis numeric(20, 0) := 0;
    v_referencia_cobertura numeric(20, 0) := 0;
    v_referencia_asignacion numeric(20, 0) := 0;
    v_referencia_fiscalizacion numeric(20, 0) := 0;
    v_referencia_subsanacion numeric(20, 0) := 0;
BEGIN
    IF CURRENT_USER <> 'vec_contratacion_temporal_propietario'
       OR p_alcance IS NULL
       OR p_consulta IS NULL
       OR p_corte_global IS NULL
       OR p_corte_global NOT BETWEEN
          1 AND 9007199254740991::numeric
       OR p_corte_global <> pg_catalog.trunc(p_corte_global) THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'detalle RRHH no disponible';
    END IF;

    PERFORM vec_contratacion_temporal.canon_alcance_rrhh_v1(
        p_alcance
    );
    PERFORM vec_contratacion_temporal.canon_consulta_detalle_rrhh_v1(
        p_consulta
    );

    -- MATERIALIZED fija primero la versión actual relativa al corte. El
    -- predicado exterior aplica después el ámbito y el control de versión.
    WITH elegida AS MATERIALIZED (
        SELECT publicacion.*, historia.agregado_json
          FROM vec_contratacion_temporal.publicacion_version_rrhh publicacion
          JOIN vec_contratacion_temporal.expediente_version_integral historia
            ON historia.expediente_ref = publicacion.expediente_ref
           AND historia.version = publicacion.version
           AND historia.flujo_ref = publicacion.flujo_ref
           AND historia.flujo_version = publicacion.flujo_version
           AND historia.flujo_huella_sha256 =
               publicacion.flujo_huella_sha256
           AND historia.fase_clave = publicacion.fase_clave
           AND historia.estado = publicacion.estado_clave
           AND historia.agregado_json_huella_sha256 =
               publicacion.agregado_huella_sha256
           AND historia.registrada_en = publicacion.registrada_en
         WHERE publicacion.expediente_ref = p_consulta.expediente_ref
           AND publicacion.corte_global <= p_corte_global
         ORDER BY publicacion.corte_global DESC
         LIMIT 1
    )
    SELECT elegida.*
      INTO STRICT v_fila
      FROM elegida
     WHERE elegida.organizacion_ref = p_alcance.organizacion_ref
       AND (
          NOT EXISTS(SELECT 1 FROM vec_contratacion_temporal.expediente_alta propia_alta
            WHERE propia_alta.expediente_ref=elegida.expediente_ref AND propia_alta.derivacion_ref IS NOT NULL)
          OR (p_alcance.clase_ambito IN ('organizacion','unidad_gestion')
            AND elegida.version=6
            AND EXISTS(SELECT 1 FROM vec_contratacion_temporal.expediente_alta propia_alta
              JOIN vec_contratacion_temporal.nombramiento_derivado_v1 propia_derivacion
                ON propia_derivacion.derivacion_ref=propia_alta.derivacion_ref
               AND propia_derivacion.hijo_ref=propia_alta.expediente_ref
              WHERE propia_alta.expediente_ref=elegida.expediente_ref
                AND propia_alta.organizacion_ref=elegida.organizacion_ref
                AND propia_derivacion.organizacion_ref=elegida.organizacion_ref
                AND propia_derivacion.material_sha256=encode(sha256(convert_to(propia_derivacion.material,'UTF8')),'hex')
                AND propia_derivacion.material::jsonb->>'hijo_ref'=elegida.expediente_ref
                AND propia_derivacion.material::jsonb->>'organizacion_ref'=elegida.organizacion_ref
                AND propia_derivacion.material::jsonb->>'unidad_ref'=elegida.unidad_ref
                AND propia_alta.decision_ref=propia_derivacion.decision_ref
                AND propia_alta.actor_ref=propia_derivacion.actor_ref
                AND propia_alta.perfil_ref=propia_derivacion.perfil_ref))
        )
       AND (
           p_consulta.version_observada = 0
           OR p_consulta.version_observada = elegida.version
       )
       AND CASE p_alcance.clase_ambito
           WHEN 'organizacion' THEN
               elegida.organizacion_ref = p_alcance.ambito_ref
           WHEN 'centro' THEN
               elegida.centro_ref = p_alcance.ambito_ref
           WHEN 'unidad_gestion' THEN
               elegida.unidad_ref = p_alcance.ambito_ref
           ELSE false
       END;

    v_agregado := v_fila.agregado_json;
    v_version_instruccion := v_fila.version;
    v_expediente_instruccion_ref := v_fila.expediente_ref;
    IF v_agregado ? 'instruccion_compartida' THEN
        SELECT derivacion.*, alta.derivacion_ref AS alta_derivacion_ref
          INTO STRICT v_derivacion
          FROM vec_contratacion_temporal.nombramiento_derivado_v1 derivacion
          JOIN vec_contratacion_temporal.expediente_alta alta
            ON alta.expediente_ref=derivacion.hijo_ref
           AND alta.derivacion_ref=derivacion.derivacion_ref
         WHERE derivacion.hijo_ref=v_fila.expediente_ref;
        -- CT234 fija snapshot y preimagen desde el mismo agregado JSONB.
        -- El recibo y la versión v6 conservan esa fuente sin leer el padre actual.
        v_padre := v_agregado #> '{instruccion_compartida,expediente_padre}';
        IF pg_catalog.jsonb_typeof(v_padre)<>'object'
           OR pg_catalog.jsonb_typeof(v_padre->'actuaciones')<>'array'
           OR v_agregado #>> '{instruccion_compartida,oferta_ref}' IS DISTINCT FROM v_derivacion.oferta_ref
           OR v_agregado #>> '{instruccion_compartida,numero_plaza}' IS DISTINCT FROM v_derivacion.numero_plaza::text
           OR v_agregado #>> '{instruccion_compartida,recibo_derivacion_ref}' IS DISTINCT FROM v_derivacion.recibo_ref
           OR pg_catalog.jsonb_typeof(v_agregado#>'{instruccion_compartida,preimagen_padre_json}') IS DISTINCT FROM 'string'
           OR pg_catalog.octet_length(v_agregado#>>'{instruccion_compartida,preimagen_padre_json}') NOT BETWEEN 2 AND 262144
           OR (v_agregado#>>'{instruccion_compartida,preimagen_padre_json}')::jsonb IS DISTINCT FROM v_padre
           OR v_agregado#>>'{instruccion_compartida,preimagen_padre_json}' IS DISTINCT FROM v_padre::text
           OR v_agregado #>> '{instruccion_compartida,huella_padre_sha256}' IS DISTINCT FROM
                pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
                    v_agregado#>>'{instruccion_compartida,preimagen_padre_json}','UTF8')),'hex')
           OR v_derivacion.recibo_json#>'{expediente,instruccion_compartida}' IS DISTINCT FROM
                v_agregado->'instruccion_compartida'
           OR NOT EXISTS(SELECT 1 FROM vec_contratacion_temporal.expediente_version_integral inicial
                WHERE inicial.expediente_ref=v_derivacion.hijo_ref AND inicial.version=6
                  AND inicial.agregado_json=v_derivacion.recibo_json->'expediente')
           OR v_padre->>'referencia' IS DISTINCT FROM v_derivacion.padre_ref
           OR v_padre->>'version' IS DISTINCT FROM v_derivacion.version_padre::text
           OR v_agregado #>> '{actuaciones,0,recibo_ref}' IS DISTINCT FROM v_derivacion.recibo_ref
           OR v_agregado #>> '{actuaciones,0,actor_ref}' IS DISTINCT FROM v_derivacion.actor_ref
           OR v_agregado #>> '{actuaciones,0,accion_clave}' IS DISTINCT FROM 'contratacion_temporal.nombramiento.derivar'
           OR v_agregado->'analisis' IS DISTINCT FROM v_padre->'analisis'
           OR v_agregado->'via_cobertura' IS DISTINCT FROM v_padre->'via_cobertura'
           OR v_agregado->'asignacion' IS DISTINCT FROM v_padre->'asignacion'
           OR v_agregado->'informe_juridico' IS DISTINCT FROM v_padre->'informe_juridico'
           OR v_agregado->'fiscalizacion' IS DISTINCT FROM v_padre->'fiscalizacion'
           OR v_agregado->>'numero_visible' IS DISTINCT FROM
                (v_padre->>'numero_visible')||'-P'||v_derivacion.numero_plaza::text
        THEN RAISE EXCEPTION 'detalle RRHH no disponible' USING ERRCODE='42501'; END IF;
        v_version_instruccion := (v_padre->>'version')::numeric(20,0);
        v_expediente_instruccion_ref := v_derivacion.padre_ref;
        IF v_version_instruccion<6 OR
           pg_catalog.jsonb_array_length(v_padre->'actuaciones')<>v_version_instruccion THEN
           RAISE EXCEPTION 'detalle RRHH no disponible' USING ERRCODE='42501'; END IF;
        v_instruccion := pg_catalog.jsonb_build_object(
          'derivacion_ref',v_derivacion.derivacion_ref,
          'padre_ref',v_derivacion.padre_ref,
          'numero_visible_padre',v_padre->>'numero_visible',
          'oferta_ref',v_derivacion.oferta_ref,
          'numero_plaza',v_derivacion.numero_plaza,
          'recibo_derivacion_ref',v_derivacion.recibo_ref,
          'huella_padre_sha256',v_agregado#>>'{instruccion_compartida,huella_padre_sha256}',
          'creado_en_padre',v_padre->>'creado_en',
          'actualizado_en_padre',v_padre->>'actualizado_en',
          'hitos_padre',(
             SELECT pg_catalog.jsonb_agg(pg_catalog.jsonb_build_object(
               'secuencia',nodo->'secuencia','version_expediente',nodo->'version_expediente',
               'accion_clave',nodo->'accion_clave','realizada_en',nodo->'realizada_en',
               'fase_origen',COALESCE(nodo->>'fase_origen',''),
               'fase_destino',nodo->'fase_destino','estado_origen',nodo->'estado_origen',
               'estado_destino',nodo->'estado_destino') ORDER BY ord)
             FROM pg_catalog.jsonb_array_elements(v_padre->'actuaciones')
               WITH ORDINALITY AS original(nodo,ord)));
    END IF;
    IF v_instruccion IS NULL THEN
        v_json_seguro := pg_catalog.octet_length(v_agregado::text)<=262144
          AND vec_contratacion_temporal.json_rrhh_seguro_v1(v_agregado);
    ELSE
        -- El hijo contiene una copia tipada y otra textual del padre. Cada
        -- fragmento estructural conserva exactamente los controles legados.
        v_json_seguro := pg_catalog.octet_length(v_agregado::text)<=2097152
          AND vec_contratacion_temporal.json_rrhh_seguro_v1(
            v_agregado - 'instruccion_compartida')
          AND vec_contratacion_temporal.json_rrhh_seguro_v1(v_padre);
    END IF;
    IF v_json_seguro IS NOT TRUE
       OR pg_catalog.jsonb_typeof(v_agregado -> 'solicitud') <> 'object'
       OR pg_catalog.jsonb_typeof(
           v_agregado #> '{solicitud,periodo}'
       ) <> 'object'
       OR pg_catalog.jsonb_typeof(v_agregado -> 'actuaciones') <> 'array'
       OR v_agregado ->> 'referencia' IS DISTINCT FROM
          v_fila.expediente_ref
       OR v_agregado ->> 'organizacion_ref' IS DISTINCT FROM
          v_fila.organizacion_ref
       OR v_agregado ->> 'numero_visible' IS DISTINCT FROM
          v_fila.numero_visible
       OR v_agregado ->> 'version' IS DISTINCT FROM
          v_fila.version::text
       OR v_agregado #>> '{flujo,definicion_ref}' IS DISTINCT FROM
          v_fila.flujo_ref
       OR v_agregado #>> '{flujo,version}' IS DISTINCT FROM
          v_fila.flujo_version::text
       OR v_agregado #>> '{flujo,huella_sha256}' IS DISTINCT FROM
          v_fila.flujo_huella_sha256
       OR v_agregado ->> 'fase_actual' IS DISTINCT FROM
          v_fila.fase_clave
       OR v_agregado ->> 'estado_actual' IS DISTINCT FROM
          v_fila.estado_clave THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'detalle RRHH no disponible';
    END IF;

    v_resumen := ROW(
        v_fila.expediente_ref,
        v_fila.organizacion_ref,
        vec_contratacion_temporal.numero_visible_vigente_v1(
            v_fila.expediente_ref, v_fila.numero_visible
        ),
        v_fila.version,
        v_fila.flujo_ref,
        v_fila.flujo_version,
        v_fila.flujo_huella_sha256,
        v_fila.fase_clave,
        v_fila.estado_clave,
        v_fila.centro_ref,
        v_fila.categoria_ref,
        COALESCE(v_fila.modalidad_clave, ''),
        COALESCE(v_fila.unidad_ref, ''),
        v_fila.creado_en,
        v_fila.actualizado_en
    );
    v_solicitud := ROW(
        v_agregado #>> '{solicitud,grupo_subgrupo}',
        v_agregado #>> '{solicitud,motivo_clave}',
        (v_agregado #>> '{solicitud,periodo,inicio}')::timestamptz,
        (v_agregado #>> '{solicitud,periodo,fin}')::timestamptz,
        v_agregado #>> '{solicitud,periodo,causa_fin}'
    );

    v_analisis_presente := v_agregado ? 'analisis';
    IF v_analisis_presente THEN
        IF pg_catalog.jsonb_typeof(v_agregado -> 'analisis') <> 'object'
           OR pg_catalog.jsonb_typeof(
               v_agregado #> '{analisis,periodo}'
           ) <> 'object'
           OR pg_catalog.jsonb_typeof(
               v_agregado #> '{analisis,validacion_rc}'
           ) <> 'object'
           OR pg_catalog.jsonb_typeof(
               v_agregado #> '{analisis,actuacion_registro}'
           ) <> 'object' THEN
            RAISE EXCEPTION USING
                ERRCODE = '42501',
                MESSAGE = 'detalle RRHH no disponible';
        END IF;
        -- Sin coste previsto Go tampoco guarda la fuente del coste: sólo
        -- entonces «hay coste» es false. Con fuente y sin coste sigue NULL y
        -- el canon lo rechaza, como antes de CT180.
        v_coste_presente := COALESCE(
            pg_catalog.jsonb_typeof(
                v_agregado #> '{analisis,coste_previsto}'
            ) = 'object',
            CASE WHEN v_agregado #> '{analisis,fuente_coste_ref}' IS NULL
                 THEN false END
        );
        v_referencia_analisis := (
            v_agregado #>>
            '{analisis,actuacion_registro,version_expediente}'
        )::numeric(20, 0);
        v_analisis := ROW(
            v_agregado #>> '{analisis,modalidad_clave}',
            v_agregado #>> '{analisis,categoria_ref}',
            v_agregado #>> '{analisis,causa_clave}',
            (v_agregado #>> '{analisis,periodo,inicio}')::timestamptz,
            (v_agregado #>> '{analisis,periodo,fin}')::timestamptz,
            (v_agregado #>> '{analisis,porcentaje_jornada}')::smallint,
            v_agregado #>> '{analisis,validacion_rc,resultado}',
            v_coste_presente,
            CASE WHEN v_coste_presente THEN
                (v_agregado #>>
                 '{analisis,coste_previsto,centimos}')::bigint
            ELSE 0::bigint END,
            CASE WHEN v_coste_presente THEN
                v_agregado #>> '{analisis,coste_previsto,moneda}'
            ELSE '' END,
            CASE WHEN v_coste_presente THEN
                v_agregado #>> '{analisis,fuente_coste_ref}'
            ELSE '' END,
            COALESCE(v_agregado #>> '{analisis,observaciones}', ''),
            v_agregado #>> '{analisis,periodo,causa_fin}'
        );
    ELSE
        v_analisis := NULL;
    END IF;

    v_cobertura_presente := v_agregado ? 'via_cobertura';
    IF v_cobertura_presente THEN
        IF pg_catalog.jsonb_typeof(
               v_agregado -> 'via_cobertura'
           ) <> 'object' THEN
            RAISE EXCEPTION USING
                ERRCODE = '42501',
                MESSAGE = 'detalle RRHH no disponible';
        END IF;
        v_decision_gobernada :=
            pg_catalog.jsonb_typeof(
                v_agregado #>
                '{via_cobertura,decision_gobernada}'
            ) = 'object';
        v_comprobaciones := ARRAY[]::vec_contratacion_temporal
            .comprobacion_operativa_rrhh_v1[];
        IF v_decision_gobernada THEN
            IF NOT v_analisis_presente THEN
                RAISE EXCEPTION USING
                    ERRCODE = '42501',
                    MESSAGE = 'detalle RRHH no disponible';
            END IF;
            v_referencia_cobertura := (
                v_agregado #>>
                '{via_cobertura,decision_gobernada,actuacion,version_expediente}'
            )::numeric(20, 0);
            SELECT ARRAY(
                SELECT ROW(
                    evidencia.comprobacion_clave,
                    evidencia.comprobacion_resultado
                )::vec_contratacion_temporal.comprobacion_operativa_rrhh_v1
                  FROM vec_contratacion_temporal.consumo_cobertura_lote lote
                  JOIN vec_contratacion_temporal.consumo_cobertura_evidencia evidencia
                    ON evidencia.lote_ref = lote.lote_ref
                 WHERE lote.preparacion_c1_ref = v_agregado #>>
                       '{via_cobertura,decision_gobernada,preparacion_evidencias_ref}'
                   AND lote.expediente_ref = v_expediente_instruccion_ref
                   AND lote.version_expediente = v_referencia_analisis
                   AND evidencia.expediente_ref = lote.expediente_ref
                   AND evidencia.version_expediente = lote.version_expediente
                 ORDER BY evidencia.posicion
            ) INTO v_comprobaciones;
            IF pg_catalog.cardinality(v_comprobaciones) = 0
               OR pg_catalog.cardinality(v_comprobaciones) > 32 THEN
                RAISE EXCEPTION USING
                    ERRCODE = '42501',
                    MESSAGE = 'detalle RRHH no disponible';
            END IF;
        ELSE
            IF NOT v_analisis_presente
               OR pg_catalog.jsonb_typeof(
                   v_agregado #> '{via_cobertura,comprobaciones}'
               ) <> 'array' THEN
                RAISE EXCEPTION USING
                    ERRCODE = '42501',
                    MESSAGE = 'detalle RRHH no disponible';
            END IF;
            v_referencia_cobertura := v_referencia_analisis + 1;
            v_total := pg_catalog.jsonb_array_length(
                v_agregado #> '{via_cobertura,comprobaciones}'
            );
            FOR v_indice IN 0..v_total - 1 LOOP
                v_nodo := (
                    v_agregado #> '{via_cobertura,comprobaciones}'
                ) -> v_indice;
                v_comprobaciones := pg_catalog.array_append(
                    v_comprobaciones,
                    ROW(
                        v_nodo ->> 'clave',
                        v_nodo ->> 'resultado'
                    )::vec_contratacion_temporal
                        .comprobacion_operativa_rrhh_v1
                );
            END LOOP;
        END IF;
        v_cobertura := ROW(
            v_agregado #>> '{via_cobertura,via_clave}',
            v_decision_gobernada,
            CASE WHEN v_decision_gobernada THEN '' ELSE
                v_agregado #>> '{via_cobertura,procedimiento_ref}' END,
            CASE WHEN v_decision_gobernada THEN '' ELSE
                COALESCE(
                    v_agregado #>> '{via_cobertura,bolsa_ref}', ''
                ) END,
            v_comprobaciones
        );
    ELSE
        v_cobertura := NULL;
    END IF;

    v_asignacion_presente := v_agregado ? 'asignacion';
    IF v_asignacion_presente THEN
        IF pg_catalog.jsonb_typeof(v_agregado -> 'asignacion') <> 'object'
           OR pg_catalog.jsonb_typeof(
               v_agregado #> '{asignacion,actuacion_registro}'
           ) <> 'object' THEN
            RAISE EXCEPTION USING
                ERRCODE = '42501',
                MESSAGE = 'detalle RRHH no disponible';
        END IF;
        v_referencia_asignacion := (
            v_agregado #>>
            '{asignacion,actuacion_registro,version_expediente}'
        )::numeric(20, 0);
        v_asignacion := ROW(
            v_agregado #>> '{asignacion,unidad_ref}',
            (v_agregado #>> '{asignacion,asignada_en}')::timestamptz,
            COALESCE(v_agregado #>> '{asignacion,motivo_clave}', '')
        );
    ELSE
        v_asignacion := NULL;
    END IF;

    v_fiscalizacion_presente := v_agregado ? 'fiscalizacion';
    v_subsanacion_presente := false;
    IF v_fiscalizacion_presente THEN
        IF NOT v_asignacion_presente
           OR pg_catalog.jsonb_typeof(v_agregado -> 'fiscalizacion') <> 'object'
           OR pg_catalog.jsonb_typeof(
               v_agregado #> '{fiscalizacion,actuacion_registro}'
           ) <> 'object'
           OR pg_catalog.jsonb_typeof(
               v_agregado #> '{fiscalizacion,resultado}'
           ) <> 'string'
           OR pg_catalog.jsonb_typeof(
               v_agregado #> '{fiscalizacion,fiscalizada_en}'
           ) <> 'string'
           OR pg_catalog.jsonb_typeof(
               v_agregado #> '{fiscalizacion,observaciones}'
           ) NOT IN ('string', 'null') THEN
            RAISE EXCEPTION USING ERRCODE = '42501',
                MESSAGE = 'detalle RRHH no disponible';
        END IF;
        v_referencia_fiscalizacion := (v_agregado #>>
            '{fiscalizacion,actuacion_registro,secuencia}')::numeric(20, 0);
        IF v_referencia_fiscalizacion <= v_referencia_asignacion
           OR v_referencia_fiscalizacion > v_version_instruccion
           OR v_agregado #>> '{fiscalizacion,resultado}' NOT IN (
                'favorable', 'favorable_con_observaciones', 'desfavorable'
           )
           OR COALESCE(v_agregado #>> '{fiscalizacion,observaciones}', '')
              ~ '[\x00-\x1F]'
           OR pg_catalog.char_length(
                COALESCE(v_agregado #>> '{fiscalizacion,observaciones}', '')
           ) > 2000
           OR (v_agregado #>> '{fiscalizacion,resultado}' = 'favorable'
               AND COALESCE(v_agregado #>> '{fiscalizacion,observaciones}', '') <> '')
           OR (v_agregado #>> '{fiscalizacion,resultado}' <> 'favorable'
               AND pg_catalog.char_length(
                   COALESCE(v_agregado #>> '{fiscalizacion,observaciones}', '')
               ) = 0) THEN
            RAISE EXCEPTION USING ERRCODE = '42501',
                MESSAGE = 'detalle RRHH no disponible';
        END IF;
        v_fiscalizacion := ROW(
            v_agregado #>> '{fiscalizacion,resultado}',
            CASE WHEN v_agregado #>> '{fiscalizacion,resultado}' = 'favorable'
                 THEN '' ELSE 'observaciones_fiscalizacion' END,
            COALESCE(v_agregado #>> '{fiscalizacion,observaciones}', ''),
            (v_agregado #>> '{fiscalizacion,fiscalizada_en}')::timestamptz,
            NULL::timestamptz, NULL::text
        );
        IF v_agregado #>> '{fiscalizacion,resultado}' = 'desfavorable' THEN
            IF pg_catalog.jsonb_typeof(
                   v_agregado #> '{fiscalizacion,retorno}'
               ) <> 'object'
               OR pg_catalog.jsonb_typeof(
                   v_agregado #> '{fiscalizacion,retorno,retorno_ref}'
               ) <> 'string' THEN
                RAISE EXCEPTION USING ERRCODE = '42501',
                    MESSAGE = 'detalle RRHH no disponible';
            END IF;
            SELECT pg_catalog.count(*)
              INTO v_subsanaciones
              FROM pg_catalog.jsonb_array_elements(
                   v_agregado -> 'actuaciones'
              ) nodo
             WHERE nodo ->> 'accion_clave' =
                   'contratacion_temporal.subsanacion_reparos.registrar'
               AND nodo ->> 'retorno_ref' =
                   v_agregado #>> '{fiscalizacion,retorno,retorno_ref}';
            IF v_subsanaciones > 1 THEN
                RAISE EXCEPTION USING ERRCODE = '42501',
                    MESSAGE = 'detalle RRHH no disponible';
            END IF;
            IF v_subsanaciones = 1 THEN
                SELECT nodo INTO STRICT v_subsanacion_nodo
                  FROM pg_catalog.jsonb_array_elements(
                       v_agregado -> 'actuaciones'
                  ) nodo
                 WHERE nodo ->> 'accion_clave' =
                       'contratacion_temporal.subsanacion_reparos.registrar'
                   AND nodo ->> 'retorno_ref' =
                       v_agregado #>> '{fiscalizacion,retorno,retorno_ref}';
                IF pg_catalog.jsonb_typeof(
                       v_subsanacion_nodo -> 'realizada_en'
                   ) <> 'string'
                   OR pg_catalog.jsonb_typeof(
                       v_subsanacion_nodo -> 'observaciones'
                   ) <> 'string'
                   OR pg_catalog.char_length(
                       v_subsanacion_nodo ->> 'observaciones'
                   ) NOT BETWEEN 1 AND 2000
                   OR v_subsanacion_nodo ->> 'observaciones' ~ '[\x00-\x1F]' THEN
                    RAISE EXCEPTION USING ERRCODE = '42501',
                        MESSAGE = 'detalle RRHH no disponible';
                END IF;
                v_referencia_subsanacion := (v_subsanacion_nodo ->>
                    'secuencia')::numeric(20, 0);
                IF v_referencia_subsanacion <= v_referencia_fiscalizacion
                   OR v_referencia_subsanacion > v_version_instruccion THEN
                    RAISE EXCEPTION USING ERRCODE = '42501',
                        MESSAGE = 'detalle RRHH no disponible';
                END IF;
                v_subsanacion_presente := true;
                v_fiscalizacion.subsanacion_registrada_en :=
                    (v_subsanacion_nodo ->> 'realizada_en')::timestamptz;
                v_fiscalizacion.subsanacion_texto :=
                    v_subsanacion_nodo ->> 'observaciones';
            END IF;
        END IF;
    ELSE
        v_fiscalizacion := NULL;
    END IF;

    v_hitos := ARRAY[]::vec_contratacion_temporal
        .hito_expediente_rrhh_v1[];
    v_total := pg_catalog.jsonb_array_length(
        v_agregado -> 'actuaciones'
    );
    IF v_total < 1 OR v_total::numeric + (CASE WHEN v_instruccion IS NULL THEN 0 ELSE 5 END) <> v_fila.version THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'detalle RRHH no disponible';
    END IF;
    FOR v_indice IN 0..v_total - 1 LOOP
        v_nodo := (v_agregado -> 'actuaciones') -> v_indice;
        v_hitos := pg_catalog.array_append(
            v_hitos,
            ROW(
                (v_nodo ->> 'secuencia')::numeric(20, 0),
                (v_nodo ->> 'version_expediente')::numeric(20, 0),
                v_nodo ->> 'accion_clave',
                (v_nodo ->> 'realizada_en')::timestamptz,
                COALESCE(v_nodo ->> 'fase_origen', ''),
                v_nodo ->> 'fase_destino',
                v_nodo ->> 'estado_origen',
                v_nodo ->> 'estado_destino'
            )::vec_contratacion_temporal.hito_expediente_rrhh_v1
        );
    END LOOP;

    v_detalle := ROW(
        v_resumen,
        v_solicitud,
        v_analisis_presente,
        v_analisis,
        v_referencia_analisis,
        v_cobertura_presente,
        v_cobertura,
        v_referencia_cobertura,
        v_asignacion_presente,
        v_asignacion,
        v_referencia_asignacion,
        v_hitos,
        v_fiscalizacion_presente,
        v_fiscalizacion,
        v_referencia_fiscalizacion,
        v_referencia_subsanacion,
        v_instruccion
    );

    -- Reutiliza la validación del canon nominal sin volver a consultar ni
    -- conservar el agregado. El instante de la versión basta para esta
    -- validación estructural; el motor canoniza después su instante real.
    PERFORM
        vec_contratacion_temporal.canon_contenido_detalle_rrhh_v1(
            v_resumen.actualizado_en,
            v_detalle
        );
    RETURN ROW(v_detalle)::
        vec_contratacion_temporal.materializacion_detalle_rrhh_v1;
EXCEPTION
    WHEN SQLSTATE '40001' OR SQLSTATE '40P01'
      OR SQLSTATE '55P03' OR SQLSTATE '57014' THEN
        RAISE;
    WHEN OTHERS THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'detalle RRHH no disponible';
END
$function$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.materializar_detalle_rrhh_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,numeric) FROM PUBLIC;

-- CT247-GENERADO: materializar_cuadro_rrhh_v1
CREATE OR REPLACE FUNCTION vec_contratacion_temporal.materializar_cuadro_rrhh_v1(p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1, p_consulta vec_contratacion_temporal.consulta_cuadro_rrhh_v1, p_estado vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1) RETURNS vec_contratacion_temporal.materializacion_cuadro_rrhh_v1
    LANGUAGE plpgsql SECURITY DEFINER
    SET search_path TO 'pg_catalog', 'pg_temp'
    SET row_security TO 'on'
    SET "TimeZone" TO 'UTC'
    SET lock_timeout TO '1s'
    SET statement_timeout TO '4s'
    SET idle_in_transaction_session_timeout TO '6s'
    AS $_$
DECLARE
    v_candidatas
        vec_contratacion_temporal.resumen_publicacion_rrhh_v1[];
    v_publicadas
        vec_contratacion_temporal.resumen_publicacion_rrhh_v1[];
    v_total integer;
    v_total_publicado integer;
    v_hay_mas boolean;
    v_ultima
        vec_contratacion_temporal.resumen_publicacion_rrhh_v1;
BEGIN
    IF CURRENT_USER <> 'vec_contratacion_temporal_propietario'
       OR p_alcance IS NULL
       OR p_consulta IS NULL
       OR p_estado IS NULL
       OR p_estado.es_continuacion IS NULL
       OR p_estado.corte_global IS NULL
       OR p_estado.corte_global NOT BETWEEN
          0 AND 9007199254740991::numeric
       OR p_estado.corte_global <>
          pg_catalog.trunc(p_estado.corte_global) THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'materialización de cuadro RRHH inválida';
    END IF;

    PERFORM vec_contratacion_temporal.canon_alcance_rrhh_v1(
        p_alcance
    );
    PERFORM vec_contratacion_temporal.canon_consulta_cuadro_rrhh_v1(
        p_consulta
    );

    IF NOT p_estado.es_continuacion AND (
           p_consulta.cursor <> ''
           OR p_estado.pagina_presentada IS DISTINCT FROM 0
           OR p_estado.familia_ref IS NOT NULL
           OR p_estado.token_presentado_huella_sha256 IS NOT NULL
           OR p_estado.acceso_emision_ref IS NOT NULL
           OR p_estado.cursor_emitida_en IS NOT NULL
           OR p_estado.familia_creada_en IS NOT NULL
           OR p_estado.familia_valida_hasta IS NOT NULL
           OR p_estado.ultimo_actualizado_en IS NOT NULL
           OR p_estado.ultimo_expediente_ref IS NOT NULL
       ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'estado inicial de cuadro RRHH inválido';
    END IF;

    IF p_estado.es_continuacion AND (
           p_estado.corte_global < 1
           OR p_consulta.cursor = ''
           OR p_estado.familia_ref IS NULL
           OR p_estado.familia_ref !~
              '^familia:cursor:rrhh:[0-9a-f]{32}$'
           OR p_estado.pagina_presentada IS NULL
           OR p_estado.pagina_presentada NOT BETWEEN
              2 AND 9007199254740991::numeric
           OR p_estado.pagina_presentada <>
              pg_catalog.trunc(p_estado.pagina_presentada)
           OR p_estado.token_presentado_huella_sha256 IS NULL
           OR p_estado.token_presentado_huella_sha256 !~
              '^[0-9a-f]{64}$'
           OR p_estado.token_presentado_huella_sha256 =
              pg_catalog.repeat('0', 64)
           OR p_estado.acceso_emision_ref IS NULL
           OR p_estado.acceso_emision_ref !~
              '^acceso:rrhh:[0-9a-f]{32}$'
           OR p_estado.cursor_emitida_en IS NULL
           OR p_estado.familia_creada_en IS NULL
           OR p_estado.familia_valida_hasta IS NULL
           OR p_estado.ultimo_actualizado_en IS NULL
           OR p_estado.ultimo_expediente_ref IS NULL
           OR p_estado.ultimo_expediente_ref !~
              '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
           OR p_estado.cursor_emitida_en <>
              pg_catalog.date_trunc(
                  'microseconds', p_estado.cursor_emitida_en
              )
           OR p_estado.familia_creada_en <>
              pg_catalog.date_trunc(
                  'microseconds', p_estado.familia_creada_en
              )
           OR p_estado.familia_valida_hasta <>
              pg_catalog.date_trunc(
                  'microseconds', p_estado.familia_valida_hasta
              )
           OR p_estado.ultimo_actualizado_en <>
              pg_catalog.date_trunc(
                  'microseconds', p_estado.ultimo_actualizado_en
              )
           OR p_estado.familia_creada_en >
              p_estado.cursor_emitida_en
           OR p_estado.cursor_emitida_en >=
              p_estado.familia_valida_hasta
           OR p_estado.ultimo_actualizado_en >
              p_estado.cursor_emitida_en
       ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'continuación de cuadro RRHH inválida';
    END IF;

    WITH ultimas AS MATERIALIZED (
        SELECT DISTINCT ON (publicada.expediente_ref COLLATE "C")
               publicada.expediente_ref,
               publicada.organizacion_ref,
               publicada.numero_visible AS numero_visible,
               publicada.version,
               publicada.flujo_ref,
               publicada.flujo_version,
               publicada.flujo_huella_sha256,
               publicada.fase_clave,
               publicada.estado_clave,
               publicada.centro_ref,
               publicada.categoria_ref,
               publicada.modalidad_clave,
               publicada.unidad_ref,
               publicada.creado_en,
               publicada.actualizado_en
          FROM vec_contratacion_temporal.publicacion_version_rrhh publicada
         WHERE publicada.corte_global <= p_estado.corte_global
         ORDER BY publicada.expediente_ref COLLATE "C",
                  publicada.corte_global DESC
    ), filtradas AS MATERIALIZED (
        SELECT ROW(
                   ultima.expediente_ref,
                   ultima.organizacion_ref,
                   COALESCE(numeracion.numero_visible, ultima.numero_visible),
                   ultima.version,
                   ultima.flujo_ref,
                   ultima.flujo_version,
                   ultima.flujo_huella_sha256,
                   ultima.fase_clave,
                   ultima.estado_clave,
                   ultima.centro_ref,
                   ultima.categoria_ref,
                   COALESCE(ultima.modalidad_clave, ''),
                   COALESCE(ultima.unidad_ref, ''),
                   ultima.creado_en,
                   ultima.actualizado_en
               )::vec_contratacion_temporal
                 .resumen_publicacion_rrhh_v1 AS resumen
          FROM ultimas ultima
          LEFT JOIN vec_contratacion_temporal.numeracion_anual_asignada numeracion
            ON numeracion.expediente_ref = ultima.expediente_ref
         WHERE ultima.organizacion_ref COLLATE "C" =
               p_alcance.organizacion_ref COLLATE "C"
       AND (
          NOT EXISTS(SELECT 1 FROM vec_contratacion_temporal.expediente_alta propia_alta
            WHERE propia_alta.expediente_ref=ultima.expediente_ref AND propia_alta.derivacion_ref IS NOT NULL)
          OR (p_alcance.clase_ambito IN ('organizacion','unidad_gestion')
            AND ultima.version=6
            AND EXISTS(SELECT 1 FROM vec_contratacion_temporal.expediente_alta propia_alta
              JOIN vec_contratacion_temporal.nombramiento_derivado_v1 propia_derivacion
                ON propia_derivacion.derivacion_ref=propia_alta.derivacion_ref
               AND propia_derivacion.hijo_ref=propia_alta.expediente_ref
              WHERE propia_alta.expediente_ref=ultima.expediente_ref
                AND propia_alta.organizacion_ref=ultima.organizacion_ref
                AND propia_derivacion.organizacion_ref=ultima.organizacion_ref
                AND propia_derivacion.material_sha256=encode(sha256(convert_to(propia_derivacion.material,'UTF8')),'hex')
                AND propia_derivacion.material::jsonb->>'hijo_ref'=ultima.expediente_ref
                AND propia_derivacion.material::jsonb->>'organizacion_ref'=ultima.organizacion_ref
                AND propia_derivacion.material::jsonb->>'unidad_ref'=ultima.unidad_ref
                AND propia_alta.decision_ref=propia_derivacion.decision_ref
                AND propia_alta.actor_ref=propia_derivacion.actor_ref
                AND propia_alta.perfil_ref=propia_derivacion.perfil_ref))
        )
           AND (
               p_alcance.clase_ambito = 'organizacion'
               OR p_alcance.clase_ambito = 'centro'
                  AND ultima.centro_ref COLLATE "C" =
                      p_alcance.ambito_ref COLLATE "C"
               OR p_alcance.clase_ambito = 'unidad_gestion'
                  AND ultima.unidad_ref IS NOT NULL
                  AND ultima.unidad_ref COLLATE "C" =
                      p_alcance.ambito_ref COLLATE "C"
           )
           AND (
               p_consulta.texto = ''
               OR pg_catalog.left(
                      COALESCE(numeracion.numero_visible, ultima.numero_visible),
                      pg_catalog.char_length(p_consulta.texto)
                  ) COLLATE "C" = p_consulta.texto COLLATE "C"
           )
           AND (
               p_consulta.estado_clave = ''
               OR ultima.estado_clave COLLATE "C" =
                  p_consulta.estado_clave COLLATE "C"
           )
           AND (
               p_consulta.fase_clave = ''
               OR ultima.fase_clave COLLATE "C" =
                  p_consulta.fase_clave COLLATE "C"
           )
           AND (
               NOT p_estado.es_continuacion
               OR ultima.actualizado_en <
                  p_estado.ultimo_actualizado_en
               OR ultima.actualizado_en =
                  p_estado.ultimo_actualizado_en
                  AND ultima.expediente_ref COLLATE "C" <
                      p_estado.ultimo_expediente_ref COLLATE "C"
           )
         ORDER BY ultima.actualizado_en DESC,
                  ultima.expediente_ref COLLATE "C" DESC
         LIMIT p_consulta.limite::integer + 1
    )
    SELECT COALESCE(
               pg_catalog.array_agg(
                   filtrada.resumen
                   ORDER BY (filtrada.resumen).actualizado_en DESC,
                            (filtrada.resumen).expediente_ref
                                COLLATE "C" DESC
               ),
               ARRAY[]::vec_contratacion_temporal
                 .resumen_publicacion_rrhh_v1[]
           )
      INTO STRICT v_candidatas
      FROM filtradas filtrada;

    v_total := pg_catalog.cardinality(v_candidatas);
    v_total_publicado := LEAST(
        v_total, p_consulta.limite::integer
    );
    v_hay_mas := v_total > p_consulta.limite::integer;

    IF v_total_publicado = 0 THEN
        v_publicadas := ARRAY[]::vec_contratacion_temporal
          .resumen_publicacion_rrhh_v1[];
        RETURN ROW(
            v_publicadas,
            false,
            NULL::timestamptz,
            NULL::text
        )::vec_contratacion_temporal.materializacion_cuadro_rrhh_v1;
    END IF;

    v_publicadas := v_candidatas[1:v_total_publicado];
    v_ultima := v_publicadas[v_total_publicado];

    RETURN ROW(
        v_publicadas,
        v_hay_mas,
        v_ultima.actualizado_en,
        v_ultima.expediente_ref
    )::vec_contratacion_temporal.materializacion_cuadro_rrhh_v1;
EXCEPTION
    WHEN SQLSTATE '40001' OR SQLSTATE '40P01'
      OR SQLSTATE '55P03' OR SQLSTATE '57014' THEN
        RAISE;
    WHEN data_exception OR invalid_text_representation
      OR numeric_value_out_of_range OR array_subscript_error THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'materialización de cuadro RRHH inválida';
END
$_$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.materializar_cuadro_rrhh_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1) FROM PUBLIC;

-- CT247-GENERADO: consultar_cambios_expediente_rrhh_atestado_v1
CREATE OR REPLACE FUNCTION vec_contratacion_temporal.consultar_cambios_expediente_rrhh_atestado_v1(p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1, p_consulta vec_contratacion_temporal.consulta_detalle_rrhh_v1, p_capacidad_canonica bytea, p_decision_canonica bytea, p_motivo_canonico bytea, p_contexto_actor_canonico bytea, p_persona_version numeric, p_perfil_version numeric, p_payload_vec_ad_3 bytea, p_sobre_cose_sign_1 bytea, p_evidencia_verificacion bytea, p_raiz_publica_spki bytea) RETURNS TABLE(expediente_ref text, version_expediente numeric, cambios jsonb, recortado boolean, consumo_vec_huella_sha256 text, auditoria_vec_ref text, auditoria_vec_huella_sha256 text, consumida_en timestamp with time zone)
    LANGUAGE plpgsql SECURITY DEFINER
    SET search_path TO 'pg_catalog', 'pg_temp'
    SET row_security TO 'on'
    SET "TimeZone" TO 'UTC'
    SET lock_timeout TO '1s'
    SET statement_timeout TO '4s'
    SET idle_in_transaction_session_timeout TO '6s'
    AS $$
DECLARE
    v_login pg_catalog.pg_roles%ROWTYPE;
    v_capacidad jsonb;
    v_decision jsonb;
    v_contexto_actor jsonb;
    v_consulta_canonica bytea;
    v_contexto_recurso bytea;
    v_contexto_huella text;
    v_material vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3;
    v_consumo vec_contratacion_temporal.evidencia_consumo_nuevo_rrhh_v3;
    v_corte_global numeric(20, 0);
    v_version numeric(20, 0);
    v_cambios jsonb;
    v_recortado boolean;
BEGIN
    SELECT * INTO v_login FROM pg_catalog.pg_roles
     WHERE rolname = SESSION_USER;
    IF CURRENT_USER <> 'vec_contratacion_temporal_propietario'
       OR SESSION_USER = CURRENT_USER
       OR v_login.oid IS NULL OR NOT v_login.rolcanlogin
       OR NOT v_login.rolinherit OR v_login.rolsuper
       OR v_login.rolcreatedb OR v_login.rolcreaterole
       OR v_login.rolreplication OR v_login.rolbypassrls
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_auth_members m
            WHERE m.member = v_login.oid) <> 1
       OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_auth_members m
           JOIN pg_catalog.pg_roles r ON r.oid = m.roleid
           WHERE m.member = v_login.oid
             AND r.rolname = 'vec_contratacion_temporal_consultor_rrhh'
             AND NOT m.admin_option AND m.inherit_option
             AND NOT m.set_option
       )
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m
                   WHERE m.roleid = v_login.oid)
       OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_roles r
            WHERE r.rolname = 'vec_contratacion_temporal_consultor_rrhh'
              AND NOT r.rolcanlogin AND r.rolinherit
              AND NOT r.rolsuper AND NOT r.rolcreatedb
              AND NOT r.rolcreaterole AND NOT r.rolreplication
              AND NOT r.rolbypassrls
       )
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_auth_members m
            WHERE m.member =
                  'vec_contratacion_temporal_consultor_rrhh'::regrole
       )
       OR pg_catalog.pg_is_in_recovery()
       OR pg_catalog.current_setting('transaction_isolation') <> 'serializable'
       OR pg_catalog.current_setting('transaction_read_only') <> 'off'
       OR pg_catalog.current_setting('TimeZone') <> 'UTC'
       OR pg_catalog.current_setting('lock_timeout') = '0'
       OR pg_catalog.current_setting('lock_timeout')::interval > interval '1 second'
       OR pg_catalog.current_setting('statement_timeout') = '0'
       OR pg_catalog.current_setting('statement_timeout')::interval > interval '4 seconds'
       OR pg_catalog.current_setting('idle_in_transaction_session_timeout') = '0'
       OR pg_catalog.current_setting(
           'idle_in_transaction_session_timeout')::interval > interval '6 seconds'
       OR p_alcance IS NULL OR p_consulta IS NULL
       OR pg_catalog.octet_length(COALESCE(p_alcance.organizacion_ref, '')) > 160
       OR pg_catalog.octet_length(COALESCE(p_alcance.clase_ambito, '')) > 16
       OR pg_catalog.octet_length(COALESCE(p_alcance.ambito_ref, '')) > 160
       OR pg_catalog.octet_length(COALESCE(p_consulta.expediente_ref, '')) > 160 THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta de cambios RRHH rechazada';
    END IF;

    -- Límites O(1) antes de construir el material privado y antes de
    -- decodificar JSON o ejecutar los cánones CT40, iguales a CT45.
    IF p_capacidad_canonica IS NULL
       OR pg_catalog.octet_length(p_capacidad_canonica)
          NOT BETWEEN 512 AND 32768
       OR p_decision_canonica IS NULL
       OR pg_catalog.octet_length(p_decision_canonica)
          NOT BETWEEN 1 AND 524288
       OR p_motivo_canonico IS NULL
       OR pg_catalog.octet_length(p_motivo_canonico)
          NOT BETWEEN 1 AND 65536
       OR p_contexto_actor_canonico IS NULL
       OR pg_catalog.octet_length(p_contexto_actor_canonico)
          NOT BETWEEN 1 AND 262144
       OR p_persona_version IS NULL
       OR p_persona_version NOT BETWEEN 1 AND 9007199254740991::numeric
       OR p_persona_version <> pg_catalog.trunc(p_persona_version)
       OR p_perfil_version IS NULL
       OR p_perfil_version NOT BETWEEN 1 AND 9007199254740991::numeric
       OR p_perfil_version <> pg_catalog.trunc(p_perfil_version)
       OR p_payload_vec_ad_3 IS NULL
       OR pg_catalog.octet_length(p_payload_vec_ad_3)
          NOT BETWEEN 1 AND 1048576
       OR p_sobre_cose_sign_1 IS NULL
       OR pg_catalog.octet_length(p_sobre_cose_sign_1)
          NOT BETWEEN 1 AND 1048576
       OR p_evidencia_verificacion IS NULL
       OR pg_catalog.octet_length(p_evidencia_verificacion)
          NOT BETWEEN 1 AND 262144
       OR p_raiz_publica_spki IS NULL
       OR pg_catalog.octet_length(p_raiz_publica_spki) <> 44 THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta de cambios RRHH rechazada';
    END IF;

    v_material := ROW(
        p_capacidad_canonica, p_decision_canonica,
        p_motivo_canonico, p_contexto_actor_canonico,
        p_persona_version, p_perfil_version, p_payload_vec_ad_3,
        p_sobre_cose_sign_1, p_evidencia_verificacion, p_raiz_publica_spki
    )::vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3;
    PERFORM vec_contratacion_temporal.acreditar_contexto_motor_consultas_rrhh_v1(
        p_alcance, v_material
    );
    v_consulta_canonica :=
        vec_contratacion_temporal.canon_consulta_detalle_rrhh_v1(p_consulta);
    v_capacidad := pg_catalog.convert_from(p_capacidad_canonica, 'UTF8')::jsonb;
    v_decision := pg_catalog.convert_from(p_decision_canonica, 'UTF8')::jsonb;
    v_contexto_actor := pg_catalog.convert_from(
        p_contexto_actor_canonico, 'UTF8')::jsonb;

    -- El detalle completo, no una proyección limitada: la historia de
    -- cambios alcanza todos los campos del agregado.
    IF v_decision ->> 'garantia_minima' IS DISTINCT FROM 'alto'
       OR v_decision -> 'campos_permitidos' IS DISTINCT FROM '[]'::jsonb
       OR pg_catalog.left(COALESCE(v_decision ->> 'version_rol_ref', ''), 44) =
          'rol:rrhh_interno_certificado_seguimiento_ct_' THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta de cambios RRHH rechazada';
    END IF;
    v_contexto_recurso := pg_catalog.convert_to(
        '{"ambitos":{"ambito_ref":"' || p_alcance.ambito_ref
        || '","clase_ambito":"' || p_alcance.clase_ambito
        || '","organizacion_ref":"' || p_alcance.organizacion_ref
        || '"},"atributos":{"consulta_dominio":"'
        || 'vec.contratacion_temporal.consulta_rrhh.detalle.v1'
        || '","consulta_huella_sha256":"'
        || pg_catalog.encode(pg_catalog.sha256(v_consulta_canonica), 'hex')
        || '"}}', 'UTF8'
    );
    v_contexto_huella := pg_catalog.encode(
        pg_catalog.sha256(v_contexto_recurso), 'hex');
    IF v_capacidad ->> 'audiencia_consumo' IS DISTINCT FROM
           'vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1'
       OR v_capacidad ->> 'operacion' IS DISTINCT FROM
          'contratacion_temporal.expediente.consultar'
       OR v_capacidad ->> 'efecto_ref' IS DISTINCT FROM
          p_consulta.expediente_ref
       OR v_capacidad ->> 'huella_efecto_sha256' IS DISTINCT FROM v_contexto_huella
       OR v_capacidad ->> 'huella_decision_sha256' IS DISTINCT FROM
          pg_catalog.encode(pg_catalog.sha256(p_decision_canonica), 'hex')
       OR v_decision ->> 'accion' IS DISTINCT FROM
          'contratacion_temporal.expediente.consultar'
       OR v_decision ->> 'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
       OR v_decision ->> 'tipo_recurso' IS DISTINCT FROM
          'expediente_contratacion_temporal'
       OR v_decision ->> 'finalidad' IS DISTINCT FROM
          'tramitacion_expediente_contratacion_temporal'
       OR v_decision ->> 'recurso_ref' IS DISTINCT FROM
          p_consulta.expediente_ref
       OR v_decision ->> 'contexto_recurso_huella_sha256' IS DISTINCT FROM
          v_contexto_huella
       OR v_decision ->> 'principal_id' IS DISTINCT FROM
          v_contexto_actor ->> 'principal_ref'
       OR v_decision ->> 'perfil_activo_ref' IS DISTINCT FROM
          v_contexto_actor ->> 'perfil_activo_ref'
       OR v_contexto_actor ->> 'persona_version' IS DISTINCT FROM
          p_persona_version::text
       OR v_contexto_actor ->> 'perfil_version' IS DISTINCT FROM
          p_perfil_version::text THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta de cambios RRHH rechazada';
    END IF;

    -- AD3 registra y consume la decisión y su auditoría en esta transacción.
    -- Solo después se lee la publicación mínima, sin tocar agregado_json.
    v_consumo :=
        vec_contratacion_temporal.consumir_autorizacion_motor_consultas_rrhh_v1(
            'detalle', v_material);
    IF v_consumo.decision_ref IS DISTINCT FROM v_decision ->> 'decision_ref'
       OR v_consumo.efecto_ref IS DISTINCT FROM p_consulta.expediente_ref
       OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM v_contexto_huella
       OR v_consumo.auditoria_ref IS NULL
       OR v_consumo.auditoria_huella_sha256 IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta de cambios RRHH rechazada';
    END IF;
    SELECT ultimo_corte INTO STRICT v_corte_global
      FROM vec_contratacion_temporal.control_publicacion_rrhh
     WHERE control;
    IF v_corte_global IS NULL OR v_corte_global NOT BETWEEN
       1 AND 9007199254740991::numeric
       OR v_corte_global <> pg_catalog.trunc(v_corte_global) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta de cambios RRHH rechazada';
    END IF;
    SELECT p.version INTO STRICT v_version
      FROM (
        SELECT p.expediente_ref, p.version, p.organizacion_ref,
               p.centro_ref, p.unidad_ref
          FROM vec_contratacion_temporal.publicacion_version_rrhh p
         WHERE p.expediente_ref = p_consulta.expediente_ref
           AND p.corte_global <= v_corte_global
         ORDER BY p.corte_global DESC
         LIMIT 1
      ) p
     WHERE p.organizacion_ref = p_alcance.organizacion_ref
       AND (
          NOT EXISTS(SELECT 1 FROM vec_contratacion_temporal.expediente_alta propia_alta
            WHERE propia_alta.expediente_ref=p.expediente_ref AND propia_alta.derivacion_ref IS NOT NULL)
          OR (p_alcance.clase_ambito IN ('organizacion','unidad_gestion')
            AND p.version=6
            AND EXISTS(SELECT 1 FROM vec_contratacion_temporal.expediente_alta propia_alta
              JOIN vec_contratacion_temporal.nombramiento_derivado_v1 propia_derivacion
                ON propia_derivacion.derivacion_ref=propia_alta.derivacion_ref
               AND propia_derivacion.hijo_ref=propia_alta.expediente_ref
              WHERE propia_alta.expediente_ref=p.expediente_ref
                AND propia_alta.organizacion_ref=p.organizacion_ref
                AND propia_derivacion.organizacion_ref=p.organizacion_ref
                AND propia_derivacion.material_sha256=encode(sha256(convert_to(propia_derivacion.material,'UTF8')),'hex')
                AND propia_derivacion.material::jsonb->>'hijo_ref'=p.expediente_ref
                AND propia_derivacion.material::jsonb->>'organizacion_ref'=p.organizacion_ref
                AND propia_derivacion.material::jsonb->>'unidad_ref'=p.unidad_ref
                AND propia_alta.decision_ref=propia_derivacion.decision_ref
                AND propia_alta.actor_ref=propia_derivacion.actor_ref
                AND propia_alta.perfil_ref=propia_derivacion.perfil_ref))
        )
       AND (p_consulta.version_observada = 0
            OR p_consulta.version_observada = p.version)
       AND CASE p_alcance.clase_ambito
           WHEN 'organizacion' THEN p.organizacion_ref = p_alcance.ambito_ref
           WHEN 'centro' THEN p.centro_ref = p_alcance.ambito_ref
           WHEN 'unidad_gestion' THEN p.unidad_ref = p_alcance.ambito_ref
           ELSE false END;

    -- Versiones consecutivas hasta la publicada. Se entregan como mucho 500
    -- cambios y como mucho 192 KiB de JSON acumulado (la respuesta HTTP
    -- admite 256 KiB); si hay más, «recortado» lo indica.
    WITH candidatos AS (
        SELECT n.version AS version_expediente,
               pg_catalog.to_char(n.registrada_en AT TIME ZONE 'UTC',
                   'YYYY-MM-DD"T"HH24:MI:SS.US"Z"') AS registrada_en,
               n.origen_version, n.operacion_ref,
               d.ruta,
               vec_contratacion_temporal.valor_traza_cambio_v1(d.ruta, d.anterior) AS valor_anterior,
               vec_contratacion_temporal.valor_traza_cambio_v1(d.ruta, d.nuevo) AS valor_nuevo
          FROM vec_contratacion_temporal.expediente_version_integral n
          JOIN vec_contratacion_temporal.expediente_version_integral a
            ON a.expediente_ref = n.expediente_ref AND a.version = n.version - 1
          CROSS JOIN LATERAL (
              SELECT COALESCE(x.ruta, y.ruta) AS ruta, x.valor AS anterior, y.valor AS nuevo
                FROM vec_contratacion_temporal.hojas_instantanea_expediente_v1(a.agregado_json) x
                FULL JOIN vec_contratacion_temporal.hojas_instantanea_expediente_v1(n.agregado_json) y
                  ON y.ruta = x.ruta
               WHERE x.valor IS DISTINCT FROM y.valor
          ) d
         WHERE n.expediente_ref = p_consulta.expediente_ref
           AND n.version <= v_version
         ORDER BY n.version, d.ruta
    ), visibles AS (
        -- Un null explícito y un campo ausente, o dos formas de la misma
        -- fecha, no son un cambio visible; dos valores protegidos distintos sí.
        SELECT c.*, pg_catalog.row_number() OVER w AS n,
               pg_catalog.sum(pg_catalog.octet_length(pg_catalog.to_jsonb(c)::text)) OVER w AS acumulado
          FROM candidatos c
         WHERE c.valor_anterior IS DISTINCT FROM c.valor_nuevo
            OR c.valor_anterior = '*protegido'
        WINDOW w AS (ORDER BY c.version_expediente, c.ruta)
         ORDER BY c.version_expediente, c.ruta
         LIMIT 501
    )
    SELECT COALESCE(pg_catalog.jsonb_agg(pg_catalog.to_jsonb(v) - 'n' - 'acumulado'
                        ORDER BY v.version_expediente, v.ruta)
                        FILTER (WHERE v.n <= 500 AND v.acumulado <= 196608), '[]'::jsonb),
           COALESCE(pg_catalog.bool_or(v.n > 500 OR v.acumulado > 196608), false)
      INTO v_cambios, v_recortado
      FROM visibles v;

    RETURN QUERY SELECT p_consulta.expediente_ref, v_version::numeric, v_cambios, v_recortado,
        v_consumo.consumo_huella_sha256,
        v_consumo.auditoria_ref, v_consumo.auditoria_huella_sha256,
        v_consumo.consumida_en::timestamptz;
EXCEPTION
    WHEN SQLSTATE '40001' OR SQLSTATE '40P01'
      OR SQLSTATE '55P03' OR SQLSTATE '57014' THEN RAISE;
    WHEN OTHERS THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta de cambios RRHH rechazada';
END
$$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.consultar_cambios_expediente_rrhh_atestado_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;

-- CT247-GENERADO: consultar_resumen_seguimiento_rrhh_atestado_v1
CREATE OR REPLACE FUNCTION vec_contratacion_temporal.consultar_resumen_seguimiento_rrhh_atestado_v1(p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1, p_consulta vec_contratacion_temporal.consulta_detalle_rrhh_v1, p_unidad_ref text, p_capacidad_canonica bytea, p_decision_canonica bytea, p_motivo_canonico bytea, p_contexto_actor_canonico bytea, p_persona_version numeric, p_perfil_version numeric, p_payload_vec_ad_3 bytea, p_sobre_cose_sign_1 bytea, p_evidencia_verificacion bytea, p_raiz_publica_spki bytea) RETURNS TABLE(expediente_ref text, version_expediente numeric, organizacion_ref text, unidad_ref text, consumo_vec_huella_sha256 text, auditoria_vec_ref text, auditoria_vec_huella_sha256 text, consumida_en timestamp with time zone)
    LANGUAGE plpgsql SECURITY DEFINER
    SET search_path TO 'pg_catalog', 'pg_temp'
    SET row_security TO 'on'
    SET "TimeZone" TO 'UTC'
    SET lock_timeout TO '1s'
    SET statement_timeout TO '4s'
    SET idle_in_transaction_session_timeout TO '6s'
    AS $_$
DECLARE
    v_login pg_catalog.pg_roles%ROWTYPE;
    v_capacidad jsonb;
    v_decision jsonb;
    v_contexto_actor jsonb;
    v_consulta_canonica bytea;
    v_contexto_recurso bytea;
    v_contexto_huella text;
    v_material vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3;
    v_consumo vec_contratacion_temporal.evidencia_consumo_nuevo_rrhh_v3;
    v_corte_global numeric(20, 0);
    v_fila record;
BEGIN
    SELECT * INTO v_login FROM pg_catalog.pg_roles
     WHERE rolname = SESSION_USER;
    IF CURRENT_USER <> 'vec_contratacion_temporal_propietario'
       OR SESSION_USER = CURRENT_USER
       OR v_login.oid IS NULL OR NOT v_login.rolcanlogin
       OR NOT v_login.rolinherit OR v_login.rolsuper
       OR v_login.rolcreatedb OR v_login.rolcreaterole
       OR v_login.rolreplication OR v_login.rolbypassrls
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_auth_members m
            WHERE m.member = v_login.oid) <> 1
       OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_auth_members m
           JOIN pg_catalog.pg_roles r ON r.oid = m.roleid
           WHERE m.member = v_login.oid
             AND r.rolname = 'vec_contratacion_temporal_consultor_rrhh'
             AND NOT m.admin_option AND m.inherit_option
             AND NOT m.set_option
       )
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m
                   WHERE m.roleid = v_login.oid)
       OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_roles r
            WHERE r.rolname = 'vec_contratacion_temporal_consultor_rrhh'
              AND NOT r.rolcanlogin AND r.rolinherit
              AND NOT r.rolsuper AND NOT r.rolcreatedb
              AND NOT r.rolcreaterole AND NOT r.rolreplication
              AND NOT r.rolbypassrls
       )
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_auth_members m
            WHERE m.member =
                  'vec_contratacion_temporal_consultor_rrhh'::regrole
       )
       OR pg_catalog.pg_is_in_recovery()
       OR pg_catalog.current_setting('transaction_isolation') <> 'serializable'
       OR pg_catalog.current_setting('transaction_read_only') <> 'off'
       OR pg_catalog.current_setting('TimeZone') <> 'UTC'
       OR pg_catalog.current_setting('lock_timeout') = '0'
       OR pg_catalog.current_setting('lock_timeout')::interval > interval '1 second'
       OR pg_catalog.current_setting('statement_timeout') = '0'
       OR pg_catalog.current_setting('statement_timeout')::interval > interval '4 seconds'
       OR pg_catalog.current_setting('idle_in_transaction_session_timeout') = '0'
       OR pg_catalog.current_setting(
           'idle_in_transaction_session_timeout')::interval > interval '6 seconds'
       OR p_alcance IS NULL OR p_consulta IS NULL
       OR pg_catalog.octet_length(COALESCE(p_alcance.organizacion_ref, '')) > 160
       OR pg_catalog.octet_length(COALESCE(p_alcance.clase_ambito, '')) > 16
       OR pg_catalog.octet_length(COALESCE(p_alcance.ambito_ref, '')) > 160
       OR pg_catalog.octet_length(COALESCE(p_consulta.expediente_ref, '')) > 160
       OR p_unidad_ref IS NULL
       OR pg_catalog.octet_length(p_unidad_ref) > 160
       OR p_unidad_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta de seguimiento RRHH rechazada';
    END IF;

    -- Límites O(1) antes de construir el material privado y antes de
    -- decodificar JSON o ejecutar los cánones CT40, iguales a CT45.
    IF p_capacidad_canonica IS NULL
       OR pg_catalog.octet_length(p_capacidad_canonica)
          NOT BETWEEN 512 AND 32768
       OR p_decision_canonica IS NULL
       OR pg_catalog.octet_length(p_decision_canonica)
          NOT BETWEEN 1 AND 524288
       OR p_motivo_canonico IS NULL
       OR pg_catalog.octet_length(p_motivo_canonico)
          NOT BETWEEN 1 AND 65536
       OR p_contexto_actor_canonico IS NULL
       OR pg_catalog.octet_length(p_contexto_actor_canonico)
          NOT BETWEEN 1 AND 262144
       OR p_persona_version IS NULL
       OR p_persona_version NOT BETWEEN 1 AND 9007199254740991::numeric
       OR p_persona_version <> pg_catalog.trunc(p_persona_version)
       OR p_perfil_version IS NULL
       OR p_perfil_version NOT BETWEEN 1 AND 9007199254740991::numeric
       OR p_perfil_version <> pg_catalog.trunc(p_perfil_version)
       OR p_payload_vec_ad_3 IS NULL
       OR pg_catalog.octet_length(p_payload_vec_ad_3)
          NOT BETWEEN 1 AND 1048576
       OR p_sobre_cose_sign_1 IS NULL
       OR pg_catalog.octet_length(p_sobre_cose_sign_1)
          NOT BETWEEN 1 AND 1048576
       OR p_evidencia_verificacion IS NULL
       OR pg_catalog.octet_length(p_evidencia_verificacion)
          NOT BETWEEN 1 AND 262144
       OR p_raiz_publica_spki IS NULL
       OR pg_catalog.octet_length(p_raiz_publica_spki) <> 44 THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta de seguimiento RRHH rechazada';
    END IF;

    v_material := ROW(
        p_capacidad_canonica, p_decision_canonica,
        p_motivo_canonico, p_contexto_actor_canonico,
        p_persona_version, p_perfil_version, p_payload_vec_ad_3,
        p_sobre_cose_sign_1, p_evidencia_verificacion, p_raiz_publica_spki
    )::vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3;
    PERFORM vec_contratacion_temporal.acreditar_contexto_motor_consultas_rrhh_v1(
        p_alcance, v_material
    );
    v_consulta_canonica :=
        vec_contratacion_temporal.canon_consulta_detalle_rrhh_v1(p_consulta);
    v_capacidad := pg_catalog.convert_from(p_capacidad_canonica, 'UTF8')::jsonb;
    v_decision := pg_catalog.convert_from(p_decision_canonica, 'UTF8')::jsonb;
    v_contexto_actor := pg_catalog.convert_from(
        p_contexto_actor_canonico, 'UTF8')::jsonb;

    -- El orden es el canónico V3. Ni un campo más ni uno menos, incluidos
    -- duplicados y permutaciones, dan acceso a esta proyección nominal.
    IF v_decision -> 'campos_permitidos' IS DISTINCT FROM
       '["actuaciones","alcance","eficacia_administrativa",'
       '"ejercicio_sintetico","esquema","estado_clave",'
       '"expediente_ref","firma_oficial","periodo",'
       '"recibo_incorporacion_ref","registrado_en","seguimiento_ref",'
       '"version_expediente","version_seguimiento"]'::jsonb THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta de seguimiento RRHH rechazada';
    END IF;
    v_contexto_recurso := pg_catalog.convert_to(
        '{"ambitos":{"ambito_ref":"' || p_alcance.ambito_ref
        || '","clase_ambito":"' || p_alcance.clase_ambito
        || '","organizacion_ref":"' || p_alcance.organizacion_ref
        || '"},"atributos":{"consulta_dominio":"'
        || 'vec.contratacion_temporal.consulta_rrhh.detalle.v1'
        || '","consulta_huella_sha256":"'
        || pg_catalog.encode(pg_catalog.sha256(v_consulta_canonica), 'hex')
        || '"}}', 'UTF8'
    );
    v_contexto_huella := pg_catalog.encode(
        pg_catalog.sha256(v_contexto_recurso), 'hex');
    IF v_capacidad ->> 'audiencia_consumo' IS DISTINCT FROM
           'vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1'
       OR v_capacidad ->> 'operacion' IS DISTINCT FROM
          'contratacion_temporal.expediente.consultar'
       OR v_capacidad ->> 'efecto_ref' IS DISTINCT FROM
          p_consulta.expediente_ref
       OR v_capacidad ->> 'huella_efecto_sha256' IS DISTINCT FROM v_contexto_huella
       OR v_capacidad ->> 'huella_decision_sha256' IS DISTINCT FROM
          pg_catalog.encode(pg_catalog.sha256(p_decision_canonica), 'hex')
       OR v_decision ->> 'accion' IS DISTINCT FROM
          'contratacion_temporal.expediente.consultar'
       OR v_decision ->> 'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
       OR v_decision ->> 'tipo_recurso' IS DISTINCT FROM
          'expediente_contratacion_temporal'
       OR v_decision ->> 'finalidad' IS DISTINCT FROM
          'tramitacion_expediente_contratacion_temporal'
       OR v_decision ->> 'recurso_ref' IS DISTINCT FROM
          p_consulta.expediente_ref
       OR v_decision ->> 'contexto_recurso_huella_sha256' IS DISTINCT FROM
          v_contexto_huella
       OR v_decision ->> 'principal_id' IS DISTINCT FROM
          v_contexto_actor ->> 'principal_ref'
       OR v_decision ->> 'perfil_activo_ref' IS DISTINCT FROM
          v_contexto_actor ->> 'perfil_activo_ref'
       OR v_contexto_actor ->> 'persona_version' IS DISTINCT FROM
          p_persona_version::text
       OR v_contexto_actor ->> 'perfil_version' IS DISTINCT FROM
          p_perfil_version::text THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta de seguimiento RRHH rechazada';
    END IF;

    -- AD3 registra y consume la decisión y su auditoría en esta transacción.
    -- Solo después se lee la publicación mínima, sin tocar agregado_json.
    v_consumo :=
        vec_contratacion_temporal.consumir_autorizacion_motor_consultas_rrhh_v1(
            'detalle', v_material);
    IF v_consumo.decision_ref IS DISTINCT FROM v_decision ->> 'decision_ref'
       OR v_consumo.efecto_ref IS DISTINCT FROM p_consulta.expediente_ref
       OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM v_contexto_huella
       OR v_consumo.auditoria_ref IS NULL
       OR v_consumo.auditoria_huella_sha256 IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta de seguimiento RRHH rechazada';
    END IF;
    SELECT ultimo_corte INTO STRICT v_corte_global
      FROM vec_contratacion_temporal.control_publicacion_rrhh
     WHERE control;
    IF v_corte_global IS NULL OR v_corte_global NOT BETWEEN
       1 AND 9007199254740991::numeric
       OR v_corte_global <> pg_catalog.trunc(v_corte_global) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta de seguimiento RRHH rechazada';
    END IF;
    WITH elegida AS MATERIALIZED (
        SELECT p.expediente_ref, p.version, p.organizacion_ref,
               p.centro_ref, p.unidad_ref
          FROM vec_contratacion_temporal.publicacion_version_rrhh p
         WHERE p.expediente_ref = p_consulta.expediente_ref
           AND p.corte_global <= v_corte_global
         ORDER BY p.corte_global DESC
         LIMIT 1
    )
    SELECT e.expediente_ref, e.version, e.organizacion_ref, e.unidad_ref
      INTO STRICT v_fila
      FROM elegida e
     WHERE e.organizacion_ref = p_alcance.organizacion_ref
       AND (
          NOT EXISTS(SELECT 1 FROM vec_contratacion_temporal.expediente_alta propia_alta
            WHERE propia_alta.expediente_ref=e.expediente_ref AND propia_alta.derivacion_ref IS NOT NULL)
          OR (p_alcance.clase_ambito IN ('organizacion','unidad_gestion')
            AND e.version=6
            AND EXISTS(SELECT 1 FROM vec_contratacion_temporal.expediente_alta propia_alta
              JOIN vec_contratacion_temporal.nombramiento_derivado_v1 propia_derivacion
                ON propia_derivacion.derivacion_ref=propia_alta.derivacion_ref
               AND propia_derivacion.hijo_ref=propia_alta.expediente_ref
              WHERE propia_alta.expediente_ref=e.expediente_ref
                AND propia_alta.organizacion_ref=e.organizacion_ref
                AND propia_derivacion.organizacion_ref=e.organizacion_ref
                AND propia_derivacion.material_sha256=encode(sha256(convert_to(propia_derivacion.material,'UTF8')),'hex')
                AND propia_derivacion.material::jsonb->>'hijo_ref'=e.expediente_ref
                AND propia_derivacion.material::jsonb->>'organizacion_ref'=e.organizacion_ref
                AND propia_derivacion.material::jsonb->>'unidad_ref'=e.unidad_ref
                AND propia_alta.decision_ref=propia_derivacion.decision_ref
                AND propia_alta.actor_ref=propia_derivacion.actor_ref
                AND propia_alta.perfil_ref=propia_derivacion.perfil_ref))
        )
       AND e.unidad_ref = p_unidad_ref
       AND (p_consulta.version_observada = 0
            OR p_consulta.version_observada = e.version)
       AND CASE p_alcance.clase_ambito
           WHEN 'organizacion' THEN e.organizacion_ref = p_alcance.ambito_ref
           WHEN 'centro' THEN e.centro_ref = p_alcance.ambito_ref
           WHEN 'unidad_gestion' THEN e.unidad_ref = p_alcance.ambito_ref
           ELSE false END;
    RETURN QUERY SELECT v_fila.expediente_ref::text, v_fila.version::numeric,
        v_fila.organizacion_ref::text, v_fila.unidad_ref::text,
        v_consumo.consumo_huella_sha256,
        v_consumo.auditoria_ref, v_consumo.auditoria_huella_sha256,
        v_consumo.consumida_en::timestamptz;
EXCEPTION
    WHEN SQLSTATE '40001' OR SQLSTATE '40P01'
      OR SQLSTATE '55P03' OR SQLSTATE '57014' THEN RAISE;
    WHEN OTHERS THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta de seguimiento RRHH rechazada';
END
$_$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.consultar_resumen_seguimiento_rrhh_atestado_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;

-- CT247-GENERADO: consultar_estadisticas_rrhh_v1
CREATE OR REPLACE FUNCTION vec_contratacion_temporal.consultar_estadisticas_rrhh_v1(p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1, p_periodo text, p_desde date, p_hasta date) RETURNS TABLE(corte_global numeric, inicio date, altas numeric, llamamientos numeric, formalizaciones numeric, cierres numeric, incidencias numeric)
    LANGUAGE plpgsql STABLE SECURITY DEFINER
    SET search_path TO 'pg_catalog', 'pg_temp'
    SET row_security TO 'on'
    SET "TimeZone" TO 'UTC'
    SET lock_timeout TO '1s'
    SET statement_timeout TO '4s'
    SET idle_in_transaction_session_timeout TO '6s'
    AS $$
DECLARE
    v_corte_global numeric;
    v_unidad text;
    v_paso interval;
    v_dias integer;
    v_inicio date;
    v_fin date;
BEGIN
    IF CURRENT_USER <> 'vec_contratacion_temporal_propietario'
       OR p_alcance IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'estadísticas RRHH no disponibles';
    END IF;
    PERFORM vec_contratacion_temporal.canon_alcance_rrhh_v1(p_alcance);
    IF p_periodo IS NULL OR p_desde IS NULL OR p_hasta IS NULL
       OR p_periodo NOT IN ('anual', 'mensual', 'semanal')
       OR p_desde > p_hasta
       OR p_desde < DATE '2000-01-01' OR p_hasta > DATE '2100-12-31' THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'consulta de estadísticas RRHH inválida';
    END IF;
    CASE p_periodo
        WHEN 'anual' THEN v_unidad := 'year'; v_paso := INTERVAL '1 year'; v_dias := 365;
        WHEN 'mensual' THEN v_unidad := 'month'; v_paso := INTERVAL '1 month'; v_dias := 28;
        ELSE v_unidad := 'week'; v_paso := INTERVAL '1 week'; v_dias := 7;
    END CASE;
    v_inicio := pg_catalog.date_trunc(v_unidad, p_desde::timestamp)::date;
    v_fin := pg_catalog.date_trunc(v_unidad, p_hasta::timestamp)::date;
    IF (v_fin - v_inicio) / v_dias > 400 THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'intervalo de estadísticas RRHH demasiado amplio';
    END IF;
    SELECT ultimo_corte INTO STRICT v_corte_global
      FROM vec_contratacion_temporal.control_publicacion_rrhh
     WHERE control;
    IF v_corte_global IS NULL OR v_corte_global NOT BETWEEN
       0 AND 9007199254740991::numeric OR
       v_corte_global <> pg_catalog.trunc(v_corte_global) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'corte de cuadro RRHH no disponible';
    END IF;
    -- Mantener el predicado de alcance alineado con
    -- contar_totales_cuadro_rrhh_v1 (000105).
    RETURN QUERY
    WITH versiones AS MATERIALIZED (
        SELECT publicada.expediente_ref, publicada.version,
               publicada.fase_clave, publicada.estado_clave,
               publicada.creado_en, publicada.actualizado_en,
               pg_catalog.lag(publicada.estado_clave) OVER (
                   PARTITION BY publicada.expediente_ref COLLATE "C"
                   ORDER BY publicada.version
               ) AS estado_anterior
          FROM vec_contratacion_temporal.publicacion_version_rrhh publicada
         WHERE publicada.corte_global <= v_corte_global
           AND publicada.organizacion_ref COLLATE "C" = p_alcance.organizacion_ref COLLATE "C"
       AND (
          NOT EXISTS(SELECT 1 FROM vec_contratacion_temporal.expediente_alta propia_alta
            WHERE propia_alta.expediente_ref=publicada.expediente_ref AND propia_alta.derivacion_ref IS NOT NULL)
          OR (p_alcance.clase_ambito IN ('organizacion','unidad_gestion')
            AND publicada.version=6
            AND EXISTS(SELECT 1 FROM vec_contratacion_temporal.expediente_alta propia_alta
              JOIN vec_contratacion_temporal.nombramiento_derivado_v1 propia_derivacion
                ON propia_derivacion.derivacion_ref=propia_alta.derivacion_ref
               AND propia_derivacion.hijo_ref=propia_alta.expediente_ref
              WHERE propia_alta.expediente_ref=publicada.expediente_ref
                AND propia_alta.organizacion_ref=publicada.organizacion_ref
                AND propia_derivacion.organizacion_ref=publicada.organizacion_ref
                AND propia_derivacion.material_sha256=encode(sha256(convert_to(propia_derivacion.material,'UTF8')),'hex')
                AND propia_derivacion.material::jsonb->>'hijo_ref'=publicada.expediente_ref
                AND propia_derivacion.material::jsonb->>'organizacion_ref'=publicada.organizacion_ref
                AND propia_derivacion.material::jsonb->>'unidad_ref'=publicada.unidad_ref
                AND propia_alta.decision_ref=propia_derivacion.decision_ref
                AND propia_alta.actor_ref=propia_derivacion.actor_ref
                AND propia_alta.perfil_ref=propia_derivacion.perfil_ref))
        )
           AND (p_alcance.clase_ambito = 'organizacion'
                OR p_alcance.clase_ambito = 'centro' AND publicada.centro_ref COLLATE "C" = p_alcance.ambito_ref COLLATE "C"
                OR p_alcance.clase_ambito = 'unidad_gestion' AND publicada.unidad_ref IS NOT NULL AND publicada.unidad_ref COLLATE "C" = p_alcance.ambito_ref COLLATE "C")
    ), hitos AS MATERIALIZED (
        SELECT 'alta' AS concepto, pg_catalog.min(v.creado_en) AS instante
          FROM versiones v GROUP BY v.expediente_ref
        UNION ALL
        SELECT 'llamamiento', pg_catalog.min(v.actualizado_en)
          FROM versiones v WHERE 'llamamiento' = v.fase_clave GROUP BY v.expediente_ref
        UNION ALL
        SELECT 'formalizacion', pg_catalog.min(v.actualizado_en)
          FROM versiones v WHERE 'nombramiento' = v.fase_clave GROUP BY v.expediente_ref
        UNION ALL
        SELECT 'cierre', pg_catalog.min(v.actualizado_en)
          FROM versiones v WHERE v.estado_clave = ANY (ARRAY['completado', 'cancelado']) GROUP BY v.expediente_ref
        UNION ALL
        SELECT 'incidencia', v.actualizado_en
          FROM versiones v
         WHERE 'incidencia' = v.estado_clave
           AND (v.estado_anterior IS NULL OR v.estado_anterior <> 'incidencia')
    ), periodos AS (
        SELECT pg_catalog.date_trunc(v_unidad, serie)::date AS inicio
          FROM pg_catalog.generate_series(v_inicio::timestamp, v_fin::timestamp, v_paso) AS serie
    ), agrupados AS (
        SELECT pg_catalog.date_trunc(v_unidad, h.instante AT TIME ZONE 'Europe/Madrid')::date AS inicio,
               h.concepto, pg_catalog.count(*) AS cuenta
          FROM hitos h
         WHERE h.instante IS NOT NULL
         GROUP BY 1, 2
    )
    SELECT v_corte_global,
           p.inicio,
           COALESCE(pg_catalog.sum(a.cuenta) FILTER (WHERE a.concepto = 'alta'), 0)::numeric,
           COALESCE(pg_catalog.sum(a.cuenta) FILTER (WHERE a.concepto = 'llamamiento'), 0)::numeric,
           COALESCE(pg_catalog.sum(a.cuenta) FILTER (WHERE a.concepto = 'formalizacion'), 0)::numeric,
           COALESCE(pg_catalog.sum(a.cuenta) FILTER (WHERE a.concepto = 'cierre'), 0)::numeric,
           COALESCE(pg_catalog.sum(a.cuenta) FILTER (WHERE a.concepto = 'incidencia'), 0)::numeric
      FROM periodos p
      LEFT JOIN agrupados a ON a.inicio = p.inicio
     GROUP BY p.inicio
     ORDER BY p.inicio;
END
$$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.consultar_estadisticas_rrhh_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,text,date,date) FROM PUBLIC;

-- CT247-GENERADO: contar_totales_cuadro_rrhh_v1
CREATE OR REPLACE FUNCTION vec_contratacion_temporal.contar_totales_cuadro_rrhh_v1(p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1, p_consulta vec_contratacion_temporal.consulta_cuadro_rrhh_v1, p_cursor text) RETURNS TABLE(total_filtrado numeric, en_tramitacion numeric, con_incidencia numeric, en_llamamiento numeric)
    LANGUAGE plpgsql STABLE SECURITY DEFINER
    SET search_path TO 'pg_catalog', 'pg_temp'
    SET row_security TO 'on'
    SET "TimeZone" TO 'UTC'
    SET lock_timeout TO '1s'
    SET statement_timeout TO '4s'
    SET idle_in_transaction_session_timeout TO '6s'
    AS $$
DECLARE
    v_corte_global numeric;
BEGIN
    IF CURRENT_USER <> 'vec_contratacion_temporal_propietario'
       OR p_alcance IS NULL OR p_consulta IS NULL
       OR p_cursor IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'totales de cuadro RRHH no disponibles';
    END IF;
    PERFORM vec_contratacion_temporal.canon_alcance_rrhh_v1(p_alcance);
    PERFORM vec_contratacion_temporal.canon_consulta_cuadro_rrhh_v1(p_consulta);
    IF p_cursor = '' THEN
        SELECT ultimo_corte INTO STRICT v_corte_global
          FROM vec_contratacion_temporal.control_publicacion_rrhh
         WHERE control;
    ELSE
        SELECT familia.corte_global INTO STRICT v_corte_global
          FROM vec_contratacion_temporal.cursor_cuadro_rrhh cursor
          JOIN vec_contratacion_temporal.familia_cursor_cuadro_rrhh familia
            USING (familia_ref)
         WHERE cursor.token_huella_sha256 = pg_catalog.encode(
             pg_catalog.sha256(pg_catalog.convert_to(p_cursor, 'UTF8')), 'hex'
         );
    END IF;
    IF v_corte_global IS NULL OR v_corte_global NOT BETWEEN
       0 AND 9007199254740991::numeric OR
       v_corte_global <> pg_catalog.trunc(v_corte_global) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'corte de cuadro RRHH no disponible';
    END IF;
    -- Mantener estos predicados alineados con materializar_cuadro_rrhh_v1
    -- (CT44): ésta cuenta todo el corte, aquélla ordena y pagina.
    RETURN QUERY
    WITH ultimas AS MATERIALIZED (
        SELECT DISTINCT ON (publicada.expediente_ref COLLATE "C")
               publicada.expediente_ref, publicada.version, publicada.organizacion_ref,
               publicada.numero_visible AS numero_visible, publicada.fase_clave,
               publicada.estado_clave, publicada.centro_ref, publicada.unidad_ref
          FROM vec_contratacion_temporal.publicacion_version_rrhh publicada
         WHERE publicada.corte_global <= v_corte_global
         ORDER BY publicada.expediente_ref COLLATE "C", publicada.corte_global DESC
    ), filtradas AS MATERIALIZED (
        SELECT ultima.estado_clave, ultima.fase_clave
          FROM ultimas ultima
          LEFT JOIN vec_contratacion_temporal.numeracion_anual_asignada numeracion
            ON numeracion.expediente_ref = ultima.expediente_ref
         WHERE ultima.organizacion_ref COLLATE "C" = p_alcance.organizacion_ref COLLATE "C"
       AND (
          NOT EXISTS(SELECT 1 FROM vec_contratacion_temporal.expediente_alta propia_alta
            WHERE propia_alta.expediente_ref=ultima.expediente_ref AND propia_alta.derivacion_ref IS NOT NULL)
          OR (p_alcance.clase_ambito IN ('organizacion','unidad_gestion')
            AND ultima.version=6
            AND EXISTS(SELECT 1 FROM vec_contratacion_temporal.expediente_alta propia_alta
              JOIN vec_contratacion_temporal.nombramiento_derivado_v1 propia_derivacion
                ON propia_derivacion.derivacion_ref=propia_alta.derivacion_ref
               AND propia_derivacion.hijo_ref=propia_alta.expediente_ref
              WHERE propia_alta.expediente_ref=ultima.expediente_ref
                AND propia_alta.organizacion_ref=ultima.organizacion_ref
                AND propia_derivacion.organizacion_ref=ultima.organizacion_ref
                AND propia_derivacion.material_sha256=encode(sha256(convert_to(propia_derivacion.material,'UTF8')),'hex')
                AND propia_derivacion.material::jsonb->>'hijo_ref'=ultima.expediente_ref
                AND propia_derivacion.material::jsonb->>'organizacion_ref'=ultima.organizacion_ref
                AND propia_derivacion.material::jsonb->>'unidad_ref'=ultima.unidad_ref
                AND propia_alta.decision_ref=propia_derivacion.decision_ref
                AND propia_alta.actor_ref=propia_derivacion.actor_ref
                AND propia_alta.perfil_ref=propia_derivacion.perfil_ref))
        )
           AND (p_alcance.clase_ambito = 'organizacion'
                OR p_alcance.clase_ambito = 'centro' AND ultima.centro_ref COLLATE "C" = p_alcance.ambito_ref COLLATE "C"
                OR p_alcance.clase_ambito = 'unidad_gestion' AND ultima.unidad_ref IS NOT NULL AND ultima.unidad_ref COLLATE "C" = p_alcance.ambito_ref COLLATE "C")
           AND (p_consulta.texto = '' OR pg_catalog.left(COALESCE(numeracion.numero_visible, ultima.numero_visible), pg_catalog.char_length(p_consulta.texto)) COLLATE "C" = p_consulta.texto COLLATE "C")
           AND (p_consulta.estado_clave = '' OR ultima.estado_clave COLLATE "C" = p_consulta.estado_clave COLLATE "C")
           AND (p_consulta.fase_clave = '' OR ultima.fase_clave COLLATE "C" = p_consulta.fase_clave COLLATE "C")
    )
    SELECT pg_catalog.count(*)::numeric,
           pg_catalog.count(*) FILTER (WHERE 'en_curso' = estado_clave)::numeric,
           pg_catalog.count(*) FILTER (WHERE 'incidencia' = estado_clave)::numeric,
           pg_catalog.count(*) FILTER (WHERE 'llamamiento' = fase_clave)::numeric
      FROM filtradas;
END
$$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.contar_totales_cuadro_rrhh_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,text) FROM PUBLIC;

-- CT247-GENERADO: contar_resumen_cuadro_rrhh_v1
CREATE OR REPLACE FUNCTION vec_contratacion_temporal.contar_resumen_cuadro_rrhh_v1(p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1, p_consulta vec_contratacion_temporal.consulta_cuadro_rrhh_v1, p_cursor text) RETURNS TABLE(clase text, estado_clave text, fase_clave text, fase_desde timestamp with time zone, urgente boolean, numero numeric, captura jsonb)
    LANGUAGE plpgsql STABLE SECURITY DEFINER
    SET search_path TO 'pg_catalog', 'pg_temp'
    SET row_security TO 'on'
    SET "TimeZone" TO 'UTC'
    SET lock_timeout TO '1s'
    SET statement_timeout TO '4s'
    SET idle_in_transaction_session_timeout TO '6s'
    AS $$
DECLARE
    v_corte_global numeric;
BEGIN
    IF CURRENT_USER <> 'vec_contratacion_temporal_propietario'
       OR p_alcance IS NULL OR p_consulta IS NULL
       OR p_cursor IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'resumen de cuadro RRHH no disponible';
    END IF;
    PERFORM vec_contratacion_temporal.canon_alcance_rrhh_v1(p_alcance);
    PERFORM vec_contratacion_temporal.canon_consulta_cuadro_rrhh_v1(p_consulta);
    IF p_cursor = '' THEN
        SELECT ultimo_corte INTO STRICT v_corte_global
          FROM vec_contratacion_temporal.control_publicacion_rrhh
         WHERE control;
    ELSE
        SELECT familia.corte_global INTO STRICT v_corte_global
          FROM vec_contratacion_temporal.cursor_cuadro_rrhh cursor
          JOIN vec_contratacion_temporal.familia_cursor_cuadro_rrhh familia
            USING (familia_ref)
         WHERE cursor.token_huella_sha256 = pg_catalog.encode(
             pg_catalog.sha256(pg_catalog.convert_to(p_cursor, 'UTF8')), 'hex'
         );
    END IF;
    IF v_corte_global IS NULL OR v_corte_global NOT BETWEEN
       0 AND 9007199254740991::numeric OR
       v_corte_global <> pg_catalog.trunc(v_corte_global) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'corte de cuadro RRHH no disponible';
    END IF;
    RETURN QUERY
    WITH ultimas AS MATERIALIZED (
        SELECT DISTINCT ON (publicada.expediente_ref COLLATE "C")
               publicada.expediente_ref, publicada.version,
               publicada.organizacion_ref,
               publicada.numero_visible AS numero_visible, publicada.fase_clave,
               publicada.estado_clave, publicada.centro_ref, publicada.unidad_ref
          FROM vec_contratacion_temporal.publicacion_version_rrhh publicada
         WHERE publicada.corte_global <= v_corte_global
         ORDER BY publicada.expediente_ref COLLATE "C", publicada.corte_global DESC
    ), filtradas AS MATERIALIZED (
        SELECT ultima.expediente_ref, ultima.version,
               ultima.estado_clave, ultima.fase_clave
          FROM ultimas ultima
          LEFT JOIN vec_contratacion_temporal.numeracion_anual_asignada numeracion
            ON numeracion.expediente_ref = ultima.expediente_ref
         WHERE ultima.organizacion_ref COLLATE "C" = p_alcance.organizacion_ref COLLATE "C"
       AND (
          NOT EXISTS(SELECT 1 FROM vec_contratacion_temporal.expediente_alta propia_alta
            WHERE propia_alta.expediente_ref=ultima.expediente_ref AND propia_alta.derivacion_ref IS NOT NULL)
          OR (p_alcance.clase_ambito IN ('organizacion','unidad_gestion')
            AND ultima.version=6
            AND EXISTS(SELECT 1 FROM vec_contratacion_temporal.expediente_alta propia_alta
              JOIN vec_contratacion_temporal.nombramiento_derivado_v1 propia_derivacion
                ON propia_derivacion.derivacion_ref=propia_alta.derivacion_ref
               AND propia_derivacion.hijo_ref=propia_alta.expediente_ref
              WHERE propia_alta.expediente_ref=ultima.expediente_ref
                AND propia_alta.organizacion_ref=ultima.organizacion_ref
                AND propia_derivacion.organizacion_ref=ultima.organizacion_ref
                AND propia_derivacion.material_sha256=encode(sha256(convert_to(propia_derivacion.material,'UTF8')),'hex')
                AND propia_derivacion.material::jsonb->>'hijo_ref'=ultima.expediente_ref
                AND propia_derivacion.material::jsonb->>'organizacion_ref'=ultima.organizacion_ref
                AND propia_derivacion.material::jsonb->>'unidad_ref'=ultima.unidad_ref
                AND propia_alta.decision_ref=propia_derivacion.decision_ref
                AND propia_alta.actor_ref=propia_derivacion.actor_ref
                AND propia_alta.perfil_ref=propia_derivacion.perfil_ref))
        )
           AND (p_alcance.clase_ambito = 'organizacion'
                OR p_alcance.clase_ambito = 'centro' AND ultima.centro_ref COLLATE "C" = p_alcance.ambito_ref COLLATE "C"
                OR p_alcance.clase_ambito = 'unidad_gestion' AND ultima.unidad_ref IS NOT NULL AND ultima.unidad_ref COLLATE "C" = p_alcance.ambito_ref COLLATE "C")
           AND (p_consulta.texto = '' OR pg_catalog.left(COALESCE(numeracion.numero_visible, ultima.numero_visible), pg_catalog.char_length(p_consulta.texto)) COLLATE "C" = p_consulta.texto COLLATE "C")
           AND (p_consulta.estado_clave = '' OR ultima.estado_clave COLLATE "C" = p_consulta.estado_clave COLLATE "C")
           AND (p_consulta.fase_clave = '' OR ultima.fase_clave COLLATE "C" = p_consulta.fase_clave COLLATE "C")
    )
    SELECT 'estado_fase'::text, filtrada.estado_clave, filtrada.fase_clave,
           NULL::timestamptz, NULL::boolean, pg_catalog.count(*)::numeric, NULL::jsonb
      FROM filtradas filtrada
     GROUP BY filtrada.estado_clave, filtrada.fase_clave
    UNION ALL
    -- Un expediente en trámite sin fecha de entrada en fase aparece con
    -- fase_desde nula: la fachada lo rechaza en lugar de omitirlo.
    SELECT 'plazo'::text, NULL::text, filtrada.fase_clave, entrada.fase_desde,
           COALESCE(urgencia.primera_version <= filtrada.version, false),
           pg_catalog.count(*)::numeric,
           jsonb_build_object('estado',CASE WHEN s.estado='legado_sin_instantanea' AND t.unico THEN 'legado_base_transicion' ELSE s.estado END,'fase',s.fase_clave,
             'fase_desde',s.fase_desde,'base_id',coalesce(s.base_catalogo_id,t.base_catalogo_id),
             'base_version',coalesce(s.base_version,t.base_version),'base_huella',coalesce(s.base_huella_sha256,t.base_huella_sha256),
             'ajustes_id',coalesce(s.ajustes_catalogo_id,t.ajustes_catalogo_id),
             'ajustes_encontrados',coalesce(s.ajustes_encontrados,t.ajustes_encontrados),
             'ajustes_version',coalesce(s.ajustes_version,t.ajustes_version),'ajustes_huella',coalesce(s.ajustes_huella_sha256,t.ajustes_huella_sha256),
             'ajustes_vigente_desde',coalesce(s.ajustes_vigente_desde,t.ajustes_vigente_desde),'capturada_en',coalesce(s.capturada_en,t.capturada_en))
      FROM filtradas filtrada
      LEFT JOIN vec_contratacion_temporal.fase_entrada_publicacion_rrhh entrada
        ON entrada.expediente_ref = filtrada.expediente_ref
       AND entrada.version = filtrada.version
      LEFT JOIN vec_contratacion_temporal.fase_regla_instantanea_v1 s
        ON s.expediente_ref=filtrada.expediente_ref AND s.version=filtrada.version
      LEFT JOIN vec_contratacion_temporal.regla_legado_transicion_v1 t
        ON s.estado='legado_sin_instantanea' AND s.fase_desde<=t.capturada_en
      LEFT JOIN (
          SELECT expediente_ref, pg_catalog.min(version) AS primera_version
            FROM vec_contratacion_temporal.urgencia_expediente_analisis
           GROUP BY expediente_ref
      ) urgencia ON urgencia.expediente_ref = filtrada.expediente_ref
     WHERE filtrada.estado_clave NOT IN ('completado', 'cancelado')
     GROUP BY filtrada.fase_clave, entrada.fase_desde, 5, 7;
END
$$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.contar_resumen_cuadro_rrhh_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,text) FROM PUBLIC;

-- CT247-GENERADO: canon_resumen_publicacion_rrhh_v1
CREATE OR REPLACE FUNCTION vec_contratacion_temporal.canon_resumen_publicacion_rrhh_v1(p_resumen vec_contratacion_temporal.resumen_publicacion_rrhh_v1)
 RETURNS bytea
 LANGUAGE plpgsql
 IMMUTABLE PARALLEL SAFE STRICT
 SET search_path TO 'pg_catalog'
AS $function$
DECLARE
    v_canon bytea := ''::bytea;
    v_campo text;
    v_campos text[];
BEGIN
    IF p_resumen.expediente_ref IS NULL
       OR p_resumen.expediente_ref !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_resumen.organizacion_ref IS NULL
       OR p_resumen.organizacion_ref !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_resumen.numero_visible IS NULL
       OR p_resumen.numero_visible !~
          '^([0-9]{4}/[A-Za-z0-9._-]{1,40}|[0-9]{4}/[A-Za-z0-9._-]{1,40}-P(100|[1-9][0-9]?))$'
       OR p_resumen.version IS NULL
       OR p_resumen.version NOT BETWEEN 1 AND 9007199254740991::numeric
       OR p_resumen.flujo_ref IS NULL
       OR p_resumen.flujo_ref !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_resumen.flujo_version IS NULL
       OR p_resumen.flujo_version
          NOT BETWEEN 1 AND 9007199254740991::numeric
       OR p_resumen.flujo_huella_sha256 IS NULL
       OR p_resumen.flujo_huella_sha256 !~ '^[0-9a-f]{64}$'
       OR p_resumen.flujo_huella_sha256 = pg_catalog.repeat('0', 64)
       OR p_resumen.fase_clave IS NULL
       OR p_resumen.fase_clave !~ '^[a-z][a-z0-9._-]{1,79}$'
       OR p_resumen.estado_clave IS NULL
       OR p_resumen.estado_clave NOT IN (
           'pendiente', 'en_curso', 'espera_externa',
           'completado', 'incidencia', 'cancelado'
       )
       OR p_resumen.centro_ref IS NULL
       OR p_resumen.centro_ref !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_resumen.categoria_ref IS NULL
       OR p_resumen.categoria_ref !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_resumen.modalidad_clave IS NULL
       OR p_resumen.modalidad_clave !~
          '^$|^[a-z][a-z0-9._-]{1,79}$'
       OR p_resumen.unidad_ref IS NULL
       OR p_resumen.unidad_ref !~
          '^$|^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_resumen.creado_en IS NULL
       OR p_resumen.actualizado_en IS NULL
       OR p_resumen.actualizado_en < p_resumen.creado_en THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'resumen canónico RRHH inválido';
    END IF;
    v_campos := ARRAY[
        p_resumen.expediente_ref, p_resumen.organizacion_ref,
        p_resumen.numero_visible, p_resumen.version::text,
        p_resumen.flujo_ref, p_resumen.flujo_version::text,
        p_resumen.flujo_huella_sha256, p_resumen.fase_clave,
        p_resumen.estado_clave, p_resumen.centro_ref,
        p_resumen.categoria_ref, p_resumen.modalidad_clave,
        p_resumen.unidad_ref,
        vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
            p_resumen.creado_en
        ),
        vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
            p_resumen.actualizado_en
        )
    ];
    FOREACH v_campo IN ARRAY v_campos LOOP
        v_canon := v_canon
            || vec_contratacion_temporal.encuadrar_valor_rrhh_v1(
                vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
                    v_campo
                )
            );
    END LOOP;
    RETURN v_canon;
EXCEPTION WHEN OTHERS THEN
    RAISE EXCEPTION USING ERRCODE = '22023',
        MESSAGE = 'resumen canónico RRHH inválido';
END
$function$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.canon_resumen_publicacion_rrhh_v1(vec_contratacion_temporal.resumen_publicacion_rrhh_v1) FROM PUBLIC;

-- CT247-GENERADO: rechazar_nombramiento_derivado_ct241
CREATE OR REPLACE FUNCTION vec_contratacion_temporal.rechazar_nombramiento_derivado_ct241()
 RETURNS trigger
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
AS $function$
BEGIN
 IF TG_OP<>'INSERT' OR TG_WHEN<>'BEFORE' OR TG_LEVEL<>'ROW'
    OR TG_TABLE_SCHEMA<>'vec_contratacion_temporal'
    OR TG_TABLE_NAME NOT IN ('expediente_version_integral','publicacion_version_rrhh','firma_documento_v1') THEN
  RAISE EXCEPTION 'CT241: disparador fuera de su tabla' USING ERRCODE='42501'; END IF;
 IF NOT EXISTS(SELECT 1 FROM vec_contratacion_temporal.expediente_alta alta
    WHERE alta.expediente_ref=NEW.expediente_ref AND alta.derivacion_ref IS NOT NULL) THEN
  RETURN NEW; END IF;
 IF TG_TABLE_NAME='expediente_version_integral' THEN
  IF NEW.version=6 AND NEW.origen_version='derivacion_nombramiento_ct234' THEN
   RETURN NEW; END IF;
 END IF;

 -- CT247: publicación inicial con ámbito propio; las demás escrituras siguen cerradas.
 IF TG_TABLE_NAME='publicacion_version_rrhh' THEN
 IF NEW.version=6 THEN
  IF EXISTS(SELECT 1 FROM vec_contratacion_temporal.expediente_alta alta
    JOIN vec_contratacion_temporal.nombramiento_derivado_v1 derivada
      ON derivada.derivacion_ref=alta.derivacion_ref AND derivada.hijo_ref=alta.expediente_ref
    JOIN vec_contratacion_temporal.expediente_version_integral inicial
      ON inicial.expediente_ref=alta.expediente_ref AND inicial.version=6
    WHERE alta.expediente_ref=NEW.expediente_ref
      AND vec_contratacion_temporal.inicio_derivado_acreditado_ct234_v1(NEW.expediente_ref,inicial.agregado_json) IS TRUE
      AND inicial.origen_version='derivacion_nombramiento_ct234'
      AND inicial.agregado_json_huella_sha256=NEW.agregado_huella_sha256
      AND inicial.agregado_json_huella_sha256=encode(sha256(convert_to(inicial.agregado_json::text,'UTF8')),'hex')
      AND inicial.flujo_ref=NEW.flujo_ref AND inicial.flujo_version=NEW.flujo_version
      AND inicial.flujo_huella_sha256=NEW.flujo_huella_sha256
      AND inicial.fase_clave=NEW.fase_clave AND inicial.estado=NEW.estado_clave
      AND derivada.material_sha256=encode(sha256(convert_to(derivada.material,'UTF8')),'hex')
      AND derivada.organizacion_ref=NEW.organizacion_ref AND alta.organizacion_ref=NEW.organizacion_ref
      AND derivada.material::jsonb->>'hijo_ref'=NEW.expediente_ref
      AND derivada.material::jsonb->>'organizacion_ref'=NEW.organizacion_ref
      AND derivada.material::jsonb->>'unidad_ref'=NEW.unidad_ref
      AND alta.decision_ref=derivada.decision_ref AND alta.actor_ref=derivada.actor_ref
      AND alta.perfil_ref=derivada.perfil_ref
      AND alta.huella_efecto_sha256=encode(sha256(convert_to(
        '{"ambitos":{"organizacion_ref":"'||NEW.organizacion_ref||'","unidad_ref":"'||NEW.unidad_ref||
        '"},"atributos":{"material_sha256":"'||derivada.material_sha256||'"}}','UTF8')),'hex')
  ) THEN RETURN NEW; END IF;
 END IF;
 END IF;
 RAISE EXCEPTION 'CT241: nombramiento derivado sin autoridad propia' USING ERRCODE='42501';
END $function$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.rechazar_nombramiento_derivado_ct241() FROM PUBLIC;
DO $control$
DECLARE f record; p pg_proc%ROWTYPE;
BEGIN
 FOR f IN SELECT * FROM (VALUES
 ('canon_contenido_detalle_rrhh_v1','1074f9d6447a4ba880f4648193ca326b953f791071e0e4bb5e54d765c518e656',ARRAY['search_path=pg_catalog']::text[],ARRAY['vec_contratacion_temporal_propietario/vec_contratacion_temporal_propietario:EXECUTE:false']::text[],false,'i','s'),
 ('materializar_detalle_rrhh_v1','b4f9a9bbfadb09e4c5b839ee09e4137b7cb6a9880cbda7a4a3443fedab108815',ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=1s','statement_timeout=4s','idle_in_transaction_session_timeout=6s']::text[],ARRAY['vec_contratacion_temporal_propietario/vec_contratacion_temporal_propietario:EXECUTE:false']::text[],true,'s','u'),
 ('materializar_cuadro_rrhh_v1','9c233d4e281ebabe6e0b22dc38926274dd9970d72eb51d95ed046c40eea87d28',ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=1s','statement_timeout=4s','idle_in_transaction_session_timeout=6s']::text[],ARRAY['vec_contratacion_temporal_propietario/vec_contratacion_temporal_propietario:EXECUTE:false']::text[],true,'v','u'),
 ('consultar_cambios_expediente_rrhh_atestado_v1','ca65fb853b95f9b8e6f66e173fe81a472b4a9b20d20cd1e96abe3a0a798a5868',ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=1s','statement_timeout=4s','idle_in_transaction_session_timeout=6s']::text[],ARRAY['vec_contratacion_temporal_consultor_rrhh/vec_contratacion_temporal_propietario:EXECUTE:false','vec_contratacion_temporal_propietario/vec_contratacion_temporal_propietario:EXECUTE:false']::text[],true,'v','u'),
 ('consultar_resumen_seguimiento_rrhh_atestado_v1','a43a3a5d435382601428d0647e600897d5aeb1184dec5e86547522b8401d873a',ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=1s','statement_timeout=4s','idle_in_transaction_session_timeout=6s']::text[],ARRAY['vec_contratacion_temporal_consultor_rrhh/vec_contratacion_temporal_propietario:EXECUTE:false','vec_contratacion_temporal_propietario/vec_contratacion_temporal_propietario:EXECUTE:false']::text[],true,'v','u'),
 ('consultar_estadisticas_rrhh_v1','a44b76989e35b3098cb92be814b1f8185066064fe1713d324d3b7a92c5e31c07',ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=1s','statement_timeout=4s','idle_in_transaction_session_timeout=6s']::text[],ARRAY['vec_contratacion_temporal_consultor_rrhh/vec_contratacion_temporal_propietario:EXECUTE:false','vec_contratacion_temporal_propietario/vec_contratacion_temporal_propietario:EXECUTE:false']::text[],true,'s','u'),
 ('contar_totales_cuadro_rrhh_v1','b6a27c8e24b17dceb4da462af33b4ac7d6dd118d012f281ee1eaedf3e76cfd84',ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=1s','statement_timeout=4s','idle_in_transaction_session_timeout=6s']::text[],ARRAY['vec_contratacion_temporal_propietario/vec_contratacion_temporal_propietario:EXECUTE:false']::text[],true,'s','u'),
 ('contar_resumen_cuadro_rrhh_v1','544efd5706d39cff2469f2242d892489de1b1ca826111206c78d857bab3024ec',ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=1s','statement_timeout=4s','idle_in_transaction_session_timeout=6s']::text[],ARRAY['vec_contratacion_temporal_propietario/vec_contratacion_temporal_propietario:EXECUTE:false']::text[],true,'s','u'),
 ('canon_resumen_publicacion_rrhh_v1','9b5ea485a08a5a40160b1db0248c1ee33699ed2a550e2c0f0472972f0059db02',ARRAY['search_path=pg_catalog']::text[],ARRAY['vec_contratacion_temporal_propietario/vec_contratacion_temporal_propietario:EXECUTE:false']::text[],false,'i','s'),
 ('rechazar_nombramiento_derivado_ct241','c871a6ced3de2998f2f32c6e7efd0dc216c56ef3151fab5ea78540defb18f91a',ARRAY['search_path=pg_catalog, pg_temp','row_security=on']::text[],ARRAY['vec_contratacion_temporal_propietario/vec_contratacion_temporal_propietario:EXECUTE:false']::text[],true,'v','u')
 ) AS esperado(nombre,huella,configuracion,acl,definidor,volatilidad,paralelo) LOOP
  SELECT * INTO STRICT p FROM pg_proc
   WHERE pronamespace='vec_contratacion_temporal'::regnamespace AND proname=f.nombre;
  IF encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM f.huella
     OR p.proowner<>'vec_contratacion_temporal_propietario'::regrole
     OR p.prosecdef IS DISTINCT FROM f.definidor
     OR p.provolatile IS DISTINCT FROM f.volatilidad::"char"
     OR p.proparallel IS DISTINCT FROM f.paralelo::"char"
     OR p.proconfig IS DISTINCT FROM f.configuracion
     OR (SELECT array_agg((a.grantee::regrole::text)||'/'||(a.grantor::regrole::text)||':'||
               a.privilege_type||':'||a.is_grantable::text ORDER BY a.grantee::regrole::text COLLATE "C")
          FROM aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a) IS DISTINCT FROM f.acl
  THEN RAISE EXCEPTION 'CT247: definición o permisos divergentes: %',f.nombre USING ERRCODE='55000'; END IF;
 END LOOP;
END $control$;
COMMIT;
