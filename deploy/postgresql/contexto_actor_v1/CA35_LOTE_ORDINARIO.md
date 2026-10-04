# CA35: contexto de perfiles para el lote ordinario

CA35 añade dos funciones privadas a Contexto Actor. Reciben el instante único
que elegirá la fachada SQL AUT44. Una alta inmediata usa ese instante como
inicio; una programada conserva la fecha futura del objetivo canónico.
Comprueban versiones, procedencia maestra y titularidad CA33, también cuando
es el primer perfil de la persona. Una revocación crea versiones nuevas y no
reactiva las anteriores.
Solo el propietario de Autorización puede ejecutarlas. CA20 permanece intacta.

Este corte está **en preparación**. AUT44 debe confirmar todos los cambios en
una transacción serializable con una única decisión V3, auditoría y recibo;
AD190 debe publicar su consumidor nominal después del gate Rol6 de AUT45.
No se ha ensayado ni instalado CA35 y no se debe aplicar junto a un adaptador
incompleto. El ensayo de PostgreSQL corresponde al escritor único del clon,
tras dos revisiones independientes de los hashes finales.
