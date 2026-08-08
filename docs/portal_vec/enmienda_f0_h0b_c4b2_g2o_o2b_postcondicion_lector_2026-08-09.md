# Enmienda O2b: postcondición estructural bajo autoridad O1b fijada

Fecha: 9 de agosto de 2026.

Estado: **doble `GO` documental final, `P0=P1=P2=0`**. La edición material
solo se reanuda después de confirmar y publicar esta enmienda con su acta, de
comprobar verde la CI de ese padre exacto y de avanzar la candidata por
*fast-forward* sin alterar sus cuatro rutas pendientes.

## Motivo y clasificación

Las revisiones independientes posteriores a la primera corrección de O2b
reprodujeron seis tuplas que la barrera `tuplaLectorValidaPreinicioM38` acepta
aunque no pueden proceder del lector O1b contratado:

```text
trama válida + estado L0 + error interno no nulo
EOF limpio + estado L3 + error interno no nulo
trama válida + lector de clase distinta de CONTROL
EOF limpio + lector con límite distinto de 1024
necesita datos + n=1024 + L1 de longitud 1
trama válida + n=1025 con límite CONTROL 1024
```

La causa común es una postcondición base incompleta. G4 valida contador,
resultado, trama y parte del estado, pero solo compara el error pegajoso en la
rama que devuelve un fallo, no revalida la identidad física del lector en
todas las ramas y no liga el contador al límite ni a la longitud parcial
posterior observables.

G2 no genera estas combinaciones. No existe evidencia de fuga de
datos, acceso indebido o exposición de credenciales. El hallazgo sigue siendo
`P1`: G4 es la barrera propietaria que debe convertir cualquier devolución
que contradiga las postcondiciones estructurales definidas en fallo interno
antes de que una fase posterior pueda interpretarla como progreso,
cancelación o éxito.

El candidato queda detenido sin commit material. O2b continúa en `NO-GO`, con
`P0=0`, `P1=1`, `P2=0`, hasta que esta enmienda obtenga doble `GO`, se publique
sobre CI verde y la corrección supere toda la evidencia final.

## Prevalencia limitada

Esta enmienda sustituye únicamente en
`enmienda_f0_h0b_c4b2_g2o_o2b_correccion_tupla_l0_2026-08-09.md`:

- las huellas y conteos finales de runner, G4, G5 y binario;
- la regresión final de `probarTuplasLectorPreinicioM38`;
- la autoridad de mutación, que pasa de 26 a 31 mutantes;
- las comprobaciones AST de la postcondición del lector.

La relación contador/L0-L1 ya aprobada se conserva. Tampoco cambian API,
máquina S1--S5, gramática, precedencias, propiedad, limpieza sensible,
imports, manifiesto, G1, G2, G3, Docker, PostgreSQL o prohibiciones O2b.

La huella y el conteo del runner previamente proyectados dejan de ser finales,
pero constituyen la base exacta de esta corrección. G4 pasa expresamente de
400 a 404 líneas y G5 de 491 a 507.

## Corrección productiva exacta

En G4, al inicio de `tuplaLectorValidaPreinicioM38`, el bloque:

```go
if l == nil || n < 0 || n > len(sufijo) {
	return false
}
if fallo != nil {
	return n == 0 && lectura == lecturaNecesitaDatosM38 && tramaCeroM38(trama) &&
		l.estado == lectorErrorTerminalM38 && l.err == fallo && lectorLimpioM38(l)
}
```

se sustituye exactamente por:

```go
if l == nil || l.clase != "CONTROL" || l.limite != 1024 ||
	n < 0 || n > len(sufijo) || n > l.limite {
	return false
}
if fallo != nil {
	return n == 0 && lectura == lecturaNecesitaDatosM38 && tramaCeroM38(trama) &&
		l.estado == lectorErrorTerminalM38 && l.err == fallo && lectorLimpioM38(l)
}
if l.err != nil {
	return false
}
```

En el `switch` posterior solo se completa la relación física aprobada para
`lecturaNecesitaDatosM38`. La condición:

