# Revisión final F0-H0b/C4b-2/G2-O/O1b

Fecha: 5 de agosto de 2026.

Estado: **GO integrado pendiente únicamente de acreditar la CI remota**.

## Alcance cerrado

O1b incorpora al supervisor privado el lector incremental puro de las tramas
definidas por O1a. Acredita fragmentación, coalescencia, monoframa, límites,
precedencias, terminalidad y fallo cerrado sin abrir descriptores, procesos,
Bash, Docker, PostgreSQL, SQL o red.

No implementa el bootstrap operativo O2 ni cierra G2-O, C4b-2, C4b, H0b, C2,
F0, O4-05 o producción.

## Historia de revisión

1. `56c0ac079419b187f851de8183d5f87b5b367b71` implementó el primer candidato
   O1b, pero recibió `NO-GO` funcional: tres mutantes de constructor,
   contenido y L1 sobrevivían y faltaban transiciones explícitas.
2. `a4737513dd5bf24d98cdd4c82d6702a4d28594df` corrigió esa matriz. La revisión
   funcional dio `GO`, pero la revisión de reproducibilidad descubrió un P1:
   un mutante que degradaba L4 a L0 después de repetir un error pegajoso seguía
   verde. El candidato no se integró.
3. `61af56b920982e2e2af6a79c561e2c7e6c7f3ff7` añadió la aserción terminal
   ausente sin aumentar el tamaño. Dos revisiones independientes finales
   emitieron `GO`, `P0=P1=P2=0`.

Los commits quedaron integrados, sin conflictos y en el mismo orden, como:

- `eb2bba0`, implementación O1b;
- `c654855`, corrección probatoria de la matriz;
- `98b753e`, persistencia terminal L4.

## Matriz funcional acreditada

- constructor real para cada clase: lector y error, clase, límite, L0,
  longitud, buffer y error interno exactos;
- rechazo cerrado de clase inválida con lector nulo y centinela exacto;
- comparación byte a byte de cada entrega con su entrada canónica;
- todos los cortes, fragmentación byte a byte, L2, entrega directa,
  coalescencia y reconstrucción desde parcial;
- estados L0, L1, L2, L3 y L4 afirmados explícitamente;
- monoframa inválida retenida en L2 y rechazada al EOF;
- dato posterior desde L2 y cola en el mismo fragmento, con precedencia sobre
  el error gramatical tanto con `fin=false` como con `fin=true`;
- EOF inicial y repetido, dato desde L3, CONTROL+EOF, limpieza, copia
  defensiva, límites y bytes hostiles;
- tupla cero, buffer limpio, identidad del error, pegajosidad y L4 absorbente
  después de cada error.

Los cuatro mutantes focales terminan en estado 65:

1. constructor que devuelve siempre CONTROL;
2. borrado de campos y ticket de la monoframa;
3. transición parcial L1 a L0;
4. transición terminal L4 a L0 después del error repetido.

## Ledger físico e invariantes

| Unidad | Líneas | SHA-256 |
| --- | ---: | --- |
| Runner | 800 | `e37d8a51a5b961ef9175833ba78c47aab4e8e29180db6ff9d771498fa3d16d87` |
| G1 | 686 | `9fab2cae4edd0b5cf8cd5d67fd7a1f9643b81085c815b0c10cb477f67a7e1afe` |
| G2 | 798 | `01acb818e9abefcbfe4c279bb0dd5e3317bf03f082f1ed3fba4f257c5642866b` |

El binario privado reproducido por dos compilaciones aisladas tiene SHA-256
`4ae175f326145be4f9cc81908bc3fa381abedc21576aaab1ade1ca8551284419`.
Capturador, adaptador, D2d, D2c y H0b permanecieron byte a byte invariantes;
el runner solo cambió los dos literales de G2 y del binario compuesto.

## Puertas reproducidas

Sobre el candidato final, antes de integrar:

- dos compilaciones privadas con raíces, `HOME`, `TMPDIR` y `GOCACHE`
  disjuntos, binarias idénticas y fuentes estables;
- 40/40 autopruebas, sin hijos ni diferencia de descriptores;
- `--supervisar-m38` y modo desconocido cerrados con estado 64;
- `gofmt`, `go vet`, `bash -n`, ShellCheck, Gitleaks y
  `git diff --check` verdes;
- `go test ./... -count=1 -timeout 20m`;
- `go test -race ./... -count=1 -timeout 30m`;
- `go vet ./...`;
- `scripts/verificar_calidad.sh`.

Después de integrar se repitió `scripts/verificar_calidad.sh` sobre el árbol
conjunto. Terminó verde e incluyó pruebas normales y con carrera, análisis
estático, compilación, aislamiento de dependencias y manifiestos público e
interno, carga TLS no privilegiada, PDF, `govulncheck`, tamaños y diferencias.
No se encontraron vulnerabilidades conocidas ni fugas de secretos.

## Decisión

O1b queda técnicamente aceptada e integrada. Las métricas funcionales no se
incrementan porque H0b continúa abierto. La siguiente minitarea es definir y
revisar el contrato y ledger de G2-O2, limitado al bootstrap
`ARMAR/ACK_LISTO/CANCELAR/EOF` sin crear Bash. O2 permanece cerrado hasta que
ese contrato obtenga revisión independiente.
