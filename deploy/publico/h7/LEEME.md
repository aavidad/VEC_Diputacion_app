# Preparar el proxy público y comprobar la convivencia

H7 prepara una entrada HTTPS para `vec-publico` y un comprobador de lectura para
observarlo junto a `vec-server` interno. Estos seis archivos no instalan Caddy,
servicios, certificados ni SQL. Tampoco reinician procesos o publican DNS.

La UI V2 de #220 está integrada en `91e56fdc30cb9045343b317194850b08511247b7`.
El paquete público de #221 tiene como candidata
`e6f69a7d9740d6d14cad977788978117d695a648`: once archivos propios, dos revisiones
sensibles y seis comprobaciones PostgreSQL acreditadas por su acta. La prueba
combinada de esa acta corresponde a `b1d1c373fb3ba297e3db3fb67c4a6b6cafdb4473`,
con Chrome a 1440 y 390 px. H7 conserva ambas entregas y sus límites.

La simultaneidad de los dos procesos sigue pendiente. El ensayo anterior de H1
no llegó a acreditar el arranque interno por diferencias en su inventario de ACL
y tipos. Antes de usar este comprobador, D/M deben entregar un clon H6 READY y su
acta revisada, con la fuente, el binario y el contrato de arranque. Una huella de
un fichero acredita sus bytes; no convierte su contenido en un dictamen READY.

## Configuración del proxy

Guarde el JSON fuera de Git, en un directorio propio `0700`, con permisos `0600`.
Los archivos deben ser regulares, tener un solo enlace y carecer de escritura
para grupo u otros. La clave privada debe ser `0600` y pertenecer al ejecutor.
Se rechazan enlaces simbólicos, claves JSON repetidas, opciones desconocidas y
rutas con componentes `.` o `..`. Las rutas admiten letras ASCII, números,
espacios, guiones, puntos y barras; se entrecomillan al generar Caddy.

El JSON tiene exactamente estos campos, sin valores predeterminados:

| Campo | Valor que debe aportar el responsable del clon |
| --- | --- |
| `public_host` | Nombre DNS completo del host público, en minúsculas y sin comodín. |
| `public_port` | Puerto HTTPS público, entero entre 1 y 65535. |
| `upstream_address` | IP numérica de loopback donde escucha exclusivamente `vec-publico`; admite IPv4 o `::1`, excluye IPv6 con IPv4 mapeada. |
| `upstream_port` | Puerto HTTPS del proceso público, entero entre 1 y 65535. |
| `upstream_server_name` | Nombre DNS que debe validar el certificado de ese proceso; también será su cabecera Host. |
| `upstream_ca_file` | Ruta absoluta al PEM de su CA aprobada. |
| `edge_certificate_file` | Ruta absoluta al certificado HTTPS del host público. |
| `edge_key_file` | Ruta absoluta a su clave privada. |

La plantilla atiende el host completo y conserva los caminos. Su único destino
es el proceso público por HTTPS. `tls_trust_pool file` y `tls_server_name` exigen
la CA y el nombre configurados; no existe una opción para omitir verificación.
No hay destino interno alternativo, `trusted_proxies` ni subcarpeta que mezcle
las superficies.

Caddy añade `X-Forwarded-For`, `X-Forwarded-Proto`, `X-Forwarded-Host` y `Via`; el
backend público rechaza esas cabeceras. La secuencia dentro de `route` rechaza
primero su presencia entrante, incluso vacía, así como `Forwarded` y
`Proxy-Authorization`. Después retira las cuatro cabeceras añadidas por Caddy.
Conserva `Cookie`, `Authorization` y las restantes cabeceras de identidad para
que Go las rechace. La guardia de `Connection` impide que sus tokens supriman
credenciales antes de llegar al backend; incluye valores repetidos.

En el clon privado autorizado, genere un archivo nuevo:

```bash
python3 -B deploy/publico/h7/preparar_proxy.py --config /ruta/privada/proxy.json --output /ruta/privada/Caddyfile.publico
```

El generador carga la CA y comprueba la pareja de certificado y clave. Crea una
salida `0600` y rechaza sobrescribirla. Solo imprime huella y estado de
preparación. No conecta al upstream, ejecuta Caddy ni comprueba DNS, vigencia o
nombre del certificado exterior. Su salida declara `caddy_validated: false`.

Para el Caddy aprobado del destino, conserve las salidas en el directorio
privado. La validación debe usar archivos de certificado y CA accesibles por
el usuario de Caddy:

```bash
umask 077
caddy adapt --config /ruta/privada/Caddyfile.publico --adapter caddyfile > /ruta/privada/caddy-adaptado.json 2> /ruta/privada/caddy-adaptacion.log
caddy validate --config /ruta/privada/Caddyfile.publico --adapter caddyfile > /ruta/privada/caddy-validacion.log 2>&1
```

No se proporciona una orden de instalación o recarga. La composición con el
Caddy existente, su versión, listener, red y certificado exterior requieren
validación del responsable del destino antes de cualquier publicación.

## Comprobación de lectura

El segundo JSON privado tiene exactamente `fuente_commit`, `binario_file`,
`binario_sha256`, `evidencia_h6_file`, `evidencia_h6_sha256`, `publico`, `interno`
y `rondas`. Las tres huellas son hexadecimales en minúsculas: 40 caracteres para
el commit y 64 para SHA256. `binario_file` identifica el binario interno aprobado;
el comprobador coteja sus bytes sin ejecutarlo. Este cotejo no demuestra qué
binario inició el proceso: su procedencia pertenece al acta de arranque. El acta H6 se lee como bytes
privados y se coteja con la huella entregada, sin interpretar un esquema READY
que todavía no se ha acordado. El estado READY se revisa fuera de este guion.

