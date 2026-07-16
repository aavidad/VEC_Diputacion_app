# Migración de baremación a idempotencia semántica V2

## Estado y objetivo

Decisión de arquitectura adoptada el 15 de julio de 2026. Su implantación está
pendiente. El flujo V1 de reserva y confirmación permanece sin exposición
productiva.

El objetivo es que una pérdida de respuesta, un reinicio, una sesión nueva o
una rotación de claves nunca dupliquen firma, custodia, retención, versión de
baremación, auditoría ni evento de salida. Cada intento obtiene autorización
actual; la autorización del intento no modifica la identidad estable de la
operación.

## Por qué V1 no se puede abrir

V1 mezcla intención administrativa y circunstancias del intento:

- la reserva incorpora clave del cliente en claro, sesión, autorización y
  ventanas temporales;
- la confirmación incorpora otro contexto de autorización y el token de
  reserva;
- PostgreSQL compara autorización, vínculo de sesión e instantes originales al
  recuperar;
- el adaptador genera el token y conserva solo su SHA-256, de modo que un
  `COMMIT` aplicado cuya respuesta se pierda deja una capacidad irrecuperable;
- reserva y confirmación se sellan sobre preimágenes distintas, pero SQL V1
  exige la misma `huella_solicitud_hmac`;
- la prueba SQL reutiliza un HMAC literal y no descubre esa incompatibilidad
  con la aplicación Go;
- las referencias de auditoría y evento se vuelven a generar en cada intento;
- el manifiesto probatorio V1 cubre autorizaciones efímeras de reserva y
  confirmación;
- el flujo custodia el PDF antes de crear una operación idempotente durable;
- los recibos de custodia y retención no tienen todavía una representación
  canónica material propia;
- un error de `COMMIT` se degrada a indisponibilidad genérica y una reserva ya
  confirmada se interpreta como conflicto.

No se corregirá este conjunto relajando una comparación o reutilizando el
sello de una fase en otra. Se crea un contrato V2 independiente y se retiran
los permisos de ejecución V1 solo cuando el corredor completo sea verde.

## Separación obligatoria de responsabilidades

La operación se divide en dos circuitos durables:

1. La saga documental prepara, firma, valida, sella, aumenta, custodia y aplica
   retención. Fija cada referencia e identidad de efecto antes de invocar el
   conector y puede reanudar desde el último recibo probado.
2. La operación transaccional incorpora a la baremación el resultado completo
   y verificado de la saga. Confirma versión, auditoría y evento de salida en un
   único `COMMIT`.

La incorporación nunca vuelve a producir un efecto documental. La saga nunca
confirma por sí sola el agregado administrativo.

## Índice e intención estables

El índice se deriva en HSM/KMS, con separación de dominio, sobre:

```text
esquema + despliegue + módulo + acción + principal interno estable
        + clave del cliente
```

No incorpora perfil, sesión, método de autenticación, autorización,
correlación ni tiempo. La clave del cliente cruza cada adaptador de entrada una
sola vez dentro de un tipo protegido y no se persiste ni registra.

Una HMAC distinta cubre la intención material exacta. La intención incluye:

- versión base y objetivo del agregado;
- decisión técnica y huellas administrativas;
- política y artefactos de firma;
- plan durable completado de la saga;
- clasificación, MIME, tamaño, inmovilización y referencia/versionado del
  objeto;
- manifiesto material V2;
- recibos canónicos V2 de recuperación, custodia y retención;
- política y fecha de retención con precisión de microsegundo;
- resultado esperado y motivo protegido mediante HMAC de dominio propio.

Se excluyen autorización, sesión, token, correlaciones, referencias aleatorias
de auditoría/evento y tiempos del intento. DNI, nombre, correo, rutas, cubos y
direcciones firmadas están prohibidos. El sujeto y el principal se representan
mediante seudónimos HMAC tipados y verificados.

La rotación conserva un número acotado de generaciones de lectura. El
derivador produce candidatos actuales e históricos en orden determinista; más
de una coincidencia durable es evidencia no confiable. Índice, intención,
principal, sujeto y motivo usan dominios y claves diferenciados.

