\set ON_ERROR_STOP on
-- Dos ceses solapados: el anterior (+9) termina después del último (+4).
-- La ficha muestra fecha de cese del último y disponibilidad máxima.
WITH referencia AS (
 SELECT 'origen:cese:b50:ultimo'::text origen_ref,
  'llamamiento:b45:2'::text llamamiento_ref,
  'evento:ct:contrato-bolsa:'||encode(sha256(convert_to(
    'cese'||chr(31)||'origen:cese:b50:ultimo','UTF8')),'hex') evento_ref
), cuerpo AS (
 SELECT r.*,jsonb_build_object('evento_ref',r.evento_ref,'tipo','cese','origen_ref',r.origen_ref,
   'organizacion_ref','organizacion:desarrollo:dipgra','expediente_ref','expediente:ct:b50',
   'llamamiento_ref',r.llamamiento_ref) evento FROM referencia r
)
INSERT INTO vec_bolsa_llamamientos.contrato_participacion(evento_ref,huella_sha256,evento,origen_ref,origen_creada_en,
 origen_posicion,tipo,organizacion_ref,expediente_ref,llamamiento_ref,participacion_ref,bolsa_ref,ocurrido_en,recibido_en)
SELECT evento_ref,encode(sha256(convert_to(evento::text,'UTF8')),'hex'),evento,origen_ref,now(),6,'cese',
 'organizacion:desarrollo:dipgra','expediente:ct:b50',llamamiento_ref,'participacion:rev:2','bolsa:rev',now(),now()
FROM cuerpo;
INSERT INTO vec_contratacion_temporal.cese_prueba(origen_ref,huella,posicion,modalidad,causa,fecha,llamamiento,relacion)
SELECT c.origen_ref,c.huella_sha256,c.origen_posicion,'interinidad',NULL,current_date,
 c.llamamiento_ref,'relacion:b50:ultima'
FROM vec_bolsa_llamamientos.contrato_participacion c WHERE c.origen_ref='origen:cese:b50:ultimo';

SET SESSION AUTHORIZATION vec_b45_relevo_test;
DO $origen$
DECLARE v record;
BEGIN
 SELECT * INTO STRICT v FROM vec_bolsa_llamamientos.registrar_restriccion_cese_bolsa_v1(
  'origen:cese:b50:ultimo',public.huella_cese_b45('origen:cese:b50:ultimo'),6);
 IF v.reutilizada OR v.politica_version<>3
    OR v.disponible_desde<>(current_date+make_interval(months=>4))::date THEN
  RAISE EXCEPTION 'B50: último cese no registrado con su regla';
 END IF;
END $origen$;
RESET SESSION AUTHORIZATION;

SET SESSION AUTHORIZATION vec_b45_ejecutor_test;
DO $lectura$
DECLARE estado record; restriccion record; esperado date;
BEGIN
 esperado:=(current_date-1+make_interval(months=>9))::date;
 SELECT * INTO STRICT estado FROM vec_bolsa_llamamientos.consultar_estado_cese_bolsa_v1('participacion:rev:2',now());
 SELECT * INTO STRICT restriccion FROM vec_bolsa_llamamientos.consultar_restriccion_cese_bolsa_v1('participacion:rev:2',now());
 IF estado.fecha_efecto<>current_date OR estado.disponible_desde<>esperado
    OR estado.en_restriccion IS NOT TRUE OR estado.trabajo_cesado IS NOT FALSE
    OR restriccion.fecha_efecto<>estado.fecha_efecto OR restriccion.disponible_desde<>estado.disponible_desde THEN
  RAISE EXCEPTION 'B50: último cese y disponibilidad máxima divergentes (%, %, %, %)',
   estado.fecha_efecto,estado.disponible_desde,restriccion.fecha_efecto,restriccion.disponible_desde;
 END IF;
END $lectura$;
RESET SESSION AUTHORIZATION;

CREATE ROLE vec_b50_sin_rol_test LOGIN;
SET SESSION AUTHORIZATION vec_b50_sin_rol_test;
DO $denegacion$
BEGIN
 BEGIN
  PERFORM * FROM vec_bolsa_llamamientos.consultar_estado_cese_bolsa_v1('participacion:rev:2',now());
  RAISE EXCEPTION 'B50: no miembro leyó estado de cese';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $denegacion$;
RESET SESSION AUTHORIZATION;

SET SESSION AUTHORIZATION vec_bolsa_llamamientos_propietario;
DO $propietario$
BEGIN
 BEGIN
  PERFORM * FROM vec_bolsa_llamamientos.consultar_estado_cese_bolsa_v1('participacion:rev:2',now());
  RAISE EXCEPTION 'B50: sesión propietaria leyó fachada operativa';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $propietario$;
RESET SESSION AUTHORIZATION;

DO $acl$
BEGIN
 IF NOT has_function_privilege('vec_bolsa_llamamientos_ejecutor',
    'vec_bolsa_llamamientos.consultar_estado_cese_bolsa_v1(text,timestamptz)','EXECUTE')
    OR has_function_privilege('vec_bolsa_llamamientos_relevo_cese',
    'vec_bolsa_llamamientos.consultar_estado_cese_bolsa_v1(text,timestamptz)','EXECUTE')
    OR has_function_privilege('vec_b50_sin_rol_test',
    'vec_bolsa_llamamientos.consultar_estado_cese_bolsa_v1(text,timestamptz)','EXECUTE')
    OR has_function_privilege('vec_bolsa_llamamientos_ejecutor',
    'vec_bolsa_llamamientos.estado_cese_bolsa_v1(text,timestamptz)','EXECUTE') THEN
  RAISE EXCEPTION 'B50: ACL de lector inválida';
 END IF;
END $acl$;
