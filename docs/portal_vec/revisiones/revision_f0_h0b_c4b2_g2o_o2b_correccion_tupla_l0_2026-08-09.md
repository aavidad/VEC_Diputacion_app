# Revisión O2b: corrección de la tupla L0 con progreso nuevo

Fecha: 9 de agosto de 2026.

Dictamen: **GO documental final, `P0=0`, `P1=0`, `P2=0`**.

## Objeto

Se revisó íntegramente la enmienda:

```text
docs/portal_vec/enmienda_f0_h0b_c4b2_g2o_o2b_correccion_tupla_l0_2026-08-09.md
```

La versión normativa aprobada antes de actualizar únicamente su estado tenía
239 líneas y SHA-256:

```text
f918b9025ace08140c8958ae97dd255dda1027a62e02eb23d39621f11ef7a602
```

La versión con estado final y esta acta se someten a contrarrevisión de ambos
revisores antes del commit. El futuro commit publicado será el padre material
exacto de la reanudación del candidato O2b.

## Hallazgo material

Dos revisores reprodujeron que la barrera actual de G4 acepta como válida:

```text
L0 limpio + sufijo {'x'} + n=1 + fin=false
+ trama cero + lecturaNecesitaDatosM38 + error nil
```

La combinación contradice O1b: un byte nuevo sin trama completa debe dejar
una parcial L1. Aunque G2 no la emite actualmente, G4 debe rechazarla para que
una regresión futura no se convierta en progreso válido antes de Bash.

El candidato actual permanece en `NO-GO`, `P0=0`, `P1=1`, `P2=0`, hasta
aplicar la corrección y cerrar toda su evidencia. La aprobación documental no
convierte el prototipo detenido en código aceptado.

## Primera revisión y correcciones

La primera redacción de 203 líneas y SHA-256
`266b2b0207372009a9b513ad62578c2706562dc22a5ab28b75f11de9e679ced7`
recibió `NO-GO` de ambos revisores. Los hallazgos fueron:

1. se exigían 48 mutantes previos y un total de 49 sin autoridad durable,
   listado versionado ni artefacto reproducible;
2. el write-set material no autorizaba los artefactos AST y mutador que el
   propio contrato exigía publicar;
3. la secuencia pedía evidencia ligada al SHA material antes de crear ese
   commit;
4. no se fijaba la huella final exacta del runner ni su huella de partida;
5. la prevalencia no nombraba expresamente los nuevos conteos G4/G5;
6. faltaba exigir publicación y CI final después de integrar.

Dirección corrigió el documento sin tocar código:

- los 25 mutantes contractuales durables más un M26 exacto sustituyen la
  referencia temporal no reproducible;
- commit material y commit de evidencia tienen write-sets separados;
- la evidencia se crea después del SHA material y se ejecuta desde su commit,
  cuyo padre debe ser ese SHA exacto;
- la doble revisión audita el par de commits y cualquier cambio material
  invalida ambos artefactos;
- runner base/final y conteos G4/G5 quedan fijados;
- integración, publicación y CI de cinco puertas preceden al cierre.

## Revisiones independientes finales

La revisión funcional volvió a leer la versión completa de 239 líneas,
reprodujo el original vulnerable, la corrección, los tres casos legítimos y
M26. Comprobó que el mutante compila y muere con estado 65 y el mensaje exacto,
sin convertir un fallo de compilación en cobertura. Emitió `GO`,
`P0=P1=P2=0`.

La revisión de seguridad, trazabilidad y ledger comprobó la frontera
defensiva, la prevalencia, ambos write-sets, la secuencia no circular, los 25
identificadores contractuales más M26, las paradas y la publicación final.
Emitió `GO`, `P0=P1=P2=0`.

Ambas revisiones confirmaron que el candidato material continúa detenido y
que ningún resultado previo sustituye las puertas finales de código.

## Reproducción focal

| Unidad | Líneas | SHA-256 |
| --- | ---: | --- |
| Runner final proyectado | 800 | `18ce9e3940bda2e3239696b3ffbbe5532bc20adc27ddc24e4c651f936417f156` |
| G1 invariante | 692 | `6b7f93b8b43c1040cc4ae2b6322c4e99e914eee415475e3fd50bf294b5a17afb` |
| G4 corregido | 400 | `d2592b4b123aa99d0f6d9537357f9d864242210f5217567274f992b46515973e` |
| G5 con regresión | 491 | `c45d87c1c26167c672d5575c3e66785f09312ace4896afc6149122ac29003514` |
| G2 invariante | 798 | `01acb818e9abefcbfe4c279bb0dd5e3317bf03f082f1ed3fba4f257c5642866b` |
| G3 invariante | 431 | `d608868ecb2cb753876f488b522975e05af06c013c82222959be5d85100c3633` |
| Binario, dos builds | — | `fe539cc83675f1cd8f6c6cffaf7e8941676b9da2525c24f93c11ad6697fefc21` |

La huella del runner detenido antes de sustituir los tres literales es
`51439be40d3647568d0d994d334e3b6e3877e2ce9327b48e1502737a7f58ef81`.
La proyección final mantiene `bash -n`, ShellCheck, `go vet`, autoprueba y
modos 64/64 verdes. Los builds se reprodujeron con Go 1.26.5 acreditado,
entorno aislado, `GOAMD64=v1`, `CGO_ENABLED=0` y `-a -trimpath` desde el
módulo del repositorio.

## Autorización limitada

Este dictamen autoriza confirmar y publicar exclusivamente la enmienda y esta
acta. Solo después de CI verde puede avanzarse la candidata y aplicar:

1. la relación exacta entre contador y L0/L1 en G4;
2. la única regresión adversarial en G5;
3. los tres literales SHA-256 del runner.

G1 no recibe una nueva edición respecto del candidato detenido, pero sus tres
líneas pendientes forman parte del commit material junto con runner, G4 y G5.
G2, G3 y todo el resto permanecen byte a byte. Los dos artefactos de evidencia
ocupan un segundo commit ligado al SHA de código. O2b solo podrá integrarse
tras AST, 26/26 mutantes compilables y muertos, dos builds, H0 PostgreSQL 18.4,
residuos cero, puerta global, doble revisión del par exacto y CI final de cinco
puertas.

No cambia ninguna métrica: F0 `10/23`, O4-05 `3/5`, Contratación temporal
`24/46`, Bolsa productiva `1/14` y producción `NO-GO`.
