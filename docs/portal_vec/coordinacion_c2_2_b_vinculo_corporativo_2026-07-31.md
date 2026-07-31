# Coordinación C2.2-B: vínculo corporativo RRHH versionado

Fecha: 31 de julio de 2026.

Estado: **contrato de implementación; no acredita todavía la capacidad**.

## Resultado único

C2.2-B añade a ContextoActor la historia append-only y el puntero actual del
vínculo corporativo para la superficie `interna_corporativa` y el uso nominal
`consulta_rrhh`. Liga versiones exactas de cuenta, persona, perfil, vínculo
ContextoActor base y organización, sin seleccionar candidatos ni exponer una
operación productiva.

La fuente normativa técnica es la
[decisión C2.2](decision_c2_2_organizacion_y_vinculo_corporativo_2026-07-30.md).
Ante una discrepancia prevalece esa decisión y la implementación se detiene.
Este contrato fija las decisiones estructurales que la decisión dejó para B;
no autoriza ampliar su alcance.

## Dependencias, autoridad y numeración

La migración reservada es:

```text
deploy/postgresql/contexto_actor_v1/migraciones/
  000004_vinculo_corporativo_rrhh_v1.up.sql
  000004_vinculo_corporativo_rrhh_v1.down.sql
```

Requiere una instalación exacta y completa de `000001`, `000002` y `000003`.
La organización C2.2-A está publicada en `54a8cde` y su cierre documental en
`2482301`; cualquier base distinta debe reacreditarse, no adoptarse.

ContextoActor es la única autoridad de estas tablas. Cuenta, persona, perfil
y vínculo base son proyecciones que ya pertenecen a su propio esquema; B no
lee ni escribe tablas de Identidad, PDP, Autorización, Contratación temporal o
Bolsa. Ningún dato procedente de un DTO, cabecera, cookie, almacenamiento web,
sesión del navegador o configuración sustituye esas referencias durables.

Las reservas posteriores permanecen intactas:

```text
contexto_actor_v1/migraciones/000005  publicación y revocación C2.3
contexto_actor_v1/migraciones/000006  recibo y selección privada C2.4
contexto_actor_v1/migraciones/000007  fachada y reconciliación C2.5
contexto_actor_v1/migraciones/000008  acreditación nominal C2.8
```

## Barreras y orden de bloqueo

Las barreras se toman siempre en este orden:

```text
A compartida = vec_contexto_actor_v1:migracion:acreditacion_uso:v2
B compartida = vec_contexto_actor_v1:organizacion-corporativa-rrhh:v1
C exclusiva  = vec_contexto_actor_v1:vinculo-corporativo-rrhh:v1
```

Tanto `up` como `down` toman `A SHARED → B SHARED → C EXCLUSIVE` antes de
bloquear relaciones. Así, `000002 down` y `000003 down` no atraviesan B, y
`000005` y consumidores posteriores podrán usar
`A SHARED → B SHARED → C SHARED → barrera propia` sin ciclos.

Tras las barreras, el alta inmoviliza las relaciones preexistentes en este
orden determinista:

1. `procedencias`, `proyeccion_cuenta_versiones` y `persona_versiones`, en
   `SHARE`;
2. `perfil_versiones` y `vinculo_contexto_versiones`, en
   `ACCESS EXCLUSIVE`, porque B les añade claves alternativas;
3. `organizacion_versiones` y
   `control_generacion_punteros_actuales_v2`, en `SHARE`;
4. los catálogos acreditados, en `SHARE`, hasta el `COMMIT`.

La retirada toma primero `ACCESS EXCLUSIVE` sobre
`vinculo_corporativo_actual` y `vinculo_corporativo_versiones`; después toma
los locks de las relaciones preexistentes en el orden anterior. Cualquier
escritor gobernado debe haber tomado antes la barrera C compartida. Un escritor
hostil que eluda la barrera espera, agota el `lock_timeout` o hace fallar la
fotografía; nunca permite retirar un catálogo distinto del acreditado.

