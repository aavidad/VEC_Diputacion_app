\set ON_ERROR_STOP on
-- D6: corregir y reenviar un documento devuelto.
-- 000006 ya admite editar y enviar desde «devuelta» con recibo, idempotencia,
-- versión esperada e historia de solo adición; el reenvío vuelve siempre a
-- la revisión del administrativo (pregunta 51 de dudas.md). Esta migración:
-- 1) proyecta a la persona titular la devolución vigente (etapa, motivo,
--    versión y fecha) mientras el documento está devuelto o en corrección;
--    se toma de la historia, sin copiarla ni reescribirla;
-- 2) impide eliminar un documento que ya entró en el circuito: la versión
--    devuelta y su historia se conservan y el documento solo se corrige;
-- 3) muestra a quien revisa un reenvío la devolución anterior (etapa,
--    motivo, versión y fecha), sin la identidad de quien la hizo. El motivo
--    lo lee también la persona titular.
-- Las funciones sustituidas conservan firma, dueño, ACL, SECURITY DEFINER y
-- entorno (proconfig) exactos de 000006 y 000008.
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
    OR to_regprocedure('vec_dietas.devolucion_anterior_comision_v1(text,bigint)') IS NOT NULL
    OR to_regprocedure('vec_dietas.impedir_eliminar_tras_circuito_v1()') IS NOT NULL
    OR EXISTS (SELECT 1 FROM pg_trigger WHERE tgrelid='vec_dietas.comision_revision'::regclass
        AND tgname='impedir_eliminar_tras_circuito')
    -- Preimagen exacta de las tres funciones sustituidas: cuerpo, dueño,
    -- SECURITY DEFINER, ACL y entorno.
    OR (SELECT count(*) FROM pg_proc p JOIN pg_roles r ON r.oid=p.proowner
        JOIN (VALUES
         ('vec_dietas.proyectar_comision_v2(text,boolean)','3c60fac48aad3522d35293c4bd482f3a',
          '{vec_dietas_propietario=X/vec_dietas_propietario}','{search_path=pg_catalog,row_security=on,TimeZone=UTC}'),
         ('vec_dietas.proyectar_revision_exacta_v2(text,bigint)','969163ddf9e7ea4942fbafc9eb01029d',
          '{vec_dietas_propietario=X/vec_dietas_propietario}','{search_path=pg_catalog,row_security=on,TimeZone=UTC}'),
         ('vec_dietas.consultar_documento_circuito_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','b3bac4e3a9a9544ad64f24961255eb02',
          '{vec_dietas_propietario=X/vec_dietas_propietario,vec_dietas_ejecutor=X/vec_dietas_propietario}','{search_path=pg_catalog,row_security=on,TimeZone=UTC,lock_timeout=2s}')
        ) e(firma,huella,acl,entorno) ON p.oid=to_regprocedure(e.firma)
        WHERE r.rolname='vec_dietas_propietario' AND p.prosecdef AND md5(p.prosrc)=e.huella
          AND p.proacl::text=e.acl AND p.proconfig::text=e.entorno)<>3
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

-- Última devolución anterior a una versión, aunque ya se haya reenviado: la
-- lee quien revisa el reenvío. Sin actor: nunca identifica al revisor previo.
CREATE FUNCTION vec_dietas.devolucion_anterior_comision_v1(p_ref text,p_version bigint) RETURNS jsonb
LANGUAGE sql STABLE SET search_path=pg_catalog AS $funcion$
 SELECT jsonb_build_object('etapa',substr(h.tipo,10),'motivo',h.motivo,'version',h.version,
  'devuelta_en',to_char(h.registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))
 FROM vec_dietas.historia_operacion_comision h
 WHERE h.comision_ref=p_ref AND h.version<p_version AND h.estado_nuevo='devuelta'
   AND h.tipo IN ('circuito_revision','circuito_autorizacion','circuito_liquidacion','circuito_fiscalizacion')
   AND h.motivo IS NOT NULL
 ORDER BY h.version DESC LIMIT 1
$funcion$;
ALTER FUNCTION vec_dietas.devolucion_anterior_comision_v1(text,bigint) OWNER TO vec_dietas_propietario;
REVOKE ALL ON FUNCTION vec_dietas.devolucion_anterior_comision_v1(text,bigint) FROM PUBLIC;

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

-- Documento del circuito (000008) con la devolución anterior de un reenvío.
CREATE OR REPLACE FUNCTION vec_dietas.consultar_documento_circuito_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,
 p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $funcion$
