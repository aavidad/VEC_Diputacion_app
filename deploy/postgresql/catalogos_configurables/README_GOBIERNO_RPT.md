# Gobierno de categorías RPT

Las categorías RPT las gestiona RRHH mediante un único circuito versionado. La orden de Dirección del 1 de octubre de 2026, 01:37 CEST, y la rectificación de las 02:00 fijan propuesta, aprobación por otra persona de RRHH y confirmación. La capacidad funcional usa autorización nominal propia de RRHH; los permisos y la ruta ADMIN no habilitan este recorrido.

Catálogos `000004` (Cat4) conserva una propuesta inmutable de publicación o deshabilitación, su contenido completo y SHA-256, fuente, motivo catalogado y preimágenes. AD3 `000134` (AD134) conecta cada acto con la autoridad V3. El rol técnico de gobierno está separado de los consumidores de usos; estas migraciones no provisionan personas ni concesiones funcionales. La adaptación SQL y Go sigue siendo candidata, pendiente de composición y de las comprobaciones del contenido final.

## Propuesta, aprobación y confirmación

Una persona de RRHH, A, propone el cambio y obtiene revisión 1. Otra persona de RRHH, B, aprueba la misma huella y obtiene revisión 2. B confirma con revisión esperada 2 y obtiene revisión 3. A no puede aprobar ni confirmar su propuesta. La confirmación conserva la huella de la propuesta aprobada; cambiar el contenido o la persona publicadora exige una propuesta nueva.

El documento completo fija a A como editor y a B como publicador. Su `aprobacion_ref` debe coincidir con el recibo de la aprobación de B. SQL coteja ambas identidades con V3. Cat1 conserva intactas las publicaciones e historia del circuito anterior con dos aprobaciones. En el nuevo circuito se registra la propuesta de A y una aprobación de B; no se genera una segunda aprobación ficticia para satisfacer el contrato anterior.

Cada acto, incluida la recuperación de un recibo, requiere una decisión nueva, positiva y vigente. Antes de confirmar, AD134 acredita que la aprobación histórica de B quedó consumida y auditada para la misma propuesta y huella. Su sobre breve puede caducar después sin borrar ese acto histórico; una revocación posterior del perfil tampoco lo retira retroactivamente. B necesita autorización V3 vigente para confirmar o recuperar el recibo. La decisión anterior no se consume otra vez. Anular una aprobación ya emitida necesitaría otro acto versionado, pendiente de especificación por RRHH.

La transacción reúne consumo y auditoría central V3, CAS del catálogo, historia, confirmación y outbox. Un fallo revierte el conjunto. El catálogo conserva el módulo de su publicación anterior canónica; Cat4 lo coteja bajo bloqueo antes de invocar la publicación.

En uso, una categoría se deshabilita y conserva su historia, autor y motivo. La deshabilitación publica una versión nueva y completa del catálogo, con la categoría afectada en estado `deshabilitada`. Bloquea reservas nuevas y conserva publicaciones, reservas y terminales anteriores. La fachada admite exclusivamente `publicar` y `deshabilitar`; borrado, cobertura histórica, tombstone y rehabilitación quedan fuera de este contrato.

## Material y contexto confiable

Las tres fachadas reciben material JSONB y las diez piezas de consumo V3. El recurso es `propuesta_categoria`, la finalidad `gobernar_categorias_rpt`, los campos `gobierno` y `recibo`, sin obligaciones. Las acciones son `vec.catalogos.categorias.gobierno.proponer`, `.aprobar` y `.confirmar`, con audiencia `vec_catalogos_configurables.gobierno_categorias.v1`. Declararlas en código no publica una concesión funcional RRHH.

Proponer recibe `propuesta_ref`, `contenido`, `huella_sha256` y `recibo_ref`. El contenido tiene exactamente estos campos:

```text
accion, catalogo_id, modulo_id, version, documento_canonico,
documento_huella_sha256, preimagenes_control, preimagenes_huella_sha256,
categoria_id, revision_esperada, motivo_ref, fuente_ref
```

Publicar y deshabilitar incluyen siempre el documento completo y la huella de sus bytes UTF-8 canónicos. En publicación, categoría y revisión esperada son `null`. Las preimágenes incluyen las categorías anteriores incluidas en la nueva publicación. En deshabilitación, la categoría y su revisión esperada identifican el control previo habilitado; su preimagen conserva `version`, `huella_sha256`, `revision` y estado `habilitada`, y el documento nuevo contiene la entrada deshabilitada. Documento y huella no se sustituyen por `null`.

