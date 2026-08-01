# Revisión F0-A2: canon del manifiesto de fuente

Fecha: 1 de agosto de 2026.

## Resultado

F0-A2 obtuvo dos revisiones independientes `GO`, ambas con
`P0=P1=P2=0`. El commit productor `6493dda` se integró en la rama estable
como `0ac8fe4`.

El corte añade exclusivamente los dos componentes `020_canon_manifiesto.sql`.
No publica una función exterior, permiso, rol, tabla ni autoridad.

## Huellas finales

```text
M020: 88730c7de8a08dad6592c7978e8988726a849e58a1d21cbe7327b62177d0345c
      164 líneas; 6.984 bytes
T020: 025640988866a4a044f38d4902bd695a7c22f1e1a03cd2b0a8086db816e63e88
      543 líneas; 21.265 bytes
```

El vector V0 común mide 705 bytes y conserva la huella:

```text
f16cab3533e7a5b4126ae1bddf8afbc989ce564330cf9703d1429ceb21678325
```

## Evidencia reproducida

El productor completó tres ejecuciones finales literales
`probar_fuente_corporativa_contexto_actor_v1_pg18_4.sh --etapa A2` sobre
PostgreSQL 18.4. Ambos revisores volvieron a ejecutar la etapa y dirección la
repitió después de integrar. Todas terminaron con rollback y línea base
exactos.

La matriz acredita:

- trece claves, orden y tipos cerrados;
- `version` y `fuente_version` como enteros JSON canónicos;
- cuatro cruces nominales audiencia–acción–efecto;
- rechazo de claves repetidas, ausentes, sobrantes o reordenadas;
- rechazo de tipos, fracciones, exponentes y JSON malformado;
- espacios estructurales sin alterar espacios dentro de cadenas;
- escapes Unicode, caracteres de control, barra, BOM, NUL y UTF-8 inválido;
- límites mínimo y máximo inclusivos;
- igualdad byte a byte entre el canon PostgreSQL y el vector Go/V0;
- catálogo, propietario, firma, ACL, paralelismo y seguridad exactos;
- mutaciones hostiles de ACL, sobrecarga, paralelismo y
  `SECURITY DEFINER`.

También quedaron verdes las pruebas Go normales y con detector de carreras,
`go vet`, Bash, ShellCheck, `git diff --check`, límites y Gitleaks. No quedaron
contenedores, procesos ni temporales atribuibles.

Dos intentos iniciales hicieron rollback limpio antes de la congelación: uno
detectó un `CASE` mal parentizado en PL/pgSQL y otro un oráculo de prueba que
eliminaba espacios dentro de cadenas. Se corrigieron sin cambiar el contrato y
no forman parte de las huellas finales.

La CI técnica `30710676474` se abrió al publicar `0ac8fe4` y seguía en curso
al redactar este corte.

## Continuación

A2 no cambia métricas ni crea todavía una migración `000007` instalable.
A3 y A4 siguen listos; B1 permanece en revisión independiente. Producción
continúa en `NO-GO`.
