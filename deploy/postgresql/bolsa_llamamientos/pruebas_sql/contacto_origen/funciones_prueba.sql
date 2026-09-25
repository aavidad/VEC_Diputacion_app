CREATE SCHEMA IF NOT EXISTS prueba; GRANT USAGE ON SCHEMA prueba TO vec_bolsa_llamamientos_ejecutor;
-- Material de autorización de ensayo para B4 (el doble solo lo refleja).
CREATE OR REPLACE FUNCTION prueba.decision(p_part text) RETURNS bytea LANGUAGE sql IMMUTABLE AS $$
 SELECT convert_to(jsonb_build_object('principal_id','per_'||repeat('a',22),'accion','bolsa.datos_contacto_participacion.registrar','modulo_id','bolsa','tipo_recurso','participacion_bolsa','finalidad','gestion_datos_contacto_participacion','recurso_ref',p_part,'campos_permitidos','[]'::jsonb,'obligaciones','[]'::jsonb,'contexto_recurso_huella_sha256',repeat('b',64))::text,'UTF8')
$$;
-- Alta con origen CONVOCA, como la hace la aplicación (rol ejecutor).
CREATE OR REPLACE FUNCTION prueba.origen(p_part text,p_version bigint,p_clave text,p_registrada timestamptz,p_origen text DEFAULT 'convoca',p_hasta timestamptz DEFAULT NULL,p_dia date DEFAULT NULL,p_regla text DEFAULT 'vec.bolsa.reglas:1:b29.contacto_origen_convoca',p_huella text DEFAULT repeat('d',64),p_capacidad text DEFAULT 'capacidad')
RETURNS TABLE(reutilizada boolean,recibo_ref text,version bigint,registrada_en timestamptz) LANGUAGE sql AS $$
 SELECT * FROM vec_bolsa_llamamientos.registrar_datos_contacto_origen_participacion_v1('bolsa:1',p_part,p_version,'clave:kms:prueba',decode('000102030405060708090a0b','hex'),decode('00000000000000000000000000000000ff','hex'),
  'Contacto traído de CONVOCA','per_'||repeat('a',22),p_registrada,p_clave,'recibo:'||p_clave,
  convert_to(p_capacidad,'UTF8'),prueba.decision(p_part),'\x00'::bytea,'\x00'::bytea,1,1,'\x00'::bytea,'\x00'::bytea,'\x00'::bytea,'\x00'::bytea,
  p_origen,coalesce(p_hasta,p_registrada+interval '1 year'),coalesce(p_dia,((p_registrada+interval '1 year') AT TIME ZONE 'Europe/Madrid')::date - 1),p_regla,p_huella)
$$;
-- Alta B4 sin origen (contacto registrado por RRHH o confirmado).
CREATE OR REPLACE FUNCTION prueba.propio(p_part text,p_version bigint,p_clave text,p_registrada timestamptz)
RETURNS TABLE(reutilizada boolean,recibo_ref text,version bigint,registrada_en timestamptz) LANGUAGE sql AS $$
 SELECT * FROM vec_bolsa_llamamientos.registrar_datos_contacto_participacion_v1('bolsa:1',p_part,p_version,'clave:kms:prueba',decode('000102030405060708090a0b','hex'),decode('00000000000000000000000000000000ff','hex'),
  'Contacto confirmado','per_'||repeat('a',22),p_registrada,p_clave,'recibo:'||p_clave,
  convert_to('capacidad','UTF8'),prueba.decision(p_part),'\x00'::bytea,'\x00'::bytea,1,1,'\x00'::bytea,'\x00'::bytea,'\x00'::bytea,'\x00'::bytea)
$$;
CREATE OR REPLACE FUNCTION prueba.espera(p_sql text,p_estado text) RETURNS void LANGUAGE plpgsql AS $$
BEGIN
 EXECUTE p_sql;
 RAISE EXCEPTION 'se esperaba % y no falló: %',p_estado,p_sql;
EXCEPTION WHEN OTHERS THEN
 IF SQLSTATE<>p_estado THEN RAISE EXCEPTION 'se esperaba % y llegó % (%): %',p_estado,SQLSTATE,SQLERRM,p_sql; END IF;
END $$;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA prueba TO vec_bolsa_llamamientos_ejecutor;
