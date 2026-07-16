# Auditoría de diseño, estructura y seguridad (2026-07-16)

Auditoría de solo lectura sobre la rama `vec-orquesta-20260619` (HEAD
`30eb3df`). Este documento tiene dos partes: los hallazgos verificados y las
**directrices vinculantes** que todo agente (humano o automático) debe seguir
mientras el plan de remediación no se complete.

## Método

Todo lo afirmado se verificó con comandos reproducibles, no leyendo la
documentación: grafo de imports con `go list -f`, métricas con `wc`/`grep`,
revisión de `Dockerfile`, `docker-compose.yml`, `internal/app/server`,
`web/static` y barrido de secretos sobre ficheros trackeados.

## Fortalezas verificadas

- La regla hexagonal se cumple: ningún `domain` importa `ports`,
  `application` ni `adapters`; ningún `ports` importa `application` ni
  `adapters`; ningún módulo importa a otro módulo.
- La allowlist CIDR se aplica sobre `RemoteAddr` del socket
  (`internal/app/server/server.go`), no sobre `X-Forwarded-For`: no es
  falsificable desde el cliente. `trusted_headers` valida el remoto antes de
  leer cabeceras.
- Servidor con `ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout`,
  `IdleTimeout`, `MaxHeaderBytes` y `MaxBytesReader`.
- `Dockerfile` con imágenes fijadas por digest, usuario no root en build y
  runtime, `COPY` cerrado por directorio y binario estático `-trimpath`.
- Compose con `read_only`, `cap_drop`, `no-new-privileges`, límites de
  pids/memoria y red interna sin puertos del host para la API.
- Sin secretos hardcodeados en ficheros trackeados; datos de demostración
  sintéticos y rotulados; material de estudio con datos personales excluido
  de Git y de la imagen.
- Frontend autocontenido: cero URLs externas; la exportación de rutas a OSM
  está anulada a propósito para que las coordenadas no salgan de la red
  corporativa (`web/static/app.js`, `openStreetMapDirectionsURL`).

## Hallazgos de diseño y estructura

### H-01 — `ports` como segundo dominio (prioridad alta)

`internal/vec/ports`: 19.733 líneas de fuente y 1.917 funciones.
`internal/modules/bolsa/ports`: 9.818 líneas y 886 funciones. La capa de
contratos concentra lógica pesada (canonización, HMAC, validación), con
ficheros de test de más de 3.000 líneas y ~40 s de ejecución. Consecuencias:
fan-in enorme, recompilación en cascada y contratos difíciles de auditar.

Remediación: dejar en `ports` únicamente interfaces y tipos de intercambio;
extraer la lógica canónica y criptográfica a subpaquetes propios (por
ejemplo `internal/vec/canonico`, `internal/modules/bolsa/internal/idempotencia`)
y trocear los contratos por capacidad (autorización, documental, almacén,
auditoría).

### H-02 — `httpapi` actúa como segundo composition root (prioridad alta)

`internal/vec/adapters/httpapi` importa directamente los cinco módulos,
incluidos sus `application` y sus `adapters/file` y `adapters/memory`
(`workspace.go`, `cronos.go`, `personal_rpt.go`). Existe
`ports.ModuleRegistryStore` y el
[contrato de módulos](contrato_modulos_vec.md), pero este cableado lo
puentea: un módulo no se puede desenchufar sin tocar el adaptador HTTP.

Remediación: mover la composición a `internal/app/bootstrap`; `httpapi`
debe recibir handlers o interfaces ya montados y no importar
`internal/modules/*`.

### H-03 — Frontend monolítico (prioridad media)

`web/static/app.js` tiene 13.211 líneas sin módulos ES ni proceso de build.
Ya exigió una limpieza de código muerto de −2.458 líneas. Remediación:
partir por módulo funcional antes de conectar la UI privada real.

### H-04 — Doble núcleo de Bolsa (decisión tomada: retirar ya)

`internal/candidate` (heredado, en inglés) convive con
`internal/modules/bolsa` (nuevo, en español). Verificado con el grafo de
imports: el único consumidor restante es `internal/app/bootstrap` (la API
heredada que solo se monta en modo `fake`); el módulo nuevo de Bolsa ya no
depende del núcleo heredado.

**Decisión del responsable (2026-07-16): el núcleo heredado se retira
portando primero al formato nuevo lo que siga haciendo falta.** Retirar no
es borrar sin más; el proceso obligatorio es:

