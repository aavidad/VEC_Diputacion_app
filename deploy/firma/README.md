# Validador de firma como servicio aparte

Decisión del operador (25/09/2026): VEC firma en el equipo de la persona por el
protocolo AutoFirma y **verifica en servidor con el validador de AutofirmaV2**,
desplegado como servicio independiente en modo `-rest-solo-verificacion`. VEC
no importa código de AutofirmaV2: lo consume por HTTP mediante el adaptador
`internal/vec/documentos/adapters/validadorautofirma`, que implementa los
puertos `VerificadorFirma` y `VerificadorFirmaMotivado` de Documentos.

Principio: **la verificación no depende de otras aplicaciones**. Anclas de
confianza y CRL son ficheros locales; el sello de tiempo (TSA) y OCSP quedan
preparados como extensiones, pero no son imprescindibles.

Este documento describe el despliegue previsto. **No está desplegado ni
probado en Cidonia**, ni con certificados, CRL o sellos de prestadores reales,
y ningún flujo de negocio de VEC lo usa todavía.

## Política de verificación de firma

VEC aplica `politica:vec:firma:verificacion-autonoma:v1`
(`ports.PoliticaVerificacionFirmaV1`). Cambiar cualquiera de estas reglas
exige una versión nueva de la política, no editar la vigente:

| Aspecto | Regla v1 |
|---|---|
| Confianza | Solo anclas locales montadas en el validador; el almacén del sistema no cuenta. Sin ruta hasta ellas: `indeterminada`. |
| Revocación | Solo `vigente` permite `valida`, con evidencia local (CRL del directorio montado o embebida en la firma). `revocado`: `no_valida`. `no_comprobada`: `indeterminada`. OCSP remoto es opcional y está desactivado. |
| Sello de tiempo | Opcional. `no_presente` y `no_comprobado` no bloquean: la validez se evalúa en el instante de la comprobación y VEC no se apoya en el sello como prueba de tiempo. Un sello presente y `no_valido`: `indeterminada`. |
| Vínculo | La firma debe cubrir el original custodiado (`acreditado`). VEC envía siempre el original, también en PAdES; `no_aportado` o `no_acreditado`: `indeterminada`. |
| Firmante | Un único firmante, identificado por la huella SHA-256 de su certificado. La correspondencia con la persona la resuelve la aplicación. |
| Integridad | `valida`; `parcial` (contenido no cubierto): `indeterminada`; `no_valida`: `no_valida`. |

Todavía no hay validación a largo plazo con prueba de existencia: vigencia,
cadena y revocación se evalúan en el instante de la comprobación.

## Contrato consumido

`POST /verify` con JSON:

| Campo | Uso por VEC |
|---|---|
| `name` | Fijo `documento`; nunca el nombre real del fichero. |
| `content_base64` | Artefacto firmado. |
| `original_content_base64` | Original custodiado, siempre (CAdES, XAdES y PAdES). |

De la respuesta VEC lee **solo** `dictamen`, con contrato
`autofirmav2.dictamen-verificacion.v1`; ignora los campos heredados (`valid`,
`result`, `details`). Asunto, emisor, serie, motivos técnicos, fuentes y
fechas del proveedor se descartan al decodificar.

Mapeo al resultado de VEC:

| Dictamen | Resultado VEC |
|---|---|
| `estado` / `motivo` | Catálogo cerrado `MotivoVerificacionFirma` (incluye `integridad_parcial`). |
| `vinculoOriginal.estado == acreditado` | `VinculoOriginal` |
| `certificadoHuellaSHA256` (un solo firmante) | `CertificadoHuellaSHA256` y `FirmanteRef = ref:<huella>` |
| `revocacion.estado` | `RevocacionEstado` (`vigente`/`revocado`/`no_comprobada`) |
| `selloTiempo.estado` | `SelloTiempoEstado` (`no_presente`/`valido`/`no_valido`/`no_comprobado`) |
| `huellaFirmadoSHA256`, `huellaOriginalSHA256` | Se comparan con las calculadas por VEC; nunca se adoptan. |

VEC no delega el veredicto: lo recalcula con su política sobre los aspectos
del dictamen. Si coincide con el del validador, lo adopta; si el validador
dice `valida` y VEC es más estricta (por ejemplo, original no aportado o
varios firmantes), prevalece el motivo de VEC; cualquier otra discrepancia es
`respuesta_no_interpretable`.

## Topología (podman)

Un pod `vec-firma` con dos contenedores que comparten loopback, más un
proceso aparte de refresco de CRL:

1. **validador**: AutofirmaV2 sin interfaz gráfica en modo
   `-rest-solo-verificacion`, escuchando en `127.0.0.1:63118`. Solo expone
   `GET /health` y `POST /verify`; no carga certificados ni claves de firma,
   no tiene rutas de fichero ni hace **ninguna conexión saliente**. Anclas y
   CRL se montan **en solo lectura**. Exige token Bearer fuera de loopback.
2. **proxy**: termina TLS 1.3 con **autenticación mutua** hacia VEC, admite
   solo `POST /verify` (y, si se quiere supervisar, `GET /health`), limita el
   cuerpo (48 MiB: dos contenidos de 16 MiB en base64) y el tiempo, y reenvía
   a `127.0.0.1:63118` añadiendo el token Bearer.
