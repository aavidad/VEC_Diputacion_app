# Almacén local de originales

El directorio raíz y sus subdirectorios deben pertenecer al uid del proceso y
ser privados. Al iniciar, el adaptador toma el cerrojo exclusivo y elimina los
ficheros regulares `tmp_*` que quedaron tras un corte; solo
registra la cantidad eliminada. Los escritores mantienen el cerrojo compartido
`.volcados` hasta concluir la operación y no se limpian durante otro arranque.
La configuración
`MaximoVolcadosConcurrentes` limita los volcados simultáneos por instancia;
su valor predeterminado es 2 y se valida entre 1 y 16 cuando se fija.

Este adaptador compone rutas con `filepath.Join`. No emplea descriptores de
directorio con `openat` ni `RESOLVE_BENEATH`: un proceso con el mismo uid puede
sustituir un subdirectorio entre la validación inicial y una operación. El
aislamiento del uid y del volumen privado forma parte del modelo de amenaza.
