# Estado y plan de ataque del proyecto

**Última actualización:** 18 de julio de 2026

**Frente principal:** primera funcionalidad real de Bolsa

**Frente paralelo:** importador gobernado de exportaciones Convoca, usando
exclusivamente ficheros sintéticos hasta que sea seguro tratar datos reales.

**Objetivo actual:** crear y editar desde la web un borrador de convocatoria,
guardarlo cifrado en PostgreSQL, recuperarlo después de reiniciar y obtener un
recibo auditable.

Este es el tablero de seguimiento para dirección. Se actualiza antes del commit
que cierre una capacidad o siempre que cambie el frente principal. Los códigos
internos como `T20` se conservan en la documentación técnica, pero no dirigen
este tablero.

Los agentes adicionales deben seguir
[ORQUESTACION_AGENTES.md](ORQUESTACION_AGENTES.md); allí se indican tareas
ocupadas, trabajos libres, dependencias, límites de archivos y formato de
entrega.

## Dónde estamos ahora

**Primera funcionalidad real: paso 3 de 6.**

| Paso del objetivo actual | Estado | Evidencia o condición de cierre |
| --- | --- | --- |
| 1. Entorno seguro de desarrollo | ✅ Terminado | mTLS 1.3, identidad local de alta garantía, cifrado KMS y sello de tiempo de desarrollo, sin claves en Git. |
| 2. Diario durable y cifrado de borradores | ✅ Terminado | PostgreSQL, reintentos, recuperación, alias de distintas generaciones y borrado de memoria probados. |
| 3. Guardado transaccional e identificadores seguros | 🚧 En revisión | PostgreSQL/KMS ha superado PostgreSQL 18 y espera revisión independiente. El identificador HMAC necesita dos pruebas públicas adicionales antes del GO. |
| 4. Autorización y lectura gobernada | ⬜ Pendiente inmediato | Migrar la lectura a autorización V2 real y devolver el sobre cifrado completo sin acceso directo a tablas. |
| 5. Conexión definitiva con la web | ⬜ Pendiente | Registrar servicios y rutas reales bajo identidad mTLS, sin cookies, fixtures ni adaptadores falsos. |
| 6. Prueba completa y entrega manual | ⬜ Pendiente | Web → API → autorización → PostgreSQL/KMS → auditoría → respuesta; reinicio, concurrencia y fallos; después guía para que Alberto/RRHH lo prueben. |

## Carriles paralelos activos

| Carril | Trabajo actual | Puede avanzar sin bloquear al resto | Siguiente entrega |
| --- | --- | --- | --- |
| 1. Camino crítico | Guardado real de borradores | Sí; concentra los contratos compartidos de identidad, autorización y PostgreSQL | Transacción e identificadores con revisión independiente cerrada. |
| 2. Datos heredados | Importador de Convoca | Sí, con hojas sintéticas y sin cargar datos personales reales | Reconocer los dos formatos, validar en staging y emitir acta idempotente. |
| 3. Calidad y seguridad | Revisión cruzada de PostgreSQL/KMS y HMAC | Sí; no modifica el código revisado | GO o defectos reproducibles antes de cada commit. |
| 4. Dirección e integración | Tablero, revisión final, pruebas globales y commits acotados | Sí | Mantener una única verdad y encadenar el siguiente trabajo desbloqueado. |

**Siguiente carril que se abrirá al quedar uno libre:** durabilidad probatoria y
registro de accesos, necesarios antes de introducir datos personales reales.

## Significado de las columnas

- **Contrato probado:** existe código real probado de forma aislada.
- **Integrado:** el servidor soportado registra la capacidad con sus
  dependencias reales; una pantalla o un test aislado no cuentan.
- **E2E técnico:** se ha probado el recorrido técnico de extremo a extremo.
- **Probable ahora:** Alberto o RRHH pueden recorrerlo manualmente sin dobles de
  prueba. `DEMO` significa que funciona, pero sin validez administrativa.
- **Aceptado RRHH:** existe una prueba de aceptación formal registrada.
- **Producción:** está desplegado con infraestructura y conectores autorizados.

`E2E` significa *end to end* o «de extremo a extremo». `UAT` significa pruebas
de aceptación de usuario; aquí se escribe siempre **Aceptado RRHH** para evitar
la sigla.

## Tabla principal de capacidades

