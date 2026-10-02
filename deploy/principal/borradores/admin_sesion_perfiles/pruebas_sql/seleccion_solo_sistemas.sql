-- Clon sintético separado: cuenta privilegiada con certificado IS9 y UNA
-- asignación propia de Sistemas, sin perfil ni concesión de Aplicación.
\set ON_ERROR_STOP on
BEGIN ISOLATION LEVEL READ COMMITTED;
SET SESSION AUTHORIZATION :login_perfiles;
SELECT count(*)=1 AND bool_and(perfil_ref=:'perfil_sys') AS solo_sys FROM vec_identidad_sesiones_v1.listar_perfiles_admin_v1(:'entorno',:'host',:'audiencia_selector',:'certificado_sha256',:'ca_sha256',:'autenticada',:'revocada',:'crl_hasta',:'certificado_hasta') \gset
\if :solo_sys
\else
\quit 1
\endif
RESET SESSION AUTHORIZATION;
ROLLBACK;
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET SESSION AUTHORIZATION :login_perfiles;
SELECT * FROM vec_identidad_sesiones_v1.seleccionar_perfil_admin_v1(:'entorno',:'host',:'audiencia_selector',:'certificado_sha256',:'ca_sha256',:'autenticada',:'revocada',:'crl_hasta',:'certificado_hasta',:'perfil_sys',0) \gset sys_
SELECT :'sys_seleccion_revision'::numeric=1 AS elegida \gset
\if :elegida
\else
\quit 1
\endif
RESET SESSION AUTHORIZATION;
ROLLBACK;
