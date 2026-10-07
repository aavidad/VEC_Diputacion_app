# Contrato pendiente de persistencia de Provisión

Este directorio contiene el contrato de preparación del 1 de octubre de 2026.
No contiene SQL ejecutable, migraciones instalables ni concesiones nuevas.
Dirección ha reservado `provision000001` en el registro externo a Git; la
reserva no acredita que exista una migración ni que pueda instalarse.

## Orden de las dependencias

1. Cerrar H6 y la dependencia del núcleo común que Dirección ha situado antes
   de este corte. Mantener la cola D vigente.
2. Aprobar las bases, reglas y causas exactas del proceso y enlazar su versión
   con las referencias de solicitud, puesto ofertado y RPT publicada.
3. Resolver las fuentes Personal y RUM mediante sus puertos autorizados. Una
   rectificación añade otra instantánea: no modifica el historial de esas
   autoridades ni una valoración publicada.
4. Implementar el repositorio de Provisión y su composición con autorización
   central, auditoría y servicios documentales existentes.
5. Revisar la candidata SQL exacta por dos revisores independientes y ensayarla
   en el clon de la principal antes de integrar. Verificar después recorrido,
   concurrencia, reintento y recuperación tras reinicio.

## Operación durable que se deberá acreditar

El borrador de puerto `RepositorioCicloProvision`, en
`internal/modules/provision/ports/ciclo_persistencia.go`, separa añadir una
revisión y recuperar su recibo. Sus referencias de concesión son opacas. El
adaptador deberá resolverlas y revalidarlas con la autoridad central; recibir
una cadena no concede permiso.

La escritura deberá comprobar proceso y versión exactos, solicitud, puesto,
fuentes, valoración reclamada, huellas y catálogo de causas. En la misma
transacción deberá consumir autorización vigente, comprobar `version_esperada`
y clave de idempotencia y conservar la nueva valoración, la decisión motivada,
su recibo, historia, auditoría y outbox. Las versiones y publicaciones previas
se mantienen.

Un reintento con idéntica clave y material deberá devolver el recibo original.
Otra huella con esa clave, una versión obsoleta o una referencia cruzada deberá
producir conflicto sin efectos parciales. Recuperar el recibo requiere una
concesión nueva de lectura. Habrá que comprobar también revocación, cancelación
y fallo entre cálculo y confirmación.

La implementación futura deberá definir propietarios y roles técnicos mínimos,
revocar privilegios de `PUBLIC`, impedir acceso directo de ejecución a tablas y
aplicar las guardas PostgreSQL del núcleo. Este documento no crea roles ni
permite reutilizar funciones documentales con acciones distintas de las que
autorizan.

## Límite actual

`application.EnsayarCiclo` y `cmd/vec-ensayar-provision` calculan datos de ensayo.
No llaman a este repositorio ni a fuentes institucionales. No existe un
adaptador de memoria que se presente como guardado real. Sus huellas permiten
reproducir y comparar revisiones; no sustituyen firma, custodia ni auditoría.
Firma de resolución, publicación y efectos sobre Personal conservan puertos
separados y pendientes.
