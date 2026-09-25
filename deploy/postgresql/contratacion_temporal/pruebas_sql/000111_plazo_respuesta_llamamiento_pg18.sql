\set ON_ERROR_STOP on
-- Pruebas CT111 sobre la preimagen desechable (000111_preimagen_pg18.sql y
-- migraciones CT reales). Los consumidores AD3 son dobles: se prueba la
-- ligadura del material, la lista de políticas, el plazo y la expiración.
-- Datos sintéticos; ninguna referencia identifica a una persona.
DO $$ BEGIN
    IF to_regclass('vec_contratacion_temporal.evento_plazo_llamamiento_rrhh') IS NULL THEN
        RAISE EXCEPTION 'CT111 no instalada';
    END IF;
END $$;
CREATE SCHEMA prueba_ct111;
GRANT USAGE ON SCHEMA prueba_ct111 TO vec_ct111_runtime, vec_ct111_ajeno;

-- Decisión de doble AD3 ligada al material exacto, como la emite la frontera.
CREATE FUNCTION prueba_ct111.decision(p_material text, p_denegar boolean DEFAULT false,
    p_actor text DEFAULT 'actor:rrhh:uno') RETURNS bytea
LANGUAGE sql STABLE SET search_path = pg_catalog AS $f$
    SELECT convert_to(jsonb_build_object(
        'accion','contratacion_temporal.llamamiento.respuesta.validacion_manual.registrar',
        'modulo_id','contratacion_temporal','tipo_recurso','resolucion_manual_respuesta_ct',
        'finalidad','gestionar_contratacion_temporal',
        'recurso_ref',(p_material::jsonb)->'Solicitud'->>'ExpedienteRef',
        'contexto_recurso_huella_sha256',encode(sha256(convert_to(
            '{"ambitos":{"organizacion_ref":"'||((p_material::jsonb)->'Solicitud'->>'OrganizacionRef')||
            '"},"atributos":{"material_sha256":"'||encode(sha256(convert_to(p_material,'UTF8')),'hex')||'"}}','UTF8')),'hex'),
        'principal_id',p_actor,'perfil_activo_ref','perfil:rrhh:gestion',
        'denegar',CASE WHEN p_denegar THEN 'si' ELSE 'no' END)::text,'UTF8')
$f$;
CREATE FUNCTION prueba_ct111.plazo(p_material text, p_denegar boolean DEFAULT false,
    p_actor text DEFAULT 'actor:rrhh:uno') RETURNS jsonb
