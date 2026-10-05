# Recorrido de propuesta, borradores y firma

Este guion abre en Chrome un expediente sintético de contratación temporal. Comprueba que la propuesta figura en la versión indicada del historial y recupera su recibo desde la consulta de seguimiento. Descarga los seis PDF de formalización y calcula la huella de sus bytes originales. Puede registrar los dos pasos de firma de prueba de un documento con AutoFirma, en dos ejecuciones y con certificados de canal distintos. Consume el preflight y el recibo V2, comprueba la revisión de entrada y descarga el PDF custodiado. GrxFirma interviene después, desde VEC, como verificador; el navegador no lo usa para firmar.

El resultado se detiene en el primer punto sin evidencia. `NO EJECUTADO` significa que ni siquiera se abrió el navegador. `CORTE` identifica el paso alcanzado y lo que falta. Un recibo de firma de prueba no da validez legal ni eficacia administrativa. La consulta técnica V2 recupera referencias de firma y recibo, fecha, custodia, revisión PDF, rangos de bytes, huella de evidencia e historia. No devuelve `material_root_sha256` ni el canon nominal. Por eso la recuperación queda parcial aunque el PDF descargado sea idéntico. El guion no acredita envío a Firmadoc ni un E2E completo.

## Preparación

El lector exige la versión exacta de Playwright fijada en [requirements.txt](requirements.txt): `1.60.0`. Esta versión se ha ensayado con Chrome y emite el evento de cierre de la sesión CDP que usa la guardia. La versión `1.55` del sistema acepta el listener, pero no emite ese evento.

La comprobación se ejecuta al importar el módulo, antes de crear Chrome, contextos o páginas. También contrasta que el paquete y la API importados pertenezcan a la distribución verificada. Si falta el catálogo, la versión difiere o las instalaciones se mezclan, el lector se detiene sin abrir Chrome ni guardar capturas o informes. El SDK debe estar preparado localmente; el lector no descarga ni instala dependencias.

Los lectores privados que importan solo `GuardiaNavegador` deben copiar el módulo completo y su `requirements.txt` al mismo directorio. Extraer únicamente la clase omite esta precondición. Las capturas anteriores obtenidas con Playwright `1.60.0` conservan su procedencia; no acreditan un recorrido con otra versión.

Dirección debe aportar fuera de Git un clon local con H3, H4 y H5 ensayados, el binario correspondiente, datos sintéticos y material mTLS. No se usa la principal ni el clon HITO1 del puerto 55441. El archivo de inventario contiene únicamente estos datos no secretos:

```json
{
  "clon": "local",
  "datos": "sinteticos",
  "hitos": ["H3", "H4", "H5"],
  "binario_sha256": "<64 caracteres hexadecimales de la compilación instalada>",
  "origen": "https://localhost:<puerto del clon>"
}
```

El inventario es una puerta de entrada explícita, no prueba por sí solo que el clon esté instalado. Quien lo prepare debe cotejar los hitos con la instancia aislada antes de ejecutar el guion. El guion calcula la huella del binario externo ejecutable y la compara con el inventario antes de abrir Chrome. El certificado y la clave se pasan por rutas externas; el informe nunca copia esos ficheros.

```sh
python3 scripts/recorridos/propuesta_firma/recorrer.py \
  --entorno /ruta/privada/inventario-clon.json \
  --binario /ruta/privada/vec-server-instalado \
  --origen https://localhost:PUERTO \
  --certificado /ruta/privada/cliente.crt \
  --clave /ruta/privada/cliente.key \
  --expediente-ref expediente:SINTETICO \
  --version-propuesta 7 \
  --captura-escritorio /ruta/privada/recorrido-1440.png \
  --captura-movil /ruta/privada/recorrido-390.png \
  --salida /ruta/privada/recorrido-antes.json
```

El guion descarga los seis PDF en escritorio y los vuelve a descargar a 390 px sin repetir ninguna escritura; exige los mismos bytes y que la página no desborde. Calcula la huella de una captura móvil. Con `--captura-movil` guarda ese PNG fuera de Git, con permiso `0600`, para revisión visual humana. La comprobación automática no sustituye esa revisión.

Con `--captura-escritorio` también guarda la pantalla a 1440 × 900. Si el recorrido se detiene antes de los PDF, ambas opciones conservan la pantalla alcanzada, con su huella y medida de desbordamiento. El informe recoge método, ruta y estado HTTP del portal y las consultas del recorrido, sin parámetros, cabeceras ni respuestas completas. Un fallo al guardar una captura queda indicado; no se sobrescriben archivos existentes.

El directorio padre de las capturas y del informe debe existir, pertenecer al usuario actual y tener permisos `0700`. El guion rechaza cualquier ruta dentro de un repositorio o worktree Git, las rutas con `..` y los enlaces simbólicos. Comprueba estas condiciones antes de abrir Chrome y otra vez al crear cada archivo.

Sin `--firmar` se comprueban la propuesta, los seis borradores en ambos anchos y el catálogo de firma. El resultado indica `autoridad_pendiente`; no abre AutoFirma. Si el montaje no ofrece el original o el preflight nominal, el guion informa esa dependencia antes de pulsar la firma.

Para el primer paso, añadir a la orden anterior `--firmar --paso 1 --documento informe_definitivo` y usar una salida nueva, por ejemplo `/ruta/privada/primera.json`. Para el segundo, repetir los argumentos del clon y expediente con el certificado y clave de la otra identidad, `--firmar --paso 2 --continuar /ruta/privada/primera.json --salida /ruta/privada/segunda.json`. El certificado de canal distinto no acredita por sí solo otra persona: esa correspondencia y su competencia las decide la autoridad central. Se admiten los seis documentos mediante `--documento`; cada continuación corresponde al mismo documento.

