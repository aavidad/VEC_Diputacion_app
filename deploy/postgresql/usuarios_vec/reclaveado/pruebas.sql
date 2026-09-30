\set ON_ERROR_STOP on
-- Ejecutar sembrar=1 DESPUÉS de Usuarios010 y ANTES de Usuarios012/013.
-- Ejecutar sembrar=0 después de Usuarios013 sobre la MISMA base desechable.
-- Direcciones y huellas sintéticas; no acredita corrección criptográfica Go.
\if :sembrar
BEGIN;
SET LOCAL timezone='UTC';
INSERT INTO vec_usuarios_correos_externo.correos_conjunto VALUES
 ('per_sql013_prueba_sintetica_001',3,'clave:kms:desarrollo:usuarios-correos-igualdad:v1','2026-09-30T10:00:00Z');
INSERT INTO vec_usuarios_correos_externo.correos_direccion
 (persona_ref,correo_ref,version_sobre,clave_sobre_ref,clave_igualdad_ref,nonce,cifrado,huella_igualdad,estado,activo,creado_en,verificado_en,retirado_en) VALUES
 ('per_sql013_prueba_sintetica_001','correo:00000000000000000000000000000001',1,
 'clave:kms:desarrollo:usuarios-correos-cifrado:v1','clave:kms:desarrollo:usuarios-correos-igualdad:v1',
 decode(repeat('01',12),'hex'),decode(repeat('11',32),'hex'),decode(repeat('21',32),'hex'),
 'verificado',true,'2026-09-30T09:00:00Z','2026-09-30T09:01:00Z',NULL),
 ('per_sql013_prueba_sintetica_001','correo:00000000000000000000000000000002',1,
 'clave:kms:desarrollo:usuarios-correos-cifrado:v1','clave:kms:desarrollo:usuarios-correos-igualdad:v1',
 decode(repeat('02',12),'hex'),decode(repeat('12',32),'hex'),decode(repeat('22',32),'hex'),
 'retirado',false,'2026-09-30T09:02:00Z',NULL,'2026-09-30T09:03:00Z'),
 ('per_sql013_prueba_sintetica_001','correo:00000000000000000000000000000003',1,
 'clave:kms:desarrollo:usuarios-correos-cifrado:v1','clave:kms:desarrollo:usuarios-correos-igualdad:v1',
 decode(repeat('03',12),'hex'),decode(repeat('13',32),'hex'),decode(repeat('23',32),'hex'),
 'pendiente',false,'2026-09-30T09:04:00Z',NULL,NULL);
INSERT INTO vec_usuarios_correos_externo.correos_desafio VALUES
 ('per_sql013_prueba_sintetica_001','correo:00000000000000000000000000000001','desafio:00000000000000000000000000000001',
 decode(repeat('31',32),'hex'),'clave:kms:desarrollo:usuarios-correos-codigo:v1','2026-10-01T09:00:00Z',
 'usado',0,'2026-09-30T09:00:00Z'),
 ('per_sql013_prueba_sintetica_001','correo:00000000000000000000000000000003','desafio:00000000000000000000000000000003',
 decode(repeat('33',32),'hex'),'clave:kms:desarrollo:usuarios-correos-codigo:v1','2026-10-01T09:04:00Z',
 'pendiente',1,'2026-09-30T09:04:00Z');
INSERT INTO vec_usuarios_correos_externo.correos_historia VALUES
 ('per_sql013_prueba_sintetica_001',1,'vec.correos.anadir','correo:00000000000000000000000000000001',
 'pendiente',false,NULL,'correo_recibo:00000000000000000000000000000001','decision:sql013:1','auditoria:sql013:1','2026-09-30T09:00:00Z');
INSERT INTO vec_usuarios_correos_externo.correos_recibo VALUES
 ('per_sql013_prueba_sintetica_001','operacion:sql013:historica:1','clave:kms:desarrollo:usuarios-correos-semantica:v1',repeat('a',64),
 'correo_recibo:00000000000000000000000000000001','vec.correos.anadir','correo:00000000000000000000000000000001',1,
 'decision:sql013:1','auditoria:sql013:1',repeat('b',64),'2026-09-30T09:00:00Z');
INSERT INTO vec_usuarios_correos_externo.correos_intento_fallido VALUES
 ('per_sql013_prueba_sintetica_001','correo:00000000000000000000000000000003','desafio:00000000000000000000000000000003',1,
 'operacion:sql013:historica:fallido','clave:kms:desarrollo:usuarios-correos-semantica:v1',repeat('c',64),
 'decision:sql013:fallido','auditoria:sql013:fallido',repeat('d',64),'2026-09-30T09:05:00Z');
