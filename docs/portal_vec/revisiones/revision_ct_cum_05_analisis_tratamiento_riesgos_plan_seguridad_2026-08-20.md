# Revisión independiente CT-CUM-05 — análisis y tratamiento de riesgos

Fecha de revisión: 20 de agosto de 2026.

## Dictamen

**GO documental** sobre el candidato exacto
`002805e3b7c01fd13c85fafd14574144b115ea48`, hijo único inmediato de
`946256e1682c33327918f2be15df1e9899b331d5` para este alcance.

| Severidad | Hallazgos abiertos |
| --- | ---: |
| P0 | 0 |
| P1 | 0 |
| P2 | 0 |

El GO acredita únicamente que el dossier candidato es coherente, revisable y
mantiene cerradas sus puertas. No cierra CT-CUM-05, no valora ni reduce riesgo,
no acredita medidas implantadas y no autoriza datos reales, red, servicios,
preproducción o producción.

## Identidad, base y write-set

| Elemento | Valor comprobado |
| --- | --- |
| Worktree | `/srv/fabrica/proyectos/VEC_Diputacion_app/.worktrees/ct-cum-05-riesgos-revision-20260820` |
| Rama de revisión | `revision/ct-cum-05-riesgos-20260820` |
| Candidato | `002805e3b7c01fd13c85fafd14574144b115ea48` |
| Padre/base exacto | `946256e1682c33327918f2be15df1e9899b331d5` |
| Árbol candidato | `3263665b9784a6204249bc5fbd20860865e55066` |
| Árbol base | `89b46b208deeed7bfc03d0afc3da1f6fe08d89fb` |
| Fichero candidato | `docs/portal_vec/ct_cum_05_analisis_tratamiento_riesgos_plan_seguridad_2026-08-20.md` |
| Huella SHA-256 del candidato | `b6354d79940298873e8eff53f129c16f04a12c41c32e968490d19de8c59bfd89` |
| Blob Git del candidato | `472ddfaa69cbc8c38a9c169f6e8309e52b706cf4` |
| Tamaño | 374 líneas |

El diff base→candidato añade únicamente el fichero CT-CUM-05: `374` líneas
añadidas y cero retiradas. No hay cambios de código, pruebas, configuración,
estado transversal ni otros documentos. El blob del fichero de trabajo
coincidía con el blob almacenado en el commit candidato antes de crear esta
acta.

El write-set de esta revisión se limita a la presente acta. El dossier
candidato no se modificó.

## Autoridades leídas y cadena documental

Se leyeron íntegramente las instrucciones de fábrica y repositorio, los cuatro
documentos de relevo/dirección obligatorios, la especificación normalizada, la
hoja de ruta, la matriz normativa, la matriz de campos de análisis y los
dossieres CT-CUM-02, CT-CUM-03 y CT-CUM-04.

Las referencias de base relevantes existen como commits Git:

| Documento o corte | Commit comprobado | Relación |
| --- | --- | --- |
| Base técnica inventariada por CT-CUM-02 | `781bb5891ba3304bfed9d104e71d48846a5b6679` | Existe; no se interpreta como aprobación. |
| Dossier CT-CUM-02 / base declarada por CT-CUM-03 | `475042ba57d63b6abf1810fd26cc65c975496029` | Existe. |
| Dossier CT-CUM-03 / base declarada por CT-CUM-04 | `b2f7a7c0d889c110a765aa9d2d70bab7beeeeabb` | Existe. |
| Dossier CT-CUM-04 / base declarada por CT-CUM-05 | `946256e1682c33327918f2be15df1e9899b331d5` | Existe y es el padre exacto del candidato. |

La matriz normativa reserva CT-CUM-05 al análisis/tratamiento de riesgos y
plan de seguridad, y mantiene el paso a preproducción bloqueado. CT-CUM-03
deja los doce escenarios EIPD sin riesgo residual concluido. CT-CUM-04 no
asigna categoría ENS, no aprueba la declaración de aplicabilidad y trata la
gestión de riesgos como bloqueo de CT-CUM-05. El candidato conserva las tres
autoridades sin ampliarlas.

## Auditoría hostil del contenido

### Autoridad y capability

El dossier limita su capability a ordenar escenarios, consecuencias, medidas
propuestas, evidencia exigida y decisiones pendientes. Niega expresamente que
sea inventario real de activos, valoración aprobada, implantación, aceptación,
plan operativo aprobado o autorización de entornos y datos. La falta de
información, responsable, evidencia o aprobación mantiene el riesgo abierto.

No se atribuye autoridad a mantenedores, agentes, revisores o commits. Método,
escalas, propietarios, prioridad, residual, aceptación y paso a preproducción
se reservan a las autoridades competentes.

### Correspondencia con los doce escenarios EIPD

La cobertura es completa y conserva el sentido de CT-CUM-03:

