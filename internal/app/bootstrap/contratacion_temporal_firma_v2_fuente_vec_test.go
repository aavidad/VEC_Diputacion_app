package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/app/composicion/interna/contrataciontemporal/firmavec"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
)

type competenciaFirmaVecPrueba struct {
	e        ports.EvidenciaCompetenciaFirmante
	llamadas int
}

func (f *competenciaFirmaVecPrueba) AcreditarCompetenciaFirmante(_ context.Context,
	q ports.SolicitudCompetenciaFirmante) (ports.EvidenciaCompetenciaFirmante, error) {
	f.llamadas++
	e := f.e
	if e.Solicitud == (ports.SolicitudCompetenciaFirmante{}) {
		e.Solicitud = q
	}
	return e, nil
}

func solicitudCompetenciaFirmaVecPrueba() ports.SolicitudCompetenciaFirmante {
	huella := strings.Repeat("a", 64)
	return ports.SolicitudCompetenciaFirmante{CertificadoHuella: huella, FirmanteRef: "ref:" + huella,
		OrganizacionRef: "org_0123456789abcdefghijkl", ExpedienteRef: "exp_0123456789abcdefghijkl",
		Documento: "informe", PasoRef: "pas_0123456789abcdefghijkl", PasoOrden: 1}
}

func evidenciaCompetenciaFirmaVecPrueba() ports.EvidenciaCompetenciaFirmante {
	return ports.EvidenciaCompetenciaFirmante{FirmantePrincipalRef: "per_0123456789abcdefghijkl",
		CuentaFirmanteRef: "cta_0123456789abcdefghijkl", PerfilActivoFirmanteRef: "prf_0123456789abcdefghijkl",
		VinculoCredencialFirmanteRef: "vcr_0123456789abcdefghijkl", VinculoCredencialFirmanteRevision: 1,
		VinculoCredencialFirmanteHuella: strings.Repeat("b", 64), RolIDFirmante: "rol_firma_vec_prueba", Vigente: true}
}

func TestFuenteCompetenciaFirmaVecV2SellaSeleccionAUT56DeCA25(t *testing.T) {
	delegada := &competenciaFirmaVecPrueba{e: evidenciaCompetenciaFirmaVecPrueba()}
	f, err := nuevaFuenteCompetenciaFirmaVecV2(delegada)
	if err != nil {
		t.Fatal(err)
	}
	r, err := prepararPeticionFirmaVecV2(httptest.NewRequest(http.MethodPost, httpinterno.RutaRegistroFirmaVec, nil))
	if err != nil {
		t.Fatal(err)
	}
	q := solicitudCompetenciaFirmaVecPrueba()
	ctx := context.WithValue(r.Context(), struct{}{}, "correlacion")
	e, err := f.AcreditarCompetenciaFirmante(ctx, q)
	if err != nil || e.Solicitud != q || delegada.llamadas != 1 {
		t.Fatalf("selección central: %v", err)
	}
	c, ok := competenciaFirmaVecV2DesdeContexto(ctx)
	guardada, peticion, presente := c.Leer()
	if !ok || !presente || peticion != r || guardada != e {
		t.Fatal("selección AUT56 no quedó sellada en la petición")
	}
	if _, err := f.AcreditarCompetenciaFirmante(ctx, q); err != nil {
		t.Fatal("misma selección idempotente", err)
	}
	delegada.e.FirmantePrincipalRef = "per_otra_persona_00000000"
	if _, err := f.AcreditarCompetenciaFirmante(ctx, q); !errors.Is(err, ports.ErrCompetenciaFirmanteNoDisponible) {
		t.Fatal("segunda selección distinta admitida", err)
	}
	if _, _, presente := c.Leer(); presente {
		t.Fatal("contenedor ambiguo todavía legible")
	}
}

