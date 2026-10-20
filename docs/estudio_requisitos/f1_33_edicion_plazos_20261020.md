# F1-33: edición de plazos sin activar

Al abrir «Reglas vigentes» desde RRHH, el panel de plazos consultaba su
ruta de edición aunque no estuviera montada. El resultado era un error
sin explicar qué faltaba.

La lectura existente de reglas indica ahora si está montada la edición de
Contratación temporal. Si no lo está, el panel explica que debe solicitarse
su activación a Administración y no consulta la ruta ausente. La tabla de
reglas permanece disponible. El indicador describe el montaje: los permisos
de lectura y escritura se siguen comprobando en las rutas de ajustes.

## Activación y vuelta atrás

CONFIG NUEVA: ninguna. SQL a instalar: ninguna.

1. Instalar el código y los archivos web de la misma revisión, con el
   procedimiento habitual de copia local.
2. Conservar la configuración de la copia. Con
   `VEC_CT_REGLAS_AJUSTES_ENABLED` ausente o desactivado, entrar como RRHH
   y abrir «Reglas vigentes». Debe aparecer el motivo de edición desactivada,
   sin peticiones a `/api/vec/contratacion-temporal/reglas/ajustes`.
3. En una copia que ya tenga activada la edición, conservar sus conexiones,
   catálogo, motivos y permisos publicados. El indicador será verdadero y
   el panel seguirá usando el GET/POST existente, con sus comprobaciones.
   Este corte no activa el selector ni concede permisos.
4. Reiniciar aplicación y PostgreSQL con el lanzador de la copia y volver a
   consultar. No se registran actos ni se modifica ningún plazo.
5. Para volver atrás, restaurar juntos el binario y los archivos web de la
   revisión anterior. No hay cambios de datos que deshacer.

## Comprobaciones

- Go: metadato falso sin ruta, sin manejador y con otra ruta; verdadero solo
  con la ruta de ajustes montada. Se conserva la protección por certificado.
- JavaScript: contrato, traducciones y panel en Chrome; sin montaje no se
  consulta ajustes, con montaje se conservan lectura, edición e historial.
- Un fallo de los textos de la tabla conserva el panel de ajustes independiente.
- Si falla la lectura del montaje, «Reintentar» vuelve a consultarla antes de
  decidir si abrir ajustes. Un fallo de textos conserva también ese estado.

El barrido completo por perfiles continúa en F1-33. Este corte corrige
únicamente el aviso de edición de plazos.

## Evidencia del 20 de octubre

Sobre `016fd30ef`, copia nueva creada en 104,53 segundos. La app arrancó con
la edición desactivada. Chrome comprobó 1440, 1024 y 390 píxeles en castellano,
y 390 en inglés; también teclado, zoom al 200 % y alto contraste. Cada vista
hizo una sola lectura de reglas con respuesta 200, sin consultar ajustes.
Tras reiniciar la app y PostgreSQL, las cuatro respuestas, mensajes y estados
fueron idénticos; sin errores JavaScript, cookies ni desbordamiento.

La revisión independiente de Codex-X dio su visto bueno a `016fd30ef`.
El ajuste posterior reutiliza la cabecera del panel para alinear el título y
asociar el nombre accesible de la sección; no cambia lecturas ni permisos.
Las capturas de antes y después se conservan en
[evidencias_f1_33](evidencias_f1_33/).

Las pruebas focales Go pasaron. Las 28 comprobaciones web quedaron verdes
entre la ejecución general y la recuperación focal de los dos fallos señalados
en revisión. Semgrep local: cuatro reglas JavaScript y 42 Go, sin hallazgos.
GoSec analizó los paquetes: 82 avisos existentes, ninguno en los tres archivos
Go modificados. No se ha ejecutado la puerta completa en local: corresponde
al trabajo de calidad de la integración continua.

El último ajuste de cabecera se capturó sobre `c044f8ddf`, a 1440 y 390
píxeles, con los archivos reales y el montaje visual local de las pruebas.
Las imágenes `plazos-panel-final-*` muestran ese resultado, sin errores
JavaScript ni desbordes. Esta comprobación de aspecto no usa la base ni
registra actos. El código Go del lector, manejador y composición coincide
con el de la copia `016fd30ef`; la actualización de la principal no lo cambió.
La preparación de otra copia completa se interrumpió por memoria insuficiente
antes de compilar o arrancar la app, y se retiró. No se presenta como ensayo.


La comprobación final sobre `58d7399ad`, ya con la cabecera alineada y la
principal actualizada, se completó después de recuperar margen de memoria.
Copia nueva creada en 99,46 segundos, mediante el mismo guion estándar.
Chrome repitió 1440/1024/390 en castellano y 390 en inglés, teclado,
zoom al 200 % y alto contraste. Tras reiniciar aplicación y PostgreSQL, las
cuatro vistas conservaron exactamente los mensajes y estados: solo GET de
reglas 200, sin consultas de ajustes, errores JavaScript, cookies ni desbordes.
Las imágenes `plazos-despues-*` corresponden a esta copia final. No se
registraron actos ni cambios de plazos; la copia se retiró tras guardar la
evidencia. Los cambios posteriores de esta entrega son solo documentales.
