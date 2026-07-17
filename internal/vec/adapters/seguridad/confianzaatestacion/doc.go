// Package confianzaatestacion aplica el perfil institucional de confianza a
// atestaciones de autorizacion VEC-AD-2. Verifica claves fijadas, audiencia,
// vigencia, revocacion y COSE, pero no concede por si solo permiso para mutar
// un agregado: el adaptador transaccional debe revalidar y consumir la prueba.
package confianzaatestacion
