\set ON_ERROR_STOP on
-- AD214. Tres consultas RRHH de Bolsa sobre el núcleo posterior a
-- AD211/P22, AD175/P32 y AD180/P34. AD213 instala sólo su fachada después.
-- Preimagen exacta medida en PostgreSQL 18 sobre la copia HX causal.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000214',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;

DO $pre$
DECLARE rol text;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR current_user<>'vec_autorizacion_atestada_v3_propietario'
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.resolver_origen_consumo_v1(text,text,text)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.capacidad_cruda_prevalida(bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_ficha_propia_empleado_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_exportacion_servicios_propios_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_historia_servicios_propios_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_consulta_rrhh_bolsa_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR to_regclass('vec_autorizacion_atestada_v3.consumo_decision_v3') IS NULL
    OR to_regclass('vec_autorizacion_atestada_v3.auditoria_consumo_v3') IS NULL
 THEN RAISE EXCEPTION 'AD214: preimagen causal o fachada incompatible' USING ERRCODE='55000'; END IF;
 IF NOT EXISTS (SELECT 1 FROM pg_attribute WHERE attrelid='vec_autorizacion_atestada_v3.consumo_decision_v3'::regclass
     AND attname='transaccion_origen' AND atttypid='xid8'::regtype AND NOT attisdropped)
    OR NOT EXISTS (SELECT 1 FROM pg_attribute WHERE attrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
     AND attname='transaccion_origen' AND atttypid='xid8'::regtype AND NOT attisdropped)
 THEN RAISE EXCEPTION 'AD214: sello TopXID AD193 ausente' USING ERRCODE='55000'; END IF;
 FOREACH rol IN ARRAY ARRAY['vec_bolsa_llamamientos_propietario','vec_bolsa_llamamientos_ejecutor'] LOOP
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=rol AND NOT rolcanlogin
      AND NOT (rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls))
  THEN RAISE EXCEPTION 'AD214: rol técnico incompatible %',rol USING ERRCODE='55000'; END IF;
 END LOOP;
 IF EXISTS (SELECT 1 FROM pg_auth_members WHERE member='vec_bolsa_llamamientos_ejecutor'::regrole)
 THEN RAISE EXCEPTION 'AD214: herencia de grupo ejecutor incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

DO $nucleo$
DECLARE f oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 original text; fuente text; nuevo text; actual text; meta jsonb; deps jsonb; compartidas jsonb;
 esperado_def_sha256 text:='f56115da72b3b59f34f0d06adb1f4d92eb67de9f24ec945a1f634bebb78918d3'; esperado_src_sha256 text:='c10f2939c88567231a8c9eb31327f0780d2c9041d40b4d38f5e052dea0e7a32b';
 marca text:=$m$       )
       OR c ->> 'suite' <> 'VEC-AD-3-COSE-EDDSA-1'$m$;
 excl text:=$e$p_perfil_mutacion IS DISTINCT FROM 'bolsa_llamamiento'$e$;
 excl_nuevo text:=excl||' AND p_perfil_mutacion IS DISTINCT FROM ''consulta_rrhh_bolsa''';
 runtime text:=$r$           OR (p_perfil_mutacion IS NOT DISTINCT FROM 'servicios_certificados_propios'$r$;
 runtime_nuevo text:=$rn$           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_rrhh_bolsa'
               AND EXISTS (SELECT 1 FROM pg_roles r WHERE r.rolname=session_user AND r.rolcanlogin AND r.rolinherit
                 AND NOT (r.rolsuper OR r.rolcreatedb OR r.rolcreaterole OR r.rolreplication OR r.rolbypassrls)
                 AND r.rolconfig IS NULL)
               AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole
                 AND m.roleid='vec_bolsa_llamamientos_ejecutor'::regrole
                 AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
               AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1
               AND NOT EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member='vec_bolsa_llamamientos_ejecutor'::regrole)
           )
