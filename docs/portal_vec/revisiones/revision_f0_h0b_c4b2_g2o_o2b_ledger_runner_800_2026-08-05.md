# Revisión O2b: ledger correctivo del runner en 800 líneas

Fecha: 5 de agosto de 2026.

Dictamen: **GO documental final, `P0=0`, `P1=0`, `P2=0`**.

## Objeto

Se revisó íntegramente la enmienda:

```text
docs/portal_vec/enmienda_f0_h0b_c4b2_g2o_o2b_ledger_runner_800_2026-08-05.md
```

La versión normativa revisada antes de actualizar únicamente su estado tenía
177 líneas y esta huella:

```text
b8af6920e5de127670e784cec3fc1f6555a14583a74009b268e2256341df07f5
```

La actualización de estado y esta acta necesitan una contrarrevisión final
antes del commit. El futuro commit publicado será el padre material exacto de
la reanudación del candidato O2b.

## Parada acreditada

El candidato llegó a una línea base focal verde con runner de 802 líneas. La
puerta global se detuvo correctamente en
`scripts/comprobar_tamano_ficheros.sh`, porque DEC-051 fija un máximo duro de
800 y no existe ledger que permita elevarlo. La parada ocurrió antes de H0,
commit o publicación.

Se conservaron sin commit:

| Unidad | Líneas | SHA-256 |
| --- | ---: | --- |
| G1 | 692 | `6b7f93b8b43c1040cc4ae2b6322c4e99e914eee415475e3fd50bf294b5a17afb` |
| G4 producción | 399 | `4edba44910779e67513cc1af60ef38752cb99f5f325305783af4493b5b3f6af4` |
| G5 autoprueba | 490 | `839df34f2669772d932b0e804ebb44bd66bf3991316bb59c63537b55769c860a` |
| G2 | 798 | `01acb818e9abefcbfe4c279bb0dd5e3317bf03f082f1ed3fba4f257c5642866b` |
| G3 | 431 | `d608868ecb2cb753876f488b522975e05af06c013c82222959be5d85100c3633` |

El binario compuesto reproducible conservado tiene SHA-256
`2c9fb2ef23f4a23d503bca5ba754eda32d0afbd611037392d96309b92e3dcacb`.

## Solución revisada

La enmienda no rebaja el límite ni comprime ninguna garantía. Mantiene
separadas las ocho líneas nuevas que acreditan rutas, huellas, comparaciones y
pares de G4/G5, y recupera dos líneas existentes mediante dos únicas
reagrupaciones Bash:

```bash
local fuente_g4 fuente_g5 segundo par esperada fase destino_binario raiz_go lineas=()
```

```bash
supervisor_m38="${temporales}/supervisor-m38" segundo="${temporales}/supervisor-m38-segundo"
```

La primera conserva `lineas` como array indexado local y no altera
`mapfile -t`, su cardinalidad ni las comparaciones. La segunda contiene dos
asignaciones simples independientes, sin órdenes encadenadas ni expansiones
dependientes. El ledger exacto es `794 + 8 - 2 = 800`.

## Revisiones independientes

La revisión funcional verificó el documento completo y emitió `GO`,
`P0=P1=P2=0`. Confirmó que las dos reagrupaciones no modifican conducta ni
seguridad y conservan las ocho garantías, las huellas, el write-set, las
puertas y las paradas; solo cambian la forma física Bash y el ledger
exactamente como `794 + 8 - 2 = 800`.

La revisión estructural detectó inicialmente un `P2`: la prosa llamaba
«segunda línea local» a la declaración que debía modificarse. Dirección la
identificó de forma inequívoca como la línea `local` que declara `fuente_g4` y
`fuente_g5`. La contrarrevisión de la versión de 177 líneas y huella
`b8af6920e5de127670e784cec3fc1f6555a14583a74009b268e2256341df07f5`
cerró el hallazgo y emitió `GO`, `P0=P1=P2=0`.

## Autorización limitada

Este dictamen autoriza confirmar y publicar exclusivamente la enmienda y esta
acta. Solo después de que la CI de cinco puertas quede verde se puede avanzar
por *fast-forward* la rama candidata y modificar el runner para aplicar las
dos reagrupaciones exactas.

G1, G4, G5, G2, G3 y el resto del árbol deben permanecer byte a byte. Una
tercera reagrupación, una huella distinta, un runner que no tenga exactamente
800 líneas o cualquier puerta fallida obliga a detener sin confirmar.

La integración del código futuro continúa sujeta a todas las pruebas focales,
dos builds reproducibles, puerta global, H0 PostgreSQL 18.4 con residuos cero,
evidencia portable y dos revisiones de código independientes.

No se cierra O2b ni cambia ninguna métrica: F0 `10/23`, O4-05 `3/5`,
Contratación temporal `24/46`, Bolsa productiva `1/14` y producción `NO-GO`.
