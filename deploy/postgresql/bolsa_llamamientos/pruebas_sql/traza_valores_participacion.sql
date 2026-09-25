-- Petición RRHH p.4 (000034): traza de valor anterior y nuevo. Se ejecuta sobre
-- una base con 000034 y al menos una participación constituida; todo termina
-- en ROLLBACK.
\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

DO $prueba$
DECLARE p text; base timestamptz; v bigint; filas bigint; r record;
BEGIN
 SELECT e.participacion_ref, clock_timestamp()+interval '1 hour' INTO STRICT p, base
   FROM vec_bolsa_llamamientos.constitucion_entrada e
   JOIN LATERAL (SELECT situacion FROM vec_bolsa_llamamientos.situacion_participacion s WHERE s.participacion_ref=e.participacion_ref ORDER BY desde DESC LIMIT 1) actual ON true
  WHERE actual.situacion='disponible' LIMIT 1;

 -- Situación: anterior y nueva; la fecha solo cuando cambia.
 INSERT INTO vec_bolsa_llamamientos.situacion_participacion(participacion_ref,situacion,desde,motivo,actor,registrada_en,clave_idempotencia,recibo_ref)
 VALUES(p,'no_disponible',base,'Pausa sintética','persona:traza',base,'traza:1','recibo:traza:1');
 SELECT * INTO STRICT r FROM vec_bolsa_llamamientos.traza_valor_participacion WHERE recibo_ref='recibo:traza:1';
 IF r.campo<>'situacion' OR r.valor_anterior<>'disponible' OR r.valor_nuevo<>'no_disponible' OR r.actor<>'persona:traza' THEN RAISE EXCEPTION 'traza: situación mal trazada %', r; END IF;
 INSERT INTO vec_bolsa_llamamientos.situacion_participacion(participacion_ref,situacion,desde,motivo,actor,registrada_en,clave_idempotencia,recibo_ref)
 VALUES(p,'disponible',base+interval '1 second','Reactivación sintética','persona:traza',base+interval '1 second','traza:2','recibo:traza:2');
 INSERT INTO vec_bolsa_llamamientos.situacion_participacion(participacion_ref,situacion,desde,motivo,actor,registrada_en,clave_idempotencia,recibo_ref)
 VALUES(p,'pendiente_incorporacion',base+interval '2 second','Llamamiento','persona:traza',base+interval '2 second','traza:3','recibo:traza:3');
 INSERT INTO vec_bolsa_llamamientos.situacion_participacion(participacion_ref,situacion,desde,motivo,actor,registrada_en,clave_idempotencia,recibo_ref)
 VALUES(p,'trabajando',base+interval '3 second','Incorporación','persona:traza',base+interval '3 second','traza:4','recibo:traza:4');
 INSERT INTO vec_bolsa_llamamientos.situacion_participacion(participacion_ref,situacion,desde,fecha_disponible,motivo,actor,registrada_en,clave_idempotencia,recibo_ref)
 VALUES(p,'disponible_desde',base+interval '4 second',base+interval '30 days','Fin de contrato','persona:traza',base+interval '4 second','traza:5','recibo:traza:5');
 SELECT count(*) INTO filas FROM vec_bolsa_llamamientos.traza_valor_participacion WHERE recibo_ref='recibo:traza:5';
 IF filas<>2 OR NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.traza_valor_participacion WHERE recibo_ref='recibo:traza:5' AND campo='fecha_disponible' AND valor_anterior IS NULL AND valor_nuevo=vec_bolsa_llamamientos.fecha_traza_v1(base+interval '30 days')) THEN
  RAISE EXCEPTION 'traza: fecha de disponibilidad no trazada';
 END IF;

 -- Datos de contacto: versión anterior y nueva; subcampos solo si la
 -- aplicación los declara, y la declaración se consume.
 SELECT coalesce(max(version),0)+1 INTO v FROM vec_bolsa_llamamientos.datos_contacto_participacion WHERE participacion_ref=p;
 PERFORM set_config('vec_bolsa.campos_contacto_cambiados','telefono_2,correo',true);
 INSERT INTO vec_bolsa_llamamientos.datos_contacto_participacion(participacion_ref,version,clave_ref,nonce,cifrado,motivo,actor,registrada_en,clave_idempotencia,recibo_ref)
 VALUES(p,v,'clave:prueba',decode(repeat('00',12),'hex'),decode(repeat('01',32),'hex'),'Cambio sintético','persona:traza',base,'traza:c1','recibo:traza:c1');
 SELECT count(*) INTO filas FROM vec_bolsa_llamamientos.traza_valor_participacion WHERE recibo_ref='recibo:traza:c1';
 IF filas<>3 OR NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.traza_valor_participacion WHERE recibo_ref='recibo:traza:c1' AND campo='correo' AND valor_nuevo='version:'||v AND valor_anterior IS NOT DISTINCT FROM CASE WHEN v>1 THEN 'version:'||(v-1) END)
    OR EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.traza_valor_participacion WHERE recibo_ref='recibo:traza:c1' AND campo='telefono_1') THEN
  RAISE EXCEPTION 'traza: subcampos de contacto mal trazados';
 END IF;
 IF current_setting('vec_bolsa.campos_contacto_cambiados',true)<>'' THEN RAISE EXCEPTION 'traza: declaración no consumida'; END IF;
 INSERT INTO vec_bolsa_llamamientos.datos_contacto_participacion(participacion_ref,version,clave_ref,nonce,cifrado,motivo,actor,registrada_en,clave_idempotencia,recibo_ref)
 VALUES(p,v+1,'clave:prueba',decode(repeat('00',12),'hex'),decode(repeat('02',32),'hex'),'Sin declaración','persona:traza',base,'traza:c2','recibo:traza:c2');
 SELECT count(*) INTO filas FROM vec_bolsa_llamamientos.traza_valor_participacion WHERE recibo_ref='recibo:traza:c2';
 IF filas<>1 THEN RAISE EXCEPTION 'traza: versión sin declaración mal trazada'; END IF;

 -- Declaración inválida (un claro) rechazada, y con ella el cambio.
 BEGIN
  PERFORM set_config('vec_bolsa.campos_contacto_cambiados','persona@ejemplo.es',true);
  INSERT INTO vec_bolsa_llamamientos.datos_contacto_participacion(participacion_ref,version,clave_ref,nonce,cifrado,motivo,actor,registrada_en,clave_idempotencia,recibo_ref)
  VALUES(p,v+2,'clave:prueba',decode(repeat('00',12),'hex'),decode(repeat('03',32),'hex'),'Inválido','persona:traza',base,'traza:c3','recibo:traza:c3');
  RAISE EXCEPTION 'traza: declaración inválida aceptada';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.datos_contacto_participacion WHERE recibo_ref='recibo:traza:c3') THEN RAISE EXCEPTION 'traza: cambio sin traza persistido'; END IF;

 -- Un claro de contacto no cabe en la traza, ni siquiera escribiendo directo.
 BEGIN
  INSERT INTO vec_bolsa_llamamientos.traza_valor_participacion(participacion_ref,recibo_ref,campo,valor_anterior,valor_nuevo,actor,registrada_en)
  VALUES(p,'recibo:traza:x','correo','antiguo@ejemplo.es','nuevo@ejemplo.es','persona:traza',base);
  RAISE EXCEPTION 'traza: claro de contacto aceptado';
 EXCEPTION WHEN check_violation THEN NULL; END;

 -- Solo adición.
 BEGIN
  UPDATE vec_bolsa_llamamientos.traza_valor_participacion SET valor_nuevo='excluido' WHERE recibo_ref='recibo:traza:1';
  RAISE EXCEPTION 'traza: modificación aceptada';
 EXCEPTION WHEN others THEN IF SQLERRM='traza: modificación aceptada' THEN RAISE; END IF; END;
 BEGIN
  DELETE FROM vec_bolsa_llamamientos.traza_valor_participacion WHERE recibo_ref='recibo:traza:1';
  RAISE EXCEPTION 'traza: borrado aceptado';
 EXCEPTION WHEN others THEN IF SQLERRM='traza: borrado aceptado' THEN RAISE; END IF; END;
