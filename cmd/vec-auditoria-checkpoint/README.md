# Checkpoints offline de desarrollo

Esta herramienta firma las coordenadas de una extracción manual de auditoría
previamente autorizada y verifica después ese recibo con una raíz pública
conservada por separado. No consulta la base ni concede permisos de extracción.
El rango debe pertenecer a una sola cadena y ser completo y consecutivo. Una
exportación filtrada no sirve como checkpoint de ese rango.

El recibo siempre declara `DESARROLLO`. La firma Ed25519 acredita que el proveedor
con la clave fijada emitió esas coordenadas. No acredita el origen de la
extracción, la ausencia de registros posteriores ni una firma jurídica. La TSA
existente produce un HMAC determinista ligado al checkpoint; no aporta una
fecha independiente. Este corte no cierra A3 ni establece un sellado periódico.

## Material y configuración

Prepare fuera de Git una copia de la configuración
[testdata/config.sintetica.json](testdata/config.sintetica.json) con las referencias
y versiones acordadas para política, clave y proveedores, y los límites de la
operación. El ejemplo solo contiene datos sintéticos. No admite producción.
Todas las entradas JSON exigen claves ASCII en minúsculas, sin duplicadas y
con los campos definidos para cada documento.

La clave maestra KMS y el secreto TSA existentes deben ser archivos externos de
32 bytes, distintos, propiedad del usuario que ejecuta, sin permisos de grupo o
de otros. Use rutas absolutas sin enlaces simbólicos ni enlaces duros. La
herramienta no genera la maestra ni acepta el material privado dentro de Git.
No pase secretos como argumentos: los argumentos contienen solo rutas.

La clave de firma se deriva con el KMS existente: maestra → envoltura → dominio
exclusivo `vec.kms.desarrollo.auditoria.checkpoint.ed25519.v1`. No utiliza las
claves de documentos, copias ni atestación de otros módulos. Cambiar la referencia
o la versión en la configuración no rota por sí solo el material criptográfico:
la rotación necesita aprovisionar material externo y una raíz pública nuevos.

La entrada de emisión es únicamente el objeto de cobertura del verificador
común, con `cadena_id`, `primera_secuencia`, `ultima_secuencia`, `registros`,
`anterior_sha256` y `cabeza_sha256`. No incluya registros, decisiones, identidades
ni preimágenes. La emisión valida estructura y límites; no recalcula la cadena.

## Emisión explícita

Después de obtener esa cobertura por el circuito autorizado, ejecute el binario
compilado con las rutas de su entorno. Todos los destinos deben ser nuevos:

```sh
vec-auditoria-checkpoint -operacion emitir \
  -config /ruta/externa/config.json -entrada /ruta/externa/cobertura.json \
  -kms-master /ruta/externa/kms-master -tsa-secret /ruta/externa/tsa-secret \
  -salida /ruta/externa/recibo.json -spki /ruta/externa/auditoria.der
```

La salida JSON informa del resultado y del SHA256 del SPKI. El fichero DER es la
pública de auditoría. Fije y distribuya ese DER y su huella mediante un canal
confiable separado del recibo. La pública no se incluye en el paquete firmado.
Una pública y una huella entregadas por quien aporta un recibo desconocido no
establecen confianza. Para recibos posteriores mantenga la raíz fijada: no
sustituya la raíz confiable con la pública de cada emisión.

La operación no sobrescribe archivos. Si falla la creación del segundo destino,
puede quedar el primero creado; el resultado será rechazado. Compruebe ambos
artefactos antes de distribuirlos. No reintente sobre los mismos destinos.

## Verificación offline

Use la raíz y la huella que ya conservaba por el canal confiable:

```sh
vec-auditoria-checkpoint -operacion verificar \
  -config /ruta/externa/config.json -entrada /ruta/externa/recibo.json \
  -spki /ruta/confiable/auditoria.der -pin-spki-sha256 HUELLA_CONSERVADA
```

Puede añadir `-cadena /ruta/externa/extraccion.json` para recalcular el rango con
el verificador común AD3 v1 o mixto v2. Esa extracción conserva su autorización
y custodia independientes; la CLI no la obtiene ni vuelve a validar permisos.

El resultado separa `firma: verificada_con_pin_externo` de
`integridad_cadena: no_evaluada` o `verificada`. Siempre conserva
`origen_extraccion: no_acreditado`, `tsa: no_verificada_offline`,
`tiempo_independiente: false` y `firma_legal: false`. Una cadena alterada devuelve
código de salida 1 aunque la firma del checkpoint sea correcta. La verificación
no requiere secretos. La firma cubre el checkpoint canónico, esquema y
versiones, procedencia de desarrollo, recibo TSA y huella SPKI.

## Continuidad entre checkpoints

Conserve un recibo previo por un canal confiable independiente del lote. Compruebe
los recibos posteriores con esa ancla, el DER y la huella que ya tenía fijados:

```sh
vec-auditoria-checkpoint -operacion verificar-continuidad \
  -config /ruta/externa/config.json -ancla /ruta/confiable/recibo-previo.json \
  -entrada /ruta/externa/lote.json -max-recibos 256 \
  -spki /ruta/confiable/auditoria.der -pin-spki-sha256 HUELLA_CONSERVADA
```

