## Lectura obligatoria de especificaciones — 12 de septiembre de 2026

Antes de asignar, implementar, revisar o integrar trabajo, leer
[ESPECIFICACIONES_AGENTES.md](ESPECIFICACIONES_AGENTES.md) y los apartados de
sus fuentes que afecten a la tarea. Dirección debe incluir requisitos concretos
en cada encargo y transmitirlos a todos los subagentes. Esta obligación procede
de la petición expresa del operador; no sustituye las instrucciones vigentes.

# Instrucciones del repositorio para agentes

## Dónde se trabaja y qué sigue vigente de los relevos anteriores

El estado y el plan de cada módulo están en `ESTADO_PROYECTO.md` y en
`docs/plan_modulos/`; las órdenes de dirección que prevalecen, en
`INSTRUCCIONES_DESATASCO.md`. Los relevos operativos de julio a septiembre de
2026 ya no forman parte de este fichero y se consultan en la historia de Git
(`git log -p -- AGENTS.md`).

- Se integra en `main` mediante PR. Cada tarea trabaja en su propia rama y en
  un worktree de `.worktrees/`. El checkout de la raíz del repositorio está
  desfasado respecto a `origin/main` y guarda WIP ajeno: no se programa allí ni
  se toma como referencia de instrucciones o código.
- Una migración ya instalada en una base con historia no se reaplica ni se
  revierte con DOWN; se corrige con una migración nueva.
- O3a (el runner endurecido) sigue en moratoria.
- Un borrador no es un documento firmado, y autenticarse con certificado no es
  firmar: ninguno de los dos se presenta como firma legal.

## Prioridad vigente

La prioridad es el módulo `contrataciontemporal`, basado en el procedimiento
remitido por RRHH. Bolsa no se borra: mantiene convocatorias, candidaturas,
posiciones, reglas y llamamientos.

Si una tarea de contratación temporal necesita una capacidad común de VEC,
Bolsa, Personal, documentos o firma:

1. se define una tarea dependiente y acotada;
2. se implementa en el módulo que posee esa autoridad;
3. se prueba e integra;
4. se vuelve inmediatamente al camino crítico de contratación temporal.

No se amplía otro módulo por conveniencia ni se cambia la prioridad sin
instrucción de dirección.

## Lectura obligatoria

Antes de editar:

1. instrucciones de mayor prioridad del entorno;
2. `AGENTS.md` y `ESPECIFICACIONES_AGENTES.md`;
3. `INSTRUCCIONES_DESATASCO.md` y `ESTADO_PROYECTO.md`;
4. el plan del módulo en `docs/plan_modulos/` cuando la tarea sea de un módulo;
5. la especificación y la matriz normativa que enlacen esos documentos.

Un agente recibe un identificador de tarea. No toma otra por iniciativa propia.


## Trabajo paralelo

- Rama y worktree exclusivos fuera de `/tmp`.
- Write-set declarado y disjunto.
- Un agente productor no integra ni da el visto bueno a su propio trabajo.
- El revisor reproduce pruebas y emite `GO` o `NO-GO`.
- Solo dirección modifica los documentos transversales de estado durante la
  integración.
- No fusionar, rebasar ni limpiar ramas ajenas.
- Commits pequeños con el identificador funcional, pruebas y documentación.

## Granularidad obligatoria

Todo VEP se desarrolla mediante minitareas, también fuera de contratación
temporal:

- una minitarea tiene una sola responsabilidad observable y un único criterio
  de cierre;
- un agente recibe una sola minitarea y no amplía su alcance por iniciativa
  propia;
- código y pruebas forman un commit local autónomo; la evidencia de revisión y
  el estado transversal se confirman inmediatamente después por integración;
- como señal de alarma, una tarea que modifica más de tres ficheros de
  producción, añade más de unas doscientas líneas productivas o necesita dos
  motivos distintos en el mensaje se divide o justifica antes de programarse;
