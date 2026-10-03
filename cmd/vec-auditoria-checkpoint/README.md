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

Comprobaciones locales de este corte: pruebas focales normales y con `-race` en
CLI y proveedor; `go vet` en CLI, bootstrap, config, dominio, puertos y aplicación;
Semgrep `p/golang` sobre los ocho archivos Go nuevos (42 reglas, sin hallazgos);
gosec de la CLI sin hallazgos; tamaño de archivos y `git diff --check` correctos.
El análisis gosec de bootstrap con las versiones 2.25 y 2.29 tuvo los mismos
errores de resolución y hallazgos en archivos sin cambios. No acredita una
revisión completa del paquete; su causa concreta de carga sigue pendiente.
No se ejecutaron servicios, SQL, navegador ni la suite global. Las dos revisiones
sensibles sobre el commit final corresponden a la integración.
