# Relevo T17A PostgreSQL

Estado al cerrar la sesión del 23/07/2026: **fundamento compilable, T17A aún no
implementado**. Este checkpoint no debe integrarse como capacidad terminada.

## Incluido

- contrato local para un protector de staging sustituible y respaldable por
  KMS/HSM;
- sobres cifrados sin datos personales en claro y derivación ciega
  HMAC-SHA-256 del documento enmascarado;
- validación de correspondencia, tamaños, claves versionadas y copias
  defensivas;
- serialización JSON cerrada de incidencias y sobres, con comprobación de la
  huella SHA-256 del contenido cifrado;
- clasificación saneada de errores PostgreSQL, sin propagar mensajes ni
  valores de la base de datos.

## Pendiente para completar T17A

1. `RepositorioImportacionesConvocaPostgreSQL` que implemente
   `GuardarSiAusente` con una sola transacción y CAS concurrente por huella.
2. Migraciones `up/down` dentro de
   `deploy/postgresql/bolsa_importacion_convoca/`: acta minimizada, staging
   cifrado append-only, funciones `SECURITY DEFINER`, roles sin acceso directo
   a tablas y confirmación explícita para el `down`.
3. Validación de identidad/ACL de la cuenta runtime y límites de sesión.
4. Pruebas unitarias del protector hostil, copias defensivas, errores
   saneados, rollback, idempotencia y concurrencia.
5. Prueba de integración y runner efímero con PostgreSQL 18 y TLS
   `verify-full`, incluida ida y vuelta `up/down` y ausencia de residuos.

No se han usado datos reales, credenciales, ficheros Convoca ni cambios fuera
del write-set asignado.
