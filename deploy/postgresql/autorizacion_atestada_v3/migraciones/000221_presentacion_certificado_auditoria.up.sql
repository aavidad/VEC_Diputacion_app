-- AD221. Acto técnico nominal de presentación en la cadena común AD207.
-- Instalar después de AD219/AUT58 y antes de IS18. Fase pre-F1 sin perfil
-- activo, decisión V3, firma documental ni efecto de autorización por perfil.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_autorizacion_atestada_v3:migracion:000221', 0));
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_autorizacion_atestada_v3:nucleo', 0));
DO $pre$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname=current_user AND rolsuper)
       OR pg_catalog.current_setting('server_version_num')::integer
          NOT BETWEEN 180000 AND 189999
       OR pg_catalog.to_regclass('vec_autorizacion_atestada_v3.auditoria_consumo_v3') IS NULL
       OR pg_catalog.to_regclass('vec_autorizacion_atestada_v3.pendiente_sellado_auditoria_v5') IS NULL
       OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.reservar_asiento_auditoria_v5()') IS NULL
       OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.verificar_cadena_auditoria_v5(boolean)') IS NULL
       OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_catalogo_acciones_admin_v1(jsonb)') IS NULL
       OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.encuadrar_mac(text)') IS NULL
       OR pg_catalog.to_regrole('vec_identidad_sesiones_v1_propietario') IS NULL
       OR pg_catalog.to_regrole('vec_identidad_sesiones_v1_revocador') IS NULL
       OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_trigger t
           WHERE t.tgrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::pg_catalog.regclass
             AND t.tgname='encolar_sellado_ad207' AND t.tgenabled='O')
       OR pg_catalog.to_regclass('vec_autorizacion_atestada_v3.auditoria_presentacion_certificado_v1') IS NOT NULL
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_attribute a
           WHERE a.attrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::pg_catalog.regclass
             AND a.attname='presentacion_certificado_detalle' AND NOT a.attisdropped)
       OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_auditoria_presentacion_certificado_v1(text,text,text,text,text,text,text,text,text,text,text,text,text,text,text,text,text,bytea,text,text,timestamptz,text,text,text,timestamptz)') IS NOT NULL THEN
        RAISE EXCEPTION 'AD221: preimagen incompatible' USING ERRCODE='55000';
    END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
LOCK TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 IN ACCESS EXCLUSIVE MODE;
ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3
    ADD COLUMN presentacion_certificado_detalle jsonb;

-- AD219 deja un CHECK disjunto: preservar su expresion íntegra y añadir solo
-- la familia técnica nueva con todas las columnas ajenas obligatoriamente NULL.
DO $familia$
DECLARE anterior text; nuevo text; nulas text;
    propias constant text[]:=ARRAY[
        'auditoria_ref','secuencia','anterior_sha256','huella_sha256',
        'registrada_en','tipo_registro','evento_ref','evento_material_sha256',
        'operador_login','accion','modulo_id','recurso_ref','finalidad_ref',
        'resultado','motivo_ref','proceso','canal','correlacion_ref',
        'presentacion_certificado_detalle'];
