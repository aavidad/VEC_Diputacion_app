# F1-36B: fuente autorizada de la ficha y anexo de selectores

Guion para una copia PostgreSQL 18 de la principal. AUT102 y CT229 se instalaron una
vez en la copia H; sus pruebas SQL focales pasan. B sigue en rama y aún
no acredita el vector de caducidad al confirmar, arranque ni navegador.

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

## Ensayo y comprobaciones pendientes

Las dos fuentes instaladas conservan sus hashes con dos revisiones
independientes de SQL e identidad. La prueba CT229 recibió únicamente los
paréntesis de su expresión CASE para que PostgreSQL la analice; esa corrección
tiene dos revisiones y pasa en la misma copia, sin reaplicar la migración. Comprobar permiso de ficha, cargo propio
y vigente, unidad coincidente; denegar cargo ajeno, retirado, otra unidad o
firma sin unidad histórica. Revisar cero y 128 selectores exactos, vínculo
uno a uno y ambos sellos; incluso la lista vacía exige anexo.

La guarda de caducidad comprueba vigencia real al cierre y en COMMIT, ligada al consumo
V3 y auditoría lógica de la misma transacción. Falta una prueba con decisión
V3 válida al leer y caducada antes de COMMIT: debe abortar sin acceso, anexo
ni recibo nuevos. Caducidad antes de entrar no cubre ese caso. No afirmar
éxito hasta observarlo en PostgreSQL real.

Arrancar con y sin complemento; verificar ficha anterior, lista y PDF según
sus permisos. Reiniciar aplicación y base; cotejar referencias, fechas y
recibos. Conservar ensayo, revisiones, pruebas y CI. Dirección integra; B
no cierra F1-36 ni `docs/producto/FLUJO_APP.md` §4.0.
