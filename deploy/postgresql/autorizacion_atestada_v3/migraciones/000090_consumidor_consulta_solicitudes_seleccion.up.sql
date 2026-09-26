\set ON_ERROR_STOP on
-- AD3-90. Consulta por RRHH de las solicitudes de participación presentadas
-- en una convocatoria de Selección («Convoca integrado», fase 1): listado
-- minimizado por convocatoria y ficha completa de una solicitud (datos
-- personales solo aquí, con auditoría de acceso en Selección). Un perfil
-- nominal nuevo 'consulta_solicitudes_seleccion' con dos operaciones, cada una
-- con su audiencia. Recursos: 'solicitudes-convocatoria:<sha256 de la
-- convocatoria>' para el listado y 'solicitud-seleccion:<solicitud>' para la
-- ficha. Único consumidor: Selección (vec_seleccion 000001), con la misma
-- guarda de sesión que AD3-89. Requiere AD3-89. Toma el cerrojo común.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000090',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));

DO $pre$
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_solicitud_propia_seleccion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR NOT has_schema_privilege('vec_seleccion_propietario','vec_autorizacion_atestada_v3','USAGE')
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_solicitudes_seleccion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_seleccion_propietario' AND NOT rolcanlogin AND NOT rolbypassrls AND NOT rolsuper)
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_seleccion_migrador' AND NOT rolcanlogin)
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_seleccion_ejecutor' AND NOT rolcanlogin AND NOT rolbypassrls AND NOT rolsuper)
 THEN RAISE EXCEPTION 'AD3-90: preimagen incompatible (AD3-89 requerida)' USING ERRCODE='55000'; END IF;
END $pre$;

