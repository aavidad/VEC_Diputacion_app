package reglasbaremo

func reconstruirVersionGobernada(
	material materialRestauracionVersionGobernada,
) (VersionGobernadaReglasBaremo, error) {
	conjunto, err := RestaurarConjuntoReglasBaremo(material.Contenido)
	if err != nil {
		return VersionGobernadaReglasBaremo{}, err
	}
	referenciaDeclarada, err := reconstruirReferenciaGobierno(material.ReferenciaContenido)
	if err != nil {
		return VersionGobernadaReglasBaremo{}, err
	}
	referenciaReal, err := conjunto.ReferenciaVersionada()
	if err != nil || !referenciasVersionadasIguales(referenciaDeclarada, referenciaReal) {
		return VersionGobernadaReglasBaremo{}, ErrGobiernoVinculoInexacto
	}
	motivo, err := reconstruirMotivoGobierno(material.MotivoCreacion)
	if err != nil {
		return VersionGobernadaReglasBaremo{}, err
	}
	version, err := NuevaVersionGobernadaReglasBaremo(
		conjunto, material.CreadaPor, motivo, material.CreadaEn,
	)
	if err != nil {
		return VersionGobernadaReglasBaremo{}, err
	}
	if material.Publicacion != nil {
		version, err = reconstruirPublicacionGobierno(version, *material.Publicacion)
		if err != nil {
			return VersionGobernadaReglasBaremo{}, err
		}
	}
	if material.Activacion != nil {
		version, err = reconstruirActivacionGobierno(version, *material.Activacion)
		if err != nil {
			return VersionGobernadaReglasBaremo{}, err
		}
	}
	if material.Terminal != nil {
		version, err = reconstruirTerminalGobierno(version, *material.Terminal)
		if err != nil {
			return VersionGobernadaReglasBaremo{}, err
		}
	}
	return version, nil
}

func reconstruirPublicacionGobierno(
	version VersionGobernadaReglasBaremo,
	material materialRestauracionPublicacion,
) (VersionGobernadaReglasBaremo, error) {
	motivo, err := reconstruirMotivoGobierno(material.Motivo)
	if err != nil {
		return VersionGobernadaReglasBaremo{}, err
	}
	aprobacion, err := reconstruirAprobacionGobierno(material.Aprobacion)
	if err != nil {
		return VersionGobernadaReglasBaremo{}, err
	}
	return version.Publicar(
		version.Revision(), material.ActorRef, motivo, aprobacion, material.Instante,
	)
}

func reconstruirActivacionGobierno(
	version VersionGobernadaReglasBaremo,
	material materialRestauracionActivacion,
) (VersionGobernadaReglasBaremo, error) {
	motivo, err := reconstruirMotivoGobierno(material.Motivo)
	if err != nil {
		return VersionGobernadaReglasBaremo{}, err
	}
	dependencias, err := reconstruirDependenciasGobierno(material.Dependencias)
	if err != nil {
		return VersionGobernadaReglasBaremo{}, err
	}
	return version.Activar(
		version.Revision(), material.ActorRef, motivo, dependencias, material.Instante,
	)
}

func reconstruirTerminalGobierno(
	version VersionGobernadaReglasBaremo,
	material materialTerminalGobiernoReglas,
) (VersionGobernadaReglasBaremo, error) {
	motivo, err := reconstruirMotivoGobierno(material.Motivo)
	if err != nil {
		return VersionGobernadaReglasBaremo{}, err
	}
	autoridad, err := reconstruirAutoridadGobierno(material.Autoridad)
	if err != nil {
		return VersionGobernadaReglasBaremo{}, err
	}
	switch material.Accion {
	case AccionSustituirReglasBaremo:
		relacionada, presente := autoridad.Relacionada()
		if !presente {
			return VersionGobernadaReglasBaremo{}, ErrGobiernoEvidenciaInvalida
		}
		return version.Sustituir(
			version.Revision(), material.ActorRef, motivo, relacionada,
			autoridad, material.Instante,
		)
	case AccionRetirarReglasBaremo:
		return version.Retirar(
			version.Revision(), material.ActorRef, motivo, autoridad, material.Instante,
		)
	case AccionDescartarReglasBaremo:
		return version.Descartar(
			version.Revision(), material.ActorRef, motivo, autoridad, material.Instante,
		)
	default:
		return VersionGobernadaReglasBaremo{}, ErrGobiernoTransicionProhibida
	}
}

func reconstruirReferenciaGobierno(
	material materialReferenciaGobiernoReglas,
) (ReferenciaVersionada, error) {
	return NuevaReferenciaVersionada(
		material.Referencia, material.Version, material.HuellaSHA256,
	)
}

func reconstruirMotivoGobierno(
	material materialMotivoGobiernoReglas,
) (MotivoCatalogadoReglasBaremo, error) {
	catalogo, err := reconstruirReferenciaGobierno(material.Catalogo)
	if err != nil {
		return MotivoCatalogadoReglasBaremo{}, ErrGobiernoValorInvalido
	}
	return NuevoMotivoCatalogadoReglasBaremo(catalogo, material.Clave)
}

func reconstruirVinculoGobierno(
	material materialVinculoGobiernoReglas,
) (VinculoEstadoReglasBaremo, error) {
	contenido, err := reconstruirReferenciaGobierno(material.Contenido)
	if err != nil {
		return VinculoEstadoReglasBaremo{}, ErrGobiernoVinculoInexacto
	}
	return NuevoVinculoEstadoReglasBaremo(
		contenido, material.Revision, material.HuellaEstadoSHA256,
	)
}
