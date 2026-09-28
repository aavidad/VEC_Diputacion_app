# Recorrido aislado de Auditoría RRHH

`scripts/probar_rrhh_auditoria_recorrido_pg18.sh` exige una preimagen **sintética**
PG18 en formato `pg_dump -Fc` y sus roles de `pg_dumpall --globals-only`. Debe
contener dos hechos o más del expediente CT y de la participación Bolsa elegidos,
AD3-90 y las historias anteriores, sin AD3-91, CT132 ni Bolsa48. Nunca se apunta
al servicio conservado. El script restaura en `postgres:18.4 --rm --network none`
con datos y socket bajo `/dev/shm`, instala las tres migraciones en orden, verifica
ROLLBACK, doble UP y ACL cruzada, y destruye todos los temporales al salir.

Se necesita un hook **privado y ejecutable** en `VEC_AUDITORIA_APP_HOOK`. Recibe
`start` y `stop` y debe arrancar/detener sólo la aplicación aislada conectada al
socket `VEC_AUDITORIA_PG_SOCKET`, con perfiles nominales CT y Bolsa y sus dos
emisores V3. El guion exporta también las referencias exactas que exige la
composición (`VEC_AUDITORIA_CONSULTA_EXPEDIENTE_CT/BOLSA`). El hook debe esperar
a que el servidor HTTPS local esté preparado
antes de retornar de `start`. No forma parte de Git porque su configuración y
certificados son privados. El test exige:

- `VEC_AUDITORIA_HTTP_URL`: HTTPS en loopback, sin usuario ni contraseña;
- `VEC_AUDITORIA_HTTP_CA`, y `VEC_AUDITORIA_HTTP_CERT_*/KEY_*` para `CT` y
  `BOLSA` (rutas de archivos sintéticos). `OPCIONES` usa CT por defecto;
- `VEC_AUDITORIA_DATOS_SINTETICOS=1`, fijado tras comprobar el volcado;
- opcionalmente `VEC_AUDITORIA_HTTP_DESDE/HASTA` (máximo 31 días), y
  `VEC_AUDITORIA_HTTP_RUTA_CONSULTA` si cambia la ruta de composición.

Ejemplo de invocación, con las variables anteriores ya exportadas:

```sh
scripts/probar_rrhh_auditoria_recorrido_pg18.sh \
  /ruta/privada/globals.sql /ruta/privada/base.dump \
  expediente:ct:sintetico participacion:bolsa:sintetica
```

El HTTP compara dos páginas CT y Bolsa, actor, instante, preimagen, postimagen,
motivo, recibo y expediente; comprueba 403 con cada perfil cruzado. Tras detener
aplicación y reiniciar PostgreSQL, vuelve a arrancar y compara las respuestas
íntegras. Seis consultas positivas por ejecución deben dejar seis consumos y
auditorías V3 nuevos, mientras las historias de los dos propietarios mantienen
su cardinalidad. Sin volcado, hook y composición final, el script es una puerta
preparada, no una prueba E2E ejecutada.
