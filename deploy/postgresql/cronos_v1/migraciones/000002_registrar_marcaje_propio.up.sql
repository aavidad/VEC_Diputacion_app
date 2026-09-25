\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_cronos_v1_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_cronos_v1:000002',0));
DO $pre$
BEGIN
    IF to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_marcaje_propio_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL THEN
        RAISE EXCEPTION 'falta consumidor nominal Cronos V3' USING ERRCODE='55000';
    END IF;
END
$pre$;

CREATE FUNCTION vec_cronos_v1.registrar_marcaje_propio_v1(
    p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,
    p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s'
AS $funcion$
DECLARE
    m jsonb; d jsonb; x jsonb; c jsonb; canal jsonb; vinculo jsonb;
    contexto text; huella text; huella_contexto text; ref text;
    instante timestamptz(6); ahora timestamptz(6); vence_en timestamptz;
    consumo record; previa vec_cronos_v1.marcaje_original%ROWTYPE;
    recibo jsonb; recibo_ref text;
BEGIN
    IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 4096 THEN
        RAISE EXCEPTION 'material Cronos inválido' USING ERRCODE='PC001';
    END IF;
    BEGIN
        m:=p_material::jsonb; canal:=m->'canal';
        d:=convert_from(p_decision,'UTF8')::jsonb;
        c:=convert_from(p_capacidad,'UTF8')::jsonb;
        x:=convert_from(p_contexto,'UTF8')::jsonb;
        IF jsonb_typeof(m) IS DISTINCT FROM 'object'
           OR (SELECT count(*) FROM jsonb_object_keys(m))<>7
           OR m-ARRAY['actor_ref','perfil_ref','empleado_ref','clave_operacion','movimiento','instante_utc','canal']<>'{}'::jsonb
           OR jsonb_typeof(canal) IS DISTINCT FROM 'object'
           OR (SELECT count(*) FROM jsonb_object_keys(canal))<>4
           OR canal-ARRAY['politica_version_ref','canal_ref','origen_ref','calidad_ref']<>'{}'::jsonb
           OR EXISTS (SELECT 1 FROM jsonb_each(m-'canal') e WHERE jsonb_typeof(e.value) IS DISTINCT FROM 'string')
           OR EXISTS (SELECT 1 FROM jsonb_each(canal) e WHERE jsonb_typeof(e.value) IS DISTINCT FROM 'string')
           OR EXISTS (SELECT 1 FROM jsonb_each_text(canal) e WHERE e.value !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{0,127}$')
           OR coalesce(m->>'actor_ref','') !~ '^per_[A-Za-z0-9_-]{22,128}$'
           OR coalesce(m->>'perfil_ref','') !~ '^prf_[A-Za-z0-9_-]{22,128}$'
           OR coalesce(m->>'empleado_ref','') !~ '^emp_[A-Za-z0-9_-]{22,128}$'
           OR coalesce(m->>'clave_operacion','') !~ '^[A-Za-z0-9][A-Za-z0-9_-]{7,127}$'
           OR coalesce(m->>'movimiento','') NOT IN ('entrada','salida','inicio_pausa','fin_pausa')
           OR coalesce(m->>'instante_utc','') !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}([.][0-9]{1,6})?Z$' THEN
            RAISE EXCEPTION 'estructura Cronos inválida' USING ERRCODE='PC001';
        END IF;
        instante:=(m->>'instante_utc')::timestamptz;
        ahora:=date_trunc('microseconds',clock_timestamp());
        IF instante>ahora THEN
            RAISE EXCEPTION 'instante Cronos inválido' USING ERRCODE='PC001';
        END IF;
        IF x->>'principal_ref' IS DISTINCT FROM m->>'actor_ref'
           OR x->>'perfil_activo_ref' IS DISTINCT FROM m->>'perfil_ref'
           OR jsonb_typeof(x->'vinculos') IS DISTINCT FROM 'array' THEN
            RAISE EXCEPTION 'contexto Cronos divergente' USING ERRCODE='PC003';
        END IF;
        -- V3 verifica la procedencia y vigencia de este contexto completo.
        -- Se exige además un único vínculo empleado activo, nunca del cliente.
        SELECT e INTO STRICT vinculo FROM jsonb_array_elements(x->'vinculos') e
         WHERE e->>'tipo'='empleado' AND e->>'estado'='activo'
           AND (e->>'vigente_desde')::timestamptz<=ahora
           AND ahora<(e->>'vigente_hasta')::timestamptz;
        IF vinculo->>'referencia' IS DISTINCT FROM m->>'empleado_ref' THEN
            RAISE EXCEPTION 'empleado Cronos divergente' USING ERRCODE='PC003';
        END IF;
    EXCEPTION WHEN data_exception OR no_data_found OR too_many_rows THEN
        RAISE EXCEPTION 'estructura Cronos inválida' USING ERRCODE='PC001';
    END;
    ref:='marcaje:cronos:'||(m->>'clave_operacion');
    huella:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
    contexto:='{"ambitos":{"empleado_ref":'||to_jsonb(m->>'empleado_ref')::text||'},"atributos":{"material_sha256":"'||huella||'"}}';
    huella_contexto:=encode(sha256(convert_to(contexto,'UTF8')),'hex');
    IF d->>'accion' IS DISTINCT FROM 'cronos.marcaje.propio.registrar'
       OR d->>'modulo_id' IS DISTINCT FROM 'cronos'
       OR d->>'tipo_recurso' IS DISTINCT FROM 'marcaje_propio'
       OR d->>'finalidad' IS DISTINCT FROM 'registrar_marcaje_propio'
       OR d->>'recurso_ref' IS DISTINCT FROM ref
       OR d->>'principal_id' IS DISTINCT FROM m->>'actor_ref'
       OR d->>'perfil_activo_ref' IS DISTINCT FROM m->>'perfil_ref'
       OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM huella_contexto THEN
        RAISE EXCEPTION 'autorización Cronos divergente' USING ERRCODE='PC003';
    END IF;
    -- Esperar al escritor de esta clave ANTES de consumir V3. Una concesión
    -- que expire durante la espera debe ser rechazada por la autoridad viva.
    PERFORM pg_advisory_xact_lock(hashtextextended('vec_cronos_v1:clave:'||(m->>'clave_operacion'),0));
    SELECT * INTO STRICT consumo
      FROM vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_marcaje_propio_v3_atestada(
        p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
        p_payload,p_sobre,p_evidencia,p_raiz);
    IF consumo.consumo_nuevo IS NOT TRUE OR consumo.efecto_ref IS DISTINCT FROM ref
       OR consumo.huella_efecto_sha256 IS DISTINCT FROM huella_contexto THEN
        RAISE EXCEPTION 'consumo Cronos divergente' USING ERRCODE='PC003';
    END IF;
    -- El consumidor mantiene sus bloqueos de revocación hasta COMMIT. Su
    -- cadena de auditoría también puede haber esperado: revisar los plazos
    -- ya atestados con reloj vivo, incluido el vínculo empleado, antes de leer
    -- un recibo o escribir negocio. Esta comprobación sólo restringe V3.
    ahora:=date_trunc('microseconds',clock_timestamp());
    -- Una relación adicional podría comenzar durante la espera: la unicidad
    -- del vínculo activo también se evalúa con el instante actual.
    IF (SELECT count(*) FROM jsonb_array_elements(x->'vinculos') e
         WHERE e->>'tipo'='empleado' AND e->>'estado'='activo'
           AND (e->>'vigente_desde')::timestamptz<=ahora
           AND ahora<(e->>'vigente_hasta')::timestamptz) <> 1 THEN
        RAISE EXCEPTION 'vínculo Cronos ambiguo o caducado' USING ERRCODE='PC003';
    END IF;
    IF (ahora < (c->>'expira_en')::timestamptz
        AND ahora < (c->>'decision_valida_hasta')::timestamptz
        AND ahora < (c->>'configuracion_expira_en')::timestamptz
        AND ahora < (c->>'raiz_valida_hasta')::timestamptz
        AND ahora < (d->>'valida_hasta')::timestamptz
        AND ahora < (x->>'vigente_hasta')::timestamptz
        AND ahora < (vinculo->>'vigente_hasta')::timestamptz) IS NOT TRUE THEN
        RAISE EXCEPTION 'vigencia Cronos agotada' USING ERRCODE='PC003';
    END IF;
    vence_en:=least((c->>'expira_en')::timestamptz,
        (c->>'decision_valida_hasta')::timestamptz,
        (c->>'configuracion_expira_en')::timestamptz,
        (c->>'raiz_valida_hasta')::timestamptz,
        (d->>'valida_hasta')::timestamptz,
        (x->>'vigente_hasta')::timestamptz,
        (vinculo->>'vigente_hasta')::timestamptz);
    -- El contexto RLS dura esta transacción; no se reutiliza entre peticiones.
    IF nullif(current_setting('vec.cronos.empleado_ref',true),'') IS NOT NULL THEN
        RAISE EXCEPTION 'contexto Cronos ya establecido' USING ERRCODE='PC003';
    END IF;
    PERFORM set_config('vec.cronos.empleado_ref',m->>'empleado_ref',true);
    SELECT * INTO previa FROM vec_cronos_v1.marcaje_original
     WHERE clave_operacion=m->>'clave_operacion';
    IF FOUND THEN
        IF previa.actor_ref IS DISTINCT FROM m->>'actor_ref'
           OR previa.perfil_ref IS DISTINCT FROM m->>'perfil_ref'
           OR previa.empleado_ref IS DISTINCT FROM m->>'empleado_ref' THEN
            RAISE EXCEPTION 'operación Cronos ajena' USING ERRCODE='PC003';
        END IF;
        IF (previa.material::jsonb-'instante_utc') IS DISTINCT FROM (m-'instante_utc') THEN
            RAISE EXCEPTION 'clave Cronos en conflicto' USING ERRCODE='PC002';
        END IF;
        INSERT INTO vec_cronos_v1.marcaje_acceso(decision_ref,marcaje_ref,empleado_ref,auditoria_ref,consumo_huella_sha256,consultada_en)
        VALUES(consumo.decision_ref,ref,m->>'empleado_ref',consumo.auditoria_ref,consumo.consumo_huella_sha256,ahora);
        IF clock_timestamp() >= vence_en THEN
            RAISE EXCEPTION 'vigencia Cronos agotada' USING ERRCODE='PC003';
        END IF;
        RETURN previa.recibo_json||jsonb_build_object('replay',true);
    END IF;
    recibo_ref:='recibo:cronos:'||gen_random_uuid()::text;
    recibo:=jsonb_build_object('referencia',recibo_ref,'instante_utc',instante,'marcaje_original_ref',ref,'replay',false);
    INSERT INTO vec_cronos_v1.marcaje_original(
        marcaje_ref,empleado_ref,clave_operacion,actor_ref,perfil_ref,material,material_sha256,
        movimiento,instante_utc,recibo_ref,recibo_json,auditoria_ref,decision_ref,consumo_huella_sha256,registrada_en)
    VALUES(ref,m->>'empleado_ref',m->>'clave_operacion',m->>'actor_ref',m->>'perfil_ref',p_material,huella,
        m->>'movimiento',instante,recibo_ref,recibo,consumo.auditoria_ref,consumo.decision_ref,consumo.consumo_huella_sha256,ahora);
    INSERT INTO vec_cronos_v1.marcaje_historia(marcaje_ref,empleado_ref,version,material_sha256,registrada_en)
    VALUES(ref,m->>'empleado_ref',1,huella,ahora);
    INSERT INTO vec_cronos_v1.marcaje_outbox(evento_ref,marcaje_ref,empleado_ref,tipo,carga_json,creada_en)
    VALUES('evento:cronos:'||gen_random_uuid()::text,ref,m->>'empleado_ref','cronos.marcaje.propio.registrado',
        jsonb_build_object('marcaje_original_ref',ref,'recibo_ref',recibo_ref,'version',1),ahora);
    IF clock_timestamp() >= vence_en THEN
        RAISE EXCEPTION 'vigencia Cronos agotada' USING ERRCODE='PC003';
    END IF;
    RETURN recibo;
EXCEPTION WHEN unique_violation THEN
    RAISE EXCEPTION 'clave Cronos en conflicto' USING ERRCODE='PC002';
END
$funcion$;
REVOKE ALL ON FUNCTION vec_cronos_v1.registrar_marcaje_propio_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC,vec_cronos_v1_migrador;
GRANT EXECUTE ON FUNCTION vec_cronos_v1.registrar_marcaje_propio_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_cronos_v1_ejecutor;
COMMIT;
