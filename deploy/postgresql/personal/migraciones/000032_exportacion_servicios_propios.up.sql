\set ON_ERROR_STOP on
-- Personal32. Añade únicamente el corte a nuevos recibos de Personal22.
-- Los recibos históricos conservan NULL; no hay backfill ni tabla de exportaciones.
-- Depende de AD175 (posterior a AD172/173/174); no ensayada ni instalada.
BEGIN;
SET LOCAL ROLE vec_personal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal:migracion:000032',0));
DO $pre$
BEGIN
 IF current_user<>'vec_personal_propietario'
    OR to_regclass('vec_personal.recibo_ficha_propia_empleado') IS NULL
    OR to_regprocedure('vec_personal.exportar_servicios_propios_empleado_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_exportacion_servicios_propios_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR NOT has_function_privilege('vec_personal_propietario','vec_autorizacion_atestada_v3.consumir_exportacion_servicios_propios_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR EXISTS (SELECT 1 FROM pg_attribute a WHERE a.attrelid='vec_personal.recibo_ficha_propia_empleado'::regclass
      AND a.attname IN ('vigente_en','conocido_en') AND NOT a.attisdropped)
 THEN RAISE EXCEPTION 'Personal32: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
LOCK TABLE vec_personal.recibo_ficha_propia_empleado IN ACCESS EXCLUSIVE MODE;
ALTER TABLE vec_personal.recibo_ficha_propia_empleado
 ADD COLUMN vigente_en date,
 ADD COLUMN conocido_en timestamptz(6),
 ADD CONSTRAINT recibo_ficha_corte_completo CHECK(
  (vigente_en IS NULL AND conocido_en IS NULL)
  OR (vigente_en IS NOT NULL AND conocido_en IS NOT NULL AND isfinite(vigente_en) AND isfinite(conocido_en)));

-- Patch único sobre la función instalada: no cambia el material, su canon,
-- el recibo devuelto, las ACL ni las guardas anteriores.
DO $consulta$
DECLARE
 f oid:=to_regprocedure('vec_personal.consultar_ficha_propia_empleado_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 original text; nuevo text; actual text; meta jsonb; deps jsonb; compartidas jsonb;
 marca text:=$marca$  (recibo_ref,empleado_ref,material_sha256,decision_ref,auditoria_ref,consumo_huella_sha256,cardinalidad,consultada_en)
 VALUES(recibo,empleado,material_sha,consumo.decision_ref,consumo.auditoria_ref,consumo.consumo_huella_sha256,
   jsonb_array_length(relaciones)+jsonb_array_length(servicios),ahora);$marca$;
 sustitucion text:=$sustitucion$  (recibo_ref,empleado_ref,material_sha256,decision_ref,auditoria_ref,consumo_huella_sha256,cardinalidad,consultada_en,vigente_en,conocido_en)
 VALUES(recibo,empleado,material_sha,consumo.decision_ref,consumo.auditoria_ref,consumo.consumo_huella_sha256,
   jsonb_array_length(relaciones)+jsonb_array_length(servicios),ahora,fecha,conocido);$sustitucion$;
BEGIN
 IF f IS NULL THEN RAISE EXCEPTION 'Personal32: consulta ausente' USING ERRCODE='55000'; END IF;
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc' INTO STRICT original,meta FROM pg_proc p WHERE p.oid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
 INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
 INTO compartidas FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database())
  AND d.classid='pg_proc'::regclass AND d.objid=f;
 IF NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.proowner='vec_personal_propietario'::regrole
      AND p.prosecdef AND p.provolatile='v' AND p.proparallel='u'
      AND p.proconfig=ARRAY['search_path=pg_catalog','row_security=on','TimeZone=UTC','lock_timeout=2s']
      AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='2d4db33c46212e754e5a8805387d02e0acd70c6e7006c7378e9978a8886e0e11')
    OR length(original)-length(replace(original,marca,''))<>length(marca) THEN
  RAISE EXCEPTION 'Personal32: consulta anterior divergente' USING ERRCODE='55000'; END IF;
 nuevo:=replace(original,marca,sustitucion);
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo OR replace(actual,sustitucion,marca) IS DISTINCT FROM original
    OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
        FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
        FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database())
          AND d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM compartidas THEN
  RAISE EXCEPTION 'Personal32: patch fuera de contrato' USING ERRCODE='55000'; END IF;
