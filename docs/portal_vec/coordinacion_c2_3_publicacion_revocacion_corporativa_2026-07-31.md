# Coordinación C2.3: proyección corporativa automática

Fecha: 31 de julio de 2026.

Estado: **candidato completo; implementación y producción NO-GO**.

## Propósito

C2.3 materializa en ContextoActor organizaciones y vínculos corporativos
procedentes de una fuente aprobada. Su versión inicial es automática: no hay
formulario humano, autoinscripción, aprobador, acreditador ni selección por el
PDP.

Este documento desarrolla las
[decisiones de contexto](decision_contexto_corporativo_rrhh_ct_000047c2_2026-07-30.md)
y de
[organización y vínculo](decision_c2_2_organizacion_y_vinculo_corporativo_2026-07-30.md).
Sustituye cualquier coordinación C2.3 anterior, pero no altera la evidencia ni
el cierre técnico de C2.2.

## Autoridades y límites

- La fuente corporativa aprobada posee el hecho de origen.
- El intermediario de confianza específico de esa fuente acredita el evento y
  emite una capacidad breve, de un solo uso y audiencia exacta.
- `autorizacion_atestada_v3` posee el consumo durable de esa capacidad.
- ContextoActor posee exclusivamente la proyección, sus punteros y la prueba
  local de cada efecto.
- El PDP solo puede denegar usos posteriores; nunca descubre, publica, revoca
  ni elige una organización o un perfil.
- La auditoría común recibe una proyección mediante buzón transaccional; no es
  otra autoridad ni sustituye el registro probatorio local.

La vía humana queda fuera de V1. Cuando exista, será una minitarea distinta,
consumirá VEC-AD-3 mediante una fachada nominal y no podrá crear hechos que la
fuente maestra no acredite.

## Inventario y grafo

El inventario de 31 de julio acredita:

```text
contexto_actor_v1          000001..000004
autorizacion_atestada_v3   000001..000006
```

El siguiente número libre de `autorizacion_atestada_v3` es `000007`. Se
revalidará inmediatamente antes de escribirlo; una colisión bloquea el nodo.

```text
D0-A decisiones y reservas
  └─ D0-B este contrato
       ├─ F0 autorizacion_atestada_v3/000007, consumidor nominal de fuente
       └─ R0 roles técnicos y retirada
             └─ M5 contexto_actor_v1/000005, prueba local y buzón
                   └─ M6 /000006, organización +/−  ← F0
                         └─ M7 /000007, vínculo +/− ← F0
```

F0 y R0 son minitareas separadas. R0 no consume número de migración. M5, M6 y
M7 son las tres únicas migraciones C2.3. Ningún nodo mezcla selección, recibo
corporativo, PDP, API Go o vía humana.

## Frontera de confianza F0

F0 añadirá un consumidor nominal de capacidad de fuente al subsistema V3. El
perfil compromete, como mínimo:

- identificador y versión de la fuente;
- audiencia y acción exactas;
- tipo de efecto y recurso opaco;
- `operacion_ref` y referencia/huella del evento de origen;
- raíz, configuración y clave específicas de la fuente;
- emisión, caducidad corta y nonce de un solo uso;
- huella exacta del efecto solicitado.

La audiencia distingue los cuatro efectos C2.3. La fachada llamante entrega el
efecto esperado; el consumidor no devuelve una lista ni permite escogerlo.
Solo el propietario/fachada final de ContextoActor obtiene `EXECUTE`; los
logins técnicos y el runtime no ejecutan el consumidor directamente.

El intermediario puede reutilizar
`internal/vec/adapters/seguridad/verificacioncose` para la mecánica estricta de
COSE Sign1. No copia su código y no confunde esa verificación con confianza:
catálogo de fuentes, raíz, audiencia, revocación y vigencia son específicos de
la fuente. V3 reutiliza su consumo HMAC breve; ContextoActor no almacena COSE,
MAC, claves ni capacidad cruda.

F0 debe poder ejecutarse dentro de la transacción `SERIALIZABLE READ WRITE`
abierta por el adaptador y verificada por la fachada C2.3. Su consumo y el
efecto comparten `COMMIT`; cualquier error posterior revierte ambos. Si la
topología deja de permitirlo, F0 y C2.3 son `NO-GO`: no se sustituye por cola,
cabecera, recibo tardío o compensación.

