# Manual mantenido del programador de VEC

## Continuación técnica vigente — 10 de septiembre de 2026

Base del cierre funcional `00558603dbd3040eacb03cb511f3b840be241b10`. Una sola línea:
`trabajo/ct-app-llamamiento-b4a-20260905`; producto publicado en
`integracion/ct-producto-ligero-20260821`. El desarrollo activo está en el
servidor y conserva trabajo pendiente ajeno. No programe en la raíz histórica
local ni copie implementaciones entre ramas. Antes de integrar, inventaríe
ramas, worktrees, diferencias y equivalencias de parches.

La capacidad cerrada es resolución manual sintética y acceso posterior a
los documentos. La API registra con autorización y PostgreSQL, devuelve
`201` al crear y recupera con `200`; repetir la misma clave no crea otro
recibo. La prueba integrada conservó resultado tras reiniciar app/base.
Los campos `firma_oficial` y `eficacia_administrativa` permanecen en `false`.

La interfaz permite descargar desde el detalle `v8` los seis PDF de la
propuesta histórica `v7`. No debe solicitar el documento con la versión
actual ni inventar otra propuesta. La resolución descargada conserva SHA256
`e9b53a1e2196b3719770cf7db9f563077d75f457669e84cb7bb3c32fd1de4130`.

La canónica `c6359f98` ya integra el preparador durable, lectores, montaje y
carga de configuración; no reconstruir esas piezas. CT70–85, trece pools y
tres capacidades con catálogos y definición están provisionados, sin precargas
de negocio. El formulario real registró solicitud `7`/expediente `8`, periodo
`2027Q1`, seguimiento `0→1` y un recibo persistente. La transacción dejó una
alta, relación y ocupación en Personal, y una incorporación, raíz y dos estados
en CT, con auditorías y outbox en ambos lados.

Separar integración de evidencia visible: tras reiniciar aplicación y
PostgreSQL, la recuperación quedó bloqueada por `GET 503` en la restauración
histórica Auth12. Auth13 tiene dos revisiones `GO` y regresión verde en
`ef6704a6`, pero no está instalada. No hay ciclo completo acreditado ni debe
repetirse la escritura. `firma_oficial` y `eficacia_administrativa` siguen en
`false`; la métrica queda en cinco pasos completos y partes de 6/7/8 hasta la
recuperación. GINPIX es el siguiente corte existente aún sin cierre visible.

