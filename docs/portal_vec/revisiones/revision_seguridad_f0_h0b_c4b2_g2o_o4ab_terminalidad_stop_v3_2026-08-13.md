# Revisión de seguridad O4AB-P0-ENMIENDA-TERMINALIDAD-STOP V3

Fecha: 13 de agosto de 2026.

Revisor: agente independiente de seguridad, rama
`revision/o4ab-p0-v3-seguridad-20260813`, sin edición del worktree ni de la
rama productora.

Dictamen: **GO**, `P0=0`, `P1=0`, `P2=0`.

## Corte exacto y alcance

Se revisó exclusivamente el candidato documental
`2abba91f6fdefbbbe90551ae622581f17d92b17a`, con padre V2
`9de3ba320dfa4beede58e5a1d34aa02a3459f073`, base
`5345d5d097b51ab3567983f048feabeceaf2957b` y árbol
`c7355fafe42630f3a406692e4b730bd15ed2f29c`. Su único commit modifica dos
Markdown con `+56/-28`:

| Documento | Delta V3 | Líneas | SHA-256 |
| --- | ---: | ---: | --- |
| `docs/portal_vec/enmienda_f0_h0b_c4b2_g2o_o4ab_terminalidad_stop_2026-08-13.md` | `+33/-15` | 282 | `2b44bd03d0422a4aecff78ad6400873ecb17686901c40a6b0a68bdabd91d769f` |
| `docs/portal_vec/revisiones/checkpoint_f0_h0b_c4b2_g2o_o4ab_terminalidad_stop_2026-08-13.md` | `+23/-13` | 125 | `ce5cf0d0248cf1ac98e7580ec210e434b7f1610cb70515606b8e2f035134bc6a` |

Se releyeron completos la enmienda V3, su checkpoint y las autoridades O4a y
O4b. Se recontrastaron las autoridades O3a/O3b/O3c conservadas:

| Autoridad | Líneas | SHA-256 |
| --- | ---: | --- |
| O3a | 800 | `39514c827486f385db89e2117ab4e8a2f43e0be7ade98158a1ab0c7a49685a90` |
| O3b | 574 | `d9aa33eddf90da2fb0e7f1aac239a18797e70b8afe7f9fe3024f1a9e5f401ada` |
| O3c | 609 | `f47395d68fa3f9e39e118f81b07fde8d8792aa61d4820dfb676ff4c7216515b6` |
| O4a | 535 | `ffe3d570b3fe51be96948f8aeb0b163e2ff071d2c2cca4fbdba852a5f9dafebc` |
| O4b | 443 | `675d33b6f96ef441843721effd332a82242ed9257a2af2246c75ed22f2984c7f` |

También se releyeron las cinco actas `NO-GO` y el `GO` de seguridad V2 que
abren este corte:

| Objeto | Commit | Líneas | SHA-256 | Resultado |
| --- | --- | ---: | --- | --- |
| O4A-P4 funcional | `e20a5fe597d9c172d59b99e75878f8e99e196c76` | 176 | `f72faa38f2c4543b46b4d8318a19a87149b84d15e4a3f7044317baa4df96de48` | `NO-GO`, `P0=1,P1=2,P2=0` |
| O4A-P4 seguridad | `a62ee60f70315a1fc0d290290b681c9c3f35227b` | 147 | `f01503425807851a9efb3a6542dd110054e387d9760626b3b921538a7f80135e` | `NO-GO`, `P0=0,P1=1,P2=0` |
| O4AB V1 funcional | `cdcedc44b31f0f997139f0e9e1210f29d1eb08ca` | 160 | `76ee47785d27a01f61c07e750e021a88abee701a713b5bd1516cb8f97b7466cf` | `NO-GO`, `P0=0,P1=1,P2=0` |
| O4AB V1 seguridad | `e7e06423941807a41c40946e5a4af12e1b30bb11` | 133 | `e968004722ebd94a86c6a1450ee46baef811067a0372ecee99f979a4d2072a72` | `NO-GO`, `P0=0,P1=1,P2=0` |
| O4AB V2 funcional | `876b7c8e330dc28003dab1d0057ca4bc9d83a727` | 189 | `458ac2e24c2cd55a03a1ec91fbfff0fabd258da9edf0bdc8112d4332d974c8b9` | `NO-GO`, `P0=0,P1=1,P2=0` |
| O4AB V2 seguridad | `569cc9ccfb94d3f195a6c08340b0e0e10429da96` | 140 | `e9d2d17da4b994103db9e8e1a5ecc73de8fbcd6a5b4ac48e514304cb2aeeea30` | `GO`, `P0=P1=P2=0` |

La lectura obligatoria completada en este mismo contexto se revalidó. Código,
pruebas, autoridades, estado transversal, métricas y candidatos históricos
permanecen inmóviles. Este dictamen no acredita código ni descendientes.

## Cierre del P1 funcional de V2

V2 corrigió el STOP tardío de V1, pero afirmó que la lectura temporal
`ahoraFinal` linealizaba conjuntamente presencia y vigencia. Esa lectura no
observa la condición física y no podía trasladar hasta su instante una
no-terminalidad ya sondeada.

V3 conserva el orden operativo seguro y separa expresamente los hitos:

1. toda la evidencia física de primario, reserva e identidad se consolida con
   permisos lease separados;
2. la presencia se linealiza exclusivamente en el último sondeo consolidado
   que completa esa acreditación física;
3. la lectura monotónica posterior `ahoraFinal` no vuelve a observar ni mueve
   la presencia: acredita solo que el permiso continúa vigente;
4. si `ahoraFinal >= finParadaFinal`, domina OBF con cero resultado y efecto;
5. si queda estrictamente antes, solo se prepara el permiso lease y STOP es la
   siguiente syscall literal.

