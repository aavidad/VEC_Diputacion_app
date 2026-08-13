# Revisión funcional independiente O4C-P0-CONTRATO

Fecha: 13 de agosto de 2026.

Identificador: `O4C-P0-CONTRATO-FUNCIONAL`.

Estado: **NO-GO, P0=0, P1=1, P2=0**.

Este dictamen revisa exclusivamente el candidato documental
`64351a469df4e7bba4850ef1500d9e2c4bf378de` sobre la base exacta
`5345d5d097b51ab3567983f048feabeceaf2957b`. No revisa descendientes, no abre
código O4c y no autoriza O4A-P5, integración, publicación, producción,
despliegue ni cambio de métricas.

## Independencia, lectura y write-set

La revisión se realizó desde el worktree exclusivo
`o4c-p0-funcional-20260813`, rama
`revision/o4c-p0-funcional-20260813`, sin editar ni mover la rama productora.
Su único write-set es esta acta.

Antes de revisar se leyeron completos `AGENTS.md`, relevo de sesión,
mapa/roadmap, tablero, relevo de contratación temporal, matriz normativa,
expediente y hoja de ruta RRHH. También se contrastaron completos O1a, O3a,
O3b, O3c, O4a y O4b, sus checkpoints y la evidencia aplicable. La revisión de
O4A-P4 es separada: este acta no incorpora ni acredita sus bytes.

## Snapshot, genealogía y autoridades

La cadena lineal revisada es:

```text
5345d5d097b51ab3567983f048feabeceaf2957b
  -> 26f55ed115b18b3b71b9cc30a3c49142c8f6afd5
  -> 870805e6ad115ae35bd0bc6cb507a3b5ec2d4074
  -> 909d19fc7b64004cb1162572eccbb890cb3584a1
  -> 64351a469df4e7bba4850ef1500d9e2c4bf378de
```

El delta contra la base contiene solo dos documentos nuevos:

| Documento | Líneas | SHA-256 |
| --- | ---: | --- |
| `decision_f0_h0b_c4b2_g2o_o4c_terminalidad_limpieza_2026-08-13.md` | 552 | `924f76b92d5988875866065eaaacf5405fd7d5c9a2f3e7013ba516c8fc6cc79c` |
| `checkpoint_f0_h0b_c4b2_g2o_o4c_p0_contrato_2026-08-13.md` | 97 | `3e6f603c81dd1e00b10da9889e2160f01a235c97e73a34928ce6a13700f51156` |

Se reprodujeron los hashes citados de O4a, checkpoint O4a, O4b, checkpoint
O4b, O3c, ledger O3c y corrección O1a. Los siete coinciden byte a byte:

```text
O4a             ffe3d570b3fe51be96948f8aeb0b163e2ff071d2c2cca4fbdba852a5f9dafebc
checkpoint O4a  0ba877cd831f7957d36cab885644dd6f8e5069d80e1629669b92d282e99d1513
O4b             675d33b6f96ef441843721effd332a82242ed9257a2af2246c75ed22f2984c7f
checkpoint O4b  1335c81e8a2ba574d21bbdd86ff853c7211fa58d8274c8763797cd1380906a45
O3c             f47395d68fa3f9e39e118f81b07fde8d8792aa61d4820dfb676ff4c7216515b6
ledger O3c      a1de39d1c80492cae8b6b858a096d8f3051b346913064fe51a83dd6573dbb3b1
O1a             29c1520f6aab91d832ec6ddb41efd75df587d240b8fb1e7dc51f107df831a659
```

El checkpoint identifica correctamente como externa la cadena O4A-P4 hasta
`2b7eaf4`. No hay código, prueba, herramienta, runner, workflow, SQL, O3,
O4a/O4b, `AGENTS.md`, roadmap, handoff, ledger transversal ni métrica modificada.

## Resultado requisito por requisito

| Requisito O4c | Resultado funcional |
| --- | --- |
| Propiedad exclusiva de terminalidad, Wait, drenaje, cierre y liberación posteriores a O4a/O4b | Conforme y sin arista inversa. |
| Agregado O4A-P5 one-shot, A8, owners O4C, estados observador 2/lease 3 y sellos exactos | Contrato total y fail-closed. |
| Límite absoluto de una sola rama, monotónico, no recreado y con borde estricto | Conforme. |
| Pareja primario/reserva no recolectora antes de un único `cmd.Wait` | Conforme; reserva coteja y no sustituye. |
| Estado real del Wait y relación cerrada causa/estado | Conforme con O1a: SALIDA 0/64/65/79 y estados canónicos restantes. |
| Wait4 WNOHANG hasta `ECHILD` y luego sonda de grupo solo `ESRCH` | Conforme; PID cero, EPERM, EINTR y duda fallan cerrados. |
| Orden pidfd primario/reserva → CONTROL → TERMINAL → inventario → observador → lease | Conforme en orden y ownership. |
| TERMINAL canónico, una emisión lógica, parciales/EINTR acotados y cuarentena | Conforme con la gramática O1a y sin sustitución de causa. |
| Inventario físico final exacto | **No conforme como contrato exacto:** la cardinalidad escrita contradice la lista cerrada de recursos. |
| Resultado privado one-shot sin PID/pidfd/nonce/ticket/error libre | Conforme. |
| DAG P0→P8, write-sets acotados, pruebas/mutantes y bloqueos entre fronteras | Conforme; P1 permanece expresamente bloqueado por O4A-P5. |
| Sin señales, nuevos pidfd, fallback PID/PGID, segundo Wait, API, SQL, HTTP o red | Conforme y explícito. |

