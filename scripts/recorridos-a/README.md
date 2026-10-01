# Recorrido B2 y firma E3

Este guion abre en Chrome del sistema un expediente sintético indicado por quien prepara el clon. Recorre los pasos de firma de prueba disponibles en E3, conserva cada recibo verificado y registra la incorporación B2 mediante el formulario de RRHH. Una segunda ejecución comprueba las mismas firmas y el recibo B2 después de que el operador reinicie externamente la aplicación y PostgreSQL del clon. No reinicia servicios ni crea expedientes.

La firma E3 indica `firma_eficaz=false`: es una firma de prueba verificada por GrxFirma, sin eficacia administrativa. La confirmación B2 indica `firma_oficial=false` y `eficacia_administrativa=false`. El guion no presenta ninguna de ellas como firma legal, envío a Firmadoc ni transmisión a GINPIX.

## Entrada privada

Crear fuera de Git un directorio propio con permisos `0700`, el inventario JSON con permisos `0600` y los certificados sintéticos. El inventario tiene esta forma; cada referencia y fecha se elige para el expediente que se va a recorrer:

```json
{
  "version": 1,
  "datos": "sinteticos",
  "entorno_controlado": true,
  "commit_servido": "<hash de 40 caracteres de la instalación>",
  "origen": "https://127.0.0.1:PUERTO",
  "expediente_ref": "expediente:ct:ejemplo",
  "identidad": {
    "certificado": "/ruta/privada/rrhh.crt",
    "clave": "/ruta/privada/rrhh.key"
  },
  "capturas": "/ruta/privada/capturas",
  "validacion": {
    "b2_instalado": true,
    "e3_instalado": true,
    "autofirma_preparada": true,
    "grxfirma_preparada": true
  },
  "auxiliares": ["wss://127.0.0.1:63117"],
  "seleccion": {
    "vacante": "plaza:ejemplo",
    "regimen": "regimen:ejemplo",
    "modalidad": "modalidad:ejemplo",
    "clase_ocupacion": "titular",
    "motivo": "incorporacion",
    "documento": "documento:ejemplo",
    "desde": "2026-10-02",
    "hasta": ""
  }
}
```

Los cuatro indicadores de `validacion` son declaraciones del responsable del clon. Debe comprobar realmente instalación, permisos, AutoFirma y GrxFirma antes de ponerlos a `true`; el guion también exige sus respuestas en el navegador. `seleccion` usa las referencias que devuelve la consulta B2 de ese expediente, sin posiciones ni un expediente fijo. Una selección ausente o ambigua corta antes de preparar el plan. El origen principal debe ser HTTPS local. Solo se permiten orígenes auxiliares locales declarados de forma exacta; no se siguen redirecciones ni se aceptan cookies.

`VEC_PLAYWRIGHT_MODULE` señala el archivo principal de una instalación **ya disponible** de Playwright para Node. El guion usa `/usr/bin/google-chrome` y no descarga navegador ni paquetes. AutoFirma requiere intervención de la persona autorizada en cada paso; su espera máxima es tres minutos. GrxFirma actúa desde el servidor: una declaración en el JSON no sustituye su verificación positiva en el recibo.

```sh
VEC_PLAYWRIGHT_MODULE=/ruta/instalada/playwright/index.mjs \
  node scripts/recorridos-a/recorrer.mjs --config /ruta/privada/b2-e3.json \
  --modo preparado --estado /ruta/privada/b2-e3-estado.json
```

`PREPARADO` solo valida la entrada. No abre Chrome ni acredita recorrido. Para ejecutar, usar **el mismo archivo de estado** y este orden:

```sh
VEC_PLAYWRIGHT_MODULE=/ruta/instalada/playwright/index.mjs \
  node scripts/recorridos-a/recorrer.mjs --config /ruta/privada/b2-e3.json \
  --modo firmar --estado /ruta/privada/b2-e3-estado.json

VEC_PLAYWRIGHT_MODULE=/ruta/instalada/playwright/index.mjs \
  node scripts/recorridos-a/recorrer.mjs --config /ruta/privada/b2-e3.json \
  --modo incorporar --estado /ruta/privada/b2-e3-estado.json
```

`firmar` abre Chrome con ventana y procesa todos los documentos y pasos pendientes que devuelve E3. Exige que todos queden firmados y compara cada recibo nuevo con la consulta. `incorporar` recorre revisión, plan y confirmación B2 en el formulario, y conserva intención, claves, plan, recibo y fecha en el estado privado. Si hay un POST incierto, el archivo marca `pendiente` y bloquea otra escritura. `--modo reconciliar` hace una consulta de solo lectura: conserva el recibo si el efecto ya aparece y mantiene el bloqueo si la consulta no lo demuestra. No borrar ni editar el estado para forzar otro POST.

```sh
VEC_PLAYWRIGHT_MODULE=/ruta/instalada/playwright/index.mjs \
  node scripts/recorridos-a/recorrer.mjs --config /ruta/privada/b2-e3.json \
  --modo reconciliar --estado /ruta/privada/b2-e3-estado.json
```

Cuando E3 solo se pudo recuperar por consulta, el recibo y la fecha quedan acreditados por esa lectura; la huella del PDF firmado del POST original queda sin recuperar.

Tras reiniciar **externamente solo el clon autorizado**, registrar fuera de Git un JSON privado con `expediente_ref`, `aplicacion_reiniciada: true`, `postgresql_reiniciado: true` e `instante_utc`. Ese documento declara el reinicio; el guion no lo observa por sí mismo. La recuperación no emite escrituras:

```sh
VEC_PLAYWRIGHT_MODULE=/ruta/instalada/playwright/index.mjs \
  node scripts/recorridos-a/recorrer.mjs --config /ruta/privada/b2-e3.json \
  --modo recuperar --estado /ruta/privada/b2-e3-estado.json \
  --reinicio /ruta/privada/reinicio-clon.json
```

El resultado `LECTURAS_RECUPERADAS` coteja recibos, fecha y estado de todos los pasos E3 registrados, además del recibo B2 completo. La consulta E3 no vuelve a entregar la huella del PDF firmado; esa huella queda solo en el recibo original. El archivo privado conserva lo necesario para cotejar sin guardar bytes de documentos ni certificados. El guion comprueba errores JavaScript, cookies, almacenamiento web, red y desbordamiento a 1440 y 390 px. Una inspección humana de la pantalla sigue pendiente antes de afirmar usabilidad.
Si se indica `capturas`, el guion guarda dos PNG privados por modo, sin sobrescribir nombres; el informe solo conserva sus huellas y tamaños.

Prueba focal de la lógica local:

```sh
node --test scripts/recorridos-a/recorrer.test.mjs
```
