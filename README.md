# VEC Diputación de Granada

El detalle muestra ocho fases del procedimiento y el historial de actuaciones
ya registrado. Solo destaca la fase actual; las demás permanecen **Sin confirmar**.
El rail enlaza la fuente RRHH y el flujo declarado mediante sus huellas, sin
crear estados ni dar por completadas fases anteriores.

La comprobación real de Unidad y del expediente `fe493…v9` pasó a 1440 y
390 px, sin desbordamiento ni errores JavaScript. Las consultas respondieron
`200`: Gestión de bolsa y Nombramiento fueron sus respectivas fases activas;
el historial abierto mostró 3 y 9 actuaciones. El CSS móvil corrige los mínimos
de las cuadrículas y del formulario, conservando el desplazamiento interno del
rail. No hubo nuevas escrituras de negocio ni descargas documentales en esta
comprobación; tampoco se mostró un recibo que permitiera volver a cotejarlo.

El resumen de la candidatura aceptada está conectado al formulario de
propuesta CT65 existente. Su asset respondió `GET 200`; falta recorrer ese
resumen con una aceptación recuperada. El corte no acredita envío ni firma.

La apertura de formularios exige ahora la asignación que proyecta el detalle:
el caso Unidad v3 presenta Asignación y no ofrece todavía Informe jurídico.
La prueba real confirmó esa secuencia y la legibilidad del aviso de rectificación
a 390 px, sin escrituras de negocio. Con el catálogo actual la interfaz indica
«Rectificación no disponible» y no monta controles de envío.

El detalle reúne ahora el resumen final para GINPIX: centro y categoría del expediente consultado, periodo, fecha y recibo original de incorporación. La ficha de carga manual se descarga por GET 200 con la misma huella `4f56c3dd607a1495852ac5e88a3b15340a50481c87a46a6fe078e3350afd4057`; el seguimiento responde 200. El envío externo figura como no conectado. Comprobado a 1440/390 px sin errores JavaScript ni desbordamiento. Los indicadores del cuadro precisan que cuentan los expedientes de la página actual.

La búsqueda del cuadro indica su criterio real: inicio del número de expediente. El botón de cierre amplía su área a 44 × 44 px. La lectura nominal de la bandeja conserva las mismas 52 referencias y fases, sin errores JavaScript ni desbordamiento en 1440/390 px.

El corte D permite reutilizar los seis borradores de la propuesta original v7
desde el expediente ya avanzado a v9. CT91, SHA256
`8759adf01dfe2d14a583ee051527367423ff5fe6ad147953be2b0dc3a0181a34`, se
instaló una sola vez; recibió dos `GO`, y pasaron el ensayo PostgreSQL del
catálogo en `ROLLBACK`, la comprobación del catálogo instalado y las pruebas
focales Go de HTTP, adaptador PostgreSQL y composición. En runtime se descargaron
dos representaciones del informe definitivo: el PDF, 29280 bytes y SHA256
`f6bd9fd9f61dc266621e9a72ab4ec0f5e24f0c55073a1de403d59cd7d9531810`.
El DOCX, OOXML válido, tiene 3378 bytes y SHA256
`7141cbc60e586086494cb4bc609ef9748c6f09675cb1f172ae4649f3f1944577`.
Cuadro, detalle y PDF respondieron con cuatro `POST 200`; incorporación y
preparación de cierre dieron `GET 200`, y el DOCX respondió `200`. Son un PDF y
un DOCX representativos de los seis borradores, no doce descargas. La comparación
de solo lectura conservó sin cambios la historia CT en 12 tablas, su estado y
las cinco tablas de Personal. No hubo escrituras de negocio, errores JavaScript
ni desbordamiento a 1440/390 px; dos `404` heredados de Bolsa son ajenos al
corte. El parser admite `menu: null` sin habilitar acciones y están desplegadas
las siete claves locales canónicas. No reaplique CT91.

Corrección de recuperación de Cobertura: el análisis registrado conserva la fase Solicitud. La vista exige resultado RC, la misma referencia y versión en cuadro/detalle, y ausencia de cobertura o asignación. El expediente existente `b50fa719…`, v2 y RC validada, abre ahora el formulario sin repetir el análisis. Cuadro y detalle responden 200; su consulta automática de propuesta devuelve 503 y sigue en corrección del backend. Esta comprobación acredita el montaje recuperado, no una decisión de cobertura ni el cierre de la pantalla 4.

El montaje normal de RRHH recibe ahora el cliente existente de Fiscalización para continuar desde Informe jurídico. La bandeja real respondió 200 sin errores JavaScript; no contiene actualmente expedientes en esa fase, por lo que este corte acredita la conexión y sus comprobaciones focales, no un nuevo registro de fiscalización.

Bandeja y detalle muestran los nombres del catálogo de centros y categorías ya cargado; conservan las referencias originales si falta una etiqueta. Comprobación real: «Centro solicitante» y «Categoría C2», sin errores JavaScript ni desbordamiento a 1440/390 px. Como antecedente, el lector histórico respondió 404 y la descarga desde v9 quedó pendiente en aquel corte; el corte D posterior la cerró con un PDF y un DOCX representativos descargados desde la propuesta original v7.

El detalle muestra la jornada como porcentaje (10000 → 100 %). La comprobación anterior a 1440/390 px acreditó aquella pantalla sin desbordamiento ni errores JavaScript y, entonces, dos columnas en móvil. El CSS vigente cambia a una columna por debajo de 620 px y mantiene botones de navegación de al menos 44 px; la comprobación final del runtime confirmó ambos casos a 390 px sin desbordamiento horizontal.

