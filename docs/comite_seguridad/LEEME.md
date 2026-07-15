# Entregables para el Comité de Seguridad

El documento preparado para remisión es
`informe_validacion_arquitectura_seguridad.pdf`. La misma versión se conserva en
Markdown y HTML para facilitar su revisión, accesibilidad y mantenimiento.

Las cuatro infografías se encuentran en `diagramas/`. Cada SVG publicable tiene
una fuente Graphviz `.dot` versionable.

## Regeneración

```sh
python3 -m pip install -r docs/comite_seguridad/requirements.txt
python3 docs/comite_seguridad/generar_informe.py
chromium --headless --disable-gpu --no-sandbox \
  --allow-file-access-from-files --no-pdf-header-footer \
  --print-to-pdf=docs/comite_seguridad/informe_validacion_arquitectura_seguridad.pdf \
  file:///ruta/absoluta/al/repositorio/docs/comite_seguridad/informe_validacion_arquitectura_seguridad.html
```

La fecha de corte y la versión se mantienen en el Markdown y en el pie generado
por `generar_informe.py`; deben actualizarse conjuntamente antes de una nueva
emisión.
