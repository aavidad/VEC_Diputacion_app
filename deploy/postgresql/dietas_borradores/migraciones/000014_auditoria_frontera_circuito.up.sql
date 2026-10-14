\set ON_ERROR_STOP on
-- Dietas 14: la frontera v2 ya valida estas diez acciones y sus rutas.
-- El CHECK de Dietas 5 conservaba solo cuatro y rechazaba el registro de
-- denegaciones del circuito. Se alinea esa lista, sin cambiar la función,
-- las filas anteriores, sus permisos ni las combinaciones de acción/ruta.
BEGIN;
SET LOCAL ROLE vec_dietas_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_dietas:migracion:000014',0));

DO $pre$
DECLARE c pg_constraint%ROWTYPE; f oid;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR current_user<>'vec_dietas_propietario'
    OR to_regclass('vec_dietas.auditoria_frontera_comision') IS NULL
 THEN RAISE EXCEPTION 'Dietas14: entorno incompatible' USING ERRCODE='55000'; END IF;
 IF (SELECT relowner FROM pg_class WHERE oid='vec_dietas.auditoria_frontera_comision'::regclass)
       IS DISTINCT FROM 'vec_dietas_propietario'::regrole
 THEN RAISE EXCEPTION 'Dietas14: propietario incompatible' USING ERRCODE='55000'; END IF;
 SELECT * INTO c FROM pg_constraint
  WHERE conrelid='vec_dietas.auditoria_frontera_comision'::regclass
    AND conname='auditoria_frontera_accion_cerrada';
 IF NOT FOUND OR c.contype<>'c' OR NOT c.convalidated OR c.connoinherit
    OR encode(sha256(convert_to(pg_get_constraintdef(c.oid),'UTF8')),'hex')
       IS DISTINCT FROM 'f9a94e5a30164a27e2505832614054cae1e5125fee25ad116baa50d6e2ca8690'
 THEN RAISE EXCEPTION 'Dietas14: preimagen de acciones incompatible' USING ERRCODE='55000'; END IF;
 f:=to_regprocedure('vec_dietas.registrar_auditoria_frontera_comision_v2(text,text,text,text,text,text)');
 IF f IS NULL OR encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex')
       IS DISTINCT FROM '6d624ebf594cc066e5fbc3d400c5475fa75913e5505c163a50682e4525b48404'
 THEN RAISE EXCEPTION 'Dietas14: frontera v2 incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

ALTER TABLE vec_dietas.auditoria_frontera_comision
 DROP CONSTRAINT auditoria_frontera_accion_cerrada;
ALTER TABLE vec_dietas.auditoria_frontera_comision
 ADD CONSTRAINT auditoria_frontera_accion_cerrada CHECK (accion = ANY (ARRAY[
  'listar'::text, 'crear'::text, 'consultar_detalle'::text, 'editar'::text,
  'borrar'::text, 'enviar'::text, 'consultar_bandeja'::text, 'decidir'::text,
  'preleer'::text, 'metodo_no_admitido'::text]));

DO $post$
DECLARE c pg_constraint%ROWTYPE;
BEGIN
 SELECT * INTO c FROM pg_constraint
  WHERE conrelid='vec_dietas.auditoria_frontera_comision'::regclass
    AND conname='auditoria_frontera_accion_cerrada';
 IF NOT FOUND OR c.contype<>'c' OR NOT c.convalidated OR c.connoinherit
    OR encode(sha256(convert_to(pg_get_constraintdef(c.oid),'UTF8')),'hex')
       IS DISTINCT FROM 'c1c3c4f45277b8edf58580f81c7f375ab37229b732b6f3e146134da6cededfc5'
    OR encode(sha256(convert_to(pg_get_functiondef(
       'vec_dietas.registrar_auditoria_frontera_comision_v2(text,text,text,text,text,text)'::regprocedure),'UTF8')),'hex')
       IS DISTINCT FROM '6d624ebf594cc066e5fbc3d400c5475fa75913e5505c163a50682e4525b48404'
 THEN RAISE EXCEPTION 'Dietas14: postimagen incompatible' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