Si falla la apertura de Análisis o del siguiente formulario de Cobertura, Asignación o Informe jurídico, aparece un aviso con reintento de apertura. El reintento conserva el recibo y no repite el POST confirmado; la prueba focal cubre dos fallos consecutivos sin duplicar avisos. Bandeja y detalle reales comprobados a 1440/390 px, sin errores JavaScript ni desbordamiento.

Copyright (c) 2026 Alberto Avidad (avidad@dipgra.es), para la Diputacion
Provincial de Granada. Publicado bajo la
[Licencia Publica de la Union Europea v1.2 (EUPL-1.2)](LICENSE).

## Incorporación, GINPIX y Word comprobados en navegador — 12 de septiembre de 2026

La base de este corte es `2eb94c1c8277a7bb93393d7ed41e66580590182f` y sus
cambios son `f1512fdbe85a891d425a1d453993c512a5f420b4` y
`cfa4d70fa14a23827f8cd15420975fe5d5d82fd3`. El runtime principal comprobado
usa el binario SHA256
`bc65b1d6213cf1da31d58fd23c96acb27b4b54c8f99ef9cbc7948d30c21b4122`, un
asset SHA256 `f0fa62246a8dbeb5b3dcc8e56e68e7e3d9ace4511f1ba813619764130c63ceb4`
y el árbol web SHA256
`b3a18ba8556946dd7b33c1e9cccbfe95c4a37f6a2c219db815a5e8ab43dd1215`.
Chrome obtuvo `200` al recuperar
la incorporación y la ficha GINPIX, sin repetir el `POST`: se conservaron el
recibo `ref:2bc3d281…`, la fecha
`2026-09-10T13:07:06.614186Z` y los seis campos cotejados. La ficha GINPIX
descargada tiene SHA256
`4f56c3dd607a1495852ac5e88a3b15340a50481c87a46a6fe078e3350afd4057`.

El corte B muestra en la ficha GINPIX el recibo de incorporación y el nombre
del archivo. Las fechas del seguimiento son legibles y conservan el valor ISO;
la descarga real respondió `GET 200` con ese mismo SHA256, el seguimiento
respondió `GET 200` y no hubo errores JavaScript. La revisión móvil de este
corte terminó sin desbordamiento a 1440 ni 390 px.

En el runtime anterior `75157434…`, los seis borradores Word se descargaron con
HTTP `200` y son ZIP válidos. Su
informe registró cero errores JavaScript. El recorrido de incorporación registró
además cero cookies y almacenamiento, y no mostró desbordamiento a 1440, 1024
o 390 px. Son borradores de desarrollo: no acreditan firma, eficacia, envío ni
transmisión.

La trazabilidad conserva recibo y fecha visibles; su detalle se abrió y cerró.
No hubo errores JavaScript, cookies ni almacenamiento, ni desbordamiento a
1440, 1024 o 390 px. No se repitieron los seis Word. En aquel recorrido, el
`GET` de preparación devolvió `404`; la comprobación vigente aparece debajo y
tampoco acredita todavía un cierre.

La agrupación a ancho completo de los seis pares PDF/Word está desplegada y
visible. La recuperación tras el reinicio conservó sus nombres, bytes y SHA256;
también mantuvo idénticas cinco huellas de Personal y seis contadores más la
fila de incorporación de CT. Los objetivos 11, recuperación, y 12, ficha manual
GINPIX, quedan acreditados funcionalmente. Esto no completa Contratación ni
ocho hitos. Auth13 está instalada una sola vez. AD3-30/CT86, AD3-31/CT87 y
CT90 se instalaron después una sola vez en principal; el recorrido se detalla
debajo y su recuperación tras reinicio ya está acreditada.

El frontend publicado integra fechas civiles UTC legibles sin desplazar
el día, paginación mediante cursor opaco de un solo uso y validación accesible
del filtro de hasta 80 caracteres. Los errores recuperables conservan el
cuadro y no lanzan otra consulta por una entrada inválida. Sus dos revisiones
estáticas dieron `GO` y la suite web terminó 351/351. En navegador, 52 filas
con límite 100 dejaron **Siguiente** correctamente inactivo; un filtro válido
redujo el cuadro a una fila y 81 caracteres mostraron aviso accesible sin otra
consulta ni perderla. No queda acreditado avanzar con cursor porque el conjunto
no supera una página.

Durante esta reactivación se incorporaron 60 fuentes SQL históricas exactas, con 60/60
entradas de manifiesto y 30/30 scripts `UP` cotejados con el registro privado.
No se ejecutaron SQL ni `DOWN`. CT70–85 y la instalación de Auth13 conservan
su historia. Los avisos de fin de fichero de cuatro originales CT70/80 se
mantienen para preservar su igualdad byte a byte. Dos clústeres aislados se
restauraron. Los ensayos posteriores de CT86/87 y la instalación principal de
CT90 se detallan debajo.