La interfaz pulsa «Comprobar» y después la vía «certificado_vec» que habilita el servidor. Una persona completa o cancela AutoFirma en el puesto. El guion espera hasta tres minutos un único `POST /api/vec/contratacion-temporal/firmas-documento/registro-vec`; exige el esquema `vec.contratacion-temporal.registro-firma-vec.v2`, custodia, revisión PDF, huellas y verificación técnica positiva. Antes del segundo paso descarga la primera revisión custodiada y exige que el preflight la señale como entrada. Cada registro conserva `firma_eficaz=false`.

Después de cada firma se exige `POST /api/vec/contratacion-temporal/firmas-documento/consultas-v2`, con la misma clave real conservada en el intento privado, sin generar otra operación ni enviar organización, persona candidata o perfil. Antes del segundo paso se coteja también la primera revisión con su consulta conservada. Una ruta ausente, denegada o sin las revisiones esperadas deja `consulta_v2_pendiente`; la consulta V1 del DOM no sustituye esa prueba. El resultado `PRIMERA_FIRMA_CONFIRMADA` permite continuar con la otra identidad. `COMPLETO` indica dos recibos confirmados y PDF custodiado descargado; sigue pendiente recuperar la trazabilidad V2 tras reinicio. Ambos informes declaran `e2e=false`. Los recibos de los informes anteriores deben cumplir el esquema cerrado V2; cualquier campo extra impide continuar. La consola proyecta solo los campos técnicos permitidos. La salida privada se reserva antes de abrir Chrome. Antes de pulsar AutoFirma queda marcada la operación incierta; la clave idempotente se conserva solo en ese archivo y no se imprime. Un corte posterior al POST bloquea otra escritura con ese informe. No borrar ni editar el estado para repetir el registro; requiere reconciliación por el responsable del clon.

Después del reinicio externo de aplicación y PostgreSQL del clon, crear un acta privada con este contenido:

```json
{
  "expediente_ref": "expediente:SINTETICO",
  "aplicacion_reiniciada": true,
  "postgresql_reiniciado": true,
  "instante_utc": "2026-10-03T15:00:00Z"
}
```

Usar los argumentos habituales y `--comparar /ruta/privada/segunda.json --reinicio /ruta/privada/reinicio.json --salida /ruta/privada/recuperada.json`, sin `--firmar` ni `--continuar`. El acta es una declaración externa; el guion no observa el reinicio. Este modo solo consulta y descarga: coteja los dos recibos, fechas, orden, custodia y el PDF final, sin POST de firma ni original. La consulta técnica V2 debe conservar exactamente referencias, rangos de bytes, entrada, revisión, huella de evidencia e historia respecto al informe anterior. Sigue devolviendo `RECUPERACION_PARCIAL`, código 2 y `e2e=false`: el endpoint declara ausentes `material_root_sha256` y `canon_nominal`. No vuelve a ejecutar la verificación criptográfica de GrxFirma. Los informes anteriores sin intento privado o consulta V2 confirmada quedan pendientes; no se completan con claves o metadatos inventados.

El guion exige HTTPS local, Chrome del sistema, Playwright de Python, mTLS sintético y certificado confiable por el sistema. Bloquea solicitudes HTTP a otros orígenes y las redirecciones. Solo al usar `--firmar` admite el WebSocket de AutoFirma en `wss://127.0.0.1:63117`. Informa estados HTTP, tamaño y SHA-256 de cada PDF; no guarda los PDF ni el contenido firmado en el informe. No incluye credenciales, bytes de documentos ni nombres de personas. Conserva recibos V2 técnicos y la clave de la operación solo en el informe privado; la salida de consola omite el intento con su clave.

## Recuperación nominal preparada

Cuando esté montada la recuperación autorizada de 48 campos, añada
`--recuperacion-nominal` a las tres ejecuciones: primera firma, continuación
con la segunda identidad y comparación tras reiniciar. El guion consulta
`POST /api/vec/contratacion-temporal/firmas-documento/recuperaciones-v2`
con la clave real conservada. Si la ruta falta, deniega o falla, se detiene;
la consulta técnica anterior no sustituye esta lectura.

El servidor valida la competencia histórica con el modelo común Go. El
guion coteja la huella de los bytes exactos del canon y la huella del material
contra cada recibo. Guarda únicamente referencias y huellas en
`recuperacion_nominal_v2`, dentro del informe privado. No conserva el canon
ni lo imprime. Antes de firmar el segundo paso vuelve a comprobar el primero;
tras reiniciar compara las dos evidencias con las guardadas previamente.
Un informe antiguo sin esa evidencia permanece pendiente.

Una comparación favorable devuelve `RECUPERACION_NOMINAL_CONFIRMADA` y código
0, después de los controles finales del navegador. Conserva `e2e=false` y
`verificacion_criptografica_repetida=false`: este modo no repite GrxFirma ni
acredita auditoría propia de la descarga. Las pruebas sintéticas del guion
preparan esa comprobación; no prueban montaje nominal, firmas ni reinicio real.
Sin la opción nueva se conserva el resultado parcial anterior.

## Prueba focal y dependencias

```sh
python3 -m unittest discover -s scripts/recorridos/propuesta_firma -p 'test_*.py'
python3 scripts/recorridos/propuesta_firma/recorrer.py
```

Sin inventario y binario acreditados, la segunda orden produce `NO EJECUTADO`, `corte=precondiciones`, sin conexión de red. Antes del navegador se necesitan montaje R5, sesión y contexto registrados, publicaciones nominales autorizadas, canon central, custodia y TLS verificado del clon y del verificador. Las pruebas de contratos no acreditan esas autoridades ni el recorrido.
