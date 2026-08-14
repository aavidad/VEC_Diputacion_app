# Revisión funcional O3a V5 CND V2 — propagación causal C18

Fecha: 14 de agosto de 2026.

Tarea revisada: `O3A-V5-CND-ESTADOS-CAUSALES-V2-C18`.

Dictamen: **GO funcional, `P0=0`, `P1=0`, `P2=0`**, limitado a la
corrección del falso verde de la primera escritura CONTROL de C18. El error se
asigna, se propaga y alcanza el estado exterior 111 sin cambiar orden,
oráculo, plazos ni conducta productiva.

Este dictamen no acredita estabilidad O3a V5, el parche de toolchain, el corte
P2 de vulnerabilidades, O4, publicación, CI remota o producción. Una corrida
verde nueva no revoca el P1 histórico de C21.

## Independencia, lectura y write-set revisor

La revisión se realizó en el worktree exclusivo
`/srv/fabrica/revisiones/o3a-v5-cnd-estados-causales-v2-c18-funcional-20260814`,
rama
`revision/o3a-v5-cnd-estados-causales-v2-c18-funcional-20260814`, creada
directamente en el SHA objetivo. El productor permaneció limpio y no se
editó, movió, integró ni rebasó.

Antes de editar se leyeron completos `AGENTS.md`, los cuatro documentos
obligatorios de relevo/mapa/tablero, el expediente RRHH normalizado, la hoja de
ruta y la matriz normativa. También se releyeron la autoridad O3a V5, la
enmienda causal base, esta V2 y los dos NO-GO independientes que localizaron el
falso verde. Las lecturas obligatorias y autoridades no cambian entre padre y
candidato.

El único write-set revisor es esta acta. No se modificaron las cinco rutas
candidatas, código productivo, conductores, workflow, evidencias históricas,
PostgreSQL, estado transversal, porcentajes ni credenciales.

## Identidad y alcance exactos

| Propiedad | Valor reproducido |
| --- | --- |
| Candidato | `ec02febaa3080cb8e7894340075beb850ea54348` |
| Padre directo | `a997acbb9a51c02168f1648225c0e69c84e648d4` |
| Árbol candidato | `e7c5dfaaa96b76ed9d25733456d56c0261ac98db` |
| Distancia | un commit lineal |
| Delta total | cinco rutas, `+99/-4`, sin cambio de modo |
| Limpieza productora | limpia en el SHA exacto |

El write-set candidato es:

| Ruta | Delta | Líneas | SHA-256 final |
| --- | ---: | ---: | --- |
| `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_pruebas_adversas.go` | `+4/-1` | 744 | `5ec6be1ccb917ec2908bfa75d2d947ae7a107c39044239166cc10e1067d7677e` |
| `docs/portal_vec/enmienda_o3a_v5_cnd_estados_causales_v2_c18_2026-08-14.md` | `+92/-0` | 92 | `df28ccf16dc08517a76d62a4def44a8f7ce69f4bbd8e9bdcf487857b7d1a6855` |
| `tools/o3a_v5_conductor/fuentes_v5.tsv` | `+1/-1` | 11 | `069ede20e8c6d6f2a7ab07baed330bd7a879c2ab7c5c763e93a36e1917850758` |
| `tools/o3b_p7_conductor/fuentes.tsv` | `+1/-1` | 23 | `25cd4ca6b8c1d888b6003b58392921c28a275dc8be59b8f0e54e31c04b37cfbd` |
| `tools/o3c_p6_conductor/fuentes.tsv` | `+1/-1` | 33 | `1d898b93a9b7f729d6f14334168b36056be2257051b5b2132e2a5c0dd83ce452` |

Los cinco objetos son blobs `100644`. G7b conserva
`//go:build ignore && linux && amd64`: es test-only. G7a queda byte a byte en
742 líneas y SHA-256
`4631384a9ecd85b09386aed312f16a52c56234795929219d2770b25234f70ec7`.
G7b queda en 744 líneas, por debajo de la parada local 750 y del tope duro 800.
Los tres ledgers fijan exactamente ambas huellas; cada conductor validó su
ledger completo. No cambia ningún conductor ni fuente productivo.

## Auditoría semántica de C18

La comparación byte a byte con el padre confirma una única transformación
conductual test-only en `probarVueltaTardiaExternaO3aM38`:

```go
if err = prepararFixtureO3aM38(f); err != nil {
	return err
}
if err = escribirControlPruebaO3aM38(f, "V1|CONTROL|CANCELAR|"); err != nil {
	return err
}
```

