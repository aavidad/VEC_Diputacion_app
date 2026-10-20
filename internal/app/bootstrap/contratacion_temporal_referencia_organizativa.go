package bootstrap

// referenciaCT se fija al componer y se copia en los soportes del mismo montaje.
// No concede permisos ni transforma referencias históricas.
type referenciaCT struct {
	organizacionRef string
	unidadRef       string
}
