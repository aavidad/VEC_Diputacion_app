// Package confianzadocumental implementa el conector PostgreSQL de ejecucion
// documental atestada V4. El nucleo solo conoce el puerto neutral de ports; pgx,
// SQL, el socket Unix y el material criptografico permanecen en este adaptador.
// Deliberadamente no se exponen fabricas de Servicio, RaizPublicaFijada,
// ConfiguracionConfianzaFijada ni de la autoridad opaca.
//
// La composicion PostgreSQL carga la confianza durable desde una identidad
// emisora aislada y entrega al proceso ejecutor una capacidad HMAC efimera por
// socket Unix. Ningun llamador puede aportar raices, claves, verificadores o
// repositorios arbitrarios. Su apertura productiva sigue condicionada a que
// Sistemas mantenga segregadas ambas credenciales y el secreto operativo. Un
// conector Oracle futuro implementara el mismo puerto sin modificar application.
package confianzadocumental
