# Enmienda O2b: rechazo de progreso nuevo incompatible con L0

Fecha: 9 de agosto de 2026.

Estado: **doble `GO` documental final, `P0=P1=P2=0`**. La edición material
solo se reanuda después de confirmar y publicar esta enmienda con su acta, de
comprobar verde la CI de ese padre exacto y de avanzar la candidata por
*fast-forward* sin alterar sus cuatro rutas pendientes.

## Motivo y parada

La revisión independiente del candidato O2b detenido reprodujo una tupla que
la barrera defensiva de G4 acepta aunque contradice el contrato físico O1b:

```text
lector posterior = L0 / lectorAbiertoVacioM38
sufijo entregado = {'x'}
consumidos = 1
fin = false
trama = cero
resultado = lecturaNecesitaDatosM38
error = nil
```

Consumir un byte nuevo sin completar una trama debe dejar una parcial L1. L0
solo es posible si no se consumió ningún byte nuevo. La implementación actual
comprueba que el lector continúe activo, pero no liga el contador al estado
posterior. Por ello la tupla anterior devuelve `true` y la autoprueba vigente
no la rechaza.

G2 no produce hoy esa combinación. Esto no rebaja el hallazgo: G4 es la
barrera propietaria que convierte una devolución imposible de O1b en fallo
interno antes de Bash. Aceptarla permitiría que una regresión futura de G2 se
presentara como progreso válido y ocultara una parcial.

La parada se produce antes de H0, commit o publicación del candidato. O2b
permanece en `NO-GO`, con `P0=0`, `P1=1`, `P2=0`, hasta integrar y revisar la
corrección.

## Prevalencia limitada

Esta enmienda sustituye únicamente la invariancia byte a byte, las huellas y
los conteos exactos G4 `399 -> 400` y G5 `490 -> 491`, además de las tres
huellas correspondientes del runner, fijados en:

- `enmienda_f0_h0b_c4b2_g2o_o2b_ledger_runner_800_2026-08-05.md`;
- `enmienda_f0_h0b_c4b2_g2o_o2b_separacion_g4_g5_2026-08-05.md`.

No modifica la API, máquina S1--S5, gramática, precedencias, propiedad,
limpieza sensible, límites, imports, manifiesto, G1, G2, G3, Docker,
PostgreSQL o prohibiciones del contrato O2b.

## Corrección productiva exacta

En G4, dentro de `tuplaLectorValidaPreinicioM38`, la rama
`lecturaNecesitaDatosM38` cambia exactamente de:

```go
return !fin && n == len(sufijo) && tramaCeroM38(trama) && lectorActivoPreinicioM38(l)
```

a:

```go
return !fin && n == len(sufijo) && tramaCeroM38(trama) && lectorActivoPreinicioM38(l) &&
	(n == 0 || l.estado == lectorAbiertoParcialM38)
```

La condición conserva los únicos casos válidos:

| Bytes nuevos | Estado posterior | Decisión |
| ---: | --- | --- |
| `0` | L0 | aceptar |
| `0` | L1 previa | aceptar |
| `>0` | L1 | aceptar |
| `>0` | L0 | rechazar |

El resto de la rama continúa exigiendo `fin=false`, contador igual al sufijo,
trama cero y lector activo. No se reconstruye el estado previo ni se amplía
la autoridad de G4.

## Regresión exacta

En G5, `probarTuplasLectorPreinicioM38` añade, inmediatamente después de la
primera condición del `if` existente, esta única condición adversarial:

```go
tuplaLectorValidaPreinicioM38(l0, []byte{'x'}, false, tramaM38{}, 1, lecturaNecesitaDatosM38, nil) ||
```

La copia original con esta regresión debe compilar y fallar con:

```text
tupla O1b imposible aceptada
```

La copia corregida debe compilar y superar la autoprueba. Un mutante que
retire solo `(n == 0 || l.estado == lectorAbiertoParcialM38)` debe compilar y
morir por la misma regresión. Un fallo de compilación no cuenta como mutante
muerto.

## Write-set exacto posterior a la autorización

### Commit material

Solo podrán modificarse:

```text
deploy/postgresql/autorizacion_atestada_v3/
  probar_fuente_corporativa_contexto_actor_v1_pg18_4.sh
deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/
  supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_control_preinicio.go
  supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_control_preinicio_pruebas.go
```

El runner cambia exclusivamente tres literales SHA-256 ya existentes: G4,
G5 y binario compuesto. No cambia rutas, comparaciones, fuentes, bucles,
órdenes, manifiesto ni las dos reagrupaciones que mantienen el runner en 800
líneas.

G1 ya contiene las tres líneas O2b autorizadas y no se modifica de nuevo. G2,
G3 y todo el resto del árbol permanecen byte a byte.

### Commit de evidencia ligado al candidato

Después del commit material, y sin cambiar de nuevo ninguna fuente, se crean
únicamente:

```text
docs/portal_vec/revisiones/evidencias/
  f0_h0b_c4b2_g2o_o2b_ast_<sha7_codigo>.go.txt
  f0_h0b_c4b2_g2o_o2b_mutantes_<sha7_codigo>.sh.txt
```

`<sha7_codigo>` se sustituye por los siete primeros caracteres del commit
material. Los artefactos registran el SHA completo del código, las huellas de
las cinco fuentes y sus propias comprobaciones de aplicabilidad. Se ejecutan
desde el commit de evidencia, cuyo padre contiene exactamente el código
referenciado. Un cambio posterior en G1--G5 o runner invalida ambos y obliga
a regenerarlos con un nuevo commit material.

## Ledger y huellas exactos

