\set ON_ERROR_STOP on
-- D6: corregir y reenviar un documento devuelto.
-- 000006 ya admite editar y enviar desde «devuelta» con recibo, idempotencia,
-- versión esperada e historia de solo adición; el reenvío vuelve siempre a
-- la revisión del administrativo (pregunta 49 de dudas.md). Esta migración:
-- 1) proyecta a la persona titular la devolución vigente (etapa, motivo,
--    versión y fecha) mientras el documento está devuelto o en corrección;
--    se toma de la historia, sin copiarla ni reescribirla;
-- 2) impide eliminar un documento que ya entró en el circuito: la versión
--    devuelta y su historia se conservan y el documento solo se corrige.
-- Las proyecciones conservan firma, dueño, ACL y SECURITY DEFINER de 000006.
BEGIN;
SET LOCAL ROLE vec_dietas_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_dietas:migracion:000010:devolucion:v1',0));
DO $pre$
BEGIN
 IF current_user<>'vec_dietas_propietario'
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_dietas_ejecutor')
    OR to_regclass('vec_dietas.historia_operacion_comision') IS NULL
    OR to_regclass('vec_dietas.tipo_otro_gasto_provisional') IS NULL
    OR to_regprocedure('vec_dietas.devolucion_vigente_comision_v1(text,bigint)') IS NOT NULL
    OR to_regprocedure('vec_dietas.impedir_eliminar_tras_circuito_v1()') IS NOT NULL
    OR EXISTS (SELECT 1 FROM pg_trigger WHERE tgrelid='vec_dietas.comision_revision'::regclass
        AND tgname='impedir_eliminar_tras_circuito')
    OR NOT EXISTS (SELECT 1 FROM pg_proc p JOIN pg_roles r ON r.oid=p.proowner
        WHERE p.oid=to_regprocedure('vec_dietas.proyectar_comision_v2(text,boolean)')
          AND r.rolname='vec_dietas_propietario' AND p.prosecdef
          AND md5(p.prosrc)='3c60fac48aad3522d35293c4bd482f3a')
    OR NOT EXISTS (SELECT 1 FROM pg_proc p JOIN pg_roles r ON r.oid=p.proowner
        WHERE p.oid=to_regprocedure('vec_dietas.proyectar_revision_exacta_v2(text,bigint)')
          AND r.rolname='vec_dietas_propietario' AND p.prosecdef
          AND md5(p.prosrc)='969163ddf9e7ea4942fbafc9eb01029d')
    OR has_function_privilege('vec_dietas_ejecutor','vec_dietas.proyectar_comision_v2(text,boolean)','EXECUTE')
    OR has_function_privilege('vec_dietas_ejecutor','vec_dietas.proyectar_revision_exacta_v2(text,bigint)','EXECUTE')
 THEN RAISE EXCEPTION 'Dietas 000010: falta 000009, preimagen alterada o ya instalada' USING ERRCODE='55000'; END IF;
END $pre$;

-- Devolución vigente en una versión: la última devolución del circuito sin
-- reenvío posterior hasta esa versión. La lee el propietario dentro de las
-- proyecciones, con la RLS de la persona titular.
CREATE FUNCTION vec_dietas.devolucion_vigente_comision_v1(p_ref text,p_hasta bigint) RETURNS jsonb
LANGUAGE sql STABLE SET search_path=pg_catalog AS $funcion$
 SELECT jsonb_build_object('etapa',substr(h.tipo,10),'motivo',h.motivo,'version',h.version,
  'devuelta_en',to_char(h.registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))
 FROM vec_dietas.historia_operacion_comision h
 WHERE h.comision_ref=p_ref AND h.version<=p_hasta AND h.estado_nuevo='devuelta'
   AND h.tipo IN ('circuito_revision','circuito_autorizacion','circuito_liquidacion','circuito_fiscalizacion')
   AND h.motivo IS NOT NULL
   AND NOT EXISTS (SELECT 1 FROM vec_dietas.historia_operacion_comision e
     WHERE e.comision_ref=p_ref AND e.tipo='enviar' AND e.version>h.version AND e.version<=p_hasta)
 ORDER BY h.version DESC LIMIT 1
$funcion$;
ALTER FUNCTION vec_dietas.devolucion_vigente_comision_v1(text,bigint) OWNER TO vec_dietas_propietario;
REVOKE ALL ON FUNCTION vec_dietas.devolucion_vigente_comision_v1(text,bigint) FROM PUBLIC;

-- Un documento que ya entró en el circuito no se elimina: se corrige y se
-- reenvía. Falla cerrado si la comisión no es visible para quien inserta.
CREATE FUNCTION vec_dietas.impedir_eliminar_tras_circuito_v1() RETURNS trigger
LANGUAGE plpgsql VOLATILE SET search_path=pg_catalog AS $funcion$
BEGIN
 IF NEW.estado IS DISTINCT FROM 'eliminado' THEN RETURN NEW; END IF;
 IF NOT EXISTS (SELECT 1 FROM vec_dietas.borrador_comision b WHERE b.referencia=NEW.comision_ref) THEN
  RAISE EXCEPTION 'comisión Dietas ausente' USING ERRCODE='PD004'; END IF;
 IF EXISTS (SELECT 1 FROM vec_dietas.comision_revision r
            WHERE r.comision_ref=NEW.comision_ref AND r.estado<>'borrador') THEN
  RAISE EXCEPTION 'versión o estado Dietas incompatible' USING ERRCODE='PD005'; END IF;
 RETURN NEW;
