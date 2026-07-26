# Encargo externo — revisión independiente O2-09A

## Mandato

Revisa la interfaz definitiva aislada del alta de contratación temporal. No la
conectes a una API falsa, no corrijas el candidato y no la declares E2E. El
objetivo es decidir si puede integrarse como vista neutral pendiente de
composición real.

Lee completos `AGENTS.md`, `ORQUESTACION_AGENTES.md`,
`ESTADO_PROYECTO.md`, `docs/instruccion_direccion_2026-07-18.md`,
`docs/portal_vec/cola_trabajo_agentes_contratacion_temporal_2026-07-23.md`,
el encargo O2-09A y `INTEGRACION.md` del módulo.

## Subagentes obligatorios

Usa dos subagentes en solo lectura:

- especialista en accesibilidad WCAG/UNE, teclado, lector, foco, contraste y
  zoom;
- especialista en seguridad del navegador, Unicode, inyección, concurrencia y
  ciclo de vida.

Reproduce personalmente cualquier fallo. Ninguno edita la rama candidata.

## Fuente exacta

Rama `agent/ct-o2-09-web`, SHA `4d2f169`, worktree:

`/home/alberto/Trabajo/VEC_Diputacion_app/.worktrees/ct-o2-09-web`

Base de comparación: `3773205`.

## Criterios bloqueantes

Verifica:

1. comando y catálogos cerrados, sin listas funcionales compiladas;
2. i18n completo, sin HTML inseguro ni textos funcionales dispersos;
3. teclado, foco, lector, errores asociados, estados, contraste, zoom 200/400 %
   y reducción de movimiento;
4. revisión antes de envío, doble clic, reintento idéntico, cancelación
   indeterminada y recibo;
5. clave idempotente CSPRNG solo en memoria, no visible ni persistida;
6. cero cookies, `localStorage`, `sessionStorage`, credenciales o cabeceras de
   identidad/rol;
7. ninguna decisión de autorización confiada a `capacidad` de presentación;
8. Unicode hostil, etiquetas, referencias, importes y fechas;
9. desmontaje completo de oyentes, temporizadores y promesas;
10. contrato exacto con O2-08A y brechas honestamente documentadas;
11. herencia del tema principal y ausencia de regresiones del shell;
12. ningún dato DEMO, éxito simulado o adaptador falso.

Reproduce además esta hipótesis de Dirección: el comando O2-09A contiene una
`clave_idempotencia` generada por el presentador, mientras O2-08A no acepta
esa clave por cuerpo ni cabecera y obtiene otra desde contexto confiable.
Comprueba si el reintento conserva realmente la identidad autoritativa de la
operación extremo a extremo. Si no puede demostrarlo, marca la conexión
O2-08B como bloqueada; no conviertas la clave del navegador en autoridad.

Debe seguir siendo utilizable por web, escritorio u otro cliente sobre el
mismo caso de uso. Revisa castellano, hexagonalidad, minimización, denegación
por defecto y cero secretos/datos personales.

## Puertas mínimas

Ejecuta la suite Node del módulo al menos cincuenta veces, las pruebas web
globales pertinentes, análisis estático disponible, Gitleaks, `diff-check`,
tamaños y `merge-tree`. Usa un navegador/capturador local para inspección
visual de escritorio si el repositorio ya ofrece el entorno; no instales ni
contamines el equipo. Registra cualquier prueba manual pendiente.

## Entrega

Crea `.worktrees/rev-o2-09a`, rama `review/rev-o2-09a`, desde `54027d0`.
Solo añade:

`docs/portal_vec/revisiones/o2_09a_revision_independiente_final_2026-07-23.md`

Incluye `GO/NO-GO`, evidencia visual cuando proceda, contraejemplos, puertas,
brechas de O2-08/O2-10 y riesgos. Commit documental único. No modifiques el
candidato.
