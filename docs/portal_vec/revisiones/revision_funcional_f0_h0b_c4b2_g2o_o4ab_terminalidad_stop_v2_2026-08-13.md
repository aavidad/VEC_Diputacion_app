# Revisión funcional O4AB-P0-ENMIENDA-TERMINALIDAD-STOP V2

Fecha: 13 de agosto de 2026.

Identificador: `O4AB-P0-ENMIENDA-TERMINALIDAD-STOP-V2-FUNCIONAL`.

Estado: **NO-GO, P0=0, P1=1, P2=0**.

Este dictamen independiente revisa exclusivamente el candidato
`9de3ba320dfa4beede58e5a1d34aa02a3459f073`, hijo de la V1 rechazada
`a89a3228554f53b32f5d81fc8b0438835f35b0f6`, con base
`5345d5d097b51ab3567983f048feabeceaf2957b` y árbol
`d44580e843d89a9311d9cbd322f3499f5f162b54`. No revisa ni acredita código,
`2b7eaf4`, ningún otro descendiente, O4B-P1, O4A-P5, O4c, O5/O6,
integración, publicación, producción, despliegue ni métricas.

## Independencia, lectura y write-set

La revisión se realizó desde el worktree exclusivo
`o4ab-p0-v2-limite-preflight-funcional-20260813`, rama
`revision/o4ab-p0-v2-limite-preflight-funcional-20260813`, creada en el SHA
objetivo. No se editó ni movió la rama productora. El único write-set de
revisión es esta acta.

Antes de editar se leyó `AGENTS.md` completo y se revalidaron las lecturas
obligatorias: relevo de sesión, mapa/roadmap, tablero, relevo de contratación
temporal, matriz normativa, expediente y hoja de ruta RRHH. Se releyeron
íntegros O4a, O4b, la enmienda V2, su checkpoint y los cuatro NO-GO
aplicables: funcional y seguridad de `2b7eaf4`, y funcional y seguridad de la
V1 `a89a322`. También se contrastaron las autoridades O3a/O3b/O3c que la
enmienda conserva para lease, identidad, primitivas y custodia.

## Identidad y huellas reproducidas

El candidato V2 es un único commit sobre V1. Su delta exacto modifica solo:

| Documento | Delta V2 | Líneas finales | SHA-256 final |
| --- | ---: | ---: | --- |
| `docs/portal_vec/enmienda_f0_h0b_c4b2_g2o_o4ab_terminalidad_stop_2026-08-13.md` | `+47/-17` | 264 | `f53ee8711eb64d7dc999e2f67154f874de7d80bfbc2e18a02c413febd04d1e15` |
| `docs/portal_vec/revisiones/checkpoint_f0_h0b_c4b2_g2o_o4ab_terminalidad_stop_2026-08-13.md` | `+25/-11` | 115 | `b380505c14de57fe0a8b1355212b55fd028971d12fb971000214b96da4ca89fe` |

El total V2 es `+72/-28`; el rango acumulado desde la base contiene solo esas
dos altas, `+379/-0`. O4a conserva 535 líneas y SHA-256
`ffe3d570b3fe51be96948f8aeb0b163e2ff071d2c2cca4fbdba852a5f9dafebc`;
O4b conserva 443 y
`675d33b6f96ef441843721effd332a82242ed9257a2af2246c75ed22f2984c7f`.

Las cuatro actas históricas se recuperaron desde sus commits exactos:

| Revisión | Padre | Líneas | SHA-256 del acta |
| --- | --- | ---: | --- |
| funcional O4A-P4 `e20a5fe597d9c172d59b99e75878f8e99e196c76` | `2b7eaf498f8f68a90b66c004f166a62f070b5064` | 176 | `f72faa38f2c4543b46b4d8318a19a87149b84d15e4a3f7044317baa4df96de48` |
| seguridad O4A-P4 `a62ee60f70315a1fc0d290290b681c9c3f35227b` | `2b7eaf498f8f68a90b66c004f166a62f070b5064` | 147 | `f01503425807851a9efb3a6542dd110054e387d9760626b3b921538a7f80135e` |
| funcional O4AB V1 `cdcedc44b31f0f997139f0e9e1210f29d1eb08ca` | `a89a3228554f53b32f5d81fc8b0438835f35b0f6` | 160 | `76ee47785d27a01f61c07e750e021a88abee701a713b5bd1516cb8f97b7466cf` |
| seguridad O4AB V1 `e7e06423941807a41c40946e5a4af12e1b30bb11` | `a89a3228554f53b32f5d81fc8b0438835f35b0f6` | 133 | `e968004722ebd94a86c6a1450ee46baef811067a0372ecee99f979a4d2072a72` |