Los catálogos inmovilizados explícitamente en `up` y `down` son:
`pg_authid`, `pg_auth_members`, `pg_db_role_setting`, `pg_database`,
`pg_class`, `pg_attribute`, `pg_attrdef`, `pg_index`, `pg_namespace`,
`pg_language`, `pg_collation`, `pg_proc`, `pg_type`, `pg_default_acl`,
`pg_description`, `pg_seclabel`, `pg_init_privs`, `pg_depend`,
`pg_shdepend`, `pg_constraint`, `pg_trigger`, `pg_policy`, `pg_inherits`,
`pg_rewrite`, `pg_am`, `pg_tablespace`, `pg_publication`,
`pg_publication_namespace`, `pg_publication_rel`, `pg_subscription_rel` y
`pg_statistic_ext`. Ningún catálogo consultado para decidir la adopción,
creación o retirada queda fuera de la fotografía protegida.

## Procedencias separadas

El vínculo corporativo y la organización son hechos gobernados distintos. B
no presume que procedan de la misma fuente o revisión. Cada fila histórica
conserva dos cuartetos separados:

- `procedencia_*`: procedencia propia del vínculo corporativo, con FK directa
  a la clave alternativa completa de `procedencias`;
- `organizacion_procedencia_*`: procedencia de la versión organizativa
  comprometida, con FK exacta a
  `organizacion_versiones_procedencia_uq`.

Ambas autoridades son exactamente `autoridad_maestra_acreditada`. Aunque en
una carga concreta coincidan los dos cuartetos, no se copian, infieren ni
confluyen. B no añade una clave alternativa que incluya la procedencia propia
del vínculo: sólo se creará cuando un consumidor concreto demuestre qué
combinación necesita como objetivo durable de una FK.

Tampoco duplica las procedencias de cuenta, persona, perfil o vínculo base.
Las FKs a sus versiones exactas comprometen ya las filas que contienen esas
procedencias; repetirlas aumentaría datos y posibilidades de contradicción sin
añadir autoridad.

## Claves alternativas aditivas en la base

`000004 up` añade, sin sustituir claves ni modificar filas, estas dos
restricciones exactas:

```text
perfil_versiones_persona_uq
  UNIQUE (perfil_ref, version, persona_ref)

vinculo_contexto_versiones_actor_uq
  UNIQUE (vinculo_ref, version, cuenta_ref, perfil_ref, persona_ref)
```

Sus índices de respaldo tienen los mismos nombres. Son alternativas
inmutables necesarias como objetivos de FKs compuestas; no son índices de
consulta. No se añade una FK nueva entre tablas históricas de `000001`, no se
añaden columnas, no se hace `UPDATE` o backfill y no cambia el significado de
ninguna fila anterior.

La primera clave prueba que la versión de perfil pertenece a la persona
referida. La segunda prueba que la versión del vínculo ContextoActor pertenece
a la misma cuenta, perfil y persona. Las FKs independientes a cuenta y persona
prueban además las versiones concretas de ambas; ninguna consulta sustituye
estas garantías durables.

## Historia `vinculo_corporativo_versiones`

Columnas, en este orden exacto:

| Nº | Columna | Tipo | Contrato |
| ---: | --- | --- | --- |
| 1 | `vinculo_corporativo_ref` | `text NOT NULL` | referencia opaca `vcr_` |
| 2 | `version` | `numeric(20,0) NOT NULL` | versión del vínculo corporativo |
| 3 | `cuenta_ref` | `text NOT NULL` | cuenta ContextoActor exacta |
| 4 | `cuenta_version` | `numeric(20,0) NOT NULL` | versión de cuenta |
| 5 | `persona_ref` | `text NOT NULL` | persona ContextoActor exacta |
| 6 | `persona_version` | `numeric(20,0) NOT NULL` | versión de persona |
| 7 | `perfil_ref` | `text NOT NULL` | perfil ContextoActor exacto |
| 8 | `perfil_version` | `numeric(20,0) NOT NULL` | versión de perfil |
| 9 | `vinculo_contexto_ref` | `text NOT NULL` | vínculo base exacto |
| 10 | `vinculo_contexto_version` | `numeric(20,0) NOT NULL` | versión del vínculo base |
| 11 | `organizacion_ref` | `text NOT NULL` | organización opaca exacta |
| 12 | `organizacion_version` | `numeric(20,0) NOT NULL` | versión organizativa |
| 13 | `organizacion_procedencia_ref` | `text NOT NULL` | procedencia de la organización |
| 14 | `organizacion_procedencia_version` | `numeric(20,0) NOT NULL` | revisión organizativa comprometida |
| 15 | `organizacion_procedencia_huella_sha256` | `text NOT NULL` | 64 hex minúsculas |
| 16 | `organizacion_procedencia_autoridad` | `text NOT NULL` | autoridad maestra exacta |
| 17 | `superficie` | `text NOT NULL` | sólo `interna_corporativa` |
| 18 | `uso` | `text NOT NULL` | sólo `consulta_rrhh` |
| 19 | `procedencia_ref` | `text NOT NULL` | procedencia propia del vínculo |
| 20 | `procedencia_version` | `numeric(20,0) NOT NULL` | revisión propia comprometida |
| 21 | `procedencia_huella_sha256` | `text NOT NULL` | 64 hex minúsculas |
| 22 | `procedencia_autoridad` | `text NOT NULL` | autoridad maestra exacta |
| 23 | `estado` | `text NOT NULL` | sólo `activo` o `revocado` |
| 24 | `vigente_desde` | `timestamptz(6) NOT NULL` | instante UTC finito |
| 25 | `vigente_hasta` | `timestamptz(6) NOT NULL` | instante UTC finito y posterior |

