# Revisión O2b: postcondición estructural bajo autoridad O1b

Fecha: 9 de agosto de 2026.

Dictamen: **GO documental final, `P0=0`, `P1=0`, `P2=0`**.

## Objeto y versión aprobada

Se revisó íntegramente:

```text
docs/portal_vec/enmienda_f0_h0b_c4b2_g2o_o2b_postcondicion_lector_2026-08-09.md
```

La versión normativa aprobada antes de cambiar únicamente su estado tenía 396
líneas y SHA-256:

```text
92d7bbe26d8e68419d209c43c7eb618370c9816e8d1c49f3249640c0680aa813
```

La versión con estado final y esta acta deben recibir contrarrevisión íntegra
de ambos revisores antes de confirmarse. El dictamen autoriza solo el par
documental; no acepta todavía código material.

## Hallazgo y parada

Después de corregir la relación entre progreso y L0/L1, la revisión reprodujo
que la barrera G4 admitía contradicciones estructurales que O1b no puede
devolver:

- resultado sin fallo junto a error interno pegajoso;
- clase o límite de lector distintos de `CONTROL/1024`;
- contador superior al límite físico;
- necesita-datos con más bytes consumidos que longitud parcial posterior.

G2 no produce esas combinaciones y no se observó fuga, acceso indebido,
credencial expuesta ni vía externa para fabricar la tupla. El candidato se
detuvo antes de commit material, H0 o integración porque G4 debe fallar cerrado
ante toda contradicción estructural que pertenezca a su contrato.

## Decisión de arquitectura

Las revisiones estudiaron dos opciones:

1. copiar el lector, repetir O1b y recodificar cada trama en G4;
2. conservar O1b/G2 como autoridad única de bytes fijada por SHA y limitar G4
   a postcondiciones estructurales observables.

Se aprobó la segunda. La tupla nace de dos call-sites directos al mismo método
G2 dentro del binario privado: consumo normal y drenaje EOF. No cruza red,
proceso, plugin, interfaz o deserialización.

Repetir el lector duplicaría CPU, buffer y representaciones de nonce, PID y
causas, sin independencia real ante un proceso o toolchain comprometidos.
Reimplementar la gramática crearía una segunda autoridad divergente. La
protección efectiva consiste en fijar G2 por 798 líneas, SHA exacto, índice 7
del manifiesto, snapshot privado y binario compuesto reproducible. Cualquier
cambio en G2 detiene O2b y exige revisión conjunta O1b/O2b.

G4 no acredita correspondencia byte a byte ni causalidad error--entrada. Esas
propiedades siguen cubiertas por la autoprueba y evidencia reales O1a/O1b. G4
sí valida clase, límite, contadores, estados, limpieza, error coherente y
relación estructural L0/L1.

## Evolución de las revisiones

### Primera corrección focal

La primera propuesta añadió clase `CONTROL`, límite 1024 y coherencia de
`l.err`. Dos revisores validaron técnicamente esa mejora, pero una matriz
ampliada detectó contadores físicamente imposibles. No se confirmó código.

### Segunda corrección focal

Se añadieron `n <= l.limite` y, con progreso necesita-datos,
`n <= l.longitud`. La prueba funcional acreditó casos legítimos hasta 1023,
trama de 1024, M30 y M31. Se corrigió también una huella binaria obtenida por
error con Go 1.25.5: la autoridad real es Go 1.26.5 y el binario reproducible
`6153f03a...`.

La revisión de seguridad planteó relaciones de contenido entre sufijo, buffer,
trama y error. El análisis conjunto concluyó que pertenecen a O1b y que
duplicarlas en G4 contradecía la separación vigente. El contrato se reescribió
para declarar la frontera exacta y sus pruebas compensatorias.

### Correcciones documentales

Antes del GO final se corrigieron:

