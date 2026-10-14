# Dato ficticio de Mi Bolsa de Antonio en HZ39

La identidad de Antonio ya apunta al candidato de la bolsa vigente, pero su asignación de Mi Bolsa conserva la referencia anterior. Dirección autorizó rehacer datos ficticios en la **copia P** para terminar el ensayo. [datos_antonio.sql](datos_antonio.sql) corrige solo el ámbito de esa asignación y recalcula la huella con la función canónica instalada. No crea actos, decisiones, firmas ni recibos.

La receta se limita a la copia local P. Se ejecuta con la aplicación parada y como superusuario de esa copia:

```sh
docker exec -i -u postgres vec-codexp-hz39-pg-20261014 \
  psql -p55639 -U postgres -dpostgres -X -v ON_ERROR_STOP=1 \
  < deploy/principal/candidata_hz39/mi_bolsa/datos_antonio.sql
```

Antes de escribir, comprueba que el vínculo vigente de `per_8014599251fc9a7aff11113c502e3317` apunta a `can_Vxd6XlZE9ctNzw38AUSmgQbCcRyoFKhaojDaFsEKC1M`. Exige la asignación interna v3 del perfil `prf_8d0fed2990714f97e78b0afc21b337e7`, el ámbito anterior y su preimagen exacta `f5920ae751f66b48755303ddcc1d7b799820fad3eebe7d4c2f59220be0632586`. Si alguno cambia, revierte toda la transacción. La huella objetivo, calculada en lectura con `vec_autorizacion.canon_asignacion_perfil_admin_v1`, es `f3f9955e9ba0a90f220f018de5c746ec4bfb4e88c782ca3eb7d95bd507026bc9`.

El guion suspende el disparador de inmutabilidad de esa tabla dentro de la transacción, modifica el documento y su huella, y reactiva el disparador antes de confirmar. La segunda ejecución reconoce el dato objetivo y termina sin cambios. No modifica el puntero, el perfil, el principal, el rol, la vigencia ni las columnas de procedencia. El acto anterior no acredita esta corrección. Las 292 decisiones previas que referencian la versión 3 conservan sus huellas originales y dejan de ser evidencia probatoria coherente con la nueva huella. Esta copia sirve solo para continuar la demostración; el guion no sirve para cidonia ni producción.

Tras ejecutarlo, Dirección reinicia la aplicación y comprueba en Chrome que Mi Bolsa muestra únicamente la participación de Antonio en la bolsa vigente. La consulta SQL final debe devolver el candidato nuevo, `huella_valida=true` y `guarda_inmutabilidad_activa=true`. Si el arranque sigue avisando `pendiente_provision`, se detiene el recorrido: puede haber otra diferencia de rol o control, y esta receta no la corrige. Para una instalación que conserve historia, el cambio de ámbito requiere un acto de Administración con preimagen, nueva versión, auditoría y recibo.
