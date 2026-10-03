\set ON_ERROR_STOP on
-- AUT37: arranque central privado 2 Aplicación + 1 Sistemas. V2 permanece cerrado.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:admin:continuidad:v1',0));
DO $pre$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
 OR pg_catalog.to_regprocedure('vec_autorizacion.canon_asignacion_perfil_admin_v1(jsonb)') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_evento_admin_preperfil_v1(jsonb)') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion.registrar_bootstrap_central_admin_v3(text,text)') IS NOT NULL
 OR pg_catalog.to_regprocedure('vec_personal.cotejar_unidad_bootstrap_admin_v1(text,jsonb,timestamptz)') IS NULL
 THEN RAISE EXCEPTION 'AUT37: PARO clave=preimagen actual=incompatible esperado=AUT24_AD171_Personal31_sin_AUT37' USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_propietario;

-- encoding/json: orden exacto pactado con PlanBootstrapAdministracionV3@158eea363.
-- Cada objeto tiene claves cerradas; JSONB nunca forma los bytes de aprobación.
CREATE FUNCTION vec_autorizacion.canon_bootstrap_central_admin_v3(v jsonb,tipo text)
RETURNS text LANGUAGE plpgsql IMMUTABLE SECURITY INVOKER
SET search_path=pg_catalog AS $f$
DECLARE campos text[];tipos text[];i integer;resultado text:='';valor text;
 elemento jsonb;lista text;clave text;minimo integer;maximo integer;
