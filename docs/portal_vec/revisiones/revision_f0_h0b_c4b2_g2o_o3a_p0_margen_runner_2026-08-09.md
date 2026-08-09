# Revisión F0-H0b/C4b-2/G2-O/O3a-P0: margen del runner

Fecha: 9 de agosto de 2026.

Estado: **cierre técnico material con doble GO; P0=0, P1=0, P2=0**. El
contrato documental se publicó en `afd00c3` y su CI `31289613463` terminó
5/5. El material exacto `ac988d6` superó todas las puertas y dos revisiones
independientes. No autoriza O3a, `Start`, Bash operativo, mapa FD, despliegue
ni uso en producción.

## Objeto y autoridad

Documento revisado:

`docs/portal_vec/decision_f0_h0b_c4b2_g2o_o3a_p0_margen_runner_2026-08-09.md`

Base exacta: `c1ca5aa64221ad6a1895b8be7563a0d43ff59c9e`.

Autoridades O2b:

- material `d86aea8b4ed2b9fffbf74ef04cd90f397b017a55`;
- evidencia `4b39265405957b47b909f0b9e1bc4960c38f4011`;
- publicación `c1ca5aa`;
- CI `31287803830`, cinco de cinco trabajos verdes.

O3a-P0 solo libera margen en el runner. Sus dos rutas materiales son:

1. `deploy/postgresql/autorizacion_atestada_v3/probar_fuente_corporativa_contexto_actor_v1_pg18_4.sh`;
2. `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/operaciones_runner_fuente_corporativa_contexto_actor_v1.sh`.

## Primera ronda y NO-GO

La primera versión tenía 218 líneas y SHA-256
`5a2e6c33e416daa952c8e66c4c82076cd12dbf353d7873d3edac3c5f7638c9fc`.

El revisor funcional acabó emitiendo `NO-GO`, `P0=0`, `P1=2`, `P2=0`. El
revisor de seguridad emitió `NO-GO`, `P0=0`, `P1=3`, `P2=2`. Se acreditaron:

- una huella G3 incorrecta introducida como canario documental;
- imposibilidad de superar ShellCheck con un traslado puramente literal;
- una secuencia circular que pedía GO antes de corregir el canario;
- ausencia de oráculo exacto para el inventario de funciones;
- una promesa demasiado amplia de identidad observable en Bash.

Ningún dictamen de esa versión se reutilizó.

## Corrección revisada

La versión corregida revisada íntegramente tenía 300 líneas y SHA-256
`22d33f6139cffba59af4903fadae53872de1288edfa674247d6eeecfa05e7811`.

Corrigió la huella G3, autorizó dos únicas directivas locales de ShellCheck,
fijó los inventarios antes y después de cargar D, acotó la observabilidad a
los modos gobernados y ordenó una nueva doble revisión antes del cambio de
estado.

Después de recibir ambos `GO`, se sustituyó exclusivamente el párrafo de
estado de las líneas 5–6. La decisión autorizada final conserva 300 líneas y
su SHA-256 es
`cbfefca5ab97a635c5000eaa31675ad3fbe1a38ed0dc3731f645b20e98b399df`.
Al restaurar en ella el estado de borrador se recupera exactamente
`22d33f6139cffba59af4903fadae53872de1288edfa674247d6eeecfa05e7811`;
no cambiaron alcance, ledger, write-set, puertas ni paradas.

La transformación proyectada quedó fijada así:

| Unidad | Anterior | Posterior | SHA-256 posterior |
| --- | ---: | ---: | --- |
| Runner R | 800 | 702 | `e617024a52c4a042971b026d0799816933b489ed4221e9b6147317936d18054c` |
| Auxiliar D | 164 | 264 | `681efbbd7f856eb539d1656cffed87c26f48609e65d6d6adf8265c350ae69442` |

Los bloques originales son 72 y 26 líneas, 98 en total, con huellas
`e811e624b7d0399368871de4ea64a05b1ca989eec5e8a7571755d984f583117a`,
`0a49085527dd7bd726759c10222345e8c7a30aca24652b4829ed014712c1a41a`
y concatenada
`7dd728116bf29f2cdba860a8ab7a74d0b406d99026dbe649297187f27176cafd`.

El binario Go debe permanecer exactamente en
`6153f03a93c0a2618fdaf922443004244aa3bec7cbe9074466b22935c693edd0`.

## Reproducción y dictámenes finales