- cada commit debe compilar y superar sus pruebas focales; no se admiten cortes
  intermedios que dejen una API pública incompleta, una migración sin consumidor
  coherente o una ruta registrada sin autoridad;
- una capacidad grande se expresa como un grafo de minitareas con dependencias,
  no como un identificador que acumula contratos, adaptadores, composición,
  interfaz y E2E;
- las piezas independientes se asignan a subagentes con write-sets disjuntos;
  otro agente revisa el resultado antes de que dirección lo integre;
- si dos minitareas necesitan el mismo fichero, se ejecutan en secuencia o se
  separa primero una abstracción propietaria; nunca se resuelve permitiendo
  edición concurrente del mismo archivo.

Dividir no significa producir commits rotos o cambios cosméticos sin valor. La
unidad mínima es una capacidad compilable, protegida y verificable de extremo a
extremo dentro de su alcance.

## Arquitectura no negociable

- Hexagonal estricta.
- `domain` no importa aplicación, puertos, adaptadores, HTTP, SQL ni
  proveedores.
- `application` coordina dominio y puertos; no conoce HTTP, PostgreSQL, Docker
  ni SDK concretos.
- `ports` contiene contratos mínimos y neutrales; no implementaciones.
- `adapters` traduce proveedores y transportes sin redefinir reglas de negocio.
- Ningún módulo lee o escribe tablas de otro módulo.
- Los intercambios usan referencias opacas, comandos, eventos e
  outbox/inbox.
- El dominio y los casos de uso son neutrales al cliente. Web, escritorio,
  CLI y MCP consumen las mismas capacidades de aplicación mediante
  adaptadores distintos; ninguna regla funcional vive en HTTP, DOM o una
  sesión de navegador.
- Base de datos, almacenamiento, firma, antivirus, identidad, comunicaciones,
  documentos, calendarios y sistemas externos son adaptadores intercambiables.
- Fases, opciones, plantillas, formatos y reglas funcionales se gobiernan por
  catálogos versionados. Solo las invariantes técnicas quedan compiladas.
- No crear una segunda autoridad de identidad, roles, permisos, auditoría,
  i18n o temas.

## Seguridad y protección de datos

- Denegación predeterminada y privilegio mínimo.
- La identidad y el perfil proceden de una frontera confiable; nunca de JSON,
  cookies, parámetros, cabeceras libres o datos del navegador.
- Cookies, `localStorage` y `sessionStorage` no son autoridades ni mecanismos
  de sesión. La composición real no depende de ellos. Los clientes usan
  credenciales breves, ligadas al emisor y verificadas por la API; escritorio
  puede emplear certificado/mTLS y Kerberos corporativo mediante conectores.
- La API rechaza por defecto `Cookie`, credenciales persistidas por el
  navegador y cualquier cabecera de identidad que no esté atestada por la
  frontera confiable. Ninguna respuesta real emite `Set-Cookie`.
- Operaciones internas sensibles exigen garantía alta y superficie interna o
  administrativa coherente.
- Una decisión positiva queda ligada a acción, recurso, finalidad, ámbito,
  motivo, actor, perfil, correlación y vigencia exactos.
- Todo efecto consume la autorización dentro de la misma transacción que
  escribe estado, auditoría y outbox.
- Idempotencia semántica, control optimista de versión e historia de solo
  adición.
- Ningún secreto, certificado, clave, token, DSN, dato personal real o ruta
  privada entra en Git, fixtures, errores o logs.
- Datos minimizados y referencias opacas en listados, auditoría y eventos.
- Cifrado en tránsito y reposo; claves fuera del proceso y rotables.
- Límites de tamaño, tiempo, profundidad y cardinalidad antes de reservar
  memoria o llamar conectores.
- La indisponibilidad nunca se interpreta como autorización, validación,
  firma, entrega o éxito.
- No usar datos reales hasta cerrar EIPD, categorización ENS, análisis de
  riesgos, registro de actividades y autorizaciones formales.
