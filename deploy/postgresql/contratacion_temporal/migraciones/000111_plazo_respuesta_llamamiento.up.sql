\set ON_ERROR_STOP on
-- CT111: plazo de respuesta del llamamiento gobernado por catálogo.
-- 1. Lista versionada de políticas admitidas: sustituye las comprobaciones
--    literales de CT58 (referencia, versión y huella) sin reescribir filas.
-- 2. Eventos de plazo registrados por RRHH: contacto efectivo (abre el plazo;
--    el aviso por correo nunca lo abre) y causa justificada acreditada. El
--    vencimiento lo calcula el servidor con las reglas del catálogo; aquí se
--    comprueba su forma, su política admitida y su coherencia temporal.
-- 3. La resolución manual admite «expirado»: RRHH confirma la propuesta de no
--    aceptación y siguiente candidato solo tras el vencimiento y sin respuesta.
--    Una respuesta fuera de plazo se trata según la regla capturada al abrir el
--    plazo (admitir, exigir causa justificada acreditada o no admitir).
-- No invoca Bolsa, no ejecuta el siguiente candidato y no acredita entrega.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000111',0));
LOCK TABLE vec_contratacion_temporal.resolucion_manual_respuesta_rrhh IN ACCESS EXCLUSIVE MODE;

DO $dependencias$
BEGIN
    IF to_regclass('vec_contratacion_temporal.politica_llamamiento_admitida') IS NOT NULL
       OR to_regclass('vec_contratacion_temporal.evento_plazo_llamamiento_rrhh') IS NOT NULL THEN
        RAISE EXCEPTION 'CT111 ya instalada: no se reaplica' USING ERRCODE='55000';
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_proc p
         WHERE p.oid=to_regprocedure('vec_contratacion_temporal.registrar_resolucion_manual_respuesta_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')
           AND p.proowner='vec_contratacion_temporal_propietario'::regrole AND p.prosecdef
           AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='3e27f8a9a1dfcec701db5721e7cc679955bbd6bdd921e3cc402e2110ce750274'
    ) OR NOT EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
           WHERE n.nspname='vec_autorizacion_atestada_v3'
             AND p.proname='registrar_y_consumir_resolucion_manual_ct_v3_atestada'
             AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef)
       OR (SELECT count(*) FROM pg_constraint
            WHERE conrelid='vec_contratacion_temporal.resolucion_manual_respuesta_rrhh'::regclass
              AND conname IN ('resolucion_manual_respuesta_rrhh_politica_ref_check',
                  'resolucion_manual_respuesta_rrhh_politica_version_check',
                  'resolucion_manual_respuesta_rrhh_politica_sha256_check',
                  'resolucion_manual_respuesta_rrhh_estado_plazo_check',
                  'resolucion_manual_comando_siguiente_check'))<>5 THEN
        RAISE EXCEPTION 'CT111 requiere la resolución CT64 y el consumidor 17 intactos' USING ERRCODE='55000';
    END IF;
END
$dependencias$;

