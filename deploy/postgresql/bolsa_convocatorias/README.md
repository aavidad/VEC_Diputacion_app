# PostgreSQL: gobierno, borradores y consulta exacta de convocatorias

Este despliegue crea la primera persistencia real de convocatorias gobernadas.
No es un repositorio en memoria ni una fuente de presentacion. Conserva los
bytes canonicos sellados por el dominio, sus huellas, el flujo asociado, el
consumo unico de la decision y una auditoria encadenada. La migracion
`000003_borradores_durables_cerrados` añade el diario de operaciones, la
reserva multigeneracion, su recuperacion tras fallos, el listado ligero y la
lectura exacta. `000004_confirmacion_kms_procedencia` sustituye el antiguo
`stub` de confirmacion por un protocolo A/B en una unica transaccion
`SERIALIZABLE`: persiste AAD, DEK envuelta, sobre cifrado, procedencia y
acreditacion KMS, y exige una relectura posterior al `COMMIT` desde un pool de
verificacion separado.

El conjunto esta **validado tecnicamente, pero no autorizado para produccion**.
El ejecutor de integracion acredita el contrato SQL sobre PostgreSQL real; no
sustituye el aprovisionamiento de identidades, PDP, HSM/KMS, reloj ni
observabilidad descrito al final de este documento.

La migracion abre solo fronteras nominales a roles runtime endurecidos. El
ejecutor puede listar y obtener borradores; el proyector puede consultar
identidades, reservar, reconciliar, reclamar y ejecutar las fases A/B. El
verificador solo puede releer un recibo por referencia, transaccion y huella;
no puede confirmar ni leer tablas. Ninguno recibe funciones internas,
pertenencia al propietario o privilegios directos sobre datos. Cada frontera
reidentifica un `LOGIN` de membresia exclusiva y falla cerrada si la cuenta
acumula otro rol tecnico.

## Orden de instalacion

Requiere el nucleo de autorizacion y el vinculo de autenticacion de actor:

1. `deploy/postgresql/autorizacion/roles_up.sql`.
2. `deploy/postgresql/autorizacion/migraciones/000001_autorizacion.up.sql`.
3. `deploy/postgresql/ejecucion_documental_v4/migraciones_autorizacion/000002_vinculo_autenticacion_actor_actual.up.sql`.
4. `roles_up.sql` de este directorio.
5. `migraciones_autorizacion/000001_revalidacion_convocatorias.up.sql`.
6. `migraciones_autorizacion/000002_revalidacion_borradores_v2.up.sql`.
7. `migraciones/000001_almacen_convocatorias.up.sql`.
8. `migraciones/000002_consulta_exacta_cerrada.up.sql`.
9. `migraciones/000003_borradores_durables_cerrados.up.sql`.
10. `migraciones/000004_confirmacion_kms_procedencia.up.sql`.

Las migraciones se aplican con una identidad nominativa miembro del grupo
migrador correspondiente. Las cuentas de aplicacion nunca reciben las
credenciales del propietario o del migrador. Las reservas, confirmaciones,
reconciliaciones y reclamaciones deben ejecutarse en transacciones
`SERIALIZABLE`. La consulta multidentidad exige como minimo `REPEATABLE READ`;
el ejecutor de integracion usa `SERIALIZABLE` tambien para esa lectura.
Las fases A y B deben usar la misma transaccion y la misma conexion. La
relectura del recibo usa otra credencial despues del `COMMIT`.

La reversion se ejecuta en orden inverso. Los `down` de `000004` y `000003` se
niegan a destruir historia de borradores salvo que el procedimiento formal
establezca:

```text
vec.confirmar_destruccion_borradores_convocatorias=
DESTRUIR_HISTORIA_BORRADORES_CONVOCATORIAS_IRREVERSIBLE
```

Despues, el `down` del almacen `000001` aplica su propia proteccion:

```text
vec.confirmar_destruccion_bolsa_convocatorias=
DESTRUIR_HISTORIA_BOLSA_CONVOCATORIAS_IRREVERSIBLE
```

