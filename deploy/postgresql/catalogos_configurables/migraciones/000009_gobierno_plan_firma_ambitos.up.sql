\set ON_ERROR_STOP on
-- CC9: confirmación del gobierno del plan nominal de firma con el recurso que
-- lleva organización y unidad (AD201, aprobado por dirección el 5/10/2026).
-- Es la de CC7 salvo dos parámetros y la huella de contexto, que incluye los
-- ámbitos; CC7 la calculaba sin ellos y no ligaría ninguna decisión emitida
-- con ámbitos. Mismas tablas, mismo replay y mismos recibos. Sólo la ejecuta el
-- propietario AD. La v1 de CC7 queda instalada para el replay histórico, que
-- no existe todavía. Una sola vez; sin DOWN.
BEGIN;
SET LOCAL ROLE vec_catalogos_configurables_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='2min';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_catalogos_configurables:migracion:000009',0));
DO $pre$
BEGIN
 IF current_user<>'vec_catalogos_configurables_propietario'
    OR pg_catalog.to_regprocedure('vec_catalogos_configurables.confirmar_gobierno_plan_nominal_firma_v1(bytea,jsonb)') IS NULL
    OR pg_catalog.to_regprocedure('vec_catalogos_configurables.confirmar_gobierno_plan_nominal_firma_v2(bytea,text,text,jsonb)') IS NOT NULL
    OR pg_catalog.to_regprocedure('vec_catalogos_configurables.validar_plan_nominal_firma_v1(bytea,text)') IS NULL
    OR pg_catalog.to_regprocedure('vec_catalogos_configurables.json_plan_sin_duplicados_v1(json,integer)') IS NULL
    OR NOT pg_catalog.has_function_privilege('vec_catalogos_configurables_propietario',
      'vec_autorizacion_atestada_v3.comprobar_consumo_gobierno_plan_firma_v1(jsonb)','EXECUTE') THEN
  RAISE EXCEPTION 'CC9: PARO clave=preimagen actual=incompatible esperado=CC7_sin_CC9' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE FUNCTION vec_catalogos_configurables.confirmar_gobierno_plan_nominal_firma_v2(
 p_material_exacto bytea,p_organizacion_ref text,p_unidad_ref text,p_consumo jsonb)