BEGIN
 CASE tipo
 WHEN 'plan' THEN
  campos:=ARRAY['version','preparado_en','caduca_en','control_continuidad_revision_esperada','bootstrap_estado_esperado','rol','fuente_identidad','fuente_ca_admin','personas','gobierno','fuente_reparto_aprobado'];
  tipos:=ARRAY['n','t','t','n','s','rol','evidencia','evidencia','persona[]','gobierno','evidencia'];
 WHEN 'persona' THEN
  campos:=ARRAY['cuenta_ref','cuenta_version','persona_ref','persona_version','perfil_ref','vinculo_ref','preimagen_huella_sha256','procedencia','vigente_hasta','certificado_admin','sistemas','ambitos'];
  tipos:=ARRAY['s','n','s','n','s','s','s','evidencia','t','certificado','sistemas[]','ambito[]'];
 WHEN 'sistemas' THEN
  campos:=ARRAY['rol','perfil_ref','vinculo_ref','vigente_hasta','ambitos'];tipos:=ARRAY['rol','s','s','t','ambito[]'];
 WHEN 'rol' THEN
  campos:=ARRAY['version_ref','huella_sha256','control_revision','control_huella_sha256'];tipos:=ARRAY['s','s','n','s'];
 WHEN 'evidencia' THEN
  campos:=ARRAY['referencia','version','huella_sha256'];tipos:=ARRAY['s','n','s'];
 WHEN 'certificado' THEN
  campos:=ARRAY['persona_ref','cuenta_ref','huella_sha256','ca_huella_sha256','acreditacion'];tipos:=ARRAY['s','s','s','s','evidencia'];
 WHEN 'ambito' THEN
  campos:=ARRAY['dimension','valores','fuente'];tipos:=ARRAY['s','s[]','evidencia'];
 WHEN 'ambito_fijo' THEN
  campos:=ARRAY['clave','valores'];tipos:=ARRAY['s','s[]'];
 WHEN 'gobierno' THEN
  campos:=ARRAY['audiencia_selector_admin','audiencia_administrativa','politica_certificado_ref','politica_certificado_huella_sha256','roles','motivos'];tipos:=ARRAY['s','s','s','s','rol_gobernado[]','motivo[]'];
 WHEN 'rol_gobernado' THEN
  campos:=ARRAY['version_ref','huella_sha256','clase','categoria_admin','unidad_requerida','ambitos_fijos','vigente_desde','vigente_hasta','duracion_propuesta_segundos','fuente_categoria','dimensiones_ambito','rol_id'];
  tipos:=ARRAY['s','s','s','s','b','ambito_fijo[]','t','t','n','evidencia','s[]','s'];
 WHEN 'motivo' THEN
  campos:=ARRAY['accion','tipo_recurso','finalidad','referencia'];tipos:=ARRAY['s','s','s','motivo_ref'];
 WHEN 'motivo_ref' THEN
  campos:=ARRAY['catalogo_id','catalogo_version','catalogo_huella_sha256','entrada_clave'];tipos:=ARRAY['s','n','s','s'];
 WHEN 's' THEN
  IF pg_catalog.jsonb_typeof(v) IS DISTINCT FROM 'string'
  OR pg_catalog.octet_length(v#>>'{}') NOT BETWEEN 1 AND 256
  OR (v#>>'{}') ~ '[[:cntrl:]]' OR pg_catalog.strpos(v#>>'{}',' ')<>0
  OR pg_catalog.strpos(v#>>'{}','*')<>0
  THEN RAISE EXCEPTION 'AUT37: PARO clave=canon_string actual=invalido esperado=referencia_acotada' USING ERRCODE='22023'; END IF;
  RETURN vec_autorizacion.json_cadena_canonica_go_admin_v1(v#>>'{}');
 WHEN 'n' THEN
  IF pg_catalog.jsonb_typeof(v) IS DISTINCT FROM 'number'
  OR (v#>>'{}') !~ '^[1-9][0-9]{0,19}$'
  OR (v#>>'{}')::numeric>18446744073709551615::numeric
  THEN RAISE EXCEPTION 'AUT37: PARO clave=canon_entero actual=invalido esperado=uint64_positivo' USING ERRCODE='22023'; END IF;
  RETURN v#>>'{}';
 WHEN 'b' THEN
  IF pg_catalog.jsonb_typeof(v) IS DISTINCT FROM 'boolean'
  THEN RAISE EXCEPTION 'AUT37: PARO clave=canon_boolean actual=invalido esperado=boolean' USING ERRCODE='22023'; END IF;
  RETURN v::text;
 WHEN 't' THEN
  IF pg_catalog.jsonb_typeof(v) IS DISTINCT FROM 'string'
  OR (v#>>'{}') !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$'
  OR vec_autorizacion.fecha_canonica_go_admin_v1(v#>>'{}',false) IS DISTINCT FROM v#>>'{}'
  THEN RAISE EXCEPTION 'AUT37: PARO clave=canon_fecha actual=invalido esperado=UTC_segundos' USING ERRCODE='22023'; END IF;
  RETURN vec_autorizacion.json_cadena_canonica_go_admin_v1(v#>>'{}');
 ELSE RAISE EXCEPTION 'AUT37: PARO clave=canon_tipo actual=desconocido esperado=ABI_V3' USING ERRCODE='22023';
 END CASE;
 IF pg_catalog.jsonb_typeof(v) IS DISTINCT FROM 'object'
 OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(v))<>pg_catalog.cardinality(campos)
 OR (v ?& campos) IS NOT TRUE
 THEN RAISE EXCEPTION 'AUT37: PARO clave=canon_campos actual=divergente esperado=ABI_cerrada' USING ERRCODE='22023'; END IF;
 FOR i IN 1..pg_catalog.cardinality(campos) LOOP
  IF pg_catalog.right(tipos[i],2)='[]' THEN
   IF pg_catalog.jsonb_typeof(v->campos[i]) IS DISTINCT FROM 'array'
   THEN RAISE EXCEPTION 'AUT37: PARO clave=canon_lista actual=invalida esperado=array' USING ERRCODE='22023'; END IF;
   minimo:=CASE WHEN tipos[i] IN('sistemas[]','motivo[]') THEN 0 ELSE 1 END;
   maximo:=CASE WHEN tipos[i]='persona[]' THEN 2 WHEN tipos[i]='sistemas[]' THEN 1 ELSE 64 END;
   IF pg_catalog.jsonb_array_length(v->campos[i]) NOT BETWEEN minimo AND maximo
   THEN RAISE EXCEPTION 'AUT37: PARO clave=canon_cardinalidad actual=invalida esperado=lista_acotada' USING ERRCODE='22023'; END IF;
   lista:='';
   FOR elemento IN SELECT e.value FROM pg_catalog.jsonb_array_elements(v->campos[i]) WITH ORDINALITY e(value,n) ORDER BY n LOOP
    IF lista<>'' THEN lista:=lista||','; END IF;
    lista:=lista||vec_autorizacion.canon_bootstrap_central_admin_v3(elemento,pg_catalog.left(tipos[i],pg_catalog.length(tipos[i])-2));
   END LOOP;
   valor:='['||lista||']';
  ELSE valor:=vec_autorizacion.canon_bootstrap_central_admin_v3(v->campos[i],tipos[i]); END IF;
  IF resultado<>'' THEN resultado:=resultado||','; END IF;
  resultado:=resultado||vec_autorizacion.json_cadena_canonica_go_admin_v1(campos[i])||':'||valor;
 END LOOP;
 RETURN '{'||resultado||'}';
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.canon_bootstrap_central_admin_v3(jsonb,text) FROM PUBLIC;


RESET ROLE;
DO $rol$
BEGIN
 IF pg_catalog.to_regrole('vec_admin_bootstrap_central_v3_ejecutor') IS NOT NULL
 THEN RAISE EXCEPTION 'AUT37: PARO clave=rol_tecnico actual=preexistente esperado=nuevo' USING ERRCODE='55000'; END IF;
 CREATE ROLE vec_admin_bootstrap_central_v3_ejecutor NOLOGIN NOINHERIT NOSUPERUSER
  NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
 EXECUTE pg_catalog.format('GRANT CONNECT ON DATABASE %I TO vec_admin_bootstrap_central_v3_ejecutor',pg_catalog.current_database());
END $rol$;
SET LOCAL ROLE vec_autorizacion_propietario;

-- Configuración DBA externa: ninguna fila, LOGIN, certificado o fuente se siembra.
-- Plan SHA y preimagen aprobados vinculan al operador; el CLI no los autoconcede.
CREATE TABLE vec_autorizacion.config_bootstrap_central_admin_v3(
 login_nombre name PRIMARY KEY,
 proceso text NOT NULL CHECK(proceso ~ '^[a-z][a-z0-9._-]{1,79}$'),
 plan_sha256 text NOT NULL UNIQUE CHECK(plan_sha256 ~ '^[0-9a-f]{64}$'),
 preimagen_sha256 text NOT NULL CHECK(preimagen_sha256 ~ '^[0-9a-f]{64}$'),
 aprobacion_ref text NOT NULL CHECK(aprobacion_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'),
 aprobacion_sha256 text NOT NULL CHECK(aprobacion_sha256 ~ '^[0-9a-f]{64}$'),
 fuente_reparto jsonb NOT NULL,
 fuente_identidad jsonb NOT NULL,
 fuente_ca_admin jsonb NOT NULL,
 vigente_desde timestamptz(6) NOT NULL,
 vigente_hasta timestamptz(6) NOT NULL,
 CHECK(pg_catalog.isfinite(vigente_desde) AND pg_catalog.isfinite(vigente_hasta) AND vigente_hasta>vigente_desde),
 CHECK(vec_autorizacion.canon_bootstrap_central_admin_v3(fuente_reparto,'evidencia') IS NOT NULL),
 CHECK(vec_autorizacion.canon_bootstrap_central_admin_v3(fuente_identidad,'evidencia') IS NOT NULL),
 CHECK(vec_autorizacion.canon_bootstrap_central_admin_v3(fuente_ca_admin,'evidencia') IS NOT NULL)
);
CREATE TABLE vec_autorizacion.bootstrap_central_admin_v3(
 singleton boolean PRIMARY KEY DEFAULT true CHECK(singleton),
 acto_ref text NOT NULL UNIQUE CHECK(acto_ref ~ '^acto_admin:[0-9a-f]{32}$'),
 recibo_ref text NOT NULL UNIQUE CHECK(recibo_ref ~ '^recibo_bootstrap:[0-9a-f]{32}$'),
 plan_canonico bytea NOT NULL CHECK(pg_catalog.octet_length(plan_canonico) BETWEEN 1 AND 65536),
 huella_plan_sha256 text NOT NULL UNIQUE CHECK(huella_plan_sha256=pg_catalog.encode(pg_catalog.sha256(plan_canonico),'hex')),
 preimagen_canonica bytea NOT NULL,
 preimagen_sha256 text NOT NULL CHECK(preimagen_sha256=pg_catalog.encode(pg_catalog.sha256(preimagen_canonica),'hex')),
 operador_login name NOT NULL,
 aprobacion_ref text NOT NULL,
 aprobacion_sha256 text NOT NULL,
 auditoria_ref text NOT NULL UNIQUE CHECK(auditoria_ref ~ '^aud_v3_p_[0-9a-f]{32}$'),
 auditoria_secuencia numeric(20,0) NOT NULL,
 auditoria_huella_sha256 text NOT NULL CHECK(auditoria_huella_sha256 ~ '^[0-9a-f]{64}$'),
 auditoria_registrada_en timestamptz(6) NOT NULL,
 resultado jsonb NOT NULL CHECK(pg_catalog.jsonb_typeof(resultado)='object'),
 confirmado_en timestamptz(6) NOT NULL CHECK(pg_catalog.isfinite(confirmado_en))
);
CREATE TABLE vec_autorizacion.outbox_bootstrap_central_admin_v3(
 acto_ref text PRIMARY KEY REFERENCES vec_autorizacion.bootstrap_central_admin_v3(acto_ref),
 evento text NOT NULL CHECK(evento='bootstrap_central_confirmado'),
 auditoria_ref text NOT NULL,
 creada_en timestamptz(6) NOT NULL CHECK(pg_catalog.isfinite(creada_en))
);
DO $historia$
DECLARE t text;
BEGIN
 FOREACH t IN ARRAY ARRAY['config_bootstrap_central_admin_v3','bootstrap_central_admin_v3','outbox_bootstrap_central_admin_v3'] LOOP
  EXECUTE pg_catalog.format('ALTER TABLE vec_autorizacion.%I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE pg_catalog.format('ALTER TABLE vec_autorizacion.%I FORCE ROW LEVEL SECURITY',t);
  EXECUTE pg_catalog.format('CREATE POLICY propietario_exacto ON vec_autorizacion.%I FOR ALL TO vec_autorizacion_propietario USING(current_user=''vec_autorizacion_propietario'') WITH CHECK(current_user=''vec_autorizacion_propietario'')',t);
  EXECUTE pg_catalog.format('CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_autorizacion.%I FOR EACH ROW EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable()',t);
  EXECUTE pg_catalog.format('CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_autorizacion.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable()',t);
  EXECUTE pg_catalog.format('REVOKE ALL ON TABLE vec_autorizacion.%I FROM PUBLIC,vec_admin_bootstrap_central_v3_ejecutor',t);
  EXECUTE pg_catalog.format('REVOKE ALL ON TYPE vec_autorizacion.%I FROM PUBLIC,vec_admin_bootstrap_central_v3_ejecutor',t);
 END LOOP;
END $historia$;

CREATE FUNCTION vec_autorizacion.exigir_operador_bootstrap_central_admin_v3()
RETURNS vec_autorizacion.config_bootstrap_central_admin_v3
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on
AS $f$
DECLARE l record;g record;cfg vec_autorizacion.config_bootstrap_central_admin_v3;
 ns oid;db oid;funciones oid[];
BEGIN
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable'
 OR pg_catalog.current_setting('transaction_read_only')<>'off'
 OR pg_catalog.current_setting('TimeZone')<>'UTC'
 OR pg_catalog.current_setting('role')<>'none'
 THEN RAISE EXCEPTION 'AUT37: PARO clave=transaccion actual=incompatible esperado=serializable_RW_UTC_sin_SETROLE' USING ERRCODE='25000'; END IF;
 SELECT * INTO l FROM pg_catalog.pg_roles WHERE rolname=session_user;
 SELECT * INTO g FROM pg_catalog.pg_roles WHERE rolname='vec_admin_bootstrap_central_v3_ejecutor';
 ns:=pg_catalog.to_regnamespace('vec_autorizacion');
 SELECT oid INTO db FROM pg_catalog.pg_database WHERE datname=pg_catalog.current_database();
 funciones:=ARRAY[pg_catalog.to_regprocedure('vec_autorizacion.preflight_bootstrap_central_admin_v3()'),pg_catalog.to_regprocedure('vec_autorizacion.registrar_bootstrap_central_admin_v3(text,text)')];
 IF l.oid IS NULL OR g.oid IS NULL OR NOT l.rolcanlogin OR NOT l.rolinherit
 OR l.rolsuper OR l.rolcreaterole OR l.rolcreatedb OR l.rolreplication OR l.rolbypassrls OR l.rolconfig IS NOT NULL
 OR l.rolname::text !~ '^[a-z][a-z0-9._-]{1,62}$'
 OR g.rolcanlogin OR g.rolinherit OR g.rolsuper OR g.rolcreaterole OR g.rolcreatedb OR g.rolreplication OR g.rolbypassrls OR g.rolconfig IS NOT NULL
 OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_auth_members WHERE member=l.oid)<>1
 OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members WHERE member=l.oid AND roleid=g.oid AND NOT admin_option AND inherit_option AND NOT set_option)
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members WHERE member=g.oid)
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_db_role_setting WHERE setrole IN(l.oid,g.oid))
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_shdepend WHERE refclassid='pg_catalog.pg_authid'::regclass AND refobjid=l.oid)
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_shdepend WHERE refclassid='pg_catalog.pg_authid'::regclass AND refobjid=g.oid AND NOT (
   (dbid=db AND classid='pg_catalog.pg_namespace'::regclass AND objid=ns AND deptype='a')
   OR (dbid=db AND classid='pg_catalog.pg_proc'::regclass AND objid=ANY(funciones) AND deptype='a')
   OR (dbid=0 AND classid='pg_catalog.pg_database'::regclass AND objid=db AND deptype='a')))
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_namespace n,
  LATERAL pg_catalog.aclexplode(COALESCE(n.nspacl,pg_catalog.acldefault('n',n.nspowner))) a
  WHERE n.oid=ns AND a.grantee=g.oid AND (a.privilege_type<>'USAGE' OR a.is_grantable))
 OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_namespace n,
  LATERAL pg_catalog.aclexplode(COALESCE(n.nspacl,pg_catalog.acldefault('n',n.nspowner))) a
  WHERE n.oid=ns AND a.grantee=g.oid AND a.privilege_type='USAGE' AND NOT a.is_grantable)<>1
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_database d,
  LATERAL pg_catalog.aclexplode(COALESCE(d.datacl,pg_catalog.acldefault('d',d.datdba))) a
  WHERE d.oid=db AND a.grantee=g.oid AND (a.privilege_type<>'CONNECT' OR a.is_grantable))
 OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_database d,
  LATERAL pg_catalog.aclexplode(COALESCE(d.datacl,pg_catalog.acldefault('d',d.datdba))) a
  WHERE d.oid=db AND a.grantee=g.oid AND a.privilege_type='CONNECT' AND NOT a.is_grantable)<>1
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_proc f,
  LATERAL pg_catalog.aclexplode(COALESCE(f.proacl,pg_catalog.acldefault('f',f.proowner))) a
  WHERE f.oid=ANY(funciones) AND a.grantee=g.oid AND (a.privilege_type<>'EXECUTE' OR a.is_grantable))
 OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_proc f,
  LATERAL pg_catalog.aclexplode(COALESCE(f.proacl,pg_catalog.acldefault('f',f.proowner))) a
  WHERE f.oid=ANY(funciones) AND a.grantee=g.oid AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)<>2
 OR pg_catalog.has_schema_privilege(l.oid,ns,'CREATE')
 OR pg_catalog.has_database_privilege(l.oid,db,'CREATE,TEMP')
 THEN RAISE EXCEPTION 'AUT37: PARO clave=operador actual=no_acreditado esperado=LOGIN_exclusivo_minimo' USING ERRCODE='42501'; END IF;
 SELECT * INTO cfg FROM vec_autorizacion.config_bootstrap_central_admin_v3 WHERE login_nombre=session_user FOR SHARE;
 IF NOT FOUND OR pg_catalog.clock_timestamp()<cfg.vigente_desde OR pg_catalog.clock_timestamp()>=cfg.vigente_hasta
 THEN RAISE EXCEPTION 'AUT37: PARO clave=configuracion actual=ausente_o_caducada esperado=plan_aprobado_vigente' USING ERRCODE='42501'; END IF;
 RETURN cfg;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.exigir_operador_bootstrap_central_admin_v3() FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.preflight_bootstrap_central_admin_v3()
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
BEGIN
 PERFORM vec_autorizacion.exigir_operador_bootstrap_central_admin_v3();
 RETURN true;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.preflight_bootstrap_central_admin_v3() FROM PUBLIC;


