# Preparar y aplicar la unidad inicial de ADMIN

El comando prepara un plan para una única unidad declarada sintética. Puede
cotejar el documento guardado y aplicarlo por el canal técnico de Personal33.
No crea LOGIN ni publica configuración, cuentas, perfiles o permisos.

La fuente separada contiene exactamente `version`, `referencia`, `entorno`,
`alcance_fuente`, `unidad` y `acto_tecnico_ref`. El plan conserva esos mismos
metadatos y añade operación, preparación, caducidad y evidencia de la fuente.
`fuente.huella_sha256` es SHA256 del canon de la fuente, sin un campo de huella
propia. Los ejemplos están en `testdata/fuente.sintetica.json` y
`testdata/material.sintetico.json`; sus fechas son fijas para las pruebas.

El contrato exige versión 1, entorno `desarrollo` y alcance
`sintetico_declarado`. El catálogo es `estructura-organizativa-dipgra`, versión
1 y revisión 1, con revisión esperada del nodo 0. La clase procede del contrato
existente de Organización: `centro`, `delegacion` o `puesto_responsabilidad`.
La referencia `acto_tecnico:` identifica el acto de inicialización; no acredita
una decisión jurídica ni concede autoridad.

La denominación se conserva completa, con hasta 300 caracteres Unicode, sin
controles ni blancos en los bordes. Se rechaza una entrada distinta en lugar de
recortarla o normalizarla. Las referencias de organización y unidad tienen el
formato de Personal y hasta 128 caracteres; la fuente también tiene hasta 128,
y el acto y la clave de entrada del catálogo hasta 160. El nodo usa UUID en
minúsculas.

Preparación y caducidad usan UTC con segundos. Las vigencias de la unidad son
fechas civiles `AAAA-MM-DD`, con fin exclusivo a las 00:00 UTC. El inicio no
supera la preparación y el fin cubre la caducidad del plan. Los años admitidos
son de 1 a 9999. La preparación comprueba su reloj confiable y rechaza un plan
futuro o caducado.

```sh
GOCACHE=$HOME/.cache/go-build go build -p 8 \
  -o /ruta/privada/vec-preparar-unidad-admin ./cmd/vec-preparar-unidad-admin
/ruta/privada/vec-preparar-unidad-admin \
  --material /ruta/privada/material.json \
  --fuente /ruta/privada/fuente.json \
  --plan /ruta/privada/plan.json \
  --textos /ruta/app/web/static/textos/es/admin-unidad-preparar.json
```

El documento guardado contiene `plan` y `huella_plan_sha256`. El orden de sus
campos lo fija el DTO de dominio; el canon es JSON compacto de Go, UTF-8 y sin
salto final. La huella de la fuente usa el mismo criterio. La presentación
original del JSON de entrada puede variar; sus datos deben coincidir. Añada
`--cotejar` para comprobar un plan existente. El comando no lo sobrescribe si
su contenido diverge.

El DBA prepara por separado el LOGIN exclusivo, la configuración y la
aprobación externa de Personal33. La conexión privada tiene exactamente `dsn`
y `permitir_socket_desarrollo`. La aprobación tiene exactamente
`huella_plan_sha256`. El comando envía esa huella externa sin sustituirla por
su propio cálculo; conocer la huella no concede permiso.

```json
{
  "dsn": "postgres://operador_sintetico@localhost/ensayo?host=/ruta/socket&sslmode=disable",
  "permitir_socket_desarrollo": true
}
```

El ejemplo usa un socket local de desarrollo permitido expresamente. Un destino
remoto exige TLS con verificación del nombre del servidor, también en todos sus
destinos alternativos. No se aceptan `role` ni `options` como parámetros de
sesión. Las credenciales reales permanecen en el archivo privado.

```sh
/ruta/privada/vec-preparar-unidad-admin \
  --material /ruta/privada/material.json \
  --fuente /ruta/privada/fuente.json \
  --plan /ruta/privada/plan.json \
  --textos /ruta/app/web/static/textos/es/admin-unidad-preparar.json \
  --cotejar --aplicar \
  --conexion /ruta/privada/conexion.json \
  --aprobacion /ruta/privada/aprobacion.json \
  --acuse /ruta/privada/acuse-primero.json --timeout 30s
```

La aplicación exige cotejo y un timeout positivo elegido por el operador. Cada
invocación reserva antes de conectar un archivo de acuse nuevo, con O_EXCL.
Las entradas privadas y el acuse usan rutas absolutas, archivos propios 0600 y
un directorio propio 0700 fuera de Git. Se rechazan enlaces, campos adicionales,
campos ausentes, claves JSON repetidas y documentos de más de 64 KiB.

El adaptador abre SERIALIZABLE de escritura, fija UTC y hace una única consulta
parametrizada a `vec_personal.inicializar_unidad_sintetica_admin_v1(text,text,text)`.
No hace SET ROLE ni reintenta automáticamente. Personal coteja la configuración
externa y la preimagen, conserva la fila real y registra auditoría común AD176.

La respuesta cerrada contiene `estado`, `codigo`, `recibo`, `replay` y
`auditoria_intento`. La CLI valida su forma, las coordenadas de auditoría y el
vínculo de la fila con el plan y la fuente. Confirma con COMMIT los tres estados
`permitido`, `denegado` y `error`, y solo entonces guarda la respuesta original
completa en el acuse privado. La salida de consola muestra estado y mensaje
traducido, sin documentos, denominaciones, DSN ni errores SQL originales.

El replay conserva el recibo original y añade otro intento de auditoría. Para
recuperarlo, use el mismo material, fuente y aprobación con otro acuse. Si
COMMIT no se confirma, el resultado es `indeterminado` y no se muestra ni guarda
un recibo como persistido. Si COMMIT está confirmado y falla el archivo, el
diagnóstico distingue ese caso para recuperar el recibo original.

Códigos de proceso: 0 preparado, cotejado o permitido con acuse; 1 entrada
rechazada, operación sin confirmar antes de COMMIT o rechazo/error confirmado
por el servidor; 2 catálogo o salida fallidos, COMMIT sin confirmar o acuse no
guardado después de COMMIT. Sin catálogo se emite únicamente el código de
protocolo `catalogo_no_disponible` a stderr.

Dependencias de aplicación: Personal33 y AD176, instaladas por el director.
Las pruebas con dobles acreditan el protocolo del consumidor; no acreditan esa
instalación. Este comando no asegura auditoría de errores locales, conexiones
fallidas, cancelaciones o invocaciones que no llegan a COMMIT. No crea otra
unidad cuando el inicializador global ya tiene una confirmación distinta.
