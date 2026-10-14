# Candidata HZ39

Versión candidata en preparación. Se ensaya entera en una copia
de cidonia y, si sale bien, se instala en cidonia exactamente igual (regla 1 de
`AGENTS.md`, «Entrega y subida»). Caso de aceptación: `docs/producto/FLUJO_APP.md`,
apartado 4.0.

- **Base de partida:** lo instalado en cidonia (HZ37 + web HZ38, código
  `44218865a`; catálogo de acciones v6).
- **Código de la candidata:** el commit de `origin/main` que se fije al congelar
  (se anota aquí antes de instalar).
- **Nada se instala desde fuera de esta carpeta**: si falta algo, se añade aquí
  en una PR y se vuelve a ensayar.

## Orden de instalación

1. **SQL**: `lista_sql.txt`, en ese orden, cada una una vez y con
   `ON_ERROR_STOP`. Comprobar el SHA-256 de cada fichero antes de aplicarlo.
   La lista incluye AD266, después de AD265; AUT235 ya forma parte de HZ37.
   Al restaurar la copia, aplicar también las ACL de tipos y las ACL de la
   base entregadas por dirección. El dump de datos no conserva las ACL de
   la base: sin ellas faltan los permisos nominales de conexión y vuelve el
   permiso temporal de PUBLIC. No se reconstruyen a mano.
2. **Datos del recorrido**: `datos/` (guion repetible de K: centro 520, puesto
   9/01, plaza 2026/0033, bolsa de Administrativo).
3. **Entrada del aspirante**: Mi Bolsa interna (HZ37) para el recorrido
   principal, con Antonio disponible en la bolsa vigente. Los pasos e1–e5 son
   opcionales y se enseñan aparte. La inscripción RRHH queda apagada
   (`VEC_BOLSA_INSCRIPCIONES_ENABLED=false`), como en HZ37; su lector nominal
   se resolverá después del ensayo principal. La confianza externa (#1120, `deploy/principal/confianza_externa_hz38.md`) solo si su
   ensayo valida el acceso y el recorrido; decisión de corte el 13/10 a las
   17:00 y se anota aquí la vía elegida.
4. **Actos de administración**, por la vía de gobierno existente y con dos
   administradores distintos:
   - inscripción RRHH (lector técnico de Carmen) cuando la vía del paso 3 lo
     permita, con reinicio vigilado;
   - plan de firma v2 de cinco pasos (`scripts/operacion/firma_ct_tres_pasos/`):
     crear y publicar; después `identidad/firma-vec.json` con los valores
     resultantes y reinicio.
5. **Arranque vigilado** con vuelta atrás automática.
6. **Recorrido** con el guion de P (`deploy/principal/ensayo_candidata/`):
   los pasos principales del caso, reinicio de app y PostgreSQL y repetición
   de lecturas; mismos recibos y sin duplicados.

## Estado del ensayo

| Pieza | Estado |
| --- | --- |
| SQL (12) | Once SQL iniciales y AD266 aplicadas una vez, sin error, en la copia nueva de P; ACL de tipos y base restauradas |
| Datos del recorrido | Recetas #1135 aplicadas: plantilla existente, Antonio disponible y primer turno; manifiesto con la nueva referencia de candidato |
| Vía del aspirante | Mi Bolsa interna; inscripción RRHH opcional apagada por dirección |
| Plan de firma y cinco firmas | Pendientes (H); el registro y el verificador siguen activados en la configuración |
| Arranque | Composición en 3,443 segundos y salud HTTPS con certificado; firma y Mi Bolsa activas, inscripción RRHH apagada |
| Recorrido completo y reinicio | En curso; paso 1 pendiente de documento de crédito y de completar los campos del guion |

Lo que no se haya ensayado en la copia no se instala en cidonia.

El ensayo se retomó el 14/10 desde el respaldo HZ37, en una copia nueva.
La restauración terminó antes de aplicar la lista; un intento previo iniciado
antes de terminar la restauración se descartó entero. Se conservaron las
conexiones separadas, TLS y las comprobaciones de permisos. Las conexiones y
la escucha se trasladaron a los puertos de la copia, accesibles solo desde
loopback, y se preparó el directorio privado del almacén documental.

Chrome del sistema abre la pantalla del centro sin errores de JavaScript ni
de red, usando el transporte HTTPS existente del guion. La búsqueda espera
el código RPT `9`; `9/01` es el código del puesto individual. El formulario
también exige plaza y datos presupuestarios. El catálogo devuelve la lista
de documentos vacía, por lo que todavía no permite elegir el documento de
retención de crédito. Ninguna petición nueva se ha presentado.

La congelación del código, el plan nominal de firma, el tramo Personal/cese
y la recuperación completa tras reiniciar siguen pendientes. El arranque y
las lecturas iniciales no acreditan esos pasos.