Esa confirmacion no debe formar parte de un despliegue normal.

Con las dependencias ya instaladas y una identidad nominativa autorizada para
asumir el rol migrador, la aplicacion manual de `000003` y `000004` es
reproducible con:

```bash
psql -X --set ON_ERROR_STOP=1 \
  --file deploy/postgresql/bolsa_convocatorias/migraciones/000003_borradores_durables_cerrados.up.sql
psql -X --set ON_ERROR_STOP=1 \
  --file deploy/postgresql/bolsa_convocatorias/migraciones/000004_confirmacion_kms_procedencia.up.sql
```

El `down` ordinario solo funciona si no existe historia:

```bash
psql -X --set ON_ERROR_STOP=1 \
  --file deploy/postgresql/bolsa_convocatorias/migraciones/000004_confirmacion_kms_procedencia.down.sql
psql -X --set ON_ERROR_STOP=1 \
  --file deploy/postgresql/bolsa_convocatorias/migraciones/000003_borradores_durables_cerrados.down.sql
```

La via destructiva se reserva a una base efimera o a un procedimiento formal
aprobado, con copia verificada y autorizacion expresa:

```bash
PGOPTIONS='-c vec.confirmar_destruccion_borradores_convocatorias=DESTRUIR_HISTORIA_BORRADORES_CONVOCATORIAS_IRREVERSIBLE' \
psql -X --set ON_ERROR_STOP=1 \
  --file deploy/postgresql/bolsa_convocatorias/migraciones/000004_confirmacion_kms_procedencia.down.sql
PGOPTIONS='-c vec.confirmar_destruccion_borradores_convocatorias=DESTRUIR_HISTORIA_BORRADORES_CONVOCATORIAS_IRREVERSIBLE' \
psql -X --set ON_ERROR_STOP=1 \
  --file deploy/postgresql/bolsa_convocatorias/migraciones/000003_borradores_durables_cerrados.down.sql
```

Estos comandos usan la configuracion normal de libpq del operador. El
repositorio no proporciona ni debe contener usuarios, contraseñas o DSN.

## Contrato SQL preparado

`obtener_version_exacta_v1(jsonb,jsonb,bytea,bytea)` recibe:

- `operacion`: objeto cerrado con `esquema`, `convocatoria_id`, `secuencia`,
  `incluir_instancia_flujo`, `accion`, `recurso_ref` y `solicitada_en`;
- `prueba`: objeto cerrado con `esquema_huella`, `decision_ref`,
  `huella_decision_sha256`, `verificada_en` y `principal_ref`;
- bytes canonicos de la decision reforzada V1;
- bytes canonicos del recurso exacto
  `{"ambitos":{},"atributos":{}}`.

El esquema de operacion es
`vec.bolsa.convocatoria.consulta-postgresql.v1`. La accion se deriva de
`incluir_instancia_flujo`; no puede elegirse una combinacion distinta. La
referencia se reconstruye como `convocatoria_id#secuencia`, por lo que no
existe consulta a «la ultima version» ni selector ambiguo.

La funcion devuelve exactamente:

```text
resultado text
version_canonica bytea
huella_version_sha256 text
instancia_flujo_canonica bytea
huella_instancia_flujo_sha256 text
autorizacion_ref text
huella_autorizacion_sha256 text
atestacion_autorizacion_ref text
huella_atestacion_autorizacion_sha256 text
consumo_autorizacion_ref text
auditoria_ref text
huella_auditoria_sha256 text
consultada_en timestamptz
```

Una consulta confirmada produce `resultado=encontrada`. Una version ausente,
un flujo incompatible o una evidencia incompleta abortan la transaccion; no se
devuelve un recibo positivo parcial. Auditar `no_encontrada` requerira ampliar
el puerto de aplicacion con un recibo negativo tipado y verificable. Hasta
entonces no se inserta una falsa auditoria que el dominio no pueda validar.

## Revalidacion y consumo atomico

La frontera de autorizacion vuelve a comprobar dentro de la misma transaccion:

