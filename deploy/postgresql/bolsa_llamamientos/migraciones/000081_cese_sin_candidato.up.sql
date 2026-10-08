\set ON_ERROR_STOP on
-- Bolsa 000081. Un cese CT cuyo llamamiento eligió una participación de una
-- bolsa que nunca se constituyó (fuente sintética del puente CT) no tiene
-- candidato: vinculo_candidato exige constitucion_entrada. Hasta ahora el
-- relevo B45 reintentaba ese cese para siempre y bloqueaba todos los
-- posteriores. Aquí queda registrado y auditado como «sin candidato» y el
-- cursor avanza. La proyección enlaza el asiento común del acto CT115. No
-- aplica restricción, no toca participaciones y no cambia
-- el caso de una bolsa constituida a la que le falta el vínculo: ese sigue
-- reintentándose hasta que se rellenen los vínculos. Requiere Bolsa 000045,
-- CT129 y CT197. No se reaplica.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000081',0));
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;

DO $pre$
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR to_regrole('vec_bolsa_llamamientos_relevo_cese') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.restriccion_cese_bolsa') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.cese_ajeno_bolsa') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.contrato_participacion') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.contrato_participacion_cuarentena') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.llamamiento_integracion_desarrollo') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.integracion_desarrollo') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.constitucion_entrada') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.constitucion') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.bolsa_constituida') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.vinculo_candidato') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.constitucion_rechazar_mutacion()') IS NULL
    OR to_regprocedure('vec_contratacion_temporal.verificar_cese_publicado_bolsa_v1(text,text,bigint)') IS NULL
    OR to_regprocedure('vec_contratacion_temporal.verificar_auditoria_cese_publicado_bolsa_v1(text,text,bigint)') IS NULL
    OR NOT has_function_privilege('vec_bolsa_llamamientos_propietario',
        'vec_contratacion_temporal.verificar_auditoria_cese_publicado_bolsa_v1(text,text,bigint)','EXECUTE')
    OR to_regclass('vec_bolsa_llamamientos.cese_sin_candidato_bolsa') IS NOT NULL
    OR to_regprocedure('vec_bolsa_llamamientos.confirmar_cese_sin_candidato_bolsa_v1(text,text,bigint)') IS NOT NULL
    OR to_regprocedure('vec_bolsa_llamamientos.listar_ceses_sin_candidato_pendientes_v1(integer)') IS NOT NULL THEN
  RAISE EXCEPTION 'Bolsa 000081: preimagen incompatible o ya instalada' USING ERRCODE='55000';
 END IF;
 -- El cursor se sustituye solo si es exactamente el de Bolsa 000045.
 IF NOT EXISTS (SELECT 1 FROM pg_proc p
     WHERE p.oid='vec_bolsa_llamamientos.cursor_restriccion_cese_bolsa_v1()'::regprocedure
       AND md5(p.prosrc)='1654ac60956eaa685f13683466fb51cd'
       AND p.proowner='vec_bolsa_llamamientos_propietario'::regrole AND p.prosecdef
       AND p.proconfig=ARRAY['search_path=pg_catalog']) THEN
  RAISE EXCEPTION 'Bolsa 000081: clave=cursor_cese_preimagen esperado=1654ac60956eaa685f13683466fb51cd actual=%',
   (SELECT md5(prosrc) FROM pg_proc WHERE oid='vec_bolsa_llamamientos.cursor_restriccion_cese_bolsa_v1()'::regprocedure)
   USING ERRCODE='55000';
 END IF;
END $pre$;