| Unidad | Líneas | SHA-256 posterior |
| --- | ---: | --- |
| Runner | 800 | `18ce9e3940bda2e3239696b3ffbbe5532bc20adc27ddc24e4c651f936417f156` |
| G1 | 692 | `6b7f93b8b43c1040cc4ae2b6322c4e99e914eee415475e3fd50bf294b5a17afb` |
| G4 | 400 | `d2592b4b123aa99d0f6d9537357f9d864242210f5217567274f992b46515973e` |
| G5 | 491 | `c45d87c1c26167c672d5575c3e66785f09312ace4896afc6149122ac29003514` |
| G2 | 798 | `01acb818e9abefcbfe4c279bb0dd5e3317bf03f082f1ed3fba4f257c5642866b` |
| G3 | 431 | `d608868ecb2cb753876f488b522975e05af06c013c82222959be5d85100c3633` |
| Binario compuesto | — | `fe539cc83675f1cd8f6c6cffaf7e8941676b9da2525c24f93c11ad6697fefc21` |

La huella de partida del runner detenido es
`51439be40d3647568d0d994d334e3b6e3877e2ce9327b48e1502737a7f58ef81`.
Sustituir exclusivamente los tres literales autorizados produce el runner
final de la tabla; cualquier otra transformación queda prohibida.

Las huellas se han reproducido con Go 1.26.5 acreditado, `GOAMD64=v1`,
`CGO_ENABLED=0`, entorno aislado y dos builds `-a -trimpath`. El binario se
construye desde el módulo del repositorio para conservar el mismo
`DefaultGODEBUG` que el runner; compilar las fuentes desde un directorio sin
su módulo no es una reproducción válida de esa huella.

El manifiesto conserva exactamente nueve entradas y su mismo orden: D2c,
H0b, adaptador M38, D2d, G1, G4, G5, G2 y G3.

## Evidencia obligatoria

La línea base positiva y cada mutante deben compilar las cinco fuentes
`G1+G4+G5+G2+G3`. El conjunto final exige:

1. los 25 mutantes numerados y definidos en el contrato O2b publicado;
2. un mutante número 26 que elimine únicamente la nueva relación contador/L1;
3. exactamente 26 mutantes compilables, aplicados una vez y muertos por
   autoprueba o AST;
4. cero falsos muertos por patrón ausente, omisión de G5, fallo de build o
   error anterior al caso mutado;
5. artefactos AST y mutador autocontenidos, portables y ligados al SHA exacto
   del commit material;
6. ejecución desde un worktree limpio del commit de evidencia, comprobando
   que su padre es ese SHA material exacto y sin rutas privadas en los
   artefactos publicados.

La revisión final comprobará de forma explícita el original vulnerable, la
corrección, el mutante de la condición nueva y los tres casos legítimos L0
vacío, L1 previa vacía y L1 creada por bytes nuevos.

## Puertas que se repiten

- `gofmt` y `go vet` de las cinco fuentes;
- dos builds privados reproducibles con la huella exacta indicada;
- autoprueba completa O1a/O1b/O2a/O2b;
- AST y los 26 mutantes sin supervivientes ni falsos muertos;
- modos `--supervisar-m38` y desconocido en 64;
- `wc -l`, `bash -n`, ShellCheck y manifiesto de nueve;
- acreditación de huellas antes y después de G1/G4/G5/G2/G3;
- `git diff --check` y Gitleaks;
- pruebas globales, carrera y `go vet` global aplicables;
- `scripts/verificar_calidad.sh` completo;
- H0 PostgreSQL 18.4 sobre imagen fijada por digest y residuos cero;
- dos revisiones independientes del par exacto de commits material/evidencia,
  con
  `P0=P1=P2=0`.

Ninguna puerta previa se hereda como sustituto de estas repeticiones.

## Secuencia de reanudación

1. Dos revisores independientes emiten `GO` documental sin hallazgos.
2. Dirección actualiza únicamente el estado de esta enmienda y crea el acta.
3. Ambos revisores contracomprueban las huellas finales del documento y acta.
4. Se confirman y publican solo ambos documentos.
5. La CI de cinco puertas termina verde sobre ese padre exacto.
6. La rama candidata avanza por *fast-forward*, conservando exactamente sus
   cuatro rutas materiales sin confirmar.
7. Se aplican las dos líneas Go, se sustituyen los tres SHA del runner y se
   repiten las puertas locales, H0 y el mutador transitorio completo.
8. Con esas puertas verdes se crea un commit exclusivamente material; todavía
   no se integra ni se publica como cierre.
9. Sobre ese padre se crean los dos artefactos portables ligados al SHA del
   código, se confirman en un segundo commit y se reproducen AST, 26 mutantes,
   builds, H0 y puertas globales desde el nuevo árbol limpio.
10. Dos revisores independientes auditan el par exacto de commits. Cualquier
    cambio material invalida la evidencia y reinicia desde el paso 7.
11. Solo con doble `GO`, `P0=P1=P2=0`, dirección integra ambos commits,
    actualiza acta, tablero y relevo, publica el corte y exige la CI final de
    cinco puertas completamente verde.

## Paradas

Se detiene sin confirmar si cambia otra línea, una huella difiere, el runner
deja de tener 800 líneas, el manifiesto deja de tener nueve entradas, un
mutante sobrevive, aparece un falso muerto, H0 deja residuos o una revisión
emite `NO-GO`.

## Efecto limitado

Esta enmienda corrige una barrera interna; no cierra O2b por sí sola ni cambia
F0 `10/23`, O4-05 `3/5`, Contratación temporal `24/46`, Bolsa productiva
`1/14` o el `NO-GO` de producción.
