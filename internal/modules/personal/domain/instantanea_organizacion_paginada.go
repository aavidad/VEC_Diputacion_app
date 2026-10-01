package domain

// PaginaInstantaneaOrganizacion separa el selector consultado de las versiones
// resueltas por la fuente. No contiene autorización ni evidencia de acceso.
type PaginaInstantaneaOrganizacion struct {
	Instantanea         InstantaneaComparacionOrganizacion `json:"instantanea"`
	VersionRPTRef       string                             `json:"version_rpt_ref"`
	VersionPlantillaRef string                             `json:"version_plantilla_ref"`
	CursorSiguiente     string                             `json:"cursor_siguiente"`
}

// Una cadena con páginas no vacías admite a lo sumo un hecho por página más
// una página final vacía. El límite de hechos sigue siendo el del comparador.
const LimitePaginasComparacionOrganizacion = LimiteHechosComparacionOrganizacion + 1

type ReunionPaginasOrganizacion struct {
	selector    SelectorOrganizacionHistorica
	instantanea InstantaneaComparacionOrganizacion
	ids         map[string]bool
	cursor      string
	paginas     int
	finalizada  bool
	invalida    bool
}

func NuevaReunionPaginasOrganizacion(s SelectorOrganizacionHistorica) (*ReunionPaginasOrganizacion, error) {
	if s.Validar() != nil || s.Cursor != "" {
		return nil, ErrComparacionOrganizacionHistoricaInvalida
	}
	return &ReunionPaginasOrganizacion{selector: s, ids: map[string]bool{}}, nil
}

func (r *ReunionPaginasOrganizacion) Agregar(p PaginaInstantaneaOrganizacion) error {
	if r == nil {
		return ErrComparacionOrganizacionHistoricaInvalida
	}
	fallo := func() error { r.invalida = true; return ErrComparacionOrganizacionHistoricaInvalida }
	if r.invalida || r.finalizada || r.paginas >= LimitePaginasComparacionOrganizacion {
		return fallo()
	}
	i := p.Instantanea
	s := i.Selector
	esperado := r.selector
	esperado.Cursor = r.cursor
	if s.Validar() != nil || s.OrganismoRef != esperado.OrganismoRef || s.UnidadClave != esperado.UnidadClave ||
		s.VigenteEn != esperado.VigenteEn || !s.ConocidoEn.Equal(esperado.ConocidoEn) || s.VersionRPTRef != esperado.VersionRPTRef ||
		s.VersionPlantillaRef != esperado.VersionPlantillaRef || s.Limite != esperado.Limite || s.Cursor != esperado.Cursor ||
		!cursorHistoricoValido(p.CursorSiguiente) || p.CursorSiguiente != "" && (p.CursorSiguiente == s.Cursor || r.ids["cursor:"+p.CursorSiguiente]) {
		return fallo()
	}
	if s.VersionRPTRef != "" && p.VersionRPTRef != s.VersionRPTRef || s.VersionPlantillaRef != "" && p.VersionPlantillaRef != s.VersionPlantillaRef {
		return fallo()
	}
	i.Selector.Cursor = ""
	i.Selector.VersionRPTRef = p.VersionRPTRef
	i.Selector.VersionPlantillaRef = p.VersionPlantillaRef
	n := cantidadHechosInstantanea(i)
	if n > s.Limite || n == 0 && p.CursorSiguiente != "" || !instantaneaComparacionValida(i) || cantidadHechosInstantanea(r.instantanea)+n > LimiteHechosComparacionOrganizacion {
		return fallo()
	}
	if r.paginas > 0 && (i.Selector.VersionRPTRef != r.instantanea.Selector.VersionRPTRef || i.Selector.VersionPlantillaRef != r.instantanea.Selector.VersionPlantillaRef || i.Cobertura != r.instantanea.Cobertura) {
		return fallo()
	}
	ids := idsInstantanea(i)
	vistos := map[string]bool{}
	for _, id := range ids {
		if r.ids["hecho:"+id] || vistos[id] {
			return fallo()
		}
		vistos[id] = true
	}
	if r.paginas == 0 {
		r.instantanea.Selector = i.Selector
		r.instantanea.Cobertura = i.Cobertura
	}
	for _, id := range ids {
		r.ids["hecho:"+id] = true
	}
	r.ids["cursor:"+s.Cursor] = true
	r.instantanea.Unidades = append(r.instantanea.Unidades, i.Unidades...)
	r.instantanea.PuestosTipo = append(r.instantanea.PuestosTipo, i.PuestosTipo...)
	r.instantanea.Dotaciones = append(r.instantanea.Dotaciones, i.Dotaciones...)
	r.instantanea.Plazas = append(r.instantanea.Plazas, i.Plazas...)
	r.instantanea.PuestosIndividuales = append(r.instantanea.PuestosIndividuales, i.PuestosIndividuales...)
	r.instantanea.Vinculos = append(r.instantanea.Vinculos, i.Vinculos...)
	r.cursor = p.CursorSiguiente
	r.finalizada = r.cursor == ""
	r.paginas++
	return nil
}