- decision durable, representacion canonica y SHA-256;
- accion, recurso exacto, finalidad y lista exacta de campos;
- garantia minima y observada `alto`;
- metodo distinto de `demo`;
- superficie `interna_corporativa` no privilegiada o
  `administracion_privilegiada` con cuenta privilegiada;
- asignacion, version y control de vigencia del rol;
- concesion RBAC exacta y sin obligaciones pendientes;
- revision, huella y manifiesto completo de politicas ABAC;
- sesion y `ContextoActor` actuales;
- ventana maxima de 30 segundos y reloj autoritativo de PostgreSQL.

Despues bloquea la atestacion activa, lee la version exacta, consume una sola
vez `decision_ref`, bloquea el checkpoint de auditoria, inserta el registro
encadenado y avanza el checkpoint. Cualquier ausencia, replay, carrera o
cambio concurrente aborta todo el efecto.

La migracion de autorizacion `000002` no reinterpreta la V1. Añade
`revalidar_decision_borrador_convocatorias_v2` con estos contratos cerrados:

| Accion | Clase | Finalidad | Campos exactos |
|---|---|---|---|
| `bolsa.convocatoria.borrador.crear` | `version_convocatoria_gobernada` | `gobierno_convocatorias` | `auditoria`, `evento_outbox`, `version_convocatoria` |
| `bolsa.convocatoria.borrador.actualizar` | `version_convocatoria_gobernada` | `gobierno_convocatorias` | `auditoria`, `evento_outbox`, `version_convocatoria` |
| `bolsa.convocatoria.borrador.listar` | `coleccion_versiones_convocatoria_gobernada` | `consulta_interna_convocatorias` | `version_convocatoria` |
| `bolsa.convocatoria.borrador.consultar` | `version_convocatoria_gobernada` | `consulta_interna_convocatorias` | `version_convocatoria` |

Crear/actualizar exigen el atributo exacto `huella_intencion_sha256`; las
lecturas exigen atributos vacios. Organizacion y unidad se vuelven a ligar al
recurso canonico dentro de la transaccion. Accion, clase, finalidad, recurso o
campos distintos fallan cerrados.

## Vertical durable de borradores `000003`

La migracion `000003_borradores_durables_cerrados.up.sql` implementa la
frontera PostgreSQL del nucleo de gobierno de borradores. La identidad y la
recuperacion se contrastan con estos contratos Go:

- `internal/modules/bolsa/application/gobiernoconvocatorias/contratos_reserva.go`;
- `internal/modules/bolsa/application/gobiernoconvocatorias/recuperacion_borradores.go`;
- `internal/modules/bolsa/application/gobiernoconvocatorias/cifrado_borradores.go`;
- `internal/modules/bolsa/ports/convocatorias_gobierno_motivo.go`.

Los valores de dominio Go no deben serializarse de forma generica. El futuro
adaptador PostgreSQL debe construir y leer expresamente los objetos cerrados
en castellano documentados por esta migracion, comprobar el resultado con los
metodos `ValidarPara` del nucleo y tratar cualquier campo adicional, ausente o
incoherente como error. La prueba SQL acredita la compatibilidad de identidad y
recuperacion. La confirmacion no es compatible todavia con
`cifrado_borradores.go` y queda cerrada como se detalla mas adelante.

### Identidad idempotente y rotacion de claves

Una reserva contiene `identidades_consulta` y una `identidad` seleccionada.
Cada identidad empareja dos HMAC nominalmente distintos:

- `localizador`: permite buscar sin conservar la clave de idempotencia;
- `huella_solicitud`: impide reutilizar el localizador para otra peticion.

Las reglas son cerradas:

- entre una y cuatro identidades, con la generacion primaria en primer lugar;
- misma version de esquema y misma generacion en L y F de cada pareja;
- generaciones en orden estrictamente decreciente;
- ningun L ni F puede repetirse;
- `reservar_decision_borrador_interna_v1` coteja todas las generaciones en una
  transaccion `SERIALIZABLE`; al crear conserva la primaria y todos los alias,
  y en un replay incorpora de forma atomica los alias nuevos de la ventana;
