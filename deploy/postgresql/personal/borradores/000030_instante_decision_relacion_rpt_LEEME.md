# Instante de la decisión en la lectura RPT

Personal30 corrige una denegación observada en Personal27: la capacidad Go
escribe `…71397Z` y la decisión `…713970Z`. Representan el mismo instante,
pero la comparación textual los rechazaba.

La migración añade una variable local `cap_dec_hasta`, convierte la fecha
junto a `dec_hasta` dentro del manejo de errores existente, rechaza nulos y
valores no finitos y compara ambos instantes. Conserva los bytes del material,
la firma, los controles de vigencia actuales y el orden consumo → lectura →
recibo y auditoría en la misma transacción.

Exige el cuerpo instalado SHA256
`9e4733bcb994a3234d8261c107aeb4b49a9d04f9544302ac34f39c77c604a560`;
la postimagen prevista es `dd0b06eb3ecb2a8317054739ad01f6c363c725f477d7f98d345f01d83487a78f`.
Preserva OID, firma, propietario, ACL, configuración y dependencias. No crea
roles, tablas ni permisos; no modifica el archivo de Personal27 y no incluye DOWN.

La lista causal contiene sólo Personal30: AD154 → Personal27 → Personal30.
Personal27 y el runtime común con CA27 deben estar acreditados; ninguna
migración instalada se reaplica. Personal30 recibió dos GO sobre
`86ee0557827da7d4d61c5e00e85af0083209afbd` y se instaló una sola vez en el
clon. Los ocho casos focales pasaron, con OID, firma, propietario, ACL,
configuración y dependencias conservados.

El caso Go final del runner
`77e07cbd98541c9633b2dccbd16885331d51545f` confirmó `40001` a las
`2026-10-03T16:10:26.792Z`. No devolvió DTO ni añadió consumo o recibo; su
intento de error quedó confirmado aparte a las `16:10:26.811Z`, con la misma
correlación técnica y nominal `2a1f7be92b8296e9a69e0247a65be726`.
Se reutilizó el binario de `332f8eee392a33ebc73c65b6d9565eb8bde78ef5`, sin
recompilar ni repetir los seis casos Go anteriores o el SQL ya pasado. Su
SHA256 fue `15377c020be8eef54be86ecb461d60836257f2b5275de8f647542b03b539c3e5`.

Tras reiniciar PostgreSQL se conservaron seis recibos, 6.246 consumos,
6.246 auditorías confirmadas y quince intentos; snapshot SHA256
`a1fb605cb0cf6bad42ace3bedc3c152af2b7c8a748cbd545bf71d46a8f227235`.
La fuente Go revisada fue `5c5e305a397306612299a11741a26a0bdce1abbe` y CA27
procedió de `7f34c77014113bbb54069312e84dd1a62d7c8b6f`.

COSE Ed25519/HMAC, SQL, servicio y pools fueron reales; fuente, IdP/PDP e
identidades del fixture fueron sintéticos y declarados. No acredita HTTP,
mTLS, navegador, IdP institucional, instalación principal ni producción.
El montaje operativo conserva la dependencia de L para proceso/canal de
consumos permitidos. Se retiraron el clon, claves, binario y temporales
propios tras conservar actas y huellas.
