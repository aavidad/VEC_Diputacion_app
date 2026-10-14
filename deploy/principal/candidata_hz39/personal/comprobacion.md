# Comprobación local de Personal, cese y vuelta a Bolsa

14/10/2026. Código de alta y cese: `81c2839b9`; el cambio posterior del
preflight, `e507d2f86`, no modifica Go. Copia propia de HZ37 en PostgreSQL
18.4, con ACL de base y tipos del respaldo. No se ha consultado ni cambiado
cidonia.

Pasaron las once SQL de la candidata, AD266 y las ocho de
`personal/lista_sql.txt`, en orden. CT215 ya estaba instalada: se cotejaron
sus posimágenes y no se reaplicó. CT211 se ensayó con la huella final
`72bf2b495b24a133cd500a278e5993c41fee514223af37ccd07ca240b1753811`.
CT213 conserva `b2edb7fa…` y B115, `be356decc…`. La prueba SQL de B115 pasó;
comprueba contratos y condiciones, sin ejecutar un alta o un cese de negocio.

El binario de ese código tiene SHA-256
`2e6feaae10bf7f11c153d51952b864769f1d9320e48ccb2b1444189079ec064b`.
Sin configuración B2, la composición tardó 2,63 segundos; con B2, 2,98.
En ambos arranques, `/livez`, `/portal-empleado/` y `/bolsa/` respondieron
200 con certificado. `/readyz` devuelve 503 en ambas configuraciones porque
la composición existente entrega un comprobador nulo; no se presenta como
una comprobación favorable de todas las dependencias.

El preflight devolvió código 0 con catorce comprobaciones favorables:
certificados, conexiones nominales con TLS, selectores, fuente de Bolsa,
bolsa vigente, CT215, puesto 9/01, plaza 2026/0033, vacante, contexto del
centro y catálogos de Personal. La conexión del cese se carga por el fichero
privado explícito `--conexiones`; no es necesario copiar su valor al lanzador.
Se preparó el cliente M10 con el origen local y el certificado de RRHH, y
los cinco datos de la necesidad en su JSON privado. El aviso «expediente B2
no preparado» describe el estado inicial y no implica un acto ejecutado.

La copia requería fechas en la Plantilla derivada de RPT. Se prepararon las
revisiones 2 de Plantilla, plaza y cobertura que reproduce
`personal/datos_plantilla.sql`. No acreditan una aprobación oficial de
Plantilla. Después del corte temporal de un minuto, la consulta real de
vacantes y el preflight respondieron correctamente.

El recorrido de negocio, las firmas, la disponibilidad tras el cese y su
recuperación después del reinicio siguen siendo comprobaciones de P sobre
la candidata completa. Este ensayo de instalación y arranque no las sustituye.
