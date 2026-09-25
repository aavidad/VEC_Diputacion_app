-- Bolsa 000029: disposición a ofertas desde «Mi bolsa». Se ejecuta como la
-- cuenta de aplicación (ejecutor) y deja historia confirmada: ofertas y
-- disposiciones que después impiden deshacer la migración.
\set ON_ERROR_STOP on
SET timezone = 'UTC';
SET ROLE vec_bolsa_llamamientos_ejecutor;
DO $p$
DECLARE
 a text := 'can_disposicion_candidato_A01'; b text := 'can_disposicion_candidato_B01'; c text := 'can_disposicion_candidato_C01'; dd text := 'can_disposicion_candidato_D01';
 o1 text := 'oferta:'||repeat('1',64); o2 text := 'oferta:'||repeat('2',64); o3 text := 'oferta:'||repeat('3',64);
 r record; r2 record; j jsonb; propuesta jsonb;
BEGIN
 -- La cuenta de aplicación no lee ni escribe las tablas ni el núcleo interno.
 PERFORM prueba_disp.espera($$SELECT count(*) FROM vec_bolsa_llamamientos.disposicion_oferta_candidato$$, '42501');
 PERFORM prueba_disp.espera($$SELECT vec_bolsa_llamamientos.participacion_oferta_candidato_v1('can_disposicion_candidato_A01','bolsa:disp:1')$$, '42501');
 PERFORM prueba_disp.espera($$SELECT * FROM vec_bolsa_llamamientos.registrar_disposicion_oferta_interna_v1('oferta:'||repeat('1',64),'recibo:disposicion:'||repeat('e',64),'can_disposicion_candidato_A01','part:disp:3','clave-interna','2026-01-01','decision:x')$$, '42501');

 PERFORM prueba_disp.publicar(o1, interval '5 seconds');
 PERFORM prueba_disp.publicar(o2, interval '2 seconds');
 PERFORM prueba_disp.publicar(o3, interval '1 day');

 -- Alta, replay idempotente y segunda clave rechazada.
 SELECT * INTO STRICT r FROM prueba_disp.manifestar(o1, a, 'clave-disp-a1');
 IF r.reutilizada OR r.oferta_ref <> o1 OR r.recibo_ref !~ '^recibo:disposicion:' THEN RAISE EXCEPTION 'alta: %', r; END IF;
 SELECT * INTO STRICT r2 FROM prueba_disp.manifestar(o1, a, 'clave-disp-a1');
 IF NOT r2.reutilizada OR r2.recibo_ref <> r.recibo_ref OR r2.manifestada_en <> r.manifestada_en THEN RAISE EXCEPTION 'replay: %', r2; END IF;
 PERFORM prueba_disp.espera(format($$SELECT * FROM prueba_disp.manifestar(%L,%L,'clave-disp-a2')$$, o1, a), 'VBO05');
 SELECT * INTO STRICT r FROM prueba_disp.manifestar(o1, b, 'clave-disp-b1');
 IF r.reutilizada THEN RAISE EXCEPTION 'alta B: %', r; END IF;

 -- Cotejo del candidato con el contexto y del material con la oferta.
 PERFORM prueba_disp.espera(format($$SELECT * FROM prueba_disp.manifestar(%L,%L,'clave-disp-c1')$$, o1, c), '42501');
 PERFORM prueba_disp.espera(format($$SELECT * FROM prueba_disp.manifestar(%L,%L,'clave-disp-x1', p_vinculos=>'[{"tipo":"candidato","estado":"activo","referencia":"can_disposicion_candidato_B01"}]')$$, o3, a), '42501');
 PERFORM prueba_disp.espera(format($$SELECT * FROM prueba_disp.manifestar(%L,%L,'clave-disp-x2', p_vinculos=>'[{"tipo":"candidato","estado":"activo","referencia":"can_disposicion_candidato_A01"},{"tipo":"candidato","estado":"activo","referencia":"can_disposicion_candidato_B01"}]')$$, o3, a), '42501');
 PERFORM prueba_disp.espera(format($$SELECT * FROM prueba_disp.manifestar(%L,%L,'clave-disp-x3', p_vinculos=>'[{"tipo":"candidato","estado":"revocado","referencia":"can_disposicion_candidato_A01"}]')$$, o3, a), '42501');
 PERFORM prueba_disp.espera(format($$SELECT * FROM prueba_disp.manifestar(%L,%L,'clave-disp-x4', p_efecto=>%L)$$, o3, a, o1), '42501');
 PERFORM prueba_disp.espera(format($$SELECT * FROM prueba_disp.manifestar(%L,%L,'clave-disp-x5', p_accion=>'bolsa.participaciones_propias.responder_llamamiento')$$, o3, a), '42501');
 PERFORM prueba_disp.espera(format($$SELECT * FROM prueba_disp.manifestar(%L,%L,'clave-disp-x6', p_tipo=>'participaciones_candidato')$$, o3, a), '42501');
 PERFORM prueba_disp.espera(format($$SELECT * FROM prueba_disp.manifestar(%L,%L,'clave-disp-x7', p_repetida=>true)$$, o3, a), '42501');
 -- Entradas inválidas y oferta inexistente.
 PERFORM prueba_disp.espera(format($$SELECT * FROM prueba_disp.manifestar(%L,%L,'corta')$$, o3, a), '22023');
 PERFORM prueba_disp.espera(format($$SELECT * FROM prueba_disp.manifestar('oferta:x',%L,'clave-disp-x8')$$, a), '22023');
 PERFORM prueba_disp.espera(format($$SELECT * FROM prueba_disp.manifestar(%L,'persona','clave-disp-x9')$$, o3), '22023');
 PERFORM prueba_disp.espera(format($$SELECT * FROM prueba_disp.manifestar(%L,%L,'clave-disp-x10', p_en=>clock_timestamp()-interval '1 hour')$$, o3, a), '22023');
 PERFORM prueba_disp.espera(format($$SELECT * FROM prueba_disp.manifestar(%L,%L,'clave-disp-x11')$$, 'oferta:'||repeat('9',64), a), '23503');
 IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.listar_ofertas_candidato_v1(a, clock_timestamp()) e
             CROSS JOIN LATERAL jsonb_array_elements(e) x WHERE x->>'oferta_ref' = o3 AND x->'disposicion' <> 'null'::jsonb) THEN
  RAISE EXCEPTION 'un negativo dejó disposición';
 END IF;

 -- Lista del candidato: abiertas de su bolsa, sin datos ajenos.
 j := vec_bolsa_llamamientos.listar_ofertas_candidato_v1(a, clock_timestamp());
 IF jsonb_array_length(j) <> 3 OR (SELECT count(*) FROM jsonb_array_elements(j) x WHERE x->>'estado' <> 'abierta') <> 0
    OR (SELECT x->'disposicion'->>'recibo' FROM jsonb_array_elements(j) x WHERE x->>'oferta_ref' = o1) IS NULL
    OR (SELECT x->'disposicion' FROM jsonb_array_elements(j) x WHERE x->>'oferta_ref' = o2) <> 'null'::jsonb
    OR j::text LIKE '%part:disp%' OR j::text LIKE '%can_disposicion_candidato_B01%' THEN
  RAISE EXCEPTION 'lista del candidato A: %', j;
 END IF;
 IF jsonb_array_length(vec_bolsa_llamamientos.listar_ofertas_candidato_v1(c, clock_timestamp())) <> 0 THEN
  RAISE EXCEPTION 'el candidato de otra bolsa ve las ofertas';
 END IF;

 -- Vencida la oferta 2 no se admite; desaparece de la lista sin disposición.
 PERFORM pg_sleep(2.2);
 PERFORM prueba_disp.espera(format($$SELECT * FROM prueba_disp.manifestar(%L,%L,'clave-disp-v1')$$, o2, a), 'VBO06');
 j := vec_bolsa_llamamientos.listar_ofertas_candidato_v1(a, clock_timestamp());
 IF EXISTS (SELECT 1 FROM jsonb_array_elements(j) x WHERE x->>'oferta_ref' = o2) THEN RAISE EXCEPTION 'vencida sin disposición visible: %', j; END IF;

 -- Al vencer la oferta 1, VEC propone al mejor orden vigente entre quienes se
 -- ofrecieron (B, posición 2) y RRHH lo confirma.
 PERFORM pg_sleep(3);
 PERFORM prueba_disp.espera(format($$SELECT * FROM prueba_disp.manifestar(%L,%L,'clave-disp-v2')$$, o1, dd), 'VBO06');
 j := vec_bolsa_llamamientos.listar_ofertas_candidato_v1(a, clock_timestamp());
 IF (SELECT x->>'estado' FROM jsonb_array_elements(j) x WHERE x->>'oferta_ref' = o1) <> 'pendiente_resolucion' THEN RAISE EXCEPTION 'pendiente: %', j; END IF;
 SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
 propuesta := vec_bolsa_llamamientos.proyectar_oferta_v1(o1, clock_timestamp());
 IF (propuesta->>'disposiciones_total')::int <> 2 OR propuesta->'propuesta'->>'participacion_ref' <> 'part:disp:2' THEN RAISE EXCEPTION 'propuesta: %', propuesta; END IF;
 RESET ROLE;
 SET LOCAL ROLE vec_bolsa_llamamientos_ejecutor;
 PERFORM vec_bolsa_llamamientos.resolver_oferta_v1(o1, 'recibo:resolucion-oferta:'||repeat('1',64), 'bolsa:disp:1', 'part:disp:2', 'per_actoractoractoractoractor', 'clave-resolucion-1',
   convert_to('{"efecto_ref":"bolsa:disp:1","nonce":"r1"}','UTF8'),
   convert_to('{"principal_id":"per_actoractoractoractoractor","accion":"llamamiento.emitir.v1","tipo_recurso":"bolsa_constituida"}','UTF8'),
   '\x00'::bytea,'\x00'::bytea,1,1,'\x00'::bytea,'\x00'::bytea,'\x00'::bytea,'\x00'::bytea);
 PERFORM prueba_disp.espera(format($$SELECT * FROM prueba_disp.manifestar(%L,%L,'clave-disp-v3')$$, o1, dd), 'VBO06');
 -- Tras resolver, el replay de la disposición original sigue devolviendo su recibo.
 SELECT * INTO STRICT r2 FROM prueba_disp.manifestar(o1, a, 'clave-disp-a1', p_en=>clock_timestamp());
 IF NOT r2.reutilizada THEN RAISE EXCEPTION 'replay tras resolver: %', r2; END IF;
 IF (SELECT x->>'estado' FROM jsonb_array_elements(vec_bolsa_llamamientos.listar_ofertas_candidato_v1(b, clock_timestamp())) x WHERE x->>'oferta_ref' = o1) <> 'adjudicada_propia'
    OR (SELECT x->>'estado' FROM jsonb_array_elements(vec_bolsa_llamamientos.listar_ofertas_candidato_v1(a, clock_timestamp())) x WHERE x->>'oferta_ref' = o1) <> 'resuelta' THEN
  RAISE EXCEPTION 'estado tras resolver';
 END IF;
 RAISE NOTICE 'disposición a ofertas: alta, replay, cotejo, negativos, vencimiento, resolución y lista OK';
END $p$;
RESET ROLE;
-- Historia de solo adición.
SET ROLE vec_bolsa_llamamientos_propietario;
DO $i$ BEGIN
 BEGIN UPDATE vec_bolsa_llamamientos.disposicion_oferta_candidato SET decision_ref = 'x'; RAISE EXCEPTION 'debe ser inmutable'; EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL; END;
 BEGIN DELETE FROM vec_bolsa_llamamientos.disposicion_oferta_candidato; RAISE EXCEPTION 'debe ser inmutable'; EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL; END;
 IF (SELECT count(*) FROM vec_bolsa_llamamientos.disposicion_oferta_candidato) <> 2
    OR (SELECT count(*) FROM vec_bolsa_llamamientos.disposicion_oferta) <> 2 THEN RAISE EXCEPTION 'recuento de disposiciones'; END IF;
 RAISE NOTICE 'historia inmutable OK';
END $i$;
RESET ROLE;
