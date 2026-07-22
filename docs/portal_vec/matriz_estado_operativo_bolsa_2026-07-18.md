# Matriz de estado operativo de Bolsa — 18 de julio de 2026

Último corte técnico: **22 de julio de 2026**, integración `8b2e991`.

## Objeto

Esta matriz separa estados que no deben volver a contabilizarse como si fueran
equivalentes. Una pantalla visible, un contrato probado, una prueba de
PostgreSQL y una capacidad aceptada por RRHH son hitos distintos.

El corte se revisa el 22 de julio de 2026 e incorpora los subtramos de identidad
durable, vínculo de actor V2, evaluación PDP V3, lectura T20B y confirmación
T20D que hayan superado revisión independiente. Un subtramo aislado no se
contabiliza como E2E ni como capacidad productiva.

## Vocabulario de estado

| Término | Significado usado en esta matriz |
| --- | --- |
| **Contrato o adaptador probado** | Dominio, caso de uso, handler o adaptador supera pruebas aisladas. No implica que el servidor lo componga. |
| **E2E técnico** | La prueba recorre las fronteras técnicas necesarias, por ejemplo web/API, aplicación, PostgreSQL o almacén y respuesta. Puede ejecutarse en desarrollo y con datos sintéticos. |
| **Probable manualmente** | Alberto o RRHH pueden recorrer ahora la capacidad arrancando el entorno indicado, sin sustituir manualmente servicios por dobles de prueba. |
| **UAT/RRHH** | Existe una validación funcional formal del usuario o departamento destinatario, con resultado y versión registrados. No equivale a una demostración. |
| **Producción** | Está desplegado con infraestructura y conectores autorizados, operación, copias, recuperación, seguridad y aprobación organizativa. |

Leyenda: `✅` completo en el alcance indicado; `🧪` demostración conectada sin
validez administrativa; `🟡` parcial o probado de forma aislada; `🚧` en curso;
`❌` no alcanzado.

## Tablero incremental de cierre

Este tablero traduce los identificadores técnicos a resultados comprobables.
Se actualiza al cerrar cada tramo; un `✅` solo acredita el alcance descrito en
su fila y no convierte por sí solo Bolsa en productiva.

| Frente comprensible | Resultado verificable | Estado | Evidencia | Siguiente bloqueo |
| --- | --- | --- | --- | --- |
| Registro de identidad y decisión V3 en PostgreSQL | Esquema, concurrencia, revocación, repetición, ACL/RLS y retirada probados con PostgreSQL 18 | ✅ Componente cerrado | `9f6825e`, `906382c`, `05d4abc`, `a6ee330` | Consumo atómico por la operación de Bolsa |
| Canal PostgreSQL productivo | Los tres pools exigen TLS verificado, TLS 1.2 o superior y nombres de servidor coincidentes; el modo local sin TLS exige doble llave | ✅ Componente cerrado | `cae4c29` | Componer los pools en una petición real |
| Frontera del portal público (C1) | `vec-publico` usa una raíz exclusiva, no arrastra dominios internos ni adaptadores DEMO y tiene puerta negativa en CI | ✅ Cerrado y revisado | `1fe43d0`; grafo positivo y autoprueba negativa | Mantener la frontera al ampliar C3/C8 |
| Artefactos público e interno (C2) | Imágenes, manifiestos, recursos y configuración físicamente separados | ✅ Cerrado en su alcance | `8b5ad33`, `8b2e991`; imágenes con UID/GID `10001:10001`, inventario exacto, fallo cerrado y autopruebas negativas | Repetir la certificación de imágenes después de cada integración |
| Consulta pública autoritativa (C3) | API pública alimentada por vistas PostgreSQL de solo lectura y sin datos personales | ❌ NO-GO de revisión | `00c1361`: PostgreSQL 18.4/TLS y gates verdes, pero auditoría independiente con 0 críticos, 0 altos y 4 medios | Cerrar bloqueo lector, disponibilidad real, categorías históricas y redacción de errores; repetir auditoría |
| Cápsula del portal interno (C4) | Listener TLS 1.3/mTLS opaco, secretos de ejecución no privilegiada, SNI, vínculo de canal y apagado seguro | ✅ Cerrado, auditado e integrado | `8051c9f` y `8b2e991`; auditoría 0 hallazgos, carrera global, tres fases Docker y grafo negativo verdes | Incorporar identidad C5 y Bolsa C6 sin reabrir la cápsula |
| Consumo V3 por una operación de Bolsa | La autorización no puede confirmarse antes del CAS/efecto real y deja una prueba acíclica y durable | 🚧 Diseño en revisión | DEC-103, todavía sin commit | Cerrar orden temporal, DAG probatorio, ACL y auditoría de rechazos |
| Primera vertical administrativa real | Navegador → identidad → PDP V3 → PostgreSQL/KMS → recibo, con reinicio, concurrencia y reconciliación | ❌ Pendiente | T20E no iniciado | Cerrar consumo V3 y composición interna C4-C6 |