func TestFuenteCompetenciaFirmaVecV2DeniegaCrucesYConservaViaExterna(t *testing.T) {
	for caso, preparar := range map[string]func(*ports.SolicitudCompetenciaFirmante, *competenciaFirmaVecPrueba){
		"ref_ca25_ajena": func(q *ports.SolicitudCompetenciaFirmante, _ *competenciaFirmaVecPrueba) {
			q.FirmanteRef = "ref:" + strings.Repeat("c", 64)
		},
		"huella_mala": func(q *ports.SolicitudCompetenciaFirmante, _ *competenciaFirmaVecPrueba) {
			q.CertificadoHuella = "otra"
		},
		"perfil_externo": func(_ *ports.SolicitudCompetenciaFirmante, f *competenciaFirmaVecPrueba) {
			f.e.RolIDFirmante = rolFirmaExternaRegistroCTDesarrollo
		},
		"sin_vinculo": func(_ *ports.SolicitudCompetenciaFirmante, f *competenciaFirmaVecPrueba) {
			f.e.VinculoCredencialFirmanteRef = ""
		},
		"otra_solicitud": func(_ *ports.SolicitudCompetenciaFirmante, f *competenciaFirmaVecPrueba) {
			f.e.Solicitud = ports.SolicitudCompetenciaFirmante{FirmanteRef: "ref:otra"}
		},
	} {
		t.Run(caso, func(t *testing.T) {
			delegada := &competenciaFirmaVecPrueba{e: evidenciaCompetenciaFirmaVecPrueba()}
			f, _ := nuevaFuenteCompetenciaFirmaVecV2(delegada)
			r, _ := prepararPeticionFirmaVecV2(httptest.NewRequest(http.MethodPost, httpinterno.RutaRegistroFirmaVec, nil))
			q := solicitudCompetenciaFirmaVecPrueba()
			preparar(&q, delegada)
			if _, err := f.AcreditarCompetenciaFirmante(r.Context(), q); !errors.Is(err, ports.ErrCompetenciaFirmanteNoAcreditada) {
				t.Fatal("cruce admitido", err)
			}
			c, _ := competenciaFirmaVecV2DesdeContexto(r.Context())
			if _, _, presente := c.Leer(); presente {
				t.Fatal("cruce guardado")
			}
		})
	}
	delegada := &competenciaFirmaVecPrueba{e: evidenciaCompetenciaFirmaVecPrueba()}
	f, _ := nuevaFuenteCompetenciaFirmaVecV2(delegada)
	if _, err := f.AcreditarCompetenciaFirmante(context.Background(), solicitudCompetenciaFirmaVecPrueba()); err != nil || delegada.llamadas != 1 {
		t.Fatal("la vía externa no conserva su fuente", err)
	}
	for _, ruta := range []string{httpinterno.RutaRegistroFirmaVec + "?x=1", httpinterno.RutaRegistroFirmaExterna} {
		if _, err := prepararPeticionFirmaVecV2(httptest.NewRequest(http.MethodPost, ruta, nil)); err == nil {
			t.Fatal("otra ruta preparada", ruta)
		}
	}
}

func TestFuenteNominalFirmaVecV2AbreUnaSesionYRevalidaEnCadaUso(t *testing.T) {
	e := nuevoEntornoFirmanteV2Prueba(t)
	proyectarEmpleadoFirmanteV2Prueba(t, &e)
	r, err := prepararPeticionFirmaVecV2(e.r)
	if err != nil {
		t.Fatal(err)
	}
	q := solicitudCompetenciaFirmaVecPrueba()
	q.CertificadoHuella, q.FirmanteRef = e.huella, "ref:"+e.huella
	delegada := &competenciaFirmaVecPrueba{e: evidenciaCompetenciaFirmaVecPrueba()}
	competencia, err := nuevaFuenteCompetenciaFirmaVecV2(delegada)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := competencia.AcreditarCompetenciaFirmante(r.Context(), q); err != nil {
		t.Fatal(err)
	}
	f, err := firmavec.NuevaFuenteNominalFirmaVecV2(e.a)
	if err != nil {
		t.Fatal(err)
	}
	uno, err := f.RevalidarContextoActorFirmaV2(r.Context())
	if err != nil || uno.Resultado.Contexto.PersonaRef != e.persona || uno.CertificadoCanalSHA256 != e.huella {
		t.Fatalf("sesión inicial: %v", err)
	}
	dos, err := f.RevalidarContextoActorFirmaV2(r.Context())
	if err != nil || dos.Vinculo.ValidarPara(dos.Resultado) != nil || len(e.registro.altas) != 1 ||
		e.contextos.base.llamadas != 2 {
		t.Fatalf("revalidación sin otra alta: %v", err)
	}
	consulta, err := f.ResolverContextoConsultaFirmasR5V2(r.Context())
	if err != nil || consulta.OrganizacionRef != q.OrganizacionRef || consulta.FirmantePrincipalCandidatoRef != e.persona {
		t.Fatalf("consulta R5 del firmante: %v", err)
	}
	ajena, err := firmavec.NuevaFuenteNominalFirmaVecV2(&autoridadSesionFirmanteV2{})
	if err != nil {
		t.Fatal(err)
	}
	cruzada, err := ajena.RevalidarContextoActorFirmaV2(r.Context())
	if !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || cruzada.Resultado.Contexto.PersonaRef != "" {
		t.Fatalf("otra autoridad reutilizó la sesión AUT56: err=%v", err)
	}
	// La lectura ajena invalida el contenedor para cualquier uso posterior,
	// igual que la cápsula anterior al traslado al paquete neutral.
	if _, err := f.RevalidarContextoActorFirmaV2(r.Context()); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) {
		t.Fatal("contenedor cruzado todavía legible", err)
	}
	if _, err := f.RevalidarContextoActorFirmaV2(context.Background()); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) {
		t.Fatal("fuente sin contenedor admitida", err)
	}
}

