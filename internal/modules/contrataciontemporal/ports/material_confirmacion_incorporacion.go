package ports

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	core "vec-diputacion-granada/internal/vec/domain"
)

const (
	EsquemaMaterialConfirmacionIncorporacionV2 = "vec.contratacion-temporal.incorporacion.ejercicio.material.v2"
	TipoRecursoConfirmacionIncorporacionV2     = "confirmacion_incorporacion_ejercicio_v2"
	AudienciaConfirmacionIncorporacionV2       = "vec_contratacion_temporal.incorporacion_ejercicio.v2"
)

var ErrMaterialConfirmacionIncorporacionV2 = errors.New("contratacion temporal: material de incorporacion de ejercicio v2 invalido")

// DatosMaterialConfirmacionIncorporacionV2 solo procede de preparación servidor.
// VersionActualExpediente NO es VersionExpediente de la solicitud histórica ni
// VersionSeguimientoEsperada. No impone equivalencias o avances de expediente.
type DatosMaterialConfirmacionIncorporacionV2 struct {
	Confirmacion            DatosConfirmacionIncorporacion
	VersionActualExpediente uint64
	Preparacion             PreparacionSeguimientoConfirmacionIncorporacion
	SolicitudContexto       SolicitudResolverContextoAutorizacionAltaV3
	Contexto                ContextoAutorizacionAltaV3
	MotivoV3                core.ReferenciaEntradaCatalogo
	CorrelacionV3           core.ReferenciaCorrelacionAutorizacionV2
	Personal                RegistroPersonalEjercicio
	EjercicioSintetico      bool
	FirmaOficial            bool
	EficaciaAdministrativa  bool
}

// MaterialConfirmacionIncorporacionV2 compromete datos, no concede permiso ni
// acredita persistencia. No contiene una decisión para evitar un hash circular.
type MaterialConfirmacionIncorporacionV2 struct {
	datos       *DatosMaterialConfirmacionIncorporacionV2
	preparadoEn time.Time
}

func NuevoMaterialConfirmacionIncorporacionV2(d DatosMaterialConfirmacionIncorporacionV2, ahora time.Time) (MaterialConfirmacionIncorporacionV2, error) {
	n, err := normalizarDatosConfirmacionIncorporacion(d.Confirmacion)
	v, ev := d.Contexto.Vinculo.Datos()
	_, ec := d.CorrelacionV3.ValorCanonico()
	_, em := core.HuellaSHA256MotivoAutorizacionV2(d.MotivoV3)
	if err != nil || ev != nil || ec != nil || em != nil || d.Preparacion.validar() != nil ||
		d.Contexto.ValidarPara(d.SolicitudContexto, ahora) != nil || v.GarantiaObservada != core.AuthAssuranceHigh ||
		(v.Superficie != core.SuperficieAutenticacionInternaCorporativaV1 && v.Superficie != core.SuperficieAutenticacionAdministracionPrivilegiadaV1) ||
		d.VersionActualExpediente == 0 || d.VersionActualExpediente > MaximoEnteroSeguroOperacionAnalisis ||
		d.Personal.ValidarEstructuraPara(n.SolicitudPersonal, ahora) != nil || d.Personal.Resultado != n.ResultadoPersonal ||
		!d.EjercicioSintetico || d.FirmaOficial || d.EficaciaAdministrativa {
		return MaterialConfirmacionIncorporacionV2{}, ErrMaterialConfirmacionIncorporacionV2
	}
	d.Confirmacion = n
	d.Contexto.Resultado, err = d.Contexto.Resultado.Clonar()
	if err != nil {
		return MaterialConfirmacionIncorporacionV2{}, ErrMaterialConfirmacionIncorporacionV2
	}
	d.Personal = d.Personal.copiar()
	return MaterialConfirmacionIncorporacionV2{datos: &d, preparadoEn: ahora}, nil
}

func (m MaterialConfirmacionIncorporacionV2) Datos() (DatosMaterialConfirmacionIncorporacionV2, error) {
	if m.datos == nil {
		return DatosMaterialConfirmacionIncorporacionV2{}, ErrMaterialConfirmacionIncorporacionV2
	}
	n, err := NuevoMaterialConfirmacionIncorporacionV2(*m.datos, m.preparadoEn)
	if err != nil {
		return DatosMaterialConfirmacionIncorporacionV2{}, err
	}
	return *n.datos, nil
}

