# F1-141: recuperar un llamamiento pendiente

Comprobación inicial del 21 de octubre de 2026 sobre la candidata `2c1a585dd`.
El último ensayo nativo, con el diagnóstico ya integrado, sigue bloqueado por
un 503; se detalla al final.
La integración y el cierre corresponden a dirección.

## Cambio preparado

Al abrir la ficha, la consulta de estado existente recupera la intención singular
pendiente con su clave y versión originales. «Reanudar llamamiento» requiere
revisión y confirmación antes de enviar la misma solicitud. La clave no se
muestra ni se guarda en el navegador. Carga, error y denegación impiden el envío.

CT257 reconstruye la lectura CT225 desde el cuerpo instalado en una copia de
main. Conserva propietario, permisos, autorización y auditoría; añade la rama
singular después de consumir la decisión. El contrato plural se conserva.
No hay configuración nueva. Se necesita el montaje de consulta de ofertas
existente, también para la lectura singular. Sin él, el portal arranca y la
selección singular nueva conserva el recorrido anterior cuando el enrutador
confirma que esa ruta no está instalada. Un error de una ruta instalada sigue
bloqueando el envío. La recuperación sin ofertas se separó en F1-142 por
decisión de dirección del 21 de octubre a las 02:46.

## Comprobaciones realizadas

- Pruebas focales Go de aplicación, decodificación y representación HTTP.
- 211 pruebas web y comprobaciones posteriores de reserva sin efecto y de
  consultas intercaladas. Se corrigió el botón activo cuando otra consulta
  impedía actuar.
- La CI encontró una regresión en la recuperación plural con comunicaciones
  caídas y dos claves nuevas sin registrar en el control de textos. Corregidas
  ambas: 39 pruebas web focales y 15 del registro de ensayos pasan después de
  incorporar main `0596d3825`. Las huellas anteriores de textos se conservan.
- La confirmación de reanudación mostraba la referencia técnica heredada del
  diálogo común. Se corrigió solo para este flujo: muestra el número del
  expediente o «Sin número asignado», manteniendo la referencia, versión y
  clave del POST. Dos pruebas focales y revisión de usabilidad favorables.
- Instalación de CT257 y rechazo SQL sin decisión en PostgreSQL 18.
- Dos revisiones independientes de código y SQL sobre `2c1a585dd`. Cada una
  ensayó 140 migraciones y 24 pruebas antes y después del reinicio, con las
  dos comprobaciones de concurrencia.
- Semgrep local sin red ni métricas: cero hallazgos. GoSec analizó los cuatro
  paquetes afectados: 107 avisos en otros ficheros y ninguno en los cambiados.
- Revisión independiente de usabilidad por inspección del código, con el
  hallazgo de disponibilidad del botón corregido.
- Chrome del sistema a 1440 y 390 píxeles, con certificado de la copia propia:
  dos recargas sin POST; tras confirmar, envío de la clave y versión del GET,
  recibo visible, clave oculta y sin desbordamiento horizontal.

La comprobación de Chrome usó respuestas controladas de Playwright **solo para
el GET de estado y el POST de selección**. Portal, ficha y comunicaciones eran
de la aplicación local. Las capturas `recuperacion-controlada-1440.png` y
`recuperacion-controlada-390.png` y su guion se conservan fuera de Git. Esto
acredita la interfaz; no acredita una recuperación con autorización real.

## Comprobación integrada pendiente

La copia inicial con el kit antiguo arrancó, pero no montó la consulta nominal
de Personal. Se retiró y se usó la orden pública `--recorrido-completo` de
`deploy/principal/recorrido_copia/LEEME.md`, sin kit privado.

