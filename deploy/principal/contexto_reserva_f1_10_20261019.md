# Comprobar el contexto antes de reservar

F1-10, primer corte. La ficha del expediente conserva su selección y su
recuperación actuales. Al reservar, se contrasta otra vez el contexto de CT y
la última comprobación de Bolsa, para detectar cambios posteriores a la lectura.
No habilita descuentos, ofrecimientos condicionados ni una excepción a la
detención por comprobación pendiente.

## Activación

CONFIG NUEVA: ninguna.

1. Partir de la lista única de main tras HZ37, con F1-09 completa. Instalar
   B140, AD297 y CT240, en ese orden, desde
   `deploy/principal/lista_sql_main_tras_hz37.txt`. No repetir SQL instalada.
2. Conservar el binario y la configuración existentes de CT y Bolsa. Este
   corte no cambia Go, crea roles ni amplía las concesiones de RRHH. La función
   privada de cotejo solo se concede al propietario de CT.
3. Reiniciar la aplicación y comprobar la consulta de la ficha y la selección.
   Con criterios sin configurar se conserva el comportamiento anterior.

El control de versión pertenece a Bolsa. El registro de hechos y las reservas
usan la misma fila por organización, expediente, necesidad y participación.
Si una rectificación se confirma después de abrir la transacción de reserva,
el conflicto exige reintentar; esperar un bloqueo no basta para actualizar su
instantánea. La historia de actos y sus recibos se conservan.

La lectura solo añade el estado de la comprobación a la persona pendiente de
comprobación. La búsqueda del último acto sigue su numeración por expediente,
necesidad y participación, aunque la bolsa haya cambiado; su vigencia compara
después la bolsa y el resto del contexto.
AD297 ajusta la validación de esa misma lectura auditada y conserva la
recuperación de las reservas anteriores, sin reconstruir el consumidor central.

## Comprobación

En una copia propia y con datos ficticios:

1. Consultar desde la ficha una selección pendiente de comprobación. Mantener
   la posición y el motivo visible; este corte no permite saltar a esa persona.
2. Entre lectura y reserva, rectificar el acto o cambiar el contexto. La nueva
   reserva debe detenerse sin crear un llamamiento, una oferta ni otro recibo.
3. Reintentar una operación ya confirmada: recuperar su recibo original.
4. Reiniciar aplicación y PostgreSQL y repetir las consultas, sin duplicados.

La prueba SQL focal comprueba los cotejos y la concurrencia en la copia;
la prueba de navegador acredita solo las operaciones que se indiquen en la
entrega. No sustituye el recorrido conjunto de F1-57, cuya pantalla de registro
y habilitación siguen pendientes. F1-10 completa también requiere resolver la
duda 165 y su contrato de cálculo y cese efectivo.

Vuelta atrás: recuperar el binario y la configuración anteriores; conservar
las migraciones, los actos y los recibos. No ejecutar DOWN.