La UI de recuperación del cierre ya está visible. En el runtime `037b…`, el
`GET` de preparación estaba bloqueado; el recorrido posterior descrito debajo
obtuvo `200`, aunque no llegó al `POST`. **Preparar cierre** crea y muestra una solicitud inmutable y ofrece
guardar su JSON sin enviar `POST`; **Guardar datos de recuperación** conserva el
archivo antes de confirmar. Una segunda acción confirmada ejecuta el cierre.
Importar coteja expediente y seguimiento contra el recibo del `GET` original;
recuperar o completar reutiliza exactamente esa solicitud, incluso si quedó
pendiente. Los cinco archivos de recuperación recibieron dos `GO`; los otros
cuatro, de trazabilidad, tuvieron revisión proporcional de dirección. La web
terminó 378/378 `PASS` y los manifiestos 111/11/3/1 pasaron. No se envía ningún
`POST` de incorporación.

Los cuatro `UP` de CT86/87 pasaron primero en dos clones aislados. Después se
instalaron una sola vez en principal junto con CT90. La instrumentación separada
de CT87 quedó bloqueada y congelada; no acredita un ensayo funcional.

La candidata de bandeja nominal añade 16 archivos Go sin SQL: técnico explícito
y lectores nominales para dos consultas, con ámbito conjunto de organización y
unidad y sin fallback. Recibió dos `GO` estáticos sobre el manifiesto `9e65…` y
Las focales de puertos/bootstrap y la campaña conjunta `go test ./...`,
`go vet ./...` y compilación pasaron. El E2E con dos lectores sigue pendiente. La fuente SMTP será una
cuenta remitente todavía por crear sobre una IP interna de la Diputación; el
destino es el correo obligatorio de un alta VEC existente. Faltan concretar
servidor, puerto, TLS y credencial reales.

El corte de Cobertura incorpora 20 fuentes revisadas con dos `GO`. RRHH debe
elegir expresamente entre **Bolsa**, **SAE** y **Nueva convocatoria**; la
recomendación visible incluye su motivo ligado y no sustituye esa decisión. La
v1 histórica se preserva y la v2 corrige las secuencias 3/4 con el DTO y
traductor reales. Tras recargar, se recupera el formulario de asignación ya
existente, sin crear otra reasignación. Todavía no está en runtime ni tiene E2E
de las tres vías. Web 385/385, manifiestos 111/11/3/1, Go global, vet y build
pasaron. El binario candidato
`a86116def448a1bae7193bcc86cc7bf3a482c78ec98d3b8f4ddae9450fc29e06`
está preparado, sin desplegar.

El lector legítimo actual está limitado a la unidad RRHH y al técnico de su
organización; no se han probado dos unidades ni cambiar el centro. En un clon,
CT87 arrancó con HTTP `200`, pero la publicación técnica falló por una
precedencia JSON incorrecta. El preflight posterior y cinco snapshots quedaron
idénticos al estado inmediatamente anterior al intento, sin tres `INSERT`
persistidos. CT90 se instaló primero únicamente en los dos clones y después en
principal, como se detalla debajo. CT87 ya está instalada: no
reaplique sus cuatro `UP`; la instrumentación bloqueada quedó congelada, no
acredita un ensayo funcional y no hay
rollback acreditado todavía.

CT90, SHA `581c69…`, y su focal final `c731…` recibieron dos `GO` estáticos y
operativos. Se instaló una sola vez en cada uno de los dos clones, nunca en la
principal. La prueba real —un positivo, nueve negativos y la sucesora
operativa— pasó; el cambio se limita a tres paréntesis de
`cierre87_validar_sucesora`: modifica su cuerpo solo en esos tres puntos y
conserva OID, firma, roles, ACL, las otras funciones y el resto de la historia.
Los recibos de CT86 y CT87 conservan SHA256
`54a437cf11aefa525b135896e7daf837962819ae0e49ea3486fa676f829f00b7` y
`27379a8fd633455ccf7878935bed791a50c04e6ea02a5e490ea1e82793663675`.
No reaplique CT86, CT87 o CT90. La recuperación `e359…` dejó sus tres filas
correctas. La campaña global Go y vet de continuidad terminó en `PASS`; esta
evidencia privada queda custodiada sin publicar sus rutas.

Un recorrido posterior de Chrome sí acreditó la anotación administrativa:
`201` a las `16:33:10.102225Z`, recibo
`recibo:56756272-1842-4778-b857-e0ae59b322db`, expediente v8→v9 y seguimiento
original v1. El `GET` de recuperación devolvió `200` con el mismo recibo. La
comparación SQL de solo lectura conservó toda la historia
anterior y encontró una anotación, una versión, una actuación y un outbox; las
cinco tablas de Personal y la incorporación permanecieron iguales. Preparar
cierre respondió `GET 200`; se guardó el JSON y un único `POST` de cierre
devolvió `400`. Tras corregir la allowlist HTTP, se recuperó exactamente el JSON
y la clave original `aed453da…`: un único replay respondió `201` a las
`17:35:57.825562Z`, recibo
`ref:2db8cfe02f7f97bc99b183ac66d579b83698981e78220f301e39f873f8b36f7c`,
seguimiento resultante 2 y expediente v9 intacto. El recibo procede de
`respuesta_json`. La lectura SQL conservó las filas previas de las 12 tablas históricas y añadió solo
preparación, registro, auditoría y outbox de CT87; las cinco tablas de Personal
permanecieron exactas. No hubo errores JS, cookies, almacenamiento ni overflow
a 1440/1024/390. Tras reiniciar la misma app y PostgreSQL del clon87, Chrome
recuperó la anotación con `GET 200` y el mismo recibo/v9, y el cierre con replay
`POST 200`, el mismo recibo y seguimiento 2. No duplicó negocio: se conservaron
registro, preparación y outbox; solo añadió la auditoría prevista de recuperación
`recuperado=true`, manteniendo la auditoría original. Personal siguió exacto.

