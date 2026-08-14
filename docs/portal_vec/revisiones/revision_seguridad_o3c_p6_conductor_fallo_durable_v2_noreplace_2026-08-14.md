# Revisión de seguridad O3c P6 V2: publicación sin reemplazo

Fecha: 14 de agosto de 2026.

Tarea: `O3C-P6-CONDUCTOR-FALLO-DURABLE-V2-NOREPLACE`.

Dictamen: **GO de seguridad, P0=0, P1=0, P2=0**.

El GO se limita a la corrección defensiva del publicador de evidencia de
fallo. No acredita estabilidad O3a/O3c, no revoca `CAP_NORMAL_021`, los dos
`NO-GO` de O3a V3 ni el P1 C21 histórico, y no autoriza O4, integración,
publicación remota, CI, despliegue, producción o cambio de métricas.

## Identidad, independencia y write-set

El candidato revisado es exactamente
`9391243f98d232a8eb78ed20ba5306e31b864bad`, con padre único
`55d6c9420e1adc3c37499b7b236ac0cfc101d6d2` y árbol
`6aa08b678c1f4ee268b57ba2872c4586fc04085e`. El rango contiene un commit,
cero merges y dos rutas:

```text
A docs/portal_vec/enmienda_o3c_p6_conductor_fallo_durable_v2_noreplace_2026-08-14.md
M tools/o3c_p6_conductor/fallo_durable.sh
```

El delta es `+159/-10`. Los modos son `100644` y `100755`. El conductor
`tools/o3c_p6_conductor/conductor.sh` permanece byte a byte en 182 líneas y
SHA-256
`6f777f59d4e157b2c4eabb13f66789cfe11b9c01f646e9512af9af388b715538`
tanto en padre como en candidato.

| Ruta candidata | Líneas | Bytes | SHA-256 |
| --- | ---: | ---: | --- |
| `docs/portal_vec/enmienda_o3c_p6_conductor_fallo_durable_v2_noreplace_2026-08-14.md` | 113 | 5.437 | `abf76b1f72424844571791fbb82d65ea892eeb872c5fd019c569663a3abd316d` |
| `tools/o3c_p6_conductor/fallo_durable.sh` | 130 | 5.368 | `b8f91102a2e98ce1e2e79ed73bfa9bd48c5d1f8ca002271dc5fc2461512bf174` |

El único write-set revisor es esta acta. No se modificaron el productor, el
conductor, las autoridades, evidencias históricas, estado transversal ni
métricas. La rama productora y la revisora estaban limpias antes del acta.

## Lectura de autoridad

Antes de editar se releyeron completos `AGENTS.md`, relevo de sesión, mapa,
tablero, relevo de contratación temporal, expediente RRHH, hoja de ruta y
matriz normativa. También se releyeron las decisiones completas O3a, O3b y
O3c, sus enmiendas y ledgers aplicables, la enmienda V2 y las dos actas V1.

La revisión de seguridad V1
`c0c6242a22a0e3b8aad074bac5f4472c96b6f6e3` se conserva como antecedente
`NO-GO P0=0/P1=1/P2=0`: el `mv` anterior podía tratar una entrada concurrente
como contenedor y declarar éxito sin publicar la ruta exacta. El GO funcional
V1 `e943cdbb095322fb360e449df437b61f5330bd89` no compensaba ese hallazgo.

## Cierre del P1 V1

La corrección satisface conjuntamente el criterio defensivo:

- `renombrar_sin_reemplazo` acepta únicamente un origen que sea directorio
  real, no enlace, y fija antes del cambio su identidad `dispositivo:inode`
  (`fallo_durable.sh:16-17`);
- `mv -n -T -- origen destino` trata el destino como la entrada exacta y
  solicita no sustituir una entrada existente (`:18`); en el entorno sellado
  se usa GNU coreutils 9.7 sobre Linux;
- como `mv -n` puede rechazar una colisión con estado cero, el retorno del
  comando no se usa como prueba única: origen debe desaparecer, destino debe
  ser un directorio real no simbólico y ambas huellas deben coincidir
  (`:19-21`);
- antes de crear el staging se rechazan origen o padre simbólicos y también un
  destino existente o incluso un enlace colgante (`:29-36`);
- `mktemp -d` crea el staging dentro del mismo padre y `umask 077` precede a
  toda creación (`:38-40`), por lo que el cambio de nombre no cruza
  filesystem;