CREATE TABLE vec_bolsa_llamamientos.cese_sin_candidato_bolsa (
 origen_ref text PRIMARY KEY CHECK (octet_length(origen_ref) BETWEEN 1 AND 512),
 evento_ref text NOT NULL UNIQUE REFERENCES vec_bolsa_llamamientos.contrato_participacion(evento_ref),
 origen_huella_sha256 text NOT NULL CHECK (origen_huella_sha256 ~ '^[a-f0-9]{64}$'),
 origen_posicion bigint NOT NULL CHECK (origen_posicion>=0),
 llamamiento_ref text NOT NULL CHECK (octet_length(llamamiento_ref) BETWEEN 1 AND 512),
 -- Solo referencias del puente sintético de CT (referenciaPuenteLlamamientoDesarrollo).
 participacion_ref text NOT NULL CHECK (participacion_ref ~ '^participacion-sintetica-[1-9][0-9]{0,2}:[a-z]{16,128}$'),
 bolsa_ref text NOT NULL CHECK (bolsa_ref ~ '^bolsa-sintetica:[a-z]{16,128}$'),
 expediente_ref text NOT NULL CHECK (octet_length(expediente_ref) BETWEEN 1 AND 512),
 organizacion_ref text NOT NULL CHECK (organizacion_ref ~ '^organizacion:desarrollo:'),
 recibo_ct_ref text NOT NULL CHECK (octet_length(recibo_ct_ref) BETWEEN 1 AND 512),
 auditoria_ct_ref text NOT NULL CHECK (auditoria_ct_ref ~ '^aud_v3_[0-9a-f]{32}$'),
 motivo text NOT NULL CHECK (motivo='participacion_no_constituida'),
 registro jsonb NOT NULL CHECK (jsonb_typeof(registro)='object'),
 CHECK (registro->>'origen_evento_ref' IS NOT DISTINCT FROM origen_ref
        AND registro->>'auditoria_ct_ref' IS NOT DISTINCT FROM auditoria_ct_ref),
 registro_sha256 text NOT NULL CHECK (registro_sha256=encode(sha256(convert_to(registro::text,'UTF8')),'hex')),
 recibido_en timestamptz(6) NOT NULL,
 recibido_por text NOT NULL DEFAULT session_user CHECK (octet_length(recibido_por) BETWEEN 1 AND 128),
 CHECK (evento_ref='evento:ct:contrato-bolsa:'||encode(sha256(convert_to('cese'||chr(31)||origen_ref,'UTF8')),'hex'))
);
CREATE INDEX cese_sin_candidato_bolsa_cursor_idx
 ON vec_bolsa_llamamientos.cese_sin_candidato_bolsa(origen_posicion DESC,origen_ref DESC);
CREATE INDEX cese_sin_candidato_bolsa_participacion_idx
 ON vec_bolsa_llamamientos.cese_sin_candidato_bolsa(participacion_ref);
CREATE INDEX constitucion_entrada_participacion_b81_idx
 ON vec_bolsa_llamamientos.constitucion_entrada(participacion_ref);
ALTER TABLE vec_bolsa_llamamientos.cese_sin_candidato_bolsa ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.cese_sin_candidato_bolsa FORCE ROW LEVEL SECURITY;
CREATE POLICY cese_sin_candidato_bolsa_solo_propietario ON vec_bolsa_llamamientos.cese_sin_candidato_bolsa
 TO vec_bolsa_llamamientos_propietario
 USING (current_user='vec_bolsa_llamamientos_propietario')
 WITH CHECK (current_user='vec_bolsa_llamamientos_propietario');
REVOKE ALL ON TABLE vec_bolsa_llamamientos.cese_sin_candidato_bolsa FROM PUBLIC;
CREATE TRIGGER cese_sin_candidato_bolsa_inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.cese_sin_candidato_bolsa
 FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();
COMMENT ON TABLE vec_bolsa_llamamientos.cese_sin_candidato_bolsa IS
 'Proyección técnica de cese CT verificado y ligado a su auditoría común: sin candidato, sin restricción; solo adición.';

-- Mismas guardas de sesión que registrar/confirmar de 000045. Solo admite el
-- cese cuando B13 ya lo tiene, con participación, sin cuarentena, coherente
-- con la propuesta del llamamiento de integración, y cuando ni la
-- participación ni la bolsa aparecen en ninguna constitución. En cualquier
-- otro caso devuelve 23503 y el relevo sigue reintentando, como antes.
CREATE FUNCTION vec_bolsa_llamamientos.confirmar_cese_sin_candidato_bolsa_v1(
 p_origen_ref text,p_huella_sha256 text,p_posicion bigint)
RETURNS boolean
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET timezone='UTC' SET lock_timeout='2s'
 SET statement_timeout='5s' AS $f$
