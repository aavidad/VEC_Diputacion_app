package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"vec-diputacion-granada/internal/modules/seleccion/domain"
	"vec-diputacion-granada/internal/modules/seleccion/ports"
)

type MaterialAportacionAdmision struct {
	MaterialS4 ports.MaterialAdmisionPreparacion  `json:"material_s4"`
	Propuesta  domain.PropuestaAportacionAdmision `json:"propuesta"`
}

// IdentificarAntecedenteAdmision comprueba la estructura S4 y calcula la huella
// de su DTO según la serialización local versionada. No es el archivo S3 ni una
// firma o aprobación. Ninguna identidad o autoridad procede de esa huella.
func IdentificarAntecedenteAdmision(ctx context.Context, m ports.MaterialAdmisionPreparacion) (domain.AntecedenteAdmision, error) {
	if _, err := PrepararAdmision(ctx, m); err != nil {
		return domain.AntecedenteAdmision{}, err
	}
	return identificarMaterialAdmision(m)
}

func identificarMaterialAdmision(m ports.MaterialAdmisionPreparacion) (domain.AntecedenteAdmision, error) {
	serializado, err := json.Marshal(struct {
		Esquema  string                            `json:"esquema"`
		Material ports.MaterialAdmisionPreparacion `json:"material"`
	}{domain.EsquemaMaterialAdmisionLocal, m})
	if err != nil || len(serializado) > 1024*1024 {
		return domain.AntecedenteAdmision{}, domain.ErrAportacionAdmision
	}
	huella := sha256.Sum256(serializado)
	return domain.AntecedenteAdmision{EsquemaMaterial: domain.EsquemaMaterialAdmisionLocal, PreparacionRef: m.PreparacionRef, Revision: m.Revision,
		HuellaMaterialSHA256: hex.EncodeToString(huella[:])}, nil
}

// PrepararAportacionAdmision reutiliza la revisión S4 vigente, comprueba el
// antecedente exacto y devuelve una propuesta separada. No cambia la revisión
// anterior, presenta documentos ni ejecuta el circuito administrativo.
func PrepararAportacionAdmision(ctx context.Context, m MaterialAportacionAdmision) (domain.AportacionAdmisionPreparada, error) {
	cero := domain.AportacionAdmisionPreparada{}
	revision, err := PrepararAdmision(ctx, m.MaterialS4)
	if err != nil {
		return cero, err
	}
	antecedente, err := identificarMaterialAdmision(m.MaterialS4)
	if err != nil {
		return cero, err
	}
	for _, requisito := range revision.Requisitos {
		if requisito.Requisito.Referencia == m.Propuesta.RequisitoRef {
			preparada, err := domain.PrepararAportacionAdmision(m.Propuesta, antecedente, requisito, revision.Bases, revision.SolicitudContexto)
			if err != nil {
				return cero, err
			}
			if err := ctx.Err(); err != nil {
				return cero, err
			}
			return preparada, nil
		}
	}
	return cero, domain.ErrAportacionAdmision
}
