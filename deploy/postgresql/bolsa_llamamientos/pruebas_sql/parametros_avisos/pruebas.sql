-- Bolsa 000041: conducta con la política por defecto, publicación desde el
-- catálogo, avisos v2, marcas, exclusión en el llamamiento y negativos.
-- Se ejecuta como superusuario; cada consulta de la aplicación, con el rol
-- ejecutor. Cada comprobación deja un NOTICE «OK …» o aborta.
\set ON_ERROR_STOP on
SET timezone = 'UTC';

-- 1. Versión 1: la bandeja v2 coincide con la v1 (saltos y tres años).
DO $p$
DECLARE v1 text; v2 text;
BEGIN
 SELECT string_agg(tipo||'|'||referencia||'|'||fecha, ',' ORDER BY referencia) INTO v1 FROM vec_bolsa_llamamientos.consultar_avisos_rrhh_v1(now());
 SELECT string_agg(tipo||'|'||referencia||'|'||fecha, ',' ORDER BY referencia) INTO v2 FROM vec_bolsa_llamamientos.consultar_avisos_rrhh_v2(now());
 IF v1 IS DISTINCT FROM v2 OR v1 IS NULL THEN RAISE EXCEPTION 'v2 por defecto distinta de v1: % / %', v1, v2; END IF;
 IF (SELECT count(*) FROM vec_bolsa_llamamientos.consultar_avisos_rrhh_v2(now()) WHERE tipo='tres_anos') <> 2 THEN
  RAISE EXCEPTION 'por defecto se esperaban dos avisos de tres años (cumplido y próximo)'; END IF;
 RAISE NOTICE 'OK versión 1 reproduce la bandeja anterior';
END $p$;

-- 2. Marcas con la versión 1: solo «en revisión».
DO $p$
DECLARE r text;
BEGIN
 SELECT string_agg(participacion_ref||'='||coalesce(presta_servicios,'-')||'/'||coalesce(en_revision,'-')||'/'||coalesce(encadenamiento_dias::text,'-'), ',' ORDER BY participacion_ref)
   INTO r FROM vec_bolsa_llamamientos.consultar_marcas_participaciones_v1('bolsa:pa:1', now());
 IF r IS DISTINCT FROM 'part:pa:2=-/renuncia_pendiente/-,part:pa:4=-/solicitud_pendiente/-' THEN RAISE EXCEPTION 'marcas por defecto: %', r; END IF;
 RAISE NOTICE 'OK marcas por defecto: renuncia y solicitud pendientes';
END $p$;

-- 3. Publicación como la aplicación; replay idéntico reutiliza la versión.
SET ROLE vec_bolsa_llamamientos_ejecutor;
DO $p$
DECLARE v record;
BEGIN
 SELECT * INTO v FROM prueba_pa.publicar('vec.bolsa.reglas:1:b17', 36, 10, 18, 24, 'aviso', ARRAY['pendiente_incorporacion','trabajando']);
 IF v.version <> 2 OR v.reutilizada THEN RAISE EXCEPTION 'publicación: %', v; END IF;
 SELECT * INTO v FROM prueba_pa.publicar('vec.bolsa.reglas:1:b17', 36, 10, 18, 24, 'aviso', ARRAY['trabajando','pendiente_incorporacion']);
 IF v.version <> 2 OR NOT v.reutilizada THEN RAISE EXCEPTION 'replay: %', v; END IF;
 IF (SELECT presta_servicios_situaciones FROM vec_bolsa_llamamientos.consultar_politica_avisos_bolsa_v1()) <> ARRAY['trabajando','pendiente_incorporacion'] THEN
  RAISE EXCEPTION 'situaciones no canónicas'; END IF;
 RAISE NOTICE 'OK publicación, orden canónico y replay';
END $p$;

