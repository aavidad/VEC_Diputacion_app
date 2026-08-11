# Decisión O4A-P1B — sellos raw mínimos para SEMILLA

Fecha: 11 de agosto de 2026.

Estado: propuesta documental para revisión independiente. No autoriza código,
P2, P3, O4b, O4c, integración, producción, despliegue ni métricas.

## Base, autoridad y alcance

La base material exacta es el cierre O4A-P1-AUTORIDAD
`0e750d41cbf72b0f0952341fa18e25474c3269fb`, rama
`trabajo/o4a-p1-autoridad-20260811`, CI `31516149966` concluida `success` con
cinco jobs. El contrato O4a autoritativo permanece en
`docs/portal_vec/decision_f0_h0b_c4b2_g2o_o4a_causa_tiempo_2026-08-11.md`,
SHA-256 `ffe3d570b3fe51be96948f8aeb0b163e2ff071d2c2cca4fbdba852a5f9dafebc`.

Esta decisión corrige exclusivamente la suficiencia de los sellos privados de
P1. No cambia precedencias, causas, owners, estados, recursos, tiempos ni el
DAG `P1→P2-SEMILLA→P3-ARBITRAJE`.

Write-set documental de P1B:

1. esta decisión;
2. checkpoint propio;
3. acta funcional;
4. acta de seguridad.

O3a/O3b/O3c, P0, P1, pruebas, herramientas, workflow, SQL y adaptadores quedan
byte-inmutables en P1B.

## NO-GO reproducido

P1 publicado exige en `entradaExactaO4aM38` que la palabra del observador sea
igual a `baselineSenal` y que el signo sea cero. O3c P4, en cambio, instala
`senal_raw` cuando su comprobación de observador deja de ser verde. O3c P5
admite ese handoff mientras el estado enmascarado siga en 2. Por tanto una
señal raw legítima puede alcanzar C5 y P1 la fataliza.

Además, `sellosO4aM38` conserva `baselineSenal` y el puntero CONTROL, pero no:

- la palabra completa observada que originó `senal_raw`;
- un valor inmutable que distinga las cuatro traducciones de `control_raw`.

El discriminante solo identifica la familia. Releer después el observador o
los campos mutables de CONTROL violaría P2, y fabricar una causa sería apertura
por defecto. El NO-GO de P2 es, por ello, causal y no una falta de pruebas.

## Sello mínimo autorizado

La implementación posterior solo puede añadir a `sellosO4aM38` dos valores
privados, no exportados y no serializables:

1. `palabraObservada uint64`: copia completa y exacta de
   `custodia.observador.palabra.Load()`;
2. `canonControlRaw`: enum privado cerrado con `VACIO`, `CANCELADO_65`,
   `PROTOCOLO_65`, `SENAL_INT_130` y `SENAL_TERM_143`.

`VACIO` no es causa. Solo acredita que el discriminante no es `control_raw`.
No se guarda `error`, texto libre, buffer, framing, nonce, ticket, PID, pidfd,
puntero adicional, lector, receptor ni copia del controlador.

La palabra completa ya contiene estado, signo y contador. No se añade un campo
de signo duplicado ni una segunda baseline.

## Fuente autoritativa y momento de captura

La única operación de P1 sigue recibiendo el agregado C5 y pone a cero el
puntero del llamador antes de observar campos.

La prevalidación ocurre antes del CAS de custodia `2→3`. En esa misma ventana,
sin syscall y antes de devolver A1, P1:

1. carga una sola vez `primera` y una sola vez la palabra del observador; esa
   carga es el punto de linealización del sello histórico;
2. valida la pareja discriminante/palabra;
3. si y solo si hay `control_raw`, normaliza el controlador ya observado —sea
   terminal funcional o activo íntegro en las condiciones cerradas posteriores—
   a uno de los cuatro valores canónicos;
4. preasigna y copia ambos sellos;
5. ejecuta el CAS irreversible `ENTREGADO_O3C→RECIBIDO_O4A`;
6. consolida A1 sin volver a cargar palabra ni CONTROL. Un cambio posterior del
   observador pertenece a P3 y no altera el sello histórico. La corrupción de
   propiedad detectada después del CAS sigue siendo AF sin rollback.

No existe captura perezosa en P2. P2 solo consumirá estos valores privados.

### Fuente de `palabraObservada`

La fuente única es el mismo `observadorSenalO3aM38` cuya autoidentidad,
registro, generación y pertenencia P1 ya acredita. La carga se acepta así:

- estado enmascarado exactamente 2 en todos los casos;
- para `senal_raw`, baseline y observado deben conservar estado 2 y la baseline
  signo cero; el contador observado no puede retroceder. La palabra se sella
  completa. Solo delta exactamente uno con signo INT o TERM será traducible;
  delta cero —pareja discriminante/palabra ambigua—, delta mayor que uno,
  signo cero u otro signo convergen a INCIDENTE en P2. Estado distinto de 2 o
  contador menor que baseline es AF;
- para los otros cuatro discriminantes, la palabra debe ser exactamente la
  baseline sellada y su signo cero.

P1 no interpreta la señal ni fija causa.

### Fuente de `canonControlRaw`

Las fuentes únicas son `custodia.primeraCausa` y `custodia.control`. O3c P4 las
produce antes del handoff; P1 las carga después de recibir C5 y antes de su CAS
2→3. P1 no relee el FD. Cuando hay terminal funcional, ambas copias de la causa
deben coincidir exactamente.

Con discriminante `control_raw`:

- un controlador terminal funcional, con causa transportada canónica y estado
  exacto, cotejado con `primeraCausa`, se normaliza a su valor correspondiente;
