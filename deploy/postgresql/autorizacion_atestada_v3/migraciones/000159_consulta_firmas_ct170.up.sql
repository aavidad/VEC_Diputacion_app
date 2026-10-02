\set ON_ERROR_STOP on
-- AD3-159: consumidor nominal de la consulta CT170 R5 de firmas por
-- expediente, versión y documento. No concede registro ni lectura bruta.
-- La fachada exige consumo nuevo incluso en replay y solo la invoca el propietario de CT.
-- La decisión liga operación, recurso, finalidad, campos y efecto exactos.
-- Se instala en serie con cualquier otra reescritura del núcleo: toma el
-- cerrojo común antes de leer su preimagen. Sin DOWN tras historia.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000159',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));

DO $pre$
BEGIN
 IF current_user IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario' THEN
  RAISE EXCEPTION 'AD3-159: PARO clave=rol_sql actual=distinto esperado=propietario_ad3' USING ERRCODE='55000'; END IF;
 IF to_regprocedure('vec_autorizacion_atestada_v3.consumir_consulta_firmas_documento_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_vinculo_propio_crn11_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_circuito_ct_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_firma_externa_ct_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_firma_vec_ct_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 THEN RAISE EXCEPTION 'AD3-159: PARO clave=preimagen_ad125_ad149_ad151_ad156_ad157 actual=incompleta esperado=instalada' USING ERRCODE='55000'; END IF;
 IF NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=
      to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')
      AND strpos(p.prosrc,'documentos.original_firmable.reservar')>0
      AND strpos(p.prosrc,'documentos.original_firmable.confirmar')>0)
    OR NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=
      to_regprocedure('vec_autorizacion_atestada_v3.consumir_operacion_documentos_replay_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')
      AND strpos(p.prosrc,'documentos.original_firmable.reservar')>0
      AND strpos(p.prosrc,'documentos.original_firmable.confirmar')>0)
 THEN RAISE EXCEPTION 'AD3-159: PARO clave=preimagen_ad158 actual=incompleta esperado=instalada' USING ERRCODE='55000'; END IF;
 IF to_regprocedure('vec_autorizacion_atestada_v3.consumir_consulta_firmas_r5_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 THEN RAISE EXCEPTION 'AD3-159: PARO clave=fachada_ya_instalada actual=true esperado=false' USING ERRCODE='55000'; END IF;
 IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_contratacion_temporal_propietario'
                   AND NOT rolcanlogin AND NOT rolsuper AND NOT rolcreaterole AND NOT rolbypassrls)
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_contratacion_temporal_ejecutor'
                   AND NOT rolbypassrls)
 THEN RAISE EXCEPTION 'AD3-159: PARO clave=roles_ct actual=incompatible esperado=cerrados' USING ERRCODE='55000'; END IF;
END $pre$;

-- PARO causal del borrador: AD157 todavía no puede instalarse y AD158 está en
-- preparación. No se conoce la preimagen íntegra post-AD158 del núcleo ni del
-- CHECK de audiencias. Esta puerta se sustituye SOLO tras capturar en PG18 la
-- definición, fuente, ACL, dependencias y constraint reales, y ensayar la
-- cadena AD149→AD151→AD156→AD157→AD158→AD159. Mientras tanto no hay efecto.
DO $preimagen_pendiente$
BEGIN
 RAISE EXCEPTION 'AD3-159: PARO clave=preimagen_post_ad158_acreditada actual=false esperado=true' USING ERRCODE='55000';
END $preimagen_pendiente$;

-- Una segunda puerta independiente conserva cerrada la lectura R5 hasta que
-- la autoridad publique y ligue perfiles exactos de RRHH y firmante VEC. El
-- perfil de registrador externo AD156 y el de CT152 no sirven para esta lectura.
DO $perfil_pendiente$
BEGIN
 RAISE EXCEPTION 'AD3-159: PARO clave=perfiles_fijos_r5_publicados_vigentes actual=false esperado=true' USING ERRCODE='55000';
END $perfil_pendiente$;

DO $nucleo$
DECLARE
 f oid:='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 original text; fuente text; nuevo text; actual text; meta jsonb; deps jsonb; deps_compartidas jsonb; acl aclitem[];
 propietario oid; config text[]; definidora boolean;
 -- Se fijan únicamente tras capturar la postimagen real AD158 en PG18.
 esperada_def_sha256 text:=NULL;
 esperada_fuente_sha256 text:=NULL;
 marca text:=E'       )\n       OR c ->> ''suite'' <> ''VEC-AD-3-COSE-EDDSA-1''';
 extension text:=$x$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_firmas_r5_ct'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.firmas_r5.consultar.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'contratacion_temporal.documento.firmas_r5.consultar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'expediente_contratacion_temporal'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestionar_contratacion_temporal'
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["CatalogoHuella","CatalogoRef","ClaveIdempotencia","ConMotivoDevolucion","Documento","DocumentoCustodiaRef","DocumentoCustodiaVersion","ExpedienteVersion","FechaPortafirmasDeclarada","FirmaRef","FirmadoHuella","FirmantePrincipalAcreditado","OriginalHuella","OriginalRef","OriginalVersion","PasoOrden","PasoRef","ReciboRef","ReferenciaPortafirmasDeclarada","RegistradaEn","Resultado","Secuencia","SelloTiempoEstado","Via"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
$x$;
BEGIN
 SELECT pg_get_functiondef(f),p.prosrc,to_jsonb(p)-'prosrc',p.proacl,p.proowner,p.proconfig,p.prosecdef
 INTO STRICT original,fuente,meta,acl,propietario,config,definidora FROM pg_proc p WHERE p.oid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
 INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
 INTO deps_compartidas FROM pg_shdepend d
 WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database())
   AND d.classid='pg_proc'::regclass AND d.objid=f;
 -- El perfil de CT pasa por el bloque general del núcleo, que exige una
 -- sesión miembro del ejecutor CT. Ninguna versión previa puede existir y la
 -- marca aparece exactamente una vez.
 IF esperada_def_sha256 IS NULL OR esperada_fuente_sha256 IS NULL
    OR encode(sha256(convert_to(original,'UTF8')),'hex') IS DISTINCT FROM esperada_def_sha256
    OR encode(sha256(convert_to(fuente,'UTF8')),'hex') IS DISTINCT FROM esperada_fuente_sha256
    OR propietario<>'vec_autorizacion_atestada_v3_propietario'::regrole OR NOT definidora
    OR config IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
    OR length(original)-length(replace(original,marca,''))<>length(marca)
    OR strpos(original,'''consulta_firmas_documento_ct''')=0
    OR strpos(original,'''ct_circuito_consultar''')=0
    OR strpos(original,'''firma_externa_documento_ct''')=0
    OR strpos(original,'vec_contratacion_temporal_ejecutor')=0
    OR strpos(original,'consulta_firmas_r5_ct')<>0
    OR strpos(original,'contratacion_temporal.documento.firmas_r5.consultar')<>0
 THEN RAISE EXCEPTION 'AD3-159: núcleo incompatible' USING ERRCODE='55000'; END IF;
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
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
        FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
        FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database())
          AND d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps_compartidas
 THEN RAISE EXCEPTION 'AD3-159: núcleo alterado fuera del contrato' USING ERRCODE='55000'; END IF;