| EIPD | Riesgo CT-CUM-05 | Resultado |
| --- | --- | --- |
| E01 | R-01, identidad/perfil/capacidad no acreditados | Conforme |
| E02 | R-02, dato inexacto o procedencia discordante | Conforme |
| E03 | R-03, duplicación de autoridad entre módulos | Conforme |
| E04 | R-04, exceso en vistas/eventos/registros/exportaciones | Conforme |
| E05 | R-05, replay/colisión/falso éxito | Conforme |
| E06 | R-06, indisponibilidad convertida en validación | Conforme |
| E07 | R-07, conservación o eliminación indebidas | Conforme |
| E08 | R-08, automatización opaca, incorrecta o discriminatoria | Conforme |
| E09 | R-09, acceso transversal o administración no segregada | Conforme |
| E10 | R-10, derechos o rectificación sin autoridad | Conforme |
| E11 | R-11, entrega incompatible o no conciliada | Conforme |
| E12 | R-12, incidente no detectado o evidencia insuficiente | Conforme |

Los riesgos adicionales R-13 a R-18 cubren, sin afirmar exhaustividad, cambio
y suministro, copias/continuidad, secretos, capacidad, terceros y operación
humana. R-17 permanece `NO_EVALUADO` y no presume la existencia de tercero;
los demás permanecen pendientes y no aceptados.

### Metodología, tratamiento y aceptación

- Probabilidad e impacto permanecen `PENDIENTE_VALORACION`.
- No se asignan frecuencia, severidad, riesgo inherente ni nivel residual.
- El residual permanece `RESIDUAL_NO_CALCULADO` donde corresponde.
- Las once líneas PT-01 a PT-11 son `TRATAMIENTO_PROPUESTO`.
- Las referencias a implantación aparecen solo como condición futura que exige
  propietario, alcance, pruebas, evidencia, revisión, vigencia y decisión.
- Ningún riesgo está aceptado. Los doce riesgos derivados de EIPD están
  explícitamente `NO_ACEPTADO`; R-17 está sin evaluar y la regla global solo
  admite mantener abierto el riesgo.
- No se transforma evidencia técnica parcial, intención de diseño o resultado
  local en eficacia demostrada o decisión competente.

### Minimización y límites materiales

No se encontraron valores personales, contactos, credenciales, DSN, claves,
tokens, certificados, direcciones IP ni rutas privadas. Tampoco se inventarían
activos, topología, cuentas, ubicaciones, redes, proveedores, responsables o
fechas operativas. Las menciones a esas materias son categorías genéricas,
ausencias o entregables futuros.

Los datos reales, efectos, altas, envíos, comunicaciones, red, servicios,
preproducción y producción permanecen expresamente prohibidos. CT-CUM-05
continúa abierto; CT-CUM-06 a CT-CUM-10 y los bloqueos técnicos de O4 y otros
carriles permanecen intactos. No cambian tablero, porcentajes ni estado
transversal.

## Puertas reproducidas

| Puerta | Resultado |
| --- | --- |
| Identidad `HEAD`, padre, rama y árboles | Verde; coinciden con el encargo. |
| `git diff --name-status` base→candidato | Verde; un único `A` para CT-CUM-05. |
| `git diff --numstat` base→candidato | Verde; `374 0` en el único fichero. |
| `git diff --check` base→candidato | Verde; sin salida. |
| Blob de trabajo frente a blob del commit | Verde; ambos `472ddfaa…`. |
| Existencia de commits citados | Verde; cinco de cinco referencias comprobadas. |
| Resolución de enlaces Markdown locales | Verde; nueve de nueve destinos existentes. |
| Inventario E01–E12 | Verde; doce de doce fichas R-01–R-12, sin duplicados ni huecos. |
| Riesgos transversales | Verde; R-13–R-18 presentes y abiertos. |
| Búsqueda de patrones de secretos/datos/infraestructura concreta | Verde; cero coincidencias materiales. |
| Búsqueda de afirmaciones positivas de implantación/aceptación/valoración | Verde; solo negaciones o condiciones futuras. |

No se ejecutaron `go test`, carrera, `go vet`, PostgreSQL, Docker, E2E,
escáneres de red ni despliegues: el diff revisado es exclusivamente documental
y esas puertas no observan mejor su contrato. No se usaron red, instalaciones,
credenciales, datos reales ni sistemas remotos.

## Limitaciones y siguiente bloqueo

Esta revisión no dispone ni debe inventar delimitación formal, inventario de
activos, propietarios, escalas, apetito, infraestructura, evidencia operativa,
riesgo residual o decisiones de aceptación. Por tanto no puede cerrar
CT-CUM-05 ni ninguna aprobación de CT-CUM-02/03/04.

Siguiente tarea desbloqueada: únicamente la integración documental por
dirección del candidato y esta evidencia, si reproduce las puertas. No se
desbloquea una tarea funcional ni preproducción. El siguiente bloqueo material
es obtener de las autoridades competentes el contexto y las decisiones
enumerados por el propio dossier, implantar y probar los tratamientos en un
entorno autorizado, recalcular el residual y documentar la decisión formal.

Revisión independiente: **GO, P0=0, P1=0, P2=0**.