Los dos archivos Go HTTP `7231…` y `d2bfc1d8…` tienen `GO`; Go global y vet
pasaron. El binario mínimo `bc65…` reinició el mismo clon87 con salud `200`, sin
tocar principal ni material web. La anotación se recuperó tras reiniciar la app
con el mismo recibo, fecha y v9. Este cierre no acredita firma, cese, eficacia,
envío ni GINPIX externo.

El corte E08 incorpora traducciones comunes de anotación y cierre en siete
archivos JS y pruebas revisados; 21 focales pasaron. Muestra un motivo humano
sin alterar el payload y propaga los mensajes configurados escapándolos. Las
dos fases y el replay permanecen intactos. E08 aún no está desplegado en runtime;
la campaña global terminó 365/365 pruebas CT y 23/23 del coordinador, 388 en
total, con manifiestos 11/111/3/1 en `PASS`. Este corte no cambió Go ni repitió
su campaña ya cerrada.

La corrección UI5 `f39…`/`37da…` recibió `GO` y quedó aplicada en integración:
invalida el estado anterior al confirmar el cierre, conserva el recibo y refresca
la lectura mediante GET. La raíz pasó 395 pruebas web y los
manifiestos 11/111/3/1. Esta corrección de interfaz aún no está en runtime; la
evidencia de navegador corresponde a la web `037b…` con el binario `bc65…`.

La prueba de CT86 quedó en `NO-GO`: la guarda requiere 10 s y la capacidad real
es 5 s. No se ejecutó el arnés SQL y el negocio quedó intacto.

El lector DER tiene dos archivos Go revisados con dos `GO`; bootstrap completo
en 23,254 s y vet pasaron. Rechaza un perfil PEM calculado antes de bootstrap y
no crea otra autoridad. Su activación principal falló a las 16:00 por ese
perfil; se restauraron binario y material propios y la salud volvió a `200`,
sin restaurar la base. El material DER corregido pasó 11 pruebas y el cargador
de contexto real, pero el reintento está pausado y el lector no está activado.
El nuevo corte fuente de correo reúne 18 archivos Go y un catálogo de traducción, manifiesto `cd5642…`, y
ocho SQL, con dos `GO` por cada grupo. Implementa E03/E06/E08 con reserva antes
de SMTP, auditoría HMAC común, autorización V3 final fresca y outbox; el JSON
contiene diez campos canónicos y el estado admite cuatro literales SQL. El replay no devuelve el secreto de finalización ni dispara otro
SMTP, y los dos accesores del parser existente no añaden autoridad. CT88, AD3-32 y T13/5 no están
instaladas. Faltan el contacto VEC candidato y acreditado, la cuenta y
configuración SMTP en la IP interna, la composición real y el ensayo PostgreSQL.
No hay SQL instalado, runtime ni envío. Go global y vet terminaron en `PASS`;
no se repitió la suite web: no cambió JavaScript; el catálogo y los manifiestos
fueron verificados. El nuevo autorizador
de bootstrap todavía no está incluido.

Cursor9 está integrado en fuente con seis archivos Go (`3e189e10…`) y el
paquete SQL CT89 (`9b9b559…`; `UP 1f27d896…`, `DOWN 85853ed…`, focal
`89b2be…`). Los Go recibieron dos `GO` de apoyo y uno independiente; los tres
SQL, dos `GO`. Corrige la confusión entre SHA ASCII del localizador y SHA raw32
de la evidencia. La sesión solo continúa con el mismo TLS, certificado y lector,
autoridad fresca revalidada y consumo único antes de delegar, dentro del mapa de hasta 64 cursores
y TTL límite. No usa cookies ni relaja el TTL SQL. Cambio TLS, expulsión,
reinicio o fallo tras reservar exigen empezar una primera página nueva. CT89 no
está instalada; faltan el ensayo PostgreSQL 50→2 y la recuperación del cursor,
por lo que no hay E2E. Go global y vet terminaron en `PASS`. Los roles
normales no cambian. Contacto13 se aplicó después, como se detalla debajo.

Contacto13 (`3191e8c3…`) está aplicado en fuente con dos `GO` nativos adicionales
y dos de apoyo. Su API interna en Go, aún no expuesta por HTTP, permite alta,
cambio y consulta del contacto VEC con emisor V3 real; cifra mediante subclave KMS y AES-GCM, AAD por
persona y versión, y buffer efímero. El nuevo módulo de usuarios declara
permisos, pero no los concede. Las pruebas de V3/PDP/COSE usan material real;
gobierno y persistencia usan dobles. El catálogo añade cinco claves y conserva
las dos de correo. No instala `roles_up.sql`, no infiere direcciones heredadas
ni envía SMTP. Siguen pendientes preparador HMAC central, almacén, AD3-35,
T13-6, gobierno, alta web, vínculo Bolsa y replay durable. Go global y vet
terminaron en `PASS`; JavaScript quedó intacto y los manifiestos 11/111/3/1
pasaron. El catálogo 2+5 conserva SHA256 `d22f70a…`. No hay runtime.

Correo14 incorpora cambios en 14 archivos Go en fuente, con dos `GO` sobre el manifiesto
`d23a5d307014b791bdca2dfdbf109ec955f64bc663f4ca749b9d5a8db71d3cc8`.
Declara un rol de correo con exactamente dos concesiones. La autoridad real
PDP/COSE/HMAC está probada, conserva la correlación nominal de audit17 y no
intercepta CT54. Go global y vet terminaron con código 0. Todavía no hay HTTP
compuesto, PostgreSQL, SMTP, runtime ni E2E nuevo.

