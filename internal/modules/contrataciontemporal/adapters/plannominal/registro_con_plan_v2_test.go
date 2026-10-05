package plannominal

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	firma "vec-diputacion-granada/internal/modules/contrataciontemporal/application/firmaautorizacionv2"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

func exportacionRegistroPlanPrueba(t *testing.T, m ports.MaterialFirmaVerificadaV2, recurso vd.RecursoAutorizable, ref, actor string) vp.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	t.Helper()
	h, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	decision, err := json.Marshal(map[string]string{
		"decision_ref": ref, "principal_id": actor, "perfil_activo_ref": m.PerfilActivoOperadorRef,
		"version_rol_ref": m.VersionRolFirmanteRef, "accion": ports.AccionRegistrarFirmaVec,
		"recurso_ref": m.RecursoRef(), "modulo_id": ports.ModuloContratacion,
		"tipo_recurso": ports.TipoRecursoFirmaVec, "finalidad": ports.FinalidadFirmaDocumento,
		"contexto_recurso_huella_sha256": h,
	})
	if err != nil {
		t.Fatal(err)
	}
	dh := sha256.Sum256(decision)
	resumen, err := vp.NuevoResumenCapacidadAtestacionAutorizacionV3(ref, hex.EncodeToString(dh[:]), strings.Repeat("a", 64),
		"contexto:plan-prueba", strings.Repeat("b", 64), ports.AccionRegistrarFirmaVec, m.RecursoRef(), h,
		ports.AudienciaFirmaVecV2, m.ComprobadaEn, m.ComprobadaEn.Add(5*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	spki, err := x509.MarshalPKIXPublicKey(ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize)).Public())
	if err != nil {
		t.Fatal(err)
	}
	x, err := vp.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte{'x'}, vp.TamanoMinimoCapacidadCanonicaV3),
		resumen, decision, []byte("{}"), []byte("{}"), 1, 1, []byte("payload"), []byte("sobre"), []byte("evidencia"), spki)
	if err != nil {
		t.Fatal(err)
	}
	return x
}

func capacidadInteriorRegistroPlanPrueba(t *testing.T, m ports.MaterialFirmaVerificadaV2, d ports.DescriptorConstructorFirmaV2) ports.CapacidadFirmaVerificadaV2 {
	t.Helper()
	descriptor, err := firma.CanonicoDescriptorFirmaVerificadaV2(m, d)
	if err != nil {
		t.Fatal(err)
	}
	recurso, err := firma.RecursoFirmaVerificadaV2(m, descriptor)
	if err != nil {
		t.Fatal(err)
	}
	x := exportacionRegistroPlanPrueba(t, m, recurso, "decision:firma:interior", m.FirmantePrincipalRef)
	h := sha256.Sum256(descriptor)
	return ports.TransportarMaterialFirmaVerificadaV2ConDescriptor(x, descriptor, hex.EncodeToString(h[:]))
}

type emisorRegistroPlanPrueba struct {
	t       *testing.T
	actor   string
	errores int
}

func (e *emisorRegistroPlanPrueba) AutorizarMaterialPlanFirmaV2(_ context.Context, m ports.MaterialFirmaVerificadaV2, r vd.RecursoAutorizable) (vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	if e.errores != 0 {
		return vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrFirmaDocumentoDenegada
	}
	actor := m.FirmantePrincipalRef
	if e.actor != "" {
		actor = e.actor
	}
	return exportacionRegistroPlanPrueba(e.t, m, r, "decision:firma:exterior", actor), nil
}

type escritorRegistroPlanPrueba struct{ llamadas int }

func (e *escritorRegistroPlanPrueba) RegistrarFirmaConPlanV2(_ context.Context, _ ports.MaterialFirmaVerificadaV2, _ ports.CapacidadFirmaConPlanV2) (ports.ReciboFirmaDocumento, error) {
	e.llamadas++
	return ports.ReciboFirmaDocumento{ReciboRef: "recibo:original"}, nil
}

type consultaRegistroPlanPrueba struct {
	escrituras, consultas int
}

func (c *consultaRegistroPlanPrueba) RegistrarFirmaVerificadaV2(context.Context, ports.MaterialFirmaVerificadaV2, ports.CapacidadFirmaVerificadaV2) (ports.ReciboFirmaDocumento, error) {
	c.escrituras++
	return ports.ReciboFirmaDocumento{}, errors.New("registro anterior invocado")
}
func (c *consultaRegistroPlanPrueba) ConsultarFirmasAutorizadasV2(context.Context, ports.MaterialConsultaFirmasR5V2, ports.CapacidadConsultaFirmasR5V2) (ports.LecturaFirmasR5V2, error) {
	c.consultas++
	return ports.LecturaFirmasR5V2{}, nil
}

func TestRegistroConPlanUsaFuenteYUnaEscritura(t *testing.T) {
	f, m, selector, pub, _ := descriptorPrueba(t, 1)
	d, err := f.DescriptorPlanFijadoFirmaV2(t.Context(), m)
	if err != nil {
		t.Fatal(err)
	}
	interior := capacidadInteriorRegistroPlanPrueba(t, m, d.Descriptor)
	emisor := &emisorRegistroPlanPrueba{t: t}
	escritor := &escritorRegistroPlanPrueba{}
	consulta := &consultaRegistroPlanPrueba{}
	r, err := firma.NuevoRegistroConPlanV2(f, emisor, escritor, consulta)
	if err != nil {
		t.Fatal(err)
	}
	got, err := r.RegistrarFirmaVerificadaV2(t.Context(), m, interior)
	if err != nil || got.ReciboRef != "recibo:original" || escritor.llamadas != 1 ||
		consulta.escrituras != 0 || selector.llamadas != 2 || pub.llamadas != 2 {
		t.Fatalf("registro no releyó una selección de servidor: %v", err)
	}
	if _, err := r.ConsultarFirmasAutorizadasV2(t.Context(), ports.MaterialConsultaFirmasR5V2{}, ports.CapacidadConsultaFirmasR5V2{}); err != nil || consulta.consultas != 1 {
		t.Fatal("consulta anterior no delegada")
	}
}

func TestRegistroConPlanDeniegaDescriptorYActorCruzados(t *testing.T) {
	for _, caso := range []string{"descriptor", "actor"} {
		t.Run(caso, func(t *testing.T) {
			f, m, selector, _, _ := descriptorPrueba(t, 1)
			d, err := f.DescriptorPlanFijadoFirmaV2(t.Context(), m)
			if err != nil {
				t.Fatal(err)
			}
			interior := capacidadInteriorRegistroPlanPrueba(t, m, d.Descriptor)
			emisor := &emisorRegistroPlanPrueba{t: t}
			if caso == "descriptor" {
				selector.seleccion.Recurso.RecursoContextoSHA256 = strings.Repeat("e", 64)
			} else {
				emisor.actor = "per_ajeno_sintetico"
			}
			escritor := &escritorRegistroPlanPrueba{}
			r, err := firma.NuevoRegistroConPlanV2(f, emisor, escritor, &consultaRegistroPlanPrueba{})
			if err != nil {
				t.Fatal(err)
			}
			if got, err := r.RegistrarFirmaVerificadaV2(t.Context(), m, interior); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) ||
				got.ReciboRef != "" || escritor.llamadas != 0 {
				t.Fatalf("cruce %s alcanzó SQL: %v", caso, err)
			}
		})
	}
}