RESET ROLE;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
-- CA coteja la organización propia. La unidad es sólo una proyección del DTO:
-- AUT debe acreditarla mediante Personal31 antes de usar esta proyección.
CREATE FUNCTION vec_contexto_actor_v1.cotejar_ambitos_bootstrap_central_admin_v3(p_ambitos jsonb,p_hasta timestamptz)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' AS $f$
DECLARE a jsonb;o record;resultado jsonb:='[]'::jsonb;anterior text:='';
BEGIN
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable' OR pg_catalog.current_setting('transaction_read_only')<>'off'
 OR pg_catalog.jsonb_typeof(p_ambitos) IS DISTINCT FROM 'array' OR pg_catalog.jsonb_array_length(p_ambitos) NOT BETWEEN 1 AND 2
 THEN RAISE EXCEPTION 'AUT37: PARO clave=ambitos actual=invalido esperado=ambitos_propietarios' USING ERRCODE='22023'; END IF;
 FOR a IN SELECT e.value FROM pg_catalog.jsonb_array_elements(p_ambitos) WITH ORDINALITY e(value,n) ORDER BY n LOOP
  IF pg_catalog.jsonb_typeof(a) IS DISTINCT FROM 'object'
  OR pg_catalog.jsonb_typeof(a->'valores') IS DISTINCT FROM 'array' OR pg_catalog.jsonb_array_length(a->'valores')<>1
  OR (a->>'dimension') COLLATE "C"<=anterior COLLATE "C" OR a->>'dimension' NOT IN('organizacion_ref','unidad_ref')
  THEN RAISE EXCEPTION 'AUT37: PARO clave=dimension actual=divergente esperado=org_unidad_ordenadas' USING ERRCODE='22023'; END IF;
  anterior:=a->>'dimension';
  IF anterior='unidad_ref' THEN
   resultado:=resultado||pg_catalog.jsonb_build_array(pg_catalog.jsonb_build_object('clave',anterior,'valores',a->'valores'));
   CONTINUE;
  END IF;
  SELECT v.* INTO o FROM vec_contexto_actor_v1.organizacion_actual c
   JOIN vec_contexto_actor_v1.organizacion_versiones v USING(organizacion_ref,version)
   WHERE c.organizacion_ref=a#>>'{valores,0}' FOR SHARE OF c;
  IF NOT FOUND OR o.estado<>'activo' OR o.procedencia_autoridad<>'autoridad_maestra_acreditada'
  OR o.procedencia_ref IS DISTINCT FROM a#>>'{fuente,referencia}'
  OR o.procedencia_version::text IS DISTINCT FROM a#>>'{fuente,version}'
  OR o.procedencia_huella_sha256 IS DISTINCT FROM a#>>'{fuente,huella_sha256}'
  OR pg_catalog.clock_timestamp()<o.vigente_desde OR pg_catalog.clock_timestamp()>=o.vigente_hasta
  OR p_hasta>o.vigente_hasta
  THEN RAISE EXCEPTION 'AUT37: PARO clave=fuente_organizacion actual=no_acreditada esperado=version_maestra_vigente' USING ERRCODE='42501'; END IF;
  resultado:=resultado||pg_catalog.jsonb_build_array(pg_catalog.jsonb_build_object('clave',anterior,'valores',a->'valores'));
 END LOOP;
 IF p_ambitos->0->>'dimension' IS DISTINCT FROM 'organizacion_ref'
 THEN RAISE EXCEPTION 'AUT37: PARO clave=organizacion actual=ausente esperado=organizacion_exacta' USING ERRCODE='42501'; END IF;
 RETURN resultado;
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.cotejar_ambitos_bootstrap_central_admin_v3(jsonb,timestamptz) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.cotejar_ambitos_bootstrap_central_admin_v3(jsonb,timestamptz) TO vec_autorizacion_propietario;

