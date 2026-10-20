\set ON_ERROR_STOP on
-- F1-128: ámbito durable de petición de centro; se activa con el consumidor Go del mismo corte.
-- El segundo argumento procede del recurso construido por el servidor y queda ligado a V3 por su huella.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog, pg_temp;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '2s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000252',0));

-- Preimagen del clon instalado: se comprueba antes del primer ALTER.
DO $ct252_pre$
DECLARE v_firma text; v_sha text; v_real text; v_acl aclitem[];
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR to_regclass('vec_contratacion_temporal.peticion_centro_revision') IS NULL
    OR EXISTS (SELECT 1 FROM pg_attribute WHERE attrelid='vec_contratacion_temporal.peticion_centro_revision'::regclass
               AND attname='organizacion_ref' AND NOT attisdropped) THEN
    RAISE EXCEPTION 'CT252: versión o tabla divergente' USING ERRCODE='55000'; END IF;
 IF to_regprocedure($ct252newsig$vec_contratacion_temporal.consultar_peticiones_centro_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)$ct252newsig$) IS NOT NULL
    OR to_regprocedure($ct252newsig$vec_contratacion_temporal.registrar_peticion_centro_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)$ct252newsig$) IS NOT NULL
    OR to_regprocedure($ct252newsig$vec_contratacion_temporal.gestionar_entrega_peticion_centro_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)$ct252newsig$) IS NOT NULL
    OR to_regprocedure($ct252newsig$vec_contratacion_temporal.preparar_entrega_peticion_centro_con_original_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)$ct252newsig$) IS NOT NULL
    OR to_regprocedure('vec_contratacion_temporal.ambitos_entrega_peticion_centro_v1(text,text)') IS NOT NULL THEN
    RAISE EXCEPTION 'CT252: firma nueva ya instalada' USING ERRCODE='55000'; END IF;
 FOR v_firma,v_sha IN SELECT * FROM (VALUES

  ($ct252preid$vec_contratacion_temporal.consultar_peticiones_centro_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)$ct252preid$,$ct252presha$3dfcdca066ef9a62ee3c04c11556cf2714957c176ad1c7df9e4b7b76972e12bd$ct252presha$),
  ($ct252preid$vec_contratacion_temporal.registrar_peticion_centro_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)$ct252preid$,$ct252presha$8425a15b4c97a6eab6627dde07104ecf30f5244e30136dd159d84156ec50c388$ct252presha$),
  ($ct252preid$vec_contratacion_temporal.gestionar_entrega_peticion_centro_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)$ct252preid$,$ct252presha$7a9a46e76e5f52b89393feb8ececc8d762426b41c0149a05ddd17ab018f53424$ct252presha$),
  ($ct252preid$vec_contratacion_temporal.preparar_entrega_peticion_centro_con_original_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)$ct252preid$,$ct252presha$bfa758988b4a3b3cd808c029aa1d0ea0a04c6cc5a4e4baab1cb56af9a42182ce$ct252presha$),
  ('vec_contratacion_temporal.ambitos_entrega_peticion_centro_v1(text)','d12cb76787bf6f5013551947af8f23d3fff06164b3fff18088fe5b5c07b897a5')
 ) AS esperadas(firma,sha) LOOP
  SELECT pg_get_functiondef(p.oid),p.proacl INTO v_real,v_acl FROM pg_proc p
  WHERE p.oid=to_regprocedure(v_firma) AND p.proowner='vec_contratacion_temporal_propietario'::regrole AND p.prosecdef;
  IF v_real IS NULL OR v_acl IS NULL OR encode(sha256(convert_to(v_real,'UTF8')),'hex')<>v_sha THEN
    RAISE EXCEPTION 'CT252: preimagen divergente: %',v_firma USING ERRCODE='55000'; END IF;
  IF v_firma='vec_contratacion_temporal.ambitos_entrega_peticion_centro_v1(text)'
     AND v_acl IS DISTINCT FROM ARRAY[
       'vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario',
       'vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario']::aclitem[] THEN
    RAISE EXCEPTION 'CT252: ACL de proyección instalada divergente' USING ERRCODE='55000';
  END IF;
 END LOOP;
END $ct252_pre$;