Lectura ejecutiva del corte: la demostración aprobada sigue disponible, los
componentes reales avanzan, pero ninguna capacidad administrativa debe
declararse productiva hasta superar la última fila extremo a extremo.

## Capacidades funcionales de Bolsa

| Capacidad | Contrato/adaptador real | E2E técnico | Probable manualmente ahora | UAT/RRHH | Producción | Brecha principal |
| --- | --- | --- | --- | --- | --- | --- |
| Consulta pública | ✅ Servicio, minimización, filtros, API, catálogo y adaptador PostgreSQL | 🟡 Runner real PostgreSQL 18.4/TLS supera listado, facetas y detalle, pero C3 conserva cuatro hallazgos medios | ✅ En `/bolsa/`, expresamente DEMO; el recorrido productivo no está autorizado | ❌ | ❌ | Corregir y reauditar C3; después componer publicación interna autorizada y operación. |
| Panel interno agregado | ✅ Dominio, aplicación, HTTP y lectura PostgreSQL | 🟡 PostgreSQL y HTTP probados por separado | 🧪 Solo presentación sintética | ❌ | ❌ | Componer identidad, PDP, productor de proyección y ruta interna. |
| Bandeja y editor de borradores | 🚧 Web, HTTP, fachada, servicio, ContextoActor V2 durable, vínculo actor V2, solicitud/evaluación PDP V3, acreditación SQL/Go del contexto, diario PostgreSQL, lectura V2 y confirmación KMS/recibo reales por piezas | ❌ T20 aún no completa el registro confirmado PDP V3 ni identidad → PDP → PostgreSQL/KMS → recibo en una petición | 🧪 Interfaz con datos de presentación | ❌ | ❌ | Cerrar servicio y registro transaccional V3, composición y T20E; la evaluación en memoria es deliberadamente no ejecutable. |
| Publicación, sustitución y retirada | 🟡 Dominio y contratos; fuera del cierre de T20 | ❌ | 🧪 Controles informativos | ❌ | ❌ | DEC-091: aprobación firmada, dependencias autoritativas y acto durable. |
| Bases y reglas de baremo | ✅ Modelo versionado, topes, jornada, redondeo y cálculo; persistencia parcial | 🟡 Pruebas de núcleo/SQL, sin editor compuesto | 🧪 Pantalla y valores sintéticos | ❌ | ❌ | Fuente normativa autoritativa, editor RRHH, publicación y simulador conectados. |
| Autobaremación del aspirante | 🟡 Flujo heredado `fake` y motor moderno aislado | 🟡 Recorrido heredado DEMO; no vertical moderna | 🧪 Sí, en modo heredado de demostración | ❌ | ❌ | Migrar a identidad, reglas publicadas, expediente, documentos y persistencia modernas. |
| Revisión técnica de baremación | ✅ Aceptación, rechazo, subsanación, rectificación, revocación, rehabilitación y firma modeladas; PostgreSQL avanzado | 🟡 Aplicación/DB probadas, sin recorrido web completo | 🧪 Concepto visual, no acto administrativo | ❌ | ❌ | Bandeja RRHH, autorización V2, evidencia, firma/custodia y composición. |
| Llamamientos | ✅ Dominio, aplicación y HTTP; PostgreSQL aún deliberadamente incompleto | ❌ No hay propuesta → confirmación → comunicación completa | 🧪 Asistente sintético sin confirmar | ❌ | ❌ | Fuente autoritativa, transacción V2, composición, respuesta y notificación fehaciente. |
| Contratos, ceses y reincorporaciones | 🟡 Conceptos y presentación; no agregado funcional completo | ❌ | 🧪 Tabla sintética | ❌ | ❌ | Modelo propio, expedientes, integración con RRHH/nómina y trazabilidad del ciclo. |
| Candidatura, solicitud, registro y alegaciones | 🟡 Recorrido heredado y modelos parciales | 🟡 DEMO `fake`; el recorrido moderno no existe completo | 🧪 Parcial | ❌ | ❌ | Solicitud firmada, registro, subsanación, alegación, resolución y notificación. |
| Comunicaciones y respuestas | 🟡 Outbox y contratos parciales; avisos DEMO | ❌ | 🧪 Canales y avisos de presentación | ❌ | ❌ | Conectores, preferencias, reintentos, entrega, acuse y notificación fehaciente. |
| Ayuda | ✅ FAQ, contenido contextual, audio y transcripción estáticos | ✅ Para el recurso estático | ✅ | ❌ formal | ❌ como sistema administrable | Catálogo gobernado, edición, contexto por expediente, cobertura y bot. |

