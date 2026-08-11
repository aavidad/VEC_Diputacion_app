# Decisión O4A-P1D — sello mínimo del deadline heredado

Fecha: 11 de agosto de 2026.

Estado: propuesta documental. No autoriza implementación, P3, P4, O4b,
O4c, integración, producción, despliegue ni cambio de métricas.

## Base y autoridad

La base exacta es `50f2a81302dc202bca3cf4d9986b7351b10fa9ae`, cierre
de `O4A-P2-SEMILLA` publicado en
`trabajo/o4a-p2-semilla-v2-20260811`. La CI
[31529462600](https://github.com/aavidad/VEC_Diputacion_app/actions/runs/31529462600)
terminó `Success` con cinco puertas sobre ese SHA y rama exactos.

El contrato O4a continúa siendo
[`decision_f0_h0b_c4b2_g2o_o4a_causa_tiempo_2026-08-11.md`](decision_f0_h0b_c4b2_g2o_o4a_causa_tiempo_2026-08-11.md),
SHA-256 `ffe3d570b3fe51be96948f8aeb0b163e2ff071d2c2cca4fbdba852a5f9dafebc`.
La decisión P1B de sellos raw conserva SHA-256
`e279786ccbd302a1d9fc6bddf53013d98855b0f4e5b7839dbc936a76c203fba9`.

Los bytes productivos congelados relevantes son:

- P1/P1C `autoridad.go`: `d2cd635c0bc6e06ce72fe50128f7e917c977b97ebefe4819cd7cba189ad913c4`;
- su prueba: `bad464cf6b92e3b8cf52ad3c1edae5617be763bb8643cd93b7997d8fc7602e9a`;
- P2 `semilla.go`: `493c99cc2dcdeb62fc3ea4f814f1bdfadb7a2bd3ea338dcb0a309675ef94d72f`;
- su prueba: `c7357d6a856b803efa11363ea95935c6999599d44c06cf45627189b33267c108`.

## NO-GO demostrado

O3c crea `ahoraCaso` mediante su única llamada `time.Now()` y deriva
`finCaso=ahoraCaso+180s` antes del único CONT. C5 entrega ambos valores dentro
de `agregadoO4aM38`. P1 conserva el puntero al agregado, pero `sellosO4aM38`
no copia `ahoraCaso` ni `finCaso`. `finBootstrap` ya fue sellado y acreditado
por O3c para el handoff, pero P1 tampoco coteja contra él la pareja recibida.

Los campos del agregado siguen siendo mutables. Sustituir coordinadamente
`ahoraCaso` y `finCaso` por otro par monotónico no cero separado exactamente
180 segundos supera las comprobaciones estructurales disponibles y reinicia
el plazo. Capturar esos valores por primera vez en P3
legitimaría cualquier sustitución ocurrida después de P1.

Por tanto P3 no puede acreditar A09 ni hacer que corrupción temporal converja
a AF sobre la base actual. El bloqueo es de autoridad, no de cobertura.

## Sello mínimo autorizado

La implementación posterior añade exclusivamente dos valores privados a
`sellosO4aM38`:

```go
ahoraCaso     time.Time
finCaso       time.Time
```

Ambos son copias por valor. No se añade puntero, duración, reloj, función,
callback, hash, texto, zona exterior, getter ni representación serializada.
`finBootstrap` no se copia: O3c ya lo selló hasta C5 y, después de acreditar
en P1 la relación de entrada, es un límite consumido que no gobierna P2/P3.
P1E lo lee únicamente para validar esa genealogía en el mismo snapshot pre-CAS.

## Fuente y punto de linealización

Las únicas fuentes son, respectivamente:

1. `agregadoO4aM38.ahoraCaso`, creado por O3c P3;
2. `agregadoO4aM38.finCaso`, derivado una vez por O3c P3;

P1 mantiene su orden de consumo:

1. pone a cero el puntero del llamador;
2. acredita C5, autoidentidades, owners, lease/observador, genealogías,
   recursos, identidad y snapshot;
3. carga por valor una sola vez la pareja temporal y lee el `finBootstrap`
   heredado para el cotejo genealógico, en la misma ventana en que
   captura `primera`, palabra observada y canon CONTROL;
4. valida íntegramente la pareja y su relación con bootstrap, y preasigna todos los sellos;
5. ejecuta el único CAS de custodia `2→3`;
6. entra A1 sin volver a cargar la pareja ni bootstrap desde el origen mutable.

La copia pre-CAS es el punto de linealización histórico. P1E no llama a
`time.Now`, no crea plazo y no normaliza ni redondea marcas. Puede usar una vez
`finCasoExactoO3cM38(ahoraCaso)` exclusivamente como cálculo puro adversarial
y exigir igualdad estructural con el `finCaso` recibido; el resultado calculado
no se almacena ni sustituye al sello. Una segunda carga posterior al CAS está
prohibida. P2 permanece byte-inmutable y no consume estos campos.

## Validación fail-closed

Antes del CAS P1E exige conjuntamente:

- las dos marcas y `finBootstrap` distintos de cero y con componente monotónico mediante el
  predicado O3c existente, sin `Round`, parseo ni reconstrucción civil;
- `ahoraCaso.Before(finCaso)`;
- `finCaso.Sub(ahoraCaso) == 180*time.Second`; además obtiene
  `finCalculado, ok := finCasoExactoO3cM38(ahoraCaso)` y exige
  `ok && finCaso == finCalculado`, incluido su componente civil y monotónico;
- `ahoraCaso.Before(finBootstrap)` exactamente;
- `finCaso` igual al resultado ya materializado por O3c; el valor transitorio
  derivado por el helper solo se coteja y jamás se almacena, sustituye ni
  reinicia el deadline recibido;
- ausencia de overflow, marca saturada o relación no representable, acreditada
  por las comparaciones y resta exactas sobre los valores recibidos;
- la validación y preasignación operan solo sobre tres locales capturados una
  vez (`ahoraCaso`, `finCaso`, `finBootstrap`), sin una segunda lectura del
  origen inmediatamente antes ni después del CAS.

Marca cero, pérdida del componente monotónico, reloj civil reconstruido,
relación invertida, duración distinta, overflow o genealogía bootstrap
inválida llevan a AF antes de consumir. No se degradan a `PLAZO` ni
`INCIDENTE`.

Después de P1, P3 solo confiará en la pareja sellada. Antes de cualquier
evento, syscall o lectura de reloj deberá cotejar mediante el operador Go `==`
—no `Time.Equal`, `Before` ni solo `Sub`— que cada `time.Time` mutable original sigue
coincidiendo con sus copias. Cualquier sustitución posterior, incluso otro par
monotónico válido de 180 segundos, lleva a AF. El reloj de ronda puede leerse
solo después de ese cotejo y nunca modifica el sello.

Una mutación entre la captura histórica y el CAS no se relee ni se legitima.
Toda divergencia persistente respecto del sello se detecta en P3 antes de
cualquier efecto. Una escritura ABA restaurada tampoco cambia la autoridad:
P3 calcula exclusivamente desde las copias y usa el origen solo para el cotejo
estructural. El CAS no convierte los campos mutables en autoridad.

## Invariantes preservadas

- owners O4A y estados lease/observador 3/2 no cambian;
- custodia, registro, generaciones, TID, identidad, tres pidfd, CONTROL,
  TERMINAL, `Cmd`, `Process`, handle y snapshot permanecen intactos;
- `primera`, retorno CONT raw, palabra histórica y canon CONTROL conservan sus
  sellos P1C exactos y separados;
- causa e incidente siguen vacíos al salir de P1;
- el CAS `2→3` sigue siendo único, irreversible y sin rollback;
- nulo, clon, alias, replay y perdedor concurrente no capturan una autoridad
  nueva ni alteran recursos;
- P2 permanece byte-inmutable y conserva su semántica ya publicada;
- la corrección no crea causa, transición P2/P3 ni efecto.

## Prohibiciones

P1E no puede introducir `time.Now`, reloj civil, `Add` nuevo para fabricar el fin,
timer, `Sleep`, ticker, contexto con deadline, pausa, extensión, recálculo o
reinicio. Tampoco puede añadir syscall, poll, señal, Wait/waitid, drenaje,
cierre/escritura de TERMINAL, liberación, parser, getter, ticket, API, log,
serialización, goroutine, canal, global mutable, `init`, hook o callback.

No puede copiar la pareja desde parámetros exteriores, fixtures, variables de
paquete o una segunda autoridad. No abre P2, P3, P4, O4b, O4c, O5 u O6.

## Implementación posterior autorizable

Nombre exacto: `O4A-P1E-SELLOS-DEADLINE`.

Base exacta: el SHA publicado de P1D, que deberá contener
`50f2a81302dc202bca3cf4d9986b7351b10fa9ae` más exclusivamente esta decisión,
su checkpoint y las dos actas independientes.

Write-set mínimo y máximo de P1E:

1. editar únicamente
   `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/causa_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_autoridad.go`;
2. editar únicamente su prueba existente `..._autoridad_test.go`.

No se crea otro fichero. Objetivo: 25--90 líneas netas entre ambos; parada si
algún fichero supera 650 líneas y tope duro 800.

## Matriz P1E

| ID | Oráculo obligatorio |
| --- | --- |
| D01 | C5 válido copia exactamente ambas marcas antes del CAS 2→3. |
| D02 | las copias conservan el componente monotónico; ninguna se redondea ni reconstruye. |
| D03 | marca cero o civil reconstruida produce AF antes del CAS. |
| D04 | duración 180s−1ns, 180s+1ns, igualdad o inversión produce AF. |
| D05 | `ahoraCaso >= finBootstrap` produce AF en el snapshot pre-CAS; bootstrap no se copia ni gobierna P3. |
| D06 | sustitución coordinada pre-P1 se valida solo como la entrada recibida; una sustitución posterior queda detectable por desigualdad con el sello. |
| D07 | overflow/saturación o relación temporal no representable produce AF sin reloj nuevo. |
| D08 | alias, replay, clon y carrera dejan un único ganador y no alteran pareja ni recursos. |
| D09 | raw CONT, primera, palabra, canon, owners y snapshot permanecen exactos. |
| D10 | AST acredita prevalidación→captura única→preasignación→CAS y prohíbe reloj/efectos/API/P2/P3. |

Mutantes mínimos: omitir cualquiera de las dos copias; intercambiarlas;
aceptar cero; eliminar monotonicidad; aceptar 180s±1ns; invertir bootstrap;
usar `Round`, `Add` o `time.Now`; capturar después del CAS; efectuar segunda
carga; comparar solo duración; aceptar sustitución posterior; modificar raw,
owners o snapshot.

Normal y race deben repetir positivos, negativos y carrera. Los BF fatales
exigen estado 65, EOF/no retorno y stdout/stderr cero, con inventario de
FD/hijos/zombis/grupos/temporales sin delta.

## Cierre y DAG

P1D cierra únicamente con doble revisión independiente `P0=P1=P2=0`, commit
documental pequeño, push normal y CI 5/5 del SHA exacto.

Aristas obligatorias:

```text
O4A-P1D-CONTRATO-DEADLINE
  -> O4A-P1E-SELLOS-DEADLINE
  -> O4A-P3-ARBITRAJE (reabierta)
```

P2 permanece cerrado e inmóvil; no se repite. P3 solo se reabre después del
doble GO y CI 5/5 de P1E. P4 y fronteras posteriores siguen bloqueadas.

Esta decisión no cambia porcentajes oficiales, integración, master,
producción ni despliegue.
