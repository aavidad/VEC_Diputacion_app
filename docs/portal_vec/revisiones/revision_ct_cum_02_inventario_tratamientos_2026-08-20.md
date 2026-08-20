# Revisión independiente CT-CUM-02 — inventario de tratamientos

Fecha: 20 de agosto de 2026.

## Dictamen

**GO documental**, con `P0=0`, `P1=0` y `P2=0`, para integrar únicamente el
[inventario de tratamientos de contratación temporal](../inventario_tratamientos_contratacion_temporal_2026-08-20.md)
como inventario candidato sujeto a validación organizativa posterior.

Este dictamen no aprueba bases jurídicas, destinatarios, accesos, plazos de
conservación, transferencias, encargados, RAT, EIPD, categorización ENS,
riesgos, competencia, accesibilidad, usos de IA, datos reales, efectos
jurídicos, preproducción ni producción. Tampoco cambia tareas, métricas o
estado transversal alguno.

## Objeto y genealogía revisados

| Elemento | Valor comprobado |
| --- | --- |
| Candidato | `475042ba57d63b6abf1810fd26cc65c975496029` |
| Padre inmediato | `781bb5891ba3304bfed9d104e71d48846a5b6679` |
| Árbol del candidato | `979e3908ca013cdba59bcfffeabf4fbea8d1cd56` |
| Asunto | `docs(CT-CUM-02): inventaría tratamientos CT` |
| Write-set material | Un único fichero nuevo: `docs/portal_vec/inventario_tratamientos_contratacion_temporal_2026-08-20.md` |
| Líneas del inventario | `273` |
| Blob Git del inventario | `7b89ca4b7356a712da6f7aa6b1bda60ca1f7a03b` |
| SHA-256 del inventario | `411ccfbce4cd08edc179f98d39f0110452307e0d60bcd1c8e4f55e16d2707435` |

`git rev-parse HEAD^` devolvió el padre indicado y `git diff-tree` confirmó
una sola alta, sin modificación ni baja adicional. Los objetos del candidato,
su padre y los dos candidatos O7 citados existen localmente como commits.

## Autoridades contrastadas

La revisión leyó las instrucciones del repositorio y contrastó el candidato
con las siguientes autoridades locales:

- [especificación normalizada de RRHH](../expediente_contratacion_temporal_rrhh.md);
- [objetivos y hoja de ruta](../objetivos_y_hoja_ruta_rrhh_2026-07-23.md);
- [mapa de tareas](../mapa_objetivos_tareas_y_paralelizacion_2026-07-23.md);
- [tablero](../tablero_tareas_contratacion_temporal_2026-07-23.md);
- [matriz normativa](../matriz_normativa_contratacion_temporal_2026-07-23.md);
- [matriz de campos del análisis RRHH](../matriz_campos_analisis_rrhh_contratacion_temporal_2026-07-23.md);
- contratos productivos del árbol base bajo
  `internal/modules/contrataciontemporal/domain` y
  `internal/modules/contrataciontemporal/ports`.

La matriz normativa define CT-CUM-02 como inventario de tratamientos, campos,
fuentes, finalidades, bases y destinatarios y mantiene bloqueada cualquier API
con datos reales. El documento candidato conserva expresamente ese límite.

## Cobertura de las diez familias

El control estructural encontró exactamente `CT-T01` a `CT-T10`. Cada una
contiene una aparición de todas las dimensiones obligatorias: campos o grupos,
fuente y autoridad, finalidad cerrada, base propuesta, interesados/categorías,
destinatarios y conservación. El contraste semántico dio el resultado
siguiente:

| Familias | Estado inventariado | Resultado |
| --- | --- | --- |
| `CT-T01`–`CT-T08` | `ARBOL_ACTUAL`, con los límites materiales propios declarados | Coherentes con los contratos y modelos presentes en la base; presencia técnica no se presenta como composición ni producción. |
| `CT-T09` | `CANDIDATO_LOCAL` `a312375` | El commit existe, define el contrato O7-01 Personal/RPT y no es ancestro de la base inventariada. No se presenta como alta activa. |
| `CT-T10` | `CANDIDATO_LOCAL` `2b15fe8` | El commit existe, define el modelo/mapeo O7-03 GINPIX y no es ancestro de la base inventariada. No se presenta como entrega activa. |

