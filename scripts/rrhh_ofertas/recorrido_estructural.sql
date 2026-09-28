\set ON_ERROR_STOP on
-- Escenario de estado de B28 con filas sintéticas fechadas en el pasado.
-- La siembra usa replica para evitar fingir una autorización histórica; las
-- resoluciones posteriores usan la función real y el rol ejecutor. B47 se
-- prueba por la publicación V2 en el guion principal.
BEGIN;
SET LOCAL timezone='UTC';
SET LOCAL session_replication_role=replica;
INSERT INTO vec_bolsa_llamamientos.constitucion_entrada VALUES
 ('inst:of:1',1,1,'part:of:1',1),('inst:of:1',1,2,'part:of:2',2),
 ('inst:of:1',1,3,'part:of:3',3),('inst:of:1',1,4,'part:of:4',4);
INSERT INTO vec_bolsa_llamamientos.situacion_participacion
 (participacion_ref,situacion,desde,hasta,fecha_disponible,motivo,actor,registrada_en,clave_idempotencia,recibo_ref)
SELECT p, CASE WHEN p='part:of:1' THEN 'no_disponible' ELSE 'disponible' END,
 clock_timestamp()-interval '9 days',NULL,NULL,'alta','per_actoractoractoractoractor',
 clock_timestamp()-interval '9 days','clave:'||p,'recibo:'||p
FROM unnest(ARRAY['part:of:1','part:of:2','part:of:3','part:of:4']) p;
INSERT INTO vec_bolsa_llamamientos.politica_orden_bolsa
 (politica_ref,bolsa_ref,version,criterio,tipo_lista,reposicion,provisional,rotulo,actor,vigente_desde,vigente_hasta,registrada_en)
VALUES ('politica:of:1','bolsa:of:1',1,'puntuacion_desc_acta','rotatoria','misma_posicion',false,
 'Ejemplo estructural','per_actoractoractoractoractor',clock_timestamp()-interval '10 days',NULL,clock_timestamp()-interval '10 days');
INSERT INTO vec_bolsa_llamamientos.oferta_publicada
 (oferta_ref,recibo_ref,bolsa_ref,actor_ref,clave_idempotencia,datos,plazo,publicada_en,vence_antes_de,huella_comando_sha256,decision_ref)
SELECT 'oferta:'||repeat(n,64),'recibo:oferta:'||repeat(n,64),'bolsa:of:1',
 'per_actoractoractoractoractor','clave-fixture-'||n,
 '{"categoria":"Auxiliar administrativo","centro":"Centro sintético","fecha_inicio":"2026-10-01","descripcion":"Oferta de prueba"}'::jsonb,
 jsonb_build_object('regla_ref','politica-ofertas:bolsa:of:1:1','huella_catalogo',
  (SELECT huella_sha256 FROM vec_bolsa_llamamientos.politica_ofertas_version WHERE bolsa_ref='bolsa:of:1' AND version=1),
  'unidad','dias_habiles','cantidad',2,'computo','administrativo','municipio_sede','18087',
  'ultimo_dia',(current_date-1)::text,'ejemplo',true,'politica_version',1,
  'calendarios',jsonb_build_array('fixture-calendario-estructural')),
 clock_timestamp()-interval '3 days',clock_timestamp()-interval '1 day',repeat(n,64),'decision:fixture:oferta:'||n
FROM unnest(ARRAY['6','7','8']) n;
INSERT INTO vec_bolsa_llamamientos.disposicion_oferta VALUES
 ('oferta:'||repeat('6',64),'part:of:1',clock_timestamp()-interval '2 days','clave-fixture-disp-1','recibo:disposicion:'||repeat('1',64)),
 ('oferta:'||repeat('6',64),'part:of:4',clock_timestamp()-interval '2 days','clave-fixture-disp-4','recibo:disposicion:'||repeat('4',64)),
 ('oferta:'||repeat('6',64),'part:of:3',clock_timestamp()-interval '2 days','clave-fixture-disp-3','recibo:disposicion:'||repeat('3',64)),
 ('oferta:'||repeat('8',64),'part:of:4',clock_timestamp()-interval '2 days','clave-fixture-disp-8','recibo:disposicion:'||repeat('8',64));
