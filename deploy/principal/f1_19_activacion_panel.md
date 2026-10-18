# F1-19 C2: activar la derivación de nombramientos

Estado: en preparación; faltan permiso, ensayo positivo y arranques con y sin configuración.

## Antes de activar

1. Instalar el prefijo completo de la lista de la candidata, incluida la fila
   de R en AD291 para acción, perfil y audiencia, y su consumidor AD292. Aplicar
   B135 y después CT234 una sola vez. Cotejar recibos, definiciones y permisos.
2. Admitir en el catálogo vigente la entrada ordinaria de la acción
   `contratacion_temporal.nombramiento.derivar`: módulo `contratacion_temporal`,
   recurso `nombramiento_derivado_ct`, finalidad
   `gestionar_contratacion_temporal`, campos vacíos, obligación `auditar` y
   garantía alta con certificado. Conservar referencia, versión y huella de la
   entrada admitida por AUT58/AUT61.
3. Dos administradores distintos publican un rol propio para el perfil
   `ct_nombramiento_derivado` por las propuestas y cierres de Gobierno de roles
   (`/api/admin/perfiles/v1/gobierno-roles/`). La versión publicada y habilitada
   debe tener solo esa concesión. Publicar el motivo por el circuito V2 y
   asignar el perfil a RRHH por el lote ordinario AUT44, con
   ámbitos `organizacion_ref` y `unidad_ref` de la unidad gestora del padre.
   No hay plan de actos preparado para este corte; revisar y conservar recibos.
   Cuenta y certificado deben ser de la misma persona;
   ningún permiso del padre se hereda.

## Archivo privado opcional

Plantilla de `identidad/nombramientos-ct.json` bajo `VEC_DEVELOPMENT_MATERIAL_DIR`:
sustituir los ejemplos por valores publicados; no sirve como configuración activa.

```json
{"esquema":"vec.ct.nombramientos.activacion.v1",
 "derivacion":{"perfil_ref":"PERFIL_PUBLICADO",
  "rol":{"rol_id":"ROL_PUBLICADO","version":1,"huella_sha256":"HUELLA_PUBLICADA"},
  "motivo":{"catalogo_id":"CATALOGO_PUBLICADO","catalogo_version":1,
   "catalogo_huella_sha256":"HUELLA_CATALOGO","entrada_clave":"MOTIVO_PUBLICADO"}},
 "fuente_autorizacion_dsn_file":"identidad/fuente-nombramientos.dsn",
 "motivos_autorizacion_dsn_file":"identidad/motivos-nombramientos.dsn"}
```

Guardar JSON y ambas DSN como archivos `0600` con rutas relativas distintas.
El archivo no concede permisos. Si falta o es inválido, solo la derivación debe
responder 503. Reiniciar vigilando salud, sesión RRHH y ruta.
Ensayar POST y recuperación con la misma clave según `f1_19_nombramientos_derivados.md`; cotejar recibo, hijo, auditoría y reinicio.
El productor interno `fef11a8c3` sigue sin ensayo con este corte.