Todas las versiones usan el rango `1..18446744073709551615`. Las ventanas son
`[vigente_desde,vigente_hasta)`: el límite inicial pertenece y el final no.

### Restricciones históricas exactas

- `vinculo_corporativo_versiones_pk`, primaria
  `(vinculo_corporativo_ref, version)`;
- `vinculo_corporativo_versiones_actual_uq`, alternativa
  `(cuenta_ref, superficie, uso, vinculo_corporativo_ref, version)`, destinada
  exclusivamente a la FK completa del puntero;
- `vinculo_corporativo_versiones_cuenta_fk`, FK `MATCH FULL` de
  `(cuenta_ref, cuenta_version)` a la primaria de
  `proyeccion_cuenta_versiones`;
- `vinculo_corporativo_versiones_persona_fk`, FK `MATCH FULL` de
  `(persona_ref, persona_version)` a la primaria de `persona_versiones`;
- `vinculo_corporativo_versiones_perfil_persona_fk`, FK `MATCH FULL` de
  `(perfil_ref, perfil_version, persona_ref)` a
  `perfil_versiones_persona_uq`;
- `vinculo_corporativo_versiones_vinculo_contexto_fk`, FK `MATCH FULL` de
  `(vinculo_contexto_ref, vinculo_contexto_version, cuenta_ref, perfil_ref,
  persona_ref)` a `vinculo_contexto_versiones_actor_uq`;
- `vinculo_corporativo_versiones_organizacion_fk`, FK `MATCH FULL` de
  `(organizacion_ref, organizacion_version,
  organizacion_procedencia_ref, organizacion_procedencia_version,
  organizacion_procedencia_huella_sha256,
  organizacion_procedencia_autoridad)` a
  `organizacion_versiones_procedencia_uq`;
- `vinculo_corporativo_versiones_procedencia_fk`, FK `MATCH FULL` del
  cuarteto `procedencia_*` a la clave alternativa completa de `procedencias`.

Todas las FKs son inmediatas, no diferibles, con `ON UPDATE NO ACTION` y
`ON DELETE NO ACTION`. No se usa `CASCADE`, `SET NULL`, validación diferida ni
una restricción `NOT VALID`.

Los diecisiete checks históricos tienen estos nombres exactos, todos dentro
del límite de identificadores PostgreSQL:

```text
vinculo_corporativo_versiones_ref_ck
vinculo_corporativo_versiones_version_ck
vinculo_corporativo_versiones_cuenta_version_ck
vinculo_corporativo_versiones_persona_version_ck
vinculo_corporativo_versiones_perfil_version_ck
vinculo_corporativo_versiones_vinculo_contexto_version_ck
vinculo_corporativo_versiones_organizacion_version_ck
vinculo_corporativo_versiones_superficie_ck
vinculo_corporativo_versiones_uso_ck
vinculo_corporativo_versiones_organizacion_procedencia_ck
vinculo_corporativo_versiones_organizacion_autoridad_ck
vinculo_corporativo_versiones_procedencia_ck
vinculo_corporativo_versiones_procedencia_autoridad_ck
vinculo_corporativo_versiones_estado_ck
vinculo_corporativo_versiones_vigente_desde_ck
vinculo_corporativo_versiones_vigente_hasta_ck
vinculo_corporativo_versiones_ventana_ck
```

