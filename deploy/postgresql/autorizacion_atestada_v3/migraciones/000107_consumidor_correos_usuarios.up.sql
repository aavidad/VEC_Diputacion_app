\set ON_ERROR_STOP on
-- AD3-107: operaciones nominales 5.08b. Instalar después de AD3-106 y antes
-- de Usuarios 000004. No presupone configuración SMTP ni eficacia de entrega.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000107',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $pre$ BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_preferencias_actualizacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_correos_v3_atestada(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_usuarios_ejecutor_interno' AND NOT rolcanlogin AND NOT rolbypassrls)
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_usuarios_ejecutor_externo' AND NOT rolcanlogin AND NOT rolbypassrls)
 THEN RAISE EXCEPTION 'AD3-107: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

DO $nucleo$
DECLARE
 f oid:='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 original text; nuevo text; actual text; excl_nuevo text; guarda_nueva text;
 marca text:=E'       )\n       OR c ->> ''suite'' <> ''VEC-AD-3-COSE-EDDSA-1''';
 excl text:=E'               AND p_perfil_mutacion IS DISTINCT FROM ''preferencias_actualizacion_usuarios''\n';
 guarda text:=E'p_perfil_mutacion IN (''preferencias_consulta_usuarios'',''preferencias_actualizacion_usuarios'')';
 perfiles text:=E'''correos_consultar_usuarios'',''correos_anadir_usuarios'',''correos_reenviar_usuarios'',''correos_verificar_usuarios'',''correos_activar_usuarios'',''correos_retirar_usuarios''';
 extension text;
 meta jsonb; deps jsonb; acl aclitem[]; propietario oid; config text[]; definidora boolean;
 i integer; a text; p text;
BEGIN
 SELECT pg_get_functiondef(f),to_jsonb(q)-'prosrc',q.proacl,q.proowner,q.proconfig,q.prosecdef
 INTO STRICT original,meta,acl,propietario,config,definidora FROM pg_proc q WHERE q.oid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.objsubid,d.deptype),'[]'::jsonb)
 INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 IF propietario<>'vec_autorizacion_atestada_v3_propietario'::regrole OR NOT definidora
    OR config IS DISTINCT FROM ARRAY['search_path=pg_catalog','lock_timeout=2s']
    OR (length(original)-length(replace(original,marca,'')))<>length(marca)
    OR (length(original)-length(replace(original,excl,'')))<>length(excl)
    OR (length(original)-length(replace(original,guarda,'')))<>3*length(guarda)
    OR strpos(original,'correos_consultar_usuarios')<>0
 THEN RAISE EXCEPTION 'AD3-107: núcleo incompatible' USING ERRCODE='55000'; END IF;
 excl_nuevo:=excl||
  E'               AND p_perfil_mutacion IS DISTINCT FROM ''correos_consultar_usuarios''\n               AND p_perfil_mutacion IS DISTINCT FROM ''correos_anadir_usuarios''\n               AND p_perfil_mutacion IS DISTINCT FROM ''correos_reenviar_usuarios''\n               AND p_perfil_mutacion IS DISTINCT FROM ''correos_verificar_usuarios''\n               AND p_perfil_mutacion IS DISTINCT FROM ''correos_activar_usuarios''\n               AND p_perfil_mutacion IS DISTINCT FROM ''correos_retirar_usuarios''\n';
 guarda_nueva:=replace(guarda,')',','||perfiles||')');
 nuevo:=replace(original,excl,excl_nuevo);
 -- Tres lugares: guardas técnicas interna/externa previas al parseo y
 -- guarda de superficie después del parseo de la decisión firmada.
 nuevo:=replace(nuevo,guarda,guarda_nueva);
 extension:='';
 FOR i IN 1..6 LOOP
  a:=(ARRAY['consultar','anadir','reenviar','verificar','activar','retirar'])[i];
  p:=(ARRAY['correos_consultar_usuarios','correos_anadir_usuarios','correos_reenviar_usuarios','correos_verificar_usuarios','correos_activar_usuarios','correos_retirar_usuarios'])[i];
  extension:=extension||format(E'           OR (\n p_perfil_mutacion IS NOT DISTINCT FROM %L\n AND ((c->>''audiencia_consumo'' IS NOT DISTINCT FROM %L\n       AND d #>> ''{vinculo_autenticacion_actor,superficie}'' IS NOT DISTINCT FROM ''interna_corporativa'')\n   OR (c->>''audiencia_consumo'' IS NOT DISTINCT FROM %L\n       AND d #>> ''{vinculo_autenticacion_actor,superficie}'' IS NOT DISTINCT FROM ''externa_personal''))\n AND c->>''operacion'' IS NOT DISTINCT FROM %L\n AND d->>''accion'' IS NOT DISTINCT FROM c->>''operacion''\n AND d->>''modulo_id'' IS NOT DISTINCT FROM ''usuarios''\n AND d->>''tipo_recurso'' IS NOT DISTINCT FROM ''correos_persona''\n AND d->>''finalidad'' IS NOT DISTINCT FROM ''finalidad:usuarios:correos-propios:v1''\n AND d->>''recurso_ref'' IS NOT DISTINCT FROM c->>''efecto_ref''\n AND d->>''contexto_recurso_huella_sha256'' IS NOT DISTINCT FROM c->>''huella_efecto_sha256''\n AND d->''campos_permitidos'' IS NOT DISTINCT FROM %L::jsonb\n AND d->''obligaciones'' IS NOT DISTINCT FROM ''[]''::jsonb)\n',
   p,'vec_usuarios.correos.'||a||'.interna_corporativa.v1',
   'vec_usuarios.correos.'||a||'.externa_personal.v1','vec.correos.'||a,
   CASE i WHEN 1 THEN '["activo","correo_ref","direccion","estado","version"]'
    WHEN 2 THEN '["correo_ref","direccion","estado","version"]'
    WHEN 3 THEN '["correo_ref","estado","version"]'
    WHEN 4 THEN '["correo_ref","estado","version"]'
    WHEN 5 THEN '["activo","correo_ref","version"]'
    WHEN 6 THEN '["activo","correo_ref","estado","version"]' END);
 END LOOP;
 nuevo:=replace(nuevo,marca,extension||marca);
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo
    OR replace(replace(replace(actual,extension||marca,marca),guarda_nueva,guarda),excl_nuevo,excl) IS DISTINCT FROM original
    OR (SELECT to_jsonb(q)-'prosrc' FROM pg_proc q WHERE q.oid=f) IS DISTINCT FROM meta
    OR (SELECT proacl FROM pg_proc WHERE oid=f) IS DISTINCT FROM acl
    OR (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM propietario
    OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM config
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS DISTINCT FROM definidora
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.objsubid,d.deptype),'[]'::jsonb) FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
 THEN RAISE EXCEPTION 'AD3-107: metadatos alterados' USING ERRCODE='55000'; END IF;
