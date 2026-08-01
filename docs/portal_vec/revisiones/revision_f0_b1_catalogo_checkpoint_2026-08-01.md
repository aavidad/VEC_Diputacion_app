# Revisión F0-B1: catálogo, revocación y checkpoint de fuente

Fecha: 1 de agosto de 2026.

## Resultado

F0-B1 obtuvo dos revisiones independientes finales `GO`, ambas con
`P0=P1=P2=0`. El commit productor `08e1818` se integró en la rama estable
como `00ff427`.

El corte añade exclusivamente los dos componentes
`050_catalogo_fuente_checkpoint.sql`. Sigue dormido: no existe aún una
migración `000007` instalable ni una API exterior.

## Huellas finales

```text
M050: 2429d854073988df554f733413536a4b315794eac3015a2de6cd9591c1ed57ba
      371 líneas; 16.186 bytes
T050: 5d006f92b7b65f5355df7749acfd919c59524002909a1ada2963c06af76a3c3b
      794 líneas; 59.252 bytes
```

El cuerpo del acreditador conserva la huella:

```text
62117d89c765f8d453d74f5b1feb7f401cc0cbb89c6a4f0ecbb65bb8f73a96ac
```

La clausura normalizada contiene 201 arcos `pg_depend` y la firma:

```text
056c54da40ca8b22b7897e3d375ccbcc20d693632410b4bc48e0159104211215
```

## Evidencia reproducida

El productor completó tres ejecuciones finales literales
`probar_fuente_corporativa_contexto_actor_v1_pg18_4.sh --etapa B1` sobre
PostgreSQL 18.4. Ambos revisores repitieron la etapa y dirección la ejecutó
después de integrar. Todas terminaron con rollback y línea base exactos.

La matriz acredita:

- audiencias exactas 3→7, catálogo y revocación `PERMANENT` y append-only;
- cinco FK exactas y sus veinte disparadores de integridad referencial;
- seis disparadores propios, RLS habilitada y forzada, políticas y ACL;
- checkpoint causal para altas y revocaciones;
- 22 y 7 atributos físicos, 29 totales y ninguno caído;
- índices, restricciones, funciones, cuerpos y dependencias exactos;
- árbol TOAST completo de ambas tablas, incluidos atributos, índices,
  opclasses, tipos y dependencias;
- ausencia de publicaciones efectivas por relación, esquema o global;
- ausencia de reglas, herencia, vistas entrantes, estadísticas extendidas y
  pertenencias hostiles a extensiones.

Las contrapruebas DBA demostraron `FALSE→TRUE` para publicaciones, extensión
`e/x`, tipo compuesto, índice TOAST, disparadores RI y restauración. El
mutante `ADD COLUMN` seguido de `DROP COLUMN` deja un atributo físico caído y
es rechazado. También se detectan alteraciones de estadísticas, almacenamiento,
compresión y opciones TOAST.

Las revisiones rechazaron candidatos previos por índices, disparadores o
cuerpos insuficientemente acreditados; publicaciones globales o por esquema;
dependencias de vistas, extensiones y estadísticas; atributos físicos y árbol
TOAST incompletos. La versión final agrupa esas garantías en inventarios
cerrados, no en excepciones nominales aisladas.

Bash, ShellCheck, `git diff --check`, límites y Gitleaks quedaron verdes. No
quedaron contenedores, procesos ni temporales atribuibles. La repetición
posterior a integración volvió a terminar verde y el rango publicado no
contiene fugas.

La CI compartida `30716722970` terminó completamente verde sobre `00ff427`.

## Continuación

B1 desbloquea B2 y, junto con A2+A3, C1. Ambos se abren en worktrees aislados.
B1 no cambia métricas ni habilita producción. Q1 y R1 conservan su alcance
posterior y producción continúa en `NO-GO`.
