\set ON_ERROR_STOP on
-- Ensayo sólo en clon PostgreSQL 18 con B87 instalada. Todo se revierte.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='20s';

DO $test$
DECLARE f oid:=pg_catalog.to_regprocedure('vec_bolsa_llamamientos.registrar_contacto_telefonico_actual_v1(text,text,text,text,text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,integer,integer,boolean,text[],text,integer,integer,boolean,text,date,boolean)');
 v_ref text:='contacto:'||pg_catalog.repeat('e',64);
 v_instante timestamptz:='2026-10-08 10:00:00+00';
BEGIN
 IF f IS NULL OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_proc p WHERE p.oid=f
      AND p.proowner='vec_bolsa_llamamientos_propietario'::pg_catalog.regrole
      AND p.prosecdef AND p.provolatile='v'
      AND p.proconfig @> ARRAY['search_path=pg_catalog, pg_temp'])
    OR NOT pg_catalog.has_function_privilege('vec_bolsa_llamamientos_ejecutor',f,'EXECUTE')
    OR EXISTS(SELECT 1 FROM pg_catalog.pg_proc p
      CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,
       pg_catalog.acldefault('f',p.proowner))) a WHERE p.oid=f AND a.grantee=0)
    OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_attribute a
      WHERE a.attrelid='vec_bolsa_llamamientos.contacto_participacion'::pg_catalog.regclass
      AND a.attname='instante_servidor' AND a.atttypid='boolean'::pg_catalog.regtype
      AND a.attnotnull AND NOT a.attisdropped) THEN
  RAISE EXCEPTION 'B87: función, ACL o columna no instalada' USING ERRCODE='55000';
 END IF;

 -- La restricción conserva la nota obligatoria de registros anteriores.
 BEGIN
  INSERT INTO vec_bolsa_llamamientos.contacto_participacion(
   contacto_ref,bolsa_ref,participacion_ref,llamamiento_ref,canal,instante,actor,
   resultado,anotacion,clave_idempotencia,recibo_ref)
  VALUES(v_ref,'bolsa:b87','participacion:b87','llamamiento:b87','telefono',v_instante,
   'per_0123456789abcdefghijkl','contactado','','b87-legacy-vacio','recibo:'||v_ref);
  RAISE EXCEPTION 'B87: nota legacy vacía admitida' USING ERRCODE='55000';
 EXCEPTION WHEN check_violation THEN NULL;
 END;

 -- El modo nuevo permite sólo su propia nota vacía y resultado comunica.
 INSERT INTO vec_bolsa_llamamientos.contacto_participacion(
  contacto_ref,bolsa_ref,participacion_ref,llamamiento_ref,canal,instante,actor,
  resultado,anotacion,clave_idempotencia,recibo_ref,instante_servidor)
 VALUES(v_ref,'bolsa:b87','participacion:b87','llamamiento:b87','telefono',v_instante,
  'per_0123456789abcdefghijkl','comunica','','b87-servidor','recibo:'||v_ref,true);

 BEGIN
  INSERT INTO vec_bolsa_llamamientos.contacto_participacion(
   contacto_ref,bolsa_ref,participacion_ref,llamamiento_ref,canal,instante,actor,
   resultado,anotacion,clave_idempotencia,recibo_ref)
  VALUES('contacto:'||pg_catalog.repeat('f',64),'bolsa:b87','participacion:b87',
   'llamamiento:b87','telefono',v_instante,'per_0123456789abcdefghijkl',
   'comunica','legacy','b87-legacy-comunica',
   'recibo:contacto:'||pg_catalog.repeat('f',64));
  RAISE EXCEPTION 'B87: comunica legacy admitido' USING ERRCODE='55000';
 EXCEPTION WHEN check_violation THEN NULL;
 END;
