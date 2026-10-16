# Confirmación del llamamiento nativo de Bolsa

CT230 corrige la confirmación del llamamiento que Bolsa ya ha emitido. El
serializador anterior omite las marcas del contrato nativo de Go y rechaza
su recibo. Se conservan la operación y su clave para recuperar el mismo
llamamiento, sin repetir la oferta ni fabricar un justificante.

## Alcance y límites

La rama nativa exige la autoridad y acción exactas del circuito constituido,
sus referencias, huellas y las dos evidencias coherentes. La marca
`sin_retencion_declarada` por sí sola no permite confirmar. La ruta anterior
mantiene sus controles y su retención obligatoria.

`retener_hasta` expresa la vigencia de la evidencia HMAC de Bolsa; no es el
plazo de conservación del expediente. En la rama nativa vale exactamente
`0001-01-01T00:00:00Z`, conforme al contrato Go existente. La migración no
crea plazos ni interpreta ese valor como conservación indefinida. La
conservación real de O6 sigue pendiente de CT-CUM-06 y de la política
VEC-DOC-CONS-01 y su calendario de series.

Go verifica HMAC y artefacto antes de llamar al ejecutor nominal de CT. SQL
conserva ese límite de confianza; este cambio no añade un verificador
criptográfico independiente ni concede permisos personales.

## Activación

CONFIG NUEVA: ninguna. No hay claves, cuentas, tablas ni permisos nuevos.

1. Preparar la candidata congelada y una copia actual de la principal.
   Cotejar las cuatro huellas, propietarios, ACL y configuración que exige
   CT230. Una diferencia detiene el ensayo; nunca se reaplica una migración.
2. Con la aplicación detenida, instalar una vez la lista
   `lista_sql_codexr_recibo_nativo_o6_ct230_20261016.txt` y ejecutar la prueba
   focal `ct230_recibo_nativo_bolsa.sql`. Conservar las confirmaciones previas.
3. Arrancar el binario de la candidata sin instrumentación. El consumidor es
   «Llamamiento de este expediente» en la ficha de Contratación temporal.
   La candidata conjunta incluye las correcciones de referencias y UTC de
   #1144; CT230 no las sustituye.
4. Recuperar la petición original con su misma clave. Debe devolver el recibo
   del llamamiento ya emitido y dejar una sola confirmación. Comprobar los
   rechazos de marcas o procedencia ajenas, con transacción revertida.
5. Reiniciar aplicación y PostgreSQL y recuperar el mismo recibo. Comprobar
   también el arranque anterior a CT230: las funciones y pantallas previas
   siguen disponibles, aunque la confirmación nativa continúe rechazada.

La vuelta atrás usa la candidata anterior y la copia de ensayo declarada por
dirección. No se ejecuta DOWN ni se alteran recibos existentes. Cidonia queda
fuera de este guion.

## Evidencia pendiente

La propuesta tiene dos revisiones independientes favorables. La preimagen
reproduce el rechazo del canon nativo y conserva seis confirmaciones previas.
Quedan la instalación única, la prueba posterior, la recuperación real y el
reinicio. La PR seguirá en borrador hasta aportar esos resultados.