## Capacidades transversales que faltaban en la tabla de 14

| Capacidad transversal | Contrato/adaptador real | E2E técnico | Probable manualmente ahora | UAT/RRHH | Producción | Brecha principal |
| --- | --- | --- | --- | --- | --- | --- |
| Identidad y separación público/interno | ✅ C1/C2 separan procesos y artefactos; C4 aporta una cápsula TLS 1.3/mTLS auditada; ContextoActor V2 registra cuenta, perfil, versiones y procedencia maestra | 🟡 Cápsula, registro y acreditación PostgreSQL 18 están probados; C4 falla antes de escuchar mientras falten C5/C6 | ❌ como ruta administrativa real, por diseño cerrado | ❌ | ❌ | C5: certificado cliente + aserción Kerberos ligada al canal, sesión y revalidación; C6: PDP y caso de uso de Bolsa. |
| Roles, RBAC/ABAC y PDP | 🚧 Núcleo V2 congelado; solicitud, evidencia y decisión V3 nominales probadas y no ejecutables sin confirmación durable | 🟡 Autorización aislada; registro V3 y enlace a las rutas de Bolsa en curso | ❌ como administración real | ❌ | ❌ | Cerrar confirmación durable V3, publicar roles/asignaciones y componer sin degradación V1/V2. |
| Auditoría, recibos y registro de accesos | ✅ CAS, huellas, outbox, recibos y auditoría en varias verticales | 🟡 Hay E2E parciales de DB, no un acto web unificado | 🧪 Cronología visual | ❌ | ❌ | T12 durabilidad probatoria y T13 accesos con finalidad; consulta y recuperación globales. |
| Documentos, carga, almacén y antivirus | ✅ Puertos de cuarentena, S3 compatible, análisis y promoción; conectores/pruebas aislados | 🟡 Flujo documental técnico por piezas, no asociado E2E a Bolsa | ❌; la carga moderna falla cerrada | ❌ | ❌ | Composición PostgreSQL/S3, antivirus real, descarga autorizada, retención y recuperación. |
| Firma, sello de tiempo, CSV/QR y cotejo | ✅ Contratos PAdES, TSA, custodia y cotejo; flujo de baremación avanzado | 🟡 Pruebas aisladas, no expediente web completo | ❌ desde la aplicación real | ❌ | ❌ | Conector Autofirma/portafirmas, política de firmas, registro, custodia y verificación pública. |
| Generación y exportación de documentos | ✅ Gobierno de formatos y adaptadores PDF/DOCX; contratos de salida extensible | 🟡 Renderizadores probados, no caso administrativo completo | ❌ como flujo real | ❌ | ❌ | Plantillas publicadas, composición, firma, custodia y descarga de PDF/ODT/CSV/JSON/etc. |
| Tasas, pagos y conciliación | ✅ Dominio y contratos robustos; activación HTTP prohibida por diseño | ❌ | ❌ | ❌ | ❌ | Pasarela aprobada, autenticación, saga durable, devolución, conciliación y recibos. |
| Catálogos, configuración y plazos | 🟡 Catálogos versionados y 68 categorías DEMO (5/60/3); 36 procesos basados en 37 publicaciones BOP reales; reglas gobernadas por piezas | 🟡 Consulta pública DEMO; no administración completa | 🧪 Consulta; no edición autorizada | ❌ | ❌ | Catálogos validados por RRHH, editor, calendario y publicación con vigencia. |
| Importación de Convoca y migraciones | 🟡 T17 aporta un primer corte sintético aislado: XLS BIFF8 endurecido, staging, validación, huella, CAS y procedencia no autoritativa | ❌ | ❌ | ❌ | ❌ | Persistencia PostgreSQL cifrada, autorización, auditoría, conservación, reconciliación corporativa y composición E2E. |
| API, CLI, MCP y bot público | 🟡 API pública DEMO operativa; contratos internos numerosos | 🟡 Solo API pública de consulta | ✅ API/web pública DEMO; no CLI/MCP/bot | ❌ | ❌ | API gobernada completa, CLI y MCP con el mismo PDP; RAG/bot limitado a información pública. |
| Protección de datos, conservación y expurgo | ✅ Reglas de minimización y seguridad incorporadas en muchos contratos | 🟡 Pruebas adversarias parciales | ❌ como gestión integral | ❌ | ❌ | Registro de accesos, política de conservación, expurgo, derechos y evidencias de ejecución. |
| Operación, copias, recuperación y observabilidad | 🟡 Salud, migraciones y algunas pruebas de recuperación | ❌ para el módulo completo | ❌ | ❌ | ❌ | Runbook, métricas, alertas, copias restauradas, recuperación, continuidad y capacidad. |
| Accesibilidad, tema y personalización | ✅ Tema común, alto contraste, texto ampliado, audio y navegación accesible en las vistas principales | 🟡 Pruebas web; falta auditoría integral WCAG | ✅ En las pantallas disponibles | ❌ formal | ❌ | Auditoría WCAG/UNE, preferencias persistidas y validación con usuarios. |

## Lectura ejecutiva

1. **No hay todavía ninguna capacidad administrativa de Bolsa desplegada en
   producción.** Esto no significa que no haya código real.
2. **Sí hay dos recorridos que pueden probarse manualmente hoy:** consulta
   pública conectada a datos DEMO y ayuda estática. El portal RRHH también se
   puede enseñar, pero sus acciones no producen actos administrativos.
3. **C1, C2 y C4 están cerrados en su alcance.** C4 es un habilitador
   transversal y permanece deliberadamente sin escuchar hasta disponer de
   identidad C5 y cableado C6; no es por sí solo una capacidad de Bolsa.
4. **C3 tiene código real e integración PostgreSQL/TLS verde, pero continúa en
   NO-GO por cuatro hallazgos medios.** No se contabiliza como productivo ni se
   ocultan esos defectos detrás del éxito de las pruebas.
5. **T20 será la primera vertical administrativa moderna E2E**: alta y
   actualización de borradores desde la web hasta PostgreSQL con identidad,
   PDP, KMS, recibo y recuperación.
6. **No consta una UAT formal de Alberto o RRHH para ninguna versión.** Ver una
   demo, comentar una captura o aprobar una dirección de diseño no debe
   registrarse como aceptación funcional.
