# Auditoría de fallos al consultar los contactos del listado de candidatos

Tarea: F0-14. El listado de candidatos consulta sus contactos
por páginas. Las consultas HTTP de contactos ya usan el decorador de auditoría,
pero el listado recibía directamente el servicio sin decorar. Una denegación
o un error posterior a preparar la consulta no dejaba el apunte común.

Se conecta al listado el mismo `ContactosRRHHAuditados` de las otras rutas.
El campo dedicado a la lectura exige ese tipo: no admite conectar el servicio
sin decorar. El servicio original se conserva para configurar los intentos
telefónicos, que usan las mismas instancias. No cambia el preparador, la autorización, los datos de la página
ni el consumo de la lectura positiva en su transacción.

Consumidor: `GET /api/vec/bolsa/bolsas/{bolsa_ref}/candidatos`, usado por
Bolsas de RRHH. `cargarContactos` sigue llamando a `ListarContactosBolsa`;
el decorador registra el fallo con actor revalidado, recurso, motivo y
correlación. No añade correos, teléfonos ni notas de contacto al apunte.
Si falla el registro, no devuelve contactos ni afirma que exista un apunte.

## Activación y comprobación

CONFIG NUEVA: ninguna. SQL a instalar: ninguno. No se crea otra autoridad,
conexión, registro ni tabla. Se reutilizan los componentes ya montados en main.
La aplicación conserva sus selectores habituales, también con Bolsa apagada;
este cambio no añade una condición de arranque.

Pruebas focales positivas:

```sh
go test -p 6 ./internal/app/bootstrap -run '^TestCandidatosContactosUsaServicioAuditadoConContextoNominal$' -count=1
go test -p 6 ./internal/modules/bolsa/application -run 'Test(ContactosAuditados|DatosContactoAuditados|DatosContactoEnmascarados)' -count=1
go test -p 6 ./internal/modules/bolsa/adapters/httpinterno -run 'Test(HandlerDatosContactoAuditaDenegacionEnPreparacion|AuditoriaBorradorLlamamientoRegistraDenegacionTempranaFueraDeProteccion|AuditoriaBorradorLlamamientoIncluyeFronterasB4YB7)' -count=1
go vet -p 6 ./internal/app/bootstrap
```

La prueba focal del consumidor usa el preparador nominal y el decorador real
con dobles de la lectura y del registro. Comprueba denegación, indisponibilidad,
positivo y fallo del registro. Las pruebas existentes del decorador y de las
rutas HTTP completan el cotejo. No acreditan un fallo inducido en una copia
ni una consulta desde Chrome.

## Límites de F0-14

Este corte cubre los fallos posteriores a preparar la consulta del listado.
Antes de disponer de actor revalidado y correlación privada de la petición
no existe un sobre nominal completo para el registro común. La bitácora de
frontera conserva esos intentos, con actor solo cuando ya fue revalidado.
No se inventan actor ni correlación para convertir una petición incompleta en
una consulta nominal. La prueba de correlación de contactos conserva ese límite.

F0-22 conserva la vía positiva existente: su autorización se consume en la
misma transacción que la consulta y devuelve la referencia de auditoría.
No se duplica ese registro en este corte. Dirección decide el estado de ambas
tareas con la evidencia y el recorrido disponible.
