# PostgreSQL: reglas gobernadas de baremo

Este paquete aporta el almacen autoritativo de
`reglasbaremo.VersionGobernadaReglasBaremo`. Es independiente de
`bolsa_baremacion`: aquella persistencia conserva decisiones sobre meritos de
una persona; esta conserva las reglas inmutables y su ciclo de gobierno. No se
mezclan agregados, permisos, historia ni plazos de conservacion.

La instalacion queda **cerrada para produccion**. La composicion V2 con la
puerta VEC-AD-2 ya existe y se prueba sobre PostgreSQL 18: consumo central,
cambio de reglas, CAS, evidencia, auditoria, outbox y recibo se confirman o se
revierten en el mismo `COMMIT`. Sin embargo, ningun rol runtime recibe todavia
acceso a sus funciones. Entre otras barreras, faltan el adaptador que restaure
el canon Go, el broker de capacidades, el seudonimizador confiable con su
llavero de finalidad y el verificador autoritativo de la evidencia de
transicion. Las funciones V1 tambien permanecen cerradas.

## Fuente de verdad canonica

`version_reglas_baremo.version_canonica` y su SHA-256 son la autoridad durable.
Las columnas de referencia, version, revision y estado son proyecciones
cotejables para indices, claves foraneas y CAS; no sustituyen los bytes.

SQL no copia el algoritmo canonico del dominio. El futuro adaptador debe:

1. obtener `RepresentacionCanonica()` y `HuellaSHA256()` de la version ya
   validada por application;
2. enviar esos bytes y sus proyecciones exactas en una transaccion
   `SERIALIZABLE`;
3. restaurar cualquier lectura mediante
   `reglasbaremo.RestaurarVersionGobernadaReglasBaremoConHuellaSHA256`;
4. cotejar contenido, revision, estado y `VinculoEstado()` antes del commit;
5. borrar de memoria las copias temporales de los bytes tras usarlas.

Por ello una prueba SQL puede ejercitar restricciones, atomicidad y
concurrencia con bytes sinteticos, pero no pretende validar el canon Go. El
runner ejecuta adicionalmente las pruebas del paquete de dominio.

## Modelo durable

- `contenido_reglas_baremo`: fija una sola huella para cada referencia y
  version de contenido;
- `version_reglas_baremo`: revisiones append-only con bytes canonicos, SHA-256,
  estado, operacion e intencion de origen;
- `estado_actual`: unico puntero mutable, avanzado por CAS exacto;
- `uso_decision`: consumo unico de `decision_ref` y preimagen del efecto;
- `uso_prueba_transicion`: consumo unico de la prueba exigida por cada cambio
  posterior al alta;
- `intencion_confirmada`: idempotencia exacta y recibo completo;
- `auditoria` y `auditoria_actual`: cadena SHA-256 serializada;
- `outbox`: evento inmutable creado en la misma transaccion;
- `recibo_cambio_atestado_v2`: prueba durable con claves foraneas al registro,
  consumo y auditoria VEC-AD-2 y, localmente, a version, uso de decision,
  intencion, auditoria y outbox;
- `configuracion_tenant`: fase inicial de inquilino unico, sin GUC controlable
  por el llamador como frontera de aislamiento.

Las tablas historicas rechazan `UPDATE`, `DELETE` y `TRUNCATE`. Los dos
punteros mutables rechazan borrado y truncado. Todas las tablas tienen RLS
habilitada y forzada; su unica politica positiva exige al propietario
`NOLOGIN`. Los roles runtime no reciben privilegios de tabla, secuencia, tipo,
esquema o funcion.

## Operaciones cerradas

`confirmar_cambio_v1(jsonb,jsonb,bytea,bytea,bytea)` confirma atomicamente:

- alta, publicacion, activacion, sustitucion, retirada o descarte;
- idempotencia de la intencion;
- CAS del estado esperado;
- nueva revision append-only y avance del puntero;
- consumo local unico de la referencia de decision nominal V2;
- consumo de prueba de transicion cuando corresponde;
- transaccion, auditoria encadenada y outbox.

Mientras la decision original siga vigente, una repeticion materialmente
identica devuelve el recibo anterior sin duplicar efectos. Esto prueba la
idempotencia mecanica, pero no es todavia la reconciliacion de un commit
ambiguo: si la decision o la sesion caducan, V1 falla antes de consultar el
recibo. V2 dispone de una consulta de reconciliacion separada y minimizada por
la identidad completa del efecto. La misma intencion con otro material se
rechaza. Una revision o huella esperada obsoleta produce conflicto de
serializacion; no existe seleccion de «la ultima version».

`obtener_version_exacta_v1(jsonb,jsonb,bytea,bytea)` solo acepta referencia,
version, huella de contenido, revision y huella de estado exactas. La consulta
historica consume una decision distinta y añade auditoria en la misma
transaccion. No resuelve alias ni consulta por fecha de insercion.

