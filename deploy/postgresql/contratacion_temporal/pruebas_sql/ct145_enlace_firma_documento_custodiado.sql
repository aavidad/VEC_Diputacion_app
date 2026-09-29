\set ON_ERROR_STOP on
-- Recorrido de CT145 (enlace firma ↔ documento custodiado) sobre PostgreSQL
-- 18 real con AD3-85, CT118 y CT145 instaladas y al menos un expediente
-- sintético. SOLO PRUEBA: dentro de una única transacción se sustituye la
-- fachada AD3-85 por un doble que devuelve un consumo sintético nuevo. Todo
-- termina en ROLLBACK: la fachada real, los roles y las filas quedan como
-- estaban. El consumo V3 real lo ejercen la composición y el núcleo AD3.
SELECT a.expediente_ref AS expediente, a.version AS version, v.agregado_json->>'organizacion_ref' AS organizacion
  FROM vec_contratacion_temporal.expediente_integral_actual a
  JOIN vec_contratacion_temporal.expediente_version_integral v USING (expediente_ref, version)
 WHERE NOT EXISTS (SELECT 1 FROM vec_contratacion_temporal.firma_documento_v1 f
                    WHERE f.expediente_ref=a.expediente_ref AND f.documento='informe_definitivo')
 ORDER BY a.expediente_ref LIMIT 1 \gset

BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path = pg_catalog;
SELECT set_config('prueba_ct145.organizacion', :'organizacion', true),
       set_config('prueba_ct145.expediente', :'expediente', true),
       set_config('prueba_ct145.version', :'version', true);
CREATE ROLE vec_ct145_recorrido_ejecutor LOGIN;
GRANT vec_contratacion_temporal_ejecutor TO vec_ct145_recorrido_ejecutor;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_firma_documento_ct_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE d jsonb := convert_from(p_decision,'UTF8')::jsonb;
BEGIN
 -- Doble de prueba: eco del efecto de la decisión y consumo único.
 RETURN QUERY SELECT 'decision:'||md5(random()::text), d->>'recurso_ref', d->>'contexto_recurso_huella_sha256',
  encode(sha256(convert_to(random()::text,'UTF8')),'hex'), 'auditoria:'||md5(random()::text), clock_timestamp(), true;
END $f$;
RESET ROLE;

CREATE FUNCTION pg_temp.solicitud(p_org text, p_exp text, p_version numeric, p_paso int, p_sec int, p_resultado text,
    p_motivo text, p_original text, p_firmado text, p_clave text, p_doc text DEFAULT NULL, p_doc_version numeric DEFAULT NULL) RETURNS text LANGUAGE sql AS $s$
  SELECT jsonb_build_object('OrganizacionRef',p_org,'ExpedienteRef',p_exp,'VersionExpediente',p_version,
    'Documento','informe_definitivo','CatalogoRef','vec.contratacion_temporal.circuito_firma:1',
    'CatalogoHuella',repeat('c',64),'PasoRef','vec.contratacion_temporal.circuito_firma:1:informe_definitivo.p'||p_paso,
    'PasoOrden',p_paso,'Secuencia',p_sec,'Resultado',p_resultado,'MotivoDevolucion',p_motivo,
    'OriginalHuella',p_original,'FirmadoHuella',p_firmado,
    'CertificadoHuella',CASE WHEN p_resultado='firmado' THEN repeat('e',64) END,
    'FirmanteRef',CASE WHEN p_resultado='firmado' THEN 'ref:'||repeat('f',64) END,
    'PoliticaVerificacion',CASE WHEN p_resultado='firmado' THEN 'politica:vec:firma:verificacion-autonoma:v1' END,
    'RevocacionEstado',CASE WHEN p_resultado='firmado' THEN 'vigente' END,
    'SelloTiempoEstado',CASE WHEN p_resultado='firmado' THEN 'no_presente' END,
    'ClaveIdempotencia',p_clave,'DocumentoCustodiaRef',p_doc,'DocumentoCustodiaVersion',p_doc_version)::text
$s$;
CREATE FUNCTION pg_temp.decision(p_solicitud text) RETURNS bytea LANGUAGE sql AS $d$
  SELECT convert_to(jsonb_build_object('accion','contratacion_temporal.documento.firmar','modulo_id','contratacion_temporal',
    'tipo_recurso','firma_documento_contratacion_temporal','finalidad','gestionar_contratacion_temporal',
    'recurso_ref','operacion-firma-ct:'||(p_solicitud::jsonb->>'ClaveIdempotencia'),
    'contexto_recurso_huella_sha256',encode(sha256(convert_to('{"ambitos":{"organizacion_ref":"'||(p_solicitud::jsonb->>'OrganizacionRef')||
      '"},"atributos":{"material_sha256":"'||encode(sha256(convert_to(p_solicitud,'UTF8')),'hex')||'"}}','UTF8')),'hex'),
    'principal_id','principal:prueba:rrhh','perfil_activo_ref','perfil:prueba:rrhh')::text,'UTF8')
