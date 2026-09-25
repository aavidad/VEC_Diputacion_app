\set ON_ERROR_STOP on
-- Prueba de 000032 (política versionada de transiciones B2). Requiere una
-- réplica sintética con 000012, 000019 y 000032 instaladas y al menos una
-- participación constituida en «disponible». Todo ocurre en una transacción
-- que se deshace al final: la base queda como estaba.
--
-- Doble de prueba: dentro de esta transacción se sustituye el consumidor de
-- autorización atestada V3 por uno que acepta el material sintético. Lo que se
-- prueba aquí es la tabla de transiciones, no la autorización, que tiene sus
-- propias pruebas. El ROLLBACK final restaura la función real.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_situacion_participacion_v3_atestada(
 p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric,
 p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
RETURNS TABLE(decision_ref text, efecto_ref text, huella_efecto_sha256 text, consumo_huella_sha256 text, auditoria_ref text, consumida_en timestamptz, consumo_nuevo boolean)
LANGUAGE sql SECURITY DEFINER SET search_path = pg_catalog AS $f$
 SELECT 'decision:doble', convert_from(p_payload, 'UTF8'), convert_from(p_decision, 'UTF8')::jsonb->>'contexto_recurso_huella_sha256',
        repeat('0', 64), 'auditoria:doble', clock_timestamp(), true
$f$;

CREATE FUNCTION pg_temp.cambiar(p_bolsa text, p_participacion text, p_situacion text, p_desde timestamptz, p_clave text)
RETURNS text LANGUAGE plpgsql AS $f$
DECLARE v_decision bytea; v_situacion text;
BEGIN
 v_decision := convert_to(jsonb_build_object('principal_id', 'persona:rrhh-prueba', 'accion', 'bolsa.situacion_participacion.cambiar',
   'modulo_id', 'bolsa', 'tipo_recurso', 'participacion_bolsa', 'finalidad', 'gestion_situacion_participacion',
   'recurso_ref', p_participacion, 'campos_permitidos', '[]'::jsonb, 'obligaciones', '[]'::jsonb,
   'contexto_recurso_huella_sha256', repeat('a', 64))::text, 'UTF8');
 SET LOCAL ROLE vec_bolsa_llamamientos_ejecutor;
 SELECT r.situacion INTO STRICT v_situacion FROM vec_bolsa_llamamientos.registrar_situacion_participacion_v1(
   p_bolsa, p_participacion, p_situacion, p_desde, NULL, 'Prueba 000032', 'persona:rrhh-prueba', p_clave, 'recibo:' || p_clave, p_desde,
   '\x00'::bytea, v_decision, '\x00'::bytea, '\x00'::bytea, 1, 1, convert_to(p_participacion, 'UTF8'), '\x00'::bytea, '\x00'::bytea, '\x00'::bytea) r;
 RESET ROLE;
 RETURN v_situacion;
EXCEPTION WHEN OTHERS THEN
 RESET ROLE;
 RETURN 'error:' || SQLSTATE;
END $f$;

CREATE FUNCTION pg_temp.publicar(p_ref text, p_transiciones text[])
RETURNS text LANGUAGE plpgsql AS $f$
DECLARE v record;
BEGIN
 SET LOCAL ROLE vec_bolsa_llamamientos_ejecutor;
 SELECT * INTO STRICT v FROM vec_bolsa_llamamientos.publicar_politica_transiciones_situacion_v1(p_ref, repeat('b', 64), p_transiciones);
 RESET ROLE;
 RETURN v.version || ':' || v.reutilizada;
EXCEPTION WHEN OTHERS THEN
 RESET ROLE;
 RETURN 'error:' || SQLSTATE;
END $f$;

DO $prueba$
DECLARE
 b text; p text; base timestamptz; v1 text[]; v2 text[]; r text; version_fila bigint; historia_antes text;
BEGIN
 SELECT c.bolsa_ref, e.participacion_ref INTO STRICT b, p
   FROM vec_bolsa_llamamientos.constitucion c
   JOIN vec_bolsa_llamamientos.constitucion_entrada e USING (instantanea_ref, version_instantanea)
   JOIN LATERAL (SELECT s.situacion FROM vec_bolsa_llamamientos.situacion_participacion s WHERE s.participacion_ref = e.participacion_ref ORDER BY s.desde DESC LIMIT 1) actual ON true
  WHERE actual.situacion = 'disponible' LIMIT 1;
 SELECT greatest(clock_timestamp(), max(desde)) + interval '1 hour' INTO base FROM vec_bolsa_llamamientos.situacion_participacion;
 SELECT md5(string_agg(s::text, '|' ORDER BY s.participacion_ref, s.desde)) INTO historia_antes FROM vec_bolsa_llamamientos.situacion_participacion s;

 -- Versión 1: el literal de 000012 más disponible>disponible_desde.
 SELECT transiciones INTO STRICT v1 FROM vec_bolsa_llamamientos.politica_transiciones_situacion WHERE version = 1;
 IF cardinality(v1) <> 18 OR NOT ('renuncia>disponible' = ANY (v1)) OR NOT ('disponible>disponible_desde' = ANY (v1)) OR 'renuncia>no_disponible' = ANY (v1) THEN
  RAISE EXCEPTION '000032: la versión 1 no es la esperada: %', v1;
 END IF;

 -- Con la versión 1: la renuncia no pasa a no disponible.
 IF pg_temp.cambiar(b, p, 'renuncia', base, 'p32:renuncia') <> 'renuncia' THEN RAISE EXCEPTION '000032: disponible→renuncia rechazada'; END IF;
 SELECT politica_transiciones_version INTO STRICT version_fila FROM vec_bolsa_llamamientos.situacion_participacion WHERE participacion_ref = p AND recibo_ref = 'recibo:p32:renuncia';
 IF version_fila IS DISTINCT FROM 1 THEN RAISE EXCEPTION '000032: la fila no registra la versión 1'; END IF;
 r := pg_temp.cambiar(b, p, 'no_disponible', base + interval '1 second', 'p32:v1-no-disponible');
 IF r <> 'error:22023' THEN RAISE EXCEPTION '000032: v1 admitió renuncia→no_disponible: %', r; END IF;

 -- Publicación de la política del catálogo del Reglamento (arts. 10 y 11).
 v2 := array_remove(v1, 'renuncia>disponible') || 'renuncia>no_disponible'::text;
 r := pg_temp.publicar('catalogo:prueba:1', v2);
 IF r <> '2:false' THEN RAISE EXCEPTION '000032: publicación v2: %', r; END IF;
 IF pg_temp.publicar('catalogo:prueba:1', v2) <> '2:true' THEN RAISE EXCEPTION '000032: la republicación idéntica creó versión'; END IF;
 IF (SELECT transiciones FROM vec_bolsa_llamamientos.politica_transiciones_situacion WHERE version = 2)
    IS DISTINCT FROM vec_bolsa_llamamientos.transiciones_situacion_canonicas(v2) THEN
  RAISE EXCEPTION '000032: v2 no quedó canónica';
 END IF;

 -- Con la versión 2: renuncia→disponible cerrada; renuncia→no_disponible abierta.
 r := pg_temp.cambiar(b, p, 'disponible', base + interval '2 second', 'p32:v2-disponible');
 IF r <> 'error:22023' THEN RAISE EXCEPTION '000032: v2 admitió renuncia→disponible: %', r; END IF;
 IF pg_temp.cambiar(b, p, 'no_disponible', base + interval '3 second', 'p32:v2-no-disponible') <> 'no_disponible' THEN
  RAISE EXCEPTION '000032: v2 rechazó renuncia→no_disponible';
 END IF;
 SELECT politica_transiciones_version INTO STRICT version_fila FROM vec_bolsa_llamamientos.situacion_participacion WHERE participacion_ref = p AND recibo_ref = 'recibo:p32:v2-no-disponible';
 IF version_fila IS DISTINCT FROM 2 THEN RAISE EXCEPTION '000032: la fila no registra la versión 2'; END IF;
 -- La repetición idempotente devuelve lo registrado sin volver a evaluar.
 IF pg_temp.cambiar(b, p, 'no_disponible', base + interval '3 second', 'p32:v2-no-disponible') <> 'no_disponible' THEN
  RAISE EXCEPTION '000032: la repetición idempotente falló';
 END IF;
 -- B8 «pausar» pasa por la misma función: renuncia justificada (art. 10).
 IF pg_temp.cambiar(b, p, 'disponible', base + interval '4 second', 'p32:vuelve') <> 'disponible'
    OR pg_temp.cambiar(b, p, 'renuncia', base + interval '5 second', 'p32:renuncia-2') <> 'renuncia' THEN
  RAISE EXCEPTION '000032: no se pudo preparar la renuncia B8';
 END IF;
 SET LOCAL ROLE vec_bolsa_llamamientos_ejecutor;
 PERFORM * FROM vec_bolsa_llamamientos.registrar_operacion_situacion_participacion_v1(
   b, p, 'pausar', base + interval '6 second', NULL, 'Renuncia justificada, art. 10', 'persona:rrhh-prueba', 'p32:b8-pausa', 'recibo:p32:b8-pausa',
   base + interval '6 second', 'informe_medico', 'justificante:p32', repeat('d', 64), 'persona:rrhh-prueba', base + interval '6 second',
   '\x00'::bytea, convert_to(jsonb_build_object('principal_id', 'persona:rrhh-prueba', 'accion', 'bolsa.situacion_participacion.cambiar',
     'modulo_id', 'bolsa', 'tipo_recurso', 'participacion_bolsa', 'finalidad', 'gestion_situacion_participacion', 'recurso_ref', p,
     'campos_permitidos', '[]'::jsonb, 'obligaciones', '[]'::jsonb, 'contexto_recurso_huella_sha256', repeat('a', 64))::text, 'UTF8'),
   '\x00'::bytea, '\x00'::bytea, 1, 1, convert_to(p, 'UTF8'), '\x00'::bytea, '\x00'::bytea, '\x00'::bytea);
 RESET ROLE;
 IF (SELECT situacion FROM vec_bolsa_llamamientos.situacion_participacion WHERE participacion_ref = p ORDER BY desde DESC LIMIT 1) <> 'no_disponible'
    OR NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.operacion_situacion_participacion WHERE participacion_ref = p AND desde = base + interval '6 second') THEN
  RAISE EXCEPTION '000032: B8 no pausó la renuncia';
 END IF;
 -- Las invariantes fijas siguen: nunca se sale de excluido.
 IF pg_temp.cambiar(b, p, 'excluido', base + interval '7 second', 'p32:excluye') <> 'excluido' THEN RAISE EXCEPTION '000032: baja rechazada'; END IF;
 IF pg_temp.cambiar(b, p, 'disponible', base + interval '8 second', 'p32:readmite') <> 'error:22023' THEN RAISE EXCEPTION '000032: readmisión admitida'; END IF;

 -- Políticas inválidas: todas rechazadas con 22023 y sin versión nueva.
 FOREACH r IN ARRAY ARRAY[
   pg_temp.publicar('catalogo:prueba:2', v2 || 'excluido>disponible'::text),
   pg_temp.publicar('catalogo:prueba:2', v2 || 'renuncia>renuncia'::text),
   pg_temp.publicar('catalogo:prueba:2', array_remove(v2, 'renuncia>excluido')),
   pg_temp.publicar('catalogo:prueba:2', v2 || 'renuncia>readmitido'::text),
   pg_temp.publicar('catalogo:prueba:2', v2 || 'renuncia>no_disponible'::text),
   pg_temp.publicar('catalogo:prueba:2', v2 || NULL::text),
   pg_temp.publicar('catalogo:prueba:2', v2 || 'a>b>c'::text),
   pg_temp.publicar('catalogo:prueba:2', ARRAY[v2]),
   pg_temp.publicar('catalogo:prueba:2', ARRAY[]::text[]),
   pg_temp.publicar('catalogo:prueba:2', NULL),
   pg_temp.publicar(' catalogo:prueba:2', v2),
   pg_temp.publicar('', v2)] LOOP
  IF r <> 'error:22023' THEN RAISE EXCEPTION '000032: política inválida aceptada: %', r; END IF;
 END LOOP;
 IF (SELECT max(version) FROM vec_bolsa_llamamientos.politica_transiciones_situacion) <> 2 THEN
  RAISE EXCEPTION '000032: una publicación inválida dejó versión';
 END IF;

 -- Solo adición: ni el propietario modifica ni borra versiones; el CHECK
 -- rechaza una inserción directa no canónica.
 BEGIN
  UPDATE vec_bolsa_llamamientos.politica_transiciones_situacion SET catalogo_ref = 'otro' WHERE version = 1;
  RAISE EXCEPTION '000032: versión modificada';
 EXCEPTION WHEN SQLSTATE '55000' OR SQLSTATE 'P0001' THEN IF SQLERRM LIKE '000032:%' THEN RAISE; END IF; END;
 BEGIN
  DELETE FROM vec_bolsa_llamamientos.politica_transiciones_situacion WHERE version = 2;
  RAISE EXCEPTION '000032: versión borrada';
 EXCEPTION WHEN SQLSTATE '55000' OR SQLSTATE 'P0001' THEN IF SQLERRM LIKE '000032:%' THEN RAISE; END IF; END;
 BEGIN
  INSERT INTO vec_bolsa_llamamientos.politica_transiciones_situacion VALUES (3, 'directa', repeat('c', 64), ARRAY['renuncia>excluido', 'disponible>excluido'], clock_timestamp());
  RAISE EXCEPTION '000032: inserción no canónica aceptada';
 EXCEPTION WHEN check_violation THEN NULL; END;

 -- La historia anterior a la prueba no cambia (tampoco las renuncias vigentes).
 IF (SELECT md5(string_agg(s::text, '|' ORDER BY s.participacion_ref, s.desde)) FROM vec_bolsa_llamamientos.situacion_participacion s
      WHERE s.clave_idempotencia NOT LIKE 'p32:%') IS DISTINCT FROM historia_antes THEN
  RAISE EXCEPTION '000032: la historia previa cambió';
 END IF;
