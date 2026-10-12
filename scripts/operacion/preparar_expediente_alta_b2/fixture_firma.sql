\set ON_ERROR_STOP on
-- Dos precondiciones FICTICIAS de remisión en el clon sintético. No hay firma.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT set_config('vec.fixture.expediente', :'expediente', true);
SELECT set_config('vec.fixture.catalogo_ref', :'catalogo_ref', true);
SELECT set_config('vec.fixture.catalogo_sha', :'catalogo_sha', true);
WITH p(n,paso,perfil,firmante) AS (VALUES
 (1,'vec.contratacion_temporal.circuito_firma:1:informe_definitivo.p1',
    'perfil:ct:tecnico_rrhh','persona:fixture:b2:rrhh'),
 (2,'vec.contratacion_temporal.circuito_firma:1:informe_definitivo.p2',
    'perfil:ct:jefatura_servicio_rrhh','persona:fixture:b2:jefatura')),
 datos AS (SELECT current_setting('vec.fixture.expediente') exp,
                  current_setting('vec.fixture.catalogo_ref') cat,
                  current_setting('vec.fixture.catalogo_sha') sha)
INSERT INTO vec_contratacion_temporal.firma_documento_v1(
 firma_ref,organizacion_ref,expediente_ref,expediente_version,documento,secuencia,
 clave_idempotencia,solicitud_huella_sha256,catalogo_ref,catalogo_huella_sha256,
 paso_ref,paso_orden,resultado,original_huella_sha256,firmado_huella_sha256,
 certificado_huella_sha256,firmante_ref,politica_verificacion,verificacion_estado,
 verificacion_motivo,revocacion_estado,sello_tiempo_estado,actor_ref,perfil_ref,
 decision_ref,consumo_huella_sha256,auditoria_consumo_ref,recibo_ref,registrada_en)
SELECT 'firma-ct:'||md5(exp||':firma:'||n)::uuid::text,
 'organizacion:desarrollo:dipgra',exp,5,'informe_definitivo',n,
 'fixture-b2-firma-'||md5(exp||':'||n),
 encode(sha256(convert_to('fixture-solicitud:'||exp||':'||n,'UTF8')),'hex'),
 cat,sha,paso,n,'firmado',
 encode(sha256(convert_to('fixture-original:'||exp,'UTF8')),'hex'),
 encode(sha256(convert_to('fixture-firmado:'||exp||':'||n,'UTF8')),'hex'),
 encode(sha256(convert_to('fixture-certificado-inexistente:'||exp||':'||n,'UTF8')),'hex'),
 firmante,'politica:vec:firma:verificacion-autonoma:v1','valida','verificada',
 'vigente','no_presente',firmante,perfil,
 'decision:fixture:b2:'||md5(exp||':'||n),
 encode(sha256(convert_to('fixture-consumo:'||exp||':'||n,'UTF8')),'hex'),
 'auditoria:fixture:b2:'||md5(exp||':'||n),
 'recibo-firma-ct:'||md5(exp||':recibo:'||n)::uuid::text,
 now()+n*interval '1 microsecond'
FROM p CROSS JOIN datos
ON CONFLICT DO NOTHING;
DO $check$
DECLARE
 exp text:=current_setting('vec.fixture.expediente');
 cat text:=current_setting('vec.fixture.catalogo_ref');
 huella text:=current_setting('vec.fixture.catalogo_sha');
 n integer;
 encontrados integer;
 paso text;
 perfil text;
 firmante text;
BEGIN
 IF (SELECT count(*) FROM vec_contratacion_temporal.firma_documento_v1
     WHERE expediente_ref=exp AND expediente_version=5 AND documento='informe_definitivo'
       AND clave_idempotencia LIKE 'fixture-b2-firma-%')<>2 THEN
   RAISE EXCEPTION 'Fixture de remisión: no hay dos filas exactas';
 END IF;
 FOR n IN 1..2 LOOP
   paso:='vec.contratacion_temporal.circuito_firma:1:informe_definitivo.p'||n;
   perfil:=CASE n WHEN 1 THEN 'perfil:ct:tecnico_rrhh' ELSE 'perfil:ct:jefatura_servicio_rrhh' END;
   firmante:=CASE n WHEN 1 THEN 'persona:fixture:b2:rrhh' ELSE 'persona:fixture:b2:jefatura' END;
   SELECT count(*) INTO encontrados FROM vec_contratacion_temporal.firma_documento_v1
    WHERE expediente_ref=exp AND expediente_version=5 AND documento='informe_definitivo'
      AND secuencia=n AND paso_orden=n AND paso_ref=paso
      AND catalogo_ref=cat AND catalogo_huella_sha256=huella
      AND ('fixture-b2-firma-'||md5(exp||':'||n))=clave_idempotencia
      AND firma_ref='firma-ct:'||md5(exp||':firma:'||n)::uuid::text
      AND recibo_ref='recibo-firma-ct:'||md5(exp||':recibo:'||n)::uuid::text
      AND resultado='firmado' AND perfil_ref=perfil AND firmante_ref=firmante;
   IF encontrados<>1 THEN
     RAISE EXCEPTION 'Fixture de remisión: paso % no coincide con su material original',n;
   END IF;
 END LOOP;
END $check$;
COMMIT;
