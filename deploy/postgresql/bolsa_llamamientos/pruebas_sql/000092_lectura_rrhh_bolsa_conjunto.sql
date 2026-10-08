\set ON_ERROR_STOP on
-- Ensayo sobre clon PostgreSQL 18 desechable con B90 y B92 instaladas.
-- Los datos sintéticos y los LOGIN de prueba desaparecen con ROLLBACK.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL statement_timeout='60s';
CREATE ROLE vec_b92_ejecutor_test LOGIN;
CREATE ROLE vec_b92_sin_permiso_test LOGIN;
GRANT vec_bolsa_llamamientos_ejecutor TO vec_b92_ejecutor_test;
SET ROLE vec_bolsa_llamamientos_propietario;
SELECT quote_literal(k.bolsa_ref) AS bolsa_sql,
       quote_literal(e.participacion_ref) AS participacion_sql,
       quote_literal(k.categoria_ref) AS categoria_sql
  FROM vec_bolsa_llamamientos.listar_constituciones_v1() k
  JOIN vec_bolsa_llamamientos.constitucion_entrada e
    ON e.instantanea_ref=k.instantanea_ref AND e.version_instantanea=k.version_instantanea
  JOIN vec_bolsa_llamamientos.vinculo_candidato vc
    ON vc.participacion_ref=e.participacion_ref
 WHERE k.estado='vigente' AND k.vigente_desde<=clock_timestamp()
   AND (k.vigente_hasta IS NULL OR k.vigente_hasta>clock_timestamp())
   AND EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(
     k.bolsa_ref,clock_timestamp()) o WHERE o.participacion_ref=e.participacion_ref)
 ORDER BY k.confirmada_en DESC,e.orden LIMIT 1 \gset
RESET ROLE;
SELECT set_config('vec.b92.bolsa',:bolsa_sql,true);
SELECT set_config('vec.b92.participacion',:participacion_sql,true);
SELECT set_config('vec.b92.categoria',:categoria_sql,true);
SELECT set_config('vec.b92.corte_antes',clock_timestamp()::text,true);