- `consultar_identidades_borrador_interna_v1` exige una instantanea
  `REPEATABLE READ` o `SERIALIZABLE`, devuelve cero o una coincidencia y aborta
  con SQLSTATE `21000` solo si las identidades resuelven a primarias distintas;
- un L existente con F distinta devuelve `conflicto`, nunca un replay;
- una primaria solo admite una pareja L/F por generacion; una pareja distinta
  de esa misma generacion aborta con `23505` y mensaje estable antes de mutar;
- consulta y replay devuelven dos valores distintos: el array ordenado
  `identidades_consultadas` que realmente resolvio como alias y la
  `identidad_primaria` completa reconstruida desde la fila autoritativa.

La relacion alias→primaria es durable y solo almacena HMAC y referencias de
clave, nunca la clave idempotente ni su preimagen. De este modo las ventanas
solapadas `[g3,g2]` y `[g2,g1]` conservan g3, g2 y g1 sin duplicar operaciones
ni cambiar un veredicto terminal ya comprometido. El adaptador puede construir
directamente `ResolucionIdentidadBorrador`: no debe inferir la primaria a
partir del primer alias devuelto.

Hasta disponer de atestacion autoritativa del servicio que deriva L/F, dicho
derivador HMAC forma parte de la base de confianza (TCB). La `UNIQUE` y el
rechazo determinista impiden dos parejas de una generacion, pero no prueban
por si solos que el HMAC fue emitido por el derivador institucional.

### Decision, sellado y recibo

La proyeccion durable de la decision conserva, sin principal ni perfil:

- accion, recurso, modulo, tipo, finalidad y huella del contexto;
- referencia y huella de asignacion;
- referencia y huella de version de rol;
- referencia, revision y huella del control de vigencia del rol;
- revision y huella del catalogo de politicas;
- emision, verificacion, caducidad y atestacion PDP exacta.

El sellado de motivo usa un objeto `hmac` anidado con dominio criptografico,
generacion, referencia de clave y valor HMAC-SHA-256. No almacena el motivo,
la clave ni la huella semantica de entrada. La atestacion, el token de consumo
y el materializador quedan ligados a accion, convocatoria, material, revision
y cercado.

El nucleo interno legado de prueba genera el recibo cerrado
`bolsa.convocatoria.borrador.recibo.v2`, que liga:

- referencia de recibo y transaccion;
- accion y estado principal confirmado;
- identidad L/F que se uso;
- proyeccion PDP y la forma de sellado heredada que aportan los fixtures;
- revision, cercado y ventana de arrendamiento;
- auditoria, evento de salida y sus huellas;
- instante autoritativo de confirmacion.

En esas pruebas, el recibo canonico y su SHA-256 se conservan en el diario y
los replays terminales devuelven esos mismos bytes logicos. Esto acredita la
maquina transaccional heredada; no acredita atestacion KMS, DEK envuelta,
perfil/AAD gobernados ni compatibilidad productiva con el nucleo Go actual.

### Estados, revision y cercado

`ausente` y `conflicto` son resultados de consulta; no son estados persistidos.
El diario durable admite estos estados:

| Estado | Regla de control |
|---|---|
| `reservado` | Un alta nueva comienza en revision 1 y cercado 1. |
| `en_curso` | La confirmacion eleva la revision dentro de su unica transaccion y conserva el cercado. |
| `indeterminado` | Obliga a reconciliar; nunca autoriza por si solo un nuevo intento. |
| `confirmado` | Es terminal: revision mayor que la observada, cercado igual y recibo V2 obligatorio. |
| `no_aplicado` | Es terminal para ese intento: revision y cercado mayores, con prueba durable obligatoria. |

`reconciliar_operacion_borrador_interna_v1` recibe estado, revision y cercado
observados. La maquina aplica estas reglas:

- un `confirmado` posterior exige `revision > observada` y
  `cercado = observado`;
- un `no_aplicado` obtenido desde un estado no terminal exige que revision y
  cercado sean mayores;
