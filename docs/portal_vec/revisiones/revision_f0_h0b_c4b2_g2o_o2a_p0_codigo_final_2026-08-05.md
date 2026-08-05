# Revisión final F0-H0b/C4b-2/G2-O/O2a-P0

Fecha: 5 de agosto de 2026.

Estado: **GO integrado, publicado y con CI verde**.

## Alcance

O2a-P0 es una minitarea estructural sin cambio funcional. Traslada la función
Shell `derivar_repo_base_h0_f0` desde el runner H0 al auxiliar privado D2d para
dejar capacidad legible en el runner antes de diseñar la fuente G3 real.

No crea G3, no implementa la máquina S0 ni abre O2a u O2b. Tampoco añade FD,
proceso, Bash operativo, modo, red, SQL, migración, ACK o autoridad nueva.

## Trazabilidad

| Hito | Referencia |
| --- | --- |
| Base exacta del candidato | `ec530091e6f157baa54ff50e9c70f21c7a014e94` |
| Autorización documental | `8f29aeb87667889198a23ff716c3b4737e9e95b1` |
| Enmienda local de ShellCheck | `771f924788f90149c8fe76b3d4b71e3534e56118` |
| Candidato revisado | `df9a422c44daf5d1ec2491bff4fd5c228d0dd13c` |
| Integración | `48a46a3` |

La autorización y su enmienda obtuvieron cada una dos GO documentales
independientes con `P0=P1=P2=0`. El primer intento material se detuvo sin
commit al descubrir `SC2154`; no se silenció globalmente ni se declaró una
variable artificial. La enmienda autorizó una sola directiva local, explicada
y sin efecto en ejecución.

## Cambio material

El commit candidato modifica únicamente:

1. `deploy/postgresql/autorizacion_atestada_v3/probar_fuente_corporativa_contexto_actor_v1_pg18_4.sh`;
2. `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/operaciones_runner_fuente_corporativa_contexto_actor_v1.sh`.

El `numstat` es:

```text
runner  +1/-18
D2d     +19/-0
```

La única línea que no pertenece al bloque trasladado es:

```bash
# shellcheck disable=SC2154  # Aportado por el runner acreditado.
```

La directiva queda inmediatamente antes de la función, no declara ni
inicializa `contenedor`, no es global y no oculta otros usos sin contrato.

## Ledger e invariantes

| Unidad | Líneas | SHA-256 |
| --- | ---: | --- |
| Runner | 783 | `da6871ca174890c85eb93ee4cfac15f32ecd1ac046d84d24fa68170ac34c52e9` |
| D2d | 164 | `039b75dd15a2888798c7f257c46fdbb97587cbdd4a6519e11cb043cce0e72e5e` |
| Bloque trasladado | 17 | `aae98945ae26e7b4f2637e662157bdaf26a414d3100b046d2c91c4cf1fa59d74` |
| G1 | 686 | `9fab2cae4edd0b5cf8cd5d67fd7a1f9643b81085c815b0c10cb477f67a7e1afe` |
| G2 | 798 | `01acb818e9abefcbfe4c279bb0dd5e3317bf03f082f1ed3fba4f257c5642866b` |
| Capturador | 799 | `4a967fd13bac213ea7ebf7316af98dcc9a9dfb39b9b3b28f68e0c91958878902` |
| Adaptador M38 | 527 | `98d22a302bfd8ad3964b9135ce78c655f7a31171088ad9c5c49c285f647a8cb7` |
| D2c | 588 | `a07057fb15315c5d2d0d10d6f3beea85f196fc78598cfcc4d1f63918bcbadde5` |
| H0b | 580 | `02a00f2fc49e181d1cf8ed147a927155899956dbdbd7f36f3443ee4d7cbafded` |

El literal del binario Go compuesto permanece
`4ae175f326145be4f9cc81908bc3fa381abedc21576aaab1ade1ca8551284419`.
El manifiesto privado conserva seis fuentes. Hay cero definiciones de la
función en el runner, una en D2d y una llamada en el runner. `contenedor` se
inicializa antes de capturar y cargar D2d, y la llamada ocurre después.

## Revisiones independientes de código

| Revisión | P0 | P1 | P2 | Veredicto |
| --- | ---: | ---: | ---: | --- |
| Función, autoridad y fallo cerrado | 0 | 0 | 0 | GO |
| Ledger, captura y reproducibilidad | 0 | 0 | 0 | GO |

La revisión funcional reprodujo `bash -n`, ShellCheck, rechazo directo y de
carga no acreditada en 64, una pasada H0 real y residuos cero. Dos invocaciones
previas se rechazaron antes de reservar recursos por no respetar el régimen
privilegiado del shebang y por heredar `LD_LIBRARY_PATH`; no produjeron un falso
H0. La ejecución material válida usó el runner acreditado con
`LD_LIBRARY_PATH` retirado.

La revisión de ledger reprodujo el write-set, `numstat`, líneas, huellas,
cardinalidades, orden, directiva local, manifiesto, invariantes, cierre 64,
`git diff --check` y Gitleaks. Una mutación virtual posterior siguió provocando
`SC2154`, por lo que la exclusión no es global.

## Puertas del candidato

- `bash -n`: verde;
- ShellCheck: verde;
- H0 completo en PostgreSQL 18.4 por digest y sin red: verde;
- `go test ./...`: verde;
- `go vet ./...`: verde;
- `go test -race ./...`: verde;
- `scripts/verificar_calidad.sh`: verde;
- Gitleaks: cero hallazgos;
- residuos Docker y temporales F0: cero.

## Reproducción tras integrar

Dirección integró mediante `cherry-pick` sin conflictos. El árbol conjunto
conservó las huellas exactas. Después reprodujo:

1. H0 completo en PostgreSQL 18.4 por digest, `--network none`, autopruebas,
   SQLSTATE reales, snapshot y arnés transaccional: verde;
2. residuos Docker y temporales F0: cero;
3. `scripts/verificar_calidad.sh`: verde, incluida carrera global, aislamiento
   de grafos, manifiestos web, TLS como proceso no privilegiado, auditoría de
   vulnerabilidades y tamaños;
4. Gitleaks sobre el commit integrado: un commit y 944 bytes, cero secretos.

## Publicación

El cierre se publicó en `ef027cf2f94955cdf6de9c091c531a85d05c2e04`. La
ejecución GitHub `31007941124` terminó con cinco de cinco puertas verdes:
secretos, artefactos productivos, ContextoActor/PDP V3 en PostgreSQL 18, Bolsa
pública en PostgreSQL 18 con TLS y calidad global.

## Límites y continuación

O2a-P0 no cambia F0 `10/23`, O4-05 `3/5`, Contratación temporal `24/46`,
Bolsa productiva `1/14` ni el `NO-GO` de producción.

El siguiente trabajo es redactar y revisar el contrato funcional y ledger real
de O2a/S0/G3. No se autoriza crear
un cascarón G3 ni programar O2a antes de ese cierre documental.
