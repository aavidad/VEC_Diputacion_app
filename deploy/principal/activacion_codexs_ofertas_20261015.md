# Publicación de ofertas y comienzo del plazo

Estado: en preparación. No instalar hasta completar las revisiones, el ensayo
SQL y el recorrido en Chrome indicados en la PR.

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
   privado y apuntar `VEC_BOLSA_REGLAS_SOURCE_PATH` a esa copia. Guardar la
   referencia anterior, reiniciar y comprobar el arranque; si falla, restaurar
   la referencia y reiniciar. La fuente anterior sigue siendo válida.
3. Abrir la bolsa con el perfil de RRHH. En su política, conservar la versión
   publicada o preparar una nueva con «Al publicar la oferta». Revisar y
   publicar por la vía de gobierno existente; no modificar una versión antigua.
4. Publicar una oferta desde su formulario. El inicio por publicación no pide
   datos de correo. El inicio por notificación conserva fecha, referencia y
   huella del correo declarado por RRHH.
5. Comprobar el recibo y la oferta en Mi Bolsa. Repetir la misma operación con
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
