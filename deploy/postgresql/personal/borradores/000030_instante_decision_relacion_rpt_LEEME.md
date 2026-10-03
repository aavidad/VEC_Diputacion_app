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

La lista causal contiene sólo Personal30 y requiere Personal27 ya instalada.
Pendientes dos revisiones del hash final y ensayo coordinado en el clon. La
prueba focal comprueba formatos equivalentes y rechazos temporales; el caso
Go pendiente se reanudará con el binario conservado, sin repetir los seis
casos ya pasados ni atribuir firma legal, IdP o política de producción.
