# Lecturas de Bolsa y Contratación temporal

Este runner prepara una pasada de lectura sobre un servidor sintético ya
arrancado. Abre Chrome del sistema a 1440 y 390 px y comprueba:

- La persona candidata consulta Mi bolsa y su historial propio.
- RRHH abre una bolsa existente y consulta sus candidatos.
- RRHH abre el cuadro de Contratación y el detalle de un expediente existente.

No arranca servicios, instala SQL ni prepara datos. Las únicas peticiones POST
admitidas son las consultas del cuadro y detalle CT. No pulsa acciones de
negocio ni ofrece un modo de escritura. Las lecturas pueden registrar auditoría
de acceso en VEC.

## Entradas privadas

Dirección entrega las identidades sintéticas con concesiones nominales y
verifica qué binario, esquema y perfiles sirven cada proceso. RRHH y candidato
usan **orígenes distintos**, ambos HTTPS en loopback numérico. Ejecútelo en el
equipo que sirve esos procesos, después de su instalación autorizada. No admite
cidonia, nombres DNS ni un destino remoto. Chrome debe confiar en la CA del
servidor; se mantiene la validación TLS.

El JSON, certificados y claves permanecen fuera de cualquier repositorio o
worktree. No se admiten enlaces simbólicos ni archivos con enlaces duros. El
JSON y las claves deben ser propios, privados (`0600`); la carpeta padre de la
evidencia debe ser propia y privada (`0700`). Ejemplo de forma, con marcadores:

```json
{
  "version": 1,
  "sintetico": true,
  "entorno_controlado": true,
  "origenes": {
    "interno": "https://127.0.0.1:PUERTO_INTERNO",
    "externo": "https://127.0.0.1:PUERTO_EXTERNO"
  },
  "commit_servido": "HASH_COMPLETO_VERIFICADO_POR_DIRECCION",
  "idioma": "es",
  "bolsa_ref": "bolsa:sintetica",
  "expediente_ref": "expediente:sintetico",
  "identidades": {
    "rrhh": { "certificado": "/RUTA_PRIVADA/rrhh.crt", "clave": "/RUTA_PRIVADA/rrhh.key" },
    "candidato": { "certificado": "/RUTA_PRIVADA/candidato.crt", "clave": "/RUTA_PRIVADA/candidato.key" }
  }
}
```

La bolsa debe estar presente en la lista RRHH y en las participaciones de la
persona candidata. El expediente debe verse en la primera página del cuadro:
el guion no crea otro ni recorre páginas para encontrarlo. Las claves opacas
admiten letras, números, dos puntos, punto, guion y guion bajo. Las dos
identidades requieren certificados y claves de contenido distinto.

Use Node 20 o posterior y una instalación local existente de Playwright 1.48 o
posterior, con `routeWebSocket` e `indexedDB.databases` disponibles. El ejecutable
de navegador está fijado a `/usr/bin/google-chrome`. `VEC_PLAYWRIGHT_MODULE`
señala el `index.mjs` absoluto de Playwright, fuera de Git. Puede utilizarse el
paquete Node que acompaña a Playwright Python, sin instalar otro paquete ni
descargar Chromium. El preflight verifica entradas y disponibilidad del módulo;
no valida los certificados con el servidor ni demuestra una instalación.

## Ejecución

```sh
export VEC_PLAYWRIGHT_MODULE=/RUTA_INSTALACION_PLAYWRIGHT/index.mjs
node scripts/recorridos-f/recorrer.mjs \
  --config /RUTA_PRIVADA/lecturas.json --modo preflight
node scripts/recorridos-f/recorrer.mjs \
  --config /RUTA_PRIVADA/lecturas.json --modo lectura \
  --salida /RUTA_PRIVADA/lecturas-antes
```

Para inglés cambie `idioma` a `en` y use otra carpeta nueva. El guion selecciona
`?lang=` y comprueba el atributo `lang`; no modifica preferencias persistidas.

Tras un reinicio autorizado de **los mismos procesos y PostgreSQL**, Dirección
puede volver a consultar y comparar el resumen y la historia CT:

```sh
node scripts/recorridos-f/recorrer.mjs \
  --config /RUTA_PRIVADA/lecturas.json --modo lectura \
  --comparar /RUTA_PRIVADA/lecturas-antes/resultado.json \
  --salida /RUTA_PRIVADA/lecturas-despues
```

El runner no reinicia ni observa el reinicio. La comparación exige el mismo
escenario, idioma y commit declarado y la misma huella del resumen e hitos CT.
Las lecturas de Bolsa se repiten; **no se comparan sus recibos ni su contenido**.
Dirección debe acreditar el reinicio y contrastar tablas, historia y outbox
antes de afirmar recuperación durable sin duplicados.

`resultado.json` conserva pasos, estados HTTP esperados, contadores y huella CT;
omite URL, credenciales, nombres, cuerpos HTTP, referencias y capturas. En móvil
comprueba el menú con Tab, Enter y Escape. En ambos anchos exige cero errores JS,
cookies, almacenamiento web, IndexedDB, Cache Storage y desbordamiento global.
Bloquea peticiones a otros orígenes, WebSocket, redirecciones y Set-Cookie. La
carpeta de salida debe ser nueva. No active trazas HAR ni perfiles persistentes.

Código `0`: preflight preparado o lecturas comprobadas, según el modo elegido.
Código `2`: no ejecutado por entradas o dependencias inválidas. Código `1`: el
recorrido comenzó y se cortó; el resultado privado identifica el paso. Los
campos `servidor_instalado_verificado` y `reinicio_verificado` permanecen `false`:
el runner no puede acreditar por sí solo esas operaciones de Dirección.

## Comprobación del runner

```sh
node --test scripts/recorridos-f/recorrer.test.mjs
node scripts/recorridos-f/smoke-chrome.mjs
semgrep --config scripts/recorridos-f/semgrep-local.yml \
  --metrics off --disable-version-check --no-git-ignore scripts/recorridos-f
```

El smoke usa Chrome y dos servidores HTTP efímeros propios en loopback. Demuestra
que el destino de un 302 recibe cero peticiones y que un POST de negocio no llega
al servidor. Ejecútelo con aislamiento de red y límites de recursos; no requiere
VEC ni PostgreSQL. Sus respuestas son fixtures, no evidencia del servidor instalado.

Las rutas y selectores proceden de `scripts/recorridos/bolsa_ofertas`,
`scripts/recorridos/analisis_informe` y del contrato de Mi bolsa. Se conservan los
guiones existentes de altas, llamamiento y recuperación, los recorridos H6 de M
y el baremador de B. Este corte no acredita las ocho fases, firma, entrega de
correo, perfiles completos ni conformidad global de accesibilidad.

Verificación del productor: cinco pruebas Node y smoke Chrome aislado verdes.
Las seis lecturas contra VEC están **pendientes**, porque no se ha entregado un
servidor con sus dos orígenes y perfiles sintéticos listo para este runner.
