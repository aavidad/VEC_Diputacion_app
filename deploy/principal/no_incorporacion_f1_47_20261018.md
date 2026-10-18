# F1-47: no incorporación desde la ficha

Comprobación del 18 de octubre de 2026 sobre la principal `2e3fc5e8f`.
El formulario, el cliente y la operación ya existían. Este corte corrige los
avisos del archivo, que aparecían en castellano también al elegir inglés.
Reutiliza el control común de archivo, sus comprobaciones y sus traducciones
inyectables; no modifica el cálculo del código ni el envío de datos.

## Activación

CONFIG NUEVA: ninguna. SQL nueva: ninguna.

El portal debe tener preparado el circuito existente de Contratación temporal:
conexiones nominales, gobierno, catálogos y permisos de RRHH. El inventario de
sus dependencias está en `config/selectores_dependencias_bolsa_ct_v1.json`.
Sobre esa preparación, hacen falta:

- `VEC_CT_SEGUIMIENTO_CESE_ENABLED=true`;
- `VEC_CT_INCORPORACION_ACREDITADA_ENABLED=true`;
- regla vigente `c22.no_incorporacion`, con sus motivos y segunda revisión;
- las conexiones de Bolsa y del relevo de no incorporación que exige el selector.

Receta existente: `scripts/operacion/hito7_b2/LEEME.md`. No copie conexiones ni
recibos de otra instancia. Detenga la aplicación antes de cambiar su material
privado y reiníciela vigilando los registros.

## Recorrido pendiente de acreditar

En un expediente en nombramiento, con aceptación y sin incorporación ni cese,
RRHH abre el seguimiento de la ficha. Elige motivo, fecha de notificación,
referencia y archivo de la resolución. Solo se registra el código del archivo;
la persona conserva el original. Con segunda revisión, otra persona de RRHH
confirma o rechaza desde su sesión. El cliente usa la operación existente
`POST /api/vec/contratacion-temporal/no-incorporaciones`.

La aceptación de F1-47 exige observar su recibo real, recuperar el estado y
comprobar la consecuencia en Bolsa después de reiniciar. Esa parte sigue
pendiente: el arranque básico del guion de copia no prepara las conexiones
nominales del circuito; la consulta de Contratación temporal responde 503.
No se ha registrado una no incorporación real en este ensayo.

## Evidencia obtenida

- Copia propia restaurada con el guion de F0-28; compilación con la herramienta
  declarada en `go.mod`. Faltaban dependencias en la caché estándar al primer
  intento. Tras prepararlas, `/livez` y el portal responden 200, también después
  de reiniciar aplicación y PostgreSQL. Sin configuración opcional.
- Chrome del sistema con el componente y los estilos reales, catálogos es/en
  y datos preparados: formulario y propuesta a 1440 y 390 px sin desborde.
  Esta comprobación aislada no acredita un recibo del servidor.
- Sin archivo, se bloquea el envío, se enfoca el campo y el aviso usa el idioma
  del panel. Con archivo, el control calcula su código de comprobación.
- El ensayo con escalado CSS al 200 % en 390 px desborda por el ancho mínimo de
  la página; no equivale al zoom del navegador y no se presenta como prueba
  superada. No se modifica la disposición con ese resultado.

Capturas del formulario y del aviso, con datos preparados:
[castellano, escritorio](../../docs/operacion/capturas/f1-47/es-1440.png),
[castellano, móvil](../../docs/operacion/capturas/f1-47/es-390.png),
[inglés, escritorio](../../docs/operacion/capturas/f1-47/en-1440.png),
[inglés, móvil](../../docs/operacion/capturas/f1-47/en-390.png).

Pruebas focales: `seguimiento-no-incorporacion.test.mjs`,
`seguimiento-cese.test.mjs`, `portal-cache-ct-a.test.mjs`,
`portal-carga-rapida.test.mjs` y `portal-i18n.test.mjs`.
La nueva regresión comprueba el aviso en inglés y el bloqueo con foco en el
campo de archivo. Las pruebas anteriores conservan la validación de la
solicitud, la separación de personas y el recibo.