La separación entre `ARBOL_ACTUAL`, `CANDIDATO_LOCAL`, `FUTURO` y
`PENDIENTE_VALIDACION` es explícita. Los candidatos locales no se usan para
elevar el estado del árbol base ni para habilitar Personal o GINPIX.

## Incertidumbres y ausencia de aprobación implícita

Las bases se formulan como propuestas pendientes de validación. Los accesos y
destinatarios no forman una lista corporativa aprobada; los plazos y series
siguen pendientes de Archivo y DPD; tampoco se acredita transferencia,
encargado o comunicación futura. Una ausencia o duda mantiene el tratamiento
cerrado.

El inventario distingue expresamente su alcance del RAT, la EIPD, la tabla de
valoración documental, la categorización ENS y cualquier autorización para
usar datos reales. No atribuye competencias ni declara conformidad.

Permanecen íntegros los bloqueos:

- `CT-CUM-03`: RAT corporativo y EIPD;
- `CT-CUM-04`: categorización ENS y declaración de aplicabilidad;
- `CT-CUM-05`: análisis y tratamiento de riesgos;
- `CT-CUM-06`: política ENI y conservación;
- `CT-CUM-07`: competencia y actuaciones humanas o automatizadas;
- `CT-CUM-08`: evaluación y declaración de accesibilidad;
- `CT-CUM-09`: clasificación y gobierno de casos de IA;
- `CT-CUM-10`: auditoría y actas conjuntas.

Por tanto continúan prohibidos los datos reales, los efectos jurídicos, las
altas en Personal, los envíos a GINPIX, las comunicaciones, las exportaciones,
la preproducción y la producción.

## Privacidad, secretos y enlaces

El fichero inventaría categorías, nombres de campos y referencias técnicas;
no contiene nombres de personas, DNI/NIE, direcciones, correos, teléfonos,
valores económicos reales, contenido documental, credenciales, claves,
tokens, DSN ni material secreto. Las cifras presentes son fechas documentales,
identificadores de tarea, artículos normativos, versiones o SHAs técnicos.

Se ejecutó una búsqueda focal de patrones de secreto y de identificadores,
correo y teléfono. No hubo coincidencias materiales; las únicas coincidencias
de la heurística numérica fueron fechas de los nombres documentales. El
binario `gitleaks` no está instalado en el entorno, por lo que no se
presenta una ejecución de Gitleaks como evidencia de este corte documental.

Los nueve enlaces Markdown locales del inventario resuelven a objetos
existentes. El enlace al RAT histórico se conserva expresamente como entrada
no aprobada, no como autoridad sustitutiva.

## Puertas reproducidas

| Puerta | Resultado |
| --- | --- |
| `git rev-parse HEAD` y `git rev-parse HEAD^` | `475042ba…` y padre exacto `781bb589…` |
| `git diff-tree --no-commit-id --name-status -r 475042ba…` | una sola alta Markdown |
| Control estructural `CT-T01`–`CT-T10` y siete dimensiones | `10/10`, cada dimensión `1/1` |
| Existencia y no pertenencia de `a312375` y `2b15fe8` al árbol base | verde |
| Resolución de enlaces Markdown locales | `9/9` |
| Inspección focal de datos reales y secretos | sin hallazgos materiales |
| `git diff --check 781bb589…475042ba…` | verde, salida vacía |

No se ejecutaron pruebas Go, PostgreSQL, Docker, red, puertas globales, push o
despliegue: el candidato añade exclusivamente documentación y el encargo las
excluye.

## Alcance del GO

El GO acredita que el único Markdown candidato es un inventario documental
trazable, completo para las dimensiones exigidas y prudente ante todas las
incertidumbres detectadas. Solo dirección puede decidir su integración y el
estado transversal. CT-CUM-03 a CT-CUM-10 y todas las validaciones de los
responsables competentes siguen siendo trabajo posterior independiente.
