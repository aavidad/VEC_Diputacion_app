\set ON_ERROR_STOP on
-- Preimagen focal aislada: dependencias mínimas con las firmas productivas.
CREATE ROLE vec_bolsa_llamamientos_propietario NOLOGIN;
CREATE ROLE vec_bolsa_llamamientos_ejecutor NOLOGIN;
CREATE SCHEMA vec_bolsa_llamamientos AUTHORIZATION vec_bolsa_llamamientos_propietario;
CREATE SCHEMA vec_contratacion_temporal;
CREATE SCHEMA vec_autorizacion_atestada_v3;
CREATE SCHEMA prueba_reincorporacion;
GRANT USAGE ON SCHEMA vec_bolsa_llamamientos TO vec_bolsa_llamamientos_ejecutor;

CREATE TABLE prueba_reincorporacion.origen (
 evento_ref text PRIMARY KEY, huella text NOT NULL, posicion bigint NOT NULL,
 expediente_ref text NOT NULL, relacion_ref text NOT NULL, fecha_efectiva date NOT NULL,
 recibo_ct_ref text NOT NULL, cese_evento_ref text NOT NULL, cese_recibo_ref text NOT NULL
);
CREATE FUNCTION vec_contratacion_temporal.verificar_reincorporacion_publicada_bolsa_v1(text,text,bigint)
RETURNS TABLE(evento_ref text, expediente_ref text, relacion_ref text, fecha_efectiva date,
 recibo_ct_ref text, cese_evento_ref text, cese_recibo_ref text)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT o.evento_ref,o.expediente_ref,o.relacion_ref,o.fecha_efectiva,o.recibo_ct_ref,o.cese_evento_ref,o.cese_recibo_ref
 FROM prueba_reincorporacion.origen o WHERE o.evento_ref=$1 AND o.huella=$2 AND o.posicion=$3
$f$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.verificar_reincorporacion_publicada_bolsa_v1(text,text,bigint) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_contratacion_temporal TO vec_bolsa_llamamientos_propietario;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.verificar_reincorporacion_publicada_bolsa_v1(text,text,bigint)
 TO vec_bolsa_llamamientos_propietario;

SET ROLE vec_bolsa_llamamientos_propietario;
CREATE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion() RETURNS trigger LANGUAGE plpgsql AS $f$
BEGIN RAISE EXCEPTION 'historia inmutable' USING ERRCODE='55000'; END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion() FROM PUBLIC;
CREATE TABLE vec_bolsa_llamamientos.vinculo_candidato(participacion_ref text PRIMARY KEY,candidato_ref text NOT NULL);
CREATE TABLE vec_bolsa_llamamientos.politica_cese_bolsa(version bigint PRIMARY KEY,catalogo_sha256 text NOT NULL);
CREATE TABLE vec_bolsa_llamamientos.restriccion_cese_bolsa(
 evento_ref text PRIMARY KEY,origen_ref text NOT NULL UNIQUE,candidato_ref text NOT NULL,relacion_ref text NOT NULL,
 recibo_ct_ref text NOT NULL,fecha_efecto date NOT NULL,disponible_desde date NOT NULL,
 politica_version bigint NOT NULL REFERENCES vec_bolsa_llamamientos.politica_cese_bolsa(version));
RESET ROLE;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_situacion_participacion_v3_atestada(
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
RETURNS TABLE(efecto_ref text,consumo_nuevo boolean,huella_efecto_sha256 text)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT convert_from($7,'UTF8'),true,convert_from($8,'UTF8')
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_situacion_participacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_bolsa_llamamientos_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_situacion_participacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_bolsa_llamamientos_propietario;

SET ROLE vec_bolsa_llamamientos_propietario;
INSERT INTO vec_bolsa_llamamientos.vinculo_candidato VALUES ('participacion:uno','can_1234567890123456789012');
INSERT INTO vec_bolsa_llamamientos.politica_cese_bolsa VALUES (1,repeat('a',64));
INSERT INTO vec_bolsa_llamamientos.restriccion_cese_bolsa VALUES
 ('evento:ct:contrato-bolsa:'||repeat('0',64),'evento:ct:cese:uno',
  'can_1234567890123456789012','relacion:uno','recibo:ct:cese:uno',DATE '2026-09-01',DATE '2027-02-01',1);
RESET ROLE;
INSERT INTO prueba_reincorporacion.origen VALUES
 ('evento:ct:reincorporacion:uno',repeat('1',64),10,'expediente:uno','relacion:uno',DATE '2026-09-01',
  'recibo:ct:retorno:uno','evento:ct:cese:uno','recibo:ct:cese:uno'),
 ('evento:ct:reincorporacion:dos',repeat('2',64),11,'expediente:dos','relacion:dos',DATE '2026-09-02',
  'recibo:ct:retorno:dos','evento:ct:cese:dos','recibo:ct:cese:dos'),
 ('evento:ct:reincorporacion:tres',repeat('3',64),12,'expediente:tres','relacion:otra',DATE '2026-09-01',
  'recibo:ct:retorno:tres','evento:ct:cese:uno','recibo:ct:cese:uno');
