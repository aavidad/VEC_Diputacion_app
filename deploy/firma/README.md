# Validador de firma como servicio aparte

Decisión del operador (25/09/2026): VEC firma en el equipo de la persona por el
protocolo AutoFirma y **verifica en servidor con el validador de AutofirmaV2**,
desplegado como servicio independiente. VEC no importa código de AutofirmaV2:
lo consume por HTTP mediante el adaptador
`internal/vec/documentos/adapters/validadorautofirma`, que implementa los
puertos `VerificadorFirma` y `VerificadorFirmaMotivado` de Documentos.

Este documento describe el despliegue previsto. **No está desplegado ni
probado en Cidonia**, y ningún flujo de negocio de VEC lo usa todavía.

## Qué ofrece hoy el validador y qué no

API usada: `POST /verify` del servidor REST de AutofirmaV2
(`autofirma -rest`), con cuerpo JSON:

| Campo | Uso por VEC |
|---|---|
| `name` | Fijo `documento`; nunca el nombre real del fichero. |
| `content_base64` | Artefacto firmado. |
| `original_content_base64` | Original custodiado, solo en firmas separadas (CAdES/XAdES). En PAdES no se envía: AutofirmaV2 lo ignora. |

La respuesta `200` trae `ok`, `valid` y `result` con los aspectos `integrity`,
`certificate` y `trust` (`valid`/`invalid`/`warning`/`unknown`),
`signerSummaries` y detalles libres. El adaptador solo conserva los estados,
el marcador `modo=detached` y la huella SHA-256 del certificado firmante;
descarta asunto, emisor, textos y avisos del proveedor.

Limitaciones que impiden hoy acreditar una firma (el adaptador devuelve como
máximo `indeterminada`, nunca `valida`):

- **Sello de tiempo**: la verificación de AutofirmaV2 no valida ni informa del
  sello de tiempo.
- **Revocación**: se consulta OCSP/CRL solo si la firma incluye la cadena, y no
  se expone un estado explícito. `certificate: valid` no prueba revocación
  vigente (cuando la consulta no concluye, el resultado puede seguir marcado
  como válido).
- **Vínculo con el original en PAdES**: el original aportado se ignora; VEC no
  puede acreditar que el PDF firmado corresponde al original custodiado.
- **Anclas de confianza**: se usa el almacén del sistema de la imagen. Debe
  contener las raíces admitidas por la política (FNMT, DNIe, etc.).
- **Superficie**: el servidor REST expone además firma, protección, importación
  de certificados y gestión de servicio. Por eso se publica solo detrás de un
  proxy que admite únicamente `POST /verify`.

Los defectos concluyentes sí se traducen a `no_valida`: integridad,
certificado (por ejemplo, revocado) o confianza inválidos.

## Topología recomendada (podman)

Un pod `vec-firma` con dos contenedores que comparten red de loopback:

1. **validador**: AutofirmaV2 sin interfaz gráfica, escuchando en
   `127.0.0.1:63118` (loopback: no hace falta `-rest-publico`), con token
   Bearer. Su certificado TLS lo genera AutofirmaV2 con su CA local y solo
   cubre `localhost`/`127.0.0.1`.
2. **proxy**: termina TLS 1.3 con **autenticación mutua** hacia VEC, admite solo
   `POST /verify`, limita el cuerpo (48 MiB: dos contenidos de 16 MiB en base64) y el tiempo, y reenvía a
   `https://127.0.0.1:63118` confiando en la CA local del validador y añadiendo
   el token Bearer.

El pod se une a una red interna (`podman network create --internal
vec-firma`) compartida con el contenedor de VEC. Ningún puerto se publica al
host. El validador necesita **salida** hacia OCSP/CRL de los prestadores: con
`--internal` no la tiene, así que hace falta una segunda red de salida limitada
por cortafuegos a esos destinos, o un proxy de salida. Es una decisión
pendiente.

Esquema de órdenes (orientativo; los nombres de imagen, versiones y rutas se
fijan en la configuración privada del servidor, fuera de Git):

```sh
# Imagen del validador: se construye desde el repositorio de AutofirmaV2,
# binario sin la etiqueta fyne_gui (modo sin interfaz). No se construye
# desde VEC ni se copia su código aquí.

podman network create --internal vec-firma
podman secret create autofirmav2_rest_token <fichero-privado-con-token>
podman pod create --name vec-firma --network vec-firma

podman volume create vec-firma-config   # CA local y certificado TLS del validador
podman run -d --pod vec-firma --name vec-firma-validador \
  --read-only --tmpfs /tmp --cap-drop=ALL --security-opt no-new-privileges \
  --secret autofirmav2_rest_token,type=env,target=AUTOFIRMAV2_REST_TOKEN \
  -v vec-firma-config:/home/validador/.config/autofirma-v2:Z \
  <imagen-validador> -rest -rest-addr 127.0.0.1:63118

podman run -d --pod vec-firma --name vec-firma-proxy \
  --read-only --cap-drop=ALL --security-opt no-new-privileges \
  -v <config-proxy-privada>:/etc/proxy:ro,Z \
  <imagen-proxy>
```

El volumen de configuración conserva la CA local para que no cambie en cada
arranque. La CA es pública; su clave y el token no salen del servidor.

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
- Respuesta no JSON, mayor de 256 KiB o con estados desconocidos:
  `indeterminada` / `respuesta_no_interpretable`.
- Ningún texto del proveedor llega a VEC, a sus errores ni a sus registros.

Solo `valida` con `verificada`, que además supere `ValidarContra` (vínculo con
el original, huellas, certificado, sello `valido` y revocación `vigente`),
permitiría marcar un documento como firmado.

## Pruebas

`internal/vec/documentos/adapters/validadorautofirma/servidorprueba` imita el
contrato `POST /verify` con respuestas sintéticas (sin datos reales). Es solo
para pruebas y no se compone en VEC.

```sh
TMPDIR=/dev/shm GOMAXPROCS=4 nice go test -race ./internal/vec/documentos/...
```
