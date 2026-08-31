# CT-LITE-O5-01-HTTP-ASIGNACION — adaptador HTTP interno

Fecha: 31 de agosto de 2026.

Estado del productor: candidato técnico con puertas focales verdes; pendiente
de revisión independiente del hash exacto. Este documento no concede `GO`, no
cierra O5-01 y no autoriza integración, registro de rutas ni producción.

## Punto de partida acreditado

- Worktree exclusivo:
  `.worktrees/ct-lite-o5-01-http-asignacion-20260831`.
- Rama: `trabajo/ct-lite-o5-01-http-asignacion-20260831`.
- `HEAD` y base exacta antes de editar:
  `9137de57678c317546620503d27a9c30151d6249`.
- Referencia local `origin/integracion/ct-producto-ligero-20260821` antes de
  editar: `9137de57678c317546620503d27a9c30151d6249`.
- Árbol inicial: limpio.
- Toolchain de las puertas válidas: `go1.26.5 linux/amd64`, con
  `GOPROXY=off`.

## Capability, invariante y write-set

Capability: aportar, sin registrarlo, un adaptador HTTP interno para:

```text
POST /api/vec/contratacion-temporal/asignaciones
POST /api/vec/contratacion-temporal/reasignaciones
```

El adaptador delega respectivamente en `ServicioAsignacion.Asignar` y
`ServicioAsignacion.Reasignar` mediante una interfaz nominal mínima.

Invariante: autenticación, sesión, perfil y organización proceden únicamente
de `AutoridadContextoCanalAsignacion`. La autoridad no recibe la petición
HTTP. Cuerpo, URL y cabeceras solo pueden expresar intención funcional y no
pueden aportar identidad, sesión, perfil, organización, actor, rol o permiso.
Cada petición válida invoca una vez la autoridad y una vez exactamente una de
las dos operaciones del ejecutor. Antes de responder se cruza el recibo con la
operación, organización autoritativa, expediente, versión anterior y
resultante, unidad, responsable, referencias, sellos HMAC e instante
canónico.

Write-set exacto:

- `internal/modules/contrataciontemporal/adapters/httpinterno/asignacion.go`;
- `internal/modules/contrataciontemporal/adapters/httpinterno/asignacion_contrato.go`;
- `internal/modules/contrataciontemporal/adapters/httpinterno/asignacion_test.go`;
- `internal/modules/contrataciontemporal/adapters/httpinterno/asignacion_seguridad_test.go`;
- `docs/portal_vec/ct_lite_o5_01_http_asignacion_2026-08-31.md`.

## Contrato cerrado

La asignación admite solo:

```json
{
  "expediente_ref": "referencia opaca",
  "version_esperada": 1,
  "clave_idempotencia": "UUIDv4 canónica",
  "unidad_ref": "referencia opaca",
  "responsable_ref": "referencia opaca"
}
```

La reasignación añade obligatoriamente
`motivo_reasignacion_clave` y `observaciones`. El motivo conserva únicamente
la clave de catálogo; las observaciones se limitan a 1.000 caracteres Unicode
NFC, sin espacios periféricos ni controles no admitidos.

Controles aplicados antes de invocar autoridad o aplicación:

- ruta y componentes de URL exactos, sin query, fragmento, usuario, forma
  escapada ni ruta opaca;
- método `POST`;
- cuerpo máximo de 8 KiB, JSON UTF-8 plano y un único valor raíz;
- claves exactas por operación, sin duplicados, campos extra, objetos o
  colecciones anidados;
- `Content-Type` JSON, `Accept` compatible y lista positiva exclusiva de esas
  dos cabeceras;
- presencia y límites de versión, UUIDv4 y referencias opacas;
- comprobación de cancelación antes y después de cada frontera lenta;
- denegación ante dependencia nil o nil tipada, contexto autoritativo
  incompleto, error acompañado de resultado, recibo discordante o resultado
  no confiable;
- respuesta minimizada, `no-store`, sin `Set-Cookie`, CORS permisivo,
  redirecciones ni detalles internos.

## Regresiones y puertas reproducidas

Todas las ejecuciones válidas siguientes terminaron en verde:

```text
GOPROXY=off go test -count=1 ./internal/modules/contrataciontemporal/adapters/httpinterno
GOPROXY=off go test -race -count=1 ./internal/modules/contrataciontemporal/adapters/httpinterno
GOPROXY=off go test -count=1 ./internal/modules/contrataciontemporal/application -run 'Asignacion|Reasignacion'
GOPROXY=off go vet ./internal/modules/contrataciontemporal/adapters/httpinterno
```

La matriz focal cubre asignación y reasignación nominales, llamada única,
traducción exacta del contexto, nil y nil tipado, identidad ausente, cookies y
cabeceras libres, autoridad inyectada en JSON, claves duplicadas o extra,
JSON compuesto, cuerpos y observaciones fuera de límite, URL y método no
exactos, cancelación antes de leer, tras autoridad y tras ejecutor, errores sin
recibo, error acompañado de recibo, recibos discordantes y saneado de
cabeceras de respuesta.

Se ejecutó además `gofmt` sobre los cuatro ficheros Go y se exige
`git diff --check` limpio antes del commit.

La búsqueda estructural sobre los dos ficheros productivos no devolvió
coincidencias para cookies, `localStorage`, `sessionStorage`, autorización o
cabeceras de identidad. Tampoco encontró lecturas `Header.Get`,
`Header.Values` o `URL.Query`. La inspección de fronteras localizó una llamada
al método de autoridad y una única bifurcación nominal entre `Asignar` y
`Reasignar`.

Una primera invocación auxiliar con `GOTOOLCHAIN=local` no ejecutó pruebas:
el binario base era Go 1.25.11 y `go.mod` exige al menos 1.25.12. Se retiró
solo ese selector, se comprobó Go 1.26.5 cacheado y se reprodujeron todas las
puertas anteriores con `GOPROXY=off`.

## Seguridad, privacidad, i18n y accesibilidad

El cuerpo y la respuesta contienen referencias opacas y datos funcionales
mínimos; no incluyen nombres, documentos, correo, teléfono ni otros datos
personales directos. Los errores son opacos y usan claves i18n bajo
`api.contratacion_temporal.asignacion.error.*`. No se incorpora interfaz
visual; la accesibilidad web no cambia en este corte.

No se usaron datos, credenciales, conectores ni secretos reales. Los valores
de prueba son sintéticos.

## Límites y siguiente corte

Quedan expresamente fuera `rutas.go`, composición raíz, manifiesto, web,
aplicación, puertos, PostgreSQL, documentos transversales, E2E y despliegue.
Estas rutas no están registradas ni son alcanzables desde la composición real;
el corte no acredita una vertical arrancable o productiva.

Siguiente corte: un agente distinto debe revisar y reproducir el hash exacto
del único commit candidato. Solo un `GO` independiente permite que dirección
lo integre y abra después una minitarea separada de registro atómico de rutas.
