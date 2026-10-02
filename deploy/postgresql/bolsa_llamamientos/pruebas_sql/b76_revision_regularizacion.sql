\set ON_ERROR_STOP on
-- Ensayo Bolsa; el consumidor AD3 es DOBLE transaccional y no acredita firma
-- criptográfica. Todo, incluido el doble, se revierte al finalizar.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL statement_timeout='30s';
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_situacion_participacion_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,
 auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
BEGIN
 IF p_capacidad IS DISTINCT FROM '\x00'::bytea THEN
  RAISE EXCEPTION 'autorizacion sintética retirada' USING ERRCODE='42501';
 END IF;
 RETURN QUERY SELECT 'decision:doble-b76',convert_from(p_payload,'UTF8'),
 convert_from(p_decision,'UTF8')::jsonb->>'contexto_recurso_huella_sha256',repeat('0',64),
 'auditoria:doble-b76',clock_timestamp(),true;
END $f$;
CREATE FUNCTION pg_temp.b76_operar(b text,p text,op text,t timestamptz,k text,esperada timestamptz,
 fin timestamptz DEFAULT NULL,doc text DEFAULT 'justificante:b76',actor_dec text DEFAULT 'persona:rrhh-b76',
 recurso_dec text DEFAULT NULL,cap bytea DEFAULT '\x00',validada timestamptz DEFAULT NULL,
 actor text DEFAULT 'persona:rrhh-b76',version integer DEFAULT 2,desde_recibo_esperado timestamptz DEFAULT NULL)
RETURNS text LANGUAGE plpgsql AS $f$
DECLARE d bytea; r record;
BEGIN
 d:=convert_to(jsonb_build_object('principal_id',actor_dec,'accion','bolsa.situacion_participacion.cambiar',
 'modulo_id','bolsa','tipo_recurso','participacion_bolsa','finalidad','gestion_situacion_participacion',
 'recurso_ref',p,'campos_permitidos','[]'::jsonb,'obligaciones','[]'::jsonb,
 'contexto_recurso_huella_sha256',repeat('a',64))::text,'UTF8');
 SET LOCAL ROLE vec_bolsa_llamamientos_ejecutor;
 IF version=1 THEN
 SELECT * INTO STRICT r FROM vec_bolsa_llamamientos.registrar_operacion_situacion_participacion_v1(
 b,p,op,t,NULL,'Fixture histórico B76',actor,k,'recibo:'||k,t,
 'resolucion',doc,repeat('b',64),'persona:validador-b76',coalesce(validada,t),
 cap,d,'\x00','\x00',1,1,convert_to(coalesce(recurso_dec,p),'UTF8'),'\x00','\x00','\x00');
 ELSE
 SELECT * INTO STRICT r FROM vec_bolsa_llamamientos.registrar_operacion_situacion_participacion_v2(
 b,p,op,t,NULL,'Fixture histórico B76',actor,k,'recibo:'||k,t,
 'resolucion',doc,repeat('b',64),'persona:validador-b76',coalesce(validada,t),esperada,fin,
 cap,d,'\x00','\x00',1,1,convert_to(coalesce(recurso_dec,p),'UTF8'),'\x00','\x00','\x00');
 END IF;
 RESET ROLE;
 IF desde_recibo_esperado IS NOT NULL AND r.desde IS DISTINCT FROM desde_recibo_esperado THEN
  RAISE EXCEPTION 'B76 recibo no conserva el instante original';
 END IF;
 RETURN CASE WHEN r.reutilizada THEN 'replay:' ELSE 'nuevo:' END||r.recibo_ref||':'||r.situacion;
EXCEPTION WHEN OTHERS THEN RESET ROLE; RETURN 'error:'||SQLSTATE;
END $f$;
CREATE FUNCTION pg_temp.b76_b2(b text,p text,s text,t timestamptz,k text)
RETURNS text LANGUAGE plpgsql AS $f$
DECLARE d bytea; r record;
BEGIN
 d:=convert_to(jsonb_build_object('principal_id','persona:rrhh-b76','accion','bolsa.situacion_participacion.cambiar',
 'modulo_id','bolsa','tipo_recurso','participacion_bolsa','finalidad','gestion_situacion_participacion',
 'recurso_ref',p,'campos_permitidos','[]'::jsonb,'obligaciones','[]'::jsonb,
 'contexto_recurso_huella_sha256',repeat('a',64))::text,'UTF8');
 SET LOCAL ROLE vec_bolsa_llamamientos_ejecutor;
 SELECT * INTO STRICT r FROM vec_bolsa_llamamientos.registrar_situacion_participacion_v1(
 b,p,s,t,NULL,'Fixture histórico B76','persona:rrhh-b76',k,'recibo:'||k,t,
 '\x00',d,'\x00','\x00',1,1,convert_to(p,'UTF8'),'\x00','\x00','\x00');
 RESET ROLE;
 RETURN CASE WHEN r.reutilizada THEN 'replay:' ELSE 'nuevo:' END||r.recibo_ref||':'||r.situacion;
