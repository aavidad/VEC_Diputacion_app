\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contacto_usuario_v1:dependencias:v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000035',0));
LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_autorizacion_atestada_v3.atestacion_decision_v3 IN ACCESS EXCLUSIVE MODE;
DO $historia$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_proc WHERE pronamespace=to_regnamespace('vec_contacto_usuario_v1')) THEN
        RAISE EXCEPTION 'contacto: retirar fachadas del almacén antes de DOWN' USING ERRCODE='55000';
    END IF;
    IF to_regprocedure('vec_bolsa_registro_accesos.registrar_contacto_usuario_v1(bytea,bytea,bytea,bytea,bytea,text,text,text)') IS NOT NULL
       OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.clave_capacidad_version
            WHERE audiencia_consumo IN ('vec.contacto_usuario.registro.v1','vec.contacto_usuario.consulta.v1'))
       OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.atestacion_decision_v3
            WHERE convert_from(capacidad_canonica,'UTF8')::jsonb->>'operacion' IN
                ('vec.contacto_usuario.alta','vec.contacto_usuario.actualizar','vec.contacto_usuario.consultar')) THEN
        RAISE EXCEPTION 'AD3-35: dependencias o historia de contacto; no admite DOWN' USING ERRCODE='55000';
    END IF;
END $historia$;
DO $restaurar$
DECLARE f oid:='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
    def text; nueva text; inicio integer; original integer; fin integer; metadata jsonb;
    extension text:=$contactoperfiles$           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'contacto_usuario_alta'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM 'vec.contacto_usuario.registro.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM 'vec.contacto_usuario.alta'
               AND d ->> 'accion' IS NOT DISTINCT FROM 'vec.contacto_usuario.alta'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'vec.module.usuarios'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'contacto_usuario'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'gestion_contacto_propio'
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'contacto_usuario_actualizar'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM 'vec.contacto_usuario.registro.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM 'vec.contacto_usuario.actualizar'
               AND d ->> 'accion' IS NOT DISTINCT FROM 'vec.contacto_usuario.actualizar'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'vec.module.usuarios'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'contacto_usuario'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'gestion_contacto_propio'
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'contacto_usuario_consultar'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM 'vec.contacto_usuario.consulta.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM 'vec.contacto_usuario.consultar'
               AND d ->> 'accion' IS NOT DISTINCT FROM 'vec.contacto_usuario.consultar'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'vec.module.usuarios'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'contacto_usuario'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'envio_llamamiento'
           )
$contactoperfiles$;
BEGIN
    SELECT pg_get_functiondef(p.oid),to_jsonb(p)-'prosrc' INTO STRICT def,metadata FROM pg_proc p WHERE p.oid=f;
    inicio:=strpos(def,'/* AD3-35 GUARDA INICIO */');
    original:=strpos(def,'/* AD3-35 GUARDA ORIGINAL */');
    fin:=strpos(def,') END) /* AD3-35 GUARDA FIN */');
    IF inicio=0 OR original<=inicio OR fin<=original
       OR length(def)-length(replace(def,extension,''))<>length(extension) THEN
        RAISE EXCEPTION 'AD3-35: postimagen de núcleo divergente' USING ERRCODE='55000';
    END IF;
    nueva:=overlay(def placing substring(def FROM original+length('/* AD3-35 GUARDA ORIGINAL */')
        FOR fin-original-length('/* AD3-35 GUARDA ORIGINAL */'))
        FROM inicio FOR fin-inicio+length(') END) /* AD3-35 GUARDA FIN */'));
    nueva:=replace(nueva,extension,'');
    EXECUTE nueva;
    IF pg_get_functiondef(f) IS DISTINCT FROM nueva OR
       (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM metadata THEN
        RAISE EXCEPTION 'AD3-35: restauración divergente' USING ERRCODE='55000';
    END IF;
END $restaurar$;
DO $audiencias$
DECLARE def text; nueva text; marca text:=', ''vec.contacto_usuario.registro.v1''::text, ''vec.contacto_usuario.consulta.v1''::text';
BEGIN
    SELECT pg_get_constraintdef(oid,true) INTO STRICT def FROM pg_constraint
        WHERE conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
        AND conname='clave_capacidad_version_audiencia_consumo_check' AND contype='c' AND convalidated;
    IF length(def)-length(replace(def,marca,''))<>length(marca) THEN
        RAISE EXCEPTION 'AD3-35: postimagen de audiencias divergente' USING ERRCODE='55000';
    END IF;
    nueva:=replace(def,marca,'');
    ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
    EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||nueva;
END $audiencias$;
DROP FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_alta_contacto_usuario_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea) RESTRICT;
DROP FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_actualizar_contacto_usuario_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea) RESTRICT;
DROP FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_consultar_contacto_usuario_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea) RESTRICT;
DO $revalidacion$
DECLARE f oid:='vec_autorizacion_atestada_v3.revalidar_consumo_consulta_rrhh_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
    def text; nueva text; metadata jsonb; inicio integer; original integer; fin integer;
