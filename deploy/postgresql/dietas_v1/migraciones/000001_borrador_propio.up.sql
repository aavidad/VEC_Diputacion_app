\set ON_ERROR_STOP on
-- Bootstrap nuevo, por migrador DBA. Después debe instalarse el consumidor
-- nominal AD3 de Dietas. No instala permisos funcionales ni cuentas LOGIN.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_dietas_v1:000001',0));

DO $precondiciones$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
       OR to_regnamespace('vec_dietas_v1') IS NOT NULL
       OR EXISTS (SELECT 1 FROM pg_roles WHERE rolname IN
           ('vec_dietas_v1_propietario','vec_dietas_v1_ejecutor'))
       OR to_regnamespace('vec_autorizacion_atestada_v3') IS NULL THEN
        RAISE EXCEPTION 'precondiciones de Dietas incompatibles' USING ERRCODE='55000';
    END IF;
END
$precondiciones$;
CREATE ROLE vec_dietas_v1_propietario NOLOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_dietas_v1_ejecutor NOLOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
CREATE SCHEMA vec_dietas_v1 AUTHORIZATION vec_dietas_v1_propietario;
REVOKE ALL ON SCHEMA vec_dietas_v1 FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_dietas_v1 TO vec_dietas_v1_ejecutor;
SET LOCAL ROLE vec_dietas_v1_propietario;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_dietas_v1_propietario
    REVOKE ALL ON FUNCTIONS FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_dietas_v1_propietario
    IN SCHEMA vec_dietas_v1 REVOKE ALL ON TABLES FROM PUBLIC;

CREATE FUNCTION vec_dietas_v1.objeto_exacto_v1(p_objeto jsonb,p_claves text[])
RETURNS boolean LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog AS $f$
BEGIN
    IF jsonb_typeof(p_objeto) IS DISTINCT FROM 'object' THEN RETURN false; END IF;
    RETURN p_objeto ?& p_claves AND p_objeto-p_claves='{}'::jsonb;
END
$f$;

-- Defensa del contrato persistido. La política llega resuelta por el servidor.
-- La ruta y sus referencias de fuente/grafo/cálculo son declaraciones NO
-- verificadas: este borrador exige revalidación antes de envío o liquidación.
-- El SQL no consulta otro módulo ni certifica un resultado OSRM.
CREATE FUNCTION vec_dietas_v1.validar_borrador_completo_v1(b jsonb)
RETURNS boolean LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog AS $f$
DECLARE
    r jsonb:=b->'ruta'; g jsonb:=b->'gastos'; p jsonb:=b->'politica_kilometraje';
    d jsonb:=b->'desglose'; e jsonb; i integer; total_km numeric:=0;
    km_eur numeric; total_eur numeric;
    codigo text:='^[A-Za-z0-9][A-Za-z0-9._:-]{1,63}$';
    dinero text:='^(0|[1-9][0-9]{0,8})[.][0-9]{2}$';
    kilometros text:='^(0|[1-9][0-9]{0,4})([.][0-9]{1,4})?$';
