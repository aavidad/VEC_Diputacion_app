CREATE ROLE cs06l_propietario NOLOGIN;
CREATE ROLE cs06l_lector NOLOGIN;
CREATE SCHEMA cs06l_sintetico AUTHORIZATION cs06l_propietario;
REVOKE ALL ON SCHEMA cs06l_sintetico FROM PUBLIC;

SET ROLE cs06l_propietario;
CREATE FUNCTION cs06l_sintetico.admite_codigo(codigo text)
RETURNS boolean
LANGUAGE sql IMMUTABLE STRICT
AS $$ SELECT codigo = 'sintetico_v1' $$;
REVOKE ALL ON FUNCTION cs06l_sintetico.admite_codigo(text) FROM PUBLIC;

CREATE TABLE cs06l_sintetico.registros (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    codigo text NOT NULL,
    CONSTRAINT codigo_admitido CHECK (cs06l_sintetico.admite_codigo(codigo))
);
INSERT INTO cs06l_sintetico.registros(codigo) VALUES ('sintetico_v1');
GRANT USAGE ON SCHEMA cs06l_sintetico TO cs06l_lector;
GRANT SELECT ON TABLE cs06l_sintetico.registros TO cs06l_lector;
GRANT SELECT ON SEQUENCE cs06l_sintetico.registros_id_seq TO cs06l_lector;
RESET ROLE;
