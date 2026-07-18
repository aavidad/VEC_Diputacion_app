# Revisión cruzada Orquesta de PostgreSQL para borradores

Fecha: 2026-07-18

Estado: **NO-GO técnico; no integrar la migración revisada**

Alcance: migración PostgreSQL `000003` del diario, persistencia y recuperación
de borradores de convocatorias, sus pruebas SQL y la paridad estática con los
contratos Go.

## Resultado ejecutivo

Orquesta confirmó un defecto alto en la idempotencia multigeneración. Dos
ventanas de rotación válidas y solapadas, por ejemplo `[g3,g2]` y `[g2,g1]`,
pueden producir dos reservas durables para la misma operación lógica porque
cada reserva busca todas sus identidades, pero solo persiste la primaria. El
testigo funciona en orden serial; `SERIALIZABLE` no lo corrige.

También se confirmó una barrera destructiva incompleta en el `down` y una
cobertura insuficiente de su inventario. El corte SQL no se comiteará hasta
corregir los tres hallazgos, ejecutar PostgreSQL real y superar otra revisión
independiente sobre hashes nuevos.

El NO-GO técnico de este corte es distinto del NO-GO productivo global. Aunque
se corrija la migración, producción continuará cerrada hasta disponer de
identidad fuerte, PDP, HSM/KMS, cuentas y DSN nominativos, privilegios mínimos,
adaptador Go y despliegue acreditado.

## Fuentes fijadas

| Fuente | SHA-256 revisado |
|---|---|
| `migraciones/000003_borradores_durables_cerrados.up.sql` | `06dec1f110a74d7e361baeadeb53e151afaa153e01f03422af1507535f0dbebe` |
| `migraciones/000003_borradores_durables_cerrados.down.sql` | `fb3ced036e4be54cf15d7b923c85d2f634ad414a3016d047b82048fd64a4d079` |
| `pruebas_sql/borradores_durables.sql` | `6c4dbe2fd78b630e6df26dc170397c5150c8cf664c3e21888b55dc0b891ce9e5` |
| `pruebas_sql/down_borradores_limpio.sql` | `01e521fc7dec954a1928b3c1604a67019200e09a86fba405f43659cbbdac22f9` |
| `pruebas_sql/acl_cierre.sql` | `f2a48dff2cdf8b89aa1687a5690140c20e62b622a14b6cd7e9b92869bfff296f` |
| `probar_integracion.sh` | `f64a3ae4087e8635651ec0f78332584f28a249387ee21ac3eda339ef7e6938b8` |
| `contratos_reserva.go` | `ca31521da4cd0f3d99a15034286abdef7722ff09035537fbaaaa7740bc111531` |
| `recuperacion_borradores.go` | `ab9068edc1a9cd5874127006556961507afdb7f1b0d1a3119f57ebadeb65a346` |
| `servicio_borradores.go` | `9abb4f604c559e9b4ca082e0eb17dc2619db4027c34109579bcede6bbf1fe48b` |

Las rutas SQL son relativas a
`deploy/postgresql/bolsa_convocatorias`; las rutas Go pertenecen a
`internal/modules/bolsa/application/gobiernoconvocatorias`.

## Revisión primaria

- Goal: `goal:d123e7a4fcf1e9779021df05306a5437`, revisión 10,
  estado `succeeded`.
- AppSpec: `app-spec:5e5237b1e12cd8c959e26fd39053bb04`.
- Hash AppSpec:
  `f486928ab0025c7f09bdd783cfe70babe7e277398052192b7539a5a61dc7247c`.
- Seguridad PostgreSQL:
  - ejecución `execution:7c89ee51c949a9a9aad28b2e8199289d`;
  - artefacto
    `artifact:sha256:e24a1769bf6af81dc5e09f09e98b9a4fc82aadbb1437f699fcfbfa0f2c468c59`.
- Atomicidad y recuperación:
  - ejecución `execution:d533179ab65f6d27c28d15f3ff05da1c`;
  - artefacto
    `artifact:sha256:82c6b11d827da5fca67b23635d46f27c44b17ed607e41529583c6b11a0d9f0b2`.
- Paridad SQL-Go:
  - ejecución `execution:f7b7fe0105f8c9e37dd0b7209d92f55c`;
  - artefacto
    `artifact:sha256:748e644b1508ffe9666dd373c5f9d17219a234b1a391f1bee594bd71ae05919d`.

## Metarrevisión independiente

- Goal: `goal:5ed123c3df7c8df31251f8bcf027e5d9`, revisión 8,
  estado `succeeded`.
- AppSpec: `app-spec:36e308aa81f9cb23ad72f47a4c948e5f`.
- Hash AppSpec:
  `eed37d0d7c61002e907283df3bcdbad770497d03381f1b6a6df36954233e6b5f`.
