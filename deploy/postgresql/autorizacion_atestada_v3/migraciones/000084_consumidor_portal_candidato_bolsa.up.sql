\set ON_ERROR_STOP on
-- AD3-84. Acciones propias del candidato en «Mi bolsa» (petición RRHH p. 1
-- punto 5 y p. 2; dudas 3, 17 y 18): solicitar pausa, solicitar reactivación,
-- responder a un llamamiento abierto y manifestar disposición a una oferta
-- publicada. Un único perfil nominal 'portal_candidato_bolsa' con cuatro
-- operaciones, cada una con su audiencia, y un único propietario consumidor:
-- Bolsa (000030 para las tres primeras; la disposición la consumirá la
-- función de ofertas). El recurso es el propio 'mi-bolsa:<candidato>' o la
-- oferta; Bolsa coteja siempre al candidato con los vínculos del contexto.
-- Se inserta junto a la primera línea de cada lista del núcleo, de modo que no
-- depende del orden de instalación de otras extensiones; toma el consultivo
-- común del núcleo antes de leer su preimagen.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000084',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));

DO $nucleo$
DECLARE
 f oid:='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 original text; nuevo text; actual text; meta jsonb; deps jsonb; acl aclitem[];
 propietario oid; config text[]; definidora boolean;
 marca text:=E'       )\n       OR c ->> ''suite'' <> ''VEC-AD-3-COSE-EDDSA-1''';
 excl text:=E'               p_perfil_mutacion IS DISTINCT FROM ''bolsa_llamamiento''\n';
 excl_nuevo text:=excl||E'               AND p_perfil_mutacion IS DISTINCT FROM ''portal_candidato_bolsa''\n';
 runtime text:=E'(p_perfil_mutacion IS NOT DISTINCT FROM ''bolsa_llamamiento''\n';
 runtime_nuevo text:=runtime||E'               OR p_perfil_mutacion IS NOT DISTINCT FROM ''portal_candidato_bolsa''\n';
 extension text:=$x$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'portal_candidato_bolsa'
 AND ((((c->>'operacion' IS NOT DISTINCT FROM 'bolsa.participaciones_propias.solicitar_pausa'
         AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.participaciones_propias.solicitar_pausa.v1')
     OR (c->>'operacion' IS NOT DISTINCT FROM 'bolsa.participaciones_propias.solicitar_reactivacion'
         AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.participaciones_propias.solicitar_reactivacion.v1')
     OR (c->>'operacion' IS NOT DISTINCT FROM 'bolsa.participaciones_propias.responder_llamamiento'
         AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.participaciones_propias.responder_llamamiento.v1'))
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'participaciones_candidato')
   OR (c->>'operacion' IS NOT DISTINCT FROM 'bolsa.participaciones_propias.manifestar_disposicion'
       AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.participaciones_propias.manifestar_disposicion.v1'
       AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'oferta_bolsa'))
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestion_participaciones_propias'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
$x$;
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_mi_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_portal_candidato_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_bolsa_llamamientos_propietario' AND NOT rolcanlogin AND NOT rolbypassrls)
 THEN RAISE EXCEPTION 'AD3-84: preimagen incompatible' USING ERRCODE='55000'; END IF;
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc',p.proacl,p.proowner,p.proconfig,p.prosecdef
 INTO STRICT original,meta,acl,propietario,config,definidora FROM pg_proc p WHERE p.oid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
 INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 -- El núcleo debe llevar la consulta propia (AD3-43) y ninguna versión previa
 -- de esta extensión; cada marca aparece exactamente una vez.
 IF propietario<>'vec_autorizacion_atestada_v3_propietario'::regrole OR NOT definidora
    OR config IS DISTINCT FROM ARRAY['search_path=pg_catalog','lock_timeout=2s']
    OR length(original)-length(replace(original,marca,''))<>length(marca)
    OR length(original)-length(replace(original,excl,''))<>length(excl)
    OR length(original)-length(replace(original,runtime,''))<>length(runtime)
    OR strpos(original,'consulta_participaciones_propias_bolsa')=0
    OR strpos(original,'portal_candidato_bolsa')<>0
 THEN RAISE EXCEPTION 'AD3-84: núcleo incompatible' USING ERRCODE='55000'; END IF;
 nuevo:=replace(original,excl,excl_nuevo);
 nuevo:=replace(nuevo,runtime,runtime_nuevo);
 nuevo:=replace(nuevo,marca,extension||marca);
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo
    OR replace(replace(replace(actual,extension||marca,marca),runtime_nuevo,runtime),excl_nuevo,excl) IS DISTINCT FROM original
    OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
    OR (SELECT proacl FROM pg_proc WHERE oid=f) IS DISTINCT FROM acl
    OR (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM propietario
    OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM config
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS DISTINCT FROM definidora
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb) FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
 THEN RAISE EXCEPTION 'AD3-84: núcleo alterado fuera del contrato' USING ERRCODE='55000'; END IF;
END $nucleo$;

LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencias$
DECLARE d text; a text;
 nuevas text[]:=ARRAY['vec_bolsa_llamamientos.participaciones_propias.solicitar_pausa.v1',
                      'vec_bolsa_llamamientos.participaciones_propias.solicitar_reactivacion.v1',
                      'vec_bolsa_llamamientos.participaciones_propias.responder_llamamiento.v1',
                      'vec_bolsa_llamamientos.participaciones_propias.manifestar_disposicion.v1'];
BEGIN
 SELECT regexp_replace(pg_get_constraintdef(c.oid,true),'\s+',' ','g') INTO STRICT d
 FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
 AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 IF strpos(d,'CHECK (audiencia_consumo = ANY (ARRAY[')<>1 OR right(d,3)<>']))' OR strpos(d,'vec.bolsa.mi-bolsa.v1')=0
 THEN RAISE EXCEPTION 'AD3-84: audiencias previas incompatibles' USING ERRCODE='55000'; END IF;
 FOREACH a IN ARRAY nuevas LOOP
  IF strpos(d,quote_literal(a))<>0 THEN RAISE EXCEPTION 'AD3-84: audiencia ya presente' USING ERRCODE='55000'; END IF;
  d:=left(d,length(d)-3)||', '||quote_literal(a)||'::text]))';
 END LOOP;
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version
  DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||d;
END $audiencias$;

-- Fachada única. Además de la guarda del núcleo exige el recurso propio
-- 'mi-bolsa:<candidato>', sin campos ni obligaciones, y un consumo nuevo.
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_portal_candidato_bolsa_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; x record;
BEGIN
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'AD3-84: material del portal del candidato inválido' USING ERRCODE='22023'; END;
 IF coalesce(c->>'operacion','') NOT IN ('bolsa.participaciones_propias.solicitar_pausa','bolsa.participaciones_propias.solicitar_reactivacion','bolsa.participaciones_propias.responder_llamamiento','bolsa.participaciones_propias.manifestar_disposicion')
    OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_bolsa_llamamientos.participaciones_propias.'||substr(c->>'operacion',length('bolsa.participaciones_propias.')+1)||'.v1'
    OR d->>'accion' IS DISTINCT FROM c->>'operacion'
    OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
    OR d->>'finalidad' IS DISTINCT FROM 'gestion_participaciones_propias'
    OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
    -- Las tres acciones sobre la propia bolsa actúan sobre 'mi-bolsa:<candidato>';
    -- manifestar disposición actúa sobre la oferta y Bolsa coteja al candidato
    -- con los vínculos del contexto.
    OR (c->>'operacion' <> 'bolsa.participaciones_propias.manifestar_disposicion'
        AND (d->>'tipo_recurso' IS DISTINCT FROM 'participaciones_candidato' OR coalesce(d->>'recurso_ref','') !~ '^mi-bolsa:can_[A-Za-z0-9_-]{22,128}$'))
    OR (c->>'operacion' = 'bolsa.participaciones_propias.manifestar_disposicion'
        AND (d->>'tipo_recurso' IS DISTINCT FROM 'oferta_bolsa' OR coalesce(d->>'recurso_ref','') !~ '^oferta:[0-9a-f]{64}$'))
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
    OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'AD3-84: material del portal del candidato rechazado' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  'portal_candidato_bolsa',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'AD3-84: el portal del candidato requiere consumo nuevo' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;

DO $acl$
DECLARE f regprocedure:='vec_autorizacion_atestada_v3.registrar_y_consumir_portal_candidato_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 permitido oid:='vec_bolsa_llamamientos_propietario'::regrole::oid; x record;
BEGIN
 EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC',f::text);
 EXECUTE format('GRANT EXECUTE ON FUNCTION %s TO vec_bolsa_llamamientos_propietario',f::text);
 IF (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario'::regrole
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS NOT TRUE
    OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM ARRAY['search_path=pg_catalog','lock_timeout=2s']
    OR NOT has_schema_privilege('vec_bolsa_llamamientos_propietario','vec_autorizacion_atestada_v3','USAGE')
 THEN RAISE EXCEPTION 'AD3-84: propietario o entorno de fachada incompatible' USING ERRCODE='55000'; END IF;
 FOR x IN SELECT a.grantee,a.privilege_type,a.is_grantable,p.proowner
  FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f LOOP
  IF (x.grantee<>x.proowner AND x.grantee IS DISTINCT FROM permitido) OR x.privilege_type<>'EXECUTE'
    OR (x.grantee=permitido AND x.is_grantable)
  THEN RAISE EXCEPTION 'AD3-84: ACL de fachada abierta' USING ERRCODE='55000'; END IF;
 END LOOP;
END $acl$;
COMMIT;
