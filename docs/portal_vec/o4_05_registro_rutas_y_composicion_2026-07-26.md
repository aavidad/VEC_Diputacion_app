# O4-05: registro seguro de rutas y composición modular

**Fecha:** 26 de julio de 2026

**Estado:** `GO` independiente para el corte mecánico aislado

## Resultado

La carcasa HTTP de VEC puede recibir adaptadores de módulos mediante rutas
exactas sin importar el dominio de Contratación ni convertirse en una fábrica
de infraestructura. El corte incorpora de forma atómica los manejadores reales
ya existentes de alta y cobertura:

- `POST /api/vec/contratacion-temporal/solicitudes`;
- `POST /api/vec/contratacion-temporal/cobertura/propuesta`;
- `POST /api/vec/contratacion-temporal/cobertura/decisiones`;
- `POST /api/vec/contratacion-temporal/cobertura/rectificaciones`.

Las tres operaciones de cobertura comparten un único adaptador cerrado. Si
falta cualquiera de las seis dependencias funcionales, incluso mediante un
nulo tipado, no se devuelve ninguna ruta parcial.

## Decisiones de seguridad

El registro de rutas y su autoridad son obligatorios conjuntamente. Antes de
delegar, la carcasa exige una autoridad de rutas que solo puede evaluar la
capacidad opaca depositada por la futura frontera corporativa en el contexto.
No puede deducirla desde URL, cuerpo, cabeceras, cookies o datos del navegador.

La denegación se produce antes del manejador y del caso de uso:

| Condición | Respuesta |
| --- | --- |
| autenticación ausente o caducada | `401 autenticacion_requerida` |
| organización o capacidad denegada | `403 acceso_denegado` |
| autoridad no disponible o error desconocido | `503 servicio_no_disponible` |

Las respuestas se redactan, incluyen una referencia de correlación opaca,
eliminan cookies, CORS, redirecciones, `Retry-After` y compresión, y aplican
cabeceras de no almacenamiento. Las URL con `RawPath`, `Opaque`, fragmento,
`ForceQuery`, autoridad embebida o escape no canónico se rechazan antes de
consultar la autoridad.

Por mínimo conocimiento, `GET /api/vec` no anuncia las rutas internas
registradas. Las capacidades funcionales deberán descubrirse mediante una
proyección autorizada propia del módulo, no mediante el inventario genérico de
la carcasa.

## Evidencia

- construcción indivisible de dos manejadores reales y cuatro rutas;
- duplicados, colisiones, rutas dinámicas, patrones, nulos y
  `http.ServeMux` rechazados al arrancar;
- copia defensiva de la declaración;
- autoridad invocada exactamente una vez con la ruta completa;
- autoridad y negocio no invocados ante una URL no canónica;
- matriz integrada de las cuatro rutas para `401`, `403` y `503`, con cero
  llamadas funcionales;
- pruebas focales con detector de carreras;
- suite Go completa, `go vet ./...`, formato y `git diff --check`;
- revisión independiente con `GO` para este alcance.

## Límite del cierre

Este corte no incrementa todavía las 19 de 46 tareas productivas verificadas.
No acredita C5, identidad corporativa, Kerberos, pools PostgreSQL productivos,
KMS/HSM ni E2E. El cliente web se cerró después en `023b890`, sin convertir
este registro mecánico en composición productiva. `cmd/vec-interno` debe seguir
fallando cerrado.

La cadena obligatoria pendiente es:

```text
TLS mutuo verificado
→ frontera corporativa mTLS/Kerberos
→ carcasa interna y lista positiva
→ API VEC
→ autoridad de rutas
→ adaptadores de Contratación
```

La frontera corporativa debe quedar entre el verificador TLS y
`server.NewHTTPServerInterno`. No puede envolverse solo alrededor de la API
porque la carcasa elimina las cabeceras de transporte antes de alcanzarla.