BEGIN
    IF NOT vec_dietas_v1.objeto_exacto_v1(b,ARRAY['persona_ref','objeto','inicio','fin','fecha','fecha_fin','hora_inicio','hora_fin','zona_horaria','vehiculo_propio','ruta','gastos','politica_kilometraje','desglose','estado','contexto_personal','liquidable','procedencia_ruta','revalidacion_ruta_requerida','ruta_etiquetas'])
       OR NOT vec_dietas_v1.objeto_exacto_v1(r,ARRAY['fuente','version','referencia','catalogo_version','alternativa_ref','kilometros','paradas','tramos','trazado','recomendada','motivo_alternativa','liquidable'])
       OR NOT vec_dietas_v1.objeto_exacto_v1(g,ARRAY['manutencion_eur','alojamiento_eur','otros_eur'])
       OR NOT vec_dietas_v1.objeto_exacto_v1(p,ARRAY['referencia','version','tarifa_eur_km'])
       OR NOT vec_dietas_v1.objeto_exacto_v1(d,ARRAY['kilometraje_eur','manutencion_eur','alojamiento_eur','otros_eur','total_eur']) THEN RETURN false; END IF;
    IF EXISTS (SELECT 1 FROM jsonb_each(b-ARRAY['vehiculo_propio','ruta','gastos','politica_kilometraje','desglose','liquidable','revalidacion_ruta_requerida','ruta_etiquetas']) e WHERE jsonb_typeof(e.value) IS DISTINCT FROM 'string')
       OR EXISTS (SELECT 1 FROM jsonb_each(r-ARRAY['paradas','tramos','trazado','recomendada','liquidable']) e WHERE jsonb_typeof(e.value) IS DISTINCT FROM 'string')
       OR EXISTS (SELECT 1 FROM jsonb_each(p) e WHERE jsonb_typeof(e.value) IS DISTINCT FROM 'string')
       OR EXISTS (SELECT 1 FROM unnest(ARRAY[b->>'objeto',b->>'zona_horaria',r->>'version',
           r->>'catalogo_version',r->>'motivo_alternativa',p->>'version']) AS z(texto)
           WHERE texto ~ '[[:cntrl:]]|^[[:space:]]|[[:space:]]$')
       OR EXISTS (SELECT 1 FROM (SELECT value FROM jsonb_each(g) UNION ALL SELECT value FROM jsonb_each(d)) e
           WHERE jsonb_typeof(e.value) IS DISTINCT FROM 'string' OR e.value#>>'{}' !~ dinero)
       OR jsonb_typeof(b->'vehiculo_propio') IS DISTINCT FROM 'boolean'
       OR jsonb_typeof(r->'recomendada') IS DISTINCT FROM 'boolean'
       OR b->>'estado' IS DISTINCT FROM 'borrador'
       OR b->>'contexto_personal' IS DISTINCT FROM 'contexto_personal_pendiente'
       OR b->>'procedencia_ruta' IS DISTINCT FROM 'declarada_no_verificada'
       OR b->'revalidacion_ruta_requerida' IS DISTINCT FROM 'true'::jsonb
       OR b->'liquidable' IS DISTINCT FROM 'false'::jsonb
       OR r->'liquidable' IS DISTINCT FROM 'false'::jsonb
       OR b->>'persona_ref' !~ '^[a-z][A-Za-z0-9_:-]{2,180}$'
       OR octet_length(b->>'objeto') NOT BETWEEN 1 AND 500
       OR btrim(b->>'objeto') IS DISTINCT FROM b->>'objeto'
       OR b->>'inicio' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}([.][0-9]{1,6})?Z$'
       OR b->>'fin' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}([.][0-9]{1,6})?Z$'
       OR b->>'fecha' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'
       OR b->>'fecha_fin' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'
       OR b->>'hora_inicio' !~ '^[0-9]{2}:[0-9]{2}$'
       OR b->>'hora_fin' !~ '^[0-9]{2}:[0-9]{2}$'
       OR octet_length(b->>'zona_horaria') NOT BETWEEN 1 AND 100
       OR r->>'fuente' IS DISTINCT FROM 'osrm_interno'
       OR octet_length(r->>'version') NOT BETWEEN 1 AND 160
       OR btrim(r->>'version') IS DISTINCT FROM r->>'version'
       OR octet_length(r->>'catalogo_version') NOT BETWEEN 1 AND 160
       OR btrim(r->>'catalogo_version') IS DISTINCT FROM r->>'catalogo_version'
       OR r->>'referencia' !~ codigo OR r->>'alternativa_ref' !~ codigo
       OR r->>'kilometros' !~ kilometros
       OR (r->>'kilometros')::numeric>10000
       OR octet_length(r->>'motivo_alternativa')>500
       OR btrim(r->>'motivo_alternativa') IS DISTINCT FROM r->>'motivo_alternativa'
       OR (r->'recomendada'='false'::jsonb AND r->>'motivo_alternativa'='')
       OR (r->'recomendada'='true'::jsonb AND r->>'motivo_alternativa'<>'')
       OR p->>'referencia' !~ '^[a-z][A-Za-z0-9_:-]{2,180}$'
       OR octet_length(p->>'version') NOT BETWEEN 1 AND 160
       OR btrim(p->>'version') IS DISTINCT FROM p->>'version'
       OR p->>'tarifa_eur_km' !~ '^(0|[1-9][0-9]{0,3})[.][0-9]{4}$'
       OR (p->>'tarifa_eur_km')::numeric>1000 THEN RETURN false; END IF;
    IF jsonb_typeof(r->'paradas') IS DISTINCT FROM 'array'
       OR jsonb_typeof(r->'tramos') IS DISTINCT FROM 'array'
       OR jsonb_typeof(r->'trazado') IS DISTINCT FROM 'array'
       OR jsonb_typeof(b->'ruta_etiquetas') IS DISTINCT FROM 'array' THEN RETURN false; END IF;
    IF b->'ruta_etiquetas' IS DISTINCT FROM (SELECT jsonb_agg(value->'nombre' ORDER BY ord) FROM jsonb_array_elements(r->'paradas') WITH ORDINALITY AS p(value,ord))
       OR jsonb_array_length(r->'paradas') NOT BETWEEN 2 AND 12
       OR jsonb_array_length(r->'tramos')<>jsonb_array_length(r->'paradas')-1
       OR jsonb_array_length(r->'trazado') NOT BETWEEN 2 AND 2000 THEN RETURN false; END IF;
    FOR e IN SELECT value FROM jsonb_array_elements(r->'paradas') LOOP
        IF NOT vec_dietas_v1.objeto_exacto_v1(e,ARRAY['codigo','nombre','latitud','longitud'])
           OR jsonb_typeof(e->'codigo') IS DISTINCT FROM 'string'
           OR jsonb_typeof(e->'nombre') IS DISTINCT FROM 'string'
           OR jsonb_typeof(e->'latitud') IS DISTINCT FROM 'number'
           OR jsonb_typeof(e->'longitud') IS DISTINCT FROM 'number'
           OR e->>'codigo' !~ codigo
           OR octet_length(e->>'nombre') NOT BETWEEN 1 AND 100
           OR btrim(e->>'nombre') IS DISTINCT FROM e->>'nombre'
           OR e->>'nombre' ~ '[[:cntrl:]]|^[[:space:]]|[[:space:]]$'
           OR (e->>'latitud')::numeric NOT BETWEEN -90 AND 90
           OR (e->>'longitud')::numeric NOT BETWEEN -180 AND 180 THEN RETURN false; END IF;
    END LOOP;
    FOR i IN 0..jsonb_array_length(r->'tramos')-1 LOOP
        e:=r->'tramos'->i;
        IF NOT vec_dietas_v1.objeto_exacto_v1(e,ARRAY['origen_codigo','destino_codigo','kilometros','duracion_minutos','ajuste_kilometros','motivo_ajuste'])
           OR EXISTS (SELECT 1 FROM jsonb_each(e-'duracion_minutos') z WHERE jsonb_typeof(z.value) IS DISTINCT FROM 'string')
           OR e->>'origen_codigo' IS DISTINCT FROM r->'paradas'->i->>'codigo'
           OR e->>'destino_codigo' IS DISTINCT FROM r->'paradas'->(i+1)->>'codigo'
           OR e->>'origen_codigo'=e->>'destino_codigo'
           OR e->>'kilometros' !~ kilometros
           OR (e->>'kilometros')::numeric<=0 OR (e->>'kilometros')::numeric>10000
           OR jsonb_typeof(e->'duracion_minutos') IS DISTINCT FROM 'number'
           OR e->>'duracion_minutos' !~ '^[1-9][0-9]{0,4}$'
           OR (e->>'duracion_minutos')::numeric>20000
           OR e->>'ajuste_kilometros' !~ kilometros
           OR (e->>'ajuste_kilometros')::numeric>1000
           OR octet_length(e->>'motivo_ajuste')>500
           OR btrim(e->>'motivo_ajuste') IS DISTINCT FROM e->>'motivo_ajuste'
           OR e->>'motivo_ajuste' ~ '[[:cntrl:]]|^[[:space:]]|[[:space:]]$'
           OR ((e->>'ajuste_kilometros')::numeric>0 AND e->>'motivo_ajuste'='')
           OR ((e->>'ajuste_kilometros')::numeric=0 AND e->>'motivo_ajuste'<>'') THEN RETURN false; END IF;
        total_km:=total_km+(e->>'kilometros')::numeric+(e->>'ajuste_kilometros')::numeric;
    END LOOP;
    FOR e IN SELECT value FROM jsonb_array_elements(r->'trazado') LOOP
        IF jsonb_typeof(e) IS DISTINCT FROM 'array' THEN RETURN false; END IF;
        IF jsonb_array_length(e)<>2 OR jsonb_typeof(e->0) IS DISTINCT FROM 'number'
           OR jsonb_typeof(e->1) IS DISTINCT FROM 'number'
           OR (e->>0)::numeric NOT BETWEEN -90 AND 90
           OR (e->>1)::numeric NOT BETWEEN -180 AND 180 THEN RETURN false; END IF;
    END LOOP;
    IF total_km<>(r->>'kilometros')::numeric THEN RETURN false; END IF;
    km_eur:=CASE WHEN b->'vehiculo_propio'='true'::jsonb
        THEN round(total_km*(p->>'tarifa_eur_km')::numeric,2) ELSE 0 END;
    total_eur:=km_eur+(g->>'manutencion_eur')::numeric+(g->>'alojamiento_eur')::numeric+(g->>'otros_eur')::numeric;
    RETURN (d->>'kilometraje_eur')::numeric=km_eur
       AND d->>'manutencion_eur'=g->>'manutencion_eur'
       AND d->>'alojamiento_eur'=g->>'alojamiento_eur'
       AND d->>'otros_eur'=g->>'otros_eur'
       AND (d->>'total_eur')::numeric=total_eur;
