# O4-05: revisión de la infraestructura de cursores RRHH

Fecha: 26 de julio de 2026

Ámbito: C2-D1/`000038`, persistencia probatoria y reversión segura de cursores
del cuadro RRHH

Estado: GO técnico con dos revisiones independientes, integrado en `2f014c4`;
no autoriza producción

## Alcance cerrado

La migración `000038` añade la infraestructura durable necesaria para paginar
el cuadro de Contratación temporal sin conservar el token ni los filtros en
claro. No crea todavía la función exterior que emite o consume cursores, el
adaptador Go, la raíz productiva ni el recorrido HTTP.

El corte incorpora:

- `control_cursores_cuadro_rrhh`, singleton de versión;
- `alcance_acceso_rrhh`, prueba minimizada de identidad y ámbito;
- `familia_cursor_cuadro_rrhh`, corte global, filtros en huella y TTL;
- `cursor_cuadro_rrhh`, frontera opaca y cadena de páginas;
- `consumo_cursor_cuadro_rrhh`, consumo único y probado;
- `revocacion_familia_cursor_rrhh`, revocación durable.

Cuatro restricciones únicas adicionales enlazan el registro de acceso C2-B
con organización, ámbito, identidad, sesión, decisión, consumo VEC e instante.
La barrera global avanza `17→18` y la de consultas `1→2`.

## Privilegio criptográfico

El delta DBA concede únicamente `USAGE` sobre `public` y `EXECUTE` sobre
`public.gen_random_bytes(integer)` al propietario NOLOGIN de Contratación
temporal. La función debe:

- pertenecer realmente a la extensión `pgcrypto`;
- coincidir con la definición oficial fijada para PostgreSQL 18.4;
- conservar propietario, lenguaje, soporte y configuración esperados;
- carecer de acceso desde `PUBLIC` o roles ajenos;
- admitir la interoperabilidad mínima con el propietario NOLOGIN de Identidad,
  pero nunca con `GRANT OPTION`.

Un privilegio previo del propietario de Contratación se rechaza y se conserva:
el delta no se atribuye permisos que no sabe restaurar. La retirada comparte
el cerrojo de migraciones con `000038.up`; por ello nunca puede revocar la
capacidad después del preflight y dejar instalado un esquema inutilizable.

## Privacidad, identidad y paginación

El token futuro será Base64URL canónico de 32 bytes CSPRNG. Solo se persistirá
`SHA-256` de su representación ASCII. Los filtros se guardan únicamente como
huella ligada a un dominio versionado.

Cada familia queda ligada a organización, clase de ámbito, ámbito, actor,
perfil y versión, sesión y huella de sesión. Su vigencia es semiabierta y no
puede superar cinco minutos.

La página 2 debe usar exactamente el acceso e instante que originaron la
familia. Una página posterior debe usar exactamente el acceso e instante con
los que se consumió su cursor padre. El acceso de consumo de un cursor no
puede ser su acceso de emisión. Las restricciones únicas impiden un segundo
consumo y más de un hijo para la misma página.

Las seis tablas son de solo adición, salvo el singleton de control. Tienen
propietario exacto, RLS habilitada y forzada, política exclusiva del
propietario y cero acceso directo para el runtime.

## Reversión fail-closed

La reversión solo admite una instalación vacía, en barreras `18/2` y sin
fachadas o dependencias posteriores. Antes de eliminar objetos compara huellas
literales del catálogo PostgreSQL 18.4:

- relaciones, propietario, RLS, `FORCE`, opciones y reglas;
- 74 columnas, tipos, valores por defecto, generación y colación;
- 39 índices y sus propiedades;
- seis políticas y sus roles, `USING` y `WITH CHECK`;
- ACL efectivas de tabla y columna;
- 142 restricciones, incluida su validación y deferrabilidad;
- diez disparadores propios y 60 disparadores internos de integridad.

También rechaza reglas propias, vistas y FK externas, funciones posteriores,
disparadores inesperados y cualquier fila histórica. No usa `CASCADE`.

## Evidencia reproducida

El runner
[`probar_o4_05_cursores_rrhh_pg18_4.sh`](../../../deploy/postgresql/contratacion_temporal/probar_o4_05_cursores_rrhh_pg18_4.sh)
usa PostgreSQL 18.4 fijado por resumen, sin red ni puertos y con contenedor
efímero. Terminó tanto en la ejecución independiente como en la ejecución de
dirección con:

```text
O4-05 C2-D1 PostgreSQL 18.4: GO técnico
```

La matriz verificó:

- función criptográfica falsa, incluso añadida como miembro de `pgcrypto`;
- `GRANT OPTION`, roles LOGIN hostiles y privilegios previos;
- carreras entre roles y migración y entre dos `up` o dos `down`;
- página 2, página 3, consumo separado y ataques con accesos alternativos;
- TTL, ventana semiabierta, identidad, ámbito, huellas y prueba canónica;
- RLS, ACL, propietarios, inmutabilidad y ausencia del token en claro;
- cada una de las cinco historias bloqueando por separado la reversión;
- once derivas de catálogo: restricciones homónimas, disparadores propios e
  internos, regla, columna, índice, RLS, política, ACL y propietario;
- dependencias y barreras futuras;
- ciclo vacío final `18/2→17/1` y retirada exacta de los permisos propios.

`bash -n`, ShellCheck, `git diff --check` y Gitleaks quedaron verdes. Los nueve
ficheros de producto y pruebas respetan el límite duro de 800 líneas y no
dejaron contenedores ni temporales.

## Hallazgos corregidos durante la revisión

Las revisiones bloquearon sucesivamente versiones candidatas que:

1. comparaban nombres de restricciones, pero no su definición;
2. no conservaban privilegios criptográficos previos;
3. omitían disparadores internos y reglas propias en el safe-down;
4. aceptaban una función homónima o una falsa pertenencia a `pgcrypto`;
5. permitían `GRANT OPTION` al rol interoperable;
6. reutilizaban un acceso para emitir y consumir;
7. no ligaban la emisión de un hijo al consumo del padre.

Ninguna de esas versiones se integró. El GO corresponde exclusivamente a los
hashes congelados del commit `2f014c4`.

## Límites y siguiente corte

C2-D1 es infraestructura, no una capacidad exterior. Siguen abiertos:

- D2, funciones nominales de emisión, consumo, lectura y auditoría en un solo
  `COMMIT`, con revocación viva y avance monotónico;
- adaptador Go y conexión con la identidad corporativa;
- composición raíz, matriz TLS y E2E HTTP;
- categorización ENS, EIPD y aprobaciones organizativas.

Por ello el procedimiento conserva `19/46` tareas verificadas, el 41 % oficial
y tres de cinco hitos internos de O4-05.
