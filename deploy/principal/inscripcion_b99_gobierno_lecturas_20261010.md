# Activación de las lecturas B99

B99 se instala después de B97, que sigue a B96. El código cambia las hojas exactas de cinco
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
cadena B96, B97 y B99: ejecutar
`deploy/postgresql/bolsa_llamamientos/pruebas_sql/b99_lecturas_pg18.sh
<nombre-contenedor>`. La prueba añade sus aserciones a la fixture B96 antes
del `ROLLBACK`; no se ejecuta en una base conservada.

El ensayo del 10/10 midió 33,97 ms para el selector y 25,53 ms para una
lista de 50 con 2.000 solicitudes ficticias. Usó la preimagen B96 y no
acredita la migración final sobre B97 ni la frontera HTTP.

El 11/10 se extrajo una copia de `base_hz19_pgdata.tar` a
`/var/tmp/vec-codexi-b99-20261011/fresh/18/docker`. El tar original no se
modificó. PostgreSQL 18.4 corrió en el contenedor
`vec-codexi-b99-fresh-20261011`, con `--rm --restart=no --memory=2g`, datos
en disco y red desactivada. El clon HZ19 ya contenía #962. Se aplicaron las
15 SQL de `~/.cache/vec-claude-t/hz20/hz20kit/sql.list` y pasaron sus dos
pruebas estructurales. Después se aplicaron, una vez cada una, las ocho SQL
de `lista_sql_inscripcion_nucleo_20261010.txt`, las dos de
`lista_sql_inscripcion_incorporacion_20261010.txt` y B99. La guarda B99 leyó
las tres preimágenes finales esperadas; su postimagen comprobó definición,
dueño, ACL, seguridad y límites de ejecución. La instalación tardó 0,07 s.

La prueba B99 existente pasó en 0,28 s dentro de una transacción revertida.
La fixture B97 pasó en 0,11 s después de B99: la lista y el detalle de una
solicitud incorporada conservaron `participacion_ref` y `bolsa_ref`. Un
segundo intento de B99 se detuvo por la guarda de preimagen y dejó idénticas
las tres definiciones. Logs de este clon: `fresh/hz20-sql.log`,
`fresh/chain.log`, `fresh/b99-final.log`, `fresh/b99-fixture.log`,
`fresh/b97-regression.log` y `fresh/replay.log`, todos bajo
`/var/tmp/vec-codexi-b99-20261011/`. Es un ensayo con datos sintéticos; aún
faltan publicación de hojas y recorrido con identidad real.

La tarjeta RRHH se inspeccionó en Chrome del sistema con la proyección del
selector obtenida en ese ensayo, servida en un HTML temporal con el CSS del
portal. Las capturas y su alcance están en
`docs/evidencias/codexo-inscripcion-20261010/`. En ambos anchos la página no tuvo
desbordamiento horizontal. El HTML temporal no usó el servidor VEC ni una
sesión nominal; estas capturas prueban sólo el aspecto de la tarjeta.

Los catálogos RRHH ES/EN cambiaron la huella global de textos a
`39b189606d91e3bc`. `comun/textos.js` se importa sin `?v=` y el servidor
lo revalida con `no-cache`; cada catálogo se solicita con
`?huella=39b189606d91e3bc`. La prueba `textos-version.test.mjs` comprueba
la huella completa. El shell y sus cachés públicos ya apuntan a las vistas
versionadas de B99.

Se repitió la instalación final en otra copia desechable del tar, con la misma
cadena causal. El LOGIN `vec_b99_migrador` no tenía privilegio de superusuario:
recibió `CONNECT` sobre la base de ensayo y permiso `SET` sobre
`vec_bolsa_llamamientos_propietario`. B99 terminó en 0,07 s y conservó el
dueño y ACL esperados. El registro está en
`/var/tmp/vec-codexi-b99-20261011/migrator/migrator.log`; el primer intento
de conexión denegado queda en ese log y no ejecutó SQL. No se crearon este
LOGIN ni estas concesiones en la principal.

CONFIG NUEVA: versión gobernada de los roles que ya conceden las cinco
acciones anteriores, con las cinco hojas exactas de la tabla. No hay variable
de entorno nueva.
