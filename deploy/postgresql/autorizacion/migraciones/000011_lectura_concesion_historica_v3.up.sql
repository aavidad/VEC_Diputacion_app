\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
-- Misma barrera de 5/6 antes de la propia; ninguna modificación a sus objetos.
SELECT pg_advisory_xact_lock_shared(hashtextextended('vec_autorizacion:migracion:registro_contexto_actor_v3:000005',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion:migracion:lectura_concesion_historica_v3:000011',0));
SET LOCAL ROLE vec_autorizacion_propietario;
LOCK TABLE vec_autorizacion.decision_concedida_contexto_actor_v3 IN ACCESS SHARE MODE;
DO $precondiciones$
DECLARE t oid; f oid; firma text;
BEGIN
 IF getdatabaseencoding()<>'UTF8' OR NOT EXISTS (SELECT 1 FROM pg_namespace
  WHERE nspname='vec_autorizacion' AND nspowner=current_user::regrole)
 OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=current_user AND NOT rolcanlogin AND NOT rolsuper
  AND NOT rolinherit AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls)
 OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_autorizacion_registro' AND NOT rolcanlogin AND NOT rolsuper
  AND rolinherit AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls)
 OR NOT has_schema_privilege('vec_autorizacion_registro','vec_autorizacion','USAGE')
 OR EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
  WHERE n.nspname='vec_autorizacion' AND p.proname='leer_concesion_historica_contexto_actor_v3') THEN
  RAISE EXCEPTION 'auth11: instalación incompatible' USING ERRCODE='55000'; END IF;
 t:='vec_autorizacion.decision_concedida_contexto_actor_v3'::regclass;
 IF NOT EXISTS (SELECT 1 FROM pg_class WHERE oid=t AND relkind='r' AND relpersistence='p' AND NOT relispartition
  AND relowner=current_user::regrole AND relrowsecurity AND relforcerowsecurity
  AND obj_description(oid,'pg_class')='vec_autorizacion:registro-contexto-actor-v3:000005')
 OR EXISTS (SELECT 1 FROM pg_inherits WHERE inhrelid=t OR inhparent=t)
 OR EXISTS (SELECT 1 FROM pg_class c,LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) a
  WHERE c.oid=t AND (a.grantee<>c.relowner OR a.grantor<>c.relowner))
 -- AD3/roles_up concede sólo REFERENCES(decision_ref) para su FK.
 -- El GRANT de superusuario se registra con el propietario de la tabla
 -- como grantor. No requiere AD3 instalado ni crea/repara sus privilegios.
 OR EXISTS (SELECT 1 FROM pg_attribute at WHERE at.attrelid=t AND at.attnum>0
  AND at.attacl IS NOT NULL AND NOT (
   at.attname='decision_ref' AND NOT at.attisdropped
   AND EXISTS (SELECT 1 FROM pg_roles receptor
    WHERE receptor.rolname='vec_autorizacion_atestada_v3_propietario'
     AND at.attacl=ARRAY[makeaclitem(receptor.oid,current_user::regrole::oid,'REFERENCES',false)])))
 OR (SELECT count(*) FROM pg_policy WHERE polrelid=t)<>1
 OR NOT EXISTS (SELECT 1 FROM pg_policy WHERE polrelid=t AND polname='acceso_propietario_exacto' AND polcmd='*' AND polpermissive
  AND polroles=ARRAY[current_user::regrole::oid]
  AND pg_get_expr(polqual,polrelid)='(CURRENT_USER = ''vec_autorizacion_propietario''::name)'
  AND pg_get_expr(polwithcheck,polrelid)='(CURRENT_USER = ''vec_autorizacion_propietario''::name)') THEN
  RAISE EXCEPTION 'auth11: fuente propietaria incompatible' USING ERRCODE='55000'; END IF;
 FOREACH firma IN ARRAY ARRAY['vec_autorizacion.decision_contexto_actor_v3_valida(jsonb)',
  'vec_autorizacion.decision_contexto_actor_v3_canonica(jsonb)',
  'vec_autorizacion.motivo_contexto_actor_v3_canonico(jsonb)'] LOOP
  f:=to_regprocedure(firma);
  IF f IS NULL OR NOT EXISTS (SELECT 1 FROM pg_proc WHERE oid=f AND proowner=current_user::regrole
   AND prokind='f' AND provolatile='i' AND NOT prosecdef AND proconfig @> ARRAY['search_path=pg_catalog']) THEN
   RAISE EXCEPTION 'auth11: canon propietario requerido' USING ERRCODE='55000'; END IF;
 END LOOP;
END $precondiciones$;

