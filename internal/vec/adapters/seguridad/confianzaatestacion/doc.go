// Package confianzaatestacion aplica perfiles institucionales versionados de
// confianza a atestaciones de autorizacion VEC-AD-2 y VEC-AD-3. Verifica
// claves fijadas, audiencia, vigencia, revocacion y COSE. En V3 separa ademas
// la prueba publica, el emisor HMAC de capacidad breve y su verificador
// consumidor. Ningun valor concede por si solo permiso para mutar un agregado:
// el adaptador transaccional debe revalidar y consumir la capacidad.
package confianzaatestacion
