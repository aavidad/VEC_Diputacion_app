\set ON_ERROR_STOP on
SET SESSION AUTHORIZATION vec_bolsa_accesos_propietario;
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE READ WRITE;
SELECT vec_bolsa_registro_accesos.registrar_interno_v1(jsonb_build_object(
    'actor_id', 'hmac-sha256:bolsa_accesos_v1:' || repeat('a', 64),
    'actor_profile', 'auditoria-interna', 'actor_roles', '[]'::jsonb,
    'represented_subject_id', '', 'auth_method', 'sso',
    'auth_assurance', 'alto', 'authorization_ref', '',
    'purpose', 'prueba-concurrencia', 'action', 'expediente.leer',
    'module_id', 'vec.module.bolsa',
    'subject_ref', 'expediente:sha256:' || repeat(:'marca', 64),
    'object_version', 1, 'expediente_ref', '', 'document_ref', '',
    'rule_ref', '', 'reason', '', 'result', 'permitido',
    'before_hash', '', 'after_hash', '',
    'correlation_ref', 'carrera-' || :'marca' || '-' || repeat(:'marca', 32),
    'metadata', '{}'::jsonb, 'occurred_at', :'instante'
));
COMMIT;
