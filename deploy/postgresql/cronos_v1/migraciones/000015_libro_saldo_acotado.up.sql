\set ON_ERROR_STOP on
-- CRN15. Optimiza la lectura derivada C3p sin cambiar firma, OID, resultado,
-- tablas, políticas ni autoridad del consumidor V3 de cronos_v1 000007.
-- La función histórica 000004 permanece intacta; no ejecutar DOWN sobre historia.
BEGIN;
SET LOCAL ROLE vec_cronos_v1_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_cronos_v1:migracion:000015:libro-saldo',0));
DO $pre$
DECLARE f pg_proc%ROWTYPE; idx record; cfg text; cuerpo_sha text;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999 THEN
  RAISE EXCEPTION 'CRN15 pre: clave=server_version_num esperado=18xxxx actual=%',current_setting('server_version_num') USING ERRCODE='55000';
 END IF;
 IF current_user<>'vec_cronos_v1_propietario' THEN
  RAISE EXCEPTION 'CRN15 pre: clave=current_user esperado=vec_cronos_v1_propietario actual=%',current_user USING ERRCODE='55000';
 END IF;
 IF to_regclass('vec_cronos_v1.marcaje_original') IS NULL THEN
  RAISE EXCEPTION 'CRN15 pre: clave=marcaje_original esperado=tabla presente actual=ausente' USING ERRCODE='55000';
 END IF;
 IF to_regclass('vec_cronos_v1.programacion_jornada') IS NULL THEN
  RAISE EXCEPTION 'CRN15 pre: clave=programacion_jornada esperado=tabla presente actual=ausente' USING ERRCODE='55000';
 END IF;
 IF to_regclass('vec_cronos_v1.marcaje_original_persona_instante_idx') IS NULL THEN
  RAISE EXCEPTION 'CRN15 pre: clave=indice_persona_instante esperado=indice presente actual=ausente' USING ERRCODE='55000';
 END IF;
 IF to_regprocedure('vec_cronos_v1.consultar_libro_saldo_interno_v1(text,date,date,text)') IS NULL THEN
  RAISE EXCEPTION 'CRN15 pre: clave=funcion esperado=firma text,date,date,text actual=ausente' USING ERRCODE='55000';
 END IF;
 SELECT * INTO STRICT f FROM pg_proc
 WHERE oid='vec_cronos_v1.consultar_libro_saldo_interno_v1(text,date,date,text)'::regprocedure;
 IF f.proowner IS DISTINCT FROM current_user::regrole THEN
  RAISE EXCEPTION 'CRN15 pre: clave=owner esperado=% actual=%',current_user,f.proowner::regrole USING ERRCODE='55000';
 END IF;
 IF f.prosecdef IS DISTINCT FROM true THEN
  RAISE EXCEPTION 'CRN15 pre: clave=security_definer esperado=true actual=%',f.prosecdef USING ERRCODE='55000';
 END IF;
 IF f.provolatile IS DISTINCT FROM 's' THEN
  RAISE EXCEPTION 'CRN15 pre: clave=volatilidad esperado=STABLE actual=%',f.provolatile USING ERRCODE='55000';
 END IF;
 IF f.prorettype IS DISTINCT FROM 'jsonb'::regtype THEN
  RAISE EXCEPTION 'CRN15 pre: clave=retorno esperado=jsonb actual=%',f.prorettype::regtype USING ERRCODE='55000';
 END IF;
 cfg:=coalesce(replace(lower(array_to_string(f.proconfig,',')),' ',''),'');
 IF cfg<>'search_path=pg_catalog,row_security=on,timezone=utc,lock_timeout=2s' THEN
  RAISE EXCEPTION 'CRN15 pre: clave=proconfig esperado=% actual=%',
   'search_path=pg_catalog,row_security=on,timezone=utc,lock_timeout=2s',cfg USING ERRCODE='55000';
 END IF;
 cuerpo_sha:=encode(sha256(convert_to(coalesce(f.prosrc,''),'UTF8')),'hex');
 IF cuerpo_sha<>'6da209aee3dc520303c9976aaefd2b8e22757d3605b35771c7ec224fb6ea5f77' THEN
  RAISE EXCEPTION 'CRN15 pre: clave=prosrc_sha256 esperado=% actual=%',
   '6da209aee3dc520303c9976aaefd2b8e22757d3605b35771c7ec224fb6ea5f77',cuerpo_sha USING ERRCODE='55000';
 END IF;
 IF has_function_privilege('vec_cronos_v1_ejecutor',f.oid,'EXECUTE') IS DISTINCT FROM false THEN
  RAISE EXCEPTION 'CRN15 pre: clave=runtime_execute esperado=false actual=%',
   has_function_privilege('vec_cronos_v1_ejecutor',f.oid,'EXECUTE') USING ERRCODE='55000';
 END IF;
 IF has_function_privilege('vec_cronos_v1_migrador',f.oid,'EXECUTE') IS DISTINCT FROM false THEN
  RAISE EXCEPTION 'CRN15 pre: clave=migrador_execute esperado=false actual=%',
   has_function_privilege('vec_cronos_v1_migrador',f.oid,'EXECUTE') USING ERRCODE='55000';
 END IF;
 SELECT i.indisvalid,i.indisready,pg_get_indexdef(i.indexrelid) AS definicion
 INTO STRICT idx FROM pg_index i
 WHERE i.indexrelid='vec_cronos_v1.marcaje_original_persona_instante_idx'::regclass;
 IF idx.indisvalid IS DISTINCT FROM true OR idx.indisready IS DISTINCT FROM true THEN
  RAISE EXCEPTION 'CRN15 pre: clave=indice_estado esperado=valid,ready actual=%,%',idx.indisvalid,idx.indisready USING ERRCODE='55000';
 END IF;
 IF strpos(coalesce(idx.definicion,''),'(empleado_ref, instante_utc, marcaje_ref)')=0 THEN
  RAISE EXCEPTION 'CRN15 pre: clave=indice_columnas esperado=empleado_ref,instante_utc,marcaje_ref actual=%',
   coalesce(idx.definicion,'<NULL>') USING ERRCODE='55000';
 END IF;
 PERFORM set_config('vec.cronos.crn15_oid',f.oid::text,true);
 PERFORM set_config('vec.cronos.crn15_acl',coalesce(f.proacl::text,'<NULL>'),true);
