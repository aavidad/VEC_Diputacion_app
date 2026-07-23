# Encargo aislado O2-09A — interfaz definitiva del alta de contratación temporal

## Mandato

Lee este documento completo y ejecuta el encargo sin ampliar alcance. Antes de
editar, lee también `AGENTS.md`, `ORQUESTACION_AGENTES.md`,
`docs/instruccion_direccion_2026-07-18.md` y
`docs/portal_vec/tablero_tareas_contratacion_temporal_2026-07-23.md`.

Si está disponible, usa la habilidad `admin-data-web`. Esta pantalla pertenece
a un portal administrativo denso y debe priorizar claridad operativa,
accesibilidad, trazabilidad y prevención de errores.

Son obligatorias todas las reglas del proyecto: código, pruebas, mensajes y
documentación en castellano coherente; i18n sin textos funcionales dispersos;
arquitectura hexagonal y adaptadores intercambiables; denegación por defecto;
neutralidad web/escritorio/CLI/MCP; ausencia de secretos y datos personales;
documentación simultánea; seguridad de Administración pública y todas las
puertas de `AGENTS.md`. Los términos técnicos universales pueden conservarse,
pero no se admite mezclar castellano e inglés en el dominio.

No uses Orquesta. No edites el árbol principal. No integres ni empujes tu rama.

## Subagentes obligatorios

El agente principal es el **único editor** del worktree. Debe crear dos
subagentes de revisión, con acceso de solo lectura y sin commits:

1. revisor de experiencia administrativa, WCAG 2.2 AA, EN 301 549, teclado,
   foco, zoom y lector de pantalla;
2. revisor de seguridad y contrato: campos cerrados, escape, doble envío,
   errores redactados, ausencia de cookies/almacenamiento y material privado.

Los revisores trabajan después de un primer corte coherente o sobre diffs
estables. Entregan GO/NO-GO y evidencias al agente principal. No pueden editar
archivos, coordinar otros agentes ni declarar integrada la tarea. El principal
reproduce los hallazgos, corrige y vuelve a pedir comprobación de los bloqueos.

## Preparación obligatoria

Desde `/home/alberto/Trabajo/VEC_Diputacion_app`:

```bash
git worktree list
test ! -e .worktrees/ct-o2-09-web
git worktree add .worktrees/ct-o2-09-web \
  -b agent/ct-o2-09-web feature/contratacion-temporal
cd .worktrees/ct-o2-09-web
```

Si la rama o el worktree ya existen, detente e informa; no crees otro nombre.
Todo el trabajo debe permanecer dentro de
`/home/alberto/Trabajo/VEC_Diputacion_app/.worktrees/ct-o2-09-web`.

## Objetivo

Construir la interfaz **definitiva** para registrar una solicitud de
contratación temporal desde el Portal del Empleado interno. Es la parte visual
paralelizable de O2-09; no se declara conectada ni cerrada hasta que O2-08
aporte la API real.

No hagas una demo, un «mock» de éxito ni un adaptador falso. La vista recibirá
por inyección un ejecutor neutral; sin ejecutor real debe mostrarse no
disponible y no producir ningún resultado ficticio. Los únicos datos
sintéticos admitidos son fixtures dentro de pruebas.

## Fuentes que mandan

1. El modelo Go real:
   - `internal/modules/contrataciontemporal/domain/solicitud.go`;
   - `internal/modules/contrataciontemporal/application/registro_solicitud.go`;
   - `internal/modules/contrataciontemporal/ports/alta.go`.
2. El manifiesto:
   - `internal/modules/contrataciontemporal/manifest.go`.
3. El diseño visual heredado:
   - `web/static/portal-empleado/index.html`;
   - `web/static/portal-empleado/portal-*.css`;
   - módulos `cronos` y `dietas` como patrón de composición, no como modelo
     funcional.
4. Las instrucciones de RRHH del documento de la raíz
   `Pantalla de procedimiento de gestión de contratación y gestión de bolsas.docx`
   se consultan solo en lectura desde el árbol principal. No se copian al
   worktree, no se versionan y no se inventan campos que contradigan el dominio.

Si el Word exige un dato que aún no representa el dominio, regístralo en
`INTEGRACION.md` como brecha bloqueante; no lo ocultes en un campo libre.

## Experiencia requerida

La vista debe cubrir, como mínimo:

- contexto y explicación del trámite;
- centro, persona responsable referenciada, categoría, grupo/subgrupo y motivo
  mediante catálogos inyectados, nunca listas compiladas en la vista;
- detalle de la necesidad con límite visible;
- periodo de inicio y fin;
- existencia de retención de crédito y, cuando proceda, fecha, importe exacto
  en céntimos/moneda y referencia documental;
- adjuntos por referencias opacas ya incorporadas al expediente documental;
  esta tarea no sube bytes;