BEGIN
    SELECT pg_catalog.pg_get_constraintdef(c.oid,false) INTO STRICT anterior
      FROM pg_catalog.pg_constraint c
     WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::pg_catalog.regclass
       AND c.conname='auditoria_tipo_disjunto_v4' AND c.contype='c' AND c.convalidated;
    IF pg_catalog.left(anterior,7)<>'CHECK ('
       OR pg_catalog.right(anterior,1)<>')'
       OR pg_catalog.strpos(anterior,'catalogo_acciones_detalle')=0
       OR pg_catalog.strpos(anterior,'identidad_operacion_ref')=0 THEN
        RAISE EXCEPTION 'AD221: familia anterior incompatible' USING ERRCODE='55000';
    END IF;
    SELECT pg_catalog.string_agg(pg_catalog.format('%I IS NULL',a.attname),
           ' AND ' ORDER BY a.attnum) INTO STRICT nulas
      FROM pg_catalog.pg_attribute a
     WHERE a.attrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::pg_catalog.regclass
       AND a.attnum>0 AND NOT a.attisdropped AND NOT a.attname=ANY(propias);
    IF nulas IS NULL OR nulas NOT LIKE '%decision_ref IS NULL%'
       OR nulas NOT LIKE '%actor_ref IS NULL%'
       OR nulas NOT LIKE '%catalogo_acciones_detalle IS NULL%' THEN
        RAISE EXCEPTION 'AD221: columnas ajenas no cerradas' USING ERRCODE='55000';
    END IF;
    nuevo:='CHECK ((presentacion_certificado_detalle IS NULL AND ('||
        pg_catalog.substr(anterior,8,pg_catalog.length(anterior)-8)||
        ')) OR (tipo_registro = ''presentacion_certificado_v1'' AND '||nulas||
        ' AND presentacion_certificado_detalle IS NOT NULL'||
        ' AND evento_ref IS NOT NULL AND evento_material_sha256 IS NOT NULL'||
        ' AND operador_login IS NOT NULL AND accion IS NOT NULL'||
        ' AND modulo_id IS NOT DISTINCT FROM ''identidad'''||
        ' AND recurso_ref IS NOT NULL'||
        ' AND finalidad_ref IS NOT DISTINCT FROM ''autenticacion_temporal_pre_f1'''||
        ' AND resultado IS NOT DISTINCT FROM ''permitido'''||
        ' AND motivo_ref IS NOT NULL'||
        ' AND proceso IS NOT DISTINCT FROM ''postgresql'''||
        ' AND canal IS NOT NULL AND correlacion_ref IS NOT NULL))';
    ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3
        DROP CONSTRAINT auditoria_tipo_disjunto_v4;
    EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 ADD CONSTRAINT auditoria_tipo_disjunto_v4 '||nuevo;
END $familia$;
ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3
    ADD CONSTRAINT auditoria_presentacion_certificado_formato_v1 CHECK (
        tipo_registro <> 'presentacion_certificado_v1' OR (
            evento_ref ~ '^evento_[0-9a-f]{32}$'
            AND evento_material_sha256 ~ '^[0-9a-f]{64}$'
            AND correlacion_ref ~ '^correlacion_[0-9a-f]{32}$'
            AND pg_catalog.jsonb_typeof(presentacion_certificado_detalle)='object'
            AND presentacion_certificado_detalle ?& ARRAY[
                'fase','operacion_ref','presentacion_ref',
                'autenticacion_original_ref','sesion_original_ref',
                'cuenta_ref','actor_cuenta_ref','actor_autenticacion_ref',
                'actor_sesion_ref','sujeto_id_hmac','superficie','tipo',
                'recibo_sha256','acr_original','acr_actual',
                'politica_original_ref','politica_original_sha256',
                'politica_actual_ref','politica_actual_sha256',
                'canal_sha256','asercion_actual_sha256',
                'presentacion_valida_hasta','control_sesion_ref',
                'control_sesion_revision','control_origen_operacion_ref',
                'registrada_en']
            AND presentacion_certificado_detalle->>'fase' = CASE
                WHEN presentacion_certificado_detalle->>'tipo'=
                    'revocacion_control_is2_sin_actor_humano_nominal'
                THEN 'sin_actor_humano_nominal'
                ELSE 'pre_f1_sin_perfil_activo' END
            AND ((presentacion_certificado_detalle->>'tipo'=
                    'revocacion_control_is2_sin_actor_humano_nominal'
                AND pg_catalog.jsonb_typeof(presentacion_certificado_detalle->'actor_cuenta_ref')='null'
                AND pg_catalog.jsonb_typeof(presentacion_certificado_detalle->'actor_autenticacion_ref')='null'
                AND pg_catalog.jsonb_typeof(presentacion_certificado_detalle->'actor_sesion_ref')='null'
                AND presentacion_certificado_detalle->>'control_sesion_ref' ~ '^cse_[A-Za-z0-9_-]{22,128}$'
                AND presentacion_certificado_detalle->>'control_sesion_revision' ~ '^[1-9][0-9]{0,19}$'
                AND presentacion_certificado_detalle->>'control_origen_operacion_ref' =
                    presentacion_certificado_detalle->>'operacion_ref')
                OR (presentacion_certificado_detalle->>'tipo'<>
                    'revocacion_control_is2_sin_actor_humano_nominal'
                AND presentacion_certificado_detalle->>'actor_cuenta_ref' ~
                    '^cta_[A-Za-z0-9_-]{22,128}$'))
            AND presentacion_certificado_detalle->>'cuenta_ref' ~ '^cta_[A-Za-z0-9_-]{22,128}$'
            AND presentacion_certificado_detalle->>'sujeto_id_hmac' ~ '^[0-9a-f]{64}$'
            AND presentacion_certificado_detalle->>'tipo' IN
                ('apertura','reanudacion','renovacion','revocacion',
                 'revocacion_control_is2_sin_actor_humano_nominal')
            AND presentacion_certificado_detalle->>'presentacion_ref' ~ '^prs_[A-Za-z0-9_-]{22,128}$'
            AND recurso_ref = 'presentacion_certificado:'||
                (presentacion_certificado_detalle->>'presentacion_ref')
            AND accion = CASE presentacion_certificado_detalle->>'tipo'
                WHEN 'revocacion' THEN 'revocar_vinculo_certificado_v1'
                WHEN 'revocacion_control_is2_sin_actor_humano_nominal'
                    THEN 'revocar_sesion_v1'
                ELSE 'presentar_certificado_v1' END
            AND canal = CASE presentacion_certificado_detalle->>'tipo'
                WHEN 'revocacion' THEN 'identidad_sesion_vigente'
                WHEN 'revocacion_control_is2_sin_actor_humano_nominal'
                    THEN 'control_sesion_is2'
                ELSE 'certificado_mtls_atestado_por_frontera' END
            AND motivo_ref = CASE presentacion_certificado_detalle->>'tipo'
                WHEN 'apertura' THEN 'presentacion_abierta'
                WHEN 'renovacion' THEN 'presentacion_renovada'
                WHEN 'reanudacion' THEN 'presentacion_reanudada'
                WHEN 'revocacion_control_is2_sin_actor_humano_nominal'
                    THEN 'control_sesion_revocado'
                ELSE 'vinculo_revocado' END
        ));

