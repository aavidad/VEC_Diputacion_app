# CT-O2-AUTH-A-VINCULO-CAPSULA-CANAL

Fecha: 31 de agosto de 2026.

Estado: **CANDIDATO CORREGIDO R4** para revisión independiente. No integrado,
no publicado, no autoaprobado y no apto para producción ni datos reales.

## Capacidad y runtime inmutable

`FachadaIdentidadOffline.AutenticarYVincular` consume la capacidad de un solo
uso emitida por el canal TLS interno, autentica o revalida la sesión mediante
la única instancia configurada de `ServicioIdentidad`, proyecta la cápsula
opaca y la liga inmediatamente al `context.Context` de esa misma petición.

La API devuelve exclusivamente el contexto derivado. No devuelve ni registra
cuenta, perfil, organización, sesión, canal, aserción o cápsula y no concede
autoridad funcional. ContextoActor y PDP siguen siendo autoridades posteriores
obligatorias.

R4 no cambia ese runtime. El blob Git de
`internal/app/composicion/interna/identidad.go` permanece exactamente en:

```text
4bcc3c7bc6a36078639cebac9e604204f16ed016
```

## Historia de NO-GO R2 y R3

R2 recibió `NO-GO`, `P0=0`, `P1=3`, `P2=1`. El analizador tipado anterior
conservaba el tipo inmediato, pero no la procedencia del canal después de
ciertas operaciones. Aceptó indebidamente:

- getter de receptor devuelto y getter almacenado;
- helper genérico convertido a `any` e interfaz genérica diferida;
- índice de array tipado y alias local;
- clausura con el selector separado de la captura; y
- `map` posterior a una conversión genérica.

El callback de método diferido sí era rechazado en R2 y permanece en la matriz
para impedir una regresión. Ninguna evidencia del candidato rechazado se
hereda como cierre.

R3 recibió después `NO-GO`, `P0=1`, `P1=2`, `P2=1`. El analizador todavía
permitía enviar el canal autenticado a un `chan any`, recibirlo más tarde y
retornarlo: no inspeccionaba `ast.SendStmt`, de modo que la procedencia se
perdía antes de la recepción. Además, un `types.Named` externo terminaba el
recorrido antes de visitar sus argumentos de tipo; por ello
`atomic.Pointer[CanalProxyAutenticado]` no quedaba marcado. La matriz R3 no
acreditaba las construcciones adicionales exigidas por la revisión. R4 no
hereda sus puertas como cierre.

## Análisis R4 tipado y de flujo

R4 combina `go/parser` y `go/types` con `Types`, `Defs`, `Uses`, `Instances` y
`Selections`. Calcula la procedencia hasta punto fijo entre objetos y
expresiones y la conserva a través de:

- parámetros de tipo, restricciones, genéricos internos o externos,
  interfaces y conversiones;
- resultados de llamada, receptores, métodos y selectores;
- variables y aliases locales, declaraciones y asignaciones;
- índices, arrays, slices, maps, punteros y literales compuestos;
- `append`, `copy`, aserciones y cambios de tipo, expresiones de método y
  retornos múltiples; y
- callbacks, llamadas diferidas, goroutines, canales Go y capturas transitivas
  de clausuras.

El recorrido de un nominal conserva la prevención de ciclos, visita primero
sus argumentos de tipo y solo deja de inspeccionar el subyacente cuando la
autoridad del nominal es externa. Así un nominal externo sin canal permanece
inocuo, mientras que `atomic.Pointer[CanalProxyAutenticado]` conserva la marca
de su argumento. Un envío falla en el propio `ast.SendStmt` si su valor todavía
porta procedencia del canal; no depende del nombre de la variable, del tipo del
canal Go ni de que una recepción posterior vuelva a revelar el tipo.

Firmas, tipos, retornos, almacenamiento de paquete o compuesto y cualquier
paso por una frontera genérica o abierta fallan cerrados. Una asignación local
no limpia ni legitima la procedencia.

