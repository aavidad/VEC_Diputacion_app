# O5-01 — asignación de unidad, responsable y bandeja

Fecha: 23 de julio de 2026.

Estado: corrección candidata de dominio en `0be5600`, probada localmente y
pendiente de caso de uso y revisión independiente. El corte original recibió
`NO-GO` porque no ligaba el contenido de la asignación a su actuación. No está
integrado ni habilita producción.

## Resultado funcional

El caso de uso permitirá:

1. asignar un expediente con vía de cobertura decidida a una unidad gestora y
   a una persona responsable activa;
2. crear en la misma operación una entrada de bandeja y una intención de
   notificación;
3. reasignar el expediente con motivo obligatorio, conservando toda la
   historia anterior;
4. repetir una petición exacta sin duplicar asignación, bandeja, auditoría ni
   notificación;
5. rechazar una misma clave idempotente con otro contenido.

La asignación no concede por sí sola acceso al expediente. El PDP sigue
resolviendo cada operación, finalidad, organización, unidad y fase con
denegación predeterminada.

## Frontera de confianza

El cliente puede aportar solamente:

- referencias de autenticación, sesión y perfil;
- organización y expediente;
- versión esperada;
- unidad y responsable solicitados;
- clave UUIDv4 de idempotencia;
- motivo gobernado de reasignación y observación limitada.

No puede aportar actor efectivo, instante oficial, permiso, flujo, estado,
catálogo publicado, recibo, referencia de notificación, entrada de bandeja,
auditoría ni outbox.

La aplicación resuelve el actor en una frontera VEC de garantía alta y cruza:

- organización del contexto, expediente y catálogo;
- unidad activa y ámbito organizativo;
- responsable activo, adscrito a la unidad y habilitado para la tarea;
- fase y estado permitidos por el flujo publicado;
- concesión exacta `contratacion_temporal.unidad.asignar`;
- segregación configurada y versión del expediente.

Las referencias de persona son opacas. Las bandejas y notificaciones no
transportan datos personales innecesarios.

## Transacción futura

La frontera durable de O5-05 realizará en un único `COMMIT`:

```text
consumir concesión e idempotencia
  + CAS del expediente
  + nueva versión append-only de la asignación
  + actuación
  + entrada de bandeja
  + auditoría
  + recibo
  + outbox de notificación
```

Un timeout posterior al `COMMIT` se reconciliará por ámbito y huella HMAC. No
se repetirá una notificación a ciegas ni se interpretará la cancelación como
rollback después de un recibo válido.

O5-01 define el dominio, el caso de uso y los puertos neutrales. No fingirá
persistencia, entrega de mensajes ni autorización productiva.

## Corte de dominio implementado

El agregado ya:

- registra una primera asignación y liga su proyección a la actuación exacta;
- liga unidad, responsable, notificación y motivo catalogado al vínculo de la
  actuación;
- reasigna con CAS, motivo no vacío, instante posterior y nuevo destino;
- impide reutilizar la referencia de notificación;
- impide cambiar fase o estado de forma implícita;
- conserva sin cambios la instantánea anterior;
- rechaza al rehidratar una asignación ligada a otro recibo o instante.

La ligadura contiene secuencia, versión, acción, fase y recibo. La asignación
conserva unidad, responsable, notificación, instante, motivo catalogado y
explicación. El agregado cruza ambos contenidos antes de considerarse válido.

La revisión adversarial y el alcance de la corrección están en
[la revisión O5-01](revisiones/o5_01_revision_y_correccion_2026-07-24.md).

Faltan todavía el caso de uso, catálogo de destinos, autoridad VEC V3,
idempotencia, bandeja, intención de notificación, frontera transaccional y
revisión independiente. Por ello O5-01 no se contabiliza como cerrada.

## Reasignación

Una reasignación:

- exige que ya exista asignación;
- exige cambiar unidad, responsable o ambos;
- exige motivo de catálogo y explicación no vacía;
- mantiene fase y estado salvo transición expresa publicada;
- usa un instante posterior a la asignación vigente;
- genera una nueva notificación y entrada de bandeja;
- nunca reescribe la actuación ni la versión anterior.

La baja o ausencia de la persona responsable no reasigna automáticamente. Crea
una incidencia operativa para decisión humana; un conector de directorio no
adopta la decisión.

## Canales

El mismo comando será consumible por web, escritorio, API, CLI y MCP. No usa
cookies ni almacenamiento del navegador como autoridad o estado durable.

## Pruebas de cierre

- alta correcta y reasignación motivada;
- rechazo sin vía de cobertura;
- rechazo de responsable inactivo, de otra unidad u organización;
- rechazo de versión concurrente;
- segregación y denegación por defecto;
- replay exacto y conflicto semántico;
- cancelación antes del efecto y resultado indeterminado posterior;
- una sola entrada de bandeja y outbox bajo concurrencia;
- copias defensivas, límites, nulos tipados y errores sin causas privadas;
- pruebas de carrera, `go vet`, tamaños, secretos y suite global.

O5-01 no se marcará cerrada hasta que un revisor distinto del productor
reproduzca estas puertas.
