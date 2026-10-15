# Publicación de ofertas y comienzo del plazo

Estado: ensayado en copia local. La instalación compartida corresponde a
dirección cuando la cabeza de la PR tenga todas las puertas verdes.

CONFIG NUEVA: ningún nombre de variable nuevo. La fuente opcional
`data/catalogos/bolsa/reglas.v7.json` propone publicación; se activa mediante
`VEC_BOLSA_REGLAS_SOURCE_PATH`, con el mismo mecanismo de las fuentes anteriores.
Se conserva el selector de política
de ofertas existente. La publicación requiere B117; sin ella la función debe
fallar cerrada y la aplicación debe seguir arrancando.

## Activación

1. Aplicar una sola vez la lista `lista_sql_codexs_varias_plazas_20261015.txt`
   sobre una copia de la principal. No reaplicar SQL instalada.
2. Si se quiere usar la propuesta nueva, copiar `reglas.v7.json` al montaje
   privado con `install -m 600` y apuntar
   `VEC_BOLSA_REGLAS_SOURCE_PATH` a esa copia. Guardar la
   referencia anterior, reiniciar y comprobar el arranque; si falla, restaurar
   la referencia y reiniciar. La fuente anterior sigue siendo válida.
3. Entrar con el perfil de RRHH en Bolsa → Llamamientos, seleccionar una
   bolsa y pulsar «Gestionar ofertas». En «Política de ofertas», conservar la versión
   publicada o preparar una nueva con «Publicación de la oferta». Guardar
   la política por la vía existente; no modificar una versión antigua.
4. Publicar una oferta desde su formulario. El inicio por publicación no pide
   datos de correo. El inicio por notificación conserva fecha, referencia y
   huella del correo declarado por RRHH.
5. Comprobar el recibo desplegado tras publicar y su consulta posterior en la
   ficha de la oferta. Entrar con el certificado del aspirante y comprobar la
   oferta en Mi Bolsa; aceptar y comprobar el recibo de aceptación. Repetir la misma operación con
   la misma clave: deben conservarse recibo, fecha y plazo originales incluso
   después de cambiar la política.
6. Reiniciar la aplicación y PostgreSQL y consultar la misma oferta. Comprobar
   también el arranque con el selector de política apagado; las demás funciones
   deben seguir disponibles.

La configuración inicial propone «publicación», conforme al artículo 8.1.a y b
del Reglamento de bolsas (BOP 16/01/2026). La alternativa «notificación» conserva
la decisión de RRHH del 02/10; la duda 153 sigue abierta. Un aviso pendiente de
envío no es una notificación.

## Límites del corte

Bolsa guarda su origen CT por referencias opacas y conserva la operación
original al recuperar. La selección y el vínculo desde la ficha de CT llegan
en el siguiente corte. También faltan la guarda de cancelación que comparte
bloqueo con la reserva, la retirada de la oferta al cancelar y el aviso por
correo mediante intención durable. No se acredita el recorrido completo de CT.

## Evidencia del ensayo

La migración B117 se instaló una sola vez desde la preimagen de la copia de la
principal. Su SHA-256 es
`af02f9b3dd73c37c7125d31ac4ddef1d4eeb9d6147f8d4313464cfdb6a84bf38`.
Se comprobaron tres vectores de canon, el rechazo de calendario inválido y las
restricciones de permisos. Dos revisiones independientes aprobaron esa fuente.

En Chrome del sistema, RRHH guardó la política de la bolsa de Administrativo
y publicó una oferta de dos plazas. El reintento devolvió el mismo recibo,
fecha y vencimiento después de cambiar la política y de reiniciar PostgreSQL
y la aplicación. Se perdió la respuesta de otra publicación ya guardada: la
pantalla conservó la operación y el reintento devolvió el recibo existente.
Atrás y Adelante conservaron el recorrido. Hay dos ofertas, dos claves y dos
recibos distintos, sin duplicados por los reintentos.

El aspirante consultó ambas ofertas desde Mi Bolsa y aceptó una con su recibo.
Después de otro reinicio, la consulta devolvió el mismo recibo y la misma fecha
de aceptación. Se comprobaron escritorio de 1440 px, móvil de 390 px y los dos
idiomas, sin desbordamiento horizontal. El ensayo incluye el arranque con la
fuente anterior y la política apagada, y con la fuente v7 y la política activa.

Los datos son ficticios. Se reutilizó el guion de Antonio de
`candidata_hz39/datos/antonio_disponible.sql` para el vínculo con la bolsa
vigente. La copia tenía el perfil anterior limitado al candidato de la bolsa
extinguida: se creó un perfil nuevo de la misma persona y cuenta, revisado
independientemente, y el arranque publicó su asignación inicial normal. No se
modificaron permisos, certificados, decisiones ni recibos anteriores. Ese
material de identidad pertenece solo a la copia de ensayo y se conserva fuera
del artefacto. Para repetir el recorrido se necesita un perfil de aspirante
vigente cuyo ámbito coincida con su candidato y participación.

Las preferencias entre bolsas siguen devolviendo 404 en esta base anterior;
su incorporación corresponde a otro corte. La carga de convocatorias de RRHH
requiere su provisión propia y no forma parte de esta comprobación. Estas
respuestas no impiden consultar ni aceptar las ofertas.