Consulte [operación actual](../manual_sistemas/README.md#entorno-privado-vigente).
El cierre funcional citado no identifica por sí solo el artefacto servido
actualmente, ni lo hace el HEAD del árbol compartido con trabajo pendiente.
Pruebas focales al terminar el hito; documentación no requiere
repetir pruebas de producto ya acreditadas.

## Contexto y recorridos anteriores

Las ubicaciones locales y cifras del 6 de septiembre conservadas a
continuación son antecedentes, no instrucciones para el servidor actual.

Guía práctica para continuar el desarrollo sin reconstruir piezas existentes.
Se mantiene a mano; no es el catálogo de firmas ni un certificado de despliegue.
Estado funcional de referencia: 6 de septiembre de 2026.

La consulta organizativa de Contratación reutiliza `CatalogoConfigurable`
del núcleo mediante `ConsultaCatalogosConfigurables`; Personal proyecta
`ConsultaEstructuraOrganizativa` sin otro catálogo, repositorio ni permisos.
`contratacion_temporal_organizacion_desarrollo.go` conecta la lectura
protegida y la vista `portal-empleado/organizacion/` consume su API real.
La ruta revalida certificado y rol con la frontera ya existente.

Las entradas tienen clave estable, tipo y adscripción opcional; los nombres
de los cargos no son enumeraciones compiladas. Se rechazan ciclos y padres
ausentes, sin inferir dependencia funcional por el orden del PDF. El lector
está fijado a versión explícita; no resuelve «la última». El paquete inicial
está en borrador y conserva las limitaciones de la extracción pública.
El editor reutiliza `CatalogoConfigurable.ActualizarBorrador`: cambia una unidad
por operación y conserva la identidad del catálogo. El adaptador PostgreSQL
de Personal es su fuente única cuando la composición lo selecciona; el
fichero original solo sirve para inicialización explícita.

`POST /api/vec/contratacion-temporal/organizacion/cambios` recibe versión,
revisión, huella esperada, clave de idempotencia, unidad y motivo. No admite
identidad ni permisos del cliente. La frontera obtiene el actor del certificado
y liga una autorización nueva a todo el material de cada petición, incluso
al recuperar un recibo. La transacción guarda revisión, recibo y evento juntos;
la misma operación devuelve el recibo original, no otra revisión.

Ante un resultado incierto, el formulario conserva exactamente cuerpo y clave
en memoria para reintentar. No usa almacenamiento web. Una revisión caducada
devuelve `409` y requiere recargar antes de preparar otro cambio.
La aprobación administrativa y conexión al alta/ratificación siguen pendientes;
no reutilice esta tabla como fuente de permisos.

**Cierre de bandeja y análisis publicado:** `b2effbaf09fd4ad8477bf42c56e4615ff52d0c62`.
La base principal conserva 51 solicitudes; bandeja y detalle consultables en `8443`/base
`55433`; el caso verificado encadena solicitud `v1` a análisis `201`/`v2` y
recupera un único recibo tras reinicio, mediante lectura independiente de
PostgreSQL y navegador. La bandeja no añade por sí sola otro paso completo.
Consultar la [guía canónica](../../GUIA_RECORRIDO_ALBERTO.md) para el recorrido
operativo.

**Incluido en esta entrega; recuperación demostrada:**
la declaración RRHH obtuvo `HTTP 201`. Corregido el defecto temporal AD3 en
ambas bases, dirección confirmó en navegador `200/200/200` para selección,
comunicación y respuesta tras el segundo reinicio de aplicación y PostgreSQL:
mismos recibo, justificante y fecha, sin nuevo registro. Sin errores JavaScript,
cookies, almacenamiento web ni desbordamiento horizontal en ese recorrido.
También se confirmó un conflicto real `409` en navegador. La entrega 3 queda
cerrada funcionalmente en desarrollo, sin nuevo paso completo.

**Corte 4 publicado en `17ea874`:** aceptación manual sintética `201`
con API/V3/CT58/Bolsa4 reales; tras reiniciar app/PostgreSQL principal,
`200/200/200/200`, mismo recibo y fecha, sin duplicados. Cierre técnico, no política
legal aprobada. Aquel corte mantenía **5/8 pasos completos más parte del sexto**.

**Corte 5 incluido en esta entrega:** renuncia manual sintética con servicios y
permisos reales, navegador `200/201/201/201` y recuperación `200/200/200/200`
tras reiniciar app/PostgreSQL principal. Mismos recibo, resolución, auditoría,
fecha e intención pendiente, sin duplicados ni siguiente ejecutado. Cero errores
JS, cookies, almacenamiento web y desbordamiento. Objetivo 5 cerrado funcionalmente;
criterio manual provisional solo sintético, sin aval legal ni del operador.

**Objetivo 7 incluido en esta entrega:** quinta operación `201` real tras renuncia
y `200/200/200/200/200` tras reiniciar app/PostgreSQL principal. Mismos 14 campos
salvo `estado_local: replay_confirmado`, sin duplicados ni errores JS, cookies,
almacenamiento web o desbordamiento. Cierre funcional solo tras renuncia sintética;
ese corte mantenía **5/8 más parte del sexto**, sin aviso al sucesor ni plazo legal.

**Objetivo 8 cerrado funcionalmente en desarrollo:** Chrome `201` y, tras reiniciar
app/PostgreSQL principal, cuatro antecedentes `200` y propuesta `200`, mismos
identificadores/recibo/fecha/v7; historia previa intacta. Cero errores JS, cookies,
almacenamiento web y desbordamiento. Esta revisión incorpora el cierre funcional;
el hash publicado se comprueba en Git.
AD3-20/CT61 instaladas en ambas bases, no reaplicar; E2E acreditado solo en principal.

**Objetivo 9, primer informe borrador demostrado:** Chrome `200`, 29267 bytes.
Rectificación: primer fallo navegador sin estado capturado; el `502` era curl con
límite 50, paginación separada sin corrección acreditada. Navegador límite 100:
sonda `200`, vista `404`; PostgreSQL `42501` por `.999340Z` frente a `.99934Z`.
AD3-21 corrige las dos comparaciones de lectura heredadas de AD3-3/5 a instantes;
AD3-14 había corregido mutaciones. No modifica firmas, guardas ni cursor. Instalada
en ambas bases, no reaplicar; UP/DOWN exacto en ROLLBACK en secundaria.
Tras el parche, cinco POST de bandeja/detalle/PDF `200`, mismo PDF y cero errores JS,
cookies y almacenamiento web. Tras reiniciar app/PostgreSQL principal: cinco POST `200`,
sin `404`, mismo PDF e historia CT/Bolsa conservada; no cierra la paginación con límite 50.
Dirección inspeccionó PDF y pantalla estable de 390 px sin obstrucción; la captura
previa era transición CSS de 180 ms, sin cambios de UI ni validación de usabilidad global.
Esta revisión añade resolución borrador: Chrome `200`, 29770 bytes, informe original
idéntico en la misma sesión; seis POST `200`, cero errores JS, cookies y almacenamiento web.
Dirección inspeccionó PDF y pantalla estable de 390 px con dos botones. Tras reiniciar
aplicación/PostgreSQL principal: otros seis POST `200`, ambos PDF idénticos en tamaño
y SHA256, historial y recibos anteriores conservados; cero errores JS, cookies y
almacenamiento web. Tercer borrador, diligencia: 28366 bytes; siete POST `200` antes
y después de reiniciar aplicación/PostgreSQL principal, los tres PDF e historial previo
idénticos, cero errores JS, cookies y almacenamiento web. Dirección inspeccionó PDF
y pantalla estable de 390 px con tres botones. Toma de posesión comprobada también tras
reinicio principal: ocho POST `200`, cuatro PDF e historial idénticos; evidencia en la guía.
Disponibles **6/6**: comunicación al centro comprobada también tras reinicio principal.
Siguiente objetivo 10, circuito de firma/evidencia admitida. Sin envío, entrega, plazo legal
ni orden de incorporación; evidencia central en la guía.

## Qué leer y qué mantener

Este manual puede consultarse desde un clon del repositorio, sin acceder a
archivos privados de otra máquina. Lea las instrucciones aplicables al
directorio y las instrucciones vigentes del operador cuando se aporten.
La evidencia del recorrido y las órdenes actuales prevalecen sobre cifras
o bloqueos históricos. Las reglas operativas de este corte son:

- Priorizar avances observables de Contratación temporal, conectando las
  piezas existentes antes de perfeccionarlas o reconstruirlas.
- Las preguntas pendientes no detienen programación independiente por orden
  del operador; no autorizan a inventar plazo, autoridad o aceptación.
- Mantener una línea canónica por capacidad y archivos de edición disjuntos;
  conservar el trabajo ajeno, sin copiar implementaciones entre ramas.
- Usar solo datos sintéticos, autorización de servidor y efectos persistentes
  reales; no presentar un adaptador DEMO como éxito ni usar cookies o
  almacenamiento web para el recorrido.
- Probar al completar cada hito, con alcance proporcionado. Exigir dos
  revisiones independientes en identidad, autorización, criptografía, SQL y
  fronteras de datos personales; no repetir suites globales por cada edición.
- No abrir el runner ni nuevas cadenas de documentos de decisión por inercia.
  Documentar el avance en los manuales y la guía existentes, sin convertir
  una mejora menor en una reescritura.

Este README explica arquitectura, conexión y trabajo cotidiano. El
[catálogo autogenerado LEEME.md](LEEME.md) sirve para localizar firmas y
comentarios Go; no debe editarse manualmente. Su advertencia histórica de no
editar este directorio se refiere al material generado, no a este README
manual expresamente separado.

Los cuatro perfiles documentales tienen finalidades distintas:

| Perfil | Entrada | Qué debe resolver |
| --- | --- | --- |
| Usuario | [Manual de usuario](../manual_usuario/manual_portal_bolsas.md) | Acceder, navegar, realizar acciones y entender estados. |
| Recursos Humanos | [Manual de RRHH](../manual_rrhh/README.md) | Recorrer el procedimiento, responsabilidades y límites funcionales. |
| Programador | Este README | Localizar, modificar y conectar las piezas reales. |
| Operador | [Manual de sistemas](../manual_sistemas/README.md) | Preparar, arrancar, parar, respaldar y recuperar el entorno. |

Los manuales distinguen funciones demostradas y pendientes; su existencia
no acredita por sí misma una entrega funcional.
La [guía del recorrido demostrado](../../GUIA_RECORRIDO_ALBERTO.md)
conserva los comandos operativos y los recibos sintéticos. No duplicar aquí
sus credenciales, recetas de instalación ni registros de cada ejecución.

### Cómo consultar el catálogo generado

El [generador](../../scripts/generar_manual_programador.py) ejecuta
`go list ./...` y `go doc -all`, y sobrescribe `LEEME.md` y nueve archivos
de áreas enumerados en `AREAS`. No escribe este `README.md`.

El índice conservado no incluye Contratación temporal. Además, el generador
agrupa los módulos distintos de Bolsa en `modulos_personal_cronos_dietas.md`,
aunque el título no los enumere todos, y omite un paquete si falla su
`go doc`. Por tanto, ausencia en el catálogo no prueba ausencia de código;
presencia tampoco prueba que esté conectado o probado en navegador.

Para una consulta puntual, desde la raíz del worktree:

```bash
rg -n 'NuevasRutas|NuevoManejadorSeleccionLlamamiento' internal/app internal/modules/contrataciontemporal
go doc vec-diputacion-granada/internal/modules/contrataciontemporal/ports
```

Solo si el hito incluye actualizar la referencia generada:

```bash
python3 scripts/generar_manual_programador.py
git diff --stat -- docs/manual_programador
```

Revisar el resultado completo: no ejecutar el generador por cada edición de
este manual ni confiar en sus resúmenes históricos como estado del producto.

## Estado funcional: cinco pasos y partes del sexto y séptimo

El recorrido usa navegador, API interna, autorización, PostgreSQL y recibos
reales con datos sintéticos. La declaración RRHH tiene su comprobación propia
de recuperación tras el segundo reinicio, además del cierre anterior:

| Paso del recorrido mantenido | Alcance demostrado |
| --- | --- |
| 1. Solicitud | Alta de una petición real. |
| 2. Análisis | Registro del análisis de Recursos Humanos. |
| 3. Bolsa | Propuesta y decisión de cobertura del expediente. No toda la aplicación Bolsa. |
| 4. Asignación | Envío del expediente a la unidad. |
| 5. Informe jurídico y fiscalización | Registro durable y resultados de fiscalización; devolución a unidad cuando corresponde. |
| 6. Llamamiento, parcial | Recorridos sintéticos, aviso CT62, declaración CT63 y aceptación manual del sucesor CT64 recuperables tras reinicio principal. Faltan vencimiento, envío corporativo y plazo. |
| 7. Nombramiento, parcial | Propuesta desde aceptación sintética `201` y replay `200` tras reinicio; propuesta `v7` y resolución manual sintética `v8` con recibo recuperable. Los PDF conservan la versión documental `7`. Sin firma ni nombramiento eficaz. |
| 8. Incorporación y seguimiento | No declarados completos de extremo a extremo; GINPIX sigue pendiente. |

El aviso local no demuestra correo enviado, entrega al destinatario, aceptación,
renuncia ni inicio de plazo. Una intención pendiente de salida (`outbox`) no
es un acuse del sistema externo. El contador es **cinco pasos completos más
partes del sexto y séptimo**, no un porcentaje global de aplicación terminada.

La base del servidor conserva 52 expedientes sintéticos, con bandeja y detalle
consultables por el acceso privado de Sistemas; no implica 52 filas visibles
simultáneas. El alcance sigue siendo cinco
pasos y partes del sexto y séptimo; la bandeja no se cuenta como paso adicional.
Un `503` debe explicarse como dependencia no disponible, no sustituirse por
datos de presentación. Un `404` del panel de Bolsa tampoco demuestra que
haya fallado la identidad de Contratación temporal.

No confundir la instalación parcial de migraciones, una compilación correcta,
el menú visible o `/livez` con un recorrido completo. El cierre exige evidencia
de la versión que realmente se arrancó.

### Declaración de respuesta recibida por RRHH

El formulario de llamamiento existente incorpora la tercera operación tras
recuperar la comunicación confirmada `v2`; deriva de su recibo los antecedentes.
El cliente `registrarRespuestaRecibida` usa
`POST /api/vec/contratacion-temporal/llamamientos/respuestas/registro`, con
respuesta `{data: respuesta}` y estados `registrada_por_rrhh` /
`replay_registrada_por_rrhh`. No devuelve `version_resultante`: la comunicación
permanece en `2`, el expediente observado en `6` y Bolsa no cambia de estado.

La entrada contiene `aceptacion` o `renuncia` declaradas, referencia opaca del
correo, huella SHA-256 y recepción UTC. El `.eml` (no vacío, hasta 2 MiB) se
procesa con WebCrypto local: solo se transmite la huella, nunca su contenido
ni su nombre. La confirmación es explícita y un resultado ambiguo conserva
el intento y la clave; no se usa almacenamiento web ni cookies.

La acción propia es `contratacion_temporal.llamamiento.respuesta.registrar`;
la autoridad de servidor vincula actor, perfil y material completo, también
para recuperar el mismo registro. No se reutiliza el permiso de comunicación
ni se añade identidad al JSON. Se conservan actor, recibo y justificante de la
declaración, sin verificar origen, firma o custodia del correo, envío, entrega
ni aceptación o renuncia terminal. El original sigue en el sistema de correo.

CT `000056_respuesta_recibida_rrhh` y AD3
`000014_consumidor_respuesta_recibida_rrhh` ya estaban instaladas en **ambas bases
locales en el corte histórico**. No reaplicarlas ni recrear bases. Las once DSN
de aquel corte no completan la configuración de incorporación actual.
Los DOWN bloquean la reversión cuando
hay registros o dependencias; no eliminan la declaración conservada.

La corrección puntual de AD3-14 compara `valida_hasta` y
`decision_valida_hasta` como instantes (`::timestamptz`): la decisión tiene seis
decimales y la capacidad RFC3339Nano omite ceros finales. No cambia bytes
firmados, hashes, MAC ni permisos. Dirección aplicó el bloque literal
`DO $fechas$` en ambas bases; sus tres regresiones pasaron. La instrumentación
de diagnóstico CT56 está retirada. No reaplicar la migración ni reconstruir
el núcleo; el [manual de Sistemas](../manual_sistemas/README.md) distingue aquel
corte histórico de la huella conservada al cierre de AD3-18.

El [manual de RRHH](../manual_rrhh/README.md) recoge el ejemplo y el recibo
observado; la [guía](../../GUIA_RECORRIDO_ALBERTO.md) conserva el recorrido vigente.

### Resolución de aceptación o renuncia manual sintética

La cuarta operación del formulario es `data-ct-llamamiento-form="resolucion"`.
Tras un recibo de declaración `aceptacion` o `renuncia`, `resolverLlamamiento` envía a
`POST /api/vec/contratacion-temporal/llamamientos/resoluciones` los ocho campos
base: `clave_idempotencia`, `organizacion_ref`, `expediente_ref`,
`llamamiento_ref`, `comunicacion_ref`, `version_esperada`, `respuesta` y
`prueba_respuesta_ref`. Deriva los antecedentes del recibo, usa versión `2`,
`justificante_ref` como prueba y una clave propia; tampoco permite cambiar la
respuesta derivada del recibo ni recibe autoridad del DOM.

La UI añade, en ese orden, `revision_respuesta_rrhh: true`, `revision_plazo_rrhh: true`
y `criterio_validacion_ref: "politica:ct:revision-manual-sintetica:20260906"`.
Son once campos canónicos: solo envía tras marcar ambas casillas inicialmente falsas;
criterio de solo lectura, confirmación explícita, sin identidad ni otro `.eml`.
Los ocho campos antiguos siguen admitidos y devuelven `409 validacion_respuesta_pendiente`,
clave i18n `api.contratacion_temporal.comunicacion_llamamiento.error.validacion_respuesta_pendiente`.
Ese rechazo conocido sin efectos permite corregir casillas conservando la clave;
ante resultado ambiguo se congelan clave y material, sin reintento automático.

El resolutor consulta el justificante mediante CT57 con permiso propio V3 real
y fresco (`contratacion_temporal.llamamiento.respuesta.consultar_justificante`),
sin doble. El resultado es interno, no DTO HTTP; `Seleccion` no sale.
AD3 `000016` / CT `000057` instaladas en ambas bases locales (`55433`/`55432`),
con consulta confirmada tras reinicio de app/PostgreSQL principal; no concede resolución.

La composición de desarrollo de doble llave fija la política sintética y exige
`contratacion_temporal.llamamiento.respuesta.validacion_manual.registrar` para
guardar declaración, actor y política en CT; Bolsa consume su permiso separado:
`bolsa.llamamiento.aceptacion_rrhh.registrar` o
`bolsa.llamamiento.renuncia_rrhh.registrar`, sin intercambiarlos.
No deriva plazo legal del aviso ni acredita entrega. Solo devuelve el recibo HTTP
tras confirmar CT y Bolsa: aceptación conserva los nueve campos y renuncia añade
`intencion_siguiente` con exactamente `referencia`, `estado_local: "pendiente"`
y `actualizada_en` UTC. La intención es obligatoria en renuncia y está prohibida
en aceptación, incluso como `null`; versión resultante `3` en ambas.
La UI valida la fecha de intención no anterior a la resolución, conserva
precisión microsegundo y muestra «Vigente solo para el ejercicio sintético».
CT59 guarda la intención y su carga real en la misma fila de resolución;
se devuelve la misma al recuperar, no se despacha otro llamamiento.
Si Bolsa falla después del commit CT, no hay éxito parcial: reintento explícito
con la misma clave/material, recibo CT conservado y autorizaciones frescas.
Los tres catálogos de motivos se publican en llamadas separadas, no como uno solo.

AD3 `000015` / Bolsa `000004` y AD3 `000017` / CT `000058` están **instaladas
en ambas bases**, con ambas aplicaciones en la compilación corregida; no reaplicar.
La prueba aislada anterior de Bolsa4 (`8197db3`) comprobó almacenamiento/replay/conflictos
con doble privado transaccional, no criptografía. El roundtrip de aceptación UP/DOWN de
las cuatro migraciones verificó reversión exacta en ROLLBACK, sin modificar
autorización ni usar dobles. El recorrido navegador sí usó criptografía y V3 reales:
`201` y replay `200` tras reinicio principal, misma fecha/recibo, una resolución CT58,
una aceptación Bolsa, tres historias y tres eventos, sin duplicados.
AD3 `000018` / Bolsa `000005` / CT `000059` están instaladas en ambas bases;
dirección confirmó UP/DOWN con ACL, funciones y comprobaciones conservadas.
El navegador registró además renuncia `201`, recuperada `200` tras reinicio con
los mismos identificadores, auditoría, fecha e intención pendiente. Totales
confirmados: dos filas CT, seis registros Bolsa (dos órdenes, dos propuestas,
una aceptación y una renuncia), seis historias/eventos; sin duplicados y con la
aceptación/declaración anteriores intactas. No reaplicar ni ejecutar DOWN sobre esos datos.
El [plan canónico](../../ESTADO_PROYECTO.md) cierra técnicamente la aceptación manual
sintética y funcionalmente la renuncia y propuesta de desarrollo. No hay política
legal aprobada ni cierre de vencimiento,
envío corporativo ni plazo; aviso, declaración y resolución manual sintética del sucesor se describen debajo.

### Continuación tras renuncia sintética

Quinta operación `data-ct-llamamiento-form="siguiente"`, cliente `continuarLlamamiento`:
`POST /api/vec/contratacion-temporal/llamamientos/siguientes`, cinco campos canónicos
en orden: `clave_idempotencia`, `organizacion_ref`, `expediente_ref`, `resolucion_ref`,
`intencion_ref`. Antecedentes del recibo de renuncia; solo clave propia, confirmación
explícita y reintento manual con igual material. `409` no permite otra clave.
HTTP devuelve 14 campos, versión de nuevo llamamiento `1`, intención `despachada`;
`201 confirmado` o `200 replay_confirmado`, mismos recibos, auditoría y fecha.
Consulta de justificante, continuación CT y apertura Bolsa mantienen permisos
separados. Éxito solo tras Bolsa y confirmación CT; un fallo entre ambos no es éxito
parcial ni autoriza compensación. AD3-19/Bolsa6/CT60 soportan este recorrido.
El puente deriva una referencia Bolsa alfabética de 256 bits desde org/exp/resolución/
intención CT y la coteja al retornar; CT conserva su referencia original y operación
estable. Corrige el rechazo accidental del UUID sin relajar validadores.
El recibo original de renuncia conserva su intención pendiente histórica; el nuevo
confirma continuidad posterior. No se conecta automáticamente a la comunicación
del sucesor ni se expone identidad/posición. [Recuperación exacta](../../GUIA_RECORRIDO_ALBERTO.md#objetivo-7-recuperar-la-continuación-tras-renuncia).

### Aviso local al sucesor, sin nueva ruta

Operación `comunicacion_siguiente`, mismo `registrarComunicacionLlamamiento` y
la ruta publicada de comunicaciones, sin endpoint nuevo. Estado independiente;
antecedentes del recibo CT60 validado, no del DOM. Añade al final del JSON
`tipo_antecedente: "continuacion_confirmada"`; la primera comunicación conserva sus seis campos exactos.
Solo clave propia editable y confirmación explícita; conflicto bloquea, ambigüedad congela
clave/material. Habilita una declaración separada, sin rearmar la resolución anterior. CT62 coteja CT60 con permiso fresco,
política local v2 y selección padre; no lee/escribe Bolsa. Versión resultante `2`, aviso JSON v2 y outbox pendiente
con `recibo_continuacion_ref`; no envío ni plazo. CT62 instalada en ambas bases, no reaplicar/DOWN.
`201`, negativos `403` sin efectos y recuperación `200` tras reinicio principal, mismo recibo/fecha/v2/outbox.
[Evidencia](../../GUIA_RECORRIDO_ALBERTO.md#aviso-local-al-sucesor-ct62).

### Declaración del sucesor, mismo contrato de diez campos

`data-ct-llamamiento-form="respuesta_siguiente"` comparte configuración, validador y método
`registrarRespuestaRecibida` con `respuesta`: mismo POST `/api/vec/contratacion-temporal/llamamientos/respuestas/registro`, diez campos,
sin `tipo_antecedente` adicional. Estado y lectura `.eml` separados, renderer/calculador comunes e IDs únicos.
Solo aviso local validado `v2`; antecedentes de solicitud/recibo `comunicacion_siguiente`, nunca del DOM.
Clave propia, confirmación explícita, huella en RAM sin contenido; ambigüedad congela material, sin retry automático.
CT63 adapta solo la función CT56: consumo fresco antes del replay exacto material/actor/perfil,
cadena CT62/CT60 verificada antes de omitir las dos comparaciones con el llamamiento/recibo raíz. Sin efectos Bolsa.
Instalada en ambas bases; no reaplicar ni DOWN con historial. `201` real y cruces `409` sin efectos;
siete operaciones `200` tras reinicio principal, mismos justificante/recibo/auditoría/fecha e historia;
solo cambia `estado` a `replay_registrada_por_rrhh`. [Evidencia y recuperación](../../GUIA_RECORRIDO_ALBERTO.md#declaración-de-respuesta-del-sucesor-ct63).
No resuelve automáticamente al sucesor ni altera la resolución anterior.

### Resolución manual del sucesor, mismo contrato de once campos

`data-ct-llamamiento-form="resolucion_siguiente"` comparte configuración, validadores, cliente y renderer
con la resolución original; estado separado, antecedentes del justificante CT63 validado, nunca del DOM.
Dos revisiones inicialmente falsas, política sintética fija y confirmación explícita; sin otro `.eml`.
CT64 liga aviso/continuación CT60 y conserva selección raíz íntegra; apertura Bolsa real y permisos propios,
sin éxito hasta confirmar CT y Bolsa. En CT64, versión resultante `3`, expediente `6`; sin propuesta ni tercer llamamiento automáticos.
Primer `503` dejó CT durable y Bolsa pendiente: excepción de formato solo para `EvaluacionPlazoRef`
`evaluacion:UUIDv4`, reutilizando el validador puro existente, sin relajar las otras referencias ni regenerarla.
Recuperación misma clave: ocho POST `200` antes y después del reinicio principal, mismos recibo/fecha/evaluación/auditoría;
tres resoluciones CT y ocho operaciones/historias/outbox Bolsa, anteriores intactos.
CT64 instalada en ambas bases; no reaplicar ni DOWN con resolución sucesora.
[Ficha y recuperación](../../GUIA_RECORRIDO_ALBERTO.md#resolución-manual-del-sucesor-ct64).
La propuesta posterior CT65 ya es recorrible, sin envío, plazo legal ni firma.

### Propuesta de nombramiento desde aceptación

Aceptación original o sucesora confirmada: operación única `data-ct-llamamiento-form="propuesta"`, método `prepararPropuestaFormalizacion`:
reutiliza `POST /api/vec/contratacion-temporal/formalizacion/propuestas`, once campos
de solicitud y seis de recibo, sin otro DTO ni autoridad del DOM. Antecedentes
de aceptación CT y terminal Bolsa real; nunca renuncia. Versión esperada `6`,
también en replay desde agregado `7`; clave propia y confirmación explícita.
CT65 conserva el recibo de aceptación correspondiente para referencias y comparación temporal;
clave de propuesta distinta de las ocho anteriores, material congelado ante ambigüedad, sin autoenvío.
Ocho antecedentes `200` y propuesta `201`; nueve `200` tras reinicio principal, mismos recibo/fecha/v7 e historia.
Consulta interna con `Continuacion` opcional y selección raíz íntegra; apertura y terminal Bolsa originales,
sin otro DTO web. CT65 instalada en ambas bases; no reaplicar ni DOWN con propuesta sucesora.
El ajuste CT60 liga `soloRecuperacion` a versión actual mayor que `6` y exige apertura existente
antes del servicio, conservando permiso fresco y replay completo; no permite un nuevo llamamiento desde v7.
[Novena operación del panel y evidencia](../../GUIA_RECORRIDO_ALBERTO.md#propuesta-desde-la-aceptación-del-sucesor-ct65);
no es otro paso RRHH. Propuesta original y antecedentes conservados; seis PDF cerrados, no repetidos.
Las cuatro publicaciones proceden del único asset `formalizacion-desarrollo.json`,
con SHA256 de contenido UTF-8, carga sin credenciales ni caché y fallo cerrado.
El permiso propio `contratacion_temporal.formalizacion.propuesta.registrar` liga
material completo y etapas; CT61 consume V3 fresco antes de lectura/replay.
El commit une propuesta, versión integral `7`, actuación y outbox; replay conserva
material/actor/perfil y recibo/fecha. DOWN bloqueado con historia. Sin firma,
renderizado documental ni nueva aceptación Bolsa. [Claves y evidencia](../../GUIA_RECORRIDO_ALBERTO.md#objetivo-8-recuperar-la-propuesta-de-nombramiento).

### Seis borradores desde la misma consulta RRHH

Objetivo 9 reutiliza `POST /api/vec/contratacion-temporal/expedientes/consultas`:
JSON de dos campos derivados: `expediente_ref` y `version_observada: 7`; `Content-Type: application/json`,
el tipo solo selecciona una de las seis representaciones cerradas:

| Tipo | Accept | Nombre attachment |
| --- | --- | --- |
| `informe_definitivo` (predeterminado) | `application/pdf; documento=informe-definitivo-desarrollo` | `informe-definitivo-borrador.pdf` |
| `resolucion` | `application/pdf; documento=resolucion-desarrollo` | `resolucion-borrador.pdf` |
| `diligencia` | `application/pdf; documento=diligencia-desarrollo` | `diligencia-borrador.pdf` |
| `toma_posesion` | `application/pdf; documento=toma-posesion-desarrollo` | `toma-posesion-borrador.pdf` |
| `notificacion` | `application/pdf; documento=notificacion-desarrollo` | `notificacion-borrador.pdf` |
| `comunicacion_centro` | `application/pdf; documento=comunicacion-centro-desarrollo` | `comunicacion-centro-borrador.pdf` |

Consulta autorizada/auditada antes del renderizado, detalle `v7/nombramiento/en_curso`
y hito 7 `registrar_propuesta_formalizacion`; generador PDF existente, sin otra fuente.
Salida `200 application/pdf`, attachment nominal según tabla, máximo 2 MiB.
Errores JSON de consulta y `409 documento_no_disponible`; sin PDF parcial ni replay de propuesta.
Botones de cabecera con `data-ct-exp-accion`: `descargar-informe-definitivo`,
`descargar-resolucion`, `descargar-diligencia`, `descargar-toma-posesion`, `descargar-notificacion`
y `descargar-comunicacion-centro`.
El mismo archivo `cliente-http-informe-definitivo.js` ofrece
`crearClienteHTTPBorradorRRHH().descargarBorrador(solicitud, {tipo, signal})`.
Manejador, exclusión de descargas simultáneas, cancelación y revocación de Blob compartidos;
el error conserva el detalle, sin almacenamiento ni reintento automático. Sin nuevo manifiesto.
PDF sin SQL propio; AD3-21 corrige la lectura existente. AD3-20/CT61 y AD3-21 instaladas
en ambas bases, no reaplicar. [Recorrido y evidencia](../../GUIA_RECORRIDO_ALBERTO.md#objetivo-9-descargar-el-primer-informe-borrador).
Disponibles **6/6 borradores**; la resolución manual sintética del objetivo 10
ya es recuperable, pero su circuito de firma oficial sigue pendiente;
**5/8 más partes del sexto y séptimo**, sin firmas ni otro paso RRHH completo.

### Resolución de ejercicio y versión documental

El detalle validado `v8` no sustituye la propuesta documental `v7`. El cliente
debe obtener de la consulta los antecedentes originales, mantener esa versión
para los seis PDF y mostrar el recibo histórico sin registrar otra resolución.
La prueba cerrada conserva el recibo tras reiniciar aplicación y PostgreSQL;
no se repite por esta actualización documental.

Antes de extender incorporación, inspeccionar las piezas de la misma canónica:

| Responsabilidad | Código existente que se reutiliza |
| --- | --- |
| Preparar y cargar configuración de ejercicio | [Configuración](../../internal/app/bootstrap/contratacion_temporal_incorporacion_configuracion.go). |
| Conectar dependencias reales | [Composición](../../internal/app/bootstrap/contratacion_temporal_incorporacion_v2.go). |
| Registrar la ruta | [Ruta interna](../../internal/app/composicion/interna/contrataciontemporal/ruta_incorporacion_v2.go). |
| Coordinar Personal y Contratación | [Aplicación](../../internal/app/incorporacionejercicio/). |

Su existencia no acredita activación, recorrido en navegador ni incorporación
registrada. Comparar diferencias con `c6359f98` y el trabajo pendiente antes
de tocar esos archivos; no copiar una rama o worktree sobre el árbol compartido.

## Arquitectura real y propiedad

El ejecutable del recorrido es [cmd/vec-server](../../cmd/vec-server/main.go).
Carga `config`, llama a `bootstrap.NewHTTPServerWithConfig` y, con el perfil
de desarrollo explícito, entra en la composición de desarrollo.

```text
Navegador: shell + módulo + cliente HTTP
  → servidor de desarrollo con certificado de cliente
  → lista de rutas y autorización de servidor
  → adaptador HTTP → caso de uso → dominio y puertos
  → adaptadores PostgreSQL / Bolsa / salida local
  → estado, historia y salida pendiente durables → recibo minimizado
```

No existe una base de negocio común que cualquier módulo pueda modificar:

- `internal/modules/contrataciontemporal/` posee el expediente y su procedimiento.
- `internal/modules/bolsa/` posee sus convocatorias, propuestas, órdenes y
  llamamientos. Contratación consume sus puertos y recibos; no escribe sus tablas.
- `internal/vec/` aporta contratos y autoridades compartidas: identidad,
  autorización, contexto del actor, documentos y evidencias.
- `internal/app/` conecta dependencias concretas y superficies.
- `internal/candidate/` es código heredado; no es un camino alternativo para
  duplicar el flujo de Contratación.

Dentro de un módulo: `domain` contiene reglas sin infraestructura;
`ports` declara contratos; `application` coordina dominio y puertos;
`adapters` traduce HTTP, SQL y proveedores. Un caso de uso no debe importar
PostgreSQL ni leer el DOM. El contrato de otro módulo se consume por su frontera,
no se reproduce con otra implementación equivalente.

La identidad sintética de desarrollo procede del canal validado y de sus
autoridades explícitas. No acredita Kerberos corporativo ni habilita producción.
No admitir cuenta, perfil o permisos declarados por el navegador.

## Dónde modificar y dónde conectar

| Necesidad | Fuente o directorio que debe inspeccionarse |
| --- | --- |
| Configuración y separación de conexiones | [config/postgresql_contratacion_temporal.go](../../config/postgresql_contratacion_temporal.go) y [config/config.go](../../config/config.go). |
| Arranque y selección del perfil | [bootstrap.go](../../internal/app/bootstrap/bootstrap.go) y [composicion_desarrollo.go](../../internal/app/bootstrap/composicion_desarrollo.go). |
| Conectar el procedimiento real | [contratacion_temporal_desarrollo.go](../../internal/app/bootstrap/contratacion_temporal_desarrollo.go); los archivos `contratacion_temporal_*_desarrollo.go` aportan cada dependencia. |
| Registrar rutas internas del módulo | [composicion/interna/contrataciontemporal/rutas.go](../../internal/app/composicion/interna/contrataciontemporal/rutas.go), función `NuevasRutas`, y su llamada en bootstrap. |
| Contrato HTTP y respuesta pública | [adapters/httpinterno](../../internal/modules/contrataciontemporal/adapters/httpinterno/): manejador, contrato cerrado y proyección del recibo. |
| Regla o coordinación funcional | [domain](../../internal/modules/contrataciontemporal/domain/), [application](../../internal/modules/contrataciontemporal/application/) y [ports](../../internal/modules/contrataciontemporal/ports/). |
| Persistencia y transacción | [adapters/postgres](../../internal/modules/contrataciontemporal/adapters/postgres/) y [deploy/postgresql/contratacion_temporal](../../deploy/postgresql/contratacion_temporal/). |
| Navegación y montaje web | [portal.js](../../web/static/portal-empleado/portal.js) y [portal-modulos-coordinador.js](../../web/static/portal-empleado/portal-modulos-coordinador.js). |
| Formulario, vista, contrato y cliente HTTP | [modulos/contratacion-temporal](../../web/static/portal-empleado/modulos/contratacion-temporal/): `formulario-*.js`, `vista*.js`, `contrato-*.js` y `cliente-http-*.js`. |
| Textos visibles | [portal-i18n.js](../../web/static/portal-empleado/portal-i18n.js); en el módulo, `i18n.js`, `i18n-expedientes.js` e `i18n-llamamiento.js`; backend compartido en [internal/shared/i18n](../../internal/shared/i18n/). |
| Publicar un recurso estático del recorrido | [web/produccion.manifest](../../web/produccion.manifest), leído por [estaticos_produccion.go](../../internal/app/server/estaticos_produccion.go). Revisar los otros manifiestos solo si cambia su superficie. |

Para ampliar una acción existente, seguir su cadena completa: contrato y
manejador → caso de uso y puertos → adaptador real → dependencia de bootstrap
→ ruta → cliente y formulario. Añadir una función o una ruta a `NuevasRutas`
no basta si la raíz no le entrega una implementación autorizada.

Ejemplo ya existente: selección llama a
`POST /api/vec/contratacion-temporal/llamamientos/seleccion` con
`expediente_ref`, `version_esperada` y `clave_idempotencia`.
El formulario de comunicación toma organización, llamamiento, versión y recibo
antecedente de la respuesta autenticada, no de referencias inventadas.

Cuando un nuevo import JavaScript devuelve `404`, comprobar el archivo y
todos sus imports transitivos contra el manifiesto. El servidor carga la lista
al arrancar: modificarla exige reiniciar el proceso para validar el resultado.
No quitar guardas que separan recursos reales y de presentación.

Conservar el shell y el tema compartidos: contexto del expediente, etiquetas,
confirmación explícita, estados de espera/error y recibo. No duplicar helpers,
CSS estructural ni reglas funcionales en la vista.

## Preparar y arrancar desarrollo local — referencia histórica

Desde la raíz del worktree canónico asignado, no desde la raíz histórica ni
desde el producto publicado mientras otra persona lo está validando.

Requisitos verificables en las fuentes:

- Go: [go.mod](../../go.mod) declara mínimo `1.25.12` y toolchain `1.26.5`.
  Usar la toolchain de entrega fijada, sin degradarla para salvar una compilación.
- Bash, Python 3, `curl`, OpenSSL y utilidades de sistema para el lanzador y
  el generador de credenciales. Git para inventario y revisión.
- Node.js compatible con `node --test` para pruebas web focales.
- PostgreSQL **18.4** para este recorrido y sus migraciones exactas; Docker
  solo si se utiliza la instancia aislada existente. No levantar otra por inercia.
- Certificados y material persistente fuera de Git; conexiones con
  `sslmode=verify-full` y autoridades de certificación correctas.

El [lanzador](../../scripts/arrancar_vec_desarrollo.sh) no instala PostgreSQL,
roles ni migraciones. Genera o verifica material de desarrollo, compila un
binario temporal y escucha en loopback con TLS 1.3 y certificado de cliente.
No usar `arrancar_presentacion_rrhh.sh` para demostrar efectos persistentes.

Para el recorrido ya conservado, preparar las once conexiones DSN en las dos
instancias PostgreSQL locales según el
[bloque histórico de la guía](../../GUIA_RECORRIDO_ALBERTO.md).
Cada aplicación usa sus once logins contra una sola base; no se mezclan entre
instancias, ni son conexiones a un remoto de desarrollo. No copiar DSN,
credenciales ni rutas privadas en este manual:

- `VEC_CT_DATABASE_URL`: ejecución del módulo.
- `VEC_CT_GOBIERNO_DATABASE_URL`: gobierno.
- `VEC_CT_CONFIRMADOR_DATABASE_URL`: confirmación de cobertura.
- `VEC_CT_LECTOR_RESULTADO_DATABASE_URL`: lectura de resultado.
- `VEC_CT_REGISTRO_AUTORIZACION_DATABASE_URL`: registro de autorización.
- `VEC_BOLSA_LLAMAMIENTOS_DATABASE_URL`: ejecución de Bolsa.
- Las cinco DSN adicionales de consultas, motivos RRHH, registro y
  revalidación de identidad y contexto del actor, también locales, se preparan
  según la guía.

Las once conexiones de cada aplicación apuntan a su única base local. No se
usa un DSN de desarrollo remoto. No activarlas parcialmente ni atribuir éxito
por existir sus variables.

Con esas conexiones preparadas y la variable `material_vec` apuntando al
directorio persistente aprobado:

```bash
go version
scripts/arrancar_vec_desarrollo.sh --puerto 8443 \
  --directorio-material "$material_vec"
```

Abrir `https://localhost:8443/portal-empleado/` con el certificado de RRHH;
Intervención usa su certificado y perfil de navegador separados.
Esta receta corresponde al antiguo entorno local. Para la instancia remota
activa use el apartado vigente de Sistemas; no arranque una segunda copia.
La importación protegida de certificados se describe en la guía; no publicar
claves, contraseñas ni cadenas de conexión en Git, capturas o mensajes.

La guía distingue la base del navegador en `55433` y la base de pruebas
en `55432`. Dirección puede reservar esta última para validar una candidata:
no lanzar pruebas en paralelo contra ella ni cambiar el destino del producto.
Nunca reconstruir o restaurar una base para resolver un fallo de arranque sin
una orden explícita.

Para parar, `Ctrl-C` en la terminal del lanzador. Para reiniciar, repetir el
mismo comando conservando conexiones, base y directorio de material. No
regenerar claves ni crear UUID nuevos para recuperar una operación anterior.
Una interrupción de transporte puede dejar un efecto confirmado: recuperar
con la misma intención y comparar recibos antes de ordenar otra operación.

## Ciclo corto: un hito observable, pruebas al terminar

1. Inventariar ramas, worktrees y cambios. Buscar primero la capacidad y sus
   llamadas; comparar deltas y patch-id antes de editar o integrar.
2. Acordar una única línea canónica y archivos de edición disjuntos. Si dos
   personas necesitan el mismo archivo, coordinar la edición, no crear copias.
3. Implementar el mínimo que permita avanzar a un usuario: una acción,
   conexión o recuperación completa. No rehacer piezas por mejoras cosméticas.
4. Al completar el hito, ejecutar las pruebas focales y revisar el recorrido.
   No repetir suites completas por cada línea ni abrir un nuevo ciclo documental.
5. Revisar seguridad según el riesgo y entregar archivos, evidencia y límites.
   Actualizar el manual afectado cuando cambie lo que su lector puede hacer.
6. Dirección integra y publica el cambio autorizado; solo después se limpia
   lo propio, integrado y sin modificaciones pendientes.

Comprobaciones iniciales de solo lectura en la línea actual:

```bash
git status --short
git worktree list --porcelain
git branch -vv
git cherry integracion/ct-producto-ligero-20260821 HEAD
git diff integracion/ct-producto-ligero-20260821...HEAD | git patch-id --stable
```

La última orden identifica el delta conjunto; `git cherry` compara equivalencia
de parches por commit. Ninguna sustituye inspeccionar cambios sin commit.
No copiar código manualmente entre ramas ni abrir worktrees sustitutos.

Elegir las pruebas del paquete o flujo modificado. Ejemplos existentes para
un hito de llamamiento, no una lista que deba repetirse para cualquier cambio:

```bash
go test ./internal/modules/contrataciontemporal/application -run Seleccion -count=1
go test ./internal/modules/contrataciontemporal/adapters/httpinterno -run Seleccion -count=1
node --test web/static/portal-empleado/modulos/contratacion-temporal/formulario-llamamiento.test.mjs
git diff --check
```

Para documentación sola: revisar contenido, enlaces y espacios; no arrancar
bases ni ejecutar las pruebas funcionales. Para interfaz: prueba focal y
revisión de navegador a 1440, 1024 y 390 píxeles, sin declarar evidencia visual
si solo se ejecutó Node.

Identidad, autorización, criptografía, SQL y datos personales requieren dos
revisiones independientes y comprobaciones específicas del riesgo. Los efectos
durables se demuestran con PostgreSQL real y un reinicio en el hito pertinente.
Las pruebas globales `go test ./...`, `go vet ./...` y la puerta de calidad
se reservan al cierre de integración que dirección programe; no son un paso
por cada edición ni sustituyen el recorrido humano.

### Commit, publicación y limpieza

Cada commit debe ser coherente, compilable y mostrar un avance o una
documentación utilizable. Añadir al índice solo las rutas revisadas; nunca
`git add -A` sobre un worktree compartido. Si la asignación exige entrega sin
commit, entregar el delta y dejar la confirmación a dirección.

Antes de integrar: repetir inventario/equivalencias, revisar el hash exacto
cuando corresponda y preservar el trabajo ajeno. La autoría configurada es
`aavidad`, sin atribuciones de IA. No cambiar la configuración global de Git.

Después del commit e integración autorizados, dirección hace push de la rama
canónica y verifica que el hash local, el de seguimiento y el remoto coinciden.
Sin push comprobado no decir «publicado». No forzar push ni borrar referencias
para ocultar divergencias. Producto limpio significa ausencia de cambios
pendientes allí; no exige borrar el trabajo de las candidatas.

Retirar exclusivamente ramas/worktrees propios, integrados y limpios, tras
confirmar que ninguna persona los usa. Preservar trabajo sin commit, ramas con
revisión pendiente y evidencias. En particular, los tres archivos históricos
sin seguimiento de `deploy/postgresql/autorizacion/` no forman parte de este
hito y no se añaden, borran ni reconstruyen.

## Cuándo actualizar este manual

Modificarlo en el mismo hito cuando cambien el arranque, la conexión de una
capacidad, sus límites o las rutas de trabajo del programador. Mantener las
firmas detalladas en Go y su catálogo generado; los pasos humanos en los
manuales de cada perfil; los recibos y comandos del recorrido en la guía.

Una entrega debe decir qué puede hacer ahora una persona, en qué entorno se
comprobó, qué prueba se ejecutó y qué falta para la acción siguiente.
Ni el número de archivos ni el volumen de pruebas son una medida de avance.