- Validez factual:
  - ejecución `execution:6af9c56692b8b88b321db1004365bfbe`;
  - artefacto
    `artifact:sha256:5d0eaccd91556b9ad74b80b089c595372ed041a28c519cbcf47a8c0720a9d7aa`.
- Riesgo público:
  - ejecución `execution:0f7a2860772fa7105f9c328d1b85e7c9`;
  - artefacto
    `artifact:sha256:e6a1f486c0641fb2aedffd24b2b16c3e4fe7f3bd33ab0edf52f258206c49dadb`.

Los cinco artefactos superaron Base64 canónico, UTF-8 estricto, MIME
`text/plain`, pertenencia al Goal, tamaño declarado y SHA-256 recalculado.
Orquesta no editó fuentes ni ejecutó pruebas.

## Hallazgos confirmados

### Alto: aliases multigeneración no persistidos

La aplicación acepta conjuntos ordenados de una a cuatro identidades. La
migración revisada consulta todas, pero inserta únicamente la identidad
primaria en el diario. Si una operación se reserva con `[g3,g2]`, `g2` no queda
como alias. Otra solicitud válida con `[g2,g1]` no busca `g3` y puede reservar
`g2` como una operación distinta.

La corrección debe persistir atómicamente todos los aliases L/F presentados,
ligarlos a una única identidad primaria y aplicar unicidad también a esos
aliases. La prueba de aceptación debe cubrir el testigo serial, las dos
direcciones concurrentes, replay y rotación durante toda la retención de la
idempotencia.

### Medio: consentimiento destructivo incompleto

La comprobación de historia de `000003.down.sql` no incluía
`material_borrador` ni `atestacion_pdp_borrador`, aunque después eliminaba ambas
tablas. Historia aislada en una de ellas podía destruirse sin la frase expresa
de confirmación.

### Bajo: inventario incompleto del down

La prueba `down_borradores_limpio.sql` no inventariaba todas las funciones de
la migración. No se demostró un residuo actual, pero la prueba no detectaría
algunas regresiones futuras.

## Controles que sí quedaron acreditados estáticamente

- DTO, nulabilidad y máquina de estados SQL-Go compatibles en el alcance.
- RLS habilitada y forzada, `PUBLIC` revocado y funciones con `search_path`
  fijado.
- No se demostró una escalada por `SECURITY DEFINER` ni acceso directo de los
  roles runtime a tablas.
- CAS de revisión y cercado, rollback transaccional y replay de una identidad
  terminal ya encontrada están presentes.
- El reloj enviado por el cliente no decide el vencimiento del lease.

Estos controles no compensan el defecto alto.

## Comprobación dinámica del director y límite descubierto

Antes de recibir el dictamen, el director ejecutó
`probar_integracion.sh` contra PostgreSQL 18 real. El runner terminó en verde y
produjo un único ganador en su carrera `SERIALIZABLE`. Ese resultado no se
atribuye a Orquesta y resultó insuficiente: la carrera usaba dos conjuntos
iguales y no ejercitaba el solape `[g3,g2]` frente a `[g2,g1]`.

La integración Go detectó además dos barreras E2E que no deben ocultarse con
dobles productivos:

1. `confirmar_borrador_v1` exige un sobre A256GCM y su atestación, mientras el
   servicio Go revisado aún no dispone de un puerto nominal que obtenga ese
   sobre de KMS/HSM y lo ligue al estado, material, revisión y cercado.
2. Los wrappers de borradores usaban una revalidación SQL cuya versión vigente
   solo admite acciones de consulta de versiones. Por ello los wrappers
   mutantes no son habilitables tal como estaban, aunque las funciones internas
   pudieran ejecutarse como propietario dentro del runner.

Estas barreras se corregirán mediante un puerto criptográfico intercambiable,
una migración nominal de autorización para borradores y pruebas con un LOGIN
runtime de mínimo privilegio. No se enviará texto claro como ciphertext, no se
concederá el rol propietario al proceso y no se reinterpretará la autorización
anterior.

## Gate de la revisión sucesora

1. Tabla o mecanismo de aliases L/F con unicidad y vínculo inmutable a la
   identidad primaria.
2. Test serial y concurrente de ventanas solapadas, más replay tras rotación.
3. Barrera destructiva e inventario de objetos completos.
4. Revalidación versionada para crear, actualizar, listar, consultar y
   recuperar borradores, con acción, recurso, finalidad y campos exactos.
5. Wrappers mínimos para consulta idempotente y reconciliación, sin acceso a
   tablas ni rol propietario.
6. Puerto Go de cifrado autenticado con KMS/HSM; preimagen de datos asociados y
   atestación nominales, versionadas y revalidadas.
7. Pruebas PostgreSQL 18 por LOGIN runtime, incluido fallo de autorización,
   caída, reinicio, rollback y pérdida de respuesta.
8. Nueva revisión primaria y metarrevisión Orquesta sobre hashes corregidos.
