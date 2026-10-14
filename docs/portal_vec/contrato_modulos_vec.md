# Contrato de módulos de VEC

El contrato de módulos está repartido ahora en los documentos de producto:

- Qué debe traer un módulo nuevo (manifiesto, registro, montaje en servidor y
  web, autorización, persistencia, dependencias, textos, operación y retirada)
  y el aislamiento de fallos:
  [TRD §3 Módulos](../producto/TRD.md#3-módulos).
- Activación por selectores, configuración y la regla de que ninguna
  configuración nueva tumba el arranque:
  [TRD §4](../producto/TRD.md#4-composición-configuración-y-arranque).
- Propiedad de los datos, formas de intercambio entre módulos (función del
  dueño, referencia opaca, outbox e inbox) y retirada sin `DOWN`:
  [ESQUEMA_BACKEND §3](../producto/ESQUEMA_BACKEND.md#3-propiedad-de-los-datos-y-cómo-se-hablan-los-módulos)
  y [§9](../producto/ESQUEMA_BACKEND.md#9-migraciones).
- El detalle del aislamiento entre módulos sigue en
  [aislamiento_modular_y_dependencias_2026-07-25.md](aislamiento_modular_y_dependencias_2026-07-25.md).

Las propuestas antiguas de este documento ya no valen: identidad solo por
Kerberos, Module Federation, componentes con Lit, GrxFirma para iniciar sesión,
un bus de eventos común y una búsqueda global del portal. Lo vigente está en
[TRD §18](../producto/TRD.md#18-contradicciones-detectadas-en-documentos-anteriores).
La versión completa anterior, con la clasificación de módulos de julio, está en
la historia de Git de este fichero.

## Lo que sigue vigente y no está en otro sitio

**Activar un módulo, paso a paso.**

1. Revisar el paquete, sus rutas y versiones sin activar escrituras. Si un
   identificador, contrato o ruta choca con otro módulo, no se activa el
   afectado; nunca se sobrescribe el registro del otro.
2. Comprobar datos, configuración y red. Preparar las migraciones nuevas y la
   forma de restaurar, conservando las ya instaladas y su historia.
3. Componer los puertos que ya existen y ofrecer el menú solo donde toque. Un
   módulo instalado puede tener una capacidad pendiente; se muestra con su
   estado real.
4. Comprobar la función desde la pantalla hasta el resultado, y la
   recuperación si escribe.
5. Si falla, se apaga la capacidad o se vuelve a la versión anterior
   compatible. No se borran tablas ni se restaura una copia encima de
   operaciones posteriores.

**Compatibilidad.** El esquema crece sin romper: los consumidores anteriores
siguen funcionando mientras se actualizan, y una ruptura exige versión y
migración explícitas. Versionar no significa que cualquier combinación de
paquetes funcione.

**Activar no es publicar.** Encender un módulo no lo abre a otra red:
registrar Cronos no expone Dietas ni Personal.

**Personal.** Un módulo que necesite datos del empleado consume referencias de
persona y los hechos de su dueño. Si aporta algo a la ficha del empleado, lo
hace con una vista de lectura autorizada, sin copiar datos. Ejemplo pendiente
de encargo: Formación aportaría su catálogo de cursos y una sección en la ficha
sin tocar Contratación ni Dietas, y sus permisos no abrirían nóminas ni trienios.

**Registro electrónico.** Cuando exista, será una capacidad común que dé número
de asiento y justificante. Ningún módulo inventa su propia numeración de
registro.

**Sin capas por obligación.** No se exige una prueba por cada interfaz ni un
número mínimo de capas o ficheros. Si un cambio obliga a tocar condiciones del
portal que preguntan por el nombre del módulo, se anota y se reduce con el
siguiente módulo que lo necesite.
