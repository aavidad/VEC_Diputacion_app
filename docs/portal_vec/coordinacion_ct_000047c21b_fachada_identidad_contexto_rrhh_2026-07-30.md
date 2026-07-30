# Coordinación CT-000047C2.1b: fachada de Identidad para ContextoActor

Fecha: 30 de julio de 2026.

Estado: **en implementación; C2.1a integrada y verificada**.

## Responsabilidad única

Crear una fachada nominal de Identidad que revalide una autenticación y una
sesión ya obtenidas de la cápsula C1 y entregue a ContextoActor solo el
resultado mínimo necesario:

```sql
vec_identidad_sesiones_v1.
revalidar_contexto_corporativo_rrhh_v1(
    p_autenticacion_ref text,
    p_sesion_ref text
)
RETURNS TABLE (
    cuenta_ref text,
    metodo_observado text,
    garantia_observada text,
    identidad_valida_hasta timestamptz
)
```

No acepta cuenta, superficie, garantía, uso, perfil, organización ni
candidatos. No devuelve aserción, controles, huellas, principal, perfil u
organización.

`identidad_valida_hasta` es:

```sql
LEAST(
    sesion_valida_hasta,
    autenticacion_verificada_en + interval '15 minutes'
)
```

El inicio de vigencia ya queda comprobado dentro de Identidad y no cruza la
frontera. C2.5 volverá a leer el reloj después de sus propios bloqueos y
exigirá `clock_timestamp() < identidad_valida_hasta`.

## Write-set exclusivo

```text
deploy/postgresql/identidad_sesiones_v1/migraciones/
  000004_revalidacion_contexto_corporativo_rrhh_v1.up.sql
  000004_revalidacion_contexto_corporativo_rrhh_v1.down.sql
deploy/postgresql/identidad_sesiones_v1/
  probar_revalidacion_contexto_corporativo_rrhh_000004_pg18_4.sh
```

Dirección modifica por separado la documentación transversal.

## Contrato de la función

La función:

- pertenece a `vec_identidad_sesiones_v1_propietario`;
- es `LANGUAGE SQL`, `VOLATILE`, `CALLED ON NULL INPUT`,
  `SECURITY DEFINER`, `PARALLEL UNSAFE`;
- usa cuerpo SQL estándar `BEGIN ATOMIC` y nombres completamente cualificados;
- fija `search_path = pg_catalog` y `lock_timeout = '1s'`;
- exige transacción `SERIALIZABLE READ WRITE` sobre el primario;
- toma una vez, en modo compartido, el advisory
  `vec_contexto_actor_v1:rol-contexto-corporativo-rrhh-selector:v1`;
- reacredita firma, propietario, cuerpo, configuración y ACL de sí misma y de
  `revalidar_autenticacion_actor_v1(text,text)`;
- llama exactamente una vez al revalidador base y conserva sus locks hasta el
  final de la transacción exterior;
- devuelve exactamente una fila solo para superficie
  `interna_corporativa`, garantía `alto`, cuenta no privilegiada e igual a la
  ordinaria, referencias cruzadas exactas y ventana todavía vigente;
- convierte cero, varias, malformadas, cruzadas, revocadas o caducadas en la
  misma ausencia de fila;
- no captura cancelación, timeout, interbloqueo, `40001` ni errores de
  infraestructura.

El cuerpo `BEGIN ATOMIC` crea una dependencia catalogada real con el
revalidador base. `000003 down` debe fallar mientras exista `000004`.

## Precisiones de implementación

La migración puede instalarse antes de que C2.7 cree el LOGIN y el pool. Hasta
que exista exactamente un LOGIN miembro con las opciones exigidas, toda
llamada falla cerrada. La función no exige en ejecución la ACL literal inicial
de C2.1a: C2.5 y C2.8 añadirán autoridad funcional a ContextoActor. Sí exige
los atributos y la marca del selector, su topología nominal y que ni el
selector ni el LOGIN tengan acceso directo a Identidad.

Una función sustituida por su propietario no puede autenticar su propio cuerpo
desde ese mismo cuerpo. Por ello, la definición y la huella exactas se
acreditan en alta, postcondición, reentrada y retirada. Durante una llamada se
reacreditan propietario, ejecutor, configuración, ACL y dependencia con el
revalidador base; el arnés no atribuye a la autoinspección una garantía
imposible.

