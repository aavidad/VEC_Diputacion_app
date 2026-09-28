# Ensayo aislado de plantillas RRHH 4.08

Desde la raíz de este worktree:

```bash
bash scripts/rrhh_plantillas/probar_pg18.sh --synthetic
```

El runner crea `postgres:18.4-alpine` con `--rm --network none`, datos y socket en
`/dev/shm`, y retira contenedor y temporales incluso al fallar. La base es nueva
y contiene solo datos sintéticos. No usa PostgreSQL ni servicios compartidos.

Instala los ficheros **reales** CT131, AD3-99 y CT135. Comprueba ROLLBACK sin
rastro en base vacía, UP y doble UP rechazado. Ejecuta la CLI migradora con el
catálogo de ejemplo, consulta la preimagen con LOGIN CT, repite la provisión y
verifica recibo/fecha/versión/huellas tras reiniciar PostgreSQL. El test Go se
inyecta con `-overlay` temporal y llama los preflights reales del paquete
`bootstrap`, sin editar producto.

La preimagen `preimagen_sintetica.sql` contiene **dobles explícitos** de AD3-94,
AD3-96 y el núcleo de consumo V3. `ct133_doble.sql` solo crea la firma que CT135
reemplaza; la migración CT133 real está ausente en esta base. Las firmas CT108 y
CT132 y las dos fachadas de Autorización usadas para probar ACL son también
dobles. Por ello el ensayo **no acredita** emisión/consumo V3 real, lectura
documental CT133, CT132/CT134 instaladas, CAS de revocación o restricción,
carrera entre transacciones, TLS, HTTP, navegador ni firma. La organización
correcta se consulta y las ajenas/ausentes reciben `42501` por CT135/AD3-99
reales, pero el consumo termina en el doble V3.

La base `09d78b702e1b7a2b00f33f436d8cc28c4f0e00b9` deja el preflight Go
rojo: el LOGIN CT no tiene `USAGE` del esquema AD3 y `to_regprocedure` devuelve
SQLSTATE `42501`. Los LOGIN fuente/motivos tampoco tienen `USAGE` CT, y la
consulta equivalente sobre CT devuelve `42501`. El runner conserva exit 1 en
ese caso, aunque continúe las sondas independientes.

Para ensayar un parche todavía no integrado sin modificar este worktree, se
pueden inyectar las dos rutas absolutas de los ficheros candidatos:

```bash
VEC_PLANTILLAS_PREFLIGHT_SOURCE=/ruta/contratacion_temporal_plantillas_catalogo_preflight.go \
VEC_PLANTILLAS_AUTORIDADES_SOURCE=/ruta/contratacion_temporal_plantillas_desarrollo.go \
bash scripts/rrhh_plantillas/probar_pg18.sh --synthetic
```

El 28/09/2026 el hash de producto `0861e9b1beed8a42ff64ff7a3118cf5d8528c80c`
dio exit 0 con esos overlays: preflight CT y fuente/motivos positivos, ausencia
de AD3-99/CT135 y concesiones cruzadas CT108/CT132 denegadas, revocación AD3
al propietario CT denegada, función AD3 sin `SECURITY DEFINER` o con propietario
distinto denegada, organización ajena/ausente `42501`, CLI y reinicio sin
duplicados. Huellas SQL usadas: CT131
`a03d6d5192fc8f5b632a4d047aa27fa67e360845a1d65e88e2e16877436083a2`,
AD3-99 `6e2168f14a2b66d250328995ca966f948f725888439bff4da0fe84088d676eaa`,
CT135 `69d21d66453b5492d5d3569e97325267546edcd9d24f0ac0de16371e3f9e9d3c`.

Para acreditar la cadena completa de autoridades se necesita una preimagen
auténtica sintética de todas las migraciones previas, con sus roles, ACL y
funciones V3 reales; `000111_preimagen_pg18.sql` no sirve porque usa dobles AD3
para CT54–64. Después deben recorrerse la composición y la descarga desde
navegador con las identidades nominales y TLS del perfil correspondiente.