Ambas funciones V1 exigen `SERIALIZABLE`, reloj de PostgreSQL, una solicitud con
menos de 30 segundos y `search_path` fijo. Permanecen sin `USAGE` ni `EXECUTE`
para los roles reservados.

## Contrato RBAC y plan V2

La frontera SQL reserva estas combinaciones cerradas:

| Operacion | Accion |
|---|---|
| Alta | `bolsa.reglas_baremo.borrador.crear` |
| Publicacion | `bolsa.reglas_baremo.publicar` |
| Activacion | `bolsa.reglas_baremo.activar` |
| Sustitucion | `bolsa.reglas_baremo.sustituir` |
| Retirada | `bolsa.reglas_baremo.retirar` |
| Descarte | `bolsa.reglas_baremo.descartar` |
| Consulta exacta | `bolsa.reglas_baremo.version.consultar` |

El tipo de recurso es `version_reglas_baremo_gobernada`. La referencia es
`reglas-baremo:<huella_estado_sha256>`. Los cambios usan la finalidad
`gobierno_reglas_baremo` y los campos `auditoria`, `estado_reglas_baremo` y
`salida_eventos`; la consulta usa `consulta_gobierno_reglas_baremo` y solo
`estado_reglas_baremo`. La garantia minima y observada es `alto`, sobre la
superficie interna o la administracion privilegiada coherente con la cuenta.

El paquete
`internal/modules/bolsa/application/gobiernoreglasbaremo` fija estas
combinaciones mediante un plan V2 tipado y canonico. El plan no es una
autorizacion ni una orden ejecutable: compromete los bytes de la version, CAS,
evidencia, motivo, contexto, efecto, sujeto seudonimizado y componentes que la
frontera ejecutora debe confirmar. El adaptador PostgreSQL real sigue pendiente
y no podra construir libremente ninguno de esos campos.

## Composicion V2 con VEC-AD-2

`vec_autorizacion.revalidar_decision_reglas_baremo_v1` coteja dentro de la
transaccion:

- bytes y SHA-256 de la decision V2 registrada;
- accion, recurso, finalidad, correlacion y campos exactos;
- motivo catalogado vigente;
- asignacion actual, version y control de vigencia del rol;
- concesion RBAC sin comodines ni obligaciones;
- revision y manifiesto completo de politicas ABAC;
- sesion y vinculo de autenticacion actuales;
- garantia `alto`, metodo no demostrativo y superficie interna coherente.

Esto acredita coherencia y vigencia nominal, no quien firmo la decision. La
migracion `migraciones_atestacion/000001_composicion_vec_ad2_reglas_baremo`
crea el puente minimo con VEC-AD-2 y retira al consumidor generico las puertas
centrales crudas. La migracion `000003_gobierno_atestado_v2` incorpora dos
fronteras nuevas:

- `confirmar_cambio_atestado_v2`: exige una transaccion nueva
  `SERIALIZABLE READ WRITE`, coteja el plan, decision y capacidad cerrados,
  toma en orden fijo todos los bloqueos locales de tablas, agregado, intencion,
  resultado, evidencia, CAS y auditoria antes de VEC-AD-2, vuelve a cotejar el
  reloj autoritativo despues del consumo y materializa todo el grafo con un
  recibo durable;
- `reconciliar_cambio_atestado_v2`: recupera solamente ese recibo mediante la
  identidad exacta de decision, efecto, plan y nonce. No repite una decision
  de un solo uso ni vuelve a ejecutar el cambio. Es una primitiva mecanica sin
  `GRANT`: produccion debera envolverla en un caso de uso que vuelva a autorizar
  identidad, sesion y permiso actuales y que no exponga la funcion directamente.

Un helper central `VOLATILE` y sin permisos runtime permite observar dentro
del mismo statement el registro, consumo y auditoria que acaba de crear la
puerta central. Las claves foraneas del recibo prueban el enlace central y
local. El down ordinario se niega a retirar V2 si existe un recibo.

Solo cuando se cierren, entre otros, los NO-GO del broker de capacidades,
seudonimizador y HSM/KMS, restaurador Go, resolvedor durable de alcance y misma
fila, verificador de evidencia, perfiles centrales por modulo y ancla externa
antirretroceso, incluido el envoltorio autorizado de reconciliacion, se
concedera exclusivamente:

- `USAGE` del esquema y `EXECUTE` de la funcion compuesta y versionada
  al ejecutor de gobierno;
- `USAGE` del esquema y `EXECUTE` de la futura consulta atestada y versionada
  al ejecutor de consulta;
- una lectura minimizada o funcion de entrega de outbox al publicador, nunca
  `SELECT` directo sobre tablas.

Las funciones V1 deben permanecer cerradas. Ningun LOGIN ni rol runtime recibe
DML o acceso directo al registrador, reconciliador o helper centrales. Solo el
propietario `NOLOGIN` del modulo puede invocarlos como llamada interna de la
funcion `SECURITY DEFINER`. El perfil central generico usado por la prueba se
sustituira por perfiles especificos de modulo y pool antes de abrir produccion.