CREATE TABLE vec_autorizacion_atestada_v3.auditoria_presentacion_certificado_v1 (
    operacion_ref text PRIMARY KEY CHECK (operacion_ref ~ '^opr_[A-Za-z0-9_-]{22,128}$'),
    auditoria_ref text NOT NULL UNIQUE REFERENCES
        vec_autorizacion_atestada_v3.auditoria_consumo_v3(auditoria_ref),
    auditoria_secuencia numeric(20,0) NOT NULL,
    auditoria_huella_sha256 text NOT NULL CHECK (auditoria_huella_sha256 ~ '^[0-9a-f]{64}$'),
    evento_material_sha256 text NOT NULL CHECK (evento_material_sha256 ~ '^[0-9a-f]{64}$'),
    presentacion_ref text NOT NULL CHECK (presentacion_ref ~ '^prs_[A-Za-z0-9_-]{22,128}$'),
    autenticacion_original_ref text NOT NULL CHECK (autenticacion_original_ref ~ '^aut_[A-Za-z0-9_-]{22,128}$'),
    sesion_original_ref text NOT NULL CHECK (sesion_original_ref ~ '^ses_[A-Za-z0-9_-]{22,128}$'),
    cuenta_ref text NOT NULL CHECK (cuenta_ref ~ '^cta_[A-Za-z0-9_-]{22,128}$'),
    actor_cuenta_ref text CHECK (actor_cuenta_ref ~ '^cta_[A-Za-z0-9_-]{22,128}$'),
    actor_autenticacion_ref text CHECK (actor_autenticacion_ref ~ '^aut_[A-Za-z0-9_-]{22,128}$'),
    actor_sesion_ref text CHECK (actor_sesion_ref ~ '^ses_[A-Za-z0-9_-]{22,128}$'),
    superficie text NOT NULL CHECK (superficie IN (
        'externa_personal','interna_corporativa','administracion_privilegiada')),
    tipo text NOT NULL CHECK (tipo IN ('apertura','reanudacion','renovacion','revocacion',
        'revocacion_control_is2_sin_actor_humano_nominal')),
    control_sesion_ref text,
    control_sesion_revision text,
    control_origen_operacion_ref text,
    CHECK ((tipo='revocacion_control_is2_sin_actor_humano_nominal'
        AND actor_cuenta_ref IS NULL AND actor_autenticacion_ref IS NULL
        AND actor_sesion_ref IS NULL
        AND control_sesion_ref IS NOT NULL
        AND control_sesion_revision IS NOT NULL
        AND control_origen_operacion_ref IS NOT NULL
        AND control_sesion_ref ~ '^cse_[A-Za-z0-9_-]{22,128}$'
        AND control_sesion_revision ~ '^[1-9][0-9]{0,19}$'
        AND control_origen_operacion_ref ~ '^opr_[A-Za-z0-9_-]{22,128}$')
        OR (tipo<>'revocacion_control_is2_sin_actor_humano_nominal'
        AND actor_cuenta_ref IS NOT NULL AND actor_autenticacion_ref IS NOT NULL
        AND actor_sesion_ref IS NOT NULL AND control_sesion_ref IS NULL
        AND control_sesion_revision IS NULL
        AND control_origen_operacion_ref IS NULL)),
    recibo_sha256 text NOT NULL CHECK (recibo_sha256 ~ '^[0-9a-f]{64}$'
        AND recibo_sha256 <> pg_catalog.repeat('0',64)),
    acr_original text NOT NULL CHECK (
        acr_original='urn:vec:acr:certificado-desarrollo-protegido'),
    acr_actual text,
    politica_original_ref text NOT NULL CHECK (
        politica_original_ref ~ '^pga_[A-Za-z0-9_-]{22,128}$'),
    politica_original_sha256 text NOT NULL CHECK (
        politica_original_sha256 ~ '^[0-9a-f]{64}$'
        AND politica_original_sha256 <> pg_catalog.repeat('0',64)),
    politica_actual_ref text,
    politica_actual_sha256 text,
    CHECK ((tipo IN ('revocacion','revocacion_control_is2_sin_actor_humano_nominal')
        AND acr_actual IS NULL
        AND politica_actual_ref IS NULL AND politica_actual_sha256 IS NULL)
       OR (tipo NOT IN ('revocacion','revocacion_control_is2_sin_actor_humano_nominal')
        AND acr_actual IS NOT NULL
        AND politica_actual_ref IS NOT NULL AND politica_actual_sha256 IS NOT NULL
        AND acr_actual=acr_original
        AND politica_actual_ref=politica_original_ref
        AND politica_actual_sha256=politica_original_sha256)),
    registrada_en timestamptz(6) NOT NULL CHECK (pg_catalog.isfinite(registrada_en)),
    UNIQUE (presentacion_ref, operacion_ref)
);
CREATE INDEX auditoria_presentacion_certificado_sesion_idx
    ON vec_autorizacion_atestada_v3.auditoria_presentacion_certificado_v1
    (sesion_original_ref, registrada_en);
