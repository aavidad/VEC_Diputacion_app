# T12-B — saga de firma de Bolsa en PostgreSQL

**Estado:** candidata implementada y probada; T12 permanece abierto  
**Fecha de corte:** 24/07/2026  
**Ámbito:** firma de decisiones de revisión de baremación

## Resultado

La saga de firma deja de depender exclusivamente del repositorio en memoria.
El adaptador PostgreSQL implementa el puerto existente sin modificar el
núcleo. Conserva versiones inmutables, estado AEAD, arrendamientos con cercado,
auditoría encadenada y outbox transaccional.

Se ha promovido al paquete de puertos una única regla de transición, compartida
por el adaptador de memoria y el PostgreSQL. La base repite la validación
estructural y la matriz de evolución para que una invocación SQL directa no
pueda reescribir identidad, historia, estado protegido ni resultados
confirmados.

## Evidencia ejecutable

El corredor
`deploy/postgresql/bolsa_firma/probar_integracion.sh`:

1. instala roles y migraciones en PostgreSQL 18.4;
2. ejecuta el adaptador Go con una cuenta de mínimo privilegio;
3. crea y reconcilia una petición;
4. reinicia la base y recupera el expediente;
5. acredita exclusión, liberación y cercado creciente;
6. persiste la declaración y la finalización de la preparación de firma;
7. reinicia de nuevo y coteja la versión rehidratada;
8. comprueba tres versiones, siete asientos de auditoría, tres eventos outbox
   y continuidad exacta de la cadena;
9. demuestra ausencia de ACL de tablas y rechazo de una reescritura;
10. revierte las migraciones y elimina los roles sin residuos.

La puerta se ejecuta también en CI. Las pruebas unitarias adicionales fuerzan
respuestas alteradas, JSON ambiguo, cifrado incoherente, transacciones
incorrectas y exposición accidental de la clave.

## Decisiones de diseño

- El JSON contiene metadatos y huellas; el cifrado, potencialmente grande, se
  transporta y almacena como `bytea` separado.
- La base usa su reloj para los arrendamientos y no confía en el reloj del
  cliente.
- El token de 256 bits nunca es serializable. PostgreSQL conserva solo su
  HMAC; el valor opaco vuelve exclusivamente a la réplica propietaria.
- Las escrituras requieren versión esperada, propietario, cercado, caducidad y
  HMAC exactos.
- Guardar no libera implícitamente el arrendamiento: una réplica puede declarar
  y completar varios puntos dentro de su ventana, y lo libera de forma
  explícita.
- La recuperación por idempotencia devuelve la versión inicial exacta; no
  inventa un alta nueva ni mezcla otra intención.

## Lo que no se da por resuelto

T12-B no acredita producción. Permanecen el HSM/KMS, atestación verificable
dentro de la transacción, anclaje externo de auditoría, diario de efectos,
despachador outbox, T13, cifrado y copias operativas, PITR, restauración de
objetos y composición productiva. El detalle y la instalación están en
[`deploy/postgresql/bolsa_firma/README.md`](../../deploy/postgresql/bolsa_firma/README.md).
