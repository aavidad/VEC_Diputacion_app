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
el verificador común. Admite los esquemas v1, mixto v2, v3, fuentes iniciales,
unidad inicial y bootstrap central. El lector compartido exige campos completos,
rechaza duplicados, alias, valores nulos y esquemas desconocidos, y coteja el
manifiesto con la cobertura firmada. La extracción conserva su autorización y
custodia independientes; la CLI no la obtiene ni vuelve a validar permisos.

El resultado separa `firma: verificada_con_pin_externo` de
`integridad_cadena: no_evaluada` o `verificada`. Siempre conserva
`origen_extraccion: no_acreditado`, `tsa: no_verificada_offline`,
`tiempo_independiente: false` y `firma_legal: false`. Una cadena alterada devuelve
código de salida 1 aunque la firma del checkpoint sea correcta. La verificación
no requiere secretos. La firma cubre el checkpoint canónico, esquema y
versiones, procedencia de desarrollo, recibo TSA y huella SPKI.

`consumos_historicos_sin_fecha_ligada` avisa de consumos anteriores cuyas fechas
no quedaron ligadas al eslabón. `fecha_consumo_ligada_cotejada` confirma el
cotejo de las fechas nominales AD173 cuando están presentes. Ambos indicadores
solo se informan después de verificar toda la cadena; si se rechaza el
documento o se omite `-cadena`, permanecen falsos. El aviso histórico no
completa ni firma retroactivamente una fecha y el cotejo AD173 no acredita
tiempo independiente.

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

También admite los esquemas de gobierno de usuarios AD188 y frontera técnica
ADMIN AD189, junto a las familias anteriores. Comprueba sus huellas y fechas
con los verificadores comunes. Estos actos identifican el LOGIN técnico; no
acreditan una persona ni un perfil humano. Alterar la fecha o un código del
registro invalida la cadena, aunque el archivo tenga una firma correcta. Un
esquema anterior no admite estas familias nuevas.

Un resultado `verificada` confirma la firma con la raíz fijada, los bytes y los
eslabones admitidos. `historicos_sin_fecha_ligada: true` advierte de tramos cuyas
fechas no estaban protegidas por su eslabón. El informe conserva siempre
`origen_extraccion: no_acreditado`: la referencia y la huella del acuse ligadas
a la firma no prueban por sí mismas que una autoridad permitiera la extracción.
Conserva también `tsa: no_verificada_offline`, `tiempo_independiente: false` y
`firma_legal: false`. La TSA existente es HMAC de desarrollo.

Los esquemas mixtos conservan también `consumo_confirmado_v4` de AD193. El
objeto `consumo` añade dos cadenas decimales uint64 obligatorias e iguales:
`transaccion_origen` de auditoría y `transaccion_consumo_origen` del consumo.
El eslabón añade sólo el primero como campo 16; el segundo se coteja sin
añadir otro encuadre. Se rechazan números JSON, valores no canónicos y sellos
distintos. Los consumos históricos conservan sus formatos y huellas. El
cotejo de XID no acredita procedencia global, firma COSE ni instalación SQL;
la exportación mantiene el pin externo y todos los límites de desarrollo.

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

## Sello periódico con autoridad técnica propia

La operación `ejecutar-periodico` consulta las fachadas AD186 y conserva el
recibo en PostgreSQL. Un temporizador externo la invoca; el intervalo vigente
de la política SQL decide si corresponde otra captura. Cada comprobación y
recuperación queda auditada. El comando no obtiene filas de personas.

La configuración del ejecutor contiene `version: 1`, `max_registros`,
`version_binario`, `pin_spki_sha256` y `timeout_segundos`. La cadencia, cadena,
política criptográfica y raíz aprobada pertenecen a la configuración SQL
versionada, publicada por CAS y auditada. La configuración no tiene plazos de
conservación ni autoriza borrados. La duda 84 sigue pendiente.

Dirección instala AD186 después de AD183, sobre la preimagen acreditada.
No se reaplican dependencias. Una variante del CHECK requiere medir y revisar
su preimagen; la migración rechaza otra estructura. La función de configuración
exige el grupo `vec_auditoria_periodica_configurador`; el ejecutor exige
`vec_auditoria_periodica_sellador`. Las cuentas LOGIN propias se aprovisionan
fuera de Git con CONNECT y SET del grupo exacto, sin INHERIT ni membresías de
owner. El adaptador activa únicamente ese grupo. No concede permisos por
petición. Las tablas y el helper permanecen privados.

Configure la política mediante `configurar_sello_periodico_v1` en una
transacción SERIALIZABLE/UTC con preimagen y versión esperadas. Ante fallo,
haga ROLLBACK y registre el intento con `registrar_intento_periodico_v1`
en otra transacción; el adaptador `ConfigurarCheckpointPeriodico` ya aplica
ese contrato. La política incluye versión, cadena, intervalo en segundos,
estado activo, política del checkpoint y pin SPKI. Una captura pendiente
bloquea el cambio; una raíz con capturas anteriores permanece inmutable.
La rotación después de capturas exige un protocolo posterior.

