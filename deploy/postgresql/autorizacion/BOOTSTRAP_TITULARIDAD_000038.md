# AUT38 — arranque desde titularidad sin perfil

AUT38 adapta sólo la fachada CA que publicó AUT37. No vuelve a provisionar
cuentas, Personas, roles, certificados ni fuentes. La función de arranque y el
plan mantienen `registrar_bootstrap_central_admin_v3(text,text)` y el esquema
V3. Las cuentas del plan son las referencias reales entregadas por IS15.

La precondición exige AUT37, CA33, IS15 y Personal31. Antes de cambiar el cuerpo,
la migración exige SHA256 de `prosrc`
`a2dfb507381b22a0d654083e3ca60582bba18a77aedc626493fe15f624ab873a`,
propietario CA, función definer, configuración fija y ACL exclusiva CA/AUT.
La definición nueva es literal, con `CREATE OR REPLACE`; después coteja toda
la metadata de `pg_proc` salvo `prosrc`. Conserva OID, firma, propietario,
configuración y ACL. No modifica AUT37 ni concede permisos nuevos.

Si hay un vínculo legado válido, la preimagen conserva sus cinco claves:
`esquema`, `contexto`, `sistemas`, `ambitos` y `organizaciones`. Si no lo hay,
CA comprueba la titularidad CA33 sin perfil. Exige cuenta y Persona reales,
versiones uno, procedencia exacta, estado y vigencia de ambas proyecciones,
fuente de titularidad original y operación sintética aprobada. Coteja el recibo
CA original y llama a la fachada IS para recuperar su recibo real; AUT no lee
tablas ajenas. Retiene la barrera de generación antes de leer los punteros.

La preimagen añade `titularidad` sólo cuando utiliza esa fuente. Conserva la
fila original y una sección `fuente` con operación, SHA del plan, aprobación
y referencias/huellas de los recibos CA e IS. No contiene HMAC, secretos,
nombres civiles ni perfiles inventados. El enum instalado de procedencia no
convierte el alcance `sintetico_declarado` en una fuente institucional.

Dirección debe preparar otro plan V3 y su aprobación externa: las huellas de
Persona y la preimagen conjunta cambian al incluir la titularidad. No sirve
reutilizar la aprobación de un plan anterior ni fijar una huella arbitraria.
El arranque conserva CAS, roles fijos, ámbito organizativo, unidad cuando la
exige el catálogo, auditoría común de bootstrap, recibo y continuidad 2+1 de
AUT37. El replay durable utiliza su recibo original y no vuelve a crear perfiles.

La auditoría de invocaciones/replay/rechazos del consumidor de arranque queda
para su sucesora común. Esta pieza no presta la familia de auditoría de fuentes
para atribuirle una acción de bootstrap. No acredita aún gestión completa,
arranque en producción ni cobertura de errores externos.

## Comprobación preparada

`pruebas_sql/bootstrap_titularidad_000038.sql` recibe el plan sintético privado
en el GUC de sesión `vec.ensayo.plan_bootstrap`, cargado por parámetros enlazados.
Comprueba la fachada propietaria: preimagen con titularidad real, repetición
idéntica, versión divergente, cruce de Personas existentes, retirada sintética
de una proyección y ausencia de perfiles/vínculos nuevos. Todo termina en
ROLLBACK; la retirada sólo simula el estado negativo y no acredita una
revocación administrativa autorizada.

La prueba de población AUT37 existente se reutiliza después del arranque real:
tres administraciones efectivas, dos Personas de Aplicación y Sistemas separado.
Dirección debe confirmar el positivo, replay, negativas y recuperación tras
reinicio con el consumidor real antes de dar el arranque por cerrado.

Personal31 exige una unidad publicada por su autoridad. El estado post-H9
indicado por Dirección todavía carece de esa fuente. AUT38 mantiene la guarda:
no elimina la dimensión ni inventa una unidad para conseguir un positivo.
La fuente de Personal se resuelve de forma independiente antes del ensayo 2+1.
