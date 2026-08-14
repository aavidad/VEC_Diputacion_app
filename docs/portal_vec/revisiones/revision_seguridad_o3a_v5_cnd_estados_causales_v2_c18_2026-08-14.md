# Revisión de seguridad O3a V5 CND V2: propagación causal C18

Fecha: 14 de agosto de 2026.

Tarea revisada: `O3A-V5-CND-ESTADOS-CAUSALES-V2-C18`.

Dictamen: **GO de seguridad, `P0=0`, `P1=0`, `P2=0`**, limitado al
falso verde de la primera escritura CONTROL de C18. La corrección asigna y
propaga el error, conserva el orden y convierte el fallo en estado exterior
111 sin stdout, stderr ni otro canal observable. Este GO no acredita la
estabilidad O3a, el parche de toolchain, el corte P2 de vulnerabilidades, O4,
publicación, CI remota ni producción.

## Identidad y alcance congelado

La revisión se realizó sobre el candidato exacto
`ec02febaa3080cb8e7894340075beb850ea54348`, cuyo padre directo es
`a997acbb9a51c02168f1648225c0e69c84e648d4` y cuyo árbol es
`e7c5dfaaa96b76ed9d25733456d56c0261ac98db`. La distancia es un único
commit lineal, la rama productora
`trabajo/o3a-v5-cnd-estados-causales-v2-c18-20260814` estaba limpia y no fue
modificada. Los cinco objetos candidatos son blobs ordinarios `100644`.

| Ruta candidata | Delta | Líneas | SHA-256 |
| --- | ---: | ---: | --- |
| `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_pruebas_adversas.go` | `+4/-1` | 744 | `5ec6be1ccb917ec2908bfa75d2d947ae7a107c39044239166cc10e1067d7677e` |
| `docs/portal_vec/enmienda_o3a_v5_cnd_estados_causales_v2_c18_2026-08-14.md` | `+92/-0` | 92 | `df28ccf16dc08517a76d62a4def44a8f7ce69f4bbd8e9bdcf487857b7d1a6855` |
| `tools/o3a_v5_conductor/fuentes_v5.tsv` | `+1/-1` | 11 | `069ede20e8c6d6f2a7ab07baed330bd7a879c2ab7c5c763e93a36e1917850758` |
| `tools/o3b_p7_conductor/fuentes.tsv` | `+1/-1` | 23 | `25cd4ca6b8c1d888b6003b58392921c28a275dc8be59b8f0e54e31c04b37cfbd` |
| `tools/o3c_p6_conductor/fuentes.tsv` | `+1/-1` | 33 | `1d898b93a9b7f729d6f14334168b36056be2257051b5b2132e2a5c0dd83ce452` |

El delta total es `+99/-4`. G7b conserva `//go:build ignore && linux &&
amd64`: es test-only. G7a permanece byte a byte en 742 líneas y SHA-256
`4631384a9ecd85b09386aed312f16a52c56234795929219d2770b25234f70ec7`.
Cada uno de los tres ledgers fue contrastado contra todos sus fuentes: 10/10,
22/22 y 32/32 entradas coinciden, y su único cambio es la huella/líneas de
G7b. Los tres conductores son byte a byte idénticos al padre.

El único write-set revisor es esta acta. No se editaron el productor, código
productivo, conductores, workflows, oráculos, plazos, cardinalidad, mapa FD,
estado transversal, métricas o credenciales.

## Lectura y criterio de seguridad

Se leyeron completos `AGENTS.md`, relevo, mapa, tablero, matriz normativa,
hoja de ruta, expediente O3a, decisión/autoridad O3a V5, la enmienda base y la
V2, y los dos NO-GO independientes de `a997acb`. Los documentos obligatorios y
las autoridades son byte a byte idénticos entre el padre y este candidato.

El criterio único heredado era impedir que un fallo de la primera escritura
CONTROL de `probarVueltaTardiaExternaO3aM38` devolviera `nil`. Se auditó que el
cambio no usara el diagnóstico como oráculo productivo ni ampliara privilegio,
espera, tolerancia o superficie observable.

## Auditoría fail-closed y de privilegio

La secuencia resultante es inequívoca:

1. crea el fixture y registra inmediatamente el `defer` que une con
   `errors.Join` el error principal y el de limpieza;
2. asigna a `err` el resultado de `prepararFixtureO3aM38` y corta en fallo;
3. solo tras preparación verde asigna a `err` la primera
   `escribirControlPruebaO3aM38` y corta en fallo;