-- Sólo identidad/canon y ventana ORIGINAL. No revalidación viva, escritura,
-- auditoría de efecto, renovación ni consumo. El llamador confirma su TX RO.
CREATE FUNCTION vec_autorizacion.leer_concesion_historica_contexto_actor_v3(
 p_decision_canonica bytea, p_motivo_canonico bytea,
 p_persona_version numeric, p_perfil_version numeric
) RETURNS TABLE(concedida boolean,codigo text,decision_huella_sha256 text,registrada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER ROWS 1
SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' SET statement_timeout='5s'
AS $lectura$
DECLARE d jsonb; m jsonb; rm jsonb; v jsonb; h text; estado text;
 original vec_autorizacion.decision_concedida_contexto_actor_v3%ROWTYPE;
BEGIN
 IF current_user<>'vec_autorizacion_propietario' THEN
  RAISE EXCEPTION 'auth11: lector no disponible' USING ERRCODE='42501'; END IF;
 IF current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'on' OR current_setting('TimeZone')<>'UTC' THEN
  RAISE EXCEPTION 'auth11: requiere SERIALIZABLE READ ONLY UTC' USING ERRCODE='25000'; END IF;
 PERFORM pg_advisory_xact_lock_shared(hashtextextended('vec_autorizacion:migracion:registro_contexto_actor_v3:000005',0));
 PERFORM pg_advisory_xact_lock_shared(hashtextextended('vec_autorizacion:migracion:lectura_concesion_historica_v3:000011',0));
 IF p_decision_canonica IS NULL OR octet_length(p_decision_canonica) NOT BETWEEN 1 AND 524288
 OR p_motivo_canonico IS NULL OR octet_length(p_motivo_canonico) NOT BETWEEN 1 AND 65536
 OR p_persona_version IS NULL OR scale(p_persona_version)<>0 OR p_persona_version NOT BETWEEN 1 AND 18446744073709551615::numeric
 OR p_perfil_version IS NULL OR scale(p_perfil_version)<>0 OR p_perfil_version NOT BETWEEN 1 AND 18446744073709551615::numeric THEN
  RAISE EXCEPTION 'auth11: entrada no válida' USING ERRCODE='22023'; END IF;
 d:=convert_from(p_decision_canonica,'UTF8')::jsonb;
 m:=convert_from(p_motivo_canonico,'UTF8')::jsonb;
 IF vec_autorizacion.decision_contexto_actor_v3_valida(d) IS NOT TRUE
 OR vec_autorizacion.decision_contexto_actor_v3_canonica(d) IS DISTINCT FROM p_decision_canonica
 OR d->'concedida' IS DISTINCT FROM 'true'::jsonb OR d->>'codigo' IS DISTINCT FROM 'concedida' THEN
  RAISE EXCEPTION 'auth11: decisión no válida' USING ERRCODE='22023'; END IF;
 -- Forma motivo auth6, en pasos para no invocar object_keys sobre escalares.
 IF jsonb_typeof(m) IS DISTINCT FROM 'object' THEN RAISE EXCEPTION 'auth11: motivo no válido' USING ERRCODE='22023'; END IF;
 IF (SELECT count(*) FROM jsonb_object_keys(m))<>2 OR NOT(m ?& ARRAY['esquema','referencia'])
 OR m->>'esquema' IS DISTINCT FROM 'vec.autorizacion.motivo.v2.referencia-opaca-catalogada'
 OR jsonb_typeof(m->'referencia') IS DISTINCT FROM 'object' THEN
  RAISE EXCEPTION 'auth11: motivo no válido' USING ERRCODE='22023'; END IF;
 rm:=m->'referencia';
 IF (SELECT count(*) FROM jsonb_object_keys(rm))<>4
 OR NOT(rm ?& ARRAY['catalogo_id','catalogo_version','catalogo_huella_sha256','entrada_clave'])
 OR jsonb_typeof(rm->'catalogo_id') IS DISTINCT FROM 'string'
 OR jsonb_typeof(rm->'catalogo_version') IS DISTINCT FROM 'number'
 OR jsonb_typeof(rm->'catalogo_huella_sha256') IS DISTINCT FROM 'string'
 OR jsonb_typeof(rm->'entrada_clave') IS DISTINCT FROM 'string'
 OR rm->>'catalogo_id' !~ '^[a-z][a-z0-9._-]{0,127}$'
 OR rm->>'catalogo_version' !~ '^[1-9][0-9]{0,9}$'
 OR (rm->>'catalogo_version')::numeric NOT BETWEEN 1 AND 2147483647
 OR rm->>'catalogo_huella_sha256' !~ '^[0-9a-f]{64}$'
 OR rm->>'catalogo_huella_sha256'=repeat('0',64)
 OR rm->>'entrada_clave' !~ '^motivo_[0-9a-f]{32}$'
 OR vec_autorizacion.motivo_contexto_actor_v3_canonico(m) IS DISTINCT FROM p_motivo_canonico
 OR encode(sha256(p_motivo_canonico),'hex') IS DISTINCT FROM d->>'motivo_huella_sha256' THEN
  RAISE EXCEPTION 'auth11: motivo no ligado' USING ERRCODE='22023'; END IF;
 h:=encode(sha256(p_decision_canonica),'hex'); v:=d->'vinculo_autenticacion_actor';
 -- Única tabla de negocio consultada; snapshot SERIALIZABLE y sin row locks de escritura.
 SELECT r.* INTO original FROM vec_autorizacion.decision_concedida_contexto_actor_v3 r
  WHERE r.decision_ref=d->>'decision_ref';
 IF NOT FOUND THEN RAISE EXCEPTION 'auth11: original no disponible' USING ERRCODE='55000'; END IF;
 IF original.huella_decision_sha256 IS DISTINCT FROM h OR original.decision_canonica IS DISTINCT FROM p_decision_canonica
 OR original.documento IS DISTINCT FROM d OR original.motivo_canonico IS DISTINCT FROM p_motivo_canonico
 OR original.persona_version IS DISTINCT FROM p_persona_version OR original.perfil_version IS DISTINCT FROM p_perfil_version
 OR original.motivo_catalogo_id IS DISTINCT FROM rm->>'catalogo_id'
 OR original.motivo_catalogo_version::numeric IS DISTINCT FROM (rm->>'catalogo_version')::numeric
 OR original.motivo_entrada_clave IS DISTINCT FROM rm->>'entrada_clave'
 OR original.registro_contexto_ref IS DISTINCT FROM v->>'registro_contexto_ref'
 OR original.contexto_actor_huella_sha256 IS DISTINCT FROM v->>'contexto_actor_huella_sha256'
 OR original.manifiesto_procedencia_huella_sha256 IS DISTINCT FROM v->>'manifiesto_procedencia_huella_sha256'
 OR original.asignacion_ref IS DISTINCT FROM d->>'asignacion_ref'
 OR original.version_rol_ref IS DISTINCT FROM d->>'version_rol_ref'
 OR original.control_vigencia_version_rol_revision IS DISTINCT FROM (d->>'control_vigencia_version_rol_revision')::numeric
 OR original.revision_catalogo_politicas IS DISTINCT FROM (d->>'revision_catalogo_politicas')::numeric
 OR original.emitida_en IS DISTINCT FROM (d->>'emitida_en')::timestamptz
 OR original.valida_hasta IS DISTINCT FROM (d->>'valida_hasta')::timestamptz
 OR original.registrada_en IS NULL OR NOT isfinite(original.registrada_en)
 OR original.registrada_en='0001-01-01 00:00:00+00'::timestamptz
 OR original.registrada_en<original.emitida_en OR original.registrada_en>=original.valida_hasta THEN
  RAISE EXCEPTION 'auth11: original no disponible' USING ERRCODE='55000'; END IF;
 -- Fecha obtenida de ESA fila, nunca now()/fecha del llamador. Una concesión
 -- caducada/retirada hoy sigue siendo historia, no autorización actual.
 RETURN QUERY SELECT (original.documento->>'concedida')::boolean,original.documento->>'codigo',original.huella_decision_sha256,original.registrada_en;
EXCEPTION WHEN OTHERS THEN
 estado:=SQLSTATE;
 IF estado LIKE '22%' THEN estado:='22023'; END IF;
 IF estado NOT IN ('22023','25000','42501','40001','40P01','55P03','55000') THEN estado:='55000'; END IF;
 RAISE EXCEPTION 'auth11: concesión histórica no disponible' USING ERRCODE=estado;
END $lectura$;
COMMENT ON FUNCTION vec_autorizacion.leer_concesion_historica_contexto_actor_v3(bytea,bytea,numeric,numeric)
 IS 'vec_autorizacion:lectura-concesion-historica-v3:000011';
-- Elimina grants heredados de defaults SOLO de esta función nueva.
DO $acl$
DECLARE f oid:='vec_autorizacion.leer_concesion_historica_contexto_actor_v3(bytea,bytea,numeric,numeric)'::regprocedure; a record;
BEGIN
 EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC',f::regprocedure);
 FOR a IN SELECT DISTINCT x.grantee FROM pg_proc p,LATERAL aclexplode(p.proacl) x
  WHERE p.oid=f AND x.grantee<>p.proowner LOOP
  EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %I',f::regprocedure,pg_get_userbyid(a.grantee));
 END LOOP;
END $acl$;
GRANT EXECUTE ON FUNCTION vec_autorizacion.leer_concesion_historica_contexto_actor_v3(bytea,bytea,numeric,numeric) TO vec_autorizacion_registro;
COMMIT;
