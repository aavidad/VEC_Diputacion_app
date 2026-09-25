\set ON_ERROR_STOP on
-- Pruebas negativas y de historia de CT118 sobre PostgreSQL 18 real con
-- AD3-85 y CT118 instaladas. Se ejecuta como superusuario del ensayo y todo
-- se deshace al final: los dos roles de prueba se borran y las filas se
-- revierten; no deja cambios.
-- El recorrido positivo (consumo V3 real) lo cubre la prueba Go del adaptador.
BEGIN;
SET LOCAL search_path = pg_catalog;
CREATE ROLE vec_ct118_prueba_ejecutor LOGIN;
GRANT vec_contratacion_temporal_ejecutor TO vec_ct118_prueba_ejecutor;
CREATE ROLE vec_ct118_prueba_ajeno LOGIN;

CREATE FUNCTION pg_temp.debe_fallar(p_sql text, p_codigo text, p_etiqueta text) RETURNS void
LANGUAGE plpgsql AS $f$
BEGIN
    BEGIN
        EXECUTE p_sql;
    EXCEPTION WHEN others THEN
        IF SQLSTATE <> p_codigo THEN
            RAISE EXCEPTION 'FALLO %: código % (esperado %): %', p_etiqueta, SQLSTATE, p_codigo, SQLERRM;
        END IF;
        RAISE NOTICE 'OK %', p_etiqueta;
        RETURN;
    END;
    RAISE EXCEPTION 'FALLO %: no se rechazó', p_etiqueta;
END $f$;
GRANT EXECUTE ON FUNCTION pg_temp.debe_fallar(text,text,text) TO PUBLIC;

