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

## Activación y límites

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
