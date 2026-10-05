\set ON_ERROR_STOP on
-- LOGINs y GRANTs de ensayo se provisionan fuera de Git por Dirección.
-- Sólo clon POST186+AD187; no instalar ni reaplicar desde esta prueba.
SET SESSION AUTHORIZATION :"preservacion_configurador_login";
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL ROLE vec_auditoria_preservacion_configurador;
SET LOCAL timezone='UTC';
SELECT vec_autorizacion_atestada_v3.configurar_preservacion_auditoria_v1(
 '{"publicacion_ref":"preservacion_11111111111111111111111111111111","version":1,"preimagen_sha256":"0000000000000000000000000000000000000000000000000000000000000000","decision_tecnica_ref":"decision_tecnica_22222222222222222222222222222222","estado":"provisional","medida":"conservar_todo_sin_expurgo"}',
 'correlacion_33333333333333333333333333333333') AS publicacion \gset
COMMIT;
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL ROLE vec_auditoria_preservacion_configurador;
SET LOCAL timezone='UTC';
SELECT vec_autorizacion_atestada_v3.configurar_preservacion_auditoria_v1(
 '{"medida":"conservar_todo_sin_expurgo","estado":"provisional","decision_tecnica_ref":"decision_tecnica_22222222222222222222222222222222","preimagen_sha256":"0000000000000000000000000000000000000000000000000000000000000000","version":1,"publicacion_ref":"preservacion_11111111111111111111111111111111"}',
 'correlacion_44444444444444444444444444444444') AS replay \gset
COMMIT;
SELECT :'publicacion'::jsonb->>'estado'='publicada' AS publicada,
 :'replay'::jsonb->>'estado'='replay' AS recuperada,
 :'publicacion'::jsonb->'acuse_original'=:'replay'::jsonb->'acuse_original' AS acuse_original_identico,
 :'publicacion'::jsonb->'configuracion_sha256'=:'replay'::jsonb->'configuracion_sha256' AS huella_identica;
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION :"preservacion_consultor_login";
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL ROLE vec_auditoria_preservacion_consultor;
SET LOCAL timezone='UTC';
SELECT vec_autorizacion_atestada_v3.consultar_preservacion_auditoria_v1(1,'correlacion_55555555555555555555555555555555') AS consulta \gset
COMMIT;
SELECT :'consulta'::jsonb->>'estado'='consultada' AS consultada,
 :'consulta'::jsonb->'acuse_original'=:'publicacion'::jsonb->'acuse_original' AS original_conservado,
 (:'consulta'::jsonb->'acuse_acceso'->>'secuencia')::numeric>(:'publicacion'::jsonb->'acuse_original'->>'secuencia')::numeric AS lectura_auditada;
RESET SESSION AUTHORIZATION;