EXCEPTION WHEN OTHERS THEN RESET ROLE; RETURN 'error:'||SQLSTATE;
END $f$;
CREATE FUNCTION pg_temp.b76_assert(actual text,esperado text,caso text) RETURNS void
LANGUAGE plpgsql AS $f$
BEGIN IF actual IS DISTINCT FROM esperado THEN RAISE EXCEPTION 'B76 %: esperado %, actual %',caso,esperado,actual; END IF; END $f$;
DO $test$
DECLARE b text;p text;t timestamptz;legacy timestamptz;politica text[];v bigint;
 hist text;n bigint;ultima timestamptz;r text;
BEGIN
 SELECT c.bolsa_ref,s.participacion_ref,s.desde INTO STRICT b,p,legacy
 FROM vec_bolsa_llamamientos.situacion_participacion s
 JOIN vec_bolsa_llamamientos.constitucion_entrada e USING(participacion_ref)
 JOIN vec_bolsa_llamamientos.constitucion c USING(instantanea_ref,version_instantanea)
 WHERE s.clave_idempotencia='b76:legacy:reactivar';
 t:=legacy+interval '1 minute';
 SELECT md5(string_agg(s::text,'|' ORDER BY s.desde)),count(*) INTO hist,n
 FROM vec_bolsa_llamamientos.situacion_participacion s WHERE s.participacion_ref=p;
 SELECT version,transiciones INTO STRICT v,politica FROM vec_bolsa_llamamientos.politica_transiciones_situacion ORDER BY version DESC LIMIT 1;
 IF vec_bolsa_llamamientos.transiciones_situacion_canonicas(politica) IS DISTINCT FROM politica THEN
  RAISE EXCEPTION 'B76 política histórica dejó de ser canónica'; END IF;
 IF vec_bolsa_llamamientos.transiciones_situacion_canonicas(politica||ARRAY['renuncia>en_revision','en_revision>disponible']) IS NOT NULL THEN
  RAISE EXCEPTION 'B76 política revisión sin arista excluido admitida'; END IF;
 IF vec_bolsa_llamamientos.transiciones_situacion_canonicas(politica||ARRAY['excluido>en_revision','en_revision>excluido']) IS NOT NULL THEN
  RAISE EXCEPTION 'B76 catálogo permitió exclusión a revisión'; END IF;
 -- Replays antiguos, aun con situación posterior: permiso nuevo, mismos recibos.
 PERFORM pg_temp.b76_assert(pg_temp.b76_operar(b,p,'pausar',legacy-interval '1 second','b76:legacy:pausar',NULL,
 NULL,'justificante:b76:legacy',validada=>legacy-interval '2 second',version=>1),
 'replay:recibo:b76:legacy:pausar:no_disponible','replay pausa histórico');
 PERFORM pg_temp.b76_assert(pg_temp.b76_operar(b,p,'reactivar',legacy,'b76:legacy:reactivar',NULL,
 NULL,'justificante:b76:legacy',validada=>legacy-interval '2 second',version=>1),
 'replay:recibo:b76:legacy:reactivar:disponible','replay reactivación histórico');
 PERFORM pg_temp.b76_assert(pg_temp.b76_operar(b,p,'pausar',t,'b76:nueva-pausa',legacy,version=>1),
 'error:22023','pausa histórica efecto nuevo');
 PERFORM pg_temp.b76_assert(pg_temp.b76_operar(b,p,'reactivar',t,'b76:nueva-reactivar',legacy,version=>1),
 'error:22023','reactivación histórica efecto nuevo');
 PERFORM pg_temp.b76_assert(pg_temp.b76_operar(b,p,'revisar',t,'b76:revisar-disponible',legacy),
 'error:22023','revisión disponible rechazada');
 PERFORM pg_temp.b76_assert(pg_temp.b76_b2(b,p,'renuncia',t,'b76:renuncia'),
 'nuevo:recibo:b76:renuncia:renuncia','renuncia preparatoria');
 PERFORM pg_temp.b76_assert(pg_temp.b76_operar(b,p,'revisar',t+interval '1 second','b76:revision',t),
 'error:22023','política sin revisión');
 -- Publicación nueva, ninguna edición de políticas previas.
 SET LOCAL ROLE vec_bolsa_llamamientos_ejecutor;
 PERFORM * FROM vec_bolsa_llamamientos.publicar_politica_transiciones_situacion_v1('catalogo:b76:ensayo',repeat('c',64),
 politica||ARRAY['renuncia>en_revision','no_disponible>en_revision','en_revision>disponible','en_revision>excluido','excluido>disponible']);
 RESET ROLE;
 PERFORM pg_temp.b76_assert(pg_temp.b76_b2(b,p,'disponible',t+interval '1 second','b76:atajo-renuncia'),
 'error:22023','B2 regularización sin documento');
 PERFORM pg_temp.b76_assert(pg_temp.b76_b2(b,p,'en_revision',t+interval '1 second','b76:atajo-revision'),
 'error:22023','B2 revisión sin documento inicial');
 PERFORM pg_temp.b76_assert(pg_temp.b76_operar(b,p,'revisar',t+interval '1 second','b76:cas',legacy),
 'error:VBS02','CAS obsoleto');
 PERFORM pg_temp.b76_assert(pg_temp.b76_operar(b,p,'revisar',t+interval '1 second','b76:actor-ajeno',t,
 actor_dec=>'persona:ajena'), 'error:42501','actor ajeno');
 PERFORM pg_temp.b76_assert(pg_temp.b76_operar(b,p,'revisar',t+interval '1 second','b76:recurso-ajeno',t,
 recurso_dec=>'participacion:ajena'), 'error:42501','recurso ajeno');
 PERFORM pg_temp.b76_assert(pg_temp.b76_operar(b,p,'revisar',t+interval '1 second','b76:sin-permiso',t,
 cap=>'\x01'), 'error:42501','sin autorización');
 PERFORM pg_temp.b76_assert(pg_temp.b76_operar(b,p,'revisar',t+interval '1 second','b76:revision',t),
 'nuevo:recibo:b76:revision:en_revision','revisión');
 ultima:=t+interval '1 second';
 PERFORM pg_temp.b76_assert(pg_temp.b76_b2(b,p,'disponible',t+interval '2 second','b76:atajo-disponible'),
 'error:22023','B2 salida revisión');
 PERFORM pg_temp.b76_assert(pg_temp.b76_b2(b,p,'excluido',t+interval '2 second','b76:atajo-excluido'),
 'error:22023','B2 exclusión revisión');
 PERFORM pg_temp.b76_assert(pg_temp.b76_operar(b,p,'regularizar',t+interval '2 second','b76:regularizacion',ultima),
 'error:22023','sin fin de causa');
 PERFORM pg_temp.b76_assert(pg_temp.b76_operar(b,p,'regularizar',t+interval '2 second','b76:futuro',ultima,t+interval '1 day'),
 'error:22023','fin de causa futuro');
 PERFORM pg_temp.b76_assert(pg_temp.b76_operar(b,p,'regularizar',t+interval '2 second','b76:regularizacion',ultima,t),
 'nuevo:recibo:b76:regularizacion:disponible','fin de causa validado');
 PERFORM pg_temp.b76_assert(pg_temp.b76_operar(b,p,'revisar',t+interval '1 second','b76:revision',t),
 'replay:recibo:b76:revision:en_revision','replay revisión después regularizar');
 PERFORM pg_temp.b76_assert(pg_temp.b76_operar(b,p,'regularizar',t+interval '2 second','b76:regularizacion',ultima,t),
 'replay:recibo:b76:regularizacion:disponible','replay regularización');
 PERFORM pg_temp.b76_assert(pg_temp.b76_operar(b,p,'regularizar',t+interval '2 second','b76:regularizacion',ultima,t,
 doc=>'justificante:alterado'),'error:VBS01','replay documento alterado');
 PERFORM pg_temp.b76_assert(pg_temp.b76_operar(b,p,'regularizar',t+interval '2 second','b76:regularizacion',ultima,t-interval '1 second'),
 'error:VBS01','replay fin causa alterado');
 PERFORM pg_temp.b76_assert(pg_temp.b76_operar(b,p,'regularizar',t+interval '2 second','b76:regularizacion',legacy,t),
 'error:VBS01','replay CAS alterado');
 -- Retry de la petición real: Go toma de nuevo ambos instantes del reloj.
 PERFORM pg_temp.b76_assert(pg_temp.b76_operar(b,p,'revisar',t+interval '1 day','b76:revision',t,
 validada=>t+interval '1 day',desde_recibo_esperado=>t+interval '1 second'),
 'replay:recibo:b76:revision:en_revision','replay revisión con reloj posterior');
 PERFORM pg_temp.b76_assert(pg_temp.b76_operar(b,p,'regularizar',t+interval '1 day','b76:regularizacion',ultima,t,
 validada=>t+interval '1 day',desde_recibo_esperado=>t+interval '2 second'),
 'replay:recibo:b76:regularizacion:disponible','replay regularización con reloj posterior');
 IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.operacion_situacion_participacion o
 JOIN vec_bolsa_llamamientos.situacion_participacion s USING(participacion_ref,desde)
 WHERE o.participacion_ref=p AND o.clave_idempotencia IN ('b76:revision','b76:regularizacion')
 AND (s.desde IS DISTINCT FROM CASE o.operacion WHEN 'revisar' THEN t+interval '1 second' ELSE t+interval '2 second' END
   OR o.validada_en IS DISTINCT FROM s.desde OR s.registrada_en IS DISTINCT FROM s.desde
   OR o.registrada_en IS DISTINCT FROM s.desde)) THEN
  RAISE EXCEPTION 'B76 replay alteró instantes de la primera ejecución'; END IF;
 PERFORM pg_temp.b76_assert(pg_temp.b76_operar(b,p,'regularizar',t+interval '1 day','b76:regularizacion',ultima,t,
 validada=>t+interval '1 day',cap=>'\x01'),'error:42501','replay autorización retirada');
 IF (SELECT count(*) FROM vec_bolsa_llamamientos.situacion_participacion WHERE participacion_ref=p)<>n+3
 OR (SELECT count(*) FROM vec_bolsa_llamamientos.operacion_situacion_participacion WHERE participacion_ref=p AND operacion IN ('revisar','regularizar'))<>2 THEN
  RAISE EXCEPTION 'B76 efecto duplicado'; END IF;
 IF (SELECT md5(string_agg(s::text,'|' ORDER BY s.desde)) FROM vec_bolsa_llamamientos.situacion_participacion s
 WHERE s.participacion_ref=p AND s.desde<t) IS DISTINCT FROM hist THEN RAISE EXCEPTION 'B76 historia alterada'; END IF;
 IF (SELECT transiciones FROM vec_bolsa_llamamientos.politica_transiciones_situacion WHERE version=v) IS DISTINCT FROM politica THEN
  RAISE EXCEPTION 'B76 política previa alterada'; END IF;
