# Modalidad y causa en las ofertas — F1-11

El cambio conserva las claves del análisis de Contratación temporal en los datos
de la oferta de Bolsa. Las vistas de RRHH y Mi Bolsa reciben esos datos por sus
lecturas actuales. No añade filtros ni cambia las respuestas del aspirante.

## Instalación

Usar PostgreSQL 18 y una copia propia del volcado de referencia HZ37. Instalar
primero la lista completa `lista_sql_main_tras_hz37.txt`, sin combinarla con
listas antiguas. Preparar antes los LOGIN nominales que exige AD278, siguiendo
`LEEME_copia_local.md`; ese ajuste es un dato de copia, no una concesión nueva.
Después aplicar una sola vez `lista_sql_codexo_modalidad_causa_oferta_f1_11_20261018.txt`:
CT235 y B136, en ese orden. Las guardas cotejan las definiciones, propietarios,
configuración y permisos instalados; cualquier diferencia revierte la migración.
No se modifica el núcleo de autorización ni se añade configuración al arranque.

Las nuevas reservas exigen un análisis con modalidad y causa válidas y la versión
vigente del expediente. Se obtiene mediante el análisis ordinario de RRHH. Una
oferta anterior se recupera con el comando guardado, sin completar sus campos
con el análisis actual. Las ofertas nativas sin origen de CT mantienen su contrato.

## Comprobación pendiente

Ejecutar las pruebas `ct235_modalidad_causa_oferta.sql` y
`b136_modalidad_causa_oferta.sql` de sus módulos después de instalar las dos SQL.
Las pruebas comprueban el par de claves, el canon anterior y el nuevo y la
restricción del ayudante SQL; no sustituyen el recorrido con actos autorizados.

En Chrome, publicar desde el expediente fiscalizado de dos plazas. Comprobar en
la oferta de RRHH y en Mi Bolsa las dos claves y la referencia y versión de CT.
Conservar los recibos reales. Repetir con la misma clave después de avanzar el
expediente: debe devolver la misma oferta, sin cambiar sus datos. Un cambio de
modalidad o causa antes de publicar debe impedir el efecto. Reiniciar aplicación
y PostgreSQL y volver a consultar y recuperar la operación. Comprobar también el
arranque sin activar el puente CT-Bolsa y con su configuración existente.

Estado a 18/10: pruebas focales de Go conformes; 75 migraciones instaladas en la
copia propia (73 del prefijo de main y CT235/B136), con las dos pruebas SQL
conformes. Las SQL ensayadas corresponden al commit `53593f247`; la lista del
prefijo procede de `8dd52c11` (#1195) y todas sus huellas coinciden con main.
Semgrep local sin hallazgos. Quedan pendientes la reserva y publicación reales,
el recorrido en navegador y el reinicio. La composición de CT en la copia
depende del trabajo F0-28.
