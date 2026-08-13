# Revisión de seguridad O4AB-P0-ENMIENDA-TERMINALIDAD-STOP

Fecha: 13 de agosto de 2026.

Revisor: agente independiente de seguridad, rama
`revision/o4ab-p0-terminalidad-stop-seguridad-20260813`, sin edición del
worktree ni de la rama productora.

Dictamen: **NO-GO**, `P0=0`, `P1=1`, `P2=0`.

## Corte exacto y alcance

Se revisó exclusivamente el candidato documental
`a89a3228554f53b32f5d81fc8b0438835f35b0f6`, con padre/base
`5345d5d097b51ab3567983f048feabeceaf2957b` y árbol
`1414790aecab87cdc3a2f2123b55377e356faaf9`. Su único commit añade dos
Markdown:

| Documento | Líneas | SHA-256 |
| --- | ---: | --- |
| `docs/portal_vec/enmienda_f0_h0b_c4b2_g2o_o4ab_terminalidad_stop_2026-08-13.md` | 234 | `d1edcd4b1468000577577cbe5f86037c64b6b9ade65e3f9d98d4edc9fa2985f9` |
| `docs/portal_vec/revisiones/checkpoint_f0_h0b_c4b2_g2o_o4ab_terminalidad_stop_2026-08-13.md` | 101 | `fff0092a9c4b3d3c21fa3c2f63163c6f865af4be5781dfa5fccc995628cff528` |

Se releyeron completos O4a, O4b, enmienda, checkpoint y los dos NO-GO del
candidato material `2b7eaf498f8f68a90b66c004f166a62f070b5064`:

- funcional `e20a5fe597d9c172d59b99e75878f8e99e196c76`, 176 líneas,
  SHA-256 `f72faa38f2c4543b46b4d8318a19a87149b84d15e4a3f7044317baa4df96de48`;
- seguridad `a62ee60f70315a1fc0d290290b681c9c3f35227b`, 147 líneas,
  SHA-256 `f01503425807851a9efb3a6542dd110054e387d9760626b3b921538a7f80135e`.

La lectura obligatoria ya completada en esta sesión se revalidó byte a byte.
Código, pruebas, autoridades, estado transversal y métricas permanecen
inmóviles. Este dictamen no acredita código ni descendientes.

## Hallazgo P1 — el preflight no conserva el límite de PARADA_FINAL

O4a autoriza `PARADA_FINAL` como un STOP antes de `finParadaFinal`. O4b
mantiene la regla general de una lectura monotónica después de validar
propiedad/identidad y antes de preparar el permiso del efecto; igualdad vence.

La enmienda comprueba `ahora < finParadaFinal` antes de iniciar el preflight
(líneas 95--100). A continuación el preflight ejecuta al menos los sondeos
separados de primario y reserva y las operaciones internas de permiso lease y
consolidación; para presencia exige además identidad vigente. Todas consumen
tiempo no acotado por una nueva lectura.

Si el preflight concluye con presencia, las líneas 109--119 convierten STOP en
el siguiente syscall y prohíben expresamente reloj o sonda intermedios. Los
mutantes y la parada dura de líneas 204--209 y 223--230 consideran inválido
intercalar ese reloj. La linealización en el último sondeo solo fija si la
terminalidad ocurrió antes o después de la decisión de presencia: ese sondeo
no es una lectura monotónica, no compara su instante con `finParadaFinal` y no
demuestra que el STOP posterior permanezca dentro del límite.

Por tanto existe una ejecución admitida por el texto:

```text
comprobación inicial < finParadaFinal
  -> sondeos/identidad/permisos/consolidaciones del preflight
  -> vence finParadaFinal
  -> presencia acreditada
  -> STOP como siguiente syscall, ya tardío
```

