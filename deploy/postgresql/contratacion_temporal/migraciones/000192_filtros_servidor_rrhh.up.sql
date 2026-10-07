\set ON_ERROR_STOP on
-- CT192: filtros de servidor del cuadro RRHH, contrato v2.
BEGIN;
SET LOCAL search_path = pg_catalog, pg_temp;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '120s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_contratacion_temporal:migracion:000192', 0));
DO $pre$
BEGIN
 IF pg_catalog.current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR pg_catalog.to_regtype('vec_contratacion_temporal.consulta_cuadro_rrhh_v2') IS NOT NULL
 OR pg_catalog.to_regprocedure('vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v5(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 OR pg_catalog.to_regclass('vec_contratacion_temporal.control_causal_familia_cursor_rrhh') IS NULL
 -- La salida de cursor v2 reutiliza la postimagen binaria instalada por CT89.
 OR (SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p.prosrc,'UTF8')),'hex') FROM pg_catalog.pg_proc p
     WHERE p.oid=pg_catalog.to_regprocedure('vec_contratacion_temporal.preparar_salida_cursor_cuadro_rrhh_v1(vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1,vec_contratacion_temporal.materializacion_cuadro_rrhh_v1)'))
    IS DISTINCT FROM '5ca9172ea709e3cac9fef939980f8be618b663940c0c19641c7c2d22a1339d89'
 OR (SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p.prosrc,'UTF8')),'hex') FROM pg_catalog.pg_proc p
     WHERE p.oid=pg_catalog.to_regprocedure('vec_contratacion_temporal.aplicar_efectos_cursor_cuadro_rrhh_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1,vec_contratacion_temporal.salida_cursor_cuadro_rrhh_v1,vec_contratacion_temporal.evidencia_consumo_nuevo_rrhh_v3,bytea,vec_contratacion_temporal.resultado_cierre_prueba_rrhh_v2)'))
    IS DISTINCT FROM '0e7d8edd3317eb74092c5a762f80b4054313346d92b61a87f90afbda5d178299'
 OR (SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p.prosrc,'UTF8')),'hex') FROM pg_catalog.pg_proc p
     WHERE p.oid=pg_catalog.to_regprocedure('vec_contratacion_temporal.materializar_cuadro_rrhh_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1)'))
    IS DISTINCT FROM '174ee9497abf84536e59d96b8c1331dea84eeb9efde1d3e5777ca314dc975d76'
 OR (SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p.prosrc,'UTF8')),'hex') FROM pg_catalog.pg_proc p
     WHERE p.oid=pg_catalog.to_regprocedure('vec_contratacion_temporal.contar_totales_cuadro_rrhh_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,text)'))
    IS DISTINCT FROM '0b293192c96a7f193fac4160f3755d9809596226d16d4636be64b73c011201e3'
 OR (SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p.prosrc,'UTF8')),'hex') FROM pg_catalog.pg_proc p
     WHERE p.oid=pg_catalog.to_regprocedure('vec_contratacion_temporal.contar_resumen_cuadro_rrhh_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,text)'))
    IS DISTINCT FROM 'b820203e211aa615dff50d6c49a26dce0e9f43daa2ec0854a9d3036bee401cf7'
 OR (SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p.prosrc,'UTF8')),'hex') FROM pg_catalog.pg_proc p
     WHERE p.oid=pg_catalog.to_regprocedure('vec_contratacion_temporal.registrar_acceso_rrhh_interno_v2(jsonb)'))
    IS DISTINCT FROM '1873ca0f31582d5683ad907755d1526c0b671a15505944b8f727f6fcef91982f'
 THEN RAISE EXCEPTION 'CT192: base incompatible o migración ya presente' USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_contratacion_temporal_propietario;

CREATE TYPE vec_contratacion_temporal.consulta_cuadro_rrhh_v2 AS (
 texto text, centro_ref text, categoria_ref text, estados_clave text[], fases_clave text[], limite smallint, cursor text
);
REVOKE ALL ON TYPE vec_contratacion_temporal.consulta_cuadro_rrhh_v2 FROM PUBLIC;

CREATE FUNCTION vec_contratacion_temporal.canon_consulta_cuadro_rrhh_v2(
 p_consulta vec_contratacion_temporal.consulta_cuadro_rrhh_v2
) RETURNS bytea LANGUAGE plpgsql IMMUTABLE STRICT PARALLEL SAFE
SET search_path=pg_catalog,pg_temp AS $funcion$
DECLARE v_estados text; v_fases text; v_anterior text; v_clave text;
BEGIN
 IF p_consulta.texto IS NULL OR pg_catalog.octet_length(p_consulta.texto)>160
 OR p_consulta.texto<>pg_catalog.btrim(p_consulta.texto)
 OR p_consulta.texto !~ '^[0-9A-Za-zÁÉÍÓÚÜÑáéíóúüñ/._ -]{0,80}$'
 OR p_consulta.centro_ref IS NULL OR p_consulta.categoria_ref IS NULL
 OR (p_consulta.centro_ref<>'' AND (pg_catalog.octet_length(p_consulta.centro_ref)>160 OR p_consulta.centro_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'))
 OR (p_consulta.categoria_ref<>'' AND (pg_catalog.octet_length(p_consulta.categoria_ref)>160 OR p_consulta.categoria_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'))
 OR p_consulta.estados_clave IS NULL OR p_consulta.fases_clave IS NULL
 OR pg_catalog.cardinality(p_consulta.estados_clave)>6 OR pg_catalog.cardinality(p_consulta.fases_clave)>32
 OR pg_catalog.array_ndims(p_consulta.estados_clave)>1 OR pg_catalog.array_ndims(p_consulta.fases_clave)>1
 OR p_consulta.limite IS NULL OR p_consulta.limite NOT BETWEEN 1 AND 100
 OR p_consulta.cursor IS NULL OR pg_catalog.octet_length(p_consulta.cursor)>43
 OR (p_consulta.cursor<>'' AND (p_consulta.cursor !~ '^[A-Za-z0-9_-]{43}$'
 OR pg_catalog.rtrim(pg_catalog.translate(pg_catalog.encode(pg_catalog.decode(pg_catalog.translate(p_consulta.cursor,'-_','+/')||'=','base64'),'base64'),'+/','-_'), E'=\n')<>p_consulta.cursor))
 THEN RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='consulta RRHH inválida'; END IF;
 v_anterior:=NULL; v_estados:='';
 FOREACH v_clave IN ARRAY p_consulta.estados_clave LOOP
  IF v_clave IS NULL OR v_clave NOT IN ('pendiente','en_curso','espera_externa','completado','incidencia','cancelado')
  OR (v_anterior IS NOT NULL AND v_anterior COLLATE "C">=v_clave COLLATE "C")
  THEN RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='consulta RRHH inválida'; END IF;
  v_estados:=v_estados||CASE WHEN v_anterior IS NULL THEN '' ELSE ',' END||vec_contratacion_temporal.texto_json_go_v1(v_clave);
  v_anterior:=v_clave;
 END LOOP;
 v_anterior:=NULL; v_fases:='';
 FOREACH v_clave IN ARRAY p_consulta.fases_clave LOOP
  IF v_clave IS NULL OR v_clave !~ '^[a-z][a-z0-9._-]{1,79}$'
  OR (v_anterior IS NOT NULL AND v_anterior COLLATE "C">=v_clave COLLATE "C")
  THEN RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='consulta RRHH inválida'; END IF;
  v_fases:=v_fases||CASE WHEN v_anterior IS NULL THEN '' ELSE ',' END||vec_contratacion_temporal.texto_json_go_v1(v_clave);
  v_anterior:=v_clave;
 END LOOP;
 RETURN pg_catalog.convert_to('{"dominio":"vec.contratacion_temporal.consulta_rrhh.cuadro.v2","version":2,"texto":'
  ||vec_contratacion_temporal.texto_json_go_v1(p_consulta.texto)
  ||',"centro_ref":'||vec_contratacion_temporal.texto_json_go_v1(p_consulta.centro_ref)
  ||',"categoria_ref":'||vec_contratacion_temporal.texto_json_go_v1(p_consulta.categoria_ref)
  ||',"estados_clave":['||v_estados||'],"fases_clave":['||v_fases||'],"limite":'||p_consulta.limite::text
  ||',"cursor":'||vec_contratacion_temporal.texto_json_go_v1(p_consulta.cursor)||'}','UTF8');
EXCEPTION WHEN OTHERS THEN
 RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='consulta RRHH inválida';
END $funcion$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.canon_consulta_cuadro_rrhh_v2(vec_contratacion_temporal.consulta_cuadro_rrhh_v2) FROM PUBLIC;

CREATE FUNCTION vec_contratacion_temporal.canon_familia_cuadro_rrhh_v2(
 p_consulta vec_contratacion_temporal.consulta_cuadro_rrhh_v2
) RETURNS bytea LANGUAGE plpgsql IMMUTABLE STRICT PARALLEL SAFE
SET search_path=pg_catalog,pg_temp AS $funcion$
DECLARE v_consulta text;
 v_prefijo_consulta text:='{"dominio":"vec.contratacion_temporal.consulta_rrhh.cuadro.v2"';
 v_prefijo_familia text:='{"dominio":"vec.contratacion_temporal.filtros_rrhh.cuadro.v2"';
BEGIN
 v_consulta:=pg_catalog.convert_from(vec_contratacion_temporal.canon_consulta_cuadro_rrhh_v2(p_consulta),'UTF8');
 IF pg_catalog.left(v_consulta,pg_catalog.char_length(v_prefijo_consulta))<>v_prefijo_consulta
 THEN RAISE EXCEPTION USING ERRCODE='22023',MESSAGE='consulta RRHH inválida'; END IF;
 RETURN pg_catalog.convert_to(v_prefijo_familia||pg_catalog.substr(v_consulta,
  pg_catalog.char_length(v_prefijo_consulta)+1,
  pg_catalog.char_length(v_consulta)-pg_catalog.char_length(v_prefijo_consulta)
   -pg_catalog.char_length(vec_contratacion_temporal.texto_json_go_v1(p_consulta.cursor))-11)||'}','UTF8');
END $funcion$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.canon_familia_cuadro_rrhh_v2(vec_contratacion_temporal.consulta_cuadro_rrhh_v2) FROM PUBLIC;


-- CT36 y CT38 fijaban los dominios v1; se amplían conservando el resto del predicado.
DO $dominios$
DECLARE v_reg text; v_fam text; v_n bigint;
BEGIN
 SELECT c.conname INTO STRICT v_reg FROM pg_catalog.pg_constraint c
 WHERE c.conrelid='vec_contratacion_temporal.registro_acceso_rrhh'::pg_catalog.regclass
 AND c.contype='c' AND c.convalidated AND c.conname='registro_acceso_rrhh_check4'
 AND pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.pg_get_constraintdef(c.oid),'UTF8')),'hex')
     ='6ab171d21bd9f658fd63ca5030478dfaa92f36d6915252760c3be05b36fd49f5';
 SELECT c.conname INTO STRICT v_fam FROM pg_catalog.pg_constraint c
 WHERE c.conrelid='vec_contratacion_temporal.familia_cursor_cuadro_rrhh'::pg_catalog.regclass
 AND c.contype='c' AND c.convalidated AND c.conname='familia_cursor_cuadro_rrhh_check1'
 AND pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.pg_get_constraintdef(c.oid),'UTF8')),'hex')
     ='b6009f1a761ef30b245a086e8f8fb3b12dee338a96f0e01d3aba16b4d7cf3bfa';
 EXECUTE pg_catalog.format('ALTER TABLE vec_contratacion_temporal.registro_acceso_rrhh DROP CONSTRAINT %I',v_reg);
 ALTER TABLE vec_contratacion_temporal.registro_acceso_rrhh
 ADD CONSTRAINT registro_acceso_rrhh_dominio_ct192_check CHECK (
 modulo_id='contratacion_temporal' AND (
 (tipo_consulta='cuadro' AND accion='contratacion_temporal.cuadro.consultar'
 AND finalidad='gestion_operativa_contratacion_temporal'
 AND audiencia='vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1'
 AND recurso_tipo='cuadro_rrhh_contratacion_temporal'
 AND dominio_huella_consulta IN ('vec.contratacion_temporal.consulta_rrhh.cuadro.v1','vec.contratacion_temporal.consulta_rrhh.cuadro.v2')
 AND recurso_ref=ambito_ref AND expediente_ref IS NULL AND version_expediente IS NULL
 AND total BETWEEN 0 AND 100 AND resultado_generico='entregado')
 OR (tipo_consulta='detalle' AND accion='contratacion_temporal.expediente.consultar'
 AND finalidad='tramitacion_expediente_contratacion_temporal'
 AND audiencia='vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1'
 AND recurso_tipo='expediente_contratacion_temporal'
 AND dominio_huella_consulta='vec.contratacion_temporal.consulta_rrhh.detalle.v1'
 AND recurso_ref=expediente_ref AND expediente_ref IS NOT NULL
 AND ((total=1 AND resultado_generico='entregado' AND version_expediente IS NOT NULL)
 OR (total=0 AND resultado_generico='sin_resultado' AND version_expediente IS NULL)))));
 EXECUTE pg_catalog.format('ALTER TABLE vec_contratacion_temporal.familia_cursor_cuadro_rrhh DROP CONSTRAINT %I',v_fam);
 ALTER TABLE vec_contratacion_temporal.familia_cursor_cuadro_rrhh
 ADD CONSTRAINT familia_cursor_cuadro_rrhh_dominio_ct192_check CHECK (
 perfil_version BETWEEN 1 AND 9007199254740991::numeric
 AND limite BETWEEN 1 AND 100 AND corte_global BETWEEN 1 AND 9007199254740991::numeric
 AND dominio_filtros IN ('vec.contratacion_temporal.filtros_rrhh.cuadro.v1','vec.contratacion_temporal.filtros_rrhh.cuadro.v2'));
END $dominios$;

-- CT183 cambió el registrador v2: transformar solo su guarda del dominio
-- desde la definición instalada preserva el sellado diferido y la historia.
DO $registrador$
DECLARE v_oid oid; v_antes text; v_despues text; v_viejo text; v_nuevo text;
 v_owner oid; v_secdef boolean; v_acl aclitem[]; v_config text[];
