# Tablero de tareas verificables de contratación temporal

Última actualización: 30 de julio de 2026.

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
| O2-05 | Función SQL de confirmación atómica. | Consumo único de autorización + reserva + expediente + actuación + auditoría + outbox en un `COMMIT`. | ✅ | Integrada en `77743a7` desde `cbe7299`, con GO 2/2. PostgreSQL 18 posterior a la fusión, globales, carrera, `go vet`, ocho fallos, cancelación, respuesta perdida, reinicio, ACL, tamaños y secretos: verdes. |
| O2-06 | Adaptador Go de confirmación y reconciliación de resultado indeterminado. | Éxito, replay, concurrencia, timeout antes/después de `COMMIT`, reinicio y recibo adulterado. | 🚧 | Candidato `157a589` publicado pero en `NO-GO`: replay tras reinicio genera otra candidatura y una cancelación segura antes de enviar `COMMIT` puede reconciliar indebidamente. Se exige resolver candidatura durable y probar la frontera pública completa en PostgreSQL 18. |
| O2-07 | Composición interna real de dependencias. | Arranque falla cerrado sin identidad, PDP, HMAC, generador o PostgreSQL. | — | — |
| O2-08 | API interna y neutral al cliente del alta. | Lista positiva, límites, sin `Cookie`/`Set-Cookie`, almacenamiento web ni cabeceras libres de autoridad; credencial breve ligada al cliente; mismo contrato para web, escritorio, CLI y MCP; contrato OpenAPI. | 🚧 | Adaptador O2-08B revisado con GO e integrado en `42dc3ac`–`94e09e8` (candidato original `3e2885c`): sobre común `{clave_idempotencia, solicitud}`, autoridad de servidor separada y cero hallazgos. Falta registrar la ruta mediante O2-07; por ello todavía no se cierra la tarea ni la puerta funcional. |
| O2-09 | Formulario definitivo conectado. | Accesibilidad de teclado/lector, errores, doble envío, recibo y misma interfaz sin adaptador DEMO. | 🚧 | O2-09B `228df6f` obtuvo GO independiente y se integró en `764fd52`: límites exactos, i18n y cero cookies/almacenamiento/autoridad de cliente. `6fb6cc6` reproduce además las 17 pantallas de RRHH y supera 51/51 capturas locales y 272/272 pruebas del portal. Falta registrar la ruta mediante O2-07 y demostrar el E2E; por ello la tarea no se contabiliza aún como cerrada. |
| O2-10 | E2E y aceptación de la vertical. | Navegador → API → autorización → PostgreSQL → recibo; reinicio y concurrencia; acta RRHH. | — | — |

## O3 — Análisis RRHH y retención de crédito

| Tarea | Entregable aislado | Verificación de cierre | Estado |
| --- | --- | --- | --- |
| O3-01 | Completar invariantes de análisis, jornada, coste y validación de RC. | Matriz de campos del documento RRHH y pruebas de importes/periodos. | ✅ `f33e100`–`9b5fabb`, `994a526` |
| O3-02 | Caso de uso de registrar/rectificar análisis con CAS. | Versiones concurrentes, rectificación motivada y segregación de funciones. | ✅ `df537b4`, integrada en `e9d461c`; GO independiente, focales ×20, carrera ×2, pruebas globales, `go vet`, tamaños y secretos verdes sobre el árbol conjunto. |
| O3-03 | Puertos de fuente presupuestaria y cálculo de coste. | Dobles contractuales; indisponibilidad nunca equivale a validación. | ✅ `1faa8e7`, rework `fca5d41`–`af124fb`, corrección TCB `4e6d14a`–`6d1be36`, integrada en `4c33336`; GO independiente sin hallazgos y puertas globales superadas. |
| O3-04 | Persistencia, auditoría y outbox del análisis. | PostgreSQL real, historia inmutable y recibo verificable. | ✅ `a3e0cf5`–`2834783`: frontera V3, Go → PostgreSQL 18, historia 1–2–3, RC/coste, rectificación segregada, replay, trece rollbacks, cancelación, reinicio, carrera de cuatro sesiones, ACL y `down` verdes; `GO` independiente sin hallazgos medios o superiores. |
| O3-05 | API y formulario de análisis RRHH. | Permisos por operación, accesibilidad, adjuntos por referencia y E2E. | — |

## O4 — Decisión de vía de cobertura