CREATE FUNCTION vec_autorizacion_atestada_v3.cotejar_asiento_presentacion_certificado_v1()
RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $funcion$
BEGIN
    IF NOT EXISTS (SELECT 1
          FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a
         WHERE a.auditoria_ref=NEW.auditoria_ref
           AND a.secuencia=NEW.auditoria_secuencia
           AND a.huella_sha256=NEW.auditoria_huella_sha256
           AND a.evento_material_sha256=NEW.evento_material_sha256
           AND a.tipo_registro='presentacion_certificado_v1'
           AND a.presentacion_certificado_detalle->>'operacion_ref'=NEW.operacion_ref
           AND a.presentacion_certificado_detalle->>'presentacion_ref'=NEW.presentacion_ref
           AND a.presentacion_certificado_detalle->>'recibo_sha256'=NEW.recibo_sha256
         FOR SHARE) THEN
        RAISE EXCEPTION 'AD221: detalle sin asiento común exacto' USING ERRCODE='23514';
    END IF;
    RETURN NEW;
END $funcion$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.cotejar_asiento_presentacion_certificado_v1()
    FROM PUBLIC;
CREATE TRIGGER asiento_exacto BEFORE INSERT
    ON vec_autorizacion_atestada_v3.auditoria_presentacion_certificado_v1
    FOR EACH ROW EXECUTE FUNCTION
        vec_autorizacion_atestada_v3.cotejar_asiento_presentacion_certificado_v1();
ALTER TABLE vec_autorizacion_atestada_v3.auditoria_presentacion_certificado_v1
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_autorizacion_atestada_v3.auditoria_presentacion_certificado_v1
    FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_exacto
    ON vec_autorizacion_atestada_v3.auditoria_presentacion_certificado_v1
    FOR ALL TO vec_autorizacion_atestada_v3_propietario
    USING (current_user = 'vec_autorizacion_atestada_v3_propietario')
    WITH CHECK (current_user = 'vec_autorizacion_atestada_v3_propietario');
CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE
    ON vec_autorizacion_atestada_v3.auditoria_presentacion_certificado_v1
    FOR EACH ROW EXECUTE FUNCTION vec_autorizacion_atestada_v3.rechazar_mutacion();
