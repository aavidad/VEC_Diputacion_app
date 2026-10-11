package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	vec "vec-diputacion-granada/internal/vec/domain"
)

func catalogoFirmantesPrueba() CatalogoRolesFirmantes {
	return CatalogoRolesFirmantes{Referencia: "catalogo:firmantes:prueba", Version: 1,
		HuellaSHA256: strings.Repeat("a", 64), Roles: []DefinicionRolFirmante{
			{Clave: "tecnico", Clase: "tecnica", Efecto: "constancia_documental"},
			{Clave: "miembro", Clase: "organo", Efecto: "decision_segun_bases"},
			{Clave: "observador", Clase: "observacion", Efecto: "sin_efecto_decisorio"},
		}}
}

func materialFirmantesPrueba() MaterialDecisionFirmantes {
	evidencias := []vec.ReferenciaDocumento{{ID: "documento:original", Version: 2}}
	b := []byte(`[{"id":"documento:original","version":2}]`)
	h := sha256.Sum256(b)
	m := MaterialDecisionFirmantes{HechoRef: "hecho:curso", HechoVersion: 3,
		DecisionRef: "decision:curso", DecisionVersion: 1,
		Evidencias: evidencias, EvidenciasSHA256: hex.EncodeToString(h[:]), MotivoRef: "motivo:revision:1", PuntosMicropuntos: "0"}
	m.ContenidoSHA256, _ = HuellaContenidoDecisionFirmantes(m)
	return m
}

func firmantesPrueba() []FirmanteDesignado {
	return []FirmanteDesignado{
		{PersonaRef: "persona:tecnico", RolClave: "tecnico", OrganoRef: "organo:rrhh", DesignacionRef: "designacion:tecnico", DesignacionVersion: 1, Orden: 1, Obligatorio: true},
		{PersonaRef: "persona:observador", RolClave: "observador", OrganoRef: "organo:acta", DesignacionRef: "designacion:observador", DesignacionVersion: 1, Orden: 2},
	}
}

func TestPlanFirmantesCambioTrasFirmaReiniciaVersion(t *testing.T) {
	c, m, firmantes := catalogoFirmantesPrueba(), materialFirmantesPrueba(), firmantesPrueba()
	primero, err := PrepararPlanFirmantesMerito(m, c, 1, firmantes)
	if err != nil {
		t.Fatal(err)
	}
	firmado, err := primero.anotarFirmaConstatada(FirmaConstatada{DesignacionRef: firmantes[0].DesignacionRef,
		PersonaRef: firmantes[0].PersonaRef, DocumentoRef: "documento:decision", Revision: 1,
		HuellaSHA256: strings.Repeat("c", 64), Sentido: "conformidad"})
	if err != nil || len(firmado.Firmas) != 1 || len(primero.Firmas) != 0 {
		t.Fatal("la firma altero instantanea previa", err)
	}
	m.DecisionVersion, m.MotivoRef = 2, "motivo:revision:2"
	m.ContenidoSHA256, _ = HuellaContenidoDecisionFirmantes(m)
	siguiente, err := firmado.NuevaVersionContenido(m, c, firmantes)
	if err != nil || siguiente.VersionRoster != 2 || siguiente.Material.DecisionVersion != 2 || len(siguiente.Firmas) != 0 ||
		siguiente.HuellaRosterSHA256 == firmado.HuellaRosterSHA256 || len(firmado.Firmas) != 1 {
		t.Fatal("cambio de motivo conservo firma eficaz", err)
	}
	if _, err := firmado.NuevaVersionContenido(firmado.Material, c, firmantes); err == nil {
		t.Fatal("se acepto otra version sin cambio material")
	}
	igual, err := PrepararPlanFirmantesMerito(materialFirmantesPrueba(), c, 1, firmantes)
	if err != nil || igual.HuellaRosterSHA256 != primero.HuellaRosterSHA256 {
		t.Fatal("mismos bytes produjeron otra instantanea", err)
	}
}

func TestPlanFirmantesDisconformidadNoApruebaYSeConservaHastaNuevaVersion(t *testing.T) {
	p, err := PrepararPlanFirmantesMerito(materialFirmantesPrueba(), catalogoFirmantesPrueba(), 1, firmantesPrueba())
	if err != nil {
		t.Fatal(err)
	}
	p, err = p.anotarFirmaConstatada(FirmaConstatada{DesignacionRef: "designacion:tecnico", PersonaRef: "persona:tecnico",
		DocumentoRef: "documento:decision", Revision: 1, HuellaSHA256: strings.Repeat("c", 64), Sentido: "disconformidad"})
	if err != nil || !p.TieneDisconformidad() {
		t.Fatal("disconformidad perdida", err)
	}
	m := p.Material
	m.DecisionVersion, m.PuntosMicropuntos = 2, "1000000"
	m.ContenidoSHA256, _ = HuellaContenidoDecisionFirmantes(m)
	nuevo, err := p.NuevaVersionContenido(m, catalogoFirmantesPrueba(), firmantesPrueba())
	if err != nil || nuevo.TieneDisconformidad() || len(p.Firmas) != 1 {
		t.Fatal("historia o reinicio de firmas incorrectos", err)
	}
}