Aprobar y confirmar reciben `propuesta_ref`, `huella_sha256`, `recibo_ref`, `revision_esperada`, `catalogo_id` y `modulo_id`. SQL coteja catálogo y módulo con la propuesta registrada. La preparación calcula `sha256(contenido::text)` y `sha256(material::text)` mediante SQL sin acceder a tablas. Go no reproduce la serialización de JSONB. El contexto canónico del recurso tiene esta estructura; los valores siguientes son ejemplos técnicos, no un catálogo aprobado:

```json
{"ambitos":{"catalogo_id":"catalogo-demo","modulo_id":"personal"},"atributos":{"material_sha256":"<SHA256 del material JSONB::text>"}}
```

El motivo procede del contexto confiable V3 y debe coincidir con el `motivo_ref` de la propuesta en los tres actos. El contexto de actor y la autorización nominal RRHH dependen de las entregas de D: ContextoActor `000021` (CA21), Autorización `000025` (AUT25) e Identidad IS10. El descriptor común de fuente de D aún no está aprobado. No se fija aquí un catálogo, versión o permiso sustitutorio, ni se toma el motivo del cliente como autoridad.

## Dependencias y comprobaciones pendientes

El orden completo previsto es B → A → CA21 → AUT25 → Cat4 → AD134. La [lista SQL del tramo RPT](../../principal/lista_sql_trabajo_codexm_rpt_gobierno_20260930.txt) conserva las dependencias y remite a las entregas de D para fijar sus rutas y hashes revisados. Cat1/2, AD117, Cat3 y AD126 pertenecen al antecedente de catálogos; no se reaplican ni se ejecuta DOWN sobre su historia.

Falta fijar y revisar la postimagen convergente B → A, incorporar las dependencias nominales de D y ensayar el contenido final en PostgreSQL 18. Son necesarias dos revisiones independientes con GO sobre el mismo hash final antes de integrar. Los ensayos anteriores de este documento no acreditan esas condiciones.

Las pruebas focales previstas están en [Cat4](pruebas_sql/000004_gobierno_categorias_rpt.sql) y [AD134](../autorizacion_atestada_v3/pruebas_sql/ad3_134_gobierno_categorias_rpt.sql). Deben acreditar separación A/B, una sola aprobación, confirmación B en revisión 3, motivo y contexto V3 exactos, deshabilitación mediante versión completa, conservación del historial Cat1, CAS y recuperación sin duplicados. Quedan pendientes las garantías V3 reales, revocación, rollback de auditoría y recuperación del circuito final sobre la postimagen acordada.

Esta actualización documental no acredita instalación en principal, publicación, recorrido Chrome ni producción. El puerto de escritura RRHH permanece cerrado hasta disponer del contrato nominal compuesto y del recorrido real. Los comandos históricos de abajo no son una orden de ejecución. No se ejecuta ningún DOWN adicional, tampoco en clones.

## Evidencia histórica de ensayos anteriores

Los hechos siguientes corresponden a los candidatos y linajes indicados en cada párrafo. Se conservan como historia y no acreditan el hash nuevo ni el circuito RRHH de revisión 3. El contrato de aquellos ensayos exigía dos aprobaciones distintas, permitía al editor aportar una y confirmaba de revisión 3 a 4 con otra identidad. Al deshabilitar, su material llevaba documento y huella `null` y una preimagen de la categoría habilitada, sin publicar una versión completa. Estas reglas quedaron sustituidas por el circuito descrito arriba.

El orden usado para los ensayos históricos estaba en `deploy/principal/lista_sql_trabajo_codexm_rpt_gobierno_20260930.txt`. Requería AD3-114 y la cadena Catálogos roles/1/2, AD3-117, Catálogos3 y AD3-126 en un clon nuevo y coherente. Las pruebas SQL de aquel corte revirtieron sus datos sintéticos.

`pruebas_sql/000004_gobierno_categorias_rpt.sql` comprobó en aquellos candidatos doble identidad, huella cambiada, módulo ajeno a la publicación, cambio de módulo entre versiones y publicación posterior con el mismo módulo, CAS, recibo idéntico al recuperar, fallo posterior al efecto, conservación del uso y denegación de una reserva posterior a deshabilitar. El fallo de persistencia por referencia de auditoría ausente prueba el rollback del catálogo; la comprobación de la auditoría central corresponde a AD3.

