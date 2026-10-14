\set ON_ERROR_STOP on
-- Dietas 15: completar la corrección de auditoría de Dietas 14 con las
-- cuatro clases de ruta ya validadas por la frontera v2. No cambia la función,
-- las combinaciones de acción/ruta, los permisos ni las filas anteriores.
BEGIN;
SET LOCAL ROLE vec_dietas_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_dietas:migracion:000015',0));

DO $pre$
DECLARE c pg_constraint%ROWTYPE; f oid;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR current_user<>'vec_dietas_propietario'
    OR to_regclass('vec_dietas.auditoria_frontera_comision') IS NULL
 THEN RAISE EXCEPTION 'Dietas15: entorno incompatible' USING ERRCODE='55000'; END IF;
 IF (SELECT relowner FROM pg_class WHERE oid='vec_dietas.auditoria_frontera_comision'::regclass)
       IS DISTINCT FROM 'vec_dietas_propietario'::regrole
 THEN RAISE EXCEPTION 'Dietas15: propietario incompatible' USING ERRCODE='55000'; END IF;
 SELECT * INTO c FROM pg_constraint
  WHERE conrelid='vec_dietas.auditoria_frontera_comision'::regclass
    AND conname='auditoria_frontera_ruta_clase_cerrada';
 IF NOT FOUND OR c.contype<>'c' OR NOT c.convalidated OR c.connoinherit
    OR encode(sha256(convert_to(pg_get_constraintdef(c.oid),'UTF8')),'hex')
       IS DISTINCT FROM 'c0ebe9519df4435c9299e49c6072cadf7e16414104d7d6d9d210f92aa84da82d'
    OR (SELECT encode(sha256(convert_to(pg_get_constraintdef(oid),'UTF8')),'hex') FROM pg_constraint
       WHERE conrelid=c.conrelid AND conname='auditoria_frontera_accion_cerrada')
       IS DISTINCT FROM 'c1c3c4f45277b8edf58580f81c7f375ab37229b732b6f3e146134da6cededfc5'
 THEN RAISE EXCEPTION 'Dietas15: preimagen de rutas incompatible' USING ERRCODE='55000'; END IF;
 f:=to_regprocedure('vec_dietas.registrar_auditoria_frontera_comision_v2(text,text,text,text,text,text)');
 IF f IS NULL OR encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex')
       IS DISTINCT FROM '6d624ebf594cc066e5fbc3d400c5475fa75913e5505c163a50682e4525b48404'
 THEN RAISE EXCEPTION 'Dietas15: frontera v2 incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

ALTER TABLE vec_dietas.auditoria_frontera_comision
 DROP CONSTRAINT auditoria_frontera_ruta_clase_cerrada;
ALTER TABLE vec_dietas.auditoria_frontera_comision
 ADD CONSTRAINT auditoria_frontera_ruta_clase_cerrada CHECK (ruta = ANY (ARRAY[
  '/api/vec/dietas/comisiones'::text, '/api/vec/dietas/comisiones/detalle'::text,
  '/api/vec/dietas/comisiones/enviar'::text, '/api/vec/dietas/comisiones/circuito'::text]));

DO $post$
DECLARE c pg_constraint%ROWTYPE;
BEGIN
 SELECT * INTO c FROM pg_constraint
  WHERE conrelid='vec_dietas.auditoria_frontera_comision'::regclass
    AND conname='auditoria_frontera_ruta_clase_cerrada';
 IF NOT FOUND OR c.contype<>'c' OR NOT c.convalidated OR c.connoinherit
    OR encode(sha256(convert_to(pg_get_constraintdef(c.oid),'UTF8')),'hex')
       IS DISTINCT FROM 'fbe1b1b30371f2ab52a9c81cc62815eb8210939d321812c17aa1af728f6d25f7'
    OR (SELECT encode(sha256(convert_to(pg_get_constraintdef(oid),'UTF8')),'hex') FROM pg_constraint
       WHERE conrelid=c.conrelid AND conname='auditoria_frontera_accion_cerrada')
       IS DISTINCT FROM 'c1c3c4f45277b8edf58580f81c7f375ab37229b732b6f3e146134da6cededfc5'
    OR encode(sha256(convert_to(pg_get_functiondef(
       'vec_dietas.registrar_auditoria_frontera_comision_v2(text,text,text,text,text,text)'::regprocedure),'UTF8')),'hex')
       IS DISTINCT FROM '6d624ebf594cc066e5fbc3d400c5475fa75913e5505c163a50682e4525b48404'
 THEN RAISE EXCEPTION 'Dietas15: postimagen incompatible' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
