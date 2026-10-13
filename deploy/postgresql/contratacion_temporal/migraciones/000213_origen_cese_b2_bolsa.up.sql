\set ON_ERROR_STOP on
-- CT213: conserva el origen CT155 y admite únicamente el selector nativo
-- confirmado por CT212. La historia y los consumidores CT115 permanecen.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000213',0));
DO $pre$
DECLARE f regprocedure:=to_regprocedure('vec_contratacion_temporal.origen_incorporacion_neutral_ct155(text,text,text)');
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario'
    OR f IS NULL
    OR encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex')
       IS DISTINCT FROM 'c477cb318013c3c753038eec402ac42c069ec7bde0060455276825634cf50a0f'
    OR (SELECT proowner<>'vec_contratacion_temporal_propietario'::regrole
           OR prosecdef OR proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog','row_security=on']::text[]
           OR proacl IS DISTINCT FROM ARRAY['vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario']::aclitem[]
        FROM pg_proc WHERE oid=f) IS NOT FALSE
    OR to_regclass('vec_contratacion_temporal.origen_incorporacion_personal_b2_v1') IS NULL
    OR to_regclass('vec_contratacion_temporal.plan_incorporacion_personal_b2_v1') IS NULL
    OR to_regprocedure('vec_contratacion_temporal.probar_cese_personal_b2_para_bolsa_v1(text)') IS NOT NULL
 THEN RAISE EXCEPTION 'CT213: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE OR REPLACE FUNCTION vec_contratacion_temporal.origen_incorporacion_neutral_ct155(org text,exp text,recibo text)
