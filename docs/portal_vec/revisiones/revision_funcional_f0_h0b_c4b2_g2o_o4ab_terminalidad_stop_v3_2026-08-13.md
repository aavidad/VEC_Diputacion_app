# Revisión funcional O4AB-P0-ENMIENDA-TERMINALIDAD-STOP V3

Fecha: 13 de agosto de 2026.

Identificador: `O4AB-P0-ENMIENDA-TERMINALIDAD-STOP-V3-FUNCIONAL`.

Estado: **GO, P0=0, P1=0, P2=0**.

Este dictamen independiente revisa exclusivamente el candidato documental
`2abba91f6fdefbbbe90551ae622581f17d92b17a`, hijo de V2
`9de3ba320dfa4beede58e5a1d34aa02a3459f073`, con base
`5345d5d097b51ab3567983f048feabeceaf2957b` y árbol
`c7355fafe42630f3a406692e4b730bd15ed2f29c`. No revisa ni acredita código,
`2b7eaf4`, ningún otro descendiente, O4B-P1, O4A-P5, O4c, O5/O6,
integración, publicación, producción, despliegue ni métricas.

## Independencia, lectura y write-set

La revisión se realizó desde el worktree exclusivo
`o4ab-p0-v3-linealizacion-funcional-20260813`, rama
`revision/o4ab-p0-v3-linealizacion-funcional-20260813`, creada en el SHA
objetivo. No se editó ni movió la rama productora. El único write-set de
revisión es esta acta.

Antes de editar se leyó `AGENTS.md` completo y se revalidaron las lecturas
obligatorias ya completadas en esta sesión: relevo, mapa/roadmap, tablero,
relevo de contratación temporal, matriz normativa, expediente y hoja de ruta
RRHH. Se releyeron completos O4a, O4b, la enmienda V3 y su checkpoint; se
contrastaron O3a/O3b/O3c para lease, primitivas, identidad y custodia; y se
releyeron las cinco actas `NO-GO` y el `GO` de seguridad V2 que forman la
genealogía de esta corrección.

## Identidad y huellas reproducidas

V3 es un único commit sobre V2. Su delta modifica solo:

| Documento | Delta V3 | Líneas finales | SHA-256 final |
| --- | ---: | ---: | --- |
| `docs/portal_vec/enmienda_f0_h0b_c4b2_g2o_o4ab_terminalidad_stop_2026-08-13.md` | `+33/-15` | 282 | `2b44bd03d0422a4aecff78ad6400873ecb17686901c40a6b0a68bdabd91d769f` |
| `docs/portal_vec/revisiones/checkpoint_f0_h0b_c4b2_g2o_o4ab_terminalidad_stop_2026-08-13.md` | `+23/-13` | 125 | `ce5cf0d0248cf1ac98e7580ec210e434b7f1610cb70515606b8e2f035134bc6a` |

El total V3 es `+56/-28`; el rango acumulado desde la base contiene solo esas
dos altas, `+407/-0`, en tres commits lineales. O4a conserva 535 líneas y
SHA-256
`ffe3d570b3fe51be96948f8aeb0b163e2ff071d2c2cca4fbdba852a5f9dafebc`;
O4b conserva 443 y
`675d33b6f96ef441843721effd332a82242ed9257a2af2246c75ed22f2984c7f`.

Las seis actas históricas existen en sus commits y padres exactos:

