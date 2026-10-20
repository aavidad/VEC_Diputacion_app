# F1-127. Propuesta tras una aceptación en Mi Bolsa

RRHH resuelve la aceptación firme desde el llamamiento y conserva el recibo.
La propuesta de formalización utiliza esa resolución, sin pedir un correo.
La vía de respuesta por correo conserva su tramitación anterior.

## Preparación

1. Crear una copia local con PostgreSQL 18 mediante la orden documentada en
   `deploy/principal/LEEME_copia_local.md`, con datos ficticios propios.
2. Instalar la candidata completa en el orden de su lista SQL. CT249 debe
   preceder a CT253; no reaplicar migraciones instaladas.
3. Mantener las publicaciones vigentes del tipo de formalización, plantilla,
   política de firma y plan de firma. La propuesta no crea esas publicaciones.
4. Preparar por los actos de la aplicación una petición fiscalizada, su
   llamamiento singular y una aceptación firme de la persona en Mi Bolsa.
5. Con certificado y perfil de RRHH autorizado para esa organización, resolver
   la aceptación. Guardar la referencia de resolución y su recibo inmediato.

## Comprobación

Desde la ficha, registrar la propuesta usando esa pareja de referencias.
Debe responder 201, conservar el recibo y avanzar el expediente a Nombramiento.
Repetir la misma solicitud con la misma clave debe recuperar el mismo recibo,
fecha y versión, sin otra propuesta ni otro avance de expediente.

Comprobar con un perfil autorizado para otro ámbito que la petición se deniega
con 403 y no produce efectos. Reiniciar aplicación y PostgreSQL y recuperar
la propuesta con el mismo recibo. Conservar también un caso de correo anterior.

Comprobar las dos representaciones originales de conservación de la selección
que admite O6, con su recibo, artefacto y fechas. La referencia de conservación
no acredita una reserva de crédito ni añade una caducidad al acto.

## Configuración y límites

No se incorpora una variable de configuración ni un permiso nuevo. Sin las
fuentes nativas de aceptación, esta operación falla cerrada; el arranque y
las demás funciones conservan su comportamiento. Comprobar el arranque con
las fuentes nativas presentes y ausentes, y guardar los resultados.

Este documento es el guion de aceptación; no acredita su ejecución. La
activación queda pendiente del ensayo conjunto con la resolución productora.