El lote contiene únicamente `esquema: vec.auditoria.continuidad.desarrollo.v1`
y `recibos`, una lista de 1 a `max-recibos` recibos completos del formato anterior.
El máximo admitido es 256. Todos los campos son obligatorios, incluso los ceros
del ancla vacía; se rechazan claves repetidas, desconocidas y valores `null`.
No admite `-cadena`, secretos ni destinos de emisión.

Se verifican las firmas del ancla y de cada sucesor con la raíz externa. Política,
raíz y cadena deben ser idénticas. Cada sucesor debe ser un tramo no vacío que
empiece después del anterior y cuyo hash anterior coincida con su cabeza.
`max_bytes` limita la suma de los dos documentos; `max_registros` cuenta el ancla
y los sucesores. `recibos_verificados`, las coordenadas y `registros_total` incluyen
el ancla. Esta puede ser vacía si conserva las coordenadas canónicas 0/0 y los
dos hashes de ceros.

El resultado separa firma y continuidad verificadas de `integridad_cadena:
no_evaluada`. Conserva las limitaciones de origen, TSA, tiempo y firma legal ya
descritas. Una firma correcta puede acompañar un rechazo por hueco, solape o
retroceso. La herramienta no acredita la procedencia del ancla, la ausencia de
registros posteriores, la periodicidad del sellado ni la integridad de registros
que no recibió. No consulta ni exporta datos personales.

Los errores devuelven códigos JSON sin reproducir las entradas ni rutas. El
resultado técnico se registra en stderr por el emisor común, con correlación
propia, componente auditoría y códigos cerrados. Ese registro técnico no
sustituye la auditoría funcional de acceso, extracción o descarga.

## Alcance de la entrega

Dominio, puertos, aplicación, configuración, proveedor y CLI forman las seis
piezas de producción mínimas del mismo recorrido. No añade tablas, rutas HTTP,
consumidores nominales ni cambios al verificador existente. Quedan pendientes
captura durable autorizada, política periódica y validación de una TSA
independiente, además de las decisiones de conservación y custodia aplicables.

Comprobaciones de continuidad del 04/10: pruebas focales normales y con `-race`
en CLI, aplicación y dominio, usando el proveedor KMS/TSA existente; `go vet`
en esos tres paquetes; Semgrep `p/golang` sobre los siete archivos Go cambiados
(42 reglas, sin hallazgos); formato y `git diff --check` correctos. Las pruebas
usaron un directorio temporal externo a Git. Gosec analizó 115 archivos de esos
paquetes: 25 avisos en archivos sin cambios, ninguno en los archivos cambiados
y ningún error de carga. Ese resultado no es una revisión limpia de todo el
código previo. No se ejecutaron servicios, SQL, navegador ni la suite global
local. Las revisiones sensibles del commit final corresponden a la integración.

## Comprobar un paquete de exportación de desarrollo

`verificar-exportacion` comprueba el manifiesto firmado y el archivo exacto de
la cadena. La raíz pública y su huella deben proceder de un canal confiable
separado. La raíz de exportación tiene su propio dominio de derivación; no use
el DER de los checkpoints anteriores. No se requieren claves privadas:

```sh
vec-auditoria-checkpoint -operacion verificar-exportacion \
  -config /ruta/externa/config.json -entrada /ruta/externa/recibo-exportacion.json \
  -cadena /ruta/externa/cadena.json \
  -spki /ruta/confiable/exportacion.der -pin-spki-sha256 HUELLA_CONSERVADA
```

El recibo liga esquema, política y proveedores, referencia y huella del acuse
de captura, fecha declarada, cobertura, tamaño y SHA256 exactos del archivo,
y el aviso de fechas históricas no ligadas. Un cambio de espacios en el archivo
también cambia su huella. La suma de archivo y recibo no puede superar
`max_bytes`; `max_registros` limita las filas. Todos los campos del recibo son
obligatorios. Se rechazan claves repetidas, desconocidas, alias y valores nulos.

Un resultado `verificada` confirma la firma con la raíz fijada, los bytes y los
eslabones admitidos. `historicos_sin_fecha_ligada: true` advierte de tramos cuyas
fechas no estaban protegidas por su eslabón. El informe conserva siempre
`origen_extraccion: no_acreditado`: la referencia y la huella del acuse ligadas
a la firma no prueban por sí mismas que una autoridad permitiera la extracción.
Conserva también `tsa: no_verificada_offline`, `tiempo_independiente: false` y
`firma_legal: false`. La TSA existente es HMAC de desarrollo.

La aplicación puede preparar ese recibo únicamente a través de
`FuenteCapturaExportacionAuditoria`: la fuente debe consumir la autorización de
exportación, fijar un rango coherente y registrar su acuse en la auditoría común
en la misma transacción, antes de devolver datos tras COMMIT confirmado. La
firma y el sello se producen después de esa captura. Un error de captura o de
verificación devuelve cero datos y no invoca los proveedores criptográficos.
No hay una opción CLI para emitir paquetes desde un JSON de captura libre.

Todavía falta el adaptador de captura nominal y su catálogo de acciones para
Aplicación. No se habilita extracción, entrega al juzgado, descarga administrativa
ni sellado periódico con esta pieza. Los ensayos del formato usan datos sintéticos.
El evento de captura futuro registrará la cabeza previa; quedará fuera de su
propio rango para evitar una referencia circular. Los fallos y la recuperación
de entrega deberán conservar su auditoría nominal antes de activar esa ruta.