END $pre$;
CREATE OR REPLACE FUNCTION vec_cronos_v1.consultar_libro_saldo_interno_v1(
    p_empleado_ref text,p_desde date,p_hasta date,p_zona_horaria text
) RETURNS jsonb LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE jornadas jsonb; marcajes jsonb; movimientos jsonb; pendientes integer;
BEGIN
    IF p_empleado_ref IS NULL OR p_empleado_ref !~ '^emp_[-A-Za-z0-9_]{22,128}$'
       OR p_desde IS NULL OR p_hasta IS NULL OR p_hasta<p_desde OR p_hasta-p_desde>366
       OR p_zona_horaria IS NULL OR p_zona_horaria NOT IN ('Europe/Madrid','Atlantic/Canary')
       OR p_empleado_ref IS DISTINCT FROM nullif(current_setting('vec.cronos.empleado_ref',true),'') THEN
        RAISE EXCEPTION 'consulta Cronos inválida' USING ERRCODE='PC003';
    END IF;
    -- Una segunda versión no sustituye silenciosamente el asiento histórico:
    -- esa fecha queda no disponible hasta un recálculo gobernado y versionado.
    WITH vigente AS (
      SELECT p.programacion_ref,p.fecha,p.turno_ref,p.politica_version_ref,
             p.fuente_ref,p.zona_horaria,p.minutos_previstos,p.version
      FROM vec_cronos_v1.programacion_jornada p
      WHERE p.empleado_ref=p_empleado_ref AND p.fecha BETWEEN p_desde AND p_hasta
        AND NOT EXISTS (SELECT 1 FROM vec_cronos_v1.programacion_jornada p2
          WHERE p2.empleado_ref=p.empleado_ref AND p2.fecha=p.fecha
            AND p2.programacion_ref<>p.programacion_ref)
    )
    SELECT coalesce(jsonb_agg(jsonb_build_object('programacion_ref',programacion_ref,'fecha',fecha,
      'turno_ref',turno_ref,'politica_version_ref',politica_version_ref,'fuente_ref',fuente_ref,
      'zona_horaria',zona_horaria,'minutos_previstos',minutos_previstos,'version',version)
      ORDER BY fecha),'[]'::jsonb) INTO jornadas FROM vigente;
    -- Los hechos limítrofes permiten parejas nocturnas. También se incluye
    -- el último hecho anterior si deja una secuencia abierta, aunque sea
    -- más antiguo que el margen. La zona local nunca depende del pool.
    SELECT coalesce(jsonb_agg(jsonb_build_object('marcaje_ref',m.marcaje_ref,'movimiento',m.movimiento,
      'instante_utc',m.instante_utc,'canal',m.material::jsonb->'canal',
      'tipo_origen',m.tipo_origen) ORDER BY m.instante_utc,m.marcaje_ref),'[]'::jsonb)
      INTO marcajes FROM vec_cronos_v1.marcaje_original m
      WHERE m.empleado_ref=p_empleado_ref
        AND ((m.instante_utc>=((p_desde-2)::timestamp AT TIME ZONE p_zona_horaria)
          AND m.instante_utc<((p_hasta+3)::timestamp AT TIME ZONE p_zona_horaria))
          OR (m.movimiento IN ('entrada','inicio_pausa','fin_pausa')
            AND m.marcaje_ref=(SELECT anterior.marcaje_ref
              FROM vec_cronos_v1.marcaje_original anterior
              WHERE anterior.empleado_ref=p_empleado_ref
                AND anterior.instante_utc<(p_desde::timestamp AT TIME ZONE p_zona_horaria)
              ORDER BY anterior.instante_utc DESC,anterior.marcaje_ref DESC LIMIT 1)));
    WITH ordenados AS (
      SELECT marcaje_ref,movimiento,instante_utc,
        lead(marcaje_ref) OVER (ORDER BY instante_utc,marcaje_ref) AS siguiente_ref,
        lead(movimiento) OVER (ORDER BY instante_utc,marcaje_ref) AS siguiente_movimiento,
        lead(instante_utc) OVER (ORDER BY instante_utc,marcaje_ref) AS siguiente_utc
      FROM vec_cronos_v1.marcaje_original
      WHERE empleado_ref=p_empleado_ref
        -- Una pareja que aporte tiempo al intervalo dura como máximo 24 h.
        -- El sucesor puede estar fuera del intervalo local; incluir 24 h UTC
        -- a ambos lados conserva su LEAD y los empates por referencia.
        AND instante_utc >= (p_desde::timestamp AT TIME ZONE p_zona_horaria) - interval '24 hours'
        AND instante_utc < ((p_hasta+1)::timestamp AT TIME ZONE p_zona_horaria) + interval '24 hours'
    ), tramos AS (
      SELECT marcaje_ref,siguiente_ref,instante_utc,siguiente_utc FROM ordenados
      WHERE (movimiento='entrada' AND siguiente_movimiento IN ('salida','inicio_pausa'))
         OR (movimiento='fin_pausa' AND siguiente_movimiento IN ('salida','inicio_pausa'))
    ), partidos AS (
      SELECT t.marcaje_ref,t.siguiente_ref,g.fecha::date AS fecha,
        (extract(epoch FROM (least(t.siguiente_utc,(((g.fecha::date+1)::timestamp) AT TIME ZONE p_zona_horaria))
                  - greatest(t.instante_utc,((g.fecha::date)::timestamp AT TIME ZONE p_zona_horaria))))*1000000)::bigint AS microsegundos
      FROM tramos t CROSS JOIN LATERAL generate_series(
        greatest(p_desde,(t.instante_utc AT TIME ZONE p_zona_horaria)::date),
        least(p_hasta,((t.siguiente_utc - interval '1 microsecond') AT TIME ZONE p_zona_horaria)::date),
        interval '1 day') g(fecha)
      WHERE t.siguiente_utc>t.instante_utc AND t.siguiente_utc-t.instante_utc<=interval '24 hours'
    ), trabajo AS (
      SELECT fecha,sum(microsegundos)::bigint AS microsegundos,
        array_agg(marcaje_ref||'/'||siguiente_ref ORDER BY marcaje_ref,siguiente_ref) AS fuentes
      FROM partidos WHERE microsegundos>0 GROUP BY fecha
    ), previsto AS (
      SELECT p.fecha,p.programacion_ref,p.version,p.minutos_previstos
      FROM vec_cronos_v1.programacion_jornada p
      WHERE p.empleado_ref=p_empleado_ref AND p.fecha BETWEEN p_desde AND p_hasta
        AND NOT EXISTS (SELECT 1 FROM vec_cronos_v1.programacion_jornada p2
          WHERE p2.empleado_ref=p.empleado_ref AND p2.fecha=p.fecha
            AND p2.programacion_ref<>p.programacion_ref)
    ), libro AS (
      SELECT fecha,'trabajado'::text AS tipo,microsegundos AS delta_microsegundos,
        to_jsonb(fuentes) AS fuentes FROM trabajo
      UNION ALL
      SELECT fecha,'previsto',-(minutos_previstos::bigint*60000000),jsonb_build_array(programacion_ref||'/v'||version)
      FROM previsto
    )
    SELECT coalesce(jsonb_agg(jsonb_build_object('fecha',fecha,'tipo',tipo,
       'delta_microsegundos',delta_microsegundos,'fuentes',fuentes) ORDER BY fecha,tipo),'[]'::jsonb)
      INTO movimientos FROM libro;
    -- No dar saldo definitivo cuando hay una secuencia inválida o falta
    -- programación. El lector verá hechos y un estado explícito.
    WITH seleccion AS (
      -- Los dos últimos anteriores aportan el hecho que aún solapa B y su
      -- LAG real, incluso si la apertura ocurrió años antes.
      SELECT marcaje_ref,movimiento,instante_utc FROM (
        SELECT marcaje_ref,movimiento,instante_utc
        FROM vec_cronos_v1.marcaje_original
        WHERE empleado_ref=p_empleado_ref
          AND instante_utc<(p_desde::timestamp AT TIME ZONE p_zona_horaria)
        ORDER BY instante_utc DESC,marcaje_ref DESC LIMIT 2
      ) anteriores
      UNION ALL
      SELECT marcaje_ref,movimiento,instante_utc
      FROM vec_cronos_v1.marcaje_original
      WHERE empleado_ref=p_empleado_ref
        AND instante_utc>=(p_desde::timestamp AT TIME ZONE p_zona_horaria)
        AND instante_utc<((p_hasta+1)::timestamp AT TIME ZONE p_zona_horaria)
      UNION ALL
      -- El primer sucesor exterior es el LEAD real del último hecho de B.
      SELECT marcaje_ref,movimiento,instante_utc FROM (
        SELECT marcaje_ref,movimiento,instante_utc
        FROM vec_cronos_v1.marcaje_original
        WHERE empleado_ref=p_empleado_ref
          AND instante_utc>=((p_hasta+1)::timestamp AT TIME ZONE p_zona_horaria)
        ORDER BY instante_utc,marcaje_ref LIMIT 1
      ) sucesor
    ), o AS (
      SELECT movimiento,lead(movimiento) OVER (ORDER BY instante_utc,marcaje_ref) AS siguiente,
       lag(movimiento) OVER (ORDER BY instante_utc,marcaje_ref) AS anterior,
       lead(instante_utc) OVER (ORDER BY instante_utc,marcaje_ref) AS siguiente_utc,
       instante_utc FROM seleccion
    )
    -- Conserva la condición de solape original: una entrada antigua abierta
    -- sigue afectando al día pedido aunque no haya hechos recientes.
    SELECT count(*) INTO pendientes FROM o
    WHERE instante_utc<((p_hasta+1)::timestamp AT TIME ZONE p_zona_horaria)
      AND coalesce(siguiente_utc,'infinity'::timestamptz)>(p_desde::timestamp AT TIME ZONE p_zona_horaria)
      AND ((movimiento IN ('entrada','fin_pausa') AND
          (siguiente IS DISTINCT FROM 'salida' AND siguiente IS DISTINCT FROM 'inicio_pausa'
           OR siguiente_utc<=instante_utc OR siguiente_utc-instante_utc>interval '24 hours'))
        OR (movimiento='inicio_pausa' AND siguiente IS DISTINCT FROM 'fin_pausa')
        OR (movimiento='salida' AND anterior IS DISTINCT FROM 'entrada' AND anterior IS DISTINCT FROM 'fin_pausa')
        OR (movimiento='inicio_pausa' AND anterior IS DISTINCT FROM 'entrada' AND anterior IS DISTINCT FROM 'fin_pausa')
        OR (movimiento='fin_pausa' AND anterior IS DISTINCT FROM 'inicio_pausa'));
    RETURN jsonb_build_object('empleado_ref',p_empleado_ref,'desde',p_desde,'hasta',p_hasta,
      'zona_horaria',p_zona_horaria,'jornadas',jornadas,'marcajes',marcajes,
      'movimientos_saldo',movimientos,'completo',
      pendientes=0 AND jsonb_array_length(jornadas)=p_hasta-p_desde+1
      AND NOT EXISTS (SELECT 1 FROM jsonb_array_elements(jornadas) j
        WHERE j->>'zona_horaria' IS DISTINCT FROM p_zona_horaria));