SET SESSION AUTHORIZATION vec_b92_ejecutor_test;
DO $lectura$
DECLARE n bigint; esperado bigint; v record; columnas integer; diferencias bigint;
BEGIN
 SELECT count(*),count(DISTINCT orden) INTO n,esperado
   FROM vec_bolsa_llamamientos.leer_situaciones_bolsa_rrhh_v1(
     current_setting('vec.b92.bolsa'),current_setting('vec.b92.corte_antes')::timestamptz);
 IF n<1 OR n<>esperado THEN
  RAISE EXCEPTION 'B92: clave=filas_unicas actual=% esperado=%',n,esperado;
 END IF;
 SELECT count(*) INTO esperado
   FROM vec_bolsa_llamamientos.listar_entradas_constitucion_v1(
    (SELECT instantanea_ref FROM vec_bolsa_llamamientos.leer_situaciones_bolsa_rrhh_v1(
      current_setting('vec.b92.bolsa'),current_setting('vec.b92.corte_antes')::timestamptz) LIMIT 1),
    (SELECT version_instantanea FROM vec_bolsa_llamamientos.leer_situaciones_bolsa_rrhh_v1(
      current_setting('vec.b92.bolsa'),current_setting('vec.b92.corte_antes')::timestamptz) LIMIT 1));
 IF n<>esperado THEN
  RAISE EXCEPTION 'B92: clave=total_bolsa actual=% esperado=%',n,esperado;
 END IF;
 -- B6 y B92 se leen con el mismo corte y dentro de esta transacción: el
 -- conjunto de participaciones y posiciones de acta debe ser idéntico.
 SELECT count(*) INTO diferencias FROM (
  (SELECT orden,participacion_ref FROM vec_bolsa_llamamientos.leer_situaciones_bolsa_rrhh_v1(
    current_setting('vec.b92.bolsa'),current_setting('vec.b92.corte_antes')::timestamptz)
   EXCEPT
   SELECT orden_acta,participacion_ref FROM vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(
    current_setting('vec.b92.bolsa'),current_setting('vec.b92.corte_antes')::timestamptz))
  UNION ALL
  (SELECT orden_acta,participacion_ref FROM vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(
    current_setting('vec.b92.bolsa'),current_setting('vec.b92.corte_antes')::timestamptz)
   EXCEPT
   SELECT orden,participacion_ref FROM vec_bolsa_llamamientos.leer_situaciones_bolsa_rrhh_v1(
    current_setting('vec.b92.bolsa'),current_setting('vec.b92.corte_antes')::timestamptz))
 ) comparacion;
 IF diferencias<>0 THEN
  RAISE EXCEPTION 'B92: clave=consistencia_b6 diferencias=% esperado=0',diferencias;
 END IF;
 SELECT * INTO STRICT v FROM vec_bolsa_llamamientos.leer_situaciones_bolsa_rrhh_v1(
  current_setting('vec.b92.bolsa'),current_setting('vec.b92.corte_antes')::timestamptz)
  WHERE participacion_ref=current_setting('vec.b92.participacion');
 IF v.categoria_ref<>current_setting('vec.b92.categoria') OR v.fila_numero<1 OR v.desde IS NULL
    OR v.cese_pendiente OR v.pendiente_desde IS NOT NULL THEN
  RAISE EXCEPTION 'B92: clave=base_no_pendiente actual=%',row_to_json(v);
 END IF;
 SELECT count(*) INTO n FROM vec_bolsa_llamamientos.leer_situaciones_bolsa_rrhh_v1(
  'bolsa:b92:inexistente',current_setting('vec.b92.corte_antes')::timestamptz);
 IF n<>0 THEN RAISE EXCEPTION 'B92: clave=inexistente actual=% esperado=0',n; END IF;
 -- RETURNS TABLE no registra un tipo compuesto independiente; comprobar el
 -- descriptor desde pg_proc evita depender de la representación del cliente.
 SELECT count(*) INTO columnas FROM pg_catalog.pg_proc p
  CROSS JOIN LATERAL pg_catalog.unnest(p.proallargtypes) WITH ORDINALITY a(tipo,n)
  WHERE p.oid='vec_bolsa_llamamientos.leer_situaciones_bolsa_rrhh_v1(text,timestamptz)'::regprocedure
    AND p.proargmodes[a.n]='t';
 IF columnas<>17 THEN RAISE EXCEPTION 'B92: clave=columnas actual=% esperado=17',columnas; END IF;
END $lectura$;
RESET SESSION AUTHORIZATION;

-- B13 sin B45: conservar fecha base de situación y fecha pendiente separada.
SET ROLE vec_bolsa_llamamientos_propietario;
WITH dato AS (
 SELECT 'origen:b92:prueba:20261008'::text AS origen_ref,
        'llamamiento:b92:prueba:20261008'::text AS llamamiento_ref,
        'evento:ct:contrato-bolsa:'||encode(sha256(convert_to(
         'cese'||chr(31)||'origen:b92:prueba:20261008','UTF8')),'hex') AS evento_ref
), evento AS (
 SELECT d.*,jsonb_build_object('evento_ref',d.evento_ref,'tipo','cese',
  'origen_ref',d.origen_ref,'organizacion_ref','organizacion:desarrollo:dipgra',
  'expediente_ref','expediente:b92:prueba','llamamiento_ref',d.llamamiento_ref) AS cuerpo
 FROM dato d
)
INSERT INTO vec_bolsa_llamamientos.contrato_participacion(
 evento_ref,huella_sha256,evento,origen_ref,origen_creada_en,origen_posicion,
 tipo,organizacion_ref,expediente_ref,llamamiento_ref,participacion_ref,bolsa_ref,
 ocurrido_en,recibido_en)
SELECT evento_ref,encode(sha256(convert_to(cuerpo::text,'UTF8')),'hex'),cuerpo,
 origen_ref,clock_timestamp(),99999992,'cese','organizacion:desarrollo:dipgra',
 'expediente:b92:prueba',llamamiento_ref,current_setting('vec.b92.participacion'),
 current_setting('vec.b92.bolsa'),clock_timestamp(),clock_timestamp()
