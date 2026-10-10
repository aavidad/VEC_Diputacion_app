# Activación de las lecturas B99

B99 se instala después de B96. El código cambia las hojas exactas de cinco
lecturas. Antes de encender la inscripción, Dirección debe publicar una versión
gobernada de cada rol vigente que conceda estas acciones. La versión nueva
conserva ámbito, finalidad, obligaciones, vigencia y todas las demás hojas de
la versión actual; añade solo la hoja indicada:

| Acción | Hoja nueva |
| --- | --- |
| `bolsa.inscripcion.propias.listar` | `solicitudes[].convocatoria_titulo` |
| `bolsa.inscripcion.propia.consultar` | `convocatoria_titulo` |
| `bolsa.inscripcion.rrhh.listar` | `solicitudes[].convocatoria_titulo` |
| `bolsa.inscripcion.rrhh.consultar` | `convocatoria_titulo` |
| `bolsa.inscripcion.rrhh.convocatorias.listar` | `convocatorias[].pendientes` |

La preparación toma la instantánea vigente del rol y su huella desde la
autoridad de autorización. La propuesta y el cierre usan el circuito de
versionado de rol Bolsa con sus dos decisiones administrativas; se comprueba
que la versión anterior y las asignaciones nominales siguen intactas. No se
añaden hojas mediante SQL manual ni se infiere un permiso desde la ruta.

Tras publicar la versión, comprobar con dos identidades de prueba que la
persona ve el título en «Mis solicitudes» y que RRHH ve el recuento pendiente.
Al abrir una convocatoria, `total` de la lista con estado `pendiente` debe
coincidir con `pendientes` de su tarjeta. Comprobar también una solicitud
histórica y una convocatoria con cero pendientes. Cada GET debe dejar su
asiento nominal de lectura. El resultado del ensayo local no acredita esta
publicación ni un recorrido con identidad real.

Ensayo focal en clon **desechable** de PostgreSQL 18, una vez instalada la
cadena B96 y B99: ejecutar
`deploy/postgresql/bolsa_llamamientos/pruebas_sql/b99_lecturas_pg18.sh
<nombre-contenedor>`. La prueba añade sus aserciones a la fixture B96 antes
del `ROLLBACK`; no se ejecuta en una base conservada.

El ensayo aislado del 10/10 creó las tres funciones sobre la preimagen literal
de B96 y 2.000 solicitudes ficticias con índices de lista y versión. La
convocatoria A mostró 1.000 pendientes y su lista devolvió 1.000; la B mostró
cero. La ficha propia en inglés mostró el título histórico y el ámbito ajeno
de RRHH devolvió cero solicitudes. En caliente, `EXPLAIN ANALYZE` midió
33,97 ms para el selector y 25,53 ms para la lista de 50. Es una medición de
las funciones internas con dobles de metadatos/catálogos; no mide la frontera
HTTP, la auditoría nominal ni el clon completo. La segunda aplicación de B99
falló por la guarda de preimagen antes de alterar funciones.

La tarjeta RRHH se inspeccionó en Chrome del sistema con la proyección del
selector obtenida en ese ensayo, servida en un HTML temporal con el CSS del
portal. Las capturas y su alcance están en
`docs/evidencias/codexo-inscripcion-20261010/`. En ambos anchos la página no tuvo
desbordamiento horizontal. El HTML temporal no usó el servidor VEC ni una
sesión nominal; estas capturas prueban sólo el aspecto de la tarjeta.

CONFIG NUEVA: versión gobernada de los roles que ya conceden las cinco
acciones anteriores, con las cinco hojas exactas de la tabla. No hay variable
de entorno nueva.