DO $nucleo$
DECLARE
 f oid:='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 original text; nuevo text; actual text; meta jsonb; deps jsonb; acl aclitem[];
 propietario oid; config text[]; definidora boolean;
 marca text:=E'       )\n       OR c ->> ''suite'' <> ''VEC-AD-3-COSE-EDDSA-1''';
 excl text:=E'               p_perfil_mutacion IS DISTINCT FROM ''bolsa_llamamiento''\n';
 excl_nuevo text:=excl||E'               AND p_perfil_mutacion IS DISTINCT FROM ''consulta_solicitudes_seleccion''\n';
 ancla text:=E'           OR (\n               p_perfil_mutacion IS NOT DISTINCT FROM ''alta_personal_ejercicio''\n               AND pg_catalog.pg_has_role(\n';
 guarda text:=$g$           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_solicitudes_seleccion'
               AND pg_catalog.pg_has_role(session_user, 'vec_seleccion_ejecutor', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(session_user, 'vec_seleccion_propietario', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(session_user, 'vec_seleccion_migrador', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(session_user, 'vec_contratacion_temporal_propietario', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(session_user, 'vec_contratacion_temporal_migrador', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(session_user, 'vec_autorizacion_atestada_v3_propietario', 'MEMBER')
           )
$g$;
 extension text:=$x$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_solicitudes_seleccion'
 AND ((c->>'operacion' IS NOT DISTINCT FROM 'seleccion.solicitudes.consultar'
       AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_seleccion.solicitudes.consultar.v1')
   OR (c->>'operacion' IS NOT DISTINCT FROM 'seleccion.solicitudes.consultar_detalle'
       AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_seleccion.solicitudes.consultar_detalle.v1'))
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'solicitudes_seleccion'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'seleccion'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'consulta_solicitudes_seleccion'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
$x$;
BEGIN
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc',p.proacl,p.proowner,p.proconfig,p.prosecdef
 INTO STRICT original,meta,acl,propietario,config,definidora FROM pg_proc p WHERE p.oid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
 INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 -- Cada marca aparece exactamente una vez; el núcleo lleva AD3-89 y aún no
 -- esta extensión.
 IF propietario<>'vec_autorizacion_atestada_v3_propietario'::regrole OR NOT definidora
    OR config IS DISTINCT FROM ARRAY['search_path=pg_catalog','lock_timeout=2s']
    OR length(original)-length(replace(original,marca,''))<>length(marca)
    OR length(original)-length(replace(original,excl,''))<>length(excl)
    OR length(original)-length(replace(original,ancla,''))<>length(ancla)
    OR strpos(original,'consulta_solicitudes_seleccion')<>0
    OR strpos(original,'solicitud_propia_seleccion')=0
 THEN RAISE EXCEPTION 'AD3-90: núcleo incompatible' USING ERRCODE='55000'; END IF;
 nuevo:=replace(original,excl,excl_nuevo);
 nuevo:=replace(nuevo,ancla,guarda||ancla);
 nuevo:=replace(nuevo,marca,extension||marca);
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo
    OR replace(replace(replace(actual,extension||marca,marca),guarda||ancla,ancla),excl_nuevo,excl) IS DISTINCT FROM original
    OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
    OR (SELECT proacl FROM pg_proc WHERE oid=f) IS DISTINCT FROM acl
    OR (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM propietario
    OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM config
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS DISTINCT FROM definidora
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb) FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
 THEN RAISE EXCEPTION 'AD3-90: núcleo alterado fuera del contrato' USING ERRCODE='55000'; END IF;
END $nucleo$;

LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencias$
DECLARE d text; a text;
 nuevas text[]:=ARRAY['vec_seleccion.solicitudes.consultar.v1',
                      'vec_seleccion.solicitudes.consultar_detalle.v1'];
BEGIN
 SELECT regexp_replace(pg_get_constraintdef(c.oid,true),'\s+',' ','g') INTO STRICT d
 FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
 AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 IF strpos(d,'CHECK (audiencia_consumo = ANY (ARRAY[')<>1 OR right(d,3)<>']))' OR strpos(d,'vec_seleccion.solicitudes_propias.presentar.v1')=0 OR strpos(d,'vec_seleccion.solicitudes.')<>0
 THEN RAISE EXCEPTION 'AD3-90: audiencias previas incompatibles' USING ERRCODE='55000'; END IF;
 FOREACH a IN ARRAY nuevas LOOP
  d:=left(d,length(d)-3)||', '||quote_literal(a)||'::text]))';
 END LOOP;
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version
  DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||d;
END $audiencias$;

-- Fachada única. Además de la guarda del núcleo exige el recurso de la
-- operación (listado por convocatoria o ficha de una solicitud), sin campos
-- ni obligaciones, y un consumo nuevo.
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_solicitudes_seleccion_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; r record;
BEGIN
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'AD3-90: material de la consulta de solicitudes inválido' USING ERRCODE='22023'; END;
 IF coalesce(c->>'operacion','') NOT IN ('seleccion.solicitudes.consultar','seleccion.solicitudes.consultar_detalle')
    OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_seleccion.solicitudes.'||substr(c->>'operacion',length('seleccion.solicitudes.')+1)||'.v1'
    OR d->>'accion' IS DISTINCT FROM c->>'operacion'
    OR d->>'modulo_id' IS DISTINCT FROM 'seleccion'
    OR d->>'finalidad' IS DISTINCT FROM 'consulta_solicitudes_seleccion'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'solicitudes_seleccion'
    OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
    OR (c->>'operacion'='seleccion.solicitudes.consultar' AND coalesce(d->>'recurso_ref','') !~ '^solicitudes-convocatoria:[0-9a-f]{64}$')
    OR (c->>'operacion'='seleccion.solicitudes.consultar_detalle' AND coalesce(d->>'recurso_ref','') !~ '^solicitud-seleccion:sol_[A-Za-z0-9_-]{22,64}$')
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
    OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'AD3-90: material de la consulta de solicitudes rechazado' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT r FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  'consulta_solicitudes_seleccion',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF r.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'AD3-90: la consulta de solicitudes requiere consumo nuevo' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT r.decision_ref,r.efecto_ref,r.huella_efecto_sha256,r.consumo_huella_sha256,r.auditoria_ref,r.consumida_en,true;
END $f$;

DO $acl$
DECLARE f regprocedure:='vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_solicitudes_seleccion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 permitido oid:='vec_seleccion_propietario'::regrole::oid; x record;
BEGIN
 FOR x IN SELECT DISTINCT a.grantee FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
  WHERE p.oid=f AND a.grantee<>p.proowner LOOP
  EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %s',f::text,CASE WHEN x.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(x.grantee)) END);
 END LOOP;
 EXECUTE format('GRANT EXECUTE ON FUNCTION %s TO vec_seleccion_propietario',f::text);
 IF (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario'::regrole
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS NOT TRUE
    OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM ARRAY['search_path=pg_catalog','lock_timeout=2s']
    OR NOT has_schema_privilege('vec_seleccion_propietario','vec_autorizacion_atestada_v3','USAGE')
 THEN RAISE EXCEPTION 'AD3-90: propietario o entorno de fachada incompatible' USING ERRCODE='55000'; END IF;
 FOR x IN SELECT a.grantee,a.privilege_type,a.is_grantable,p.proowner
  FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f LOOP
  IF (x.grantee<>x.proowner AND x.grantee IS DISTINCT FROM permitido) OR x.privilege_type<>'EXECUTE'
    OR (x.grantee=permitido AND x.is_grantable)
  THEN RAISE EXCEPTION 'AD3-90: ACL de fachada abierta' USING ERRCODE='55000'; END IF;
 END LOOP;
END $acl$;
COMMIT;