END $consulta$;

CREATE FUNCTION vec_personal.exportar_servicios_propios_empleado_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE
 m jsonb; d jsonb; c jsonb; consumo record; proyeccion record;
 fecha date; conocido timestamptz(6); empleado text; persona text;
 material_canon text; material_sha text; contexto_canon text; contexto_sha text;
 servicios jsonb; ajenas integer; ahora timestamptz(6); previo record; campo_version text;
 campos constant jsonb:='["corte","evidencia","servicios"]';
BEGIN
 IF session_user=current_user
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR current_user<>'vec_personal_propietario'
    OR current_setting('TimeZone')<>'UTC'
    OR NOT EXISTS (SELECT 1 FROM pg_roles r WHERE r.rolname=session_user AND r.rolcanlogin AND r.rolinherit
       AND NOT r.rolsuper AND NOT r.rolcreatedb AND NOT r.rolcreaterole AND NOT r.rolreplication AND NOT r.rolbypassrls
       AND r.rolconfig IS NULL)
    OR NOT EXISTS (SELECT 1 FROM pg_auth_members r WHERE r.member=session_user::regrole
       AND r.roleid='vec_personal_ejecutor'::regrole AND r.inherit_option AND NOT r.set_option AND NOT r.admin_option)
    OR (SELECT count(*) FROM pg_auth_members r WHERE r.member=session_user::regrole)<>1
    OR EXISTS (SELECT 1 FROM pg_auth_members r WHERE r.member='vec_personal_ejecutor'::regrole)
    OR pg_has_role(session_user,'vec_personal_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_personal_migrador','MEMBER')
    OR p_material IS NULL OR octet_length(p_material) NOT BETWEEN 2 AND 4096 OR p_capacidad IS NULL OR p_decision IS NULL OR p_motivo IS NULL
    OR p_contexto IS NULL OR p_persona_version IS NULL OR p_perfil_version IS NULL
    OR p_payload IS NULL OR p_sobre IS NULL OR p_evidencia IS NULL OR p_raiz IS NULL THEN
   RAISE EXCEPTION 'exportación de servicios propios denegada' USING ERRCODE='42501';
 END IF;
 BEGIN
  m:=p_material::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
  c:=convert_from(p_capacidad,'UTF8')::jsonb;
  fecha:=(m->>'vigente_en')::date; conocido:=(m->>'conocido_en')::timestamptz;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'material de exportación de servicios propios inválido' USING ERRCODE='22023'; END;
 empleado:=m->>'empleado_ref'; persona:=m->>'persona_ref';
 IF jsonb_typeof(m) IS DISTINCT FROM 'object'
    OR ARRAY(SELECT jsonb_object_keys(m) ORDER BY 1) IS DISTINCT FROM ARRAY[
      'actor_ref','catalogo_sha256','conocido_en','contexto_actor_ref','contexto_version','cuenta_ref','cuenta_version',
      'empleado_ref','esquema','formato_ref','formato_version','idioma','perfil_ref','perfil_version','persona_ref','persona_version','recibo_ref','vigente_en']
    OR m->>'esquema' IS DISTINCT FROM 'vec.personal.servicios-propios.exportacion.v1'
    OR m->>'recibo_ref' IS NULL OR m->>'recibo_ref' !~ '^fichapropia:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
    OR m->>'idioma' IS NULL OR m->>'idioma' !~ '^[a-z]{2,3}(-[A-Za-z0-9]{2,8}){0,3}$'
    OR m->>'formato_ref' IS NULL OR m->>'formato_ref' !~ '^[a-z][a-z0-9._:-]{1,127}$'
    OR m->>'catalogo_sha256' IS NULL OR m->>'catalogo_sha256' !~ '^[0-9a-f]{64}$'
    OR empleado IS NULL OR empleado !~ '^emp_[A-Za-z0-9_-]{22,128}$'
    OR fecha IS NULL OR NOT isfinite(fecha) OR conocido IS NULL OR NOT isfinite(conocido)
    OR conocido>transaction_timestamp()
    OR fecha::text IS DISTINCT FROM m->>'vigente_en'
    OR to_char(conocido AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"') IS DISTINCT FROM m->>'conocido_en'
    OR m->>'actor_ref' IS NULL OR m->>'actor_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR m->>'contexto_actor_ref' IS NULL OR m->>'contexto_actor_ref' !~ '^[a-z][A-Za-z0-9_:-]{2,159}$'
    OR m->>'cuenta_ref' IS NULL OR m->>'cuenta_ref' !~ '^cta_[A-Za-z0-9_-]{22,128}$'
    OR m->>'perfil_ref' IS NULL OR m->>'perfil_ref' !~ '^prf_[A-Za-z0-9_-]{22,128}$'
    OR persona IS NULL OR persona !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR m->>'contexto_version' !~ '^[1-9][0-9]{0,19}$'
    OR m->>'cuenta_version' !~ '^[1-9][0-9]{0,19}$'
    OR m->>'perfil_version' !~ '^[1-9][0-9]{0,18}$'
    OR m->>'persona_version' !~ '^[1-9][0-9]{0,18}$'
    OR m->>'persona_version' IS DISTINCT FROM p_persona_version::text
    OR m->>'perfil_version' IS DISTINCT FROM p_perfil_version::text
    OR m->>'actor_ref' IS DISTINCT FROM persona
    OR m->>'actor_ref' IS DISTINCT FROM d->>'principal_id'
    OR m->>'perfil_ref' IS DISTINCT FROM d->>'perfil_activo_ref'
    OR d->'concedida' IS DISTINCT FROM 'true'::jsonb
    OR d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'interna_corporativa'
    OR d->>'modulo_id' IS DISTINCT FROM 'personal'
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb THEN
  RAISE EXCEPTION 'material de exportación de servicios propios incompatible' USING ERRCODE='42501';
 END IF;
 FOREACH campo_version IN ARRAY ARRAY['formato_version','contexto_version','cuenta_version','perfil_version','persona_version'] LOOP
  IF jsonb_typeof(m->campo_version) IS DISTINCT FROM 'number'
     OR coalesce(m->>campo_version,'') !~ '^[1-9][0-9]{0,19}$'
     OR (campo_version='formato_version' AND (m->>campo_version)::numeric>9007199254740991) THEN
   RAISE EXCEPTION 'versión de exportación inválida' USING ERRCODE='42501'; END IF;
 END LOOP;
 -- json.Marshal de la estructura Go, en orden de campos y bytes UTF-8 exactos.
 material_canon:='{"esquema":"vec.personal.servicios-propios.exportacion.v1","empleado_ref":'||to_jsonb(empleado)::text||',"recibo_ref":'||to_jsonb(m->>'recibo_ref')::text||
  ',"vigente_en":'||to_jsonb(m->>'vigente_en')::text||',"conocido_en":'||to_jsonb(m->>'conocido_en')::text||
  ',"idioma":'||to_jsonb(m->>'idioma')::text||',"formato_ref":'||to_jsonb(m->>'formato_ref')::text||
  ',"formato_version":'||(m->>'formato_version')||',"catalogo_sha256":'||to_jsonb(m->>'catalogo_sha256')::text||
  ',"actor_ref":'||to_jsonb(m->>'actor_ref')::text||',"contexto_actor_ref":'||to_jsonb(m->>'contexto_actor_ref')::text||
  ',"contexto_version":'||(m->>'contexto_version')||',"cuenta_ref":'||to_jsonb(m->>'cuenta_ref')::text||
  ',"cuenta_version":'||(m->>'cuenta_version')||',"perfil_ref":'||to_jsonb(m->>'perfil_ref')::text||
  ',"perfil_version":'||(m->>'perfil_version')||',"persona_ref":'||to_jsonb(persona)::text||
  ',"persona_version":'||(m->>'persona_version')||'}';
 IF p_material IS DISTINCT FROM material_canon THEN
  RAISE EXCEPTION 'material de exportación de servicios propios no canónico' USING ERRCODE='22023'; END IF;
 material_sha:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 contexto_canon:='{"ambitos":{"empleado_ref":'||to_jsonb(empleado)::text||
  '},"atributos":{"catalogo_sha256":'||to_jsonb(m->>'catalogo_sha256')::text||
  ',"conocido_en":'||to_jsonb(m->>'conocido_en')::text||',"formato_ref":'||to_jsonb(m->>'formato_ref')::text||
  ',"formato_version":'||to_jsonb(m->>'formato_version')::text||',"idioma":'||to_jsonb(m->>'idioma')::text||
  ',"material_sha256":"'||material_sha||'","operacion":"servicios_propios_exportar"'||
  ',"recibo_ref":'||to_jsonb(m->>'recibo_ref')::text||',"vigente_en":'||to_jsonb(m->>'vigente_en')::text||'}}';
 contexto_sha:=encode(sha256(convert_to(contexto_canon,'UTF8')),'hex');
 IF d->>'accion' IS DISTINCT FROM 'personal.registro_empleado.ficha_propia.servicios.exportar'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'exportacion_servicios_propios'
    OR d->>'finalidad' IS DISTINCT FROM 'exportar_servicios_propios' OR d->'campos_permitidos' IS DISTINCT FROM campos
    OR d->>'recurso_ref' IS DISTINCT FROM empleado
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM contexto_sha
    OR c->>'operacion' IS DISTINCT FROM 'personal.registro_empleado.ficha_propia.servicios.exportar'
    OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_personal.registro_empleado.ficha_propia.servicios.exportar.v1'
    OR c->>'efecto_ref' IS DISTINCT FROM empleado OR c->>'huella_efecto_sha256' IS DISTINCT FROM contexto_sha THEN
  RAISE EXCEPTION 'concesión de exportación de servicios propios divergente' USING ERRCODE='42501'; END IF;
 -- Autoridad propia de Personal: el empleado pedido debe ser, ahora, el único
 -- empleado canónico de la persona que consulta. Ausencia, ambigüedad,
 -- revocación o un empleado ajeno se deniegan igual. Barrera de lectores de
 -- 000016 antes de resolver (mismo contrato que ContextoActor 000007): una
 -- publicación confirmada tras la instantánea de esta transacción da 40001.
 PERFORM vec_personal.bloquear_generacion_proyeccion_empleado_persona_v1(persona);
 SELECT * INTO STRICT proyeccion
  FROM vec_personal.resolver_empleado_canonico_persona_v1(persona,transaction_timestamp());
 IF proyeccion.resultado IS DISTINCT FROM 'empleado' OR proyeccion.empleado_ref IS DISTINCT FROM empleado
    OR proyeccion.persona_ref IS DISTINCT FROM persona THEN
  RAISE EXCEPTION 'empleado ajeno a la persona' USING ERRCODE='42501'; END IF;
 -- El recibo y el corte proceden de la consulta previa persistida; nunca
 -- se aceptan cortes reconstruidos desde su huella ni recibos históricos NULL.
 SELECT * INTO previo FROM vec_personal.recibo_ficha_propia_empleado r WHERE r.recibo_ref=m->>'recibo_ref';
 IF NOT FOUND OR previo.empleado_ref IS DISTINCT FROM empleado
    OR previo.vigente_en IS NULL OR previo.conocido_en IS NULL
    OR previo.vigente_en IS DISTINCT FROM fecha OR previo.conocido_en IS DISTINCT FROM conocido THEN
  RAISE EXCEPTION 'exportación de servicios propios denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.consumir_exportacion_servicios_propios_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.consumo_nuevo IS NOT TRUE OR consumo.decision_ref IS DISTINCT FROM d->>'decision_ref'
    OR consumo.efecto_ref IS DISTINCT FROM empleado
    OR consumo.huella_efecto_sha256 IS DISTINCT FROM contexto_sha
    OR consumo.auditoria_ref IS NULL OR consumo.auditoria_ref=''
    OR consumo.consumo_huella_sha256 IS NULL OR consumo.consumo_huella_sha256 !~ '^[0-9a-f]{64}$' THEN
  RAISE EXCEPTION 'consumo de exportación de servicios propios divergente' USING ERRCODE='42501'; END IF;
 -- La última revisión conocida de cada hecho manda. Una relación del empleado
 -- inscrita para otra persona es una incoherencia: no se muestra nada.
 SELECT count(*) INTO ajenas
 FROM (SELECT DISTINCT ON (relacion_ref) persona_ref FROM vec_personal.relacion_servicio_historia
       WHERE empleado_ref=empleado AND conocido_desde<=conocido
       ORDER BY relacion_ref,conocido_desde DESC,revision DESC) r
 WHERE r.persona_ref<>persona;
 IF ajenas<>0 THEN RAISE EXCEPTION 'exportación de servicios propios incoherente' USING ERRCODE='55000'; END IF;
 WITH srv AS (SELECT DISTINCT ON (servicio_ref) * FROM vec_personal.servicio_reconocido_historia
     WHERE empleado_ref=empleado AND conocido_desde<=conocido
     ORDER BY servicio_ref,conocido_desde DESC,revision DESC)
 SELECT coalesce(jsonb_agg(jsonb_build_object(
   'inicio',s.periodo_desde::text,'fin',s.periodo_hasta::text,
   'clase',coalesce(s.catalogo_snapshot #>> '{clase_servicio,denominacion}',''),
   'dias',s.dias_reconocidos,'estado',s.estado)
   ORDER BY s.periodo_desde DESC,s.servicio_ref),'[]'::jsonb)
 INTO servicios FROM srv s
 WHERE s.vigente_desde<=fecha AND (s.vigente_hasta IS NULL OR fecha<s.vigente_hasta);
 IF jsonb_array_length(servicios)>200 THEN
  RAISE EXCEPTION 'exportación de servicios propios excede límite' USING ERRCODE='54000'; END IF;
 ahora:=clock_timestamp();
 IF d->>'valida_hasta' IS NULL OR ahora>=(d->>'valida_hasta')::timestamptz THEN
  RAISE EXCEPTION 'exportación de servicios propios caducada' USING ERRCODE='42501'; END IF;
 -- La evidencia referencia el recibo fuente y el consumo nuevo de generación.
 -- CSV se serializa/valida en el adaptador antes de COMMIT; no acredita entrega.
 RETURN jsonb_build_object('corte',jsonb_build_object('vigente_en',fecha::text,'conocido_en',m->>'conocido_en'),
   'servicios',servicios,
   'evidencia',jsonb_build_object('recibo_ref',previo.recibo_ref,'decision_ref',consumo.decision_ref,
   'efecto_ref',consumo.efecto_ref,'consumo_huella_sha256',consumo.consumo_huella_sha256,
   'auditoria_ref',consumo.auditoria_ref,
   'consultada_en',to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')));
END $f$;
REVOKE ALL ON FUNCTION vec_personal.exportar_servicios_propios_empleado_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 FROM PUBLIC,vec_personal_registrador_frontera,vec_personal_migrador;
GRANT EXECUTE ON FUNCTION vec_personal.exportar_servicios_propios_empleado_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_personal_ejecutor;
DO $acl$
DECLARE f oid:=to_regprocedure('vec_personal.exportar_servicios_propios_empleado_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.proowner='vec_personal_propietario'::regrole
      AND p.prosecdef AND p.provolatile='v' AND p.proparallel='u'
      AND p.proconfig=ARRAY['search_path=pg_catalog','row_security=on','TimeZone=UTC','lock_timeout=2s'])
    OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f)<>2
    OR EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
       WHERE p.oid=f AND (a.grantee NOT IN (p.proowner,'vec_personal_ejecutor'::regrole)
         OR a.grantor<>p.proowner OR a.privilege_type<>'EXECUTE' OR a.is_grantable)) THEN
  RAISE EXCEPTION 'Personal32: ACL exportación incompatible' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