-- Lista versionada. «referencia_exacta» admite una política concreta (la
-- histórica del ejercicio sintético); «entrada_catalogo» admite cualquier
-- versión publicada de una entrada de catálogo, identificada como
-- catalogo:version:entrada con la huella del catálogo. Retirar una admisión
-- es fijar retirada_en; no requiere cambiar código ni restricciones.
CREATE TABLE vec_contratacion_temporal.politica_llamamiento_admitida (
    admision_ref text PRIMARY KEY CHECK (admision_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
    tipo text NOT NULL CHECK (tipo IN ('referencia_exacta','entrada_catalogo')),
    politica_ref text,
    politica_version numeric(20,0),
    politica_sha256 text,
    catalogo_id text,
    entrada_clave text,
    usos text[] NOT NULL CHECK (cardinality(usos) BETWEEN 1 AND 4
        AND usos <@ ARRAY['plazo','aceptacion','renuncia','expiracion_gobernada']::text[]),
    vigente_desde timestamptz(6) NOT NULL CHECK (isfinite(vigente_desde)),
    retirada_en timestamptz(6) CHECK (retirada_en IS NULL OR retirada_en>vigente_desde),
    CHECK ((tipo='referencia_exacta'
            AND politica_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
            AND politica_version>=1 AND politica_sha256 ~ '^[0-9a-f]{64}$'
            AND catalogo_id IS NULL AND entrada_clave IS NULL)
        OR (tipo='entrada_catalogo'
            AND catalogo_id ~ '^[a-z0-9][a-z0-9._-]{1,79}$' AND entrada_clave ~ '^[a-z0-9][a-z0-9._-]{1,79}$'
            AND politica_ref IS NULL AND politica_version IS NULL AND politica_sha256 IS NULL))
);
COMMENT ON TABLE vec_contratacion_temporal.politica_llamamiento_admitida IS
    'Lista versionada de políticas de plazo y resolución de llamamiento admitidas. Los datos iniciales son la política histórica sintética y las entradas del catálogo de reglas de Bolsa; su contenido funcional vive en el catálogo.';
ALTER TABLE vec_contratacion_temporal.politica_llamamiento_admitida ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contratacion_temporal.politica_llamamiento_admitida FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario ON vec_contratacion_temporal.politica_llamamiento_admitida
    TO vec_contratacion_temporal_propietario USING (true) WITH CHECK (true);
REVOKE ALL ON TABLE vec_contratacion_temporal.politica_llamamiento_admitida
    FROM PUBLIC, vec_contratacion_temporal_ejecutor;
INSERT INTO vec_contratacion_temporal.politica_llamamiento_admitida
    (admision_ref,tipo,politica_ref,politica_version,politica_sha256,catalogo_id,entrada_clave,usos,vigente_desde)
VALUES
    ('admision:historica:revision-manual-sintetica:v1','referencia_exacta',
     'politica:ct:revision-manual-sintetica:20260906',1,
     'ea41d65808044fa75b597855e81a469ed274403a521890bafa07c33ae89ec2e3',NULL,NULL,
     ARRAY['aceptacion','renuncia'],'2026-09-06T00:00:00Z'),
    ('admision:reglas-bolsa:plazo-respuesta','entrada_catalogo',NULL,NULL,NULL,
     'vec.bolsa.reglas','b05.plazo_respuesta',ARRAY['plazo'],'2026-09-25T00:00:00Z'),
    ('admision:reglas-bolsa:fuera-de-plazo','entrada_catalogo',NULL,NULL,NULL,
     'vec.bolsa.reglas','b07.fuera_de_plazo',ARRAY['aceptacion','renuncia'],'2026-09-25T00:00:00Z'),
    ('admision:reglas-bolsa:sin-respuesta','entrada_catalogo',NULL,NULL,NULL,
     'vec.bolsa.reglas','b08.sin_respuesta_baja',ARRAY['expiracion_gobernada'],'2026-09-25T00:00:00Z');

-- Devuelve el tipo de admisión vigente o NULL. Solo la usan las funciones
-- SECURITY DEFINER de este esquema, que se ejecutan como propietario.
CREATE FUNCTION vec_contratacion_temporal.politica_llamamiento_admitida_v1(
    p_ref text,p_version numeric,p_sha256 text,p_uso text
) RETURNS text
LANGUAGE sql STABLE
SET search_path = pg_catalog
AS $funcion$
    SELECT a.tipo
      FROM vec_contratacion_temporal.politica_llamamiento_admitida a
     WHERE p_uso=ANY(a.usos)
       AND a.vigente_desde<=clock_timestamp()
       AND (a.retirada_en IS NULL OR a.retirada_en>clock_timestamp())
       AND p_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       AND p_sha256 ~ '^[0-9a-f]{64}$' AND p_sha256<>repeat('0',64)
       AND p_version>=1 AND p_version<=9007199254740991 AND p_version=trunc(p_version)
       AND ((a.tipo='referencia_exacta' AND a.politica_ref=p_ref
             AND a.politica_version=p_version AND a.politica_sha256=p_sha256)
         OR (a.tipo='entrada_catalogo'
             AND p_ref=a.catalogo_id||':'||p_version::text||':'||a.entrada_clave))
     ORDER BY a.admision_ref
     LIMIT 1
$funcion$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.politica_llamamiento_admitida_v1(text,numeric,text,text)
    FROM PUBLIC, vec_contratacion_temporal_ejecutor;

-- Hechos de plazo inmutables, uno por tipo y comunicación. El vencimiento y
-- las reglas aplicables se capturan al abrir el plazo con el contacto.
CREATE TABLE vec_contratacion_temporal.evento_plazo_llamamiento_rrhh (
    evento_ref text PRIMARY KEY,
    tipo text NOT NULL CHECK (tipo IN ('contacto_efectivo','causa_justificada')),
    organizacion_ref text NOT NULL,
    expediente_ref text NOT NULL,
    llamamiento_ref text NOT NULL,
    comunicacion_ref text NOT NULL REFERENCES vec_contratacion_temporal.comunicacion_llamamiento_local,
    clave_idempotencia uuid NOT NULL,
    actor_ref text NOT NULL,
    perfil_ref text NOT NULL,
    instante_en timestamptz(6) NOT NULL CHECK (isfinite(instante_en) AND instante_en<>'0001-01-01T00:00:00Z'::timestamptz),
    prueba_ref text NOT NULL CHECK (prueba_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
    respuesta_hasta timestamptz(6),
    ultimo_dia date,
    politica_ref text,
    politica_version numeric(20,0),
    politica_sha256 text,
    tratamiento_fuera_plazo text CHECK (tratamiento_fuera_plazo IN ('admitir','exige_causa_justificada','no_admitir')),
    confirmacion_expiracion text CHECK (confirmacion_expiracion='rrhh'),
    material text NOT NULL CHECK (octet_length(material) BETWEEN 1 AND 16384),
    material_json jsonb NOT NULL CHECK (jsonb_typeof(material_json)='object' AND material_json=material::jsonb),
    material_huella_sha256 text NOT NULL CHECK (material_huella_sha256=encode(sha256(convert_to(material,'UTF8')),'hex')),
    solicitud_json jsonb NOT NULL CHECK (jsonb_typeof(solicitud_json)='object' AND solicitud_json=material_json->'Solicitud'),
    auditoria_ref text NOT NULL UNIQUE,
    decision_ref text NOT NULL UNIQUE,
    consumo_huella_sha256 text NOT NULL UNIQUE CHECK (consumo_huella_sha256 ~ '^[0-9a-f]{64}$'),
    evidencia_huella_sha256 text NOT NULL CHECK (evidencia_huella_sha256 ~ '^[0-9a-f]{64}$'),
    recibo_ref text NOT NULL UNIQUE,
    recibo_json jsonb NOT NULL CHECK (jsonb_typeof(recibo_json)='object'),
    registrado_en timestamptz(6) NOT NULL CHECK (isfinite(registrado_en) AND instante_en<=registrado_en),
    UNIQUE (organizacion_ref,clave_idempotencia),
    UNIQUE (organizacion_ref,comunicacion_ref,tipo),
    CHECK ((tipo='contacto_efectivo'
            AND respuesta_hasta IS NOT NULL AND isfinite(respuesta_hasta) AND respuesta_hasta>instante_en
            AND ultimo_dia IS NOT NULL AND ultimo_dia>=(instante_en AT TIME ZONE 'Europe/Madrid')::date
            AND politica_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
            AND politica_version>=1 AND politica_sha256 ~ '^[0-9a-f]{64}$'
            AND tratamiento_fuera_plazo IS NOT NULL AND confirmacion_expiracion IS NOT NULL
            AND material_json ? 'Plazo')
        OR (tipo='causa_justificada'
            AND num_nonnulls(respuesta_hasta,ultimo_dia,politica_ref,politica_version,politica_sha256,
                tratamiento_fuera_plazo,confirmacion_expiracion)=0
            AND NOT (material_json ? 'Plazo')))
);
COMMENT ON TABLE vec_contratacion_temporal.evento_plazo_llamamiento_rrhh IS
    'Hechos de plazo declarados por RRHH, append-only: contacto efectivo (abre el plazo calculado por el servidor con el catálogo) y causa justificada acreditada. No acreditan entrega del aviso, respuesta ni efecto en Bolsa.';
ALTER TABLE vec_contratacion_temporal.evento_plazo_llamamiento_rrhh ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contratacion_temporal.evento_plazo_llamamiento_rrhh FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario ON vec_contratacion_temporal.evento_plazo_llamamiento_rrhh
    TO vec_contratacion_temporal_propietario USING (true) WITH CHECK (true);
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE
    ON vec_contratacion_temporal.evento_plazo_llamamiento_rrhh FOR EACH ROW
    EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();
REVOKE ALL ON TABLE vec_contratacion_temporal.evento_plazo_llamamiento_rrhh
    FROM PUBLIC, vec_contratacion_temporal_ejecutor;

CREATE FUNCTION vec_contratacion_temporal.registrar_evento_plazo_llamamiento_v1(
    p_material text,
    p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,
    p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog SET row_security = on SET timezone = 'UTC'
SET lock_timeout = '2s'
AS $funcion$
DECLARE
    m jsonb; s jsonb; pl jsonb; d jsonb; v_referencia text; v_criterio record;
    v_material_huella text; v_contexto_huella text; v_consumo record;
    v_comunicacion vec_contratacion_temporal.comunicacion_llamamiento_local%ROWTYPE;
    v_contacto vec_contratacion_temporal.evento_plazo_llamamiento_rrhh%ROWTYPE;
    v_previa vec_contratacion_temporal.evento_plazo_llamamiento_rrhh%ROWTYPE;
    v_instante timestamptz(6); v_hasta timestamptz(6); v_dia date; v_ahora timestamptz(6);
    v_evento text; v_recibo text; v_resultado jsonb;
BEGIN
    IF current_user<>'vec_contratacion_temporal_propietario'
       OR session_user=current_user
       OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
       OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
       OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
       OR current_setting('transaction_isolation')<>'serializable'
       OR current_setting('transaction_read_only')<>'off' THEN
        RAISE EXCEPTION 'registro de plazo denegado' USING ERRCODE='P0593';
    END IF;
    IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 16384 THEN
        RAISE EXCEPTION 'material de plazo inválido' USING ERRCODE='P0590';
    END IF;
    BEGIN
        m:=p_material::jsonb;
    EXCEPTION WHEN data_exception THEN
        RAISE EXCEPTION 'JSON de plazo inválido' USING ERRCODE='P0590';
    END;
    s:=m->'Solicitud'; pl:=m->'Plazo';
    -- json.Marshal directo: jsonb elimina duplicados, se cuentan los originales.
    IF vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(s,ARRAY[
           'ClaveIdempotencia','OrganizacionRef','ExpedienteRef','LlamamientoRef','ComunicacionRef',
           'VersionComunicacionEsperada','Tipo','InstanteEn','PruebaRef']) IS NOT TRUE
       OR (SELECT count(*) FROM json_each((p_material::json)->'Solicitud'))<>9
       OR (s->>'Tipo'='contacto_efectivo' AND (
            vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(m,ARRAY['Solicitud','Plazo']) IS NOT TRUE
            OR (SELECT count(*) FROM json_each(p_material::json))<>2
            OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(pl,ARRAY[
                'RespuestaHasta','UltimoDia','Politica','TratamientoFueraDePlazo','ConfirmacionExpiracion',
                'CriterioRespuesta','CriterioExpiracion','ReglaEjemplo']) IS NOT TRUE
            OR (SELECT count(*) FROM json_each((p_material::json)->'Plazo'))<>8))
       OR (s->>'Tipo'='causa_justificada' AND (
            vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(m,ARRAY['Solicitud']) IS NOT TRUE
            OR (SELECT count(*) FROM json_each(p_material::json))<>1))
       OR (s->'Tipo') NOT IN ('"contacto_efectivo"'::jsonb,'"causa_justificada"'::jsonb) THEN
        RAISE EXCEPTION 'campos de plazo inválidos' USING ERRCODE='P0590';
    END IF;
    IF jsonb_typeof(s->'ClaveIdempotencia') IS DISTINCT FROM 'string'
       OR (s->>'ClaveIdempotencia')!~'^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
       OR s->>'ClaveIdempotencia'='00000000-0000-4000-8000-000000000000'
       OR jsonb_typeof(s->'VersionComunicacionEsperada') IS DISTINCT FROM 'number'
       OR s->>'VersionComunicacionEsperada' IS DISTINCT FROM '2'
       OR jsonb_typeof(s->'InstanteEn') IS DISTINCT FROM 'string'
       OR (s->>'InstanteEn')!~'^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,6})?Z$' THEN
        RAISE EXCEPTION 'solicitud de plazo inválida' USING ERRCODE='P0590';
    END IF;
    FOREACH v_referencia IN ARRAY ARRAY[
        'OrganizacionRef','ExpedienteRef','LlamamientoRef','ComunicacionRef','PruebaRef'
    ] LOOP
        IF jsonb_typeof(s->v_referencia) IS DISTINCT FROM 'string'
           OR (s->>v_referencia)!~'^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN
            RAISE EXCEPTION 'referencia de plazo inválida' USING ERRCODE='P0590';
        END IF;
    END LOOP;
    BEGIN
        v_instante:=(s->>'InstanteEn')::timestamptz;
        IF pl IS NOT NULL THEN
            IF jsonb_typeof(pl->'RespuestaHasta') IS DISTINCT FROM 'string'
               OR (pl->>'RespuestaHasta')!~'^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,6})?Z$'
               OR jsonb_typeof(pl->'UltimoDia') IS DISTINCT FROM 'string'
               OR (pl->>'UltimoDia')!~'^[0-9]{4}-[0-9]{2}-[0-9]{2}$' THEN
                RAISE EXCEPTION 'vencimiento inválido' USING ERRCODE='P0590';
            END IF;
            v_hasta:=(pl->>'RespuestaHasta')::timestamptz;
            v_dia:=(pl->>'UltimoDia')::date;
        END IF;
    EXCEPTION WHEN data_exception THEN
        RAISE EXCEPTION 'fecha de plazo inválida' USING ERRCODE='P0590';
    END;
    IF NOT isfinite(v_instante) OR (pl IS NOT NULL AND (NOT isfinite(v_hasta) OR v_hasta<=v_instante
        OR v_dia<(v_instante AT TIME ZONE 'Europe/Madrid')::date)) THEN
        RAISE EXCEPTION 'vencimiento incoherente' USING ERRCODE='P0590';
    END IF;
    IF pl IS NOT NULL THEN
        -- Referencias gobernadas: tres campos exactos y admitidas por uso.
        FOR v_criterio IN SELECT * FROM (VALUES
            (pl->'Politica','plazo'),(pl->'CriterioRespuesta','aceptacion'),
            (pl->'CriterioRespuesta','renuncia'),(pl->'CriterioExpiracion','expiracion_gobernada')
        ) AS c(ref,uso) LOOP
            IF vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(v_criterio.ref,ARRAY[
                   'Referencia','Version','HuellaSHA256']) IS NOT TRUE
               OR jsonb_typeof(v_criterio.ref->'Version') IS DISTINCT FROM 'number'
               OR (v_criterio.ref->>'Version')!~'^[1-9][0-9]{0,15}$'
               OR vec_contratacion_temporal.politica_llamamiento_admitida_v1(v_criterio.ref->>'Referencia',
                   (v_criterio.ref->>'Version')::numeric,v_criterio.ref->>'HuellaSHA256',v_criterio.uso) IS NULL THEN
                RAISE EXCEPTION 'política de plazo no admitida' USING ERRCODE='P0590';
            END IF;
        END LOOP;
        IF (SELECT count(*) FROM json_each((p_material::json)->'Plazo'->'Politica'))<>3
           OR (SELECT count(*) FROM json_each((p_material::json)->'Plazo'->'CriterioRespuesta'))<>3
           OR (SELECT count(*) FROM json_each((p_material::json)->'Plazo'->'CriterioExpiracion'))<>3
           OR (pl->'TratamientoFueraDePlazo') NOT IN ('"admitir"'::jsonb,'"exige_causa_justificada"'::jsonb,'"no_admitir"'::jsonb)
           OR pl->'ConfirmacionExpiracion' IS DISTINCT FROM '"rrhh"'::jsonb
           OR jsonb_typeof(pl->'ReglaEjemplo') IS DISTINCT FROM 'boolean' THEN
            RAISE EXCEPTION 'reglas de plazo inválidas' USING ERRCODE='P0590';
        END IF;
    END IF;
    v_material_huella:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
    v_contexto_huella:=encode(sha256(convert_to(
        '{"ambitos":{"organizacion_ref":"'||(s->>'OrganizacionRef')||
        '"},"atributos":{"material_sha256":"'||v_material_huella||'"}}','UTF8')),'hex');
    IF p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 1 AND 524288 THEN
        RAISE EXCEPTION 'decisión de plazo inválida' USING ERRCODE='P0593';
    END IF;
    BEGIN
        d:=convert_from(p_decision,'UTF8')::jsonb;
    EXCEPTION WHEN data_exception THEN
        RAISE EXCEPTION 'decisión de plazo inválida' USING ERRCODE='P0593';
    END;
    -- Mismo permiso nominal de validación manual de respuesta y plazo (AD3-17).
    IF d->>'accion' IS DISTINCT FROM 'contratacion_temporal.llamamiento.respuesta.validacion_manual.registrar'
       OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
       OR d->>'tipo_recurso' IS DISTINCT FROM 'resolucion_manual_respuesta_ct'
       OR d->>'finalidad' IS DISTINCT FROM 'gestionar_contratacion_temporal'
       OR d->>'recurso_ref' IS DISTINCT FROM s->>'ExpedienteRef'
       OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_contexto_huella THEN
        RAISE EXCEPTION 'autorización de plazo divergente' USING ERRCODE='P0593';
    END IF;
    -- Consumo nuevo antes de cualquier lectura o replay.
    BEGIN
        SELECT * INTO STRICT v_consumo
          FROM vec_autorizacion_atestada_v3.registrar_y_consumir_resolucion_manual_ct_v3_atestada(
            p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,
            p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
    EXCEPTION
        WHEN insufficient_privilege OR data_exception OR SQLSTATE 'P0583' THEN
            RAISE EXCEPTION 'consumo de plazo denegado' USING ERRCODE='P0593';
    END;
    IF v_consumo.consumo_nuevo IS NOT TRUE
       OR v_consumo.efecto_ref IS DISTINCT FROM s->>'ExpedienteRef'
       OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM v_contexto_huella THEN
        RAISE EXCEPTION 'plazo requiere consumo nuevo ligado al material' USING ERRCODE='P0593';
    END IF;
    -- El replay devuelve el recibo original aunque el catálogo haya cambiado:
    -- se compara la solicitud declarada, no el vencimiento recalculado.
    SELECT * INTO v_previa FROM vec_contratacion_temporal.evento_plazo_llamamiento_rrhh
     WHERE organizacion_ref=s->>'OrganizacionRef'
       AND clave_idempotencia=(s->>'ClaveIdempotencia')::uuid;
    IF FOUND THEN
        IF v_previa.actor_ref IS DISTINCT FROM d->>'principal_id'
           OR v_previa.perfil_ref IS DISTINCT FROM d->>'perfil_activo_ref' THEN
            RAISE EXCEPTION 'replay de plazo denegado' USING ERRCODE='P0593';
        END IF;
        IF v_previa.solicitud_json IS DISTINCT FROM s THEN
            RAISE EXCEPTION 'clave de plazo divergente' USING ERRCODE='P0591';
        END IF;
        RETURN v_previa.recibo_json || jsonb_build_object('Estado','replay_registrado');
    END IF;
    SELECT * INTO v_comunicacion FROM vec_contratacion_temporal.comunicacion_llamamiento_local c
     WHERE c.comunicacion_ref=s->>'ComunicacionRef'
       AND c.organizacion_ref=s->>'OrganizacionRef'
       AND c.expediente_ref=s->>'ExpedienteRef'
       AND c.llamamiento_ref=s->>'LlamamientoRef'
       AND c.version_resultante=2 AND c.estado='registrada_localmente'
     FOR SHARE;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'comunicación incompatible' USING ERRCODE='P0592';
    END IF;
    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.resolucion_manual_respuesta_rrhh
        WHERE organizacion_ref=s->>'OrganizacionRef' AND comunicacion_ref=s->>'ComunicacionRef') THEN
        RAISE EXCEPTION 'comunicación ya resuelta' USING ERRCODE='P0592';
    END IF;
    v_ahora:=date_trunc('microseconds',clock_timestamp());
    IF v_instante>v_ahora OR v_instante<v_comunicacion.registrada_en THEN
        RAISE EXCEPTION 'instante de plazo fuera de rango' USING ERRCODE='P0590';
    END IF;
    IF s->>'Tipo'='causa_justificada' THEN
        SELECT * INTO v_contacto FROM vec_contratacion_temporal.evento_plazo_llamamiento_rrhh
         WHERE organizacion_ref=s->>'OrganizacionRef' AND comunicacion_ref=s->>'ComunicacionRef'
           AND tipo='contacto_efectivo'
         FOR SHARE;
        IF NOT FOUND OR v_contacto.tratamiento_fuera_plazo IS DISTINCT FROM 'exige_causa_justificada' THEN
            RAISE EXCEPTION 'causa justificada sin plazo que la admita' USING ERRCODE='P0592';
        END IF;
        IF v_instante<v_contacto.instante_en THEN
            RAISE EXCEPTION 'causa anterior al contacto efectivo' USING ERRCODE='P0590';
        END IF;
    END IF;
    v_evento:='evento:'||gen_random_uuid()::text;
    v_recibo:='recibo:'||gen_random_uuid()::text;
    v_resultado:=jsonb_build_object('Solicitud',s,'EventoRef',v_evento,'ReciboRef',v_recibo,
        'AuditoriaRef',v_consumo.auditoria_ref,
        'RegistradoEn',to_char(v_ahora,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'Estado','registrado');
    IF pl IS NOT NULL THEN
        v_resultado:=v_resultado||jsonb_build_object('Plazo',pl);
    END IF;
    INSERT INTO vec_contratacion_temporal.evento_plazo_llamamiento_rrhh (
        evento_ref,tipo,organizacion_ref,expediente_ref,llamamiento_ref,comunicacion_ref,
        clave_idempotencia,actor_ref,perfil_ref,instante_en,prueba_ref,
        respuesta_hasta,ultimo_dia,politica_ref,politica_version,politica_sha256,
        tratamiento_fuera_plazo,confirmacion_expiracion,
        material,material_json,material_huella_sha256,solicitud_json,
        auditoria_ref,decision_ref,consumo_huella_sha256,evidencia_huella_sha256,
        recibo_ref,recibo_json,registrado_en
    ) VALUES (
        v_evento,s->>'Tipo',s->>'OrganizacionRef',s->>'ExpedienteRef',s->>'LlamamientoRef',s->>'ComunicacionRef',
        (s->>'ClaveIdempotencia')::uuid,d->>'principal_id',d->>'perfil_activo_ref',v_instante,s->>'PruebaRef',
        v_hasta,v_dia,pl->'Politica'->>'Referencia',(pl->'Politica'->>'Version')::numeric,pl->'Politica'->>'HuellaSHA256',
        pl->>'TratamientoFueraDePlazo',pl->>'ConfirmacionExpiracion',
        p_material,m,v_material_huella,s,
        v_consumo.auditoria_ref,v_consumo.decision_ref,v_consumo.consumo_huella_sha256,encode(sha256(p_evidencia),'hex'),
        v_recibo,v_resultado,v_ahora);
    RETURN v_resultado;
EXCEPTION
    WHEN unique_violation THEN
        RAISE EXCEPTION 'evento de plazo ya registrado' USING ERRCODE='P0592';
    WHEN serialization_failure OR deadlock_detected OR lock_not_available THEN
        RAISE EXCEPTION 'registro de plazo no disponible' USING ERRCODE='P0594';
END
$funcion$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.registrar_evento_plazo_llamamiento_v1(
    text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
    FROM PUBLIC, vec_contratacion_temporal_ejecutor;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.registrar_evento_plazo_llamamiento_v1(
    text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
    TO vec_contratacion_temporal_ejecutor;

-- Filas históricas intactas: todas cumplen las nuevas restricciones porque
-- son aceptaciones o renuncias con justificante, sin contacto ni causa.
ALTER TABLE vec_contratacion_temporal.resolucion_manual_respuesta_rrhh
    DROP CONSTRAINT resolucion_manual_respuesta_rrhh_politica_ref_check,
    DROP CONSTRAINT resolucion_manual_respuesta_rrhh_politica_version_check,
    DROP CONSTRAINT resolucion_manual_respuesta_rrhh_politica_sha256_check,
    DROP CONSTRAINT resolucion_manual_respuesta_rrhh_estado_plazo_check,
    DROP CONSTRAINT resolucion_manual_comando_siguiente_check,
    ALTER COLUMN justificante_ref DROP NOT NULL,
    ADD COLUMN contacto_ref text REFERENCES vec_contratacion_temporal.evento_plazo_llamamiento_rrhh,
    ADD COLUMN causa_justificada_ref text REFERENCES vec_contratacion_temporal.evento_plazo_llamamiento_rrhh,
    ADD COLUMN respuesta_fuera_de_plazo boolean NOT NULL DEFAULT false,
    ADD CONSTRAINT resolucion_manual_politica_forma_check CHECK (
        politica_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND politica_version>=1 AND politica_sha256 ~ '^[0-9a-f]{64}$'),
    ADD CONSTRAINT resolucion_manual_estado_plazo_check CHECK (estado_plazo IN ('vigente','expirado')),
    ADD CONSTRAINT resolucion_manual_plazo_coherente_check CHECK ((
        (solicitud_json->>'Respuesta'='expiracion_gobernada')=(estado_plazo='expirado')
        AND (solicitud_json->>'Respuesta'='expiracion_gobernada')=(justificante_ref IS NULL)
        AND (solicitud_json->>'Respuesta'<>'expiracion_gobernada' OR contacto_ref IS NOT NULL)
        AND (causa_justificada_ref IS NULL OR (respuesta_fuera_de_plazo AND contacto_ref IS NOT NULL))
        AND (NOT respuesta_fuera_de_plazo OR contacto_ref IS NOT NULL)
    ) IS TRUE),
    ADD CONSTRAINT resolucion_manual_comando_siguiente_check CHECK ((
        (solicitud_json->>'Respuesta'='aceptacion'
            AND comando_siguiente_ref IS NULL AND comando_siguiente_json IS NULL)
        OR (solicitud_json->>'Respuesta' IN ('renuncia','expiracion_gobernada')
            AND comando_siguiente_ref IS NOT NULL AND comando_siguiente_json IS NOT NULL
            AND comando_siguiente_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
            AND vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(comando_siguiente_json,ARRAY[
                'esquema','comando_ref','intencion_ref','organizacion_ref','expediente_ref',
                'llamamiento_ref','justificante_ref','seleccion_clave']) IS TRUE
            AND comando_siguiente_json->>'esquema'='vec.contratacion-temporal.siguiente-candidato.intencion.v1'
            AND comando_siguiente_json->>'comando_ref'=comando_siguiente_ref
            AND comando_siguiente_json->>'intencion_ref' ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
            AND comando_siguiente_json->>'organizacion_ref'=organizacion_ref
            AND comando_siguiente_json->>'expediente_ref'=expediente_ref
            AND comando_siguiente_json->>'llamamiento_ref'=llamamiento_ref
            AND comando_siguiente_json->>'justificante_ref' IS NOT DISTINCT FROM justificante_ref
            AND comando_siguiente_json->>'seleccion_clave'=seleccion_clave::text
            AND recibo_json->'IntencionSiguiente'=jsonb_build_object(
                'Solicitud',solicitud_json,'ResolucionRef',resolucion_ref,'LlamamientoRef',llamamiento_ref,
                'ClaveIdempotencia',clave_idempotencia::text,'VersionEsperada',2,'VersionResultante',3,
                'IntencionRef',comando_siguiente_json->>'intencion_ref','ComandoOpacoRef',comando_siguiente_ref,
                'Estado','pendiente','ActualizadaEn',recibo_json->>'ResueltaEn'))
    ) IS TRUE);
COMMENT ON COLUMN vec_contratacion_temporal.resolucion_manual_respuesta_rrhh.contacto_ref IS
    'Contacto efectivo CT111 cuyo vencimiento evaluó la resolución; NULL en las resoluciones históricas sin plazo abierto.';
COMMENT ON COLUMN vec_contratacion_temporal.resolucion_manual_respuesta_rrhh.causa_justificada_ref IS
    'Causa justificada acreditada que admitió una respuesta fuera de plazo; no la deduce el sistema.';

-- Fragmentos exactos sobre el cuerpo CT64; cada uno debe aparecer una vez.
DO $resolucion$
DECLARE v_antes record; v_despues record; v_fragmento record; v_definicion text;
BEGIN
    SELECT p.oid,pg_get_functiondef(p.oid) AS definicion,p.proacl AS acl,
           p.proowner AS propietario,p.proconfig AS configuracion
      INTO STRICT v_antes FROM pg_proc p
     WHERE p.oid='vec_contratacion_temporal.registrar_resolucion_manual_respuesta_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
    v_definicion:=v_antes.definicion;
    FOR v_fragmento IN SELECT anterior,nuevo FROM (VALUES
    (1,$a1$    v_comando_ref text; v_comando_json jsonb; v_intencion_ref text; v_intencion_siguiente jsonb;$a1$,
       $n1$    v_comando_ref text; v_comando_json jsonb; v_intencion_ref text; v_intencion_siguiente jsonb;
    -- CT111: plazo evaluado con hechos durables, nunca con el material.
    v_tipo_politica text; v_contacto vec_contratacion_temporal.evento_plazo_llamamiento_rrhh%ROWTYPE;
    v_con_contacto boolean; v_seleccion_expiracion uuid; v_estado_plazo text;
    v_fuera_plazo boolean; v_causa_ref text;$n1$),
    (2,$a2$       OR (s->'Respuesta') NOT IN ('"aceptacion"'::jsonb,'"renuncia"'::jsonb)$a2$,
       $n2$       OR (s->'Respuesta') NOT IN ('"aceptacion"'::jsonb,'"renuncia"'::jsonb,'"expiracion_gobernada"'::jsonb)
       OR (s->>'Respuesta'='expiracion_gobernada' AND s->'PruebaRespuestaRef' IS DISTINCT FROM '""'::jsonb)$n2$),
    (3,$a3$        'OrganizacionRef','ExpedienteRef','LlamamientoRef','ComunicacionRef','PruebaRespuestaRef'
    ] LOOP$a3$,
       $n3$        'OrganizacionRef','ExpedienteRef','LlamamientoRef','ComunicacionRef'
    ] || CASE WHEN s->>'Respuesta'='expiracion_gobernada' THEN ARRAY[]::text[]
              ELSE ARRAY['PruebaRespuestaRef'] END LOOP$n3$),
    (4,$a4$    IF s->'CriterioValidacionRef' IS DISTINCT FROM '"politica:ct:revision-manual-sintetica:20260906"'::jsonb
       OR p->'Referencia' IS DISTINCT FROM s->'CriterioValidacionRef'
       OR jsonb_typeof(p->'Version') IS DISTINCT FROM 'number'
       OR p->>'Version' IS DISTINCT FROM '1'
       OR p->'HuellaSHA256' IS DISTINCT FROM '"ea41d65808044fa75b597855e81a469ed274403a521890bafa07c33ae89ec2e3"'::jsonb THEN
        RAISE EXCEPTION 'política de ejercicio sintético incompatible' USING ERRCODE='P0580';
    END IF;$a4$,
       $n4$    -- CT111: política admitida por la lista versionada, no por un literal.
    IF jsonb_typeof(s->'CriterioValidacionRef') IS DISTINCT FROM 'string'
       OR p->'Referencia' IS DISTINCT FROM s->'CriterioValidacionRef'
       OR jsonb_typeof(p->'Version') IS DISTINCT FROM 'number'
       OR (p->>'Version')!~'^[1-9][0-9]{0,15}$'
       OR jsonb_typeof(p->'HuellaSHA256') IS DISTINCT FROM 'string' THEN
        RAISE EXCEPTION 'política de resolución inválida' USING ERRCODE='P0580';
    END IF;
    v_tipo_politica:=vec_contratacion_temporal.politica_llamamiento_admitida_v1(
        p->>'Referencia',(p->>'Version')::numeric,p->>'HuellaSHA256',s->>'Respuesta');
    IF v_tipo_politica IS NULL THEN
        RAISE EXCEPTION 'política de resolución no admitida' USING ERRCODE='P0580';
    END IF;$n4$),
    (5,$a5$    -- Solo tablas propias CT. La declaración, el aviso local y la selección
    -- original deben conservar todas sus coordenadas y la propuesta confirmada.
    SELECT r.* INTO v_declaracion$a5$,
       $n5$    -- CT111: contacto efectivo de esta comunicación, si RRHH lo registró.
    SELECT * INTO v_contacto FROM vec_contratacion_temporal.evento_plazo_llamamiento_rrhh
     WHERE organizacion_ref=s->>'OrganizacionRef' AND comunicacion_ref=s->>'ComunicacionRef'
       AND expediente_ref=s->>'ExpedienteRef' AND llamamiento_ref=s->>'LlamamientoRef'
       AND tipo='contacto_efectivo'
     FOR SHARE;
    v_con_contacto:=FOUND;
    IF s->>'Respuesta'='expiracion_gobernada' THEN
        -- Sin respuesta personal: el antecedente es el aviso local con plazo abierto.
        IF NOT v_con_contacto THEN
            RAISE EXCEPTION 'expiración sin contacto efectivo' USING ERRCODE='P0582';
        END IF;
        SELECT c.seleccion_clave INTO v_seleccion_expiracion
          FROM vec_contratacion_temporal.comunicacion_llamamiento_local c
         WHERE c.comunicacion_ref=s->>'ComunicacionRef'
           AND c.organizacion_ref=s->>'OrganizacionRef'
           AND c.expediente_ref=s->>'ExpedienteRef'
           AND c.llamamiento_ref=s->>'LlamamientoRef'
           AND c.version_resultante=2 AND c.estado='registrada_localmente'
         FOR SHARE;
        IF NOT FOUND THEN
            RAISE EXCEPTION 'aviso incompatible con la expiración' USING ERRCODE='P0582';
        END IF;
        v_declaracion.justificante_ref:=NULL;
        v_declaracion.seleccion_clave:=v_seleccion_expiracion;
        v_declaracion.registrada_en:=v_contacto.registrado_en;
    ELSE
    -- Solo tablas propias CT. La declaración, el aviso local y la selección
    -- original deben conservar todas sus coordenadas y la propuesta confirmada.
    SELECT r.* INTO v_declaracion$n5$),
    (6,$a6$        RAISE EXCEPTION 'justificante o antecedentes incompatibles' USING ERRCODE='P0582';
    END IF;$a6$,
       $n6$        RAISE EXCEPTION 'justificante o antecedentes incompatibles' USING ERRCODE='P0582';
    END IF;
    END IF;$n6$),
    (7,$a7$        RAISE EXCEPTION 'reloj anterior al justificante registrado' USING ERRCODE='P0584';
    END IF;$a7$,
       $n7$        RAISE EXCEPTION 'reloj anterior al justificante registrado' USING ERRCODE='P0584';
    END IF;
    -- CT111: la regla aplicable es la capturada al abrir el plazo.
    v_estado_plazo:='vigente'; v_fuera_plazo:=false; v_causa_ref:=NULL;
    IF v_tipo_politica='entrada_catalogo' AND NOT v_con_contacto THEN
        RAISE EXCEPTION 'política de catálogo sin contacto efectivo' USING ERRCODE='P0582';
    END IF;
    IF v_con_contacto AND s->>'Respuesta'='expiracion_gobernada' THEN
        IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.respuesta_recibida_rrhh
            WHERE organizacion_ref=s->>'OrganizacionRef' AND comunicacion_ref=s->>'ComunicacionRef') THEN
            RAISE EXCEPTION 'respuesta registrada: no procede expiración' USING ERRCODE='P0582';
        END IF;
        IF v_ahora<v_contacto.respuesta_hasta THEN
            RAISE EXCEPTION 'plazo de respuesta no vencido' USING ERRCODE='P0586';
        END IF;
        IF v_contacto.confirmacion_expiracion IS DISTINCT FROM 'rrhh' THEN
            RAISE EXCEPTION 'confirmación de expiración no admitida' USING ERRCODE='P0580';
        END IF;
        v_estado_plazo:='expirado';
    ELSIF v_con_contacto AND v_declaracion.recibida_en>=v_contacto.respuesta_hasta THEN
        v_fuera_plazo:=true;
        IF v_contacto.tratamiento_fuera_plazo='exige_causa_justificada' THEN
            SELECT evento_ref INTO v_causa_ref FROM vec_contratacion_temporal.evento_plazo_llamamiento_rrhh
             WHERE organizacion_ref=s->>'OrganizacionRef' AND comunicacion_ref=s->>'ComunicacionRef'
               AND tipo='causa_justificada'
             FOR SHARE;
            IF NOT FOUND THEN
                RAISE EXCEPTION 'respuesta fuera de plazo sin causa justificada acreditada' USING ERRCODE='P0585';
            END IF;
        ELSIF v_contacto.tratamiento_fuera_plazo IS DISTINCT FROM 'admitir' THEN
            RAISE EXCEPTION 'respuesta fuera de plazo no admitida' USING ERRCODE='P0585';
        END IF;
    END IF;$n7$),
    (8,$a8$    IF s->>'Respuesta'='renuncia' THEN
        v_comando_ref:='comando:'||gen_random_uuid()::text;$a8$,
       $n8$    IF s->>'Respuesta' IN ('renuncia','expiracion_gobernada') THEN
        v_comando_ref:='comando:'||gen_random_uuid()::text;$n8$),
    (9,$a9$        'Solicitud',s,'Politica',p,'EvaluacionPlazoRef',v_evaluacion,'EstadoPlazo','vigente',$a9$,
       $n9$        'Solicitud',s,'Politica',p,'EvaluacionPlazoRef',v_evaluacion,'EstadoPlazo',v_estado_plazo,$n9$),
    (10,$a10$        'ResueltaEn',to_char(v_ahora,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'Estado','confirmado');
    INSERT INTO$a10$,
       $n10$        'ResueltaEn',to_char(v_ahora,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'Estado','confirmado');
    IF v_con_contacto THEN
        v_resultado:=v_resultado||jsonb_build_object(
            'RespuestaHasta',to_char(v_contacto.respuesta_hasta,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
            'RespuestaFueraDePlazo',v_fuera_plazo)
            ||CASE WHEN v_causa_ref IS NULL THEN '{}'::jsonb
                   ELSE jsonb_build_object('CausaJustificadaRef',v_causa_ref) END;
    END IF;
    INSERT INTO$n10$),
    (11,$a11$        comando_siguiente_ref,comando_siguiente_json
    ) VALUES ($a11$,
       $n11$        comando_siguiente_ref,comando_siguiente_json,
        contacto_ref,causa_justificada_ref,respuesta_fuera_de_plazo
    ) VALUES ($n11$),
    (12,$a12$        d->>'principal_id',d->>'perfil_activo_ref',true,true,p->>'Referencia',1,p->>'HuellaSHA256',
        v_evaluacion,'vigente',3,p_material,m,v_material_huella,s,v_consumo.auditoria_ref,$a12$,
       $n12$        d->>'principal_id',d->>'perfil_activo_ref',true,true,p->>'Referencia',(p->>'Version')::numeric,p->>'HuellaSHA256',
        v_evaluacion,v_estado_plazo,3,p_material,m,v_material_huella,s,v_consumo.auditoria_ref,$n12$),
    (13,$a13$        v_recibo,v_resultado,'confirmado',v_ahora,v_comando_ref,v_comando_json);$a13$,
       $n13$        v_recibo,v_resultado,'confirmado',v_ahora,v_comando_ref,v_comando_json,
        CASE WHEN v_con_contacto THEN v_contacto.evento_ref END,v_causa_ref,v_fuera_plazo);$n13$)
    ) AS fragmentos(orden,anterior,nuevo) ORDER BY orden
    LOOP
        IF length(v_definicion)-length(replace(v_definicion,v_fragmento.anterior,''))<>length(v_fragmento.anterior) THEN
            RAISE EXCEPTION 'fragmento CT111 incompatible' USING ERRCODE='55000';
        END IF;
        v_definicion:=replace(v_definicion,v_fragmento.anterior,v_fragmento.nuevo);
    END LOOP;
    EXECUTE v_definicion;
    SELECT p.prosrc AS cuerpo,pg_get_functiondef(p.oid) AS definicion,p.proacl AS acl,
           p.proowner AS propietario,p.proconfig AS configuracion,p.prosecdef AS definidor
      INTO STRICT v_despues FROM pg_proc p WHERE p.oid=v_antes.oid;
    IF v_despues.definicion IS DISTINCT FROM v_definicion
       OR encode(sha256(convert_to(v_despues.cuerpo,'UTF8')),'hex')
          IS DISTINCT FROM 'c0c5d353a295b70920d6e3e831b1a3a9faee7165e22cfad452f817d656ab8a74'
       OR v_despues.acl IS DISTINCT FROM v_antes.acl
       OR v_despues.propietario IS DISTINCT FROM v_antes.propietario
       OR v_despues.configuracion IS DISTINCT FROM v_antes.configuracion
       OR v_despues.definidor IS NOT TRUE THEN
        RAISE EXCEPTION 'CT111 alteró definición o permisos de la resolución' USING ERRCODE='55000';
    END IF;
END
$resolucion$;
COMMIT;
