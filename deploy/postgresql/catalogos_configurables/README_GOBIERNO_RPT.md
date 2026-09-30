# Gobierno de categorías RPT

Catálogos `000004` conserva una propuesta inmutable de publicación o deshabilitación, su contenido completo y SHA-256, fuente, motivo catalogado y preimágenes. AD3 `000134` conecta los actos con la autoridad V3. El rol técnico de gobierno está separado de los consumidores de usos; la migración no provisiona personas ni concesiones funcionales.

La propuesta empieza en revisión 1. Dos identidades distintas del editor aprueban la misma huella y llevan la revisión a 2 y 3. La confirmación exige revisión 3 y devuelve revisión 4. Quien confirma puede ser uno de los aprobadores. En una publicación, el documento fija al editor y al publicador y SQL coteja ambas identidades con V3. Cada acto, incluido recuperar un recibo, requiere una decisión nueva y positiva.

Antes de confirmar, AD3 recupera las dos atestaciones históricas por su decisión y revalida sus personas, perfiles, concesiones y confianza vigente con la autoridad existente. Las decisiones anteriores no se consumen otra vez. La transacción reúne el consumo y auditoría central V3, CAS del catálogo, historia, confirmación y outbox. Un fallo revierte el conjunto.

La deshabilitación bloquea reservas nuevas y conserva publicaciones, reservas y terminales anteriores. La fachada permite exclusivamente `publicar` y `deshabilitar`. Cobertura histórica, tombstone y rehabilitación quedan fuera de este contrato.

## Material y preparación

Las tres fachadas tienen la firma habitual: material JSONB y las diez piezas de consumo V3. El recurso es `propuesta_categoria`, la finalidad `gobernar_categorias_rpt`, los campos `gobierno` y `recibo`, sin obligaciones. Las acciones son `vec.catalogos.categorias.gobierno.proponer`, `.aprobar` y `.confirmar`, con audiencia `vec_catalogos_configurables.gobierno_categorias.v1`.

Proponer recibe `propuesta_ref`, `contenido`, `huella_sha256` y `recibo_ref`. El contenido tiene exactamente estos campos:

```text
accion, catalogo_id, modulo_id, version, documento_canonico,
documento_huella_sha256, preimagenes_control, preimagenes_huella_sha256,
categoria_id, revision_esperada, motivo_ref, fuente_ref
```

En publicación, categoría y revisión esperada son `null`. El documento conserva sus bytes UTF-8 canónicos. Las preimágenes incluyen exclusivamente las categorías anteriores incluidas en esa publicación. En deshabilitación, documento y su huella son `null`; hay una preimagen de la categoría, con `version`, `huella_sha256`, `revision` y estado `habilitada`.

Aprobar y confirmar reciben `propuesta_ref`, `huella_sha256`, `recibo_ref`, `revision_esperada`, `catalogo_id` y `modulo_id`. SQL coteja catálogo y módulo con la propuesta registrada. La preparación calcula `sha256(contenido::text)` y `sha256(material::text)` mediante SQL sin acceder a tablas. Go no reproduce la serialización de JSONB. El contexto canónico de recurso es:

```json
{"ambitos":{"catalogo_id":"catalogo-demo","modulo_id":"personal"},"atributos":{"material_sha256":"<SHA256 del material JSONB::text>"}}
```

## Ensayo y límites

El orden de las migraciones propias está en `deploy/principal/lista_sql_trabajo_codexm_rpt_gobierno_20260930.txt`. Requiere AD3-114 y la cadena Catálogos roles/1/2, AD3-117, Catálogos3 y AD3-126 en un clon nuevo y coherente. Las pruebas SQL revierten todos sus datos sintéticos.

`pruebas_sql/000004_gobierno_categorias_rpt.sql` comprueba doble identidad, huella cambiada, CAS, recibo idéntico al recuperar, fallo posterior al efecto, conservación del uso y denegación de una reserva posterior a deshabilitar. El fallo de persistencia por referencia de auditoría ausente prueba el rollback del catálogo; la comprobación de la auditoría central corresponde a AD3.

Pendientes de acreditar en el corte inicial: ensayo completo sobre la Cat1 corregida, atestaciones criptográficas V3 para tres identidades, revocación entre aprobación y confirmación, fallo de auditoría central y carrera real en dos conexiones entre reserva y deshabilitación. Las comprobaciones secuenciales de ambos órdenes no sustituyen ese ensayo de concurrencia. Dirección organiza las dos revisiones independientes del mismo hash antes de integrar.

No hay instalación en la principal, publicación, recorrido web ni permisos de datos reales acreditados por estos archivos. Las migraciones DOWN se reservan al clon sin hechos y rechazan la reversión si existe historia de gobierno.
