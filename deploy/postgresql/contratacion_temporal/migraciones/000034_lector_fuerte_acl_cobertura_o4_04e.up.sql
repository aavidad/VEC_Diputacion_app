-- O4-04E/11: lector fuerte primario, barrera final y ACL exterior.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_catalog.pg_advisory_xact_lock(
 pg_catalog.hashtextextended(
  'vec_contratacion_temporal:o4_04:migraciones',0));
SELECT control FROM vec_contratacion_temporal.control_migracion_cobertura_o4
 WHERE control AND version_esquema=13 FOR UPDATE;
DO $prevalidacion$
DECLARE
 v_tabla text;
BEGIN
 IF NOT EXISTS(
  SELECT 1 FROM vec_contratacion_temporal.control_migracion_cobertura_o4
   WHERE control AND version_esquema=13
 ) OR pg_catalog.to_regprocedure(
  'vec_contratacion_temporal.leer_terminal_primario_decision_cobertura_o404e_v1(jsonb)'
 ) IS NOT NULL OR NOT EXISTS(
  SELECT 1 FROM pg_catalog.pg_roles
   WHERE rolname='vec_contratacion_temporal_propietario'
     AND NOT rolcanlogin AND NOT rolsuper AND NOT rolbypassrls
 ) OR NOT EXISTS(
  SELECT 1 FROM pg_catalog.pg_roles
   WHERE rolname='vec_contratacion_temporal_confirmador_cobertura'
     AND NOT rolcanlogin AND NOT rolsuper AND NOT rolbypassrls
 ) THEN
  RAISE EXCEPTION USING ERRCODE='55000',
   MESSAGE='estado incompatible para lector fuerte O4-04E';
 END IF;
 FOREACH v_tabla IN ARRAY ARRAY[
  'acreditacion_gobierno_decision_cobertura',
  'auditoria_decision_cobertura',
  'confirmacion_operacion_decision_cobertura',
  'decision_cobertura_gobernada_durable',
  'prueba_denegacion_decision_cobertura',
  'terminal_operacion_decision_cobertura'
 ]::text[] LOOP
  IF NOT EXISTS(
   SELECT 1 FROM pg_catalog.pg_class c
   JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
   WHERE n.nspname='vec_contratacion_temporal' AND c.relname=v_tabla
     AND c.relrowsecurity AND c.relforcerowsecurity
  ) THEN
   RAISE EXCEPTION USING ERRCODE='55000',
    MESSAGE='RLS/FORCE RLS incompleto en '||v_tabla;
  END IF;
 END LOOP;
END
$prevalidacion$;

CREATE FUNCTION
vec_contratacion_temporal.leer_terminal_primario_decision_cobertura_o404e_v1(
 p_consulta jsonb
)
RETURNS TABLE(resultado_json jsonb)
LANGUAGE plpgsql
STABLE
SECURITY DEFINER
SET search_path=pg_catalog
SET row_security='on'
SET timezone='UTC'
SET lock_timeout='2s'
AS $funcion$
DECLARE
 v_base record;
 v_recibo jsonb;
 v_encontrado boolean:=false;
 v_observada timestamptz(6);