RETURNS TABLE(recibo_ref text,estado text,revision bigint,huella_comun text,publicacion_sha256 text,
 actor_ref text,confirmado_en timestamptz,auditoria_ref text,outbox_recibo_ref text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog SET row_security='on' SET lock_timeout='2s' SET statement_timeout='30s' SET TimeZone='UTC' AS $f$
DECLARE m jsonb; c jsonb; traza jsonb; evento jsonb; v3 jsonb;
 canon bytea; traza_bytes bytea; evento_bytes bytea;
 op text; cat text; actor text; clave text; accion_evento text;
 ver bigint; rev bigint; rev_esperada bigint; h text; h_esperada text;
 h_material text; h_contexto text; h_publicacion text; recibo text; existe boolean;
 actor_original text; fecha_original timestamptz(6); auditoria_original text; outbox_original text;
 actual vec_catalogos_configurables.plan_firma_control%ROWTYPE;
 previo vec_catalogos_configurables.plan_firma_efecto%ROWTYPE;
 origen jsonb; momento timestamptz(6);
BEGIN
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable'
    OR pg_catalog.current_setting('transaction_read_only')<>'off'
    OR pg_catalog.pg_is_in_recovery()
    OR p_material_exacto IS NULL OR pg_catalog.octet_length(p_material_exacto) NOT BETWEEN 2 AND 4194304
    OR pg_catalog.jsonb_typeof(p_consumo) IS DISTINCT FROM 'object'
    OR (SELECT count(*) FROM pg_catalog.jsonb_object_keys(p_consumo))<>7
    OR NOT (p_consumo ?& ARRAY['consumo','actor_ref','perfil_ref','accion','finalidad','proceso','canal'])
    OR pg_catalog.jsonb_typeof(p_consumo->'consumo') IS DISTINCT FROM 'object'
    -- Ámbitos del recurso de gobierno, con los formatos de AD201 y AUT52.
    OR (p_organizacion_ref ~ '^org_[a-z0-9]{16,80}$') IS NOT TRUE
    OR (p_unidad_ref ~ '^[A-Za-z0-9][A-Za-z0-9:._-]{2,159}$') IS NOT TRUE THEN
  RAISE EXCEPTION 'CC9: material o consumo inválido' USING ERRCODE='42501'; END IF;
 BEGIN
  IF NOT vec_catalogos_configurables.json_plan_sin_duplicados_v1(pg_catalog.convert_from(p_material_exacto,'UTF8')::json,0) THEN
   RAISE EXCEPTION 'CC9: material ambiguo' USING ERRCODE='22023'; END IF;
  m:=pg_catalog.convert_from(p_material_exacto,'UTF8')::jsonb;
  canon:=pg_catalog.decode(m->>'catalogo_canonico_base64','base64');
  traza_bytes:=pg_catalog.decode(m->>'traza_canonica_base64','base64');
  evento_bytes:=pg_catalog.decode(m->>'evento_canonico_base64','base64');
  IF NOT vec_catalogos_configurables.json_plan_sin_duplicados_v1(pg_catalog.convert_from(traza_bytes,'UTF8')::json,0)
     OR NOT vec_catalogos_configurables.json_plan_sin_duplicados_v1(pg_catalog.convert_from(evento_bytes,'UTF8')::json,0) THEN
   RAISE EXCEPTION 'CC9: evidencia ambigua' USING ERRCODE='22023'; END IF;
  traza:=pg_catalog.convert_from(traza_bytes,'UTF8')::jsonb;
  evento:=pg_catalog.convert_from(evento_bytes,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception THEN
  RAISE EXCEPTION 'CC9: bytes de material inválidos' USING ERRCODE='22023'; END;
 IF pg_catalog.jsonb_typeof(m) IS DISTINCT FROM 'object'
    OR (SELECT count(*) FROM pg_catalog.jsonb_object_keys(m))<>13
    OR NOT (m ?& ARRAY['esquema','operacion','catalogo_id','version','revision_esperada',
      'huella_esperada','clave_operacion','catalogo_canonico_base64','catalogo_sha256',
      'traza_canonica_base64','traza_sha256','evento_canonico_base64','evento_sha256'])
    OR m->>'esquema' IS DISTINCT FROM 'vec.catalogos.plan-firma.gobierno.v1'
    OR (m->>'operacion' IN ('crear','actualizar','publicar','retirar')) IS NOT TRUE
    OR (m->>'version' ~ '^[1-9][0-9]{0,9}$') IS NOT TRUE
    OR (m->>'revision_esperada' ~ '^(0|[1-9][0-9]{0,9})$') IS NOT TRUE
    OR (m->>'clave_operacion' ~ '^[A-Za-z0-9][A-Za-z0-9._-]{15,127}$') IS NOT TRUE
    OR (m->>'catalogo_sha256' ~ '^[0-9a-f]{64}$') IS NOT TRUE
    OR (m->>'traza_sha256' ~ '^[0-9a-f]{64}$') IS NOT TRUE
    OR (m->>'evento_sha256' ~ '^[0-9a-f]{64}$') IS NOT TRUE
    OR pg_catalog.octet_length(traza_bytes) NOT BETWEEN 2 AND 65536
    OR pg_catalog.octet_length(evento_bytes) NOT BETWEEN 2 AND 65536
    OR pg_catalog.encode(pg_catalog.sha256(traza_bytes),'hex') IS DISTINCT FROM m->>'traza_sha256'
    OR pg_catalog.encode(pg_catalog.sha256(evento_bytes),'hex') IS DISTINCT FROM m->>'evento_sha256' THEN
  RAISE EXCEPTION 'CC9: esquema de material inválido' USING ERRCODE='22023'; END IF;
 op:=m->>'operacion'; cat:=m->>'catalogo_id'; clave:=m->>'clave_operacion';
 ver:=(m->>'version')::bigint; rev_esperada:=(m->>'revision_esperada')::bigint;
 h:=m->>'catalogo_sha256'; h_esperada:=m->>'huella_esperada';
 IF ver>2147483647 OR rev_esperada>2147483647
    OR (cat ~ '^[a-z][a-z0-9._-]{2,127}$') IS NOT TRUE
    OR (op='crear' AND m->'huella_esperada' IS DISTINCT FROM 'null'::jsonb)
    OR (op<>'crear' AND (h_esperada ~ '^[0-9a-f]{64}$') IS NOT TRUE) THEN
  RAISE EXCEPTION 'CC9: referencia o CAS inválido' USING ERRCODE='22023'; END IF;
 c:=vec_catalogos_configurables.validar_plan_nominal_firma_v1(canon,h);
 rev:=(c->>'revision')::bigint;
 IF c->>'id' IS DISTINCT FROM cat OR c->>'version' IS DISTINCT FROM ver::text
    OR c->>'estado' IS DISTINCT FROM (CASE op WHEN 'crear' THEN 'borrador' WHEN 'actualizar' THEN 'borrador'
      WHEN 'publicar' THEN 'publicado' ELSE 'retirado' END)
    OR c->>'version_anterior_ref' IS DISTINCT FROM (CASE WHEN ver=1 THEN NULL ELSE cat||':'||(ver-1)::text END) THEN
  RAISE EXCEPTION 'CC9: catálogo no corresponde al efecto' USING ERRCODE='22023'; END IF;
 h_material:=pg_catalog.encode(pg_catalog.sha256(p_material_exacto),'hex');
 h_contexto:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
  '{"ambitos":{"organizacion_ref":"'||p_organizacion_ref||'","unidad_ref":"'||p_unidad_ref||
  '"},"atributos":{"estado":"'||(c->>'estado')||'","material_sha256":"'||h_material||
  '","revision":"'||rev::text||'"}}','UTF8')),'hex');
 -- El comprobador AD177 no confía en este envelope: relee la decisión,
 -- consumo y auditoría originales, comprueba xmin/ventana y devuelve actor.
 SELECT vec_autorizacion_atestada_v3.comprobar_consumo_gobierno_plan_firma_v1(p_consumo->'consumo') INTO STRICT v3;
 actor:=v3->>'registrador_principal_ref';
 IF pg_catalog.jsonb_typeof(v3) IS DISTINCT FROM 'object'
    OR v3->>'decision_ref' IS DISTINCT FROM p_consumo#>>'{consumo,decision_ref}'
    OR v3->>'efecto_ref' IS DISTINCT FROM cat||':'||ver::text
    OR v3->>'huella_efecto_sha256' IS DISTINCT FROM h_contexto
    OR v3->>'consumo_huella_sha256' IS DISTINCT FROM p_consumo#>>'{consumo,consumo_huella_sha256}'
    OR v3->>'auditoria_ref' IS DISTINCT FROM p_consumo#>>'{consumo,auditoria_ref}'
    OR actor IS NULL OR actor IS DISTINCT FROM p_consumo->>'actor_ref'
    OR v3->>'registrador_perfil_ref' IS DISTINCT FROM p_consumo->>'perfil_ref'
    OR v3->>'operacion' IS DISTINCT FROM 'vec.catalogos.'||op
    OR v3->>'operacion' IS DISTINCT FROM p_consumo->>'accion'
    OR v3->>'finalidad' IS DISTINCT FROM 'gestionar_contratacion_temporal'
    OR v3->>'finalidad' IS DISTINCT FROM p_consumo->>'finalidad'
    OR v3->>'proceso' IS NULL OR v3->>'canal' IS NULL
    OR v3->>'proceso' IS DISTINCT FROM p_consumo->>'proceso'
    OR v3->>'canal' IS DISTINCT FROM p_consumo->>'canal'
    OR (v3->>'decision_valida_hasta' IS NULL OR (v3->>'decision_valida_hasta')::timestamptz<=pg_catalog.clock_timestamp()) THEN
  RAISE EXCEPTION 'CC9: decisión o consumo no ligado' USING ERRCODE='42501'; END IF;
 accion_evento:=CASE op WHEN 'crear' THEN 'vec.catalogos.borrador.creado'
  WHEN 'actualizar' THEN 'vec.catalogos.borrador.actualizado'
  WHEN 'publicar' THEN 'vec.catalogos.publicado' ELSE 'vec.catalogos.retirado' END;
 IF pg_catalog.jsonb_typeof(traza) IS DISTINCT FROM 'object' OR pg_catalog.jsonb_typeof(evento) IS DISTINCT FROM 'object'
    OR traza->>'actor_id' IS DISTINCT FROM actor OR evento->>'actor_id' IS DISTINCT FROM actor
    OR traza->>'action' IS DISTINCT FROM accion_evento OR evento->>'type' IS DISTINCT FROM accion_evento
    OR traza->>'module_id' IS DISTINCT FROM 'contratacion_temporal'
    OR evento->>'module_id' IS DISTINCT FROM 'contratacion_temporal'
    OR traza->>'subject_ref' IS DISTINCT FROM cat||':'||ver::text
    OR evento->>'subject_ref' IS DISTINCT FROM cat||':'||ver::text
    OR traza->>'after_hash' IS DISTINCT FROM h
    OR evento#>>'{payload,huella_sha256}' IS DISTINCT FROM h
    OR traza->>'result' IS DISTINCT FROM 'correcto'
    OR coalesce(traza->>'before_hash','') IS DISTINCT FROM coalesce(h_esperada,'') THEN
  RAISE EXCEPTION 'CC9: traza o evento no ligado' USING ERRCODE='22023'; END IF;
 IF (op='crear' AND (rev<>1 OR rev_esperada<>0 OR c->>'creado_por' IS DISTINCT FROM actor))
    OR (op='actualizar' AND (rev<>rev_esperada+1 OR rev_esperada<1 OR c->>'ultima_modificacion_por' IS DISTINCT FROM actor))
    OR (op IN ('publicar','retirar') AND (rev<>rev_esperada OR rev_esperada<1))
    OR (op='publicar' AND c->>'publicado_por' IS DISTINCT FROM actor)
    OR (op='retirar' AND c->>'retirado_por' IS DISTINCT FROM actor) THEN
  RAISE EXCEPTION 'CC9: transición o actor inválido' USING ERRCODE='22023'; END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_catalogos_configurables:plan_firma:'||cat,0));
 SELECT * INTO previo FROM vec_catalogos_configurables.plan_firma_efecto x
  WHERE x.actor_ref=actor AND x.catalogo_id=cat AND x.clave_operacion=clave;
 IF FOUND THEN
  IF previo.material_sha256 IS DISTINCT FROM h_material OR previo.operacion IS DISTINCT FROM op
     OR previo.version IS DISTINCT FROM ver OR previo.huella_resultado IS DISTINCT FROM h THEN
   RAISE EXCEPTION 'CC9: clave reutilizada con otro material' USING ERRCODE='23505'; END IF;
  SELECT x.recibo_ref INTO outbox_original FROM vec_catalogos_configurables.plan_firma_outbox x
   WHERE x.recibo_ref=previo.recibo_ref AND x.evento_sha256=m->>'evento_sha256';
  IF NOT FOUND THEN
   RAISE EXCEPTION 'CC9: recibo histórico sin outbox original' USING ERRCODE='55000'; END IF;
  IF (v3->>'decision_valida_hasta' IS NULL OR (v3->>'decision_valida_hasta')::timestamptz<=pg_catalog.clock_timestamp()) THEN
   RAISE EXCEPTION 'CC9: decisión de replay caducada' USING ERRCODE='42501'; END IF;
  RETURN QUERY SELECT previo.recibo_ref,previo.estado,previo.revision,previo.huella_resultado,
   previo.publicacion_sha256_resultado,previo.actor_ref,previo.confirmado_en,previo.auditoria_ref,outbox_original;
  RETURN;
 END IF;
 SELECT * INTO actual FROM vec_catalogos_configurables.plan_firma_control x
  WHERE x.catalogo_id=cat AND x.version=ver FOR UPDATE;
 existe:=FOUND;
 IF op='crear' THEN
  IF existe OR (ver>1 AND NOT EXISTS(SELECT 1 FROM vec_catalogos_configurables.plan_firma_control x
       WHERE x.catalogo_id=cat AND x.version=ver-1 AND x.estado IN ('publicado','retirado'))) THEN
   RAISE EXCEPTION 'CC9: versión de borrador en conflicto' USING ERRCODE='40001'; END IF;
  INSERT INTO vec_catalogos_configurables.plan_firma_control
   (catalogo_id,version,revision,estado,canonico_actual,huella_actual,creado_por,ultimo_editor)
  VALUES(cat,ver,rev,'borrador',canon,h,actor,actor);
 ELSE
  IF NOT existe OR actual.revision IS DISTINCT FROM rev_esperada
     OR actual.huella_actual IS DISTINCT FROM h_esperada
     OR (op IN ('actualizar','publicar') AND actual.estado IS DISTINCT FROM 'borrador')
     OR (op='retirar' AND actual.estado IS DISTINCT FROM 'publicado')
     OR c->>'creado_por' IS DISTINCT FROM actual.creado_por THEN
   RAISE EXCEPTION 'CC9: CAS de plan fallido' USING ERRCODE='40001'; END IF;
  origen:=pg_catalog.convert_from(actual.canonico_actual,'UTF8')::jsonb;
  IF op='actualizar' THEN
   IF c->>'publicado_por' IS NOT NULL OR c->>'retirado_por' IS NOT NULL
      OR c->>'creado_en' IS DISTINCT FROM origen->>'creado_en' THEN
    RAISE EXCEPTION 'CC9: edición fuera de borrador' USING ERRCODE='42501'; END IF;
  ELSIF op='publicar' THEN
   -- Un único plan publicado por módulo: no se publica junto a otro catálogo.
   PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_catalogos_configurables:plan_firma:modulo:contratacion_temporal',0));
   IF EXISTS(SELECT 1 FROM vec_catalogos_configurables.plan_firma_control x
      WHERE x.modulo_id='contratacion_temporal' AND x.estado='publicado' AND x.catalogo_id<>cat) THEN
    RAISE EXCEPTION 'CC9: ya hay otro plan publicado en el módulo' USING ERRCODE='42501'; END IF;
   -- Separación de funciones: nadie que haya creado o editado esta versión publica.
   IF actor IN (actual.creado_por,actual.ultimo_editor)
      OR EXISTS(SELECT 1 FROM vec_catalogos_configurables.plan_firma_historia x
        WHERE x.catalogo_id=cat AND x.version=ver AND x.operacion IN ('crear','actualizar') AND x.actor_ref=actor)
      OR c->>'aprobacion_ref' IS NULL
      OR c - 'estado' - 'publicado_por' - 'publicado_en' - 'aprobacion_ref' - 'motivo_publicacion'
         IS DISTINCT FROM origen - 'estado' - 'publicado_en' THEN
    RAISE EXCEPTION 'CC9: publicación sin separación o borrador idéntico' USING ERRCODE='42501'; END IF;
  ELSIF op='retirar' THEN
   IF actor=actual.publicado_por OR c->>'aprobacion_ref' IS DISTINCT FROM origen->>'aprobacion_ref'
      OR c->>'retirada_aprobacion_ref' IS NULL
      OR c->>'retirada_aprobacion_ref' IS NOT DISTINCT FROM origen->>'aprobacion_ref'
      OR c - 'estado' - 'retirado_por' - 'retirado_en' - 'retirada_aprobacion_ref' - 'motivo_retirada'
         IS DISTINCT FROM origen - 'estado' - 'retirado_en' THEN
    RAISE EXCEPTION 'CC9: retirada sin separación o publicación alterada' USING ERRCODE='42501'; END IF;
  END IF;
  UPDATE vec_catalogos_configurables.plan_firma_control SET
   revision=rev,estado=c->>'estado',canonico_actual=canon,huella_actual=h,
   ultimo_editor=CASE WHEN op='actualizar' THEN actor ELSE actual.ultimo_editor END,
   publicado_por=CASE WHEN op='publicar' THEN actor ELSE actual.publicado_por END,
   retirado_por=CASE WHEN op='retirar' THEN actor ELSE actual.retirado_por END,
   publicacion_sha256=CASE WHEN op='publicar' THEN h ELSE actual.publicacion_sha256 END,
   publicacion_revision=CASE WHEN op='publicar' THEN rev ELSE actual.publicacion_revision END
  WHERE catalogo_id=cat AND version=ver;
 END IF;
 recibo:='recibo:'||pg_catalog.gen_random_uuid()::text;
 INSERT INTO vec_catalogos_configurables.plan_firma_historia
  (catalogo_id,version,revision,operacion,estado,canonico_exacto,huella_sha256,actor_ref,decision_ref,recibo_ref)
 VALUES(cat,ver,rev,op,c->>'estado',canon,h,actor,v3->>'decision_ref',recibo);
 IF op='publicar' THEN
  momento:=(c->>'publicado_en')::timestamptz;
  INSERT INTO vec_catalogos_configurables.plan_firma_publicacion
   (catalogo_id,version,revision,canonico_exacto,publicacion_sha256,publicado_por,publicada_en,aprobacion_ref)
  VALUES(cat,ver,rev,canon,h,actor,momento,c->>'aprobacion_ref');
 END IF;
 SELECT x.publicacion_sha256 INTO h_publicacion FROM vec_catalogos_configurables.plan_firma_control x
  WHERE x.catalogo_id=cat AND x.version=ver;
 INSERT INTO vec_catalogos_configurables.plan_firma_efecto
  (actor_ref,catalogo_id,clave_operacion,operacion,version,revision,estado,huella_resultado,publicacion_sha256_resultado,
   material_sha256,recibo_ref,decision_ref,auditoria_ref,consumo_huella_sha256,traza_sha256,evento_sha256)
 VALUES(actor,cat,clave,op,ver,rev,c->>'estado',h,h_publicacion,h_material,recibo,v3->>'decision_ref',
  v3->>'auditoria_ref',v3->>'consumo_huella_sha256',m->>'traza_sha256',m->>'evento_sha256');
 INSERT INTO vec_catalogos_configurables.plan_firma_outbox(recibo_ref,evento_exacto,evento_sha256)
 VALUES(recibo,evento_bytes,m->>'evento_sha256');
 SELECT x.actor_ref,x.confirmado_en,x.auditoria_ref,o.recibo_ref
  INTO actor_original,fecha_original,auditoria_original,outbox_original
  FROM vec_catalogos_configurables.plan_firma_efecto x
  JOIN vec_catalogos_configurables.plan_firma_outbox o ON o.recibo_ref=x.recibo_ref
  WHERE x.recibo_ref=recibo AND x.actor_ref=actor AND x.material_sha256=h_material
    AND o.evento_sha256=m->>'evento_sha256';
 IF NOT FOUND OR actor_original IS NULL OR fecha_original IS NULL OR auditoria_original IS NULL OR outbox_original IS NULL THEN
  RAISE EXCEPTION 'CC9: efecto confirmado sin outbox original' USING ERRCODE='55000'; END IF;
 IF (v3->>'decision_valida_hasta' IS NULL OR (v3->>'decision_valida_hasta')::timestamptz<=pg_catalog.clock_timestamp()) THEN
  RAISE EXCEPTION 'CC9: decisión caducada antes del efecto' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT recibo,c->>'estado',rev,h,h_publicacion,
  actor_original,fecha_original,auditoria_original,outbox_original;
