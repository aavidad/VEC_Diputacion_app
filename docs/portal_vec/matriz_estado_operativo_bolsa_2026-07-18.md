# Matriz de estado operativo de Bolsa — 18 de julio de 2026

## Objeto

Esta matriz separa estados que no deben volver a contabilizarse como si fueran
equivalentes. Una pantalla visible, un contrato probado, una prueba de
PostgreSQL y una capacidad aceptada por RRHH son hitos distintos.

El corte se revisa el 21 de julio de 2026 e incorpora los subtramos de identidad
durable, lectura T20B y confirmación T20D que hayan superado revisión
independiente. Un subtramo aislado no se contabiliza como E2E ni como capacidad
productiva.

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

## Capacidades funcionales de Bolsa

| Capacidad | Contrato/adaptador real | E2E técnico | Probable manualmente ahora | UAT/RRHH | Producción | Brecha principal |
| --- | --- | --- | --- | --- | --- | --- |
| Consulta pública | ✅ Servicio, minimización, filtros, API y catálogo | 🧪 Web → API → fuente de fichero DEMO | ✅ En `/bolsa/`, expresamente DEMO | ❌ | ❌ | Proyección durable de convocatorias aprobadas y publicador autorizado. |
| Panel interno agregado | ✅ Dominio, aplicación, HTTP y lectura PostgreSQL | 🟡 PostgreSQL y HTTP probados por separado | 🧪 Solo presentación sintética | ❌ | ❌ | Componer identidad, PDP, productor de proyección y ruta interna. |
| Bandeja y editor de borradores | 🚧 Web, HTTP, fachada, servicio, diario PostgreSQL, lectura V2 y confirmación KMS/recibo reales por piezas | ❌ T20 aún no completa identidad → web → PDP → PostgreSQL/KMS → recibo | 🧪 Interfaz con datos de presentación | ❌ | ❌ | Cerrar `ContextoActor`, opciones/PDP, composición y T20E; lectura y confirmación ya están cerradas de forma aislada. |
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
| Identidad y separación público/interno | ✅ Superficies separadas; T21 aporta mTLS 1.3 e identidad local de alta garantía | ✅ Perfil de seguridad de desarrollo; no una operación completa de Bolsa | 🟡 Se puede arrancar en desarrollo con material local | ❌ | ❌ | Certificado/Kerberos corporativos, ciclo de sesión y aprobación de Sistemas. |
| Roles, RBAC/ABAC y PDP | ✅ Núcleo V2 y piezas PostgreSQL con pruebas | 🟡 Autorización aislada; no enlazada a todas las rutas de Bolsa | ❌ como administración real | ❌ | ❌ | Migrar evidencias V1 a V2, registrar denegaciones, publicar roles/asignaciones y componer. |
| Auditoría, recibos y registro de accesos | ✅ CAS, huellas, outbox, recibos y auditoría en varias verticales | 🟡 Hay E2E parciales de DB, no un acto web unificado | 🧪 Cronología visual | ❌ | ❌ | T12 durabilidad probatoria y T13 accesos con finalidad; consulta y recuperación globales. |
| Documentos, carga, almacén y antivirus | ✅ Puertos de cuarentena, S3 compatible, análisis y promoción; conectores/pruebas aislados | 🟡 Flujo documental técnico por piezas, no asociado E2E a Bolsa | ❌; la carga moderna falla cerrada | ❌ | ❌ | Composición PostgreSQL/S3, antivirus real, descarga autorizada, retención y recuperación. |
| Firma, sello de tiempo, CSV/QR y cotejo | ✅ Contratos PAdES, TSA, custodia y cotejo; flujo de baremación avanzado | 🟡 Pruebas aisladas, no expediente web completo | ❌ desde la aplicación real | ❌ | ❌ | Conector Autofirma/portafirmas, política de firmas, registro, custodia y verificación pública. |
| Generación y exportación de documentos | ✅ Gobierno de formatos y adaptadores PDF/DOCX; contratos de salida extensible | 🟡 Renderizadores probados, no caso administrativo completo | ❌ como flujo real | ❌ | ❌ | Plantillas publicadas, composición, firma, custodia y descarga de PDF/ODT/CSV/JSON/etc. |
| Tasas, pagos y conciliación | ✅ Dominio y contratos robustos; activación HTTP prohibida por diseño | ❌ | ❌ | ❌ | ❌ | Pasarela aprobada, autenticación, saga durable, devolución, conciliación y recibos. |
| Catálogos, configuración y plazos | 🟡 Catálogos versionados y 68 categorías DEMO (5/60/3); 36 procesos basados en 37 publicaciones BOP reales; reglas gobernadas por piezas | 🟡 Consulta pública DEMO; no administración completa | 🧪 Consulta; no edición autorizada | ❌ | ❌ | Catálogos validados por RRHH, editor, calendario y publicación con vigencia. |
| Importación de Convoca y migraciones | ❌ T17 aún pendiente | ❌ | ❌ | ❌ | ❌ | Importador gobernado, validación, reconciliación, incidencias y trazabilidad de procedencia. |
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
3. **T21 sí tiene E2E técnico de seguridad en desarrollo**, pero es un
   habilitador transversal, no una capacidad funcional de Bolsa.
4. **T20 será la primera vertical administrativa moderna E2E**: alta y
   actualización de borradores desde la web hasta PostgreSQL con identidad,
   PDP, KMS, recibo y recuperación.
5. **No consta una UAT formal de Alberto o RRHH para ninguna versión.** Ver una
   demo, comentar una captura o aprobar una dirección de diseño no debe
   registrarse como aceptación funcional.