1. **Inventario** de capacidades del núcleo heredado: candidatos, méritos,
   documentos, alegaciones, avisos, auditoría, manifiesto operacional, la
   API bajo `/api` (`/api/portal`, `/api/demo`, …), la autenticación `fake`
   heredada y la persistencia `file`/`local_durable`.
2. **Análisis de brecha** contra `internal/modules/bolsa`: qué está ya
   cubierto por el módulo nuevo, qué falta y se necesita, y qué se descarta
   expresamente (con constancia en el registro de decisiones).
3. **Portar lo necesario** al formato nuevo: español, hexagonal, fallo
   cerrado, autorización por caso de uso y ficheros de 500 líneas como
   máximo, con sus tests.
4. **Solo entonces** eliminar `internal/candidate/**`, su cableado en
   `internal/app/bootstrap` y los restos de configuración que solo existían
   para la API heredada.

Mientras tanto, ningún agente debe añadir código, tests ni documentación
nuevos que dependan de `internal/candidate`: el heredado queda congelado en
solo-mantenimiento hasta completar el porte.

### H-05 — Asimetría de módulos (prioridad baja)

Bolsa está completo (domain/ports/application/adapters); Cronos y Personal
son parciales; Dietas y Administración son solo manifiesto. El contrato de
módulos debería declarar el nivel de madurez de cada módulo para que el
shell no asuma capacidades inexistentes.

### H-06 — Ficheros que agotan el contexto de un agente (decisión tomada)

**Decisión del responsable (2026-07-16): ningún fichero de código debe
superar aproximadamente 500 líneas.** El motivo es operativo: este proyecto
se desarrolla con agentes cuyo contexto es finito; un fichero de miles de
líneas impide trabajarlo con seguridad.

Inventario a fecha de auditoría: 50 ficheros Go de fuente y 49 de test
superan el límite, además de `web/static/app.js` (13.211 líneas). Los
peores:

| Líneas | Fichero |
| --- | --- |
| 13.211 | `web/static/app.js` |
| 4.215 | `internal/modules/bolsa/ports/idempotencia_semantica_baremacion.go` |
| 4.122 | `internal/vec/ports/ejecuciones_documentales_v3.go` |
| 2.723 | `internal/vec/domain/pagos.go` |
| 2.235 | `internal/modules/bolsa/application/baremacion.go` |
| 2.185 | `internal/modules/bolsa/ports/baremacion.go` |
| 1.754 | `internal/vec/ports/recibo_escritura_objeto_material_v2.go` |
| 1.716 | `internal/vec/ports/almacen_objetos.go` |

Remediación: los ficheros nuevos cumplen el límite desde ya (directriz 9);
los existentes se reducen al tocarlos y dentro de los trabajos H-01 y H-03.
Partir un fichero Go en varios del mismo paquete no cambia la API ni el
comportamiento: es la vía preferida y segura.

El límite se hace cumplir en la puerta de calidad:
`scripts/comprobar_tamano_ficheros.sh` falla si un fichero de código supera
las 500 líneas por encima de su línea base congelada
(`scripts/tamano_ficheros_base.txt`). La línea base solo puede menguar; no
se regenera para dar cabida a crecimiento nuevo.

## Hallazgos de seguridad

### S-01 — Keylog TLS en el puesto de trabajo (urgente, fuera del repo)

Existía `SSLKEYLOGFILE=.ssl-key.log` exportado en el `~/.bashrc` del puesto
de desarrollo, con ruta relativa: cualquier proceso lanzado desde el
directorio del proyecto volcaba los secretos de sesión TLS 1.3 a
`.ssl-key.log` en la raíz del repositorio (113 KB acumulados). El fichero
estaba en `.gitignore` y nunca se publicó, pero combinado con una captura
de tráfico permite descifrar sesiones reales. Corregido el 2026-07-16:
variable eliminada y fichero borrado. Ningún keylog debe volver a residir
dentro del árbol del proyecto.

### S-02 — Sin integración continua (alta)

No existía `.github/workflows/`; la puerta `scripts/verificar_calidad.sh`
dependía de la disciplina manual y `govulncheck` no estaba instalado en el
puesto. Corregido el 2026-07-16 con un workflow que ejecuta la puerta
canónica completa en cada push y pull request.

### S-03 — Perfil sin TLS ni identidad real (conocida, pendiente productivo)