BEGIN
 v_oid:=pg_catalog.to_regprocedure('vec_contratacion_temporal.registrar_acceso_rrhh_interno_v2(jsonb)');
 IF v_oid IS NULL THEN RAISE EXCEPTION 'CT192: registrador ausente' USING ERRCODE='55000'; END IF;
 SELECT pg_catalog.pg_get_functiondef(v_oid),p.proowner,p.prosecdef,p.proacl,p.proconfig
 INTO STRICT v_antes,v_owner,v_secdef,v_acl,v_config FROM pg_catalog.pg_proc p WHERE p.oid=v_oid;
 IF v_owner<>'vec_contratacion_temporal_propietario'::pg_catalog.regrole OR NOT v_secdef
 OR v_acl::text IS DISTINCT FROM '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario}'
 OR NOT ('search_path=pg_catalog'=ANY(v_config))
 THEN RAISE EXCEPTION 'CT192: metadatos de registrador inesperados' USING ERRCODE='55000'; END IF;
 v_viejo:=$old$r ->> 'dominio_huella_consulta' <>
                   'vec.contratacion_temporal.consulta_rrhh.cuadro.v1'$old$;
 v_nuevo:=$new$r ->> 'dominio_huella_consulta' NOT IN (
                   'vec.contratacion_temporal.consulta_rrhh.cuadro.v1',
                   'vec.contratacion_temporal.consulta_rrhh.cuadro.v2')$new$;
 IF (pg_catalog.length(v_antes)-pg_catalog.length(pg_catalog.replace(v_antes,v_viejo,'')))<>pg_catalog.length(v_viejo)
 THEN RAISE EXCEPTION 'CT192: guarda de registrador inesperada' USING ERRCODE='55000'; END IF;
 v_despues:=pg_catalog.replace(v_antes,v_viejo,v_nuevo);
 EXECUTE v_despues;
 IF pg_catalog.pg_get_functiondef(v_oid) IS DISTINCT FROM v_despues
 OR (SELECT p.proowner FROM pg_catalog.pg_proc p WHERE p.oid=v_oid) IS DISTINCT FROM v_owner
 OR (SELECT p.prosecdef FROM pg_catalog.pg_proc p WHERE p.oid=v_oid) IS DISTINCT FROM v_secdef
 OR (SELECT p.proacl FROM pg_catalog.pg_proc p WHERE p.oid=v_oid) IS DISTINCT FROM v_acl
 OR (SELECT p.proconfig FROM pg_catalog.pg_proc p WHERE p.oid=v_oid) IS DISTINCT FROM v_config
 THEN RAISE EXCEPTION 'CT192: registrador no conservado' USING ERRCODE='55000'; END IF;
END $registrador$;
ALTER FUNCTION vec_contratacion_temporal.registrar_acceso_rrhh_interno_v2(jsonb) SET search_path=pg_catalog,pg_temp;


CREATE TYPE vec_contratacion_temporal.contexto_cierre_prueba_rrhh_v3 AS (
 organizacion_ref text, clase_ambito text, ambito_ref text,
 consulta_cuadro vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
 consulta_cuadro_v2 vec_contratacion_temporal.consulta_cuadro_rrhh_v2,
 consulta_detalle vec_contratacion_temporal.consulta_detalle_rrhh_v1,
 familia_ref text);
REVOKE ALL ON TYPE vec_contratacion_temporal.contexto_cierre_prueba_rrhh_v3 FROM PUBLIC;

CREATE FUNCTION
vec_contratacion_temporal.resolver_estado_cursor_cuadro_rrhh_v2(
    p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    p_consulta vec_contratacion_temporal.consulta_cuadro_rrhh_v2,
    p_actor_ref text,
    p_perfil_ref text,
    p_perfil_version numeric,
    p_sesion_ref text,
    p_sesion_huella_sha256 text
)
RETURNS vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
PARALLEL UNSAFE
SET search_path = pg_catalog, pg_temp
SET row_security = 'on'
SET timezone = 'UTC'
SET lock_timeout = '1s'
SET statement_timeout = '4s'
SET idle_in_transaction_session_timeout = '6s'
AS $funcion$
DECLARE
    v_token_huella text;
    v_familia_ref text;
    v_filtros_huella text;
    v_ahora timestamptz(6);
    v_control record;
    v_ligadura record;
    v_corte numeric(20, 0);
BEGIN
    IF CURRENT_USER <> 'vec_contratacion_temporal_propietario'
       OR pg_catalog.pg_is_in_recovery()
       OR pg_catalog.current_setting('transaction_isolation')
          <> 'serializable'
       OR pg_catalog.current_setting('transaction_read_only') <> 'off'
       OR p_actor_ref IS NULL
       OR p_actor_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_perfil_ref IS NULL
       OR p_perfil_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_perfil_version IS NULL
       OR p_perfil_version NOT BETWEEN 1 AND 9007199254740991::numeric
       OR p_perfil_version <> pg_catalog.trunc(p_perfil_version)
       OR p_sesion_ref IS NULL
       OR p_sesion_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_sesion_huella_sha256 IS NULL
       OR p_sesion_huella_sha256 !~ '^[0-9a-f]{64}$'
       OR p_sesion_huella_sha256 = pg_catalog.repeat('0', 64) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'resolución de cursor RRHH rechazada';
    END IF;
    PERFORM vec_contratacion_temporal.canon_alcance_rrhh_v1(p_alcance);
    PERFORM vec_contratacion_temporal.canon_consulta_cuadro_rrhh_v2(
        p_consulta
    );

    IF p_consulta.cursor = '' THEN
        SELECT ultimo_corte
          INTO STRICT v_corte
          FROM vec_contratacion_temporal.control_publicacion_rrhh
         WHERE control;
        RETURN ROW(
            false, NULL, v_corte, 0, NULL, NULL, NULL, NULL, NULL,
            NULL, NULL
        )::vec_contratacion_temporal
             .estado_cursor_entrada_cuadro_rrhh_v1;
    END IF;

    v_token_huella := pg_catalog.encode(pg_catalog.sha256(
        pg_catalog.convert_to(p_consulta.cursor, 'UTF8')
    ), 'hex');
    SELECT cursor.familia_ref
      INTO STRICT v_familia_ref
      FROM vec_contratacion_temporal.cursor_cuadro_rrhh cursor
     WHERE cursor.token_huella_sha256 = v_token_huella;

    SELECT causal.revision, causal.familia_creada_en
      INTO STRICT v_control
      FROM vec_contratacion_temporal
           .control_causal_familia_cursor_rrhh causal
     WHERE causal.familia_ref = v_familia_ref
     FOR UPDATE;

    SELECT familia.*, cursor.token_huella_sha256,
           cursor.pagina, cursor.emitida_en AS cursor_emitida_en,
           cursor.acceso_emision_ref, cursor.ultimo_actualizado_en,
           cursor.ultimo_expediente_ref
      INTO STRICT v_ligadura
      FROM vec_contratacion_temporal.familia_cursor_cuadro_rrhh familia
      JOIN vec_contratacion_temporal.cursor_cuadro_rrhh cursor
        USING (familia_ref)
     WHERE familia.familia_ref = v_familia_ref
       AND cursor.token_huella_sha256 = v_token_huella
     FOR SHARE OF familia, cursor;

    v_filtros_huella := pg_catalog.encode(pg_catalog.sha256(
        vec_contratacion_temporal.canon_familia_cuadro_rrhh_v2(p_consulta)
    ), 'hex');
    v_ahora := pg_catalog.date_trunc(
        'microseconds', pg_catalog.clock_timestamp()
    );
    IF v_control.revision <> 0
       OR v_control.familia_creada_en IS DISTINCT FROM v_ligadura.creada_en
       OR v_ligadura.organizacion_ref IS DISTINCT FROM
          p_alcance.organizacion_ref
       OR v_ligadura.clase_ambito IS DISTINCT FROM p_alcance.clase_ambito
       OR v_ligadura.ambito_ref IS DISTINCT FROM p_alcance.ambito_ref
       OR v_ligadura.actor_ref IS DISTINCT FROM p_actor_ref
       OR v_ligadura.perfil_ref IS DISTINCT FROM p_perfil_ref
       OR v_ligadura.perfil_version IS DISTINCT FROM p_perfil_version
       OR v_ligadura.sesion_ref IS DISTINCT FROM p_sesion_ref
       OR v_ligadura.sesion_huella_sha256 IS DISTINCT FROM
          p_sesion_huella_sha256
       OR v_ligadura.filtros_huella_sha256 IS DISTINCT FROM
          v_filtros_huella
       OR v_ligadura.limite IS DISTINCT FROM p_consulta.limite
       OR v_ahora < v_ligadura.creada_en
       OR v_ahora >= v_ligadura.valida_hasta
       OR EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal
                  .revocacion_familia_cursor_rrhh revocacion
            WHERE revocacion.familia_ref = v_familia_ref
       )
       OR EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal
                  .consumo_cursor_cuadro_rrhh consumo
            WHERE consumo.token_huella_sha256 = v_token_huella
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'resolución de cursor RRHH rechazada';
    END IF;

    RETURN ROW(
        true, v_ligadura.familia_ref, v_ligadura.corte_global,
        v_ligadura.pagina, v_token_huella,
        v_ligadura.acceso_emision_ref, v_ligadura.cursor_emitida_en,
        v_ligadura.creada_en, v_ligadura.valida_hasta,
        v_ligadura.ultimo_actualizado_en,
        v_ligadura.ultimo_expediente_ref
    )::vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1;
EXCEPTION
    WHEN SQLSTATE '40001' OR SQLSTATE '40P01'
      OR SQLSTATE '55P03' OR SQLSTATE '57014' THEN
        RAISE;
    WHEN OTHERS THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'resolución de cursor RRHH rechazada';
END
$funcion$;

CREATE FUNCTION
vec_contratacion_temporal.materializar_cuadro_rrhh_v2(
    p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    p_consulta vec_contratacion_temporal.consulta_cuadro_rrhh_v2,
    p_estado
        vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1
)
RETURNS vec_contratacion_temporal.materializacion_cuadro_rrhh_v1
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
PARALLEL UNSAFE
SET search_path = pg_catalog, pg_temp
SET row_security = 'on'
SET timezone = 'UTC'
SET lock_timeout = '1s'
SET statement_timeout = '4s'
SET idle_in_transaction_session_timeout = '6s'
AS $funcion$
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
    PERFORM vec_contratacion_temporal.canon_consulta_cuadro_rrhh_v2(
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
               publicada.numero_visible,
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
           AND (p_consulta.centro_ref = '' OR ultima.centro_ref COLLATE "C" = p_consulta.centro_ref COLLATE "C")
           AND (p_consulta.categoria_ref = '' OR ultima.categoria_ref COLLATE "C" = p_consulta.categoria_ref COLLATE "C")
           AND (pg_catalog.cardinality(p_consulta.estados_clave)=0 OR ultima.estado_clave COLLATE "C" = ANY(p_consulta.estados_clave))
           AND (pg_catalog.cardinality(p_consulta.fases_clave)=0 OR ultima.fase_clave COLLATE "C" = ANY(p_consulta.fases_clave))
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
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.contar_totales_cuadro_rrhh_v2(
    p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    p_consulta vec_contratacion_temporal.consulta_cuadro_rrhh_v2,
    p_cursor text
)
RETURNS TABLE(
    total_filtrado numeric,
    en_tramitacion numeric,
    con_incidencia numeric,
    en_llamamiento numeric
)
LANGUAGE plpgsql
STABLE
SECURITY DEFINER
PARALLEL UNSAFE
SET search_path = pg_catalog, pg_temp
SET row_security = 'on'
SET timezone = 'UTC'
SET lock_timeout = '1s'
SET statement_timeout = '4s'
SET idle_in_transaction_session_timeout = '6s'
AS $funcion$
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
    PERFORM vec_contratacion_temporal.canon_consulta_cuadro_rrhh_v2(p_consulta);
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
               publicada.expediente_ref, publicada.organizacion_ref,
               publicada.numero_visible, publicada.fase_clave,
               publicada.estado_clave, publicada.centro_ref, publicada.categoria_ref, publicada.unidad_ref
          FROM vec_contratacion_temporal.publicacion_version_rrhh publicada
         WHERE publicada.corte_global <= v_corte_global
         ORDER BY publicada.expediente_ref COLLATE "C", publicada.corte_global DESC
    ), filtradas AS MATERIALIZED (
        SELECT ultima.estado_clave, ultima.fase_clave
          FROM ultimas ultima
         LEFT JOIN vec_contratacion_temporal.numeracion_anual_asignada numeracion
            ON numeracion.expediente_ref = ultima.expediente_ref
         WHERE ultima.organizacion_ref COLLATE "C" = p_alcance.organizacion_ref COLLATE "C"
           AND (p_alcance.clase_ambito = 'organizacion'
                OR p_alcance.clase_ambito = 'centro' AND ultima.centro_ref COLLATE "C" = p_alcance.ambito_ref COLLATE "C"
                OR p_alcance.clase_ambito = 'unidad_gestion' AND ultima.unidad_ref IS NOT NULL AND ultima.unidad_ref COLLATE "C" = p_alcance.ambito_ref COLLATE "C")
           AND (p_consulta.texto = '' OR pg_catalog.left(COALESCE(numeracion.numero_visible, ultima.numero_visible), pg_catalog.char_length(p_consulta.texto)) COLLATE "C" = p_consulta.texto COLLATE "C")
           AND (p_consulta.centro_ref = '' OR ultima.centro_ref COLLATE "C" = p_consulta.centro_ref COLLATE "C")
           AND (p_consulta.categoria_ref = '' OR ultima.categoria_ref COLLATE "C" = p_consulta.categoria_ref COLLATE "C")
           AND (pg_catalog.cardinality(p_consulta.estados_clave)=0 OR ultima.estado_clave COLLATE "C" = ANY(p_consulta.estados_clave))
           AND (pg_catalog.cardinality(p_consulta.fases_clave)=0 OR ultima.fase_clave COLLATE "C" = ANY(p_consulta.fases_clave))
    )
    SELECT pg_catalog.count(*)::numeric,
           pg_catalog.count(*) FILTER (WHERE 'en_curso' = estado_clave)::numeric,
           pg_catalog.count(*) FILTER (WHERE 'incidencia' = estado_clave)::numeric,
           pg_catalog.count(*) FILTER (WHERE 'llamamiento' = fase_clave)::numeric
      FROM filtradas;
