# Revisión funcional O4AB-P0-ENMIENDA-TERMINALIDAD-STOP

Fecha: 13 de agosto de 2026.

Identificador: `O4AB-P0-ENMIENDA-TERMINALIDAD-STOP-FUNCIONAL`.

Estado: **NO-GO, P0=0, P1=1, P2=0**.

Este dictamen independiente revisa exclusivamente el candidato
`a89a3228554f53b32f5d81fc8b0438835f35b0f6`, padre/base exacto
`5345d5d097b51ab3567983f048feabeceaf2957b` y árbol
`1414790aecab87cdc3a2f2123b55377e356faaf9`. No revisa ni acredita código,
`2b7eaf4`, ningún descendiente, O4B-P1, O4A-P5, O4c, O5/O6, integración,
publicación, producción, despliegue ni métricas.

## Independencia, lectura y write-set

La revisión se realizó desde el worktree exclusivo
`o4ab-p0-enmienda-terminalidad-stop-funcional-20260813`, rama
`revision/o4ab-p0-enmienda-terminalidad-stop-funcional-20260813`, creada en el
SHA objetivo. No se editó ni movió la rama productora. El único write-set de
revisión es esta acta.

Antes de editar se releyó `AGENTS.md` completo y se comprobó que las lecturas
obligatorias ya completadas en esta misma sesión conservan exactamente sus
bytes. Se releyeron íntegros O4a, O4b, la enmienda, el checkpoint y los dos
NO-GO de `2b7eaf4`: funcional
`e20a5fe597d9c172d59b99e75878f8e99e196c76` y seguridad
`a62ee60f70315a1fc0d290290b681c9c3f35227b`.

## Identidad, autoridades y delta

El candidato es un único commit sobre la base declarada. Su delta contiene
solo dos altas Markdown:

| Documento | Líneas | SHA-256 |
| --- | ---: | --- |
| `docs/portal_vec/enmienda_f0_h0b_c4b2_g2o_o4ab_terminalidad_stop_2026-08-13.md` | 234 | `d1edcd4b1468000577577cbe5f86037c64b6b9ade65e3f9d98d4edc9fa2985f9` |
| `docs/portal_vec/revisiones/checkpoint_f0_h0b_c4b2_g2o_o4ab_terminalidad_stop_2026-08-13.md` | 101 | `fff0092a9c4b3d3c21fa3c2f63163c6f865af4be5781dfa5fccc995628cff528` |

El total es `+335/-0`. La rama productora estaba limpia. O4a conserva 535
líneas y SHA-256
`ffe3d570b3fe51be96948f8aeb0b163e2ff071d2c2cca4fbdba852a5f9dafebc`;
O4b conserva 443 y
`675d33b6f96ef441843721effd332a82242ed9257a2af2246c75ed22f2984c7f`.
Las actas NO-GO citadas existen, tienen padre `2b7eaf4` y hashes de contenido
`f72faa38f2c4543b46b4d8318a19a87149b84d15e4a3f7044317baa4df96de48`
y `f01503425807851a9efb3a6542dd110054e387d9760626b3b921538a7f80135e`.

No cambiaron código, pruebas, O3, O4a, O4b, O4c, herramientas, runner,
workflows, SQL, `AGENTS.md`, handoffs, roadmap, ledger transversal, candidatos
históricos ni métricas.

## Auditoría requisito por requisito

| Requisito revisado | Resultado |
| --- | --- |
| Prevalencia: O4a decide causa/tiempo/etapa y O4b solo evidencia/efecto | Conforme y explícito. |
| `TERMINAL` post-CONT anterior o igual a `finGracia` | Conforme: A7, drenaje cooperativo y cero señal. |
| `GRUPO_PRESENTE` anterior a gracia | Conforme: vuelve A3 y no prueba presencia futura. |
| `GRUPO_PRESENTE` en igualdad | Conforme: habilita una sola autorización final condicional. |
| Presencia anterior revalidada al borde | Conforme mediante preflight final primario/reserva e identidad. |
| Preflight terminal | Conforme: resultado `TERMINAL`, cardinalidad real 0, raws cero y cero STOP/KILL/incidente. |
| Preflight presente | **No conforme en el borde de `finParadaFinal`: puede cruzar el límite antes de STOP sin nueva comprobación.** |
| `cardinalidadMaxima=1` frente a cardinalidad real 0/1 | Conforme y distinguida de los raws. |
| Terminalidad posterior a STOP final | Conforme: `TERMINAL`, cardinalidad 1, A7 y cero KILL/incidente. |
| STOP final estable/no estable/raw error | Conserva las tres ramas O4a publicadas. |
| Asimetría de `PARADA_INICIAL` | Conforme y deliberada: cardinal 1, terminalidad se normaliza a `NO_ESTABLE`, forja `TERMINAL` es AF. |
| Marcas, raws, replay, ownership y duda física | Cerrados salvo la salida temporal del hallazgo P1. |
| DAG y materialización secuencial | Acotados, pero el contrato no puede materializar P3 sin resolver el borde y su resultado. |
| Cero Wait, señal cero, fallback, parser, getter, datos o alcance transversal | Conforme. |

