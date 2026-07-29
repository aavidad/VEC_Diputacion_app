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
validación pública del resultado antes de integrar. Los P2 restantes quedaron
cerrados después en dos minitareas revisadas de forma independiente:
`cd82caa` comprueba cancelación tras decodificar y `fc039c2` completa la
matriz negativa de componentes de `URL`. CT-000047A queda con P0=P1=P2=0.
Véase [la evidencia de revisión](revisiones/o4_05_revision_http_consultas_ct_000047a_2026-07-29.md).

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

La retención nominal quedó integrada en `be59d58` con
[GO independiente](revisiones/o4_05_revision_retencion_nominal_ct_000047b1_2026-07-29.md)
y P0=P1=P2=0. `ContextoConsultaRRHH` conserva ahora el par exacto mediante un
clon privado, sin getter ni serialización, y vuelve a comprobarlo en cada uso.

La [decisión CT-000047B](decision_puentes_autoridad_consulta_rrhh_ct_000047b_2026-07-29.md)
elige guardianes de Contratación temporal y una fachada VEC de alto nivel. Se
descarta una tercera capacidad común sin segundo consumidor real. El siguiente
corte D0 quedó integrado en `6a18b43` con
[GO independiente](revisiones/o4_05_revision_recurso_opaco_ct_000047d0_2026-07-29.md)
y P0=P1=P2=0. D1 y D2 quedan desbloqueados para construir por separado los
recursos cerrados de cuadro y detalle, sin getters ni mapas aportados por el
cliente.

D1 quedó integrado en `c908ce6` con
[GO independiente](revisiones/o4_05_revision_recurso_cuadro_ct_000047d1_2026-07-29.md)
y P0=P1=P2=0. Organización, referencia, módulo, tipo, ámbitos, dominio y
huella se derivan exclusivamente del contexto y la solicitud tipada. D2
quedó integrado después en `c174644` con
[GO independiente](revisiones/o4_05_revision_recurso_detalle_ct_000047d2_2026-07-29.md)
y las mismas garantías cerradas para el expediente y su versión observada.

A2 se dividió para no crear un commit monolítico. A2.1 quedó integrado en
`d69a744` con
[GO independiente](revisiones/o4_05_revision_raiz_nominal_ct_000047a21_2026-07-29.md)
y P0=P1=P2=0. El servicio de confianza conserva y recupera internamente la
raíz pública nominal ligada a una prueba exacta; A2.2 puede ahora construir la
fachada de emisión sin exponer el catálogo o una clave.

A2.2 quedó integrado en `a677b14` con
[GO independiente](revisiones/o4_05_revision_fachada_emision_vec_ct_000047a22_2026-07-29.md)
y P0=P1=P2=0. La fachada encadena concesión durable, atestación, confianza,
capacidad, raíz y material sin aceptar selectores libres. A4 puede empezar a
componer los dos emisores nominales y guardianes de cuadro y detalle.

A4.1 quedó integrado en `05d6767` con
[GO independiente](revisiones/o4_05_revision_par_emisores_ct_000047a41_2026-07-29.md).
El par privado rechaza nulos, implementaciones sin identidad física y la misma
instancia en ambas posiciones. A4.2 continúa con la preparación nominal pura.

El contrato M0 de motivos quedó integrado en `c1bb5ec` con
[GO independiente](revisiones/o4_05_revision_contrato_motivos_ct_000047m0_2026-07-29.md).
Cuadro y detalle tienen métodos nominales separados, sin selectores libres.
M1/M2 siguen pendientes de la publicación gobernada y del adaptador
PostgreSQL; no se sustituirán por referencias fijadas en el código.

El emisor HMAC actual permanece limitado a pruebas. Antes de producción falta
el puerto de MAC no exportable y el adaptador KMS/HSM, además del firmante COSE
V3 y el conjunto de confianza proporcionados por Sistemas.

La cronología de detalle quedó corregida en `e3e12d5`: contexto, autorización
y orden ya no comparten un instante capturado antes de las fronteras lentas.
La minitarea obtuvo `GO` independiente sin hallazgos. Véase la
[revisión CT-000047C2](revisiones/o4_05_revision_cronologia_detalle_ct_000047c2_2026-07-29.md).

La misma garantía quedó cerrada para el cuadro en `ce77db3`, también con `GO`
independiente y P0=P1=P2=0. Véase la
[revisión CT-000047C1](revisiones/o4_05_revision_cronologia_cuadro_ct_000047c1_2026-07-29.md).

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