El primer check reutiliza `referencia_valida` con `vcr_`; su sufijo admite de
22 a 128 octetos `[A-Za-z0-9_-]`. Los checks de versión cubren las seis
versiones de entidad. Los dos checks compuestos de procedencia cubren además
cada versión, referencia y huella de procedencia; sus checks de autoridad
exigen por separado `autoridad_maestra_acreditada`. Los restantes fijan los
dos literales, el estado cerrado, los instantes finitos y la ventana
estrictamente creciente. No se crean checks anónimos.

No se duplican checks para `cuenta_ref`, `persona_ref`, `perfil_ref`,
`vinculo_contexto_ref` u `organizacion_ref`: las seis FKs históricas ya
acreditan filas cuyas referencias satisfacen las gramáticas publicadas.

Los únicos índices de la historia son los respaldos de
`vinculo_corporativo_versiones_pk` y
`vinculo_corporativo_versiones_actual_uq`, con esos mismos nombres. No se
anticipa un índice de selección, de estado, de vigencia ni de procedencia.

La historia usa exactamente:

- `historia_inmutable`, `BEFORE UPDATE OR DELETE FOR EACH ROW`, con
  `rechazar_mutacion_historia()`;
- `historia_no_truncable`, `BEFORE TRUNCATE FOR EACH STATEMENT`, con
  `rechazar_truncado()`.

Una revocación futura será una nueva versión. No hay borrado lógico mutable.

## Puntero `vinculo_corporativo_actual`

El puntero contiene exactamente cinco columnas, en este orden:

```text
cuenta_ref text NOT NULL
superficie text NOT NULL
uso text NOT NULL
vinculo_corporativo_ref text NOT NULL
version numeric(20,0) NOT NULL
```

`vinculo_corporativo_actual_pk` es la primaria exacta por
`(cuenta_ref, superficie, uso)`. La FK
`vinculo_corporativo_actual_version_fk` es `MATCH FULL`, inmediata y no
diferible, desde las cinco columnas a
`vinculo_corporativo_versiones_actual_uq`, con `NO ACTION` en actualización y
borrado. Así el puntero no puede combinar la coordenada de una cuenta con una
fila histórica perteneciente a otra.

Los tres checks `vinculo_corporativo_actual_superficie_ck`,
`vinculo_corporativo_actual_uso_ck` y
`vinculo_corporativo_actual_version_ck` aplican los mismos literales y rango
que la historia. La cuenta y la referencia corporativa quedan acreditadas por
la FK completa; no se duplican checks nominales.

El único índice del puntero es `vinculo_corporativo_actual_pk`. En particular,
**no** se crea `UNIQUE (vinculo_corporativo_ref)`: esa unicidad parcial no
impediría reutilizar históricamente una referencia en otra coordenada y puede
dificultar una sustitución atómica del puntero. C2.3 deberá bloquear por la
referencia corporativa y rechazar su publicación si cualquier versión previa
la asocia a otra `(cuenta_ref, superficie, uso)`. Esa regla de publicación no
se simula ni se expone en B.

El puntero puede apuntar a una versión revocada. Representa la última versión
conocida, no una autorización. La selección y reacreditación posteriores
comprobarán además punteros, estados, procedencias y vigencias de todos los
componentes; B no convierte esos checks en una selección implícita.

Reutiliza exactamente los tres triggers comunes de `000002`:

- `puntero_actual_no_truncable_v2`, `BEFORE TRUNCATE`, ejecuta
  `rechazar_truncado()`;
- `serializar_mutacion_punteros_actuales_v2`, `BEFORE INSERT OR UPDATE OR
  DELETE`, ejecuta la función homónima;
- `avanzar_generacion_punteros_actuales_v2`, `AFTER INSERT OR UPDATE OR
  DELETE`, ejecuta la función homónima.

Son triggers de sentencia, sin argumentos ni tablas de transición y
habilitados en modo ordinario. No se crea otro contador, reloj o cerrojo.

### Conteos catalogales congelados

Sobre PostgreSQL 18.4, el manifiesto acredita estos conteos junto con los
nombres y definiciones; un conteo aislado nunca permite adoptar un objeto:

