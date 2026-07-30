# Inventario de capturas pendientes al cierre de la web

Fecha de auditoría: 30 de julio de 2026.

Estado: **no regeneradas por decisión de dirección**.

## Resultado

Las veinte imágenes publicadas en `docs/manual_usuario/capturas/` no acreditan
el aspecto del corte vigente. Su última versión es del 17 de julio de 2026:
diecinueve quedaron en `d1b6754` y `13_ayuda.png` en `199431c`. Desde
`d1b6754` la superficie `portal-empleado`, su lanzador y los capturadores han
recibido cambios materiales de navegación, seguridad, módulos, contratación
temporal, llamamientos, documentos y estilos.

La comparación Git del corte `d1b6754..cc6041e` sobre esas fuentes registra
137 ficheros afectados y más de treinta mil líneas añadidas. Por tanto, las
imágenes se conservan como evidencia histórica del manual de presentación,
pero deben considerarse **pendientes de actualización**, no prueba visual de
la aplicación actual.

No se ha retocado ni sustituido ninguna imagen. Tampoco se ha arrancado una
composición incompleta para producir capturas que parezcan definitivas.

## Inventario exacto

El generador histórico usa el perfil `administrador` de la composición
`presentacion=rrhh` y la ruta base `/portal-empleado/`. Esos selectores son de
demostración y no constituyen identidad o autorización productivas.

| Captura publicada | Ruta o interacción histórica | Flujo representado | Motivo para repetirla al cierre |
| --- | --- | --- | --- |
| `01_portada.png` | `#portal` | Entrada al Portal del Empleado | Cambiaron lanzador, módulos y separación de perfiles. |
| `02_cuadro_mando.png` | `#bolsa/resumen` | Cuadro de mando de Bolsa | Cambiaron resumen, menú, módulos y composición interna. |
| `03_elaboracion.png` | `#bolsa/elaboracion` | Elaboración y gestión de bolsas | Se añadieron gestión gobernada y borradores conectables. |
| `03b_detalle_expediente.png` | `#bolsa/elaboracion` y seleccionar expediente | Detalle del expediente | Cambiaron componentes de expediente y paneles operativos. |
| `03c_configurar_bases.png` | diálogo `configurar-bases` | Configuración de bases y baremo | Cambiaron vistas de gobierno, reglas y baremación. |
| `04a_llamamiento_paso1.png` | `#bolsa/llamamientos` y elegir necesidad | Llamamiento: elegir necesidad | Se incorporó el asistente profesional de llamamientos. |
| `04b_llamamiento_paso2.png` | solicitar propuesta | Llamamiento: propuesta calculada | Cambiaron contrato, propuesta y presentación de resultados. |
| `04c_llamamiento_paso3.png` | siguiente paso | Llamamiento: configuración | Cambiaron controles, contrato y flujo de llamamientos. |
| `04d_llamamiento_paso4.png` | siguiente paso | Llamamiento: revisión | Cambiaron revisión, recibos y separación de autoridad. |
| `05_contratos.png` | `#bolsa/contratos` | Contratos, ceses y reincorporaciones | Cambiaron shell, vistas operativas y navegación. |
| `06_reglas.png` | `#bolsa/reglas` | Motor de reglas | Cambiaron vistas de reglas y composición gobernada. |
| `06b_detalle_regla.png` | diálogo `detalle-regla` | Detalle de regla | Cambiaron presentación y componentes de reglas. |
| `07_consulta.png` | `#bolsa/consulta` | Consulta segura de candidaturas | Cambiaron fronteras pública/interna y retirada de credenciales web. |
| `08_estadisticas.png` | `#bolsa/estadisticas` | Estadísticas | Cambiaron shell, menú y componentes comunes. |
| `09_documentos.png` | `#bolsa/documentos` | Documentos y firma | Se añadieron recibos PDF/QR y controles de descarga. |
| `10_comunicaciones.png` | `#bolsa/comunicaciones` | Correo y mensajería | Cambiaron shell, menú y componentes comunes. |
| `11_auditoria.png` | `#bolsa/auditoria` | Auditoría y trazabilidad | Cambiaron contexto de actor y separación de autoridad. |
| `12_avisos.png` | diálogo `avisos` desde reglas | Avisos | Cambiaron shell, menú y componentes comunes. |
| `13_ayuda.png` | diálogo `ayuda` desde reglas | Ayuda | Cambiaron contenido, navegación y componentes comunes. |
| `14_alto_contraste.png` | activar contraste desde reglas | Preferencia de alto contraste | Debe repetirse sobre todos los estilos definitivos. |

