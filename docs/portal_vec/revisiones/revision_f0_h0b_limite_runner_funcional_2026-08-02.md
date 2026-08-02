# Revisión F0-H0b: límite del runner funcional

Fecha: 2 de agosto de 2026.

## Resultado

El commit documental publicado `6d904735` obtuvo dos revisiones independientes
finales `GO`, ambas con `P0=P1=P2=0`. Aprueba la excepción local medida para
los checkpoints 3 y 4; no acredita todavía su implementación ni cierra H0b.

## Primer candidato: doble NO-GO

El objeto candidato `a3547c4` no se publicó. Los dos revisores documentales
emitieron inicialmente `NO-GO`:

1. El revisor 1 atribuyó únicamente `P1` a la aritmética. Un coste bruto de
   72–92 líneas y un ahorro descrito como 10–30 no demostraban el intervalo
   607–642.
2. El revisor 2 rechazó cuatro ambigüedades contractuales: no subordinar 640
   al objetivo general de 500 líneas de DEC-051; afirmar que no se relajaban
   límites pese a cambiar uno local; citar una reserva para I0 sin
   cuantificarla; y no separar con precisión el presupuesto de los
   checkpoints 3 y 4 del trabajo posterior de I0.

Ningún `NO-GO` autorizó código, minificación, retirada de pruebas, ampliación
del *write-set* ni cambio de propietario.

## Corrección exacta

El commit corregido sustituye el cálculo por un intervalo conservador y
reproducible:

```text
coste bruto = 72..92
ahorro potencial = 0..30, no garantizado ni acreditado en el extremo superior
mínimo = 550 + 72 - 30 = 592
máximo conservador = 550 + 92 = 642
rango = 592–642
```

El valor 640 queda definido como presupuesto o umbral local de revisión por
excepción medida. No sustituye el objetivo general de diseño de 500 líneas de
[DEC-051](../registro_decisiones.md#dec-051--limite-de-tamano-de-los-ficheros-de-codigo).
El tope local de 650 permanece por debajo del límite duro global de 800 y ya
incluye activación, finalizador retornable, *seam* y matriz de inyección de los
checkpoints 3 y 4 completos. Si no caben, se detiene el corte y se exige otra
decisión; nunca se excede 650 por conveniencia.

El auxiliar privado H0b conserva objetivo de 460 líneas o menos y límite duro
inferior a 800.

Tras 650 quedan al menos 150 líneas hasta el límite global para que I0 integre
en el runner bootstrap, P0, reintentos, Q1–Q3, oráculo V0, tres pasadas y
cierre operativo, y conserve byte a byte el auxiliar R0/H0b y su SHA literal.
Cualquier necesidad posterior exige una decisión nueva.

La enmienda auxiliar previa ahora distingue correctamente fronteras y
controles —que no se relajan— del único límite local cambiado expresamente.
D2c, D2d y el capturador permanecen byte a byte inmutables; no se añade un
cuarto auxiliar ni se elimina o minifica cobertura.

## Dictámenes finales

| Revisión | Dictamen | Recuento | Motivo de cierre |
| --- | --- | --- | --- |
| Revisor 1 | `GO` | `P0=P1=P2=0` | Las dos ecuaciones, el rango 592–642 y la relación con DEC-051 son coherentes y reproducibles. |
| Revisor 2 | `GO` | `P0=P1=P2=0` | Los presupuestos H0b/I0, las fronteras y la obligación de nueva decisión quedan cerrados sin contradicción. |

Los dictámenes son documentales y no heredan como prueba funcional ninguna
ejecución del candidato rechazado.

## Verificaciones del commit publicado

- `git show --check 6d904735`: limpio;
- 86 enlaces Markdown locales comprobados, incluida la referencia a DEC-051;
- tamaños de los seis documentos: `800/123/175/126/730/524` líneas;
- Gitleaks: un commit y 9,31 kB analizados, cero fugas;
- *write-set* limitado a los seis documentos declarados, sin código ni Word
  de RRHH.

La ejecución CI
[`30730111840`](https://github.com/aavidad/VEC_Diputacion_app/actions/runs/30730111840)
terminó con `success`: sus cinco puertas —calidad, secretos, artefactos
productivos, PostgreSQL ContextoActor V3 y PostgreSQL Bolsa pública— quedaron
completamente verdes para `6d904735`.

## Estado

El doble `GO` satisface la puerta documental previa al checkpoint 3. No cierra
H0b, C2 ni F0 y no aumenta métricas: F0 permanece en `10/23`, O4-05 en `3/5`,
Contratación en `24/46` y Bolsa productiva en `1/14`. Producción continúa en
`NO-GO`.