END $test$;

CREATE FUNCTION pg_temp.b76_fixture(p text,s text,t timestamptz,k text,op text DEFAULT NULL)
RETURNS void LANGUAGE plpgsql AS $f$
DECLARE v bigint;
BEGIN
 SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
 SELECT max(version) INTO v FROM vec_bolsa_llamamientos.politica_transiciones_situacion;
 INSERT INTO vec_bolsa_llamamientos.situacion_participacion
 (participacion_ref,situacion,desde,motivo,actor,registrada_en,clave_idempotencia,recibo_ref,politica_transiciones_version)
 VALUES(p,s,t,'Fixture histórico B76','persona:rrhh-b76',t,k,'recibo:'||k,v);
 IF op IS NOT NULL THEN
 INSERT INTO vec_bolsa_llamamientos.operacion_situacion_participacion
 (participacion_ref,desde,operacion,justificante_tipo,justificante_ref,justificante_sha256,
 actor,validador,validada_en,registrada_en,clave_idempotencia)
 VALUES(p,t,op,'resolucion','justificante:b76',repeat('b',64),
 'persona:rrhh-b76','persona:validador-b76',t,t,k);
 END IF;
 RESET ROLE;
END $f$;
DO $causal$
DECLARE b text;p text;t timestamptz;r text;
BEGIN
 SELECT c.bolsa_ref,s.participacion_ref INTO STRICT b,p
 FROM vec_bolsa_llamamientos.situacion_participacion s
 JOIN vec_bolsa_llamamientos.constitucion_entrada e USING(participacion_ref)
 JOIN vec_bolsa_llamamientos.constitucion c USING(instantanea_ref,version_instantanea)
 WHERE s.clave_idempotencia='b76:legacy:reactivar';
 SELECT max(desde)+interval '1 minute' INTO t FROM vec_bolsa_llamamientos.situacion_participacion WHERE participacion_ref=p;
 -- no_disponible arbitrario: no equivale a una renuncia justificada.
 PERFORM pg_temp.b76_fixture(p,'no_disponible',t,'b76:arbitraria','pausar');
 PERFORM pg_temp.b76_assert(pg_temp.b76_operar(b,p,'revisar',t+interval '1 second','b76:arbitraria-revision',t),
 'error:22023','pausa sin renuncia causal');
 PERFORM pg_temp.b76_assert(pg_temp.b76_b2(b,p,'disponible',t+interval '1 second','b76:atajo-no-disponible'),
 'error:22023','B2 salida no disponible');
 PERFORM pg_temp.b76_assert(pg_temp.b76_operar(b,p,'regularizar',t+interval '1 second','b76:regularizar-no-disponible',t,t),
 'error:22023','regularización salta revisión de pausa');
 -- renuncia -> no_disponible sin operación B8 no se adopta como justificada.
 PERFORM pg_temp.b76_fixture(p,'renuncia',t+interval '2 second','b76:renuncia-sin-b8');
 PERFORM pg_temp.b76_fixture(p,'no_disponible',t+interval '3 second','b76:no-disponible-sin-b8');
 PERFORM pg_temp.b76_assert(pg_temp.b76_operar(b,p,'revisar',t+interval '4 second','b76:sin-b8',t+interval '3 second'),
 'error:22023','sin acto inicial B8');
 -- Cadena histórica renuncia -> pausar B8: puede entrar en revisión.
 PERFORM pg_temp.b76_fixture(p,'renuncia',t+interval '4 second','b76:renuncia-con-b8');
 PERFORM pg_temp.b76_fixture(p,'no_disponible',t+interval '5 second','b76:pausa-con-b8','pausar');
 PERFORM pg_temp.b76_assert(pg_temp.b76_operar(b,p,'revisar',t+interval '6 second','b76:revision-con-b8',t+interval '5 second'),
 'nuevo:recibo:b76:revision-con-b8:en_revision','antecedente B8 causal');
 PERFORM pg_temp.b76_assert(pg_temp.b76_operar(b,p,'regularizar',t+interval '7 second','b76:fin-con-b8',t+interval '6 second',t+interval '5 second'),
 'nuevo:recibo:b76:fin-con-b8:disponible','regularización desde cadena causal');
 -- Exclusión ajena a una renuncia: no se convierte en renuncia justificada.
 PERFORM pg_temp.b76_assert(pg_temp.b76_operar(b,p,'excluir',t+interval '8 second','b76:exclusion-no-renuncia',t+interval '7 second'),
 'nuevo:recibo:b76:exclusion-no-renuncia:excluido','exclusión ajena');
 PERFORM pg_temp.b76_assert(pg_temp.b76_operar(b,p,'regularizar',t+interval '9 second','b76:atajo-excluido',t+interval '8 second',t),
 'error:22023','exclusión sin antecedente renuncia');
 PERFORM pg_temp.b76_assert(pg_temp.b76_operar(b,p,'revisar',t+interval '9 second','b76:revisar-excluido',t+interval '8 second'),
 'error:22023','excluido no usa revisión');
 -- Fixture histórico de renuncia excluida sin sanción: B73 conserva opt-in.
 PERFORM pg_temp.b76_fixture(p,'renuncia',t+interval '10 second','b76:renuncia-excluida');
 PERFORM pg_temp.b76_assert(pg_temp.b76_operar(b,p,'excluir',t+interval '11 second','b76:exclusion-renuncia',t+interval '10 second'),
 'nuevo:recibo:b76:exclusion-renuncia:excluido','exclusión de renuncia');
 PERFORM pg_temp.b76_assert(pg_temp.b76_operar(b,p,'regularizar',t+interval '12 second','b76:regularizar-exclusion',t+interval '11 second',t),
 'nuevo:recibo:b76:regularizar-exclusion:disponible','B73 renuncia justificada');
 -- Misma procedencia con sanción viva: permanece bloqueada por B73.
 PERFORM pg_temp.b76_fixture(p,'renuncia',t+interval '13 second','b76:renuncia-sancion');
 PERFORM pg_temp.b76_assert(pg_temp.b76_operar(b,p,'excluir',t+interval '14 second','b76:exclusion-sancion',t+interval '13 second'),
 'nuevo:recibo:b76:exclusion-sancion:excluido','exclusión sancionada');
 SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
 INSERT INTO vec_bolsa_llamamientos.sancion_participacion(
 sancion_ref,participacion_ref,bolsa_ref,consecuencia,consecuencia_etiqueta,efecto,causa,
 fecha_notificacion,resolucion_ref,resolucion_sha256,resuelta_por,regla_ref,regla_huella_sha256,
 suspension_hasta,recurso_vence,recurso_regla_ref,recurso_regla_huella_sha256,
 situacion_desde,recibo_ref,actor,registrada_en,clave_idempotencia)
 VALUES('sancion:'||encode(sha256(convert_to('b76-prueba-sancion','UTF8')),'hex'),p,b,
 'baja','Baja sintética','excluir','Prueba B76',current_date,'resolucion:b76',repeat('d',64),
 'persona:validador-b76','regla:b76',repeat('e',64),NULL,current_date+10,
 'regla:recurso:b76',repeat('f',64),t+interval '14 second','recibo:b76:sancion',
 'persona:rrhh-b76',t+interval '14 second','b76:sancion');
 RESET ROLE;
 PERFORM pg_temp.b76_assert(pg_temp.b76_operar(b,p,'regularizar',t+interval '15 second','b76:bloqueada',t+interval '14 second',t),
 'error:22023','sanción viva no eludida');
 -- Cada acto nuevo conserva su traza, identidad, documento y política.
 IF EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.operacion_situacion_participacion o
 WHERE o.operacion IN ('revisar','regularizar') AND o.participacion_ref=p
 AND (o.situacion_esperada_desde IS NULL OR NOT EXISTS
   (SELECT 1 FROM vec_bolsa_llamamientos.traza_valor_participacion a
    JOIN vec_bolsa_llamamientos.situacion_participacion s
      ON s.participacion_ref=a.participacion_ref AND s.recibo_ref=a.recibo_ref
    WHERE s.participacion_ref=o.participacion_ref AND s.desde=o.desde AND a.campo='situacion'))) THEN
  RAISE EXCEPTION 'B76 falta CAS o traza en acto nuevo'; END IF;