En principal, Auth13 quedó intacta y AD3-30/CT86, AD3-31/CT87 y CT90 se
instalaron una vez (`a65ba4…`). Con el mismo contenedor, `GET` mTLS devolvió
`200`; incorporación y ficha GINPIX conservaron recibo, fecha y SHA256, sin
`POST`. Chrome registró la anotación `201`, recibo
`430b3ba1-da78-4743-951c-bf5a89737f10`, a las
`2026-09-12T19:22:55.043368Z`, y llevó el expediente v8→v9. El cierre respondió
`201`, recibo `cbbc4406…2a3f6`, clave `3a9924a3-1944-4fc8-b510-2566f6767504`
y seguimiento `cerrado_administrativamente/v2`; el expediente conserva
`nombramiento/en_curso/v9`. Las filas anteriores se conservaron en las 12 tablas
y las cinco de Personal quedaron intactas, sin duplicar incorporación
o raíz. No hubo JS, cookies, almacenamiento ni overflow a 1440/1024/390. Dos
`404` ajenos de Bolsa no forman parte del recorrido. Tras reiniciar la misma app
y PostgreSQL principal, Chrome recuperó la anotación con `GET 200`, mismo
recibo, fecha y v9; el replay exacto del cierre devolvió `POST 200`, mismo recibo
y seguimiento 2, seguido de `GET 200` y UI **Cerrado**. Incorporación y ficha
GINPIX volvieron a dar `200` con recibo, fecha y SHA intactos. La comparación
final conservó once tablas CT, estados y Personal; solo añadió una auditoría
`recuperado=true`, manteniendo preparación, registro y outbox únicos.

El cambio UI `cfa4d70fa14a23827f8cd15420975fe5d5d82fd3` tiene dos archivos,
focal 14 y `GO`, y fue revisado sin secretos.
La web terminó 396/396 y los manifiestos 11/111/3/1 pasaron. No se repitieron Go
global ni vet por esta etiqueta JS. El navegador usó el asset `f0fa…`. Este
recorrido no acredita cese ni cierre jurídico del expediente, firma, eficacia,
correo, efecto legal o confirmación externa de GINPIX.

## Historia del estado funcional anterior — 12 de septiembre de 2026

Este apartado y los bloques cronológicos inferiores se conservan como historia;
el estado vivo es el descrito al inicio de este documento.

Base del cierre funcional: `00558603dbd3040eacb03cb511f3b840be241b10`, rama
`integracion/ct-producto-ligero-20260821`. El desarrollo activo y la base
sintética están en el servidor; las instrucciones antiguas de arranque local
no describen la instancia actual.
Este entorno usa exclusivamente datos sintéticos y no está autorizado para
producción ni para tratar datos reales.

Se pueden enseñar **cinco pasos completos y partes del sexto, séptimo y octavo**.
El 10 de septiembre se registró de forma real en desarrollo una
incorporación sintética: formulario `GET 200`/`POST 200`, dos confirmaciones
expresas, recibo registrado y alta sintética en Personal. Esta capacidad pertenece al
octavo paso, pero todavía no puede enseñarse como ciclo recuperado completo.

La resolución manual de ejercicio se comprobó con Chromium, certificado de
pruebas, autorización del servidor y PostgreSQL reales: alta `201`,
recuperación y repetición con la misma clave `200`, mismo recibo y sin duplicado. El caso avanza de versión
`7` a `8`; eso **no significa ocho pasos terminados**. Se conservan 52
expedientes y una nueva resolución de ejercicio; el caso original en versión
`7` permanece disponible.

**No hay firma oficial, eficacia administrativa ni envío.** La incorporación
ya está integrada y tuvo escritura visible en navegador, pero su recuperación
después del reinicio aún no está acreditada: la lectura devolvió `GET 503` por
la restauración histórica de Auth12. Auth13 conserva dos dictámenes `GO` y
regresión verde en `ef6704a6`, pero sigue pendiente de instalación. Por eso no
se declara todavía recorrible de extremo a extremo ni se eleva la métrica.

CT70–85, trece pools y las tres capacidades con sus catálogos y definición ya
están provisionados; no son trabajo pendiente ni deben reaplicarse. No hubo
precargas de negocio. GINPIX ya tiene preparación, montaje y descarga V2
integrados en código; falta comprobarlos desde el recibo recuperado en el
servidor. La anotación y el cierre también tienen piezas de dominio y
persistencia integradas, pendientes de su recorrido conjunto.