El alta inicial tendrá su propia `IntencionAltaBaremacionV1`. No se fabrican
valores vacíos para hacerla pasar por una incorporación de decisión.

## Estados y capacidad de reserva

La operación idempotente solo admite:

- `ausente`: no existe operación;
- `en_curso`: existe la intención, pero no el resultado final;
- `confirmada`: existen conjuntamente versión, auditoría y evento.

La capacidad de reserva tiene un ciclo separado:

- `activa`;
- `sustituida`;
- `caducada`;
- `destruida`.

Caducar o rotar la capacidad no cambia la intención ni libera el índice para
otros datos.

El token se genera con 256 bits aleatorios antes de abrir la transacción. Se
cifra mediante AEAD con una clave de KMS/HSM y datos asociados formados por
esquema, operación, índice, HMAC de intención, revisión y caducidad. PostgreSQL
conserva sobre cifrado, referencia de clave, huella de datos asociados y
SHA-256 del token, nunca el token ni una clave de descifrado.

Una respuesta perdida se recupera con autorización nueva y devuelve el mismo
sobre. Rotar la capacidad crea otro token y revisión; reenvolver la clave de
cifrado no cambia token ni intención. Tras confirmar no existe un camino
ordinario para recuperar el token.

## Puerto de operaciones V2

La frontera propuesta es:

```go
type RepositorioOperacionesBaremacion interface {
    ConsultarEstadoOperacion(...)
    PrepararOperacion(...)
    ReanudarOperacion(...)
    RotarCapacidadOperacion(...)
    ConfirmarOperacion(...)
    RecuperarResultadoOperacion(...)
}
```

Cada método recibe un `ContextoOperacionBaremacion` nuevo, exacto y de un solo
uso. Las acciones de política son independientes:

- `bolsa.baremacion.operacion.consultar`;
- `bolsa.baremacion.operacion.preparar`;
- `bolsa.baremacion.operacion.reanudar`;
- `bolsa.baremacion.operacion.capacidad.rotar`;
- `bolsa.baremacion.operacion.confirmar`;
- `bolsa.baremacion.operacion.resultado.recuperar`;
- `bolsa.baremacion.operacion.reconciliar`, reservada al trabajador interno.

Si `ConfirmarOperacion` descubre que la operación ya estaba confirmada, no
reutiliza su autorización para revelar el resultado. Devuelve un estado tipado
y la aplicación solicita una concesión específica de recuperación.

## Resultado de transacción indeterminado

La referencia de operación se fija antes de comenzar la transacción. Cualquier
error durante o después de enviar `COMMIT` se clasifica como «podría haberse
aplicado». No equivale a `ROLLBACK`, no borra objetos y no permite repetir
efectos.

El siguiente intento se autoriza de nuevo y consulta los índices candidatos:

- si está confirmada, recupera versión, auditoría y evento completos;
- si está en curso, recupera la misma capacidad protegida y reanuda;
- solo crea una operación si una consulta autoritativa acredita que sigue
  ausente.

La forma sintáctica de una HMAC no acredita «no aplicada». Esa conclusión
requiere evidencia autenticada y un verificador autoritativo. Cualquier valor
nulo, desconocido, incompleto o manipulado obliga a reconciliar.

## Puerta previa al DDL — dictamen de 16 de julio de 2026

**Estado: NO-GO para redactar o aplicar DDL productivo V2.** Este dictamen no
revoca las decisiones de arquitectura anteriores: constata que su proyección
persistible todavía no está cerrada. Las tablas y funciones de la sección
siguiente describen el destino previsto, no una migración ya aprobada.

Antes de fijar columnas, restricciones o firmas SQL deben resolverse
conjuntamente estas brechas contractuales:

- definir `RepositorioOperacionesBaremacion`, sus solicitudes, resultados y
  las siete acciones de autorización, incluido el camino exclusivo de
  reconciliación;
