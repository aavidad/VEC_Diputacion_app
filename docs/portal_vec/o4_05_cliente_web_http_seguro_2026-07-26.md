# O4-05: cliente web HTTP seguro de Contratación temporal

**Fecha:** 26 de julio de 2026

**Commit funcional:** `023b890`

**Puerta de tamaño web:** `cdea9cf`

**Resultado:** `GO` independiente para el adaptador web aislado

## Alcance cerrado

El portal interno dispone de un cliente HTTP productivo para las cuatro
operaciones reales ya publicadas por los adaptadores de Contratación temporal:

- alta de solicitud;
- propuesta de vía de cobertura;
- decisión de cobertura;
- rectificación de una decisión anterior.

El cliente consume los mismos DTO que los manejadores Go y se incorpora a las
listas positivas `web/interno.manifest` y `web/produccion.manifest`. No se ha
creado otra web ni se han duplicado las diecisiete pantallas de RRHH.

Este corte no conecta falsamente el cliente con la fuente general
`listar/obtener/ejecutar`: todavía no existen las proyecciones productivas de
cuadro, detalle, documentos, auditoría, catálogos y capacidades que requieren
esas pantallas. En modo productivo permanecen cerradas hasta disponer de esos
puertos; en presentación continúan usando exclusivamente su adaptador
sintético aislado.

## Contrato de transporte

Las cuatro operaciones usan `POST` sobre rutas exactas, sin consulta,
fragmento ni URL configurable. Cada petición fija:

```text
credentials: omit
mode: same-origin
cache: no-store
redirect: error
referrerPolicy: no-referrer
```

Las únicas cabeceras son `Accept` y `Content-Type`. El cliente no admite
`Authorization`, cookies, claves de idempotencia en cabecera, identidad,
perfil, organización, roles ni cabeceras `X-*`. La autoridad sigue llegando
al servidor desde la frontera corporativa confiable y no desde el navegador.

Las respuestas se leen de forma incremental y acotada, con:

- `Content-Type` JSON UTF-8 exacto;
- `Content-Length` decimal canónico y coincidente;
- rechazo de compresión y redirecciones;
- máximo por operación y por número de fragmentos;
- UTF-8 fatal;
- JSON compacto y canónico;
- envoltorios, estados HTTP, códigos e i18n cerrados por ruta;
- cancelación de respuestas tardías;
- cero reintentos automáticos.

Las causas privadas de transporte no se conservan en el error público ni se
escriben en consola.

## DTO e invariantes

Los DTO rechazan campos extra, símbolos, accesores, prototipos alterados,
arrays dispersos, propiedades adicionales y valores no canónicos. Los límites
coinciden con Go para referencias, UUID v4, versiones interoperables, vías,
prioridades, claves gobernadas, huellas y precisión temporal.

La identidad semántica recibida se valida nominalmente y se devuelve congelada.
No se recalcula en el navegador: el servidor vuelve a calcularla desde la
propuesta fresca antes del efecto. Un recibo aplicado solo se acepta si su
versión resultante es exactamente la versión esperada más uno.

## Resultado indeterminado

Después de enviar una operación con efecto, una pérdida de transporte, una
respuesta no contractual o `503 operacion_pendiente` no permiten decidir si el
`COMMIT` ocurrió. El cliente marca el resultado como indeterminado y exige
recuperación.

Los dos presentadores aplican la misma regla:

- conservan la intención únicamente en memoria;
- retiran los botones de confirmar, volver y reintentar;
- bloquean cualquier segundo efecto;
- mantienen el bloqueo durante lecturas y navegación ordinarias;
- no confunden la cancelación de una lectura con un efecto incierto.

El bloqueo solo podrá retirarlo la futura consulta protegida de recibo. No se
usa `localStorage`, `sessionStorage`, `IndexedDB` ni cookie para sobrevivir a
una recarga.

## Evidencia

- 51/51 pruebas focales de Contratación web;
- 378/378 pruebas de toda la web;
- suite Go completa;
- `go vet ./...`;
- detector de carreras en el dispatcher y los adaptadores HTTP afectados;
- verificación de manifiestos y autopruebas negativas;
- control de tamaño ampliado a `.mjs`, con las pruebas del módulo por debajo
  del tope duro;
- reglas positivas y negativas de Gitleaks;
- análisis del commit sin secretos;
- dos revisiones independientes con `GO`.

## Límites del cierre

El `GO` acredita el contrato y el cliente aislados, no una capacidad
administrativa productiva. No incrementa las 19 de 46 tareas cerradas porque
siguen pendientes:

1. composición real con frontera mTLS/Kerberos y dependencias productivas;
2. proyecciones protegidas de cuadro, expediente, catálogos y capacidades;
3. consulta mínima de recuperación de recibo;
4. conexión del coordinador sin caída a DEMO;
5. E2E con PostgreSQL 18, pérdida de respuesta y aceptación de RRHH.

El corte O4-05 queda en tres de cinco hitos técnicos internos. La siguiente
entrega debe empezar por las proyecciones y la recuperación protegidas; no por
habilitar pantallas con datos incompletos.
