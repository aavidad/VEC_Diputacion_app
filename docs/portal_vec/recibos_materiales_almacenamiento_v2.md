# Recibos materiales de almacenamiento V2

## Estado y alcance

Decisión de arquitectura adoptada el 15 de julio de 2026. Su implementación
está pendiente y es un requisito de la migración de baremación a idempotencia
semántica V2.

Los recibos V1 continúan siendo válidos para autorización y auditoría del
intento que los produjo. No se reinterpretan como evidencia material estable.
V2 crea tipos, fábricas, persistencia y verificadores nuevos.

El contrato se divide en dos capas:

- el núcleo común prueba efectos técnicos sobre objetos sin conocer bolsas,
  decisiones ni firmas;
- Bolsa encadena esos efectos con la recuperación del PDF firmado y el plan
  administrativo durable.

## Por qué V1 no es reutilizable

`EvidenciaOperacionAlmacen` mezcla el resultado técnico con autorización,
correlación, decisión vigente, plan del intento y HMAC de solicitud. Una
respuesta idempotente de S3 puede conservar el objeto y, a la vez, emitir otra
referencia y otra hora de evidencia. La retención tampoco dispone todavía de
una clave idempotente y un diario durable recuperable.

Por tanto, hashear una respuesta V1 produciría intenciones distintas para el
mismo efecto material. Tampoco es válido sustituir el recibo por la huella del
PDF: contenido, recuperación, custodia y retención son dominios probatorios
distintos.

El manifiesto probatorio V1 incorpora además sujeto, autorizaciones e instante
del intento. No se promoverá a manifiesto material V2.

## Contratos del núcleo común

Se crearán tipos nominales independientes en `internal/vec/ports`:

- `InstantaneaObjetoMaterialV2`;
- `PerfilCapacidadesAlmacenMaterialV2`;
- `ReciboEscrituraObjetoMaterialV2`;
- `ReciboRetencionObjetoMaterialV2`;
- fábricas y verificadores criptográficos de cada recibo.

El contrato nunca expone operaciones de listado ni detalles del soporte
físico. Un conector puede usar MinIO, S3, una cabina, un gestor documental,
Oracle u otro almacén sin modificar el núcleo.

### Recibo de escritura

La proyección canónica incluye exclusivamente:

1. Esquema fijo y versión 2.
2. Referencia durable del recibo original.
3. Identidad lógica del conector.
4. Referencia, versión y huella del perfil atestado de capacidades.
5. Módulo, acción de negocio y acción técnica estables.
6. Recurso, operación, carga y efecto materiales.
7. Huella del plan material, separada del plan de autorización V1.
8. Clasificación.
9. Referencia y versión opacas del objeto.
10. Zona semántica.
11. MIME, tamaño y SHA-256 de los bytes.
12. Referencia de creación original.
13. Instante original de almacenamiento.
14. Retención inicial, cuando exista.
15. Estado nominal de inmovilización.
16. Estado nominal `activo`.

En un reintento, la fábrica valida primero la escritura completa y usa la
referencia e instante originales del objeto. Excluye la nueva evidencia de
respuesta y su indicador de reintento. La misma escritura debe producir los
mismos bytes canónicos y la misma huella material.

### Recibo de retención

La retención incorpora:

- la identidad estable de operación y objeto;
- la huella de la instantánea anterior o del recibo de custodia;
- la retención anterior;
- referencia, versión y huella de la política;
- plazo exacto solicitado y aplicado;
- estado final de inmovilización y estado `activo`;
- referencia e instante originales del efecto de retención;
- referencia y huella del recibo anterior.

El puerto de almacenamiento deberá ofrecer una operación equivalente a
`AplicarORecuperarRetencion`, con clave de efecto estable y diario durable. Si
el backend muestra el plazo aplicado pero no existe el asiento probatorio, el
resultado es indeterminado: no se fabrica retrospectivamente un recibo.

Retención temporal e inmovilización legal son estados diferentes. Declarar la
capacidad de bloqueo legal no acredita que esté aplicada. Si una política
exige inmovilización, se añade un paso y recibo específicos.

## Contratos específicos de Bolsa

Se crearán en `internal/modules/bolsa/ports`:

- `ReciboRecuperacionFirmadoBaremacionV2`;
- `ReciboCustodiaFirmadoBaremacionV2`;
- `ReciboRetencionFirmadoBaremacionV2`;
- `ManifiestoMaterialBaremacionV2`;
- verificadores materiales de la cadena completa.

### Recuperación del documento firmado

El recibo de recuperación liga:

- conector lógico de firma;
- efecto y plan durable;
- proceso, solicitud, baremación y decisión mediante referencias opacas;
- referencia, versión y huella del documento firmable;
- referencia y huella de la firma;
- referencia, MIME, tamaño y huella del documento firmado;
- referencia durable del recibo;
- estado cerrado `verificado_completo`;
- instante durable original, si el proveedor lo acredita.

Solo se emite tras consumir el flujo completo y comprobar límite, tamaño,
SHA-256, fin exacto y cierre correcto. Sesión, URL, token, firmante y perfil
del intento quedan fuera.

### Encadenamiento probatorio

La cadena se construye en este orden:

1. Recuperación V2 prueba los bytes firmados obtenidos.
2. Custodia V2 incorpora la referencia y huella de recuperación junto con la
   referencia y huella del recibo técnico de escritura.
3. Retención V2 incorpora la referencia y huella de custodia junto con el
   recibo técnico de retención.
4. El manifiesto material incorpora las tres huellas distintas.

Siempre deben ser dominios diferentes:

- SHA-256 del PDF;
- SHA-256 del recibo de recuperación;
- SHA-256 del recibo de custodia;
- SHA-256 del recibo de retención.

Una coincidencia entre cualquiera de esas huellas invalida la intención.

## Proyección incorporada a la intención

La intención de cambio de baremación conserva:

- conector lógico;
- objeto y versión;
- zona;
- huella del objeto, igual a la del PDF;
- clasificación, MIME y tamaño;
- estado nominal de inmovilización y estado `activo/no_eliminado`;
- esquema, versión, referencia y huella de cada recibo;
- referencia, versión y huella de la política de retención;
- plazo de retención;
- plan durable de recuperación, custodia y retención;
- prueba de que el recibo de retención encadena el de custodia.

Los instantes originales pueden permanecer comprometidos dentro de cada
recibo; no se duplican en la intención.

## Referencias opacas e independencia del backend

El puerto solo admite:

- `ConectorLogicoID`;
- `ReferenciaObjeto` y `VersionObjeto`;
- zona semántica;
- política y plazo de retención;
- estado de inmovilización;
- perfil atestado de capacidades.

Las referencias son alias lógicos. El validador V2 rechaza `/`, `\\`, `..`,
`://`, `?` y `#`. Nunca atraviesan el puerto bucket, clave física, ruta, URL,
endpoint, ARN, ETag, identificador de carga, referencia KMS ni credencial.

El versionado se acredita mediante referencia y versión exactas, capacidad
atestada `Versionado=true` y una verificación posterior del mismo objeto. La
inmovilización se acredita con un estado nominal y recibo atestado, no con un
booleano implícito.

## Exclusiones de toda preimagen material

Quedan prohibidos:

- contexto, principal, perfil, autenticación, autorización y sesión;
- correlación y HMAC de solicitud;
- huella de la concesión vigente y plan de autorización V1;
- clave idempotente en claro;
- referencia u hora fabricadas al responder un reintento;
- auditoría, outbox y token de reserva;
- DNI, nombre, correo y nombre original del fichero;
- cualquier detalle físico o secreto del almacén.

## Canonización y autenticidad

Los recibos usan TLV binario: etiqueta `uint16`, longitud `uint64` y orden de
bytes big-endian. La etiqueta 0 contiene el esquema. Los enteros tienen tamaño
fijo y las huellas se codifican como 32 bytes, no como texto hexadecimal.

Los tiempos son UTC con precisión exacta de microsegundo y se codifican como
`UnixMicro`. Las zonas distintas de UTC, los submicrosegundos, los campos o
versiones desconocidos y los valores cero implícitos fallan cerrados.

El payload canónico produce una huella SHA-256 que no se incluye a sí misma.
Además se exige HMAC versionada o una atestación COSE verificable: la huella
sola aporta integridad, no autenticidad ni procedencia.

## Corredor mínimo de aceptación

Antes de habilitar V2 se debe demostrar:

- vectores dorados de bytes y SHA-256 para los tres recibos;
- mutación individual de cada campo y cobertura por reflexión;
- dos autorizaciones y sesiones distintas producen el mismo recibo material;
- escritura original y respuesta idempotente producen la misma huella;
- pérdida de respuesta y reinicio recuperan el mismo recibo;
- retención visible sin diario durable queda indeterminada;
- PDF corto, largo, alterado, truncado o con cierre erróneo no produce recibo;
- cualquier cambio de objeto, versión, política, plazo o inmovilización
  invalida la cadena;
- las cuatro huellas de contenido y recibos son siempre distintas;
- los bytes no contienen datos personales, rutas ni contexto del intento;
- atestación manipulada, clave desconocida o recibo cruzado fallan cerrados.

Hasta superar este corredor, los campos V2 ya reservados en la intención son
puertas de seguridad sin productor y el flujo completo permanece en NO-GO.