$d$;
CREATE FUNCTION pg_temp.registrar(p_solicitud text) RETURNS jsonb LANGUAGE sql AS $r$
  SELECT vec_contratacion_temporal.registrar_firma_documento_v2(p_solicitud,'\x',pg_temp.decision(p_solicitud),'\x','\x',1,1,'\x','\x','\x','\x')
$r$;
CREATE FUNCTION pg_temp.debe_fallar(p_sql text, p_codigo text, p_etiqueta text) RETURNS void LANGUAGE plpgsql AS $f$
BEGIN
    BEGIN
        EXECUTE p_sql;
    EXCEPTION WHEN others THEN
        IF SQLSTATE <> p_codigo THEN
            RAISE EXCEPTION 'FALLO %: código % (esperado %): %', p_etiqueta, SQLSTATE, p_codigo, SQLERRM;
        END IF;
        RAISE NOTICE 'OK %', p_etiqueta;
        RETURN;
    END;
    RAISE EXCEPTION 'FALLO %: no se rechazó', p_etiqueta;
END $f$;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA pg_temp TO PUBLIC;
SELECT set_config('prueba_ct145.enlaces_previos', (SELECT count(*) FROM vec_contratacion_temporal.firma_documento_custodia_v1)::text, true);
SET SESSION AUTHORIZATION vec_ct145_recorrido_ejecutor;
SET LOCAL statement_timeout = '15s';

-- Firma con enlace, recuperación idempotente con el mismo enlace, firma sin
-- enlace y consulta v2.
DO $recorrido$
DECLARE r jsonb; r2 jsonb; s text; org text := current_setting('prueba_ct145.organizacion'); exp text := current_setting('prueba_ct145.expediente');
    ver numeric := current_setting('prueba_ct145.version')::numeric;
BEGIN
    s := pg_temp.solicitud(org,exp,ver,1,1,'firmado',NULL,repeat('a',64),repeat('1',64),'clave-ct145-00000001','ref:'||repeat('d',64),1);
    r := pg_temp.registrar(s);
    IF r->>'YaRegistrada' <> 'false' OR r->>'DocumentoCustodiaRef' <> 'ref:'||repeat('d',64)
       OR (r->>'DocumentoCustodiaVersion')::int <> 1 OR jsonb_typeof(r->'DocumentoCustodiaVersion') <> 'number' THEN
        RAISE EXCEPTION 'FALLO firma con enlace: %', r;
    END IF;
    r2 := pg_temp.registrar(s);
    IF r2->>'YaRegistrada' <> 'true' OR r2->>'ReciboRef' <> r->>'ReciboRef'
       OR r2->>'DocumentoCustodiaRef' <> r->>'DocumentoCustodiaRef' OR r2->'DocumentoCustodiaVersion' <> r->'DocumentoCustodiaVersion' THEN
        RAISE EXCEPTION 'FALLO recuperación con enlace: %', r2;
    END IF;
    PERFORM set_config('prueba_ct145.firma', r->>'FirmaRef', true);
    -- Paso 2 del mismo borrador, sin documento custodiado: se admite.
    r := pg_temp.registrar(pg_temp.solicitud(org,exp,ver,2,2,'firmado',NULL,repeat('a',64),repeat('2',64),'clave-ct145-00000002'));
    IF r ? 'DocumentoCustodiaRef' AND jsonb_typeof(r->'DocumentoCustodiaRef') <> 'null' THEN
        RAISE EXCEPTION 'FALLO firma sin enlace: %', r;
    END IF;
    IF (SELECT count(*) FROM jsonb_array_elements(vec_contratacion_temporal.consultar_firmas_documento_v2(org,exp)) f
         WHERE f->>'DocumentoCustodiaRef' = 'ref:'||repeat('d',64) AND (f->>'DocumentoCustodiaVersion')::int = 1) <> 1
       OR EXISTS (SELECT 1 FROM jsonb_array_elements(vec_contratacion_temporal.consultar_firmas_documento_v2(org,exp)) f
                   WHERE f ?| ARRAY['FirmanteRef','CertificadoHuella','ActorRef','PerfilRef','MotivoDevolucion']) THEN
        RAISE EXCEPTION 'FALLO consulta v2';
    END IF;
    RAISE NOTICE 'OK firma con enlace, recuperación, firma sin enlace y consulta v2';
END $recorrido$;