Ambos revisores reconstruyeron la proyección desde `c1ca5aa` y obtuvieron los
mismos conteos y huellas R/D. `bash -n` y ShellCheck 0.11.0 quedaron verdes.

La carga directa y la carga no acreditada de D terminaron en 64 sin definir su
API. La carga acreditada consumió el marcador y añadió exactamente diecinueve
funciones, sin colisiones ni uso prematuro de las ocho trasladadas. El
inventario final coincidió con la base.

La reconstrucción inversa retiró las dos directivas, devolvió los dos bloques a
R y restauró el literal de D. Recuperó byte a byte:

- R: 800 líneas, SHA `8e15443b120dc68721aa4cc0959610ca393af44d842f0e07ed7e0b18873fc059`;
- D: 164 líneas, SHA `039b75dd15a2888798c7f257c46fdbb97587cbdd4a6519e11cb043cce0e72e5e`.

La revisión funcional de la versión corregida emitió **GO**, `P0=0`, `P1=0`,
`P2=0`. La revisión de seguridad emitió **GO**, `P0=0`, `P1=0`, `P2=0`.

## Autorización documental anterior

En el momento de publicar el contrato se autorizó confirmar esta acta y
producir P0 únicamente desde aquel commit y después de una CI 5/5.

El candidato material debía superar en un worktree exclusivo:

- write-set exacto de dos rutas;
- conteos y huellas R/D contractuales;
- reconstrucción directa e inversa;
- guardas e inventario de carga;
- Bash y ShellCheck;
- manifiesto de nueve entradas y captura privada;
- fuentes Go invariantes y dos builds reproducibles;
- autoprueba y modos cerrados en 64;
- H0 PostgreSQL 18.4 por digest;
- calidad global, Gitleaks, diff y residuos cero;
- doble revisión independiente antes de integrar o publicar.

Esta sección conserva la autorización documental histórica. El apartado
«Cierre material posterior» registra que todas esas puertas se ejecutaron
después sobre `ac988d6`; no se atribuyeron al candidato antes de existir.

## Alcance y métricas

P0 no implementa `Start`, mapa FD, pidfd, señales, `/proc`, ticket, plazos,
`CONT`, `Wait`, terminal ni modo operativo. Tampoco cierra O3a, C4b-2, C4b,
H0b, C2, F0 u O4-05.

Las métricas permanecen:

- F0 `10/23`;
- O4-05 `3/5`;
- Contratación temporal `24/46`;
- Bolsa productiva `1/14`;
- producción `NO-GO`.

## Cierre material posterior

El traslado se confirmó en
`ac988d615d021b14cbcfc2929f715ab4d9bb5567`, hijo directo del contrato
publicado `afd00c35d7981cf136198746be88e631213bb08e`. Su write-set contiene
exclusivamente las dos rutas autorizadas y el árbol quedó limpio.

| Unidad | Líneas | SHA-256 |
| --- | ---: | --- |
| Runner R | 702 | `e617024a52c4a042971b026d0799816933b489ed4221e9b6147317936d18054c` |
| Auxiliar D2d | 264 | `681efbbd7f856eb539d1656cffed87c26f48609e65d6d6adf8265c350ae69442` |
| Binario Go reproducible | — | `6153f03a93c0a2618fdaf922443004244aa3bec7cbe9074466b22935c693edd0` |

Los revisores funcional y de seguridad emitieron por separado **GO**,
`P0=0`, `P1=0`, `P2=0`. Ambos reconstruyeron byte a byte el padre, verificaron
las dos directivas locales, las guardas de carga, el inventario exacto de
diecinueve funciones, el manifiesto de nueve entradas y la invariancia de las
fuentes Go, capturador y adaptador.

Las dos revisiones reprodujeron `bash -n`, ShellCheck, dos builds Go 1.26.5,
autoprueba, modos cerrados, H0 PostgreSQL 18.4, calidad global, carrera,
`govulncheck`, Gitleaks y residuos cero. La primera ejecución H0 con
`LD_LIBRARY_PATH` heredado cerró correctamente en 65 antes de reservar
recursos; la ejecución canónica retirando `LD_*` satisfizo la precondición
explícita y terminó verde. No se ocultó un fallo material.

O3a-P0 no suma una tarea funcional ni cambia las métricas. Antes de redactar
el contrato O3a debe publicarse este cierre y obtenerse CI 5/5. O3a, `Start`,
mapa FD y producción permanecen en `NO-GO`.
