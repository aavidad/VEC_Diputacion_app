# Lecturas de Bolsa y Contratación temporal, 12 de octubre de 2026

El banco local midió 30 respuestas por lectura, después de tres calentamientos,
con HTTPS y certificado de cliente verificados. Sesión, módulos, cuadro y lista
CT, ficha CT y resumen de Bolsa cumplen sus umbrales. Preferencias y candidatos
devuelven 503 y todavía no tienen una medida válida. La condición 4 completa
sigue pendiente.

| Lectura | HTTP 200 | p50 ms | p95 ms | Umbral ms |
| --- | ---: | ---: | ---: | ---: |
| Sesión | 30 | 0,178 | 0,227 | 100 |
| Módulos | 30 | 0,242 | 0,318 | 100 |
| Cuadro CT, límite 1 | 30 | 205,598 | 278,804 | 300 |
| Cuadro CT, límite 50 | 30 | 235,001 | 273,135 | 300 |
| Cuadro CT, límite 100 | 30 | 265,748 | 275,294 | 300 |
| Lista CT, límite 1 | 30 | 142,791 | 160,816 | 300 |
| Lista CT, límite 50 | 30 | 173,371 | 214,172 | 300 |
| Lista CT, límite 100 | 30 | 204,072 | 298,168 | 300 |
| Ficha CT original | 30 | 118,484 | 140,505 | 300 |
| Resumen de Bolsa | 30 | 63,204 | 68,444 | 300 |
| Filtro CT sin resultados | 30 | 142,000 | 157,047 | 300 |
| Preferencias | 0 | — | — | 100 |
| Candidatos de bolsa original, límite 50 | 0 | — | — | 300 |

Cada uno de los dos últimos casos recibió 30 respuestas 503 y otras tres en
calentamiento. El cliente terminó con código 1. Los errores no entran en los
percentiles. El resumen de Bolsa y candidatos se repitieron con 30 muestras
y tres calentamientos después de corregir el canon de la fixture; esa es la
medida de Bolsa de la tabla. Las demás lecturas conservan el lote inicial.
La ficha del candidato carga datos de la lista y varias secciones;
su lectura completa no queda acreditada con la lista fallida.

## Base, servidor y volumen

La copia HZ21 y su kit fueron asignados por dirección para este ensayo. Se
restauraron en PostgreSQL 18.4 propio, sin red exterior, con `--rm`,
`--restart=no`, 2 GB de memoria y datos en disco. Aplicación, PostgreSQL y
cliente compartieron el contenedor, limitado a dos CPU; el servidor usó
`GOMAXPROCS=2`, 512 MB de `shared_buffers` y 100 conexiones máximas.

Fuente del servidor: `1a622d1defae707382c1f2fd815ddf68b366ddba`.
SHA256 del binario:
`4a8ecb0a70c18d3c27dd7df33d4e902059660c9d23f466367bbc025725b216d0`.
Se compiló con `-buildvcs=false` y la marca existente `telemetria.Revision`
fijada al commit del checkout. El cliente es el de #1027, ya integrado.

La carga reutiliza la semilla CT187 de #840, el protocolo de #916 y las
lecturas globales de Bolsa de #921. Conserva 71 expedientes integrales y
170 publicaciones originales. Añade 5.000 referencias CT de proyección,
5.500 publicaciones en seis fases, 833 incidencias y 500 marcas de urgencia.
Bolsa conserva sus 12 bolsas y 390 participaciones originales; una categoría
RPT sin bolsa activa añade 2.000 participaciones de proyección. El resumen
lee 2.390 participaciones.

Las filas nuevas miden listas y agregados. No representan 5.000 expedientes
integrales ni 2.000 fichas nominales nuevas: la bolsa añadida carece de lote
CONVOCA protegido. Las fichas se consultan sobre originales válidos. La
antigüedad de una fase no prueba por sí sola que su plazo esté vencido. Una
consulta HTTPS posterior al lote, con el mismo servidor y sin cambiar filas,
confirmó HTTP 200 y el resumen calculado: 2.979 vencidos, 834 incidencias,
84 vencimientos durante la semana y 42 plazos sin calcular. Las cien filas de
esa página tenían `urgente: true`; incluían plazos vencidos y en plazo. No se
publican sus cuerpos ni referencias personales.

