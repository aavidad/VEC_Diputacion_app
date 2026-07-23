# Tablero de tareas verificables de contratación temporal

Última actualización: 23 de julio de 2026.

Este tablero descompone los objetivos `O1` a `O8` en unidades que pueden
revisarse, probarse y confirmarse en Git sin mezclar responsabilidades.

## Regla de una tarea

Una tarea:

1. produce un resultado técnico o funcional observable;
2. declara qué no incluye;
3. tiene pruebas concretas;
4. actualiza documentación y relevo;
5. termina en uno o varios commits consecutivos con el identificador de tarea;
6. no cambia a «cerrada» hasta registrar aquí el commit y la evidencia.

Si una tarea crece por encima de una revisión razonable, se divide antes de
programarla. No se juntan dos objetivos distintos para ahorrar commits.

Estados:

- `✅ Cerrada`: commit publicado y puertas superadas;
- `🚧 En curso`: cambios locales o revisión pendiente;
- `⬜ Lista`: dependencias cerradas y puede comenzar;
- `⛔ Bloqueada`: falta una decisión o dependencia externa identificada;
- `— Futura`: todavía depende de tareas anteriores.

## O1 — Base del módulo

| Tarea | Entregable aislado | Verificación de cierre | Estado | Commit |
| --- | --- | --- | --- | --- |
| O1-01 | Normalizar el documento de RRHH y decidir módulo complementario. | Revisión de campos, fases, responsabilidades y regla de no duplicación. | ✅ | `e88ac16` |
| O1-02 | Publicar manifiesto, permisos y navegación sin asignar roles. | Pruebas de manifiesto, claves únicas y ausencia de concesiones. | ✅ | `e88ac16` |
| O1-03 | Modelar solicitud, análisis, cobertura, asignación y cronología. | `go test`, carrera, `go vet`, concurrencia optimista y copias defensivas. | ✅ | `c7a64da` |
| O1-04 | Documentar objetivos, tareas y relevo reproducible. | Enlaces desde README/estado y siguiente tarea inequívoca. | ✅ | `e2e9612` |

## O2 — Primera vertical real: alta de solicitud

| Tarea | Entregable aislado | Verificación de cierre | Estado | Commit |
| --- | --- | --- | --- | --- |
| O2-01 | Caso de uso con identidad alta, flujo, HMAC, idempotencia y autorización opaca. | Éxito, reintento, denegación, adulteración, cancelación y nulos tipados. | ✅ | `40105d2` |
| O2-02 | Adaptador Go de preparación PostgreSQL y conectores de sellado/generación. No instala SQL. | Unitarias de respuesta, ACL contractual, clave no serializada, carrera, `go vet`. | ✅ | `5288498`, corrección revisada `effa911` |
| O2-03 | Roles, migración y prueba SQL real de preparación idempotente; convivencia de generaciones HMAC antes del cierre. No confirma expedientes. | PostgreSQL efímero: alta, reintento, conflicto, ACL, `down` protegido y rotación v1→v2 sin segunda reserva ni falso conflicto. | ✅ | `55fdf19`–`6e007c8`; GO independiente y PostgreSQL real 3/3 |
| O2-04 | Sustituir la autorización local provisional por la capacidad durable común de VEC. | Revisión arquitectónica; no existe constructor fabricable ni segunda autoridad; pruebas de replay y vigencia. | ✅ | `edbd0db`; GO independiente, PostgreSQL real y dos pruebas concurrentes 16/16 sin fallos |
| O2-05 | Función SQL de confirmación atómica. | Consumo único de autorización + reserva + expediente + actuación + auditoría + outbox en un `COMMIT`. | 🚧 | Diseño `1f16164`; VEC-AD-3 nominal/parser con GO en `fe00ed9`; confianza, capacidad breve y consumo SQL pendientes |
| O2-06 | Adaptador Go de confirmación y reconciliación de resultado indeterminado. | Éxito, replay, concurrencia, timeout antes/después de `COMMIT`, reinicio y recibo adulterado. | — | — |
| O2-07 | Composición interna real de dependencias. | Arranque falla cerrado sin identidad, PDP, HMAC, generador o PostgreSQL. | — | — |
| O2-08 | API interna y neutral al cliente del alta. | Lista positiva, límites, sin `Cookie`/`Set-Cookie`, almacenamiento web ni cabeceras libres de autoridad; credencial breve ligada al cliente; mismo contrato para web, escritorio, CLI y MCP; contrato OpenAPI. | — | — |
| O2-09 | Formulario definitivo conectado. | Accesibilidad de teclado/lector, errores, doble envío, recibo y misma interfaz sin adaptador DEMO. | — | — |
| O2-10 | E2E y aceptación de la vertical. | Navegador → API → autorización → PostgreSQL → recibo; reinicio y concurrencia; acta RRHH. | — | — |