La contradicción O4a/O4b descubierta al revisar O4A-P4 no se oculta: bloquea
la ruta material que llegaría a O4A-P5. No altera por sí sola la
responsabilidad O4c aquí delimitada, que consume una custodia ya corregida y
acreditada en el futuro. El NO-GO de este documento procede del hallazgo propio
siguiente.

## Hallazgo P1-01 — inventario final con cardinalidad contradictoria

La secuencia de cierre define inequívocamente cinco FD de caso que desaparecen
del snapshot físico:

1. el handle pidfd opaco, consumido por el único `cmd.Wait`;
2. pidfd primario;
3. pidfd reserva;
4. CONTROL;
5. TERMINAL.

Esto coincide con O3a/O3c: existen exactamente tres referencias pidfd
contando el handle opaco, más CONTROL y TERMINAL. También coincide con los
pasos 2 y 5--8 de la propia secuencia O4c y con el DAG P4/P5.

Sin embargo, la línea 328 de la decisión exige que el inventario sea el
snapshot menos «los cinco recursos poseídos **y** el handle opaco». Leído
literalmente ordena seis eliminaciones y cuenta dos veces el handle. Ese
oráculo no puede coexistir con la custodia cerrada de cinco FD y permitiría
que P6 implemente o pruebe una cardinalidad distinta.

La corrección accionable es documental y acotada: sustituir esa frase por una
enumeración cerrada, por ejemplo «menos exactamente los cinco FD poseídos
`{pidfdOpaco, pidfdPrimario, pidfdReserva, CONTROL, TERMINAL}`», y hacer que
OC02, OC16 y el mutante de inventario sellen esa cardinalidad. `Cmd` y
`Process` se ponen a cero lógicamente después, pero no añaden un sexto FD al
delta físico.

Al tratarse del contrato que debe gobernar implementación, pruebas y mutantes,
la ambigüedad no se difiere a P6. Requiere un candidato documental corregido y
dos revisiones independientes sobre sus nuevos bytes.

## Puertas reproducidas

```bash
git merge-base --is-ancestor \
  5345d5d097b51ab3567983f048feabeceaf2957b \
  64351a469df4e7bba4850ef1500d9e2c4bf378de
git diff --name-status \
  5345d5d097b51ab3567983f048feabeceaf2957b..64351a469df4e7bba4850ef1500d9e2c4bf378de
wc -l docs/portal_vec/decision_f0_h0b_c4b2_g2o_o4c_terminalidad_limpieza_2026-08-13.md \
  docs/portal_vec/revisiones/checkpoint_f0_h0b_c4b2_g2o_o4c_p0_contrato_2026-08-13.md
sha256sum docs/portal_vec/decision_f0_h0b_c4b2_g2o_o4c_terminalidad_limpieza_2026-08-13.md \
  docs/portal_vec/revisiones/checkpoint_f0_h0b_c4b2_g2o_o4c_p0_contrato_2026-08-13.md
sha256sum -c <manifiesto efímero de las siete autoridades citadas>
git diff --check \
  5345d5d097b51ab3567983f048feabeceaf2957b..64351a469df4e7bba4850ef1500d9e2c4bf378de
gitleaks git --no-banner --redact \
  --log-opts='5345d5d097b51ab3567983f048feabeceaf2957b..64351a469df4e7bba4850ef1500d9e2c4bf378de'
```

Resultados: base/ancestry, cadena, dos altas exclusivas, líneas, hashes,
siete autoridades, enlaces Markdown locales y `git diff --check` verdes.
Gitleaks recorrió cuatro commits y 32,60 KB sin hallazgos. No existe paquete
Go afectado: focal normal/race, gofmt y vet son no aplicables a este corte
Markdown, como declara el propio checkpoint.

## Seguridad, privacidad y relevo

No se encontraron secretos, credenciales, datos personales, rutas privadas ni
alcance impropio. El contrato conserva capacidades privadas, denegación por
duda, causa inmutable, cuarentena y cero caso siguiente. No hubo Docker,
PostgreSQL, HTTP, E2E, push, despliegue ni cambio de porcentajes.

El relevo inequívoco es **NO-GO**. `64351a4` debe conservarse como candidato
histórico sin autoaprobación. La siguiente acción es una corrección documental
pequeña que selle los cinco FD por nombre, seguida de revisión funcional y de
seguridad independientes sobre el nuevo SHA. `O4C-P1-AUTORIDAD` no queda
desbloqueada por esta acta.
