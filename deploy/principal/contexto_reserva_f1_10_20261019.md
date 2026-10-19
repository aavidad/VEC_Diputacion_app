# Comprobar el contexto antes de reservar

F1-10, primer corte. Al reservar una selección, el servidor vuelve a leer en
Bolsa el contexto del expediente y la última comprobación de cada persona
pendiente, y los compara con lo que CT leyó antes. Si algo ha cambiado, la
reserva no se guarda. Este corte no habilita el descuento del tiempo trabajado,
el ofrecimiento condicionado ni la continuación hasta el cese efectivo (duda
165): la persona pendiente de comprobación sigue deteniendo la selección.

## Qué cambia

- B140 (Bolsa) crea la tabla de control `version_comprobacion_reserva_v1`, una
  fila por organización, expediente, necesidad y participación. El registro de
  la comprobación y la reserva toman la misma fila y la misma llave de
  bloqueo. La historia de comprobaciones sigue siendo de solo adición.
- B140 añade `cotejar_comprobacion_reserva_ct_v1`, que solo puede ejecutar el
  propietario de CT. La reserva singular de Bolsa (`reservar_llamamiento_ct_v1`)
  la llama cuando hay criterios publicados y la reserva es nueva; un reintento
  con la misma clave recupera el recibo original sin volver a cotejar.
- B140 deja el estado de la comprobación solo en la persona con razón
  `pendiente_comprobacion_rrhh` y busca su último acto sin filtrar por bolsa;
  la marca `vigente` sigue comparando la bolsa y el resto del contexto.
- AD297 adapta la fachada auditada de AD293 a esa proyección. Las reservas
  antiguas que guardaron el estado en todas las entradas se recuperan igual.
- CT240 hace lo mismo en la reserva de oferta de varias plazas (CT220): consume
  la autorización, ordena dos reservas del mismo expediente con una llave
  propia, coteja en Bolsa y después bloquea el expediente. La oferta no
  selecciona persona, así que solo comprueba que nada ha cambiado; no añade
  detención por pendientes, igual que antes.

Una rectificación confirmada después de la lectura detiene la reserva
(SQLSTATE 55000). Si se confirma en mitad de la transacción de reserva, la
reserva falla con 40001 y hay que reintentarla; si la reserva llega antes, es
la rectificación la que debe reintentarse. En la reserva singular de Bolsa,
Go traduce ambos casos a «no disponible»; en la oferta CT, a conflicto del
vínculo. Al reintentar, la ficha lee de nuevo el orden y la comprobación.

## Activación

CONFIG NUEVA: ninguna. No cambia Go, roles ni concesiones de RRHH.

1. Partir de la lista única de main tras HZ37, con F1-09 completa (B137 y
   AD293). Instalar, en este orden y una sola vez, B140, AD297 y CT240 desde
   `deploy/principal/lista_sql_main_tras_hz37.txt`. Cada una comprueba la
   huella de los cuerpos que sustituye y la de los que deja; no hay DOWN.
2. Conservar binario y configuración. Reiniciar la aplicación y consultar la
   ficha y la selección. Sin criterios de ofrecimiento publicados, la reserva
   se comporta como antes.

Falta para su uso real: publicar criterios de ofrecimiento en la bolsa, la
pantalla de registro de la comprobación y su habilitación (AD291 y F1-57,
aparcadas) y el contrato de cálculo y cese efectivo de la duda 165.

## Comprobación

En una copia propia, con datos ficticios y superusuario local:

```text
psql -f deploy/postgresql/bolsa_llamamientos/pruebas_sql/b140_recotejo_reserva.sql
psql -f deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/ad297_estado_solo_pendiente.sql
psql -f deploy/postgresql/contratacion_temporal/pruebas_sql/ct240_recotejo_reserva_oferta.sql
```

`b140_recotejo_reserva.sql` prepara su caso con `b140_preparar_caso.sql`
(expediente fiscalizado pasado a la categoría de la bolsa, criterios
publicados y una persona pendiente detrás de la primera elegible) y lo revierte
todo. Comprueba la proyección, la reserva y su reintento, la oferta sin
persona, la detención, el contexto cambiado, la fecha escrita con `Z` por Go,
la rectificación con otra bolsa, la rectificación fuera del control, los
permisos y el aislamiento.

`b140_carrera_rectificacion_reserva.sh` prueba la carrera con dos sesiones
reales. Confirma datos ficticios, así que solo se ejecuta en una copia que
luego se borra (`B140_COPIA_DESECHABLE=si`).

Tras instalar B140 y AD297, `b137_guardas_estado.sql` y
`ad293_estado_comprobacion_guardas.sql` fallan por huella, porque fijan el
cuerpo anterior; `b137_estado_comprobacion_fixture.sql` exige además que el
contexto tenga una persona pendiente. Sus sustitutas son las tres pruebas de
arriba.

Vuelta atrás: recuperar binario y configuración anteriores; conservar
migraciones, actos y recibos. No ejecutar DOWN.