La rama productora estaba limpia. No cambiaron código, pruebas, O3, O4a,
O4b, O4c, herramientas, runner, workflows, SQL, `AGENTS.md`, handoffs,
roadmap, ledger transversal, candidatos históricos ni métricas.

## Auditoría requisito por requisito

| Requisito vigente | Resultado funcional |
| --- | --- |
| Prevalencia: O4a decide causa/tiempo/etapa y O4b solo acredita y ejecuta | Conforme y acotado. |
| `TERMINAL` post-CONT antes o en igualdad de `finGracia` | Conforme: A7, drenaje cooperativo y cero señal. |
| Presencia anterior frente a presencia exacta en gracia | Conforme: la primera espera y la igualdad abre una autorización condicional. |
| Presencia anterior revalidada al llegar al borde | Conforme: no se reutiliza como prueba final. |
| Preflight terminal | Conforme: cardinalidad real 0, raws cero, A7 y cero STOP/KILL/incidente. |
| `cardinalidadMaxima=1` frente a resultado real 0/1 | Conforme y sin reutilizar el campo raw. |
| Lectura monotónica final posterior a toda evidencia física | Conforme en el orden operativo; corrige el P1 de V1. |
| Igualdad/vencimiento en la lectura final | Conforme: OBF, autorización consumida y cero resultado/STOP/efecto. |
| Lectura final verde, permiso lease y STOP como siguiente syscall | Conforme con la secuencia inmediata publicada, sin prometer el instante interno del kernel. |
| Punto causal atribuido a la presencia y la vigencia conjuntas | **No conforme: fija presencia en una lectura que solo observa tiempo.** |
| Terminalidad posterior al STOP final | Conforme: cardinalidad 1, A7 y cero KILL/incidente. |
| STOP final estable/no estable/raw error | Conserva las tres ramas exactas. |
| Asimetría de `PARADA_INICIAL` | Conforme: cardinalidad 1, terminalidad normalizada a `NO_ESTABLE` y forja `TERMINAL` fatal. |
| Raws, marcas, replay, owner, lease, duda física y fatalidad | Cerrados fuera del punto causal descrito abajo. |
| DAG y materialización | Acotados y sin aristas inversas, pero no debe materializarse este texto mientras persista el P1. |
| Wait, señal cero, fallback, parser, getter, datos y alcance transversal | Ausentes. |

## Hallazgo P1-01 — `ahoraFinal` no puede linealizar una presencia anterior

V2 corrige el defecto de deadline de V1: las líneas 125--132 ordenan primero
consolidar toda la evidencia física de no terminalidad e identidad y después
leer el reloj; igualdad o vencimiento producen OBF y cero STOP. Las líneas
134--137 mantienen además STOP como siguiente syscall tras preparar el permiso.
Ese orden conserva la vigencia temporal sin introducir otra sonda tardía.

Sin embargo, las líneas 134--140 fijan en `ahoraFinal` la «linealización
conjunta de presencia y vigencia» y solo clasifican como posterior a la
decisión la terminalidad que ocurra después de `ahoraFinal`. La lectura de
reloj no observa presencia. La no-terminalidad acreditada por los pidfd y
`/proc` tampoco es una propiedad monotónica que pueda transportarse desde la
última sonda hasta una lectura posterior.

Por tanto el propio orden exigido admite esta ejecución:

```text
t0: la última evidencia consolidada acredita GRUPO_PRESENTE
t1: ambas referencias pasan a terminalidad natural
t2: ahoraFinal < finParadaFinal
    -> el texto fija presencia y vigencia conjuntamente en t2
    -> se prepara el permiso y STOP es la siguiente syscall
```

La terminalidad de `t1` es anterior al punto que el contrato declara
autorizante, pero no existe ninguna observación en `t2` que permita afirmar
presencia. A la vez, añadir una sonda después del reloj reabriría exactamente
la ventana temporal rechazada en V1 y contradiría la secuencia de syscall
inmediata. El problema no es el orden nuevo, sino atribuir a ese orden un
punto causal único que no puede observar ambos predicados.

Se clasifica P1, no P0: el pidfd primario, identidad, owner y lease continúan
acreditados; no se demuestra señal a un proceso ajeno, fuga de autoridad o
dato. Sí queda indeterminado cuándo gana terminalidad frente a la autorización
de STOP, precisamente la arista causal que esta enmienda debe cerrar antes de
materializar O4A-P4/O4B-P3.

### Corrección accionable

Debe conservarse el orden operativo V2 y distinguir dos hitos en vez de
atribuir ambos a `ahoraFinal`:

1. la presencia se linealiza en la última evidencia física consolidada
   `observadoPresencia`;