Ya documentado en el README: el perfil actual descansa en loopback y red
interna. El fallo cerrado existente (rechazo de `fake` fuera de loopback)
mitiga. No desplegar este perfil fuera del puesto local.

### S-04 — CSRF para el futuro adaptador de identidad (nota de diseño)

La API heredada usa token portador sin cookies, así que hoy no aplica. Si
la identidad real termina en cookie de sesión, serán obligatorios
`SameSite` y tokens anti-CSRF. Debe decidirse al diseñar el adaptador de
aserciones protegidas, no después.

## Directrices vinculantes para agentes

Mientras las remediaciones H-01 y H-02 no estén ejecutadas y en verde:

1. **No añadir lógica nueva a `internal/vec/ports` ni a
   `internal/modules/bolsa/ports`.** Los contratos nuevos se limitan a
   interfaces y tipos de intercambio; toda derivación canónica,
   criptográfica o de validación va a un subpaquete de dominio o interno.
2. **`internal/vec/adapters/httpapi` no puede ganar nuevos imports de
   `internal/modules/*`.** Cualquier cableado nuevo de módulos se hace en
   `internal/app/bootstrap`.
3. **`web/static/app.js` no debe crecer.** Funcionalidad nueva de frontend
   va en ficheros propios servidos junto a `app.js`.
4. **No commitear en rojo.** `go test ./...` debe pasar antes de cada
   commit. El corte en curso de baremación (idempotencia semántica y flujo
   durable de firma) debe cerrar su test de no-filtración antes de abrir
   frentes nuevos.
5. **Mensajes de commit en español**, en el estilo del repositorio
   (`area: verbo en infinitivo ...`). Prohibidos los prefijos
   `feat:`/`fix:`/`chore:` y cualquier atribución de herramientas de IA
   (`Co-Authored-By`, "Generated with", etc.). El histórico ya fue
   normalizado el 2026-07-16; no debe volver a desviarse.
6. **Prohibido reescribir historia o hacer force push** en
   `vec-orquesta-20260619` sin coordinación humana expresa.
7. **Commits con rutas explícitas** (`git add <fichero>...`): hay varios
   procesos trabajando sobre el mismo árbol y un `git add -A` puede
   arrastrar trabajo ajeno a medias.
8. **Nunca commitear** `deploy/osrm-granada/data/`, keylogs, material de
   estudio (`Baremador/`, `Bolsa_Diputacion*/`, `convoca_dipgra/`,
   `cegid_peoplenet_aapp/`, `fotos/`) ni catálogos generados; ya están en
   `.gitignore` y deben seguir estándolo.
9. **Ningún fichero nuevo de código puede superar 500 líneas** (fuente,
   test, JS o script). Un fichero existente que ya supere el límite no
   puede crecer: antes de ampliarlo hay que trocearlo (en Go, dividir en
   varios ficheros del mismo paquete conserva API y comportamiento). El
   objetivo es que ningún fichero agote por sí solo el contexto de un
   agente.
10. **Nada nuevo sobre `internal/candidate`**: el núcleo heredado está en
    solo-mantenimiento hasta completar el porte descrito en H-04.

## Plan de remediación

| Ref | Acción | Estado | Responsable sugerido |
| --- | --- | --- | --- |
| S-01 | Eliminar `SSLKEYLOGFILE` y el keylog | Hecho 2026-07-16 | — |
| S-02 | CI con la puerta de calidad canónica | Hecho 2026-07-16 | — |
| H-01 | Extraer lógica de `ports` a subpaquetes | Pendiente | Agente, tras cerrar el WIP de baremación |
| H-02 | Mover cableado de módulos de `httpapi` a `bootstrap` | Pendiente | Agente, en rama aislada |
| H-03 | Partir `web/static/app.js` por módulos | Pendiente | Agente, antes de conectar UI privada |
| H-04 | Inventario y análisis de brecha del núcleo heredado | Pendiente | Agente |
| H-04 | Portar al módulo nuevo lo necesario y retirar `internal/candidate` | Pendiente, tras la brecha | Agente |
| H-05 | Declarar nivel de madurez por módulo en el contrato | Pendiente | Agente |
| H-06 | Límite de 500 líneas: exigido a ficheros nuevos; reducir los existentes al tocarlos | En vigor desde 2026-07-16 | Todos |
| S-04 | Decidir estrategia CSRF con el adaptador de identidad | Pendiente | Decisión humana (DEC en registro) |
