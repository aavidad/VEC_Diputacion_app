# Relevo de sesión: inicio de CT-000047

Fecha: 29 de julio de 2026.

## Motivo

Después de cerrar CT-000046, dos inventarios independientes contrastaron el
camino crítico documental con el código no de prueba. La composición raíz no
puede ser el siguiente commit aislado: antes faltan dos fronteras que el estado
anterior daba por disponibles demasiado pronto.

No se ha perdido trabajo ni se revierte CT-000046. El hallazgo corrige el orden
de integración y evita construir una raíz con dobles o autoridad ficticia.

## Estado confirmado

Sí existen y se reutilizan:

- las fachadas PostgreSQL CT-000045;
- el adaptador y pool nominal CT-000046;
- los casos de uso de cuadro y detalle;
- el dispatcher de rutas exactas;
- la cápsula mTLS/TLS 1.3;
- el ciclo de vida con propiedad exclusiva y cierre inverso;
- las piezas comunes de sesión, contexto actor y PDP V3.

No existen todavía en código productivo:

1. los manejadores HTTP de cuadro y detalle;
2. una implementación de `AutoridadContextoConsultaRRHH`;
3. una implementación de `AutorizadorConsultaRRHH`;
4. el ensamblado de esas piezas en la raíz y en la web.

Además, Sistemas debe proporcionar los conectores y materiales reales de
verificación de aserciones corporativas, política de garantía, KMS/firma,
certificados, roles y secretos. Su ausencia mantiene el arranque cerrado.

## Orden corregido

```text
CT-000047A: HTTP protegido de cuadro/detalle
→ CT-000047B: puentes de autoridad y PDP reales
→ composición raíz y propiedad de recursos
→ matriz TLS/mTLS viva
→ fuente HTTP de la misma web definitiva
→ E2E HTTP completo contra PostgreSQL 18
→ conformidades RRHH, DPD y Sistemas
```

## CT-000047A en ejecución

Write-set exclusivo:

```text
internal/modules/contrataciontemporal/adapters/httpinterno/consulta_rrhh*.go
```

Alcance:

- dos rutas `POST` exactas ya fijadas por el diseño;
- cuerpos JSON cerrados que solo transportan intención;
- límites, tipos y cabeceras en lista positiva;
- rechazo de cookies, credenciales y cabeceras de identidad libres;
- respuestas `no-store`, cero `Set-Cookie` y errores opacos localizables;
- reutilización de los casos de uso y constructores existentes;
- pruebas de contrato, límites, cancelación y no oráculo.

No toca dominio, aplicación, puertos, SQL, PostgreSQL, raíz, web, estilos ni
adaptadores de presentación.

## CT-000047B

Antes de programarlo debe quedar probado que cada conversión reutiliza las
autoridades VEC existentes y no fabrica material de confianza. Los proveedores
corporativos ausentes se inyectarán por puertos; no se sustituirán por memoria,
cabeceras, cookies, datos DEMO o configuración de desarrollo.

## Métricas

| Ámbito | Estado |
| --- | --- |
| Contratación temporal | `24/46`, 52 % |
| O4-05 | `3/5` hitos |
| Bolsa productiva | `1/14`, 7 % |
| Producción | `NO-GO` |

El inventario no reduce el avance funcional: hace visible una dependencia de
integración que ya estaba abierta. CT-000047 solo aumentará la métrica cuando
su candidato esté publicado, revisado y con CI verde.

## Autoridad de trabajo

Rama estable:

```text
integracion/ct-o4-04e-20260726
```

Worktree estable:

```text
.worktrees/ct-stable-docs
```

Candidato CT-000047A:

```text
.worktrees/ct47-http-consultas-20260729
agent/ct47-http-consultas-20260729
```

No se programa en el directorio raíz histórico ni se modifica el Word de RRHH
sin seguimiento.