| Objeto y dictamen | Commit | Padre | Líneas | SHA-256 del acta |
| --- | --- | --- | ---: | --- |
| O4A-P4 funcional, `NO-GO P0=1 P1=2 P2=0` | `e20a5fe597d9c172d59b99e75878f8e99e196c76` | `2b7eaf498f8f68a90b66c004f166a62f070b5064` | 176 | `f72faa38f2c4543b46b4d8318a19a87149b84d15e4a3f7044317baa4df96de48` |
| O4A-P4 seguridad, `NO-GO P0=0 P1=1 P2=0` | `a62ee60f70315a1fc0d290290b681c9c3f35227b` | `2b7eaf498f8f68a90b66c004f166a62f070b5064` | 147 | `f01503425807851a9efb3a6542dd110054e387d9760626b3b921538a7f80135e` |
| O4AB V1 funcional, `NO-GO P0=0 P1=1 P2=0` | `cdcedc44b31f0f997139f0e9e1210f29d1eb08ca` | `a89a3228554f53b32f5d81fc8b0438835f35b0f6` | 160 | `76ee47785d27a01f61c07e750e021a88abee701a713b5bd1516cb8f97b7466cf` |
| O4AB V1 seguridad, `NO-GO P0=0 P1=1 P2=0` | `e7e06423941807a41c40946e5a4af12e1b30bb11` | `a89a3228554f53b32f5d81fc8b0438835f35b0f6` | 133 | `e968004722ebd94a86c6a1450ee46baef811067a0372ecee99f979a4d2072a72` |
| O4AB V2 funcional, `NO-GO P0=0 P1=1 P2=0` | `876b7c8e330dc28003dab1d0057ca4bc9d83a727` | `9de3ba320dfa4beede58e5a1d34aa02a3459f073` | 189 | `458ac2e24c2cd55a03a1ec91fbfff0fabd258da9edf0bdc8112d4332d974c8b9` |
| O4AB V2 seguridad, `GO P0=P1=P2=0` | `569cc9ccfb94d3f195a6c08340b0e0e10429da96` | `9de3ba320dfa4beede58e5a1d34aa02a3459f073` | 140 | `e9d2d17da4b994103db9e8e1a5ecc73de8fbcd6a5b4ac48e514304cb2aeeea30` |

La rama productora estaba limpia. No cambiaron código, pruebas, O3, O4a,
O4b, O4c, herramientas, runner, workflows, SQL, `AGENTS.md`, handoffs,
roadmap, ledger transversal, candidatos históricos ni métricas.

## Auditoría funcional completa

| Requisito vigente | Resultado |
| --- | --- |
| O4a conserva causa, precedencia, reloj, etapa y autorización; O4b solo evidencia y efecto | Conforme; la prevalencia queda limitada a las aristas incompatibles. |
| `TERMINAL` post-CONT antes o exactamente en `finGracia` | Conforme: A7, drenaje cooperativo y cero señal posterior. |
| Presencia anterior frente al borde exacto de gracia | Conforme: la anterior espera; la igualdad abre una autorización final condicional. |
| Presencia anterior revalidada al llegar al borde | Conforme: el preflight vuelve a acreditar referencias e identidad. |
| Terminalidad en preflight | Conforme: resultado `TERMINAL`, cardinalidad real 0, raws cero y cero STOP/KILL/incidente. |
| Presencia en preflight | Conforme: permite como máximo un STOP, no obliga ante terminalidad acreditada. |
| `cardinalidadMaxima=1` frente a resultado real 0/1 | Conforme; no se mezcla con raw ni se fabrica un intento. |
| Lectura final después de toda evidencia física | Conforme: `ahoraFinal` acredita solo vigencia; igualdad/vencimiento es OBF sin resultado ni efecto. |
| Punto causal de presencia | Conforme: queda exclusivamente en el último sondeo consolidado y no se traslada al reloj. |
| Terminalidad entre sondeo y reloj | Conforme: es posterior a la decisión física; con reloj verde no revoca STOP. |
| Secuencia posterior al reloj | Conforme: sobre preasignado, preparar permiso y STOP como siguiente syscall, sin sonda/reloj/log/asignación falible. |
| Terminalidad posterior a STOP final | Conforme: cardinalidad 1, A7 y cero KILL/incidente. |
| STOP final estable, no estable o raw no cero | Conserva las tres ramas exactas y sus límites cooperativos. |
| Asimetría `PARADA_INICIAL` | Conforme: cardinalidad 1, `TERMINAL` no admitido y terminalidad controlable como `NO_ESTABLE`. |
| Raws, marcas, owners, lease, replay, duda física y fatalidad | Cerrados y con denegación predeterminada. |
| Matriz y mutantes | Incluyen cruce temporal, igualdad, atribución causal, carrera sondeo→reloj, terminalidad 0/1 y ramas inicial/final. |
| DAG y alcance | Secuencial, sin retorno O4→O3 ni apertura de material/descendientes. |

## Cierre de los hallazgos anteriores

V3 conserva las correcciones ya introducidas por V1/V2 para terminalidad
final, borde de gracia y plazo absoluto. La diferencia funcional decisiva está
en sus líneas 142--155:

1. la presencia se fija en la última evidencia física consolidada;
2. `ahoraFinal`, ejecutado después, solo demuestra vigencia estricta;
3. no se afirma que ambos hechos sean simultáneos;
4. terminalidad posterior al sondeo, incluso anterior al reloj, queda después
   de la decisión física;
