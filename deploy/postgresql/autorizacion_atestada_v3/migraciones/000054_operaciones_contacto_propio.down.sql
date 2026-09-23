\set ON_ERROR_STOP on
-- Reversión únicamente de ensayo sin historia. Orden: Contacto3 DOWN,
-- T13/8 DOWN y luego AD3-54 DOWN. No retirar permisos con recibos conservados.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contacto_usuario_v1:dependencias:v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000054',0));
LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_autorizacion_atestada_v3.atestacion_decision_v3 IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_autorizacion_atestada_v3.consumo_decision_v3 IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 IN ACCESS EXCLUSIVE MODE;
DO $historia$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
    OR to_regclass('vec_contacto_usuario_v1.operaciones') IS NOT NULL
    OR to_regprocedure('vec_bolsa_registro_accesos.registrar_operacion_contacto_v1(text,bytea,bytea,bytea,bytea,bytea,text,text,text,text,text)') IS NOT NULL
    OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.clave_capacidad_version
        WHERE audiencia_consumo IN ('vec.contacto_usuario.operacion.preparar.v1','vec.contacto_usuario.operacion.cancelar.v1',
            'vec.contacto_usuario.operacion.listar.v1','vec.contacto_usuario.operacion.detalle.v1'))
    OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.atestacion_decision_v3
        WHERE convert_from(capacidad_canonica,'UTF8')::jsonb->>'audiencia_consumo' IN
            ('vec.contacto_usuario.operacion.preparar.v1','vec.contacto_usuario.operacion.cancelar.v1',
             'vec.contacto_usuario.operacion.listar.v1','vec.contacto_usuario.operacion.detalle.v1'))
    OR EXISTS(SELECT 1 FROM vec_bolsa_registro_accesos.registro_acceso
        WHERE action IN ('vec.contacto_usuario.operacion.preparar','vec.contacto_usuario.operacion.cancelar',
                         'vec.contacto_usuario.operacion.listar','vec.contacto_usuario.operacion.detalle')) THEN
    RAISE EXCEPTION 'AD3-54: DOWN exige DBA, dependencias retiradas y cero historia' USING ERRCODE='55000';
 END IF;
END $historia$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;

DROP FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_operacion_contacto_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea) RESTRICT;
DROP FUNCTION vec_autorizacion_atestada_v3.revalidar_operacion_contacto_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea) RESTRICT;
DROP FUNCTION vec_autorizacion_atestada_v3.contacto_operacion_material_auditoria_v1(text,bytea,bytea,bytea,bytea,bytea) RESTRICT;
DROP FUNCTION vec_autorizacion_atestada_v3.contacto_operacion_validar_material_v1(text,bytea,bytea,bytea,bytea,bytea) RESTRICT;
DROP FUNCTION vec_autorizacion_atestada_v3.contacto_operacion_auditoria_previa_v1(bytea) RESTRICT;

DO $audiencias$
DECLARE def text; valores text[]; canon text; finales text[]:=ARRAY[
   'vec.contacto_usuario.operacion.preparar.v1','vec.contacto_usuario.operacion.cancelar.v1',
   'vec.contacto_usuario.operacion.listar.v1','vec.contacto_usuario.operacion.detalle.v1'];
