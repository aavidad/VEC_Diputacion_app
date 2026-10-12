# Preparar el original de resolución en vec-server

RRHH puede preparar la resolución desde la ficha del expediente y descargar el
PDF conservado en Documentos. La pantalla presenta «Documento preparado;
pendiente de envío a firma», su número VEC y la fecha de custodia. Esta operación
no registra una firma ni envía el documento a Portafirmas.

## Reutiliza

Se monta la fuente CT de #458, la custodia de Documentos #461, los tipos del
catálogo V2 de #459 y las autorizaciones CT de #746. El consumidor de la ficha
es #1033. Se reutilizan el PDP, las asignaciones fijas, las sesiones registradas,
la provisión CAS y el almacén existente. La captura de perfiles compartida con
incorporación B2 conserva sus comprobaciones anteriores. No se añade otro
servicio de identidad, motor de permisos ni almacén.

Documentos conserva el catálogo V1 en sus rutas anteriores. Solo el servicio
del original CT utiliza el catálogo V2 y sus tipos reservados. Ambos comparten
repositorio y almacén. Cada petición resuelve los perfiles de CT y Documentos
de la misma cuenta certificada, sin recibir el perfil desde el navegador.

## Configuración e instalación

CONFIG NUEVA: `VEC_CT_ORIGINAL_FIRMABLE_ENABLED=true`, con
`VEC_DOCUMENTOS_ENABLED=true` y el material existente de Documentos configurado
en `identidad/documentos.json`, bajo el directorio de material de identidad.
El selector del original queda apagado por defecto. Si está activado y faltan
sus dependencias, el arranque falla con el componente correspondiente.

La lista SQL es `lista_sql_codexh_r5_bootstrap_20261012.txt`. AD247 corrige el
orden canónico de dos campos en dos funciones ya instaladas. La migración
comprueba la definición anterior y posterior, propietario, permisos y
configuración; conserva sus OID. Se aplica una sola vez, sin DOWN. AD158 y D13
ya instaladas no se reaplican. AD247 no puede instalarse a ciegas después de
otra migración que reconstruya esas funciones: Dirección debe cotejar su
preimagen y el orden con AD238/AD240 antes del despliegue.

Las asignaciones nominales se publican mediante el mecanismo existente.
Si el arranque indica una asignación pendiente, Dirección debe aprobar su
sustitución CAS con la huella que muestre ese destino. Las huellas de la copia
de ensayo no son valores de configuración para cidonia. Después del arranque
aprobado se retiran las variables de provisión. Una asignación revocada sigue
denegada.

## Comprobación local del 12 de octubre

Ensayo en una copia sintética HZ21 propia, PostgreSQL 18.6, contenedor
`--rm --restart=no --memory 2g`, datos en disco, y Chrome del sistema con
certificado RRHH. No se accedió a cidonia. La copia contiene 71 expedientes,
con plazos vencidos e incidencias.

Desde la ficha de un expediente en nombramiento, versión 7:

1. Pulsar «Preparar resolución para firma». Documentos reserva y confirma el
   original. El primer intento detectó un fallo de representación UTC del
   comprobante después de confirmar la custodia; la corrección normaliza la
   fecha sin modificar el instante persistido.
2. Reintentar: HTTP 200 y recibo `VEC-2026-1`, fecha
   `2026-10-12T13:15:29.549946Z`. Descargar el PDF desde el botón de la ficha:
   otro POST nominal HTTP 200, 29.928 bytes.
3. Reiniciar aplicación y PostgreSQL. Cinco peticiones desde el mismo Chrome
   devolvieron HTTP 200, el mismo recibo, fecha, versión y PDF. La base mantuvo
   una reserva, un intento y una confirmación.

SHA256 del PDF:
`271e6d80cac7e5ac08cbb07a056230027122e73cbab878d3b9e4c50a60e9345b`.
Las cinco recuperaciones después del reinicio duraron 178,5 / 118,6 / 113,9 /
109,1 / 106,9 ms en Chrome. La primera preparación completa observada antes de
la corrección tardó 498,9 ms: esta medición no acredita que la preparación
inicial cumpla 300 ms. La recuperación consulta 14 veces; no recorre ni audita
una vez por cada expediente de la bandeja.

El comprobante y el PDF corresponden al catálogo antiguo de la copia. Sus
textos pendientes de actualización no se reescriben en el original conservado;
la plantilla publicada nueva debe comprobarse con otro original. La copia
también mostró fallos de preferencias, formalización e incorporación B2,
anotados en el canal para sus responsables. Esta evidencia acredita la
preparación y recuperación del original, no el recorrido CT completo, la firma
ni la aceptación de Portafirmas.

## Validación del candidato

Pruebas focales de montaje, perfiles, revocación, frontera y verbos HTTP,
contrato V2 y replay verdes. AD247 instalada una vez en el clon tras revisión,
con comparación de todos los metadatos de `pg_proc` y de sus dos cuerpos.
Revisiones sensibles del cableado y de los deltas: P0=0, P1=0, P2=0.

Semgrep local sin métricas: sin resultados en los archivos modificados.
Gosec de HTTP/custodia: sin errores de carga ni hallazgos en el cambio.
El análisis de bootstrap encontró errores de carga de tipos de la herramienta;
no se presenta como una revisión completa verde. La puerta completa de calidad
se ejecuta en la CI de la PR; su resultado debe comprobarse antes de fusionar.
