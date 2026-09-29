# Recorrido de propuesta, borradores y firma

Este guion abre en Chrome un expediente sintético de contratación temporal. Comprueba que la propuesta figura en la versión indicada del historial y recupera su recibo desde la consulta de seguimiento. Descarga los seis PDF de formalización y calcula la huella de sus bytes originales. Puede abrir AutoFirma del puesto para una firma de prueba del informe definitivo y contrastar el recibo con la consulta del registro. GrxFirma interviene después, desde VEC, como verificador; el navegador no lo usa para firmar.

El resultado se detiene en el primer punto sin evidencia. `NO EJECUTADO` significa que ni siquiera se abrió el navegador. `CORTE` identifica el paso alcanzado y lo que falta. Un recibo de firma de prueba no da validez legal ni eficacia administrativa. CT118 conserva referencias y huellas, pero este recorrido no encuentra todavía un recibo de custodia verificable de los bytes originales y firmados. Tampoco acredita envío a Firmadoc.

## Preparación

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

El inventario es una puerta de entrada explícita, no prueba por sí solo que el clon esté instalado. Quien lo prepare debe cotejar sus hitos y la huella con la instancia aislada antes de ejecutar el guion. El certificado y la clave se pasan por rutas externas; el informe nunca copia esos ficheros.

```sh
python3 scripts/recorridos/propuesta_firma/recorrer.py \
  --entorno /ruta/privada/inventario-clon.json \
  --origen https://localhost:PUERTO \
  --certificado /ruta/privada/cliente.crt \
  --clave /ruta/privada/cliente.key \
  --expediente-ref expediente:SINTETICO \
  --version-propuesta 7 \
  --salida /ruta/privada/recorrido-antes.json
```

Sin `--firmar` solo se comprueban la propuesta, los seis borradores y el catálogo de firma. El corte esperado es `firma_admitida`. Con `--firmar`, Chrome abre con ventana y pulsa una sola vez «Firmar» para el informe definitivo. Una persona debe completar o cancelar AutoFirma en el puesto. El guion espera hasta tres minutos el único `POST` de registro; no reintenta una firma ni sustituye el firmante. Si VEC devuelve un recibo con `verificacion.estado=valida`, `motivo=verificada` y `firma_eficaz=false`, consulta de nuevo el mismo recibo. El corte final sigue siendo `custodia` mientras no exista evidencia de custodia.

Tras reiniciar externamente la aplicación y PostgreSQL **del clon**, usar `--comparar /ruta/privada/recorrido-antes.json` con las demás opciones y otra salida. Ese modo solo lee: vuelve a descargar los seis PDF, compara sus huellas y la propuesta y, si el informe anterior conserva una firma, recupera su recibo por consulta sin repetir el `POST` de firma. No se puede combinar `--comparar` con `--firmar`.

El guion exige HTTPS local, Chrome del sistema, Playwright de Python, mTLS sintético y certificado confiable por el sistema. Bloquea solicitudes HTTP a otros orígenes y las redirecciones. Solo al usar `--firmar` admite el WebSocket de AutoFirma en `wss://127.0.0.1:63117`. Informa estados HTTP, tamaño y SHA-256 de cada PDF; no guarda los PDF ni el contenido firmado en el informe. No incluye credenciales, documentos, nombres de personas ni respuestas completas de API.

## Prueba focal y estado inicial

```sh
python3 -m unittest discover -s scripts/recorridos/propuesta_firma -p 'test_*.py'
python3 scripts/recorridos/propuesta_firma/recorrer.py
```

En el corte actual no hay clon H3-H5 y binario acreditados para este guion. La segunda orden debe producir `NO EJECUTADO`, `corte=precondiciones`, sin conexión de red. Preparar y comprobar ese clon es la primera dependencia. Después se necesita una firma admitida de prueba en AutoFirma; por último, el contrato de custodia y su recibo verificable.