BEGIN
    SELECT pg_get_functiondef(p.oid),to_jsonb(p)-'prosrc' INTO STRICT def,metadata FROM pg_proc p
        WHERE p.oid=f AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
        AND encode(sha256(convert_to(prosrc,'UTF8')),'hex')='7b93dd88a453fc052344825762a8ea0d46a8072fe6e058db09186c3bd50bde80';
    inicio:=strpos(def,'/* AD3-35 RV INICIO */'); original:=strpos(def,'/* AD3-35 RV ORIGINAL */');
    fin:=strpos(def,') END) /* AD3-35 RV FIN */');
    IF inicio=0 OR original<=inicio OR fin<=original THEN
        RAISE EXCEPTION 'AD3-35: revalidación posterior incompatible' USING ERRCODE='55000';
    END IF;
    nueva:=overlay(def placing substring(def FROM original+length('/* AD3-35 RV ORIGINAL */')
        FOR fin-original-length('/* AD3-35 RV ORIGINAL */')) FROM inicio FOR fin-inicio+length(') END) /* AD3-35 RV FIN */'));
    nueva:=replace(nueva,$antes$p_perfil_consulta NOT IN ('cuadro', 'detalle', 'contacto_usuario', 'contacto_usuario_alta', 'contacto_usuario_actualizar')$antes$,$despues$p_perfil_consulta NOT IN ('cuadro', 'detalle')$despues$);
    nueva:=replace(nueva,$antes$IF p_perfil_consulta = 'contacto_usuario_alta' THEN
        v_audiencia := 'vec.contacto_usuario.registro.v1';
        v_operacion := 'vec.contacto_usuario.alta';
        v_tipo_recurso := 'contacto_usuario';
        v_finalidad := 'gestion_contacto_propio';
    ELSIF p_perfil_consulta = 'contacto_usuario_actualizar' THEN
        v_audiencia := 'vec.contacto_usuario.registro.v1';
        v_operacion := 'vec.contacto_usuario.actualizar';
        v_tipo_recurso := 'contacto_usuario';
        v_finalidad := 'gestion_contacto_propio';
    ELSIF p_perfil_consulta = 'contacto_usuario' THEN
        v_audiencia := 'vec.contacto_usuario.consulta.v1';
        v_operacion := 'vec.contacto_usuario.consultar';
        v_tipo_recurso := 'contacto_usuario';
        v_finalidad := 'envio_llamamiento';
    ELSIF p_perfil_consulta = 'cuadro' THEN$antes$,$despues$IF p_perfil_consulta = 'cuadro' THEN$despues$);
    nueva:=replace(nueva,$antes$d ->> 'modulo_id' IS DISTINCT FROM (CASE WHEN p_perfil_consulta IN ('contacto_usuario','contacto_usuario_alta','contacto_usuario_actualizar')
           THEN 'vec.module.usuarios' ELSE 'contratacion_temporal' END)$antes$,$despues$d ->> 'modulo_id' <> 'contratacion_temporal'$despues$);
    EXECUTE nueva;
    IF (SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') FROM pg_proc WHERE oid=f) IS DISTINCT FROM '7cfc002cff8878fc36288fa1200de1c51965e9ae4d84b4ffe6179bc62372b5ff'
       OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM metadata THEN
        RAISE EXCEPTION 'AD3-35: restauración de revalidación divergente' USING ERRCODE='55000';
    END IF;
END $revalidacion$;
DROP FUNCTION vec_autorizacion_atestada_v3.revalidar_consulta_contacto_usuario_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea) RESTRICT;
DROP FUNCTION vec_autorizacion_atestada_v3.revalidar_alta_contacto_usuario_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea) RESTRICT;
DROP FUNCTION vec_autorizacion_atestada_v3.revalidar_actualizar_contacto_usuario_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea) RESTRICT;
DROP FUNCTION vec_autorizacion_atestada_v3.contacto_material_auditoria_v1(text,bytea,bytea,bytea,bytea,bytea) RESTRICT;
DROP FUNCTION vec_autorizacion_atestada_v3.contacto_validar_material_v1(text,bytea,bytea,bytea,bytea,bytea) RESTRICT;
DROP FUNCTION vec_autorizacion_atestada_v3.contacto_auditoria_previa_v1(bytea) RESTRICT;
DROP FUNCTION vec_autorizacion_atestada_v3.contacto_mapa_canonico_v1(jsonb) RESTRICT;
DROP FUNCTION vec_autorizacion_atestada_v3.contacto_objeto_v1(jsonb,jsonb) RESTRICT;
DROP FUNCTION vec_autorizacion_atestada_v3.contacto_sesion_nominal_v1(text) RESTRICT;
-- USAGE conservado: no se retiran concesiones compartidas posteriores.
COMMIT;
