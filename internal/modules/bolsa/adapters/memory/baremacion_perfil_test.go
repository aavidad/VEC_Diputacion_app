package memory

// PerfilRepositorioBaremacionesSoloPruebas solo existe durante `go test`.
// Una composicion ordinaria puede nombrar el tipo opaco, pero no fabricar el
// campo privado que habilita este adaptador efimero.
func PerfilRepositorioBaremacionesSoloPruebas() PerfilUsoRepositorioBaremacionesMemoria {
	return PerfilUsoRepositorioBaremacionesMemoria{soloPruebas: true}
}