5. solo un reloj verde permite preparar el permiso y hacer de STOP la siguiente
   syscall; igualdad o vencimiento conserva OBF y cero efecto.

Por tanto desaparece la contradicción V2: el reloj no acredita
no-terminalidad ni mueve su instante causal. Tampoco reaparece la ventana V1:
la comprobación temporal sigue después de toda evidencia y precede
inmediatamente al permiso/STOP. Los oráculos E07/E08 y los mutantes que
atribuyen presencia al reloj, exigen una falsa monotonía hasta el reloj o
reclasifican terminalidad posterior al sondeo hacen observable esta frontera.

La elección causal no elimina carreras físicas ni promete observar el instante
interno del kernel. Define cuál observación gana, conserva el destino bajo
pidfd/identidad/lease y hace que la evidencia terminal posterior a STOP evite
KILL. No se encontró una arista sin resultado, una cardinalidad ambigua ni una
ampliación de autoridad.

## Puertas reproducidas

```bash
git show -s --format='%H%n%P%n%T%n%s' 2abba91f6fdefbbbe90551ae622581f17d92b17a
git rev-list --count 9de3ba320dfa4beede58e5a1d34aa02a3459f073..2abba91f6fdefbbbe90551ae622581f17d92b17a
git rev-list --count 5345d5d097b51ab3567983f048feabeceaf2957b..2abba91f6fdefbbbe90551ae622581f17d92b17a
git diff --name-status 9de3ba320dfa4beede58e5a1d34aa02a3459f073..2abba91f6fdefbbbe90551ae622581f17d92b17a
git diff --numstat 9de3ba320dfa4beede58e5a1d34aa02a3459f073..2abba91f6fdefbbbe90551ae622581f17d92b17a
wc -l docs/portal_vec/enmienda_f0_h0b_c4b2_g2o_o4ab_terminalidad_stop_2026-08-13.md \
  docs/portal_vec/revisiones/checkpoint_f0_h0b_c4b2_g2o_o4ab_terminalidad_stop_2026-08-13.md
sha256sum docs/portal_vec/enmienda_f0_h0b_c4b2_g2o_o4ab_terminalidad_stop_2026-08-13.md \
  docs/portal_vec/revisiones/checkpoint_f0_h0b_c4b2_g2o_o4ab_terminalidad_stop_2026-08-13.md
git diff --check 9de3ba320dfa4beede58e5a1d34aa02a3459f073..2abba91f6fdefbbbe90551ae622581f17d92b17a
git diff --check 5345d5d097b51ab3567983f048feabeceaf2957b..2abba91f6fdefbbbe90551ae622581f17d92b17a
gitleaks git --no-banner --redact \
  --log-opts='9de3ba320dfa4beede58e5a1d34aa02a3459f073..2abba91f6fdefbbbe90551ae622581f17d92b17a'
gitleaks git --no-banner --redact \
  --log-opts='5345d5d097b51ab3567983f048feabeceaf2957b..2abba91f6fdefbbbe90551ae622581f17d92b17a'
```

Resultados: commit, padre, árbol, ancestry, un commit V3, tres commits desde
base, write-set, deltas, líneas, hashes, autoridades, seis actas históricas,
enlaces Markdown locales, productor limpio, `git diff --check` y búsqueda
focal de secretos verdes. Gitleaks recorrió un commit/4,05 KB para V3 y tres
commits/25,90 KB para el rango acumulado, sin filtraciones.

No existe paquete Go afectado: normal/race, gofmt, vet, mutantes, PostgreSQL,
Docker, HTTP y E2E no aplican a dos modificaciones Markdown. No se declaran
ejecutados ni se sustituyen por puertas documentales.

## Seguridad, privacidad y relevo

La enmienda reduce permisos ante terminalidad acreditada y conserva
autorización opaca one-shot, deadline absoluto, identidad pidfd, lease,
fatalidad cerrada y cero Wait/señal cero/fallback. No añade secreto,
credencial, dato humano, API, red, persistencia o interfaz; i18n y
accesibilidad no aplican.

El candidato exacto recibe **GO funcional, P0=P1=P2=0**. Este dictamen es solo
una de las dos revisiones independientes exigidas y no autoacredita el
candidato, el código ni sus descendientes. Dirección debe comprobar el segundo
dictamen sobre estos mismos bytes antes de publicación y CI 5/5. No se hizo
push, deploy, cambio de credenciales, estado transversal, porcentajes ni
producción.
