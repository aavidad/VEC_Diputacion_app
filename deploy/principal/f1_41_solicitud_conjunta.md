# F1-41 — Solicitud conjunta con su recibo y reintento (servidor)

Dos migraciones, en este orden y después de BC12 y B142:

1. `bolsa_convocatorias/000013_comprobacion_inscripcion_conjunta_v1` (BC13):
   comprobación de publicación para la solicitud conjunta. Solo crea una
   función; no toca BC9 ni BC12.
2. `bolsa_llamamientos/000146_solicitud_conjunta` (B146): exige BC13 con su
   huella exacta. Retira la unicidad persona y convocatoria (contenido de B105,
   que no se instala aparte), crea las dos tablas de la presentación conjunta,
   reconstruye `solicitar_inscripcion_v1` y el proyector del aspirante desde
   sus definiciones instaladas (B96 y B142) y crea
   `solicitar_inscripcion_conjunta_v1`.

Ambas comprueban preimagen y posimagen (huella, propietario, ACL, configuración
y OID). No tienen DOWN ni se reaplican. Las líneas están al final de
`lista_sql_main_tras_hz37.txt`.

No hace falta configuración nueva: la ruta y la autoridad V3 son las de la
solicitud por categoría. Una convocatoria conjunta de prueba se carga como
dato de la copia, igual que en F1-40.

Pruebas: `bolsa_llamamientos/pruebas_sql/b146_solicitud_conjunta.sql` (en
`ensayar.py --pruebas`), dentro de una transacción revertida y con una fachada
del consumo V3. No acredita V3 real.

Pendiente: la pantalla de la aspirante (elegir varias categorías y ver el
recibo conjunto) va en el corte siguiente.
