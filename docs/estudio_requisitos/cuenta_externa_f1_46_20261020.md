# Cuenta externa propia del aspirante: comprobación de F1-46

Fecha: 20 de octubre de 2026. Código comprobado: `main@fb5200b68`.
La rama de esta entrega solo corrige documentación; no añade Go, SQL,
permisos ni configuración. Dirección decide el cierre de la tarea.

## Qué se reutiliza

La cuenta externa de #1112, la publicación nominal de confianza de HZ38
(AD263/265) y la autoridad documental integrada con #1147 (AUT99/100,
AD277/281/284). La PR #1140 fue rechazada por dirección y no se amplía.

El montaje combinado exige cuenta y perfil externos distintos de los internos,
conservando persona, candidatura y certificado. Las rutas propias pasan por
la autoridad externa antes de la raíz interna; sus permisos no se suman.
La publicación de confianza usa un LOGIN nominal y su conexión administrativa
se entrega al preparador separado del proceso web.

El catálogo `motivos_autorizacion_justificante_renuncia_bolsa` está admitido
por AUT99 para subir y descargar el justificante propio. Su decisión exige
acción, recurso, finalidad y actor exactos. AD284 usa confianza y auditoría
externas para el aspirante; la descarga de RRHH conserva su autoridad interna.
Las dos revisiones independientes del código integrado no encontraron un fallo
que exigiera otra implementación de esta capacidad.

## Reproducir en una copia nueva

Seguir [la receta de copia local](../../deploy/principal/LEEME_copia_local.md),
apartados 2 y 4. El volcado de origen solo se lee. Los productores de roles y
datos se copian al material privado propio y reciben conexiones nuevas.
No se reutilizan credenciales, planes ni recibos de otro equipo.

Con `VEC_COPIA_ORIGEN`, `VEC_COPIA_ROLES_SCRIPT`, `VEC_COPIA_DATOS_SCRIPT`,
`VEC_COPIA_MATERIAL_RRHH` y `VEC_COPIA_PORTAL_ESTADO` apuntando al material
privado correspondiente, desde la raíz de la rama:

```sh
deploy/principal/copia_local.sh crear --equipo o --instancia f146-20261020 \
  --puerto-pg 55476 --puerto-app 18476 --puerto-smtp 11076 \
  --modulos --portal-externo --sin-inscripcion \
  --rrhh-original "$VEC_COPIA_MATERIAL_RRHH" \
  --lista-sql deploy/principal/lista_sql_main_tras_hz37.txt
```

Los puertos deben estar libres. La creación comprobada duró 110,31 segundos
con 131 migraciones de esa lista. PostgreSQL se creó con memoria limitada a
2 GiB, eliminación del contenedor al parar y sin reinicio automático.
La confianza externa publicada dura 89 días en esta copia de desarrollo.

1. Con el certificado del candidato creado por el preparador, consultar
   `GET /api/vec/bolsa/mi-bolsa` y abrir Mi Bolsa en Chrome del sistema.
2. Repetir esa consulta con el certificado de RRHH: no debe autenticar como
   aspirante externo.
3. Con el LOGIN nominal de motivos y TLS verificado, consultar en una
   transacción de solo lectura el motivo publicado de Mi Bolsa. Consultar
   también un catálogo ajeno y comprobar su rechazo.
4. Reiniciar aplicación y PostgreSQL con la orden propia de la copia:

   ```sh
   deploy/principal/copia_local.sh reiniciar --equipo o --instancia f146-20261020
   ```

5. Repetir la consulta de Mi Bolsa con el mismo certificado del candidato.
   Conservar fuera de Git las respuestas, el registro del reinicio y las
   capturas. No publicar certificados, conexiones ni referencias personales.

## Resultado y límites

| Comprobación | Resultado |
| --- | --- |
| Mi Bolsa con certificado del candidato, antes del reinicio | 200; esquema de Mi Bolsa v1; participaciones vacías |
| La misma consulta después de reiniciar aplicación y PostgreSQL | 200; mismo esquema y participaciones vacías |
| Mi Bolsa con certificado de RRHH | 401 en la autenticación externa |
| Motivo de Mi Bolsa publicado, por su LOGIN externo nominal | `true` |
| Catálogo ajeno, por ese LOGIN | Rechazado con `42501` |
| Motivo documental del justificante en esta copia | No está publicado; no hay comprobación positiva de ese motivo |

La cuenta externa nueva aún no tiene participaciones. Este ensayo comprueba
el acceso propio y su recuperación, y no acredita inscripción, aceptación,
renuncia ni subida de PDF. El motivo documental está soportado y revisado en
el código; publicarlo con sus valores exactos y disponer del permiso, almacén
y antivirus son precondiciones del [recorrido del justificante](../../deploy/principal/justificante_renuncia_activacion.md).

El preparador usa admisiones y asignaciones ficticias declaradas como datos de
copia. No son actos de Administración ni resuelven el productor gobernado de
admisiones de F0-21. El rol de la copia tiene su tarea propia, F1-115.
No se ha tocado cidonia ni se acredita aquí una instalación compartida.