- releer un `no_aplicado` exige estado, revision y cercado exactamente iguales;
- un estado no terminal sin cambio exige los tres controles exactos;
- una reclamacion solo parte de `no_aplicado`, despues de vencer el
  arrendamiento, con prueba durable exacta y una concesion PDP nueva; eleva de
  nuevo revision y cercado;
- cualquier control obsoleto aborta con conflicto de serializacion.

El reloj del protocolo es `clock_timestamp()` de PostgreSQL truncado a
microsegundos. Un instante futuro enviado por el cliente no vence un
arrendamiento ni cambia el resultado durable.

### Confirmacion KMS y procedencia: `000004`

`000004` conserva el motor CAS de `000003`, pero ya no expone su forma
criptografica heredada. La frontera publica se divide en dos llamadas sobre la
misma conexion y la misma transaccion `SERIALIZABLE`:

1. `preparar_confirmacion_borrador_v1` reidentifica al `LOGIN` proyector,
   revalida PDP, diario, lease, material y CAS, y exige objetos cerrados para
   perfil, politica, evidencia de perfil, AAD, DEK envuelta, sobre AEAD,
   atestacion KMS y procedencia. Persiste los bytes tentativos y devuelve la
   preimagen exacta del cuerpo de recibo.
2. El conector KMS independiente revalida la atestacion y produce la
   acreditacion con firmas A/B. El proceso espera fuera de SQL y respeta la
   cancelacion de su contexto.
3. `confirmar_borrador_v1` relee la preparacion por `txid`, reconstruye cuerpo,
   acreditacion y preimagenes A/B, coteja todas las huellas y enlaces, y
   persiste acreditacion y recibo final antes de consumir la preparacion.
4. Un disparador diferido hace imposible confirmar la transaccion si falta la
   fase B; otros disparadores exigen que agregado, diario, auditoria y outbox
   tengan una acreditacion coincidente en el mismo `COMMIT`.

PostgreSQL gobierna el instante de cierre. El perfil de desarrollo fija, en
`presupuesto_confirmacion_kms_borrador_v1`, un `not-before` de 1000 ms y una
tolerancia final de 500 ms. El llamador no recibe un parametro de fecha, no
tiene `EXECUTE` sobre esa funcion y no puede ampliar el plazo mediante GUC.
Una configuracion productiva se versionara por migracion y queda sujeta a los
limites duros del esquema. Este instante coordina la transaccion; no sustituye
una TSA ni aporta por si solo tiempo cualificado.

SQL valida forma, canonicalizacion, referencias, vigencia, hashes y
atomicidad. No pretende verificar Ed25519 ni convertirse en autoridad KMS. El
adaptador criptografico valida las firmas antes de la fase B y
`verificar_recibo_borrador_v1` vuelve a cargar tras el `COMMIT` los bytes
durables y devuelve las dos preimagenes a un verificador independiente. Esa
funcion exige una cuenta `LOGIN` con una unica membresia directa en
`vec_bolsa_convocatorias_verificador_recibo`; la cuenta no puede ejecutar A/B
ni leer tablas.

Se almacenan AAD no sensible, referencias y versiones de clave, DEK envuelta,
nonce, texto cifrado, firmas y evidencias. Nunca se almacenan DEK en claro,
claves maestras, motivo en claro ni contenido del borrador sin cifrar. La
procedencia queda dentro de AAD, acreditacion y recibo. Los actos del perfil
`desarrollo` son `no_autoritativo` y `no_migrable`; una procedencia distinta
falla cerrada.

La migracion y sus vectores cruzados Go/PostgreSQL estan probados. La
habilitacion operativa completa sigue condicionada a T20E: recorrido A/B desde
Go, cancelacion durante la espera, reinicio, carreras, fallos de frontera y
relectura criptografica poscommit.

Ni el motivo en claro ni claves criptograficas disponen de columnas en el
almacen de borradores.

### Tablas y funciones principales

El vertical añade estas familias de tablas:

- autorizacion e intencion: `atestacion_pdp_borrador`, `material_borrador` y
  `uso_decision_borrador`;
