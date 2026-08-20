# Revisión independiente CT-CUM-09 — clasificación y gobierno candidatos de IA

Fecha de revisión: 20 de agosto de 2026.

## Dictamen

**GO documental**, con `P0=0`, `P1=0` y `P2=0`, para el candidato exacto
`ef80ddce760d0712f5be5a2af97ae553ec43ef74`, cuyo padre inmediato es
`e5477b25461747bf5758f6c543d4e1cf0e279754`.

El `GO` acredita únicamente coherencia, trazabilidad y cierre predeterminado
del dossier candidato. No cierra CT-CUM-09, no clasifica jurídicamente un
sistema, no habilita IA ni autoriza pruebas, datos, proveedores, modelos,
infraestructura, decisiones, efectos, preproducción o producción.

## Identidad, independencia y alcance

La revisión se ejecutó en:

- worktree exclusivo
  `/srv/fabrica/proyectos/VEC_Diputacion_app/.worktrees/ct-cum-09-ia-revision-20260820`;
- rama `revision/ct-cum-09-ia-20260820`;
- candidato `ef80ddce760d0712f5be5a2af97ae553ec43ef74`;
- base inmediata `e5477b25461747bf5758f6c543d4e1cf0e279754`.

El candidato añade 237 líneas en un solo fichero:

`docs/portal_vec/ct_cum_09_clasificacion_gobierno_casos_uso_ia_2026-08-20.md`.

No se modificó el candidato. El único write-set de esta revisión es la
presente acta. El revisor es distinto del productor y no integra, publica,
activa capacidades ni altera estado transversal o métricas.

## Autoridades contrastadas

Se leyeron completas las instrucciones de `/srv/fabrica/AGENTS.md` y
`AGENTS.md`, las lecturas obligatorias del repositorio, la matriz normativa,
el expediente y la hoja de ruta de RRHH y los candidatos CT-CUM-02 a
CT-CUM-08.

La matriz normativa exige clasificar y gobernar cada caso de uso de IA y
bloquea cualquiera que no sea meramente informativo público. Además trata
como potencialmente de alto riesgo la IA destinada a contratación, selección,
evaluación, promoción, asignación de tareas o supervisión laboral. El bot
informativo solo podría usar corpus público gobernado, sin datos internos ni
decisiones sobre personas.

Las bases documentales comprobadas existen como objetos Git:

| Documento | Base comprobada |
| --- | --- |
| CT-CUM-09 | `e5477b25461747bf5758f6c543d4e1cf0e279754` |
| CT-CUM-02 | `781bb5891ba3304bfed9d104e71d48846a5b6679` |
| CT-CUM-03 | `475042ba57d63b6abf1810fd26cc65c975496029` |
| CT-CUM-04 | `b2f7a7c0d889c110a765aa9d2d70bab7beeeeabb` |
| CT-CUM-05 | `946256e1682c33327918f2be15df1e9899b331d5` |
| CT-CUM-06 | `002805e3b7c01fd13c85fafd14574144b115ea48` |
| CT-CUM-07 | `74d20d309d4cf4030f8fd15e45caac2c9ab155cd` |
| CT-CUM-08 | `0fdc31d19865ddb2183eeda958aff5b2d3c826dd` |

Todos esos documentos conservan su estado candidato y sus aprobaciones
pendientes. CT-CUM-09 no hereda ni amplía su autoridad.

## Revisión hostil del contrato

| Propiedad exigida | Resultado reproducido |
| --- | --- |
| Estado global | El dossier declara `CANDIDATO_LOCAL_NO_APROBADO` e `IA_CERRADA`; la falta o incertidumbre nunca autorizan, puntúan, priorizan ni equivalen a éxito. |
| Diez clases | `IA-01` a `IA-10`, exactamente diez y sin duplicados; las diez terminan en `IA_CERRADA`. |
| Caso público | `IA-01` solo describe un candidato informativo sobre corpus público gobernado; también queda cerrado y exige procedencia, límites, accesibilidad y canal humano. |
| Empleo y selección | Ordenación, baremación, evaluación, cobertura, asignación y supervisión se señalan como `POTENCIAL_ALTO_RIESGO` y permanecen cerradas. Los demás casos incompletos quedan pendientes de clasificación, no rebajados ni habilitados. |
| Diez puertas | `GOV-01` a `GOV-10`, exactamente diez; las diez mantienen `EVIDENCIA_NO_APORTADA` y son acumulativas cuando proceda. |
| Frontera determinista/IA | Distingue reglas, plantillas, búsquedas y transformaciones deterministas de aprendizaje, inferencia, perfilado, clasificación, generación o recomendación. Exige evidencia técnica y no acepta la etiqueta comercial como prueba. |
| Canales | Web, API, CLI, MCP o integración no cambian la clasificación, no amplían permisos ni convierten propuesta o probabilidad en hecho. |
| Ficha por caso | Exige finalidad, contexto, datos, salidas, roles, clasificación, riesgos, supervisión, transparencia, seguridad, registros, evaluaciones y aprobaciones; una ficha incompleta falla cerrada. |
| Supervisión humana | Exige competencia, autoridad, tiempo, contexto, formación, facultad de detener o rechazar, motivación propia y ausencia de confirmación inducida. Una firma mecánica no basta. |
| Cambio y retirada | Finalidad, datos, población, modelo, proveedor, versión, umbral, integración o canal reabren la clasificación; fallo o anomalía detienen el uso. |
| Datos y realidad operativa | Declara cero datos, expedientes, candidaturas, puestos, historiales, prompts, respuestas o corpus reales y no identifica proveedor, modelo, dataset, endpoint, infraestructura, marca o contrato. |
| Autoridad y bloqueos | No acepta riesgos ni excepciones, no afirma controles implantados y conserva CT-CUM-09, CT-CUM-10, aprobaciones competentes, cualquier IA real y producción como bloqueos. |