SET LOCAL session_replication_role=origin;

DO $prueba$
DECLARE cap bytea:=convert_to('{"efecto_ref":"bolsa:of:1"}','UTF8');
 cap7 bytea:=convert_to('{"efecto_ref":"bolsa:of:1","nonce":"segunda-resolucion"}','UTF8');
 cap_get bytea:=convert_to('{"efecto_ref":"bolsa:of:1","nonce":"lectura-version-2"}','UTF8');
 repetida bytea:=convert_to('{"efecto_ref":"bolsa:of:1","repetida":true}','UTF8');
 dec bytea:=convert_to('{"principal_id":"per_actoractoractoractoractor","accion":"llamamiento.emitir.v1","modulo_id":"bolsa","finalidad":"gestion_llamamientos_bolsa","recurso_ref":"bolsa:of:1","tipo_recurso":"bolsa_constituida"}','UTF8');
 dec_publicar bytea:=convert_to('{"principal_id":"per_actoractoractoractoractor","accion":"bolsa.politica_ofertas.publicar","modulo_id":"bolsa","tipo_recurso":"bolsa_constituida","finalidad":"gobierno_politica_ofertas_bolsa","recurso_ref":"bolsa:of:1"}','UTF8');
 dec_get bytea:=convert_to('{"principal_id":"per_actoractoractoractoractor","accion":"bolsa.politica_ofertas.consultar","modulo_id":"bolsa","tipo_recurso":"bolsa_constituida","finalidad":"consultar_politica_ofertas_bolsa","recurso_ref":"bolsa:of:1","contexto_recurso_huella_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","campos_permitidos":["politica_ofertas"],"obligaciones":[]}','UTF8');
 a text:='oferta:'||repeat('6',64); b text:='oferta:'||repeat('7',64); c text:='oferta:'||repeat('8',64);
 r record; propuesta jsonb; politica2 jsonb; lectura jsonb;
