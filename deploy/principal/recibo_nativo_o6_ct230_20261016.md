# Confirmación del llamamiento nativo de Bolsa

CT230 corrige la confirmación del llamamiento que Bolsa ya ha emitido. Los
serializadores anteriores omiten las marcas del contrato nativo de Go y la
huella SQL omite dos campos del material que firma Bolsa; por ello se rechaza
el recibo. Se conservan la operación y su clave para recuperar el mismo
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
   Cotejar las cinco huellas, propietarios, ACL y configuración que exige
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

## Comprobación local del 16/10/2026

Candidata: aplicación `28320e6c7` de #1144 y SQL `6713aaa13` de #1170,
sin instrumentación. La copia parte de HZ37 y contiene el prefijo completo
de #1147, las cuatro migraciones propias de R y AD278/AD283. Conserva los
anexos de permisos de base y tipos y las tres filas instaladas por AD278.
La preparación y el traslado de datos tuvieron dos revisiones independientes.

- Antes de CT230, aplicación, sesión y portal responden 200. Lo mismo ocurre
  con CT230 y después de reiniciar aplicación y PostgreSQL, una vez disponibles.
- CT230 se instaló una sola vez; sus cinco preimágenes y postimágenes,
  propietarios y permisos coinciden. La prueba focal termina correctamente.
- La petición original, recuperada mediante el cliente web real desde Chrome,
  devuelve el recibo del llamamiento que Bolsa ya había emitido. Hay un solo
  origen y una sola confirmación nativa; las seis anteriores conservan sus bytes.
- Alterar la marca o la política provisional sin actualizar la huella rompe
  el vínculo del material. La política alterada tampoco supera el cotejo
  canónico. El negativo usa el artefacto real en una transacción de solo lectura
  y termina con rollback; no crea firmas ni confirmaciones.
- Después de reiniciar ambos procesos, la misma petición devuelve exactamente
  el mismo recibo HTTP. El recibo y el artefacto almacenados conservan su huella
  y sigue habiendo un origen y una confirmación.

Las dos ratificaciones SQL y de seguridad cubren la fuente exacta:
`2e9c42038af5c1c639fbbf05186ebf4026a13b5f3013c9cb62c6dae19aca6442`.
La prueba tiene huella
`2f088f5533e41785e741daf156e6f5b1f79fa46b7f8449fd9d3871e21165e9c7`.
Semgrep local, con tres reglas genéricas verificadas, no encuentra incidencias
en los dos archivos SQL. Las revisiones semánticas son independientes de ese
análisis de texto.

El primer ensayo, que detectó el quinto cuerpo pendiente, permanece archivado
sin reaplicar la migración. Las identidades copiadas conservan sus bytes; el
material nuevo de gobierno y TLS dura noventa días. Se conserva la excepción
anterior de identidad heredada, hasta el guion de copia de L; no se regeneró
para eludir sus vínculos.

## Límites del recorrido visible

La recuperación prueba la confirmación y la idempotencia. Al recargar la ficha,
su consulta enumera solo comunicaciones y aún no recupera la selección confirmada
pendiente de aviso: muestra un vacío incorrecto y vuelve a ofrecer selección.
Ese hueco necesita un corte propio sobre la lista existente, con sus campos
autorizados y recuperación del formulario. No se ha creado otro llamamiento
para sortearlo. CT230 no acredita aviso por correo, respuesta de la persona ni
el recorrido completo de FLUJO_APP §4.0. La conservación real sigue pendiente
de su política general.
