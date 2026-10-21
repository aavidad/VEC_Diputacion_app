# F1-141: recuperar un llamamiento pendiente

Comprobación del 21 de octubre de 2026 sobre la candidata `2c1a585dd`.
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
real rechaza el cuerpo antiguo y el 404 de expediente.

La copia quedó parada y Chrome cerrado. Falta comprobar en Chrome la selección
singular nueva desde la candidata corregida y la CI final. No se afirma una
recuperación singular con autorización real sin configuración de ofertas.
La candidata todavía no está lista para fusionar.
