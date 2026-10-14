\set ON_ERROR_STOP on
-- Ejecutar tras CT213 y B115. Comprueba el contrato instalado sin crear actos.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL statement_timeout='10s';
DO $test$
DECLARE b text; a text; ct text; v_evento_ref text:='evento:inexistente'; v_ok boolean;
BEGIN
 SELECT pg_get_functiondef(to_regprocedure(
  'vec_bolsa_llamamientos.aplicar_cese_nativo_ct_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')) INTO b;
 SELECT pg_get_functiondef(to_regprocedure(
  'vec_bolsa_llamamientos.aplicar_alta_nativa_ct_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')) INTO a;
 SELECT pg_get_functiondef(to_regprocedure(
  'vec_contratacion_temporal.probar_cese_personal_b2_para_bolsa_v1(text)')) INTO ct;
 IF b IS NULL OR a IS NULL OR ct IS NULL THEN
  RAISE EXCEPTION 'B115: funciones nativas ausentes';
 END IF;

 -- Compila la búsqueda de evento con la variable PL/pgSQL y la tabla real.
 PERFORM 1 FROM vec_bolsa_llamamientos.contrato_participacion x
  WHERE x.evento_ref=v_evento_ref;
 IF position('b.evento_ref=v_evento_ref' in a)=0
    OR position('''incorporacion''||chr(31)||(c->>''origen_ref'')' in a)=0
    OR position('''cese''||chr(31)||(c->>''origen_ref'')' in b)=0
    OR position('''bolsa:alta-nativa:''||(personal->>''recibo_ct_ref'')' in a)=0
    OR position('politica.mapeo->>((c->>''modalidad_clave'')||''|''||(c->>''causa_contrato_clave''))' in b)=0
    OR position('estado_cese_bolsa_lote_interno_v2' in a)=0
    OR position('LEFT JOIN LATERAL vec_bolsa_llamamientos.estado_cese_bolsa_lote_interno_v2' in a)=0
    OR position('coalesce(cese_actual.cese_pendiente,false) IS FALSE' in a)=0
    OR position('coalesce(cese_actual.en_restriccion,false) IS FALSE' in a)=0
    OR position('probar_cierre_cese_ct_v1' in b)=0
    OR position('OR hasta NOT IN (fecha,fecha+1)' in b)=0
    OR position('cese_actual.cese_pendiente IS FALSE' in a)=0
    OR position('cese_actual.en_restriccion IS FALSE' in a)=0
    OR position('cese_actual.trabajo_cesado IS TRUE' in a)=0
    OR position('c->>''relacion_recibo_ref'' IS DISTINCT FROM personal->>''relacion_recibo_ref''' in b)>0
    OR position('c->>''ocupacion_recibo_ref'' IS DISTINCT FROM personal->>''ocupacion_recibo_ref''' in b)>0
    OR position('''relacion_recibo_ref'',h->>''relacion_recibo_ref''' in ct)>0
    OR position('''ocupacion_recibo_ref'',h->>''ocupacion_recibo_ref''' in ct)>0
 THEN RAISE EXCEPTION 'B115: contrato de cese SQL divergente'; END IF;

 -- B90 no devuelve fila para una participación que nunca tuvo cese.
 -- La primera alta permite esa ausencia; la segunda exige cese acreditado.
 WITH casos(situacion,trabajo_cesado,cese_pendiente,en_restriccion,disponible_desde,esperado) AS (
  VALUES
   ('disponible',NULL::boolean,NULL::boolean,NULL::boolean,NULL::date,true),
   ('trabajando',true,false,false,date '2026-10-13',true),
   ('trabajando',true,false,false,date '2026-10-15',false),
   ('trabajando',true,true,false,date '2026-10-13',false),
   ('trabajando',true,false,true,date '2026-10-13',false),
   ('trabajando',NULL::boolean,NULL::boolean,NULL::boolean,NULL::date,false)
 )
 SELECT bool_and((
   (situacion='disponible' AND coalesce(cese_pendiente,false) IS FALSE
      AND coalesce(en_restriccion,false) IS FALSE)
   OR (situacion='trabajando' AND trabajo_cesado IS TRUE
      AND cese_pendiente IS FALSE AND en_restriccion IS FALSE
      AND disponible_desde<=date '2026-10-14')
  ) IS NOT DISTINCT FROM esperado) INTO v_ok FROM casos;
 IF v_ok IS NOT TRUE THEN RAISE EXCEPTION 'B115: disponibilidad de primera o segunda alta divergente'; END IF;

 IF NOT (date '2026-10-14' IN (date '2026-10-14',date '2026-10-13'+1))
    OR NOT (date '2026-10-15' IN (date '2026-10-14',date '2026-10-14'+1))
    OR date '2026-10-13' IN (date '2026-10-14',date '2026-10-14'+1)
    OR date '2026-10-16' IN (date '2026-10-14',date '2026-10-14'+1)
 THEN RAISE EXCEPTION 'B115: intervalo de cierre divergente'; END IF;
END $test$;
ROLLBACK;
