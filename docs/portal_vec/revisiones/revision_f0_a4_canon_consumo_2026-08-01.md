# Revisión F0-A4: canon y huella de consumo

Fecha: 1 de agosto de 2026.

## Resultado

F0-A4 obtuvo dos revisiones independientes finales `GO`, ambas con
`P0=P1=P2=0`. El commit productor `bf300b6` se integró en la rama estable
como `4dd2ff9`.

El corte añade exclusivamente los dos componentes `040_canon_consumo.sql`.
No publica una fachada, permiso, rol, tabla ni autoridad exterior.

## Huellas finales

```text
M040: 3faf15ea0ba013719b62536ee952c9ac5c0b87a4882058d6a0042520bb1ffe51
      316 líneas; 14.483 bytes
T040: dc864b46f203122490dc278a541b76928f4f21a18df6d64f7e8ef2a5f5e0d04f
      772 líneas; 32.405 bytes
```

Los vectores V0 de capacidad y consumo miden 1.891 y 2.021 bytes y conservan
estas huellas:

```text
capacidad: d3baaa6bf9e8e757d659f42233186a799e3c0b6e9a8e5eab1b5930ca0e7f7e54
consumo, fragmentos consecutivos:
  0755995c42bdbdf7
  de83d6066c3b17c3f95534bc17de35a5d91f43560b3f1e85
```

## Evidencia reproducida

El productor completó tres ejecuciones finales literales
`probar_fuente_corporativa_contexto_actor_v1_pg18_4.sh --etapa A4` sobre
PostgreSQL 18.4. Ambos revisores repitieron la etapa y dirección volvió a
ejecutarla después de integrar. Todas terminaron con rollback y línea base
exactos.

La matriz acredita:

- canon de consumo cerrado y derivación interna de referencia y huella;
- reconstrucción SQL de la capacidad desde 33 campos tipados, sin copiar el
  literal canónico del fixture;
- diez huellas y cinco textos técnicos ligados individualmente a su validador;
- seis enteros comprobados uno a uno contra léxico decimal, cero y exceso;
- evento anterior o simultáneo a emisión, con el caso de igualdad aceptado;
- intervalo de consumo `[emitida_en, expira_en)`, duración positiva y máximo
  inclusivo de cinco segundos;
- cuatro cruces nominales, UTF-8, NUL, tamaños, nulos, escapes y JSON cerrado;
- vector PostgreSQL igual byte a byte al oráculo Go/V0;
- firma, propietario, ACL, atributos, coste y huella exacta del cuerpo.

Las revisiones rechazaron candidatos que no probaban el máximo exacto de
cinco segundos, copiaban un canon literal, permitían omitir validadores de
campos, no cubrían la igualdad temporal o dejaban incompleto el catálogo y
los seis enteros. La versión final mata por separado mutantes de tiempo,
coste, cuerpo, MAC, emisor y enteros, incluso cuando el mutante ajusta su
propia huella catalogal para alcanzar la prueba funcional.

Go normal y con detector de carreras, `go vet`, Bash, ShellCheck,
`git diff --check`, límites y Gitleaks también quedaron verdes. No quedaron
contenedores, procesos ni temporales atribuibles. La repetición posterior a
integración volvió a terminar verde y el rango publicado no contiene fugas.

La CI compartida `30716722970` terminó completamente verde sobre `00ff427`.

## Continuación

A4 no cambia métricas ni crea todavía una migración `000007` instalable. Su
salida alimentará C2 cuando también estén cerrados C1 y B2. Producción
continúa en `NO-GO`.