- maquina durable: `diario_borrador_version`, `diario_borrador_actual`,
  `identidad_alias_borrador` y `prueba_desenlace_borrador`;
- sellado y agregado: `sellado_motivo_borrador`,
  `borrador_convocatoria_version` y `borrador_convocatoria_actual`;
- trazabilidad: `auditoria_borrador`, `auditoria_borrador_actual` y
  `outbox_borrador`;
- lectura: `uso_decision_lectura_borrador`,
  `auditoria_lectura_borrador` y `cursor_listado_borrador`.
- confirmacion KMS: `preparacion_confirmacion_kms_borrador`,
  `cifrado_kms_borrador` y `acreditacion_kms_borrador`.

Las fronteras runtime abiertas son `consultar_identidades_borrador_v1`,
`reservar_decision_borrador_v1`, `reconciliar_operacion_borrador_v1`,
`reclamar_reserva_borrador_v1`, `listar_borradores_v1` y
`obtener_borrador_v1`. El proyector recibe ademas
`preparar_confirmacion_borrador_v1` y `confirmar_borrador_v1`; el verificador
solo `verificar_recibo_borrador_v1`. Las variantes `*_interna_v1`, funciones
canonicas y validadores forman el nucleo tecnico de PostgreSQL; no son API
publica ni justifican conceder al proceso de aplicacion el rol propietario.

## Modelo durable de consulta `000001` y `000002`

- `version_convocatoria`: historia inmutable, bytes canonicos y SHA-256;
- `instancia_flujo_version`: flujo exacto ligado a esa version;
- `atestacion_autorizacion_version`: evidencia COSE versionada y append-only;
- `atestacion_autorizacion_actual`: puntero gobernado a activa o revocada;
- `uso_decision_consulta`: consumo unico y preimagen del efecto;
- `auditoria`: registros inmutables con huella anterior;
- `auditoria_actual`: checkpoint serializado de la cadena.

Todas las tablas tienen RLS habilitada y forzada. La unica politica positiva
exige que `current_user` sea el propietario `NOLOGIN`. Los roles runtime no
poseen privilegios de tabla, secuencia ni funciones internas; solo reciben
`USAGE` y los `EXECUTE` nominales enumerados en este documento. Las funciones
definidoras fijan `search_path` y no resuelven objetos controlados por el
llamador.

`000003` aplica ese cierre a sus dieciseis tablas nuevas. El propietario tambien
es `NOBYPASSRLS`; las tablas historicas rechazan `UPDATE`, `DELETE` y
`TRUNCATE`, y los punteros mutables solo admiten el avance gobernado previsto
por sus disparadores. Las ACL por defecto del propietario permanecen cerradas
para tablas, secuencias, funciones y tipos; `PUBLIC` y runtime tampoco reciben
`CREATE` sobre el esquema. Cada bloque DDL revoca inmediatamente a `PUBLIC`
todos los permisos sobre las funciones que acaba de crear y la migracion
concluye revocando todos los privilegios de tabla —incluidos `REFERENCES`,
`TRIGGER` y `MAINTAIN` de PostgreSQL 18—, columna, secuencia, tipo autonomo y
funcion que pudieran existir.

Las fronteras `SECURITY DEFINER` revalidan la decision y fijan
`search_path = pg_catalog, pg_temp` y UTC antes de delegar. Las funciones
`*_interna_v1` son `SECURITY INVOKER`, comprueban que el usuario efectivo sea
el propietario y no deben concederse a cuentas de aplicacion. La integracion
usa al superusuario de la base efimera para las pruebas internas, pero los
wrappers publicados se ejercitan con cuentas `LOGIN` reales sin pertenencia al
propietario.

## Roles