$rn$||runtime;
 extension text:=$x$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_rrhh_bolsa'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb
 AND (
   (c->>'operacion' IS NOT DISTINCT FROM 'bolsa.rrhh.bolsas.consultar'
    AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.rrhh.bolsas.consultar.v1'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'coleccion_bolsas_rrhh'
    AND d->>'recurso_ref' IS NOT DISTINCT FROM 'coleccion:bolsa:rrhh:bolsas'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'consulta_rrhh_bolsas'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["bolsas","conteos"]'::jsonb)
   OR (c->>'operacion' IS NOT DISTINCT FROM 'bolsa.rrhh.estadisticas.consultar'
    AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.rrhh.estadisticas.consultar.v1'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'estadisticas_bolsas_rrhh'
    AND d->>'recurso_ref' IS NOT DISTINCT FROM 'coleccion:bolsa:rrhh:estadisticas'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'consulta_rrhh_estadisticas'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["estadisticas"]'::jsonb)
   OR (c->>'operacion' IS NOT DISTINCT FROM 'bolsa.rrhh.candidatos.consultar'
    AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.rrhh.candidatos.consultar.v1'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'consulta_candidatos_bolsa'
    AND d->>'recurso_ref' ~ '^bolsa:[A-Za-z0-9:_-]+:filtro:[a-f0-9]{64}$'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'consulta_rrhh_candidatos'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["candidatos","contactos","turno"]'::jsonb)))
$x$;
BEGIN
 IF f IS NULL OR esperado_def_sha256 IS NULL OR esperado_src_sha256 IS NULL
 THEN RAISE EXCEPTION 'AD214: preimagen final sin medir' USING ERRCODE='55000'; END IF;
 SELECT pg_get_functiondef(f),p.prosrc,to_jsonb(p)-'prosrc' INTO STRICT original,fuente,meta FROM pg_proc p WHERE p.oid=f;
 IF encode(sha256(convert_to(original,'UTF8')),'hex') IS DISTINCT FROM esperado_def_sha256
    OR encode(sha256(convert_to(fuente,'UTF8')),'hex') IS DISTINCT FROM esperado_src_sha256
    OR NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=f
       AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef
       AND p.provolatile='v' AND p.proparallel='u'
       AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s'])
    OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL
        aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f)<>1
    OR NOT EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL
        aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f
        AND a.grantee=p.proowner AND a.grantor=p.proowner AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
    OR length(original)-length(replace(original,marca,''))<>length(marca)
    OR length(original)-length(replace(original,excl,''))<>length(excl)
    OR length(original)-length(replace(original,runtime,''))<>length(runtime)
    OR strpos(original,'consulta_rrhh_bolsa')<>0
    OR strpos(original,'resolver_origen_consumo_v1')=0
    OR strpos(original,'transaccion_origen')=0
    OR strpos(original,'consumo_confirmado_v4')=0
 THEN RAISE EXCEPTION 'AD214: núcleo, origen, familia v4 o marcas incompatibles' USING ERRCODE='55000'; END IF;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
   INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
   INTO compartidas FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database())
       AND d.classid='pg_proc'::regclass AND d.objid=f;
 nuevo:=replace(original,excl,excl_nuevo);
 nuevo:=replace(nuevo,runtime,runtime_nuevo);
 nuevo:=replace(nuevo,marca,extension||marca);
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo
    OR replace(replace(replace(actual,extension||marca,marca),runtime_nuevo,runtime),excl_nuevo,excl) IS DISTINCT FROM original
    OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
        FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
        FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database())
          AND d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM compartidas
 THEN RAISE EXCEPTION 'AD214: postimagen del núcleo divergente' USING ERRCODE='55000'; END IF;
END $nucleo$;

DO $audiencias$
DECLARE anterior text; nueva text; esperada_aud_sha256 text:='a390816c9facd6993e1c0deb016c1fa3571cf23bc2ca071cfff96d9e70454640'; audiencia text;
BEGIN
 IF esperada_aud_sha256 IS NULL THEN
  RAISE EXCEPTION 'AD214: CHECK de audiencias final sin medir' USING ERRCODE='55000'; END IF;
 SELECT pg_get_constraintdef(c.oid,false) INTO STRICT anterior FROM pg_constraint c
  WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
    AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 IF encode(sha256(convert_to(anterior,'UTF8')),'hex') IS DISTINCT FROM esperada_aud_sha256
    OR left(anterior,7)<>'CHECK (' OR right(anterior,1)<>')'
 THEN RAISE EXCEPTION 'AD214: CHECK de audiencias incompatible' USING ERRCODE='55000'; END IF;
 FOREACH audiencia IN ARRAY ARRAY[
  'vec_bolsa_llamamientos.rrhh.bolsas.consultar.v1',
  'vec_bolsa_llamamientos.rrhh.estadisticas.consultar.v1',
  'vec_bolsa_llamamientos.rrhh.candidatos.consultar.v1'] LOOP
  IF strpos(anterior,quote_literal(audiencia))<>0 THEN
   RAISE EXCEPTION 'AD214: audiencia ya presente %',audiencia USING ERRCODE='55000'; END IF;
 END LOOP;
 nueva:='CHECK (('||substr(anterior,8,length(anterior)-8)||') OR audiencia_consumo = ANY (ARRAY['||
  quote_literal('vec_bolsa_llamamientos.rrhh.bolsas.consultar.v1')||'::text,'||
  quote_literal('vec_bolsa_llamamientos.rrhh.estadisticas.consultar.v1')||'::text,'||
  quote_literal('vec_bolsa_llamamientos.rrhh.candidatos.consultar.v1')||'::text]))';
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||nueva;
 IF NOT EXISTS (SELECT 1 FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
    AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated)
 THEN RAISE EXCEPTION 'AD214: CHECK de audiencias no quedó validado' USING ERRCODE='55000'; END IF;
END $audiencias$;

COMMIT;