El aislamiento, escritura, primario y `role=none` se comprueban antes de
entregar referencias reales al revalidador. Un contexto inválido no puede
provocar una consulta con esas referencias. La prueba de dependencia dura de
`000003 down` se ejecuta en una instalación sin historia, para no confundir el
bloqueo por `pg_depend` con el bloqueo previo por cuentas o sesiones.

## Identidad técnica nominal

`session_user` debe ser un LOGIN real con:

- `INHERIT`, sin superusuario, `CREATEDB`, `CREATEROLE`, replicación ni
  `BYPASSRLS`;
- cero ajustes globales o por base;
- una única membresía directa al grupo
  `vec_contexto_actor_corporativo_rrhh_selector`;
- `ADMIN FALSE`, `INHERIT TRUE`, `SET FALSE`;
- cero membresías adicionales o transitivas y cero uso como grupo u otorgante;
- `current_setting('role') = 'none'`.

El selector debe conservar exactamente los atributos, comentario, ACL y
ausencia de membresías de C2.1a, y tener un único LOGIN miembro. Un segundo
LOGIN se rechaza; la rotación operativa debe drenar el pool antes de sustituir
esa membresía, no ampliar silenciosamente la autoridad.

## ACL y separación propietaria

Solo se añaden:

```sql
GRANT USAGE ON SCHEMA vec_identidad_sesiones_v1
    TO vec_contexto_actor_v1_propietario;
GRANT EXECUTE ON FUNCTION
    vec_identidad_sesiones_v1.
    revalidar_contexto_corporativo_rrhh_v1(text,text)
    TO vec_contexto_actor_v1_propietario;
```

`PUBLIC`, selector, LOGIN técnico, runtime ContextoActor, revalidador genérico,
PDP y Contratación temporal no reciben acceso. El propietario ContextoActor no
obtiene lectura de tablas ni ejecución de la fachada rica existente. La
llamada positiva solo ocurre desde una función `SECURITY DEFINER` propiedad de
ContextoActor, conservando el LOGIN técnico en `session_user`.

## Barreras y safe-down

Orden obligatorio:

```text
1. SHARED:
   vec_contexto_actor_v1:rol-contexto-corporativo-rrhh-selector:v1
2. EXCLUSIVE para 000004 up/down:
   vec_identidad_sesiones_v1:fachada-contexto-corporativo-rrhh:v1
```

C2.5 y C2.8 tomarán ambas en el mismo orden, la segunda en modo compartido,
mientras creen o retiren cualquier consumidor.

El `down` reacredita manifiestos, ACL y dependencias; rechaza todas las
sobrecargas de las fachadas C2.5/C2.8; revoca solo el `EXECUTE` creado; elimina
la función con `RESTRICT`; y revoca `USAGE` únicamente si no existe otro
consumidor nominal. No elimina historia ni usa `CASCADE`.

## Matriz PostgreSQL 18.4

El runner usa la imagen fijada por digest del proyecto e incluye:

- resultado exacto de cuatro columnas;
- nulos, malformados, inexistentes, cruzados y cardinalidad hostil;
- superficies externa/administrativa y garantías baja/sustancial;
- cuenta privilegiada o distinta de la ordinaria;
- caducidad de sesión y frontera exacta de quince minutos;
- revocación de sesión/cuenta y ambos órdenes de carrera;
- aislamiento incorrecto, solo lectura y recuperación;
- LOGIN, selector, atributos, ajustes, membresías y opciones alterados;
- segundo LOGIN, membresía transitiva, `SET ROLE` y llamada directa;
- ACL de `PUBLIC`, LOGIN, selector, runtime, revalidador y CT;
- propietario, lenguaje, seguridad, paralelismo, configuración, cuerpo,
  retorno y sobrecargas envenenados;
- dependencia dura contra `000003 down`;
- carreras C2.1a/C2.1b y C2.1b/C2.5 mediante locks observables, sin sleeps
  probatorios;
- reentrada cerrada, `up → down → up`, OID nuevo y base intacta;
- cancelación, timeouts, interbloqueo, `40001`, reconexión y reinicio;
- tres ejecuciones limpias acumuladas entre productor, revisión y dirección.

La prueba positiva usa una fachada proxy temporal propiedad de ContextoActor;
el LOGIN nunca ejecuta Identidad directamente. El proxy vive solo en el
contenedor efímero del arnés.

## Fuera de alcance

Este corte no selecciona perfil u organización, no crea recibo, pool, LOGIN,
PDP, composición CT, HTTP o web. C2.5 será la única consumidora SQL y C2.7
creará el pool exclusivo. No aumenta métricas funcionales ni habilita datos
reales o producción.