END $funcion$;
ALTER FUNCTION vec_dietas.impedir_eliminar_tras_circuito_v1() OWNER TO vec_dietas_propietario;
REVOKE ALL ON FUNCTION vec_dietas.impedir_eliminar_tras_circuito_v1() FROM PUBLIC;
CREATE TRIGGER impedir_eliminar_tras_circuito BEFORE INSERT ON vec_dietas.comision_revision
 FOR EACH ROW WHEN (NEW.estado='eliminado')
 EXECUTE FUNCTION vec_dietas.impedir_eliminar_tras_circuito_v1();

CREATE OR REPLACE FUNCTION vec_dietas.proyectar_comision_v2(p_ref text,p_repeticion boolean)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on SET timezone='UTC' AS $$
DECLARE b vec_dietas.borrador_comision%ROWTYPE; x vec_dietas.comision_revision%ROWTYPE;
        r vec_dietas.recibo_operacion_comision%ROWTYPE; r1 vec_dietas.recibo_borrador_comision%ROWTYPE;
        regla_creacion vec_dietas.recibo_regla_creacion_comision%ROWTYPE;
        numero vec_dietas.numero_documento_comision%ROWTYPE;
        calc jsonb; comision jsonb; recibo jsonb; devolucion jsonb;
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
  IF x.estado IN ('devuelta','borrador') THEN
   devolucion:=vec_dietas.devolucion_vigente_comision_v1(p_ref,x.version);
   IF devolucion IS NOT NULL THEN
    comision:=comision||jsonb_build_object('devolucion',devolucion);
   END IF;
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
        devolucion jsonb;
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
 IF x.estado IN ('devuelta','borrador') THEN
  devolucion:=vec_dietas.devolucion_vigente_comision_v1(p_ref,x.version);
 END IF;
 RETURN jsonb_build_object('resultado','concedido',
  'comision',jsonb_build_object('referencia',p_ref,'numero_documento',numero.numero_documento,
    'fecha_apertura',to_char(numero.fecha_apertura AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
    'estado',x.estado,'version',x.version,
    'fecha_inicio',x.fecha_inicio::text,'fecha_fin',x.fecha_fin::text,'motivo',x.motivo,
    'codigos_ruta',x.codigos_ruta,'relacion_ref',b.relacion_ref,
    'unidad_ref',b.unidad_ref,'centro_ref',x.centro_ref,
    'calculo',x.calculo,'documento',x.documento)||
      CASE WHEN x.vehiculo_propio IS NULL THEN '{}'::jsonb
           ELSE jsonb_build_object('vehiculo_propio',x.vehiculo_propio,'rutas',x.rutas) END||
      CASE WHEN devolucion IS NULL THEN '{}'::jsonb
           ELSE jsonb_build_object('devolucion',devolucion) END,
  'recibo',jsonb_build_object('referencia',r.referencia,'version',r.version,
    'regla_ref',r.regla_ref,'regla_huella_sha256',r.regla_huella_sha256,
    'registrado_en',to_char(r.registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
    'repeticion',true));
END $$;

DO $post$
DECLARE firma text;
BEGIN
 FOREACH firma IN ARRAY ARRAY['vec_dietas.proyectar_comision_v2(text,boolean)',
   'vec_dietas.proyectar_revision_exacta_v2(text,bigint)',
   'vec_dietas.devolucion_vigente_comision_v1(text,bigint)',
   'vec_dietas.impedir_eliminar_tras_circuito_v1()'] LOOP
  IF (SELECT r.rolname FROM pg_proc p JOIN pg_roles r ON r.oid=p.proowner
      WHERE p.oid=firma::regprocedure)<>'vec_dietas_propietario'
     OR has_function_privilege('vec_dietas_ejecutor',firma,'EXECUTE')
     OR has_function_privilege('public',firma,'EXECUTE') THEN
   RAISE EXCEPTION 'Dietas 000010: postcondición incumplida en %',firma USING ERRCODE='55000'; END IF;
 END LOOP;
 IF NOT (SELECT bool_and(prosecdef) FROM pg_proc WHERE oid IN
     ('vec_dietas.proyectar_comision_v2(text,boolean)'::regprocedure,
      'vec_dietas.proyectar_revision_exacta_v2(text,bigint)'::regprocedure))
    OR (SELECT bool_or(prosecdef) FROM pg_proc WHERE oid IN
     ('vec_dietas.devolucion_vigente_comision_v1(text,bigint)'::regprocedure,
      'vec_dietas.impedir_eliminar_tras_circuito_v1()'::regprocedure))
 THEN RAISE EXCEPTION 'Dietas 000010: SECURITY DEFINER alterado' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
