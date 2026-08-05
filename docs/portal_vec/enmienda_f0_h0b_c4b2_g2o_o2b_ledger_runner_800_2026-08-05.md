# Enmienda O2b: ledger del runner bajo el límite global de 800

Fecha: 5 de agosto de 2026.

Estado: **doble `GO` documental final, `P0=P1=P2=0`**. La edición material
solo se reanuda después de confirmar y publicar esta enmienda con su acta y de
comprobar verde la CI de ese padre exacto.

## Motivo y parada acreditada

La enmienda G4/G5 publicada en
`971a18026a01d50d46e82ee79f60fc88a78f97f1` proyectó el runner desde 794
hasta 802 líneas. Su CI `31028723874` terminó con cinco de cinco puertas
verdes, por lo que autorizó reanudar el candidato.

El candidato material llegó a una línea base focal verde, pero
`scripts/verificar_calidad.sh` lo detuvo al ejecutar
`scripts/comprobar_tamano_ficheros.sh`: DEC-051 fija un límite global duro de
800 líneas y no permite crear una excepción nueva. La salida observada fue:

```text
deploy/postgresql/autorizacion_atestada_v3/probar_fuente_corporativa_contexto_actor_v1_pg18_4.sh: 802 lineas (limite 800, linea base 0)
```

La parada se produjo antes de H0, commit o publicación. No se modifica la
puerta global ni su línea base. Se corrige el ledger para conservar todos los
controles en exactamente 800 líneas.

## Prevalencia limitada

Esta enmienda sustituye únicamente, para O2b, las cláusulas de la separación
G4/G5 que exigen:

```text
runner 794 -> 802
delta neto +8
```

La forma vigente pasa a ser:

```text
runner 794 -> 800
ocho líneas de garantía añadidas
dos líneas existentes reagrupadas
delta neto +6
```

No cambia API, producción G4, autoprueba G5, G1, G2, G3, manifiesto, hashes,
matrices, AST, mutantes, puertas ni prohibiciones de los contratos publicados.

## Forma Bash exacta

Las ocho líneas nuevas de garantía permanecen independientes:

1. ruta G4;
2. ruta G5;
3. SHA-256 G4;
4. SHA-256 G5;
5. comparación G4 en el índice 5 del manifiesto;
6. comparación G5 en el índice 6;
7. par G4 en el bucle de acreditación;
8. par G5 en el bucle de acreditación.

No se agrupan rutas, hashes, comparaciones, pares, órdenes de captura,
argumentos de `go vet` o fuentes de los builds.

Se recuperan dos líneas agrupando solo declaraciones y asignaciones simples
ya existentes.

Primero, la declaración local del array se incorpora a la línea `local` que
declara `fuente_g4` y `fuente_g5`:

```bash
local fuente_g4 fuente_g5 segundo par esperada fase destino_binario raiz_go lineas=()
```

Se elimina la línea separada:

```bash
local -a lineas=()
```

En Bash, `lineas=()` conserva el tipo de array indexado local. `mapfile -t`,
la cardinalidad y las nueve comparaciones mantienen su conducta.

Segundo, los dos destinos temporales se asignan en una sola orden de
asignación simple:

```bash
supervisor_m38="${temporales}/supervisor-m38" segundo="${temporales}/supervisor-m38-segundo"
```

No hay `;`, tubería, condición, expansión dependiente, orden encadenada ni
minificación. Las líneas quedan dentro del margen legible del runner.

## Ledger material

| Unidad | Estado detenido | Estado autorizado | Invariante |
| --- | ---: | ---: | --- |
| Runner | 802 | exactamente 800 | solo las dos reagrupaciones anteriores |
| G1 | 692 | exactamente 692 | byte a byte |
| G4 producción | 399 | exactamente 399 | byte a byte |
| G5 autoprueba | 490 | exactamente 490 | byte a byte |
| G2 | 798 | exactamente 798 | byte a byte |
| G3 | 431 | exactamente 431 | byte a byte |
| Manifiesto | 9 | exactamente 9 | mismo orden y contenido |

Huellas del candidato detenido que la corrección no puede alterar:

| Fuente | SHA-256 |
| --- | --- |
| G1 | `6b7f93b8b43c1040cc4ae2b6322c4e99e914eee415475e3fd50bf294b5a17afb` |
| G4 | `4edba44910779e67513cc1af60ef38752cb99f5f325305783af4493b5b3f6af4` |
| G5 | `839df34f2669772d932b0e804ebb44bd66bf3991316bb59c63537b55769c860a` |
| G2 | `01acb818e9abefcbfe4c279bb0dd5e3317bf03f082f1ed3fba4f257c5642866b` |
| G3 | `d608868ecb2cb753876f488b522975e05af06c013c82222959be5d85100c3633` |
| Binario compuesto | `2c9fb2ef23f4a23d503bca5ba754eda32d0afbd611037392d96309b92e3dcacb` |

La reagrupación Bash no cambia el binario Go esperado.

El manifiesto conserva exactamente:

| Índice | Fuente |
| ---: | --- |
| 0 | D2c |
| 1 | H0b |
| 2 | adaptador M38 |
| 3 | D2d |
| 4 | G1 |
| 5 | G4 producción |
| 6 | G5 autoprueba |
| 7 | G2 operativo |
| 8 | G3 sobre S0 |

## Write-set y reanudación

El contrato y esta enmienda se confirman primero en la rama integradora. Solo
después de publicación y CI de cinco puertas verde se avanza por fast-forward
la rama candidata al nuevo padre exacto.

La corrección material posterior solo puede modificar el runner detenido para
aplicar las dos reagrupaciones. G1, G4, G5, G2, G3 y todo el resto quedan byte
a byte. No se actualizan documentos dentro del commit de código candidato.

## Puertas que deben repetirse

La corrección de dos líneas obliga a repetir, no a heredar:

- `wc -l`: runner exactamente 800;
- `bash -n` y ShellCheck;
- captura privada y manifiesto real de nueve;
- acreditación individual antes/después de G1, G4, G5, G2 y G3;
- `go list`, `go vet`, autoprueba y modos 64/64;
- dos builds aislados `-a -trimpath` con el mismo SHA binario;
- `git diff --check` y Gitleaks;
- `scripts/verificar_calidad.sh` completo y verde;
- H0 PostgreSQL 18.4 sobre imagen fijada por digest, con residuos cero;
- dos revisiones independientes del futuro commit de código.

Las pruebas focales verdes previas no sustituyen estas repeticiones.

## Paradas

Detener sin confirmar si:

- el runner no queda exactamente en 800;
- cambia alguna de las ocho líneas de garantía;
- aparece una tercera reagrupación o una línea supera el margen legible;
- cambia una huella o línea de G1/G4/G5/G2/G3;
- cambia el orden o cardinalidad del manifiesto;
- la puerta global no termina verde;
- H0 deja residuos o cualquier revisión emite `NO-GO`.

## Efecto limitado

Esta corrección resuelve exclusivamente el conflicto de tamaño. No cierra O2b
ni modifica F0 `10/23`, O4-05 `3/5`, Contratación temporal `24/46`, Bolsa
productiva `1/14` o el `NO-GO` de producción.
