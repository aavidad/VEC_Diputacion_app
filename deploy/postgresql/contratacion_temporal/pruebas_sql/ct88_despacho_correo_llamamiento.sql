\set ON_ERROR_STOP on
-- Focal de catálogo y formato puro, para instalación candidata ya revisada.
-- No genera correo/capacidad ni afirma prueba de atomicidad, concurrencia o SMTP.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
DO $contrato$
DECLARE
    nombre text; t oid; r record; f oid; privilegio record;
    propietario oid := 'vec_contratacion_temporal_propietario'::regrole;
    ejecutor oid := 'vec_contratacion_temporal_ejecutor'::regrole;
BEGIN
    FOREACH nombre IN ARRAY ARRAY['despacho_correo_llamamiento_intento_v1',
        'despacho_correo_llamamiento_resultado_v1','despacho_correo_llamamiento_outbox_v1'] LOOP
        t := to_regclass('vec_contratacion_temporal.'||nombre);
        IF t IS NULL OR NOT EXISTS (SELECT 1 FROM pg_class WHERE oid=t
            AND relowner=propietario AND relrowsecurity AND relforcerowsecurity) THEN
            RAISE EXCEPTION 'CT88: tabla/propietario/RLS incorrectos';
        END IF;
        IF EXISTS (SELECT 1 FROM pg_class c,
            LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) a
            WHERE c.oid=t AND a.grantee<>propietario) THEN
            RAISE EXCEPTION 'CT88: privilegio directo de tabla inesperado';
        END IF;
        IF EXISTS (SELECT 1 FROM pg_class c JOIN pg_type ty ON ty.oid=c.reltype,
            LATERAL aclexplode(coalesce(ty.typacl,acldefault('T',ty.typowner))) a
            WHERE c.oid=t AND a.grantee<>propietario) THEN
            RAISE EXCEPTION 'CT88: privilegio de tipo de fila inesperado';
        END IF;
        IF EXISTS (SELECT 1 FROM pg_attribute WHERE attrelid=t AND NOT attisdropped
            AND attname IN ('destino','correo','asunto','cuerpo','capacidad_finalizacion',
                            'token_finalizacion','secreto','respuesta_smtp')) THEN
            RAISE EXCEPTION 'CT88: datos no minimizados';
        END IF;
        IF nombre<>'despacho_correo_llamamiento_intento_v1'
           AND NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgrelid=t AND NOT tgisinternal
               AND tgname='historia_inmutable' AND tgenabled='O' AND tgtype=58
               AND tgfoid='vec_contratacion_temporal.rechazar_mutacion_historia_v1()'::regprocedure) THEN
            RAISE EXCEPTION 'CT88: historia sin protección UPDATE/DELETE/TRUNCATE';
        END IF;
    END LOOP;
    FOREACH nombre IN ARRAY ARRAY[
        'reservar_intento_despacho_correo_llamamiento_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
        'registrar_resultado_intento_despacho_correo_llamamiento_v1(text,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'] LOOP
        f := to_regprocedure('vec_contratacion_temporal.'||nombre);
        IF f IS NULL OR NOT EXISTS (SELECT 1 FROM pg_proc WHERE oid=f
            AND proowner=propietario AND prosecdef AND provolatile='v'
            AND prorettype='jsonb'::regtype AND pronargdefaults=0
            AND cardinality(proconfig)=4 AND proconfig[1]='search_path=pg_catalog'
            AND proconfig[2]='row_security=on' AND lower(proconfig[3])='timezone=utc'
            AND proconfig[4]='lock_timeout=2s') THEN
            RAISE EXCEPTION 'CT88: contrato de función/owner/config divergente';
        END IF;
        IF NOT COALESCE((SELECT count(*)=2 AND count(DISTINCT a.grantee)=2
            AND bool_and(a.grantee IN (propietario,ejecutor) AND a.grantor=propietario
                         AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
            FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
            WHERE p.oid=f),false) THEN
            RAISE EXCEPTION 'CT88: ACL ejecutor/propietario no exacta';
        END IF;
    END LOOP;
    IF to_regprocedure('vec_contratacion_temporal.registrar_resultado_intento_despacho_correo_llamamiento_v1(text,text,bytea,text,text)') IS NOT NULL
       OR to_regprocedure('vec_contratacion_temporal.registrar_resultado_intento_despacho_correo_llamamiento_v1(text,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
       OR to_regprocedure('vec_contratacion_temporal.registrar_resultado_intento_despacho_correo_llamamiento_v1(text,text,text)') IS NOT NULL THEN
        RAISE EXCEPTION 'CT88: firma de finalización anterior conservada';
    END IF;
    IF EXISTS (SELECT 1 FROM pg_proc WHERE oid=
        'vec_contratacion_temporal.registrar_resultado_intento_despacho_correo_llamamiento_v1(text,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
        AND strpos(prosrc,'CT88: integración de auditoría común pendiente')>0) THEN
        RAISE EXCEPTION 'CT88: candidato incompleto, autoridad común pendiente';
    END IF;
    f := 'vec_contratacion_temporal.correo88_validar_resultado_v1(text)'::regprocedure;
    IF NOT EXISTS (SELECT 1 FROM pg_proc WHERE oid=f AND proowner=propietario
        AND NOT prosecdef AND provolatile='i' AND proconfig=ARRAY['search_path=pg_catalog'])
       OR EXISTS (SELECT 1 FROM pg_proc p,
            LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
            WHERE p.oid=f AND a.grantee<>propietario) THEN
        RAISE EXCEPTION 'CT88: validador puro expuesto o divergente';
    END IF;
    -- La FK compuesta liga el evento con el hecho original completo, no sólo el intento.
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE contype='f' AND convalidated
        AND conrelid='vec_contratacion_temporal.despacho_correo_llamamiento_outbox_v1'::regclass
        AND confrelid='vec_contratacion_temporal.despacho_correo_llamamiento_resultado_v1'::regclass
        AND cardinality(conkey)=7 AND cardinality(confkey)=7) THEN
        RAISE EXCEPTION 'CT88: vínculo completo resultado/outbox ausente';
    END IF;
END
$contrato$;

DO $formato$
DECLARE
    canonico text := $json${"OrganizacionRef":"organizacion:prueba","ExpedienteRef":"expediente:prueba","LlamamientoRef":"llamamiento:prueba","ComunicacionRef":"comunicacion:prueba","IntencionEnvioRef":"outbox:prueba","IntentoRef":"intento-correo:prueba","SolicitudHuella":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","Estado":"aceptado_por_relay","PlantillaRef":"llamamiento_rrhh_v1","VersionEsperada":1}$json$;
    entrada text; estado text; esperado text; denegado boolean;
BEGIN
    IF encode(sha256(convert_to(canonico,'UTF8')),'hex') <> 'c5891f8d86ba7fd0382918af240e3f0a87d2fe8d88b4cc0c1b6da98f2c57165a' THEN
        RAISE EXCEPTION 'CT88: vector canónico divergente';
    END IF;
    FOREACH estado IN ARRAY ARRAY['no_aceptado_transitorio','no_aceptado_permanente',
                                  'indeterminado','aceptado_por_relay'] LOOP
        esperado := replace(canonico,'aceptado_por_relay',estado);
        IF vec_contratacion_temporal.correo88_validar_resultado_v1(esperado) IS DISTINCT FROM esperado::jsonb THEN
            RAISE EXCEPTION 'CT88: resultado canónico rechazado';
        END IF;
    END LOOP;
    FOREACH entrada IN ARRAY ARRAY[
        NULL::text,'[]','{}','{',repeat('x',16385),
        ' '||canonico,
        replace(canonico,'"VersionEsperada":1','"VersionEsperada":1.0'),
        replace(canonico,'"VersionEsperada":1','"VersionEsperada":2'),
        replace(canonico,'"VersionEsperada":1','"VersionEsperada":"1"'),
        replace(canonico,'"Estado":"aceptado_por_relay"','"Estado":null'),
        replace(canonico,'aceptado_por_relay','iniciado'),
        replace(canonico,'llamamiento_rrhh_v1','otra_plantilla'),
        replace(canonico,repeat('a',64),repeat('0',64)),
        replace(canonico,'organizacion:prueba','organizacion:prueba@invalida'),
        replace(canonico,'"IntentoRef":"intento-correo:prueba"','"IntentoRef":false'),
        replace(canonico,'"VersionEsperada":1}','"VersionEsperada":1,"extra":"x"}'),
        replace(canonico,'"VersionEsperada":1}','"VersionEsperada":1,"VersionEsperada":1}'),
        replace(canonico,'"IntentoRef":"intento-correo:prueba",',''),
        canonico::jsonb::text
    ] LOOP
        denegado := false;
        BEGIN
            PERFORM vec_contratacion_temporal.correo88_validar_resultado_v1(entrada);
        EXCEPTION WHEN SQLSTATE '22023' THEN denegado := true;
        END;
        IF NOT denegado THEN RAISE EXCEPTION 'CT88: material no canónico admitido'; END IF;
    END LOOP;
END
$formato$;
ROLLBACK;
-- Puerta dinámica adicional obligatoria: autorizaciones V3 reales de reserva y
-- resultado; cuatro estados; secreto cruzado/cero y material divergente;
-- revocación/expiración; replay nuevo V3; rollback por fallo de auditoría/outbox;
-- finalizadores concurrentes y COMMIT incierto sin reenvío. Este focal no los acredita.
