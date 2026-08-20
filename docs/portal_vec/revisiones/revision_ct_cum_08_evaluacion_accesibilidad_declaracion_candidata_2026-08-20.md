# Revisión independiente CT-CUM-08 — evaluación y declaración de accesibilidad candidatas

Fecha de revisión: 20 de agosto de 2026.

## Dictamen

**GO documental**, con `P0=0`, `P1=0` y `P2=0`, para el candidato exacto
`e5477b25461747bf5758f6c543d4e1cf0e279754`, cuyo padre inmediato es
`0fdc31d19865ddb2183eeda958aff5b2d3c826dd`.

El `GO` acredita únicamente coherencia, trazabilidad y cierre semántico del
expediente candidato. No cierra CT-CUM-08, no evalúa una interfaz, no declara
conformidad y no autoriza web, documentos, datos, preproducción, publicación
productiva ni producción.

## Identidad, independencia y alcance

La revisión se ejecutó en:

- worktree exclusivo
  `/srv/fabrica/proyectos/VEC_Diputacion_app/.worktrees/ct-cum-08-accesibilidad-revision-20260820`;
- rama `revision/ct-cum-08-accesibilidad-20260820`;
- candidato `e5477b25461747bf5758f6c543d4e1cf0e279754`;
- base inmediata `0fdc31d19865ddb2183eeda958aff5b2d3c826dd`.

El candidato añade 288 líneas en un solo fichero:

`docs/portal_vec/ct_cum_08_evaluacion_accesibilidad_declaracion_candidata_2026-08-20.md`.

No se modificó el candidato. El único write-set de esta revisión es la
presente acta. El revisor es distinto del productor y no integra, publica ni
altera estado transversal o métricas.

## Autoridades contrastadas

Se leyeron completas las instrucciones de `/srv/fabrica/AGENTS.md` y
`AGENTS.md`, las lecturas obligatorias del repositorio, la matriz normativa,
el expediente y la hoja de ruta de RRHH y los candidatos CT-CUM-02 a
CT-CUM-07.

La revisión confirmó que CT-CUM-08 corresponde a la evaluación de
accesibilidad y su declaración y que continúa bloqueando la publicación
productiva. Las bases documentales citadas existen en Git:

| Documento | Base comprobada |
| --- | --- |
| CT-CUM-08 | `0fdc31d19865ddb2183eeda958aff5b2d3c826dd` |
| CT-CUM-02 | `781bb5891ba3304bfed9d104e71d48846a5b6679` |
| CT-CUM-03 | `475042ba57d63b6abf1810fd26cc65c975496029` |
| CT-CUM-04 | `b2f7a7c0d889c110a765aa9d2d70bab7beeeeabb` |
| CT-CUM-05 | `946256e1682c33327918f2be15df1e9899b331d5` |
| CT-CUM-06 | `002805e3b7c01fd13c85fafd14574144b115ea48` |
| CT-CUM-07 | `74d20d309d4cf4030f8fd15e45caac2c9ab155cd` |

CT-CUM-02 a CT-CUM-07 se conservan como candidatos locales sujetos a
integración y aprobación competentes; el candidato revisado no hereda ni
amplía su autoridad.

## Revisión hostil del contrato

| Propiedad exigida | Resultado reproducido |
| --- | --- |
| Doce familias de superficie | `AX-01` a `AX-12`, exactamente 12, todas `NO_EVALUADO`. Incluyen web interna y pública, formularios, autenticación, flujos, tablas, firma/registro/notificación, documentos, ayuda/errores, multimedia, comunicaciones y clientes alternativos. |
| Cuatro principios | Perceptible, operable, comprensible y robusto están desarrollados por separado, sin resultado positivo. |
| Métodos separados | Distingue alcance, análisis automático, inspección manual de código/semántica, teclado/foco, tecnologías de asistencia, evaluación visual, documentos y pruebas con personas; declara las limitaciones de cada capa. |
| Muestreo | Siete muestras exactas: navegación, formulario, tabla/listado, flujo por pasos, diálogo, documento y ayuda/reclamación; las siete quedan `NO_EVALUADO`. |
| Resultados | No usa `CONFORME`, `PARCIALMENTE_CONFORME` ni `NO_CONFORME` como resultado real. Los estados `EVIDENCIA_NO_APORTADA` y `CRITERIO_PENDIENTE` describen trabajo futuro y no sustituyen el resultado `NO_EVALUADO` de superficies y muestras. |
| Evidencia | Exige versión o artefacto exactos, método, pasos reproducibles, evidencia minimizada, remediación, repetición y revisión independiente. Un verde automático o una captura aislada no acredita conformidad. |
| Hallazgos | Mantiene registro, remediación y verificación separados; un hallazgo no se cierra por presunción, silencio o ausencia de quejas. |
| Excepciones | Todas permanecen `EXCEPCION_NO_APROBADA`; no se aprueba exclusión ni carga desproporcionada. |
| Declaración | Permanece `DECLARACION_NO_PUBLICADA`, sin conformidad determinada, fecha oficial, URL, órgano, contacto o mecanismo activo. |
| Reclamación | Solo define requisitos futuros; no presume correo, teléfono, formulario, órgano ni plazo. |
| Autoridad y bloqueo | Reserva alcance, resultado, excepciones, declaración y publicación a las autoridades competentes. CT-CUM-08 y publicación productiva continúan bloqueados. |
| Datos y realidad operativa | No incorpora URL, contacto, persona, cuenta, expediente, dato, documento, proveedor, credencial, secreto, activo o infraestructura reales. |