Producción sigue en `NO-GO` hasta disponer de intermediario aprobado,
HSM/KMS, raíces y rotación gobernadas, protección de transporte, EIPD/ENS y
aprobaciones de RRHH, Sistemas y DPD.

## Persistencia M5

`000005` crea únicamente estructuras locales de ContextoActor:

1. registro probatorio append-only, único globalmente por `operacion_ref`;
2. asociación tipada operación–organización con FK completa a la historia de
   `000003`;
3. asociación tipada operación–vínculo con FK completa a la historia de
   `000004`;
4. buzón transaccional append-only, único por `evento_ref` y operación;
5. historial append-only de intentos de entrega del buzón.

Una operación organizativa tiene exactamente su asociación de organización.
Una operación de vínculo asocia su versión de vínculo y la versión de
organización comprometida. No se admite una asociación genérica
`tipo+referencia`, una FK parcial ni una cadena sin restricción durable.

El registro conserva referencias opacas, versiones, efecto, procedencia,
evento de origen, huellas, consumo V3 acreditado, instante autoritativo y
resultado. No crea una segunda cadena de auditoría. El buzón proyecta el
evento al servicio común con entrega al menos una vez e idempotencia por
`evento_ref`; un fallo de transporte nunca pierde el evento ya confirmado.

Historia, asociaciones y buzón rechazan `UPDATE`, `DELETE` y `TRUNCATE`. RLS
forzada es defensa adicional. `PUBLIC`, runtime, logins técnicos y demás
módulos no obtienen acceso directo a tablas, secuencias, tipos o funciones
internas.

## Cuatro fachadas nominales

M6 y M7 exponen solo:

```sql
publicar_organizacion_corporativa_desde_fuente_v1(...)
revocar_organizacion_corporativa_desde_fuente_v1(...)
publicar_vinculo_corporativo_desde_fuente_v1(...)
revocar_vinculo_corporativo_desde_fuente_v1(...)
```

Son `SECURITY DEFINER`, con propietario exacto y
`search_path=pg_catalog`. Reciben escalares validados, no JSON libre, perfil
elegido, organización de cabecera, cookie ni dato procedente del cliente web.

Como máximo existen dos grupos técnicos `NOLOGIN`: publicador y revocador.
Sus logins de servicio son exclusivos, sin `ADMIN/SET OPTION`, herencias,
atributos administrativos, `CREATE`, `TEMP` ni acceso a tablas. El publicador
ejecuta las dos altas; el revocador, las dos revocaciones. Ninguno puede
conceder membresías, aprobar la fuente o cambiar raíces.

R0 posee sus scripts `roles_up/down`, otorgantes exactos, membresías y prueba
de retirada. Toma el protocolo DDL de este documento. Su `down` deniega ante
miembros, grants, funciones o consumos dependientes y usa drops explícitos con
`RESTRICT`. M6 y M7 solo conceden o retiran `EXECUTE` a grupos R0 ya
acreditados; nunca crean roles implícitamente.

## Canon e idempotencia global

PostgreSQL deriva una única preimagen mínima para cada efecto a partir de los
escalares recibidos y las filas autoritativas bloqueadas. El contrato no
acepta `p_preimagen_huella`, canon o resultado calculado por el llamante.

El encuadre tiene dominio y versión, longitud antes de cada campo, orden fijo,
UTF-8 estricto, enteros decimales canónicos y UTC con microsegundos. Distingue
nulo de vacío. Incluye operación, efecto, referencia, versión esperada,
procedencia, evento de origen, huella de capacidad, vigencia o motivo y todas
las coordenadas corporativas pertinentes. Excluye secretos y su propia
huella. Los vectores dorados SQL prueban límites, campos cruzados y
reordenación.

`operacion_ref` es única para las cuatro fachadas, no por tabla ni por efecto.
El protocolo es:

- primera operación: consume capacidad, aplica efecto y registra prueba;
- replay con la misma preimagen: devuelve el mismo resultado sin consumir de
  nuevo ni escribir otro evento;
