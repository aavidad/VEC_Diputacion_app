\set ON_ERROR_STOP on
-- Superficie EXCLUSIVA del contenedor efímero del ensayo AD3-61. Nunca se
-- instala en una base conservada. Requiere antes las sondas de AD3-59
-- (esquema prueba59 con el alta CT ya consumida). No crea claves,
-- configuración ni decisiones: deriva material coherente para D7b/D7c cuyo
-- contexto añade un vínculo de empleado activo, con las huellas recalculadas,
-- de modo que el núcleo solo puede detenerse después de la ligadura.
BEGIN;
SET LOCAL search_path=pg_catalog;
CREATE SCHEMA prueba61 AUTHORIZATION postgres;
REVOKE ALL ON SCHEMA prueba61 FROM PUBLIC;
CREATE TABLE prueba61.material_personal(caso text PRIMARY KEY, texto text NOT NULL);
REVOKE ALL ON prueba61.material_personal FROM PUBLIC;

-- Material de fachada: operación, audiencia, recurso y campos exactos.
CREATE FUNCTION prueba61.preparar(p_caso text,p_operacion text,p_audiencia text,
 p_tipo text,p_finalidad text,p_campos jsonb,p_recurso text,p_huella text) RETURNS void
LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
DECLARE b prueba59.material; c jsonb; d jsonb; x jsonb; contexto bytea; decision bytea;
BEGIN
 SELECT * INTO STRICT b FROM prueba59.material WHERE caso='ct_alta';
 x:=convert_from(b.contexto,'UTF8')::jsonb || jsonb_build_object('vinculos',jsonb_build_array(
    jsonb_build_object('tipo','empleado','estado','activo','referencia','emp_prueba61aaaaaaaaaaaaaaaaaaaaaa')));
 contexto:=convert_to(x::text,'UTF8');
 d:=convert_from(b.decision,'UTF8')::jsonb
   || jsonb_build_object('decision_ref','decision:prueba61:'||p_caso,'accion',p_operacion,
      'modulo_id','personal','tipo_recurso',p_tipo,'finalidad',p_finalidad,'recurso_ref',p_recurso,
      'contexto_recurso_huella_sha256',p_huella,'campos_permitidos',p_campos,'obligaciones','[]'::jsonb);
 d:=jsonb_set(d,'{vinculo_autenticacion_actor,contexto_actor_huella_sha256}',
    to_jsonb(encode(sha256(contexto),'hex')));
 decision:=convert_to(d::text,'UTF8');
 c:=convert_from(b.capacidad,'UTF8')::jsonb
   || jsonb_build_object('decision_ref',d->>'decision_ref','operacion',p_operacion,
      'audiencia_consumo',p_audiencia,'efecto_ref',p_recurso,'huella_efecto_sha256',p_huella,
      'huella_decision_sha256',encode(sha256(decision),'hex'),
      'huella_contexto_sha256',encode(sha256(contexto),'hex'),
      'nonce',encode(sha256(convert_to('nonce:prueba61:'||p_caso,'UTF8')),'hex'));
 INSERT INTO prueba59.material VALUES (p_caso,
  vec_autorizacion_atestada_v3.capacidad_canonica(c),decision,b.motivo,contexto,
  b.persona_version,b.perfil_version,b.payload,b.sobre,b.evidencia,b.raiz);
END $f$;

-- Material de lista (D7b competencias o D7c lista competente) que supera las
-- comprobaciones de Personal: recurso = persona del actor y huella calculada
-- exactamente como la calcula Personal 000014/000015.
CREATE FUNCTION prueba61.preparar_lista(p_caso text,p_esquema text,p_operacion text,
 p_audiencia text,p_tipo text,p_finalidad text,p_campos jsonb) RETURNS void
LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
DECLARE b prueba59.material; x jsonb; texto text; persona text; sha text; recurso text;
BEGIN
 SELECT * INTO STRICT b FROM prueba59.material WHERE caso='ct_alta';
 x:=convert_from(b.contexto,'UTF8')::jsonb;
 persona:=x->>'persona_ref';
 texto:=jsonb_build_object('esquema',p_esquema,
  'fecha_referencia',((clock_timestamp() AT TIME ZONE 'UTC')::date)::text,
  'identidad',jsonb_build_object('actor_ref',x->'principal_ref','contexto_actor_ref',x->'contexto_actor_ref',
   'contexto_version',x->'contexto_version','cuenta_ref',x->'cuenta_ref','cuenta_version',x->'cuenta_version',
   'empleado_ref','emp_prueba61aaaaaaaaaaaaaaaaaaaaaa','perfil_ref',x->'perfil_activo_ref',
   'perfil_version',x->'perfil_version','persona_ref',persona,'persona_version',x->'persona_version'))::text;
 sha:=encode(sha256(convert_to(texto,'UTF8')),'hex');
 recurso:=encode(sha256(convert_to('{"ambitos":{"persona_ref":'||to_jsonb(persona)::text||'},"atributos":'
   ||'{"fecha_referencia":'||to_jsonb(((clock_timestamp() AT TIME ZONE 'UTC')::date)::text)::text
   ||',"material_sha256":'||to_jsonb(sha)::text||',"operacion":"lista"}}','UTF8')),'hex');
 PERFORM prueba61.preparar(p_caso,p_operacion,p_audiencia,p_tipo,p_finalidad,p_campos,persona,recurso);
 INSERT INTO prueba61.material_personal VALUES (p_caso,texto);
END $f$;

-- Llamadas del login a las funciones públicas de Personal (SECURITY INVOKER:
-- el login de sesión es el que Personal y el núcleo cotejan).
CREATE FUNCTION prueba61.competencias(p_caso text) RETURNS void
LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
DECLARE m prueba59.material; t text;
BEGIN
 SELECT * INTO STRICT m FROM prueba59.material WHERE caso=p_caso;
 SELECT texto INTO STRICT t FROM prueba61.material_personal WHERE caso=p_caso;
 PERFORM vec_personal.consultar_competencias_asignacion_dietas_v1(t,m.capacidad,m.decision,m.motivo,
  m.contexto,m.persona_version,m.perfil_version,m.payload,m.sobre,m.evidencia,m.raiz);
END $f$;
CREATE FUNCTION prueba61.competentes(p_caso text) RETURNS void
LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
DECLARE m prueba59.material; t text;
BEGIN
 SELECT * INTO STRICT m FROM prueba59.material WHERE caso=p_caso;
 SELECT texto INTO STRICT t FROM prueba61.material_personal WHERE caso=p_caso;
 PERFORM vec_personal.consultar_rectificaciones_competentes_dietas_v1(t,m.capacidad,m.decision,m.motivo,
  m.contexto,m.persona_version,m.perfil_version,m.payload,m.sobre,m.evidencia,m.raiz);
END $f$;
REVOKE ALL ON FUNCTION prueba61.competencias(text), prueba61.competentes(text) FROM PUBLIC;
COMMIT;
