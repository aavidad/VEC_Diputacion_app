# Persistencia PostgreSQL de la saga de firma de Bolsa

Este directorio implementa la candidata **T12-B** del puerto
`RepositorioFlujosFirmaBaremacion`. No sustituye al dominio ni añade una
dependencia de PostgreSQL al núcleo: es un adaptador hexagonal intercambiable.

## Qué queda probado

- PostgreSQL 18.4 fijado por digest.
- Cinco fachadas `SECURITY DEFINER`; la cuenta ejecutora no puede acceder a
  tablas, secuencias, tipos fila ni helpers.
- Roles `NOLOGIN`, mínimo privilegio, RLS habilitado y forzado, `search_path`
  cerrado y transacciones `SERIALIZABLE`.
- Estado de trabajo AEAD separado del JSON, identificado por SHA-256 y sin
  copiar el cifrado en auditoría ni outbox.
- Versiones append-only, puntero actual protegido por clave foránea diferida,
  índice de idempotencia único y transición validada tanto en Go como en SQL.
- Arrendamiento con token opaco, HMAC en base de datos, reloj de PostgreSQL,
  caducidad máxima de cinco minutos y cercado monotónico.
- Auditoría append-only encadenada y outbox en el mismo `COMMIT` que cada
  versión.
- Reinicio real entre creación, continuación y lectura final; reconciliación
  de alta, exclusión de un segundo trabajador, dos transiciones, liberación
  idempotente, ACL negativas, intento de reescritura y migraciones `down`.

La prueba no usa datos personales ni secretos persistentes:

```bash
bash deploy/postgresql/bolsa_firma/probar_integracion.sh
```

Las claves y contraseñas del corredor se generan en memoria para un contenedor
efímero. La imagen puede sustituirse mediante `VEC_POSTGRES_TEST_IMAGE`, pero
en CI se conserva el digest aprobado.

## Instalación

Orden, con una identidad DBA controlada:

1. `roles_up.sql`;
2. `migraciones/000001_almacen_flujo_firma.up.sql`;
3. `migraciones/000002_operaciones_flujo_firma.up.sql`;
4. crear fuera del repositorio el `LOGIN` de aplicación y concederle solo
   `vec_bolsa_firma_ejecutor`.

La aplicación inyecta:

- un `VerificadorEstadoFlujoFirmaBaremacion` real;
- una clave HMAC de 256 bits para comprometer tokens de arrendamiento;
- un pool exclusivo cuya identidad pertenezca solo al grupo ejecutor.

La clave se captura en una operación privada no reflectible y nunca se
formatea, serializa ni persiste. Rotarla invalida, de forma cerrada, los
arrendamientos aún vivos; la ventana máxima es de cinco minutos. Sistemas debe
evitar el registro de parámetros en PostgreSQL, proxy, APM y trazas.

Las reversiones exigen literales deliberados en `PGOPTIONS` y destruyen los
datos. Solo se usan en el corredor o en un procedimiento de cambio autorizado:

- `REVERTIR_OPERACIONES_FIRMA_BOLSA`;
- `REVERTIR_ALMACEN_FIRMA_BOLSA`.

## Límites: NO-GO productivo

Este corte es una candidata técnica, no una autorización para tratar
expedientes reales:

1. el constructor PostgreSQL recibe todavía la clave HMAC en el proceso; falta
   el proveedor HSM/KMS no exportable y su política de rotación;
2. PostgreSQL valida estructura, hashes y transición, pero no puede verificar
   por sí solo el sello HMAC del expediente. Un proceso ejecutor totalmente
   comprometido podría invocar la fachada sin pasar por el verificador Go;
   falta una atestación criptográfica consumible y verificable en la
   transacción;
3. la cadena SHA-256 detecta alteraciones ordinarias, pero no resiste a una
   administración de base hostil sin anclaje externo monotónico;
4. outbox aún no tiene despachador productivo ni política operativa de replay;
5. faltan el diario durable/idempotencia de los efectos externos de Autofirma,
   custodia, retención y confirmación;
6. faltan T13 (registro de lecturas con finalidad), cifrado de volumen y
   copias, WAL/PITR, repositorio externo, RPO/RTO y restauración conjunta con
   el almacén de objetos;
7. el adaptador todavía no está compuesto en el perfil productivo de la
   aplicación.

T12 permanece abierto. Ninguna cuenta de producción debe recibir
`vec_bolsa_firma_ejecutor` hasta cerrar estos puntos y obtener las
conformidades de Sistemas, Seguridad y DPD.