| Capacidad | Contrato probado | Integrado | E2E técnico | Probable ahora | Aceptado RRHH | Producción |
| --- | --- | --- | --- | --- | --- | --- |
| Consulta pública de convocatorias | ✅ | 🧪 DEMO | 🧪 DEMO | ✅ DEMO | ❌ | ❌ |
| Panel interno agregado de Bolsa | ✅ | ❌ | ❌ | 🧪 Presentación | ❌ | ❌ |
| Creación y edición de convocatorias | ✅ | 🚧 En curso | ❌ | 🧪 Presentación | ❌ | ❌ |
| Publicación, sustitución y retirada | 🟡 Parcial | ❌ | ❌ | 🧪 Presentación | ❌ | ❌ |
| Bases y reglas de baremo | ✅ | ❌ | 🟡 Núcleo/BD | 🧪 Presentación | ❌ | ❌ |
| Autobaremación del aspirante | ✅ | 🧪 Legado | 🧪 Legado | ✅ DEMO | ❌ | ❌ |
| Revisión técnica y rectificación firmada | ✅ | ❌ | 🟡 Aplicación/BD | 🧪 Presentación | ❌ | ❌ |
| Listas, ranking y desempates | 🟡 Parcial | ❌ | ❌ | 🧪 Presentación | ❌ | ❌ |
| Llamamientos | ✅ | ❌ | ❌ | 🧪 Presentación | ❌ | ❌ |
| Contratos, ceses y reincorporaciones | 🟡 Parcial | ❌ | ❌ | 🧪 Presentación | ❌ | ❌ |
| Candidatura, solicitud y registro | 🟡 Parcial | 🧪 Legado | 🧪 Legado | 🧪 Parcial | ❌ | ❌ |
| Subsanaciones y alegaciones | 🟡 Parcial | 🧪 Legado | ❌ | 🧪 Parcial | ❌ | ❌ |
| Documentos, carga, cuarentena y antivirus | ✅ | ❌ | 🟡 Piezas aisladas | ❌ | ❌ | ❌ |
| Firma, sello de tiempo, CSV/QR y cotejo | ✅ | ❌ | 🟡 Piezas aisladas | ❌ | ❌ | ❌ |
| Generación y descarga de PDF/DOCX/ODT/CSV/JSON/etc. | ✅ | ❌ | 🟡 Renderizadores | ❌ | ❌ | ❌ |
| Comunicaciones, correo, Telegram y notificación | 🟡 Parcial | ❌ | ❌ | 🧪 Presentación | ❌ | ❌ |
| Tasas, pagos, devoluciones y conciliación | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| Ayuda, audio y transcripción | ✅ | ✅ Estática | ✅ Estática | ✅ | ❌ formal | ❌ administrable |
| Bot de ayuda pública | 🟡 Diseño | ❌ | ❌ | ❌ | ❌ | ❌ |
| Identidad y separación público/interno | ✅ | 🟡 Desarrollo | ✅ Desarrollo | 🟡 Desarrollo | ❌ | ❌ |
| Roles, permisos y autorización por operación | ✅ | ❌ en Bolsa | 🟡 Piezas aisladas | ❌ | ❌ | ❌ |
| Auditoría, recibos y registro de accesos | ✅ | ❌ completo | 🟡 Piezas aisladas | 🧪 Cronología | ❌ | ❌ |
| Catálogos, configuración y plazos administrables | 🟡 Parcial | 🧪 Consulta | 🧪 Consulta | 🧪 Parcial | ❌ | ❌ |
| Importación de datos de Convoca | 🚧 En desarrollo | ❌ | ❌ | ❌ | ❌ | ❌ |
| API pública | ✅ | 🧪 DEMO | 🧪 DEMO | ✅ DEMO | ❌ | ❌ |
| API interna completa | 🟡 Parcial | ❌ | ❌ | ❌ | ❌ | ❌ |
| CLI, MCP y acceso gobernado para IA | 🟡 Contratos | ❌ | ❌ | ❌ | ❌ | ❌ |
| Protección de datos, conservación y expurgo | ✅ Diseño/núcleo | ❌ integral | 🟡 Pruebas parciales | ❌ | ❌ | ❌ |
| Copias, recuperación, observabilidad y operación | 🟡 Parcial | ❌ integral | 🟡 Pruebas parciales | ❌ | ❌ | ❌ |
| Accesibilidad, tema y preferencias visuales | ✅ | 🟡 Vistas actuales | 🟡 Web | ✅ Parcial | ❌ formal | ❌ |

La tabla distingue trabajo reutilizable de funcionalidad utilizable. Por eso
puede haber `✅` en «Contrato probado» y `❌` en «Integrado» sin que exista una
contradicción.

## Plan de ataque funcional

| Orden | Entregable comprensible | Estado | Cuándo se considera terminado |
| --- | --- | --- | --- |
| 1 | Crear y editar convocatorias reales | 🚧 **Ahora** | RRHH puede crear, listar, abrir y modificar un borrador desde la web; persiste tras reinicio y deja autorización, cifrado, auditoría y recibo. |
| 2 | Poder usar datos reales en un piloto seguro | ⬜ Después | Toda prueba y acceso queda durable; se registra quién accede, para qué y cuándo. |
| 3 | Importar la información existente de Convoca | 🚧 **En paralelo**, sin datos reales | Importación repetible, validada, con incidencias, procedencia y sin duplicados. |
| 4 | Gestionar bases, reglas, puntuaciones y publicación | ⬜ Pendiente | RRHH configura las bases sin programar; se calculan puntuaciones y se publican actos aprobados y firmados. |
| 5 | Completar el expediente del aspirante | ⬜ Pendiente | Perfil, documentos, solicitud, autobaremación, registro, subsanación y alegaciones funcionan juntos. |
| 6 | Completar revisión técnica, listas y llamamientos | ⬜ Pendiente | RRHH revisa, firma, rectifica, genera listas y realiza llamamientos trazables. |
| 7 | Completar contratos, comunicaciones y pagos | ⬜ Pendiente | Ciclo posterior, notificaciones, respuestas, tasas, devoluciones y conciliación quedan conectados. |
| 8 | Validación formal y producción | ⬜ Pendiente | Alberto/RRHH aceptan una versión; Sistemas y seguridad autorizan conectores, despliegue, copias y operación. |

## Regla de actualización

Cada cierre debe actualizar, en este orden:

1. «Dónde estamos ahora».
2. La fila afectada de la tabla principal.
3. El plan de ataque si cambia el frente.
4. La fecha y el historial inferior.
5. Solo después, el commit de la funcionalidad.

## Historial de cambios de este tablero

| Fecha | Cambio |
| --- | --- |
| 18/07/2026 | Creación del tablero único; separación entre código probado, integración, E2E técnico, prueba manual, aceptación RRHH y producción. |

Para el detalle técnico y la brecha de cada fila se mantiene la
[matriz ampliada](docs/portal_vec/matriz_estado_operativo_bolsa_2026-07-18.md).