### Limitaciones de V1 y NO-GO restantes de V2

V1 es una prueba ejecutable del modelo de persistencia, no una frontera de
autoridad reutilizable mediante un `GRANT`. V2 corrige su composicion atomica,
bloqueos, referencias y reconciliacion, pero sigue cerrada hasta resolver:

- SQL contrasta la huella de `version_canonica`, pero no reproduce el
  restaurador Go ni demuestra que sus proyecciones, prueba de transicion y
  bytes tengan la semantica de
  `VersionGobernadaReglasBaremo`. Las pruebas SQL usan bytes sinteticos para
  probar restricciones; el adaptador exigira una restauracion exacta
  y una capacidad atestada que ligue bytes, proyecciones y efecto.
- V1 revalida la decision antes de adquirir los bloqueos del agregado y de la
  cabeza de auditoria. V2 preadquiere todos los bloqueos locales de tablas y
  datos, limita la espera y vuelve a cotejar el instante autoritativo despues
  del consumo central.
- V1 no es una API de reconciliacion tras respuesta ambigua. V2 separa
  confirmacion y reconciliacion mediante la identidad material exacta y nunca
  reintenta la decision consumida. Aun debe probarse la recuperacion despues de
  un `COMMIT` desde otra sesion.
- Las referencias V1 solo tienen una gramatica tecnica generica. V2 ya usa
  espacios nominales y sufijos opacos; falta que el seudonimo HMAC sea acuñado
  y cotejado por un componente confiable con la clave fuera del proceso.

Estas barreras se prueban como negativas de despliegue: aunque V2 ya existe,
los roles reservados conservan cero `USAGE`, `EXECUTE` y DML.

## Instalacion

Requiere el nucleo V2 de autorizacion:

1. `autorizacion/roles_up.sql`;
2. `autorizacion/migraciones/000001_autorizacion.up.sql`;
3. `ejecucion_documental_v4/migraciones_autorizacion/000002_vinculo_autenticacion_actor_actual.up.sql`;
4. `autorizacion/roles_v2_up.sql`;
5. `autorizacion/migraciones/000003_proyeccion_motivos_autorizacion_v2.up.sql`;
6. `autorizacion/migraciones/000004_registro_decisiones_solicitud_ligada_v2.up.sql`;
7. `confianza_atestacion_v2/roles_up.sql` y su migracion `000001`;
8. `autorizacion_atestada_v2/roles_up.sql`, sus vinculos con autorizacion y
   confianza y su migracion `000001_registro_consumo_atestado_v2.up.sql`;
9. `roles_up.sql` de este paquete;
10. `migraciones_autorizacion/000001_revalidacion_reglas_baremo.up.sql`;
11. `migraciones/000001_almacen_reglas_baremo.up.sql`;
12. `migraciones/000002_operaciones_reglas_baremo.up.sql`;
13. `migraciones_atestacion/000001_composicion_vec_ad2_reglas_baremo.up.sql`;
14. `migraciones/000003_gobierno_atestado_v2.up.sql`.

La retirada usa el orden inverso. El down del almacen se niega si existe
historia. Una destruccion formal e irreversible requiere establecer durante
esa sesion:

```text
vec.confirmar_destruccion_bolsa_reglas_baremo=
DESTRUIR_HISTORIA_BOLSA_REGLAS_BAREMO_IRREVERSIBLE
```

Esa confirmacion nunca debe formar parte de un despliegue ordinario.

## Prueba PostgreSQL 18

```bash
./deploy/postgresql/bolsa_reglas_baremo/probar_integracion.sh
```

El runner usa PostgreSQL 18.4 fijado por digest y verifica:

- instalación, retirada y reinstalación;
- alta, sucesión, activación y retirada;
- CAS obsoleto, idempotencia, replay y una carrera con un solo ganador;
- alteración de bytes/huella, consulta histórica exacta e inmutabilidad;
- consumos, auditoría encadenada y outbox atómicos;
- alta y publicacion V2 con consumo VEC-AD-2 y recibo enlazado por claves
  foraneas;
- rechazo precentral de CAS y sujeto cruzados, y segundo intento sin replay;
- fallo local posterior a VEC-AD-2 con rollback conjunto del consumo y del
  efecto;
- caducidad de evidencia durante VEC-AD-2, cotejada con una lectura fresca del
  reloj al regresar y revertida conjuntamente;
- reconciliacion exacta y rechazo de un nonce distinto;
- RLS forzada y ACL con identidades LOGIN reales;
- imposibilidad de alcanzar la puerta central cruda desde el LOGIN modular;
- ausencia de `USAGE`, `EXECUTE` y DML runtime;
- negativa del down V1 y V2 ante historia y conservacion tras el intento
  fallido.