2. la lectura posterior `ahoraFinal` solo acredita la vigencia temporal, y
   monotonía demuestra también que `observadoPresencia < finParadaFinal`;
3. el contrato debe declarar expresamente que terminalidad posterior a
   `observadoPresencia`, incluso si ocurre antes de `ahoraFinal`, queda después
   de la decisión física y será resuelta por el raw/evidencia posterior al
   STOP, sin KILL;
4. con `ahoraFinal < finParadaFinal`, solo se prepara el permiso y STOP sigue
   siendo la siguiente syscall; igualdad/vencimiento conserva OBF y cero
   resultado/efecto;
5. matriz y mutantes deben cubrir terminalidad entre la última sonda y el
   reloj final, además del cruce temporal ya añadido.

Si la intención normativa fuese prohibir STOP ante terminalidad real ocurrida
hasta `ahoraFinal`, esa propiedad no es materializable con la secuencia actual
y requeriría otra primitiva o una decisión separada; no puede darse por probada
mediante una lectura de reloj.

## Puertas reproducidas

Se ejecutaron sobre el SHA exacto, tanto para el delta V2 como para el rango
acumulado cuando correspondía:

```bash
git show -s --format='%H%n%P%n%T%n%s' 9de3ba320dfa4beede58e5a1d34aa02a3459f073
git rev-list --count a89a3228554f53b32f5d81fc8b0438835f35b0f6..9de3ba320dfa4beede58e5a1d34aa02a3459f073
git diff --name-status a89a3228554f53b32f5d81fc8b0438835f35b0f6..9de3ba320dfa4beede58e5a1d34aa02a3459f073
git diff --numstat a89a3228554f53b32f5d81fc8b0438835f35b0f6..9de3ba320dfa4beede58e5a1d34aa02a3459f073
git diff --name-status 5345d5d097b51ab3567983f048feabeceaf2957b..9de3ba320dfa4beede58e5a1d34aa02a3459f073
wc -l docs/portal_vec/enmienda_f0_h0b_c4b2_g2o_o4ab_terminalidad_stop_2026-08-13.md \
  docs/portal_vec/revisiones/checkpoint_f0_h0b_c4b2_g2o_o4ab_terminalidad_stop_2026-08-13.md
sha256sum docs/portal_vec/enmienda_f0_h0b_c4b2_g2o_o4ab_terminalidad_stop_2026-08-13.md \
  docs/portal_vec/revisiones/checkpoint_f0_h0b_c4b2_g2o_o4ab_terminalidad_stop_2026-08-13.md
git diff --check a89a3228554f53b32f5d81fc8b0438835f35b0f6..9de3ba320dfa4beede58e5a1d34aa02a3459f073
git diff --check 5345d5d097b51ab3567983f048feabeceaf2957b..9de3ba320dfa4beede58e5a1d34aa02a3459f073
gitleaks git --no-banner --redact \
  --log-opts='a89a3228554f53b32f5d81fc8b0438835f35b0f6..9de3ba320dfa4beede58e5a1d34aa02a3459f073'
gitleaks git --no-banner --redact \
  --log-opts='5345d5d097b51ab3567983f048feabeceaf2957b..9de3ba320dfa4beede58e5a1d34aa02a3459f073'
```

Resultados: commit, padre, árbol, ancestry, un commit V2, write-set, deltas,
líneas, SHA-256, autoridades, cuatro NO-GO, enlaces Markdown locales,
productor limpio, `git diff --check` y búsqueda focal de secretos verdes.
Gitleaks recorrió un commit/5,28 KB para V2 y dos commits/21,85 KB para el
rango acumulado, sin filtraciones.

No hay paquete Go afectado: normal/race, gofmt, vet, mutantes, PostgreSQL,
Docker, HTTP y E2E no aplican a dos modificaciones Markdown. No se declaran
ejecutados ni sustituyen la revisión causal.

## Seguridad, privacidad y relevo

V2 sí cierra el STOP tardío de V1 y conserva denegación predeterminada,
permisos one-shot, fatalidad sin retorno, datos opacos y cero alcance exterior.
No añade secreto, credencial, dato humano, API, red, persistencia o interfaz;
i18n y accesibilidad no aplican.

El relevo inequívoco es **NO-GO**. `9de3ba3` debe conservarse como candidato
documental histórico sin autoaprobación. La siguiente acción es corregir solo
la atribución causal entre la última evidencia de presencia y la lectura final,
manteniendo el orden V2, y obtener dos revisiones independientes nuevas sobre
los bytes corregidos. No se hizo push, deploy, cambio de credenciales, estado
transversal, porcentajes ni producción.
