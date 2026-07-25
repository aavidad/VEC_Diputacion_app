# Relevo de cierre de sesión — 25 de julio de 2026

## Referencias conservadas

- Rama aprobada: `real/ct-o4-03`, commit `0775853`.
- Rama de resguardo no aprobada: `resguardo/ct-o4-04-20260725`.
- O4-04C con revisión independiente `GO`: `a540522`.
- Aislamiento Docker y documentación: `577f157`.
- Aislamiento web y corrección visual: `0775853`.

La rama aprobada está publicada en GitHub. Los tres commits fueron revisados,
probados y analizados con Gitleaks antes del envío.

## Trabajo interrumpido y no integrable todavía

La rama de resguardo conserva dos commits marcados expresamente como `WIP`:

- `f480ad5`: corrección O4-04A interrumpida durante la tercera vuelta. La
  segunda revisión mantuvo `NO-GO` porque la prueba `go/types` no enumeraba de
  forma exhaustiva aliases, interfaces embebidas y valores con tipos anónimos.
  El agente empezó la regresión exhaustiva, pero la sesión se cerró antes de
  congelar un hash y obtener una nueva revisión.
- `f274f76`: corrección O4-04B interrumpida antes de congelar el candidato y
  ejecutar su revisión final. Debía cerrar canones Go/SQL, revocar todo
  `EXECUTE PUBLIC`, proteger los `down` fuera de orden y unificar la advisory
  lock con `vec_contratacion_temporal:o4_04:migraciones`.

Estos commits existen para impedir pérdida de trabajo. **No deben fusionarse
en `real/ct-o4-03` ni contarse como capacidad cerrada** hasta terminar las
correcciones, repetir PostgreSQL 18 y obtener un `GO` independiente por corte.

## Estado del camino crítico

| Corte | Estado al cerrar |
| --- | --- |
| O4-04A | WIP resguardado; sin GO |
| O4-04B | WIP resguardado; sin GO |
| O4-04C | Cerrado, commit `a540522`, GO independiente |
| O4-04D | Análisis listo; no iniciar hasta que A/B tengan GO |
| O4-04E | Bloqueado por A–D |

O4-04D usará `000023`, exigirá la versión de barrera `3` creada por C y no
debe modificar `000017`–`000022`. El wrapper VEC deberá concederse únicamente
al propietario `NOLOGIN`; solo la futura función exterior de E será ejecutable
por el rol de ejecución.

## Presentación web

La presentación aprobada superó:

- 361 pruebas web;
- 9 pruebas Python de composición y aislamiento;
- 51/51 recorridos visuales de Contratación temporal;
- capturas reales de Bolsa y Contratación sin errores de consola;
- caída de cartografía con Bolsa y Contratación todavía disponibles.

Las evidencias de navegador permanecen locales e ignoradas por Git en
`.evidencias-locales/`; no contienen credenciales ni forman parte del
artefacto productivo.

## Reanudación recomendada

1. Partir de `resguardo/ct-o4-04-20260725`.
2. Terminar A sin tocar B y congelar su write-set.
3. Revisar A con un agente distinto; integrar solo con `GO`.
4. Terminar B, repetir PostgreSQL 18 y revisión independiente.
5. Reconstruir una rama limpia desde `real/ct-o4-03` e incorporar únicamente
   los commits aprobados de A/B.
6. Implementar O4-04D y, después, O4-04E.

El procedimiento de Contratación temporal continúa oficialmente en 18/46
tareas verificadas (39 %). No se incrementa por trabajo WIP.
