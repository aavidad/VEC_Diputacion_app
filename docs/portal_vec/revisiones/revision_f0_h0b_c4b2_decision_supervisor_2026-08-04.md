# Revisión F0-H0b C4b-2: decisión del supervisor exterior

Fecha: 4 de agosto de 2026.

## Resultado

**GO documental**, confirmado sobre el commit `dd6d619` por tres revisores
independientes con `P0=0`, `P1=0` y `P2=0`.

La decisión aceptada está en:

- `docs/portal_vec/decision_f0_h0b_c4b2_captura_quinta_supervision_exterior_2026-08-04.md`;
- `docs/portal_vec/procedimiento_operativo_f0_h0b_supervisor_irrecuperable_2026-08-04.md`.

Los revisores trabajaron en copias inmutables distintas del candidato. No
modificaron el código, no ejecutaron Docker y no alteraron el árbol estable.

## Contrarrevisión funcional

Resultado: `GO`, `P0=0`, `P1=0`, `P2=0`.

Comprobó:

- duplicación inmediata de los dos descriptores del `coproc` antes de `ARMAR`;
- protocolo cerrado `ARMAR -> ACK_LISTO -> INICIAR|CANCELAR -> ACK_CASO`;
- ausencia de Bash de caso antes de `INICIAR`;
- plazos absolutos y no reiniciables para cada espera;
- acreditación de terminalidad antes de la única espera del proceso;
- convergencia de las dos ramas de STOP y drenaje siempre inicializado;
- un único resultado funcional y un único epílogo exterior;
- tratamiento de la caída total del anfitrión mediante evidencia exterior,
  cuarentena y reprovisión.

La prueba focal Bash conservó el recibo mediante los descriptores duplicados
después de terminar el `coproc`. `git diff --check 31c9169..dd6d619` quedó
limpio.

## Contrarrevisión de seguridad

Resultado: `GO`, `P0=0`, `P1=0`, `P2=0`.

Comprobó:

- descriptores estables y ausencia de dependencia posterior del array o PID
  volátiles del `coproc`;
- plazos monotónicos en Shell y Go;
- `waitid` con `WEXITED|WNOHANG|WNOWAIT` antes del único `cmd.Wait`;
- dos `pidfd` del mismo objeto, promoción única ante `EBADF` y barrera FD 9;
- señalización exclusiva del grupo por `pidfd`, sin alternativa por PID o PGID
  numérico;
- espera terminal de un segundo y cuarentena ante supervisor todavía vivo,
  recibo sin salida o pérdida irreversible de autoridad;
- runbook seguro ante caída completa del anfitrión.

La prueba focal recorrió `ARMAR -> ACK_LISTO -> INICIAR -> recibo estable ->
terminalidad -> wait -f`. `git show --check dd6d619` quedó limpio.

## Contrarrevisión de capacidad y alcance

Resultado: `GO`, `P0=0`, `P1=0`, `P2=0`.

Comprobó:

- ledger actualizado y reservas positivas para Q5a, C4b-2 y C4b-3;
- barrera de arranque FD 9;
- duplicación estable del `coproc` y plazos cerrados;
- límite local C4c explícitamente elevado de 580 a 600 líneas sin modificar
  el límite final DEC-051 de 800 líneas.

## Decisión de dirección

Se acepta `dd6d619` como contrato implementable de C4b-2. La autorización no
permite saltar fases ni ampliar sus conjuntos de escritura:

1. Q5a: captura privada de cinco fuentes, compilación cerrada y autoprueba ABI
   del supervisor sin recursos de caso;
2. revisión independiente de Q5a;
3. C4b-2 operativo, todavía sin reconciliación Docker;
4. doble revisión independiente de C4b-2;
5. C4b-3, C4c y C4d según el DAG vigente.

Este cierre no aumenta los contadores de capacidad y mantiene el `NO-GO` de
producción. La primera unidad autorizada es Q5a, no C4b-2 completo.
