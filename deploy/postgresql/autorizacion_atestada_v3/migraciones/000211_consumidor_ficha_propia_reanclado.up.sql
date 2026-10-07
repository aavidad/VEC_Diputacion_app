\set ON_ERROR_STOP on
-- Reanclaje de AD74 para la ficha propia. AD74 no se aplica: su preimagen del
-- núcleo y el formato del CHECK preceden a las extensiones posteriores.
-- Conserva la acción, audiencia, perfil y firma que consume Personal22.
-- Orden: núcleo y audiencias medidos -> AD211 -> Personal22. Una sola vez.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000211',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;

DO $pre$
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR current_user<>'vec_autorizacion_atestada_v3_propietario'
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_registro_empleado_b2_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_catalogo_registro_empleado_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_empleados_registro_b2_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_ficha_propia_empleado_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.capacidad_cruda_prevalida(bytea)') IS NULL
 THEN RAISE EXCEPTION 'AD211: PARO clave=preimagen esperado=PG18_B2_sin_ficha actual=incompatible' USING ERRCODE='55000'; END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_personal_propietario'
   AND NOT(rolcanlogin OR rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls))
    OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_personal_ejecutor'
   AND NOT(rolcanlogin OR rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls))
 THEN RAISE EXCEPTION 'AD211: PARO clave=roles esperado=Personal_privado actual=incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

-- Comprobar también el CHECK antes de recrear el núcleo. El bloqueo evita que
-- una extensión concurrente cambie esta preimagen entre lectura y ALTER.
LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencias_pre$
DECLARE anterior text; h text;
BEGIN
 SELECT pg_get_constraintdef(c.oid,false) INTO STRICT anterior FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
  AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 h:=encode(sha256(convert_to(anterior,'UTF8')),'hex');
 IF h IS DISTINCT FROM '8496cf66ff2f32b4afb6954708dab3a0e720f53a12aad564ed717828eb4284d4'
    OR left(anterior,7)<>'CHECK (' OR right(anterior,1)<>')'
    OR strpos(anterior,'vec_personal.registro_empleado.ficha_propia.v1')<>0
 THEN RAISE EXCEPTION 'AD211: PARO clave=CHECK_audiencias_SHA esperado=% actual=%','8496cf66ff2f32b4afb6954708dab3a0e720f53a12aad564ed717828eb4284d4',h USING ERRCODE='55000'; END IF;
END $audiencias_pre$;

DO $nucleo$
DECLARE
 f oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 original text; fuente text; nuevo text; actual text; h text;
 meta jsonb; deps jsonb; compartidas jsonb;
 marca text:=E'       )\n       OR c ->> ''suite'' <> ''VEC-AD-3-COSE-EDDSA-1''';
 extension text:=$x$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'registro_empleado_b2'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'personal'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb
 AND c->>'operacion' IS NOT DISTINCT FROM 'personal.registro_empleado.ficha_propia.consultar'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.registro_empleado.ficha_propia.v1'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'ficha_propia_empleado'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_ficha_propia'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["corte","evidencia","relaciones","servicios"]'::jsonb)
$x$;
BEGIN
 IF f IS NULL THEN RAISE EXCEPTION 'AD211: PARO clave=nucleo esperado=presente actual=ausente' USING ERRCODE='55000'; END IF;
 SELECT pg_get_functiondef(f),p.prosrc,to_jsonb(p)-'prosrc'
 INTO STRICT original,fuente,meta FROM pg_proc p WHERE p.oid=f;
 h:=encode(sha256(convert_to(original,'UTF8')),'hex');
 IF h IS DISTINCT FROM '092367a3c6be54e163eceb26d58a86442f83addc044e0cae2e85e572c7e56faf'
 THEN RAISE EXCEPTION 'AD211: PARO clave=nucleo_def_SHA esperado=% actual=%','092367a3c6be54e163eceb26d58a86442f83addc044e0cae2e85e572c7e56faf',h USING ERRCODE='55000'; END IF;
 h:=encode(sha256(convert_to(fuente,'UTF8')),'hex');
 IF h IS DISTINCT FROM '559555ec535ad40cc3aad6361286899c28aede91ff73b2901eb30970d95ac986'
 THEN RAISE EXCEPTION 'AD211: PARO clave=nucleo_src_SHA esperado=% actual=%','559555ec535ad40cc3aad6361286899c28aede91ff73b2901eb30970d95ac986',h USING ERRCODE='55000'; END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f
   AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
   AND p.prosecdef AND p.provolatile='v' AND p.proparallel='u'
   AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s'])
    OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f)<>1
    OR NOT EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
      WHERE p.oid=f AND a.grantee=p.proowner AND a.grantor=p.proowner
       AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
    OR length(original)-length(replace(original,marca,''))<>length(marca)
    OR strpos(original,'personal.registro_empleado.ficha_propia.consultar')<>0
    OR strpos(original,'registro_empleado_b2')=0
 THEN RAISE EXCEPTION 'AD211: PARO clave=contrato_nucleo esperado=privado_marca_unica_sin_ficha actual=incompatible' USING ERRCODE='55000'; END IF;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
 INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
 INTO compartidas FROM pg_shdepend d WHERE d.classid='pg_proc'::regclass AND d.objid=f
  AND d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database());
 nuevo:=replace(original,marca,extension||marca);
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo
    OR replace(actual,extension||marca,marca) IS DISTINCT FROM original
    OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
        FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
        FROM pg_shdepend d WHERE d.classid='pg_proc'::regclass AND d.objid=f
         AND d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database())) IS DISTINCT FROM compartidas
 THEN RAISE EXCEPTION 'AD211: PARO clave=postimagen_nucleo esperado=solo_ficha_metadatos_intactos actual=divergente' USING ERRCODE='55000'; END IF;
