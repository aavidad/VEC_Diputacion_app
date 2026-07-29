# Relevo de sesión: autorización V3 y CT-000043

Fecha de corte: 29 de julio de 2026, zona horaria Europe/Madrid.

## Estado confirmado y publicado

La única rama integradora es:

```text
rama: integracion/ct-o4-04e-20260726
worktree: /home/alberto/Trabajo/VEC_Diputacion_app/.worktrees/ct-stable-docs
HEAD local/remoto: 67b0fb0acf0c49ae5c8b5f57e8feac387d9c274f
```

El árbol está limpio y coincide con GitHub. Las tres últimas ejecuciones de la
rama terminaron correctamente:

| Ejecución | Commit | Resultado |
| --- | --- | --- |
| `30400689200` | `67b0fb0` | verde |
| `30399280237` | `e8d2afc` | verde |
| `30396784705` | `ca8a517` | verde |

No se tocó producción ni se hizo despliegue.

## Dependencia VEC-AD-3: prueba autoritativa de consumo

La necesidad apareció durante CT-000043. Contratación temporal debe persistir
la prueba completa de la autorización consumida, pero no puede leer tablas de
otro módulo ni aceptar referencias de auditoría aportadas por el llamador. La
solución se implementó de forma aditiva en el módulo propietario de la
autoridad:

```text
rama: agent/aut-v3-prueba-consumo-20260729
worktree:
  /home/alberto/Trabajo/VEC_Diputacion_app/.worktrees/
  aut-v3-prueba-consumo-20260729
base: 67b0fb0
estado: cinco archivos locales, sin commit ni push
```

Archivos:

```text
deploy/postgresql/autorizacion_atestada_v3/
  README.md
  migraciones/000006_prueba_consumo_consultas_rrhh_v3.up.sql
  migraciones/000006_prueba_consumo_consultas_rrhh_v3.down.sql
  probar_prueba_consumo_consultas_rrhh_v3_pg18_4.sh
  pruebas_sql/prueba_consumo_consultas_rrhh_v3.sql
```

El productor emitió `GO`: dos funciones nominales llaman primero a la
revalidación de `000005`, releen con `FOR SHARE` atestación, consumo y auditoría
propias, ligan otra vez las diez piezas de entrada y devuelven exactamente:

```text
decision_ref
efecto_ref
huella_efecto_sha256
consumo_huella_sha256
auditoria_ref
auditoria_huella_sha256
consumida_en
revalidada_en
```

El productor ejecutó dos veces el runner PostgreSQL 18.4 y la regresión
histórica. También informó verdes las puertas Go, Bash, ShellCheck, Markdown,
tamaños, diferencias y Gitleaks. El verde del caso de revocación concurrente
era, sin embargo, un falso positivo por timeout.

Dirección reprodujo antes del corte:

- `git diff --check`;
- `bash -n`;
- ShellCheck, excluyendo únicamente `SC2154` por variables inyectadas en el
  runner;
- Gitleaks sobre el directorio completo: cero hallazgos;
- límite de 800 líneas: máximo de 520.

La revisión independiente emitió `NO-GO P1` y lo reprodujo en PostgreSQL 18.4:

- el runner oficial retiene el lock durante 1,2 segundos mientras la función
  usa `lock_timeout=1s`; el rechazo observado demuestra el timeout, no una
  revocación concurrente correctamente serializada;
- al repetir exactamente el caso reduciendo solo la espera a 0,55 segundos, la
  revocación confirmó y la función devolvió éxito con las ocho piezas;
- `000005` y `000006` trabajan con el snapshot de la transacción
  `SERIALIZABLE`: `000005` lee revocaciones antes, se bloquea en la
  revalidación viva y su segunda consulta no ve la revocación confirmada
  después; los `FOR SHARE` posteriores de `000006` no cierran esa carrera.

No confirmar ni integrar la dependencia en este estado. Debe serializarse la
revocación con la revalidación y la producción de la prueba dentro de la
autoridad VEC-AD-3; después hay que corregir la regresión para acreditar la
revocación sin depender de un timeout y repetir revisión independiente.

El marcador externo preexistente `/tmp/.git` hace fallar la suite global con el
`TMPDIR` predeterminado; no borrarlo. La suite se ejecuta con:

```bash
TMPDIR=/home/alberto/.cache go test ./...
```

## CT-000043: evidencia durable del resultado y Recibo RRHH

El borrador sigue aislado en:

```text
rama: agent/ct43-prueba-durable-20260728
worktree:
  /home/alberto/Trabajo/VEC_Diputacion_app/.worktrees/
  ct43-prueba-durable-20260728
base: e8d2afc
estado: seis archivos locales, sin commit ni push
```

Archivos:

```text
deploy/postgresql/contratacion_temporal/migraciones/
  000043_prueba_resultado_recibo_rrhh.up.sql
  000043_prueba_resultado_recibo_rrhh.down.sql
  000043_componentes/010_tipos_cierre.sql
  000043_componentes/020_relaciones_y_prueba.sql
  000043_componentes/030_primitiva_cierre.sql
  000043_componentes/090_acl_catalogo_y_barrera.sql
```

