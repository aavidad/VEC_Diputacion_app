# Revisión de seguridad O3C-P6-CONDUCTOR-FALLO-DURABLE-P0

Fecha de revisión: 14 de agosto de 2026

Rol: revisión de seguridad independiente

Dictamen: **NO-GO**

Severidad: **P0=0, P1=1, P2=0**

## Objeto y límites

Se revisó exclusivamente el candidato
`55d6c9420e1adc3c37499b7b236ac0cfc101d6d2`, cuyo padre es
`fea52f3ddf796991c93c85cae992ca695db1ae63` y cuyo árbol es
`b9857dc7d82c0a52bffa575da5957d103558408f`.

La revisión comprueba que el conductor O3c conserve de forma privada,
íntegra y cerrada el primer fallo ordinario o de borde fatal. No acredita la
estabilidad de O3a/O3c, no revoca los dos `NO-GO` de O3A-V5-CND-V3, no
compensa `CAP_NORMAL_021`, no acredita código productivo, publicación, CI ni
producción y no modifica porcentajes o estado transversal.

El único fichero añadido por el revisor es esta acta. El candidato modificó
exactamente:

| Ruta | Operación | Líneas | SHA-256 |
| --- | --- | ---: | --- |
| `docs/portal_vec/enmienda_o3c_p6_conductor_fallo_durable_2026-08-14.md` | A | 120 | `6b795bf0fe6b433414dfa8e684a77e7ea8ca9e934235d36b277ab62a0f3e03ea` |
| `tools/o3c_p6_conductor/conductor.sh` | M | 182 | `6f777f59d4e157b2c4eabb13f66789cfe11b9c01f646e9512af9af388b715538` |
| `tools/o3c_p6_conductor/fallo_durable.sh` | A | 94 | `8827cd2953aaf0787897062fd1652ef716eaef969717f803715df85573ba7641` |

El delta es un único commit lineal, tres rutas, 234 inserciones y tres
borrados. Ambos ejecutables conservan modo `100755`.

## Autoridad leída

Antes de editar se leyeron completos `AGENTS.md`, el relevo de sesión, mapa,
tablero y relevo de contratación temporal, expediente RRHH, roadmap, matriz
normativa y las decisiones vigentes O3a, O3a-V5, estados causales, V2-C18,
V3-C21, O3b y O3c. También se leyeron la evidencia final O3b, las revisiones
funcional y de seguridad O3c, la enmienda candidata y las dos actas `NO-GO`
de O3A-V5-CND-V3.

De esas autoridades se conservaron como invariantes de esta revisión:

- denegación predeterminada y privilegio mínimo;
- evidencia fuera de Git, privada, íntegra y ligada al HEAD y toolchain;
- ningún fallo del propio publicador puede convertirse en `GO`;
- el primer fallo ordinario o BF debe quedar en el destino exacto, sin
  sobrescritura, redirección ni publicación parcial;
- el éxito canónico anterior no debe cambiar de significado;
- un verde posterior no compensa el rojo durable histórico ni acredita
  estabilidad.

## Hallazgo bloqueante

### P1 — carrera entre comprobación y `mv` permite redirigir el paquete

El helper comprueba `! -e $destino` en
`tools/o3c_p6_conductor/fallo_durable.sh:18`, crea un temporal hermano en la
línea 27 y solo después publica con `mv -- "$temporal" "$destino"` en la
línea 48. No existe una primitiva atómica de publicación exacta y
`no-replace` que una la ausencia del destino con el cambio de nombre.

Una sonda sincronizada creó un enlace simbólico en el destino después de que
apareciera el temporal —por tanto, después del precheck— y antes del `mv`. El
helper devolvió estado cero, dejó el enlace intacto y movió el paquete dentro
del directorio señalado por el enlace:

- raíz durable:
  `/srv/fabrica/revisiones/evidencia-seguridad-o3c-p6-fallo-durable-55d6c94-race-symlink-r1`;
- destino solicitado: `destino`, todavía enlace simbólico, modo `777`;
- paquete real:
  `redirect/.destino.fallo.pC0EdS`, directorio modo `700`;
- SHA-256 de `SHA256SUMS` del paquete:
  `6b9566fc183a23bdb2a6ae28aa1835768fd6b62631a610fd5bc96c115f4c6439`;
- SHA-256 de `resumen.txt`:
  `5a14e27d099a49ddd527a1656ca050edb78fcd13a1da5cedad5405d398f49183`;
- stdout y stderr del publicador: cero bytes, SHA-256
  `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`.

El paquete interno mantuvo permisos privados e integridad, pero quedó fuera
de la ruta exacta solicitada y el publicador informó éxito. Esto rompe
simultáneamente la ausencia de path traversal, la atomicidad contractual del
destino y el fallo cerrado del publicador. El hallazgo es P1 porque bloquea el
único criterio de cierre de esta minitarea documental/test-only, aunque no
demuestra por sí solo una escalada o exposición en producción.

La presencia previa de un enlace colgante sí fue rechazada por `mv`; no se
clasifica como defecto. El problema demostrado requiere la intercalación
entre el precheck y la publicación.

La corrección debe usar una publicación atómica, exacta y sin reemplazo
(por ejemplo, una envoltura acotada de `renameat2(RENAME_NOREPLACE)` o una
primitiva equivalente), verificar el tipo/identidad esperados y añadir un
mutante que inserte enlace o directorio durante la ventana. No corresponde a
esta rama de revisión corregir el candidato.

## Controles que sí resultaron conformes

- `set -euo pipefail` y la invocación directa del helper hacen que cualquier
  rechazo o fallo del publicador termine el conductor con estado no cero; no
  existe una ruta del fallo hacia `resultado=GO`.
