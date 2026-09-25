\set ON_ERROR_STOP on
-- Pruebas funcionales de Bolsa 000039 sobre la función real de guardado
-- (cadena 000001-000006 + 000039) con dobles AD3. Los registros canónicos los
-- genera el servicio Go de integración (orden, apertura, terminal «sin
-- respuesta» y siguiente llamamiento) y llegan en /fixtures/registros.json.
-- La orden y la apertura son de hace dos minutos; la expiración y el
-- siguiente se confirman al alcanzar su instante (+20 s y +40 s).
\set registros `cat /fixtures/registros.json`
CREATE SCHEMA prueba_b39;
GRANT USAGE ON SCHEMA prueba_b39 TO vec_b39_runtime, vec_b39_ajeno;
CREATE TABLE prueba_b39.registro AS
SELECT key AS operacion_ref, decode(value,'base64') AS canonico
  FROM jsonb_each_text(:'registros'::jsonb) WHERE key<>'generado_en';
CREATE TABLE prueba_b39.generado AS SELECT ((:'registros'::jsonb)->>'generado_en')::timestamptz AS en;
GRANT SELECT ON prueba_b39.registro, prueba_b39.generado TO vec_b39_runtime, vec_b39_ajeno;

-- Capacidad estructural ligada al registro exacto, como la exige la función.
CREATE FUNCTION prueba_b39.capacidad(p_registro bytea, p_operacion text, p_denegar boolean) RETURNS bytea
LANGUAGE sql IMMUTABLE SET search_path = pg_catalog AS $f$
    SELECT convert_to(jsonb_build_object(
        'efecto_ref',r->>'operacion_ref',
        'huella_efecto_sha256',encode(sha256(convert_to('{"ambitos":{"categoria_ref":'||to_json(r->>'categoria_ref')::text||
            ',"unidad_ref":'||to_json(r->>'unidad_ref')::text||'},"atributos":{"contenido_sha256":'||
            to_json(encode(sha256(p_registro),'hex'))::text||',"necesidad_ref":'||to_json(r->>'necesidad_ref')::text||'}}','UTF8')),'hex'),
        'operacion',p_operacion,
        'audiencia_consumo','vec_bolsa_llamamientos.confirmar_integracion_desarrollo.v1',
        'denegar',CASE WHEN p_denegar THEN 'si' ELSE 'no' END,
        'relleno',repeat('x',600))::text,'UTF8')
      FROM (SELECT convert_from(p_registro,'UTF8')::jsonb AS r) x