CREATE FUNCTION vec_contexto_actor_v1.preimagen_persona_bootstrap_central_admin_v3(p jsonb)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' AS $f$
DECLARE c jsonb;sub jsonb;sistema jsonb;sistemas jsonb:='[]'::jsonb;ambitos jsonb;organizaciones jsonb;
BEGIN
 c:=vec_contexto_actor_v1.preimagen_admin_interna_v1(p->>'cuenta_ref',p->>'persona_ref',p->>'perfil_ref',p->>'vinculo_ref');
 IF c#>>'{cuenta,version}' IS DISTINCT FROM p->>'cuenta_version'
 OR c#>>'{persona,version}' IS DISTINCT FROM p->>'persona_version'
 OR c->'perfil' IS DISTINCT FROM 'null'::jsonb OR c->'vinculo' IS DISTINCT FROM 'null'::jsonb
 OR c#>>'{persona,procedencia_ref}' IS DISTINCT FROM p#>>'{procedencia,referencia}'
 OR c#>>'{persona,procedencia_version}' IS DISTINCT FROM p#>>'{procedencia,version}'
 OR c#>>'{persona,procedencia_huella_sha256}' IS DISTINCT FROM p#>>'{procedencia,huella_sha256}'
 OR (p->>'vigente_hasta')::timestamptz>LEAST((c#>>'{persona,vigente_hasta}')::timestamptz,(c#>>'{cuenta,vigente_hasta}')::timestamptz)
 OR EXISTS(SELECT 1 FROM vec_contexto_actor_v1.perfil_versiones WHERE perfil_ref=p->>'perfil_ref')
 OR EXISTS(SELECT 1 FROM vec_contexto_actor_v1.vinculo_contexto_versiones WHERE vinculo_ref=p->>'vinculo_ref')
 OR NOT EXISTS(SELECT 1 FROM vec_contexto_actor_v1.vinculo_contexto_actual a
  JOIN vec_contexto_actor_v1.vinculo_contexto_versiones v USING(vinculo_ref,version)
  WHERE v.cuenta_ref=p->>'cuenta_ref' AND v.persona_ref=p->>'persona_ref' AND v.estado='activo'
   AND v.procedencia_autoridad='autoridad_maestra_acreditada'
   AND pg_catalog.clock_timestamp()>=v.vigente_desde AND (p->>'vigente_hasta')::timestamptz<=v.vigente_hasta)
 THEN RAISE EXCEPTION 'AUT37: PARO clave=persona_preimagen actual=divergente esperado=fuente_viva_refs_nuevas' USING ERRCODE='40001'; END IF;
 ambitos:=vec_contexto_actor_v1.cotejar_ambitos_bootstrap_central_admin_v3(p->'ambitos',(p->>'vigente_hasta')::timestamptz);
 FOR sistema IN SELECT value FROM pg_catalog.jsonb_array_elements(p->'sistemas') LOOP
  sub:=vec_contexto_actor_v1.preimagen_admin_interna_v1(p->>'cuenta_ref',p->>'persona_ref',sistema->>'perfil_ref',sistema->>'vinculo_ref');
  IF sub->'perfil' IS DISTINCT FROM 'null'::jsonb OR sub->'vinculo' IS DISTINCT FROM 'null'::jsonb
  OR EXISTS(SELECT 1 FROM vec_contexto_actor_v1.perfil_versiones WHERE perfil_ref=sistema->>'perfil_ref')
  OR EXISTS(SELECT 1 FROM vec_contexto_actor_v1.vinculo_contexto_versiones WHERE vinculo_ref=sistema->>'vinculo_ref')
  OR (sistema->>'vigente_hasta')::timestamptz>(p->>'vigente_hasta')::timestamptz
  THEN RAISE EXCEPTION 'AUT37: PARO clave=sistemas_preimagen actual=divergente esperado=perfil_separado_nuevo' USING ERRCODE='40001'; END IF;
  sistemas:=sistemas||pg_catalog.jsonb_build_array(pg_catalog.jsonb_build_object('contexto',sub,
    'ambitos',vec_contexto_actor_v1.cotejar_ambitos_bootstrap_central_admin_v3(sistema->'ambitos',(sistema->>'vigente_hasta')::timestamptz)));
 END LOOP;
 SELECT pg_catalog.jsonb_agg(pg_catalog.to_jsonb(v) ORDER BY v.organizacion_ref) INTO organizaciones
 FROM vec_contexto_actor_v1.organizacion_actual a
 JOIN vec_contexto_actor_v1.organizacion_versiones v USING(organizacion_ref,version)
 WHERE v.organizacion_ref IN(SELECT x.value#>>'{valores,0}' FROM pg_catalog.jsonb_array_elements(p->'ambitos') x WHERE x.value->>'dimension'='organizacion_ref')
 OR v.organizacion_ref IN(SELECT x.value#>>'{valores,0}' FROM pg_catalog.jsonb_array_elements(p->'sistemas') sy,
   LATERAL pg_catalog.jsonb_array_elements(sy.value->'ambitos') x WHERE x.value->>'dimension'='organizacion_ref');
 RETURN pg_catalog.jsonb_build_object('esquema','vec.admin.bootstrap.persona.preimagen.v3','contexto',c,'sistemas',sistemas,'ambitos',ambitos,'organizaciones',organizaciones);
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.preimagen_persona_bootstrap_central_admin_v3(jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.preimagen_persona_bootstrap_central_admin_v3(jsonb) TO vec_autorizacion_propietario;

SET LOCAL ROLE vec_identidad_sesiones_v1_propietario;
CREATE FUNCTION vec_identidad_sesiones_v1.preimagen_certificado_bootstrap_central_admin_v3(p jsonb,gobierno jsonb)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' AS $f$
DECLARE identidad jsonb;v record;cert jsonb:=p->'certificado_admin';anterior jsonb:='null'::jsonb;
BEGIN
 identidad:=vec_identidad_sesiones_v1.preimagen_bootstrap_admin_interna_v2(p->>'persona_ref',p->>'cuenta_ref',gobierno,cert->>'ca_huella_sha256');
 IF cert->>'persona_ref' IS DISTINCT FROM p->>'persona_ref' OR cert->>'cuenta_ref' IS DISTINCT FROM p->>'cuenta_ref'
 OR cert->>'huella_sha256' IS NULL OR cert->>'huella_sha256' !~ '^[0-9a-f]{64}$'
 OR cert->>'huella_sha256'=pg_catalog.repeat('0',64)
 OR (p->>'vigente_hasta')::timestamptz>(identidad->>'vigente_hasta')::timestamptz
 THEN RAISE EXCEPTION 'AUT37: PARO clave=certificado actual=divergente esperado=persona_cuenta_CA_aprobadas' USING ERRCODE='42501'; END IF;
 SELECT x.* INTO v FROM vec_identidad_sesiones_v1.vinculo_certificado_admin_actual_v1 a
  JOIN vec_identidad_sesiones_v1.vinculo_certificado_admin_v1 x USING(vinculo_ref,version)
  WHERE x.certificado_sha256=cert->>'huella_sha256' FOR SHARE OF a;
 IF FOUND THEN
  IF v.persona_ref IS DISTINCT FROM p->>'persona_ref' OR v.cuenta_privilegiada_ref IS DISTINCT FROM p->>'cuenta_ref'
  OR v.ca_sha256 IS DISTINCT FROM cert->>'ca_huella_sha256' OR v.politica_ref IS DISTINCT FROM gobierno->>'politica_certificado_ref'
  OR v.estado<>'activo' OR pg_catalog.clock_timestamp()<v.vigente_desde OR (p->>'vigente_hasta')::timestamptz>v.vigente_hasta
  THEN RAISE EXCEPTION 'AUT37: PARO clave=certificado_existente actual=incompatible esperado=original_reutilizable' USING ERRCODE='40001'; END IF;
  anterior:=pg_catalog.to_jsonb(v);
 ELSIF EXISTS(SELECT 1 FROM vec_identidad_sesiones_v1.vinculo_certificado_admin_v1
   WHERE certificado_sha256=cert->>'huella_sha256' OR vinculo_ref=p->>'vinculo_ref') THEN
  RAISE EXCEPTION 'AUT37: PARO clave=certificado_historia actual=referencia_usada esperado=sin_revivir' USING ERRCODE='40001';
 END IF;
 RETURN pg_catalog.jsonb_build_object('identidad',identidad,'certificado_previo',anterior,'acreditacion_aprobada',cert->'acreditacion');
END $f$;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.preimagen_certificado_bootstrap_central_admin_v3(jsonb,jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_identidad_sesiones_v1.preimagen_certificado_bootstrap_central_admin_v3(jsonb,jsonb) TO vec_autorizacion_propietario;
SET LOCAL ROLE vec_autorizacion_propietario;


SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
-- Cotejo histórico de la misma corriente AD171. La recuperación no crea otro
-- eslabón cuando ya existe un recibo; una restauración incompleta se deniega.
CREATE FUNCTION vec_autorizacion_atestada_v3.cotejar_acuse_bootstrap_central_admin_v3(
 p_ref text,p_secuencia numeric,p_huella text,p_instante timestamptz,
 p_plan_sha256 text,p_operador name,p_aprobacion text,p_preimagen_sha256 text)
RETURNS boolean LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
 SELECT EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a
  WHERE a.auditoria_ref=p_ref AND a.secuencia=p_secuencia AND a.huella_sha256=p_huella
   AND a.registrada_en=p_instante AND a.tipo_registro='bootstrap_operador'
   AND a.actor_ref IS NULL AND a.perfil_activo_ref IS NULL AND a.decision_ref IS NULL
   AND a.operador_login=p_operador AND a.plan_sha256=p_plan_sha256 AND a.aprobacion_ref=p_aprobacion
   AND a.fuente_ref='bootstrap_preimagen:'||pg_catalog.substr(p_plan_sha256,1,32)
   AND a.fuente_sha256=p_preimagen_sha256 AND a.resultado='permitido'
   AND a.accion='ejecutar_plan_bootstrap_admin' AND a.canal='operacion_tecnica_privada'
   AND a.finalidad_ref='bootstrap_admin')
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.cotejar_acuse_bootstrap_central_admin_v3(text,numeric,text,timestamptz,text,name,text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.cotejar_acuse_bootstrap_central_admin_v3(text,numeric,text,timestamptz,text,name,text,text) TO vec_autorizacion_propietario;
SET LOCAL ROLE vec_autorizacion_propietario;

CREATE FUNCTION vec_autorizacion.cotejar_ambitos_bootstrap_central_admin_v3(p_ambitos jsonb,p_hasta timestamptz)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' AS $f$
DECLARE ambitos jsonb;unidades jsonb:='[]'::jsonb;a jsonb;org text;
BEGIN
 ambitos:=vec_contexto_actor_v1.cotejar_ambitos_bootstrap_central_admin_v3(p_ambitos,p_hasta);
 org:=p_ambitos#>>'{0,valores,0}';
 FOR a IN SELECT value FROM pg_catalog.jsonb_array_elements(p_ambitos) LOOP
  IF a->>'dimension'='unidad_ref' THEN
   unidades:=unidades||pg_catalog.jsonb_build_array(vec_personal.cotejar_unidad_bootstrap_admin_v1(org,a,p_hasta));
  END IF;
 END LOOP;
 RETURN pg_catalog.jsonb_build_object('ambitos',ambitos,'unidades',unidades);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.cotejar_ambitos_bootstrap_central_admin_v3(jsonb,timestamptz) FROM PUBLIC;

-- El ámbito asignado es un prefijo del límite publicado: organización siempre,
-- unidad cuando se asigna o es obligatoria. Los valores siguen siendo exactos.
CREATE FUNCTION vec_autorizacion.ambitos_limite_bootstrap_central_admin_v3(asignados jsonb,gobierno jsonb)
RETURNS boolean LANGUAGE sql IMMUTABLE SECURITY INVOKER SET search_path=pg_catalog AS $f$
 SELECT pg_catalog.jsonb_typeof(asignados)='array'
  AND pg_catalog.jsonb_array_length(asignados) BETWEEN 1 AND pg_catalog.jsonb_array_length(gobierno->'ambitos_fijos')
  AND (NOT (gobierno->>'unidad_requerida')::boolean OR pg_catalog.jsonb_array_length(asignados)=2)
  AND NOT EXISTS(SELECT 1 FROM pg_catalog.jsonb_array_elements(asignados) WITH ORDINALITY a(valor,n)
    WHERE a.valor IS DISTINCT FROM (gobierno->'ambitos_fijos')->(a.n::integer-1))
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion.ambitos_limite_bootstrap_central_admin_v3(jsonb,jsonb) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.administradores_aplicacion_efectivos_internos_v3()
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' AS $f$
DECLARE e jsonb;a record;r record;m record;c record;resultado jsonb:='[]'::jsonb;
BEGIN
 FOR e IN SELECT value FROM pg_catalog.jsonb_array_elements(vec_autorizacion.administradores_efectivos_internos_v1()) LOOP
  SELECT x.* INTO STRICT a FROM vec_autorizacion.asignacion_perfil x WHERE asignacion_ref=e->>'asignacion_ref';
  SELECT x.* INTO STRICT r FROM vec_autorizacion.version_rol x WHERE version_rol_ref=a.version_rol_ref FOR SHARE;
  SELECT x.* INTO m FROM vec_autorizacion.perfil_fijo_categoria_nominal_v1 x WHERE version_rol_ref=r.version_rol_ref FOR SHARE;
  -- La historia v1-v3 se conserva. Sin categoría positiva se exige adaptación
  -- gobernada; nunca se deduce Aplicación por el nombre o prefijo del rol.
  IF NOT FOUND OR m.tipo_perfil<>'fijo_sistema' OR m.categoria_administrativa NOT IN('aplicacion','sistemas')
  OR m.version_rol_huella_sha256 IS DISTINCT FROM r.huella_sha256 OR r.documento->>'estado'<>'publicada'
  OR r.huella_sha256 IS DISTINCT FROM pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(vec_autorizacion.canon_version_rol_admin_v1(r.documento),'UTF8')),'hex')
  OR a.huella_sha256 IS DISTINCT FROM pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(vec_autorizacion.canon_asignacion_perfil_admin_v1(a.documento),'UTF8')),'hex')
  THEN RAISE EXCEPTION 'AUT37: PARO clave=poblacion_categoria actual=no_acreditada esperado=adaptacion_gobernada_version_SHA' USING ERRCODE='42501'; END IF;
  SELECT x.* INTO STRICT c FROM vec_autorizacion.control_vigencia_version_rol_actual ac
   JOIN vec_autorizacion.control_vigencia_version_rol x USING(version_rol_ref,revision)
   WHERE ac.version_rol_ref=r.version_rol_ref FOR SHARE OF ac;
  IF c.estado<>'habilitada' OR c.huella_sha256 IS DISTINCT FROM pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(vec_autorizacion.canon_control_rol_admin_v1(c.documento),'UTF8')),'hex')
  THEN RAISE EXCEPTION 'AUT37: PARO clave=poblacion_control actual=no_vigente esperado=control_version_SHA_actual' USING ERRCODE='42501'; END IF;
  IF m.categoria_administrativa='aplicacion' THEN resultado:=resultado||pg_catalog.jsonb_build_array(e); END IF;
 END LOOP;
 RETURN resultado;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.administradores_aplicacion_efectivos_internos_v3() FROM PUBLIC;
-- Los seis consumidores de continuidad/doble control usan la nueva fuente.
-- Conserva OID, firma, propietario, ACL y configuración; AUT24 no se reescribe.
DO $poblacion_consumidores$
DECLARE item record;f oid;original text;fuente text;nueva text;meta jsonb;
 dup text:='EXISTS(SELECT 1 FROM jsonb_array_elements(antes->''administradores_efectivos'') x WHERE x->>''persona_ref''=o->>''persona_ref'')';
 dup_nueva text:='EXISTS(SELECT 1 FROM jsonb_array_elements(vec_autorizacion.administradores_efectivos_internos_v1()) x WHERE x->>''persona_ref''=o->>''persona_ref'')';
