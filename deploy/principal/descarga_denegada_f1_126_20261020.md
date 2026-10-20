# F1-126: descarga denegada con constancia del intento

Estado: corrección implementada y regresiones focales verdes. El ensayo
en una copia nueva y las dos revisiones finales siguen pendientes.
Encargo de dirección del 20 de octubre, a las 12:13.

## Fallo reproducido

El Técnico puede consultar la historia de una resolución de su unidad, pero
su cargo no tiene un paso de firma para ese documento. La descarga se deniega
antes de leer el PDF. En la copia de F1-58, esa petición respondió 503, también
en una consulta posterior con un selector vigente. Secretaría descargó el
mismo PDF con 200 antes y después de reiniciar.

La denegación debería responder 404 después de registrar el intento. Si no
se puede guardar esa constancia, debe conservar el 503 y no entregar el PDF.

## Recorrido del error

`descarga_firma_r5_v2.go` reutiliza el manejador de consulta para auditar los
fallos. Este pide una orden con `AccionConsultarFirmasR5V2`. La fábrica nominal
comprueba la pareja exacta de ruta y acción antes de crearla: admite las rutas
de consulta y recuperación, pero omite la de descarga. La creación de la orden
falla y el manejador sustituye la denegación por servicio no disponible.

El punto a corregir es `accionAuditoriaLecturaFirmaNominalCT`, en
`internal/app/bootstrap/contratacion_temporal_firma_v2_nominal.go`. Se reutiliza
la lista existente; no hace falta otra función, permiso humano ni migración.

## Comprobación prevista

- Admitir únicamente la pareja de descarga y acción de consulta ya utilizada.
- Mantener denegadas las parejas ajenas y la descarga sin paso del cargo.
- Comprobar 404 con apunte válido y 503 si falla realmente la auditoría.
- Conservar identidad, recurso, correlación y resultado denegado del intento.
- Desde Chrome, comprobar la negativa y una descarga autorizada; repetir las
  lecturas tras reiniciar sin firmar de nuevo ni cambiar las concesiones.

## Cambio y comprobación focal

La lista admite ahora la pareja exacta de descarga V2 y acción de consulta.
El selector de descarga conserva el control del paso del cargo; no se cambia
el lector, la autorización ni el contenido del PDF.

La prueba de la fábrica real comprueba intento, contexto, recurso,
correlación y resultado denegado, y mantiene cerradas otras rutas o acciones.
La prueba HTTP comprueba que un selector denegado no consulta el registro ni
lee el PDF: devuelve 404 con acuse válido, y 503 si falla la orden, el registro
o la validación del acuse. Pasaron ambas pruebas y `go vet` de los dos paquetes,
con Go 1.26.9 y sin acceso a la red.

Semgrep local ejecutó 42 reglas sobre los tres archivos Go sin hallazgos.
GoSec cargó ambos paquetes y no señaló el archivo de producción modificado;
registró 83 diagnósticos en otros archivos, fuera del cambio. Su configuración
oficial excluye los archivos de pruebas. Los informes completos son privados.

CONFIG NUEVA: ninguna. SQL NUEVA: ninguna. La copia de aceptación debe usar
su propia revisión, materiales y firmas reales; no se alterará la revisión
registrada de la copia anterior para arrancar otro binario.