END $f$;
REVOKE ALL ON FUNCTION vec_catalogos_configurables.confirmar_gobierno_plan_nominal_firma_v2(bytea,text,text,jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_catalogos_configurables.confirmar_gobierno_plan_nominal_firma_v2(bytea,text,text,jsonb)
 TO vec_autorizacion_atestada_v3_propietario;
DO $acl$
DECLARE f regprocedure:='vec_catalogos_configurables.confirmar_gobierno_plan_nominal_firma_v2(bytea,text,text,jsonb)'::regprocedure; x record;
BEGIN
 FOR x IN SELECT DISTINCT a.grantee FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
  WHERE p.oid=f AND a.grantee NOT IN(p.proowner,'vec_autorizacion_atestada_v3_propietario'::pg_catalog.regrole) LOOP
  EXECUTE pg_catalog.format('REVOKE ALL ON FUNCTION %s FROM %s',f::text,
   CASE WHEN x.grantee=0 THEN 'PUBLIC' ELSE pg_catalog.quote_ident(pg_catalog.pg_get_userbyid(x.grantee)) END);
 END LOOP;
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_proc p WHERE p.oid=f AND p.prosecdef AND p.proowner='vec_catalogos_configurables_propietario'::pg_catalog.regrole)
  OR (SELECT count(*) FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a WHERE p.oid=f)<>2
  OR EXISTS(SELECT 1 FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a WHERE p.oid=f
   AND (a.grantee NOT IN(p.proowner,'vec_autorizacion_atestada_v3_propietario'::pg_catalog.regrole) OR a.privilege_type<>'EXECUTE' OR a.is_grantable OR a.grantor<>p.proowner)) THEN
  RAISE EXCEPTION 'CC9: PARO clave=ACL actual=incompatible esperado=propietario_y_AD' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
