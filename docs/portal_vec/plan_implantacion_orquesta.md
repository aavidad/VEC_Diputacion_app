# Plan de implantacion con Orquesta

## Alcance

Este plan convierte la evolucion del portal VEC/portal empleado en runs
pequenas, trazables y ejecutables por Orquesta. No declara el prototipo como
produccion: define el camino de implantacion, los cortes de validacion y la
forma de coordinar agentes sin romper la frontera hexagonal existente.

Refs opacas preservadas: `portal-vec-plan-orquesta-005`,
`worktree-bolsa-vec-portal-005` y `branch-bolsa-vec-portal-005`. Estas refs no
son rutas locales ni nombres Git.

## Principios de ejecucion

- Write-set cerrado por run: cada tarea declara archivos exactos antes de
  arrancar.
- Nucleo neutral: dominio y casos de uso no importan HTTP, Orquesta, DB,
  directorios de usuario, proveedor, credenciales ni rutas locales.
- Puertos pequenos: identidad, repositorio, expediente, firma, notificacion,
  auditoria y documentos entran como contratos minimos.
- Adaptadores opt-in: memoria/fake siguen sirviendo al prototipo; Clave,
  Kerberos/AD, firma, archivo, notificacion o DB solo entran por composicion.
- Configuracion canonica: variables en `config`, documentadas en README, sin
  `os.Getenv` repartidos.
- i18n centralizada: textos visibles y errores via catalogos, con `es` por
  defecto.
- Pruebas de frontera: cada run trae prueba focal y cierre con `go test ./...`
  cuando toque app Go.

## Olas recomendadas

### Ola 0 - Preparacion y corte base

Objetivo: congelar el estado del prototipo, inventario y criterios de
implantacion.

Runs:

- `run-portal-vec-baseline`: validar `go test ./...`, mapa de rutas actuales y
  write-set inicial por vertical.
- `run-portal-vec-backlog`: convertir documentos `docs/portal_vec/*` en tareas
  Txx con area hexagonal, prueba esperada y dependencia.

WaitAgentRefs:

- `wait-agent-ref-baseline-tests`
- `wait-agent-ref-backlog-clasificado`

Cierre: no se programa codigo si el backlog no tiene archivo destino,
dependencia y prueba focal.

### Ola 1 - Nucleo y puertos

Objetivo: completar capacidades de convocatoria, solicitud, expediente y
baremo sin tocar adaptadores productivos.

Runs:

- `run-portal-vec-convocatoria-domain`: estados, reglas versionadas,
  convocatorias y plazos.
- `run-portal-vec-solicitud-domain`: solicitud, alegacion, subsanacion y
  estados auditables.
- `run-portal-vec-expediente-ports`: puertos para documento, expediente,
  auditoria, firma, notificacion y reloj.
- `run-portal-vec-i18n-catalogo`: claves nuevas en catalogo `es`, sin texto
  hardcodeado en handlers.

WaitAgentRefs:

- `wait-agent-ref-domain-green`
- `wait-agent-ref-ports-contracts`
- `wait-agent-ref-i18n-ready`

Cierre: dominio puro con tests; puertos no nombran tecnologias concretas.

### Ola 2 - Adaptadores de prototipo y portal

Objetivo: exponer flujo navegable con adaptadores en memoria/fake y UI
estatica, manteniendo opcion futura de produccion.

Runs:

- `run-portal-vec-memory-adapters`: repositorios en memoria para convocatoria,
  solicitud y expediente.
- `run-portal-vec-http-adapters`: endpoints finos bajo `/api`, envelopes JSON,
  autorizacion por puerto e i18n.
- `run-portal-vec-static-ui`: pantalla de candidato/gestor sobre API estable,
  sin build step salvo decision posterior.
- `run-portal-vec-smoke`: smoke local con `httptest`, demo administrativa y
  navegacion estatica.

WaitAgentRefs:

- `wait-agent-ref-adapters-contract`
- `wait-agent-ref-http-boundary`
- `wait-agent-ref-ui-smoke`

Cierre: handlers no calculan reglas de dominio; UI no duplica mensajes de
negocio si existe clave i18n.

### Ola 3 - Cumplimiento y preparacion productiva

Objetivo: dejar lista la lista de integraciones reales sin activarlas por
defecto.

Runs:

- `run-portal-vec-auth-real-plan`: contrato para Clave/certificado y
  Kerberos/AD, con fake como adaptador de demo.
- `run-portal-vec-storage-plan`: contrato para persistencia duradera,
  migraciones, backup y cifrado, sin DB obligatoria en el prototipo.
- `run-portal-vec-expediente-electronico`: metadatos ENI, huellas, antivirus,
  firma/sello y exportacion reproducible como puertos.
- `run-portal-vec-observabilidad-ens`: logs sin datos sensibles, auditoria,
  correlacion de request y evidencias ENS.
- `run-portal-vec-release-review`: revision cruzada, README, riesgos y
  checklist de no-produccion.

WaitAgentRefs:

- `wait-agent-ref-security-review`
- `wait-agent-ref-storage-boundary`
- `wait-agent-ref-release-ready`

Cierre: no hay secretos ni proveedor concreto en dominio; produccion queda como
decision de composicion y operacion.

## Dependencias entre runs