$f$;
CREATE FUNCTION prueba_b39.guardar(p_registro bytea, p_operacion text, p_denegar boolean DEFAULT false)
RETURNS TABLE(registro_canonico bytea,recibo_ref text,auditoria_ref text,evento_ref text,confirmada_en timestamptz)
LANGUAGE sql VOLATILE SET search_path = pg_catalog AS $f$
    SELECT * FROM vec_bolsa_llamamientos.guardar_integracion_desarrollo_v1(p_registro,
        prueba_b39.capacidad(p_registro,p_operacion,p_denegar),'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,1,1,
        '\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea)
$f$;
CREATE FUNCTION prueba_b39.canonico(p_operacion text) RETURNS bytea
LANGUAGE sql STABLE SET search_path = pg_catalog AS $f$
    SELECT canonico FROM prueba_b39.registro WHERE operacion_ref=p_operacion
$f$;
-- Variante editada del registro (nuevo canon): para negativas.
CREATE FUNCTION prueba_b39.variante(p_operacion text, p_cambio jsonb) RETURNS bytea
LANGUAGE sql STABLE SET search_path = pg_catalog AS $f$
    SELECT convert_to((convert_from(canonico,'UTF8')::jsonb||p_cambio)::text,'UTF8')
      FROM prueba_b39.registro WHERE operacion_ref=p_operacion
$f$;
CREATE FUNCTION prueba_b39.esperar(p_sql text, p_estado text, p_caso text) RETURNS void
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
CREATE FUNCTION prueba_b39.exigir(p_condicion boolean, p_caso text) RETURNS void
LANGUAGE plpgsql SET search_path = pg_catalog AS $f$
BEGIN
    IF p_condicion IS NOT TRUE THEN
        RAISE EXCEPTION 'caso fallido: %',p_caso;
    END IF;
    RAISE NOTICE 'comprobado: %',p_caso;
END
$f$;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA prueba_b39 TO vec_b39_runtime, vec_b39_ajeno;
SELECT prueba_b39.exigir((SELECT count(*) FROM prueba_b39.registro)=4
    AND (SELECT convert_from(prueba_b39.canonico('operacion:expiracion'),'UTF8')::jsonb->>'tipo')='expiracion_rrhh'
    AND (SELECT convert_from(prueba_b39.canonico('operacion:expiracion'),'UTF8')::jsonb->>'estado_llamamiento')='expiracion_gobernada',
    'registros generados por el servicio Go: orden, apertura, sin respuesta y siguiente');

SET SESSION AUTHORIZATION vec_b39_runtime;
SET TimeZone='UTC';
SET statement_timeout='15s';
SET idle_in_transaction_session_timeout='20s';
CREATE TEMP TABLE r39 (caso text PRIMARY KEY, recibo text, evento text, confirmada timestamptz);
BEGIN ISOLATION LEVEL SERIALIZABLE;
INSERT INTO r39 SELECT 'orden',g.recibo_ref,g.evento_ref,g.confirmada_en
  FROM prueba_b39.guardar(prueba_b39.canonico('operacion:orden'),'bolsa.orden.preparar') g;
INSERT INTO r39 SELECT 'apertura',g.recibo_ref,g.evento_ref,g.confirmada_en
  FROM prueba_b39.guardar(prueba_b39.canonico('operacion:apertura'),'bolsa.llamamiento.abrir') g;
COMMIT;
-- Hasta el instante de la resolución (+20 s desde la generación).
SET statement_timeout=0;
SELECT pg_sleep(greatest(0,extract(epoch FROM (SELECT en FROM prueba_b39.generado)+interval '21 seconds'-clock_timestamp())));
SET statement_timeout='15s';

BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT prueba_b39.esperar($s$SELECT * FROM prueba_b39.guardar(prueba_b39.canonico('operacion:expiracion'),
    'bolsa.llamamiento.aceptacion_rrhh.registrar')$s$,'42501','permiso de aceptación para cerrar sin respuesta');
SELECT prueba_b39.esperar($s$SELECT * FROM prueba_b39.guardar(prueba_b39.variante('operacion:expiracion',
    '{"estado_llamamiento":"renuncia"}'),'bolsa.llamamiento.renuncia_rrhh.registrar')$s$,
    '22023','sin respuesta con estado de renuncia');
SELECT prueba_b39.esperar($s$SELECT * FROM prueba_b39.guardar(prueba_b39.variante('operacion:expiracion',
    '{"tipo":"renuncia_rrhh"}'),'bolsa.llamamiento.renuncia_rrhh.registrar')$s$,
    '22023','renuncia con estado expirado');
SELECT prueba_b39.esperar($s$SELECT * FROM prueba_b39.guardar(prueba_b39.canonico('operacion:expiracion'),
    'bolsa.llamamiento.renuncia_rrhh.registrar',true)$s$,'42501','autorización denegada por el consumidor (doble)');
COMMIT;
BEGIN ISOLATION LEVEL SERIALIZABLE;
INSERT INTO r39 SELECT 'expiracion',g.recibo_ref,g.evento_ref,g.confirmada_en
  FROM prueba_b39.guardar(prueba_b39.canonico('operacion:expiracion'),'bolsa.llamamiento.renuncia_rrhh.registrar') g;
COMMIT;
BEGIN ISOLATION LEVEL SERIALIZABLE;
INSERT INTO r39 SELECT 'expiracion-replay',g.recibo_ref,g.evento_ref,g.confirmada_en
  FROM prueba_b39.guardar(prueba_b39.canonico('operacion:expiracion'),'bolsa.llamamiento.renuncia_rrhh.registrar') g;
SELECT prueba_b39.esperar($s$SELECT * FROM prueba_b39.guardar(prueba_b39.variante('operacion:expiracion',
    '{"operacion_ref":"operacion:renuncia-tras-expiracion","tipo":"renuncia_rrhh","estado_llamamiento":"renuncia"}'),
    'bolsa.llamamiento.renuncia_rrhh.registrar')$s$,'23505','segundo terminal sobre la misma apertura');
COMMIT;
SELECT prueba_b39.exigir((SELECT r.recibo=o.recibo AND r.evento=o.evento AND r.confirmada=o.confirmada
    FROM r39 r, r39 o WHERE r.caso='expiracion-replay' AND o.caso='expiracion'),'replay del terminal sin efecto nuevo');
-- Hasta el instante del siguiente (+40 s).
SET statement_timeout=0;
SELECT pg_sleep(greatest(0,extract(epoch FROM (SELECT en FROM prueba_b39.generado)+interval '41 seconds'-clock_timestamp())));
SET statement_timeout='15s';
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT prueba_b39.esperar($s$SELECT * FROM prueba_b39.guardar(prueba_b39.canonico('operacion:siguiente'),
    'bolsa.llamamiento.abrir')$s$,'42501','siguiente con permiso de primer llamamiento');
INSERT INTO r39 SELECT 'siguiente',g.recibo_ref,g.evento_ref,g.confirmada_en
  FROM prueba_b39.guardar(prueba_b39.canonico('operacion:siguiente'),'bolsa.llamamiento.siguiente.abrir') g;
COMMIT;
BEGIN ISOLATION LEVEL READ COMMITTED;
SELECT prueba_b39.esperar($s$SELECT * FROM prueba_b39.guardar(prueba_b39.canonico('operacion:expiracion'),
    'bolsa.llamamiento.renuncia_rrhh.registrar')$s$,'42501','aislamiento no serializable');
COMMIT;
SELECT prueba_b39.esperar('SELECT 1 FROM vec_bolsa_llamamientos.integracion_desarrollo','42501','lectura directa del registro');
RESET SESSION AUTHORIZATION;

SET SESSION AUTHORIZATION vec_b39_ajeno;
SELECT prueba_b39.esperar($s$SELECT * FROM prueba_b39.guardar(prueba_b39.canonico('operacion:expiracion'),
    'bolsa.llamamiento.renuncia_rrhh.registrar')$s$,'42501','identidad sin rol ejecutor');
RESET SESSION AUTHORIZATION;
SELECT prueba_b39.exigir((SELECT count(*) FROM vec_bolsa_llamamientos.integracion_desarrollo)=4
    AND (SELECT tipo FROM vec_bolsa_llamamientos.integracion_desarrollo WHERE operacion_ref='operacion:expiracion')='expiracion_rrhh'
    AND (SELECT apertura_operacion_ref FROM vec_bolsa_llamamientos.integracion_desarrollo WHERE operacion_ref='operacion:expiracion')='operacion:apertura'
    AND (SELECT terminal_anterior_ref FROM vec_bolsa_llamamientos.integracion_desarrollo WHERE operacion_ref='operacion:siguiente')='operacion:expiracion'
    AND (SELECT convert_from(evento_canonico,'UTF8')::jsonb->>'tipo' FROM vec_bolsa_llamamientos.outbox_integracion_desarrollo
          WHERE operacion_ref='operacion:expiracion')='bolsa.llamamiento.expiracion_rrhh.registrada'
    AND (SELECT count(*) FROM vec_bolsa_llamamientos.auditoria_integracion_desarrollo)=4
    AND (SELECT count(*) FROM vec_bolsa_llamamientos.outbox_integracion_desarrollo)=4,
    'terminal sin respuesta, siguiente ligado, cuatro auditorías y cuatro eventos');
SELECT prueba_b39.esperar($s$UPDATE vec_bolsa_llamamientos.integracion_desarrollo SET recibo_ref='recibo:mutado'
    WHERE operacion_ref='operacion:expiracion'$s$,'55000','historia inmutable');
