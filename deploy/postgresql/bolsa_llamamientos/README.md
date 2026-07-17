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

El script usa PostgreSQL fijado por imagen y digest, verifica ACL negativas,
RLS, `SECURITY DEFINER`, claves de idempotencia y una carrera real por la misma
necesidad. Los datos sinteticos se insertan unicamente como propietario en la
base efimera para probar restricciones; no crean una via de carga productiva.
La prueba positiva de la funcion sigue cerrada hasta disponer de COSE real.
