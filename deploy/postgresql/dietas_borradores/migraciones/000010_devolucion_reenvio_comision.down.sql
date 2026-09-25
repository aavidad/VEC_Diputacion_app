\set ON_ERROR_STOP on
-- Solo para PostgreSQL desechable. Nunca en una base con historia: una
-- corrección o un reenvío registrados desde «devuelta» impiden el DOWN.
BEGIN;
RESET ROLE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_dietas:migracion:000010:devolucion:v1',0));
DO $preimagen_dba$
BEGIN
 IF current_user IS DISTINCT FROM session_user
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
    OR to_regprocedure('vec_dietas.devolucion_vigente_comision_v1(text,bigint)') IS NULL
    OR to_regprocedure('vec_dietas.impedir_eliminar_tras_circuito_v1()') IS NULL
    OR NOT EXISTS (SELECT 1 FROM pg_proc p
        WHERE p.oid=to_regprocedure('vec_dietas.proyectar_comision_v2(text,boolean)')
          AND md5(p.prosrc)='3cf1b715a1711dfffd34d63cbbbc1397')
    OR NOT EXISTS (SELECT 1 FROM pg_proc p
        WHERE p.oid=to_regprocedure('vec_dietas.proyectar_revision_exacta_v2(text,bigint)')
          AND md5(p.prosrc)='4dcae3043307e3bed4d44267c06a8692')
 THEN RAISE EXCEPTION 'Dietas 000010 DOWN requiere DBA y preimagen completa' USING ERRCODE='55000'; END IF;
END $preimagen_dba$;
-- El DBA ve todas las filas pese a FORCE RLS contextual.
LOCK TABLE vec_dietas.comision_revision IN ACCESS EXCLUSIVE MODE;
DO $historia_dba$
BEGIN
 IF EXISTS (SELECT 1 FROM vec_dietas.historia_operacion_comision WHERE estado_anterior='devuelta')
 THEN RAISE EXCEPTION 'Dietas 000010 DOWN protege correcciones y reenvíos registrados' USING ERRCODE='55000'; END IF;
END $historia_dba$;
SET LOCAL ROLE vec_dietas_propietario;
DROP TRIGGER impedir_eliminar_tras_circuito ON vec_dietas.comision_revision;
DROP FUNCTION vec_dietas.impedir_eliminar_tras_circuito_v1();
CREATE OR REPLACE FUNCTION vec_dietas.proyectar_comision_v2(p_ref text,p_repeticion boolean)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on SET timezone='UTC' AS $$
DECLARE b vec_dietas.borrador_comision%ROWTYPE; x vec_dietas.comision_revision%ROWTYPE;
        r vec_dietas.recibo_operacion_comision%ROWTYPE; r1 vec_dietas.recibo_borrador_comision%ROWTYPE;
        regla_creacion vec_dietas.recibo_regla_creacion_comision%ROWTYPE;
        numero vec_dietas.numero_documento_comision%ROWTYPE;
        calc jsonb; comision jsonb; recibo jsonb;