EXCEPTION WHEN data_exception THEN RETURN false;
END
$f$;

-- Mismo horizonte del dominio para detectar offsets antes/después de una
-- transición: exige un solo instante civil, incluso en retrocesos de 30 minutos.
CREATE FUNCTION vec_dietas_v1.instante_civil_unico_v1(
    p_instante timestamptz,p_fecha text,p_hora text,p_zona text
) RETURNS boolean LANGUAGE sql STABLE SET search_path=pg_catalog AS $f$
    WITH civil AS (
        SELECT (p_fecha||' '||p_hora)::timestamp AS fecha
    ), offsets AS (
        SELECT DISTINCT (muestra AT TIME ZONE p_zona)-(muestra AT TIME ZONE 'UTC') AS diferencia
        FROM civil, LATERAL generate_series(
            (fecha AT TIME ZONE 'UTC')-interval '48 hours',
            (fecha AT TIME ZONE 'UTC')+interval '48 hours',interval '1 hour') AS s(muestra)
    ), candidatos AS (
        SELECT (fecha AT TIME ZONE 'UTC')-diferencia AS instante,fecha
        FROM civil,offsets
    )
    SELECT count(*)=1 AND min(instante)=p_instante
    FROM candidatos WHERE instante AT TIME ZONE p_zona=fecha
$f$;

