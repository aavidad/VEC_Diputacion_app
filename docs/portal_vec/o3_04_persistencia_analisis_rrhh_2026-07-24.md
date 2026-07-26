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
| Contrato canónico de confirmación | `995656b` | La serialización Go conserva política, sello de fuente, publicación gobernada y pruebas canónicas; PostgreSQL reconstruye y coteja las huellas exactas. |
| Confirmación SQL atómica | `74f0702` | CAS, versión 2, actuación, consumo de decisión y fuentes, auditoría, outbox y recibo en un único `COMMIT`; replay exacto y PostgreSQL 18 verdes. |
| Cierre endurecido V3 | `2834783` | Restaura las migraciones históricas, añade actualizaciones compatibles, liga todos los campos autorizados, prueba registro con RC/coste, rectificación, trece rollbacks, cancelación y cuatro sesiones concurrentes. |

La frontera PostgreSQL confirma análisis y O3-04 está cerrada con
[GO independiente](revisiones/o3_04_revision_independiente_final_2026-07-24.md).
Este corte no habilita producción: O3-05, composición, identidad real y
aceptaciones formales conservan sus propias puertas.

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

## Evidencia superada

El runner
`deploy/postgresql/autorizacion_atestada_v3/probar_integracion_o2_05.sh`
instala las quince migraciones de Contratación y ejecuta la confirmación O3 con
PostgreSQL 18 real en un contenedor efímero sin red ni puertos. Comprueba:

- decisión VEC V3 viva y ligada al contexto exacto;
- pruebas canónicas de RC y coste, con sello, publicación y vínculo exacto;
- transición registrar/rectificar, política, unidad, motivo y segregación;
- historia integral 1–2–3 y puntero CAS;
- actuación, consumos, auditoría encadenada, outbox encadenado y recibo;
- replay exacto sin duplicados y rechazo del replay divergente;
- rollback exacto en los trece puntos de escritura durable;
- cancelación antes de `COMMIT` y carrera real de cuatro sesiones;
- ACL, RLS, inmutabilidad y reversión protegida;
- reinstalación y retirada completa sin historia.

Las pruebas focales Go, con carrera, y `go vet` del dominio, aplicación,
puertos y adaptador PostgreSQL también están verdes.

## Cierre y siguiente incremento

O3-04 supera el recorrido Go → PostgreSQL, replay, cancelación, resultado
ambiguo, reinicio, concurrencia y revisión independiente. Se contabiliza como
la tarea 17 de 46.

El siguiente incremento del camino crítico es O4-03: propuesta y decisión
motivada de la vía de cobertura, incluida la rectificación y los resultados
contradictorios o ausentes.