Ejecute con archivos externos privados; la conexión TCP exige verificar el
servidor por TLS y no admite caída a texto claro. El socket local sirve al
ensayo aislado:

```sh
vec-auditoria-checkpoint -operacion ejecutar-periodico \
  -config /ruta/privada/ejecutor.json -conexion /ruta/privada/conexion \
  -kms-master /ruta/privada/kms-master -tsa-secret /ruta/privada/tsa-secret
```

Si no vence el intervalo, devuelve `no_vencido` con acuse y no firma. Cuando
vence, captura la cabeza previa y añade su acuse en la misma transacción.
El checkpoint termina en ese acuse; el material de este liga la cabeza
previa, evitando autorreferencia. Firma y TSA se ejecutan tras COMMIT. Otra
transacción conserva recibo y auditoría; el resultado queda cubierto por un
checkpoint posterior. Una carrera serializable se rechaza y audita.

Ante una respuesta perdida conserve `captura_ref` y recupere con los mismos
archivos y `-captura-ref REFERENCIA`. Si ya estaba confirmado, devuelve el
recibo conservado, sin otra firma ni confirmación. Un COMMIT incierto no se
presenta como rollback. PostgreSQL conserva texto y SHA256 exactos; la salida
`recibo_texto` permite recuperar esos bytes, incluso con formato alternativo.
Guárdelos sin añadir un salto de línea antes de comprobar su huella.

Compruebe el recibo con `verificar`, la raíz y el pin conservados por un canal
separado. El nuevo formato `vec.auditoria.verificacion.periodica.v1` permite
recalcular los eslabones técnicos junto a las familias anteriores. El detalle
se reconstruye con la serialización PostgreSQL original; los saltos del
transporte Base64 no cambian sus bytes. El sello por sí solo no recalcula
registros ni acredita ausencia de otros posteriores.

Todas las salidas mantienen DESARROLLO, TSA HMAC sin tiempo independiente y
firma legal falsa. La captura técnica no sustituye la captura judicial
nominal ni concede consulta administrativa de personas.

El diseño sigue los patrones de resúmenes periódicos enlazados de
[CloudTrail](https://docs.aws.amazon.com/awscloudtrail/latest/userguide/cloudtrail-log-file-validation-digest-file-structure.html),
estado verificable con raíz separada de
[immudb](https://docs.immudb.io/1.5.0/management/state) y conservación de auditoría
para completar operaciones de [Vault](https://developer.hashicorp.com/vault/docs/audit).
No incorpora esos servicios ni les envía datos.

## Preservación técnica provisional (AD187)

`configurar-preservacion` publica una versión por CAS y referencia idempotente.
`consultar-preservacion` consulta la vigente (`--version 0`) o una versión
histórica. Ambas operaciones usan grupos técnicos propios y dejan auditoría
común antes de devolver datos. El recibo original se conserva; cada replay o
consulta añade su registro de acceso.

La medida es `conservar_todo_sin_expurgo`, con estado `provisional`, para el
conjunto técnico `auditoria_periodica:comun_interna`. No es una serie documental,
un plazo efectivo, una resolución del Archivo ni una autorización de borrado.
La referencia de decisión técnica identifica una decisión externa al código;
no la convierte en resolución legal. La duda 84 sigue abierta.

Configuración del ejecutor, sin credenciales ni reglas documentales:

```json
{"version":1,"timeout_segundos":30,"version_binario":"desarrollo:preservacion:1"}
```

Entrada sintética de publicación (conservarla para recuperar la operación):

```json
{"publicacion_ref":"preservacion_11111111111111111111111111111111","version":1,"preimagen_sha256":"0000000000000000000000000000000000000000000000000000000000000000","decision_tecnica_ref":"decision_tecnica_22222222222222222222222222222222","estado":"provisional","medida":"conservar_todo_sin_expurgo"}
```

```sh
vec-auditoria-checkpoint --operacion configurar-preservacion --config ejecutor.json --conexion conexion-configurador.txt --entrada publicacion.json
vec-auditoria-checkpoint --operacion consultar-preservacion --config ejecutor.json --conexion conexion-consultor.txt --version 0
```

Los archivos de conexión son privados, fuera de Git, con LOGIN nominal propio,
NOINHERIT y concesión del grupo exacto de la operación. TCP exige verificación
TLS del servidor y no admite alternativas en texto claro; el socket local se
reserva al clon de desarrollo. Nunca se necesitan claves KMS/TSA en estos modos.

La versión nueva exige la huella de la versión anterior. La primera exige
preimagen cero. El material idempotente y su huella son el objeto JSON cerrado
normalizado por PostgreSQL `jsonb::text` (SHA256 UTF-8). Un orden o espaciado
diferente con los mismos seis valores conserva la operación; una referencia
reutilizada con material distinto se rechaza. Un COMMIT incierto no devuelve
versión, huella ni acuse: repetir la entrada original permite recuperarla.

AD187 depende de AD186 y de su CHECK medido. No se reaplica ni se ejecuta DOWN.
El código requiere revisión y ensayo en el clon antes de instalarse; este
archivo no acredita instalación en la principal. No cambia la política
criptográfica del sello ni borra registros.