BEGIN
 SET LOCAL ROLE vec_bolsa_llamamientos_ejecutor;
 SELECT x INTO propuesta FROM jsonb_array_elements(vec_bolsa_llamamientos.listar_ofertas_bolsa_v1('bolsa:of:1',clock_timestamp(),20)) x WHERE x->>'oferta_ref'=a;
 IF propuesta->>'estado'<>'pendiente_resolucion' OR propuesta->'propuesta'->>'participacion_ref'<>'part:of:3' THEN
  RAISE EXCEPTION 'orden o elegibilidad incorrectos: %',propuesta;
 END IF;
 BEGIN
  PERFORM vec_bolsa_llamamientos.resolver_oferta_v1(a,'recibo:resolucion-oferta:'||repeat('6',64),'bolsa:of:1','part:of:4',
   'per_actoractoractoractoractor','clave-resolver-fixture-6',cap,dec,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
  RAISE EXCEPTION 'RRHH confirmó persona fuera de orden';
 EXCEPTION WHEN SQLSTATE 'VBO04' THEN NULL; END;
 SELECT * INTO r FROM vec_bolsa_llamamientos.resolver_oferta_v1(a,'recibo:resolucion-oferta:'||repeat('6',64),
  'bolsa:of:1','part:of:3','per_actoractoractoractoractor','clave-resolver-fixture-6',cap,dec,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
 IF r.reutilizada OR r.oferta->>'estado'<>'adjudicada' OR r.oferta->'resolucion'->>'participacion_ref'<>'part:of:3' THEN
  RAISE EXCEPTION 'confirmación RRHH distinta: %',r.oferta;
 END IF;
 SELECT * INTO r FROM vec_bolsa_llamamientos.resolver_oferta_v1(a,'recibo:resolucion-oferta:'||repeat('6',64),
  'bolsa:of:1','part:of:3','per_actoractoractoractoractor','clave-resolver-fixture-6',cap,dec,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
 IF NOT r.reutilizada THEN RAISE EXCEPTION 'replay de confirmación duplicó el efecto'; END IF;
 BEGIN
  PERFORM vec_bolsa_llamamientos.resolver_oferta_v1(a,'recibo:resolucion-oferta:'||repeat('6',64),
   'bolsa:of:1','part:of:3','per_actoractoractoractoractor','clave-resolver-fixture-6',repetida,dec,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
  RAISE EXCEPTION 'replay sin autorización viva aceptado';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 SELECT * INTO r FROM vec_bolsa_llamamientos.resolver_oferta_v1(b,'recibo:resolucion-oferta:'||repeat('7',64),
  'bolsa:of:1',NULL,'per_actoractoractoractoractor','clave-resolver-fixture-7',cap7,dec,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
 IF r.oferta->>'estado'<>'llamamiento_directo' OR r.oferta->'resolucion'->>'participacion_ref' IS NOT NULL THEN
  RAISE EXCEPTION 'no cubierta incorrecta: %',r.oferta;
 END IF;
 SELECT x INTO propuesta FROM jsonb_array_elements(vec_bolsa_llamamientos.listar_ofertas_bolsa_v1('bolsa:of:1',clock_timestamp(),20)) x WHERE x->>'oferta_ref'=c;
 IF propuesta->'propuesta'->>'participacion_ref'<>'part:of:4' THEN RAISE EXCEPTION 'siguiente elegible incorrecta: %',propuesta; END IF;
 RESET ROLE;
 IF (SELECT count(*) FROM vec_bolsa_llamamientos.resolucion_oferta WHERE oferta_ref IN (a,b))<>2 THEN
  RAISE EXCEPTION 'historia de resoluciones incorrecta'; END IF;
 -- Editar publica otra versión, sin alterar la huella ni el plazo de la oferta
 -- creada antes. Después la lectura propia B51 ve la versión nueva.
 politica2:=jsonb_set((SELECT politica FROM vec_bolsa_llamamientos.politica_ofertas_version
                       WHERE bolsa_ref='bolsa:of:1' AND version=1),'{plazo,cantidad}','3'::jsonb);
 SELECT * INTO r FROM vec_bolsa_llamamientos.publicar_politica_ofertas_v1('bolsa:of:1',1,politica2,
  'per_actoractoractoractoractor','clave-politica-version-2','recibo:politica-ofertas:'||repeat('b',64),
  cap,dec_publicar,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
 IF r.reutilizada OR r.politica->>'version'<>'2' OR
    (SELECT plazo->>'politica_version' FROM vec_bolsa_llamamientos.oferta_publicada WHERE oferta_ref=a)<>'1'
 THEN RAISE EXCEPTION 'edición reescribió la oferta anterior'; END IF;
 SET LOCAL ROLE vec_bolsa_llamamientos_ejecutor;
 SELECT vec_bolsa_llamamientos.consultar_politica_ofertas_v2('bolsa:of:1',cap_get,dec_get,
  '\x00','\x00',1,1,'\x00','\x00','\x00','\x00') INTO lectura;
 IF lectura->>'version'<>'2' OR lectura#>>'{politica,plazo,cantidad}'<>'3' THEN
  RAISE EXCEPTION 'GET propio no vio la edición: %',lectura; END IF;
 RESET ROLE;
 IF (SELECT count(*) FROM vec_bolsa_llamamientos.politica_ofertas_lectura_v3)<>3 THEN
  RAISE EXCEPTION 'GET de versión nueva sin traza'; END IF;
END $prueba$;
COMMIT;
SELECT 'RRHH OFERTAS RECORRIDO ESTRUCTURAL OK';
