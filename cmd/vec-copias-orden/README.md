# Órdenes de copia: contrato y ensayo CS08

La CLI verifica una orden sintética con sobre autenticado, acepta una vez y recupera
el mismo recibo al reenviarla. No abre red, PostgreSQL ni servicios de plataforma.
La aceptación de la CLI vive en memoria y desaparece al terminar. Copie
`config.ejemplo.json` a un directorio privado, fije referencia y versión de clave,
y señale el material maestro sintético existente de 32 bytes. El archivo de
configuración y el material requieren permisos privados. No se generan claves
al arrancar. Nunca use material de producción para este ensayo.

La CLI usa únicamente el ejemplo que recibe. No resuelve la identidad de quien
lo ejecuta ni concede permisos ADMIN. Las dos Personas, la política, la decisión
V3 y el instante del ejemplo no son observaciones de autoridades reales. Tampoco
sustituye una base, un archivo o un servicio, ni cambia la política vigente.

```bash
go run ./cmd/vec-copias-orden \
  --textos web/static/textos/es/copias_orden.json \
  --config /RUTA_PRIVADA/orden-config.json \
  < cmd/vec-copias-orden/testdata/orden.sintetica.json
```

El resultado indica `consumo_v3_acreditado: false`,
`aceptacion_durable: false` y `efecto_plataforma: false`. El último paso tiene
`replay: true` y conserva el recibo del anterior. El sobre utiliza material privado del proveedor de desarrollo; no es firma jurídica ni la autoridad del PDP.

La orden conserva operación, conjunto, manifiesto, preimagen, destino, dos Personas
canónicas distintas, política y huella, ventana, época, cercado y versión CAS.
También compromete la decisión V3, el consumo, la auditoría y el outbox. Las
referencias de la entrada sintética son declaraciones; su formato no les concede
validez ni acredita aprobaciones reales.

El PDP compromete `PlanBytes`/`PlanSHA256`, que contiene los datos operativos y
excluye los recibos de autoridad. Después del consumo se autentica la orden completa.
Así la huella de la decisión no depende de una orden que contiene esa misma huella.

## Adaptadores preparados

`adapters/ordenescopias/ProtectorOrden` reutiliza `Protector` de CS03 para un sobre
JWE autenticado. La composición debe obtener del KMS existente una clave exclusiva
de órdenes y su versión fija; la clave de componentes de copia no sirve para este
uso. La orden, su operación, huella y cercado quedan ligados al contenido y al AAD.
No se introduce un proveedor criptográfico ni se solicita una clave al cliente.

`RegistroExterno` envía la orden ya verificada al diario CS07, fuera de las raíces
restauradas. Exige una operación previamente reservada con solicitud y destino
idénticos. CS07 conserva aceptación única, cercado y recibo mediante bloqueo y fsync.
La prueba focal usa JWE real con material sintético y recupera el recibo tras reabrir
el diario. No demuestra un reinicio del servidor ni una restauración.

`adapters/ordenescopias/postgres/Registro` implementa el puerto privado
`ConsumidorV3` y la relectura del compromiso. Recibe material V3 de un
`MaterializadorV3` de la autoridad común, coteja la representación canónica de la
decisión e invoca la función SQL. El candidato registrado por
`AlmacenAutorizacion.RegistrarConcesionCandidata…` no concede permiso por sí solo.

`Servicio.Publicar` relee los bytes comprometidos y coteja su huella antes de
sellarlos. `Verificar` y `Aceptar` exigen un sobre auténtico, ventana vigente
y `Anclaje.ValidarActual`. El anclaje debe cotejar la autoridad externa actual,
revocaciones, época y cercado. Tras restaurar, bloquea órdenes nuevas hasta conciliar
esa autoridad; una delegación de mantenimiento debe ser específica y caducar.

La aceptación externa se conserva antes de detener PostgreSQL. La ejecución de
plataforma queda a cargo de CS07/CS11 y no se repite por recibir la misma orden.
No existe una transacción que abarque SQL, volúmenes y servicios.

## Condición de COMMIT y auditoría V3

