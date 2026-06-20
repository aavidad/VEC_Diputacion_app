# Autoprogramacion Orquesta pendientes 2026-05-23

## Alcance

- Tarea origen preservada: `task-ref-self-improvement-fba55e3296a0`.
- Tarea de ejecucion documental: `task-autoprogramming-fba55e3296a0-g01`.
- Refs opacas preservadas sin convertirlas en rutas ni ramas Git:
  `worktree-ref-orquesta-server-idle-self-improvement` y
  `branch-ref-orquesta-server-idle-self-improvement`.
- Correccion acotada: antes de programar cambios de codigo, el backlog de
  automejora degradado debe pasar por scanner documental focal.

## Evidencia revisada

- `agent_packet.json`: contexto requerido `ref_only` presente con
  `required_ref_action=ack_evidence_required`; no trae materializacion
  adicional, asi que la evidencia se limita a lectura local y busquedas.
- `README.md`: la app Bolsa declara arquitectura hexagonal, i18n centralizada y
  estado de prototipo; no justifica abrir codigo para este contrato documental.
- `orquesta_spec_nucleo.json`: hay backlog de tareas de app Bolsa y
  `autoprogramming_request`; sirve como ejemplo de tareas con write-set y tests,
  no como backlog especifico de esta automejora.
- `docs/portal_vec/plan_implantacion_orquesta.md`: ya contiene el gate
  `gate-ref-only` y el cierre que impide programar codigo desde backlog sin
  archivo destino, dependencia y prueba focal.
- Busquedas focales locales: `autoprogram`, `self-improvement`, `backlog`,
  `rail`, `duplic`, `outbox`, `idle`,
  `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT` y
  `task-ref-self-improvement`.
- Validacion 2026-06-19: no hay `AGENTS.md` local; la lectura disponible es
  `README.md`, `orquesta_spec_nucleo.json`, este write-set y
  `docs/portal_vec/plan_implantacion_orquesta.md`.
- Decision `ref_only`: resuelto para esta pasada documental porque el paquete
  aporta la accion requerida y los docs locales aportan gates suficientes; no
  se infiere contenido ausente ni se programa codigo.
- Cierre causal 2026-06-19: la busqueda focal se limita a rutas existentes
  (`docs`, `cmd`, `internal`, specs JSON y README); la ausencia de `scripts`
  no desbloquea codigo ni amplia el write-set.

## Regla general corregida

No se debe abrir trabajo de codigo desde texto narrativo, entradas ya visibles
en cola, errores de rail sin clasificar o duplicaciones aparentes. El scanner
documental produce primero una lista pequeña de pendientes Txx con:

- `origen`: paquete, spec, doc local o rail observado.
- `estado`: nuevo, ya_en_cola, narrativo, duplicado, pendiente_evidencia.
- `area_hexagonal`: nucleo, puerto, adaptador, composicion, i18n o docs.
- `accion`: documentar, fusionar, aplazar, programar o pedir revision.
- `evidencia`: ref compacta, busqueda usada o archivo local.

## Scanner obligatorio antes de codigo

1. Leer paquete, `README.md`, `AGENTS.md` si existe, docs locales relevantes y
   el write-set de la run.
2. Ejecutar busquedas focales sobre `docs`, `cmd`, `internal`, `scripts`, specs
   y variantes locales con `backlog`, `autoprogram`, `self-improvement`,
   `rail`, `duplic`, `outbox`, `idle` y refs de tarea.
3. Marcar cada hallazgo como `narrativo`, `ya_en_cola`,
   `outbox_pendiente`, `duplicado`, `hueco_real` o
   `pendiente_evidencia`.
4. Para `hueco_real`, exigir frontera hexagonal, archivo destino en write-set
   futuro y prueba focal. Si falta cualquiera, queda como pendiente documental.
5. Cerrar con ACK que declare contexto `ref_only` resuelto o pendiente sin
   inventar datos.

## Pendientes Txx

| ID | Pendiente | Estado | Accion |
| --- | --- | --- | --- |
| T01 | Filtrar secciones narrativas del backlog antes de proponer codigo. | nuevo | Mantener como regla de scanner; no programar desde narrativa sin objetivo verificable. |
| T02 | Saltar tareas ya visibles en cola/publicacion outbox. | pendiente_evidencia | Exigir evidencia de cola/outbox antes de preparar run nueva. |
| T03 | Distinguir outbox pendiente frente a hueco real de backlog. | pendiente_evidencia | Registrar criterio en scanner y no deduplicar solo por texto parecido. |
| T04 | Detectar huecos reales de nucleo/director/adaptador antes de abrir codigo. | nuevo | Clasificar por frontera hexagonal y requerir archivo destino dentro del write-set de la run futura. |
| T05 | Convertir rail errors observados en causas generales, no parches de caso unico. | nuevo | Enlazar con `docs/rail_errors_observados_2026-05-23.md`. |
| T06 | Fusionar duplicaciones de rail/backlog con una clave estable. | nuevo | Enlazar con `docs/duplicaciones_railes_pendientes_2026-05-24.md`. |
| T07 | Validar contexto `ref_only` antes de decidir codigo. | nuevo | Resolver con lectura local/paquete/director; si no basta, anotar pendiente y no inventar contenido. |

## Criterio para programacion futura

Una tarea futura puede pasar de scanner documental a codigo solo si cumple todo:

- Hay archivo destino dentro de su write-set.
- Hay causa general reproducible.
- La frontera hexagonal esta clara: nucleo neutral, puerto pequeno, adaptador
  opt-in o configuracion de composicion.
- Existe prueba focal razonable o evidencia documental suficiente.
- El contexto `ref_only` queda resuelto o marcado pendiente sin inventar datos.
- Las refs opacas de worktree y branch se conservan como identificadores, no
  como rutas ni nombres Git.

## Resultado de esta pasada

No se programa codigo. El patron general corregido queda como gate documental:
scanner focal primero, clasificacion Txx despues, y solo entonces una run futura
con write-set minimo si aparece hueco real verificable.

La pasada `task-autoprogramming-fba55e3296a0-g01` queda cerrada como
documental: preserva refs opacas, resuelve el contexto `ref_only` por evidencia
local disponible y no abre adaptadores, puertos ni nucleo nuevo.