-- ACL: ni PUBLIC ni un rol ajeno ejecutan ni leen; el ejecutor no toca tablas.
DO $acl$
BEGIN
    IF has_function_privilege('vec_ct118_prueba_ajeno','vec_contratacion_temporal.registrar_firma_documento_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
       OR has_function_privilege('vec_ct118_prueba_ajeno','vec_contratacion_temporal.consultar_firmas_documento_v1(text,text)','EXECUTE')
       OR has_function_privilege('vec_contratacion_temporal_ejecutor','vec_autorizacion_atestada_v3.registrar_y_consumir_firma_documento_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
       OR has_table_privilege('vec_contratacion_temporal_ejecutor','vec_contratacion_temporal.firma_documento_v1','SELECT,INSERT,UPDATE,DELETE')
       OR has_table_privilege('vec_contratacion_temporal_ejecutor','vec_contratacion_temporal.firma_documento_outbox_v1','SELECT,INSERT,UPDATE,DELETE')
       OR has_table_privilege('vec_contratacion_temporal_ejecutor','vec_contratacion_temporal.firma_documento_auditoria_v1','SELECT,INSERT,UPDATE,DELETE')
       OR NOT has_function_privilege('vec_contratacion_temporal_ejecutor','vec_contratacion_temporal.registrar_firma_documento_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
       OR NOT has_function_privilege('vec_contratacion_temporal_propietario','vec_autorizacion_atestada_v3.registrar_y_consumir_firma_documento_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE') THEN
        RAISE EXCEPTION 'FALLO ACL de CT118/AD3-85';
    END IF;
    RAISE NOTICE 'OK ACL nominal';
END $acl$;

-- Un rol ajeno no alcanza la función.
SET SESSION AUTHORIZATION vec_ct118_prueba_ajeno;
SELECT pg_temp.debe_fallar($$SELECT vec_contratacion_temporal.consultar_firmas_documento_v1('organizacion:desarrollo:dipgra','expediente:ct:x')$$,'42501','consulta por rol ajeno');
RESET SESSION AUTHORIZATION;

SET SESSION AUTHORIZATION vec_ct118_prueba_ejecutor;
-- Fuera de SERIALIZABLE se deniega antes de leer nada.
SELECT pg_temp.debe_fallar($$SELECT vec_contratacion_temporal.registrar_firma_documento_v1('{}','\x','\x','\x','\x',1,1,'\x','\x','\x','\x')$$,'42501','registro fuera de serializable');
-- La consulta de un expediente sin firmas devuelve una lista vacía.
DO $c$
BEGIN
    IF vec_contratacion_temporal.consultar_firmas_documento_v1('organizacion:desarrollo:dipgra','expediente:ct:sin-firmas') <> '[]'::jsonb THEN
        RAISE EXCEPTION 'FALLO consulta vacía';
    END IF;
    RAISE NOTICE 'OK consulta vacía';
END $c$;
SELECT pg_temp.debe_fallar($$SELECT vec_contratacion_temporal.consultar_firmas_documento_v1('x','expediente:ct:a')$$,'22023','consulta con organización inválida');
RESET SESSION AUTHORIZATION;
COMMIT;

-- Registro en SERIALIZABLE: solicitudes inválidas y decisión divergente.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path = pg_catalog;
SET SESSION AUTHORIZATION vec_ct118_prueba_ejecutor;
SELECT pg_temp.debe_fallar($$SELECT vec_contratacion_temporal.registrar_firma_documento_v1('{"a":1}','\x','\x','\x','\x',1,1,'\x','\x','\x','\x')$$,'22023','claves no exactas');
SELECT pg_temp.debe_fallar($$SELECT vec_contratacion_temporal.registrar_firma_documento_v1('no es json','\x','\x','\x','\x',1,1,'\x','\x','\x','\x')$$,'22023','solicitud no JSON');
-- Firma sin dictamen: incoherente.
SELECT pg_temp.debe_fallar($$SELECT vec_contratacion_temporal.registrar_firma_documento_v1('{"OrganizacionRef":"organizacion:desarrollo:dipgra","ExpedienteRef":"expediente:ct:a","VersionExpediente":1,"Documento":"informe_definitivo","CatalogoRef":"vec.contratacion_temporal.circuito_firma:1","CatalogoHuella":"1111111111111111111111111111111111111111111111111111111111111111","PasoRef":"vec.contratacion_temporal.circuito_firma:1:informe_definitivo.p1","PasoOrden":1,"Secuencia":1,"Resultado":"firmado","MotivoDevolucion":null,"OriginalHuella":null,"FirmadoHuella":null,"CertificadoHuella":null,"FirmanteRef":null,"PoliticaVerificacion":null,"RevocacionEstado":null,"SelloTiempoEstado":null,"ClaveIdempotencia":"clave-firma-prueba-0001"}','\x','\x','\x','\x',1,1,'\x','\x','\x','\x')$$,'22023','firma sin dictamen');
-- Devolución bien formada pero con una decisión que no es la de firmar.
SELECT pg_temp.debe_fallar($$SELECT vec_contratacion_temporal.registrar_firma_documento_v1('{"OrganizacionRef":"organizacion:desarrollo:dipgra","ExpedienteRef":"expediente:ct:a","VersionExpediente":1,"Documento":"informe_definitivo","CatalogoRef":"vec.contratacion_temporal.circuito_firma:1","CatalogoHuella":"1111111111111111111111111111111111111111111111111111111111111111","PasoRef":"vec.contratacion_temporal.circuito_firma:1:informe_definitivo.p1","PasoOrden":1,"Secuencia":1,"Resultado":"devuelto","MotivoDevolucion":"Falta la fecha de efectos","OriginalHuella":null,"FirmadoHuella":null,"CertificadoHuella":null,"FirmanteRef":null,"PoliticaVerificacion":null,"RevocacionEstado":null,"SelloTiempoEstado":null,"ClaveIdempotencia":"clave-firma-prueba-0002"}','\x','{"accion":"contratacion_temporal.seguimiento.cerrar"}','\x','\x',1,1,'\x','\x','\x','\x')$$,'42501','decisión divergente');
RESET SESSION AUTHORIZATION;
ROLLBACK;

-- Historia de solo adición y DOWN protegido (filas sintéticas del propietario).
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
INSERT INTO vec_contratacion_temporal.firma_documento_v1 (
    firma_ref,organizacion_ref,expediente_ref,expediente_version,documento,secuencia,clave_idempotencia,
    solicitud_huella_sha256,catalogo_ref,catalogo_huella_sha256,paso_ref,paso_orden,resultado,motivo_devolucion,
    actor_ref,perfil_ref,decision_ref,consumo_huella_sha256,auditoria_consumo_ref,recibo_ref,registrada_en)
VALUES ('firma-ct:00000000-0000-4000-8000-000000000001','organizacion:desarrollo:dipgra','expediente:ct:a',1,
    'informe_definitivo',1,'clave-firma-prueba-0003',repeat('a',64),'vec.contratacion_temporal.circuito_firma:1',
    repeat('b',64),'vec.contratacion_temporal.circuito_firma:1:informe_definitivo.p1',1,'devuelto','Falta la fecha',
    'principal:prueba','perfil:prueba','decision:prueba',repeat('c',64),'auditoria:prueba',
    'recibo-firma-ct:00000000-0000-4000-8000-000000000001',date_trunc('microseconds',clock_timestamp()));
SELECT pg_temp.debe_fallar($$UPDATE vec_contratacion_temporal.firma_documento_v1 SET motivo_devolucion='Otro motivo'$$,'55000','UPDATE de historia');
SELECT pg_temp.debe_fallar($$DELETE FROM vec_contratacion_temporal.firma_documento_v1$$,'55000','DELETE de historia');
-- Una firma sin su dictamen completo no cabe en la tabla.
SELECT pg_temp.debe_fallar($$INSERT INTO vec_contratacion_temporal.firma_documento_v1 (
    firma_ref,organizacion_ref,expediente_ref,expediente_version,documento,secuencia,clave_idempotencia,
    solicitud_huella_sha256,catalogo_ref,catalogo_huella_sha256,paso_ref,paso_orden,resultado,
    actor_ref,perfil_ref,decision_ref,consumo_huella_sha256,auditoria_consumo_ref,recibo_ref,registrada_en)
VALUES ('firma-ct:00000000-0000-4000-8000-000000000002','organizacion:desarrollo:dipgra','expediente:ct:a',1,
    'informe_definitivo',2,'clave-firma-prueba-0004',repeat('a',64),'vec.contratacion_temporal.circuito_firma:1',
    repeat('b',64),'vec.contratacion_temporal.circuito_firma:1:informe_definitivo.p1',1,'firmado',
    'principal:prueba','perfil:prueba','decision:prueba',repeat('d',64),'auditoria:prueba2',
    'recibo-firma-ct:00000000-0000-4000-8000-000000000002',now())$$,'23514','firma sin dictamen en tabla');
RESET ROLE;
ROLLBACK;
DROP ROLE vec_ct118_prueba_ejecutor;
DROP ROLE vec_ct118_prueba_ajeno;
