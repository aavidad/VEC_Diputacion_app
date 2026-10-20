# F1-126: descarga denegada con constancia del intento

La corrección mantiene denegada la descarga cuando el cargo no tiene paso
para el documento y registra el intento antes de responder. Adjuntar a la
entrega las evidencias del ensayo y la revisión del binario utilizado.

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

## Activación y comprobación en la copia propia

Se reutilizan el [guion de copia local](LEEME_copia_local.md), la
[activación de cargos de firma](firma_por_cargos_f1_58_20261020.md), la
[semilla de firmantes](candidata_hz39/firma/semilla_firmantes/LEEME.md) y el
[preparador del plan](../../cmd/vec-plan-firma-validar/README.md). Cada copia
genera sus conexiones, certificados, cargos, publicaciones y recibos. Los
archivos de otra copia no sirven como prueba de esta corrección.

1. Crear la copia con la lista SQL completa de su revisión y conservar el
   arranque sin configuración de firma. Comprobar que se abren el portal y
   la ficha de Contratación temporal.
2. Activar los dos administradores y publicar los cargos con los medios
   existentes. La fuente inicial tiene dieciséis roles; los cuatro de firma
   incluyen consulta, firma y recuperación. La semilla procede del generador
   versionado, sin modificarlo; las operaciones de esos cuatro cargos se
   ajustan como datos de la fuente antes de publicarla.
3. Publicar las cuatro versiones de consulta que conservan cuadro y descarga
   y conceden los selectores de firmas. Suceder solo esas cuatro asignaciones,
   conservando personas y ámbitos. Las asignaciones ficticias se declaran como
   tales; no representan actos de asignación de un administrador.
4. Publicar cinco cargos, cinco enlaces y cuatro admisiones de certificado,
   la correspondencia organizativa y el plan de cinco pasos mediante sus
   operaciones existentes. Conservar los recibos originales de cada acto.
   Limitar la autoridad de desarrollo a entre 60 y 90 días; cada asignación
   debe quedar cubierta por el cargo, certificado y admisión correspondientes.
5. Instalar la configuración de firma y reiniciar con vigilancia. Técnico
   prepara el original de una resolución desde su ficha; Jefatura firma
   el primer paso con GrxFirma 0.0.106 o posterior. Registrar el PDF por la
   vía nominal y exigir verificación válida, custodia y recibo. Basta una
   firma nueva para esta comprobación; no se reutilizan firmas anteriores.
6. En la ficha del Técnico, usar la referencia visible de esa firma para
   intentar descargarla. Debe responder 404 sin PDF. Con la correlación de
   la respuesta, cotejar el apunte nominal denegado del mismo expediente.
   Con Jefatura, descargar el PDF y comparar sus bytes con el registrado.
7. Reiniciar aplicación, PostgreSQL y verificador. Releer el apunte original
   y comprobar que conserva referencia, secuencia, fecha y huella. Repetir
   las dos lecturas desde Chrome: una nueva denegación deja otro apunte;
   la descarga autorizada conserva el mismo PDF y la firma original.

La configuración técnica de origen para recuperar firmas es la existente
de AD172: una fila para el proceso, canal y operación exactos del lector.
Se coteja contra el núcleo instalado y se ensaya con reversión antes de
aplicarla. No añade un permiso humano de descarga. Los LOGIN de activación
pertenecen solo a la copia propia y deben tener retirada definida.

La comprobación local puede usar GrxFirma por línea de comandos y registrar
su PDF mediante la API nominal; las consultas y descargas se comprueban en
Chrome. Esto no acredita el selector gráfico del firmador, el recorrido
completo en un solo expediente ni una confirmación externa de Firmadoc.
