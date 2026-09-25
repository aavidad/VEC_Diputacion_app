\set ON_ERROR_STOP on
-- AD3-85: consumo nominal del registro de firmas de los borradores de
-- Contratación temporal (CT118). Una audiencia propia, un perfil de mutación
-- propio y una fachada con un único llamante: el propietario de CT. No
-- reescribe otros consumidores ni concede claves. Admite instalarse antes o
-- después de otras extensiones del núcleo, siempre en serie: toma el cerrojo
-- común del núcleo antes de leer su preimagen.
-- La firma que registra CT118 es una firma de prueba verificada: no tiene
-- eficacia administrativa hasta el portafirmas corporativo.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000085',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));

DO $pre$
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_despacho_correo_llamamiento_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
               WHERE n.nspname='vec_autorizacion_atestada_v3'
                 AND p.proname='registrar_y_consumir_firma_documento_ct_v3_atestada')
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_contratacion_temporal_propietario'
                    AND NOT rolcanlogin AND NOT rolsuper AND NOT rolcreatedb AND NOT rolcreaterole
                    AND NOT rolreplication AND NOT rolbypassrls)
 THEN RAISE EXCEPTION 'AD3-85: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

DO $nucleo$
DECLARE
 f oid:='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 original text; nuevo text; actual text; meta jsonb; deps jsonb; acl aclitem[];
 propietario oid; config text[]; definidora boolean;
 marca text:=E'       )\n       OR c ->> ''suite'' <> ''VEC-AD-3-COSE-EDDSA-1''';
 extension text:=$x$           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'firma_documento_ct'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.firma_documento.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM 'contratacion_temporal.documento.firmar'
               AND c ->> 'operacion' IS NOT DISTINCT FROM d ->> 'accion'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'firma_documento_contratacion_temporal'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'gestionar_contratacion_temporal'
               AND d ->> 'recurso_ref' IS NOT DISTINCT FROM c ->> 'efecto_ref'
               AND d ->> 'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c ->> 'huella_efecto_sha256'
               AND d -> 'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb
           )
$x$;
BEGIN
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc',p.proacl,p.proowner,p.proconfig,p.prosecdef
 INTO STRICT original,meta,acl,propietario,config,definidora FROM pg_proc p WHERE p.oid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
 INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 -- La marca aparece exactamente una vez; el perfil nuevo no aparece aún y el
 -- núcleo conserva los consumidores de CT de los que depende la forma.
 IF propietario<>'vec_autorizacion_atestada_v3_propietario'::regrole OR NOT definidora
    OR config IS DISTINCT FROM ARRAY['search_path=pg_catalog','lock_timeout=2s']
    OR length(original)-length(replace(original,marca,''))<>length(marca)
    OR strpos(original,'firma_documento_ct')<>0
    OR strpos(original,'vec_contratacion_temporal.firma_documento.v1')<>0
    OR strpos(original,'despacho_correo_llamamiento_ct')=0
    OR strpos(original,'cierre_administrativo_sin_cese_ct')=0
 THEN RAISE EXCEPTION 'AD3-85: núcleo incompatible' USING ERRCODE='55000'; END IF;
 nuevo:=replace(original,marca,extension||marca);
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo
    OR replace(actual,extension||marca,marca) IS DISTINCT FROM original
    OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
    OR (SELECT proacl FROM pg_proc WHERE oid=f) IS DISTINCT FROM acl
    OR (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM propietario
    OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM config
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS DISTINCT FROM definidora
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb) FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
 THEN RAISE EXCEPTION 'AD3-85: núcleo alterado fuera del contrato' USING ERRCODE='55000'; END IF;
END $nucleo$;

LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencias$
DECLARE d text; audiencia text:='vec_contratacion_temporal.firma_documento.v1';
BEGIN
 SELECT regexp_replace(pg_get_constraintdef(c.oid,true),'\s+',' ','g') INTO STRICT d
 FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
 AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 IF strpos(d,'CHECK (audiencia_consumo = ANY (ARRAY[')<>1 OR right(d,3)<>']))'
    OR strpos(d,'vec_contratacion_temporal.despacho_correo_llamamiento.v1')=0
    OR strpos(d,quote_literal(audiencia))<>0
 THEN RAISE EXCEPTION 'AD3-85: audiencias previas incompatibles' USING ERRCODE='55000'; END IF;
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version
  DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '
  ||left(d,length(d)-3)||', '||quote_literal(audiencia)||'::text]))';
END $audiencias$;

-- La fachada exige un consumo nuevo en cada registro: una recuperación por
-- la misma clave de idempotencia también presenta una concesión nueva.
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_firma_documento_ct_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; x record;
BEGIN
 -- Comprobaciones previas, como las demás fachadas: el material que no es de
 -- firma de un documento de CT se deniega antes de llegar al núcleo.
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'AD3-85: material de firma inválido' USING ERRCODE='22023'; END;
 IF c->>'operacion' IS DISTINCT FROM 'contratacion_temporal.documento.firmar'
    OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_contratacion_temporal.firma_documento.v1'
    OR d->>'accion' IS DISTINCT FROM c->>'operacion'
    OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'firma_documento_contratacion_temporal'
    OR d->>'finalidad' IS DISTINCT FROM 'gestionar_contratacion_temporal'
    OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'AD3-85: firma denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  'firma_documento_ct',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE THEN
  RAISE EXCEPTION 'AD3-85: la firma requiere una concesión nueva' USING ERRCODE='P1102';
 END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;

DO $acl$
DECLARE f regprocedure:='vec_autorizacion_atestada_v3.registrar_y_consumir_firma_documento_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 permitido oid:='vec_contratacion_temporal_propietario'::regrole::oid; x record;
BEGIN
 EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC',f::text);
 FOR x IN SELECT DISTINCT a.grantee FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
  WHERE p.oid=f AND a.grantee<>0 AND a.grantee<>p.proowner LOOP
  EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %I',f::text,pg_get_userbyid(x.grantee));
 END LOOP;
 EXECUTE format('GRANT EXECUTE ON FUNCTION %s TO vec_contratacion_temporal_propietario',f::text);
 IF (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario'::regrole
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS NOT TRUE
    OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM ARRAY['search_path=pg_catalog','lock_timeout=2s']
 THEN RAISE EXCEPTION 'AD3-85: propietario o entorno de fachada incompatible' USING ERRCODE='55000'; END IF;
 FOR x IN SELECT a.grantee,a.privilege_type,a.is_grantable,p.proowner
  FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f LOOP
  IF (x.grantee<>x.proowner AND x.grantee IS DISTINCT FROM permitido) OR x.privilege_type<>'EXECUTE'
    OR (x.grantee=permitido AND x.is_grantable)
  THEN RAISE EXCEPTION 'AD3-85: ACL de fachada abierta' USING ERRCODE='55000'; END IF;
 END LOOP;
END $acl$;
COMMIT;
