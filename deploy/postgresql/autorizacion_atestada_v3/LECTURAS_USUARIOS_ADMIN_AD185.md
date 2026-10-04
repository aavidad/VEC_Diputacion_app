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
El CHECK de audiencias se coteja del mismo modo antes de añadir las dos
audiencias propias. No se asume una huella posterior a AD184. Ninguna fila
histórica se reescribe.

Estado: código preparado en rama aislada. Falta el ensayo causal en el clon,
dos revisiones independientes y una lectura V3 real de lista y ficha. La
composición debe confirmar COMMIT antes de devolver datos. Un error que revierta
la transacción necesita el registrador común de intentos de su canal.