La orden completa se detuvo en `copia_local_personal.py:130`: la consulta de
categorías devolvió 503. El depurador sobre el binario sin modificar localizó
la causa: el lector de catálogos rechazó un posible dato personal o secreto.
El catálogo generado del plan de firma contiene referencias de fuente y
aprobación con un token de 24 caracteres hexadecimales y más prefijos de los
admitidos como referencia opaca. La coincidencia se comunicó al responsable de
la receta para corregir el productor; no se alteraron el lector, el catálogo
publicado, las firmas ni la autoridad.

La corrección del productor se integró en main mediante #1407. La candidata
`3c6d79f57` creó la copia pública completa en 248,05 segundos y montó CT y
Personal. La consulta de estado devolvió 503 sin consultar PostgreSQL porque
la receta no activa `ofertas-ct.json`. Es una dependencia de composición:
CT225 consume la autoridad de vinculación de ofertas, que la guarda de lectura
singular no sustituye. No se improvisaron permisos ni configuración.

Dirección limitó este corte a conservar el recorrido singular anterior cuando
la ruta no está instalada. La recuperación sin esa configuración pasa a F1-142.
La corrección distingue el 404 catalogado del enrutador de un 404 de expediente,
un 503 y una respuesta malformada; conserva bloqueadas las selecciones plurales
y las intenciones que la pantalla ya había recuperado. Las 28 pruebas focales
del formulario de ofertas pasan después de este cambio.

La siguiente copia, desde `f63f91ba4`, arrancó en 231,99 segundos. Chrome
comprobó que la ruta ausente devolvía todavía el cuerpo antiguo del router
exterior, sin catálogo. El cliente lo rechazó y mantuvo el bloqueo. Se corrige
esa salida reutilizando el error catalogado existente; una prueba Go distingue
ruta ausente y 404 propio de una ruta montada, y otra con el cliente JavaScript
real rechaza el cuerpo antiguo y el 404 de expediente. La respuesta por defecto
reutiliza también las protecciones del helper: no se almacena en caché, impide
interpretar otro tipo de contenido y usa una política de contenido cerrada;
retira cabeceras de cookies, acceso desde otros orígenes y redirección. El 404
anterior solo fijaba el tipo JSON. Dirección aprobó esta diferencia a las
03:06; la prueba incluye ruta arbitraria, POST ausente y 404/405 propios de una
ruta montada, que se conservan.

La copia quedó parada y Chrome cerrado. Falta comprobar en Chrome la selección
singular nueva desde la candidata corregida y la CI final. No se afirma una
recuperación singular con autorización real sin configuración de ofertas.
La candidata todavía no está lista para fusionar.

## Ensayo final con la receta nativa y Go corregido

La candidata `a5d78c211` incorpora main con Go 1.26.10 y la receta de carga
nativa. La copia completa arrancó en 292,20 segundos. Chrome del sistema,
a 1440 y 390 píxeles, comprobó el 404 catalogado real de la ruta ausente,
la selección singular habilitada y ausencia de desbordamiento horizontal.

Los expedientes fiscalizados del volcado son del lote antiguo; el elegido
rechazó el envío con 409 antes de crear una intención. Se preparó un expediente
nativo por las pantallas con los perfiles propios de la copia, reutilizando
solo el código del guion cedido por el equipo que recorre el caso completo.

El primer informe se firmó con el manejador instalado 0.0.105. Su segunda firma
fue rechazada: el apartado 11 de la receta documenta que esa versión impide
cofirmar. Se conservó sin alteraciones y se creó otro expediente, `2026/95042`,
con el manejador cedido y declarado por su huella
`89a7ecb8bcddd1501873a5debb4fb777f535ca7a62f0cc73be62c927a0403b36`.
No se acredita una distribución pública ni un compilador actualizado de ese
manejador; se usó únicamente para las identidades ficticias del ensayo.

El segundo expediente completó petición, ratificación, alta, análisis, vía,
asignación e informe. Las dos firmas nativas devolvieron 201 y verificación
técnica válida; ambas conservan `firma_eficaz: false`. La fiscalización
favorable devolvió 201 y versión 6.