-- La historia anterior conserva explícitamente su ámbito original. No se altera material, canon ni recibo.
LOCK TABLE vec_contratacion_temporal.peticion_centro_revision IN ACCESS EXCLUSIVE MODE;
ALTER TABLE vec_contratacion_temporal.peticion_centro_revision ADD COLUMN organizacion_ref text NOT NULL DEFAULT 'organizacion:desarrollo:dipgra';
ALTER TABLE vec_contratacion_temporal.peticion_centro_revision ADD CONSTRAINT peticion_centro_revision_organizacion_ref_forma CHECK (organizacion_ref='organizacion:desarrollo:dipgra' OR organizacion_ref ~ '^org_[a-z0-9]{16,80}$');
-- Reconstrucción nominada de consultar_peticiones_centro_v1 desde la definición instalada.
DO $ct252_consultar_peticiones_centro_v1$
DECLARE v_def text; v_acl aclitem[]; v_owner oid; v_patch record;
BEGIN
 SELECT pg_get_functiondef(p.oid),p.proacl,p.proowner INTO STRICT v_def,v_acl,v_owner
 FROM pg_proc p WHERE p.oid=$ct252sig$vec_contratacion_temporal.consultar_peticiones_centro_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)$ct252sig$::regprocedure
 AND p.proowner='vec_contratacion_temporal_propietario'::regrole AND p.prosecdef;
 IF encode(sha256(convert_to(v_def,'UTF8')),'hex')<>$ct252sha$3dfcdca066ef9a62ee3c04c11556cf2714957c176ad1c7df9e4b7b76972e12bd$ct252sha$ OR v_acl IS NULL THEN RAISE EXCEPTION 'CT252: preimagen/ACL divergente: consultar_peticiones_centro_v1' USING ERRCODE='55000'; END IF;
 FOR v_patch IN SELECT * FROM (VALUES
  ($ct252label$firma_organizacion$ct252label$,$ct252old$p_consulta text, p_capacidad bytea$ct252old$,$ct252new$p_consulta text, p_organizacion_ref text, p_capacidad bytea$ct252new$,1),
  ($ct252label$search_path$ct252label$,$ct252old$SET search_path TO 'pg_catalog'$ct252old$,$ct252new$SET search_path TO 'pg_catalog', 'pg_temp'$ct252new$,1),
  ($ct252label$validar_forma_organizacion$ct252label$,$ct252old$    IF p_consulta IS NULL OR octet_length(p_consulta) NOT BETWEEN 1 AND 65536 THEN$ct252old$,$ct252new$    IF p_organizacion_ref IS NULL OR (p_organizacion_ref <> 'organizacion:desarrollo:dipgra' AND p_organizacion_ref !~ '^org_[a-z0-9]{16,80}$') THEN
        RAISE EXCEPTION 'organización de petición inválida' USING ERRCODE='P0673';
    END IF;
    IF p_consulta IS NULL OR octet_length(p_consulta) NOT BETWEEN 1 AND 65536 THEN$ct252new$,1),
  ($ct252label$hash_consulta$ct252label$,$ct252old$||',"organizacion_ref":"organizacion:desarrollo:dipgra"},"atributos":{"material_sha256":"'||$ct252old$,$ct252new$||',"organizacion_ref":'||to_jsonb(p_organizacion_ref)::text||'},"atributos":{"material_sha256":"'||$ct252new$,1),
  ($ct252label$operacion_ambito$ct252label$,$ct252old$WHERE clave_idempotencia=v_ref::uuid;$ct252old$,$ct252new$WHERE clave_idempotencia=v_ref::uuid AND organizacion_ref=p_organizacion_ref;$ct252new$,1),
  ($ct252label$peticion_ambito$ct252label$,$ct252old$WHERE peticion_ref=v_ref ORDER BY version DESC LIMIT 1;$ct252old$,$ct252new$WHERE peticion_ref=v_ref AND organizacion_ref=p_organizacion_ref ORDER BY version DESC LIMIT 1;$ct252new$,1),
  ($ct252label$bandeja_ambito$ct252label$,$ct252old$WHERE a=r.peticion->'configuracion'->'solicitante' OR a=r.peticion->'configuracion'->'ratificador'$ct252old$,$ct252new$WHERE r.organizacion_ref=p_organizacion_ref AND (a=r.peticion->'configuracion'->'solicitante' OR a=r.peticion->'configuracion'->'ratificador')$ct252new$,1)
 ) AS marcas(nombre,antes,despues,apariciones) LOOP
  IF (length(v_def)-length(replace(v_def,v_patch.antes,'')))<>length(v_patch.antes)*v_patch.apariciones THEN
    RAISE EXCEPTION 'CT252: marca divergente %.%', $ct252n$consultar_peticiones_centro_v1$ct252n$,v_patch.nombre USING ERRCODE='55000';
  END IF;
  v_def:=replace(v_def,v_patch.antes,v_patch.despues);
 END LOOP;
 IF (length(v_def)-length(replace(v_def,'organizacion:desarrollo:dipgra','')))/length('organizacion:desarrollo:dipgra')<>1 OR position('SET search_path TO ''pg_catalog'', ''pg_temp''' in v_def)=0 THEN
    RAISE EXCEPTION 'CT252: ámbito o search_path incompleto: consultar_peticiones_centro_v1' USING ERRCODE='55000'; END IF;
 EXECUTE v_def;
 EXECUTE $ct252revoke$REVOKE ALL ON FUNCTION vec_contratacion_temporal.consultar_peticiones_centro_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC,vec_contratacion_temporal_ejecutor,vec_contratacion_temporal_migrador$ct252revoke$;
 EXECUTE $ct252grant$GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.consultar_peticiones_centro_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_contratacion_temporal_ejecutor$ct252grant$;
 IF (SELECT proowner FROM pg_proc WHERE oid=$ct252newid$vec_contratacion_temporal.consultar_peticiones_centro_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)$ct252newid$::regprocedure) IS DISTINCT FROM v_owner
    OR (SELECT proacl FROM pg_proc WHERE oid=$ct252newid2$vec_contratacion_temporal.consultar_peticiones_centro_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)$ct252newid2$::regprocedure) IS DISTINCT FROM v_acl
    OR NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=$ct252newid3$vec_contratacion_temporal.consultar_peticiones_centro_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)$ct252newid3$::regprocedure
      AND p.prosecdef AND p.proconfig @> ARRAY['search_path=pg_catalog, pg_temp']::text[]) THEN
    RAISE EXCEPTION 'CT252: owner/ACL nueva divergente: consultar_peticiones_centro_v1' USING ERRCODE='55000'; END IF;