4. solo tras ambas puertas verdes conserva el flujo previo de aplazamiento,
   escritura restante, retirada y causa `CANCELADO`.

Así, un error de la escritura no puede perderse aunque la limpieza termine
verde; si la limpieza también falla, ambos errores siguen unidos. El llamador
`ejecutarCasosLinealesExternosO3aM38` traduce cualquier error de vuelta tardía
a `estadoVueltaTardiaExternoO3aM38`, cuyo valor es 111. El conductor C18 sigue
aceptando exclusivamente estado 0: 111 nunca es GO.

No cambian el orden parcial→plazo→vuelta tardía, el segundo write CONTROL, los
defer de limpieza, los estados 99..113, los códigos previos, el oráculo, los
cien procesos C21, los plazos ni el comportamiento productivo. El barrido de
los diez fuentes y la inspección de todos los usos de
`escribirControlPruebaO3aM38` no hallaron otro `return err` que descarte el
error de una segunda operación falible. Tampoco aparecen nuevos `Print`, log,
sleep, señal, `Wait`, fallback, salida o sidechannel.

## Mutante focal C18

Se verificó el mutante efímero preservado fuera de Git en
`/srv/fabrica/revisiones/o3a-v5-cnd-v2-mutante-c18-20260814`. Difiere del
candidato en una única línea: sustituye exclusivamente la primera escritura
CONTROL de la vuelta tardía por `errors.New("mutante C18: escritura inicial")`.

| Artefacto | SHA-256 |
| --- | --- |
| G7b mutado | `fc6f0eaae8fbe1a97202a594236e3b47f7b58aa46b8d98b83443d4b88082ea54` |
| Binario normal | `24cd4d3e4488ac6b4058e9594b43b19cd64deb41fd47fd14143811e752b2cd42` |
| Binario race | `6ab309a135f13ec43037e87e67d3402ee97f5b7092d51bd81826238bee1398ad` |

La reproducción independiente válida se ejecutó una vez por modo desde el
clon exacto propiedad de `orquesta`, cerrando antes todo FD mayor o igual que
3. Normal y race devolvieron exactamente 111; stdout y stderr fueron cero en
ambos y cada uno de los cuatro ficheros tiene SHA-256
`e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`.
La evidencia queda en
`/srv/fabrica/revisiones/evidencia-seguridad-o3a-v5-cnd-v2-c18-ec02feb-r2-valida`.

Una sonda previa desde el worktree revisor propiedad de root terminó antes en
estado 109 por no poder satisfacer la barrera parcial en aquel directorio. No
alcanzó la línea mutada; se conserva como invocación ambiental inválida y no se
usa ni como rojo del candidato ni como pase. No hubo reintento de una puerta
canónica roja.

## Gates y evidencia sellada

La evidencia productora O3a en
`/srv/fabrica/revisiones/evidencias-o3a-v5-cnd-ec02feb/o3a-r1` valida
íntegramente su `SHA256SUMS` y declara Go 1.26.6, GO 14/14 bloques, 74 casos,
FD 4→4 y residuos cero. Sus huellas principales son:

| Artefacto | SHA-256 |
| --- | --- |
| `resumen.txt` | `a700192ecc5c28b6ba3d27b16d5e33f59664f8aa68f90289b139a43b1ed85ecb` |
| `manifiesto.tsv` | `3601ad33db16cdcb2df42f91dd1c0e1c29e602e22c07c72570f00abb4a9f8e78` |
| `c15_c21_race_c21_indices.tsv` | `ee530c3fa1d78a2467529bcfec729ad12edee28c28c640f4be055f7ed0fb2c91` |
| `SHA256SUMS` | `a629d7a39d5e99963ae6d375db28de78da6c328dde14b1bdf1c03eb6ebee6929` |

La evidencia O3b en el hermano `o3b-r1` también valida íntegramente su
`SHA256SUMS`: Go 1.26.5, GO, 234 casos, 100+100 capturas, O17 directo 6/6 con
estado 65 y salida cero, y residuos cero. `resumen.txt` tiene SHA-256
`b1bbf347c322cceb261c8c31b5f65b2db7cac79e4a3698a3b32be80d6301913c`
y `SHA256SUMS`,
`6676c3b89fbfd92b9471cb7278896be2ab3b40182ee63dbc3ba33dc417b6e9f7`.