| Elemento | Conteo exacto |
| --- | ---: |
| Columnas de historia | 25 |
| Restricciones de historia, incluidas 25 `NOT NULL` | 50 |
| Columnas de puntero | 5 |
| Restricciones de puntero, incluidas 5 `NOT NULL` | 10 |
| Restricciones en las dos tablas nuevas | 60 |
| Claves alternativas añadidas a tablas base | 2 |
| Restricciones propiedad de B | 62 |
| Restricciones nominales, sin los `NOT NULL` automáticos | 32 |
| FKs `MATCH FULL` | 7 |
| Índices nuevos, incluidos los respaldos de restricciones | 5 |
| Triggers de usuario | 5 |
| Triggers internos de FK ubicados en tablas nuevas | 16 |
| Triggers internos de FK totales aportados por B | 28 |
| Políticas RLS | 2 |
| Tipos fila y array protegidos | 4 |
| Relaciones TOAST e índices TOAST | 2 y 2 |

Los 32 nombres nominales se descomponen en dos claves base, dos primarias,
una clave alternativa histórica, siete FKs y veinte checks. Los triggers
internos se descubren por sus dependencias y forma exactas, no por un prefijo
ni por asumir que sus OID sean estables.

## Propiedad, RLS, ACL y minimización

Las dos tablas, sus tipos fila, restricciones, índices, políticas, triggers y
TOAST derivados pertenecen a `vec_contexto_actor_v1_propietario`. Las claves
alternativas añadidas a las tablas base y sus índices conservan ese mismo
propietario.

Ambas tablas nuevas tienen `ENABLE ROW LEVEL SECURITY` y
`FORCE ROW LEVEL SECURITY`. Cada una posee una sola política
`acceso_propietario_exacto`, `PERMISSIVE`, `FOR ALL`, limitada exactamente al
propietario en `USING` y `WITH CHECK`.

No se concede ningún privilegio. Se revoca expresamente todo acceso a tablas,
columnas y tipos fila para `PUBLIC` y `vec_contexto_actor_v1_runtime`. El rol
selector, sus LOGIN, PDP, Autorización, Contratación temporal y Bolsa
conservan cero acceso directo o efectivo. B no modifica el `USAGE` existente
del esquema ni crea roles o funciones ejecutables.

El modelo contiene sólo referencias opacas, versiones, procedencias, estado y
vigencia. No incorpora nombre, DNI/NIE, correo, CIF, código visible, unidad,
centro, puesto, jerarquía o dato RPT. Las correlaciones siguen siendo datos
personales seudonimizados: la minimización no sustituye la EIPD, la
categorización ENS, el análisis de riesgos ni la política aprobada de
conservación. Hasta esas aprobaciones sólo se usan datos sintéticos gobernados.

B no tiene textos de interfaz ni errores públicos, por lo que no añade claves
i18n ni HTML. Identificadores, comentarios y diagnósticos operativos se
mantienen en castellano coherente; la futura fachada será responsable de
traducir una denegación opaca a mensajes visibles localizados.

## Alta atómica y fallo cerrado

El `up` es un documento SQL transaccional, autónomo, sin metacomandos y
ejecutable byte a byte mediante `pgx.Conn.Exec`. Fija UTC y límites finitos de
bloqueo y sentencia. Antes de crear:

1. valida sólo superusuario, PostgreSQL compatible `>=18.0 y <19.0`, UTF-8 y
   configuración transaccional;
2. toma las tres barreras y los locks relacionales y catalogales definidos;
3. reacredita después de los locks el propietario, roles, esquema,
   restricciones, funciones, tablas, RLS, ACL y generación exactos de
   `000001..000003`;
4. acredita la ausencia total de objetos nominales de `000004`, incluidas las
   dos claves alternativas sobre tablas base;
5. hace `SET LOCAL ROLE` al propietario, añade las claves, crea las tablas y
   cierra inmediatamente sus ACL;
6. verifica todas las postcondiciones antes del `COMMIT`.

No hay `IF NOT EXISTS`, adopción de objetos, reparación silenciosa ni entrada
permisiva. Una discrepancia revierte el documento completo. Cada fichero SQL
permanece por debajo del tope duro de 800 líneas; si el contrato exacto no
cabe, se detiene B y se revisa esta descomposición, sin `\ir`, `\if`, `\gset`,
funciones temporales ni un empaquetador implícito.

