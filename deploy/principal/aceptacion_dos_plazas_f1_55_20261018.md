# Dos plazas con aceptación previa

Tarea F1-55, en preparación. Dirección fijó el alcance el 18/10: una oferta de
dos plazas, aceptación previa de dos personas, adjudicación según el orden de
la bolsa y recuperación tras un fallo y un reinicio. La retirada por cancelación
queda fuera, en F1-17. No responder a la oferta no es un acto ni causa una baja.

## Preparación de la copia

Usar `deploy/principal/copia_local.py` de la principal con `--modulos`, la lista
común completa y el kit de preparación privado de la copia. Los certificados,
conexiones y datos son propios. No se instalan nuevas migraciones para este corte.

Reutilizar dos candidaturas de la misma bolsa constituida, con participaciones
distintas, disponibles y sin vínculo vigente con otra persona. El preparador
`preparar_candidato_desarrollo.sh` conserva las comprobaciones de vigencia,
pertenencia al acta y coincidencia de la proyección antes de confirmarla. La
preparación de datos de copia no acredita un alta administrativa.

Cada aspirante usa su directorio privado de material y su propio certificado,
cuenta, persona, perfil y vínculo. La CA y los materiales técnicos proceden de
la propia copia. El preparador recibe la identidad, el nombre visible y la
referencia existente que se selecciona, y emite certificados de 90 días. La
configuración existente `VEC_DEVELOPMENT_MATERIAL_DIR` selecciona el directorio
activo; cambiarlo exige parar la app, conservar la configuración anterior,
reiniciar y comprobar salud y consulta propia. La copia no necesita atender
simultáneamente a ambos aspirantes.

Antes de actuar, comprobar desde Mi Bolsa que cada certificado muestra su propia
participación. Un certificado no sustituye a otro, ni una referencia del
formulario identifica a la persona. Las denegaciones conservan sus comprobaciones.

## Recorrido que se debe comprobar

1. RRHH publica desde «Gestionar ofertas» una oferta de dos plazas con una
   política vigente de aceptación previa. Guarda el recibo y el vencimiento.
2. La segunda persona en el orden de la bolsa acepta desde Mi Bolsa. Después
   acepta la primera. Así se comprueba que la adjudicación sigue el orden de
   la bolsa y no el de llegada de las respuestas. Cada aceptación deja recibo.
3. Al terminar el plazo, RRHH ve una propuesta por plaza y confirma las dos
   adjudicaciones. La oferta conserva su recibo y cada acto tiene el suyo.
4. Se pierde deliberadamente una respuesta ya guardada. La misma intención
   recupera el recibo, sin otro acto ni otra adjudicación.
5. Tras reiniciar aplicación y PostgreSQL, se recuperan oferta, aceptaciones y
   adjudicaciones con las mismas referencias y fechas, sin duplicados.

La preparación y las comprobaciones están pendientes. No se acredita todavía
el recorrido de dos aceptaciones, adjudicación o recuperación de esta tarea.
El ensayo de correo de F1-18 tiene su guía y no sustituye estos pasos.
