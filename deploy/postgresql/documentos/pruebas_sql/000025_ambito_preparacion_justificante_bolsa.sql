\set ON_ERROR_STOP on
-- Tras D25. Vector de RecursoAutorizable.HuellaContextoAutorizacionSHA256:
-- internal/app/bootstrap/bolsa_justificante_autoridad.go y
-- internal/vec/domain/autorizacion.go. No fabrica capacidades V3 ni consume actos.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $prueba$
DECLARE p pg_proc%ROWTYPE; definicion text; canon text; huella text;
 persona constant text:='per_0123456789abcdefghijkl';
 candidato constant text:='can_0123456789abcdefghijkl';
 bolsa constant text:='bolsa:2026:01';
BEGIN
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=
  'vec_documentos.resolver_justificante_renuncia_bolsa_v1(text,text,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 definicion:=pg_get_functiondef(p.oid);
 IF p.proowner IS DISTINCT FROM 'vec_documentos_propietario'::regrole OR
  p.prosecdef IS NOT TRUE OR p.provolatile IS DISTINCT FROM 'v' OR
  p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on',
   'TimeZone=UTC','lock_timeout=2s','statement_timeout=15s']::text[] OR
  p.proacl::text IS DISTINCT FROM
   '{vec_documentos_propietario=X/vec_documentos_propietario,vec_documentos_bolsa_subida_datos=X/vec_documentos_propietario}' OR
  encode(sha256(convert_to(definicion,'UTF8')),'hex') IS DISTINCT FROM
   '606edd1ed9c21e52dd937a55029d406625787c6cfc266a007ee82915ebf23e39'
 THEN RAISE EXCEPTION 'D25 prueba: función, propietario o ACL divergentes'; END IF;
 -- La huella se coteja antes del consumo y de la proyección. La persona
 -- continúa como principal de la decisión y titular de la proyección.
 IF position('''{"ambitos":{"candidato_ref":"%s"},"atributos":{"bolsa_ref":"%s","candidato_ref":"%s"}}''' IN definicion)=0 OR
  position('p_candidato_ref,p_bolsa_ref,p_candidato_ref' IN definicion)=0 OR
  position('c->>''huella_efecto_sha256'' IS DISTINCT FROM h' IN definicion)=0 OR
  position('d->>''contexto_recurso_huella_sha256'' IS DISTINCT FROM h' IN definicion)=0 OR
  position('v.huella_efecto_sha256 IS DISTINCT FROM h' IN definicion)=0 OR
  position('d->>''principal_id'' IS DISTINCT FROM p_persona_ref' IN definicion)=0 OR
  position('p.persona_ref IS DISTINCT FROM p_persona_ref' IN definicion)=0 OR
  position('candidato_propio_contexto_justificante_bolsa_v1(' IN definicion)=0 OR
  position('''{"ambitos":{"persona_ref":"%s"},"atributos":{"bolsa_ref":"%s","candidato_ref":"%s"}}''' IN definicion)<>0
 THEN RAISE EXCEPTION 'D25 prueba: contrato preparación Go/SQL divergente'; END IF;
 canon:=format(
  '{"ambitos":{"candidato_ref":"%s"},"atributos":{"bolsa_ref":"%s","candidato_ref":"%s"}}',
  candidato,bolsa,candidato);
 huella:=encode(sha256(convert_to(canon,'UTF8')),'hex');
 IF canon IS DISTINCT FROM
  '{"ambitos":{"candidato_ref":"can_0123456789abcdefghijkl"},"atributos":{"bolsa_ref":"bolsa:2026:01","candidato_ref":"can_0123456789abcdefghijkl"}}' OR
  huella IS DISTINCT FROM 'c64dce39c0589638a8aa3ec9ac75f6722eb5d13551e2728a35d23e3d2979e517' OR
  huella=encode(sha256(convert_to(format(
   '{"ambitos":{"persona_ref":"%s"},"atributos":{"bolsa_ref":"%s","candidato_ref":"%s"}}',
   persona,bolsa,candidato),'UTF8')),'hex') OR
  huella=encode(sha256(convert_to(format(
   '{"ambitos":{"candidato_ref":"%s"},"atributos":{"bolsa_ref":"%s","candidato_ref":"%s"}}',
   'can_zzzzzzzzzzzzzzzzzzzzzz',bolsa,candidato),'UTF8')),'hex') OR
  huella=encode(sha256(convert_to(format(
   '{"ambitos":{"candidato_ref":"%s"},"atributos":{"bolsa_ref":"%s","candidato_ref":"%s"}}',
   candidato,'bolsa:ajena',candidato),'UTF8')),'hex') OR
  huella=encode(sha256(convert_to(format(
   '{"ambitos":{"candidato_ref":"%s"},"atributos":{"bolsa_ref":"%s","candidato_ref":"%s"}}',
   candidato,bolsa,'can_zzzzzzzzzzzzzzzzzzzzzz'),'UTF8')),'hex')
 THEN RAISE EXCEPTION 'D25 prueba: vector Go o separación de ámbitos divergente'; END IF;