// MaterialCanonico usa JSON de un DTO cerrado y versionado, sin mapas ni
// autoridad serializada por el canal. Los bytes V3 conservan SU canon original.
// Documentos se ordenan mediante el normalizador existente; no hay truncado.
func (m MaterialConfirmacionIncorporacionV2) MaterialCanonico() ([]byte, error) {
	d, err := m.Datos()
	if err != nil {
		return nil, err
	}
	v, err := d.Contexto.Vinculo.Datos()
	if err != nil {
		return nil, ErrMaterialConfirmacionIncorporacionV2
	}
	cor, _ := d.CorrelacionV3.ValorCanonico()
	return json.Marshal(struct {
		Esquema                                                  string
		Confirmacion                                             DatosConfirmacionIncorporacion
		VersionActualExpediente                                  uint64
		Preparacion                                              PreparacionSeguimientoConfirmacionIncorporacion
		SolicitudContexto                                        SolicitudResolverContextoAutorizacionAltaV3
		Vinculo                                                  core.DatosVinculoAutenticacionActorV2
		ContextoCanonico, ProcedenciaCanonica                    []byte
		MotivoV3                                                 core.ReferenciaEntradaCatalogo
		CorrelacionV3                                            string
		Personal                                                 RegistroPersonalEjercicio
		EjercicioSintetico, FirmaOficial, EficaciaAdministrativa bool
	}{EsquemaMaterialConfirmacionIncorporacionV2, d.Confirmacion, d.VersionActualExpediente,
		d.Preparacion, d.SolicitudContexto, v, d.Contexto.Resultado.RepresentacionCanonica,
		d.Contexto.Resultado.ManifiestoProcedenciaCanonico, d.MotivoV3, cor, d.Personal, true, false, false})
}

func (m MaterialConfirmacionIncorporacionV2) HuellaSHA256() (string, error) {
	b, err := m.MaterialCanonico()
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}

// RecursoConfirmacionIncorporacionV2 no acepta el recurso v1 ni un hash
// opcional. Los pares de actor/correlación siguen separados, nunca traducidos.
func RecursoConfirmacionIncorporacionV2(m MaterialConfirmacionIncorporacionV2) (core.RecursoAutorizable, error) {
	d, err := m.Datos()
	if err != nil {
		return core.RecursoAutorizable{}, err
	}
	h, err := m.HuellaSHA256()
	if err != nil {
		return core.RecursoAutorizable{}, err
	}
	v, _ := d.Contexto.Vinculo.Datos()
	cor, _ := d.CorrelacionV3.ValorCanonico()
	r := core.RecursoAutorizable{Referencia: d.Confirmacion.SolicitudPersonal.ExpedienteRef,
		ModuloID: ModuloContratacion, Tipo: TipoRecursoConfirmacionIncorporacionV2,
		Ambitos: map[string]string{"organizacion_ref": d.Preparacion.OrganizacionRef, "unidad_ref": d.Preparacion.UnidadRef},
		Atributos: map[string]string{"material_sha256": h, "principal_v3_ref": v.PrincipalID,
			"perfil_v3_ref": v.PerfilActivoRef, "actor_seguimiento_ref": d.Preparacion.ActorRef,
			"correlacion_v3_ref": cor, "correlacion_seguimiento_ref": d.Preparacion.CorrelacionRef,
			"motivo_v3_ref": d.MotivoV3.EntradaClave, "tipo_validacion": "ejercicio_sintetico"}}
	if _, err := r.HuellaContextoAutorizacionSHA256(); err != nil {
		return core.RecursoAutorizable{}, ErrMaterialConfirmacionIncorporacionV2
	}
	return r, nil
}

func (m MaterialConfirmacionIncorporacionV2) validarEn(ahora time.Time) error {
	d, err := m.Datos()
	if err != nil || !domain.InstanteUTCCanonico(ahora) || ahora.Before(m.preparadoEn) ||
		d.Contexto.ValidarPara(d.SolicitudContexto, ahora) != nil {
		return ErrMaterialConfirmacionIncorporacionV2
	}
	return nil
}
