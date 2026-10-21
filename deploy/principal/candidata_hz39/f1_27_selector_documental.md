# F1-27: selector interno para los documentos

El documento necesita identificar la aceptación de Bolsa que ya figura en la
propuesta registrada. Este lector recupera de Contratación temporal diez
referencias opacas conservadas en esa propuesta. No devuelve datos personales
ni añade campos a la respuesta pública del expediente. Su consumidor es la
lectura nominal de Bolsa que preparará los documentos del paso 14.

Reutiliza el permiso de detalle, su sesión ligada al certificado, la decisión
y el apunte de acceso. Antes de entregar el selector comprueba expediente,
organización, unidad, categoría, propuesta de la versión 7 y recibo de la
aceptación. Una discrepancia impide preparar el documento. No cambia los
permisos de las personas que firman.

La propuesta deja de servir a los documentos si después hubo una no
incorporación efectiva o una propuesta que la sustituye: el lector responde
42501. La señal que cuenta es la actuación de no incorporación del expediente
actual. La tabla de no incorporaciones solo la ve la sesión del ejecutor, y la
del consultor de RRHH la ve vacía; se consulta igualmente, pero no basta por
sí sola. La sustitución sí es visible y se comprueba en su tabla. Si falta el
expediente actual, la versión 7 o su propuesta, la respuesta también es 42501.
Antes salía como «sin datos» (P0002) o «demasiadas filas» (P0003).

## Activación y límites

La propuesta tiene que haberse registrado en la versión 7 del expediente
(`version_resultante=7`). Una propuesta registrada en otra versión, como la
sucesora tras una sustitución, falla cerrada: el documento no se prepara. Si
hace falta para ese caso, será otra tarea.

La consulta del selector se repite ante una carrera de serialización (40001)
igual que el detalle, porque consume la misma decisión de lectura.

Instalar CT262 según la lista única de SQL de main, antes del consumidor
documental de autorización y de la lectura propia de Bolsa. No requiere
configuración ni actos nuevos para este lector. No se registra al arrancar ni
se altera el arranque habitual. La composición del consumidor irá en el corte
documental; sin CT262 esa función devuelve un error y no emite un documento.

Las pruebas Go comprueban la lectura con permiso de detalle en versiones 7, 8,
9 y posterior, y rechazan otro expediente, organización, unidad o categoría,
la ausencia de propuesta o selector y el permiso denegado. La prueba SQL mira
el contrato instalado, los permisos y el orden de las guardas; no acredita un
consumo real de autorización. El recorrido con material genuino y el arranque
del consumidor quedan pendientes del montaje y de sus actos de activación.

La migración comprueba la huella del escritor nativo instalado que determina
el formato conservado de la propuesta. No reconstruye ese escritor ni consulta
tablas de Bolsa. La huella se cotejó con la función instalada en la copia J.

Ensayo de instalación: PostgreSQL 18.4 propio, sin red y limitado a 2 GiB,
restaurado del volcado coherente de J con sus roles. La instalación de CT262,
la comprobación del contrato y la prueba de vigencia terminan con código 0.
La prueba de vigencia inserta una sustitución sintética y un agregado con la
no incorporación efectiva, y comprueba que ambos se rechazan; también que una
no incorporación solo propuesta no retira la aceptación. Termina en ROLLBACK.
No se ejecutó una lectura nominal real con material V3, así que el 42501 del
lector completo se comprueba por la posición de la guarda en el cuerpo
instalado. J no recibió SQL ni actos nuevos.