END $causal$;
DO $acl$
DECLARE f regprocedure; n text;
BEGIN
 FOREACH n IN ARRAY ARRAY['registrar_situacion_participacion_interna_b73','registrar_situacion_participacion_interna_b76'] LOOP
 SELECT p.oid::regprocedure INTO STRICT f FROM pg_proc p JOIN pg_namespace ns ON ns.oid=p.pronamespace
 WHERE ns.nspname='vec_bolsa_llamamientos' AND p.proname=n;
 IF has_function_privilege('vec_bolsa_llamamientos_ejecutor',f,'EXECUTE')
 OR EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
 WHERE p.oid=f AND a.grantee<>p.proowner AND a.privilege_type='EXECUTE') THEN
  RAISE EXCEPTION 'B76 helper accesible desde frontera: %',n; END IF;
 END LOOP;
 SELECT p.oid::regprocedure INTO STRICT f FROM pg_proc p JOIN pg_namespace ns ON ns.oid=p.pronamespace
 WHERE ns.nspname='vec_bolsa_llamamientos' AND p.proname='registrar_operacion_situacion_participacion_v2';
 IF NOT has_function_privilege('vec_bolsa_llamamientos_ejecutor',f,'EXECUTE')
 OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
 WHERE p.oid=f AND a.grantee NOT IN (p.proowner,'vec_bolsa_llamamientos_ejecutor'::regrole) AND a.privilege_type='EXECUTE') THEN
  RAISE EXCEPTION 'B76 fachada no conserva EXECUTE nominal'; END IF;
 IF has_table_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.operacion_situacion_participacion','INSERT')
 OR has_table_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.situacion_participacion','INSERT') THEN
  RAISE EXCEPTION 'B76 ejecutor escribe tablas directamente'; END IF;
END $acl$;

SELECT 'b76_revision_regularizacion: correcto' AS resultado;
ROLLBACK;
