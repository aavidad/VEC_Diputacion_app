# Analizador PDF de VEC

Este servicio recibe el original desde la cuarentena de VEC. Solo un PDF
íntegro, sin contenido activo según la política restrictiva y con resultado
limpio de ClamAV puede obtener `limpio`. No admite Office ni sustituye la
autorización, el plan durable o la promoción del objeto.

## Dependencias y aislamiento

- El proceso VEC arranca y vigila su propio `clamd` dentro de `bwrap`, sin red.
  Rechaza sockets ajenos. Antes de arrancarlo copia su configuración a un
  fichero privado de solo lectura y comprueba `AlertExceedsMax yes`, `ScanPDF
  yes`, socket Unix exacto, ausencia de TCP e inclusiones, y límites
  `StreamMaxLength`, `MaxFileSize` y `MaxScanSize` de al menos 32 MiB.
  FreshClam actualiza la base en un proceso separado; el daemon controlado
  la monta en modo solo lectura. Tras actualizarla, hay que reiniciar o
  recargar el daemon. `VERSION` debe informar firmas cargadas con fecha no
  superior a 48 horas. Una firma caducada o la caída del hijo bloquea
  `limpio`. La huella de la política copiada queda disponible para el
  despliegue.
- `qpdf`, `bwrap` y `prlimit` son binarios locales. Cada inspección PDF corre
  sin red, con montajes de solo lectura de bibliotecas y del directorio
  temporal privado, CPU máximo de 4 segundos, memoria virtual máxima de
  512 MiB y salida máxima de 2 MiB. El proceso tiene 5 segundos de plazo.
  El directorio temporal se elimina al terminar.
- El servidor solo escucha en una dirección IP de loopback. Use un canal
  privado para conectar VEC. Requiere TLS 1.3, CA de clientes y una URI SAN
  permitida; el nombre común del certificado no concede acceso.

## Configuración privada

`cmd/vec-analizador` recibe estas variables. Los certificados, claves y el
socket se preparan fuera de Git, con permisos del usuario del servicio:

| Variable | Contenido |
| --- | --- |
| `VEC_ANALIZADOR_ESCUCHA` | IP de loopback y puerto del servicio HTTPS. |
| `VEC_ANALIZADOR_CONECTOR_ID` | Identificador opaco acordado con la composición VEC. |
| `VEC_ANALIZADOR_CLAMD_BIN` | Binario local de `clamd`. |
| `VEC_ANALIZADOR_CLAMD_CONFIG` | Configuración privada del daemon. |
| `VEC_ANALIZADOR_CLAMD_DB_DIR` | Directorio privado de firmas actualizado por FreshClam. |
| `VEC_ANALIZADOR_CLAMD_SOCKET_DIR` | Directorio privado donde el hijo crea `clamd.sock`. |
| `VEC_ANALIZADOR_CLIENTES_URI` | URI SAN `spiffe://...` permitidas, separadas por coma. |
| `VEC_ANALIZADOR_TLS_CERT` | Ruta al certificado del servidor. |
| `VEC_ANALIZADOR_TLS_KEY` | Ruta a la clave privada del servidor. |
| `VEC_ANALIZADOR_CLIENTE_CA` | Ruta a la CA que firma a los clientes. |

El cliente VEC necesita una CA de servidor, certificado y clave propios,
el nombre TLS del servidor y el mismo `ConectorID`. El adaptador no admite
HTTP, redirecciones ni un cliente TLS que omita la validación.

Los directorios de firmas y socket deben existir bajo
`/tmp/vec-analizador-clamd-*` con modo 0700. La configuración debe tener modo
0600 o más restrictivo, `DatabaseDirectory` igual al directorio de firmas,
`LocalSocket` igual a `<socket_dir>/clamd.sock` y `LocalSocketMode 600`.
El servicio no acepta `User`, `TCPSocket`, `TCPAddr`, `LogFile` ni directivas
`Include` en esa configuración. El hijo no dispone de acceso a red.

## Contrato operativo

`GET /v1/capacidades` solo responde cuando `clamd` tiene firmas vigentes y
el sandbox de qpdf procesa un PDF pasivo sintético. GET y POST comparten un
cupo máximo de dos operaciones simultáneas. `POST /v1/analisis` acepta metadatos y archivo por
multipart hasta 32 MiB. El servidor coteja tamaño y SHA256 antes de escanear.
El cliente vuelve a cotejar objeto, versión, huella, tamaño, contexto y la
huella de evidencia del resultado.

La política PDF rechaza cifrado, estructura dañada o parcial, datos tras el
último `%%EOF`, más de una revisión y diccionarios de acciones, formularios,
anotaciones, adjuntos y multimedia. Un error de ClamAV, qpdf, tiempo o firmas
no se convierte en `limpio`. La primera integración debe probar el recorrido
de cuarentena, análisis y promoción con certificados privados y firmas reales
antes de declarar el servicio disponible.
