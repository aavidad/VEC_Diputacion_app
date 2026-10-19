# Modalidad y causa en las ofertas — F1-11

Una oferta de varias plazas que nace en Contratación temporal guarda en sus
datos la modalidad y la causa del análisis del expediente
(`datos.modalidad_clave` y `datos.causa_clave`). La fuente queda en
`origen_ct`, que ya guardaba la referencia del expediente y su versión. Las
vistas de RRHH y Mi Bolsa reciben esos datos por sus lecturas actuales; no se
añaden filtros ni pantallas.

Cómo se protege:

- CT235 exige el par en toda reserva nueva y lo coteja con la lectura CT221 del
  análisis vigente, en la misma versión del expediente. Si no coincide, la
  reserva se deniega (42501) y no escribe nada.
- B136 vuelve a cotejar el par al publicar y liga las dos claves a la huella
  de la operación (canon v6, encadenado al v5). Una oferta sin origen CT no
  puede llevar el par.
- Una operación anterior a F1-11 se recupera con su comando guardado, sin
  completar campos ni cambiar su huella. Las ofertas nativas de Bolsa siguen
  como estaban.

No toca el núcleo de autorización, ni acciones, audiencias, perfiles o
permisos, ni la configuración de arranque.

## Instalación

PostgreSQL 18 y una copia nueva del volcado HZ37. Instalar la lista común
`lista_sql_main_tras_hz37.txt` entera, una sola vez: CT235 y B136 son sus dos
últimas entradas y van después de AD291. No hay sublista propia. Las guardas
comprueban la huella de cada función que se rehace, su propietario, su
configuración y sus permisos; cualquier diferencia revierte la migración.

| Migración | SHA-256 del fichero |
| --- | --- |
| CT235 `contratacion_temporal/migraciones/000235_modalidad_causa_oferta.up.sql` | `cb33eb2c12fe2f0689c416676b3f47b2620caaa767adb3b9f9bedbe1a2dbe5ed` |
| B136 `bolsa_llamamientos/migraciones/000136_modalidad_causa_oferta.up.sql` | `e761b8607939a58ebcbca9bbbe56a8bcdefe02712dd46c9667a1b70278f1a1b7` |

No reaplicar ninguna de las dos: su guarda previa rechaza una segunda
ejecución.

## Pruebas SQL

Después de instalar, ejecutar como superusuario de la copia, cada una en su
transacción revertida:

1. `contratacion_temporal/pruebas_sql/ct235_modalidad_causa_oferta.sql`: el
   validador del comando admite el formato anterior y el par válido, y rechaza
   pares incompletos, booleanos, nulos o con mayúsculas.
2. `contratacion_temporal/pruebas_sql/ct235_reserva_modalidad_causa.sql`:
   reserva sobre un expediente fiscalizado real de la copia. Solo dentro de la
   transacción, el expediente pasa a pedir dos personas y el consumo de
   autorización se sustituye por un doble. Comprueba que la reserva nueva sin
   el par se rechaza, que una causa distinta del análisis se deniega sin
   escribir y que una operación anterior sin el par se recupera. Con el par del
   análisis reserva una sola vez; el reintento con la misma clave no duplica,
   cambiar la causa después de reservar da colisión y otra clave no abre una
   segunda oferta del mismo expediente. Revisa también los permisos.
3. `bolsa_llamamientos/pruebas_sql/b136_modalidad_causa_oferta.sql`: vectores
   del canon v5 y v6, rechazo del par incompleto y del par sin origen CT al
   publicar, recuperación denegada fuera del LOGIN de Bolsa y permisos del
   ayudante.

## Arranque

Sin el material privado `identidad/ofertas-ct.json`, la aplicación arranca y
la oferta desde CT queda cerrada: la consulta
`GET /api/vec/contratacion-temporal/llamamientos/seleccion/estado` responde 503
`servicio_no_disponible`, con su clave de traducción. El resto de Contratación
y Bolsa responde con normalidad.

## Ensayo del 19/10 (copia `claude-f111b`)

Rama en `f135b4207`, con la principal unida hasta AD291. Copia nueva del
volcado HZ37 con `--rrhh-original`, creada en 88 segundos: 101 migraciones de
la lista común, 101 confirmadas y ningún error. Las tres pruebas SQL terminaron
sin error y revirtieron sus transacciones; la tabla de ejecuciones siguió sin
ofertas. La sonda de la copia dio 200 en salud, Bolsa, catálogos de alta,
circuito de firma y configuración del análisis, y 403 en peticiones del centro
para RRHH, que es lo esperado con este certificado. La consulta de estado de
oferta dio 503. Tras reiniciar aplicación y PostgreSQL se repitieron la sonda,
las tres pruebas y la consulta, con los mismos resultados.

La copia anterior `claude-f111`, sin AD291, mostraba al arrancar el fallo de
gobierno de CT por la audiencia de comprobación del ofrecimiento (23514). Con
AD291 instalada ya no aparece.

## Lo que falta para publicar desde la ficha

La publicación real desde un expediente, con su recibo de Bolsa, no se ha hecho
todavía. Faltan dos piezas ajenas a esta tarea, que no se suplen con datos
inventados:

1. El material `identidad/ofertas-ct.json` pide tres perfiles distintos
   (vínculo, cierre de la intención en Bolsa y descarte), con sus roles y
   motivos publicados por Administración y asignados a la técnica de RRHH. El
   de cierre se autoriza por unidad y ámbito de la bolsa, y hoy ninguna vía del
   producto asigna un rol con esas dos dimensiones: es la pieza pendiente de
   F0-04 (lote ordinario con las dimensiones de cada rol). La forma del fichero
   está en `activacion_codexs_varias_plazas_ct_20261015.md`, apartado
   «Configuración privada».
2. Un expediente de dos personas fiscalizado favorablemente. El alta directa ya
   transmite el número de personas (F1-73), pero la fiscalización exige el
   informe jurídico firmado y la copia no monta el circuito de firma. La
   alternativa que admite `LEEME_copia_local.md` es la variante plural del caso
   4.0 como dato de copia, fiscalizada después por la web con Intervención.

Con esas dos piezas, el recorrido es: en Chrome, publicar desde la ficha del
expediente; comprobar en la oferta de RRHH, en Mi Bolsa y en la consulta de
estado de CT la modalidad, la causa, la referencia del expediente y su
versión; repetir con la misma clave y obtener la misma oferta y los mismos
recibos; y reiniciar aplicación y PostgreSQL y volver a consultar.