El adaptador PostgreSQL está preparado, sin montaje ni funciones SQL incluidas
en esta entrega. Sólo puede confirmar una orden cuando la autoridad común haya
revalidado la identidad, la sesión, las Personas, la política, las aprobaciones
y la concesión V3 exactas y vigentes. El consumo único, el CAS, los bytes de la
orden, la auditoría V3 y el outbox deben quedar juntos en una transacción que
termine con COMMIT confirmado. Una respuesta parcial o un fallo de COMMIT no
acreditan el compromiso. La publicación debe leer después esa misma orden
comprometida, sin reconstruirla desde la petición ni aceptar una lectura pendiente
de COMMIT.

La confirmación de registro de una concesión candidata no es el consumo V3 de
la orden. El registro externo CS07 tampoco sustituye la auditoría común segregada:
su actor declarado y su cadena SHA256 no acreditan identidad nominal ni protección
frente a quien puede reescribir el diario. Antes de un efecto operativo, la
composición debe conectar las autoridades comunes y verificar también la época,
el cercado y la política actuales mediante el anclaje inyectado. No se crean
perfiles, permisos o políticas desde esta CLI.

## Dependencias pendientes

El borrador SQL del corte `f47006358` se conserva en su fuente y queda fuera de
esta entrega. No está instalado ni se recupera su migración. AD143 sigue siendo
la reserva exclusiva de copias de K; esta referencia no acredita que su ABI esté
implementada o instalada. La acción propuesta es `administracion.copias.orden.emitir`, recurso
`orden_copia`, finalidad `emitir_orden_copia` y campos exactos `orden`/`recibo`.
Debe acordarla la autoridad común; no provisiona concesiones funcionales.

Faltan el consumidor SQL y el materializador de la autoridad común, el anclaje
operativo y la auditoría V3 segregada. La implementación SQL debe respetar la
reserva AD143 de K y requiere ensayo PostgreSQL aislado y dos revisiones independientes
del hash final antes de cualquier instalación. La ausencia de las funciones
`comprometer_orden_v1` o `leer_orden_comprometida_v1` devuelve error; no se sustituye
por el entorno sintético.
No hay API ADMIN montada, calendario, retención ni copia o restauración operativa
acreditados por esta pieza.

## Comprobaciones históricas del corte f47006358

Pruebas focales, race, vet, gosec sobre los paquetes propios y `gopls check`
terminaron con código 0. Cubren alteración de compromisos/sobre, clave distinta,
caducidad, anclaje revocado, compromiso distinto, 32 aceptaciones concurrentes,
configuración privada, límites JSON y recuperación del diario CS07 al reabrirlo.
Semgrep local recorrió los ocho archivos Go de implementación con cuatro reglas
aplicables y no encontró hallazgos. Estos controles no son una auditoría completa.

Se ejecutaron sin red, con fuente y herramientas de solo lectura, entorno mínimo
y límites de CPU, memoria, procesos, archivo y tiempo. La caché Go compartida agotó
su cuota durante la comprobación final; la repetición usó una caché propia temporal.
No se limpiaron cachés ajenas ni se ejecutó SQL contra una base de datos.

Estas comprobaciones pertenecen al corte original. La recuperación de octubre
mantiene la lógica Go y los catálogos; aclara aquí y en el puerto el alcance
y los contratos.
La instalación SQL, el ensayo PostgreSQL y las revisiones sensibles de esta entrega
siguen pendientes. La puerta global y la integración pertenecen a Dirección.

## Comprobación de la recuperación del 4 de octubre de 2026

Sobre `origin/main@92aedcc59`, Go 1.26.6 local completó la prueba focal existente
de la CLI, aplicación y adaptadores con `-p 8`, límite externo de 180 segundos
y límite de prueba de 90 segundos. Dominio, puertos y adaptadores PostgreSQL y
sintético compilaron sin pruebas propias. No se añadieron pruebas ni se ejecutó SQL.
`gopls check` de los ocho archivos de implementación y gosec focal terminaron
con código 0. Semgrep local aplicó 42 reglas Go a esos ocho archivos, sin
hallazgos, con métricas desactivadas. No se repitieron race, vet ni la campaña
global histórica. La revisión de reconciliación sensible sigue pendiente.
