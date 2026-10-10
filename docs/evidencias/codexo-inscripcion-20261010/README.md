# Lecturas de inscripción B99: comprobación visual

Las capturas muestran el selector de convocatorias RRHH a 1440 y 390 píxeles
en Chrome del sistema. La pantalla usó los CSS del portal y
[`selector-en.json`](selector-en.json), proyección de B99 obtenida en
PostgreSQL 18 desechable con 2.000 solicitudes de prueba. La convocatoria
«Administrative assistant pool 2026» tenía 1.000 pendientes y
«Maintenance staff pool 2026», cero. El navegador estaba en inglés y los
títulos y categorías procedían del resolutor sintético de metadatos en ese
idioma. El catálogo español de la interfaz tiene las mismas claves.

La tarjeta entera enlaza a la lista de solicitudes pendientes de su
convocatoria. En ambos anchos el documento tuvo el mismo ancho que la ventana,
sin desbordamiento horizontal. La captura de 390 píxeles muestra las tarjetas
apiladas y el recuento legible.

El HTML temporal montó la vista con esa respuesta JSON de PostgreSQL y se eliminó
después. No hubo API VEC, sesión nominal, auditoría de lectura ni recorrido
completo de inscripción en el navegador. Esos pasos siguen pendientes de la
integración de B96/B99 y la publicación gobernada de los permisos.