- `umask 077`, el temporal creado con `mktemp -d` y el trap mantienen los
  paquetes negativos en directorio `0700` y ficheros `0600`; los datos raw
  permanecen fuera de Git y no se interpretan como órdenes.
- El trap elimina solamente rutas temporales creadas por el propio proceso.
- Tanto el primer fallo de caso ordinario como el primer fallo BF pasan por
  el mismo publicador, conservan stdout/stderr íntegros, metadatos, inventarios
  y contexto, y terminan en `NO-GO`.
- `contexto.tsv` liga HEAD, Go, conductor, helper, matriz, ledger de fuentes y
  target; `SHA256SUMS` cubre todos los ficheros regulares del paquete.
- La autoprueba acredita copia byte a byte, ausencia de `GO`, suma válida y
  rechazo secuencial de sobrescritura. Cuatro mutantes previstos murieron.
- La ruta canónica verde conservó 244 casos (122 normal y 122 race), seis BF,
  100+100 capturas, inventarios y residuos cero. Esto acredita compatibilidad
  del camino de éxito en esa ejecución, no estabilidad.

## Evidencia reproducida

Identidad y contenido:

```text
HEAD   55d6c9420e1adc3c37499b7b236ac0cfc101d6d2
parent fea52f3ddf796991c93c85cae992ca695db1ae63
tree   b9857dc7d82c0a52bffa575da5957d103558408f
commits/merges 1/0
diff   3 rutas, +234/-3
```

Puertas focales:

```text
bash -n conductor.sh fallo_durable.sh                    GO
shellcheck -x conductor.sh fallo_durable.sh              GO
fallo_durable.sh --autoprueba                            GO
4/4 mutantes del helper                                  muertos
git diff --check HEAD^ HEAD                              GO
Gitleaks fea52f3..55d6c94 (1 commit, 10.21 KB)           sin fugas
Gitleaks 5345d5d..55d6c94 (25 commits, 199.42 KB)        sin fugas
```

La evidencia canónica quedó en
`/srv/fabrica/revisiones/evidencia-o3c-p6-fallo-durable-55d6c94/r1`.
`sha256sum -c SHA256SUMS` validó siete ficheros. Sus hashes de control son:

```text
resumen.txt  26a15b3be925fefd9d0bb119bed76f67a39281075e1d0cb8a92c168268d8def0
contexto.tsv 79b304751c182175245f9d4b9665347da8dd3665bb3ac3c02f52f19f5f5d0cae
SHA256SUMS   54535fafa63bae9a1b95b8d0ad9cb8e7b66466ff2ccf41d76b0e8e01
```

Los paquetes sintéticos negativos ordinario y BF validaron sus sumas y
terminaron en `NO-GO`, con E/S cero y permisos privados:

```text
ordinario SHA256SUMS 71f12f546f14227bf73c5df6d0cdc134f748c8468b589dbcd707d7a6321f2df2
BF         SHA256SUMS fd478542890593e921f461c2003783348618947e90e486152e06c89a3e811492
```

## Comandos de reproducción

Se ejecutaron, una sola vez cuando correspondía, comandos equivalentes a:

```bash
git rev-parse HEAD HEAD^ HEAD^{tree}
git rev-list --count HEAD^..HEAD
git rev-list --merges --count HEAD^..HEAD
git diff --stat HEAD^ HEAD
git diff --check HEAD^ HEAD
wc -l docs/portal_vec/enmienda_o3c_p6_conductor_fallo_durable_2026-08-14.md tools/o3c_p6_conductor/conductor.sh tools/o3c_p6_conductor/fallo_durable.sh
sha256sum docs/portal_vec/enmienda_o3c_p6_conductor_fallo_durable_2026-08-14.md tools/o3c_p6_conductor/conductor.sh tools/o3c_p6_conductor/fallo_durable.sh
bash -n tools/o3c_p6_conductor/conductor.sh tools/o3c_p6_conductor/fallo_durable.sh
shellcheck -x tools/o3c_p6_conductor/conductor.sh tools/o3c_p6_conductor/fallo_durable.sh
tools/o3c_p6_conductor/fallo_durable.sh --autoprueba
(cd /srv/fabrica/revisiones/evidencia-o3c-p6-fallo-durable-55d6c94/r1 && sha256sum -c SHA256SUMS)
gitleaks git --no-banner --redact --log-opts='fea52f3ddf796991c93c85cae992ca695db1ae63..55d6c9420e1adc3c37499b7b236ac0cfc101d6d2'
gitleaks git --no-banner --redact --log-opts='5345d5d097b51ab3567983f048feabeceaf2957b..55d6c9420e1adc3c37499b7b236ac0cfc101d6d2'
```

La sonda de carrera usó únicamente directorios bajo
`/srv/fabrica/revisiones`, esperó la creación del temporal hermano, insertó
el symlink controlado y esperó la terminación del único proceso publicador.
No modificó Git ni el worktree productor.

## Dictamen y relevo

**NO-GO, P0=0, P1=1, P2=0.** El candidato mejora de manera verificable la
durabilidad y privacidad del fallo, pero no satisface la publicación atómica
en el destino exacto ni falla cerrado ante una sustitución concurrente de esa
ruta. Debe corregirse el helper, matar el mutante de intercalación y recibir
de nuevo revisión funcional y de seguridad independientes.

Continúan abiertos, fuera de este write-set, `CAP_NORMAL_021`, los dos
dictámenes `NO-GO` de O3A-V5-CND-V3 y la acreditación de estabilidad. La
corrida canónica verde de este candidato no los revoca ni los compensa.
