\set ON_ERROR_STOP on
-- AD3-157. Fachada preparatoria de la firma VEC de un original CT custodiado.
-- La decisión V3 queda ligada a los bytes canónicos de CT170 y a una audiencia
-- propia. El emisor confiable debe contrastar el certificado del PDF verificado
-- con el certificado del canal sellado antes de emitir la decisión. AD157 no
-- observa ese canal y no afirma realizar esa comparación de forma independiente.
--
-- PARO deliberado: aún no existe una fuente publicada que ate el rol V3 exacto
-- al perfil, cargo, unidad y asignación nominal del paso del catálogo. Por ello
-- esta fachada no consume ninguna decisión ni abre la audiencia en el núcleo.
-- No modifica AD85/CT118 ni concede roles, permisos o eficacia administrativa.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000157',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));

DO $pre$
BEGIN
 IF current_user IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario' THEN
  RAISE EXCEPTION 'AD3-157: PARO clave=rol_sql actual=distinto esperado=propietario_ad3' USING ERRCODE='55000'; END IF;
 IF to_regclass('vec_autorizacion_atestada_v3.clave_capacidad_version') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_solicitud_documental_bolsa_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_circuito_ct_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_firma_externa_ct_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL THEN
  RAISE EXCEPTION 'AD3-157: PARO clave=preimagen_ad155_ad151_ad156 actual=incompleta esperado=instalada' USING ERRCODE='55000'; END IF;
 IF to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_firma_vec_ct_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL THEN
  RAISE EXCEPTION 'AD3-157: PARO clave=fachada_ya_instalada actual=true esperado=false' USING ERRCODE='55000'; END IF;
 IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_contratacion_temporal_propietario'
    AND NOT rolcanlogin AND NOT rolsuper AND NOT rolcreatedb AND NOT rolcreaterole
    AND NOT rolreplication AND NOT rolbypassrls) THEN
  RAISE EXCEPTION 'AD3-157: PARO clave=rol_ct_propietario actual=incompatible esperado=cerrado' USING ERRCODE='55000'; END IF;
 IF NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=
    'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
    AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
    AND p.prosecdef AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
    AND strpos(p.prosrc,'firma_externa_documento_ct')>0
    AND strpos(p.prosrc,'ct_circuito_consultar')>0
    AND strpos(p.prosrc,'firma_vec_documento_ct')=0) THEN
  RAISE EXCEPTION 'AD3-157: PARO clave=nucleo_post_ad156 actual=divergente esperado=nominal' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_firma_vec_ct_v3_atestada(
 p_solicitud text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,
 consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET lock_timeout='2s' AS $f$
DECLARE s jsonb; c jsonb; d jsonb; material_h text; contexto_h text; recurso text;
BEGIN
 IF p_solicitud IS NULL OR octet_length(p_solicitud) NOT BETWEEN 2 AND 65536
    OR p_capacidad IS NULL OR octet_length(p_capacidad) NOT BETWEEN 1 AND 65536
    OR p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 1 AND 524288 THEN
  RAISE EXCEPTION 'AD3-157: material inválido' USING ERRCODE='22023'; END IF;
 BEGIN
  s:=p_solicitud::jsonb;
  c:=convert_from(p_capacidad,'UTF8')::jsonb;
  d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception THEN
  RAISE EXCEPTION 'AD3-157: material inválido' USING ERRCODE='22023'; END;
 IF jsonb_typeof(s) IS DISTINCT FROM 'object' OR jsonb_typeof(c) IS DISTINCT FROM 'object'
    OR jsonb_typeof(d) IS DISTINCT FROM 'object'
    OR (SELECT count(*) FROM jsonb_object_keys(s))<>43
    OR NOT (s ?& ARRAY[
     'Via','OrganizacionRef','ExpedienteRef','VersionExpediente','Documento','CatalogoRef','CatalogoHuella',
     'PasoRef','PasoOrden','Secuencia','HistoriaRevision','HistoriaHuella',
     'OriginalRef','OriginalVersion','OriginalHuella','FirmadoHuella',
     'CertificadoHuella','FirmanteRef','FirmantePrincipalRef','PerfilFirmanteRef','CargoFirmante',
     'UnidadFirmanteRef','PerfilActivoFirmanteRef','PuestoFirmanteRef','AmbitoFirmanteRef','AsignacionFirmanteRef',
     'AsignacionFirmanteVersion','AsignacionFirmanteHuella','VersionRolFirmanteRef',
     'VersionRolFirmanteHuella','ControlVigenciaFirmanteRef','ControlVigenciaFirmanteRevision',
     'ControlVigenciaFirmanteHuella','AsignacionVigenteDesde','AsignacionVigenteHasta',
     'ActoCompetenciaRef','DelegacionRef','PoliticaVerificacion',
     'RevocacionEstado','SelloTiempoEstado','ClaveIdempotencia','DocumentoCustodiaRef','DocumentoCustodiaVersion'])
    OR s->>'Via' IS DISTINCT FROM 'certificado_vec'
    OR s->>'PoliticaVerificacion' IS DISTINCT FROM 'politica:vec:firma:verificacion-autonoma:v1'
    OR s->>'RevocacionEstado' IS DISTINCT FROM 'vigente'
    OR s->>'SelloTiempoEstado' IS NULL
    OR s->>'SelloTiempoEstado' NOT IN ('no_presente','valido','no_comprobado')
    OR s->>'OrganizacionRef' IS NULL OR s->>'OrganizacionRef' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
    OR s->>'ClaveIdempotencia' IS NULL OR s->>'ClaveIdempotencia' !~ '^[A-Za-z0-9][A-Za-z0-9._-]{15,63}$'
    OR s->>'FirmantePrincipalRef' IS NULL OR s->>'FirmantePrincipalRef' !~ '^per_[A-Za-z0-9_-]{2,159}$'
    OR s->>'CertificadoHuella' IS NULL OR s->>'CertificadoHuella' !~ '^[0-9a-f]{64}$'
    OR s->>'FirmanteRef' IS DISTINCT FROM 'ref:'||(s->>'CertificadoHuella')
    OR s->>'OriginalRef' IS NULL OR s->>'OriginalRef' !~ '^ref:[0-9a-f]{64}$'
    OR s->>'OriginalHuella' IS NULL OR s->>'OriginalHuella' !~ '^[0-9a-f]{64}$'
    OR s->>'FirmadoHuella' IS NULL OR s->>'FirmadoHuella' !~ '^[0-9a-f]{64}$'
    OR s->>'CatalogoHuella' IS NULL OR s->>'CatalogoHuella' !~ '^[0-9a-f]{64}$'
    OR jsonb_typeof(s->'HistoriaRevision') IS DISTINCT FROM 'number'
    OR s->>'HistoriaRevision' !~ '^(0|[1-9][0-9]{0,15})$'
    OR (s->>'HistoriaRevision')::numeric>9007199254740991::numeric
    OR s->>'HistoriaHuella' IS NULL OR s->>'HistoriaHuella' !~ '^[0-9a-f]{64}$'
    OR s->>'HistoriaHuella' = repeat('0',64)
    OR s->>'AsignacionFirmanteHuella' IS NULL OR s->>'AsignacionFirmanteHuella' !~ '^[0-9a-f]{64}$'
    OR s->>'VersionRolFirmanteHuella' IS NULL OR s->>'VersionRolFirmanteHuella' !~ '^[0-9a-f]{64}$'
    OR s->>'ControlVigenciaFirmanteHuella' IS NULL OR s->>'ControlVigenciaFirmanteHuella' !~ '^[0-9a-f]{64}$'
    OR jsonb_typeof(s->'ControlVigenciaFirmanteRevision') IS DISTINCT FROM 'number'
    OR s->>'ControlVigenciaFirmanteRevision' !~ '^[1-9][0-9]{0,15}$'
    OR (s->>'ControlVigenciaFirmanteRevision')::numeric>9007199254740991::numeric
    OR s->>'ControlVigenciaFirmanteRef' IS DISTINCT FROM s->>'VersionRolFirmanteRef'
    OR nullif(s->>'PerfilFirmanteRef','') IS NULL
    OR nullif(s->>'CargoFirmante','') IS NULL
    OR nullif(s->>'UnidadFirmanteRef','') IS NULL
    OR nullif(s->>'PerfilActivoFirmanteRef','') IS NULL
    OR nullif(s->>'VersionRolFirmanteRef','') IS NULL
    OR nullif(s->>'AsignacionFirmanteRef','') IS NULL
    OR nullif(s->>'DocumentoCustodiaRef','') IS NULL THEN
  RAISE EXCEPTION 'AD3-157: material VEC inválido' USING ERRCODE='22023'; END IF;
 material_h:=encode(sha256(convert_to(p_solicitud,'UTF8')),'hex');
 recurso:='operacion-firma-vec-ct:'||(s->>'ClaveIdempotencia');
 contexto_h:=encode(sha256(convert_to('{"ambitos":{"organizacion_ref":"'||(s->>'OrganizacionRef')||
  '"},"atributos":{"material_sha256":"'||material_h||'"}}','UTF8')),'hex');
 IF c->>'operacion' IS DISTINCT FROM 'contratacion_temporal.documento.firma_vec.registrar'
    OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_contratacion_temporal.firma_vec.v1'
    OR d->>'accion' IS DISTINCT FROM c->>'operacion'
    OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'firma_vec_documento_contratacion_temporal'
    OR d->>'finalidad' IS DISTINCT FROM 'gestionar_contratacion_temporal'
    OR d->>'recurso_ref' IS DISTINCT FROM recurso OR c->>'efecto_ref' IS DISTINCT FROM recurso
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM contexto_h
    OR c->>'huella_efecto_sha256' IS DISTINCT FROM contexto_h
    OR c->>'huella_decision_sha256' IS DISTINCT FROM encode(sha256(p_decision),'hex')
    OR d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'interna_corporativa'
    OR d->>'principal_id' IS DISTINCT FROM s->>'FirmantePrincipalRef'
    OR d->>'perfil_activo_ref' IS DISTINCT FROM s->>'PerfilActivoFirmanteRef'
    OR d->>'version_rol_ref' IS DISTINCT FROM s->>'VersionRolFirmanteRef'
    OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb THEN
  RAISE EXCEPTION 'AD3-157: firma VEC denegada' USING ERRCODE='42501'; END IF;
 -- Una concesión V3 válida aún no prueba que su rol sea el perfil de cargo
 -- publicado para este paso. Tampoco hay atestación SQL independiente del
 -- certificado del canal. La comparación nominal pertenece al emisor sellado.
 RAISE EXCEPTION 'AD3-157: PARO clave=vinculo_rol_cargo_publicado actual=ausente esperado=atestacion_exacta' USING ERRCODE='55000';
END $f$;

REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_firma_vec_ct_v3_atestada(
 text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_firma_vec_ct_v3_atestada(
 text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_contratacion_temporal_propietario;

DO $acl$
DECLARE f regprocedure:='vec_autorizacion_atestada_v3.registrar_y_consumir_firma_vec_ct_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 x record;
BEGIN
 FOR x IN SELECT DISTINCT a.grantee FROM pg_proc p
  CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
  WHERE p.oid=f AND a.grantee<>p.proowner LOOP
  EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %s',f::text,
   CASE WHEN x.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(x.grantee)) END);
 END LOOP;
 EXECUTE format('GRANT EXECUTE ON FUNCTION %s TO vec_contratacion_temporal_propietario',f::text);
 IF (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario'::regrole
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS NOT TRUE
    OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
    OR EXISTS (SELECT 1 FROM pg_proc p
       CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
       WHERE p.oid=f AND (a.privilege_type<>'EXECUTE' OR a.is_grantable
          OR a.grantee NOT IN (p.proowner,'vec_contratacion_temporal_propietario'::regrole))) THEN
  RAISE EXCEPTION 'AD3-157: PARO clave=acl_fachada actual=abierta esperado=propietario_ct' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