## Retirada gobernada

El `down` es igualmente transaccional, autónomo, literal y sin `CASCADE`.
Permanece denegado salvo que el GUC de sesión tenga exactamente:

```text
vec.confirmar_retirada_vinculo_corporativo_rrhh_v1
= RETIRAR_VINCULO_CORPORATIVO_RRHH_V1
```

La confirmación no autoriza destruir evidencia. Después de bloquear y volver
a acreditar la fotografía, la retirada:

- rechaza cualquier fila en historia o puntero de B;
- no exige que `perfil_versiones`, `vinculo_contexto_versiones` u otra tabla
  base estén vacías y nunca modifica sus filas;
- exige el manifiesto exacto de tablas, tipos, columnas, checks, FKs, claves,
  índices, RLS, políticas, cinco triggers y TOAST;
- usa un manifiesto simbólico exacto y ajeno a OID para `000001..000004`, de
  modo que cualquier `000005` u otro consumidor posterior —también código
  dinámico sin una dependencia catalogal útil— cierre la retirada;
- acredita también forma, propietario, ACL y dependencias exactas de las dos
  claves alternativas añadidas a las tablas base;
- rechaza ACL, propietario, comentario, etiqueta, publicación, estadística,
  objeto o dependencia hostil, y cualquier consumidor posterior, incluida
  una `000005` sintética;
- elimina explícitamente, con `RESTRICT`, triggers, políticas y tablas de B;
- sólo después elimina con `RESTRICT` las dos restricciones alternativas de
  las tablas base, cuyos índices de respaldo desaparecen con ellas;
- deja filas, generación y catálogo de `000001..000003` equivalentes a su
  estado anterior a B.

La comprobación del GUC, ejecutor y versión anterior a los locks no decide
sobre un catálogo mutable. GUC, filas y manifiesto se vuelven a comprobar con
la fotografía inmóvil y los drops empiezan inmediatamente después. Todo error
produce rollback completo.

## Desacoplamiento previo del orquestador de pruebas

`probar_integracion.sh` tiene 790 líneas antes de B y ya se encuentra a diez
líneas del tope duro. B no puede ampliarlo ni trasladar esa deuda a otro
fichero monolítico. La minitarea B-S0 extrae su cuerpo base, sin alterar
semántica, a este runner autónomo nuevo:

```text
deploy/postgresql/contexto_actor_v1/
  probar_contexto_actor_v1_base_pg18_4.sh
```

El cuerpo base conserva por sí mismo imagen, contenedor, trampas, helpers,
aserciones y código de salida actuales. No llama a ningún otro runner. El
mensaje global de integración permanece al final de `probar_integracion.sh`,
nunca en un hijo antes de que terminen los demás. El main queda como un
orquestador corto que sólo calcula la raíz, activa fallo cerrado, invoca
directamente cada runner focal una vez y en orden y emite ese mensaje final.
No comparte contenedor, variables, funciones o estado mutable entre hijos.

B-S0 debe demostrar equivalencia antes y después: mismos runners ya
compuestos, en el mismo orden y una sola vez; mismas puertas base; mismo fallo
ante cada hijo; mismo código de salida; y limpieza sin residuos en éxito,
error, `INT` y `TERM`. La revisión compara también el diff movido y rechaza
cualquier cambio funcional encubierto.

### Disponibilidad inequívoca de PostgreSQL

B-S1, posterior a B-S0, corrige en una minitarea separada la espera del runner
base extraído y de `probar_organizacion_corporativa_v1_pg18_4.sh`. Ninguno
puede aceptar el servidor temporal que la imagen oficial abre sólo por socket
durante `initdb`. Ambos deben observar mediante TCP `127.0.0.1`, contraseña
explícita y consulta real la respuesta exacta `180004|false` tres veces
consecutivas. La espera es acotada y muestra `docker logs` al agotarse.

Los tres runners nuevos de B nacen ya con ese mismo patrón: PostgreSQL 18.4
exacto, primario, estable y alcanzable por TCP. B-S1 no modifica migraciones,
datos ni aserciones funcionales. B4 no puede componerse hasta que la extracción
B-S0 y la corrección B-S1 tengan GO independiente.

## Matriz de aceptación PostgreSQL 18.4

B divide la matriz en tres runners autónomos para mantener una responsabilidad
por fichero y respetar el límite de 800 líneas:

