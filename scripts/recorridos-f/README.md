# Lecturas de Bolsa y Contratación temporal

Este runner prepara una pasada de lectura sobre un servidor sintético ya
arrancado. Abre Chrome del sistema a 1440 y 390 px y comprueba:

- La persona candidata consulta Mi bolsa y su historial propio.
- RRHH abre una bolsa existente y consulta sus candidatos.
- RRHH abre el cuadro de Contratación y el detalle de un expediente existente.

No arranca servicios, instala SQL ni prepara datos. Las únicas peticiones POST
admitidas son cuatro consultas CT: cuadro, detalle, catálogo de borradores
disponibles y estado de firmas. Las dos últimas se montan automáticamente al
abrir la ficha; se admiten solo sus cuerpos exactos. No pulsa acciones de
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

Para comprobar accesibilidad en esas mismas lecturas:

```sh
node scripts/recorridos-f/recorrer.mjs \
  --config /RUTA_PRIVADA/lecturas.json --modo accesibilidad \
  --salida /RUTA_PRIVADA/lecturas-accesibilidad
```

El modo `accesibilidad` observa Mi bolsa, su historial, la lista RRHH de bolsas,
los candidatos de la bolsa, el cuadro CT y el detalle CT. Repite las seis
superficies a 1440 y 390 px, y a 1440 px con **zoom nativo de Chrome al 200 %**.
Use una configuración y una salida nueva por idioma, español e inglés.
Las dos aperturas de lectura usan Tab y Enter. En los demás controles sólo
recorre Tab y Shift+Tab; no activa acciones ni cambia valores.

Comprueba el nombre accesible calculado por Chrome y la etiqueta visible de
cada campo habilitado, incluida la opacidad de sus antecesores. Recorre los controles habilitados de cada superficie
con teclado y exige un cambio visible del indicador al recibir el foco.
Comprueba el centro y los cuatro puntos medios de los bordes del control para
detectar obstáculos sin tomar las esquinas de un botón redondeado por zonas
tapadas. También exige salir y volver con Tab y Shift+Tab, incluido el último
control. Exige que no haya desbordamiento global.
La búsqueda admite 256 iteraciones por superficie, además de las pruebas de
ida y vuelta; superarlas corta la comprobación.
En un grupo de radios comprueba la entrada con Tab, sin cambiar la selección.
Una superficie sin controles habilitados registra cero controles de teclado.

Chrome aplica el zoom desde su página interna de ajustes. Usa un perfil temporal
privado para esos ajustes y contextos separados, sin persistencia, para las
identidades mTLS. Cierra Chrome y elimina el perfil al terminar o al fallar.
La comprobación verifica ancho CSS, densidad, escala visual y ausencia de zoom
CSS: 1440 px deben dar 720 px CSS, DPR 2 y escala visual 1 al 200 %.
No cambia el perfil habitual de Chrome ni añade destinos a las lecturas.

Cada paso añade `accesibilidad`: lectura, zoom, métricas y contadores, sin textos
de etiquetas, nombres accesibles ni fragmentos de página. Los fallos cortan el
recorrido; consulte `etiquetas_ausentes`, `nombres_ausentes`,
`controles_fuera_de_tab`, `teclado_no_alcanzados`, `trampas_teclado`,
`foco_invisible` y `foco_tapado`.
El estado final de este modo es `LECTURAS_Y_ACCESIBILIDAD_COMPROBADAS`.
El modo `lectura` conserva su resultado v1 y su comparación; `accesibilidad`
no acepta `--comparar` ni sustituye la evidencia del reinicio.

Estas comprobaciones son parciales. El indicador de foco se observa mediante
estilos y puntos visibles; no mide su contraste ni todo su contorno. No acredita
orden lógico, interacción con flechas, lector de pantalla, PDF accesible ni
conformidad global WCAG. La revisión humana sigue siendo necesaria.

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

Los sobres se comprueban con los validadores del frontend de este mismo árbol,
sin copiar sus contratos. Un HTTP 200 incompatible corta el paso. El historial
debe terminar de cargar sin botón de reintento; CT debe mostrar la cabecera de
la ficha, sin el aviso de error global. Las respuestas 200 de las consultas
automáticas de borradores y firmas también pasan por sus contratos originales.
Su estado HTTP queda en el informe: un 404 o una denegación de esos paneles no
acredita que estén disponibles ni invalida la lectura del detalle principal.
No se descargan borradores ni se registran firmas.

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
node --test scripts/recorridos-f/recorrer.test.mjs scripts/recorridos-f/a11y.test.mjs
node scripts/recorridos-f/smoke-chrome.mjs
VEC_F_TEST_SCRATCH=/RUTA_PRIVADA/TEMPORALES node scripts/recorridos-f/smoke-a11y-chrome.mjs
# Sólo las regresiones de la revisión de usabilidad:
VEC_F_TEST_SCRATCH=/RUTA_PRIVADA/TEMPORALES node scripts/recorridos-f/smoke-a11y-chrome.mjs --regresiones-ux
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

El primer corte (`9f33eb4f7`) verificó once pruebas Node y once casos de
accesibilidad en Chrome del sistema aislado, con fixtures propios. Los casos
cubren español e inglés, 1440 y 390 px y zoom nativo al 200 %; detectan etiqueta
sólo accesible, nombre ausente, foco invisible, control tapado y exclusión de Tab.
No hubo escrituras al servidor del fixture y se eliminó el perfil temporal.

El correctivo de la revisión repite las once pruebas Node y ejecuta sólo seis
regresiones nuevas en Chrome: botón redondeado visible, etiqueta con opacidad
cero, antecesor con opacidad cero, contorno permanente y trampa en el último
control en ambas direcciones o sólo hacia delante. Los seis casos pasan, sin
escrituras y con el perfil eliminado. Este correctivo no repite los once casos
Chrome anteriores ni el zoom nativo al 200 %.

Las regresiones conservadas cubren los HTTP 200 incompatibles y el resultado terminal
saneado cuando falla Chrome o el contexto mTLS. El smoke de transporte se
conserva del primer corte; no se repitió para esta extensión de accesibilidad.
Las seis lecturas contra VEC están **pendientes**, porque no se ha entregado un
servidor con sus dos orígenes y perfiles sintéticos listo para este runner.
