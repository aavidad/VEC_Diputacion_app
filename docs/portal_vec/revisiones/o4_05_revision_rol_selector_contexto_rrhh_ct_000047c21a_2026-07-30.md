# Revisión final CT-000047C2.1a: rol selector de contexto RRHH

Fecha: 30 de julio de 2026.

## Corte

```text
base: 7512513
candidato final: a7f0cb4
integración técnica: e8c950c
```

Veredicto de dos revisores independientes: **GO**.

| Severidad | Hallazgos abiertos |
| --- | ---: |
| P0 | 0 |
| P1 | 0 |
| P2 | 0 |

## Resultado

Se crea el grupo técnico `NOLOGIN`
`vec_contexto_actor_corporativo_rrhh_selector`. Su única autoridad efectiva es
`CONNECT` sobre la base actual. No recibe membresías, acceso a datos,
ejecución de funciones, contraseña, ajustes ni capacidad de inicio de sesión.

El alta rechaza homónimos y estados hostiles sin adoptarlos. La retirada
reacredita el manifiesto completo, bloquea las carreras relevantes y falla de
forma segura ante cualquier dependencia o privilegio adicional.

## Correcciones exigidas durante la revisión

El candidato inicial omitía la autoridad implícita de `PUBLIC` sobre tipos con
`typacl` nulo. Una primera corrección usó una clasificación heurística que
excluía tipos base legítimos con `CATEGORY='A'` o `ELEMENT`.

La corrección final distingue un array generado únicamente mediante la
relación inversa del catálogo:

```sql
elemento.oid = tipo.typelem
AND elemento.typarray = tipo.oid
```

Así se rechaza la autoridad efectiva sobre tipos base reales, incluidos los
casos `CATEGORY='A'` y `ELEMENT=int4`, sin convertir el array automático en un
falso positivo. El mismo criterio se aplica en la prevalidación,
postvalidación y retirada.

## Evidencia reproducida

Los revisores y dirección reprodujeron sobre PostgreSQL 18.4:

- tipo base ordinario antes y después del alta;
- tipo base explícito con `CATEGORY='A'`;
- tipo base con `ELEMENT=int4` y su array automático;
- `DOMAIN` con `typacl` nulo y autoridad implícita de `PUBLIC`;
- lenguaje de usuario con `lanacl` nulo;
- alta concurrente, reentrada, homónimos y aislamiento;
- atributos, ajustes, membresías, ACL locales e interbase;
- privilegios predeterminados, políticas y dependencias;
- retirada segura frente a funciones C2 nominales y carreras futuras;
- ciclo `up → down → up` sin residuos.

Puertas:

```text
bash -n
shellcheck
runner PostgreSQL 18.4
git diff --check
gitleaks
```

Todas terminaron correctamente. Los tres ficheros permanecen por debajo del
límite de 800 líneas.

## Alcance

Este corte prepara un principal mínimo, pero no concede acceso funcional ni
habilita producción. La fachada de Identidad C2.1b, la selección y registro
transaccional, el PDP, la composición raíz, TLS/mTLS y el E2E siguen
pendientes. Las métricas funcionales no cambian.