```text
deploy/postgresql/contexto_actor_v1/
  probar_vinculo_corporativo_rrhh_v1_estructura_adversarial_pg18_4.sh
  probar_vinculo_corporativo_rrhh_v1_concurrencia_preservacion_pg18_4.sh
  probar_vinculo_corporativo_rrhh_v1_pgx_pg18_4.sh
```

El runner estructural invoca como datos de prueba —no como runners— estos dos
SQL focales autónomos, también inferiores a 800 líneas:

```text
deploy/postgresql/contexto_actor_v1/pruebas_sql/
  vinculo_corporativo_rrhh_v1_estructura_catalogal.sql
  vinculo_corporativo_rrhh_v1_integridad_relacional.sql
```

El primero acredita columnas, restricciones, huellas, índices, ACL, RLS,
tipos, triggers y TOAST. El segundo contiene sólo fixtures sintéticos y prueba
cruces de actor y procedencia, límites, puntero, generación e inmutabilidad.
El shell conserva instalación, retirada adversarial y huellas de preservación.

Cada uno usa la misma imagen PostgreSQL 18.4 fijada por digest que C2.2-A, un
contenedor efímero propio y limpieza ante `EXIT`, `INT` y `TERM`. Ninguno llama
a otro runner ni depende del directorio o estado temporal dejado por otro. La
matriz completa acredita trece casos numerados:

1. instalación exacta `000001 → 000002 → 000003 → 000004`, atomicidad y
   rechazo de reentrada o adopción parcial;
2. nombres, orden, tipos, typmods, checks, FKs, claves, índices, propietarios,
   RLS forzada, política, ACL de tabla/columna/tipo, triggers y TOAST exactos,
   incluidas las dos claves aditivas base;
3. gramática `vcr_` en sus límites y rechazo de nulo, vacío, longitudes,
   caracteres o controles inválidos, más superficie y uso no literales;
4. rangos de todas las versiones, referencias cruzadas, perfil de otra
   persona, vínculo base de otro actor, organización o procedencias distintas,
   huellas, autoridad, estados, instantes no finitos, precisión y ventanas;
5. inserción válida de versiones activa y revocada, avance del puntero,
   vínculo completo con su coordenada y aumento de la generación común;
6. rechazo de `UPDATE`, `DELETE` y `TRUNCATE` de historia y de `TRUNCATE` del
   puntero, conservando datos y generación;
7. cero acceso para `PUBLIC`, runtime, selector, LOGIN sintéticos, PDP,
   Autorización, Contratación y Bolsa, incluso mediante columnas o tipos;
8. rechazo sin deriva de `000002 down` y `000003 down` mientras B existe;
9. rechazo de `000004 down` sin opt-in, con opt-in incorrecto, con filas, ACL,
   objetos, dependencias o consumidores hostiles y ante `000005` sintética;
10. retirada con B vacío pero tablas base pobladas, seguida de
    `000004 down → 000004 up`, sin cambiar filas, generación o catálogo de
    `000001..000003`;
11. carreras reales de instalación, retirada, DDL, inserción y avance de
    puntero, con espera acotada, sin interbloqueo y con rollback íntegro;
12. lectura de todos los bytes del `down` y ejecución sin preprocesado mediante
    `pgx.Conn.Exec`, incluida cancelación, rollback y saneamiento de la
    conexión dedicada antes de devolverla;
13. ausencia de contenedores, ficheros, conexiones o procesos residuales.

El runner estructural/adversarial posee los casos 1–10; el de
concurrencia/preservación posee el caso 11; y el runner pgx junto con la prueba
Go posee el caso 12. La limpieza del caso 13 es una postcondición obligatoria
de los tres y se vuelve a comprobar tras su composición. No se duplican los
casos para aparentar cobertura.

Cero o varias coincidencias, selección y denegación indistinguible no son una
operación de B y se prueban en C2.4. La matriz de B sólo demuestra que el
modelo durable no permite fabricar combinaciones cruzadas.

## Minitareas, productores y write-sets

Los write-sets son exclusivos. B-S0 y B1 pueden comenzar en paralelo tras el
GO de B0: el primero sólo mueve el lanzador existente y el segundo crea
migración y pruebas nuevas. B-S1 espera a B-S0 porque corrige el runner base
ya extraído. B2 y B3 pueden ejecutarse en paralelo después de B1 porque sólo
crean runners y pruebas distintos. La revisión puede paralelizar análisis de
solo lectura, nunca la edición del mismo corte.