BEGIN
 FOR item IN SELECT * FROM (VALUES
('vec_autorizacion.preimagen_cambio_admin_interna_v1(jsonb)','8cc2ac15190f72e791bef0d186f8dc46d1a2bd554710e0e04cf086fbbbe5b4db'),
('vec_autorizacion.ejecutar_cambio_admin_interno_v1(jsonb,text,text,text,text,boolean)','d889a8ee47a0a8fbeb3b20d19850fcdc32d1a4509949f0a01dad916c890fb320'),
('vec_autorizacion.aplicar_acto_ordinario_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','b31fcfd1f3883cf23f1424ea5398ec5f85eab018ff8bf46ce78b793ac6574a69'),
('vec_autorizacion.proponer_acto_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','fe4c3ea2d50ef711caffb1068400aa6adb1cb28600363fc494fcb592f530bc91'),
('vec_autorizacion.cerrar_propuesta_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','bc80bc6555e2a3e8234f621815a8ae423bb178bffd88634767fa6bb4032797f8'),
('vec_autorizacion.preparar_preimagen_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','6d132721412ef0e079dd4a681b6fd9bedf304a03e26ba4deb629c9e13079d19a')
 ) d(firma,sha_fuente) LOOP
  f:=pg_catalog.to_regprocedure(item.firma);
  SELECT pg_catalog.pg_get_functiondef(p.oid),p.prosrc,pg_catalog.to_jsonb(p)-'prosrc' INTO STRICT original,fuente,meta FROM pg_catalog.pg_proc p WHERE p.oid=f;
  IF pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(fuente,'UTF8')),'hex') IS DISTINCT FROM item.sha_fuente
  THEN RAISE EXCEPTION 'AUT37: PARO clave=consumidor_poblacion actual=fuente_divergente esperado=AUT24_exacta' USING ERRCODE='55000'; END IF;
  nueva:=pg_catalog.replace(original,'vec_autorizacion.administradores_efectivos_internos_v1()','vec_autorizacion.administradores_aplicacion_efectivos_internos_v3()');
  IF item.firma='vec_autorizacion.ejecutar_cambio_admin_interno_v1(jsonb,text,text,text,text,boolean)' THEN
   IF pg_catalog.length(nueva)-pg_catalog.length(pg_catalog.replace(nueva,dup,''))<>pg_catalog.length(dup)
   THEN RAISE EXCEPTION 'AUT37: PARO clave=guarda_normal actual=marca_divergente esperado=persona_administrativa_unica' USING ERRCODE='55000'; END IF;
   nueva:=pg_catalog.replace(nueva,dup,dup_nueva);
  END IF;
  EXECUTE nueva;
  IF (SELECT pg_catalog.to_jsonb(p)-'prosrc' FROM pg_catalog.pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
  THEN RAISE EXCEPTION 'AUT37: PARO clave=metadata_funcion actual=alterada esperado=OID_firma_ACL_config_originales' USING ERRCODE='55000'; END IF;
 END LOOP;
END $poblacion_consumidores$;

CREATE FUNCTION vec_autorizacion.cotejar_roles_bootstrap_central_admin_v3(p jsonb)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' AS $f$
DECLARE g jsonb;r record;ct record;meta record;clase record;esperado jsonb;sys jsonb;configuracion jsonb;
 resultado jsonb:='[]'::jsonb;dimensiones jsonb;anterior text:='';aplicaciones integer:=0;sistemas integer:=0;
BEGIN
 IF pg_catalog.jsonb_array_length(p#>'{gobierno,roles}')<>2
 THEN RAISE EXCEPTION 'AUT37: PARO clave=roles actual=cardinalidad_divergente esperado=dos_categorias_del_kit' USING ERRCODE='22023'; END IF;
 SELECT s.value INTO STRICT sys FROM pg_catalog.jsonb_array_elements(p->'personas') pe,
  LATERAL pg_catalog.jsonb_array_elements(pe.value->'sistemas') s;
 FOR g IN SELECT e.value FROM pg_catalog.jsonb_array_elements(p#>'{gobierno,roles}') WITH ORDINALITY e(value,n) ORDER BY n LOOP
  IF (g->>'version_ref') COLLATE "C"<=anterior COLLATE "C" OR g->>'clase' IS DISTINCT FROM 'administrador'
  OR g->>'categoria_admin' NOT IN('aplicacion','sistemas')
  THEN RAISE EXCEPTION 'AUT37: PARO clave=roles_orden actual=divergente esperado=versiones_categorias_exactas' USING ERRCODE='22023'; END IF;
  anterior:=g->>'version_ref';
  esperado:=CASE g->>'categoria_admin' WHEN 'aplicacion' THEN p->'rol' ELSE sys->'rol' END;
  IF g->>'version_ref' IS DISTINCT FROM esperado->>'version_ref' OR g->>'huella_sha256' IS DISTINCT FROM esperado->>'huella_sha256'
  THEN RAISE EXCEPTION 'AUT37: PARO clave=rol_referenciado actual=divergente esperado=publicacion_usada_por_kit' USING ERRCODE='42501'; END IF;
  SELECT x.* INTO STRICT r FROM vec_autorizacion.version_rol x WHERE version_rol_ref=g->>'version_ref' FOR SHARE;
  SELECT x.* INTO STRICT ct FROM vec_autorizacion.control_vigencia_version_rol_actual a
   JOIN vec_autorizacion.control_vigencia_version_rol x USING(version_rol_ref,revision)
   WHERE a.version_rol_ref=r.version_rol_ref FOR SHARE OF a;
  SELECT x.* INTO STRICT meta FROM vec_autorizacion.perfil_fijo_categoria_nominal_v1 x WHERE version_rol_ref=r.version_rol_ref FOR SHARE;
  SELECT x.* INTO STRICT clase FROM vec_autorizacion.rol_sensible_exacto x WHERE version_rol_ref=r.version_rol_ref FOR SHARE;
  IF r.rol_id IS DISTINCT FROM g->>'rol_id' OR r.huella_sha256 IS DISTINCT FROM g->>'huella_sha256'
  OR r.documento->>'estado'<>'publicada'
  OR r.huella_sha256 IS DISTINCT FROM pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(vec_autorizacion.canon_version_rol_admin_v1(r.documento),'UTF8')),'hex')
  OR ct.estado<>'habilitada' OR ct.revision::text IS DISTINCT FROM esperado->>'control_revision'
  OR ct.huella_sha256 IS DISTINCT FROM esperado->>'control_huella_sha256'
  OR ct.huella_sha256 IS DISTINCT FROM pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(vec_autorizacion.canon_control_rol_admin_v1(ct.documento),'UTF8')),'hex')
  OR clase.clase<>'administrador' OR clase.huella_sha256 IS DISTINCT FROM r.huella_sha256
  OR meta.tipo_perfil<>'fijo_sistema' OR meta.categoria_administrativa IS DISTINCT FROM g->>'categoria_admin'
  OR meta.version_rol_huella_sha256 IS DISTINCT FROM r.huella_sha256
  OR meta.fuente_ref IS DISTINCT FROM g#>>'{fuente_categoria,referencia}'
  OR meta.fuente_version::text IS DISTINCT FROM g#>>'{fuente_categoria,version}'
  OR meta.fuente_huella_sha256 IS DISTINCT FROM g#>>'{fuente_categoria,huella_sha256}'
  OR (g->>'vigente_desde')::timestamptz>(p->>'preparado_en')::timestamptz
  OR (g->>'vigente_hasta')::timestamptz<(p->>'caduca_en')::timestamptz
  OR (g->>'duracion_propuesta_segundos')::numeric>EXTRACT(epoch FROM ((g->>'vigente_hasta')::timestamptz-(p->>'preparado_en')::timestamptz))
  THEN RAISE EXCEPTION 'AUT37: PARO clave=rol_fuente actual=no_acreditado esperado=version_control_categoria_vigentes' USING ERRCODE='42501'; END IF;
  SELECT pg_catalog.jsonb_agg(d ORDER BY d COLLATE "C") INTO dimensiones FROM (
   SELECT DISTINCT e.value AS d FROM vec_autorizacion.catalogo_accion_nominal_v1 a,
    LATERAL pg_catalog.jsonb_array_elements_text(a.dimensiones_ambito) e
   WHERE a.version_rol_ref=r.version_rol_ref AND a.fuente_ref=meta.fuente_ref
    AND a.fuente_version=meta.fuente_version AND a.fuente_huella_sha256=meta.fuente_huella_sha256
    AND a.clase_control=CASE meta.categoria_administrativa WHEN 'aplicacion' THEN 'administrador_aplicacion' ELSE 'administrador_sistemas' END
    AND EXISTS(SELECT 1 FROM pg_catalog.jsonb_array_elements(r.documento->'concesiones') c WHERE c.value=a.concesion)
    AND pg_catalog.clock_timestamp()>=a.vigente_desde
    AND (a.vigente_hasta IS NULL OR pg_catalog.clock_timestamp()<a.vigente_hasta)
  ) publicadas;
  IF dimensiones IS DISTINCT FROM g->'dimensiones_ambito'
  OR dimensiones NOT IN('["organizacion_ref"]'::jsonb,'["organizacion_ref","unidad_ref"]'::jsonb)
  OR ((g->>'unidad_requerida')::boolean AND dimensiones<>'["organizacion_ref","unidad_ref"]'::jsonb)
  OR vec_autorizacion.ambitos_positivos_validos(pg_catalog.jsonb_build_object('ambitos',g->'ambitos_fijos')) IS NOT TRUE
  THEN RAISE EXCEPTION 'AUT37: PARO clave=dimensiones actual=divergentes esperado=catalogo_publicado_y_limites_positivos' USING ERRCODE='42501'; END IF;
  IF g->>'categoria_admin'='aplicacion' THEN aplicaciones:=aplicaciones+1; ELSE sistemas:=sistemas+1; END IF;
  SELECT pg_catalog.to_jsonb(x) INTO configuracion FROM vec_autorizacion.rol_administrable_exacto_v1 x WHERE version_rol_ref=r.version_rol_ref FOR SHARE;
  resultado:=resultado||pg_catalog.jsonb_build_array(pg_catalog.jsonb_build_object('rol',pg_catalog.to_jsonb(r),
    'control',pg_catalog.to_jsonb(ct),'categoria',pg_catalog.to_jsonb(meta),'clase',pg_catalog.to_jsonb(clase),'dimensiones',dimensiones,'configuracion_administrable',configuracion));
 END LOOP;
 IF aplicaciones<>1 OR sistemas<>1
 THEN RAISE EXCEPTION 'AUT37: PARO clave=categorias actual=divergentes esperado=aplicacion_y_sistemas_separadas' USING ERRCODE='42501'; END IF;
 RETURN resultado;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.cotejar_roles_bootstrap_central_admin_v3(jsonb) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.preimagen_bootstrap_central_admin_v3(p jsonb)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' AS $f$