| Rol | Situacion actual |
|---|---|
| `vec_bolsa_convocatorias_propietario` | `NOLOGIN`; posee esquema y funciones. |
| `vec_bolsa_convocatorias_migrador` | Solo puede asumir al propietario durante migraciones. |
| `vec_bolsa_convocatorias_ejecutor_consulta` | `USAGE`; solo `listar_borradores_v1` y `obtener_borrador_v1`. |
| `vec_bolsa_convocatorias_proyector_gobierno` | `USAGE`; consulta multidentidad, reserva, reconciliacion, reclamacion y fases A/B. |
| `vec_bolsa_convocatorias_registrador_atestacion` | Reservado para el verificador COSE; sin acceso. |
| `vec_bolsa_convocatorias_verificador_recibo` | `USAGE`; solo relectura exacta poscommit. Sin tablas ni capacidad A/B. |

## Condiciones para habilitacion productiva

La consulta exacta V1 y el gobierno nominal V2 son habilitaciones distintas.
Los permisos SQL minimos del V2 ya estan definidos, pero una instalacion no es
productiva hasta aprovisionar las cuentas `LOGIN`, PDP/COSE y controles
operativos institucionales. Nunca se deben conceder tablas, secuencias,
funciones internas ni pertenencia al propietario.

Para la consulta exacta, una migracion posterior, independiente y revisable
debe:

1. registrar una atestacion solo desde una capacidad efimera emitida por el
   verificador COSE aislado;
2. comprobar suite, audiencia, clave versionada, ventanas, rotacion,
   revocacion y configuracion de confianza;
3. ligar decision completa, efecto `consulta-version-exacta` y despliegue;
4. evitar que el portal, el ejecutor o el registrador fabriquen filas;
5. probar replay, clave revocada, firma alterada, carrera de revocacion y ACL;
6. conceder, solo al final, `USAGE` y `EXECUTE` de la consulta exacta V1 si se
   decide publicar esa capacidad adicional.

No se debe insertar atestaciones mediante fixtures, triggers permisivos ni
credenciales compartidas en un entorno productivo.

Para las operaciones de borrador, la autorizacion V2 exige acciones separadas
para crear, actualizar, listar y consultar, clases de recurso, finalidades y
campos exactos. La puesta en servicio debe probar revocacion concurrente de
asignaciones, roles, politicas, claves y atestaciones. Consulta multidentidad y
reconciliacion son capacidades del proceso tecnico de gobierno, no del portal.
La confirmacion exige los pools separados, el conector KMS y la relectura
poscommit descritos arriba; que la funcion SQL exista no autoriza produccion.

## Requisitos operativos pendientes

Antes de autorizar produccion, Sistemas y Seguridad deben aceptar y verificar
como minimo:

1. **Identidades y conexiones.** Aprovisionar cuentas `LOGIN` nominativas o de
   carga de trabajo, DSN y certificados fuera del repositorio, mediante el
   gestor de secretos institucional. Deben existir TLS, rotacion, caducidad,
   separacion por entorno, limite de conexiones y minimo privilegio. Ningun
   proceso recibe credenciales del propietario o del migrador. Proyector y
   verificador usan cuentas, DSN y pools distintos; una cuenta con doble
   membresia falla cerrada.
2. **PDP y atestacion.** Implantar las politicas de accion y recurso, la
   emision durable de decisiones y el registrador COSE aislado. Deben quedar
   cubiertos alta, modificacion, listado, detalle, caducidad y revocacion de
   sesion, asignacion, rol, control de vigencia y catalogo de politicas, con
   pruebas negativas y de carrera en el entorno productivo equivalente.
3. **HSM/KMS.** Conectar la generacion y verificacion de HMAC de identidad y de
   motivo, y el cifrado autenticado gobernado por un perfil versionado. El
   conector debe gobernar AAD, DEK envuelta, referencias y generaciones de
   clave, rotacion, revocacion, atestacion, disponibilidad y recuperacion; las
   claves y el texto en claro nunca se almacenan en PostgreSQL ni en registros
   de aplicacion. Hasta atestar ese derivador, su codigo, despliegue, claves y
   canal forman parte expresa de la TCB y no se considera cerrada la identidad.
4. **Reloj.** Sincronizar y supervisar el reloj de los nodos PostgreSQL con la
   fuente horaria corporativa. La base opera en UTC y su reloj es autoritativo
   para decisiones, arrendamientos, caducidades y reconciliacion; debe alertarse
   cualquier deriva fuera de tolerancia.