## O3 — Análisis RRHH y retención de crédito

| Tarea | Entregable aislado | Verificación de cierre | Estado |
| --- | --- | --- | --- |
| O3-01 | Completar invariantes de análisis, jornada, coste y validación de RC. | Matriz de campos del documento RRHH y pruebas de importes/periodos. | ✅ `f33e100`–`9b5fabb`, `994a526` |
| O3-02 | Caso de uso de registrar/rectificar análisis con CAS. | Versiones concurrentes, rectificación motivada y segregación de funciones. | 🚧 Dominio de rectificación en `2cd3da1`; aplicación y revisión pendientes. |
| O3-03 | Puertos de fuente presupuestaria y cálculo de coste. | Dobles contractuales; indisponibilidad nunca equivale a validación. | 🚧 Implementado en `1faa8e7`; revisión independiente pendiente. |
| O3-04 | Persistencia, auditoría y outbox del análisis. | PostgreSQL real, historia inmutable y recibo verificable. | — |
| O3-05 | API y formulario de análisis RRHH. | Permisos por operación, accesibilidad, adjuntos por referencia y E2E. | — |

## O4 — Decisión de vía de cobertura

| Tarea | Entregable aislado | Verificación de cierre | Estado |
| --- | --- | --- | --- |
| O4-01 | Catálogo versionado de vías y comprobaciones exigibles. | Publicación inmutable; nueva opción sin recompilar. | ✅ `dff8156`–`baebb55` |
| O4-02 | Consultas minimizadas a Bolsa, SAE y convocatorias. | No accede a tablas ajenas; timeouts y procedencia registrada. | 🚧 Puerto genérico, minimizado y limitado temporalmente en `a34d462`, reforzado en `fd766a2`; adaptadores y revisión independiente pendientes. |
| O4-03 | Caso de uso de propuesta y decisión motivada. | Resultados contradictorios, ausencia de datos, rectificación y CAS. | — |
| O4-04 | Persistencia, autorización, auditoría y outbox. | PostgreSQL real, consumo único y recibo. | — |
| O4-05 | API, pantalla comparativa y E2E. | La vía elegida muestra fuentes y justificación sin exponer PII indebida. | — |

## O5 — Asignación, informe jurídico y fiscalización

| Tarea | Entregable aislado | Verificación de cierre | Estado |
| --- | --- | --- | --- |
| O5-01 | Asignación de unidad, responsable y bandeja. | Ámbito organizativo, reasignación motivada y notificación. | ⬜ |
| O5-02 | Plantilla/datos del informe jurídico. | Generación por conector, versión, anexos y referencias normativas. | — |
| O5-03 | Revisión y firma del informe. | Firma múltiple, rechazo, subsanación, CSV/QR y validación. | — |
| O5-04 | Solicitud y resultado de fiscalización. | Intervención segregada, reparo, subsanación y recibo. | — |
| O5-05 | Persistencia/API/web/E2E de las tres fases. | Historia completa y ningún salto de fase no autorizado. | — |

