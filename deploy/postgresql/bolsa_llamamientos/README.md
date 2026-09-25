# PostgreSQL: propuestas de llamamiento

Este despliegue implementa el primer adaptador durable del puerto
`TransaccionPropuestasLlamamiento`. No es una demostracion ni un repositorio
alternativo: relee y bloquea la necesidad y sus fuentes autoritativas, conserva
la propuesta canonica, consume una unica decision, encadena auditoria y crea el
evento outbox dentro del mismo `COMMIT` `SERIALIZABLE READ WRITE`.

La funcion permanece **cerrada por defecto**. Ningun rol runtime recibe
`USAGE` del esquema ni `EXECUTE` de `guardar_propuesta_v1` hasta que exista un
registrador COSE productivo. Un JSON de decision y su SHA-256 aportan
integridad, pero no prueban por si solos la identidad del PDP. No se ha creado
una firma ficticia para aparentar que esa dependencia esta resuelta.

## Instalacion

1. Nucleo de autorizacion: roles y migracion `000001`.
2. Vinculo actual de autenticacion de actor de `ejecucion_documental_v4`.
3. `roles_up.sql` de este directorio.
4. `migraciones_autorizacion/000001_revalidacion_llamamientos.up.sql`.
5. `migraciones/000001_almacen_llamamientos.up.sql`.
6. `migraciones/000002_guardado_cerrado.up.sql`.

Las identidades LOGIN se aprovisionan fuera del repositorio. Propietario y
migrador son `NOLOGIN`; las cuentas de aplicacion no reciben sus credenciales.
La reversion del almacen exige la confirmacion explicita e irreversible que
figura en la migracion `down`.

## Contrato cerrado

`guardar_propuesta_v1(jsonb,jsonb,bytea,bytea)` recibe:

- un sobre de operacion con esquema versionado, clave exacta de necesidad,
  huellas, accion, finalidad, tipo de recurso y hora del servidor;
- la proyeccion tipada de `EvidenciaUsoDecisionAutorizacion` V1;
- los bytes canonicos de la decision reforzada;
- los bytes de la propuesta ya validada por el dominio Go.

Antes de escribir, PostgreSQL comprueba de nuevo:

- objeto raiz cerrado: el doble analisis `json`/`jsonb` detecta claves
  duplicadas en ese nivel, ademas de acotar tamano y comprobar SHA-256;
- version y huella exactas de necesidad, bolsa, politica e instantanea;
- necesidad actual abierta, vigencias y ausencia de sustitucion concurrente;
- cada evaluacion autoritativa, en orden, hasta el primer resultado elegible;
- identidad de la seleccion final y unicidad de todos los recibos;
- decision durable, RBAC, politicas restrictivas, sesion y `ContextoActor`;
- garantia minima y observada alta, metodo no `demo` y superficie interna;
- atestacion activa, versionada y ligada a la huella de decision;
- uso unico de decision y de
  `(necesidad_ref, version, huella_necesidad_sha256)`.

La funcion toma primero todos los bloqueos de necesidad, fuentes, referencias,
autorizacion, atestacion y checkpoint de auditoria. Solo despues captura un
reloj fresco y vuelve a revalidar decision, rol, politicas, sesion, actor y
todas las ventanas de vigencia. Una espera de bloqueo no puede convertir una
capacidad ya caducada en un efecto confirmado.

Un reintento byte a byte exacto devuelve el recibo ya confirmado. Una misma
referencia con otro contenido, otra decision para la misma necesidad o el
reuso de instantanea/recibos falla cerrado. Las carreras se resuelven mediante
bloqueos, restricciones unicas y el nivel `SERIALIZABLE`.

## Recibo tecnico

Aunque el puerto devuelve solo `error`, la funcion entrega al adaptador Go un
recibo completo antes del `COMMIT`: propuesta y huella, decision, atestacion,
consumo, auditoria encadenada, outbox y hora. Go recalcula las huellas de todos
los documentos, rechaza claves repetidas incluso en objetos anidados,
decodifica objetos cerrados, cruza todas las referencias y solo entonces
confirma la transaccion. Una respuesta truncada o manipulada produce rollback.

PostgreSQL normaliza la propuesta a `jsonb` para sus comparaciones semanticas.
Eso no demuestra por si solo un canon lexico de los bytes ni detecta claves
duplicadas dentro de objetos anidados despues de normalizarlos. La propuesta
que envia el adaptador ya procede del `encoding/json` del dominio, pero la
futura frontera de apertura COSE debera verificar los bytes canonicos y la
ausencia de duplicados **antes** de cualquier conversion a `jsonb`; no puede
atribuir esa garantia a esta comprobacion SQL.