DECLARE persona jsonb;ca jsonb;identidad jsonb;roles jsonb;ct record;personas jsonb:='[]'::jsonb;
 esquema jsonb;hash text;sistema jsonb;gob jsonb;ambitos jsonb;fuentes_ambito jsonb;fuentes_sistemas jsonb;
BEGIN
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:admin:continuidad:v1',0));
 SELECT * INTO STRICT ct FROM vec_autorizacion.control_continuidad_admin WHERE control_id FOR UPDATE;
 IF ct.bootstrap_estado IS DISTINCT FROM p->>'bootstrap_estado_esperado'
 OR ct.revision::text IS DISTINCT FROM p->>'control_continuidad_revision_esperada'
 OR ct.bootstrap_estado<>'pendiente' OR ct.bootstrap_minimo_personas<>2 OR ct.minimo_personas<>1
 OR ct.bootstrap_acto_ref IS NOT NULL
 OR pg_catalog.jsonb_array_length(vec_autorizacion.administradores_aplicacion_efectivos_internos_v3())<>0
 THEN RAISE EXCEPTION 'AUT37: PARO clave=continuidad actual=divergente esperado=bootstrap2_pendiente_sin_admin' USING ERRCODE='40001'; END IF;
 roles:=vec_autorizacion.cotejar_roles_bootstrap_central_admin_v3(p);
 FOR persona IN SELECT value FROM pg_catalog.jsonb_array_elements(p->'personas') LOOP
  ca:=vec_contexto_actor_v1.preimagen_persona_bootstrap_central_admin_v3(persona);
  hash:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(ca::text,'UTF8')),'hex');
  IF hash IS DISTINCT FROM persona->>'preimagen_huella_sha256'
  THEN RAISE EXCEPTION 'AUT37: PARO clave=preimagen_persona actual=divergente esperado=huella_propietaria_exacta' USING ERRCODE='40001'; END IF;
  identidad:=vec_identidad_sesiones_v1.preimagen_certificado_bootstrap_central_admin_v3(persona,p->'gobierno');
  fuentes_ambito:=vec_autorizacion.cotejar_ambitos_bootstrap_central_admin_v3(persona->'ambitos',(persona->>'vigente_hasta')::timestamptz);
  fuentes_sistemas:='[]'::jsonb;
  SELECT value INTO STRICT gob FROM pg_catalog.jsonb_array_elements(p#>'{gobierno,roles}') WHERE value->>'version_ref'=p#>>'{rol,version_ref}';
  IF (persona->>'vigente_hasta')::timestamptz>(gob->>'vigente_hasta')::timestamptz
  OR fuentes_ambito->'ambitos' IS DISTINCT FROM ca->'ambitos'
  OR vec_autorizacion.ambitos_limite_bootstrap_central_admin_v3(ca->'ambitos',gob) IS NOT TRUE
  OR ((gob->>'unidad_requerida')::boolean AND NOT EXISTS(SELECT 1 FROM pg_catalog.jsonb_array_elements(ca->'ambitos') a(valor) WHERE a.valor->>'clave'='unidad_ref'))
  THEN RAISE EXCEPTION 'AUT37: PARO clave=ambitos_aplicacion actual=divergentes esperado=limites_gobernados' USING ERRCODE='42501'; END IF;
  FOR sistema IN SELECT value FROM pg_catalog.jsonb_array_elements(persona->'sistemas') LOOP
   SELECT value INTO STRICT gob FROM pg_catalog.jsonb_array_elements(p#>'{gobierno,roles}') WHERE value->>'version_ref'=sistema#>>'{rol,version_ref}';
   esquema:=vec_autorizacion.cotejar_ambitos_bootstrap_central_admin_v3(sistema->'ambitos',(sistema->>'vigente_hasta')::timestamptz);
   ambitos:=esquema->'ambitos';
   fuentes_sistemas:=fuentes_sistemas||pg_catalog.jsonb_build_array(esquema);
   IF (sistema->>'vigente_hasta')::timestamptz>(gob->>'vigente_hasta')::timestamptz
   OR vec_autorizacion.ambitos_limite_bootstrap_central_admin_v3(ambitos,gob) IS NOT TRUE
   OR ((gob->>'unidad_requerida')::boolean AND NOT EXISTS(SELECT 1 FROM pg_catalog.jsonb_array_elements(ambitos) a(valor) WHERE a.valor->>'clave'='unidad_ref'))
   THEN RAISE EXCEPTION 'AUT37: PARO clave=ambitos_sistemas actual=divergentes esperado=limites_gobernados' USING ERRCODE='42501'; END IF;
  END LOOP;
  IF EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil WHERE perfil_activo_ref=persona->>'perfil_ref')
  OR EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil a,pg_catalog.jsonb_array_elements(persona->'sistemas') s WHERE a.perfil_activo_ref=s->>'perfil_ref')
  THEN RAISE EXCEPTION 'AUT37: PARO clave=asignaciones actual=referencia_usada esperado=tres_referencias_nuevas' USING ERRCODE='40001'; END IF;
  personas:=personas||pg_catalog.jsonb_build_array(pg_catalog.jsonb_build_object('contexto',ca,'identidad',identidad,'fuentes_ambito',fuentes_ambito,'fuentes_sistemas',fuentes_sistemas));
 END LOOP;
 RETURN pg_catalog.jsonb_build_object('esquema','vec.admin.bootstrap.central.preimagen.v3','continuidad',pg_catalog.to_jsonb(ct),'roles',roles,'personas',personas);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.preimagen_bootstrap_central_admin_v3(jsonb) FROM PUBLIC;