END $nucleo$;

DO $audiencias$
DECLARE anterior text; nueva text; h text;
BEGIN
 SELECT pg_get_constraintdef(c.oid,false) INTO STRICT anterior FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
  AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 h:=encode(sha256(convert_to(anterior,'UTF8')),'hex');
 IF h IS DISTINCT FROM '8496cf66ff2f32b4afb6954708dab3a0e720f53a12aad564ed717828eb4284d4'
 THEN RAISE EXCEPTION 'AD211: PARO clave=CHECK_relectura_SHA esperado=% actual=%','8496cf66ff2f32b4afb6954708dab3a0e720f53a12aad564ed717828eb4284d4',h USING ERRCODE='55000'; END IF;
 nueva:='CHECK (('||substr(anterior,8,length(anterior)-8)||') OR audiencia_consumo = ''vec_personal.registro_empleado.ficha_propia.v1'')';
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||nueva;
 IF NOT EXISTS(SELECT 1 FROM pg_constraint c
   WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
    AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated
    AND strpos(pg_get_constraintdef(c.oid,false),'vec_personal.registro_empleado.ficha_propia.v1')>0)
 THEN RAISE EXCEPTION 'AD211: PARO clave=CHECK_postimagen esperado=audiencia_ficha actual=ausente' USING ERRCODE='55000'; END IF;
END $audiencias$;

CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_ficha_propia_empleado_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,
 consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; x record;
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR current_setting('TimeZone')<>'UTC'
    OR p_capacidad IS NULL
    OR p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 1 AND 524288
    OR p_motivo IS NULL OR octet_length(p_motivo) NOT BETWEEN 1 AND 65536
    OR p_contexto IS NULL OR octet_length(p_contexto) NOT BETWEEN 1 AND 262144
    OR p_persona_version IS NULL OR p_perfil_version IS NULL
    OR p_payload IS NULL OR octet_length(p_payload) NOT BETWEEN 1 AND 1048576
    OR p_sobre IS NULL OR octet_length(p_sobre) NOT BETWEEN 1 AND 1048576
    OR p_evidencia IS NULL OR octet_length(p_evidencia) NOT BETWEEN 1 AND 262144
    OR p_raiz IS NULL OR octet_length(p_raiz)<>44
 THEN RAISE EXCEPTION 'AD211: ficha propia denegada' USING ERRCODE='42501'; END IF;
 IF vec_autorizacion_atestada_v3.capacidad_cruda_prevalida(p_capacidad) IS NOT TRUE
 THEN RAISE EXCEPTION 'AD211: ficha propia denegada' USING ERRCODE='42501'; END IF;
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'AD211: material inválido' USING ERRCODE='22023'; END;
 IF jsonb_typeof(c) IS DISTINCT FROM 'object' OR jsonb_typeof(d) IS DISTINCT FROM 'object'
    OR d->'concedida' IS DISTINCT FROM 'true'::jsonb
    OR d->>'modulo_id' IS DISTINCT FROM 'personal'
    OR d->>'accion' IS DISTINCT FROM c->>'operacion'
    OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
    OR c->>'operacion' IS DISTINCT FROM 'personal.registro_empleado.ficha_propia.consultar'
    OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_personal.registro_empleado.ficha_propia.v1'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'ficha_propia_empleado'
    OR d->>'finalidad' IS DISTINCT FROM 'consultar_ficha_propia'
    OR d->'campos_permitidos' IS DISTINCT FROM '["corte","evidencia","relaciones","servicios"]'::jsonb
 THEN RAISE EXCEPTION 'AD211: ficha propia denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  'registro_empleado_b2',p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE OR x.decision_ref IS DISTINCT FROM d->>'decision_ref'
    OR x.efecto_ref IS DISTINCT FROM c->>'efecto_ref'
    OR x.huella_efecto_sha256 IS DISTINCT FROM c->>'huella_efecto_sha256'
    OR x.consumo_huella_sha256 IS NULL OR x.consumo_huella_sha256 !~ '^[0-9a-f]{64}$'
    OR x.auditoria_ref IS DISTINCT FROM 'aud_v3_'||substr(x.consumo_huella_sha256,1,32)
    OR x.consumida_en IS NULL OR NOT isfinite(x.consumida_en)
 THEN RAISE EXCEPTION 'AD211: consumo de ficha propia divergente' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,
  x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_ficha_propia_empleado_v3_atestada(
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_ficha_propia_empleado_v3_atestada(
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_personal_propietario;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_personal_propietario;
DO $acl$
DECLARE f oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_ficha_propia_empleado_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f
   AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef
   AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s'])
    OR NOT has_function_privilege('vec_personal_propietario',f,'EXECUTE')
    OR has_function_privilege('vec_personal_ejecutor',f,'EXECUTE')
    OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f)<>2
    OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
     WHERE p.oid=f AND (a.grantee NOT IN(p.proowner,'vec_personal_propietario'::regrole)
      OR a.privilege_type<>'EXECUTE' OR a.is_grantable OR a.grantor<>p.proowner))
 THEN RAISE EXCEPTION 'AD211: PARO clave=ACL esperado=solo_AD_y_Personal_propietarios actual=ampliada' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