END
$ct252_consultar_peticiones_centro_v1$;

-- Reconstrucción nominada de registrar_peticion_centro_v1 desde la definición instalada.
DO $ct252_registrar_peticion_centro_v1$
DECLARE v_def text; v_acl aclitem[]; v_owner oid; v_patch record;
BEGIN
 SELECT pg_get_functiondef(p.oid),p.proacl,p.proowner INTO STRICT v_def,v_acl,v_owner
 FROM pg_proc p WHERE p.oid=$ct252sig$vec_contratacion_temporal.registrar_peticion_centro_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)$ct252sig$::regprocedure
 AND p.proowner='vec_contratacion_temporal_propietario'::regrole AND p.prosecdef;
 IF encode(sha256(convert_to(v_def,'UTF8')),'hex')<>$ct252sha$8425a15b4c97a6eab6627dde07104ecf30f5244e30136dd159d84156ec50c388$ct252sha$ OR v_acl IS NULL THEN RAISE EXCEPTION 'CT252: preimagen/ACL divergente: registrar_peticion_centro_v1' USING ERRCODE='55000'; END IF;
 FOR v_patch IN SELECT * FROM (VALUES
  ($ct252label$firma_organizacion$ct252label$,$ct252old$p_material text, p_capacidad bytea$ct252old$,$ct252new$p_material text, p_organizacion_ref text, p_capacidad bytea$ct252new$,1),
  ($ct252label$validar_forma_organizacion$ct252label$,$ct252old$    IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 65536 THEN$ct252old$,$ct252new$    IF p_organizacion_ref IS NULL OR (p_organizacion_ref <> 'organizacion:desarrollo:dipgra' AND p_organizacion_ref !~ '^org_[a-z0-9]{16,80}$') THEN
        RAISE EXCEPTION 'organización de petición inválida' USING ERRCODE='P0673';
    END IF;
    IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 65536 THEN$ct252new$,1),
  ($ct252label$hash_escritura$ct252label$,$ct252old$||',"organizacion_ref":"organizacion:desarrollo:dipgra"},"atributos":{"material_sha256":"'||$ct252old$,$ct252new$||',"organizacion_ref":'||to_jsonb(p_organizacion_ref)::text||'},"atributos":{"material_sha256":"'||$ct252new$,1),
  ($ct252label$preconsulta_replay$ct252label$,$ct252old$WHERE clave_idempotencia=v_clave) THEN$ct252old$,$ct252new$WHERE clave_idempotencia=v_clave AND organizacion_ref=p_organizacion_ref) THEN$ct252new$,1),
  ($ct252label$replay_ambito$ct252label$,$ct252old$IF v_previa.actor_ref IS DISTINCT FROM$ct252old$,$ct252new$IF v_previa.organizacion_ref IS DISTINCT FROM p_organizacion_ref OR v_previa.actor_ref IS DISTINCT FROM$ct252new$,1),
  ($ct252label$replay_denegacion_opaca$ct252label$,$ct252old$RAISE EXCEPTION 'operación de petición ajena' USING ERRCODE='P0673';$ct252old$,$ct252new$RAISE EXCEPTION 'petición no visible' USING ERRCODE='P0673';$ct252new$,1),
  ($ct252label$ratificacion_ambito$ct252label$,$ct252old$WHERE peticion_ref=v_ref ORDER BY version DESC LIMIT 1 FOR UPDATE;$ct252old$,$ct252new$WHERE peticion_ref=v_ref AND organizacion_ref=p_organizacion_ref ORDER BY version DESC LIMIT 1 FOR UPDATE;$ct252new$,1),
  ($ct252label$insert_columna$ct252label$,$ct252old$peticion_ref,version,clave_idempotencia,operacion,material,material_sha256,actor_ref,perfil_ref,centro_ref,puesto_ref,$ct252old$,$ct252new$peticion_ref,version,clave_idempotencia,operacion,material,material_sha256,organizacion_ref,actor_ref,perfil_ref,centro_ref,puesto_ref,$ct252new$,1),
  ($ct252label$insert_valor$ct252label$,$ct252old$v_ref,v_version,v_clave,v_operacion,p_material,v_material_huella,a->>$ct252old$,$ct252new$v_ref,v_version,v_clave,v_operacion,p_material,v_material_huella,p_organizacion_ref,a->>$ct252new$,1)
 ) AS marcas(nombre,antes,despues,apariciones) LOOP
  IF (length(v_def)-length(replace(v_def,v_patch.antes,'')))<>length(v_patch.antes)*v_patch.apariciones THEN
    RAISE EXCEPTION 'CT252: marca divergente %.%', $ct252n$registrar_peticion_centro_v1$ct252n$,v_patch.nombre USING ERRCODE='55000';
  END IF;
  v_def:=replace(v_def,v_patch.antes,v_patch.despues);
 END LOOP;
 IF (length(v_def)-length(replace(v_def,'organizacion:desarrollo:dipgra','')))/length('organizacion:desarrollo:dipgra')<>1 OR position('SET search_path TO ''pg_catalog'', ''pg_temp''' in v_def)=0 THEN
    RAISE EXCEPTION 'CT252: ámbito o search_path incompleto: registrar_peticion_centro_v1' USING ERRCODE='55000'; END IF;
 EXECUTE v_def;
 EXECUTE $ct252revoke$REVOKE ALL ON FUNCTION vec_contratacion_temporal.registrar_peticion_centro_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC,vec_contratacion_temporal_ejecutor,vec_contratacion_temporal_migrador$ct252revoke$;
 EXECUTE $ct252grant$GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.registrar_peticion_centro_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_contratacion_temporal_ejecutor$ct252grant$;
 IF (SELECT proowner FROM pg_proc WHERE oid=$ct252newid$vec_contratacion_temporal.registrar_peticion_centro_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)$ct252newid$::regprocedure) IS DISTINCT FROM v_owner
    OR (SELECT proacl FROM pg_proc WHERE oid=$ct252newid2$vec_contratacion_temporal.registrar_peticion_centro_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)$ct252newid2$::regprocedure) IS DISTINCT FROM v_acl
    OR NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=$ct252newid3$vec_contratacion_temporal.registrar_peticion_centro_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)$ct252newid3$::regprocedure
      AND p.prosecdef AND p.proconfig @> ARRAY['search_path=pg_catalog, pg_temp']::text[]) THEN
    RAISE EXCEPTION 'CT252: owner/ACL nueva divergente: registrar_peticion_centro_v1' USING ERRCODE='55000'; END IF;
