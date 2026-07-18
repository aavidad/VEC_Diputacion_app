package gobiernoconvocatorias

const (
	esquemaProcedenciaActoBorradorV1 = "vec.acto.procedencia.v1"
	perfilActoDesarrollo             = "desarrollo"
	AutoridadActoAutoritativa        = "autoritativo"
	AutoridadActoNoAutoritativa      = "no_autoritativo"
)

// ProcedenciaActoBorrador es una marca obligatoria del dato, no una opción de
// presentación. Viaja en AAD, agregado, recibo, auditoría y outbox. Su valor
// cero es inválido para impedir que la ausencia se interprete como producción.
type ProcedenciaActoBorrador struct {
	bloqueoSerializacionDiario
	Esquema            string
	PerfilEjecucion    string
	Autoridad          string
	ProveedorRef       string
	MigrableProduccion bool
}

func NuevaProcedenciaActoBorrador(
	perfilEjecucion, autoridad, proveedorRef string,
	migrableProduccion bool,
) (ProcedenciaActoBorrador, error) {
	p := ProcedenciaActoBorrador{
		Esquema: esquemaProcedenciaActoBorradorV1, PerfilEjecucion: perfilEjecucion,
		Autoridad: autoridad, ProveedorRef: proveedorRef,
		MigrableProduccion: migrableProduccion,
	}
	if !p.valida() {
		return ProcedenciaActoBorrador{}, ErrResultadoBorradorInseguro
	}
	return p, nil
}

func (p ProcedenciaActoBorrador) valida() bool {
	if p.Esquema != esquemaProcedenciaActoBorradorV1 ||
		!referenciaProyeccionValida(p.PerfilEjecucion) ||
		!referenciaProyeccionValida(p.ProveedorRef) {
		return false
	}
	switch p.Autoridad {
	case AutoridadActoAutoritativa:
		// El perfil local nunca puede convertirse en autoridad cambiando dos
		// campos exportados antes de construir el acto.
		return p.MigrableProduccion && p.PerfilEjecucion != perfilActoDesarrollo
	case AutoridadActoNoAutoritativa:
		return !p.MigrableProduccion
	default:
		return false
	}
}

func procedenciasActoCoinciden(a, b ProcedenciaActoBorrador) bool {
	return a.valida() && b.valida() && a.Esquema == b.Esquema &&
		a.PerfilEjecucion == b.PerfilEjecucion && a.Autoridad == b.Autoridad &&
		a.ProveedorRef == b.ProveedorRef && a.MigrableProduccion == b.MigrableProduccion
}

// IdentidadAutoridadBorrador describe una frontera de confianza sin incluir
// secretos. La composición la usa para impedir que quien produce una prueba
// sea también quien la verifica bajo la misma credencial o rol.
type IdentidadAutoridadBorrador struct {
	ProveedorRef  string
	InstanciaRef  string
	CredencialRef string
	RolRef        string
}

func NuevaIdentidadAutoridadBorrador(
	proveedorRef, instanciaRef, credencialRef, rolRef string,
) (IdentidadAutoridadBorrador, error) {
	i := IdentidadAutoridadBorrador{
		ProveedorRef: proveedorRef, InstanciaRef: instanciaRef,
		CredencialRef: credencialRef, RolRef: rolRef,
	}
	if !i.valida() {
		return IdentidadAutoridadBorrador{}, ErrServicioBorradoresInvalido
	}
	return i, nil
}

func (i IdentidadAutoridadBorrador) valida() bool {
	return referenciaProyeccionValida(i.ProveedorRef) &&
		referenciaProyeccionValida(i.InstanciaRef) &&
		referenciaProyeccionValida(i.CredencialRef) &&
		referenciaProyeccionValida(i.RolRef)
}

func autoridadesOperativasBorradorSeparadas(a, b IdentidadAutoridadBorrador) bool {
	return a.valida() && b.valida() && a != b &&
		a.CredencialRef != b.CredencialRef && a.RolRef != b.RolRef &&
		(a.ProveedorRef != b.ProveedorRef || a.InstanciaRef != b.InstanciaRef)
}

func autoridadesPoliticaBorradorSeparadas(a, b IdentidadAutoridadBorrador) bool {
	return autoridadesOperativasBorradorSeparadas(a, b) &&
		a.ProveedorRef != b.ProveedorRef
}

type DescriptorAutoridadBorrador interface {
	IdentidadAutoridadBorrador() IdentidadAutoridadBorrador
}