FROM evento;
RESET ROLE;
SET SESSION AUTHORIZATION vec_b92_ejecutor_test;
DO $pendiente$
DECLARE antes record; despues record; lote record;
BEGIN
 SELECT * INTO STRICT antes FROM vec_bolsa_llamamientos.leer_situaciones_bolsa_rrhh_v1(
  current_setting('vec.b92.bolsa'),current_setting('vec.b92.corte_antes')::timestamptz)
  WHERE participacion_ref=current_setting('vec.b92.participacion');
 SELECT * INTO STRICT despues FROM vec_bolsa_llamamientos.leer_situaciones_bolsa_rrhh_v1(
  current_setting('vec.b92.bolsa'),clock_timestamp())
  WHERE participacion_ref=current_setting('vec.b92.participacion');
 SELECT * INTO STRICT lote FROM vec_bolsa_llamamientos.consultar_estado_cese_bolsa_lote_v2(
  ARRAY[current_setting('vec.b92.participacion')]::text[],clock_timestamp());
 IF antes.desde IS DISTINCT FROM despues.desde OR despues.cese_pendiente IS NOT TRUE
    OR despues.pendiente_desde IS DISTINCT FROM lote.pendiente_desde
    OR despues.cese_fecha_efecto IS NOT NULL OR despues.cese_disponible_desde IS NOT NULL
    OR despues.cese_en_restriccion IS NOT TRUE OR despues.cese_trabajo_cesado IS NOT FALSE THEN
  RAISE EXCEPTION 'B92: clave=pendiente_separado antes=% despues=% lote=%',
   row_to_json(antes),row_to_json(despues),row_to_json(lote);
 END IF;
END $pendiente$;
RESET SESSION AUTHORIZATION;

-- Dos actas de una categoría: solo la más reciente se puede leer. Un empate
-- exacto en los dos criterios B14 falla cerrado, sin elegir una por azar.
SET ROLE vec_bolsa_llamamientos_propietario;
DO $seleccion$
DECLARE n integer; bolsa text; instantanea text; acta text; canon bytea;
 t timestamptz:=clock_timestamp()-interval '2 days';
BEGIN
 FOR n IN 1..2 LOOP
  bolsa:='bolsa:b92:seleccion:'||n;
  instantanea:='instantanea:b92:seleccion:'||n;
  acta:='acta:b92:seleccion:'||n;
  canon:=convert_to(jsonb_build_object('bolsa_ref',bolsa,'orden',n)::text,'UTF8');
  INSERT INTO vec_bolsa_llamamientos.bolsa_constituida
   (bolsa_ref,version,huella_bolsa_sha256,bolsa_canonica,categoria_ref,
    vigente_desde,estado,registrada_en)
   VALUES(bolsa,1,encode(sha256(canon),'hex'),canon,'categoria:b92:seleccion',
    t-interval '1 day','vigente',t+make_interval(hours=>n));
  INSERT INTO vec_bolsa_llamamientos.instantanea_orden_bolsa
   (instantanea_ref,version,huella_instantanea_sha256,instantanea_canonica,
    bolsa_ref,version_bolsa,huella_bolsa_sha256,total_participaciones,
    referida_en,generada_en,registrada_en)
   VALUES(instantanea,1,encode(sha256(canon),'hex'),canon,bolsa,1,
    encode(sha256(canon),'hex'),1,t-interval '1 day',t,t+make_interval(hours=>n));
  INSERT INTO vec_bolsa_llamamientos.constitucion
   (acta_ref,bolsa_ref,version_bolsa,huella_bolsa_sha256,instantanea_ref,
    version_instantanea,huella_instantanea_sha256,categoria_ref,actor_ref,
    confirmada_en,registrada_en)
   VALUES(acta,bolsa,1,encode(sha256(canon),'hex'),instantanea,1,
    encode(sha256(canon),'hex'),'categoria:b92:seleccion','actor:b92:prueba',
    t+make_interval(hours=>n),t+make_interval(hours=>n));
  INSERT INTO vec_bolsa_llamamientos.constitucion_entrada
   (instantanea_ref,version_instantanea,orden,participacion_ref,fila_numero)
   VALUES(instantanea,1,1,'participacion:b92:seleccion:'||n,1);
 END LOOP;