- EOF limpio canónico conserva `CANCELADO_65`;
- un controlador todavía activo pero íntegro según
  `falloInvarianteActivoPreinicioM38(c)==nil`, exactamente en fase S3, con
  `control.causa`, `custodia.primeraCausa` vacías y `control.fallo==nil`, cuando
  el discriminante ya es `control_raw`, acredita presupuesto/EINTR/EAGAIN no
  iniciable y se normaliza a `PROTOCOLO_65` sin nueva lectura; S1/S2 son fase
  imposible y AF;
- framing, parcial+EOF, secuencia o dominio que el parser ya convirtió en un
  terminal funcional PROTOCOLO conserva `PROTOCOLO_65`;
- terminal interno, causa funcional que no coincide con `primeraCausa`, fase
  imposible, punteros residuales o canon desconocido es AF, nunca causa
  funcional, `VACIO` ni autorización.

La normalización debe reutilizar exclusivamente predicados y tipos privados ya
existentes (`terminalFuncionalValidoPreinicioM38`,
`terminalInternoValidoPreinicioM38` y `causaTransportadaPreinicioM38`). No crea
parser ni acepta texto fuera de la unión cerrada.

Con cualquier discriminante distinto de `control_raw`, `canonControlRaw` debe
ser `VACIO`; no se inspecciona ni copia una causa CONTROL.

## Invariantes que permanecen intactas

- owners observador y lease permanecen O4A;
- lease 3, observador 2, registro, generaciones, TID, baseline y pertenencias
  conservan la comprobación P1;
- agregado, autoridad conjunta, custodia, Cmd, Process, tres pidfd, CONTROL,
  TERMINAL, identidad y snapshot mantienen autoidentidad y sellos previos;
- `primera` y `retornoCont` permanecen separados, exactos e inmutables;
- causa e incidente siguen vacíos al salir de P1;
- CAS de custodia 2→3 único, irreversible y sin rollback;
- nulo, clon, alias y replay no consumen ni tocan recursos;
- corrupción tras consumo produce AF 65/EOF/stdout=0/stderr=0, sin E/S.

No se modifica la máquina A0--AF ni se crea transición de P2.

## Prohibiciones

La corrección de sellos no puede introducir:

- syscall, reloj, deadline, poll, señal, Wait/waitid, drenaje, cierre o escritura
  de TERMINAL, liberación o efecto;
- nueva lectura de CONTROL, observador o pidfd después de P1;
- parser, getter, ticket, API pública, interfaz, serialización o log;
- goroutine, canal, global mutable, `init`, hook, callback o función variable;
- PID/PGID/pidfd/nonce en el sello nuevo;
- código P2/P3, O4b/O4c, O5/O6 o cambios de producción ajenos.

## Implementación posterior autorizable

Nombre exacto: `O4A-P1C-SELLOS-RAW`.

Dependencia: P1B publicado con doble GO y CI 5/5. La base material que P1C debe
contener es `0e750d41cbf72b0f0952341fa18e25474c3269fb` más, exclusivamente, el
cierre documental P1B publicado; dirección comunicará el SHA final P1B como
base exacta al asignar P1C.

Write-set máximo y mínimo de P1C:

1. editar únicamente
   `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/causa_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_autoridad.go`;
2. editar únicamente su prueba existente
   `..._autoridad_test.go`.

No se crea un tercer fichero ni se toca O3/P0/contrato. Objetivo 60--140 líneas
netas adicionales entre ambos; parada si el productivo supera 300 líneas o la
prueba 400; topes contractuales de 650/800 siguen dominando.

## Matriz mínima de P1C

| ID | Oráculo obligatorio |
| --- | --- |
| S01 | `senal_raw` con INT exacto entra A1 y sella palabra completa. |
| S02 | `senal_raw` con TERM exacto entra A1 y sella palabra completa. |
| S03 | estado distinto de 2 o contador regresivo es AF; delta 0 o >1, signo cero/ajeno se sella y P2 lo traducirá exclusivamente a INCIDENTE. |
| S04 | no `senal_raw` exige palabra==baseline y signo cero. |
| S05 | las cuatro causas CONTROL canónicas producen los cuatro enums exactos. |
| S06 | terminal funcional cotejado o CONTROL activo íntegro S3 con causas/fallo vacíos normaliza su canon; S1/S2, terminal interno, primeraCausa adversa o combinación imposible es AF. |
| S07 | no `control_raw` exige `VACIO` y no copia causa CONTROL. |
| S08 | `primera`, raw CONT, owners, recursos y sellos anteriores quedan byte/lógicamente intactos. |
| S09 | clon/alias/replay y corrupción pre/post CAS conservan precedencia y cero efectos. |
| S10 | AST prohíbe syscalls, reloj, parser nuevo, API/getter/log/serialización y P2/P3. |

Normal y race se repiten; los mutantes deben matar al menos: aceptar
palabra-baseline para señal, omitir palabra, truncar signo/contador, intercambiar
INT/TERM, aceptar canon libre, confundir interno con causa funcional, aceptar
primeraCausa adversa, rechazar CONTROL activo íntegro, copiar CONTROL cuando no corresponde, efectuar una
segunda carga de palabra/CONTROL tras el punto de linealización, alterar
primera/raw y omitir copia defensiva/inmutabilidad.

## Criterio de cierre y aristas

P1B cierra solo con dos revisiones independientes `P0=P1=P2=0`, commit
documental pequeño, push normal y CI 5/5 del SHA exacto.

P1C solo se abre después de ese cierre y debe obtener su propio doble GO y CI
5/5. Únicamente entonces se reabre `O4A-P2-SEMILLA`. `O4A-P3-ARBITRAJE`
permanece bloqueada por P2. Esta decisión no cambia porcentajes ni integra nada.