```go
return !fin && n == len(sufijo) && tramaCeroM38(trama) && lectorActivoPreinicioM38(l) &&
	(n == 0 || l.estado == lectorAbiertoParcialM38)
```

se sustituye exactamente por:

```go
return !fin && n == len(sufijo) && tramaCeroM38(trama) && lectorActivoPreinicioM38(l) &&
	(n == 0 || l.estado == lectorAbiertoParcialM38 && n <= l.longitud)
```

El resto del `switch` no cambia.

La decisión queda definida para toda devolución:

| Propiedad | Requisito |
| --- | --- |
| Identidad | clase exacta `CONTROL` y límite exacto `1024` |
| Contador general | `0 <= n <= len(sufijo)` y `n <= limite` |
| Fallo devuelto no nulo | estado L4, error pegajoso idéntico y lector limpio |
| Fallo devuelto nulo | error pegajoso necesariamente nulo |
| Necesita datos | si hay progreso, L1 y `n <= longitud` posterior |
| Trama o EOF limpio | estado, limpieza, contador y trama contratados |

No se normaliza ni oculta un error, no se acepta otra clase o límite y no se
añade autoridad al controlador.

## Frontera de autoridad: una sola gramática

La entrada exterior controla únicamente `fragmento` y `fin`. La tupla no se
deserializa, recibe ni inyecta desde otra frontera: nace de exactamente dos
sitios directos y tipados que llaman al mismo método G2 dentro del paquete y
binario privado. Son el consumo normal `(sufijo, fin)` y el drenaje EOF
`(nil, true)`; no existe otro sitio, interfaz, fábrica, variable funcional o
mock productivo.

O1b/G2 permanece como autoridad única de:

- correspondencia entre bytes, buffer parcial y trama devuelta;
- gramática, terminador, fragmentación y coalescencia;
- precedencia y causalidad de los errores físicos;
- limpieza del buffer realizada por el lector.

G4 no recodifica tramas, no repite el autómata de O1b, no copia fragmentos ni
buffers y no intenta reconstruir los bytes previos que G2 ya limpió. La
correspondencia byte a byte entre sufijo y contenido, y la causalidad concreta
entre entrada y uno de los cuatro errores normalizables, quedan expresamente
fuera de `tuplaLectorValidaPreinicioM38`. Se acreditan mediante las pruebas y
mutantes reales de O1a/O1b.

Esta frontera evita una segunda gramática y nuevas copias transitorias de
nonce, PID y causas. No reduce la denegación segura porque G2 está fijado por:

```text
ruta: supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_operativo.go
líneas: 798
SHA-256: 01acb818e9abefcbfe4c279bb0dd5e3317bf03f082f1ed3fba4f257c5642866b
índice del manifiesto de nueve: 7
```

El runner compara esa huella antes y después del snapshot y compila
exactamente G1+G4+G5+G2+G3. Cualquier cambio en G2 impide construir o ejecutar
O2b y exige revisar conjuntamente O1b y O2b; actualizar solo el literal SHA
queda prohibido.

Si O1b pasa en el futuro a otro proceso, plugin o conector intercambiable, esta
frontera deja de ser válida. En ese caso se rediseñará su API para aportar un
resultado atestado; no se añadirá un segundo parser ad hoc en G4.

### Alternativas descartadas

Se evaluó copiar los 4096 bytes del lector antes de cada consumo, repetir O1b
sobre la copia y recodificar cada trama para cotejarla. Se descarta porque
duplica trabajo y datos sensibles en cada iteración, rebasa la parada física de
G4 y vuelve a ejecutar la misma autoridad, sin crear independencia frente a un
fallo de G2. Una corrupción del proceso o toolchain afectaría a ambas
ejecuciones; una evolución legítima tendría dos recorridos que mantener.

También se descarta comparar solo la cola parcial: daría una garantía
relacional incompleta y engañosa, pues no cubriría la trama ya limpiada ni la
precedencia de errores. La protección compensatoria correcta es fijar y probar
la única autoridad de bytes, y detener ante cualquier cambio de su SHA.

## Regresión exacta

En G5, dentro de `probarTuplasLectorPreinicioM38`, después del primer bloque
negativo ya existente y antes del bucle de errores normalizables, se añade:

```go
l0Error := &lectorTramaM38{clase: "CONTROL", limite: 1024, err: errInvarianteControlPreinicioM38}
l0Clase := &lectorTramaM38{clase: "OTRA", limite: 1024}
l3Limite := &lectorTramaM38{clase: "CONTROL", limite: 2048, estado: lectorEOFLimpioM38}
if tuplaLectorValidaPreinicioM38(l0Error, nil, false, tramaM38{}, 0, lecturaNecesitaDatosM38, nil) ||
	tuplaLectorValidaPreinicioM38(l0Error, []byte{'x'}, false, valida, 1, lecturaTramaM38, nil) ||
	tuplaLectorValidaPreinicioM38(l0Clase, []byte{'x'}, false, valida, 1, lecturaTramaM38, nil) ||
	tuplaLectorValidaPreinicioM38(l3Limite, nil, true, tramaM38{}, 0, lecturaEOFLimpioM38, nil) {
	return errors.New("tupla O1b con lector incompatible aceptada")
}
```

La copia corregida debe compilar y superar la autoprueba. G4 base de la
corrección L0, 400 líneas y SHA
`d2592b4b123aa99d0f6d9537357f9d864242210f5217567274f992b46515973e`,
compilada con G5 final de 507 líneas y SHA
`10ccaf8347bfcaa5f3990b75b4c9becd62cd39b60249b628af6c7a1fc6bc8867`,
debe fallar con estado 65, `stdout` vacío y esta línea exacta en `stderr`:

```text
autoprueba del supervisor: control previo tuplas O1b: tupla O1b con lector incompatible aceptada
```

La revisión reproduce además tuplas legítimas L0 sin byte nuevo, L1 previa y
L1 creada por progreso nuevo.

Después de ese bloque se añaden los dos límites de contador:

```go
l1Contador := &lectorTramaM38{clase: "CONTROL", limite: 1024, estado: lectorAbiertoParcialM38, longitud: 1}
l1Contador.buffer[0] = 'x'
sufijoLimite := []byte(strings.Repeat("x", 1025))
if tuplaLectorValidaPreinicioM38(l1Contador, sufijoLimite[:1024], false, tramaM38{}, 1024, lecturaNecesitaDatosM38, nil) ||
	tuplaLectorValidaPreinicioM38(l0, sufijoLimite, false, valida, 1025, lecturaTramaM38, nil) {
	return errors.New("tupla O1b con contador superior al límite aceptada")
}
```

El primer caso acredita que una lectura sin trama no puede consumir más bytes
que los conservados en la parcial posterior. El segundo acredita que una sola
devolución CONTROL nunca supera su límite físico.

## Mutantes nuevos y autoridad durable

Permanecen los 25 identificadores enumerados en el contrato O2b. M26 conserva
su significado de retirar la relación contador/L0-L1, pero esta enmienda
sustituye su patrón de aplicación: reemplaza exactamente el subárbol único

```go
(n == 0 || l.estado == lectorAbiertoParcialM38 && n <= l.longitud)
```

de la rama necesita-datos por:

```go
true
```

El cambio focal deja el `&&` precedente seguido de `true`, por lo que debe
pasar `gofmt`, `go vet` y build antes de atribuirle la muerte. Se añaden
exactamente:

27. M27 elimina solo `l.clase != "CONTROL"` de la guarda;
28. M28 elimina solo `l.limite != 1024` de la guarda;
29. M29 elimina solo la guarda `if l.err != nil { return false }`.
30. M30 elimina solo `n > l.limite` de la guarda común;
31. M31 elimina solo `n <= l.longitud` de la rama necesita-datos.

M1--M25 conservan sus transformaciones y oráculos del contrato O2b publicado.
Cada mutante M26--M31 debe aplicar un patrón único, pasar `gofmt`, `go vet` y
build de G1+G4+G5+G2+G3, y morir después con estado 65 por el oráculo exacto:

```text
autoprueba del supervisor: control previo tuplas O1b: tupla O1b imposible aceptada
autoprueba del supervisor: control previo tuplas O1b: tupla O1b con lector incompatible aceptada
autoprueba del supervisor: control previo tuplas O1b: tupla O1b con contador superior al límite aceptada
```