En aquel corte quedaron pendientes de acreditar: atestaciones criptográficas V3 para las dos identidades del circuito, caducidad del sobre de las aprobaciones manteniendo su efecto histórico, revocación del confirmador antes de su acto y fallo de auditoría central. La carrera real en dos conexiones se ensayó en ambos órdenes sobre el core de persistencia, observando el bloqueo: una reserva anterior se conserva y una posterior se deniega sin fila. Esta comprobación usa identidades sintéticas del core y no acredita V3. La revisión independiente del hash final seguía pendiente; esta evidencia histórica no la cierra.

Aquel corte no acreditó instalación en la principal, publicación, recorrido web ni permisos de datos reales. Entonces las migraciones DOWN se reservaban al clon sin hechos y rechazaban la reversión si existía historia de gobierno; la orden posterior de no ejecutar más DOWN permanece vigente.

Los intentos iniciales de DOWN fueron transaccionales: AD134 falló por un literal mal citado antes de retirar objetos y Cat4 rechazó historia con SQLSTATE `55000`. Ambas conexiones revirtieron. Dirección precisó después que no se ejecutaría ningún DOWN adicional, también en clones; el literal de AD134 se corrigió por inspección.

La preimagen positiva de AD134 en aquel candidato era el núcleo de AD126 con `search_path=pg_catalog`. El ensayo en clon nuevo H1 y las 41 SQL previas exactas de `e528c7eaa` rechazó AD133 `af68c20b` después de AD126 con `55000: preimagen global incompatible`. La matriz de 17 funciones de aquel AD133 no admitía ese linaje. AD134 conservaba la guarda exacta; aquel resultado exigía una convergencia nueva y revisada, y otro ensayo causal, antes de integrar AD133 o AD135 en el mismo producto. Aceptar otra configuración de `search_path` no acreditaba compatibilidad. Este rechazo histórico no describe la postimagen B → A pendiente de fijar para el circuito final.

El candidato SQL `39ef1e909` superó el ensayo de 43 SQL UP en un clon PG18 nuevo: las 41 previas exactas de `e528c7eaa` y Cat4/AD134 aplicadas una sola vez. Las dos pruebas SQL focales terminaron con ROLLBACK. Las carreras reales usaron dos conexiones, con el bloqueo de la segunda observado antes de liberar la primera. Tras `pg_ctl restart` del PostgreSQL aislado, cuatro confirmaciones del core devolvieron los mismos recibos y resultados; revisiones, usos y outbox quedaron idénticos. Estas confirmaciones usan el propietario técnico como doble de persistencia y no acreditan un consumo V3 real.

Comandos de comprobación focal de aquel corte, exclusivamente contra su clon desechable preparado con el orden anterior:

```sh
docker exec -i <clon_aislado> env -i PATH=/usr/lib/postgresql/18/bin:/usr/bin:/bin HOME=/scratch \
  psql -X -q -U postgres -d postgres -v ON_ERROR_STOP=1 \
  < deploy/postgresql/catalogos_configurables/pruebas_sql/000004_gobierno_categorias_rpt.sql
docker exec -i <clon_aislado> env -i PATH=/usr/lib/postgresql/18/bin:/usr/bin:/bin HOME=/scratch \
  psql -X -q -U postgres -d postgres -v ON_ERROR_STOP=1 \
  < deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/ad3_134_gobierno_categorias_rpt.sql
```

El análisis local Semgrep de las cuatro migraciones usa dos reglas genéricas para concesiones positivas a PUBLIC y funciones definidoras sin ruta fija. No envía código y no sustituye las dos revisiones SQL independientes. `git diff --check` forma parte del cierre del candidato.

El correctivo `a233017cf` se ensayó de forma focal en otro clon PG18 nuevo con las SQL previas de `2085c5981`, que incluyen el refuerzo Cat1. Cat4 y AD134 se aplicaron una sola vez. La regresión Bolsa v1 a Personal v2 exigió el error exacto de Cat4 `42501` antes de invocar Cat1; no cambió la publicación ni el control previo y no creó confirmación, historia ni outbox del intento. Bolsa v1 a Bolsa v2 se confirmó y su replay devolvió el mismo resultado. Las dos pruebas SQL focales terminaron en ROLLBACK. No se repitieron carreras, reinicio ni pruebas globales para este correctivo.
