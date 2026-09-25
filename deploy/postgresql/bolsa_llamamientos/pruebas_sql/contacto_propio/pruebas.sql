-- AD3-86 y Bolsa 000040: la fachada admite solo la confirmación de contacto
-- sobre el recurso propio y Bolsa confirma la versión vista, una vez, con
-- idempotencia y cotejo del candidato. Deja historia confirmada.
\set ON_ERROR_STOP on
SET timezone = 'UTC';
-- 1. Fachada AD3-86 como su único llamante (propietario de Bolsa): los
--    cruces se rechazan antes del núcleo; la forma correcta llega a él.
SET ROLE vec_bolsa_llamamientos_propietario;
DO $f$
DECLARE caso record; x record; cand text := 'mi-bolsa:can_contacto_propio_candidato_A1';
BEGIN
 FOR caso IN SELECT * FROM (VALUES
   ('confirmar_contacto','confirmar_contacto','participaciones_candidato',cand,true),
   ('confirmar_contacto','responder_llamamiento','participaciones_candidato',cand,false),
   ('responder_llamamiento','responder_llamamiento','participaciones_candidato',cand,false),
   ('confirmar_contacto','confirmar_contacto','oferta_bolsa',cand,false),
   ('confirmar_contacto','confirmar_contacto','participaciones_candidato','mi-bolsa:otro',false),
   ('confirmar_contacto','confirmar_contacto','participaciones_candidato','oferta:'||repeat('a',64),false)
  ) AS t(operacion, audiencia, tipo, recurso, admitido) LOOP
  BEGIN
   SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.registrar_y_consumir_contacto_propio_bolsa_v3_atestada(
    convert_to(jsonb_build_object('operacion','bolsa.participaciones_propias.'||caso.operacion,
      'audiencia_consumo','vec_bolsa_llamamientos.participaciones_propias.'||caso.audiencia||'.v1',
      'efecto_ref',caso.recurso,'huella_efecto_sha256',repeat('0',64),'suite','VEC-AD-3-COSE-EDDSA-1')::text,'UTF8'),
    convert_to(jsonb_build_object('accion','bolsa.participaciones_propias.'||caso.operacion,'modulo_id','bolsa',
      'tipo_recurso',caso.tipo,'finalidad','gestion_participaciones_propias','recurso_ref',caso.recurso,
      'contexto_recurso_huella_sha256',repeat('0',64),'campos_permitidos','[]'::jsonb,'obligaciones','[]'::jsonb)::text,'UTF8'),
    NULL, NULL, 1, 1, NULL, NULL, NULL, NULL);
   IF NOT caso.admitido OR x.efecto_ref <> caso.recurso OR NOT x.consumo_nuevo THEN RAISE EXCEPTION 'AD3-86: caso % admitido', caso; END IF;
  EXCEPTION WHEN insufficient_privilege THEN
   IF caso.admitido THEN RAISE EXCEPTION 'AD3-86: forma correcta rechazada (%)', SQLERRM; END IF;
   IF SQLERRM NOT LIKE 'AD3-86%' THEN RAISE EXCEPTION 'AD3-86: cruce que llegó al núcleo (%)', caso; END IF;
  END;
 END LOOP;
 -- Las acciones de AD3-84 siguen entrando por su fachada, no por esta.
 BEGIN
  PERFORM vec_autorizacion_atestada_v3.registrar_y_consumir_portal_candidato_bolsa_v3_atestada(
   convert_to(jsonb_build_object('operacion','bolsa.participaciones_propias.confirmar_contacto',
     'audiencia_consumo','vec_bolsa_llamamientos.participaciones_propias.confirmar_contacto.v1','efecto_ref',cand,'huella_efecto_sha256',repeat('0',64))::text,'UTF8'),
   convert_to(jsonb_build_object('accion','bolsa.participaciones_propias.confirmar_contacto','modulo_id','bolsa','tipo_recurso','participaciones_candidato',
     'finalidad','gestion_participaciones_propias','recurso_ref',cand,'contexto_recurso_huella_sha256',repeat('0',64),'campos_permitidos','[]'::jsonb,'obligaciones','[]'::jsonb)::text,'UTF8'),
   NULL,NULL,1,1,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'AD3-84 admite la confirmación de contacto';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 RAISE NOTICE 'fachada AD3-86: forma única y cruces rechazados OK';
END $f$;
RESET ROLE;

-- 2. Bolsa 000040 como cuenta de aplicación.
SET ROLE vec_bolsa_llamamientos_ejecutor;
DO $p$
DECLARE a text := 'can_contacto_propio_candidato_A1'; b text := 'can_contacto_propio_candidato_B1'; r record; r2 record; j jsonb;
BEGIN
 PERFORM prueba_cp.espera($$SELECT count(*) FROM vec_bolsa_llamamientos.confirmacion_contacto_participacion$$, '42501');
 PERFORM prueba_cp.espera($$SELECT vec_bolsa_llamamientos.participacion_contacto_candidato_v1('can_contacto_propio_candidato_A1','bolsa:cp:1')$$, '42501');
 -- La lectura exige la consulta propia consumida en la misma transacción.
 PERFORM prueba_cp.espera($$SELECT vec_bolsa_llamamientos.leer_contacto_candidato_v1('can_contacto_propio_candidato_A1', clock_timestamp())$$, '42501');
 PERFORM prueba_cp.espera($$SELECT prueba_cp.consultar('can_contacto_propio_candidato_B1'), vec_bolsa_llamamientos.leer_contacto_candidato_v1('can_contacto_propio_candidato_A1', clock_timestamp())$$, '42501');
 -- Estado antes de confirmar: origen CONVOCA sin confirmar.
 j := prueba_cp.contacto(a);
 IF jsonb_array_length(j) <> 1 OR j->0->>'version' <> '1' OR j->0->'origen'->>'origen' <> 'convoca' OR j->0->'confirmada_en' <> 'null'::jsonb
    OR j::text LIKE '%part:cp%' OR j::text LIKE '%kms%' THEN RAISE EXCEPTION 'lectura previa: %', j; END IF;
 IF jsonb_array_length(prueba_cp.contacto(b)) <> 0 THEN RAISE EXCEPTION 'B sin contacto aparece'; END IF;
 -- Alta, replay y segunda clave para la misma versión.
 SELECT * INTO STRICT r FROM prueba_cp.confirmar(a, 1, 'clave-contacto-1');
 IF r.reutilizada OR r.version <> 1 OR r.recibo_ref !~ '^recibo:confirmacion-contacto:' THEN RAISE EXCEPTION 'alta: %', r; END IF;
 SELECT * INTO STRICT r2 FROM prueba_cp.confirmar(a, 1, 'clave-contacto-1');
 IF NOT r2.reutilizada OR r2.recibo_ref <> r.recibo_ref OR r2.confirmada_en <> r.confirmada_en THEN RAISE EXCEPTION 'replay: %', r2; END IF;
 -- Un reintento sin decisión viva nueva no devuelve el recibo.
 PERFORM prueba_cp.espera($$SELECT * FROM prueba_cp.confirmar('can_contacto_propio_candidato_A1', 1, 'clave-contacto-1', p_repetida=>true)$$, '42501');
 PERFORM prueba_cp.espera($$SELECT * FROM prueba_cp.confirmar('can_contacto_propio_candidato_A1', 1, 'clave-contacto-2')$$, 'VBC04');
 PERFORM prueba_cp.espera($$SELECT * FROM prueba_cp.confirmar('can_contacto_propio_candidato_A1', 2, 'clave-contacto-1')$$, 'VBC01');
 IF vec_bolsa_llamamientos.leer_confirmacion_contacto_participacion_v1('part:cp:1', 1) IS NULL
    OR prueba_cp.contacto(a)->0->>'confirmada_en' IS NULL THEN RAISE EXCEPTION 'confirmación no visible'; END IF;
 -- Sin contacto, versión que no es la vigente, candidato ajeno y cruces.
 PERFORM prueba_cp.espera($$SELECT * FROM prueba_cp.confirmar('can_contacto_propio_candidato_B1', 1, 'clave-contacto-b1')$$, 'VBC03');
 PERFORM prueba_cp.espera($$SELECT * FROM prueba_cp.confirmar('can_contacto_propio_candidato_A1', 7, 'clave-contacto-7')$$, 'VBC02');
 PERFORM prueba_cp.espera($$SELECT * FROM prueba_cp.confirmar('can_contacto_propio_ajeno_zzzzzz', 1, 'clave-contacto-z')$$, '42501');
 PERFORM prueba_cp.espera($$SELECT * FROM prueba_cp.confirmar('can_contacto_propio_candidato_A1', 1, 'clave-contacto-x1', p_bolsa=>'bolsa:otra')$$, '42501');
 PERFORM prueba_cp.espera($$SELECT * FROM prueba_cp.confirmar('can_contacto_propio_candidato_A1', 1, 'clave-contacto-x2', p_vinculos=>'[{"tipo":"candidato","estado":"activo","referencia":"can_contacto_propio_candidato_B1"}]')$$, '42501');
 PERFORM prueba_cp.espera($$SELECT * FROM prueba_cp.confirmar('can_contacto_propio_candidato_A1', 1, 'clave-contacto-x3', p_efecto=>'mi-bolsa:can_contacto_propio_candidato_B1')$$, '42501');
 PERFORM prueba_cp.espera($$SELECT * FROM prueba_cp.confirmar('can_contacto_propio_candidato_A1', 1, 'clave-contacto-x4', p_accion=>'bolsa.participaciones_propias.responder_llamamiento')$$, '42501');
 PERFORM prueba_cp.espera($$SELECT * FROM prueba_cp.confirmar('can_contacto_propio_candidato_A1', 1, 'corta')$$, '22023');
 RAISE NOTICE 'confirmación de contacto: alta, replay, versión vista, cotejo y negativos OK';
END $p$;
RESET ROLE;

-- 3. RRHH registra una versión nueva: la anterior confirmada no la cubre y
--    la persona confirma la nueva; una concesión repetida no registra nada.
BEGIN;
SET LOCAL session_replication_role = replica;
INSERT INTO vec_bolsa_llamamientos.datos_contacto_participacion VALUES
 ('part:cp:1',2,'kms:prueba',decode(repeat('03',12),'hex'),decode(repeat('04',32),'hex'),'Corregido por RRHH','per_actoractoractoractoractor',now()-interval '1 minute','rrhh-2','recibo:contacto:cp2');
COMMIT;
SET ROLE vec_bolsa_llamamientos_ejecutor;
DO $v$
DECLARE a text := 'can_contacto_propio_candidato_A1'; r record; j jsonb;
BEGIN
 j := prueba_cp.contacto(a);
 IF j->0->>'version' <> '2' OR j->0->'origen' <> 'null'::jsonb OR j->0->'confirmada_en' <> 'null'::jsonb THEN RAISE EXCEPTION 'versión nueva: %', j; END IF;
 PERFORM prueba_cp.espera($$SELECT * FROM prueba_cp.confirmar('can_contacto_propio_candidato_A1', 1, 'clave-contacto-3')$$, 'VBC02');
 PERFORM prueba_cp.espera($$SELECT * FROM prueba_cp.confirmar('can_contacto_propio_candidato_A1', 2, 'clave-contacto-4', p_repetida=>true)$$, '42501');
 SELECT * INTO STRICT r FROM prueba_cp.confirmar(a, 2, 'clave-contacto-4');
 IF r.reutilizada OR r.version <> 2 THEN RAISE EXCEPTION 'versión 2: %', r; END IF;
 RAISE NOTICE 'versión nueva de RRHH: confirmación de la versión vista OK';
END $v$;
RESET ROLE;
SET ROLE vec_bolsa_llamamientos_propietario;
DO $i$ BEGIN
 BEGIN UPDATE vec_bolsa_llamamientos.confirmacion_contacto_participacion SET decision_ref = 'x'; RAISE EXCEPTION 'debe ser inmutable'; EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL; END;
 BEGIN DELETE FROM vec_bolsa_llamamientos.confirmacion_contacto_participacion; RAISE EXCEPTION 'debe ser inmutable'; EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL; END;
 IF (SELECT count(*) FROM vec_bolsa_llamamientos.confirmacion_contacto_participacion) <> 2 THEN RAISE EXCEPTION 'recuento'; END IF;
 RAISE NOTICE 'historia inmutable OK';
END $i$;
RESET ROLE;
