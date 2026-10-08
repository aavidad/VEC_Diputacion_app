\set ON_ERROR_STOP on

-- Solo para la base PostgreSQL 18 desechable del runner CT193.
-- El catálogo es sintético; sus bytes son la instantánea atestada.
CREATE FUNCTION public.ct193_e3_preparar_vector(
    p_caso text, p_jornada integer, p_colision boolean DEFAULT false
) RETURNS void LANGUAGE plpgsql VOLATILE SET search_path=pg_catalog AS $funcion$
DECLARE
    v_catalogo jsonb;
    v_catalogo_bytes bytea;
    v_periodo jsonb := '{"inicio":"2026-10-01","fin":"2027-06-30"}'::jsonb;
    v_alta jsonb;
    v_sellos jsonb;
    v_decision jsonb;
    v_origen jsonb;
    v_origen_sellos jsonb;
    v_necesidad jsonb;
    v_causa jsonb;
    v_campos jsonb;
    v_causa_clave text := 'acumulacion_tareas';
    v_catalogo_version integer := 2;
BEGIN
    IF p_caso NOT IN ('e3_alta','abierto','colision','concurrente')
       OR p_jornada NOT BETWEEN 1 AND 2250
       OR p_colision IS DISTINCT FROM (p_caso='colision') THEN
        RAISE EXCEPTION 'CT193 E3: fase de vector inválida';
    END IF;

    PERFORM public.preparar_vector_o2_05(p_caso,'valido',1);
    SELECT convert_from(alta,'UTF8')::jsonb,
           convert_from(sellos,'UTF8')::jsonb,
           convert_from(decision,'UTF8')::jsonb
      INTO STRICT v_alta,v_sellos,v_decision
      FROM public.vectores_o2_05 WHERE caso=p_caso;

    v_campos:=jsonb_build_object(
       'numero_personas','2',
       'justificacion_temporal','Refuerzo limitado.',
       'organica_codigo','100','funcional_codigo','200',
       'proyecto_gasto_codigo','300','porcentaje_financiacion','100');
    v_causa:=jsonb_build_object(
       'clave','acumulacion_tareas',
       'etiqueta_clave','ct.necesidad.acumulacion',
       'fuente_ref','circular:ct:prueba',
       'fuente_url','https://www.dipgra.es/ct',
       'regla_ref','regla:ct:acumulacion:v1',
       'fecha_fin','obligatoria',
       'maximo_meses',9,'ventana_meses',18,
       'campos_permitidos','["numero_personas","justificacion_temporal","organica_codigo","funcional_codigo","proyecto_gasto_codigo","porcentaje_financiacion"]'::jsonb,
       'campos_obligatorios','["numero_personas","justificacion_temporal","organica_codigo","funcional_codigo","proyecto_gasto_codigo","porcentaje_financiacion"]'::jsonb);
    IF p_caso='abierto' THEN
       v_causa_clave:='sustitucion';
       v_catalogo_version:=1;
       v_campos:=jsonb_build_object(
         'puesto_codigo','3388',
         'rpt_catalogo_ref','fuente:importacion:rpt:prueba',
         'rpt_catalogo_huella_sha256',repeat('a',64),
         'organica_codigo','100','funcional_codigo','200',
         'proyecto_gasto_codigo','300','porcentaje_financiacion','100');
       v_causa:=jsonb_build_object(
         'clave','sustitucion',
         'etiqueta_clave','ct.necesidad.sustitucion',
         'fuente_ref','circular:ct:prueba',
         'fuente_url','https://www.dipgra.es/ct',
         'regla_ref','regla:ct:sustitucion:v1',
         'fecha_fin','opcional','causa_fin','reincorporacion_titular',
         'maximo_meses',36,
         'campos_permitidos','["puesto_codigo","plaza_codigo","rpt_catalogo_ref","rpt_catalogo_huella_sha256","titular_ref","organica_codigo","funcional_codigo","proyecto_gasto_codigo","porcentaje_financiacion"]'::jsonb,
         'campos_obligatorios','["puesto_codigo","rpt_catalogo_ref","rpt_catalogo_huella_sha256","organica_codigo","funcional_codigo","proyecto_gasto_codigo","porcentaje_financiacion"]'::jsonb);
    END IF;
    v_catalogo:=jsonb_build_object(
      'esquema','vec.ct.necesidades_alta.v1',
      'referencia','catalogo:ct:paridad','version',v_catalogo_version,'es_ejemplo',true,
      'fuente_ref','circular:ct:prueba','fuente_url','https://www.dipgra.es/ct',
      'jornada_referencia_minutos',2250,
      'jornada_fuente_ref','operador:ct:prueba',
      'causas',jsonb_build_array(v_causa));
    v_catalogo_bytes:=convert_to(v_catalogo::text,'UTF8');
    IF p_caso='abierto' THEN
       v_periodo:=jsonb_build_object(
         'inicio','2026-10-01',
         'causa_fin','reincorporacion_titular',
         'politica_fin',jsonb_build_object(
           'regla_ref','regla:ct:sustitucion:v1',
           'catalogo_version',v_catalogo_version,
           'catalogo_huella_sha256',encode(sha256(v_catalogo_bytes),'hex'),
           'fecha_fin','opcional',
           'causa_fin','reincorporacion_titular'));
    END IF;
    v_necesidad:=jsonb_build_object(
      'esquema','vec.ct.necesidad_alta.v1',
      'catalogo_ref','catalogo:ct:paridad',
      'catalogo_version',v_catalogo_version,
      'catalogo_huella_sha256',encode(sha256(v_catalogo_bytes),'hex'),
      'causa_clave',v_causa_clave,
      'periodo',v_periodo,
      'jornada_minutos',p_jornada,
      'campos',v_campos,
      'catalogo_instantanea',replace(encode(v_catalogo_bytes,'base64'),E'\n','')
    );

    IF p_colision THEN
        SELECT convert_from(alta,'UTF8')::jsonb,
               convert_from(sellos,'UTF8')::jsonb
          INTO STRICT v_origen,v_origen_sellos
          FROM public.vectores_o2_05 WHERE caso='e3_alta';
        IF v_origen#>>'{solicitud,necesidad,jornada_minutos}' <> '1125' THEN
            RAISE EXCEPTION 'CT193 E3: falta el vector E3 previo';
        END IF;
        v_alta:=v_origen;
        v_sellos:=v_origen_sellos;
        v_sellos:=jsonb_set(v_sellos,'{activo,huella_hmac}',
          to_jsonb('hmac-sha256:vec.contratacion-temporal.huella-peticion/v2:' ||
            encode(sha256(convert_to('ct193:colision:jornada:1124','UTF8')),'hex')));
        v_decision:=jsonb_set(v_decision,'{recurso_ref}',
          to_jsonb(v_sellos#>>'{activo,ambito_hmac}'));
    END IF;

    v_alta:=jsonb_set(v_alta,'{esquema}',
      '"vec.contratacion-temporal.efecto-alta.v3"'::jsonb);
    v_alta:=jsonb_set(v_alta,'{solicitud,rc,fecha}','""'::jsonb);
    v_alta:=jsonb_set(v_alta,'{solicitud,rc,importe,moneda}',
      '"EUR"'::jsonb);
    v_alta:=jsonb_set(v_alta,'{solicitud,motivo_clave}',to_jsonb(v_causa_clave));
    v_alta:=jsonb_set(v_alta,'{solicitud,periodo}',v_periodo);
    v_alta:=jsonb_set(v_alta,'{solicitud,necesidad}',v_necesidad,true);
    IF vec_contratacion_temporal.necesidad_alta_valida_v3(
         v_alta->'solicitud') IS NOT TRUE THEN
        RAISE EXCEPTION 'CT193 E3: necesidad sintética inválida';
    END IF;

    UPDATE public.vectores_o2_05
       SET alta=vec_contratacion_temporal.reconstruir_efecto_alta_v3(v_alta),
           sellos=vec_contratacion_temporal.reconstruir_sellos_hmac_v1(v_sellos),
           decision=vec_autorizacion.decision_contexto_actor_v3_canonica(v_decision)
     WHERE caso=p_caso;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'CT193 E3: vector no preparado';
    END IF;
END $funcion$;

REVOKE ALL ON FUNCTION public.ct193_e3_preparar_vector(text,integer,boolean)
    FROM PUBLIC;
