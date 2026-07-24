# O3-04 — persistencia del análisis de RRHH

Fecha de corte: 24 de julio de 2026.

## Resultado que debe cerrar la tarea

Registrar o rectificar un análisis de RRHH exige una sola transacción
serializable y autoritativa. El `COMMIT` debe dejar unidos:

1. el consumo único de la decisión VEC V3 todavía vigente;
2. el consumo único del conjunto atestado de fuentes de RC y coste;
3. el CAS sobre la versión integral vigente del expediente;
4. la nueva versión integral del agregado;
5. la actuación funcional exacta;
6. la auditoría encadenada;
7. el recibo verificable;
8. el evento outbox.

Si falta una pieza, falla una validación o se cancela antes del `COMMIT`, no
puede quedar ningún efecto funcional. Un resultado construido antes de
confirmar la transacción nunca se presenta como durable.

## Implementado y probado

| Incremento | Commit | Evidencia |
| --- | --- | --- |
| Historia integral versionada | `a3e0cf5` | Versión 1 materializada desde O2, puntero actual, RLS, ACL, inmutabilidad y reversión protegida. |
| Adaptador de preparación | `b03d427` | Transacción serializable, JSON estricto, reintento limitado, consulta de confirmación y pruebas Go. |
| Reservas durables | `e835280` | HMAC generacional, referencias deterministas, replay, conflicto y prueba PostgreSQL 18. |

Este corte no confirma análisis y no habilita producción.

## Fronteras de autoridad

- PostgreSQL proporciona el reloj y el límite transaccional final.
- VEC conserva una única autoridad de identidad, sesión, rol, política y
  autorización. Contratación no reconstruye permisos desde datos del cliente.
- Las fuentes de RC y coste se consumen por una orden opaca validada dentro de
  la transacción; contratación guarda su procedencia y huella, no duplica sus
  catálogos.
- `expediente_version_integral` es la única historia durable compartida por
  análisis, cobertura y asignación. Ninguna fase crea su propio agregado
  competidor.
- Web, escritorio, CLI y MCP usarán el mismo caso de uso y no aportarán
  autoridad mediante cookies, cabeceras libres o almacenamiento del navegador.

## Siguiente incremento verificable

La migración de confirmación y su adaptador Go deben:

- revalidar con el reloj de base de datos la orden, la autorización y las
  fuentes;
- rechazar autorización revocada, fuente caducada, versión obsoleta,
  reutilización de decisión o de conjunto de fuentes y contenido adulterado;
- devolver en replay exactamente el recibo durable original;
- validar el recibo en Go antes de ejecutar `COMMIT`;
- reintentar únicamente fallos de serialización o interbloqueo;
- probar concurrencia, fallos inyectados, reinicio, ACL, RLS, reversión
  protegida y ausencia de referencias cruzadas entre esquemas.

O3-04 solo cambiará a cerrada tras superar PostgreSQL real, pruebas Go,
revisión independiente y registro del commit en el tablero.