`publico` e `interno` contienen `address`, `port`, `server_name`, `ca_file` y
`checks`. `interno` añade `client_certificate_file` y `client_key_file` para el
certificado sintético de cliente admitido por el clon. `address` siempre es una
IP de loopback; los dos listeners deben ser diferentes. `server_name` es el
nombre DNS verificado en TLS y enviado como Host. La conexión usa directamente
esa IP, sin resolver el nombre ni usar los proxies del entorno. Para comprobar
Caddy, `publico` debe señalar su listener de prueba y su CA; para comprobar los
dos procesos directamente, debe señalar el listener HTTPS de `vec-publico`.
Conserve ambas pruebas con sus respectivas configuraciones si necesita acreditar
ambos recorridos.

Cada entrada de `checks` tiene exactamente `path` y `sha256`. La huella procede
del cuerpo esperado y revisado del clon sintético; no se aprende de la respuesta
durante la comprobación. Todas estas lecturas deben devolver `200`:

| Superficie | Lecturas obligatorias |
| --- | --- |
| Pública | `/`, `/bolsa/`, `/readyz`, las listas `/api/publico/bolsa/convocatorias` y `/api/publico/bolsa/bolsas`, un detalle existente de convocatoria y una lista existente `/api/publico/bolsa/bolsas/REFERENCIA_PUBLICA/lista`. |
| Interna | `/livez`, `/readyz` y `/portal-empleado/`. |

Puede añadir lecturas públicas bajo `/api/publico/bolsa/` y lecturas internas
bajo `/portal-empleado/` o `/api/vec/`, siempre que su contrato aprobado admita
GET sin efectos de negocio. El guion limita cada superficie a 32 lecturas,
rechaza rutas repetidas, consultas, escapes, redirecciones y caminos ajenos.
Los datos y referencias deben ser sintéticos. Ninguna lectura GET acredita
autorización de una escritura ni el recorrido completo de RRHH.

`rondas` es un entero entre 2 y 5. Cada ronda empieza las comprobaciones pública
e interna con una barrera común. Contrasta las respuestas con las huellas
esperadas y entre rondas; no conserva los cuerpos en su salida. Exige `404` en
ocho rutas privadas de la superficie pública y `400` ante cabeceras de
credenciales o identidad sintéticas. Rechaza `Set-Cookie`, `Location`, trailers,
cuerpos mayores de 4 MiB y fallos de TLS. La espera de socket tiene un límite de
diez segundos; no es un plazo total del recorrido. No sigue redirecciones ni
persiste cookies.

El acceso a `/portal-empleado/` sin certificado de cliente debe ser denegado
por TLS, cierre de conexión o `401`/`403`; una lectura autenticada posterior
debe conservar su respuesta. Esa observación comprueba el rechazo anónimo en
ese instante. La política de certificado, identidad, revocación y permisos
sigue perteneciendo a la frontera interna de H6.

```bash
python3 -B deploy/publico/h7/comprobar_coexistencia.py --config /ruta/privada/coexistencia.json
```

Un resultado `checks_passed` informa las huellas de procedencia, el número de
rondas y la huella conjunta de las respuestas. Declara que no acredita READY,
reinicio ni escrituras. Los errores se reducen a un código fijo, sin host,
ruta privada, clave, respuesta o credencial. El guion no instala ni arranca
ningún proceso. La continuidad observada son lecturas repetidas de servicios
ya preparados; no sustituye recuperación tras reinicio.

## Pruebas de este corte

Python 3.11 o posterior y OpenSSL local permiten generar material TLS sintético
temporal y ejecutar las pruebas:

```bash
python3 -B -m unittest discover -s deploy/publico/h7 -p 'test_*.py' -v
```

Las pruebas de Caddy requieren `H7_CADDY_BINARY`, ruta al binario local aprobado.
Si no está disponible, esas pruebas se omiten y debe anotarse la omisión. El
resto comprueba configuración, archivos privados, TLS, mTLS, lectura concurrente,
huellas, exposición privada y rechazo de cookies y redirecciones.

Para este corte se usa Caddy `v2.11.3` de la imagen ya presente
`caddy@sha256:86deaf5e3d3408a6ccec08fbb79989783dd26e206ae10bcf78a801dc8c9ab794`.
El binario extraído en un directorio temporal tiene SHA256
`f16be85d67d7a8369c7255a8514204112bb63a58490bba7d458b38818a75fb94`.
La extracción se realiza con `--pull never --network none --read-only --rm`.
Las pruebas ejecutan ese binario con listeners propios de loopback, certificado
sintético, administración desactivada y sin redirección HTTP automática.
Comprueban `adapt`, `validate` y el comportamiento real de las cabeceras, incluidas
las vacías y `Connection` repetida, además del rechazo por CA o nombre incorrectos.
Sus procesos y archivos temporales se retiran al terminar.

Resultado focal del 30 de septiembre de 2026: 18 pruebas verdes, ninguna omitida,
incluidas las cinco de Caddy. Semgrep con las tres reglas locales de
`scripts/recorridos/intervencion/semgrep-local.yml`, métricas y comprobación de
versión desactivadas: cuatro archivos Python analizados y cero hallazgos.

La sintaxis procede de la documentación oficial de Caddy:
[reverse_proxy](https://caddyserver.com/docs/caddyfile/directives/reverse_proxy),
[matchers](https://caddyserver.com/docs/caddyfile/matchers) y
[route](https://caddyserver.com/docs/caddyfile/directives/route).
El ensayo sintético de Caddy acredita este comportamiento con esa versión.
La validación del Caddy y los certificados del destino, la simultaneidad sobre
H6 READY y el recorrido de navegador siguen pendientes. Dos revisores
independientes deben revisar el commit final antes de integrarlo.
