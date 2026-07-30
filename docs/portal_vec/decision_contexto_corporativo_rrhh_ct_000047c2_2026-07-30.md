# Decisión CT-000047C2: contexto corporativo RRHH

Fecha: 30 de julio de 2026.

Estado: **diseño cerrado; implementación pendiente y denegada por defecto**.

## Problema

La identidad productiva acredita una cuenta, pero no puede escoger un perfil
o una organización. ContextoActor acredita un par exacto cuenta–perfil, pero
su operación actual recibe el perfil ya elegido. El PDP necesita un perfil y
un ámbito exactos para autorizar, por lo que tampoco puede descubrirlos.

Usar asignaciones PDP para probar perfiles sería circular y permitiría sumar
permisos entre perfiles. Elegir el primero, el habitual o un valor de
configuración introduciría una autoridad no acreditada.

## Decisión

Antes del PDP se resuelve un único hecho corporativo:

```text
cuenta acreditada
+ superficie interna_corporativa
+ uso nominal consulta_rrhh
→ exactamente un perfil activo
+ exactamente una organización activa
```

La selección y el registro ContextoActor forman una sola operación
`SERIALIZABLE` de escritura. No existe una función o un puerto productivo que
devuelva candidatos ni una selección sin registrar.

El PDP actúa después sobre el par exacto. Puede denegar o reducir el ámbito,
pero nunca cambiar el perfil, elegir otra organización ni sumar permisos.

## Propiedad de las autoridades

- `vec_identidad_sesiones_v1` sigue siendo la única autoridad de sesión y
  cuenta.
- `vec_contexto_actor_v1` posee cuenta–persona–perfil, el vínculo corporativo,
  la organización acreditada y los recibos ContextoActor.
- Contratación temporal consume el recibo; no selecciona ni escribe tablas de
  ContextoActor.
- Autorización/PDP consume y reacredita el mismo recibo; no lo reconstruye.

La fachada pública pertenece a ContextoActor:

```sql
vec_contexto_actor_v1.
resolver_y_registrar_contexto_corporativo_rrhh_v1(...)
```

Dentro de la misma transacción llama exclusivamente a la fachada nominal de
Identidad:

```sql
vec_identidad_sesiones_v1.
revalidar_contexto_corporativo_rrhh_v1(
    p_autenticacion_ref,
    p_sesion_ref
)
```

La fachada de Identidad devuelve cuenta, método, garantía y vigencia ya
revalidados. No devuelve ni acepta perfil u organización. Solo ContextoActor
puede ejecutarla y ambos extremos reacreditan el `session_user` nominal.

## Modelo durable

ContextoActor añadirá:

1. historia append-only del vínculo
   cuenta–superficie–uso–perfil–organización;
2. puntero actual único por cuenta, superficie y uso;
3. recibo corporativo 1:1 con `registros_contexto`;
4. fachadas gobernadas de publicación y revocación;
5. acreditación nominal del uso del recibo.

El recibo corporativo compromete, como mínimo:

- `rca_` y huella del ContextoActor base;
- cuenta, principal, perfil y sus versiones;
- organización, superficie y uso;
- vínculo corporativo y versión;
- procedencia, versión, huella y autoridad;
- instante autoritativo y límite de vigencia;
- canon y huella SHA-256 calculados por PostgreSQL.

Las versiones usan `numeric(20,0)` en el rango `uint64`, los estados son
`activo` o `revocado` y todas las ventanas son `[desde,hasta)`.

No se permite `UPDATE`, `DELETE` ni `TRUNCATE` de historia o recibos. Un
`down` nunca elimina evidencia para facilitar una reversión.

## Operación atómica

Orden obligatorio:

1. acreditar el login técnico;
2. bloquear la operación idempotente;
3. revalidar autenticación y sesión mediante Identidad;
4. bloquear cuenta y todos los punteros candidatos en orden determinista;
5. exigir una única vinculación corporativa para la superficie y uso
   nominales;
6. resolver cuenta–persona–perfil y referencias asociadas;
7. leer el reloj autoritativo después de los bloqueos;
8. exigir estado, vigencia y procedencia maestra de todos los componentes;
9. insertar el recibo ContextoActor y el recibo corporativo 1:1;
10. confirmar ambos en el mismo `COMMIT`.

Cero o varias coincidencias producen la misma denegación. Quedan prohibidos
`LIMIT 1`, orden de preferencia, perfil predeterminado y enumeración PDP.

El replay de la misma operación y preimagen devuelve el mismo resultado. La
misma operación con otra preimagen colisiona cerrada. Un `COMMIT` incierto se
reconcilia solo con los mismos identificadores y huellas.

## Frontera Go

`domain` y `ports` no importan `httpseguridad` ni aceptan perfil,
organización, autenticación o sesión procedentes de un DTO.

La composición interna realiza:

```text
canal real
→ VincularCapsulaIdentidadPeticion
→ context.Context con clave privada
→ ExtraerCapsulaIdentidadPeticion
→ material efímero de revalidación
→ adaptador ContextoActor PostgreSQL
→ recibo corporativo durable
→ ContextoConsultaRRHH
```