## Hallazgo P1-01 — el preflight puede autorizar STOP después del límite final

La enmienda declara que los límites absolutos O4a/O4b permanecen vigentes
(líneas 54--58). Sin embargo, para `PARADA_FINAL` acredita
`ahora < finParadaFinal` **antes** de ejecutar el preflight no recolector
(líneas 95--100). Ese preflight requiere varias acreditaciones y syscalls con
permisos lease separados y no tiene una duración máxima que garantice que
termine antes del borde.

Si el preflight termina en presencia, la enmienda ordena que STOP sea el
siguiente syscall funcional y prohíbe reloj, sonda, log o asignación
intermedia (líneas 107--119). La familia mutante repite expresamente la
prohibición de intercalar reloj entre presencia y STOP (líneas 204--209).
Por tanto existe esta ejecución admitida:

```text
ahora < finParadaFinal
  -> preflight comienza
  -> el reloj cruza finParadaFinal durante sus observaciones
  -> presencia acreditada
  -> STOP siguiente, ya fuera de vigencia
```

La evidencia posterior no puede convertir retroactivamente ese STOP en un
efecto dentro del deadline. Se viola el límite absoluto y el privilegio
temporal mínimo de la autorización.

Se clasifica P1, no P0: pidfd, identidad, grupo, owner y lease permanecen
acreditados y no se demuestra señal a un proceso ajeno, pero sí una señal
irreversible fuera de la vigencia contractual. El defecto bloquea la
enmienda.

### Corrección accionable

La enmienda debe fijar sin ambigüedad:

1. todas las observaciones físicas de presencia del preflight;
2. una lectura monotónica final, integrada en el propio preflight, que exija
   `ahora < finParadaFinal` después de esas observaciones;
3. solo con ese verde, STOP como siguiente syscall, sin operación intermedia;
4. en igualdad o vencimiento, cero STOP y una salida cerrada con
   cardinalidad, raws, marca, incidente y transición O4a definidos. Hoy
   cardinalidad 0 está reservada exclusivamente a `TERMINAL`, por lo que no
   puede reutilizarse silenciosamente para el vencimiento;
5. un oráculo y mutante específicos donde el preflight cruza
   `finParadaFinal`.

La linealización de presencia puede mantenerse en el último sondeo: la
comprobación temporal posterior solo limita la vigencia, y después de su verde
STOP sigue siendo el siguiente syscall. La decisión debe indicar también si el
vencimiento produce resultado consumible o OBF; no puede dejarlo implícito.

## Puertas reproducidas

```bash
git show -s --format='%H%n%P%n%T%n%s' a89a3228554f53b32f5d81fc8b0438835f35b0f6
git diff --name-status 5345d5d097b51ab3567983f048feabeceaf2957b..a89a3228554f53b32f5d81fc8b0438835f35b0f6
git diff --numstat 5345d5d097b51ab3567983f048feabeceaf2957b..a89a3228554f53b32f5d81fc8b0438835f35b0f6
wc -l docs/portal_vec/enmienda_f0_h0b_c4b2_g2o_o4ab_terminalidad_stop_2026-08-13.md \
  docs/portal_vec/revisiones/checkpoint_f0_h0b_c4b2_g2o_o4ab_terminalidad_stop_2026-08-13.md
sha256sum docs/portal_vec/enmienda_f0_h0b_c4b2_g2o_o4ab_terminalidad_stop_2026-08-13.md \
  docs/portal_vec/revisiones/checkpoint_f0_h0b_c4b2_g2o_o4ab_terminalidad_stop_2026-08-13.md
git diff --check 5345d5d097b51ab3567983f048feabeceaf2957b..a89a3228554f53b32f5d81fc8b0438835f35b0f6
gitleaks git --no-banner --redact \
  --log-opts='5345d5d097b51ab3567983f048feabeceaf2957b..a89a3228554f53b32f5d81fc8b0438835f35b0f6'
```

Resultados: commit, padre, árbol, ramas, productor limpio, dos altas,
`+335/-0`, líneas, hashes, O4a/O4b, ambos NO-GO, enlaces locales y
`git diff --check` verdes. Gitleaks recorrió un commit y 16,57 KB sin
hallazgos. La búsqueda focal de claves, tokens, DSN y credenciales tampoco
encontró material sensible.

No hay paquete Go afectado: normal/race, gofmt, vet, mutantes, PostgreSQL,
Docker, HTTP y E2E no aplican a estas dos altas Markdown. No se declaran
ejecutados ni se usan para compensar el NO-GO.

## Seguridad, privacidad y relevo

La enmienda reduce correctamente STOP/KILL ante terminalidad ya acreditada,
no añade autoridad, datos humanos, secreto, API, red ni persistencia y conserva
la separación O4a/O4b. El P1 temporal impide, no obstante, aprobar sus bytes.

El relevo inequívoco es **NO-GO**. `a89a322` debe conservarse como candidato
histórico sin autoaprobación. La siguiente acción es una corrección documental
acotada del preflight y su borde, seguida de dos revisiones independientes
sobre el nuevo SHA. No se hizo push, deploy, cambio de credenciales, estado
transversal, porcentajes ni producción.