INSERT INTO vec_usuarios_correos_externo.correos_envio VALUES
 ('correo_envio:00000000000000000000000000000001','per_sql013_prueba_sintetica_001',
 'correo:00000000000000000000000000000001','externa_personal','verificacion',
 'desafio:00000000000000000000000000000001','correo_recibo:00000000000000000000000000000001',
 repeat('e',64),'aceptado','2026-09-30T09:00:00Z','2026-09-30T09:00:01Z');
COMMIT;
\else
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
CREATE FUNCTION pg_temp.exigir(p boolean,p_m text) RETURNS void LANGUAGE plpgsql AS $f$
BEGIN IF p IS DISTINCT FROM true THEN RAISE EXCEPTION 'FALLO: %',p_m; END IF; END $f$;
SET SESSION AUTHORIZATION vec_externo_preflight_v3_desarrollo;
SELECT pg_temp.exigir(NOT vec_usuarios_correos_externo.poblacion_sin_claves_ajenas_v1(),'preflight antes');
RESET SESSION AUTHORIZATION;
CREATE TEMP TABLE prueba_estado ON COMMIT DROP AS
SELECT vec_usuarios_correos_reclaveado.snapshot_reclaveado_persona_v1('per_sql013_prueba_sintetica_001') AS snapshot;
ALTER TABLE prueba_estado ADD COLUMN material jsonb;
UPDATE prueba_estado SET material=jsonb_build_object(
 'clave_sobre_ref','clave:kms:desarrollo:usuarios-correos-externo-cifrado:v1',
 'clave_igualdad_ref','clave:kms:desarrollo:usuarios-correos-externo-igualdad:v1',
 'clave_codigo_ref','clave:kms:desarrollo:usuarios-correos-externo-codigo:v1',
 'clave_semantica_ref','clave:kms:desarrollo:usuarios-correos-externo-semantica:v1',
 'direcciones',(SELECT jsonb_agg(jsonb_build_object('correo_ref',d->>'correo_ref','version_sobre',d->'version_sobre',
 'nonce_hex',repeat('04',12),'cifrado_hex',repeat('14',32),'huella_igualdad_hex',
 encode(sha256(convert_to(d->>'correo_ref','UTF8')),'hex')) ORDER BY d->>'correo_ref')
 FROM jsonb_array_elements(snapshot#>'{filas,correos_direccion}') d));
SELECT pg_temp.exigir(vec_usuarios_correos_reclaveado.recuperar_reclaveado_persona_v1(
 'per_sql013_prueba_sintetica_001',snapshot->>'preimagen_sha256','lote:sql013:prueba:001','aprobacion:sql013:prueba:001') IS NULL,'recuperación previa') FROM prueba_estado;
DO $cas$ BEGIN
 BEGIN
  PERFORM vec_usuarios_correos_reclaveado.aplicar_reclaveado_persona_v1('per_sql013_prueba_sintetica_001',
   repeat('f',64),(SELECT material FROM prueba_estado),'lote:sql013:prueba:001','aprobacion:sql013:prueba:001');
  RAISE EXCEPTION 'FALLO: CAS obsoleto aceptado';
 EXCEPTION WHEN serialization_failure THEN NULL; END;
END $cas$;
DO $vivos$ BEGIN
 BEGIN
  INSERT INTO vec_usuarios_correos_externo.correos_envio
  SELECT 'correo_envio:00000000000000000000000000000009',persona_ref,correo_ref,superficie,tipo,
   desafio_ref,recibo_ref,repeat('9',64),'reservado',creado_en,NULL
  FROM vec_usuarios_correos_externo.correos_envio WHERE envio_ref='correo_envio:00000000000000000000000000000001';
  PERFORM vec_usuarios_correos_reclaveado.snapshot_reclaveado_persona_v1('per_sql013_prueba_sintetica_001');
  RAISE EXCEPTION 'FALLO: envío reservado aceptado';
 EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL; END;
 BEGIN
  INSERT INTO vec_usuarios_correos_externo.correos_contexto
   (xid,backend_pid,sesion,superficie,persona_ref,modo,accion,decision_ref,auditoria_ref,consumo_huella_sha256)
  VALUES(pg_current_xact_id(),pg_backend_pid(),session_user,'externa_personal','per_sql013_prueba_sintetica_001',
   'consultar','vec.correos.consultar','decision:sql013:contexto','auditoria:sql013:contexto',repeat('8',64));
  PERFORM vec_usuarios_correos_reclaveado.snapshot_reclaveado_persona_v1('per_sql013_prueba_sintetica_001');
  RAISE EXCEPTION 'FALLO: contexto vivo aceptado';
 EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL; END;
END $vivos$;
SAVEPOINT simulacion;
SELECT pg_temp.exigir((vec_usuarios_correos_reclaveado.aplicar_reclaveado_persona_v1(
 'per_sql013_prueba_sintetica_001',snapshot->>'preimagen_sha256',material,'lote:sql013:prueba:001',
 'aprobacion:sql013:prueba:001')->>'replay')='false','efecto simulación') FROM prueba_estado;
ROLLBACK TO SAVEPOINT simulacion;
SELECT pg_temp.exigir(NOT EXISTS(SELECT 1 FROM vec_usuarios_correos_reclaveado.recibo WHERE persona_ref='per_sql013_prueba_sintetica_001'),'rollback sin recibo');
SELECT pg_temp.exigir(vec_usuarios_correos_reclaveado.snapshot_reclaveado_persona_v1(
 'per_sql013_prueba_sintetica_001')=snapshot,'rollback sin cambios') FROM prueba_estado;
DO $sin_evento$ BEGIN
 BEGIN
  UPDATE vec_usuarios_correos_externo.correos_direccion SET clave_sobre_ref='clave:kms:desarrollo:usuarios-correos-externo-cifrado:v1',
  clave_igualdad_ref='clave:kms:desarrollo:usuarios-correos-externo-igualdad:v1'
  WHERE persona_ref='per_sql013_prueba_sintetica_001';
  RAISE EXCEPTION 'FALLO: conversión sin ledger aceptada';
 EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL; END;
END $sin_evento$;
ALTER TABLE prueba_estado ADD COLUMN recibo jsonb;
UPDATE prueba_estado SET recibo=vec_usuarios_correos_reclaveado.aplicar_reclaveado_persona_v1(
 'per_sql013_prueba_sintetica_001',snapshot->>'preimagen_sha256',material,'lote:sql013:prueba:001','aprobacion:sql013:prueba:001');
SELECT pg_temp.exigir((SELECT version FROM vec_usuarios_correos_externo.correos_conjunto WHERE persona_ref='per_sql013_prueba_sintetica_001')=3,'sin versión negocio nueva');
SELECT pg_temp.exigir((SELECT estado='sustituido' AND intentos=1 AND clave_ref='clave:kms:desarrollo:usuarios-correos-codigo:v1'
 FROM vec_usuarios_correos_externo.correos_desafio WHERE desafio_ref='desafio:00000000000000000000000000000003'),'desafío invalidado y HMAC original');
SELECT pg_temp.exigir((SELECT estado='retirado' AND clave_sobre_ref='clave:kms:desarrollo:usuarios-correos-externo-cifrado:v1'
 FROM vec_usuarios_correos_externo.correos_direccion WHERE correo_ref='correo:00000000000000000000000000000002'),'también retiradas convertidas');
SELECT pg_temp.exigir(vec_usuarios_correos_reclaveado.recuperar_reclaveado_persona_v1(
 'per_sql013_prueba_sintetica_001',snapshot->>'preimagen_sha256','lote:sql013:prueba:001','aprobacion:sql013:prueba:001')
 =recibo||jsonb_build_object('replay',true),'mismo recibo recuperación') FROM prueba_estado;
SELECT pg_temp.exigir(vec_usuarios_correos_reclaveado.aplicar_reclaveado_persona_v1(
 'per_sql013_prueba_sintetica_001',snapshot->>'preimagen_sha256',material,'lote:sql013:prueba:001','aprobacion:sql013:prueba:001')
 =recibo||jsonb_build_object('replay',true),'aplicar replay sin duplicar') FROM prueba_estado;
SELECT pg_temp.exigir((SELECT count(*) FROM vec_usuarios_correos_reclaveado.recibo WHERE persona_ref='per_sql013_prueba_sintetica_001')=1,'un solo recibo');
SELECT pg_temp.exigir(vec_usuarios_correos_reclaveado.snapshot_reclaveado_persona_v1('per_sql013_prueba_sintetica_001')#>'{filas,correos_recibo}'=snapshot#>'{filas,correos_recibo}','recibos HMAC sin reescritura') FROM prueba_estado;
SELECT pg_temp.exigir(vec_usuarios_correos_reclaveado.snapshot_reclaveado_persona_v1('per_sql013_prueba_sintetica_001')#>'{filas,correos_intento_fallido}'=snapshot#>'{filas,correos_intento_fallido}','intentos HMAC sin reescritura') FROM prueba_estado;
DO $conflictos$ BEGIN
 BEGIN
  PERFORM vec_usuarios_correos_reclaveado.recuperar_reclaveado_persona_v1('per_sql013_prueba_sintetica_001',repeat('f',64),'lote:sql013:prueba:001','aprobacion:sql013:prueba:001');
  RAISE EXCEPTION 'FALLO: replay cambiado aceptado';
 EXCEPTION WHEN SQLSTATE 'P1409' THEN NULL; END;
 BEGIN
  UPDATE vec_usuarios_correos_reclaveado.fila SET postimagen_sha256=repeat('f',64);
  RAISE EXCEPTION 'FALLO: ledger mutable';
 EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL; END;
 BEGIN
  INSERT INTO vec_usuarios_correos_externo.correos_desafio
  SELECT persona_ref,correo_ref,'desafio:00000000000000000000000000000004',huella_codigo,clave_ref,
   vence_en,'sustituido',intentos,creado_en FROM vec_usuarios_correos_externo.correos_desafio WHERE desafio_ref='desafio:00000000000000000000000000000003';
  RAISE EXCEPTION 'FALLO: nuevo desafío con clave antigua aceptado';
 EXCEPTION WHEN check_violation THEN NULL; END;
END $conflictos$;
SET LOCAL timezone='Europe/Madrid';
SET SESSION AUTHORIZATION vec_externo_preflight_v3_desarrollo;
SELECT pg_temp.exigir(vec_usuarios_correos_externo.poblacion_sin_claves_ajenas_v1(),'preflight tras conversión en otro TZ');
SELECT pg_temp.exigir(NOT has_table_privilege(session_user,'vec_usuarios_correos_externo.correos_direccion','SELECT'),'sin acceso directo preflight');
RESET SESSION AUTHORIZATION;
SELECT pg_temp.exigir(NOT has_function_privilege('vec_usuarios_ejecutor_externo','vec_usuarios_correos_reclaveado.snapshot_reclaveado_persona_v1(text)','EXECUTE'),'offline inaccesible al portal');
SELECT pg_temp.exigir(NOT has_table_privilege('vec_usuarios_correos_externo_propietario','vec_usuarios_correos_reclaveado.fila','SELECT,INSERT'),'ledger inaccesible al propietario');
-- La operación ordinaria sigue funcionando con su propietario (sin SELECT
-- al ledger). La política temporal sustituye solo el contexto en esta prueba.
SAVEPOINT propietario_ordinario;
CREATE POLICY prueba_propietario ON vec_usuarios_correos_externo.correos_direccion
 FOR ALL TO vec_usuarios_correos_externo_propietario USING(true) WITH CHECK(true);
SET LOCAL ROLE vec_usuarios_correos_externo_propietario;
UPDATE vec_usuarios_correos_externo.correos_direccion SET activo=false
 WHERE correo_ref='correo:00000000000000000000000000000001';
RESET ROLE;
SET SESSION AUTHORIZATION vec_externo_preflight_v3_desarrollo;
SELECT pg_temp.exigir(vec_usuarios_correos_externo.poblacion_sin_claves_ajenas_v1(),'evolución ordinaria no bloquea preflight');
RESET SESSION AUTHORIZATION;
ROLLBACK TO SAVEPOINT propietario_ordinario;
SAVEPOINT alteracion;
ALTER TABLE vec_usuarios_correos_externo.correos_historia ADD COLUMN alteracion_sintetica integer DEFAULT 1;
SET SESSION AUTHORIZATION vec_externo_preflight_v3_desarrollo;
SELECT pg_temp.exigir(NOT vec_usuarios_correos_externo.poblacion_sin_claves_ajenas_v1(),'hash exacto detecta historia distinta');
RESET SESSION AUTHORIZATION;
ROLLBACK TO SAVEPOINT alteracion;
COMMIT;
\echo ENSAYO-OK SQL013: CAS, rollback, sellado exacto, histórico intacto, desafíos inválidos, replay y ACL
\endif
