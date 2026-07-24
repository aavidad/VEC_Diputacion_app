# Revisión local de las 17 pantallas de contratación de RRHH

Fecha: 24 de julio de 2026.

## Alcance

Esta revisión corrige la evidencia que provocó el NO-GO visual anterior. No
sustituye la aceptación de RRHH ni habilita producción.

La fuente funcional es el documento recibido:
«Pantalla de procedimiento de gestión de contratación y gestión de bolsas».
Las cuatro imágenes incorporadas al Word miden `1536 × 1024` y contienen las
diecisiete pantallas numeradas que se recogen en
[el estado de la web](../estado_web_contratacion_temporal_2026-07-23.md).

## Correcciones realizadas

1. Se fijó una matriz contractual exacta del 1 al 17.
2. Preparación y envío a GINPIX son pantallas distintas.
3. Incorporación y seguimiento se conservan como capacidades adicionales del
   producto, sin alterar el recorrido mínimo de RRHH.
4. Las referencias internas, versión y huella del flujo se conservan en un
   bloque técnico plegado. La cabecera prioriza número, centro, categoría,
   modalidad, procedimiento, periodo, coste, unidad y estado.
5. El capturador usa únicamente área visible, elimina animaciones y foco
   accidental, y asienta cada vista bajo la cabecera fija.

## Herramienta reproducible

```bash
python3 scripts/capturar_contratacion_rrhh.py \
  --ejecutable-navegador /snap/bin/chromium \
  --salida var/revision-web-contratacion-rrhh
```

La herramienta solo acepta un origen local, exige la cabecera técnica del modo
de presentación aislada y falla si detecta:

- desbordamiento horizontal;
- desplazamiento horizontal;
- enlace de salto visible por foco accidental;
- cookies;
- `localStorage` o `sessionStorage`;
- errores JavaScript o de consola.

## Resultado local

| Evidencia | Resultado |
|---|---:|
| Pantallas contractuales | 17 de 17 |
| Capturas `1536 × 1024` | 17 de 17 |
| Capturas `1440 × 1000` | 17 de 17 |
| Capturas `1280 × 900` | 17 de 17 |
| Total automático | **51 de 51, VERDE** |
| Pruebas del capturador | **5 de 5** |
| Pruebas JavaScript focales | **33 de 33** |
| Pruebas JavaScript del portal | **272 de 272** |
| Paquetes Go de contratación con carrera | **7 de 7** |
| Suite Go completa y `go vet ./...` | **VERDE** |

La inspección del mosaico de las diecisiete capturas confirma navegación
persistente, tipografía y colores comunes, ausencia de huecos superiores
artificiales y una jerarquía administrativa coherente con la referencia.

## Límites que permanecen

- La presentación usa datos y efectos sintéticos, aislados de los manifiestos
  productivos.
- Algunas tareas aparecen en consulta cuando el perfil sintético no posee la
  competencia exacta. Esto acredita denegación predeterminada; la aceptación
  funcional deberá recorrer cada pantalla con el rol competente.
- La interfaz definitiva necesita la composición O2-07 y la API interna real
  O2-08 antes de poder cerrar O2-09 y ejecutar el E2E O2-10.
- RRHH debe aceptar campos, textos, orden operativo y densidad. Sistemas,
  Seguridad, DPD, Intervención y Archivo mantienen sus puertas propias.

## Veredicto

**VERDE local para continuar la revisión visual y funcional con RRHH.**

**NO-GO productivo** hasta conectar la vertical real, ejecutar pruebas E2E y
obtener las aprobaciones organizativas exigidas.