Un patrón ausente, múltiple, un fallo de formato, vet o compilación, o un error
anterior al oráculo no cuenta como mutante muerto. La ejecución válida devuelve
estado 65, `stdout` vacío y una de esas líneas completas exactas en `stderr`.
M26 debe morir por el primer oráculo, M27--M29 por el segundo y M30--M31 por el
tercero. El total final exigido es exactamente 31 mutantes compilables,
aplicados una vez y muertos.

## Control AST añadido

El artefacto O2b ligado al futuro SHA material debe comprobar, mediante AST y
tipado de G1+G4+G5+G2+G3:

1. la guarda común exige clase literal `CONTROL` y límite literal `1024` antes
   de cualquier rama de resultado;
2. toda devolución con `fallo == nil` rechaza un `l.err` no nulo;
3. la rama con fallo exige identidad exacta entre `l.err` y `fallo`, L4,
   contador cero, trama cero y limpieza;
4. toda rama impide `n > l.limite`;
5. necesita-datos con progreso exige L1 y `n <= l.longitud`;
6. M26 permanece incluido en esa relación exacta;
7. los modos M26--M31 relajan exclusivamente la fuente declarada y nunca las
   huellas invariantes de G2 o G3;
8. el build se ejecuta antes de atribuir una muerte al control AST.

Se mantienen todas las comprobaciones estructurales O2b anteriores: sets
exactos de declaraciones, imports, alcanzabilidad tipada, única construcción
CONTROL, única copia del nonce, cero exposición de ticket/identidad/sobre,
transiciones, causas, limpieza y ausencia de procesos, FD, red, reloj, Bash o
señales interpretadas.

El AST comprueba además exactamente dos sitios productivos directos y tipados a
`(*lectorTramaM38).consumir`: consumo normal y drenaje EOF. Comprueba ausencia
de cualquier otro sitio, interfaz, fábrica, función variable o mock productivo
para sustituir G2, y cero llamadas desde G4 a
`codificarTramaM38`, `decodificarTramaM38` o ayudantes equivalentes.

## Write-set exacto

### Commit material acumulado

El candidato ya conserva sin confirmar cuatro rutas O2b: runner, G1, G4 y G5.
La corrección nueva solo vuelve a editar estas tres:

```text
deploy/postgresql/autorizacion_atestada_v3/
  probar_fuente_corporativa_contexto_actor_v1_pg18_4.sh
deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/
  supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_control_preinicio.go
  supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_control_preinicio_pruebas.go
```

El runner cambia exclusivamente los literales SHA-256 de G4, G5 y binario.
G1 no recibe una edición nueva, pero sus tres líneas pendientes forman parte
del commit material acumulado. G2, G3 y el resto permanecen byte a byte.

### Commit de evidencia

Después del commit material se crean únicamente:

```text
docs/portal_vec/revisiones/evidencias/
  f0_h0b_c4b2_g2o_o2b_ast_<sha7_codigo>.go.txt
  f0_h0b_c4b2_g2o_o2b_mutantes_<sha7_codigo>.sh.txt
```

Los artefactos registran el SHA material completo y las huellas de las cinco
fuentes. El commit de evidencia tiene como padre exacto el commit material.
Un cambio posterior de código los invalida.

## Ledger y huellas exactos