func (r *ReunionPaginasOrganizacion) Finalizar() (InstantaneaComparacionOrganizacion, error) {
	if r == nil || r.invalida || !r.finalizada || r.paginas == 0 {
		return InstantaneaComparacionOrganizacion{}, ErrComparacionOrganizacionHistoricaInvalida
	}
	i := r.instantanea
	i.Unidades = append([]UnidadOrganizacionHistorica{}, i.Unidades...)
	i.PuestosTipo = append([]PuestoTipoOrganizacionHistorica{}, i.PuestosTipo...)
	i.Dotaciones = append([]DotacionOrganizacionHistorica{}, i.Dotaciones...)
	i.Plazas = append([]PlazaOrganizacionHistorica{}, i.Plazas...)
	i.PuestosIndividuales = append([]PuestoIndividualOrganizacionHistorica{}, i.PuestosIndividuales...)
	i.Vinculos = append([]VinculoPlazaPuestoHistorico{}, i.Vinculos...)
	return i, nil
}

func ReunirPaginasOrganizacionHistorica(paginas []PaginaInstantaneaOrganizacion) (InstantaneaComparacionOrganizacion, error) {
	if len(paginas) == 0 || len(paginas) > LimitePaginasComparacionOrganizacion {
		return InstantaneaComparacionOrganizacion{}, ErrComparacionOrganizacionHistoricaInvalida
	}
	r, err := NuevaReunionPaginasOrganizacion(paginas[0].Instantanea.Selector)
	if err != nil {
		return InstantaneaComparacionOrganizacion{}, err
	}
	for _, p := range paginas {
		if err := r.Agregar(p); err != nil {
			return InstantaneaComparacionOrganizacion{}, err
		}
	}
	return r.Finalizar()
}

func cantidadHechosInstantanea(i InstantaneaComparacionOrganizacion) int {
	return len(i.Unidades) + len(i.PuestosTipo) + len(i.Dotaciones) + len(i.Plazas) + len(i.PuestosIndividuales) + len(i.Vinculos)
}
func idsInstantanea(i InstantaneaComparacionOrganizacion) []string {
	ids := make([]string, 0, cantidadHechosInstantanea(i))
	for _, v := range i.Unidades {
		ids = append(ids, v.Traza.ID)
	}
	for _, v := range i.PuestosTipo {
		ids = append(ids, v.Traza.ID)
	}
	for _, v := range i.Dotaciones {
		ids = append(ids, v.Traza.ID)
	}
	for _, v := range i.Plazas {
		ids = append(ids, v.Traza.ID)
	}
	for _, v := range i.PuestosIndividuales {
		ids = append(ids, v.Traza.ID)
	}
	for _, v := range i.Vinculos {
		ids = append(ids, v.Traza.ID)
	}
	return ids
}
