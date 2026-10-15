\set ON_ERROR_STOP on
-- AUT97. Corrige el alias de la asignación en la validación nominal de firma CT.
-- Fuente: definición instalada tras AUT93; la variable local a conserva su nombre.
-- Preimagen pg_get_functiondef: fcbb8e96de2e280e99805e2b4111f794f49ccdcb9c8bf35a7ad9d45e6263e95c
-- Postimagen pg_get_functiondef: 5d4b0799ab66aabdbd667c914a2463a574477861a565c28b7beb594c40316903
BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion:migracion:000097',0));
DO $pre$
DECLARE f pg_proc%ROWTYPE; actual text;
BEGIN
 IF current_setting('server_version_num')::integer < 180000
    OR current_setting('server_version_num')::integer >= 190000 THEN
  RAISE EXCEPTION 'AUT97: PostgreSQL 18 requerido' USING ERRCODE='55000';
 END IF;
 SELECT * INTO f FROM pg_proc WHERE oid=to_regprocedure('vec_autorizacion.validar_competencia_nominal_firmante_ct_v1(bytea,jsonb,jsonb)');
 IF NOT FOUND THEN
  RAISE EXCEPTION 'AUT97: función instalada ausente' USING ERRCODE='55000';
 END IF;
 actual:=encode(sha256(convert_to(pg_get_functiondef(f.oid),'UTF8')),'hex');
 IF actual IS DISTINCT FROM 'fcbb8e96de2e280e99805e2b4111f794f49ccdcb9c8bf35a7ad9d45e6263e95c'
  OR f.proowner IS DISTINCT FROM 'vec_autorizacion_propietario'::regrole
  OR f.proacl::text IS DISTINCT FROM '{vec_autorizacion_propietario=X/vec_autorizacion_propietario}'
  OR f.prosecdef IS DISTINCT FROM true
  OR f.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','lock_timeout=2s','TimeZone=UTC'] THEN
  RAISE EXCEPTION 'AUT97: PARO preimagen/propietario/ACL/configuración incompatibles; huella=%',actual USING ERRCODE='55000';
 END IF;
END $pre$;

CREATE OR REPLACE FUNCTION vec_autorizacion.validar_competencia_nominal_firmante_ct_v1(p_contexto_nominal bytea, p_relacion_ct jsonb, p_consumo_v3 jsonb)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET lock_timeout TO '2s'
 SET "TimeZone" TO 'UTC'
AS $function$
DECLARE c jsonb; ident jsonb; comp jsonb; per jsonb; rec jsonb; ct jsonb;
 ca jsonb; org_ca jsonb; prueba_org jsonb; fuente_personal jsonb; v3 jsonb; a record; rol record;
 catalogo record;
 concedida boolean; amb_org boolean; amb_unidad boolean;
 instante timestamptz(6); fecha_historica timestamptz(6); efecto text;