La secuencia observable queda cerrada:

1. se crea el fixture;
2. se registra el mismo `defer` que une con `errors.Join` el error principal y
   el de limpieza;
3. la preparación se asigna a `err` y corta si falla;
4. solo tras preparación verde se ejecuta la primera escritura CONTROL, cuyo
   resultado también se asigna a `err` y corta si falla;
5. solo tras ambas puertas verdes continúan aplazamiento, segunda escritura,
   retirada y comprobación de la causa `CANCELADO`.

Así, una escritura fallida ya no puede hacer verdadera una condición mientras
`err` permanece `nil`. Si la limpieza termina verde, se conserva el error de
escritura; si también falla, el `defer` conserva ambos. El llamador mantiene el
orden barrera parcial 109 → barrera de plazo 110 → vuelta tardía 111 y traduce
cualquier error de esta función a `estadoVueltaTardiaExternoO3aM38`, valor
111. El conductor sigue esperando exclusivamente estado 0 para C18: 111 no se
reclasifica como éxito.

El barrido de los diez fuentes y de todos los usos de
`escribirControlPruebaO3aM38` no encontró otro gemelo que descarte el error de
una segunda operación y devuelva el error anterior. La escritura posterior de
la propia vuelta tardía ya converge a `errInvarianteO3aM38`, no a `nil`. No se
añaden retry, sleep, tolerancia, impresión, señal, FD, proceso ni estado.
Permanecen byte a byte los cien procesos C21, cardinalidades, plazos,
watchdogs, estados 99..113 y oráculos.

## Mutante focal

Se verificó la evidencia efímera preservada fuera de Git en
`/srv/fabrica/revisiones/o3a-v5-cnd-v2-mutante-c18-20260814`. El fuente mutado
difiere del candidato en una sola línea: sustituye exclusivamente la primera
escritura CONTROL de C18 por un error nuevo asignado a `err`.

| Artefacto | SHA-256 |
| --- | --- |
| G7b mutado | `fc6f0eaae8fbe1a97202a594236e3b47f7b58aa46b8d98b83443d4b88082ea54` |
| Binario normal | `24cd4d3e4488ac6b4058e9594b43b19cd64deb41fd47fd14143811e752b2cd42` |
| Binario race | `6ab309a135f13ec43037e87e67d3402ee97f5b7092d51bd81826238bee1398ad` |

La reproducción funcional ejecutó una sola vez cada binario para
`C18_BORDES`: normal y race devolvieron exactamente 111. Los cuatro artefactos
sellados de stdout/stderr son vacíos, SHA-256
`e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`.
El mutante mata por tanto el falso verde sin cambiar el oráculo esperado 0.

## Gates y evidencia

La compilación focal propia, serializada con
`flock --close /srv/fabrica/proyectos/VEC_Diputacion_app/.sec-toolchain-review-gates.lock`,
terminó verde sobre los diez fuentes: `gofmt`, `go vet`, build normal
`CGO_ENABLED=0` y build race real `CGO_ENABLED=1 -race`.

Se validaron íntegramente con `sha256sum -c` las evidencias productoras:

| Gate | Resultado y huellas principales |
| --- | --- |
| O3a V5, Go 1.26.6 | GO 14/14, 74 casos, FD 4→4, residuos cero; resumen `a700192ecc5c28b6ba3d27b16d5e33f59664f8aa68f90289b139a43b1ed85ecb`, manifiesto `3601ad33db16cdcb2df42f91dd1c0e1c29e602e22c07c72570f00abb4a9f8e78`, C21 race `ee530c3fa1d78a2467529bcfec729ad12edee28c28c640f4be055f7ed0fb2c91`, sumas `a629d7a39d5e99963ae6d375db28de78da6c328dde14b1bdf1c03eb6ebee6929` |
| O3b P7, Go 1.26.5 | GO, 234 casos, 100+100 capturas, seis O17, residuos cero; resumen `b1bbf347c322cceb261c8c31b5f65b2db7cac79e4a3698a3b32be80d6301913c`, casos `83d0366f4a5253cf6638d1cada04df2a44476063d042254a4192c01cd543951a`, O17 `17abff3c1768fc4b937aecbb1a8aa9de5954f723b93c9e8587b886af830a1172`, sumas `6676c3b89fbfd92b9471cb7278896be2ab3b40182ee63dbc3ba33dc417b6e9f7` |
| O3c P6, Go 1.26.5 | GO, 244 casos, 100+100 capturas, seis BF, residuos cero; resumen `ea425fb334ab94021f12d862c1ab7f791b3269e3c3f42cd98643ef31c135da48`, casos `45cb503ce661eb6e31a1fa1dc44bec75a8e60b9ea8f98d5f7e88d7d913c51159`, BF `cb88a01c96bf91857bfb63be0cbdecb3037596829e6fdfd6610e272f7517177c`, sumas `2b34678a2d1d322c8d15d309f8f0c907ba2fca5ab6aea9f9793bcb69d70af186` |

