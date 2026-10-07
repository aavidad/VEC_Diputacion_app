\set ON_ERROR_STOP on
-- AUT32: competencia nominal del firmante dentro de la transacción CT.
-- Requiere las fachadas propietarias CA25 y Personal28. No abre AD164.
BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion:migracion:000032', 0));

DO $preimagen$
DECLARE n text;
BEGIN
 IF current_user <> 'vec_autorizacion_propietario' THEN
  RAISE EXCEPTION 'aut32_preimagen_incompatible' USING ERRCODE = '55000';
 END IF;
 FOREACH n IN ARRAY ARRAY['version_rol','asignacion_perfil','asignacion_perfil_actual',
   'control_vigencia_version_rol','control_vigencia_version_rol_actual',
   'politica_restrictiva','politica_restrictiva_actual','control_catalogo_politicas'] LOOP
  IF to_regclass('vec_autorizacion.' || n) IS NULL THEN
   RAISE EXCEPTION 'aut32_preimagen_incompatible' USING ERRCODE = '55000';
  END IF;
 END LOOP;
 IF EXISTS(SELECT 1 FROM pg_class c WHERE c.oid IN (
   'vec_autorizacion.politica_restrictiva'::regclass,
   'vec_autorizacion.politica_restrictiva_actual'::regclass,
   'vec_autorizacion.control_catalogo_politicas'::regclass)
   AND (c.relowner<>'vec_autorizacion_propietario'::regrole
    OR NOT c.relrowsecurity OR NOT c.relforcerowsecurity)) THEN
  RAISE EXCEPTION 'aut32_preimagen_incompatible' USING ERRCODE='55000'; END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_trigger t WHERE t.tgrelid=
   'vec_autorizacion.politica_restrictiva_actual'::regclass
   AND t.tgname='politica_actual_exige_catalogo'
   AND t.tgenabled IN ('O','A') AND t.tgdeferrable
   AND t.tgfoid='vec_autorizacion.exigir_catalogo_actualizado_en_transaccion()'::regprocedure) THEN
  RAISE EXCEPTION 'aut32_preimagen_incompatible' USING ERRCODE='55000'; END IF;
 IF to_regprocedure('vec_contexto_actor_v1.leer_revalidar_certificado_firmante_ct_v2(text)') IS NULL
  OR to_regprocedure('vec_contexto_actor_v1.recuperar_historia_certificado_firmante_ct_v2(text,numeric,text)') IS NULL
  OR to_regprocedure('vec_personal.leer_revalidar_cargo_ocupante_ct_v1(bytea)') IS NULL
  OR to_regprocedure('vec_autorizacion_atestada_v3.comprobar_consumo_firma_ct_v1(jsonb)') IS NULL
  OR to_regprocedure('vec_autorizacion.acreditar_competencia_nominal_firmante_ct_v1(bytea,jsonb,jsonb)') IS NOT NULL
  OR to_regprocedure('vec_autorizacion.recuperar_evidencia_competencia_firmante_ct_v1(text,text,bytea,jsonb,jsonb)') IS NOT NULL
  OR to_regclass('vec_autorizacion.evidencia_competencia_firmante_ct_v1') IS NOT NULL
  OR NOT has_function_privilege('vec_autorizacion_propietario',
    'vec_contexto_actor_v1.leer_revalidar_certificado_firmante_ct_v2(text)', 'EXECUTE')
  OR NOT has_function_privilege('vec_autorizacion_propietario',
    'vec_contexto_actor_v1.recuperar_historia_certificado_firmante_ct_v2(text,numeric,text)', 'EXECUTE')
  OR NOT has_function_privilege('vec_autorizacion_propietario',
    'vec_personal.leer_revalidar_cargo_ocupante_ct_v1(bytea)', 'EXECUTE')
  OR NOT has_function_privilege('vec_autorizacion_propietario',
    'vec_autorizacion_atestada_v3.comprobar_consumo_firma_ct_v1(jsonb)', 'EXECUTE') THEN
  RAISE EXCEPTION 'aut32_preimagen_incompatible' USING ERRCODE = '55000';
 END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=
    'vec_contexto_actor_v1.leer_revalidar_certificado_firmante_ct_v2(text)'::regprocedure
    AND strpos(p.prosrc,'organizacion_destino')>0)
  OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=
    'vec_contexto_actor_v1.recuperar_historia_certificado_firmante_ct_v2(text,numeric,text)'::regprocedure
    AND strpos(p.prosrc,'organizacion_destino')>0) THEN
  RAISE EXCEPTION 'aut32_preimagen_incompatible' USING ERRCODE='55000'; END IF;
