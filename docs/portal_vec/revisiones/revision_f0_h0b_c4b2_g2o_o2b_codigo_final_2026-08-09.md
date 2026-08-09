# Revisión final F0-H0b/C4b-2/G2-O/O2b

Fecha: 9 de agosto de 2026.

Estado: **GO técnico integrado localmente; P0=0, P1=0 y P2=0**.

Este cierre no autoriza producción, no abre todavía el modo operativo y no
incrementa las métricas funcionales. La publicación y su CI siguen siendo una
puerta posterior a este acta.

## Alcance exacto

O2b incorpora el control previo de `ARMAR` y `CANCELAR` sin crear Bash, abrir
FD operativos, ejecutar procesos, acceder a red o modificar PostgreSQL. La
máquina privada:

- recibe y valida el control incremental bajo la autoridad única del lector
  O1b fijado por huella;
- conserva una única copia acotada del `nonce` necesario y prohíbe retener el
  sobre completo;
- admite las transiciones previas S1→S2→S3 y enclava los fallos en S5;
- normaliza solo las causas previstas y falla cerrada ante estados, errores o
  tuplas estructurales incompatibles;
- limpia lector, receptor y referencias sensibles antes de una salida
  terminal;
- mantiene `--supervisar-m38` y cualquier modo desconocido cerrados con estado
  64.

La correspondencia byte a byte y la gramática pertenecen exclusivamente a
O1b/G2. O2b no implementa una segunda gramática: verifica las postcondiciones
estructurales observables y liga G2 por ruta, líneas y SHA-256.

## Trazabilidad inmutable

| Hito | Commit |
| --- | --- |
| Base documental autorizada | `5dff807652e11db6c7d446c5135e7301fc721c36` |
| Código material | `d86aea8b4ed2b9fffbf74ef04cd90f397b017a55` |
| Evidencia AST y mutaciones | `4b39265405957b47b909f0b9e1bc4960c38f4011` |

El commit material modifica exactamente el runner y G1, y añade G4 y G5. El
commit probatorio añade únicamente el analizador AST y el mutador como texto
regular con modo Git `100644`. La genealogía exacta es:

```text
5dff807652e11db6c7d446c5135e7301fc721c36
  -> d86aea8b4ed2b9fffbf74ef04cd90f397b017a55
  -> 4b39265405957b47b909f0b9e1bc4960c38f4011
```

## Ledger final

| Unidad | Líneas | SHA-256 |
| --- | ---: | --- |
| Runner H0 | 800 | `8e15443b120dc68721aa4cc0959610ca393af44d842f0e07ed7e0b18873fc059` |
| G1, supervisor privado | 692 | `6b7f93b8b43c1040cc4ae2b6322c4e99e914eee415475e3fd50bf294b5a17afb` |
| G4, control previo | 404 | `2befe2a4c16fc7a57aacd421ea6c8419ab49160bb2ae0d0eb6f03786194aa744` |
| G5, autoprueba O2b | 507 | `10ccaf8347bfcaa5f3990b75b4c9becd62cd39b60249b628af6c7a1fc6bc8867` |
| G2, lector O1b invariante | 798 | `01acb818e9abefcbfe4c279bb0dd5e3317bf03f082f1ed3fba4f257c5642866b` |
| G3, receptor O2a invariante | 431 | `d608868ecb2cb753876f488b522975e05af06c013c82222959be5d85100c3633` |
| Binario, dos builds Go 1.26.5 | — | `6153f03a93c0a2618fdaf922443004244aa3bec7cbe9074466b22935c693edd0` |

El manifiesto privado contiene nueve entradas en orden fijo. G2 y G3 se
mantuvieron byte a byte durante todos los mutantes O2b.

## Evidencia durable