END
$ct252_registrar_peticion_centro_v1$;

-- Reconstrucción nominada de gestionar_entrega_peticion_centro_v1 desde la definición instalada.
DO $ct252_gestionar_entrega_peticion_centro_v1$
DECLARE v_def text; v_acl aclitem[]; v_owner oid; v_patch record;
BEGIN
 SELECT pg_get_functiondef(p.oid),p.proacl,p.proowner INTO STRICT v_def,v_acl,v_owner
 FROM pg_proc p WHERE p.oid=$ct252sig$vec_contratacion_temporal.gestionar_entrega_peticion_centro_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)$ct252sig$::regprocedure
 AND p.proowner='vec_contratacion_temporal_propietario'::regrole AND p.prosecdef;
 IF encode(sha256(convert_to(v_def,'UTF8')),'hex')<>$ct252sha$7a9a46e76e5f52b89393feb8ececc8d762426b41c0149a05ddd17ab018f53424$ct252sha$ OR v_acl IS NULL THEN RAISE EXCEPTION 'CT252: preimagen/ACL divergente: gestionar_entrega_peticion_centro_v1' USING ERRCODE='55000'; END IF;
 FOR v_patch IN SELECT * FROM (VALUES
  ($ct252label$firma_organizacion$ct252label$,$ct252old$p_material text, p_capacidad bytea$ct252old$,$ct252new$p_material text, p_organizacion_ref text, p_capacidad bytea$ct252new$,1),
  ($ct252label$validar_forma_organizacion$ct252label$,$ct252old$    IF session_user=current_user$ct252old$,$ct252new$    IF p_organizacion_ref IS NULL OR (p_organizacion_ref <> 'organizacion:desarrollo:dipgra' AND p_organizacion_ref !~ '^org_[a-z0-9]{16,80}$') THEN
        RAISE EXCEPTION 'organización de petición inválida' USING ERRCODE='P0683';
    END IF;
    IF session_user=current_user$ct252new$,1),
  ($ct252label$ratificada_ambito$ct252label$,$ct252old$WHERE peticion_ref=v_peticion_ref AND version=2 AND operacion='ratificar' AND estado='ratificada';$ct252old$,$ct252new$WHERE peticion_ref=v_peticion_ref AND version=2 AND operacion='ratificar' AND estado='ratificada' AND organizacion_ref=p_organizacion_ref;$ct252new$,2),
  ($ct252label$hash_bandeja$ct252label$,$ct252old$v_contexto:='{"ambitos":{"organizacion_ref":"organizacion:desarrollo:dipgra"},"atributos":{"material_sha256":"'||v_material_sha256||'"}}';$ct252old$,$ct252new$v_contexto:='{"ambitos":{"organizacion_ref":'||vec_contratacion_temporal.texto_json_go_v1(p_organizacion_ref)||'},"atributos":{"material_sha256":"'||v_material_sha256||'"}}';$ct252new$,1),
  ($ct252label$hash_entrega$ct252label$,',"organizacion_ref":"organizacion:desarrollo:dipgra"},"atributos":{"material_sha256":"''||v_material_sha256||''"}}'';',',"organizacion_ref":''||vec_contratacion_temporal.texto_json_go_v1(p_organizacion_ref)||''},"atributos":{"material_sha256":"''||v_material_sha256||''"}}'';',1),
  ($ct252label$bandeja_ambito$ct252label$,$ct252old$WHERE revision.version=2 AND revision.operacion='ratificar' AND revision.estado='ratificada'$ct252old$,$ct252new$WHERE revision.version=2 AND revision.operacion='ratificar' AND revision.estado='ratificada'
                AND revision.organizacion_ref=p_organizacion_ref$ct252new$,1),
  ($ct252label$identidad_alta_ambito$ct252label$,$ct252old$AND identidad.organizacion_ref='organizacion:desarrollo:dipgra'$ct252old$,$ct252new$AND identidad.organizacion_ref=v_peticion.organizacion_ref$ct252new$,2),
  ($ct252label$reserva_replay_identidad$ct252label$,$ct252old$    SELECT * INTO v_reserva FROM vec_contratacion_temporal.entrega_peticion_centro_reserva
     WHERE peticion_ref=v_peticion_ref;$ct252old$,$ct252new$    SELECT * INTO v_reserva FROM vec_contratacion_temporal.entrega_peticion_centro_reserva
     WHERE peticion_ref=v_peticion_ref;
    -- Cualquier alias ya materializado debe pertenecer a la organización
    -- de la revisión. También se coteja antes de devolver un replay confirmado.
    IF v_reserva.peticion_ref IS NOT NULL THEN
        IF EXISTS (
            SELECT 1 FROM vec_contratacion_temporal.alias_ambito_alta alias
            LEFT JOIN vec_contratacion_temporal.identidad_reserva_alta identidad
              ON identidad.ambito_hmac=alias.ambito_raiz_hmac
            WHERE alias.alias_hmac=v_reserva.ambito_alta_hmac
              AND identidad.organizacion_ref IS DISTINCT FROM v_peticion.organizacion_ref
        ) OR (
            EXISTS (SELECT 1 FROM vec_contratacion_temporal.entrega_peticion_centro_confirmacion c
                    WHERE c.peticion_ref=v_peticion_ref)
            AND NOT EXISTS (
                SELECT 1 FROM vec_contratacion_temporal.entrega_peticion_centro_confirmacion c
                JOIN vec_contratacion_temporal.alias_ambito_alta alias
                  ON alias.alias_hmac=v_reserva.ambito_alta_hmac
                JOIN vec_contratacion_temporal.identidad_reserva_alta identidad
                  ON identidad.ambito_hmac=alias.ambito_raiz_hmac
                JOIN vec_contratacion_temporal.expediente_alta alta
                  ON alta.expediente_ref=identidad.expediente_ref
                WHERE c.peticion_ref=v_peticion_ref
                  AND c.ambito_alta_hmac=v_reserva.ambito_alta_hmac
                  AND c.expediente_ref=alta.expediente_ref
                  AND identidad.organizacion_ref=v_peticion.organizacion_ref
                  AND alta.organizacion_ref=v_peticion.organizacion_ref
            )
        ) THEN RAISE EXCEPTION 'entrega de otra organización' USING ERRCODE='P0681'; END IF;
    END IF;$ct252new$,1),
  ($ct252label$canon_alta_ambito$ct252label$,$ct252old$OR v_alta->>'organizacion_ref' IS DISTINCT FROM 'organizacion:desarrollo:dipgra'$ct252old$,$ct252new$OR v_alta->>'organizacion_ref' IS DISTINCT FROM v_peticion.organizacion_ref$ct252new$,1)
 ) AS marcas(nombre,antes,despues,apariciones) LOOP
  IF (length(v_def)-length(replace(v_def,v_patch.antes,'')))<>length(v_patch.antes)*v_patch.apariciones THEN
    RAISE EXCEPTION 'CT252: marca divergente %.%', $ct252n$gestionar_entrega_peticion_centro_v1$ct252n$,v_patch.nombre USING ERRCODE='55000';
  END IF;
  v_def:=replace(v_def,v_patch.antes,v_patch.despues);
 END LOOP;
 IF (length(v_def)-length(replace(v_def,'organizacion:desarrollo:dipgra','')))/length('organizacion:desarrollo:dipgra')<>1 OR position('SET search_path TO ''pg_catalog'', ''pg_temp''' in v_def)=0 THEN
    RAISE EXCEPTION 'CT252: ámbito o search_path incompleto: gestionar_entrega_peticion_centro_v1' USING ERRCODE='55000'; END IF;
 EXECUTE v_def;
 EXECUTE $ct252revoke$REVOKE ALL ON FUNCTION vec_contratacion_temporal.gestionar_entrega_peticion_centro_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC,vec_contratacion_temporal_ejecutor,vec_contratacion_temporal_migrador$ct252revoke$;
 EXECUTE $ct252grant$GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.gestionar_entrega_peticion_centro_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_contratacion_temporal_ejecutor$ct252grant$;
 IF (SELECT proowner FROM pg_proc WHERE oid=$ct252newid$vec_contratacion_temporal.gestionar_entrega_peticion_centro_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)$ct252newid$::regprocedure) IS DISTINCT FROM v_owner
    OR (SELECT proacl FROM pg_proc WHERE oid=$ct252newid2$vec_contratacion_temporal.gestionar_entrega_peticion_centro_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)$ct252newid2$::regprocedure) IS DISTINCT FROM v_acl
    OR NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=$ct252newid3$vec_contratacion_temporal.gestionar_entrega_peticion_centro_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)$ct252newid3$::regprocedure
      AND p.prosecdef AND p.proconfig @> ARRAY['search_path=pg_catalog, pg_temp']::text[]) THEN
    RAISE EXCEPTION 'CT252: owner/ACL nueva divergente: gestionar_entrega_peticion_centro_v1' USING ERRCODE='55000'; END IF;
