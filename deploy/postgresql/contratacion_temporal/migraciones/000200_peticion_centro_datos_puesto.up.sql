\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog, pg_temp;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_contratacion_temporal:000200',0));

-- Preimagen literal del clon causal HZ15. Cualquier divergencia detiene la instalación.
DO $preimagen_ct200$
DECLARE item record; actual text;
BEGIN
 FOR item IN SELECT * FROM (VALUES
  ('vec_contratacion_temporal.registrar_peticion_centro_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','1bec5f865a1ce3ac32ab9811b59a6234a1ff5a75d7d23f3645e9bff716c25f7e'),
  ('vec_contratacion_temporal.confirmar_alta_atestada_v1(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea)','35d99b5e553f95ca06a67c3ae11854eb842482c201b315edab3c6081ffc9a218'),
  ('vec_contratacion_temporal.reconstruir_solicitud_efecto_v2(jsonb)','b3db277720f71c2334317d8aaea207cb67aef00a851cff967aa2e7c88d1612cd')
 ) AS esperado(firma,huella) LOOP
   IF pg_catalog.to_regprocedure(item.firma) IS NULL THEN
     RAISE EXCEPTION 'CT200: PARO clave=funcion esperado=% actual=ausente',item.firma USING ERRCODE='55000';
   END IF;
   IF (SELECT p.proowner::pg_catalog.regrole FROM pg_catalog.pg_proc p
       WHERE p.oid=pg_catalog.to_regprocedure(item.firma))
      IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::pg_catalog.regrole THEN
     RAISE EXCEPTION 'CT200: PARO clave=propietario_funcion firma=% esperado=% actual=%',
       item.firma,'vec_contratacion_temporal_propietario',
       (SELECT p.proowner::pg_catalog.regrole FROM pg_catalog.pg_proc p
        WHERE p.oid=pg_catalog.to_regprocedure(item.firma)) USING ERRCODE='55000';
   END IF;
   SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.pg_get_functiondef(pg_catalog.to_regprocedure(item.firma)),'UTF8')),'hex') INTO actual;
   IF actual IS DISTINCT FROM item.huella THEN
     RAISE EXCEPTION 'CT200: PARO clave=% esperado=% actual=%',item.firma,item.huella,actual USING ERRCODE='55000';
   END IF;
 END LOOP;
 IF pg_catalog.to_regprocedure('vec_contratacion_temporal.datos_puesto_peticion_validos_ct200(jsonb)') IS NOT NULL THEN
   RAISE EXCEPTION 'CT200: PARO clave=helper esperado=ausente actual=presente' USING ERRCODE='55000';
 END IF;
END $preimagen_ct200$;

CREATE FUNCTION vec_contratacion_temporal.datos_puesto_peticion_validos_ct200(s jsonb)
RETURNS boolean LANGUAGE sql IMMUTABLE STRICT
SET search_path = pg_catalog, pg_temp AS $funcion$
 SELECT CASE WHEN s ?| ARRAY['jornada_minutos','numero_personas','puesto_solicitado'] THEN
   s ?& ARRAY['jornada_minutos','numero_personas','puesto_solicitado']
   AND (CASE WHEN pg_catalog.jsonb_typeof(s->'jornada_minutos')='number'
                   AND s->>'jornada_minutos' ~ '^[0-9]+$'
             THEN (s->>'jornada_minutos')::numeric BETWEEN 1 AND 10080 ELSE false END)
   AND (CASE WHEN pg_catalog.jsonb_typeof(s->'numero_personas')='number'
                   AND s->>'numero_personas' ~ '^[0-9]+$'
             THEN (s->>'numero_personas')::numeric BETWEEN 1 AND 4294967295 ELSE false END)
   AND pg_catalog.jsonb_typeof(s->'puesto_solicitado')='string'
   AND pg_catalog.length(s->>'puesto_solicitado') BETWEEN 1 AND 160
   AND pg_catalog.btrim(s->>'puesto_solicitado')=s->>'puesto_solicitado'
   AND (s->>'puesto_solicitado') !~ '[[:cntrl:]]'
 ELSE true END
$funcion$;

CREATE OR REPLACE FUNCTION vec_contratacion_temporal.reconstruir_solicitud_efecto_v2(s jsonb)
 RETURNS text
 LANGUAGE sql
 IMMUTABLE STRICT
 SET search_path TO 'pg_catalog'