END $preimagen$;

-- Una fila original por efecto. El recibo CT apunta a evidencia_ref y huella;
-- esta tabla conserva los bytes previos, sin reserializar jsonb.
CREATE TABLE vec_autorizacion.evidencia_competencia_firmante_ct_v1 (
 evidencia_ref text PRIMARY KEY,
 efecto_ref text NOT NULL UNIQUE,
 esquema text NOT NULL CHECK (esquema = 'vec.competencia-firmante.historica.v1'),
 canonico bytea NOT NULL CHECK (octet_length(canonico) BETWEEN 512 AND 32768),
 huella_sha256 text NOT NULL CHECK (huella_sha256 ~ '^[0-9a-f]{64}$'),
 organizacion_ref text NOT NULL,
 unidad_ref text NOT NULL,
 expediente_ref text NOT NULL,
 documento_ref text NOT NULL,
 persona_ref text NOT NULL,
 certificado_der_sha256 text NOT NULL CHECK (certificado_der_sha256 ~ '^[0-9a-f]{64}$'),
 organizacion_destino jsonb NOT NULL,
 organizacion_destino_huella_sha256 text NOT NULL CHECK (organizacion_destino_huella_sha256 ~ '^[0-9a-f]{64}$'),
 catalogo_politicas_revision numeric(20,0) NOT NULL CHECK (catalogo_politicas_revision>0),
 catalogo_politicas_huella_sha256 text NOT NULL CHECK (catalogo_politicas_huella_sha256 ~ '^[0-9a-f]{64}$'),
 consumo_decision_ref text NOT NULL,
 consumo_huella_sha256 text NOT NULL CHECK (consumo_huella_sha256 ~ '^[0-9a-f]{64}$'),
 auditoria_consumo_ref text NOT NULL,
 auditoria_consumo_huella_sha256 text NOT NULL CHECK (auditoria_consumo_huella_sha256 ~ '^[0-9a-f]{64}$'),
 registrador_principal_ref text NOT NULL,
 registrador_perfil_ref text NOT NULL,
 creada_en timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
 CHECK (huella_sha256 = encode(sha256(canonico),'hex')),
 CHECK (jsonb_typeof(organizacion_destino)='object'),
 CHECK (organizacion_destino ?& ARRAY['organizacion_ref','organizacion_version',
  'organizacion_huella_sha256','organizacion_procedencia_ref',
  'organizacion_procedencia_version','organizacion_procedencia_sha256',
  'vinculo_corporativo_ref','vinculo_corporativo_version',
  'vinculo_corporativo_huella_sha256']),
 CHECK (organizacion_destino->>'organizacion_ref' = organizacion_ref),
 CHECK (organizacion_destino_huella_sha256 =
   encode(sha256(convert_to(organizacion_destino::text,'UTF8')),'hex')),
 CHECK (evidencia_ref = 'evidencia:competencia-firmante-ct:' ||
   encode(sha256(convert_to(efecto_ref,'UTF8') || canonico),'hex')),
 CHECK (vec_autorizacion.texto_positivo_valido(efecto_ref,512)),
 CHECK (vec_autorizacion.texto_positivo_valido(organizacion_ref,512)),
 CHECK (vec_autorizacion.texto_positivo_valido(unidad_ref,512)),
 CHECK (vec_autorizacion.texto_positivo_valido(expediente_ref,512)),
 CHECK (vec_autorizacion.texto_positivo_valido(documento_ref,512)),
 CHECK (vec_autorizacion.texto_positivo_valido(persona_ref,512)),
 CHECK (vec_autorizacion.texto_positivo_valido(consumo_decision_ref,512)),
 CHECK (vec_autorizacion.texto_positivo_valido(auditoria_consumo_ref,512)),
 CHECK (vec_autorizacion.texto_positivo_valido(registrador_principal_ref,512)),
 CHECK (vec_autorizacion.texto_positivo_valido(registrador_perfil_ref,512))
);
REVOKE ALL ON TABLE vec_autorizacion.evidencia_competencia_firmante_ct_v1 FROM PUBLIC;
REVOKE ALL ON TYPE vec_autorizacion.evidencia_competencia_firmante_ct_v1 FROM PUBLIC;
DO $tabla_acl$
DECLARE a record;
BEGIN
 FOR a IN SELECT DISTINCT x.grantee FROM pg_class c CROSS JOIN LATERAL
  aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) x
  WHERE c.oid='vec_autorizacion.evidencia_competencia_firmante_ct_v1'::regclass
   AND x.grantee<>'vec_autorizacion_propietario'::regrole LOOP
  EXECUTE format('REVOKE ALL ON TABLE vec_autorizacion.evidencia_competencia_firmante_ct_v1 FROM %s',
   CASE WHEN a.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(a.grantee)) END);
 END LOOP;