END
$ct252_gestionar_entrega_peticion_centro_v1$;

-- Reconstrucción nominada de preparar_entrega_peticion_centro_con_original_v1 desde la definición instalada.
DO $ct252_preparar_entrega_peticion_centro_con_original_v1$
DECLARE v_def text; v_acl aclitem[]; v_owner oid; v_patch record;
BEGIN
 SELECT pg_get_functiondef(p.oid),p.proacl,p.proowner INTO STRICT v_def,v_acl,v_owner
 FROM pg_proc p WHERE p.oid=$ct252sig$vec_contratacion_temporal.preparar_entrega_peticion_centro_con_original_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)$ct252sig$::regprocedure
 AND p.proowner='vec_contratacion_temporal_propietario'::regrole AND p.prosecdef;
 IF encode(sha256(convert_to(v_def,'UTF8')),'hex')<>$ct252sha$bfa758988b4a3b3cd808c029aa1d0ea0a04c6cc5a4e4baab1cb56af9a42182ce$ct252sha$ OR v_acl IS NULL THEN RAISE EXCEPTION 'CT252: preimagen/ACL divergente: preparar_entrega_peticion_centro_con_original_v1' USING ERRCODE='55000'; END IF;
 FOR v_patch IN SELECT * FROM (VALUES
  ($ct252label$firma_organizacion$ct252label$,$ct252old$p_material text, p_capacidad bytea$ct252old$,$ct252new$p_material text, p_organizacion_ref text, p_capacidad bytea$ct252new$,1),
  ($ct252label$search_path$ct252label$,$ct252old$SET search_path TO 'pg_catalog'$ct252old$,$ct252new$SET search_path TO 'pg_catalog', 'pg_temp'$ct252new$,1),
  ($ct252label$validar_forma_organizacion$ct252label$,$ct252old$ IF current_user<>'vec_contratacion_temporal_propietario'$ct252old$,$ct252new$ IF p_organizacion_ref IS NULL OR (p_organizacion_ref <> 'organizacion:desarrollo:dipgra' AND p_organizacion_ref !~ '^org_[a-z0-9]{16,80}$') THEN
     RAISE EXCEPTION 'organización de petición inválida' USING ERRCODE='P0683';
 END IF;
 IF current_user<>'vec_contratacion_temporal_propietario'$ct252new$,1),
  ($ct252label$llamar_gestion_ambito$ct252label$,$ct252old$    p_material,p_capacidad,p_decision,p_motivo,p_contexto,$ct252old$,$ct252new$    p_material,p_organizacion_ref,p_capacidad,p_decision,p_motivo,p_contexto,$ct252new$,1),
  ($ct252label$revision_reserva$ct252label$,$ct252old$        v.alta_canonica,v.confirmacion_ref AS version_confirmacion,$ct252old$,$ct252new$        revision.organizacion_ref AS peticion_organizacion,
        v.alta_canonica,v.confirmacion_ref AS version_confirmacion,$ct252new$,1),
  ($ct252label$join_revision$ct252label$,$ct252old$   FROM vec_contratacion_temporal.entrega_peticion_centro_reserva s
$ct252old$,$ct252new$   FROM vec_contratacion_temporal.entrega_peticion_centro_reserva s
   JOIN vec_contratacion_temporal.peticion_centro_revision revision
      ON revision.peticion_ref=s.peticion_ref AND revision.version=2
$ct252new$,1),
  ($ct252label$ambito_sin_alta$ct252label$,$ct252old$ IF r.confirmacion_ref IS NULL THEN$ct252old$,$ct252new$ IF r.peticion_organizacion IS DISTINCT FROM p_organizacion_ref THEN
     RAISE EXCEPTION 'revisión de otra organización' USING ERRCODE='P0683';
 END IF;
 IF r.confirmacion_ref IS NULL THEN$ct252new$,1),
  ($ct252label$identidad_preparada_ambito$ct252label$,$ct252old$ IF r.confirmacion_ref IS NULL THEN$ct252old$,$ct252new$ IF r.organizacion_ref IS NOT NULL
    AND r.organizacion_ref IS DISTINCT FROM r.peticion_organizacion THEN
    RAISE EXCEPTION 'identidad original de otra organización' USING ERRCODE='P0683';
 END IF;
 IF r.confirmacion_ref IS NULL THEN$ct252new$,1),
  ($ct252label$identidad_ambito$ct252label$,$ct252old$IF r.organizacion_ref IS DISTINCT FROM 'organizacion:desarrollo:dipgra'$ct252old$,$ct252new$IF r.organizacion_ref IS DISTINCT FROM r.peticion_organizacion$ct252new$,1)
 ) AS marcas(nombre,antes,despues,apariciones) LOOP
  IF (length(v_def)-length(replace(v_def,v_patch.antes,'')))<>length(v_patch.antes)*v_patch.apariciones THEN
    RAISE EXCEPTION 'CT252: marca divergente %.%', $ct252n$preparar_entrega_peticion_centro_con_original_v1$ct252n$,v_patch.nombre USING ERRCODE='55000';
  END IF;
  v_def:=replace(v_def,v_patch.antes,v_patch.despues);
 END LOOP;
 IF (length(v_def)-length(replace(v_def,'organizacion:desarrollo:dipgra','')))/length('organizacion:desarrollo:dipgra')<>1 OR position('SET search_path TO ''pg_catalog'', ''pg_temp''' in v_def)=0 THEN
    RAISE EXCEPTION 'CT252: ámbito o search_path incompleto: preparar_entrega_peticion_centro_con_original_v1' USING ERRCODE='55000'; END IF;
 EXECUTE v_def;
 EXECUTE $ct252revoke$REVOKE ALL ON FUNCTION vec_contratacion_temporal.preparar_entrega_peticion_centro_con_original_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC,vec_contratacion_temporal_ejecutor,vec_contratacion_temporal_migrador$ct252revoke$;
 EXECUTE $ct252grant$GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.preparar_entrega_peticion_centro_con_original_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_contratacion_temporal_ejecutor$ct252grant$;
 IF (SELECT proowner FROM pg_proc WHERE oid=$ct252newid$vec_contratacion_temporal.preparar_entrega_peticion_centro_con_original_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)$ct252newid$::regprocedure) IS DISTINCT FROM v_owner
    OR (SELECT proacl FROM pg_proc WHERE oid=$ct252newid2$vec_contratacion_temporal.preparar_entrega_peticion_centro_con_original_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)$ct252newid2$::regprocedure) IS DISTINCT FROM v_acl
    OR NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=$ct252newid3$vec_contratacion_temporal.preparar_entrega_peticion_centro_con_original_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)$ct252newid3$::regprocedure
      AND p.prosecdef AND p.proconfig @> ARRAY['search_path=pg_catalog, pg_temp']::text[]) THEN
    RAISE EXCEPTION 'CT252: owner/ACL nueva divergente: preparar_entrega_peticion_centro_con_original_v1' USING ERRCODE='55000'; END IF;
