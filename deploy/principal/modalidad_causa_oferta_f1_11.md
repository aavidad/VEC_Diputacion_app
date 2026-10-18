# Modalidad y causa en las ofertas — F1-11

El cambio conserva las claves del análisis de Contratación temporal en los datos
de la oferta de Bolsa. Las vistas de RRHH y Mi Bolsa reciben esos datos por sus
lecturas actuales. No añade filtros ni cambia las respuestas del aspirante.

## Instalación

Usar PostgreSQL 18 y una copia propia del volcado de referencia HZ37. Instalar
la lista completa `lista_sql_main_tras_hz37.txt`, sin combinarla con listas
antiguas. Preparar antes los LOGIN nominales que exige AD278, siguiendo
`LEEME_copia_local.md`; ese ajuste es un dato de copia, no una concesión nueva.
La lista tiene 87 entradas: las 85 de main y, al final, CT235 y B136 en ese
orden. El subconjunto `lista_sql_codexo_modalidad_causa_oferta_f1_11_20261018.txt`
sirve solo para una copia que ya tenga las 85 anteriores; no se combina con la
lista completa. Las guardas cotejan definiciones, propietarios, configuración
y permisos instalados; cualquier diferencia revierte la migración.
No se modifica el núcleo de autorización ni se añade configuración al arranque.

Las nuevas reservas exigen un análisis con modalidad y causa válidas y la versión
vigente del expediente. Se obtiene mediante el análisis ordinario de RRHH. Una
oferta anterior se recupera con el comando guardado, sin completar sus campos
con el análisis actual. Las ofertas nativas sin origen de CT mantienen su contrato.

## Comprobación

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

Estado a 18/10, candidata `6f95ef7e9`: en una copia nueva se instalaron una vez
las 87 migraciones (85 de main, CT235 y B136), con 87 confirmaciones y ningún
error. Las dos pruebas SQL focales terminaron sin error y revirtieron sus
transacciones. Las cinco puertas de CI pasaron. Las pruebas Go y las dos
revisiones independientes del cambio son favorables. Chrome abrió la ficha de
Contratación temporal de RRHH con respuesta 200. Esa lectura no acredita una
reserva ni una oferta publicada desde el expediente.

La activación CT→oferta sigue pendiente. La receta compartida aún no aporta
los tres perfiles y motivos publicados por Administración ni el material
privado `identidad/ofertas-ct.json` que exige el puente. También se debe cotejar
el cliente M10 de organización indicado por `VEC_CT_BOLSA_ORGANIZACION_SOURCE_PATH`
con el organismo y el catálogo de puestos de la copia. El archivo privado no
concede permisos; la guía `activacion_codexs_varias_plazas_ct_20261015.md`
describe las vías de gobierno que deben comprobarse antes de usarlo.

Las altas directas enviadas por HTTP no prueban que la petición original llevara
`numero_personas` literal ni que conservara su necesidad de la RPT. La vía del
centro sí conserva esa necesidad. El caso plural debe nacer por las pantallas
del producto, con una modalidad cuyo catálogo permita dos personas, y llegar
a fiscalización favorable con Intervención. La modalidad de sustitución solo
permite una persona. No se atribuye a los expedientes singulares anteriores una
fiscalización o un recibo plural.

Quedan por comprobar el arranque sin y con la configuración privada, la reserva
y publicación desde el expediente, sus recibos, la lectura en RRHH y Mi Bolsa,
la recuperación con la misma clave y el reinicio de aplicación y PostgreSQL.
La guarda de publicación exige modalidad y causa en ofertas CT nuevas después
de permitir la recuperación literal de una oferta histórica; esto está cubierto
por pruebas focales, pero todavía no por ese recorrido. F1-11 no se declara
terminada ni lista para instalar por la sola lectura 200 de la ficha.