| Minitarea | Dependencia y escritura | Entrega y responsable |
| --- | --- | --- |
| B0 | sólo este documento | productor documental; contrato revisado y commit documental antes de SQL |
| B-S0 | tras GO de B0; `probar_integracion.sh` y `probar_contexto_actor_v1_base_pg18_4.sh` nuevo | productor de pruebas; extracción mecánica equivalente, orquestador corto, suite anterior verde y revisión independiente |
| B-S1 | tras B-S0; runner base extraído y `probar_organizacion_corporativa_v1_pg18_4.sh` | productor de pruebas; espera TCP estable, logs en timeout, dos runners reales verdes y revisión independiente |
| B1 | tras GO de B0, paralela a B-S0; los dos SQL `000004`, `probar_vinculo_corporativo_rrhh_v1_estructura_adversarial_pg18_4.sh` y los dos SQL focales nuevos | productor SQL; migración reversible y casos 1–10 y 13 verdes en un commit autónomo; los cinco ficheros son una unidad porque alta, retirada y acreditación estructural forman el contrato reversible, pero cada uno queda por debajo de 800 líneas |
| B2 | tras B1; sólo `probar_vinculo_corporativo_rrhh_v1_concurrencia_preservacion_pg18_4.sh` nuevo | productor de concurrencia; caso 11 y 13 verdes, sin editar SQL ni el runner B1 |
| B3 | tras B1; prueba nueva `internal/vec/adapters/contextoactor/postgres/retirada_vinculo_corporativo_rrhh_v1_integracion_test.go` y `probar_vinculo_corporativo_rrhh_v1_pgx_pg18_4.sh` nuevo | productor de integración; casos 12 y 13 verdes; ningún Go productivo; puede avanzar en paralelo con B2 |
| B4 | tras B-S0, B-S1, B2 y B3; `probar_integracion.sh` y README de ContextoActor | integrador; invocación directa y exacta una vez de los tres runners B, matriz 1–13 y calidad global verdes |
| B5 | tras B4; sólo `docs/portal_vec/revisiones/revision_c2_2_b_vinculo_corporativo_2026-07-31.md` nuevo | revisor independiente; GO compuesto con `P0=P1=P2=0` |
| B6 | tras B5; sólo documentos transversales reservados por dirección | dirección; sincronización de estado, publicación y comprobación del CI remoto |

Productor y revisor no pueden ser la misma persona o agente. Tras cada
candidato pueden trabajar en paralelo dos revisores de lectura: uno de
integridad catalogal, relaciones y ACL, y otro de concurrencia, retirada y
TOCTOU. Un hallazgo vuelve al productor como minitarea focal con su prueba y
obliga a repetir ambas revisiones. B4, B5 o B6 nunca anticipan el resultado de
un corte técnico no demostrado.

## Fuera de alcance y criterio de cierre

C2.2-B no:

- publica ni revoca vínculos u organizaciones; eso pertenece a C2.3;
- resuelve candidatos, deniega cardinalidades ni registra el recibo 1:1; eso
  pertenece a C2.4;
- expone fachada o reconcilia un resultado incierto; eso pertenece a C2.5;
- añade contratos, puertos, adaptadores o composición Go, API, CLI, MCP, HTTP
  o web;
- autoriza mediante PDP ni integra efectos de Contratación temporal;
- importa AD, nómina, RPT o datos reales, ni crea jerarquías, unidades,
  centros, multitenencia o equivalencias de referencias legadas.

El cierre técnico exige B-S0, B-S1, B1, B2, B3 y B4 verdes y una revisión B5
independiente con `P0=P1=P2=0`. Sólo después dirección puede hacer B6. Hasta
entonces, y también después de B mientras falten C2.3–C2.11 y las aprobaciones
externas, producción permanece en **NO-GO**.

B es infraestructura preparatoria: su cierre no aumenta por sí solo las
métricas funcionales vigentes de Contratación `24/46`, O4-05 `3/5` o Bolsa
productiva `1/14`. No se declara una capacidad funcional, E2E o cumplimiento
normativo por disponer de las tablas.
