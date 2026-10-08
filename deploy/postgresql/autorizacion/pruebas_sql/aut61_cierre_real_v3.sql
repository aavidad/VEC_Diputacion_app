\set ON_ERROR_STOP on
-- Ensayo prospectivo en una base desechable con AUT61 instalada una sola vez.
-- Concatenar delante un fichero privado de \set aut61_db, aut61_login y los
-- once campos aut61_propuesta_* / aut61_cierre_* indicados abajo. Los dos
-- materiales han de proceder del emisor V3 real y de dos ADMIN distintos.
-- Ejecutar como usuario postgres de la instancia de ensayo, sin -a ni -e:
-- cat fixture_privada.sql aut61_cierre_real_v3.sql | psql -X -f -
-- El fichero privado contiene material atestado y no se guarda en Git.
\if :{?aut61_db}
\else
  \echo 'AUT61 E2E: falta aut61_db'
  SELECT 1/0 AS falta_aut61_db;
\endif
\if :{?aut61_login}
\else
  \echo 'AUT61 E2E: falta aut61_login'
  SELECT 1/0 AS falta_aut61_login;
\endif
\if :{?aut61_propuesta_material}
\else
  \echo 'AUT61 E2E: falta material V3 de propuesta'
  SELECT 1/0 AS falta_material_v3_propuesta;
\endif
\if :{?aut61_cierre_material}
\else
  \echo 'AUT61 E2E: falta material V3 de cierre'
  SELECT 1/0 AS falta_material_v3_cierre;
\endif

-- Cada fase requiere material, capacidad_hex, decision_hex, motivo_hex,
-- contexto_hex, persona_version, perfil_version, payload_hex, sobre_hex,
-- evidencia_hex y raiz_hex. El LOGIN sólo posee las fachadas AUT60.
\connect :aut61_db postgres
SET search_path=pg_catalog,pg_temp;
SELECT (:'aut61_propuesta_material'::jsonb->>'material_canon')::jsonb->>'OperacionRef' AS aut61_propuesta_ref,
 (:'aut61_propuesta_material'::jsonb->>'material_canon')::jsonb#>>'{Plan,version_rol_objetivo_ref}' AS aut61_version_rol_ref,
 (:'aut61_propuesta_material'::jsonb->>'material_canon')::jsonb#>>'{Plan,definicion_nueva,rol_id}' AS aut61_rol_id,
 :'aut61_cierre_material'::jsonb->>'operacion_ref' AS aut61_cierre_ref
\gset
SELECT 1 / CASE WHEN :'aut61_propuesta_ref' ~ '^propuesta_admin:[0-9a-f]{32}$'
 AND :'aut61_cierre_ref' ~ '^cierre_admin:[0-9a-f]{32}$'
 AND :'aut61_version_rol_ref'='rol:'||:'aut61_rol_id'||':v1'
 AND :'aut61_cierre_material'::jsonb->>'propuesta_ref'=:'aut61_propuesta_ref'
 AND :'aut61_cierre_material'::jsonb->>'propuesta_huella_sha256'=:'aut61_propuesta_material'::jsonb->>'material_sha256'
 AND (:'aut61_propuesta_material'::jsonb->>'material_canon')::jsonb->>'ProponentePersonaRef'
     <>:'aut61_cierre_material'::jsonb->>'actor_persona_ref'
 AND NOT EXISTS(SELECT 1 FROM vec_autorizacion.version_rol WHERE rol_id=:'aut61_rol_id')
 AND NOT EXISTS(SELECT 1 FROM vec_autorizacion.propuesta_gobierno_rol_nuevo_v1 WHERE propuesta_ref=:'aut61_propuesta_ref')
 AND NOT EXISTS(SELECT 1 FROM vec_autorizacion.cierre_gobierno_rol_nuevo_v1 WHERE propuesta_ref=:'aut61_propuesta_ref')
 THEN 1 ELSE 0 END AS preimagen_exclusiva;

\connect :aut61_db :aut61_login
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL TIME ZONE 'UTC';
SELECT vec_autorizacion.proponer_gobierno_rol_nuevo_v1(
 :'aut61_propuesta_material',
 decode(:'aut61_propuesta_capacidad_hex','hex'),decode(:'aut61_propuesta_decision_hex','hex'),
 decode(:'aut61_propuesta_motivo_hex','hex'),decode(:'aut61_propuesta_contexto_hex','hex'),
 :'aut61_propuesta_persona_version'::numeric,:'aut61_propuesta_perfil_version'::numeric,
 decode(:'aut61_propuesta_payload_hex','hex'),decode(:'aut61_propuesta_sobre_hex','hex'),
 decode(:'aut61_propuesta_evidencia_hex','hex'),decode(:'aut61_propuesta_raiz_hex','hex')) AS aut61_propuesta_resultado
