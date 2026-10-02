-- Solo clon sintético. Fixture real AUT24: dos perfiles propios A/B vigentes,
-- una sesión común recién registrada y ninguna selección anterior. No usa dobles.
-- Variables: login_perfiles; F9 entorno/host/audiencia/certificado_sha256/ca_sha256/
-- autenticada/revocada/crl_hasta/certificado_hasta; perfil_a/perfil_b;
-- autenticacion_ref/sesion_ref/cuenta_ref/cuenta_ordinaria_ref/persona_ref/politica_ref/politica_huella_sha256.
-- Segunda sesión común real recién registrada: autenticacion_nueva_ref y sesion_nueva_ref.
\set ON_ERROR_STOP on
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SET SESSION AUTHORIZATION :login_perfiles;
SELECT count(*)=0 AS exige_elegir FROM vec_identidad_sesiones_v1.autoseleccionar_perfil_admin_unico_v1(:'entorno',:'host',:'audiencia',:'certificado_sha256',:'ca_sha256',:'autenticada',:'revocada',:'crl_hasta',:'certificado_hasta') \gset
\if :exige_elegir
\else
\quit 1
\endif
SELECT * FROM vec_identidad_sesiones_v1.seleccionar_perfil_admin_v1(:'entorno',:'host',:'audiencia',:'certificado_sha256',:'ca_sha256',:'autenticada',:'revocada',:'crl_hasta',:'certificado_hasta',:'perfil_a',0) \gset a_
SELECT :'a_seleccion_revision'::numeric=1 AS revision_inicial \gset
\if :revision_inicial
\else
\quit 1
\endif
SELECT * FROM vec_identidad_sesiones_v1.seleccionar_perfil_admin_v1(:'entorno',:'host',:'audiencia',:'certificado_sha256',:'ca_sha256',:'autenticada',:'revocada',:'crl_hasta',:'certificado_hasta',:'perfil_a',1) \gset noop_
SELECT :'noop_seleccion_revision'::numeric=1 AND :'noop_auditoria_ref'=:'a_auditoria_ref' AS noop_sin_efecto \gset
\if :noop_sin_efecto
\else
\quit 1
\endif
SELECT vec_identidad_sesiones_v1.vincular_sesion_admin_perfiles_v1(:'entorno',:'host',:'audiencia',:'certificado_sha256',:'ca_sha256',:'autenticada',:'revocada',:'crl_hasta',:'certificado_hasta',:'autenticacion_ref',:'sesion_ref') AS vinculada \gset
\if :vinculada
\else
\quit 1
\endif
RESET SESSION AUTHORIZATION;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SELECT vec_identidad_sesiones_v1.revalidar_sesion_admin_perfiles_v1(:'autenticacion_ref',:'sesion_ref',:'cuenta_ref',:'cuenta_ordinaria_ref',:'persona_ref',:'perfil_a',:'politica_ref',:'politica_huella_sha256') AS vigente_a \gset
\if :vigente_a
\else
\quit 1
\endif
RESET ROLE;
SET SESSION AUTHORIZATION :login_perfiles;
SELECT * FROM vec_identidad_sesiones_v1.seleccionar_perfil_admin_v1(:'entorno',:'host',:'audiencia',:'certificado_sha256',:'ca_sha256',:'autenticada',:'revocada',:'crl_hasta',:'certificado_hasta',:'perfil_b',1) \gset b_
SAVEPOINT cas_obsoleto;
\set ON_ERROR_STOP off
SELECT * FROM vec_identidad_sesiones_v1.seleccionar_perfil_admin_v1(:'entorno',:'host',:'audiencia',:'certificado_sha256',:'ca_sha256',:'autenticada',:'revocada',:'crl_hasta',:'certificado_hasta',:'perfil_a',1);
\set cas_sqlstate :SQLSTATE
ROLLBACK TO SAVEPOINT cas_obsoleto;
\set ON_ERROR_STOP on
SELECT :'cas_sqlstate'='40001' AS cas_denegado \gset
\if :cas_denegado
\else
\quit 1
\endif
SELECT * FROM vec_identidad_sesiones_v1.seleccionar_perfil_admin_v1(:'entorno',:'host',:'audiencia',:'certificado_sha256',:'ca_sha256',:'autenticada',:'revocada',:'crl_hasta',:'certificado_hasta',:'perfil_a',2) \gset regreso_
SELECT :'b_seleccion_revision'::numeric=2 AND :'regreso_seleccion_revision'::numeric=3 AS aba_versionado \gset
\if :aba_versionado
\else
\quit 1
\endif
RESET SESSION AUTHORIZATION;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SELECT NOT vec_identidad_sesiones_v1.revalidar_sesion_admin_perfiles_v1(:'autenticacion_ref',:'sesion_ref',:'cuenta_ref',:'cuenta_ordinaria_ref',:'persona_ref',:'perfil_a',:'politica_ref',:'politica_huella_sha256') AS antigua_no_revive \gset
\if :antigua_no_revive
\else
\quit 1
\endif
RESET ROLE;
SELECT :'autenticacion_nueva_ref'<>:'autenticacion_ref' AND :'sesion_nueva_ref'<>:'sesion_ref' AS nuevas_referencias \gset
\if :nuevas_referencias
\else
\quit 1
\endif
SET SESSION AUTHORIZATION :login_perfiles;
SELECT vec_identidad_sesiones_v1.vincular_sesion_admin_perfiles_v1(:'entorno',:'host',:'audiencia',:'certificado_sha256',:'ca_sha256',:'autenticada',:'revocada',:'crl_hasta',:'certificado_hasta',:'autenticacion_nueva_ref',:'sesion_nueva_ref') AS nueva_vinculada \gset
\if :nueva_vinculada
\else
\quit 1
\endif
RESET SESSION AUTHORIZATION;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SELECT vec_identidad_sesiones_v1.revalidar_sesion_admin_perfiles_v1(:'autenticacion_nueva_ref',:'sesion_nueva_ref',:'cuenta_ref',:'cuenta_ordinaria_ref',:'persona_ref',:'perfil_a',:'politica_ref',:'politica_huella_sha256') AS nueva_vigente \gset
\if :nueva_vigente
\else
\quit 1
\endif
RESET ROLE;
-- Audit-ref solo acredita los eventos almacenados: tres filas, tres revisiones,
-- dos cambios después de la inicial; no-op y CAS fallido no añaden otros eventos.
SELECT count(*)=3 AND count(DISTINCT auditoria_ref)=3 AS historial_unico FROM vec_contexto_actor_v1.seleccion_perfil_admin_v1 WHERE cuenta_ref=:'cuenta_ref' \gset
\if :historial_unico
\else
\quit 1
\endif
ROLLBACK;
