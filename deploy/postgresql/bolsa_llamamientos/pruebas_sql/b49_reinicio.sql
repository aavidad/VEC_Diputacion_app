\set ON_ERROR_STOP on
-- Nueva conexión después del COMMIT de b49_publicacion_cese.sql.
SET SESSION AUTHORIZATION vec_b49_publicador_test;
DO $reinicio$
DECLARE v record; a boolean;
BEGIN
 SELECT * INTO STRICT v FROM vec_bolsa_llamamientos.siguiente_publicacion_cese_b10_v1();
 IF v.origen_posicion<>0 OR v.origen_ref<>'origen:cese:b49:tardio' OR v.fase<>'cese' THEN
  RAISE EXCEPTION 'B49: cese tardío ocultado por cursor máximo';
 END IF;
 a:=vec_bolsa_llamamientos.confirmar_publicacion_cese_b10_v1(0,'origen:cese:b49:tardio','cese',repeat('c',64));
 IF a IS DISTINCT FROM false THEN RAISE EXCEPTION 'B49: alta tardía incorrecta'; END IF;
 -- Aunque hoy ya venza, el ACK de cese no acredita un snapshot posterior
 -- a medianoche. La fase nueva conserva su propio ancla y ACK.
 SELECT * INTO STRICT v FROM vec_bolsa_llamamientos.siguiente_publicacion_cese_b10_v1();
 IF v.origen_posicion<>0 OR v.fase<>'vencimiento' THEN
  RAISE EXCEPTION 'B49: vencimiento prematuramente confirmado';
 END IF;
 a:=vec_bolsa_llamamientos.confirmar_publicacion_cese_b10_v1(0,'origen:cese:b49:tardio','vencimiento',repeat('9',64));
 IF a IS DISTINCT FROM false THEN RAISE EXCEPTION 'B49: alta de vencimiento incorrecta'; END IF;
 SELECT * INTO STRICT v FROM vec_bolsa_llamamientos.siguiente_publicacion_cese_b10_v1();
 IF v.origen_posicion<>2 OR v.fase<>'cese' THEN RAISE EXCEPTION 'B49: segundo cese omitido'; END IF;
 a:=vec_bolsa_llamamientos.confirmar_publicacion_cese_b10_v1(2,'origen:cese:b45:2','cese',repeat('d',64));
 IF a IS DISTINCT FROM false THEN RAISE EXCEPTION 'B49: segunda alta incorrecta'; END IF;
 BEGIN
  PERFORM vec_bolsa_llamamientos.confirmar_publicacion_cese_b10_v1(2,'origen:cese:b45:2','vencimiento',repeat('d',64));
  RAISE EXCEPTION 'B49: vencimiento futuro aceptado';
 EXCEPTION WHEN SQLSTATE '55000' THEN NULL; END;
 SELECT * INTO STRICT v FROM vec_bolsa_llamamientos.siguiente_publicacion_cese_b10_v1();
 IF v.origen_posicion<>3 OR v.fase<>'cese' THEN RAISE EXCEPTION 'B49: tercer cese omitido'; END IF;
 a:=vec_bolsa_llamamientos.confirmar_publicacion_cese_b10_v1(3,'origen:cese:b45:3','cese',repeat('e',64));
 IF a IS DISTINCT FROM false OR EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.siguiente_publicacion_cese_b10_v1()) THEN
  RAISE EXCEPTION 'B49: feed no agotado';
 END IF;
 a:=vec_bolsa_llamamientos.confirmar_publicacion_cese_b10_v1(1,'origen:cese:b45:1','cese',repeat('a',64));
 IF a IS DISTINCT FROM true THEN RAISE EXCEPTION 'B49: replay perdido tras COMMIT'; END IF;
END $reinicio$;
RESET SESSION AUTHORIZATION;
DO $historia$
BEGIN
 IF (SELECT count(*) FROM vec_bolsa_llamamientos.publicacion_cese_b10)<>5
    OR (SELECT count(*) FROM vec_bolsa_llamamientos.restriccion_cese_bolsa)<>4 THEN
  RAISE EXCEPTION 'B49: confirmaciones o ceses duplicados';
 END IF;
END $historia$;