- autoridad ambigua de la enmienda sustituida;
- dos call-sites G2, no uno;
- par exacto G4 vulnerable/G5 final y stderr esperado;
- transformación compilable M26 mediante reemplazo del subárbol por `true`;
- tres oráculos completos con estado 65, stdout vacío y stderr exacto;
- separación de M1--M25 y mapeo M26, M27--M29 y M30--M31;
- toolchain, binario y runner proyectados de Go 1.26.5.

Cada corrección fue solo documental o se validó en un árbol temporal. Ningún
revisor editó la candidata ni la integradora.

## Ledger reproducido

| Unidad | Líneas | SHA-256 final proyectado |
| --- | ---: | --- |
| Runner | 800 | `8e15443b120dc68721aa4cc0959610ca393af44d842f0e07ed7e0b18873fc059` |
| G1 | 692 | `6b7f93b8b43c1040cc4ae2b6322c4e99e914eee415475e3fd50bf294b5a17afb` |
| G4 | 404 | `2befe2a4c16fc7a57aacd421ea6c8419ab49160bb2ae0d0eb6f03786194aa744` |
| G5 | 507 | `10ccaf8347bfcaa5f3990b75b4c9becd62cd39b60249b628af6c7a1fc6bc8867` |
| G2 | 798 | `01acb818e9abefcbfe4c279bb0dd5e3317bf03f082f1ed3fba4f257c5642866b` |
| G3 | 431 | `d608868ecb2cb753876f488b522975e05af06c013c82222959be5d85100c3633` |
| Binario Go 1.26.5, dos builds | — | `6153f03a93c0a2618fdaf922443004244aa3bec7cbe9074466b22935c693edd0` |

G4 404 y G5 507 superan sus objetivos orientativos, pero esta revisión los
autoriza expresamente y ambos permanecen por debajo de sus paradas 420 y 540.
El runner conserva 800 y el manifiesto nueve entradas.

## Mutación y evidencia

M1--M25 conservan el contrato publicado. La enmienda añade o redefine:

- M26: sustituye la relación L0/L1 completa por `true` y muere por «tupla O1b
  imposible aceptada»;
- M27--M29: retiran clase, límite fijo o coherencia de error y mueren por
  «lector incompatible»;
- M30--M31: retiran límite general o relación con longitud parcial y mueren por
  «contador superior al límite».

Los 31 deben aplicar un patrón único, compilar y morir sin falsos positivos.
El AST ligado al futuro SHA material fija G2/G3, los dos call-sites, ausencia
de segunda gramática, máquina, limpieza y prohibiciones productivas.

## Dictámenes independientes

La revisión funcional reprodujo casos legítimos, seis negativos, M26, M30,
M31, hashes y binarios Go 1.26.5. Después de las correcciones emitió `GO`,
`P0=P1=P2=0` sobre la versión de 396 líneas y SHA
`92d7bbe26d8e68419d209c43c7eb618370c9816e8d1c49f3249640c0680aa813`.

La revisión de seguridad contrastó amenaza, frontera G2, call-sites, AST,
mutantes, write-sets, ledger y secuencia. Reprodujo M26 y el par G4 base/G5
final con sus stderr completos y emitió `GO`, `P0=P1=P2=0` sobre la misma
versión.

## Autorización limitada

Tras contrarrevisar el par documental exacto, se autoriza únicamente:

1. confirmar y publicar enmienda y acta;
2. esperar la CI de cinco puertas completamente verde;
3. avanzar la candidata por fast-forward sin perder sus cuatro rutas;
4. aplicar la corrección exacta de G4/G5 y tres hashes del runner;
5. ejecutar todas las puertas, H0 y 31 mutantes antes del commit material.

El código aún necesitará un commit material y otro de evidencia ligado a su
SHA, reproducción desde árbol limpio, doble revisión final, integración,
actualización transversal, publicación y CI 5/5. Un cambio material invalida
la evidencia.

No cambia ninguna métrica: F0 `10/23`, O4-05 `3/5`, Contratación temporal
`24/46`, Bolsa productiva `1/14` y producción `NO-GO`.