CREATE FUNCTION vec_dietas_v1.rechazar_mutacion_historia_v1()
RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
BEGIN
    RAISE EXCEPTION 'historia de Dietas inmutable' USING ERRCODE='55000';
END
$f$;

-- En este corte solamente existe la versión 1: esta revisión es tanto el estado
-- recuperable como su historia original. No hay transición ni actualización.
CREATE TABLE vec_dietas_v1.borrador_revision (
    comision_ref text PRIMARY KEY,
    version smallint NOT NULL CHECK (version=1),
    clave_operacion text NOT NULL UNIQUE CHECK (clave_operacion ~ '^[A-Za-z0-9_-]{16,128}$'),
    actor jsonb NOT NULL CHECK (vec_dietas_v1.objeto_exacto_v1(actor,
        ARRAY['actor_ref','perfil_ref','persona_ref','empleado_ref'])),
    estado text NOT NULL CHECK (estado='borrador'),
    material text NOT NULL CHECK (octet_length(material) BETWEEN 1 AND 98304),
    material_sha256 text NOT NULL CHECK (material_sha256=encode(sha256(convert_to(material,'UTF8')),'hex')),
    borrador jsonb NOT NULL CHECK (jsonb_typeof(borrador)='object' AND octet_length(borrador::text)<=196608
        AND octet_length(jsonb_set(borrador,'{ruta}',(borrador->'ruta')-ARRAY['paradas','tramos','trazado'])::text)<=8192),
    recibo_ref text NOT NULL UNIQUE,
    recibo_json jsonb NOT NULL CHECK (jsonb_typeof(recibo_json)='object' AND octet_length(recibo_json::text)<=4096),
    auditoria_ref text NOT NULL UNIQUE,
    decision_ref text NOT NULL UNIQUE,
    consumo_huella_sha256 text NOT NULL UNIQUE CHECK (consumo_huella_sha256 ~ '^[0-9a-f]{64}$'),
    registrada_en timestamptz(6) NOT NULL,
    CHECK (comision_ref='dietas:borrador:'||clave_operacion),
    CHECK (borrador->>'persona_ref'=actor->>'persona_ref'),
    UNIQUE (comision_ref,version)
);
CREATE TABLE vec_dietas_v1.borrador_outbox (
    evento_ref text PRIMARY KEY,
    comision_ref text NOT NULL,
    version smallint NOT NULL CHECK (version=1),
    tipo text NOT NULL CHECK (tipo='dietas.borrador.creado'),
    carga_json jsonb NOT NULL CHECK (jsonb_typeof(carga_json)='object'),
    creada_en timestamptz(6) NOT NULL,
    UNIQUE (comision_ref,version),
    FOREIGN KEY (comision_ref,version) REFERENCES vec_dietas_v1.borrador_revision(comision_ref,version)
);
CREATE TABLE vec_dietas_v1.borrador_acceso (
    acceso_ref text PRIMARY KEY,
    recurso_ref text NOT NULL,
    comision_ref text REFERENCES vec_dietas_v1.borrador_revision(comision_ref),
    actor jsonb NOT NULL CHECK (vec_dietas_v1.objeto_exacto_v1(actor,
        ARRAY['actor_ref','perfil_ref','persona_ref','empleado_ref'])),
    operacion text NOT NULL CHECK (operacion IN ('crear','replay','recuperar','listar')),
    correlacion_ref text NOT NULL,
    auditoria_ref text NOT NULL UNIQUE,
    decision_ref text NOT NULL UNIQUE,
    consumo_huella_sha256 text NOT NULL UNIQUE CHECK (consumo_huella_sha256 ~ '^[0-9a-f]{64}$'),
    registrada_en timestamptz(6) NOT NULL,
    CHECK ((operacion='listar' AND comision_ref IS NULL AND recurso_ref='dietas:borradores:propios')
        OR (operacion<>'listar' AND comision_ref IS NOT NULL AND recurso_ref=comision_ref))
);
CREATE INDEX borrador_propio_pagina ON vec_dietas_v1.borrador_revision(actor,comision_ref COLLATE "C");

DO $seguridad$
DECLARE t text;
BEGIN
    FOREACH t IN ARRAY ARRAY['borrador_revision','borrador_outbox','borrador_acceso'] LOOP
        EXECUTE format('ALTER TABLE vec_dietas_v1.%I ENABLE ROW LEVEL SECURITY',t);
        EXECUTE format('ALTER TABLE vec_dietas_v1.%I FORCE ROW LEVEL SECURITY',t);
        EXECUTE format('CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_dietas_v1.%I FOR EACH ROW EXECUTE FUNCTION vec_dietas_v1.rechazar_mutacion_historia_v1()',t);
        EXECUTE format('CREATE TRIGGER historia_no_truncable BEFORE TRUNCATE ON vec_dietas_v1.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_dietas_v1.rechazar_mutacion_historia_v1()',t);
        EXECUTE format('REVOKE ALL ON TABLE vec_dietas_v1.%I FROM PUBLIC,vec_dietas_v1_ejecutor',t);
    END LOOP;
    FOREACH t IN ARRAY ARRAY['borrador_revision','borrador_acceso'] LOOP
        EXECUTE format('CREATE POLICY propia_lectura ON vec_dietas_v1.%I FOR SELECT TO vec_dietas_v1_propietario USING (actor=nullif(current_setting(''vec.dietas.actor'',true),'''')::jsonb)',t);
        EXECUTE format('CREATE POLICY propia_adicion ON vec_dietas_v1.%I FOR INSERT TO vec_dietas_v1_propietario WITH CHECK (actor=nullif(current_setting(''vec.dietas.actor'',true),'''')::jsonb)',t);
    END LOOP;