3. **refresco de CRL** (contenedor o temporizador separado, fuera del pod):
   descarga periódicamente las CRL de cada CA de la ruta, **incluida la ARL
   de la raíz**, comprueba firma y vigencia y las deja por sustitución
   atómica en el volumen de CRL. Es el único componente con salida a red, y
   solo hacia los puntos de distribución de los prestadores. El validador
   relee el directorio en cada petición, así que no hace falta reiniciarlo.
   Si el refresco falla, las CRL caducan y la revocación pasa a
   `no_comprobada`: la firma queda `indeterminada`, nunca `valida`.

El pod usa una red interna (`podman network create --internal vec-firma`)
compartida con el contenedor de VEC. Ningún puerto se publica al host y el
validador no tiene salida a Internet.

Esquema de órdenes (orientativo; nombres de imagen, versiones y rutas se fijan
en la configuración privada del servidor, fuera de Git):

```sh
# Imagen del validador: se construye desde el repositorio de AutofirmaV2,
# binario sin la etiqueta fyne_gui. No se construye desde VEC ni se copia su
# código aquí.

podman network create --internal vec-firma
podman secret create autofirmav2_rest_token <fichero-privado-con-token>
podman pod create --name vec-firma --network vec-firma

# Anclas: PEM/DER de las raíces admitidas por la política (FNMT, DNIe...),
# preparadas por el operador. CRL: volumen que solo escribe el refresco.
podman volume create vec-firma-crl

podman run -d --pod vec-firma --name vec-firma-validador \
  --read-only --tmpfs /tmp --cap-drop=ALL --security-opt no-new-privileges \
  --secret autofirmav2_rest_token,type=env,target=AUTOFIRMAV2_REST_TOKEN \
  -v <anclas-privadas>:/etc/autofirmav2/anclas:ro,Z \
  -v vec-firma-crl:/var/lib/autofirmav2/crl:ro,z \
  <imagen-validador> -rest-solo-verificacion \
    -verificacion-anclas /etc/autofirmav2/anclas \
    -verificacion-crl /var/lib/autofirmav2/crl \
    -direccion-rest 127.0.0.1:63118

podman run -d --pod vec-firma --name vec-firma-proxy \
  --read-only --cap-drop=ALL --security-opt no-new-privileges \
  -v <config-proxy-privada>:/etc/proxy:ro,Z \
  <imagen-proxy>

# Refresco de CRL: fuera del pod, en una red de salida limitada por
# cortafuegos a los puntos de distribución; escribe en vec-firma-crl.
podman run -d --name vec-firma-crl-refresco \
  --read-only --tmpfs /tmp --cap-drop=ALL --security-opt no-new-privileges \
  --network <red-salida-crl> \
  -v <lista-privada-de-puntos-crl>:/etc/crl:ro,Z \
  -v vec-firma-crl:/var/lib/autofirmav2/crl:z \
  <imagen-refresco-crl>
```

**Sin secretos en Git**: el token, la clave del proxy, su certificado, la CA
de VEC y las rutas de anclas viven en secretos de podman o en la configuración
privada del servidor. Las anclas y las CRL son públicas, pero su selección es
una decisión de despliegue y tampoco se versiona aquí.

## Configuración de VEC

Cuando exista un consumidor compuesto, la composición leerá de la
configuración privada (nunca de Git ni del navegador) los campos de
`validadorautofirma.Configuracion`:

| Campo | Contenido |
|---|---|
| `URL` | Origen `https://host:puerto` del proxy en la red interna, sin ruta. |
| `CAPEM` | CA que emite el certificado del proxy. |
| `NombreServidorTLS` | Solo si difiere del host de la URL. |
| `CertificadoClientePEM` / `ClaveClientePEM` | Identidad mTLS de VEC ante el proxy. |
| `Token` | Solo si VEC habla directamente con el validador sin proxy. |
| `Timeout` | Por defecto 20 s; máximo 60 s. |

Se exige TLS 1.3, sin proxy ambiental, sin redirecciones y al menos una
credencial. Con el proxy recomendado basta el certificado cliente: el token lo
añade el proxy.

## Comportamiento de fallo cerrado

- Indisponibilidad, tiempo agotado, redirección, TLS inválido o `5xx`:
  `indeterminada` / `validador_no_disponible`.
- `401`/`403`: `indeterminada` / `credencial_rechazada`.
- `400`/`413`/`422`: `indeterminada` / `rechazada_por_validador`.
- Respuesta no JSON, mayor de 256 KiB, sin `dictamen`, con otro contrato,
  con estados fuera de catálogo, con huellas de eco distintas o con un
  veredicto negativo incoherente con sus aspectos:
  `indeterminada` / `respuesta_no_interpretable`, sin adoptar ningún aspecto.
- Ningún texto del proveedor llega a VEC, a sus errores ni a sus registros.

Solo `valida` con `verificada`, que además supere `ValidarContra` (vínculo con
el original, huellas, firmante, revocación `vigente` y sello admisible),
permite marcar un documento como firmado.

## Pruebas

`internal/vec/documentos/adapters/validadorautofirma/servidorprueba` imita el
contrato `POST /verify` con dictámenes sintéticos (sin datos reales): válida
sin sello, válida con sello, sello no comprobado o no válido, revocado,
revocación no comprobada, vínculo no acreditado o no aportado, integridad rota
o parcial, sin anclas, varios firmantes, contrato desconocido, sin dictamen,
huellas de eco distintas y veredictos incoherentes, además de los fallos de
transporte. Es solo para pruebas y no se compone en VEC.

```sh
TMPDIR=/dev/shm GOMAXPROCS=4 nice go test -race ./internal/vec/documentos/...
```