DECLARE v_ct record; v_b13 record; v_llamamiento record; v_procedencia record;
 v_previa vec_bolsa_llamamientos.cese_sin_candidato_bolsa;
 v_evento_ref text; v_registro jsonb; v_ahora timestamptz;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_bolsa_llamamientos_relevo_cese','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_migrador','MEMBER')
    OR p_origen_ref IS NULL OR octet_length(p_origen_ref) NOT BETWEEN 1 AND 512
    OR p_origen_ref !~ '^[A-Za-z0-9][A-Za-z0-9:._/-]*$'
    OR p_huella_sha256 IS NULL OR p_huella_sha256 !~ '^[a-f0-9]{64}$'
    OR p_posicion IS NULL OR p_posicion<0 THEN
  RAISE EXCEPTION 'cese sin candidato no autorizado' USING ERRCODE='42501';
 END IF;
 SELECT * INTO v_ct FROM vec_contratacion_temporal.verificar_cese_publicado_bolsa_v1(p_origen_ref,p_huella_sha256,p_posicion);
 IF NOT FOUND OR v_ct.llamamiento_ref IS NULL OR v_ct.expediente_ref IS NULL
    OR v_ct.organizacion_ref IS NULL OR v_ct.recibo_ref IS NULL THEN
  RAISE EXCEPTION 'cese CT no acreditado' USING ERRCODE='42501';
 END IF;
 SELECT * INTO v_procedencia FROM vec_contratacion_temporal.verificar_auditoria_cese_publicado_bolsa_v1(
  p_origen_ref,p_huella_sha256,p_posicion);
 IF NOT FOUND OR v_procedencia.origen_ref IS DISTINCT FROM p_origen_ref
    OR v_procedencia.auditoria_ref IS NULL
    OR v_procedencia.auditoria_ref !~ '^aud_v3_[0-9a-f]{32}$' THEN
  RAISE EXCEPTION 'auditoría del cese CT no acreditada' USING ERRCODE='42501';
 END IF;
 v_evento_ref:='evento:ct:contrato-bolsa:'||encode(sha256(convert_to('cese'||chr(31)||p_origen_ref,'UTF8')),'hex');
 -- Mismo cerrojo que B13 y B45 para el evento: nada cambia hasta COMMIT.
 PERFORM pg_advisory_xact_lock(hashtextextended('bolsa:contrato-participacion:'||v_evento_ref,0));
 SELECT c.participacion_ref,c.bolsa_ref,c.llamamiento_ref INTO v_b13
 FROM vec_bolsa_llamamientos.contrato_participacion c
 WHERE c.evento_ref=v_evento_ref AND c.origen_ref=p_origen_ref
   AND c.huella_sha256=p_huella_sha256 AND c.origen_posicion=p_posicion AND c.tipo='cese'
   AND c.participacion_ref IS NOT NULL AND c.bolsa_ref IS NOT NULL
   AND NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.contrato_participacion_cuarentena q
                    WHERE q.evento_ref=c.evento_ref);
 IF NOT FOUND OR v_b13.llamamiento_ref IS DISTINCT FROM v_ct.llamamiento_ref THEN
  RAISE EXCEPTION 'cese B13 pendiente o divergente' USING ERRCODE='23503';
 END IF;
 SELECT l.bolsa_ref,convert_from(i.registro_canonico,'UTF8')::jsonb #>> '{propuesta,participacion_seleccionada_ref}' AS participacion
   INTO v_llamamiento
 FROM vec_bolsa_llamamientos.llamamiento_integracion_desarrollo l
 JOIN vec_bolsa_llamamientos.integracion_desarrollo i ON i.operacion_ref=l.operacion_ref
 WHERE l.llamamiento_ref=v_ct.llamamiento_ref;
 IF NOT FOUND OR v_llamamiento.participacion IS DISTINCT FROM v_b13.participacion_ref
    OR v_llamamiento.bolsa_ref IS DISTINCT FROM v_b13.bolsa_ref THEN
  RAISE EXCEPTION 'llamamiento de cese divergente' USING ERRCODE='23503';
 END IF;
 -- Guarda positiva: solo el puente sintético de CT en una organización de
 -- desarrollo, como la política sintética de 000045. Otra fuente de bolsas
 -- nunca cae aquí aunque no pase por constitucion.
 IF v_ct.organizacion_ref !~ '^organizacion:desarrollo:'
    OR v_b13.bolsa_ref !~ '^bolsa-sintetica:[a-z]{16,128}$'
    OR v_b13.participacion_ref !~ '^participacion-sintetica-[1-9][0-9]{0,2}:[a-z]{16,128}$' THEN
  RAISE EXCEPTION 'cese sin candidato fuera del puente sintético' USING ERRCODE='23503';
 END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('bolsa:cese-sin-candidato:'||p_origen_ref,0));
 SELECT * INTO v_previa FROM vec_bolsa_llamamientos.cese_sin_candidato_bolsa s WHERE s.origen_ref=p_origen_ref;
 IF FOUND THEN
  IF v_previa.origen_huella_sha256<>p_huella_sha256 OR v_previa.origen_posicion<>p_posicion
     OR v_previa.llamamiento_ref<>v_ct.llamamiento_ref OR v_previa.participacion_ref<>v_b13.participacion_ref
     OR v_previa.bolsa_ref<>v_b13.bolsa_ref
     OR v_previa.auditoria_ct_ref<>v_procedencia.auditoria_ref THEN
   RAISE EXCEPTION 'cese sin candidato divergente' USING ERRCODE='VBC01';
  END IF;
  RETURN true;
 END IF;
 IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.restriccion_cese_bolsa r WHERE r.evento_ref=v_evento_ref OR r.origen_ref=p_origen_ref)
    OR EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.cese_ajeno_bolsa a WHERE a.origen_ref=p_origen_ref) THEN
  RAISE EXCEPTION 'cese ya resuelto por otra vía' USING ERRCODE='VBC01';
 END IF;
 -- Una repetición conserva la proyección histórica. Para un cese nuevo, una
 -- participación o bolsa ya constituida exige resolver primero su vínculo.
 IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.constitucion_entrada e WHERE e.participacion_ref=v_b13.participacion_ref)
    OR EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.vinculo_candidato vc WHERE vc.participacion_ref=v_b13.participacion_ref)
    OR EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.constitucion c WHERE c.bolsa_ref=v_b13.bolsa_ref)
    OR EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.bolsa_constituida b WHERE b.bolsa_ref=v_b13.bolsa_ref) THEN
  RAISE EXCEPTION 'candidato de cese pendiente en bolsa constituida' USING ERRCODE='23503';
 END IF;
 v_ahora:=date_trunc('microseconds',clock_timestamp());
 v_registro:=jsonb_build_object('esquema','vec.bolsa.cese.sin-candidato.proyeccion.v1','evento_ref',v_evento_ref,
   'accion','cese_sin_candidato_proyectado','resultado','sin_candidato','motivo','participacion_no_constituida',
   'proceso',session_user,'origen_evento_ref',p_origen_ref,'auditoria_ct_ref',v_procedencia.auditoria_ref,
   'correlacion_ref',p_origen_ref,'llamamiento_ref',v_ct.llamamiento_ref,
   'participacion_ref',v_b13.participacion_ref,'bolsa_ref',v_b13.bolsa_ref,'recibo_ct_ref',v_ct.recibo_ref,
   'registrada_en',to_char(v_ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'));
 INSERT INTO vec_bolsa_llamamientos.cese_sin_candidato_bolsa(origen_ref,evento_ref,origen_huella_sha256,origen_posicion,
  llamamiento_ref,participacion_ref,bolsa_ref,expediente_ref,organizacion_ref,recibo_ct_ref,auditoria_ct_ref,
  motivo,registro,registro_sha256,recibido_en)
 VALUES(p_origen_ref,v_evento_ref,p_huella_sha256,p_posicion,v_ct.llamamiento_ref,v_b13.participacion_ref,v_b13.bolsa_ref,
  v_ct.expediente_ref,v_ct.organizacion_ref,v_ct.recibo_ref,v_procedencia.auditoria_ref,
  'participacion_no_constituida',v_registro,
  encode(sha256(convert_to(v_registro::text,'UTF8')),'hex'),v_ahora);
 RETURN false;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.confirmar_cese_sin_candidato_bolsa_v1(text,text,bigint) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.confirmar_cese_sin_candidato_bolsa_v1(text,text,bigint) TO vec_bolsa_llamamientos_relevo_cese;

-- B8 puede vincular después del cese. Esta página técnica conserva la
-- proyección pendiente hasta que el mismo relevo aplica B45; no mueve el
-- cursor de publicaciones CT ni el recibo histórico de B81.
CREATE FUNCTION vec_bolsa_llamamientos.listar_ceses_sin_candidato_pendientes_v1(p_limite integer)
RETURNS jsonb
LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path=pg_catalog,pg_temp
SET statement_timeout='5s' AS $pendientes$
DECLARE v_pagina jsonb;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR session_user=current_user
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_relevo_cese','MEMBER') IS NOT TRUE
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_migrador','MEMBER')
    OR p_limite IS NULL OR p_limite NOT BETWEEN 1 AND 100 THEN
  RAISE EXCEPTION 'B81: consulta técnica pendiente no autorizada' USING ERRCODE='42501';
 END IF;
 SELECT coalesce(jsonb_agg(jsonb_build_object(
   'origen_ref',x.origen_ref,'huella_sha256',x.origen_huella_sha256,
   'origen_posicion',x.origen_posicion) ORDER BY x.origen_posicion,x.origen_ref),'[]'::jsonb)
 INTO v_pagina
 FROM (
  SELECT s.origen_ref,s.origen_huella_sha256,s.origen_posicion
  FROM vec_bolsa_llamamientos.cese_sin_candidato_bolsa s
  JOIN vec_bolsa_llamamientos.vinculo_candidato vc
    ON vc.participacion_ref=s.participacion_ref
  WHERE NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.restriccion_cese_bolsa r
                    WHERE r.origen_ref=s.origen_ref)
  ORDER BY s.origen_posicion,s.origen_ref LIMIT p_limite
 ) x;
 RETURN v_pagina;
END $pendientes$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.listar_ceses_sin_candidato_pendientes_v1(integer) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.listar_ceses_sin_candidato_pendientes_v1(integer)
 TO vec_bolsa_llamamientos_relevo_cese;

-- El cursor cuenta también los ceses sin candidato. Mismas guardas, firma,
-- ACL que la preimagen instalada postHX+HZ+B85+B86+CT193+B87.
-- Literal obtenido con pg_get_functiondef; se añade pg_temp al search_path.
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.cursor_restriccion_cese_bolsa_v1()
 RETURNS TABLE(origen_posicion bigint, origen_ref text)
 LANGUAGE plpgsql
 STABLE SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
AS $function$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_bolsa_llamamientos_relevo_cese','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_migrador','MEMBER') THEN
  RAISE EXCEPTION 'cursor de cese no autorizado' USING ERRCODE='42501';
 END IF;
 RETURN QUERY SELECT x.posicion,x.ref FROM (
  SELECT r.origen_posicion AS posicion,r.origen_ref AS ref FROM vec_bolsa_llamamientos.restriccion_cese_bolsa r
  UNION ALL
  SELECT a.origen_posicion,a.origen_ref FROM vec_bolsa_llamamientos.cese_ajeno_bolsa a
  UNION ALL
  SELECT s.origen_posicion,s.origen_ref FROM vec_bolsa_llamamientos.cese_sin_candidato_bolsa s
 ) x ORDER BY x.posicion DESC,x.ref DESC LIMIT 1;
END $function$;

DO $post$
DECLARE f regprocedure:='vec_bolsa_llamamientos.confirmar_cese_sin_candidato_bolsa_v1(text,text,bigint)'::regprocedure;
 c regprocedure:='vec_bolsa_llamamientos.cursor_restriccion_cese_bolsa_v1()'::regprocedure;
 p regprocedure:='vec_bolsa_llamamientos.listar_ceses_sin_candidato_pendientes_v1(integer)'::regprocedure;
BEGIN
 IF (SELECT array_agg(a.grantee::regrole::text ORDER BY a.grantee::regrole::text) FROM pg_proc p
       CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f)
      IS DISTINCT FROM ARRAY['vec_bolsa_llamamientos_propietario','vec_bolsa_llamamientos_relevo_cese']
    OR (SELECT array_agg(a.grantee::regrole::text ORDER BY a.grantee::regrole::text) FROM pg_proc p
       CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=c)
      IS DISTINCT FROM ARRAY['vec_bolsa_llamamientos_propietario','vec_bolsa_llamamientos_relevo_cese']
    OR (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'vec_bolsa_llamamientos_propietario'::regrole
    OR (SELECT proowner FROM pg_proc WHERE oid=p) IS DISTINCT FROM 'vec_bolsa_llamamientos_propietario'::regrole
    OR (SELECT prosecdef FROM pg_proc WHERE oid=c) IS NOT TRUE
    OR (SELECT prosecdef FROM pg_proc WHERE oid=p) IS NOT TRUE
    OR (SELECT proconfig FROM pg_proc WHERE oid=c) IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp']
    OR (SELECT 'search_path=pg_catalog, pg_temp'=ANY(p.proconfig) FROM pg_proc p WHERE p.oid=f) IS NOT TRUE
    OR (SELECT 'search_path=pg_catalog, pg_temp'=ANY(q.proconfig) FROM pg_proc q WHERE q.oid=p) IS NOT TRUE
    OR (SELECT array_agg(a.grantee::regrole::text ORDER BY a.grantee::regrole::text) FROM pg_proc q
        CROSS JOIN LATERAL aclexplode(coalesce(q.proacl,acldefault('f',q.proowner))) a WHERE q.oid=p)
       IS DISTINCT FROM ARRAY['vec_bolsa_llamamientos_propietario','vec_bolsa_llamamientos_relevo_cese']
    OR has_table_privilege('vec_bolsa_llamamientos_relevo_cese','vec_bolsa_llamamientos.cese_sin_candidato_bolsa','SELECT,INSERT,UPDATE,DELETE')
    OR has_table_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.cese_sin_candidato_bolsa','SELECT,INSERT,UPDATE,DELETE') THEN
  RAISE EXCEPTION 'Bolsa 000081: postimagen incompatible' USING ERRCODE='55000';
 END IF;
END $post$;
COMMIT;
