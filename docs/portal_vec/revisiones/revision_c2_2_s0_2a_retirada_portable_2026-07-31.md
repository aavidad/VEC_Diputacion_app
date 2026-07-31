# Revisión C2.2-S0.2a: retirada portable de `000002`

Fecha: 31 de julio de 2026.

## Resultado

Estado del alcance S0.2a: **GO técnico e integración local**.

S0.2 completa no está cerrada. Quedan pendientes el runner de estructura, ACL
y consumidores (S0.2b), el runner de concurrencia, preservación y ciclos
(S0.2c) y su composición (S0.2d). Producción continúa en `NO-GO`.

## Artefactos revisados

La serie integrada es:

| Commit | Resultado |
| --- | --- |
| `9f96bcf` | `down` SQL autónomo y portable, ejecución literal desde `pgx` y primera composición focal. |
| `8adb656` | ampliación del inventario y de la carrera de retirada. |
| `d4c0295` | cierre del TOCTOU sobre relaciones base, dependencias implícitas y regresiones deterministas. |
| `e654312` | acreditación de la clausura TOAST de la tabla de control. |

El commit `5c9ef35` corrige después la composición entre S0.1 y S0.2. Limita
`PGOPTIONS` al `docker exec` que ejecuta `000002.down.sql`; el valor es una
confirmación no secreta y la retirada permanece denegada si falta el GUC. La
revisión final independiente reprodujo la denegación y el rollback, los
runners de S0.1 y otorgantes, Bash, ShellCheck y tamaños, y emitió `GO`,
P0=P1=P2=0. No se atribuye publicación o CI a estos commits locales.

## Hallazgos y correcciones

Las revisiones sucesivas rechazaron correctamente los candidatos anteriores:

1. un `ALTER ... SET STATISTICS` concurrente podía modificar
   `attstattarget` después del inventario y sobrevivir a una retirada
   aceptada;
2. `pg_index.indisclustered` no formaba parte del canon y una preferencia
   `CLUSTER` podía persistir sin detección;
3. una fila de `pg_shdepend` de otra base con el mismo OID provocaba un falso
   rechazo;
4. dependencias de extensión `e/x` y estadísticas extendidas dependientes
   podían escapar del inventario y desaparecer por efecto del `DROP`;
5. el cierre inicial no alcanzaba la tabla y el índice TOAST implícitos de la
   tabla de control, ni todos sus metadatos derivados.

La serie final:

- toma `ACCESS EXCLUSIVE` sobre la tabla de control y las doce relaciones de
  `000001` antes de inventariar;
- mantiene bloqueos `SHARE` sobre los catálogos acreditados hasta el `COMMIT`;
- incorpora la forma completa de los índices, incluido `indisclustered`;
- da ámbito de base a las dependencias compartidas;
- deniega dependencias de extensión y estadísticas extendidas no canónicas;
- recorre la clausura automática e interna que desaparecería con los objetos
  retirados, incluidos TOAST, índices y tipos implícitos;
- conserva `RESTRICT`, confirmación explícita, rollback total y saneamiento o
  descarte de la conexión `pgx`.

CT124, CT125 y CT129 emitieron `GO` técnico sobre el alcance final, sin
hallazgos bloqueantes abiertos en S0.2a.

## Evidencia reproducida

La matriz técnica cubre:

- instalación y retirada normal sobre PostgreSQL 18.4;
- base con ICU `es-ES`;
- ejecución byte a byte del documento SQL mediante `pgx`;
- detector de carreras de Go;
- cancelación durante espera, rollback y reutilización o descarte seguro de la
  conexión;
- DDL concurrente bloqueado hasta finalizar la ventana exclusiva;
- perturbación limpia de OID y base clonada con OID coincidentes;
- `attstattarget`, `CLUSTER` e índice derivado;
- dependencia `pg_shdepend` perteneciente a otra base;
- dependencias de extensión `e/x` y estadísticas extendidas;
- tabla e índice TOAST con ACL, comentarios, opciones, preferencia de
  `CLUSTER`, forma de índice y dependencia de extensión;
- retirada tras restaurar el estado exacto y reinstalación de `000002`;
- Bash, ShellCheck, límites de tamaño, `git diff --check` y búsqueda de
  secretos aplicables al corte.

## Frontera de amenaza

El contrato acredita DDL, ACL, `COMMENT` y dependencias producidos mediante
operaciones soportadas por PostgreSQL 18. La ventana exclusiva impide que
estas modificaciones atraviesen el intervalo entre inventario y retirada. Si
un superusuario contiende y no se obtienen los bloqueos dentro de los límites,
la operación aborta y PostgreSQL revierte todo el cambio.

Una escritura DML directa de superusuario sobre `pg_catalog` equivale a un
compromiso total de la base y queda fuera del contrato. Las defensas
adversariales incluidas son controles puntuales frente a estados reproducidos,
no un canon universal de cualquier manipulación arbitraria de catálogos
internos.

## Decisión de continuación

S0.2b y S0.2c pueden producirse en paralelo, con write-sets disjuntos y un
revisor distinto de cada productor. Solo tras ambos `GO` se implementará
S0.2d. Ninguno de estos cierres internos modifica las métricas oficiales:
Contratación temporal permanece en `24/46` (52 %), la primera vertical en
`5/10` (50 %) y Bolsa productiva en `1/14` (7 %).