\gset
SELECT 1 / CASE WHEN :'aut61_propuesta_resultado'::jsonb->>'estado'='permitido'
 AND :'aut61_propuesta_resultado'::jsonb->>'replay'='false'
 AND :'aut61_propuesta_resultado'::jsonb->>'huella_sha256'=:'aut61_propuesta_material'::jsonb->>'material_sha256'
 AND :'aut61_propuesta_resultado'::jsonb->>'auditoria_acceso_ref' ~ '^aud_v3_'
 THEN 1 ELSE 0 END AS propuesta_real_v3;
COMMIT;

BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL TIME ZONE 'UTC';
SELECT vec_autorizacion.cerrar_gobierno_rol_nuevo_v1(
 :'aut61_cierre_material',
 decode(:'aut61_cierre_capacidad_hex','hex'),decode(:'aut61_cierre_decision_hex','hex'),
 decode(:'aut61_cierre_motivo_hex','hex'),decode(:'aut61_cierre_contexto_hex','hex'),
 :'aut61_cierre_persona_version'::numeric,:'aut61_cierre_perfil_version'::numeric,
 decode(:'aut61_cierre_payload_hex','hex'),decode(:'aut61_cierre_sobre_hex','hex'),
 decode(:'aut61_cierre_evidencia_hex','hex'),decode(:'aut61_cierre_raiz_hex','hex')) AS aut61_cierre_resultado
\gset
SELECT 1 / CASE WHEN :'aut61_cierre_resultado'::jsonb->>'estado'='permitido'
 AND :'aut61_cierre_resultado'::jsonb->>'replay'='false'
 AND :'aut61_cierre_resultado'::jsonb->>'operacion_ref'=:'aut61_cierre_ref'
 AND :'aut61_cierre_resultado'::jsonb#>>'{recibo,version_rol,rol_id}'=:'aut61_rol_id'
 AND :'aut61_cierre_resultado'::jsonb#>>'{recibo,control_posterior,estado}'='habilitada'
 AND :'aut61_cierre_resultado'::jsonb->>'auditoria_acceso_ref' ~ '^aud_v3_'
 THEN 1 ELSE 0 END AS cierre_real_v3;
COMMIT;

-- El lector DBA comprueba el efecto y la auditoría común persistidos. El
-- LOGIN del consumidor no recibe SELECT sobre estas tablas.
\connect :aut61_db postgres
SET search_path=pg_catalog,pg_temp;
SELECT 1 / CASE WHEN
 (SELECT count(*) FROM vec_autorizacion.propuesta_gobierno_rol_nuevo_v1 WHERE propuesta_ref=:'aut61_propuesta_ref')=1
 AND (SELECT count(*) FROM vec_autorizacion.cierre_gobierno_rol_nuevo_v1 WHERE propuesta_ref=:'aut61_propuesta_ref' AND operacion_ref=:'aut61_cierre_ref')=1
 AND (SELECT count(*) FROM vec_autorizacion.version_rol WHERE version_rol_ref=:'aut61_version_rol_ref' AND rol_id=:'aut61_rol_id' AND version=1)=1
 AND (SELECT count(*) FROM vec_autorizacion.control_vigencia_version_rol_actual WHERE version_rol_ref=:'aut61_version_rol_ref' AND revision=1)=1
 AND (SELECT count(*) FROM vec_autorizacion.outbox_gobierno_rol_nuevo_v1 WHERE operacion_ref IN (:'aut61_propuesta_ref',:'aut61_cierre_ref'))=2
 AND (SELECT count(*) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a
  JOIN vec_autorizacion.propuesta_gobierno_rol_nuevo_v1 p ON p.auditoria_ref=a.auditoria_ref
  WHERE p.propuesta_ref=:'aut61_propuesta_ref' AND a.accion='administracion.perfiles.definicion.proponer')=1
 AND (SELECT count(*) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a
  JOIN vec_autorizacion.cierre_gobierno_rol_nuevo_v1 c ON c.auditoria_ref=a.auditoria_ref
  WHERE c.propuesta_ref=:'aut61_propuesta_ref' AND a.accion='administracion.perfiles.definicion.aprobar')=1
 AND EXISTS(SELECT 1 FROM vec_autorizacion.cierre_gobierno_rol_nuevo_v1 c
  JOIN vec_autorizacion.version_rol v ON v.version_rol_ref=c.version_rol_ref AND v.huella_sha256=c.rol_sha256
  JOIN vec_autorizacion.control_vigencia_version_rol cv ON cv.version_rol_ref=v.version_rol_ref
   AND cv.revision=1 AND cv.huella_sha256=c.control_sha256
  WHERE c.propuesta_ref=:'aut61_propuesta_ref'
   AND c.resultado#>>'{recibo,recibo_ref}'=:'aut61_cierre_resultado'::jsonb#>>'{recibo,recibo_ref}'
   AND c.aprobador_persona_ref<>(SELECT p.proponente_persona_ref
    FROM vec_autorizacion.propuesta_gobierno_rol_nuevo_v1 p WHERE p.propuesta_ref=c.propuesta_ref))
 THEN 1 ELSE 0 END AS efecto_y_auditoria_reales;