END
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.contar_resumen_cuadro_rrhh_v2(
    p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    p_consulta vec_contratacion_temporal.consulta_cuadro_rrhh_v2,
    p_cursor text
)
RETURNS TABLE(
    clase text,
    estado_clave text,
    fase_clave text,
    fase_desde timestamptz,
    urgente boolean,
    numero numeric
)
LANGUAGE plpgsql
STABLE
SECURITY DEFINER
PARALLEL UNSAFE
SET search_path = pg_catalog, pg_temp
SET row_security = 'on'
SET timezone = 'UTC'
SET lock_timeout = '1s'
SET statement_timeout = '4s'
SET idle_in_transaction_session_timeout = '6s'
AS $funcion$
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
    PERFORM vec_contratacion_temporal.canon_consulta_cuadro_rrhh_v2(p_consulta);
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
               publicada.estado_clave, publicada.centro_ref, publicada.categoria_ref, publicada.unidad_ref
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
           AND (p_alcance.clase_ambito = 'organizacion'
                OR p_alcance.clase_ambito = 'centro' AND ultima.centro_ref COLLATE "C" = p_alcance.ambito_ref COLLATE "C"
                OR p_alcance.clase_ambito = 'unidad_gestion' AND ultima.unidad_ref IS NOT NULL AND ultima.unidad_ref COLLATE "C" = p_alcance.ambito_ref COLLATE "C")
           AND (p_consulta.texto = '' OR pg_catalog.left(COALESCE(numeracion.numero_visible, ultima.numero_visible), pg_catalog.char_length(p_consulta.texto)) COLLATE "C" = p_consulta.texto COLLATE "C")
           AND (p_consulta.centro_ref = '' OR ultima.centro_ref COLLATE "C" = p_consulta.centro_ref COLLATE "C")
           AND (p_consulta.categoria_ref = '' OR ultima.categoria_ref COLLATE "C" = p_consulta.categoria_ref COLLATE "C")
           AND (pg_catalog.cardinality(p_consulta.estados_clave)=0 OR ultima.estado_clave COLLATE "C" = ANY(p_consulta.estados_clave))
           AND (pg_catalog.cardinality(p_consulta.fases_clave)=0 OR ultima.fase_clave COLLATE "C" = ANY(p_consulta.fases_clave))
    )
    SELECT 'estado_fase'::text, filtrada.estado_clave, filtrada.fase_clave,
           NULL::timestamptz, NULL::boolean, pg_catalog.count(*)::numeric
      FROM filtradas filtrada
     GROUP BY filtrada.estado_clave, filtrada.fase_clave
    UNION ALL
    -- Un expediente en trámite sin fecha de entrada en fase aparece con
    -- fase_desde nula: la fachada lo rechaza en lugar de omitirlo.
    SELECT 'plazo'::text, NULL::text, filtrada.fase_clave, entrada.fase_desde,
           COALESCE(urgencia.primera_version <= filtrada.version, false),
           pg_catalog.count(*)::numeric
      FROM filtradas filtrada
      LEFT JOIN vec_contratacion_temporal.fase_entrada_publicacion_rrhh entrada
        ON entrada.expediente_ref = filtrada.expediente_ref
       AND entrada.version = filtrada.version
      LEFT JOIN (
          SELECT expediente_ref, pg_catalog.min(version) AS primera_version
            FROM vec_contratacion_temporal.urgencia_expediente_analisis
           GROUP BY expediente_ref
      ) urgencia ON urgencia.expediente_ref = filtrada.expediente_ref
     WHERE filtrada.estado_clave NOT IN ('completado', 'cancelado')
     GROUP BY filtrada.fase_clave, entrada.fase_desde, 5;
END
$funcion$;

CREATE FUNCTION
vec_contratacion_temporal.aplicar_efectos_cursor_cuadro_rrhh_v2(
    p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    p_consulta vec_contratacion_temporal.consulta_cuadro_rrhh_v2,
    p_estado
        vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1,
    p_salida
        vec_contratacion_temporal.salida_cursor_cuadro_rrhh_v1,
    p_consumo
        vec_contratacion_temporal.evidencia_consumo_nuevo_rrhh_v3,
    p_decision_canonica bytea,
    p_cierre
        vec_contratacion_temporal.resultado_cierre_prueba_rrhh_v2
)
RETURNS void
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
PARALLEL UNSAFE
SET search_path = pg_catalog, pg_temp
SET row_security = 'on'
SET timezone = 'UTC'
SET lock_timeout = '1s'
SET statement_timeout = '4s'
SET idle_in_transaction_session_timeout = '6s'
AS $funcion$
DECLARE
    v_alcance
        vec_contratacion_temporal.alcance_acceso_rrhh%ROWTYPE;
    v_registro record;
    v_filtros_huella text;
    v_consulta_huella text;
    v_decision_huella text;
    v_familia_prueba bytea;
    v_cursor_prueba bytea;
    v_consumo_prueba bytea;
    v_valida_hasta timestamptz(6);
    v_padre_emitida_en timestamptz(6);
    v_estado_durable record;
