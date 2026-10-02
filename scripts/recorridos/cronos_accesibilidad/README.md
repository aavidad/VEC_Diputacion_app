# Comprobación local de accesibilidad de Cronos

Este guion monta cuatro vistas del checkout en Chrome: fichaje remoto, saldo,
calendario con incidencias y bandeja de permisos. Usa los catálogos i18n reales
y datos sintéticos de `casos.json`. Hace 24 visitas: español e inglés a 1440 y
390 px, y 1440 px con zoom nativo al 200 %.
El host mínimo usa `main#espacio-trabajo`: el CSS del portal le asigna la altura
y el desplazamiento interno en escritorio, donde el documento no se desplaza.
El título permanece dentro de ese marco. Esto no monta el shell completo.

```sh
node scripts/recorridos/cronos_accesibilidad/recorrer.mjs --plan
```

El plan solo lee el catálogo público de casos. No carga Playwright, abre Chrome,
lee configuración privada ni contacta con un servidor.

La ejecución necesita Playwright ya instalado y `/usr/bin/google-chrome`.
`VEC_PLAYWRIGHT_MODULE` apunta a su `index.mjs`. `--salida` debe ser una carpeta
nueva, fuera del checkout, dentro de una carpeta propia con permisos 0700.

```sh
node --test scripts/recorridos/cronos_accesibilidad/recorrer.test.mjs
VEC_PLAYWRIGHT_MODULE=/RUTA_PLAYWRIGHT_EXISTENTE/index.mjs \
  node scripts/recorridos/cronos_accesibilidad/recorrer.test.mjs --ensayar-chrome
VEC_PLAYWRIGHT_MODULE=/RUTA_PLAYWRIGHT_EXISTENTE/index.mjs \
  node scripts/recorridos/cronos_accesibilidad/recorrer.test.mjs --ensayar-host
VEC_PLAYWRIGHT_MODULE=/RUTA_PLAYWRIGHT_EXISTENTE/index.mjs \
  node scripts/recorridos/cronos_accesibilidad/recorrer.mjs \
  --salida /CARPETA_PRIVADA/SALIDA_NUEVA
```

Ejecute ambos comandos dentro de un sandbox: checkout, Chrome y Playwright de
solo lectura; entorno vacío con PATH, HOME, TMPDIR, XDG_CACHE_HOME y
VEC_PLAYWRIGHT_MODULE explícitos; red aislada; escritura solo en scratch.
El ensayo de esta entrega usa bwrap con `--unshare-all --clearenv`, scratch
tmpfs de 512 MiB, systemd con MemoryMax=4G, TasksMax=512, CPUQuota=800% y
RuntimeMaxSec=600, y límites de 180 segundos CPU, 16 MiB por archivo y 1024
descriptores. No hace falta habilitar una interfaz de red ni abrir un puerto.

Playwright entrega el HTML, CSS, módulos y JSON mediante `route.fulfill` desde
`web/static` del checkout. Rechaza API, POST, otros orígenes, recursos enlazados,
WebSocket y solicitudes fuera de la lista. No usa `route.fetch`, `continue`,
HTTP/listen, PKI ni certificados. Cada contexto bloquea cookies y acceso a
almacenamiento web. El perfil temporal de Chrome y scratch se eliminan al acabar.

Comprueba nombres accesibles calculados por Chrome, etiquetas visibles, Tab y
Shift+Tab, foco visible sin obstrucción y desbordamiento. Reutiliza los helpers F.
Agrupa hasta doce pasos internos de Tab en los campos nativos de fecha/hora de
Chrome para contar cada campo como un control. No evalúa cada segmento por separado.
El ensayo focal comprueba que Tab sale de una fecha y que Shift+Tab vuelve.
También exige detectar un mutante que bloquea Tab: el límite no lo da por válido.
Antes de medir foco espera tres fotogramas con rectángulo y scroll estables,
con un máximo de sesenta. Conserva el desplazamiento y las animaciones del producto.
Abre con teclado el rango del saldo, el formulario de olvido y la resolución de
un permiso. Recorre el filtro de justificantes y comprueba las marcas de ausencia.
No pulsa fichar, enviar el olvido ni resolver: esos adaptadores fallan si se llaman.
La justificación pendiente conserva el aviso de función sin conectar de la vista
vigente; el guion no monta un formulario de justificación inventado.

`informe.json` contiene contadores, estados y geometría; las capturas contienen
solo los datos sintéticos del catálogo. Código 0: todas las comprobaciones
terminaron. Código 1: hallazgo o ejecución incompleta. Una pantalla con hallazgos
no detiene las restantes; un fallo de montaje o interacción corta y registra la
etapa. No se sobreescriben salidas anteriores.

El resultado es parcial: no acredita autenticación, permisos, backend,
persistencia, reinicio ni E2E. Tampoco comprueba contraste, lector de pantalla,
orden lógico completo, estados de error/denegación o el shell del portal. El
helper no incluye `summary` entre sus controles; la apertura del detalle de
marcajes queda pendiente. Ningún resultado certifica WCAG global.
La medida de foco exige que cinco puntos del control estén visibles y libres.
Marca `CORTADO` si uno no lo está, incluso con el control parcialmente visible.
Es más conservadora que el mínimo AA de
[WCAG 2.2, 2.4.11](https://www.w3.org/WAI/WCAG22/Understanding/focus-not-obscured-minimum.html),
que exige que el componente no quede completamente oculto por contenido del autor.
Un aviso de visibilidad parcial requiere revisión; no demuestra por sí solo un
incumplimiento global. El ensayo del host comprueba el desplazamiento por teclado
y rechaza un mutante que pierde la autoridad CSS de `espacio-trabajo`.

# Local Cronos accessibility check

`--plan` reads only the public fixture catalog. It does not open Chrome, read
private configuration or contact a server. Run `--salida` in the isolated
sandbox described above, with an existing Playwright module and system Chrome.
The output directory must be new and its parent private (0700).

The script checks four real views using synthetic data in both languages, at
desktop/mobile widths and native 200% zoom. It opens forms without submitting
them and blocks network forwarding, storage and writes. `informe.json` records
partial accessibility checks; exit 1 also covers findings in the current views.
The result does not establish authentication, authorization, persistence or E2E.
The minimal host uses the portal's real `espacio-trabajo` scrolling contract.
The conservative focus check requires all five sampled points to be clear. It
can flag partially visible controls and is not a WCAG conformance decision.