La evidencia O3c en `o3c-r1` valida igualmente su `SHA256SUMS`: Go 1.26.5,
GO, 244 casos, 100+100 capturas, seis BF directos con EOF/no retorno y salida
cero, y residuos cero. Sus huellas son `resumen.txt`
`ea425fb334ab94021f12d862c1ab7f791b3269e3c3f42cd98643ef31c135da48`,
`casos.tsv`
`45cb503ce661eb6e31a1fa1dc44bec75a8e60b9ea8f98d5f7e88d7d913c51159`,
`bf_directos.tsv`
`cb88a01c96bf91857bfb63be0cbdecb3037596829e6fdfd6610e272f7517177c` y
`SHA256SUMS`
`2b34678a2d1d322c8d15d309f8f0c907ba2fca5ab6aea9f9793bcb69d70af186`.

La compilación focal revisora de los diez fuentes, serializada con el candado,
pasó `gofmt`, `go vet`, build normal y build race. Los binarios efímeros
tienen respectivamente SHA-256
`790bf9496015526fda08b46872fc99b7c032f80a8fb0373fc55037f1c5c72d99`
y `564b6da68cb2d0d9401e4002781cdf5136ba4ecaf62e01d5499b301ada300b99`;
se preservan fuera de Git en
`/srv/fabrica/revisiones/evidencia-seguridad-o3a-v5-cnd-v2-c18-ec02feb-build-r1`.

Gitleaks v8.30.0 recorrió sin fugas el delta `a997acb..ec02feb` (un commit,
4,81 KB) y el rango acumulado `5345d5d..ec02feb` (23 commits, 183,09 KB).

`git diff --check` quedó verde, los enlaces locales de la enmienda son cero y
la búsqueda focal no encontró secreto, token, clave privada, DSN ni
credencial. No se generaron residuos dentro del repositorio.

## Rectificación y límites

Esta revisión no reutiliza el estado 66 de una antigua sonda C16 directa bajo
`flock`: quedó acreditado que heredaba el FD 3 del candado sin `CLOEXEC`, a
diferencia del conductor completo que cierra todos los FD ≥3. Aquella sonda
ambiental permanece retirada del razonamiento.

La corrida O3a verde de este candidato no revoca ni reduce el P1 histórico de
estabilidad sellado sobre el conductor completo de `74f2495`: una muestra
verde no compensa una roja y V2 solo mejora la fidelidad causal del siguiente
fallo. Tampoco acredita el parche Go 1.26.6 ni el candidato documental P2 de
vulnerabilidades. PostgreSQL, Docker, E2E, publicación y CI remota son N/A al
write-set test-only/documental de esta revisión. La calidad global no se
repitió: el control proporcional queda cubierto por compilación focal y los
tres conductores sellados, y no existe delta productivo que ampliar.

## Comandos principales reproducidos

```text
git rev-parse HEAD HEAD^ HEAD^{tree}
git merge-base --is-ancestor a997acbb9a51c02168f1648225c0e69c84e648d4 ec02febaa3080cb8e7894340075beb850ea54348
git diff --name-status HEAD^ HEAD
git diff --numstat HEAD^ HEAD
git diff --check HEAD^ HEAD
git ls-tree -r HEAD -- RUTAS_CANDIDATAS
wc -l RUTAS_CANDIDATAS
sha256sum RUTAS_CANDIDATAS
gofmt -d FUENTES_O3A
go vet FUENTES_O3A
CGO_ENABLED=0 go build -trimpath -o BINARIO_NORMAL FUENTES_O3A
CGO_ENABLED=1 go build -race -trimpath -o BINARIO_RACE FUENTES_O3A
BINARIO_MUTANTE --autoprueba-o3a-caso C18_BORDES
(cd EVIDENCIA && sha256sum -c SHA256SUMS)
go run github.com/zricethezav/gitleaks/v8@v8.30.0 git . --no-banner --redact --no-color --log-opts=RANGO
```

## Relevo

El candidato recibe **GO de seguridad, `P0=0`, `P1=0`, `P2=0`** únicamente
para la propagación causal C18. Requiere dictamen funcional independiente y
decisión de dirección; este revisor no integra ni autoacredita. El siguiente
propietario de estabilidad deberá usar el primer estado negativo de una nueva
ejecución canónica, sin mayoría ni retry. No se autoriza push, despliegue,
producción, credenciales ni cambio de métricas.