Los dos hechos son deliberadamente ordenados, no simultáneos. Puesto que las
referencias pertenecen a la misma identidad y la terminalidad pidfd es una
condición monotónica, el último sondeo verde completa la decisión física. Una
terminalidad posterior a ese sondeo, incluso en la ventana sondeo→reloj, está
después de esa decisión y no la revoca retroactivamente. El reloj verde limita
la vigencia; no pretende probar presencia.

No hay contradicción con privilegio mínimo ni con la ausencia de intercalación:
la autorización se apoya en la última evidencia física disponible y, tras el
reloj, no se permite otra sonda, reloj, validación, espera, log o asignación
falible. Es el mismo modelo publicado por O3c y O4b: elegibilidad fijada en una
lectura estricta y efecto como syscall literal siguiente, sin sobredeclarar el
instante interno en que el kernel comienza a ejecutarla.

Si la lectura final vence, la autorización permanece consumida; no se sella ni
devuelve resultado parcial; no existen cardinalidad, raw, marca o incidente
ordinarios; O4a no toma otra transición desde A5; y rige 65/EOF/stdout=0/
stderr=0, sin señal, cierre, log, limpieza ni efecto posterior. E07/E08 y los
mutantes específicos impiden volver a atribuir presencia al reloj, exigir una
falsa monotonía hasta él o clasificar como previa terminalidad posterior al
sondeo.

## Privilegio, cardinalidad y fronteras

- O4a conserva causa, precedencia, tiempo, etapa y autorización; O4b solo
  acredita la condición física y ejecuta el efecto autorizado. La prevalencia
  sustituye únicamente las frases O4b incompatibles con las aristas terminales
  ya definidas por O4a.
- La presencia post-CONT anterior no se reutiliza al vencer gracia. El
  preflight vuelve a acreditar primario, reserva e identidad; duda física,
  discordancia, flags o lease inciertos son OBF, nunca presencia o éxito.
- Terminalidad acreditada en el preflight da cardinalidad real 0 y campos raw
  canónicos ausentes: cero STOP, KILL e incidente. No se fabrica raw de éxito.
- Presencia y reloj verde autorizan como máximo un STOP. Si STOP raw cero
  habilita evidencia posterior terminal, el resultado tiene cardinalidad 1 y
  lleva A7 sin KILL ni incidente. Raw no cero conserva `SIN_EVIDENCIA`, cero
  retry y su rama publicada; no se reinterpreta como terminalidad.
- `cardinalidadMaxima=1` no se confunde con la cardinalidad real 0/1. Permiso y
  resultado siguen privados, autoidénticos, one-shot y ligados a slot, etapa,
  operación, deadline, generación y TID. Alias, clon, replay, forja y carreras
  permiten un ganador o fallan cerrados, sin rollback.
- `PARADA_INICIAL` mantiene cardinalidad 1 y su semántica distinta:
  terminalidad controlable tras STOP se normaliza a `NO_ESTABLE`; `TERMINAL`
  o cardinalidad 0 forjados son AF.
- Fallo de `comenzar` o duda de `consolidarCritico` es OBF, no raw ni permiso
  posterior. Cada syscall conserva permiso lease propio y el pidfd primario es
  el único señalizable; reserva y handle siguen como testigos, sin promoción.
- No aparecen Wait/waitid/wait4, señal cero, PID/PGID, fallback, repetición,
  parser, FD, cierre, limpieza, goroutine, canal, API, getter, log, dato humano,
  secreto, HTTP, SQL, red, Docker ni producción.

El DAG continúa acíclico y cerrado: V3 necesita doble revisión, publicación y
CI 5/5 antes de un nuevo O4A-P4. O4B-P1, O4A-P5, O4C-P1 y O5/O6 permanecen
bloqueados; la corrección O4C-P0 independiente no acredita este corte.

## Puertas reproducidas

- `git rev-parse HEAD HEAD^ 'HEAD^{tree}'`: candidato, padre y árbol exactos;
- `git merge-base --is-ancestor` para base→V3 y V2→V3: verde;
- `git rev-list --count`, `git diff --name-status` y `--numstat`: un commit,
  solo dos Markdown y `+56/-28`;
- `sha256sum` y `wc -l`: huellas y líneas de V3, checkpoint, O3a/O3b/O3c,
  O4a/O4b y las seis actas históricas exactas;
- enlace Markdown local del checkpoint: verde;
- `git diff --check` en
  `9de3ba320dfa4beede58e5a1d34aa02a3459f073..2abba91f6fdefbbbe90551ae622581f17d92b17a`
  y en
  `5345d5d097b51ab3567983f048feabeceaf2957b..2abba91f6fdefbbbe90551ae622581f17d92b17a`:
  verde;
- búsqueda focal de secretos, credenciales, DSN y datos personales: solo
  menciones normativas;
- Gitleaks V2→V3: un commit, 4,05 KB, cero filtraciones;
- Gitleaks base→V3: tres commits, 25,90 KB, cero filtraciones;
- worktree candidato limpio antes de añadir esta acta.

Go normal/race, gofmt, vet, PostgreSQL y E2E no aplican al delta exclusivo de
documentación. No se declaran ejecutados ni se sustituyen por las puertas
documentales.

## Cierre

El candidato documental exacto recibe **GO**, `P0=P1=P2=0`. Este dictamen no
es autoaprobación ni sustituye la revisión funcional independiente sobre los
mismos bytes. Dirección conserva publicación, CI y cualquier acreditación.
No se autoriza código O4A-P4, O4B-P1, O4A-P5, O4c, O5/O6, integración, push,
despliegue, producción ni cambio de métricas.