La selección singular real devolvió 503 y dejó la intención en estado
`indeterminada`, con efecto `solicitar_llamamiento` y ambas ventanas abiertas.
Un único reintento por la pantalla devolvió también 503. La solicitud original
se conserva en la pantalla y en el registro privado previo al envío; no se ha
recargado ni creado otra clave. Se detienen los intentos y se comunica el
bloqueo a dirección. No se modificaron permisos, firmas, recibos ni datos para
forzar la selección. No hay comprobación positiva de recibo o reinicio de esta
operación; la candidata sigue sin estar lista para fusionar.


## Ensayo con F1-141 y el diagnóstico integrado

Dirección integró F1-144 y autorizó una copia nueva con F1-141 más ese
diagnóstico, una sola selección y parada si devolvía 503. La candidata
`c8ef0d2c2` incorpora main `b0248f31d`. CT257 conserva exactamente la huella
revisada; las listas mantienen CT253 de main antes de CT257.

La copia completa arrancó en 256,78 segundos. El expediente nuevo `2026/95044`
se preparó por las pantallas desde la petición del centro: ratificación, alta,
análisis, vía, asignación e informe. Las dos firmas nativas devolvieron 201 con
verificación válida y `firma_eficaz: false`. La fiscalización favorable devolvió
201 y versión 6. No se añadió configuración de ofertas; el GET de estado
respondió con el 404 catalogado de ruta no montada y el formulario permitió
la solicitud singular nueva.

El único POST de selección, a las 05:47:43 de Madrid, devolvió 503. El registro
indicó `causa_interna=*errors.errorString`, etapa desconocida, centinela no
clasificado y SQLSTATE vacío. El tipo Go no identifica la causa concreta.
Se comunicó a dirección y se detuvo el recorrido: sin reintento, recarga,
reinicio, otra clave ni otra selección. La copia y la intención se conservan;
los registros y la captura a 1440 píxeles permanecen fuera de Git. No hay
comprobación positiva de recuperación real ni candidata lista para fusionar.

La CI posterior a unir main se detuvo antes del ensayo SQL: el control de la
lista esperaba 25 casos, pero CT253 y CT257 juntos suman 26. Se actualizaron
solo las dos expectativas numéricas a 26, manteniendo la comprobación de
bytes versionados y la lista cerrada. La prueba focal de ese método pasa.
No cambió el SQL ni el código de producto ensayado.


Después del ensayo, main incorporó CT256 y su pantalla de aceptación. Para
resolver los conflictos de integración se conservaron la preparación de esa
aceptación y la consulta singular de F1-141. Las listas mantienen CT256 antes
de CT257; ahora contienen 27 pruebas. La línea base usa las 1388 líneas reales
del formulario. Pasan cinco casos web focales de ambos recorridos y la prueba
del listado. No se ha vuelto a ejecutar la operación nativa: la copia conservada
sigue en `c8ef0d2c2` y el bloqueo 503 permanece abierto.


Dirección pidió cotejar la hipótesis de una fila inexistente. El delta Go no
añade `QueryRow`, `Scan`, `errors.New` ni `fmt.Errorf` en el POST de selección;
la decodificación modificada pertenece al GET que respondió 404 de ruta ausente.
No se ha demostrado la causa con una prueba roja y no se ha cambiado el
tratamiento de ausencias ni la lógica de negocio. Las solicitudes guardadas
del control 200 y del intento 503 tienen los mismos nombres de campos y versión
esperada 6; no incluyen una acción de recuperación adicional.

El diagnóstico reconoce ahora `pgx.ErrNoRows`, cancelación y plazo agotado con
códigos cerrados. La prueba focal verifica errores envueltos y que sus mensajes
no aparezcan en el registro; pasa con un límite de memoria de 2 GiB. Esta mejora
no identifica por sí sola la causa del 503 ya registrado. La copia conserva la
intención original y no se ha ejecutado otra selección.