END $f$;
REVOKE ALL ON FUNCTION vec_cronos_v1.consultar_libro_saldo_interno_v1(text,date,date,text)
 FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador;
DO $post$
DECLARE f pg_proc%ROWTYPE; cfg text; cuerpo_sha text;
BEGIN
 SELECT * INTO STRICT f FROM pg_proc
 WHERE oid='vec_cronos_v1.consultar_libro_saldo_interno_v1(text,date,date,text)'::regprocedure;
 IF f.oid::text IS DISTINCT FROM current_setting('vec.cronos.crn15_oid',true) THEN
  RAISE EXCEPTION 'CRN15 post: clave=oid esperado=% actual=%',current_setting('vec.cronos.crn15_oid',true),f.oid USING ERRCODE='55000';
 END IF;
 IF coalesce(f.proacl::text,'<NULL>') IS DISTINCT FROM current_setting('vec.cronos.crn15_acl',true) THEN
  RAISE EXCEPTION 'CRN15 post: clave=acl esperado=% actual=%',current_setting('vec.cronos.crn15_acl',true),coalesce(f.proacl::text,'<NULL>') USING ERRCODE='55000';
 END IF;
 IF f.proowner IS DISTINCT FROM current_user::regrole THEN
  RAISE EXCEPTION 'CRN15 post: clave=owner esperado=% actual=%',current_user,f.proowner::regrole USING ERRCODE='55000';
 END IF;
 IF f.prosecdef IS DISTINCT FROM true THEN
  RAISE EXCEPTION 'CRN15 post: clave=security_definer esperado=true actual=%',f.prosecdef USING ERRCODE='55000';
 END IF;
 IF f.provolatile IS DISTINCT FROM 's' THEN
  RAISE EXCEPTION 'CRN15 post: clave=volatilidad esperado=STABLE actual=%',f.provolatile USING ERRCODE='55000';
 END IF;
 IF f.prorettype IS DISTINCT FROM 'jsonb'::regtype THEN
  RAISE EXCEPTION 'CRN15 post: clave=retorno esperado=jsonb actual=%',f.prorettype::regtype USING ERRCODE='55000';
 END IF;
 cfg:=coalesce(replace(lower(array_to_string(f.proconfig,',')),' ',''),'');
 IF cfg<>'search_path=pg_catalog,pg_temp,row_security=on,timezone=utc,lock_timeout=2s' THEN
  RAISE EXCEPTION 'CRN15 post: clave=proconfig esperado=% actual=%',
   'search_path=pg_catalog,pg_temp,row_security=on,timezone=utc,lock_timeout=2s',cfg USING ERRCODE='55000';
 END IF;
 cuerpo_sha:=encode(sha256(convert_to(coalesce(f.prosrc,''),'UTF8')),'hex');
 IF cuerpo_sha<>'37d8975b2beb7676f27442dcacd6440442aa8b4c40494c5fd12d066295407375' THEN
  RAISE EXCEPTION 'CRN15 post: clave=prosrc_sha256 esperado=% actual=%',
   '37d8975b2beb7676f27442dcacd6440442aa8b4c40494c5fd12d066295407375',cuerpo_sha USING ERRCODE='55000';
 END IF;
 IF has_function_privilege('vec_cronos_v1_ejecutor',f.oid,'EXECUTE') IS DISTINCT FROM false THEN
  RAISE EXCEPTION 'CRN15 post: clave=runtime_execute esperado=false actual=%',
   has_function_privilege('vec_cronos_v1_ejecutor',f.oid,'EXECUTE') USING ERRCODE='55000';
 END IF;
 IF has_function_privilege('vec_cronos_v1_migrador',f.oid,'EXECUTE') IS DISTINCT FROM false THEN
  RAISE EXCEPTION 'CRN15 post: clave=migrador_execute esperado=false actual=%',
   has_function_privilege('vec_cronos_v1_migrador',f.oid,'EXECUTE') USING ERRCODE='55000';
 END IF;
END $post$;
COMMIT;
