# CT-O2-AUTH-A-VINCULO-CAPSULA-CANAL

Fecha: 31 de agosto de 2026.

Estado: **CANDIDATO CORREGIDO R3** para revisión independiente. No integrado,
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

R3 no cambia ese runtime. El blob Git de
`internal/app/composicion/interna/identidad.go` permanece exactamente en:

```text
4bcc3c7bc6a36078639cebac9e604204f16ed016
```

## NO-GO R2 corregido

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

## Análisis R3 tipado y de flujo

R3 combina `go/parser` y `go/types` con `Types`, `Defs`, `Uses`, `Instances` y
`Selections`. Calcula la procedencia hasta punto fijo entre objetos y
expresiones y la conserva a través de:

- parámetros de tipo, restricciones, genéricos, interfaces y conversiones;
- resultados de llamada, receptores, métodos y selectores;
- variables y aliases locales, declaraciones y asignaciones;
- índices, arrays, slices, maps y literales compuestos; y
- callbacks, llamadas diferidas, goroutines y capturas de clausuras.

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

## Matriz automática R3

Cada mutante debe completar primero parseo, importación y tipado. Solo después
se acepta como muerto si devuelve un error propio de la frontera. La matriz
contiene diecisiete casos:

1. los ocho mutantes anteriores: helper con retorno directo, alias de tipo,
   callback, clausura por captura, struct anónimo, slice `any`, retorno `any` y
   almacenamiento de paquete;
2. los ocho escapes R2 enumerados arriba; y
3. el callback de método diferido ya rechazado por R2.

Los diecisiete se tipan y se rechazan. Un homónimo inocuo, incluido su paso por
una función genérica sin canal, se tipa y pasa para controlar falsos positivos.
La fuente real `identidad.go` también se tipa y pasa.

## Invariantes funcionales conservadas

- éxito y extracción posterior únicamente por la instancia emisora;
- conservación de valores, deadline y cancelación del padre;
- rechazo de fachada/contexto nulos, canal ausente y aserción inválida;
- rechazo de sesión revocada y de cápsula previa propia o ajena;
- un único ganador entre contendientes y frente a `Autenticar`;
- cancelación previa sin consumo y cancelación durante verificación sin alta;
- ningún resultado utilizable ante error; y
- regresión completa de `Autenticar`.

## Evidencia local R3

Con `GOTOOLCHAIN=go1.26.5` y `GOPROXY=off`:

- `gofmt` de los dos ficheros Go: verde;
- paquete `./internal/app/composicion/interna` como UID 10001: verde;
- fuente real, cincuenta repeticiones: `50/50` verdes;
- matriz de diecisiete mutantes, cincuenta repeticiones: `850/850` muertos;
- pruebas funcionales `AutenticarYVincular`, cincuenta repeticiones: verdes;
- las mismas pruebas con detector de carreras, cinco repeticiones: verdes;
- `go vet ./internal/app/composicion/interna`: verde;
- `git diff --check`: verde;
- blob Git inmutable de `identidad.go`: el valor exigido; y
- Gitleaks acumulado sobre el cambio exacto: cero hallazgos con
  `/tmp/vec-gitleaks-20260831`, cuya huella SHA-256 es
  `c100de843d374f76143b03487de20fe341fb20cae8a71b6fdff896aec561391d`.

La ejecución del paquete usa UID/GID no privilegiados y cachés efímeras porque
las pruebas TLS rechazan deliberadamente UID 0. Los resultados exactos y el
hash del único commit se entregan al solicitar revisión.

## Límites

Este corte no lee HTTP, cabeceras, cookies, cuerpos, almacenamiento web ni
configuración de perfil. No selecciona perfil u organización, no invoca
ContextoActor ni PDP, no registra rutas y no hace alcanzable o arrancable la
raíz real. No sustituye autoridades corporativas, conformidades, integración,
PostgreSQL, Docker, E2E o revisión independiente.

Revisión independiente: **PENDIENTE** sobre el hash exacto del candidato R3.
