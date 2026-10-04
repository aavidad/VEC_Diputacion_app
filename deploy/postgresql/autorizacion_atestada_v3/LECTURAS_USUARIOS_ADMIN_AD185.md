# Consumo nominal de las lecturas de usuarios — AD185

AD185 añade al consumidor V3 común las acciones `administracion.usuarios.listar`
y `administracion.usuarios.consultar`. AUT43 conserva el material canónico, el
ámbito, la proyección y el gate de administrador. AD185 consume una decisión
nueva por cada consulta, también si la lista está vacía o la Persona está ausente
o fuera del conjunto. El acuse contiene los siete campos V3 existentes.

La fachada `registrar_y_consumir_usuarios_admin_v3_atestada` recibe, en orden,
`material text`, `capacidad`, `decision`, `motivo`, `contexto` como `bytea`, las
versiones de Persona y perfil como `numeric`, y `payload`, `sobre`, `evidencia`,
`raiz` como `bytea`. Solo el propietario AUT puede ejecutarla. El LOGIN técnico
debe pertenecer únicamente al grupo lector de AUT43 con herencia, sin opción de
administración ni de cambio de rol. El grupo no recibe acceso a tablas AD.

La decisión y la capacidad deben coincidir con acción, audiencia, recurso y
huella del material AUT43. La decisión exige módulo `administracion`, finalidad
`gestion_usuarios`, garantía `alto`, obligación `auditar` y los campos exactos
de AUT42. El gate AUT43 se ejecuta antes y después del núcleo, junto a la
revalidación viva del contexto. El núcleo verifica firma, raíz, revocación,
PDP y origen técnico según AD173; el asiento nuevo conserva actor, perfil,
finalidad y fecha de consumo. No se crea auditoría paralela.

La migración exige AUT42, AD184 y AUT43 instaladas. Invierte las tres
sustituciones concretas de AD184 y coteja las huellas de definición y fuente
anteriores a AD184. Si falta una marca o hay otra edición del núcleo, aborta.
El núcleo exige además las huellas posteriores a AD184 de definición
`ff77db3d6dac93c3ba8f359e6bca22a8a03489954b9c120acef4e44dc90a0bd6` y fuente
`73cb05e1c57c82c13e066a1f3c27cf03d9bdbaed3e028184ee9d040b9b230735`.
El CHECK de audiencias exige la huella `4c57e39c9b725b428149fea490bd38e6c41d608b8c727609d548363008576ebd`
de `pg_get_constraintdef(false)`, medida después de AD184. PostgreSQL deparsa
`IN` como `ANY`; el SQL conserva el predicado completo antes de añadir las dos audiencias.
Ninguna fila histórica se reescribe.

Estado: código preparado en rama aislada. Falta el ensayo causal en el clon,
dos revisiones independientes y una lectura V3 real de lista y ficha. La
composición debe confirmar COMMIT antes de devolver datos. Un error que revierta
la transacción necesita el registrador común de intentos de su canal.