La clasificación provisional no se usa para resolver efectos. Los casos cuya
finalidad concreta aún falta se mantienen `PENDIENTE_CLASIFICACION` y
`IA_CERRADA`; una finalidad, población, entrada, salida, modelo o canal nuevos
exigen identificador y evaluación propios.

## Enlaces, huellas y comprobaciones reproducidas

Los ocho enlaces locales del candidato resolvieron a ficheros presentes: la
matriz normativa, CT-CUM-02 a CT-CUM-08. `git cat-file -t` devolvió `commit`
para el candidato, su padre y las ocho bases documentales indicadas.

Huella SHA-256 del fichero candidato:

```text
657a8f9aeacc75fe53ef47ee74f6ceba61736b1bf1bb0a6c97365562a0e4840a
```

Comprobaciones ejecutadas y resultado:

```text
git rev-parse HEAD                                  OK: ef80ddce760d0712f5be5a2af97ae553ec43ef74
git rev-parse HEAD^                                 OK: e5477b25461747bf5758f6c543d4e1cf0e279754
git diff-tree --no-commit-id --name-status -r HEAD  OK: alta de un único Markdown
git diff --numstat HEAD^ HEAD                       OK: 237 inserciones, 0 borrados
conteo IA-01..IA-10 / IA_CERRADA                    OK: 10 / 10
conteo GOV-01..GOV-10 / EVIDENCIA_NO_APORTADA       OK: 10 / 10
resolución de enlaces locales                       OK: 8 / 8
existencia de candidato, padre y bases              OK: 10 / 10 objetos commit
búsqueda de URL, contacto, credencial o secreto     OK: sin valores reales
git diff --check HEAD^ HEAD                         OK
```

## Puertas proporcionales y omisiones

No se ejecutaron modelos, inferencias, entrenamientos, evaluaciones, corpus,
pruebas con personas, análisis de sesgo o interfaces. El dossier mantiene todo
caso en `IA_CERRADA` y no existe un sistema autorizado sobre el que esas
pruebas pudieran producir evidencia válida.

Tampoco se ejecutaron Go, carrera, `go vet`, PostgreSQL, Docker, E2E, red ni
despliegue: el diff es exclusivamente documental y no cambia código,
dependencias, SQL, HTTP, composición o datos. Repetir puertas técnicas ajenas
al predicado no acreditaría clasificación, gobierno ni aprobación.

## Hallazgos

| Severidad | Cantidad | Detalle |
| --- | ---: | --- |
| P0 | 0 | Sin hallazgos. |
| P1 | 0 | Sin hallazgos. |
| P2 | 0 | Sin hallazgos. |

## Límites y siguiente paso

CT-CUM-09 permanece abierto y no aprobado. Toda IA continúa cerrada, incluido
el caso informativo público. No existe clasificación jurídica final, sistema
de riesgos y calidad, gobierno de datos, documentación técnica, supervisión
operativa, evaluación, registro, aprobación o control implantado.

El siguiente paso no es implementar: las autoridades competentes deben
definir y clasificar un caso exacto, sus roles, finalidad, población, datos,
salidas y obligaciones; después deben resolver las diez puertas aplicables con
evidencia independiente. CT-CUM-10 conserva la auditoría conjunta y producción
continúa bloqueada.

## Entrega

```text
Tarea: CT-CUM-09, revisión independiente documental
Estado: GO documental; CT-CUM-09 continúa abierto y bloqueante
Commit(s): candidato ef80ddce760d0712f5be5a2af97ae553ec43ef74; acta en commit separado
Archivos modificados: únicamente esta acta de revisión
Resultado: P0=0, P1=0, P2=0
Pruebas ejecutadas: identidad Git, genealogía, write-set, cardinalidades, enlaces, hashes, búsquedas negativas y git diff --check
Pruebas omitidas y motivo: IA, personas, Go, PostgreSQL y E2E; alcance exclusivamente documental sin sistema autorizado
Seguridad, privacidad, i18n y accesibilidad: cero datos, secretos, contactos, proveedor, modelo o infraestructura reales; IA y accesibilidad permanecen cerradas
Limitaciones: no clasificación final, no aprobación, no control implantado, no IA, no efecto, no publicación
Riesgos: cualquier uso del GO como cierre, conformidad o habilitación sería inválido
Siguiente tarea desbloqueada: ninguna implementación; queda pendiente decisión competente y CT-CUM-10
Revisión independiente: GO documental sobre el par exacto ef80ddc/e5477b2
```
