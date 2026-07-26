package ports

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

const (
	dominioHuellaFlujoVisualRRHH    = "VEC-CT-PUBLICACION-FLUJO-VISUAL-RRHH-V1"
	dominioHuellaCatalogoVisualRRHH = "VEC-CT-PUBLICACION-CATALOGO-VISUAL-RRHH-V1"
)

func validarLimitesDefinicionFlujoVisualRRHH(
	flujo DefinicionFlujoVisualRRHH,
) error {
	if len(flujo.Fases) < 1 ||
		len(flujo.Fases) > MaximoFasesComposicionVisualRRHH ||
		len(flujo.Tareas) < 1 ||
		len(flujo.Tareas) > MaximoTareasComposicionVisualRRHH ||
		len(flujo.Paneles) < 1 ||
		len(flujo.Paneles) > MaximoPanelesComposicionVisualRRHH {
		return ErrResultadoComposicionVisualRRHHNoConfiable
	}
	totalCampos, totalOperaciones, totalReferenciasPaneles := 0, 0, 0
	for _, panel := range flujo.Paneles {
		if len(panel.Campos) > MaximoCamposComposicionVisualRRHH-totalCampos {
			return ErrResultadoComposicionVisualRRHHNoConfiable
		}
		totalCampos += len(panel.Campos)
	}
	for _, tarea := range flujo.Tareas {
		if len(tarea.Paneles) > MaximoPanelesPorTareaVisualRRHH ||
			len(tarea.Paneles) >
				MaximoReferenciasPanelesVisualRRHH-totalReferenciasPaneles ||
			len(tarea.Operaciones) >
				MaximoOperacionesComposicionVisualRRHH-totalOperaciones {
			return ErrResultadoComposicionVisualRRHHNoConfiable
		}
		totalReferenciasPaneles += len(tarea.Paneles)
		totalOperaciones += len(tarea.Operaciones)
	}
	return nil
}

// CalcularHuellaDefinicionFlujoVisualRRHH deriva la identidad del contenido
// exacto. La huella declarada queda excluida de la preimagen.
func CalcularHuellaDefinicionFlujoVisualRRHH(
	flujo DefinicionFlujoVisualRRHH,
) (string, error) {
	if validarLimitesDefinicionFlujoVisualRRHH(flujo) != nil {
		return "", ErrResultadoComposicionVisualRRHHNoConfiable
	}
	copia := normalizarDefinicionFlujoVisualRRHH(flujo)
	copia.Huella = ""
	return huellaCanonicaPublicacionVisualRRHH(
		dominioHuellaFlujoVisualRRHH, copia,
	)
}

// CalcularHuellaCatalogoVisualRRHH deriva la identidad del contenido exacto
// de todas las opciones publicadas.
func CalcularHuellaCatalogoVisualRRHH(
	catalogo CatalogoVisualRRHH,
) (string, error) {
	if len(catalogo.Opciones) < 1 ||
		len(catalogo.Opciones) > MaximoOpcionesCatalogoVisualRRHH {
		return "", ErrResultadoComposicionVisualRRHHNoConfiable
	}
	copia := catalogo
	copia.Huella = ""
	copia.Opciones = make(
		[]OpcionCatalogoVisualRRHH, len(catalogo.Opciones),
	)
	copy(copia.Opciones, catalogo.Opciones)
	return huellaCanonicaPublicacionVisualRRHH(
		dominioHuellaCatalogoVisualRRHH, copia,
	)
}

func huellaCanonicaPublicacionVisualRRHH(
	dominio string,
	publicacion any,
) (string, error) {
	contenido, err := json.Marshal(publicacion)
	if err != nil {
		return "", ErrResultadoComposicionVisualRRHHNoConfiable
	}
	material := make([]byte, 0, len(dominio)+1+len(contenido))
	material = append(material, dominio...)
	material = append(material, 0)
	material = append(material, contenido...)
	huella := sha256.Sum256(material)
	return hex.EncodeToString(huella[:]), nil
}