END $seleccion$;
RESET ROLE;
SET SESSION AUTHORIZATION vec_b92_ejecutor_test;
DO $ultima$
DECLARE antigua bigint; actual bigint;
BEGIN
 SELECT count(*) INTO antigua FROM vec_bolsa_llamamientos.leer_situaciones_bolsa_rrhh_v1(
  'bolsa:b92:seleccion:1',clock_timestamp());
 SELECT count(*) INTO actual FROM vec_bolsa_llamamientos.leer_situaciones_bolsa_rrhh_v1(
  'bolsa:b92:seleccion:2',clock_timestamp());
 IF antigua<>0 OR actual<>1 THEN
  RAISE EXCEPTION 'B92: clave=ultima_categoria antigua=% actual=% esperado=0/1',antigua,actual;
 END IF;
END $ultima$;
RESET SESSION AUTHORIZATION;
SET ROLE vec_bolsa_llamamientos_propietario;
DO $empate$
DECLARE canon bytea; t timestamptz;
BEGIN
 SELECT confirmada_en INTO STRICT t FROM vec_bolsa_llamamientos.constitucion
  WHERE acta_ref='acta:b92:seleccion:2';
 canon:=convert_to('{"bolsa_ref":"bolsa:b92:seleccion:3"}','UTF8');
 INSERT INTO vec_bolsa_llamamientos.bolsa_constituida
  (bolsa_ref,version,huella_bolsa_sha256,bolsa_canonica,categoria_ref,
   vigente_desde,estado,registrada_en)
  VALUES('bolsa:b92:seleccion:3',1,encode(sha256(canon),'hex'),canon,
   'categoria:b92:seleccion',t-interval '1 day','vigente',t);
 INSERT INTO vec_bolsa_llamamientos.instantanea_orden_bolsa
  (instantanea_ref,version,huella_instantanea_sha256,instantanea_canonica,
   bolsa_ref,version_bolsa,huella_bolsa_sha256,total_participaciones,
   referida_en,generada_en,registrada_en)
  VALUES('instantanea:b92:seleccion:3',1,encode(sha256(canon),'hex'),canon,
   'bolsa:b92:seleccion:3',1,encode(sha256(canon),'hex'),1,t-interval '1 day',t,t);
 INSERT INTO vec_bolsa_llamamientos.constitucion
  (acta_ref,bolsa_ref,version_bolsa,huella_bolsa_sha256,instantanea_ref,
   version_instantanea,huella_instantanea_sha256,categoria_ref,actor_ref,
   confirmada_en,registrada_en)
  VALUES('acta:b92:seleccion:3','bolsa:b92:seleccion:3',1,
   encode(sha256(canon),'hex'),'instantanea:b92:seleccion:3',1,
   encode(sha256(canon),'hex'),'categoria:b92:seleccion','actor:b92:prueba',t,t);
END $empate$;
RESET ROLE;
SET SESSION AUTHORIZATION vec_b92_ejecutor_test;
DO $rechazo_empate$
DECLARE rechazado boolean:=false;
BEGIN
 BEGIN
  PERFORM 1 FROM vec_bolsa_llamamientos.leer_situaciones_bolsa_rrhh_v1(
   'bolsa:b92:seleccion:2',clock_timestamp());
 EXCEPTION WHEN SQLSTATE '55000' THEN
  rechazado:=true;
 END;
 IF NOT rechazado THEN RAISE EXCEPTION 'B92: clave=empate_no_rechazado'; END IF;
END $rechazo_empate$;
RESET SESSION AUTHORIZATION;

-- La ACL se prueba como sesión de un LOGIN ajeno, no con un cálculo de
-- privilegios desde la sesión administrativa.
SET SESSION AUTHORIZATION vec_b92_sin_permiso_test;
DO $rechazo_permiso$
DECLARE rechazado boolean:=false;
BEGIN
 BEGIN
  PERFORM 1 FROM vec_bolsa_llamamientos.leer_situaciones_bolsa_rrhh_v1(
   current_setting('vec.b92.bolsa'),clock_timestamp());
 EXCEPTION WHEN SQLSTATE '42501' THEN
  rechazado:=true;
 END;
 IF NOT rechazado THEN RAISE EXCEPTION 'B92: clave=login_ajeno_no_rechazado'; END IF;
