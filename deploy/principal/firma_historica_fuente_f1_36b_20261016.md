# F1-36B: fuente autorizada de la ficha y anexo de selectores

Guion para una copia PostgreSQL 18 de la principal. AUT102 y CT229 se instalaron una
vez en la copia H; sus pruebas SQL focales pasan. B sigue en rama. La activación de consulta y el vector de caducidad al
confirmar están comprobados; quedan Chrome y reinicio conjunto.

## Orden y activación

Partir de A instalado en la copia: AD286, CT228 y AUT103. Comprobar historia,
ACL y preimágenes. Aplicar solo AUT102 → CT229 ausentes, con sus roles
propietarios. No usar DOWN ni reaplicar historia; detenerse ante divergencia.
Conexiones y materiales de identidad y aprobación siguen fuera de Git.

AUT102 comprueba por fachada nominal que el cargo de quien consulta sigue
vigente en la unidad histórica; no concede acceso a la ficha. CT229 añade
fachadas de detalle y un anexo privado e inmutable al recibo CT43. SQL forma
el conjunto exacto y liga acceso y dos sellos. La aplicación pide la variante
ampliada solo con concesión del campo. La ficha anterior conserva su respuesta.
B no publica selectores por HTTP ni monta panel; C hará eso y los actos.

## Ensayo y límites

Las dos fuentes instaladas conservan sus hashes con dos revisiones
independientes de SQL e identidad. La prueba CT229 recibió únicamente los
paréntesis de su expresión CASE para que PostgreSQL la analice; esa corrección
tiene dos revisiones y pasa en la misma copia, sin reaplicar la migración. Comprobar permiso de ficha, cargo propio
y vigente, unidad coincidente; denegar cargo ajeno, retirado, otra unidad o
firma sin unidad histórica. Revisar cero y 128 selectores exactos, vínculo
uno a uno y ambos sellos; incluso la lista vacía exige anexo.

La guarda de caducidad comprueba vigencia real al cierre y en COMMIT, ligada al consumo
V3 y auditoría lógica de la misma transacción. La prueba con una decisión V3 genuina válida al leer y caducada antes de
COMMIT pasó en PostgreSQL 18. No se modificaron el reloj, los plazos del
producto, la decisión ni las restricciones diferidas.

Arrancar con y sin complemento; verificar ficha anterior, lista y PDF según
sus permisos. Reiniciar aplicación y base; cotejar referencias, fechas y
recibos. Conservar ensayo, revisiones, pruebas y CI. Dirección integra; B
no cierra F1-36 ni `docs/producto/FLUJO_APP.md` §4.0.

El 16/10 se ensayaron dos vías con decisiones distintas emitidas por el
productor real y el certificado ficticio de Jefatura: detalle con resultados
de Bolsa (17 argumentos) y original histórico de propuesta v7 desde una ficha
v9 (15 argumentos). El arnés privado, con dos revisiones favorables, captura
la orden antes de abrir la transacción; esa petición HTTP falla cerrada y no
acredita una lectura. En la vía histórica, la consulta preparatoria sí deja
su acceso real; se distingue de la decisión del original que se somete al cierre.

La conexión posterior usa el LOGIN nominal de consulta, la misma orden y una
transacción serializable. En ambas vías el SELECT devuelve acceso y sellos
antes de vencer; después de esperar con el reloj real, COMMIT devuelve 42501
con el motivo específico de caducidad de CT229. Un observador de la misma
copia comprueba cero filas para ese acceso en CT43, anexo y cierre, y cero
consumos y apuntes de consumo para esa decisión. La atestación previa no se
confunde con el consumo. La captura se borra y el observador temporal se retira.
El arnés y su binario no forman parte del producto ni se instalan.

Los cuatro cargos de consulta V2 se publicaron con el CLI existente y su recibo.
Las cuatro asignaciones sucesoras se cargaron como datos ficticios autorizados
por dirección, con historia, guardas y ámbitos conservados; no se atribuyen
actos de Administración. La configuración privada ya seleccionaba los mismos
perfiles. La ficha autorizada devuelve tres selectores y recibo; la anterior
sin concesión omite ambos campos. No se repiten esas publicaciones ni cargas.