## Privilegios y RLS

Las catorce tablas tienen RLS habilitada y forzada. La unica politica positiva
corresponde al propietario `NOLOGIN`. Ejecutor, proyector autoritativo,
registrador de atestaciones y despachador outbox nacen sin acceso a esquema,
tablas o funciones. Las funciones fijan `search_path` y zona horaria.

Abrir la frontera requiere otra migracion independiente que:

1. reciba una capacidad efimera de un verificador COSE aislado;
2. valide bytes canonicos pre-`jsonb`, suite, audiencia, clave, rotacion,
   revocacion y confianza;
3. ligue decision completa y efecto de propuesta;
4. registre la atestacion sin conceder escritura de tabla;
5. pruebe firma alterada, replay, revocacion y carrera;
6. conceda finalmente solo `USAGE` y `EXECUTE` al ejecutor.

El proyector de datos autoritativos y el despachador outbox tambien necesitaran
funciones estrechas propias. Nunca se deben abrir privilegios directos sobre
tablas para resolver esas integraciones.

## Pruebas

```bash
./deploy/postgresql/bolsa_llamamientos/probar_integracion.sh
```

Pruebas por migración (todas en PostgreSQL 18 desechable; las que se
ejecutan como superusuario terminan en ROLLBACK y sustituyen, solo dentro de
su transacción, el consumidor de autorización V3 por un doble: la
autorización tiene sus propias pruebas):

- `000018` (B6, orden vigente): `pruebas_sql/b6_orden_vigente.sql` sobre una
  réplica sintética. Comprueba pausa, retorno tras contrato, recolocación al
  final, una sola fila append-only y denegación de lectura directa.
- `000024` (B13, histórico de contratos): `pruebas_sql/b13_historico_contratos.sql`
  sobre la estructura real con CT `000113` y Bolsa `000024`. Proyecta las
  incorporaciones CT, las entrega dos veces al inbox sin duplicar, rechaza
  contenidos divergentes o inválidos y comprueba las ACL. El relevo Go con
  adaptadores reales se prueba con `TestEntregaContratosCTBolsaPostgreSQL18`
  (`VEC_B13_PG18_CT_DSN` / `VEC_B13_PG18_BOLSA_DSN`). Cadencia y lote del
  relevo: `VEC_BOLSA_CONTRATOS_CT_INTERVALO` (`0` lo desactiva) y
  `VEC_BOLSA_CONTRATOS_CT_LOTE`; `VEC_BOLSA_CONTRATOS_CT_RELECTURA` queda como
  margen adicional. El inbox confía en el relevo (ver el comentario de
  `000024`): Bolsa no lee tablas de CT y el evento no lleva firma de origen.
- `000031` (intentos de contacto): `probar_intentos_contacto_pg18.sh`.
- `000035` (contacto de origen CONVOCA): `probar_origen_datos_contacto_pg18.sh`.
- `000039` (expiración): `probar_expiracion_rrhh_pg18.sh`.
- `000028`, `000030`, `000032` y `000037`: `probar_revision_bolsa_pg18.sh`
  instala la cadena, prueba UP/DOWN/UP y dependencias de `000032`/`000037` y
  ejecuta `b2_politica_transiciones.sql` (la versión 1 es el literal de
  `000012` más `disponible→disponible_desde`; el Reglamento cierra
  `renuncia→disponible` y abre `renuncia→no_disponible`, invariantes fijas,
  publicación idempotente, solo adición y ACL), `bof_ofertas_publicadas.sql`,
  `b30_portal_candidato.sql`, `revision/b30_replay_autorizacion.sql` y
  `b37_efectos_sanciones.sql` (suspensión con fin por la política vigente,
  readmisión por recurso estimado como única salida de «excluido»).
  `000032` no modifica situaciones ya registradas.

El script usa PostgreSQL fijado por imagen y digest, verifica ACL negativas,
RLS, `SECURITY DEFINER`, claves de idempotencia y una carrera real por la misma
necesidad. Los datos sinteticos se insertan unicamente como propietario en la
base efimera para probar restricciones; no crean una via de carga productiva.
La prueba positiva de la funcion sigue cerrada hasta disponer de COSE real.