func TestPlanFirmantesDeniegaRolDuplicadoYDesignacionAusente(t *testing.T) {
	c, m, base := catalogoFirmantesPrueba(), materialFirmantesPrueba(), firmantesPrueba()
	casos := []struct {
		nombre  string
		cambiar func([]FirmanteDesignado)
	}{
		{"rol_sin_catalogo", func(f []FirmanteDesignado) { f[0].RolClave = "sindicato_autorizado" }},
		{"sin_designacion", func(f []FirmanteDesignado) { f[0].DesignacionRef = "" }},
		{"persona_duplicada", func(f []FirmanteDesignado) { f[1].PersonaRef = f[0].PersonaRef }},
		{"designacion_duplicada", func(f []FirmanteDesignado) { f[1].DesignacionRef = f[0].DesignacionRef }},
		{"orden_duplicado", func(f []FirmanteDesignado) { f[1].Orden = f[0].Orden }},
		{"ningun_obligatorio", func(f []FirmanteDesignado) { f[0].Obligatorio = false }},
	}
	for _, tc := range casos {
		t.Run(tc.nombre, func(t *testing.T) {
			f := append([]FirmanteDesignado(nil), base...)
			tc.cambiar(f)
			if _, err := PrepararPlanFirmantesMerito(m, c, 1, f); err == nil {
				t.Fatal("lista sin designacion o competencia aceptada")
			}
		})
	}
}

func TestPlanFirmantesSustitutoExigeDesignacionNuevaYReiniciaFirmas(t *testing.T) {
	c := catalogoFirmantesPrueba()
	p, err := PrepararPlanFirmantesMerito(materialFirmantesPrueba(), c, 1, firmantesPrueba())
	if err != nil {
		t.Fatal(err)
	}
	p, err = p.anotarFirmaConstatada(FirmaConstatada{DesignacionRef: "designacion:tecnico", PersonaRef: "persona:tecnico",
		DocumentoRef: "documento:decision", Revision: 1, HuellaSHA256: strings.Repeat("c", 64), Sentido: "conformidad"})
	if err != nil {
		t.Fatal(err)
	}
	sustituto := FirmanteDesignado{PersonaRef: "persona:sustituto", RolClave: "tecnico", OrganoRef: "organo:rrhh",
		DesignacionRef: "designacion:sustituto", DesignacionVersion: 1, Orden: 1, Obligatorio: true,
		SustituyeRef: "designacion:tecnico"}
	nuevo, err := p.SustituirFirmante(c, "designacion:tecnico", sustituto)
	if err != nil || nuevo.VersionRoster != 2 || len(nuevo.Firmas) != 0 || nuevo.Firmantes[0].PersonaRef != sustituto.PersonaRef {
		t.Fatal("sustitucion no creo version sin firmas", err)
	}
	sustituto.DesignacionRef = "designacion:tecnico"
	if _, err := p.SustituirFirmante(c, "designacion:tecnico", sustituto); err == nil {
		t.Fatal("sustituto reutilizo designacion anterior")
	}
}

func TestPlanFirmantesRespetaOrdenYNoConvierteObservacionEnAprobacion(t *testing.T) {
	p, err := PrepararPlanFirmantesMerito(materialFirmantesPrueba(), catalogoFirmantesPrueba(), 1, firmantesPrueba())
	if err != nil {
		t.Fatal(err)
	}
	observacion := FirmaConstatada{DesignacionRef: "designacion:observador", PersonaRef: "persona:observador",
		DocumentoRef: "documento:acta", Revision: 2, HuellaSHA256: strings.Repeat("d", 64), Sentido: "conformidad"}
	if _, err := p.anotarFirmaConstatada(observacion); err == nil {
		t.Fatal("observacion adelanto firma tecnica obligatoria")
	}
	tecnica := FirmaConstatada{DesignacionRef: "designacion:tecnico", PersonaRef: "persona:tecnico",
		DocumentoRef: "documento:decision", Revision: 1, HuellaSHA256: strings.Repeat("c", 64), Sentido: "conformidad"}
	p, err = p.anotarFirmaConstatada(tecnica)
	if err != nil {
		t.Fatal(err)
	}
	p, err = p.anotarFirmaConstatada(observacion)
	if err != nil || p.Estado != "en_firma" || len(p.Firmas) != 2 {
		t.Fatal("firmas no respetaron orden", err)
	}
}