END $tabla_acl$;
ALTER TABLE vec_autorizacion.evidencia_competencia_firmante_ct_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_autorizacion.evidencia_competencia_firmante_ct_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY evidencia_ct_propietario ON vec_autorizacion.evidencia_competencia_firmante_ct_v1
 FOR ALL TO vec_autorizacion_propietario
 USING (current_user = 'vec_autorizacion_propietario')
 WITH CHECK (current_user = 'vec_autorizacion_propietario');

CREATE FUNCTION vec_autorizacion.bloquear_evidencia_competencia_ct_v1()
RETURNS trigger LANGUAGE plpgsql VOLATILE SECURITY INVOKER
SET search_path = pg_catalog AS $f$
BEGIN
 RAISE EXCEPTION 'aut32_evidencia_inmutable' USING ERRCODE = '55000';
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.bloquear_evidencia_competencia_ct_v1() FROM PUBLIC;
CREATE TRIGGER evidencia_competencia_ct_inmutable
 BEFORE UPDATE OR DELETE ON vec_autorizacion.evidencia_competencia_firmante_ct_v1
 FOR EACH ROW EXECUTE FUNCTION vec_autorizacion.bloquear_evidencia_competencia_ct_v1();

-- Este predicado vuelve a leer fuentes actuales y conserva sus bloqueos hasta
-- COMMIT. La autorización del registrador pertenece al consumo V3 previo en CT.
CREATE FUNCTION vec_autorizacion.validar_competencia_nominal_firmante_ct_v1(
 p_contexto_nominal bytea, p_relacion_ct jsonb, p_consumo_v3 jsonb
) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path = pg_catalog SET row_security = 'on' SET lock_timeout = '2s' SET TimeZone = 'UTC'
AS $f$
DECLARE c jsonb; ident jsonb; comp jsonb; per jsonb; rec jsonb; ct jsonb;
 ca jsonb; org_ca jsonb; fuente_personal jsonb; v3 jsonb; a record; rol record;
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
  OR org_ca->>'organizacion_ref' IS DISTINCT FROM rec->>'organizacion_ref'
  OR org_ca->>'organizacion_ref' IS DISTINCT FROM p_relacion_ct->>'organizacion_ref'
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
 fuente_personal := vec_personal.leer_revalidar_cargo_ocupante_ct_v1(p_contexto_nominal);
 IF fuente_personal->>'esquema' IS DISTINCT FROM 'vec.personal.cargo-ocupante.ct.v1'
  OR fuente_personal->'cargo'->>'referencia' IS DISTINCT FROM per#>>'{cargo,referencia}'
  OR (fuente_personal#>>'{cargo,version}')::numeric IS DISTINCT FROM (per#>>'{cargo,version}')::numeric
  OR fuente_personal#>>'{cargo,huella_sha256}' IS DISTINCT FROM per#>>'{cargo,huella_sha256}'
  OR fuente_personal->'enlace_ocupante'->>'referencia' IS DISTINCT FROM per#>>'{enlace_ocupante,referencia}'
  OR (fuente_personal#>>'{enlace_ocupante,version}')::numeric IS DISTINCT FROM (per#>>'{enlace_ocupante,version}')::numeric
  OR fuente_personal#>>'{enlace_ocupante,huella_sha256}' IS DISTINCT FROM per#>>'{enlace_ocupante,huella_sha256}'
  OR fuente_personal->>'ocupante_persona_ref' IS DISTINCT FROM per->>'ocupante_persona_ref'
  OR fuente_personal->>'cargo_ref_enlace' IS DISTINCT FROM per->>'cargo_ref_enlace'
  OR fuente_personal->>'organizacion_ref' IS DISTINCT FROM rec->>'organizacion_ref'
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
 SELECT a.asignacion_ref,a.version,a.huella_sha256,a.principal_id,a.perfil_activo_ref,
   a.version_rol_ref,a.documento
 INTO STRICT a FROM vec_autorizacion.asignacion_perfil_actual p
 JOIN vec_autorizacion.asignacion_perfil a ON a.asignacion_ref=p.asignacion_ref
 WHERE p.perfil_activo_ref=comp->>'perfil_activo_ref' FOR SHARE OF p,a;
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
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.validar_competencia_nominal_firmante_ct_v1(bytea,jsonb,jsonb) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.acreditar_competencia_nominal_firmante_ct_v1(
 p_contexto_nominal bytea,p_relacion_ct jsonb,p_consumo_v3 jsonb
) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path = pg_catalog SET row_security = 'on' SET lock_timeout = '2s' SET TimeZone = 'UTC'
AS $f$
DECLARE c jsonb; h text; ref text; previo vec_autorizacion.evidencia_competencia_firmante_ct_v1%ROWTYPE;
 v3 jsonb; org_ca jsonb; catalogo record;
BEGIN
 c := vec_autorizacion.validar_competencia_nominal_firmante_ct_v1(
   p_contexto_nominal,p_relacion_ct,p_consumo_v3);
 org_ca := (vec_contexto_actor_v1.leer_revalidar_certificado_firmante_ct_v2(
   c#>>'{identidad,certificado_der_sha256}'))->'organizacion_destino';
 IF jsonb_typeof(org_ca) IS DISTINCT FROM 'object'
  OR org_ca->>'organizacion_ref' IS DISTINCT FROM c#>>'{recurso,organizacion_ref}' THEN
  RAISE EXCEPTION 'aut32_organizacion_no_admitida' USING ERRCODE='42501'; END IF;
 SELECT revision,huella_sha256 INTO STRICT catalogo
  FROM vec_autorizacion.control_catalogo_politicas WHERE control_id FOR SHARE;
 h := encode(sha256(p_contexto_nominal),'hex');
 v3 := vec_autorizacion_atestada_v3.comprobar_consumo_firma_ct_v1(p_consumo_v3);
 ref := 'evidencia:competencia-firmante-ct:' ||
  encode(sha256(convert_to(p_consumo_v3->>'efecto_ref','UTF8') || p_contexto_nominal),'hex');
 PERFORM pg_advisory_xact_lock(hashtextextended(
   'vec_autorizacion:competencia_ct:' || (p_consumo_v3->>'efecto_ref'),0));
 SELECT * INTO previo FROM vec_autorizacion.evidencia_competencia_firmante_ct_v1
  WHERE efecto_ref=p_consumo_v3->>'efecto_ref' FOR SHARE;
 IF FOUND THEN
  -- La vigencia actual ya se ha comprobado. El enlace histórico permanece.
  IF previo.organizacion_ref IS DISTINCT FROM c#>>'{recurso,organizacion_ref}'
   OR previo.unidad_ref IS DISTINCT FROM c#>>'{recurso,unidad_ref}'
   OR previo.expediente_ref IS DISTINCT FROM c#>>'{recurso,expediente_ref}'
   OR previo.documento_ref IS DISTINCT FROM c#>>'{recurso,documento_ref}'
   OR previo.persona_ref IS DISTINCT FROM c#>>'{identidad,persona_ref}'
   OR previo.certificado_der_sha256 IS DISTINCT FROM c#>>'{identidad,certificado_der_sha256}'
   OR (convert_from(previo.canonico,'UTF8')::jsonb)->'recurso' IS DISTINCT FROM c->'recurso'
   OR (convert_from(previo.canonico,'UTF8')::jsonb)->>'accion' IS DISTINCT FROM c->>'accion'
   OR (convert_from(previo.canonico,'UTF8')::jsonb)->>'finalidad' IS DISTINCT FROM c->>'finalidad'
   OR (convert_from(previo.canonico,'UTF8')::jsonb)->'motivo' IS DISTINCT FROM c->'motivo'
   OR (convert_from(previo.canonico,'UTF8')::jsonb)->'circuito' IS DISTINCT FROM c->'circuito'
   OR (convert_from(previo.canonico,'UTF8')::jsonb)->>'paso_ref' IS DISTINCT FROM c->>'paso_ref'
   OR (convert_from(previo.canonico,'UTF8')::jsonb)->>'paso_orden' IS DISTINCT FROM c->>'paso_orden' THEN
   RAISE EXCEPTION 'aut32_efecto_divergente' USING ERRCODE = '42501';
  END IF;
  RETURN jsonb_build_object('evidencia_ref',previo.evidencia_ref,'esquema',previo.esquema,
   'version',1,'huella_sha256',previo.huella_sha256,
   'organizacion_destino_huella_sha256',previo.organizacion_destino_huella_sha256,
   'catalogo_politicas_revision',previo.catalogo_politicas_revision,
   'catalogo_politicas_huella_sha256',previo.catalogo_politicas_huella_sha256,
   'recuperada',true);
 END IF;
 INSERT INTO vec_autorizacion.evidencia_competencia_firmante_ct_v1 (
  evidencia_ref,efecto_ref,esquema,canonico,huella_sha256,organizacion_ref,unidad_ref,
  expediente_ref,documento_ref,persona_ref,certificado_der_sha256,
  organizacion_destino,organizacion_destino_huella_sha256,
  catalogo_politicas_revision,catalogo_politicas_huella_sha256,
  consumo_decision_ref,consumo_huella_sha256,auditoria_consumo_ref,
  auditoria_consumo_huella_sha256,registrador_principal_ref,registrador_perfil_ref)
 VALUES (ref,p_consumo_v3->>'efecto_ref','vec.competencia-firmante.historica.v1',
  p_contexto_nominal,h,c#>>'{recurso,organizacion_ref}',c#>>'{recurso,unidad_ref}',
  c#>>'{recurso,expediente_ref}',c#>>'{recurso,documento_ref}',c#>>'{identidad,persona_ref}',
  c#>>'{identidad,certificado_der_sha256}',org_ca,
  encode(sha256(convert_to(org_ca::text,'UTF8')),'hex'),
  catalogo.revision,catalogo.huella_sha256,p_consumo_v3->>'decision_ref',
  p_consumo_v3->>'consumo_huella_sha256',p_consumo_v3->>'auditoria_ref',
  v3->>'auditoria_huella_sha256',v3->>'registrador_principal_ref',
  v3->>'registrador_perfil_ref');
 RETURN jsonb_build_object('evidencia_ref',ref,'esquema','vec.competencia-firmante.historica.v1',
  'version',1,'huella_sha256',h,
  'organizacion_destino_huella_sha256',encode(sha256(convert_to(org_ca::text,'UTF8')),'hex'),
  'catalogo_politicas_revision',catalogo.revision,
  'catalogo_politicas_huella_sha256',catalogo.huella_sha256,
  'recuperada',false);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.acreditar_competencia_nominal_firmante_ct_v1(bytea,jsonb,jsonb) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.recuperar_evidencia_competencia_firmante_ct_v1(
 p_evidencia_ref text,p_huella_sha256 text,p_contexto_actual bytea,
 p_relacion_ct jsonb,p_consumo_v3 jsonb
) RETURNS bytea
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path = pg_catalog SET row_security = 'on' SET lock_timeout = '2s' SET TimeZone = 'UTC'
AS $f$
DECLARE c jsonb; historico jsonb; fuente_ca jsonb;
 original vec_autorizacion.evidencia_competencia_firmante_ct_v1%ROWTYPE;
BEGIN
 c := vec_autorizacion.validar_competencia_nominal_firmante_ct_v1(
   p_contexto_actual,p_relacion_ct,p_consumo_v3);
 SELECT * INTO STRICT original FROM vec_autorizacion.evidencia_competencia_firmante_ct_v1
  WHERE evidencia_ref=p_evidencia_ref AND huella_sha256=p_huella_sha256 FOR SHARE;
 IF original.efecto_ref IS DISTINCT FROM p_consumo_v3->>'efecto_ref'
  OR original.organizacion_ref IS DISTINCT FROM c#>>'{recurso,organizacion_ref}'
  OR original.unidad_ref IS DISTINCT FROM c#>>'{recurso,unidad_ref}'
  OR original.expediente_ref IS DISTINCT FROM c#>>'{recurso,expediente_ref}'
  OR original.documento_ref IS DISTINCT FROM c#>>'{recurso,documento_ref}'
  OR original.persona_ref IS DISTINCT FROM c#>>'{identidad,persona_ref}'
  OR original.certificado_der_sha256 IS DISTINCT FROM c#>>'{identidad,certificado_der_sha256}'
  OR (convert_from(original.canonico,'UTF8')::jsonb)->'recurso' IS DISTINCT FROM c->'recurso'
  OR (convert_from(original.canonico,'UTF8')::jsonb)->>'accion' IS DISTINCT FROM c->>'accion'
  OR (convert_from(original.canonico,'UTF8')::jsonb)->>'finalidad' IS DISTINCT FROM c->>'finalidad'
  OR (convert_from(original.canonico,'UTF8')::jsonb)->'motivo' IS DISTINCT FROM c->'motivo'
  OR (convert_from(original.canonico,'UTF8')::jsonb)->'circuito' IS DISTINCT FROM c->'circuito'
  OR (convert_from(original.canonico,'UTF8')::jsonb)->>'paso_ref' IS DISTINCT FROM c->>'paso_ref'
  OR (convert_from(original.canonico,'UTF8')::jsonb)->>'paso_orden' IS DISTINCT FROM c->>'paso_orden'
  OR original.huella_sha256 IS DISTINCT FROM encode(sha256(original.canonico),'hex') THEN
  RAISE EXCEPTION 'aut32_evidencia_no_disponible' USING ERRCODE = '42501';
 END IF;
 historico := convert_from(original.canonico,'UTF8')::jsonb;
 fuente_ca := vec_contexto_actor_v1.recuperar_historia_certificado_firmante_ct_v2(
  historico#>>'{identidad,vinculo_certificado,referencia}',
  (historico#>>'{identidad,vinculo_certificado,version}')::numeric,
  historico#>>'{identidad,vinculo_certificado,huella_sha256}');
 IF fuente_ca->>'esquema' IS DISTINCT FROM 'vec.contexto-actor.certificado-firmante-historia.v2'
  OR fuente_ca->>'cuenta_ref' IS DISTINCT FROM historico#>>'{identidad,cuenta,referencia}'
  OR fuente_ca->>'cuenta_version' IS DISTINCT FROM historico#>>'{identidad,cuenta,version}'
  OR fuente_ca->>'cuenta_huella_sha256' IS DISTINCT FROM historico#>>'{identidad,cuenta,huella_sha256}'
  OR fuente_ca->>'persona_ref' IS DISTINCT FROM historico#>>'{identidad,persona,referencia}'
  OR fuente_ca->>'persona_version' IS DISTINCT FROM historico#>>'{identidad,persona,version}'
  OR fuente_ca->>'persona_huella_sha256' IS DISTINCT FROM historico#>>'{identidad,persona,huella_sha256}'
  OR fuente_ca->>'vinculo_cuenta_persona_ref' IS DISTINCT FROM historico#>>'{identidad,vinculo_cuenta_persona,referencia}'
  OR fuente_ca->>'vinculo_cuenta_persona_version' IS DISTINCT FROM historico#>>'{identidad,vinculo_cuenta_persona,version}'
  OR fuente_ca->>'vinculo_cuenta_persona_huella_sha256' IS DISTINCT FROM historico#>>'{identidad,vinculo_cuenta_persona,huella_sha256}'
  OR fuente_ca->'organizacion_destino' IS DISTINCT FROM original.organizacion_destino
  OR original.organizacion_destino_huella_sha256 IS DISTINCT FROM
   encode(sha256(convert_to(original.organizacion_destino::text,'UTF8')),'hex') THEN
  RAISE EXCEPTION 'aut32_evidencia_no_disponible' USING ERRCODE = '42501';
 END IF;
 RETURN original.canonico;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.recuperar_evidencia_competencia_firmante_ct_v1(text,text,bytea,jsonb,jsonb) FROM PUBLIC;

GRANT USAGE ON SCHEMA vec_autorizacion TO vec_contratacion_temporal_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion.acreditar_competencia_nominal_firmante_ct_v1(bytea,jsonb,jsonb)
 TO vec_contratacion_temporal_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion.recuperar_evidencia_competencia_firmante_ct_v1(text,text,bytea,jsonb,jsonb)
 TO vec_contratacion_temporal_propietario;

DO $acl$
DECLARE f regprocedure; a record; permitido oid;
BEGIN
 FOREACH f IN ARRAY ARRAY[
  'vec_autorizacion.validar_competencia_nominal_firmante_ct_v1(bytea,jsonb,jsonb)'::regprocedure,
  'vec_autorizacion.acreditar_competencia_nominal_firmante_ct_v1(bytea,jsonb,jsonb)'::regprocedure,
  'vec_autorizacion.recuperar_evidencia_competencia_firmante_ct_v1(text,text,bytea,jsonb,jsonb)'::regprocedure
 ] LOOP
  permitido := CASE WHEN f = 'vec_autorizacion.validar_competencia_nominal_firmante_ct_v1(bytea,jsonb,jsonb)'::regprocedure
   THEN 'vec_autorizacion_propietario'::regrole::oid
   ELSE 'vec_contratacion_temporal_propietario'::regrole::oid END;
  FOR a IN SELECT DISTINCT x.grantee FROM pg_proc p CROSS JOIN LATERAL
   aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x WHERE p.oid=f
   AND x.grantee NOT IN ('vec_autorizacion_propietario'::regrole,permitido)
  LOOP
   EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %s',f::text,
   CASE WHEN a.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(a.grantee)) END);
  END LOOP;
  IF EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL
   aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x WHERE p.oid=f
   AND x.grantee NOT IN ('vec_autorizacion_propietario'::regrole,permitido))
   OR NOT has_function_privilege('vec_autorizacion_propietario',f,'EXECUTE')
   OR (permitido <> 'vec_autorizacion_propietario'::regrole::oid
    AND NOT has_function_privilege('vec_contratacion_temporal_propietario',f,'EXECUTE')) THEN
   RAISE EXCEPTION 'aut32_acl_incompatible' USING ERRCODE='55000'; END IF;
 END LOOP;
END $acl$;
COMMIT;