El portal está servido de forma **privada**, no en una URL pública.
`https://localhost:8443/portal-empleado/` corresponde al servidor remoto;
no funciona directamente en el equipo del visitante sin acceso preparado.
Consulte [acceso y operación actuales](docs/manual_sistemas/README.md#entorno-privado-vigente).
El certificado identifica al usuario de pruebas: no firma los documentos.

El 12 de septiembre se inventarió y conservó el trabajo local y remoto. El
backend pendiente de anotación y cierre está revisado y se ha corregido el error
que presentaba observaciones inválidas como indisponibilidad. La base conjunta
supera las pruebas Go, `go vet`, 314 pruebas web y los manifiestos. El director
remoto y sus agentes continúan la corrección del frontend, la preparación
operativa y la documentación; todavía no se acredita un nuevo recorrido
en la aplicación servida. El
[estado y plan vigentes](ESTADO_PROYECTO.md) conservan la continuación exacta.

## Empiece por su perfil

| Perfil | Manual | Para qué sirve |
|---|---|---|
| Usuario del portal | [Manual de usuario](docs/manual_usuario/manual_portal_bolsas.md) | Acceso, navegación, opciones reales frente a DEMO, recibos, mensajes y ayuda. |
| Recursos Humanos | [Manual de RRHH](docs/manual_rrhh/README.md) | Tramitación, responsabilidades, resultados esperados y límites de cada paso. |
| Programación | [Manual del programador](docs/manual_programador/README.md) | Composición, código reutilizable, contratos, desarrollo y comprobaciones por hito. |
| Sistemas | [Manual de Sistemas](docs/manual_sistemas/README.md) | Preparación del entorno, configuración, certificados, persistencia y operación. |

La [Guía de recorrido de Alberto](GUIA_RECORRIDO_ALBERTO.md) es la referencia
de los datos sintéticos conservados y de los recorridos anteriores. Para
el arranque y acceso al servidor actual, use el manual de Sistemas; no ejecute
las recetas locales históricas como si describieran esta instalación.

## Qué funciona de extremo a extremo

El recorrido acreditado utiliza navegador con certificado de cliente,
servicios reales de aplicación, autorización de servidor y PostgreSQL.
No utiliza el adaptador DEMO para afirmar un guardado.

La [organización de referencia](GUIA_RECORRIDO_ALBERTO.md#centros-y-organización-de-referencia)
permite consultar centros y preparar altas o cambios de unidades con motivo,
revisión y recibo persistentes. Alta y edición sintéticas comprobadas tras
reinicio, sin duplicados. No asigna ocupantes, concede permisos ni habilita
todavía la ratificación multicientro.

El [circuito previo del centro](GUIA_RECORRIDO_ALBERTO.md#petición-del-centro-y-ratificación)
permite presentar y ratificar una petición con dos identidades sintéticas
configuradas, formularios reales y recibos persistentes tras reinicio.
La entrega posterior al alta de RRHH ya se comprobó en el entorno privado,
con recibo conservado tras reinicio. No es firma documental ni habilita
automáticamente a todos los centros del catálogo.

| Paso | Recorrido disponible | Límite |
|---|---|---|
| 1. Solicitud | Alta desde formulario y primer recibo del expediente. | Datos y catálogos de desarrollo. |
| 2. Análisis | Registro del análisis por RRHH y nueva versión del expediente. | El ejemplo recorrido utiliza Sustitución; el formulario ofrece cinco modalidades. |
| 3. Bolsa | Propuesta y decisión de cobertura por **Bolsa vigente**. | No equivale a gestionar de principio a fin una convocatoria de Bolsa. |
| 4. Asignación | Registro de unidad y persona responsable referenciada. | Destino sintético configurado. |
| 5. Informe jurídico y Fiscalización | Documento de desarrollo y resultado favorable, favorable con observaciones o desfavorable; este último registra devolución a la unidad. | El documento no tiene firma ni validez jurídica. Fiscalización corresponde al perfil de Intervención. |
| 6. Llamamiento, parcial | Recorridos sintéticos, aviso CT62, declaración CT63 y aceptación manual del sucesor CT64 recuperables tras reinicio principal, sin duplicados. | Faltan vencimiento, envío corporativo y plazo. No acredita entrega ni plazo legal aprobado. |
| 7. Nombramiento, parcial | Propuesta desde aceptación sintética, `201` y recuperación `200` tras reinicio, expediente `6→7`; seis borradores descargables y validación manual sintética con recibo `7→8`. | Sin nombramiento eficaz, posesión real, firma, envío ni entrega; validación manual sintética disponible, no firma oficial. |
| 8. Incorporación y seguimiento, parcial | La incorporación y su asiento en Personal se recuperan por `GET` tras reiniciar aplicación y PostgreSQL, con recibo y fecha originales; la ficha manual GINPIX devuelve `200` y conserva sus seis campos y SHA256. | La ficha admitida para carga manual no acredita transmisión ni confirmación de GINPIX. Anotación y cierre siguen pendientes de instalar y recorrer. |

El último recuento formal se conserva en **cinco pasos completos más partes del
sexto y séptimo**. Las partes del octavo acreditadas arriba no se usan aquí para
recalcularlo; tampoco es un porcentaje global ni un recuento de pantallas,
contratos o pruebas.

### Antecedentes de los recorridos, no inventario de la instancia actual

Los párrafos siguientes conservan cifras y pruebas de sus cortes originales.
El inventario actual es de 52 expedientes sintéticos en el servidor privado;
las referencias a dos bases y aplicaciones corresponden al entorno anterior.

La bandeja y el detalle ya están conectados en `b2effba`: se demostraron 50
solicitudes conservadas y un análisis desde una de sus filas, sin otra alta.
Esto mejora la continuidad del trabajo; no cierra por sí solo otro paso del flujo.
El registro de respuesta conserva una respuesta, un asiento y un evento;
no cambia Bolsa, no avanza el expediente y mantiene comunicación versión `2`.
El `.eml` sintético se lee y resume con SHA256 en el navegador: no se sube ni
se custodia. El aviso sigue siendo local, no correo corporativo entregado.
AD3 `000015` / Bolsa `000004` y AD3 `000017` / CT `000058` están instaladas en
ambas bases, con ambas apps en la compilación corregida; AD3-16/CT57 ya estaban.
No reaplicar. La prueba aislada anterior de Bolsa4 (`8197db3`) usó un doble
privado transaccional. El roundtrip de aceptación UP/DOWN de esas cuatro migraciones
verificó reversión exacta en ROLLBACK, sin modificar autorización ni usar dobles.
El navegador actual sí usó criptografía real; no confundir estas comprobaciones.
AD3 `000018` / Bolsa `000005` / CT `000059` también están instaladas en ambas
bases: dirección confirmó UP/DOWN con ACL, funciones y comprobaciones conservadas.
No reaplicar ni ejecutar DOWN sobre los registros guardados.
Las preguntas pendientes no detienen la programación independiente ni autorizan
a inventar plazo o autoridad; continúa **5/8 más partes del sexto y séptimo**.

## Probar el recorrido disponible

1. Lea el manual de su perfil y la [guía canónica](GUIA_RECORRIDO_ALBERTO.md).
   Sistemas prepara el servidor, la base y el acceso del navegador.
2. Abra `/portal-empleado/` en el entorno autorizado, con el certificado de
   desarrollo correspondiente. RRHH e Intervención usan perfiles separados.
3. Para enseñar el trabajo conservado, entre en **Contratación temporal**,
   localice el expediente indicado por el operador y abra su detalle.
   Use **Nueva petición** solo si se ha acordado crear otro caso sintético.
4. Confirme una sola vez cada actuación y conserve la referencia del
   expediente, su versión, la clave de operación cuando corresponda y el
   recibo. Un error de conexión no demuestra que no se haya guardado nada.
5. Para continuar el llamamiento existente, utilice sus datos y claves
   originales. Recuperar no significa crear otra solicitud o preparar una
   clave nueva. La guía conserva el ejemplo exacto de la respuesta y enlaza
   el `.eml` sintético de aceptación que debe cargarse sin cambios. El caso de
   renuncia usa su propio material y claves; su recuperación tras reinicio también está confirmada.
   No eluda un rechazo cambiando claves o repitiendo el registro.
6. La recuperación tras reinicio ya está acreditada. No la repita para leer
   esta documentación o presentar el caso. Si Sistemas acuerda una nueva
   comprobación, conservará base, material, clave, recibo y fecha originales.

**Registrada localmente · Sin entrega acreditada** significa que se ha
guardado un aviso en el servidor de desarrollo. No es un correo enviado,
una notificación recibida ni una aceptación de candidatura.

## Módulos y superficies: estado honesto

| Área | Qué existe | Qué no debe darse por terminado |
|---|---|---|
| Contratación temporal | Bandeja, detalle y recorrido real descrito arriba. | Resto del paso 6, formalización oficial e incorporación/seguimiento completos. |
| Bolsa interna | Dominio, servicios, persistencia y pantallas reutilizables; proveedor durable conectado al llamamiento de Contratación temporal. | Gestión completa de convocatorias, borradores, méritos, alegaciones, contratos, firma y notificaciones desde el portal. |
| Bolsa pública | Consulta de convocatorias, categorías, detalle y documentos; composición pública separada. | Inscripción personal, consulta privada de posición y tramitación administrativa completas. Su disponibilidad depende del entorno configurado. |
| Personal y Nóminas | Módulo, contratos y material funcional de desarrollo/presentación. | Maestro de personal, nómina y procedimientos corporativos completos. |
| Cronos | Módulo y pantallas/capacidades de desarrollo y presentación. | Gestión corporativa completa de jornada, fichajes y aprobaciones. |
| Dietas | Módulo, presentación y conector cartográfico interno para cálculo de rutas. | Liquidación oficial, aprobación y pago completos; una ruta calculada no acredita una dieta aprobada. |
| Capacidades comunes | Identidad, permisos, catálogos, documentos, auditoría y eventos con distintos grados de conexión. | Disponibilidad automática de todas las capacidades en todos los módulos. |

Las 17 vistas del menú de Bolsa **no son 17 funciones administrativas
terminadas**. El [manual de usuario](docs/manual_usuario/manual_portal_bolsas.md)
clasifica cada opción sin confundir componentes escritos con recorridos
habilitados.

La presentación se identifica como DEMO y permite explorar pantallas y
resultados sintéticos. Sus adaptadores volátiles no sustituyen un servicio
real no disponible. La ayuda, los audios y las capturas de presentación
tampoco acreditan un envío, una firma o una actuación administrativa.

## Arquitectura y orientación en el código

Arquitectura hexagonal: el dominio expresa las reglas, la aplicación coordina
los casos de uso, los puertos fijan los intercambios y los adaptadores
conectan HTTP, PostgreSQL y otros proveedores. La raíz conecta esas piezas;
la existencia de un adaptador no garantiza que esté expuesto.

| Directorio | Responsabilidad |
|---|---|
| [internal/modules/contrataciontemporal](internal/modules/contrataciontemporal) | Procedimiento de contratación temporal y coordinación con otros módulos. |
| [internal/modules/bolsa](internal/modules/bolsa) | Reglas y capacidades propietarias de Bolsa. |
| [internal/vec](internal/vec) | Capacidades comunes del portal. |
| [internal/app/bootstrap](internal/app/bootstrap) | Ensamblaje y dependencias del entorno de desarrollo. |
| [internal/app/composicion](internal/app/composicion) | Raíces y superficies separadas de la aplicación. |
| [web/static](web/static) | Portal, formularios, recursos públicos y presentación. |
| [deploy/postgresql](deploy/postgresql) | Persistencia, funciones y roles por capacidad. |
| [config](config) | Configuración de los procesos y conexiones. |

Cada módulo conserva su autoridad: otro módulo no copia sus reglas ni accede
directamente a sus tablas. Las integraciones reutilizan sus puertos.
Las superficies pública, interna y de presentación tienen alcances distintos;
no deben intercambiarse como atajo para habilitar operaciones.

## Requisitos y configuración

- [go.mod](go.mod) declara Go `1.25.12` como mínimo y `go1.26.5` como
  herramienta de referencia del proyecto.
- El recorrido real requiere PostgreSQL, conexiones nominales separadas y
  certificados de desarrollo; su preparación está en la guía y el manual
  de Sistemas. No basta con arrancar una pantalla estática.
- La instancia y el material conservados se operan según el
  [entorno privado vigente](docs/manual_sistemas/README.md#entorno-privado-vigente).
  Las recetas Docker locales de la guía son históricas; no crean otra instancia.
- La referencia de [procesos y configuración](docs/manual_programador/cmd_y_configuracion.md)
  complementa el [código de configuración](config). Use los valores del
  entorno autorizado; no copie secretos ni conexiones a Git.

No se incluyen aquí contraseñas, certificados, conexiones privadas ni órdenes
de despliegue. El arranque de desarrollo no constituye una puesta en
producción.

## Documentación de referencia

Para uso actual, empiece por los cuatro manuales anteriores. En GitHub,
la rama de producto es `integracion/ct-producto-ligero-20260821`; la rama
predeterminada `vec-orquesta-20260619` puede mostrar el corte del 31 de julio.
Esta actualización no cambia la rama predeterminada ni acredita su publicación.
La verificación y cualquier cambio de esa selección corresponden a dirección.

- [Especificación del expediente remitido por RRHH](docs/portal_vec/expediente_contratacion_temporal_rrhh.md).
- [Arquitectura técnica modular](docs/portal_vec/arquitectura_tecnica.md).
- [Catálogo de contratos de API por módulo](docs/portal_vec/contratos_api_modulos.md).
  Describe contratos; para saber qué está conectado consulte la guía y los
  manuales de esta edición.
- [Historial de decisiones](docs/portal_vec/registro_decisiones.md).
  Conserva antecedentes con su fecha y alcance, no una orden de trabajo
  vigente por el mero hecho de estar enlazado.
- [Índice de requisitos históricos](docs/estudio_requisitos/README.md):
  estudio de julio, no guía de arranque ni plan operativo actual.
- [Referencias externas archivadas](docs/referencias_portales_aapp/README.md).
- [Paquete de cumplimiento pendiente de validación](docs/cumplimiento/LEEME.md)
  e [informe histórico del Comité de Seguridad](docs/comite_seguridad/LEEME.md).
- [Documentación del proyecto](docs/): archivo técnico; sus actas, revisiones,
  decisiones y relevos fechados mantienen su alcance original, no son el
  punto de arranque actual. No se reescriben ni se renuevan sus fechas.

Los manuales explican el uso; la guía conserva los comandos y datos del
recorrido. Los documentos históricos y las exportaciones anteriores deben
leerse con su fecha, sin convertir resultados antiguos en estado actual.

## Desarrollo, seguridad y ayuda

Antes de trabajar, lea completas las [instrucciones del repositorio](AGENTS.md)
y las instrucciones de desatasco vigentes facilitadas por dirección. Reutilice
la línea canónica de cada capacidad: inventar una implementación equivalente
en otra rama no es avance.

Los cambios se agrupan en hitos observables y se comprueban según su riesgo;
la documentación no exige ejecutar suites de producto. Las zonas sensibles
conservan las revisiones requeridas. El manual del programador recoge el
procedimiento técnico, sin imponerlo a quien solo utiliza el portal.

Se mantienen denegación por defecto, autorización en servidor, datos
minimizados y ausencia de cookies y almacenamiento web. No se usan datos
reales sin autorización expresa ni se interpreta una indisponibilidad como
éxito. Un recibo confirma únicamente el efecto que describe.

Para ayuda de uso, consulte el manual de usuario y **Ayuda** en el portal.
Para incidencias, conserve el mensaje, fecha, perfil y referencia de operación
por el canal autorizado. No publique datos privados ni credenciales en GitHub.

## Licencia y autoria

Este software es obra de Alberto Avidad (avidad@dipgra.es), desarrollado para
la Diputacion Provincial de Granada, y se publica bajo la
[EUPL-1.2](LICENSE) para que cualquier administracion publica u organizacion
pueda reutilizarlo, adaptarlo y redistribuirlo.

Condiciones esenciales de la reutilizacion (articulo 5 de la EUPL-1.2):

- Mantener intactos los avisos de autoria y de licencia, incluido el nombre
  del autor original, en el codigo y en las obras derivadas.
- Distribuir las obras derivadas bajo la EUPL o una licencia compatible de
  las enumeradas en su apendice.
- Indicar los cambios realizados sobre la obra original.

Todas las versiones linguisticas oficiales de la EUPL publicadas por la
Comision Europea tienen identico valor juridico. El fichero `LICENSE` incluye
primero el texto oficial en español y, a continuacion, el texto oficial en
ingles; ninguno prevalece sobre el otro.