- resumen previo a confirmar;
- envío único, bloqueo de doble pulsación, estado ocupado y cancelación segura;
- errores por campo, error general redactado, conservación del foco y anuncio
  `aria-live`;
- recibo final con referencia de expediente, número visible, versión,
  referencia de recibo y fecha, sin exponer decisiones, HMAC, identidad,
  capacidad ni material probatorio privado.

No pedir DNI, nombre, correo, rol, permisos ni identidad declarada. El contexto
de actor y los catálogos proceden de composición/servidor.

## Arquitectura y archivos

Crea exclusivamente:

```text
web/static/portal-empleado/modulos/contratacion-temporal/
  contrato.js
  presentador.js
  vista.js
  i18n.js
  contratacion-temporal.css
  contratacion-temporal.test.mjs
  INTEGRACION.md
```

Puedes modificar solo si resulta imprescindible:

```text
web/static/portal-empleado/portal-modulos-coordinador.js
web/static/portal-empleado/portal-modulos-coordinador.test.mjs
web/static/portal-empleado/index.html
locales/es.json
web/interno.locales.manifest
```

No modifiques Go, SQL, `ports`, adaptadores PostgreSQL, Bolsa, Cronos, Dietas,
datos de presentación, composición de desarrollo ni documentación compartida.
No añadas dependencias.

Separación obligatoria:

- `contrato.js`: DTO cerrado, límites, clonación defensiva y validación;
- `presentador.js`: estado y coordinación de la interacción; recibe funciones
  inyectadas y no conoce HTTP;
- `vista.js`: DOM, eventos, foco y renderizado; no contiene reglas de negocio;
- `i18n.js`: claves y mensajes en castellano; la vista no dispersa textos de
  error;
- CSS: solo variables/tokens heredados del portal; nada de estilos en línea;
- `INTEGRACION.md`: contrato que deberá satisfacer O2-08 y lista exacta de
  puntos aún no conectados.

El adaptador HTTP real pertenecerá a O2-08. No lo anticipes ni fijes una URL.

## Seguridad e interoperabilidad

- Sin `Cookie`, `Set-Cookie`, `document.cookie`, `localStorage`,
  `sessionStorage`, IndexedDB ni credenciales en URL.
- Sin autoridad derivada de cabeceras o campos controlados por el navegador.
- Neutral para web, escritorio, CLI y MCP: la UI consume el mismo comando y
  recibo del caso de uso.
- Denegación por defecto: sin capacidad y ejecutor expresos, no hay botón
  funcional.
- No registrar ni mostrar claves de idempotencia, sellos HMAC, decisiones,
  atestaciones, tokens, trazas privadas o datos personales.
- No usar `innerHTML` con datos dinámicos sin escape estricto. Preferir nodos y
  `textContent`.
- Nada de llamadas externas, CDN, telemetría ni fuentes remotas.

## Accesibilidad y aspecto

- WCAG 2.2 AA y EN 301 549: teclado completo, foco visible, etiquetas,
  descripciones, agrupaciones `fieldset/legend`, errores asociados,
  contraste y movimiento reducido.
- Escritorio es prioritario, pero no debe romperse a 200 % de zoom ni a
  1280×720.
- Hereda logo, cabecera, lateral, tipografía, temas y alto contraste del shell.
- La información densa se agrupa en secciones; acciones principales visibles;
  ningún dato importante depende solo del color.

## Pruebas de cierre del candidato

Incluye pruebas Node sin red para:

1. contrato cerrado y rechazo de campos extra, tipos, tamaños y fechas;
2. catálogos inyectados y ausencia de opciones compiladas;
3. ausencia de datos personales y material de seguridad en comando/recibo;
4. envío correcto, doble envío, cancelación, error redactado y reintento;
5. recibo adulterado o incompleto;
6. capacidad ausente: fallo cerrado;
7. escape de contenido hostil;
8. recorrido de teclado, etiquetas y estados `aria`;
9. búsqueda negativa de cookies y almacenamiento web;
10. integración del módulo sin romper Bolsa, Cronos ni Dietas.

Ejecuta:

```bash
node --test web/static/portal-empleado/**/*.test.mjs
go test ./...
go vet ./...
git diff --check
```

Comprueba también que ningún fichero Go nuevo supere 800 líneas, que no haya
secretos con Gitleaks si está disponible y que `git status` quede limpio.

## Entrega

Realiza commits pequeños en castellano. No empujes la rama. Entrega:

- SHA de cada commit;
- archivos modificados;
- pruebas y resultado;
- capturas o inspección visual realizada, si el entorno lo permite;
- brechas registradas para O2-08;
- riesgos restantes;
- declaración expresa: **candidato O2-09A, no conectado y no integrado**.

El director asignará un revisor distinto. Solo un GO independiente permitirá
integrar, y O2-09 no se marcará completo hasta probarlo contra O2-08 real.