CREATE TRIGGER no_truncar BEFORE TRUNCATE
    ON vec_autorizacion_atestada_v3.auditoria_presentacion_certificado_v1
    FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion_atestada_v3.rechazar_truncado();
REVOKE ALL ON TABLE vec_autorizacion_atestada_v3.auditoria_presentacion_certificado_v1 FROM PUBLIC;
REVOKE ALL ON TYPE vec_autorizacion_atestada_v3.auditoria_presentacion_certificado_v1 FROM PUBLIC;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_auditoria_presentacion_certificado_v1(
    p_operacion_ref text, p_presentacion_ref text,
    p_autenticacion_original_ref text, p_sesion_original_ref text,
    p_cuenta_ref text, p_actor_cuenta_ref text,
    p_actor_autenticacion_ref text, p_actor_sesion_ref text,
    p_superficie text, p_tipo text,
    p_recibo_sha256 text,p_acr_original text,p_acr_actual text,
    p_politica_original_ref text,p_politica_original_sha256 text,
    p_politica_actual_ref text,p_politica_actual_sha256 text,
    p_sujeto_id_hmac bytea,p_canal_sha256 text,
    p_asercion_actual_sha256 text,p_presentacion_valida_hasta timestamptz,
    p_control_sesion_ref text,p_control_sesion_revision text,
    p_control_origen_operacion_ref text,
    p_registrada_en timestamptz
) RETURNS void
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path = pg_catalog, pg_temp
SET row_security = on
SET lock_timeout = '2s'
SET statement_timeout = '10s'
AS $funcion$
DECLARE
    v_detalle jsonb;v_material bytea;v_material_sha text;
    v_evento_ref text;v_correlacion_ref text;v_ref text;v_huella text;
    v_secuencia numeric;v_anterior text;v_existente record;
    v_accion text;v_motivo text;v_canal text;v_campo text;
