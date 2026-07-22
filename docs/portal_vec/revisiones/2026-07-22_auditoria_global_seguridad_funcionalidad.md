# Auditoría global de seguridad y cobertura funcional

**Fecha de corte:** 22 de julio de 2026

**Commit auditado:** `cae4c29` (`seguridad: exigir tls verificado en postgresql`)

**Resultado:** **NO-GO para producción**; apto para continuar el desarrollo y para una demostración expresamente rotulada como no autoritativa.

**Alcance:** código, configuración, migraciones, contenedores, automatización y cobertura funcional de Bolsa presentes en el commit indicado.

## 1. Alcance y criterio

Esta revisión está congelada sobre `cae4c29`. No incorpora ni acredita los cambios sin confirmar que existían en el árbol de trabajo durante la auditoría, incluida la ejecución en curso de C1. Cuando esos cambios se integren deberán someterse a una revisión incremental y a las mismas puertas de calidad.

El análisis distingue cuatro estados:

| Estado | Significado en este informe |
|---|---|
| **Implementado** | Existe código ejecutable y probado para la capacidad indicada. No implica que esté montado en el producto. |
| **Contrato** | Existen dominio, puertos, invariantes o pruebas, pero falta al menos un adaptador real o el efecto completo. |
| **Demostración** | Funciona con datos sintéticos o adaptadores de memoria/fichero y no tiene valor administrativo. |
| **Inexistente** | No se encontró una implementación funcional para el alcance requerido. |

“E2E productivo” significa una prueba de extremo a extremo sobre la cadena real: entrada por su superficie definitiva, identidad real, autorización, caso de uso, persistencia autoritativa, efectos externos, auditoría y recuperación. No significa únicamente que una persona haya pulsado una pantalla ni que una prueba unitaria esté en verde.

La severidad expresa el riesgo de autorizar producción en el estado auditado:

- **Crítica:** bloquea por sí sola cualquier producción con datos o efectos reales.
- **Alta:** bloquea la capacidad afectada o deja sin una garantía obligatoria.
- **Media:** deuda relevante que debe cerrarse antes del piloto o mediante una aceptación de riesgo formal y acotada.
- **Baja:** endurecimiento o mantenibilidad sin explotación directa demostrada.

Este documento es una revisión técnica del repositorio, no una certificación del ENS, una declaración de conformidad RGPD/LOPDGDD, una EIPD aprobada ni una autorización de puesta en producción.

## 2. Veredicto ejecutivo

La base arquitectónica es considerable y, en varias áreas, exigente: denegación por defecto, contratos de cuatro superficies, identidad de alta garantía, registro PostgreSQL V3, separación de roles de base de datos, TLS verificado, almacenamiento S3 compatible con cuarentena y retención, modelos probatorios y una suite extensa. El commit auditado supera la suite Go completa, `go vet` y la compilación de todos los binarios.

Sin embargo, esas piezas no forman todavía una aplicación productiva. La raíz integrada rechaza deliberadamente el perfil de producción, compone almacenes en memoria en desarrollo y los manejadores reales de Bolsa, los tres pools PostgreSQL, la identidad, el PDP V3, S3, antivirus y firma no están montados juntos. Tampoco existe aún un despliegue con separación física completa, auditoría global durable, copias/restauración probadas ni observabilidad operativa.

Por ello:

1. **No existe ninguna capacidad de Bolsa E2E productiva en `cae4c29`.**
2. **Sí existen capacidades implementadas y probadas por componentes**, especialmente gobierno de borradores, autorización, baremación, llamamientos, S3 y modelos de firma.
3. **La presentación no debe confundirse con producción**: contiene datos y efectos sintéticos y lo declara expresamente.
4. **TLS PostgreSQL y la puerta CI del registro ContextoActor/PDP V3 se consideran cerrados a nivel de componente**, no a nivel de composición productiva.

La separación física tiene su análisis y plan C1-C10 en [Revisión de separación física de superficies](./2026-07-22_separacion_fisica_superficies_no_go.md), documento de revisión añadido después del commit auditado. Esta auditoría lo toma como referencia y no lo duplica. La validación independiente de sus destinos de código y configuración contra `cae4c29` confirma sus rutas, evidencias y veredicto **NO-GO**.

## 3. Resumen por área

