-- Requiere fixture sintético AUT24+IS9 con bootstrap consumido y sesión común
-- ya registrada. Todas las variables son referencias/fechas sintéticas, sin credenciales.
-- El llamador abre SERIALIZABLE y revierte; LOGIN nominal debe existir sin privilegios directos.
\set ON_ERROR_STOP on
SET SESSION AUTHORIZATION :login_perfiles;
SELECT vec_identidad_sesiones_v1.vincular_sesion_admin_perfiles_v1(:'entorno',:'host',:'audiencia',:'certificado_sha256',:'ca_sha256',:'autenticada',:'revocada',:'crl_hasta',:'certificado_hasta',:'autenticacion_ref',:'sesion_ref') AS vinculada \gset
\if :vinculada
\else
\quit 1
\endif
SELECT vec_identidad_sesiones_v1.vincular_sesion_admin_perfiles_v1(:'entorno',:'host',:'audiencia',:'certificado_sha256',:'ca_sha256',:'autenticada',:'revocada',:'crl_hasta',:'certificado_hasta',:'autenticacion_ref',:'sesion_ref') AS replay \gset
\if :replay
\else
\quit 1
\endif
RESET SESSION AUTHORIZATION;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SELECT vec_identidad_sesiones_v1.revalidar_sesion_admin_perfiles_v1(:'autenticacion_ref',:'sesion_ref',:'cuenta_ref',:'cuenta_ordinaria_ref',:'persona_ref',:'perfil_ref',:'politica_ref',:'politica_huella_sha256') AS vigente \gset
\if :vigente
\else
\quit 1
\endif
SELECT NOT vec_identidad_sesiones_v1.revalidar_sesion_admin_perfiles_v1(:'autenticacion_ref',:'sesion_ref',:'cuenta_ref',:'cuenta_ordinaria_ref','per_fixture_ajeno_aaaaaaaaaaaaaaa',:'perfil_ref',:'politica_ref',:'politica_huella_sha256') AS ajeno_rechazado \gset
\if :ajeno_rechazado
\else
\quit 1
\endif
RESET ROLE;
-- Tras retirar el certificado conservado, ni sesión histórica ni replay habilitan el efecto.
SELECT vec_identidad_sesiones_v1.revocar_vinculo_certificado_admin_v1(:'certificado_vinculo_ref',:certificado_vinculo_version,'acto_admin:12000000000000000000000000000001');
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SELECT NOT vec_identidad_sesiones_v1.revalidar_sesion_admin_perfiles_v1(:'autenticacion_ref',:'sesion_ref',:'cuenta_ref',:'cuenta_ordinaria_ref',:'persona_ref',:'perfil_ref',:'politica_ref',:'politica_huella_sha256') AS revocada \gset
\if :revocada
\else
\quit 1
\endif
RESET ROLE;