- IA aplicada a selección, empleo, baremación, evaluación o asignación queda
  cerrada hasta completar la clasificación y obligaciones del Reglamento de
  IA. El bot público solo accede a corpus público gobernado.

La matriz vigente del módulo es
`docs/portal_vec/matriz_normativa_contratacion_temporal_2026-07-23.md`.

## i18n, idioma y presentación

- Código de dominio en castellano coherente, salvo términos técnicos
  universalmente adoptados y convenciones de Go.
- No mezclar castellano e inglés en el mismo vocabulario.
- Todo texto visible, mensaje de validación, estado, ayuda, documento y
  notificación usa claves i18n.
- i18n puro (orden del operador del 29 de septiembre de 2026): los textos
  viven en catálogos de datos por idioma (`web/static/textos/<idioma>/<módulo>.json`),
  nunca en diccionarios dentro de ficheros `.js` o `.go`. El código no nombra
  idiomas concretos: los idiomas disponibles y el de respaldo salen del índice
  `web/static/textos/idiomas.json`. Añadir un idioma es copiar una carpeta y
  traducirla, sin tocar código.
- Fechas, números, moneda, zonas horarias y plurales se formatean por
  localización.
- El tema común es la autoridad visual; ningún módulo duplica CSS estructural.
- Accesibilidad desde diseño: teclado, foco, contraste, zoom, lector de
  pantalla, estados alternativos y documentos descargables accesibles.

## Herramientas obligatorias en VEC (orden de Alberto, 29/09/2026)

Todo agente (Claude, subagente o Codex) usa SIEMPRE las skills, agentes y plugins instalados en el momento que les toca, y los nombra en cada encargo que delega:
- Pantallas, textos visibles o flujos: skills `usabilidad-vec`, `aspecto-vec`, `disenar-sistema-visual-vec` e `impeccable` (su `VEC-PRIORIDAD.md`) antes de programar; revisión de usabilidad independiente antes de fusionar.
- Textos que lee una persona (RRHH, Alberto, ayudas, documentos, `dudas.md`, correos): skill `humanizer` (su `VEC-USO.md`) antes de entregarlos.
- SQL: skill `revisar-sql-vec` / `ensayar-sql` y revisión SQL independiente antes de fusionar, con ensayo en el clon de la principal.
- Cambios y PR: `revisar-cambios-vec`, `documentar-entregar-vec`, `pr-vec`; backend con `programar-backend-vec` y `persistir-autorizar-vec`; interfaz con `programar-interfaz-vec`; recorridos con `probar-recorridos-vec`.
- Buscar código: primero el índice (codebase-memory-mcp); grep solo para texto y configuración.
- Navegador y capturas: Playwright con el Chrome del sistema.
- i18n puro: ningún texto ni idioma dentro del código; textos en catálogos de datos por idioma.
- Modelo: el más barato que baste (Sonnet para lo mecánico; Opus para diseño, SQL, seguridad y revisiones).
Codex lee las skills de `.agents/skills/` del repositorio y Claude, de `~/.claude/skills/`.
`ensayar-sql` y `pr-vec` solo existen en `~/.claude/skills/`: en Codex ese paso lo hace dirección.
`security-audit` está en `~/.claude/skills/` y en `skills/` de cada CODEX_HOME.

## Calidad

### Rendimiento instantáneo — orden del operador, 6 de octubre de 2026

La aplicación debe responder de forma instantánea en todos los módulos y
soportar miles de personas conectadas a la vez. Es requisito de aceptación, no
una mejora.

- Objetivo: cada lectura responde en el servidor en menos de 300 ms (p95) con
  volumen de datos real; la pantalla útil aparece en menos de 1 s.
- Toda PR que añada o cambie una ruta aporta una medición con volumen realista
  (`EXPLAIN ANALYZE` y tiempo de la ruta) o una prueba que fije el número de
  consultas.
- Prohibido el N+1: ni una consulta ni un consumo de autorización por fila. Se
  trabaja en lotes, con índices y con transacciones cortas.