BEGIN
    IF CURRENT_USER <> 'vec_contratacion_temporal_propietario'
       OR pg_catalog.pg_is_in_recovery()
       OR pg_catalog.current_setting('transaction_isolation')
          <> 'serializable'
       OR pg_catalog.current_setting('transaction_read_only') <> 'off'
       OR p_alcance IS NULL
       OR p_consulta IS NULL
       OR p_estado IS NULL
       OR p_salida IS NULL
       OR p_estado.es_continuacion IS NULL
       OR p_salida.hay_mas IS NULL
       OR p_consumo IS NULL
       OR p_decision_canonica IS NULL
       OR p_cierre IS NULL
       OR p_consumo.consumo_nuevo IS DISTINCT FROM true
       OR p_cierre.registrada_en IS NULL
       OR p_cierre.registrada_en <>
          pg_catalog.date_trunc('microseconds', p_cierre.registrada_en)
       OR p_cierre.acceso_ref IS NULL
       OR p_cierre.acceso_ref !~ '^acceso:rrhh:[0-9a-f]{32}$'
       OR p_salida.hay_mas IS DISTINCT FROM
          (p_salida.cursor_siguiente <> '')
       OR (
           NOT p_salida.hay_mas
           AND (
               p_salida.cursor_siguiente IS DISTINCT FROM ''
               OR p_salida.cursor_huella IS DISTINCT FROM ''::bytea
               OR p_salida.familia_ref IS NOT NULL
               OR p_salida.pagina_nueva IS DISTINCT FROM 0
               OR p_salida.token_nuevo_huella_sha256 IS NOT NULL
               OR p_salida.padre_token_huella_sha256 IS NOT NULL
               OR p_salida.ultimo_actualizado_en IS NOT NULL
               OR p_salida.ultimo_expediente_ref IS NOT NULL
           )
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'efectos de cursor RRHH rechazados';
    END IF;
    PERFORM vec_contratacion_temporal.canon_alcance_rrhh_v1(p_alcance);
    PERFORM vec_contratacion_temporal.canon_consulta_cuadro_rrhh_v2(
        p_consulta
    );

    SELECT alcance.*
      INTO STRICT v_alcance
      FROM vec_contratacion_temporal.alcance_acceso_rrhh alcance
     WHERE alcance.acceso_ref = p_cierre.acceso_ref
       AND alcance.acceso_registrado_en = p_cierre.registrada_en;
    SELECT registro.decision_ref, registro.decision_huella_sha256,
           registro.consulta_huella_sha256,
           registro.consumo_vec_huella_sha256,
           prueba.tipo_consulta AS prueba_tipo_consulta,
           prueba.generada_en AS prueba_generada_en,
           prueba.total AS prueba_total,
           prueba.resumenes AS prueba_resumenes,
           prueba.hay_mas AS prueba_hay_mas,
           COALESCE(
               prueba.cursor_huella_sha256, ''
           ) AS prueba_cursor_huella_sha256
      INTO STRICT v_registro
      FROM vec_contratacion_temporal.registro_acceso_rrhh registro
      JOIN vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2 prueba
        ON prueba.acceso_ref = registro.acceso_ref
       AND prueba.registrada_en = registro.registrada_en
     WHERE registro.acceso_ref = p_cierre.acceso_ref
       AND registro.registrada_en = p_cierre.registrada_en
       AND registro.tipo_consulta = 'cuadro';

    v_consulta_huella := pg_catalog.encode(pg_catalog.sha256(
        vec_contratacion_temporal.canon_consulta_cuadro_rrhh_v2(
            p_consulta
        )
    ), 'hex');
    v_filtros_huella := pg_catalog.encode(pg_catalog.sha256(
        vec_contratacion_temporal.canon_familia_cuadro_rrhh_v2(
            p_consulta
        )
    ), 'hex');
    IF p_estado.es_continuacion IS DISTINCT FROM
           (p_consulta.cursor <> '')
       OR (
           p_estado.es_continuacion
           AND p_estado.token_presentado_huella_sha256
               IS DISTINCT FROM pg_catalog.encode(
                   pg_catalog.sha256(pg_catalog.convert_to(
                       p_consulta.cursor, 'UTF8'
                   )), 'hex'
               )
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'efectos de cursor RRHH rechazados';
    END IF;
    v_decision_huella := pg_catalog.encode(
        pg_catalog.sha256(p_decision_canonica), 'hex'
    );
    IF v_alcance.tipo_consulta <> 'cuadro'
       OR v_alcance.organizacion_ref IS DISTINCT FROM
          p_alcance.organizacion_ref
       OR v_alcance.clase_ambito IS DISTINCT FROM p_alcance.clase_ambito
       OR v_alcance.ambito_ref IS DISTINCT FROM p_alcance.ambito_ref
       OR v_registro.decision_ref IS DISTINCT FROM p_consumo.decision_ref
       OR v_registro.prueba_tipo_consulta IS DISTINCT FROM 'cuadro'
       OR v_registro.prueba_generada_en IS DISTINCT FROM
          p_cierre.generada_en
       OR v_registro.consulta_huella_sha256 IS DISTINCT FROM
          v_consulta_huella
       OR v_registro.prueba_total IS DISTINCT FROM p_cierre.total
       OR v_registro.prueba_total IS DISTINCT FROM
          pg_catalog.cardinality(v_registro.prueba_resumenes)
       OR v_registro.prueba_hay_mas IS DISTINCT FROM p_salida.hay_mas
       OR v_registro.prueba_cursor_huella_sha256 IS DISTINCT FROM
          p_cierre.cursor_huella_sha256
       OR p_cierre.cursor_huella_sha256 IS DISTINCT FROM
          (CASE WHEN p_salida.hay_mas
                THEN pg_catalog.encode(p_salida.cursor_huella, 'hex') ELSE '' END)
       OR v_registro.decision_huella_sha256 IS DISTINCT FROM
          v_decision_huella
       OR v_registro.consumo_vec_huella_sha256 IS DISTINCT FROM
          p_consumo.consumo_huella_sha256
       OR p_cierre.consumo_vec_huella_sha256 IS DISTINCT FROM
          p_consumo.consumo_huella_sha256 THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'efectos de cursor RRHH rechazados';
    END IF;

    IF p_estado.es_continuacion THEN
        SELECT familia.organizacion_ref, familia.clase_ambito,
               familia.ambito_ref, familia.actor_ref, familia.perfil_ref,
               familia.perfil_version, familia.sesion_ref,
               familia.sesion_huella_sha256, familia.dominio_filtros,
               familia.filtros_huella_sha256, familia.limite,
               familia.corte_global,
               familia.creada_en, familia.valida_hasta,
               cursor.token_huella_sha256, cursor.pagina,
               cursor.emitida_en, cursor.acceso_emision_ref,
               cursor.ultimo_actualizado_en,
               cursor.ultimo_expediente_ref, causal.revision
          INTO STRICT v_estado_durable
          FROM vec_contratacion_temporal
               .familia_cursor_cuadro_rrhh familia
          JOIN vec_contratacion_temporal.cursor_cuadro_rrhh cursor
            USING (familia_ref)
          JOIN vec_contratacion_temporal
               .control_causal_familia_cursor_rrhh causal
            USING (familia_ref)
         WHERE familia.familia_ref = p_estado.familia_ref
           AND cursor.token_huella_sha256 =
               p_estado.token_presentado_huella_sha256
         FOR UPDATE OF causal
         FOR SHARE OF familia, cursor;
        IF v_estado_durable.revision <> 0
           OR EXISTS (
               SELECT 1
                 FROM vec_contratacion_temporal
                      .revocacion_familia_cursor_rrhh revocacion
                WHERE revocacion.familia_ref = p_estado.familia_ref
           )
           OR EXISTS (
               SELECT 1
                 FROM vec_contratacion_temporal
                      .consumo_cursor_cuadro_rrhh consumo
                WHERE consumo.token_huella_sha256 =
                      p_estado.token_presentado_huella_sha256
           )
           OR v_estado_durable.organizacion_ref IS DISTINCT FROM
              v_alcance.organizacion_ref
           OR v_estado_durable.clase_ambito IS DISTINCT FROM
              v_alcance.clase_ambito
           OR v_estado_durable.ambito_ref IS DISTINCT FROM
              v_alcance.ambito_ref
           OR v_estado_durable.actor_ref IS DISTINCT FROM v_alcance.actor_ref
           OR v_estado_durable.perfil_ref IS DISTINCT FROM
              v_alcance.perfil_ref
           OR v_estado_durable.perfil_version IS DISTINCT FROM
              v_alcance.perfil_version
           OR v_estado_durable.sesion_ref IS DISTINCT FROM
              v_alcance.sesion_ref
           OR v_estado_durable.sesion_huella_sha256 IS DISTINCT FROM
              v_alcance.sesion_huella_sha256
           OR v_estado_durable.dominio_filtros IS DISTINCT FROM
              'vec.contratacion_temporal.filtros_rrhh.cuadro.v2'
           OR v_estado_durable.filtros_huella_sha256 IS DISTINCT FROM
              v_filtros_huella
           OR v_estado_durable.limite IS DISTINCT FROM p_consulta.limite
           OR v_estado_durable.corte_global IS DISTINCT FROM
              p_estado.corte_global
           OR v_estado_durable.pagina IS DISTINCT FROM
              p_estado.pagina_presentada
           OR v_estado_durable.emitida_en IS DISTINCT FROM
              p_estado.cursor_emitida_en
           OR v_estado_durable.acceso_emision_ref IS DISTINCT FROM
              p_estado.acceso_emision_ref
           OR v_estado_durable.creada_en IS DISTINCT FROM
              p_estado.familia_creada_en
           OR v_estado_durable.valida_hasta IS DISTINCT FROM
              p_estado.familia_valida_hasta
           OR v_estado_durable.ultimo_actualizado_en IS DISTINCT FROM
              p_estado.ultimo_actualizado_en
           OR v_estado_durable.ultimo_expediente_ref IS DISTINCT FROM
              p_estado.ultimo_expediente_ref THEN
            RAISE EXCEPTION USING ERRCODE = '42501',
                MESSAGE = 'efectos de cursor RRHH rechazados';
        END IF;
    ELSE
        SELECT ultimo_corte AS corte_global
          INTO STRICT v_estado_durable
          FROM vec_contratacion_temporal.control_publicacion_rrhh
         WHERE control;
        IF p_estado.familia_ref IS NOT NULL
           OR p_estado.corte_global IS DISTINCT FROM
              v_estado_durable.corte_global
           OR p_estado.pagina_presentada IS DISTINCT FROM 0
           OR p_estado.token_presentado_huella_sha256 IS NOT NULL
           OR p_estado.acceso_emision_ref IS NOT NULL
           OR p_estado.cursor_emitida_en IS NOT NULL
           OR p_estado.familia_creada_en IS NOT NULL
           OR p_estado.familia_valida_hasta IS NOT NULL
           OR p_estado.ultimo_actualizado_en IS NOT NULL
           OR p_estado.ultimo_expediente_ref IS NOT NULL THEN
            RAISE EXCEPTION USING ERRCODE = '42501',
                MESSAGE = 'efectos de cursor RRHH rechazados';
        END IF;
    END IF;

    IF p_salida.hay_mas
       AND (
           v_registro.prueba_total = 0
           OR (v_registro.prueba_resumenes[
                   v_registro.prueba_total
               ]).actualizado_en IS DISTINCT FROM
              p_salida.ultimo_actualizado_en
           OR (v_registro.prueba_resumenes[
                   v_registro.prueba_total
               ]).expediente_ref IS DISTINCT FROM
              p_salida.ultimo_expediente_ref
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'efectos de cursor RRHH rechazados';
    END IF;

    IF NOT p_estado.es_continuacion AND NOT p_salida.hay_mas THEN
        IF v_alcance.familia_ref IS NOT NULL
           THEN
            RAISE EXCEPTION USING ERRCODE = '42501',
                MESSAGE = 'efectos de cursor RRHH rechazados';
        END IF;
        RETURN;
    END IF;

    IF p_salida.hay_mas
       AND (
           p_salida.cursor_siguiente !~ '^[A-Za-z0-9_-]{43}$'
           OR p_salida.token_nuevo_huella_sha256 !~ '^[0-9a-f]{64}$'
           OR p_salida.cursor_huella IS DISTINCT FROM pg_catalog.sha256(
               pg_catalog.decode(pg_catalog.rpad(pg_catalog.translate(
                   p_salida.cursor_siguiente, '-_', '+/'
               ), 44, '='), 'base64')
           )
           OR pg_catalog.rtrim(pg_catalog.translate(pg_catalog.encode(
               pg_catalog.decode(pg_catalog.rpad(pg_catalog.translate(
                   p_salida.cursor_siguiente, '-_', '+/'
               ), 44, '='), 'base64'), 'base64'
           ), '+/', '-_'), E'=\n') IS DISTINCT FROM p_salida.cursor_siguiente
           OR p_salida.token_nuevo_huella_sha256 IS DISTINCT FROM
              pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
                  p_salida.cursor_siguiente, 'UTF8'
              )), 'hex')
           OR p_salida.ultimo_actualizado_en IS NULL
           OR p_salida.ultimo_expediente_ref IS NULL
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'efectos de cursor RRHH rechazados';
    END IF;

    IF NOT p_estado.es_continuacion THEN
        IF NOT p_salida.hay_mas
           OR v_alcance.familia_ref IS DISTINCT FROM p_salida.familia_ref
           OR p_salida.pagina_nueva IS DISTINCT FROM 2
           OR p_salida.padre_token_huella_sha256 IS NOT NULL
           OR p_estado.corte_global NOT BETWEEN
              1 AND 9007199254740991::numeric THEN
            RAISE EXCEPTION USING ERRCODE = '42501',
                MESSAGE = 'efectos de cursor RRHH rechazados';
        END IF;
        v_valida_hasta := p_cierre.registrada_en + interval '5 minutes';
        v_familia_prueba := pg_catalog.convert_to(
            'VEC-CT-FAMILIA-CURSOR-CUADRO-RRHH-V1'
                || pg_catalog.chr(10), 'UTF8'
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            p_salida.familia_ref
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            v_alcance.organizacion_ref
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            v_alcance.clase_ambito
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            v_alcance.ambito_ref
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(v_alcance.actor_ref)
        || vec_contratacion_temporal.encuadrar_texto_v1(
            v_alcance.perfil_ref
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            v_alcance.perfil_version::text
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            v_alcance.sesion_ref
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            v_alcance.sesion_huella_sha256
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            'vec.contratacion_temporal.filtros_rrhh.cuadro.v2'
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(v_filtros_huella)
        || vec_contratacion_temporal.encuadrar_texto_v1(
            p_consulta.limite::text
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            p_estado.corte_global::text
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            vec_contratacion_temporal.instante_utc_v1(
                p_cierre.registrada_en
            )
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            vec_contratacion_temporal.instante_utc_v1(v_valida_hasta)
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            p_cierre.acceso_ref
        );
        INSERT INTO
        vec_contratacion_temporal.familia_cursor_cuadro_rrhh (
            familia_ref, organizacion_ref, clase_ambito, ambito_ref,
            actor_ref, perfil_ref, perfil_version, sesion_ref,
            sesion_huella_sha256, dominio_filtros,
            filtros_huella_sha256, limite, corte_global, creada_en,
            valida_hasta, acceso_origen_ref, prueba_canonica,
            prueba_huella_sha256
        ) VALUES (
            p_salida.familia_ref, v_alcance.organizacion_ref,
            v_alcance.clase_ambito, v_alcance.ambito_ref,
            v_alcance.actor_ref, v_alcance.perfil_ref,
            v_alcance.perfil_version, v_alcance.sesion_ref,
            v_alcance.sesion_huella_sha256,
            'vec.contratacion_temporal.filtros_rrhh.cuadro.v2',
            v_filtros_huella, p_consulta.limite,
            p_estado.corte_global, p_cierre.registrada_en,
            v_valida_hasta, p_cierre.acceso_ref, v_familia_prueba,
            pg_catalog.encode(
                pg_catalog.sha256(v_familia_prueba), 'hex'
            )
        );
        INSERT INTO
        vec_contratacion_temporal.control_causal_familia_cursor_rrhh (
            familia_ref, familia_creada_en, revision, actualizada_en
        ) VALUES (
            p_salida.familia_ref, p_cierre.registrada_en, 0,
            p_cierre.registrada_en
        );
        v_padre_emitida_en := NULL;
    ELSE
        IF v_alcance.familia_ref IS DISTINCT FROM p_estado.familia_ref
           OR p_salida.familia_ref IS DISTINCT FROM
              (CASE WHEN p_salida.hay_mas
                    THEN p_estado.familia_ref ELSE NULL END)
           OR p_salida.padre_token_huella_sha256 IS DISTINCT FROM
              (CASE WHEN p_salida.hay_mas
                    THEN p_estado.token_presentado_huella_sha256
                    ELSE NULL END)
           OR p_salida.pagina_nueva IS DISTINCT FROM
              (CASE WHEN p_salida.hay_mas
                    THEN p_estado.pagina_presentada + 1 ELSE 0 END)
           OR p_cierre.registrada_en >= p_estado.familia_valida_hasta THEN
            RAISE EXCEPTION USING ERRCODE = '42501',
                MESSAGE = 'efectos de cursor RRHH rechazados';
        END IF;
        v_consumo_prueba := pg_catalog.convert_to(
            'VEC-CT-CONSUMO-CURSOR-CUADRO-RRHH-V1'
                || pg_catalog.chr(10), 'UTF8'
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            p_estado.token_presentado_huella_sha256
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            p_estado.familia_ref
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            p_consumo.decision_ref
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(v_decision_huella)
        || vec_contratacion_temporal.encuadrar_texto_v1(
            p_consumo.consumo_huella_sha256
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            p_estado.acceso_emision_ref
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            p_cierre.acceso_ref
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            vec_contratacion_temporal.instante_utc_v1(
                p_estado.cursor_emitida_en
            )
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            vec_contratacion_temporal.instante_utc_v1(
                p_estado.familia_valida_hasta
            )
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            vec_contratacion_temporal.instante_utc_v1(
                p_cierre.registrada_en
            )
        );
        INSERT INTO
        vec_contratacion_temporal.consumo_cursor_cuadro_rrhh (
            token_huella_sha256, familia_ref, decision_ref,
            decision_huella_sha256, consumo_vec_huella_sha256,
            acceso_emision_ref, acceso_consumo_ref, cursor_emitida_en,
            familia_valida_hasta, consumido_en, prueba_canonica,
            prueba_huella_sha256
        ) VALUES (
            p_estado.token_presentado_huella_sha256,
            p_estado.familia_ref, p_consumo.decision_ref,
            v_decision_huella, p_consumo.consumo_huella_sha256,
            p_estado.acceso_emision_ref, p_cierre.acceso_ref,
            p_estado.cursor_emitida_en, p_estado.familia_valida_hasta,
            p_cierre.registrada_en, v_consumo_prueba,
            pg_catalog.encode(
                pg_catalog.sha256(v_consumo_prueba), 'hex'
            )
        );
        v_valida_hasta := p_estado.familia_valida_hasta;
        v_padre_emitida_en := p_estado.cursor_emitida_en;
    END IF;

    IF p_salida.hay_mas THEN
        v_cursor_prueba := pg_catalog.convert_to(
            'VEC-CT-CURSOR-CUADRO-RRHH-V1'
                || pg_catalog.chr(10), 'UTF8'
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            p_salida.token_nuevo_huella_sha256
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            p_salida.familia_ref
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            COALESCE(p_salida.padre_token_huella_sha256, '')
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            p_salida.pagina_nueva::text
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            CASE WHEN v_padre_emitida_en IS NULL THEN ''
                 ELSE vec_contratacion_temporal.instante_utc_v1(
                     v_padre_emitida_en
                 ) END
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            vec_contratacion_temporal.instante_utc_v1(
                p_salida.ultimo_actualizado_en
            )
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            p_salida.ultimo_expediente_ref
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            vec_contratacion_temporal.instante_utc_v1(
                CASE WHEN p_estado.es_continuacion
                     THEN p_estado.familia_creada_en
                     ELSE p_cierre.registrada_en END
            )
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            vec_contratacion_temporal.instante_utc_v1(v_valida_hasta)
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            vec_contratacion_temporal.instante_utc_v1(
                p_cierre.registrada_en
            )
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            p_cierre.acceso_ref
        );
        INSERT INTO vec_contratacion_temporal.cursor_cuadro_rrhh (
            token_huella_sha256, familia_ref,
            padre_token_huella_sha256, pagina, padre_emitida_en,
            ultimo_actualizado_en, ultimo_expediente_ref,
            familia_creada_en, familia_valida_hasta, emitida_en,
            acceso_emision_ref, prueba_canonica, prueba_huella_sha256
        ) VALUES (
            p_salida.token_nuevo_huella_sha256, p_salida.familia_ref,
            p_salida.padre_token_huella_sha256,
            p_salida.pagina_nueva, v_padre_emitida_en,
            p_salida.ultimo_actualizado_en,
            p_salida.ultimo_expediente_ref,
            CASE WHEN p_estado.es_continuacion
                 THEN p_estado.familia_creada_en
                 ELSE p_cierre.registrada_en END,
            v_valida_hasta, p_cierre.registrada_en,
            p_cierre.acceso_ref, v_cursor_prueba,
            pg_catalog.encode(pg_catalog.sha256(v_cursor_prueba), 'hex')
        );
    END IF;
EXCEPTION
    WHEN SQLSTATE '40001' OR SQLSTATE '40P01'
      OR SQLSTATE '55P03' OR SQLSTATE '57014' THEN
        RAISE;
    WHEN OTHERS THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'efectos de cursor RRHH rechazados';
END;
$funcion$;

CREATE FUNCTION
vec_contratacion_temporal.cerrar_prueba_resultado_recibo_rrhh_v3(
    p_contexto
        vec_contratacion_temporal.contexto_cierre_prueba_rrhh_v3,
    p_contenido
        vec_contratacion_temporal.contenido_cierre_prueba_rrhh_v2,
    p_consumo
        vec_contratacion_temporal.evidencia_consumo_nuevo_rrhh_v3,
    p_capacidad_canonica bytea,
    p_decision_canonica bytea,
    p_motivo_canonico bytea,
    p_contexto_actor_canonico bytea,
    p_persona_version numeric,
    p_perfil_version numeric,
    p_payload_vec_ad_3 bytea,
    p_sobre_cose_sign_1 bytea,
    p_evidencia_verificacion bytea,
    p_raiz_publica_spki bytea
)
RETURNS
    vec_contratacion_temporal.resultado_cierre_prueba_rrhh_v2
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
PARALLEL UNSAFE
SET search_path = pg_catalog, pg_temp
SET row_security = 'on'
SET timezone = 'UTC'
SET lock_timeout = '1s'
SET statement_timeout = '12s'
AS $funcion$
DECLARE
    v_capacidad jsonb;
    v_decision jsonb;
    v_contexto_actor jsonb;
    v_tipo text;
    v_accion text;
    v_finalidad text;
    v_audiencia text;
    v_dominio_consulta text;
    v_tipo_recurso text;
    v_expediente_ref text;
    v_version_expediente numeric(20, 0);
    v_total smallint;
    v_consulta_canonica bytea;
    v_consulta_huella text;
    v_contexto_recurso_canonico bytea;
    v_contexto_recurso_huella text;
    v_contenido_canonico bytea;
    v_contenido_huella text;
    v_cursor_huella text;
    v_resultado_canonico bytea;
    v_resultado_huella text;
    v_material_huella text;
    v_capacidad_huella text;
    v_decision_huella text;
    v_revalidacion record;
    v_registro_peticion jsonb;
    v_registro_salida jsonb;
    v_registro record;
    v_identidad record;
    v_alcance vec_contratacion_temporal.alcance_acceso_rrhh%ROWTYPE;
    v_recibo_evidencia
        vec_contratacion_temporal.evidencia_recibo_lectura_rrhh_v2;
    v_recibo_canonico bytea;
    v_recibo_sello text;
    v_esquema_constante text :=
        'vec.contratacion-temporal.recibo-acceso-rrhh.o4-05.v2';
BEGIN
    IF CURRENT_USER <>
           'vec_contratacion_temporal_propietario'
       OR SESSION_USER =
          'vec_contratacion_temporal_propietario'
       OR pg_catalog.pg_is_in_recovery()
       OR pg_catalog.current_setting('transaction_isolation')
          <> 'serializable'
       OR pg_catalog.current_setting('transaction_read_only') <> 'off'
       OR p_contexto IS NULL
       OR p_contenido IS NULL
       OR p_consumo IS NULL
       OR p_consumo.consumo_nuevo IS DISTINCT FROM true
       OR p_consumo.consumida_en IS NULL
       OR p_consumo.consumida_en <>
          pg_catalog.date_trunc('microseconds', p_consumo.consumida_en)
       OR p_contenido.generada_en IS NULL
       OR p_contenido.generada_en <
          p_consumo.consumida_en THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'cierre de prueba RRHH rechazado';
    END IF;

    BEGIN
        v_capacidad := pg_catalog.convert_from(
            p_capacidad_canonica, 'UTF8'
        )::jsonb;
        v_decision := pg_catalog.convert_from(
            p_decision_canonica, 'UTF8'
        )::jsonb;
        v_contexto_actor := pg_catalog.convert_from(
            p_contexto_actor_canonico, 'UTF8'
        )::jsonb;
    EXCEPTION
        WHEN data_exception OR invalid_text_representation
          OR character_not_in_repertoire
          OR untranslatable_character THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'cierre de prueba RRHH inválido';
    END;

    v_tipo := p_contenido.tipo_consulta;
    IF v_tipo = 'cuadro' AND p_contexto.consulta_cuadro_v2 IS NOT NULL THEN
        IF p_contexto.consulta_cuadro IS NOT NULL OR p_contexto.consulta_detalle IS NOT NULL THEN
            RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='cierre de prueba RRHH inválido';
        END IF;
    ELSIF p_contexto.consulta_cuadro_v2 IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='cierre de prueba RRHH inválido';
    END IF;
    IF v_tipo = 'cuadro' THEN
        v_accion := 'contratacion_temporal.cuadro.consultar';
        v_finalidad := 'gestion_operativa_contratacion_temporal';
        v_audiencia :=
            'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1';
        v_dominio_consulta :=
            'vec.contratacion_temporal.consulta_rrhh.cuadro.v1';
        v_tipo_recurso := 'cuadro_rrhh_contratacion_temporal';
        IF (p_contexto.consulta_cuadro IS NULL AND p_contexto.consulta_cuadro_v2 IS NULL)
           OR (p_contexto.consulta_cuadro IS NOT NULL AND p_contexto.consulta_cuadro_v2 IS NOT NULL)
           OR p_contexto.consulta_detalle IS DISTINCT FROM
               NULL::vec_contratacion_temporal.consulta_detalle_rrhh_v1
           OR p_contenido.resumenes IS NULL
           OR p_contenido.hay_mas IS NULL
           OR p_contenido.cursor_huella IS NULL
           OR p_contenido.detalle IS DISTINCT FROM
               NULL::vec_contratacion_temporal
                   .entrada_detalle_expediente_rrhh_v1 THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'cierre de prueba RRHH inválido';
        END IF;
        IF p_contexto.consulta_cuadro_v2 IS NOT NULL THEN
            v_dominio_consulta := 'vec.contratacion_temporal.consulta_rrhh.cuadro.v2';
            v_consulta_canonica := vec_contratacion_temporal.canon_consulta_cuadro_rrhh_v2(p_contexto.consulta_cuadro_v2);
        ELSE
            v_consulta_canonica := vec_contratacion_temporal.canon_consulta_cuadro_rrhh_v1(p_contexto.consulta_cuadro);
        END IF;
        v_contenido_canonico :=
            vec_contratacion_temporal.canon_contenido_cuadro_rrhh_v1(
                p_contenido.generada_en, p_contenido.resumenes,
                p_contenido.hay_mas, p_contenido.cursor_huella
            );
        v_total := pg_catalog.cardinality(
            p_contenido.resumenes
        )::smallint;
        v_cursor_huella := CASE
            WHEN p_contenido.hay_mas THEN
                pg_catalog.encode(p_contenido.cursor_huella, 'hex')
            ELSE ''
        END;
    ELSIF v_tipo = 'detalle' THEN
        v_accion := 'contratacion_temporal.expediente.consultar';
        v_finalidad :=
            'tramitacion_expediente_contratacion_temporal';
        v_audiencia :=
            'vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1';
        v_dominio_consulta :=
            'vec.contratacion_temporal.consulta_rrhh.detalle.v1';
        v_tipo_recurso := 'expediente_contratacion_temporal';
        IF p_contexto.consulta_cuadro IS NOT NULL
           OR p_contexto.consulta_cuadro_v2 IS NOT NULL
           OR p_contexto.consulta_detalle IS NOT DISTINCT FROM
               NULL::vec_contratacion_temporal.consulta_detalle_rrhh_v1
           OR p_contenido.detalle IS NOT DISTINCT FROM
               NULL::vec_contratacion_temporal
                   .entrada_detalle_expediente_rrhh_v1
           OR p_contenido.resumenes IS NULL
           OR p_contenido.hay_mas IS DISTINCT FROM false
           OR pg_catalog.cardinality(p_contenido.resumenes) <> 0
           OR pg_catalog.array_ndims(p_contenido.resumenes) IS NOT NULL
           OR p_contenido.cursor_huella IS NULL
           OR pg_catalog.octet_length(p_contenido.cursor_huella) <> 0
           OR p_contexto.familia_ref IS NOT NULL THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'cierre de prueba RRHH inválido';
        END IF;
        v_consulta_canonica :=
            vec_contratacion_temporal.canon_consulta_detalle_rrhh_v1(
                p_contexto.consulta_detalle
            );
        v_contenido_canonico :=
            vec_contratacion_temporal.canon_contenido_detalle_rrhh_v1(
                p_contenido.generada_en, p_contenido.detalle
            );
        v_expediente_ref :=
            (p_contenido.detalle).resumen.expediente_ref;
        v_version_expediente :=
            (p_contenido.detalle).resumen.version;
        IF (p_contexto.consulta_detalle).expediente_ref
               <> v_expediente_ref
           OR v_version_expediente IS NULL
           OR v_version_expediente <= 0
           OR ((p_contexto.consulta_detalle).version_observada <> 0
               AND (p_contexto.consulta_detalle).version_observada <> v_version_expediente) THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'cierre de prueba RRHH inválido';
        END IF;
        v_total := 1;
        v_cursor_huella := '';
    ELSE
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'cierre de prueba RRHH inválido';
    END IF;

    v_consulta_huella := pg_catalog.encode(
        pg_catalog.sha256(v_consulta_canonica), 'hex'
    );
    v_contexto_recurso_canonico := pg_catalog.convert_to(
        '{"ambitos":{"ambito_ref":"'
        || p_contexto.ambito_ref
        || '","clase_ambito":"' || p_contexto.clase_ambito
        || '","organizacion_ref":"' || p_contexto.organizacion_ref
        || '"},"atributos":{"consulta_dominio":"'
        || v_dominio_consulta
        || '","consulta_huella_sha256":"' || v_consulta_huella
        || '"}}', 'UTF8'
    );
    v_contexto_recurso_huella := pg_catalog.encode(
        pg_catalog.sha256(v_contexto_recurso_canonico), 'hex'
    );

    v_contenido_huella := pg_catalog.encode(
        pg_catalog.sha256(v_contenido_canonico), 'hex'
    );
    v_resultado_canonico :=
        vec_contratacion_temporal
        .canon_resultado_consulta_rrhh_puro_v1(ROW(
            v_tipo, p_contenido.generada_en, v_total,
            v_contenido_huella, v_cursor_huella
        )::vec_contratacion_temporal.evidencia_resultado_rrhh_v1);
    v_resultado_huella := pg_catalog.encode(
        pg_catalog.sha256(v_resultado_canonico), 'hex'
    );
    v_material_huella :=
        vec_contratacion_temporal.huella_material_consumo_rrhh_v3(
            p_capacidad_canonica, p_decision_canonica,
            p_motivo_canonico, p_contexto_actor_canonico,
            p_persona_version, p_perfil_version,
            p_payload_vec_ad_3, p_sobre_cose_sign_1,
            p_evidencia_verificacion, p_raiz_publica_spki
        );
    v_capacidad_huella := pg_catalog.encode(
        pg_catalog.sha256(p_capacidad_canonica), 'hex'
    );
    v_decision_huella := pg_catalog.encode(
        pg_catalog.sha256(p_decision_canonica), 'hex'
    );

    IF p_contexto.organizacion_ref IS NULL
       OR p_contexto.organizacion_ref !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_contexto.clase_ambito NOT IN (
           'organizacion', 'centro', 'unidad_gestion'
       )
       OR p_contexto.ambito_ref IS NULL
       OR p_contexto.ambito_ref !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR (
           p_contexto.clase_ambito = 'organizacion'
           AND p_contexto.ambito_ref <> p_contexto.organizacion_ref
       )
       OR (
           p_contexto.familia_ref IS NOT NULL
           AND p_contexto.familia_ref !~
               '^familia:cursor:rrhh:[0-9a-f]{32}$'
       )
       OR v_capacidad ->> 'decision_ref' IS DISTINCT FROM
          p_consumo.decision_ref
       OR v_capacidad ->> 'efecto_ref' IS DISTINCT FROM
          p_consumo.efecto_ref
       OR v_capacidad ->> 'huella_efecto_sha256' IS DISTINCT FROM
          p_consumo.huella_efecto_sha256
       OR v_capacidad ->> 'huella_efecto_sha256' IS DISTINCT FROM
          v_contexto_recurso_huella
       OR v_capacidad ->> 'operacion' IS DISTINCT FROM v_accion
       OR v_capacidad ->> 'audiencia_consumo' IS DISTINCT FROM
          v_audiencia
       OR v_capacidad ->> 'huella_decision_sha256' IS DISTINCT FROM
          v_decision_huella
       OR v_decision ->> 'decision_ref' IS DISTINCT FROM
          p_consumo.decision_ref
       OR v_decision ->> 'recurso_ref' IS DISTINCT FROM
          (CASE WHEN v_tipo = 'cuadro'
                THEN p_contexto.ambito_ref ELSE v_expediente_ref END)
       OR v_decision ->> 'modulo_id' IS DISTINCT FROM
          'contratacion_temporal'
       OR v_decision ->> 'tipo_recurso' IS DISTINCT FROM
          v_tipo_recurso
       OR v_decision ->> 'contexto_recurso_huella_sha256'
          IS DISTINCT FROM v_contexto_recurso_huella
       OR v_decision ->> 'accion' IS DISTINCT FROM v_accion
       OR v_decision ->> 'finalidad' IS DISTINCT FROM v_finalidad
       OR v_decision ->> 'principal_id' IS DISTINCT FROM
          v_contexto_actor ->> 'principal_ref'
       OR v_decision ->> 'perfil_activo_ref' IS DISTINCT FROM
          v_contexto_actor ->> 'perfil_activo_ref'
       OR v_contexto_actor ->> 'perfil_version' IS DISTINCT FROM
          p_perfil_version::text
       OR v_contexto_actor ->> 'persona_version' IS DISTINCT FROM
          p_persona_version::text
       OR pg_catalog.encode(
          pg_catalog.sha256(p_contexto_actor_canonico), 'hex'
       ) IS DISTINCT FROM v_capacidad ->> 'huella_contexto_sha256'
       OR p_consumo.consumo_huella_sha256 !~ '^[0-9a-f]{64}$'
       OR p_consumo.auditoria_ref !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_consumo.auditoria_huella_sha256 !~ '^[0-9a-f]{64}$'
       OR EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal.registro_acceso_rrhh registro
            WHERE registro.consumo_vec_huella_sha256 =
                  p_consumo.consumo_huella_sha256
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'cierre de prueba RRHH rechazado';
    END IF;

    IF v_tipo = 'cuadro' THEN
        SELECT *
          INTO STRICT v_revalidacion
          FROM vec_autorizacion_atestada_v3
               .revalidar_evidencia_consumo_consulta_cuadro_rrhh_v3_atestada(
              p_capacidad_canonica, p_decision_canonica,
              p_motivo_canonico, p_contexto_actor_canonico,
              p_persona_version, p_perfil_version,
              p_payload_vec_ad_3, p_sobre_cose_sign_1,
              p_evidencia_verificacion, p_raiz_publica_spki
          );
    ELSE
        SELECT *
          INTO STRICT v_revalidacion
          FROM vec_autorizacion_atestada_v3
               .revalidar_evidencia_consumo_consulta_detalle_rrhh_v3_atestada(
              p_capacidad_canonica, p_decision_canonica,
              p_motivo_canonico, p_contexto_actor_canonico,
              p_persona_version, p_perfil_version,
              p_payload_vec_ad_3, p_sobre_cose_sign_1,
              p_evidencia_verificacion, p_raiz_publica_spki
          );
    END IF;
    IF v_revalidacion.decision_ref IS DISTINCT FROM
           p_consumo.decision_ref
       OR v_revalidacion.efecto_ref IS DISTINCT FROM
          p_consumo.efecto_ref
       OR v_revalidacion.huella_efecto_sha256 IS DISTINCT FROM
          p_consumo.huella_efecto_sha256
       OR v_revalidacion.consumo_huella_sha256 IS DISTINCT FROM
          p_consumo.consumo_huella_sha256
       OR v_revalidacion.auditoria_ref IS DISTINCT FROM
          p_consumo.auditoria_ref
       OR v_revalidacion.auditoria_huella_sha256 IS DISTINCT FROM
          p_consumo.auditoria_huella_sha256
       OR v_revalidacion.consumida_en IS DISTINCT FROM
          p_consumo.consumida_en
       OR v_revalidacion.revalidada_en IS NULL
       OR v_revalidacion.revalidada_en <
          p_consumo.consumida_en
       OR v_revalidacion.revalidada_en <
          p_contenido.generada_en THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'cierre de prueba RRHH rechazado';
    END IF;

    v_registro_peticion := pg_catalog.jsonb_build_object(
        'registro', pg_catalog.jsonb_build_object(
            'accion', v_accion,
            'actor_ref', v_decision ->> 'principal_id',
            'ambito_ref', p_contexto.ambito_ref,
            'audiencia', v_audiencia,
            'auditoria_vec_huella_sha256',
                p_consumo.auditoria_huella_sha256,
            'auditoria_vec_ref', p_consumo.auditoria_ref,
            'capacidad_huella_sha256', v_capacidad_huella,
            'consulta_huella_sha256', v_consulta_huella,
            'consumo_vec_huella_sha256',
                p_consumo.consumo_huella_sha256,
            'correlacion_ref', v_decision ->> 'correlacion_ref',
            'decision_huella_sha256', v_decision_huella,
            'decision_ref', p_consumo.decision_ref,
            'dominio_huella_consulta', v_dominio_consulta,
            'expediente_ref', v_expediente_ref,
            'finalidad', v_finalidad,
            'modulo_id', 'contratacion_temporal',
            'organizacion_ref', p_contexto.organizacion_ref,
            'perfil_id', v_decision ->> 'perfil_activo_ref',
            'perfil_version', p_perfil_version,
            'recurso_ref', p_consumo.efecto_ref,
            'recurso_tipo', v_tipo_recurso,
            'resultado_generico', 'entregado',
            'resultado_huella_sha256', v_resultado_huella,
            'sesion_huella_sha256',
                v_decision #>>
                '{vinculo_autenticacion_actor,control_sesion_huella_sha256}',
            'sesion_id',
                v_decision #>>
                '{vinculo_autenticacion_actor,sesion_ref}',
            'tipo_consulta', v_tipo,
            'total', v_total,
            'version_expediente', v_version_expediente
        ),
        'alcance', pg_catalog.jsonb_build_object(
            'clase_ambito', p_contexto.clase_ambito,
            'familia_ref', p_contexto.familia_ref
        ),
        'identidad', pg_catalog.jsonb_build_object(
            'actor_ref', v_decision ->> 'principal_id',
            'autenticacion_huella_sha256',
                v_decision #>>
                '{vinculo_autenticacion_actor,autenticacion_huella_sha256}',
            'autenticacion_ref',
                v_decision #>>
                '{vinculo_autenticacion_actor,autenticacion_ref}',
            'control_sesion_huella_sha256',
                v_decision #>>
                '{vinculo_autenticacion_actor,control_sesion_huella_sha256}',
            'control_sesion_ref',
                v_decision #>>
                '{vinculo_autenticacion_actor,control_sesion_ref}',
            'control_sesion_revision',
                v_decision #>>
                '{vinculo_autenticacion_actor,control_sesion_revision}',
            'organizacion_ref', p_contexto.organizacion_ref,
            'perfil_ref', v_decision ->> 'perfil_activo_ref',
            'perfil_version', p_perfil_version,
            'sesion_ref',
                v_decision #>>
                '{vinculo_autenticacion_actor,sesion_ref}'
        )
    );
    v_registro_salida :=
        vec_contratacion_temporal.registrar_acceso_rrhh_interno_v2(
            v_registro_peticion
        );

    SELECT *
      INTO STRICT v_registro
      FROM vec_contratacion_temporal.registro_acceso_rrhh
     WHERE acceso_ref = v_registro_salida ->> 'acceso_ref';
    SELECT *
      INTO STRICT v_identidad
      FROM vec_contratacion_temporal.vinculo_identidad_acceso_rrhh_v2
     WHERE acceso_ref = v_registro.acceso_ref;
    IF v_tipo = 'cuadro' THEN
        SELECT *
          INTO STRICT v_alcance
          FROM vec_contratacion_temporal.alcance_acceso_rrhh
         WHERE acceso_ref = v_registro.acceso_ref;
    END IF;

    v_recibo_evidencia := ROW(
        v_esquema_constante, v_registro.acceso_ref,
        v_registro.secuencia, v_registro.anterior_sha256,
        v_registro.huella_sha256,
        v_identidad.prueba_huella_sha256,
        CASE WHEN v_tipo = 'cuadro'
             THEN v_alcance.prueba_huella_sha256 ELSE '' END,
        v_registro.registrada_en, v_registro.auditoria_vec_ref,
        v_registro.auditoria_vec_huella_sha256,
        v_registro.consumo_vec_huella_sha256,
        v_registro.decision_ref, v_registro.decision_huella_sha256,
        v_registro.capacidad_huella_sha256, v_material_huella,
        v_registro.consulta_huella_sha256,
        v_registro.correlacion_ref,
        v_identidad.autenticacion_ref,
        v_identidad.autenticacion_huella_sha256,
        v_identidad.sesion_ref, v_identidad.control_sesion_ref,
        v_identidad.control_sesion_revision,
        v_identidad.control_sesion_huella_sha256,
        v_identidad.actor_ref, v_identidad.perfil_ref,
        v_identidad.perfil_version, v_identidad.organizacion_ref,
        v_identidad.clase_ambito, v_identidad.ambito_ref,
        v_accion, v_finalidad, COALESCE(v_expediente_ref, ''),
        COALESCE(v_version_expediente, 0), v_total,
        v_contenido_huella, v_resultado_huella,
        v_cursor_huella, p_contenido.generada_en
    )::vec_contratacion_temporal.evidencia_recibo_lectura_rrhh_v2;
    v_recibo_canonico :=
        vec_contratacion_temporal.canon_recibo_lectura_rrhh_v2(
            v_recibo_evidencia
        );
    v_recibo_sello := pg_catalog.encode(
        pg_catalog.sha256(v_recibo_canonico), 'hex'
    );

    INSERT INTO
    vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2 (
        acceso_ref, tipo_consulta, expediente_ref,
        version_expediente, total, generada_en,
        resumenes, hay_mas, cursor_material_huella_sha256,
        detalle,
        contenido_canonico, contenido_huella_sha256,
        cursor_huella_sha256, resultado_canonico,
        resultado_huella_sha256, material_huella_sha256,
        revalidada_en, recibo_canonico, recibo_sello_sha256,
        secuencia, anterior_sha256, huella_sha256,
        vinculo_identidad_huella_sha256, alcance_acceso_ref,
        alcance_huella_sha256, registrada_en,
        auditoria_vec_ref, auditoria_vec_huella_sha256,
        consumo_vec_huella_sha256, decision_ref,
        decision_huella_sha256, capacidad_huella_sha256,
        consulta_huella_sha256, correlacion_ref,
        autenticacion_ref, autenticacion_huella_sha256,
        sesion_ref, sesion_huella_sha256, control_sesion_ref,
        control_sesion_revision, control_sesion_huella_sha256,
        actor_ref, perfil_ref, perfil_version, organizacion_ref,
        clase_ambito, ambito_ref, accion, finalidad
    ) VALUES (
        v_registro.acceso_ref, v_tipo, v_expediente_ref,
        v_version_expediente, v_total, p_contenido.generada_en,
        p_contenido.resumenes, p_contenido.hay_mas,
        p_contenido.cursor_huella, p_contenido.detalle,
        v_contenido_canonico, v_contenido_huella,
        NULLIF(v_cursor_huella, ''), v_resultado_canonico,
        v_resultado_huella, v_material_huella,
        v_revalidacion.revalidada_en, v_recibo_canonico,
        v_recibo_sello, v_registro.secuencia,
        v_registro.anterior_sha256, v_registro.huella_sha256,
        v_identidad.prueba_huella_sha256,
        CASE WHEN v_tipo = 'cuadro'
             THEN v_alcance.acceso_ref ELSE NULL END,
        CASE WHEN v_tipo = 'cuadro'
             THEN v_alcance.prueba_huella_sha256 ELSE NULL END,
        v_registro.registrada_en, v_registro.auditoria_vec_ref,
        v_registro.auditoria_vec_huella_sha256,
        v_registro.consumo_vec_huella_sha256,
        v_registro.decision_ref, v_registro.decision_huella_sha256,
        v_registro.capacidad_huella_sha256,
        v_registro.consulta_huella_sha256,
        v_registro.correlacion_ref,
        v_identidad.autenticacion_ref,
        v_identidad.autenticacion_huella_sha256,
        v_identidad.sesion_ref, v_identidad.sesion_huella_sha256,
        v_identidad.control_sesion_ref,
        v_identidad.control_sesion_revision,
        v_identidad.control_sesion_huella_sha256,
        v_identidad.actor_ref, v_identidad.perfil_ref,
        v_identidad.perfil_version, v_identidad.organizacion_ref,
        v_identidad.clase_ambito, v_identidad.ambito_ref,
        v_accion, v_finalidad
    );

    RETURN ROW(
        v_esquema_constante, v_registro.acceso_ref,
        v_registro.secuencia, v_registro.anterior_sha256,
        v_registro.huella_sha256,
        v_identidad.prueba_huella_sha256,
        CASE WHEN v_tipo = 'cuadro'
             THEN v_alcance.prueba_huella_sha256 ELSE '' END,
        v_registro.registrada_en, v_registro.auditoria_vec_ref,
        v_registro.auditoria_vec_huella_sha256,
        v_registro.consumo_vec_huella_sha256,
        v_contenido_huella, v_resultado_huella,
        v_cursor_huella, p_contenido.generada_en,
        COALESCE(v_expediente_ref, ''),
        COALESCE(v_version_expediente, 0), v_total,
        v_recibo_sello
    )::vec_contratacion_temporal.resultado_cierre_prueba_rrhh_v2;
EXCEPTION
    WHEN SQLSTATE '40001' OR SQLSTATE '40P01'
      OR SQLSTATE '55P03' OR SQLSTATE '57014' THEN
        RAISE;
    WHEN OTHERS THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'cierre de prueba RRHH rechazado';
END
$funcion$;

CREATE FUNCTION
vec_contratacion_temporal.motor_consultar_cuadro_rrhh_v2(
    p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    p_consulta vec_contratacion_temporal.consulta_cuadro_rrhh_v2,
    p_material
        vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3
)
RETURNS vec_contratacion_temporal.resultado_motor_cuadro_rrhh_v1
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
PARALLEL UNSAFE
SET search_path = pg_catalog, pg_temp
SET row_security = 'on'
SET timezone = 'UTC'
SET lock_timeout = '1s'
SET statement_timeout = '4s'
SET idle_in_transaction_session_timeout = '6s'
AS $funcion$
DECLARE
    v_decision jsonb;
    v_contexto_actor jsonb;
    v_actor_ref text;
    v_perfil_ref text;
    v_sesion_ref text;
    v_sesion_huella text;
    v_consulta_canonica bytea;
    v_contexto_recurso bytea;
    v_contexto_huella text;
    v_estado
        vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1;
    v_materializacion
        vec_contratacion_temporal.materializacion_cuadro_rrhh_v1;
    v_salida
        vec_contratacion_temporal.salida_cursor_cuadro_rrhh_v1;
    v_consumo
        vec_contratacion_temporal.evidencia_consumo_nuevo_rrhh_v3;
    v_contexto
        vec_contratacion_temporal.contexto_cierre_prueba_rrhh_v3;
    v_contenido
        vec_contratacion_temporal.contenido_cierre_prueba_rrhh_v2;
    v_cierre
        vec_contratacion_temporal.resultado_cierre_prueba_rrhh_v2;
    v_generada_en timestamptz(6);
    v_familia_ref text;
BEGIN
    PERFORM
        vec_contratacion_temporal.acreditar_contexto_motor_consultas_rrhh_v1(
            p_alcance, p_material
        );
    PERFORM vec_contratacion_temporal.canon_consulta_cuadro_rrhh_v2(
        p_consulta
    );

    -- El consumo sucede exactamente una vez y antes de cualquier lectura.
    v_consumo :=
        vec_contratacion_temporal
        .consumir_autorizacion_motor_consultas_rrhh_v1(
            'cuadro', p_material
        );
    v_decision := pg_catalog.convert_from(
        p_material.decision_canonica, 'UTF8'
    )::jsonb;
    v_contexto_actor := pg_catalog.convert_from(
        p_material.contexto_actor_canonico, 'UTF8'
    )::jsonb;
    IF v_decision ->> 'decision_ref' IS DISTINCT FROM
           v_consumo.decision_ref
       OR v_decision ->> 'principal_id' IS DISTINCT FROM
          v_contexto_actor ->> 'principal_ref'
       OR v_decision ->> 'perfil_activo_ref' IS DISTINCT FROM
          v_contexto_actor ->> 'perfil_activo_ref'
       OR v_contexto_actor ->> 'persona_version' IS DISTINCT FROM
          p_material.persona_version::text
       OR v_contexto_actor ->> 'perfil_version' IS DISTINCT FROM
          p_material.perfil_version::text THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta de cuadro RRHH rechazada';
    END IF;
    -- No existe un validador puro reutilizable para esta ligadura. Se
    -- reconstruye el mismo contexto canónico de CT43 antes de leer datos.
    v_consulta_canonica :=
        vec_contratacion_temporal.canon_consulta_cuadro_rrhh_v2(
            p_consulta
        );
    v_contexto_recurso := pg_catalog.convert_to(
        '{"ambitos":{"ambito_ref":"' || p_alcance.ambito_ref
        || '","clase_ambito":"' || p_alcance.clase_ambito
        || '","organizacion_ref":"' || p_alcance.organizacion_ref
        || '"},"atributos":{"consulta_dominio":"'
        || 'vec.contratacion_temporal.consulta_rrhh.cuadro.v2'
        || '","consulta_huella_sha256":"'
        || pg_catalog.encode(
            pg_catalog.sha256(v_consulta_canonica), 'hex'
        ) || '"}}', 'UTF8'
    );
    v_contexto_huella := pg_catalog.encode(
        pg_catalog.sha256(v_contexto_recurso), 'hex'
    );
    IF v_decision ->> 'accion' IS DISTINCT FROM
           'contratacion_temporal.cuadro.consultar'
       OR v_decision ->> 'modulo_id' IS DISTINCT FROM
          'contratacion_temporal'
       OR v_decision ->> 'tipo_recurso' IS DISTINCT FROM
          'cuadro_rrhh_contratacion_temporal'
       OR v_decision ->> 'finalidad' IS DISTINCT FROM
          'gestion_operativa_contratacion_temporal'
       OR v_decision ->> 'recurso_ref' IS DISTINCT FROM
          p_alcance.ambito_ref
       OR v_decision ->> 'contexto_recurso_huella_sha256'
          IS DISTINCT FROM v_contexto_huella
       OR v_consumo.efecto_ref IS DISTINCT FROM p_alcance.ambito_ref
       OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM
          v_contexto_huella THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta de cuadro RRHH rechazada';
    END IF;
    v_actor_ref := v_decision ->> 'principal_id';
    v_perfil_ref := v_decision ->> 'perfil_activo_ref';
    v_sesion_ref := v_decision #>>
        '{vinculo_autenticacion_actor,sesion_ref}';
    v_sesion_huella := v_decision #>>
        '{vinculo_autenticacion_actor,control_sesion_huella_sha256}';

    v_estado :=
        vec_contratacion_temporal
        .resolver_estado_cursor_cuadro_rrhh_v2(
            p_alcance, p_consulta, v_actor_ref, v_perfil_ref,
            p_material.perfil_version, v_sesion_ref, v_sesion_huella
        );
    v_materializacion :=
        vec_contratacion_temporal.materializar_cuadro_rrhh_v2(
            p_alcance, p_consulta, v_estado
        );
    v_salida :=
        vec_contratacion_temporal
        .preparar_salida_cursor_cuadro_rrhh_v1(
            v_estado, v_materializacion
        );
    v_generada_en := pg_catalog.date_trunc(
        'microseconds', pg_catalog.clock_timestamp()
    );
    v_familia_ref := CASE
        WHEN v_estado.es_continuacion THEN v_estado.familia_ref
        WHEN v_salida.hay_mas THEN v_salida.familia_ref
        ELSE NULL
    END;
    v_contexto := ROW(
        p_alcance.organizacion_ref, p_alcance.clase_ambito,
        p_alcance.ambito_ref, NULL::vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
        p_consulta, NULL::vec_contratacion_temporal.consulta_detalle_rrhh_v1,
        v_familia_ref
    );
    v_contenido := ROW(
        'cuadro', v_generada_en, v_materializacion.resumenes,
        v_materializacion.hay_mas, v_salida.cursor_huella,
        NULL::vec_contratacion_temporal
            .entrada_detalle_expediente_rrhh_v1
    );
    v_cierre :=
        vec_contratacion_temporal.cerrar_prueba_resultado_recibo_rrhh_v3(
            v_contexto, v_contenido, v_consumo,
            p_material.capacidad_canonica,
            p_material.decision_canonica,
            p_material.motivo_canonico,
            p_material.contexto_actor_canonico,
            p_material.persona_version,
            p_material.perfil_version,
            p_material.payload_vec_ad_3,
            p_material.sobre_cose_sign_1,
            p_material.evidencia_verificacion,
            p_material.raiz_publica_spki
        );
    PERFORM
        vec_contratacion_temporal.aplicar_efectos_cursor_cuadro_rrhh_v2(
            p_alcance, p_consulta, v_estado, v_salida, v_consumo,
            p_material.decision_canonica, v_cierre
        );

    RETURN ROW(
        v_generada_en, v_materializacion.resumenes,
        v_materializacion.hay_mas, v_salida.cursor_siguiente, v_cierre
    )::vec_contratacion_temporal.resultado_motor_cuadro_rrhh_v1;
EXCEPTION
    WHEN SQLSTATE '40001' OR SQLSTATE '40P01'
      OR SQLSTATE '55P03' OR SQLSTATE '57014' THEN
        RAISE;
    WHEN OTHERS THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta de cuadro RRHH rechazada';
END;
$funcion$;

CREATE FUNCTION
vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v6_base(
    p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    p_consulta vec_contratacion_temporal.consulta_cuadro_rrhh_v2,
    p_capacidad_canonica bytea,
    p_decision_canonica bytea,
    p_motivo_canonico bytea,
    p_contexto_actor_canonico bytea,
    p_persona_version numeric,
    p_perfil_version numeric,
    p_payload_vec_ad_3 bytea,
    p_sobre_cose_sign_1 bytea,
    p_evidencia_verificacion bytea,
    p_raiz_publica_spki bytea
)
RETURNS TABLE(
    contenido_canonico bytea,
    cursor_siguiente text,
    esquema text,
    acceso_ref text,
    secuencia numeric,
    anterior_sha256 text,
    huella_sha256 text,
    vinculo_identidad_huella_sha256 text,
    alcance_huella_sha256 text,
    registrada_en timestamptz,
    auditoria_vec_ref text,
    auditoria_vec_huella_sha256 text,
    consumo_vec_huella_sha256 text,
    contenido_huella_sha256 text,
    resultado_huella_sha256 text,
    cursor_huella_sha256 text,
    generada_en timestamptz,
    expediente_ref text,
    version_expediente numeric,
    total smallint,
    recibo_sello_sha256 text
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
PARALLEL UNSAFE
SET search_path = pg_catalog, pg_temp
SET row_security = 'on'
SET timezone = 'UTC'
SET lock_timeout = '1s'
SET statement_timeout = '4s'
SET idle_in_transaction_session_timeout = '6s'
AS $funcion$
DECLARE
    v_login pg_catalog.pg_roles%ROWTYPE;
    v_capacidad jsonb;
    v_decision jsonb;
    v_consulta_canonica bytea;
    v_contexto_recurso bytea;
    v_contexto_huella text;
    v_decision_huella text;
    v_material
        vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3;
    v_resultado
        vec_contratacion_temporal.resultado_motor_cuadro_rrhh_v1;
    v_cierre
        vec_contratacion_temporal.resultado_cierre_prueba_rrhh_v2;
    v_contenido bytea;
    v_cursor_huella bytea;
BEGIN
    SELECT *
      INTO v_login
      FROM pg_catalog.pg_roles rol
     WHERE rol.rolname = SESSION_USER;
    IF CURRENT_USER <>
           'vec_contratacion_temporal_propietario'
       OR SESSION_USER = CURRENT_USER
       OR v_login.oid IS NULL
       OR NOT v_login.rolcanlogin
       OR NOT v_login.rolinherit
       OR v_login.rolsuper
       OR v_login.rolcreatedb
       OR v_login.rolcreaterole
       OR v_login.rolreplication
       OR v_login.rolbypassrls
       OR (
           SELECT pg_catalog.count(*)
             FROM pg_catalog.pg_auth_members membresia
            WHERE membresia.member = v_login.oid
       ) <> 1
       OR NOT EXISTS (
           SELECT 1
             FROM pg_catalog.pg_auth_members membresia
             JOIN pg_catalog.pg_roles grupo
               ON grupo.oid = membresia.roleid
            WHERE membresia.member = v_login.oid
              AND grupo.rolname =
                  'vec_contratacion_temporal_consultor_rrhh'
              AND NOT membresia.admin_option
              AND membresia.inherit_option
              AND NOT membresia.set_option
       )
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_auth_members membresia
            WHERE membresia.roleid = v_login.oid
       )
       OR NOT EXISTS (
           SELECT 1
             FROM pg_catalog.pg_roles grupo
            WHERE grupo.rolname =
                  'vec_contratacion_temporal_consultor_rrhh'
              AND NOT grupo.rolcanlogin
              AND grupo.rolinherit
              AND NOT grupo.rolsuper
              AND NOT grupo.rolcreatedb
              AND NOT grupo.rolcreaterole
              AND NOT grupo.rolreplication
              AND NOT grupo.rolbypassrls
       )
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_auth_members membresia
            WHERE membresia.member =
                  'vec_contratacion_temporal_consultor_rrhh'
                      ::pg_catalog.regrole
       )
       OR pg_catalog.pg_is_in_recovery()
       OR pg_catalog.current_setting('transaction_isolation')
          <> 'serializable'
       OR pg_catalog.current_setting('transaction_read_only') <> 'off'
       OR pg_catalog.current_setting('TimeZone') <> 'UTC'
       OR pg_catalog.current_setting('lock_timeout') = '0'
       OR pg_catalog.current_setting('lock_timeout')::interval >
          interval '1 second'
       OR pg_catalog.current_setting('statement_timeout') = '0'
       OR pg_catalog.current_setting('statement_timeout')::interval >
          interval '4 seconds'
       OR pg_catalog.current_setting(
           'idle_in_transaction_session_timeout'
       ) = '0'
       OR pg_catalog.current_setting(
           'idle_in_transaction_session_timeout'
       )::interval > interval '6 seconds' THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta RRHH rechazada';
    END IF;

    -- Los límites O(1) preceden a los cánones CT40 y, por tanto, a cualquier
    -- expresión regular sobre varlena exterior.
    IF p_alcance IS NOT DISTINCT FROM
           NULL::vec_contratacion_temporal.alcance_consulta_rrhh_v1
       OR pg_catalog.octet_length(
           COALESCE(p_alcance.organizacion_ref, '')
       ) > 160
       OR pg_catalog.octet_length(
           COALESCE(p_alcance.clase_ambito, '')
       ) > 16
       OR pg_catalog.octet_length(
           COALESCE(p_alcance.ambito_ref, '')
       ) > 160
       OR p_consulta IS NOT DISTINCT FROM
          NULL::vec_contratacion_temporal.consulta_cuadro_rrhh_v2
       OR pg_catalog.octet_length(
           COALESCE(p_consulta.texto, '')
       ) > 160
       OR pg_catalog.octet_length(
           COALESCE(p_consulta.centro_ref, '')
       ) > 160
       OR pg_catalog.octet_length(COALESCE(p_consulta.categoria_ref, '')) > 160
       OR pg_catalog.cardinality(p_consulta.estados_clave)>6
       OR pg_catalog.cardinality(p_consulta.fases_clave)>32
       OR pg_catalog.octet_length(
           COALESCE(p_consulta.cursor, '')
       ) > 43 THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta RRHH rechazada';
    END IF;

    -- Limita antes de decodificar o analizar cualquier pieza controlada por
    -- el llamador. Repite exactamente la frontera privada CT44.
    IF p_capacidad_canonica IS NULL
       OR pg_catalog.octet_length(
           p_capacidad_canonica
       ) NOT BETWEEN 512 AND 32768
       OR p_decision_canonica IS NULL
       OR pg_catalog.octet_length(
           p_decision_canonica
       ) NOT BETWEEN 1 AND 524288
       OR p_motivo_canonico IS NULL
       OR pg_catalog.octet_length(
           p_motivo_canonico
       ) NOT BETWEEN 1 AND 65536
       OR p_contexto_actor_canonico IS NULL
       OR pg_catalog.octet_length(
           p_contexto_actor_canonico
       ) NOT BETWEEN 1 AND 262144
       OR p_persona_version IS NULL
       OR p_persona_version NOT BETWEEN
          1 AND 9007199254740991::numeric
       OR p_persona_version <> pg_catalog.trunc(p_persona_version)
       OR p_perfil_version IS NULL
       OR p_perfil_version NOT BETWEEN
          1 AND 9007199254740991::numeric
       OR p_perfil_version <> pg_catalog.trunc(p_perfil_version)
       OR p_payload_vec_ad_3 IS NULL
       OR pg_catalog.octet_length(
           p_payload_vec_ad_3
       ) NOT BETWEEN 1 AND 1048576
       OR p_sobre_cose_sign_1 IS NULL
       OR pg_catalog.octet_length(
           p_sobre_cose_sign_1
       ) NOT BETWEEN 1 AND 1048576
       OR p_evidencia_verificacion IS NULL
       OR pg_catalog.octet_length(
           p_evidencia_verificacion
       ) NOT BETWEEN 1 AND 262144
       OR p_raiz_publica_spki IS NULL
       OR pg_catalog.octet_length(p_raiz_publica_spki) <> 44 THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta RRHH rechazada';
    END IF;

    PERFORM vec_contratacion_temporal.canon_alcance_rrhh_v1(
        p_alcance
    );
    v_consulta_canonica :=
        vec_contratacion_temporal.canon_consulta_cuadro_rrhh_v2(
            p_consulta
        );
    v_capacidad := pg_catalog.convert_from(
        p_capacidad_canonica, 'UTF8'
    )::jsonb;
    v_decision := pg_catalog.convert_from(
        p_decision_canonica, 'UTF8'
    )::jsonb;
    v_contexto_recurso := pg_catalog.convert_to(
        '{"ambitos":{"ambito_ref":"' || p_alcance.ambito_ref
        || '","clase_ambito":"' || p_alcance.clase_ambito
        || '","organizacion_ref":"' || p_alcance.organizacion_ref
        || '"},"atributos":{"consulta_dominio":"'
        || 'vec.contratacion_temporal.consulta_rrhh.cuadro.v2'
        || '","consulta_huella_sha256":"'
        || pg_catalog.encode(
            pg_catalog.sha256(v_consulta_canonica), 'hex'
        ) || '"}}', 'UTF8'
    );
    v_contexto_huella := pg_catalog.encode(
        pg_catalog.sha256(v_contexto_recurso), 'hex'
    );
    v_decision_huella := pg_catalog.encode(
        pg_catalog.sha256(p_decision_canonica), 'hex'
    );
    IF v_capacidad ->> 'operacion' IS DISTINCT FROM
           'contratacion_temporal.cuadro.consultar'
       OR v_capacidad ->> 'audiencia_consumo' IS DISTINCT FROM
          'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1'
       OR v_capacidad ->> 'efecto_ref' IS DISTINCT FROM
          p_alcance.ambito_ref
       OR v_capacidad ->> 'huella_efecto_sha256' IS DISTINCT FROM
          v_contexto_huella
       OR v_capacidad ->> 'huella_decision_sha256' IS DISTINCT FROM
          v_decision_huella
       OR v_decision ->> 'accion' IS DISTINCT FROM
          'contratacion_temporal.cuadro.consultar'
       OR v_decision ->> 'modulo_id' IS DISTINCT FROM
          'contratacion_temporal'
       OR v_decision ->> 'tipo_recurso' IS DISTINCT FROM
          'cuadro_rrhh_contratacion_temporal'
       OR v_decision ->> 'finalidad' IS DISTINCT FROM
          'gestion_operativa_contratacion_temporal'
       OR v_decision ->> 'recurso_ref' IS DISTINCT FROM
          p_alcance.ambito_ref
       OR v_decision ->> 'contexto_recurso_huella_sha256'
          IS DISTINCT FROM v_contexto_huella THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta RRHH rechazada';
    END IF;

    v_material := ROW(
        p_capacidad_canonica, p_decision_canonica,
        p_motivo_canonico, p_contexto_actor_canonico,
        p_persona_version, p_perfil_version,
        p_payload_vec_ad_3, p_sobre_cose_sign_1,
        p_evidencia_verificacion, p_raiz_publica_spki
    )::vec_contratacion_temporal
       .material_autorizacion_consulta_rrhh_v3;
    v_resultado :=
        vec_contratacion_temporal.motor_consultar_cuadro_rrhh_v2(
            p_alcance, p_consulta, v_material
        );
    v_cierre := v_resultado.cierre;
    IF v_resultado IS NOT DISTINCT FROM
           NULL::vec_contratacion_temporal.resultado_motor_cuadro_rrhh_v1
       OR v_cierre IS NOT DISTINCT FROM
          NULL::vec_contratacion_temporal.resultado_cierre_prueba_rrhh_v2
       OR v_resultado.generada_en IS NULL
       OR v_resultado.resumenes IS NULL
       OR v_resultado.hay_mas IS NULL
       OR v_resultado.cursor_siguiente IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta RRHH rechazada';
    END IF;
    IF v_resultado.hay_mas THEN
        IF (v_resultado.cursor_siguiente ~ '^[A-Za-z0-9_-]{43}$')
              IS DISTINCT FROM true
           OR (v_cierre.cursor_huella_sha256 ~ '^[0-9a-f]{64}$')
              IS DISTINCT FROM true
           OR pg_catalog.encode(pg_catalog.sha256(
               pg_catalog.decode(pg_catalog.rpad(pg_catalog.translate(
                   v_resultado.cursor_siguiente, '-_', '+/'
               ), 44, '='), 'base64')
           ), 'hex') IS DISTINCT FROM
              v_cierre.cursor_huella_sha256 THEN
            RAISE EXCEPTION USING ERRCODE = '42501',
                MESSAGE = 'consulta RRHH rechazada';
        END IF;
        v_cursor_huella := pg_catalog.decode(
            v_cierre.cursor_huella_sha256, 'hex'
        );
    ELSE
        IF v_resultado.cursor_siguiente IS DISTINCT FROM ''
           OR v_cierre.cursor_huella_sha256 IS DISTINCT FROM '' THEN
            RAISE EXCEPTION USING ERRCODE = '42501',
                MESSAGE = 'consulta RRHH rechazada';
        END IF;
        v_cursor_huella := ''::bytea;
    END IF;
    v_contenido :=
        vec_contratacion_temporal.canon_contenido_cuadro_rrhh_v1(
            v_resultado.generada_en, v_resultado.resumenes,
            v_resultado.hay_mas, v_cursor_huella
        );
    IF v_cierre.generada_en IS DISTINCT FROM
           v_resultado.generada_en
       OR v_cierre.total IS DISTINCT FROM
          pg_catalog.cardinality(v_resultado.resumenes)::smallint
       OR v_cierre.expediente_ref IS DISTINCT FROM ''
       OR v_cierre.version_expediente IS DISTINCT FROM 0
       OR v_cierre.contenido_huella_sha256 IS DISTINCT FROM
          pg_catalog.encode(pg_catalog.sha256(v_contenido), 'hex') THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta RRHH rechazada';
    END IF;

    RETURN QUERY SELECT
        v_contenido, v_resultado.cursor_siguiente,
        v_cierre.esquema, v_cierre.acceso_ref, v_cierre.secuencia,
        v_cierre.anterior_sha256, v_cierre.huella_sha256,
        v_cierre.vinculo_identidad_huella_sha256,
        v_cierre.alcance_huella_sha256, v_cierre.registrada_en,
        v_cierre.auditoria_vec_ref,
        v_cierre.auditoria_vec_huella_sha256,
        v_cierre.consumo_vec_huella_sha256,
        v_cierre.contenido_huella_sha256,
        v_cierre.resultado_huella_sha256,
        v_cierre.cursor_huella_sha256, v_cierre.generada_en,
        v_cierre.expediente_ref, v_cierre.version_expediente,
        v_cierre.total, v_cierre.recibo_sello_sha256;
EXCEPTION
    WHEN SQLSTATE '40001' OR SQLSTATE '40P01'
      OR SQLSTATE '55P03' OR SQLSTATE '57014' THEN
        RAISE;
    WHEN OTHERS THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta RRHH rechazada';
END
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v6(
 p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1,
 p_consulta vec_contratacion_temporal.consulta_cuadro_rrhh_v2,
 p_resumen boolean,
 p_capacidad_canonica bytea, p_decision_canonica bytea,
 p_motivo_canonico bytea, p_contexto_actor_canonico bytea,
 p_persona_version numeric, p_perfil_version numeric,
 p_payload_vec_ad_3 bytea, p_sobre_cose_sign_1 bytea,
 p_evidencia_verificacion bytea, p_raiz_publica_spki bytea
)
RETURNS TABLE(
 contenido_canonico bytea, cursor_siguiente text, esquema text,
 acceso_ref text, secuencia numeric, anterior_sha256 text,
 huella_sha256 text, vinculo_identidad_huella_sha256 text,
 alcance_huella_sha256 text, registrada_en timestamptz,
 auditoria_vec_ref text, auditoria_vec_huella_sha256 text,
 consumo_vec_huella_sha256 text, contenido_huella_sha256 text,
 resultado_huella_sha256 text, cursor_huella_sha256 text,
 generada_en timestamptz, expediente_ref text,
 version_expediente numeric, total smallint, recibo_sello_sha256 text,
 total_filtrado numeric, en_tramitacion numeric,
 con_incidencia numeric, en_llamamiento numeric,
 fase_desde_expedientes text[], fase_desde_instantes timestamptz[],
 urgente_expedientes boolean[],
 recuento_estados text[], recuento_fases text[], recuento_numeros numeric[],
 plazo_fases text[], plazo_desde timestamptz[], plazo_urgentes boolean[],
 plazo_numeros numeric[]
)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security='on' SET timezone='UTC'
SET lock_timeout='1s' SET statement_timeout='4s'
SET idle_in_transaction_session_timeout='6s'
AS $funcion$
DECLARE
 v_base record; v_totales record;
 v_refs text[]; v_instantes timestamptz[]; v_urgentes boolean[];
 v_filas integer; v_con_fase integer;
 v_estados text[]:='{}'; v_fases text[]:='{}'; v_numeros numeric[]:='{}';
 v_plazo_fases text[]:='{}'; v_plazo_desde timestamptz[]:='{}';
 v_plazo_urgentes boolean[]:='{}'; v_plazo_numeros numeric[]:='{}';
 v_total_recuento numeric; v_en_tramite numeric; v_total_plazos numeric; v_sin_fase integer;
BEGIN
 IF p_resumen IS NULL THEN RAISE EXCEPTION USING ERRCODE='42501', MESSAGE='consulta RRHH rechazada'; END IF;
 SELECT * INTO STRICT v_base FROM vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v6_base(
  p_alcance,p_consulta,p_capacidad_canonica,p_decision_canonica,
  p_motivo_canonico,p_contexto_actor_canonico,p_persona_version,p_perfil_version,
  p_payload_vec_ad_3,p_sobre_cose_sign_1,p_evidencia_verificacion,p_raiz_publica_spki);
 SELECT * INTO STRICT v_totales FROM vec_contratacion_temporal.contar_totales_cuadro_rrhh_v2(
  p_alcance,p_consulta,p_consulta.cursor);
 IF v_totales.total_filtrado<0 OR v_totales.en_tramitacion<0
 OR v_totales.con_incidencia<0 OR v_totales.en_llamamiento<0
 OR v_totales.en_tramitacion>v_totales.total_filtrado
 OR v_totales.con_incidencia>v_totales.total_filtrado
 OR v_totales.en_llamamiento>v_totales.total_filtrado
 THEN RAISE EXCEPTION USING ERRCODE='42501', MESSAGE='totales de cuadro RRHH no disponibles'; END IF;
 SELECT COALESCE(pg_catalog.array_agg(leido.expediente_ref ORDER BY leido.orden),'{}'),
  COALESCE(pg_catalog.array_agg(entrada.fase_desde ORDER BY leido.orden),'{}'),
  COALESCE(pg_catalog.array_agg(EXISTS (
   SELECT 1 FROM vec_contratacion_temporal.urgencia_expediente_analisis u
   WHERE u.expediente_ref=leido.expediente_ref AND u.version<=leido.version
  ) ORDER BY leido.orden),'{}'),
  pg_catalog.count(*)::integer,pg_catalog.count(entrada.fase_desde)::integer
 INTO v_refs,v_instantes,v_urgentes,v_filas,v_con_fase
 FROM vec_contratacion_temporal.expedientes_contenido_cuadro_rrhh_v1(v_base.contenido_canonico) leido
 LEFT JOIN vec_contratacion_temporal.fase_entrada_publicacion_rrhh entrada
 ON entrada.expediente_ref=leido.expediente_ref AND entrada.version=leido.version;
 IF v_filas<>v_base.total OR v_con_fase<>v_filas
 THEN RAISE EXCEPTION USING ERRCODE='42501', MESSAGE='fase de cuadro RRHH no disponible'; END IF;
 IF p_resumen THEN
  WITH resumen AS MATERIALIZED (
   SELECT * FROM vec_contratacion_temporal.contar_resumen_cuadro_rrhh_v2(
    p_alcance,p_consulta,p_consulta.cursor)),
  recuento AS (SELECT * FROM resumen WHERE clase='estado_fase'),
  plazo AS (SELECT * FROM resumen WHERE clase='plazo')
  SELECT (SELECT COALESCE(pg_catalog.array_agg(r.estado_clave ORDER BY r.estado_clave COLLATE "C",r.fase_clave COLLATE "C"),'{}') FROM recuento r),
   (SELECT COALESCE(pg_catalog.array_agg(r.fase_clave ORDER BY r.estado_clave COLLATE "C",r.fase_clave COLLATE "C"),'{}') FROM recuento r),
   (SELECT COALESCE(pg_catalog.array_agg(r.numero ORDER BY r.estado_clave COLLATE "C",r.fase_clave COLLATE "C"),'{}') FROM recuento r),
   (SELECT COALESCE(pg_catalog.sum(r.numero),0) FROM recuento r),
   (SELECT COALESCE(pg_catalog.sum(r.numero),0) FROM recuento r WHERE r.estado_clave NOT IN ('completado','cancelado')),
   (SELECT COALESCE(pg_catalog.array_agg(r.fase_clave ORDER BY r.fase_clave COLLATE "C",r.fase_desde,r.urgente),'{}') FROM plazo r),
   (SELECT COALESCE(pg_catalog.array_agg(r.fase_desde ORDER BY r.fase_clave COLLATE "C",r.fase_desde,r.urgente),'{}') FROM plazo r),
   (SELECT COALESCE(pg_catalog.array_agg(r.urgente ORDER BY r.fase_clave COLLATE "C",r.fase_desde,r.urgente),'{}') FROM plazo r),
   (SELECT COALESCE(pg_catalog.array_agg(r.numero ORDER BY r.fase_clave COLLATE "C",r.fase_desde,r.urgente),'{}') FROM plazo r),
   (SELECT COALESCE(pg_catalog.sum(r.numero),0) FROM plazo r),
   (SELECT pg_catalog.count(*)::integer FROM plazo r WHERE r.fase_desde IS NULL)
  INTO v_estados,v_fases,v_numeros,v_total_recuento,v_en_tramite,
   v_plazo_fases,v_plazo_desde,v_plazo_urgentes,v_plazo_numeros,v_total_plazos,v_sin_fase;
  IF v_total_recuento<>v_totales.total_filtrado OR v_total_plazos<>v_en_tramite OR v_sin_fase<>0
  THEN RAISE EXCEPTION USING ERRCODE='42501', MESSAGE='resumen de cuadro RRHH no disponible'; END IF;
 END IF;
 RETURN QUERY SELECT v_base.contenido_canonico,v_base.cursor_siguiente,v_base.esquema,
 v_base.acceso_ref,v_base.secuencia,v_base.anterior_sha256,v_base.huella_sha256,
 v_base.vinculo_identidad_huella_sha256,v_base.alcance_huella_sha256,v_base.registrada_en,
 v_base.auditoria_vec_ref,v_base.auditoria_vec_huella_sha256,v_base.consumo_vec_huella_sha256,
 v_base.contenido_huella_sha256,v_base.resultado_huella_sha256,v_base.cursor_huella_sha256,
 v_base.generada_en,v_base.expediente_ref,v_base.version_expediente,v_base.total,v_base.recibo_sello_sha256,
 v_totales.total_filtrado,v_totales.en_tramitacion,v_totales.con_incidencia,v_totales.en_llamamiento,
 v_refs,v_instantes,v_urgentes,v_estados,v_fases,v_numeros,
 v_plazo_fases,v_plazo_desde,v_plazo_urgentes,v_plazo_numeros;
EXCEPTION WHEN SQLSTATE '40001' OR SQLSTATE '40P01' OR SQLSTATE '55P03' OR SQLSTATE '57014' THEN RAISE;
 WHEN OTHERS THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='consulta RRHH rechazada';
END $funcion$;


REVOKE ALL ON FUNCTION vec_contratacion_temporal.resolver_estado_cursor_cuadro_rrhh_v2(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v2,text,text,numeric,text,text) FROM PUBLIC;

REVOKE ALL ON FUNCTION vec_contratacion_temporal.materializar_cuadro_rrhh_v2(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v2,vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1) FROM PUBLIC;

REVOKE ALL ON FUNCTION vec_contratacion_temporal.contar_totales_cuadro_rrhh_v2(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v2,text) FROM PUBLIC;

REVOKE ALL ON FUNCTION vec_contratacion_temporal.contar_resumen_cuadro_rrhh_v2(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v2,text) FROM PUBLIC;

REVOKE ALL ON FUNCTION vec_contratacion_temporal.aplicar_efectos_cursor_cuadro_rrhh_v2(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v2,vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1,vec_contratacion_temporal.salida_cursor_cuadro_rrhh_v1,vec_contratacion_temporal.evidencia_consumo_nuevo_rrhh_v3,bytea,vec_contratacion_temporal.resultado_cierre_prueba_rrhh_v2) FROM PUBLIC;

REVOKE ALL ON FUNCTION vec_contratacion_temporal.cerrar_prueba_resultado_recibo_rrhh_v3(vec_contratacion_temporal.contexto_cierre_prueba_rrhh_v3,vec_contratacion_temporal.contenido_cierre_prueba_rrhh_v2,vec_contratacion_temporal.evidencia_consumo_nuevo_rrhh_v3,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;

REVOKE ALL ON FUNCTION vec_contratacion_temporal.motor_consultar_cuadro_rrhh_v2(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v2,vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3) FROM PUBLIC;

REVOKE ALL ON FUNCTION vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v6_base(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v2,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;

REVOKE ALL ON FUNCTION vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v6(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v2,boolean,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;

GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v6(
 vec_contratacion_temporal.alcance_consulta_rrhh_v1,
 vec_contratacion_temporal.consulta_cuadro_rrhh_v2,
 boolean,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea
) TO vec_contratacion_temporal_consultor_rrhh;
COMMIT;
