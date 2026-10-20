# F1-33: edición de plazos sin activar

Al abrir «Reglas vigentes», el panel de plazos consultaba su ruta de edición
aunque no estuviera montada. El resultado era un error genérico.

La lectura existente de reglas indica ahora si está montada la edición de
Contratación temporal. Si no lo está, el panel explica que debe solicitarse
su activación a Administración y evita consultar la ruta ausente. La tabla
permanece disponible. El indicador describe el montaje; los permisos de
lectura y escritura se siguen comprobando en las rutas de ajustes.

## Activación y vuelta atrás

CONFIG NUEVA: ninguna. SQL a instalar: ninguna.

1. Instalar el código y los archivos web de la misma revisión mediante el
   procedimiento habitual de copia local.
2. Conservar la configuración. Con `VEC_CT_REGLAS_AJUSTES_ENABLED` ausente o
   desactivado, abrir «Reglas vigentes» como RRHH. Debe aparecer el motivo de
   edición desactivada, sin peticiones a
   `/api/vec/contratacion-temporal/reglas/ajustes`.
3. Con la edición ya activada, conservar sus conexiones, catálogo, motivos
   y permisos publicados. El panel sigue usando el GET/POST existente.
   Este corte no activa el selector ni concede permisos.
4. Reiniciar aplicación y PostgreSQL con el lanzador de la copia y consultar
   de nuevo. No se registran actos ni se modifica ningún plazo.
5. Para volver atrás, restaurar juntos el binario y los archivos web de la
   revisión anterior. No hay cambios de datos que deshacer.

## Validación del 20 de octubre

El código `58d7399ad` se ensayó en una copia nueva, creada en 99,46 segundos.
Chrome comprobó 1440/1024/390 píxeles en castellano y 390 en inglés, teclado,
zoom al 200 % y alto contraste. Tras reiniciar aplicación y PostgreSQL, las
cuatro vistas conservaron exactamente mensajes y estados: solo GET de reglas
200, sin consultas de ajustes, errores JavaScript, cookies ni desbordes.
La copia se retiró después de guardar la evidencia.

Las pruebas focales Go comprueban que el indicador solo es verdadero con la
ruta montada. Las 28 comprobaciones web cubren contrato, idiomas, historial
y recuperación: el reintento conserva la edición desactivada y una lectura
fallida se recupera antes de decidir si abrir ajustes. Semgrep local pasó
cuatro reglas JavaScript y 42 Go sin hallazgos. GoSec encontró 82 avisos
existentes en los paquetes, ninguno en los tres archivos Go modificados.
La puerta completa corresponde a la integración continua.

Las [capturas de antes y después](evidencias_f1_33/) muestran el resultado
real a 1440 y 390 píxeles. El [informe del barrido](barrido_f1_33_20261020.md)
separa esta corrección de las activaciones pendientes del recorrido completo.