END
$seguridad$;
CREATE POLICY propia_lectura ON vec_dietas_v1.borrador_outbox FOR SELECT
    TO vec_dietas_v1_propietario USING (EXISTS (
        SELECT 1 FROM vec_dietas_v1.borrador_revision r
        WHERE r.comision_ref=borrador_outbox.comision_ref AND r.version=borrador_outbox.version));
CREATE POLICY propia_adicion ON vec_dietas_v1.borrador_outbox FOR INSERT
    TO vec_dietas_v1_propietario WITH CHECK (EXISTS (
        SELECT 1 FROM vec_dietas_v1.borrador_revision r
        WHERE r.comision_ref=borrador_outbox.comision_ref AND r.version=borrador_outbox.version));

-- Entrada interna sin EXECUTE para el ejecutor. Las tres funciones públicas
-- fijan la operación; el navegador nunca suministra una decisión ni identidad.
CREATE FUNCTION vec_dietas_v1.operar_borrador_propio_v1(
    p_operacion text,p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,
    p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET timezone='UTC'
SET lock_timeout='2s' SET statement_timeout='15s'
AS $funcion$
DECLARE
    m jsonb; a jsonb; b jsonb; r jsonb; d jsonb; x jsonb; c jsonb; vinculo jsonb;
    v_ref text; v_clave text; v_accion text; v_modo text; v_material_huella text;
    v_contexto_recurso text; v_contexto_huella text; v_actor_previo text;
    v_inicio timestamptz; v_fin timestamptz; v_fecha timestamptz(6); v_vence_en timestamptz;
    v_consumo record; v_previa vec_dietas_v1.borrador_revision%ROWTYPE;
    v_recibo jsonb; v_recibo_ref text; v_listado jsonb:='[]'::jsonb;
    v_limite integer; v_despues text; v_siguiente text:=''; v_cantidad integer:=0;
BEGIN
    IF p_operacion IS NULL OR p_operacion NOT IN ('crear','recuperar','listar')
       OR p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 98304 THEN
        RAISE EXCEPTION 'material de Dietas inválido' USING ERRCODE='PDI00';
    END IF;
    BEGIN
        m:=p_material::jsonb; a:=m->'actor';
        IF NOT vec_dietas_v1.objeto_exacto_v1(a,ARRAY['actor_ref','perfil_ref','persona_ref','empleado_ref'])
           OR EXISTS (SELECT 1 FROM jsonb_each(a) e WHERE jsonb_typeof(e.value) IS DISTINCT FROM 'string')
           OR EXISTS (SELECT 1 FROM jsonb_each_text(a) e WHERE e.value !~ '^[a-z][A-Za-z0-9_:-]{2,180}$') THEN
            RAISE EXCEPTION 'actor de Dietas inválido' USING ERRCODE='PDI00';
        END IF;
        IF p_operacion='crear' THEN
            IF NOT vec_dietas_v1.objeto_exacto_v1(m,ARRAY['actor','clave_operacion','version_esperada','borrador'])
               OR jsonb_typeof(m->'clave_operacion') IS DISTINCT FROM 'string'
               OR m->>'clave_operacion' !~ '^[A-Za-z0-9_-]{16,128}$'
               OR m->'version_esperada' IS DISTINCT FROM '0'::jsonb THEN
                RAISE EXCEPTION 'comando de Dietas inválido' USING ERRCODE='PDI00';
            END IF;
            v_clave:=m->>'clave_operacion'; v_ref:='dietas:borrador:'||v_clave;
            b:=m->'borrador'; r:=b->'ruta';
            IF octet_length(b::text)>196608
               OR octet_length(jsonb_set(b,'{ruta}',(b->'ruta')-ARRAY['paradas','tramos','trazado'])::text)>8192 OR NOT vec_dietas_v1.validar_borrador_completo_v1(b)
               OR b->>'persona_ref' IS DISTINCT FROM a->>'persona_ref' THEN
                RAISE EXCEPTION 'borrador de Dietas inválido' USING ERRCODE='PDI00';
            END IF;
            v_inicio:=(b->>'inicio')::timestamptz; v_fin:=(b->>'fin')::timestamptz;
            IF NOT isfinite(v_inicio) OR NOT isfinite(v_fin) OR v_fin<v_inicio THEN
                RAISE EXCEPTION 'periodo de Dietas inválido' USING ERRCODE='PDI00';
            END IF;
            IF NOT EXISTS (SELECT 1 FROM pg_timezone_names WHERE name=b->>'zona_horaria')
               OR NOT vec_dietas_v1.instante_civil_unico_v1(v_inicio,b->>'fecha',b->>'hora_inicio',b->>'zona_horaria')
               OR NOT vec_dietas_v1.instante_civil_unico_v1(v_fin,b->>'fecha_fin',b->>'hora_fin',b->>'zona_horaria')
               OR to_char(v_inicio AT TIME ZONE (b->>'zona_horaria'),'YYYY-MM-DD HH24:MI:SS.US')
                  IS DISTINCT FROM (b->>'fecha')||' '||(b->>'hora_inicio')||':00.000000'
               OR to_char(v_fin AT TIME ZONE (b->>'zona_horaria'),'YYYY-MM-DD HH24:MI:SS.US')
                  IS DISTINCT FROM (b->>'fecha_fin')||' '||(b->>'hora_fin')||':00.000000' THEN
                RAISE EXCEPTION 'periodo civil de Dietas inválido' USING ERRCODE='PDI00';
            END IF;
        ELSIF p_operacion='recuperar' THEN
            IF NOT vec_dietas_v1.objeto_exacto_v1(m,ARRAY['actor','comision_ref'])
               OR jsonb_typeof(m->'comision_ref') IS DISTINCT FROM 'string'
               OR m->>'comision_ref' !~ '^dietas:borrador:[A-Za-z0-9_-]{16,128}$' THEN
                RAISE EXCEPTION 'consulta de Dietas inválida' USING ERRCODE='PDI00';
            END IF;
            v_ref:=m->>'comision_ref';
        ELSE
            IF NOT vec_dietas_v1.objeto_exacto_v1(m,ARRAY['actor','limite','despues'])
               OR jsonb_typeof(m->'limite') IS DISTINCT FROM 'number'
               OR m->>'limite' !~ '^([1-9]|1[0-9]|20)$'
               OR jsonb_typeof(m->'despues') IS DISTINCT FROM 'string'
               OR (m->>'despues'<>'' AND m->>'despues' !~ '^dietas:borrador:[A-Za-z0-9_-]{16,128}$') THEN
                RAISE EXCEPTION 'paginación de Dietas inválida' USING ERRCODE='PDI00';
            END IF;
            v_ref:='dietas:borradores:propios';
            v_limite:=(m->>'limite')::integer; v_despues:=m->>'despues';
        END IF;
    EXCEPTION WHEN data_exception THEN
        RAISE EXCEPTION 'estructura de Dietas inválida' USING ERRCODE='PDI00';
    END;

    v_accion:=CASE p_operacion WHEN 'crear' THEN 'dietas.borrador.crear_propio'
        WHEN 'listar' THEN 'dietas.borrador.listar_propios'
        ELSE 'dietas.borrador.recuperar_propio' END;
    v_material_huella:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
    v_contexto_recurso:='{"ambitos":{"empleado_ref":'||to_jsonb(a->>'empleado_ref')::text||
        ',"persona_ref":'||to_jsonb(a->>'persona_ref')::text||
        '},"atributos":{"material_sha256":"'||v_material_huella||'"}}';
    v_contexto_huella:=encode(sha256(convert_to(v_contexto_recurso,'UTF8')),'hex');
    IF p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 1 AND 65536
       OR p_contexto IS NULL OR octet_length(p_contexto) NOT BETWEEN 1 AND 65536 THEN
        RAISE EXCEPTION 'autoridad de Dietas inválida' USING ERRCODE='PDI03';
    END IF;
    BEGIN
        d:=convert_from(p_decision,'UTF8')::jsonb;
        x:=convert_from(p_contexto,'UTF8')::jsonb;
        IF d->>'accion' IS DISTINCT FROM v_accion
           OR d->>'modulo_id' IS DISTINCT FROM 'dietas'
           OR d->>'tipo_recurso' IS DISTINCT FROM 'borrador_comision'
           OR d->>'finalidad' IS DISTINCT FROM 'gestionar_borrador_propio'
           OR d->>'recurso_ref' IS DISTINCT FROM v_ref
           OR d->>'principal_id' IS DISTINCT FROM a->>'actor_ref'
           OR d->>'perfil_activo_ref' IS DISTINCT FROM a->>'perfil_ref'
           OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_contexto_huella
           OR coalesce(d->>'correlacion_ref','') !~ '^[a-z][A-Za-z0-9_:-]{2,180}$'
           OR x->>'principal_ref' IS DISTINCT FROM a->>'actor_ref'
           OR x->>'perfil_activo_ref' IS DISTINCT FROM a->>'perfil_ref'
           OR x->>'persona_ref' IS DISTINCT FROM a->>'persona_ref'
           OR a->>'actor_ref' IS DISTINCT FROM a->>'persona_ref'
           OR x->>'esquema' IS DISTINCT FROM 'vec.contexto-actor.vinculado.v2'
           OR jsonb_typeof(x->'vinculos') IS DISTINCT FROM 'array' THEN
            RAISE EXCEPTION 'autoridad de Dietas divergente' USING ERRCODE='PDI03';
        END IF;
        SELECT e.value INTO STRICT vinculo FROM jsonb_array_elements(x->'vinculos') e
        WHERE e.value->>'tipo'='empleado' AND e.value->>'estado'='activo'
          AND (e.value->>'vigente_desde')::timestamptz<=clock_timestamp()
          AND (e.value->>'vigente_hasta')::timestamptz>clock_timestamp();
        IF vinculo->>'referencia' IS DISTINCT FROM a->>'empleado_ref' THEN
            RAISE EXCEPTION 'vínculo empleado divergente' USING ERRCODE='PDI03';
        END IF;
    EXCEPTION WHEN data_exception OR no_data_found OR too_many_rows THEN
        RAISE EXCEPTION 'autoridad de Dietas inválida' USING ERRCODE='PDI03';
    END;
    -- El bloqueo precede la revalidación final central: esperar no permite usar
    -- una concesión que haya vencido mientras otra petición confirmaba.
    PERFORM pg_advisory_xact_lock(hashtextextended('vec_dietas_v1:borrador:'||v_ref,0));
    SELECT * INTO STRICT v_consumo
      FROM vec_autorizacion_atestada_v3.registrar_y_consumir_borrador_dietas_v3_atestada(
        p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
        p_payload,p_sobre,p_evidencia,p_raiz);
    IF v_consumo.consumo_nuevo IS NOT TRUE
       OR v_consumo.efecto_ref IS DISTINCT FROM v_ref
       OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM v_contexto_huella
       OR v_consumo.decision_ref IS DISTINCT FROM d->>'decision_ref' THEN
        RAISE EXCEPTION 'consumo de Dietas divergente' USING ERRCODE='PDI03';
    END IF;
    -- AD3 mantiene sus bloqueos de revocación hasta COMMIT. Puede haber
    -- esperado en su propia auditoría: restringir de nuevo todas las vigencias
    -- con reloj vivo, incluido el vínculo empleado, antes de leer o escribir.
    c:=convert_from(p_capacidad,'UTF8')::jsonb;
    v_fecha:=date_trunc('microseconds',clock_timestamp());
    IF (SELECT count(*) FROM jsonb_array_elements(x->'vinculos') e
        WHERE e.value->>'tipo'='empleado' AND e.value->>'estado'='activo'
          AND (e.value->>'vigente_desde')::timestamptz<=v_fecha
          AND v_fecha<(e.value->>'vigente_hasta')::timestamptz)<>1
       OR (v_fecha<(c->>'expira_en')::timestamptz
           AND v_fecha<(c->>'decision_valida_hasta')::timestamptz
           AND v_fecha<(c->>'configuracion_expira_en')::timestamptz
           AND v_fecha<(c->>'raiz_valida_hasta')::timestamptz
           AND v_fecha<(d->>'valida_hasta')::timestamptz
           AND v_fecha<(x->>'vigente_hasta')::timestamptz
           AND v_fecha<(vinculo->>'vigente_hasta')::timestamptz) IS NOT TRUE THEN
        RAISE EXCEPTION 'vigencia de Dietas agotada' USING ERRCODE='PDI03';
    END IF;
    v_vence_en:=least((c->>'expira_en')::timestamptz,
        (c->>'decision_valida_hasta')::timestamptz,
        (c->>'configuracion_expira_en')::timestamptz,
        (c->>'raiz_valida_hasta')::timestamptz,
        (d->>'valida_hasta')::timestamptz,
        (x->>'vigente_hasta')::timestamptz,
        (vinculo->>'vigente_hasta')::timestamptz);
    v_actor_previo:=nullif(current_setting('vec.dietas.actor',true),'');
    IF v_actor_previo IS NOT NULL AND v_actor_previo::jsonb IS DISTINCT FROM a THEN
        RAISE EXCEPTION 'contexto transaccional de Dietas divergente' USING ERRCODE='PDI03';
    END IF;
    PERFORM set_config('vec.dietas.actor',a::text,true);
    v_fecha:=date_trunc('microseconds',clock_timestamp());
    IF p_operacion='listar' THEN
        FOR v_previa IN SELECT * FROM vec_dietas_v1.borrador_revision
            WHERE actor=a AND comision_ref COLLATE "C">v_despues COLLATE "C"
            ORDER BY comision_ref COLLATE "C" LIMIT v_limite+1 LOOP
            v_cantidad:=v_cantidad+1;
            IF v_cantidad>v_limite THEN
                v_siguiente:=v_listado->(v_limite-1)->'recibo'->>'comision_ref';
                EXIT;
            END IF;
            v_listado:=v_listado||jsonb_build_array(jsonb_build_object(
                'borrador',jsonb_set(v_previa.borrador,'{ruta}',
                    (v_previa.borrador->'ruta')-ARRAY['paradas','tramos','trazado']),
                'recibo',v_previa.recibo_json));
        END LOOP;
        INSERT INTO vec_dietas_v1.borrador_acceso(acceso_ref,recurso_ref,comision_ref,actor,
            operacion,correlacion_ref,auditoria_ref,decision_ref,consumo_huella_sha256,registrada_en)
        VALUES('acceso:dietas:'||gen_random_uuid()::text,v_ref,NULL,a,'listar',d->>'correlacion_ref',
            v_consumo.auditoria_ref,v_consumo.decision_ref,v_consumo.consumo_huella_sha256,v_fecha);
        IF clock_timestamp()>=v_vence_en THEN
            RAISE EXCEPTION 'vigencia de Dietas agotada' USING ERRCODE='PDI03';
        END IF;
        RETURN jsonb_build_object('borradores',v_listado,'siguiente',v_siguiente);
    END IF;
    SELECT * INTO v_previa FROM vec_dietas_v1.borrador_revision WHERE comision_ref=v_ref;
    IF p_operacion='recuperar' THEN
        IF NOT FOUND THEN
            RAISE EXCEPTION 'borrador propio no encontrado' USING ERRCODE='PDI04';
        END IF;
        v_recibo:=v_previa.recibo_json; v_modo:='recuperar';
    ELSIF FOUND THEN
        IF v_previa.actor IS DISTINCT FROM a OR v_previa.borrador IS DISTINCT FROM b THEN
            RAISE EXCEPTION 'clave de Dietas en conflicto' USING ERRCODE='PDI01';
        END IF;
        v_recibo:=v_previa.recibo_json||jsonb_build_object('repeticion',true);
        v_modo:='replay';
    ELSE
        v_recibo_ref:='recibo:dietas:'||gen_random_uuid()::text;
        v_recibo:=jsonb_build_object('comision_ref',v_ref,'recibo_ref',v_recibo_ref,
            'correlacion_ref',d->>'correlacion_ref','version',1,'repeticion',false,
            'registrado_en',to_char(v_fecha AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'));
        BEGIN
            INSERT INTO vec_dietas_v1.borrador_revision(comision_ref,version,clave_operacion,
                actor,estado,material,material_sha256,borrador,recibo_ref,recibo_json,
                auditoria_ref,decision_ref,consumo_huella_sha256,registrada_en)
            VALUES(v_ref,1,v_clave,a,'borrador',p_material,v_material_huella,b,v_recibo_ref,v_recibo,
                v_consumo.auditoria_ref,v_consumo.decision_ref,v_consumo.consumo_huella_sha256,v_fecha);
        EXCEPTION WHEN unique_violation THEN
            -- Una referencia ajena queda invisible por RLS. No revelar su actor.
            RAISE EXCEPTION 'clave de Dietas en conflicto' USING ERRCODE='PDI01';
        END;
        INSERT INTO vec_dietas_v1.borrador_outbox(evento_ref,comision_ref,version,tipo,carga_json,creada_en)
        VALUES('evento:dietas:'||gen_random_uuid()::text,v_ref,1,'dietas.borrador.creado',
            jsonb_build_object('comision_ref',v_ref,'version',1,'recibo_ref',v_recibo_ref),v_fecha);
        v_modo:='crear';
    END IF;
    -- La autoridad central conserva el consumo/auditoría criptográfico. Esta
    -- tabla mínima enlaza cada acceso correcto, incluido replay, con ese hecho.
    -- No acredita auditoría de fallos posteriores a la emisión PDP: si esta
    -- transacción revierte, el registro segregado del fallo sigue pendiente.
    INSERT INTO vec_dietas_v1.borrador_acceso(acceso_ref,recurso_ref,comision_ref,actor,operacion,correlacion_ref,
        auditoria_ref,decision_ref,consumo_huella_sha256,registrada_en)
    VALUES('acceso:dietas:'||gen_random_uuid()::text,v_ref,v_ref,a,v_modo,d->>'correlacion_ref',
        v_consumo.auditoria_ref,v_consumo.decision_ref,v_consumo.consumo_huella_sha256,v_fecha);
    IF clock_timestamp()>=v_vence_en THEN
        RAISE EXCEPTION 'vigencia de Dietas agotada' USING ERRCODE='PDI03';
    END IF;
    IF p_operacion='recuperar' THEN
        RETURN jsonb_build_object('borrador',v_previa.borrador,'recibo',v_recibo);
    END IF;
    RETURN v_recibo;
END
$funcion$;

CREATE FUNCTION vec_dietas_v1.crear_borrador_propio_v1(
    p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,
    p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
    SELECT vec_dietas_v1.operar_borrador_propio_v1('crear',p_material,p_capacidad,p_decision,
        p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz)
$f$;
CREATE FUNCTION vec_dietas_v1.recuperar_borrador_propio_v1(
    p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,
    p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
    SELECT vec_dietas_v1.operar_borrador_propio_v1('recuperar',p_material,p_capacidad,p_decision,
        p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz)
$f$;
CREATE FUNCTION vec_dietas_v1.listar_borradores_propios_v1(
    p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,
    p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
    SELECT vec_dietas_v1.operar_borrador_propio_v1('listar',p_material,p_capacidad,p_decision,
        p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz)
$f$;
REVOKE ALL ON ALL FUNCTIONS IN SCHEMA vec_dietas_v1 FROM PUBLIC,vec_dietas_v1_ejecutor;
GRANT EXECUTE ON FUNCTION
    vec_dietas_v1.crear_borrador_propio_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
    vec_dietas_v1.recuperar_borrador_propio_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
    vec_dietas_v1.listar_borradores_propios_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
    TO vec_dietas_v1_ejecutor;
COMMIT;