LANGUAGE sql VOLATILE SET search_path = pg_catalog AS $f$
    SELECT vec_contratacion_temporal.registrar_evento_plazo_llamamiento_v1(p_material,
        '\x01'::bytea,prueba_ct111.decision(p_material,p_denegar,p_actor),'\x01'::bytea,'\x01'::bytea,1,1,
        '\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea)
$f$;
CREATE FUNCTION prueba_ct111.resolver(p_material text) RETURNS jsonb
LANGUAGE sql VOLATILE SET search_path = pg_catalog AS $f$
    SELECT vec_contratacion_temporal.registrar_resolucion_manual_respuesta_rrhh_v1(p_material,
        '\x01'::bytea,prueba_ct111.decision(p_material),'\x01'::bytea,'\x01'::bytea,1,1,
        '\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea)
$f$;
-- Ejecuta una sentencia y exige un SQLSTATE exacto.
CREATE FUNCTION prueba_ct111.esperar(p_sql text, p_estado text, p_caso text) RETURNS void
LANGUAGE plpgsql SET search_path = pg_catalog AS $f$
BEGIN
    BEGIN
        EXECUTE p_sql;
    EXCEPTION WHEN OTHERS THEN
        IF SQLSTATE<>p_estado THEN
            RAISE EXCEPTION 'caso %: se esperaba % y se obtuvo % (%)',p_caso,p_estado,SQLSTATE,SQLERRM;
        END IF;
        RAISE NOTICE 'rechazo verificado: % (%)',p_caso,p_estado;
        RETURN;
    END;
    RAISE EXCEPTION 'caso %: se esperaba rechazo %',p_caso,p_estado;
END
$f$;
CREATE FUNCTION prueba_ct111.exigir(p_condicion boolean, p_caso text) RETURNS void
LANGUAGE plpgsql SET search_path = pg_catalog AS $f$
BEGIN
    IF p_condicion IS NOT TRUE THEN
        RAISE EXCEPTION 'caso fallido: %',p_caso;
    END IF;
    RAISE NOTICE 'comprobado: %',p_caso;
END
$f$;
-- Materiales en el mismo orden que json.Marshal de los puertos Go.
CREATE FUNCTION prueba_ct111.solicitud_plazo(p_n int, p_clave uuid, p_tipo text, p_instante timestamptz,
    p_prueba text DEFAULT 'prueba:contacto:llamada') RETURNS text
LANGUAGE sql IMMUTABLE SET search_path = pg_catalog AS $f$
    SELECT format('{"ClaveIdempotencia":"%s","OrganizacionRef":"organizacion:ct111","ExpedienteRef":"expediente:ct111:%s","LlamamientoRef":"llamamiento:ct111:%s","ComunicacionRef":"comunicacion:ct111:%s","VersionComunicacionEsperada":2,"Tipo":"%s","InstanteEn":"%s","PruebaRef":"%s"}',
        p_clave,p_n,p_n,p_n,p_tipo,to_char(p_instante AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),p_prueba)
$f$;
CREATE FUNCTION prueba_ct111.referencia(p_entrada text, p_version int DEFAULT 3) RETURNS text
LANGUAGE sql IMMUTABLE SET search_path = pg_catalog AS $f$
    SELECT format('{"Referencia":"vec.bolsa.reglas:%s:%s","Version":%s,"HuellaSHA256":"%s"}',
        p_version,p_entrada,p_version,repeat('ab',32))
$f$;
CREATE FUNCTION prueba_ct111.material_contacto(p_n int, p_clave uuid, p_instante timestamptz,
    p_hasta timestamptz, p_tratamiento text DEFAULT 'exige_causa_justificada',
    p_politica text DEFAULT 'b05.plazo_respuesta') RETURNS text
LANGUAGE sql IMMUTABLE SET search_path = pg_catalog AS $f$
    SELECT format('{"Solicitud":%s,"Plazo":{"RespuestaHasta":"%s","UltimoDia":"%s","Politica":%s,"TratamientoFueraDePlazo":"%s","ConfirmacionExpiracion":"rrhh","CriterioRespuesta":%s,"CriterioExpiracion":%s,"ReglaEjemplo":true}}',
        prueba_ct111.solicitud_plazo(p_n,p_clave,'contacto_efectivo',p_instante),
        to_char(p_hasta AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
        to_char((p_hasta AT TIME ZONE 'Europe/Madrid')::date,'YYYY-MM-DD'),
        prueba_ct111.referencia(p_politica),p_tratamiento,
        prueba_ct111.referencia('b07.fuera_de_plazo'),prueba_ct111.referencia('b08.sin_respuesta_baja'))
$f$;
CREATE FUNCTION prueba_ct111.material_resolucion(p_n int, p_clave uuid, p_respuesta text,
    p_politica text) RETURNS text
LANGUAGE sql IMMUTABLE SET search_path = pg_catalog AS $f$
    SELECT format('{"Solicitud":{"ClaveIdempotencia":"%s","OrganizacionRef":"organizacion:ct111","ExpedienteRef":"expediente:ct111:%s","LlamamientoRef":"llamamiento:ct111:%s","ComunicacionRef":"comunicacion:ct111:%s","VersionEsperada":2,"Respuesta":"%s","PruebaRespuestaRef":"%s","RevisionRespuestaRRHH":true,"RevisionPlazoRRHH":true,"CriterioValidacionRef":"%s"},"Politica":%s}',
        p_clave,p_n,p_n,p_n,p_respuesta,
        CASE WHEN p_respuesta='expiracion_gobernada' THEN '' ELSE 'justificante:ct111:'||p_n END,
        (p_politica::jsonb)->>'Referencia',p_politica)
$f$;
CREATE FUNCTION prueba_ct111.historica() RETURNS text
LANGUAGE sql IMMUTABLE SET search_path = pg_catalog AS $f$
    SELECT '{"Referencia":"politica:ct:revision-manual-sintetica:20260906","Version":1,"HuellaSHA256":"ea41d65808044fa75b597855e81a469ed274403a521890bafa07c33ae89ec2e3"}'
$f$;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA prueba_ct111 TO vec_ct111_runtime, vec_ct111_ajeno;

-- Antecedentes sintéticos: ocho llamamientos con selección confirmada y aviso
-- local registrado hace tres días. Solo los 4 a 6 tienen respuesta declarada.
INSERT INTO vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
SELECT ('00000000-0000-4000-8000-0000000000'||lpad(n::text,2,'0'))::uuid,'confirmada',
       jsonb_build_object('organizacion_ref','organizacion:ct111','expediente_ref','expediente:ct111:'||n),
       jsonb_build_object('organizacion_ref','organizacion:ct111','expediente_ref','expediente:ct111:'||n,
           'llamamiento_ref','llamamiento:ct111:'||n,'recibo_ref','recibo:seleccion:ct111:'||n,'propuesta_generada',true)
  FROM generate_series(1,8) n;
INSERT INTO vec_contratacion_temporal.comunicacion_llamamiento_local
SELECT 'comunicacion:ct111:'||n,'organizacion:ct111','expediente:ct111:'||n,'llamamiento:ct111:'||n,
       gen_random_uuid(),('00000000-0000-4000-8000-0000000000'||lpad(n::text,2,'0'))::uuid,
       'actor:rrhh:uno','perfil:rrhh:gestion',1,2,
       jsonb_build_object('solicitud',jsonb_build_object('PruebaEntregaRef','recibo:seleccion:ct111:'||n)),
       '{}'::jsonb,'registrada_localmente',clock_timestamp()-interval '3 days'
  FROM generate_series(1,8) n;

-- 1. Contactos efectivos y sus rechazos.
SET SESSION AUTHORIZATION vec_ct111_runtime;
CREATE TEMP TABLE resultado (caso text PRIMARY KEY, valor jsonb NOT NULL);
BEGIN ISOLATION LEVEL SERIALIZABLE;
-- Llamamiento 1: contacto hace dos días, vencido hace uno.
INSERT INTO resultado SELECT 'contacto1', prueba_ct111.plazo(prueba_ct111.material_contacto(1,
    '11111111-1111-4111-8111-000000000001',clock_timestamp()-interval '2 days',clock_timestamp()-interval '1 day'));
COMMIT;
BEGIN ISOLATION LEVEL SERIALIZABLE;
INSERT INTO resultado SELECT 'contacto1-replay', prueba_ct111.plazo(prueba_ct111.material_contacto(1,
    '11111111-1111-4111-8111-000000000001',
    ((SELECT valor FROM resultado WHERE caso='contacto1')->'Solicitud'->>'InstanteEn')::timestamptz,
    clock_timestamp()+interval '9 days'));
SELECT prueba_ct111.exigir(
    (SELECT valor->>'Estado' FROM resultado WHERE caso='contacto1')='registrado'
    AND (SELECT valor->>'Estado' FROM resultado WHERE caso='contacto1-replay')='replay_registrado'
    AND (SELECT valor->>'EventoRef' FROM resultado WHERE caso='contacto1')
        =(SELECT valor->>'EventoRef' FROM resultado WHERE caso='contacto1-replay')
    AND (SELECT valor->'Plazo' FROM resultado WHERE caso='contacto1')
        =(SELECT valor->'Plazo' FROM resultado WHERE caso='contacto1-replay'),
    'replay devuelve el recibo y el vencimiento originales');
COMMIT;
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT prueba_ct111.esperar($s$SELECT prueba_ct111.plazo(prueba_ct111.material_contacto(1,
    '11111111-1111-4111-8111-000000000001',clock_timestamp()-interval '1 day',clock_timestamp()+interval '1 day'))$s$,
    'P0591','misma clave con otra solicitud');
SELECT prueba_ct111.esperar($s$SELECT prueba_ct111.plazo(prueba_ct111.material_contacto(1,
    '11111111-1111-4111-8111-000000000002',clock_timestamp()-interval '1 day',clock_timestamp()+interval '1 day'))$s$,
    'P0592','segundo contacto de la misma comunicación');
SELECT prueba_ct111.esperar($s$SELECT prueba_ct111.plazo(prueba_ct111.material_contacto(2,
    '11111111-1111-4111-8111-000000000003',clock_timestamp()-interval '1 day',clock_timestamp()+interval '1 day',
    'exige_causa_justificada','b99.inventada'))$s$,'P0590','política de plazo no admitida');
SELECT prueba_ct111.esperar($s$SELECT prueba_ct111.plazo(prueba_ct111.material_contacto(2,
    '11111111-1111-4111-8111-000000000004',clock_timestamp()-interval '1 day',clock_timestamp()+interval '1 day',
    'exige_causa_justificada','b07.fuera_de_plazo'))$s$,'P0590','entrada admitida para otro uso');
SELECT prueba_ct111.esperar($s$SELECT prueba_ct111.plazo(prueba_ct111.material_contacto(2,
    '11111111-1111-4111-8111-000000000005',clock_timestamp()+interval '1 hour',clock_timestamp()+interval '2 days'))$s$,
    'P0590','contacto en el futuro');
SELECT prueba_ct111.esperar($s$SELECT prueba_ct111.plazo(prueba_ct111.material_contacto(2,
    '11111111-1111-4111-8111-000000000006',clock_timestamp()-interval '4 days',clock_timestamp()+interval '2 days'))$s$,
    'P0590','contacto anterior al aviso');
SELECT prueba_ct111.esperar($s$SELECT prueba_ct111.plazo(prueba_ct111.material_contacto(2,
    '11111111-1111-4111-8111-000000000007',clock_timestamp()-interval '1 day',clock_timestamp()-interval '2 days'))$s$,
    'P0590','vencimiento anterior al contacto');
SELECT prueba_ct111.esperar($s$SELECT prueba_ct111.plazo(prueba_ct111.material_contacto(2,
    '11111111-1111-4111-8111-000000000008',clock_timestamp()-interval '1 day',clock_timestamp()+interval '1 day',
    'inventado'))$s$,'P0590','tratamiento fuera de la lista técnica');
SELECT prueba_ct111.esperar($s$SELECT prueba_ct111.plazo(prueba_ct111.material_contacto(2,
    '11111111-1111-4111-8111-000000000009',clock_timestamp()-interval '1 day',clock_timestamp()+interval '1 day'),true)$s$,
    'P0593','autorización denegada');
SELECT prueba_ct111.esperar($s$SELECT prueba_ct111.plazo(replace(prueba_ct111.material_contacto(2,
    '11111111-1111-4111-8111-000000000010',clock_timestamp()-interval '1 day',clock_timestamp()+interval '1 day'),
    '"ReglaEjemplo":true','"ReglaEjemplo":true,"Extra":1'))$s$,'P0590','campo añadido al plazo');
SELECT prueba_ct111.esperar($s$SELECT prueba_ct111.plazo(prueba_ct111.solicitud_plazo(2,
    '11111111-1111-4111-8111-000000000011','causa_justificada',clock_timestamp()-interval '1 hour'))$s$,
    'P0590','causa sin envoltorio de material');
SELECT prueba_ct111.esperar($s$SELECT prueba_ct111.plazo('{"Solicitud":'||prueba_ct111.solicitud_plazo(2,
    '11111111-1111-4111-8111-000000000012','causa_justificada',clock_timestamp()-interval '1 hour')||'}')$s$,
    'P0592','causa sin contacto efectivo');
COMMIT;
BEGIN ISOLATION LEVEL READ COMMITTED;
SELECT prueba_ct111.esperar($s$SELECT prueba_ct111.plazo(prueba_ct111.material_contacto(2,
    '11111111-1111-4111-8111-000000000013',clock_timestamp()-interval '1 day',clock_timestamp()+interval '1 day'))$s$,
    'P0593','aislamiento no serializable');
COMMIT;
-- Llamamientos 2 (en plazo), 3, 4, 5 (vencidos) y 7 (admitir).
BEGIN ISOLATION LEVEL SERIALIZABLE;
INSERT INTO resultado SELECT 'contacto2', prueba_ct111.plazo(prueba_ct111.material_contacto(2,
    '11111111-1111-4111-8111-000000000020',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 day'));
INSERT INTO resultado SELECT 'contacto3', prueba_ct111.plazo(prueba_ct111.material_contacto(3,
    '11111111-1111-4111-8111-000000000030',clock_timestamp()-interval '2 days',clock_timestamp()-interval '1 day'));
INSERT INTO resultado SELECT 'contacto4', prueba_ct111.plazo(prueba_ct111.material_contacto(4,
    '11111111-1111-4111-8111-000000000040',clock_timestamp()-interval '2 days',clock_timestamp()-interval '1 day'));
INSERT INTO resultado SELECT 'contacto5', prueba_ct111.plazo(prueba_ct111.material_contacto(5,
    '11111111-1111-4111-8111-000000000050',clock_timestamp()-interval '2 days',clock_timestamp()-interval '1 day',
    'no_admitir'));
INSERT INTO resultado SELECT 'contacto7', prueba_ct111.plazo(prueba_ct111.material_contacto(7,
    '11111111-1111-4111-8111-000000000070',clock_timestamp()-interval '2 days',clock_timestamp()-interval '1 day',
    'admitir'));
COMMIT;
RESET SESSION AUTHORIZATION;

-- Declaraciones de respuesta CT56 (sintéticas): 3 dentro de plazo, 4 y 5 y 7
-- fuera de plazo, 6 sin contacto efectivo (circuito histórico).
INSERT INTO vec_contratacion_temporal.respuesta_recibida_rrhh
SELECT 'justificante:ct111:'||n,'organizacion:ct111','expediente:ct111:'||n,'llamamiento:ct111:'||n,
       'comunicacion:ct111:'||n,('00000000-0000-4000-8000-0000000000'||lpad(n::text,2,'0'))::uuid,gen_random_uuid(),
       'actor:rrhh:uno','perfil:rrhh:gestion',2,'aceptacion','correo:ct111:'||n,repeat('cd',32),
       rec,mat,mat::jsonb,encode(sha256(convert_to(mat,'UTF8')),'hex'),'recibo:respuesta:ct111:'||n,
       jsonb_build_object('JustificanteRef','justificante:ct111:'||n,'ReciboRef','recibo:respuesta:ct111:'||n,
           'Solicitud',mat::jsonb,'Estado','registrada_por_rrhh'),'registrada_por_rrhh',rec+interval '1 minute'
  FROM (SELECT n, CASE WHEN n=3 THEN clock_timestamp()-interval '40 hours'
                       ELSE clock_timestamp()-interval '12 hours' END AS rec,
               format('{"n":%s}',n) AS mat
          FROM unnest(ARRAY[3,4,5,6,7]) n) x;

-- 2. Resoluciones.
SET SESSION AUTHORIZATION vec_ct111_runtime;
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT prueba_ct111.esperar($s$SELECT prueba_ct111.resolver(prueba_ct111.material_resolucion(2,
    '22222222-2222-4222-8222-000000000002','expiracion_gobernada',prueba_ct111.referencia('b08.sin_respuesta_baja')))$s$,
    'P0586','expiración antes del vencimiento');
SELECT prueba_ct111.esperar($s$SELECT prueba_ct111.resolver(prueba_ct111.material_resolucion(3,
    '22222222-2222-4222-8222-000000000003','expiracion_gobernada',prueba_ct111.referencia('b08.sin_respuesta_baja')))$s$,
    'P0582','expiración con respuesta registrada');
SELECT prueba_ct111.esperar($s$SELECT prueba_ct111.resolver(prueba_ct111.material_resolucion(8,
    '22222222-2222-4222-8222-000000000008','expiracion_gobernada',prueba_ct111.referencia('b08.sin_respuesta_baja')))$s$,
    'P0582','expiración sin contacto efectivo');
SELECT prueba_ct111.esperar($s$SELECT prueba_ct111.resolver(prueba_ct111.material_resolucion(1,
    '22222222-2222-4222-8222-000000000011','expiracion_gobernada',prueba_ct111.historica()))$s$,
    'P0580','política histórica no admitida para expirar');
SELECT prueba_ct111.esperar($s$SELECT prueba_ct111.resolver(prueba_ct111.material_resolucion(1,
    '22222222-2222-4222-8222-000000000012','expiracion_gobernada',prueba_ct111.referencia('b07.fuera_de_plazo')))$s$,
    'P0580','criterio de respuesta usado para expirar');
SELECT prueba_ct111.esperar($s$SELECT prueba_ct111.resolver(replace(prueba_ct111.material_resolucion(1,
    '22222222-2222-4222-8222-000000000013','expiracion_gobernada',prueba_ct111.referencia('b08.sin_respuesta_baja')),
    '"PruebaRespuestaRef":""','"PruebaRespuestaRef":"justificante:ct111:1"'))$s$,
    'P0580','expiración con justificante');
SELECT prueba_ct111.esperar($s$SELECT prueba_ct111.resolver(prueba_ct111.material_resolucion(4,
    '22222222-2222-4222-8222-000000000004','aceptacion',prueba_ct111.referencia('b07.fuera_de_plazo')))$s$,
    'P0585','respuesta fuera de plazo sin causa acreditada');
SELECT prueba_ct111.esperar($s$SELECT prueba_ct111.resolver(prueba_ct111.material_resolucion(5,
    '22222222-2222-4222-8222-000000000005','aceptacion',prueba_ct111.referencia('b07.fuera_de_plazo')))$s$,
    'P0585','respuesta fuera de plazo no admitida');
SELECT prueba_ct111.esperar($s$SELECT prueba_ct111.resolver(prueba_ct111.material_resolucion(6,
    '22222222-2222-4222-8222-000000000006','aceptacion',prueba_ct111.referencia('b07.fuera_de_plazo')))$s$,
    'P0582','política de catálogo sin contacto efectivo');
SELECT prueba_ct111.esperar($s$SELECT prueba_ct111.resolver(prueba_ct111.material_resolucion(6,
    '22222222-2222-4222-8222-000000000016','aceptacion',
    replace(prueba_ct111.historica(),'"Version":1','"Version":2')))$s$,
    'P0580','política histórica con otra versión');
COMMIT;
BEGIN ISOLATION LEVEL SERIALIZABLE;
INSERT INTO resultado SELECT 'expiracion1', prueba_ct111.resolver(prueba_ct111.material_resolucion(1,
    '22222222-2222-4222-8222-000000000001','expiracion_gobernada',prueba_ct111.referencia('b08.sin_respuesta_baja')));
INSERT INTO resultado SELECT 'aceptacion3', prueba_ct111.resolver(prueba_ct111.material_resolucion(3,
    '22222222-2222-4222-8222-000000000030','aceptacion',prueba_ct111.referencia('b07.fuera_de_plazo')));
INSERT INTO resultado SELECT 'historica6', prueba_ct111.resolver(prueba_ct111.material_resolucion(6,
    '22222222-2222-4222-8222-000000000060','aceptacion',prueba_ct111.historica()));
INSERT INTO resultado SELECT 'admitida7', prueba_ct111.resolver(prueba_ct111.material_resolucion(7,
    '22222222-2222-4222-8222-000000000070','aceptacion',prueba_ct111.referencia('b07.fuera_de_plazo')));
INSERT INTO resultado SELECT 'causa4', prueba_ct111.plazo('{"Solicitud":'||prueba_ct111.solicitud_plazo(4,
    '11111111-1111-4111-8111-000000000041','causa_justificada',clock_timestamp()-interval '1 hour',
    'prueba:causa:acreditacion')||'}');
COMMIT;
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT prueba_ct111.esperar($s$SELECT prueba_ct111.plazo('{"Solicitud":'||prueba_ct111.solicitud_plazo(5,
    '11111111-1111-4111-8111-000000000051','causa_justificada',clock_timestamp()-interval '1 hour')||'}')$s$,
    'P0592','causa en un plazo que no la admite');
SELECT prueba_ct111.esperar($s$SELECT prueba_ct111.plazo(prueba_ct111.material_contacto(1,
    '11111111-1111-4111-8111-000000000014',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 day'))$s$,
    'P0592','contacto sobre comunicación ya resuelta');
INSERT INTO resultado SELECT 'aceptacion4', prueba_ct111.resolver(prueba_ct111.material_resolucion(4,
    '22222222-2222-4222-8222-000000000040','aceptacion',prueba_ct111.referencia('b07.fuera_de_plazo')));
INSERT INTO resultado SELECT 'expiracion1-replay', prueba_ct111.resolver(prueba_ct111.material_resolucion(1,
    '22222222-2222-4222-8222-000000000001','expiracion_gobernada',prueba_ct111.referencia('b08.sin_respuesta_baja')));
COMMIT;
SELECT prueba_ct111.exigir((SELECT valor->>'EstadoPlazo'='expirado' AND valor->>'Estado'='confirmado'
    AND valor->'IntencionSiguiente'->>'Estado'='pendiente' AND valor ? 'RespuestaHasta'
    AND valor->'RespuestaFueraDePlazo'='false'::jsonb AND NOT valor ? 'CausaJustificadaRef'
    FROM resultado WHERE caso='expiracion1'),'expiración confirmada con intención de siguiente pendiente');
SELECT prueba_ct111.exigir((SELECT r.valor->>'Estado'='replay_confirmado'
    AND r.valor->>'ResolucionRef'=o.valor->>'ResolucionRef'
    FROM resultado r, resultado o WHERE r.caso='expiracion1-replay' AND o.caso='expiracion1'),
    'replay de la expiración sin efecto nuevo');
SELECT prueba_ct111.exigir((SELECT valor->>'EstadoPlazo'='vigente' AND valor->'RespuestaFueraDePlazo'='false'::jsonb
    AND valor->'IntencionSiguiente'='{}'::jsonb FROM resultado WHERE caso='aceptacion3'),'respuesta dentro de plazo');
SELECT prueba_ct111.exigir((SELECT valor->>'EstadoPlazo'='vigente' AND NOT valor ? 'RespuestaHasta'
    AND NOT valor ? 'RespuestaFueraDePlazo' FROM resultado WHERE caso='historica6'),'circuito histórico sin plazo abierto');
SELECT prueba_ct111.exigir((SELECT valor->'RespuestaFueraDePlazo'='true'::jsonb AND NOT valor ? 'CausaJustificadaRef'
    FROM resultado WHERE caso='admitida7'),'fuera de plazo admitida por la regla');
SELECT prueba_ct111.exigir((SELECT a.valor->'RespuestaFueraDePlazo'='true'::jsonb
    AND a.valor->>'CausaJustificadaRef'=c.valor->>'EventoRef' AND a.valor->>'EstadoPlazo'='vigente'
    FROM resultado a, resultado c WHERE a.caso='aceptacion4' AND c.caso='causa4'),
    'fuera de plazo admitida con causa justificada acreditada');
RESET SESSION AUTHORIZATION;

-- 3. La lista es configuración: retirar la admisión cierra la política.
UPDATE vec_contratacion_temporal.politica_llamamiento_admitida
   SET retirada_en=clock_timestamp() WHERE admision_ref='admision:reglas-bolsa:plazo-respuesta';
SET SESSION AUTHORIZATION vec_ct111_runtime;
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT prueba_ct111.esperar($s$SELECT prueba_ct111.plazo(prueba_ct111.material_contacto(8,
    '11111111-1111-4111-8111-000000000080',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 day'))$s$,
    'P0590','admisión retirada por configuración');
COMMIT;
RESET SESSION AUTHORIZATION;
UPDATE vec_contratacion_temporal.politica_llamamiento_admitida
   SET retirada_en=NULL WHERE admision_ref='admision:reglas-bolsa:plazo-respuesta';

-- 4. ACL y roles reales: ni lectura directa ni otra identidad.
SET SESSION AUTHORIZATION vec_ct111_runtime;
SELECT prueba_ct111.esperar('SELECT 1 FROM vec_contratacion_temporal.evento_plazo_llamamiento_rrhh','42501','lectura directa de eventos');
SELECT prueba_ct111.esperar('SELECT 1 FROM vec_contratacion_temporal.politica_llamamiento_admitida','42501','lectura directa de la lista');
SELECT prueba_ct111.esperar($s$SELECT vec_contratacion_temporal.politica_llamamiento_admitida_v1('a:1:b',1,repeat('a',64),'plazo')$s$,'42501','función interna de admisión');
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION vec_ct111_ajeno;
SELECT prueba_ct111.esperar($s$SELECT prueba_ct111.plazo(prueba_ct111.material_contacto(8,
    '11111111-1111-4111-8111-000000000081',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 day'))$s$,
    '42501','identidad sin rol ejecutor');
RESET SESSION AUTHORIZATION;
SELECT prueba_ct111.exigir(NOT has_function_privilege('vec_ct111_ajeno',
    'vec_contratacion_temporal.registrar_evento_plazo_llamamiento_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    AND has_function_privilege('vec_contratacion_temporal_ejecutor',
    'vec_contratacion_temporal.registrar_evento_plazo_llamamiento_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE'),
    'EXECUTE solo para el ejecutor');
SELECT prueba_ct111.esperar($s$UPDATE vec_contratacion_temporal.evento_plazo_llamamiento_rrhh SET prueba_ref='prueba:mutada'$s$,
    '55000','historia de plazo inmutable');
SELECT prueba_ct111.exigir((SELECT count(*) FROM vec_contratacion_temporal.evento_plazo_llamamiento_rrhh)=7
    AND (SELECT count(*) FROM vec_contratacion_temporal.resolucion_manual_respuesta_rrhh)=5,
    'siete eventos y cinco resoluciones, sin duplicados');
-- Recibos persistidos para la comprobación tras reinicio.
CREATE TABLE prueba_ct111.antes_reinicio AS SELECT * FROM resultado;
GRANT SELECT ON prueba_ct111.antes_reinicio TO vec_ct111_runtime;
