# Censo de los cargos de firma

F1-118 corrige la admisión del catálogo después de publicar un cargo con
`vec-publicar-cargos-firma`. El documento SQL del cargo omite `retirada_en`;
la representación canónica de Go incluye `0001-01-01T00:00:00Z`. El cotejo
exacto de AUT58 y AUT61 los trataba como documentos distintos.

AUT106 reconstruye únicamente
`vec_autorizacion.normalizar_rol_catalogo_acciones_admin_v1(jsonb)`, desde la
definición instalada en la copia de main. Completa ese instante solo para un
rol con estado `publicada` que no contiene la clave. Un instante explícito,
incluso nulo, se conserva. Los roles retirados y los documentos sin estado no
reciben ese campo. El normalizador sigue quitando únicamente las listas
vacías de campos y obligaciones que ya quitaba.

La corrección protege el cotejo completo: no elimina concesiones, campos
extraños, retiradas ni controles. La prueba SQL contrasta las versiones de rol
almacenadas con la normalización anterior y comprueba los casos negativos.
No cambia el publicador de cargos, las versiones o huellas guardadas, los
permisos ni las asignaciones de las personas.

## Activación

CONFIG NUEVA: ninguna. No hay selectores ni ficheros de arranque nuevos.

La puesta al día con las fusiones del 20/10 sigue el orden de dirección:
F1-95 B (AD301/BC17), F1-105 (AD303) y después F1-118 (AUT106).
No se da por actualizada esta entrega hasta que estén integradas las dos
anteriores. Al unir main, se conservan sus entradas y pruebas; AUT106 queda
al final de la lista y se ajusta el recuento al número real de pruebas SQL.
Las pruebas del ensayador y de la copia local se ejecutan desde la raíz.
El ensayo descrito más abajo corresponde a la base anterior y conserva su
alcance; no acredita por sí solo la candidata que resulte de esa integración.

1. Crear una copia propia con la orden completa de
   `LEEME_copia_local.md`, incluidos `--modulos`, `--rrhh-original` y los
   preparadores privados propios. Instalar la lista única de main en su orden.
2. Aplicar AUT106 una sola vez, al final de
   `lista_sql_main_tras_hz37.txt`. Su guarda exige PostgreSQL 18 y la huella,
   el propietario y los privilegios exactos de la función instalada.
3. Ejecutar la prueba
   `deploy/postgresql/autorizacion/pruebas_sql/000106_canon_retirada_cargos_firma.sql`.
4. Publicar un cargo ficticio mediante la herramienta existente y preparar
   el censo completo desde Go. Presentar su plan aprobado por el circuito
   técnico de copia, con un LOGIN exclusivo y la configuración protegida.
   Conservar el plan y el recibo; repetir la misma operación debe recuperar
   el original. Ningún dato de copia acredita una aprobación personal real.
5. Reiniciar la aplicación y PostgreSQL propios. Consultar la admisión exacta
   y repetir la misma operación; cotejar recibo, fecha y ausencia de duplicados.
   Comprobar también el arranque sin AUT106: el portal sigue atendiendo y la
   admisión del cargo reproduce el rechazo de censo previo.

No reaplicar una migración instalada ni ejecutar DOWN. Ante una guarda
incompatible, conservar el registro y revisar la definición real. No
reescribir los roles para sortear el cotejo.

## Comprobación de esta candidata

En una copia propia de `main@fb5200b68`, la herramienta existente publicó
un cargo nuevo cuyo documento omitía `retirada_en`. El plan de admisión,
generado con el censo completo de Go, fue rechazado antes de AUT106.
Instalada la migración una sola vez, el mismo plan quedó admitido: catálogo
versión 7, 68 perfiles y 22 entradas, sin añadir fuentes ni concesiones.

La repetición recuperó el recibo original. Después de reiniciar PostgreSQL y
la aplicación, la repetición conservó sus bytes y la consulta devolvió el mismo
catálogo. La prueba SQL contrastó 92 versiones históricas y el constructor
real de un cargo. El portal y las consultas principales de Contratación
temporal respondieron 200; Chrome mostró el cuadro sin
desbordamiento en escritorio y sin excepciones JavaScript de la página.
La consola registró un error de certificado del service worker
`/portal-empleado/sw.js` en el origen local de la copia. El aviso persistió
con la CA importada en un perfil temporal de Chrome; sesión, módulos, cuadro
y catálogos respondieron 200. No se ha demostrado relación con este cambio
SQL ni se ha modificado la validación TLS del producto.

Los LOGIN técnicos y la aprobación de la copia son ficticios y están
declarados como tales en su material privado. No se asignó el cargo a ninguna
persona ni se firmó un documento. El consumidor web sigue siendo la firma
del expediente de Contratación temporal; este corte comprueba su preparación
administrativa, sin atribuirle el recorrido completo de firma.

La prueba SQL y las 15 pruebas focales del preparador son favorables.
Las revisiones independientes y el resultado de calidad se registran en la
solicitud de integración.