La primera carga CT dio 502 porque 3.780 publicaciones situaban la entrada
en fase antes del alta. Se corrigieron únicamente esas filas sintéticas,
conservando los validadores. El recuento de cronologías incoherentes quedó a
cero y el POST del cuadro volvió a 200 antes del lote.

El selector de Mis correos se desactivó únicamente en el lanzador privado
del banco: su dependencia impedía arrancar esta copia. Las lecturas de CT,
Bolsa y preferencias conservan sus autorizaciones. No se instaló SQL ni se
modificaron perfiles, roles o funciones.

## Repetir el ensayo

1. Copiar la base y el kit aprobados por dirección en un contenedor propio
   con los límites anteriores y la etiqueta `vec.propietario`. Conservar la
   configuración y el material privado fuera de Git. No arrancar ni modificar
   la copia de origen.
2. Con Go instalado, aplicar una sola vez `scripts/carga/rrhh/cargar.sh CONTENEDOR PROPIETARIO`.
   El lanzador genera los dos cánones con los constructores de dominio de Bolsa
   y los valida antes de cargar. Los guiones son DML de carga, no migraciones. Usan
   `session_replication_role=replica` en la sesión del banco; no deben aplicarse
   a una base de trabajo compartida. Conservar su salida de recuentos.
3. Compilar el servidor del commit indicado, con su marca de revisión,
   y arrancarlo mediante el lanzador del kit en la copia. Antes de medir,
   comprobar que escucha y que los casos responden con el contrato esperado.
4. Preparar los casos privados según
   [README_RRHH.md](../../scripts/carga/README_RRHH.md): cuadro con
   `resumen: true`, lista sin resumen y referencias originales para fichas.
   Ejecutar el cliente con `-usuarios 1 -muestras 30 -calentamientos 3 -pausa 0s`.
5. Guardar tabla, estado de salida, recuentos por fase y situación y registro
   técnico. Los cuerpos personales, claves y conexiones quedan fuera de Git.
   Retirar el contenedor y los temporales propios al acabar.

## Pendiente de diagnóstico

Preferencias falla en la fase de sesión: la única consulta observada es
`vec_identidad_sesiones_v1.registrar_sesion_v1`; no llega a revalidación ni
contexto. La única cuenta seleccionada por la configuración privada de preferencias no existe en
`vec_identidad_sesiones_v1.cuenta`: recuento esperado 1, observado 0, y cero
alias. Su alta nominal queda a cargo del dueño de identidad mediante el
mecanismo existente.

Antes del volumen, candidatos falló tras siete consultas con un error sin
clasificar. En el lote publicado falló antes, tras una consulta, con
`ErrConstitucionBolsaNoDisponible`: la instantánea añadida por la fixture no
seguía el canon que valida el lector de todas las bolsas. Se corrigió
únicamente la fixture propia con los constructores de dominio.
El lector PostgreSQL real vuelve a aceptar las trece constituciones y la
instantánea de 2.000 entradas. Una segunda copia limpia recibió los siete
guiones por `cargar.sh` sin errores y conservó los recuentos originales.
Después de la corrección, las 30 muestras y tres calentamientos de candidatos
vuelven al fallo previo: siete consultas, 503 y causa sin clasificar. Su
recuperación del lote protegido sigue pendiente del dueño de Bolsa; no se
atribuye el error al rendimiento ni se rebajan validadores.

No se atribuye a esta tabla una mejora antes/después ni una validación de
cidonia, Chrome o la condición 3 completa. Las lecturas válidas medidas no
excedieron sus presupuestos; la lista de 100 expedientes quedó cerca del límite.

CONFIG NUEVA: ninguna para la aplicación.