| Área | Estado real en `cae4c29` | Severidad pendiente | Resultado |
|---|---|---:|---|
| Superficies y raíces de composición | Contratos implementados; separación productiva no materializada | Crítica | NO-GO |
| Identidad, sesiones y cabeceras | Contratos y adaptadores PostgreSQL implementados; no montados | Alta | NO-GO interno/personal/admin |
| RBAC, ABAC y PDP V3 | Implementados y probados por componentes; no montados | Alta | NO-GO para efectos reales |
| PostgreSQL, TLS y roles | T20/TLS implementado; V3 en CI; sin composición de producto | Alta | Cierre parcial acreditado |
| Secretos | No se encontraron secretos reales en el árbol auditado; falta puerta CI e histórico acreditado | Media | Aceptable para continuar, no suficiente para declarar histórico limpio |
| Ficheros, S3 y antivirus | S3 fuerte implementado; antivirus solo contrato; flujo no montado | Alta | NO-GO documental |
| Documentos y firma | PDF/DOCX reales; formatos y PAdES/TSA por contrato; conectores reales ausentes | Alta | NO-GO firma administrativa |
| Auditoría y privacidad | Modelo rico y registros verticales; almacén global en memoria y ciclo RGPD incompleto | Crítica | NO-GO probatorio |
| Docker y redes | Imágenes fijadas/endurecidas; artefactos y superficies productivas incorrectos | Crítica | NO-GO despliegue |
| CI y cadena de suministro | Puerta Go sólida y V3 PostgreSQL; faltan varias puertas de artefacto/seguridad/E2E | Alta | Incompleto |
| Copias, restauración y observabilidad | Requisitos documentados; operación real no encontrada | Crítica | NO-GO operación |
| Bolsa | Mucho núcleo implementado por piezas; ninguna vertical productiva completa | Alta | NO-GO funcional productivo |

## 4. Hallazgos de seguridad y operación

### CR-01 — No existe raíz productiva completa ni separación física acreditada

**Estado:** contrato e implementación parcial; no montado.

**Severidad:** crítica.

