# Revisión F0-A1: validadores privados

Fecha: 1 de agosto de 2026.

## Resultado

F0-A1 obtuvo `GO` funcional y adversarial final con `P0=P1=P2=0`. El commit
productor `e5de24b` se integró en la rama estable como `169a055`.

El corte añade únicamente los dos componentes `010_validadores.sql`. No crea
una migración `000007` instalable, API, rol, permiso exterior ni dato.

## Huellas finales

```text
M010: d99bbe8861dcdef6ba17b42d28e2d6ed7deecef805778b08971ad6c173b2cdef
      282 líneas; 8.067 bytes
T010: 208b9c2d756d8bf616ed84e56ac46005d38552e35b41d4df67c26e2d32b48382
      635 líneas; 26.506 bytes
```

## Ciclos de revisión

El primer candidato superó tres ejecuciones PostgreSQL 18.4 y la revisión
funcional, pero la revisión adversarial emitió `NO-GO`, `P0=0`, `P1=1` y
`P2=1`:

- la prueba catalogal seguía verde con un permiso al migrador, una sobrecarga
  homónima y `PARALLEL SAFE`;
- la primitiva de intervalos devolvía `NULL`, no `FALSE`, si un límite era
  nulo.

La corrección hizo total el predicado de límites y añadió una aserción común
de catálogo, inventario y ACL. La contrarrevisión funcional y adversarial
desde cero terminó con doble `GO`, `P0=P1=P2=0`.

## Evidencia reproducida

Productor y revisores ejecutaron el runner literal
`probar_fuente_corporativa_contexto_actor_v1_pg18_4.sh --etapa A1` sobre
PostgreSQL 18.4 por digest. El productor completó tres pasadas finales y
dirección repitió una después de integrar.

La matriz acredita:

- UTF-8 estricto, BOM, NUL, Unicode visual y límites de tamaño;
- referencias opacas, operaciones, enteros JSON seguros, SHA-256 e instantes
  UTC canónicos;
- trece funciones privadas exactas, sin sobrecargas;
- propietario, lenguaje, clase, argumentos, retornos, valores predeterminados,
  variadicidad, conjuntos, volatilidad, estrictitud, `SECURITY INVOKER`,
  `search_path`, `LEAKPROOF` y paralelismo;
- ACL efectiva exclusiva del propietario y ausencia de opción delegable;
- mutaciones hostiles de ACL, sobrecarga, lenguaje, retorno, argumentos,
  valores predeterminados, variadicidad, procedimiento, `SETOF`, volatilidad,
  seguridad, `search_path` y paralelismo;
- límites nulos con resultado exacto `FALSE`;
- rollback, línea base, txid, snapshot y cero residuos.

También quedaron verdes los límites de 800 líneas y 1 MiB,
`git diff --check`, Bash, ShellCheck, analizador SQL y Gitleaks del commit.

La CI `30709399752` se abrió al publicar `169a055` y seguía en curso al
redactar este corte.

## Continuación

A1 no cambia las métricas y producción continúa en `NO-GO`. Quedan
desbloqueados, con write-sets disjuntos, A2, A3, A4 y B1. Cada nodo debe pasar
su propia matriz PostgreSQL 18.4 y revisión independiente antes de integrarse.
