package plannominal

import (
	"context"
	"reflect"
	"strconv"
	"time"

	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	vd "vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/reglas"
)

// PublicacionAutorizada coteja versión, bytes y procedencia contra la autoridad
// de gobierno existente. Los campos publicado_por/aprobacion_ref de un fichero
// y una marca demostracion=false nunca sustituyen este cotejo.
type PublicacionAutorizada interface {
	ComprobarPublicacionPlanFirmaV2(context.Context, vd.CatalogoConfigurable, string, time.Time) error
}

type Fuente struct {
	resolutor   *reglas.Resolutor
	publicacion PublicacionAutorizada
	version     ct.VersionPlanFirmaV2
}

func NuevaFuente(resolutor *reglas.Resolutor, publicacion PublicacionAutorizada, version ct.VersionPlanFirmaV2) (*Fuente, error) {
	if resolutor == nil || !version.Valida() || resolutor.CatalogoID() != version.Referencia || nula(publicacion) {
		return nil, ct.ErrPlanCompetenciaFirmaV2
	}
	return &Fuente{resolutor: resolutor, publicacion: publicacion, version: version}, nil
}

func nula(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Func, reflect.Map, reflect.Slice, reflect.Chan:
		return r.IsNil()
	}
	return false
}

// Plan relee publicación y vigencias; no conserva cachés de autorización.
func (f *Fuente) Plan(ctx context.Context) (ct.PlanCompetenciaFirmaV2, error) {
	if f == nil || ctx == nil || ctx.Err() != nil {
		return ct.PlanCompetenciaFirmaV2{}, ct.ErrPlanCompetenciaFirmaV2
	}
	c, sha, en, err := f.resolutor.CatalogoVigente(ctx)
	if err != nil || c.ID != f.version.Referencia || uint64(c.Version) != f.version.Version || sha != f.version.HuellaSHA256 {
		return ct.PlanCompetenciaFirmaV2{}, ct.ErrPlanCompetenciaFirmaV2
	}
	plan, err := DesdeCatalogo(c, en)
	if err != nil {
		return ct.PlanCompetenciaFirmaV2{}, err
	}
	if err := f.publicacion.ComprobarPublicacionPlanFirmaV2(ctx, c, sha, en); err != nil || ctx.Err() != nil {
		return ct.PlanCompetenciaFirmaV2{}, ct.ErrPlanCompetenciaFirmaV2
	}
	return plan, nil
}

// DesdeCatalogo solo interpreta datos. Su resultado no es autorización ni
// acredita publicación, aunque la estructura declare un estado publicado.
func DesdeCatalogo(c vd.CatalogoConfigurable, en time.Time) (ct.PlanCompetenciaFirmaV2, error) {
	c, err := c.ClonarCanonico()
	if err != nil || en.IsZero() || c.ModuloID != "contratacion_temporal" || c.FuenteRef == reglas.MarcaPaqueteEjemplo || c.PublicadoEn.After(en) || c.PublicadoEn.IsZero() || (c.Estado != vd.EstadoCatalogoPublicado && !(c.Estado == vd.EstadoCatalogoRetirado && c.RetiradoEn.After(en))) {
		return ct.PlanCompetenciaFirmaV2{}, ct.ErrPlanCompetenciaFirmaV2
	}
	sha, err := c.HuellaSHA256()
	if err != nil {
		return ct.PlanCompetenciaFirmaV2{}, ct.ErrPlanCompetenciaFirmaV2
	}
	p := ct.PlanCompetenciaFirmaV2{Version: ct.VersionPlanFirmaV2{Referencia: c.ID, Version: uint64(c.Version), HuellaSHA256: sha}, FuenteRef: c.FuenteRef}
	for _, e := range c.Entradas {
		if !e.VigenteEn(en) {
			continue
		}
		a := e.Atributos
		if a["esquema"] != ct.EsquemaPlanCompetenciaFirmaV2 || a["origen"] == "ejemplo" {
			return ct.PlanCompetenciaFirmaV2{}, ct.ErrPlanCompetenciaFirmaV2
		}
		cv, ok1 := numero(a["circuito_version"])
		orden, ok2 := numero(a["paso_orden"])
		mv, ok3 := numero(a["mapeo_version"])
		if !ok1 || !ok2 || !ok3 {
			return ct.PlanCompetenciaFirmaV2{}, ct.ErrPlanCompetenciaFirmaV2
		}
		p.Pasos = append(p.Pasos, ct.CompetenciaPasoFirmaV2{EntradaClave: e.Clave, Circuito: ct.VersionPlanFirmaV2{Referencia: a["circuito_ref"], Version: cv, HuellaSHA256: a["circuito_sha256"]}, Documento: a["documento"], PasoRef: a["paso_ref"], PasoOrden: orden, PerfilEsperadoRef: a["perfil_esperado_ref"], RolID: a["rol_id"], CargoRef: a["cargo_ref"], OrganizacionRef: a["organizacion_ref"], UnidadRef: a["unidad_ref"], Accion: a["accion_competencial"], Finalidad: a["finalidad"], TipoRecurso: a["tipo_recurso"], EsquemaContexto: a["esquema_contexto"], MapeoVersion: mv, MapeoFuenteRef: a["mapeo_fuente_ref"]})
	}
	if p.Validar() != nil {
		return ct.PlanCompetenciaFirmaV2{}, ct.ErrPlanCompetenciaFirmaV2
	}
	return p, nil
}

func numero(s string) (uint64, bool) {
	v, err := strconv.ParseUint(s, 10, 64)
	return v, err == nil && strconv.FormatUint(v, 10) == s && v > 0
}
