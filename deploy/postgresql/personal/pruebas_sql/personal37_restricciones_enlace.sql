\set ON_ERROR_STOP on
-- Vector de Personal37. Se ejecuta como superusuario sobre un clon desechable
-- con Personal28 y Personal37. Copia las CHECK de la tabla de enlaces en una
-- tabla temporal y comprueba que se evalúan: antes de Personal37 el primer
-- INSERT ya falla con «invalid repetition count(s)». Termina en ROLLBACK.
BEGIN;
CREATE TEMP TABLE enlace_prueba (LIKE vec_personal.enlace_cargo_competencial_historia INCLUDING CONSTRAINTS) ON COMMIT DROP;
CREATE FUNCTION pg_temp.insertar(recurso text,finalidad text) RETURNS text LANGUAGE plpgsql AS $f$
BEGIN
 INSERT INTO enlace_prueba(enlace_ref,version,huella_sha256,cargo_ref,cargo_version,persona_ref,clase,accion_ref,recurso_ref,finalidad_ref,
  estado,vigente_desde,vigente_hasta,acto_ref,acto_version,acto_huella_sha256,fuente_ref,fuente_version,fuente_huella_sha256,
  requiere_enlace_laboral,publicada_en,decision_ref,auditoria_ref,recibo_ref)
 VALUES('enc_'||repeat('A',22),1,repeat('a',64),'car_'||repeat('B',22),1,'per_'||repeat('c',22),'titular','contratacion_temporal.documento.firma_vec.registrar',
  recurso,finalidad,'vigente',now(),now()+interval '1 day','acto:prueba',1,repeat('d',64),'fuente:prueba',1,repeat('e',64),
  false,now(),'decision:prueba','aud_prueba','percar_prueba');
 RETURN 'insertado';
EXCEPTION WHEN check_violation THEN RETURN 'rechazado';
END $f$;
SELECT CASE WHEN pg_temp.insertar('documento_contratacion_temporal','gestionar_contratacion_temporal')='insertado' THEN 'OK valido_insertado' ELSE 'FALLO valido_insertado' END;
SELECT CASE WHEN pg_temp.insertar('a/'||repeat('b',510),'f'||repeat('g',511))='insertado' THEN 'OK limite_512' ELSE 'FALLO limite_512' END;
SELECT CASE WHEN pg_temp.insertar('a'||repeat('b',512),'gestionar_x')='rechazado' THEN 'OK recurso_513_rechazado' ELSE 'FALLO recurso_513_rechazado' END;
SELECT CASE WHEN pg_temp.insertar('documento_x','f'||repeat('g',512))='rechazado' THEN 'OK finalidad_513_rechazada' ELSE 'FALLO finalidad_513_rechazada' END;
SELECT CASE WHEN pg_temp.insertar('Documento','gestionar_x')='rechazado' THEN 'OK mayuscula_rechazada' ELSE 'FALLO mayuscula_rechazada' END;
SELECT CASE WHEN pg_temp.insertar('ab','gestionar_x')='rechazado' THEN 'OK corta_rechazada' ELSE 'FALLO corta_rechazada' END;
SELECT CASE WHEN pg_temp.insertar('documento_x','gestionar/x')='rechazado' THEN 'OK finalidad_sin_barra' ELSE 'FALLO finalidad_sin_barra' END;
ROLLBACK;