| Tarea | Entregable aislado | Verificación de cierre | Estado |
| --- | --- | --- | --- |
| O4-01 | Catálogo versionado de vías y comprobaciones exigibles. | Publicación inmutable; nueva opción sin recompilar. | ✅ `dff8156`–`baebb55` |
| O4-02 | Consultas minimizadas a Bolsa, SAE y convocatorias. | No accede a tablas ajenas; timeouts y procedencia registrada. | ✅ `913cb7a`, integrada en `0be7467`; GO independiente sin hallazgos, regresión temporal exacta ×100, focales ×50, carrera ×5, puertas globales, tamaños y secretos verdes. |
| O4-03 | Caso de uso de propuesta y decisión motivada. | Resultados contradictorios, ausencia de datos, rectificación y CAS. | ✅ `f5f5f5a`: orquestador nominal de proponer, decidir y rectificar; autorización específica, replay, reserva, motivo gobernado, confirmación única y reconciliación probados; `GO` independiente. No acredita persistencia productiva. |
| O4-04 | Persistencia, autorización, auditoría y outbox. | PostgreSQL real, consumo único y recibo. | ✅ `faa5a5f`, `5954c29`: A/B/C/D/E cerrados; PostgreSQL 18.4, migraciones, roles, ACL/RLS, ambas ramas, C1, carreras, fallos, reinicio y reversión verdes; doble `GO` independiente. |
| O4-05 | API, pantalla comparativa y E2E. | La vía elegida muestra fuentes y justificación sin exponer PII indebida. | 🚧 Tres de cinco hitos. HTTP, registro modular, cliente seguro, recuperación, ciclo de vida, proyecciones, autorización, recibos, acceso durable, cursores y composición visual están cerrados aisladamente. CT-000039 a CT-000046 cierran registro, contrato, cánones, prueba durable, motor privado, fachadas y adaptador Go; CT-000047A cierra los manejadores. CT-000047B cierra retención, recursos, motivos, guardianes, consumidores y M1/M2. C1 cierra la cápsula; C2.1a el rol selector; C2.1b la fachada mínima de Identidad; y S0.1/S0.2 las retiradas ContextoActor, todas con revisión independiente. Estas piezas internas no cambian la métrica. Producción conserva `NO-GO`: faltan C2.2-A/B, selección/registro corporativos, PDP, composición raíz, TLS/mTLS viva y E2E completo. |

Desglose verificable del camino crítico `O4-04`:

| Corte | Entregable | Estado y evidencia |
| --- | --- | --- |
| O4-04A | Sesión TCB sellada e inventario exhaustivo de implementadores y atajos nominales. | ✅ `b819961`; `GO` independiente, focales, carrera y `go vet` verdes. |
| O4-04B | Gobierno durable de catálogo, políticas y resolución de cobertura. | ✅ `54a755e`; `GO` independiente, PostgreSQL 18.4, ciclo ascendente/reversión, ACL, RLS y concurrencia verdes. |
| O4-04C | Reserva terminal y preparación durable sobre el primario. | ✅ `a540522`; `GO` independiente. |
| O4-04D | Lote durable de consumos C1 y wrapper interno VEC. | ✅ `dacf0e1`–`a2bb302`; PostgreSQL 18 real, vínculo exacto decisión/recurso, ACL runtime, replay, concurrencia, reinicio y reversión verdes; GO independiente. No expone una función de ejecución. |
| O4-04E | Confirmación exterior única, auditoría/outbox/recibo y reconciliación primaria. | ✅ `faa5a5f`, `5954c29`; una sola función exterior y transacción, lector fuerte, rol nominativo mínimo, `40001`/`40P01` sin reintento ciego, replays, retirada, historia y `down` protegidos. Véase el [informe de cierre](o4_04e_informe_cierre_confirmacion_durable_2026-07-26.md). |

## O5 — Asignación, informe jurídico y fiscalización

| Tarea | Entregable aislado | Verificación de cierre | Estado |
| --- | --- | --- | --- |
| O5-01 | Asignación de unidad, responsable y bandeja. | Ámbito organizativo, reasignación motivada y notificación. | 🚧 Candidato `0be5600`–`ff6c847`: dominio, caso de uso, PDP V3 y adaptador Go de preparación probados. SQL bloqueado correctamente hasta que O3-04 y O4-04 aporten la versión durable completa; después faltan fuente corporativa, composición y revisión independiente. |
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
| O7-03 | Modelo canónico y mapeo versionado GINPIX. | Datos sintéticos aprobados; campos obligatorios, nulos y compatibilidad. | — |
| O7-04 | Adaptador GINPIX API. | Autenticación, timeout, reintento, conciliación y recibo externo. | — |
| O7-05 | Adaptador GINPIX por fichero. | Mismo modelo canónico, firma/huella, exportación y acuse. | — |
| O7-06 | API/web/E2E y recuperación. | Caída del conector deja espera recuperable, nunca falso éxito. | — |

## O8 — Seguimiento, cierre y conservación

| Tarea | Entregable aislado | Verificación de cierre | Estado |
| --- | --- | --- | --- |
| O8-01 | Estados de seguimiento, prórroga, incidencia y cese. | Transiciones versionadas, motivos y calendario hábil. | ✅ `7b56962`, integrada en `ec8e758`; GO independiente, límites adversariales antes de asignar, replay `O(D+A)`, focales ×50, carrera ×5, globales, `go vet`, tamaños y secretos verdes sobre el árbol conjunto. |
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