El PDF `docs/manual_usuario/manual_portal_bolsas.pdf` incorpora estas capturas
y queda igualmente identificado como manual histórico pendiente de
regeneración. No debe presentarse como evidencia del corte productivo.

## Evidencia visual no versionada

Existen dos puertas históricas reproducibles cuyos PNG se guardan bajo `var/`
y no forman parte de Git:

- `var/revision-web/`: 183 escenarios y capturas de la presentación, con corte
  documentado el 20 de julio;
- `var/revision-web-contratacion-rrhh/`: 51 capturas de las 17 pantallas de
  RRHH en tres resoluciones, con corte documentado el 24 de julio.

Sus informes conservan valor histórico, pero las dos matrices deben repetirse
cuando se cierre la web. Ninguna acredita por sí sola composición productiva,
identidad real, PostgreSQL o aceptación de RRHH.

## Por qué no se generan las definitivas ahora

C2.1b, selección y registro corporativos, PDP, composición raíz, TLS/mTLS y el
E2E siguen abiertos. La composición de presentación usa adaptadores
sintéticos y una marca técnica propia; producir desde ella nuevas imágenes no
demostraría la aplicación real. No se usarán datos personales reales para
solventar esa carencia.

## Procedimiento reproducible al cierre

1. Fijar el commit estable exacto y comprobar que web, API, identidad, PDP y
   PostgreSQL pertenecen al mismo corte revisado.
2. Arrancar la composición real de integración en contenedores, con datos
   completamente sintéticos cargados por los puertos productivos. Se prohíben
   autoridad `fake`, cookies, almacenamiento web y adaptadores DEMO.
3. Registrar para cada imagen: commit, URL/ruta, superficie, perfil
   autorizado, flujo, resolución y referencia del conjunto sintético.
4. Ejecutar primero las pruebas de contrato, seguridad y accesibilidad de la
   web. Una puerta fallida impide publicar capturas.
5. Adaptar `docs/manual_usuario/generar_capturas.py`, que hoy fija
   `presentacion=rrhh&perfil=administrador`, para reconocer la composición
   real y recorrer perfiles autorizados; no se desactiva su validación de
   destino ni se retocan los PNG.
6. Recorrer como mínimo las veinte vistas del inventario y las diecisiete
   pantallas RRHH en `1536 × 1024`, `1440 × 1000` y `1280 × 900`.
7. Revisar teclado, foco, contraste, desbordamientos, consola, errores de red,
   ausencia de PII y coherencia con el contrato funcional.
8. Sustituir las imágenes del manual, regenerar su PDF y confirmar juntos
   capturas, manual, informe de revisión y huellas.
9. Obtener revisión visual independiente y aceptación de RRHH antes de
   declarar las capturas definitivas.

La presentación histórica puede reproducirse, sin convertirla en evidencia
productiva, mediante:

```bash
scripts/arrancar_presentacion_rrhh.sh
docker compose --profile presentacion --profile herramientas-presentacion run \
  --rm --no-deps revision-web-presentacion
```

La puerta específica de las diecisiete pantallas se conserva en
[la revisión local de RRHH](revisiones/web_contratacion_rrhh_revision_local_2026-07-24.md).
El procedimiento y las salidas de la presentación están en
[captura y revisión de la presentación](revision_web_presentacion.md).