-- 4. Bandeja v2 con la versión publicada.
DO $p$
DECLARE n integer; d jsonb;
BEGIN
 SELECT count(*) INTO n FROM vec_bolsa_llamamientos.consultar_avisos_rrhh_v2(now()) WHERE tipo='tres_anos';
 IF n <> 1 THEN RAISE EXCEPTION 'con diez días de antelación se esperaba un aviso de tres años, hay %', n; END IF;
 IF (SELECT detalle->>'participacion_ref' FROM vec_bolsa_llamamientos.consultar_avisos_rrhh_v2(now()) WHERE tipo='tres_anos') <> 'part:pa:5' THEN
  RAISE EXCEPTION 'aviso de tres años de otra participación'; END IF;
 SELECT count(*) INTO n FROM vec_bolsa_llamamientos.consultar_avisos_rrhh_v2(now()) WHERE tipo='encadenamiento';
 IF n <> 1 THEN RAISE EXCEPTION 'se esperaba un aviso de encadenamiento, hay %', n; END IF;
 SELECT detalle INTO d FROM vec_bolsa_llamamientos.consultar_avisos_rrhh_v2(now()) WHERE tipo='encadenamiento';
 IF d->>'participacion_ref' <> 'part:pa:3' OR (d->>'contratos')::int <> 2 OR (d->>'umbral_meses')::int <> 18 OR (d->>'ventana_meses')::int <> 24
    OR (d->>'dias_acumulados')::int NOT BETWEEN 570 AND 590 OR d ? 'candidato_ref' THEN
  RAISE EXCEPTION 'detalle de encadenamiento: %', d; END IF;
 RAISE NOTICE 'OK bandeja v2: plazo publicado y encadenamiento sin contar solapes';
END $p$;

-- 5. Marcas con la versión publicada.
DO $p$
DECLARE r text;
BEGIN
 SELECT string_agg(participacion_ref||'='||coalesce(presta_servicios,'-')||'/'||coalesce(en_revision,'-')||'/'||CASE WHEN encadenamiento_dias IS NULL THEN '-' ELSE 'enc' END, ',' ORDER BY participacion_ref)
   INTO r FROM vec_bolsa_llamamientos.consultar_marcas_participaciones_v1('bolsa:pa:1', now());
 IF r IS DISTINCT FROM 'part:pa:1=aviso/-/-,part:pa:2=aviso/renuncia_pendiente/-,part:pa:3=-/-/enc,part:pa:4=-/solicitud_pendiente/-' THEN
  RAISE EXCEPTION 'marcas publicadas: %', r; END IF;
 IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.consultar_marcas_participaciones_v1('bolsa:pa:2', now()) WHERE participacion_ref NOT IN ('part:pa:5','part:pa:6')) THEN
  RAISE EXCEPTION 'marcas de otra bolsa'; END IF;
 RAISE NOTICE 'OK marcas: servicios en otra bolsa, revisión y encadenamiento';
END $p$;
RESET ROLE;

-- 6. La renuncia deja de estar en revisión cuando RRHH la refleja.
BEGIN;
SET LOCAL session_replication_role = replica;
INSERT INTO vec_bolsa_llamamientos.situacion_participacion(participacion_ref,situacion,desde,motivo,actor,registrada_en,clave_idempotencia,recibo_ref)
VALUES ('part:pa:2','renuncia',now(),'Renuncia confirmada','per_actoractoractoractoractor',now(),'pa:2:r','recibo:pa:2:r');
SET LOCAL session_replication_role = origin;
DO $p$ BEGIN
 IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.consultar_marcas_participaciones_v1('bolsa:pa:1', now()) WHERE en_revision='renuncia_pendiente') THEN
  RAISE EXCEPTION 'renuncia reflejada sigue en revisión'; END IF;
 RAISE NOTICE 'OK renuncia reflejada sale de revisión';
END $p$;
ROLLBACK;

