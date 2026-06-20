# Duplicaciones railes pendientes 2026-05-24

## Alcance

Lista de duplicaciones probables que deben resolverse antes de lanzar nuevas
runs de autoprogramacion. Documento acotado a diagnostico y reglas de fusion.

## Clave estable de deduplicacion

Usar clave compuesta:

`task_ref_origen + area_hexagonal + archivo_destino + causa_general + prueba_focal`

No usar solo titulo ni texto parecido. Dos entradas con el mismo sintoma pueden
ser distintas si afectan fronteras distintas; dos textos distintos pueden ser
la misma tarea si comparten causa, archivo y prueba.

## Duplicaciones pendientes

| ID | Patron duplicado | Riesgo | Resolucion |
| --- | --- | --- | --- |
| D01 | "idle self-improvement" repetido como backlog, criterio y rail error. | Tres runs para una sola causa. | Mantener una tarea canonica y enlazar criterios como evidencia. |
| D02 | "outbox pendiente" y "tarea ya visible en cola" tratados como huecos separados. | Trabajo duplicado o preparacion de run innecesaria. | Fusionar bajo estado `ya_en_cola/outbox_pendiente`; solo programar si hay hueco real. |
| D03 | "filtrar narrativa" repetido en docs y criterios de cierre. | Parches documentales sin prueba de frontera. | Convertir en regla previa del scanner, no en cambio de codigo automatico. |
| D04 | "rail error observado" y "duplicacion rail" sin causa compartida. | Dos documentos divergen y pierden causalidad. | Enlazar Rxx con Dxx y exigir causa general comun antes de preparar run. |
| D05 | "nucleo/director/adaptador" citado sin archivo destino. | Se abre alcance demasiado amplio. | Bloquear programacion hasta tener write-set concreto. |
| D06 | Refs de worktree/branch repetidas como si fueran rutas Git. | Exploracion fuera de alcance o confusion de aislamiento. | Mantenerlas opacas y no incluirlas en la clave salvo como evidencia. |
| D07 | Contexto `ref_only` repetido entre paquete, gate y rail error. | La falta de materializacion se resuelve varias veces con notas distintas. | Una unica decision por run: `contexto_ref_only_resuelto` si hay evidencia local suficiente, o `contexto_ref_only_pendiente` si no. |
| D08 | Pruebas documentales repetidas sin receipt diferenciado. | Se pierde causalidad entre scanner y ACK. | Mantener una evidencia por prueba obligatoria y referenciarla de forma compacta. |

## Politica antes de programar

- Si una entrada queda duplicada, fusionar evidencia y conservar el ID canonico.
- Si el write-set no incluye el archivo necesario, no editar fuera; anotar
  faltante.
- Si afecta nucleo, mantener dominio neutral sin HTTP, MCP, Codex, OPES, DB,
  proveedor, credenciales ni rutas locales.
- Si afecta adaptador, hacerlo opt-in y con puerto pequeno.
- Si afecta configuracion, centralizarla en composicion canonica.

## Decision canonica de fusion

Para esta revision, la tarea canonica es
`task-ref-self-improvement-fba55e3296a0`. Las menciones a idle
self-improvement, backlog degradado, rail error, outbox pendiente y contexto
`ref_only` son evidencias del mismo scanner previo, no tareas independientes.

Una run futura solo debe separarlas si aporta una clave distinta completa:
frontera hexagonal distinta, archivo destino distinto, causa general distinta y
prueba focal distinta. Si solo cambia el texto narrativo, se fusiona.

## Estado de esta revision

La fusion fisica se limita a estos tres documentos del write-set. Hay docs
locales utiles (`README.md`, `orquesta_spec_nucleo.json` y
`docs/portal_vec/plan_implantacion_orquesta.md`), pero no hay evidencia de una
cola/outbox viva consultable dentro del contrato. Quedan reglas y pendientes
preparados para que una run futura deduplique sin ampliar alcance.

Para `task-autoprogramming-fba55e3296a0-g01`, la decision canonica se mantiene:
fusionar narrativa, rail error, duplicacion y `ref_only` en un solo scanner
documental. La ausencia de cola/outbox viva y de materializacion adicional no
se interpreta como hueco real de codigo.
