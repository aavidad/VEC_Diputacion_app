BEGIN;
SELECT actual.cuenta_ref
  FROM vec_identidad_sesiones_v1.alias_hmac_cuenta AS alias
  JOIN vec_identidad_sesiones_v1.estado_cuenta_actual AS actual
    ON actual.cuenta_ref = alias.cuenta_ref
 WHERE alias.esquema_hmac = 'vec.identidad.hmac-sha256.v1'
   AND alias.dominio_hmac_ref = 'idh_aaaaaaaaaaaaaaaaaaaaaaaa'
   AND alias.clave_hmac_id = 'clave-hsm-prueba'
   AND alias.clave_hmac_version = 1
   AND alias.cuenta_id_hmac = decode(repeat('ab', 32), 'hex')
 FOR UPDATE OF actual;
SELECT pg_advisory_xact_lock(817263540118);
-- Trabajo determinista, sin pg_sleep: da tiempo a que la otra transaccion se
-- bloquee y cruce la frontera de frescura mientras esta fila sigue retenida.
SELECT sum(valor) FROM generate_series(1, 20000000) AS valor;
COMMIT;
