\set ON_ERROR_STOP on
-- Fixture para un clon sintético con un inbox U14 ya aceptado, una persona
-- candidata externa vigente y un correo propio activo/verificado. No envía SMTP.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='30s';
CREATE TEMP TABLE u16_caso AS
 SELECT i.recibo_ref, p.persona_ref, d.correo_ref
 FROM vec_usuarios_correos_externo.avisos_inbox i
 CROSS JOIN LATERAL (
  SELECT vec_contexto_actor_v1.persona_candidato_externo_avisos_v1(
    i.material::jsonb->>'destinatario_externo_ref') AS persona_ref
 ) p
 JOIN vec_usuarios_correos_externo.correos_direccion d ON d.persona_ref=p.persona_ref
 WHERE d.activo AND d.estado='verificado'
   AND NOT EXISTS(SELECT 1 FROM vec_usuarios_correos_externo.avisos_reserva r WHERE r.recibo_ref=i.recibo_ref)
 ORDER BY i.aceptado_en,i.recibo_ref LIMIT 1;
DO $pre$ BEGIN
 IF (SELECT count(*) FROM pg_temp.u16_caso)<>1 THEN
  RAISE EXCEPTION 'U16: falta inbox y correo externo verificado sintéticos' USING ERRCODE='55000';
 END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_proc p
  WHERE p.oid=pg_catalog.to_regprocedure('vec_usuarios_correos_externo.reservar_aviso_externo_v1(text)')
   AND pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p.prosrc,'UTF8')),'hex')
    = 'eb1a20bda0ed7ded62ce80e1f39484d69faa42017484d270a8521377730c8ce8') THEN
  RAISE EXCEPTION 'U16: cuerpo correctivo no instalado' USING ERRCODE='55000';
 END IF;
END $pre$;
DO $esquema_temporal$ DECLARE nombre name; BEGIN
 SELECT nspname INTO STRICT nombre FROM pg_catalog.pg_namespace WHERE oid=pg_catalog.pg_my_temp_schema();
 EXECUTE pg_catalog.format('GRANT USAGE ON SCHEMA %I TO vec_externo_avisos_usuarios',nombre);
END $esquema_temporal$;
GRANT SELECT ON pg_temp.u16_caso TO vec_externo_avisos_usuarios;
SET SESSION AUTHORIZATION vec_externo_avisos_usuarios;
DO $recorrido$ DECLARE c record; primera jsonb; replay jsonb; BEGIN
 SELECT * INTO STRICT c FROM pg_temp.u16_caso;
 primera:=vec_usuarios_correos_externo.reservar_aviso_externo_v1(c.recibo_ref);
 IF primera->>'estado' IS DISTINCT FROM 'reservado'
    OR primera->>'replay' IS DISTINCT FROM 'false'
    OR primera->>'reserva_ref' IS NULL
    OR primera->>'persona_ref' IS DISTINCT FROM c.persona_ref
    OR primera->>'correo_ref' IS DISTINCT FROM c.correo_ref
    OR primera#>>'{sobre,clave_ref}' IS NULL
    OR primera->>'auditoria_ref' !~ '^auditoria_tecnica_externa:[0-9a-f]{32}$' THEN
  RAISE EXCEPTION 'U16: reserva con correo verificado incompleta' USING ERRCODE='55000';
 END IF;
 replay:=vec_usuarios_correos_externo.reservar_aviso_externo_v1(c.recibo_ref);
 IF replay->>'estado' IS DISTINCT FROM 'reservado'
    OR replay->>'replay' IS DISTINCT FROM 'true'
    OR replay#>>'{recibo,recibo_ref}' IS DISTINCT FROM c.recibo_ref
    OR replay->>'auditoria_ref' IS NULL
    OR replay->>'auditoria_ref'=primera->>'auditoria_ref'
    OR replay ?| ARRAY['reserva_ref','evento','persona_ref','correo_ref','sobre'] THEN
  RAISE EXCEPTION 'U16: replay reabrió la reserva o filtró el sobre' USING ERRCODE='55000';
 END IF;
END $recorrido$;
RESET SESSION AUTHORIZATION;
DO $historia$ DECLARE c record; BEGIN
 SELECT * INTO STRICT c FROM pg_temp.u16_caso;
 IF (SELECT count(*) FROM vec_usuarios_correos_externo.avisos_reserva WHERE recibo_ref=c.recibo_ref)<>1
    OR (SELECT count(*) FROM vec_usuarios_correos_externo.avisos_historia WHERE recibo_ref=c.recibo_ref AND accion='reservar')<>1
    OR (SELECT count(*) FROM vec_usuarios_correos_externo.avisos_historia WHERE recibo_ref=c.recibo_ref AND accion='replay_reservar')<>1 THEN
  RAISE EXCEPTION 'U16: reserva, replay o historia divergentes' USING ERRCODE='55000';
 END IF;
END $historia$;
ROLLBACK;
