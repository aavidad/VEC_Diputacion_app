\set ON_ERROR_STOP on
-- AD3-159: consumidor nominal de la consulta CT170 R5 de firmas por
-- expediente, versión, documento, candidato verificado, clave y paso.
-- El material canónico de ocho campos queda ligado por la huella del recurso
-- que CT170 valida antes de consumir esta fachada de diez argumentos.
-- La proyección concede 27 campos por fila y dos indicadores de cabecera;
-- no concede registro ni lectura bruta de identidad o certificado.
-- La fachada exige consumo nuevo incluso en replay y solo la invoca el propietario de CT.
-- La decisión liga operación, recurso, finalidad, campos y efecto exactos.
-- Se instala en serie con cualquier otra reescritura del núcleo: toma el
-- cerrojo común antes de leer su preimagen POST-AD162. Sin DOWN tras historia.
-- CT170 reconstruye el material canónico de ocho campos y su contexto de
-- recurso con el único ámbito organizacion_ref antes de invocar esta fachada.
-- No habilita escritura V1 ni amplía la proyección con actor o certificado.
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
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_solicitud_documental_bolsa_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_circuito_ct_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_firma_externa_ct_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_firma_vec_ct_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 THEN RAISE EXCEPTION 'AD3-159: PARO clave=preimagen_ad125_ad155_ad151_ad156_ad157 actual=incompleta esperado=instalada' USING ERRCODE='55000'; END IF;
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
                   AND NOT rolcanlogin AND NOT rolsuper AND NOT rolcreatedb
                   AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls)
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_contratacion_temporal_ejecutor'
                   AND NOT rolcanlogin AND NOT rolsuper AND NOT rolcreatedb
                   AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls)
 THEN RAISE EXCEPTION 'AD3-159: PARO clave=roles_ct actual=incompatible esperado=cerrados' USING ERRCODE='55000'; END IF;
END $pre$;

-- La instalación valida sólo el corte estructural POST-AD162. Los roles y
-- asignaciones funcionales se provisionan por la autoridad central después
-- del SQL y se revalidan en cada consumo; su ausencia siempre deniega.

DO $fachadas_ad162$
DECLARE v record; p record; propietario oid:='vec_autorizacion_atestada_v3_propietario'::regrole;
 permitido oid:='vec_contratacion_temporal_propietario'::regrole;