func normalizarDefinicionFlujoVisualRRHH(
	flujo DefinicionFlujoVisualRRHH,
) DefinicionFlujoVisualRRHH {
	copia := flujo
	copia.Fases = make([]FaseVisualRRHH, len(flujo.Fases))
	copy(copia.Fases, flujo.Fases)
	copia.Tareas = make([]TareaVisualRRHH, len(flujo.Tareas))
	for indice, tarea := range flujo.Tareas {
		copia.Tareas[indice] = tarea
		copia.Tareas[indice].Paneles = make([]string, len(tarea.Paneles))
		copy(copia.Tareas[indice].Paneles, tarea.Paneles)
		copia.Tareas[indice].Operaciones = make(
			[]OperacionVisualRRHH, len(tarea.Operaciones),
		)
		copy(copia.Tareas[indice].Operaciones, tarea.Operaciones)
	}
	copia.Paneles = make([]PanelVisualRRHH, len(flujo.Paneles))
	for indice, panel := range flujo.Paneles {
		copia.Paneles[indice] = panel
		copia.Paneles[indice].Campos = make(
			[]CampoVisualRRHH, len(panel.Campos),
		)
		copy(copia.Paneles[indice].Campos, panel.Campos)
	}
	return copia
}

func validarRelacionesComposicionVisualRRHH(
	composicion ComposicionVisualRRHH,
	catalogos map[string]CatalogoVisualRRHH,
) error {
	fases := make(map[domain.ClaveFase]struct{}, len(composicion.Flujo.Fases))
	for _, fase := range composicion.Flujo.Fases {
		fases[fase.Clave] = struct{}{}
	}
	paneles, catalogosUsados, operaciones, err :=
		validarPanelesVisualesRRHH(composicion.Flujo.Paneles, catalogos)
	if err != nil {
		return err
	}
	fasesUsadas, panelesUsados, err := validarTareasVisualesRRHH(
		composicion.Flujo.Tareas, fases, paneles, operaciones,
	)
	if err != nil || len(fasesUsadas) != len(fases) ||
		len(panelesUsados) != len(paneles) ||
		len(catalogosUsados) != len(catalogos) {
		return ErrResultadoComposicionVisualRRHHNoConfiable
	}
	return validarCapacidadesVisualesRRHH(composicion.Capacidades, operaciones)
}

func validarPanelesVisualesRRHH(
	origen []PanelVisualRRHH,
	catalogos map[string]CatalogoVisualRRHH,
) (
	map[string]PanelVisualRRHH,
	map[string]struct{},
	map[string]OperacionVisualRRHH,
	error,
) {
	paneles := make(map[string]PanelVisualRRHH, len(origen))
	catalogosUsados := make(map[string]struct{}, len(catalogos))
	ordenes := make(map[uint16]struct{}, len(origen))
	for _, panel := range origen {
		if !domain.ReferenciaOpacaValida(panel.Referencia) ||
			panel.Orden < 1 || !panel.Tipo.valido() ||
			!claveI18nVisualRRHHValida(panel.ClaveI18n) {
			return nil, nil, nil, ErrResultadoComposicionVisualRRHHNoConfiable
		}
		if _, repetido := paneles[panel.Referencia]; repetido {
			return nil, nil, nil, ErrResultadoComposicionVisualRRHHNoConfiable
		}
		if _, repetido := ordenes[panel.Orden]; repetido {
			return nil, nil, nil, ErrResultadoComposicionVisualRRHHNoConfiable
		}
		if validarCamposVisualesRRHH(
			panel.Campos, catalogos, catalogosUsados,
		) != nil {
			return nil, nil, nil, ErrResultadoComposicionVisualRRHHNoConfiable
		}
		paneles[panel.Referencia], ordenes[panel.Orden] = panel, struct{}{}
	}
	return paneles, catalogosUsados,
		make(map[string]OperacionVisualRRHH), nil
}

func validarCamposVisualesRRHH(
	campos []CampoVisualRRHH,
	catalogos map[string]CatalogoVisualRRHH,
	catalogosUsados map[string]struct{},
) error {
	claves := make(map[string]struct{}, len(campos))
	ordenes := make(map[uint16]struct{}, len(campos))
	for _, campo := range campos {
		if !claveVisualRRHHValida(campo.Clave) || campo.Orden < 1 ||
			!claveI18nVisualRRHHValida(campo.ClaveI18n) ||
			!campo.Control.valido() {
			return ErrResultadoComposicionVisualRRHHNoConfiable
		}
		if _, repetida := claves[campo.Clave]; repetida {
			return ErrResultadoComposicionVisualRRHHNoConfiable
		}
		if _, repetido := ordenes[campo.Orden]; repetido {
			return ErrResultadoComposicionVisualRRHHNoConfiable
		}
		claves[campo.Clave], ordenes[campo.Orden] = struct{}{}, struct{}{}
		requiereCatalogo := campo.Control == ControlVisualSeleccion ||
			campo.Control == ControlVisualRadio
		tieneCatalogo := campo.CatalogoRef != "" || campo.CatalogoVersion != 0
		if requiereCatalogo != tieneCatalogo ||
			(tieneCatalogo && (!domain.ReferenciaOpacaValida(campo.CatalogoRef) ||
				campo.CatalogoVersion < 1 ||
				campo.CatalogoVersion > versionMaximaJSONSegura)) {
			return ErrResultadoComposicionVisualRRHHNoConfiable
		}
		if !tieneCatalogo {
			continue
		}
		identidad := identidadCatalogoVisualRRHH(
			campo.CatalogoRef, campo.CatalogoVersion,
		)
		if _, existe := catalogos[identidad]; !existe {
			return ErrResultadoComposicionVisualRRHHNoConfiable
		}
		catalogosUsados[identidad] = struct{}{}
	}
	return nil
}

