# Fechas prospectivas del mantenimiento — AUT46

Las revisiones nuevas deben cumplir EmitidaEn ≤ Desde < Hasta. AUT42 cambiaba
la emisión al publicar Rol5 y dejaba Desde anterior; ese destino no pasa el
validador de asignaciones Go. AUT46 corrige sólo efectos nuevos, sin UPDATE de
historia ni cambio del dominio global.

La utilidad privada no lee tablas ni concede permisos. Elige Desde como el
mayor instante entre Desde original y la nueva emisión aprobada. Conserva el
string del instante elegido para no alterar el canon. Mantiene Hasta original
y rechaza una ventana vacía o fechas no finitas. El constructor histórico
AUT42 queda intacto.

En replay, el efecto coteja el documento persistido con el candidato histórico.
Si es distinto, coteja el candidato prospectivo corregido. No sustituye el
documento, su SHA ni el recibo. Esto conserva replay de las revisiones antiguas
y de las nuevas. Un destino revocado, caducado o fuera del CAS no se rescata.

Efecto, append permitido y revalidación tras la última espera comparten el
subbloque transaccional. Se comprueban configuración, plan y ventanas
originales; si caducan esperando, se revierte el efecto/append permitido y se
registra sólo la negativa gestionada. Los sellos y la auditoría siguen siendo
los existentes AUT24/AD183. No hay familia nueva ni tabla de auditoría aparte.

AUT45 usa la misma utilidad para Rol6. Orden causal: estructura
AUT42 → AUT46 → efecto42 aprobado → estructura AUT45 → efecto45 aprobado.
Instalar las estructuras no publica Rol5/6 ni modifica asignaciones.

Las tres guardas de AUT46 proceden de capturas reales de Source antes del
primer efecto42 del fixture vivo: helper histórico, efecto y wrapper. Se
conservan sus OID, propietario, ACL y configuración mediante reemplazo sólo
de las dos definiciones de ejecución; el helper histórico no se reemplaza.

Validación Go focal de CLI: normal/race/vet verdes, Semgrep local y gosec sin
hallazgos, diff/gofmt verdes. Los JSON de testdata son resultados independientes
que los vectores SQL46/45 comparan con los constructores reales. Go comprueba
que los destinos pasan AsignacionPerfil.Validar sin prolongar Hasta ni cambiar
identidad/ámbitos; el driver devuelve bytes y SHA originales de replay.
La revisión histórica con Desde anterior a EmitidaEn sigue como historia,
no se regenera para hacerla pasar por un destino nuevo.

Dos revisiones independientes cerraron el contenido exacto. En el clon nuevo
POSTH9, AUT46 y su vector terminaron con código 0. La CLI real completó el
mantenimiento42 y su replay con el mismo recibo. Los tres documentos actuales
persistidos pasaron `AsignacionPerfil.Validar`, sin normalizar sus fechas.

Después, AUT45 y sus vectores terminaron con código 0. La CLI real publicó
Rol6 y recuperó el mismo recibo, fecha y huella en replay y después de reiniciar
PostgreSQL. Las dos APP quedaron en revisión3; Sistemas conservó su revisión1.
Identidades, CA, ámbitos y Hasta permanecieron intactos. El fixture se produjo
por el pipeline autorizado con planes nuevos, sin extender accesos vencidos.

La aprobación divergente fue denegada por un LOGIN real, con recibo nulo e
intento durable. El CAS superado devolvió `40001` en una lectura revertida.
La auditoría común terminó en 6254. No se ejecutó la espera temporal con varias
sesiones: ese caso conserva revisión estática y vector puro, sin atribuirle un
ensayo dinámico. Tampoco se acredita acceso nominal HTTP, V2 ni navegador.

Acta privada `acta-entrega46-42-45.json`, SHA256
`cc784eaeb887e143b9a1df7697a73f034a49520a098f2d80a02cee543f581422`.
Copia fría de Rol6 antes del último replay, SHA256
`8cf463a9c2b2147a58b62fbacfda89f82578177e495e135d7fbd1f808c7961aa`.
La puerta completa se ejecutó una vez; los deltas posteriores de fechas y
terminadores SQL tienen pruebas focales y dos ratificaciones exactas.