- misma operación con otro efecto, actor o preimagen: colisión cerrada;
- misma capacidad con otra operación: rechazo del consumidor V3.

## CAS y reglas funcionales

- Alta inicial exige versión esperada cero y ausencia de historia/puntero;
  crea versión uno.
- Avance exige puntero y versión actual exactos; crea `actual+1`, sin huecos ni
  desbordamiento.
- Una referencia revocada no se reactiva en V1. Un alta posterior con la misma
  referencia se rechaza aunque cambie la procedencia.
- Revocar exige puntero y versión exactos y copia las coordenadas de la fila
  bloqueada; no recibe sustitutos del llamante.
- La revocación puede ganar aunque la versión proyectada o la acreditación de
  origen que la creó hayan caducado, porque solo reduce autoridad. La nueva
  capacidad de revocación sí debe estar activa y vigente.
- El reloj autoritativo se lee una vez tras todos los bloqueos. Ventanas
  finitas usan `[desde,hasta)` y precisión `timestamptz(6)`.
- Un alta o avance exige fuente, capacidad, organización y dependencias
  activas y vigentes al reloj post-bloqueo.
- Vínculo fija `interna_corporativa` y `consulta_rrhh`; cruza mediante FKs
  compuestas cuenta, persona, perfil, vínculo base, organización y versiones.
- Revocar organización no reescribe vínculos; consumidores posteriores los
  deniegan al reacreditar la organización.

## Transacción de efecto

El adaptador ejecuta `BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE` antes de
invocar una única fachada. La función verifica `transaction_isolation` y
`transaction_read_only=off`, y deniega cualquier llamada sin ese contrato;
nunca intenta abrir ni confirmar transacciones. El flujo es:

1. el adaptador abre `SERIALIZABLE READ WRITE` y fija límites finitos;
2. la fachada acredita `session_user`, nivel y efecto nominal;
3. toma A→B→C→D compartidas, en sentencias separadas;
4. bloquea `operacion_ref` y recursos mediante advisory locks en orden fijo;
5. toma E exclusiva **antes** de cualquier lock de fila o puntero;
6. busca el registro global; replay exacto retorna, colisión deniega;
7. bloquea punteros e historias en orden organización→vínculo;
8. lee el reloj y ramifica: alta/avance validan CAS, estado, vigencia y
   dependencias; revocación valida puntero, CAS, estado activo y procedencia
   del antecedente sin exigir que su vigencia anterior siga abierta;
9. consume nominalmente la capacidad F0 y exige consumo nuevo;
10. inserta historia/CAS, prueba, asociaciones y buzón;
11. comprueba postcondiciones y devuelve el resultado;
12. el adaptador confirma inmediatamente el único `COMMIT`.

Las barreras son:

```text
A = migracion:acreditacion_uso:v2
B = organizacion-corporativa-rrhh:v1
C = vinculo-corporativo-rrhh:v1
D = proyeccion-corporativa-automatica:v1
E = mutacion_punteros_actuales:v2
```

Los advisory de operación se ordenan por bytes canónicos; para vínculo se
bloquea organización antes que vínculo. El trigger de generación de `000002`
retoma E de forma reentrante. Ningún camino toma una fila/puntero antes de E.

## Protocolo DDL sin ciclo

`up`, `down` y cambios de roles toman A→B→C compartidas y D exclusiva. D
exclusiva drena las cuatro fachadas. Después, **antes de E**, toman en orden
canónico los locks de relaciones y catálogos:

```text
1  relaciones base acreditadas de C2.2                  SHARE
2  organización: historia y puntero                     SHARE
3  vínculo: historia y puntero                          SHARE
4  relaciones C2.3 existentes                           SHARE o ACCESS EXCLUSIVE
5  catálogos acreditados, en el orden heredado C2.2     SHARE
6  E                                                     EXCLUSIVE
7  inventario, DDL y postcondición
```

Una relación modificada o retirada toma directamente `ACCESS EXCLUSIVE` en su
posición; no asciende desde `SHARE`. Una relación aún inexistente se omite y
se acredita su ausencia. Los nombres y OID se vuelven a comprobar bajo lock.