El resultado posterior puede contener una marca dentro del límite para
`TERMINAL`, pero eso no corrige el instante del STOP ya intentado. Tampoco la
cardinalidad máxima impide la señal tardía. La enmienda afirma que no cambia
deadlines, aunque en esta rama elimina materialmente la última comprobación
temporal que el contrato publicado exige antes del efecto.

### Impacto y corrección requerida

El defecto permite ejercer un permiso de señal fuera de su vigencia exacta.
Viola privilegio mínimo, vinculación de autorización a vigencia y la regla de
que indisponibilidad o demora no habilitan un efecto. Se clasifica P1 porque el
destino continúa siendo el pidfd primario con identidad/lease acreditados y no
se demuestra señal a proceso ajeno, fuga de autoridad o exposición de datos.

La corrección documental debe integrar en el propio preflight una comprobación
monotónica final, posterior a toda evidencia necesaria de presencia. Solo si
esa lectura acredita `ahora.Before(finParadaFinal)` puede declararse presencia
autorizante; desde esa comprobación STOP debe ser el siguiente syscall literal,
sin otro sondeo, reloj, log ni asignación falible. Igualdad/vencimiento debe ir
a OBF con cero STOP y cero resultado parcial. Así se preservan tanto la regla
publicada de deadline como la inmediatez: el reloj final forma parte del
preflight y no se intercala después de que este haya quedado consolidado.

La matriz y los mutantes deben probar demora que cruce el borde durante el
preflight, igualdad exacta en la última lectura, cero STOP vencido y omisión o
adelanto de esa lectura final.

## Controles conformes que no compensan el P1

- `TERMINAL` post-CONT antes o exactamente en gracia llega A7; presencia
  anterior espera y presencia exactamente en gracia abre el permiso final;
- presencia anterior no se reutiliza como evidencia final: el preflight vuelve
  a acreditar primario, reserva e identidad, cada syscall con permiso lease;
- terminalidad en preflight produce cardinalidad real 0 y STOP/KILL/incidente
  cero; los raws cero son campos canónicos ausentes, no éxito fabricado;
- presencia preflight permite como máximo un STOP; terminalidad posterior al
  STOP, cardinalidad 1 y raw cero llega A7 sin KILL/incidente;
- duda física, discordancia, flags, identidad, comenzar o consolidación
  inciertos son OBF sin resultado ni efecto posterior;
- PARADA_INICIAL conserva cardinalidad 1, no acepta `TERMINAL` y mantiene su
  rama `NO_ESTABLE`/KILL rápido;
- no se añaden Wait, waitid/wait4, señal cero, PID/PGID, fallback, parser,
  recurso, API, log, dato o credencial;
- el DAG mantiene la enmienda antes del nuevo O4A-P4, este antes de O4B-P1 y
  O4A-P5/O4c cerrados.

## Puertas reproducidas

- HEAD, padre, árbol, ancestry, commit único y write-set de dos altas: exactos;
- líneas y SHA-256 de enmienda, checkpoint, O4a, O4b y ambos NO-GO: exactos;
- todos los enlaces Markdown locales: verdes;
- `git diff --check` sobre
  `5345d5d097b51ab3567983f048feabeceaf2957b..a89a3228554f53b32f5d81fc8b0438835f35b0f6`:
  verde;
- búsqueda focal de secretos/credenciales/DSN/datos personales: solo menciones
  normativas;
- Gitleaks sobre el mismo rango: un commit, 16,57 KB, cero filtraciones;
- worktree candidato limpio antes del acta.

Go normal/race, gofmt, vet, PostgreSQL y E2E no aplican a dos altas Markdown.
No se declaran ejecutados ni sustituyen la revisión semántica.

## Cierre

El candidato exacto permanece **NO-GO**. Requiere una enmienda documental que
cierre el deadline durante el preflight y dos revisiones independientes sobre
los nuevos bytes. No se autoriza nuevo O4A-P4, O4B-P1, O4A-P5, O4c, O5/O6,
integración, publicación, despliegue, producción ni cambio de métricas.
