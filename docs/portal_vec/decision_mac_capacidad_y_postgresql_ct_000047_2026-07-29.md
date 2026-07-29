# Decisión pendiente: MAC de capacidad y verificación PostgreSQL

Fecha: 29 de julio de 2026.

Estado: **NO-GO productivo** hasta decisión conjunta de Sistemas, DBA,
Seguridad y arquitectura.

## Problema

La capacidad VEC-AD-3 actual usa HMAC-SHA-256:

- Go genera un JSON canónico de 37 propiedades;
- la MAC autentica 36 valores, excluido `mac_sha256`;
- cada valor se encuadra con longitud decimal de bytes UTF-8;
- PostgreSQL reconstruye la misma preimagen y verifica la MAC dentro de la
  transacción que consume la capacidad.

El SQL actual conserva `secreto_hmac bytea` y llama a `public.hmac`. Por tanto,
PostgreSQL necesita el secreto simétrico. Esto es incompatible con exigir una
clave HMAC realmente no exportable en KMS/HSM sin una integración adicional
del propio motor de base de datos.

Verificar la MAC solo en Go queda descartado: eliminaría la revalidación final
dentro de la transacción durable y abriría una ventana entre autorización y
efecto.

## Alternativas que deben decidirse

1. **Secreto simétrico en PostgreSQL.** Compatible con el protocolo actual,
   pero no satisface la exigencia de clave no exportable.
2. **HSM accesible desde PostgreSQL.** Mantiene el protocolo, pero exige una
   extensión o integración homologada, disponibilidad transaccional y soporte
   operativo de Sistemas/DBA.
3. **Nueva capacidad asimétrica.** El HSM firma y PostgreSQL conserva solo la
   clave pública. Es la separación criptográfica más limpia, pero requiere
   nueva versión del protocolo, verificador SQL aprobado, migración y nuevos
   vectores.

No se elegirá por conveniencia del código. La decisión debe considerar ENS,
rendimiento, disponibilidad, recuperación, rotación, revocación, licencia,
soporte y operación.

## Trabajo autorizado mientras se decide

Puede avanzarse sin activar producción:

- puerto neutral de cálculo MAC;
- perfil, solicitud y resultado opacos;
- compatibilidad del emisor actual mediante un calculador local solo de
  pruebas;
- separación nominal de cuadro y detalle;
- ciclo de vida, cierre, concurrencia y borrado del material local;
- conservación byte a byte de los vectores Go/PostgreSQL.

El adaptador KMS/HSM real, la configuración de producción y cualquier
declaración de clave no exportable permanecen bloqueados.

## Puertas de la decisión final

La alternativa elegida deberá demostrar:

- verificación dentro de la misma transacción que consume la capacidad;
- claves versionadas, rotación y revocación;
- caída cerrada y semántica de reintento;
- concurrencia y latencia bajo carga;
- ausencia de secretos en tablas, logs, WAL, copias y trazas cuando se declare
  no exportabilidad;
- restauración y continuidad;
- compatibilidad con los 37 campos, o versionado explícito si cambia el
  protocolo;
- revisión formal de Sistemas, DBA, Seguridad y DPD cuando proceda.