BEGIN
    IF pg_catalog.current_setting('transaction_read_only') <> 'off'
       OR (p_tipo IS DISTINCT FROM 'revocacion_control_is2_sin_actor_humano_nominal'
           AND pg_catalog.current_setting('transaction_isolation') <> 'serializable')
       OR pg_catalog.current_setting('role') <> 'none'
       OR pg_catalog.pg_is_in_recovery()
       OR (p_tipo IS DISTINCT FROM 'revocacion_control_is2_sin_actor_humano_nominal'
           AND pg_catalog.current_setting('TimeZone')<>'UTC')
       OR p_operacion_ref !~ '^opr_[A-Za-z0-9_-]{22,128}$'
       OR p_presentacion_ref !~ '^prs_[A-Za-z0-9_-]{22,128}$'
       OR p_recibo_sha256 !~ '^[0-9a-f]{64}$'
       OR p_tipo IS NULL
       OR p_tipo NOT IN ('apertura','reanudacion','renovacion','revocacion',
           'revocacion_control_is2_sin_actor_humano_nominal')
       OR p_acr_original IS DISTINCT FROM 'urn:vec:acr:certificado-desarrollo-protegido'
       OR p_politica_original_ref !~ '^pga_[A-Za-z0-9_-]{22,128}$'
       OR p_politica_original_sha256 !~ '^[0-9a-f]{64}$'
       OR p_sujeto_id_hmac IS NULL OR pg_catalog.octet_length(p_sujeto_id_hmac)<>32
       OR p_registrada_en IS NULL OR NOT pg_catalog.isfinite(p_registrada_en)
       OR p_registrada_en>pg_catalog.clock_timestamp()
       OR (p_tipo='revocacion_control_is2_sin_actor_humano_nominal' AND NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_roles login
            JOIN pg_catalog.pg_auth_members m ON m.member=login.oid
           WHERE login.rolname=session_user
             AND login.rolcanlogin AND login.rolinherit
             AND m.roleid='vec_identidad_sesiones_v1_revocador'::pg_catalog.regrole
             AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option
             AND (SELECT pg_catalog.count(*) FROM pg_catalog.pg_auth_members z
                   WHERE z.member=login.oid)=1))
       OR (p_tipo='revocacion_control_is2_sin_actor_humano_nominal' AND (
           p_actor_cuenta_ref IS NOT NULL
           OR p_actor_autenticacion_ref IS NOT NULL
           OR p_actor_sesion_ref IS NOT NULL
           OR p_control_sesion_ref IS NULL
           OR p_control_sesion_ref !~ '^cse_[A-Za-z0-9_-]{22,128}$'
           OR p_control_sesion_revision IS NULL
           OR p_control_sesion_revision !~ '^[1-9][0-9]{0,19}$'
           OR p_control_origen_operacion_ref IS DISTINCT FROM p_operacion_ref))
       OR (p_tipo<>'revocacion_control_is2_sin_actor_humano_nominal' AND (
           p_actor_cuenta_ref IS NULL OR p_actor_autenticacion_ref IS NULL
           OR p_actor_sesion_ref IS NULL OR p_control_sesion_ref IS NOT NULL
           OR p_control_sesion_revision IS NOT NULL
           OR p_control_origen_operacion_ref IS NOT NULL))
       OR (p_tipo IN ('revocacion','revocacion_control_is2_sin_actor_humano_nominal') AND (
           p_acr_actual IS NOT NULL OR p_politica_actual_ref IS NOT NULL
           OR p_politica_actual_sha256 IS NOT NULL OR p_canal_sha256 IS NOT NULL
           OR p_asercion_actual_sha256 IS NOT NULL
           OR p_presentacion_valida_hasta IS NOT NULL))
       OR (p_tipo NOT IN ('revocacion','revocacion_control_is2_sin_actor_humano_nominal') AND (
           p_acr_actual IS DISTINCT FROM p_acr_original
           OR p_politica_actual_ref IS DISTINCT FROM p_politica_original_ref
           OR p_politica_actual_sha256 IS DISTINCT FROM p_politica_original_sha256
           OR p_canal_sha256 !~ '^[0-9a-f]{64}$'
           OR p_asercion_actual_sha256 !~ '^[0-9a-f]{64}$'
           OR p_presentacion_valida_hasta IS NULL
           OR p_presentacion_valida_hasta<=p_registrada_en)) THEN
        RAISE EXCEPTION 'AD221: transaccion no acreditada' USING ERRCODE='42501';
    END IF;
    -- Esta fase precede al perfil activo: cuenta y sujeto de la prueba
    -- actual son nominales; actor_ref V3 y decision_ref permanecen NULL.
    v_detalle:=pg_catalog.jsonb_build_object(
        'fase',CASE WHEN p_tipo='revocacion_control_is2_sin_actor_humano_nominal'
            THEN 'sin_actor_humano_nominal' ELSE 'pre_f1_sin_perfil_activo' END,
        'operacion_ref',p_operacion_ref,'presentacion_ref',p_presentacion_ref,
        'autenticacion_original_ref',p_autenticacion_original_ref,
        'sesion_original_ref',p_sesion_original_ref,
        'cuenta_ref',p_cuenta_ref,'actor_cuenta_ref',p_actor_cuenta_ref,
        'actor_autenticacion_ref',p_actor_autenticacion_ref,
        'actor_sesion_ref',p_actor_sesion_ref,
        'sujeto_id_hmac',pg_catalog.encode(p_sujeto_id_hmac,'hex'),
        'superficie',p_superficie,'tipo',p_tipo,'recibo_sha256',p_recibo_sha256,
        'acr_original',p_acr_original,'acr_actual',p_acr_actual,
        'politica_original_ref',p_politica_original_ref,
        'politica_original_sha256',p_politica_original_sha256,
        'politica_actual_ref',p_politica_actual_ref,
        'politica_actual_sha256',p_politica_actual_sha256,
        'canal_sha256',p_canal_sha256,
        'asercion_actual_sha256',p_asercion_actual_sha256,
        'presentacion_valida_hasta',p_presentacion_valida_hasta,
        'control_sesion_ref',p_control_sesion_ref,
        'control_sesion_revision',p_control_sesion_revision,
        'control_origen_operacion_ref',p_control_origen_operacion_ref,
        'registrada_en',p_registrada_en);
    IF pg_catalog.octet_length(v_detalle::text)>8192 THEN
        RAISE EXCEPTION 'AD221: detalle excede limite' USING ERRCODE='22023';
    END IF;
    v_evento_ref:='evento_'||pg_catalog.substr(pg_catalog.encode(pg_catalog.sha256(
        pg_catalog.convert_to(p_operacion_ref,'UTF8')),'hex'),1,32);
    v_correlacion_ref:='correlacion_'||pg_catalog.substr(pg_catalog.encode(
        pg_catalog.sha256(pg_catalog.convert_to(
            p_operacion_ref||':'||p_presentacion_ref,'UTF8')),'hex'),1,32);
    v_accion:=CASE p_tipo
        WHEN 'revocacion' THEN 'revocar_vinculo_certificado_v1'
        WHEN 'revocacion_control_is2_sin_actor_humano_nominal' THEN 'revocar_sesion_v1'
        ELSE 'presentar_certificado_v1' END;
    v_motivo:=CASE p_tipo WHEN 'apertura' THEN 'presentacion_abierta'
        WHEN 'renovacion' THEN 'presentacion_renovada'
        WHEN 'reanudacion' THEN 'presentacion_reanudada'
        WHEN 'revocacion_control_is2_sin_actor_humano_nominal' THEN 'control_sesion_revocado'
        ELSE 'vinculo_revocado' END;
    v_canal:=CASE p_tipo
        WHEN 'revocacion' THEN 'identidad_sesion_vigente'
        WHEN 'revocacion_control_is2_sin_actor_humano_nominal' THEN 'control_sesion_is2'
        ELSE 'certificado_mtls_atestado_por_frontera' END;
    -- Preimagen ordenada y encuadrada como AD219. NULL de la revocacion se
    -- representa por cadena vacia, diferenciada por tipo y dominio cerrados.
    v_material:=vec_autorizacion_atestada_v3.encuadrar_mac(
        'vec.auditoria.presentacion-certificado.v1');
    FOREACH v_campo IN ARRAY ARRAY[
        v_evento_ref,session_user::text,p_operacion_ref,p_presentacion_ref,
        p_autenticacion_original_ref,p_sesion_original_ref,p_cuenta_ref,
        COALESCE(p_actor_cuenta_ref,''),COALESCE(p_actor_autenticacion_ref,''),
        COALESCE(p_actor_sesion_ref,''),
        pg_catalog.encode(p_sujeto_id_hmac,'hex'),p_superficie,p_tipo,
        p_recibo_sha256,p_acr_original,COALESCE(p_acr_actual,''),
        p_politica_original_ref,p_politica_original_sha256,
        COALESCE(p_politica_actual_ref,''),
        COALESCE(p_politica_actual_sha256,''),COALESCE(p_canal_sha256,''),
        COALESCE(p_asercion_actual_sha256,''),
        CASE WHEN p_presentacion_valida_hasta IS NULL THEN '' ELSE
            pg_catalog.to_char(p_presentacion_valida_hasta AT TIME ZONE 'UTC',
                'YYYY-MM-DD"T"HH24:MI:SS.US"Z"') END,
        pg_catalog.to_char(p_registrada_en AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
        COALESCE(p_control_sesion_ref,''),
        COALESCE(p_control_sesion_revision,''),
        COALESCE(p_control_origen_operacion_ref,''),
        v_accion,v_motivo,v_canal,v_correlacion_ref
    ] LOOP
        v_material:=v_material||vec_autorizacion_atestada_v3.encuadrar_mac(v_campo);
    END LOOP;
    v_material_sha:=pg_catalog.encode(pg_catalog.sha256(v_material),'hex');
    PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
        'vec_autorizacion_atestada_v3:evento-presentacion:'||v_evento_ref,0));
    SELECT a.auditoria_ref,a.auditoria_secuencia,a.auditoria_huella_sha256,
           a.evento_material_sha256 INTO v_existente
      FROM vec_autorizacion_atestada_v3.auditoria_presentacion_certificado_v1 a
     WHERE a.operacion_ref=p_operacion_ref;
    IF FOUND THEN
        IF v_existente.evento_material_sha256 IS DISTINCT FROM v_material_sha THEN
            RAISE EXCEPTION 'AD221: operacion repetida con material distinto'
                USING ERRCODE='23505';
        END IF;
        RETURN;
    END IF;
    SELECT r.secuencia_previa,r.anterior_sha256 INTO STRICT v_secuencia,v_anterior
      FROM vec_autorizacion_atestada_v3.reservar_asiento_auditoria_v5() r;
    IF v_secuencia>=9007199254740991::numeric THEN
        RAISE EXCEPTION 'AD221: secuencia agotada' USING ERRCODE='22003';
    END IF;
    v_secuencia:=v_secuencia+1;
    v_ref:='aud_v3_pc_'||pg_catalog.substr(v_evento_ref,8,32);
    v_huella:=pg_catalog.encode(pg_catalog.sha256(
        vec_autorizacion_atestada_v3.encuadrar_mac(
            'vec.auditoria.eslabon.presentacion-certificado.v1')||
        vec_autorizacion_atestada_v3.encuadrar_mac(v_secuencia::text)||
        vec_autorizacion_atestada_v3.encuadrar_mac(v_anterior)||
        vec_autorizacion_atestada_v3.encuadrar_mac(v_ref)||
        vec_autorizacion_atestada_v3.encuadrar_mac(v_material_sha)||
        vec_autorizacion_atestada_v3.encuadrar_mac(pg_catalog.to_char(
            p_registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))),'hex');
    INSERT INTO vec_autorizacion_atestada_v3.auditoria_consumo_v3(
        auditoria_ref,secuencia,anterior_sha256,huella_sha256,registrada_en,
        tipo_registro,evento_ref,evento_material_sha256,operador_login,
        presentacion_certificado_detalle,accion,modulo_id,recurso_ref,
        finalidad_ref,resultado,motivo_ref,proceso,canal,correlacion_ref)
    VALUES (v_ref,v_secuencia,v_anterior,v_huella,p_registrada_en,
        'presentacion_certificado_v1',v_evento_ref,v_material_sha,
        session_user::name,v_detalle,v_accion,'identidad',
        'presentacion_certificado:'||p_presentacion_ref,
        'autenticacion_temporal_pre_f1','permitido',v_motivo,
        'postgresql',v_canal,v_correlacion_ref);
    INSERT INTO vec_autorizacion_atestada_v3.auditoria_presentacion_certificado_v1 (
        operacion_ref,auditoria_ref,auditoria_secuencia,auditoria_huella_sha256,
        evento_material_sha256,presentacion_ref, autenticacion_original_ref,
        sesion_original_ref, cuenta_ref, actor_cuenta_ref,
        actor_autenticacion_ref,actor_sesion_ref,
        control_sesion_ref,control_sesion_revision,control_origen_operacion_ref,
        superficie, tipo, recibo_sha256,acr_original,acr_actual,
        politica_original_ref,politica_original_sha256,
        politica_actual_ref,politica_actual_sha256,
        registrada_en
    ) VALUES (
        p_operacion_ref,v_ref,v_secuencia,v_huella,v_material_sha,
        p_presentacion_ref,p_autenticacion_original_ref,
        p_sesion_original_ref, p_cuenta_ref, p_actor_cuenta_ref,
        p_actor_autenticacion_ref,p_actor_sesion_ref,
        p_control_sesion_ref,p_control_sesion_revision,
        p_control_origen_operacion_ref,p_superficie, p_tipo,
        p_recibo_sha256,p_acr_original,p_acr_actual,
        p_politica_original_ref,p_politica_original_sha256,
        p_politica_actual_ref,p_politica_actual_sha256,p_registrada_en
    );