`git diff --check` está limpio y ningún archivo supera 800 líneas. El mayor es
`090_acl_catalogo_y_barrera.sql`, con 753.

Las correcciones ya incorporadas al borrador son:

- centinelas generados no nulos para ligar cuadro y detalle al registro de
  acceso sin dejar huecos por `MATCH SIMPLE`;
- claves foráneas separadas para resultado, cadena y prueba VEC;
- contenido tipado persistido de cuadro, detalle y cursor;
- recálculo de contenido, resultado y Recibo RRHH;
- comparación de las ocho piezas autoritativas devueltas por `000006`;
- referencias de auditoría opacas, sin derivarlas localmente;
- conservación de los SQLSTATE reintentables antes de normalizar errores;
- ligadura de `total` y de la identidad tipada de detalle;
- RLS forzada, política y catálogo alineados con el propietario;
- huella portable de roles de política mediante nombres `regrole`;
- comentarios de constraints, triggers, policy, rules y estadísticas incluidos
  en el manifiesto;
- normalización de método de acceso, tablespace y collation;
- `down` no destructivo con advisory lock, locks exclusivos, puerta de
  evidencia, `DROP RESTRICT` y retirada de barreras al final;
- búsqueda de constraints del `down` acotada a las tablas de CT;
- prevalidación antes de locks y verificación de tipos fila y array implícitos.

La tarea permanece `NO-GO` porque el worktree aún no contiene la dependencia
`000006`, no existe runner final de CT-000043, no se ha acreditado la huella
literal en una ejecución completa de PostgreSQL 18.4 y falta revisión
independiente. Los archivos son trabajo recuperable, no una capacidad
terminada.

El productor llegó a compilar el UP completo en PostgreSQL 18.4 usando una
copia externa de `000006`. La ejecución alcanzó el control de catálogo y
devolvió la huella con prefijo `2356e2c` y 63 restricciones; después hizo
rollback por el literal anterior. Actualizó ambos literales, pero, por la orden
de cierre, no reejecutó el UP, no ejecutó el DOWN y no añadió el runner. Esos
valores todavía no están acreditados y deben tratarse como borrador.

## Continuación exacta

1. Leer por completo `AGENTS.md`, este relevo y
   `coordinacion_ct_000041b_postgresql_recibo_rrhh_000042_000043_2026-07-28.md`.
2. Corregir el `NO-GO P1` de
   `agent/aut-v3-prueba-consumo-20260729`: serializar revocación,
   revalidación y prueba, y sustituir el falso positivo por timeout por una
   regresión que deje confirmar la revocación antes del resultado.
3. Reproducir el runner PostgreSQL 18.4 y las puertas estáticas y obtener un
   nuevo `GO` independiente. Solo entonces confirmar los cinco archivos en su
   rama con autor `aavidad <avidad@dipgra.es>`.
4. Integrar el commit aprobado mediante `cherry-pick` primero en
   `integracion/ct-o4-04e-20260726` y después en el worktree CT-000043. No usar
   `rebase` ni copiar archivos a mano.
5. Completar CT-000043: runner PostgreSQL 18.4 por resumen, huella literal,
   ataques de catálogo y comentarios, ACL/RLS, DML hostil, cruces de claves,
   replay, revocación, concurrencia, rollback, reentrada, safe-down con
   evidencia y `up → down → up`.
6. Ejecutar PostgreSQL real, Go focal/global y carrera aplicable, `go vet`,
   Bash, ShellCheck, formato, tamaños, `git diff --check` y Gitleaks.
7. Obtener dos dictámenes independientes: seguridad/arquitectura y pruebas.
   Corregir cualquier `NO-GO` antes de confirmar.
8. Tras ambos `GO`, confirmar CT-000043 en su rama, integrarlo mediante
   `cherry-pick`, actualizar tablero, mapa y relevo, enviar un único corte
   consolidado y comprobar GitHub hasta su conclusión.
9. Volver al camino crítico:

```text
CT-000044 motor interno
→ CT-000045 fachadas autorizadas
→ adaptador Go/PostgreSQL
→ composición raíz, identidad/PDP y TLS
→ navegador/API/aplicación/PostgreSQL/Recibo RRHH E2E
```

## Métricas oficiales

No deben incrementarse por estos borradores técnicos:

| Ámbito | Estado |
| --- | --- |
| Contratación temporal | `20/46`, 43 % |
| O4-05 | `3/5` hitos |
| Bolsa productiva | `1/14`, 7 % |
| Producción | `NO-GO` |

## Límites y directorios que no deben tocarse

- El directorio raíz está en la rama histórica `vec-orquesta-20260619`; no se
  programa allí.
- El Word de RRHH permanece sin seguimiento en el directorio raíz. No
  añadirlo, modificarlo ni eliminarlo.
- No limpiar worktrees o ramas ajenas sin inventario y autorización.
- No borrar `/tmp/.git`.
- No usar ni publicar credenciales, datos personales reales, certificados,
  tokens, claves, DSN ni rutas privadas.
- No desplegar en producción como parte de este relevo.
