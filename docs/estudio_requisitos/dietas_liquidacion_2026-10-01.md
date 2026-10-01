# Preparación de la liquidación de Dietas — 1 de octubre de 2026

La liquidación debe conservar qué gastos se admiten, cuáles se rechazan y por
qué, con la versión exacta de las tarifas aplicadas. El documento enviado por
la persona contiene importes orientativos y topes; no acredita ese acto.

Este corte permite preparar una propuesta local y revisar su informe con
datos sintéticos. La aprobación efectiva sigue esperando la fuente de
competencias de Personal, los perfiles nominales de AUT27 y el registro
transaccional de Dietas. No da por cerrado D9 de la
[ficha de requisitos](ficha_dietas_2026-09-23.md).

## Decisión de implementación

Codex-G y la revisión independiente de arquitectura acordaron reutilizar el
documento de comisión y separar dos resultados:

- La preparación contiene la comisión y su versión, el catálogo importado,
  los importes originales y propuestos y los motivos de reducción o rechazo.
- La liquidación registrada deberá incorporar además el acto, el recibo,
  la fecha y las referencias de autorización, competencia y auditoría que
  produzca el repositorio. Esas referencias no se generan en el ensayo.

Los importes se expresan en céntimos. Se comprueban las sumas y los límites
antes de preparar la propuesta. La copia del catálogo y del documento permite
reproducir el resultado aunque cambien sus entradas. Sus huellas comprueban
integridad; no acreditan aprobación ni procedencia administrativa.

La futura escritura guardará juntos la instantánea económica, la transición,
el consumo de autorización, la revalidación de competencia, la auditoría y el
evento de salida. No se llamará primero a la decisión genérica y después a
otra escritura económica. El informe definitivo consumirá la instantánea
conservada, sin recalcularla con las tarifas del momento de descarga.

## Paquete de ejemplo

El primer ejemplo se limita a una comisión nacional ordinaria, sin
alojamiento. Las cuantías y las reglas se aportan como datos de una versión
retirable. La versión del ejemplo en Git permite reproducir el ensayo; la
publicación operativa de un catálogo requiere su propio acto y auditoría.

El ejemplo ilustra una selección previa para grupo 2, de 09:00 a 18:00:
manutención de 18,70 euros y 100 kilómetros sintéticos en automóvil a
0,26 euros por kilómetro. El preparador recibe los importes del documento;
no vuelve a calcular los horarios. Los 100 kilómetros no proceden de OSRM.
El total propuesto es 44,70 euros.
Las excepciones, los anticipos, los viajes fuera de España y el alojamiento
quedan fuera de este paquete.

## Comprobar la propuesta y su informe

Desde la raíz del repositorio, el comando produce la preparación como JSON:

```sh
go run ./cmd/vec-dietas --preparar-liquidacion \
  < cmd/vec-dietas/testdata/preparacion_liquidacion.json
```

Para generar el informe en castellano con el tema común:

```sh
go run ./cmd/vec-dietas --preparar-liquidacion --informe \
  --textos web/static/textos/es/dietas-liquidacion-informe.json \
  --tema web/static/comun/tema-vec.css \
  < cmd/vec-dietas/testdata/preparacion_liquidacion.json \
  > propuesta-dietas.html
```

El catálogo equivalente en `textos/en/` genera el informe en inglés. El HTML
incluye la misma preparación, sus diferencias motivadas y el detalle del
catálogo. Se puede abrir e imprimir sin conexión. La impresión conserva las
fuentes y huellas aunque su apartado esté plegado. Este informe de propuesta
no es el PDF del documento liquidado exigido por D9.

El ensayo de devengo anterior sigue disponible sin argumentos. Los catálogos
de idioma y el tema se eligen mediante rutas explícitas del operador; los
errores no muestran esas rutas ni el contenido de los archivos.

## Contraste con fuentes públicas

| Fuente | Consecuencia para VEC |
| --- | --- |
| [Bases de ejecución de Granada 2026, artículo 30](https://www.dipgra.es/export/sites/diputaciongranada/diputacion/delegaciones/economia-y-atencion-al-alcalde/.galleries/DIPUTACION-Delegaciones-Economia-y-Patrimonio-Economia/DIPUTACION-Delegaciones-Economia-y-Patrimonio-Economia-Presupuestos/DIPUTACION-Delegaciones-Economia-y-Patrimonio-Economia-Presupuesto-Diputacion/Presupuesto-Diputacion-Ejercicio-2026/Ejercicio-2026-Presupuesto-General/BASES-DE-EJECUCION-DEL-PRESUPUESTO-2026-TRAS-ENMIENDAS-Y-RECTIFICACION-Y-ANEXOS-modificaciones-actualizadas-26-08-26.pdf) | Conservar los antecedentes de autorización y justificación que correspondan; un perfil informático no los sustituye. |
| [RD 462/2002, artículos 10 y 12](https://www.boe.es/buscar/act.php?id=BOE-A-2002-10337#a10) | Distinguir manutención de alojamiento y comprobar las condiciones temporales; un tope de alojamiento no es automáticamente un gasto admitido. |
| [Resolución de 2 de diciembre de 2005](https://www.boe.es/buscar/doc.php?id=BOE-A-2005-19988) y [Orden HFP/793/2023](https://www.boe.es/buscar/doc.php?id=BOE-A-2023-16462) | Identificar la fuente de las cuantías nacionales y del kilometraje del ejemplo. |
| [Normas de dietas UGR 2026, artículos 34 y 35](https://gerencia.ugr.es/sites/webugr/gerencia/public/ficheros/Presupuestos%20UGR/2026/Normars%20liquidaci%C3%B3n%20y%20tramitaci%C3%B3n%20dietas%202026.pdf) | Separar autorización, justificación y tramitación; no trasladar sus plazos a Diputación. |
| [Decreto 54/1989 de Andalucía, artículo 39](https://www.juntadeandalucia.es/boja/1989/31/11) | Referencia comparativa para orden, itinerario realizado y certificación del servicio; no atribuye competencia en VEC. |

## Cuestiones para RRHH

Estas cuestiones completan las preguntas 23–26 de `dudas.md` y deben
incorporarse allí en la edición coordinada del fichero:

1. Las bases de Granada, apartado 30.4.3, remiten al artículo 12.4 del RD
   462/2002 al tratar la justificación de alojamiento. Ese precepto regula
   la cena de regreso y el artículo 10.3 se refiere al alojamiento
   efectivamente gastado y justificado. ¿Qué justificante se exige en
   Diputación y qué regla debe aplicar VEC?
2. ¿Qué acto aprueba la liquidación, quién lo firma y cómo se relaciona con
   la autorización previa del Diputado prevista en las bases? ¿Cómo se
   rectifica después sin borrar la liquidación anterior?
3. ¿En qué momento y con qué criterio se redondean porcentajes y
   kilometraje? ¿Qué regla se aplica a las excepciones y a comisiones que
   atraviesan un cambio de tarifa?
4. Las bases prevén diez días para la cuenta justificativa. ¿Desde qué hecho
   se cuentan, qué calendario se aplica y qué ocurre si se presentan fuera
   de plazo?

Hasta aclararlo, el ensayo conserva su carácter de propuesta. El circuito
real continúa cerrado cuando falta una competencia acreditada.