La cápsula solo puede vincularla la misma instancia de
`ServicioIdentidad`, aportando el canal autenticado actual. La extracción
revalida sesión y cuenta y no recibe un canal, perfil u organización libres.

El material con referencias de autenticación y sesión vive únicamente
durante la llamada del adaptador. PostgreSQL vuelve a revalidarlo y coteja
cuenta, método y garantía; el valor Go nunca basta para conceder autoridad.

`NuevoContextoConsultaRRHH` deberá sustituir la cadena libre de organización
por el recibo corporativo opaco y cruzar todas sus huellas y versiones.

## Reacreditación

El recibo no es un vale permanente. La fachada:

```sql
vec_contexto_actor_v1.
acreditar_uso_registro_contexto_corporativo_rrhh_v1(...)
```

reutiliza la acreditación de ContextoActor y vuelve a comprobar el puntero
corporativo, versión, perfil, organización, procedencia, generación y
vigencia.

Se invoca:

1. al registrar durablemente la decisión PDP;
2. al abrir la transacción final de consulta;
3. de nuevo después de los locks del consumidor e inmediatamente antes de
   consumir capacidad, leer y registrar acceso.

Una revocación o avance de versión ganador produce denegación o rollback
total. Nunca selecciona una alternativa.

## Grafo de minitareas

| ID | Responsabilidad única | Dependencia |
| --- | --- | --- |
| C2.1 | Rol nominal y fachada de Identidad para ContextoActor | C1 |
| C2.2 | Retirada segura, organización y vínculo corporativo | C2.1b |
| C2.3 | Publicación y revocación gobernadas | C2.2 |
| C2.4 | Recibo 1:1 y función privada selección+registro | C2.2 |
| C2.5 | Fachada pública ContextoActor y reconciliación | C2.4 |
| C2.6 | Contrato Go opaco del recibo corporativo | C2.5 |
| C2.7 | Adaptador PostgreSQL y pool exclusivo | C2.6 |
| C2.8 | Acreditación nominal del recibo | C2.5 |
| C2.9 | Integración exacta en registro PDP | C2.8 |
| C2.10 | Integración exacta en efecto CT | C2.8 |
| C2.11 | E2E PostgreSQL 18.4 y revisión independiente | C2.7–C2.10 |

El inventario previo de C2.2 demostró que organización, historia, puntero,
generación común y retirada segura no caben en una sola minitarea. La
[decisión específica C2.2](decision_c2_2_organizacion_y_vinculo_corporativo_2026-07-30.md)
la divide en D0, S0.1, S0.2, A y B, mantiene la organización como entidad
versionada de primera clase y reserva `000003` y `000004` para sus dos
migraciones. C2.3 comenzará en `000005`.

Reservas iniciales:

```text
identidad_sesiones_v1/migraciones/000004
contexto_actor_v1/migraciones/000003..000008
autorizacion/migraciones/000011
contratacion_temporal/migraciones/000046
```

Cada migración se divide en componentes si se aproxima al límite de 800
líneas. Los números PDP y CT se reconfirmarán contra el árbol estable justo
antes de escribirlos.

## Privilegios

Un login dedicado consume una única fachada nominal mediante un grupo
`NOLOGIN`, sin `ADMIN OPTION`, `SET OPTION`, membresías adicionales ni
atributos administrativos. No obtiene lectura de tablas, funciones genéricas,
secuencias, `CREATE`, `TEMP`, `MAINTAIN`, parámetros ni asignaciones PDP.

RLS solo es defensa adicional; la frontera primaria son función nominal,
propietario exacto, `SECURITY DEFINER`, `search_path=pg_catalog` y ACL cerrada.

## Matriz PostgreSQL 18.4

Debe acreditar, al menos:

- cero, una y varias vinculaciones, con denegación indistinguible;
- futuro, caducidad y fronteras exactas;
- cuenta, persona, perfil, vínculo u organización revocados;
- superficie, uso, perfil, persona u organización cruzados;
- procedencia no maestra, versión, huella o puntero alterados;
- fallo del segundo `INSERT` con cero recibos;
- publicación o revocación concurrente contra selección;
- revocación entre PDP y efecto final;
- replay exacto, colisión de preimagen y `COMMIT` incierto;
- ausencia de enumeración o suma de perfiles en PDP;
- ACL, membresías, propietario, `prosecdef`, `search_path`, RLS y parámetros;
- intentos directos de tablas, función genérica, `SET ROLE`, `TRUNCATE` y
  `MAINTAIN`;
- reinicio, reconexión, cancelación, timeouts y tres ejecuciones limpias.

## Bloqueo externo

La estructura puede implementarse con datos sintéticos gobernados, pero no se
activa con datos reales hasta que RRHH, Sistemas y DPD aprueben:

- la fuente maestra y su procedimiento de carga;
- la semántica de perfil y organización;
- responsables de publicación y revocación;
- política de vigencia, conservación y auditoría;
- EIPD, categorización ENS y matriz PDP.

La ausencia de estas decisiones nunca se sustituye por configuración, memoria,
cabeceras, cookies o valores de demostración. Producción permanece en
**NO-GO**.
