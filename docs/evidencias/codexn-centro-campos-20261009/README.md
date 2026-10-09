# Petición del centro: personas, jornada y puesto

La petición recoge por separado el número de personas, la jornada semanal en horas y minutos y el puesto solicitado. La persona ratificadora y RRHH ven los tres datos antes de actuar. El alta de RRHH los conserva en `expediente.solicitud` sin convertirlos en una descripción libre.

La [circular de peticiones de 8 de mayo de 2026](https://www.dipgra.es/export/sites/diputaciongranada/diputacion/delegaciones/transparencia-recursos-humanos-y-administracion-electronica/.galleries/DIPUTACION-Delegaciones-Galerias-Normativa-RRHH/CIRCULAR-PETICIONES-DE-PERSONAL_2026.report.pdf) pide número de personas y, para vacantes, código de plaza y puesto. No fija la unidad de jornada. El formulario propone horas y minutos semanales con la referencia del catálogo de necesidades ya configurado (`jornada_referencia_minutos`; en el ensayo, 2250 minutos). La duda 154 de `dudas.md` pide a RRHH confirmar la unidad y la fuente por ámbito; el valor de ejemplo no es una aprobación formal.

## Recorrido local verificable

Se usó PostgreSQL 18 en un contenedor desechable limitado a 2 GB, con datos sintéticos y la base causal HZ15. En Chrome del sistema, el centro presentó la petición (`POST 200`), otra identidad la ratificó (`POST 200`) y RRHH creó el expediente (`POST 201`). La base conservó dos revisiones de la petición, una entrega confirmada y un expediente v1 cuyos campos fueron `numero_personas=2`, `jornada_minutos=2250` y `puesto_solicitado="Administrativo C1, puesto de apoyo del centro"`. Tras reiniciar aplicación y PostgreSQL, una lectura RRHH `GET 200` recuperó esos datos sin repetir los POST.

El primer ensayo aislado detectó `POST 403` en la entrega después de que el alta original quedase registrada: el constructor canónico de Go aún omitía los tres campos. Se corrigieron sus variantes v2/v3 y se repitió el recorrido en una copia limpia. Aquel `403` no se cuenta como éxito ni como parte del recorrido final.

CT200 se reserva y se instala una sola vez según la [lista causal](../../../deploy/principal/lista_sql_codexn_centro_campos_20261009.txt). Su SHA256 es `adf75fea28a7a59bd3b78fa83c43d11786d42b151293c64f437f233d5534d41e`. La migración y la prueba SQL focal pasaron en PostgreSQL 18. Los 71 efectos de alta previos reconstruyeron los mismos bytes canónicos antes y después de CT200. Dos revisiones SQL independientes dieron GO a la primera versión de CT200, y una revisión de usabilidad dio GO a los mensajes finales. Una tercera revisión SQL encontró que esa versión daba al ejecutor permiso sobre `confirmar_alta_atestada_v1`, que desde CT47/CT48 solo puede usar el propietario. La versión actual deja las ACL como estaban antes de CT200 y lo comprueba al instalar; además rechaza una presentación nueva sin los tres datos antes de consumir la autorización. Se repitió el ensayo en PostgreSQL 18.4 desechable sobre la base sintética con HZ9 a HZ12: migración y prueba OK, los mismos 71 efectos, ACL iguales a las previas y una segunda instalación se detiene limpia. No se aplicó SQL en cidonia.

Las pruebas focales Go y los escaneos locales Semgrep/gosec no encontraron hallazgos en líneas modificadas. La puerta integral anterior terminó con tres fallos Node de i18n/caché; Go normal, race, vet y build pasaron. Tras corregir el alcance del catálogo Centro y renovar una sola cohorte de URL, la suite Node completa pasó **3678/3678**. También pasaron las guardas de i18n, dependencias, manifiestos, tamaños y `git diff --check`. El GET de la bandeja RRHH tardó 99 y 170 ms en dos lecturas locales de Chrome; son muestras individuales, no un percentil.

## Capturas finales

| Idioma y ancho | Contenido | Archivo |
| --- | --- | --- |
| ES, 1440 px | Valores del formulario | [es-1440-valores.webp](es-1440-valores.webp) |
| ES, 390 px | Errores de jornada y puesto | [es-390-errores.webp](es-390-errores.webp) |
| EN, 1440 px | Valores del formulario | [en-1440-valores.webp](en-1440-valores.webp) |
| EN, 390 px | Errores de jornada y puesto | [en-390-errores.webp](en-390-errores.webp) |

Son capturas reales del Chrome local, convertidas a WebP **sin pérdida**: la comparación de píxeles con cada PNG original dio diferencia cero. Cada archivo ocupa menos de 200 kB. En ambos idiomas se pidió solo el catálogo de peticiones de la pantalla abierta; los errores no generaron POST ni revisión. No hubo errores JavaScript ni desbordamiento horizontal en 1440 o 390 px.