- ante colisión, anidamiento, no movimiento o identidad distinta,
  `renombrar_sin_reemplazo` devuelve dos. `temporal_publicacion` sigue armado
  y el trap retira exclusivamente el staging privado (`:7-11`, `:64-65`);
- solo una publicación que conserva el mismo inode desarma la limpieza. El
  destino preexistente, su inode y su contenido no se alteran y el origen de
  la primitiva de rename se conserva ante colisión.

La autoprueba comprobó el éxito exacto, el rechazo de una segunda publicación,
la preservación de un directorio guardia y su inode, el rechazo de enlace
simbólico y la ausencia de contenido desviado. Terminó `GO`.

La revisión posterior al filtro se limitó a análisis estático y evidencia ya
sellada en temporales propios. No se ejecutó ni reformuló una prueba
concurrente bloqueada. Esta limitación no convierte una muestra dinámica en
prueba de ausencia: el dictamen se apoya en la semántica no reemplazante, la
postcondición de identidad y las colisiones deterministas existentes.

## Checksums, permisos y fallo cerrado

El manifiesto negativo se construye desde dentro del staging con nombres
relativos, orden byte a byte y transporte NUL (`:54-61`). No interpola la ruta
temporal en `sed`. Se verifica antes de publicar (`:62`) e incluye los raw
`fallo.stdout` y `fallo.stderr`, `fallo.tsv`, `resumen.txt` y los artefactos
parciales del conductor.

`set -euo pipefail` y el `return 2` del helper hacen que un fallo de copia,
checksum, rename o postcondición sea siempre no cero. El conductor invoca el
publicador antes de imprimir su diagnóstico y retornar uno; un fallo del
publicador tampoco puede alcanzar el resumen verde. El helper no interpreta
los raw, no abre red, no toca Git y no cambia oráculos, plazos, cardinalidad o
inventarios O3c.

Los paquetes sintéticos sellados ordinario y BF son directorios `0700` con
todos sus ficheros `0600`, incluidos los dos raw. Sus manifiestos relativos se
validaron completos:

| Paquete | Resultado | SHA-256 `SHA256SUMS` |
| --- | --- | --- |
| `/srv/fabrica/revisiones/o3c-p6-fallo-durable-v2-integracion-caso-20260814/destinos/r1` | `NO-GO`, `C01_ENTRADA`, estado 1, E/S 0, inventarios `6/6,0/0,0/0,0/0,0/0` | `0bf790fde24608d1988b8ad844bd3071f51c1c9f2316bb0b847a44b37a9a7104` |
| `/srv/fabrica/revisiones/o3c-p6-fallo-durable-v2-integracion-bf-20260814/destinos/r1` | `NO-GO`, `C01_BF_AUTO`, estado 1, E/S 0, inventarios `5/5,0/0,0/0,0/0,0/0` | `fce51de9c3f4007f60c726173a2f5369f56695baa1b51f85b10e89a7862c3834` |

Ambos contextos fijan el helper exacto V2. Los conductores sintéticos tienen
una única inyección explícita para forzar, respectivamente, la primera rama
ordinaria y BF; sus hashes distintos quedan registrados en `contexto.tsv` y
no se confunden con el conductor candidato byte-inmóvil.

## Mutantes y evidencia sellada

Se verificó el paquete existente
`/srv/fabrica/revisiones/o3c-p6-fallo-durable-v2-noreplace-mutantes-20260814`.
Su `SHA256SUMS` completo es válido; la huella del manifiesto es
`6e437958b012f9b6f4025ab3010e21f16ddeac80ce5f22d1fbdc20f7cd2e345e`
y la de `resultados.tsv` es
`318f6e2ea908c90221df61829e3f8155491e91a02ac96103b5a84e5ef781d9d5`.

La base terminó cero. Los cinco mutantes exigidos quedaron muertos:
sustitución, anidamiento, confianza exclusiva en el retorno de `mv`, omisión
de identidad y exclusión de raw. Cuatro devolvieron uno mediante la autoprueba;
la sonda de identidad devolvió dos en base y cero únicamente en el mutante.
No se ejecutaron nuevos casos concurrentes.

## Evidencia canónica verificada sin reejecutar

No se repitió O3c. Solo se comprobó el paquete único ya sellado en
`/srv/fabrica/revisiones/evidencia-o3c-p6-fallo-durable-v2-9391243/r1`.
`sha256sum -c SHA256SUMS` validó sus siete artefactos:

