# F0-27: arranque cuando Contratación temporal no se compone

Esta entrega no añade configuración ni SQL. No requiere actos de administración ni cambios de datos.

En la copia local de la principal, arrancar la candidata con la configuración completa de Contratación temporal y comprobar el caso de `docs/producto/FLUJO_APP.md` §4.0. Reiniciar el proceso y repetir la consulta del expediente y la sesión. Después, detener ese proceso y arrancar el mismo binario en la copia con un entorno separado que omita la configuración de CT. Conservar intacta la configuración completa para recuperarla. No modificar la configuración de la principal para este ensayo.

Con CT disponible, sus rutas deben conservar las respuestas y los permisos anteriores. Sin CT, `/livez` responde 200, `/healthz` responde 503, `/portal-empleado/` se sirve y `/api/vec/session` responde a una persona con certificado de desarrollo. La consulta pública de convocatorias y las consultas públicas de Personal siguen disponibles si sus propias fuentes son válidas. Las rutas de CT y las capacidades que consumen su autoridad responden 503 con `servicio_no_disponible`. Una petición sin certificado no supera mTLS.

El registro técnico anota `contratacion_temporal_no_disponible`, el componente y una clase de causa cerrada. No incluye textos libres del error, rutas privadas ni conexiones. La falta del borrador de llamamiento se clasifica como `borrador_llamamiento_no_disponible`. Un fichero de reglas de CT ausente o las preferencias de Usuarios inválidas también dejan disponibles el portal y la sesión; sus funciones quedan cerradas.

La raíz compartida impide montar por separado Bolsa RRHH, Documentos, Dietas, Cronos, Personal empleado, Usuarios y la consulta de Auditoría cuando falla CT. Las consultas internas de Calendarios y Reglas tampoco conservan su frontera de autorización y quedan cerradas. El ensayo completo con y sin CT, incluida la recuperación tras reiniciar, corresponde a la candidata integrada por dirección.

La puerta de calidad de la ejecución `38816882035` sobre `c72aee774` falló en dos pruebas que aún exigían detener el proceso sin PostgreSQL de CT: `TestServidorDesarrolloConDocumentosActivadoSinGobiernoNoArranca` y `TestProcesoInternoSigueLaComposicionDeSiempre`. Sus casos ahora comprueban el contrato de F0-27: servidor con mTLS, portal disponible y rutas dependientes en 503. La prueba focal de ambos casos pasó con `go test -p 6 ./internal/app/bootstrap -run 'Test(ServidorDesarrolloConDocumentosActivadoSinGobiernoAislaFuncion|ProcesoInternoSinCTConservaPortalYSeguridad)' -count=1`.

## Resultado local del 16/10

El binario de `9f1a11b1c` se ensayó con material ficticio y PostgreSQL 18 propios,
sin instalar SQL nuevo ni cambiar permisos. Con CT completa, la consulta del
cuadro respondió 200. Tras detener la aplicación, reiniciar PostgreSQL y volver
a arrancar, los diez expedientes de la página conservaron exactamente sus datos;
solo cambiaron el instante de consulta y el cursor temporal.

El mismo binario arrancó sin las conexiones de CT y, por separado, con el
archivo de reglas de CT ausente. En ambos casos, portal, sesión, catálogo público
y `/livez` respondieron 200; las rutas dependientes devolvieron 503 con
`Cache-Control: no-store`. Sin certificado, TLS rechazó la conexión. `/healthz`
respondió 503 también con CT completa: esta copia no instala un comprobador de
preparación y no se presenta ese resultado como disponibilidad de las funciones.

Chrome conservó la sesión, el menú y el título al abrir Peticiones de personal
temporal sin CT, con un aviso y Reintentar. El aviso reutiliza la traducción
específica existente, que identifica la función y remite a Informática si el
problema continúa. No se añaden textos ni almacenamiento en el navegador.

Las aplicaciones se ejecutaron sucesivamente. El LOGIN dedicado de auditoría
tiene un límite de conexiones: arrancar dos composiciones completas con el
mismo LOGIN agotó ese límite en este ensayo. Se detuvo el primer proceso antes
de repetir el arranque; no se amplió la concesión ni el límite.

Las pruebas focales de composición, cadena TLS, cierres de rutas y telemetría
pasaron. El grafo de recursos e idiomas pasó sus once casos. Semgrep local no
encontró problemas en el cambio. Gosec no señaló los archivos Go modificados,
pero su análisis de bootstrap es parcial por errores de carga en otros archivos;
conserva 17 avisos anteriores. En telemetría hay un aviso G115 previo en un
archivo sin cambios, sin errores de carga. El ensayo no sustituye el recorrido
completo de la candidata integrada por dirección.
