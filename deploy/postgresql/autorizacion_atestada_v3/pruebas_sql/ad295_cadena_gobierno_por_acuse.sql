\set ON_ERROR_STOP on
-- Auditoría de datos ficticios por sus escritores instalados, nunca un acto
-- de administración. No se cambia ningún permiso ni guarda; todo ROLLBACK.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';SET LOCAL statement_timeout='60s';
UPDATE vec_autorizacion_atestada_v3.sellado_auditoria_v5 SET latido=clock_timestamp();
CREATE FUNCTION pg_temp.exigir(caso text,ok boolean) RETURNS void LANGUAGE plpgsql AS $f$
BEGIN IF ok IS NOT TRUE THEN RAISE EXCEPTION 'AD295 prueba: %',caso; END IF; END $f$;
CREATE FUNCTION pg_temp.gobierno(caso text) RETURNS text LANGUAGE plpgsql AS $f$
DECLARE h text:=encode(sha256(convert_to('ad295:'||caso||clock_timestamp()::text,'UTF8')),'hex');r text;
BEGIN
 SELECT auditoria_ref INTO STRICT r FROM vec_autorizacion_atestada_v3.registrar_gobierno_usuarios_admin_v1(
 jsonb_build_object('tipo_registro','gobierno_usuarios_admin','evento_ref','evento_'||substr(h,1,32),
 'operador_login',session_user,'plan_sha256',h,'preimagen_sha256',repeat('a',64),
 'configuracion_origen_ref','config:ad295:origen','configuracion_destino_ref','config:ad295:destino',
 'claves_sha256',repeat('b',64),'proceso','postgresql','canal','operacion_tecnica_privada',
 'finalidad_ref','gobierno_usuarios_admin','correlacion_ref','correlacion_'||substr(h,1,32)));
 RETURN r;
END $f$;
CREATE FUNCTION pg_temp.otro() RETURNS void LANGUAGE plpgsql AS $f$
BEGIN PERFORM vec_autorizacion_atestada_v3.registrar_operacion_periodica_v1(
 'capturar_sello_periodico_v1','permitido','no_vencido',
 'correlacion_'||substr(encode(sha256(convert_to(clock_timestamp()::text,'UTF8')),'hex'),1,32),
 jsonb_build_object('perfil_tecnico_ref','vec_auditoria_periodica_sellador','configuracion_sha256',repeat('a',64)));
END $f$;
CREATE FUNCTION pg_temp.lectura(ref text) RETURNS jsonb LANGUAGE sql AS $f$
 SELECT vec_autorizacion_atestada_v3.cadena_gobierno_admin_lectura_v1('vec.auditoria.verificacion.gobierno-usuarios-admin.v1',ref)
$f$;
CREATE FUNCTION pg_temp.denegada(ref text) RETURNS text LANGUAGE plpgsql AS $f$
BEGIN PERFORM pg_temp.lectura(ref);RETURN 'admitida';EXCEPTION WHEN OTHERS THEN RETURN SQLSTATE;END $f$;
SELECT pg_temp.exigir('firma anterior retirada',to_regprocedure('vec_autorizacion_atestada_v3.cadena_gobierno_admin_lectura_v1(text)') IS NULL);
SELECT pg_temp.exigir('ACL sin ampliación',(SELECT p.prosecdef AND p.provolatile='s'
 AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp']
 AND p.proacl::text='{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario,vec_autorizacion_atestada_v3_lector_gobierno=X/vec_autorizacion_atestada_v3_propietario}'
 FROM pg_proc p WHERE oid='vec_autorizacion_atestada_v3.cadena_gobierno_admin_lectura_v1(text,text)'::regprocedure));
-- Los casos B/R: dos asientos de gobierno seguidos por otro tipo de asiento.
SELECT pg_temp.otro();
SELECT pg_temp.gobierno('B1') AS b1,pg_temp.gobierno('B2') AS b2 \gset
SELECT pg_temp.otro();
SELECT pg_temp.gobierno('R1') AS r1,pg_temp.gobierno('R2') AS r2 \gset
SELECT pg_temp.otro();
SELECT vec_autorizacion_atestada_v3.sellar_tramo_auditoria_v5(false,50000);
SELECT pg_temp.exigir('B: tramo conservado tras otro tipo',
 (SELECT j#>>'{manifiesto,registros}'='2' AND j#>>'{registros,0,gobierno_usuarios,auditoria_ref}'=:'b1'
 AND j#>>'{registros,1,gobierno_usuarios,auditoria_ref}'=:'b2' FROM (SELECT pg_temp.lectura(:'b1') j) s));
SELECT pg_temp.exigir('R: tramo conservado tras otro tipo',
 (SELECT j#>>'{manifiesto,registros}'='2' AND j#>>'{registros,0,gobierno_usuarios,auditoria_ref}'=:'r1'
 AND j#>>'{registros,1,gobierno_usuarios,auditoria_ref}'=:'r2' FROM (SELECT pg_temp.lectura(:'r1') j) s));
SELECT pg_temp.exigir('sin referencia conserva el sufijo actual',pg_temp.lectura(NULL)#>>'{manifiesto,registros}'='0');
-- Otro tramo de gobierno posterior no desplaza la consulta del acuse antiguo.
SELECT pg_temp.gobierno('posterior') AS posterior \gset
SELECT vec_autorizacion_atestada_v3.sellar_tramo_auditoria_v5(false,50000);
SELECT pg_temp.exigir('histórico tras otro gobierno',pg_temp.lectura(:'b1')#>>'{manifiesto,registros}'='2');
SELECT pg_temp.exigir('último tramo y llamada sin referencia compatibles',pg_temp.lectura(:'posterior')=pg_temp.lectura(NULL));
SELECT pg_temp.gobierno('sin-sellar') AS sin_sellar \gset
SELECT pg_temp.exigir('no inventa cobertura pendiente',pg_temp.denegada(:'sin_sellar')='42501');
SELECT pg_temp.exigir('referencia inexistente',pg_temp.denegada('aud_v3_gu_'||repeat('0',32))='42501');
SELECT pg_temp.exigir('referencia mal formada',pg_temp.denegada('otra')='22023');
ROLLBACK;