-- Enlaces incoherentes: se rechazan sin escribir.
SELECT pg_temp.debe_fallar(format($q$SELECT pg_temp.registrar(pg_temp.solicitud(%L,%L,%s,1,3,'devuelto','Falta la fecha de efectos',NULL,NULL,'clave-ct145-00000003','ref:'||repeat('e',64),1))$q$,
    :'organizacion',:'expediente',:'version'),'22023','devolución con documento custodiado');
SELECT pg_temp.debe_fallar(format($q$SELECT pg_temp.registrar(pg_temp.solicitud(%L,%L,%s,1,3,'firmado',NULL,repeat('a',64),repeat('3',64),'clave-ct145-00000004','ref:'||repeat('e',64),NULL))$q$,
    :'organizacion',:'expediente',:'version'),'22023','referencia sin versión');
SELECT pg_temp.debe_fallar(format($q$SELECT pg_temp.registrar(pg_temp.solicitud(%L,%L,%s,1,3,'firmado',NULL,repeat('a',64),repeat('3',64),'clave-ct145-00000005',NULL,1))$q$,
    :'organizacion',:'expediente',:'version'),'22023','versión sin referencia');
SELECT pg_temp.debe_fallar(format($q$SELECT pg_temp.registrar(pg_temp.solicitud(%L,%L,%s,1,3,'firmado',NULL,repeat('a',64),repeat('3',64),'clave-ct145-00000006','ref con espacio',1))$q$,
    :'organizacion',:'expediente',:'version'),'22023','referencia con forma inválida');
SELECT pg_temp.debe_fallar(format($q$SELECT pg_temp.registrar(pg_temp.solicitud(%L,%L,%s,1,3,'firmado',NULL,repeat('a',64),repeat('3',64),'clave-ct145-00000007','ref:'||repeat('e',64),0))$q$,
    :'organizacion',:'expediente',:'version'),'22023','versión 0');
SELECT pg_temp.debe_fallar(format($q$SELECT pg_temp.registrar(replace(pg_temp.solicitud(%L,%L,%s,1,3,'firmado',NULL,repeat('a',64),repeat('3',64),'clave-ct145-00000008','ref:'||repeat('e',64),1),'"DocumentoCustodiaVersion": 1','"DocumentoCustodiaVersion": "1"'))$q$,
    :'organizacion',:'expediente',:'version'),'22023','versión como texto');
-- El mismo documento custodiado no puede enlazarse a otra firma.
SELECT pg_temp.debe_fallar(format($q$SELECT pg_temp.registrar(pg_temp.solicitud(%L,%L,%s,1,3,'firmado',NULL,repeat('a',64),repeat('3',64),'clave-ct145-00000009','ref:'||repeat('d',64),1))$q$,
    :'organizacion',:'expediente',:'version'),'23505','documento ya enlazado a otra firma');
-- Las v1 ya no están al alcance de la aplicación.
SELECT pg_temp.debe_fallar(format($q$SELECT vec_contratacion_temporal.consultar_firmas_documento_v1(%L,%L)$q$,
    :'organizacion',:'expediente'),'42501','consulta v1 retirada');
SELECT pg_temp.debe_fallar(format($q$SELECT vec_contratacion_temporal.registrar_firma_documento_v1(pg_temp.solicitud(%L,%L,%s,1,3,'firmado',NULL,repeat('a',64),repeat('3',64),'clave-ct145-00000010'),'\x',NULL,'\x','\x',1,1,'\x','\x','\x','\x')$q$,
    :'organizacion',:'expediente',:'version'),'42501','registro v1 retirado');
RESET SESSION AUTHORIZATION;

-- Un solo enlace nuevo, con la huella del firmado, y ninguno más.
DO $enlace$
BEGIN
    IF (SELECT count(*) FROM vec_contratacion_temporal.firma_documento_custodia_v1) <> current_setting('prueba_ct145.enlaces_previos')::bigint + 1
       OR NOT EXISTS (SELECT 1 FROM vec_contratacion_temporal.firma_documento_custodia_v1
                       WHERE firma_ref=current_setting('prueba_ct145.firma') AND documento_huella_sha256=repeat('1',64)
                         AND documento_version=1 AND documento_ref='ref:'||repeat('d',64)) THEN
        RAISE EXCEPTION 'FALLO fila del enlace';
    END IF;
    RAISE NOTICE 'OK un enlace con la huella del firmado';
END $enlace$;
-- El enlace es historia inmutable (el disparador responde 55000).
SELECT pg_temp.debe_fallar($q$UPDATE vec_contratacion_temporal.firma_documento_custodia_v1 SET documento_version=2$q$,
    '55000','enlace inmutable');
ROLLBACK;