BEGIN
 v_observada:=pg_catalog.date_trunc(
  'microseconds',pg_catalog.clock_timestamp());
 IF current_user<>'vec_contratacion_temporal_propietario'
    OR session_user=current_user
    OR NOT pg_catalog.pg_has_role(
      session_user,
      'vec_contratacion_temporal_confirmador_cobertura','MEMBER')
    OR (SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_auth_members m
          JOIN pg_catalog.pg_roles u ON u.oid=m.member
         WHERE u.rolname=session_user)<>1
    OR NOT EXISTS(
      SELECT 1 FROM pg_catalog.pg_auth_members m
      JOIN pg_catalog.pg_roles u ON u.oid=m.member
      JOIN pg_catalog.pg_roles r ON r.oid=m.roleid
      WHERE u.rolname=session_user
        AND r.rolname='vec_contratacion_temporal_confirmador_cobertura'
        AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)
    OR pg_catalog.pg_has_role(
      session_user,'vec_contratacion_temporal_propietario','MEMBER')
    OR pg_catalog.pg_has_role(
      session_user,'vec_contratacion_temporal_migrador','MEMBER')
    OR pg_catalog.pg_has_role(
      session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
    OR pg_catalog.pg_has_role(
      session_user,'vec_contratacion_temporal_gobernador','MEMBER')
    OR vec_contratacion_temporal.gobi_o404b_entorno_valido(true)
       IS NOT TRUE
    OR pg_catalog.pg_is_in_recovery()
    OR p_consulta IS NULL
    OR pg_catalog.jsonb_typeof(p_consulta)<>'object'
    OR pg_catalog.pg_column_size(p_consulta)>65536
    OR NOT vec_contratacion_temporal.o404e_claves_exactas_v1(
      p_consulta,ARRAY[
       'correlacion_vec_ref','decision_vec_ref','esquema',
       'expediente_ref','huella_orden_sha256','organizacion_ref',
       'recibo_ref','reserva_ref','revision_cercado',
       'version_expediente']::text[])
    OR p_consulta->>'esquema'<>
      'vec.contratacion-temporal.consulta-primaria-decision-cobertura.o4-04e.v1'
    OR (p_consulta->>'huella_orden_sha256')!~'^[a-f0-9]{64}$'
    OR vec_contratacion_temporal.numero_entero_json_canonico_v2(
      p_consulta->'version_expediente',2,9007199254740990::numeric)
      IS NOT TRUE
    OR vec_contratacion_temporal.numero_entero_json_canonico_v2(
      p_consulta->'revision_cercado',1,9007199254740991::numeric)
      IS NOT TRUE THEN
  RAISE EXCEPTION USING ERRCODE='42501',
   MESSAGE='lectura primaria O4-04E no autorizada';
 END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock_shared(
 pg_catalog.hashtextextended(
   'vec_contratacion_temporal:o4_04:migraciones',0));
 PERFORM 1 FROM vec_contratacion_temporal.control_migracion_cobertura_o4
  WHERE control AND version_esquema=14;
 IF NOT FOUND THEN
  RAISE EXCEPTION USING ERRCODE='55000',
   MESSAGE='barrera primaria O4-04E no disponible';
 END IF;
 SELECT b.* INTO v_base
  FROM vec_contratacion_temporal.reserva_operacion_decision_cobertura b
  JOIN vec_contratacion_temporal.confirmacion_operacion_decision_cobertura c
   USING(ambito_raiz_hmac)
  WHERE b.organizacion_ref=p_consulta->>'organizacion_ref'
    AND b.expediente_ref=p_consulta->>'expediente_ref'
    AND b.version_expediente=
      (p_consulta->>'version_expediente')::numeric
    AND b.reserva_ref=p_consulta->>'reserva_ref'
    AND b.recibo_ref=p_consulta->>'recibo_ref'
    AND b.correlacion_vec_ref=p_consulta->>'correlacion_vec_ref'
    AND b.decision_vec_ref=p_consulta->>'decision_vec_ref'
    AND c.revision_cercado=(p_consulta->>'revision_cercado')::numeric
    AND c.huella_orden_sha256=p_consulta->>'huella_orden_sha256';
 IF FOUND THEN
  v_recibo:=vec_contratacion_temporal.o404e_leer_terminal_interno_v1(
   v_base.ambito_raiz_hmac,p_consulta->>'huella_orden_sha256');
  v_encontrado:=v_recibo IS NOT NULL;
 END IF;
 RETURN QUERY SELECT pg_catalog.jsonb_build_object(
  'esquema',
   'vec.contratacion-temporal.resultado-primario-decision-cobertura.o4-04e.v1',
  'encontrado',v_encontrado,
  'consulta',CASE WHEN v_encontrado THEN p_consulta ELSE NULL END,
  'recibo',CASE WHEN v_encontrado THEN v_recibo ELSE NULL END,
  'observada_en_primario',
   vec_contratacion_temporal.texto_instante_utc_go_v2(v_observada::text)
 );
END
$funcion$;

UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
 SET version_esquema=14,
 actualizada_en=pg_catalog.date_trunc(
  'microseconds',pg_catalog.clock_timestamp())
 WHERE control AND version_esquema=13;

REVOKE ALL ON FUNCTION
 vec_contratacion_temporal.confirmar_operacion_decision_cobertura_o404e_v1(
  jsonb),
 vec_contratacion_temporal.leer_terminal_primario_decision_cobertura_o404e_v1(
  jsonb)
FROM PUBLIC,vec_contratacion_temporal_migrador,
 vec_contratacion_temporal_gobernador,
 vec_contratacion_temporal_ejecutor;
GRANT USAGE ON SCHEMA vec_contratacion_temporal
 TO vec_contratacion_temporal_confirmador_cobertura;
GRANT EXECUTE ON FUNCTION
 vec_contratacion_temporal.confirmar_operacion_decision_cobertura_o404e_v1(
  jsonb),
 vec_contratacion_temporal.leer_terminal_primario_decision_cobertura_o404e_v1(
  jsonb)
TO vec_contratacion_temporal_confirmador_cobertura;
COMMIT;