END $test$;
-- Llamamiento emitido por el asistente: la llamada se admite para quien está
-- en su lista con el aviso por correo registrado; no para otra persona de la
-- bolsa. Con material V3 falso, la primera pasa la comprobación de llamamiento
-- y falla después (autorización); la segunda falla antes con 23503.
DO $b7$
DECLARE l record; v_dentro text; v_fuera text; v_estado text; v_rev bigint; v_rev2 bigint;
 f constant text:='SELECT * FROM vec_bolsa_llamamientos.registrar_contacto_telefonico_actual_v1($1,$2,$3,$4,''per_0123456789abcdefghijkl'',''no_contesta'','''',''b87-b7-''||$3,''recibo:''||$1,''\x00''::bytea,''\x00''::bytea,''\x00''::bytea,''\x00''::bytea,1,1,''\x00''::bytea,''\x00''::bytea,''\x00''::bytea,''\x00''::bytea,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL)';
BEGIN
 SELECT e.* INTO l FROM vec_bolsa_llamamientos.llamamiento_emitido e
  WHERE EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.contacto_participacion c
   WHERE c.llamamiento_ref=e.llamamiento_ref AND c.canal='correo')
  ORDER BY e.emitido_en DESC LIMIT 1;
 IF l.llamamiento_ref IS NULL THEN RAISE NOTICE 'B87: sin llamamiento emitido sintético; se omite'; RETURN; END IF;
 SELECT c.participacion_ref INTO v_dentro FROM vec_bolsa_llamamientos.contacto_participacion c
  WHERE c.llamamiento_ref=l.llamamiento_ref AND c.canal='correo' LIMIT 1;
 SELECT e.participacion_ref INTO v_fuera FROM vec_bolsa_llamamientos.constitucion_entrada e
  JOIN vec_bolsa_llamamientos.constitucion k USING(instantanea_ref,version_instantanea)
  WHERE k.bolsa_ref=l.bolsa_ref AND NOT l.participaciones ? e.participacion_ref LIMIT 1;
 BEGIN
  EXECUTE f USING 'contacto:'||pg_catalog.repeat('a',64),l.bolsa_ref,v_dentro,l.llamamiento_ref;
  RAISE EXCEPTION 'B87: material V3 falso admitido' USING ERRCODE='55000';
 EXCEPTION WHEN foreign_key_violation THEN RAISE EXCEPTION 'B87: llamada a persona del llamamiento emitido rechazada';
  WHEN OTHERS THEN GET STACKED DIAGNOSTICS v_estado=RETURNED_SQLSTATE;
   IF v_estado='55000' THEN RAISE; END IF;
 END;
 IF v_fuera IS NOT NULL THEN
  BEGIN
   EXECUTE f USING 'contacto:'||pg_catalog.repeat('b',64),l.bolsa_ref,v_fuera,l.llamamiento_ref;
   RAISE EXCEPTION 'B87: persona ajena admitida' USING ERRCODE='55000';
  EXCEPTION WHEN foreign_key_violation THEN NULL;
  END;
 END IF;
 -- Una llamada no toca la completitud del correo; un correo, sí.
 SELECT coalesce(max(revision),0) INTO v_rev FROM vec_bolsa_llamamientos.coordinacion_completitud_correo WHERE llamamiento_ref=l.llamamiento_ref;
 INSERT INTO vec_bolsa_llamamientos.contacto_participacion(contacto_ref,bolsa_ref,participacion_ref,llamamiento_ref,canal,instante,actor,resultado,anotacion,clave_idempotencia,recibo_ref,instante_servidor)
 VALUES('contacto:'||pg_catalog.repeat('c',64),l.bolsa_ref,v_dentro,l.llamamiento_ref,'telefono',pg_catalog.clock_timestamp(),'per_0123456789abcdefghijkl','comunica','','b87-b7-tel','recibo:contacto:'||pg_catalog.repeat('c',64),true);
 SELECT coalesce(max(revision),0) INTO v_rev2 FROM vec_bolsa_llamamientos.coordinacion_completitud_correo WHERE llamamiento_ref=l.llamamiento_ref;
 IF v_rev2<>v_rev THEN RAISE EXCEPTION 'B87: la llamada serializó la completitud (% -> %)',v_rev,v_rev2 USING ERRCODE='55000'; END IF;
 INSERT INTO vec_bolsa_llamamientos.contacto_participacion(contacto_ref,bolsa_ref,participacion_ref,llamamiento_ref,canal,instante,actor,resultado,anotacion,clave_idempotencia,recibo_ref)
 VALUES('contacto:'||pg_catalog.repeat('d',64),l.bolsa_ref,v_dentro,l.llamamiento_ref,'correo',pg_catalog.clock_timestamp(),'per_0123456789abcdefghijkl','enviado','Prueba de completitud','b87-b7-correo','recibo:contacto:'||pg_catalog.repeat('d',64));
 SELECT coalesce(max(revision),0) INTO v_rev2 FROM vec_bolsa_llamamientos.coordinacion_completitud_correo WHERE llamamiento_ref=l.llamamiento_ref;
 IF v_rev2<>v_rev+1 THEN RAISE EXCEPTION 'B87: el correo no recalculó la completitud (% -> %)',v_rev,v_rev2 USING ERRCODE='55000'; END IF;
END $b7$;
ROLLBACK;