| Artefacto | Líneas | SHA-256 |
| --- | ---: | --- |
| [Analizador AST](evidencias/f0_h0b_c4b2_g2o_o2b_ast_d86aea8.go.txt) | 799 | `1993cc4e6b9e35d9a08e5318b2713bd2e9826eb78b205c6b6e40fb1a33d00ec1` |
| [Mutador](evidencias/f0_h0b_c4b2_g2o_o2b_mutantes_d86aea8.sh.txt) | 509 | `37814b9491885103ce33dc4e2315542b35fefe974f30d697c03964e2c68aa2a4` |

La reproducción debe hacerse en un worktree limpio separado y situado
exactamente en `4b39265405957b47b909f0b9e1bc4960c38f4011`. Los artefactos son
evidencia histórica ligada al par de commits anterior, no analizadores
genéricos para un descendiente futuro.

La matriz final obtuvo:

```text
mutantes aplicados:       31/31
muertos por AST:           7
muertos por autoprueba:   24
supervivientes:            0
fallos de build:           0
falsos muertos:            0
residuos temporales:       0
```

Cada mutante pasa primero por patrón único, `gofmt`, `go vet` y build. Solo
después se aplica el oráculo AST o conductual. El modo mutante relaja la huella
de G4, pero conserva G1, G2, G3 y G5, por lo que una mera discrepancia de hash
no puede contarse como mutante muerto.

## Defectos detectados antes del GO

La revisión independiente detuvo varias versiones sin confirmarlas como
evidencia final:

1. una tupla L0 físicamente imposible podía aceptarse;
2. faltaban postcondiciones comunes de clase, límite, error y contador;
3. una huella binaria se había calculado con Go 1.25.5 y no con Go 1.26.5;
4. la limpieza inicial del mutador podía dejar su directorio temporal;
5. la primera guarda solo contemplaba el estado anterior al commit de
   evidencia y no su reproducción posterior desde un árbol limpio;
6. el AST podía contar una discrepancia de G4 como falso mutante muerto;
7. varias envolturas tipadas permitían ocultar una segunda copia del sobre.

Las correcciones finales son fail-closed, bifásicas y tipadas. Se falsaron con
variantes compilables de parámetro y retorno por valor, estructura, puntero,
array, mapa, slice, interfaz, conversión, clausura y canal. Todas fueron
rechazadas y la base legítima permaneció verde.

## Puertas reproducidas

- `gofmt`, `go vet`, Bash, ShellCheck y `git diff --check`: verdes;
- dos builds privados con Go 1.26.5: byte a byte idénticos;
- autoprueba real: estado 0, salida exacta y `stderr` vacío;
- modo operativo y modo desconocido: estado 64;
- AST base y 31 mutantes: verdes conforme a sus oráculos;
- H0 completo sobre PostgreSQL 18.4 fijado por digest: verde;
- `scripts/verificar_calidad.sh`: verde, incluida la auditoría de
  vulnerabilidades y el límite de tamaño;
- Gitleaks sobre ambos commits: cero hallazgos;
- contenedores y temporales residuales atribuibles a la prueba: cero.

## Revisiones independientes

| Revisión | P0 | P1 | P2 | Veredicto |
| --- | ---: | ---: | ---: | --- |
| Funcional, postcondiciones y falsación tipada | 0 | 0 | 0 | GO |
| Seguridad, write-set, mutaciones y secretos | 0 | 0 | 0 | GO |

Dirección reprodujo además la matriz completa, H0 y la puerta global sobre el
par exacto antes de integrarlo mediante avance rápido.

## Límites y continuación

O2b no acredita Docker operativo, Bash hijo, mapa de FD, `pidfd`, barreras,
STOP/CONT/TERM/KILL, `Wait`, canal de vida, consumo Shell, modo operativo,
interfaz web ni E2E. Tampoco cierra C4b-2, C4b, H0b, C2, F0 u O4-05.

Las métricas permanecen F0 `10/23`, O4-05 `3/5`, Contratación temporal
`24/46`, Bolsa productiva `1/14` y producción `NO-GO`.

La siguiente minitarea es **O3a: `Start` y mapa FD**. Debe recibir un contrato
nuevo, acotado y revisado antes de modificar código.