BEGIN
 IF current_user<>'vec_dietas_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_dietas_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_dietas_propietario','MEMBER') THEN
  RAISE EXCEPTION 'proyección Dietas inválida' USING ERRCODE='42501'; END IF;
 SELECT * INTO b FROM vec_dietas.borrador_comision
 WHERE referencia=p_ref AND persona_ref=current_setting('vec.dietas.persona_ref',true);
 IF NOT FOUND THEN RETURN jsonb_build_object('resultado','no_encontrado'); END IF;
 SELECT * INTO STRICT numero FROM vec_dietas.numero_documento_comision WHERE comision_ref=p_ref;
 SELECT * INTO x FROM vec_dietas.comision_revision WHERE comision_ref=p_ref ORDER BY version DESC LIMIT 1;
 IF FOUND THEN
  SELECT * INTO STRICT r FROM vec_dietas.recibo_operacion_comision
   WHERE comision_ref=p_ref AND version=x.version;
  comision:=jsonb_build_object('referencia',p_ref,'numero_documento',numero.numero_documento,
   'fecha_apertura',to_char(numero.fecha_apertura AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
   'estado',x.estado,'version',x.version,
   'fecha_inicio',x.fecha_inicio::text,'fecha_fin',x.fecha_fin::text,'motivo',x.motivo,
   'codigos_ruta',x.codigos_ruta,'relacion_ref',b.relacion_ref,
   'unidad_ref',b.unidad_ref,'centro_ref',x.centro_ref,'calculo',x.calculo,
   'documento',x.documento);
  IF x.vehiculo_propio IS NOT NULL THEN
   comision:=comision||jsonb_build_object('vehiculo_propio',x.vehiculo_propio,'rutas',x.rutas);
  END IF;
  recibo:=jsonb_build_object('referencia',r.referencia,'version',r.version,
   'regla_ref',r.regla_ref,'regla_huella_sha256',r.regla_huella_sha256,
   'registrado_en',to_char(r.registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
   'repeticion',p_repeticion);
 ELSE
  SELECT calculo INTO STRICT calc FROM vec_dietas.calculo_comision WHERE comision_ref=p_ref;
  SELECT * INTO STRICT r1 FROM vec_dietas.recibo_borrador_comision WHERE comision_ref=p_ref;
  comision:=jsonb_build_object('referencia',p_ref,'numero_documento',numero.numero_documento,
   'fecha_apertura',to_char(numero.fecha_apertura AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
   'estado','borrador','version',1,
   'fecha_inicio',b.fecha_inicio::text,'fecha_fin',b.fecha_fin::text,'motivo',b.motivo,
   'codigos_ruta',b.codigos_ruta,'relacion_ref',b.relacion_ref,
   'unidad_ref',b.unidad_ref,'centro_ref',NULL,'calculo',calc,
   'documento',NULL);
  recibo:=jsonb_build_object('referencia',r1.referencia,'version',1,
   'registrado_en',to_char(r1.registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
   'repeticion',p_repeticion);
  SELECT * INTO regla_creacion FROM vec_dietas.recibo_regla_creacion_comision
   WHERE comision_ref=p_ref AND recibo_ref=r1.referencia;
  IF FOUND THEN recibo:=recibo||jsonb_build_object('regla_ref',regla_creacion.regla_ref,
   'regla_huella_sha256',regla_creacion.regla_huella_sha256); END IF;
 END IF;
 RETURN jsonb_build_object('resultado','concedido','comision',comision,'recibo',recibo);
END $$;

CREATE OR REPLACE FUNCTION vec_dietas.proyectar_revision_exacta_v2(p_ref text,p_version bigint)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog
 SET row_security=on SET timezone='UTC' AS $$
DECLARE b vec_dietas.borrador_comision%ROWTYPE; x vec_dietas.comision_revision%ROWTYPE;
        r vec_dietas.recibo_operacion_comision%ROWTYPE;
        numero vec_dietas.numero_documento_comision%ROWTYPE;
BEGIN
 IF current_user<>'vec_dietas_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_dietas_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_dietas_propietario','MEMBER') THEN
  RAISE EXCEPTION 'proyección Dietas inválida' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT b FROM vec_dietas.borrador_comision
  WHERE referencia=p_ref AND persona_ref=current_setting('vec.dietas.persona_ref',true);
 SELECT * INTO STRICT numero FROM vec_dietas.numero_documento_comision WHERE comision_ref=p_ref;
 SELECT * INTO STRICT x FROM vec_dietas.comision_revision
  WHERE comision_ref=p_ref AND version=p_version;
 SELECT * INTO STRICT r FROM vec_dietas.recibo_operacion_comision
  WHERE comision_ref=p_ref AND version=p_version;
 RETURN jsonb_build_object('resultado','concedido',
  'comision',jsonb_build_object('referencia',p_ref,'numero_documento',numero.numero_documento,
    'fecha_apertura',to_char(numero.fecha_apertura AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
    'estado',x.estado,'version',x.version,
    'fecha_inicio',x.fecha_inicio::text,'fecha_fin',x.fecha_fin::text,'motivo',x.motivo,
    'codigos_ruta',x.codigos_ruta,'relacion_ref',b.relacion_ref,
    'unidad_ref',b.unidad_ref,'centro_ref',x.centro_ref,
    'calculo',x.calculo,'documento',x.documento)||
      CASE WHEN x.vehiculo_propio IS NULL THEN '{}'::jsonb
           ELSE jsonb_build_object('vehiculo_propio',x.vehiculo_propio,'rutas',x.rutas) END,
  'recibo',jsonb_build_object('referencia',r.referencia,'version',r.version,
    'regla_ref',r.regla_ref,'regla_huella_sha256',r.regla_huella_sha256,
    'registrado_en',to_char(r.registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
    'repeticion',true));
END $$;

DROP FUNCTION vec_dietas.devolucion_vigente_comision_v1(text,bigint);
DO $post$
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_proc WHERE oid='vec_dietas.proyectar_comision_v2(text,boolean)'::regprocedure
        AND md5(prosrc)='3c60fac48aad3522d35293c4bd482f3a')
    OR NOT EXISTS (SELECT 1 FROM pg_proc WHERE oid='vec_dietas.proyectar_revision_exacta_v2(text,bigint)'::regprocedure
        AND md5(prosrc)='969163ddf9e7ea4942fbafc9eb01029d')
 THEN RAISE EXCEPTION 'Dietas 000010 DOWN: preimagen no restaurada' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