RETURNS TABLE(protocolo text,organizacion_ref text,expediente_ref text,recibo_ref text,inicio date,hasta date,llamamiento_ref text,relacion_ref text,seguimiento_ref text)
LANGUAGE plpgsql STABLE SET search_path=pg_catalog,pg_temp SET row_security=on AS $f$
DECLARE total integer;
BEGIN
 SELECT count(*) INTO total FROM (
 SELECT i.recibo_ref FROM vec_contratacion_temporal.incorporacion_registro_v2 i WHERE i.organizacion_ref=org AND i.expediente_ref=exp AND (recibo IS NULL OR i.recibo_ref=recibo)
 UNION ALL SELECT i.recibo_ref FROM vec_contratacion_temporal.origen_incorporacion_personal_b2_v1 i WHERE i.organizacion_ref=org AND i.expediente_ref=exp AND (recibo IS NULL OR i.recibo_ref=recibo)) x;
 IF total>1 THEN RAISE EXCEPTION 'CT155: origen ambiguo' USING ERRCODE='55000'; END IF;
 IF EXISTS (
   SELECT 1 FROM vec_contratacion_temporal.origen_incorporacion_personal_b2_v1 i
   JOIN vec_contratacion_temporal.plan_incorporacion_personal_b2_v1 p
     ON p.organizacion_ref=i.organizacion_ref AND p.expediente_ref=i.expediente_ref AND p.plan_ref=i.plan_ref
   WHERE i.organizacion_ref=org AND i.expediente_ref=exp AND (recibo IS NULL OR i.recibo_ref=recibo)
     AND ((p.material_json#>'{material,bolsa_nativa}' IS NOT NULL)::integer+
          (p.material_json#>'{material,bolsa}' IS NOT NULL)::integer)<>1
 ) THEN RAISE EXCEPTION 'CT213: selector de Bolsa ambiguo o ausente' USING ERRCODE='55000'; END IF;
 RETURN QUERY
 SELECT 'ejercicio_v2'::text,i.organizacion_ref,i.expediente_ref,i.recibo_ref,vec_contratacion_temporal.inicio_incorporacion_ct115(i.material_json),
 ((i.material_json#>>'{Confirmacion,PeriodoIncorporacion,hasta}')::timestamptz AT TIME ZONE 'Europe/Madrid')::date,
 (SELECT p.llamamiento_ref FROM vec_contratacion_temporal.propuesta_formalizacion p WHERE p.organizacion_ref=org AND p.expediente_ref=exp ORDER BY p.confirmada_en DESC,p.propuesta_ref COLLATE "C" DESC LIMIT 1),
 r.relacion_ref,i.seguimiento_ref
 FROM vec_contratacion_temporal.incorporacion_registro_v2 i
 JOIN vec_contratacion_temporal.seguimiento_raiz_v2 r ON r.seguimiento_ref=i.seguimiento_ref AND r.organizacion_ref=i.organizacion_ref AND r.expediente_ref=i.expediente_ref AND r.relacion_ref=i.material_json#>>'{Confirmacion,ResultadoPersonal,relacion_ref}'
 WHERE i.organizacion_ref=org AND i.expediente_ref=exp AND (recibo IS NULL OR i.recibo_ref=recibo)
 UNION ALL
 SELECT i.protocolo,i.organizacion_ref,i.expediente_ref,i.recibo_ref,(p.material_json#>>'{material,desde}')::date,nullif(p.material_json#>>'{material,hasta}','')::date,
 CASE WHEN p.material_json#>'{material,bolsa_nativa}' IS NOT NULL THEN
   p.material_json#>>'{material,bolsa_nativa,llamamiento_ref}'
 ELSE p.material_json#>>'{material,bolsa,llamamiento_ref}' END,i.relacion_ref,NULL::text
 FROM vec_contratacion_temporal.origen_incorporacion_personal_b2_v1 i
 JOIN vec_contratacion_temporal.plan_incorporacion_personal_b2_v1 p ON p.organizacion_ref=i.organizacion_ref AND p.expediente_ref=i.expediente_ref AND p.plan_ref=i.plan_ref
 WHERE i.organizacion_ref=org AND i.expediente_ref=exp AND (recibo IS NULL OR i.recibo_ref=recibo)
   AND ((p.material_json#>'{material,bolsa_nativa}' IS NOT NULL)::integer+
        (p.material_json#>'{material,bolsa}' IS NOT NULL)::integer)=1;
END $f$;

-- Lectura propietaria de un único cese CT ya publicado. No permite al LOGIN
-- Bolsa leer tablas CT ni acepta el cese de su propia transacción.
CREATE FUNCTION vec_contratacion_temporal.probar_cese_personal_b2_para_bolsa_v1(p_recibo_ct text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE c vec_contratacion_temporal.cese_nombramiento_v1%ROWTYPE;
 i vec_contratacion_temporal.origen_incorporacion_personal_b2_v1%ROWTYPE;
 p vec_contratacion_temporal.plan_incorporacion_personal_b2_v1%ROWTYPE;
 o vec_contratacion_temporal.outbox_expediente_integral%ROWTYPE;
 n record; cuerpo jsonb; evento jsonb; publicacion_ref text; h jsonb; posicion bigint;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_migrador','MEMBER')
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR current_setting('TimeZone')<>'UTC'
 THEN RAISE EXCEPTION 'CT213: sesión de prueba denegada' USING ERRCODE='42501'; END IF;
 IF p_recibo_ct IS NULL OR octet_length(p_recibo_ct) NOT BETWEEN 8 AND 512
 THEN RAISE EXCEPTION 'CT213: recibo inválido' USING ERRCODE='22023'; END IF;
 SELECT * INTO c FROM vec_contratacion_temporal.cese_nombramiento_v1 x
  WHERE x.recibo_ref=p_recibo_ct AND x.estado='confirmada'
    AND x.incorporacion_protocolo='personal_b2_v1'
    AND x.incorporacion_ref IS NULL AND x.incorporacion_b2_ref IS NOT NULL;
 IF NOT FOUND OR c.transaccion_publicacion IS NULL
    OR c.transaccion_publicacion>=pg_snapshot_xmin(pg_current_snapshot())
 THEN RAISE EXCEPTION 'CT213: cese CT no publicado' USING ERRCODE='23503'; END IF;
 SELECT * INTO i FROM vec_contratacion_temporal.origen_incorporacion_personal_b2_v1 x
  WHERE x.recibo_ref=c.incorporacion_b2_ref
    AND x.organizacion_ref=c.organizacion_ref AND x.expediente_ref=c.expediente_ref;
 IF NOT FOUND OR i.transaccion_publicacion IS NULL
    OR i.transaccion_publicacion>=pg_snapshot_xmin(pg_current_snapshot())
    OR i.material_sha256 IS DISTINCT FROM encode(sha256(convert_to(i.material,'UTF8')),'hex')
    OR i.recibo_json->>'recibo_ref' IS DISTINCT FROM i.recibo_ref
    OR i.recibo_json->>'outbox_ref' IS DISTINCT FROM i.outbox_ref
    OR i.recibo_json->'confirmacion' IS DISTINCT FROM i.material_json
    OR NOT EXISTS(SELECT 1 FROM vec_contratacion_temporal.outbox_expediente_integral x
       WHERE x.evento_ref=i.outbox_ref AND x.expediente_ref=i.expediente_ref
         AND x.tipo_evento='ct.incorporacion-personal.v1')
 THEN RAISE EXCEPTION 'CT213: incorporación B2 no publicada' USING ERRCODE='23503'; END IF;
 SELECT * INTO p FROM vec_contratacion_temporal.plan_incorporacion_personal_b2_v1 x
  WHERE x.organizacion_ref=i.organizacion_ref AND x.expediente_ref=i.expediente_ref AND x.plan_ref=i.plan_ref;
 IF NOT FOUND OR p.material_json#>'{material,bolsa_nativa}' IS NULL
    OR p.material_json#>'{material,bolsa}' IS NOT NULL
    OR i.material_json#>>'{plan_ref}' IS DISTINCT FROM p.plan_ref
    OR i.material_json#>>'{plan_sha256}' IS DISTINCT FROM p.material_sha256
    OR i.material_json#>'{hechos}' IS NULL
 THEN RAISE EXCEPTION 'CT213: plan nativo divergente' USING ERRCODE='23503'; END IF;
 SELECT * INTO n FROM vec_contratacion_temporal.origen_incorporacion_neutral_ct155(
  c.organizacion_ref,c.expediente_ref,c.incorporacion_b2_ref);
 IF NOT FOUND OR n.protocolo IS DISTINCT FROM 'personal_b2_v1'
    OR n.recibo_ref IS DISTINCT FROM i.recibo_ref
    OR n.relacion_ref IS DISTINCT FROM i.relacion_ref
    OR n.llamamiento_ref IS DISTINCT FROM c.llamamiento_ref
    OR n.inicio IS DISTINCT FROM c.inicio_incorporacion
 THEN RAISE EXCEPTION 'CT213: origen neutral divergente' USING ERRCODE='23503'; END IF;
 h:=i.material_json->'hechos';
 IF h->>'relacion_ref' IS DISTINCT FROM i.relacion_ref
    OR h->>'ocupacion_ref' IS DISTINCT FROM i.ocupacion_ref
    OR nullif(h->>'empleado_ref','') IS NULL
    OR nullif(h->>'relacion_recibo_ref','') IS NULL
    OR nullif(h->>'ocupacion_recibo_ref','') IS NULL
 THEN RAISE EXCEPTION 'CT213: hechos Personal incompletos' USING ERRCODE='23503'; END IF;
 SELECT * INTO o FROM vec_contratacion_temporal.outbox_expediente_integral x
  WHERE x.evento_ref=c.evento_ref AND x.expediente_ref=c.expediente_ref
    AND x.tipo_evento='ct.cese.v1';
 IF NOT FOUND OR o.version_expediente IS DISTINCT FROM c.version_esperada+1
    OR o.payload_huella_sha256 IS DISTINCT FROM encode(sha256(o.payload_canonico),'hex')
    OR convert_from(o.payload_canonico,'UTF8')::jsonb IS DISTINCT FROM jsonb_build_object(
      'esquema','vec.contratacion-temporal.cese.v1',
      'organizacion_ref',c.organizacion_ref,'expediente_ref',c.expediente_ref,
      'version_resultante',c.version_esperada+1,
      'incorporacion_ref',c.incorporacion_b2_ref,'llamamiento_ref',c.llamamiento_ref,
      'causa_clave',c.causa_clave,'fecha_efecto',to_char(c.fecha_efecto,'YYYY-MM-DD'),
      'recibo_ref',c.recibo_ref,'registrada_en',c.recibo_json->'registrada_en')
    OR c.recibo_json->>'recibo_ref' IS DISTINCT FROM c.recibo_ref
    OR c.recibo_json->>'evento_ref' IS DISTINCT FROM c.evento_ref
    OR c.recibo_json->>'auditoria_ref' IS DISTINCT FROM c.auditoria_ref
    OR c.recibo_json->>'actor_ref' IS DISTINCT FROM c.actor_ref
 THEN RAISE EXCEPTION 'CT213: outbox o recibo CT divergente' USING ERRCODE='23503'; END IF;
 IF p.material_json#>>'{material,bolsa_nativa,organizacion_ref}' IS DISTINCT FROM c.organizacion_ref
    OR p.material_json#>>'{material,bolsa_nativa,expediente_ref}' IS DISTINCT FROM c.expediente_ref
    OR p.material_json#>>'{material,bolsa_nativa,llamamiento_ref}' IS DISTINCT FROM c.llamamiento_ref
    OR p.material_json#>>'{material,categoria_ref}' IS DISTINCT FROM c.expediente_siguiente_json#>>'{analisis,categoria_ref}'
    OR coalesce(c.expediente_siguiente_json#>>'{analisis,modalidad_clave}','') !~ '^[a-z][a-z0-9_.-]{1,79}$'
    OR coalesce(c.expediente_siguiente_json#>>'{analisis,causa_clave}','') !~ '^[a-z][a-z0-9_.-]{1,79}$'
    OR c.fecha_efecto IS NULL OR NOT isfinite(c.fecha_efecto)
    OR c.fecha_efecto<n.inicio
    OR nullif(c.justificante_tipo,'') IS NULL OR nullif(c.justificante_ref,'') IS NULL
    OR coalesce(c.justificante_sha256,'') !~ '^[0-9a-f]{64}$'
 THEN RAISE EXCEPTION 'CT213: metadatos del cese divergentes' USING ERRCODE='23503'; END IF;
 posicion:=vec_contratacion_temporal.posicion_contrato_bolsa_v1(c.transaccion_publicacion);
 publicacion_ref:='evento:ct:contrato-bolsa:'||encode(sha256(convert_to(
   'cese'||chr(31)||c.evento_ref,'UTF8')),'hex');
 cuerpo:=jsonb_build_object('esquema','vec.contratacion-temporal.contrato-bolsa.v1',
  'tipo','cese','origen_ref',c.evento_ref,
  'organizacion_ref',c.organizacion_ref,'expediente_ref',c.expediente_ref,
  'llamamiento_ref',c.llamamiento_ref,
  'inicio',vec_contratacion_temporal.instante_contrato_bolsa_v1(n.inicio::timestamp AT TIME ZONE 'UTC'),
  'fin_previsto',vec_contratacion_temporal.instante_contrato_bolsa_v1(c.fecha_efecto::timestamp AT TIME ZONE 'UTC'),
  'modalidad_clave',c.expediente_siguiente_json#>>'{analisis,modalidad_clave}',
  'categoria_ref',c.expediente_siguiente_json#>>'{analisis,categoria_ref}',
  'causa_clave',c.causa_clave,
  'ocurrido_en',vec_contratacion_temporal.instante_contrato_bolsa_v1(c.registrada_en));
 evento:=cuerpo||jsonb_build_object('evento_ref',publicacion_ref);
 RETURN jsonb_build_object(
  'origen_ref',c.evento_ref,'origen_huella_sha256',encode(sha256(convert_to(evento::text,'UTF8')),'hex'),
  'origen_posicion',posicion,'origen_creada_en',c.confirmada_en,'evento',evento,
  'fecha_efecto',to_char(c.fecha_efecto,'YYYY-MM-DD'),
  'organizacion_ref',c.organizacion_ref,'expediente_ref',c.expediente_ref,
  'llamamiento_ref',c.llamamiento_ref,'recibo_ct_ref',c.recibo_ref,
  'incorporacion_recibo_ref',i.recibo_ref,
  'version_ct',c.version_esperada+1,'actor_ref',c.actor_ref,
  'relacion_ref',i.relacion_ref,'ocupacion_ref',i.ocupacion_ref,
  'empleado_ref',h->>'empleado_ref',
  'relacion_recibo_ref',h->>'relacion_recibo_ref',
  'ocupacion_recibo_ref',h->>'ocupacion_recibo_ref',
  'modalidad_clave',c.expediente_siguiente_json#>>'{analisis,modalidad_clave}',
  'causa_contrato_clave',c.expediente_siguiente_json#>>'{analisis,causa_clave}',
  'fuente_tipo',c.justificante_tipo,'fuente_ref',c.justificante_ref,
  'fuente_sha256',c.justificante_sha256);
END $f$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.probar_cese_personal_b2_para_bolsa_v1(text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.probar_cese_personal_b2_para_bolsa_v1(text)
 TO vec_bolsa_llamamientos_propietario;

-- El objeto CT155 conserva OID, propietario y la ACL exacta anterior.
DO $post$
DECLARE f regprocedure:='vec_contratacion_temporal.origen_incorporacion_neutral_ct155(text,text,text)'::regprocedure;
BEGIN
 IF (SELECT proowner<>'vec_contratacion_temporal_propietario'::regrole
       OR prosecdef OR proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on']::text[]
       OR proacl IS DISTINCT FROM ARRAY['vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario']::aclitem[]
      FROM pg_proc WHERE oid=f) IS NOT FALSE
    OR has_function_privilege('public','vec_contratacion_temporal.probar_cese_personal_b2_para_bolsa_v1(text)','EXECUTE')
    OR NOT has_function_privilege('vec_bolsa_llamamientos_propietario',
       'vec_contratacion_temporal.probar_cese_personal_b2_para_bolsa_v1(text)','EXECUTE')
    OR has_function_privilege('vec_bolsa_llamamientos_ejecutor',
       'vec_contratacion_temporal.probar_cese_personal_b2_para_bolsa_v1(text)','EXECUTE')
 THEN RAISE EXCEPTION 'CT213: proyección neutral cambió ACL' USING ERRCODE='42501'; END IF;
END $post$;
COMMIT;
