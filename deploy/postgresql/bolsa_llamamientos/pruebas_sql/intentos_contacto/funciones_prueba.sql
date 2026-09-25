CREATE SCHEMA IF NOT EXISTS prueba; GRANT USAGE ON SCHEMA prueba TO vec_bolsa_llamamientos_ejecutor;
-- Invocación de la v2 como la hace la aplicación (rol ejecutor).
CREATE OR REPLACE FUNCTION prueba.v2(p_ref text,p_canal text,p_instante timestamptz,p_resultado text,p_clave text,p_max integer,p_sep integer,p_impedir boolean,p_sin text[],p_anotacion text DEFAULT 'Llamada de prueba')
RETURNS TABLE(reutilizado boolean,recibo_ref text,contacto_ref text,previos_sin_contacto integer,previo_contactado boolean,previo_ultimo timestamptz) LANGUAGE sql AS $$
 SELECT * FROM vec_bolsa_llamamientos.registrar_contacto_participacion_v2(p_ref,'bolsa:1','participacion:1','llamamiento:1',p_canal,p_instante,'per_'||repeat('a',22),p_resultado,p_anotacion,p_clave,'recibo:'||p_ref,
  convert_to('capacidad','UTF8'),convert_to('{"principal_id":"per_aaaaaaaaaaaaaaaaaaaaaa","accion":"bolsa.contacto_participacion.registrar","modulo_id":"bolsa","tipo_recurso":"participacion_bolsa","finalidad":"gestion_contactos_participacion","recurso_ref":"participacion:1","campos_permitidos":[],"obligaciones":[]}','UTF8'),
  '\x00'::bytea,'\x00'::bytea,1,1,'\x00'::bytea,'\x00'::bytea,'\x00'::bytea,'\x00'::bytea,p_max,p_sep,p_impedir,p_sin)
$$;
CREATE OR REPLACE FUNCTION prueba.v2b(p_ref text,p_canal text,p_instante timestamptz,p_resultado text,p_clave text,p_max integer,p_sep integer,p_impedir boolean,p_sin text[],p_anotacion text DEFAULT 'Llamada de prueba')
RETURNS TABLE(reutilizado boolean,recibo_ref text,contacto_ref text,previos_sin_contacto integer,previo_contactado boolean,previo_ultimo timestamptz) LANGUAGE sql AS $$
 SELECT * FROM vec_bolsa_llamamientos.registrar_contacto_participacion_v2(p_ref,'bolsa:1','participacion:1','llamamiento:2',p_canal,p_instante,'per_'||repeat('a',22),p_resultado,p_anotacion,p_clave,'recibo:'||p_ref,
  convert_to('capacidad','UTF8'),convert_to('{"principal_id":"per_aaaaaaaaaaaaaaaaaaaaaa","accion":"bolsa.contacto_participacion.registrar","modulo_id":"bolsa","tipo_recurso":"participacion_bolsa","finalidad":"gestion_contactos_participacion","recurso_ref":"participacion:1","campos_permitidos":[],"obligaciones":[]}','UTF8'),
  '\x00'::bytea,'\x00'::bytea,1,1,'\x00'::bytea,'\x00'::bytea,'\x00'::bytea,'\x00'::bytea,p_max,p_sep,p_impedir,p_sin)
$$;
CREATE OR REPLACE FUNCTION prueba.espera(p_sql text,p_estado text) RETURNS void LANGUAGE plpgsql AS $$
BEGIN
 EXECUTE p_sql;
 RAISE EXCEPTION 'se esperaba % y no falló: %',p_estado,p_sql;
EXCEPTION WHEN OTHERS THEN
 IF SQLSTATE<>p_estado THEN RAISE EXCEPTION 'se esperaba % y llegó % (%): %',p_estado,SQLSTATE,SQLERRM,p_sql; END IF;
END $$;