END $rechazo_permiso$;
RESET SESSION AUTHORIZATION;

-- B7 admite instantáneas mayores. El lector debe contar y rechazar antes de
-- agregar referencias o llamar al núcleo B90, cuya cota es 20.000.
SET ROLE vec_bolsa_llamamientos_propietario;
DO $demasiadas_entradas$
DECLARE canon bytea:=convert_to('{"bolsa_ref":"bolsa:b92:20001","total_participaciones":20001}','UTF8');
 t timestamptz:=clock_timestamp()-interval '1 day';
BEGIN
 INSERT INTO vec_bolsa_llamamientos.bolsa_constituida
  (bolsa_ref,version,huella_bolsa_sha256,bolsa_canonica,categoria_ref,
   vigente_desde,estado,registrada_en)
  VALUES('bolsa:b92:20001',1,encode(sha256(canon),'hex'),canon,
   'categoria:b92:20001',t,'vigente',t);
 INSERT INTO vec_bolsa_llamamientos.instantanea_orden_bolsa
  (instantanea_ref,version,huella_instantanea_sha256,instantanea_canonica,
   bolsa_ref,version_bolsa,huella_bolsa_sha256,total_participaciones,
   referida_en,generada_en,registrada_en)
  VALUES('instantanea:b92:20001',1,encode(sha256(canon),'hex'),canon,
   'bolsa:b92:20001',1,encode(sha256(canon),'hex'),20001,t,t,t);
 INSERT INTO vec_bolsa_llamamientos.constitucion
  (acta_ref,bolsa_ref,version_bolsa,huella_bolsa_sha256,instantanea_ref,
   version_instantanea,huella_instantanea_sha256,categoria_ref,actor_ref,
   confirmada_en,registrada_en)
  VALUES('acta:b92:20001','bolsa:b92:20001',1,encode(sha256(canon),'hex'),
   'instantanea:b92:20001',1,encode(sha256(canon),'hex'),
   'categoria:b92:20001','actor:b92:prueba',t,t);
 INSERT INTO vec_bolsa_llamamientos.constitucion_entrada
  (instantanea_ref,version_instantanea,orden,participacion_ref,fila_numero)
  SELECT 'instantanea:b92:20001',1,g,'participacion:b92:20001:'||g,g
    FROM generate_series(1,20001) g;
END $demasiadas_entradas$;
RESET ROLE;
SET SESSION AUTHORIZATION vec_b92_ejecutor_test;
DO $rechazo_cota$
DECLARE rechazado boolean:=false; detalle text;
BEGIN
 BEGIN
  PERFORM 1 FROM vec_bolsa_llamamientos.leer_situaciones_bolsa_rrhh_v1(
   'bolsa:b92:20001',clock_timestamp());
 EXCEPTION WHEN SQLSTATE '55000' THEN
  GET STACKED DIAGNOSTICS detalle=MESSAGE_TEXT;
  rechazado:=detalle LIKE 'B92: clave=entradas_bolsa bolsa=bolsa:b92:20001 actual=20001 esperado=20001 maximo=20000';
 END;
 IF NOT rechazado THEN
  RAISE EXCEPTION 'B92: clave=cota_no_rechazada detalle=%',coalesce(detalle,'sin_error');
 END IF;
END $rechazo_cota$;
RESET SESSION AUTHORIZATION;

DO $acl$
DECLARE f regprocedure:='vec_bolsa_llamamientos.leer_situaciones_bolsa_rrhh_v1(text,timestamptz)'::regprocedure;
BEGIN
 IF NOT has_function_privilege('vec_b92_ejecutor_test',f,'EXECUTE')
    OR has_function_privilege('vec_b92_sin_permiso_test',f,'EXECUTE')
    OR has_function_privilege('vec_bolsa_llamamientos_relevo_cese',f,'EXECUTE')
    OR EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) x
                WHERE p.oid=f AND x.grantee=0) THEN
  RAISE EXCEPTION 'B92: clave=acl_lectura_bolsa';
 END IF;
END $acl$;
ROLLBACK;