func TestFirmaVecV2PDPAdmiteSoloMotivoRutaYUnidad(t *testing.T) {
	h := strings.Repeat("a", 64)
	motivo := motivoFirmaV2CTDesarrollo()
	recurso := core.RecursoAutorizable{Referencia: ports.PrefijoRecursoFirmaVec + "clave-firma-vec-prueba-0001",
		ModuloID: ports.ModuloContratacion, Tipo: ports.TipoRecursoFirmaVec,
		Ambitos:   map[string]string{"organizacion_ref": "org_0123456789abcdefghijkl", "unidad_ref": "uni_0123456789abcdefghijkl"},
		Atributos: map[string]string{"material_sha256": h, "descriptor_firma_sha256": h}}
	datos := core.DatosSolicitudAutorizacionLigadaV3{Accion: ports.AccionRegistrarFirmaVec,
		Finalidad: ports.FinalidadFirmaDocumento, ReferenciaMotivo: motivo, Recurso: recurso}
	if !solicitudAutorizacionFirmaVecV2CTValida(datos, motivo) {
		t.Fatal("firma VEC del paso denegada")
	}
	plan := datos
	plan.Recurso.Atributos = map[string]string{"material_sha256": h, "plan_firma_sha256": h}
	if !solicitudAutorizacionFirmaVecV2CTValida(plan, motivo) {
		t.Fatal("plan de firma VEC denegado")
	}
	consulta := datos
	consulta.Accion = ports.AccionConsultarFirmasR5V2
	consulta.Recurso.Tipo = ports.TipoRecursoConsultaFirmasR5
	consulta.Recurso.Referencia = "exp_0123456789abcdefghijkl"
	consulta.Recurso.Atributos = map[string]string{"material_sha256": h}
	if !solicitudAutorizacionFirmaVecV2CTValida(consulta, motivo) {
		t.Fatal("consulta previa del firmante denegada")
	}
	for caso, cambiar := range map[string]func(*core.DatosSolicitudAutorizacionLigadaV3){
		"otra_accion":        func(d *core.DatosSolicitudAutorizacionLigadaV3) { d.Accion = ports.AccionRegistrarFirmaExterna },
		"sin_unidad":         func(d *core.DatosSolicitudAutorizacionLigadaV3) { delete(d.Recurso.Ambitos, "unidad_ref") },
		"otro_tipo":          func(d *core.DatosSolicitudAutorizacionLigadaV3) { d.Recurso.Tipo = ports.TipoRecursoFirmaExterna },
		"ambos_descriptores": func(d *core.DatosSolicitudAutorizacionLigadaV3) { d.Recurso.Atributos["plan_firma_sha256"] = h },
		"otra_finalidad":     func(d *core.DatosSolicitudAutorizacionLigadaV3) { d.Finalidad = "otra" },
	} {
		copia := datos
		copia.Recurso.Ambitos = map[string]string{"organizacion_ref": recurso.Ambitos["organizacion_ref"], "unidad_ref": recurso.Ambitos["unidad_ref"]}
		copia.Recurso.Atributos = map[string]string{"material_sha256": h, "descriptor_firma_sha256": h}
		cambiar(&copia)
		if solicitudAutorizacionFirmaVecV2CTValida(copia, motivo) {
			t.Fatal("recurso ajeno admitido", caso)
		}
	}
	if solicitudAutorizacionFirmaVecV2CTValida(datos, motivoFirmasR5V2CTDesarrollo()) {
		t.Fatal("motivo de otra operación admitido")
	}
}