```text
resumen.txt     c7b80b882938d0a7492cfc6256cd1e65d8598f1c89ea007f9a37175515356089
contexto.tsv    c78e2554651f09e0e6e23c2a71435ea3d70488b4b09f36f988eba7ad0be5b57b
casos.tsv       f5e682cdcb08899fa443c828d5b2c5bc4d1e55966669e046131b38c35f13940a
bf_directos.tsv 28ccd3cde5516ff2d72dc0ec0dc3b609592bd85a5c15c3a0278c4277de33e6f0
SHA256SUMS      865ac3ce603eb460907aa59b7bf6ea06dfa415a0f6616d1331040a9700c5ea45
```

`contexto.tsv` liga HEAD `9391243f…`, Go 1.26.5, conductor y helper exactos.
El paquete registra 244/244 `GO`, 122 normal y 122 race, seis BF 65/0/0/EOF/no
retorno, 100+100 capturas, cinco inventarios sin delta y `residuos.txt` vacío.
El directorio es `0700`. Los artefactos de éxito históricos son `0644`; no
contienen raw negativos. Esta única corrida acredita compatibilidad del
mecanismo, no estabilidad.

## Puertas reproducidas

```text
identidad/padre/árbol/1 commit/0 merges                 GO
write-set, modos, líneas, bytes y SHA-256               GO
conductor byte a byte                                   GO
bash -n conductor.sh fallo_durable.sh                   GO
ShellCheck 0.11.0, sin exclusiones                      GO
fallo_durable.sh --autoprueba                           GO
cinco mutantes sellados                                 muertos
dos paquetes sintéticos: SHA256SUMS/permisos            GO
paquete canónico: SHA256SUMS/244+6/residuos             GO, sin ejecución
git diff --check 55d6c94..9391243                      GO
Gitleaks 55d6c94..9391243, 1 commit, 7,49 KB            sin fugas
Gitleaks 5345d5d..9391243, 26 commits, 206,91 KB         sin fugas
limpieza productor/revisor antes del acta               GO
```

Comandos principales:

```bash
git show -s --format='commit=%H%nparents=%P%ntree=%T' 9391243f98d232a8eb78ed20ba5306e31b864bad
git diff --name-status 55d6c9420e1adc3c37499b7b236ac0cfc101d6d2..9391243f98d232a8eb78ed20ba5306e31b864bad
git diff --numstat 55d6c9420e1adc3c37499b7b236ac0cfc101d6d2..9391243f98d232a8eb78ed20ba5306e31b864bad
sha256sum docs/portal_vec/enmienda_o3c_p6_conductor_fallo_durable_v2_noreplace_2026-08-14.md tools/o3c_p6_conductor/{conductor,fallo_durable}.sh
bash -n tools/o3c_p6_conductor/{conductor,fallo_durable}.sh
shellcheck -x tools/o3c_p6_conductor/{conductor,fallo_durable}.sh
tools/o3c_p6_conductor/fallo_durable.sh --autoprueba
(cd /srv/fabrica/revisiones/evidencia-o3c-p6-fallo-durable-v2-9391243/r1 && sha256sum -c SHA256SUMS)
git diff --check 55d6c9420e1adc3c37499b7b236ac0cfc101d6d2..9391243f98d232a8eb78ed20ba5306e31b864bad
gitleaks git --no-banner --redact --log-opts='55d6c9420e1adc3c37499b7b236ac0cfc101d6d2..9391243f98d232a8eb78ed20ba5306e31b864bad'
gitleaks git --no-banner --redact --log-opts='5345d5d097b51ab3567983f048feabeceaf2957b..9391243f98d232a8eb78ed20ba5306e31b864bad'
```

No se ejecutaron concurrencia nueva, O3c, Go global, PostgreSQL, Docker, HTTP
ni E2E. La restricción vigente prohibió repetir o reformular la prueba
bloqueada; los demás gates son inaplicables a este delta Bash/documental.

## Dictamen y relevo

**GO de seguridad, P0=0, P1=0, P2=0.** V2 cierra el P1 V1 dentro del contrato
de filesystem defensivo: el destino se trata como entrada exacta, no se
sustituye una colisión, un retorno cero silencioso no basta y la identidad del
directorio publicado debe ser la del staging privado. Checksums relativos,
permisos, cleanup y propagación no-cero quedan cerrados.

Este dictamen requiere todavía revisión funcional independiente del mismo
SHA. Dirección conserva la integración y cualquier publicación. Continúan
abiertos `CAP_NORMAL_021`, O3a V3/C21, estabilidad, O4 y todas las fases
posteriores.
