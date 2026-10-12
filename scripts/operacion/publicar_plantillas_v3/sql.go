package main

// SQL de datos: no crea funciones, roles, permisos o tablas. Mantiene los
// recibos anteriores y el lector documental CT133/135 instalado.
const sqlCabeza = `SELECT secuencia,version,revision,estado,origen,catalogo_huella_sha256,
 contenido_json_sha256,coalesce(recibo_ref,''),auditoria_ref,actor_ref,decision_ref,catalogo,
 coalesce(clave_idempotencia::text,''),coalesce(solicitud_huella_sha256,''),consumo_huella_sha256
 FROM vec_contratacion_temporal.catalogo_plantillas_historia_v1 `

const sqlBaseAuditoria = `SELECT EXISTS (
 SELECT 1 FROM vec_contratacion_temporal.catalogo_plantillas_historia_v1 h
 JOIN vec_contratacion_temporal.catalogo_plantillas_provision_auditoria_v1 a ON a.historia_secuencia=h.secuencia
 WHERE h.secuencia=$1 AND h.auditoria_ref=a.auditoria_ref
 AND h.consumo_huella_sha256=a.solicitud_huella_sha256
 AND h.actor_ref=a.instalador_ref AND h.provision_fuente_ref=a.fuente_ref
 AND h.provision_aprobacion_ref=a.aprobacion_ref AND h.registrada_en=a.registrada_en
 AND h.catalogo->>'fuente_ref'=a.fuente_ref AND h.catalogo->>'aprobacion_ref'=a.aprobacion_ref
 AND h.contenido_json_sha256=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(h.catalogo::text,'UTF8')),'hex'))`

const sqlReplayIntegro = `SELECT EXISTS (
 SELECT 1 FROM vec_contratacion_temporal.catalogo_plantillas_historia_v1 h
 JOIN vec_contratacion_temporal.catalogo_plantillas_provision_auditoria_v1 a ON a.historia_secuencia=h.secuencia
 JOIN vec_contratacion_temporal.catalogo_plantillas_outbox_v1 o ON o.historia_secuencia=h.secuencia
 WHERE h.secuencia=$1 AND h.estado='publicado' AND h.origen='publicacion'
 AND h.organizacion_ref='organizacion:desarrollo:dipgra'
 AND h.recibo_ref=a.recibo_ref AND h.recibo_ref=o.recibo_ref AND h.auditoria_ref=a.auditoria_ref
 AND h.solicitud_huella_sha256=a.solicitud_huella_sha256 AND h.consumo_huella_sha256=a.solicitud_huella_sha256
 AND h.actor_ref=a.instalador_ref AND h.decision_ref=a.aprobacion_ref
 AND h.catalogo->>'fuente_ref'=a.fuente_ref AND h.registrada_en=a.registrada_en AND h.registrada_en=o.creada_en
 AND o.tipo='contratacion_temporal.plantillas_documentos.publicado'
 AND h.contenido_json_sha256=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(h.catalogo::text,'UTF8')),'hex'))`

// La referencia operativa indica que es la alternativa autorizada de datos
// ficticios. No se inventa un consumo V3 ni una aprobación RRHH por A/B.
const sqlPublicar = `WITH material AS (
 SELECT $1::jsonb AS catalogo,$2::text AS huella,$3::uuid AS clave,$4::text AS aprobacion,
 pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to($1::jsonb::text,'UTF8')),'hex') AS json_h,
 'recibo:'||pg_catalog.gen_random_uuid()::text AS recibo,
 'provision:'||pg_catalog.gen_random_uuid()::text AS auditoria,
 pg_catalog.date_trunc('microseconds',pg_catalog.clock_timestamp()) AS instante
), solicitud AS (
 SELECT *,pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
 huella||'|'||json_h||'|'||clave::text||'|'||aprobacion,'UTF8')),'hex') AS solicitud_h FROM material
), historia AS (
 INSERT INTO vec_contratacion_temporal.catalogo_plantillas_historia_v1(
 version,revision,estado,origen,catalogo,catalogo_huella_sha256,contenido_json_sha256,
 clave_idempotencia,solicitud_huella_sha256,recibo_ref,actor_ref,decision_ref,
 consumo_huella_sha256,auditoria_ref,registrada_en,organizacion_ref)
 SELECT 3,1,'publicado','publicacion',catalogo,huella,json_h,clave,solicitud_h,recibo,
 session_user::text,aprobacion,solicitud_h,auditoria,instante,'organizacion:desarrollo:dipgra' FROM solicitud
 RETURNING *
), auditoria AS (
 INSERT INTO vec_contratacion_temporal.catalogo_plantillas_provision_auditoria_v1(
 recibo_ref,historia_secuencia,auditoria_ref,solicitud_huella_sha256,instalador_ref,fuente_ref,aprobacion_ref,registrada_en)
 SELECT recibo_ref,secuencia,auditoria_ref,solicitud_huella_sha256,actor_ref,catalogo->>'fuente_ref',decision_ref,registrada_en FROM historia
 RETURNING historia_secuencia
), evento AS (
 INSERT INTO vec_contratacion_temporal.catalogo_plantillas_outbox_v1(
 evento_ref,historia_secuencia,recibo_ref,tipo,creada_en)
 SELECT 'evento:'||pg_catalog.gen_random_uuid()::text,secuencia,recibo_ref,
 'contratacion_temporal.plantillas_documentos.publicado',registrada_en FROM historia
 RETURNING historia_secuencia
)
SELECT h.recibo_ref,h.auditoria_ref FROM historia h
 JOIN auditoria a ON a.historia_secuencia=h.secuencia JOIN evento o ON o.historia_secuencia=h.secuencia`