BEGIN
 SELECT regexp_replace(pg_get_constraintdef(oid,true),'\s+',' ','g') INTO STRICT def FROM pg_constraint
   WHERE conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
     AND conname='clave_capacidad_version_audiencia_consumo_check' AND contype='c' AND convalidated AND conkey=ARRAY[8]::smallint[];
 SELECT array_agg(m[1] ORDER BY n) INTO valores FROM regexp_matches(def,'''([a-zA-Z0-9_.-]+)''::text','g') WITH ORDINALITY x(m,n);
 canon:='CHECK (audiencia_consumo = ANY (ARRAY['||array_to_string(ARRAY(
   SELECT quote_literal(v)||'::text' FROM unnest(valores) WITH ORDINALITY x(v,n) ORDER BY n),', ')||']))';
 IF def IS DISTINCT FROM canon OR cardinality(valores) NOT BETWEEN 19 AND 72
    OR valores[cardinality(valores)-3:cardinality(valores)] IS DISTINCT FROM finales THEN
    RAISE EXCEPTION 'AD3-54: audiencias posteriores incompatibles' USING ERRCODE='55000';
 END IF;
 valores:=valores[1:cardinality(valores)-4];
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check CHECK (audiencia_consumo IN ('||
   array_to_string(ARRAY(SELECT quote_literal(v) FROM unnest(valores) WITH ORDINALITY x(v,n) ORDER BY n),', ')||'))';
END $audiencias$;

DO $operacion_nucleo_down$
DECLARE f oid:='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
    def text; nueva text; metadata jsonb; deps jsonb; r record;
BEGIN
    SELECT pg_get_functiondef(p.oid),to_jsonb(p)-'prosrc' INTO STRICT def,metadata FROM pg_proc p
      WHERE p.oid=f AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
        AND p.prosecdef AND p.provolatile='v' AND p.pronargdefaults=0
        AND p.proconfig=ARRAY['search_path=pg_catalog','lock_timeout=2s']
        AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='3434a876cc67ee9331d1f5a9340790495c3ce531994ba8ef235d4a455d627942';
    SELECT jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype)
      INTO deps FROM pg_depend d WHERE (d.classid='pg_proc'::regclass AND d.objid=f)
         OR (d.refclassid='pg_proc'::regclass AND d.refobjid=f);
    nueva:=def;
    FOR r IN SELECT * FROM (VALUES
      ($antes0$p_perfil_mutacion IN ('contacto_usuario_alta','contacto_usuario_actualizar','contacto_usuario_consultar','contacto_usuario_recibo','contacto_usuario_version_propia','contacto_usuario_version_llamamiento')$antes0$,
       $despues0$p_perfil_mutacion IN ('contacto_usuario_alta','contacto_usuario_actualizar','contacto_usuario_consultar','contacto_usuario_recibo','contacto_usuario_version_propia','contacto_usuario_version_llamamiento','contacto_operacion_preparar','contacto_operacion_cancelar','contacto_operacion_listar','contacto_operacion_detalle')$despues0$,1),
      ($antes1$contacto_sesion_nominal_v1(CASE WHEN p_perfil_mutacion IN ('contacto_usuario_recibo','contacto_usuario_version_propia') THEN 'contacto_usuario_alta' WHEN p_perfil_mutacion='contacto_usuario_version_llamamiento' THEN 'contacto_usuario_consultar' ELSE p_perfil_mutacion END)$antes1$,
       $despues1$contacto_sesion_nominal_v1(CASE WHEN p_perfil_mutacion IN ('contacto_usuario_recibo','contacto_usuario_version_propia','contacto_operacion_preparar','contacto_operacion_cancelar','contacto_operacion_listar','contacto_operacion_detalle') THEN 'contacto_usuario_alta' WHEN p_perfil_mutacion='contacto_usuario_version_llamamiento' THEN 'contacto_usuario_consultar' ELSE p_perfil_mutacion END)$despues1$,1),
      ($antes2$           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'contacto_usuario_version_llamamiento'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM 'vec.contacto_usuario.version_llamamiento.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM 'vec.contacto_usuario.version_para_llamamiento'
               AND d ->> 'accion' IS NOT DISTINCT FROM 'vec.contacto_usuario.version_para_llamamiento'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'vec.module.usuarios'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'contacto_usuario'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'envio_llamamiento'
           )$antes2$,
       $despues2$           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'contacto_usuario_version_llamamiento'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM 'vec.contacto_usuario.version_llamamiento.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM 'vec.contacto_usuario.version_para_llamamiento'
               AND d ->> 'accion' IS NOT DISTINCT FROM 'vec.contacto_usuario.version_para_llamamiento'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'vec.module.usuarios'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'contacto_usuario'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'envio_llamamiento'
           )
           OR (
               p_perfil_mutacion IN ('contacto_operacion_preparar','contacto_operacion_cancelar',
                                     'contacto_operacion_listar','contacto_operacion_detalle')
               AND p_perfil_mutacion IS NOT DISTINCT FROM CASE d ->> 'accion'
                   WHEN 'vec.contacto_usuario.operacion.preparar' THEN 'contacto_operacion_preparar'
                   WHEN 'vec.contacto_usuario.operacion.cancelar' THEN 'contacto_operacion_cancelar'
                   WHEN 'vec.contacto_usuario.operacion.listar' THEN 'contacto_operacion_listar'
                   WHEN 'vec.contacto_usuario.operacion.detalle' THEN 'contacto_operacion_detalle' END
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM CASE d ->> 'accion'
                   WHEN 'vec.contacto_usuario.operacion.preparar' THEN 'vec.contacto_usuario.operacion.preparar.v1'
                   WHEN 'vec.contacto_usuario.operacion.cancelar' THEN 'vec.contacto_usuario.operacion.cancelar.v1'
                   WHEN 'vec.contacto_usuario.operacion.listar' THEN 'vec.contacto_usuario.operacion.listar.v1'
                   WHEN 'vec.contacto_usuario.operacion.detalle' THEN 'vec.contacto_usuario.operacion.detalle.v1' END
               AND c ->> 'operacion' IS NOT DISTINCT FROM d ->> 'accion'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'vec.module.usuarios'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'contacto_usuario'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'gestion_contacto_propio'
           )$despues2$,1)
    ) AS cambios(antes,despues,veces) LOOP
        IF length(nueva)-length(replace(nueva,r.despues,''))<>length(r.despues)*r.veces THEN
            RAISE EXCEPTION 'AD3-54: fragmento del núcleo no exacto' USING ERRCODE='55000';
        END IF;
        nueva:=replace(nueva,r.despues,r.antes);
    END LOOP;
    EXECUTE nueva;
    IF pg_get_functiondef(f) IS DISTINCT FROM nueva
       OR (SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') FROM pg_proc WHERE oid=f) IS DISTINCT FROM '63c14d42c5fce79d92be437bd5bb61328ab14a26e2e5392cf7063f87371e2ffe'
       OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM metadata
       OR (SELECT jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype)
             FROM pg_depend d WHERE (d.classid='pg_proc'::regclass AND d.objid=f)
                OR (d.refclassid='pg_proc'::regclass AND d.refobjid=f)) IS DISTINCT FROM deps THEN
        RAISE EXCEPTION 'AD3-54: inversión de nucleo divergente' USING ERRCODE='55000';
    END IF;
END $operacion_nucleo_down$;

DO $operacion_revalidacion_down$
DECLARE f oid:='vec_autorizacion_atestada_v3.revalidar_consumo_consulta_rrhh_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
    def text; nueva text; metadata jsonb; deps jsonb; r record;
BEGIN
    SELECT pg_get_functiondef(p.oid),to_jsonb(p)-'prosrc' INTO STRICT def,metadata FROM pg_proc p
      WHERE p.oid=f AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
        AND p.prosecdef AND p.provolatile='v' AND p.pronargdefaults=0
        AND p.proconfig=ARRAY['search_path=pg_catalog','lock_timeout=1s']
        AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='51f4b05feb19efeff9f1e985a723ed7f64e03d1b8d264691ad57131676467b14';
    SELECT jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype)
      INTO deps FROM pg_depend d WHERE (d.classid='pg_proc'::regclass AND d.objid=f)
         OR (d.refclassid='pg_proc'::regclass AND d.refobjid=f);
    nueva:=def;
    FOR r IN SELECT * FROM (VALUES
      ($antes0$p_perfil_consulta IN ('contacto_usuario','contacto_usuario_alta','contacto_usuario_actualizar','contacto_usuario_recibo','contacto_usuario_version_propia','contacto_usuario_version_llamamiento')$antes0$,
       $despues0$p_perfil_consulta IN ('contacto_usuario','contacto_usuario_alta','contacto_usuario_actualizar','contacto_usuario_recibo','contacto_usuario_version_propia','contacto_usuario_version_llamamiento','contacto_operacion_preparar','contacto_operacion_cancelar','contacto_operacion_listar','contacto_operacion_detalle')$despues0$,2),
      ($antes1$CASE WHEN p_perfil_consulta='contacto_usuario' THEN 'contacto_usuario_consultar' WHEN p_perfil_consulta IN ('contacto_usuario_recibo','contacto_usuario_version_propia') THEN 'contacto_usuario_alta' WHEN p_perfil_consulta='contacto_usuario_version_llamamiento' THEN 'contacto_usuario_consultar' ELSE p_perfil_consulta END$antes1$,
       $despues1$CASE WHEN p_perfil_consulta='contacto_usuario' THEN 'contacto_usuario_consultar' WHEN p_perfil_consulta IN ('contacto_usuario_recibo','contacto_usuario_version_propia','contacto_operacion_preparar','contacto_operacion_cancelar','contacto_operacion_listar','contacto_operacion_detalle') THEN 'contacto_usuario_alta' WHEN p_perfil_consulta='contacto_usuario_version_llamamiento' THEN 'contacto_usuario_consultar' ELSE p_perfil_consulta END$despues1$,1),
      ($antes2$p_perfil_consulta NOT IN ('cuadro', 'detalle', 'contacto_usuario', 'contacto_usuario_alta', 'contacto_usuario_actualizar', 'contacto_usuario_recibo', 'contacto_usuario_version_propia', 'contacto_usuario_version_llamamiento')$antes2$,
       $despues2$p_perfil_consulta NOT IN ('cuadro', 'detalle', 'contacto_usuario', 'contacto_usuario_alta', 'contacto_usuario_actualizar', 'contacto_usuario_recibo', 'contacto_usuario_version_propia', 'contacto_usuario_version_llamamiento', 'contacto_operacion_preparar', 'contacto_operacion_cancelar', 'contacto_operacion_listar', 'contacto_operacion_detalle')$despues2$,1),
      ($antes3$IF p_perfil_consulta = 'contacto_usuario_version_propia' THEN$antes3$,
       $despues3$IF p_perfil_consulta IN ('contacto_operacion_preparar','contacto_operacion_cancelar',
                                'contacto_operacion_listar','contacto_operacion_detalle') THEN
        v_audiencia := CASE p_perfil_consulta
          WHEN 'contacto_operacion_preparar' THEN 'vec.contacto_usuario.operacion.preparar.v1'
          WHEN 'contacto_operacion_cancelar' THEN 'vec.contacto_usuario.operacion.cancelar.v1'
          WHEN 'contacto_operacion_listar' THEN 'vec.contacto_usuario.operacion.listar.v1'
          WHEN 'contacto_operacion_detalle' THEN 'vec.contacto_usuario.operacion.detalle.v1' END;
        v_operacion := CASE p_perfil_consulta
          WHEN 'contacto_operacion_preparar' THEN 'vec.contacto_usuario.operacion.preparar'
          WHEN 'contacto_operacion_cancelar' THEN 'vec.contacto_usuario.operacion.cancelar'
          WHEN 'contacto_operacion_listar' THEN 'vec.contacto_usuario.operacion.listar'
          WHEN 'contacto_operacion_detalle' THEN 'vec.contacto_usuario.operacion.detalle' END;
        v_tipo_recurso := 'contacto_usuario';
        v_finalidad := 'gestion_contacto_propio';
    ELSIF p_perfil_consulta = 'contacto_usuario_version_propia' THEN$despues3$,1)
    ) AS cambios(antes,despues,veces) LOOP
        IF length(nueva)-length(replace(nueva,r.despues,''))<>length(r.despues)*r.veces THEN
            RAISE EXCEPTION 'AD3-54: fragmento de revalidación no exacto' USING ERRCODE='55000';
        END IF;
        nueva:=replace(nueva,r.despues,r.antes);
    END LOOP;
    EXECUTE nueva;
    IF pg_get_functiondef(f) IS DISTINCT FROM nueva
       OR (SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') FROM pg_proc WHERE oid=f) IS DISTINCT FROM '58d7d00d858132f8e08cc470c716ca04fc0521268834482e293894f7be3d806b'
       OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM metadata
       OR (SELECT jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype)
             FROM pg_depend d WHERE (d.classid='pg_proc'::regclass AND d.objid=f)
                OR (d.refclassid='pg_proc'::regclass AND d.refobjid=f)) IS DISTINCT FROM deps THEN
        RAISE EXCEPTION 'AD3-54: inversión de revalidacion divergente' USING ERRCODE='55000';
    END IF;
END $operacion_revalidacion_down$;

-- Inversión textual con hashes candidatos: faltan cotejo PG18 y E10.
DO $incompleta$ BEGIN RAISE EXCEPTION 'AD3-54 DOWN WIP: postimagen PG18 pendiente' USING ERRCODE='55000'; END $incompleta$;
COMMIT;