-- 7. «Excluir»: la base rechaza el llamamiento con quien ya presta servicios.
SET ROLE vec_bolsa_llamamientos_ejecutor;
SELECT * FROM prueba_pa.publicar('vec.bolsa.reglas:1:b16', 36, 10, 18, 24, 'excluir', ARRAY['trabajando']);
RESET ROLE;
DO $p$
BEGIN
 BEGIN
  INSERT INTO vec_bolsa_llamamientos.llamamiento_emitido(llamamiento_ref,recibo_ref,bolsa_ref,actor_ref,clave_idempotencia,participaciones,configuracion,huella_comando_sha256,huella_finalizacion,estado,emitido_en,decision_ref)
  VALUES ('llamamiento:'||repeat('1',64),'recibo:llamamiento:'||repeat('1',64),'bolsa:pa:1','per_actoractoractoractoractor','pa:emision:x','["part:pa:3","part:pa:1"]','{}',repeat('b',64),decode(repeat('0c',32),'hex'),'emision_reservada',now(),'decision:pa:x');
  RAISE EXCEPTION 'llamamiento con quien presta servicios aceptado';
 EXCEPTION WHEN invalid_parameter_value THEN NULL;
 END;
 INSERT INTO vec_bolsa_llamamientos.llamamiento_emitido(llamamiento_ref,recibo_ref,bolsa_ref,actor_ref,clave_idempotencia,participaciones,configuracion,huella_comando_sha256,huella_finalizacion,estado,emitido_en,decision_ref)
 VALUES ('llamamiento:'||repeat('2',64),'recibo:llamamiento:'||repeat('2',64),'bolsa:pa:1','per_actoractoractoractoractor','pa:emision:y','["part:pa:3"]','{}',repeat('b',64),decode(repeat('0c',32),'hex'),'emision_reservada',now(),'decision:pa:y');
 RAISE NOTICE 'OK exclusión en la base: rechaza a quien presta servicios y admite al resto';
END $p$;

-- 8. Negativos de publicación y ACL.
SET ROLE vec_bolsa_llamamientos_ejecutor;
DO $p$
DECLARE casos text[][] := ARRAY[
  ['situación disponible', $$SELECT * FROM prueba_pa.publicar('r', 36, 10, NULL, NULL, 'aviso', ARRAY['disponible'])$$],
  ['situación repetida', $$SELECT * FROM prueba_pa.publicar('r', 36, 10, NULL, NULL, 'aviso', ARRAY['trabajando','trabajando'])$$],
  ['umbral mayor que ventana', $$SELECT * FROM prueba_pa.publicar('r', 36, 10, 30, 24, NULL, NULL)$$],
  ['umbral sin ventana', $$SELECT * FROM prueba_pa.publicar('r', 36, 10, 18, NULL, NULL, NULL)$$],
  ['plazo sin antelación', $$SELECT * FROM prueba_pa.publicar('r', 36, NULL, NULL, NULL, NULL, NULL)$$],
  ['modo desconocido', $$SELECT * FROM prueba_pa.publicar('r', 36, 10, NULL, NULL, 'bloquear', ARRAY['trabajando'])$$],
  ['modo sin situaciones', $$SELECT * FROM prueba_pa.publicar('r', 36, 10, NULL, NULL, 'aviso', NULL)$$],
  ['plazo fuera de rango', $$SELECT * FROM prueba_pa.publicar('r', 0, 10, NULL, NULL, NULL, NULL)$$]];
 i integer;
BEGIN
 FOR i IN 1..array_length(casos, 1) LOOP
  BEGIN
   EXECUTE casos[i][2];
   RAISE EXCEPTION 'aceptado: %', casos[i][1];
  EXCEPTION WHEN invalid_parameter_value THEN NULL;
  END;
 END LOOP;
 RAISE NOTICE 'OK negativos de publicación (%)', array_length(casos, 1);
END $p$;
DO $p$
BEGIN
 BEGIN PERFORM vec_bolsa_llamamientos.situaciones_en_v1(now()); RAISE EXCEPTION 'ejecutor alcanza función interna';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 BEGIN PERFORM vec_bolsa_llamamientos.politica_avisos_bolsa_vigente(); RAISE EXCEPTION 'ejecutor alcanza la política directa';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 BEGIN PERFORM 1 FROM vec_bolsa_llamamientos.politica_avisos_bolsa; RAISE EXCEPTION 'ejecutor lee la tabla';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 RAISE NOTICE 'OK el ejecutor solo alcanza las funciones públicas';
END $p$;
RESET ROLE;

-- 9. Historia de solo adición.
DO $p$
BEGIN
 BEGIN
  SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
  UPDATE vec_bolsa_llamamientos.politica_avisos_bolsa SET continuado_meses = 12 WHERE version = 1;
  RAISE EXCEPTION 'política mutable';
 EXCEPTION WHEN OTHERS THEN
  IF SQLERRM LIKE 'política mutable%' THEN RAISE; END IF;
 END;
 RAISE NOTICE 'OK historia de la política inmutable';
END $p$;
