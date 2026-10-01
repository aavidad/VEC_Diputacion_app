# Comprobar la fuente del gobierno de categorías RPT

Desde la raíz de esta rama:

```bash
python3 scripts/ensayos/rpt_gobierno_v3/preflight.py
```

El programa lee el árbol y devuelve un inventario JSON. El código de salida 2
significa **bloqueado**: este inventario nunca autoriza instalar SQL, habilitar
un perfil ni usar las rutas HTTP. Publica las huellas de Cat4/AD134 y señala
si faltan las fuentes CA21, AUT25 e IS10, o la composición de las rutas RRHH.
No usa el antiguo plan de 41 SQL ni la vía ADMIN: el gobierno pertenece a
RRHH, con personas distintas para proponer y aprobar.

Antes del ensayo de servicio falta integrar en orden la fuente B y A, CA21,
AUT25 e IS10, fijar un descriptor de catálogo aprobado por RRHH y su huella,
y revisar la postimagen de Cat4/AD134 sobre esa cadena exacta. Un descriptor
de demostración no concede permisos. La política High de IS10 es sintética de
desarrollo y no acredita identidad corporativa ni firma legal.

El ensayo posterior necesita tres identidades nominales: una persona propone,
otra aprueba y esta última confirma la versión. Debe comprobar también actor
incorrecto, concesión revocada, reintento con el mismo recibo, cambio de
versión, historia y auditoría. En PostgreSQL 18 se ensayarán la secuencia
causal, dos sesiones concurrentes, reversión y recuperación tras reiniciar.
Después, el navegador comprobará las rutas montadas y los recibos reales.
Hasta entonces no hay una operación de gobierno RPT disponible para RRHH.