END
$ct252_preparar_entrega_peticion_centro_con_original_v1$;

-- La proyección anterior a V3 usa el mismo ámbito que el recurso de entrega.
DO $ct252_ambitos_entrega$
DECLARE v_def text; v_acl aclitem[]; v_owner oid; v_patch record;
BEGIN
 SELECT pg_get_functiondef(p.oid),p.proacl,p.proowner INTO STRICT v_def,v_acl,v_owner
 FROM pg_proc p
 WHERE p.oid='vec_contratacion_temporal.ambitos_entrega_peticion_centro_v1(text)'::regprocedure
   AND p.proowner='vec_contratacion_temporal_propietario'::regrole
   AND p.prosecdef AND p.provolatile='s';
 IF encode(sha256(convert_to(v_def,'UTF8')),'hex')<>'d12cb76787bf6f5013551947af8f23d3fff06164b3fff18088fe5b5c07b897a5'
    OR v_acl IS DISTINCT FROM ARRAY[
      'vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario',
      'vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario']::aclitem[] THEN
   RAISE EXCEPTION 'CT252: preimagen/ACL de proyección divergente' USING ERRCODE='55000';
 END IF;
 FOR v_patch IN SELECT * FROM (VALUES
  ('firma',
   $ct252old$p_peticion_ref text)$ct252old$,
   $ct252new$p_peticion_ref text, p_organizacion_ref text)$ct252new$),
  ('search_path',
   $ct252old$SET search_path TO 'pg_catalog'$ct252old$,
   $ct252new$SET search_path TO 'pg_catalog', 'pg_temp'$ct252new$),
  ('forma_ambito',
   $ct252old$       OR p_peticion_ref IS NULL$ct252old$,
   $ct252new$       OR p_organizacion_ref IS NULL
       OR (p_organizacion_ref <> 'organizacion:desarrollo:dipgra'
           AND p_organizacion_ref !~ '^org_[a-z0-9]{16,80}$')
       OR p_peticion_ref IS NULL$ct252new$),
  ('revision_ratificada',
   $ct252old$WHERE r.peticion_ref=p_peticion_ref AND r.version=2$ct252old$,
   $ct252new$WHERE r.peticion_ref=p_peticion_ref AND r.organizacion_ref=p_organizacion_ref AND r.version=2$ct252new$)
 ) AS marcas(nombre,antes,despues) LOOP
   IF length(v_def)-length(replace(v_def,v_patch.antes,''))<>length(v_patch.antes) THEN
     RAISE EXCEPTION 'CT252: marca divergente en proyección: %',v_patch.nombre USING ERRCODE='55000';
   END IF;
   v_def:=replace(v_def,v_patch.antes,v_patch.despues);
 END LOOP;
 IF position('SET search_path TO ''pg_catalog'', ''pg_temp''' in v_def)=0 THEN
   RAISE EXCEPTION 'CT252: search_path de proyección divergente' USING ERRCODE='55000';
 END IF;
 EXECUTE v_def;
 REVOKE ALL ON FUNCTION vec_contratacion_temporal.ambitos_entrega_peticion_centro_v1(text,text)
   FROM PUBLIC,vec_contratacion_temporal_ejecutor,vec_contratacion_temporal_migrador;
 GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.ambitos_entrega_peticion_centro_v1(text,text)
   TO vec_contratacion_temporal_ejecutor;
 IF NOT EXISTS (SELECT 1 FROM pg_proc p
   WHERE p.oid='vec_contratacion_temporal.ambitos_entrega_peticion_centro_v1(text,text)'::regprocedure
     AND p.proowner=v_owner AND p.proacl=v_acl AND p.prosecdef AND p.provolatile='s'
     AND p.proconfig @> ARRAY['search_path=pg_catalog, pg_temp']::text[]) THEN
   RAISE EXCEPTION 'CT252: contrato de proyección divergente' USING ERRCODE='55000';
 END IF;
END
$ct252_ambitos_entrega$;

DROP FUNCTION vec_contratacion_temporal.ambitos_entrega_peticion_centro_v1(text);
DROP FUNCTION vec_contratacion_temporal.preparar_entrega_peticion_centro_con_original_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_contratacion_temporal.gestionar_entrega_peticion_centro_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_contratacion_temporal.registrar_peticion_centro_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_contratacion_temporal.consultar_peticiones_centro_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
COMMIT;
