# Gobierno de categorías RPT

Catálogos `000004` conserva una propuesta inmutable de publicación o deshabilitación, su contenido completo y SHA-256, fuente, motivo catalogado y preimágenes. AD3 `000134` conecta los actos con la autoridad V3. El rol técnico de gobierno está separado de los consumidores de usos; la migración no provisiona personas ni concesiones funcionales.

La propuesta empieza en revisión 1. Dos identidades distintas entre sí aprueban la misma huella y llevan la revisión a 2 y 3. La confirmación exige revisión 3 y devuelve revisión 4. El editor puede aportar una de las aprobaciones. Quien confirma debe ser distinto del editor y puede ser el otro aprobador. En una publicación, el documento fija al editor y al publicador y SQL coteja ambas identidades con V3. Cada acto, incluido recuperar un recibo, requiere una decisión nueva y positiva. El publicador se elige al preparar la propuesta. Cambiarlo exige una propuesta nueva y repetir ambas aprobaciones; recuperar el recibo conserva al confirmador original.

Antes de confirmar, AD3 recupera las dos atestaciones históricas por su decisión y acredita que cada aprobación quedó consumida y auditada con la misma propuesta y huella. La autorización se comprueba al emitir cada aprobación. Su sobre breve puede caducar después sin borrar ese acto histórico; tampoco una revocación posterior del perfil retira retroactivamente la aprobación. Confirmar exige una autorización V3 nueva y vigente del confirmador. Las decisiones anteriores no se consumen otra vez. Anular una aprobación ya emitida necesitaría otro acto versionado, pendiente de especificación por RRHH. La transacción reúne el consumo y auditoría central V3, CAS del catálogo, historia, confirmación y outbox. Un fallo revierte el conjunto.

La deshabilitación coteja el módulo contra la publicación inmutable de la categoría y bloquea reservas nuevas y conserva publicaciones, reservas y terminales anteriores. La fachada permite exclusivamente `publicar` y `deshabilitar`. Cobertura histórica, tombstone y rehabilitación quedan fuera de este contrato.

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

`pruebas_sql/000004_gobierno_categorias_rpt.sql` comprueba doble identidad, huella cambiada, módulo ajeno a la publicación, CAS, recibo idéntico al recuperar, fallo posterior al efecto, conservación del uso y denegación de una reserva posterior a deshabilitar. El fallo de persistencia por referencia de auditoría ausente prueba el rollback del catálogo; la comprobación de la auditoría central corresponde a AD3.

Pendientes de acreditar en el corte inicial: ensayo completo sobre la Cat1 corregida, atestaciones criptográficas V3 para tres identidades, caducidad del sobre de las aprobaciones manteniendo su efecto histórico, revocación del confirmador antes de su acto, fallo de auditoría central y el recorrido completo con atestaciones V3 reales. La carrera real en dos conexiones se ensayó en ambos órdenes sobre el core de persistencia, observando el bloqueo: una reserva anterior se conserva y una posterior se deniega sin fila. Esta comprobación usa identidades sintéticas del core y no acredita V3. Dirección organiza las dos revisiones independientes del mismo hash antes de integrar.

No hay instalación en la principal, publicación, recorrido web ni permisos de datos reales acreditados por estos archivos. Las migraciones DOWN se reservan al clon sin hechos y rechazan la reversión si existe historia de gobierno.

Los intentos iniciales de DOWN fueron transaccionales: AD134 falló por un literal mal citado antes de retirar objetos y Cat4 rechazó historia con SQLSTATE `55000`. Ambas conexiones revirtieron. Dirección precisó después que no se ejecutaría ningún DOWN adicional, también en clones; el literal de AD134 se corrigió por inspección.

La preimagen positiva de AD134 es el núcleo de AD126 con `search_path=pg_catalog`. El ensayo en clon nuevo H1 y las 41 SQL previas exactas de `e528c7eaa` rechazó AD133 `af68c20b` después de AD126 con `55000: preimagen global incompatible`. Su matriz de 17 funciones todavía no admite este linaje. AD134 conserva la guarda exacta; integrar AD133 o AD135 en el mismo producto exige una convergencia nueva y revisada, y otro ensayo causal. No se acredita compatibilidad por aceptar otra configuración de `search_path`.