AS $function$
    SELECT
      '{"centro_ref":' || vec_contratacion_temporal.texto_json_go_v1(s ->> 'centro_ref') ||
      ',"contacto_ref":' || vec_contratacion_temporal.texto_json_go_v1(s ->> 'contacto_ref') ||
      ',"categoria_ref":' || vec_contratacion_temporal.texto_json_go_v1(s ->> 'categoria_ref') ||
      ',"grupo_subgrupo":' || vec_contratacion_temporal.texto_json_go_v1(s ->> 'grupo_subgrupo') ||
      ',"motivo_clave":' || vec_contratacion_temporal.texto_json_go_v1(s ->> 'motivo_clave') ||
      ',"detalle":' || vec_contratacion_temporal.texto_json_go_v1(s ->> 'detalle') ||
      ',"periodo":{"inicio":' || vec_contratacion_temporal.texto_json_go_v1(s #>> '{periodo,inicio}') ||
      (CASE WHEN s #> '{periodo,fin}' IS NOT NULL THEN
          ',"fin":' || vec_contratacion_temporal.texto_json_go_v1(s #>> '{periodo,fin}')
        ELSE
          ',"causa_fin":' || vec_contratacion_temporal.texto_json_go_v1(s #>> '{periodo,causa_fin}')
        END) ||
        (CASE WHEN s #> '{periodo,politica_fin}' IS NOT NULL THEN
          ',"politica_fin":{"regla_ref":' ||
          vec_contratacion_temporal.texto_json_go_v1(s #>> '{periodo,politica_fin,regla_ref}') ||
          ',"catalogo_version":' || (s #> '{periodo,politica_fin,catalogo_version}')::text ||
          ',"catalogo_huella_sha256":' ||
          vec_contratacion_temporal.texto_json_go_v1(s #>> '{periodo,politica_fin,catalogo_huella_sha256}') ||
          ',"fecha_fin":' ||
          vec_contratacion_temporal.texto_json_go_v1(s #>> '{periodo,politica_fin,fecha_fin}') ||
          (CASE WHEN s #> '{periodo,politica_fin,causa_fin}' IS NOT NULL THEN
            ',"causa_fin":' ||
            vec_contratacion_temporal.texto_json_go_v1(s #>> '{periodo,politica_fin,causa_fin}')
           ELSE '' END) || '}'
         ELSE '' END) || '}' ||
      ',"rc":{"existe":' || (s #> '{rc,existe}')::text ||
      ',"numero":' || vec_contratacion_temporal.texto_json_go_v1(s #>> '{rc,numero}') ||
      ',"fecha":' || vec_contratacion_temporal.texto_json_go_v1(s #>> '{rc,fecha}') ||
      ',"importe":{"centimos":' || (s #> '{rc,importe,centimos}')::text ||
      ',"moneda":' || vec_contratacion_temporal.texto_json_go_v1(s #>> '{rc,importe,moneda}') || '}' ||
      ',"documento_ref":' || vec_contratacion_temporal.texto_json_go_v1(s #>> '{rc,documento_ref}') || '}' ||
      ',"documentos_adjuntos":' ||
        vec_contratacion_temporal.lista_textos_json_v1(
            s -> 'documentos_adjuntos'
        ) ||
      ',"observaciones":' || vec_contratacion_temporal.texto_json_go_v1(s ->> 'observaciones') ||
      (CASE WHEN s ? 'puesto_solicitado' THEN
        ',"jornada_minutos":' || (s -> 'jornada_minutos')::text ||
        ',"numero_personas":' || (s -> 'numero_personas')::text ||
        ',"puesto_solicitado":' || vec_contratacion_temporal.texto_json_go_v1(s ->> 'puesto_solicitado')
       ELSE '' END) || '}'
$function$;

CREATE OR REPLACE FUNCTION vec_contratacion_temporal.registrar_peticion_centro_v1(p_material text, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
AS $function$
DECLARE
    m jsonb; c jsonb; a jsonb; p jsonb; cfg jsonb; sol jsonb; docs jsonb; d jsonb;
    v_operacion text; v_accion text; v_clave_text text; v_clave uuid; v_ref text; v_cfg_version bigint;
    v_material_huella text; v_contexto text; v_contexto_huella text;
    v_creada timestamptz(6); v_ratificada timestamptz(6); v_fecha timestamptz(6);
    v_consumo record; v_previa vec_contratacion_temporal.peticion_centro_revision%ROWTYPE;
    v_actual vec_contratacion_temporal.peticion_centro_revision%ROWTYPE;
    v_recibo_ref text; v_recibo jsonb; v_estado text; v_version smallint;
BEGIN
    IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 65536 THEN
        RAISE EXCEPTION 'material de petición de centro inválido' USING ERRCODE='P0670';
    END IF;
    BEGIN
        m:=p_material::jsonb; c:=m->'comando'; a:=m->'actor'; p:=m->'peticion';
        cfg:=p->'configuracion'; sol:=p->'solicitud';
        docs:=CASE WHEN sol->'documentos_adjuntos'='null'::jsonb THEN '[]'::jsonb ELSE sol->'documentos_adjuntos' END;
        v_operacion:=c->>'operacion';
        v_clave_text:=c->>'clave_idempotencia'; v_ref:=p->>'referencia';
        IF v_clave_text !~ '^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$' THEN RAISE data_exception; END IF;
        v_clave:=v_clave_text::uuid; v_cfg_version:=(cfg->>'version')::bigint;
        v_creada:=(p->>'creada_en')::timestamptz;
        IF p ? 'ratificada_en' THEN v_ratificada:=(p->>'ratificada_en')::timestamptz; END IF;
    EXCEPTION WHEN data_exception THEN
        RAISE EXCEPTION 'estructura de petición de centro inválida' USING ERRCODE='P0670';
    END;
    BEGIN
    IF jsonb_typeof(m) IS DISTINCT FROM 'object' OR (SELECT count(*) FROM jsonb_object_keys(m))<>3
       OR (m-ARRAY['comando','actor','peticion'])<>'{}'::jsonb
       OR jsonb_typeof(c) IS DISTINCT FROM 'object' OR jsonb_typeof(a) IS DISTINCT FROM 'object'
       OR jsonb_typeof(p) IS DISTINCT FROM 'object' OR jsonb_typeof(cfg) IS DISTINCT FROM 'object'
       OR jsonb_typeof(sol) IS DISTINCT FROM 'object'
       OR (SELECT count(*) FROM jsonb_object_keys(a))<>4 OR (a-ARRAY['actor_ref','perfil_ref','centro_ref','puesto_ref'])<>'{}'::jsonb
       OR (SELECT count(*) FROM jsonb_object_keys(cfg))<>4 OR (cfg-ARRAY['referencia','version','solicitante','ratificador'])<>'{}'::jsonb
       OR jsonb_typeof(cfg->'solicitante') IS DISTINCT FROM 'object'
       OR jsonb_typeof(cfg->'ratificador') IS DISTINCT FROM 'object'
       OR (SELECT count(*) FROM jsonb_object_keys(cfg->'solicitante'))<>4
       OR (SELECT count(*) FROM jsonb_object_keys(cfg->'ratificador'))<>4
       OR ((cfg->'solicitante')-ARRAY['actor_ref','perfil_ref','centro_ref','puesto_ref'])<>'{}'::jsonb
       OR ((cfg->'ratificador')-ARRAY['actor_ref','perfil_ref','centro_ref','puesto_ref'])<>'{}'::jsonb
       OR EXISTS (SELECT 1 FROM jsonb_each(a) e WHERE jsonb_typeof(e.value) IS DISTINCT FROM 'string')
       OR EXISTS (SELECT 1 FROM jsonb_each(cfg->'solicitante') e WHERE jsonb_typeof(e.value) IS DISTINCT FROM 'string')
       OR EXISTS (SELECT 1 FROM jsonb_each(cfg->'ratificador') e WHERE jsonb_typeof(e.value) IS DISTINCT FROM 'string')
       OR (SELECT count(*) FROM jsonb_object_keys(sol)) NOT IN (9,10,12,13)
       OR (sol-ARRAY['centro_ref','contacto_ref','categoria_ref','grupo_subgrupo','motivo_clave','detalle','periodo','rc','documentos_adjuntos','observaciones','jornada_minutos','numero_personas','puesto_solicitado'])<>'{}'::jsonb
       OR jsonb_typeof(sol->'periodo') IS DISTINCT FROM 'object' OR jsonb_typeof(sol->'rc') IS DISTINCT FROM 'object'
       OR NOT vec_contratacion_temporal.periodo_previsto_estructural_v1(sol->'periodo')
       OR vec_contratacion_temporal.datos_puesto_peticion_validos_ct200(sol) IS DISTINCT FROM true
       OR NOT (sol ?& ARRAY['centro_ref','contacto_ref','categoria_ref','grupo_subgrupo','motivo_clave','detalle','periodo','rc','documentos_adjuntos'])
       OR jsonb_typeof(docs) IS DISTINCT FROM 'array'
       OR coalesce(jsonb_array_length(docs),65)>64
       OR EXISTS (SELECT 1 FROM jsonb_array_elements_text(docs) AS elementos(valor)
                  WHERE elementos.valor !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$')
       OR (SELECT count(*) FROM jsonb_array_elements_text(docs))<>
          (SELECT count(DISTINCT elementos.valor) FROM jsonb_array_elements_text(docs) AS elementos(valor))
       OR EXISTS (SELECT 1 FROM jsonb_each_text(a) e WHERE e.value !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$')
       OR EXISTS (SELECT 1 FROM jsonb_each_text(cfg->'solicitante') e WHERE e.value !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$')
       OR EXISTS (SELECT 1 FROM jsonb_each_text(cfg->'ratificador') e WHERE e.value !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$')
       OR a->>'actor_ref' IS NULL OR a->>'perfil_ref' IS NULL OR a->>'centro_ref' IS NULL OR a->>'puesto_ref' IS NULL
       OR cfg->>'referencia' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' OR v_cfg_version IS NULL OR v_cfg_version<1
       OR cfg->'solicitante'->>'actor_ref'=cfg->'ratificador'->>'actor_ref'
       OR cfg->'solicitante'->>'centro_ref' IS DISTINCT FROM cfg->'ratificador'->>'centro_ref'
       OR sol->>'centro_ref' IS DISTINCT FROM a->>'centro_ref'
       OR a->>'centro_ref' IS DISTINCT FROM cfg->'solicitante'->>'centro_ref'
       OR a->>'centro_ref' IS DISTINCT FROM cfg->'ratificador'->>'centro_ref'
       OR v_ref IS NULL OR v_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR v_creada IS NULL OR p->>'creada_en' !~ 'Z$' OR v_creada<>date_trunc('microseconds',v_creada) THEN
        RAISE EXCEPTION 'petición de centro incompatible' USING ERRCODE='P0670';
    END IF;
    IF v_operacion='presentar' THEN
        v_accion:='contratacion_temporal.peticion_centro.presentar'; v_version:=1; v_estado:='pendiente_ratificacion';
        IF (SELECT count(*) FROM jsonb_object_keys(c))<>3 OR (c-ARRAY['operacion','clave_idempotencia','solicitud'])<>'{}'::jsonb OR NOT c ? 'solicitud'
           OR c->'solicitud' IS DISTINCT FROM sol OR v_ref IS DISTINCT FROM 'peticion:centro:'||v_clave_text
           OR p->>'version' IS DISTINCT FROM '1' OR p->>'estado' IS DISTINCT FROM v_estado
           OR (SELECT count(*) FROM jsonb_object_keys(p))<>6 OR (p-ARRAY['referencia','version','configuracion','solicitud','estado','creada_en'])<>'{}'::jsonb
           OR a IS DISTINCT FROM cfg->'solicitante' THEN
            RAISE EXCEPTION 'presentación de centro incompatible' USING ERRCODE='P0670';
        END IF;
    ELSIF v_operacion='ratificar' THEN
        v_accion:='contratacion_temporal.peticion_centro.ratificar'; v_version:=2; v_estado:='ratificada';
        IF (SELECT count(*) FROM jsonb_object_keys(c))<>5 OR (c-ARRAY['operacion','clave_idempotencia','peticion_ref','version_esperada','motivo'])<>'{}'::jsonb
           OR c->>'peticion_ref' IS DISTINCT FROM v_ref OR c->>'version_esperada' IS DISTINCT FROM '1'
           OR p->>'version' IS DISTINCT FROM '2' OR p->>'estado' IS DISTINCT FROM v_estado
           OR p->>'motivo_ratificacion' IS DISTINCT FROM c->>'motivo' OR coalesce(octet_length(c->>'motivo'),0) NOT BETWEEN 1 AND 1000
           OR btrim(c->>'motivo') IS DISTINCT FROM c->>'motivo' OR c->>'motivo' ~ '[[:cntrl:]]'
           OR v_ratificada IS NULL OR p->>'ratificada_en' !~ 'Z$'
           OR v_ratificada<>date_trunc('microseconds',v_ratificada) OR v_ratificada<v_creada
           OR (SELECT count(*) FROM jsonb_object_keys(p))<>8 OR (p-ARRAY['referencia','version','configuracion','solicitud','estado','creada_en','ratificada_en','motivo_ratificacion'])<>'{}'::jsonb
           OR a IS DISTINCT FROM cfg->'ratificador' THEN
            RAISE EXCEPTION 'ratificación de centro incompatible' USING ERRCODE='P0670';
        END IF;
    ELSE
        RAISE EXCEPTION 'operación de petición de centro inválida' USING ERRCODE='P0670';
    END IF;
    EXCEPTION WHEN data_exception THEN
        RAISE EXCEPTION 'estructura de petición de centro inválida' USING ERRCODE='P0670';
    END;
    v_material_huella:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
    v_contexto:='{"ambitos":{"centro_ref":'||to_jsonb(a->>'centro_ref')::text||',"organizacion_ref":"organizacion:desarrollo:dipgra"},"atributos":{"material_sha256":"'||v_material_huella||'"}}';
    v_contexto_huella:=encode(sha256(convert_to(v_contexto,'UTF8')),'hex');
    BEGIN d:=convert_from(p_decision,'UTF8')::jsonb;
    EXCEPTION WHEN data_exception THEN RAISE EXCEPTION 'decisión de petición inválida' USING ERRCODE='P0673'; END;
    IF d->>'accion' IS DISTINCT FROM v_accion OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
       OR d->>'tipo_recurso' IS DISTINCT FROM 'peticion_centro' OR d->>'finalidad' IS DISTINCT FROM 'gestionar_peticion_centro'
       OR d->>'recurso_ref' IS DISTINCT FROM v_ref OR d->>'principal_id' IS DISTINCT FROM a->>'actor_ref'
       OR d->>'perfil_activo_ref' IS DISTINCT FROM a->>'perfil_ref'
       OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_contexto_huella THEN
        RAISE EXCEPTION 'autorización de petición divergente' USING ERRCODE='P0673';
    END IF;
    SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_peticion_centro_v3_atestada(
        p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
    IF v_consumo.consumo_nuevo IS NOT TRUE OR v_consumo.efecto_ref IS DISTINCT FROM v_ref
       OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM v_contexto_huella THEN
        RAISE EXCEPTION 'consumo de petición divergente' USING ERRCODE='P0673';
    END IF;
    PERFORM pg_advisory_xact_lock(hashtextextended('vec:peticion-centro:clave:'||v_clave_text,0));
    PERFORM pg_advisory_xact_lock(hashtextextended('vec:peticion-centro:ref:'||v_ref,0));
    SELECT * INTO v_previa FROM vec_contratacion_temporal.peticion_centro_revision WHERE clave_idempotencia=v_clave;
    IF FOUND THEN
        IF v_previa.actor_ref IS DISTINCT FROM a->>'actor_ref' OR v_previa.perfil_ref IS DISTINCT FROM a->>'perfil_ref'
           OR v_previa.centro_ref IS DISTINCT FROM a->>'centro_ref' OR v_previa.puesto_ref IS DISTINCT FROM a->>'puesto_ref' THEN
            RAISE EXCEPTION 'operación de petición ajena' USING ERRCODE='P0673';
        ELSIF v_previa.operacion IS DISTINCT FROM v_operacion OR v_previa.comando IS DISTINCT FROM c THEN
            RAISE EXCEPTION 'clave usada por otro comando' USING ERRCODE='P0671';
        ELSIF v_previa.material IS DISTINCT FROM p_material THEN
            RAISE EXCEPTION 'material concurrente divergente; reintentar con la misma clave' USING ERRCODE='P0674';
        END IF;
        RETURN v_previa.recibo_json||jsonb_build_object('estado_local','replay_confirmado');
    END IF;
    -- Las revisiones antiguas conservan su replay exacto. Una presentación
    -- nueva ya debe llevar los tres datos, también si entra por SQL.
    IF v_operacion='presentar' AND NOT (sol ?& ARRAY['jornada_minutos','numero_personas','puesto_solicitado']) THEN
        RAISE EXCEPTION 'datos de puesto de la petición ausentes' USING ERRCODE='P0670';
    END IF;
    SELECT * INTO v_actual FROM vec_contratacion_temporal.peticion_centro_revision
     WHERE peticion_ref=v_ref ORDER BY version DESC LIMIT 1 FOR UPDATE;
    IF v_operacion='presentar' THEN
        IF FOUND THEN RAISE EXCEPTION 'referencia de petición ya usada' USING ERRCODE='P0671'; END IF;
    ELSE
        IF NOT FOUND OR v_actual.version<>1 OR v_actual.estado<>'pendiente_ratificacion' THEN
            RAISE EXCEPTION 'versión de petición en conflicto' USING ERRCODE='P0672';
        END IF;
        IF (p-ARRAY['version','estado','ratificada_en','motivo_ratificacion']) IS DISTINCT FROM
           (v_actual.peticion-ARRAY['version','estado','ratificada_en','motivo_ratificacion'])
           OR a->>'actor_ref' IS NOT DISTINCT FROM v_actual.actor_ref THEN
            RAISE EXCEPTION 'ratificación no conserva la petición' USING ERRCODE='P0673';
        END IF;
    END IF;
    v_fecha:=date_trunc('microseconds',clock_timestamp());
    IF v_fecha<v_creada OR (v_ratificada IS NOT NULL AND v_fecha<v_ratificada) THEN
        RAISE EXCEPTION 'cronología de petición inválida' USING ERRCODE='P0670';
    END IF;
    v_recibo_ref:='recibo:peticion-centro:'||gen_random_uuid()::text;
    v_recibo:=jsonb_build_object('recibo_ref',v_recibo_ref,'peticion_ref',v_ref,'version',v_version,
        'estado',v_estado,'actor_ref',a->>'actor_ref','registrado_en',v_fecha,'estado_local','registrado');
    INSERT INTO vec_contratacion_temporal.peticion_centro_revision(
        peticion_ref,version,clave_idempotencia,operacion,material,material_sha256,actor_ref,perfil_ref,centro_ref,puesto_ref,
        configuracion_ref,configuracion_version,comando,peticion,estado,recibo_ref,recibo_json,auditoria_ref,decision_ref,consumo_huella_sha256,registrada_en)
    VALUES(v_ref,v_version,v_clave,v_operacion,p_material,v_material_huella,a->>'actor_ref',a->>'perfil_ref',a->>'centro_ref',a->>'puesto_ref',
        cfg->>'referencia',v_cfg_version,c,p,v_estado,v_recibo_ref,v_recibo,v_consumo.auditoria_ref,v_consumo.decision_ref,v_consumo.consumo_huella_sha256,v_fecha);
    INSERT INTO vec_contratacion_temporal.peticion_centro_outbox(evento_ref,peticion_ref,version,tipo,carga_json,creada_en)
    VALUES('evento:peticion-centro:'||gen_random_uuid()::text,v_ref,v_version,
        CASE v_operacion WHEN 'presentar' THEN 'contratacion_temporal.peticion_centro.presentada' ELSE 'contratacion_temporal.peticion_centro.ratificada' END,
        jsonb_build_object('recibo_ref',v_recibo_ref,'peticion_ref',v_ref,'version',v_version,'estado',v_estado),v_fecha);
    RETURN v_recibo;
END
$function$;

CREATE OR REPLACE FUNCTION vec_contratacion_temporal.confirmar_alta_atestada_v1(p_capacidad_canonica bytea, p_decision_canonica bytea, p_motivo_canonico bytea, p_contexto_actor_canonico bytea, p_persona_version numeric, p_perfil_version numeric, p_payload_vec_ad_3 bytea, p_sobre_cose_sign1 bytea, p_evidencia_verificacion bytea, p_raiz_publica_spki bytea, p_alta_canonica bytea, p_sellos_hmac_canonicos bytea)
 RETURNS TABLE(expediente_ref text, numero_visible text, version numeric, recibo_ref text, auditoria_ref text, evento_ref text, confirmada_en timestamp with time zone, recibo_huella_sha256 text)
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET lock_timeout TO '2s'
AS $function$
DECLARE
    a jsonb;
    s jsonb;
    d jsonb;
    v_consumo record;
    v_identidad record;
    v_par jsonb;
    v_pares jsonb;
    v_generaciones integer[];
    v_generaciones_politica integer[];
    v_aliases text[];
    v_raices text[];
    v_raiz text;
    v_activo_ambito text;
    v_activo_huella text;
    v_huella_alta text;
    v_huella_contexto_recurso text;
    v_contexto_recurso bytea;
    v_ahora timestamptz(6);
    v_revision bigint;
    v_auditoria_ref text;
    v_evento_ref text;
    v_anterior_auditoria text;
    v_anterior_outbox text;
    v_secuencia_auditoria numeric(20, 0);
    v_secuencia_outbox numeric(20, 0);
    v_huella_auditoria text;
    v_huella_outbox text;
    v_payload_outbox bytea;
    v_huella_payload text;
    v_confirmacion_ref text;
    v_huella_actuacion text;
    v_huella_agregado text;
    v_recibo_huella text;
    v_statement numeric;
    v_idle numeric;
BEGIN
    IF pg_catalog.current_setting('transaction_isolation') <> 'serializable'
       OR pg_catalog.current_setting('transaction_read_only') <> 'off'
       OR pg_catalog.current_setting('TimeZone') <> 'UTC'
       OR session_user = current_user
       OR NOT pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER')
       OR pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_migrador', 'MEMBER')
       OR pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_propietario', 'MEMBER') THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'confirmación de alta rechazada';
    END IF;
    SELECT setting::numeric INTO v_statement
      FROM pg_catalog.pg_settings
     WHERE name = 'statement_timeout' AND unit = 'ms';
    SELECT setting::numeric INTO v_idle
      FROM pg_catalog.pg_settings
     WHERE name = 'idle_in_transaction_session_timeout' AND unit = 'ms';
    IF v_statement IS NULL OR v_statement NOT BETWEEN 1 AND 15000
       OR v_idle IS NULL OR v_idle NOT BETWEEN 1 AND 20000
       OR pg_catalog.octet_length(p_alta_canonica) NOT BETWEEN 256 AND 32768
       OR pg_catalog.octet_length(p_sellos_hmac_canonicos)
          NOT BETWEEN 256 AND 8192 THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'entrada de alta inválida';
    END IF;
    BEGIN
        a := pg_catalog.convert_from(p_alta_canonica, 'UTF8')::jsonb;
        s := pg_catalog.convert_from(p_sellos_hmac_canonicos, 'UTF8')::jsonb;
        d := pg_catalog.convert_from(p_decision_canonica, 'UTF8')::jsonb;
    EXCEPTION
        WHEN data_exception OR invalid_text_representation
          OR character_not_in_repertoire OR untranslatable_character THEN
            RAISE EXCEPTION USING
                ERRCODE = '22023',
                MESSAGE = 'entrada de alta inválida';
    END;
    IF pg_catalog.jsonb_typeof(a) <> 'object'
       OR (SELECT pg_catalog.count(*)
             FROM pg_catalog.jsonb_object_keys(a)) <> 16
       OR NOT (a ?& ARRAY[
           'esquema', 'reserva_ref', 'expediente_ref', 'numero_visible',
           'recibo_ref', 'organizacion_ref', 'actor_ref', 'perfil_ref',
           'version', 'flujo', 'fase_actual', 'estado_actual',
           'solicitud', 'creado_en', 'actualizado_en', 'actuacion'
       ])
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.jsonb_object_keys(a) AS k(clave)
            WHERE CASE WHEN k.clave IN (
                'version', 'flujo', 'solicitud', 'actuacion') THEN false
            ELSE pg_catalog.jsonb_typeof(a -> k.clave) <> 'string'
            END)
       OR pg_catalog.jsonb_typeof(a -> 'version') <> 'number'
       OR pg_catalog.jsonb_typeof(a -> 'flujo') <> 'object'
       OR (SELECT pg_catalog.count(*)
             FROM pg_catalog.jsonb_object_keys(a -> 'flujo')) <> 3
       OR NOT ((a -> 'flujo') ?& ARRAY[
           'definicion_ref', 'version', 'huella_sha256'
       ])
       OR pg_catalog.jsonb_typeof(a #> '{flujo,definicion_ref}')
          <> 'string'
       OR pg_catalog.jsonb_typeof(a #> '{flujo,version}') <> 'number'
       OR pg_catalog.jsonb_typeof(a #> '{flujo,huella_sha256}')
          <> 'string'
       OR pg_catalog.jsonb_typeof(a -> 'solicitud') <> 'object'
       OR (SELECT pg_catalog.count(*)
             FROM pg_catalog.jsonb_object_keys(a -> 'solicitud')) <>
       (CASE WHEN a ->> 'esquema' = 'vec.contratacion-temporal.efecto-alta.v3'
             THEN 11 ELSE 10 END) +
       (CASE WHEN (a -> 'solicitud') ? 'puesto_solicitado' THEN 3 ELSE 0 END)
       OR NOT ((a -> 'solicitud') ?& ARRAY[
           'centro_ref', 'contacto_ref', 'categoria_ref',
           'grupo_subgrupo', 'motivo_clave', 'detalle', 'periodo', 'rc',
           'documentos_adjuntos', 'observaciones'
       ])
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.jsonb_object_keys(a -> 'solicitud') AS k(clave)
            WHERE CASE WHEN k.clave IN (
                'periodo', 'rc', 'documentos_adjuntos', 'necesidad',
                'jornada_minutos', 'numero_personas') THEN false
            ELSE pg_catalog.jsonb_typeof(
                (a -> 'solicitud') -> k.clave) <> 'string'
            END)
       OR vec_contratacion_temporal.datos_puesto_peticion_validos_ct200(a -> 'solicitud') IS DISTINCT FROM true
       OR pg_catalog.jsonb_typeof(a #> '{solicitud,periodo}') <> 'object'
       OR NOT vec_contratacion_temporal.periodo_previsto_estructural_v1(a #> '{solicitud,periodo}')
       OR pg_catalog.jsonb_typeof(a #> '{solicitud,rc}') <> 'object'
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(
               a #> '{solicitud,rc}')) <> 5
       OR NOT ((a #> '{solicitud,rc}') ?& ARRAY[
           'existe', 'numero', 'fecha', 'importe', 'documento_ref'
       ])
       OR pg_catalog.jsonb_typeof(a #> '{solicitud,rc,existe}')
          <> 'boolean'
       OR pg_catalog.jsonb_typeof(a #> '{solicitud,rc,numero}')
          <> 'string'
       OR pg_catalog.jsonb_typeof(a #> '{solicitud,rc,fecha}')
          <> 'string'
       OR pg_catalog.jsonb_typeof(a #> '{solicitud,rc,documento_ref}')
          <> 'string'
       OR pg_catalog.jsonb_typeof(a #> '{solicitud,rc,importe}')
          <> 'object'
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(
               a #> '{solicitud,rc,importe}')) <> 2
       OR NOT ((a #> '{solicitud,rc,importe}') ?&
               ARRAY['centimos', 'moneda'])
       OR pg_catalog.jsonb_typeof(
           a #> '{solicitud,rc,importe,centimos}') <> 'number'
       OR pg_catalog.jsonb_typeof(a #> '{solicitud,rc,importe,moneda}')
          <> 'string'
       OR pg_catalog.jsonb_typeof(
           a #> '{solicitud,documentos_adjuntos}') <> 'array'
       OR pg_catalog.jsonb_array_length(
           a #> '{solicitud,documentos_adjuntos}') > 100
       OR EXISTS (
           SELECT 1 FROM pg_catalog.jsonb_array_elements(
               a #> '{solicitud,documentos_adjuntos}') AS e(valor)
           WHERE pg_catalog.jsonb_typeof(e.valor) <> 'string'
              OR e.valor #>> '{}' !~
                 '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$')
       OR pg_catalog.jsonb_typeof(a -> 'actuacion') <> 'object'
       OR (SELECT pg_catalog.count(*)
             FROM pg_catalog.jsonb_object_keys(a -> 'actuacion')) <> 13
       OR NOT ((a -> 'actuacion') ?& ARRAY[
           'secuencia', 'version_expediente', 'accion_clave', 'actor_ref',
           'unidad_ref', 'recibo_ref', 'realizada_en', 'fase_origen',
           'fase_destino', 'estado_origen', 'estado_destino',
           'observaciones', 'documentos_ref'
       ])
       OR pg_catalog.jsonb_typeof(a #> '{actuacion,secuencia}')
          <> 'number'
       OR pg_catalog.jsonb_typeof(a #> '{actuacion,version_expediente}')
          <> 'number'
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.jsonb_object_keys(a -> 'actuacion') AS k(clave)
            WHERE CASE WHEN k.clave IN (
                'secuencia', 'version_expediente', 'documentos_ref') THEN false
            ELSE pg_catalog.jsonb_typeof(
                (a -> 'actuacion') -> k.clave) <> 'string'
            END)
       OR pg_catalog.jsonb_typeof(a #> '{actuacion,documentos_ref}')
          <> 'array'
       OR pg_catalog.jsonb_array_length(
           a #> '{actuacion,documentos_ref}') > 100
       OR EXISTS (
           SELECT 1 FROM pg_catalog.jsonb_array_elements(
               a #> '{actuacion,documentos_ref}') AS e(valor)
           WHERE pg_catalog.jsonb_typeof(e.valor) <> 'string'
              OR e.valor #>> '{}' !~
                 '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$')
       OR vec_contratacion_temporal.reconstruir_efecto_segun_esquema_v3(a)
          IS DISTINCT FROM p_alta_canonica
       OR a ->> 'esquema' NOT IN (
          'vec.contratacion-temporal.efecto-alta.v2',
          'vec.contratacion-temporal.efecto-alta.v3')
       OR (a ->> 'esquema' = 'vec.contratacion-temporal.efecto-alta.v3'
           AND vec_contratacion_temporal.necesidad_alta_valida_v3(a -> 'solicitud') IS NOT TRUE)
       OR (a ->> 'version') !~ '^[1-9][0-9]{0,15}$'
       OR (a ->> 'version')::numeric <> 1
       OR (a #>> '{flujo,version}') !~ '^[1-9][0-9]{0,15}$'
       OR (a #>> '{flujo,version}')::numeric >
          9007199254740991::numeric
       OR (a #>> '{actuacion,secuencia}') !~ '^[1-9][0-9]{0,15}$'
       OR (a #>> '{actuacion,secuencia}')::numeric <> 1
       OR (a #>> '{actuacion,version_expediente}')
          !~ '^[1-9][0-9]{0,15}$'
       OR (a #>> '{actuacion,version_expediente}')::numeric <> 1
       OR (a #>> '{solicitud,rc,importe,centimos}')
          !~ '^(0|[1-9][0-9]{0,15})$'
       OR (a #>> '{solicitud,rc,importe,centimos}')::numeric >
          9007199254740991::numeric
       OR a ->> 'estado_actual' <> 'en_curso'
       OR a ->> 'numero_visible' !~
          '^[0-9]{4}/[A-Za-z0-9._-]{1,40}$'
       OR EXISTS (
           SELECT 1 FROM pg_catalog.unnest(ARRAY[
               a ->> 'reserva_ref', a ->> 'expediente_ref',
               a ->> 'recibo_ref', a ->> 'organizacion_ref',
               a #>> '{solicitud,centro_ref}',
               a #>> '{solicitud,categoria_ref}',
               a ->> 'actor_ref', a ->> 'perfil_ref',
               a #>> '{flujo,definicion_ref}', a ->> 'fase_actual',
               a #>> '{actuacion,accion_clave}',
               a #>> '{actuacion,unidad_ref}'
           ]) AS r(valor)
           WHERE r.valor !~
             '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$')
       OR a #>> '{flujo,huella_sha256}' !~ '^[0-9a-f]{64}$'
       OR a ->> 'creado_en' !~
          '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}[.][0-9]{6}Z$'
       OR a ->> 'actualizado_en' <> a ->> 'creado_en'
       OR a #>> '{actuacion,realizada_en}' <> a ->> 'creado_en'
       OR a #>> '{actuacion,actor_ref}' <> a ->> 'actor_ref'
       OR a #>> '{actuacion,recibo_ref}' <> a ->> 'recibo_ref'
       OR a #>> '{actuacion,fase_origen}' <> ''
       OR a #>> '{actuacion,fase_destino}' <> a ->> 'fase_actual'
       OR a #>> '{actuacion,estado_origen}' <> 'pendiente'
       OR a #>> '{actuacion,estado_destino}' <> a ->> 'estado_actual'
       OR a ->> 'actor_ref' <> d ->> 'principal_id'
       OR a ->> 'perfil_ref' <> d ->> 'perfil_activo_ref' THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'proyección de alta inválida';
    END IF;
    IF pg_catalog.jsonb_typeof(s) <> 'object'
       OR (SELECT pg_catalog.count(*)
             FROM pg_catalog.jsonb_object_keys(s)) <> 3
       OR NOT (s ?& ARRAY['esquema', 'activo', 'retenidos'])
       OR s ->> 'esquema' <>
          'vec.contratacion-temporal.sellos-hmac.v1'
       OR pg_catalog.jsonb_typeof(s -> 'activo') <> 'object'
       OR pg_catalog.jsonb_typeof(s -> 'retenidos') <> 'array'
       OR pg_catalog.jsonb_array_length(s -> 'retenidos') > 3
       OR vec_contratacion_temporal.reconstruir_sellos_hmac_v1(s)
          IS DISTINCT FROM p_sellos_hmac_canonicos THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'sellos HMAC inválidos';
    END IF;
    v_pares := pg_catalog.jsonb_build_array(s -> 'activo') ||
               (s -> 'retenidos');
    IF EXISTS (
        SELECT 1
          FROM pg_catalog.jsonb_array_elements(v_pares) AS e(valor)
         WHERE pg_catalog.jsonb_typeof(e.valor) <> 'object'
            OR (SELECT pg_catalog.count(*)
                  FROM pg_catalog.jsonb_object_keys(e.valor)) <> 3
            OR NOT (e.valor ?& ARRAY[
                'generacion', 'ambito_hmac', 'huella_hmac'
            ])
            OR pg_catalog.jsonb_typeof(e.valor -> 'generacion')
               <> 'number'
            OR pg_catalog.jsonb_typeof(e.valor -> 'ambito_hmac')
               <> 'string'
            OR pg_catalog.jsonb_typeof(e.valor -> 'huella_hmac')
               <> 'string'
            OR e.valor ->> 'generacion' !~ '^[1-9][0-9]{0,8}$'
            OR (e.valor ->> 'generacion')::numeric >
               9007199254740991::numeric
            OR e.valor ->> 'ambito_hmac' !~
               (
                 '^hmac-sha256:vec[.]contratacion-temporal[.]'
                 || 'ambito-idempotencia/v[1-9][0-9]{0,8}:[a-f0-9]{64}$')
            OR e.valor ->> 'huella_hmac' !~
               (
                 '^hmac-sha256:vec[.]contratacion-temporal[.]'
                 || 'huella-peticion/v[1-9][0-9]{0,8}:[a-f0-9]{64}$')
            OR pg_catalog.right(e.valor ->> 'ambito_hmac', 64) =
               pg_catalog.repeat('0', 64)
            OR pg_catalog.right(e.valor ->> 'huella_hmac', 64) =
               pg_catalog.repeat('0', 64)
            OR substring(
                 e.valor ->> 'ambito_hmac'
                 FROM '/v([1-9][0-9]{0,8}):')::integer <> (e.valor ->> 'generacion')::integer
            OR substring(
                 e.valor ->> 'huella_hmac'
                 FROM '/v([1-9][0-9]{0,8}):')::integer <> (e.valor ->> 'generacion')::integer) THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'sellos HMAC inválidos';
    END IF;
    SELECT pg_catalog.array_agg(
               (e.valor ->> 'generacion')::integer ORDER BY e.orden),
           pg_catalog.array_agg(
               e.valor ->> 'ambito_hmac' ORDER BY e.orden)
      INTO v_generaciones, v_aliases
      FROM pg_catalog.jsonb_array_elements(v_pares)
           WITH ORDINALITY AS e(valor, orden);
    SELECT pg_catalog.array_agg(
               generacion ORDER BY posicion)
      INTO v_generaciones_politica
      FROM vec_contratacion_temporal.politica_generaciones_hmac_alta;
    IF v_generaciones IS DISTINCT FROM v_generaciones_politica
       OR pg_catalog.cardinality(v_generaciones) <>
          pg_catalog.cardinality(
              ARRAY(SELECT DISTINCT x FROM pg_catalog.unnest(
                  v_generaciones) AS u(x))) THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'política HMAC no satisfecha';
    END IF;
    v_activo_ambito := s #>> '{activo,ambito_hmac}';
    v_activo_huella := s #>> '{activo,huella_hmac}';
    v_huella_alta := pg_catalog.encode(
        pg_catalog.sha256(p_alta_canonica), 'hex');
    -- Cierra la ligadura completa del recurso que originó la decisión V3.
    v_contexto_recurso := pg_catalog.convert_to(
        '{"ambitos":{"categoria_ref":' ||
          vec_contratacion_temporal.texto_json_go_v1(
              a #>> '{solicitud,categoria_ref}') ||
        ',"centro_ref":' ||
          vec_contratacion_temporal.texto_json_go_v1(
              a #>> '{solicitud,centro_ref}') ||
        ',"organizacion_ref":' ||
          vec_contratacion_temporal.texto_json_go_v1(
              a ->> 'organizacion_ref') ||
        '},"atributos":{"efecto_huella_sha256":' ||
          vec_contratacion_temporal.texto_json_go_v1(v_huella_alta) ||
        ',"flujo_huella_sha256":' ||
          vec_contratacion_temporal.texto_json_go_v1(
              a #>> '{flujo,huella_sha256}') ||
        ',"flujo_ref":' ||
          vec_contratacion_temporal.texto_json_go_v1(
              a #>> '{flujo,definicion_ref}') ||
        ',"flujo_version":' ||
          vec_contratacion_temporal.texto_json_go_v1(
              a #>> '{flujo,version}') ||
        ',"huella_peticion_hmac_activa":' ||
          vec_contratacion_temporal.texto_json_go_v1(v_activo_huella) ||
        '}}',
        'UTF8');
    v_huella_contexto_recurso := pg_catalog.encode(
        pg_catalog.sha256(v_contexto_recurso), 'hex');
    IF d ->> 'recurso_ref' <> v_activo_ambito
       OR d ->> 'modulo_id' <> 'contratacion_temporal'
       OR d ->> 'tipo_recurso' <> 'expediente_contratacion_temporal'
       OR d ->> 'accion' <> 'contratacion_temporal.solicitud.crear'
       OR d ->> 'finalidad' <> 'tramitar_necesidad_personal_temporal'
       OR d ->> 'contexto_recurso_huella_sha256' <>
          v_huella_contexto_recurso THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'efecto de alta no autorizado';
    END IF;
    SELECT * INTO STRICT v_consumo
      FROM vec_autorizacion_atestada_v3.
           registrar_y_consumir_decision_v3_atestada(
               p_capacidad_canonica, p_decision_canonica,
               p_motivo_canonico, p_contexto_actor_canonico,
               p_persona_version, p_perfil_version,
               p_payload_vec_ad_3, p_sobre_cose_sign1,
               p_evidencia_verificacion, p_raiz_publica_spki);
    IF v_consumo.efecto_ref <> v_activo_ambito
       OR v_consumo.huella_efecto_sha256 <>
          v_huella_contexto_recurso
       OR v_consumo.consumo_nuevo IS NULL THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'consumo de alta incoherente';
    END IF;
    v_confirmacion_ref := 'cnf_ct_' || pg_catalog.substr(
        pg_catalog.encode(pg_catalog.sha256(
            vec_contratacion_temporal.encuadrar_texto_v1(
                v_consumo.decision_ref
            ) ||
            vec_contratacion_temporal.encuadrar_texto_v1(
                a ->> 'expediente_ref'
            ) ||
            vec_contratacion_temporal.encuadrar_texto_v1(
                v_consumo.consumo_huella_sha256
            )
        ), 'hex'), 1, 32
    );
    IF v_consumo.consumo_nuevo IS FALSE THEN
        RETURN QUERY
        SELECT *
          FROM vec_contratacion_temporal.reconciliar_agregado_alta_v1(
              p_alta_canonica, v_activo_ambito, v_activo_huella,
              v_consumo.decision_ref, v_consumo.efecto_ref,
              v_consumo.huella_efecto_sha256,
              v_consumo.consumo_huella_sha256
          );
        RETURN;
    END IF;
    -- Orden total de locks de alias para que todas las sesiones converjan.
    PERFORM pg_catalog.pg_advisory_xact_lock(
        pg_catalog.hashtextextended('vec_ct:alias:' || alias, 0))
      FROM pg_catalog.unnest(v_aliases) AS u(alias)
     ORDER BY alias COLLATE "C";
    SELECT pg_catalog.array_agg(
               DISTINCT ambito_raiz_hmac ORDER BY ambito_raiz_hmac)
      INTO v_raices
      FROM vec_contratacion_temporal.alias_ambito_alta
     WHERE alias_hmac = ANY (v_aliases);
    IF pg_catalog.cardinality(v_raices) > 1 THEN
        RAISE EXCEPTION USING
            ERRCODE = '23505',
            MESSAGE = 'alias HMAC divergentes';
    END IF;
    IF pg_catalog.cardinality(v_raices) = 1 THEN
        v_raiz := v_raices[1];
    ELSE
        v_raiz := v_activo_ambito;
        INSERT INTO vec_contratacion_temporal.identidad_reserva_alta (
            ambito_hmac, reserva_ref, expediente_ref, numero_visible,
            recibo_ref, huella_peticion_hmac, organizacion_ref,
            actor_ref, perfil_ref, creada_en) VALUES (
            v_raiz, a ->> 'reserva_ref', a ->> 'expediente_ref',
            a ->> 'numero_visible', a ->> 'recibo_ref',
            v_activo_huella, a ->> 'organizacion_ref',
            a ->> 'actor_ref', a ->> 'perfil_ref',
            (a #>> '{actuacion,realizada_en}')::timestamptz);
        INSERT INTO vec_contratacion_temporal.reserva_alta_version (
            ambito_hmac, revision, estado, registrada_en) VALUES (v_raiz, 1, 'reservada', clock_timestamp());
        INSERT INTO vec_contratacion_temporal.reserva_alta_actual
            VALUES (v_raiz, 1);
    END IF;
    SELECT * INTO STRICT v_identidad
      FROM vec_contratacion_temporal.identidad_reserva_alta
     WHERE ambito_hmac = v_raiz;
    IF v_identidad.reserva_ref <> a ->> 'reserva_ref'
       OR v_identidad.expediente_ref <> a ->> 'expediente_ref'
       OR v_identidad.numero_visible <> a ->> 'numero_visible'
       OR v_identidad.recibo_ref <> a ->> 'recibo_ref'
       OR v_identidad.organizacion_ref <> a ->> 'organizacion_ref'
       OR v_identidad.actor_ref <> a ->> 'actor_ref'
       OR v_identidad.perfil_ref <> a ->> 'perfil_ref' THEN
        RAISE EXCEPTION USING
            ERRCODE = '23505',
            MESSAGE = 'reserva de alta en conflicto';
    END IF;
    FOR v_par IN
        SELECT e.valor
          FROM pg_catalog.jsonb_array_elements(v_pares)
               WITH ORDINALITY AS e(valor, orden)
         ORDER BY e.orden
    LOOP
        IF EXISTS (
            SELECT 1
              FROM vec_contratacion_temporal.alias_ambito_alta x
             WHERE x.alias_hmac = v_par ->> 'ambito_hmac'
               AND x.ambito_raiz_hmac <> v_raiz) OR EXISTS (
            SELECT 1
              FROM vec_contratacion_temporal.alias_huella_alta x
             WHERE x.ambito_raiz_hmac = v_raiz
               AND x.generacion =
                   (v_par ->> 'generacion')::integer
               AND x.alias_hmac <> v_par ->> 'huella_hmac') THEN
            RAISE EXCEPTION USING
                ERRCODE = '23505',
                MESSAGE = 'par HMAC en conflicto';
        END IF;
        INSERT INTO vec_contratacion_temporal.alias_ambito_alta (
            alias_hmac, ambito_raiz_hmac, generacion, registrada_en) VALUES (
            v_par ->> 'ambito_hmac', v_raiz,
            (v_par ->> 'generacion')::integer, clock_timestamp()) ON CONFLICT DO NOTHING;
        INSERT INTO vec_contratacion_temporal.alias_huella_alta (
            ambito_raiz_hmac, generacion, alias_hmac, registrada_en) VALUES (
            v_raiz, (v_par ->> 'generacion')::integer,
            v_par ->> 'huella_hmac', clock_timestamp()) ON CONFLICT DO NOTHING;
    END LOOP;
    IF EXISTS (
        SELECT 1 FROM vec_contratacion_temporal.expediente_alta e
         WHERE e.expediente_ref = a ->> 'expediente_ref'
    ) OR EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.confirmacion_agregado_alta m
         WHERE m.confirmacion_ref = v_confirmacion_ref
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = 'V2070',
            MESSAGE = 'estado previo de alta incoherente';
    END IF;
    v_ahora := pg_catalog.date_trunc(
        'microseconds', clock_timestamp());
    IF v_ahora < (a #>> '{actuacion,realizada_en}')::timestamptz THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'instante de alta inválido';
    END IF;
    v_auditoria_ref := 'aud_ct_' || pg_catalog.substr(
        pg_catalog.encode(pg_catalog.sha256(
            vec_contratacion_temporal.encuadrar_texto_v1(
                v_consumo.decision_ref) ||
            vec_contratacion_temporal.encuadrar_texto_v1(
                a ->> 'expediente_ref')), 'hex'), 1, 32);
    v_evento_ref := 'evt_ct_' || pg_catalog.substr(
        pg_catalog.encode(pg_catalog.sha256(
            vec_contratacion_temporal.encuadrar_texto_v1(
                a ->> 'expediente_ref') ||
            vec_contratacion_temporal.encuadrar_texto_v1(
                v_consumo.consumo_huella_sha256)), 'hex'), 1, 32);
    INSERT INTO vec_contratacion_temporal.expediente_alta (
        expediente_ref, reserva_ref, numero_visible, organizacion_ref,
        actor_ref, perfil_ref, decision_ref, efecto_ref,
        huella_efecto_sha256, creada_en, confirmacion_ref) VALUES (
        a ->> 'expediente_ref', a ->> 'reserva_ref',
        a ->> 'numero_visible', a ->> 'organizacion_ref',
        a ->> 'actor_ref', a ->> 'perfil_ref',
        v_consumo.decision_ref, v_consumo.efecto_ref,
        v_consumo.huella_efecto_sha256,
        (a ->> 'creado_en')::timestamptz, v_confirmacion_ref);
    INSERT INTO vec_contratacion_temporal.expediente_alta_version (
        expediente_ref, version, alta_canonica, huella_alta_sha256,
        flujo_ref, flujo_version, flujo_huella_sha256, fase_clave,
        estado, solicitud_huella_sha256, registrada_en,
        confirmacion_ref) VALUES (
        a ->> 'expediente_ref', 1, p_alta_canonica, v_huella_alta,
        a #>> '{flujo,definicion_ref}',
        (a #>> '{flujo,version}')::numeric,
        a #>> '{flujo,huella_sha256}', a ->> 'fase_actual',
        a ->> 'estado_actual',
        pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
            vec_contratacion_temporal.reconstruir_solicitud_segun_esquema_v3(
                a -> 'solicitud'), 'UTF8')), 'hex'),
        v_ahora, v_confirmacion_ref);
    v_huella_actuacion := pg_catalog.encode(pg_catalog.sha256(
        vec_contratacion_temporal.encuadrar_texto_v1(
            a ->> 'expediente_ref'
        ) ||
        vec_contratacion_temporal.encuadrar_texto_v1(
            (a #> '{actuacion}')::text
        )
    ), 'hex');
    INSERT INTO vec_contratacion_temporal.actuacion_alta (
        expediente_ref, secuencia, version_expediente, accion_clave,
        actor_ref, unidad_ref, recibo_ref, fase_destino,
        estado_destino, realizada_en, huella_sha256,
        confirmacion_ref) VALUES (
        a ->> 'expediente_ref', 1, 1,
        a #>> '{actuacion,accion_clave}',
        a #>> '{actuacion,actor_ref}',
        a #>> '{actuacion,unidad_ref}',
        a #>> '{actuacion,recibo_ref}',
        a #>> '{actuacion,fase_destino}',
        a #>> '{actuacion,estado_destino}',
        (a #>> '{actuacion,realizada_en}')::timestamptz,
        v_huella_actuacion, v_confirmacion_ref);
    SELECT secuencia_auditoria, cabeza_auditoria_sha256,
           secuencia_outbox, cabeza_outbox_sha256
      INTO STRICT v_secuencia_auditoria, v_anterior_auditoria,
                  v_secuencia_outbox, v_anterior_outbox
     FROM vec_contratacion_temporal.control_cadenas_alta
     WHERE control_id
     FOR UPDATE;
    IF v_secuencia_auditoria >= 9007199254740991::numeric
       OR v_secuencia_outbox >= 9007199254740991::numeric THEN
        RAISE EXCEPTION USING
            ERRCODE = '22003',
            MESSAGE = 'límite de secuencia alcanzado';
    END IF;
    v_secuencia_auditoria := v_secuencia_auditoria + 1;
    v_secuencia_outbox := v_secuencia_outbox + 1;
    v_huella_auditoria := pg_catalog.encode(pg_catalog.sha256(
        vec_contratacion_temporal.encuadrar_texto_v1(
            v_secuencia_auditoria::text) ||
        vec_contratacion_temporal.encuadrar_texto_v1(
            v_anterior_auditoria) ||
        vec_contratacion_temporal.encuadrar_texto_v1(v_auditoria_ref) ||
        vec_contratacion_temporal.encuadrar_texto_v1(
            a ->> 'expediente_ref') ||
        vec_contratacion_temporal.encuadrar_texto_v1(
            v_consumo.decision_ref) ||
        vec_contratacion_temporal.encuadrar_texto_v1(
            v_consumo.consumo_huella_sha256) ||
        vec_contratacion_temporal.encuadrar_texto_v1(v_huella_alta)), 'hex');
    INSERT INTO vec_contratacion_temporal.auditoria_alta (
        auditoria_ref, secuencia, expediente_ref, decision_ref,
        consumo_huella_sha256, anterior_sha256, huella_sha256,
        registrada_en, confirmacion_ref) VALUES (
        v_auditoria_ref, v_secuencia_auditoria,
        a ->> 'expediente_ref', v_consumo.decision_ref,
        v_consumo.consumo_huella_sha256, v_anterior_auditoria,
        v_huella_auditoria, v_ahora, v_confirmacion_ref);
    v_payload_outbox := pg_catalog.convert_to(
        '{"esquema":"vec.contratacion-temporal.evento-expediente-registrado.v1"' ||
        ',"evento_ref":' ||
          vec_contratacion_temporal.texto_json_go_v1(v_evento_ref) ||
        ',"expediente_ref":' ||
          vec_contratacion_temporal.texto_json_go_v1(
              a ->> 'expediente_ref') ||
        ',"version":1,"ocurrido_en":' ||
          vec_contratacion_temporal.texto_json_go_v1(
              vec_contratacion_temporal.instante_utc_v1(v_ahora)) || '}',
        'UTF8');
    v_huella_payload := pg_catalog.encode(
        pg_catalog.sha256(v_payload_outbox), 'hex');
    v_huella_outbox := pg_catalog.encode(pg_catalog.sha256(
        vec_contratacion_temporal.encuadrar_texto_v1(
            v_secuencia_outbox::text) ||
        vec_contratacion_temporal.encuadrar_texto_v1(
            v_anterior_outbox) ||
        vec_contratacion_temporal.encuadrar_texto_v1(v_evento_ref) ||
        vec_contratacion_temporal.encuadrar_texto_v1(v_huella_payload)), 'hex');
    INSERT INTO vec_contratacion_temporal.outbox_alta (
        evento_ref, secuencia, expediente_ref, tipo_evento,
        payload_canonico, payload_huella_sha256, anterior_sha256,
        huella_sha256, registrada_en, confirmacion_ref) VALUES (
        v_evento_ref, v_secuencia_outbox, a ->> 'expediente_ref',
        'contratacion_temporal.expediente.registrado.v1',
        v_payload_outbox, v_huella_payload, v_anterior_outbox,
        v_huella_outbox, v_ahora, v_confirmacion_ref);
    UPDATE vec_contratacion_temporal.control_cadenas_alta
       SET secuencia_auditoria = v_secuencia_auditoria,
           cabeza_auditoria_sha256 = v_huella_auditoria,
           secuencia_outbox = v_secuencia_outbox,
           cabeza_outbox_sha256 = v_huella_outbox,
           actualizada_en = v_ahora
     WHERE control_id;
    SELECT revision INTO STRICT v_revision
      FROM vec_contratacion_temporal.reserva_alta_actual
     WHERE ambito_hmac = v_raiz
     FOR UPDATE;
    v_revision := v_revision + 1;
    INSERT INTO vec_contratacion_temporal.reserva_alta_version (
        ambito_hmac, revision, estado, version_expediente,
        auditoria_ref, evento_ref, confirmada_en, registrada_en,
        confirmacion_ref) VALUES (
        v_raiz, v_revision, 'confirmada', 1,
        v_auditoria_ref, v_evento_ref, v_ahora, v_ahora,
        v_confirmacion_ref);
    UPDATE vec_contratacion_temporal.reserva_alta_actual
       SET revision = v_revision
     WHERE ambito_hmac = v_raiz;
    v_recibo_huella := pg_catalog.encode(pg_catalog.sha256(
        vec_contratacion_temporal.encuadrar_texto_v1(
            a ->> 'expediente_ref') ||
        vec_contratacion_temporal.encuadrar_texto_v1(
            a ->> 'numero_visible') ||
        vec_contratacion_temporal.encuadrar_texto_v1('1') ||
        vec_contratacion_temporal.encuadrar_texto_v1(a ->> 'recibo_ref') ||
        vec_contratacion_temporal.encuadrar_texto_v1(v_auditoria_ref) ||
        vec_contratacion_temporal.encuadrar_texto_v1(v_evento_ref) ||
        vec_contratacion_temporal.encuadrar_texto_v1(
            vec_contratacion_temporal.instante_utc_v1(v_ahora))), 'hex');
    v_huella_agregado :=
      vec_contratacion_temporal.huella_prueba_agregado_alta_v1(
        VARIADIC ARRAY[
          'vec.contratacion-temporal.confirmacion-agregado-alta.v1',
          v_confirmacion_ref, v_raiz, v_revision::text,
          a ->> 'reserva_ref', a ->> 'expediente_ref',
          a ->> 'numero_visible', a ->> 'recibo_ref',
          v_consumo.decision_ref, v_consumo.efecto_ref,
          v_consumo.huella_efecto_sha256,
          v_consumo.consumo_huella_sha256, '1', v_huella_alta, '1',
          v_huella_actuacion, v_auditoria_ref,
          v_secuencia_auditoria::text, v_anterior_auditoria,
          v_huella_auditoria, v_evento_ref, v_secuencia_outbox::text,
          v_huella_payload, v_anterior_outbox, v_huella_outbox,
          vec_contratacion_temporal.instante_utc_v1(v_ahora),
          v_recibo_huella
        ]
      );
    INSERT INTO vec_contratacion_temporal.confirmacion_agregado_alta (
        confirmacion_ref, agregado_huella_sha256, ambito_hmac,
        reserva_revision, reserva_ref, expediente_ref, numero_visible,
        recibo_ref, decision_ref, efecto_ref, huella_efecto_sha256,
        consumo_huella_sha256, version_expediente, huella_alta_sha256,
        actuacion_secuencia, actuacion_huella_sha256, auditoria_ref,
        auditoria_secuencia, auditoria_anterior_sha256,
        auditoria_huella_sha256, evento_ref, outbox_secuencia,
        payload_huella_sha256, outbox_anterior_sha256,
        outbox_huella_sha256, confirmada_en, recibo_huella_sha256,
        creada_en
    ) VALUES (
        v_confirmacion_ref, v_huella_agregado, v_raiz, v_revision,
        a ->> 'reserva_ref', a ->> 'expediente_ref',
        a ->> 'numero_visible', a ->> 'recibo_ref',
        v_consumo.decision_ref, v_consumo.efecto_ref,
        v_consumo.huella_efecto_sha256,
        v_consumo.consumo_huella_sha256, 1, v_huella_alta, 1,
        v_huella_actuacion, v_auditoria_ref, v_secuencia_auditoria,
        v_anterior_auditoria, v_huella_auditoria, v_evento_ref,
        v_secuencia_outbox, v_huella_payload, v_anterior_outbox,
        v_huella_outbox, v_ahora, v_recibo_huella, v_ahora
    );
    RETURN QUERY
    SELECT *
      FROM vec_contratacion_temporal.reconciliar_agregado_alta_v1(
          p_alta_canonica, v_activo_ambito, v_activo_huella,
          v_consumo.decision_ref, v_consumo.efecto_ref,
          v_consumo.huella_efecto_sha256,
          v_consumo.consumo_huella_sha256
      );
EXCEPTION
    WHEN invalid_text_representation OR datetime_field_overflow
      OR numeric_value_out_of_range THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'entrada de alta inválida';
END
$function$;

REVOKE ALL ON FUNCTION vec_contratacion_temporal.datos_puesto_peticion_validos_ct200(jsonb) FROM PUBLIC, vec_contratacion_temporal_migrador;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.registrar_peticion_centro_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea), vec_contratacion_temporal.confirmar_alta_atestada_v1(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea) FROM PUBLIC, vec_contratacion_temporal_migrador;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.registrar_peticion_centro_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea), vec_contratacion_temporal.confirmar_alta_atestada_v1(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea) TO vec_contratacion_temporal_ejecutor;
COMMIT;
