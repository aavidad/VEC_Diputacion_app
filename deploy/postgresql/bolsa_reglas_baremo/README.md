# PostgreSQL: reglas gobernadas de baremo

Este paquete aporta el almacen autoritativo de
`reglasbaremo.VersionGobernadaReglasBaremo`. Es independiente de
`bolsa_baremacion`: aquella persistencia conserva decisiones sobre meritos de
una persona; esta conserva las reglas inmutables y su ciclo de gobierno. No se
mezclan agregados, permisos, historia ni plazos de conservacion.

La instalacion queda **cerrada para produccion**. Compila y se prueba de extremo
a extremo en PostgreSQL 18, pero ningun rol runtime recibe acceso al esquema o
a sus funciones. La puerta VEC-AD-2 existe en
`deploy/postgresql/autorizacion_atestada_v2`, pero este almacen aun no la
invoca ni liga su recibo al efecto en el mismo `COMMIT`. Abrirlo ahora
permitiria presentar como atestada una operacion que solo revalido la decision
nominal.

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
- consumo unico de autorizacion V2;
- consumo de prueba de transicion cuando corresponde;
- transaccion, auditoria encadenada y outbox.

Mientras la decision original siga vigente, una repeticion materialmente
identica devuelve el recibo anterior sin duplicar efectos. Esto prueba la
idempotencia mecanica, pero no es todavia la reconciliacion de un commit
ambiguo: si la decision o la sesion caducan, V1 falla antes de consultar el
recibo. La futura V2 tendra una consulta de reconciliacion separada,
minimizada y autorizada por la identidad completa del efecto. La misma
intencion con otro material se rechaza. Una revision o huella esperada
obsoleta produce conflicto de serializacion; no existe seleccion de «la
ultima version».

`obtener_version_exacta_v1(jsonb,jsonb,bytea,bytea)` solo acepta referencia,
version, huella de contenido, revision y huella de estado exactas. La consulta
historica consume una decision distinta y añade auditoria en la misma
transaccion. No resuelve alias ni consulta por fecha de insercion.

Ambas funciones exigen `SERIALIZABLE`, reloj de PostgreSQL, una solicitud con
menos de 30 segundos y `search_path` fijo. Permanecen sin `USAGE` ni `EXECUTE`
para los roles reservados.

## Contrato RBAC pendiente de publicar en application

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

Estas constantes aun no tienen constructores publicos en los puertos de Bolsa.
Por eso este corte no incorpora un adaptador Go: hacerlo ahora obligaria al
adaptador a inventar la solicitud que el PDP debe autorizar. El siguiente corte
debe publicar constructores de recurso/accion/finalidad en application o ports,
probar que el PDP produce exactamente ese contrato y solo entonces implementar
`RepositorioGobiernoReglasBaremo` y `ConsultaAutorizadaReglasBaremo`.

## Revalidacion V2 y composicion pendiente con VEC-AD-2

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

Esto acredita coherencia y vigencia, no quien firmo la decision. La puerta
`vec_autorizacion_atestada_v2.registrar_y_consumir_decision_v2_atestada` ya
registra y consume una capacidad ligada a la prueba criptografica. Sin embargo,
las funciones V1 de este paquete no la invocan, no reciben su
`consumo_ref`/`auditoria_ref` ni los ligan a `intencion_confirmada`. Por tanto,
no satisfacen todavia la composicion atomica exigida por VEC-AD-2.

El siguiente corte debe invocar la puerta existente y aplicar el cambio de
reglas dentro de la misma transaccion `SERIALIZABLE`, guardar sus referencias
exactas, reconciliar respuestas ambiguas y demostrar que el rollback del efecto
revierte tambien registro y consumo. Debe ofrecer una nueva funcion versionada;
no se debe conceder produccion a estas funciones V1. Solo tras ese cierre y los
restantes NO-GO propios de `autorizacion_atestada_v2` se concedera
exclusivamente:

- `USAGE` del esquema y `EXECUTE` de la futura funcion compuesta y versionada
  al ejecutor de gobierno;
- `USAGE` del esquema y `EXECUTE` de la futura consulta atestada y versionada
  al ejecutor de consulta;
- una lectura minimizada o funcion de entrega de outbox al publicador, nunca
  `SELECT` directo sobre tablas.

Las funciones V1 de este corte deben permanecer cerradas. No se debe conceder
DML ni acceso directo al registrador V2 central.

### Limitaciones deliberadas de V1

V1 es una prueba ejecutable del modelo de persistencia, no una frontera de
autoridad reutilizable mediante un `GRANT`. Antes de abrir el runtime, una
nueva funcion V2 compuesta debe cerrar conjuntamente estas brechas:

- SQL contrasta la huella de `version_canonica`, pero no reproduce el
  restaurador Go ni demuestra que sus proyecciones, prueba de transicion y
  bytes tengan la semantica de
  `VersionGobernadaReglasBaremo`. Las pruebas SQL usan bytes sinteticos para
  probar restricciones; la V2 exigira una restauracion exacta en el adaptador
  y una capacidad atestada que ligue bytes, proyecciones y efecto.
- V1 revalida la decision antes de adquirir los bloqueos del agregado y de la
  cabeza de auditoria. Una espera podria superar su vigencia. La V2 adquirira
  primero todos los bloqueos que puedan esperar, aplicara limites de
  `lock_timeout`/transaccion y repetira reloj, autoridad y confianza
  inmediatamente antes del efecto.
- V1 no es una API de reconciliacion tras respuesta ambigua. La V2 separara
  confirmacion y reconciliacion, usando una clave material exacta y una nueva
  autorizacion de consulta; nunca intentara repetir una decision consumida o
  caducada para averiguar si el primer `COMMIT` ocurrio.
- Las referencias V1 solo tienen una gramatica tecnica generica. La V2 usara
  espacios nominales y sufijos opacos acuñados por servidor para impedir que
  DNI, nombres u otros datos personales terminen en claves e indices.

Estas barreras se prueban como negativas de despliegue: mientras no exista V2,
los roles reservados conservan cero `USAGE`, `EXECUTE` y DML.

## Instalacion

Requiere el nucleo V2 de autorizacion:

1. `autorizacion/roles_up.sql`;
2. `autorizacion/migraciones/000001_autorizacion.up.sql`;
3. `ejecucion_documental_v4/migraciones_autorizacion/000002_vinculo_autenticacion_actor_actual.up.sql`;
4. `autorizacion/roles_v2_up.sql`;
5. `autorizacion/migraciones/000003_proyeccion_motivos_autorizacion_v2.up.sql`;
6. `autorizacion/migraciones/000004_registro_decisiones_solicitud_ligada_v2.up.sql`;
7. `roles_up.sql` de este paquete;
8. `migraciones_autorizacion/000001_revalidacion_reglas_baremo.up.sql`;
9. `migraciones/000001_almacen_reglas_baremo.up.sql`;
10. `migraciones/000002_operaciones_reglas_baremo.up.sql`.

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
- RLS forzada y ACL con identidades LOGIN reales;
- ausencia de `USAGE`, `EXECUTE` y DML runtime;
- negativa del down ante historia y conservación tras el intento fallido.
