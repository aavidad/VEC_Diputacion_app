# Ensayo B57: causas catalogadas

Ejecutar `bash scripts/rrhh_motivos_b57/probar_pg18.sh` desde el worktree del
candidato. Requiere Docker y `postgres:18.4-alpine`; crea un contenedor sin red
ni volumen Docker y lo elimina al finalizar.

El ensayo instala AD3-101/102/103 reales sobre su preimagen SQL oficial y
B48/B56/B57 reales. La preimagen añade dobles explícitos de B12/B16/B19/B35 y
del consumidor de auditoría. Antes de probar la publicación B57, reemplaza el
consumidor AD3-102 por un doble declarado en el contenedor; su propio ensayo
PG18 separado comprueba la fachada real. Comprueba UP/DOWN en ROLLBACK,
doble UP, ACL, situación y contacto con catálogo v1→v2, publicación y replay
del catálogo, sidecars, consulta auditada con causa y cambio, historia libre
reservada, retirada posterior de etiqueta sin reescribir recibos, denegación
de material desacoplado y replay tras reiniciar PG18.
DOWN se ejecuta solo antes de crear historia.

La B56 final se toma del árbol integrado; antes de integrarla puede sustituirse
con `B56_UP_SQL=/ruta/000056_motivo_traza_auditoria_participacion.up.sql`.

No acredita criptografía, concesión V3, B12/B16/B19/B35, HTTP ni una instalación
sobre bases con historia. B56 final debe existir en el árbol base; antes de
integrarlo puede apuntarse a su archivo con `B56_UP_SQL=/ruta/000056...up.sql`.
Los dobles AD3 no acreditan V3 real; AD3-102/103 conservan ensayos PG18
separados.
B19 v2 cubre pausa, reactivación y exclusión manual. B26/B37 (sanciones) y
B42 (efecto CT) insertan B2 por reglas propias: sin migración de esos circuitos
sus motivos quedan reservados en auditoría y no catalogados por B57. El 4.12
sigue PARCIAL.