La única forma estructural de originar la variable inmediata de canal es el
resultado exacto `CanalProxyAutenticado` de una llamada externa concreta y no
genérica, sin otro canal como argumento o receptor. A partir de ahí el valor
solo puede cruzar parámetros o receptores cuyo tipo sea exactamente el del
canal. No existe una lista de nombres de funciones, variables o métodos que se
pueda reutilizar como atajo.

Así se admite el flujo real local autenticación→credencial→identidad→cápsula→
vínculo dentro de la misma operación, pero se rechazan aliases, getters,
contenedores, interfaces, callbacks y cierres que permitan diferirlo.

## Matriz automática R4

Cada mutante debe completar primero parseo, importación y tipado. Solo después
se acepta como muerto si devuelve un error propio de la frontera. La matriz
conserva los diecisiete casos R3 y añade trece mutantes de una sola dimensión:

1. los ocho mutantes anteriores: helper con retorno directo, alias de tipo,
   callback, clausura por captura, struct anónimo, slice `any`, retorno `any` y
   almacenamiento de paquete;
2. los ocho escapes R2 enumerados arriba; y
3. el callback de método diferido ya rechazado por R2;
4. puntero y genérico externo `atomic.Pointer`;
5. canal Go, `append`, `copy` y `go statement`;
6. `method expression`, tipo nominal, `type assertion` y `type switch`; y
7. retorno múltiple/tupla, genéricos anidados con `embed` y clausura
   transitiva.

Los treinta completan parseo, importación y tipado y mueren únicamente con un
error prefijado por `frontera: `. Los controles inocuos cubren homónimos,
genérico y `atomic.Pointer[int]`, canal Go de `int`, `append`, `copy`, aserción
y cambio de tipo sin canal. Todos se tipan y pasan. La fuente real
`identidad.go` también se tipa y pasa.

## Invariantes funcionales conservadas

- éxito y extracción posterior únicamente por la instancia emisora;
- conservación de valores, deadline y cancelación del padre;
- rechazo de fachada/contexto nulos, canal ausente y aserción inválida;
- rechazo de sesión revocada y de cápsula previa propia o ajena;
- un único ganador entre contendientes y frente a `Autenticar`;
- cancelación previa sin consumo y cancelación durante verificación sin alta;
- ningún resultado utilizable ante error; y
- regresión completa de `Autenticar`.

## Evidencia local R4

Con `GOTOOLCHAIN=go1.26.5` y `GOPROXY=off`:

- `gofmt` de los dos ficheros Go: verde;
- paquete `./internal/app/composicion/interna` como UID 10001 y sin red: verde;
- fuente real, cincuenta repeticiones: `50/50` verdes;
- matriz completa de treinta mutantes, veinte repeticiones: `600/600` muertos;
- pruebas funcionales `AutenticarYVincular`, veinte repeticiones: verdes;
- las mismas pruebas con detector de carreras, dos repeticiones: verdes;
- `go vet ./internal/app/composicion/interna`: verde;
- `git diff --check`: verde;
- blob Git inmutable de `identidad.go`: el valor exigido; y
- Gitleaks acumulado sobre el cambio exacto: cero hallazgos con
  `/tmp/vec-gitleaks-20260831`, cuya huella SHA-256 es
  `c100de843d374f76143b03487de20fe341fb20cae8a71b6fdff896aec561391d`.

Los tres ficheros del write-set permanecen bajo el tope duro de 800 líneas. La
ejecución del paquete usa UID/GID no privilegiados y cachés efímeras porque
las pruebas TLS rechazan deliberadamente UID 0. Los resultados exactos y el
hash del único commit se entregan al solicitar revisión.

## Límites

Este corte no lee HTTP, cabeceras, cookies, cuerpos, almacenamiento web ni
configuración de perfil. No selecciona perfil u organización, no invoca
ContextoActor ni PDP, no registra rutas y no hace alcanzable o arrancable la
raíz real. No sustituye autoridades corporativas, conformidades, integración,
PostgreSQL, Docker, E2E o revisión independiente.

Revisión independiente: **PENDIENTE** sobre el hash exacto del candidato R4.