CREATE FUNCTION vec_autorizacion.crear_asignacion_bootstrap_central_admin_v3(
 p_persona jsonb,p_perfil text,p_vinculo text,p_rol text,p_ambitos jsonb,p_hasta timestamptz,
 p_plan_sha256 text,p_auditoria text,p_acto text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' AS $f$
DECLARE ahora timestamptz(6):=pg_catalog.clock_timestamp();id text;ref text;doc jsonb;
BEGIN
 -- Fachada categorial privada: las tres preimágenes se cotejan antes de llamarla.
 -- Conserva intacta la prohibición genérica AUT24 de dos administradores normales.
 IF vec_autorizacion.ambitos_positivos_validos(pg_catalog.jsonb_build_object('ambitos',p_ambitos)) IS NOT TRUE
 OR NOT EXISTS(SELECT 1 FROM vec_autorizacion.perfil_fijo_categoria_nominal_v1 m
  JOIN vec_autorizacion.version_rol r USING(version_rol_ref)
  WHERE m.version_rol_ref=p_rol AND m.version_rol_huella_sha256=r.huella_sha256
    AND m.categoria_administrativa IN('aplicacion','sistemas') AND m.tipo_perfil='fijo_sistema')
 THEN RAISE EXCEPTION 'AUT37: PARO clave=efecto_categoria actual=incompatible esperado=perfil_fijo_gobernado' USING ERRCODE='42501'; END IF;
 PERFORM vec_contexto_actor_v1.crear_perfil_vinculo_admin_v1(p_persona->>'cuenta_ref',p_persona->>'persona_ref',
  (p_persona->>'cuenta_version')::numeric,(p_persona->>'persona_version')::numeric,p_perfil,p_vinculo,
  p_persona#>>'{procedencia,referencia}',(p_persona#>>'{procedencia,version}')::numeric,
  p_persona#>>'{procedencia,huella_sha256}',p_hasta);
 id:='bootstrap_'||pg_catalog.substr(pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p_plan_sha256||':'||p_perfil,'UTF8')),'hex'),1,32);
 ref:='asignacion:'||id||':v1';
 doc:=pg_catalog.jsonb_build_object('asignacion_id',id,'version',1,'perfil_activo_ref',p_perfil,
  'principal_id',p_persona->>'persona_ref','version_rol_ref',p_rol,'estado','activa','ambitos',p_ambitos,
  'vigente_desde',pg_catalog.to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'vigente_hasta',pg_catalog.to_char(p_hasta AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'emitida_por','bootstrap_operador:'||session_user::text,
  'emitida_en',pg_catalog.to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'revocada_en','0001-01-01T00:00:00Z');
 INSERT INTO vec_autorizacion.asignacion_perfil(asignacion_ref,asignacion_id,version,perfil_activo_ref,principal_id,version_rol_ref,huella_sha256,emitida_en,documento)
 VALUES(ref,id,1,p_perfil,p_persona->>'persona_ref',p_rol,
  pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(vec_autorizacion.canon_asignacion_perfil_admin_v1(doc),'UTF8')),'hex'),ahora,doc);
 INSERT INTO vec_autorizacion.sello_efecto_admin_tx_v1(asignacion_ref,transaccion,operacion_ref,auditoria_ref)
 VALUES(ref,pg_catalog.txid_current(),p_acto,p_auditoria);
 INSERT INTO vec_autorizacion.asignacion_perfil_actual(perfil_activo_ref,asignacion_ref,actualizada_en,actualizada_por,acto_ref)
 VALUES(p_perfil,ref,ahora,'bootstrap_operador:'||session_user::text,p_acto);
 RETURN pg_catalog.jsonb_build_object('persona_ref',p_persona->>'persona_ref','cuenta_ref',p_persona->>'cuenta_ref',
  'perfil_ref',p_perfil,'vinculo_ref',p_vinculo,'asignacion_ref',ref,'rol_version_ref',p_rol);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.crear_asignacion_bootstrap_central_admin_v3(jsonb,text,text,text,jsonb,timestamptz,text,text,text) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.registrar_bootstrap_central_admin_v3(p_plan_canonico text,p_huella_aprobada text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC'
SET row_security=on SET lock_timeout='2s' SET statement_timeout='30s' AS $f$
DECLARE cfg vec_autorizacion.config_bootstrap_central_admin_v3;p jsonb;sha text;previo record;
 preimagen jsonb;preimagen_bytes bytea;preimagen_sha text;persona jsonb;sistema jsonb;gob jsonb;config_rol record;
 perfiles text[]:=ARRAY[]::text[];vinculos text[]:=ARRAY[]::text[];sys_count integer:=0;hash_campo record;
 acto text;recibo text;cert_acto text;aud record;resultados jsonb:='[]'::jsonb;resultado jsonb;
 ambitos jsonb;deadline timestamptz;ahora timestamptz(6);i integer:=0;
BEGIN
 cfg:=vec_autorizacion.exigir_operador_bootstrap_central_admin_v3();
 IF p_plan_canonico IS NULL OR pg_catalog.octet_length(p_plan_canonico) NOT BETWEEN 1 AND 65536
 OR p_huella_aprobada IS NULL OR p_huella_aprobada !~ '^[0-9a-f]{64}$'
 THEN RAISE EXCEPTION 'AUT37: PARO clave=plan actual=invalido esperado=canon_V3_acotado_aprobado' USING ERRCODE='22023'; END IF;
 p:=p_plan_canonico::jsonb;
 IF vec_autorizacion.canon_bootstrap_central_admin_v3(p,'plan') IS DISTINCT FROM p_plan_canonico
 OR p->>'version' IS DISTINCT FROM '3' OR p->>'bootstrap_estado_esperado' IS DISTINCT FROM 'pendiente'
 OR pg_catalog.jsonb_array_length(p->'personas')<>2
 THEN RAISE EXCEPTION 'AUT37: PARO clave=plan_canon actual=divergente esperado=ABI_V3_dos_personas' USING ERRCODE='22023'; END IF;
 sha:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p_plan_canonico,'UTF8')),'hex');
 IF sha IS DISTINCT FROM p_huella_aprobada OR sha IS DISTINCT FROM cfg.plan_sha256
 OR p->'fuente_reparto_aprobado' IS DISTINCT FROM cfg.fuente_reparto
 OR p->'fuente_identidad' IS DISTINCT FROM cfg.fuente_identidad
 OR p->'fuente_ca_admin' IS DISTINCT FROM cfg.fuente_ca_admin
 THEN RAISE EXCEPTION 'AUT37: PARO clave=aprobacion actual=no_coincidente esperado=plan_fuentes_aprobadas_del_LOGIN' USING ERRCODE='42501'; END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:admin:continuidad:v1',0));
 SELECT * INTO previo FROM vec_autorizacion.bootstrap_central_admin_v3 WHERE singleton FOR SHARE;
 IF FOUND THEN
  IF previo.huella_plan_sha256 IS DISTINCT FROM sha OR previo.plan_canonico IS DISTINCT FROM pg_catalog.convert_to(p_plan_canonico,'UTF8')
  OR previo.operador_login IS DISTINCT FROM session_user::name OR previo.aprobacion_ref IS DISTINCT FROM cfg.aprobacion_ref
  OR previo.aprobacion_sha256 IS DISTINCT FROM cfg.aprobacion_sha256
  OR vec_autorizacion_atestada_v3.cotejar_acuse_bootstrap_central_admin_v3(previo.auditoria_ref,previo.auditoria_secuencia,previo.auditoria_huella_sha256,previo.auditoria_registrada_en,sha,previo.operador_login,previo.aprobacion_ref,previo.preimagen_sha256) IS NOT TRUE
  OR NOT EXISTS(SELECT 1 FROM vec_autorizacion.control_continuidad_admin
    WHERE control_id AND bootstrap_estado='consumido' AND bootstrap_acto_ref=previo.acto_ref)
  THEN RAISE EXCEPTION 'AUT37: PARO clave=replay actual=incompatible esperado=operador_aprobacion_plan_originales' USING ERRCODE='23505'; END IF;
  PERFORM vec_autorizacion.exigir_operador_bootstrap_central_admin_v3();
  RETURN previo.resultado;
 END IF;
 deadline:=LEAST(cfg.vigente_hasta,(p->>'caduca_en')::timestamptz);
 IF (p->>'preparado_en')::timestamptz>pg_catalog.clock_timestamp()
 OR (p->>'caduca_en')::timestamptz<=(p->>'preparado_en')::timestamptz OR pg_catalog.clock_timestamp()>=deadline
 OR (p#>>'{personas,0,persona_ref}') COLLATE "C">=(p#>>'{personas,1,persona_ref}') COLLATE "C"
 OR p#>>'{personas,0,cuenta_ref}'=p#>>'{personas,1,cuenta_ref}'
 OR p#>>'{personas,0,certificado_admin,huella_sha256}'=p#>>'{personas,1,certificado_admin,huella_sha256}'
 THEN RAISE EXCEPTION 'AUT37: PARO clave=reparto actual=divergente esperado=dos_personas_independientes_y_vigencia' USING ERRCODE='22023'; END IF;
 FOR hash_campo IN SELECT e.key,e.value FROM pg_catalog.jsonb_path_query(p,'$.** ? (@.type() == "object")') o(objeto),
   LATERAL pg_catalog.jsonb_each(o.objeto) e
  WHERE e.key IN('huella_sha256','ca_huella_sha256','preimagen_huella_sha256','control_huella_sha256') LOOP
  IF pg_catalog.jsonb_typeof(hash_campo.value) IS DISTINCT FROM 'string'
  OR hash_campo.value#>>'{}' !~ '^[0-9a-f]{64}$' OR hash_campo.value#>>'{}'=pg_catalog.repeat('0',64)
  THEN RAISE EXCEPTION 'AUT37: PARO clave=huella_fuente actual=invalida esperado=SHA256_positivo' USING ERRCODE='22023'; END IF;
 END LOOP;
 FOR persona IN SELECT value FROM pg_catalog.jsonb_array_elements(p->'personas') LOOP
  IF persona->>'persona_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$' OR persona->>'cuenta_ref' !~ '^cta_[A-Za-z0-9_-]{22,128}$'
  OR persona->>'perfil_ref' !~ '^prf_[A-Za-z0-9_-]{22,128}$' OR persona->>'vinculo_ref' !~ '^vca_[A-Za-z0-9_-]{22,128}$'
  OR persona->>'perfil_ref'=ANY(perfiles) OR persona->>'vinculo_ref'=ANY(vinculos)
  OR persona#>>'{certificado_admin,persona_ref}' IS DISTINCT FROM persona->>'persona_ref'
  OR persona#>>'{certificado_admin,cuenta_ref}' IS DISTINCT FROM persona->>'cuenta_ref'
  OR persona#>>'{certificado_admin,ca_huella_sha256}' IS DISTINCT FROM p#>>'{fuente_ca_admin,huella_sha256}'
  OR (persona->>'vigente_hasta')::timestamptz<(p->>'caduca_en')::timestamptz
  THEN RAISE EXCEPTION 'AUT37: PARO clave=persona actual=invalida esperado=identidad_refs_certificado_coherentes' USING ERRCODE='22023'; END IF;
  perfiles:=pg_catalog.array_append(perfiles,persona->>'perfil_ref');vinculos:=pg_catalog.array_append(vinculos,persona->>'vinculo_ref');
 END LOOP;
 FOR persona IN SELECT value FROM pg_catalog.jsonb_array_elements(p->'personas') LOOP
  FOR sistema IN SELECT value FROM pg_catalog.jsonb_array_elements(persona->'sistemas') LOOP
   sys_count:=sys_count+1;
   IF sistema->>'perfil_ref' !~ '^prf_[A-Za-z0-9_-]{22,128}$' OR sistema->>'vinculo_ref' !~ '^vca_[A-Za-z0-9_-]{22,128}$'
   OR sistema->>'perfil_ref'=ANY(perfiles) OR sistema->>'vinculo_ref'=ANY(vinculos)
   OR sistema#>>'{rol,version_ref}' IS NOT DISTINCT FROM p#>>'{rol,version_ref}'
   OR (sistema->>'vigente_hasta')::timestamptz<(p->>'caduca_en')::timestamptz
   OR (sistema->>'vigente_hasta')::timestamptz>(persona->>'vigente_hasta')::timestamptz
   THEN RAISE EXCEPTION 'AUT37: PARO clave=sistemas actual=invalido esperado=perfil_separado_una_misma_persona' USING ERRCODE='22023'; END IF;
   perfiles:=pg_catalog.array_append(perfiles,sistema->>'perfil_ref');vinculos:=pg_catalog.array_append(vinculos,sistema->>'vinculo_ref');
  END LOOP;
 END LOOP;
 IF sys_count<>1
 THEN RAISE EXCEPTION 'AUT37: PARO clave=kit actual=cardinalidad_divergente esperado=dos_Aplicacion_un_Sistemas' USING ERRCODE='22023'; END IF;
 -- Todos los roles, fuentes y tres objetivos se acreditan antes del primer INSERT.
 preimagen:=vec_autorizacion.preimagen_bootstrap_central_admin_v3(p);
 preimagen_bytes:=pg_catalog.convert_to(preimagen::text,'UTF8');
 preimagen_sha:=pg_catalog.encode(pg_catalog.sha256(preimagen_bytes),'hex');
 IF preimagen_sha IS DISTINCT FROM cfg.preimagen_sha256
 THEN RAISE EXCEPTION 'AUT37: PARO clave=preimagen_aprobada actual=divergente esperado=estado_completo_aprobado' USING ERRCODE='40001'; END IF;
 -- Gobernanza existente se coteja; nunca se reemplaza ni alarga su historia.
 FOR gob IN SELECT value FROM pg_catalog.jsonb_array_elements(p#>'{gobierno,roles}') LOOP
  SELECT * INTO config_rol FROM vec_autorizacion.rol_administrable_exacto_v1 WHERE version_rol_ref=gob->>'version_ref' FOR SHARE;
  IF FOUND AND (config_rol.clase IS DISTINCT FROM gob->>'clase' OR config_rol.huella_sha256 IS DISTINCT FROM gob->>'huella_sha256'
   OR config_rol.vigente_desde IS DISTINCT FROM (gob->>'vigente_desde')::timestamptz
   OR config_rol.vigente_hasta IS DISTINCT FROM (gob->>'vigente_hasta')::timestamptz
   OR config_rol.unidad_requerida IS DISTINCT FROM (gob->>'unidad_requerida')::boolean
   OR config_rol.audiencia_administrativa IS DISTINCT FROM p#>>'{gobierno,audiencia_administrativa}'
   OR config_rol.ambitos_fijos IS DISTINCT FROM gob->'ambitos_fijos'
   OR config_rol.duracion_propuesta IS DISTINCT FROM pg_catalog.make_interval(secs=>(gob->>'duracion_propuesta_segundos')::double precision))
  THEN RAISE EXCEPTION 'AUT37: PARO clave=gobierno_existente actual=divergente esperado=preimagen_exacta_sin_sustitucion' USING ERRCODE='40001'; END IF;
 END LOOP;
 acto:='acto_admin:'||pg_catalog.substr(sha,1,32);recibo:='recibo_bootstrap:'||pg_catalog.substr(sha,33,32);
 SELECT * INTO STRICT aud FROM vec_autorizacion_atestada_v3.registrar_evento_admin_preperfil_v1(pg_catalog.jsonb_build_object(
  'tipo_registro','bootstrap_operador','evento_ref','evento_'||pg_catalog.substr(sha,1,32),'operador_login',session_user::text,
  'plan_sha256',sha,'aprobacion_ref',cfg.aprobacion_ref,'accion','ejecutar_plan_bootstrap_admin',
  'recurso_ref',acto,'resultado','permitido','motivo_ref','admin.bootstrap.plan_aprobado','proceso',cfg.proceso,
  'canal','operacion_tecnica_privada','finalidad_ref','bootstrap_admin','correlacion_ref','correlacion_'||pg_catalog.substr(sha,33,32),
  'fuente_ref','bootstrap_preimagen:'||pg_catalog.substr(sha,1,32),'fuente_sha256',preimagen_sha));
 FOR gob IN SELECT value FROM pg_catalog.jsonb_array_elements(p#>'{gobierno,roles}') LOOP
  IF NOT EXISTS(SELECT 1 FROM vec_autorizacion.rol_administrable_exacto_v1 WHERE version_rol_ref=gob->>'version_ref') THEN
   INSERT INTO vec_autorizacion.rol_administrable_exacto_v1(version_rol_ref,clase,huella_sha256,vigente_desde,vigente_hasta,unidad_requerida,audiencia_administrativa,ambitos_fijos,duracion_propuesta)
   VALUES(gob->>'version_ref',gob->>'clase',gob->>'huella_sha256',(gob->>'vigente_desde')::timestamptz,(gob->>'vigente_hasta')::timestamptz,
    (gob->>'unidad_requerida')::boolean,p#>>'{gobierno,audiencia_administrativa}',gob->'ambitos_fijos',pg_catalog.make_interval(secs=>(gob->>'duracion_propuesta_segundos')::double precision));
  END IF;
 END LOOP;
 FOR persona IN SELECT value FROM pg_catalog.jsonb_array_elements(p->'personas') LOOP
  i:=i+1;
  cert_acto:='acto_admin:'||pg_catalog.substr(pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(sha||':cert:'||i::text,'UTF8')),'hex'),1,32);
  PERFORM vec_identidad_sesiones_v1.crear_certificado_bootstrap_admin_interno_v2(persona,p->'gobierno',cert_acto);
  ambitos:=(vec_autorizacion.cotejar_ambitos_bootstrap_central_admin_v3(persona->'ambitos',(persona->>'vigente_hasta')::timestamptz))->'ambitos';
  resultados:=resultados||pg_catalog.jsonb_build_array(vec_autorizacion.crear_asignacion_bootstrap_central_admin_v3(persona,persona->>'perfil_ref',persona->>'vinculo_ref',p#>>'{rol,version_ref}',ambitos,(persona->>'vigente_hasta')::timestamptz,sha,aud.auditoria_ref,acto));
  FOR sistema IN SELECT value FROM pg_catalog.jsonb_array_elements(persona->'sistemas') LOOP
   ambitos:=(vec_autorizacion.cotejar_ambitos_bootstrap_central_admin_v3(sistema->'ambitos',(sistema->>'vigente_hasta')::timestamptz))->'ambitos';
   resultados:=resultados||pg_catalog.jsonb_build_array(vec_autorizacion.crear_asignacion_bootstrap_central_admin_v3(persona,sistema->>'perfil_ref',sistema->>'vinculo_ref',sistema#>>'{rol,version_ref}',ambitos,(sistema->>'vigente_hasta')::timestamptz,sha,aud.auditoria_ref,acto));
  END LOOP;
 END LOOP;
 ahora:=pg_catalog.clock_timestamp();
 UPDATE vec_autorizacion.control_continuidad_admin SET revision=revision+1,bootstrap_estado='consumido',bootstrap_acto_ref=acto,actualizado_en=ahora
 WHERE control_id AND revision=(p->>'control_continuidad_revision_esperada')::bigint AND bootstrap_estado='pendiente';
 IF NOT FOUND OR (SELECT pg_catalog.count(DISTINCT e.value->>'persona_ref') FROM pg_catalog.jsonb_array_elements(vec_autorizacion.administradores_aplicacion_efectivos_internos_v3()) e)<>2
 THEN RAISE EXCEPTION 'AUT37: PARO clave=CAS_final actual=divergente esperado=dos_personas_y_un_consumo' USING ERRCODE='40001'; END IF;
 PERFORM vec_autorizacion.exigir_operador_bootstrap_central_admin_v3();
 IF pg_catalog.clock_timestamp()>=deadline
 THEN RAISE EXCEPTION 'AUT37: PARO clave=vigencia_final actual=caducada esperado=plan_configuracion_vigentes' USING ERRCODE='42501'; END IF;
 resultado:=pg_catalog.jsonb_build_object('acto_ref',acto,'recibo_ref',recibo,'huella_plan_sha256',sha,
  'primera_persona_ref',p#>>'{personas,0,persona_ref}','segunda_persona_ref',p#>>'{personas,1,persona_ref}',
  'auditoria_ref',aud.auditoria_ref,'perfiles',resultados,'confirmado_en',ahora);
 INSERT INTO vec_autorizacion.bootstrap_central_admin_v3(singleton,acto_ref,recibo_ref,plan_canonico,huella_plan_sha256,preimagen_canonica,preimagen_sha256,operador_login,aprobacion_ref,aprobacion_sha256,auditoria_ref,auditoria_secuencia,auditoria_huella_sha256,auditoria_registrada_en,resultado,confirmado_en)
 VALUES(true,acto,recibo,pg_catalog.convert_to(p_plan_canonico,'UTF8'),sha,preimagen_bytes,preimagen_sha,session_user::name,cfg.aprobacion_ref,cfg.aprobacion_sha256,aud.auditoria_ref,aud.secuencia,aud.huella_sha256,aud.registrada_en,resultado,ahora);
 INSERT INTO vec_autorizacion.outbox_bootstrap_central_admin_v3 VALUES(acto,'bootstrap_central_confirmado',aud.auditoria_ref,ahora);
 RETURN resultado;
EXCEPTION WHEN no_data_found OR too_many_rows THEN
 RAISE EXCEPTION 'AUT37: PARO clave=fuente actual=ausente_o_ambigua esperado=fuentes_propietarias_exactas' USING ERRCODE='42501';
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.registrar_bootstrap_central_admin_v3(text,text) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion TO vec_admin_bootstrap_central_v3_ejecutor;
GRANT EXECUTE ON FUNCTION vec_autorizacion.preflight_bootstrap_central_admin_v3(),vec_autorizacion.registrar_bootstrap_central_admin_v3(text,text)
 TO vec_admin_bootstrap_central_v3_ejecutor;
DO $acl$
DECLARE f record;t regclass;aut oid:='vec_autorizacion_propietario'::regrole;
 runtime oid:='vec_admin_bootstrap_central_v3_ejecutor'::regrole;permitidos oid[];
BEGIN
 FOR f IN SELECT p.oid,p.proname,p.proowner,p.proacl FROM pg_catalog.pg_proc p
 JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
 WHERE (n.nspname='vec_autorizacion' AND p.proname IN(
  'canon_bootstrap_central_admin_v3','exigir_operador_bootstrap_central_admin_v3',
  'preflight_bootstrap_central_admin_v3','cotejar_roles_bootstrap_central_admin_v3',
  'preimagen_bootstrap_central_admin_v3','crear_asignacion_bootstrap_central_admin_v3',
  'administradores_aplicacion_efectivos_internos_v3','cotejar_ambitos_bootstrap_central_admin_v3','ambitos_limite_bootstrap_central_admin_v3',
  'registrar_bootstrap_central_admin_v3'))
 OR (n.nspname='vec_contexto_actor_v1' AND p.proname IN('cotejar_ambitos_bootstrap_central_admin_v3','preimagen_persona_bootstrap_central_admin_v3'))
 OR (n.nspname='vec_identidad_sesiones_v1' AND p.proname='preimagen_certificado_bootstrap_central_admin_v3')
 OR (n.nspname='vec_autorizacion_atestada_v3' AND p.proname='cotejar_acuse_bootstrap_central_admin_v3') LOOP
  permitidos:=ARRAY[f.proowner];
  IF f.proowner<>aut THEN permitidos:=pg_catalog.array_append(permitidos,aut); END IF;
  IF f.proname IN('preflight_bootstrap_central_admin_v3','registrar_bootstrap_central_admin_v3') THEN permitidos:=pg_catalog.array_append(permitidos,runtime); END IF;
  IF EXISTS(SELECT 1 FROM pg_catalog.aclexplode(COALESCE(f.proacl,pg_catalog.acldefault('f',f.proowner))) a
    WHERE NOT a.grantee=ANY(permitidos) OR a.privilege_type<>'EXECUTE' OR a.is_grantable)
  THEN RAISE EXCEPTION 'AUT37: PARO clave=ACL_funcion actual=ampliada esperado=consumidores_propietarios_exactos' USING ERRCODE='55000'; END IF;
 END LOOP;
 FOREACH t IN ARRAY ARRAY['vec_autorizacion.config_bootstrap_central_admin_v3'::regclass,
  'vec_autorizacion.bootstrap_central_admin_v3'::regclass,'vec_autorizacion.outbox_bootstrap_central_admin_v3'::regclass] LOOP
  IF EXISTS(SELECT 1 FROM pg_catalog.pg_class c,
   LATERAL pg_catalog.aclexplode(COALESCE(c.relacl,pg_catalog.acldefault('r',c.relowner))) a
   WHERE c.oid=t AND (a.grantee<>aut OR a.is_grantable))
  THEN RAISE EXCEPTION 'AUT37: PARO clave=ACL_tabla actual=ampliada esperado=owner_exclusivo' USING ERRCODE='55000'; END IF;
 END LOOP;
END $acl$;
COMMIT;
