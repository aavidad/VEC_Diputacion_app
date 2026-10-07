\set ON_ERROR_STOP on
-- Sólo en copia sintética PG18 con AD211/P22→AD175/P32→AD180/P34→AD214.
-- Contrato estructural y rechazo temprano; sin decisión V3 firmada fabricada.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
DO $prueba$
DECLARE
 nucleo oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 definicion text; fuente text; audiencia text; actual text;
BEGIN
 IF nucleo IS NULL THEN RAISE EXCEPTION 'AD214 prueba: falta núcleo' USING ERRCODE='55000'; END IF;
 SELECT pg_get_functiondef(nucleo),p.prosrc INTO STRICT definicion,fuente FROM pg_proc p WHERE p.oid=nucleo;
 IF encode(sha256(convert_to(definicion,'UTF8')),'hex') IS DISTINCT FROM
     '63afb3d4e6f33d4ee8efce1e54d7e8f92ac9f8cec58506c2af146e88dca4552e'
    OR encode(sha256(convert_to(fuente,'UTF8')),'hex') IS DISTINCT FROM
     '78137d8750422597c0da56797fd3ddff797cd54f18d98b0c1c8f4b7f742079dd'
    OR strpos(fuente,'resolver_origen_consumo_v1')=0
    OR strpos(fuente,'transaccion_origen')=0 OR strpos(fuente,'consumo_confirmado_v4')=0
 THEN RAISE EXCEPTION 'AD214 prueba: núcleo/origen/familia v4 divergentes' USING ERRCODE='55000'; END IF;
 IF strpos(fuente,'personal.vinculo_propio.crn11.consultar')<>0
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_vinculo_propio_crn11_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR strpos(fuente,'personal.rpt_publica.consultar')<>0
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_rpt_publica_v2_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 THEN RAISE EXCEPTION 'AD214 prueba: CRN11/RPT añadidos fuera de alcance' USING ERRCODE='55000'; END IF;
 IF to_regprocedure('vec_autorizacion_atestada_v3.consumir_consulta_rrhh_bolsa_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 THEN RAISE EXCEPTION 'AD214 prueba: fachada AD213 instalada fuera de orden' USING ERRCODE='55000'; END IF;
 IF strpos(fuente,'bolsa.rrhh.bolsas.consultar')=0
    OR strpos(fuente,'bolsa.rrhh.estadisticas.consultar')=0
    OR strpos(fuente,'bolsa.rrhh.candidatos.consultar')=0
    OR strpos(fuente,'consulta_rrhh_bolsa')=0
    OR strpos(fuente,'vec_bolsa_llamamientos_ejecutor')=0
    OR strpos(fuente,'["bolsas","conteos"]')=0
    OR strpos(fuente,'["candidatos","contactos","turno"]')=0
    OR strpos(fuente,'interna_corporativa')=0
 THEN RAISE EXCEPTION 'AD214 prueba: tuplas Bolsa incompletas' USING ERRCODE='55000'; END IF;
 SELECT pg_get_constraintdef(c.oid,false) INTO STRICT actual FROM pg_constraint c
  WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
    AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 IF encode(sha256(convert_to(actual,'UTF8')),'hex') IS DISTINCT FROM
     '8dae0267b85ad237770d0ccdb7fbf9862afe6d7d022c0c93f4385c182bcbc68e'
    OR strpos(actual,'vec_personal.rpt_publica.consultar.v2')<>0
 THEN RAISE EXCEPTION 'AD214 prueba: CHECK de audiencias divergente' USING ERRCODE='55000'; END IF;
 FOREACH audiencia IN ARRAY ARRAY[
  'vec_bolsa_llamamientos.rrhh.bolsas.consultar.v1',
  'vec_bolsa_llamamientos.rrhh.estadisticas.consultar.v1',
  'vec_bolsa_llamamientos.rrhh.candidatos.consultar.v1'] LOOP
  IF strpos(actual,audiencia)=0 THEN RAISE EXCEPTION 'AD214 prueba: audiencia ausente %',audiencia USING ERRCODE='55000'; END IF;
 END LOOP;
 IF NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=nucleo
      AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
      AND p.prosecdef AND p.provolatile='v' AND p.proparallel='u'
      AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s'])
    OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL
      aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=nucleo)<>1
    OR EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL
      aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
      WHERE p.oid=nucleo AND (a.grantee<>p.proowner OR a.grantor<>p.proowner
        OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
 THEN RAISE EXCEPTION 'AD214 prueba: ACL/ABI del núcleo abierta' USING ERRCODE='55000'; END IF;
END $prueba$;
ROLLBACK;

BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
DO $sin_efecto$
DECLARE antes_consumo bigint; antes_auditoria bigint;
BEGIN
 SELECT count(*) INTO antes_consumo FROM vec_autorizacion_atestada_v3.consumo_decision_v3;
 SELECT count(*) INTO antes_auditoria FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3;
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
   'consulta_rrhh_bolsa',NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'AD214 prueba: material nulo admitido' USING ERRCODE='P0001';
 EXCEPTION WHEN insufficient_privilege THEN NULL;
 END;
 IF (SELECT count(*) FROM vec_autorizacion_atestada_v3.consumo_decision_v3)<>antes_consumo
    OR (SELECT count(*) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3)<>antes_auditoria
 THEN RAISE EXCEPTION 'AD214 prueba: denegación creó consumo o auditoría' USING ERRCODE='55000'; END IF;
END $sin_efecto$;
ROLLBACK;