| Run | Depende de | Entrega |
| --- | --- | --- |
| `run-portal-vec-baseline` | Ninguna | Pruebas base y mapa de rutas actuales. |
| `run-portal-vec-backlog` | `run-portal-vec-baseline` | Tareas Txx con write-set y prueba. |
| `run-portal-vec-convocatoria-domain` | `run-portal-vec-backlog` | Dominio de convocatoria y reglas. |
| `run-portal-vec-solicitud-domain` | `run-portal-vec-convocatoria-domain` | Estados de solicitud, alegacion y subsanacion. |
| `run-portal-vec-expediente-ports` | `run-portal-vec-solicitud-domain` | Puertos neutrales de expediente/documentos. |
| `run-portal-vec-i18n-catalogo` | `run-portal-vec-backlog` | Catalogo de claves del portal. |
| `run-portal-vec-memory-adapters` | Puertos y dominio verdes | Adaptadores en memoria para demo. |
| `run-portal-vec-http-adapters` | Memoria, auth fake, i18n | API estable por casos de uso. |
| `run-portal-vec-static-ui` | HTTP adapters | UI estatica navegable. |
| `run-portal-vec-smoke` | UI y API | Pruebas end-to-end locales del prototipo. |
| `run-portal-vec-release-review` | Todas las anteriores | Riesgos, README y cierre de implantacion. |

## Agentes recomendados

Numero recomendado: 6 agentes principales por burst, con subagentes solo para
revision focal o pruebas acotadas.

| Agente | Rol | Puede usar subagentes |
| --- | --- | --- |
| `agent-backlog` | Clasifica docs, tareas Txx, dependencias y write-set. | 1 subagente revisor de duplicados. |
| `agent-domain` | Dominio, casos de uso y tests puros. | 2 subagentes para frontera y edge cases. |
| `agent-adapters` | HTTP, memoria, auth fake e i18n en composicion. | 2 subagentes para contrato handler/repositorio. |
| `agent-ui` | HTML/CSS/JS estatico y accesibilidad basica. | 1 subagente visual/smoke. |
| `agent-compliance` | ENS, RGPD, expediente, auditoria y riesgos. | 1 subagente de checklist documental. |
| `agent-release` | Integracion, README, pruebas finales y ACK. | 1 subagente de revision final. |

Reglas para subagentes:

- No comparten archivo si no hay turno claro de integracion.
- Devuelven hallazgos y parches sugeridos; el agente propietario integra.
- Cada subagente conserva `parent_agent_ref`, `task_ref`, write-set y prueba
  usada.
- Maximo 6 subagentes activos por tarea amplia; para docs pequenas, ninguno.

## Contrato minimo por run

Cada run de Orquesta debe declarar:

- `run_ref`, `task_ref`, `parent_ref` si aplica y objetivo verificable.
- Write-set exacto, relativo al repo del proyecto.
- Dependencias y `WaitAgentRefs` que desbloquean la siguiente ola.
- Pruebas obligatorias y pruebas focales.
- Criterios de cierre con evidencia compacta.
- Riesgos conocidos y tareas derivadas si falta alcance.

Plantilla compacta:

```text
run_ref: run-portal-vec-<area>
depends_on: [run-portal-vec-...]
wait_for: [wait-agent-ref-...]
write_set: [ruta/exacta.go, ruta/exacta_test.go]
required_tests: [go test ./...]
boundary: nucleo|puerto|adaptador|composicion|i18n|docs
done: pruebas verdes, ACK escrito, refs opacas preservadas
```

## Gates de calidad

- `gate-hexagonal`: dominio sin imports de infraestructura.
- `gate-i18n`: textos nuevos con clave o justificacion.
- `gate-config`: una sola superficie canonica en `config` y README.
- `gate-security`: sin secretos, tokens reales, rutas personales ni datos
  sensibles en logs/tests.
- `gate-tests`: prueba focal de run y `go test ./...` antes de cierre de ola.
- `gate-docs`: README y docs no prometen produccion si solo existe prototipo.
- `gate-ref-only`: contexto no materializado se marca como resuelto por
  evidencia disponible o pendiente sin inventar datos.

## Secuencia operativa

1. Orquesta prepara burst con maximo 6 agentes principales y dependencias.
2. `agent-backlog` genera lista Txx y abre waits de Ola 0.
3. Agentes de dominio trabajan primero; adaptadores esperan
   `wait-agent-ref-domain-green`.
4. Agentes de adaptadores y UI trabajan en paralelo solo tras contratos verdes.
5. `agent-compliance` revisa fronteras de identidad, expediente, logs y
   proteccion de datos antes de release.
6. `agent-release` ejecuta pruebas finales, actualiza docs permitidos y escribe
   ACK con archivos reales tocados.

## Riesgos y mitigacion

| Riesgo | Mitigacion |
| --- | --- |
| Run abre codigo desde backlog narrativo. | Exigir Txx con archivo destino, frontera y prueba. |
| Adaptador productivo contamina dominio. | Gate hexagonal y puertos neutrales. |
| DB o proveedor quedan implicitos. | Memoria por defecto; persistencia real solo por composicion. |
| UI duplica reglas de negocio. | API devuelve estado explicable; UI solo presenta. |
| Contexto ref_only falta. | ACK anota pendiente/resuelto sin inventar contenido. |
| Agentes pisan archivos. | Write-set cerrado, waits y propietario unico por archivo. |

## Cierre esperado

La implantacion queda lista para programacion incremental cuando:

- Todas las olas tienen `WaitAgentRefs` satisfechos.
- Las pruebas obligatorias de cada run aparecen como pasadas en ACK.
- README enlaza el plan y mantiene el estado de prototipo.
- Refs opacas `portal-vec-plan-orquesta-005`,
  `worktree-bolsa-vec-portal-005` y `branch-bolsa-vec-portal-005` se preservan
  como identificadores, no como rutas.