#variable_conflict use_variable
DECLARE m jsonb; i jsonb; cap jsonb; d jsonb; ctx jsonb; v record;
 q vec_dietas.cola_circuito_comision%ROWTYPE;
 b vec_dietas.borrador_comision%ROWTYPE; r vec_dietas.comision_revision%ROWTYPE;
 numero vec_dietas.numero_documento_comision%ROWTYPE;
 etapa text; estado_esperado text; comision jsonb; devolucion jsonb;
BEGIN
 IF current_user<>'vec_dietas_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_dietas_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_dietas_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_dietas_migrador','MEMBER')
 THEN RAISE EXCEPTION 'ejecutor Dietas inválido' USING ERRCODE='42501'; END IF;
 IF current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR current_setting('TimeZone')<>'UTC'
 THEN RAISE EXCEPTION 'transacción Dietas incompatible' USING ERRCODE='25000'; END IF;
 BEGIN
  m:=p_material::jsonb; cap:=convert_from(p_capacidad,'UTF8')::jsonb;
  d:=convert_from(p_decision,'UTF8')::jsonb; ctx:=convert_from(p_contexto,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'material Dietas inválido' USING ERRCODE='22023'; END;
 i:=m->'identidad'; etapa:=m->>'etapa';
 estado_esperado:=CASE etapa WHEN 'revision' THEN 'enviado_pendiente_revision'
  WHEN 'autorizacion' THEN 'pendiente_autorizacion'
  WHEN 'liquidacion' THEN 'pendiente_liquidacion'
  WHEN 'fiscalizacion' THEN 'pendiente_fiscalizacion' END;
 IF m->>'operacion' IS DISTINCT FROM 'consultar_documento' OR estado_esperado IS NULL
    OR vec_dietas.cotejar_efecto_circuito_v2(p_material,p_capacidad,p_decision,p_contexto) IS NOT TRUE
    OR p_persona_version IS DISTINCT FROM (i->>'persona_version')::numeric
    OR p_perfil_version IS DISTINCT FROM (i->>'perfil_version')::numeric
    OR p_persona_version IS DISTINCT FROM (ctx->>'persona_version')::numeric
    OR p_perfil_version IS DISTINCT FROM (ctx->>'perfil_version')::numeric
 THEN RAISE EXCEPTION 'lectura de revisión Dietas incompatible' USING ERRCODE='PD003'; END IF;
 SELECT * INTO STRICT v FROM vec_autorizacion_atestada_v3.registrar_y_consumir_dietas_revisor_documento_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 IF v.consumo_nuevo IS NOT TRUE OR v.decision_ref IS DISTINCT FROM d->>'decision_ref'
    OR v.efecto_ref IS DISTINCT FROM m->>'recurso_ref'
    OR v.huella_efecto_sha256 IS DISTINCT FROM cap->>'huella_efecto_sha256'
 THEN RAISE EXCEPTION 'consumo AD3 de revisión incompatible' USING ERRCODE='PD003'; END IF;
 PERFORM set_config('vec.dietas.circuito_etapa',etapa,true);
 PERFORM set_config('vec.dietas.circuito_unidad_ref',m->>'unidad_ref',true);
 SELECT * INTO q FROM vec_dietas.cola_circuito_comision cola
  WHERE cola.comision_ref=m->>'recurso_ref' AND cola.etapa=etapa
    AND cola.unidad_ref=m->>'unidad_ref'
  ORDER BY cola.version DESC LIMIT 1;
 IF NOT FOUND THEN RETURN jsonb_build_object('resultado','no_encontrado'); END IF;
 PERFORM set_config('vec.dietas.persona_ref',q.persona_ref,true);
 SELECT * INTO b FROM vec_dietas.borrador_comision WHERE referencia=q.comision_ref;
 SELECT * INTO r FROM vec_dietas.comision_revision
  WHERE comision_ref=q.comision_ref ORDER BY version DESC LIMIT 1;
 IF NOT FOUND OR b.referencia IS NULL OR b.unidad_ref IS DISTINCT FROM q.unidad_ref
    OR r.version IS DISTINCT FROM q.version OR r.estado IS DISTINCT FROM estado_esperado
    OR r.asignacion_ref IS DISTINCT FROM q.asignacion_ref
    OR r.asignacion_version IS DISTINCT FROM q.asignacion_version
    OR i->>'persona_ref'=q.persona_ref
    OR (q.destinatario_persona_ref IS NOT NULL
      AND q.destinatario_persona_ref IS DISTINCT FROM i->>'persona_ref')
    OR (etapa='revision' AND r.administrativo_persona_ref IS DISTINCT FROM i->>'persona_ref')
    OR (etapa='autorizacion' AND r.responsable_persona_ref IS DISTINCT FROM i->>'persona_ref')
    OR EXISTS (SELECT 1 FROM vec_dietas.historia_operacion_comision h
      WHERE h.comision_ref=q.comision_ref AND h.actor_ref=i->>'actor_ref'
        AND h.tipo IS DISTINCT FROM ('circuito_'||etapa))
 THEN RETURN jsonb_build_object('resultado','no_encontrado'); END IF;
 BEGIN
  IF vec_personal.revalidar_asignacion_dietas_v1(b.relacion_ref,q.persona_ref,
    b.unidad_ref,q.asignacion_ref,q.asignacion_version,r.grupo_dieta,r.centro_ref,
    r.administrativo_persona_ref,r.responsable_persona_ref,current_date) IS NOT TRUE
  THEN RETURN jsonb_build_object('resultado','no_encontrado'); END IF;
 EXCEPTION WHEN SQLSTATE 'P7201' THEN
  RETURN jsonb_build_object('resultado','no_encontrado');
 END;
 SELECT * INTO STRICT numero FROM vec_dietas.numero_documento_comision WHERE comision_ref=q.comision_ref;
 comision:=jsonb_build_object('referencia',q.comision_ref,'numero_documento',numero.numero_documento,
  'fecha_apertura',to_char(numero.fecha_apertura AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'estado',r.estado,'version',r.version,
  'fecha_inicio',r.fecha_inicio::text,'fecha_fin',r.fecha_fin::text,
  'hora_inicio',r.hora_inicio,'hora_fin',r.hora_fin,'motivo',r.motivo,
  'codigos_ruta',r.codigos_ruta,'calculo',r.calculo,'documento',r.documento);
 IF r.vehiculo_propio IS NOT NULL THEN
  comision:=comision||jsonb_build_object('vehiculo_propio',r.vehiculo_propio,'rutas',r.rutas);
 END IF;
 -- Reenvío: quien revisa ve la última devolución anterior (etapa, motivo,
 -- versión y fecha), nunca quién la hizo.
 devolucion:=vec_dietas.devolucion_anterior_comision_v1(q.comision_ref,r.version);
 IF devolucion IS NOT NULL THEN
  comision:=comision||jsonb_build_object('devolucion',devolucion);
 END IF;
 RETURN jsonb_build_object('resultado','concedido','comision',comision);
END $funcion$;

DO $post$
BEGIN
 IF (SELECT count(*) FROM pg_proc p JOIN pg_roles r ON r.oid=p.proowner
     JOIN (VALUES
      ('vec_dietas.proyectar_comision_v2(text,boolean)',true,
       '{vec_dietas_propietario=X/vec_dietas_propietario}','{search_path=pg_catalog,row_security=on,TimeZone=UTC}'),
      ('vec_dietas.proyectar_revision_exacta_v2(text,bigint)',true,
       '{vec_dietas_propietario=X/vec_dietas_propietario}','{search_path=pg_catalog,row_security=on,TimeZone=UTC}'),
      ('vec_dietas.consultar_documento_circuito_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',true,
       '{vec_dietas_propietario=X/vec_dietas_propietario,vec_dietas_ejecutor=X/vec_dietas_propietario}','{search_path=pg_catalog,row_security=on,TimeZone=UTC,lock_timeout=2s}'),
      ('vec_dietas.devolucion_vigente_comision_v1(text,bigint)',false,
       '{vec_dietas_propietario=X/vec_dietas_propietario}','{search_path=pg_catalog}'),
      ('vec_dietas.devolucion_anterior_comision_v1(text,bigint)',false,
       '{vec_dietas_propietario=X/vec_dietas_propietario}','{search_path=pg_catalog}'),
      ('vec_dietas.impedir_eliminar_tras_circuito_v1()',false,
       '{vec_dietas_propietario=X/vec_dietas_propietario}','{search_path=pg_catalog}')
     ) e(firma,definidora,acl,entorno) ON p.oid=to_regprocedure(e.firma)
     WHERE r.rolname='vec_dietas_propietario' AND p.prosecdef=e.definidora
       AND p.proacl::text=e.acl AND p.proconfig::text=e.entorno)<>6
    OR NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgrelid='vec_dietas.comision_revision'::regclass
       AND tgname='impedir_eliminar_tras_circuito' AND tgenabled='O')
 THEN RAISE EXCEPTION 'Dietas 000010: postcondición incumplida (dueño, ACL, entorno o SECURITY DEFINER)' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