BEGIN
 IF current_setting('transaction_isolation') <> 'serializable'
  OR current_setting('transaction_read_only') <> 'off'
  OR pg_is_in_recovery() OR current_setting('TimeZone') <> 'UTC' THEN
  RAISE EXCEPTION 'aut32_transaccion_no_admitida' USING ERRCODE = '42501';
 END IF;
 IF p_contexto_nominal IS NULL OR octet_length(p_contexto_nominal) NOT BETWEEN 512 AND 32768
  OR jsonb_typeof(p_relacion_ct) IS DISTINCT FROM 'object'
  OR jsonb_typeof(p_consumo_v3) IS DISTINCT FROM 'object' THEN
  RAISE EXCEPTION 'aut32_contexto_no_admitido' USING ERRCODE = '42501';
 END IF;
 c := convert_from(p_contexto_nominal,'UTF8')::jsonb;
 ident := c->'identidad'; comp := c->'competencia'; per := c->'personal';
 rec := c->'recurso'; ct := c->'relacion_ct';
 IF c->>'esquema' IS DISTINCT FROM 'vec.competencia-firmante.historica.v1'
  OR jsonb_typeof(ident) IS DISTINCT FROM 'object'
  OR jsonb_typeof(comp) IS DISTINCT FROM 'object'
  OR jsonb_typeof(per) IS DISTINCT FROM 'object'
  OR jsonb_typeof(rec) IS DISTINCT FROM 'object'
  OR jsonb_typeof(ct) IS DISTINCT FROM 'object'
  OR (SELECT count(*) FROM jsonb_object_keys(c)) <> 13
  OR c->>'accion' IS NULL OR c->>'finalidad' IS NULL
  OR vec_autorizacion.texto_positivo_valido(c->>'accion',256) IS NOT TRUE
  OR vec_autorizacion.texto_positivo_valido(c->>'finalidad',512) IS NOT TRUE
  OR vec_autorizacion.texto_positivo_valido(comp->>'rol_id',128) IS NOT TRUE
  OR vec_autorizacion.texto_positivo_valido(comp->>'perfil_esperado_ref',512) IS NOT TRUE
  OR vec_autorizacion.texto_positivo_valido(comp->>'perfil_activo_ref',512) IS NOT TRUE
  OR vec_autorizacion.texto_positivo_valido(comp->>'modulo_id',128) IS NOT TRUE
  OR vec_autorizacion.texto_positivo_valido(comp->>'tipo_recurso',128) IS NOT TRUE
  OR vec_autorizacion.texto_positivo_valido(comp->>'recurso_ref',512) IS NOT TRUE
  OR vec_autorizacion.instante_utc_microsegundo_valido(c->>'fecha_historica') IS NOT TRUE
  OR rec->>'organizacion_ref' IS NULL OR rec->>'unidad_ref' IS NULL
  OR rec->>'expediente_ref' IS NULL OR rec->>'documento_ref' IS NULL
  OR rec->>'recurso_autorizable_ref' IS NULL
  OR (rec->>'recurso_contexto_sha256' ~ '^[0-9a-f]{64}$') IS NOT TRUE
  OR (rec->>'pdf_raiz_sha256' ~ '^[0-9a-f]{64}$') IS NOT TRUE
  OR (rec->>'pdf_firmado_sha256' ~ '^[0-9a-f]{64}$') IS NOT TRUE
  OR rec->>'recurso_autorizable_ref' IS DISTINCT FROM rec->>'documento_ref'
  OR rec#>>'{original,huella_sha256}' IS DISTINCT FROM rec->>'pdf_raiz_sha256'
  OR rec#>>'{firmado,huella_sha256}' IS DISTINCT FROM rec->>'pdf_firmado_sha256'
  OR ((rec->>'numero_firmas')::numeric BETWEEN 1 AND 9007199254740991) IS NOT TRUE
  OR ((rec->>'numero_firmas')::numeric=1 AND rec->'entrada_revision' IS DISTINCT FROM 'null'::jsonb)
  OR ((rec->>'numero_firmas')::numeric>1 AND jsonb_typeof(rec->'entrada_revision') IS DISTINCT FROM 'object')
  OR c#>>'{motivo,catalogo_id}' IS NULL
  OR ((c#>>'{motivo,catalogo_version}')::numeric BETWEEN 1 AND 9007199254740991) IS NOT TRUE
  OR (c#>>'{motivo,catalogo_huella_sha256}' ~ '^[0-9a-f]{64}$') IS NOT TRUE
  OR c#>>'{motivo,entrada_clave}' IS NULL
  OR ((c#>>'{circuito,version}')::numeric BETWEEN 1 AND 9007199254740991) IS NOT TRUE
  OR (c#>>'{circuito,huella_sha256}' ~ '^[0-9a-f]{64}$') IS NOT TRUE
  OR c->>'paso_ref' IS NULL
  OR ((c->>'paso_orden')::numeric BETWEEN 1 AND 9007199254740991) IS NOT TRUE THEN
  RAISE EXCEPTION 'aut32_contexto_no_admitido' USING ERRCODE = '42501';
 END IF;
 -- CT174 procede del propietario CT y conserva el puntero bajo FOR SHARE.
 IF p_relacion_ct->>'esquema' IS DISTINCT FROM 'vec.contratacion-temporal.relacion-unidad-expediente.v1'
  OR ct->>'expediente_ref' IS DISTINCT FROM rec->>'expediente_ref'
  OR ct->>'unidad_ref' IS DISTINCT FROM rec->>'unidad_ref'
  OR p_relacion_ct->>'organizacion_ref' IS DISTINCT FROM rec->>'organizacion_ref'
  OR p_relacion_ct->>'expediente_ref' IS DISTINCT FROM ct->>'expediente_ref'
  OR p_relacion_ct->>'unidad_ref' IS DISTINCT FROM ct->>'unidad_ref'
  OR p_relacion_ct->>'unidad_ref_esperada' IS DISTINCT FROM ct->>'unidad_ref'
  OR p_relacion_ct->>'unidad_snapshot_solicitado_ref' IS DISTINCT FROM ct->>'unidad_ref'
  OR p_relacion_ct->>'operacion_origen_ref' IS DISTINCT FROM ct->>'origen_ref'
  OR (p_relacion_ct->>'version_origen_vinculo')::numeric IS DISTINCT FROM (ct->>'origen_version')::numeric
  OR p_relacion_ct->>'prueba_snapshot_origen_huella_sha256' IS DISTINCT FROM ct->>'prueba_snapshot_sha256'
  OR p_relacion_ct->>'evento_asignacion_ref' IS DISTINCT FROM ct->>'evento_ref'
  OR p_relacion_ct->>'evento_payload_huella_sha256' IS DISTINCT FROM ct->>'evento_huella_sha256'
  OR (p_relacion_ct->>'asignacion_confirmada_en')::timestamptz
     IS DISTINCT FROM (ct->>'confirmada_en')::timestamptz THEN
  RAISE EXCEPTION 'aut32_relacion_ct_no_admitida' USING ERRCODE = '42501';
 END IF;
 fecha_historica := (c->>'fecha_historica')::timestamptz;
 instante := clock_timestamp();
 IF fecha_historica IS NULL OR fecha_historica > instante
  OR p_relacion_ct->>'tipo_evento_origen' IS DISTINCT FROM 'contratacion_temporal.asignacion_confirmada'
  OR (ct->>'confirmada_en')::timestamptz > fecha_historica THEN
  RAISE EXCEPTION 'aut32_relacion_ct_no_admitida' USING ERRCODE = '42501';
 END IF;
 -- La correlación del consumo identifica el efecto CT. Su auditoría pertenece
 -- al núcleo V3; AUT sólo conserva sus referencias, no crea otra cadena.
 efecto := p_consumo_v3->>'efecto_ref';
 IF vec_autorizacion.texto_positivo_valido(efecto,512) IS NOT TRUE
  OR vec_autorizacion.texto_positivo_valido(p_consumo_v3->>'decision_ref',512) IS NOT TRUE
  OR vec_autorizacion.texto_positivo_valido(p_consumo_v3->>'auditoria_ref',512) IS NOT TRUE
  OR (p_consumo_v3->>'consumo_huella_sha256' ~ '^[0-9a-f]{64}$') IS NOT TRUE
  OR (p_consumo_v3->>'huella_efecto_sha256' ~ '^[0-9a-f]{64}$') IS NOT TRUE
  OR p_consumo_v3->'consumo_nuevo' IS DISTINCT FROM 'true'::jsonb THEN
  RAISE EXCEPTION 'aut32_consumo_v3_no_admitido' USING ERRCODE = '42501';
 END IF;
 v3 := vec_autorizacion_atestada_v3.comprobar_consumo_firma_ct_v1(p_consumo_v3);
 IF v3->>'decision_ref' IS DISTINCT FROM p_consumo_v3->>'decision_ref'
  OR v3->>'efecto_ref' IS DISTINCT FROM efecto
  OR v3->>'huella_efecto_sha256' IS DISTINCT FROM p_consumo_v3->>'huella_efecto_sha256'
  OR v3->>'consumo_huella_sha256' IS DISTINCT FROM p_consumo_v3->>'consumo_huella_sha256'
  OR v3->>'auditoria_ref' IS DISTINCT FROM p_consumo_v3->>'auditoria_ref' THEN
  RAISE EXCEPTION 'aut32_consumo_v3_no_admitido' USING ERRCODE = '42501';
 END IF;
 ca := vec_contexto_actor_v1.leer_revalidar_certificado_firmante_ct_v2(ident->>'certificado_der_sha256');
 org_ca := ca->'organizacion_destino';
 IF ca->>'esquema' IS DISTINCT FROM 'vec.contexto-actor.certificado-firmante-ct.v2'
  OR ca->>'estado' IS DISTINCT FROM 'vigente'
  OR ca->>'certificado_der_sha256' IS DISTINCT FROM ident->>'certificado_der_sha256'
  OR ca->>'persona_ref' IS DISTINCT FROM ident->>'persona_ref'
  OR ca->'persona' IS DISTINCT FROM ident->'persona'
  OR ca->'cuenta' IS DISTINCT FROM ident->'cuenta'
  OR ca#>>'{vinculo_cuenta_persona,referencia}' IS DISTINCT FROM ident#>>'{vinculo_cuenta_persona,referencia}'
  OR (ca#>>'{vinculo_cuenta_persona,version}')::numeric IS DISTINCT FROM (ident#>>'{vinculo_cuenta_persona,version}')::numeric
  OR ca#>>'{vinculo_cuenta_persona,huella_sha256}' IS DISTINCT FROM ident#>>'{vinculo_cuenta_persona,huella_sha256}'
  OR ca#>>'{vinculo_cuenta_persona,cuenta_ref}' IS DISTINCT FROM ident->>'cuenta_persona_cuenta_ref'
  OR ca#>>'{vinculo_cuenta_persona,persona_ref}' IS DISTINCT FROM ident->>'cuenta_persona_persona_ref'
  OR ca#>>'{persona,referencia}' IS DISTINCT FROM ident->>'persona_ref'
  OR ca->'vinculo_certificado'->>'referencia' IS DISTINCT FROM ident#>>'{vinculo_certificado,referencia}'
  OR (ca#>>'{vinculo_certificado,version}')::numeric IS DISTINCT FROM (ident#>>'{vinculo_certificado,version}')::numeric
  OR ca#>>'{vinculo_certificado,huella_sha256}' IS DISTINCT FROM ident#>>'{vinculo_certificado,huella_sha256}'
  OR ca#>>'{vinculo_certificado,cuenta_ref}' IS DISTINCT FROM ident#>>'{cuenta,referencia}'
  OR ca#>>'{vinculo_certificado,cuenta_ref}' IS DISTINCT FROM ident->>'vinculo_cuenta_ref'
  OR ca#>>'{vinculo_certificado,persona_ref}' IS DISTINCT FROM ident->>'persona_ref'
  OR ca#>>'{vinculo_certificado,persona_ref}' IS DISTINCT FROM ident->>'vinculo_persona_ref'
  OR ca#>>'{vinculo_certificado,certificado_der_sha256}' IS DISTINCT FROM ident->>'certificado_der_sha256'
  OR ca#>>'{vinculo_certificado,certificado_der_sha256}' IS DISTINCT FROM ident->>'vinculo_der_sha256'
  OR ca#>>'{vinculo_cuenta_persona,cuenta_ref}' IS DISTINCT FROM ident#>>'{cuenta,referencia}'
  OR ca#>>'{vinculo_cuenta_persona,persona_ref}' IS DISTINCT FROM ident->>'persona_ref' THEN
  RAISE EXCEPTION 'aut32_identidad_no_admitida' USING ERRCODE = '42501';
 END IF;
 -- La organización acreditada por CA4 se conserva fuera del canon V1: el
 -- documento CA25 original liga estos metadatos a la huella de su vínculo.
 IF jsonb_typeof(org_ca) IS DISTINCT FROM 'object'
  OR (SELECT count(*) FROM jsonb_object_keys(org_ca)) <> 9
  OR vec_autorizacion.texto_positivo_valido(org_ca->>'organizacion_procedencia_ref',512) IS NOT TRUE
  OR vec_autorizacion.texto_positivo_valido(org_ca->>'vinculo_corporativo_ref',512) IS NOT TRUE
  OR ((org_ca->>'organizacion_version')::numeric BETWEEN 1 AND 9007199254740991) IS NOT TRUE
  OR ((org_ca->>'organizacion_procedencia_version')::numeric BETWEEN 1 AND 9007199254740991) IS NOT TRUE
  OR ((org_ca->>'vinculo_corporativo_version')::numeric BETWEEN 1 AND 9007199254740991) IS NOT TRUE
  OR (org_ca->>'organizacion_huella_sha256' ~ '^[0-9a-f]{64}$') IS NOT TRUE
  OR (org_ca->>'organizacion_procedencia_sha256' ~ '^[0-9a-f]{64}$') IS NOT TRUE
  OR (org_ca->>'vinculo_corporativo_huella_sha256' ~ '^[0-9a-f]{64}$') IS NOT TRUE THEN
  RAISE EXCEPTION 'aut32_organizacion_no_admitida' USING ERRCODE = '42501';
 END IF;
 prueba_org:=vec_autorizacion.comprobar_correspondencia_organizacion_firma_ct_v1(
  rec->>'organizacion_ref',org_ca);
 -- Personal38 recibe únicamente un DTO de consulta. El canon firmado V1 y
 -- todos sus selectores CT siguen íntegros en p_contexto_nominal.
 fuente_personal := vec_personal.leer_revalidar_cargo_ocupante_ct_v1(convert_to(
  jsonb_build_object(
   'esquema',c->'esquema','fecha_historica',c->'fecha_historica',
   'personal',per,'identidad',ident,'accion',c->'accion','finalidad',c->'finalidad',
   'recurso',jsonb_build_object(
    'organizacion_ref',org_ca->>'organizacion_ref',
    'unidad_ref',rec->'unidad_ref','expediente_ref',rec->'expediente_ref',
    'documento_ref',rec->'documento_ref',
    'recurso_autorizable_ref',rec->'recurso_autorizable_ref',
    'modulo_id',rec->'modulo_id','tipo_recurso',rec->'tipo_recurso',
    'recurso_contexto_sha256',rec->'recurso_contexto_sha256',
    'original',rec->'original','pdf_raiz_sha256',rec->'pdf_raiz_sha256',
    'firmado',rec->'firmado','pdf_firmado_sha256',rec->'pdf_firmado_sha256',
    'numero_firmas',rec->'numero_firmas','entrada_revision',rec->'entrada_revision'))::text,'UTF8'));
 IF fuente_personal->>'esquema' IS DISTINCT FROM 'vec.personal.cargo-ocupante.ct.v1'
  OR fuente_personal->'cargo'->>'referencia' IS DISTINCT FROM per#>>'{cargo,referencia}'
  OR (fuente_personal#>>'{cargo,version}')::numeric IS DISTINCT FROM (per#>>'{cargo,version}')::numeric
  OR fuente_personal#>>'{cargo,huella_sha256}' IS DISTINCT FROM per#>>'{cargo,huella_sha256}'
  OR fuente_personal->'enlace_ocupante'->>'referencia' IS DISTINCT FROM per#>>'{enlace_ocupante,referencia}'
  OR (fuente_personal#>>'{enlace_ocupante,version}')::numeric IS DISTINCT FROM (per#>>'{enlace_ocupante,version}')::numeric
  OR fuente_personal#>>'{enlace_ocupante,huella_sha256}' IS DISTINCT FROM per#>>'{enlace_ocupante,huella_sha256}'
  OR fuente_personal->>'ocupante_persona_ref' IS DISTINCT FROM per->>'ocupante_persona_ref'
  OR fuente_personal->>'cargo_ref_enlace' IS DISTINCT FROM per->>'cargo_ref_enlace'
  OR fuente_personal->>'organizacion_ref' IS DISTINCT FROM org_ca->>'organizacion_ref'
  OR fuente_personal->>'unidad_ref' IS DISTINCT FROM rec->>'unidad_ref'
  OR fuente_personal->>'persona_ejerciente_ref' IS DISTINCT FROM ident->>'persona_ref'
  OR vec_autorizacion.texto_positivo_valido(fuente_personal->>'procedencia_ref',512) IS NOT TRUE
  OR vec_autorizacion.texto_positivo_valido(fuente_personal->>'recibo_ref',512) IS NOT TRUE
  OR (per->'delegacion' = 'null'::jsonb AND fuente_personal->>'tipo_ejercicio' IS DISTINCT FROM 'titular')
  OR (per->'delegacion' <> 'null'::jsonb AND
      (fuente_personal->>'tipo_ejercicio' IN
       ('delegacion_competencia','delegacion_firma','suplencia')) IS NOT TRUE)
  OR (fuente_personal->>'cargo_vigente_desde')::timestamptz IS DISTINCT FROM (per->>'cargo_vigente_desde')::timestamptz
  OR (fuente_personal->>'cargo_vigente_hasta')::timestamptz IS DISTINCT FROM (per->>'cargo_vigente_hasta')::timestamptz
  OR (fuente_personal->>'enlace_vigente_desde')::timestamptz IS DISTINCT FROM (per->>'enlace_vigente_desde')::timestamptz
  OR (fuente_personal->>'enlace_vigente_hasta')::timestamptz IS DISTINCT FROM (per->>'enlace_vigente_hasta')::timestamptz
  OR (fuente_personal->'delegacion' IS DISTINCT FROM 'null'::jsonb
      AND per->'delegacion' IS DISTINCT FROM 'null'::jsonb
      AND (fuente_personal#>'{delegacion,acto}' IS DISTINCT FROM per#>'{delegacion,acto}'
       OR fuente_personal#>>'{delegacion,delegante_persona_ref}' IS DISTINCT FROM per#>>'{delegacion,delegante_persona_ref}'
       OR fuente_personal#>>'{delegacion,delegado_persona_ref}' IS DISTINCT FROM per#>>'{delegacion,delegado_persona_ref}'
       OR fuente_personal#>>'{delegacion,cargo_ref}' IS DISTINCT FROM per#>>'{delegacion,cargo_ref}'
       OR (fuente_personal#>>'{delegacion,vigente_desde}')::timestamptz IS DISTINCT FROM (per#>>'{delegacion,vigente_desde}')::timestamptz
       OR (fuente_personal#>>'{delegacion,vigente_hasta}')::timestamptz IS DISTINCT FROM (per#>>'{delegacion,vigente_hasta}')::timestamptz))
  OR (fuente_personal->'delegacion' IS DISTINCT FROM 'null'::jsonb)
     IS DISTINCT FROM (per->'delegacion' IS DISTINCT FROM 'null'::jsonb) THEN
  RAISE EXCEPTION 'aut32_cargo_no_admitido' USING ERRCODE = '42501';
 END IF;
 IF per->'delegacion' = 'null'::jsonb AND per->>'ocupante_persona_ref' IS DISTINCT FROM ident->>'persona_ref'
  OR per->'delegacion' <> 'null'::jsonb AND (
    per#>>'{delegacion,delegante_persona_ref}' IS DISTINCT FROM per->>'ocupante_persona_ref'
    OR per#>>'{delegacion,delegado_persona_ref}' IS DISTINCT FROM ident->>'persona_ref') THEN
  RAISE EXCEPTION 'aut32_cargo_no_admitido' USING ERRCODE = '42501';
 END IF;
 -- Protección del puntero y de las versiones centrales. La asignación propia
 -- pertenece a la persona del certificado y a un único perfil activo.
 SELECT ap.asignacion_ref,ap.version,ap.huella_sha256,ap.principal_id,ap.perfil_activo_ref,
   ap.version_rol_ref,ap.documento
 INTO STRICT a FROM vec_autorizacion.asignacion_perfil_actual p
 JOIN vec_autorizacion.asignacion_perfil ap ON ap.asignacion_ref=p.asignacion_ref
 WHERE p.perfil_activo_ref=comp->>'perfil_activo_ref' FOR SHARE OF p,ap;
 SELECT r.version_rol_ref,r.version,r.rol_id,r.huella_sha256,r.documento,
   v.revision,v.estado,v.huella_sha256 AS control_huella,v.documento AS control_documento
 INTO STRICT rol FROM vec_autorizacion.version_rol r
 JOIN vec_autorizacion.control_vigencia_version_rol_actual x ON x.version_rol_ref=r.version_rol_ref
 JOIN vec_autorizacion.control_vigencia_version_rol v ON v.version_rol_ref=x.version_rol_ref AND v.revision=x.revision
 WHERE r.version_rol_ref=a.version_rol_ref FOR SHARE OF r,x,v;
 -- Toda mutación del puntero de políticas exige actualizar esta fila en la
 -- misma transacción. FOR SHARE impide su COMMIT mientras se decide el efecto.
 SELECT revision,huella_sha256 INTO STRICT catalogo
  FROM vec_autorizacion.control_catalogo_politicas WHERE control_id FOR SHARE;
 instante := clock_timestamp();
 IF a.principal_id IS DISTINCT FROM ident->>'persona_ref'
  OR a.principal_id IS DISTINCT FROM comp->>'persona_ref'
  OR a.asignacion_ref IS DISTINCT FROM comp#>>'{asignacion,referencia}'
  OR a.version IS DISTINCT FROM (comp#>>'{asignacion,version}')::bigint
  OR a.huella_sha256 IS DISTINCT FROM comp#>>'{asignacion,huella_sha256}'
  OR a.perfil_activo_ref IS DISTINCT FROM comp->>'perfil_activo_ref'
  OR a.version_rol_ref IS DISTINCT FROM comp->>'asignacion_rol_ref'
  OR rol.version_rol_ref IS DISTINCT FROM comp#>>'{rol,referencia}'
  OR rol.version_rol_ref IS DISTINCT FROM comp->>'control_rol_ref'
  OR rol.version_rol_ref IS DISTINCT FROM comp#>>'{control_rol,referencia}'
  OR rol.version IS DISTINCT FROM (comp#>>'{rol,version}')::bigint
  OR rol.rol_id IS DISTINCT FROM comp->>'rol_id'
  OR rol.huella_sha256 IS DISTINCT FROM comp#>>'{rol,huella_sha256}'
  OR rol.documento->>'estado' IS DISTINCT FROM 'publicada'
  OR rol.documento->>'rol_id' IS DISTINCT FROM rol.rol_id
  OR rol.documento->>'version' IS DISTINCT FROM rol.version::text
  OR (rol.documento->>'publicada_en')::timestamptz > instante
  OR rol.estado IS DISTINCT FROM 'habilitada'
  OR (rol.control_documento->>'actualizado_en')::timestamptz > instante
  OR rol.revision IS DISTINCT FROM (comp#>>'{control_rol,version}')::numeric
  OR rol.control_huella IS DISTINCT FROM comp#>>'{control_rol,huella_sha256}'
  OR a.documento->>'estado' IS DISTINCT FROM 'activa'
  OR a.documento->>'vigente_desde' IS DISTINCT FROM comp->>'vigente_desde'
  OR a.documento->>'vigente_hasta' IS DISTINCT FROM comp->>'vigente_hasta'
  OR instante < (a.documento->>'vigente_desde')::timestamptz
  OR instante >= (a.documento->>'vigente_hasta')::timestamptz
  OR fecha_historica < (a.documento->>'vigente_desde')::timestamptz
  OR fecha_historica >= (a.documento->>'vigente_hasta')::timestamptz
  OR instante < (ca#>>'{vinculo_certificado,vigente_desde}')::timestamptz
  OR instante >= (ca#>>'{vinculo_certificado,vigente_hasta}')::timestamptz
  OR fecha_historica < (ca#>>'{vinculo_certificado,vigente_desde}')::timestamptz
  OR fecha_historica >= (ca#>>'{vinculo_certificado,vigente_hasta}')::timestamptz
  OR instante < (per->>'cargo_vigente_desde')::timestamptz
  OR instante >= (per->>'cargo_vigente_hasta')::timestamptz
  OR fecha_historica < (per->>'cargo_vigente_desde')::timestamptz
  OR fecha_historica >= (per->>'cargo_vigente_hasta')::timestamptz
  OR instante < (per->>'enlace_vigente_desde')::timestamptz
  OR instante >= (per->>'enlace_vigente_hasta')::timestamptz
  OR instante >= (v3->>'decision_valida_hasta')::timestamptz
  OR fecha_historica < (per->>'enlace_vigente_desde')::timestamptz
  OR fecha_historica >= (per->>'enlace_vigente_hasta')::timestamptz
  OR (per->'delegacion' IS DISTINCT FROM 'null'::jsonb AND
    (instante < (per#>>'{delegacion,vigente_desde}')::timestamptz
     OR instante >= (per#>>'{delegacion,vigente_hasta}')::timestamptz))
  OR comp->>'modulo_id' IS DISTINCT FROM rec->>'modulo_id'
  OR comp->>'tipo_recurso' IS DISTINCT FROM rec->>'tipo_recurso'
  OR comp->>'recurso_ref' IS DISTINCT FROM rec->>'recurso_autorizable_ref'
  OR comp->>'ambito_organizacion_ref' IS DISTINCT FROM rec->>'organizacion_ref'
  OR comp->>'ambito_unidad_ref' IS DISTINCT FROM rec->>'unidad_ref' THEN
  RAISE EXCEPTION 'aut32_concesion_no_admitida' USING ERRCODE = '42501';
 END IF;
 SELECT EXISTS(SELECT 1 FROM jsonb_array_elements(rol.documento->'concesiones') q
   WHERE q->>'accion'=c->>'accion' AND q->>'modulo_id'=comp->>'modulo_id'
    AND q->>'tipo_recurso'=comp->>'tipo_recurso'
    AND q->'finalidades' ? (c->>'finalidad')
    AND coalesce(q->'obligaciones','[]'::jsonb)='[]'::jsonb
    AND coalesce(q->'campos_permitidos','[]'::jsonb)='[]'::jsonb) INTO concedida;
 SELECT EXISTS(SELECT 1 FROM jsonb_array_elements(a.documento->'ambitos') q
   WHERE q->>'clave'='organizacion_ref' AND q->'valores' ? (rec->>'organizacion_ref')) INTO amb_org;
 SELECT EXISTS(SELECT 1 FROM jsonb_array_elements(a.documento->'ambitos') q
   WHERE q->>'clave'='unidad_ref' AND q->'valores' ? (rec->>'unidad_ref')) INTO amb_unidad;
 IF concedida IS NOT TRUE OR amb_org IS NOT TRUE OR amb_unidad IS NOT TRUE
  OR EXISTS(SELECT 1 FROM jsonb_array_elements(a.documento->'ambitos') q
   WHERE q->>'clave' NOT IN ('organizacion_ref','unidad_ref','expediente_ref','documento_ref')
    OR (q->>'clave'='expediente_ref' AND
       (q->'valores' ? (rec->>'expediente_ref')) IS NOT TRUE)
    OR (q->>'clave'='documento_ref' AND
       (q->'valores' ? (rec->>'documento_ref')) IS NOT TRUE)) THEN
  RAISE EXCEPTION 'aut32_concesion_no_admitida' USING ERRCODE = '42501';
 END IF;
 -- No existe un evaluador ABAC offline autorizado para el firmante. El
 -- catálogo protege el conjunto actual hasta COMMIT. Una política aplicable
 -- publicada, incluso de inicio futuro, deja esta vía cerrada.
 IF catalogo.huella_sha256 !~ '^[0-9a-f]{64}$'
  OR EXISTS(SELECT 1 FROM vec_autorizacion.politica_restrictiva_actual x
   JOIN vec_autorizacion.politica_restrictiva p USING(politica_id,politica_ref)
   WHERE jsonb_typeof(p.documento->'acciones') IS DISTINCT FROM 'array'
    OR jsonb_typeof(p.documento->'modulos') IS DISTINCT FROM 'array'
    OR jsonb_typeof(p.documento->'tipos_recurso') IS DISTINCT FROM 'array'
    OR (p.documento->>'estado' IN ('publicada','retirada')) IS NOT TRUE
    OR (p.documento->>'efecto' IN ('restringir','denegar')) IS NOT TRUE
    OR vec_autorizacion.instante_utc_microsegundo_valido(p.documento->>'vigente_desde') IS NOT TRUE
    OR vec_autorizacion.instante_utc_microsegundo_valido(p.documento->>'vigente_hasta') IS NOT TRUE
    OR (p.documento->>'vigente_hasta')::timestamptz <=
       (p.documento->>'vigente_desde')::timestamptz)
  OR EXISTS(SELECT 1 FROM vec_autorizacion.politica_restrictiva_actual x
   JOIN vec_autorizacion.politica_restrictiva p USING(politica_id,politica_ref)
   WHERE p.documento->>'estado'='publicada'
    AND instante < (p.documento->>'vigente_hasta')::timestamptz
    AND ((p.documento->'acciones' ? (c->>'accion')) OR (p.documento->'acciones' ? '*'))
    AND ((p.documento->'modulos' ? (comp->>'modulo_id')) OR (p.documento->'modulos' ? '*'))
    AND ((p.documento->'tipos_recurso' ? (comp->>'tipo_recurso')) OR (p.documento->'tipos_recurso' ? '*'))) THEN
  RAISE EXCEPTION 'aut32_politica_no_evaluable' USING ERRCODE='42501';
 END IF;
 RETURN c;
END $function$
;
REVOKE ALL ON FUNCTION vec_autorizacion.validar_competencia_nominal_firmante_ct_v1(bytea,jsonb,jsonb) FROM PUBLIC;
DO $post$
DECLARE f pg_proc%ROWTYPE; actual text;
BEGIN
 SELECT * INTO f FROM pg_proc WHERE oid=to_regprocedure('vec_autorizacion.validar_competencia_nominal_firmante_ct_v1(bytea,jsonb,jsonb)');
 actual:=encode(sha256(convert_to(pg_get_functiondef(f.oid),'UTF8')),'hex');
 IF actual IS DISTINCT FROM '5d4b0799ab66aabdbd667c914a2463a574477861a565c28b7beb594c40316903'
  OR f.proowner IS DISTINCT FROM 'vec_autorizacion_propietario'::regrole
  OR f.proacl::text IS DISTINCT FROM '{vec_autorizacion_propietario=X/vec_autorizacion_propietario}'
  OR f.prosecdef IS DISTINCT FROM true
  OR f.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','lock_timeout=2s','TimeZone=UTC'] THEN
  RAISE EXCEPTION 'AUT97: PARO postimagen/propietario/ACL/configuración incompatibles; huella=%',actual USING ERRCODE='55000';
 END IF;
END $post$;
COMMIT;