La composición integrada corta el arranque productivo mediante `ErrComposicionProductivaNoDisponible` y explica que faltan identidad y repositorios autoritativos ([`bootstrap.go`, líneas 51-54](../../../internal/app/bootstrap/bootstrap.go#L51-L54), [80-87](../../../internal/app/bootstrap/bootstrap.go#L80-L87), [293-301](../../../internal/app/bootstrap/bootstrap.go#L293-L301)). La composición disponible crea un almacén en memoria y registra conjuntamente Personal, Cronos, Dietas, Bolsa y Administración ([líneas 224-260](../../../internal/app/bootstrap/bootstrap.go#L224-L260)). Esto es un cierre seguro frente a una falsa producción, pero también prueba que aún no hay producto desplegable.

El binario público de ese commit llama al constructor público, pero este permanece dentro del paquete monolítico `bootstrap` ([`main.go`, líneas 8-15](../../../cmd/vec-publico/main.go#L8-L15); [`bootstrap.go`, líneas 113-125](../../../internal/app/bootstrap/bootstrap.go#L113-L125)). Como la unidad de compilación de Go es el paquete, sus importaciones arrastran al menos veintiún paquetes claramente ajenos a la superficie pública: candidatura, Administración, Cronos, Dietas y Personal ([`bootstrap.go`, líneas 13-34](../../../internal/app/bootstrap/bootstrap.go#L13-L34)). El [`Dockerfile`](../../../Dockerfile#L22-L60) no construye `vec-publico`; el runtime copia `vec-server` ([líneas 119-144](../../../Dockerfile#L119-L144)). Además, `vec-api` no fija `target`, por lo que selecciona la última etapa del Dockerfile, destinada a Playwright ([`docker-compose.yml`, líneas 335-374](../../../docker-compose.yml#L335-L374); [`Dockerfile`, líneas 146-163](../../../Dockerfile#L146-L163)).

**Cierre exigido:** completar y revisar C1-C10 del informe de separación física; binarios, cargadores de configuración, manifiestos web, cuentas, listeners, redes y secretos independientes; pruebas negativas de cruces entre zonas.

### AL-01 — Identidad y sesión seguras existen, pero no protegen una raíz real

**Estado:** implementado por componentes; no montado.

**Severidad:** alta.

El contrato reconoce cuatro superficies cerradas ([`superficie.go`, líneas 31-41](../../../internal/vec/adapters/httpseguridad/superficie.go#L31-L41)), exige política de red explícita, impide escucha comodín interna ([líneas 74-124](../../../internal/vec/adapters/httpseguridad/superficie.go#L74-L124)) y obliga a Kerberos más certificado, dos grupos criptográficos y garantía alta en las superficies internas ([líneas 156-169](../../../internal/vec/adapters/httpseguridad/superficie.go#L156-L169)). También exige la arquitectura completa y listeners separados ([líneas 275-341](../../../internal/vec/adapters/httpseguridad/superficie.go#L275-L341)).

Existen servicio de identidad y canal TLS mutuo ([`identidad.go`](../../../internal/vec/adapters/httpseguridad/identidad.go)), registro durable de sesiones con cuentas PostgreSQL separadas ([`registro.go`, líneas 50-90](../../../internal/vec/adapters/httpseguridad/postgres/registro.go#L50-L90)) y revalidación del actor dentro de una transacción ([`revalidador_autenticacion_actor.go`, líneas 30-91](../../../internal/vec/adapters/httpseguridad/postgres/revalidador_autenticacion_actor.go#L30-L91)). No se encontró una raíz no de prueba que los componga con los manejadores internos.

El servidor HTTP sí aplica listas positivas y defensas útiles: rutas distintas por superficie ([`server.go`, líneas 86-183](../../../internal/app/server/server.go#L86-L183)), rechazo de rutas no canónicas ([líneas 282-310](../../../internal/app/server/server.go#L282-L310)), cookies, cabeceras proxy, identidad heredada y tráileres ([líneas 318-460](../../../internal/app/server/server.go#L318-L460)), además de `Authorization` ([líneas 391-399](../../../internal/app/server/server.go#L391-L399)). Deben conservarse.

**Riesgo residual medio:** el cliente de borradores conserva una rama opcional de `Bearer` aunque la composición actual le entrega `null` y el servidor lo rechaza. Debe retirarse del artefacto productivo para evitar una regresión de política ([`portal-borradores-api.js`, líneas 328-447](../../../web/static/portal-empleado/portal-borradores-api.js#L328-L447)).

### AL-02 — RBAC/ABAC/PDP V3 es sólido como componente, pero no gobierna el producto

**Estado:** implementado y probado por componentes; no montado.

**Severidad:** alta.

`ServicioAutorizacion` evalúa RBAC seguido de restricciones ABAC y falla cerrado ante dependencias inválidas ([`autorizacion.go`, líneas 21-63](../../../internal/vec/application/autorizacion.go#L21-L63), [66-110](../../../internal/vec/application/autorizacion.go#L66-L110)). Parte de una denegación por defecto, vincula la decisión a actor, perfil, versión de rol, políticas, finalidad y recurso ([líneas 164-248](../../../internal/vec/application/autorizacion.go#L164-L248)), registra tanto denegaciones como concesiones antes de devolver resultado ([líneas 250-319](../../../internal/vec/application/autorizacion.go#L250-L319)) y solo permite que ABAC restrinja una concesión RBAC ([líneas 322-380](../../../internal/vec/application/autorizacion.go#L322-L380)).

La variante ligada a solicitud V3 y su adaptador PostgreSQL implementan consumo/registro transaccional, serializable y confirmación antes del éxito ([`autorizacion_servicio_solicitud_v3.go`, líneas 34-109](../../../internal/vec/application/autorizacion_servicio_solicitud_v3.go#L34-L109), [128-245](../../../internal/vec/application/autorizacion_servicio_solicitud_v3.go#L128-L245); [`autorizacion_solicitud_v3.go`, líneas 22-150](../../../internal/vec/adapters/postgres/autorizacion_solicitud_v3.go#L22-L150)). No hay consumidor productivo de `NuevoServicioAutorizacionSolicitudLigadaV3` en el commit.

**Cierre exigido:** montar identidad → `ContextActor` → instantánea RBAC/ABAC → PDP V3 → efecto en la misma transacción o protocolo durable definido; pruebas de denegación para cada rol, finalidad, ámbito y superficie.

### AL-03 — PostgreSQL T20/TLS/roles está cerrado solo a nivel de componente

**Estado:** implementado y probado; no montado.

**Severidad:** alta por ausencia de integración; el defecto TLS revisado está cerrado.

La configuración exige tres DSN distintos y no expone su contenido en representaciones genéricas ([`postgresql_borradores.go`, líneas 10-66](../../../config/postgresql_borradores.go#L10-L66), [91-112](../../../config/postgresql_borradores.go#L91-L112)). Los pools fijan roles exactos, límites, `search_path`, aislamiento serializable, tiempos máximos y verificación de identidad tras conectar ([`postgresql_borradores_configuracion.go`, líneas 17-30](../../../internal/app/bootstrap/postgresql_borradores_configuracion.go#L17-L30), [62-181](../../../internal/app/bootstrap/postgresql_borradores_configuracion.go#L62-L181), [277-297](../../../internal/app/bootstrap/postgresql_borradores_configuracion.go#L277-L297)).

El commit auditado cierra el problema TLS: exige el equivalente a `verify-full` en la configuración principal y todos los fallbacks, limita la excepción sin TLS a loopback/socket Unix de desarrollo y rechaza TLS inferior a 1.2 o intervalos imposibles ([líneas 184-260](../../../internal/app/bootstrap/postgresql_borradores_configuracion.go#L184-L260)).

La CI ejecuta el registro ContextoActor/PDP V3 con PostgreSQL 18.4 fijado por digest ([`.github/workflows/ci.yml`, líneas 30-41](../../../.github/workflows/ci.yml#L30-L41)). El corredor aplica migraciones y prueba concurrencia, identidades `LOGIN`, capacidades negativas y retirada bloqueada con evidencia ([`probar_integracion_contexto_actor_v3.sh`, líneas 72-94](../../../deploy/postgresql/autorizacion/probar_integracion_contexto_actor_v3.sh#L72-L94), [174-499](../../../deploy/postgresql/autorizacion/probar_integracion_contexto_actor_v3.sh#L174-L499)).

No se encontró un uso no de prueba de `NuevosPoolsPostgreSQLBorradores`; por tanto, estas garantías no alcanzan todavía una petición HTTP real.

### AL-04 — La cadena segura de ficheros no está operativa

**Estado:** S3 implementado; análisis de contenido solo contrato; composición inexistente.

**Severidad:** alta.

El registro de almacenamiento permite conectores intercambiables y sondas de capacidades ([`registro.go`, líneas 23-109](../../../internal/vec/adapters/almacen/registro.go#L23-L109)). El adaptador S3 compatible valida HTTPS, buckets separados, cifrado, KMS, retención y redes permitidas, y redacta sus credenciales en cualquier representación ([`configuracion.go`, líneas 38-97](../../../internal/vec/adapters/almacen/s3/configuracion.go#L38-L97), [182-203](../../../internal/vec/adapters/almacen/s3/configuracion.go#L182-L203)). Su transporte no hereda proxy ambiental, restringe destinos por CIDR, exige TLS 1.2 y prohíbe redirecciones ([líneas 225-350](../../../internal/vec/adapters/almacen/s3/configuracion.go#L225-L350)).

La carga directa genera un `PUT` firmado con tamaño, MIME, SHA-256, idempotencia y cifrado; la confirmación verifica la versión exacta, contenido, checksum y cifrado antes de consolidar en cuarentena ([`carga_directa.go`, líneas 33-103](../../../internal/vec/adapters/almacen/s3/carga_directa.go#L33-L103), [106-227](../../../internal/vec/adapters/almacen/s3/carga_directa.go#L106-L227)). Es trabajo reutilizable y no una simulación.

El antivirus está correctamente abstraído: admite ICAP, API corporativa o proceso aislado; declara que MCP no es el transporte de seguridad y exige canal autenticado, cifrado, identidad mutua, firmas actualizadas, malware y contenido activo ([`analisis_contenido.go`, líneas 25-69](../../../internal/vec/ports/analisis_contenido.go#L25-L69)). El navegador nunca entrega el objeto directamente al motor y el resultado queda ligado a la versión en cuarentena ([líneas 72-98](../../../internal/vec/ports/analisis_contenido.go#L72-L98), [165-257](../../../internal/vec/ports/analisis_contenido.go#L165-L257)). No existe un adaptador concreto que implemente `AnalizadorContenido`, ni una composición productiva del flujo.

**Cierre exigido:** seleccionar y probar un motor real, montar S3 → cuarentena → análisis → promoción/retención → expediente/auditoría, incluir fallos, reinicios, resultados no concluyentes, ficheros activos, límites y reconciliación.

### AL-05 — Documento real no equivale aún a documento administrativo firmado

**Estado:** PDF/DOCX implementados; formato genérico y firma por contrato; conectores productivos inexistentes.

**Severidad:** alta.

Los renderizadores PDF y DOCX producen binarios reales. PDF evita duplicar datos personales en metadatos y declara que firma, sello temporal, CSV y registro son pasos posteriores ([`renderizador.go`, líneas 1-3](../../../internal/vec/adapters/documentos/pdf/renderizador.go#L1-L3), [22-78](../../../internal/vec/adapters/documentos/pdf/renderizador.go#L22-L78)). DOCX genera un contenedor Open XML sin macros ni recursos externos ([`renderizador.go`, líneas 1-2](../../../internal/vec/adapters/documentos/docx/renderizador.go#L1-L2), [46-88](../../../internal/vec/adapters/documentos/docx/renderizador.go#L46-L88)).

El modelo de formatos no está limitado a una lista fija: perfiles versionados definen identificador, MIME, extensión, charset, conformidad, tamaño y capacidades ([`formatos_documentales_gobernados.go`, líneas 29-40](../../../internal/vec/domain/formatos_documentales_gobernados.go#L29-L40), [158-303](../../../internal/vec/domain/formatos_documentales_gobernados.go#L158-L303)). No obstante, la persistencia de evidencia del catálogo se reconoce como futura y los únicos renderizadores reales encontrados son PDF y DOCX ([`formatos_documentales_gobernados.go`, líneas 22-40](../../../internal/vec/application/formatos_documentales_gobernados.go#L22-L40)).

Bolsa dispone de contratos exigentes para PAdES Baseline B/T/LTA, revisiones incrementales y sello de tiempo ([`firma_pades.go`, líneas 8-58](../../../internal/modules/bolsa/ports/firma_pades.go#L8-L58), [61-121](../../../internal/modules/bolsa/ports/firma_pades.go#L61-L121)). No se encontró adaptador real de Autofirma/portafirmas, validación PAdES, TSA cualificada o firma múltiple montado en producto. La interfaz de presentación reconoce expresamente que su recibo no crea un fichero ni invoca Autofirma ([`portal-vistas-operaciones.js`, línea 47](../../../web/static/portal-empleado/portal-vistas-operaciones.js#L47)).

**Cierre exigido:** conector AutofirmaV3/portafirmas, validación servidor, TSA autorizada, custodia de cada revisión, CSV/QR verificable, revocación y descarga autorizada; pruebas criptográficas y de recuperación, no solo DTO o PDF visual.

### CR-02 — No hay auditoría global durable ni ciclo completo de privacidad

**Estado:** modelo y registros verticales implementados; garantía global inexistente.

**Severidad:** crítica.

`AuditEntry` contiene actor, perfil, roles, representación, método y garantía de autenticación, autorización, finalidad, acción, recurso, expediente, documento, regla, antes/después, correlación e integridad ([`types.go`, líneas 284-312](../../../internal/vec/domain/types.go#L284-L312)). El único implementador global de `AuditStore` encontrado es el almacén en memoria, que encadena SHA-256 pero se pierde con el proceso ([`store.go`, líneas 175-227](../../../internal/vec/adapters/memory/store.go#L175-L227)). La raíz de desarrollo usa precisamente ese almacén ([`bootstrap.go`, líneas 237-258](../../../internal/app/bootstrap/bootstrap.go#L237-L258)).

Hay registros PostgreSQL append-only y recibos probatorios en verticales concretas, pero no se encontró un registro durable común que cubra todas las lecturas de datos personales, cambios, denegaciones, exportaciones, descargas, firma, administración y operaciones de soporte. Tampoco existe un anclaje externo global que permita detectar la restauración completa de una copia antigua.

En privacidad se observan buenas decisiones locales —finalidad, seudonimización, minimización de DTO, metadatos sin PII—, pero no una implementación operativa integral de inventario de tratamientos, retención y expurgo por expediente, bloqueo/conservación, derechos de las personas, exportación, rectificación, supresión cuando proceda y prueba de ejecución. Las retenciones inmutables de S3 deberán armonizarse con la base jurídica y los plazos aprobados, no configurarse aisladamente.

**Cierre exigido:** registro de accesos y efectos durable, transaccional y consultable; anclaje externo; política versionada de conservación/bloqueo/expurgo; ejecución y evidencia de derechos; revisión formal por DPD/Seguridad y EIPD cuando corresponda.

### CR-03 — Copia, restauración, continuidad y observabilidad no están implantadas

**Estado:** requisitos documentados; implementación operativa no encontrada.

**Severidad:** crítica.

No se encontraron scripts o configuración operativa de copia PostgreSQL, WAL/PITR, pgBackRest/WAL-G, restauración verificada, copias del almacén de objetos, objetivos RPO/RTO, réplica/failover ni simulacro de recuperación. La propia documentación exige copias cifradas, PITR, pruebas periódicas y anclaje externo ([`bolsa_baremacion/README.md`, líneas 267-285](../../../deploy/postgresql/bolsa_baremacion/README.md#L267-L285)) y reconoce que la base viva no detecta una restauración completa o un failover atrasado ([`confianza_atestacion_v2/README.md`, líneas 105-115](../../../deploy/postgresql/confianza_atestacion_v2/README.md#L105-L115)).

Tampoco se encontraron `/livez`, `/readyz`, métricas Prometheus/OpenTelemetry, alertas o cuadros operativos. `/healthz` devuelve siempre `{"status":"ok"}` sin verificar dependencias ([`server.go`, líneas 547-549](../../../internal/app/server/server.go#L547-L549)). Esto impide distinguir proceso vivo de servicio preparado y practicar recuperación con evidencia.

**Cierre exigido:** arquitectura y runbook de copias cifradas, PITR y objetos; restauración periódica en entorno aislado; ancla monotónica externa; RPO/RTO aprobados; `/livez` y `/readyz`; métricas, trazas, logs redactados, alertas, capacidad y simulacros.

### ME-01 — No aparecen secretos reales en el árbol auditado, pero la garantía es incompleta

**Estado:** controles implementados parcialmente.

**Severidad:** media.

El inventario de nombres sensibles de `cae4c29` solo devuelve dos `.env.example`; el de S3 deja vacíos credenciales y claves ([`.env.example`, líneas 1-20](../../../deploy/almacen_s3_ceph/.env.example#L1-L20)). No se encontraron patrones de claves AWS o tokens GitHub. El único encabezado `BEGIN PRIVATE KEY` está dentro de una prueba negativa de detección de datos prohibidos, no contiene una clave ([`catalogos_test.go`, líneas 88-94](../../../internal/vec/adapters/fichero/catalogos_test.go#L88-L94)).

`.gitignore` excluye fuentes locales con posibles datos personales, bases geográficas y material criptográfico ([`.gitignore`, líneas 11-31](../../../.gitignore#L11-L31), [37-59](../../../.gitignore#L37-L59)); `.dockerignore` impide introducir documentos y fuentes locales en el contexto de construcción ([`.dockerignore`, líneas 19-36](../../../.dockerignore#L19-L36)). El generador de desarrollo usa `umask 077`, prohíbe el repositorio como destino y valida permisos, enlaces y separación de claves ([`generar_credenciales_desarrollo.sh`, líneas 1-30](../../../scripts/generar_credenciales_desarrollo.sh#L1-L30), [63-146](../../../scripts/generar_credenciales_desarrollo.sh#L63-L146)).

Esta revisión no escaneó ni certifica todo el historial Git. La CI tampoco contiene una puerta propia de detección de secretos. Por tanto, el resultado correcto es “sin secreto real encontrado en el árbol auditado”, no “repositorio e histórico garantizados limpios”.

**Cierre exigido:** secret scanning y push protection en el alojamiento, escaneo del historial con herramienta aprobada, política de rotación/gestor de secretos y puerta CI que analice diferencias y artefactos sin imprimir valores.

### AL-06 — CI sólida para Go y V3, incompleta como puerta productiva

**Estado:** implementado parcialmente.

**Severidad:** alta.

La puerta canónica comprueba formato, módulos, pruebas, carrera, `vet`, compilación, PDF, vulnerabilidades Go, tamaños y diferencias ([`verificar_calidad.sh`, líneas 7-24](../../../scripts/verificar_calidad.sh#L7-L24)). Las acciones están fijadas por SHA, sin credenciales persistentes y con permisos de solo lectura ([`.github/workflows/ci.yml`, líneas 11-28](../../../.github/workflows/ci.yml#L11-L28)). La integración PostgreSQL V3 está incorporada como segundo trabajo.

Solo existe ese flujo de CI. Faltan, como mínimo:

- todos los corredores PostgreSQL que sean obligatorios para las verticales desplegadas;
- construcción por target y prueba del contenido exacto de cada imagen;
- comprobación automática de dependencias entre superficies;
- E2E de navegador/API sobre una topología productiva efímera;
- escaneo de secretos, SBOM, licencias, imagen/contenedor y firma/procedencia de artefactos;
- prueba de migración, copia/restauración y compatibilidad de vuelta atrás autorizada;
- pruebas de red y rutas cruzadas sobre el Compose ya renderizado.

### ME-02 — Endurecimiento HTTP útil con salud operativa y política web pendientes

**Estado:** implementado parcialmente.

**Severidad:** media.

Las respuestas incluyen no-cache, CSP, aislamiento de origen, política de permisos, no-referrer, HSTS, `nosniff` y anti-frame ([`server.go`, líneas 527-545](../../../internal/app/server/server.go#L527-L545)). El filtrado CIDR falla cerrado ante lista vacía o inválida y usa la dirección del canal, no `X-Forwarded-For` ([líneas 643-714](../../../internal/app/server/server.go#L643-L714)).

Antes del piloto deben separarse salud y disponibilidad, eliminarse la rama Bearer, comprobarse que el artefacto productivo no usa cookies ni almacenamiento de credenciales y reducirse `style-src 'unsafe-inline'` cuando el diseño lo permita. La política CSP actual no constituye por sí sola una vulnerabilidad crítica, pero sí deuda de endurecimiento.

## 5. Cobertura funcional real de Bolsa

La tabla no mide cantidad de código. Mide si cada capacidad puede atravesar hoy la cadena productiva completa.

| Capacidad | Núcleo/contrato | Adaptador real | Montaje productivo | Demostración | Estado honesto y cierre principal |
|---|---|---|---|---|---|
| Consulta pública | Implementado | Fichero marcado como demo; falta proyección PostgreSQL pública | No | Sí | **Demostración.** Crear proyección pública autoritativa, cuenta de solo lectura y raíz exterior. |
| Panel interno de RRHH | Implementado | HTTP y consultas PostgreSQL por componentes | No | Sí | **Implementado por piezas.** Montar tras identidad/PDP en `vec-interno`. |
| Gobierno de convocatorias y borradores | Implementado, T20 avanzado | PostgreSQL, diario, recibos, cifrado/atestación por piezas | No | Parcial | **Implementado por piezas.** Primera vertical candidata a E2E real. |
| Publicación, sustitución, retirada y sucesión | Dominio y transiciones implementados | Persistencia parcial ligada al gobierno | No | Parcial | **Implementado por piezas.** Completar efecto público, firma, registro y recuperación. |
| Bases y reglas de baremo | Dominio/gobierno versionado implementado | Esquema PostgreSQL parcial; su README mantiene NO-GO | No | Sí | **Contrato avanzado.** Completar restaurador/adaptador, publicación y consumo por convocatoria. |
| Autobaremación y cálculo por bases | Implementado y muy probado | Memoria y PostgreSQL por componentes | No | Sí | **Implementado por piezas.** Conectar candidatura, evidencias, versión de bases y expediente. |
| Experiencia oficial de oficio | Dominio y aplicación implementados | No existe fuente corporativa de nómina/RRHH montada | No | No completa | **Contrato avanzado.** Conector autorizado, reconciliación y trazabilidad de procedencia. |
| Revisión técnica, aceptación, rechazo y revocación | Dominio, decisiones y flujo probatorio implementados | Firma/custodia reales ausentes | No | Parcial | **Contrato avanzado.** UI operativa, doble revisión cuando aplique, firma y mutación durable. |
| Llamamientos | Aplicación y HTTP implementados | Transacción PostgreSQL implementada por componentes | No | Sí | **Implementado por piezas.** Montar orden, respuesta, plazos, notificación y auditoría. |
| Contratos y ceses | Modelo parcial heredado/demostración | No se encontró repositorio productivo específico | No | Sí | **Demostración/parcial.** Diseñar agregado, estados, efectos, firma, archivo e integración corporativa. |
| Candidatura, solicitud, registro y alegaciones | Módulo heredado con dominio/casos de uso | Memoria y fichero local; autenticación fake/heredada | No | Sí | **Demostración/parcial.** Integrar en el modelo moderno de identidad, expediente, S3, PDP y PostgreSQL. |
| Documentación, descarga y firma | Contratos fuertes; PDF/DOCX reales | S3 real; Autofirma/TSA/antivirus ausentes | No | Sí | **Contrato/implementación parcial.** Cerrar la cadena documental de AL-04/AL-05. |
| Comunicaciones | Entidades/notificaciones y conceptos de outbox parciales | Sin conectores correo/Telegram y sin entrega operativa encontrada | No | Sí visual | **Contrato parcial.** Outbox durable, plantillas, preferencias, reintentos, evidencias y conectores. |
| Importación Convoca | Lector BIFF8 y caso de uso implementados | XLS real y repositorio en memoria | No | Pruebas sintéticas | **Implementado no autoritativo.** El acta se marca expresamente no autoritativa; falta staging PostgreSQL, aprobación y endpoint interno. |
| Ayuda y accesibilidad documental | Contenido web de presentación | Estático | No | Sí | **Demostración.** Publicar contenido gobernado, búsqueda, audio accesible, versiones y métricas sin PII. |
| Pagos | Dominio, puertos y aplicación por componentes | Sin pasarela real montada | No | No completa | **Contrato.** Solo será necesaria en procesos con tasa; conector, conciliación, devolución y auditoría. |
| API pública | Rutas/DTO públicos implementados | Fuente demo | No | Sí | **Demostración.** Queda incluida en la raíz exterior y su proyección autoritativa. |
| CLI, MCP y bot público | Diseño documentado | No | No | No | **Inexistente en ejecución.** Clientes finos sobre API pública; sin acceso directo a datos internos. |

Evidencias representativas:

- La consulta pública tiene caso de uso y manejador HTTP, pero la fuente disponible es de fichero ([`consulta_publica.go`](../../../internal/modules/bolsa/application/consulta_publica.go), [`httppublico/handler.go`](../../../internal/modules/bolsa/adapters/httppublico/handler.go), [`fichero/convocatorias.go`](../../../internal/modules/bolsa/adapters/fichero/convocatorias.go)).
- Los manejadores internos existen y no tienen consumidores productivos: [`NuevoHandler`](../../../internal/modules/bolsa/adapters/httpinterno/handler.go), [`NuevoHandlerBorradores`](../../../internal/modules/bolsa/adapters/httpinterno/borradores.go#L79), [`NuevoHandlerPropuestasLlamamiento`](../../../internal/modules/bolsa/adapters/httpinterno/llamamientos_http.go#L65).
- El importador limita tamaño y nombre, calcula SHA-256, valida staging y genera acta idempotente, pero etiqueta la procedencia como no autoritativa ([`servicio.go`, líneas 23-43](../../../internal/modules/bolsa/application/importacionconvoca/servicio.go#L23-L43), [81-134](../../../internal/modules/bolsa/application/importacionconvoca/servicio.go#L81-L134)). El lector BIFF8 valida contenedor, hoja, filas, columnas, celdas y cancelación ([`lector.go`, líneas 16-28](../../../internal/modules/bolsa/adapters/xlsconvoca/lector.go#L16-L28), [34-108](../../../internal/modules/bolsa/adapters/xlsconvoca/lector.go#L34-L108)). No aparece en una raíz de composición.
- No se encontraron consumidores no de prueba de `NuevoServicioBaremacion`, `NuevoServicioLlamamientos`, `NuevosPoolsPostgreSQLBorradores` ni `NuevoServicioAutorizacionSolicitudLigadaV3`.

## 6. Pruebas ejecutadas sobre el corte

Las pruebas se ejecutaron desde una exportación limpia de `cae4c29`, no desde el árbol con cambios sin confirmar:

| Comprobación | Resultado |
|---|---|
| `go test ./... -count=1 -timeout 20m` | **Superada** |
| `go vet ./...` | **Superada** |
| `go build ./cmd/...` | **Superada** |
| Grafo `go list -deps ./cmd/vec-publico` | **Al menos 21 paquetes claramente ajenos a la superficie pública** bajo el filtro conservador de CR-01; confirma el hallazgo en el corte |
| Destinos de código y configuración del informe de separación física | **Todos válidos contra `cae4c29`**; el documento de revisión fue añadido después del corte |
| Búsqueda acotada de nombres/patrones de secreto en el árbol | **Sin secreto real encontrado**; un literal negativo de prueba |
| `probar_integracion_contexto_actor_v3.sh` con PostgreSQL 18.4 fijado por digest | **Superada**; concurrencia, identidades, capacidades negativas y retirada con evidencia |

La suite verde acredita los componentes probados. No acredita infraestructura corporativa, identidad real, KMS/HSM, TSA, antivirus, S3 desplegado, recuperación, redes físicas, navegador E2E ni cumplimiento organizativo.

## 7. Orden de cierre bloqueante

El orden reduce retrabajo y permite obtener pronto una vertical real sin rebajar garantías:

1. **C1-C3 exterior:** integrar la frontera pública, artefactos separados y proyección pública PostgreSQL sin datos personales. Añadir las puertas de dependencias y contenido de imagen.
2. **C4-C5 interno:** crear `vec-interno`; montar TLS mutuo, Kerberos+certificado, sesiones durables, revalidación y `ContextActor`. Debe fallar antes de escuchar si falta una dependencia.
3. **Primera vertical real:** conectar borradores/gobierno de convocatorias T20 con los tres pools PostgreSQL TLS, KMS, PDP V3, registro y manejadores HTTP. Probar reinicios, concurrencia, errores ambiguos y recuperación.
4. **Cadena documental:** S3/cuarentena, antivirus real, promoción, PDF/DOCX, Autofirma/portafirmas, validación, TSA, CSV/QR, descarga y revocación autorizada.
5. **Auditoría y privacidad:** registro global durable de accesos/efectos, anclaje externo, retención/expurgo y derechos con evidencia.
6. **Operación:** copias cifradas, WAL/PITR, restauración probada, objetos, RPO/RTO, `livez`/`readyz`, métricas, alertas y runbooks.
7. **CI productiva:** todos los corredores necesarios, artefactos, dependencias, SBOM/secretos/contenedores, topología y E2E.
8. **Replicar el patrón:** reglas/bases → autobaremación/revisión → llamamientos → candidatura/alegaciones → comunicaciones → contratos/ceses.
9. **C7-C10:** administración privilegiada, despliegue final, MCP/CLI y retirada de cualquier ruta o artefacto de demostración.
10. **Validación institucional:** pruebas de aceptación con RRHH y Sistemas, análisis ENS/RGPD, EIPD si procede, continuidad, accesibilidad y autorización formal antes de datos reales.

## 8. Condición para cambiar el veredicto

El NO-GO solo puede revisarse cuando, al menos, una topología productiva efímera demuestre de extremo a extremo:

- superficies y artefactos físicamente separados;
- identidad real, sesión durable, revalidación, PDP V3 y mínimo privilegio;
- datos autoritativos PostgreSQL con TLS verificado y roles exactos;
- documentos en S3 cifrado, cuarentena y antivirus real;
- firma/validación/sello/CSV cuando el trámite lo exija;
- auditoría durable y anclada de acceso y efecto;
- copia y restauración probadas;
- salud, disponibilidad, métricas y alertas;
- pruebas automáticas de rutas, redes, imágenes, migraciones y recuperación;
- ausencia verificable de adaptadores, datos o selectores de demostración en los artefactos productivos.

Hasta entonces, la formulación correcta es: **“núcleo amplio y probado por componentes; demostración funcional disponible; integración productiva y garantías operativas todavía en construcción”**.