END $prueba$;
ROLLBACK;
-- Requiere el LOGIN documental externo creado por Dirección. Material V3
-- deliberadamente inválido: el caso correcto debe superar el cotejo D25 y
-- pararse en el consumidor V3; las variantes se deniegan antes del consumo.
BEGIN;
SET TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TIME ZONE 'UTC';
SET SESSION AUTHORIZATION vec_externo_documentos_bolsa_datos;
DO $denegaciones$
DECLARE persona constant text:='per_0123456789abcdefghijkl';
 candidato constant text:='can_0123456789abcdefghijkl';
 bolsa constant text:='bolsa:2026:01';
 modo text; contexto jsonb; contexto_bytes bytea; h text; c jsonb; d jsonb;
 fallo text; desde text; hasta text;
BEGIN
 desde:=(clock_timestamp()-interval '1 hour')::text;
 hasta:=(clock_timestamp()+interval '1 hour')::text;
 contexto:=jsonb_build_object('esquema','vec.contexto-actor.vinculado.v2',
  'persona_ref',persona,'estado','activo','vigente_desde',desde,'vigente_hasta',hasta,
  'vinculos',jsonb_build_array(jsonb_build_object('tipo','candidato',
   'estado','activo','referencia',candidato,'vigente_desde',desde,'vigente_hasta',hasta)));
 contexto_bytes:=convert_to(contexto::text,'UTF8');
 FOREACH modo IN ARRAY ARRAY['correcto','ambito_persona','candidato_ajeno',
  'bolsa_ajena','atributo_candidato_ajeno','principal_ajeno'] LOOP
  h:=encode(sha256(convert_to(CASE modo
   WHEN 'ambito_persona' THEN format(
    '{"ambitos":{"persona_ref":"%s"},"atributos":{"bolsa_ref":"%s","candidato_ref":"%s"}}',
    persona,bolsa,candidato)
   WHEN 'candidato_ajeno' THEN format(
    '{"ambitos":{"candidato_ref":"%s"},"atributos":{"bolsa_ref":"%s","candidato_ref":"%s"}}',
    'can_zzzzzzzzzzzzzzzzzzzzzz',bolsa,candidato)
   WHEN 'bolsa_ajena' THEN format(
    '{"ambitos":{"candidato_ref":"%s"},"atributos":{"bolsa_ref":"%s","candidato_ref":"%s"}}',
    candidato,'bolsa:ajena',candidato)
   WHEN 'atributo_candidato_ajeno' THEN format(
    '{"ambitos":{"candidato_ref":"%s"},"atributos":{"bolsa_ref":"%s","candidato_ref":"%s"}}',
    candidato,bolsa,'can_zzzzzzzzzzzzzzzzzzzzzz')
   ELSE format(
    '{"ambitos":{"candidato_ref":"%s"},"atributos":{"bolsa_ref":"%s","candidato_ref":"%s"}}',
    candidato,bolsa,candidato) END,'UTF8')),'hex');
  c:=jsonb_build_object('contexto_ref','ctx:prueba',
   'huella_contexto_sha256',encode(sha256(contexto_bytes),'hex'),
   'efecto_ref',bolsa,'operacion','bolsa.justificante_renuncia.subir_propio',
   'audiencia_consumo','vec_documentos.bolsa.justificante_renuncia.preparar_subida.v1',
   'huella_efecto_sha256',h);
  d:=jsonb_build_object('accion','bolsa.justificante_renuncia.subir_propio',
   'recurso_ref',bolsa,'modulo_id','bolsa','tipo_recurso','justificante_renuncia',
   'finalidad','custodiar_justificante_renuncia_bolsa',
   'principal_id',CASE WHEN modo='principal_ajeno' THEN 'per_ajena' ELSE persona END,
   'contexto_recurso_huella_sha256',h,
   'campos_permitidos','["contenido","estado","recibo"]'::jsonb,
   'obligaciones','[]'::jsonb,
   'vinculo_autenticacion_actor',jsonb_build_object(
    'registro_contexto_ref','ctx:prueba',
    'contexto_actor_huella_sha256',encode(sha256(contexto_bytes),'hex'),
    'superficie','externa_personal'));
  fallo:='sin_error';
  BEGIN
   PERFORM vec_documentos.resolver_justificante_renuncia_bolsa_v1(
    persona,candidato,bolsa,'ctx:prueba',clock_timestamp(),
    convert_to(c::text,'UTF8'),convert_to(d::text,'UTF8'),NULL,contexto_bytes,
    1,1,NULL,NULL,NULL,NULL);
  EXCEPTION WHEN OTHERS THEN GET STACKED DIAGNOSTICS fallo=MESSAGE_TEXT;
  END;
  IF modo='correcto' THEN
   IF fallo='sin_error' OR fallo LIKE 'D22: preparación propia %' THEN
    RAISE EXCEPTION 'D25 prueba: caso Go no superó preparación (%)',fallo; END IF;
  ELSIF fallo IS DISTINCT FROM 'D22: preparación propia divergente' THEN
   RAISE EXCEPTION 'D25 prueba: % no se denegó antes del consumo (%)',modo,fallo;
  END IF;
 END LOOP;
END $denegaciones$;
ROLLBACK;
\echo D25_PRUEBA_CONTRATO_OK