END $prueba$;

DO $acl$ BEGIN
 IF NOT (SELECT relrowsecurity AND relforcerowsecurity FROM pg_class WHERE oid='vec_bolsa_llamamientos.traza_valor_participacion'::regclass) THEN RAISE EXCEPTION 'traza: sin RLS forzada'; END IF;
 IF has_table_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.traza_valor_participacion','SELECT,INSERT,UPDATE,DELETE') THEN RAISE EXCEPTION 'traza: ejecutor con acceso directo'; END IF;
 IF has_function_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.trazar_situacion_participacion_v1()','EXECUTE')
    OR has_function_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.trazar_datos_contacto_participacion_v1()','EXECUTE')
    OR has_function_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.fecha_traza_v1(timestamptz)','EXECUTE') THEN RAISE EXCEPTION 'traza: funciones internas abiertas'; END IF;
 IF NOT has_function_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.listar_historial_participacion_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE') THEN RAISE EXCEPTION 'traza: lectura cerrada al ejecutor'; END IF;
 IF has_function_privilege('public','vec_bolsa_llamamientos.listar_historial_participacion_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE') THEN RAISE EXCEPTION 'traza: lectura abierta a PUBLIC'; END IF;
END $acl$;

SET LOCAL ROLE vec_bolsa_llamamientos_ejecutor;
DO $ejecutor$ BEGIN
 BEGIN
  PERFORM 1 FROM vec_bolsa_llamamientos.traza_valor_participacion;
  RAISE EXCEPTION 'traza: lectura directa permitida';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 BEGIN
  PERFORM 1 FROM vec_bolsa_llamamientos.listar_historial_participacion_v1(
   'participacion:sin_autorizacion','persona:sin_autorizacion',NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'traza: consulta sin material V3 permitida';
 EXCEPTION WHEN others THEN IF SQLERRM='traza: consulta sin material V3 permitida' THEN RAISE; END IF; END;
 BEGIN
  PERFORM 1 FROM vec_bolsa_llamamientos.listar_historial_participacion_v1(
   'participacion:x','persona:x','x'::bytea,'{"accion":"bolsa.situacion_participacion.cambiar"}'::bytea,'x'::bytea,'x'::bytea,1,1,'x'::bytea,'x'::bytea,'x'::bytea,'x'::bytea);
  RAISE EXCEPTION 'traza: consulta con material falso permitida';
 EXCEPTION WHEN others THEN IF SQLERRM='traza: consulta con material falso permitida' THEN RAISE; END IF; END;
END $ejecutor$;
ROLLBACK;
