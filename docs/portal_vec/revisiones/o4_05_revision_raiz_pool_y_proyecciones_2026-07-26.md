# O4-05: revisión de raíz, pool y proyecciones RRHH

Fecha: 26 de julio de 2026.

## Resultado de dirección

La cápsula de ciclo de vida obtuvo `GO` y se publicó en `69b6a14`. La primera
versión de la acreditación del pool PostgreSQL y la primera versión de las
proyecciones de cuadro y detalle recibieron `NO-GO` independiente. No se
integraron ni incrementaron el porcentaje funcional.

El contrato corregido está en
[proyecciones RRHH](../proyecciones_rrhh_cuadro_detalle_contratacion_2026-07-26.md).

## Cápsula de aplicación: GO

Quedaron acreditados:

- propietario atómico y exclusivo de servidor y recursos;
- ausencia de registro global que retenga pools o material sensible;
- apagado limitado por contexto;
- cierre inverso, idempotente y saneado;
- rechazo de copia, nulo tipado, duplicado y doble propietario;
- ausencia de cierre de recursos durante una escucha activa;
- constructor privado y producción todavía cerrada.

Evidencia:

- `go test -count=1 ./internal/app/composicion/interna`;
- `go test -race -count=1 ./internal/app/composicion/interna`;
- `go vet ./internal/app/composicion/interna`;
- cincuenta repeticiones focales con detector de carreras;
- `gofmt` y `git diff --check`.

## Pool lector O4-05: NO-GO inicial

La implementación inicial validaba la configuración estática y una conexión
física, la devolvía al pool y ejecutaba después sobre otra conexión posible.
Esto no ligaba la evidencia TLS, identidad y ACL a la operación real.

Además:

- `BeforeConnect` podía cambiar host, TLS o usuario por conexión;
- `pg_stat_ssl.ssl` confirmaba cifrado, pero no `verify-full`;
- el runner real usaba socket Unix sin TLS y no acreditaba la ruta productiva.

Condiciones de nueva revisión:

1. adquirir y acreditar la configuración efectiva y los catálogos en cada
   operación;
2. comenzar la transacción en esa misma conexión;
3. liberar exactamente una vez tras `COMMIT` o `ROLLBACK`, sin cerrar el pool;
4. rechazar callbacks capaces de alterar conexión, TLS, identidad o sesión;
5. probar el runner actual y añadir una matriz TLS PostgreSQL con CA y
   certificado de ensayo antes de autorizar producción.

## Proyecciones RRHH: NO-GO inicial

La primera versión era hexagonal, no serializaba contexto o capacidad y
saneaba errores, pero no podía aceptarse porque:

- no validaba organización, centro o unidad en cada fila;
- no comprobaba filtros ni orden estable en el resultado;
- aceptaba `version_observada` sin cotejarla con el detalle;
- el DTO omitía datos operativos necesarios de solicitud, análisis, cobertura
  y asignación;
- serializaba referencias internas de lectura y auditoría;
- no revalidaba contexto y capacidad en el instante durable del recibo;
- el ámbito de organización no exigía su referencia exacta;
- la capacidad no tenía una duración máxima.

Condiciones de nueva revisión:

1. validar página y detalle contra la orden sellada completa;
2. ligar cada fila a organización y ámbito;
3. revalidar vigencia en el instante durable de lectura;
4. aplicar filtros y orden en la segunda barrera de aplicación;
5. usar DTO explícitos suficientes y minimizados;
6. mantener fuera actores, contactos, DNI, responsables, notificaciones,
   documentos, textos libres y recibos internos;
7. probar cruces de organización/ámbito, versión obsoleta, caducidad,
   serialización, orden y clonación defensiva.

## Métrica

El procedimiento permanece en `19/46` tareas verificadas, un 41 %. O4-05
permanece en tres de cinco hitos internos hasta completar raíz con dependencias,
proyecciones protegidas y E2E HTTP real.
