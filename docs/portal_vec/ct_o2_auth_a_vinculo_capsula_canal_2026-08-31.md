# CT-O2-AUTH-A-VINCULO-CAPSULA-CANAL

Fecha: 31 de agosto de 2026.

Estado: candidato local para revisión independiente. No integrado, no
publicado y no apto para producción ni datos reales.

## Capacidad

`FachadaIdentidadOffline.AutenticarYVincular` consume la capacidad de un solo
uso emitida por el canal TLS interno, autentica o revalida la sesión mediante
la única instancia configurada de `ServicioIdentidad`, proyecta la cápsula
opaca y la liga inmediatamente al `context.Context` de esa misma petición.

La API devuelve exclusivamente el contexto derivado. No devuelve ni registra
la cuenta, perfil, organización, sesión, canal, aserción o cápsula. Tampoco
concede autoridad funcional: el consumidor posterior todavía debe extraer la
cápsula con la misma instancia y ejecutar las autoridades ContextoActor y PDP
que correspondan.

## Implementación

La secuencia privada común de `Autenticar` conserva el
`CanalProxyAutenticado` validado junto a la cápsula. La API histórica mantiene
su firma y recibe solo la cápsula; la operación nueva entrega ambos valores
directamente a `VincularCapsulaIdentidadPeticion`, sin serialización, getter ni
tránsito por HTTP.

El orden es cerrado:

```text
capacidad one-shot de C4 en context.Context
→ consumo atómico por el propietario exacto
→ acreditación del handshake TLS 1.3
→ credencial desde la aserción protegida
→ resolución y alta/revalidación de sesión
→ proyección de cápsula ligada a servicio, sesión y canal
→ vinculación one-shot al mismo context.Context
→ contexto derivado o nil
```

Toda salida con error contiene un contexto nulo. La operación no devuelve el
contexto original como sustituto autorizado. La cancelación anterior se
observa antes de consumir el canal; una cancelación durante la autenticación,
una sesión revocada, un contexto ya vinculado o cualquier cruce terminan sin
un resultado utilizable.

## Invariantes acreditadas por pruebas

- éxito y extracción posterior únicamente por la instancia emisora;
- conservación de valores, deadline y propagación de cancelación del padre;
- rechazo de fachada o contexto nulos, canal ausente y aserción inválida;
- rechazo de sesión revocada y de contexto con cápsula previa propia o ajena;
- rechazo de extracción mediante otro `ServicioIdentidad`;
- un único ganador entre dieciséis contendientes y replay posterior denegado;
- cancelación previa sin consumir la capacidad y cancelación durante la
  verificación sin alta durable;
- regresión de `Autenticar`, incluida vinculación y extracción posterior.

## Límites deliberados

Este corte no lee HTTP, cabeceras, cookies, cuerpos, almacenamiento web ni
configuración de perfil. No selecciona perfil u organización, no invoca
ContextoActor ni PDP, no registra rutas y no hace alcanzable o arrancable la
raíz real. No sustituye las decisiones corporativas, las conformidades ni las
pruebas de integración posteriores.

La suite del paquete debe ejecutarse como runtime no privilegiado: sus pruebas
de material TLS rechazan deliberadamente el UID 0. La ejecución focal no
necesita ese cambio de identidad; la ejecución completa se reproduce como UID
10001 con cachés efímeras y el módulo Go local, siempre con
`GOTOOLCHAIN=go1.26.5` y `GOPROXY=off`.

## Evidencia local

- `gofmt` sobre los dos ficheros Go: verde;
- `GOTOOLCHAIN=go1.26.5 GOPROXY=off go test -count=1
  ./internal/app/composicion/interna`, como UID 10001: verde;
- prueba focal `TestFachadaIdentidadOfflineAutenticarYVincular`, veinte
  repeticiones: verde;
- la misma prueba focal con detector de carreras, cinco repeticiones: verde;
- `go vet ./internal/app/composicion/interna`: verde;
- `git diff --check`: verde;
- Gitleaks v8.30.1, desde la caché local, sobre el contenido exacto de los tres
  ficheros del corte: cero hallazgos.
