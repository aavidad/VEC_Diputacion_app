# Preparación durable de bases S2

AD153 y Convocatorias8 preparan una versión incompleta bajo autoridad Bolsa.
Conservan contenido canónico, revisión, intención, historia, auditoría, outbox
y recibo. No aprueban, firman, publican ni activan bases. Las referencias
propuestas permanecen pendientes; los 23 faltantes se derivan únicamente en
`preparacionbases.EvaluarMaterialBases`, compartido por los consumidores Go.

El contrato procede del material funcional `63a999b9` y de la posterior
interfaz V3 acordada con el responsable Go. No convierte evidencia funcional
V1 en una concesión V3. Cada llamada requiere las diez piezas de autorización
actual y una transacción `SERIALIZABLE READ WRITE`.

## Fronteras nominales

| Operación | Acción | Campos exactos |
|---|---|---|
| Guardar | `bolsa.preparacion_bases.guardar` | `auditoria`, `evento_outbox`, `historia`, `material_preparacion`, `recibo_preparacion` |
| Consultar actual o exacta | `bolsa.preparacion_bases.consultar` | `material_preparacion`, `recibo_preparacion` |

Los dos perfiles técnicos son `guardar_preparacion_bases_bolsa` y
`consultar_preparacion_bases_bolsa`; sus audiencias son
`vec_bolsa_convocatorias.preparacion_bases.guardar.v1` y
`vec_bolsa_convocatorias.preparacion_bases.consultar.v1`.

El recurso es la preparación opaca; módulo `bolsa`, tipo y finalidad
`preparacion_bases`. El ámbito contiene organización y, cuando existe, unidad
suministradas por la frontera confiable. El contexto liga SHA256 del envelope
UTF8 exacto, incluido modo y selector. SQL vuelve a comparar el ámbito
persistido antes de entregar datos. Una cadena no puede cambiar de ámbito.

Guardar recibe `p_material text`, `p_contenido bytea` y las diez piezas V3.
Consultar recibe el envelope y las mismas piezas. Ambas devuelven las 17
columnas acordadas, con material/recibo históricos separados del acceso nuevo.
`recibo_preparacion` comprende cuatro referencias históricas, fecha e intención;
no concede lectura del contenido de auditoría, historia o eventos.

Cada LOGIN tiene una sola membresía directa, `INHERIT TRUE`, `SET FALSE` y
`ADMIN FALSE`, en su grupo exclusivo: ejecutor o lector de preparación de bases.
No se reutilizan el proyector V2 ni el lector S1. Los grupos reciben `USAGE`
y ejecución exclusivamente de su función; ninguna tabla, tipo o helper.
Las fachadas AD153 son internas al propietario y el núcleo sigue privado.
Los límites runtime son 15 segundos de sentencia y 2 segundos de locks.

## Persistencia y recuperación

La clave de operación se separa por actor. El guardado bloquea actor/clave y
preparación, consume autorización nueva y busca replay antes del CAS. Mismo
actor, clave, intención y ámbito recupera la revisión y recibo originales,
aunque la cabeza haya avanzado. Cambiar material, referencia o ámbito devuelve
`clave_reutilizada` sin historia nueva. El CAS combina revisión y SHA256,
y admite alta sólo con revisión cero y huella anterior vacía.

La versión, historia, evento y recibo son de sólo adición. La cabeza admite
únicamente avance de una revisión con ámbito inmutable. El material se conserva
como bytes; sus hashes nunca se calculan sobre `jsonb::text`. La intención
reproduce el JSON canónico del dominio, sin actor, clave, correlación o fechas.

Consulta `actual` fija la cabeza visible en su misma transacción. Consulta
`exacta` requiere revisión y huella conocidas. Los conflictos, claves reusadas
y ausencias autorizadas devuelven sólo el acceso nuevo y se confirman para
conservar auditoría. La denegación de ámbito o autoridad produce `42501`.
Ninguna operación escribe tablas formales de gobierno de convocatorias.

## Estado verificable

Este corte es un borrador. Las tres huellas de AD153 proceden del núcleo
POST145 real del clon. Una preimagen posterior exige remedir y revisar el delta;
el ensayo de esta candidata no acredita automáticamente otra versión.
No depende de la numeración anterior: la lista de despliegue sigue el orden de
fusión y sólo los objetos reales requeridos. No reaplicar migraciones instaladas
ni ejecutar DOWN sobre historia.

Las pruebas propias preparan contratos, ACL y entradas negativas sin inventar
una atestación positiva. Falta el ensayo causal autorizado en el clon sintético,
alta, actualización, replay, consultas actual/exacta, conflictos, ámbito cruzado
y recuperación tras reinicio con actores, PDP y capacidades reales.
El titular del clon es el único escritor PostgreSQL. No hay instalación en la
principal, rutas HTTP compuestas ni capacidad productiva acreditadas.