5. **Observabilidad sin datos personales.** Medir rechazos `42501`,
   ambiguedades `21000`, conflictos de serializacion `40001`, reintentos,
   arrendamientos vencidos, reconciliaciones, desenlaces `no_aplicado`, retraso
   del buzon de salida, continuidad de auditoria, latencia y errores KMS/HSM y
   saturacion de conexiones. Las alertas deben conservar referencias de
   correlacion, no material canonico, identidades HMAC completas, motivos,
   principales ni sobres cifrados.

Hasta que estos controles tengan responsable, configuracion, prueba y acta de
aceptacion, el resultado sigue siendo **NO-GO para produccion**. En particular,
la migracion KMS y sus vectores no sustituyen el recorrido E2E, la gestion
institucional de claves ni la aceptacion de Sistemas y Seguridad.

## Prueba reproducible

```bash
./deploy/postgresql/bolsa_convocatorias/probar_integracion.sh
```

La prueba usa PostgreSQL 18.4 fijado por digest, instala las dependencias,
compila las migraciones, crea identidades LOGIN distintas y demuestra que:

- cuentas `LOGIN` de ejecutor, proyector y verificador alcanzan solo sus
  wrappers exactos, sin pertenecer al propietario; registrador y `PUBLIC`
  permanecen cerrados y ningun runtime lee o escribe tablas directamente;
- las diecinueve tablas aportadas por `000003` y `000004` tienen RLS forzada,
  y la historia no se puede modificar ni eliminar por las vias ordinarias;
- los vectores literales compartidos con Go acreditan los bytes canonicos y
  hashes de AAD, firmas KMS A/B y acreditacion, incluida la invariancia ante
  reordenacion y los rechazos por campos ausentes o alterados;
- el verificador poscommit usa una credencial independiente, no puede ejecutar
  A/B ni consultar tablas y falla cerrado si recibe una segunda membresia; la
  prueba SQL valida relaciones, hashes y preimagenes, pero no afirma verificar
  criptograficamente Ed25519 ni sustituye el recorrido A-B del adaptador Go;
- el nucleo interno heredado sigue probando atomicidad de CAS, auditoria,
  evento y recibo como regresion de `000003`;
- la consulta y reserva multidentidad conservan todos los alias de ventanas
  seriales y concurrentes `[g3,g2]`/`[g2,g1]`, devuelven aliases consultados y
  primaria completa, rechazan con `21000` solo una resolucion realmente
  ambigua y con `23505` una segunda pareja de la misma generacion;
- un fallo real inyectado al escribir el buzon de salida revierte agregado,
  consumo, auditoria, evento, recibo y diario en conjunto;
- dos conexiones `SERIALIZABLE` compiten por la misma identidad: existe un
  unico ganador, el perdedor acredita expresamente SQLSTATE `40001` y su
  reintento recupera la reserva sin duplicarla;
- la reserva sobrevive a un reinicio real del servidor y el reloj futuro del
  cliente no vence el arrendamiento;
- reconciliacion y reclamacion exigen revision, cercado, prueba durable y una
  decision PDP nueva, y rechazan controles obsoletos;
- el `down` de `000004` deja restaurado el contrato cerrado de `000003`; despues,
  el `down` protegido de `000003` se niega ante historia, la confirmacion
  destructiva explicita funciona solo en la base efimera y la prueba final
  acredita que ninguna de las dos migraciones deja tablas, funciones,
  disparadores o restricciones residuales.

PostgreSQL puede mostrar el error de serializacion de la conexion perdedora
durante la carrera; es evidencia esperada y el ejecutor solo termina con exito
si hubo exactamente un ganador. Los datos son sinteticos y viven en una base y
un contenedor efimeros. La mayor parte se revierte dentro de sus pruebas; la
carrera conserva historia temporal para demostrar reinicio y proteccion del
`down`, y el contenedor se destruye al finalizar. Nada de ello constituye una
fuente de datos alternativa para la aplicacion.