## O6 — Llamamiento y formalización

| Tarea | Entregable aislado | Verificación de cierre | Estado |
| --- | --- | --- | --- |
| O6-01 | Contrato de integración con el módulo Bolsa. | Referencias/eventos; ninguna lectura directa de sus tablas. | ✅ `2b67c7a`–`20935bd`; doble GO independiente, pruebas focales y globales, carrera y `go vet`. |
| O6-02 | Selección y propuesta de llamamiento. | Orden, disponibilidad, exclusiones, desempate y evidencia de regla. | — |
| O6-03 | Comunicación, aceptación, renuncia y siguiente candidato. | Plazos, canales, entrega, reintento e idempotencia. | — |
| O6-04 | Propuesta de nombramiento/contrato y documentación. | Plantillas gobernadas, anexos, firma múltiple y descarga interesada. | — |
| O6-05 | Persistencia/API/web/E2E. | Ciclo completo con auditoría y rectificación sin reescritura. | — |

## O7 — Incorporación, Personal y GINPIX

| Tarea | Entregable aislado | Verificación de cierre | Estado |
| --- | --- | --- | --- |
| O7-01 | Contrato de alta con Personal/RPT. | Puesto/plaza/relación por referencia y rechazo coherente. | ⬜ |
| O7-02 | Confirmación de incorporación. | Fecha, ocupación, documentos y evento durable. | — |
| O7-03 | Modelo canónico y mapeo versionado GINPIX. | Fixture aprobado; campos obligatorios, nulos y compatibilidad. | — |
| O7-04 | Adaptador GINPIX API. | Autenticación, timeout, reintento, conciliación y recibo externo. | — |
| O7-05 | Adaptador GINPIX por fichero. | Mismo modelo canónico, firma/huella, exportación y acuse. | — |
| O7-06 | API/web/E2E y recuperación. | Caída del conector deja espera recuperable, nunca falso éxito. | — |

## O8 — Seguimiento, cierre y conservación

| Tarea | Entregable aislado | Verificación de cierre | Estado |
| --- | --- | --- | --- |
| O8-01 | Estados de seguimiento, prórroga, incidencia y cese. | Transiciones versionadas, motivos y calendario hábil. | ⬜ |
| O8-02 | Caso de uso de cierre administrativo. | Tareas pendientes impiden cierre; reapertura excepcional auditada. | — |
| O8-03 | Política de conservación y bloqueo. | Tabla aprobada por tipo documental/procedimiento y base jurídica. | — |
| O8-04 | Expurgo gobernado y prueba de eliminación. | Doble autorización, retención legal, recibo y ausencia de borrado silencioso. | — |
| O8-05 | Consulta histórica y exportación autorizada. | Registro de lectura/descarga y minimización por perfil. | — |
| O8-06 | API/web/E2E y manual operativo. | Recuperación, copias, restauración y aceptación formal. | — |

## Puertas comunes antes de cada commit

Como mínimo:

```text
gofmt
go test del paquete afectado
go test -race del paquete afectado
go vet del paquete afectado
git diff --check
```

Si cambia PostgreSQL se añade prueba contra una instancia efímera real,
aplicación ascendente, ACL runtime, reintento/concurrencia y reversión
protegida. Si cambia API o web se añaden pruebas de contrato, seguridad,
accesibilidad y captura/revisión visual. Si cambia composición se ejecutan las
puertas globales compatibles antes de integrar.

Además, cada tarea declara y supera las puertas aplicables de
[la matriz normativa](matriz_normativa_contratacion_temporal_2026-07-23.md).
Las tareas `CT-CUM-01` a `CT-CUM-10` son transversales y pueden desarrollarse
en paralelo, pero sus aprobaciones bloquean datos reales, efectos jurídicos o
producción según indica la propia matriz.