END $nucleo$;

LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencias$
DECLARE d text; audiencia text:='vec_contratacion_temporal.firmas_r5.consultar.v1';
 esperada_check_sha256 text:=NULL; -- pendiente de captura post-AD158 en PG18
BEGIN
 SELECT regexp_replace(pg_get_constraintdef(c.oid,true),'\s+',' ','g') INTO STRICT d
 FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
 AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 IF esperada_check_sha256 IS NULL
    OR encode(sha256(convert_to(d,'UTF8')),'hex') IS DISTINCT FROM esperada_check_sha256
    OR strpos(d,'CHECK (audiencia_consumo = ANY (ARRAY[')<>1 OR right(d,3)<>']))'
    OR strpos(d,'vec_contratacion_temporal.lectura_reincorporacion_titular.v1')=0
    OR strpos(d,quote_literal(audiencia))<>0
 THEN RAISE EXCEPTION 'AD3-159: audiencias previas incompatibles' USING ERRCODE='55000'; END IF;
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version
  DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '
  ||left(d,length(d)-3)||', '||quote_literal(audiencia)||'::text]))';
END $audiencias$;

-- Fachada única: la lectura queda ligada al expediente, campos exactos y
-- decisión sin obligaciones. CT170 consume antes de leer incluso si no hay filas.
CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_consulta_firmas_r5_ct_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; x record;
BEGIN
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'AD3-159: material de consulta inválido' USING ERRCODE='22023'; END;
 IF c->>'operacion' IS DISTINCT FROM 'contratacion_temporal.documento.firmas_r5.consultar'
    OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_contratacion_temporal.firmas_r5.consultar.v1'
    OR d->>'accion' IS DISTINCT FROM c->>'operacion'
    OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'expediente_contratacion_temporal'
    OR d->>'finalidad' IS DISTINCT FROM 'gestionar_contratacion_temporal'
    OR d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'interna_corporativa'
    OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
    OR d->'campos_permitidos' IS DISTINCT FROM '["CatalogoHuella","CatalogoRef","ClaveIdempotencia","ConMotivoDevolucion","Documento","DocumentoCustodiaRef","DocumentoCustodiaVersion","ExpedienteVersion","FechaPortafirmasDeclarada","FirmaRef","FirmadoHuella","FirmantePrincipalAcreditado","OriginalHuella","OriginalRef","OriginalVersion","PasoOrden","PasoRef","ReciboRef","ReferenciaPortafirmasDeclarada","RegistradaEn","Resultado","Secuencia","SelloTiempoEstado","Via"]'::jsonb
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'AD3-159: consulta de firmas denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  'consulta_firmas_r5_ct',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'AD3-159: la lectura requiere consumo nuevo' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;

DO $acl$
DECLARE f regprocedure:='vec_autorizacion_atestada_v3.consumir_consulta_firmas_r5_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 permitido oid:='vec_contratacion_temporal_propietario'::regrole::oid; x record;
BEGIN
 -- También las ACL por defecto: ningún rol conserva acceso por haber sido
 -- destinatario predeterminado del propietario.
 FOR x IN SELECT DISTINCT a.grantee FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
  WHERE p.oid=f AND a.grantee<>p.proowner LOOP
  EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %s',f::text,CASE WHEN x.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(x.grantee)) END);
 END LOOP;
 EXECUTE format('GRANT EXECUTE ON FUNCTION %s TO vec_contratacion_temporal_propietario',f::text);
 IF (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario'::regrole
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS NOT TRUE
    OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
 THEN RAISE EXCEPTION 'AD3-159: propietario o entorno de fachada incompatible' USING ERRCODE='55000'; END IF;
 FOR x IN SELECT a.grantee,a.privilege_type,a.is_grantable,p.proowner
  FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f LOOP
  IF (x.grantee<>x.proowner AND x.grantee IS DISTINCT FROM permitido) OR x.privilege_type<>'EXECUTE'
    OR (x.grantee=permitido AND x.is_grantable)
  THEN RAISE EXCEPTION 'AD3-159: ACL de fachada abierta' USING ERRCODE='55000'; END IF;
 END LOOP;
END $acl$;
COMMIT;