Este orden evita el ciclo `E → relación` frente a
`RowExclusive → trigger → E`: DDL no posee E mientras espera una relación. La
DML anterior termina; la posterior espera la relación antes de ejecutar su
trigger. Timeout, cancelación o deriva catalogal revierten toda la operación.

## `COMMIT` incierto y rollback

Tras pérdida de conexión, el reconciliador abre otra transacción, toma las
mismas barreras y lock de operación y deriva otra vez la preimagen desde los
mismos escalares. Si encuentra prueba exacta devuelve el resultado; una
preimagen distinta deniega. Si no existe fila una vez adquirido el lock, el
primer intento no confirmó y puede repetirse con la misma capacidad. Nunca se
aplica un efecto tardío desde una cola.

Las pruebas inyectan fallo después del consumo F0 y después de cada escritura.
Cada caso debe dejar sin cambios capacidad, historia, puntero, generación,
prueba, asociaciones y buzón. Tres ejecuciones limpias consecutivas deben
producir las mismas huellas e inventario.

## Retirada segura

Cada `down` usa el protocolo DDL, inventario exacto, drops explícitos con
`RESTRICT` y rollback total. No usa `CASCADE`. Deniega ante:

- cualquier prueba, asociación, evento o intento de entrega;
- historia/puntero que dependa del componente retirado;
- consumidores, objetos, propietarios, ACL u OID no inventariados;
- migración posterior o función nominal todavía dependiente.

La retirada sigue M7→M6→M5→R0. F0 no se retira mientras exista una fachada,
prueba o consumo C2.3. La comodidad operativa nunca justifica borrar
evidencia.

## Privacidad y observabilidad

Registro y buzón usan referencias opacas y minimización. El evento común no
incluye nombre, DNI, correo, títulos, trazas COSE, capacidad, MAC, clave ni
payload de fuente. Los datos de vínculo siguen siendo datos personales y
quedan limitados por propósito, conservación, cifrado de soporte, copias
protegidas y registro de acceso.

Logs y métricas no imprimen argumentos. Publican códigos estables, latencia y
correlación opaca. Las denegaciones externas no distinguen inexistencia,
caducidad, revocación, falta de rol o cruce de fuente. El detalle probatorio
queda solo en la frontera autorizada.

## Write-sets y revisiones

| Nodo | Escritura máxima | Prueba |
| --- | --- | --- |
| F0 | `autorizacion_atestada_v3/000007` up/down, README y prueba focal integrada | consumidor, replay, rollback y ACL |
| R0 | `roles_proyeccion_corporativa_v1_up/down.sql`, README, runner 1 y composición | otorgantes, membresías, ACL y retirada |
| M5 | `contexto_actor_v1/000005` up/down, README y ampliación del runner 1 | persistencia, FKs, buzón y retirada |
| M6 | `contexto_actor_v1/000006` up/down, runner 2 y composición | cuatro carreras org, autoridad y DDL |
| M7 | `contexto_actor_v1/000007` up/down, runner 3 y composición | vínculo, dependencias, carreras y DDL |

F0 amplía la integración existente de V3; no crea un cuarto runner C2.3.
R0 crea el runner 1 y M5 lo amplía después: su write-set compartido es
secuencial, nunca paralelo. ContextoActor tendrá como máximo tres runners y
`probar_integracion.sh` será el único compositor: los runners no se llaman
entre sí.

Cada nodo es un commit verificable con productor distinto de revisor. Antes de
confirmar: `diff --check`, enlaces, Gitleaks, límites de líneas, prueba focal,
integración PostgreSQL 18.4, inventario ACL/catálogo y revisión P0/P1/P2. Un
`NO-GO` se corrige y vuelve a revisar; no se integra parcialmente.

## Criterio de cierre

C2.3 solo queda técnicamente cerrada cuando F0, R0 y M5–M7 están confirmadas,
revisadas y reproducidas tres veces en PostgreSQL 18.4; las cuatro fachadas
acreditan autoridad, idempotencia, CAS, carreras, rollback, retirada y
`COMMIT` incierto. Eso no autoriza producción: los bloqueos externos de
fuente, confianza, HSM/KMS, ENS/EIPD y aprobación formal siguen vigentes.
