package reglasbaremo

// IncorporaAprobacionExacta indica si esta version conserva exactamente la
// atestacion recibida en su transicion de publicacion. No compara solo una
// referencia: coteja todos los vinculos, firmantes y tiempos de la evidencia.
func (v VersionGobernadaReglasBaremo) IncorporaAprobacionExacta(
	evidencia AtestacionAprobacionFirmadaReglasBaremo,
) bool {
	return v.Validar() == nil && evidencia.validar() == nil && v.publicacion != nil &&
		aprobacionesExactamenteIguales(v.publicacion.aprobacion, evidencia)
}

// IncorporaDependenciasExactas comprueba la atestacion completa que llevo a
// la activacion. Las colecciones canonicas se comparan en su orden exacto.
func (v VersionGobernadaReglasBaremo) IncorporaDependenciasExactas(
	evidencia AtestacionDependenciasVigentesReglasBaremo,
) bool {
	return v.Validar() == nil && evidencia.validar() == nil && v.activacion != nil &&
		dependenciasExactamenteIguales(v.activacion.dependencias, evidencia)
}

// IncorporaAutoridadExacta comprueba la autoridad terminal completa, incluida
// la accion y la referencia relacionada cuando la sustitucion la exige.
func (v VersionGobernadaReglasBaremo) IncorporaAutoridadExacta(
	evidencia AtestacionAutoridadReglasBaremo,
) bool {
	return v.Validar() == nil && evidencia.validar() == nil && v.terminal != nil &&
		autoridadesExactamenteIguales(v.terminal.autoridad, evidencia)
}

func aprobacionesExactamenteIguales(
	a, b AtestacionAprobacionFirmadaReglasBaremo,
) bool {
	return referenciasVersionadasIguales(a.atestacion, b.atestacion) &&
		vinculosEstadoIguales(a.vinculo, b.vinculo) &&
		referenciasVersionadasIguales(a.firma, b.firma) &&
		referenciasVersionadasIguales(a.politicaFirma, b.politicaFirma) &&
		listasTextoExactamenteIguales(a.firmantes, b.firmantes) &&
		a.firmadaEn.Equal(b.firmadaEn) && a.verificadaEn.Equal(b.verificadaEn) &&
		a.validaHasta.Equal(b.validaHasta)
}

func dependenciasExactamenteIguales(
	a, b AtestacionDependenciasVigentesReglasBaremo,
) bool {
	return referenciasVersionadasIguales(a.atestacion, b.atestacion) &&
		vinculosEstadoIguales(a.vinculo, b.vinculo) &&
		referenciasVersionadasIguales(a.convocatoria, b.convocatoria) &&
		referenciasVersionadasIguales(a.bases, b.bases) &&
		listasReferenciasVersionadasIguales(a.dependencias, b.dependencias) &&
		a.verificadorRef == b.verificadorRef &&
		a.verificadaEn.Equal(b.verificadaEn) && a.validaHasta.Equal(b.validaHasta)
}

func autoridadesExactamenteIguales(a, b AtestacionAutoridadReglasBaremo) bool {
	return referenciasVersionadasIguales(a.atestacion, b.atestacion) &&
		vinculosEstadoIguales(a.vinculo, b.vinculo) && a.accion == b.accion &&
		a.principalRef == b.principalRef &&
		referenciasOpcionalesIguales(a.relacionada, b.relacionada) &&
		a.emitidaEn.Equal(b.emitidaEn) && a.validaHasta.Equal(b.validaHasta)
}

func listasTextoExactamenteIguales(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for indice := range a {
		if a[indice] != b[indice] {
			return false
		}
	}
	return true
}