END $funcion$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_auditoria_presentacion_certificado_v1(text,text,text,text,text,text,text,text,text,text,text,text,text,text,text,text,text,bytea,text,text,timestamptz,text,text,text,timestamptz)
    FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3
    TO vec_identidad_sesiones_v1_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_auditoria_presentacion_certificado_v1(text,text,text,text,text,text,text,text,text,text,text,text,text,text,text,text,text,bytea,text,text,timestamptz,text,text,text,timestamptz)
    TO vec_identidad_sesiones_v1_propietario;
DO $acl$
DECLARE f pg_catalog.oid:=pg_catalog.to_regprocedure(
    'vec_autorizacion_atestada_v3.registrar_auditoria_presentacion_certificado_v1(text,text,text,text,text,text,text,text,text,text,text,text,text,text,text,text,text,bytea,text,text,timestamptz,text,text,text,timestamptz)');
BEGIN
    IF f IS NULL OR NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_proc p WHERE p.oid=f
          AND p.proowner='vec_autorizacion_atestada_v3_propietario'::pg_catalog.regrole
          AND p.prosecdef AND p.provolatile='v' AND p.proparallel='u')
       OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_proc p
        CROSS JOIN LATERAL pg_catalog.aclexplode(
            COALESCE(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
        WHERE p.oid=f AND (a.grantee NOT IN (
            'vec_autorizacion_atestada_v3_propietario'::pg_catalog.regrole,
            'vec_identidad_sesiones_v1_propietario'::pg_catalog.regrole)
            OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
       OR NOT pg_catalog.has_function_privilege(
           'vec_identidad_sesiones_v1_propietario',f,'EXECUTE') THEN
        RAISE EXCEPTION 'AD221: ACL de escritor incompatible' USING ERRCODE='55000';
    END IF;
END $acl$;
COMMIT;
