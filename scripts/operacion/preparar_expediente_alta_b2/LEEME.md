# Preparación del expediente de alta B2

El guion prepara un expediente ficticio hasta el informe jurídico y registra su vínculo con la RPT. Se detiene antes de la firma del informe. La firma, la fiscalización, el llamamiento y los pasos posteriores se hacen en las pantallas del caso único de `docs/producto/FLUJO_APP.md`, apartado 4.0. El guion no acredita por sí solo el recorrido ni el alta en Personal.

`preparar.sh --comprobar --material <directorio-privado> --estado <estado-privado.json>` revisa CT215, la fuente nativa B110/AD261, la bolsa Administrativo vigente `bolsa:administrativo:algicjlleggibhgagmjmoipmakdijibk`, los catálogos, la RPT publicada, la vacante de Personal y la configuración del cese. Las consultas HTTP pueden dejar su auditoría normal. La comprobación no envía actos de negocio. Consulte también el guion de activación en `deploy/principal/candidata_hz39/personal/LEEME.md`.

`preparar.sh --aplicar PREPARA-ALTA-B2 --material <directorio-privado> --estado <estado-privado.json>` registra la petición, la ratificación y las actuaciones de RRHH hasta el vínculo RPT. `--hasta alta` permite detenerse tras el alta. Cada POST guarda su cuerpo y clave antes de enviarlo; al reintentar usa los mismos datos. Mantenga el fichero de estado fuera de Git, con permisos 0600. Si la respuesta fue incierta, recupere primero la operación original con su clave.

La necesidad de la petición usa el contexto publicado por CT215. El JSON privado `necesidad_ejemplo_b2.json` aporta cinco datos del ejercicio: `plaza_codigo`, `organica_codigo`, `funcional_codigo`, `proyecto_gasto_codigo` y `porcentaje_financiacion`. El guion exige que la plaza coincida con el puesto 9/01 del centro 520 en la RPT publicada y con una vacante de Personal sin ocupación. Si falta el fichero, `--comprobar` muestra un comando para crearlo. El ejemplo de plaza 0033 no demuestra por sí mismo que esté vacante.

Se admite `VEC_B2_NECESIDAD_EJEMPLO` para indicar la ruta del JSON, y `VEC_B2_MATERIAL` y `VEC_B2_ESTADO` para evitar repetir las dos rutas privadas. El guion también acepta `--origen`, `--launcher`, `--config`, `--motor` y `--contenedor`. Use el `servidor.json` del lanzador activo si hay varias instalaciones de Personal.

Los antiguos `fixture_firma.sql` y `fixture_anclaje.sql` se han retirado. Si un estado anterior indica que los usó, el guion se detiene para revisión administrativa. No borre ese estado ni fabrique recibos para continuar.
