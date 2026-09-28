# Ensayo focal B56

Ejecutar `bash scripts/rrhh_auditoria_bolsa56/probar_pg18.sh` desde el worktree del candidato. Requiere Docker y la imagen local `postgres:18.4-alpine`. Crea un PostgreSQL sin red con datos sintéticos y lo elimina al salir.

Comprueba ROLLBACK de UP y DOWN, instalación UP, rechazo de doble UP, dos actuaciones de situación, dos de contacto, motivo minimizado en cada fila `cambio:`, cursor de límite 1, ACL, rol ejecutor nominal, denegación de acción ajena y recuperación tras reinicio. El DOWN se ensaya solo en ROLLBACK antes de crear historia.

La preimagen incluye un **doble del consumidor AD3** para aislar la proyección SQL B48/B56. Por ello este ensayo no acredita firma, concesión ni criptografía V3 reales, y tampoco demuestra la instalación sobre una base conservada ni el recorrido HTTP. Esas comprobaciones requieren la integración V3 y su revisión independiente.