| Unidad | Base corregida L0 | Resultado final | Líneas finales |
| --- | --- | --- | ---: |
| Runner | `18ce9e3940bda2e3239696b3ffbbe5532bc20adc27ddc24e4c651f936417f156` | `8e15443b120dc68721aa4cc0959610ca393af44d842f0e07ed7e0b18873fc059` | 800 |
| G1 | `6b7f93b8b43c1040cc4ae2b6322c4e99e914eee415475e3fd50bf294b5a17afb` | igual | 692 |
| G4 | `d2592b4b123aa99d0f6d9537357f9d864242210f5217567274f992b46515973e` | `2befe2a4c16fc7a57aacd421ea6c8419ab49160bb2ae0d0eb6f03786194aa744` | 404 |
| G5 | `c45d87c1c26167c672d5575c3e66785f09312ace4896afc6149122ac29003514` | `10ccaf8347bfcaa5f3990b75b4c9becd62cd39b60249b628af6c7a1fc6bc8867` | 507 |
| G2 | `01acb818e9abefcbfe4c279bb0dd5e3317bf03f082f1ed3fba4f257c5642866b` | igual | 798 |
| G3 | `d608868ecb2cb753876f488b522975e05af06c013c82222959be5d85100c3633` | igual | 431 |
| Binario | `fe539cc83675f1cd8f6c6cffaf7e8941676b9da2525c24f93c11ad6697fefc21` | `6153f03a93c0a2618fdaf922443004244aa3bec7cbe9074466b22935c693edd0` | — |

Las huellas se reprodujeron con Go 1.26.5, `GOAMD64=v1`, `CGO_ENABLED=0`,
`-a -trimpath` y dos entornos privados desde el módulo del repositorio. El
runner final se obtiene sustituyendo solo sus tres literales autorizados.

Esta enmienda revisa y autoriza expresamente que G4 supere el objetivo de 400
y quede en 404, por debajo de la parada 420. También revisa que G5 supere el
objetivo 500 y quede en 507, por debajo de la parada 540. Cualquier nueva línea
exige revisar de nuevo el ledger. El manifiesto mantiene exactamente nueve
entradas y su orden.

## Puertas obligatorias

- `gofmt` y `go vet` de G1+G4+G5+G2+G3;
- dos builds privados reproducibles con la huella final exacta;
- autoprueba completa y matriz de seis negativos más casos legítimos;
- AST portable y 31/31 mutantes sin falsos muertos;
- autoprueba real O1a/O1b de fragmentación, cola, LF, bytes inválidos,
  1023/1024/1025, coalescencia, EOF y cuatro errores normalizables;
- comprobación de que la evidencia histórica O1b corresponde al mismo G2 de
  798 líneas y SHA
  `01acb818e9abefcbfe4c279bb0dd5e3317bf03f082f1ed3fba4f257c5642866b`;
- modos `--supervisar-m38` y desconocido en 64;
- conteos, hashes y manifiesto de nueve antes y después;
- `bash -n`, ShellCheck, `git diff --check` y Gitleaks;
- pruebas globales, carrera, `go vet` global y calidad completa;
- H0 PostgreSQL 18.4 por digest y residuos cero;
- doble revisión independiente del par material/evidencia con
  `P0=P1=P2=0`;
- integración, publicación y CI final de cinco puertas verdes.

Ninguna prueba focal sustituye las puertas completas.

## Secuencia y paradas

1. Dos revisores independientes emiten `GO` documental.
2. Dirección cambia solo el estado, crea el acta y obtiene contrarrevisión del
   par exacto.
3. Se confirman y publican solo enmienda y acta; la CI debe terminar 5/5 verde.
4. La candidata avanza por fast-forward conservando sus cuatro rutas.
5. Se aplican la corrección G4, regresión G5 y tres hashes del runner.
6. Se ejecutan todas las puertas, H0 y los 31 mutantes transitorios.
7. Solo en verde se confirma el commit material; todavía no se integra.
8. Se crean y confirman en un segundo commit los artefactos ligados a su padre.
9. Desde un árbol limpio se reproducen evidencia y puertas completas.
10. Dos revisores auditan el par exacto. Cualquier cambio material reinicia
    desde el paso 5.
11. Solo con doble `GO` se integra, actualiza el estado, publica y exige CI 5/5.

Se detiene sin confirmar si cambia otra ruta, difiere una huella, el runner no
queda en 800, G4 supera 404, G5 supera 507, cambia el manifiesto, sobrevive un
mutante, aparece un falso muerto, H0 deja residuos o una revisión emite
`NO-GO`.

## Efecto limitado

Esta corrección solo completa la barrera defensiva O2b. No cierra O2b por sí
sola ni cambia F0 `10/23`, O4-05 `3/5`, Contratación temporal `24/46`, Bolsa
productiva `1/14` o el `NO-GO` de producción.
