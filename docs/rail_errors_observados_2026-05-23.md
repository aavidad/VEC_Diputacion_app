# Rail errors observados 2026-05-23

## Alcance

Registro documental de errores de rail que degradan automejora antes de
programar. No modifica codigo ni abre adaptadores nuevos.

Refs de esta revision: `task-ref-self-improvement-fba55e3296a0`,
`task-autoprogramming-fba55e3296a0-g01`,
`worktree-ref-orquesta-server-idle-self-improvement` y
`branch-ref-orquesta-server-idle-self-improvement`. Las refs de worktree y
branch son opacas.

## Observaciones

| ID | Error observado | Impacto | Regla de correccion |
| --- | --- | --- | --- |
| R01 | Backlog narrativo tratado como tarea programable. | Crea runs sin write-set claro o sin prueba focal. | Exigir ID Txx, archivo destino y criterio verificable. |
| R02 | Falta de contexto materializado `ref_only`. | Riesgo de inventar causa o ampliar alcance. | Usar lectura local/paquete como evidencia y anotar `contexto_ref_only_resuelto` o `contexto_ref_only_pendiente`. |
| R03 | Entradas de cola/outbox mezcladas con huecos reales. | Duplica trabajo o pisa avances de otros agentes. | Antes de preparar run, clasificar `ya_en_cola`, `outbox_pendiente` o `hueco_real`. |
| R04 | Error puntual convertido en parche local. | No corrige patron repetible. | Registrar causa general y frontera hexagonal antes de tocar codigo. |
| R05 | Write-set insuficiente para corregir causa. | Tentacion de editar fuera de contrato. | Conservar avance documental y dejar faltante como nota/tarea derivada. |
| R06 | Refs opacas tratadas como rutas o ramas Git. | Se consulta o edita fuera del contrato. | Preservar refs como identificadores y no usarlas para filesystem ni Git. |
| R07 | Prueba obligatoria declarada como pasada sin evidencia focal. | ACK causal debil y rework posterior. | Cada prueba obligatoria debe enlazar lectura o busqueda local y receipt compacto. |

## Scanner focal recomendado

1. Buscar en docs/spec/cmd/internal/scripts por terminos: `backlog`,
   `autoprogram`, `self-improvement`, `rail`, `duplic`, `outbox`, `idle`.
2. Separar hallazgos en narrativos, cola/outbox, duplicados y huecos reales.
3. Para huecos reales, asignar frontera: nucleo, puerto, adaptador,
   composicion o i18n.
4. Si falta evidencia requerida, no programar; dejar tarea derivada.
5. Si la evidencia basta, generar run futura con write-set minimo y prueba
   focal.

## Matriz de cierre

| Rail | Pendiente vinculado | Duplicacion vinculada | Cierre documental |
| --- | --- | --- | --- |
| R01 | T01 | D03 | Backlog narrativo queda como regla de scanner, no run de codigo. |
| R02 | T07 | D07 | `ref_only` se resuelve por paquete y lectura local o se declara pendiente. |
| R03 | T02/T03 | D02 | Cola/outbox no se considera hueco hasta tener evidencia operacional. |
| R04 | T04/T05 | D01/D04 | Solo causa general con frontera clara puede generar programacion futura. |
| R05 | T04 | D05 | Si falta write-set, no se edita fuera; se deja tarea derivada. |
| R06 | T07 | D06 | Worktree y branch se conservan como refs opacas. |
| R07 | T07 | D07 | ACK incluye receipts compactos de las pruebas obligatorias. |

## Evidencia de esta pasada

- No hay `AGENTS.md` local en el workdir; si aparece en una run futura, debe
  leerse antes de tocar archivos.
- Si hay `README.md` y docs locales. `README.md` confirma prototipo, frontera
  hexagonal e i18n; `docs/portal_vec/plan_implantacion_orquesta.md` confirma
  gates para backlog, write-set, pruebas y `ref_only`.
- `orquesta_spec_nucleo.json` define backlog de app Bolsa, no backlog
  especifico de automejora.
- El paquete de Orquesta aporta los criterios de automejora y contexto
  `ref_only`; se usa como evidencia primaria.
- Busqueda focal local encontro coincidencias de `backlog`, `outbox`,
  `autoprogram`, `rail`, `duplic`, `idle`, `task-ref-self-improvement` y
  `ref_only`; no encontro codigo nuevo que deba editarse bajo este write-set.
- La lectura local basta para el gate `ref_only` de esta pasada porque no se
  decide codigo: el faltante de materializacion queda compensado por paquete,
  docs y scanner; no se consulta director ni API externa.
- La busqueda se ejecuta sobre rutas existentes; si `scripts` no existe, el
  rail no falla por alcance ausente y conserva la limitacion como evidencia
  documental, no como permiso para crear o editar fuera del write-set.

## Resultado

Los rail errors quedan convertidos en reglas de scanner. Cualquier cambio de
codigo posterior debe nacer de un hueco real clasificado, no de texto narrativo
ni de duplicaciones pendientes.

El cierre de `task-autoprogramming-fba55e3296a0-g01` es documental y no crea
efectos externos: `ref_only` queda resuelto por lectura local y paquete, con
receipts compactos en el ACK de Orquesta.