END $nucleo$;

LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencias$
DECLARE d text; nueva text; a text; s text;
BEGIN
 SELECT regexp_replace(pg_get_constraintdef(c.oid,true),'\s+',' ','g') INTO STRICT d
 FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
 AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 IF strpos(d,'CHECK (audiencia_consumo = ANY (ARRAY[')<>1 OR right(d,3)<>']))'
    OR strpos(d,'vec_usuarios.preferencias.actualizar.externa_personal.v1')=0
 THEN RAISE EXCEPTION 'AD3-107: audiencias incompatibles' USING ERRCODE='55000'; END IF;
 nueva:=left(d,length(d)-3);
 FOREACH a IN ARRAY ARRAY['consultar','anadir','reenviar','verificar','activar','retirar'] LOOP
  FOR s IN SELECT unnest(ARRAY['interna_corporativa','externa_personal']) LOOP
   IF strpos(d,quote_literal('vec_usuarios.correos.'||a||'.'||s||'.v1'))<>0
   THEN RAISE EXCEPTION 'AD3-107: audiencia registrada' USING ERRCODE='55000'; END IF;
   nueva:=nueva||', '||quote_literal('vec_usuarios.correos.'||a||'.'||s||'.v1')||'::text';
  END LOOP;
 END LOOP;
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||nueva||']))';
END $audiencias$;

CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_correos_v3_atestada(
 p_accion text,p_superficie text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; x record; perfil text; audiencia text; campos jsonb; segmento text;
BEGIN
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'AD3-107: material inválido' USING ERRCODE='22023'; END;
 SELECT q.perfil,q.segmento,q.campos INTO perfil,segmento,campos
 FROM (VALUES
 ('vec.correos.consultar','correos_consultar_usuarios','consultar','["activo","correo_ref","direccion","estado","version"]'::jsonb),
 ('vec.correos.anadir','correos_anadir_usuarios','anadir','["correo_ref","direccion","estado","version"]'::jsonb),
 ('vec.correos.reenviar','correos_reenviar_usuarios','reenviar','["correo_ref","estado","version"]'::jsonb),
 ('vec.correos.verificar','correos_verificar_usuarios','verificar','["correo_ref","estado","version"]'::jsonb),
 ('vec.correos.activar','correos_activar_usuarios','activar','["activo","correo_ref","version"]'::jsonb),
 ('vec.correos.retirar','correos_retirar_usuarios','retirar','["activo","correo_ref","estado","version"]'::jsonb)
 ) q(accion,perfil,segmento,campos) WHERE q.accion=p_accion;
 audiencia:='vec_usuarios.correos.'||segmento||'.'||p_superficie||'.v1';
 IF perfil IS NULL OR p_superficie NOT IN ('interna_corporativa','externa_personal')
    OR d #>> '{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM p_superficie
    OR c->>'audiencia_consumo' IS DISTINCT FROM audiencia
    OR c->>'operacion' IS DISTINCT FROM p_accion OR d->>'accion' IS DISTINCT FROM p_accion
    OR d->>'modulo_id' IS DISTINCT FROM 'usuarios' OR d->>'tipo_recurso' IS DISTINCT FROM 'correos_persona'
    OR d->>'finalidad' IS DISTINCT FROM 'finalidad:usuarios:correos-propios:v1' OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
    OR d->'campos_permitidos' IS DISTINCT FROM campos OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'AD3-107: correo denegado' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  perfil,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'AD3-107: requiere consumo nuevo' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_correos_v3_atestada(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_correos_v3_atestada(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_usuarios_propietario;
DO $acl$ DECLARE f regprocedure:='vec_autorizacion_atestada_v3.consumir_correos_v3_atestada(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
BEGIN
 IF (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario'::regrole
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS NOT TRUE
    OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM ARRAY['search_path=pg_catalog','lock_timeout=2s']
    OR NOT has_function_privilege('vec_usuarios_propietario',f,'EXECUTE')
    OR has_function_privilege('vec_usuarios_ejecutor_interno',f,'EXECUTE')
    OR has_function_privilege('vec_usuarios_ejecutor_externo',f,'EXECUTE')
    OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
      WHERE p.oid=f AND (a.grantee=0 OR a.grantee NOT IN (p.proowner,'vec_usuarios_propietario'::regrole)))
 THEN RAISE EXCEPTION 'AD3-107: ACL incompatible' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
