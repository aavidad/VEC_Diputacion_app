# Relevo del frente de contratación temporal — 23/07/2026

Documento de entrada obligatorio para cualquier agente que continúe este
frente. Debe actualizarse en cada commit que cambie alcance, arquitectura,
estado o siguiente paso.

## Objetivo

Implementar el procedimiento remitido por RRHH para gestionar expedientes de
contratación temporal desde la petición del centro hasta GINPIX, conservando
íntegramente las capacidades existentes de Bolsa.

## Rama y base

- Rama: `feature/contratacion-temporal`.
- Base técnica: `real/c3-convergencia`, commit `eeac3a2`.
- Motivo: contiene C3 público auditado y la cápsula interna C4.
- No desarrollar este frente sobre `vec-orquesta-20260619`: en el momento de
  abrirlo tenía cuatro commits documentales propios, pero estaba 73 commits por
  detrás de C3.

Las ramas `real/t17a-postgres-importador` y
`real/t20-borradores-postgresql` siguen siendo trabajos separados e
incompletos. No deben mezclarse sin auditoría y reconciliación explícitas.

## Fuente de requisitos

Original local recibido:

```text
/home/alberto/Trabajo/VEC_Diputacion_app/
Pantalla de procedimiento de gestión de contratación y gestión de bolsas.docx
```

El fichero temporal de LibreOffice cuyo nombre empieza por `.~lock` no es
documentación y no debe versionarse.

La lectura normalizada se conserva en:

- `docs/portal_vec/expediente_contratacion_temporal_rrhh.md`.

## Decisión principal

Se crea `internal/modules/contrataciontemporal`. No se amplía Bolsa hasta
convertirla en un monolito.

```text
contrataciontemporal
  coordina → bolsa
  coordina → personal
  coordina → documentos/firma
  coordina → comunicaciones
  coordina → intervención
  coordina → GINPIX
```

Cada módulo mantiene su autoridad y solo intercambia referencias opacas,
comandos y eventos. No se permiten lecturas o escrituras directas en tablas
ajenas.

## Estado comprobado

| Entregable | Estado |
| --- | --- |
| Especificación inicial | Implementada |
| Manifiesto del módulo | Implementado y probado |
| Dominio de expediente | Pendiente |
| Casos de uso y puertos | Pendiente |
| PostgreSQL | Pendiente |
| API interna | Pendiente |
| Web conectada | Pendiente |
| E2E administrativo | Pendiente |

## Permisos iniciales

- `contratacion_temporal.cuadro.consultar`
- `contratacion_temporal.expediente.consultar`
- `contratacion_temporal.solicitud.crear`
- `contratacion_temporal.analisis.validar`
- `contratacion_temporal.cobertura.decidir`
- `contratacion_temporal.unidad.asignar`
- `contratacion_temporal.flujo.configurar`
- `contratacion_temporal.auditoria.consultar`

Son capacidades técnicas publicadas por el módulo. No se asignan a perfiles
desde el manifiesto ni conceden acceso sin una decisión positiva del PDP.

## Siguiente corte exacto

1. Crear el dominio sin dependencias técnicas:
   - referencia de flujo versionada;
   - solicitud del centro;
   - análisis RRHH y RC;
   - vía de cobertura;
   - asignación;
   - fase, estado y actuaciones de solo adición.
2. Probar validaciones, copias defensivas, versiones y transiciones.
3. Incorporar puertos de repositorio, flujo, autorización, reloj, referencias
   e idempotencia.
4. Añadir el primer caso de uso: registrar una solicitud del centro.

## Invariantes

- Las fases y opciones funcionales proceden de definiciones y catálogos
  gobernados; no se añaden como listas cerradas en el código.
- Los estados técnicos de seguridad y consistencia sí permanecen cerrados.
- Importes en unidades monetarias menores, nunca `float64`.
- Instantes UTC canónicos; los calendarios civiles se resuelven por puerto.
- Historial de solo adición.
- Una transición exige versión esperada y falla ante concurrencia.
- Ninguna identidad, rol, ámbito o decisión procede del navegador.
- No se copian DNI, teléfonos o correos en cuadros de mando.
- La integración GINPIX será un puerto con adaptadores API y fichero.
- Una pantalla DEMO no se contabiliza como integración.

## Puertas antes de commit

Como mínimo:

```text
gofmt
go test ./internal/modules/contrataciontemporal/...
go test -race ./internal/modules/contrataciontemporal/...
go vet ./internal/modules/contrataciontemporal/...
git diff --check
```

Al tocar composición, seguridad o artefactos deben ejecutarse además las
puertas focales de C3/C4 y `scripts/verificar_calidad.sh`.

## Regla de documentación

Cada corte debe actualizar:

1. este relevo;
2. la especificación si cambia una decisión;
3. `README.md` si cambia el estado visible;
4. la matriz operativa cuando una capacidad cambie realmente de nivel;
5. las pruebas y evidencia exacta del commit.

Nunca sustituir «pendiente» por «terminado» porque exista una pantalla, un
puerto o una prueba aislada.
