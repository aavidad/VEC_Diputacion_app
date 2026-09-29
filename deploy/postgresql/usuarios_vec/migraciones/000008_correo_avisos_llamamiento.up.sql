\set ON_ERROR_STOP on
-- Usuarios 000008: correo activo de «Mis correos» para el aviso de un
-- llamamiento de Bolsa. Aplicar tras AD3-109, ContextoActor 000010 y
-- Usuarios 000004. Sin roles nuevos.
--
-- Una sola lectura nominal, desde la superficie interna (RRHH emite el
-- llamamiento). Consume una V3 fresca antes de mirar ninguna fila y devuelve,
-- como mucho, el sobre cifrado del correo ACTIVO y VERIFICADO de la persona
-- candidata, y sólo si ese correo se añadió y confirmó desde el área personal
-- externa: a una persona candidata sólo se le escribe al correo que ella dio
-- en la superficie externa. Nada de la lista, del estado del resto de
-- direcciones ni de otras personas. Si no hay tal correo, responde que no lo
-- hay; Bolsa usará entonces el correo del alta.
BEGIN;
SET LOCAL ROLE vec_usuarios_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_usuarios:migracion:000008',0));
DO $pre$ BEGIN
 IF current_user<>'vec_usuarios_propietario'
    OR to_regclass('vec_usuarios.correos_contexto') IS NULL
    OR to_regclass('vec_usuarios.correos_direccion') IS NULL
    OR to_regprocedure('vec_usuarios.contexto_autorizado_correos(text,text[])') IS NULL
    OR to_regprocedure('vec_usuarios.superficie_sesion_correos()') IS NULL
    OR to_regprocedure('vec_usuarios.sobre_correo_json(vec_usuarios.correos_direccion)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_correo_avisos_llamamiento_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR NOT has_function_privilege('vec_autorizacion_atestada_v3.consumir_correo_avisos_llamamiento_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR to_regprocedure('vec_contexto_actor_v1.persona_candidato_avisos_v1(text)') IS NULL
    OR NOT has_function_privilege('vec_contexto_actor_v1.persona_candidato_avisos_v1(text)','EXECUTE')
    OR to_regprocedure('vec_usuarios.correo_activo_avisos_llamamiento_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR NOT EXISTS(SELECT 1 FROM pg_constraint WHERE conrelid='vec_usuarios.correos_contexto'::regclass
      AND conname='correos_contexto_modo_check'
      AND pg_get_constraintdef(oid)='CHECK ((modo = ANY (ARRAY[''consultar''::text, ''recuperar''::text, ''actualizar''::text, ''verificar''::text, ''envio''::text])))')
 THEN RAISE EXCEPTION 'Usuarios 000008: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

-- Nuevo modo de contexto, sólo de lectura y sólo para las tres tablas que la
-- selección necesita. Las políticas se suman a las existentes.
ALTER TABLE vec_usuarios.correos_contexto DROP CONSTRAINT correos_contexto_modo_check;
ALTER TABLE vec_usuarios.correos_contexto ADD CONSTRAINT correos_contexto_modo_check
 CHECK(modo IN ('consultar','recuperar','actualizar','verificar','envio','avisos'));
CREATE POLICY correos_direccion_avisos ON vec_usuarios.correos_direccion FOR SELECT TO vec_usuarios_propietario
 USING (vec_usuarios.contexto_autorizado_correos(persona_ref,ARRAY['avisos']));
CREATE POLICY correos_desafio_avisos ON vec_usuarios.correos_desafio FOR SELECT TO vec_usuarios_propietario
 USING (vec_usuarios.contexto_autorizado_correos(persona_ref,ARRAY['avisos']));
CREATE POLICY correos_envio_avisos ON vec_usuarios.correos_envio FOR SELECT TO vec_usuarios_propietario
 USING (vec_usuarios.contexto_autorizado_correos(persona_ref,ARRAY['avisos']));

-- El material es el JSON literal que Go serializa
-- (canonico.SerializarMaterialCorreoAvisos). Las referencias no admiten
-- caracteres que Go y PostgreSQL escapen distinto, para que la huella del
-- recurso sea idéntica en los dos lados.
CREATE FUNCTION vec_usuarios.correo_activo_avisos_llamamiento_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE m jsonb; c jsonb; d jsonb; k text[]; h text; x record; persona text; fila vec_usuarios.correos_direccion;
 xid_actual xid8; n integer; encontrado boolean:=false;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
    OR vec_usuarios.superficie_sesion_correos() IS DISTINCT FROM 'interna_corporativa'
    OR p_material IS NULL OR octet_length(p_material) NOT BETWEEN 2 AND 4096
    OR p_capacidad IS NULL OR p_decision IS NULL OR p_persona_version IS NULL OR p_perfil_version IS NULL
 THEN RAISE EXCEPTION 'Usuarios: aviso de correo denegado' USING ERRCODE='42501'; END IF;
 BEGIN m:=p_material::jsonb; c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'Usuarios: material de aviso inválido' USING ERRCODE='22023'; END;
 IF jsonb_typeof(m) IS DISTINCT FROM 'object' OR jsonb_typeof(c) IS DISTINCT FROM 'object' OR jsonb_typeof(d) IS DISTINCT FROM 'object'
 THEN RAISE EXCEPTION 'Usuarios: material de aviso inválido' USING ERRCODE='22023'; END IF;
 SELECT array_agg(z ORDER BY z) INTO k FROM jsonb_object_keys(m) z;
 IF k IS DISTINCT FROM ARRAY['ambito_ref','bolsa_ref','candidato_ref','esquema','finalidad_ref','llamamiento_ref','superficie','unidad_ref']
    OR EXISTS(SELECT 1 FROM jsonb_each(m) z WHERE jsonb_typeof(z.value)<>'string'
      OR octet_length(z.value#>>'{}') NOT BETWEEN 1 AND 512 OR (z.value#>>'{}') ~ '[<>&"\\[:cntrl:]\u2028\u2029]')
    OR m->>'esquema' IS DISTINCT FROM 'vec.usuarios.correo-avisos-llamamiento.v1'
    OR m->>'candidato_ref' !~ '^can_[A-Za-z0-9_-]{22,128}$'
    OR m->>'llamamiento_ref' !~ '^llamamiento:[0-9a-f]{64}$'
 THEN RAISE EXCEPTION 'Usuarios: material de aviso inválido' USING ERRCODE='22023'; END IF;
 h:=encode(sha256(convert_to('{"ambitos":{"ambito_ref":'||to_jsonb(m->>'ambito_ref')::text||',"unidad_ref":'||to_jsonb(m->>'unidad_ref')::text||
   '},"atributos":{"material_sha256":"'||encode(sha256(convert_to(p_material,'UTF8')),'hex')||'"}}','UTF8')),'hex');
 IF m->>'superficie' IS DISTINCT FROM 'interna_corporativa'
    OR m->>'superficie' IS DISTINCT FROM d #>> '{vinculo_autenticacion_actor,superficie}'
    OR m->>'finalidad_ref' IS DISTINCT FROM 'gestion_llamamientos_bolsa'
    OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_usuarios.correos.avisos_llamamiento.interna_corporativa.v1'
    OR c->>'operacion' IS DISTINCT FROM 'llamamiento.emitir.v1' OR d->>'accion' IS DISTINCT FROM 'llamamiento.emitir.v1'
    OR d->>'modulo_id' IS DISTINCT FROM 'bolsa' OR d->>'tipo_recurso' IS DISTINCT FROM 'bolsa_constituida'
    OR d->>'finalidad' IS DISTINCT FROM m->>'finalidad_ref' OR d->'concedida' IS DISTINCT FROM 'true'::jsonb
    OR d->>'recurso_ref' IS DISTINCT FROM m->>'bolsa_ref' OR c->>'efecto_ref' IS DISTINCT FROM m->>'bolsa_ref'
    OR d->>'decision_ref' IS NULL OR length(d->>'decision_ref') NOT BETWEEN 1 AND 256
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM h OR c->>'huella_efecto_sha256' IS DISTINCT FROM h
 THEN RAISE EXCEPTION 'Usuarios: aviso de correo no autorizado' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_correo_avisos_llamamiento_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE OR x.efecto_ref IS DISTINCT FROM m->>'bolsa_ref'
    OR x.decision_ref IS DISTINCT FROM d->>'decision_ref' OR x.huella_efecto_sha256 IS DISTINCT FROM h
 THEN RAISE EXCEPTION 'Usuarios: consumo de aviso divergente' USING ERRCODE='42501'; END IF;
 persona:=vec_contexto_actor_v1.persona_candidato_avisos_v1(m->>'candidato_ref');
 IF persona IS NOT NULL THEN
  xid_actual:=pg_current_xact_id();
  DELETE FROM vec_usuarios.correos_contexto WHERE backend_pid=pg_backend_pid() AND xid<>xid_actual;
  IF EXISTS(SELECT 1 FROM vec_usuarios.correos_contexto WHERE xid=xid_actual AND backend_pid=pg_backend_pid() AND sesion=session_user)
  THEN RAISE EXCEPTION 'Usuarios: contexto correo ya activo' USING ERRCODE='42501'; END IF;
  INSERT INTO vec_usuarios.correos_contexto(xid,backend_pid,sesion,superficie,persona_ref,modo,accion,decision_ref,auditoria_ref,consumo_huella_sha256)
  VALUES(xid_actual,pg_backend_pid(),session_user,'interna_corporativa',persona,'avisos','vec.correos.avisos_llamamiento',x.decision_ref,x.auditoria_ref,x.consumo_huella_sha256);
  SELECT * INTO fila FROM vec_usuarios.correos_direccion dir
   WHERE dir.persona_ref=persona AND dir.activo AND dir.estado='verificado'
     AND EXISTS(SELECT 1 FROM vec_usuarios.correos_desafio h
                JOIN vec_usuarios.correos_envio e ON e.desafio_ref=h.desafio_ref AND e.persona_ref=h.persona_ref AND e.correo_ref=h.correo_ref
               WHERE h.persona_ref=dir.persona_ref AND h.correo_ref=dir.correo_ref AND h.estado='usado'
                 AND e.tipo='verificacion' AND e.superficie='externa_personal');
  encontrado:=FOUND;
  DELETE FROM vec_usuarios.correos_contexto WHERE xid=xid_actual AND backend_pid=pg_backend_pid()
   AND sesion=session_user AND persona_ref=persona AND modo='avisos';
  GET DIAGNOSTICS n=ROW_COUNT;
  IF n<>1 THEN RAISE EXCEPTION 'Usuarios: contexto correo incompleto' USING ERRCODE='42501'; END IF;
 END IF;
 IF NOT encontrado THEN
  RETURN jsonb_build_object('encontrado',false,'auditoria_ref',x.auditoria_ref);
 END IF;
 RETURN jsonb_build_object('encontrado',true,'auditoria_ref',x.auditoria_ref,'persona_ref',persona,
  'correo_ref',fila.correo_ref,'sobre',vec_usuarios.sobre_correo_json(fila));
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios.correo_activo_avisos_llamamiento_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_usuarios.correo_activo_avisos_llamamiento_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_usuarios_ejecutor_interno;

DO $post$
DECLARE f regprocedure:='vec_usuarios.correo_activo_avisos_llamamiento_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
BEGIN
 IF (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'vec_usuarios_propietario'::regrole
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS NOT TRUE
    OR NOT has_function_privilege('vec_usuarios_ejecutor_interno',f,'EXECUTE')
    OR has_function_privilege('vec_usuarios_ejecutor_externo',f,'EXECUTE')
    OR has_table_privilege('vec_usuarios_ejecutor_interno','vec_usuarios.correos_direccion','SELECT')
    OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
      WHERE p.oid=f AND (a.grantee=0 OR a.grantee NOT IN (p.proowner,'vec_usuarios_ejecutor_interno'::regrole)))
 THEN RAISE EXCEPTION 'Usuarios 000008: ACL incompatible' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
