# Ejecutor de checkpoints periódicos

Este paquete valida los límites técnicos del ejecutor. No publica la
periodicidad, una política de conservación ni permisos. El código requiere una
fuente durable con autoridad técnica propia; aquí no hay un adaptador instalado.
La conservación de auditoría por serie sigue pendiente, incluida la duda 84.

La configuración JSON acepta únicamente `version` (1), `max_registros`
(1 a 1.000.000), `version_binario`, `pin_spki_sha256` y `timeout_segundos`
(1 a 300). El pin procede de una fuente externa confiable. No admite DSN,
credenciales, actores, perfiles, periodicidad o política del sello. El timeout
es un límite de ejecución, no el intervalo entre capturas.

La composición debe seguir este orden:

1. Leer la configuración con `Decodificar` y abrir la fuente privada autorizada.
2. Aplicar `Ejecutor.Timeout()` al contexto y llamar una vez a
   `application.CapturarCheckpointPeriodico`.
3. Si el estado es `pendiente`, crear los proveedores existentes desde
   `captura.Checkpoint.Politica` y cotejar `firmador.PinCheckpoint()` con el pin
   externo de la configuración. Ante una discrepancia, detener la emisión.
4. Llamar a `application.SellarConfirmarCheckpointPeriodico` con la misma
   fuente y la captura recibida. Para `no_vencido` no se necesitan proveedores.
5. Entregar el recibo únicamente después de obtener `confirmado` y su acuse.

La fuente calcula la ventana y conserva la captura y su registro de auditoría
en una misma transacción. Sólo devuelve una captura tras confirmar COMMIT.
`pendiente` incluye el acuse original en la cabeza y última secuencia del
checkpoint. `no_vencido` incluye un acuse nuevo y no contiene una captura.
Confirmar conserva el recibo exacto con su auditoría y devuelve el acuse sólo
tras COMMIT. Su `ReciboHuellaSHA256` es SHA256 de `json.Marshal` del
`domain.ReciboCheckpointDesarrollo` completo, incluida `FirmaBase64`.

Ante un fallo no se recaptura ni se reintenta automáticamente. La fuente
conserva el pendiente original para su recuperación. Un fallo de confirmación
no devuelve un recibo como confirmado; la recuperación durable de la fuente
debe resolver un COMMIT desconocido. Las causas se conservan para `errors.Is`
y los mensajes del proveedor quedan redactados.

Los DTO de estas funciones son entradas de composición confiable. No deben
decodificarse de una petición o archivo del cliente. El ejecutor no recibe
coordenadas, registros personales o permisos libres. La configuración no
acredita que exista un servicio activo o una operación auditada.

Se reutilizan checkpoint v1 y los proveedores de desarrollo existentes. La TSA
HMAC de desarrollo no acredita tiempo independiente ni firma legal. El código
no borra registros ni habilita expurgo.

El [ENS, RD 311/2022, anexo II, op.exp.8](https://www.boe.es/buscar/act.php?id=BOE-A-2022-7191)
exige registros de actividad y su protección. La conservación documental tiene
su propio circuito: [Ley 7/2011, artículo 18](https://www.boe.es/buscar/act.php?id=BOE-A-2011-18654#a18),
[Comisión Andaluza de Valoración de Documentos](https://www.juntadeandalucia.es/organismos/culturapatrimoniohistoricoydeporte/areas/cultura/archivos/cavad.html)
y [Decreto 49/2025](https://ws040.juntadeandalucia.es/sedeboja/lconsolidada/eli/es-an/d/2025/02/24/49/dof/20250228/spa/html/LE0000948934_20250228.html).
Estas fuentes no fijan por sí solas una periodicidad o un plazo de conservación
para VEC.

Comprobación focal:

```sh
GOCACHE=$HOME/.cache/go-build go test -p 8 ./config/auditoriaperiodica ./internal/vec/application -run 'TestEjecutor|TestCheckpointPeriodico'
GOCACHE=$HOME/.cache/go-build go test -race -p 8 ./config/auditoriaperiodica ./internal/vec/application -run 'TestEjecutor|TestCheckpointPeriodico'
GOCACHE=$HOME/.cache/go-build go vet -p 8 ./config/auditoriaperiodica ./internal/vec/application ./internal/vec/ports
```

Las pruebas usan dobles unitarios. No acreditan SQL, autorización nominal,
instalación, cron o recuperación tras reinicio.