- completar `IntencionAltaBaremacionV1` y decidir la convivencia de
  identificadores V1 con las referencias opacas exigidas por V2;
- distinguir y vincular la HMAC de la huella semántica con el sobre
  probatorio exacto, sin tratarlos como un único valor intercambiable;
- fijar una única proyección persistible del índice versionado y armonizarla
  con `IdentificadorOperacionTransaccionalBaremacion`;
- cerrar la capacidad AEAD: suite, sobre, límites, AAD canónico, caducidad,
  revisión de token, revisión de reenvoltura, rotación y destrucción;
- completar recibo de retención, los tres recibos específicos de Bolsa,
  manifiesto material, restauración y verificadores productivos;
- definir el resultado canónico que enlaza operación, versión, auditoría y
  evento, así como el origen V2 de `version_baremacion`, que actualmente exige
  una `reserva_ref` V1;
- especificar consumo de autorización ante una consulta negativa, formato y
  límite de candidatos, orden de bloqueo y respuesta ante coincidencia
  múltiple;
- resolver cómo una caducidad incluida en el AAD puede proceder de PostgreSQL
  si el sobre se cifra antes de abrir la transacción;
- exigir o acreditar el aislamiento transaccional en la propia frontera
  durable, sin depender únicamente de que un cliente concreto solicite
  `SERIALIZABLE`.

### Secuencia propuesta para revisión

La siguiente partición reduce el radio de cada cambio; **no constituye todavía
una decisión adoptada**:

1. Cerrar los contratos Go, vectores canónicos y referencia adversaria en
   memoria, sin DDL.
2. Crear una migración inerte de identidad de operación con
   `operacion_idempotente_version`, `operacion_idempotente_actual` e
   `indice_operacion_idempotente`; solo incluiría restricciones, RLS, ACL y
   disparadores privados, sin funciones operativas ni concesiones.
3. Añadir en otra migración capacidad recuperable y consumo de autorización.
4. Añadir la barrera `decision_incorporada` y el enlace completo del resultado
   transaccional.
5. Incorporar las funciones cerradas y el adaptador PostgreSQL; mantenerlas
   sin tráfico hasta superar el corredor real.
6. Activar la composición V2 y retirar `EXECUTE` de V1 en una migración de
   activación independiente y reversible antes de recibir tráfico V2.

Los roles técnicos se aprovisionarían mediante migración DBA separada, nunca
desde el propietario `NOCREATEROLE`. La propuesta mínima es un grupo `NOLOGIN`
exclusivo para el proceso privado de idempotencia y, si se conserva una
superficie SQL propia, otro para el reconciliador. El ejecutor V1, el proceso
web y el lector outbox no recibirían funciones V2; el rol V2 solo recibiría
`CONNECT`, `USAGE` del esquema y `EXECUTE` de sus funciones definitivas. Incorporar
un rol obliga también a actualizar la guarda de ACL de tipos, la reversión de
roles y sus pruebas de membresía.

### Condiciones propuestas para revisar el primer DDL

De aprobarse la partición anterior, el primer corte deberá acreditar estas
invariantes antes de considerarse instalable:

- `ausente` no es una fila; una operación comienza `en_curso` y `confirmada`
  es terminal.
- La referencia, el índice y la intención quedan fijados antes del primer
  efecto y son inmutables.
- Existe exactamente un índice durable por operación y un índice completo no
  puede pertenecer a dos operaciones.
- El puntero actual solo avanza por revisión consecutiva y operación exacta de
  comparación y sustitución (CAS).
- Una operación confirmada referencia conjuntamente una versión, una
  auditoría y un evento coherentes; no se admite resultado parcial.
- La identidad estable no contiene sesión, autorización, correlación ni
  tiempos del intento.
- Cero o una coincidencia de candidatos es admisible; una coincidencia múltiple
  falla como evidencia no confiable.
- Todas las tablas son inmutables o monotónicas, tienen RLS forzado y carecen
  de acceso directo para identidades en ejecución.