No se detectó una vía por la que una API, CLI, MCP o cliente alternativo
amplíe permisos o sea presentado como sustituto accesible por presunción. La
familia `AX-12` exige los mismos permisos y alternativas y prohíbe considerar
equivalente una salida inaccesible.

## Enlaces, huellas y comprobaciones reproducidas

Los nueve enlaces locales del candidato resolvieron a ficheros presentes: la
matriz normativa, CT-CUM-02 a CT-CUM-07, el expediente de RRHH y la hoja de
ruta. `git cat-file -t` devolvió `commit` para el candidato, su padre y las
siete bases documentales indicadas en la tabla anterior.

Huella SHA-256 del fichero candidato:

```text
6a8ae52b244ebb0673a679a254f1a43b0747f9c6058ac15d8bbd16e21c5575f8
```

Comprobaciones ejecutadas y resultado:

```text
git rev-parse HEAD                                  OK: e5477b25461747bf5758f6c543d4e1cf0e279754
git rev-parse HEAD^                                 OK: 0fdc31d19865ddb2183eeda958aff5b2d3c826dd
git diff-tree --no-commit-id --name-only -r HEAD    OK: único fichero candidato
git diff --numstat HEAD^ HEAD                       OK: 288 inserciones, 0 borrados
conteo AX-01..AX-12                                 OK: 12
conteo de principios                               OK: 4
conteo de muestras / NO_EVALUADO                   OK: 7 / 7
resolución de enlaces locales                      OK: 9 / 9
existencia de commits citados                      OK: 9 / 9
búsqueda de URL/credencial/contacto material       OK: sin valores reales
git diff --check HEAD^ HEAD                         OK
```

## Puertas proporcionales y omisiones

No se ejecutaron pruebas automáticas de accesibilidad, tecnologías de
asistencia, revisión visual, pruebas con personas ni evaluación de PDF u
ofimática. No existe en este corte una URL, interfaz, artefacto desplegado,
documento o entorno autorizado que pueda evaluarse; inventar esos resultados
contradiría la invariante `NO_EVALUADO`.

Tampoco se ejecutaron Go, carrera, `go vet`, PostgreSQL, Docker, E2E, red ni
despliegue: el diff es exclusivamente documental y no cambia código,
dependencias, SQL, HTTP o composición. La skill `admin-data-web` no está
disponible y no es necesaria para auditar documentalmente un corte que no
modifica UI.

## Hallazgos

| Severidad | Cantidad | Detalle |
| --- | ---: | --- |
| P0 | 0 | Sin hallazgos. |
| P1 | 0 | Sin hallazgos. |
| P2 | 0 | Sin hallazgos. |

## Límites y siguiente paso

CT-CUM-08 permanece abierto, no evaluado y no aprobado. La declaración sigue
sin publicar; no existe mecanismo activo de comunicación o reclamación y no
hay excepción aprobada. Datos reales, documentos reales y publicación
productiva continúan prohibidos.

El siguiente paso no es publicar: las autoridades competentes deben fijar una
versión y alcance evaluables, combinaciones técnicas, muestra, evaluadores y
criterios; después deben ejecutarse y revisarse las capas automática, manual,
tecnologías de asistencia, visual, documental y con personas. Solo una
decisión formal posterior podrá determinar conformidad, aprobar excepciones y
autorizar una declaración y su mecanismo.

## Entrega

```text
Tarea: CT-CUM-08, revisión independiente documental
Estado: GO documental; CT-CUM-08 continúa abierto y bloqueante
Commit(s): candidato e5477b25461747bf5758f6c543d4e1cf0e279754; acta en commit separado
Archivos modificados: únicamente esta acta de revisión
Resultado: P0=0, P1=0, P2=0
Pruebas ejecutadas: identidad Git, write-set, conteos, enlaces, hashes, búsquedas negativas y git diff --check
Pruebas omitidas y motivo: UI/AT/visual/documentos/personas, Go, PostgreSQL y E2E; alcance exclusivamente documental sin superficie evaluable
Seguridad, privacidad, i18n y accesibilidad: sin datos, secretos, contactos ni infraestructura reales; ninguna conformidad atribuida
Limitaciones: no evaluación, no aprobación, no declaración, no publicación
Riesgos: cualquier uso del GO como cierre o conformidad sería inválido
Siguiente tarea desbloqueada: ninguna publicación; queda pendiente evaluación real y decisión competente
Revisión independiente: GO documental sobre el par exacto e5477b2/0fdc31d
```