La reproducción contractual funcional propia se ejecutó una vez, sin retries,
sobre el clon exacto, limpio y propiedad de `orquesta`. Terminó GO 14/14, 74
casos, Go 1.26.6, FD 4→4 y residuos cero. C21 completó 100/100 tanto normal
como race. La evidencia está en
`/srv/fabrica/revisiones/evidencia-funcional-o3a-v5-cnd-v2-c18-ec02feb-r1`
y valida íntegramente su `SHA256SUMS`:

| Artefacto propio | SHA-256 |
| --- | --- |
| `resumen.txt` | `890c5431f5ee1d97d01b289cf78b445585448ad5f922a54494220bbb00137e62` |
| `casos.ndjson` | `2a3783181edd95282f9794ee8e9f632907a4b3be5f3703e86015447a0cea0bd4` |
| `manifiesto.tsv` | `2efd2f5afcf5028639eead3e572672b7a5ead544cc046c2bada731d61f8041de` |
| `c15_c21_race_c21_indices.tsv` | `ee530c3fa1d78a2467529bcfec729ad12edee28c28c640f4be055f7ed0fb2c91` |
| `SHA256SUMS` | `e87c82416544b55d3278791ef1d5ec17405c234119e9611b2e8c0a866db87f7b` |

Gitleaks v8.30.0 recorrió el delta `a997acb..ec02feb` —un commit, 4,81 KB—
y el acumulado `5345d5d..ec02feb` —23 commits, 183,09 KB— sin fugas.
`git diff --check` quedó verde. La enmienda no contiene enlaces Markdown
locales. Productor, clon contractual y rama revisora quedaron limpios.

No se repitieron O3b ni O3c: sus gates históricos exactos acababan de
ejecutarse una vez, sus checksums fueron reproducidos y el cambio compartido
es solo la huella de G7b. Tampoco se repitieron Docker, PostgreSQL, E2E o
calidad global, que no son proporcionales a un delta test-only/documental sin
código productivo.

## Dictamen y relevo

El candidato exacto recibe **GO funcional, `P0=0`, `P1=0`, `P2=0`** solo
para la propagación causal C18. La corrección cumple el criterio único y queda
lista para decisión de dirección tras la revisión de seguridad independiente.

El P1 histórico de estabilidad O3a V5 permanece abierto: el primer estado
negativo de una futura corrida canónica deberá asignar el siguiente parche,
sin mayoría ni retry. Este acta tampoco acredita el toolchain ni el candidato
P2 de vulnerabilidades. No autoriza integración, push, despliegue, producción,
credenciales, seguridad, estado transversal o cambio de métricas.

## Comandos principales

```text
git rev-parse HEAD HEAD^ 'HEAD^{tree}'
git merge-base --is-ancestor a997acbb9a51c02168f1648225c0e69c84e648d4 ec02febaa3080cb8e7894340075beb850ea54348
git diff --name-status HEAD^..HEAD
git diff --numstat HEAD^..HEAD
git diff --check HEAD^..HEAD
git ls-tree HEAD -- RUTAS_CANDIDATAS
wc -l -c RUTAS_CANDIDATAS
sha256sum RUTAS_CANDIDATAS
gofmt -d FUENTES_O3A
go vet FUENTES_O3A
CGO_ENABLED=0 go build -trimpath -o /dev/null FUENTES_O3A
CGO_ENABLED=1 go build -race -trimpath -o /dev/null FUENTES_O3A
BINARIO_MUTANTE --autoprueba-o3a-caso C18_BORDES
flock --close LOCK runuser -u orquesta -- tools/o3a_v5_conductor/conductor.sh CLON EVIDENCIA_NUEVA
(cd EVIDENCIA && sha256sum -c SHA256SUMS)
go run github.com/zricethezav/gitleaks/v8@v8.30.0 git . --no-banner --redact --no-color --log-opts=RANGO
```
