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

## CT-000047A integrado

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

El conjunto `c430785`–`b00d2ec` obtuvo `GO` independiente con P0=0 y P1=0.
La revisión inicial obligó a corregir ruta opaca, prioridad de cancelación y
validación pública del resultado antes de integrar.

Quedan dos P2 separados: comprobación de cancelación tras decodificar y tabla
negativa exhaustiva de componentes de `URL`. Se conservan en
[la evidencia de revisión](revisiones/o4_05_revision_http_consultas_ct_000047a_2026-07-29.md).

El corte no toca SQL, PostgreSQL, raíz, web, estilos ni adaptadores de
presentación. Añade validadores neutrales en `ports` para impedir que HTTP
publique una proyección no coherente con la solicitud; las garantías privadas
de capacidad, ámbito y recibo permanecen en sus validadores internos.

## CT-000047B

Antes de programarlo debe quedar probado que cada conversión reutiliza las
autoridades VEC existentes y no fabrica material de confianza. Los proveedores
corporativos ausentes se inyectarán por puertos; no se sustituirán por memoria,
cabeceras, cookies, datos DEMO o configuración de desarrollo.

El primer prerrequisito mínimo está integrado:

- `700d72a` devuelve vínculo y resultado registrado desde una única
  resolución;
- `f49afd0` conserva el contrato y los errores del constructor compatible;
- `f186ce9` cierra las copias defensivas de todos los contenedores mutables.

El conjunto obtuvo `GO` independiente con P0=0, P1=0 y P2=0. La
[evidencia](revisiones/o4_05_revision_vinculo_resultado_ct_000047b_a_2026-07-29.md)
no acredita todavía autoridad de canal, PDP, raíz ni E2E.

Dos inventarios están descomponiendo el resto en unidades mínimas compilables.
No se abrirá una tarea genérica de «autoridad/PDP»: retención nominal, recursos
cerrados, decisión, material atestado, cronología, rutas y ciclo de vida se
confirmarán y revisarán por separado.

La cronología de detalle quedó corregida en `e3e12d5`: contexto, autorización
y orden ya no comparten un instante capturado antes de las fronteras lentas.
La minitarea obtuvo `GO` independiente sin hallazgos. Véase la
[revisión CT-000047C2](revisiones/o4_05_revision_cronologia_detalle_ct_000047c2_2026-07-29.md).

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
