# Revisión y corrección candidata O5-01

Fecha: 24 de julio de 2026.

Estado: **el corte original recibe `NO-GO`; la corrección `0be5600` queda
pendiente de revisión independiente**. O5-01 continúa abierto porque aún no
incluye caso de uso, bandeja, autorización, persistencia ni notificación
durable.

## Alcance revisado

Se revisaron los commits originales `f4d8c98`–`c36de78`, que modelan la
asignación y reasignación de unidad y responsable en el agregado de
contratación temporal.

La revisión no evalúa todavía una vertical utilizable por RRHH. El corte
original solo modificaba dominio y pruebas.

## Hallazgo bloqueante

La proyección de asignación se enlazaba con la secuencia, versión, acción, fase
y recibo de una actuación, pero no con su propio contenido. Tras rehidratar un
expediente era posible sustituir por referencias distintas y formalmente
válidas:

- la unidad asignada;
- la persona responsable;
- la intención de notificación.

`Expediente.Validar()` aceptaba después el agregado adulterado. La prueba
`TestExpedienteRehidratadoRechazaContenidoDeAsignacionAdulterado` reprodujo
los tres falsos válidos antes de la corrección.

Además, la documentación exigía un motivo gobernado para reasignar, pero el
modelo solo conservaba texto libre en `Observaciones`.

## Corrección candidata

`0be5600`:

- enlaza la proyección con unidad, responsable y notificación exactas;
- conserva y enlaza una `MotivoClave` de catálogo;
- prohíbe motivo de reasignación en el alta inicial;
- exige motivo catalogado y explicación no vacía al reasignar;
- rechaza la adulteración de cualquiera de esos cuatro campos;
- mantiene las referencias opacas y no añade datos personales a la
  actuación.

La ligadura estructural no sustituye el recibo, la historia append-only ni la
integridad criptográfica que deberá aplicar la persistencia O5-05. Impide que
una proyección válida se adjunte accidentalmente a la actuación de otro
contenido.

## Evidencia local

- paquete de dominio: 100 repeticiones;
- paquete de dominio con detector de carreras: 10 repeticiones;
- todos los paquetes de contratación temporal, normales y con carrera;
- `go vet` de contratación temporal;
- `git diff --check`;
- todos los ficheros afectados por debajo de 500 líneas.

## Pendiente para cerrar O5-01

1. resolución autoritativa de unidad, responsable activo y ámbito;
2. caso de uso con PDP V3, segregación e idempotencia;
3. entrada durable de bandeja e intención de notificación en el mismo límite
   transaccional;
4. replay exacto, conflicto semántico y resultado indeterminado;
5. revisión independiente de la corrección;
6. API, web y E2E pertenecen a O5-05 y no se simulan aquí.