func validarTareasVisualesRRHH(
	tareas []TareaVisualRRHH,
	fases map[domain.ClaveFase]struct{},
	paneles map[string]PanelVisualRRHH,
	operaciones map[string]OperacionVisualRRHH,
) (map[domain.ClaveFase]struct{}, map[string]struct{}, error) {
	referencias := make(map[string]struct{}, len(tareas))
	ordenes := make(map[string]struct{}, len(tareas))
	fasesUsadas := make(map[domain.ClaveFase]struct{}, len(fases))
	panelesUsados := make(map[string]struct{}, len(paneles))
	for _, tarea := range tareas {
		if !domain.ReferenciaOpacaValida(tarea.Referencia) ||
			tarea.Orden < 1 || !claveI18nVisualRRHHValida(tarea.ClaveI18n) ||
			len(tarea.Paneles) < 1 {
			return nil, nil, ErrResultadoComposicionVisualRRHHNoConfiable
		}
		if _, existe := fases[tarea.FaseClave]; !existe {
			return nil, nil, ErrResultadoComposicionVisualRRHHNoConfiable
		}
		if _, repetida := referencias[tarea.Referencia]; repetida {
			return nil, nil, ErrResultadoComposicionVisualRRHHNoConfiable
		}
		orden := string(tarea.FaseClave) + "\x00" +
			strconv.FormatUint(uint64(tarea.Orden), 10)
		if _, repetido := ordenes[orden]; repetido {
			return nil, nil, ErrResultadoComposicionVisualRRHHNoConfiable
		}
		referencias[tarea.Referencia], ordenes[orden] = struct{}{}, struct{}{}
		fasesUsadas[tarea.FaseClave] = struct{}{}
		vistos := make(map[string]struct{}, len(tarea.Paneles))
		for _, panelRef := range tarea.Paneles {
			if _, existe := paneles[panelRef]; !existe {
				return nil, nil, ErrResultadoComposicionVisualRRHHNoConfiable
			}
			if _, repetido := vistos[panelRef]; repetido {
				return nil, nil, ErrResultadoComposicionVisualRRHHNoConfiable
			}
			vistos[panelRef], panelesUsados[panelRef] = struct{}{}, struct{}{}
		}
		for _, operacion := range tarea.Operaciones {
			if !claveVisualRRHHValida(operacion.Clave) ||
				!claveI18nVisualRRHHValida(operacion.ClaveI18n) ||
				!claveVocabularioComposicionVisualValida(
					operacion.CapacidadClave,
				) {
				return nil, nil, ErrResultadoComposicionVisualRRHHNoConfiable
			}
			if _, repetida := operaciones[operacion.Clave]; repetida {
				return nil, nil, ErrResultadoComposicionVisualRRHHNoConfiable
			}
			operaciones[operacion.Clave] = operacion
		}
	}
	return fasesUsadas, panelesUsados, nil
}

func validarCapacidadesVisualesRRHH(
	capacidades []CapacidadVisualConcedidaRRHH,
	operaciones map[string]OperacionVisualRRHH,
) error {
	vistas := make(map[string]struct{}, len(capacidades))
	for _, capacidad := range capacidades {
		operacion, soportada := operaciones[capacidad.OperacionClave]
		if !soportada ||
			capacidad.CapacidadClave != operacion.CapacidadClave {
			return ErrResultadoComposicionVisualRRHHNoConfiable
		}
		identidad := capacidad.OperacionClave + "\x00" +
			capacidad.CapacidadClave
		if _, repetida := vistas[identidad]; repetida {
			return ErrResultadoComposicionVisualRRHHNoConfiable
		}
		vistas[identidad] = struct{}{}
	}
	return nil
}
