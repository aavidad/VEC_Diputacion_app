# Comprobación de la ficha GINPIX desde Personal

14/10/2026. La ficha usa el alta B2 confirmada y sus recibos históricos de
Personal. El caso único de `FLUJO_APP.md` mantiene el orden
17 → 18 → 16a → 16b → 19 → 20 → 21.

La revisión independiente dio GO estático para el servidor, el cliente y la
vista. Pasaron las focales Go y JavaScript de descarga, montaje, denegación,
conflicto, recuperación histórica y cadena de caché. El servidor y el cliente
usan el nombre `ficha-ginpix-personal.json`; la descarga conserva sus bytes
originales y no representa una transmisión automática.

El binario ensayado desde `188a0d749` tiene SHA-256
`f67779dcfd2cd0eee6d68d08a35e6f1946017ce1c35282cd4feff837bc2cf0ec`.
Con y sin B2, `/livez` y `/portal-empleado/` respondieron 200 con certificado.
Con B2, la consulta de un expediente existente sin incorporación devolvió
409 a RRHH y 403 al centro. Ninguna respuesta entregó un archivo. El preflight
de la copia terminó con código 0 y catorce comprobaciones favorables.
`/readyz` conserva el 503 por el comprobador nulo de la composición previa.

`cargo_manuel.sql` se aplicó sobre el catálogo de HZ37 y añadió la revisión
5 con «Subdirección de Informática». La repetición conservó cinco revisiones.
Se decodificó el resultado con `CatalogoConfigurable`, se validó y se cotejó
su huella canónica de Go con la guardada en PostgreSQL: coinciden. El guion
conserva las revisiones anteriores y no crea cargos, permisos, actos o recibos.

La descarga positiva después del alta, la confirmación manual de GINPIX, el
cese y las consultas posteriores al reinicio siguen pendientes del recorrido
de P con las firmas reales. Las comprobaciones anteriores no sustituyen ese
recorrido ni acreditan confirmación externa de GINPIX.