- El portal muestra primero lo que ya tiene y carga cada módulo por separado,
  sin que uno lento bloquee a los demás.
- Antes de cada hito, prueba de carga local a cientos y miles de usuarios
  concurrentes, con percentiles y errores.
- La velocidad nunca se consigue quitando autorización, auditoría ni otras
  comprobaciones.

### Validación eficiente — orden del operador, 24 de septiembre de 2026

- Durante la edición, ejecutar solo comprobaciones focales proporcionales al
  cambio: formato, prueba del paquete o fichero afectado, `git diff --check` y
  ensayo PostgreSQL desechable únicamente si se modifica su contrato. No lanzar
  `go test -race ./...` ni `scripts/verificar_calidad.sh` por cada ajuste pequeño.
- Congelar el candidato y pasar `scripts/verificar_calidad.sh` una vez sobre el
  hash final antes del PR o la integración, conforme a la puerta vigente. Si
  una revisión detecta `NO-GO`, detener la puerta de ese hash; corregir y validar
  el candidato nuevo. No repetir una campaña verde sin cambio relevante o fallo
  nuevo. Ningún encargo se cierra con su puerta exigida en rojo.
- Ejecutar como máximo una puerta completa local a la vez y coordinarla con
  PostgreSQL y CI para no superar dos campañas intensivas globales. La batería
  `go test -race ./...`, en especial `internal/vec/ports`, usa CPU. Go, Node y
  PostgreSQL no obtienen una aceleración útil trasladando esas pruebas a la GPU;
  reducir campañas y usar pruebas focales es la medida de rendimiento aplicable.

- Código documentado cuando la intención o el contrato no sean obvios.
- Sin adaptadores ficticios en la composición real.
- No declarar E2E, producción o cumplimiento por tener una pantalla o una
  prueba aislada.
- Tamaño de fichero: objetivo de 500 líneas y aviso a partir de 800 conforme a
  DEC-051. Por decisión del operador del 23 de septiembre de 2026 **no es un
  tope duro**: sirve para detectar ficheros que crecen sin justificación, no
  para obligar a trocear. **Un fichero cohesionado de 1.000 líneas es preferible
  a tres de 400 que solo se entienden juntos**; se parte cuando hay dos
  responsabilidades de verdad, no para cumplir un número. Al superar el aviso se
  congela el tamaño actual —en `scripts/tamano_ficheros_base.txt` o en el tope
  de la prueba que lo mida— y se justifica en una línea del mensaje del commit.
  Ningún corte de producto se detiene ni se reestructura por esta puerta.
- Sin funciones monolíticas, estados globales mutables ni errores que filtren
  detalles internos.
- Contextos y cancelación en todas las fronteras lentas.
- Copias defensivas al cruzar límites mutables.
- Dependencias externas solo si están mantenidas, son compatibles en licencia
  y reducen riesgo real.

Puertas mínimas:

```text
gofmt
go test del paquete afectado
go test -race del paquete afectado
go vet del paquete afectado
git diff --check
```

Sobre el hash final, antes del PR o la integración, se pasa una vez
`scripts/verificar_calidad.sh`, que ya incluye `go test ./...`, `go test -race ./...`,
`go vet ./...` y la batería web. PostgreSQL se
prueba en instancia efímera real con roles, ACL, concurrencia, reintento y
reversión protegida. HTTP/web exige contrato, seguridad, accesibilidad y
revisión visual.

## Entrega

```text
Tarea:
Estado: GO / NO-GO / bloqueada
Commit(s):
Archivos modificados:
Resultado:
Pruebas ejecutadas:
Pruebas omitidas y motivo:
Seguridad, privacidad, i18n y accesibilidad:
Limitaciones:
Riesgos:
Siguiente tarea desbloqueada:
Revisión independiente:
```

La documentación, pruebas y código se entregan juntos. El agente no cambia una
tarea a cerrada ni actualiza el porcentaje; lo hace dirección tras verificar e
integrar el commit.