END $prueba$;

-- ACL: el ejecutor no lee ni escribe la tabla; solo usa las funciones.
DO $acl$
BEGIN
 SET LOCAL ROLE vec_bolsa_llamamientos_ejecutor;
 BEGIN
  PERFORM 1 FROM vec_bolsa_llamamientos.politica_transiciones_situacion;
  RAISE EXCEPTION '000032: el ejecutor lee la tabla';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 BEGIN
  PERFORM vec_bolsa_llamamientos.transiciones_situacion_canonicas(ARRAY['renuncia>excluido']);
  RAISE EXCEPTION '000032: el ejecutor invoca la función interna';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 IF (SELECT version FROM vec_bolsa_llamamientos.consultar_politica_transiciones_situacion_v1()) <> 2 THEN
  RAISE EXCEPTION '000032: la consulta no devuelve la vigente';
 END IF;
 RESET ROLE;
END $acl$;

CREATE ROLE vec_prueba_000032_ajeno NOLOGIN;
DO $ajeno$
BEGIN
 SET LOCAL ROLE vec_prueba_000032_ajeno;
 BEGIN
  PERFORM * FROM vec_bolsa_llamamientos.publicar_politica_transiciones_situacion_v1('x', repeat('b', 64), ARRAY['renuncia>excluido']);
  RAISE EXCEPTION '000032: un rol ajeno publica';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 BEGIN
  PERFORM * FROM vec_bolsa_llamamientos.consultar_politica_transiciones_situacion_v1();
  RAISE EXCEPTION '000032: un rol ajeno consulta';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 RESET ROLE;
END $ajeno$;

SELECT 'b2_politica_transiciones: correcto' AS resultado;
-- Constancia de quién publicó: cada versión guarda la cuenta de conexión.
DO $publicador$ BEGIN
 IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.politica_transiciones_situacion WHERE publicada_por IS DISTINCT FROM session_user) THEN
  RAISE EXCEPTION 'la política no deja constancia de quién la publicó';
 END IF;
END $publicador$;
ROLLBACK;