BEGIN
 FOR v IN SELECT * FROM (VALUES
  ('vec_autorizacion_atestada_v3.registrar_y_consumir_firma_verificada_ct_v2_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','1fd2930ccae9bfdf78ffc109f3bdbc4d0819cd87005f93c7a88eb62c3ed7ff6d','daa83b0c5365166521c60f7260f6626f9de007930c4c6fb446a93532e15dbb77'),
  ('vec_autorizacion_atestada_v3.consumir_consulta_firmas_r5_ct_v2_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','91dbe9ebd4f008fdccb242db16df16b91a6b9e401530cc6738326e4347b075c1','c5b819ec5bba24e1aed4a0ae06b74fffe2833f0c6a5da65cb2d814f7afa24492')
 ) AS v(firma,def_sha,src_sha) LOOP
  SELECT x.* INTO p FROM pg_proc x WHERE x.oid=to_regprocedure(v.firma);
  IF NOT FOUND OR p.proowner<>propietario OR NOT p.prosecdef
     OR p.prokind<>'f' OR p.provolatile<>'v' OR p.proparallel<>'u'
     OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
     OR encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex') IS DISTINCT FROM v.def_sha
     OR encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM v.src_sha
     OR (SELECT count(*) FROM aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))))<>2
     OR NOT EXISTS (SELECT 1 FROM aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
         WHERE a.grantee=permitido AND a.grantor=propietario AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
     OR EXISTS (SELECT 1 FROM aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
         WHERE a.grantee NOT IN (propietario,permitido) OR a.grantor<>propietario
           OR a.privilege_type<>'EXECUTE' OR a.is_grantable) THEN
   RAISE EXCEPTION 'AD3-159: fachada AD162 incompatible' USING ERRCODE='55000'; END IF;
 END LOOP;
END $fachadas_ad162$;

DO $nucleo$
DECLARE
 f oid:='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 original text; fuente text; nuevo text; actual text; meta jsonb; deps jsonb; deps_compartidas jsonb; acl aclitem[];
 propietario oid; config text[]; definidora boolean;
 -- Postimagen completa POST-AD162 capturada en PostgreSQL 18; conserva V2.
 esperada_def_sha256 text:='8ee729ac5b3740fadc260dbd2a686cc03d411abde71e13b2b213330696cbabbf';
 esperada_fuente_sha256 text:='ed4fdb20579f858900aa6382a2c65e65312a12fb97f9d5b03a598be7422c337a';
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
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["CatalogoHuella","CatalogoRef","ClaveIdempotencia","CoincideFirmanteCandidato","CoincideFirmanteEnOtroPaso","ConMotivoDevolucion","Documento","DocumentoCustodiaRef","DocumentoCustodiaVersion","ExpedienteVersion","FechaPortafirmasDeclarada","FirmaRef","FirmadoHuella","FirmantePrincipalAcreditado","HistoriaHuella","HistoriaRevision","HistoriaSeparacionAcreditada","OriginalHuella","OriginalRef","OriginalVersion","PasoOrden","PasoRef","ReciboRef","ReferenciaPortafirmasDeclarada","RegistradaEn","Resultado","Secuencia","SelloTiempoEstado","Via"]'::jsonb
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
 IF encode(sha256(convert_to(original,'UTF8')),'hex') IS DISTINCT FROM esperada_def_sha256
    OR encode(sha256(convert_to(fuente,'UTF8')),'hex') IS DISTINCT FROM esperada_fuente_sha256
    OR propietario<>'vec_autorizacion_atestada_v3_propietario'::regrole OR NOT definidora
    OR config IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
    OR NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=f
         AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
         AND p.prokind='f' AND p.provolatile='v' AND p.proparallel='u'
         AND p.prosecdef AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s'])
    OR EXISTS (SELECT 1 FROM pg_database db
         CROSS JOIN LATERAL aclexplode(coalesce(db.datacl,acldefault('d',db.datdba))) a
         WHERE db.datname=current_database() AND a.grantee=0 AND a.privilege_type='TEMPORARY')
    OR EXISTS (SELECT 1 FROM pg_roles r WHERE left(r.rolname,4)='vec_' AND r.rolcanlogin
         AND has_database_privilege(r.oid,current_database(),'TEMPORARY'))
    OR NOT EXISTS (SELECT 1 FROM aclexplode(coalesce(acl,acldefault('f',propietario))) a
         WHERE a.grantee=propietario AND a.grantor=propietario
           AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
    OR EXISTS (SELECT 1 FROM aclexplode(coalesce(acl,acldefault('f',propietario))) a
         WHERE a.grantee<>propietario OR a.grantor<>propietario
            OR a.privilege_type<>'EXECUTE' OR a.is_grantable)
    OR deps IS DISTINCT FROM jsonb_build_array(
         jsonb_build_object('classid','pg_proc'::regclass::oid,'objid',f::oid,'objsubid',0,
           'refclassid','pg_language'::regclass::oid,
           'refobjid',(SELECT oid FROM pg_language WHERE lanname='plpgsql'),
           'refobjsubid',0,'deptype','n'),
         jsonb_build_object('classid','pg_proc'::regclass::oid,'objid',f::oid,'objsubid',0,
           'refclassid','pg_namespace'::regclass::oid,
           'refobjid','vec_autorizacion_atestada_v3'::regnamespace::oid,
           'refobjsubid',0,'deptype','n'))
    OR deps_compartidas IS DISTINCT FROM jsonb_build_array(
         jsonb_build_object('dbid',(SELECT oid FROM pg_database WHERE datname=current_database()),
           'classid','pg_proc'::regclass::oid,'objid',f::oid,'objsubid',0,
           'refclassid','pg_authid'::regclass::oid,'refobjid',propietario,'deptype','o'))
    OR length(original)-length(replace(original,marca,''))<>length(marca)
    OR strpos(original,'''consulta_firmas_documento_ct''')=0
    OR strpos(original,'''ct_circuito_consultar''')=0
    OR strpos(original,'''firma_externa_documento_ct''')=0
    OR strpos(original,'''consulta_firmas_r5_ct_v2''')=0
    OR strpos(original,'''firma_vec_documento_ct_v2''')=0
    OR strpos(original,'''firma_externa_documento_ct_v2''')=0
    OR strpos(original,'vec_contratacion_temporal_ejecutor')=0
    OR strpos(original,'''consulta_firmas_r5_ct''')<>0
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
DECLARE d text; original text; nueva text; audiencia text:='vec_contratacion_temporal.firmas_r5.consultar.v1';
 esperada_check_sha256 text:='5cb9438b926cd0f8a5afee30f11bcf9a399f7c17786509089816bab639216585';
BEGIN
 SELECT regexp_replace(pg_get_constraintdef(c.oid,true),'\s+',' ','g') INTO STRICT d
 FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
 AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 IF encode(sha256(convert_to(d,'UTF8')),'hex') IS DISTINCT FROM esperada_check_sha256
    OR strpos(d,'CHECK (audiencia_consumo = ANY (ARRAY[')<>1 OR right(d,3)<>']))'
    OR strpos(d,'vec_contratacion_temporal.lectura_reincorporacion_titular.v1')=0
    OR strpos(d,'''vec_contratacion_temporal.firmas_r5.consultar.v2''::text')=0
    OR strpos(d,quote_literal(audiencia))<>0
 THEN RAISE EXCEPTION 'AD3-159: audiencias previas incompatibles' USING ERRCODE='55000'; END IF;
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version
  DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 original:=d;
 nueva:=left(d,length(d)-3)||', '||quote_literal(audiencia)||'::text]))';
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||nueva;
 SELECT pg_get_constraintdef(c.oid,true) INTO STRICT d FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
   AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 IF d IS DISTINCT FROM nueva
    OR left(d,length(d)-length(', '||quote_literal(audiencia)||'::text]))'))||']))' IS DISTINCT FROM original THEN
  RAISE EXCEPTION 'AD3-159: audiencias previas alteradas' USING ERRCODE='55000'; END IF;
END $audiencias$;

-- Fachada única: la lectura queda ligada al expediente, 29 campos exactos y
-- decisión sin obligaciones. CT170 liga los ocho campos del material a la
-- capacidad y consume antes de leer, también en ausencia y replay.
CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_consulta_firmas_r5_ct_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; x record;
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR current_setting('TimeZone')<>'UTC' THEN
  RAISE EXCEPTION 'AD3-159: transacción de consulta denegada' USING ERRCODE='42501'; END IF;
 IF p_capacidad IS NULL OR octet_length(p_capacidad) NOT BETWEEN 1 AND 65536
    OR p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 1 AND 524288 THEN
  RAISE EXCEPTION 'AD3-159: material de consulta inválido' USING ERRCODE='22023'; END IF;
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'AD3-159: material de consulta inválido' USING ERRCODE='22023'; END;
 IF jsonb_typeof(c) IS DISTINCT FROM 'object' OR jsonb_typeof(d) IS DISTINCT FROM 'object'
    OR d->>'concedida' IS DISTINCT FROM 'true'
    OR c->>'decision_ref' IS DISTINCT FROM d->>'decision_ref'
    OR c->>'huella_decision_sha256' IS DISTINCT FROM encode(sha256(p_decision),'hex')
    OR nullif(d->>'perfil_activo_ref','') IS NULL
    OR nullif(d->>'version_rol_ref','') IS NULL
    OR d->>'recurso_ref' IS NULL OR d->>'recurso_ref' !~ '^expediente:[A-Za-z0-9._:/#-]{2,149}$'
    OR d->>'contexto_recurso_huella_sha256' IS NULL OR d->>'contexto_recurso_huella_sha256' !~ '^[0-9a-f]{64}$'
    OR c->>'operacion' IS DISTINCT FROM 'contratacion_temporal.documento.firmas_r5.consultar'
    OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_contratacion_temporal.firmas_r5.consultar.v1'
    OR d->>'accion' IS DISTINCT FROM c->>'operacion'
    OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'expediente_contratacion_temporal'
    OR d->>'finalidad' IS DISTINCT FROM 'gestionar_contratacion_temporal'
    OR d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'interna_corporativa'
    OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
    OR d->'campos_permitidos' IS DISTINCT FROM '["CatalogoHuella","CatalogoRef","ClaveIdempotencia","CoincideFirmanteCandidato","CoincideFirmanteEnOtroPaso","ConMotivoDevolucion","Documento","DocumentoCustodiaRef","DocumentoCustodiaVersion","ExpedienteVersion","FechaPortafirmasDeclarada","FirmaRef","FirmadoHuella","FirmantePrincipalAcreditado","HistoriaHuella","HistoriaRevision","HistoriaSeparacionAcreditada","OriginalHuella","OriginalRef","OriginalVersion","PasoOrden","PasoRef","ReciboRef","ReferenciaPortafirmasDeclarada","RegistradaEn","Resultado","Secuencia","SelloTiempoEstado","Via"]'::jsonb
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'AD3-159: consulta de firmas denegada' USING ERRCODE='42501'; END IF;
 -- El selector siguiente es técnico: el perfil activo real viene del vínculo
 -- autenticado. El núcleo registra y revalida rol publicado, asignación, actor,
 -- perfil y catálogo vigentes en esta misma transacción, antes de autorizar lectura.
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  'consulta_firmas_r5_ct',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE OR x.decision_ref IS DISTINCT FROM d->>'decision_ref'
    OR x.efecto_ref IS DISTINCT FROM d->>'recurso_ref'
    OR x.huella_efecto_sha256 IS DISTINCT FROM d->>'contexto_recurso_huella_sha256'
    OR x.consumida_en IS NULL OR x.auditoria_ref IS NULL
    OR x.consumo_huella_sha256 IS NULL OR x.consumo_huella_sha256 !~ '^[0-9a-f]{64}$'
 THEN RAISE EXCEPTION 'AD3-159: la lectura requiere consumo nuevo vinculado' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_consulta_firmas_r5_ct_v3_atestada(
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;

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
    OR NOT has_function_privilege(permitido,f,'EXECUTE')
    OR NOT has_schema_privilege(permitido,'vec_autorizacion_atestada_v3','USAGE')
    OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f)<>2
 THEN RAISE EXCEPTION 'AD3-159: propietario o entorno de fachada incompatible' USING ERRCODE='55000'; END IF;
 FOR x IN SELECT a.grantee,a.grantor,a.privilege_type,a.is_grantable,p.proowner
  FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f LOOP
  IF x.grantee NOT IN (x.proowner,permitido) OR x.grantor<>x.proowner
     OR x.privilege_type<>'EXECUTE' OR x.is_grantable
  THEN RAISE EXCEPTION 'AD3-159: ACL de fachada abierta' USING ERRCODE='55000'; END IF;
 END LOOP;
END $acl$;
COMMIT;
