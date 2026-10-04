# AD175: consumidor de exportación de servicios propios

AD175 sigue siendo un borrador. El núcleo se ha reanclado a la medición de K
posterior a AD173/174/176 en el clon L. Su definición tiene SHA-256
`6c22fdbb165a00c4f37cb2f7dbb7add4e939e5b0134c0c599b9519bfe3b86db9` y
su cuerpo `bbb932ef29375e88645fb524e6aae5fd3059d51cb0dfd9d4952ffd509470534c`.
El propietario es `vec_autorizacion_atestada_v3_propietario`, con EXECUTE sólo
para ese rol; conserva `search_path=pg_catalog, pg_temp` y `lock_timeout=2s`.

La sustitución añade el perfil técnico `exportacion_servicios_propios` a la
exclusión general, a la rama de LOGIN con un único grupo ejecutor de Personal y
al contrato nominal de acción, audiencia, campos y finalidad. Conserva las
otras familias, la resolución de origen de AD172 y la auditoría común de AD173.
No crea cuentas, membresías, perfiles funcionales, permisos humanos ni origen
técnico. La provisión positiva fija y CAS siguen a cargo de su autoridad.

La medición K del CHECK `auditoria_tipo_disjunto_v4` tiene SHA-256
`8346e28593ad3b35a2dc88de0023a41c8c4a89571b31ba6226d1b9bf44523e03`.
AD175 no modifica ese CHECK. Aún falta la definición y huella de
`clave_capacidad_version_audiencia_consumo_check` tras la misma cadena causal;
por ello su guarda conserva `NULL` y detiene la migración antes de cambiar el
núcleo o el catálogo de audiencias. Se comprueba otra vez bajo bloqueo antes de
alterar el catálogo. Tampoco se ha ensayado ni instalado AD175 o Personal32.

La comprobación estática toma el JSON medido y el borrador SQL:

```sh
python3 deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/verificar_ad175_preimagen.py \
  /ruta/privada/nucleo-post176-L.json \
  deploy/postgresql/autorizacion_atestada_v3/migraciones/000175_consumidor_exportacion_servicios_propios.up.sql
```

Comprueba huellas, ACL, configuración, tres anclas únicas y la inversión exacta
de la sustitución sobre la definición medida. No accede a PostgreSQL. Cuando K
entregue el CHECK de audiencias, Dirección fijará su huella, revisará el SQL
final con dos lectores independientes y ordenará el ensayo en el clon.
