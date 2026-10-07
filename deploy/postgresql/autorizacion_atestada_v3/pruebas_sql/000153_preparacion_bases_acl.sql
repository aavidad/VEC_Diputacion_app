\set ON_ERROR_STOP on
-- Ensayo aislado después de AD153: sólo negativos sin atestación ficticia.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL statement_timeout='15s';
DO $acl$
DECLARE op text;f oid;owner oid:='vec_autorizacion_atestada_v3_propietario'::regrole;grupo text;
BEGIN
 FOREACH op IN ARRAY ARRAY['guardar','consultar'] LOOP
  f:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_'||op||'_preparacion_bases_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
  grupo:=CASE WHEN op='guardar' THEN 'vec_bolsa_convocatorias_ejecutor_preparacion_bases' ELSE 'vec_bolsa_convocatorias_lector_preparacion_bases' END;
  IF f IS NULL OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.proowner=owner AND p.prosecdef
    AND p.provolatile='v' AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
    AND p.proargnames=ARRAY['p_capacidad','p_decision','p_motivo','p_contexto','p_persona_version','p_perfil_version','p_payload','p_sobre','p_evidencia','p_raiz',
      'decision_ref','efecto_ref','huella_efecto_sha256','consumo_huella_sha256','auditoria_ref','consumida_en','consumo_nuevo'])
  OR NOT has_function_privilege('vec_bolsa_convocatorias_propietario',f,'EXECUTE')
  OR has_function_privilege(grupo,f,'EXECUTE')
  OR EXISTS(SELECT 1 FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
    WHERE p.oid=f AND (a.grantee NOT IN(owner,'vec_bolsa_convocatorias_propietario'::regrole) OR a.grantor<>owner OR a.is_grantable))
  THEN RAISE EXCEPTION 'AD153: fachada o ACL fuera de contrato'; END IF;
 END LOOP;
 IF has_function_privilege('vec_bolsa_convocatorias_ejecutor_preparacion_bases',
  'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,'EXECUTE')
 OR has_function_privilege('vec_bolsa_convocatorias_lector_preparacion_bases',
  'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,'EXECUTE')
 THEN RAISE EXCEPTION 'AD153: runtime alcanza el núcleo'; END IF;
END $acl$;
DO $proyeccion$
DECLARE op text;c jsonb;d jsonb;msg text;errores integer:=0;
BEGIN
 FOREACH op IN ARRAY ARRAY['guardar','consultar'] LOOP
  c:=jsonb_build_object('operacion','bolsa.preparacion_bases.'||op,
   'audiencia_consumo','vec_bolsa_convocatorias.preparacion_bases.'||op||'.v1',
   'efecto_ref','prep:sintetica','huella_efecto_sha256',repeat('a',64));
  d:=jsonb_build_object('accion',c->>'operacion','modulo_id','bolsa','tipo_recurso','preparacion_bases','finalidad','preparacion_bases',
   'recurso_ref',c->>'efecto_ref','contexto_recurso_huella_sha256',c->>'huella_efecto_sha256',
   'vinculo_autenticacion_actor',jsonb_build_object('superficie','interna_corporativa','cuenta_privilegiada',false),
   'campos_permitidos',CASE WHEN op='guardar' THEN '["auditoria","evento_outbox","historia","material_preparacion"]'::jsonb ELSE '["material_preparacion"]'::jsonb END,
   'obligaciones','[]'::jsonb);
  BEGIN
   EXECUTE 'SELECT * FROM vec_autorizacion_atestada_v3.consumir_'||op||'_preparacion_bases_bolsa_v3_atestada($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)'
    USING convert_to(c::text,'UTF8'),convert_to(d::text,'UTF8'),''::bytea,''::bytea,1::numeric,1::numeric,''::bytea,''::bytea,''::bytea,''::bytea;
   RAISE EXCEPTION 'AD153: recibo implícito permitido';
  EXCEPTION WHEN insufficient_privilege THEN
   GET STACKED DIAGNOSTICS msg=MESSAGE_TEXT;
   IF msg IS DISTINCT FROM 'AD153: preparación de bases denegada' THEN RAISE EXCEPTION 'AD153: proyección inválida alcanzó el núcleo'; END IF;
   errores:=errores+1;
  END;
 END LOOP;
 IF errores<>2 THEN RAISE EXCEPTION 'AD153: negativos incompletos'; END IF;
END $proyeccion$;
ROLLBACK;
