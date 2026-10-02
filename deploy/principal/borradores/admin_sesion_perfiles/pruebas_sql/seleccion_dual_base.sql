-- Solo clon y fuente AUT24 central real. La cuenta debe tener perfiles App/Sys,
-- los dos roles/categorías publicados y audiencia de selector configurada.
-- F9 usa audiencia_selector; resolver App usa audiencia_app del catálogo.
-- Variables: login_perfiles, entorno, host, certificado_sha256, ca_sha256,
-- autenticada, revocada, crl_hasta, certificado_hasta, perfil_app, perfil_sys.
\set ON_ERROR_STOP on
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET SESSION AUTHORIZATION :login_perfiles;
SELECT * FROM vec_identidad_sesiones_v1.seleccionar_perfil_admin_v1(:'entorno',:'host',:'audiencia_selector',:'certificado_sha256',:'ca_sha256',:'autenticada',:'revocada',:'crl_hasta',:'certificado_hasta',:'perfil_sys',0) \gset sys_
RESET SESSION AUTHORIZATION;
COMMIT;
-- GET con Sys activo usa la identidad base, no el resolver de Aplicación.
BEGIN ISOLATION LEVEL READ COMMITTED;
SET SESSION AUTHORIZATION :login_perfiles;
SELECT count(*)=2 AND count(DISTINCT categoria_admin)=2 AND bool_and(rol_version_ref IS NOT NULL AND clave_i18n IS NOT NULL) AS dos_propios FROM vec_identidad_sesiones_v1.listar_perfiles_admin_v1(:'entorno',:'host',:'audiencia_selector',:'certificado_sha256',:'ca_sha256',:'autenticada',:'revocada',:'crl_hasta',:'certificado_hasta') \gset
\if :dos_propios
\else
\quit 1
\endif
RESET SESSION AUTHORIZATION;
COMMIT;
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET SESSION AUTHORIZATION :login_perfiles;
SELECT * FROM vec_identidad_sesiones_v1.seleccionar_perfil_admin_v1(:'entorno',:'host',:'audiencia_selector',:'certificado_sha256',:'ca_sha256',:'autenticada',:'revocada',:'crl_hasta',:'certificado_hasta',:'perfil_app',1) \gset app_
SELECT count(*)=1 AND bool_and(rol_id='administracion_perfiles') AS app_resuelve FROM vec_identidad_sesiones_v1.resolver_cuenta_admin_perfiles_v1(:'entorno',:'host',:'audiencia_app',:'certificado_sha256',:'ca_sha256',:'autenticada',:'revocada') \gset
\if :app_resuelve
\else
\quit 1
\endif
SELECT * FROM vec_contexto_actor_v1.resolver_y_registrar_contexto_admin_perfiles_v1('oca_fixture_dual_app_aaaaaaaaaaaa','rca_fixture_dual_app_aaaaaaaaaaaa',:'cuenta_ref',:'autenticada');
SELECT * FROM vec_identidad_sesiones_v1.seleccionar_perfil_admin_v1(:'entorno',:'host',:'audiencia_selector',:'certificado_sha256',:'ca_sha256',:'autenticada',:'revocada',:'crl_hasta',:'certificado_hasta',:'perfil_sys',2) \gset regreso_sys_
-- Si las audiencias coinciden, SQL devuelve el rol Sys real y la guarda App de
-- Go debe denegar. En ambos casos, nunca se presenta Sys como rol de App.
SELECT count(*)=0 AS sql_no_otorga_app FROM vec_identidad_sesiones_v1.resolver_cuenta_admin_perfiles_v1(:'entorno',:'host',:'audiencia_app',:'certificado_sha256',:'ca_sha256',:'autenticada',:'revocada') WHERE rol_id='administracion_perfiles' \gset
\if :sql_no_otorga_app
\else
\quit 1
\endif
RESET SESSION AUTHORIZATION;
COMMIT;
BEGIN ISOLATION LEVEL READ COMMITTED;
SET SESSION AUTHORIZATION :login_perfiles;
SELECT count(*)=2 AND bool_or(seleccionado AND perfil_ref=:'perfil_sys') AS selector_sigue FROM vec_identidad_sesiones_v1.listar_perfiles_admin_v1(:'entorno',:'host',:'audiencia_selector',:'certificado_sha256',:'ca_sha256',:'autenticada',:'revocada',:'crl_hasta',:'certificado_hasta') \gset
\if :selector_sigue
\else
\quit 1
\endif
RESET SESSION AUTHORIZATION;
COMMIT;
-- Esta sonda conserva historia sintética en el clon para cubrir GET tras COMMIT;
-- su contenedor/copia se desecha al terminar. Nunca se ejecuta en principal.