La reversión del primer corte exigirá una confirmación explícita V2, bloqueará
únicamente sus relaciones en orden estable y abortará antes de mutar si existe una fila V2.
Una instalación V2 vacía podrá retirarse aunque exista historia V1; no se
vaciará, reinterpretará ni alterará esa historia. Los objetos se retirarán de
forma exacta con `RESTRICT` y los roles, mediante su reversión DBA separada.

Las pruebas mínimas de esta puerta son: instalación sobre V1 vacía y poblada;
catálogo exacto de restricciones, RLS y ACL; denegación con cuentas `LOGIN`
reales; rechazo de estados, revisiones, índices y punteros inválidos;
inmutabilidad frente a actualización, borrado y truncado; dos sesiones
compitiendo por el mismo índice con cardinalidad uno; y reversión sin
confirmación explícita, con historia V2, con historia solo V1 y con dependencias externas. Estas
pruebas estructurales no sustituyen el corredor Go → adaptador → PostgreSQL de
la sección de aceptación.

## Persistencia PostgreSQL V2

La migración `000005_idempotencia_semantica` creará, sin reinterpretar las
reservas V1:

- `operacion_idempotente_version` y `operacion_idempotente_actual`;
- `indice_operacion_idempotente`;
- `capacidad_reserva_version` y `capacidad_reserva_actual`;
- `uso_autorizacion_operacion`;
- `decision_incorporada`, con unicidad de `decision_ref` como segunda barrera.

Las funciones cerradas serán:

- `consultar_estado_operacion_v2`;
- `preparar_operacion_v2`;
- `reanudar_operacion_v2`;
- `rotar_capacidad_operacion_v2`;
- `confirmar_operacion_v2`;
- `recuperar_resultado_operacion_v2`.

Todas revalidan la decisión actual antes de revelar existencia, capacidad o
resultado. No comparan el intento actual con sesión, autorización, correlación
o tiempo originales. Los candidatos se bloquean en orden estable; cero o una
coincidencia es admisible. La confirmación une obligatoriamente operación,
versión, auditoría y evento. Los tiempos proceden de PostgreSQL y quedan fuera
de la intención.

Las tablas permanecen sin acceso directo, con RLS forzado y funciones
`SECURITY DEFINER` cerradas. El lector de eventos no puede leer capacidades.
V1 pierde `EXECUTE` al activar la nueva composición.

## Corredor de aceptación

La prueba obligatoria atraviesa caso de uso Go, adaptador y PostgreSQL real. No
se sustituye por llamadas SQL manuales. Debe demostrar:

- `COMMIT` aplicado seguido de `context.DeadlineExceeded`;
- recuperación desde una segunda instancia tras reinicio;
- sesión y autorización nuevas para el mismo actor;
- revocación antes de recuperar, sin revelar existencia ni resultado;
- dos reintentos concurrentes con cardinalidad uno;
- rotación de claves y capacidad;
- reutilización del índice con otra intención denegada;
- manifiesto o recibo alterado rechazado;
- una sola firma, custodia, retención, versión, auditoría y evento;
- igualdad byte a byte del resultado recuperado.

## Orden de implantación

1. Completar tipos de índice, intención y resultado indeterminado.
2. Crear recibos canónicos materiales V2 de almacenamiento.
3. Tipar y proteger la capacidad recuperable.
4. Crear el manifiesto probatorio material V2.
5. Definir el repositorio de operaciones V2.
6. Separar la finalización de la saga de la incorporación administrativa.
7. Implementar una referencia V2 en memoria con pruebas adversarias.
8. Añadir DDL, restricciones, ACL y reversión conservadora V2.
9. Implementar preparación, reanudación y rotación en PostgreSQL.
10. Implementar confirmación atómica y recuperación completa.
11. Conectar el adaptador Go V2.
12. Superar el corredor real de pérdida de respuesta y concurrencia.
13. Cambiar la composición y retirar `EXECUTE` de V1.

Hasta completar el último punto, V2 es un contrato de seguridad en
construcción y V1 continúa siendo exclusivamente experimental.
